package e2e

import (
	"fmt"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/require"
	"github.com/wayli-app/fluxbase/test"
)

// setupSQLEditorTest prepares the test context for SQL editor tests
func setupSQLEditorTest(t *testing.T) (*test.TestContext, string) {
	tc := test.NewTestContext(t)
	tc.EnsureAuthSchema()

	// Create dashboard admin user
	timestamp := time.Now().UnixNano()
	email := fmt.Sprintf("sql-admin-%s-%d@test.com", t.Name(), timestamp)
	password := "adminpass123456"
	_, token := tc.CreateDashboardAdminUser(email, password)

	return tc, token
}

// TestSQLEditor_Authentication tests authentication requirements
func TestSQLEditor_Authentication(t *testing.T) {
	tc, token := setupSQLEditorTest(t)
	defer tc.Close()

	t.Run("requires authentication", func(t *testing.T) {
		resp := tc.NewRequest("POST", "/api/v1/admin/sql/execute").
			WithJSON(fiber.Map{
				"query": "SELECT 1",
			}).
			Send()

		resp.AssertStatus(fiber.StatusUnauthorized)
	})

	t.Run("accepts valid token", func(t *testing.T) {
		resp := tc.NewRequest("POST", "/api/v1/admin/sql/execute").
			WithAuth(token).
			WithJSON(fiber.Map{
				"query": "SELECT 1",
			}).
			Send()

		resp.AssertStatus(fiber.StatusOK)
	})
}

// TestSQLEditor_SingleSelectQuery tests executing a single SELECT query
func TestSQLEditor_SingleSelectQuery(t *testing.T) {
	tc, token := setupSQLEditorTest(t)
	defer tc.Close()

	resp := tc.NewRequest("POST", "/api/v1/admin/sql/execute").
		WithAuth(token).
		WithJSON(fiber.Map{
			"query": "SELECT 1 as number, 'test' as text",
		}).
		Send().
		AssertStatus(fiber.StatusOK)

	var result struct {
		Results []struct {
			Columns         []string                 `json:"columns"`
			Rows            []map[string]interface{} `json:"rows"`
			RowCount        int                      `json:"row_count"`
			ExecutionTimeMS float64                  `json:"execution_time_ms"`
			Statement       string                   `json:"statement"`
		} `json:"results"`
	}
	resp.JSON(&result)

	require.Len(t, result.Results, 1, "Should have 1 result")
	require.Len(t, result.Results[0].Columns, 2, "Should have 2 columns")
	require.Contains(t, result.Results[0].Columns, "number")
	require.Contains(t, result.Results[0].Columns, "text")
	require.Len(t, result.Results[0].Rows, 1, "Should have 1 row")
	require.Equal(t, 1, result.Results[0].RowCount)
	require.GreaterOrEqual(t, result.Results[0].ExecutionTimeMS, 0.0)

	t.Logf("Query executed in %.2fms", result.Results[0].ExecutionTimeMS)
}

// TestSQLEditor_MultipleStatements tests executing multiple SQL statements
func TestSQLEditor_MultipleStatements(t *testing.T) {
	tc, token := setupSQLEditorTest(t)
	defer tc.Close()

	// Create a temporary table, insert data, and query it
	query := `
		CREATE TEMP TABLE test_table (id INT, name TEXT);
		INSERT INTO test_table (id, name) VALUES (1, 'Alice'), (2, 'Bob');
		SELECT * FROM test_table ORDER BY id;
	`

	resp := tc.NewRequest("POST", "/api/v1/admin/sql/execute").
		WithAuth(token).
		WithJSON(fiber.Map{
			"query": query,
		}).
		Send().
		AssertStatus(fiber.StatusOK)

	var result struct {
		Results []struct {
			Columns         []string                 `json:"columns,omitempty"`
			Rows            []map[string]interface{} `json:"rows,omitempty"`
			RowCount        int                      `json:"row_count"`
			AffectedRows    int64                    `json:"affected_rows,omitempty"`
			ExecutionTimeMS float64                  `json:"execution_time_ms"`
			Error           *string                  `json:"error,omitempty"`
			Statement       string                   `json:"statement"`
		} `json:"results"`
	}
	resp.JSON(&result)

	require.Len(t, result.Results, 3, "Should have 3 results (CREATE, INSERT, SELECT)")

	// First result: CREATE TABLE
	require.Nil(t, result.Results[0].Error, "CREATE TABLE should succeed")
	require.Equal(t, 0, result.Results[0].RowCount)

	// Second result: INSERT
	require.Nil(t, result.Results[1].Error, "INSERT should succeed")
	require.Equal(t, int64(2), result.Results[1].AffectedRows)

	// Third result: SELECT
	require.Nil(t, result.Results[2].Error, "SELECT should succeed")
	require.Len(t, result.Results[2].Columns, 2, "Should have 2 columns")
	require.Len(t, result.Results[2].Rows, 2, "Should have 2 rows")
	require.Equal(t, 2, result.Results[2].RowCount)
}

// TestSQLEditor_QueryErrors tests handling of SQL errors
func TestSQLEditor_QueryErrors(t *testing.T) {
	tc, token := setupSQLEditorTest(t)
	defer tc.Close()

	t.Run("syntax error", func(t *testing.T) {
		resp := tc.NewRequest("POST", "/api/v1/admin/sql/execute").
			WithAuth(token).
			WithJSON(fiber.Map{
				"query": "SELEC * FROM users", // Typo: SELEC instead of SELECT
			}).
			Send().
			AssertStatus(fiber.StatusOK) // API returns 200 even with query errors

		var result struct {
			Results []struct {
				Error           *string `json:"error,omitempty"`
				ExecutionTimeMS float64 `json:"execution_time_ms"`
			} `json:"results"`
		}
		resp.JSON(&result)

		require.Len(t, result.Results, 1)
		require.NotNil(t, result.Results[0].Error, "Should have an error")
		require.Contains(t, *result.Results[0].Error, "syntax", "Error should mention syntax")
	})

	t.Run("table does not exist", func(t *testing.T) {
		resp := tc.NewRequest("POST", "/api/v1/admin/sql/execute").
			WithAuth(token).
			WithJSON(fiber.Map{
				"query": "SELECT * FROM nonexistent_table_xyz",
			}).
			Send().
			AssertStatus(fiber.StatusOK)

		var result struct {
			Results []struct {
				Error *string `json:"error,omitempty"`
			} `json:"results"`
		}
		resp.JSON(&result)

		require.Len(t, result.Results, 1)
		require.NotNil(t, result.Results[0].Error, "Should have an error")
		require.Contains(t, *result.Results[0].Error, "does not exist", "Error should mention table doesn't exist")
	})
}

// TestSQLEditor_EmptyQuery tests validation of empty queries
func TestSQLEditor_EmptyQuery(t *testing.T) {
	tc, token := setupSQLEditorTest(t)
	defer tc.Close()

	resp := tc.NewRequest("POST", "/api/v1/admin/sql/execute").
		WithAuth(token).
		WithJSON(fiber.Map{
			"query": "",
		}).
		Send()

	resp.AssertStatus(fiber.StatusBadRequest)

	var errResp fiber.Map
	resp.JSON(&errResp)
	require.Contains(t, errResp["error"], "empty", "Should mention query is empty")
}

// TestSQLEditor_DDLOperations tests DDL operations
func TestSQLEditor_DDLOperations(t *testing.T) {
	tc, token := setupSQLEditorTest(t)
	defer tc.Close()

	t.Run("create and drop table", func(t *testing.T) {
		// Create a table
		resp := tc.NewRequest("POST", "/api/v1/admin/sql/execute").
			WithAuth(token).
			WithJSON(fiber.Map{
				"query": "CREATE TEMP TABLE test_ddl (id SERIAL PRIMARY KEY, name TEXT NOT NULL)",
			}).
			Send().
			AssertStatus(fiber.StatusOK)

		var result struct {
			Results []struct {
				Error    *string `json:"error,omitempty"`
				RowCount int     `json:"row_count"`
			} `json:"results"`
		}
		resp.JSON(&result)

		require.Len(t, result.Results, 1)
		require.Nil(t, result.Results[0].Error, "CREATE TABLE should succeed")
	})

	t.Run("alter table", func(t *testing.T) {
		// Create a table first
		tc.NewRequest("POST", "/api/v1/admin/sql/execute").
			WithAuth(token).
			WithJSON(fiber.Map{
				"query": "CREATE TEMP TABLE test_alter (id INT)",
			}).
			Send().
			AssertStatus(fiber.StatusOK)

		// Alter the table
		resp := tc.NewRequest("POST", "/api/v1/admin/sql/execute").
			WithAuth(token).
			WithJSON(fiber.Map{
				"query": "ALTER TABLE test_alter ADD COLUMN name TEXT",
			}).
			Send().
			AssertStatus(fiber.StatusOK)

		var result struct {
			Results []struct {
				Error *string `json:"error,omitempty"`
			} `json:"results"`
		}
		resp.JSON(&result)

		require.Len(t, result.Results, 1)
		require.Nil(t, result.Results[0].Error, "ALTER TABLE should succeed")
	})
}

// TestSQLEditor_DMLOperations tests DML operations
func TestSQLEditor_DMLOperations(t *testing.T) {
	tc, token := setupSQLEditorTest(t)
	defer tc.Close()

	// Setup: Create a temp table
	tc.NewRequest("POST", "/api/v1/admin/sql/execute").
		WithAuth(token).
		WithJSON(fiber.Map{
			"query": "CREATE TEMP TABLE test_dml (id SERIAL PRIMARY KEY, name TEXT)",
		}).
		Send().
		AssertStatus(fiber.StatusOK)

	t.Run("insert", func(t *testing.T) {
		resp := tc.NewRequest("POST", "/api/v1/admin/sql/execute").
			WithAuth(token).
			WithJSON(fiber.Map{
				"query": "INSERT INTO test_dml (name) VALUES ('Alice'), ('Bob'), ('Charlie')",
			}).
			Send().
			AssertStatus(fiber.StatusOK)

		var result struct {
			Results []struct {
				AffectedRows int64   `json:"affected_rows"`
				Error        *string `json:"error,omitempty"`
			} `json:"results"`
		}
		resp.JSON(&result)

		require.Len(t, result.Results, 1)
		require.Nil(t, result.Results[0].Error)
		require.Equal(t, int64(3), result.Results[0].AffectedRows)
	})

	t.Run("update", func(t *testing.T) {
		resp := tc.NewRequest("POST", "/api/v1/admin/sql/execute").
			WithAuth(token).
			WithJSON(fiber.Map{
				"query": "UPDATE test_dml SET name = 'Updated' WHERE name = 'Alice'",
			}).
			Send().
			AssertStatus(fiber.StatusOK)

		var result struct {
			Results []struct {
				AffectedRows int64   `json:"affected_rows"`
				Error        *string `json:"error,omitempty"`
			} `json:"results"`
		}
		resp.JSON(&result)

		require.Len(t, result.Results, 1)
		require.Nil(t, result.Results[0].Error)
		require.Equal(t, int64(1), result.Results[0].AffectedRows)
	})

	t.Run("delete", func(t *testing.T) {
		resp := tc.NewRequest("POST", "/api/v1/admin/sql/execute").
			WithAuth(token).
			WithJSON(fiber.Map{
				"query": "DELETE FROM test_dml WHERE name = 'Bob'",
			}).
			Send().
			AssertStatus(fiber.StatusOK)

		var result struct {
			Results []struct {
				AffectedRows int64   `json:"affected_rows"`
				Error        *string `json:"error,omitempty"`
			} `json:"results"`
		}
		resp.JSON(&result)

		require.Len(t, result.Results, 1)
		require.Nil(t, result.Results[0].Error)
		require.Equal(t, int64(1), result.Results[0].AffectedRows)
	})
}

// TestSQLEditor_QueryAuthUsers tests querying the auth.users table
func TestSQLEditor_QueryAuthUsers(t *testing.T) {
	tc, token := setupSQLEditorTest(t)
	defer tc.Close()

	resp := tc.NewRequest("POST", "/api/v1/admin/sql/execute").
		WithAuth(token).
		WithJSON(fiber.Map{
			"query": "SELECT id, email FROM auth.users LIMIT 5",
		}).
		Send().
		AssertStatus(fiber.StatusOK)

	var result struct {
		Results []struct {
			Columns  []string                 `json:"columns"`
			Rows     []map[string]interface{} `json:"rows"`
			RowCount int                      `json:"row_count"`
			Error    *string                  `json:"error,omitempty"`
		} `json:"results"`
	}
	resp.JSON(&result)

	require.Len(t, result.Results, 1)
	require.Nil(t, result.Results[0].Error)
	require.Contains(t, result.Results[0].Columns, "id")
	require.Contains(t, result.Results[0].Columns, "email")
	require.GreaterOrEqual(t, result.Results[0].RowCount, 0)
}

// TestSQLEditor_InvalidJSON tests invalid request body
func TestSQLEditor_InvalidJSON(t *testing.T) {
	tc, token := setupSQLEditorTest(t)
	defer tc.Close()

	resp := tc.NewRequest("POST", "/api/v1/admin/sql/execute").
		WithAuth(token).
		WithBody([]byte("{invalid json")).
		Send()

	resp.AssertStatus(fiber.StatusBadRequest)
}

// TestSQLEditor_LongRunningQuery tests timeout behavior
func TestSQLEditor_LongRunningQuery(t *testing.T) {
	tc, token := setupSQLEditorTest(t)
	defer tc.Close()

	// This test checks if long-running queries are handled
	// Using pg_sleep to simulate a query that takes time
	resp := tc.NewRequest("POST", "/api/v1/admin/sql/execute").
		WithAuth(token).
		WithJSON(fiber.Map{
			"query": "SELECT pg_sleep(0.5)", // Sleep for 0.5 seconds
		}).
		Send().
		AssertStatus(fiber.StatusOK)

	var result struct {
		Results []struct {
			ExecutionTimeMS float64 `json:"execution_time_ms"`
			Error           *string `json:"error,omitempty"`
		} `json:"results"`
	}
	resp.JSON(&result)

	require.Len(t, result.Results, 1)
	require.Nil(t, result.Results[0].Error, "pg_sleep should succeed")
	require.GreaterOrEqual(t, result.Results[0].ExecutionTimeMS, 500.0, "Should take at least 500ms")

	t.Logf("pg_sleep(0.5) took %.2fms", result.Results[0].ExecutionTimeMS)
}
