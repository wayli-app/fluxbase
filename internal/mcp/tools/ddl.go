package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/database"
	"github.com/nimbleflux/fluxbase/internal/mcp"
)

// Validation patterns for DDL operations
var (
	// identifierPattern matches valid PostgreSQL identifiers
	identifierPattern = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

	// reservedKeywords that should not be used as identifiers
	reservedKeywords = map[string]bool{
		"user": true, "table": true, "column": true, "index": true,
		"select": true, "insert": true, "update": true, "delete": true,
		"from": true, "where": true, "group": true, "order": true,
		"limit": true, "offset": true, "join": true, "on": true,
	}

	// validDataTypes for PostgreSQL
	validDataTypes = map[string]bool{
		"text": true, "varchar": true, "char": true,
		"integer": true, "bigint": true, "smallint": true,
		"numeric": true, "decimal": true, "real": true, "double precision": true,
		"boolean": true, "bool": true,
		"date": true, "timestamp": true, "timestamptz": true, "time": true, "timetz": true,
		"uuid": true, "json": true, "jsonb": true,
		"bytea": true, "inet": true, "cidr": true, "macaddr": true,
		"serial": true, "bigserial": true, "smallserial": true,
	}

	// systemSchemas that cannot be modified
	systemSchemas = map[string]bool{
		"auth":               true,
		"storage":            true,
		"jobs":               true,
		"functions":          true,
		"branching":          true,
		"information_schema": true,
		"pg_catalog":         true,
		"pg_toast":           true,
	}
)

// validateIdentifier validates a PostgreSQL identifier (schema/table/column name)
func validateDDLIdentifier(name, entityType string) error {
	if name == "" {
		return fmt.Errorf("%s name cannot be empty", entityType)
	}

	if len(name) > 63 {
		return fmt.Errorf("%s name cannot exceed 63 characters", entityType)
	}

	if !identifierPattern.MatchString(name) {
		return fmt.Errorf("%s name must start with a letter or underscore and contain only letters, numbers, and underscores", entityType)
	}

	if reservedKeywords[strings.ToLower(name)] {
		return fmt.Errorf("'%s' is a reserved keyword and cannot be used as a %s name", name, entityType)
	}

	return nil
}

// isSystemSchema checks if the schema is a system schema
func isSystemSchema(schema string) bool {
	return systemSchemas[strings.ToLower(schema)]
}

// escapeLiteral escapes a string literal for SQL
func escapeDDLLiteral(value string) string {
	escaped := strings.ReplaceAll(value, "'", "''")
	return fmt.Sprintf("'%s'", escaped)
}

// ListSchemasTool implements the list_schemas MCP tool
type ListSchemasTool struct {
	db *database.Connection
}

// NewListSchemasTool creates a new list_schemas tool
func NewListSchemasTool(db *database.Connection) *ListSchemasTool {
	return &ListSchemasTool{db: db}
}

func (t *ListSchemasTool) Name() string {
	return "list_schemas"
}

func (t *ListSchemasTool) Description() string {
	return "List all database schemas. By default excludes system schemas."
}

func (t *ListSchemasTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"include_system": map[string]any{
				"type":        "boolean",
				"description": "Include system schemas (information_schema, pg_catalog, etc.)",
				"default":     false,
			},
		},
	}
}

func (t *ListSchemasTool) RequiredScopes() []string {
	return []string{mcp.ScopeReadTables}
}

func (t *ListSchemasTool) Execute(ctx context.Context, args map[string]any, authCtx *mcp.AuthContext) (*mcp.ToolResult, error) {
	includeSystem := false
	if is, ok := args["include_system"].(bool); ok {
		includeSystem = is
	}

	schemas, err := t.db.Inspector().GetSchemas(ctx)
	if err != nil {
		log.Error().Err(err).Msg("MCP DDL: Failed to list schemas")
		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.ErrorContent(fmt.Sprintf("Failed to list schemas: %v", err))},
			IsError: true,
		}, nil
	}

	type schemaInfo struct {
		Name     string `json:"name"`
		IsSystem bool   `json:"is_system"`
	}

	var result []schemaInfo
	for _, schema := range schemas {
		isSystem := isSystemSchema(schema)
		if !includeSystem && isSystem {
			continue
		}
		result = append(result, schemaInfo{Name: schema, IsSystem: isSystem})
	}

	resultJSON, err := json.MarshalIndent(map[string]any{
		"schemas": result,
		"count":   len(result),
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

// CreateSchemaTool implements the create_schema MCP tool
type CreateSchemaTool struct {
	db *database.Connection
}

// NewCreateSchemaTool creates a new create_schema tool
func NewCreateSchemaTool(db *database.Connection) *CreateSchemaTool {
	return &CreateSchemaTool{db: db}
}

func (t *CreateSchemaTool) Name() string {
	return "create_schema"
}

func (t *CreateSchemaTool) Description() string {
	return "Create a new database schema. Requires admin:ddl scope."
}

func (t *CreateSchemaTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name": map[string]any{
				"type":        "string",
				"description": "Name of the schema to create",
			},
		},
		"required": []string{"name"},
	}
}

func (t *CreateSchemaTool) RequiredScopes() []string {
	return []string{mcp.ScopeAdminDDL}
}

func (t *CreateSchemaTool) Execute(ctx context.Context, args map[string]any, authCtx *mcp.AuthContext) (*mcp.ToolResult, error) {
	name, ok := args["name"].(string)
	if !ok || name == "" {
		return nil, fmt.Errorf("schema name is required")
	}

	if err := validateDDLIdentifier(name, "schema"); err != nil {
		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.ErrorContent(err.Error())},
			IsError: true,
		}, nil
	}

	// Check for system schema names
	if isSystemSchema(name) {
		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.ErrorContent(fmt.Sprintf("Cannot create schema with reserved name: %s", name))},
			IsError: true,
		}, nil
	}

	// Check if schema already exists
	schemas, err := t.db.Inspector().GetSchemas(ctx)
	if err != nil {
		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.ErrorContent(fmt.Sprintf("Failed to check schema existence: %v", err))},
			IsError: true,
		}, nil
	}

	for _, s := range schemas {
		if s == name {
			return &mcp.ToolResult{
				Content: []mcp.Content{mcp.ErrorContent(fmt.Sprintf("Schema '%s' already exists", name))},
				IsError: true,
			}, nil
		}
	}

	query := fmt.Sprintf("CREATE SCHEMA %s", quoteIdentifier(name))
	log.Info().Str("schema", name).Str("query", query).Msg("MCP DDL: Creating schema")

	err = t.db.ExecuteWithAdminRole(ctx, func(tx pgx.Tx) error {
		_, execErr := tx.Exec(ctx, query)
		return execErr
	})
	if err != nil {
		log.Error().Err(err).Str("schema", name).Msg("MCP DDL: Failed to create schema")
		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.ErrorContent(fmt.Sprintf("Failed to create schema: %v", err))},
			IsError: true,
		}, nil
	}

	log.Info().Str("schema", name).Msg("MCP DDL: Schema created successfully")
	resultJSON, _ := json.MarshalIndent(map[string]any{
		"success": true,
		"schema":  name,
		"message": fmt.Sprintf("Schema '%s' created successfully", name),
	}, "", "  ")

	return &mcp.ToolResult{
		Content: []mcp.Content{mcp.TextContent(string(resultJSON))},
	}, nil
}
