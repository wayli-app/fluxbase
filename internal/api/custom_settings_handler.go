package api

import (
	"context"
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/wayli-app/fluxbase/internal/settings"
)

// CustomSettingsHandler handles custom settings operations
type CustomSettingsHandler struct {
	settingsService *settings.CustomSettingsService
}

// NewCustomSettingsHandler creates a new custom settings handler
func NewCustomSettingsHandler(settingsService *settings.CustomSettingsService) *CustomSettingsHandler {
	return &CustomSettingsHandler{
		settingsService: settingsService,
	}
}

// CreateSetting creates a new custom setting
// POST /api/v1/admin/settings/custom
func (h *CustomSettingsHandler) CreateSetting(c *fiber.Ctx) error {
	ctx := context.Background()

	// Get user ID and role from context (set by auth middleware)
	userIDStr := c.Locals("user_id")
	userRole := c.Locals("user_role")

	if userIDStr == nil || userRole == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authentication required",
		})
	}

	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		log.Error().Err(err).Msg("Invalid user ID in context")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Invalid user ID",
		})
	}

	var req settings.CreateCustomSettingRequest
	if err := c.BodyParser(&req); err != nil {
		log.Error().Err(err).Msg("Failed to parse request body")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate required fields
	if req.Key == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Setting key is required",
		})
	}

	if req.Value == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Setting value is required",
		})
	}

	setting, err := h.settingsService.CreateSetting(ctx, req, userID)
	if err != nil {
		if errors.Is(err, settings.ErrCustomSettingDuplicate) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": "A setting with this key already exists",
				"code":  "DUPLICATE_KEY",
			})
		}
		if errors.Is(err, settings.ErrCustomSettingInvalidKey) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid setting key format",
				"code":  "INVALID_KEY",
			})
		}
		log.Error().Err(err).Str("key", req.Key).Msg("Failed to create custom setting")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create setting",
		})
	}

	log.Info().
		Str("key", req.Key).
		Str("user_id", userID.String()).
		Str("user_role", userRole.(string)).
		Msg("Custom setting created")

	return c.Status(fiber.StatusCreated).JSON(setting)
}

// ListSettings returns all custom settings
// GET /api/v1/admin/settings/custom
func (h *CustomSettingsHandler) ListSettings(c *fiber.Ctx) error {
	ctx := context.Background()

	// Get user role from context
	userRole := c.Locals("user_role")
	if userRole == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authentication required",
		})
	}

	settings, err := h.settingsService.ListSettings(ctx, userRole.(string))
	if err != nil {
		log.Error().Err(err).Msg("Failed to list custom settings")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve custom settings",
		})
	}

	return c.JSON(settings)
}

// GetSetting returns a specific custom setting by key
// GET /api/v1/admin/settings/custom/:key
func (h *CustomSettingsHandler) GetSetting(c *fiber.Ctx) error {
	ctx := context.Background()
	key := c.Params("key")

	if key == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Setting key is required",
		})
	}

	setting, err := h.settingsService.GetSetting(ctx, key)
	if err != nil {
		if errors.Is(err, settings.ErrCustomSettingNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Setting not found",
			})
		}
		log.Error().Err(err).Str("key", key).Msg("Failed to get custom setting")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve setting",
		})
	}

	return c.JSON(setting)
}

// UpdateSetting updates an existing custom setting
// PUT /api/v1/admin/settings/custom/:key
func (h *CustomSettingsHandler) UpdateSetting(c *fiber.Ctx) error {
	ctx := context.Background()
	key := c.Params("key")

	if key == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Setting key is required",
		})
	}

	// Get user ID and role from context
	userIDStr := c.Locals("user_id")
	userRole := c.Locals("user_role")

	if userIDStr == nil || userRole == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authentication required",
		})
	}

	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		log.Error().Err(err).Msg("Invalid user ID in context")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Invalid user ID",
		})
	}

	var req settings.UpdateCustomSettingRequest
	if err := c.BodyParser(&req); err != nil {
		log.Error().Err(err).Msg("Failed to parse request body")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate that at least value is provided
	if req.Value == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Setting value is required",
		})
	}

	setting, err := h.settingsService.UpdateSetting(ctx, key, req, userID, userRole.(string))
	if err != nil {
		if errors.Is(err, settings.ErrCustomSettingNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Setting not found",
			})
		}
		if errors.Is(err, settings.ErrCustomSettingPermissionDenied) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "You do not have permission to edit this setting",
				"code":  "PERMISSION_DENIED",
			})
		}
		log.Error().Err(err).Str("key", key).Msg("Failed to update custom setting")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update setting",
		})
	}

	log.Info().
		Str("key", key).
		Str("user_id", userID.String()).
		Str("user_role", userRole.(string)).
		Msg("Custom setting updated")

	return c.JSON(setting)
}

// DeleteSetting deletes a custom setting
// DELETE /api/v1/admin/settings/custom/:key
func (h *CustomSettingsHandler) DeleteSetting(c *fiber.Ctx) error {
	ctx := context.Background()
	key := c.Params("key")

	if key == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Setting key is required",
		})
	}

	// Get user role from context
	userRole := c.Locals("user_role")
	if userRole == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authentication required",
		})
	}

	err := h.settingsService.DeleteSetting(ctx, key, userRole.(string))
	if err != nil {
		if errors.Is(err, settings.ErrCustomSettingNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Setting not found",
			})
		}
		if errors.Is(err, settings.ErrCustomSettingPermissionDenied) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "You do not have permission to delete this setting",
				"code":  "PERMISSION_DENIED",
			})
		}
		log.Error().Err(err).Str("key", key).Msg("Failed to delete custom setting")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete setting",
		})
	}

	log.Info().
		Str("key", key).
		Str("user_role", userRole.(string)).
		Msg("Custom setting deleted")

	return c.SendStatus(fiber.StatusNoContent)
}
