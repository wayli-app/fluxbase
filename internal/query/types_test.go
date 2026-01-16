package query

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// =============================================================================
// FilterOperator Constants Tests
// =============================================================================

func TestFilterOperator_BasicOperators(t *testing.T) {
	t.Run("comparison operators have expected values", func(t *testing.T) {
		assert.Equal(t, FilterOperator("eq"), OpEqual)
		assert.Equal(t, FilterOperator("neq"), OpNotEqual)
		assert.Equal(t, FilterOperator("gt"), OpGreaterThan)
		assert.Equal(t, FilterOperator("gte"), OpGreaterOrEqual)
		assert.Equal(t, FilterOperator("lt"), OpLessThan)
		assert.Equal(t, FilterOperator("lte"), OpLessOrEqual)
	})

	t.Run("text matching operators have expected values", func(t *testing.T) {
		assert.Equal(t, FilterOperator("like"), OpLike)
		assert.Equal(t, FilterOperator("ilike"), OpILike)
	})

	t.Run("set operators have expected values", func(t *testing.T) {
		assert.Equal(t, FilterOperator("in"), OpIn)
		assert.Equal(t, FilterOperator("nin"), OpNotIn)
	})

	t.Run("null operators have expected values", func(t *testing.T) {
		assert.Equal(t, FilterOperator("is"), OpIs)
		assert.Equal(t, FilterOperator("isnot"), OpIsNot)
	})
}

func TestFilterOperator_ArrayJsonOperators(t *testing.T) {
	t.Run("array/jsonb operators have expected values", func(t *testing.T) {
		assert.Equal(t, FilterOperator("cs"), OpContains)
		assert.Equal(t, FilterOperator("cd"), OpContained)
		assert.Equal(t, FilterOperator("cd"), OpContainedBy) // Alias
		assert.Equal(t, FilterOperator("ov"), OpOverlap)
		assert.Equal(t, FilterOperator("ov"), OpOverlaps) // Alias
	})

	t.Run("aliases point to same value", func(t *testing.T) {
		assert.Equal(t, OpContained, OpContainedBy)
		assert.Equal(t, OpOverlap, OpOverlaps)
	})
}

func TestFilterOperator_TextSearchOperators(t *testing.T) {
	t.Run("full text search operators have expected values", func(t *testing.T) {
		assert.Equal(t, FilterOperator("fts"), OpTextSearch)
		assert.Equal(t, FilterOperator("plfts"), OpPhraseSearch)
		assert.Equal(t, FilterOperator("wfts"), OpWebSearch)
	})
}

func TestFilterOperator_RangeOperators(t *testing.T) {
	t.Run("range operators have expected values", func(t *testing.T) {
		assert.Equal(t, FilterOperator("not"), OpNot)
		assert.Equal(t, FilterOperator("adj"), OpAdjacent)
		assert.Equal(t, FilterOperator("sl"), OpStrictlyLeft)
		assert.Equal(t, FilterOperator("sr"), OpStrictlyRight)
		assert.Equal(t, FilterOperator("nxr"), OpNotExtendRight)
		assert.Equal(t, FilterOperator("nxl"), OpNotExtendLeft)
	})
}

func TestFilterOperator_PostGISOperators(t *testing.T) {
	t.Run("PostGIS spatial operators have expected values", func(t *testing.T) {
		assert.Equal(t, FilterOperator("st_intersects"), OpSTIntersects)
		assert.Equal(t, FilterOperator("st_contains"), OpSTContains)
		assert.Equal(t, FilterOperator("st_within"), OpSTWithin)
		assert.Equal(t, FilterOperator("st_dwithin"), OpSTDWithin)
		assert.Equal(t, FilterOperator("st_distance"), OpSTDistance)
		assert.Equal(t, FilterOperator("st_touches"), OpSTTouches)
		assert.Equal(t, FilterOperator("st_crosses"), OpSTCrosses)
		assert.Equal(t, FilterOperator("st_overlaps"), OpSTOverlaps)
	})
}

func TestFilterOperator_VectorOperators(t *testing.T) {
	t.Run("pgvector similarity operators have expected values", func(t *testing.T) {
		assert.Equal(t, FilterOperator("vec_l2"), OpVectorL2)
		assert.Equal(t, FilterOperator("vec_cos"), OpVectorCosine)
		assert.Equal(t, FilterOperator("vec_ip"), OpVectorIP)
	})
}

func TestFilterOperator_Distinctness(t *testing.T) {
	t.Run("all operators are distinct (excluding aliases)", func(t *testing.T) {
		// Get all unique operator values (excluding aliases)
		operators := []FilterOperator{
			OpEqual, OpNotEqual, OpGreaterThan, OpGreaterOrEqual,
			OpLessThan, OpLessOrEqual, OpLike, OpILike, OpIn, OpNotIn,
			OpIs, OpIsNot, OpContains, OpContained, OpOverlap,
			OpTextSearch, OpPhraseSearch, OpWebSearch, OpNot,
			OpAdjacent, OpStrictlyLeft, OpStrictlyRight,
			OpNotExtendRight, OpNotExtendLeft,
			OpSTIntersects, OpSTContains, OpSTWithin, OpSTDWithin,
			OpSTDistance, OpSTTouches, OpSTCrosses, OpSTOverlaps,
			OpVectorL2, OpVectorCosine, OpVectorIP,
		}

		seen := make(map[FilterOperator]bool)
		for _, op := range operators {
			if seen[op] {
				// Only allow known aliases
				if op != OpContainedBy && op != OpOverlaps {
					t.Errorf("Unexpected duplicate operator: %s", op)
				}
			}
			seen[op] = true
		}
	})
}

func TestFilterOperator_StringConversion(t *testing.T) {
	t.Run("can convert to string", func(t *testing.T) {
		assert.Equal(t, "eq", string(OpEqual))
		assert.Equal(t, "neq", string(OpNotEqual))
		assert.Equal(t, "st_intersects", string(OpSTIntersects))
		assert.Equal(t, "vec_l2", string(OpVectorL2))
	})

	t.Run("can create from string", func(t *testing.T) {
		op := FilterOperator("eq")
		assert.Equal(t, OpEqual, op)

		op2 := FilterOperator("st_contains")
		assert.Equal(t, OpSTContains, op2)
	})
}

// =============================================================================
// Filter Struct Tests
// =============================================================================

func TestFilter_Struct(t *testing.T) {
	t.Run("all fields accessible", func(t *testing.T) {
		filter := Filter{
			Column:    "age",
			Operator:  OpGreaterThan,
			Value:     21,
			IsOr:      true,
			OrGroupID: 1,
		}

		assert.Equal(t, "age", filter.Column)
		assert.Equal(t, OpGreaterThan, filter.Operator)
		assert.Equal(t, 21, filter.Value)
		assert.True(t, filter.IsOr)
		assert.Equal(t, 1, filter.OrGroupID)
	})

	t.Run("zero value filter", func(t *testing.T) {
		var filter Filter

		assert.Empty(t, filter.Column)
		assert.Empty(t, filter.Operator)
		assert.Nil(t, filter.Value)
		assert.False(t, filter.IsOr)
		assert.Equal(t, 0, filter.OrGroupID)
	})

	t.Run("filter with nil value", func(t *testing.T) {
		filter := Filter{
			Column:   "deleted_at",
			Operator: OpIs,
			Value:    nil,
		}

		assert.Equal(t, "deleted_at", filter.Column)
		assert.Equal(t, OpIs, filter.Operator)
		assert.Nil(t, filter.Value)
	})

	t.Run("filter with slice value", func(t *testing.T) {
		filter := Filter{
			Column:   "status",
			Operator: OpIn,
			Value:    []string{"active", "pending"},
		}

		assert.Equal(t, "status", filter.Column)
		assert.Equal(t, OpIn, filter.Operator)
		values, ok := filter.Value.([]string)
		assert.True(t, ok)
		assert.Equal(t, []string{"active", "pending"}, values)
	})

	t.Run("filter with OR grouping", func(t *testing.T) {
		// Multiple filters in same OR group
		filter1 := Filter{
			Column:    "status",
			Operator:  OpEqual,
			Value:     "active",
			IsOr:      true,
			OrGroupID: 1,
		}
		filter2 := Filter{
			Column:    "status",
			Operator:  OpEqual,
			Value:     "pending",
			IsOr:      true,
			OrGroupID: 1,
		}

		assert.Equal(t, filter1.OrGroupID, filter2.OrGroupID)
		assert.True(t, filter1.IsOr)
		assert.True(t, filter2.IsOr)
	})
}

func TestFilter_DifferentValueTypes(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		expected interface{}
	}{
		{"string value", "test", "test"},
		{"int value", 42, 42},
		{"float value", 3.14, 3.14},
		{"bool value", true, true},
		{"nil value", nil, nil},
		{"slice of strings", []string{"a", "b"}, []string{"a", "b"}},
		{"slice of ints", []int{1, 2, 3}, []int{1, 2, 3}},
		{"map value", map[string]interface{}{"key": "value"}, map[string]interface{}{"key": "value"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			filter := Filter{
				Column:   "column",
				Operator: OpEqual,
				Value:    tc.value,
			}

			assert.Equal(t, tc.expected, filter.Value)
		})
	}
}

// =============================================================================
// OrderBy Struct Tests
// =============================================================================

func TestOrderBy_Struct(t *testing.T) {
	t.Run("all fields accessible", func(t *testing.T) {
		orderBy := OrderBy{
			Column:      "created_at",
			Desc:        true,
			Nulls:       "last",
			NullsFirst:  false,
			VectorOp:    OpVectorCosine,
			VectorValue: []float32{0.1, 0.2, 0.3},
		}

		assert.Equal(t, "created_at", orderBy.Column)
		assert.True(t, orderBy.Desc)
		assert.Equal(t, "last", orderBy.Nulls)
		assert.Equal(t, OpVectorCosine, orderBy.VectorOp)
	})

	t.Run("zero value orderBy", func(t *testing.T) {
		var orderBy OrderBy

		assert.Empty(t, orderBy.Column)
		assert.False(t, orderBy.Desc)
		assert.Empty(t, orderBy.Nulls)
		assert.False(t, orderBy.NullsFirst)
		assert.Empty(t, orderBy.VectorOp)
		assert.Nil(t, orderBy.VectorValue)
	})

	t.Run("ascending order with nulls first", func(t *testing.T) {
		orderBy := OrderBy{
			Column:     "priority",
			Desc:       false,
			Nulls:      "first",
			NullsFirst: true,
		}

		assert.False(t, orderBy.Desc)
		assert.Equal(t, "first", orderBy.Nulls)
	})

	t.Run("descending order with nulls last", func(t *testing.T) {
		orderBy := OrderBy{
			Column: "updated_at",
			Desc:   true,
			Nulls:  "last",
		}

		assert.True(t, orderBy.Desc)
		assert.Equal(t, "last", orderBy.Nulls)
	})

	t.Run("vector similarity ordering", func(t *testing.T) {
		embedding := []float32{0.1, 0.2, 0.3, 0.4}
		orderBy := OrderBy{
			Column:      "embedding",
			Desc:        false,
			VectorOp:    OpVectorL2,
			VectorValue: embedding,
		}

		assert.Equal(t, "embedding", orderBy.Column)
		assert.Equal(t, OpVectorL2, orderBy.VectorOp)
		assert.Equal(t, embedding, orderBy.VectorValue)
	})
}

// =============================================================================
// Operator Categories Tests
// =============================================================================

func TestOperatorCategories(t *testing.T) {
	t.Run("comparison operators", func(t *testing.T) {
		comparisonOps := []FilterOperator{
			OpEqual, OpNotEqual, OpGreaterThan, OpGreaterOrEqual,
			OpLessThan, OpLessOrEqual,
		}

		for _, op := range comparisonOps {
			assert.NotEmpty(t, string(op), "Operator should have non-empty string value")
		}
	})

	t.Run("spatial operators start with st_", func(t *testing.T) {
		spatialOps := []FilterOperator{
			OpSTIntersects, OpSTContains, OpSTWithin, OpSTDWithin,
			OpSTDistance, OpSTTouches, OpSTCrosses, OpSTOverlaps,
		}

		for _, op := range spatialOps {
			str := string(op)
			assert.True(t, len(str) > 3 && str[:3] == "st_",
				"Spatial operator %s should start with 'st_'", op)
		}
	})

	t.Run("vector operators start with vec_", func(t *testing.T) {
		vectorOps := []FilterOperator{
			OpVectorL2, OpVectorCosine, OpVectorIP,
		}

		for _, op := range vectorOps {
			str := string(op)
			assert.True(t, len(str) > 4 && str[:4] == "vec_",
				"Vector operator %s should start with 'vec_'", op)
		}
	})
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkFilterCreation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = Filter{
			Column:   "name",
			Operator: OpEqual,
			Value:    "test",
		}
	}
}

func BenchmarkOrderByCreation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = OrderBy{
			Column: "created_at",
			Desc:   true,
			Nulls:  "last",
		}
	}
}

func BenchmarkOperatorStringConversion(b *testing.B) {
	ops := []FilterOperator{
		OpEqual, OpNotEqual, OpIn, OpSTIntersects, OpVectorCosine,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, op := range ops {
			_ = string(op)
		}
	}
}

func BenchmarkFilterWithSliceValue(b *testing.B) {
	values := []string{"active", "pending", "completed"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Filter{
			Column:   "status",
			Operator: OpIn,
			Value:    values,
		}
	}
}

// =============================================================================
// Edge Cases
// =============================================================================

func TestFilter_EdgeCases(t *testing.T) {
	t.Run("empty column name", func(t *testing.T) {
		filter := Filter{
			Column:   "",
			Operator: OpEqual,
			Value:    "test",
		}

		assert.Empty(t, filter.Column)
	})

	t.Run("empty operator", func(t *testing.T) {
		filter := Filter{
			Column:   "name",
			Operator: "",
			Value:    "test",
		}

		assert.Empty(t, filter.Operator)
	})

	t.Run("custom operator string", func(t *testing.T) {
		customOp := FilterOperator("custom_op")
		filter := Filter{
			Column:   "field",
			Operator: customOp,
			Value:    "value",
		}

		assert.Equal(t, FilterOperator("custom_op"), filter.Operator)
	})

	t.Run("complex nested value", func(t *testing.T) {
		complexValue := map[string]interface{}{
			"type": "Point",
			"coordinates": []float64{
				-122.4194, 37.7749,
			},
		}

		filter := Filter{
			Column:   "location",
			Operator: OpSTWithin,
			Value:    complexValue,
		}

		val, ok := filter.Value.(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, "Point", val["type"])
	})
}

func TestOrderBy_EdgeCases(t *testing.T) {
	t.Run("empty column name", func(t *testing.T) {
		orderBy := OrderBy{
			Column: "",
			Desc:   true,
		}

		assert.Empty(t, orderBy.Column)
	})

	t.Run("invalid nulls value", func(t *testing.T) {
		// The struct doesn't validate, so invalid values are allowed
		orderBy := OrderBy{
			Column: "name",
			Nulls:  "invalid",
		}

		assert.Equal(t, "invalid", orderBy.Nulls)
	})

	t.Run("vector ordering without vector value", func(t *testing.T) {
		orderBy := OrderBy{
			Column:   "embedding",
			VectorOp: OpVectorL2,
			// VectorValue is nil
		}

		assert.Equal(t, OpVectorL2, orderBy.VectorOp)
		assert.Nil(t, orderBy.VectorValue)
	})

	t.Run("deprecated NullsFirst field", func(t *testing.T) {
		orderBy := OrderBy{
			Column:     "priority",
			NullsFirst: true,
			Nulls:      "", // New field not set
		}

		// Both can coexist
		assert.True(t, orderBy.NullsFirst)
		assert.Empty(t, orderBy.Nulls)
	})
}
