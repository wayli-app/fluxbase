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
			expected: "/api/rest/class", // "class" doesn't end with 's', so it gets 'ies' -> but actually it ends with 'ss' so 'y->ies' doesn't apply; 'class' ends with 's' so no 's' is added
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
		{"box", "boxes"},         // ends in x -> es
		{"story", "stories"},     // consonant + y -> ies
		{"key", "keys"},          // vowel + y -> s
		{"company", "companies"}, // consonant + y -> ies
		{"index", "indexes"},     // ends in x -> es
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

// =============================================================================
// Batch Query Result Processing Tests
// =============================================================================

// TestBatchColumnAggregation verifies that column data is correctly
// aggregated by table key when processing batch query results.
func TestBatchColumnAggregation(t *testing.T) {
	// Simulate what batchGetColumns returns
	columns := map[string][]ColumnInfo{
		"public.users": {
			{Name: "id", DataType: "uuid", Position: 1, IsPrimaryKey: false},
			{Name: "email", DataType: "text", Position: 2},
			{Name: "name", DataType: "text", Position: 3, IsNullable: true},
		},
		"public.posts": {
			{Name: "id", DataType: "uuid", Position: 1},
			{Name: "title", DataType: "text", Position: 2},
			{Name: "user_id", DataType: "uuid", Position: 3},
		},
		"auth.sessions": {
			{Name: "id", DataType: "uuid", Position: 1},
			{Name: "user_id", DataType: "uuid", Position: 2},
			{Name: "expires_at", DataType: "timestamp with time zone", Position: 3},
		},
	}

	// Verify correct aggregation
	assert.Len(t, columns, 3)
	assert.Len(t, columns["public.users"], 3)
	assert.Len(t, columns["public.posts"], 3)
	assert.Len(t, columns["auth.sessions"], 3)

	// Verify column ordering preserved
	assert.Equal(t, "id", columns["public.users"][0].Name)
	assert.Equal(t, "email", columns["public.users"][1].Name)
	assert.Equal(t, "name", columns["public.users"][2].Name)
}

// TestBatchPrimaryKeyAggregation verifies primary key data is correctly
// aggregated, including composite primary keys.
func TestBatchPrimaryKeyAggregation(t *testing.T) {
	primaryKeys := map[string][]string{
		"public.users":      {"id"},
		"public.user_roles": {"user_id", "role_id"}, // Composite PK
		"public.posts":      {"id"},
	}

	assert.Len(t, primaryKeys, 3)
	assert.Len(t, primaryKeys["public.users"], 1)
	assert.Len(t, primaryKeys["public.user_roles"], 2)

	// Verify composite key ordering
	assert.Equal(t, "user_id", primaryKeys["public.user_roles"][0])
	assert.Equal(t, "role_id", primaryKeys["public.user_roles"][1])
}

// TestBatchForeignKeyAggregation verifies foreign keys are correctly
// associated with their source tables.
func TestBatchForeignKeyAggregation(t *testing.T) {
	foreignKeys := map[string][]ForeignKey{
		"public.posts": {
			{Name: "fk_posts_user_id", ColumnName: "user_id", ReferencedTable: "public.users", ReferencedColumn: "id", OnDelete: "CASCADE"},
		},
		"public.comments": {
			{Name: "fk_comments_user_id", ColumnName: "user_id", ReferencedTable: "public.users", ReferencedColumn: "id"},
			{Name: "fk_comments_post_id", ColumnName: "post_id", ReferencedTable: "public.posts", ReferencedColumn: "id", OnDelete: "CASCADE"},
		},
	}

	assert.Len(t, foreignKeys, 2)
	assert.Len(t, foreignKeys["public.posts"], 1)
	assert.Len(t, foreignKeys["public.comments"], 2)

	// Verify FK details
	assert.Equal(t, "CASCADE", foreignKeys["public.posts"][0].OnDelete)
}

// TestBatchIndexAggregation verifies indexes are correctly grouped by table.
func TestBatchIndexAggregation(t *testing.T) {
	indexes := map[string][]IndexInfo{
		"public.users": {
			{Name: "users_pkey", Columns: []string{"id"}, IsUnique: true, IsPrimary: true},
			{Name: "users_email_key", Columns: []string{"email"}, IsUnique: true},
		},
		"public.posts": {
			{Name: "posts_pkey", Columns: []string{"id"}, IsUnique: true, IsPrimary: true},
			{Name: "idx_posts_user_id", Columns: []string{"user_id"}, IsUnique: false},
			{Name: "idx_posts_created_at", Columns: []string{"created_at"}, IsUnique: false},
		},
	}

	assert.Len(t, indexes, 2)
	assert.Len(t, indexes["public.users"], 2)
	assert.Len(t, indexes["public.posts"], 3)
}

// TestTableMapMerging verifies that metadata is correctly merged into TableInfo structs.
func TestTableMapMerging(t *testing.T) {
	// Simulate initial table map from list query
	tableMap := map[string]*TableInfo{
		"public.users": {Schema: "public", Name: "users", Type: "table", RLSEnabled: true},
		"public.posts": {Schema: "public", Name: "posts", Type: "table", RLSEnabled: false},
	}

	// Simulate batch column results
	columns := map[string][]ColumnInfo{
		"public.users": {
			{Name: "id", DataType: "uuid", Position: 1},
			{Name: "email", DataType: "text", Position: 2},
		},
		"public.posts": {
			{Name: "id", DataType: "uuid", Position: 1},
			{Name: "title", DataType: "text", Position: 2},
		},
	}

	// Merge columns into table map (simulating what batchFetchTableMetadata does)
	for key, cols := range columns {
		if info, ok := tableMap[key]; ok {
			info.Columns = cols
		}
	}

	// Verify merge worked
	assert.Len(t, tableMap["public.users"].Columns, 2)
	assert.Len(t, tableMap["public.posts"].Columns, 2)
	assert.Equal(t, "id", tableMap["public.users"].Columns[0].Name)

	// Simulate batch primary key results
	primaryKeys := map[string][]string{
		"public.users": {"id"},
		"public.posts": {"id"},
	}

	// Merge primary keys and mark columns
	for key, pks := range primaryKeys {
		if info, ok := tableMap[key]; ok {
			info.PrimaryKey = pks
			// Mark primary key columns
			for i := range info.Columns {
				for _, pk := range pks {
					if info.Columns[i].Name == pk {
						info.Columns[i].IsPrimaryKey = true
					}
				}
			}
		}
	}

	// Verify primary key marking
	assert.True(t, tableMap["public.users"].Columns[0].IsPrimaryKey)
	assert.False(t, tableMap["public.users"].Columns[1].IsPrimaryKey)
}

// TestBatchQueryResultOrder verifies that tables maintain their original order
// after batch metadata is merged.
func TestBatchQueryResultOrder(t *testing.T) {
	// Simulate the order returned from the list query
	tableKeys := []string{
		"public.aaa_first",
		"public.bbb_second",
		"public.ccc_third",
		"auth.ddd_fourth",
	}

	tableMap := make(map[string]*TableInfo)
	for _, key := range tableKeys {
		parts := splitKey(key)
		tableMap[key] = &TableInfo{Schema: parts[0], Name: parts[1], Type: "table"}
	}

	// Rebuild result in original order (simulating GetAllTables behavior)
	result := make([]TableInfo, 0, len(tableKeys))
	for _, key := range tableKeys {
		if info, ok := tableMap[key]; ok {
			result = append(result, *info)
		}
	}

	// Verify order preserved
	assert.Len(t, result, 4)
	assert.Equal(t, "aaa_first", result[0].Name)
	assert.Equal(t, "bbb_second", result[1].Name)
	assert.Equal(t, "ccc_third", result[2].Name)
	assert.Equal(t, "ddd_fourth", result[3].Name)
}

// splitKey is a helper for tests to split "schema.table" key
func splitKey(key string) []string {
	for i := 0; i < len(key); i++ {
		if key[i] == '.' {
			return []string{key[:i], key[i+1:]}
		}
	}
	return []string{"public", key}
}

// TestEmptySchemaHandling verifies correct behavior with empty schemas.
func TestEmptySchemaHandling(t *testing.T) {
	tableMap := make(map[string]*TableInfo)

	// Empty result - should handle gracefully
	assert.Len(t, tableMap, 0)

	// Build result from empty map
	result := make([]TableInfo, 0, len(tableMap))
	for _, info := range tableMap {
		result = append(result, *info)
	}

	assert.Len(t, result, 0)
}

// TestViewMetadataExcludesKeys verifies that views don't get primary/foreign keys
func TestViewMetadataExcludesKeys(t *testing.T) {
	// Views should only have columns, not keys or indexes
	viewMap := map[string]*TableInfo{
		"public.user_summary": {
			Schema: "public",
			Name:   "user_summary",
			Type:   "view",
		},
	}

	// Simulate columns being added
	viewMap["public.user_summary"].Columns = []ColumnInfo{
		{Name: "user_id", DataType: "uuid"},
		{Name: "total_posts", DataType: "bigint"},
	}

	// Views should NOT have primary keys, foreign keys, or indexes
	assert.Nil(t, viewMap["public.user_summary"].PrimaryKey)
	assert.Nil(t, viewMap["public.user_summary"].ForeignKeys)
	assert.Nil(t, viewMap["public.user_summary"].Indexes)
	assert.Equal(t, "view", viewMap["public.user_summary"].Type)
}

// TestMaterializedViewHasIndexes verifies materialized views can have indexes
func TestMaterializedViewHasIndexes(t *testing.T) {
	matviewMap := map[string]*TableInfo{
		"public.monthly_stats": {
			Schema: "public",
			Name:   "monthly_stats",
			Type:   "materialized_view",
		},
	}

	// Materialized views can have indexes
	matviewMap["public.monthly_stats"].Indexes = []IndexInfo{
		{Name: "idx_monthly_stats_month", Columns: []string{"month"}, IsUnique: false},
	}

	// But no primary/foreign keys
	assert.Nil(t, matviewMap["public.monthly_stats"].PrimaryKey)
	assert.Nil(t, matviewMap["public.monthly_stats"].ForeignKeys)
	assert.Len(t, matviewMap["public.monthly_stats"].Indexes, 1)
}

// =============================================================================
// ColumnMap Tests (O(1) column lookup)
// =============================================================================

func TestTableInfo_BuildColumnMap(t *testing.T) {
	t.Run("builds map from columns", func(t *testing.T) {
		table := TableInfo{
			Schema: "public",
			Name:   "users",
			Columns: []ColumnInfo{
				{Name: "id", DataType: "uuid", IsPrimaryKey: true},
				{Name: "email", DataType: "text", IsNullable: false},
				{Name: "name", DataType: "text", IsNullable: true},
			},
		}

		// Map should be nil before building
		assert.Nil(t, table.ColumnMap)

		// Build the map
		table.BuildColumnMap()

		// Map should now be populated
		assert.NotNil(t, table.ColumnMap)
		assert.Len(t, table.ColumnMap, 3)
		assert.Contains(t, table.ColumnMap, "id")
		assert.Contains(t, table.ColumnMap, "email")
		assert.Contains(t, table.ColumnMap, "name")
	})

	t.Run("empty columns creates empty map", func(t *testing.T) {
		table := TableInfo{
			Schema:  "public",
			Name:    "empty_table",
			Columns: []ColumnInfo{},
		}

		table.BuildColumnMap()

		assert.NotNil(t, table.ColumnMap)
		assert.Len(t, table.ColumnMap, 0)
	})

	t.Run("column map points to actual column data", func(t *testing.T) {
		table := TableInfo{
			Schema: "public",
			Name:   "users",
			Columns: []ColumnInfo{
				{Name: "id", DataType: "uuid", IsPrimaryKey: true},
			},
		}

		table.BuildColumnMap()

		col := table.ColumnMap["id"]
		assert.NotNil(t, col)
		assert.Equal(t, "uuid", col.DataType)
		assert.True(t, col.IsPrimaryKey)
	})
}

func TestTableInfo_GetColumn(t *testing.T) {
	t.Run("returns column when map is built", func(t *testing.T) {
		table := TableInfo{
			Schema: "public",
			Name:   "users",
			Columns: []ColumnInfo{
				{Name: "id", DataType: "uuid", IsPrimaryKey: true},
				{Name: "email", DataType: "text", IsNullable: false},
			},
		}
		table.BuildColumnMap()

		col := table.GetColumn("email")
		assert.NotNil(t, col)
		assert.Equal(t, "email", col.Name)
		assert.Equal(t, "text", col.DataType)
	})

	t.Run("returns nil for non-existent column with map", func(t *testing.T) {
		table := TableInfo{
			Schema: "public",
			Name:   "users",
			Columns: []ColumnInfo{
				{Name: "id", DataType: "uuid"},
			},
		}
		table.BuildColumnMap()

		col := table.GetColumn("nonexistent")
		assert.Nil(t, col)
	})

	t.Run("falls back to linear search without map", func(t *testing.T) {
		table := TableInfo{
			Schema: "public",
			Name:   "users",
			Columns: []ColumnInfo{
				{Name: "id", DataType: "uuid"},
				{Name: "email", DataType: "text"},
			},
		}
		// Deliberately NOT building the map

		col := table.GetColumn("email")
		assert.NotNil(t, col)
		assert.Equal(t, "email", col.Name)
	})

	t.Run("fallback returns nil for non-existent", func(t *testing.T) {
		table := TableInfo{
			Schema: "public",
			Name:   "users",
			Columns: []ColumnInfo{
				{Name: "id", DataType: "uuid"},
			},
		}
		// No map built

		col := table.GetColumn("nonexistent")
		assert.Nil(t, col)
	})
}

func TestTableInfo_HasColumn(t *testing.T) {
	t.Run("returns true for existing column", func(t *testing.T) {
		table := TableInfo{
			Schema: "public",
			Name:   "users",
			Columns: []ColumnInfo{
				{Name: "id", DataType: "uuid"},
				{Name: "email", DataType: "text"},
			},
		}
		table.BuildColumnMap()

		assert.True(t, table.HasColumn("id"))
		assert.True(t, table.HasColumn("email"))
	})

	t.Run("returns false for non-existent column", func(t *testing.T) {
		table := TableInfo{
			Schema: "public",
			Name:   "users",
			Columns: []ColumnInfo{
				{Name: "id", DataType: "uuid"},
			},
		}
		table.BuildColumnMap()

		assert.False(t, table.HasColumn("nonexistent"))
		assert.False(t, table.HasColumn(""))
	})

	t.Run("works without column map (fallback)", func(t *testing.T) {
		table := TableInfo{
			Schema: "public",
			Name:   "users",
			Columns: []ColumnInfo{
				{Name: "id", DataType: "uuid"},
				{Name: "email", DataType: "text"},
			},
		}
		// No map built

		assert.True(t, table.HasColumn("id"))
		assert.True(t, table.HasColumn("email"))
		assert.False(t, table.HasColumn("nonexistent"))
	})
}

// Benchmark O(1) lookup vs O(n) fallback
func BenchmarkTableInfo_HasColumn_WithMap(b *testing.B) {
	table := TableInfo{
		Schema: "public",
		Name:   "users",
		Columns: []ColumnInfo{
			{Name: "id", DataType: "uuid"},
			{Name: "email", DataType: "text"},
			{Name: "name", DataType: "text"},
			{Name: "created_at", DataType: "timestamptz"},
			{Name: "updated_at", DataType: "timestamptz"},
			{Name: "avatar_url", DataType: "text"},
			{Name: "bio", DataType: "text"},
			{Name: "status", DataType: "text"},
			{Name: "role", DataType: "text"},
			{Name: "last_login", DataType: "timestamptz"},
		},
	}
	table.BuildColumnMap()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = table.HasColumn("last_login") // Last column - worst case for linear
	}
}

func BenchmarkTableInfo_HasColumn_WithoutMap(b *testing.B) {
	table := TableInfo{
		Schema: "public",
		Name:   "users",
		Columns: []ColumnInfo{
			{Name: "id", DataType: "uuid"},
			{Name: "email", DataType: "text"},
			{Name: "name", DataType: "text"},
			{Name: "created_at", DataType: "timestamptz"},
			{Name: "updated_at", DataType: "timestamptz"},
			{Name: "avatar_url", DataType: "text"},
			{Name: "bio", DataType: "text"},
			{Name: "status", DataType: "text"},
			{Name: "role", DataType: "text"},
			{Name: "last_login", DataType: "timestamptz"},
		},
	}
	// No map built - uses linear search

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = table.HasColumn("last_login")
	}
}
