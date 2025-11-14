package api

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
	"github.com/wayli-app/fluxbase/internal/auth"
)

// SystemSettingsHandler handles system settings operations
type SystemSettingsHandler struct {
	settingsService *auth.SystemSettingsService
	settingsCache   *auth.SettingsCache
}

// NewSystemSettingsHandler creates a new system settings handler
func NewSystemSettingsHandler(settingsService *auth.SystemSettingsService, settingsCache *auth.SettingsCache) *SystemSettingsHandler {
	return &SystemSettingsHandler{
		settingsService: settingsService,
		settingsCache:   settingsCache,
	}
}

// ListSettings returns all system settings
// GET /api/v1/admin/system/settings
func (h *SystemSettingsHandler) ListSettings(c *fiber.Ctx) error {
	ctx := context.Background()

	settings, err := h.settingsService.ListSettings(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get system settings")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve system settings",
		})
	}

	// Populate override information for each setting
	if h.settingsCache != nil {
		for i := range settings {
			settings[i].IsOverridden = h.settingsCache.IsOverriddenByEnv(settings[i].Key)
			if settings[i].IsOverridden {
				settings[i].OverrideSource = h.settingsCache.GetEnvVarName(settings[i].Key)
			}
		}
	}

	return c.JSON(settings)
}

// GetSetting returns a specific setting by key
// GET /api/v1/admin/system/settings/:key
func (h *SystemSettingsHandler) GetSetting(c *fiber.Ctx) error {
	ctx := context.Background()
	key := c.Params("key")

	if key == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Setting key is required",
		})
	}

	setting, err := h.settingsService.GetSetting(ctx, key)
	if err != nil {
		if err == auth.ErrSettingNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Setting not found",
			})
		}
		log.Error().Err(err).Str("key", key).Msg("Failed to get setting")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve setting",
		})
	}

	// Populate override information
	if h.settingsCache != nil {
		setting.IsOverridden = h.settingsCache.IsOverriddenByEnv(key)
		if setting.IsOverridden {
			setting.OverrideSource = h.settingsCache.GetEnvVarName(key)
		}
	}

	return c.JSON(setting)
}

// UpdateSetting updates a specific setting
// PUT /api/v1/admin/system/settings/:key
func (h *SystemSettingsHandler) UpdateSetting(c *fiber.Ctx) error {
	ctx := context.Background()
	key := c.Params("key")

	if key == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Setting key is required",
		})
	}

	var req struct {
		Value       map[string]interface{} `json:"value"`
		Description string                 `json:"description"`
	}

	if err := c.BodyParser(&req); err != nil {
		log.Error().Err(err).Msg("Failed to parse request body")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate setting key is in whitelist
	if !h.isValidSettingKey(key) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid setting key",
			"code":  "INVALID_SETTING_KEY",
		})
	}

	// Check if setting is overridden by environment variable
	if h.settingsCache != nil && h.settingsCache.IsOverriddenByEnv(key) {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": "This setting cannot be updated because it is overridden by an environment variable",
			"code":  "ENV_OVERRIDE",
			"key":   key,
		})
	}

	if err := h.settingsService.SetSetting(ctx, key, req.Value, req.Description); err != nil {
		log.Error().Err(err).Str("key", key).Msg("Failed to update setting")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update setting",
		})
	}

	log.Info().Str("key", key).Interface("value", req.Value).Msg("System setting updated")

	// Return the updated setting
	setting, err := h.settingsService.GetSetting(ctx, key)
	if err != nil {
		// Setting was created but we can't retrieve it - return success anyway
		return c.JSON(fiber.Map{
			"key":         key,
			"value":       req.Value,
			"description": req.Description,
		})
	}

	return c.JSON(setting)
}

// DeleteSetting deletes a specific setting
// DELETE /api/v1/admin/system/settings/:key
func (h *SystemSettingsHandler) DeleteSetting(c *fiber.Ctx) error {
	ctx := context.Background()
	key := c.Params("key")

	if key == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Setting key is required",
		})
	}

	if err := h.settingsService.DeleteSetting(ctx, key); err != nil {
		if err == auth.ErrSettingNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Setting not found",
			})
		}
		log.Error().Err(err).Str("key", key).Msg("Failed to delete setting")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete setting",
		})
	}

	log.Info().Str("key", key).Msg("System setting deleted")

	return c.SendStatus(fiber.StatusNoContent)
}

// isValidSettingKey checks if a setting key is in the whitelist
func (h *SystemSettingsHandler) isValidSettingKey(key string) bool {
	validKeys := map[string]bool{
		"app.auth.enable_signup":                true,
		"app.auth.enable_magic_link":            true,
		"app.auth.password_min_length":          true,
		"app.auth.require_email_verification":   true,
		"app.features.enable_realtime":          true,
		"app.features.enable_storage":           true,
		"app.features.enable_functions":         true,
		"app.email.enabled":                     true,
		"app.email.provider":                    true,
		"app.security.enable_global_rate_limit": true,
	}

	return validKeys[key]
}
