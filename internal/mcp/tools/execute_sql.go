package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/fluxbase-eu/fluxbase/internal/ai"
	"github.com/fluxbase-eu/fluxbase/internal/database"
	"github.com/fluxbase-eu/fluxbase/internal/mcp"
	"github.com/fluxbase-eu/fluxbase/internal/middleware"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"
)

const (
	defaultMaxRows      = 100
	defaultQueryTimeout = 30 * time.Second
)

// ExecuteSQLTool implements the execute_sql MCP tool for running raw SQL queries
type ExecuteSQLTool struct {
	db      *database.Connection
	maxRows int
	timeout time.Duration
}

// NewExecuteSQLTool creates a new execute_sql tool
func NewExecuteSQLTool(db *database.Connection) *ExecuteSQLTool {
	return &ExecuteSQLTool{
		db:      db,
		maxRows: defaultMaxRows,
		timeout: defaultQueryTimeout,
	}
}

// NewExecuteSQLToolWithOptions creates a new execute_sql tool with custom options
func NewExecuteSQLToolWithOptions(db *database.Connection, maxRows int, timeout time.Duration) *ExecuteSQLTool {
	return &ExecuteSQLTool{
		db:      db,
		maxRows: maxRows,
		timeout: timeout,
	}
}

func (t *ExecuteSQLTool) Name() string {
	return "execute_sql"
}

func (t *ExecuteSQLTool) Description() string {
	return "Execute a read-only SQL query against the database. Returns a summary of results. Only SELECT queries are allowed by default."
}

func (t *ExecuteSQLTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"sql": map[string]any{
				"type":        "string",
				"description": "The SQL SELECT query to execute",
			},
			"description": map[string]any{
				"type":        "string",
				"description": "A brief description of what this query is meant to find",
			},
		},
		"required": []string{"sql", "description"},
	}
}

func (t *ExecuteSQLTool) RequiredScopes() []string {
	return []string{mcp.ScopeExecuteSQL}
}

func (t *ExecuteSQLTool) Execute(ctx context.Context, args map[string]any, authCtx *mcp.AuthContext) (*mcp.ToolResult, error) {
	// Extract arguments
	sqlQuery, ok := args["sql"].(string)
	if !ok || sqlQuery == "" {
		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.ErrorContent("sql is required")},
			IsError: true,
		}, nil
	}

	description, _ := args["description"].(string)

	// Get configs from metadata
	allowedSchemas := authCtx.GetMetadataStringSlice(mcp.MetadataKeyAllowedSchemas)
	allowedTables := authCtx.GetMetadataStringSlice(mcp.MetadataKeyAllowedTables)
	allowedOperations := authCtx.GetMetadataStringSlice(mcp.MetadataKeyAllowedOperations)

	// Default to SELECT only if no operations specified
	if len(allowedOperations) == 0 {
		allowedOperations = []string{"SELECT"}
	}

	log.Debug().
		Str("sql", sqlQuery).
		Str("description", description).
		Strs("allowed_schemas", allowedSchemas).
		Strs("allowed_tables", allowedTables).
		Strs("allowed_operations", allowedOperations).
		Msg("MCP: Executing SQL query")

	// Execute the query
	result := t.executeSQL(ctx, sqlQuery, allowedSchemas, allowedTables, allowedOperations, authCtx)

	return result, nil
}

func (t *ExecuteSQLTool) executeSQL(
	ctx context.Context,
	sqlQuery string,
	allowedSchemas, allowedTables, allowedOperations []string,
	authCtx *mcp.AuthContext,
) *mcp.ToolResult {
	start := time.Now()

	// Create a context with timeout
	queryCtx, cancel := context.WithTimeout(ctx, t.timeout)
	defer cancel()

	// Validate the SQL using the existing SQLValidator
	validator := ai.NewSQLValidator(allowedSchemas, allowedTables, allowedOperations)
	validationResult, normalizedSQL, err := validator.ValidateAndNormalize(sqlQuery)

	if err != nil {
		log.Warn().
			Str("sql", sqlQuery).
			Strs("errors", validationResult.Errors).
			Msg("MCP: SQL validation failed")

		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.ErrorContent(fmt.Sprintf("Query validation failed: %s", err.Error()))},
			IsError: true,
		}
	}

	// Start transaction with RLS
	tx, err := t.db.Pool().Begin(queryCtx)
	if err != nil {
		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.ErrorContent(fmt.Sprintf("Failed to begin transaction: %s", err.Error()))},
			IsError: true,
		}
	}
	defer func() { _ = tx.Rollback(queryCtx) }()

	// Set RLS context from MCP AuthContext
	userID := ""
	if authCtx.UserID != nil {
		userID = *authCtx.UserID
	}
	role := authCtx.UserRole
	if role == "" {
		role = "anon"
	}

	if err := middleware.SetRLSContext(queryCtx, tx, userID, role, nil); err != nil {
		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.ErrorContent(fmt.Sprintf("Failed to set RLS context: %s", err.Error()))},
			IsError: true,
		}
	}

	// Set search_path to include allowed schemas
	if len(allowedSchemas) > 0 {
		quotedSchemas := make([]string, len(allowedSchemas))
		for i, schema := range allowedSchemas {
			quotedSchemas[i] = pgx.Identifier{schema}.Sanitize()
		}
		searchPathSQL := fmt.Sprintf("SET LOCAL search_path TO %s", strings.Join(quotedSchemas, ", "))
		if _, err := tx.Exec(queryCtx, searchPathSQL); err != nil {
			return &mcp.ToolResult{
				Content: []mcp.Content{mcp.ErrorContent(fmt.Sprintf("Failed to set search_path: %s", err.Error()))},
				IsError: true,
			}
		}
	}

	// Execute the query
	rows, err := tx.Query(queryCtx, normalizedSQL)
	if err != nil {
		log.Error().
			Err(err).
			Str("sql", normalizedSQL).
			Msg("MCP: SQL execution failed")

		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.ErrorContent(fmt.Sprintf("Query execution failed: %s", err.Error()))},
			IsError: true,
		}
	}
	defer rows.Close()

	// Get column names
	fieldDescs := rows.FieldDescriptions()
	columns := make([]string, len(fieldDescs))
	for i, fd := range fieldDescs {
		columns[i] = string(fd.Name)
	}

	// Collect rows
	resultRows := []map[string]any{}
	rowCount := 0
	for rows.Next() {
		if rowCount >= t.maxRows {
			break
		}

		values, err := rows.Values()
		if err != nil {
			log.Warn().Err(err).Msg("Failed to scan row values")
			continue
		}

		row := make(map[string]any)
		for i, col := range columns {
			row[col] = convertSQLValue(values[i])
		}
		resultRows = append(resultRows, row)
		rowCount++
	}

	if err := rows.Err(); err != nil {
		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.ErrorContent(fmt.Sprintf("Error reading results: %s", err.Error()))},
			IsError: true,
		}
	}

	// Commit transaction (important for RLS cleanup)
	if err := tx.Commit(queryCtx); err != nil {
		log.Warn().Err(err).Msg("Failed to commit transaction")
		// Continue anyway, we have the results
	}

	durationMs := time.Since(start).Milliseconds()

	// Build summary
	summary := fmt.Sprintf("Returned %d row(s) in %dms", rowCount, durationMs)
	if rowCount >= t.maxRows {
		summary = fmt.Sprintf("Returned %d row(s) (limited) in %dms", rowCount, durationMs)
	}

	log.Debug().
		Int("row_count", rowCount).
		Int64("duration_ms", durationMs).
		Strs("tables", validationResult.TablesAccessed).
		Msg("MCP: SQL query executed successfully")

	// Format result
	result := map[string]any{
		"success":     true,
		"row_count":   rowCount,
		"columns":     columns,
		"rows":        resultRows,
		"summary":     summary,
		"duration_ms": durationMs,
		"tables":      validationResult.TablesAccessed,
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.TextContent(summary)},
		}
	}

	return &mcp.ToolResult{
		Content: []mcp.Content{mcp.TextContent(string(resultJSON))},
	}
}

// convertSQLValue converts database values to JSON-friendly formats
func convertSQLValue(v any) any {
	if v == nil {
		return nil
	}

	switch val := v.(type) {
	case time.Time:
		return val.Format(time.RFC3339)
	case []byte:
		return string(val)
	default:
		return val
	}
}
