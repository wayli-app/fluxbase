package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/fluxbase-eu/fluxbase/internal/database"
	"github.com/fluxbase-eu/fluxbase/internal/mcp"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"
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

	err = t.db.ExecuteWithAdminRole(ctx, func(conn *pgx.Conn) error {
		_, execErr := conn.Exec(ctx, query)
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

// CreateTableTool implements the create_table MCP tool
type CreateTableTool struct {
	db *database.Connection
}

// NewCreateTableTool creates a new create_table tool
func NewCreateTableTool(db *database.Connection) *CreateTableTool {
	return &CreateTableTool{db: db}
}

func (t *CreateTableTool) Name() string {
	return "create_table"
}

func (t *CreateTableTool) Description() string {
	return "Create a new database table with specified columns. Requires admin:ddl scope."
}

func (t *CreateTableTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"schema": map[string]any{
				"type":        "string",
				"description": "Schema name (default: 'public')",
				"default":     "public",
			},
			"name": map[string]any{
				"type":        "string",
				"description": "Table name",
			},
			"columns": map[string]any{
				"type":        "array",
				"description": "Column definitions",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"name": map[string]any{
							"type":        "string",
							"description": "Column name",
						},
						"type": map[string]any{
							"type":        "string",
							"description": "PostgreSQL data type (e.g., 'text', 'integer', 'uuid', 'timestamptz')",
						},
						"nullable": map[string]any{
							"type":        "boolean",
							"description": "Whether the column can be NULL (default: true)",
							"default":     true,
						},
						"default_value": map[string]any{
							"type":        "string",
							"description": "Default value (e.g., 'gen_random_uuid()', 'now()', or a literal value)",
						},
						"primary_key": map[string]any{
							"type":        "boolean",
							"description": "Whether this column is part of the primary key",
							"default":     false,
						},
					},
					"required": []string{"name", "type"},
				},
			},
		},
		"required": []string{"name", "columns"},
	}
}

func (t *CreateTableTool) RequiredScopes() []string {
	return []string{mcp.ScopeAdminDDL}
}

func (t *CreateTableTool) Execute(ctx context.Context, args map[string]any, authCtx *mcp.AuthContext) (*mcp.ToolResult, error) {
	schema := "public"
	if s, ok := args["schema"].(string); ok && s != "" {
		schema = s
	}

	name, ok := args["name"].(string)
	if !ok || name == "" {
		return nil, fmt.Errorf("table name is required")
	}

	columnsRaw, ok := args["columns"].([]any)
	if !ok || len(columnsRaw) == 0 {
		return nil, fmt.Errorf("at least one column is required")
	}

	// Validate schema
	if err := validateDDLIdentifier(schema, "schema"); err != nil {
		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.ErrorContent(err.Error())},
			IsError: true,
		}, nil
	}

	// Block system schemas
	if isSystemSchema(schema) {
		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.ErrorContent(fmt.Sprintf("Cannot create table in system schema: %s", schema))},
			IsError: true,
		}, nil
	}

	// Validate table name
	if err := validateDDLIdentifier(name, "table"); err != nil {
		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.ErrorContent(err.Error())},
			IsError: true,
		}, nil
	}

	// Build column definitions
	var columnDefs []string
	var primaryKeys []string

	for i, colRaw := range columnsRaw {
		col, ok := colRaw.(map[string]any)
		if !ok {
			return &mcp.ToolResult{
				Content: []mcp.Content{mcp.ErrorContent(fmt.Sprintf("Column %d: invalid format", i+1))},
				IsError: true,
			}, nil
		}

		colName, ok := col["name"].(string)
		if !ok || colName == "" {
			return &mcp.ToolResult{
				Content: []mcp.Content{mcp.ErrorContent(fmt.Sprintf("Column %d: name is required", i+1))},
				IsError: true,
			}, nil
		}

		colType, ok := col["type"].(string)
		if !ok || colType == "" {
			return &mcp.ToolResult{
				Content: []mcp.Content{mcp.ErrorContent(fmt.Sprintf("Column %d: type is required", i+1))},
				IsError: true,
			}, nil
		}

		// Validate column name
		if err := validateDDLIdentifier(colName, "column"); err != nil {
			return &mcp.ToolResult{
				Content: []mcp.Content{mcp.ErrorContent(fmt.Sprintf("Column '%s': %v", colName, err))},
				IsError: true,
			}, nil
		}

		// Validate data type
		dataType := strings.ToLower(strings.TrimSpace(colType))
		if !validDataTypes[dataType] {
			return &mcp.ToolResult{
				Content: []mcp.Content{mcp.ErrorContent(fmt.Sprintf("Column '%s': invalid data type '%s'", colName, colType))},
				IsError: true,
			}, nil
		}

		// Build column definition
		colDef := fmt.Sprintf("%s %s", quoteIdentifier(colName), dataType)

		// Add NOT NULL constraint
		nullable := true
		if n, ok := col["nullable"].(bool); ok {
			nullable = n
		}
		if !nullable {
			colDef += " NOT NULL"
		}

		// Add DEFAULT value
		if defaultVal, ok := col["default_value"].(string); ok && defaultVal != "" {
			defaultVal = strings.TrimSpace(defaultVal)
			// Allow safe function calls
			if defaultVal == "gen_random_uuid()" || defaultVal == "now()" || defaultVal == "current_timestamp" {
				colDef += fmt.Sprintf(" DEFAULT %s", defaultVal)
			} else {
				colDef += fmt.Sprintf(" DEFAULT %s", escapeDDLLiteral(defaultVal))
			}
		}

		columnDefs = append(columnDefs, colDef)

		// Track primary keys
		if pk, ok := col["primary_key"].(bool); ok && pk {
			primaryKeys = append(primaryKeys, quoteIdentifier(colName))
		}
	}

	// Add PRIMARY KEY constraint if any
	if len(primaryKeys) > 0 {
		columnDefs = append(columnDefs, fmt.Sprintf("PRIMARY KEY (%s)", strings.Join(primaryKeys, ", ")))
	}

	// Build CREATE TABLE statement
	query := fmt.Sprintf(
		"CREATE TABLE %s.%s (\n  %s\n)",
		quoteIdentifier(schema),
		quoteIdentifier(name),
		strings.Join(columnDefs, ",\n  "),
	)

	log.Info().
		Str("table", fmt.Sprintf("%s.%s", schema, name)).
		Str("query", query).
		Int("columns", len(columnsRaw)).
		Msg("MCP DDL: Creating table")

	err := t.db.ExecuteWithAdminRole(ctx, func(conn *pgx.Conn) error {
		_, execErr := conn.Exec(ctx, query)
		return execErr
	})
	if err != nil {
		log.Error().Err(err).Str("table", fmt.Sprintf("%s.%s", schema, name)).Msg("MCP DDL: Failed to create table")
		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.ErrorContent(fmt.Sprintf("Failed to create table: %v", err))},
			IsError: true,
		}, nil
	}

	log.Info().Str("table", fmt.Sprintf("%s.%s", schema, name)).Msg("MCP DDL: Table created successfully")
	resultJSON, _ := json.MarshalIndent(map[string]any{
		"success": true,
		"schema":  schema,
		"table":   name,
		"message": fmt.Sprintf("Table '%s.%s' created successfully", schema, name),
	}, "", "  ")

	return &mcp.ToolResult{
		Content: []mcp.Content{mcp.TextContent(string(resultJSON))},
	}, nil
}

// DropTableTool implements the drop_table MCP tool
type DropTableTool struct {
	db *database.Connection
}

// NewDropTableTool creates a new drop_table tool
func NewDropTableTool(db *database.Connection) *DropTableTool {
	return &DropTableTool{db: db}
}

func (t *DropTableTool) Name() string {
	return "drop_table"
}

func (t *DropTableTool) Description() string {
	return "Drop (delete) a database table. Requires admin:ddl scope. Use with caution!"
}

func (t *DropTableTool) InputSchema() map[string]any {
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
				"description": "Table name to drop",
			},
			"cascade": map[string]any{
				"type":        "boolean",
				"description": "Drop dependent objects (CASCADE)",
				"default":     false,
			},
		},
		"required": []string{"table"},
	}
}

func (t *DropTableTool) RequiredScopes() []string {
	return []string{mcp.ScopeAdminDDL}
}

func (t *DropTableTool) Execute(ctx context.Context, args map[string]any, authCtx *mcp.AuthContext) (*mcp.ToolResult, error) {
	schema := "public"
	if s, ok := args["schema"].(string); ok && s != "" {
		schema = s
	}

	table, ok := args["table"].(string)
	if !ok || table == "" {
		return nil, fmt.Errorf("table name is required")
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

	// Block system schemas
	if isSystemSchema(schema) {
		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.ErrorContent(fmt.Sprintf("Cannot drop table from system schema: %s", schema))},
			IsError: true,
		}, nil
	}

	// Check if table exists
	tables, err := t.db.Inspector().GetAllTables(ctx, schema)
	if err != nil {
		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.ErrorContent(fmt.Sprintf("Failed to check table existence: %v", err))},
			IsError: true,
		}, nil
	}

	found := false
	for _, tbl := range tables {
		if tbl.Name == table {
			found = true
			break
		}
	}
	if !found {
		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.ErrorContent(fmt.Sprintf("Table '%s.%s' does not exist", schema, table))},
			IsError: true,
		}, nil
	}

	query := fmt.Sprintf("DROP TABLE %s.%s", quoteIdentifier(schema), quoteIdentifier(table))
	if cascade {
		query += " CASCADE"
	}

	log.Info().Str("table", fmt.Sprintf("%s.%s", schema, table)).Str("query", query).Msg("MCP DDL: Dropping table")

	err = t.db.ExecuteWithAdminRole(ctx, func(conn *pgx.Conn) error {
		_, execErr := conn.Exec(ctx, query)
		return execErr
	})
	if err != nil {
		log.Error().Err(err).Str("table", fmt.Sprintf("%s.%s", schema, table)).Msg("MCP DDL: Failed to drop table")
		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.ErrorContent(fmt.Sprintf("Failed to drop table: %v", err))},
			IsError: true,
		}, nil
	}

	log.Info().Str("table", fmt.Sprintf("%s.%s", schema, table)).Msg("MCP DDL: Table dropped successfully")
	resultJSON, _ := json.MarshalIndent(map[string]any{
		"success": true,
		"message": fmt.Sprintf("Table '%s.%s' dropped successfully", schema, table),
	}, "", "  ")

	return &mcp.ToolResult{
		Content: []mcp.Content{mcp.TextContent(string(resultJSON))},
	}, nil
}

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

	err := t.db.ExecuteWithAdminRole(ctx, func(conn *pgx.Conn) error {
		_, execErr := conn.Exec(ctx, query)
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

	err := t.db.ExecuteWithAdminRole(ctx, func(conn *pgx.Conn) error {
		_, execErr := conn.Exec(ctx, query)
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

// RenameTableTool implements the rename_table MCP tool
type RenameTableTool struct {
	db *database.Connection
}

// NewRenameTableTool creates a new rename_table tool
func NewRenameTableTool(db *database.Connection) *RenameTableTool {
	return &RenameTableTool{db: db}
}

func (t *RenameTableTool) Name() string {
	return "rename_table"
}

func (t *RenameTableTool) Description() string {
	return "Rename an existing table. Requires admin:ddl scope."
}

func (t *RenameTableTool) InputSchema() map[string]any {
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
				"description": "Current table name",
			},
			"new_name": map[string]any{
				"type":        "string",
				"description": "New table name",
			},
		},
		"required": []string{"table", "new_name"},
	}
}

func (t *RenameTableTool) RequiredScopes() []string {
	return []string{mcp.ScopeAdminDDL}
}

func (t *RenameTableTool) Execute(ctx context.Context, args map[string]any, authCtx *mcp.AuthContext) (*mcp.ToolResult, error) {
	schema := "public"
	if s, ok := args["schema"].(string); ok && s != "" {
		schema = s
	}

	table, ok := args["table"].(string)
	if !ok || table == "" {
		return nil, fmt.Errorf("table name is required")
	}

	newName, ok := args["new_name"].(string)
	if !ok || newName == "" {
		return nil, fmt.Errorf("new table name is required")
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
	if err := validateDDLIdentifier(newName, "table"); err != nil {
		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.ErrorContent(fmt.Sprintf("New name: %v", err))},
			IsError: true,
		}, nil
	}

	// Block system schemas
	if isSystemSchema(schema) {
		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.ErrorContent(fmt.Sprintf("Cannot rename table in system schema: %s", schema))},
			IsError: true,
		}, nil
	}

	// Check if table exists
	tables, err := t.db.Inspector().GetAllTables(ctx, schema)
	if err != nil {
		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.ErrorContent(fmt.Sprintf("Failed to check table existence: %v", err))},
			IsError: true,
		}, nil
	}

	found := false
	targetExists := false
	for _, tbl := range tables {
		if tbl.Name == table {
			found = true
		}
		if tbl.Name == newName {
			targetExists = true
		}
	}
	if !found {
		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.ErrorContent(fmt.Sprintf("Table '%s.%s' does not exist", schema, table))},
			IsError: true,
		}, nil
	}
	if targetExists {
		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.ErrorContent(fmt.Sprintf("Table '%s.%s' already exists", schema, newName))},
			IsError: true,
		}, nil
	}

	query := fmt.Sprintf("ALTER TABLE %s.%s RENAME TO %s",
		quoteIdentifier(schema), quoteIdentifier(table), quoteIdentifier(newName))

	log.Info().
		Str("table", fmt.Sprintf("%s.%s", schema, table)).
		Str("newName", newName).
		Str("query", query).
		Msg("MCP DDL: Renaming table")

	err = t.db.ExecuteWithAdminRole(ctx, func(conn *pgx.Conn) error {
		_, execErr := conn.Exec(ctx, query)
		return execErr
	})
	if err != nil {
		log.Error().Err(err).Str("table", fmt.Sprintf("%s.%s", schema, table)).Str("newName", newName).Msg("MCP DDL: Failed to rename table")
		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.ErrorContent(fmt.Sprintf("Failed to rename table: %v", err))},
			IsError: true,
		}, nil
	}

	log.Info().Str("table", fmt.Sprintf("%s.%s", schema, table)).Str("newName", newName).Msg("MCP DDL: Table renamed successfully")
	resultJSON, _ := json.MarshalIndent(map[string]any{
		"success": true,
		"message": fmt.Sprintf("Table '%s.%s' renamed to '%s.%s'", schema, table, schema, newName),
	}, "", "  ")

	return &mcp.ToolResult{
		Content: []mcp.Content{mcp.TextContent(string(resultJSON))},
	}, nil
}
