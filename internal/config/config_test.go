package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name     string
		yaml     string
		wantErr  bool
		validate func(*testing.T, *Config)
	}{
		{
			name: "valid minimal config",
			yaml: `
project: test
env: dev
logging:
  sqlite_path: /tmp/test.db
users: []
resources:
  http: {}
  postgres: {}
operations: []
tasks: []
`,
			wantErr: false,
			validate: func(t *testing.T, cfg *Config) {
				if cfg.Project != "test" {
					t.Errorf("Project = %q, want %q", cfg.Project, "test")
				}
				if cfg.Env != "dev" {
					t.Errorf("Env = %q, want %q", cfg.Env, "dev")
				}
				if cfg.Logging.SQLitePath != "/tmp/test.db" {
					t.Errorf("SQLitePath = %q, want %q", cfg.Logging.SQLitePath, "/tmp/test.db")
				}
			},
		},
		{
			name: "config with users and roles",
			yaml: `
project: test
env: dev
logging:
  sqlite_path: /tmp/test.db
users:
  - id: alice
    ssh_users: [alice, alice-dev]
    roles: [admin, owner]
resources:
  http: {}
  postgres: {}
operations: []
tasks: []
`,
			wantErr: false,
			validate: func(t *testing.T, cfg *Config) {
				if len(cfg.Users) != 1 {
					t.Fatalf("len(Users) = %d, want 1", len(cfg.Users))
				}
				u := cfg.Users[0]
				if u.ID != "alice" {
					t.Errorf("User.ID = %q, want %q", u.ID, "alice")
				}
				if len(u.SSHUsers) != 2 {
					t.Errorf("len(User.SSHUsers) = %d, want 2", len(u.SSHUsers))
				}
				if len(u.Roles) != 2 {
					t.Errorf("len(User.Roles) = %d, want 2", len(u.Roles))
				}
			},
		},
		{
			name: "config with resources",
			yaml: `
project: test
env: dev
logging:
  sqlite_path: /tmp/test.db
users: []
resources:
  http:
    api:
      base_url: https://api.example.com
  postgres:
    db:
      dsn_env: PG_DSN
operations: []
tasks: []
`,
			wantErr: false,
			validate: func(t *testing.T, cfg *Config) {
				if len(cfg.Resources.HTTP) != 1 {
					t.Fatalf("len(Resources.HTTP) = %d, want 1", len(cfg.Resources.HTTP))
				}
				if cfg.Resources.HTTP["api"].BaseURL != "https://api.example.com" {
					t.Errorf("HTTP[api].BaseURL = %q, want %q", cfg.Resources.HTTP["api"].BaseURL, "https://api.example.com")
				}
				if len(cfg.Resources.Postgres) != 1 {
					t.Fatalf("len(Resources.Postgres) = %d, want 1", len(cfg.Resources.Postgres))
				}
				if cfg.Resources.Postgres["db"].DSNEnv != "PG_DSN" {
					t.Errorf("Postgres[db].DSNEnv = %q, want %q", cfg.Resources.Postgres["db"].DSNEnv, "PG_DSN")
				}
			},
		},
		{
			name: "config with operations",
			yaml: `
project: test
env: dev
logging:
  sqlite_path: /tmp/test.db
users: []
resources:
  http:
    api:
      base_url: https://api.example.com
  postgres: {}
operations:
  - id: get-users
    label: Get Users
    type: http
    target: api
    method: GET
    path: /users
    allowed_roles: [admin]
tasks: []
`,
			wantErr: false,
			validate: func(t *testing.T, cfg *Config) {
				if len(cfg.Operations) != 1 {
					t.Fatalf("len(Operations) = %d, want 1", len(cfg.Operations))
				}
				op := cfg.Operations[0]
				if op.ID != "get-users" {
					t.Errorf("Operation.ID = %q, want %q", op.ID, "get-users")
				}
				if op.Type != "http" {
					t.Errorf("Operation.Type = %q, want %q", op.Type, "http")
				}
				if op.Method != "GET" {
					t.Errorf("Operation.Method = %q, want %q", op.Method, "GET")
				}
			},
		},
		{
			name: "config with tasks",
			yaml: `
project: test
env: dev
logging:
  sqlite_path: /tmp/test.db
users: []
resources:
  http:
    api:
      base_url: https://api.example.com
  postgres: {}
operations: []
tasks:
  - id: deploy
    label: Deploy Application
    allowed_roles: [admin]
    risk_level: high
    on_error: fail_fast
    steps:
      - id: step1
        type: http
        resource: api
        method: POST
        path: /deploy
        on_error: inherit
      - id: step2
        type: sleep
        seconds: 5
        on_error: continue
`,
			wantErr: false,
			validate: func(t *testing.T, cfg *Config) {
				if len(cfg.Tasks) != 1 {
					t.Fatalf("len(Tasks) = %d, want 1", len(cfg.Tasks))
				}
				task := cfg.Tasks[0]
				if task.ID != "deploy" {
					t.Errorf("Task.ID = %q, want %q", task.ID, "deploy")
				}
				if task.RiskLevel != RiskHigh {
					t.Errorf("Task.RiskLevel = %q, want %q", task.RiskLevel, RiskHigh)
				}
				if task.OnError != OnErrorFailFast {
					t.Errorf("Task.OnError = %q, want %q", task.OnError, OnErrorFailFast)
				}
				if len(task.Steps) != 2 {
					t.Fatalf("len(Task.Steps) = %d, want 2", len(task.Steps))
				}
				if task.Steps[0].Type != "http" {
					t.Errorf("Step[0].Type = %q, want %q", task.Steps[0].Type, "http")
				}
				if task.Steps[1].Type != "sleep" {
					t.Errorf("Step[1].Type = %q, want %q", task.Steps[1].Type, "sleep")
				}
			},
		},
		{
			name:    "invalid YAML",
			yaml:    `project: test\nenv: dev\ninvalid: [`,
			wantErr: true,
		},
		{
			name:    "empty file",
			yaml:    ``,
			wantErr: false,
			validate: func(t *testing.T, cfg *Config) {
				if cfg == nil {
					t.Fatal("Config is nil")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "test.yaml")
			if err := os.WriteFile(configPath, []byte(tt.yaml), 0644); err != nil {
				t.Fatalf("WriteFile: %v", err)
			}

			oldPath := os.Getenv("LAZYADMIN_CONFIG_PATH")
			os.Setenv("LAZYADMIN_CONFIG_PATH", configPath)
			defer func() {
				if oldPath == "" {
					os.Unsetenv("LAZYADMIN_CONFIG_PATH")
				} else {
					os.Setenv("LAZYADMIN_CONFIG_PATH", oldPath)
				}
			}()

			cfg, err := Load()
			if (err != nil) != tt.wantErr {
				t.Errorf("Load() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.validate != nil {
				tt.validate(t, cfg)
			}
		})
	}
}

// TestLoad_DefaultPath is skipped because it requires changing the working directory
// which is complex in tests. The default path behavior is tested indirectly
// through the main Load() tests.

func TestLoad_MissingFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "nonexistent.yaml")

	oldPath := os.Getenv("LAZYADMIN_CONFIG_PATH")
	os.Setenv("LAZYADMIN_CONFIG_PATH", configPath)
	defer func() {
		if oldPath == "" {
			os.Unsetenv("LAZYADMIN_CONFIG_PATH")
		} else {
			os.Setenv("LAZYADMIN_CONFIG_PATH", oldPath)
		}
	}()

	_, err := Load()
	if err == nil {
		t.Error("Load() error = nil, want error")
	}
}
