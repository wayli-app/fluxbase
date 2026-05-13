package api

import (
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"

	apperrors "github.com/nimbleflux/fluxbase/internal/errors"
	"github.com/nimbleflux/fluxbase/internal/logutil"
)

// AddColumnRequest represents a request to add a column to a table
type AddColumnRequest struct {
	Name         string `json:"name"`
	Type         string `json:"type"`
	Nullable     bool   `json:"nullable"`
	DefaultValue string `json:"defaultValue,omitempty"`
}

// AddColumn adds a new column to an existing table
func (h *DDLHandler) AddColumn(c fiber.Ctx) error {
	schema := c.Params("schema")
	table := c.Params("table")

	// Validate identifiers
	if err := validateIdentifier(schema, "schema"); err != nil {
		return SendBadRequest(c, err.Error(), ErrCodeValidationFailed)
	}
	if err := validateIdentifier(table, "table"); err != nil {
		return SendBadRequest(c, err.Error(), ErrCodeValidationFailed)
	}

	var req AddColumnRequest
	if err := ParseBody(c, &req); err != nil {
		return err
	}

	// Validate column name
	if err := validateIdentifier(req.Name, "column"); err != nil {
		return SendBadRequest(c, err.Error(), ErrCodeValidationFailed)
	}

	if err := h.requireDB(c); err != nil {
		return err
	}

	// Validate data type
	dataType := strings.ToLower(strings.TrimSpace(req.Type))
	if !validDataTypes[dataType] {
		return SendBadRequest(c, fmt.Sprintf("Invalid data type '%s'", req.Type), ErrCodeInvalidInput)
	}

	ctx := c.RequestCtx()

	// Check if table exists
	exists, err := h.tableExists(ctx, c, schema, table)
	if err != nil {
		log.Error().Err(err).Str("table", schema+"."+table).Msg("Failed to check table existence")
		return SendOperationFailed(c, "check table existence")
	}
	if !exists {
		return SendNotFound(c, fmt.Sprintf("Table '%s.%s' does not exist", schema, table))
	}

	// Check if column already exists
	colExists, err := h.columnExists(ctx, c, schema, table, req.Name)
	if err != nil {
		log.Error().Err(err).Msg("Failed to check column existence")
		return SendOperationFailed(c, "check column existence")
	}
	if colExists {
		return SendConflict(c, fmt.Sprintf("Column '%s' already exists in table '%s.%s'", req.Name, schema, table), ErrCodeAlreadyExists)
	}

	// Build ALTER TABLE ADD COLUMN statement
	colDef := fmt.Sprintf("%s %s", quoteIdentifier(req.Name), dataType)
	if !req.Nullable {
		colDef += " NOT NULL"
	}
	if req.DefaultValue != "" {
		colDef += fmt.Sprintf(" DEFAULT %s", sanitizeDefaultValue(req.DefaultValue))
	}

	query := fmt.Sprintf("ALTER TABLE %s.%s ADD COLUMN %s",
		quoteIdentifier(schema), quoteIdentifier(table), colDef)

	log.Info().Str("table", schema+"."+table).Str("column", req.Name).Str("operation", logutil.ExtractDDLMetadata(query)).Msg("Adding column")

	err = h.executeWithAdminRole(ctx, c, func(tx pgx.Tx) error {
		_, execErr := tx.Exec(ctx, query)
		return execErr
	})
	if err != nil {
		log.Error().Err(err).Str("table", schema+"."+table).Str("column", req.Name).Msg("Failed to add column")
		return SendInternalError(c, "Failed to add column")
	}

	h.invalidateCache(ctx)
	log.Info().Str("table", schema+"."+table).Str("column", req.Name).Msg("Column added successfully")
	return c.Status(201).JSON(fiber.Map{
		"success": true,
		"message": fmt.Sprintf("Column '%s' added to table '%s.%s'", req.Name, schema, table),
	})
}

// DropColumn removes a column from a table
func (h *DDLHandler) DropColumn(c fiber.Ctx) error {
	schema := c.Params("schema")
	table := c.Params("table")
	column := c.Params("column")

	// Validate identifiers
	if err := validateIdentifier(schema, "schema"); err != nil {
		return SendBadRequest(c, err.Error(), ErrCodeValidationFailed)
	}
	if err := validateIdentifier(table, "table"); err != nil {
		return SendBadRequest(c, err.Error(), ErrCodeValidationFailed)
	}
	if err := validateIdentifier(column, "column"); err != nil {
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
		return SendOperationFailed(c, "check table existence")
	}
	if !exists {
		return SendNotFound(c, fmt.Sprintf("Table '%s.%s' does not exist", schema, table))
	}

	// Check if column exists
	colExists, err := h.columnExists(ctx, c, schema, table, column)
	if err != nil {
		log.Error().Err(err).Msg("Failed to check column existence")
		return SendOperationFailed(c, "check column existence")
	}
	if !colExists {
		return SendNotFound(c, fmt.Sprintf("Column '%s' does not exist in table '%s.%s'", column, schema, table))
	}

	query := fmt.Sprintf("ALTER TABLE %s.%s DROP COLUMN %s",
		quoteIdentifier(schema), quoteIdentifier(table), quoteIdentifier(column))

	log.Info().Str("table", schema+"."+table).Str("column", column).Str("operation", logutil.ExtractDDLMetadata(query)).Msg("Dropping column")

	err = h.executeWithAdminRole(ctx, c, func(tx pgx.Tx) error {
		_, execErr := tx.Exec(ctx, query)
		return execErr
	})
	if err != nil {
		log.Error().Err(err).Str("table", schema+"."+table).Str("column", column).Msg("Failed to drop column")
		return SendInternalError(c, fmt.Sprintf("Failed to drop column: %v", err))
	}

	h.invalidateCache(ctx)
	log.Info().Str("table", schema+"."+table).Str("column", column).Msg("Column dropped successfully")
	return apperrors.SendSuccess(c, fmt.Sprintf("Column '%s' dropped from table '%s.%s'", column, schema, table))
}
