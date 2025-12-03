package api

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/fluxbase-eu/fluxbase/internal/config"
	"github.com/rs/zerolog/log"
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

	// PostGIS spatial operators
	OpSTIntersects FilterOperator = "st_intersects" // ST_Intersects - geometries intersect
	OpSTContains   FilterOperator = "st_contains"   // ST_Contains - geometry A contains B
	OpSTWithin     FilterOperator = "st_within"     // ST_Within - geometry A is within B
	OpSTDWithin    FilterOperator = "st_dwithin"    // ST_DWithin - geometries within distance
	OpSTDistance   FilterOperator = "st_distance"   // ST_Distance - distance between geometries
	OpSTTouches    FilterOperator = "st_touches"    // ST_Touches - geometries touch
	OpSTCrosses    FilterOperator = "st_crosses"    // ST_Crosses - geometries cross
	OpSTOverlaps   FilterOperator = "st_overlaps"   // ST_Overlaps - geometries overlap
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
type QueryParser struct {
	config *config.Config
}

// NewQueryParser creates a new query parser
func NewQueryParser(cfg *config.Config) *QueryParser {
	return &QueryParser{
		config: cfg,
	}
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

			// Enforce max_page_size (unless it's -1 for unlimited)
			if qp.config.API.MaxPageSize > 0 && limit > qp.config.API.MaxPageSize {
				log.Debug().
					Int("requested", limit).
					Int("max", qp.config.API.MaxPageSize).
					Msg("Limit capped to max_page_size")
				limit = qp.config.API.MaxPageSize
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
			// Process ALL values for the same key to support range queries like:
			// ?recorded_at=gte.2025-01-01&recorded_at=lte.2025-12-31
			for _, val := range vals {
				if strings.Contains(key, ".") || strings.Contains(val, ".") || key == "or" || key == "and" {
					if err := qp.parseFilter(key, val, params); err != nil {
						return nil, fmt.Errorf("invalid filter parameter %s: %w", key, err)
					}
				}
			}
		}
	}

	// Apply default limit if none specified (unless default is -1)
	if params.Limit == nil && qp.config.API.DefaultPageSize > 0 {
		defaultLimit := qp.config.API.DefaultPageSize
		params.Limit = &defaultLimit
		log.Debug().
			Int("default", defaultLimit).
			Msg("Applied default_page_size")
	}

	// Validate total results limit (offset + limit <= max_total_results)
	if qp.config.API.MaxTotalResults > 0 {
		offset := 0
		if params.Offset != nil {
			offset = *params.Offset
		}

		limit := 0
		if params.Limit != nil {
			limit = *params.Limit
		}

		totalRows := offset + limit
		if totalRows > qp.config.API.MaxTotalResults {
			// Cap the limit so that offset + limit = max_total_results
			cappedLimit := qp.config.API.MaxTotalResults - offset
			if cappedLimit < 0 {
				cappedLimit = 0
			}

			log.Debug().
				Int("offset", offset).
				Int("requested_limit", limit).
				Int("capped_limit", cappedLimit).
				Int("max_total", qp.config.API.MaxTotalResults).
				Msg("Limit capped due to max_total_results")

			params.Limit = &cappedLimit
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

// parseJSONBPath parses a column name that may contain JSONB path operators
// and returns the properly formatted SQL expression.
// Examples:
//   - "name" -> "name" (simple column)
//   - "data->key" -> "data"->'key' (JSON access)
//   - "data->>key" -> "data"->>'key' (text access)
//   - "data->nested->>value" -> "data"->'nested'->>'value' (chained)
//   - "data->0->name" -> "data"->0->'name' (array index)
func parseJSONBPath(column string) string {
	// Check if column contains JSONB path operators
	if !strings.Contains(column, "->") {
		// Simple column name - quote it
		return fmt.Sprintf(`"%s"`, column)
	}

	// Split the path into segments, preserving ->> vs ->
	// We need to handle both -> (JSON) and ->> (text) operators
	var result strings.Builder
	remaining := column

	isFirst := true
	for len(remaining) > 0 {
		// Find the next operator (->> or ->)
		textOpIdx := strings.Index(remaining, "->>")
		jsonOpIdx := strings.Index(remaining, "->")

		// Determine which operator comes first
		var opIdx int
		var opLen int
		var op string

		if textOpIdx >= 0 && (jsonOpIdx < 0 || textOpIdx <= jsonOpIdx) {
			opIdx = textOpIdx
			opLen = 3
			op = "->>"
		} else if jsonOpIdx >= 0 {
			opIdx = jsonOpIdx
			opLen = 2
			op = "->"
		} else {
			// No more operators - this is the last key
			key := remaining
			if isFirst {
				result.WriteString(fmt.Sprintf(`"%s"`, key))
			} else {
				result.WriteString(formatJSONKey(key))
			}
			break
		}

		// Extract the part before the operator
		part := remaining[:opIdx]
		if isFirst {
			// First part is the column name - quote it as identifier
			result.WriteString(fmt.Sprintf(`"%s"`, part))
			isFirst = false
		} else {
			// Subsequent parts are JSON keys
			result.WriteString(formatJSONKey(part))
		}

		// Add the operator
		result.WriteString(op)

		// Move past the operator
		remaining = remaining[opIdx+opLen:]
	}

	return result.String()
}

// formatJSONKey formats a JSON key for use in a JSONB path expression.
// Numeric keys are left unquoted (for array access), string keys are quoted.
func formatJSONKey(key string) string {
	// Check if it's a numeric key (array index)
	if _, err := strconv.Atoi(key); err == nil {
		return key
	}
	// String key - wrap in single quotes
	return fmt.Sprintf("'%s'", key)
}

// needsNumericCast checks if a JSONB path expression needs numeric casting
// for comparison operations. This is needed when:
// 1. The path ends with ->> (returns text)
// 2. The value is numeric
func needsNumericCast(column string, value interface{}) bool {
	// Check if path uses text extraction (->>)
	if !strings.Contains(column, "->>") {
		return false
	}

	// Check if value is numeric
	switch value.(type) {
	case int, int8, int16, int32, int64:
		return true
	case uint, uint8, uint16, uint32, uint64:
		return true
	case float32, float64:
		return true
	case string:
		// Try to parse as number
		if _, err := strconv.ParseFloat(value.(string), 64); err == nil {
			return true
		}
	}
	return false
}

// toSQL converts a filter to SQL condition
func (f *Filter) toSQL(argCounter *int) (string, interface{}) {
	// Parse JSONB path for proper SQL formatting
	colExpr := parseJSONBPath(f.Column)

	switch f.Operator {
	case OpEqual:
		sql := fmt.Sprintf("%s = $%d", colExpr, *argCounter)
		*argCounter++
		return sql, f.Value

	case OpNotEqual:
		sql := fmt.Sprintf("%s != $%d", colExpr, *argCounter)
		*argCounter++
		return sql, f.Value

	case OpGreaterThan:
		expr := colExpr
		if needsNumericCast(f.Column, f.Value) {
			expr = fmt.Sprintf("(%s)::numeric", colExpr)
		}
		sql := fmt.Sprintf("%s > $%d", expr, *argCounter)
		*argCounter++
		return sql, f.Value

	case OpGreaterOrEqual:
		expr := colExpr
		if needsNumericCast(f.Column, f.Value) {
			expr = fmt.Sprintf("(%s)::numeric", colExpr)
		}
		sql := fmt.Sprintf("%s >= $%d", expr, *argCounter)
		*argCounter++
		return sql, f.Value

	case OpLessThan:
		expr := colExpr
		if needsNumericCast(f.Column, f.Value) {
			expr = fmt.Sprintf("(%s)::numeric", colExpr)
		}
		sql := fmt.Sprintf("%s < $%d", expr, *argCounter)
		*argCounter++
		return sql, f.Value

	case OpLessOrEqual:
		expr := colExpr
		if needsNumericCast(f.Column, f.Value) {
			expr = fmt.Sprintf("(%s)::numeric", colExpr)
		}
		sql := fmt.Sprintf("%s <= $%d", expr, *argCounter)
		*argCounter++
		return sql, f.Value

	case OpLike:
		sql := fmt.Sprintf("%s LIKE $%d", colExpr, *argCounter)
		*argCounter++
		return sql, f.Value

	case OpILike:
		sql := fmt.Sprintf("%s ILIKE $%d", colExpr, *argCounter)
		*argCounter++
		return sql, f.Value

	case OpIn:
		// Use PostgreSQL's ANY() syntax to properly handle array parameters
		// This avoids the bug where IN ($2,$3) expects multiple args but we pass a single array
		sql := fmt.Sprintf("%s = ANY($%d)", colExpr, *argCounter)
		*argCounter++
		return sql, f.Value

	case OpIs:
		if f.Value == nil {
			return fmt.Sprintf("%s IS NULL", colExpr), nil
		}
		sql := fmt.Sprintf("%s IS $%d", colExpr, *argCounter)
		*argCounter++
		return sql, f.Value

	case OpContains:
		sql := fmt.Sprintf("%s @> $%d", colExpr, *argCounter)
		*argCounter++
		return sql, f.Value

	case OpContained:
		sql := fmt.Sprintf("%s <@ $%d", colExpr, *argCounter)
		*argCounter++
		return sql, f.Value

	case OpOverlap:
		sql := fmt.Sprintf("%s && $%d", colExpr, *argCounter)
		*argCounter++
		return sql, f.Value

	case OpTextSearch:
		sql := fmt.Sprintf("%s @@ plainto_tsquery($%d)", colExpr, *argCounter)
		*argCounter++
		return sql, f.Value

	case OpPhraseSearch:
		sql := fmt.Sprintf("%s @@ phraseto_tsquery($%d)", colExpr, *argCounter)
		*argCounter++
		return sql, f.Value

	case OpWebSearch:
		sql := fmt.Sprintf("%s @@ websearch_to_tsquery($%d)", colExpr, *argCounter)
		*argCounter++
		return sql, f.Value

	case OpNot:
		// NOT operator - negates the condition
		// Value format: "operator.value" (e.g., "eq.deleted" or "is.null")
		valueStr, ok := f.Value.(string)
		if !ok {
			return "", fmt.Errorf("NOT operator requires string value in format operator.value")
		}

		// Parse nested operator and value
		dotIndex := strings.Index(valueStr, ".")
		if dotIndex <= 0 {
			return "", fmt.Errorf("NOT operator value must be in format operator.value, got: %s", valueStr)
		}

		nestedOp := FilterOperator(valueStr[:dotIndex])
		nestedValue := valueStr[dotIndex+1:]

		// Parse the nested value based on nested operator
		var parsedValue interface{}
		switch nestedOp {
		case OpIn:
			// Parse array values: (1,2,3) or ["a","b","c"]
			trimmed := strings.Trim(nestedValue, "()[]")
			items := strings.Split(trimmed, ",")
			parsedValue = items
		case OpIs:
			switch nestedValue {
			case "null":
				parsedValue = nil
			case "true":
				parsedValue = true
			case "false":
				parsedValue = false
			default:
				parsedValue = nestedValue
			}
		default:
			parsedValue = nestedValue
		}

		// Create a filter with the nested operator
		nestedFilter := Filter{
			Column:   f.Column,
			Operator: nestedOp,
			Value:    parsedValue,
		}

		// Generate SQL for the nested filter
		nestedSQL, nestedArg := nestedFilter.toSQL(argCounter)

		// Wrap in NOT
		sql := fmt.Sprintf("NOT (%s)", nestedSQL)
		return sql, nestedArg

	case OpAdjacent:
		sql := fmt.Sprintf("%s << $%d", colExpr, *argCounter)
		*argCounter++
		return sql, f.Value

	case OpStrictlyLeft:
		sql := fmt.Sprintf("%s << $%d", colExpr, *argCounter)
		*argCounter++
		return sql, f.Value

	case OpStrictlyRight:
		sql := fmt.Sprintf("%s >> $%d", colExpr, *argCounter)
		*argCounter++
		return sql, f.Value

	case OpNotExtendRight:
		sql := fmt.Sprintf("%s &< $%d", colExpr, *argCounter)
		*argCounter++
		return sql, f.Value

	case OpNotExtendLeft:
		sql := fmt.Sprintf("%s &> $%d", colExpr, *argCounter)
		*argCounter++
		return sql, f.Value

	// PostGIS spatial operators
	case OpSTIntersects:
		sql := fmt.Sprintf("ST_Intersects(%s, ST_GeomFromGeoJSON($%d))", colExpr, *argCounter)
		*argCounter++
		return sql, f.Value

	case OpSTContains:
		sql := fmt.Sprintf("ST_Contains(%s, ST_GeomFromGeoJSON($%d))", colExpr, *argCounter)
		*argCounter++
		return sql, f.Value

	case OpSTWithin:
		sql := fmt.Sprintf("ST_Within(%s, ST_GeomFromGeoJSON($%d))", colExpr, *argCounter)
		*argCounter++
		return sql, f.Value

	case OpSTDWithin:
		// ST_DWithin expects: ST_DWithin(geom1, geom2, distance)
		// Value should be a map with "geometry" and "distance" fields
		// For now, we'll expect a GeoJSON string and use a default distance
		// TODO: Support distance parameter
		sql := fmt.Sprintf("ST_DWithin(%s, ST_GeomFromGeoJSON($%d), $%d)", colExpr, *argCounter, *argCounter+1)
		*argCounter += 2
		// For now, return placeholder - needs proper implementation with distance
		return sql, f.Value

	case OpSTDistance:
		sql := fmt.Sprintf("ST_Distance(%s, ST_GeomFromGeoJSON($%d))", colExpr, *argCounter)
		*argCounter++
		return sql, f.Value

	case OpSTTouches:
		sql := fmt.Sprintf("ST_Touches(%s, ST_GeomFromGeoJSON($%d))", colExpr, *argCounter)
		*argCounter++
		return sql, f.Value

	case OpSTCrosses:
		sql := fmt.Sprintf("ST_Crosses(%s, ST_GeomFromGeoJSON($%d))", colExpr, *argCounter)
		*argCounter++
		return sql, f.Value

	case OpSTOverlaps:
		sql := fmt.Sprintf("ST_Overlaps(%s, ST_GeomFromGeoJSON($%d))", colExpr, *argCounter)
		*argCounter++
		return sql, f.Value

	default:
		sql := fmt.Sprintf("%s = $%d", colExpr, *argCounter)
		*argCounter++
		return sql, f.Value
	}
}
