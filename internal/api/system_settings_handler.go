package api

import (
	"context"

	"github.com/fluxbase-eu/fluxbase/internal/auth"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
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
// GET /api/v1/admin/system/settings/*
func (h *SystemSettingsHandler) GetSetting(c *fiber.Ctx) error {
	ctx := context.Background()
	key := c.Params("*")

	if key == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Setting key is required",
		})
	}

	setting, err := h.settingsService.GetSetting(ctx, key)
	if err != nil {
		if err == auth.ErrSettingNotFound {
			// Return default value for known settings instead of 404
			if defaultSetting := h.getDefaultSetting(key); defaultSetting != nil {
				// Populate override information for defaults too
				if h.settingsCache != nil {
					defaultSetting.IsOverridden = h.settingsCache.IsOverriddenByEnv(key)
					if defaultSetting.IsOverridden {
						defaultSetting.OverrideSource = h.settingsCache.GetEnvVarName(key)
					}
				}
				return c.JSON(defaultSetting)
			}
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
// PUT /api/v1/admin/system/settings/*
func (h *SystemSettingsHandler) UpdateSetting(c *fiber.Ctx) error {
	ctx := context.Background()
	key := c.Params("*")

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

	// Validate setting key is in allowlist
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
// DELETE /api/v1/admin/system/settings/*
func (h *SystemSettingsHandler) DeleteSetting(c *fiber.Ctx) error {
	ctx := context.Background()
	key := c.Params("*")

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

// settingDefaults defines default values for known settings
var settingDefaults = map[string]map[string]interface{}{
	"app.auth.signup_enabled":               {"value": true},
	"app.auth.magic_link_enabled":           {"value": false},
	"app.auth.password_min_length":          {"value": 12},
	"app.auth.require_email_verification":   {"value": false},
	"app.realtime.enabled":                  {"value": true},
	"app.storage.enabled":                   {"value": true},
	"app.functions.enabled":                 {"value": true},
	"app.ai.enabled":                        {"value": true},
	"app.rpc.enabled":                       {"value": true},
	"app.jobs.enabled":                      {"value": true},
	"app.email.enabled":                     {"value": true},
	"app.email.provider":                    {"value": ""},
	"app.security.enable_global_rate_limit": {"value": false},
	// Email provider settings (for UI configuration)
	"app.email.from_address":     {"value": ""},
	"app.email.from_name":        {"value": ""},
	"app.email.smtp_host":        {"value": ""},
	"app.email.smtp_port":        {"value": 587},
	"app.email.smtp_username":    {"value": ""},
	"app.email.smtp_password":    {"value": ""}, // Encrypted in database
	"app.email.smtp_tls":         {"value": true},
	"app.email.sendgrid_api_key": {"value": ""}, // Encrypted in database
	"app.email.mailgun_api_key":  {"value": ""}, // Encrypted in database
	"app.email.mailgun_domain":   {"value": ""},
	"app.email.ses_access_key":   {"value": ""}, // Encrypted in database
	"app.email.ses_secret_key":   {"value": ""}, // Encrypted in database
	"app.email.ses_region":       {"value": "us-east-1"},
	// Captcha provider settings (for UI configuration)
	"app.security.captcha.enabled":         {"value": false},
	"app.security.captcha.provider":        {"value": "hcaptcha"},
	"app.security.captcha.site_key":        {"value": ""},
	"app.security.captcha.secret_key":      {"value": ""}, // Encrypted in database
	"app.security.captcha.score_threshold": {"value": 0.5},
	"app.security.captcha.endpoints":       {"value": []string{"signup", "login", "password_reset", "magic_link"}},
	"app.security.captcha.cap_server_url":  {"value": ""},
	"app.security.captcha.cap_api_key":     {"value": ""}, // Encrypted in database
}

// isValidSettingKey checks if a setting key is in the allowlist
func (h *SystemSettingsHandler) isValidSettingKey(key string) bool {
	_, exists := settingDefaults[key]
	return exists
}

// getDefaultSetting returns a default setting for a known key
func (h *SystemSettingsHandler) getDefaultSetting(key string) *auth.SystemSetting {
	defaultValue, exists := settingDefaults[key]
	if !exists {
		return nil
	}
	return &auth.SystemSetting{
		Key:   key,
		Value: defaultValue,
	}
}
