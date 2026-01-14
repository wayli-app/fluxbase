package ai

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// NewSQLValidator Tests
// =============================================================================

func TestNewSQLValidator(t *testing.T) {
	t.Run("creates validator with allowed schemas", func(t *testing.T) {
		v := NewSQLValidator(
			[]string{"public", "api"},
			nil,
			[]string{"SELECT"},
		)

		require.NotNil(t, v)
		assert.True(t, v.allowedSchemas["public"])
		assert.True(t, v.allowedSchemas["api"])
		assert.False(t, v.allowedSchemas["private"])
	})

	t.Run("creates validator with allowed tables", func(t *testing.T) {
		v := NewSQLValidator(
			nil,
			[]string{"users", "orders"},
			[]string{"SELECT"},
		)

		require.NotNil(t, v)
		assert.True(t, v.allowedTables["users"])
		assert.True(t, v.allowedTables["orders"])
		assert.False(t, v.allowedTables["secrets"])
	})

	t.Run("creates validator with allowed operations", func(t *testing.T) {
		v := NewSQLValidator(
			nil,
			nil,
			[]string{"SELECT", "INSERT"},
		)

		require.NotNil(t, v)
		assert.True(t, v.allowedOperations["SELECT"])
		assert.True(t, v.allowedOperations["INSERT"])
		assert.False(t, v.allowedOperations["DELETE"])
	})

	t.Run("normalizes schema names to lowercase", func(t *testing.T) {
		v := NewSQLValidator(
			[]string{"Public", "API", "CUSTOM"},
			nil,
			nil,
		)

		assert.True(t, v.allowedSchemas["public"])
		assert.True(t, v.allowedSchemas["api"])
		assert.True(t, v.allowedSchemas["custom"])
	})

	t.Run("normalizes table names to lowercase", func(t *testing.T) {
		v := NewSQLValidator(
			nil,
			[]string{"Users", "ORDERS"},
			nil,
		)

		assert.True(t, v.allowedTables["users"])
		assert.True(t, v.allowedTables["orders"])
	})

	t.Run("normalizes operations to uppercase", func(t *testing.T) {
		v := NewSQLValidator(
			nil,
			nil,
			[]string{"select", "Insert", "DELETE"},
		)

		assert.True(t, v.allowedOperations["SELECT"])
		assert.True(t, v.allowedOperations["INSERT"])
		assert.True(t, v.allowedOperations["DELETE"])
	})

	t.Run("includes blocked patterns", func(t *testing.T) {
		v := NewSQLValidator(nil, nil, nil)

		assert.NotEmpty(t, v.blockedPatterns)
		assert.Contains(t, v.blockedPatterns, "pg_catalog")
		assert.Contains(t, v.blockedPatterns, "information_schema")
		assert.Contains(t, v.blockedPatterns, "--")
		assert.Contains(t, v.blockedPatterns, "/*")
	})

	t.Run("handles empty inputs", func(t *testing.T) {
		v := NewSQLValidator([]string{}, []string{}, []string{})

		require.NotNil(t, v)
		assert.Empty(t, v.allowedSchemas)
		assert.Empty(t, v.allowedTables)
		assert.Empty(t, v.allowedOperations)
	})
}

// =============================================================================
// Validate Tests
// =============================================================================

func TestSQLValidator_Validate(t *testing.T) {
	t.Run("validates simple SELECT", func(t *testing.T) {
		v := NewSQLValidator(
			[]string{"public"},
			[]string{"users"},
			[]string{"SELECT"},
		)

		result := v.Validate("SELECT * FROM users")

		assert.True(t, result.Valid)
		assert.Empty(t, result.Errors)
		assert.Contains(t, result.OperationsUsed, "SELECT")
		assert.Contains(t, result.TablesAccessed, "users")
	})

	t.Run("rejects blocked patterns - pg_catalog", func(t *testing.T) {
		v := NewSQLValidator(nil, nil, []string{"SELECT"})

		result := v.Validate("SELECT * FROM pg_catalog.pg_tables")

		assert.False(t, result.Valid)
		assert.NotEmpty(t, result.Errors)
	})

	t.Run("rejects blocked patterns - information_schema", func(t *testing.T) {
		v := NewSQLValidator(nil, nil, []string{"SELECT"})

		result := v.Validate("SELECT * FROM information_schema.tables")

		assert.False(t, result.Valid)
		assert.NotEmpty(t, result.Errors)
	})

	t.Run("rejects SQL comment injection", func(t *testing.T) {
		v := NewSQLValidator(nil, nil, []string{"SELECT"})

		result := v.Validate("SELECT * FROM users WHERE id = 1 -- admin")

		assert.False(t, result.Valid)
		assert.NotEmpty(t, result.Errors)
	})

	t.Run("rejects block comment injection", func(t *testing.T) {
		v := NewSQLValidator(nil, nil, []string{"SELECT"})

		result := v.Validate("SELECT * FROM users /* WHERE id = 1 */")

		assert.False(t, result.Valid)
		assert.NotEmpty(t, result.Errors)
	})

	t.Run("rejects multiple statements", func(t *testing.T) {
		v := NewSQLValidator(nil, nil, []string{"SELECT", "DELETE"})

		result := v.Validate("SELECT * FROM users; DELETE FROM users")

		assert.False(t, result.Valid)
		assert.Contains(t, result.Errors[0], "Multiple SQL statements not allowed")
	})

	t.Run("rejects empty statement", func(t *testing.T) {
		v := NewSQLValidator(nil, nil, []string{"SELECT"})

		result := v.Validate("")

		assert.False(t, result.Valid)
	})

	t.Run("rejects disallowed operation", func(t *testing.T) {
		v := NewSQLValidator(nil, nil, []string{"SELECT"})

		result := v.Validate("DELETE FROM users")

		assert.False(t, result.Valid)
		assert.NotEmpty(t, result.Errors)
	})

	t.Run("rejects disallowed schema", func(t *testing.T) {
		v := NewSQLValidator(
			[]string{"public"},
			nil,
			[]string{"SELECT"},
		)

		result := v.Validate("SELECT * FROM private.secrets")

		assert.False(t, result.Valid)
	})

	t.Run("rejects disallowed table", func(t *testing.T) {
		v := NewSQLValidator(
			[]string{"public"},
			[]string{"users"},
			[]string{"SELECT"},
		)

		result := v.Validate("SELECT * FROM credentials")

		assert.False(t, result.Valid)
	})

	t.Run("allows qualified table name with allowed schema", func(t *testing.T) {
		v := NewSQLValidator(
			[]string{"public"},
			[]string{"users"},
			[]string{"SELECT"},
		)

		result := v.Validate("SELECT * FROM public.users")

		assert.True(t, result.Valid)
	})

	t.Run("validates INSERT statement", func(t *testing.T) {
		v := NewSQLValidator(
			[]string{"public"},
			[]string{"users"},
			[]string{"INSERT"},
		)

		result := v.Validate("INSERT INTO users (name) VALUES ('test')")

		assert.True(t, result.Valid)
		assert.Contains(t, result.OperationsUsed, "INSERT")
	})

	t.Run("validates UPDATE statement", func(t *testing.T) {
		v := NewSQLValidator(
			[]string{"public"},
			[]string{"users"},
			[]string{"UPDATE"},
		)

		result := v.Validate("UPDATE users SET name = 'test' WHERE id = 1")

		assert.True(t, result.Valid)
		assert.Contains(t, result.OperationsUsed, "UPDATE")
	})

	t.Run("validates DELETE statement", func(t *testing.T) {
		v := NewSQLValidator(
			[]string{"public"},
			[]string{"users"},
			[]string{"DELETE"},
		)

		result := v.Validate("DELETE FROM users WHERE id = 1")

		assert.True(t, result.Valid)
		assert.Contains(t, result.OperationsUsed, "DELETE")
	})

	t.Run("extracts multiple tables from JOIN", func(t *testing.T) {
		v := NewSQLValidator(
			[]string{"public"},
			[]string{"users", "orders"},
			[]string{"SELECT"},
		)

		result := v.Validate("SELECT * FROM users JOIN orders ON users.id = orders.user_id")

		assert.True(t, result.Valid)
		assert.Len(t, result.TablesAccessed, 2)
		assert.Contains(t, result.TablesAccessed, "users")
		assert.Contains(t, result.TablesAccessed, "orders")
	})

	t.Run("normalizes query in result", func(t *testing.T) {
		v := NewSQLValidator(nil, nil, []string{"SELECT"})

		result := v.Validate("SELECT   *   FROM   users\n\t  WHERE id = 1")

		assert.NotEmpty(t, result.NormalizedQuery)
		assert.NotContains(t, result.NormalizedQuery, "\n")
		assert.NotContains(t, result.NormalizedQuery, "\t")
	})
}

// =============================================================================
// Dangerous Functions Tests
// =============================================================================

func TestSQLValidator_DangerousFunctions(t *testing.T) {
	dangerousFuncs := []string{
		"pg_read_file",
		"pg_read_binary_file",
		"pg_ls_dir",
		"lo_import",
		"lo_export",
		"dblink",
		"dblink_exec",
		"set_config",
	}

	for _, funcName := range dangerousFuncs {
		t.Run("rejects "+funcName, func(t *testing.T) {
			v := NewSQLValidator(nil, nil, []string{"SELECT"})

			result := v.Validate("SELECT " + funcName + "('test')")

			assert.False(t, result.Valid)
			assert.NotEmpty(t, result.Errors)
		})
	}
}

// =============================================================================
// normalizeQuery Tests
// =============================================================================

func TestSQLValidator_normalizeQuery(t *testing.T) {
	v := NewSQLValidator(nil, nil, nil)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "trims whitespace",
			input:    "  SELECT * FROM users  ",
			expected: "SELECT * FROM users",
		},
		{
			name:     "collapses newlines",
			input:    "SELECT *\nFROM\nusers",
			expected: "SELECT * FROM users",
		},
		{
			name:     "collapses tabs",
			input:    "SELECT *\tFROM\tusers",
			expected: "SELECT * FROM users",
		},
		{
			name:     "collapses multiple spaces",
			input:    "SELECT   *    FROM     users",
			expected: "SELECT * FROM users",
		},
		{
			name:     "handles mixed whitespace",
			input:    "SELECT  \n\t *  \n FROM \t users",
			expected: "SELECT * FROM users",
		},
		{
			name:     "preserves single spaces",
			input:    "SELECT * FROM users WHERE id = 1",
			expected: "SELECT * FROM users WHERE id = 1",
		},
		{
			name:     "handles empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "handles only whitespace",
			input:    "   \n\t   ",
			expected: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := v.normalizeQuery(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// =============================================================================
// ValidationResult Tests
// =============================================================================

func TestValidationResult_Struct(t *testing.T) {
	t.Run("valid result", func(t *testing.T) {
		result := ValidationResult{
			Valid:           true,
			Errors:          nil,
			Warnings:        nil,
			TablesAccessed:  []string{"users", "orders"},
			OperationsUsed:  []string{"SELECT"},
			NormalizedQuery: "SELECT * FROM users",
		}

		assert.True(t, result.Valid)
		assert.Empty(t, result.Errors)
		assert.Len(t, result.TablesAccessed, 2)
		assert.Len(t, result.OperationsUsed, 1)
	})

	t.Run("invalid result with errors", func(t *testing.T) {
		result := ValidationResult{
			Valid:  false,
			Errors: []string{"Table not allowed", "Operation not allowed"},
		}

		assert.False(t, result.Valid)
		assert.Len(t, result.Errors, 2)
	})

	t.Run("result with warnings", func(t *testing.T) {
		result := ValidationResult{
			Valid:    true,
			Warnings: []string{"Consider adding an index"},
		}

		assert.True(t, result.Valid)
		assert.Len(t, result.Warnings, 1)
	})

	t.Run("zero value result", func(t *testing.T) {
		var result ValidationResult

		assert.False(t, result.Valid)
		assert.Nil(t, result.Errors)
		assert.Nil(t, result.TablesAccessed)
	})
}

// =============================================================================
// ValidateAndNormalize Tests
// =============================================================================

func TestSQLValidator_ValidateAndNormalize(t *testing.T) {
	t.Run("returns result and normalized query on success", func(t *testing.T) {
		v := NewSQLValidator(
			[]string{"public"},
			[]string{"users"},
			[]string{"SELECT"},
		)

		result, normalized, err := v.ValidateAndNormalize("SELECT  *  FROM  users")

		require.NoError(t, err)
		assert.True(t, result.Valid)
		assert.Equal(t, "SELECT * FROM users", normalized)
	})

	t.Run("returns error on validation failure", func(t *testing.T) {
		v := NewSQLValidator(nil, nil, []string{"SELECT"})

		result, normalized, err := v.ValidateAndNormalize("DELETE FROM users")

		require.Error(t, err)
		assert.False(t, result.Valid)
		assert.Empty(t, normalized)
		assert.Contains(t, err.Error(), "validation failed")
	})
}

// =============================================================================
// Operation Type Detection Tests
// =============================================================================

func TestSQLValidator_getOperationType(t *testing.T) {
	v := NewSQLValidator(nil, nil, []string{
		"SELECT", "INSERT", "UPDATE", "DELETE",
		"CREATE", "DROP", "ALTER", "TRUNCATE", "GRANT",
	})

	tests := []struct {
		sql          string
		expectedOp   string
		shouldDetect bool
	}{
		{"SELECT * FROM users", "SELECT", true},
		{"INSERT INTO users VALUES (1)", "INSERT", true},
		{"UPDATE users SET name = 'x'", "UPDATE", true},
		{"DELETE FROM users", "DELETE", true},
		{"CREATE TABLE test (id int)", "CREATE", true},
		{"DROP TABLE test", "DROP", true},
		{"ALTER TABLE test ADD col int", "ALTER", true},
		{"TRUNCATE users", "TRUNCATE", true},
		{"GRANT SELECT ON users TO role", "GRANT", true},
	}

	for _, tc := range tests {
		t.Run(tc.sql, func(t *testing.T) {
			result := v.Validate(tc.sql)

			if tc.shouldDetect {
				assert.Contains(t, result.OperationsUsed, tc.expectedOp)
			}
		})
	}
}

// =============================================================================
// Edge Cases
// =============================================================================

func TestSQLValidator_EdgeCases(t *testing.T) {
	t.Run("handles subqueries", func(t *testing.T) {
		v := NewSQLValidator(
			[]string{"public"},
			[]string{"users", "orders"},
			[]string{"SELECT"},
		)

		result := v.Validate("SELECT * FROM users WHERE id IN (SELECT user_id FROM orders)")

		assert.True(t, result.Valid)
		assert.Contains(t, result.TablesAccessed, "users")
		assert.Contains(t, result.TablesAccessed, "orders")
	})

	t.Run("handles CTE (WITH clause)", func(t *testing.T) {
		v := NewSQLValidator(
			[]string{"public"},
			[]string{"users"},
			[]string{"SELECT"},
		)

		result := v.Validate("WITH active AS (SELECT * FROM users WHERE active = true) SELECT * FROM active")

		assert.True(t, result.Valid)
		assert.Contains(t, result.TablesAccessed, "users")
	})

	t.Run("case insensitive blocked pattern check", func(t *testing.T) {
		v := NewSQLValidator(nil, nil, []string{"SELECT"})

		result := v.Validate("SELECT * FROM PG_CATALOG.pg_tables")

		assert.False(t, result.Valid)
	})

	t.Run("handles UNION queries", func(t *testing.T) {
		v := NewSQLValidator(
			[]string{"public"},
			[]string{"users", "admins"},
			[]string{"SELECT"},
		)

		result := v.Validate("SELECT name FROM users UNION SELECT name FROM admins")

		assert.True(t, result.Valid)
		assert.Contains(t, result.TablesAccessed, "users")
		assert.Contains(t, result.TablesAccessed, "admins")
	})

	t.Run("handles invalid SQL gracefully", func(t *testing.T) {
		v := NewSQLValidator(nil, nil, []string{"SELECT"})

		result := v.Validate("SELEKT * FORM users")

		assert.False(t, result.Valid)
		assert.NotEmpty(t, result.Errors)
	})
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkNewSQLValidator(b *testing.B) {
	schemas := []string{"public", "api", "auth"}
	tables := []string{"users", "orders", "products", "categories"}
	ops := []string{"SELECT", "INSERT", "UPDATE", "DELETE"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewSQLValidator(schemas, tables, ops)
	}
}

func BenchmarkValidate_SimpleSelect(b *testing.B) {
	v := NewSQLValidator(
		[]string{"public"},
		[]string{"users"},
		[]string{"SELECT"},
	)
	sql := "SELECT * FROM users WHERE id = 1"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = v.Validate(sql)
	}
}

func BenchmarkValidate_ComplexJoin(b *testing.B) {
	v := NewSQLValidator(
		[]string{"public"},
		[]string{"users", "orders", "products"},
		[]string{"SELECT"},
	)
	sql := `SELECT u.name, o.total, p.name
		FROM users u
		JOIN orders o ON u.id = o.user_id
		JOIN products p ON o.product_id = p.id
		WHERE u.active = true`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = v.Validate(sql)
	}
}

func BenchmarkNormalizeQuery(b *testing.B) {
	v := NewSQLValidator(nil, nil, nil)
	sql := "SELECT   *  \n  FROM   users  \t WHERE   id = 1"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = v.normalizeQuery(sql)
	}
}
