package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nimbleflux/fluxbase/internal/database"
	"github.com/nimbleflux/fluxbase/internal/util"
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
			result := util.TruncateString(tt.input, tt.maxLen)
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

func TestConvertValue(t *testing.T) {
	t.Run("nil value", func(t *testing.T) {
		result := convertValue(nil)
		assert.Nil(t, result)
	})

	t.Run("UUID as [16]byte", func(t *testing.T) {
		// UUID: 550e8400-e29b-41d4-a716-446655440000
		uuid := [16]byte{0x55, 0x0e, 0x84, 0x00, 0xe2, 0x9b, 0x41, 0xd4, 0xa7, 0x16, 0x44, 0x66, 0x55, 0x44, 0x00, 0x00}
		result := convertValue(uuid)
		assert.Equal(t, "550e8400-e29b-41d4-a716-446655440000", result)
	})

	t.Run("UUID as []byte", func(t *testing.T) {
		// UUID: 550e8400-e29b-41d4-a716-446655440000
		uuid := []byte{0x55, 0x0e, 0x84, 0x00, 0xe2, 0x9b, 0x41, 0xd4, 0xa7, 0x16, 0x44, 0x66, 0x55, 0x44, 0x00, 0x00}
		result := convertValue(uuid)
		assert.Equal(t, "550e8400-e29b-41d4-a716-446655440000", result)
	})

	t.Run("printable 16-byte string stays as is", func(t *testing.T) {
		// "ABCDEFGHIJKLMNOP" - 16 printable ASCII chars
		printable := []byte("ABCDEFGHIJKLMNOP")
		result := convertValue(printable)
		assert.Equal(t, printable, result)
	})

	t.Run("string value unchanged", func(t *testing.T) {
		result := convertValue("hello")
		assert.Equal(t, "hello", result)
	})

	t.Run("int value unchanged", func(t *testing.T) {
		result := convertValue(42)
		assert.Equal(t, 42, result)
	})

	t.Run("float value unchanged", func(t *testing.T) {
		result := convertValue(3.14)
		assert.Equal(t, 3.14, result)
	})

	t.Run("bool value unchanged", func(t *testing.T) {
		result := convertValue(true)
		assert.Equal(t, true, result)
	})
}

func TestFormatUUID(t *testing.T) {
	t.Run("standard UUID", func(t *testing.T) {
		uuid := []byte{0x55, 0x0e, 0x84, 0x00, 0xe2, 0x9b, 0x41, 0xd4, 0xa7, 0x16, 0x44, 0x66, 0x55, 0x44, 0x00, 0x00}
		result := formatUUID(uuid)
		assert.Equal(t, "550e8400-e29b-41d4-a716-446655440000", result)
	})

	t.Run("all zeros UUID", func(t *testing.T) {
		uuid := []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
		result := formatUUID(uuid)
		assert.Equal(t, "00000000-0000-0000-0000-000000000000", result)
	})

	t.Run("all ones UUID", func(t *testing.T) {
		uuid := []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
		result := formatUUID(uuid)
		assert.Equal(t, "ffffffff-ffff-ffff-ffff-ffffffffffff", result)
	})
}

func newTestSQLHandler(pool *pgxpool.Pool) *SQLHandler {
	return &SQLHandler{db: database.NewConnectionWithPool(pool)}
}

func TestGetPoolForQuery_TenantPoolUsed(t *testing.T) {
	handler := newTestSQLHandler(nil)
	tenantPool := &pgxpool.Pool{}

	app := newTestApp(t)
	app.Get("/test", func(c fiber.Ctx) error {
		c.Locals("tenant_db", tenantPool)
		pool := handler.getPoolForQuery(c, "SELECT * FROM public.foo")
		assert.Equal(t, tenantPool, pool, "should use tenant pool when set")
		return c.SendString("OK")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestGetPoolForQuery_BranchPoolOverridesTenantPool(t *testing.T) {
	handler := newTestSQLHandler(nil)
	branchPool := &pgxpool.Pool{}
	tenantPool := &pgxpool.Pool{}

	app := newTestApp(t)
	app.Get("/test", func(c fiber.Ctx) error {
		c.Locals("branch_pool", branchPool)
		c.Locals("tenant_db", tenantPool)
		pool := handler.getPoolForQuery(c, "SELECT * FROM auth.users")
		assert.Equal(t, branchPool, pool, "branch pool should override tenant pool")
		return c.SendString("OK")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestGetPoolForQuery_NoContext_ReturnsMainDB(t *testing.T) {
	mainPool := &pgxpool.Pool{}
	handler := newTestSQLHandler(mainPool)

	app := newTestApp(t)
	app.Get("/test", func(c fiber.Ctx) error {
		pool := handler.getPoolForQuery(c, "SELECT 1")
		assert.Equal(t, mainPool, pool, "should use main pool when no tenant context")
		return c.SendString("OK")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestGetPoolForQuery_TenantPoolForAllSchemas(t *testing.T) {
	handler := newTestSQLHandler(nil)
	tenantPool := &pgxpool.Pool{}

	queries := []string{
		"SELECT * FROM public.users",
		"SELECT * FROM auth.users WHERE id = 1",
		"SELECT * FROM storage.objects",
		"INSERT INTO app.settings (key, value) VALUES ('k', 'v')",
		"SELECT u.*, p.name FROM public.profiles p JOIN auth.users u ON u.id = p.user_id",
	}

	for _, query := range queries {
		name := query
		if len(name) > 50 {
			name = name[:50]
		}
		t.Run(name, func(t *testing.T) {
			app := newTestApp(t)
			app.Get("/test", func(c fiber.Ctx) error {
				c.Locals("tenant_db", tenantPool)
				pool := handler.getPoolForQuery(c, query)
				assert.Equal(t, tenantPool, pool)
				return c.SendString("OK")
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			_, err := app.Test(req)
			require.NoError(t, err)
		})
	}
}
