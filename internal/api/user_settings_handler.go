package api

import (
	"context"
	"errors"

	"github.com/fluxbase-eu/fluxbase/internal/settings"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// UserSettingsHandler handles user-specific secret settings operations
type UserSettingsHandler struct {
	settingsService *settings.CustomSettingsService
}

// NewUserSettingsHandler creates a new user settings handler
func NewUserSettingsHandler(settingsService *settings.CustomSettingsService) *UserSettingsHandler {
	return &UserSettingsHandler{
		settingsService: settingsService,
	}
}

// CreateSecret creates a new encrypted user-specific secret setting
// POST /api/v1/settings/secret
func (h *UserSettingsHandler) CreateSecret(c *fiber.Ctx) error {
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

	// Create user-specific secret
	metadata, err := h.settingsService.CreateSecretSetting(ctx, req, &userID, userID)
	if err != nil {
		if errors.Is(err, settings.ErrCustomSettingDuplicate) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": "A secret with this key already exists",
				"code":  "DUPLICATE_KEY",
			})
		}
		if errors.Is(err, settings.ErrCustomSettingInvalidKey) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid setting key format",
				"code":  "INVALID_KEY",
			})
		}
		log.Error().Err(err).Str("key", req.Key).Str("user_id", userID.String()).Msg("Failed to create user secret")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create secret",
		})
	}

	log.Info().
		Str("key", req.Key).
		Str("user_id", userID.String()).
		Msg("User secret created")

	return c.Status(fiber.StatusCreated).JSON(metadata)
}

// GetSecret returns metadata for a user's secret setting (never returns the value)
// GET /api/v1/settings/secret/*
func (h *UserSettingsHandler) GetSecret(c *fiber.Ctx) error {
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

	// Get user's secret
	metadata, err := h.settingsService.GetSecretSettingMetadata(ctx, key, &userID)
	if err != nil {
		if errors.Is(err, settings.ErrCustomSettingNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Secret not found",
			})
		}
		log.Error().Err(err).Str("key", key).Str("user_id", userID.String()).Msg("Failed to get user secret")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve secret",
		})
	}

	return c.JSON(metadata)
}

// UpdateSecret updates a user's secret setting
// PUT /api/v1/settings/secret/*
func (h *UserSettingsHandler) UpdateSecret(c *fiber.Ctx) error {
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

	// Update user's secret
	metadata, err := h.settingsService.UpdateSecretSetting(ctx, key, req, &userID, userID)
	if err != nil {
		if errors.Is(err, settings.ErrCustomSettingNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Secret not found",
			})
		}
		log.Error().Err(err).Str("key", key).Str("user_id", userID.String()).Msg("Failed to update user secret")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update secret",
		})
	}

	log.Info().
		Str("key", key).
		Str("user_id", userID.String()).
		Msg("User secret updated")

	return c.JSON(metadata)
}

// DeleteSecret deletes a user's secret setting
// DELETE /api/v1/settings/secret/*
func (h *UserSettingsHandler) DeleteSecret(c *fiber.Ctx) error {
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

	// Delete user's secret
	err = h.settingsService.DeleteSecretSetting(ctx, key, &userID)
	if err != nil {
		if errors.Is(err, settings.ErrCustomSettingNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Secret not found",
			})
		}
		log.Error().Err(err).Str("key", key).Str("user_id", userID.String()).Msg("Failed to delete user secret")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete secret",
		})
	}

	log.Info().
		Str("key", key).
		Str("user_id", userID.String()).
		Msg("User secret deleted")

	return c.SendStatus(fiber.StatusNoContent)
}

// ListSecrets returns metadata for all user's secret settings
// GET /api/v1/settings/secrets
func (h *UserSettingsHandler) ListSecrets(c *fiber.Ctx) error {
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

	// List user's secrets
	secrets, err := h.settingsService.ListSecretSettings(ctx, &userID)
	if err != nil {
		log.Error().Err(err).Str("user_id", userID.String()).Msg("Failed to list user secrets")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve secrets",
		})
	}

	return c.JSON(secrets)
}
