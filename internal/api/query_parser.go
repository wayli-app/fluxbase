package api

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/fluxbase-eu/fluxbase/internal/config"
	"github.com/fluxbase-eu/fluxbase/internal/query"
	"github.com/rs/zerolog/log"
)

// validIdentifierRegex validates SQL identifiers (column names, table names, etc.)
// Allows alphanumeric characters, underscores, and must start with a letter or underscore
var validIdentifierRegex = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// isValidIdentifier checks if a string is a valid SQL identifier
func isValidIdentifier(s string) bool {
	return validIdentifierRegex.MatchString(s)
}

// quoteIdentifier safely quotes an SQL identifier to prevent injection
// Returns empty string if the identifier is invalid
func quoteIdentifier(s string) string {
	if !isValidIdentifier(s) {
		return ""
	}
	// Escape any embedded double quotes and wrap in double quotes
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}

// QueryParams represents parsed query parameters for REST API
type QueryParams struct {
	Select         []string           // Fields to select
	Filters        []Filter           // WHERE conditions
	Order          []OrderBy          // ORDER BY clauses
	Limit          *int               // LIMIT clause
	Offset         *int               // OFFSET clause
	Cursor         *string            // Base64-encoded cursor for keyset pagination
	CursorColumn   *string            // Column to use for cursor (default: primary key)
	Embedded       []EmbeddedRelation // Relations to embed
	Count          CountType          // Count preference
	Aggregations   []Aggregation      // Aggregation functions
	GroupBy        []string           // GROUP BY columns
	orGroupCounter int                // Counter for assigning OR group IDs
}

// Filter is an alias for query.Filter for backward compatibility
type Filter = query.Filter

// FilterOperator is an alias for query.FilterOperator for backward compatibility
type FilterOperator = query.FilterOperator

// Re-export filter operator constants for backward compatibility
const (
	OpEqual          = query.OpEqual
	OpNotEqual       = query.OpNotEqual
	OpGreaterThan    = query.OpGreaterThan
	OpGreaterOrEqual = query.OpGreaterOrEqual
	OpLessThan       = query.OpLessThan
	OpLessOrEqual    = query.OpLessOrEqual
	OpLike           = query.OpLike
	OpILike          = query.OpILike
	OpIn             = query.OpIn
	OpNotIn          = query.OpNotIn
	OpIs             = query.OpIs
	OpIsNot          = query.OpIsNot
	OpContains       = query.OpContains
	OpContained      = query.OpContained
	OpContainedBy    = query.OpContainedBy
	OpOverlap        = query.OpOverlap
	OpOverlaps       = query.OpOverlaps
	OpTextSearch     = query.OpTextSearch
	OpPhraseSearch   = query.OpPhraseSearch
	OpWebSearch      = query.OpWebSearch
	OpNot            = query.OpNot
	OpAdjacent       = query.OpAdjacent
	OpStrictlyLeft   = query.OpStrictlyLeft
	OpStrictlyRight  = query.OpStrictlyRight
	OpNotExtendRight = query.OpNotExtendRight
	OpNotExtendLeft  = query.OpNotExtendLeft

	// PostGIS spatial operators
	OpSTIntersects = query.OpSTIntersects
	OpSTContains   = query.OpSTContains
	OpSTWithin     = query.OpSTWithin
	OpSTDWithin    = query.OpSTDWithin
	OpSTDistance   = query.OpSTDistance
	OpSTTouches    = query.OpSTTouches
	OpSTCrosses    = query.OpSTCrosses
	OpSTOverlaps   = query.OpSTOverlaps

	// pgvector similarity operators
	OpVectorL2     = query.OpVectorL2
	OpVectorCosine = query.OpVectorCosine
	OpVectorIP     = query.OpVectorIP
)

// OrderBy is an alias for query.OrderBy for backward compatibility
type OrderBy = query.OrderBy

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

// ParseOptions configures query parsing behavior
type ParseOptions struct {
	// BypassMaxTotalResults skips the max_total_results enforcement.
	// Use for admin/dashboard requests that should have unlimited access.
	BypassMaxTotalResults bool
}

// NewQueryParser creates a new query parser
func NewQueryParser(cfg *config.Config) *QueryParser {
	return &QueryParser{
		config: cfg,
	}
}

// Parse parses URL query parameters into QueryParams with default options
func (qp *QueryParser) Parse(values url.Values) (*QueryParams, error) {
	return qp.ParseWithOptions(values, ParseOptions{})
}

// ParseWithOptions parses URL query parameters into QueryParams with custom options
func (qp *QueryParser) ParseWithOptions(values url.Values, opts ParseOptions) (*QueryParams, error) {
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

		case "cursor":
			// Base64-encoded cursor for keyset pagination
			cursor := vals[0]
			if cursor != "" {
				params.Cursor = &cursor
			}

		case "cursor_column":
			// Column to use for cursor (must be a valid identifier)
			col := vals[0]
			if col != "" {
				if !isValidIdentifier(col) {
					return nil, fmt.Errorf("invalid cursor_column: must be a valid column name")
				}
				params.CursorColumn = &col
			}

		case "count":
			params.Count = CountType(vals[0])

		case "group_by":
			// Parse GROUP BY columns: group_by=category,status
			columns := strings.Split(vals[0], ",")
			for _, col := range columns {
				col = strings.TrimSpace(col)
				if col != "" {
					// Validate column name to prevent SQL injection
					if !isValidIdentifier(col) {
						return nil, fmt.Errorf("invalid group_by column name: %s", col)
					}
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
	// Skip this check if BypassMaxTotalResults is set (e.g., for admin users)
	if !opts.BypassMaxTotalResults && qp.config.API.MaxTotalResults > 0 {
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
	// Vector ordering format: order=embedding.vec_cos.[0.1,0.2,...].asc
	orders := splitOrderParams(value)

	for _, order := range orders {
		order = strings.TrimSpace(order)
		if order == "" {
			continue
		}

		// Check for vector ordering format: column.vec_op.[vector].direction
		// The vector is enclosed in brackets, so we need special parsing
		if vectorOrder, ok := qp.parseVectorOrder(order); ok {
			params.Order = append(params.Order, vectorOrder)
			continue
		}

		// Standard ordering: column.direction.nulls
		parts := strings.Split(order, ".")
		if len(parts) < 2 {
			return fmt.Errorf("invalid order format: %s", order)
		}

		// Validate column name to prevent SQL injection
		colName := parts[0]
		if !isValidIdentifier(colName) {
			return fmt.Errorf("invalid order column name: %s", colName)
		}

		orderBy := OrderBy{
			Column: colName,
			Desc:   parts[1] == "desc",
		}

		// Check for nulls first/last
		if len(parts) > 2 {
			switch parts[2] {
			case "nullsfirst":
				orderBy.Nulls = "first"
			case "nullslast":
				orderBy.Nulls = "last"
			}
		}

		params.Order = append(params.Order, orderBy)
	}

	return nil
}

// splitOrderParams splits order parameters by comma, respecting brackets
func splitOrderParams(value string) []string {
	var orders []string
	var current strings.Builder
	bracketDepth := 0

	for _, ch := range value {
		switch ch {
		case '[':
			bracketDepth++
			current.WriteRune(ch)
		case ']':
			bracketDepth--
			current.WriteRune(ch)
		case ',':
			if bracketDepth == 0 {
				if s := strings.TrimSpace(current.String()); s != "" {
					orders = append(orders, s)
				}
				current.Reset()
			} else {
				current.WriteRune(ch)
			}
		default:
			current.WriteRune(ch)
		}
	}

	if s := strings.TrimSpace(current.String()); s != "" {
		orders = append(orders, s)
	}

	return orders
}

// parseVectorOrder parses vector ordering format: column.vec_op.[vector].direction
// Example: embedding.vec_cos.[0.1,0.2,0.3].asc
func (qp *QueryParser) parseVectorOrder(order string) (OrderBy, bool) {
	// Look for vector operator pattern
	vectorOps := []string{".vec_l2.", ".vec_cos.", ".vec_ip."}
	var opIdx = -1
	var opStr string

	for _, op := range vectorOps {
		if idx := strings.Index(order, op); idx > 0 {
			opIdx = idx
			opStr = strings.Trim(op, ".")
			break
		}
	}

	if opIdx < 0 {
		return OrderBy{}, false
	}

	// Extract column name
	colName := order[:opIdx]
	if !isValidIdentifier(colName) {
		return OrderBy{}, false
	}

	// Extract the rest after the operator
	remainder := order[opIdx+len(opStr)+2:] // +2 for the dots

	// Find the vector value in brackets
	bracketStart := strings.Index(remainder, "[")
	bracketEnd := strings.LastIndex(remainder, "]")

	if bracketStart < 0 || bracketEnd < bracketStart {
		return OrderBy{}, false
	}

	vectorStr := remainder[bracketStart : bracketEnd+1]

	// Get direction if present (after the closing bracket)
	var desc bool
	afterVector := remainder[bracketEnd+1:]
	if strings.Contains(afterVector, ".desc") {
		desc = true
	}
	// Default is ASC (ascending) for distance-based ordering (lower = more similar)

	return OrderBy{
		Column:      colName,
		Desc:        desc,
		VectorOp:    FilterOperator(opStr),
		VectorValue: vectorStr,
	}, true
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

// parseLogicalFilter parses or/and grouped filters with support for nested expressions
// Supports formats like:
//   - or=(name.eq.John,age.gt.30)
//   - and=(or(col.lt.min1,col.gt.max1),or(col.lt.min2,col.gt.max2))
func (qp *QueryParser) parseLogicalFilter(value string, params *QueryParams, isOr bool) error {
	// Parse format: or=(name.eq.John,age.gt.30)
	// Only remove one pair of outer parentheses (not all leading/trailing parens)
	if strings.HasPrefix(value, "(") && strings.HasSuffix(value, ")") {
		value = value[1 : len(value)-1]
	}

	// Use parentheses-aware splitting to handle nested expressions
	filters, err := qp.parseNestedFilters(value)
	if err != nil {
		return err
	}

	for _, filter := range filters {
		filter = strings.TrimSpace(filter)
		if filter == "" {
			continue
		}

		// Check for nested or() expression
		if strings.HasPrefix(filter, "or(") && strings.HasSuffix(filter, ")") {
			// Nested OR expression - parse recursively with new group ID
			innerValue := strings.TrimPrefix(filter, "or(")
			innerValue = strings.TrimSuffix(innerValue, ")")
			if err := qp.parseNestedOrGroup(innerValue, params); err != nil {
				return err
			}
			continue
		}

		// Check for nested and() expression
		if strings.HasPrefix(filter, "and(") && strings.HasSuffix(filter, ")") {
			// Nested AND expression - parse recursively
			innerValue := strings.TrimPrefix(filter, "and(")
			innerValue = strings.TrimSuffix(innerValue, ")")
			if err := qp.parseLogicalFilter(innerValue, params, false); err != nil {
				return err
			}
			continue
		}

		// Regular filter: column.operator.value
		parts := strings.SplitN(filter, ".", 3)
		if len(parts) != 3 {
			return fmt.Errorf("invalid filter format in logical group: %s", filter)
		}

		column := parts[0]
		operator := FilterOperator(parts[1])
		rawValue := parts[2]

		// Parse value based on operator (same logic as regular filter parsing)
		var parsedValue interface{}
		switch operator {
		case OpIn:
			// Parse array values: (1,2,3) or ["a","b","c"]
			parsedValue = qp.parseArrayValue(rawValue)
		case OpIs:
			// Parse null/true/false
			switch rawValue {
			case "null":
				parsedValue = nil
			case "true":
				parsedValue = true
			case "false":
				parsedValue = false
			default:
				parsedValue = rawValue
			}
		default:
			parsedValue = rawValue
		}

		params.Filters = append(params.Filters, Filter{
			Column:   column,
			Operator: operator,
			Value:    parsedValue,
			IsOr:     isOr,
		})
	}

	return nil
}

// parseNestedOrGroup parses an OR group and assigns a unique group ID to all filters
func (qp *QueryParser) parseNestedOrGroup(value string, params *QueryParams) error {
	// Increment group counter for this OR group
	params.orGroupCounter++
	groupID := params.orGroupCounter

	// Split by comma (respecting parentheses)
	filters, err := qp.parseNestedFilters(value)
	if err != nil {
		return err
	}

	for _, filter := range filters {
		filter = strings.TrimSpace(filter)
		if filter == "" {
			continue
		}

		// Parse each filter: column.operator.value
		parts := strings.SplitN(filter, ".", 3)
		if len(parts) != 3 {
			return fmt.Errorf("invalid filter format in OR group: %s", filter)
		}

		column := parts[0]
		operator := FilterOperator(parts[1])
		rawValue := parts[2]

		// Parse value based on operator (same logic as regular filter parsing)
		var parsedValue interface{}
		switch operator {
		case OpIn:
			// Parse array values: (1,2,3) or ["a","b","c"]
			parsedValue = qp.parseArrayValue(rawValue)
		case OpIs:
			// Parse null/true/false
			switch rawValue {
			case "null":
				parsedValue = nil
			case "true":
				parsedValue = true
			case "false":
				parsedValue = false
			default:
				parsedValue = rawValue
			}
		default:
			parsedValue = rawValue
		}

		params.Filters = append(params.Filters, Filter{
			Column:    column,
			Operator:  operator,
			Value:     parsedValue,
			IsOr:      true,
			OrGroupID: groupID,
		})
	}

	return nil
}

// parseNestedFilters splits a filter string by commas while respecting parentheses nesting
func (qp *QueryParser) parseNestedFilters(value string) ([]string, error) {
	var filters []string
	var current strings.Builder
	depth := 0

	for _, ch := range value {
		switch ch {
		case '(':
			depth++
			current.WriteRune(ch)
		case ')':
			depth--
			current.WriteRune(ch)
			if depth < 0 {
				return nil, fmt.Errorf("unbalanced parentheses in filter expression")
			}
		case ',':
			if depth == 0 {
				if s := strings.TrimSpace(current.String()); s != "" {
					filters = append(filters, s)
				}
				current.Reset()
			} else {
				current.WriteRune(ch)
			}
		default:
			current.WriteRune(ch)
		}
	}

	if depth != 0 {
		return nil, fmt.Errorf("unbalanced parentheses in filter expression")
	}

	if s := strings.TrimSpace(current.String()); s != "" {
		filters = append(filters, s)
	}

	return filters, nil
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

	// Add regular select fields - quote identifiers for safety
	if len(params.Select) > 0 {
		for _, field := range params.Select {
			// Skip empty fields
			if field == "" {
				continue
			}
			// Check if it's already a complex expression (contains operators or functions)
			// In which case, assume it's been validated elsewhere
			if strings.ContainsAny(field, "()+-*/ ") {
				parts = append(parts, field)
			} else {
				// Simple column name - quote it for safety
				parts = append(parts, quoteIdentifier(field))
			}
		}
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
	// Quote all identifiers for safety
	quotedCols := make([]string, len(params.GroupBy))
	for i, col := range params.GroupBy {
		quotedCols[i] = quoteIdentifier(col)
	}
	return " GROUP BY " + strings.Join(quotedCols, ", ")
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

	// Validate alias to prevent injection
	if !isValidIdentifier(alias) {
		alias = "result"
	}

	var funcSQL string
	switch agg.Function {
	case AggCountAll:
		funcSQL = "COUNT(*)"
	case AggCount:
		// Validate column name to prevent injection
		quotedCol := quoteIdentifier(agg.Column)
		if quotedCol == "" {
			return "NULL AS " + quoteIdentifier(alias)
		}
		funcSQL = fmt.Sprintf("COUNT(%s)", quotedCol)
	case AggSum:
		quotedCol := quoteIdentifier(agg.Column)
		if quotedCol == "" {
			return "NULL AS " + quoteIdentifier(alias)
		}
		funcSQL = fmt.Sprintf("SUM(%s)", quotedCol)
	case AggAvg:
		quotedCol := quoteIdentifier(agg.Column)
		if quotedCol == "" {
			return "NULL AS " + quoteIdentifier(alias)
		}
		funcSQL = fmt.Sprintf("AVG(%s)", quotedCol)
	case AggMin:
		quotedCol := quoteIdentifier(agg.Column)
		if quotedCol == "" {
			return "NULL AS " + quoteIdentifier(alias)
		}
		funcSQL = fmt.Sprintf("MIN(%s)", quotedCol)
	case AggMax:
		quotedCol := quoteIdentifier(agg.Column)
		if quotedCol == "" {
			return "NULL AS " + quoteIdentifier(alias)
		}
		funcSQL = fmt.Sprintf("MAX(%s)", quotedCol)
	default:
		funcSQL = "NULL"
	}

	return fmt.Sprintf("%s AS %s", funcSQL, quoteIdentifier(alias))
}

// buildWhereClause builds the WHERE clause from filters
func (params *QueryParams) buildWhereClause(argCounter *int) (string, []interface{}) {
	var args []interface{}

	// Build SQL for each filter and collect arguments
	type filterSQL struct {
		condition string
		filter    Filter
	}
	filterSQLs := make([]filterSQL, len(params.Filters))

	for i, filter := range params.Filters {
		condition, arg := filterToSQL(filter, argCounter)
		filterSQLs[i] = filterSQL{condition: condition, filter: filter}
		if arg != nil {
			// Handle multi-argument operators (e.g., ST_DWithin returns []interface{})
			if argSlice, ok := arg.([]interface{}); ok {
				args = append(args, argSlice...)
			} else {
				args = append(args, arg)
			}
		}
	}

	// Group OR conditions by OrGroupID
	// Filters with OrGroupID > 0 are grouped together by their ID
	// Filters with OrGroupID == 0 and IsOr == true use legacy consecutive grouping
	// Filters with IsOr == false are ANDed directly
	orGroups := make(map[int][]string) // OrGroupID -> conditions
	var legacyOrGroup []string         // For backward compat with IsOr=true, OrGroupID=0
	var finalConditions []string
	lastWasLegacyOr := false

	for _, fs := range filterSQLs {
		if fs.filter.OrGroupID > 0 {
			// New-style OR group with explicit ID
			orGroups[fs.filter.OrGroupID] = append(orGroups[fs.filter.OrGroupID], fs.condition)
		} else if fs.filter.IsOr {
			// Legacy OR (consecutive grouping for backward compatibility)
			legacyOrGroup = append(legacyOrGroup, fs.condition)
			lastWasLegacyOr = true
		} else {
			// AND condition - flush any pending legacy OR group first
			if lastWasLegacyOr && len(legacyOrGroup) > 0 {
				finalConditions = append(finalConditions, "("+strings.Join(legacyOrGroup, " OR ")+")")
				legacyOrGroup = nil
			}
			lastWasLegacyOr = false
			finalConditions = append(finalConditions, fs.condition)
		}
	}

	// Flush remaining legacy OR group
	if len(legacyOrGroup) > 0 {
		finalConditions = append(finalConditions, "("+strings.Join(legacyOrGroup, " OR ")+")")
	}

	// Add new-style OR groups (each group becomes a parenthesized OR expression)
	// Sort by group ID for deterministic output
	groupIDs := make([]int, 0, len(orGroups))
	for id := range orGroups {
		groupIDs = append(groupIDs, id)
	}
	// Simple insertion sort for small number of groups
	for i := 1; i < len(groupIDs); i++ {
		for j := i; j > 0 && groupIDs[j] < groupIDs[j-1]; j-- {
			groupIDs[j], groupIDs[j-1] = groupIDs[j-1], groupIDs[j]
		}
	}

	for _, id := range groupIDs {
		conditions := orGroups[id]
		if len(conditions) == 1 {
			finalConditions = append(finalConditions, conditions[0])
		} else {
			finalConditions = append(finalConditions, "("+strings.Join(conditions, " OR ")+")")
		}
	}

	return strings.Join(finalConditions, " AND "), args
}

// buildOrderClause builds the ORDER BY clause
func (params *QueryParams) buildOrderClause() string {
	var orderParts []string

	for _, order := range params.Order {
		// Quote column name to prevent SQL injection
		quotedCol := quoteIdentifier(order.Column)
		if quotedCol == "" {
			continue // Skip invalid column names
		}

		var part string

		// Check if this is a vector ordering
		if order.VectorOp != "" && order.VectorValue != nil {
			// Vector similarity ordering: column <=> '[0.1,0.2,...]'::vector
			var opSQL string
			switch order.VectorOp {
			case OpVectorL2:
				opSQL = "<->"
			case OpVectorCosine:
				opSQL = "<=>"
			case OpVectorIP:
				opSQL = "<#>"
			default:
				continue // Skip unknown vector operators
			}

			vectorVal := formatVectorValue(order.VectorValue)
			part = fmt.Sprintf("%s %s '%s'::vector", quotedCol, opSQL, vectorVal)
		} else {
			// Standard column ordering
			part = quotedCol
		}

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
	// String key - wrap in single quotes with proper escaping
	// Escape single quotes by doubling them to prevent SQL injection
	escaped := strings.ReplaceAll(key, "'", "''")
	return fmt.Sprintf("'%s'", escaped)
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
	switch v := value.(type) {
	case int, int8, int16, int32, int64:
		return true
	case uint, uint8, uint16, uint32, uint64:
		return true
	case float32, float64:
		return true
	case string:
		// Try to parse as number
		if _, err := strconv.ParseFloat(v, 64); err == nil {
			return true
		}
	}
	return false
}

// filterToSQL converts a filter to SQL condition
func filterToSQL(f Filter, argCounter *int) (string, interface{}) {
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
		nestedSQL, nestedArg := filterToSQL(nestedFilter, argCounter)

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
		// Value format: "distance,{geojson}" (e.g., "1000,{"type":"Point","coordinates":[-122.4,37.8]}")
		valueStr, ok := f.Value.(string)
		if !ok {
			return "", nil
		}

		distance, geometry, err := parseSTDWithinValue(valueStr)
		if err != nil {
			return "", nil
		}

		sql := fmt.Sprintf("ST_DWithin(%s, ST_GeomFromGeoJSON($%d), $%d)", colExpr, *argCounter, *argCounter+1)
		*argCounter += 2
		// Return a slice with both arguments (geometry first, then distance)
		return sql, []interface{}{geometry, distance}

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

	// pgvector similarity operators
	// These operators calculate distance - lower values = more similar
	// Used for vector search with ORDER BY to find most similar vectors
	case OpVectorL2:
		// L2/Euclidean distance: <->
		// Value should be a vector array formatted as '[0.1,0.2,...]'
		vectorVal := formatVectorValue(f.Value)
		sql := fmt.Sprintf("%s <-> $%d::vector", colExpr, *argCounter)
		*argCounter++
		return sql, vectorVal

	case OpVectorCosine:
		// Cosine distance: <=>
		// Value should be a vector array formatted as '[0.1,0.2,...]'
		vectorVal := formatVectorValue(f.Value)
		sql := fmt.Sprintf("%s <=> $%d::vector", colExpr, *argCounter)
		*argCounter++
		return sql, vectorVal

	case OpVectorIP:
		// Negative inner product: <#>
		// Value should be a vector array formatted as '[0.1,0.2,...]'
		vectorVal := formatVectorValue(f.Value)
		sql := fmt.Sprintf("%s <#> $%d::vector", colExpr, *argCounter)
		*argCounter++
		return sql, vectorVal

	default:
		sql := fmt.Sprintf("%s = $%d", colExpr, *argCounter)
		*argCounter++
		return sql, f.Value
	}
}

// parseSTDWithinValue parses a compound value for ST_DWithin operator
// Format: distance,{geojson} (e.g., "1000,{"type":"Point","coordinates":[-122.4,37.8]}")
// Returns the distance (float64) and the GeoJSON geometry (string)
func parseSTDWithinValue(value string) (float64, string, error) {
	// Find the first comma that's not inside braces/brackets
	braceDepth := 0
	commaIdx := -1
outer:
	for i, ch := range value {
		switch ch {
		case '{', '[':
			braceDepth++
		case '}', ']':
			braceDepth--
		case ',':
			if braceDepth == 0 {
				commaIdx = i
				break outer
			}
		}
	}

	if commaIdx <= 0 {
		return 0, "", fmt.Errorf("st_dwithin value must be in format: distance,{geojson}")
	}

	distanceStr := strings.TrimSpace(value[:commaIdx])
	geometry := strings.TrimSpace(value[commaIdx+1:])

	distance, err := strconv.ParseFloat(distanceStr, 64)
	if err != nil {
		return 0, "", fmt.Errorf("invalid distance value: %w", err)
	}

	if distance < 0 {
		return 0, "", fmt.Errorf("distance cannot be negative")
	}

	// Basic validation that geometry looks like JSON
	if !strings.HasPrefix(geometry, "{") || !strings.HasSuffix(geometry, "}") {
		return 0, "", fmt.Errorf("geometry must be a valid GeoJSON object")
	}

	return distance, geometry, nil
}

// formatVectorValue converts a vector value to PostgreSQL vector literal format
// Accepts []float32, []float64, []interface{}, or string (already formatted)
func formatVectorValue(value interface{}) string {
	switch v := value.(type) {
	case string:
		// Already a string - could be formatted like "[0.1,0.2]" or "0.1,0.2"
		// Clean it up to ensure proper format
		s := strings.TrimSpace(v)
		if !strings.HasPrefix(s, "[") {
			s = "[" + s
		}
		if !strings.HasSuffix(s, "]") {
			s = s + "]"
		}
		return s

	case []float32:
		parts := make([]string, len(v))
		for i, f := range v {
			parts[i] = strconv.FormatFloat(float64(f), 'f', -1, 32)
		}
		return "[" + strings.Join(parts, ",") + "]"

	case []float64:
		parts := make([]string, len(v))
		for i, f := range v {
			parts[i] = strconv.FormatFloat(f, 'f', -1, 64)
		}
		return "[" + strings.Join(parts, ",") + "]"

	case []interface{}:
		parts := make([]string, len(v))
		for i, item := range v {
			switch num := item.(type) {
			case float64:
				parts[i] = strconv.FormatFloat(num, 'f', -1, 64)
			case float32:
				parts[i] = strconv.FormatFloat(float64(num), 'f', -1, 32)
			case int:
				parts[i] = strconv.Itoa(num)
			case int64:
				parts[i] = strconv.FormatInt(num, 10)
			default:
				parts[i] = fmt.Sprintf("%v", num)
			}
		}
		return "[" + strings.Join(parts, ",") + "]"

	default:
		// Try to convert to string
		return fmt.Sprintf("%v", v)
	}
}
