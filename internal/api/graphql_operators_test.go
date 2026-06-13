package api

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/nimbleflux/fluxbase/internal/database"
)

// TestBuildFiltersFromArgs_LogicalOperators tests _and, _or, _not
// in GraphQL filter arguments. The resolver implementation was already
// present but the schema never exposed these fields — now it does.
func TestBuildFiltersFromArgs_LogicalOperators(t *testing.T) {
	gen := NewGraphQLSchemaGenerator(nil, nil, true)

	table := database.TableInfo{
		Schema: "public",
		Name:   "products",
		Columns: []database.ColumnInfo{
			{Name: "id", DataType: "integer"},
			{Name: "name", DataType: "text"},
			{Name: "price", DataType: "numeric"},
			{Name: "in_stock", DataType: "boolean"},
		},
	}

	t.Run("_and flattens conditions into AND filters", func(t *testing.T) {
		args := map[string]interface{}{
			"_and": []interface{}{
				map[string]interface{}{
					"price_gt": 10,
				},
				map[string]interface{}{
					"inStock_eq": true,
				},
			},
		}

		filters := gen.buildFiltersFromArgs(table, args)
		assert.Len(t, filters, 2)
		assert.Equal(t, "price", filters[0].Column)
		assert.Equal(t, "gt", string(filters[0].Operator))
		assert.Equal(t, 10, filters[0].Value)
		assert.Equal(t, "in_stock", filters[1].Column)
		assert.Equal(t, "eq", string(filters[1].Operator))
		assert.False(t, filters[0].IsOr)
		assert.False(t, filters[1].IsOr)
		// _and should not assign OR group IDs
		assert.Equal(t, 0, filters[0].OrGroupID)
		assert.Equal(t, 0, filters[1].OrGroupID)
	})

	t.Run("_or assigns same OrGroupID to conditions", func(t *testing.T) {
		args := map[string]interface{}{
			"_or": []interface{}{
				map[string]interface{}{
					"name_eq": "Widget",
				},
				map[string]interface{}{
					"name_eq": "Gadget",
				},
			},
		}

		filters := gen.buildFiltersFromArgs(table, args)
		assert.Len(t, filters, 2)
		assert.Equal(t, "name", filters[0].Column)
		assert.Equal(t, "name", filters[1].Column)
		// Both should share the same non-zero OrGroupID
		assert.NotEqual(t, 0, filters[0].OrGroupID)
		assert.Equal(t, filters[0].OrGroupID, filters[1].OrGroupID)
	})

	t.Run("_not negates enclosed conditions", func(t *testing.T) {
		args := map[string]interface{}{
			"_not": map[string]interface{}{
				"price_gt": 100,
			},
		}

		filters := gen.buildFiltersFromArgs(table, args)
		assert.Len(t, filters, 1)
		// _not(price > 100) should become price <= 100
		assert.Equal(t, "price", filters[0].Column)
		assert.Equal(t, "lte", string(filters[0].Operator))
	})

	t.Run("nested _and inside _or", func(t *testing.T) {
		args := map[string]interface{}{
			"_or": []interface{}{
				map[string]interface{}{
					"_and": []interface{}{
						map[string]interface{}{"price_gt": 10},
						map[string]interface{}{"price_lt": 50},
					},
				},
				map[string]interface{}{
					"inStock_eq": true,
				},
			},
		}

		filters := gen.buildFiltersFromArgs(table, args)
		assert.Len(t, filters, 3)

		// First two filters (from _and) should share an OrGroupID
		assert.Equal(t, "price", filters[0].Column)
		assert.Equal(t, "price", filters[1].Column)
		assert.NotEqual(t, 0, filters[0].OrGroupID)
		assert.Equal(t, filters[0].OrGroupID, filters[1].OrGroupID)

		// Third filter (inStock) should share the same OrGroupID (from _or)
		assert.Equal(t, "in_stock", filters[2].Column)
		assert.Equal(t, filters[0].OrGroupID, filters[2].OrGroupID)
	})

	t.Run("_not with _or inside", func(t *testing.T) {
		args := map[string]interface{}{
			"_not": map[string]interface{}{
				"_or": []interface{}{
					map[string]interface{}{"name_eq": "A"},
					map[string]interface{}{"name_eq": "B"},
				},
			},
		}

		filters := gen.buildFiltersFromArgs(table, args)
		assert.Len(t, filters, 2)

		// _not(name = A OR name = B) → name != A AND name != B
		assert.Equal(t, "name", filters[0].Column)
		assert.Equal(t, "neq", string(filters[0].Operator))
		assert.Equal(t, "name", filters[1].Column)
		assert.Equal(t, "neq", string(filters[1].Operator))

		// Negated OR conditions should NOT have OrGroupID (they become AND)
		assert.Equal(t, 0, filters[0].OrGroupID)
		assert.Equal(t, 0, filters[1].OrGroupID)
	})

	t.Run("mixed logical and column filters", func(t *testing.T) {
		args := map[string]interface{}{
			"inStock_eq": true,
			"_and": []interface{}{
				map[string]interface{}{"price_gt": 10},
				map[string]interface{}{"price_lt": 100},
			},
		}

		filters := gen.buildFiltersFromArgs(table, args)
		assert.GreaterOrEqual(t, len(filters), 3)

		// All should be AND conditions (no OrGroupID unless from _or)
		for _, f := range filters {
			assert.Equal(t, 0, f.OrGroupID, "filter on %s should not have OrGroupID", f.Column)
		}
	})
}

// TestGenerateFilterType_IncludesLogicalOperators verifies the generated
// filter input type exposes _and, _or, _not fields for self-referencing.
func TestGenerateFilterType_IncludesLogicalOperators(t *testing.T) {
	gen := NewGraphQLSchemaGenerator(nil, nil, true)

	table := database.TableInfo{
		Schema: "public",
		Name:   "products",
		Columns: []database.ColumnInfo{
			{Name: "id", DataType: "integer"},
			{Name: "name", DataType: "text"},
		},
	}

	filterType := gen.generateFilterType(table)
	assert.NotNil(t, filterType)

	// The thunk is resolved when Fields() is called (during schema assembly).
	// Call it to verify the self-referencing fields exist.
	fields := filterType.Fields()

	_, hasAnd := fields["_and"]
	assert.True(t, hasAnd, "filter type should have _and field")

	_, hasOr := fields["_or"]
	assert.True(t, hasOr, "filter type should have _or field")

	_, hasNot := fields["_not"]
	assert.True(t, hasNot, "filter type should have _not field")

	// Verify column-based filters still exist
	_, hasNameEq := fields["name_eq"]
	assert.True(t, hasNameEq, "filter type should still have column-based filters")
}
