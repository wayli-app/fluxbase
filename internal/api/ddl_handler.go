package api

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/database"
	"github.com/nimbleflux/fluxbase/internal/logutil"
	"github.com/nimbleflux/fluxbase/internal/middleware"
)

// DDLHandler handles Database Definition Language (DDL) operations
// for schema and table management
type DDLHandler struct {
	db                 *database.Connection
	schemaCache        *database.SchemaCache
	graphQLInvalidator func()
}

// NewDDLHandler creates a new DDL handler
func NewDDLHandler(db *database.Connection, schemaCache *database.SchemaCache) *DDLHandler {
	return &DDLHandler{db: db, schemaCache: schemaCache}
}

func (h *DDLHandler) requireDB(c fiber.Ctx) error {
	if h.db == nil {
		return fiber.NewError(fiber.StatusServiceUnavailable, "Database not initialized")
	}
	return nil
}

// SetSchemaCache sets the schema cache for invalidation after DDL operations
func (h *DDLHandler) SetSchemaCache(cache *database.SchemaCache) {
	h.schemaCache = cache
}

func (h *DDLHandler) SetGraphQLInvalidator(invalidator func()) {
	h.graphQLInvalidator = invalidator
}

// Validation patterns
var (
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

// CreateSchema creates a new database schema
func (h *DDLHandler) CreateSchema(c fiber.Ctx) error {
	var req CreateSchemaRequest
	if err := ParseBody(c, &req); err != nil {
		return err
	}

	// Validate schema name
	if err := validateIdentifier(req.Name, "schema"); err != nil {
		return SendBadRequest(c, err.Error(), ErrCodeValidationFailed)
	}

	if err := h.requireDB(c); err != nil {
		return err
	}

	ctx := c.RequestCtx()

	// Check if schema already exists
	exists, err := h.schemaExists(ctx, c, req.Name)
	if err != nil {
		log.Error().Err(err).Str("schema", req.Name).Msg("Failed to check schema existence")
		return SendInternalError(c, "Failed to check schema existence")
	}
	if exists {
		return SendConflict(c, fmt.Sprintf("Schema '%s' already exists", req.Name), ErrCodeAlreadyExists)
	}

	// Create schema (using quoted identifier for safety)
	// Use admin role to ensure full DDL access (superuser privileges)
	query := fmt.Sprintf("CREATE SCHEMA %s", quoteIdentifier(req.Name))
	queryMetadata := logutil.ExtractDDLMetadata(query)
	log.Info().Str("schema", req.Name).Str("operation", queryMetadata).Msg("Creating schema")

	err = h.executeWithAdminRole(ctx, c, func(tx pgx.Tx) error {
		_, execErr := tx.Exec(ctx, query)
		return execErr
	})
	if err != nil {
		log.Error().Err(err).Str("schema", req.Name).Msg("Failed to create schema")
		return SendInternalError(c, "Failed to create schema")
	}

	// Set up default privileges for tables created in this schema by the admin user
	// This ensures that future tables created via DDL API will automatically get grants to service_role
	if err := h.setupSchemaDefaultPrivileges(ctx, c, req.Name); err != nil {
		log.Error().Err(err).Str("schema", req.Name).Msg("Failed to set up default privileges")
		// Don't fail the request - schema was created successfully, just log the error
	}

	h.invalidateCache(ctx)
	log.Info().Str("schema", req.Name).Msg("Schema created successfully")
	return c.Status(201).JSON(fiber.Map{
		"success": true,
		"schema":  req.Name,
		"message": fmt.Sprintf("Schema '%s' created successfully", req.Name),
	})
}

// validateIdentifier validates a PostgreSQL identifier (schema/table/column name)
func validateIdentifier(name, entityType string) error {
	if name == "" {
		return fmt.Errorf("%s name cannot be empty", entityType)
	}

	if len(name) > 63 {
		return fmt.Errorf("%s name cannot exceed 63 characters", entityType)
	}

	if !validIdentifierRegex.MatchString(name) {
		return fmt.Errorf("%s name must start with a letter or underscore and contain only letters, numbers, and underscores", entityType)
	}

	// Check for reserved keywords
	if reservedKeywords[strings.ToLower(name)] {
		return fmt.Errorf("'%s' is a reserved keyword and cannot be used as a %s name", name, entityType)
	}

	return nil
}

// schemaExists checks if a schema exists, using tenant pool when available.
func (h *DDLHandler) schemaExists(ctx context.Context, c fiber.Ctx, schema string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM information_schema.schemata WHERE schema_name = $1)`
	err := h.queryPool(c).QueryRow(ctx, query, schema).Scan(&exists)
	return exists, err
}

// tableExists checks if a table exists, using tenant pool when available.
func (h *DDLHandler) tableExists(ctx context.Context, c fiber.Ctx, schema, table string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_schema = $1 AND table_name = $2)`
	err := h.queryPool(c).QueryRow(ctx, query, schema, table).Scan(&exists)
	return exists, err
}

// columnExists checks if a column exists in a table, using tenant pool when available.
func (h *DDLHandler) columnExists(ctx context.Context, c fiber.Ctx, schema, table, column string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM information_schema.columns WHERE table_schema = $1 AND table_name = $2 AND column_name = $3)`
	err := h.queryPool(c).QueryRow(ctx, query, schema, table, column).Scan(&exists)
	return exists, err
}

// queryPool returns the tenant pool if available, otherwise the main pool.
func (h *DDLHandler) queryPool(c fiber.Ctx) *pgxpool.Pool {
	if pool := middleware.GetTenantPool(c); pool != nil {
		return pool
	}
	return h.db.Pool()
}

// executeWithAdminRole executes a function with admin role, routing to the
// tenant database when a tenant context is active.
func (h *DDLHandler) executeWithAdminRole(ctx context.Context, c fiber.Ctx, fn func(tx pgx.Tx) error) error {
	if dbName, _ := c.Locals("tenant_db_name").(string); dbName != "" {
		return h.db.ExecuteWithAdminRoleForDB(ctx, dbName, fn)
	}
	return h.db.ExecuteWithAdminRole(ctx, fn)
}

// invalidateCache invalidates the schema cache after DDL operations.
func (h *DDLHandler) invalidateCache(ctx context.Context) {
	if h.schemaCache != nil {
		h.schemaCache.InvalidateAll(ctx)
	}
	if h.graphQLInvalidator != nil {
		h.graphQLInvalidator()
	}
}

// safeDefaultFunctions is a set of PostgreSQL functions that are safe to use as DEFAULT values
// These functions are allowed to pass through without escaping
var safeDefaultFunctions = map[string]bool{
	// UUID functions
	"gen_random_uuid()":    true,
	"uuid_generate_v4()":   true,
	"uuid_generate_v1()":   true,
	"uuid_generate_v1mc()": true,
	"uuid_generate_v3()":   true,
	"uuid_generate_v5()":   true,
	// Date/Time functions
	"now()":                   true,
	"current_timestamp":       true,
	"CURRENT_TIMESTAMP":       true,
	"current_date":            true,
	"CURRENT_DATE":            true,
	"current_time":            true,
	"CURRENT_TIME":            true,
	"localtime":               true,
	"LOCALTIME":               true,
	"localtimestamp":          true,
	"LOCALTIMESTAMP":          true,
	"transaction_timestamp()": true,
	"statement_timestamp()":   true,
	"clock_timestamp()":       true,
	// Boolean
	"true":  true,
	"TRUE":  true,
	"false": true,
	"FALSE": true,
	// Null
	"NULL": true,
	"null": true,
}

// sanitizeDefaultValue sanitizes a DEFAULT value for SQL
// It returns safe SQL functions directly or escapes literal values
func sanitizeDefaultValue(value string) string {
	defaultVal := strings.TrimSpace(value)

	// Check if it's a safe function
	if safeDefaultFunctions[defaultVal] {
		return defaultVal
	}

	// Check for numeric literals (integers and floats)
	if _, err := strconv.ParseInt(defaultVal, 10, 64); err == nil {
		return defaultVal
	}
	if _, err := strconv.ParseFloat(defaultVal, 64); err == nil {
		return defaultVal
	}

	// Check for type casts with safe functions (e.g., "now()::date", "'2024-01-01'::date")
	if strings.Contains(defaultVal, "::") {
		parts := strings.SplitN(defaultVal, "::", 2)
		if len(parts) == 2 {
			baseValue := strings.TrimSpace(parts[0])
			castType := strings.TrimSpace(parts[1])
			// Validate the cast type is alphanumeric (prevent injection)
			if isValidCastType(castType) {
				// If base is a safe function, allow the cast
				if safeDefaultFunctions[baseValue] {
					return defaultVal
				}
				// If base is already a quoted string, escape its content to prevent injection
				if strings.HasPrefix(baseValue, "'") && strings.HasSuffix(baseValue, "'") && len(baseValue) >= 2 {
					inner := baseValue[1 : len(baseValue)-1]
					escaped := strings.ReplaceAll(inner, "'", "''")
					escaped = strings.ReplaceAll(escaped, "\x00", "")
					return fmt.Sprintf("'%s'::%s", escaped, castType)
				}
			}
		}
	}

	// For all other values, escape as a string literal
	return escapeLiteral(defaultVal)
}

// isValidCastType checks if a cast type is valid (alphanumeric with allowed chars)
func isValidCastType(t string) bool {
	if t == "" {
		return false
	}
	for _, r := range t {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' && r != '[' && r != ']' && r != ' ' && r != ',' {
			return false
		}
	}
	return true
}

// escapeLiteral escapes a string literal for SQL using PostgreSQL-compatible rules.
// Handles single quotes, backslashes, and null bytes.
func escapeLiteral(value string) string {
	// Remove null bytes (never valid in SQL literals)
	cleaned := strings.ReplaceAll(value, "\x00", "")
	// Escape backslashes
	cleaned = strings.ReplaceAll(cleaned, `\`, `\\`)
	// Replace single quotes with double single quotes (PostgreSQL standard)
	cleaned = strings.ReplaceAll(cleaned, "'", "''")
	return fmt.Sprintf("'%s'", cleaned)
}

// setupSchemaDefaultPrivileges sets up default privileges for a schema
// so that tables created by the admin user automatically get grants to service_role
func (h *DDLHandler) setupSchemaDefaultPrivileges(ctx context.Context, c fiber.Ctx, schema string) error {
	// Set up default privileges for tables created in this schema
	// This ensures that future tables created via DDL API will automatically get grants to service_role
	queries := []string{
		// Grant ALL on future tables to service_role
		fmt.Sprintf("ALTER DEFAULT PRIVILEGES FOR ROLE CURRENT_USER IN SCHEMA %s GRANT ALL ON TABLES TO service_role", quoteIdentifier(schema)),
		// Grant USAGE on future functions to service_role
		fmt.Sprintf("ALTER DEFAULT PRIVILEGES FOR ROLE CURRENT_USER IN SCHEMA %s GRANT ALL ON FUNCTIONS TO service_role", quoteIdentifier(schema)),
		// Grant USAGE on future sequences to service_role
		fmt.Sprintf("ALTER DEFAULT PRIVILEGES FOR ROLE CURRENT_USER IN SCHEMA %s GRANT USAGE, SELECT ON SEQUENCES TO service_role", quoteIdentifier(schema)),
	}

	for _, query := range queries {
		err := h.executeWithAdminRole(ctx, c, func(tx pgx.Tx) error {
			_, err := tx.Exec(ctx, query)
			return err
		})
		if err != nil {
			return fmt.Errorf("failed to set up default privileges: %w", err)
		}
	}

	// Also grant USAGE on the schema itself to service_role, anon, and authenticated
	grantSchemaQuery := fmt.Sprintf("GRANT USAGE ON SCHEMA %s TO service_role, anon, authenticated", quoteIdentifier(schema))
	err := h.executeWithAdminRole(ctx, c, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, grantSchemaQuery)
		return err
	})
	if err != nil {
		return fmt.Errorf("failed to grant schema usage: %w", err)
	}

	log.Debug().Str("schema", schema).Msg("Set up default privileges for schema")

	return nil
}

// ListSchemas returns all user schemas (excluding system schemas)
func (h *DDLHandler) ListSchemas(c fiber.Ctx) error {
	if err := h.requireDB(c); err != nil {
		return err
	}

	ctx := c.RequestCtx()
	inspector := h.db.Inspector()

	var schemas []string
	var err error
	if tenantPool := middleware.GetTenantPool(c); tenantPool != nil {
		schemas, err = inspector.GetSchemasFromQ(ctx, database.PoolQuerier(tenantPool))
	} else {
		schemas, err = inspector.GetSchemas(ctx)
	}
	if err != nil {
		log.Error().Err(err).Msg("Failed to list schemas")
		return SendOperationFailed(c, "list schemas")
	}

	// Filter out system schemas and build response
	type schemaInfo struct {
		Name  string `json:"name"`
		Owner string `json:"owner"`
	}
	var result []schemaInfo
	for _, schema := range schemas {
		// Skip system schemas
		if schema == "information_schema" || schema == "pg_catalog" || schema == "pg_toast" {
			continue
		}
		result = append(result, schemaInfo{Name: schema, Owner: "postgres"})
	}

	// Filter schemas for tenant admins: only show schemas with tenant-visible tables
	if userRole, ok := GetUserRole(c); ok {
		isInstanceAdmin := userRole == "admin" || userRole == "instance_admin" || userRole == "service_role" || userRole == "tenant_service"
		if !isInstanceAdmin {
			tenantVisible := map[string]bool{
				"public": true, "auth": true, "storage": true, "functions": true,
				"jobs": true, "ai": true, "rpc": true, "mcp": true,
				"realtime": true, "branching": true, "logging": true, "platform": true,
			}
			var filtered []schemaInfo
			for _, s := range result {
				if tenantVisible[s.Name] {
					filtered = append(filtered, s)
				}
			}
			result = filtered
		}
	}

	return c.JSON(fiber.Map{"schemas": result})
}
