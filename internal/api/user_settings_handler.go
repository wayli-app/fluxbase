package api

import (
	"errors"

	"github.com/fluxbase-eu/fluxbase/internal/database"
	"github.com/fluxbase-eu/fluxbase/internal/middleware"
	"github.com/fluxbase-eu/fluxbase/internal/settings"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"
)

// UserSettingsHandler handles user-specific secret settings operations
type UserSettingsHandler struct {
	db              *database.Connection
	settingsService *settings.CustomSettingsService
	secretsService  *settings.SecretsService
}

// NewUserSettingsHandler creates a new user settings handler
func NewUserSettingsHandler(db *database.Connection, settingsService *settings.CustomSettingsService) *UserSettingsHandler {
	return &UserSettingsHandler{
		db:              db,
		settingsService: settingsService,
	}
}

// SetSecretsService sets the secrets service for decryption operations
func (h *UserSettingsHandler) SetSecretsService(secretsService *settings.SecretsService) {
	h.secretsService = secretsService
}

// CreateSecret creates a new encrypted user-specific secret setting
// POST /api/v1/settings/secret
func (h *UserSettingsHandler) CreateSecret(c *fiber.Ctx) error {
	ctx := c.Context()

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

	// Create user-specific secret with RLS context
	var metadata *settings.SecretSettingMetadata
	err = middleware.WrapWithRLS(ctx, h.db, c, func(tx pgx.Tx) error {
		var txErr error
		metadata, txErr = h.settingsService.CreateSecretSettingWithTx(ctx, tx, req, &userID, userID)
		return txErr
	})
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
	ctx := c.Context()
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

	// Get user's secret with RLS context
	var metadata *settings.SecretSettingMetadata
	err = middleware.WrapWithRLS(ctx, h.db, c, func(tx pgx.Tx) error {
		var txErr error
		metadata, txErr = h.settingsService.GetSecretSettingMetadataWithTx(ctx, tx, key, &userID)
		return txErr
	})
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
	ctx := c.Context()
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

	// Update user's secret with RLS context
	var metadata *settings.SecretSettingMetadata
	err = middleware.WrapWithRLS(ctx, h.db, c, func(tx pgx.Tx) error {
		var txErr error
		metadata, txErr = h.settingsService.UpdateSecretSettingWithTx(ctx, tx, key, req, &userID, userID)
		return txErr
	})
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
	ctx := c.Context()
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

	// Delete user's secret with RLS context
	err = middleware.WrapWithRLS(ctx, h.db, c, func(tx pgx.Tx) error {
		return h.settingsService.DeleteSecretSettingWithTx(ctx, tx, key, &userID)
	})
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
	ctx := c.Context()

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

	// List user's secrets with RLS context
	var secrets []settings.SecretSettingMetadata
	err = middleware.WrapWithRLS(ctx, h.db, c, func(tx pgx.Tx) error {
		var txErr error
		secrets, txErr = h.settingsService.ListSecretSettingsWithTx(ctx, tx, &userID)
		return txErr
	})
	if err != nil {
		log.Error().Err(err).Str("user_id", userID.String()).Msg("Failed to list user secrets")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve secrets",
		})
	}

	return c.JSON(secrets)
}

// GetUserSecretValue retrieves the decrypted value of a specific user's secret
// This is a privileged operation that requires service_role
// GET /api/v1/admin/settings/user/:user_id/secret/:key/decrypt
func (h *UserSettingsHandler) GetUserSecretValue(c *fiber.Ctx) error {
	ctx := c.Context()

	// Require service_role for this privileged operation
	role := c.Locals("user_role")
	if role != "service_role" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "This operation requires service_role",
		})
	}

	// Check if secrets service is configured
	if h.secretsService == nil {
		log.Error().Msg("SecretsService not configured for UserSettingsHandler")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Secrets service not configured",
		})
	}

	// Parse target user ID from URL
	targetUserIDStr := c.Params("user_id")
	targetUserID, err := uuid.Parse(targetUserIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid user_id format",
		})
	}

	// Get secret key from URL
	key := c.Params("key")
	if key == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Secret key is required",
		})
	}

	// Retrieve and decrypt the secret (service_role bypasses RLS)
	value, err := h.secretsService.GetUserSecret(ctx, targetUserID, key)
	if err != nil {
		if errors.Is(err, settings.ErrSecretNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Secret not found",
			})
		}
		if errors.Is(err, settings.ErrDecryptionFailed) {
			log.Error().Err(err).Str("key", key).Str("user_id", targetUserID.String()).Msg("Failed to decrypt user secret")
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to decrypt secret",
			})
		}
		log.Error().Err(err).Str("key", key).Str("user_id", targetUserID.String()).Msg("Failed to retrieve user secret")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve secret",
		})
	}

	log.Debug().
		Str("key", key).
		Str("target_user_id", targetUserID.String()).
		Msg("User secret decrypted via service role")

	return c.JSON(fiber.Map{
		"value": value,
	})
}

// ============================================================================
// User Settings (non-encrypted, with system fallback support)
// These endpoints mirror the edge function secrets helper pattern for regular settings
// ============================================================================

// GetSetting retrieves a setting with user -> system fallback
// GET /api/v1/settings/user/:key
func (h *UserSettingsHandler) GetSetting(c *fiber.Ctx) error {
	ctx := c.Context()
	key := c.Params("key")

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

	// Get setting with fallback using RLS context
	var result *settings.UserSettingWithSource
	err = middleware.WrapWithRLS(ctx, h.db, c, func(tx pgx.Tx) error {
		var txErr error
		result, txErr = h.settingsService.GetUserSettingWithFallbackWithTx(ctx, tx, userID, key)
		return txErr
	})
	if err != nil {
		if errors.Is(err, settings.ErrCustomSettingNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Setting not found",
			})
		}
		log.Error().Err(err).Str("key", key).Str("user_id", userID.String()).Msg("Failed to get setting with fallback")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve setting",
		})
	}

	return c.JSON(result)
}

// GetUserOwnSetting retrieves only the user's own setting (no fallback)
// GET /api/v1/settings/user/own/:key
func (h *UserSettingsHandler) GetUserOwnSetting(c *fiber.Ctx) error {
	ctx := c.Context()
	key := c.Params("key")

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

	// Get user's own setting with RLS context
	var setting *settings.UserSetting
	err = middleware.WrapWithRLS(ctx, h.db, c, func(tx pgx.Tx) error {
		var txErr error
		setting, txErr = h.settingsService.GetUserOwnSettingWithTx(ctx, tx, userID, key)
		return txErr
	})
	if err != nil {
		if errors.Is(err, settings.ErrCustomSettingNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Setting not found",
			})
		}
		log.Error().Err(err).Str("key", key).Str("user_id", userID.String()).Msg("Failed to get user setting")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve setting",
		})
	}

	return c.JSON(setting)
}

// GetSystemSettingPublic retrieves a system-level setting (user_id IS NULL)
// GET /api/v1/settings/user/system/:key
func (h *UserSettingsHandler) GetSystemSettingPublic(c *fiber.Ctx) error {
	ctx := c.Context()
	key := c.Params("key")

	if key == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Setting key is required",
		})
	}

	// Get system setting with RLS context
	var setting *settings.CustomSetting
	err := middleware.WrapWithRLS(ctx, h.db, c, func(tx pgx.Tx) error {
		var txErr error
		setting, txErr = h.settingsService.GetSystemSettingWithTx(ctx, tx, key)
		return txErr
	})
	if err != nil {
		if errors.Is(err, settings.ErrCustomSettingNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Setting not found",
			})
		}
		log.Error().Err(err).Str("key", key).Msg("Failed to get system setting")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve setting",
		})
	}

	// Return only the value (not all metadata) for public access
	return c.JSON(fiber.Map{
		"key":   setting.Key,
		"value": setting.Value,
	})
}

// SetSetting creates or updates a user setting
// PUT /api/v1/settings/user/:key
func (h *UserSettingsHandler) SetSetting(c *fiber.Ctx) error {
	ctx := c.Context()
	key := c.Params("key")

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

	var req settings.CreateUserSettingRequest
	if err := c.BodyParser(&req); err != nil {
		log.Error().Err(err).Msg("Failed to parse request body")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Use key from URL
	req.Key = key

	// Validate required fields
	if req.Value == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Setting value is required",
		})
	}

	// Upsert the setting with RLS context
	var setting *settings.UserSetting
	err = middleware.WrapWithRLS(ctx, h.db, c, func(tx pgx.Tx) error {
		var txErr error
		setting, txErr = h.settingsService.UpsertUserSettingWithTx(ctx, tx, userID, req)
		return txErr
	})
	if err != nil {
		if errors.Is(err, settings.ErrCustomSettingInvalidKey) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid setting key format",
				"code":  "INVALID_KEY",
			})
		}
		log.Error().Err(err).Str("key", key).Str("user_id", userID.String()).Msg("Failed to set user setting")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to save setting",
		})
	}

	log.Info().
		Str("key", key).
		Str("user_id", userID.String()).
		Msg("User setting saved")

	return c.JSON(setting)
}

// DeleteSetting removes a user's setting
// DELETE /api/v1/settings/user/:key
func (h *UserSettingsHandler) DeleteSetting(c *fiber.Ctx) error {
	ctx := c.Context()
	key := c.Params("key")

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

	// Delete user's setting with RLS context
	err = middleware.WrapWithRLS(ctx, h.db, c, func(tx pgx.Tx) error {
		return h.settingsService.DeleteUserSettingWithTx(ctx, tx, userID, key)
	})
	if err != nil {
		if errors.Is(err, settings.ErrCustomSettingNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Setting not found",
			})
		}
		log.Error().Err(err).Str("key", key).Str("user_id", userID.String()).Msg("Failed to delete user setting")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete setting",
		})
	}

	log.Info().
		Str("key", key).
		Str("user_id", userID.String()).
		Msg("User setting deleted")

	return c.SendStatus(fiber.StatusNoContent)
}

// ListSettings returns all user's own settings
// GET /api/v1/settings/user/list
func (h *UserSettingsHandler) ListSettings(c *fiber.Ctx) error {
	ctx := c.Context()

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

	// List user's settings with RLS context
	var userSettings []settings.UserSetting
	err = middleware.WrapWithRLS(ctx, h.db, c, func(tx pgx.Tx) error {
		var txErr error
		userSettings, txErr = h.settingsService.ListUserOwnSettingsWithTx(ctx, tx, userID)
		return txErr
	})
	if err != nil {
		log.Error().Err(err).Str("user_id", userID.String()).Msg("Failed to list user settings")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve settings",
		})
	}

	// Return empty array instead of null
	if userSettings == nil {
		userSettings = []settings.UserSetting{}
	}

	return c.JSON(userSettings)
}
