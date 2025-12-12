// Package e2e tests the AI SQL validation functionality
package e2e

import (
	"testing"

	"github.com/fluxbase-eu/fluxbase/internal/ai"
	"github.com/stretchr/testify/require"
)

// TestSQLValidatorAllowsValidSelect tests that valid SELECT queries pass validation
func TestSQLValidatorAllowsValidSelect(t *testing.T) {
	// GIVEN: A validator configured for SELECT on users table
	validator := ai.NewSQLValidator(
		[]string{"public"},
		[]string{"users", "orders"},
		[]string{"SELECT"},
	)

	testCases := []struct {
		name string
		sql  string
	}{
		{"simple_select", "SELECT * FROM users"},
		{"select_with_columns", "SELECT id, name, email FROM users"},
		{"select_with_where", "SELECT * FROM users WHERE id = '123'"},
		{"select_with_limit", "SELECT * FROM users LIMIT 10"},
		{"select_with_order", "SELECT * FROM users ORDER BY created_at DESC"},
		{"select_with_join", "SELECT u.*, o.total FROM users u JOIN orders o ON u.id = o.user_id"},
		{"select_count", "SELECT COUNT(*) FROM users"},
		{"select_with_group_by", "SELECT status, COUNT(*) FROM orders GROUP BY status"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// WHEN: Validating the query
			result := validator.Validate(tc.sql)

			// THEN: Validation passes
			require.True(t, result.Valid, "Query should be valid: %s. Errors: %v", tc.sql, result.Errors)
			require.Contains(t, result.OperationsUsed, "SELECT", "Should detect SELECT operation")
		})
	}
}

// TestSQLValidatorBlocksDisallowedOperations tests that non-SELECT operations are blocked
func TestSQLValidatorBlocksDisallowedOperations(t *testing.T) {
	// GIVEN: A validator that only allows SELECT
	validator := ai.NewSQLValidator(
		[]string{"public"},
		[]string{"users"},
		[]string{"SELECT"},
	)

	testCases := []struct {
		name          string
		sql           string
		expectedOp    string
		expectedError string
	}{
		{"insert", "INSERT INTO users (name) VALUES ('test')", "INSERT", "Operation not allowed: INSERT"},
		{"update", "UPDATE users SET name = 'test' WHERE id = 1", "UPDATE", "Operation not allowed: UPDATE"},
		{"delete", "DELETE FROM users WHERE id = 1", "DELETE", "Operation not allowed: DELETE"},
		{"drop_table", "DROP TABLE users", "DROP", "Operation not allowed: DROP"},
		{"truncate", "TRUNCATE TABLE users", "TRUNCATE", "Operation not allowed: TRUNCATE"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// WHEN: Validating the query
			result := validator.Validate(tc.sql)

			// THEN: Validation fails
			require.False(t, result.Valid, "Query should be invalid: %s", tc.sql)
			require.Contains(t, result.Errors, tc.expectedError, "Should have expected error")
		})
	}
}

// TestSQLValidatorBlocksDisallowedTables tests that queries on non-allowed tables are blocked
func TestSQLValidatorBlocksDisallowedTables(t *testing.T) {
	// GIVEN: A validator that only allows 'users' table
	validator := ai.NewSQLValidator(
		[]string{"public"},
		[]string{"users"},
		[]string{"SELECT"},
	)

	testCases := []struct {
		name string
		sql  string
	}{
		{"direct_access", "SELECT * FROM secrets"},
		{"schema_qualified", "SELECT * FROM public.secrets"},
		{"join_with_disallowed", "SELECT * FROM users u JOIN secrets s ON u.id = s.user_id"},
		{"subquery", "SELECT * FROM users WHERE id IN (SELECT user_id FROM secrets)"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// WHEN: Validating the query
			result := validator.Validate(tc.sql)

			// THEN: Validation fails with table error
			require.False(t, result.Valid, "Query should be invalid: %s", tc.sql)
			// At least one error should mention table not allowed
			hasTableError := false
			for _, err := range result.Errors {
				if err == "Table not allowed: secrets" || err == "Table not allowed: public.secrets" {
					hasTableError = true
					break
				}
			}
			require.True(t, hasTableError, "Should have table not allowed error. Errors: %v", result.Errors)
		})
	}
}

// TestSQLValidatorBlocksSystemCatalog tests that system catalog access is blocked
func TestSQLValidatorBlocksSystemCatalog(t *testing.T) {
	// GIVEN: A validator with public schema
	validator := ai.NewSQLValidator(
		[]string{"public"},
		[]string{}, // Wildcard - any table in allowed schemas
		[]string{"SELECT"},
	)

	testCases := []struct {
		name string
		sql  string
	}{
		{"pg_catalog_tables", "SELECT * FROM pg_catalog.pg_tables"},
		{"pg_catalog_users", "SELECT * FROM pg_catalog.pg_user"},
		{"information_schema", "SELECT * FROM information_schema.tables"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// WHEN: Validating the query
			result := validator.Validate(tc.sql)

			// THEN: Validation fails with blocked pattern error
			require.False(t, result.Valid, "Query should be invalid: %s", tc.sql)
		})
	}
}

// TestSQLValidatorBlocksSQLInjection tests that SQL injection attempts are blocked
func TestSQLValidatorBlocksSQLInjection(t *testing.T) {
	// GIVEN: A validator
	validator := ai.NewSQLValidator(
		[]string{"public"},
		[]string{"users"},
		[]string{"SELECT"},
	)

	testCases := []struct {
		name string
		sql  string
	}{
		{"comment_injection", "SELECT * FROM users; DROP TABLE users--"},
		{"block_comment", "SELECT * FROM users /* malicious */ WHERE 1=1"},
		{"union_system_table", "SELECT * FROM users UNION SELECT * FROM pg_catalog.pg_user"},
		{"multiple_statements", "SELECT * FROM users; DELETE FROM users"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// WHEN: Validating the query
			result := validator.Validate(tc.sql)

			// THEN: Validation fails
			require.False(t, result.Valid, "Query should be invalid (injection attempt): %s", tc.sql)
		})
	}
}

// TestSQLValidatorBlocksDangerousFunctions tests that dangerous functions are blocked
func TestSQLValidatorBlocksDangerousFunctions(t *testing.T) {
	// GIVEN: A validator
	validator := ai.NewSQLValidator(
		[]string{"public"},
		[]string{"users"},
		[]string{"SELECT"},
	)

	testCases := []struct {
		name     string
		sql      string
		funcName string
	}{
		{"pg_read_file", "SELECT pg_read_file('/etc/passwd')", "pg_read_file"},
		{"pg_ls_dir", "SELECT pg_ls_dir('/tmp')", "pg_ls_dir"},
		{"lo_import", "SELECT lo_import('/etc/passwd')", "lo_import"},
		{"set_config", "SELECT set_config('log_statement', 'none', false)", "set_config"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// WHEN: Validating the query
			result := validator.Validate(tc.sql)

			// THEN: Validation fails with dangerous function error
			require.False(t, result.Valid, "Query should be invalid (dangerous function): %s", tc.sql)
			hasFunctionError := false
			for _, err := range result.Errors {
				if err == "Dangerous function not allowed: "+tc.funcName {
					hasFunctionError = true
					break
				}
			}
			require.True(t, hasFunctionError, "Should have dangerous function error. Errors: %v", result.Errors)
		})
	}
}

// TestSQLValidatorAllowsWildcardTables tests that wildcard table configuration works
func TestSQLValidatorAllowsWildcardTables(t *testing.T) {
	// GIVEN: A validator with wildcard tables (empty list means all tables in allowed schemas)
	validator := ai.NewSQLValidator(
		[]string{"public"},
		[]string{}, // Empty = wildcard
		[]string{"SELECT"},
	)

	// WHEN: Querying any table in public schema
	result := validator.Validate("SELECT * FROM any_table")

	// THEN: Validation passes
	require.True(t, result.Valid, "Query should be valid with wildcard tables. Errors: %v", result.Errors)
}

// TestSQLValidatorExtractsTableNames tests that table names are correctly extracted
func TestSQLValidatorExtractsTableNames(t *testing.T) {
	// GIVEN: A validator
	validator := ai.NewSQLValidator(
		[]string{"public"},
		[]string{"users", "orders", "products"},
		[]string{"SELECT"},
	)

	// WHEN: Validating a complex query with multiple tables
	sql := "SELECT u.name, o.total, p.name FROM users u JOIN orders o ON u.id = o.user_id JOIN products p ON o.product_id = p.id"
	result := validator.Validate(sql)

	// THEN: All tables are extracted
	require.True(t, result.Valid, "Query should be valid. Errors: %v", result.Errors)
	require.Contains(t, result.TablesAccessed, "users", "Should detect users table")
	require.Contains(t, result.TablesAccessed, "orders", "Should detect orders table")
	require.Contains(t, result.TablesAccessed, "products", "Should detect products table")
}

// TestSQLValidatorBlocksMultipleStatements tests that multiple statements are blocked
func TestSQLValidatorBlocksMultipleStatements(t *testing.T) {
	// GIVEN: A validator
	validator := ai.NewSQLValidator(
		[]string{"public"},
		[]string{"users"},
		[]string{"SELECT", "UPDATE"},
	)

	// WHEN: Validating multiple statements
	result := validator.Validate("SELECT * FROM users; UPDATE users SET name = 'hack'")

	// THEN: Validation fails
	require.False(t, result.Valid, "Multiple statements should be blocked")
	hasMultipleError := false
	for _, err := range result.Errors {
		if err == "Multiple SQL statements not allowed" {
			hasMultipleError = true
			break
		}
	}
	require.True(t, hasMultipleError, "Should have multiple statements error")
}

// TestSQLValidatorHandlesCTEs tests that Common Table Expressions are validated
func TestSQLValidatorHandlesCTEs(t *testing.T) {
	// GIVEN: A validator with wildcard tables (to allow CTE aliases)
	validator := ai.NewSQLValidator(
		[]string{"public"},
		[]string{}, // Wildcard - allows any table including CTE aliases
		[]string{"SELECT"},
	)

	// WHEN: Validating a CTE query
	sql := `
		WITH recent_orders AS (
			SELECT user_id, COUNT(*) as order_count
			FROM orders
			WHERE created_at > '2024-01-01'
			GROUP BY user_id
		)
		SELECT u.name, ro.order_count
		FROM users u
		JOIN recent_orders ro ON u.id = ro.user_id
	`
	result := validator.Validate(sql)

	// THEN: Validation passes and detects actual tables
	require.True(t, result.Valid, "CTE query should be valid. Errors: %v", result.Errors)
	require.Contains(t, result.TablesAccessed, "orders", "Should detect orders table in CTE")
	require.Contains(t, result.TablesAccessed, "users", "Should detect users table")
}

// TestSQLValidatorBlocksDisallowedSchemas tests that queries on non-allowed schemas are blocked
func TestSQLValidatorBlocksDisallowedSchemas(t *testing.T) {
	// GIVEN: A validator that only allows public schema
	validator := ai.NewSQLValidator(
		[]string{"public"},
		[]string{}, // Wildcard
		[]string{"SELECT"},
	)

	testCases := []struct {
		name string
		sql  string
	}{
		{"auth_schema", "SELECT * FROM auth.users"},
		{"app_schema", "SELECT * FROM app.settings"},
		{"custom_schema", "SELECT * FROM custom.data"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// WHEN: Validating the query
			result := validator.Validate(tc.sql)

			// THEN: Validation fails
			require.False(t, result.Valid, "Query should be invalid: %s", tc.sql)
			hasSchemaError := false
			for _, err := range result.Errors {
				if contains(err, "Schema not allowed") {
					hasSchemaError = true
					break
				}
			}
			require.True(t, hasSchemaError, "Should have schema not allowed error. Errors: %v", result.Errors)
		})
	}
}

// TestSQLValidatorNormalizesQuery tests that queries are normalized correctly
func TestSQLValidatorNormalizesQuery(t *testing.T) {
	// GIVEN: A validator
	validator := ai.NewSQLValidator(
		[]string{"public"},
		[]string{"users"},
		[]string{"SELECT"},
	)

	// WHEN: Validating a query with extra whitespace
	sql := `SELECT   *   FROM   users
		WHERE   id = 1`

	result, normalized, err := validator.ValidateAndNormalize(sql)

	// THEN: Query is valid and normalized
	require.NoError(t, err, "Validation should pass")
	require.True(t, result.Valid, "Query should be valid")
	require.NotContains(t, normalized, "\n", "Normalized query should not contain newlines")
	require.NotContains(t, normalized, "  ", "Normalized query should not contain double spaces")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
