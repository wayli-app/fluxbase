package api

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestQueryParser_ParseSelect(t *testing.T) {
	parser := NewQueryParser()

	tests := []struct {
		name     string
		query    string
		expected []string
	}{
		{
			name:     "simple select",
			query:    "select=id,name,email",
			expected: []string{"id", "name", "email"},
		},
		{
			name:     "select with spaces",
			query:    "select=id, name, email",
			expected: []string{"id", "name", "email"},
		},
		{
			name:     "select with relation",
			query:    "select=id,name,posts(id,title)",
			expected: []string{"id", "name"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values, _ := url.ParseQuery(tt.query)
			params, err := parser.Parse(values)

			assert.NoError(t, err)
			assert.Equal(t, tt.expected, params.Select)
		})
	}
}

func TestQueryParser_ParseFilters(t *testing.T) {
	parser := NewQueryParser()

	tests := []struct {
		name           string
		query          string
		expectedColumn string
		expectedOp     FilterOperator
		expectedValue  interface{}
	}{
		{
			name:           "equal filter",
			query:          "name.eq=John",
			expectedColumn: "name",
			expectedOp:     OpEqual,
			expectedValue:  "John",
		},
		{
			name:           "greater than filter",
			query:          "age.gt=18",
			expectedColumn: "age",
			expectedOp:     OpGreaterThan,
			expectedValue:  "18",
		},
		{
			name:           "like filter",
			query:          "email.like=*@example.com",
			expectedColumn: "email",
			expectedOp:     OpLike,
			expectedValue:  "*@example.com",
		},
		{
			name:           "is null filter",
			query:          "deleted_at.is=null",
			expectedColumn: "deleted_at",
			expectedOp:     OpIs,
			expectedValue:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values, _ := url.ParseQuery(tt.query)
			params, err := parser.Parse(values)

			assert.NoError(t, err)
			assert.Len(t, params.Filters, 1)
			assert.Equal(t, tt.expectedColumn, params.Filters[0].Column)
			assert.Equal(t, tt.expectedOp, params.Filters[0].Operator)
			assert.Equal(t, tt.expectedValue, params.Filters[0].Value)
		})
	}
}

func TestQueryParser_ParseOrder(t *testing.T) {
	parser := NewQueryParser()

	tests := []struct {
		name           string
		query          string
		expectedColumn string
		expectedDesc   bool
		expectedNulls  string
	}{
		{
			name:           "ascending order",
			query:          "order=name.asc",
			expectedColumn: "name",
			expectedDesc:   false,
			expectedNulls:  "",
		},
		{
			name:           "descending order",
			query:          "order=created_at.desc",
			expectedColumn: "created_at",
			expectedDesc:   true,
			expectedNulls:  "",
		},
		{
			name:           "order with nulls last",
			query:          "order=updated_at.desc.nullslast",
			expectedColumn: "updated_at",
			expectedDesc:   true,
			expectedNulls:  "last",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values, _ := url.ParseQuery(tt.query)
			params, err := parser.Parse(values)

			assert.NoError(t, err)
			assert.Len(t, params.Order, 1)
			assert.Equal(t, tt.expectedColumn, params.Order[0].Column)
			assert.Equal(t, tt.expectedDesc, params.Order[0].Desc)
			assert.Equal(t, tt.expectedNulls, params.Order[0].Nulls)
		})
	}
}

func TestQueryParser_ParsePagination(t *testing.T) {
	parser := NewQueryParser()

	values, _ := url.ParseQuery("limit=10&offset=20")
	params, err := parser.Parse(values)

	assert.NoError(t, err)
	assert.NotNil(t, params.Limit)
	assert.Equal(t, 10, *params.Limit)
	assert.NotNil(t, params.Offset)
	assert.Equal(t, 20, *params.Offset)
}

func TestQueryParams_ToSQL(t *testing.T) {
	tests := []struct {
		name         string
		params       QueryParams
		expectedSQL  string
		expectedArgs []interface{}
	}{
		{
			name: "simple where clause",
			params: QueryParams{
				Filters: []Filter{
					{Column: "name", Operator: OpEqual, Value: "John"},
				},
			},
			expectedSQL:  "WHERE name = $1",
			expectedArgs: []interface{}{"John"},
		},
		{
			name: "multiple filters",
			params: QueryParams{
				Filters: []Filter{
					{Column: "name", Operator: OpEqual, Value: "John"},
					{Column: "age", Operator: OpGreaterThan, Value: "18"},
				},
			},
			expectedSQL:  "WHERE name = $1 AND age > $2",
			expectedArgs: []interface{}{"John", "18"},
		},
		{
			name: "with order and limit",
			params: QueryParams{
				Order: []OrderBy{
					{Column: "created_at", Desc: true},
				},
				Limit: intPtr(10),
			},
			expectedSQL:  "ORDER BY created_at DESC LIMIT $1",
			expectedArgs: []interface{}{10},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql, args := tt.params.ToSQL("users")
			assert.Equal(t, tt.expectedSQL, sql)
			assert.Equal(t, tt.expectedArgs, args)
		})
	}
}

func intPtr(i int) *int {
	return &i
}

func TestQueryParser_ParseAggregations(t *testing.T) {
	parser := NewQueryParser()

	tests := []struct {
		name                string
		query               string
		expectedSelect      []string
		expectedAggregations []Aggregation
	}{
		{
			name:           "count(*)",
			query:          "select=count(*)",
			expectedSelect: []string{},
			expectedAggregations: []Aggregation{
				{Function: AggCountAll, Column: "", Alias: ""},
			},
		},
		{
			name:           "count(column)",
			query:          "select=count(id)",
			expectedSelect: []string{},
			expectedAggregations: []Aggregation{
				{Function: AggCount, Column: "id", Alias: ""},
			},
		},
		{
			name:           "sum",
			query:          "select=sum(price)",
			expectedSelect: []string{},
			expectedAggregations: []Aggregation{
				{Function: AggSum, Column: "price", Alias: ""},
			},
		},
		{
			name:           "avg",
			query:          "select=avg(rating)",
			expectedSelect: []string{},
			expectedAggregations: []Aggregation{
				{Function: AggAvg, Column: "rating", Alias: ""},
			},
		},
		{
			name:           "min",
			query:          "select=min(created_at)",
			expectedSelect: []string{},
			expectedAggregations: []Aggregation{
				{Function: AggMin, Column: "created_at", Alias: ""},
			},
		},
		{
			name:           "max",
			query:          "select=max(updated_at)",
			expectedSelect: []string{},
			expectedAggregations: []Aggregation{
				{Function: AggMax, Column: "updated_at", Alias: ""},
			},
		},
		{
			name:           "multiple aggregations",
			query:          "select=count(*),sum(price),avg(rating)",
			expectedSelect: []string{},
			expectedAggregations: []Aggregation{
				{Function: AggCountAll, Column: "", Alias: ""},
				{Function: AggSum, Column: "price", Alias: ""},
				{Function: AggAvg, Column: "rating", Alias: ""},
			},
		},
		{
			name:           "aggregation with regular fields",
			query:          "select=category,count(*),sum(price)",
			expectedSelect: []string{"category"},
			expectedAggregations: []Aggregation{
				{Function: AggCountAll, Column: "", Alias: ""},
				{Function: AggSum, Column: "price", Alias: ""},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values, _ := url.ParseQuery(tt.query)
			params, err := parser.Parse(values)

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedSelect, params.Select)
			assert.Equal(t, len(tt.expectedAggregations), len(params.Aggregations))

			for i, expectedAgg := range tt.expectedAggregations {
				assert.Equal(t, expectedAgg.Function, params.Aggregations[i].Function)
				assert.Equal(t, expectedAgg.Column, params.Aggregations[i].Column)
			}
		})
	}
}

func TestQueryParser_ParseGroupBy(t *testing.T) {
	parser := NewQueryParser()

	tests := []struct {
		name             string
		query            string
		expectedGroupBy  []string
	}{
		{
			name:            "single group by",
			query:           "group_by=category",
			expectedGroupBy: []string{"category"},
		},
		{
			name:            "multiple group by",
			query:           "group_by=category,status",
			expectedGroupBy: []string{"category", "status"},
		},
		{
			name:            "group by with spaces",
			query:           "group_by=category, status, region",
			expectedGroupBy: []string{"category", "status", "region"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values, _ := url.ParseQuery(tt.query)
			params, err := parser.Parse(values)

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedGroupBy, params.GroupBy)
		})
	}
}

func TestQueryParams_BuildSelectClause(t *testing.T) {
	tests := []struct {
		name         string
		params       QueryParams
		expectedSQL  string
	}{
		{
			name: "aggregation only - count(*)",
			params: QueryParams{
				Aggregations: []Aggregation{
					{Function: AggCountAll, Column: "", Alias: ""},
				},
			},
			expectedSQL: "COUNT(*) AS count",
		},
		{
			name: "aggregation only - sum",
			params: QueryParams{
				Aggregations: []Aggregation{
					{Function: AggSum, Column: "price", Alias: ""},
				},
			},
			expectedSQL: "SUM(price) AS sum_price",
		},
		{
			name: "multiple aggregations",
			params: QueryParams{
				Aggregations: []Aggregation{
					{Function: AggCount, Column: "id", Alias: ""},
					{Function: AggSum, Column: "price", Alias: ""},
					{Function: AggAvg, Column: "rating", Alias: ""},
				},
			},
			expectedSQL: "COUNT(id) AS count_id, SUM(price) AS sum_price, AVG(rating) AS avg_rating",
		},
		{
			name: "fields with aggregations",
			params: QueryParams{
				Select: []string{"category"},
				Aggregations: []Aggregation{
					{Function: AggCountAll, Column: "", Alias: ""},
					{Function: AggSum, Column: "price", Alias: "total"},
				},
			},
			expectedSQL: "category, COUNT(*) AS count, SUM(price) AS total",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql := tt.params.BuildSelectClause("products")
			assert.Equal(t, tt.expectedSQL, sql)
		})
	}
}

func TestQueryParams_BuildGroupByClause(t *testing.T) {
	tests := []struct {
		name         string
		params       QueryParams
		expectedSQL  string
	}{
		{
			name: "no group by",
			params: QueryParams{
				GroupBy: []string{},
			},
			expectedSQL: "",
		},
		{
			name: "single group by",
			params: QueryParams{
				GroupBy: []string{"category"},
			},
			expectedSQL: " GROUP BY category",
		},
		{
			name: "multiple group by",
			params: QueryParams{
				GroupBy: []string{"category", "status", "region"},
			},
			expectedSQL: " GROUP BY category, status, region",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql := tt.params.BuildGroupByClause()
			assert.Equal(t, tt.expectedSQL, sql)
		})
	}
}

func TestAggregation_ToSQL(t *testing.T) {
	tests := []struct {
		name         string
		agg          Aggregation
		expectedSQL  string
	}{
		{
			name:        "COUNT(*)",
			agg:         Aggregation{Function: AggCountAll, Column: "", Alias: ""},
			expectedSQL: "COUNT(*) AS count",
		},
		{
			name:        "COUNT(column)",
			agg:         Aggregation{Function: AggCount, Column: "id", Alias: ""},
			expectedSQL: "COUNT(id) AS count_id",
		},
		{
			name:        "SUM",
			agg:         Aggregation{Function: AggSum, Column: "price", Alias: ""},
			expectedSQL: "SUM(price) AS sum_price",
		},
		{
			name:        "AVG",
			agg:         Aggregation{Function: AggAvg, Column: "rating", Alias: ""},
			expectedSQL: "AVG(rating) AS avg_rating",
		},
		{
			name:        "MIN",
			agg:         Aggregation{Function: AggMin, Column: "price", Alias: ""},
			expectedSQL: "MIN(price) AS min_price",
		},
		{
			name:        "MAX",
			agg:         Aggregation{Function: AggMax, Column: "price", Alias: ""},
			expectedSQL: "MAX(price) AS max_price",
		},
		{
			name:        "custom alias",
			agg:         Aggregation{Function: AggSum, Column: "price", Alias: "total"},
			expectedSQL: "SUM(price) AS total",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql := tt.agg.ToSQL()
			assert.Equal(t, tt.expectedSQL, sql)
		})
	}
}