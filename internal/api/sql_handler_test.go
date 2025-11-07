package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSplitSQLStatements(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		expected []string
	}{
		{
			name:     "single statement",
			query:    "SELECT * FROM users",
			expected: []string{"SELECT * FROM users"},
		},
		{
			name:     "single statement with semicolon",
			query:    "SELECT * FROM users;",
			expected: []string{"SELECT * FROM users"},
		},
		{
			name:  "multiple statements",
			query: "SELECT * FROM users; SELECT * FROM posts;",
			expected: []string{
				"SELECT * FROM users",
				"SELECT * FROM posts",
			},
		},
		{
			name:  "multiple statements with newlines",
			query: "SELECT * FROM users;\n\nSELECT * FROM posts;\n",
			expected: []string{
				"SELECT * FROM users",
				"SELECT * FROM posts",
			},
		},
		{
			name:     "empty query",
			query:    "",
			expected: []string{},
		},
		{
			name:     "only whitespace",
			query:    "   \n\t  ",
			expected: []string{},
		},
		{
			name:     "semicolons only",
			query:    ";;;",
			expected: []string{},
		},
		{
			name:  "statements with extra semicolons",
			query: "SELECT 1;; SELECT 2;;",
			expected: []string{
				"SELECT 1",
				"SELECT 2",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitSQLStatements(tt.query)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{
			name:     "string shorter than max",
			input:    "hello",
			maxLen:   10,
			expected: "hello",
		},
		{
			name:     "string equal to max",
			input:    "hello",
			maxLen:   5,
			expected: "hello",
		},
		{
			name:     "string longer than max",
			input:    "hello world",
			maxLen:   5,
			expected: "hello...",
		},
		{
			name:     "empty string",
			input:    "",
			maxLen:   10,
			expected: "",
		},
		{
			name:     "max length zero",
			input:    "hello",
			maxLen:   0,
			expected: "...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateString(tt.input, tt.maxLen)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExecuteSQLRequest_Validation(t *testing.T) {
	t.Run("valid request", func(t *testing.T) {
		req := ExecuteSQLRequest{
			Query: "SELECT * FROM users",
		}
		assert.NotEmpty(t, req.Query)
	})

	t.Run("empty query", func(t *testing.T) {
		req := ExecuteSQLRequest{
			Query: "",
		}
		assert.Empty(t, req.Query)
	})
}

func TestSQLResult_Structure(t *testing.T) {
	t.Run("SELECT result with data", func(t *testing.T) {
		result := SQLResult{
			Columns:         []string{"id", "name"},
			Rows:            []map[string]any{{"id": 1, "name": "test"}},
			RowCount:        1,
			ExecutionTimeMS: 10.5,
			Statement:       "SELECT * FROM users",
		}

		assert.Len(t, result.Columns, 2)
		assert.Len(t, result.Rows, 1)
		assert.Equal(t, 1, result.RowCount)
		assert.Nil(t, result.Error)
	})

	t.Run("INSERT result", func(t *testing.T) {
		result := SQLResult{
			AffectedRows:    5,
			RowCount:        5,
			ExecutionTimeMS: 8.2,
			Statement:       "INSERT INTO users (name) VALUES ('test')",
		}

		assert.Equal(t, int64(5), result.AffectedRows)
		assert.Equal(t, 5, result.RowCount)
		assert.Nil(t, result.Columns)
		assert.Nil(t, result.Rows)
	})

	t.Run("result with error", func(t *testing.T) {
		errorMsg := "syntax error at or near \"SELEC\""
		result := SQLResult{
			Error:           &errorMsg,
			ExecutionTimeMS: 2.1,
			Statement:       "SELEC * FROM users",
		}

		require.NotNil(t, result.Error)
		assert.Equal(t, errorMsg, *result.Error)
	})
}

func TestExecuteSQLResponse_Structure(t *testing.T) {
	t.Run("single result", func(t *testing.T) {
		response := ExecuteSQLResponse{
			Results: []SQLResult{
				{
					Columns:         []string{"id"},
					Rows:            []map[string]any{{"id": 1}},
					RowCount:        1,
					ExecutionTimeMS: 5.0,
					Statement:       "SELECT 1",
				},
			},
		}

		assert.Len(t, response.Results, 1)
		assert.Equal(t, 1, response.Results[0].RowCount)
	})

	t.Run("multiple results", func(t *testing.T) {
		response := ExecuteSQLResponse{
			Results: []SQLResult{
				{
					Columns:         []string{"id"},
					Rows:            []map[string]any{{"id": 1}},
					RowCount:        1,
					ExecutionTimeMS: 5.0,
					Statement:       "SELECT 1",
				},
				{
					Columns:         []string{"name"},
					Rows:            []map[string]any{{"name": "test"}},
					RowCount:        1,
					ExecutionTimeMS: 7.5,
					Statement:       "SELECT 'test'",
				},
			},
		}

		assert.Len(t, response.Results, 2)
	})

	t.Run("empty results", func(t *testing.T) {
		response := ExecuteSQLResponse{
			Results: []SQLResult{},
		}

		assert.Len(t, response.Results, 0)
	})
}

func TestConstants(t *testing.T) {
	t.Run("max rows per query", func(t *testing.T) {
		assert.Equal(t, 1000, maxRowsPerQuery)
	})

	t.Run("query timeout", func(t *testing.T) {
		assert.NotZero(t, queryTimeout)
	})
}
