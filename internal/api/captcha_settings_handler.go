package api

import (
	"context"
	"fmt"

	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/auth"
	"github.com/nimbleflux/fluxbase/internal/config"
	"github.com/nimbleflux/fluxbase/internal/settings"
)

type CaptchaSettingsHandler struct {
	settingsService *auth.SystemSettingsService
	settingsCache   *settings.SettingsCache
	secretsService  *settings.SecretsService
	envConfig       *config.SecurityConfig
	captchaService  *auth.CaptchaService
}

func NewCaptchaSettingsHandler(
	settingsService *auth.SystemSettingsService,
	settingsCache *settings.SettingsCache,
	secretsService *settings.SecretsService,
	envConfig *config.SecurityConfig,
	captchaService *auth.CaptchaService,
) *CaptchaSettingsHandler {
	return &CaptchaSettingsHandler{
		settingsService: settingsService,
		settingsCache:   settingsCache,
		secretsService:  secretsService,
		envConfig:       envConfig,
		captchaService:  captchaService,
	}
}

type CaptchaSettingsResponse struct {
	Enabled        bool     `json:"enabled"`
	Provider       string   `json:"provider"`
	SiteKey        string   `json:"site_key"`
	SecretKeySet   bool     `json:"secret_key_set"`
	ScoreThreshold float64  `json:"score_threshold"`
	Endpoints      []string `json:"endpoints"`
	CapServerURL   string   `json:"cap_server_url"`
	CapAPIKeySet   bool     `json:"cap_api_key_set"`

	Overrides map[string]OverrideInfo `json:"_overrides"`
}

type UpdateCaptchaSettingsRequest struct {
	Enabled        *bool     `json:"enabled,omitempty"`
	Provider       *string   `json:"provider,omitempty"`
	SiteKey        *string   `json:"site_key,omitempty"`
	SecretKey      *string   `json:"secret_key,omitempty"`
	ScoreThreshold *float64  `json:"score_threshold,omitempty"`
	Endpoints      *[]string `json:"endpoints,omitempty"`
	CapServerURL   *string   `json:"cap_server_url,omitempty"`
	CapAPIKey      *string   `json:"cap_api_key,omitempty"`
}

var validProviders = map[string]bool{
	"hcaptcha":     true,
	"recaptcha_v3": true,
	"turnstile":    true,
	"cap":          true,
}

var validEndpoints = map[string]bool{
	"signup":         true,
	"login":          true,
	"password_reset": true,
	"magic_link":     true,
}

func (h *CaptchaSettingsHandler) GetSettings(c fiber.Ctx) error {
	ctx := context.Background()

	response := CaptchaSettingsResponse{
		Overrides: make(map[string]OverrideInfo),
	}

	getString := func(key, defaultVal string) (string, bool) {
		if h.settingsCache != nil && h.settingsCache.IsOverriddenByEnv(key) {
			return h.settingsCache.GetString(ctx, key, defaultVal), true
		}
		return h.settingsCache.GetString(ctx, key, defaultVal), false
	}

	getBool := func(key string, defaultVal bool) (bool, bool) {
		if h.settingsCache != nil && h.settingsCache.IsOverriddenByEnv(key) {
			return h.settingsCache.GetBool(ctx, key, defaultVal), true
		}
		return h.settingsCache.GetBool(ctx, key, defaultVal), false
	}

	getFloat64 := func(key string, defaultVal float64) (float64, bool) {
		var result float64
		if h.settingsCache != nil {
			if err := h.settingsCache.GetJSON(ctx, key, &result); err == nil {
				isOverridden := h.settingsCache.IsOverriddenByEnv(key)
				return result, isOverridden
			}
		}
		return defaultVal, false
	}

	getStringSlice := func(key string, defaultVal []string) ([]string, bool) {
		var result []string
		if h.settingsCache != nil {
			if err := h.settingsCache.GetJSON(ctx, key, &result); err == nil {
				isOverridden := h.settingsCache.IsOverriddenByEnv(key)
				return result, isOverridden
			}
		}
		return defaultVal, false
	}

	addOverride := func(field, key string) {
		if h.settingsCache != nil && h.settingsCache.IsOverriddenByEnv(key) {
			response.Overrides[field] = OverrideInfo{
				IsOverridden: true,
				EnvVar:       h.settingsCache.GetEnvVarName(key),
			}
		}
	}

	response.Enabled, _ = getBool("app.security.captcha.enabled", false)
	addOverride("enabled", "app.security.captcha.enabled")

	response.Provider, _ = getString("app.security.captcha.provider", "hcaptcha")
	addOverride("provider", "app.security.captcha.provider")

	response.SiteKey, _ = getString("app.security.captcha.site_key", "")
	addOverride("site_key", "app.security.captcha.site_key")

	secretKey, _ := getString("app.security.captcha.secret_key", "")
	response.SecretKeySet = secretKey != ""
	addOverride("secret_key", "app.security.captcha.secret_key")

	response.ScoreThreshold, _ = getFloat64("app.security.captcha.score_threshold", 0.5)
	addOverride("score_threshold", "app.security.captcha.score_threshold")

	response.Endpoints, _ = getStringSlice("app.security.captcha.endpoints", []string{"signup", "login", "password_reset", "magic_link"})
	addOverride("endpoints", "app.security.captcha.endpoints")

	response.CapServerURL, _ = getString("app.security.captcha.cap_server_url", "")
	addOverride("cap_server_url", "app.security.captcha.cap_server_url")

	capAPIKey, _ := getString("app.security.captcha.cap_api_key", "")
	response.CapAPIKeySet = capAPIKey != ""
	addOverride("cap_api_key", "app.security.captcha.cap_api_key")

	return c.JSON(response)
}

func (h *CaptchaSettingsHandler) UpdateSettings(c fiber.Ctx) error {
	ctx := context.Background()

	var req UpdateCaptchaSettingsRequest
	if err := ParseBody(c, &req); err != nil {
		return err
	}

	if req.Provider != nil {
		if !validProviders[*req.Provider] {
			return SendBadRequest(c, "Invalid provider. Must be one of: hcaptcha, recaptcha_v3, turnstile, cap", ErrCodeInvalidInput)
		}
	}

	if req.Endpoints != nil {
		for _, endpoint := range *req.Endpoints {
			if !validEndpoints[endpoint] {
				return SendBadRequest(c, fmt.Sprintf("Invalid endpoint: %s. Must be one of: signup, login, password_reset, magic_link", endpoint), ErrCodeInvalidInput)
			}
		}
	}

	if req.ScoreThreshold != nil {
		if *req.ScoreThreshold < 0.0 || *req.ScoreThreshold > 1.0 {
			return SendBadRequest(c, "Score threshold must be between 0.0 and 1.0", ErrCodeInvalidInput)
		}
	}

	if h.settingsService == nil {
		return SendInternalError(c, "Settings service not initialized")
	}

	var updatedKeys []string

	updateSetting := func(key string, value interface{}) error {
		if h.settingsCache != nil && h.settingsCache.IsOverriddenByEnv(key) {
			return SendConflict(c, "This setting is controlled by configuration file or environment variable and cannot be changed", ErrCodeConflict)
		}

		if err := h.settingsService.SetSetting(ctx, key, map[string]interface{}{"value": value}, ""); err != nil {
			log.Error().Err(err).Str("key", key).Msg("Failed to update setting")
			return err
		}
		updatedKeys = append(updatedKeys, key)
		return nil
	}

	updateSecret := func(key string, value *string) error {
		if value == nil {
			return nil
		}

		if h.settingsCache != nil && h.settingsCache.IsOverriddenByEnv(key) {
			return SendConflict(c, "This setting is controlled by configuration file or environment variable and cannot be changed", ErrCodeConflict)
		}

		if h.secretsService != nil && *value != "" {
			if err := h.secretsService.SetSystemSecret(ctx, key, *value, "Captcha provider secret"); err != nil {
				log.Error().Err(err).Str("key", key).Msg("Failed to store secret")
				return err
			}
		} else if *value == "" {
			if h.secretsService != nil {
				_ = h.secretsService.DeleteSystemSecret(ctx, key)
			}
		}

		updatedKeys = append(updatedKeys, key)
		return nil
	}

	if req.Enabled != nil {
		if err := updateSetting("app.security.captcha.enabled", *req.Enabled); err != nil {
			return err
		}
	}

	if req.Provider != nil {
		if err := updateSetting("app.security.captcha.provider", *req.Provider); err != nil {
			return err
		}
	}

	if req.SiteKey != nil {
		if err := updateSetting("app.security.captcha.site_key", *req.SiteKey); err != nil {
			return err
		}
	}

	if err := updateSecret("app.security.captcha.secret_key", req.SecretKey); err != nil {
		return err
	}

	if req.ScoreThreshold != nil {
		if err := updateSetting("app.security.captcha.score_threshold", *req.ScoreThreshold); err != nil {
			return err
		}
	}

	if req.Endpoints != nil {
		if err := updateSetting("app.security.captcha.endpoints", *req.Endpoints); err != nil {
			return err
		}
	}

	if req.CapServerURL != nil {
		if err := updateSetting("app.security.captcha.cap_server_url", *req.CapServerURL); err != nil {
			return err
		}
	}

	if err := updateSecret("app.security.captcha.cap_api_key", req.CapAPIKey); err != nil {
		return err
	}

	if h.settingsCache != nil && len(updatedKeys) > 0 {
		for _, key := range updatedKeys {
			h.settingsCache.Invalidate(key)
		}
	}

	if h.captchaService != nil && len(updatedKeys) > 0 {
		if err := h.captchaService.ReloadFromSettings(ctx, h.settingsCache, h.envConfig); err != nil {
			log.Warn().Err(err).Msg("Failed to refresh captcha service after settings update")
		}
	}

	log.Info().Strs("keys", updatedKeys).Msg("Captcha settings updated")

	return h.GetSettings(c)
}
