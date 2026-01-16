package mcp

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Mock Tool Handler
// =============================================================================

type mockToolHandler struct {
	name           string
	description    string
	inputSchema    map[string]any
	requiredScopes []string
	executeFunc    func(ctx context.Context, args map[string]any, authCtx *AuthContext) (*ToolResult, error)
}

func (m *mockToolHandler) Name() string {
	return m.name
}

func (m *mockToolHandler) Description() string {
	return m.description
}

func (m *mockToolHandler) InputSchema() map[string]any {
	return m.inputSchema
}

func (m *mockToolHandler) RequiredScopes() []string {
	return m.requiredScopes
}

func (m *mockToolHandler) Execute(ctx context.Context, args map[string]any, authCtx *AuthContext) (*ToolResult, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, args, authCtx)
	}
	return &ToolResult{Content: []Content{{Type: "text", Text: "executed"}}}, nil
}

// =============================================================================
// Mock Resource Provider
// =============================================================================

type mockResourceProvider struct {
	uri            string
	name           string
	description    string
	mimeType       string
	requiredScopes []string
	readFunc       func(ctx context.Context, authCtx *AuthContext) ([]Content, error)
}

func (m *mockResourceProvider) URI() string {
	return m.uri
}

func (m *mockResourceProvider) Name() string {
	return m.name
}

func (m *mockResourceProvider) Description() string {
	return m.description
}

func (m *mockResourceProvider) MimeType() string {
	return m.mimeType
}

func (m *mockResourceProvider) RequiredScopes() []string {
	return m.requiredScopes
}

func (m *mockResourceProvider) Read(ctx context.Context, authCtx *AuthContext) ([]Content, error) {
	if m.readFunc != nil {
		return m.readFunc(ctx, authCtx)
	}
	return []Content{{Type: "text", Text: "resource content"}}, nil
}

// =============================================================================
// Mock Template Resource Provider
// =============================================================================

type mockTemplateResourceProvider struct {
	*mockResourceProvider
	isTemplate     bool
	matchFunc      func(uri string) (map[string]string, bool)
	readParamsFunc func(ctx context.Context, authCtx *AuthContext, params map[string]string) ([]Content, error)
}

func (m *mockTemplateResourceProvider) IsTemplate() bool {
	return m.isTemplate
}

func (m *mockTemplateResourceProvider) MatchURI(uri string) (map[string]string, bool) {
	if m.matchFunc != nil {
		return m.matchFunc(uri)
	}
	return nil, false
}

func (m *mockTemplateResourceProvider) ReadWithParams(ctx context.Context, authCtx *AuthContext, params map[string]string) ([]Content, error) {
	if m.readParamsFunc != nil {
		return m.readParamsFunc(ctx, authCtx, params)
	}
	return []Content{{Type: "text", Text: "template content"}}, nil
}

// =============================================================================
// ToolRegistry Tests
// =============================================================================

func TestNewToolRegistry(t *testing.T) {
	registry := NewToolRegistry()
	assert.NotNil(t, registry)
	assert.NotNil(t, registry.tools)
}

func TestToolRegistry_Register(t *testing.T) {
	registry := NewToolRegistry()

	tool := &mockToolHandler{
		name:        "test-tool",
		description: "A test tool",
	}

	registry.Register(tool)

	// Verify tool was registered
	retrieved := registry.GetTool("test-tool")
	assert.NotNil(t, retrieved)
	assert.Equal(t, "test-tool", retrieved.Name())
}

func TestToolRegistry_GetTool(t *testing.T) {
	t.Run("returns registered tool", func(t *testing.T) {
		registry := NewToolRegistry()
		tool := &mockToolHandler{name: "my-tool"}
		registry.Register(tool)

		result := registry.GetTool("my-tool")
		assert.NotNil(t, result)
		assert.Equal(t, "my-tool", result.Name())
	})

	t.Run("returns nil for unregistered tool", func(t *testing.T) {
		registry := NewToolRegistry()
		result := registry.GetTool("nonexistent")
		assert.Nil(t, result)
	})
}

func TestToolRegistry_ListTools(t *testing.T) {
	t.Run("lists all tools for service role", func(t *testing.T) {
		registry := NewToolRegistry()

		registry.Register(&mockToolHandler{
			name:           "tool-1",
			description:    "Tool 1",
			requiredScopes: []string{"read:tables"},
		})
		registry.Register(&mockToolHandler{
			name:           "tool-2",
			description:    "Tool 2",
			requiredScopes: []string{"admin:ddl"},
		})

		authCtx := &AuthContext{IsServiceRole: true}
		tools := registry.ListTools(authCtx)

		assert.Len(t, tools, 2)
	})

	t.Run("filters tools by scope", func(t *testing.T) {
		registry := NewToolRegistry()

		registry.Register(&mockToolHandler{
			name:           "read-tool",
			description:    "Read Tool",
			requiredScopes: []string{"read:tables"},
		})
		registry.Register(&mockToolHandler{
			name:           "admin-tool",
			description:    "Admin Tool",
			requiredScopes: []string{"admin:ddl"},
		})

		authCtx := &AuthContext{
			Scopes: []string{"read:tables"},
		}
		tools := registry.ListTools(authCtx)

		assert.Len(t, tools, 1)
		assert.Equal(t, "read-tool", tools[0].Name)
	})

	t.Run("returns empty for no scopes", func(t *testing.T) {
		registry := NewToolRegistry()

		registry.Register(&mockToolHandler{
			name:           "tool",
			requiredScopes: []string{"read:tables"},
		})

		authCtx := &AuthContext{Scopes: []string{}}
		tools := registry.ListTools(authCtx)

		assert.Empty(t, tools)
	})

	t.Run("tool with no required scopes is always listed", func(t *testing.T) {
		registry := NewToolRegistry()

		registry.Register(&mockToolHandler{
			name:           "public-tool",
			requiredScopes: []string{},
		})

		authCtx := &AuthContext{Scopes: []string{}}
		tools := registry.ListTools(authCtx)

		assert.Len(t, tools, 1)
	})
}

func TestToolRegistry_Register_Overwrite(t *testing.T) {
	registry := NewToolRegistry()

	tool1 := &mockToolHandler{
		name:        "tool",
		description: "Original",
	}
	tool2 := &mockToolHandler{
		name:        "tool",
		description: "Replacement",
	}

	registry.Register(tool1)
	registry.Register(tool2)

	result := registry.GetTool("tool")
	assert.Equal(t, "Replacement", result.Description())
}

// =============================================================================
// ResourceRegistry Tests
// =============================================================================

func TestNewResourceRegistry(t *testing.T) {
	registry := NewResourceRegistry()
	assert.NotNil(t, registry)
	assert.NotNil(t, registry.providers)
}

func TestResourceRegistry_Register(t *testing.T) {
	registry := NewResourceRegistry()

	provider := &mockResourceProvider{
		uri:  "fluxbase://schema/tables",
		name: "Tables",
	}

	registry.Register(provider)

	// Verify provider was registered
	result := registry.GetProvider("fluxbase://schema/tables")
	assert.NotNil(t, result)
}

func TestResourceRegistry_GetProvider(t *testing.T) {
	t.Run("exact match", func(t *testing.T) {
		registry := NewResourceRegistry()
		provider := &mockResourceProvider{uri: "fluxbase://test"}
		registry.Register(provider)

		result := registry.GetProvider("fluxbase://test")
		assert.NotNil(t, result)
	})

	t.Run("template match", func(t *testing.T) {
		registry := NewResourceRegistry()

		templateProvider := &mockTemplateResourceProvider{
			mockResourceProvider: &mockResourceProvider{
				uri: "fluxbase://schema/table/{name}",
			},
			isTemplate: true,
			matchFunc: func(uri string) (map[string]string, bool) {
				if uri == "fluxbase://schema/table/users" {
					return map[string]string{"name": "users"}, true
				}
				return nil, false
			},
		}
		registry.Register(templateProvider)

		result := registry.GetProvider("fluxbase://schema/table/users")
		assert.NotNil(t, result)
	})

	t.Run("not found", func(t *testing.T) {
		registry := NewResourceRegistry()
		result := registry.GetProvider("nonexistent")
		assert.Nil(t, result)
	})
}

func TestResourceRegistry_ListResources(t *testing.T) {
	t.Run("lists non-template resources", func(t *testing.T) {
		registry := NewResourceRegistry()

		registry.Register(&mockResourceProvider{
			uri:            "fluxbase://resource-1",
			name:           "Resource 1",
			requiredScopes: []string{},
		})
		registry.Register(&mockResourceProvider{
			uri:            "fluxbase://resource-2",
			name:           "Resource 2",
			requiredScopes: []string{},
		})

		authCtx := &AuthContext{IsServiceRole: true}
		resources := registry.ListResources(authCtx)

		assert.Len(t, resources, 2)
	})

	t.Run("excludes templates", func(t *testing.T) {
		registry := NewResourceRegistry()

		registry.Register(&mockResourceProvider{
			uri:            "fluxbase://static",
			name:           "Static Resource",
			requiredScopes: []string{},
		})
		registry.Register(&mockTemplateResourceProvider{
			mockResourceProvider: &mockResourceProvider{
				uri:            "fluxbase://template/{id}",
				name:           "Template Resource",
				requiredScopes: []string{},
			},
			isTemplate: true,
		})

		authCtx := &AuthContext{IsServiceRole: true}
		resources := registry.ListResources(authCtx)

		assert.Len(t, resources, 1)
		assert.Equal(t, "Static Resource", resources[0].Name)
	})

	t.Run("filters by scope", func(t *testing.T) {
		registry := NewResourceRegistry()

		registry.Register(&mockResourceProvider{
			uri:            "fluxbase://public",
			name:           "Public",
			requiredScopes: []string{},
		})
		registry.Register(&mockResourceProvider{
			uri:            "fluxbase://admin",
			name:           "Admin",
			requiredScopes: []string{"admin:ddl"},
		})

		authCtx := &AuthContext{Scopes: []string{}}
		resources := registry.ListResources(authCtx)

		assert.Len(t, resources, 1)
		assert.Equal(t, "Public", resources[0].Name)
	})
}

func TestResourceRegistry_ListTemplates(t *testing.T) {
	t.Run("lists only templates", func(t *testing.T) {
		registry := NewResourceRegistry()

		registry.Register(&mockResourceProvider{
			uri:            "fluxbase://static",
			name:           "Static",
			requiredScopes: []string{},
		})
		registry.Register(&mockTemplateResourceProvider{
			mockResourceProvider: &mockResourceProvider{
				uri:            "fluxbase://template/{id}",
				name:           "Template",
				requiredScopes: []string{},
			},
			isTemplate: true,
		})

		authCtx := &AuthContext{IsServiceRole: true}
		templates := registry.ListTemplates(authCtx)

		assert.Len(t, templates, 1)
		assert.Equal(t, "Template", templates[0].Name)
	})

	t.Run("filters by scope", func(t *testing.T) {
		registry := NewResourceRegistry()

		registry.Register(&mockTemplateResourceProvider{
			mockResourceProvider: &mockResourceProvider{
				uri:            "fluxbase://public/{id}",
				name:           "Public Template",
				requiredScopes: []string{},
			},
			isTemplate: true,
		})
		registry.Register(&mockTemplateResourceProvider{
			mockResourceProvider: &mockResourceProvider{
				uri:            "fluxbase://admin/{id}",
				name:           "Admin Template",
				requiredScopes: []string{"admin:ddl"},
			},
			isTemplate: true,
		})

		authCtx := &AuthContext{Scopes: []string{}}
		templates := registry.ListTemplates(authCtx)

		assert.Len(t, templates, 1)
		assert.Equal(t, "Public Template", templates[0].Name)
	})
}

func TestResourceRegistry_ReadResource(t *testing.T) {
	t.Run("reads static resource", func(t *testing.T) {
		registry := NewResourceRegistry()

		registry.Register(&mockResourceProvider{
			uri:            "fluxbase://test",
			requiredScopes: []string{},
			readFunc: func(ctx context.Context, authCtx *AuthContext) ([]Content, error) {
				return []Content{{Type: "text", Text: "test content"}}, nil
			},
		})

		authCtx := &AuthContext{IsServiceRole: true}
		contents, err := registry.ReadResource(context.Background(), "fluxbase://test", authCtx)

		require.NoError(t, err)
		assert.Len(t, contents, 1)
		assert.Equal(t, "test content", contents[0].Text)
	})

	t.Run("reads template resource with params", func(t *testing.T) {
		registry := NewResourceRegistry()

		registry.Register(&mockTemplateResourceProvider{
			mockResourceProvider: &mockResourceProvider{
				uri:            "fluxbase://table/{name}",
				requiredScopes: []string{},
			},
			isTemplate: true,
			matchFunc: func(uri string) (map[string]string, bool) {
				if uri == "fluxbase://table/users" {
					return map[string]string{"name": "users"}, true
				}
				return nil, false
			},
			readParamsFunc: func(ctx context.Context, authCtx *AuthContext, params map[string]string) ([]Content, error) {
				return []Content{{Type: "text", Text: "table: " + params["name"]}}, nil
			},
		})

		authCtx := &AuthContext{IsServiceRole: true}
		contents, err := registry.ReadResource(context.Background(), "fluxbase://table/users", authCtx)

		require.NoError(t, err)
		assert.Len(t, contents, 1)
		assert.Equal(t, "table: users", contents[0].Text)
	})

	t.Run("error for not found", func(t *testing.T) {
		registry := NewResourceRegistry()

		authCtx := &AuthContext{IsServiceRole: true}
		_, err := registry.ReadResource(context.Background(), "nonexistent", authCtx)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "resource not found")
	})

	t.Run("error for missing scope", func(t *testing.T) {
		registry := NewResourceRegistry()

		registry.Register(&mockResourceProvider{
			uri:            "fluxbase://admin",
			requiredScopes: []string{"admin:ddl"},
		})

		authCtx := &AuthContext{Scopes: []string{}}
		_, err := registry.ReadResource(context.Background(), "fluxbase://admin", authCtx)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "access denied")
	})
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkToolRegistry_GetTool(b *testing.B) {
	registry := NewToolRegistry()
	for i := 0; i < 10; i++ {
		registry.Register(&mockToolHandler{name: "tool-" + string(rune('0'+i))})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = registry.GetTool("tool-5")
	}
}

func BenchmarkToolRegistry_ListTools(b *testing.B) {
	registry := NewToolRegistry()
	for i := 0; i < 20; i++ {
		registry.Register(&mockToolHandler{
			name:           "tool-" + string(rune('0'+i)),
			requiredScopes: []string{"read:tables"},
		})
	}

	authCtx := &AuthContext{Scopes: []string{"read:tables"}}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = registry.ListTools(authCtx)
	}
}

func BenchmarkResourceRegistry_GetProvider(b *testing.B) {
	registry := NewResourceRegistry()
	for i := 0; i < 10; i++ {
		registry.Register(&mockResourceProvider{uri: "fluxbase://resource-" + string(rune('0'+i))})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = registry.GetProvider("fluxbase://resource-5")
	}
}

func BenchmarkResourceRegistry_ReadResource(b *testing.B) {
	registry := NewResourceRegistry()
	registry.Register(&mockResourceProvider{
		uri:            "fluxbase://test",
		requiredScopes: []string{},
		readFunc: func(ctx context.Context, authCtx *AuthContext) ([]Content, error) {
			return []Content{{Type: "text", Text: "content"}}, nil
		},
	})

	authCtx := &AuthContext{IsServiceRole: true}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = registry.ReadResource(ctx, "fluxbase://test", authCtx)
	}
}
