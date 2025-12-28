package api

import (
	"context"
	"errors"

	"github.com/fluxbase-eu/fluxbase/internal/settings"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
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

	// If is_secret is true, encrypt the value using the secret settings path
	if req.IsSecret {
		// Extract string value from the map for encryption
		valueStr := extractStringValueFromMap(req.Value)
		if valueStr == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Secret value must be a non-empty string (use {\"value\": \"your-secret\"})",
			})
		}

		secretReq := settings.CreateSecretSettingRequest{
			Key:         req.Key,
			Value:       valueStr,
			Description: req.Description,
		}

		// System secret (userID = nil for system-level)
		metadata, err := h.settingsService.CreateSecretSetting(ctx, secretReq, nil, userID)
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
			log.Error().Err(err).Str("key", req.Key).Msg("Failed to create secret setting")
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to create setting",
			})
		}

		log.Info().
			Str("key", req.Key).
			Str("user_id", userID.String()).
			Str("user_role", userRole.(string)).
			Bool("is_secret", true).
			Msg("Secret setting created")

		return c.Status(fiber.StatusCreated).JSON(metadata)
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
// GET /api/v1/admin/settings/custom/*
func (h *CustomSettingsHandler) GetSetting(c *fiber.Ctx) error {
	ctx := context.Background()
	key := c.Params("*")

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
// PUT /api/v1/admin/settings/custom/*
func (h *CustomSettingsHandler) UpdateSetting(c *fiber.Ctx) error {
	ctx := context.Background()
	key := c.Params("*")

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
// DELETE /api/v1/admin/settings/custom/*
func (h *CustomSettingsHandler) DeleteSetting(c *fiber.Ctx) error {
	ctx := context.Background()
	key := c.Params("*")

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

// CreateSecretSetting creates a new encrypted system-level secret setting
// POST /api/v1/admin/settings/custom/secret
func (h *CustomSettingsHandler) CreateSecretSetting(c *fiber.Ctx) error {
	ctx := context.Background()

	// Get user ID from context
	userIDStr := c.Locals("user_id")
	if userIDStr == nil {
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

	var req settings.CreateSecretSettingRequest
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

	if req.Value == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Setting value is required",
		})
	}

	// Create system secret (userID = nil for system-level)
	metadata, err := h.settingsService.CreateSecretSetting(ctx, req, nil, userID)
	if err != nil {
		if errors.Is(err, settings.ErrCustomSettingDuplicate) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": "A secret setting with this key already exists",
				"code":  "DUPLICATE_KEY",
			})
		}
		if errors.Is(err, settings.ErrCustomSettingInvalidKey) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid setting key format",
				"code":  "INVALID_KEY",
			})
		}
		log.Error().Err(err).Str("key", req.Key).Msg("Failed to create secret setting")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create secret setting",
		})
	}

	log.Info().
		Str("key", req.Key).
		Str("user_id", userID.String()).
		Msg("System secret setting created")

	return c.Status(fiber.StatusCreated).JSON(metadata)
}

// GetSecretSetting returns metadata for a system-level secret setting (never returns the value)
// GET /api/v1/admin/settings/custom/secret/*
func (h *CustomSettingsHandler) GetSecretSetting(c *fiber.Ctx) error {
	ctx := context.Background()
	key := c.Params("*")

	if key == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Setting key is required",
		})
	}

	// Get system secret (userID = nil)
	metadata, err := h.settingsService.GetSecretSettingMetadata(ctx, key, nil)
	if err != nil {
		if errors.Is(err, settings.ErrCustomSettingNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Secret setting not found",
			})
		}
		log.Error().Err(err).Str("key", key).Msg("Failed to get secret setting")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve secret setting",
		})
	}

	return c.JSON(metadata)
}

// UpdateSecretSetting updates a system-level secret setting
// PUT /api/v1/admin/settings/custom/secret/*
func (h *CustomSettingsHandler) UpdateSecretSetting(c *fiber.Ctx) error {
	ctx := context.Background()
	key := c.Params("*")

	if key == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Setting key is required",
		})
	}

	// Get user ID from context
	userIDStr := c.Locals("user_id")
	if userIDStr == nil {
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

	var req settings.UpdateSecretSettingRequest
	if err := c.BodyParser(&req); err != nil {
		log.Error().Err(err).Msg("Failed to parse request body")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Update system secret (userID = nil)
	metadata, err := h.settingsService.UpdateSecretSetting(ctx, key, req, nil, userID)
	if err != nil {
		if errors.Is(err, settings.ErrCustomSettingNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Secret setting not found",
			})
		}
		log.Error().Err(err).Str("key", key).Msg("Failed to update secret setting")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update secret setting",
		})
	}

	log.Info().
		Str("key", key).
		Str("user_id", userID.String()).
		Msg("System secret setting updated")

	return c.JSON(metadata)
}

// DeleteSecretSetting deletes a system-level secret setting
// DELETE /api/v1/admin/settings/custom/secret/*
func (h *CustomSettingsHandler) DeleteSecretSetting(c *fiber.Ctx) error {
	ctx := context.Background()
	key := c.Params("*")

	if key == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Setting key is required",
		})
	}

	// Delete system secret (userID = nil)
	err := h.settingsService.DeleteSecretSetting(ctx, key, nil)
	if err != nil {
		if errors.Is(err, settings.ErrCustomSettingNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Secret setting not found",
			})
		}
		log.Error().Err(err).Str("key", key).Msg("Failed to delete secret setting")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete secret setting",
		})
	}

	log.Info().
		Str("key", key).
		Msg("System secret setting deleted")

	return c.SendStatus(fiber.StatusNoContent)
}

// ListSecretSettings returns metadata for all system-level secret settings
// GET /api/v1/admin/settings/custom/secrets
func (h *CustomSettingsHandler) ListSecretSettings(c *fiber.Ctx) error {
	ctx := context.Background()

	// List system secrets (userID = nil)
	secrets, err := h.settingsService.ListSecretSettings(ctx, nil)
	if err != nil {
		log.Error().Err(err).Msg("Failed to list secret settings")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve secret settings",
		})
	}

	return c.JSON(secrets)
}

// extractStringValueFromMap extracts a string value from a map.
// It looks for a "value" key first, then tries to convert the whole map to a string if it has one key.
func extractStringValueFromMap(m map[string]interface{}) string {
	// Try to get "value" key
	if v, ok := m["value"]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}

	// If map has only one key, try to use its value
	if len(m) == 1 {
		for _, v := range m {
			if s, ok := v.(string); ok {
				return s
			}
		}
	}

	return ""
}
