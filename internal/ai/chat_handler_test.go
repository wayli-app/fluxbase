package ai

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseMCPExecuteSQLResult(t *testing.T) {
	h := &ChatHandler{} // Method doesn't use handler fields

	t.Run("parses valid execute_sql result", func(t *testing.T) {
		args := map[string]any{
			"sql":         "SELECT * FROM trips ORDER BY date DESC LIMIT 4",
			"description": "Get last 4 trips",
		}
		resultContent := `{
			"success": true,
			"row_count": 4,
			"columns": ["id", "destination", "date"],
			"rows": [
				{"id": 1, "destination": "Paris", "date": "2024-01-15"},
				{"id": 2, "destination": "Tokyo", "date": "2024-02-20"},
				{"id": 3, "destination": "London", "date": "2024-03-10"},
				{"id": 4, "destination": "Berlin", "date": "2024-04-05"}
			],
			"summary": "Returned 4 row(s) in 15ms",
			"duration_ms": 15,
			"tables": ["trips"]
		}`

		result := h.parseMCPExecuteSQLResult(args, resultContent)

		require.NotNil(t, result)
		assert.Equal(t, "SELECT * FROM trips ORDER BY date DESC LIMIT 4", result.Query)
		assert.Equal(t, "Returned 4 row(s) in 15ms", result.Summary)
		assert.Equal(t, 4, result.RowCount)
		assert.Len(t, result.Data, 4)
		assert.Equal(t, "Paris", result.Data[0]["destination"])
	})

	t.Run("returns nil for invalid JSON", func(t *testing.T) {
		args := map[string]any{
			"sql": "SELECT * FROM trips",
		}
		resultContent := "not valid json"

		result := h.parseMCPExecuteSQLResult(args, resultContent)

		assert.Nil(t, result)
	})

	t.Run("returns nil for failed query", func(t *testing.T) {
		args := map[string]any{
			"sql": "SELECT * FROM nonexistent",
		}
		resultContent := `{
			"success": false,
			"row_count": 0,
			"columns": [],
			"rows": [],
			"summary": "Query failed",
			"duration_ms": 5,
			"tables": []
		}`

		result := h.parseMCPExecuteSQLResult(args, resultContent)

		assert.Nil(t, result)
	})

	t.Run("handles empty rows", func(t *testing.T) {
		args := map[string]any{
			"sql": "SELECT * FROM trips WHERE 1=0",
		}
		resultContent := `{
			"success": true,
			"row_count": 0,
			"columns": ["id", "destination"],
			"rows": [],
			"summary": "Returned 0 row(s) in 5ms",
			"duration_ms": 5,
			"tables": ["trips"]
		}`

		result := h.parseMCPExecuteSQLResult(args, resultContent)

		require.NotNil(t, result)
		assert.Equal(t, 0, result.RowCount)
		assert.Empty(t, result.Data)
	})

	t.Run("handles missing sql in args", func(t *testing.T) {
		args := map[string]any{
			"description": "Some query",
		}
		resultContent := `{
			"success": true,
			"row_count": 1,
			"columns": ["id"],
			"rows": [{"id": 1}],
			"summary": "Returned 1 row(s) in 5ms",
			"duration_ms": 5,
			"tables": ["trips"]
		}`

		result := h.parseMCPExecuteSQLResult(args, resultContent)

		require.NotNil(t, result)
		assert.Equal(t, "", result.Query)
		assert.Equal(t, 1, result.RowCount)
	})
}
