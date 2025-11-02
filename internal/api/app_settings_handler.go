package api

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
	"github.com/wayli-app/fluxbase/internal/auth"
)

// AppSettingsHandler handles application-wide settings
type AppSettingsHandler struct {
	settingsService *auth.SystemSettingsService
}

// NewAppSettingsHandler creates a new app settings handler
func NewAppSettingsHandler(settingsService *auth.SystemSettingsService) *AppSettingsHandler {
	return &AppSettingsHandler{
		settingsService: settingsService,
	}
}

// AppSettings represents the structured application settings
type AppSettings struct {
	Authentication AuthenticationSettings `json:"authentication"`
	Features       FeatureSettings        `json:"features"`
	Email          EmailSettings          `json:"email"`
	Security       SecuritySettings       `json:"security"`
}

// AuthenticationSettings contains authentication-related settings
type AuthenticationSettings struct {
	EnableSignup             bool `json:"enable_signup"`
	EnableMagicLink          bool `json:"enable_magic_link"`
	PasswordMinLength        int  `json:"password_min_length"`
	RequireEmailVerification bool `json:"require_email_verification"`
}

// FeatureSettings contains feature flags
type FeatureSettings struct {
	EnableRealtime  bool `json:"enable_realtime"`
	EnableStorage   bool `json:"enable_storage"`
	EnableFunctions bool `json:"enable_functions"`
}

// EmailSettings contains email configuration
type EmailSettings struct {
	Enabled  bool   `json:"enabled"`
	Provider string `json:"provider"`
}

// SecuritySettings contains security-related settings
type SecuritySettings struct {
	EnableGlobalRateLimit bool `json:"enable_global_rate_limit"`
}

// GetAppSettings returns all application settings in a structured format
// GET /api/v1/admin/app/settings
func (h *AppSettingsHandler) GetAppSettings(c *fiber.Ctx) error {
	ctx := context.Background()

	settings := &AppSettings{
		Authentication: AuthenticationSettings{
			EnableSignup:             h.getBoolSetting(ctx, "app.auth.enable_signup", false),
			EnableMagicLink:          h.getBoolSetting(ctx, "app.auth.enable_magic_link", true),
			PasswordMinLength:        h.getIntSetting(ctx, "app.auth.password_min_length", 8),
			RequireEmailVerification: h.getBoolSetting(ctx, "app.auth.require_email_verification", false),
		},
		Features: FeatureSettings{
			EnableRealtime:  h.getBoolSetting(ctx, "app.features.enable_realtime", true),
			EnableStorage:   h.getBoolSetting(ctx, "app.features.enable_storage", true),
			EnableFunctions: h.getBoolSetting(ctx, "app.features.enable_functions", true),
		},
		Email: EmailSettings{
			Enabled:  h.getBoolSetting(ctx, "app.email.enabled", false),
			Provider: h.getStringSetting(ctx, "app.email.provider", "smtp"),
		},
		Security: SecuritySettings{
			EnableGlobalRateLimit: h.getBoolSetting(ctx, "app.security.enable_global_rate_limit", false),
		},
	}

	return c.JSON(settings)
}

// UpdateAppSettings updates multiple application settings at once
// PUT /api/v1/admin/app/settings
func (h *AppSettingsHandler) UpdateAppSettings(c *fiber.Ctx) error {
	ctx := context.Background()

	var updates map[string]interface{}
	if err := c.BodyParser(&updates); err != nil {
		log.Error().Err(err).Msg("Failed to parse app settings update")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Process authentication settings
	if auth, ok := updates["authentication"].(map[string]interface{}); ok {
		h.updateAuthSettings(ctx, auth)
	}

	// Process feature settings
	if features, ok := updates["features"].(map[string]interface{}); ok {
		h.updateFeatureSettings(ctx, features)
	}

	// Process email settings
	if email, ok := updates["email"].(map[string]interface{}); ok {
		h.updateEmailSettings(ctx, email)
	}

	// Process security settings
	if security, ok := updates["security"].(map[string]interface{}); ok {
		h.updateSecuritySettings(ctx, security)
	}

	log.Info().Interface("updates", updates).Msg("App settings updated")

	// Return updated settings
	return h.GetAppSettings(c)
}

// ResetAppSettings resets all application settings to defaults
// POST /api/v1/admin/app/settings/reset
func (h *AppSettingsHandler) ResetAppSettings(c *fiber.Ctx) error {
	ctx := context.Background()

	// Delete all app settings to fall back to defaults
	settingsToReset := []string{
		"app.auth.enable_signup",
		"app.auth.enable_magic_link",
		"app.auth.password_min_length",
		"app.auth.require_email_verification",
		"app.features.enable_realtime",
		"app.features.enable_storage",
		"app.features.enable_functions",
		"app.email.enabled",
		"app.email.provider",
		"app.security.enable_global_rate_limit",
	}

	for _, key := range settingsToReset {
		if err := h.settingsService.DeleteSetting(ctx, key); err != nil {
			log.Warn().Err(err).Str("key", key).Msg("Failed to delete setting during reset")
		}
	}

	log.Info().Msg("App settings reset to defaults")

	// Return default settings
	return h.GetAppSettings(c)
}

// Helper methods to get settings with defaults

func (h *AppSettingsHandler) getBoolSetting(ctx context.Context, key string, defaultValue bool) bool {
	setting, err := h.settingsService.GetSetting(ctx, key)
	if err != nil {
		return defaultValue
	}

	// The value is stored as map[string]interface{}, we need to extract the actual value
	if boolValue, ok := setting.Value["value"].(bool); ok {
		return boolValue
	}

	return defaultValue
}

func (h *AppSettingsHandler) getIntSetting(ctx context.Context, key string, defaultValue int) int {
	setting, err := h.settingsService.GetSetting(ctx, key)
	if err != nil {
		return defaultValue
	}

	// The value is stored as map[string]interface{}, handle both int and float64
	switch v := setting.Value["value"].(type) {
	case int:
		return v
	case float64:
		return int(v)
	}

	return defaultValue
}

func (h *AppSettingsHandler) getStringSetting(ctx context.Context, key string, defaultValue string) string {
	setting, err := h.settingsService.GetSetting(ctx, key)
	if err != nil {
		return defaultValue
	}

	// The value is stored as map[string]interface{}, we need to extract the actual value
	if strValue, ok := setting.Value["value"].(string); ok {
		return strValue
	}

	return defaultValue
}

// Helper methods to update settings

func (h *AppSettingsHandler) updateAuthSettings(ctx context.Context, auth map[string]interface{}) {
	if val, ok := auth["enable_signup"].(bool); ok {
		h.setSetting(ctx, "app.auth.enable_signup", val, "Enable user signup")
	}
	if val, ok := auth["enable_magic_link"].(bool); ok {
		h.setSetting(ctx, "app.auth.enable_magic_link", val, "Enable magic link authentication")
	}
	if val, ok := auth["password_min_length"].(float64); ok {
		h.setSetting(ctx, "app.auth.password_min_length", int(val), "Minimum password length")
	}
	if val, ok := auth["require_email_verification"].(bool); ok {
		h.setSetting(ctx, "app.auth.require_email_verification", val, "Require email verification")
	}
}

func (h *AppSettingsHandler) updateFeatureSettings(ctx context.Context, features map[string]interface{}) {
	if val, ok := features["enable_realtime"].(bool); ok {
		h.setSetting(ctx, "app.features.enable_realtime", val, "Enable realtime features")
	}
	if val, ok := features["enable_storage"].(bool); ok {
		h.setSetting(ctx, "app.features.enable_storage", val, "Enable storage features")
	}
	if val, ok := features["enable_functions"].(bool); ok {
		h.setSetting(ctx, "app.features.enable_functions", val, "Enable edge functions")
	}
}

func (h *AppSettingsHandler) updateEmailSettings(ctx context.Context, email map[string]interface{}) {
	if val, ok := email["enabled"].(bool); ok {
		h.setSetting(ctx, "app.email.enabled", val, "Enable email service")
	}
	if val, ok := email["provider"].(string); ok {
		h.setSetting(ctx, "app.email.provider", val, "Email provider")
	}
}

func (h *AppSettingsHandler) updateSecuritySettings(ctx context.Context, security map[string]interface{}) {
	if val, ok := security["enable_global_rate_limit"].(bool); ok {
		h.setSetting(ctx, "app.security.enable_global_rate_limit", val, "Enable global rate limiting")
	}
}

// setSetting is a generic helper to set a setting with proper value wrapping
func (h *AppSettingsHandler) setSetting(ctx context.Context, key string, value interface{}, description string) {
	// Wrap the value in a map to match the expected structure
	valueMap := map[string]interface{}{
		"value": value,
	}

	if err := h.settingsService.SetSetting(ctx, key, valueMap, description); err != nil {
		log.Error().Err(err).Str("key", key).Interface("value", value).Msg("Failed to set setting")
	}
}
