package api

import (
	"context"
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/database"
	apperrors "github.com/nimbleflux/fluxbase/internal/errors"
	"github.com/nimbleflux/fluxbase/internal/logutil"
	"github.com/nimbleflux/fluxbase/internal/middleware"
)

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

// RenameTableRequest represents a request to rename a table
type RenameTableRequest struct {
	NewName string `json:"newName"`
}

// CreateTable creates a new table with specified columns
func (h *DDLHandler) CreateTable(c fiber.Ctx) error {
	var req CreateTableRequest
	if err := ParseBody(c, &req); err != nil {
		return err
	}

	if err := validateIdentifier(req.Schema, "schema"); err != nil {
		return SendBadRequest(c, err.Error(), ErrCodeValidationFailed)
	}

	if err := validateIdentifier(req.Name, "table"); err != nil {
		return SendBadRequest(c, err.Error(), ErrCodeValidationFailed)
	}

	if len(req.Columns) == 0 {
		return SendBadRequest(c, "At least one column is required", ErrCodeValidationFailed)
	}

	if err := h.requireDB(c); err != nil {
		return err
	}

	ctx := c.RequestCtx()

	// Check if schema exists
	exists, err := h.schemaExists(ctx, c, req.Schema)
	if err != nil {
		log.Error().Err(err).Str("schema", req.Schema).Msg("Failed to check schema existence")
		return SendInternalError(c, "Failed to check schema existence")
	}
	if !exists {
		return SendNotFound(c, fmt.Sprintf("Schema '%s' does not exist", req.Schema))
	}

	// Check if table already exists
	tableExists, err := h.tableExists(ctx, c, req.Schema, req.Name)
	if err != nil {
		log.Error().Err(err).Str("table", req.Schema+"."+req.Name).Msg("Failed to check table existence")
		return SendInternalError(c, "Failed to check table existence")
	}
	if tableExists {
		return SendConflict(c, fmt.Sprintf("Table '%s.%s' already exists", req.Schema, req.Name), ErrCodeAlreadyExists)
	}

	// Build CREATE TABLE statement
	query, err := h.buildCreateTableQuery(req)
	if err != nil {
		return SendBadRequest(c, err.Error(), ErrCodeValidationFailed)
	}

	log.Info().
		Str("table", req.Schema+"."+req.Name).
		Str("operation", logutil.ExtractDDLMetadata(query)).
		Int("columns", len(req.Columns)).
		Msg("Creating table")

	// Execute CREATE TABLE with admin role for full DDL access (superuser privileges)
	err = h.executeWithAdminRole(ctx, c, func(tx pgx.Tx) error {
		_, execErr := tx.Exec(ctx, query)
		return execErr
	})
	if err != nil {
		log.Error().Err(err).Str("table", req.Schema+"."+req.Name).Msg("Failed to create table")
		return SendInternalError(c, "Failed to create table")
	}

	// Grant permissions to service_role for instance_admin access
	// This is necessary because tables created via ExecuteWithAdminRole don't
	// inherit default privileges from migration 027 (which only applies to CURRENT_USER)
	if err := h.grantTablePermissions(ctx, c, req.Schema, req.Name); err != nil {
		log.Error().Err(err).Str("table", req.Schema+"."+req.Name).Msg("Failed to grant permissions to service_role")
	}

	h.autoCreateTenantServicePolicy(ctx, c, req.Schema, req.Name)

	h.invalidateCache(ctx)
	log.Info().Str("table", req.Schema+"."+req.Name).Msg("Table created successfully")
	return c.Status(201).JSON(fiber.Map{
		"success": true,
		"schema":  req.Schema,
		"table":   req.Name,
		"message": fmt.Sprintf("Table '%s.%s' created successfully", req.Schema, req.Name),
	})
}

// DeleteTable drops a table from the database
func (h *DDLHandler) DeleteTable(c fiber.Ctx) error {
	schema := c.Params("schema")
	table := c.Params("table")

	// Validate identifiers
	if err := validateIdentifier(schema, "schema"); err != nil {
		return SendBadRequest(c, err.Error(), ErrCodeValidationFailed)
	}
	if err := validateIdentifier(table, "table"); err != nil {
		return SendBadRequest(c, err.Error(), ErrCodeValidationFailed)
	}

	if err := h.requireDB(c); err != nil {
		return err
	}

	ctx := c.RequestCtx()

	// Check if table exists
	exists, err := h.tableExists(ctx, c, schema, table)
	if err != nil {
		log.Error().Err(err).Str("table", schema+"."+table).Msg("Failed to check table existence")
		return SendInternalError(c, "Failed to check table existence")
	}
	if !exists {
		return SendNotFound(c, fmt.Sprintf("Table '%s.%s' does not exist", schema, table))
	}

	// Build DROP TABLE statement
	query := fmt.Sprintf("DROP TABLE %s.%s", quoteIdentifier(schema), quoteIdentifier(table))
	log.Info().Str("table", schema+"."+table).Str("operation", logutil.ExtractDDLMetadata(query)).Msg("Dropping table")

	// Execute DROP TABLE with admin role for full DDL access (superuser privileges)
	err = h.executeWithAdminRole(ctx, c, func(tx pgx.Tx) error {
		_, execErr := tx.Exec(ctx, query)
		return execErr
	})
	if err != nil {
		log.Error().Err(err).Str("table", schema+"."+table).Msg("Failed to drop table")
		return SendInternalError(c, "Failed to drop table")
	}

	h.invalidateCache(ctx)
	log.Info().Str("table", schema+"."+table).Msg("Table dropped successfully")
	return apperrors.SendSuccess(c, fmt.Sprintf("Table '%s.%s' deleted successfully", schema, table))
}

// RenameTable renames a table
func (h *DDLHandler) RenameTable(c fiber.Ctx) error {
	schema := c.Params("schema")
	table := c.Params("table")

	// Validate identifiers
	if err := validateIdentifier(schema, "schema"); err != nil {
		return SendBadRequest(c, err.Error(), ErrCodeValidationFailed)
	}
	if err := validateIdentifier(table, "table"); err != nil {
		return SendBadRequest(c, err.Error(), ErrCodeValidationFailed)
	}

	var req RenameTableRequest
	if err := ParseBody(c, &req); err != nil {
		return err
	}

	if err := h.requireDB(c); err != nil {
		return err
	}

	// Validate new table name
	if err := validateIdentifier(req.NewName, "table"); err != nil {
		return SendBadRequest(c, err.Error(), ErrCodeValidationFailed)
	}

	ctx := c.RequestCtx()

	// Check if source table exists
	exists, err := h.tableExists(ctx, c, schema, table)
	if err != nil {
		log.Error().Err(err).Str("table", schema+"."+table).Msg("Failed to check table existence")
		return SendOperationFailed(c, "check table existence")
	}
	if !exists {
		return SendNotFound(c, fmt.Sprintf("Table '%s.%s' does not exist", schema, table))
	}

	// Check if target table name already exists
	targetExists, err := h.tableExists(ctx, c, schema, req.NewName)
	if err != nil {
		log.Error().Err(err).Msg("Failed to check target table existence")
		return SendOperationFailed(c, "check target table existence")
	}
	if targetExists {
		return SendConflict(c, fmt.Sprintf("Table '%s.%s' already exists", schema, req.NewName), ErrCodeAlreadyExists)
	}

	query := fmt.Sprintf("ALTER TABLE %s.%s RENAME TO %s",
		quoteIdentifier(schema), quoteIdentifier(table), quoteIdentifier(req.NewName))

	log.Info().Str("table", schema+"."+table).Str("newName", req.NewName).Str("operation", logutil.ExtractDDLMetadata(query)).Msg("Renaming table")

	err = h.executeWithAdminRole(ctx, c, func(tx pgx.Tx) error {
		_, execErr := tx.Exec(ctx, query)
		return execErr
	})
	if err != nil {
		log.Error().Err(err).Str("table", schema+"."+table).Str("newName", req.NewName).Msg("Failed to rename table")
		return SendInternalError(c, "Failed to rename table")
	}

	h.invalidateCache(ctx)
	log.Info().Str("table", schema+"."+table).Str("newName", req.NewName).Msg("Table renamed successfully")
	return apperrors.SendSuccess(c, fmt.Sprintf("Table '%s.%s' renamed to '%s.%s'", schema, table, schema, req.NewName))
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
			colDef += fmt.Sprintf(" DEFAULT %s", sanitizeDefaultValue(col.DefaultValue))
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

// grantTablePermissions grants necessary permissions on a table to service_role
// This ensures that instance_admin (which maps to service_role) can access the table
func (h *DDLHandler) grantTablePermissions(ctx context.Context, c fiber.Ctx, schema, table string) error {
	// Grant SELECT, INSERT, UPDATE, DELETE on the table to service_role
	grantTableQuery := fmt.Sprintf(
		"GRANT SELECT, INSERT, UPDATE, DELETE ON %s.%s TO service_role",
		quoteIdentifier(schema),
		quoteIdentifier(table),
	)

	err := h.executeWithAdminRole(ctx, c, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, grantTableQuery)
		return err
	})
	if err != nil {
		return fmt.Errorf("failed to grant table permissions: %w", err)
	}

	// Grant USAGE on all sequences for this table (for auto-increment/identity columns)
	// This query finds all sequences belonging to the table and grants USAGE
	grantSequencesQuery := `
		SELECT sequence_name
		FROM information_schema.sequences
		WHERE sequence_schema = $1
		  AND sequence_name LIKE $2
	`

	rows, err := h.queryPool(c).Query(ctx, grantSequencesQuery, schema, table+"_%")
	if err != nil {
		// Don't fail if we can't query sequences - table permissions are already granted
		log.Debug().Err(err).Str("table", schema+"."+table).Msg("Failed to query sequences for table")
		return nil
	}
	defer rows.Close()

	var sequenceNames []string
	for rows.Next() {
		var seqName string
		if err := rows.Scan(&seqName); err != nil {
			continue
		}
		sequenceNames = append(sequenceNames, seqName)
	}

	// Grant USAGE on each sequence
	for _, seqName := range sequenceNames {
		grantSeqQuery := fmt.Sprintf(
			"GRANT USAGE, SELECT ON SEQUENCE %s.%s TO service_role",
			quoteIdentifier(schema),
			quoteIdentifier(seqName),
		)
		err := h.executeWithAdminRole(ctx, c, func(tx pgx.Tx) error {
			_, err := tx.Exec(ctx, grantSeqQuery)
			return err
		})
		if err != nil {
			log.Debug().Err(err).Str("sequence", schema+"."+seqName).Msg("Failed to grant sequence permissions")
		}
	}

	log.Debug().
		Str("table", schema+"."+table).
		Int("sequences_granted", len(sequenceNames)).
		Msg("Granted permissions to service_role for table")

	return nil
}

// autoCreateTenantServicePolicy creates a tenant_service RLS policy on a table
// that has a tenant_id column. This ensures functions/jobs using tenant_service
// can access tenant-scoped data when RLS is enabled on the table.
func (h *DDLHandler) autoCreateTenantServicePolicy(ctx context.Context, c fiber.Ctx, schema, table string) {
	if schema != "public" {
		return
	}

	var hasTenantCol bool
	err := h.queryPool(c).QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.columns
			WHERE table_schema = $1 AND table_name = $2 AND column_name = 'tenant_id'
		)
	`, schema, table).Scan(&hasTenantCol)
	if err != nil || !hasTenantCol {
		return
	}

	policyName := fmt.Sprintf("%s_tenant_service_auto", table)
	policySQL := fmt.Sprintf(
		`CREATE POLICY IF NOT EXISTS %s ON %s.%s TO tenant_service
		 USING (auth.has_tenant_access(tenant_id))
		 WITH CHECK (auth.has_tenant_access(tenant_id))`,
		quoteIdentifier(policyName),
		quoteIdentifier(schema),
		quoteIdentifier(table),
	)

	_ = h.executeWithAdminRole(ctx, c, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, policySQL)
		return err
	})
}

// ListTables returns all tables, optionally filtered by schema
func (h *DDLHandler) ListTables(c fiber.Ctx) error {
	if err := h.requireDB(c); err != nil {
		return err
	}

	ctx := c.RequestCtx()
	schemaParam := c.Query("schema")
	inspector := h.db.Inspector()
	tenantPool := middleware.GetTenantPool(c)

	var schemasToQuery []string

	if schemaParam != "" {
		// If schema parameter provided, query only that schema
		schemasToQuery = []string{schemaParam}
	} else {
		// Otherwise, get all schemas
		var schemas []string
		var err error
		if tenantPool != nil {
			schemas, err = inspector.GetSchemasFromQ(ctx, database.PoolQuerier(tenantPool))
		} else {
			schemas, err = inspector.GetSchemas(ctx)
		}
		if err != nil {
			log.Error().Err(err).Msg("Failed to list schemas")
			return SendOperationFailed(c, "list schemas")
		}

		// Filter out system schemas
		for _, schema := range schemas {
			if schema == "information_schema" || schema == "pg_catalog" || schema == "pg_toast" {
				continue
			}
			schemasToQuery = append(schemasToQuery, schema)
		}
	}

	// Collect tables from requested schema(s)
	type tableInfo struct {
		Schema string `json:"schema"`
		Name   string `json:"name"`
	}
	var tables []tableInfo

	for _, schema := range schemasToQuery {
		var dbTables []database.TableInfo
		var err error
		if tenantPool != nil {
			dbTables, err = inspector.GetAllTablesFromQ(ctx, database.PoolQuerier(tenantPool), schema)
		} else {
			dbTables, err = inspector.GetAllTables(ctx, schema)
		}
		if err != nil {
			log.Warn().Err(err).Str("schema", schema).Msg("Failed to get tables from schema")
			continue
		}
		for _, t := range dbTables {
			tables = append(tables, tableInfo{Schema: t.Schema, Name: t.Name})
		}
	}

	return c.JSON(fiber.Map{"tables": tables})
}
