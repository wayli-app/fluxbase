package resources

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/fluxbase-eu/fluxbase/internal/functions"
	"github.com/fluxbase-eu/fluxbase/internal/mcp"
)

// FunctionsResource provides edge functions information
type FunctionsResource struct {
	storage *functions.Storage
}

// NewFunctionsResource creates a new functions resource
func NewFunctionsResource(storage *functions.Storage) *FunctionsResource {
	return &FunctionsResource{
		storage: storage,
	}
}

func (r *FunctionsResource) URI() string {
	return "fluxbase://functions"
}

func (r *FunctionsResource) Name() string {
	return "Edge Functions"
}

func (r *FunctionsResource) Description() string {
	return "List of available edge functions with their metadata"
}

func (r *FunctionsResource) MimeType() string {
	return "application/json"
}

func (r *FunctionsResource) RequiredScopes() []string {
	return []string{mcp.ScopeExecuteFunctions}
}

func (r *FunctionsResource) Read(ctx context.Context, authCtx *mcp.AuthContext) ([]mcp.Content, error) {
	if r.storage == nil {
		return nil, fmt.Errorf("functions storage not available")
	}

	// List all functions
	funcs, err := r.storage.ListFunctions(ctx, nil, nil, 1000, 0, "", false)
	if err != nil {
		return nil, fmt.Errorf("failed to list functions: %w", err)
	}

	// Build response
	functionList := make([]map[string]any, 0, len(funcs))
	for _, fn := range funcs {
		fnInfo := map[string]any{
			"name":        fn.Name,
			"namespace":   fn.Namespace,
			"enabled":     fn.Enabled,
			"description": fn.Description,
			"created_at":  fn.CreatedAt.Format("2006-01-02T15:04:05Z"),
		}

		if fn.AllowUnauthenticated {
			fnInfo["allow_unauthenticated"] = true
		}

		if fn.RateLimitRPS > 0 {
			fnInfo["rate_limit_rps"] = fn.RateLimitRPS
		}

		if fn.MaxConcurrency > 0 {
			fnInfo["max_concurrency"] = fn.MaxConcurrency
		}

		functionList = append(functionList, fnInfo)
	}

	result := map[string]any{
		"functions": functionList,
		"count":     len(functionList),
	}

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to serialize functions: %w", err)
	}

	return []mcp.Content{mcp.TextContent(string(data))}, nil
}
