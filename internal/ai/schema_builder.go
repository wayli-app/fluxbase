package ai

import (
	"context"
	"fmt"
	"strings"

	"github.com/fluxbase-eu/fluxbase/internal/database"
	"github.com/rs/zerolog/log"
)

// SchemaBuilder builds schema descriptions for LLM context
type SchemaBuilder struct {
	db *database.Connection
}

// NewSchemaBuilder creates a new schema builder
func NewSchemaBuilder(db *database.Connection) *SchemaBuilder {
	return &SchemaBuilder{
		db: db,
	}
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
	// Get schema description for allowed tables
	schemaDesc, err := s.BuildSchemaDescription(ctx, chatbot.AllowedSchemas, chatbot.AllowedTables)
	if err != nil {
		return "", fmt.Errorf("failed to build schema description: %w", err)
	}

	// Extract the system prompt from the chatbot code
	userPrompt := ParseSystemPrompt(chatbot.Code)
	if userPrompt == "" {
		userPrompt = "You are a helpful AI assistant that can query the database to answer questions."
	}

	// Replace template variables
	userPrompt = strings.ReplaceAll(userPrompt, "{{user_id}}", userID)

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
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("Allowed operations: %s\n", strings.Join(chatbot.AllowedOperations, ", ")))
	sb.WriteString(fmt.Sprintf("Current user ID: %s\n", userID))

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
