package api

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// QueryParams represents parsed query parameters for REST API
type QueryParams struct {
	Select       []string           // Fields to select
	Filters      []Filter           // WHERE conditions
	Order        []OrderBy          // ORDER BY clauses
	Limit        *int               // LIMIT clause
	Offset       *int               // OFFSET clause
	Embedded     []EmbeddedRelation // Relations to embed
	Count        CountType          // Count preference
	Aggregations []Aggregation      // Aggregation functions
	GroupBy      []string           // GROUP BY columns
}

// Filter represents a WHERE condition
type Filter struct {
	Column   string
	Operator FilterOperator
	Value    interface{}
	IsOr     bool // OR instead of AND
}

// FilterOperator represents comparison operators
type FilterOperator string

const (
	OpEqual          FilterOperator = "eq"
	OpNotEqual       FilterOperator = "neq"
	OpGreaterThan    FilterOperator = "gt"
	OpGreaterOrEqual FilterOperator = "gte"
	OpLessThan       FilterOperator = "lt"
	OpLessOrEqual    FilterOperator = "lte"
	OpLike           FilterOperator = "like"
	OpILike          FilterOperator = "ilike"
	OpIn             FilterOperator = "in"
	OpIs             FilterOperator = "is"
	OpContains       FilterOperator = "cs"    // contains (array/jsonb) @>
	OpContained      FilterOperator = "cd"    // contained by (array/jsonb) <@
	OpOverlap        FilterOperator = "ov"    // overlap (array) &&
	OpTextSearch     FilterOperator = "fts"   // full text search
	OpPhraseSearch   FilterOperator = "plfts" // phrase search
	OpWebSearch      FilterOperator = "wfts"  // web search
	OpNot            FilterOperator = "not"   // negation
	OpAdjacent       FilterOperator = "adj"   // adjacent range <<
	OpStrictlyLeft   FilterOperator = "sl"    // strictly left of <<
	OpStrictlyRight  FilterOperator = "sr"    // strictly right of >>
	OpNotExtendRight FilterOperator = "nxr"   // does not extend to right &<
	OpNotExtendLeft  FilterOperator = "nxl"   // does not extend to left &>
)

// OrderBy represents an ORDER BY clause
type OrderBy struct {
	Column string
	Desc   bool
	Nulls  string // "first" or "last"
}

// EmbeddedRelation represents a relation to embed
type EmbeddedRelation struct {
	Name    string   // Relation name
	Select  []string // Fields to select from relation
	Filters []Filter // Filters for the relation
}

// CountType represents row count preferences
type CountType string

const (
	CountNone      CountType = "none"
	CountExact     CountType = "exact"
	CountPlanned   CountType = "planned"
	CountEstimated CountType = "estimated"
)

// Aggregation represents an aggregation function
type Aggregation struct {
	Function AggregateFunction
	Column   string
	Alias    string // Optional alias for the result
}

// AggregateFunction represents aggregation functions
type AggregateFunction string

const (
	AggCount    AggregateFunction = "count"
	AggSum      AggregateFunction = "sum"
	AggAvg      AggregateFunction = "avg"
	AggMin      AggregateFunction = "min"
	AggMax      AggregateFunction = "max"
	AggCountAll AggregateFunction = "count(*)"
)

// QueryParser parses PostgREST-compatible query parameters
type QueryParser struct{}

// NewQueryParser creates a new query parser
func NewQueryParser() *QueryParser {
	return &QueryParser{}
}

// Parse parses URL query parameters into QueryParams
func (qp *QueryParser) Parse(values url.Values) (*QueryParams, error) {
	params := &QueryParams{
		Filters: []Filter{},
		Order:   []OrderBy{},
	}

	// Parse each parameter type
	for key, vals := range values {
		switch key {
		case "select":
			if err := qp.parseSelect(vals[0], params); err != nil {
				return nil, fmt.Errorf("invalid select parameter: %w", err)
			}

		case "order":
			if err := qp.parseOrder(vals[0], params); err != nil {
				return nil, fmt.Errorf("invalid order parameter: %w", err)
			}

		case "limit":
			limit, err := strconv.Atoi(vals[0])
			if err != nil {
				return nil, fmt.Errorf("invalid limit parameter: %w", err)
			}
			params.Limit = &limit

		case "offset":
			offset, err := strconv.Atoi(vals[0])
			if err != nil {
				return nil, fmt.Errorf("invalid offset parameter: %w", err)
			}
			params.Offset = &offset

		case "count":
			params.Count = CountType(vals[0])

		case "group_by":
			// Parse GROUP BY columns: group_by=category,status
			columns := strings.Split(vals[0], ",")
			for _, col := range columns {
				col = strings.TrimSpace(col)
				if col != "" {
					params.GroupBy = append(params.GroupBy, col)
				}
			}

		default:
			// Check if it's a filter parameter
			// PostgREST format: column=operator.value (dot in value)
			// Old format: column.operator=value (dot in key)
			if strings.Contains(key, ".") || strings.Contains(vals[0], ".") || key == "or" || key == "and" {
				if err := qp.parseFilter(key, vals[0], params); err != nil {
					return nil, fmt.Errorf("invalid filter parameter %s: %w", key, err)
				}
			}
		}
	}

	return params, nil
}

// parseSelect parses the select parameter
func (qp *QueryParser) parseSelect(value string, params *QueryParams) error {
	// Parse format: select=id,name,posts(id,title,author(name))
	// Or with aggregations: select=category,count(*),sum(price),avg(rating)
	fields, embedded := qp.parseSelectFields(value)

	// Separate regular fields from aggregations
	regularFields := []string{}
	for _, field := range fields {
		if agg := qp.parseAggregation(field); agg != nil {
			params.Aggregations = append(params.Aggregations, *agg)
		} else {
			regularFields = append(regularFields, field)
		}
	}

	params.Select = regularFields

	for name, subSelect := range embedded {
		params.Embedded = append(params.Embedded, EmbeddedRelation{
			Name:   name,
			Select: subSelect,
		})
	}

	return nil
}

// parseSelectFields parses select fields and embedded relations
func (qp *QueryParser) parseSelectFields(value string) ([]string, map[string][]string) {
	fields := []string{}
	embedded := make(map[string][]string)

	// Known aggregation function names
	aggFuncs := map[string]bool{
		"count": true,
		"sum":   true,
		"avg":   true,
		"min":   true,
		"max":   true,
	}

	// Simple parser for nested parentheses
	var current strings.Builder
	var relationName string
	var depth int
	var inRelation bool
	var isAggregation bool

	for i := 0; i < len(value); i++ {
		ch := value[i]

		switch ch {
		case '(':
			if depth == 0 {
				relationName = strings.TrimSpace(current.String())
				// Check if this is an aggregation function
				isAggregation = aggFuncs[strings.ToLower(relationName)]
				if !isAggregation {
					// It's a relation, not an aggregation
					current.Reset()
					inRelation = true
				} else {
					// It's an aggregation function, keep building the field string
					current.WriteByte(ch)
				}
			} else {
				current.WriteByte(ch)
			}
			depth++

		case ')':
			depth--
			if depth == 0 && inRelation && !isAggregation {
				// End of relation fields
				subFields := strings.Split(current.String(), ",")
				for j := range subFields {
					subFields[j] = strings.TrimSpace(subFields[j])
				}
				embedded[relationName] = subFields
				current.Reset()
				inRelation = false
			} else if depth == 0 && isAggregation {
				// End of aggregation function
				current.WriteByte(ch)
				isAggregation = false
			} else if depth > 0 {
				current.WriteByte(ch)
			}

		case ',':
			if depth == 0 {
				if field := strings.TrimSpace(current.String()); field != "" {
					fields = append(fields, field)
				}
				current.Reset()
			} else {
				current.WriteByte(ch)
			}

		default:
			current.WriteByte(ch)
		}
	}

	// Add the last field
	if field := strings.TrimSpace(current.String()); field != "" {
		fields = append(fields, field)
	}

	return fields, embedded
}

// parseAggregation parses aggregation functions from a select field
// Examples: count(*), sum(price), avg(rating), count(id), min(created_at), max(updated_at)
func (qp *QueryParser) parseAggregation(field string) *Aggregation {
	field = strings.TrimSpace(field)

	// Check for aggregation function pattern: function(column) or function(*)
	funcEnd := strings.Index(field, "(")
	if funcEnd == -1 {
		return nil // Not an aggregation
	}

	funcName := strings.ToLower(strings.TrimSpace(field[:funcEnd]))
	remainder := field[funcEnd+1:]

	// Find closing parenthesis
	parenEnd := strings.Index(remainder, ")")
	if parenEnd == -1 {
		return nil // Malformed
	}

	column := strings.TrimSpace(remainder[:parenEnd])

	// Map function name to AggregateFunction
	var aggFunc AggregateFunction
	switch funcName {
	case "count":
		if column == "*" {
			aggFunc = AggCountAll
			column = "" // count(*) doesn't need a column
		} else {
			aggFunc = AggCount
		}
	case "sum":
		aggFunc = AggSum
	case "avg":
		aggFunc = AggAvg
	case "min":
		aggFunc = AggMin
	case "max":
		aggFunc = AggMax
	default:
		return nil // Unknown aggregation function
	}

	return &Aggregation{
		Function: aggFunc,
		Column:   column,
		Alias:    "", // Will be generated if needed
	}
}

// parseOrder parses the order parameter
func (qp *QueryParser) parseOrder(value string, params *QueryParams) error {
	// Parse format: order=name.asc,created_at.desc.nullslast
	orders := strings.Split(value, ",")

	for _, order := range orders {
		parts := strings.Split(strings.TrimSpace(order), ".")
		if len(parts) < 2 {
			return fmt.Errorf("invalid order format: %s", order)
		}

		orderBy := OrderBy{
			Column: parts[0],
			Desc:   parts[1] == "desc",
		}

		// Check for nulls first/last
		if len(parts) > 2 {
			if parts[2] == "nullsfirst" {
				orderBy.Nulls = "first"
			} else if parts[2] == "nullslast" {
				orderBy.Nulls = "last"
			}
		}

		params.Order = append(params.Order, orderBy)
	}

	return nil
}

// parseFilter parses filter parameters
func (qp *QueryParser) parseFilter(key, value string, params *QueryParams) error {
	// Handle logical operators
	if key == "or" {
		return qp.parseLogicalFilter(value, params, true)
	}
	if key == "and" {
		return qp.parseLogicalFilter(value, params, false)
	}

	// Check for classic format first: column.operator=value
	// This takes precedence over PostgREST format
	if strings.Contains(key, ".") {
		parts := strings.SplitN(key, ".", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid filter format: %s", key)
		}

		column := parts[0]
		operator := FilterOperator(parts[1])

		// Parse value based on operator
		var filterValue interface{}
		switch operator {
		case OpIn:
			// Parse array values: (1,2,3) or ["a","b","c"]
			filterValue = qp.parseArrayValue(value)
		case OpIs:
			// Parse null/true/false
			switch value {
			case "null":
				filterValue = nil
			case "true":
				filterValue = true
			case "false":
				filterValue = false
			default:
				filterValue = value
			}
		default:
			filterValue = value
		}

		params.Filters = append(params.Filters, Filter{
			Column:   column,
			Operator: operator,
			Value:    filterValue,
			IsOr:     false,
		})

		return nil
	}

	// Try PostgREST format: column=operator.value
	// Split value by first dot to extract operator
	dotIndex := strings.Index(value, ".")
	if dotIndex > 0 {
		// PostgREST format: column=operator.value
		column := key
		operatorStr := value[:dotIndex]
		filterValue := value[dotIndex+1:]

		operator := FilterOperator(operatorStr)

		// Parse value based on operator
		var parsedValue interface{}
		switch operator {
		case OpIn:
			// Parse array values: (1,2,3) or ["a","b","c"]
			parsedValue = qp.parseArrayValue(filterValue)
		case OpIs:
			// Parse null/true/false
			switch filterValue {
			case "null":
				parsedValue = nil
			case "true":
				parsedValue = true
			case "false":
				parsedValue = false
			default:
				parsedValue = filterValue
			}
		default:
			parsedValue = filterValue
		}

		params.Filters = append(params.Filters, Filter{
			Column:   column,
			Operator: operator,
			Value:    parsedValue,
			IsOr:     false,
		})

		return nil
	}

	// If neither format matched, return an error
	return fmt.Errorf("invalid filter format: %s", key)
}

// parseLogicalFilter parses or/and grouped filters
func (qp *QueryParser) parseLogicalFilter(value string, params *QueryParams, isOr bool) error {
	// Parse format: or=(name.eq.John,age.gt.30)
	value = strings.Trim(value, "()")
	filters := strings.Split(value, ",")

	for _, filter := range filters {
		// Parse each filter: column.operator.value
		parts := strings.SplitN(filter, ".", 3)
		if len(parts) != 3 {
			return fmt.Errorf("invalid filter format in logical group: %s", filter)
		}

		params.Filters = append(params.Filters, Filter{
			Column:   parts[0],
			Operator: FilterOperator(parts[1]),
			Value:    parts[2],
			IsOr:     isOr,
		})
	}

	return nil
}

// parseArrayValue parses array values from string
func (qp *QueryParser) parseArrayValue(value string) []string {
	// Remove parentheses or brackets
	value = strings.Trim(value, "()[]")

	// Split by comma
	items := strings.Split(value, ",")
	result := make([]string, len(items))

	for i, item := range items {
		// Remove quotes if present
		result[i] = strings.Trim(strings.TrimSpace(item), "\"'")
	}

	return result
}

// ToSQL converts QueryParams to SQL WHERE, ORDER BY, LIMIT, OFFSET clauses
func (params *QueryParams) ToSQL(tableName string) (string, []interface{}) {
	var sqlParts []string
	var args []interface{}
	argCounter := 1

	// Build WHERE clause
	if len(params.Filters) > 0 {
		whereClause, whereArgs := params.buildWhereClause(&argCounter)
		if whereClause != "" {
			sqlParts = append(sqlParts, "WHERE "+whereClause)
			args = append(args, whereArgs...)
		}
	}

	// Build ORDER BY clause
	if len(params.Order) > 0 {
		orderClause := params.buildOrderClause()
		sqlParts = append(sqlParts, "ORDER BY "+orderClause)
	}

	// Build LIMIT clause
	if params.Limit != nil {
		sqlParts = append(sqlParts, fmt.Sprintf("LIMIT $%d", argCounter))
		args = append(args, *params.Limit)
		argCounter++
	}

	// Build OFFSET clause
	if params.Offset != nil {
		sqlParts = append(sqlParts, fmt.Sprintf("OFFSET $%d", argCounter))
		args = append(args, *params.Offset)
		argCounter++
	}

	return strings.Join(sqlParts, " "), args
}

// BuildSelectClause builds the SELECT clause, including aggregations
func (params *QueryParams) BuildSelectClause(tableName string) string {
	var parts []string

	// Add regular select fields
	if len(params.Select) > 0 {
		parts = append(parts, params.Select...)
	} else if len(params.Aggregations) == 0 && len(params.GroupBy) == 0 {
		// Default to * if no select, aggregations, or group by
		parts = append(parts, "*")
	}

	// Add aggregation functions
	for _, agg := range params.Aggregations {
		aggSQL := agg.ToSQL()
		parts = append(parts, aggSQL)
	}

	// If we have only aggregations (no GROUP BY columns), select only aggregations
	if len(params.Select) == 0 && len(params.Aggregations) > 0 && len(params.GroupBy) == 0 {
		return strings.Join(parts[len(parts)-len(params.Aggregations):], ", ")
	}

	return strings.Join(parts, ", ")
}

// BuildGroupByClause builds the GROUP BY clause
func (params *QueryParams) BuildGroupByClause() string {
	if len(params.GroupBy) == 0 {
		return ""
	}
	return " GROUP BY " + strings.Join(params.GroupBy, ", ")
}

// ToSQL converts an Aggregation to SQL
func (agg *Aggregation) ToSQL() string {
	alias := agg.Alias
	if alias == "" {
		// Generate default alias
		if agg.Function == AggCountAll {
			alias = "count"
		} else {
			alias = string(agg.Function) + "_" + agg.Column
		}
	}

	var funcSQL string
	switch agg.Function {
	case AggCountAll:
		funcSQL = "COUNT(*)"
	case AggCount:
		funcSQL = fmt.Sprintf("COUNT(%s)", agg.Column)
	case AggSum:
		funcSQL = fmt.Sprintf("SUM(%s)", agg.Column)
	case AggAvg:
		funcSQL = fmt.Sprintf("AVG(%s)", agg.Column)
	case AggMin:
		funcSQL = fmt.Sprintf("MIN(%s)", agg.Column)
	case AggMax:
		funcSQL = fmt.Sprintf("MAX(%s)", agg.Column)
	default:
		funcSQL = "NULL"
	}

	return fmt.Sprintf("%s AS %s", funcSQL, alias)
}

// buildWhereClause builds the WHERE clause from filters
func (params *QueryParams) buildWhereClause(argCounter *int) (string, []interface{}) {
	var conditions []string
	var args []interface{}

	for _, filter := range params.Filters {
		condition, arg := filter.toSQL(argCounter)
		conditions = append(conditions, condition)
		if arg != nil {
			args = append(args, arg)
		}
	}

	// Group OR conditions
	var finalConditions []string
	var orGroup []string

	for i, condition := range conditions {
		if params.Filters[i].IsOr {
			orGroup = append(orGroup, condition)
		} else {
			if len(orGroup) > 0 {
				finalConditions = append(finalConditions, "("+strings.Join(orGroup, " OR ")+")")
				orGroup = []string{}
			}
			finalConditions = append(finalConditions, condition)
		}
	}

	if len(orGroup) > 0 {
		finalConditions = append(finalConditions, "("+strings.Join(orGroup, " OR ")+")")
	}

	return strings.Join(finalConditions, " AND "), args
}

// buildOrderClause builds the ORDER BY clause
func (params *QueryParams) buildOrderClause() string {
	var orderParts []string

	for _, order := range params.Order {
		part := order.Column
		if order.Desc {
			part += " DESC"
		} else {
			part += " ASC"
		}

		if order.Nulls != "" {
			part += " NULLS " + strings.ToUpper(order.Nulls)
		}

		orderParts = append(orderParts, part)
	}

	return strings.Join(orderParts, ", ")
}

// toSQL converts a filter to SQL condition
func (f *Filter) toSQL(argCounter *int) (string, interface{}) {
	switch f.Operator {
	case OpEqual:
		sql := fmt.Sprintf("%s = $%d", f.Column, *argCounter)
		*argCounter++
		return sql, f.Value

	case OpNotEqual:
		sql := fmt.Sprintf("%s != $%d", f.Column, *argCounter)
		*argCounter++
		return sql, f.Value

	case OpGreaterThan:
		sql := fmt.Sprintf("%s > $%d", f.Column, *argCounter)
		*argCounter++
		return sql, f.Value

	case OpGreaterOrEqual:
		sql := fmt.Sprintf("%s >= $%d", f.Column, *argCounter)
		*argCounter++
		return sql, f.Value

	case OpLessThan:
		sql := fmt.Sprintf("%s < $%d", f.Column, *argCounter)
		*argCounter++
		return sql, f.Value

	case OpLessOrEqual:
		sql := fmt.Sprintf("%s <= $%d", f.Column, *argCounter)
		*argCounter++
		return sql, f.Value

	case OpLike:
		sql := fmt.Sprintf("%s LIKE $%d", f.Column, *argCounter)
		*argCounter++
		return sql, f.Value

	case OpILike:
		sql := fmt.Sprintf("%s ILIKE $%d", f.Column, *argCounter)
		*argCounter++
		return sql, f.Value

	case OpIn:
		if arr, ok := f.Value.([]string); ok {
			placeholders := make([]string, len(arr))
			for i := range arr {
				placeholders[i] = fmt.Sprintf("$%d", *argCounter)
				*argCounter++
			}
			sql := fmt.Sprintf("%s IN (%s)", f.Column, strings.Join(placeholders, ","))
			return sql, arr
		}
		return fmt.Sprintf("%s IN ($%d)", f.Column, *argCounter), f.Value

	case OpIs:
		if f.Value == nil {
			return fmt.Sprintf("%s IS NULL", f.Column), nil
		}
		sql := fmt.Sprintf("%s IS $%d", f.Column, *argCounter)
		*argCounter++
		return sql, f.Value

	case OpContains:
		sql := fmt.Sprintf("%s @> $%d", f.Column, *argCounter)
		*argCounter++
		return sql, f.Value

	case OpContained:
		sql := fmt.Sprintf("%s <@ $%d", f.Column, *argCounter)
		*argCounter++
		return sql, f.Value

	case OpOverlap:
		sql := fmt.Sprintf("%s && $%d", f.Column, *argCounter)
		*argCounter++
		return sql, f.Value

	case OpTextSearch:
		sql := fmt.Sprintf("%s @@ plainto_tsquery($%d)", f.Column, *argCounter)
		*argCounter++
		return sql, f.Value

	case OpPhraseSearch:
		sql := fmt.Sprintf("%s @@ phraseto_tsquery($%d)", f.Column, *argCounter)
		*argCounter++
		return sql, f.Value

	case OpWebSearch:
		sql := fmt.Sprintf("%s @@ websearch_to_tsquery($%d)", f.Column, *argCounter)
		*argCounter++
		return sql, f.Value

	case OpNot:
		// NOT operator - negates the condition
		sql := fmt.Sprintf("NOT (%s = $%d)", f.Column, *argCounter)
		*argCounter++
		return sql, f.Value

	case OpAdjacent:
		sql := fmt.Sprintf("%s << $%d", f.Column, *argCounter)
		*argCounter++
		return sql, f.Value

	case OpStrictlyLeft:
		sql := fmt.Sprintf("%s << $%d", f.Column, *argCounter)
		*argCounter++
		return sql, f.Value

	case OpStrictlyRight:
		sql := fmt.Sprintf("%s >> $%d", f.Column, *argCounter)
		*argCounter++
		return sql, f.Value

	case OpNotExtendRight:
		sql := fmt.Sprintf("%s &< $%d", f.Column, *argCounter)
		*argCounter++
		return sql, f.Value

	case OpNotExtendLeft:
		sql := fmt.Sprintf("%s &> $%d", f.Column, *argCounter)
		*argCounter++
		return sql, f.Value

	default:
		sql := fmt.Sprintf("%s = $%d", f.Column, *argCounter)
		*argCounter++
		return sql, f.Value
	}
}
