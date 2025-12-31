package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/fluxbase-eu/fluxbase/internal/database"
	"github.com/fluxbase-eu/fluxbase/internal/mcp"
	"github.com/fluxbase-eu/fluxbase/internal/middleware"
	"github.com/fluxbase-eu/fluxbase/internal/query"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"
)

// validIdentifierRegex validates SQL identifiers (column names, table names, etc.)
var validIdentifierRegex = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)


// executeWithRLS wraps a database operation with RLS context from MCP AuthContext
// This is similar to middleware.WrapWithRLS but works without a Fiber context
func executeWithRLS(ctx context.Context, db *database.Connection, authCtx *mcp.AuthContext, fn func(tx pgx.Tx) error) error {
	// Start transaction
	tx, err := db.Pool().Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Set RLS context from MCP AuthContext
	userID := ""
	if authCtx.UserID != nil {
		userID = *authCtx.UserID
	}
	role := authCtx.UserRole
	if role == "" {
		role = "anon"
	}

	log.Debug().
		Str("user_id", userID).
		Str("role", role).
		Str("auth_type", authCtx.AuthType).
		Msg("MCP executeWithRLS: Setting RLS context")

	// Use the existing SetRLSContext from middleware
	// Pass nil for claims since MCP doesn't have full JWT claims
	if err := middleware.SetRLSContext(ctx, tx, userID, role, nil); err != nil {
		return err
	}

	// Execute the wrapped function
	if err := fn(tx); err != nil {
		return err
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// QueryTableTool implements the query_table MCP tool
type QueryTableTool struct {
	db          *database.Connection
	schemaCache *database.SchemaCache
}

// NewQueryTableTool creates a new query_table tool
func NewQueryTableTool(db *database.Connection, schemaCache *database.SchemaCache) *QueryTableTool {
	return &QueryTableTool{
		db:          db,
		schemaCache: schemaCache,
	}
}

func (t *QueryTableTool) Name() string {
	return "query_table"
}

func (t *QueryTableTool) Description() string {
	return "Query a database table with filters, ordering, and pagination. Respects Row Level Security (RLS) policies."
}

func (t *QueryTableTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"table": map[string]any{
				"type":        "string",
				"description": "The table name to query (can include schema prefix like 'public.users')",
			},
			"select": map[string]any{
				"type":        "string",
				"description": "Comma-separated list of columns to select, or '*' for all columns",
				"default":     "*",
			},
			"filter": map[string]any{
				"type":        "object",
				"description": "Filter conditions as key-value pairs where key is 'column' and value is 'operator.value' (e.g., {\"is_active\": \"eq.true\", \"age\": \"gt.18\"})",
				"additionalProperties": map[string]any{
					"type": "string",
				},
			},
			"order": map[string]any{
				"type":        "string",
				"description": "Order by clause (e.g., 'created_at.desc', 'name.asc')",
			},
			"limit": map[string]any{
				"type":        "integer",
				"description": "Maximum number of rows to return",
				"default":     100,
				"maximum":     1000,
			},
			"offset": map[string]any{
				"type":        "integer",
				"description": "Number of rows to skip (for pagination)",
				"default":     0,
			},
		},
		"required": []string{"table"},
	}
}

func (t *QueryTableTool) RequiredScopes() []string {
	return []string{mcp.ScopeReadTables}
}

func (t *QueryTableTool) Execute(ctx context.Context, args map[string]any, authCtx *mcp.AuthContext) (*mcp.ToolResult, error) {
	// Parse arguments
	tableName, ok := args["table"].(string)
	if !ok || tableName == "" {
		return nil, fmt.Errorf("table name is required")
	}

	// Parse schema.table format
	schema := "public"
	table := tableName
	if strings.Contains(tableName, ".") {
		parts := strings.SplitN(tableName, ".", 2)
		schema = parts[0]
		table = parts[1]
	}

	// Validate table exists
	if t.schemaCache != nil {
		_, exists, err := t.schemaCache.GetTable(ctx, schema, table)
		if err != nil {
			return nil, fmt.Errorf("failed to get table metadata: %w", err)
		}
		if !exists {
			return nil, fmt.Errorf("table not found: %s.%s", schema, table)
		}
	}

	// Parse select columns
	columns := []string{}
	if selectStr, ok := args["select"].(string); ok && selectStr != "" && selectStr != "*" {
		for _, col := range strings.Split(selectStr, ",") {
			columns = append(columns, strings.TrimSpace(col))
		}
	}

	// Parse filters
	var filters []query.Filter
	if filterMap, ok := args["filter"].(map[string]any); ok {
		for column, value := range filterMap {
			valueStr, ok := value.(string)
			if !ok {
				continue
			}

			// Parse operator.value format
			filter, err := parseFilterValue(column, valueStr)
			if err != nil {
				return nil, fmt.Errorf("invalid filter for column %s: %w", column, err)
			}
			filters = append(filters, filter)
		}
	}

	// Parse order
	var orderBy []query.OrderBy
	if orderStr, ok := args["order"].(string); ok && orderStr != "" {
		order, err := parseOrder(orderStr)
		if err != nil {
			return nil, fmt.Errorf("invalid order: %w", err)
		}
		orderBy = order
	}

	// Parse limit and offset
	limit := 100
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
		if limit > 1000 {
			limit = 1000
		}
	}

	offset := 0
	if o, ok := args["offset"].(float64); ok {
		offset = int(o)
	}

	// Build query using simple SQL construction
	sqlQuery, queryArgs := buildSelectQuery(schema, table, columns, filters, orderBy, limit, offset)

	log.Debug().
		Str("query", sqlQuery).
		Interface("args", queryArgs).
		Str("schema", schema).
		Str("table", table).
		Msg("MCP: Executing query_table")

	// Execute query with RLS context
	var results []map[string]any
	err := executeWithRLS(ctx, t.db, authCtx, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, sqlQuery, queryArgs...)
		if err != nil {
			return err
		}
		defer rows.Close()

		results, err = scanRowsToMaps(rows)
		return err
	})

	if err != nil {
		log.Error().Err(err).Str("query", sqlQuery).Msg("MCP: query_table execution failed")
		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.ErrorContent(fmt.Sprintf("Query failed: %v", err))},
			IsError: true,
		}, nil
	}

	// Serialize results to JSON
	resultJSON, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.ErrorContent(fmt.Sprintf("Failed to serialize results: %v", err))},
			IsError: true,
		}, nil
	}

	return &mcp.ToolResult{
		Content: []mcp.Content{mcp.TextContent(string(resultJSON))},
	}, nil
}

// buildSelectQuery builds a SELECT SQL query from the given parameters
func buildSelectQuery(schema, table string, columns []string, filters []query.Filter, orderBy []query.OrderBy, limit, offset int) (string, []interface{}) {
	var args []interface{}
	argCounter := 1

	// Build SELECT clause
	selectClause := "*"
	if len(columns) > 0 {
		quotedCols := make([]string, 0, len(columns))
		for _, col := range columns {
			if quoted := quoteIdentifier(col); quoted != "" {
				quotedCols = append(quotedCols, quoted)
			}
		}
		if len(quotedCols) > 0 {
			selectClause = strings.Join(quotedCols, ", ")
		}
	}

	// Build FROM clause
	sqlQuery := fmt.Sprintf("SELECT %s FROM %s.%s",
		selectClause,
		quoteIdentifier(schema),
		quoteIdentifier(table))

	// Build WHERE clause
	if len(filters) > 0 {
		var conditions []string
		for _, filter := range filters {
			quotedCol := quoteIdentifier(filter.Column)
			if quotedCol == "" {
				continue
			}

			condition, arg := filterToSQL(filter, quotedCol, argCounter)
			if condition != "" {
				conditions = append(conditions, condition)
				if arg != nil {
					args = append(args, arg)
					argCounter++
				}
			}
		}
		if len(conditions) > 0 {
			sqlQuery += " WHERE " + strings.Join(conditions, " AND ")
		}
	}

	// Build ORDER BY clause
	if len(orderBy) > 0 {
		var orderParts []string
		for _, order := range orderBy {
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
			orderParts = append(orderParts, part)
		}
		if len(orderParts) > 0 {
			sqlQuery += " ORDER BY " + strings.Join(orderParts, ", ")
		}
	}

	// Build LIMIT clause
	sqlQuery += fmt.Sprintf(" LIMIT %d", limit)

	// Build OFFSET clause
	if offset > 0 {
		sqlQuery += fmt.Sprintf(" OFFSET %d", offset)
	}

	return sqlQuery, args
}

// filterToSQL converts a filter to SQL condition and argument
func filterToSQL(filter query.Filter, quotedCol string, argCounter int) (string, interface{}) {
	placeholder := fmt.Sprintf("$%d", argCounter)

	switch filter.Operator {
	case query.OpEqual:
		return fmt.Sprintf("%s = %s", quotedCol, placeholder), filter.Value
	case query.OpNotEqual:
		return fmt.Sprintf("%s <> %s", quotedCol, placeholder), filter.Value
	case query.OpGreaterThan:
		return fmt.Sprintf("%s > %s", quotedCol, placeholder), filter.Value
	case query.OpGreaterOrEqual:
		return fmt.Sprintf("%s >= %s", quotedCol, placeholder), filter.Value
	case query.OpLessThan:
		return fmt.Sprintf("%s < %s", quotedCol, placeholder), filter.Value
	case query.OpLessOrEqual:
		return fmt.Sprintf("%s <= %s", quotedCol, placeholder), filter.Value
	case query.OpLike:
		return fmt.Sprintf("%s LIKE %s", quotedCol, placeholder), filter.Value
	case query.OpILike:
		return fmt.Sprintf("%s ILIKE %s", quotedCol, placeholder), filter.Value
	case query.OpIs:
		if filter.Value == nil || filter.Value == "null" {
			return fmt.Sprintf("%s IS NULL", quotedCol), nil
		}
		return fmt.Sprintf("%s IS %v", quotedCol, filter.Value), nil
	case query.OpIsNot:
		if filter.Value == nil || filter.Value == "null" {
			return fmt.Sprintf("%s IS NOT NULL", quotedCol), nil
		}
		return fmt.Sprintf("%s IS NOT %v", quotedCol, filter.Value), nil
	case query.OpIn:
		return fmt.Sprintf("%s = ANY(%s)", quotedCol, placeholder), filter.Value
	case query.OpNotIn:
		return fmt.Sprintf("%s <> ALL(%s)", quotedCol, placeholder), filter.Value
	case query.OpContains:
		return fmt.Sprintf("%s @> %s", quotedCol, placeholder), filter.Value
	case query.OpContainedBy:
		return fmt.Sprintf("%s <@ %s", quotedCol, placeholder), filter.Value
	case query.OpOverlaps:
		return fmt.Sprintf("%s && %s", quotedCol, placeholder), filter.Value
	default:
		return fmt.Sprintf("%s = %s", quotedCol, placeholder), filter.Value
	}
}

// parseFilterValue parses a filter value in "operator.value" format
func parseFilterValue(column, valueStr string) (query.Filter, error) {
	parts := strings.SplitN(valueStr, ".", 2)
	if len(parts) != 2 {
		return query.Filter{}, fmt.Errorf("expected format 'operator.value', got '%s'", valueStr)
	}

	operator := query.FilterOperator(parts[0])
	value := parts[1]

	// Validate operator
	validOperators := map[query.FilterOperator]bool{
		query.OpEqual: true, query.OpNotEqual: true,
		query.OpGreaterThan: true, query.OpGreaterOrEqual: true,
		query.OpLessThan: true, query.OpLessOrEqual: true,
		query.OpLike: true, query.OpILike: true,
		query.OpIn: true, query.OpNotIn: true,
		query.OpIs: true, query.OpIsNot: true,
		query.OpContains: true, query.OpContainedBy: true,
		query.OpOverlaps: true,
	}

	if !validOperators[operator] {
		return query.Filter{}, fmt.Errorf("invalid operator: %s", parts[0])
	}

	// Parse value based on operator
	var parsedValue any = value
	if operator == query.OpIs || operator == query.OpIsNot {
		// is.null, is.true, is.false
		switch strings.ToLower(value) {
		case "null":
			parsedValue = nil
		case "true":
			parsedValue = true
		case "false":
			parsedValue = false
		}
	} else if operator == query.OpIn || operator == query.OpNotIn {
		// Parse as array: (val1,val2,val3)
		value = strings.TrimPrefix(value, "(")
		value = strings.TrimSuffix(value, ")")
		parsedValue = strings.Split(value, ",")
	}

	return query.Filter{
		Column:   column,
		Operator: operator,
		Value:    parsedValue,
	}, nil
}

// parseOrder parses an order string like "column.asc" or "column.desc"
func parseOrder(orderStr string) ([]query.OrderBy, error) {
	var result []query.OrderBy

	for _, part := range strings.Split(orderStr, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Default to ascending
		column := part
		desc := false

		if strings.HasSuffix(part, ".desc") {
			column = strings.TrimSuffix(part, ".desc")
			desc = true
		} else if strings.HasSuffix(part, ".asc") {
			column = strings.TrimSuffix(part, ".asc")
		}

		result = append(result, query.OrderBy{
			Column: column,
			Desc:   desc,
		})
	}

	return result, nil
}

// scanRowsToMaps scans pgx rows into a slice of maps
func scanRowsToMaps(rows pgx.Rows) ([]map[string]any, error) {
	var results []map[string]any

	fieldDescs := rows.FieldDescriptions()
	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			return nil, err
		}

		row := make(map[string]any)
		for i, fd := range fieldDescs {
			row[string(fd.Name)] = values[i]
		}
		results = append(results, row)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
}
