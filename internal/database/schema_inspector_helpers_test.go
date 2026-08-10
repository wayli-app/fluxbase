package database

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Unit tests for the pure helpers in schema_inspector.go whose branch coverage
// was incomplete under -short: BuildRESTPath (pluralization + schema path) and
// TableInfo.GetColumn (the nil-ColumnMap linear-search fallback).
//
// The DB-query methods (getColumns, getPrimaryKey, batch*, getFunctionParameters,
// etc.) are covered by integration tests in schema_inspector_internal_test.go
// and intentionally not duplicated here.
//
// Convention matches schema_inspector_test.go: testify, package database.

// =============================================================================
// BuildRESTPath — the pluralization branches that were uncovered
// =============================================================================
//
// Contract source: schema_inspector.go:1156. Pluralization rules, evaluated only
// when the name does not already end in "s":
//   - ends in x/ch/sh -> +"es"
//   - ends in "y" preceded by a vowel -> +"s"        (day -> days)
//   - ends in "y" preceded by a consonant/other -> trim "y" +"ies" (city -> cities)
//   - otherwise -> +"s"
// A non-"public" schema is included in the path as /api/rest/<schema>/<plural>.

func TestSchemaInspector_BuildRESTPath_Pluralization(t *testing.T) {
	t.Parallel()
	inspector := &SchemaInspector{}

	tests := []struct {
		name  string
		table TableInfo
		want  string
	}{
		// Vowel-before-y branch: keeps the y, just appends "s".
		{"day -> days (vowel before y)", TableInfo{Schema: "public", Name: "day"}, "/api/rest/days"},
		{"key -> keys (vowel before y)", TableInfo{Schema: "public", Name: "key"}, "/api/rest/keys"},
		{"boy -> boys (vowel before y)", TableInfo{Schema: "public", Name: "boy"}, "/api/rest/boys"},
		// Consonant-before-y branch: drops the y and appends "ies".
		{"city -> cities (consonant before y)", TableInfo{Schema: "public", Name: "city"}, "/api/rest/cities"},
		{"category -> categories (consonant before y)", TableInfo{Schema: "public", Name: "category"}, "/api/rest/categories"},
		{"policy -> policies (consonant before y)", TableInfo{Schema: "public", Name: "policy"}, "/api/rest/policies"},
		// Names already ending in "s" are returned unchanged.
		{"already plural stays same", TableInfo{Schema: "public", Name: "users"}, "/api/rest/users"},
		{"status ends in s", TableInfo{Schema: "public", Name: "status"}, "/api/rest/status"},
		// Non-public schema is included in the path.
		{"non-public schema included", TableInfo{Schema: "auth", Name: "users"}, "/api/rest/auth/users"},
		{"non-public schema with city", TableInfo{Schema: "storage", Name: "city"}, "/api/rest/storage/cities"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, inspector.BuildRESTPath(tt.table))
		})
	}
}

// =============================================================================
// GetColumn — the nil-ColumnMap fallback branch
// =============================================================================
//
// Contract source: schema_inspector.go:55. When ColumnMap is nil (BuildColumnMap
// not yet called), GetColumn falls back to a linear scan of Columns. The map
// path is already covered; this test exercises the fallback.

func TestTableInfo_GetColumn_NilMapFallback(t *testing.T) {
	t.Parallel()
	t.Run("linear scan finds column when map is nil", func(t *testing.T) {
		t.Parallel()
		// Deliberately do NOT call BuildColumnMap — ColumnMap stays nil.
		table := TableInfo{
			Schema: "public",
			Name:   "users",
			Columns: []ColumnInfo{
				{Name: "id", DataType: "uuid"},
				{Name: "email", DataType: "text"},
			},
		}
		assert.Nil(t, table.ColumnMap, "precondition: map must be nil")

		col := table.GetColumn("email")
		assert.NotNil(t, col)
		assert.Equal(t, "email", col.Name)
		assert.Equal(t, "text", col.DataType)
	})

	t.Run("linear scan returns nil for missing column when map is nil", func(t *testing.T) {
		t.Parallel()
		table := TableInfo{
			Schema: "public",
			Name:   "users",
			Columns: []ColumnInfo{
				{Name: "id", DataType: "uuid"},
			},
		}
		assert.Nil(t, table.GetColumn("does_not_exist"))
	})

	t.Run("linear scan on empty columns returns nil", func(t *testing.T) {
		t.Parallel()
		table := TableInfo{Schema: "public", Name: "empty"}
		assert.Nil(t, table.GetColumn("anything"))
	})

	t.Run("returns pointer into the underlying slice (mutable)", func(t *testing.T) {
		t.Parallel()
		// The fallback returns &Columns[i]; confirm it aliases the slice element.
		table := TableInfo{
			Columns: []ColumnInfo{{Name: "count", DataType: "int"}},
		}
		col := table.GetColumn("count")
		require := assert.New(t)
		require.NotNil(col)
		// Mutate via the returned pointer and confirm the slice reflects it.
		col.DataType = "bigint"
		require.Equal("bigint", table.Columns[0].DataType)
	})
}
