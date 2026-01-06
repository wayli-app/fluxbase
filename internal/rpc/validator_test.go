package rpc

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewValidator(t *testing.T) {
	v := NewValidator()
	require.NotNil(t, v)
}

func TestValidator_ValidateInput(t *testing.T) {
	v := NewValidator()

	t.Run("accepts any input when schema is empty", func(t *testing.T) {
		err := v.ValidateInput(map[string]interface{}{
			"foo": "bar",
			"num": 123,
		}, nil)
		assert.NoError(t, err)
	})

	t.Run("validates required fields", func(t *testing.T) {
		schema := json.RawMessage(`{"user_id": "uuid", "name": "string"}`)

		err := v.ValidateInput(map[string]interface{}{
			"user_id": "550e8400-e29b-41d4-a716-446655440000",
			"name":    "John",
		}, schema)
		assert.NoError(t, err)
	})

	t.Run("returns error for missing required field", func(t *testing.T) {
		schema := json.RawMessage(`{"user_id": "uuid", "name": "string"}`)

		err := v.ValidateInput(map[string]interface{}{
			"user_id": "550e8400-e29b-41d4-a716-446655440000",
		}, schema)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing required parameter: name")
	})

	t.Run("allows missing optional fields", func(t *testing.T) {
		schema := json.RawMessage(`{"user_id": "uuid", "name?": "string"}`)

		err := v.ValidateInput(map[string]interface{}{
			"user_id": "550e8400-e29b-41d4-a716-446655440000",
		}, schema)
		assert.NoError(t, err)
	})

	t.Run("validates type for string", func(t *testing.T) {
		schema := json.RawMessage(`{"name": "string"}`)

		// Valid string
		err := v.ValidateInput(map[string]interface{}{"name": "John"}, schema)
		assert.NoError(t, err)

		// Invalid - number instead of string
		err = v.ValidateInput(map[string]interface{}{"name": 123}, schema)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must be a string")
	})

	t.Run("validates type for number", func(t *testing.T) {
		schema := json.RawMessage(`{"count": "number"}`)

		// Valid - int
		err := v.ValidateInput(map[string]interface{}{"count": 123}, schema)
		assert.NoError(t, err)

		// Valid - float
		err = v.ValidateInput(map[string]interface{}{"count": 123.45}, schema)
		assert.NoError(t, err)

		// Valid - json.Number
		err = v.ValidateInput(map[string]interface{}{"count": json.Number("123")}, schema)
		assert.NoError(t, err)

		// Invalid - string
		err = v.ValidateInput(map[string]interface{}{"count": "123"}, schema)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must be a number")
	})

	t.Run("validates type for boolean", func(t *testing.T) {
		schema := json.RawMessage(`{"active": "boolean"}`)

		// Valid
		err := v.ValidateInput(map[string]interface{}{"active": true}, schema)
		assert.NoError(t, err)

		// Invalid
		err = v.ValidateInput(map[string]interface{}{"active": "true"}, schema)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must be a boolean")
	})

	t.Run("validates type for array", func(t *testing.T) {
		schema := json.RawMessage(`{"items": "array"}`)

		// Valid
		err := v.ValidateInput(map[string]interface{}{"items": []interface{}{1, 2, 3}}, schema)
		assert.NoError(t, err)

		// Invalid
		err = v.ValidateInput(map[string]interface{}{"items": "not an array"}, schema)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must be an array")
	})

	t.Run("validates type for object/json", func(t *testing.T) {
		schema := json.RawMessage(`{"metadata": "json"}`)

		// Valid
		err := v.ValidateInput(map[string]interface{}{"metadata": map[string]interface{}{"key": "value"}}, schema)
		assert.NoError(t, err)

		// Invalid
		err = v.ValidateInput(map[string]interface{}{"metadata": "not an object"}, schema)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must be an object")
	})

	t.Run("allows null values", func(t *testing.T) {
		schema := json.RawMessage(`{"name": "string"}`)

		err := v.ValidateInput(map[string]interface{}{"name": nil}, schema)
		assert.NoError(t, err)
	})

	t.Run("returns error for invalid schema JSON", func(t *testing.T) {
		schema := json.RawMessage(`invalid json`)

		err := v.ValidateInput(map[string]interface{}{"name": "John"}, schema)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid input schema")
	})
}

func TestValidator_ValidateSQL(t *testing.T) {
	v := NewValidator()

	t.Run("validates simple SELECT", func(t *testing.T) {
		result := v.ValidateSQL("SELECT * FROM users", nil, nil)
		assert.True(t, result.Valid)
		assert.Contains(t, result.OperationsUsed, "SELECT")
		assert.Contains(t, result.TablesAccessed, "users")
	})

	t.Run("validates INSERT", func(t *testing.T) {
		result := v.ValidateSQL("INSERT INTO users (name) VALUES ('John')", nil, nil)
		assert.True(t, result.Valid)
		assert.Contains(t, result.OperationsUsed, "INSERT")
	})

	t.Run("validates UPDATE", func(t *testing.T) {
		result := v.ValidateSQL("UPDATE users SET name = 'Jane' WHERE id = 1", nil, nil)
		assert.True(t, result.Valid)
		assert.Contains(t, result.OperationsUsed, "UPDATE")
	})

	t.Run("validates DELETE", func(t *testing.T) {
		result := v.ValidateSQL("DELETE FROM users WHERE id = 1", nil, nil)
		assert.True(t, result.Valid)
		assert.Contains(t, result.OperationsUsed, "DELETE")
	})

	t.Run("blocks pg_catalog access", func(t *testing.T) {
		result := v.ValidateSQL("SELECT * FROM pg_catalog.pg_tables", nil, nil)
		assert.False(t, result.Valid)
		assert.Contains(t, result.Errors[0], "pg_catalog")
	})

	t.Run("blocks information_schema access", func(t *testing.T) {
		result := v.ValidateSQL("SELECT * FROM information_schema.tables", nil, nil)
		assert.False(t, result.Valid)
		assert.Contains(t, result.Errors[0], "information_schema")
	})

	t.Run("allows SQL comments", func(t *testing.T) {
		result := v.ValidateSQL("SELECT * FROM users -- comment", nil, nil)
		assert.True(t, result.Valid)
	})

	t.Run("allows block comments", func(t *testing.T) {
		result := v.ValidateSQL("SELECT * FROM users /* comment */", nil, nil)
		assert.True(t, result.Valid)
	})

	t.Run("blocks multiple statements", func(t *testing.T) {
		result := v.ValidateSQL("SELECT 1; SELECT 2;", nil, nil)
		assert.False(t, result.Valid)
		assert.Contains(t, result.Errors[0], "Multiple SQL statements")
	})

	t.Run("blocks empty statement", func(t *testing.T) {
		result := v.ValidateSQL("", nil, nil)
		assert.False(t, result.Valid)
	})

	t.Run("enforces allowed tables", func(t *testing.T) {
		result := v.ValidateSQL("SELECT * FROM users", []string{"orders"}, nil)
		assert.False(t, result.Valid)
		assert.Contains(t, result.Errors[0], "Table not allowed: users")
	})

	t.Run("allows specified tables", func(t *testing.T) {
		result := v.ValidateSQL("SELECT * FROM users", []string{"users", "orders"}, nil)
		assert.True(t, result.Valid)
	})

	t.Run("enforces allowed schemas", func(t *testing.T) {
		result := v.ValidateSQL("SELECT * FROM private.secrets", nil, []string{"public"})
		assert.False(t, result.Valid)
		assert.Contains(t, result.Errors[0], "Schema not allowed: private")
	})

	t.Run("allows specified schemas", func(t *testing.T) {
		result := v.ValidateSQL("SELECT * FROM public.users", nil, []string{"public"})
		assert.True(t, result.Valid)
	})

	t.Run("handles named parameters", func(t *testing.T) {
		result := v.ValidateSQL("SELECT * FROM users WHERE id = $user_id AND name = $name", nil, nil)
		assert.True(t, result.Valid)
	})

	t.Run("handles JOIN queries", func(t *testing.T) {
		result := v.ValidateSQL(`
			SELECT u.name, o.total
			FROM users u
			JOIN orders o ON u.id = o.user_id
		`, nil, nil)
		assert.True(t, result.Valid)
		assert.Contains(t, result.TablesAccessed, "users")
		assert.Contains(t, result.TablesAccessed, "orders")
	})

	t.Run("handles subqueries", func(t *testing.T) {
		result := v.ValidateSQL(`
			SELECT * FROM users
			WHERE id IN (SELECT user_id FROM orders WHERE total > 100)
		`, nil, nil)
		assert.True(t, result.Valid)
	})

	t.Run("returns parse error for invalid SQL", func(t *testing.T) {
		result := v.ValidateSQL("SELEKT * FORM users", nil, nil)
		assert.False(t, result.Valid)
		assert.Contains(t, result.Errors[0], "Failed to parse SQL")
	})
}

func TestValidator_ValidateAccess(t *testing.T) {
	v := NewValidator()

	t.Run("public procedure allows unauthenticated access", func(t *testing.T) {
		proc := &Procedure{IsPublic: true}
		err := v.ValidateAccess(proc, "", false)
		assert.NoError(t, err)
	})

	t.Run("non-public procedure requires authentication", func(t *testing.T) {
		proc := &Procedure{IsPublic: false}
		err := v.ValidateAccess(proc, "", false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "requires authentication")
	})

	t.Run("authenticated user can access non-public procedure", func(t *testing.T) {
		proc := &Procedure{IsPublic: false}
		err := v.ValidateAccess(proc, "user", true)
		assert.NoError(t, err)
	})

	t.Run("require-role anon allows anyone", func(t *testing.T) {
		proc := &Procedure{IsPublic: true, RequireRoles: []string{"anon"}}
		err := v.ValidateAccess(proc, "", false)
		assert.NoError(t, err)
	})

	t.Run("require-role authenticated requires login", func(t *testing.T) {
		proc := &Procedure{IsPublic: true, RequireRoles: []string{"authenticated"}}

		// Not authenticated
		err := v.ValidateAccess(proc, "", false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "requires one of roles")

		// Authenticated
		err = v.ValidateAccess(proc, "user", true)
		assert.NoError(t, err)
	})

	t.Run("specific role is enforced", func(t *testing.T) {
		proc := &Procedure{IsPublic: true, RequireRoles: []string{"admin"}}

		// Wrong role
		err := v.ValidateAccess(proc, "user", true)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "requires one of roles")

		// Correct role
		err = v.ValidateAccess(proc, "admin", true)
		assert.NoError(t, err)
	})

	t.Run("multiple roles allows any matching role", func(t *testing.T) {
		proc := &Procedure{IsPublic: true, RequireRoles: []string{"admin", "editor", "moderator"}}

		// User with one of the allowed roles
		err := v.ValidateAccess(proc, "editor", true)
		assert.NoError(t, err)

		// User with different allowed role
		err = v.ValidateAccess(proc, "moderator", true)
		assert.NoError(t, err)

		// User with non-allowed role
		err = v.ValidateAccess(proc, "viewer", true)
		require.Error(t, err)
	})

	t.Run("service_role bypasses role check", func(t *testing.T) {
		proc := &Procedure{IsPublic: true, RequireRoles: []string{"admin"}}

		err := v.ValidateAccess(proc, "service_role", true)
		assert.NoError(t, err)
	})

	t.Run("dashboard_admin bypasses role check", func(t *testing.T) {
		proc := &Procedure{IsPublic: true, RequireRoles: []string{"admin"}}

		err := v.ValidateAccess(proc, "dashboard_admin", true)
		assert.NoError(t, err)
	})

	t.Run("empty require_roles does not restrict", func(t *testing.T) {
		proc := &Procedure{IsPublic: false, RequireRoles: []string{}}

		err := v.ValidateAccess(proc, "user", true)
		assert.NoError(t, err)
	})

	t.Run("nil require_roles does not restrict", func(t *testing.T) {
		proc := &Procedure{IsPublic: false, RequireRoles: nil}

		err := v.ValidateAccess(proc, "user", true)
		assert.NoError(t, err)
	})
}

func TestValidator_PreprocessNamedParams(t *testing.T) {
	v := NewValidator()

	t.Run("replaces named params with positional", func(t *testing.T) {
		sql := "SELECT * FROM users WHERE id = $user_id AND name = $name"
		processed, params := v.preprocessNamedParams(sql)

		assert.Equal(t, "SELECT * FROM users WHERE id = $1 AND name = $2", processed)
		assert.Equal(t, []string{"user_id", "name"}, params)
	})

	t.Run("handles same param multiple times", func(t *testing.T) {
		sql := "SELECT * FROM users WHERE id = $id OR parent_id = $id"
		processed, params := v.preprocessNamedParams(sql)

		// Each occurrence gets a unique placeholder
		assert.Equal(t, "SELECT * FROM users WHERE id = $1 OR parent_id = $2", processed)
		assert.Equal(t, []string{"id", "id"}, params)
	})

	t.Run("handles underscore in param names", func(t *testing.T) {
		sql := "SELECT * FROM users WHERE user_id = $user_id"
		processed, params := v.preprocessNamedParams(sql)

		assert.Equal(t, "SELECT * FROM users WHERE user_id = $1", processed)
		assert.Equal(t, []string{"user_id"}, params)
	})

	t.Run("no params returns original", func(t *testing.T) {
		sql := "SELECT * FROM users"
		processed, params := v.preprocessNamedParams(sql)

		assert.Equal(t, "SELECT * FROM users", processed)
		assert.Empty(t, params)
	})
}

func TestValidator_GetOperationType(t *testing.T) {
	v := NewValidator()

	testCases := []struct {
		sql      string
		expected string
	}{
		{"SELECT * FROM users", "SELECT"},
		{"INSERT INTO users (name) VALUES ('John')", "INSERT"},
		{"UPDATE users SET name = 'Jane'", "UPDATE"},
		{"DELETE FROM users WHERE id = 1", "DELETE"},
	}

	for _, tc := range testCases {
		t.Run(tc.expected, func(t *testing.T) {
			result := v.ValidateSQL(tc.sql, nil, nil)
			assert.Contains(t, result.OperationsUsed, tc.expected)
		})
	}
}

func TestValidator_ExtractTables(t *testing.T) {
	v := NewValidator()

	t.Run("extracts single table", func(t *testing.T) {
		result := v.ValidateSQL("SELECT * FROM users", nil, nil)
		assert.Equal(t, []string{"users"}, result.TablesAccessed)
	})

	t.Run("extracts multiple tables from JOIN", func(t *testing.T) {
		result := v.ValidateSQL("SELECT * FROM users u JOIN orders o ON u.id = o.user_id", nil, nil)
		assert.Contains(t, result.TablesAccessed, "users")
		assert.Contains(t, result.TablesAccessed, "orders")
	})

	t.Run("extracts schema-qualified table", func(t *testing.T) {
		result := v.ValidateSQL("SELECT * FROM public.users", nil, nil)
		assert.Contains(t, result.TablesAccessed, "public.users")
	})

	t.Run("extracts tables from UPDATE with FROM clause", func(t *testing.T) {
		result := v.ValidateSQL("UPDATE users SET total = o.total FROM orders o WHERE users.id = o.user_id", nil, nil)
		assert.Contains(t, result.TablesAccessed, "users")
		assert.Contains(t, result.TablesAccessed, "orders")
	})

	t.Run("extracts tables from INSERT with SELECT", func(t *testing.T) {
		result := v.ValidateSQL("INSERT INTO archive SELECT * FROM users", nil, nil)
		assert.Contains(t, result.TablesAccessed, "archive")
		assert.Contains(t, result.TablesAccessed, "users")
	})
}

func TestValidationResult(t *testing.T) {
	t.Run("default result is valid", func(t *testing.T) {
		result := &ValidationResult{
			Valid:          true,
			TablesAccessed: []string{},
			OperationsUsed: []string{},
		}

		assert.True(t, result.Valid)
		assert.Empty(t, result.Errors)
		assert.Empty(t, result.Warnings)
	})

	t.Run("invalid result has errors", func(t *testing.T) {
		result := &ValidationResult{
			Valid:  false,
			Errors: []string{"error1", "error2"},
		}

		assert.False(t, result.Valid)
		assert.Len(t, result.Errors, 2)
	})
}

func TestValidator_BlockedPatterns(t *testing.T) {
	v := NewValidator()

	blockedPatterns := []struct {
		sql     string
		pattern string
	}{
		{"SELECT * FROM pg_catalog.pg_tables", "pg_catalog"},
		{"SELECT * FROM information_schema.columns", "information_schema"},
		{"SELECT * FROM pg_temp.my_table", "pg_temp"},
		{"SELECT * FROM pg_toast.my_table", "pg_toast"},
		{"EXEC xp_cmdshell 'dir'", "xp_"},
		{"exec('SELECT 1')", "exec("},
		{"execute('SELECT 1')", "execute("},
	}

	for _, tc := range blockedPatterns {
		t.Run(tc.pattern, func(t *testing.T) {
			result := v.ValidateSQL(tc.sql, nil, nil)
			assert.False(t, result.Valid, "SQL should be blocked: %s", tc.sql)
			found := false
			for _, err := range result.Errors {
				if assert.Contains(t, err, tc.pattern) {
					found = true
					break
				}
			}
			assert.True(t, found || len(result.Errors) > 0, "Should contain error for pattern: %s", tc.pattern)
		})
	}
}

func TestValidator_CaseInsensitiveBlocking(t *testing.T) {
	v := NewValidator()

	t.Run("blocks patterns case insensitively", func(t *testing.T) {
		sqls := []string{
			"SELECT * FROM PG_CATALOG.pg_tables",
			"SELECT * FROM Pg_Catalog.pg_tables",
			"SELECT * FROM INFORMATION_SCHEMA.tables",
			"SELECT * FROM Information_Schema.tables",
		}

		for _, sql := range sqls {
			result := v.ValidateSQL(sql, nil, nil)
			assert.False(t, result.Valid, "Should block: %s", sql)
		}
	})
}
