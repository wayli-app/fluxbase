package tools

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseOrder(t *testing.T) {
	t.Run("dot-separated desc", func(t *testing.T) {
		result, err := parseOrder("created_at.desc")
		assert.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Equal(t, "created_at", result[0].Column)
		assert.True(t, result[0].Desc)
	})

	t.Run("dot-separated asc", func(t *testing.T) {
		result, err := parseOrder("name.asc")
		assert.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Equal(t, "name", result[0].Column)
		assert.False(t, result[0].Desc)
	})

	t.Run("space-separated desc lowercase", func(t *testing.T) {
		result, err := parseOrder("visit_count desc")
		assert.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Equal(t, "visit_count", result[0].Column)
		assert.True(t, result[0].Desc)
	})

	t.Run("space-separated DESC uppercase", func(t *testing.T) {
		result, err := parseOrder("visit_count DESC")
		assert.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Equal(t, "visit_count", result[0].Column)
		assert.True(t, result[0].Desc)
	})

	t.Run("space-separated asc", func(t *testing.T) {
		result, err := parseOrder("name asc")
		assert.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Equal(t, "name", result[0].Column)
		assert.False(t, result[0].Desc)
	})

	t.Run("column only defaults to asc", func(t *testing.T) {
		result, err := parseOrder("created_at")
		assert.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Equal(t, "created_at", result[0].Column)
		assert.False(t, result[0].Desc)
	})

	t.Run("multiple columns dot-separated", func(t *testing.T) {
		result, err := parseOrder("created_at.desc,name.asc")
		assert.NoError(t, err)
		assert.Len(t, result, 2)
		assert.Equal(t, "created_at", result[0].Column)
		assert.True(t, result[0].Desc)
		assert.Equal(t, "name", result[1].Column)
		assert.False(t, result[1].Desc)
	})

	t.Run("multiple columns space-separated", func(t *testing.T) {
		result, err := parseOrder("created_at desc, name asc")
		assert.NoError(t, err)
		assert.Len(t, result, 2)
		assert.Equal(t, "created_at", result[0].Column)
		assert.True(t, result[0].Desc)
		assert.Equal(t, "name", result[1].Column)
		assert.False(t, result[1].Desc)
	})

	t.Run("mixed formats", func(t *testing.T) {
		result, err := parseOrder("created_at.desc, name asc")
		assert.NoError(t, err)
		assert.Len(t, result, 2)
		assert.Equal(t, "created_at", result[0].Column)
		assert.True(t, result[0].Desc)
		assert.Equal(t, "name", result[1].Column)
		assert.False(t, result[1].Desc)
	})

	t.Run("empty string", func(t *testing.T) {
		result, err := parseOrder("")
		assert.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("whitespace only", func(t *testing.T) {
		result, err := parseOrder("   ")
		assert.NoError(t, err)
		assert.Empty(t, result)
	})
}

func TestIsSQLExpression(t *testing.T) {
	t.Run("simple column names are not expressions", func(t *testing.T) {
		simpleNames := []string{
			"id",
			"user_id",
			"created_at",
			"firstName",
			"column123",
		}
		for _, name := range simpleNames {
			assert.False(t, isSQLExpression(name), "expected %s to not be an expression", name)
		}
	})

	t.Run("SQL functions are expressions", func(t *testing.T) {
		expressions := []string{
			"sum(visit_count)",
			"COUNT(*)",
			"avg(price)",
			"MIN(created_at)",
			"max(id)",
			"COALESCE(name, 'unknown')",
		}
		for _, expr := range expressions {
			assert.True(t, isSQLExpression(expr), "expected %s to be an expression", expr)
		}
	})

	t.Run("aliases are expressions", func(t *testing.T) {
		expressions := []string{
			"sum(x) as total",
			"name AS display_name",
			"id as identifier",
		}
		for _, expr := range expressions {
			assert.True(t, isSQLExpression(expr), "expected %s to be an expression", expr)
		}
	})

	t.Run("arithmetic expressions", func(t *testing.T) {
		expressions := []string{
			"price * quantity",
			"total - discount",
			"a + b",
			"count / 100",
		}
		for _, expr := range expressions {
			assert.True(t, isSQLExpression(expr), "expected %s to be an expression", expr)
		}
	})

	t.Run("type casting is expression", func(t *testing.T) {
		expressions := []string{
			"id::text",
			"created_at::date",
		}
		for _, expr := range expressions {
			assert.True(t, isSQLExpression(expr), "expected %s to be an expression", expr)
		}
	})

	t.Run("standalone * is not expression", func(t *testing.T) {
		assert.False(t, isSQLExpression("*"))
	})
}

func TestQuoteColumnOrExpression(t *testing.T) {
	t.Run("quotes simple column names", func(t *testing.T) {
		assert.Equal(t, `"id"`, quoteColumnOrExpression("id"))
		assert.Equal(t, `"user_id"`, quoteColumnOrExpression("user_id"))
		assert.Equal(t, `"created_at"`, quoteColumnOrExpression("created_at"))
	})

	t.Run("passes through SQL expressions unchanged", func(t *testing.T) {
		assert.Equal(t, "sum(visit_count)", quoteColumnOrExpression("sum(visit_count)"))
		assert.Equal(t, "COUNT(*)", quoteColumnOrExpression("COUNT(*)"))
		assert.Equal(t, "sum(visit_count) as total_visits", quoteColumnOrExpression("sum(visit_count) as total_visits"))
		assert.Equal(t, "price * quantity", quoteColumnOrExpression("price * quantity"))
	})
}
