package migrations

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/fluxbase-eu/fluxbase/internal/database"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// Handler manages HTTP endpoints for migrations
type Handler struct {
	storage     *Storage
	executor    *Executor
	schemaCache *database.SchemaCache
}

// NewHandler creates a new migrations handler
func NewHandler(db *database.Connection, schemaCache *database.SchemaCache) *Handler {
	return &Handler{
		storage:     NewStorage(db),
		executor:    NewExecutor(db),
		schemaCache: schemaCache,
	}
}

// RegisterRoutes registers all migration routes (admin only)
func (h *Handler) RegisterRoutes(app *fiber.App, authMiddleware ...fiber.Handler) {
	migrations := app.Group("/api/v1/admin/migrations", authMiddleware...)

	// CRUD operations
	migrations.Post("/", h.CreateMigration)
	migrations.Get("/", h.ListMigrations)
	migrations.Get("/:name", h.GetMigration)
	migrations.Put("/:name", h.UpdateMigration)
	migrations.Delete("/:name", h.DeleteMigration)

	// Execution operations
	migrations.Post("/:name/apply", h.ApplyMigration)
	migrations.Post("/:name/rollback", h.RollbackMigration)
	migrations.Post("/apply-pending", h.ApplyPending)

	// Bulk operations
	migrations.Post("/sync", h.SyncMigrations)

	// Execution history
	migrations.Get("/:name/executions", h.GetExecutions)
}

// CreateMigration creates a new migration
func (h *Handler) CreateMigration(c *fiber.Ctx) error {
	var req struct {
		Namespace   string  `json:"namespace"`
		Name        string  `json:"name"`
		Description *string `json:"description"`
		UpSQL       string  `json:"up_sql"`
		DownSQL     *string `json:"down_sql"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	// Validation
	if req.Namespace == "" {
		req.Namespace = "default"
	}
	if req.Name == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Migration name is required"})
	}
	if req.UpSQL == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Up SQL is required"})
	}

	// Get user ID from context
	var createdBy *uuid.UUID
	if userID := c.Locals("user_id"); userID != nil {
		if uid, ok := userID.(string); ok {
			parsed, err := uuid.Parse(uid)
			if err == nil {
				createdBy = &parsed
			}
		}
	}

	migration := &Migration{
		Namespace:   req.Namespace,
		Name:        req.Name,
		Description: req.Description,
		UpSQL:       req.UpSQL,
		DownSQL:     req.DownSQL,
		CreatedBy:   createdBy,
	}

	if err := h.storage.CreateMigration(c.Context(), migration); err != nil {
		if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
			return c.Status(409).JSON(fiber.Map{
				"error": fmt.Sprintf("Migration '%s' already exists in namespace '%s'", req.Name, req.Namespace),
			})
		}
		return c.Status(500).JSON(fiber.Map{"error": "Failed to create migration", "details": err.Error()})
	}

	return c.Status(201).JSON(migration)
}

// GetMigration retrieves a migration by name
func (h *Handler) GetMigration(c *fiber.Ctx) error {
	name := c.Params("name")
	namespace := c.Query("namespace", "default")

	migration, err := h.storage.GetMigration(c.Context(), namespace, name)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Migration not found"})
	}

	return c.JSON(migration)
}

// ListMigrations lists all migrations in a namespace
func (h *Handler) ListMigrations(c *fiber.Ctx) error {
	namespace := c.Query("namespace", "default")
	status := c.Query("status")

	var statusPtr *string
	if status != "" {
		statusPtr = &status
	}

	migrations, err := h.storage.ListMigrations(c.Context(), namespace, statusPtr)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to list migrations", "details": err.Error()})
	}

	return c.JSON(migrations)
}

// UpdateMigration updates a migration (only if pending)
func (h *Handler) UpdateMigration(c *fiber.Ctx) error {
	name := c.Params("name")
	namespace := c.Query("namespace", "default")

	var updates map[string]interface{}
	if err := c.BodyParser(&updates); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if err := h.storage.UpdateMigration(c.Context(), namespace, name, updates); err != nil {
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "already applied") {
			return c.Status(400).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(500).JSON(fiber.Map{"error": "Failed to update migration", "details": err.Error()})
	}

	// Return updated migration
	migration, err := h.storage.GetMigration(c.Context(), namespace, name)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Migration updated but failed to retrieve"})
	}

	return c.JSON(migration)
}

// DeleteMigration deletes a migration (only if pending)
func (h *Handler) DeleteMigration(c *fiber.Ctx) error {
	name := c.Params("name")
	namespace := c.Query("namespace", "default")

	if err := h.storage.DeleteMigration(c.Context(), namespace, name); err != nil {
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "already applied") {
			return c.Status(400).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(500).JSON(fiber.Map{"error": "Failed to delete migration", "details": err.Error()})
	}

	return c.JSON(fiber.Map{"message": "Migration deleted successfully"})
}

// ApplyMigration applies a single migration
func (h *Handler) ApplyMigration(c *fiber.Ctx) error {
	name := c.Params("name")

	var req struct {
		Namespace string `json:"namespace"`
	}
	if err := c.BodyParser(&req); err != nil {
		req.Namespace = "default"
	}
	if req.Namespace == "" {
		req.Namespace = "default"
	}

	// Get user ID
	var executedBy *uuid.UUID
	if userID := c.Locals("user_id"); userID != nil {
		if uid, ok := userID.(string); ok {
			parsed, err := uuid.Parse(uid)
			if err == nil {
				executedBy = &parsed
			}
		}
	}

	if err := h.executor.ApplyMigration(c.Context(), req.Namespace, name, executedBy); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to apply migration", "details": err.Error()})
	}

	// Invalidate schema cache so REST API reflects changes immediately
	if h.schemaCache != nil {
		h.schemaCache.Invalidate()
		log.Debug().Str("migration", name).Msg("Schema cache invalidated after applying migration")
	}

	return c.JSON(fiber.Map{"message": "Migration applied successfully"})
}

// RollbackMigration rolls back a migration
func (h *Handler) RollbackMigration(c *fiber.Ctx) error {
	name := c.Params("name")

	var req struct {
		Namespace string `json:"namespace"`
	}
	if err := c.BodyParser(&req); err != nil {
		req.Namespace = "default"
	}
	if req.Namespace == "" {
		req.Namespace = "default"
	}

	// Get user ID
	var executedBy *uuid.UUID
	if userID := c.Locals("user_id"); userID != nil {
		if uid, ok := userID.(string); ok {
			parsed, err := uuid.Parse(uid)
			if err == nil {
				executedBy = &parsed
			}
		}
	}

	if err := h.executor.RollbackMigration(c.Context(), req.Namespace, name, executedBy); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to rollback migration", "details": err.Error()})
	}

	// Invalidate schema cache so REST API reflects changes immediately
	if h.schemaCache != nil {
		h.schemaCache.Invalidate()
		log.Debug().Str("migration", name).Msg("Schema cache invalidated after rolling back migration")
	}

	return c.JSON(fiber.Map{"message": "Migration rolled back successfully"})
}

// ApplyPending applies all pending migrations in order
func (h *Handler) ApplyPending(c *fiber.Ctx) error {
	var req struct {
		Namespace string `json:"namespace"`
	}
	if err := c.BodyParser(&req); err != nil {
		req.Namespace = "default"
	}
	if req.Namespace == "" {
		req.Namespace = "default"
	}

	// Get user ID
	var executedBy *uuid.UUID
	if userID := c.Locals("user_id"); userID != nil {
		if uid, ok := userID.(string); ok {
			parsed, err := uuid.Parse(uid)
			if err == nil {
				executedBy = &parsed
			}
		}
	}

	applied, failed, err := h.executor.ApplyPendingMigrations(c.Context(), req.Namespace, executedBy)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error":   "Failed to apply pending migrations",
			"details": err.Error(),
			"applied": applied,
			"failed":  failed,
		})
	}

	// Invalidate schema cache if any migrations were applied
	if len(applied) > 0 && h.schemaCache != nil {
		h.schemaCache.Invalidate()
		log.Debug().Int("count", len(applied)).Msg("Schema cache invalidated after applying pending migrations")
	}

	return c.JSON(fiber.Map{
		"message": fmt.Sprintf("Applied %d migrations successfully", len(applied)),
		"applied": applied,
		"failed":  failed,
	})
}

// GetExecutions returns execution history for a migration
func (h *Handler) GetExecutions(c *fiber.Ctx) error {
	name := c.Params("name")
	namespace := c.Query("namespace", "default")
	limit := c.QueryInt("limit", 50)

	if limit > 100 {
		limit = 100
	}

	migration, err := h.storage.GetMigration(c.Context(), namespace, name)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Migration not found"})
	}

	logs, err := h.storage.GetExecutionLogs(c.Context(), migration.ID, limit)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to get execution logs", "details": err.Error()})
	}

	return c.JSON(logs)
}

// SyncMigrations performs bulk sync with smart diffing
func (h *Handler) SyncMigrations(c *fiber.Ctx) error {
	var req struct {
		Namespace  string `json:"namespace"`
		Migrations []struct {
			Name        string  `json:"name"`
			Description *string `json:"description"`
			UpSQL       string  `json:"up_sql"`
			DownSQL     *string `json:"down_sql"`
		} `json:"migrations"`
		Options struct {
			UpdateIfChanged bool `json:"update_if_changed"`
			AutoApply       bool `json:"auto_apply"`
			DryRun          bool `json:"dry_run"`
		} `json:"options"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if req.Namespace == "" {
		req.Namespace = "default"
	}

	// Get user ID
	var createdBy *uuid.UUID
	if userID := c.Locals("user_id"); userID != nil {
		if uid, ok := userID.(string); ok {
			parsed, err := uuid.Parse(uid)
			if err == nil {
				createdBy = &parsed
			}
		}
	}

	// Get existing migrations
	existing, err := h.storage.ListMigrations(c.Context(), req.Namespace, nil)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to list existing migrations"})
	}

	// Build lookup map
	existingMap := make(map[string]*Migration)
	for i := range existing {
		existingMap[existing[i].Name] = &existing[i]
	}

	// Track results
	summary := struct {
		Created   int `json:"created"`
		Updated   int `json:"updated"`
		Unchanged int `json:"unchanged"`
		Skipped   int `json:"skipped"`
		Applied   int `json:"applied"`
		Errors    int `json:"errors"`
	}{}

	details := struct {
		Created   []string `json:"created"`
		Updated   []string `json:"updated"`
		Unchanged []string `json:"unchanged"`
		Skipped   []string `json:"skipped"`
		Applied   []string `json:"applied"`
		Errors    []string `json:"errors"`
	}{}

	warnings := []string{}

	// Track if any migration failed during auto-apply
	// Migrations must be applied sequentially - stop on first failure
	autoApplyFailed := false

	// Process each migration
	for _, reqMig := range req.Migrations {
		// Stop processing if a previous migration failed during auto-apply
		if autoApplyFailed {
			summary.Skipped++
			details.Skipped = append(details.Skipped, reqMig.Name)
			warnings = append(warnings, fmt.Sprintf("Migration '%s' skipped due to previous failure", reqMig.Name))
			continue
		}

		// Calculate content hash
		contentHash := calculateHash(reqMig.UpSQL + valueOrEmpty(reqMig.DownSQL))

		existingMig, exists := existingMap[reqMig.Name]

		if !exists {
			// New migration - create it
			if !req.Options.DryRun {
				newMig := &Migration{
					Namespace:   req.Namespace,
					Name:        reqMig.Name,
					Description: reqMig.Description,
					UpSQL:       reqMig.UpSQL,
					DownSQL:     reqMig.DownSQL,
					CreatedBy:   createdBy,
				}
				if err := h.storage.CreateMigration(c.Context(), newMig); err != nil {
					summary.Errors++
					details.Errors = append(details.Errors, fmt.Sprintf("%s: %v", reqMig.Name, err))
					continue
				}
			}
			summary.Created++
			details.Created = append(details.Created, reqMig.Name)

			// Auto-apply if requested
			if req.Options.AutoApply && !req.Options.DryRun {
				if err := h.executor.ApplyMigration(c.Context(), req.Namespace, reqMig.Name, createdBy); err != nil {
					log.Error().Err(err).Str("name", reqMig.Name).Msg("Failed to auto-apply migration")
					summary.Errors++
					details.Errors = append(details.Errors, fmt.Sprintf("%s: failed to apply - %v", reqMig.Name, err))
					autoApplyFailed = true // Stop processing subsequent migrations
				} else {
					summary.Applied++
					details.Applied = append(details.Applied, reqMig.Name)
				}
			}
			continue
		}

		// Migration exists - check if content changed
		existingHash := calculateHash(existingMig.UpSQL + valueOrEmpty(existingMig.DownSQL))

		if existingHash == contentHash {
			// Content unchanged - check if we should retry failed migrations
			if existingMig.Status == "failed" && req.Options.AutoApply && !req.Options.DryRun {
				// Retry failed migration
				log.Info().Str("name", reqMig.Name).Msg("Retrying failed migration")
				if err := h.executor.ApplyMigration(c.Context(), req.Namespace, reqMig.Name, createdBy); err != nil {
					log.Error().Err(err).Str("name", reqMig.Name).Msg("Failed to retry migration")
					summary.Errors++
					details.Errors = append(details.Errors, fmt.Sprintf("%s: retry failed - %v", reqMig.Name, err))
					autoApplyFailed = true // Stop processing subsequent migrations
				} else {
					summary.Applied++
					details.Applied = append(details.Applied, reqMig.Name)
				}
				continue
			}

			// Content unchanged - no action needed
			summary.Unchanged++
			details.Unchanged = append(details.Unchanged, reqMig.Name)
			continue
		}

		// Content changed
		if existingMig.Status == "pending" && req.Options.UpdateIfChanged {
			// Update pending migration
			if !req.Options.DryRun {
				updates := map[string]interface{}{
					"up_sql":   reqMig.UpSQL,
					"down_sql": reqMig.DownSQL,
				}
				if reqMig.Description != nil {
					updates["description"] = *reqMig.Description
				}
				if err := h.storage.UpdateMigration(c.Context(), req.Namespace, reqMig.Name, updates); err != nil {
					summary.Errors++
					details.Errors = append(details.Errors, fmt.Sprintf("%s: %v", reqMig.Name, err))
					continue
				}
			}
			summary.Updated++
			details.Updated = append(details.Updated, reqMig.Name)
			continue
		}

		// Already applied with different content - skip and warn
		if existingMig.Status == "applied" {
			summary.Skipped++
			details.Skipped = append(details.Skipped, reqMig.Name)
			warnings = append(warnings, fmt.Sprintf("Migration '%s' already applied with different content (skipped)", reqMig.Name))
			continue
		}

		// Failed migration with changed content - update and retry if autoApply
		if existingMig.Status == "failed" && req.Options.UpdateIfChanged {
			// Update failed migration with new content
			if !req.Options.DryRun {
				updates := map[string]interface{}{
					"up_sql":   reqMig.UpSQL,
					"down_sql": reqMig.DownSQL,
					"status":   "pending", // Reset to pending for retry
				}
				if reqMig.Description != nil {
					updates["description"] = *reqMig.Description
				}
				if err := h.storage.UpdateMigration(c.Context(), req.Namespace, reqMig.Name, updates); err != nil {
					summary.Errors++
					details.Errors = append(details.Errors, fmt.Sprintf("%s: %v", reqMig.Name, err))
					continue
				}
			}
			summary.Updated++
			details.Updated = append(details.Updated, reqMig.Name)

			// Auto-apply if requested
			if req.Options.AutoApply && !req.Options.DryRun {
				log.Info().Str("name", reqMig.Name).Msg("Retrying updated failed migration")
				if err := h.executor.ApplyMigration(c.Context(), req.Namespace, reqMig.Name, createdBy); err != nil {
					log.Error().Err(err).Str("name", reqMig.Name).Msg("Failed to apply updated migration")
					summary.Errors++
					details.Errors = append(details.Errors, fmt.Sprintf("%s: failed to apply after update - %v", reqMig.Name, err))
					autoApplyFailed = true // Stop processing subsequent migrations
				} else {
					summary.Applied++
					details.Applied = append(details.Applied, reqMig.Name)
				}
			}
			continue
		}

		// Rolled back status or failed without updateIfChanged - skip
		summary.Skipped++
		details.Skipped = append(details.Skipped, reqMig.Name)
		warnings = append(warnings, fmt.Sprintf("Migration '%s' has status '%s' (skipped)", reqMig.Name, existingMig.Status))
	}

	// Invalidate schema cache if any migrations were applied
	if summary.Applied > 0 && h.schemaCache != nil {
		h.schemaCache.Invalidate()
		log.Info().Int("applied", summary.Applied).Msg("Schema cache invalidated after sync")
	}

	// Build response message
	message := fmt.Sprintf("Sync complete: %d created, %d updated, %d unchanged", summary.Created, summary.Updated, summary.Unchanged)
	if summary.Errors > 0 {
		message = fmt.Sprintf("Sync completed with errors: %d created, %d updated, %d unchanged, %d errors", summary.Created, summary.Updated, summary.Unchanged, summary.Errors)
	}

	response := fiber.Map{
		"message":   message,
		"namespace": req.Namespace,
		"summary":   summary,
		"details":   details,
		"dry_run":   req.Options.DryRun,
	}

	if len(warnings) > 0 {
		response["warnings"] = warnings
	}

	// Return appropriate status code
	// 422 Unprocessable Entity if there were errors (partial failure in batch operation)
	if summary.Errors > 0 {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(response)
	}

	return c.JSON(response)
}

// Helper functions

func calculateHash(content string) string {
	hash := sha256.Sum256([]byte(content))
	return hex.EncodeToString(hash[:])
}

func valueOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
