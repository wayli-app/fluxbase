package api

import (
	"context"
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5"

	"github.com/nimbleflux/fluxbase/internal/auth"
	"github.com/nimbleflux/fluxbase/internal/database"
	"github.com/nimbleflux/fluxbase/internal/middleware"
)

// BulkOperationsHandler handles bulk data operations
type BulkOperationsHandler struct {
	db          *database.Connection
	authService *auth.Service
	schemaCache *database.SchemaCache
}

// NewBulkOperationsHandler creates a new bulk operations handler
func NewBulkOperationsHandler(db *database.Connection, authService *auth.Service, schemaCache *database.SchemaCache) *BulkOperationsHandler {
	return &BulkOperationsHandler{
		db:          db,
		authService: authService,
		schemaCache: schemaCache,
	}
}

// BulkActionRequest represents a bulk action request
type BulkActionRequest struct {
	Action  string   `json:"action"`          // delete, export
	Targets []string `json:"targets"`         // Array of IDs
	Table   string   `json:"table,omitempty"` // Optional table name (derived from context if not provided)
}

// parseTableIdentifier parses a table identifier in the format "schema.table" or "table"
// Returns (schema, table, error)
func parseTableIdentifier(tableIdentifier string) (string, string, error) {
	parts := strings.SplitN(tableIdentifier, ".", 2)
	if len(parts) == 2 {
		schema := strings.TrimSpace(parts[0])
		table := strings.TrimSpace(parts[1])
		if schema == "" || table == "" {
			return "", "", fmt.Errorf("invalid table identifier: %s", tableIdentifier)
		}
		return schema, table, nil
	}
	// Default to public schema
	table := strings.TrimSpace(parts[0])
	if table == "" {
		return "", "", fmt.Errorf("invalid table identifier: %s", tableIdentifier)
	}
	return "public", table, nil
}

// HandleBulkAction processes a bulk action request
func (h *BulkOperationsHandler) HandleBulkAction(c fiber.Ctx) error {
	var req BulkActionRequest
	if err := ParseBody(c, &req); err != nil {
		return err
	}

	// Validate request
	if req.Action == "" {
		return SendMissingField(c, "action")
	}

	if len(req.Targets) == 0 {
		return SendBadRequest(c, "At least one target ID is required", ErrCodeMissingField)
	}

	if len(req.Targets) > 1000 {
		return SendBadRequest(c, "Too many targets (max 1000)", ErrCodeInvalidInput)
	}

	// Get table name from request or context
	tableName := req.Table
	if tableName == "" {
		// Try to get from query parameter
		tableName = c.Query("table", "")
		if tableName == "" {
			return SendMissingField(c, "table")
		}
	}

	// Parse schema and table name
	schema, table, err := parseTableIdentifier(tableName)
	if err != nil {
		return SendBadRequest(c, fmt.Sprintf("Invalid table name: %v", err), ErrCodeInvalidInput)
	}

	// Get table metadata to verify it exists and get primary key
	tableInfo, exists, err := h.schemaCache.GetTable(c.RequestCtx(), schema, table)
	if err != nil {
		return SendInternalError(c, "Failed to lookup table")
	}
	if !exists {
		return SendNotFound(c, fmt.Sprintf("Table not found: %s", tableName))
	}

	// Get primary key column
	pkColumn := "id"
	if len(tableInfo.PrimaryKey) > 0 {
		pkColumn = tableInfo.PrimaryKey[0]
	}

	ctx := c.RequestCtx()

	switch req.Action {
	case "delete":
		return h.handleBulkDelete(c, ctx, schema, table, pkColumn, req.Targets)

	case "export":
		return h.handleBulkExport(c, ctx, schema, table, pkColumn, req.Targets)

	default:
		return SendBadRequest(c, fmt.Sprintf("Unknown action: %s. Supported actions: delete, export", req.Action), ErrCodeInvalidInput)
	}
}

// handleBulkDelete performs a bulk delete operation
func (h *BulkOperationsHandler) handleBulkDelete(c fiber.Ctx, ctx context.Context, schema, table, pkColumn string, targetIds []string) error {
	// Build DELETE query with RLS
	quotedTableName := quoteIdentifier(schema) + "." + quoteIdentifier(table)
	quotedPKColumn := quoteIdentifier(pkColumn)
	query := fmt.Sprintf("DELETE FROM %s WHERE %s = ANY($1)", quotedTableName, quotedPKColumn)

	// Set target schema for tenant-aware pool routing
	middleware.SetTargetSchema(c, schema)

	// Execute with RLS context
	var rowsAffected int64
	err := middleware.WrapWithRLS(ctx, h.db, c, func(tx pgx.Tx) error {
		result, err := tx.Exec(ctx, query, targetIds)
		if err != nil {
			return err
		}
		rowsAffected = result.RowsAffected()
		return nil
	})
	if err != nil {
		return SendInternalError(c, "Failed to delete records")
	}

	return c.JSON(fiber.Map{
		"success":       true,
		"message":       fmt.Sprintf("Deleted %d records", rowsAffected),
		"rows_affected": rowsAffected,
	})
}

// handleBulkExport performs a bulk export operation
func (h *BulkOperationsHandler) handleBulkExport(c fiber.Ctx, ctx context.Context, schema, table, pkColumn string, targetIds []string) error {
	// Build SELECT query
	quotedTableName := quoteIdentifier(schema) + "." + quoteIdentifier(table)
	quotedPKColumn := quoteIdentifier(pkColumn)
	query := fmt.Sprintf("SELECT * FROM %s WHERE %s = ANY($1)", quotedTableName, quotedPKColumn)

	// Set target schema for tenant-aware pool routing
	middleware.SetTargetSchema(c, schema)

	// Execute with RLS context
	var results []map[string]interface{}
	err := middleware.WrapWithRLS(ctx, h.db, c, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, query, targetIds)
		if err != nil {
			return err
		}
		defer rows.Close()

		results, err = pgxRowsToJSON(rows)
		return err
	})
	if err != nil {
		return SendInternalError(c, "Failed to export records")
	}

	return c.JSON(fiber.Map{
		"success": true,
		"count":   len(results),
		"records": results,
	})
}
