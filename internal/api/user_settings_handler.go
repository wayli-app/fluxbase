package api

import (
	"errors"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/database"
	"github.com/nimbleflux/fluxbase/internal/middleware"
	"github.com/nimbleflux/fluxbase/internal/settings"
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
func (h *UserSettingsHandler) CreateSecret(c fiber.Ctx) error {
	ctx := c.RequestCtx()

	// Get user ID from context
	userIDStr := middleware.GetUserID(c)
	if userIDStr == "" {
		return SendUnauthorized(c, "Authentication required", ErrCodeAuthRequired)
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		log.Error().Err(err).Msg("Invalid user ID in context")
		return SendInternalError(c, "Invalid user ID")
	}

	var req settings.CreateSecretSettingRequest
	if err := ParseBody(c, &req); err != nil {
		return err
	}

	// Validate required fields
	if req.Key == "" {
		return SendMissingField(c, "key")
	}

	if req.Value == "" {
		return SendMissingField(c, "value")
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
			return SendConflict(c, "A secret with this key already exists", ErrCodeDuplicateKey)
		}
		if errors.Is(err, settings.ErrCustomSettingInvalidKey) {
			return SendBadRequest(c, "Invalid setting key format", ErrCodeInvalidInput)
		}
		log.Error().Err(err).Str("key", req.Key).Str("user_id", userID.String()).Msg("Failed to create user secret")
		return SendInternalError(c, "Failed to create secret")
	}

	log.Info().
		Str("key", req.Key).
		Str("user_id", userID.String()).
		Msg("User secret created")

	return c.Status(fiber.StatusCreated).JSON(metadata)
}

// GetSecret returns metadata for a user's secret setting (never returns the value)
// GET /api/v1/settings/secret/*
func (h *UserSettingsHandler) GetSecret(c fiber.Ctx) error {
	ctx := c.RequestCtx()
	key := c.Params("*")

	if key == "" {
		return SendMissingField(c, "key")
	}

	// Get user ID from context
	userIDStr := middleware.GetUserID(c)
	if userIDStr == "" {
		return SendUnauthorized(c, "Authentication required", ErrCodeAuthRequired)
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		log.Error().Err(err).Msg("Invalid user ID in context")
		return SendInternalError(c, "Invalid user ID")
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
			return SendNotFound(c, "Secret not found")
		}
		log.Error().Err(err).Str("key", key).Str("user_id", userID.String()).Msg("Failed to get user secret")
		return SendInternalError(c, "Failed to retrieve secret")
	}

	return c.JSON(metadata)
}

// UpdateSecret updates a user's secret setting
// PUT /api/v1/settings/secret/*
func (h *UserSettingsHandler) UpdateSecret(c fiber.Ctx) error {
	ctx := c.RequestCtx()
	key := c.Params("*")

	if key == "" {
		return SendMissingField(c, "key")
	}

	// Get user ID from context
	userIDStr := middleware.GetUserID(c)
	if userIDStr == "" {
		return SendUnauthorized(c, "Authentication required", ErrCodeAuthRequired)
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		log.Error().Err(err).Msg("Invalid user ID in context")
		return SendInternalError(c, "Invalid user ID")
	}

	var req settings.UpdateSecretSettingRequest
	if err := ParseBody(c, &req); err != nil {
		return err
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
			return SendNotFound(c, "Secret not found")
		}
		log.Error().Err(err).Str("key", key).Str("user_id", userID.String()).Msg("Failed to update user secret")
		return SendInternalError(c, "Failed to update secret")
	}

	log.Info().
		Str("key", key).
		Str("user_id", userID.String()).
		Msg("User secret updated")

	return c.JSON(metadata)
}

// DeleteSecret deletes a user's secret setting
// DELETE /api/v1/settings/secret/*
func (h *UserSettingsHandler) DeleteSecret(c fiber.Ctx) error {
	ctx := c.RequestCtx()
	key := c.Params("*")

	if key == "" {
		return SendMissingField(c, "key")
	}

	// Get user ID from context
	userIDStr := middleware.GetUserID(c)
	if userIDStr == "" {
		return SendUnauthorized(c, "Authentication required", ErrCodeAuthRequired)
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		log.Error().Err(err).Msg("Invalid user ID in context")
		return SendInternalError(c, "Invalid user ID")
	}

	// Delete user's secret with RLS context
	err = middleware.WrapWithRLS(ctx, h.db, c, func(tx pgx.Tx) error {
		return h.settingsService.DeleteSecretSettingWithTx(ctx, tx, key, &userID)
	})
	if err != nil {
		if errors.Is(err, settings.ErrCustomSettingNotFound) {
			return SendNotFound(c, "Secret not found")
		}
		log.Error().Err(err).Str("key", key).Str("user_id", userID.String()).Msg("Failed to delete user secret")
		return SendInternalError(c, "Failed to delete secret")
	}

	log.Info().
		Str("key", key).
		Str("user_id", userID.String()).
		Msg("User secret deleted")

	return c.SendStatus(fiber.StatusNoContent)
}

// ListSecrets returns metadata for all user's secret settings
// GET /api/v1/settings/secrets
func (h *UserSettingsHandler) ListSecrets(c fiber.Ctx) error {
	ctx := c.RequestCtx()

	// Get user ID from context
	userIDStr := middleware.GetUserID(c)
	if userIDStr == "" {
		return SendUnauthorized(c, "Authentication required", ErrCodeAuthRequired)
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		log.Error().Err(err).Msg("Invalid user ID in context")
		return SendInternalError(c, "Invalid user ID")
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
		return SendInternalError(c, "Failed to retrieve secrets")
	}

	return c.JSON(secrets)
}

// GetUserSecretValue retrieves the decrypted value of a specific user's secret
// This is a privileged operation that requires service_role
// GET /api/v1/admin/settings/user/:user_id/secret/:key/decrypt
func (h *UserSettingsHandler) GetUserSecretValue(c fiber.Ctx) error {
	ctx := c.RequestCtx()

	// Require service_role for this privileged operation
	role := c.Locals("user_role")
	if role != "service_role" && role != "tenant_service" {
		return SendForbidden(c, "This operation requires service_role", ErrCodeAdminRequired)
	}

	// Check if secrets service is configured
	if h.secretsService == nil {
		log.Error().Msg("SecretsService not configured for UserSettingsHandler")
		return SendInternalError(c, "Secrets service not configured")
	}

	// Parse target user ID from URL
	targetUserIDStr := c.Params("user_id")
	targetUserID, err := uuid.Parse(targetUserIDStr)
	if err != nil {
		return SendInvalidID(c, "user_id")
	}

	// Get secret key from URL
	key := c.Params("key")
	if key == "" {
		return SendMissingField(c, "key")
	}

	// Retrieve and decrypt the secret (service_role bypasses RLS)
	value, err := h.secretsService.GetUserSecret(ctx, targetUserID, key)
	if err != nil {
		if errors.Is(err, settings.ErrSecretNotFound) {
			return SendNotFound(c, "Secret not found")
		}
		if errors.Is(err, settings.ErrDecryptionFailed) {
			log.Error().Err(err).Str("key", key).Str("user_id", targetUserID.String()).Msg("Failed to decrypt user secret")
			return SendInternalError(c, "Failed to decrypt secret")
		}
		log.Error().Err(err).Str("key", key).Str("user_id", targetUserID.String()).Msg("Failed to retrieve user secret")
		return SendInternalError(c, "Failed to retrieve secret")
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
func (h *UserSettingsHandler) GetSetting(c fiber.Ctx) error {
	ctx := c.RequestCtx()
	key := c.Params("key")

	if key == "" {
		return SendMissingField(c, "key")
	}

	// Get user ID from context
	userIDStr := middleware.GetUserID(c)
	if userIDStr == "" {
		return SendUnauthorized(c, "Authentication required", ErrCodeAuthRequired)
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		log.Error().Err(err).Msg("Invalid user ID in context")
		return SendInternalError(c, "Invalid user ID")
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
			return SendNotFound(c, "Setting not found")
		}
		log.Error().Err(err).Str("key", key).Str("user_id", userID.String()).Msg("Failed to get setting with fallback")
		return SendInternalError(c, "Failed to retrieve setting")
	}

	return c.JSON(result)
}

// GetUserOwnSetting retrieves only the user's own setting (no fallback)
// GET /api/v1/settings/user/own/:key
func (h *UserSettingsHandler) GetUserOwnSetting(c fiber.Ctx) error {
	ctx := c.RequestCtx()
	key := c.Params("key")

	if key == "" {
		return SendMissingField(c, "key")
	}

	// Get user ID from context
	userIDStr := middleware.GetUserID(c)
	if userIDStr == "" {
		return SendUnauthorized(c, "Authentication required", ErrCodeAuthRequired)
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		log.Error().Err(err).Msg("Invalid user ID in context")
		return SendInternalError(c, "Invalid user ID")
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
			return SendNotFound(c, "Setting not found")
		}
		log.Error().Err(err).Str("key", key).Str("user_id", userID.String()).Msg("Failed to get user setting")
		return SendInternalError(c, "Failed to retrieve setting")
	}

	return c.JSON(setting)
}

// GetSystemSettingPublic retrieves a system-level setting (user_id IS NULL)
// GET /api/v1/settings/user/system/:key
func (h *UserSettingsHandler) GetSystemSettingPublic(c fiber.Ctx) error {
	ctx := c.RequestCtx()
	key := c.Params("key")

	if key == "" {
		return SendMissingField(c, "key")
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
			return SendNotFound(c, "Setting not found")
		}
		log.Error().Err(err).Str("key", key).Msg("Failed to get system setting")
		return SendInternalError(c, "Failed to retrieve setting")
	}

	// Return only the value (not all metadata) for public access
	return c.JSON(fiber.Map{
		"key":   setting.Key,
		"value": setting.Value,
	})
}

// SetSetting creates or updates a user setting
// PUT /api/v1/settings/user/:key
func (h *UserSettingsHandler) SetSetting(c fiber.Ctx) error {
	ctx := c.RequestCtx()
	key := c.Params("key")

	if key == "" {
		return SendMissingField(c, "key")
	}

	// Get user ID from context
	userIDStr := middleware.GetUserID(c)
	if userIDStr == "" {
		return SendUnauthorized(c, "Authentication required", ErrCodeAuthRequired)
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		log.Error().Err(err).Msg("Invalid user ID in context")
		return SendInternalError(c, "Invalid user ID")
	}

	var req settings.CreateUserSettingRequest
	if err := ParseBody(c, &req); err != nil {
		return err
	}

	// Use key from URL
	req.Key = key

	// Validate required fields
	if req.Value == nil {
		return SendMissingField(c, "value")
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
			return SendBadRequest(c, "Invalid setting key format", ErrCodeInvalidInput)
		}
		log.Error().Err(err).Str("key", key).Str("user_id", userID.String()).Msg("Failed to set user setting")
		return SendInternalError(c, "Failed to save setting")
	}

	log.Info().
		Str("key", key).
		Str("user_id", userID.String()).
		Msg("User setting saved")

	return c.JSON(setting)
}

// DeleteSetting removes a user's setting
// DELETE /api/v1/settings/user/:key
func (h *UserSettingsHandler) DeleteSetting(c fiber.Ctx) error {
	ctx := c.RequestCtx()
	key := c.Params("key")

	if key == "" {
		return SendMissingField(c, "key")
	}

	// Get user ID from context
	userIDStr := middleware.GetUserID(c)
	if userIDStr == "" {
		return SendUnauthorized(c, "Authentication required", ErrCodeAuthRequired)
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		log.Error().Err(err).Msg("Invalid user ID in context")
		return SendInternalError(c, "Invalid user ID")
	}

	// Delete user's setting with RLS context
	err = middleware.WrapWithRLS(ctx, h.db, c, func(tx pgx.Tx) error {
		return h.settingsService.DeleteUserSettingWithTx(ctx, tx, userID, key)
	})
	if err != nil {
		if errors.Is(err, settings.ErrCustomSettingNotFound) {
			return SendNotFound(c, "Setting not found")
		}
		log.Error().Err(err).Str("key", key).Str("user_id", userID.String()).Msg("Failed to delete user setting")
		return SendInternalError(c, "Failed to delete setting")
	}

	log.Info().
		Str("key", key).
		Str("user_id", userID.String()).
		Msg("User setting deleted")

	return c.SendStatus(fiber.StatusNoContent)
}

// ListSettings returns all user's own settings
// GET /api/v1/settings/user/list
func (h *UserSettingsHandler) ListSettings(c fiber.Ctx) error {
	ctx := c.RequestCtx()

	// Get user ID from context
	userIDStr := middleware.GetUserID(c)
	if userIDStr == "" {
		return SendUnauthorized(c, "Authentication required", ErrCodeAuthRequired)
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		log.Error().Err(err).Msg("Invalid user ID in context")
		return SendInternalError(c, "Invalid user ID")
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
		return SendInternalError(c, "Failed to retrieve settings")
	}

	// Return empty array instead of null
	if userSettings == nil {
		userSettings = []settings.UserSetting{}
	}

	return c.JSON(userSettings)
}

// fiber:context-methods migrated
