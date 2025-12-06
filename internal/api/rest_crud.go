package api

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/fluxbase-eu/fluxbase/internal/database"
	"github.com/fluxbase-eu/fluxbase/internal/middleware"
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"
)

// makeGetHandler creates a GET handler for listing records
func (h *RESTHandler) makeGetHandler(table database.TableInfo) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx := c.Context()

		// Parse raw query string to preserve multiple values for the same key
		// (e.g., recorded_at=gte.2025-01-01&recorded_at=lte.2025-12-31)
		// Note: c.Queries() returns map[string]string which loses duplicate keys
		rawQuery := string(c.Request().URI().QueryString())
		urlValues, err := url.ParseQuery(rawQuery)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{
				"error": fmt.Sprintf("Invalid query string: %v", err),
			})
		}

		// Parse query parameters
		params, err := h.parser.Parse(urlValues)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{
				"error": fmt.Sprintf("Invalid query parameters: %v", err),
			})
		}

		// Build SELECT query using fresh metadata
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
			log.Debug().Int("count", len(results)).Msg("Query results")
			return err
		})
		if err != nil {
			log.Error().Err(err).Str("table", fmt.Sprintf("%s.%s", table.Schema, table.Name)).Msg("Failed to fetch records")
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

		// Build query - quote identifiers to prevent SQL injection
		query := fmt.Sprintf(
			`SELECT * FROM "%s"."%s" WHERE "%s" = $1`,
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

			// Check if value is GeoJSON and needs PostGIS conversion
			if isGeoJSON(val) {
				// Convert GeoJSON to JSON string and use ST_GeomFromGeoJSON
				geoJSON, err := json.Marshal(val)
				if err != nil {
					return c.Status(400).JSON(fiber.Map{
						"error": fmt.Sprintf("Invalid GeoJSON for column %s: %v", col, err),
					})
				}
				values = append(values, string(geoJSON))
				placeholders = append(placeholders, fmt.Sprintf("ST_GeomFromGeoJSON($%d)", i))
			} else if isPartialGeoJSON(val) {
				// Value looks like GeoJSON but is incomplete - return validation error
				return c.Status(400).JSON(fiber.Map{
					"error": fmt.Sprintf("Invalid GeoJSON for column %s: missing required 'coordinates' field", col),
				})
			} else {
				values = append(values, val)
				placeholders = append(placeholders, fmt.Sprintf("$%d", i))
			}
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

		query += buildReturningClause(table)

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

			// Check if value is GeoJSON and needs PostGIS conversion
			if isGeoJSON(val) {
				// Convert GeoJSON to JSON string and use ST_GeomFromGeoJSON
				geoJSON, err := json.Marshal(val)
				if err != nil {
					return c.Status(400).JSON(fiber.Map{
						"error": fmt.Sprintf("Invalid GeoJSON for column %s: %v", col, err),
					})
				}
				setClauses = append(setClauses, fmt.Sprintf("%s = ST_GeomFromGeoJSON($%d)", col, i))
				values = append(values, string(geoJSON))
			} else {
				setClauses = append(setClauses, fmt.Sprintf("%s = $%d", col, i))
				values = append(values, val)
			}
			i++
		}

		values = append(values, id)

		query := fmt.Sprintf(
			`UPDATE "%s"."%s" SET %s WHERE "%s" = $%d`,
			table.Schema, table.Name,
			strings.Join(setClauses, ", "),
			pkColumn, i,
		) + buildReturningClause(table)

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

		// Build DELETE query - quote identifiers to prevent SQL injection
		query := fmt.Sprintf(
			`DELETE FROM "%s"."%s" WHERE "%s" = $1`,
			table.Schema, table.Name, pkColumn,
		) + buildReturningClause(table)

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
