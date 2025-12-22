package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsValidScope(t *testing.T) {
	t.Run("all defined scopes are valid", func(t *testing.T) {
		// Test all constant scopes
		validScopes := []string{
			ScopeTablesRead,
			ScopeTablesWrite,
			ScopeStorageRead,
			ScopeStorageWrite,
			ScopeFunctionsRead,
			ScopeFunctionsExecute,
			ScopeAuthRead,
			ScopeAuthWrite,
			ScopeAPIKeysRead,
			ScopeAPIKeysWrite,
			ScopeWebhooksRead,
			ScopeWebhooksWrite,
			ScopeMonitoringRead,
			ScopeRealtimeConnect,
			ScopeRealtimeBroadcast,
			ScopeRPCRead,
			ScopeRPCExecute,
			ScopeJobsRead,
			ScopeJobsWrite,
			ScopeAIRead,
			ScopeAIWrite,
			ScopeWildcard,
		}

		for _, scope := range validScopes {
			assert.True(t, IsValidScope(scope), "scope %s should be valid", scope)
		}
	})

	t.Run("invalid scopes return false", func(t *testing.T) {
		invalidScopes := []string{
			"",
			"invalid",
			"read:",
			":tables",
			"READ:TABLES", // case sensitive
			"read:nonexistent",
			"write:nonexistent",
			"execute:nonexistent",
			"**",
			"admin",
			"superuser",
		}

		for _, scope := range invalidScopes {
			assert.False(t, IsValidScope(scope), "scope %s should be invalid", scope)
		}
	})

	t.Run("wildcard scope is valid", func(t *testing.T) {
		assert.True(t, IsValidScope("*"))
		assert.True(t, IsValidScope(ScopeWildcard))
	})
}

func TestValidateScopes(t *testing.T) {
	t.Run("empty scopes returns error", func(t *testing.T) {
		err := ValidateScopes([]string{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "at least one scope must be specified")
	})

	t.Run("nil scopes returns error", func(t *testing.T) {
		err := ValidateScopes(nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "at least one scope must be specified")
	})

	t.Run("single valid scope passes", func(t *testing.T) {
		err := ValidateScopes([]string{ScopeTablesRead})
		assert.NoError(t, err)
	})

	t.Run("multiple valid scopes pass", func(t *testing.T) {
		err := ValidateScopes([]string{
			ScopeTablesRead,
			ScopeTablesWrite,
			ScopeStorageRead,
		})
		assert.NoError(t, err)
	})

	t.Run("wildcard scope alone passes", func(t *testing.T) {
		err := ValidateScopes([]string{ScopeWildcard})
		assert.NoError(t, err)
	})

	t.Run("all scopes pass validation", func(t *testing.T) {
		err := ValidateScopes(AllScopes)
		assert.NoError(t, err)
	})

	t.Run("single invalid scope fails", func(t *testing.T) {
		err := ValidateScopes([]string{"invalid:scope"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid scopes")
		assert.Contains(t, err.Error(), "invalid:scope")
	})

	t.Run("mixed valid and invalid scopes fails", func(t *testing.T) {
		err := ValidateScopes([]string{
			ScopeTablesRead,
			"invalid:scope",
			ScopeStorageRead,
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid scopes")
		assert.Contains(t, err.Error(), "invalid:scope")
	})

	t.Run("multiple invalid scopes all reported", func(t *testing.T) {
		err := ValidateScopes([]string{
			"invalid1",
			ScopeTablesRead,
			"invalid2",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid1")
		assert.Contains(t, err.Error(), "invalid2")
	})
}

func TestHasScope(t *testing.T) {
	t.Run("returns true when scope is present", func(t *testing.T) {
		scopes := []string{ScopeTablesRead, ScopeStorageRead}
		assert.True(t, HasScope(scopes, ScopeTablesRead))
		assert.True(t, HasScope(scopes, ScopeStorageRead))
	})

	t.Run("returns false when scope is absent", func(t *testing.T) {
		scopes := []string{ScopeTablesRead, ScopeStorageRead}
		assert.False(t, HasScope(scopes, ScopeTablesWrite))
		assert.False(t, HasScope(scopes, ScopeAuthRead))
	})

	t.Run("wildcard grants any scope", func(t *testing.T) {
		scopes := []string{ScopeWildcard}

		// Wildcard should grant access to any scope
		assert.True(t, HasScope(scopes, ScopeTablesRead))
		assert.True(t, HasScope(scopes, ScopeTablesWrite))
		assert.True(t, HasScope(scopes, ScopeStorageRead))
		assert.True(t, HasScope(scopes, ScopeAuthWrite))
		assert.True(t, HasScope(scopes, ScopeRPCExecute))
	})

	t.Run("empty scopes returns false", func(t *testing.T) {
		assert.False(t, HasScope([]string{}, ScopeTablesRead))
		assert.False(t, HasScope(nil, ScopeTablesRead))
	})

	t.Run("wildcard with other scopes still works", func(t *testing.T) {
		scopes := []string{ScopeTablesRead, ScopeWildcard}
		assert.True(t, HasScope(scopes, ScopeStorageWrite))
	})
}

func TestHasAllScopes(t *testing.T) {
	t.Run("returns true when all scopes present", func(t *testing.T) {
		scopes := []string{ScopeTablesRead, ScopeStorageRead, ScopeAuthRead}
		required := []string{ScopeTablesRead, ScopeStorageRead}

		assert.True(t, HasAllScopes(scopes, required))
	})

	t.Run("returns true when exact match", func(t *testing.T) {
		scopes := []string{ScopeTablesRead, ScopeStorageRead}
		required := []string{ScopeTablesRead, ScopeStorageRead}

		assert.True(t, HasAllScopes(scopes, required))
	})

	t.Run("returns false when missing one scope", func(t *testing.T) {
		scopes := []string{ScopeTablesRead, ScopeStorageRead}
		required := []string{ScopeTablesRead, ScopeAuthRead}

		assert.False(t, HasAllScopes(scopes, required))
	})

	t.Run("returns false when missing all scopes", func(t *testing.T) {
		scopes := []string{ScopeTablesRead}
		required := []string{ScopeStorageRead, ScopeAuthRead}

		assert.False(t, HasAllScopes(scopes, required))
	})

	t.Run("wildcard grants all scopes", func(t *testing.T) {
		scopes := []string{ScopeWildcard}
		required := []string{
			ScopeTablesRead,
			ScopeTablesWrite,
			ScopeStorageRead,
			ScopeAuthWrite,
		}

		assert.True(t, HasAllScopes(scopes, required))
	})

	t.Run("empty required returns true", func(t *testing.T) {
		scopes := []string{ScopeTablesRead}
		assert.True(t, HasAllScopes(scopes, []string{}))
		assert.True(t, HasAllScopes(scopes, nil))
	})

	t.Run("empty scopes with requirements returns false", func(t *testing.T) {
		assert.False(t, HasAllScopes([]string{}, []string{ScopeTablesRead}))
		assert.False(t, HasAllScopes(nil, []string{ScopeTablesRead}))
	})

	t.Run("empty scopes with empty required returns true", func(t *testing.T) {
		assert.True(t, HasAllScopes([]string{}, []string{}))
		assert.True(t, HasAllScopes(nil, nil))
	})
}

func TestAllScopes(t *testing.T) {
	t.Run("AllScopes contains expected count", func(t *testing.T) {
		// 21 scopes: 2 tables + 2 storage + 2 functions + 2 auth + 2 apikeys +
		// 2 webhooks + 1 monitoring + 2 realtime + 2 rpc + 2 jobs + 2 ai
		assert.Len(t, AllScopes, 21)
	})

	t.Run("AllScopes does not contain wildcard", func(t *testing.T) {
		for _, scope := range AllScopes {
			assert.NotEqual(t, ScopeWildcard, scope)
		}
	})

	t.Run("all scopes in AllScopes are unique", func(t *testing.T) {
		seen := make(map[string]bool)
		for _, scope := range AllScopes {
			if seen[scope] {
				t.Errorf("duplicate scope found: %s", scope)
			}
			seen[scope] = true
		}
	})
}

func TestScopeConstants(t *testing.T) {
	t.Run("scope constants have expected format", func(t *testing.T) {
		// Tables
		assert.Equal(t, "read:tables", ScopeTablesRead)
		assert.Equal(t, "write:tables", ScopeTablesWrite)

		// Storage
		assert.Equal(t, "read:storage", ScopeStorageRead)
		assert.Equal(t, "write:storage", ScopeStorageWrite)

		// Functions
		assert.Equal(t, "read:functions", ScopeFunctionsRead)
		assert.Equal(t, "execute:functions", ScopeFunctionsExecute)

		// Auth
		assert.Equal(t, "read:auth", ScopeAuthRead)
		assert.Equal(t, "write:auth", ScopeAuthWrite)

		// API Keys
		assert.Equal(t, "read:apikeys", ScopeAPIKeysRead)
		assert.Equal(t, "write:apikeys", ScopeAPIKeysWrite)

		// Webhooks
		assert.Equal(t, "read:webhooks", ScopeWebhooksRead)
		assert.Equal(t, "write:webhooks", ScopeWebhooksWrite)

		// Monitoring
		assert.Equal(t, "read:monitoring", ScopeMonitoringRead)

		// Realtime
		assert.Equal(t, "realtime:connect", ScopeRealtimeConnect)
		assert.Equal(t, "realtime:broadcast", ScopeRealtimeBroadcast)

		// RPC
		assert.Equal(t, "read:rpc", ScopeRPCRead)
		assert.Equal(t, "execute:rpc", ScopeRPCExecute)

		// Jobs
		assert.Equal(t, "read:jobs", ScopeJobsRead)
		assert.Equal(t, "write:jobs", ScopeJobsWrite)

		// AI
		assert.Equal(t, "read:ai", ScopeAIRead)
		assert.Equal(t, "write:ai", ScopeAIWrite)

		// Wildcard
		assert.Equal(t, "*", ScopeWildcard)
	})
}

func TestScopeUseCases(t *testing.T) {
	t.Run("read-only API key", func(t *testing.T) {
		readOnlyScopes := []string{
			ScopeTablesRead,
			ScopeStorageRead,
			ScopeFunctionsRead,
			ScopeMonitoringRead,
		}

		// Should have read access
		assert.True(t, HasScope(readOnlyScopes, ScopeTablesRead))
		assert.True(t, HasScope(readOnlyScopes, ScopeStorageRead))

		// Should NOT have write access
		assert.False(t, HasScope(readOnlyScopes, ScopeTablesWrite))
		assert.False(t, HasScope(readOnlyScopes, ScopeStorageWrite))
		assert.False(t, HasScope(readOnlyScopes, ScopeAuthWrite))
	})

	t.Run("full access API key", func(t *testing.T) {
		fullAccessScopes := []string{ScopeWildcard}

		// Should have all access
		assert.True(t, HasScope(fullAccessScopes, ScopeTablesRead))
		assert.True(t, HasScope(fullAccessScopes, ScopeTablesWrite))
		assert.True(t, HasScope(fullAccessScopes, ScopeAuthWrite))
		assert.True(t, HasScope(fullAccessScopes, ScopeAPIKeysWrite))
	})

	t.Run("function execution only", func(t *testing.T) {
		functionScopes := []string{
			ScopeFunctionsRead,
			ScopeFunctionsExecute,
		}

		// Should have function access
		assert.True(t, HasScope(functionScopes, ScopeFunctionsExecute))

		// Should NOT have other access
		assert.False(t, HasScope(functionScopes, ScopeTablesRead))
		assert.False(t, HasScope(functionScopes, ScopeStorageRead))
	})

	t.Run("realtime client", func(t *testing.T) {
		realtimeScopes := []string{
			ScopeRealtimeConnect,
			ScopeRealtimeBroadcast,
			ScopeTablesRead,
		}

		// Validate all scopes are valid
		err := ValidateScopes(realtimeScopes)
		assert.NoError(t, err)

		// Check realtime access
		assert.True(t, HasScope(realtimeScopes, ScopeRealtimeConnect))
		assert.True(t, HasScope(realtimeScopes, ScopeRealtimeBroadcast))
	})

	t.Run("multiple required scopes for complex operation", func(t *testing.T) {
		// API operation that needs both read tables and execute RPC
		operationScopes := []string{ScopeTablesRead, ScopeRPCExecute}

		// User with exact required scopes
		userScopes1 := []string{ScopeTablesRead, ScopeRPCExecute}
		assert.True(t, HasAllScopes(userScopes1, operationScopes))

		// User with extra scopes
		userScopes2 := []string{ScopeTablesRead, ScopeTablesWrite, ScopeRPCExecute, ScopeRPCRead}
		assert.True(t, HasAllScopes(userScopes2, operationScopes))

		// User missing one required scope
		userScopes3 := []string{ScopeTablesRead}
		assert.False(t, HasAllScopes(userScopes3, operationScopes))

		// User with wildcard
		userScopes4 := []string{ScopeWildcard}
		assert.True(t, HasAllScopes(userScopes4, operationScopes))
	})
}
