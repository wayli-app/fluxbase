package api

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"

	apperrors "github.com/nimbleflux/fluxbase/internal/errors"

	"github.com/nimbleflux/fluxbase/internal/auth"
	"github.com/nimbleflux/fluxbase/internal/database"
	"github.com/nimbleflux/fluxbase/internal/middleware"
)

// DataExportHandler handles data export operations
type DataExportHandler struct {
	db          *database.Connection
	authService *auth.Service
	schemaCache *database.SchemaCache
}

// NewDataExportHandler creates a new data export handler
func NewDataExportHandler(db *database.Connection, authService *auth.Service, schemaCache *database.SchemaCache) *DataExportHandler {
	return &DataExportHandler{
		db:          db,
		authService: authService,
		schemaCache: schemaCache,
	}
}

// HandleDataExport processes a data export request
func (h *DataExportHandler) HandleDataExport(c fiber.Ctx) error {
	format := c.Query("format", "csv")
	selectedItems := c.Query("items", "[]")
	tableName := c.Query("table", "")

	// Validate request
	if format != "csv" && format != "json" {
		return apperrors.SendBadRequest(c, "Invalid export format. Must be 'csv' or 'json'", apperrors.ErrCodeInvalidFormat)
	}

	if tableName == "" {
		return apperrors.SendMissingField(c, "table")
	}

	// Parse selected items as JSON array
	var targetIds []string
	if err := json.Unmarshal([]byte(selectedItems), &targetIds); err != nil {
		return apperrors.SendBadRequest(c, fmt.Sprintf("Invalid items format: %v", err), apperrors.ErrCodeInvalidFormat)
	}

	if len(targetIds) == 0 {
		return apperrors.SendBadRequest(c, "At least one item ID is required", apperrors.ErrCodeMissingField)
	}

	if len(targetIds) > 1000 {
		return apperrors.SendBadRequest(c, "Too many items (max 1000)", apperrors.ErrCodeInvalidInput)
	}

	// Parse schema and table name
	schema, table, err := parseTableIdentifier(tableName)
	if err != nil {
		return apperrors.SendBadRequest(c, fmt.Sprintf("Invalid table name: %v", err), apperrors.ErrCodeInvalidFormat)
	}

	// Get table metadata
	tableInfo, exists, err := h.schemaCache.GetTable(c.RequestCtx(), schema, table)
	if err != nil {
		log.Error().Err(err).Str("table", tableName).Msg("Failed to lookup table")
		return apperrors.SendInternalError(c, "Internal error")
	}
	if !exists {
		return apperrors.SendNotFound(c, fmt.Sprintf("Table not found: %s", tableName))
	}

	// Get primary key column
	pkColumn := "id"
	if len(tableInfo.PrimaryKey) > 0 {
		pkColumn = tableInfo.PrimaryKey[0]
	}

	ctx := c.RequestCtx()

	// Build SELECT query
	quotedTableName := quoteIdentifier(schema) + "." + quoteIdentifier(table)
	quotedPKColumn := quoteIdentifier(pkColumn)
	query := fmt.Sprintf("SELECT * FROM %s WHERE %s = ANY($1) ORDER BY %s", quotedTableName, quotedPKColumn, quotedPKColumn)

	// Set target schema for tenant-aware pool routing
	middleware.SetTargetSchema(c, schema)

	// Execute query with RLS
	var results []map[string]interface{}
	err = middleware.WrapWithRLS(ctx, h.db, c, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, query, targetIds)
		if err != nil {
			return err
		}
		defer rows.Close()

		results, err = pgxRowsToJSON(rows)
		return err
	})
	if err != nil {
		log.Error().Err(err).Str("table", tableName).Msg("Failed to export records")
		return apperrors.SendInternalError(c, "Internal error")
	}

	if len(results) == 0 {
		return apperrors.SendNotFound(c, "No records found")
	}

	// Format based on requested format
	switch format {
	case "csv":
		return h.exportAsCSV(c, tableInfo, results)
	case "json":
		return c.JSON(results)
	default:
		return apperrors.SendBadRequest(c, "Invalid export format", apperrors.ErrCodeInvalidFormat)
	}
}

// exportAsCSV converts results to CSV format
func (h *DataExportHandler) exportAsCSV(c fiber.Ctx, tableInfo *database.TableInfo, results []map[string]interface{}) error {
	if len(results) == 0 {
		return apperrors.SendNotFound(c, "No records to export")
	}

	// Get column names from table metadata
	columns := make([]string, 0, len(tableInfo.Columns))
	for _, col := range tableInfo.Columns {
		columns = append(columns, col.Name)
	}

	// Create CSV writer
	var csvData strings.Builder
	writer := csv.NewWriter(&csvData)

	// Write header
	if err := writer.Write(columns); err != nil {
		log.Error().Err(err).Msg("Failed to write CSV header")
		return apperrors.SendInternalError(c, "Internal error")
	}

	// Write rows
	for _, row := range results {
		record := make([]string, len(columns))
		for i, col := range columns {
			val := row[col]
			record[i] = formatValue(val)
		}
		if err := writer.Write(record); err != nil {
			log.Error().Err(err).Msg("Failed to write CSV row")
			return apperrors.SendInternalError(c, "Internal error")
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		log.Error().Err(err).Msg("Failed to generate CSV")
		return apperrors.SendInternalError(c, "Internal error")
	}

	// Set headers for CSV download
	c.Set("Content-Type", "text/csv")
	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=export_%d.csv", len(results)))

	return c.SendString(csvData.String())
}

// formatValue converts a value to string for CSV output
func formatValue(v interface{}) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case float64:
		return fmt.Sprintf("%.6f", val)
	case int, int64:
		return fmt.Sprintf("%d", val)
	case bool:
		return fmt.Sprintf("%t", val)
	default:
		return fmt.Sprintf("%v", val)
	}
}
