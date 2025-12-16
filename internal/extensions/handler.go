package extensions

import (
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
)

// Handler handles extension management HTTP endpoints
type Handler struct {
	service *Service
}

// NewHandler creates a new extension handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// ListExtensions returns all available extensions with their status
// GET /api/v1/admin/extensions
func (h *Handler) ListExtensions(c *fiber.Ctx) error {
	ctx := c.Context()

	response, err := h.service.ListExtensions(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to list extensions")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to list extensions",
		})
	}

	return c.JSON(response)
}

// GetExtensionStatus returns the status of a specific extension
// GET /api/v1/admin/extensions/:name/status
func (h *Handler) GetExtensionStatus(c *fiber.Ctx) error {
	ctx := c.Context()
	name := c.Params("name")

	if name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Extension name is required",
		})
	}

	status, err := h.service.GetExtensionStatus(ctx, name)
	if err != nil {
		log.Error().Err(err).Str("extension", name).Msg("Failed to get extension status")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get extension status",
		})
	}

	return c.JSON(status)
}

// EnableExtension enables a PostgreSQL extension
// POST /api/v1/admin/extensions/:name/enable
func (h *Handler) EnableExtension(c *fiber.Ctx) error {
	ctx := c.Context()
	name := c.Params("name")

	if name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Extension name is required",
		})
	}

	// Parse optional request body
	var req EnableExtensionRequest
	_ = c.BodyParser(&req) // Ignore error - body is optional

	// Get user ID from context if available
	var userID *string
	if uid := c.Locals("user_id"); uid != nil {
		if uidStr, ok := uid.(string); ok {
			userID = &uidStr
		}
	}

	response, err := h.service.EnableExtension(ctx, name, userID, req.Schema)
	if err != nil {
		log.Error().Err(err).Str("extension", name).Msg("Failed to enable extension")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to enable extension",
		})
	}

	if !response.Success {
		return c.Status(fiber.StatusBadRequest).JSON(response)
	}

	return c.JSON(response)
}

// DisableExtension disables a PostgreSQL extension
// POST /api/v1/admin/extensions/:name/disable
func (h *Handler) DisableExtension(c *fiber.Ctx) error {
	ctx := c.Context()
	name := c.Params("name")

	if name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Extension name is required",
		})
	}

	// Get user ID from context if available
	var userID *string
	if uid := c.Locals("user_id"); uid != nil {
		if uidStr, ok := uid.(string); ok {
			userID = &uidStr
		}
	}

	response, err := h.service.DisableExtension(ctx, name, userID)
	if err != nil {
		log.Error().Err(err).Str("extension", name).Msg("Failed to disable extension")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to disable extension",
		})
	}

	if !response.Success {
		return c.Status(fiber.StatusBadRequest).JSON(response)
	}

	return c.JSON(response)
}

// SyncExtensions syncs the extension catalog with PostgreSQL
// POST /api/v1/admin/extensions/sync
func (h *Handler) SyncExtensions(c *fiber.Ctx) error {
	ctx := c.Context()

	err := h.service.SyncFromPostgres(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to sync extensions")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to sync extensions",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Extensions synced successfully",
	})
}
