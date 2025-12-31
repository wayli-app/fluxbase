package tools

import (
	"context"
	"sync"

	"github.com/fluxbase-eu/fluxbase/internal/mcp"
)

// ToolHandler defines the interface for an MCP tool
type ToolHandler interface {
	// Name returns the tool name
	Name() string

	// Description returns a human-readable description of the tool
	Description() string

	// InputSchema returns the JSON Schema for the tool's input parameters
	InputSchema() map[string]any

	// RequiredScopes returns the scopes required to execute this tool
	RequiredScopes() []string

	// Execute executes the tool with the given arguments and returns a result
	Execute(ctx context.Context, args map[string]any, authCtx *mcp.AuthContext) (*mcp.ToolResult, error)
}

// Registry manages MCP tools
type Registry struct {
	tools map[string]ToolHandler
	mu    sync.RWMutex
}

// NewRegistry creates a new tool registry
func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]ToolHandler),
	}
}

// Register adds a tool to the registry
func (r *Registry) Register(tool ToolHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[tool.Name()] = tool
}

// Get returns a tool by name, or nil if not found
func (r *Registry) Get(name string) ToolHandler {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.tools[name]
}

// List returns all tools that the user has access to
func (r *Registry) List(authCtx *mcp.AuthContext) []mcp.Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var tools []mcp.Tool
	for _, handler := range r.tools {
		// Check if user has required scopes
		if !authCtx.HasScopes(handler.RequiredScopes()...) {
			continue
		}

		tools = append(tools, mcp.Tool{
			Name:        handler.Name(),
			Description: handler.Description(),
			InputSchema: handler.InputSchema(),
		})
	}

	return tools
}

// Names returns the names of all registered tools
func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	return names
}
