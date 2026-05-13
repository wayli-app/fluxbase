package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/database"
	"github.com/nimbleflux/fluxbase/internal/mcp"
)

// AddColumnTool implements the add_column MCP tool
type AddColumnTool struct {
	db *database.Connection
}

// NewAddColumnTool creates a new add_column tool
func NewAddColumnTool(db *database.Connection) *AddColumnTool {
	return &AddColumnTool{db: db}
}

func (t *AddColumnTool) Name() string {
	return "add_column"
}

func (t *AddColumnTool) Description() string {
	return "Add a new column to an existing table. Requires admin:ddl scope."
}

func (t *AddColumnTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"schema": map[string]any{
				"type":        "string",
				"description": "Schema name (default: 'public')",
				"default":     "public",
			},
			"table": map[string]any{
				"type":        "string",
				"description": "Table name",
			},
			"name": map[string]any{
				"type":        "string",
				"description": "Column name",
			},
			"type": map[string]any{
				"type":        "string",
				"description": "PostgreSQL data type",
			},
			"nullable": map[string]any{
				"type":        "boolean",
				"description": "Whether the column can be NULL (default: true)",
				"default":     true,
			},
			"default_value": map[string]any{
				"type":        "string",
				"description": "Default value for the column",
			},
		},
		"required": []string{"table", "name", "type"},
	}
}

func (t *AddColumnTool) RequiredScopes() []string {
	return []string{mcp.ScopeAdminDDL}
}

func (t *AddColumnTool) Execute(ctx context.Context, args map[string]any, authCtx *mcp.AuthContext) (*mcp.ToolResult, error) {
	schema := "public"
	if s, ok := args["schema"].(string); ok && s != "" {
		schema = s
	}

	table, ok := args["table"].(string)
	if !ok || table == "" {
		return nil, fmt.Errorf("table name is required")
	}

	columnName, ok := args["name"].(string)
	if !ok || columnName == "" {
		return nil, fmt.Errorf("column name is required")
	}

	columnType, ok := args["type"].(string)
	if !ok || columnType == "" {
		return nil, fmt.Errorf("column type is required")
	}

	// Validate identifiers
	if err := validateDDLIdentifier(schema, "schema"); err != nil {
		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.ErrorContent(err.Error())},
			IsError: true,
		}, nil
	}
	if err := validateDDLIdentifier(table, "table"); err != nil {
		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.ErrorContent(err.Error())},
			IsError: true,
		}, nil
	}
	if err := validateDDLIdentifier(columnName, "column"); err != nil {
		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.ErrorContent(err.Error())},
			IsError: true,
		}, nil
	}

	// Block system schemas
	if isSystemSchema(schema) {
		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.ErrorContent(fmt.Sprintf("Cannot add column to table in system schema: %s", schema))},
			IsError: true,
		}, nil
	}

	// Validate data type
	dataType := strings.ToLower(strings.TrimSpace(columnType))
	if !validDataTypes[dataType] {
		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.ErrorContent(fmt.Sprintf("Invalid data type: %s", columnType))},
			IsError: true,
		}, nil
	}

	// Build column definition
	colDef := fmt.Sprintf("%s %s", quoteIdentifier(columnName), dataType)

	nullable := true
	if n, ok := args["nullable"].(bool); ok {
		nullable = n
	}
	if !nullable {
		colDef += " NOT NULL"
	}

	if defaultVal, ok := args["default_value"].(string); ok && defaultVal != "" {
		defaultVal = strings.TrimSpace(defaultVal)
		if defaultVal == "gen_random_uuid()" || defaultVal == "now()" || defaultVal == "current_timestamp" {
			colDef += fmt.Sprintf(" DEFAULT %s", defaultVal)
		} else {
			colDef += fmt.Sprintf(" DEFAULT %s", escapeDDLLiteral(defaultVal))
		}
	}

	query := fmt.Sprintf("ALTER TABLE %s.%s ADD COLUMN %s",
		quoteIdentifier(schema), quoteIdentifier(table), colDef)

	log.Info().
		Str("table", fmt.Sprintf("%s.%s", schema, table)).
		Str("column", columnName).
		Str("query", query).
		Msg("MCP DDL: Adding column")

	err := t.db.ExecuteWithAdminRole(ctx, func(tx pgx.Tx) error {
		_, execErr := tx.Exec(ctx, query)
		return execErr
	})
	if err != nil {
		log.Error().Err(err).Str("table", fmt.Sprintf("%s.%s", schema, table)).Str("column", columnName).Msg("MCP DDL: Failed to add column")
		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.ErrorContent(fmt.Sprintf("Failed to add column: %v", err))},
			IsError: true,
		}, nil
	}

	log.Info().Str("table", fmt.Sprintf("%s.%s", schema, table)).Str("column", columnName).Msg("MCP DDL: Column added successfully")
	resultJSON, _ := json.MarshalIndent(map[string]any{
		"success": true,
		"message": fmt.Sprintf("Column '%s' added to table '%s.%s'", columnName, schema, table),
	}, "", "  ")

	return &mcp.ToolResult{
		Content: []mcp.Content{mcp.TextContent(string(resultJSON))},
	}, nil
}

// DropColumnTool implements the drop_column MCP tool
type DropColumnTool struct {
	db *database.Connection
}

// NewDropColumnTool creates a new drop_column tool
func NewDropColumnTool(db *database.Connection) *DropColumnTool {
	return &DropColumnTool{db: db}
}

func (t *DropColumnTool) Name() string {
	return "drop_column"
}

func (t *DropColumnTool) Description() string {
	return "Remove a column from a table. Requires admin:ddl scope. Use with caution!"
}

func (t *DropColumnTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"schema": map[string]any{
				"type":        "string",
				"description": "Schema name (default: 'public')",
				"default":     "public",
			},
			"table": map[string]any{
				"type":        "string",
				"description": "Table name",
			},
			"column": map[string]any{
				"type":        "string",
				"description": "Column name to drop",
			},
			"cascade": map[string]any{
				"type":        "boolean",
				"description": "Drop dependent objects (CASCADE)",
				"default":     false,
			},
		},
		"required": []string{"table", "column"},
	}
}

func (t *DropColumnTool) RequiredScopes() []string {
	return []string{mcp.ScopeAdminDDL}
}

func (t *DropColumnTool) Execute(ctx context.Context, args map[string]any, authCtx *mcp.AuthContext) (*mcp.ToolResult, error) {
	schema := "public"
	if s, ok := args["schema"].(string); ok && s != "" {
		schema = s
	}

	table, ok := args["table"].(string)
	if !ok || table == "" {
		return nil, fmt.Errorf("table name is required")
	}

	column, ok := args["column"].(string)
	if !ok || column == "" {
		return nil, fmt.Errorf("column name is required")
	}

	cascade := false
	if c, ok := args["cascade"].(bool); ok {
		cascade = c
	}

	// Validate identifiers
	if err := validateDDLIdentifier(schema, "schema"); err != nil {
		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.ErrorContent(err.Error())},
			IsError: true,
		}, nil
	}
	if err := validateDDLIdentifier(table, "table"); err != nil {
		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.ErrorContent(err.Error())},
			IsError: true,
		}, nil
	}
	if err := validateDDLIdentifier(column, "column"); err != nil {
		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.ErrorContent(err.Error())},
			IsError: true,
		}, nil
	}

	// Block system schemas
	if isSystemSchema(schema) {
		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.ErrorContent(fmt.Sprintf("Cannot drop column from table in system schema: %s", schema))},
			IsError: true,
		}, nil
	}

	query := fmt.Sprintf("ALTER TABLE %s.%s DROP COLUMN %s",
		quoteIdentifier(schema), quoteIdentifier(table), quoteIdentifier(column))
	if cascade {
		query += " CASCADE"
	}

	log.Info().
		Str("table", fmt.Sprintf("%s.%s", schema, table)).
		Str("column", column).
		Str("query", query).
		Msg("MCP DDL: Dropping column")

	err := t.db.ExecuteWithAdminRole(ctx, func(tx pgx.Tx) error {
		_, execErr := tx.Exec(ctx, query)
		return execErr
	})
	if err != nil {
		log.Error().Err(err).Str("table", fmt.Sprintf("%s.%s", schema, table)).Str("column", column).Msg("MCP DDL: Failed to drop column")
		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.ErrorContent(fmt.Sprintf("Failed to drop column: %v", err))},
			IsError: true,
		}, nil
	}

	log.Info().Str("table", fmt.Sprintf("%s.%s", schema, table)).Str("column", column).Msg("MCP DDL: Column dropped successfully")
	resultJSON, _ := json.MarshalIndent(map[string]any{
		"success": true,
		"message": fmt.Sprintf("Column '%s' dropped from table '%s.%s'", column, schema, table),
	}, "", "  ")

	return &mcp.ToolResult{
		Content: []mcp.Content{mcp.TextContent(string(resultJSON))},
	}, nil
}
