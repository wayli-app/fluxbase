package api

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
)

// CursorData represents decoded cursor pagination data
type CursorData struct {
	Column string      `json:"c"` // Column name (short key for smaller cursor)
	Value  interface{} `json:"v"` // Last value
	Desc   bool        `json:"d"` // True if descending order
}

// EncodeCursor creates a base64-encoded cursor from the given data
func EncodeCursor(column string, value interface{}, desc bool) string {
	data := CursorData{Column: column, Value: value, Desc: desc}
	jsonBytes, _ := json.Marshal(data)
	return base64.URLEncoding.EncodeToString(jsonBytes)
}

// DecodeCursor decodes a base64-encoded cursor
func DecodeCursor(cursor string) (*CursorData, error) {
	jsonBytes, err := base64.URLEncoding.DecodeString(cursor)
	if err != nil {
		return nil, fmt.Errorf("invalid cursor encoding: %w", err)
	}
	var data CursorData
	if err := json.Unmarshal(jsonBytes, &data); err != nil {
		return nil, fmt.Errorf("invalid cursor format: %w", err)
	}
	if data.Column == "" {
		return nil, fmt.Errorf("cursor missing column")
	}
	return &data, nil
}

// QueryBuilder provides a fluent interface for building SQL queries.
// It separates query construction from execution, enabling unit testing
// of query generation without database access.
type QueryBuilder struct {
	schema       string
	table        string
	columns      []string
	filters      []Filter
	orderBy      []OrderBy
	limit        *int
	offset       *int
	groupBy      []string
	returning    []string
	argCounter   int
	cursorData   *CursorData // Cursor for keyset pagination
	cursorColumn string      // Column override for cursor
}

// NewQueryBuilder creates a new QueryBuilder for the given schema and table.
func NewQueryBuilder(schema, table string) *QueryBuilder {
	return &QueryBuilder{
		schema:     schema,
		table:      table,
		argCounter: 1,
	}
}

// WithColumns sets the columns to select.
func (qb *QueryBuilder) WithColumns(columns []string) *QueryBuilder {
	qb.columns = columns
	return qb
}

// WithFilters sets the WHERE conditions.
func (qb *QueryBuilder) WithFilters(filters []Filter) *QueryBuilder {
	qb.filters = filters
	return qb
}

// WithOrder sets the ORDER BY clauses.
func (qb *QueryBuilder) WithOrder(order []OrderBy) *QueryBuilder {
	qb.orderBy = order
	return qb
}

// WithLimit sets the LIMIT clause.
func (qb *QueryBuilder) WithLimit(limit int) *QueryBuilder {
	qb.limit = &limit
	return qb
}

// WithOffset sets the OFFSET clause.
func (qb *QueryBuilder) WithOffset(offset int) *QueryBuilder {
	qb.offset = &offset
	return qb
}

// WithGroupBy sets the GROUP BY columns.
func (qb *QueryBuilder) WithGroupBy(columns []string) *QueryBuilder {
	qb.groupBy = columns
	return qb
}

// WithReturning sets the RETURNING clause columns.
func (qb *QueryBuilder) WithReturning(columns []string) *QueryBuilder {
	qb.returning = columns
	return qb
}

// WithCursor sets cursor pagination parameters.
// The cursor is a base64-encoded string containing the last row's value.
// cursorColumn overrides the column in the cursor (optional).
func (qb *QueryBuilder) WithCursor(cursor string, cursorColumn string) error {
	if cursor == "" {
		return nil
	}
	data, err := DecodeCursor(cursor)
	if err != nil {
		return err
	}
	qb.cursorData = data
	if cursorColumn != "" {
		qb.cursorColumn = cursorColumn
	}
	return nil
}

// BuildSelect builds a SELECT query and returns the SQL string and arguments.
func (qb *QueryBuilder) BuildSelect() (string, []interface{}) {
	// Build SELECT clause
	selectClause := "*"
	if len(qb.columns) > 0 {
		quotedCols := make([]string, 0, len(qb.columns))
		for _, col := range qb.columns {
			if quoted := quoteIdentifier(col); quoted != "" {
				quotedCols = append(quotedCols, quoted)
			}
		}
		if len(quotedCols) > 0 {
			selectClause = strings.Join(quotedCols, ", ")
		}
	}

	// Build FROM clause
	query := fmt.Sprintf("SELECT %s FROM %s.%s",
		selectClause,
		quoteIdentifier(qb.schema),
		quoteIdentifier(qb.table))

	var args []interface{}
	var whereClauses []string

	// Build WHERE clause from filters
	if len(qb.filters) > 0 {
		whereClause, whereArgs := qb.buildWhereClause()
		if whereClause != "" {
			whereClauses = append(whereClauses, whereClause)
			args = append(args, whereArgs...)
		}
	}

	// Add cursor condition for keyset pagination
	if qb.cursorData != nil {
		cursorClause, cursorArg := qb.buildCursorCondition()
		if cursorClause != "" {
			whereClauses = append(whereClauses, cursorClause)
			args = append(args, cursorArg)
		}
	}

	// Combine WHERE clauses
	if len(whereClauses) > 0 {
		query += " WHERE " + strings.Join(whereClauses, " AND ")
	}

	// Build GROUP BY clause
	if len(qb.groupBy) > 0 {
		groupByClause := qb.buildGroupByClause()
		if groupByClause != "" {
			query += groupByClause
		}
	}

	// Build ORDER BY clause
	if len(qb.orderBy) > 0 {
		orderClause := qb.buildOrderClause()
		if orderClause != "" {
			query += " ORDER BY " + orderClause
		}
	}

	// Build LIMIT clause
	if qb.limit != nil {
		query += fmt.Sprintf(" LIMIT %d", *qb.limit)
	}

	// Build OFFSET clause
	if qb.offset != nil {
		query += fmt.Sprintf(" OFFSET %d", *qb.offset)
	}

	return query, args
}

// BuildCount builds a COUNT query and returns the SQL string and arguments.
func (qb *QueryBuilder) BuildCount() (string, []interface{}) {
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s.%s",
		quoteIdentifier(qb.schema),
		quoteIdentifier(qb.table))

	var args []interface{}

	// Build WHERE clause
	if len(qb.filters) > 0 {
		whereClause, whereArgs := qb.buildWhereClause()
		if whereClause != "" {
			query += " WHERE " + whereClause
			args = append(args, whereArgs...)
		}
	}

	return query, args
}

// BuildInsert builds an INSERT query and returns the SQL string, arguments,
// and column order (for value mapping).
func (qb *QueryBuilder) BuildInsert(data map[string]interface{}) (string, []interface{}) {
	if len(data) == 0 {
		return "", nil
	}

	var columns []string
	var placeholders []string
	var args []interface{}

	for col, val := range data {
		quoted := quoteIdentifier(col)
		if quoted == "" {
			continue
		}
		columns = append(columns, quoted)
		placeholders = append(placeholders, fmt.Sprintf("$%d", qb.argCounter))
		args = append(args, val)
		qb.argCounter++
	}

	if len(columns) == 0 {
		return "", nil
	}

	query := fmt.Sprintf("INSERT INTO %s.%s (%s) VALUES (%s)",
		quoteIdentifier(qb.schema),
		quoteIdentifier(qb.table),
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "))

	// Add RETURNING clause
	if len(qb.returning) > 0 {
		quotedRet := make([]string, 0, len(qb.returning))
		for _, col := range qb.returning {
			if quoted := quoteIdentifier(col); quoted != "" {
				quotedRet = append(quotedRet, quoted)
			}
		}
		if len(quotedRet) > 0 {
			query += " RETURNING " + strings.Join(quotedRet, ", ")
		}
	}

	return query, args
}

// BuildUpdate builds an UPDATE query and returns the SQL string and arguments.
func (qb *QueryBuilder) BuildUpdate(data map[string]interface{}) (string, []interface{}) {
	if len(data) == 0 {
		return "", nil
	}

	var setClauses []string
	var args []interface{}

	for col, val := range data {
		quoted := quoteIdentifier(col)
		if quoted == "" {
			continue
		}
		setClauses = append(setClauses, fmt.Sprintf("%s = $%d", quoted, qb.argCounter))
		args = append(args, val)
		qb.argCounter++
	}

	if len(setClauses) == 0 {
		return "", nil
	}

	query := fmt.Sprintf("UPDATE %s.%s SET %s",
		quoteIdentifier(qb.schema),
		quoteIdentifier(qb.table),
		strings.Join(setClauses, ", "))

	// Build WHERE clause
	if len(qb.filters) > 0 {
		whereClause, whereArgs := qb.buildWhereClause()
		if whereClause != "" {
			query += " WHERE " + whereClause
			args = append(args, whereArgs...)
		}
	}

	// Add RETURNING clause
	if len(qb.returning) > 0 {
		quotedRet := make([]string, 0, len(qb.returning))
		for _, col := range qb.returning {
			if quoted := quoteIdentifier(col); quoted != "" {
				quotedRet = append(quotedRet, quoted)
			}
		}
		if len(quotedRet) > 0 {
			query += " RETURNING " + strings.Join(quotedRet, ", ")
		}
	}

	return query, args
}

// BuildDelete builds a DELETE query and returns the SQL string and arguments.
func (qb *QueryBuilder) BuildDelete() (string, []interface{}) {
	query := fmt.Sprintf("DELETE FROM %s.%s",
		quoteIdentifier(qb.schema),
		quoteIdentifier(qb.table))

	var args []interface{}

	// Build WHERE clause
	if len(qb.filters) > 0 {
		whereClause, whereArgs := qb.buildWhereClause()
		if whereClause != "" {
			query += " WHERE " + whereClause
			args = append(args, whereArgs...)
		}
	}

	// Add RETURNING clause
	if len(qb.returning) > 0 {
		quotedRet := make([]string, 0, len(qb.returning))
		for _, col := range qb.returning {
			if quoted := quoteIdentifier(col); quoted != "" {
				quotedRet = append(quotedRet, quoted)
			}
		}
		if len(quotedRet) > 0 {
			query += " RETURNING " + strings.Join(quotedRet, ", ")
		}
	}

	return query, args
}

// buildWhereClause builds the WHERE clause from filters.
// This is a simplified version that handles basic AND/OR grouping.
func (qb *QueryBuilder) buildWhereClause() (string, []interface{}) {
	var args []interface{}

	// Build SQL for each filter
	type filterSQL struct {
		condition string
		filter    Filter
	}
	filterSQLs := make([]filterSQL, 0, len(qb.filters))

	for _, filter := range qb.filters {
		condition, arg := qb.filterToSQL(filter)
		if condition != "" {
			filterSQLs = append(filterSQLs, filterSQL{condition: condition, filter: filter})
			if arg != nil {
				args = append(args, arg)
			}
		}
	}

	// Group OR conditions by OrGroupID
	orGroups := make(map[int][]string)
	var finalConditions []string

	for _, fs := range filterSQLs {
		if fs.filter.OrGroupID > 0 {
			orGroups[fs.filter.OrGroupID] = append(orGroups[fs.filter.OrGroupID], fs.condition)
		} else if fs.filter.IsOr {
			// Legacy OR support - treat as single OR group
			orGroups[-1] = append(orGroups[-1], fs.condition)
		} else {
			finalConditions = append(finalConditions, fs.condition)
		}
	}

	// Add OR groups
	for _, conditions := range orGroups {
		if len(conditions) == 1 {
			finalConditions = append(finalConditions, conditions[0])
		} else if len(conditions) > 1 {
			finalConditions = append(finalConditions, "("+strings.Join(conditions, " OR ")+")")
		}
	}

	return strings.Join(finalConditions, " AND "), args
}

// filterToSQL converts a single filter to SQL condition and argument.
func (qb *QueryBuilder) filterToSQL(filter Filter) (string, interface{}) {
	quotedCol := quoteIdentifier(filter.Column)
	if quotedCol == "" {
		return "", nil
	}

	placeholder := fmt.Sprintf("$%d", qb.argCounter)
	qb.argCounter++

	switch filter.Operator {
	case OpEqual:
		return fmt.Sprintf("%s = %s", quotedCol, placeholder), filter.Value
	case OpNotEqual:
		return fmt.Sprintf("%s <> %s", quotedCol, placeholder), filter.Value
	case OpGreaterThan:
		return fmt.Sprintf("%s > %s", quotedCol, placeholder), filter.Value
	case OpGreaterOrEqual:
		return fmt.Sprintf("%s >= %s", quotedCol, placeholder), filter.Value
	case OpLessThan:
		return fmt.Sprintf("%s < %s", quotedCol, placeholder), filter.Value
	case OpLessOrEqual:
		return fmt.Sprintf("%s <= %s", quotedCol, placeholder), filter.Value
	case OpLike:
		return fmt.Sprintf("%s LIKE %s", quotedCol, placeholder), filter.Value
	case OpILike:
		return fmt.Sprintf("%s ILIKE %s", quotedCol, placeholder), filter.Value
	case OpIs:
		qb.argCounter-- // IS doesn't use a placeholder
		if filter.Value == nil || filter.Value == "null" {
			return fmt.Sprintf("%s IS NULL", quotedCol), nil
		}
		return fmt.Sprintf("%s IS %v", quotedCol, filter.Value), nil
	case OpIn:
		return fmt.Sprintf("%s = ANY(%s)", quotedCol, placeholder), filter.Value
	case OpContains:
		return fmt.Sprintf("%s @> %s", quotedCol, placeholder), filter.Value
	case OpContained:
		return fmt.Sprintf("%s <@ %s", quotedCol, placeholder), filter.Value
	case OpOverlap:
		return fmt.Sprintf("%s && %s", quotedCol, placeholder), filter.Value
	default:
		return fmt.Sprintf("%s = %s", quotedCol, placeholder), filter.Value
	}
}

// buildOrderClause builds the ORDER BY clause.
func (qb *QueryBuilder) buildOrderClause() string {
	var parts []string

	for _, order := range qb.orderBy {
		quoted := quoteIdentifier(order.Column)
		if quoted == "" {
			continue
		}

		part := quoted
		if order.Desc {
			part += " DESC"
		} else {
			part += " ASC"
		}

		switch order.Nulls {
		case "first":
			part += " NULLS FIRST"
		case "last":
			part += " NULLS LAST"
		}

		parts = append(parts, part)
	}

	return strings.Join(parts, ", ")
}

// buildGroupByClause builds the GROUP BY clause.
func (qb *QueryBuilder) buildGroupByClause() string {
	if len(qb.groupBy) == 0 {
		return ""
	}

	var quotedCols []string
	for _, col := range qb.groupBy {
		if quoted := quoteIdentifier(col); quoted != "" {
			quotedCols = append(quotedCols, quoted)
		}
	}

	if len(quotedCols) == 0 {
		return ""
	}

	return " GROUP BY " + strings.Join(quotedCols, ", ")
}

// buildCursorCondition builds a keyset pagination condition.
// For ascending order: column > value
// For descending order: column < value
func (qb *QueryBuilder) buildCursorCondition() (string, interface{}) {
	if qb.cursorData == nil {
		return "", nil
	}

	// Use cursor column override if provided, otherwise use the column from cursor data
	column := qb.cursorData.Column
	if qb.cursorColumn != "" {
		column = qb.cursorColumn
	}

	quoted := quoteIdentifier(column)
	if quoted == "" {
		return "", nil
	}

	// Determine comparison operator based on order direction
	op := ">"
	if qb.cursorData.Desc {
		op = "<"
	}

	// Build the condition: column > $N or column < $N
	condition := fmt.Sprintf("%s %s $%d", quoted, op, qb.argCounter)
	qb.argCounter++

	return condition, qb.cursorData.Value
}
