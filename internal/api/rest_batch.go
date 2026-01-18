package api

import (
	"context"
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

// batchInsert handles batch insert operations
func (h *RESTHandler) batchInsert(ctx context.Context, c *fiber.Ctx, table database.TableInfo, dataArray []map[string]interface{}, isUpsert bool, ignoreDuplicates bool, defaultToNull bool, onConflict string) error {
	if len(dataArray) == 0 {
		return c.Status(400).JSON(fiber.Map{
			"error": "Empty array provided",
		})
	}

	// Get all unique columns from the first record
	firstRecord := dataArray[0]
	columns := make([]string, 0, len(firstRecord))     // Quoted column names for SQL
	columnNames := make([]string, 0, len(firstRecord)) // Unquoted column names for conflict checking
	for col := range firstRecord {
		if !h.columnExists(table, col) {
			return c.Status(400).JSON(fiber.Map{
				"error": fmt.Sprintf("Unknown column: %s", col),
			})
		}
		columns = append(columns, quoteIdentifier(col))
		columnNames = append(columnNames, col)
	}

	// Build VALUES clauses
	var valueClauses []string
	var values []interface{}
	argCounter := 1

	for _, record := range dataArray {
		placeholders := make([]string, len(columnNames))
		for i, col := range columnNames {
			val, exists := record[col]
			if !exists {
				val = nil // Use NULL for missing columns
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
				values = append(values, string(geoJSON))
				placeholders[i] = fmt.Sprintf("ST_GeomFromGeoJSON($%d)", argCounter)
			} else {
				values = append(values, val)
				placeholders[i] = fmt.Sprintf("$%d", argCounter)
			}
			argCounter++
		}
		valueClauses = append(valueClauses, fmt.Sprintf("(%s)", strings.Join(placeholders, ", ")))
	}

	query := fmt.Sprintf(
		`INSERT INTO "%s"."%s" (%s) VALUES %s`,
		table.Schema, table.Name,
		strings.Join(columns, ", "),
		strings.Join(valueClauses, ", "),
	)

	// Add ON CONFLICT clause for upsert
	if isUpsert {
		// Use custom conflict target if provided, otherwise auto-detect
		var conflictTarget string
		var conflictTargetColumns []string

		if onConflict != "" {
			// Validate and quote custom conflict target columns
			conflictCols := strings.Split(onConflict, ",")
			quotedConflictCols := make([]string, 0, len(conflictCols))
			for _, col := range conflictCols {
				col = strings.TrimSpace(col)
				if !h.columnExists(table, col) {
					return c.Status(400).JSON(fiber.Map{
						"error": fmt.Sprintf("Unknown column in on_conflict: %s", col),
					})
				}
				quotedConflictCols = append(quotedConflictCols, quoteIdentifier(col))
				conflictTargetColumns = append(conflictTargetColumns, col)
			}
			conflictTarget = strings.Join(quotedConflictCols, ", ")
		} else {
			conflictTarget = h.getConflictTarget(table)
			conflictTargetColumns = h.getConflictTargetUnquoted(table)
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
					if h.isInConflictTarget(colName, conflictTargetColumns) {
						continue
					}

					quotedColName := quoteIdentifier(colName)

					// Check if column was provided in the data
					columnProvided := false
					for _, providedCol := range columnNames {
						if providedCol == colName {
							columnProvided = true
							break
						}
					}

					if columnProvided {
						// Use the provided value
						updateClauses = append(updateClauses, fmt.Sprintf("%s = EXCLUDED.%s", quotedColName, quotedColName))
					} else {
						// Set to NULL (missing column)
						updateClauses = append(updateClauses, fmt.Sprintf("%s = NULL", quotedColName))
					}
				}
			} else {
				// Only update columns that were provided
				for i, col := range columns {
					// Skip columns that are part of the conflict target (use unquoted name for comparison)
					if !h.isInConflictTarget(columnNames[i], conflictTargetColumns) {
						// col is already quoted
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

	query += buildReturningClause(table)

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

	// Set affected count headers
	affectedCount := len(results)
	c.Set("Content-Range", fmt.Sprintf("*/%d", affectedCount))
	c.Set("X-Affected-Count", fmt.Sprintf("%d", affectedCount))

	// Check Prefer header for response format
	prefer := c.Get("Prefer")
	switch {
	case strings.Contains(prefer, "return=minimal"):
		return c.Status(201).Send(nil)
	case strings.Contains(prefer, "return=headers-only"):
		return c.Status(201).JSON(fiber.Map{"affected": affectedCount})
	default:
		// return=representation or no preference - return full records
		return c.Status(201).JSON(results)
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

		// Parse raw query string to preserve multiple values for the same key
		rawQuery := string(c.Request().URI().QueryString())
		urlValues, err := url.ParseQuery(rawQuery)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{
				"error": fmt.Sprintf("Invalid query string: %v", err),
			})
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

			quotedCol := quoteIdentifier(col)
			// Check if value is GeoJSON and needs PostGIS conversion
			if isGeoJSON(val) {
				// Convert GeoJSON to JSON string and use ST_GeomFromGeoJSON
				geoJSON, err := json.Marshal(val)
				if err != nil {
					return c.Status(400).JSON(fiber.Map{
						"error": fmt.Sprintf("Invalid GeoJSON for column %s: %v", col, err),
					})
				}
				setClauses = append(setClauses, fmt.Sprintf("%s = ST_GeomFromGeoJSON($%d)", quotedCol, argCounter))
				values = append(values, string(geoJSON))
			} else {
				setClauses = append(setClauses, fmt.Sprintf("%s = $%d", quotedCol, argCounter))
				values = append(values, val)
			}
			argCounter++
		}

		// Build WHERE clause from filters
		whereSQL, whereArgs := params.buildWhereClause(&argCounter)
		values = append(values, whereArgs...)

		// Build UPDATE query
		query := fmt.Sprintf(
			`UPDATE "%s"."%s" SET %s`,
			table.Schema, table.Name,
			strings.Join(setClauses, ", "),
		)

		if whereSQL != "" {
			query += " WHERE " + whereSQL
		}

		query += buildReturningClause(table)

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

		// Set affected count headers
		affectedCount := len(results)
		c.Set("Content-Range", fmt.Sprintf("*/%d", affectedCount))
		c.Set("X-Affected-Count", fmt.Sprintf("%d", affectedCount))

		// Check Prefer header for response format
		prefer := c.Get("Prefer")
		switch {
		case strings.Contains(prefer, "return=minimal"):
			return c.Status(200).Send(nil)
		case strings.Contains(prefer, "return=headers-only"):
			return c.JSON(fiber.Map{"affected": affectedCount})
		default:
			// return=representation or no preference - return full records
			return c.JSON(results)
		}
	}
}

// makeBatchDeleteHandler creates a DELETE handler for batch deletes with filters
func (h *RESTHandler) makeBatchDeleteHandler(table database.TableInfo) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx := c.Context()

		// Parse raw query string to preserve multiple values for the same key
		rawQuery := string(c.Request().URI().QueryString())
		urlValues, err := url.ParseQuery(rawQuery)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{
				"error": fmt.Sprintf("Invalid query string: %v", err),
			})
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
			`DELETE FROM "%s"."%s" WHERE %s`,
			table.Schema, table.Name, whereSQL,
		) + buildReturningClause(table)

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

		// Set affected count headers
		affectedCount := len(results)
		c.Set("Content-Range", fmt.Sprintf("*/%d", affectedCount))
		c.Set("X-Affected-Count", fmt.Sprintf("%d", affectedCount))

		// Check Prefer header for response format
		prefer := c.Get("Prefer")
		switch {
		case strings.Contains(prefer, "return=minimal"):
			return c.Status(200).Send(nil)
		case strings.Contains(prefer, "return=headers-only"):
			return c.Status(200).JSON(fiber.Map{"affected": affectedCount})
		default:
			// return=representation or no preference - return deleted records with count
			return c.Status(200).JSON(fiber.Map{
				"affected": affectedCount,
				"records":  results,
			})
		}
	}
}
