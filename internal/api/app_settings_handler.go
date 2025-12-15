package api

import (
	"context"
	"fmt"

	"github.com/fluxbase-eu/fluxbase/internal/auth"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
)

// AppSettingsHandler handles application settings operations
type AppSettingsHandler struct {
	settingsService *auth.SystemSettingsService
	settingsCache   *auth.SettingsCache
}

// NewAppSettingsHandler creates a new app settings handler
func NewAppSettingsHandler(settingsService *auth.SystemSettingsService, settingsCache *auth.SettingsCache) *AppSettingsHandler {
	return &AppSettingsHandler{
		settingsService: settingsService,
		settingsCache:   settingsCache,
	}
}

// AppSettings represents the structured application settings
type AppSettings struct {
	Authentication AuthenticationSettings `json:"authentication"`
	Features       FeatureSettings        `json:"features"`
	Email          EmailSettings          `json:"email"`
	Security       SecuritySettings       `json:"security"`
	Overrides      SettingOverrides       `json:"overrides,omitempty"` // Indicates which settings are overridden by environment variables
}

// SettingOverrides indicates which settings are overridden by environment variables
type SettingOverrides struct {
	Authentication map[string]bool `json:"authentication,omitempty"`
	Features       map[string]bool `json:"features,omitempty"`
	Email          map[string]bool `json:"email,omitempty"`
	Security       map[string]bool `json:"security,omitempty"`
}

// AuthenticationSettings contains authentication-related settings
type AuthenticationSettings struct {
	EnableSignup             bool `json:"enable_signup"`
	EnableMagicLink          bool `json:"enable_magic_link"`
	PasswordMinLength        int  `json:"password_min_length"`
	RequireEmailVerification bool `json:"require_email_verification"`
	PasswordRequireUppercase bool `json:"password_require_uppercase"`
	PasswordRequireLowercase bool `json:"password_require_lowercase"`
	PasswordRequireNumber    bool `json:"password_require_number"`
	PasswordRequireSpecial   bool `json:"password_require_special"`
	SessionTimeoutMinutes    int  `json:"session_timeout_minutes"`
	MaxSessionsPerUser       int  `json:"max_sessions_per_user"`
}

// FeatureSettings contains feature flag settings
type FeatureSettings struct {
	EnableRealtime  bool `json:"enable_realtime"`
	EnableStorage   bool `json:"enable_storage"`
	EnableFunctions bool `json:"enable_functions"`
}

// EmailSettings contains email configuration
type EmailSettings struct {
	Enabled        bool              `json:"enabled"`
	Provider       string            `json:"provider"`
	FromAddress    string            `json:"from_address,omitempty"`
	FromName       string            `json:"from_name,omitempty"`
	ReplyToAddress string            `json:"reply_to_address,omitempty"`
	SMTP           *SMTPSettings     `json:"smtp,omitempty"`
	SendGrid       *SendGridSettings `json:"sendgrid,omitempty"`
	Mailgun        *MailgunSettings  `json:"mailgun,omitempty"`
	SES            *SESSettings      `json:"ses,omitempty"`
}

// SMTPSettings contains SMTP configuration
type SMTPSettings struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password,omitempty"` // Omit in responses for security
	TLS      bool   `json:"tls"`
}

// SendGridSettings contains SendGrid configuration
type SendGridSettings struct {
	APIKey string `json:"api_key,omitempty"` // Omit in responses for security
}

// MailgunSettings contains Mailgun configuration
type MailgunSettings struct {
	APIKey string `json:"api_key,omitempty"` // Omit in responses for security
	Domain string `json:"domain"`
}

// SESSettings contains AWS SES configuration
type SESSettings struct {
	Region          string `json:"region"`
	AccessKeyID     string `json:"access_key_id,omitempty"`     // Omit in responses for security
	SecretAccessKey string `json:"secret_access_key,omitempty"` // Omit in responses for security
}

// SecuritySettings contains security-related settings
type SecuritySettings struct {
	EnableGlobalRateLimit bool `json:"enable_global_rate_limit"`
}

// UpdateAppSettingsRequest represents the request to update app settings
type UpdateAppSettingsRequest struct {
	Authentication *AuthenticationSettings `json:"authentication,omitempty"`
	Features       *FeatureSettings        `json:"features,omitempty"`
	Email          *EmailSettings          `json:"email,omitempty"`
	Security       *SecuritySettings       `json:"security,omitempty"`
}

// GetAppSettings returns all application settings in a structured format
// GET /api/v1/admin/app/settings
func (h *AppSettingsHandler) GetAppSettings(c *fiber.Ctx) error {
	ctx := context.Background()

	// Get all system settings
	settings, err := h.settingsService.ListSettings(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to list system settings")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve application settings",
		})
	}

	// Build structured response
	appSettings := h.buildAppSettings(settings)

	return c.JSON(appSettings)
}

// UpdateAppSettings updates application settings
// PUT /api/v1/admin/app/settings
func (h *AppSettingsHandler) UpdateAppSettings(c *fiber.Ctx) error {
	ctx := context.Background()

	var req UpdateAppSettingsRequest
	if err := c.BodyParser(&req); err != nil {
		log.Error().Err(err).Msg("Failed to parse request body")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Update authentication settings
	if req.Authentication != nil {
		if err := h.updateAuthSettings(ctx, req.Authentication); err != nil {
			log.Error().Err(err).Msg("Failed to update authentication settings")
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to update authentication settings",
			})
		}
	}

	// Update feature settings
	if req.Features != nil {
		if err := h.updateFeatureSettings(ctx, req.Features); err != nil {
			log.Error().Err(err).Msg("Failed to update feature settings")
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to update feature settings",
			})
		}
	}

	// Update email settings
	if req.Email != nil {
		if err := h.updateEmailSettings(ctx, req.Email); err != nil {
			log.Error().Err(err).Msg("Failed to update email settings")
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to update email settings",
			})
		}
	}

	// Update security settings
	if req.Security != nil {
		if err := h.updateSecuritySettings(ctx, req.Security); err != nil {
			log.Error().Err(err).Msg("Failed to update security settings")
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to update security settings",
			})
		}
	}

	// Get updated settings
	settings, err := h.settingsService.ListSettings(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to list system settings after update")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve updated settings",
		})
	}

	appSettings := h.buildAppSettings(settings)

	log.Info().Msg("Application settings updated")

	return c.JSON(appSettings)
}

// Helper function to build AppSettings from system settings
func (h *AppSettingsHandler) buildAppSettings(settings []auth.SystemSetting) AppSettings {
	appSettings := AppSettings{
		Authentication: AuthenticationSettings{
			PasswordMinLength:     8, // defaults
			SessionTimeoutMinutes: 60,
			MaxSessionsPerUser:    5,
		},
		Features: FeatureSettings{
			EnableRealtime:  true,
			EnableStorage:   true,
			EnableFunctions: true,
		},
		Email: EmailSettings{
			Provider: "smtp",
		},
		Security: SecuritySettings{},
		Overrides: SettingOverrides{
			Authentication: make(map[string]bool),
			Features:       make(map[string]bool),
			Email:          make(map[string]bool),
			Security:       make(map[string]bool),
		},
	}

	for _, setting := range settings {
		switch setting.Key {
		// Authentication
		case "app.auth.enable_signup":
			if val, ok := setting.Value["value"].(bool); ok {
				appSettings.Authentication.EnableSignup = val
			}
		case "app.auth.enable_magic_link":
			if val, ok := setting.Value["value"].(bool); ok {
				appSettings.Authentication.EnableMagicLink = val
			}
		case "app.auth.password_min_length":
			if val, ok := setting.Value["value"].(float64); ok {
				appSettings.Authentication.PasswordMinLength = int(val)
			}
		case "app.auth.require_email_verification":
			if val, ok := setting.Value["value"].(bool); ok {
				appSettings.Authentication.RequireEmailVerification = val
			}

		// Features
		case "app.realtime.enabled":
			if val, ok := setting.Value["value"].(bool); ok {
				appSettings.Features.EnableRealtime = val
			}
		case "app.storage.enabled":
			if val, ok := setting.Value["value"].(bool); ok {
				appSettings.Features.EnableStorage = val
			}
		case "app.functions.enabled":
			if val, ok := setting.Value["value"].(bool); ok {
				appSettings.Features.EnableFunctions = val
			}

		// Email
		case "app.email.enabled":
			if val, ok := setting.Value["value"].(bool); ok {
				appSettings.Email.Enabled = val
			}
		case "app.email.provider":
			if val, ok := setting.Value["value"].(string); ok {
				appSettings.Email.Provider = val
			}

		// Security
		case "app.security.enable_global_rate_limit":
			if val, ok := setting.Value["value"].(bool); ok {
				appSettings.Security.EnableGlobalRateLimit = val
			}
		}
	}

	// Check for environment variable overrides
	if h.settingsCache != nil {
		// Authentication overrides
		if h.settingsCache.IsOverriddenByEnv("app.auth.enable_signup") {
			appSettings.Overrides.Authentication["enable_signup"] = true
		}
		if h.settingsCache.IsOverriddenByEnv("app.auth.enable_magic_link") {
			appSettings.Overrides.Authentication["enable_magic_link"] = true
		}
		if h.settingsCache.IsOverriddenByEnv("app.auth.password_min_length") {
			appSettings.Overrides.Authentication["password_min_length"] = true
		}
		if h.settingsCache.IsOverriddenByEnv("app.auth.require_email_verification") {
			appSettings.Overrides.Authentication["require_email_verification"] = true
		}

		// Features overrides
		if h.settingsCache.IsOverriddenByEnv("app.realtime.enabled") {
			appSettings.Overrides.Features["enable_realtime"] = true
		}
		if h.settingsCache.IsOverriddenByEnv("app.storage.enabled") {
			appSettings.Overrides.Features["enable_storage"] = true
		}
		if h.settingsCache.IsOverriddenByEnv("app.functions.enabled") {
			appSettings.Overrides.Features["enable_functions"] = true
		}

		// Email overrides
		if h.settingsCache.IsOverriddenByEnv("app.email.enabled") {
			appSettings.Overrides.Email["enabled"] = true
		}
		if h.settingsCache.IsOverriddenByEnv("app.email.provider") {
			appSettings.Overrides.Email["provider"] = true
		}

		// Security overrides
		if h.settingsCache.IsOverriddenByEnv("app.security.enable_global_rate_limit") {
			appSettings.Overrides.Security["enable_global_rate_limit"] = true
		}
	}

	return appSettings
}

// Helper functions to update specific setting categories
func (h *AppSettingsHandler) updateAuthSettings(ctx context.Context, auth *AuthenticationSettings) error {
	settingsMap := map[string]interface{}{
		"app.auth.enable_signup":              auth.EnableSignup,
		"app.auth.enable_magic_link":          auth.EnableMagicLink,
		"app.auth.password_min_length":        auth.PasswordMinLength,
		"app.auth.require_email_verification": auth.RequireEmailVerification,
	}

	for key, value := range settingsMap {
		// Check if setting is overridden by environment variable
		if h.settingsCache != nil && h.settingsCache.IsOverriddenByEnv(key) {
			return fiber.NewError(fiber.StatusConflict,
				fmt.Sprintf("Setting '%s' cannot be updated because it is overridden by an environment variable", key))
		}

		if err := h.settingsService.SetSetting(ctx, key, map[string]interface{}{"value": value}, ""); err != nil {
			return err
		}
	}

	return nil
}

func (h *AppSettingsHandler) updateFeatureSettings(ctx context.Context, features *FeatureSettings) error {
	settingsMap := map[string]interface{}{
		"app.realtime.enabled":  features.EnableRealtime,
		"app.storage.enabled":   features.EnableStorage,
		"app.functions.enabled": features.EnableFunctions,
	}

	for key, value := range settingsMap {
		// Check if setting is overridden by environment variable
		if h.settingsCache != nil && h.settingsCache.IsOverriddenByEnv(key) {
			return fiber.NewError(fiber.StatusConflict,
				fmt.Sprintf("Setting '%s' cannot be updated because it is overridden by an environment variable", key))
		}

		if err := h.settingsService.SetSetting(ctx, key, map[string]interface{}{"value": value}, ""); err != nil {
			return err
		}
	}

	return nil
}

func (h *AppSettingsHandler) updateEmailSettings(ctx context.Context, email *EmailSettings) error {
	settingsMap := map[string]interface{}{
		"app.email.enabled":  email.Enabled,
		"app.email.provider": email.Provider,
	}

	for key, value := range settingsMap {
		// Check if setting is overridden by environment variable
		if h.settingsCache != nil && h.settingsCache.IsOverriddenByEnv(key) {
			return fiber.NewError(fiber.StatusConflict,
				fmt.Sprintf("Setting '%s' cannot be updated because it is overridden by an environment variable", key))
		}

		if err := h.settingsService.SetSetting(ctx, key, map[string]interface{}{"value": value}, ""); err != nil {
			return err
		}
	}

	return nil
}

func (h *AppSettingsHandler) updateSecuritySettings(ctx context.Context, security *SecuritySettings) error {
	key := "app.security.enable_global_rate_limit"

	// Check if setting is overridden by environment variable
	if h.settingsCache != nil && h.settingsCache.IsOverriddenByEnv(key) {
		return fiber.NewError(fiber.StatusConflict,
			fmt.Sprintf("Setting '%s' cannot be updated because it is overridden by an environment variable", key))
	}

	return h.settingsService.SetSetting(
		ctx,
		key,
		map[string]interface{}{"value": security.EnableGlobalRateLimit},
		"",
	)
}
