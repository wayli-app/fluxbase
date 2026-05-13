package api

import (
	"fmt"
	"strconv"
	"strings"
)

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
		orderClause, orderArgs := params.buildOrderClause(&argCounter)
		if orderClause != "" {
			sqlParts = append(sqlParts, "ORDER BY "+orderClause)
			args = append(args, orderArgs...)
		}
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
			// In which case, validate it against SQL injection patterns
			if strings.ContainsAny(field, "()+-*/ ") {
				upper := strings.ToUpper(field)
				for _, kw := range []string{"INSERT", "UPDATE", "DELETE", "DROP", "CREATE", "ALTER", "EXECUTE", "GRANT", "REVOKE", "EXEC", "UNION"} {
					if strings.Contains(upper, kw) {
						return ""
					}
				}
				if strings.Contains(upper, "SELECT") {
					return ""
				}
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
		switch {
		case fs.filter.OrGroupID > 0:
			// New-style OR group with explicit ID
			orGroups[fs.filter.OrGroupID] = append(orGroups[fs.filter.OrGroupID], fs.condition)
		case fs.filter.IsOr:
			// Legacy OR (consecutive grouping for backward compatibility)
			legacyOrGroup = append(legacyOrGroup, fs.condition)
			lastWasLegacyOr = true
		default:
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

// buildOrderClause builds the ORDER BY clause with parameterized vector values
// Returns the clause string and any arguments that need to be passed to the query
func (params *QueryParams) buildOrderClause(argCounter *int) (string, []interface{}) {
	var orderParts []string
	var args []interface{}

	for _, order := range params.Order {
		// Quote column name to prevent SQL injection
		quotedCol := quoteIdentifier(order.Column)
		if quotedCol == "" {
			continue // Skip invalid column names
		}

		var part string

		// Check if this is a vector ordering
		if order.VectorOp != "" && order.VectorValue != nil {
			// Vector similarity ordering: column <=> $N::vector
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

			// Validate and sanitize vector value before parameterization
			vectorVal, err := validateAndFormatVector(order.VectorValue)
			if err != nil {
				continue // Skip invalid vector values
			}

			// Use parameterized query for vector values
			part = fmt.Sprintf("%s %s $%d::vector", quotedCol, opSQL, *argCounter)
			args = append(args, vectorVal)
			*argCounter++
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

	return strings.Join(orderParts, ", "), args
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
		// SECURITY: OpIs values are validated during parsing to only accept "true", "false", or "null".
		// The parsed Go bool value is passed via parameterized query to prevent SQL injection.
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

// validateAndFormatVector validates a vector value and returns it in PostgreSQL format
// Returns an error if the vector contains potentially dangerous content
func validateAndFormatVector(value interface{}) (string, error) {
	vectorStr := formatVectorValue(value)

	// Validate that the vector only contains valid characters
	// Allowed: digits, decimal point, comma, space, brackets, minus sign
	for i, ch := range vectorStr {
		switch {
		case ch >= '0' && ch <= '9':
			// Digits are always safe
		case ch == '.' || ch == ',' || ch == ' ' || ch == '[' || ch == ']':
			// Structural characters are safe
		case ch == '-' && i > 0 && vectorStr[i-1] != '-':
			// Minus sign is safe if not doubled (no SQL comment)
		case ch == 'e' || ch == 'E':
			// Scientific notation is safe
		default:
			// Any other character is potentially dangerous
			return "", fmt.Errorf("invalid character in vector value: %q at position %d", ch, i)
		}
	}

	// Additional check: ensure no SQL metacharacters
	if strings.Contains(vectorStr, "'") || strings.Contains(vectorStr, ";") || strings.Contains(vectorStr, "--") {
		return "", fmt.Errorf("vector value contains forbidden SQL characters")
	}

	return vectorStr, nil
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

		//nolint:gocritic // Conditions check different indices, not switch-compatible
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
				fmt.Fprintf(&result, `"%s"`, key)
			} else {
				result.WriteString(formatJSONKey(key))
			}
			break
		}

		// Extract the part before the operator
		part := remaining[:opIdx]
		if isFirst {
			// First part is the column name - quote it as identifier
			fmt.Fprintf(&result, `"%s"`, part)
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
			s += "]"
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
