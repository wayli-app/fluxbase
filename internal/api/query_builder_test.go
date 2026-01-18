package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestQueryBuilder_BuildSelect(t *testing.T) {
	tests := []struct {
		name         string
		setup        func() *QueryBuilder
		expectedSQL  string
		expectedArgs []interface{}
	}{
		{
			name: "simple select all",
			setup: func() *QueryBuilder {
				return NewQueryBuilder("public", "users")
			},
			expectedSQL:  `SELECT * FROM "public"."users"`,
			expectedArgs: nil,
		},
		{
			name: "select specific columns",
			setup: func() *QueryBuilder {
				return NewQueryBuilder("public", "users").
					WithColumns([]string{"id", "email", "name"})
			},
			expectedSQL:  `SELECT "id", "email", "name" FROM "public"."users"`,
			expectedArgs: nil,
		},
		{
			name: "select with single filter",
			setup: func() *QueryBuilder {
				return NewQueryBuilder("public", "users").
					WithFilters([]Filter{
						{Column: "id", Operator: OpEqual, Value: 123},
					})
			},
			expectedSQL:  `SELECT * FROM "public"."users" WHERE "id" = $1`,
			expectedArgs: []interface{}{123},
		},
		{
			name: "select with multiple AND filters",
			setup: func() *QueryBuilder {
				return NewQueryBuilder("public", "users").
					WithFilters([]Filter{
						{Column: "status", Operator: OpEqual, Value: "active"},
						{Column: "age", Operator: OpGreaterOrEqual, Value: 18},
					})
			},
			expectedSQL:  `SELECT * FROM "public"."users" WHERE "status" = $1 AND "age" >= $2`,
			expectedArgs: []interface{}{"active", 18},
		},
		{
			name: "select with OR filters in same group",
			setup: func() *QueryBuilder {
				return NewQueryBuilder("public", "users").
					WithFilters([]Filter{
						{Column: "role", Operator: OpEqual, Value: "admin", OrGroupID: 1},
						{Column: "role", Operator: OpEqual, Value: "moderator", OrGroupID: 1},
					})
			},
			expectedSQL:  `SELECT * FROM "public"."users" WHERE ("role" = $1 OR "role" = $2)`,
			expectedArgs: []interface{}{"admin", "moderator"},
		},
		{
			name: "select with ordering",
			setup: func() *QueryBuilder {
				return NewQueryBuilder("public", "users").
					WithOrder([]OrderBy{
						{Column: "created_at", Desc: true},
					})
			},
			expectedSQL:  `SELECT * FROM "public"."users" ORDER BY "created_at" DESC`,
			expectedArgs: nil,
		},
		{
			name: "select with ordering and nulls",
			setup: func() *QueryBuilder {
				return NewQueryBuilder("public", "users").
					WithOrder([]OrderBy{
						{Column: "name", Desc: false, Nulls: "last"},
					})
			},
			expectedSQL:  `SELECT * FROM "public"."users" ORDER BY "name" ASC NULLS LAST`,
			expectedArgs: nil,
		},
		{
			name: "select with limit and offset",
			setup: func() *QueryBuilder {
				return NewQueryBuilder("public", "users").
					WithLimit(10).
					WithOffset(20)
			},
			expectedSQL:  `SELECT * FROM "public"."users" LIMIT 10 OFFSET 20`,
			expectedArgs: nil,
		},
		{
			name: "select with group by",
			setup: func() *QueryBuilder {
				return NewQueryBuilder("public", "orders").
					WithColumns([]string{"status"}).
					WithGroupBy([]string{"status"})
			},
			expectedSQL:  `SELECT "status" FROM "public"."orders" GROUP BY "status"`,
			expectedArgs: nil,
		},
		{
			name: "complex select with all clauses",
			setup: func() *QueryBuilder {
				return NewQueryBuilder("app", "products").
					WithColumns([]string{"category", "name"}).
					WithFilters([]Filter{
						{Column: "active", Operator: OpEqual, Value: true},
					}).
					WithOrder([]OrderBy{
						{Column: "name", Desc: false},
					}).
					WithLimit(50).
					WithOffset(100)
			},
			expectedSQL:  `SELECT "category", "name" FROM "app"."products" WHERE "active" = $1 ORDER BY "name" ASC LIMIT 50 OFFSET 100`,
			expectedArgs: []interface{}{true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			qb := tt.setup()
			sql, args := qb.BuildSelect()
			assert.Equal(t, tt.expectedSQL, sql)
			assert.Equal(t, tt.expectedArgs, args)
		})
	}
}

func TestQueryBuilder_BuildCount(t *testing.T) {
	tests := []struct {
		name         string
		setup        func() *QueryBuilder
		expectedSQL  string
		expectedArgs []interface{}
	}{
		{
			name: "count all",
			setup: func() *QueryBuilder {
				return NewQueryBuilder("public", "users")
			},
			expectedSQL:  `SELECT COUNT(*) FROM "public"."users"`,
			expectedArgs: nil,
		},
		{
			name: "count with filter",
			setup: func() *QueryBuilder {
				return NewQueryBuilder("public", "users").
					WithFilters([]Filter{
						{Column: "status", Operator: OpEqual, Value: "active"},
					})
			},
			expectedSQL:  `SELECT COUNT(*) FROM "public"."users" WHERE "status" = $1`,
			expectedArgs: []interface{}{"active"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			qb := tt.setup()
			sql, args := qb.BuildCount()
			assert.Equal(t, tt.expectedSQL, sql)
			assert.Equal(t, tt.expectedArgs, args)
		})
	}
}

func TestQueryBuilder_BuildInsert(t *testing.T) {
	tests := []struct {
		name         string
		setup        func() *QueryBuilder
		data         map[string]interface{}
		expectedSQL  string
		expectedArgs int // Just check count since map iteration order is non-deterministic
	}{
		{
			name: "simple insert",
			setup: func() *QueryBuilder {
				return NewQueryBuilder("public", "users")
			},
			data: map[string]interface{}{
				"email": "test@example.com",
			},
			expectedSQL:  `INSERT INTO "public"."users" ("email") VALUES ($1)`,
			expectedArgs: 1,
		},
		{
			name: "insert with returning",
			setup: func() *QueryBuilder {
				return NewQueryBuilder("public", "users").
					WithReturning([]string{"id", "created_at"})
			},
			data: map[string]interface{}{
				"email": "test@example.com",
			},
			expectedSQL:  `INSERT INTO "public"."users" ("email") VALUES ($1) RETURNING "id", "created_at"`,
			expectedArgs: 1,
		},
		{
			name: "insert empty data",
			setup: func() *QueryBuilder {
				return NewQueryBuilder("public", "users")
			},
			data:         map[string]interface{}{},
			expectedSQL:  "",
			expectedArgs: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			qb := tt.setup()
			sql, args := qb.BuildInsert(tt.data)

			if tt.expectedSQL == "" {
				assert.Empty(t, sql)
				assert.Nil(t, args)
				return
			}

			// For single-column inserts, we can check exact SQL
			if len(tt.data) == 1 {
				assert.Equal(t, tt.expectedSQL, sql)
			}
			assert.Equal(t, tt.expectedArgs, len(args))
		})
	}
}

func TestQueryBuilder_BuildUpdate(t *testing.T) {
	tests := []struct {
		name         string
		setup        func() *QueryBuilder
		data         map[string]interface{}
		expectedArgs int
		checkSQL     func(t *testing.T, sql string)
	}{
		{
			name: "update with filter",
			setup: func() *QueryBuilder {
				return NewQueryBuilder("public", "users").
					WithFilters([]Filter{
						{Column: "id", Operator: OpEqual, Value: 123},
					})
			},
			data: map[string]interface{}{
				"name": "Updated Name",
			},
			expectedArgs: 2, // 1 for SET, 1 for WHERE
			checkSQL: func(t *testing.T, sql string) {
				assert.Contains(t, sql, `UPDATE "public"."users" SET`)
				assert.Contains(t, sql, `WHERE "id" =`)
			},
		},
		{
			name: "update with returning",
			setup: func() *QueryBuilder {
				return NewQueryBuilder("public", "users").
					WithFilters([]Filter{
						{Column: "id", Operator: OpEqual, Value: 1},
					}).
					WithReturning([]string{"id", "name"})
			},
			data: map[string]interface{}{
				"name": "New Name",
			},
			expectedArgs: 2,
			checkSQL: func(t *testing.T, sql string) {
				assert.Contains(t, sql, `RETURNING "id", "name"`)
			},
		},
		{
			name: "update empty data",
			setup: func() *QueryBuilder {
				return NewQueryBuilder("public", "users")
			},
			data:         map[string]interface{}{},
			expectedArgs: 0,
			checkSQL: func(t *testing.T, sql string) {
				assert.Empty(t, sql)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			qb := tt.setup()
			sql, args := qb.BuildUpdate(tt.data)

			if tt.expectedArgs == 0 && len(tt.data) == 0 {
				assert.Empty(t, sql)
				assert.Nil(t, args)
				return
			}

			assert.Equal(t, tt.expectedArgs, len(args))
			if tt.checkSQL != nil {
				tt.checkSQL(t, sql)
			}
		})
	}
}

func TestQueryBuilder_BuildDelete(t *testing.T) {
	tests := []struct {
		name         string
		setup        func() *QueryBuilder
		expectedSQL  string
		expectedArgs []interface{}
	}{
		{
			name: "delete all (dangerous but valid)",
			setup: func() *QueryBuilder {
				return NewQueryBuilder("public", "temp_data")
			},
			expectedSQL:  `DELETE FROM "public"."temp_data"`,
			expectedArgs: nil,
		},
		{
			name: "delete with filter",
			setup: func() *QueryBuilder {
				return NewQueryBuilder("public", "users").
					WithFilters([]Filter{
						{Column: "id", Operator: OpEqual, Value: 123},
					})
			},
			expectedSQL:  `DELETE FROM "public"."users" WHERE "id" = $1`,
			expectedArgs: []interface{}{123},
		},
		{
			name: "delete with returning",
			setup: func() *QueryBuilder {
				return NewQueryBuilder("public", "users").
					WithFilters([]Filter{
						{Column: "id", Operator: OpEqual, Value: 1},
					}).
					WithReturning([]string{"id", "email"})
			},
			expectedSQL:  `DELETE FROM "public"."users" WHERE "id" = $1 RETURNING "id", "email"`,
			expectedArgs: []interface{}{1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			qb := tt.setup()
			sql, args := qb.BuildDelete()
			assert.Equal(t, tt.expectedSQL, sql)
			assert.Equal(t, tt.expectedArgs, args)
		})
	}
}

func TestQueryBuilder_FilterOperators(t *testing.T) {
	tests := []struct {
		name         string
		filter       Filter
		expectedSQL  string
		expectedArgs []interface{}
	}{
		{
			name:         "equal",
			filter:       Filter{Column: "name", Operator: OpEqual, Value: "test"},
			expectedSQL:  `SELECT * FROM "public"."t" WHERE "name" = $1`,
			expectedArgs: []interface{}{"test"},
		},
		{
			name:         "not equal",
			filter:       Filter{Column: "status", Operator: OpNotEqual, Value: "deleted"},
			expectedSQL:  `SELECT * FROM "public"."t" WHERE "status" <> $1`,
			expectedArgs: []interface{}{"deleted"},
		},
		{
			name:         "greater than",
			filter:       Filter{Column: "age", Operator: OpGreaterThan, Value: 18},
			expectedSQL:  `SELECT * FROM "public"."t" WHERE "age" > $1`,
			expectedArgs: []interface{}{18},
		},
		{
			name:         "less than or equal",
			filter:       Filter{Column: "price", Operator: OpLessOrEqual, Value: 100.0},
			expectedSQL:  `SELECT * FROM "public"."t" WHERE "price" <= $1`,
			expectedArgs: []interface{}{100.0},
		},
		{
			name:         "like",
			filter:       Filter{Column: "email", Operator: OpLike, Value: "%@example.com"},
			expectedSQL:  `SELECT * FROM "public"."t" WHERE "email" LIKE $1`,
			expectedArgs: []interface{}{"%@example.com"},
		},
		{
			name:         "ilike",
			filter:       Filter{Column: "name", Operator: OpILike, Value: "%john%"},
			expectedSQL:  `SELECT * FROM "public"."t" WHERE "name" ILIKE $1`,
			expectedArgs: []interface{}{"%john%"},
		},
		{
			name:         "is null",
			filter:       Filter{Column: "deleted_at", Operator: OpIs, Value: nil},
			expectedSQL:  `SELECT * FROM "public"."t" WHERE "deleted_at" IS NULL`,
			expectedArgs: nil,
		},
		{
			name:         "contains (jsonb @>)",
			filter:       Filter{Column: "metadata", Operator: OpContains, Value: `{"role":"admin"}`},
			expectedSQL:  `SELECT * FROM "public"."t" WHERE "metadata" @> $1`,
			expectedArgs: []interface{}{`{"role":"admin"}`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			qb := NewQueryBuilder("public", "t").
				WithFilters([]Filter{tt.filter})
			sql, args := qb.BuildSelect()
			assert.Equal(t, tt.expectedSQL, sql)
			assert.Equal(t, tt.expectedArgs, args)
		})
	}
}

func TestQueryBuilder_InvalidIdentifiers(t *testing.T) {
	t.Run("invalid column name is skipped", func(t *testing.T) {
		qb := NewQueryBuilder("public", "users").
			WithColumns([]string{"valid_col", "invalid col", "another_valid"})
		sql, _ := qb.BuildSelect()
		// Invalid column should be skipped
		assert.Contains(t, sql, `"valid_col"`)
		assert.Contains(t, sql, `"another_valid"`)
		assert.NotContains(t, sql, "invalid col")
	})

	t.Run("filter with invalid column is skipped", func(t *testing.T) {
		qb := NewQueryBuilder("public", "users").
			WithFilters([]Filter{
				{Column: "valid", Operator: OpEqual, Value: 1},
				{Column: "has space", Operator: OpEqual, Value: 2},
			})
		sql, args := qb.BuildSelect()
		assert.Contains(t, sql, `"valid" = $1`)
		assert.NotContains(t, sql, "has space")
		assert.Equal(t, 1, len(args))
	})
}

func TestNewQueryBuilder(t *testing.T) {
	t.Run("initializes with correct defaults", func(t *testing.T) {
		qb := NewQueryBuilder("myschema", "mytable")
		assert.NotNil(t, qb)

		sql, args := qb.BuildSelect()
		assert.Equal(t, `SELECT * FROM "myschema"."mytable"`, sql)
		assert.Nil(t, args)
	})
}

// =============================================================================
// Cursor Pagination Tests
// =============================================================================

func TestEncodeCursor(t *testing.T) {
	t.Run("encodes cursor correctly", func(t *testing.T) {
		cursor := EncodeCursor("id", "abc123", false)
		assert.NotEmpty(t, cursor)

		// Should be valid base64
		decoded, err := DecodeCursor(cursor)
		assert.NoError(t, err)
		assert.Equal(t, "id", decoded.Column)
		assert.Equal(t, "abc123", decoded.Value)
		assert.False(t, decoded.Desc)
	})

	t.Run("encodes descending cursor", func(t *testing.T) {
		cursor := EncodeCursor("created_at", "2025-01-01", true)
		decoded, err := DecodeCursor(cursor)
		assert.NoError(t, err)
		assert.Equal(t, "created_at", decoded.Column)
		assert.True(t, decoded.Desc)
	})

	t.Run("encodes numeric value", func(t *testing.T) {
		cursor := EncodeCursor("count", 42, false)
		decoded, err := DecodeCursor(cursor)
		assert.NoError(t, err)
		assert.Equal(t, float64(42), decoded.Value) // JSON unmarshals numbers as float64
	})
}

func TestDecodeCursor(t *testing.T) {
	t.Run("decodes valid cursor", func(t *testing.T) {
		// First encode, then decode
		original := EncodeCursor("id", "test123", false)
		decoded, err := DecodeCursor(original)
		assert.NoError(t, err)
		assert.Equal(t, "id", decoded.Column)
		assert.Equal(t, "test123", decoded.Value)
	})

	t.Run("fails on invalid base64", func(t *testing.T) {
		_, err := DecodeCursor("not-valid-base64!!!")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid cursor encoding")
	})

	t.Run("fails on invalid JSON", func(t *testing.T) {
		// Valid base64 but invalid JSON
		_, err := DecodeCursor("bm90LWpzb24=") // "not-json" in base64
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid cursor format")
	})

	t.Run("fails on missing column", func(t *testing.T) {
		// Valid base64 of {"v": "value"} without column
		_, err := DecodeCursor("eyJ2IjoidmFsdWUifQ==")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cursor missing column")
	})
}

func TestQueryBuilder_WithCursor(t *testing.T) {
	t.Run("applies cursor condition ascending", func(t *testing.T) {
		cursor := EncodeCursor("id", "last123", false)

		qb := NewQueryBuilder("public", "users")
		err := qb.WithCursor(cursor, "")
		assert.NoError(t, err)

		sql, args := qb.BuildSelect()
		assert.Contains(t, sql, `WHERE "id" > $1`)
		assert.Len(t, args, 1)
		assert.Equal(t, "last123", args[0])
	})

	t.Run("applies cursor condition descending", func(t *testing.T) {
		cursor := EncodeCursor("created_at", "2025-01-01", true)

		qb := NewQueryBuilder("public", "users")
		err := qb.WithCursor(cursor, "")
		assert.NoError(t, err)

		sql, args := qb.BuildSelect()
		assert.Contains(t, sql, `WHERE "created_at" < $1`)
		assert.Len(t, args, 1)
	})

	t.Run("cursor column override", func(t *testing.T) {
		cursor := EncodeCursor("old_column", "value", false)

		qb := NewQueryBuilder("public", "users")
		err := qb.WithCursor(cursor, "new_column")
		assert.NoError(t, err)

		sql, args := qb.BuildSelect()
		assert.Contains(t, sql, `WHERE "new_column" > $1`)
		assert.Len(t, args, 1)
	})

	t.Run("combines cursor with filters", func(t *testing.T) {
		cursor := EncodeCursor("id", "last123", false)

		qb := NewQueryBuilder("public", "users").
			WithFilters([]Filter{{Column: "status", Operator: OpEqual, Value: "active"}})
		err := qb.WithCursor(cursor, "")
		assert.NoError(t, err)

		sql, args := qb.BuildSelect()
		assert.Contains(t, sql, "WHERE")
		assert.Contains(t, sql, `"status" = $1`)
		assert.Contains(t, sql, `"id" > $2`)
		assert.Len(t, args, 2)
	})

	t.Run("empty cursor is no-op", func(t *testing.T) {
		qb := NewQueryBuilder("public", "users")
		err := qb.WithCursor("", "")
		assert.NoError(t, err)

		sql, args := qb.BuildSelect()
		assert.Equal(t, `SELECT * FROM "public"."users"`, sql)
		assert.Nil(t, args)
	})

	t.Run("invalid cursor returns error", func(t *testing.T) {
		qb := NewQueryBuilder("public", "users")
		err := qb.WithCursor("invalid!!!", "")
		assert.Error(t, err)
	})
}
