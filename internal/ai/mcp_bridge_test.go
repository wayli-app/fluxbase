package ai

import (
	"testing"

	"github.com/fluxbase-eu/fluxbase/internal/mcp"
	"github.com/stretchr/testify/assert"
)

func TestChatbotAuthContext(t *testing.T) {
	t.Run("creates auth context with user info", func(t *testing.T) {
		userID := "user-123"
		chatCtx := &ChatContext{
			UserID: &userID,
			Role:   "authenticated",
		}
		chatbot := &Chatbot{
			MCPTools: []string{"query_table", "insert_record"},
		}

		authCtx := ChatbotAuthContext(chatCtx, chatbot)

		assert.Equal(t, &userID, authCtx.UserID)
		assert.Equal(t, "authenticated", authCtx.UserRole)
		assert.Equal(t, "chatbot", authCtx.AuthType)
		assert.False(t, authCtx.IsServiceRole)
		assert.Contains(t, authCtx.Scopes, mcp.ScopeReadTables)
		assert.Contains(t, authCtx.Scopes, mcp.ScopeWriteTables)
	})

	t.Run("handles nil user ID", func(t *testing.T) {
		chatCtx := &ChatContext{
			UserID: nil,
			Role:   "anon",
		}
		chatbot := &Chatbot{
			MCPTools: []string{"query_table"},
		}

		authCtx := ChatbotAuthContext(chatCtx, chatbot)

		assert.Nil(t, authCtx.UserID)
		assert.Equal(t, "anon", authCtx.UserRole)
	})

	t.Run("derives scopes from MCP tools", func(t *testing.T) {
		userID := "user-123"
		chatCtx := &ChatContext{
			UserID: &userID,
			Role:   "authenticated",
		}
		chatbot := &Chatbot{
			MCPTools: []string{"query_table", "invoke_function", "list_objects"},
		}

		authCtx := ChatbotAuthContext(chatCtx, chatbot)

		assert.Contains(t, authCtx.Scopes, mcp.ScopeReadTables)
		assert.Contains(t, authCtx.Scopes, mcp.ScopeExecuteFunctions)
		assert.Contains(t, authCtx.Scopes, mcp.ScopeReadStorage)
	})

	t.Run("empty MCP tools results in empty scopes", func(t *testing.T) {
		userID := "user-123"
		chatCtx := &ChatContext{
			UserID: &userID,
			Role:   "authenticated",
		}
		chatbot := &Chatbot{
			MCPTools: []string{},
		}

		authCtx := ChatbotAuthContext(chatCtx, chatbot)

		assert.Empty(t, authCtx.Scopes)
	})
}

func TestValidateChatbotToolAccess(t *testing.T) {
	t.Run("no MCP tools configured returns nil", func(t *testing.T) {
		chatbot := &Chatbot{MCPTools: nil}
		err := ValidateChatbotToolAccess(chatbot, "query_table")
		assert.NoError(t, err)
	})

	t.Run("allowed tool returns nil", func(t *testing.T) {
		chatbot := &Chatbot{MCPTools: []string{"query_table", "insert_record"}}
		err := ValidateChatbotToolAccess(chatbot, "query_table")
		assert.NoError(t, err)
	})

	t.Run("not allowed tool returns error", func(t *testing.T) {
		chatbot := &Chatbot{MCPTools: []string{"query_table"}}
		err := ValidateChatbotToolAccess(chatbot, "delete_record")
		assert.Error(t, err)

		var toolErr *ToolNotAllowedError
		assert.ErrorAs(t, err, &toolErr)
		assert.Equal(t, "delete_record", toolErr.Tool)
	})
}

func TestIsTableAllowed(t *testing.T) {
	t.Run("simple table name allowed", func(t *testing.T) {
		chatbot := &Chatbot{
			AllowedTables:  []string{"users", "orders"},
			AllowedSchemas: []string{"public"},
		}
		assert.True(t, IsTableAllowed("users", chatbot))
		assert.True(t, IsTableAllowed("orders", chatbot))
		assert.False(t, IsTableAllowed("products", chatbot))
	})

	t.Run("qualified table name allowed", func(t *testing.T) {
		chatbot := &Chatbot{
			AllowedTables:  []string{"analytics.metrics", "public.users"},
			AllowedSchemas: []string{"public", "analytics"},
		}
		assert.True(t, IsTableAllowed("analytics.metrics", chatbot))
		assert.True(t, IsTableAllowed("users", chatbot)) // defaults to public
		assert.False(t, IsTableAllowed("analytics.other", chatbot))
	})

	t.Run("schema-wide access", func(t *testing.T) {
		chatbot := &Chatbot{
			AllowedTables:  []string{}, // No specific tables
			AllowedSchemas: []string{"public", "analytics"},
		}
		assert.True(t, IsTableAllowed("users", chatbot))
		assert.True(t, IsTableAllowed("analytics.metrics", chatbot))
		assert.False(t, IsTableAllowed("private.secrets", chatbot))
	})

	t.Run("mixed schema and table restrictions", func(t *testing.T) {
		chatbot := &Chatbot{
			AllowedTables:  []string{"users", "analytics.metrics"},
			AllowedSchemas: []string{"public"},
		}
		// users in public schema - allowed via table list
		assert.True(t, IsTableAllowed("users", chatbot))
		// analytics.metrics - allowed via explicit table
		assert.True(t, IsTableAllowed("analytics.metrics", chatbot))
		// orders - not in allowed tables
		assert.False(t, IsTableAllowed("orders", chatbot))
	})

	t.Run("empty allowed tables with allowed schemas", func(t *testing.T) {
		chatbot := &Chatbot{
			AllowedTables:  []string{},
			AllowedSchemas: []string{"public"},
		}
		// All public tables should be allowed
		assert.True(t, IsTableAllowed("users", chatbot))
		assert.True(t, IsTableAllowed("orders", chatbot))
		// Other schemas not allowed
		assert.False(t, IsTableAllowed("private.secrets", chatbot))
	})
}

func TestToolNotAllowedError(t *testing.T) {
	err := &ToolNotAllowedError{
		Tool:         "delete_record",
		AllowedTools: []string{"query_table", "insert_record"},
	}

	assert.Equal(t, "tool 'delete_record' is not allowed for this chatbot", err.Error())
}

func TestTableNotAllowedError(t *testing.T) {
	err := &TableNotAllowedError{
		Table:         "secrets",
		AllowedTables: []string{"users", "orders"},
	}

	assert.Equal(t, "table 'secrets' is not allowed for this chatbot", err.Error())
}
