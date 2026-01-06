package tools

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// ============================================================================
// SYNC FUNCTION TESTS
// ============================================================================

func TestParseFluxbaseAnnotations(t *testing.T) {
	t.Run("parses basic annotations", func(t *testing.T) {
		code := `
// @fluxbase:description Handle user registration
// @fluxbase:public
// @fluxbase:allow-unauthenticated
export default async function handler(req: Request) {
  return new Response("Hello!");
}`
		config := parseFluxbaseAnnotations(code)

		assert.Equal(t, "Handle user registration", config.Description)
		assert.True(t, config.IsPublic)
		assert.True(t, config.AllowUnauthenticated)
	})

	t.Run("parses timeout and memory", func(t *testing.T) {
		code := `
// @fluxbase:timeout 60
// @fluxbase:memory 256
export default async function handler(req: Request) {}`
		config := parseFluxbaseAnnotations(code)

		assert.Equal(t, 60, config.Timeout)
		assert.Equal(t, 256, config.Memory)
	})

	t.Run("parses rate limiting", func(t *testing.T) {
		code := `
// @fluxbase:rate-limit 100/min
// @fluxbase:rate-limit 1000/hour
// @fluxbase:rate-limit 10000/day
export default async function handler(req: Request) {}`
		config := parseFluxbaseAnnotations(code)

		assert.Equal(t, 100, config.RateLimitPerMinute)
		assert.Equal(t, 1000, config.RateLimitPerHour)
		assert.Equal(t, 10000, config.RateLimitPerDay)
	})

	t.Run("parses CORS origins", func(t *testing.T) {
		code := `
// @fluxbase:cors-origins https://example.com,https://app.example.com
export default async function handler(req: Request) {}`
		config := parseFluxbaseAnnotations(code)

		assert.Equal(t, "https://example.com,https://app.example.com", config.CorsOrigins)
	})

	t.Run("parses deny annotations", func(t *testing.T) {
		code := `
// @fluxbase:deny-net
// @fluxbase:deny-env
export default async function handler(req: Request) {}`
		config := parseFluxbaseAnnotations(code)

		assert.False(t, config.AllowNet)
		assert.False(t, config.AllowEnv)
	})

	t.Run("parses disable-logs", func(t *testing.T) {
		code := `
// @fluxbase:disable-logs
export default async function handler(req: Request) {}`
		config := parseFluxbaseAnnotations(code)

		assert.True(t, config.DisableLogs)
	})

	t.Run("uses defaults when no annotations", func(t *testing.T) {
		code := `export default async function handler(req: Request) {}`
		config := parseFluxbaseAnnotations(code)

		assert.Equal(t, 30, config.Timeout)
		assert.Equal(t, 128, config.Memory)
		assert.True(t, config.AllowNet)
		assert.True(t, config.AllowEnv)
		assert.False(t, config.IsPublic)
		assert.False(t, config.AllowUnauthenticated)
	})

	t.Run("handles block comment style", func(t *testing.T) {
		code := `
/*
 * @fluxbase:description Block comment annotation
 * @fluxbase:public
 */
export default async function handler(req: Request) {}`
		config := parseFluxbaseAnnotations(code)

		assert.Equal(t, "Block comment annotation", config.Description)
		assert.True(t, config.IsPublic)
	})
}

func TestIsValidFunctionName(t *testing.T) {
	t.Run("valid names", func(t *testing.T) {
		validNames := []string{
			"hello",
			"hello_world",
			"hello-world",
			"HelloWorld",
			"_private",
			"fn123",
			"a",
		}
		for _, name := range validNames {
			t.Run(name, func(t *testing.T) {
				assert.True(t, isValidFunctionName(name), "name %q should be valid", name)
			})
		}
	})

	t.Run("invalid names", func(t *testing.T) {
		invalidNames := []string{
			"",
			"123abc",      // Starts with number
			"hello.world", // Has dot
			"hello world", // Has space
			"hello@world", // Has special char
		}
		for _, name := range invalidNames {
			t.Run(name, func(t *testing.T) {
				assert.False(t, isValidFunctionName(name), "name %q should be invalid", name)
			})
		}
	})

	t.Run("rejects names over 63 characters", func(t *testing.T) {
		longName := "a" + string(make([]byte, 63)) // 64 characters
		assert.False(t, isValidFunctionName(longName))
	})
}

// ============================================================================
// SYNC JOB TESTS
// ============================================================================

func TestParseJobAnnotations(t *testing.T) {
	t.Run("parses schedule annotation", func(t *testing.T) {
		code := `
// @fluxbase:schedule "0 */5 * * *"
// @fluxbase:description Runs every 5 minutes
export default async function cleanup() {}`
		config := parseJobAnnotations(code)

		assert.Equal(t, "0 */5 * * *", config.Schedule)
		assert.Equal(t, "Runs every 5 minutes", config.Description)
	})

	t.Run("parses max-retries", func(t *testing.T) {
		code := `
// @fluxbase:schedule "0 0 * * *"
// @fluxbase:max-retries 5
export default async function cleanup() {}`
		config := parseJobAnnotations(code)

		assert.Equal(t, 5, config.MaxRetries)
	})

	t.Run("parses require-role with single role", func(t *testing.T) {
		code := `
// @fluxbase:schedule "0 0 * * *"
// @fluxbase:require-role admin
export default async function adminJob() {}`
		config := parseJobAnnotations(code)

		assert.Equal(t, []string{"admin"}, config.RequireRoles)
	})

	t.Run("parses require-role with multiple roles", func(t *testing.T) {
		code := `
// @fluxbase:schedule "0 0 * * *"
// @fluxbase:require-role admin, editor, moderator
export default async function adminJob() {}`
		config := parseJobAnnotations(code)

		assert.Equal(t, []string{"admin", "editor", "moderator"}, config.RequireRoles)
	})

	t.Run("uses job defaults", func(t *testing.T) {
		code := `
// @fluxbase:schedule "0 0 * * *"
export default async function job() {}`
		config := parseJobAnnotations(code)

		assert.Equal(t, 300, config.Timeout) // 5 minutes for jobs
		assert.Equal(t, 256, config.Memory)
		assert.Equal(t, 3, config.MaxRetries)
		assert.True(t, config.AllowNet)
		assert.True(t, config.AllowEnv)
	})
}

func TestIsValidCronExpression(t *testing.T) {
	t.Run("valid expressions", func(t *testing.T) {
		valid := []string{
			"* * * * *",
			"0 */5 * * *",
			"0 0 * * *",
			"30 4 1,15 * 5",
			"0 0 1 * *",
		}
		for _, expr := range valid {
			t.Run(expr, func(t *testing.T) {
				assert.True(t, isValidCronExpression(expr), "expression %q should be valid", expr)
			})
		}
	})

	t.Run("invalid expressions", func(t *testing.T) {
		invalid := []string{
			"",
			"* * *",         // Too few fields
			"* * * * * * *", // Too many fields
			"a b c d e",     // Invalid characters
		}
		for _, expr := range invalid {
			t.Run(expr, func(t *testing.T) {
				assert.False(t, isValidCronExpression(expr), "expression %q should be invalid", expr)
			})
		}
	})
}

// ============================================================================
// SYNC RPC TESTS
// ============================================================================

func TestParseRPCAnnotations(t *testing.T) {
	t.Run("parses SQL comment annotations", func(t *testing.T) {
		sqlCode := `
-- @fluxbase:description Get user profile with stats
-- @fluxbase:public
-- @fluxbase:timeout 10
SELECT u.*, s.total_posts
FROM users u
WHERE u.id = $1;`
		config := parseRPCAnnotations(sqlCode)

		assert.Equal(t, "Get user profile with stats", config.Description)
		assert.True(t, config.IsPublic)
		assert.Equal(t, 10, config.Timeout)
	})

	t.Run("parses allowed-tables", func(t *testing.T) {
		sqlCode := `
-- @fluxbase:allowed-tables users, orders, products
SELECT * FROM users WHERE id = $1;`
		config := parseRPCAnnotations(sqlCode)

		assert.Equal(t, []string{"users", "orders", "products"}, config.AllowedTables)
	})

	t.Run("parses allowed-schemas", func(t *testing.T) {
		sqlCode := `
-- @fluxbase:allowed-schemas public, reporting
SELECT * FROM users;`
		config := parseRPCAnnotations(sqlCode)

		assert.Equal(t, []string{"public", "reporting"}, config.AllowedSchemas)
	})

	t.Run("parses require-role with single role", func(t *testing.T) {
		sqlCode := `
-- @fluxbase:require-role authenticated
SELECT * FROM users WHERE id = $1;`
		config := parseRPCAnnotations(sqlCode)

		assert.Equal(t, []string{"authenticated"}, config.RequireRoles)
	})

	t.Run("parses require-role with multiple roles", func(t *testing.T) {
		sqlCode := `
-- @fluxbase:require-role admin, editor, moderator
SELECT * FROM users WHERE id = $1;`
		config := parseRPCAnnotations(sqlCode)

		assert.Equal(t, []string{"admin", "editor", "moderator"}, config.RequireRoles)
	})

	t.Run("parses schedule for scheduled procedures", func(t *testing.T) {
		sqlCode := `
-- @fluxbase:schedule "0 0 * * *"
DELETE FROM temp_data WHERE created_at < NOW() - INTERVAL '7 days';`
		config := parseRPCAnnotations(sqlCode)

		assert.NotNil(t, config.Schedule)
		assert.Equal(t, "0 0 * * *", *config.Schedule)
	})

	t.Run("uses defaults", func(t *testing.T) {
		sqlCode := `SELECT 1;`
		config := parseRPCAnnotations(sqlCode)

		assert.Equal(t, 30, config.Timeout)
		assert.Equal(t, []string{"public"}, config.AllowedSchemas)
		assert.False(t, config.IsPublic)
	})
}

func TestParseCommaSeparatedList(t *testing.T) {
	t.Run("parses comma-separated values", func(t *testing.T) {
		result := parseCommaSeparatedList("a, b, c")
		assert.Equal(t, []string{"a", "b", "c"}, result)
	})

	t.Run("trims whitespace", func(t *testing.T) {
		result := parseCommaSeparatedList("  a  ,  b  ,  c  ")
		assert.Equal(t, []string{"a", "b", "c"}, result)
	})

	t.Run("returns nil for empty string", func(t *testing.T) {
		result := parseCommaSeparatedList("")
		assert.Nil(t, result)
	})

	t.Run("handles single value", func(t *testing.T) {
		result := parseCommaSeparatedList("single")
		assert.Equal(t, []string{"single"}, result)
	})
}

// ============================================================================
// SYNC MIGRATION TESTS
// ============================================================================

func TestIsValidMigrationName(t *testing.T) {
	t.Run("valid names", func(t *testing.T) {
		valid := []string{
			"add_users_table",
			"create_index",
			"v1_migration",
			"AddUsersTable",
			"_temp",
		}
		for _, name := range valid {
			t.Run(name, func(t *testing.T) {
				assert.True(t, isValidMigrationName(name), "name %q should be valid", name)
			})
		}
	})

	t.Run("invalid names", func(t *testing.T) {
		invalid := []string{
			"",
			"123abc",    // Starts with number
			"has space", // Has space
			"has.dot",   // Has dot
		}
		for _, name := range invalid {
			t.Run(name, func(t *testing.T) {
				assert.False(t, isValidMigrationName(name), "name %q should be invalid", name)
			})
		}
	})

	t.Run("rejects names over 100 characters", func(t *testing.T) {
		longName := "a" + string(make([]byte, 100)) // 101 characters
		assert.False(t, isValidMigrationName(longName))
	})
}

func TestValidateMigrationSQL(t *testing.T) {
	t.Run("allows valid DDL", func(t *testing.T) {
		validSQL := []string{
			"CREATE TABLE users (id UUID PRIMARY KEY);",
			"ALTER TABLE users ADD COLUMN name TEXT;",
			"DROP TABLE temp_data;",
			"CREATE INDEX idx_users_email ON users(email);",
		}
		for _, sql := range validSQL {
			t.Run(sql[:20], func(t *testing.T) {
				err := validateMigrationSQL(sql)
				assert.NoError(t, err)
			})
		}
	})

	t.Run("blocks dangerous operations on system schemas", func(t *testing.T) {
		dangerous := []string{
			"DROP SCHEMA auth CASCADE;",
			"DROP SCHEMA storage;",
			"ALTER SCHEMA jobs RENAME TO old_jobs;",
			"DROP TABLE auth.users;",
			"TRUNCATE storage.objects;",
		}
		for _, sql := range dangerous {
			t.Run(sql[:20], func(t *testing.T) {
				err := validateMigrationSQL(sql)
				assert.Error(t, err)
			})
		}
	})

	t.Run("blocks database operations", func(t *testing.T) {
		err := validateMigrationSQL("DROP DATABASE production;")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database")

		err = validateMigrationSQL("CREATE DATABASE test;")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database")
	})

	t.Run("rejects empty SQL", func(t *testing.T) {
		err := validateMigrationSQL("")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "empty")
	})
}

// ============================================================================
// SYNC CHATBOT TESTS
// ============================================================================

func TestParseChatbotAnnotations(t *testing.T) {
	t.Run("parses basic annotations", func(t *testing.T) {
		code := `
// @fluxbase:description Customer support assistant
// @fluxbase:public
// @fluxbase:allow-unauthenticated
You are a helpful customer support agent...`
		config := parseChatbotAnnotations(code)

		assert.Equal(t, "Customer support assistant", config.Description)
		assert.True(t, config.IsPublic)
		assert.True(t, config.AllowUnauthenticated)
	})

	t.Run("parses allowed-tables", func(t *testing.T) {
		code := `
// @fluxbase:allowed-tables users, orders, products
You are a helpful assistant...`
		config := parseChatbotAnnotations(code)

		assert.Equal(t, []string{"users", "orders", "products"}, config.AllowedTables)
	})

	t.Run("parses allowed-operations", func(t *testing.T) {
		code := `
// @fluxbase:allowed-operations SELECT, INSERT
You are a helpful assistant...`
		config := parseChatbotAnnotations(code)

		assert.Equal(t, []string{"SELECT", "INSERT"}, config.AllowedOperations)
	})

	t.Run("parses model and tokens", func(t *testing.T) {
		code := `
// @fluxbase:model gpt-4
// @fluxbase:max-tokens 8192
// @fluxbase:temperature 0.5
You are a helpful assistant...`
		config := parseChatbotAnnotations(code)

		assert.Equal(t, "gpt-4", config.Model)
		assert.Equal(t, 8192, config.MaxTokens)
		assert.Equal(t, 0.5, config.Temperature)
	})

	t.Run("parses rate limiting", func(t *testing.T) {
		code := `
// @fluxbase:rate-limit 30/min
// @fluxbase:daily-limit 1000
// @fluxbase:daily-token-budget 500000
You are a helpful assistant...`
		config := parseChatbotAnnotations(code)

		assert.Equal(t, 30, config.RateLimitPerMinute)
		assert.Equal(t, 1000, config.DailyRequestLimit)
		assert.Equal(t, 500000, config.DailyTokenBudget)
	})

	t.Run("parses conversation settings", func(t *testing.T) {
		code := `
// @fluxbase:persist-conversations
// @fluxbase:conversation-ttl 48
// @fluxbase:max-turns 100
You are a helpful assistant...`
		config := parseChatbotAnnotations(code)

		assert.True(t, config.PersistConversations)
		assert.Equal(t, 48, config.ConversationTTLHours)
		assert.Equal(t, 100, config.MaxTurns)
	})

	t.Run("parses response language", func(t *testing.T) {
		code := `
// @fluxbase:response-language es
You are a helpful assistant...`
		config := parseChatbotAnnotations(code)

		assert.Equal(t, "es", config.ResponseLanguage)
	})

	t.Run("uses defaults", func(t *testing.T) {
		code := `You are a helpful assistant...`
		config := parseChatbotAnnotations(code)

		assert.Equal(t, 4096, config.MaxTokens)
		assert.Equal(t, 0.7, config.Temperature)
		assert.Equal(t, 20, config.RateLimitPerMinute)
		assert.Equal(t, 500, config.DailyRequestLimit)
		assert.Equal(t, 24, config.ConversationTTLHours)
		assert.Equal(t, "auto", config.ResponseLanguage)
		assert.Equal(t, []string{"SELECT"}, config.AllowedOperations)
		assert.Equal(t, []string{"public"}, config.AllowedSchemas)
	})

	t.Run("parses mcp-tools annotation", func(t *testing.T) {
		code := `
// @fluxbase:mcp-tools query_table, insert_record, invoke_function
You are a helpful assistant...`
		config := parseChatbotAnnotations(code)

		assert.Equal(t, []string{"query_table", "insert_record", "invoke_function"}, config.MCPTools)
	})

	t.Run("parses use-mcp-schema annotation without value", func(t *testing.T) {
		code := `
// @fluxbase:use-mcp-schema
You are a helpful assistant...`
		config := parseChatbotAnnotations(code)

		assert.True(t, config.UseMCPSchema)
	})

	t.Run("parses use-mcp-schema annotation with true", func(t *testing.T) {
		code := `
// @fluxbase:use-mcp-schema true
You are a helpful assistant...`
		config := parseChatbotAnnotations(code)

		assert.True(t, config.UseMCPSchema)
	})

	t.Run("parses use-mcp-schema annotation with false", func(t *testing.T) {
		code := `
// @fluxbase:use-mcp-schema false
You are a helpful assistant...`
		config := parseChatbotAnnotations(code)

		assert.False(t, config.UseMCPSchema)
	})

	t.Run("parses complete MCP chatbot config", func(t *testing.T) {
		code := `
// @fluxbase:description Order management assistant
// @fluxbase:allowed-tables orders,order_items,products,analytics.order_metrics
// @fluxbase:mcp-tools query_table,insert_record,invoke_function
// @fluxbase:use-mcp-schema
// @fluxbase:public
// @fluxbase:persist-conversations
// @fluxbase:rate-limit 30/min
You are an order management assistant.`
		config := parseChatbotAnnotations(code)

		assert.Equal(t, "Order management assistant", config.Description)
		assert.Equal(t, []string{"orders", "order_items", "products", "analytics.order_metrics"}, config.AllowedTables)
		assert.Equal(t, []string{"query_table", "insert_record", "invoke_function"}, config.MCPTools)
		assert.True(t, config.UseMCPSchema)
		assert.True(t, config.IsPublic)
		assert.True(t, config.PersistConversations)
		assert.Equal(t, 30, config.RateLimitPerMinute)
	})

	t.Run("mcp-tools defaults to empty slice", func(t *testing.T) {
		code := `You are a helpful assistant...`
		config := parseChatbotAnnotations(code)

		assert.Equal(t, []string{}, config.MCPTools)
		assert.False(t, config.UseMCPSchema)
	})

	t.Run("parses require-role with single role", func(t *testing.T) {
		code := `
// @fluxbase:require-role admin
You are an admin assistant...`
		config := parseChatbotAnnotations(code)

		assert.Equal(t, []string{"admin"}, config.RequireRoles)
	})

	t.Run("parses require-role with multiple roles", func(t *testing.T) {
		code := `
// @fluxbase:require-role admin, editor, moderator
You are a privileged assistant...`
		config := parseChatbotAnnotations(code)

		assert.Equal(t, []string{"admin", "editor", "moderator"}, config.RequireRoles)
	})
}
