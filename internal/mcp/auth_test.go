package mcp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// =============================================================================
// AuthContext.HasScope Tests
// =============================================================================

func TestAuthContext_HasScope(t *testing.T) {
	tests := []struct {
		name          string
		ctx           *AuthContext
		scope         string
		expectedHas   bool
	}{
		{
			name: "service role has all scopes",
			ctx: &AuthContext{
				IsServiceRole: true,
				Scopes:        []string{},
			},
			scope:       "any:scope",
			expectedHas: true,
		},
		{
			name: "wildcard scope grants all",
			ctx: &AuthContext{
				Scopes: []string{"*"},
			},
			scope:       "read:tables",
			expectedHas: true,
		},
		{
			name: "exact scope match",
			ctx: &AuthContext{
				Scopes: []string{"read:tables", "write:tables"},
			},
			scope:       "read:tables",
			expectedHas: true,
		},
		{
			name: "scope not present",
			ctx: &AuthContext{
				Scopes: []string{"read:tables"},
			},
			scope:       "write:tables",
			expectedHas: false,
		},
		{
			name: "empty scopes",
			ctx: &AuthContext{
				Scopes: []string{},
			},
			scope:       "read:tables",
			expectedHas: false,
		},
		{
			name: "nil scopes",
			ctx: &AuthContext{
				Scopes: nil,
			},
			scope:       "read:tables",
			expectedHas: false,
		},
		{
			name: "service role overrides empty scopes",
			ctx: &AuthContext{
				IsServiceRole: true,
				Scopes:        nil,
			},
			scope:       "admin:ddl",
			expectedHas: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.ctx.HasScope(tt.scope)
			assert.Equal(t, tt.expectedHas, result)
		})
	}
}

// =============================================================================
// AuthContext.HasScopes Tests
// =============================================================================

func TestAuthContext_HasScopes(t *testing.T) {
	tests := []struct {
		name        string
		ctx         *AuthContext
		scopes      []string
		expectedHas bool
	}{
		{
			name: "has all required scopes",
			ctx: &AuthContext{
				Scopes: []string{"read:tables", "write:tables", "execute:functions"},
			},
			scopes:      []string{"read:tables", "write:tables"},
			expectedHas: true,
		},
		{
			name: "missing one scope",
			ctx: &AuthContext{
				Scopes: []string{"read:tables"},
			},
			scopes:      []string{"read:tables", "write:tables"},
			expectedHas: false,
		},
		{
			name: "service role has all",
			ctx: &AuthContext{
				IsServiceRole: true,
			},
			scopes:      []string{"read:tables", "write:tables", "admin:ddl"},
			expectedHas: true,
		},
		{
			name: "wildcard grants all",
			ctx: &AuthContext{
				Scopes: []string{"*"},
			},
			scopes:      []string{"read:tables", "write:tables"},
			expectedHas: true,
		},
		{
			name: "empty required scopes",
			ctx: &AuthContext{
				Scopes: []string{"read:tables"},
			},
			scopes:      []string{},
			expectedHas: true,
		},
		{
			name: "no scopes available",
			ctx: &AuthContext{
				Scopes: []string{},
			},
			scopes:      []string{"read:tables"},
			expectedHas: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.ctx.HasScopes(tt.scopes...)
			assert.Equal(t, tt.expectedHas, result)
		})
	}
}

// =============================================================================
// AuthContext.HasAnyScope Tests
// =============================================================================

func TestAuthContext_HasAnyScope(t *testing.T) {
	tests := []struct {
		name        string
		ctx         *AuthContext
		scopes      []string
		expectedHas bool
	}{
		{
			name: "has first scope",
			ctx: &AuthContext{
				Scopes: []string{"read:tables"},
			},
			scopes:      []string{"read:tables", "write:tables"},
			expectedHas: true,
		},
		{
			name: "has second scope",
			ctx: &AuthContext{
				Scopes: []string{"write:tables"},
			},
			scopes:      []string{"read:tables", "write:tables"},
			expectedHas: true,
		},
		{
			name: "has neither scope",
			ctx: &AuthContext{
				Scopes: []string{"execute:functions"},
			},
			scopes:      []string{"read:tables", "write:tables"},
			expectedHas: false,
		},
		{
			name: "service role has any",
			ctx: &AuthContext{
				IsServiceRole: true,
			},
			scopes:      []string{"admin:ddl"},
			expectedHas: true,
		},
		{
			name: "empty required scopes returns false",
			ctx: &AuthContext{
				Scopes: []string{"read:tables"},
			},
			scopes:      []string{},
			expectedHas: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.ctx.HasAnyScope(tt.scopes...)
			assert.Equal(t, tt.expectedHas, result)
		})
	}
}

// =============================================================================
// AuthContext.IsAuthenticated Tests
// =============================================================================

func TestAuthContext_IsAuthenticated(t *testing.T) {
	userID := "user-123"

	tests := []struct {
		name     string
		ctx      *AuthContext
		expected bool
	}{
		{
			name: "authenticated with user ID",
			ctx: &AuthContext{
				UserID: &userID,
			},
			expected: true,
		},
		{
			name: "authenticated with service key",
			ctx: &AuthContext{
				AuthType: "service_key",
			},
			expected: true,
		},
		{
			name: "authenticated with both",
			ctx: &AuthContext{
				UserID:   &userID,
				AuthType: "service_key",
			},
			expected: true,
		},
		{
			name: "not authenticated - nil user ID",
			ctx: &AuthContext{
				AuthType: "jwt",
			},
			expected: false,
		},
		{
			name: "not authenticated - empty context",
			ctx:      &AuthContext{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.ctx.IsAuthenticated()
			assert.Equal(t, tt.expected, result)
		})
	}
}

// =============================================================================
// AuthContext.GetMetadata Tests
// =============================================================================

func TestAuthContext_GetMetadata(t *testing.T) {
	t.Run("nil metadata returns nil", func(t *testing.T) {
		ctx := &AuthContext{}
		result := ctx.GetMetadata("key")
		assert.Nil(t, result)
	})

	t.Run("empty metadata returns nil", func(t *testing.T) {
		ctx := &AuthContext{
			Metadata: make(map[string]any),
		}
		result := ctx.GetMetadata("key")
		assert.Nil(t, result)
	})

	t.Run("returns existing value", func(t *testing.T) {
		ctx := &AuthContext{
			Metadata: map[string]any{
				"key": "value",
			},
		}
		result := ctx.GetMetadata("key")
		assert.Equal(t, "value", result)
	})

	t.Run("missing key returns nil", func(t *testing.T) {
		ctx := &AuthContext{
			Metadata: map[string]any{
				"key": "value",
			},
		}
		result := ctx.GetMetadata("other")
		assert.Nil(t, result)
	})
}

// =============================================================================
// AuthContext.GetMetadataStringSlice Tests
// =============================================================================

func TestAuthContext_GetMetadataStringSlice(t *testing.T) {
	t.Run("nil metadata returns nil", func(t *testing.T) {
		ctx := &AuthContext{}
		result := ctx.GetMetadataStringSlice("key")
		assert.Nil(t, result)
	})

	t.Run("missing key returns nil", func(t *testing.T) {
		ctx := &AuthContext{
			Metadata: map[string]any{},
		}
		result := ctx.GetMetadataStringSlice("key")
		assert.Nil(t, result)
	})

	t.Run("returns string slice", func(t *testing.T) {
		ctx := &AuthContext{
			Metadata: map[string]any{
				"domains": []string{"example.com", "test.com"},
			},
		}
		result := ctx.GetMetadataStringSlice("domains")
		assert.Equal(t, []string{"example.com", "test.com"}, result)
	})

	t.Run("wrong type returns nil", func(t *testing.T) {
		ctx := &AuthContext{
			Metadata: map[string]any{
				"domains": "not-a-slice",
			},
		}
		result := ctx.GetMetadataStringSlice("domains")
		assert.Nil(t, result)
	})

	t.Run("interface slice returns nil", func(t *testing.T) {
		ctx := &AuthContext{
			Metadata: map[string]any{
				"domains": []interface{}{"example.com"},
			},
		}
		result := ctx.GetMetadataStringSlice("domains")
		assert.Nil(t, result)
	})
}

// =============================================================================
// AuthContext.HasNamespaceAccess Tests
// =============================================================================

func TestAuthContext_HasNamespaceAccess(t *testing.T) {
	tests := []struct {
		name       string
		ctx        *AuthContext
		namespace  string
		expected   bool
	}{
		{
			name: "service role bypasses all checks",
			ctx: &AuthContext{
				IsServiceRole:     true,
				AllowedNamespaces: []string{"specific-ns"},
			},
			namespace: "any-namespace",
			expected:  true,
		},
		{
			name: "nil allowed namespaces allows all",
			ctx: &AuthContext{
				AllowedNamespaces: nil,
			},
			namespace: "any-namespace",
			expected:  true,
		},
		{
			name: "empty allowed namespaces allows only default",
			ctx: &AuthContext{
				AllowedNamespaces: []string{},
			},
			namespace: "default",
			expected:  true,
		},
		{
			name: "empty allowed namespaces denies non-default",
			ctx: &AuthContext{
				AllowedNamespaces: []string{},
			},
			namespace: "other",
			expected:  false,
		},
		{
			name: "namespace in allowed list",
			ctx: &AuthContext{
				AllowedNamespaces: []string{"ns-1", "ns-2"},
			},
			namespace: "ns-1",
			expected:  true,
		},
		{
			name: "namespace not in allowed list",
			ctx: &AuthContext{
				AllowedNamespaces: []string{"ns-1", "ns-2"},
			},
			namespace: "ns-3",
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.ctx.HasNamespaceAccess(tt.namespace)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// =============================================================================
// AuthContext.FilterNamespaces Tests
// =============================================================================

func TestAuthContext_FilterNamespaces(t *testing.T) {
	t.Run("service role returns all", func(t *testing.T) {
		ctx := &AuthContext{
			IsServiceRole:     true,
			AllowedNamespaces: []string{"specific"},
		}
		namespaces := []string{"ns-1", "ns-2", "ns-3"}
		result := ctx.FilterNamespaces(namespaces)
		assert.Equal(t, namespaces, result)
	})

	t.Run("nil allowed namespaces returns all", func(t *testing.T) {
		ctx := &AuthContext{
			AllowedNamespaces: nil,
		}
		namespaces := []string{"ns-1", "ns-2"}
		result := ctx.FilterNamespaces(namespaces)
		assert.Equal(t, namespaces, result)
	})

	t.Run("filters to allowed only", func(t *testing.T) {
		ctx := &AuthContext{
			AllowedNamespaces: []string{"ns-1", "ns-3"},
		}
		namespaces := []string{"ns-1", "ns-2", "ns-3", "ns-4"}
		result := ctx.FilterNamespaces(namespaces)
		assert.Equal(t, []string{"ns-1", "ns-3"}, result)
	})

	t.Run("empty allowed only returns default", func(t *testing.T) {
		ctx := &AuthContext{
			AllowedNamespaces: []string{},
		}
		namespaces := []string{"default", "ns-1", "ns-2"}
		result := ctx.FilterNamespaces(namespaces)
		assert.Equal(t, []string{"default"}, result)
	})

	t.Run("empty input returns empty", func(t *testing.T) {
		ctx := &AuthContext{
			AllowedNamespaces: []string{"ns-1"},
		}
		result := ctx.FilterNamespaces([]string{})
		assert.Empty(t, result)
	})
}

// =============================================================================
// inferScopesFromRole Tests
// =============================================================================

func TestInferScopesFromRole(t *testing.T) {
	tests := []struct {
		name           string
		role           string
		expectedScopes []string
	}{
		{
			name:           "admin gets wildcard",
			role:           "admin",
			expectedScopes: []string{"*"},
		},
		{
			name:           "dashboard_admin gets wildcard",
			role:           "dashboard_admin",
			expectedScopes: []string{"*"},
		},
		{
			name: "authenticated gets read/write scopes",
			role: "authenticated",
			expectedScopes: []string{
				"read:tables",
				"write:tables",
				"execute:functions",
				"execute:rpc",
				"read:storage",
				"write:storage",
				"execute:jobs",
			},
		},
		{
			name:           "anon gets only read:tables",
			role:           "anon",
			expectedScopes: []string{"read:tables"},
		},
		{
			name:           "unknown role gets empty scopes",
			role:           "custom_role",
			expectedScopes: []string{},
		},
		{
			name:           "empty role gets empty scopes",
			role:           "",
			expectedScopes: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := inferScopesFromRole(tt.role)
			assert.Equal(t, tt.expectedScopes, result)
		})
	}
}

// =============================================================================
// Scope Constants Tests
// =============================================================================

func TestScopeConstants(t *testing.T) {
	t.Run("table scopes", func(t *testing.T) {
		assert.Equal(t, "read:tables", ScopeReadTables)
		assert.Equal(t, "write:tables", ScopeWriteTables)
	})

	t.Run("function scopes", func(t *testing.T) {
		assert.Equal(t, "execute:functions", ScopeExecuteFunctions)
		assert.Equal(t, ScopeExecuteFunctions, ScopeInvokeFunctions) // Alias
	})

	t.Run("RPC scopes", func(t *testing.T) {
		assert.Equal(t, "execute:rpc", ScopeExecuteRPC)
		assert.Equal(t, ScopeExecuteRPC, ScopeInvokeRPC) // Alias
	})

	t.Run("storage scopes", func(t *testing.T) {
		assert.Equal(t, "read:storage", ScopeReadStorage)
		assert.Equal(t, "write:storage", ScopeWriteStorage)
	})

	t.Run("job scopes", func(t *testing.T) {
		assert.Equal(t, "execute:jobs", ScopeExecuteJobs)
		assert.Equal(t, ScopeExecuteJobs, ScopeSubmitJobs) // Alias
	})

	t.Run("vector scopes", func(t *testing.T) {
		assert.Equal(t, "read:vectors", ScopeReadVectors)
		assert.Equal(t, ScopeReadVectors, ScopeSearchVectors) // Alias
	})

	t.Run("admin scopes", func(t *testing.T) {
		assert.Equal(t, "admin:schemas", ScopeAdminSchemas)
		assert.Equal(t, "admin:ddl", ScopeAdminDDL)
	})

	t.Run("sync scopes", func(t *testing.T) {
		assert.Equal(t, "sync:functions", ScopeSyncFunctions)
		assert.Equal(t, "sync:jobs", ScopeSyncJobs)
		assert.Equal(t, "sync:rpc", ScopeSyncRPC)
		assert.Equal(t, "sync:migrations", ScopeSyncMigrations)
		assert.Equal(t, "sync:chatbots", ScopeSyncChatbots)
	})

	t.Run("branching scopes", func(t *testing.T) {
		assert.Equal(t, "branch:read", ScopeBranchRead)
		assert.Equal(t, "branch:write", ScopeBranchWrite)
		assert.Equal(t, "branch:access", ScopeBranchAccess)
	})

	t.Run("metadata keys", func(t *testing.T) {
		assert.Equal(t, "http_allowed_domains", MetadataKeyHTTPAllowedDomains)
	})
}

// =============================================================================
// AuthContext Struct Tests
// =============================================================================

func TestAuthContext_Struct(t *testing.T) {
	userID := "user-123"

	t.Run("all fields set", func(t *testing.T) {
		ctx := &AuthContext{
			UserID:                 &userID,
			UserEmail:              "user@example.com",
			UserRole:               "authenticated",
			AuthType:               "jwt",
			ClientKeyID:            "key-123",
			ClientKeyName:          "my-key",
			Scopes:                 []string{"read:tables"},
			IsServiceRole:          false,
			AllowedNamespaces:      []string{"default"},
			IsImpersonating:        true,
			ImpersonationAdminID:   "admin-123",
			ImpersonationSessionID: "session-456",
			Metadata:               map[string]any{"key": "value"},
		}

		assert.Equal(t, &userID, ctx.UserID)
		assert.Equal(t, "user@example.com", ctx.UserEmail)
		assert.Equal(t, "authenticated", ctx.UserRole)
		assert.Equal(t, "jwt", ctx.AuthType)
		assert.Equal(t, "key-123", ctx.ClientKeyID)
		assert.Equal(t, "my-key", ctx.ClientKeyName)
		assert.Equal(t, []string{"read:tables"}, ctx.Scopes)
		assert.False(t, ctx.IsServiceRole)
		assert.Equal(t, []string{"default"}, ctx.AllowedNamespaces)
		assert.True(t, ctx.IsImpersonating)
		assert.Equal(t, "admin-123", ctx.ImpersonationAdminID)
		assert.Equal(t, "session-456", ctx.ImpersonationSessionID)
		assert.Equal(t, map[string]any{"key": "value"}, ctx.Metadata)
	})

	t.Run("zero value", func(t *testing.T) {
		ctx := &AuthContext{}

		assert.Nil(t, ctx.UserID)
		assert.Empty(t, ctx.UserEmail)
		assert.Empty(t, ctx.UserRole)
		assert.Empty(t, ctx.AuthType)
		assert.Empty(t, ctx.ClientKeyID)
		assert.Empty(t, ctx.ClientKeyName)
		assert.Nil(t, ctx.Scopes)
		assert.False(t, ctx.IsServiceRole)
		assert.Nil(t, ctx.AllowedNamespaces)
		assert.False(t, ctx.IsImpersonating)
		assert.Empty(t, ctx.ImpersonationAdminID)
		assert.Empty(t, ctx.ImpersonationSessionID)
		assert.Nil(t, ctx.Metadata)
	})
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkAuthContext_HasScope_ServiceRole(b *testing.B) {
	ctx := &AuthContext{
		IsServiceRole: true,
	}

	for i := 0; i < b.N; i++ {
		_ = ctx.HasScope("read:tables")
	}
}

func BenchmarkAuthContext_HasScope_Wildcard(b *testing.B) {
	ctx := &AuthContext{
		Scopes: []string{"*"},
	}

	for i := 0; i < b.N; i++ {
		_ = ctx.HasScope("read:tables")
	}
}

func BenchmarkAuthContext_HasScope_ExactMatch(b *testing.B) {
	ctx := &AuthContext{
		Scopes: []string{"read:tables", "write:tables", "execute:functions", "execute:rpc"},
	}

	for i := 0; i < b.N; i++ {
		_ = ctx.HasScope("execute:rpc")
	}
}

func BenchmarkAuthContext_HasScopes_Multiple(b *testing.B) {
	ctx := &AuthContext{
		Scopes: []string{"read:tables", "write:tables", "execute:functions", "execute:rpc"},
	}

	for i := 0; i < b.N; i++ {
		_ = ctx.HasScopes("read:tables", "write:tables")
	}
}

func BenchmarkAuthContext_FilterNamespaces(b *testing.B) {
	ctx := &AuthContext{
		AllowedNamespaces: []string{"ns-1", "ns-2", "ns-3"},
	}
	namespaces := []string{"ns-1", "ns-2", "ns-3", "ns-4", "ns-5"}

	for i := 0; i < b.N; i++ {
		_ = ctx.FilterNamespaces(namespaces)
	}
}

func BenchmarkInferScopesFromRole_Authenticated(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = inferScopesFromRole("authenticated")
	}
}
