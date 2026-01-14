package database

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// =============================================================================
// TableInfo Type Tests
// =============================================================================

func TestTableInfo_Struct(t *testing.T) {
	t.Run("basic table info", func(t *testing.T) {
		table := TableInfo{
			Schema:     "public",
			Name:       "users",
			Type:       "table",
			RLSEnabled: true,
			PrimaryKey: []string{"id"},
			Columns: []ColumnInfo{
				{Name: "id", DataType: "uuid", IsPrimaryKey: true},
				{Name: "name", DataType: "text", IsNullable: true},
			},
		}

		assert.Equal(t, "public", table.Schema)
		assert.Equal(t, "users", table.Name)
		assert.Equal(t, "table", table.Type)
		assert.True(t, table.RLSEnabled)
		assert.Len(t, table.PrimaryKey, 1)
		assert.Len(t, table.Columns, 2)
	})

	t.Run("view info", func(t *testing.T) {
		view := TableInfo{
			Schema:     "analytics",
			Name:       "daily_stats",
			Type:       "view",
			RLSEnabled: false,
		}

		assert.Equal(t, "view", view.Type)
		assert.False(t, view.RLSEnabled)
	})

	t.Run("materialized view info", func(t *testing.T) {
		matview := TableInfo{
			Schema: "reports",
			Name:   "monthly_summary",
			Type:   "materialized_view",
		}

		assert.Equal(t, "materialized_view", matview.Type)
	})

	t.Run("table with composite primary key", func(t *testing.T) {
		table := TableInfo{
			Schema:     "public",
			Name:       "user_roles",
			PrimaryKey: []string{"user_id", "role_id"},
		}

		assert.Len(t, table.PrimaryKey, 2)
		assert.Contains(t, table.PrimaryKey, "user_id")
		assert.Contains(t, table.PrimaryKey, "role_id")
	})

	t.Run("table with REST path", func(t *testing.T) {
		table := TableInfo{
			Schema:   "public",
			Name:     "products",
			RESTPath: "/api/rest/products",
		}

		assert.Equal(t, "/api/rest/products", table.RESTPath)
	})
}

// =============================================================================
// ColumnInfo Type Tests
// =============================================================================

func TestColumnInfo_Struct(t *testing.T) {
	t.Run("nullable column", func(t *testing.T) {
		col := ColumnInfo{
			Name:       "description",
			DataType:   "text",
			IsNullable: true,
			Position:   2,
		}

		assert.True(t, col.IsNullable)
		assert.Equal(t, 2, col.Position)
	})

	t.Run("non-nullable column", func(t *testing.T) {
		col := ColumnInfo{
			Name:       "id",
			DataType:   "uuid",
			IsNullable: false,
		}

		assert.False(t, col.IsNullable)
	})

	t.Run("column with default value", func(t *testing.T) {
		defaultVal := "now()"
		col := ColumnInfo{
			Name:         "created_at",
			DataType:     "timestamp with time zone",
			DefaultValue: &defaultVal,
		}

		assert.NotNil(t, col.DefaultValue)
		assert.Equal(t, "now()", *col.DefaultValue)
	})

	t.Run("column with max length", func(t *testing.T) {
		maxLen := 255
		col := ColumnInfo{
			Name:      "name",
			DataType:  "character varying",
			MaxLength: &maxLen,
		}

		assert.NotNil(t, col.MaxLength)
		assert.Equal(t, 255, *col.MaxLength)
	})

	t.Run("primary key column", func(t *testing.T) {
		col := ColumnInfo{
			Name:         "id",
			DataType:     "uuid",
			IsPrimaryKey: true,
			IsNullable:   false,
		}

		assert.True(t, col.IsPrimaryKey)
		assert.False(t, col.IsNullable)
	})

	t.Run("foreign key column", func(t *testing.T) {
		col := ColumnInfo{
			Name:         "user_id",
			DataType:     "uuid",
			IsForeignKey: true,
		}

		assert.True(t, col.IsForeignKey)
	})

	t.Run("unique column", func(t *testing.T) {
		col := ColumnInfo{
			Name:     "email",
			DataType: "text",
			IsUnique: true,
		}

		assert.True(t, col.IsUnique)
	})

	t.Run("geometry column", func(t *testing.T) {
		col := ColumnInfo{
			Name:     "location",
			DataType: "geometry",
		}

		assert.Equal(t, "geometry", col.DataType)
	})

	t.Run("jsonb column", func(t *testing.T) {
		col := ColumnInfo{
			Name:     "metadata",
			DataType: "jsonb",
		}

		assert.Equal(t, "jsonb", col.DataType)
	})
}

// =============================================================================
// ForeignKey Type Tests
// =============================================================================

func TestForeignKey_Struct(t *testing.T) {
	t.Run("basic foreign key", func(t *testing.T) {
		fk := ForeignKey{
			Name:             "fk_posts_user_id",
			ColumnName:       "user_id",
			ReferencedTable:  "public.users",
			ReferencedColumn: "id",
			OnDelete:         "CASCADE",
			OnUpdate:         "NO ACTION",
		}

		assert.Equal(t, "fk_posts_user_id", fk.Name)
		assert.Equal(t, "user_id", fk.ColumnName)
		assert.Equal(t, "public.users", fk.ReferencedTable)
		assert.Equal(t, "id", fk.ReferencedColumn)
		assert.Equal(t, "CASCADE", fk.OnDelete)
		assert.Equal(t, "NO ACTION", fk.OnUpdate)
	})

	t.Run("foreign key with SET NULL", func(t *testing.T) {
		fk := ForeignKey{
			OnDelete: "SET NULL",
			OnUpdate: "SET NULL",
		}

		assert.Equal(t, "SET NULL", fk.OnDelete)
		assert.Equal(t, "SET NULL", fk.OnUpdate)
	})

	t.Run("foreign key with RESTRICT", func(t *testing.T) {
		fk := ForeignKey{
			OnDelete: "RESTRICT",
			OnUpdate: "RESTRICT",
		}

		assert.Equal(t, "RESTRICT", fk.OnDelete)
	})
}

// =============================================================================
// IndexInfo Type Tests
// =============================================================================

func TestIndexInfo_Struct(t *testing.T) {
	t.Run("primary key index", func(t *testing.T) {
		idx := IndexInfo{
			Name:      "users_pkey",
			Columns:   []string{"id"},
			IsUnique:  true,
			IsPrimary: true,
		}

		assert.True(t, idx.IsPrimary)
		assert.True(t, idx.IsUnique)
		assert.Len(t, idx.Columns, 1)
	})

	t.Run("composite index", func(t *testing.T) {
		idx := IndexInfo{
			Name:     "idx_user_roles",
			Columns:  []string{"user_id", "role_id"},
			IsUnique: true,
		}

		assert.Len(t, idx.Columns, 2)
		assert.True(t, idx.IsUnique)
		assert.False(t, idx.IsPrimary)
	})

	t.Run("non-unique index", func(t *testing.T) {
		idx := IndexInfo{
			Name:     "idx_users_email",
			Columns:  []string{"email"},
			IsUnique: false,
		}

		assert.False(t, idx.IsUnique)
	})
}

// =============================================================================
// FunctionInfo Type Tests
// =============================================================================

func TestFunctionInfo_Struct(t *testing.T) {
	t.Run("basic function", func(t *testing.T) {
		fn := FunctionInfo{
			Schema:      "public",
			Name:        "get_user_by_id",
			Description: "Returns a user by their ID",
			ReturnType:  "users",
			IsSetOf:     false,
			Volatility:  "STABLE",
			Language:    "plpgsql",
		}

		assert.Equal(t, "public", fn.Schema)
		assert.Equal(t, "get_user_by_id", fn.Name)
		assert.Equal(t, "STABLE", fn.Volatility)
		assert.False(t, fn.IsSetOf)
	})

	t.Run("set-returning function", func(t *testing.T) {
		fn := FunctionInfo{
			Name:       "get_all_users",
			ReturnType: "users",
			IsSetOf:    true,
		}

		assert.True(t, fn.IsSetOf)
	})

	t.Run("immutable function", func(t *testing.T) {
		fn := FunctionInfo{
			Name:       "calculate_hash",
			Volatility: "IMMUTABLE",
		}

		assert.Equal(t, "IMMUTABLE", fn.Volatility)
	})

	t.Run("volatile function", func(t *testing.T) {
		fn := FunctionInfo{
			Name:       "insert_log",
			Volatility: "VOLATILE",
		}

		assert.Equal(t, "VOLATILE", fn.Volatility)
	})

	t.Run("sql function", func(t *testing.T) {
		fn := FunctionInfo{
			Language: "sql",
		}

		assert.Equal(t, "sql", fn.Language)
	})
}

// =============================================================================
// FunctionParam Type Tests
// =============================================================================

func TestFunctionParam_Struct(t *testing.T) {
	t.Run("input parameter", func(t *testing.T) {
		param := FunctionParam{
			Name:       "user_id",
			Type:       "uuid",
			Mode:       "IN",
			HasDefault: false,
			Position:   1,
		}

		assert.Equal(t, "IN", param.Mode)
		assert.False(t, param.HasDefault)
	})

	t.Run("output parameter", func(t *testing.T) {
		param := FunctionParam{
			Name: "result",
			Type: "text",
			Mode: "OUT",
		}

		assert.Equal(t, "OUT", param.Mode)
	})

	t.Run("inout parameter", func(t *testing.T) {
		param := FunctionParam{
			Name: "counter",
			Type: "integer",
			Mode: "INOUT",
		}

		assert.Equal(t, "INOUT", param.Mode)
	})

	t.Run("parameter with default", func(t *testing.T) {
		param := FunctionParam{
			Name:       "limit",
			Type:       "integer",
			HasDefault: true,
		}

		assert.True(t, param.HasDefault)
	})
}

// =============================================================================
// VectorColumnInfo Type Tests
// =============================================================================

func TestVectorColumnInfo_Struct(t *testing.T) {
	t.Run("fixed dimension vector", func(t *testing.T) {
		col := VectorColumnInfo{
			SchemaName: "public",
			TableName:  "embeddings",
			ColumnName: "embedding",
			Dimensions: 1536,
		}

		assert.Equal(t, 1536, col.Dimensions)
	})

	t.Run("variable dimension vector", func(t *testing.T) {
		col := VectorColumnInfo{
			SchemaName: "public",
			TableName:  "documents",
			ColumnName: "vector",
			Dimensions: -1, // Variable
		}

		assert.Equal(t, -1, col.Dimensions)
	})
}

// =============================================================================
// BuildRESTPath Tests
// =============================================================================

func TestSchemaInspector_BuildRESTPath(t *testing.T) {
	inspector := &SchemaInspector{}

	tests := []struct {
		name     string
		table    TableInfo
		expected string
	}{
		{
			name: "public schema - simple plural",
			table: TableInfo{
				Schema: "public",
				Name:   "user",
			},
			expected: "/api/rest/users",
		},
		{
			name: "public schema - already plural",
			table: TableInfo{
				Schema: "public",
				Name:   "users",
			},
			expected: "/api/rest/users",
		},
		{
			name: "public schema - ends with y",
			table: TableInfo{
				Schema: "public",
				Name:   "category",
			},
			expected: "/api/rest/categories",
		},
		{
			name: "custom schema",
			table: TableInfo{
				Schema: "auth",
				Name:   "user",
			},
			expected: "/api/rest/auth/users",
		},
		{
			name: "custom schema - already plural",
			table: TableInfo{
				Schema: "storage",
				Name:   "objects",
			},
			expected: "/api/rest/storage/objects",
		},
		{
			name: "ends with s already",
			table: TableInfo{
				Schema: "public",
				Name:   "status",
			},
			expected: "/api/rest/status",
		},
		{
			name: "ends with ss",
			table: TableInfo{
				Schema: "public",
				Name:   "class",
			},
			expected: "/api/rest/classs",
		},
		{
			name: "underscore in name",
			table: TableInfo{
				Schema: "public",
				Name:   "user_profile",
			},
			expected: "/api/rest/user_profiles",
		},
		{
			name: "nested schema",
			table: TableInfo{
				Schema: "analytics",
				Name:   "event",
			},
			expected: "/api/rest/analytics/events",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := inspector.BuildRESTPath(tt.table)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSchemaInspector_BuildRESTPath_Pluralization(t *testing.T) {
	inspector := &SchemaInspector{}

	// Test various pluralization rules
	pluralTests := []struct {
		name           string
		expectedSuffix string
	}{
		{"book", "books"},
		{"box", "boxs"},         // Simple s (not smart pluralization)
		{"story", "stories"},    // y -> ies
		{"key", "keys"},         // Ends in y but preceded by vowel - still gets 'ies' rule applied
		{"company", "companies"}, // y -> ies
		{"index", "indexs"},     // Simple s
	}

	for _, tt := range pluralTests {
		t.Run(tt.name, func(t *testing.T) {
			table := TableInfo{
				Schema: "public",
				Name:   tt.name,
			}
			result := inspector.BuildRESTPath(table)
			assert.Equal(t, "/api/rest/"+tt.expectedSuffix, result)
		})
	}
}

// =============================================================================
// NewSchemaInspector Tests
// =============================================================================

func TestNewSchemaInspector(t *testing.T) {
	t.Run("creates inspector with nil connection", func(t *testing.T) {
		inspector := NewSchemaInspector(nil)
		assert.NotNil(t, inspector)
		assert.Nil(t, inspector.conn)
	})
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkBuildRESTPath_PublicSchema(b *testing.B) {
	inspector := &SchemaInspector{}
	table := TableInfo{Schema: "public", Name: "user"}

	for i := 0; i < b.N; i++ {
		_ = inspector.BuildRESTPath(table)
	}
}

func BenchmarkBuildRESTPath_CustomSchema(b *testing.B) {
	inspector := &SchemaInspector{}
	table := TableInfo{Schema: "auth", Name: "session"}

	for i := 0; i < b.N; i++ {
		_ = inspector.BuildRESTPath(table)
	}
}

func BenchmarkBuildRESTPath_EndsWithY(b *testing.B) {
	inspector := &SchemaInspector{}
	table := TableInfo{Schema: "public", Name: "category"}

	for i := 0; i < b.N; i++ {
		_ = inspector.BuildRESTPath(table)
	}
}
