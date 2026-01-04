package api

import (
	"context"
	"fmt"

	"github.com/fluxbase-eu/fluxbase/internal/auth"
	"github.com/fluxbase-eu/fluxbase/internal/config"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
)

// AppSettingsHandler handles application settings operations
type AppSettingsHandler struct {
	settingsService *auth.SystemSettingsService
	settingsCache   *auth.SettingsCache
	config          *config.Config
}

// NewAppSettingsHandler creates a new app settings handler
func NewAppSettingsHandler(settingsService *auth.SystemSettingsService, settingsCache *auth.SettingsCache, cfg *config.Config) *AppSettingsHandler {
	return &AppSettingsHandler{
		settingsService: settingsService,
		settingsCache:   settingsCache,
		config:          cfg,
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
	SignupEnabled            bool `json:"enable_signup"`
	MagicLinkEnabled         bool `json:"enable_magic_link"`
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
	APIKey   string `json:"api_key,omitempty"` // Omit in responses for security
	Domain   string `json:"domain"`
	EURegion bool   `json:"eu_region"`
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
		case "app.auth.signup_enabled":
			if val, ok := setting.Value["value"].(bool); ok {
				appSettings.Authentication.SignupEnabled = val
			}
		case "app.auth.magic_link_enabled":
			if val, ok := setting.Value["value"].(bool); ok {
				appSettings.Authentication.MagicLinkEnabled = val
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

		// Email - basic settings
		case "app.email.enabled":
			if val, ok := setting.Value["value"].(bool); ok {
				appSettings.Email.Enabled = val
			}
		case "app.email.provider":
			if val, ok := setting.Value["value"].(string); ok {
				appSettings.Email.Provider = val
			}
		case "app.email.from_address":
			if val, ok := setting.Value["value"].(string); ok {
				appSettings.Email.FromAddress = val
			}
		case "app.email.from_name":
			if val, ok := setting.Value["value"].(string); ok {
				appSettings.Email.FromName = val
			}
		case "app.email.reply_to_address":
			if val, ok := setting.Value["value"].(string); ok {
				appSettings.Email.ReplyToAddress = val
			}

		// Email - SMTP settings
		case "app.email.smtp.host":
			if val, ok := setting.Value["value"].(string); ok {
				if appSettings.Email.SMTP == nil {
					appSettings.Email.SMTP = &SMTPSettings{}
				}
				appSettings.Email.SMTP.Host = val
			}
		case "app.email.smtp.port":
			if val, ok := setting.Value["value"].(float64); ok {
				if appSettings.Email.SMTP == nil {
					appSettings.Email.SMTP = &SMTPSettings{}
				}
				appSettings.Email.SMTP.Port = int(val)
			}
		case "app.email.smtp.username":
			if val, ok := setting.Value["value"].(string); ok {
				if appSettings.Email.SMTP == nil {
					appSettings.Email.SMTP = &SMTPSettings{}
				}
				appSettings.Email.SMTP.Username = val
			}
		case "app.email.smtp.tls":
			if val, ok := setting.Value["value"].(bool); ok {
				if appSettings.Email.SMTP == nil {
					appSettings.Email.SMTP = &SMTPSettings{}
				}
				appSettings.Email.SMTP.TLS = val
			}
		// Note: app.email.smtp.password is stored but never returned (omitted for security)

		// Email - Mailgun settings
		case "app.email.mailgun.domain":
			if val, ok := setting.Value["value"].(string); ok {
				if appSettings.Email.Mailgun == nil {
					appSettings.Email.Mailgun = &MailgunSettings{}
				}
				appSettings.Email.Mailgun.Domain = val
			}
		case "app.email.mailgun.eu_region":
			if val, ok := setting.Value["value"].(bool); ok {
				if appSettings.Email.Mailgun == nil {
					appSettings.Email.Mailgun = &MailgunSettings{}
				}
				appSettings.Email.Mailgun.EURegion = val
			}
		// Note: app.email.mailgun.api_key is stored but never returned (omitted for security)

		// Email - SES settings
		case "app.email.ses.region":
			if val, ok := setting.Value["value"].(string); ok {
				if appSettings.Email.SES == nil {
					appSettings.Email.SES = &SESSettings{}
				}
				appSettings.Email.SES.Region = val
			}
		// Note: app.email.ses.access_key_id and app.email.ses.secret_access_key are stored but never returned

		// Security
		case "app.security.enable_global_rate_limit":
			if val, ok := setting.Value["value"].(bool); ok {
				appSettings.Security.EnableGlobalRateLimit = val
			}
		}
	}

	// Apply email settings from config (environment variables take precedence)
	if h.config != nil {
		emailCfg := h.config.Email

		// Basic email settings from config
		if emailCfg.Enabled {
			appSettings.Email.Enabled = emailCfg.Enabled
		}
		if emailCfg.Provider != "" {
			appSettings.Email.Provider = emailCfg.Provider
		}
		if emailCfg.FromAddress != "" {
			appSettings.Email.FromAddress = emailCfg.FromAddress
		}
		if emailCfg.FromName != "" {
			appSettings.Email.FromName = emailCfg.FromName
		}
		if emailCfg.ReplyToAddress != "" {
			appSettings.Email.ReplyToAddress = emailCfg.ReplyToAddress
		}

		// SMTP settings from config
		if emailCfg.SMTPHost != "" || emailCfg.SMTPPort != 0 || emailCfg.SMTPUsername != "" {
			if appSettings.Email.SMTP == nil {
				appSettings.Email.SMTP = &SMTPSettings{}
			}
			if emailCfg.SMTPHost != "" {
				appSettings.Email.SMTP.Host = emailCfg.SMTPHost
			}
			if emailCfg.SMTPPort != 0 {
				appSettings.Email.SMTP.Port = emailCfg.SMTPPort
			}
			if emailCfg.SMTPUsername != "" {
				appSettings.Email.SMTP.Username = emailCfg.SMTPUsername
			}
			if emailCfg.SMTPTLS {
				appSettings.Email.SMTP.TLS = emailCfg.SMTPTLS
			}
			// Note: Password is never returned
		}

		// Mailgun settings from config
		if emailCfg.MailgunDomain != "" {
			if appSettings.Email.Mailgun == nil {
				appSettings.Email.Mailgun = &MailgunSettings{}
			}
			appSettings.Email.Mailgun.Domain = emailCfg.MailgunDomain
			// Note: API key is never returned
		}

		// SES settings from config
		if emailCfg.SESRegion != "" {
			if appSettings.Email.SES == nil {
				appSettings.Email.SES = &SESSettings{}
			}
			appSettings.Email.SES.Region = emailCfg.SESRegion
			// Note: Access key and secret key are never returned
		}
	}

	// Check for environment variable overrides
	if h.settingsCache != nil {
		// Authentication overrides
		if h.settingsCache.IsOverriddenByEnv("app.auth.signup_enabled") {
			appSettings.Overrides.Authentication["enable_signup"] = true
		}
		if h.settingsCache.IsOverriddenByEnv("app.auth.magic_link_enabled") {
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

		// Email overrides - basic settings
		if h.settingsCache.IsOverriddenByEnv("app.email.enabled") {
			appSettings.Overrides.Email["enabled"] = true
		}
		if h.settingsCache.IsOverriddenByEnv("app.email.provider") {
			appSettings.Overrides.Email["provider"] = true
		}
		if h.settingsCache.IsOverriddenByEnv("app.email.from_address") {
			appSettings.Overrides.Email["from_address"] = true
		}
		if h.settingsCache.IsOverriddenByEnv("app.email.from_name") {
			appSettings.Overrides.Email["from_name"] = true
		}
		if h.settingsCache.IsOverriddenByEnv("app.email.reply_to_address") {
			appSettings.Overrides.Email["reply_to_address"] = true
		}
		// Email overrides - SMTP
		if h.settingsCache.IsOverriddenByEnv("app.email.smtp.host") {
			appSettings.Overrides.Email["smtp.host"] = true
		}
		if h.settingsCache.IsOverriddenByEnv("app.email.smtp.port") {
			appSettings.Overrides.Email["smtp.port"] = true
		}
		if h.settingsCache.IsOverriddenByEnv("app.email.smtp.username") {
			appSettings.Overrides.Email["smtp.username"] = true
		}
		if h.settingsCache.IsOverriddenByEnv("app.email.smtp.password") {
			appSettings.Overrides.Email["smtp.password"] = true
		}
		if h.settingsCache.IsOverriddenByEnv("app.email.smtp.tls") {
			appSettings.Overrides.Email["smtp.tls"] = true
		}
		// Email overrides - SendGrid
		if h.settingsCache.IsOverriddenByEnv("app.email.sendgrid.api_key") {
			appSettings.Overrides.Email["sendgrid.api_key"] = true
		}
		// Email overrides - Mailgun
		if h.settingsCache.IsOverriddenByEnv("app.email.mailgun.api_key") {
			appSettings.Overrides.Email["mailgun.api_key"] = true
		}
		if h.settingsCache.IsOverriddenByEnv("app.email.mailgun.domain") {
			appSettings.Overrides.Email["mailgun.domain"] = true
		}
		if h.settingsCache.IsOverriddenByEnv("app.email.mailgun.eu_region") {
			appSettings.Overrides.Email["mailgun.eu_region"] = true
		}
		// Email overrides - SES
		if h.settingsCache.IsOverriddenByEnv("app.email.ses.region") {
			appSettings.Overrides.Email["ses.region"] = true
		}
		if h.settingsCache.IsOverriddenByEnv("app.email.ses.access_key_id") {
			appSettings.Overrides.Email["ses.access_key_id"] = true
		}
		if h.settingsCache.IsOverriddenByEnv("app.email.ses.secret_access_key") {
			appSettings.Overrides.Email["ses.secret_access_key"] = true
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
		"app.auth.signup_enabled":             auth.SignupEnabled,
		"app.auth.magic_link_enabled":         auth.MagicLinkEnabled,
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
	// Helper to update a setting with env override check
	setSetting := func(key string, value interface{}) error {
		if h.settingsCache != nil && h.settingsCache.IsOverriddenByEnv(key) {
			return fiber.NewError(fiber.StatusConflict,
				fmt.Sprintf("Setting '%s' cannot be updated because it is overridden by an environment variable", key))
		}
		return h.settingsService.SetSetting(ctx, key, map[string]interface{}{"value": value}, "")
	}

	// Basic email settings (always update)
	if err := setSetting("app.email.enabled", email.Enabled); err != nil {
		return err
	}
	if err := setSetting("app.email.provider", email.Provider); err != nil {
		return err
	}
	if err := setSetting("app.email.from_address", email.FromAddress); err != nil {
		return err
	}
	if err := setSetting("app.email.from_name", email.FromName); err != nil {
		return err
	}
	if err := setSetting("app.email.reply_to_address", email.ReplyToAddress); err != nil {
		return err
	}

	// SMTP settings
	if email.SMTP != nil {
		if err := setSetting("app.email.smtp.host", email.SMTP.Host); err != nil {
			return err
		}
		if err := setSetting("app.email.smtp.port", email.SMTP.Port); err != nil {
			return err
		}
		if err := setSetting("app.email.smtp.username", email.SMTP.Username); err != nil {
			return err
		}
		if err := setSetting("app.email.smtp.tls", email.SMTP.TLS); err != nil {
			return err
		}
		// Only update password if provided (non-empty) - preserves existing password
		if email.SMTP.Password != "" {
			if err := setSetting("app.email.smtp.password", email.SMTP.Password); err != nil {
				return err
			}
		}
	}

	// SendGrid settings
	if email.SendGrid != nil {
		// Only update API key if provided (non-empty) - preserves existing key
		if email.SendGrid.APIKey != "" {
			if err := setSetting("app.email.sendgrid.api_key", email.SendGrid.APIKey); err != nil {
				return err
			}
		}
	}

	// Mailgun settings
	if email.Mailgun != nil {
		if err := setSetting("app.email.mailgun.domain", email.Mailgun.Domain); err != nil {
			return err
		}
		if err := setSetting("app.email.mailgun.eu_region", email.Mailgun.EURegion); err != nil {
			return err
		}
		// Only update API key if provided (non-empty) - preserves existing key
		if email.Mailgun.APIKey != "" {
			if err := setSetting("app.email.mailgun.api_key", email.Mailgun.APIKey); err != nil {
				return err
			}
		}
	}

	// SES settings
	if email.SES != nil {
		if err := setSetting("app.email.ses.region", email.SES.Region); err != nil {
			return err
		}
		// Only update credentials if provided (non-empty) - preserves existing credentials
		if email.SES.AccessKeyID != "" {
			if err := setSetting("app.email.ses.access_key_id", email.SES.AccessKeyID); err != nil {
				return err
			}
		}
		if email.SES.SecretAccessKey != "" {
			if err := setSetting("app.email.ses.secret_access_key", email.SES.SecretAccessKey); err != nil {
				return err
			}
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
