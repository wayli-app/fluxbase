package api

import (
	"errors"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/middleware"
	"github.com/nimbleflux/fluxbase/internal/settings"
)

type CustomSettingsHandler struct {
	settingsService *settings.CustomSettingsService
}

func NewCustomSettingsHandler(settingsService *settings.CustomSettingsService) *CustomSettingsHandler {
	return &CustomSettingsHandler{
		settingsService: settingsService,
	}
}

func (h *CustomSettingsHandler) requireService(c fiber.Ctx) error {
	if h.settingsService == nil {
		return fiber.NewError(fiber.StatusServiceUnavailable, "Settings service not initialized")
	}
	return nil
}

func (h *CustomSettingsHandler) CreateSetting(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)

	userIDStr := middleware.GetUserID(c)
	userRole := c.Locals("user_role")

	if userIDStr == "" || userRole == nil {
		return SendUnauthorized(c, "Authentication required", ErrCodeAuthRequired)
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		log.Error().Err(err).Msg("Invalid user ID in context")
		return SendInternalError(c, "Invalid user ID")
	}

	var req settings.CreateCustomSettingRequest
	if err := ParseBody(c, &req); err != nil {
		return err
	}

	if req.Key == "" {
		return SendMissingField(c, "Setting key")
	}

	if req.Value == nil {
		return SendMissingField(c, "Setting value")
	}

	if req.IsSecret {
		valueStr := extractStringValueFromMap(req.Value)
		if valueStr == "" {
			return SendBadRequest(c, "Secret value must be a non-empty string (use {\"value\": \"your-secret\"})", ErrCodeInvalidInput)
		}

		if err := h.requireService(c); err != nil {
			return err
		}

		secretReq := settings.CreateSecretSettingRequest{
			Key:         req.Key,
			Value:       valueStr,
			Description: req.Description,
		}

		metadata, err := h.settingsService.CreateSecretSetting(ctx, secretReq, nil, userID)
		if err != nil {
			if errors.Is(err, settings.ErrCustomSettingDuplicate) {
				return SendConflict(c, "A setting with this key already exists", ErrCodeDuplicateKey)
			}
			if errors.Is(err, settings.ErrCustomSettingInvalidKey) {
				return SendBadRequest(c, "Invalid setting key format", ErrCodeInvalidFormat)
			}
			log.Error().Err(err).Str("key", req.Key).Msg("Failed to create secret setting")
			return SendInternalError(c, "Failed to create setting")
		}

		log.Info().
			Str("key", req.Key).
			Str("user_id", userID.String()).
			Str("user_role", userRole.(string)).
			Bool("is_secret", true).
			Msg("Secret setting created")

		return c.Status(fiber.StatusCreated).JSON(metadata)
	}

	if err := h.requireService(c); err != nil {
		return err
	}

	setting, err := h.settingsService.CreateSetting(ctx, req, userID)
	if err != nil {
		if errors.Is(err, settings.ErrCustomSettingDuplicate) {
			return SendConflict(c, "A setting with this key already exists", ErrCodeDuplicateKey)
		}
		if errors.Is(err, settings.ErrCustomSettingInvalidKey) {
			return SendBadRequest(c, "Invalid setting key format", ErrCodeInvalidFormat)
		}
		log.Error().Err(err).Str("key", req.Key).Msg("Failed to create custom setting")
		return SendInternalError(c, "Failed to create setting")
	}

	log.Info().
		Str("key", req.Key).
		Str("user_id", userID.String()).
		Str("user_role", userRole.(string)).
		Msg("Custom setting created")

	return c.Status(fiber.StatusCreated).JSON(setting)
}

func (h *CustomSettingsHandler) ListSettings(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)

	userRole := c.Locals("user_role")
	if userRole == nil {
		return SendUnauthorized(c, "Authentication required", ErrCodeAuthRequired)
	}

	if err := h.requireService(c); err != nil {
		return err
	}

	settings, err := h.settingsService.ListSettings(ctx, userRole.(string))
	if err != nil {
		log.Error().Err(err).Msg("Failed to list custom settings")
		return SendInternalError(c, "Failed to retrieve custom settings")
	}

	return c.JSON(settings)
}

func (h *CustomSettingsHandler) GetSetting(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)
	key := c.Params("*")

	if key == "" {
		return SendMissingField(c, "Setting key")
	}

	if err := h.requireService(c); err != nil {
		return err
	}

	setting, err := h.settingsService.GetSetting(ctx, key)
	if err != nil {
		if errors.Is(err, settings.ErrCustomSettingNotFound) {
			return SendNotFound(c, "Setting not found")
		}
		log.Error().Err(err).Str("key", key).Msg("Failed to get custom setting")
		return SendInternalError(c, "Failed to retrieve setting")
	}

	return c.JSON(setting)
}

func (h *CustomSettingsHandler) UpdateSetting(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)
	key := c.Params("*")

	if key == "" {
		return SendMissingField(c, "Setting key")
	}

	userIDStr := middleware.GetUserID(c)
	userRole := c.Locals("user_role")

	if userIDStr == "" || userRole == nil {
		return SendUnauthorized(c, "Authentication required", ErrCodeAuthRequired)
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		log.Error().Err(err).Msg("Invalid user ID in context")
		return SendInternalError(c, "Invalid user ID")
	}

	var req settings.UpdateCustomSettingRequest
	if err := ParseBody(c, &req); err != nil {
		return err
	}

	if req.Value == nil {
		return SendMissingField(c, "Setting value")
	}

	if err := h.requireService(c); err != nil {
		return err
	}

	setting, err := h.settingsService.UpdateSetting(ctx, key, req, userID, userRole.(string))
	if err != nil {
		if errors.Is(err, settings.ErrCustomSettingNotFound) {
			return SendNotFound(c, "Setting not found")
		}
		if errors.Is(err, settings.ErrCustomSettingPermissionDenied) {
			return SendForbidden(c, "You do not have permission to edit this setting", ErrCodeAccessDenied)
		}
		log.Error().Err(err).Str("key", key).Msg("Failed to update custom setting")
		return SendInternalError(c, "Failed to update setting")
	}

	log.Info().
		Str("key", key).
		Str("user_id", userID.String()).
		Str("user_role", userRole.(string)).
		Msg("Custom setting updated")

	return c.JSON(setting)
}

func (h *CustomSettingsHandler) DeleteSetting(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)
	key := c.Params("*")

	if key == "" {
		return SendMissingField(c, "Setting key")
	}

	userRole := c.Locals("user_role")
	if userRole == nil {
		return SendUnauthorized(c, "Authentication required", ErrCodeAuthRequired)
	}

	if err := h.requireService(c); err != nil {
		return err
	}

	err := h.settingsService.DeleteSetting(ctx, key, userRole.(string))
	if err != nil {
		if errors.Is(err, settings.ErrCustomSettingNotFound) {
			return SendNotFound(c, "Setting not found")
		}
		if errors.Is(err, settings.ErrCustomSettingPermissionDenied) {
			return SendForbidden(c, "You do not have permission to delete this setting", ErrCodeAccessDenied)
		}
		log.Error().Err(err).Str("key", key).Msg("Failed to delete custom setting")
		return SendInternalError(c, "Failed to delete setting")
	}

	log.Info().
		Str("key", key).
		Str("user_role", userRole.(string)).
		Msg("Custom setting deleted")

	return c.SendStatus(fiber.StatusNoContent)
}

func (h *CustomSettingsHandler) CreateSecretSetting(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)

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

	if req.Key == "" {
		return SendMissingField(c, "Setting key")
	}

	if req.Value == "" {
		return SendMissingField(c, "Setting value")
	}

	if err := h.requireService(c); err != nil {
		return err
	}

	metadata, err := h.settingsService.CreateSecretSetting(ctx, req, nil, userID)
	if err != nil {
		if errors.Is(err, settings.ErrCustomSettingDuplicate) {
			return SendConflict(c, "A secret setting with this key already exists", ErrCodeDuplicateKey)
		}
		if errors.Is(err, settings.ErrCustomSettingInvalidKey) {
			return SendBadRequest(c, "Invalid setting key format", ErrCodeInvalidFormat)
		}
		log.Error().Err(err).Str("key", req.Key).Msg("Failed to create secret setting")
		return SendInternalError(c, "Failed to create secret setting")
	}

	log.Info().
		Str("key", req.Key).
		Str("user_id", userID.String()).
		Msg("System secret setting created")

	return c.Status(fiber.StatusCreated).JSON(metadata)
}

func (h *CustomSettingsHandler) GetSecretSetting(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)
	key := c.Params("*")

	if key == "" {
		return SendMissingField(c, "Setting key")
	}

	if err := h.requireService(c); err != nil {
		return err
	}

	metadata, err := h.settingsService.GetSecretSettingMetadata(ctx, key, nil)
	if err != nil {
		if errors.Is(err, settings.ErrCustomSettingNotFound) {
			return SendNotFound(c, "Secret setting not found")
		}
		log.Error().Err(err).Str("key", key).Msg("Failed to get secret setting")
		return SendInternalError(c, "Failed to retrieve secret setting")
	}

	return c.JSON(metadata)
}

func (h *CustomSettingsHandler) UpdateSecretSetting(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)
	key := c.Params("*")

	if key == "" {
		return SendMissingField(c, "Setting key")
	}

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

	if err := h.requireService(c); err != nil {
		return err
	}

	metadata, err := h.settingsService.UpdateSecretSetting(ctx, key, req, nil, userID)
	if err != nil {
		if errors.Is(err, settings.ErrCustomSettingNotFound) {
			return SendNotFound(c, "Secret setting not found")
		}
		log.Error().Err(err).Str("key", key).Msg("Failed to update secret setting")
		return SendInternalError(c, "Failed to update secret setting")
	}

	log.Info().
		Str("key", key).
		Str("user_id", userID.String()).
		Msg("System secret setting updated")

	return c.JSON(metadata)
}

func (h *CustomSettingsHandler) DeleteSecretSetting(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)
	key := c.Params("*")

	if key == "" {
		return SendMissingField(c, "Setting key")
	}

	if err := h.requireService(c); err != nil {
		return err
	}

	err := h.settingsService.DeleteSecretSetting(ctx, key, nil)
	if err != nil {
		if errors.Is(err, settings.ErrCustomSettingNotFound) {
			return SendNotFound(c, "Secret setting not found")
		}
		log.Error().Err(err).Str("key", key).Msg("Failed to delete secret setting")
		return SendInternalError(c, "Failed to delete secret setting")
	}

	log.Info().
		Str("key", key).
		Msg("System secret setting deleted")

	return c.SendStatus(fiber.StatusNoContent)
}

func (h *CustomSettingsHandler) ListSecretSettings(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)

	if err := h.requireService(c); err != nil {
		return err
	}

	secrets, err := h.settingsService.ListSecretSettings(ctx, nil)
	if err != nil {
		log.Error().Err(err).Msg("Failed to list secret settings")
		return SendInternalError(c, "Failed to retrieve secret settings")
	}

	return c.JSON(secrets)
}

func extractStringValueFromMap(m map[string]interface{}) string {
	if v, ok := m["value"]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}

	if len(m) == 1 {
		for _, v := range m {
			if s, ok := v.(string); ok {
				return s
			}
		}
	}

	return ""
}
