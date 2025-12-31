package api

import (
	"net/url"
	"testing"

	"github.com/fluxbase-eu/fluxbase/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testConfig creates a test config with default API settings for testing
func testConfig() *config.Config {
	return &config.Config{
		API: config.APIConfig{
			MaxPageSize:     -1, // Unlimited for most tests
			MaxTotalResults: -1, // Unlimited for most tests
			DefaultPageSize: -1, // No default for most tests
		},
	}
}

func TestQueryParser_ParseSelect(t *testing.T) {
	parser := NewQueryParser(testConfig())

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
	parser := NewQueryParser(testConfig())

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
		{
			name:           "in filter with array",
			query:          "status.in=queued,running",
			expectedColumn: "status",
			expectedOp:     OpIn,
			expectedValue:  []string{"queued", "running"},
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

func TestQueryParser_MultipleFiltersOnSameColumn(t *testing.T) {
	parser := NewQueryParser(testConfig())

	// Test range query: recorded_at=gte.2025-01-01&recorded_at=lte.2025-12-31
	// This should create TWO filters, not just one
	values := url.Values{}
	values.Add("recorded_at", "gte.2025-01-01")
	values.Add("recorded_at", "lte.2025-12-31")

	params, err := parser.Parse(values)
	require.NoError(t, err)
	require.Len(t, params.Filters, 2, "Expected 2 filters for range query")

	// Find gte and lte filters (order may vary due to map iteration)
	var gteFilter, lteFilter *Filter
	for i := range params.Filters {
		if params.Filters[i].Operator == OpGreaterOrEqual {
			gteFilter = &params.Filters[i]
		}
		if params.Filters[i].Operator == OpLessOrEqual {
			lteFilter = &params.Filters[i]
		}
	}

	require.NotNil(t, gteFilter, "Expected gte filter")
	assert.Equal(t, "recorded_at", gteFilter.Column)
	assert.Equal(t, "2025-01-01", gteFilter.Value)

	require.NotNil(t, lteFilter, "Expected lte filter")
	assert.Equal(t, "recorded_at", lteFilter.Column)
	assert.Equal(t, "2025-12-31", lteFilter.Value)
}

func TestQueryParser_ParseOrder(t *testing.T) {
	parser := NewQueryParser(testConfig())

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
	parser := NewQueryParser(testConfig())

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
			expectedSQL:  `WHERE "name" = $1`,
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
			expectedSQL:  `WHERE "name" = $1 AND "age" > $2`,
			expectedArgs: []interface{}{"John", "18"},
		},
		{
			name: "in filter with string array",
			params: QueryParams{
				Filters: []Filter{
					{Column: "status", Operator: OpIn, Value: []string{"queued", "running"}},
				},
			},
			expectedSQL:  `WHERE "status" = ANY($1)`,
			expectedArgs: []interface{}{[]string{"queued", "running"}},
		},
		{
			name: "in filter with single element",
			params: QueryParams{
				Filters: []Filter{
					{Column: "status", Operator: OpIn, Value: []string{"active"}},
				},
			},
			expectedSQL:  `WHERE "status" = ANY($1)`,
			expectedArgs: []interface{}{[]string{"active"}},
		},
		{
			name: "in filter with multiple filters",
			params: QueryParams{
				Filters: []Filter{
					{Column: "user_id", Operator: OpEqual, Value: "123"},
					{Column: "status", Operator: OpIn, Value: []string{"queued", "running"}},
				},
			},
			expectedSQL:  `WHERE "user_id" = $1 AND "status" = ANY($2)`,
			expectedArgs: []interface{}{"123", []string{"queued", "running"}},
		},
		{
			name: "with order and limit",
			params: QueryParams{
				Order: []OrderBy{
					{Column: "created_at", Desc: true},
				},
				Limit: intPtr(10),
			},
			expectedSQL:  `ORDER BY "created_at" DESC LIMIT $1`,
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
	parser := NewQueryParser(testConfig())

	tests := []struct {
		name                 string
		query                string
		expectedSelect       []string
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
	parser := NewQueryParser(testConfig())

	tests := []struct {
		name            string
		query           string
		expectedGroupBy []string
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
		name        string
		params      QueryParams
		expectedSQL string
	}{
		{
			name: "aggregation only - count(*)",
			params: QueryParams{
				Aggregations: []Aggregation{
					{Function: AggCountAll, Column: "", Alias: ""},
				},
			},
			expectedSQL: `COUNT(*) AS "count"`,
		},
		{
			name: "aggregation only - sum",
			params: QueryParams{
				Aggregations: []Aggregation{
					{Function: AggSum, Column: "price", Alias: ""},
				},
			},
			expectedSQL: `SUM("price") AS "sum_price"`,
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
			expectedSQL: `COUNT("id") AS "count_id", SUM("price") AS "sum_price", AVG("rating") AS "avg_rating"`,
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
			expectedSQL: `"category", COUNT(*) AS "count", SUM("price") AS "total"`,
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
		name        string
		params      QueryParams
		expectedSQL string
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
			expectedSQL: ` GROUP BY "category"`,
		},
		{
			name: "multiple group by",
			params: QueryParams{
				GroupBy: []string{"category", "status", "region"},
			},
			expectedSQL: ` GROUP BY "category", "status", "region"`,
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
		name        string
		agg         Aggregation
		expectedSQL string
	}{
		{
			name:        "COUNT(*)",
			agg:         Aggregation{Function: AggCountAll, Column: "", Alias: ""},
			expectedSQL: `COUNT(*) AS "count"`,
		},
		{
			name:        "COUNT(column)",
			agg:         Aggregation{Function: AggCount, Column: "id", Alias: ""},
			expectedSQL: `COUNT("id") AS "count_id"`,
		},
		{
			name:        "SUM",
			agg:         Aggregation{Function: AggSum, Column: "price", Alias: ""},
			expectedSQL: `SUM("price") AS "sum_price"`,
		},
		{
			name:        "AVG",
			agg:         Aggregation{Function: AggAvg, Column: "rating", Alias: ""},
			expectedSQL: `AVG("rating") AS "avg_rating"`,
		},
		{
			name:        "MIN",
			agg:         Aggregation{Function: AggMin, Column: "price", Alias: ""},
			expectedSQL: `MIN("price") AS "min_price"`,
		},
		{
			name:        "MAX",
			agg:         Aggregation{Function: AggMax, Column: "price", Alias: ""},
			expectedSQL: `MAX("price") AS "max_price"`,
		},
		{
			name:        "custom alias",
			agg:         Aggregation{Function: AggSum, Column: "price", Alias: "total"},
			expectedSQL: `SUM("price") AS "total"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql := tt.agg.ToSQL()
			assert.Equal(t, tt.expectedSQL, sql)
		})
	}
}

func TestPaginationLimitEnforcement(t *testing.T) {
	tests := []struct {
		name           string
		config         *config.Config
		queryString    string
		expectedLimit  *int
		expectedOffset *int
		description    string
	}{
		{
			name: "Enforce max_page_size - cap requested limit",
			config: &config.Config{
				API: config.APIConfig{
					MaxPageSize:     1000,
					MaxTotalResults: -1,
					DefaultPageSize: -1,
				},
			},
			queryString:   "limit=5000",
			expectedLimit: intPtr(1000),
			description:   "Requested limit of 5000 should be capped to max_page_size of 1000",
		},
		{
			name: "Apply default_page_size when no limit specified",
			config: &config.Config{
				API: config.APIConfig{
					MaxPageSize:     10000,
					MaxTotalResults: -1,
					DefaultPageSize: 1000,
				},
			},
			queryString:   "",
			expectedLimit: intPtr(1000),
			description:   "No limit specified should apply default_page_size of 1000",
		},
		{
			name: "No default applied when default_page_size is -1",
			config: &config.Config{
				API: config.APIConfig{
					MaxPageSize:     10000,
					MaxTotalResults: -1,
					DefaultPageSize: -1,
				},
			},
			queryString:   "",
			expectedLimit: nil,
			description:   "When default_page_size is -1, no default limit should be applied",
		},
		{
			name: "Enforce max_total_results - cap limit based on offset",
			config: &config.Config{
				API: config.APIConfig{
					MaxPageSize:     5000,
					MaxTotalResults: 10000,
					DefaultPageSize: -1,
				},
			},
			queryString:    "offset=9500&limit=1000",
			expectedLimit:  intPtr(500),
			expectedOffset: intPtr(9500),
			description:    "Offset 9500 + limit 1000 exceeds max_total_results 10000, should cap limit to 500",
		},
		{
			name: "Enforce max_total_results - zero limit when offset exceeds max",
			config: &config.Config{
				API: config.APIConfig{
					MaxPageSize:     5000,
					MaxTotalResults: 10000,
					DefaultPageSize: -1,
				},
			},
			queryString:    "offset=10500&limit=1000",
			expectedLimit:  intPtr(0),
			expectedOffset: intPtr(10500),
			description:    "Offset 10500 exceeds max_total_results 10000, should cap limit to 0",
		},
		{
			name: "Allow unlimited when max_page_size is -1",
			config: &config.Config{
				API: config.APIConfig{
					MaxPageSize:     -1,
					MaxTotalResults: -1,
					DefaultPageSize: -1,
				},
			},
			queryString:   "limit=100000",
			expectedLimit: intPtr(100000),
			description:   "When max_page_size is -1, allow any limit",
		},
		{
			name: "Combine max_page_size and max_total_results enforcement",
			config: &config.Config{
				API: config.APIConfig{
					MaxPageSize:     1000,
					MaxTotalResults: 5000,
					DefaultPageSize: 500,
				},
			},
			queryString:    "offset=4500&limit=2000",
			expectedLimit:  intPtr(500),
			expectedOffset: intPtr(4500),
			description:    "Limit 2000 capped to max_page_size 1000, then further capped to 500 due to max_total_results",
		},
		{
			name: "Default limit respects max_total_results",
			config: &config.Config{
				API: config.APIConfig{
					MaxPageSize:     5000,
					MaxTotalResults: 10000,
					DefaultPageSize: 1000,
				},
			},
			queryString:    "offset=9800",
			expectedLimit:  intPtr(200),
			expectedOffset: intPtr(9800),
			description:    "Default limit 1000 applied, then capped to 200 due to max_total_results",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewQueryParser(tt.config)
			values, err := url.ParseQuery(tt.queryString)
			require.NoError(t, err)

			params, err := parser.Parse(values)
			require.NoError(t, err, tt.description)

			if tt.expectedLimit != nil {
				require.NotNil(t, params.Limit, "Expected limit to be set but got nil")
				assert.Equal(t, *tt.expectedLimit, *params.Limit, tt.description)
			} else {
				assert.Nil(t, params.Limit, "Expected limit to be nil but got a value")
			}

			if tt.expectedOffset != nil {
				require.NotNil(t, params.Offset, "Expected offset to be set but got nil")
				assert.Equal(t, *tt.expectedOffset, *params.Offset)
			}
		})
	}
}

func TestParseJSONBPath(t *testing.T) {
	tests := []struct {
		name     string
		column   string
		expected string
	}{
		{
			name:     "simple column",
			column:   "name",
			expected: `"name"`,
		},
		{
			name:     "json access single key",
			column:   "data->key",
			expected: `"data"->'key'`,
		},
		{
			name:     "text access single key",
			column:   "data->>key",
			expected: `"data"->>'key'`,
		},
		{
			name:     "chained json access",
			column:   "data->nested->value",
			expected: `"data"->'nested'->'value'`,
		},
		{
			name:     "mixed json and text access",
			column:   "data->nested->>value",
			expected: `"data"->'nested'->>'value'`,
		},
		{
			name:     "deep nesting",
			column:   "a->b->c->d->>e",
			expected: `"a"->'b'->'c'->'d'->>'e'`,
		},
		{
			name:     "array index",
			column:   "data->0",
			expected: `"data"->0`,
		},
		{
			name:     "array index with nested key",
			column:   "data->0->name",
			expected: `"data"->0->'name'`,
		},
		{
			name:     "array index with text extraction",
			column:   "data->0->>name",
			expected: `"data"->0->>'name'`,
		},
		{
			name:     "realistic geocode example",
			column:   "geocode->properties->>country",
			expected: `"geocode"->'properties'->>'country'`,
		},
		{
			name:     "metadata stats count",
			column:   "metadata->stats->>count",
			expected: `"metadata"->'stats'->>'count'`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseJSONBPath(tt.column)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFilterToSQLWithJSONBPath(t *testing.T) {
	tests := []struct {
		name        string
		filter      Filter
		expectedSQL string
		expectValue bool
	}{
		{
			name: "simple column equality",
			filter: Filter{
				Column:   "name",
				Operator: OpEqual,
				Value:    "John",
			},
			expectedSQL: `"name" = $1`,
			expectValue: true,
		},
		{
			name: "jsonb path equality",
			filter: Filter{
				Column:   "data->key",
				Operator: OpEqual,
				Value:    "value",
			},
			expectedSQL: `"data"->'key' = $1`,
			expectValue: true,
		},
		{
			name: "jsonb text extraction equality",
			filter: Filter{
				Column:   "data->>key",
				Operator: OpEqual,
				Value:    "value",
			},
			expectedSQL: `"data"->>'key' = $1`,
			expectValue: true,
		},
		{
			name: "nested jsonb path IS NULL",
			filter: Filter{
				Column:   "geocode->properties->>country",
				Operator: OpIs,
				Value:    nil,
			},
			expectedSQL: `"geocode"->'properties'->>'country' IS NULL`,
			expectValue: false,
		},
		{
			name: "jsonb text extraction greater than with numeric",
			filter: Filter{
				Column:   "metadata->stats->>count",
				Operator: OpGreaterThan,
				Value:    10,
			},
			expectedSQL: `("metadata"->'stats'->>'count')::numeric > $1`,
			expectValue: true,
		},
		{
			name: "jsonb text extraction less than with string number",
			filter: Filter{
				Column:   "data->>amount",
				Operator: OpLessThan,
				Value:    "100",
			},
			expectedSQL: `("data"->>'amount')::numeric < $1`,
			expectValue: true,
		},
		{
			name: "jsonb json access greater than (no cast)",
			filter: Filter{
				Column:   "data->count",
				Operator: OpGreaterThan,
				Value:    10,
			},
			expectedSQL: `"data"->'count' > $1`,
			expectValue: true,
		},
		{
			name: "jsonb IN operator",
			filter: Filter{
				Column:   "data->>status",
				Operator: OpIn,
				Value:    []string{"active", "pending"},
			},
			expectedSQL: `"data"->>'status' = ANY($1)`,
			expectValue: true,
		},
		{
			name: "jsonb LIKE operator",
			filter: Filter{
				Column:   "data->>email",
				Operator: OpLike,
				Value:    "%@example.com",
			},
			expectedSQL: `"data"->>'email' LIKE $1`,
			expectValue: true,
		},
		{
			name: "array index access",
			filter: Filter{
				Column:   "items->0->>name",
				Operator: OpEqual,
				Value:    "first",
			},
			expectedSQL: `"items"->0->>'name' = $1`,
			expectValue: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			argCounter := 1
			sql, value := filterToSQL(tt.filter, &argCounter)

			assert.Equal(t, tt.expectedSQL, sql)

			if tt.expectValue {
				assert.Equal(t, tt.filter.Value, value)
			} else {
				assert.Nil(t, value)
			}
		})
	}
}

func TestNeedsNumericCast(t *testing.T) {
	tests := []struct {
		name     string
		column   string
		value    interface{}
		expected bool
	}{
		{
			name:     "text extraction with int",
			column:   "data->>count",
			value:    10,
			expected: true,
		},
		{
			name:     "text extraction with float",
			column:   "data->>price",
			value:    19.99,
			expected: true,
		},
		{
			name:     "text extraction with string number",
			column:   "data->>count",
			value:    "10",
			expected: true,
		},
		{
			name:     "text extraction with non-numeric string",
			column:   "data->>name",
			value:    "John",
			expected: false,
		},
		{
			name:     "json access with int (no cast needed)",
			column:   "data->count",
			value:    10,
			expected: false,
		},
		{
			name:     "simple column with int (no cast)",
			column:   "count",
			value:    10,
			expected: false,
		},
		{
			name:     "nested text extraction with int",
			column:   "metadata->stats->>total",
			value:    100,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := needsNumericCast(tt.column, tt.value)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestQueryParser_NestedLogicalFilters(t *testing.T) {
	parser := NewQueryParser(testConfig())

	tests := []struct {
		name           string
		query          string
		expectedCount  int
		expectOrGroups bool
	}{
		{
			name:           "simple or filter",
			query:          "or=(name.eq.John,name.eq.Jane)",
			expectedCount:  2,
			expectOrGroups: false,
		},
		{
			name:           "nested or in and filter",
			query:          "and=(or(col.lt.10,col.gt.20),or(col.lt.30,col.gt.40))",
			expectedCount:  4,
			expectOrGroups: true,
		},
		{
			name:           "complex nested expression",
			query:          "and=(or(date.lt.2024-01-01,date.gt.2024-01-10),or(date.lt.2024-02-01,date.gt.2024-02-10),or(date.lt.2024-03-01,date.gt.2024-03-10))",
			expectedCount:  6,
			expectOrGroups: true,
		},
		{
			name:           "or filter with is.null",
			query:          "or=(name.is.null,name.eq.)",
			expectedCount:  2,
			expectOrGroups: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values, _ := url.ParseQuery(tt.query)
			params, err := parser.Parse(values)

			require.NoError(t, err)
			assert.Equal(t, tt.expectedCount, len(params.Filters))

			if tt.expectOrGroups {
				// Check that filters have OrGroupID set
				groupIDs := make(map[int]bool)
				for _, f := range params.Filters {
					if f.OrGroupID > 0 {
						groupIDs[f.OrGroupID] = true
					}
				}
				assert.Greater(t, len(groupIDs), 0, "expected OR groups to be assigned")
			}
		})
	}
}

func TestQueryParser_OrFilterIsNullValueParsing(t *testing.T) {
	parser := NewQueryParser(testConfig())

	// Test that is.null in OR filters gets properly parsed to nil, not string "null"
	values, _ := url.ParseQuery("or=(name.is.null,status.is.true,active.is.false)")
	params, err := parser.Parse(values)

	require.NoError(t, err)
	require.Equal(t, 3, len(params.Filters))

	// Find each filter and verify its value
	for _, f := range params.Filters {
		switch f.Column {
		case "name":
			assert.Equal(t, OpIs, f.Operator)
			assert.Nil(t, f.Value, "is.null should parse to nil, not string 'null'")
		case "status":
			assert.Equal(t, OpIs, f.Operator)
			assert.Equal(t, true, f.Value, "is.true should parse to bool true")
		case "active":
			assert.Equal(t, OpIs, f.Operator)
			assert.Equal(t, false, f.Value, "is.false should parse to bool false")
		}
	}

	// Verify SQL generation produces IS NULL, not IS $1
	argCounter := 1
	whereClause, args := params.buildWhereClause(&argCounter)
	assert.Contains(t, whereClause, "IS NULL")
	assert.Contains(t, whereClause, "IS $1") // for true
	assert.Contains(t, whereClause, "IS $2") // for false
	assert.Equal(t, 2, len(args), "should have 2 args (true, false), null should not be parameterized")
}

func TestQueryParser_ParseNestedFilters(t *testing.T) {
	parser := NewQueryParser(testConfig())

	tests := []struct {
		name     string
		value    string
		expected []string
	}{
		{
			name:     "simple comma separated",
			value:    "a.eq.1,b.eq.2",
			expected: []string{"a.eq.1", "b.eq.2"},
		},
		{
			name:     "nested parentheses",
			value:    "or(a.eq.1,b.eq.2),or(c.eq.3,d.eq.4)",
			expected: []string{"or(a.eq.1,b.eq.2)", "or(c.eq.3,d.eq.4)"},
		},
		{
			name:     "single nested expression",
			value:    "or(col.lt.10,col.gt.20)",
			expected: []string{"or(col.lt.10,col.gt.20)"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.parseNestedFilters(tt.value)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestQueryParams_BuildWhereClause_OrGroups(t *testing.T) {
	tests := []struct {
		name           string
		filters        []Filter
		expectedParts  []string
		unexpectedPart string
	}{
		{
			name: "separate OR groups",
			filters: []Filter{
				{Column: "col", Operator: OpLessThan, Value: "10", IsOr: true, OrGroupID: 1},
				{Column: "col", Operator: OpGreaterThan, Value: "20", IsOr: true, OrGroupID: 1},
				{Column: "col", Operator: OpLessThan, Value: "30", IsOr: true, OrGroupID: 2},
				{Column: "col", Operator: OpGreaterThan, Value: "40", IsOr: true, OrGroupID: 2},
			},
			expectedParts: []string{
				`("col" < $1 OR "col" > $2)`,
				`("col" < $3 OR "col" > $4)`,
				" AND ",
			},
			unexpectedPart: "col" + ` < $1 OR "col" > $2 OR "col" < $3`, // Should NOT group all together
		},
		{
			name: "mixed AND and OR groups",
			filters: []Filter{
				{Column: "status", Operator: OpEqual, Value: "active", IsOr: false},
				{Column: "col", Operator: OpLessThan, Value: "10", IsOr: true, OrGroupID: 1},
				{Column: "col", Operator: OpGreaterThan, Value: "20", IsOr: true, OrGroupID: 1},
			},
			expectedParts: []string{
				`"status" = $1`,
				`("col" < $2 OR "col" > $3)`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := &QueryParams{Filters: tt.filters}
			argCounter := 1
			whereClause, _ := params.buildWhereClause(&argCounter)

			for _, expected := range tt.expectedParts {
				assert.Contains(t, whereClause, expected)
			}

			if tt.unexpectedPart != "" {
				assert.NotContains(t, whereClause, tt.unexpectedPart)
			}
		})
	}
}

func TestFormatVectorValue(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{
			name:     "string with brackets",
			input:    "[0.1,0.2,0.3]",
			expected: "[0.1,0.2,0.3]",
		},
		{
			name:     "string without brackets",
			input:    "0.1,0.2,0.3",
			expected: "[0.1,0.2,0.3]",
		},
		{
			name:     "float64 slice",
			input:    []float64{0.1, 0.2, 0.3},
			expected: "[0.1,0.2,0.3]",
		},
		{
			name:     "float32 slice",
			input:    []float32{0.1, 0.2, 0.3},
			expected: "[0.1,0.2,0.3]",
		},
		{
			name:     "interface slice with floats",
			input:    []interface{}{0.1, 0.2, 0.3},
			expected: "[0.1,0.2,0.3]",
		},
		{
			name:     "interface slice with ints",
			input:    []interface{}{1, 2, 3},
			expected: "[1,2,3]",
		},
		{
			name:     "empty slice",
			input:    []float64{},
			expected: "[]",
		},
		{
			name:     "string with leading bracket only",
			input:    "[0.1,0.2",
			expected: "[0.1,0.2]",
		},
		{
			name:     "string with trailing bracket only",
			input:    "0.1,0.2]",
			expected: "[0.1,0.2]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatVectorValue(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseSTDWithinValue(t *testing.T) {
	tests := []struct {
		name             string
		input            string
		expectedDistance float64
		expectedGeometry string
		expectError      bool
		errorContains    string
	}{
		{
			name:             "valid point with integer distance",
			input:            `1000,{"type":"Point","coordinates":[-122.4783,37.8199]}`,
			expectedDistance: 1000,
			expectedGeometry: `{"type":"Point","coordinates":[-122.4783,37.8199]}`,
			expectError:      false,
		},
		{
			name:             "valid point with float distance",
			input:            `1500.5,{"type":"Point","coordinates":[-122.4783,37.8199]}`,
			expectedDistance: 1500.5,
			expectedGeometry: `{"type":"Point","coordinates":[-122.4783,37.8199]}`,
			expectError:      false,
		},
		{
			name:             "valid polygon with distance",
			input:            `500,{"type":"Polygon","coordinates":[[[-122.5,37.7],[-122.5,37.85],[-122.35,37.85],[-122.35,37.7],[-122.5,37.7]]]}`,
			expectedDistance: 500,
			expectedGeometry: `{"type":"Polygon","coordinates":[[[-122.5,37.7],[-122.5,37.85],[-122.35,37.85],[-122.35,37.7],[-122.5,37.7]]]}`,
			expectError:      false,
		},
		{
			name:             "zero distance",
			input:            `0,{"type":"Point","coordinates":[-122.4783,37.8199]}`,
			expectedDistance: 0,
			expectedGeometry: `{"type":"Point","coordinates":[-122.4783,37.8199]}`,
			expectError:      false,
		},
		{
			name:          "negative distance",
			input:         `-100,{"type":"Point","coordinates":[-122.4783,37.8199]}`,
			expectError:   true,
			errorContains: "distance cannot be negative",
		},
		{
			name:          "missing distance",
			input:         `{"type":"Point","coordinates":[-122.4783,37.8199]}`,
			expectError:   true,
			errorContains: "st_dwithin value must be in format",
		},
		{
			name:          "invalid distance - not a number",
			input:         `abc,{"type":"Point","coordinates":[-122.4783,37.8199]}`,
			expectError:   true,
			errorContains: "invalid distance value",
		},
		{
			name:          "missing geometry",
			input:         `1000,`,
			expectError:   true,
			errorContains: "geometry must be a valid GeoJSON object",
		},
		{
			name:          "invalid geometry - not JSON",
			input:         `1000,not-json`,
			expectError:   true,
			errorContains: "geometry must be a valid GeoJSON object",
		},
		{
			name:          "empty input",
			input:         ``,
			expectError:   true,
			errorContains: "st_dwithin value must be in format",
		},
		{
			name:          "only comma",
			input:         `,`,
			expectError:   true,
			errorContains: "st_dwithin value must be in format",
		},
		{
			name:             "distance with spaces",
			input:            ` 1000 , {"type":"Point","coordinates":[-122.4783,37.8199]}`,
			expectedDistance: 1000,
			expectedGeometry: `{"type":"Point","coordinates":[-122.4783,37.8199]}`,
			expectError:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			distance, geometry, err := parseSTDWithinValue(tt.input)

			if tt.expectError {
				require.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedDistance, distance)
				assert.Equal(t, tt.expectedGeometry, geometry)
			}
		})
	}
}

func TestSTDWithinFilter(t *testing.T) {
	parser := NewQueryParser(testConfig())

	tests := []struct {
		name        string
		query       string
		expectSQL   string
		expectArgs  []interface{}
		expectError bool
	}{
		{
			name:       "st_dwithin with point",
			query:      `location.st_dwithin=1000,{"type":"Point","coordinates":[-122.4783,37.8199]}`,
			expectSQL:  `ST_DWithin("location", ST_GeomFromGeoJSON($1), $2)`,
			expectArgs: []interface{}{`{"type":"Point","coordinates":[-122.4783,37.8199]}`, float64(1000)},
		},
		{
			name:       "st_dwithin with polygon",
			query:      `geom.st_dwithin=500.5,{"type":"Polygon","coordinates":[[[-122.5,37.7],[-122.5,37.85],[-122.35,37.85],[-122.5,37.7]]]}`,
			expectSQL:  `ST_DWithin("geom", ST_GeomFromGeoJSON($1), $2)`,
			expectArgs: []interface{}{`{"type":"Polygon","coordinates":[[[-122.5,37.7],[-122.5,37.85],[-122.35,37.85],[-122.5,37.7]]]}`, float64(500.5)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values, err := url.ParseQuery(tt.query)
			require.NoError(t, err)

			params, err := parser.Parse(values)
			require.NoError(t, err)
			require.Len(t, params.Filters, 1)

			argCounter := 1
			sql, args := params.buildWhereClause(&argCounter)

			assert.Equal(t, tt.expectSQL, sql)
			assert.Equal(t, tt.expectArgs, args)
		})
	}
}
