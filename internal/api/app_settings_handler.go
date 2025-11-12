package api

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
	"github.com/wayli-app/fluxbase/internal/auth"
)

// AppSettingsHandler handles application settings operations
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
		case "app.features.enable_realtime":
			if val, ok := setting.Value["value"].(bool); ok {
				appSettings.Features.EnableRealtime = val
			}
		case "app.features.enable_storage":
			if val, ok := setting.Value["value"].(bool); ok {
				appSettings.Features.EnableStorage = val
			}
		case "app.features.enable_functions":
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
		if err := h.settingsService.SetSetting(ctx, key, map[string]interface{}{"value": value}, ""); err != nil {
			return err
		}
	}

	return nil
}

func (h *AppSettingsHandler) updateFeatureSettings(ctx context.Context, features *FeatureSettings) error {
	settingsMap := map[string]interface{}{
		"app.features.enable_realtime":  features.EnableRealtime,
		"app.features.enable_storage":   features.EnableStorage,
		"app.features.enable_functions": features.EnableFunctions,
	}

	for key, value := range settingsMap {
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
		if err := h.settingsService.SetSetting(ctx, key, map[string]interface{}{"value": value}, ""); err != nil {
			return err
		}
	}

	return nil
}

func (h *AppSettingsHandler) updateSecuritySettings(ctx context.Context, security *SecuritySettings) error {
	return h.settingsService.SetSetting(
		ctx,
		"app.security.enable_global_rate_limit",
		map[string]interface{}{"value": security.EnableGlobalRateLimit},
		"",
	)
}
