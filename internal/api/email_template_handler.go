package api

import (
	"bytes"
	"context"
	"html/template"
	"time"

	"github.com/fluxbase-eu/fluxbase/internal/database"
	"github.com/fluxbase-eu/fluxbase/internal/email"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"
)

// EmailTemplateHandler handles email template management
type EmailTemplateHandler struct {
	db           *database.Connection
	emailService email.Service
}

// NewEmailTemplateHandler creates a new email template handler
func NewEmailTemplateHandler(db *database.Connection, emailService email.Service) *EmailTemplateHandler {
	return &EmailTemplateHandler{
		db:           db,
		emailService: emailService,
	}
}

// EmailTemplate represents an email template
type EmailTemplate struct {
	ID           uuid.UUID `json:"id"`
	TemplateType string    `json:"template_type"`
	Subject      string    `json:"subject"`
	HTMLBody     string    `json:"html_body"`
	TextBody     *string   `json:"text_body,omitempty"`
	IsCustom     bool      `json:"is_custom"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// UpdateTemplateRequest represents the request to update a template
type UpdateTemplateRequest struct {
	Subject  string  `json:"subject"`
	HTMLBody string  `json:"html_body"`
	TextBody *string `json:"text_body,omitempty"`
}

// TestEmailRequest represents a request to send a test email
type TestEmailRequest struct {
	RecipientEmail string `json:"recipient_email"`
}

// Default email templates
var defaultTemplates = map[string]EmailTemplate{
	"magic_link": {
		TemplateType: "magic_link",
		Subject:      "Your Magic Link - Sign in to {{.AppName}}",
		HTMLBody: `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333; max-width: 600px; margin: 0 auto; padding: 20px;">
    <div style="background-color: #f4f4f4; padding: 20px; border-radius: 5px;">
        <h1 style="color: #2c3e50; margin-bottom: 20px;">Sign in to {{.AppName}}</h1>
        <p>Click the button below to sign in to your account. This link will expire in 15 minutes.</p>
        <div style="text-align: center; margin: 30px 0;">
            <a href="{{.MagicLink}}" style="background-color: #3498db; color: white; padding: 12px 30px; text-decoration: none; border-radius: 5px; display: inline-block;">Sign In</a>
        </div>
        <p style="color: #7f8c8d; font-size: 14px;">If you didn't request this email, you can safely ignore it.</p>
        <p style="color: #7f8c8d; font-size: 14px;">If the button doesn't work, copy and paste this link into your browser:</p>
        <p style="word-break: break-all; color: #3498db; font-size: 12px;">{{.MagicLink}}</p>
    </div>
</body>
</html>`,
		TextBody: stringPtr(`Sign in to {{.AppName}}

Click the link below to sign in to your account. This link will expire in 15 minutes.

{{.MagicLink}}

If you didn't request this email, you can safely ignore it.`),
		IsCustom: false,
	},
	"email_verification": {
		TemplateType: "email_verification",
		Subject:      "Verify Your Email - {{.AppName}}",
		HTMLBody: `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333; max-width: 600px; margin: 0 auto; padding: 20px;">
    <div style="background-color: #f4f4f4; padding: 20px; border-radius: 5px;">
        <h1 style="color: #2c3e50; margin-bottom: 20px;">Verify Your Email</h1>
        <p>Thank you for signing up for {{.AppName}}! Please verify your email address by clicking the button below.</p>
        <div style="text-align: center; margin: 30px 0;">
            <a href="{{.VerificationLink}}" style="background-color: #27ae60; color: white; padding: 12px 30px; text-decoration: none; border-radius: 5px; display: inline-block;">Verify Email</a>
        </div>
        <p style="color: #7f8c8d; font-size: 14px;">This link will expire in 24 hours.</p>
        <p style="color: #7f8c8d; font-size: 14px;">If you didn't create an account, you can safely ignore this email.</p>
        <p style="color: #7f8c8d; font-size: 14px;">If the button doesn't work, copy and paste this link into your browser:</p>
        <p style="word-break: break-all; color: #3498db; font-size: 12px;">{{.VerificationLink}}</p>
    </div>
</body>
</html>`,
		TextBody: stringPtr(`Verify Your Email

Thank you for signing up for {{.AppName}}! Please verify your email address by clicking the link below.

{{.VerificationLink}}

This link will expire in 24 hours.

If you didn't create an account, you can safely ignore this email.`),
		IsCustom: false,
	},
	"password_reset": {
		TemplateType: "password_reset",
		Subject:      "Reset Your Password - {{.AppName}}",
		HTMLBody: `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333; max-width: 600px; margin: 0 auto; padding: 20px;">
    <div style="background-color: #f4f4f4; padding: 20px; border-radius: 5px;">
        <h1 style="color: #2c3e50; margin-bottom: 20px;">Reset Your Password</h1>
        <p>We received a request to reset your password for {{.AppName}}. Click the button below to create a new password.</p>
        <div style="text-align: center; margin: 30px 0;">
            <a href="{{.ResetLink}}" style="background-color: #e74c3c; color: white; padding: 12px 30px; text-decoration: none; border-radius: 5px; display: inline-block;">Reset Password</a>
        </div>
        <p style="color: #7f8c8d; font-size: 14px;">This link will expire in 1 hour.</p>
        <p style="color: #7f8c8d; font-size: 14px;">If you didn't request a password reset, you can safely ignore this email. Your password will not be changed.</p>
        <p style="color: #7f8c8d; font-size: 14px;">If the button doesn't work, copy and paste this link into your browser:</p>
        <p style="word-break: break-all; color: #3498db; font-size: 12px;">{{.ResetLink}}</p>
    </div>
</body>
</html>`,
		TextBody: stringPtr(`Reset Your Password

We received a request to reset your password for {{.AppName}}. Click the link below to create a new password.

{{.ResetLink}}

This link will expire in 1 hour.

If you didn't request a password reset, you can safely ignore this email. Your password will not be changed.`),
		IsCustom: false,
	},
}

// ListTemplates returns all email templates
// GET /api/v1/admin/email/templates
func (h *EmailTemplateHandler) ListTemplates(c *fiber.Ctx) error {
	ctx := context.Background()

	rows, err := h.db.Query(ctx, `
		SELECT id, template_type, subject, html_body, text_body, is_custom, created_at, updated_at
		FROM dashboard.email_templates
		ORDER BY template_type
	`)
	if err != nil {
		log.Error().Err(err).Msg("Failed to list email templates")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve email templates",
		})
	}
	defer rows.Close()

	var templates []EmailTemplate
	existingTypes := make(map[string]bool)

	for rows.Next() {
		var template EmailTemplate
		err := rows.Scan(
			&template.ID,
			&template.TemplateType,
			&template.Subject,
			&template.HTMLBody,
			&template.TextBody,
			&template.IsCustom,
			&template.CreatedAt,
			&template.UpdatedAt,
		)
		if err != nil {
			log.Error().Err(err).Msg("Failed to scan email template")
			continue
		}
		templates = append(templates, template)
		existingTypes[template.TemplateType] = true
	}

	// Add default templates for types that don't exist in the database
	for templateType, defaultTemplate := range defaultTemplates {
		if !existingTypes[templateType] {
			templates = append(templates, defaultTemplate)
		}
	}

	return c.JSON(templates)
}

// GetTemplate returns a specific email template by type
// GET /api/v1/admin/email/templates/:type
func (h *EmailTemplateHandler) GetTemplate(c *fiber.Ctx) error {
	ctx := context.Background()
	templateType := c.Params("type")

	// Validate template type
	if _, exists := defaultTemplates[templateType]; !exists {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid template type",
		})
	}

	var template EmailTemplate
	err := h.db.QueryRow(ctx, `
		SELECT id, template_type, subject, html_body, text_body, is_custom, created_at, updated_at
		FROM dashboard.email_templates
		WHERE template_type = $1
	`, templateType).Scan(
		&template.ID,
		&template.TemplateType,
		&template.Subject,
		&template.HTMLBody,
		&template.TextBody,
		&template.IsCustom,
		&template.CreatedAt,
		&template.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			// Return default template
			defaultTemplate, exists := defaultTemplates[templateType]
			if !exists {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
					"error": "Template not found",
				})
			}
			return c.JSON(defaultTemplate)
		}
		log.Error().Err(err).Str("type", templateType).Msg("Failed to get email template")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve email template",
		})
	}

	return c.JSON(template)
}

// UpdateTemplate updates or creates an email template
// PUT /api/v1/admin/email/templates/:type
func (h *EmailTemplateHandler) UpdateTemplate(c *fiber.Ctx) error {
	ctx := context.Background()
	templateType := c.Params("type")

	// Validate template type
	if _, exists := defaultTemplates[templateType]; !exists {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid template type",
		})
	}

	var req UpdateTemplateRequest
	if err := c.BodyParser(&req); err != nil {
		log.Error().Err(err).Msg("Failed to parse update template request")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate required fields
	if req.Subject == "" || req.HTMLBody == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Subject and HTML body are required",
		})
	}

	// Insert or update template
	var templateID uuid.UUID
	err := h.db.QueryRow(ctx, `
		INSERT INTO dashboard.email_templates (template_type, subject, html_body, text_body, is_custom)
		VALUES ($1, $2, $3, $4, true)
		ON CONFLICT (template_type) DO UPDATE
		SET subject = EXCLUDED.subject,
		    html_body = EXCLUDED.html_body,
		    text_body = EXCLUDED.text_body,
		    is_custom = true,
		    updated_at = NOW()
		RETURNING id
	`, templateType, req.Subject, req.HTMLBody, req.TextBody).Scan(&templateID)

	if err != nil {
		log.Error().Err(err).Str("type", templateType).Msg("Failed to update email template")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update email template",
		})
	}

	log.Info().Str("type", templateType).Str("id", templateID.String()).Msg("Email template updated")

	// Fetch and return the updated template
	return h.GetTemplate(c)
}

// ResetTemplate resets an email template to its default
// POST /api/v1/admin/email/templates/:type/reset
func (h *EmailTemplateHandler) ResetTemplate(c *fiber.Ctx) error {
	ctx := context.Background()
	templateType := c.Params("type")

	// Validate template type
	defaultTemplate, exists := defaultTemplates[templateType]
	if !exists {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid template type",
		})
	}

	// Delete custom template to fall back to default
	_, err := h.db.Exec(ctx, `
		DELETE FROM dashboard.email_templates
		WHERE template_type = $1
	`, templateType)

	if err != nil {
		log.Error().Err(err).Str("type", templateType).Msg("Failed to reset email template")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to reset email template",
		})
	}

	log.Info().Str("type", templateType).Msg("Email template reset to default")

	return c.JSON(defaultTemplate)
}

// TestTemplate sends a test email using the specified template
// POST /api/v1/admin/email/templates/:type/test
func (h *EmailTemplateHandler) TestTemplate(c *fiber.Ctx) error {
	templateType := c.Params("type")

	// Validate template type
	if _, exists := defaultTemplates[templateType]; !exists {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid template type",
		})
	}

	var req TestEmailRequest
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

	// Check if email service is available
	if h.emailService == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "Email service not configured",
		})
	}

	ctx := context.Background()

	// Get template (custom or default)
	var emailTemplate EmailTemplate
	err := h.db.QueryRow(ctx, `
		SELECT id, template_type, subject, html_body, text_body, is_custom, created_at, updated_at
		FROM dashboard.email_templates
		WHERE template_type = $1
	`, templateType).Scan(
		&emailTemplate.ID,
		&emailTemplate.TemplateType,
		&emailTemplate.Subject,
		&emailTemplate.HTMLBody,
		&emailTemplate.TextBody,
		&emailTemplate.IsCustom,
		&emailTemplate.CreatedAt,
		&emailTemplate.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			// Use default template
			emailTemplate = defaultTemplates[templateType]
		} else {
			log.Error().Err(err).Str("type", templateType).Msg("Failed to get email template")
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to get email template",
			})
		}
	}

	// Create test data for rendering
	testData := map[string]string{
		"AppName":     "Test Application",
		"Link":        "https://example.com/test-link",
		"Token":       "test-token-12345",
		"MagicLink":   "https://example.com/magic-link/test-token",
		"ResetLink":   "https://example.com/reset/test-token",
		"VerifyLink":  "https://example.com/verify/test-token",
		"InviteLink":  "https://example.com/invite/test-token",
		"InviterName": "Test Admin",
		"Expiry":      "15 minutes",
	}

	// Render template with test data
	renderedSubject := renderTemplateString(emailTemplate.Subject, testData)
	renderedBody := renderTemplateString(emailTemplate.HTMLBody, testData)

	// Send test email
	if err := h.emailService.Send(ctx, req.RecipientEmail, renderedSubject, renderedBody); err != nil {
		log.Error().Err(err).
			Str("type", templateType).
			Str("recipient", req.RecipientEmail).
			Msg("Failed to send test email")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to send test email: " + err.Error(),
		})
	}

	log.Info().
		Str("type", templateType).
		Str("recipient", req.RecipientEmail).
		Msg("Test email sent successfully")

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Test email sent successfully",
	})
}

// renderTemplateString renders a template string with the given data
func renderTemplateString(templateStr string, data map[string]string) string {
	tmpl, err := template.New("email").Parse(templateStr)
	if err != nil {
		return templateStr // Return original if parsing fails
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return templateStr // Return original if execution fails
	}

	return buf.String()
}

// Helper function to create string pointers
func stringPtr(s string) *string {
	return &s
}
