package api

import (
	"testing"

	"github.com/fluxbase-eu/fluxbase/internal/database"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// NewRESTHandler Tests
// =============================================================================

func TestNewRESTHandler(t *testing.T) {
	t.Run("creates handler with nil dependencies", func(t *testing.T) {
		handler := NewRESTHandler(nil, nil, nil)
		assert.NotNil(t, handler)
		assert.Nil(t, handler.db)
		assert.Nil(t, handler.parser)
		assert.Nil(t, handler.schemaCache)
	})

	t.Run("creates handler with parser", func(t *testing.T) {
		parser := NewQueryParser()
		handler := NewRESTHandler(nil, parser, nil)
		assert.NotNil(t, handler)
		assert.Equal(t, parser, handler.parser)
	})
}

func TestRESTHandler_SchemaCache(t *testing.T) {
	t.Run("returns nil when not set", func(t *testing.T) {
		handler := NewRESTHandler(nil, nil, nil)
		assert.Nil(t, handler.SchemaCache())
	})
}

// =============================================================================
// parseTableFromPath Tests
// =============================================================================

func TestParseTableFromPath(t *testing.T) {
	handler := NewRESTHandler(nil, nil, nil)

	tests := []struct {
		name           string
		schemaParam    string
		tableParam     string
		expectedSchema string
		expectedTable  string
	}{
		{
			name:           "single segment - defaults to public schema",
			schemaParam:    "posts",
			tableParam:     "",
			expectedSchema: "public",
			expectedTable:  "posts",
		},
		{
			name:           "two segments - explicit schema",
			schemaParam:    "auth",
			tableParam:     "users",
			expectedSchema: "auth",
			expectedTable:  "users",
		},
		{
			name:           "storage schema",
			schemaParam:    "storage",
			tableParam:     "objects",
			expectedSchema: "storage",
			expectedTable:  "objects",
		},
		{
			name:           "custom schema",
			schemaParam:    "my_schema",
			tableParam:     "my_table",
			expectedSchema: "my_schema",
			expectedTable:  "my_table",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()

			var capturedSchema, capturedTable string
			if tt.tableParam == "" {
				// Single segment path
				app.Get("/tables/:schema", func(c *fiber.Ctx) error {
					capturedSchema, capturedTable = handler.parseTableFromPath(c)
					return c.SendStatus(200)
				})
				_, err := app.Test(fiber.TestConfig{}.NewRequest("GET", "/tables/"+tt.schemaParam, nil))
				require.NoError(t, err)
			} else {
				// Two segment path
				app.Get("/tables/:schema/:table", func(c *fiber.Ctx) error {
					capturedSchema, capturedTable = handler.parseTableFromPath(c)
					return c.SendStatus(200)
				})
				_, err := app.Test(fiber.TestConfig{}.NewRequest("GET", "/tables/"+tt.schemaParam+"/"+tt.tableParam, nil))
				require.NoError(t, err)
			}

			assert.Equal(t, tt.expectedSchema, capturedSchema)
			assert.Equal(t, tt.expectedTable, capturedTable)
		})
	}
}

// =============================================================================
// BuildTablePath Tests
// =============================================================================

func TestBuildTablePath(t *testing.T) {
	handler := NewRESTHandler(nil, nil, nil)

	tests := []struct {
		name     string
		table    database.TableInfo
		expected string
	}{
		{
			name: "public schema",
			table: database.TableInfo{
				Schema: "public",
				Name:   "users",
			},
			expected: "/users",
		},
		{
			name: "auth schema",
			table: database.TableInfo{
				Schema: "auth",
				Name:   "users",
			},
			expected: "/auth/users",
		},
		{
			name: "storage schema",
			table: database.TableInfo{
				Schema: "storage",
				Name:   "buckets",
			},
			expected: "/storage/buckets",
		},
		{
			name: "custom schema",
			table: database.TableInfo{
				Schema: "tenant_123",
				Name:   "orders",
			},
			expected: "/tenant_123/orders",
		},
		{
			name: "table with underscore",
			table: database.TableInfo{
				Schema: "public",
				Name:   "user_profiles",
			},
			expected: "/user_profiles",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.BuildTablePath(tt.table)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// =============================================================================
// BuildFullTablePath Tests
// =============================================================================

func TestBuildFullTablePath(t *testing.T) {
	handler := NewRESTHandler(nil, nil, nil)

	tests := []struct {
		name     string
		table    database.TableInfo
		expected string
	}{
		{
			name: "public schema",
			table: database.TableInfo{
				Schema: "public",
				Name:   "posts",
			},
			expected: "/api/v1/tables/posts",
		},
		{
			name: "auth schema",
			table: database.TableInfo{
				Schema: "auth",
				Name:   "sessions",
			},
			expected: "/api/v1/tables/auth/sessions",
		},
		{
			name: "storage schema",
			table: database.TableInfo{
				Schema: "storage",
				Name:   "objects",
			},
			expected: "/api/v1/tables/storage/objects",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.BuildFullTablePath(tt.table)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// =============================================================================
// columnExists Tests
// =============================================================================

func TestRESTHandler_columnExists(t *testing.T) {
	handler := NewRESTHandler(nil, nil, nil)

	table := database.TableInfo{
		Schema: "public",
		Name:   "users",
		Columns: []database.ColumnInfo{
			{Name: "id", DataType: "uuid"},
			{Name: "email", DataType: "text"},
			{Name: "created_at", DataType: "timestamp"},
		},
	}

	tests := []struct {
		name     string
		column   string
		expected bool
	}{
		{
			name:     "existing column - id",
			column:   "id",
			expected: true,
		},
		{
			name:     "existing column - email",
			column:   "email",
			expected: true,
		},
		{
			name:     "existing column - created_at",
			column:   "created_at",
			expected: true,
		},
		{
			name:     "non-existing column",
			column:   "password",
			expected: false,
		},
		{
			name:     "empty column name",
			column:   "",
			expected: false,
		},
		{
			name:     "case sensitive - uppercase",
			column:   "ID",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.columnExists(table, tt.column)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRESTHandler_columnExists_EmptyTable(t *testing.T) {
	handler := NewRESTHandler(nil, nil, nil)

	table := database.TableInfo{
		Schema:  "public",
		Name:    "empty",
		Columns: []database.ColumnInfo{},
	}

	result := handler.columnExists(table, "any_column")
	assert.False(t, result)
}

// =============================================================================
// TableInfo Type Tests
// =============================================================================

func TestTableInfoType(t *testing.T) {
	t.Run("table type", func(t *testing.T) {
		info := database.TableInfo{
			Schema: "public",
			Name:   "users",
			Type:   "table",
		}
		assert.Equal(t, "table", info.Type)
	})

	t.Run("view type", func(t *testing.T) {
		info := database.TableInfo{
			Schema: "public",
			Name:   "user_view",
			Type:   "view",
		}
		assert.Equal(t, "view", info.Type)
	})

	t.Run("materialized view type", func(t *testing.T) {
		info := database.TableInfo{
			Schema: "public",
			Name:   "stats_mv",
			Type:   "materialized_view",
		}
		assert.Equal(t, "materialized_view", info.Type)
	})
}

// =============================================================================
// RLS Enabled Tests
// =============================================================================

func TestTableInfo_RLSEnabled(t *testing.T) {
	t.Run("RLS enabled", func(t *testing.T) {
		info := database.TableInfo{
			Schema:     "public",
			Name:       "users",
			RLSEnabled: true,
		}
		assert.True(t, info.RLSEnabled)
	})

	t.Run("RLS disabled", func(t *testing.T) {
		info := database.TableInfo{
			Schema:     "public",
			Name:       "config",
			RLSEnabled: false,
		}
		assert.False(t, info.RLSEnabled)
	})
}

// =============================================================================
// Primary Key Tests
// =============================================================================

func TestTableInfo_PrimaryKey(t *testing.T) {
	t.Run("single column primary key", func(t *testing.T) {
		info := database.TableInfo{
			Schema:     "public",
			Name:       "users",
			PrimaryKey: []string{"id"},
		}
		assert.Len(t, info.PrimaryKey, 1)
		assert.Equal(t, "id", info.PrimaryKey[0])
	})

	t.Run("composite primary key", func(t *testing.T) {
		info := database.TableInfo{
			Schema:     "public",
			Name:       "user_roles",
			PrimaryKey: []string{"user_id", "role_id"},
		}
		assert.Len(t, info.PrimaryKey, 2)
		assert.Contains(t, info.PrimaryKey, "user_id")
		assert.Contains(t, info.PrimaryKey, "role_id")
	})

	t.Run("no primary key", func(t *testing.T) {
		info := database.TableInfo{
			Schema:     "public",
			Name:       "logs",
			PrimaryKey: []string{},
		}
		assert.Empty(t, info.PrimaryKey)
	})
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkBuildTablePath_PublicSchema(b *testing.B) {
	handler := NewRESTHandler(nil, nil, nil)
	table := database.TableInfo{
		Schema: "public",
		Name:   "users",
	}

	for i := 0; i < b.N; i++ {
		_ = handler.BuildTablePath(table)
	}
}

func BenchmarkBuildTablePath_CustomSchema(b *testing.B) {
	handler := NewRESTHandler(nil, nil, nil)
	table := database.TableInfo{
		Schema: "auth",
		Name:   "users",
	}

	for i := 0; i < b.N; i++ {
		_ = handler.BuildTablePath(table)
	}
}

func BenchmarkBuildFullTablePath(b *testing.B) {
	handler := NewRESTHandler(nil, nil, nil)
	table := database.TableInfo{
		Schema: "public",
		Name:   "posts",
	}

	for i := 0; i < b.N; i++ {
		_ = handler.BuildFullTablePath(table)
	}
}

func BenchmarkColumnExists(b *testing.B) {
	handler := NewRESTHandler(nil, nil, nil)
	table := database.TableInfo{
		Columns: []database.ColumnInfo{
			{Name: "id"},
			{Name: "name"},
			{Name: "email"},
			{Name: "created_at"},
			{Name: "updated_at"},
		},
	}

	for i := 0; i < b.N; i++ {
		_ = handler.columnExists(table, "email")
	}
}
