package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"
	"github.com/wayli-app/fluxbase/internal/database"
	"github.com/wayli-app/fluxbase/internal/middleware"
)

// RESTHandler handles dynamic REST API endpoints
type RESTHandler struct {
	db     *database.Connection
	parser *QueryParser
}

// NewRESTHandler creates a new REST handler
func NewRESTHandler(db *database.Connection, parser *QueryParser) *RESTHandler {
	return &RESTHandler{
		db:     db,
		parser: parser,
	}
}

// handleDatabaseError returns an appropriate HTTP error response based on the database error.
// This centralizes error handling logic for all REST operations.
func handleDatabaseError(c *fiber.Ctx, err error, operation string) error {
	errMsg := err.Error()

	// Duplicate key violation (unique constraint)
	if strings.Contains(errMsg, "duplicate key") || strings.Contains(errMsg, "unique constraint") {
		return c.Status(409).JSON(fiber.Map{
			"error": "Record with this value already exists",
		})
	}

	// Foreign key constraint violation
	if strings.Contains(errMsg, "foreign key constraint") {
		return c.Status(409).JSON(fiber.Map{
			"error": "Cannot complete operation due to foreign key constraint",
		})
	}

	// NOT NULL constraint violation (missing required field)
	if strings.Contains(errMsg, "null value in column") || strings.Contains(errMsg, "not-null constraint") {
		return c.Status(400).JSON(fiber.Map{
			"error": "Missing required field",
		})
	}

	// Invalid input syntax (type mismatch, invalid data)
	if strings.Contains(errMsg, "invalid input syntax") {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid data type provided",
		})
	}

	// Check constraint violation
	if strings.Contains(errMsg, "check constraint") {
		return c.Status(400).JSON(fiber.Map{
			"error": "Data violates table constraints",
		})
	}

	// Generic server error for other cases
	return c.Status(500).JSON(fiber.Map{
		"error": fmt.Sprintf("Failed to %s", operation),
	})
}

// isUserAuthenticated checks if the user is authenticated based on RLS context
func isUserAuthenticated(c *fiber.Ctx) bool {
	role := c.Locals("rls_role")
	if role == nil {
		return false
	}
	roleStr, ok := role.(string)
	if !ok {
		return false
	}
	// User is authenticated if they have any role other than "anon"
	return roleStr != "anon" && roleStr != ""
}

// handleRLSViolation returns appropriate error response for RLS policy violations.
// For mutations (INSERT/UPDATE/DELETE), when a query succeeds but returns 0 rows,
// it's likely an RLS policy blocking the operation rather than the record not existing.
func (h *RESTHandler) handleRLSViolation(c *fiber.Ctx, operation string, tableName string) error {
	ctx := c.Context()

	// Check if user is authenticated
	authenticated := isUserAuthenticated(c)

	// Log the violation to the audit table
	// Note: This is synchronous to avoid Fiber context issues with goroutines
	// The DB insert is fast enough to not significantly impact response time
	middleware.LogRLSViolation(ctx, h.db, c, operation, tableName)

	if !authenticated {
		// Anonymous users get 401 - need authentication
		log.Warn().
			Str("operation", operation).
			Str("table", tableName).
			Str("role", "anon").
			Msg("RLS violation: Anonymous user attempted operation")

		return c.Status(401).JSON(fiber.Map{
			"error": "Authentication required",
			"code":  "AUTHENTICATION_REQUIRED",
		})
	}

	// Authenticated users get 403 - insufficient permissions
	userID := c.Locals("rls_user_id")
	role := c.Locals("rls_role")

	log.Warn().
		Interface("user_id", userID).
		Interface("role", role).
		Str("operation", operation).
		Str("table", tableName).
		Msg("RLS violation: Insufficient permissions")

	return c.Status(403).JSON(fiber.Map{
		"error":   "Insufficient permissions",
		"code":    "RLS_POLICY_VIOLATION",
		"message": "Row-level security policy blocks this operation",
		"hint":    "Verify your authentication and table access policies",
	})
}

// RegisterTableRoutes registers REST routes for a table
func (h *RESTHandler) RegisterTableRoutes(router fiber.Router, table database.TableInfo) {
	// Build the REST path for this table
	basePath := h.BuildTablePath(table)

	log.Info().
		Str("table", fmt.Sprintf("%s.%s", table.Schema, table.Name)).
		Str("path", basePath).
		Bool("rls_enabled", table.RLSEnabled).
		Msg("Registering REST endpoints")

	// Register routes
	router.Get(basePath, h.makeGetHandler(table))
	router.Get(basePath+"/:id", h.makeGetByIdHandler(table))
	router.Post(basePath, h.makePostHandler(table))
	router.Put(basePath+"/:id", h.makePutHandler(table))
	router.Patch(basePath+"/:id", h.makePatchHandler(table))   // Single record update
	router.Patch(basePath, h.makeBatchPatchHandler(table))     // Batch update with filters
	router.Delete(basePath+"/:id", h.makeDeleteHandler(table)) // Single record delete
	router.Delete(basePath, h.makeBatchDeleteHandler(table))   // Batch delete with filters
}

// RegisterViewRoutes registers read-only REST routes for a database view
func (h *RESTHandler) RegisterViewRoutes(router fiber.Router, view database.TableInfo) {
	// Build the REST path for this view
	basePath := h.BuildTablePath(view)

	log.Info().
		Str("view", fmt.Sprintf("%s.%s", view.Schema, view.Name)).
		Str("path", basePath).
		Msg("Registering read-only view endpoints")

	// Register only GET routes for views (read-only)
	router.Get(basePath, h.makeGetHandler(view))
	router.Get(basePath+"/:id", h.makeGetByIdHandler(view))
}

// BuildTablePath builds the REST API path for a table (relative to router group)
// Used for registering routes on the /api/v1/tables router group
func (h *RESTHandler) BuildTablePath(table database.TableInfo) string {
	// Use table name as-is without pluralization
	tableName := table.Name

	// Paths are relative to the router group
	if table.Schema != "public" {
		return "/" + table.Schema + "/" + tableName
	}
	return "/" + tableName
}

// BuildFullTablePath builds the full REST API path for a table (including /api/v1/tables prefix)
// Used for client consumption in API responses
func (h *RESTHandler) BuildFullTablePath(table database.TableInfo) string {
	relativePath := h.BuildTablePath(table)
	return "/api/v1/tables" + relativePath
}

// makeGetHandler creates a GET handler for listing records
func (h *RESTHandler) makeGetHandler(table database.TableInfo) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx := c.Context()

		// Convert Fiber queries to url.Values
		queries := c.Queries()
		urlValues := make(url.Values)
		for k, v := range queries {
			urlValues.Add(k, v)
		}

		// Parse query parameters
		params, err := h.parser.Parse(urlValues)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{
				"error": fmt.Sprintf("Invalid query parameters: %v", err),
			})
		}

		// Build SELECT query
		query, args := h.buildSelectQuery(table, params)

		// Execute query with RLS context
		var results []map[string]interface{}
		err = middleware.WrapWithRLS(ctx, h.db, c, func(tx pgx.Tx) error {
			log.Debug().Str("query", query).Interface("args", args).Msg("Executing SELECT query")
			rows, err := tx.Query(ctx, query, args...)
			if err != nil {
				log.Error().Err(err).Str("query", query).Msg("Failed to execute query")
				return err
			}
			defer rows.Close()

			// Convert rows to JSON
			results, err = pgxRowsToJSON(rows)
			log.Debug().Interface("results", results).Int("count", len(results)).Msg("Query results")
			return err
		})
		if err != nil {
			return c.Status(500).JSON(fiber.Map{
				"error": "Failed to fetch records",
			})
		}

		// Handle count if requested
		if params.Count != CountNone && params.Count != "" {
			count, err := h.getCount(ctx, c, table, params)
			if err != nil {
				log.Warn().Err(err).Msg("Failed to get count")
			} else {
				c.Set("Content-Range", fmt.Sprintf("0-%d/%d", len(results)-1, count))
			}
		}

		return c.JSON(results)
	}
}

// makeGetByIdHandler creates a GET handler for fetching a single record
func (h *RESTHandler) makeGetByIdHandler(table database.TableInfo) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx := c.Context()
		id := c.Params("id")

		// Determine primary key column
		pkColumn := "id"
		if len(table.PrimaryKey) > 0 {
			pkColumn = table.PrimaryKey[0]
		}

		// Build query
		query := fmt.Sprintf(
			"SELECT * FROM %s.%s WHERE %s = $1",
			table.Schema, table.Name, pkColumn,
		)

		// Execute query with RLS context
		var results []map[string]interface{}
		err := middleware.WrapWithRLS(ctx, h.db, c, func(tx pgx.Tx) error {
			rows, err := tx.Query(ctx, query, id)
			if err != nil {
				return err
			}
			defer rows.Close()

			// Convert to JSON
			results, err = pgxRowsToJSON(rows)
			return err
		})
		if err != nil {
			return c.Status(500).JSON(fiber.Map{
				"error": "Failed to fetch record",
			})
		}

		if len(results) == 0 {
			return c.Status(404).JSON(fiber.Map{
				"error": "Record not found",
			})
		}

		return c.JSON(results[0])
	}
}

// makePostHandler creates a POST handler for creating records (single or batch)
func (h *RESTHandler) makePostHandler(table database.TableInfo) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx := c.Context()

		// Check for upsert preferences
		preferHeader := c.Get("Prefer", "")
		isUpsert := strings.Contains(preferHeader, "resolution=merge-duplicates") || strings.Contains(preferHeader, "resolution=ignore-duplicates")
		ignoreDuplicates := strings.Contains(preferHeader, "resolution=ignore-duplicates")
		defaultToNull := strings.Contains(preferHeader, "missing=default")

		// Get custom conflict target from query parameter
		onConflict := c.Query("on_conflict", "")

		// Try to parse as array first (batch insert)
		var dataArray []map[string]interface{}
		if err := c.BodyParser(&dataArray); err == nil && len(dataArray) > 0 {
			// Batch insert
			return h.batchInsert(ctx, c, table, dataArray, isUpsert, ignoreDuplicates, defaultToNull, onConflict)
		}

		// Otherwise parse as single object
		var data map[string]interface{}
		if err := c.BodyParser(&data); err != nil {
			return c.Status(400).JSON(fiber.Map{
				"error": "Invalid request body",
			})
		}

		// Build INSERT query
		columns := make([]string, 0, len(data))
		values := make([]interface{}, 0, len(data))
		placeholders := make([]string, 0, len(data))

		i := 1
		for col, val := range data {
			// Validate column exists
			if !h.columnExists(table, col) {
				return c.Status(400).JSON(fiber.Map{
					"error": fmt.Sprintf("Unknown column: %s", col),
				})
			}

			columns = append(columns, col)
			values = append(values, val)
			placeholders = append(placeholders, fmt.Sprintf("$%d", i))
			i++
		}

		query := fmt.Sprintf(
			"INSERT INTO %s.%s (%s) VALUES (%s)",
			table.Schema, table.Name,
			strings.Join(columns, ", "),
			strings.Join(placeholders, ", "),
		)

		// Add ON CONFLICT clause for upsert
		if isUpsert {
			// Use custom conflict target if provided, otherwise auto-detect
			conflictTarget := onConflict
			if conflictTarget == "" {
				conflictTarget = h.getConflictTarget(table)
			}

			if conflictTarget == "" {
				return c.Status(400).JSON(fiber.Map{
					"error": "Cannot perform upsert: table has no primary key or unique constraint",
				})
			}

			// Handle ignore duplicates (DO NOTHING)
			if ignoreDuplicates {
				query += fmt.Sprintf(
					" ON CONFLICT (%s) DO NOTHING",
					conflictTarget,
				)
			} else {
				// Build UPDATE SET clause (all columns except conflict target)
				updateClauses := make([]string, 0)
				for _, col := range columns {
					// Skip columns that are part of the conflict target
					if !h.isInConflictTarget(col, conflictTarget) {
						// If defaultToNull is true, set missing columns to NULL explicitly
						// (This doesn't apply here since we're setting all provided columns)
						updateClauses = append(updateClauses, fmt.Sprintf("%s = EXCLUDED.%s", col, col))
					}
				}

				if len(updateClauses) > 0 {
					query += fmt.Sprintf(
						" ON CONFLICT (%s) DO UPDATE SET %s",
						conflictTarget,
						strings.Join(updateClauses, ", "),
					)
				} else {
					query += fmt.Sprintf(
						" ON CONFLICT (%s) DO NOTHING",
						conflictTarget,
					)
				}
			}
		}

		query += " RETURNING *"

		// Execute query with RLS context
		var results []map[string]interface{}
		if err := middleware.WrapWithRLS(ctx, h.db, c, func(tx pgx.Tx) error {
			rows, err := tx.Query(ctx, query, values...)
			if err != nil {
				log.Error().Err(err).Str("query", query).Msg("Failed to insert record")
				return err
			}
			defer rows.Close()

			// Convert to JSON
			results, err = pgxRowsToJSON(rows)
			return err
		}); err != nil {
			return handleDatabaseError(c, err, "create record")
		}

		if len(results) == 0 {
			// INSERT with RETURNING 0 rows typically indicates RLS policy blocked the operation
			return h.handleRLSViolation(c, "INSERT", fmt.Sprintf("%s.%s", table.Schema, table.Name))
		}

		return c.Status(201).JSON(results[0])
	}
}

// batchInsert handles batch insert operations
func (h *RESTHandler) batchInsert(ctx context.Context, c *fiber.Ctx, table database.TableInfo, dataArray []map[string]interface{}, isUpsert bool, ignoreDuplicates bool, defaultToNull bool, onConflict string) error {
	if len(dataArray) == 0 {
		return c.Status(400).JSON(fiber.Map{
			"error": "Empty array provided",
		})
	}

	// Get all unique columns from the first record
	firstRecord := dataArray[0]
	columns := make([]string, 0, len(firstRecord))
	for col := range firstRecord {
		if !h.columnExists(table, col) {
			return c.Status(400).JSON(fiber.Map{
				"error": fmt.Sprintf("Unknown column: %s", col),
			})
		}
		columns = append(columns, col)
	}

	// Build VALUES clauses
	var valueClauses []string
	var values []interface{}
	argCounter := 1

	for _, record := range dataArray {
		placeholders := make([]string, len(columns))
		for i, col := range columns {
			val, exists := record[col]
			if !exists {
				val = nil // Use NULL for missing columns
			}
			values = append(values, val)
			placeholders[i] = fmt.Sprintf("$%d", argCounter)
			argCounter++
		}
		valueClauses = append(valueClauses, fmt.Sprintf("(%s)", strings.Join(placeholders, ", ")))
	}

	query := fmt.Sprintf(
		"INSERT INTO %s.%s (%s) VALUES %s",
		table.Schema, table.Name,
		strings.Join(columns, ", "),
		strings.Join(valueClauses, ", "),
	)

	// Add ON CONFLICT clause for upsert
	if isUpsert {
		// Use custom conflict target if provided, otherwise auto-detect
		conflictTarget := onConflict
		if conflictTarget == "" {
			conflictTarget = h.getConflictTarget(table)
		}

		if conflictTarget == "" {
			return c.Status(400).JSON(fiber.Map{
				"error": "Cannot perform upsert: table has no primary key or unique constraint",
			})
		}

		// Handle ignore duplicates (DO NOTHING)
		if ignoreDuplicates {
			query += fmt.Sprintf(
				" ON CONFLICT (%s) DO NOTHING",
				conflictTarget,
			)
		} else {
			// Build UPDATE SET clause (all columns except conflict target)
			updateClauses := make([]string, 0)

			// If defaultToNull is true, we need to update ALL columns in the table, not just the ones provided
			if defaultToNull {
				// Get all columns from table and set them either to EXCLUDED.column or NULL
				for _, tableCol := range table.Columns {
					colName := tableCol.Name
					// Skip columns that are part of the conflict target
					if h.isInConflictTarget(colName, conflictTarget) {
						continue
					}

					// Check if column was provided in the data
					columnProvided := false
					for _, providedCol := range columns {
						if providedCol == colName {
							columnProvided = true
							break
						}
					}

					if columnProvided {
						// Use the provided value
						updateClauses = append(updateClauses, fmt.Sprintf("%s = EXCLUDED.%s", colName, colName))
					} else {
						// Set to NULL (missing column)
						updateClauses = append(updateClauses, fmt.Sprintf("%s = NULL", colName))
					}
				}
			} else {
				// Only update columns that were provided
				for _, col := range columns {
					// Skip columns that are part of the conflict target
					if !h.isInConflictTarget(col, conflictTarget) {
						updateClauses = append(updateClauses, fmt.Sprintf("%s = EXCLUDED.%s", col, col))
					}
				}
			}

			if len(updateClauses) > 0 {
				query += fmt.Sprintf(
					" ON CONFLICT (%s) DO UPDATE SET %s",
					conflictTarget,
					strings.Join(updateClauses, ", "),
				)
			} else {
				query += fmt.Sprintf(
					" ON CONFLICT (%s) DO NOTHING",
					conflictTarget,
				)
			}
		}
	}

	query += " RETURNING *"

	// Execute query with RLS context
	var results []map[string]interface{}
	err := middleware.WrapWithRLS(ctx, h.db, c, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, query, values...)
		if err != nil {
			log.Error().Err(err).Str("query", query).Msg("Failed to batch insert records")
			return err
		}
		defer rows.Close()

		// Convert to JSON
		results, err = pgxRowsToJSON(rows)
		return err
	})
	if err != nil {
		return handleDatabaseError(c, err, "create records")
	}

	c.Set("Content-Range", fmt.Sprintf("*/%d", len(results)))
	return c.Status(201).JSON(results)
}

// makePutHandler creates a PUT handler for replacing records
func (h *RESTHandler) makePutHandler(table database.TableInfo) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx := c.Context()
		id := c.Params("id")

		// Parse request body
		var data map[string]interface{}
		if err := c.BodyParser(&data); err != nil {
			return c.Status(400).JSON(fiber.Map{
				"error": "Invalid request body",
			})
		}

		// Determine primary key column
		pkColumn := "id"
		if len(table.PrimaryKey) > 0 {
			pkColumn = table.PrimaryKey[0]
		}

		// Build UPDATE query
		setClauses := make([]string, 0, len(data))
		values := make([]interface{}, 0, len(data)+1)

		i := 1
		for col, val := range data {
			// Skip primary key in update
			if col == pkColumn {
				continue
			}

			// Validate column exists
			if !h.columnExists(table, col) {
				return c.Status(400).JSON(fiber.Map{
					"error": fmt.Sprintf("Unknown column: %s", col),
				})
			}

			setClauses = append(setClauses, fmt.Sprintf("%s = $%d", col, i))
			values = append(values, val)
			i++
		}

		values = append(values, id)

		query := fmt.Sprintf(
			"UPDATE %s.%s SET %s WHERE %s = $%d RETURNING *",
			table.Schema, table.Name,
			strings.Join(setClauses, ", "),
			pkColumn, i,
		)

		// Execute query with RLS context
		var results []map[string]interface{}
		err := middleware.WrapWithRLS(ctx, h.db, c, func(tx pgx.Tx) error {
			rows, err := tx.Query(ctx, query, values...)
			if err != nil {
				log.Error().Err(err).Str("query", query).Msg("Failed to update record")
				return err
			}
			defer rows.Close()

			// Convert to JSON
			results, err = pgxRowsToJSON(rows)
			return err
		})
		if err != nil {
			return handleDatabaseError(c, err, "update record")
		}

		if len(results) == 0 {
			// UPDATE with RETURNING 0 rows could be either RLS blocking or record doesn't exist
			// For authenticated users, assume RLS issue for better debugging (403 vs 404)
			return h.handleRLSViolation(c, "UPDATE", fmt.Sprintf("%s.%s", table.Schema, table.Name))
		}

		return c.JSON(results[0])
	}
}

// makePatchHandler creates a PATCH handler for partial updates
func (h *RESTHandler) makePatchHandler(table database.TableInfo) fiber.Handler {
	// PATCH is the same as PUT but allows partial updates
	return h.makePutHandler(table)
}

// makeDeleteHandler creates a DELETE handler for removing records
func (h *RESTHandler) makeDeleteHandler(table database.TableInfo) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx := c.Context()
		id := c.Params("id")

		// Determine primary key column
		pkColumn := "id"
		if len(table.PrimaryKey) > 0 {
			pkColumn = table.PrimaryKey[0]
		}

		// Build DELETE query
		query := fmt.Sprintf(
			"DELETE FROM %s.%s WHERE %s = $1 RETURNING *",
			table.Schema, table.Name, pkColumn,
		)

		// Execute query with RLS context
		var results []map[string]interface{}
		err := middleware.WrapWithRLS(ctx, h.db, c, func(tx pgx.Tx) error {
			rows, err := tx.Query(ctx, query, id)
			if err != nil {
				return err
			}
			defer rows.Close()

			// Convert to JSON to check if record existed
			results, err = pgxRowsToJSON(rows)
			return err
		})
		if err != nil {
			return handleDatabaseError(c, err, "delete record")
		}

		if len(results) == 0 {
			// DELETE with RETURNING 0 rows could be either RLS blocking or record doesn't exist
			// For authenticated users, assume RLS issue for better debugging (403 vs 404)
			return h.handleRLSViolation(c, "DELETE", fmt.Sprintf("%s.%s", table.Schema, table.Name))
		}

		return c.Status(204).Send(nil)
	}
}

// makeBatchPatchHandler creates a PATCH handler for batch updates with filters
func (h *RESTHandler) makeBatchPatchHandler(table database.TableInfo) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx := c.Context()

		// Parse request body
		var data map[string]interface{}
		if err := c.BodyParser(&data); err != nil {
			return c.Status(400).JSON(fiber.Map{
				"error": "Invalid request body",
			})
		}

		if len(data) == 0 {
			return c.Status(400).JSON(fiber.Map{
				"error": "No fields to update",
			})
		}

		// Parse query parameters for filters
		queries := c.Queries()
		urlValues := make(url.Values)
		for k, v := range queries {
			urlValues.Add(k, v)
		}

		params, err := h.parser.Parse(urlValues)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{
				"error": fmt.Sprintf("Invalid query parameters: %v", err),
			})
		}

		// Build SET clause
		setClauses := make([]string, 0, len(data))
		values := make([]interface{}, 0, len(data))
		argCounter := 1

		for col, val := range data {
			if !h.columnExists(table, col) {
				return c.Status(400).JSON(fiber.Map{
					"error": fmt.Sprintf("Unknown column: %s", col),
				})
			}
			setClauses = append(setClauses, fmt.Sprintf("%s = $%d", col, argCounter))
			values = append(values, val)
			argCounter++
		}

		// Build WHERE clause from filters
		whereSQL, whereArgs := params.buildWhereClause(&argCounter)
		values = append(values, whereArgs...)

		// Build UPDATE query
		query := fmt.Sprintf(
			"UPDATE %s.%s SET %s",
			table.Schema, table.Name,
			strings.Join(setClauses, ", "),
		)

		if whereSQL != "" {
			query += " WHERE " + whereSQL
		}

		query += " RETURNING *"

		// Execute query with RLS context
		var results []map[string]interface{}
		err = middleware.WrapWithRLS(ctx, h.db, c, func(tx pgx.Tx) error {
			rows, err := tx.Query(ctx, query, values...)
			if err != nil {
				log.Error().Err(err).Str("query", query).Msg("Failed to batch update records")
				return err
			}
			defer rows.Close()

			// Convert to JSON
			results, err = pgxRowsToJSON(rows)
			return err
		})
		if err != nil {
			return handleDatabaseError(c, err, "update records")
		}

		c.Set("Content-Range", fmt.Sprintf("*/%d", len(results)))
		return c.JSON(results)
	}
}

// makeBatchDeleteHandler creates a DELETE handler for batch deletes with filters
func (h *RESTHandler) makeBatchDeleteHandler(table database.TableInfo) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx := c.Context()

		// Parse query parameters for filters
		queries := c.Queries()
		urlValues := make(url.Values)
		for k, v := range queries {
			urlValues.Add(k, v)
		}

		params, err := h.parser.Parse(urlValues)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{
				"error": fmt.Sprintf("Invalid query parameters: %v", err),
			})
		}

		// Require at least one filter for safety
		if len(params.Filters) == 0 {
			return c.Status(400).JSON(fiber.Map{
				"error": "Batch delete requires at least one filter. Use DELETE /:id for single deletes",
			})
		}

		// Build WHERE clause from filters
		argCounter := 1
		whereSQL, whereArgs := params.buildWhereClause(&argCounter)

		// Build DELETE query
		query := fmt.Sprintf(
			"DELETE FROM %s.%s WHERE %s RETURNING *",
			table.Schema, table.Name, whereSQL,
		)

		// Execute query with RLS context
		var results []map[string]interface{}
		err = middleware.WrapWithRLS(ctx, h.db, c, func(tx pgx.Tx) error {
			rows, err := tx.Query(ctx, query, whereArgs...)
			if err != nil {
				log.Error().Err(err).Str("query", query).Msg("Failed to batch delete records")
				return err
			}
			defer rows.Close()

			// Convert to JSON
			results, err = pgxRowsToJSON(rows)
			return err
		})
		if err != nil {
			return handleDatabaseError(c, err, "delete records")
		}

		c.Set("Content-Range", fmt.Sprintf("*/%d", len(results)))
		return c.Status(200).JSON(fiber.Map{
			"deleted": len(results),
			"records": results,
		})
	}
}

// buildSelectQuery builds a SELECT query from parameters
func (h *RESTHandler) buildSelectQuery(table database.TableInfo, params *QueryParams) (string, []interface{}) {
	var selectClause string

	// If we have aggregations, use BuildSelectClause (handles aggregations)
	if len(params.Aggregations) > 0 || len(params.GroupBy) > 0 {
		selectClause = params.BuildSelectClause(table.Name)
	} else if len(params.Select) > 0 {
		// Validate and sanitize column names for regular selects
		validColumns := []string{}
		for _, col := range params.Select {
			if h.columnExists(table, col) {
				validColumns = append(validColumns, col)
			}
		}
		if len(validColumns) > 0 {
			selectClause = strings.Join(validColumns, ", ")
		} else {
			selectClause = "*"
		}
	} else {
		selectClause = "*"
	}

	// Start building query
	query := fmt.Sprintf("SELECT %s FROM %s.%s", selectClause, table.Schema, table.Name)

	// Add WHERE, ORDER BY, LIMIT, OFFSET
	whereAndMore, args := params.ToSQL(table.Name)
	if whereAndMore != "" {
		query += " " + whereAndMore
	}

	// Add GROUP BY clause
	groupByClause := params.BuildGroupByClause()
	if groupByClause != "" {
		query += groupByClause
	}

	return query, args
}

// columnExists checks if a column exists in the table
func (h *RESTHandler) columnExists(table database.TableInfo, columnName string) bool {
	for _, col := range table.Columns {
		if col.Name == columnName {
			return true
		}
	}
	return false
}

// getCount gets the row count for a query
func (h *RESTHandler) getCount(ctx context.Context, c *fiber.Ctx, table database.TableInfo, params *QueryParams) (int, error) {
	// Build count query
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s.%s", table.Schema, table.Name)

	// Build WHERE clause
	var args []interface{}
	if len(params.Filters) > 0 {
		argCounter := 1
		whereClause, whereArgs := params.buildWhereClause(&argCounter)
		if whereClause != "" {
			query += " WHERE " + whereClause
			args = whereArgs
		}
	}

	// Execute count query with RLS context
	var count int
	err := middleware.WrapWithRLS(ctx, h.db, c, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, query, args...).Scan(&count)
	})

	return count, err
}

// HandleGetTables returns metadata about available tables
func (h *RESTHandler) HandleGetTables(c *fiber.Ctx) error {
	ctx := c.Context()

	// Get schema parameter
	schemas := []string{"public"}
	if schemaParam := c.Query("schema"); schemaParam != "" {
		schemas = strings.Split(schemaParam, ",")
	}

	// Get all tables
	tables, err := h.db.Inspector().GetAllTables(ctx, schemas...)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to get tables",
		})
	}

	// Format response
	response := make([]fiber.Map, 0, len(tables))
	for _, table := range tables {
		response = append(response, fiber.Map{
			"schema":      table.Schema,
			"name":        table.Name,
			"path":        h.BuildFullTablePath(table),
			"columns":     table.Columns,
			"primary_key": table.PrimaryKey,
			"rls_enabled": table.RLSEnabled,
		})
	}

	return c.JSON(response)
}

// pgxRowsToJSON converts pgx rows to JSON-serializable format
func pgxRowsToJSON(rows pgx.Rows) ([]map[string]interface{}, error) {
	// Get column descriptions
	fields := rows.FieldDescriptions()

	results := []map[string]interface{}{}

	for rows.Next() {
		// Create a slice to hold the values
		values := make([]interface{}, len(fields))
		valuePtrs := make([]interface{}, len(fields))

		for i := range values {
			valuePtrs[i] = &values[i]
		}

		// Scan the row
		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}

		// Build the result map
		row := make(map[string]interface{})
		for i, field := range fields {
			columnName := string(field.Name)

			// Handle special types
			switch v := values[i].(type) {
			case []byte:
				// Try to parse as JSON
				var jsonData interface{}
				if err := json.Unmarshal(v, &jsonData); err == nil {
					row[columnName] = jsonData
				} else {
					// If not JSON, convert to string
					row[columnName] = string(v)
				}
			case [16]byte:
				// Convert UUID bytes to string
				uid, err := uuid.FromBytes(v[:])
				if err == nil {
					row[columnName] = uid.String()
				} else {
					row[columnName] = v
				}
			default:
				row[columnName] = v
			}
		}

		results = append(results, row)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
}

// getConflictTarget determines the conflict target for ON CONFLICT clause
// Returns the primary key columns as a comma-separated string, or empty string if no PK exists
func (h *RESTHandler) getConflictTarget(table database.TableInfo) string {
	if len(table.PrimaryKey) == 0 {
		return ""
	}
	return strings.Join(table.PrimaryKey, ", ")
}

// isInConflictTarget checks if a column is part of the conflict target
func (h *RESTHandler) isInConflictTarget(column string, conflictTarget string) bool {
	// Split conflict target by comma and check if column is in the list
	targets := strings.Split(conflictTarget, ", ")
	for _, target := range targets {
		if strings.TrimSpace(target) == column {
			return true
		}
	}
	return false
}
