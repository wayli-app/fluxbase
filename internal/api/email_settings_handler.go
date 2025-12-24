package api

import (
	"context"

	"github.com/fluxbase-eu/fluxbase/internal/auth"
	"github.com/fluxbase-eu/fluxbase/internal/config"
	"github.com/fluxbase-eu/fluxbase/internal/crypto"
	"github.com/fluxbase-eu/fluxbase/internal/email"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
)

// EmailSettingsHandler handles email configuration management
type EmailSettingsHandler struct {
	settingsService *auth.SystemSettingsService
	settingsCache   *auth.SettingsCache
	emailManager    *email.Manager
	encryptionKey   string
	envConfig       *config.EmailConfig // Fallback config from environment
}

// NewEmailSettingsHandler creates a new email settings handler
func NewEmailSettingsHandler(
	settingsService *auth.SystemSettingsService,
	settingsCache *auth.SettingsCache,
	emailManager *email.Manager,
	encryptionKey string,
	envConfig *config.EmailConfig,
) *EmailSettingsHandler {
	return &EmailSettingsHandler{
		settingsService: settingsService,
		settingsCache:   settingsCache,
		emailManager:    emailManager,
		encryptionKey:   encryptionKey,
		envConfig:       envConfig,
	}
}

// EmailSettingsResponse represents the email settings returned to the UI
type EmailSettingsResponse struct {
	Enabled     bool   `json:"enabled"`
	Provider    string `json:"provider"`
	FromAddress string `json:"from_address"`
	FromName    string `json:"from_name"`

	// SMTP
	SMTPHost        string `json:"smtp_host"`
	SMTPPort        int    `json:"smtp_port"`
	SMTPUsername    string `json:"smtp_username"`
	SMTPPasswordSet bool   `json:"smtp_password_set"` // true if password is configured
	SMTPTLS         bool   `json:"smtp_tls"`

	// SendGrid
	SendGridAPIKeySet bool `json:"sendgrid_api_key_set"`

	// Mailgun
	MailgunAPIKeySet bool   `json:"mailgun_api_key_set"`
	MailgunDomain    string `json:"mailgun_domain"`

	// AWS SES
	SESAccessKeySet bool   `json:"ses_access_key_set"`
	SESSecretKeySet bool   `json:"ses_secret_key_set"`
	SESRegion       string `json:"ses_region"`

	// Override information
	Overrides map[string]OverrideInfo `json:"_overrides"`
}

// OverrideInfo indicates if a setting is overridden by environment variable
type OverrideInfo struct {
	IsOverridden bool   `json:"is_overridden"`
	EnvVar       string `json:"env_var,omitempty"`
}

// UpdateEmailSettingsRequest represents the request to update email settings
type UpdateEmailSettingsRequest struct {
	Enabled     *bool   `json:"enabled,omitempty"`
	Provider    *string `json:"provider,omitempty"`
	FromAddress *string `json:"from_address,omitempty"`
	FromName    *string `json:"from_name,omitempty"`

	// SMTP
	SMTPHost     *string `json:"smtp_host,omitempty"`
	SMTPPort     *int    `json:"smtp_port,omitempty"`
	SMTPUsername *string `json:"smtp_username,omitempty"`
	SMTPPassword *string `json:"smtp_password,omitempty"` // Only set if changing
	SMTPTLS      *bool   `json:"smtp_tls,omitempty"`

	// SendGrid
	SendGridAPIKey *string `json:"sendgrid_api_key,omitempty"`

	// Mailgun
	MailgunAPIKey *string `json:"mailgun_api_key,omitempty"`
	MailgunDomain *string `json:"mailgun_domain,omitempty"`

	// AWS SES
	SESAccessKey *string `json:"ses_access_key,omitempty"`
	SESSecretKey *string `json:"ses_secret_key,omitempty"`
	SESRegion    *string `json:"ses_region,omitempty"`
}

// TestEmailSettingsRequest represents a test email request
type TestEmailSettingsRequest struct {
	RecipientEmail string `json:"recipient_email"`
}

// GetSettings returns the current email settings
// GET /api/v1/admin/email/settings
func (h *EmailSettingsHandler) GetSettings(c *fiber.Ctx) error {
	ctx := context.Background()

	response := EmailSettingsResponse{
		Overrides: make(map[string]OverrideInfo),
	}

	// Helper to get string value with override check
	getString := func(key, defaultVal string) (string, bool) {
		if h.settingsCache != nil && h.settingsCache.IsOverriddenByEnv(key) {
			return h.settingsCache.GetString(ctx, key, defaultVal), true
		}
		return h.settingsCache.GetString(ctx, key, defaultVal), false
	}

	// Helper to get int value with override check
	getInt := func(key string, defaultVal int) (int, bool) {
		if h.settingsCache != nil && h.settingsCache.IsOverriddenByEnv(key) {
			return h.settingsCache.GetInt(ctx, key, defaultVal), true
		}
		return h.settingsCache.GetInt(ctx, key, defaultVal), false
	}

	// Helper to get bool value with override check
	getBool := func(key string, defaultVal bool) (bool, bool) {
		if h.settingsCache != nil && h.settingsCache.IsOverriddenByEnv(key) {
			return h.settingsCache.GetBool(ctx, key, defaultVal), true
		}
		return h.settingsCache.GetBool(ctx, key, defaultVal), false
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
	response.Enabled, _ = getBool("app.email.enabled", false)
	addOverride("enabled", "app.email.enabled")

	response.Provider, _ = getString("app.email.provider", "smtp")
	addOverride("provider", "app.email.provider")

	response.FromAddress, _ = getString("app.email.from_address", "")
	addOverride("from_address", "app.email.from_address")

	response.FromName, _ = getString("app.email.from_name", "")
	addOverride("from_name", "app.email.from_name")

	// SMTP settings
	response.SMTPHost, _ = getString("app.email.smtp_host", "")
	addOverride("smtp_host", "app.email.smtp_host")

	response.SMTPPort, _ = getInt("app.email.smtp_port", 587)
	addOverride("smtp_port", "app.email.smtp_port")

	response.SMTPUsername, _ = getString("app.email.smtp_username", "")
	addOverride("smtp_username", "app.email.smtp_username")

	response.SMTPTLS, _ = getBool("app.email.smtp_tls", true)
	addOverride("smtp_tls", "app.email.smtp_tls")

	// Check if password is set (don't return the actual value)
	smtpPassword, _ := getString("app.email.smtp_password", "")
	response.SMTPPasswordSet = smtpPassword != ""
	addOverride("smtp_password", "app.email.smtp_password")

	// SendGrid
	sendgridKey, _ := getString("app.email.sendgrid_api_key", "")
	response.SendGridAPIKeySet = sendgridKey != ""
	addOverride("sendgrid_api_key", "app.email.sendgrid_api_key")

	// Mailgun
	mailgunKey, _ := getString("app.email.mailgun_api_key", "")
	response.MailgunAPIKeySet = mailgunKey != ""
	addOverride("mailgun_api_key", "app.email.mailgun_api_key")

	response.MailgunDomain, _ = getString("app.email.mailgun_domain", "")
	addOverride("mailgun_domain", "app.email.mailgun_domain")

	// AWS SES
	sesAccessKey, _ := getString("app.email.ses_access_key", "")
	response.SESAccessKeySet = sesAccessKey != ""
	addOverride("ses_access_key", "app.email.ses_access_key")

	sesSecretKey, _ := getString("app.email.ses_secret_key", "")
	response.SESSecretKeySet = sesSecretKey != ""
	addOverride("ses_secret_key", "app.email.ses_secret_key")

	response.SESRegion, _ = getString("app.email.ses_region", "us-east-1")
	addOverride("ses_region", "app.email.ses_region")

	return c.JSON(response)
}

// UpdateSettings updates email settings
// PUT /api/v1/admin/email/settings
func (h *EmailSettingsHandler) UpdateSettings(c *fiber.Ctx) error {
	ctx := context.Background()

	var req UpdateEmailSettingsRequest
	if err := c.BodyParser(&req); err != nil {
		log.Error().Err(err).Msg("Failed to parse update email settings request")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Track which settings were updated
	var updatedKeys []string

	// Helper to update a setting with override check
	updateSetting := func(key string, value interface{}) error {
		// Check if overridden by env var
		if h.settingsCache != nil && h.settingsCache.IsOverriddenByEnv(key) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": "This setting is controlled by an environment variable and cannot be changed",
				"code":  "ENV_OVERRIDE",
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

	// Helper to encrypt and update a secret
	updateSecret := func(key string, value *string) error {
		if value == nil {
			return nil // Not updating this field
		}

		// Check if overridden by env var
		if h.settingsCache != nil && h.settingsCache.IsOverriddenByEnv(key) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": "This setting is controlled by an environment variable and cannot be changed",
				"code":  "ENV_OVERRIDE",
				"key":   key,
			})
		}

		// Encrypt the value if encryption key is available
		storedValue := *value
		if h.encryptionKey != "" && *value != "" {
			encrypted, err := crypto.Encrypt(*value, h.encryptionKey)
			if err != nil {
				log.Error().Err(err).Str("key", key).Msg("Failed to encrypt secret")
				return err
			}
			storedValue = encrypted
		}

		if err := h.settingsService.SetSetting(ctx, key, map[string]interface{}{"value": storedValue}, ""); err != nil {
			log.Error().Err(err).Str("key", key).Msg("Failed to update secret")
			return err
		}
		updatedKeys = append(updatedKeys, key)
		return nil
	}

	// Update basic settings
	if req.Enabled != nil {
		if err := updateSetting("app.email.enabled", *req.Enabled); err != nil {
			return err
		}
	}

	if req.Provider != nil {
		if err := updateSetting("app.email.provider", *req.Provider); err != nil {
			return err
		}
	}

	if req.FromAddress != nil {
		if err := updateSetting("app.email.from_address", *req.FromAddress); err != nil {
			return err
		}
	}

	if req.FromName != nil {
		if err := updateSetting("app.email.from_name", *req.FromName); err != nil {
			return err
		}
	}

	// SMTP settings
	if req.SMTPHost != nil {
		if err := updateSetting("app.email.smtp_host", *req.SMTPHost); err != nil {
			return err
		}
	}

	if req.SMTPPort != nil {
		if err := updateSetting("app.email.smtp_port", *req.SMTPPort); err != nil {
			return err
		}
	}

	if req.SMTPUsername != nil {
		if err := updateSetting("app.email.smtp_username", *req.SMTPUsername); err != nil {
			return err
		}
	}

	if err := updateSecret("app.email.smtp_password", req.SMTPPassword); err != nil {
		return err
	}

	if req.SMTPTLS != nil {
		if err := updateSetting("app.email.smtp_tls", *req.SMTPTLS); err != nil {
			return err
		}
	}

	// SendGrid
	if err := updateSecret("app.email.sendgrid_api_key", req.SendGridAPIKey); err != nil {
		return err
	}

	// Mailgun
	if err := updateSecret("app.email.mailgun_api_key", req.MailgunAPIKey); err != nil {
		return err
	}

	if req.MailgunDomain != nil {
		if err := updateSetting("app.email.mailgun_domain", *req.MailgunDomain); err != nil {
			return err
		}
	}

	// AWS SES
	if err := updateSecret("app.email.ses_access_key", req.SESAccessKey); err != nil {
		return err
	}

	if err := updateSecret("app.email.ses_secret_key", req.SESSecretKey); err != nil {
		return err
	}

	if req.SESRegion != nil {
		if err := updateSetting("app.email.ses_region", *req.SESRegion); err != nil {
			return err
		}
	}

	// Refresh email service with new settings
	if h.emailManager != nil && len(updatedKeys) > 0 {
		if err := h.emailManager.RefreshFromSettings(ctx); err != nil {
			log.Warn().Err(err).Msg("Failed to refresh email service after settings update")
			// Don't fail the request - settings are saved, service will refresh on next restart
		}
	}

	log.Info().Strs("keys", updatedKeys).Msg("Email settings updated")

	// Return updated settings
	return h.GetSettings(c)
}

// TestSettings sends a test email with current settings
// POST /api/v1/admin/email/settings/test
func (h *EmailSettingsHandler) TestSettings(c *fiber.Ctx) error {
	var req TestEmailSettingsRequest
	if err := c.BodyParser(&req); err != nil {
		log.Error().Err(err).Msg("Failed to parse test email request")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.RecipientEmail == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Recipient email is required",
		})
	}

	// Get current email service
	if h.emailManager == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "Email service not initialized",
		})
	}

	service := h.emailManager.GetService()
	if service == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "Email service not available",
		})
	}

	// Send test email
	ctx := context.Background()
	subject := "Fluxbase Email Configuration Test"
	body := `<html>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333; max-width: 600px; margin: 0 auto; padding: 20px;">
    <div style="background-color: #f4f4f4; padding: 20px; border-radius: 5px;">
        <h1 style="color: #2c3e50; margin-bottom: 20px;">Email Configuration Test</h1>
        <p>This is a test email from Fluxbase to verify your email configuration is working correctly.</p>
        <p style="color: #27ae60; font-weight: bold;">If you received this email, your email settings are configured correctly!</p>
        <hr style="border: none; border-top: 1px solid #ddd; margin: 20px 0;">
        <p style="color: #7f8c8d; font-size: 12px;">This is an automated test email. No action is required.</p>
    </div>
</body>
</html>`

	if err := service.Send(ctx, req.RecipientEmail, subject, body); err != nil {
		log.Error().Err(err).Str("recipient", req.RecipientEmail).Msg("Failed to send test email")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Failed to send test email",
			"details": err.Error(),
		})
	}

	log.Info().Str("recipient", req.RecipientEmail).Msg("Test email sent successfully")

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Test email sent successfully",
	})
}
