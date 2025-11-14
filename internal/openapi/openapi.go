package openapi

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/you/lazyadmin/internal/config"
)

type Generator struct {
	httpClient *http.Client
}

func NewGenerator() *Generator {
	return &Generator{
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}
}

// GenerateOperations loads OpenAPI specifications and converts eligible endpoints
// into config.Operation entries. Returns a new slice without modifying cfg.
func (g *Generator) GenerateOperations(ctx context.Context, cfg *config.Config) ([]config.Operation, error) {
	var ops []config.Operation

	for name, backend := range cfg.OpenAPI.Backends {
		backendOps, err := g.generateForBackend(ctx, cfg, name, backend)
		if err != nil {
			return nil, fmt.Errorf("openapi backend %s: %w", name, err)
		}

		ops = append(ops, backendOps...)
	}

	return ops, nil
}

func (g *Generator) generateForBackend(ctx context.Context, cfg *config.Config, name string, backend config.OpenAPIBackend) ([]config.Operation, error) {
	loader := &openapi3.Loader{
		Context: ctx,
	}

	doc, err := loader.LoadFromURI(mustParseURL(backend.DocURL))
	if err != nil {
		return nil, fmt.Errorf("load openapi from %s: %w", backend.DocURL, err)
	}

	if err := doc.Validate(ctx); err != nil {
		return nil, fmt.Errorf("validate openapi: %w", err)
	}

	var ops []config.Operation

	if doc.Paths != nil {
		for path, pathItem := range doc.Paths.Map() {
			if pathItem == nil {
				continue
			}
			for method, op := range pathItem.Operations() {
				if !operationEligible(op, backend) {
					continue
				}

				if hasRequiredRequestBody(op) {
					continue
				}

				opID := op.OperationID
				if opID == "" {
					opID = fmt.Sprintf("%s_%s_%s", strings.ToLower(method), name, sanitizePath(path))
				}

				if backend.OpIDPrefix != "" {
					opID = backend.OpIDPrefix + opID
				}

				ops = append(ops, config.Operation{
					ID:           opID,
					Label:        buildLabel(op, method, path),
					Type:         "http",
					Target:       name,
					Method:       strings.ToUpper(method),
					Path:         path,
					AllowedRoles: []string{"owner", "admin"},
				})
			}
		}
	}

	return ops, nil
}

func operationEligible(op *openapi3.Operation, backend config.OpenAPIBackend) bool {
	if len(backend.TagFilter) == 0 {
		if backend.IncludeUntagged {
			return true
		}
		return len(op.Tags) > 0
	}

	for _, tag := range op.Tags {
		for _, allowed := range backend.TagFilter {
			if tag == allowed {
				return true
			}
		}
	}

	return false
}

func hasRequiredRequestBody(op *openapi3.Operation) bool {
	if op.RequestBody == nil || op.RequestBody.Value == nil {
		return false
	}

	return op.RequestBody.Value.Required
}

func buildLabel(op *openapi3.Operation, method, path string) string {
	if op.Summary != "" {
		return op.Summary
	}

	return fmt.Sprintf("%s %s", strings.ToUpper(method), path)
}

func sanitizePath(path string) string {
	path = strings.Trim(path, "/")
	path = strings.ReplaceAll(path, "/", "_")
	path = strings.ReplaceAll(path, "{", "")
	path = strings.ReplaceAll(path, "}", "")
	if path == "" {
		path = "root"
	}
	return path
}

func mustParseURL(s string) *url.URL {
	u, err := url.Parse(s)
	if err != nil {
		panic(fmt.Sprintf("invalid URL %q: %v", s, err))
	}
	return u
}
