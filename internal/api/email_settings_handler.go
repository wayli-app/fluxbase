package api

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/auth"
	"github.com/nimbleflux/fluxbase/internal/config"
	"github.com/nimbleflux/fluxbase/internal/email"
	apperrors "github.com/nimbleflux/fluxbase/internal/errors"
	"github.com/nimbleflux/fluxbase/internal/settings"
)

// EmailSettingsHandler handles email configuration management
type EmailSettingsHandler struct {
	settingsService *auth.SystemSettingsService
	settingsCache   *auth.SettingsCache
	emailManager    *email.Manager
	secretsService  *settings.SecretsService
	config          *config.Config // Full config for tenant resolution
	unifiedService  *settings.UnifiedService
}

// NewEmailSettingsHandler creates a new email settings handler
func NewEmailSettingsHandler(
	settingsService *auth.SystemSettingsService,
	settingsCache *auth.SettingsCache,
	emailManager *email.Manager,
	secretsService *settings.SecretsService,
	cfg *config.Config,
	unifiedService *settings.UnifiedService,
) *EmailSettingsHandler {
	return &EmailSettingsHandler{
		settingsService: settingsService,
		settingsCache:   settingsCache,
		emailManager:    emailManager,
		secretsService:  secretsService,
		config:          cfg,
		unifiedService:  unifiedService,
	}
}

func (h *EmailSettingsHandler) requireService(c fiber.Ctx) error {
	if h.settingsService == nil || h.settingsCache == nil {
		return fiber.NewError(fiber.StatusServiceUnavailable, "Email settings service not initialized")
	}
	return nil
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
func (h *EmailSettingsHandler) GetSettings(c fiber.Ctx) error {
	ctx := context.Background()

	if err := h.requireService(c); err != nil {
		return err
	}

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
func (h *EmailSettingsHandler) UpdateSettings(c fiber.Ctx) error {
	ctx := context.Background()

	var req UpdateEmailSettingsRequest
	if err := ParseBody(c, &req); err != nil {
		return err
	}

	if err := h.requireService(c); err != nil {
		return err
	}

	var updatedKeys []string

	// Helper to update a setting with override check
	updateSetting := func(key string, value interface{}) error {
		// Check if overridden by env var
		if h.settingsCache != nil && h.settingsCache.IsOverriddenByEnv(key) {
			return SendConflict(c, "This setting is controlled by an environment variable and cannot be changed", "ENV_OVERRIDE")
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

		// Check if overridden by env var
		if h.settingsCache != nil && h.settingsCache.IsOverriddenByEnv(key) {
			return SendConflict(c, "This setting is controlled by an environment variable and cannot be changed", "ENV_OVERRIDE")
		}

		// Use SecretsService to encrypt and store the secret
		if h.secretsService != nil && *value != "" {
			if err := h.secretsService.SetSystemSecret(ctx, key, *value, "Email provider secret"); err != nil {
				log.Error().Err(err).Str("key", key).Msg("Failed to store secret")
				return err
			}
		} else if *value == "" {
			// Clear the secret by deleting it
			if h.secretsService != nil {
				_ = h.secretsService.DeleteSystemSecret(ctx, key) // Ignore not found errors
			}
		}

		// Invalidate settings cache so GetSettings reflects the change
		if h.settingsCache != nil {
			h.settingsCache.Invalidate(key)
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
func (h *EmailSettingsHandler) TestSettings(c fiber.Ctx) error {
	var req TestEmailSettingsRequest
	if err := ParseBody(c, &req); err != nil {
		return err
	}

	if req.RecipientEmail == "" {
		return SendBadRequest(c, "Recipient email is required", ErrCodeMissingField)
	}

	// Get current email service
	if h.emailManager == nil {
		return SendErrorWithCode(c, 503, "Email service not initialized", ErrCodeFeatureDisabled)
	}

	service := h.emailManager.GetService()
	if service == nil {
		return SendErrorWithCode(c, 503, "Email service not available", ErrCodeFeatureDisabled)
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
		return SendInternalError(c, "Failed to send test email")
	}

	log.Info().Str("recipient", req.RecipientEmail).Msg("Test email sent successfully")

	return apperrors.SendSuccess(c, "Test email sent successfully")
}

// TenantEmailSettingsResponse extends EmailSettingsResponse with source information per field.
type TenantEmailSettingsResponse struct {
	EmailSettingsResponse
	Sources map[string]string `json:"_sources"` // field -> "instance" | "tenant" | "config" | "default"
}

// GetSettingsForTenant returns email settings resolved through the cascade for a specific tenant.
// GET /api/v1/admin/email/settings/tenant
func (h *EmailSettingsHandler) GetSettingsForTenant(c fiber.Ctx) error {
	if h.unifiedService == nil {
		// Fall back to instance-level settings
		return h.GetSettings(c)
	}

	tenantID := GetTenantID(c)
	if tenantID == "" {
		return SendBadRequest(c, "tenant context required", ErrCodeMissingField)
	}

	isDefaultTenant, _ := c.Locals("is_default_tenant").(bool)
	tenantSlug, _ := c.Locals("tenant_slug").(string)

	ctx := context.Background()
	response := TenantEmailSettingsResponse{
		EmailSettingsResponse: EmailSettingsResponse{
			Overrides: make(map[string]OverrideInfo),
		},
		Sources: make(map[string]string),
	}

	// Resolve each email setting through the unified cascade
	resolveBool := func(path string, defaultVal bool) (bool, string) {
		resolved, err := h.unifiedService.ResolveSettingWithDefault(ctx, tenantID, "email."+path, defaultVal, isDefaultTenant, tenantSlug)
		if err != nil {
			return defaultVal, "default"
		}
		val, _ := resolved.Value.(bool)
		return val, resolved.Source
	}

	resolveString := func(path, defaultVal string) (string, string) {
		resolved, err := h.unifiedService.ResolveSettingWithDefault(ctx, tenantID, "email."+path, defaultVal, isDefaultTenant, tenantSlug)
		if err != nil {
			return defaultVal, "default"
		}
		if val, ok := resolved.Value.(string); ok {
			return val, resolved.Source
		}
		return defaultVal, resolved.Source
	}

	resolveInt := func(path string, defaultVal int) (int, string) {
		resolved, err := h.unifiedService.ResolveSettingWithDefault(ctx, tenantID, "email."+path, defaultVal, isDefaultTenant, tenantSlug)
		if err != nil {
			return defaultVal, "default"
		}
		switch v := resolved.Value.(type) {
		case int:
			return v, resolved.Source
		case float64:
			return int(v), resolved.Source
		case json.Number:
			if i, err := v.Int64(); err == nil {
				return int(i), resolved.Source
			}
		}
		return defaultVal, resolved.Source
	}

	// Secret fields: check if a non-empty value exists (don't reveal the actual value)
	resolveSecretSet := func(path string) (bool, string) {
		resolved, err := h.unifiedService.ResolveSetting(ctx, tenantID, "email."+path, isDefaultTenant, tenantSlug)
		if err != nil || resolved.Value == nil {
			return false, "default"
		}
		if str, ok := resolved.Value.(string); ok && str != "" {
			return true, resolved.Source
		}
		return false, resolved.Source
	}

	var src string

	response.Enabled, src = resolveBool("enabled", false)
	response.Sources["enabled"] = src

	response.Provider, src = resolveString("provider", "smtp")
	response.Sources["provider"] = src

	response.FromAddress, src = resolveString("from_address", "")
	response.Sources["from_address"] = src

	response.FromName, src = resolveString("from_name", "")
	response.Sources["from_name"] = src

	response.SMTPHost, src = resolveString("smtp_host", "")
	response.Sources["smtp_host"] = src

	response.SMTPPort, src = resolveInt("smtp_port", 587)
	response.Sources["smtp_port"] = src

	response.SMTPUsername, src = resolveString("smtp_username", "")
	response.Sources["smtp_username"] = src

	response.SMTPPasswordSet, src = resolveSecretSet("smtp_password")
	response.Sources["smtp_password"] = src

	response.SMTPTLS, src = resolveBool("smtp_tls", true)
	response.Sources["smtp_tls"] = src

	response.SendGridAPIKeySet, src = resolveSecretSet("sendgrid_api_key")
	response.Sources["sendgrid_api_key"] = src

	response.MailgunAPIKeySet, src = resolveSecretSet("mailgun_api_key")
	response.Sources["mailgun_api_key"] = src

	response.MailgunDomain, src = resolveString("mailgun_domain", "")
	response.Sources["mailgun_domain"] = src

	response.SESAccessKeySet, src = resolveSecretSet("ses_access_key")
	response.Sources["ses_access_key"] = src

	response.SESSecretKeySet, src = resolveSecretSet("ses_secret_key")
	response.Sources["ses_secret_key"] = src

	response.SESRegion, src = resolveString("ses_region", "us-east-1")
	response.Sources["ses_region"] = src

	return c.JSON(response)
}

// UpdateSettingsForTenant updates email settings for a specific tenant.
// PUT /api/v1/admin/email/settings/tenant
func (h *EmailSettingsHandler) UpdateSettingsForTenant(c fiber.Ctx) error {
	if h.unifiedService == nil {
		return SendInternalError(c, "Unified settings service not available")
	}

	tenantID := GetTenantID(c)
	if tenantID == "" {
		return SendBadRequest(c, "tenant context required", ErrCodeMissingField)
	}

	var req UpdateEmailSettingsRequest
	if err := ParseBody(c, &req); err != nil {
		return err
	}

	ctx := context.Background()
	var updatedKeys []string

	// Helper to update a non-secret setting via UnifiedService
	updateSetting := func(path string, value any) error {
		if err := h.unifiedService.SetTenantSetting(ctx, tenantID, "email."+path, value, false); err != nil {
			if errors.Is(err, settings.ErrNotOverridable) {
				return SendForbidden(c, "This setting cannot be overridden at tenant level", "NOT_OVERRIDABLE")
			}
			log.Error().Err(err).Str("path", path).Msg("Failed to update tenant email setting")
			return err
		}
		updatedKeys = append(updatedKeys, path)
		return nil
	}

	// Helper to update a secret setting via UnifiedService
	updateSecret := func(path string, value *string) error {
		if value == nil {
			return nil
		}
		if err := h.unifiedService.SetTenantSetting(ctx, tenantID, "email."+path, *value, true); err != nil {
			if errors.Is(err, settings.ErrNotOverridable) {
				return SendForbidden(c, "This setting cannot be overridden at tenant level", "NOT_OVERRIDABLE")
			}
			log.Error().Err(err).Str("path", path).Msg("Failed to update tenant email secret")
			return err
		}
		updatedKeys = append(updatedKeys, path)
		return nil
	}

	if req.Enabled != nil {
		if err := updateSetting("enabled", *req.Enabled); err != nil {
			return err
		}
	}
	if req.Provider != nil {
		if err := updateSetting("provider", *req.Provider); err != nil {
			return err
		}
	}
	if req.FromAddress != nil {
		if err := updateSetting("from_address", *req.FromAddress); err != nil {
			return err
		}
	}
	if req.FromName != nil {
		if err := updateSetting("from_name", *req.FromName); err != nil {
			return err
		}
	}
	if req.SMTPHost != nil {
		if err := updateSetting("smtp_host", *req.SMTPHost); err != nil {
			return err
		}
	}
	if req.SMTPPort != nil {
		if err := updateSetting("smtp_port", *req.SMTPPort); err != nil {
			return err
		}
	}
	if req.SMTPUsername != nil {
		if err := updateSetting("smtp_username", *req.SMTPUsername); err != nil {
			return err
		}
	}
	if err := updateSecret("smtp_password", req.SMTPPassword); err != nil {
		return err
	}
	if req.SMTPTLS != nil {
		if err := updateSetting("smtp_tls", *req.SMTPTLS); err != nil {
			return err
		}
	}
	if err := updateSecret("sendgrid_api_key", req.SendGridAPIKey); err != nil {
		return err
	}
	if err := updateSecret("mailgun_api_key", req.MailgunAPIKey); err != nil {
		return err
	}
	if req.MailgunDomain != nil {
		if err := updateSetting("mailgun_domain", *req.MailgunDomain); err != nil {
			return err
		}
	}
	if err := updateSecret("ses_access_key", req.SESAccessKey); err != nil {
		return err
	}
	if err := updateSecret("ses_secret_key", req.SESSecretKey); err != nil {
		return err
	}
	if req.SESRegion != nil {
		if err := updateSetting("ses_region", *req.SESRegion); err != nil {
			return err
		}
	}

	log.Info().Str("tenant_id", tenantID).Strs("keys", updatedKeys).Msg("Tenant email settings updated")

	// Invalidate cache for this tenant's email settings
	h.unifiedService.InvalidateCache(tenantID, "email")

	return h.GetSettingsForTenant(c)
}

// DeleteSettingForTenant removes a tenant-level email setting override, reverting to instance default.
// DELETE /api/v1/admin/email/settings/tenant/:field
func (h *EmailSettingsHandler) DeleteSettingForTenant(c fiber.Ctx) error {
	if h.unifiedService == nil {
		return SendInternalError(c, "Unified settings service not available")
	}

	tenantID := GetTenantID(c)
	if tenantID == "" {
		return SendBadRequest(c, "tenant context required", ErrCodeMissingField)
	}

	field := c.Params("field")
	if field == "" {
		return SendBadRequest(c, "field parameter required", ErrCodeMissingField)
	}

	ctx := context.Background()
	if err := h.unifiedService.DeleteTenantSetting(ctx, tenantID, "email."+field); err != nil {
		log.Error().Err(err).Str("field", field).Msg("Failed to delete tenant email setting")
		return SendInternalError(c, "Failed to delete tenant email setting override")
	}

	h.unifiedService.InvalidateCache(tenantID, "email."+field)

	log.Info().Str("tenant_id", tenantID).Str("field", field).Msg("Tenant email setting override deleted")

	return h.GetSettingsForTenant(c)
}

// TestSettingsForTenant sends a test email using the tenant-resolved email configuration.
// POST /api/v1/admin/email/settings/tenant/test
func (h *EmailSettingsHandler) TestSettingsForTenant(c fiber.Ctx) error {
	tenantID := GetTenantID(c)
	if tenantID == "" {
		return SendBadRequest(c, "tenant context required", ErrCodeMissingField)
	}

	var req TestEmailSettingsRequest
	if err := ParseBody(c, &req); err != nil {
		return err
	}

	if req.RecipientEmail == "" {
		return SendBadRequest(c, "Recipient email is required", ErrCodeMissingField)
	}

	if h.emailManager == nil {
		return SendErrorWithCode(c, 503, "Email service not initialized", ErrCodeFeatureDisabled)
	}

	// Get tenant-resolved email config
	emailCfg := GetEmailConfig(c, h.config)
	if emailCfg == nil || !emailCfg.IsConfigured() {
		return SendErrorWithCode(c, 503, "Email service not configured for this tenant", ErrCodeFeatureDisabled)
	}

	service, err := h.emailManager.GetServiceForConfig(emailCfg)
	if err != nil {
		return SendErrorWithCode(c, 503, "Failed to create email service for tenant", ErrCodeFeatureDisabled)
	}

	ctx := context.Background()
	subject := "Fluxbase Email Configuration Test"
	body := `<html>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333; max-width: 600px; margin: 0 auto; padding: 20px;">
    <div style="background-color: #f4f4f4; padding: 20px; border-radius: 5px;">
        <h1 style="color: #2c3e50; margin-bottom: 20px;">Email Configuration Test</h1>
        <p>This is a test email from Fluxbase to verify your tenant email configuration is working correctly.</p>
        <p style="color: #27ae60; font-weight: bold;">If you received this email, your email settings are configured correctly!</p>
        <hr style="border: none; border-top: 1px solid #ddd; margin: 20px 0;">
        <p style="color: #7f8c8d; font-size: 12px;">This is an automated test email. No action is required.</p>
    </div>
</body>
</html>`

	if err := service.Send(ctx, req.RecipientEmail, subject, body); err != nil {
		log.Error().Err(err).Str("recipient", req.RecipientEmail).Str("tenant_id", tenantID).Msg("Failed to send tenant test email")
		return SendInternalError(c, "Failed to send test email")
	}

	log.Info().Str("recipient", req.RecipientEmail).Str("tenant_id", tenantID).Msg("Tenant test email sent successfully")

	return apperrors.SendSuccess(c, "Test email sent successfully")
}
