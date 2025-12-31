package mcp

import (
	"context"
	"sync"
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
	Execute(ctx context.Context, args map[string]any, authCtx *AuthContext) (*ToolResult, error)
}

// ToolRegistry manages MCP tools
type ToolRegistry struct {
	tools map[string]ToolHandler
	mu    sync.RWMutex
}

// NewToolRegistry creates a new tool registry
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		tools: make(map[string]ToolHandler),
	}
}

// Register adds a tool to the registry
func (r *ToolRegistry) Register(tool ToolHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[tool.Name()] = tool
}

// GetTool returns a tool by name, or nil if not found
func (r *ToolRegistry) GetTool(name string) ToolHandler {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.tools[name]
}

// ListTools returns all tools that the user has access to
func (r *ToolRegistry) ListTools(authCtx *AuthContext) []Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var tools []Tool
	for _, handler := range r.tools {
		// Check if user has required scopes
		if !authCtx.HasScopes(handler.RequiredScopes()...) {
			continue
		}

		tools = append(tools, Tool{
			Name:        handler.Name(),
			Description: handler.Description(),
			InputSchema: handler.InputSchema(),
		})
	}

	return tools
}

// ResourceProvider defines the interface for an MCP resource provider
type ResourceProvider interface {
	// URI returns the resource URI (can be a pattern for templates)
	URI() string

	// Name returns a human-readable name for the resource
	Name() string

	// Description returns a human-readable description of the resource
	Description() string

	// MimeType returns the MIME type of the resource content
	MimeType() string

	// RequiredScopes returns the scopes required to read this resource
	RequiredScopes() []string

	// Read reads the resource contents
	Read(ctx context.Context, authCtx *AuthContext) ([]Content, error)
}

// TemplateResourceProvider extends ResourceProvider with template capabilities
type TemplateResourceProvider interface {
	ResourceProvider

	// IsTemplate returns true for template resources
	IsTemplate() bool

	// MatchURI checks if a URI matches and extracts parameters
	MatchURI(uri string) (map[string]string, bool)

	// ReadWithParams reads the resource with extracted parameters
	ReadWithParams(ctx context.Context, authCtx *AuthContext, params map[string]string) ([]Content, error)
}

// ResourceRegistry manages MCP resources
type ResourceRegistry struct {
	providers []ResourceProvider
	mu        sync.RWMutex
}

// NewResourceRegistry creates a new resource registry
func NewResourceRegistry() *ResourceRegistry {
	return &ResourceRegistry{
		providers: make([]ResourceProvider, 0),
	}
}

// Register adds a resource provider to the registry
func (r *ResourceRegistry) Register(provider ResourceProvider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers = append(r.providers, provider)
}

// GetProvider returns the provider that handles the given URI, or nil if not found
func (r *ResourceRegistry) GetProvider(uri string) ResourceProvider {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// First try exact match (non-templates)
	for _, provider := range r.providers {
		if provider.URI() == uri {
			return provider
		}
	}

	// Then try template match
	for _, provider := range r.providers {
		if tp, ok := provider.(TemplateResourceProvider); ok && tp.IsTemplate() {
			if _, matched := tp.MatchURI(uri); matched {
				return provider
			}
		}
	}

	return nil
}

// ListResources returns all static resources that the user has access to
func (r *ResourceRegistry) ListResources(authCtx *AuthContext) []Resource {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var resources []Resource
	for _, provider := range r.providers {
		// Skip templates - they're listed separately
		if tp, ok := provider.(TemplateResourceProvider); ok && tp.IsTemplate() {
			continue
		}

		// Check if user has required scopes
		if !authCtx.HasScopes(provider.RequiredScopes()...) {
			continue
		}

		resources = append(resources, Resource{
			URI:         provider.URI(),
			Name:        provider.Name(),
			Description: provider.Description(),
			MimeType:    provider.MimeType(),
		})
	}

	return resources
}

// ListTemplates returns all resource templates that the user has access to
func (r *ResourceRegistry) ListTemplates(authCtx *AuthContext) []ResourceTemplate {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var templates []ResourceTemplate
	for _, provider := range r.providers {
		// Only include templates
		tp, ok := provider.(TemplateResourceProvider)
		if !ok || !tp.IsTemplate() {
			continue
		}

		// Check if user has required scopes
		if !authCtx.HasScopes(provider.RequiredScopes()...) {
			continue
		}

		templates = append(templates, ResourceTemplate{
			URITemplate: provider.URI(),
			Name:        provider.Name(),
			Description: provider.Description(),
			MimeType:    provider.MimeType(),
		})
	}

	return templates
}
