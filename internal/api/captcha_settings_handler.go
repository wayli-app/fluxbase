package api

import (
	"context"
	"fmt"

	"github.com/fluxbase-eu/fluxbase/internal/auth"
	"github.com/fluxbase-eu/fluxbase/internal/config"
	"github.com/fluxbase-eu/fluxbase/internal/settings"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
)

// CaptchaSettingsHandler handles captcha configuration management
type CaptchaSettingsHandler struct {
	settingsService *auth.SystemSettingsService
	settingsCache   *auth.SettingsCache
	secretsService  *settings.SecretsService
	envConfig       *config.SecurityConfig // Fallback config from environment
	captchaService  *auth.CaptchaService   // Service to reload after updates
}

// NewCaptchaSettingsHandler creates a new captcha settings handler
func NewCaptchaSettingsHandler(
	settingsService *auth.SystemSettingsService,
	settingsCache *auth.SettingsCache,
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

// CaptchaSettingsResponse represents the captcha settings returned to the UI
type CaptchaSettingsResponse struct {
	Enabled        bool     `json:"enabled"`
	Provider       string   `json:"provider"`
	SiteKey        string   `json:"site_key"`
	SecretKeySet   bool     `json:"secret_key_set"`  // true if secret key is configured
	ScoreThreshold float64  `json:"score_threshold"` // For reCAPTCHA v3
	Endpoints      []string `json:"endpoints"`       // Protected endpoints
	CapServerURL   string   `json:"cap_server_url"`  // For Cap provider
	CapAPIKeySet   bool     `json:"cap_api_key_set"` // true if Cap API key is configured

	// Override information
	Overrides map[string]OverrideInfo `json:"_overrides"`
}

// UpdateCaptchaSettingsRequest represents the request to update captcha settings
type UpdateCaptchaSettingsRequest struct {
	Enabled        *bool     `json:"enabled,omitempty"`
	Provider       *string   `json:"provider,omitempty"`
	SiteKey        *string   `json:"site_key,omitempty"`
	SecretKey      *string   `json:"secret_key,omitempty"`      // Only set if changing
	ScoreThreshold *float64  `json:"score_threshold,omitempty"` // For reCAPTCHA v3
	Endpoints      *[]string `json:"endpoints,omitempty"`
	CapServerURL   *string   `json:"cap_server_url,omitempty"` // For Cap provider
	CapAPIKey      *string   `json:"cap_api_key,omitempty"`    // Only set if changing
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

// GetSettings returns the current captcha settings
// GET /api/v1/admin/settings/captcha
func (h *CaptchaSettingsHandler) GetSettings(c *fiber.Ctx) error {
	ctx := context.Background()

	response := CaptchaSettingsResponse{
		Overrides: make(map[string]OverrideInfo),
	}

	// Helper to get string value with override check
	getString := func(key, defaultVal string) (string, bool) {
		if h.settingsCache != nil && h.settingsCache.IsOverriddenByEnv(key) {
			return h.settingsCache.GetString(ctx, key, defaultVal), true
		}
		return h.settingsCache.GetString(ctx, key, defaultVal), false
	}

	// Helper to get bool value with override check
	getBool := func(key string, defaultVal bool) (bool, bool) {
		if h.settingsCache != nil && h.settingsCache.IsOverriddenByEnv(key) {
			return h.settingsCache.GetBool(ctx, key, defaultVal), true
		}
		return h.settingsCache.GetBool(ctx, key, defaultVal), false
	}

	// Helper to get float64 value with override check
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

	// Helper to get string slice with override check
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

	// Helper to add override info
	addOverride := func(field, key string) {
		if h.settingsCache != nil && h.settingsCache.IsOverriddenByEnv(key) {
			response.Overrides[field] = OverrideInfo{
				IsOverridden: true,
				EnvVar:       h.settingsCache.GetEnvVarName(key),
			}
		}
	}

	// Get basic settings
	response.Enabled, _ = getBool("app.security.captcha.enabled", false)
	addOverride("enabled", "app.security.captcha.enabled")

	response.Provider, _ = getString("app.security.captcha.provider", "hcaptcha")
	addOverride("provider", "app.security.captcha.provider")

	response.SiteKey, _ = getString("app.security.captcha.site_key", "")
	addOverride("site_key", "app.security.captcha.site_key")

	// Check if secret key is set (don't return the actual value)
	secretKey, _ := getString("app.security.captcha.secret_key", "")
	response.SecretKeySet = secretKey != ""
	addOverride("secret_key", "app.security.captcha.secret_key")

	response.ScoreThreshold, _ = getFloat64("app.security.captcha.score_threshold", 0.5)
	addOverride("score_threshold", "app.security.captcha.score_threshold")

	response.Endpoints, _ = getStringSlice("app.security.captcha.endpoints", []string{"signup", "login", "password_reset", "magic_link"})
	addOverride("endpoints", "app.security.captcha.endpoints")

	// Cap provider settings
	response.CapServerURL, _ = getString("app.security.captcha.cap_server_url", "")
	addOverride("cap_server_url", "app.security.captcha.cap_server_url")

	capAPIKey, _ := getString("app.security.captcha.cap_api_key", "")
	response.CapAPIKeySet = capAPIKey != ""
	addOverride("cap_api_key", "app.security.captcha.cap_api_key")

	return c.JSON(response)
}

// UpdateSettings updates captcha settings
// PUT /api/v1/admin/settings/captcha
func (h *CaptchaSettingsHandler) UpdateSettings(c *fiber.Ctx) error {
	ctx := context.Background()

	var req UpdateCaptchaSettingsRequest
	if err := c.BodyParser(&req); err != nil {
		log.Error().Err(err).Msg("Failed to parse update captcha settings request")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate provider if provided
	if req.Provider != nil {
		if !validProviders[*req.Provider] {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid provider. Must be one of: hcaptcha, recaptcha_v3, turnstile, cap",
				"code":  "INVALID_PROVIDER",
			})
		}
	}

	// Validate endpoints if provided
	if req.Endpoints != nil {
		for _, endpoint := range *req.Endpoints {
			if !validEndpoints[endpoint] {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": fmt.Sprintf("Invalid endpoint: %s. Must be one of: signup, login, password_reset, magic_link", endpoint),
					"code":  "INVALID_ENDPOINT",
				})
			}
		}
	}

	// Validate score threshold if provided
	if req.ScoreThreshold != nil {
		if *req.ScoreThreshold < 0.0 || *req.ScoreThreshold > 1.0 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Score threshold must be between 0.0 and 1.0",
				"code":  "INVALID_SCORE_THRESHOLD",
			})
		}
	}

	// Track which settings were updated
	var updatedKeys []string

	// Helper to update a setting with override check
	updateSetting := func(key string, value interface{}) error {
		// Check if overridden by env var or config
		if h.settingsCache != nil && h.settingsCache.IsOverriddenByEnv(key) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": "This setting is controlled by configuration file or environment variable and cannot be changed",
				"code":  "CONFIG_OVERRIDE",
				"key":   key,
			})
		}

		if err := h.settingsService.SetSetting(ctx, key, map[string]interface{}{"value": value}, ""); err != nil {
			log.Error().Err(err).Str("key", key).Msg("Failed to update setting")
			return err
		}
		updatedKeys = append(updatedKeys, key)
		return nil
	}

	// Helper to encrypt and update a secret using SecretsService
	updateSecret := func(key string, value *string) error {
		if value == nil {
			return nil // Not updating this field
		}

		// Check if overridden by env var or config
		if h.settingsCache != nil && h.settingsCache.IsOverriddenByEnv(key) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": "This setting is controlled by configuration file or environment variable and cannot be changed",
				"code":  "CONFIG_OVERRIDE",
				"key":   key,
			})
		}

		// Use SecretsService to encrypt and store the secret
		if h.secretsService != nil && *value != "" {
			if err := h.secretsService.SetSystemSecret(ctx, key, *value, "Captcha provider secret"); err != nil {
				log.Error().Err(err).Str("key", key).Msg("Failed to store secret")
				return err
			}
		} else if *value == "" {
			// Clear the secret by deleting it
			if h.secretsService != nil {
				_ = h.secretsService.DeleteSystemSecret(ctx, key) // Ignore not found errors
			}
		}

		updatedKeys = append(updatedKeys, key)
		return nil
	}

	// Update basic settings
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

	// Cap provider settings
	if req.CapServerURL != nil {
		if err := updateSetting("app.security.captcha.cap_server_url", *req.CapServerURL); err != nil {
			return err
		}
	}

	if err := updateSecret("app.security.captcha.cap_api_key", req.CapAPIKey); err != nil {
		return err
	}

	// Refresh captcha service with new settings
	if h.captchaService != nil && len(updatedKeys) > 0 {
		if err := h.captchaService.ReloadFromSettings(ctx, h.settingsCache, h.envConfig); err != nil {
			log.Warn().Err(err).Msg("Failed to refresh captcha service after settings update")
			// Don't fail the request - settings are saved, service will refresh on next restart
		}
	}

	log.Info().Strs("keys", updatedKeys).Msg("Captcha settings updated")

	// Return updated settings
	return h.GetSettings(c)
}
