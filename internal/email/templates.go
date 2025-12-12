package email

import (
	"bytes"
	"html/template"
	"os"

	"github.com/rs/zerolog/log"
)

// renderMagicLinkHTML renders the magic link email template
func renderMagicLinkHTML(link, token, customTemplatePath string) string {
	data := map[string]string{
		"Link":  link,
		"Token": token,
	}

	// Try custom template if provided
	if customTemplatePath != "" {
		if html := loadAndRenderTemplate(customTemplatePath, data); html != "" {
			return html
		}
	}

	// Use default template
	tmpl := template.Must(template.New("magic-link").Parse(defaultMagicLinkTemplate))
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fallbackMagicLinkHTML(link)
	}
	return buf.String()
}

// renderVerificationHTML renders the email verification template
func renderVerificationHTML(link, token, customTemplatePath string) string {
	data := map[string]string{
		"Link":  link,
		"Token": token,
	}

	// Try custom template if provided
	if customTemplatePath != "" {
		if html := loadAndRenderTemplate(customTemplatePath, data); html != "" {
			return html
		}
	}

	// Use default template
	tmpl := template.Must(template.New("verification").Parse(defaultVerificationTemplate))
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fallbackVerificationHTML(link)
	}
	return buf.String()
}

// renderPasswordResetHTML renders the password reset email template
func renderPasswordResetHTML(link, token, customTemplatePath string) string {
	data := map[string]string{
		"Link":  link,
		"Token": token,
	}

	// Try custom template if provided
	if customTemplatePath != "" {
		if html := loadAndRenderTemplate(customTemplatePath, data); html != "" {
			return html
		}
	}

	// Use default template
	tmpl := template.Must(template.New("password-reset").Parse(defaultPasswordResetTemplate))
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fallbackPasswordResetHTML(link)
	}
	return buf.String()
}

// loadAndRenderTemplate loads and renders a custom template from file
func loadAndRenderTemplate(templatePath string, data map[string]string) string {
	// Read template file
	templateBytes, err := os.ReadFile(templatePath)
	if err != nil {
		log.Warn().
			Err(err).
			Str("template_path", templatePath).
			Msg("Failed to read custom email template, using default")
		return ""
	}

	// Parse and execute template
	tmpl, err := template.New("custom").Parse(string(templateBytes))
	if err != nil {
		log.Warn().
			Err(err).
			Str("template_path", templatePath).
			Msg("Failed to parse custom email template, using default")
		return ""
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		log.Warn().
			Err(err).
			Str("template_path", templatePath).
			Msg("Failed to execute custom email template, using default")
		return ""
	}

	return buf.String()
}

// renderInvitationHTML renders the invitation email template
func renderInvitationHTML(inviterName, inviteLink string) string {
	data := map[string]string{
		"InviterName": inviterName,
		"InviteLink":  inviteLink,
	}

	// Use default template
	tmpl := template.Must(template.New("invitation").Parse(defaultInvitationTemplate))
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fallbackInvitationHTML(inviteLink)
	}
	return buf.String()
}

// Fallback HTML templates (simple versions)
func fallbackMagicLinkHTML(link string) string {
	return `<html><body><h2>Your Login Link</h2><p>Click the link below to log in:</p><p><a href="` + link + `">Log In</a></p><p>This link will expire soon</p></body></html>`
}

func fallbackVerificationHTML(link string) string {
	return `<html><body><h2>Verify Your Email</h2><p>Click the link below to verify your email:</p><p><a href="` + link + `">Verify Email</a></p></body></html>`
}

func fallbackPasswordResetHTML(link string) string {
	return `<html><body><h2>Reset Your Password</h2><p>Click the link below to reset your password:</p><p><a href="` + link + `">Reset Password</a></p><p>This link will expire soon</p></body></html>`
}

func fallbackInvitationHTML(link string) string {
	return `<html><body><h2>You've Been Invited!</h2><p>Click the link below to accept your invitation:</p><p><a href="` + link + `">Accept Invitation</a></p><p>This invitation expires in 7 days</p></body></html>`
}
