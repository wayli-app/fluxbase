package api

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/fluxbase-eu/fluxbase/internal/database"
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"
)

// DDLHandler handles Database Definition Language (DDL) operations
// for schema and table management
type DDLHandler struct {
	db *database.Connection
}

// NewDDLHandler creates a new DDL handler
func NewDDLHandler(db *database.Connection) *DDLHandler {
	return &DDLHandler{db: db}
}

// Validation patterns
var (
	// identifierPattern matches valid PostgreSQL identifiers (schema/table/column names)
	// Must start with letter or underscore, followed by letters, numbers, underscores
	identifierPattern = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

	// Reserved PostgreSQL keywords that should not be used as identifiers
	reservedKeywords = map[string]bool{
		"user": true, "table": true, "column": true, "index": true,
		"select": true, "insert": true, "update": true, "delete": true,
		"from": true, "where": true, "group": true, "order": true,
		"limit": true, "offset": true, "join": true, "on": true,
	}

	// Valid PostgreSQL data types
	validDataTypes = map[string]bool{
		"text": true, "varchar": true, "char": true,
		"integer": true, "bigint": true, "smallint": true,
		"numeric": true, "decimal": true, "real": true, "double precision": true,
		"boolean": true, "bool": true,
		"date": true, "timestamp": true, "timestamptz": true, "time": true, "timetz": true,
		"uuid": true, "json": true, "jsonb": true,
		"bytea": true, "inet": true, "cidr": true, "macaddr": true,
	}
)

// CreateSchemaRequest represents a request to create a new schema
type CreateSchemaRequest struct {
	Name string `json:"name"`
}

// CreateTableRequest represents a request to create a new table
type CreateTableRequest struct {
	Schema  string                `json:"schema"`
	Name    string                `json:"name"`
	Columns []CreateColumnRequest `json:"columns"`
}

// CreateColumnRequest represents a column definition
type CreateColumnRequest struct {
	Name         string `json:"name"`
	Type         string `json:"type"`
	Nullable     bool   `json:"nullable"`
	PrimaryKey   bool   `json:"primaryKey"`
	DefaultValue string `json:"defaultValue"`
}

// CreateSchema creates a new database schema
func (h *DDLHandler) CreateSchema(c *fiber.Ctx) error {
	var req CreateSchemaRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate schema name
	if err := validateIdentifier(req.Name, "schema"); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	ctx := c.Context()

	// Check if schema already exists
	exists, err := h.schemaExists(ctx, req.Name)
	if err != nil {
		log.Error().Err(err).Str("schema", req.Name).Msg("Failed to check schema existence")
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to check schema existence",
		})
	}
	if exists {
		return c.Status(409).JSON(fiber.Map{
			"error": fmt.Sprintf("Schema '%s' already exists", req.Name),
		})
	}

	// Create schema (using quoted identifier for safety)
	// Use admin role to ensure full DDL access (superuser privileges)
	query := fmt.Sprintf("CREATE SCHEMA %s", quoteIdentifier(req.Name))
	log.Info().Str("schema", req.Name).Str("query", query).Msg("Creating schema")

	err = h.db.ExecuteWithAdminRole(ctx, func(conn *pgx.Conn) error {
		_, execErr := conn.Exec(ctx, query)
		return execErr
	})
	if err != nil {
		log.Error().Err(err).Str("schema", req.Name).Msg("Failed to create schema")
		return c.Status(500).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to create schema: %v", err),
		})
	}

	log.Info().Str("schema", req.Name).Msg("Schema created successfully")
	return c.Status(201).JSON(fiber.Map{
		"success": true,
		"schema":  req.Name,
		"message": fmt.Sprintf("Schema '%s' created successfully", req.Name),
	})
}

// CreateTable creates a new table with specified columns
func (h *DDLHandler) CreateTable(c *fiber.Ctx) error {
	var req CreateTableRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate schema name
	if err := validateIdentifier(req.Schema, "schema"); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Validate table name
	if err := validateIdentifier(req.Name, "table"); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Validate columns
	if len(req.Columns) == 0 {
		return c.Status(400).JSON(fiber.Map{
			"error": "At least one column is required",
		})
	}

	ctx := c.Context()

	// Check if schema exists
	exists, err := h.schemaExists(ctx, req.Schema)
	if err != nil {
		log.Error().Err(err).Str("schema", req.Schema).Msg("Failed to check schema existence")
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to check schema existence",
		})
	}
	if !exists {
		return c.Status(404).JSON(fiber.Map{
			"error": fmt.Sprintf("Schema '%s' does not exist", req.Schema),
		})
	}

	// Check if table already exists
	tableExists, err := h.tableExists(ctx, req.Schema, req.Name)
	if err != nil {
		log.Error().Err(err).Str("table", req.Schema+"."+req.Name).Msg("Failed to check table existence")
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to check table existence",
		})
	}
	if tableExists {
		return c.Status(409).JSON(fiber.Map{
			"error": fmt.Sprintf("Table '%s.%s' already exists", req.Schema, req.Name),
		})
	}

	// Build CREATE TABLE statement
	query, err := h.buildCreateTableQuery(req)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	log.Info().
		Str("table", req.Schema+"."+req.Name).
		Str("query", query).
		Int("columns", len(req.Columns)).
		Msg("Creating table")

	// Execute CREATE TABLE with admin role for full DDL access (superuser privileges)
	err = h.db.ExecuteWithAdminRole(ctx, func(conn *pgx.Conn) error {
		_, execErr := conn.Exec(ctx, query)
		return execErr
	})
	if err != nil {
		log.Error().Err(err).Str("table", req.Schema+"."+req.Name).Msg("Failed to create table")
		return c.Status(500).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to create table: %v", err),
		})
	}

	log.Info().Str("table", req.Schema+"."+req.Name).Msg("Table created successfully")
	return c.Status(201).JSON(fiber.Map{
		"success": true,
		"schema":  req.Schema,
		"table":   req.Name,
		"message": fmt.Sprintf("Table '%s.%s' created successfully", req.Schema, req.Name),
	})
}

// DeleteTable drops a table from the database
func (h *DDLHandler) DeleteTable(c *fiber.Ctx) error {
	schema := c.Params("schema")
	table := c.Params("table")

	// Validate identifiers
	if err := validateIdentifier(schema, "schema"); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	if err := validateIdentifier(table, "table"); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	ctx := c.Context()

	// Check if table exists
	exists, err := h.tableExists(ctx, schema, table)
	if err != nil {
		log.Error().Err(err).Str("table", schema+"."+table).Msg("Failed to check table existence")
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to check table existence",
		})
	}
	if !exists {
		return c.Status(404).JSON(fiber.Map{
			"error": fmt.Sprintf("Table '%s.%s' does not exist", schema, table),
		})
	}

	// Build DROP TABLE statement
	query := fmt.Sprintf("DROP TABLE %s.%s", quoteIdentifier(schema), quoteIdentifier(table))
	log.Info().Str("table", schema+"."+table).Str("query", query).Msg("Dropping table")

	// Execute DROP TABLE with admin role for full DDL access (superuser privileges)
	err = h.db.ExecuteWithAdminRole(ctx, func(conn *pgx.Conn) error {
		_, execErr := conn.Exec(ctx, query)
		return execErr
	})
	if err != nil {
		log.Error().Err(err).Str("table", schema+"."+table).Msg("Failed to drop table")
		return c.Status(500).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to drop table: %v", err),
		})
	}

	log.Info().Str("table", schema+"."+table).Msg("Table dropped successfully")
	return c.JSON(fiber.Map{
		"success": true,
		"message": fmt.Sprintf("Table '%s.%s' deleted successfully", schema, table),
	})
}

// Helper functions

// validateIdentifier validates a PostgreSQL identifier (schema/table/column name)
func validateIdentifier(name, entityType string) error {
	if name == "" {
		return fmt.Errorf("%s name cannot be empty", entityType)
	}

	if len(name) > 63 {
		return fmt.Errorf("%s name cannot exceed 63 characters", entityType)
	}

	if !identifierPattern.MatchString(name) {
		return fmt.Errorf("%s name must start with a letter or underscore and contain only letters, numbers, and underscores", entityType)
	}

	// Check for reserved keywords
	if reservedKeywords[strings.ToLower(name)] {
		return fmt.Errorf("'%s' is a reserved keyword and cannot be used as a %s name", name, entityType)
	}

	return nil
}


// schemaExists checks if a schema exists
func (h *DDLHandler) schemaExists(ctx context.Context, schema string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM information_schema.schemata WHERE schema_name = $1)`
	err := h.db.Pool().QueryRow(ctx, query, schema).Scan(&exists)
	return exists, err
}

// tableExists checks if a table exists
func (h *DDLHandler) tableExists(ctx context.Context, schema, table string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_schema = $1 AND table_name = $2)`
	err := h.db.Pool().QueryRow(ctx, query, schema, table).Scan(&exists)
	return exists, err
}

// buildCreateTableQuery constructs a CREATE TABLE query from the request
func (h *DDLHandler) buildCreateTableQuery(req CreateTableRequest) (string, error) {
	var columnDefs []string
	var primaryKeys []string

	for i, col := range req.Columns {
		// Validate column name
		if err := validateIdentifier(col.Name, "column"); err != nil {
			return "", fmt.Errorf("column %d: %w", i+1, err)
		}

		// Validate data type
		dataType := strings.ToLower(strings.TrimSpace(col.Type))
		if !validDataTypes[dataType] {
			return "", fmt.Errorf("column '%s': invalid data type '%s'", col.Name, col.Type)
		}

		// Build column definition
		colDef := fmt.Sprintf("%s %s", quoteIdentifier(col.Name), dataType)

		// Add NOT NULL constraint
		if !col.Nullable {
			colDef += " NOT NULL"
		}

		// Add DEFAULT value
		if col.DefaultValue != "" {
			// Sanitize default value - prevent SQL injection
			// For now, we'll use parameterized approach for known functions
			defaultVal := strings.TrimSpace(col.DefaultValue)
			if defaultVal == "gen_random_uuid()" || defaultVal == "now()" || defaultVal == "current_timestamp" {
				colDef += fmt.Sprintf(" DEFAULT %s", defaultVal)
			} else {
				// For literal values, quote them safely
				colDef += fmt.Sprintf(" DEFAULT %s", escapeLiteral(defaultVal))
			}
		}

		columnDefs = append(columnDefs, colDef)

		// Track primary keys
		if col.PrimaryKey {
			primaryKeys = append(primaryKeys, quoteIdentifier(col.Name))
		}
	}

	// Add PRIMARY KEY constraint if any
	if len(primaryKeys) > 0 {
		columnDefs = append(columnDefs, fmt.Sprintf("PRIMARY KEY (%s)", strings.Join(primaryKeys, ", ")))
	}

	// Build final CREATE TABLE statement
	query := fmt.Sprintf(
		"CREATE TABLE %s.%s (\n  %s\n)",
		quoteIdentifier(req.Schema),
		quoteIdentifier(req.Name),
		strings.Join(columnDefs, ",\n  "),
	)

	return query, nil
}

// escapeLiteral escapes a string literal for SQL
// This is a simple implementation - for production, consider using a proper SQL builder
func escapeLiteral(value string) string {
	// Replace single quotes with double single quotes
	escaped := strings.ReplaceAll(value, "'", "''")
	return fmt.Sprintf("'%s'", escaped)
}
