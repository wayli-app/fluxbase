package api

import (
	"context"
	"fmt"
	"strings"

	"github.com/fluxbase-eu/fluxbase/internal/database"
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"
)

// RealtimeAdminHandler handles realtime enablement for user tables
type RealtimeAdminHandler struct {
	db *database.Connection
}

// NewRealtimeAdminHandler creates a new realtime admin handler
func NewRealtimeAdminHandler(db *database.Connection) *RealtimeAdminHandler {
	return &RealtimeAdminHandler{db: db}
}

// EnableRealtimeRequest represents a request to enable realtime on a table
type EnableRealtimeRequest struct {
	Schema  string   `json:"schema"`
	Table   string   `json:"table"`
	Events  []string `json:"events,omitempty"`  // INSERT, UPDATE, DELETE (default: all)
	Exclude []string `json:"exclude,omitempty"` // Columns to exclude from notifications
}

// EnableRealtimeResponse represents the response after enabling realtime
type EnableRealtimeResponse struct {
	Schema      string   `json:"schema"`
	Table       string   `json:"table"`
	Events      []string `json:"events"`
	TriggerName string   `json:"trigger_name"`
	Exclude     []string `json:"exclude,omitempty"`
}

// RealtimeTableStatus represents the status of a realtime-enabled table
type RealtimeTableStatus struct {
	ID              int      `json:"id"`
	Schema          string   `json:"schema"`
	Table           string   `json:"table"`
	RealtimeEnabled bool     `json:"realtime_enabled"`
	Events          []string `json:"events"`
	ExcludedColumns []string `json:"excluded_columns,omitempty"`
	CreatedAt       string   `json:"created_at"`
	UpdatedAt       string   `json:"updated_at"`
}

// HandleEnableRealtime enables realtime on a table
func (h *RealtimeAdminHandler) HandleEnableRealtime(c *fiber.Ctx) error {
	var req EnableRealtimeRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Default schema to public
	if req.Schema == "" {
		req.Schema = "public"
	}

	// Validate schema name
	if err := validateIdentifier(req.Schema, "schema"); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Validate table name
	if req.Table == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "Table name is required",
		})
	}
	if err := validateIdentifier(req.Table, "table"); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Validate and set default events
	if len(req.Events) == 0 {
		req.Events = []string{"INSERT", "UPDATE", "DELETE"}
	} else {
		for _, event := range req.Events {
			if event != "INSERT" && event != "UPDATE" && event != "DELETE" {
				return c.Status(400).JSON(fiber.Map{
					"error": fmt.Sprintf("Invalid event type: %s. Must be INSERT, UPDATE, or DELETE", event),
				})
			}
		}
	}

	// Validate excluded columns
	for _, col := range req.Exclude {
		if err := validateIdentifier(col, "column"); err != nil {
			return c.Status(400).JSON(fiber.Map{
				"error": fmt.Sprintf("Invalid excluded column: %s", err.Error()),
			})
		}
	}

	// Prevent enabling realtime on system schemas
	systemSchemas := map[string]bool{
		"pg_catalog":         true,
		"information_schema": true,
		"dashboard":          true,
		"auth":               true, // auth tables have their own triggers if needed
		"realtime":           true,
	}
	if systemSchemas[req.Schema] {
		return c.Status(400).JSON(fiber.Map{
			"error": fmt.Sprintf("Cannot enable realtime on system schema '%s'", req.Schema),
		})
	}

	ctx := c.Context()

	// Check if table exists
	exists, err := h.tableExists(ctx, req.Schema, req.Table)
	if err != nil {
		log.Error().Err(err).Str("table", req.Schema+"."+req.Table).Msg("Failed to check table existence")
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to check table existence",
		})
	}
	if !exists {
		return c.Status(404).JSON(fiber.Map{
			"error": fmt.Sprintf("Table '%s.%s' does not exist", req.Schema, req.Table),
		})
	}

	triggerName := fmt.Sprintf("%s_realtime_notify", req.Table)

	// Execute all DDL in a transaction with admin role
	err = h.db.ExecuteWithAdminRole(ctx, func(conn *pgx.Conn) error {
		// Start transaction
		tx, txErr := conn.Begin(ctx)
		if txErr != nil {
			return fmt.Errorf("failed to begin transaction: %w", txErr)
		}
		defer tx.Rollback(ctx) //nolint:errcheck

		// 1. Set REPLICA IDENTITY FULL (required for UPDATE/DELETE to include old values)
		replicaQuery := fmt.Sprintf("ALTER TABLE %s.%s REPLICA IDENTITY FULL",
			quoteIdentifier(req.Schema), quoteIdentifier(req.Table))
		log.Debug().Str("query", replicaQuery).Msg("Setting REPLICA IDENTITY FULL")
		if _, execErr := tx.Exec(ctx, replicaQuery); execErr != nil {
			return fmt.Errorf("failed to set REPLICA IDENTITY: %w", execErr)
		}

		// 2. Drop existing trigger if any
		dropQuery := fmt.Sprintf("DROP TRIGGER IF EXISTS %s ON %s.%s",
			quoteIdentifier(triggerName), quoteIdentifier(req.Schema), quoteIdentifier(req.Table))
		log.Debug().Str("query", dropQuery).Msg("Dropping existing trigger")
		if _, execErr := tx.Exec(ctx, dropQuery); execErr != nil {
			return fmt.Errorf("failed to drop existing trigger: %w", execErr)
		}

		// 3. Create trigger
		triggerQuery := fmt.Sprintf(`CREATE TRIGGER %s
AFTER INSERT OR UPDATE OR DELETE ON %s.%s
FOR EACH ROW EXECUTE FUNCTION public.notify_realtime_change()`,
			quoteIdentifier(triggerName), quoteIdentifier(req.Schema), quoteIdentifier(req.Table))
		log.Debug().Str("query", triggerQuery).Msg("Creating realtime trigger")
		if _, execErr := tx.Exec(ctx, triggerQuery); execErr != nil {
			return fmt.Errorf("failed to create trigger: %w", execErr)
		}

		// 4. Upsert into realtime.schema_registry
		upsertQuery := `
INSERT INTO realtime.schema_registry (schema_name, table_name, realtime_enabled, events, excluded_columns)
VALUES ($1, $2, true, $3, $4)
ON CONFLICT (schema_name, table_name) DO UPDATE
SET realtime_enabled = true,
    events = EXCLUDED.events,
    excluded_columns = EXCLUDED.excluded_columns,
    updated_at = NOW()`
		log.Debug().Str("query", upsertQuery).Msg("Upserting schema registry")
		if _, execErr := tx.Exec(ctx, upsertQuery, req.Schema, req.Table, req.Events, req.Exclude); execErr != nil {
			return fmt.Errorf("failed to update schema registry: %w", execErr)
		}

		return tx.Commit(ctx)
	})

	if err != nil {
		log.Error().Err(err).Str("table", req.Schema+"."+req.Table).Msg("Failed to enable realtime")
		return c.Status(500).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to enable realtime: %v", err),
		})
	}

	log.Info().
		Str("schema", req.Schema).
		Str("table", req.Table).
		Strs("events", req.Events).
		Strs("exclude", req.Exclude).
		Msg("Realtime enabled on table")

	return c.Status(201).JSON(EnableRealtimeResponse{
		Schema:      req.Schema,
		Table:       req.Table,
		Events:      req.Events,
		TriggerName: triggerName,
		Exclude:     req.Exclude,
	})
}

// HandleDisableRealtime disables realtime on a table
func (h *RealtimeAdminHandler) HandleDisableRealtime(c *fiber.Ctx) error {
	schema := c.Params("schema")
	table := c.Params("table")

	// Validate identifiers
	if err := validateIdentifier(schema, "schema"); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	if err := validateIdentifier(table, "table"); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
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

	triggerName := fmt.Sprintf("%s_realtime_notify", table)

	// Execute DDL with admin role
	err = h.db.ExecuteWithAdminRole(ctx, func(conn *pgx.Conn) error {
		tx, txErr := conn.Begin(ctx)
		if txErr != nil {
			return fmt.Errorf("failed to begin transaction: %w", txErr)
		}
		defer tx.Rollback(ctx) //nolint:errcheck

		// 1. Drop the trigger
		dropQuery := fmt.Sprintf("DROP TRIGGER IF EXISTS %s ON %s.%s",
			quoteIdentifier(triggerName), quoteIdentifier(schema), quoteIdentifier(table))
		log.Debug().Str("query", dropQuery).Msg("Dropping realtime trigger")
		if _, execErr := tx.Exec(ctx, dropQuery); execErr != nil {
			return fmt.Errorf("failed to drop trigger: %w", execErr)
		}

		// 2. Update registry (set realtime_enabled = false, keep record for history)
		updateQuery := `
UPDATE realtime.schema_registry
SET realtime_enabled = false, updated_at = NOW()
WHERE schema_name = $1 AND table_name = $2`
		log.Debug().Str("query", updateQuery).Msg("Updating schema registry")
		if _, execErr := tx.Exec(ctx, updateQuery, schema, table); execErr != nil {
			return fmt.Errorf("failed to update schema registry: %w", execErr)
		}

		return tx.Commit(ctx)
	})

	if err != nil {
		log.Error().Err(err).Str("table", schema+"."+table).Msg("Failed to disable realtime")
		return c.Status(500).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to disable realtime: %v", err),
		})
	}

	log.Info().Str("schema", schema).Str("table", table).Msg("Realtime disabled on table")

	return c.JSON(fiber.Map{
		"success": true,
		"message": fmt.Sprintf("Realtime disabled on table '%s.%s'", schema, table),
	})
}

// HandleListRealtimeTables lists all realtime-enabled tables
func (h *RealtimeAdminHandler) HandleListRealtimeTables(c *fiber.Ctx) error {
	ctx := c.Context()

	// Get optional filter for enabled-only
	enabledOnly := c.Query("enabled", "true") == "true"

	query := `
SELECT id, schema_name, table_name, realtime_enabled, events,
       COALESCE(excluded_columns, '{}') as excluded_columns,
       created_at, updated_at
FROM realtime.schema_registry`

	if enabledOnly {
		query += " WHERE realtime_enabled = true"
	}

	query += " ORDER BY schema_name, table_name"

	rows, err := h.db.Pool().Query(ctx, query)
	if err != nil {
		log.Error().Err(err).Msg("Failed to list realtime tables")
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to list realtime tables",
		})
	}
	defer rows.Close()

	tables := []RealtimeTableStatus{}
	for rows.Next() {
		var t RealtimeTableStatus
		var createdAt, updatedAt interface{}
		if err := rows.Scan(&t.ID, &t.Schema, &t.Table, &t.RealtimeEnabled, &t.Events, &t.ExcludedColumns, &createdAt, &updatedAt); err != nil {
			log.Error().Err(err).Msg("Failed to scan realtime table row")
			continue
		}
		if createdAt != nil {
			t.CreatedAt = fmt.Sprintf("%v", createdAt)
		}
		if updatedAt != nil {
			t.UpdatedAt = fmt.Sprintf("%v", updatedAt)
		}
		tables = append(tables, t)
	}

	return c.JSON(fiber.Map{
		"tables": tables,
		"count":  len(tables),
	})
}

// HandleGetRealtimeStatus gets the realtime status for a specific table
func (h *RealtimeAdminHandler) HandleGetRealtimeStatus(c *fiber.Ctx) error {
	schema := c.Params("schema")
	table := c.Params("table")

	// Validate identifiers
	if err := validateIdentifier(schema, "schema"); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	if err := validateIdentifier(table, "table"); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}

	ctx := c.Context()

	query := `
SELECT id, schema_name, table_name, realtime_enabled, events,
       COALESCE(excluded_columns, '{}') as excluded_columns,
       created_at, updated_at
FROM realtime.schema_registry
WHERE schema_name = $1 AND table_name = $2`

	var t RealtimeTableStatus
	var createdAt, updatedAt interface{}
	err := h.db.Pool().QueryRow(ctx, query, schema, table).Scan(
		&t.ID, &t.Schema, &t.Table, &t.RealtimeEnabled, &t.Events, &t.ExcludedColumns, &createdAt, &updatedAt,
	)

	if err == pgx.ErrNoRows {
		// Table not registered - check if the table exists at all
		exists, checkErr := h.tableExists(ctx, schema, table)
		if checkErr != nil {
			return c.Status(500).JSON(fiber.Map{
				"error": "Failed to check table existence",
			})
		}
		if !exists {
			return c.Status(404).JSON(fiber.Map{
				"error": fmt.Sprintf("Table '%s.%s' does not exist", schema, table),
			})
		}
		// Table exists but not registered for realtime
		return c.JSON(RealtimeTableStatus{
			Schema:          schema,
			Table:           table,
			RealtimeEnabled: false,
			Events:          []string{},
			ExcludedColumns: []string{},
		})
	}
	if err != nil {
		log.Error().Err(err).Str("table", schema+"."+table).Msg("Failed to get realtime status")
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to get realtime status",
		})
	}

	if createdAt != nil {
		t.CreatedAt = fmt.Sprintf("%v", createdAt)
	}
	if updatedAt != nil {
		t.UpdatedAt = fmt.Sprintf("%v", updatedAt)
	}

	return c.JSON(t)
}

// HandleUpdateRealtimeConfig updates the realtime configuration for a table
func (h *RealtimeAdminHandler) HandleUpdateRealtimeConfig(c *fiber.Ctx) error {
	schema := c.Params("schema")
	table := c.Params("table")

	// Validate identifiers
	if err := validateIdentifier(schema, "schema"); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	if err := validateIdentifier(table, "table"); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}

	var req struct {
		Events  []string `json:"events,omitempty"`
		Exclude []string `json:"exclude,omitempty"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	ctx := c.Context()

	// Build update query dynamically
	updates := []string{}
	args := []interface{}{schema, table}
	argIdx := 3

	if len(req.Events) > 0 {
		for _, event := range req.Events {
			if event != "INSERT" && event != "UPDATE" && event != "DELETE" {
				return c.Status(400).JSON(fiber.Map{
					"error": fmt.Sprintf("Invalid event type: %s", event),
				})
			}
		}
		updates = append(updates, fmt.Sprintf("events = $%d", argIdx))
		args = append(args, req.Events)
		argIdx++
	}

	if req.Exclude != nil { // Allow empty array to clear exclusions
		for _, col := range req.Exclude {
			if err := validateIdentifier(col, "column"); err != nil {
				return c.Status(400).JSON(fiber.Map{
					"error": fmt.Sprintf("Invalid excluded column: %s", err.Error()),
				})
			}
		}
		updates = append(updates, fmt.Sprintf("excluded_columns = $%d", argIdx))
		args = append(args, req.Exclude)
	}

	if len(updates) == 0 {
		return c.Status(400).JSON(fiber.Map{
			"error": "No updates provided",
		})
	}

	query := fmt.Sprintf(`
UPDATE realtime.schema_registry
SET %s, updated_at = NOW()
WHERE schema_name = $1 AND table_name = $2 AND realtime_enabled = true
RETURNING id`, strings.Join(updates, ", "))

	var id int
	err := h.db.Pool().QueryRow(ctx, query, args...).Scan(&id)
	if err == pgx.ErrNoRows {
		return c.Status(404).JSON(fiber.Map{
			"error": fmt.Sprintf("Realtime not enabled on table '%s.%s'", schema, table),
		})
	}
	if err != nil {
		log.Error().Err(err).Str("table", schema+"."+table).Msg("Failed to update realtime config")
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to update realtime configuration",
		})
	}

	log.Info().Str("schema", schema).Str("table", table).Msg("Realtime config updated")

	return c.JSON(fiber.Map{
		"success": true,
		"message": fmt.Sprintf("Realtime configuration updated for '%s.%s'", schema, table),
	})
}

// tableExists checks if a table exists in the database
func (h *RealtimeAdminHandler) tableExists(ctx context.Context, schema, table string) (bool, error) {
	var exists bool
	err := h.db.Pool().QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM information_schema.tables
			WHERE table_schema = $1 AND table_name = $2
		)
	`, schema, table).Scan(&exists)
	return exists, err
}
