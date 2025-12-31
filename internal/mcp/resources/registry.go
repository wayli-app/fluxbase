package resources

import (
	"context"
	"strings"
	"sync"

	"github.com/fluxbase-eu/fluxbase/internal/mcp"
)

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
	Read(ctx context.Context, authCtx *mcp.AuthContext) ([]mcp.Content, error)
}

// TemplateResourceProvider extends ResourceProvider with template capabilities
type TemplateResourceProvider interface {
	ResourceProvider

	// IsTemplate returns true for template resources
	IsTemplate() bool

	// MatchURI checks if a URI matches and extracts parameters
	MatchURI(uri string) (map[string]string, bool)

	// ReadWithParams reads the resource with extracted parameters
	ReadWithParams(ctx context.Context, authCtx *mcp.AuthContext, params map[string]string) ([]mcp.Content, error)
}

// Registry manages MCP resources
type Registry struct {
	providers []ResourceProvider
	mu        sync.RWMutex
}

// NewRegistry creates a new resource registry
func NewRegistry() *Registry {
	return &Registry{
		providers: make([]ResourceProvider, 0),
	}
}

// Register adds a resource provider to the registry
func (r *Registry) Register(provider ResourceProvider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers = append(r.providers, provider)
}

// GetProvider returns the provider that handles the given URI, or nil if not found
func (r *Registry) GetProvider(uri string) ResourceProvider {
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

// ReadResource reads a resource by URI
func (r *Registry) ReadResource(ctx context.Context, uri string, authCtx *mcp.AuthContext) ([]mcp.Content, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// First try exact match (non-templates)
	for _, provider := range r.providers {
		if provider.URI() == uri {
			// Check scopes
			if !authCtx.HasScopes(provider.RequiredScopes()...) {
				continue
			}
			return provider.Read(ctx, authCtx)
		}
	}

	// Then try template match
	for _, provider := range r.providers {
		if tp, ok := provider.(TemplateResourceProvider); ok && tp.IsTemplate() {
			if params, matched := tp.MatchURI(uri); matched {
				// Check scopes
				if !authCtx.HasScopes(provider.RequiredScopes()...) {
					continue
				}
				return tp.ReadWithParams(ctx, authCtx, params)
			}
		}
	}

	return nil, nil
}

// ListResources returns all static resources that the user has access to
func (r *Registry) ListResources(authCtx *mcp.AuthContext) []mcp.Resource {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var resources []mcp.Resource
	for _, provider := range r.providers {
		// Skip templates - they're listed separately
		if tp, ok := provider.(TemplateResourceProvider); ok && tp.IsTemplate() {
			continue
		}

		// Check if user has required scopes
		if !authCtx.HasScopes(provider.RequiredScopes()...) {
			continue
		}

		resources = append(resources, mcp.Resource{
			URI:         provider.URI(),
			Name:        provider.Name(),
			Description: provider.Description(),
			MimeType:    provider.MimeType(),
		})
	}

	return resources
}

// ListTemplates returns all resource templates that the user has access to
func (r *Registry) ListTemplates(authCtx *mcp.AuthContext) []mcp.ResourceTemplate {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var templates []mcp.ResourceTemplate
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

		templates = append(templates, mcp.ResourceTemplate{
			URITemplate: provider.URI(),
			Name:        provider.Name(),
			Description: provider.Description(),
			MimeType:    provider.MimeType(),
		})
	}

	return templates
}

// MatchTemplate matches a URI against a template pattern and extracts parameters
func MatchTemplate(template, uri string) (map[string]string, bool) {
	templateParts := strings.Split(template, "/")
	uriParts := strings.Split(uri, "/")

	if len(templateParts) != len(uriParts) {
		return nil, false
	}

	params := make(map[string]string)
	for i, part := range templateParts {
		// Template parameter - extract value
		if strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}") {
			paramName := part[1 : len(part)-1]
			params[paramName] = uriParts[i]
			continue
		}
		// Exact match required
		if part != uriParts[i] {
			return nil, false
		}
	}

	return params, true
}
