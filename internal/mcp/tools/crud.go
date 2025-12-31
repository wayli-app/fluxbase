package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/fluxbase-eu/fluxbase/internal/database"
	"github.com/fluxbase-eu/fluxbase/internal/mcp"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"
)

// InsertRecordTool implements the insert_record MCP tool
type InsertRecordTool struct {
	db          *database.Connection
	schemaCache *database.SchemaCache
}

// NewInsertRecordTool creates a new insert_record tool
func NewInsertRecordTool(db *database.Connection, schemaCache *database.SchemaCache) *InsertRecordTool {
	return &InsertRecordTool{
		db:          db,
		schemaCache: schemaCache,
	}
}

func (t *InsertRecordTool) Name() string {
	return "insert_record"
}

func (t *InsertRecordTool) Description() string {
	return "Insert a new record into a database table. Respects Row Level Security (RLS) policies."
}

func (t *InsertRecordTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"table": map[string]any{
				"type":        "string",
				"description": "The table name to insert into (can include schema prefix like 'public.users')",
			},
			"data": map[string]any{
				"type":        "object",
				"description": "The record data as key-value pairs where keys are column names",
			},
			"returning": map[string]any{
				"type":        "string",
				"description": "Columns to return after insert (default: '*')",
				"default":     "*",
			},
		},
		"required": []string{"table", "data"},
	}
}

func (t *InsertRecordTool) RequiredScopes() []string {
	return []string{mcp.ScopeWriteTables}
}

func (t *InsertRecordTool) Execute(ctx context.Context, args map[string]any, authCtx *mcp.AuthContext) (*mcp.ToolResult, error) {
	// Parse arguments
	tableName, ok := args["table"].(string)
	if !ok || tableName == "" {
		return nil, fmt.Errorf("table name is required")
	}

	data, ok := args["data"].(map[string]any)
	if !ok || len(data) == 0 {
		return nil, fmt.Errorf("data is required and must be a non-empty object")
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
	var tableInfo *database.TableInfo
	if t.schemaCache != nil {
		info, exists, err := t.schemaCache.GetTable(ctx, schema, table)
		if err != nil {
			return nil, fmt.Errorf("failed to get table metadata: %w", err)
		}
		if !exists || info == nil {
			return nil, fmt.Errorf("table not found: %s.%s", schema, table)
		}
		tableInfo = info
	}

	// Parse returning columns
	returning := "*"
	if ret, ok := args["returning"].(string); ok && ret != "" {
		returning = ret
	}

	// Build INSERT query
	columns := make([]string, 0, len(data))
	values := make([]any, 0, len(data))
	placeholders := make([]string, 0, len(data))

	i := 1
	for col, val := range data {
		// Validate column exists if we have schema cache
		if tableInfo != nil && !columnExists(tableInfo, col) {
			return nil, fmt.Errorf("unknown column: %s", col)
		}

		columns = append(columns, quoteIdentifier(col))
		values = append(values, val)
		placeholders = append(placeholders, fmt.Sprintf("$%d", i))
		i++
	}

	query := fmt.Sprintf(
		`INSERT INTO "%s"."%s" (%s) VALUES (%s) RETURNING %s`,
		schema, table,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "),
		returning,
	)

	log.Debug().
		Str("query", query).
		Interface("values", values).
		Str("schema", schema).
		Str("table", table).
		Msg("MCP: Executing insert_record")

	// Execute query with RLS context
	var results []map[string]any
	err := executeWithRLS(ctx, t.db, authCtx, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, query, values...)
		if err != nil {
			return err
		}
		defer rows.Close()

		results, err = scanRowsToMaps(rows)
		return err
	})

	if err != nil {
		log.Error().Err(err).Str("query", query).Msg("MCP: insert_record execution failed")
		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.ErrorContent(fmt.Sprintf("Insert failed: %v", err))},
			IsError: true,
		}, nil
	}

	if len(results) == 0 {
		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.ErrorContent("Insert blocked by Row Level Security policy")},
			IsError: true,
		}, nil
	}

	// Serialize results to JSON
	resultJSON, err := json.MarshalIndent(results[0], "", "  ")
	if err != nil {
		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.ErrorContent(fmt.Sprintf("Failed to serialize result: %v", err))},
			IsError: true,
		}, nil
	}

	return &mcp.ToolResult{
		Content: []mcp.Content{mcp.TextContent(string(resultJSON))},
	}, nil
}

// UpdateRecordTool implements the update_record MCP tool
type UpdateRecordTool struct {
	db          *database.Connection
	schemaCache *database.SchemaCache
}

// NewUpdateRecordTool creates a new update_record tool
func NewUpdateRecordTool(db *database.Connection, schemaCache *database.SchemaCache) *UpdateRecordTool {
	return &UpdateRecordTool{
		db:          db,
		schemaCache: schemaCache,
	}
}

func (t *UpdateRecordTool) Name() string {
	return "update_record"
}

func (t *UpdateRecordTool) Description() string {
	return "Update records in a database table matching given filters. Respects Row Level Security (RLS) policies."
}

func (t *UpdateRecordTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"table": map[string]any{
				"type":        "string",
				"description": "The table name to update (can include schema prefix like 'public.users')",
			},
			"data": map[string]any{
				"type":        "object",
				"description": "The data to update as key-value pairs where keys are column names",
			},
			"filter": map[string]any{
				"type":        "object",
				"description": "Filter conditions as key-value pairs where key is 'column' and value is 'operator.value' (e.g., {\"id\": \"eq.123\"})",
				"additionalProperties": map[string]any{
					"type": "string",
				},
			},
			"returning": map[string]any{
				"type":        "string",
				"description": "Columns to return after update (default: '*')",
				"default":     "*",
			},
		},
		"required": []string{"table", "data", "filter"},
	}
}

func (t *UpdateRecordTool) RequiredScopes() []string {
	return []string{mcp.ScopeWriteTables}
}

func (t *UpdateRecordTool) Execute(ctx context.Context, args map[string]any, authCtx *mcp.AuthContext) (*mcp.ToolResult, error) {
	// Parse arguments
	tableName, ok := args["table"].(string)
	if !ok || tableName == "" {
		return nil, fmt.Errorf("table name is required")
	}

	data, ok := args["data"].(map[string]any)
	if !ok || len(data) == 0 {
		return nil, fmt.Errorf("data is required and must be a non-empty object")
	}

	filterMap, ok := args["filter"].(map[string]any)
	if !ok || len(filterMap) == 0 {
		return nil, fmt.Errorf("filter is required and must be a non-empty object (to prevent accidental full table updates)")
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
	var tableInfo *database.TableInfo
	if t.schemaCache != nil {
		info, exists, err := t.schemaCache.GetTable(ctx, schema, table)
		if err != nil {
			return nil, fmt.Errorf("failed to get table metadata: %w", err)
		}
		if !exists || info == nil {
			return nil, fmt.Errorf("table not found: %s.%s", schema, table)
		}
		tableInfo = info
	}

	// Parse returning columns
	returning := "*"
	if ret, ok := args["returning"].(string); ok && ret != "" {
		returning = ret
	}

	// Build UPDATE query
	setClauses := make([]string, 0, len(data))
	values := make([]any, 0, len(data)+len(filterMap))

	i := 1
	for col, val := range data {
		// Validate column exists if we have schema cache
		if tableInfo != nil && !columnExists(tableInfo, col) {
			return nil, fmt.Errorf("unknown column: %s", col)
		}

		setClauses = append(setClauses, fmt.Sprintf("%s = $%d", quoteIdentifier(col), i))
		values = append(values, val)
		i++
	}

	// Build WHERE clause from filters
	whereClauses := make([]string, 0, len(filterMap))
	for column, value := range filterMap {
		valueStr, ok := value.(string)
		if !ok {
			continue
		}

		whereClause, whereValue, err := parseFilterToSQL(column, valueStr, i)
		if err != nil {
			return nil, fmt.Errorf("invalid filter for column %s: %w", column, err)
		}
		whereClauses = append(whereClauses, whereClause)
		if whereValue != nil {
			values = append(values, whereValue)
			i++
		}
	}

	query := fmt.Sprintf(
		`UPDATE "%s"."%s" SET %s WHERE %s RETURNING %s`,
		schema, table,
		strings.Join(setClauses, ", "),
		strings.Join(whereClauses, " AND "),
		returning,
	)

	log.Debug().
		Str("query", query).
		Str("schema", schema).
		Str("table", table).
		Msg("MCP: Executing update_record")

	// Execute query with RLS context
	var results []map[string]any
	err := executeWithRLS(ctx, t.db, authCtx, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, query, values...)
		if err != nil {
			return err
		}
		defer rows.Close()

		results, err = scanRowsToMaps(rows)
		return err
	})

	if err != nil {
		log.Error().Err(err).Str("query", query).Msg("MCP: update_record execution failed")
		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.ErrorContent(fmt.Sprintf("Update failed: %v", err))},
			IsError: true,
		}, nil
	}

	// Serialize results to JSON
	resultJSON, err := json.MarshalIndent(map[string]any{
		"updated": len(results),
		"records": results,
	}, "", "  ")
	if err != nil {
		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.ErrorContent(fmt.Sprintf("Failed to serialize result: %v", err))},
			IsError: true,
		}, nil
	}

	return &mcp.ToolResult{
		Content: []mcp.Content{mcp.TextContent(string(resultJSON))},
	}, nil
}

// DeleteRecordTool implements the delete_record MCP tool
type DeleteRecordTool struct {
	db          *database.Connection
	schemaCache *database.SchemaCache
}

// NewDeleteRecordTool creates a new delete_record tool
func NewDeleteRecordTool(db *database.Connection, schemaCache *database.SchemaCache) *DeleteRecordTool {
	return &DeleteRecordTool{
		db:          db,
		schemaCache: schemaCache,
	}
}

func (t *DeleteRecordTool) Name() string {
	return "delete_record"
}

func (t *DeleteRecordTool) Description() string {
	return "Delete records from a database table matching given filters. Respects Row Level Security (RLS) policies."
}

func (t *DeleteRecordTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"table": map[string]any{
				"type":        "string",
				"description": "The table name to delete from (can include schema prefix like 'public.users')",
			},
			"filter": map[string]any{
				"type":        "object",
				"description": "Filter conditions as key-value pairs where key is 'column' and value is 'operator.value' (e.g., {\"id\": \"eq.123\"})",
				"additionalProperties": map[string]any{
					"type": "string",
				},
			},
			"returning": map[string]any{
				"type":        "string",
				"description": "Columns to return after delete (default: '*')",
				"default":     "*",
			},
		},
		"required": []string{"table", "filter"},
	}
}

func (t *DeleteRecordTool) RequiredScopes() []string {
	return []string{mcp.ScopeWriteTables}
}

func (t *DeleteRecordTool) Execute(ctx context.Context, args map[string]any, authCtx *mcp.AuthContext) (*mcp.ToolResult, error) {
	// Parse arguments
	tableName, ok := args["table"].(string)
	if !ok || tableName == "" {
		return nil, fmt.Errorf("table name is required")
	}

	filterMap, ok := args["filter"].(map[string]any)
	if !ok || len(filterMap) == 0 {
		return nil, fmt.Errorf("filter is required and must be a non-empty object (to prevent accidental full table deletes)")
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

	// Parse returning columns
	returning := "*"
	if ret, ok := args["returning"].(string); ok && ret != "" {
		returning = ret
	}

	// Build WHERE clause from filters
	whereClauses := make([]string, 0, len(filterMap))
	values := make([]any, 0, len(filterMap))

	i := 1
	for column, value := range filterMap {
		valueStr, ok := value.(string)
		if !ok {
			continue
		}

		whereClause, whereValue, err := parseFilterToSQL(column, valueStr, i)
		if err != nil {
			return nil, fmt.Errorf("invalid filter for column %s: %w", column, err)
		}
		whereClauses = append(whereClauses, whereClause)
		if whereValue != nil {
			values = append(values, whereValue)
			i++
		}
	}

	query := fmt.Sprintf(
		`DELETE FROM "%s"."%s" WHERE %s RETURNING %s`,
		schema, table,
		strings.Join(whereClauses, " AND "),
		returning,
	)

	log.Debug().
		Str("query", query).
		Str("schema", schema).
		Str("table", table).
		Msg("MCP: Executing delete_record")

	// Execute query with RLS context
	var results []map[string]any
	err := executeWithRLS(ctx, t.db, authCtx, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, query, values...)
		if err != nil {
			return err
		}
		defer rows.Close()

		results, err = scanRowsToMaps(rows)
		return err
	})

	if err != nil {
		log.Error().Err(err).Str("query", query).Msg("MCP: delete_record execution failed")
		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.ErrorContent(fmt.Sprintf("Delete failed: %v", err))},
			IsError: true,
		}, nil
	}

	// Serialize results to JSON
	resultJSON, err := json.MarshalIndent(map[string]any{
		"deleted": len(results),
		"records": results,
	}, "", "  ")
	if err != nil {
		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.ErrorContent(fmt.Sprintf("Failed to serialize result: %v", err))},
			IsError: true,
		}, nil
	}

	return &mcp.ToolResult{
		Content: []mcp.Content{mcp.TextContent(string(resultJSON))},
	}, nil
}

// Helper functions

// quoteIdentifier quotes a SQL identifier to prevent SQL injection
func quoteIdentifier(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

// columnExists checks if a column exists in the table
func columnExists(table *database.TableInfo, columnName string) bool {
	for _, col := range table.Columns {
		if col.Name == columnName {
			return true
		}
	}
	return false
}

// parseFilterToSQL converts a filter value like "eq.123" to SQL
// Returns the WHERE clause fragment, the value for the placeholder (if any), and an error
func parseFilterToSQL(column, valueStr string, paramIndex int) (string, any, error) {
	parts := strings.SplitN(valueStr, ".", 2)
	if len(parts) != 2 {
		return "", nil, fmt.Errorf("expected format 'operator.value', got '%s'", valueStr)
	}

	operator := parts[0]
	value := parts[1]
	quotedCol := quoteIdentifier(column)

	switch operator {
	case "eq":
		return fmt.Sprintf("%s = $%d", quotedCol, paramIndex), value, nil
	case "neq":
		return fmt.Sprintf("%s != $%d", quotedCol, paramIndex), value, nil
	case "gt":
		return fmt.Sprintf("%s > $%d", quotedCol, paramIndex), value, nil
	case "gte":
		return fmt.Sprintf("%s >= $%d", quotedCol, paramIndex), value, nil
	case "lt":
		return fmt.Sprintf("%s < $%d", quotedCol, paramIndex), value, nil
	case "lte":
		return fmt.Sprintf("%s <= $%d", quotedCol, paramIndex), value, nil
	case "like":
		return fmt.Sprintf("%s LIKE $%d", quotedCol, paramIndex), value, nil
	case "ilike":
		return fmt.Sprintf("%s ILIKE $%d", quotedCol, paramIndex), value, nil
	case "is":
		switch strings.ToLower(value) {
		case "null":
			return fmt.Sprintf("%s IS NULL", quotedCol), nil, nil
		case "true":
			return fmt.Sprintf("%s IS TRUE", quotedCol), nil, nil
		case "false":
			return fmt.Sprintf("%s IS FALSE", quotedCol), nil, nil
		default:
			return "", nil, fmt.Errorf("invalid 'is' value: %s (expected null, true, or false)", value)
		}
	case "in":
		// Parse as array: (val1,val2,val3)
		value = strings.TrimPrefix(value, "(")
		value = strings.TrimSuffix(value, ")")
		inValues := strings.Split(value, ",")
		return fmt.Sprintf("%s = ANY($%d)", quotedCol, paramIndex), inValues, nil
	default:
		return "", nil, fmt.Errorf("unsupported operator: %s", operator)
	}
}
