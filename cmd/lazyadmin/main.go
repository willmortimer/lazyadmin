package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/you/lazyadmin/internal/auth"
	"github.com/you/lazyadmin/internal/clients"
	"github.com/you/lazyadmin/internal/config"
	"github.com/you/lazyadmin/internal/logging"
	"github.com/you/lazyadmin/internal/openapi"
	"github.com/you/lazyadmin/internal/tasks"
	"github.com/you/lazyadmin/internal/ui"
	"github.com/you/lazyadmin/internal/users"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if len(cfg.OpenAPI.Backends) > 0 {
		gen := openapi.NewGenerator()
		autoOps, err := gen.GenerateOperations(ctx, cfg)
		if err != nil {
			log.Printf("openapi: %v", err)
		} else {
			log.Printf("openapi: generated %d operations", len(autoOps))
			cfg.Operations = append(cfg.Operations, autoOps...)
		}
	}

	logger, err := logging.NewAuditLogger(cfg.Logging.SQLitePath)
	if err != nil {
		log.Fatalf("audit logger: %v", err)
	}
	defer logger.Close()

	// Initialize user store (uses same SQLite database)
	userStore, err := users.NewStore(cfg.Logging.SQLitePath)
	if err != nil {
		log.Fatalf("user store: %v", err)
	}
	defer userStore.Close()

	principal, err := auth.ResolvePrincipal(cfg, userStore)
	if err != nil {
		log.Fatalf("auth: %v", err)
	}

	if err := auth.RequireYubiKeyIfConfigured(cfg, principal); err != nil {
		log.Fatalf("yubikey: %v", err)
	}

	httpClients := make(map[string]*clients.HTTPClient)
	for name, res := range cfg.Resources.HTTP {
		httpClients[name] = clients.NewHTTPClient(res.BaseURL)
	}

	pgClients := make(map[string]*clients.PostgresClient)
	for name, res := range cfg.Resources.Postgres {
		dsn := os.Getenv(res.DSNEnv)
		if dsn == "" {
			fmt.Fprintf(os.Stderr, "warning: env %s not set, skipping pg resource %s\n", res.DSNEnv, name)
			continue
		}
		client, err := clients.NewPostgresClient(dsn)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: cannot init pg resource %s: %v\n", name, err)
			continue
		}
		pgClients[name] = client
	}

	runner := tasks.NewRunner(cfg, logger, httpClients, pgClients)

	m := ui.NewModel(cfg, principal, logger, userStore, httpClients, pgClients, runner)

	if err := tea.NewProgram(m).Start(); err != nil {
		log.Fatalf("tui error: %v", err)
	}
}

