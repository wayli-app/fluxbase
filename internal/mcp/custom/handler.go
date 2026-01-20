package custom

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/fluxbase-eu/fluxbase/internal/mcp"
	"github.com/rs/zerolog/log"
)

// DynamicToolHandler wraps a CustomTool to implement the mcp.ToolHandler interface.
type DynamicToolHandler struct {
	tool     *CustomTool
	executor *Executor
}

// NewDynamicToolHandler creates a new DynamicToolHandler for the given custom tool.
func NewDynamicToolHandler(tool *CustomTool, executor *Executor) *DynamicToolHandler {
	return &DynamicToolHandler{
		tool:     tool,
		executor: executor,
	}
}

// Name returns the tool name with namespace prefix.
// Format: "custom:{namespace}:{name}" for non-default namespaces
//
//	"custom:{name}" for default namespace (backwards compatible)
func (h *DynamicToolHandler) Name() string {
	if h.tool.Namespace == "" || h.tool.Namespace == "default" {
		return "custom:" + h.tool.Name
	}
	return "custom:" + h.tool.Namespace + ":" + h.tool.Name
}

// Description returns the tool description.
func (h *DynamicToolHandler) Description() string {
	if h.tool.Description != "" {
		return h.tool.Description
	}
	return fmt.Sprintf("Custom tool: %s", h.tool.Name)
}

// InputSchema returns the JSON Schema for the tool's input parameters.
func (h *DynamicToolHandler) InputSchema() map[string]any {
	if h.tool.InputSchema != nil {
		return h.tool.InputSchema
	}
	return map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	}
}

// RequiredScopes returns the scopes required to execute this tool.
func (h *DynamicToolHandler) RequiredScopes() []string {
	// Always require execute:custom scope plus any tool-specific scopes
	scopes := []string{"execute:custom"}
	scopes = append(scopes, h.tool.RequiredScopes...)
	return scopes
}

// Execute runs the custom tool with the given arguments.
func (h *DynamicToolHandler) Execute(
	ctx context.Context,
	args map[string]any,
	authCtx *mcp.AuthContext,
) (*mcp.ToolResult, error) {
	log.Debug().
		Str("tool", h.tool.Name).
		Interface("args", args).
		Msg("Executing custom MCP tool")

	return h.executor.ExecuteTool(ctx, h.tool, args, authCtx)
}

// DynamicResourceProvider wraps a CustomResource to implement the mcp.ResourceProvider interface.
type DynamicResourceProvider struct {
	resource *CustomResource
	executor *Executor
}

// NewDynamicResourceProvider creates a new DynamicResourceProvider for the given custom resource.
func NewDynamicResourceProvider(resource *CustomResource, executor *Executor) *DynamicResourceProvider {
	return &DynamicResourceProvider{
		resource: resource,
		executor: executor,
	}
}

// URI returns the resource URI.
func (p *DynamicResourceProvider) URI() string {
	return p.resource.URI
}

// Name returns the resource name.
func (p *DynamicResourceProvider) Name() string {
	return p.resource.Name
}

// Description returns the resource description.
func (p *DynamicResourceProvider) Description() string {
	if p.resource.Description != "" {
		return p.resource.Description
	}
	return fmt.Sprintf("Custom resource: %s", p.resource.Name)
}

// MimeType returns the resource MIME type.
func (p *DynamicResourceProvider) MimeType() string {
	return p.resource.MimeType
}

// RequiredScopes returns the scopes required to read this resource.
func (p *DynamicResourceProvider) RequiredScopes() []string {
	// Always require read:custom scope plus any resource-specific scopes
	scopes := []string{"read:custom"}
	scopes = append(scopes, p.resource.RequiredScopes...)
	return scopes
}

// Read reads the resource contents (for non-template resources).
func (p *DynamicResourceProvider) Read(ctx context.Context, authCtx *mcp.AuthContext) ([]mcp.Content, error) {
	if p.resource.IsTemplate {
		return nil, fmt.Errorf("template resource requires parameters, use ReadWithParams")
	}

	log.Debug().
		Str("uri", p.resource.URI).
		Msg("Reading custom MCP resource")

	return p.executor.ExecuteResource(ctx, p.resource, nil, authCtx)
}

// IsTemplate returns true if this resource supports URI parameters.
func (p *DynamicResourceProvider) IsTemplate() bool {
	return p.resource.IsTemplate
}

// MatchURI attempts to match a URI against this resource's template pattern.
// Returns the extracted parameters if successful.
func (p *DynamicResourceProvider) MatchURI(uri string) (map[string]string, bool) {
	if !p.resource.IsTemplate {
		// Non-template resource: exact match only
		if uri == p.resource.URI {
			return map[string]string{}, true
		}
		return nil, false
	}

	// Template resource: extract parameters
	pattern := p.resource.URI
	params := make(map[string]string)

	// Convert template pattern to regex
	// e.g., "fluxbase://custom/users/{id}" -> "^fluxbase://custom/users/([^/]+)$"
	paramNames := []string{}
	regexPattern := "^" + regexp.QuoteMeta(pattern) + "$"

	// Find all {param} placeholders and replace with capture groups
	paramRegex := regexp.MustCompile(`\\\{([^}]+)\\\}`)
	matches := paramRegex.FindAllStringSubmatch(regexPattern, -1)
	for _, match := range matches {
		paramNames = append(paramNames, match[1])
	}
	regexPattern = paramRegex.ReplaceAllString(regexPattern, `([^/]+)`)

	re, err := regexp.Compile(regexPattern)
	if err != nil {
		log.Warn().Err(err).Str("pattern", pattern).Msg("Failed to compile resource URI pattern")
		return nil, false
	}

	uriMatches := re.FindStringSubmatch(uri)
	if uriMatches == nil {
		return nil, false
	}

	// Extract parameter values
	for i, name := range paramNames {
		if i+1 < len(uriMatches) {
			params[name] = uriMatches[i+1]
		}
	}

	return params, true
}

// ReadWithParams reads the resource with the given URI parameters.
func (p *DynamicResourceProvider) ReadWithParams(
	ctx context.Context,
	authCtx *mcp.AuthContext,
	params map[string]string,
) ([]mcp.Content, error) {
	log.Debug().
		Str("uri", p.resource.URI).
		Interface("params", params).
		Msg("Reading custom MCP resource with params")

	return p.executor.ExecuteResource(ctx, p.resource, params, authCtx)
}

// Manager handles registration and lifecycle of custom MCP tools and resources.
type Manager struct {
	storage          *Storage
	executor         *Executor
	toolRegistry     *mcp.ToolRegistry
	resourceRegistry *mcp.ResourceRegistry

	mu              sync.RWMutex
	registeredTools map[string]*DynamicToolHandler      // key: tool name
	registeredRes   map[string]*DynamicResourceProvider // key: resource URI
}

// NewManager creates a new Manager instance.
func NewManager(
	storage *Storage,
	executor *Executor,
	toolRegistry *mcp.ToolRegistry,
	resourceRegistry *mcp.ResourceRegistry,
) *Manager {
	return &Manager{
		storage:          storage,
		executor:         executor,
		toolRegistry:     toolRegistry,
		resourceRegistry: resourceRegistry,
		registeredTools:  make(map[string]*DynamicToolHandler),
		registeredRes:    make(map[string]*DynamicResourceProvider),
	}
}

// LoadAndRegisterAll loads all enabled custom tools and resources from the database
// and registers them with the MCP registries.
func (m *Manager) LoadAndRegisterAll(ctx context.Context) error {
	// Load and register tools
	tools, err := m.storage.ListTools(ctx, ListToolsFilter{EnabledOnly: true})
	if err != nil {
		return fmt.Errorf("failed to load custom tools: %w", err)
	}

	for _, tool := range tools {
		if err := m.RegisterTool(tool); err != nil {
			log.Warn().Err(err).Str("tool", tool.Name).Msg("Failed to register custom tool")
		}
	}

	// Load and register resources
	resources, err := m.storage.ListResources(ctx, ListResourcesFilter{EnabledOnly: true})
	if err != nil {
		return fmt.Errorf("failed to load custom resources: %w", err)
	}

	for _, resource := range resources {
		if err := m.RegisterResource(resource); err != nil {
			log.Warn().Err(err).Str("resource", resource.URI).Msg("Failed to register custom resource")
		}
	}

	log.Info().
		Int("tools", len(tools)).
		Int("resources", len(resources)).
		Msg("Loaded custom MCP tools and resources")

	return nil
}

// RegisterTool registers a custom tool with the MCP tool registry.
func (m *Manager) RegisterTool(tool *CustomTool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	handler := NewDynamicToolHandler(tool, m.executor)

	// Unregister existing if present
	if existing, ok := m.registeredTools[tool.Name]; ok {
		m.toolRegistry.Unregister(existing.Name())
	}

	// Register with MCP
	m.toolRegistry.Register(handler)
	m.registeredTools[tool.Name] = handler

	log.Debug().Str("tool", tool.Name).Msg("Registered custom MCP tool")
	return nil
}

// UnregisterTool unregisters a custom tool from the MCP tool registry.
func (m *Manager) UnregisterTool(toolName string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if handler, ok := m.registeredTools[toolName]; ok {
		m.toolRegistry.Unregister(handler.Name())
		delete(m.registeredTools, toolName)
		log.Debug().Str("tool", toolName).Msg("Unregistered custom MCP tool")
	}
}

// RegisterResource registers a custom resource with the MCP resource registry.
func (m *Manager) RegisterResource(resource *CustomResource) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	provider := NewDynamicResourceProvider(resource, m.executor)

	// Unregister existing if present
	if existing, ok := m.registeredRes[resource.URI]; ok {
		m.resourceRegistry.Unregister(existing.URI())
	}

	// Register with MCP
	m.resourceRegistry.Register(provider)
	m.registeredRes[resource.URI] = provider

	log.Debug().Str("resource", resource.URI).Msg("Registered custom MCP resource")
	return nil
}

// UnregisterResource unregisters a custom resource from the MCP resource registry.
func (m *Manager) UnregisterResource(resourceURI string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if provider, ok := m.registeredRes[resourceURI]; ok {
		m.resourceRegistry.Unregister(provider.URI())
		delete(m.registeredRes, resourceURI)
		log.Debug().Str("resource", resourceURI).Msg("Unregistered custom MCP resource")
	}
}

// RefreshTool reloads a tool from the database and re-registers it.
func (m *Manager) RefreshTool(ctx context.Context, toolName, namespace string) error {
	tool, err := m.storage.GetToolByName(ctx, toolName, namespace)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			m.UnregisterTool(toolName)
			return nil
		}
		return err
	}

	if !tool.Enabled {
		m.UnregisterTool(toolName)
		return nil
	}

	return m.RegisterTool(tool)
}

// RefreshResource reloads a resource from the database and re-registers it.
func (m *Manager) RefreshResource(ctx context.Context, resourceURI, namespace string) error {
	resource, err := m.storage.GetResourceByURI(ctx, resourceURI, namespace)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			m.UnregisterResource(resourceURI)
			return nil
		}
		return err
	}

	if !resource.Enabled {
		m.UnregisterResource(resourceURI)
		return nil
	}

	return m.RegisterResource(resource)
}

// GetRegisteredToolNames returns the names of all registered custom tools.
func (m *Manager) GetRegisteredToolNames() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.registeredTools))
	for name := range m.registeredTools {
		names = append(names, name)
	}
	return names
}

// GetRegisteredResourceURIs returns the URIs of all registered custom resources.
func (m *Manager) GetRegisteredResourceURIs() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	uris := make([]string, 0, len(m.registeredRes))
	for uri := range m.registeredRes {
		uris = append(uris, uri)
	}
	return uris
}

// ExecuteToolForTest executes a custom tool for testing purposes.
func (m *Manager) ExecuteToolForTest(ctx context.Context, tool *CustomTool, args map[string]any) (*mcp.ToolResult, error) {
	// Create a test auth context with admin permissions
	authCtx := &mcp.AuthContext{
		UserID:    "test-user",
		UserEmail: "test@example.com",
		UserRole:  "admin",
		Scopes:    []string{"*"},
	}

	return m.executor.ExecuteTool(ctx, tool, args, authCtx)
}

// ExecuteResourceForTest executes a custom resource for testing purposes.
func (m *Manager) ExecuteResourceForTest(ctx context.Context, resource *CustomResource, params map[string]string) ([]mcp.Content, error) {
	// Create a test auth context with admin permissions
	authCtx := &mcp.AuthContext{
		UserID:    "test-user",
		UserEmail: "test@example.com",
		UserRole:  "admin",
		Scopes:    []string{"*"},
	}

	return m.executor.ExecuteResource(ctx, resource, params, authCtx)
}
