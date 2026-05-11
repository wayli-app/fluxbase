package api

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/auth"
)

type SystemSettingsHandler struct {
	settingsService *auth.SystemSettingsService
	settingsCache   *auth.SettingsCache
}

func NewSystemSettingsHandler(settingsService *auth.SystemSettingsService, settingsCache *auth.SettingsCache) *SystemSettingsHandler {
	return &SystemSettingsHandler{
		settingsService: settingsService,
		settingsCache:   settingsCache,
	}
}

func (h *SystemSettingsHandler) requireService(c fiber.Ctx) error {
	if h.settingsService == nil {
		return SendNotInitialized(c, "Settings service")
	}
	return nil
}

func (h *SystemSettingsHandler) ListSettings(c fiber.Ctx) error {
	ctx := context.Background()

	if err := h.requireService(c); err != nil {
		return err
	}

	settings, err := h.settingsService.ListSettings(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get system settings")
		return SendInternalError(c, "Failed to retrieve system settings")
	}

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

func (h *SystemSettingsHandler) GetSetting(c fiber.Ctx) error {
	ctx := context.Background()
	key := c.Params("*")

	if key == "" {
		return SendMissingField(c, "Setting key")
	}

	if err := h.requireService(c); err != nil {
		return err
	}

	setting, err := h.settingsService.GetSetting(ctx, key)
	if err != nil {
		if errors.Is(err, auth.ErrSettingNotFound) {
			if defaultSetting := h.getDefaultSetting(key); defaultSetting != nil {
				if h.settingsCache != nil {
					defaultSetting.IsOverridden = h.settingsCache.IsOverriddenByEnv(key)
					if defaultSetting.IsOverridden {
						defaultSetting.OverrideSource = h.settingsCache.GetEnvVarName(key)
					}
				}
				return c.JSON(defaultSetting)
			}
			return SendNotFound(c, "Setting not found")
		}
		log.Error().Err(err).Str("key", key).Msg("Failed to get setting")
		return SendInternalError(c, "Failed to retrieve setting")
	}

	if h.settingsCache != nil {
		setting.IsOverridden = h.settingsCache.IsOverriddenByEnv(key)
		if setting.IsOverridden {
			setting.OverrideSource = h.settingsCache.GetEnvVarName(key)
		}
	}

	return c.JSON(setting)
}

func (h *SystemSettingsHandler) UpdateSetting(c fiber.Ctx) error {
	ctx := context.Background()
	key := c.Params("*")

	if key == "" {
		return SendMissingField(c, "Setting key")
	}

	var req struct {
		Value       map[string]interface{} `json:"value"`
		Description string                 `json:"description"`
	}

	if err := ParseBody(c, &req); err != nil {
		return err
	}

	if !h.isValidSettingKey(key) {
		return SendBadRequest(c, "Invalid setting key", ErrCodeInvalidInput)
	}

	if h.settingsCache != nil && h.settingsCache.IsOverriddenByEnv(key) {
		return SendConflict(c, "This setting cannot be updated because it is overridden by an environment variable", ErrCodeConflict)
	}

	if err := h.requireService(c); err != nil {
		return err
	}

	if err := h.settingsService.SetSetting(ctx, key, req.Value, req.Description); err != nil {
		log.Error().Err(err).Str("key", key).Msg("Failed to update setting")
		return SendInternalError(c, "Failed to update setting")
	}

	log.Info().Str("key", key).Interface("value", req.Value).Msg("System setting updated")

	setting, err := h.settingsService.GetSetting(ctx, key)
	if err != nil {
		return c.JSON(fiber.Map{
			"key":         key,
			"value":       req.Value,
			"description": req.Description,
		})
	}

	return c.JSON(setting)
}

func (h *SystemSettingsHandler) DeleteSetting(c fiber.Ctx) error {
	ctx := context.Background()
	key := c.Params("*")

	if key == "" {
		return SendMissingField(c, "Setting key")
	}

	if err := h.requireService(c); err != nil {
		return err
	}

	if err := h.settingsService.DeleteSetting(ctx, key); err != nil {
		if errors.Is(err, auth.ErrSettingNotFound) {
			return SendNotFound(c, "Setting not found")
		}
		log.Error().Err(err).Str("key", key).Msg("Failed to delete setting")
		return SendInternalError(c, "Failed to delete setting")
	}

	log.Info().Str("key", key).Msg("System setting deleted")

	return c.SendStatus(fiber.StatusNoContent)
}

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
	"app.security.enable_global_rate_limit": {"value": true},
	"app.email.from_address":                {"value": ""},
	"app.email.from_name":                   {"value": ""},
	"app.email.smtp_host":                   {"value": ""},
	"app.email.smtp_port":                   {"value": 587},
	"app.email.smtp_username":               {"value": ""},
	"app.email.smtp_password":               {"value": ""},
	"app.email.smtp_tls":                    {"value": true},
	"app.email.sendgrid_api_key":            {"value": ""},
	"app.email.mailgun_api_key":             {"value": ""},
	"app.email.mailgun_domain":              {"value": ""},
	"app.email.ses_access_key":              {"value": ""},
	"app.email.ses_secret_key":              {"value": ""},
	"app.email.ses_region":                  {"value": "us-east-1"},
	"app.security.captcha.enabled":          {"value": false},
	"app.security.captcha.provider":         {"value": "hcaptcha"},
	"app.security.captcha.site_key":         {"value": ""},
	"app.security.captcha.secret_key":       {"value": ""},
	"app.security.captcha.score_threshold":  {"value": 0.5},
	"app.security.captcha.endpoints":        {"value": []string{"signup", "login", "password_reset", "magic_link"}},
	"app.security.captcha.cap_server_url":   {"value": ""},
	"app.security.captcha.cap_api_key":      {"value": ""},
}

func (h *SystemSettingsHandler) isValidSettingKey(key string) bool {
	_, exists := settingDefaults[key]
	return exists
}

func (h *SystemSettingsHandler) getDefaultSetting(key string) *auth.SystemSetting {
	defaultValue, exists := settingDefaults[key]
	if !exists {
		return nil
	}

	if h.settingsCache != nil {
		ctx := context.Background()

		if val, ok := defaultValue["value"].(bool); ok {
			actualValue := h.settingsCache.GetBool(ctx, key, val)
			return &auth.SystemSetting{
				Key:   key,
				Value: map[string]interface{}{"value": actualValue},
			}
		}

		if val, ok := defaultValue["value"].(string); ok {
			actualValue := h.settingsCache.GetString(ctx, key, val)
			return &auth.SystemSetting{
				Key:   key,
				Value: map[string]interface{}{"value": actualValue},
			}
		}

		if val, ok := defaultValue["value"].(int); ok {
			actualValue := h.settingsCache.GetInt(ctx, key, val)
			return &auth.SystemSetting{
				Key:   key,
				Value: map[string]interface{}{"value": actualValue},
			}
		}

		if val, ok := defaultValue["value"].(float64); ok {
			strVal := fmt.Sprintf("%v", val)
			actualStrValue := h.settingsCache.GetString(ctx, key, strVal)
			if actualStrValue != "" {
				if actualVal, err := strconv.ParseFloat(actualStrValue, 64); err == nil {
					return &auth.SystemSetting{
						Key:   key,
						Value: map[string]interface{}{"value": actualVal},
					}
				}
			}
			return &auth.SystemSetting{
				Key:   key,
				Value: defaultValue,
			}
		}
	}

	return &auth.SystemSetting{
		Key:   key,
		Value: defaultValue,
	}
}
