package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/fluxbase-eu/fluxbase/internal/database"
	"github.com/fluxbase-eu/fluxbase/internal/mcp"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// MCPResourceReader is an interface for reading MCP resources
type MCPResourceReader interface {
	ReadResource(ctx context.Context, uri string, authCtx *mcp.AuthContext) ([]mcp.Content, error)
}

// SchemaBuilder builds schema descriptions for LLM context
type SchemaBuilder struct {
	db               *database.Connection
	settingsResolver *SettingsResolver
	mcpResources     MCPResourceReader
}

// NewSchemaBuilder creates a new schema builder
func NewSchemaBuilder(db *database.Connection) *SchemaBuilder {
	return &SchemaBuilder{
		db: db,
	}
}

// SetSettingsResolver sets the settings resolver for template variable resolution
func (s *SchemaBuilder) SetSettingsResolver(resolver *SettingsResolver) {
	s.settingsResolver = resolver
}

// GetSettingsResolver returns the settings resolver for template resolution
func (s *SchemaBuilder) GetSettingsResolver() *SettingsResolver {
	return s.settingsResolver
}

// SetMCPResources sets the MCP resource reader for schema fetching
func (s *SchemaBuilder) SetMCPResources(resources MCPResourceReader) {
	s.mcpResources = resources
}

// TableInfo represents information about a database table
type TableInfo struct {
	Schema      string       `json:"schema"`
	Name        string       `json:"name"`
	Description string       `json:"description,omitempty"`
	Columns     []ColumnInfo `json:"columns"`
}

// ColumnInfo represents information about a database column
type ColumnInfo struct {
	Name         string  `json:"name"`
	DataType     string  `json:"data_type"`
	IsNullable   bool    `json:"is_nullable"`
	IsPrimaryKey bool    `json:"is_primary_key"`
	IsForeignKey bool    `json:"is_foreign_key"`
	ForeignTable *string `json:"foreign_table,omitempty"`
	ForeignCol   *string `json:"foreign_column,omitempty"`
	Default      *string `json:"default,omitempty"`
	Description  string  `json:"description,omitempty"`
}

// BuildSchemaDescription builds a text description of the allowed tables
// This is what gets included in the LLM's system prompt
func (s *SchemaBuilder) BuildSchemaDescription(ctx context.Context, allowedSchemas, allowedTables []string) (string, error) {
	// Get table information
	tables, err := s.GetTableInfo(ctx, allowedSchemas, allowedTables)
	if err != nil {
		return "", fmt.Errorf("failed to get table info: %w", err)
	}

	if len(tables) == 0 {
		return "No tables available.", nil
	}

	// Build schema description
	var sb strings.Builder
	sb.WriteString("## Available Database Tables\n\n")

	for _, table := range tables {
		sb.WriteString(fmt.Sprintf("### %s.%s\n", table.Schema, table.Name))
		if table.Description != "" {
			sb.WriteString(fmt.Sprintf("%s\n\n", table.Description))
		}

		sb.WriteString("| Column | Type | Nullable | Notes |\n")
		sb.WriteString("|--------|------|----------|-------|\n")

		for _, col := range table.Columns {
			nullable := "YES"
			if !col.IsNullable {
				nullable = "NO"
			}

			notes := []string{}
			if col.IsPrimaryKey {
				notes = append(notes, "PK")
			}
			if col.IsForeignKey && col.ForeignTable != nil {
				notes = append(notes, fmt.Sprintf("FK → %s.%s", *col.ForeignTable, *col.ForeignCol))
			}
			if col.Description != "" {
				notes = append(notes, col.Description)
			}

			notesStr := strings.Join(notes, ", ")
			sb.WriteString(fmt.Sprintf("| %s | %s | %s | %s |\n",
				col.Name, col.DataType, nullable, notesStr))
		}

		sb.WriteString("\n")
	}

	return sb.String(), nil
}

// GetTableInfo retrieves table information from the database
func (s *SchemaBuilder) GetTableInfo(ctx context.Context, allowedSchemas, allowedTables []string) ([]TableInfo, error) {
	// Build the query to get columns
	query := `
		SELECT
			c.table_schema,
			c.table_name,
			c.column_name,
			c.data_type,
			c.is_nullable = 'YES' as is_nullable,
			c.column_default,
			COALESCE(
				(
					SELECT pg_catalog.col_description(
						(c.table_schema || '.' || c.table_name)::regclass::oid,
						c.ordinal_position
					)
				),
				''
			) as column_description,
			COALESCE(
				(
					SELECT pg_catalog.obj_description(
						(c.table_schema || '.' || c.table_name)::regclass::oid,
						'pg_class'
					)
				),
				''
			) as table_description
		FROM information_schema.columns c
		WHERE c.table_schema = ANY($1)
		  AND ($2::text[] IS NULL OR c.table_name = ANY($2))
		ORDER BY c.table_schema, c.table_name, c.ordinal_position
	`

	// Convert nil arrays to empty arrays for proper NULL handling
	schemas := allowedSchemas
	if len(schemas) == 0 {
		schemas = []string{"public"}
	}

	var tables []string
	if len(allowedTables) > 0 {
		tables = allowedTables
	}

	rows, err := s.db.Query(ctx, query, schemas, tables)
	if err != nil {
		return nil, fmt.Errorf("failed to query columns: %w", err)
	}
	defer rows.Close()

	// Build table map
	tableMap := make(map[string]*TableInfo)

	for rows.Next() {
		var (
			tableSchema       string
			tableName         string
			columnName        string
			dataType          string
			isNullable        bool
			columnDefault     *string
			columnDescription string
			tableDescription  string
		)

		err := rows.Scan(
			&tableSchema, &tableName, &columnName, &dataType,
			&isNullable, &columnDefault, &columnDescription, &tableDescription,
		)
		if err != nil {
			log.Warn().Err(err).Msg("Failed to scan column row")
			continue
		}

		key := tableSchema + "." + tableName

		if _, exists := tableMap[key]; !exists {
			tableMap[key] = &TableInfo{
				Schema:      tableSchema,
				Name:        tableName,
				Description: tableDescription,
				Columns:     []ColumnInfo{},
			}
		}

		tableMap[key].Columns = append(tableMap[key].Columns, ColumnInfo{
			Name:        columnName,
			DataType:    dataType,
			IsNullable:  isNullable,
			Default:     columnDefault,
			Description: columnDescription,
		})
	}

	// Get primary keys
	pkQuery := `
		SELECT
			tc.table_schema,
			tc.table_name,
			kcu.column_name
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu
			ON tc.constraint_name = kcu.constraint_name
			AND tc.table_schema = kcu.table_schema
		WHERE tc.constraint_type = 'PRIMARY KEY'
		  AND tc.table_schema = ANY($1)
		  AND ($2::text[] IS NULL OR tc.table_name = ANY($2))
	`

	pkRows, err := s.db.Query(ctx, pkQuery, schemas, tables)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to query primary keys")
	} else {
		defer pkRows.Close()
		for pkRows.Next() {
			var tableSchema, tableName, columnName string
			if err := pkRows.Scan(&tableSchema, &tableName, &columnName); err != nil {
				continue
			}
			key := tableSchema + "." + tableName
			if table, exists := tableMap[key]; exists {
				for i := range table.Columns {
					if table.Columns[i].Name == columnName {
						table.Columns[i].IsPrimaryKey = true
						break
					}
				}
			}
		}
	}

	// Get foreign keys
	fkQuery := `
		SELECT
			tc.table_schema,
			tc.table_name,
			kcu.column_name,
			ccu.table_schema AS foreign_table_schema,
			ccu.table_name AS foreign_table_name,
			ccu.column_name AS foreign_column_name
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu
			ON tc.constraint_name = kcu.constraint_name
			AND tc.table_schema = kcu.table_schema
		JOIN information_schema.constraint_column_usage ccu
			ON ccu.constraint_name = tc.constraint_name
		WHERE tc.constraint_type = 'FOREIGN KEY'
		  AND tc.table_schema = ANY($1)
		  AND ($2::text[] IS NULL OR tc.table_name = ANY($2))
	`

	fkRows, err := s.db.Query(ctx, fkQuery, schemas, tables)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to query foreign keys")
	} else {
		defer fkRows.Close()
		for fkRows.Next() {
			var tableSchema, tableName, columnName string
			var foreignSchema, foreignTable, foreignColumn string
			if err := fkRows.Scan(&tableSchema, &tableName, &columnName,
				&foreignSchema, &foreignTable, &foreignColumn); err != nil {
				continue
			}
			key := tableSchema + "." + tableName
			if table, exists := tableMap[key]; exists {
				for i := range table.Columns {
					if table.Columns[i].Name == columnName {
						table.Columns[i].IsForeignKey = true
						fkTable := foreignSchema + "." + foreignTable
						table.Columns[i].ForeignTable = &fkTable
						table.Columns[i].ForeignCol = &foreignColumn
						break
					}
				}
			}
		}
	}

	// Convert map to slice
	result := make([]TableInfo, 0, len(tableMap))
	for _, table := range tableMap {
		result = append(result, *table)
	}

	return result, nil
}

// BuildSystemPrompt builds the complete system prompt for a chatbot
func (s *SchemaBuilder) BuildSystemPrompt(ctx context.Context, chatbot *Chatbot, userID string) (string, error) {
	return s.BuildSystemPromptWithAuth(ctx, chatbot, userID, nil)
}

// BuildSystemPromptWithAuth builds the complete system prompt for a chatbot with MCP auth context
// When authCtx is provided and chatbot.UseMCPSchema is true, schema is fetched from MCP resources
func (s *SchemaBuilder) BuildSystemPromptWithAuth(ctx context.Context, chatbot *Chatbot, userID string, authCtx *mcp.AuthContext) (string, error) {
	var schemaDesc string
	var err error

	// Use MCP schema if enabled and auth context is available
	if chatbot.UseMCPSchema && authCtx != nil && s.mcpResources != nil {
		schemaDesc, err = s.BuildSchemaDescriptionFromMCP(ctx, chatbot.AllowedSchemas, chatbot.AllowedTables, authCtx)
		if err != nil {
			// Fall back to direct DB introspection on error
			log.Warn().Err(err).Msg("Failed to fetch schema from MCP, falling back to direct DB introspection")
			schemaDesc, err = s.BuildSchemaDescription(ctx, chatbot.AllowedSchemas, chatbot.AllowedTables)
		}
	} else {
		// Get schema description via direct DB introspection
		schemaDesc, err = s.BuildSchemaDescription(ctx, chatbot.AllowedSchemas, chatbot.AllowedTables)
	}
	if err != nil {
		return "", fmt.Errorf("failed to build schema description: %w", err)
	}

	// Extract the system prompt from the chatbot code
	userPrompt := ParseSystemPrompt(chatbot.Code)
	if userPrompt == "" {
		userPrompt = "You are a helpful AI assistant that can query the database to answer questions."
	}

	// Replace built-in template variables
	userPrompt = strings.ReplaceAll(userPrompt, "{{user_id}}", userID)

	// Resolve settings template variables ({{key}}, {{user:key}}, {{system:key}})
	if s.settingsResolver != nil {
		var userUUID *uuid.UUID
		if userID != "" {
			if parsed, err := uuid.Parse(userID); err == nil {
				userUUID = &parsed
			}
		}

		resolved, err := s.settingsResolver.ResolveTemplate(ctx, userPrompt, userUUID)
		if err != nil {
			log.Warn().Err(err).Msg("Failed to resolve settings in system prompt")
			// Continue with unresolved template - don't fail the request
		} else {
			userPrompt = resolved
		}
	}

	// Build the complete system prompt
	var sb strings.Builder

	sb.WriteString(userPrompt)
	sb.WriteString("\n\n")
	sb.WriteString(schemaDesc)
	sb.WriteString("\n")
	sb.WriteString("## Query Guidelines\n\n")
	sb.WriteString("1. Only use SELECT queries unless explicitly allowed other operations.\n")
	sb.WriteString("2. Always include appropriate LIMIT clauses (max 100 rows).\n")
	sb.WriteString("3. Filter by user_id when querying user-specific data.\n")
	sb.WriteString("4. Describe what you're querying before executing.\n")
	sb.WriteString("5. You will receive a summary of results, not raw data.\n")
	sb.WriteString("6. You can execute multiple queries if needed to answer complex questions.\n")
	sb.WriteString("7. For questions spanning multiple tables, query each table separately and combine the insights in your response.\n")

	// Add default table hint if configured
	if chatbot.DefaultTable != "" {
		sb.WriteString(fmt.Sprintf("\n**Default/Primary table**: %s - Use this table first unless the question specifically requires a different table.\n", chatbot.DefaultTable))
	}

	// Add intent rules hints if configured
	if len(chatbot.IntentRules) > 0 {
		// Check if any rules have table constraints
		hasTableRules := false
		hasToolRules := false
		for _, rule := range chatbot.IntentRules {
			if rule.RequiredTable != "" || rule.ForbiddenTable != "" {
				hasTableRules = true
			}
			if rule.RequiredTool != "" || rule.ForbiddenTool != "" {
				hasToolRules = true
			}
		}

		// Add table selection guidelines
		if hasTableRules {
			sb.WriteString("\n## Table Selection Guidelines\n\n")
			sb.WriteString("Use the correct table based on what the user is asking about:\n")
			for _, rule := range chatbot.IntentRules {
				if rule.RequiredTable != "" {
					sb.WriteString(fmt.Sprintf("- For questions about %s → use **%s**\n",
						strings.Join(rule.Keywords, ", "), rule.RequiredTable))
				}
				if rule.ForbiddenTable != "" {
					sb.WriteString(fmt.Sprintf("- Do NOT use '%s' for queries about %s\n",
						rule.ForbiddenTable, strings.Join(rule.Keywords, ", ")))
				}
			}
		}

		// Add tool selection guidelines
		if hasToolRules {
			sb.WriteString("\n## Tool Selection Guidelines\n\n")
			sb.WriteString("Use the correct tool based on what the user is asking about:\n")
			for _, rule := range chatbot.IntentRules {
				if rule.RequiredTool != "" {
					sb.WriteString(fmt.Sprintf("- For questions about %s → use the **%s** tool\n",
						strings.Join(rule.Keywords, ", "), rule.RequiredTool))
				}
				if rule.ForbiddenTool != "" {
					sb.WriteString(fmt.Sprintf("- Do NOT use the '%s' tool for queries about %s\n",
						rule.ForbiddenTool, strings.Join(rule.Keywords, ", ")))
				}
			}
		}
	}

	// Add required columns hints if configured
	if len(chatbot.RequiredColumns) > 0 {
		sb.WriteString("\n## Required Columns\n\n")
		sb.WriteString("When querying these tables, always include these columns:\n")
		for table, cols := range chatbot.RequiredColumns {
			sb.WriteString(fmt.Sprintf("- **%s**: %s\n", table, strings.Join(cols, ", ")))
		}
	}

	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("Allowed operations: %s\n", strings.Join(chatbot.AllowedOperations, ", ")))
	sb.WriteString(fmt.Sprintf("Current user ID: %s\n", userID))
	sb.WriteString(fmt.Sprintf("Current date and time: %s\n", time.Now().UTC().Format("Monday, January 2, 2006 at 3:04 PM MST")))

	// Add response language instruction
	sb.WriteString("\n## Response Language\n\n")
	if chatbot.ResponseLanguage == "" || chatbot.ResponseLanguage == "auto" {
		sb.WriteString("IMPORTANT: Always respond in the same language as the user's message. ")
		sb.WriteString("Detect the language of each user message and reply in that exact language.\n")
	} else {
		sb.WriteString(fmt.Sprintf("IMPORTANT: Always respond in %s, regardless of the language the user writes in.\n",
			chatbot.ResponseLanguage))
	}

	return sb.String(), nil
}

// GetCompactSchemaDescription returns a compact schema description for token efficiency
func (s *SchemaBuilder) GetCompactSchemaDescription(ctx context.Context, allowedSchemas, allowedTables []string) (string, error) {
	tables, err := s.GetTableInfo(ctx, allowedSchemas, allowedTables)
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	for _, table := range tables {
		sb.WriteString(fmt.Sprintf("%s.%s: ", table.Schema, table.Name))
		cols := make([]string, len(table.Columns))
		for i, col := range table.Columns {
			colDesc := col.Name + " (" + col.DataType
			if col.IsPrimaryKey {
				colDesc += ", PK"
			}
			if col.IsForeignKey && col.ForeignTable != nil {
				colDesc += ", FK→" + *col.ForeignTable
			}
			colDesc += ")"
			cols[i] = colDesc
		}
		sb.WriteString(strings.Join(cols, ", "))
		sb.WriteString("\n")
	}

	return sb.String(), nil
}

// GetTableInfoFromMCP fetches table information from MCP schema resources
// This provides cached schema data instead of querying the database directly
func (s *SchemaBuilder) GetTableInfoFromMCP(ctx context.Context, allowedSchemas, allowedTables []string, authCtx *mcp.AuthContext) ([]TableInfo, error) {
	if s.mcpResources == nil {
		return nil, fmt.Errorf("MCP resources not configured")
	}

	// Read schema from MCP resource
	contents, err := s.mcpResources.ReadResource(ctx, "fluxbase://schema/tables", authCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to read MCP schema resource: %w", err)
	}

	if len(contents) == 0 {
		return nil, fmt.Errorf("empty MCP schema response")
	}

	// Parse the JSON schema data
	var schemaData map[string]json.RawMessage
	if err := json.Unmarshal([]byte(contents[0].Text), &schemaData); err != nil {
		return nil, fmt.Errorf("failed to parse MCP schema: %w", err)
	}

	// Build filter sets for efficient lookup
	allowedSchemaSet := make(map[string]bool)
	for _, schema := range allowedSchemas {
		allowedSchemaSet[schema] = true
	}
	if len(allowedSchemaSet) == 0 {
		allowedSchemaSet["public"] = true
	}

	// Parse qualified table names to handle schema.table format
	qualifiedTables := ParseQualifiedTables(allowedTables, "public")
	tablesBySchema := GroupTablesBySchema(qualifiedTables)

	// Convert MCP schema format to TableInfo
	var tables []TableInfo
	for tableKey, tableJSON := range schemaData {
		var tableData struct {
			Schema  string `json:"schema"`
			Name    string `json:"name"`
			Columns []struct {
				Name       string  `json:"name"`
				Type       string  `json:"type"`
				Nullable   bool    `json:"nullable"`
				Default    *string `json:"default,omitempty"`
				PrimaryKey bool    `json:"primary_key,omitempty"`
			} `json:"columns"`
			PrimaryKey  []string `json:"primary_key,omitempty"`
			ForeignKeys []struct {
				Name             string `json:"name"`
				Column           string `json:"column"`
				ReferencedTable  string `json:"referenced_table"`
				ReferencedColumn string `json:"referenced_column"`
			} `json:"foreign_keys,omitempty"`
		}

		if err := json.Unmarshal(tableJSON, &tableData); err != nil {
			log.Warn().Str("table", tableKey).Err(err).Msg("Failed to parse table data from MCP schema")
			continue
		}

		// Check if this table should be included
		if !s.isTableAllowed(tableData.Schema, tableData.Name, allowedSchemaSet, tablesBySchema) {
			continue
		}

		// Build foreign key map for column lookup
		fkMap := make(map[string]struct {
			Table  string
			Column string
		})
		for _, fk := range tableData.ForeignKeys {
			fkMap[fk.Column] = struct {
				Table  string
				Column string
			}{
				Table:  fk.ReferencedTable,
				Column: fk.ReferencedColumn,
			}
		}

		// Convert columns
		columns := make([]ColumnInfo, 0, len(tableData.Columns))
		for _, col := range tableData.Columns {
			colInfo := ColumnInfo{
				Name:         col.Name,
				DataType:     col.Type,
				IsNullable:   col.Nullable,
				IsPrimaryKey: col.PrimaryKey,
				Default:      col.Default,
			}

			// Check if column is a foreign key
			if fk, exists := fkMap[col.Name]; exists {
				colInfo.IsForeignKey = true
				colInfo.ForeignTable = &fk.Table
				colInfo.ForeignCol = &fk.Column
			}

			columns = append(columns, colInfo)
		}

		tables = append(tables, TableInfo{
			Schema:  tableData.Schema,
			Name:    tableData.Name,
			Columns: columns,
		})
	}

	return tables, nil
}

// isTableAllowed checks if a table should be included based on allowed schemas and tables
func (s *SchemaBuilder) isTableAllowed(schema, tableName string, allowedSchemaSet map[string]bool, tablesBySchema map[string][]string) bool {
	// If specific tables are configured for this schema, check against them
	if tables, exists := tablesBySchema[schema]; exists && len(tables) > 0 {
		for _, t := range tables {
			if t == tableName {
				return true
			}
		}
		return false
	}

	// If the schema is in the allowed list (and no specific tables), allow all tables in that schema
	if allowedSchemaSet[schema] {
		return true
	}

	return false
}

// BuildSchemaDescriptionFromMCP builds schema description using MCP resources
func (s *SchemaBuilder) BuildSchemaDescriptionFromMCP(ctx context.Context, allowedSchemas, allowedTables []string, authCtx *mcp.AuthContext) (string, error) {
	tables, err := s.GetTableInfoFromMCP(ctx, allowedSchemas, allowedTables, authCtx)
	if err != nil {
		return "", err
	}

	if len(tables) == 0 {
		return "No tables available.", nil
	}

	// Build schema description using the same format as BuildSchemaDescription
	var sb strings.Builder
	sb.WriteString("## Available Database Tables\n\n")

	for _, table := range tables {
		sb.WriteString(fmt.Sprintf("### %s.%s\n", table.Schema, table.Name))
		if table.Description != "" {
			sb.WriteString(fmt.Sprintf("%s\n\n", table.Description))
		}

		sb.WriteString("| Column | Type | Nullable | Notes |\n")
		sb.WriteString("|--------|------|----------|-------|\n")

		for _, col := range table.Columns {
			nullable := "YES"
			if !col.IsNullable {
				nullable = "NO"
			}

			notes := []string{}
			if col.IsPrimaryKey {
				notes = append(notes, "PK")
			}
			if col.IsForeignKey && col.ForeignTable != nil {
				notes = append(notes, fmt.Sprintf("FK → %s.%s", *col.ForeignTable, *col.ForeignCol))
			}
			if col.Description != "" {
				notes = append(notes, col.Description)
			}

			notesStr := strings.Join(notes, ", ")
			sb.WriteString(fmt.Sprintf("| %s | %s | %s | %s |\n",
				col.Name, col.DataType, nullable, notesStr))
		}

		sb.WriteString("\n")
	}

	return sb.String(), nil
}
