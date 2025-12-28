package email

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"net/smtp"

	"github.com/fluxbase-eu/fluxbase/internal/config"
)

// SMTPService handles email sending via SMTP
type SMTPService struct {
	config *config.EmailConfig
}

// NewSMTPService creates a new SMTP email service
func NewSMTPService(cfg *config.EmailConfig) *SMTPService {
	return &SMTPService{config: cfg}
}

// SendMagicLink sends a magic link email
func (s *SMTPService) SendMagicLink(ctx context.Context, to, token, link string) error {
	subject := "Your login link"
	body := s.renderMagicLinkTemplate(link, token)

	return s.Send(ctx, to, subject, body)
}

// SendVerificationEmail sends an email verification link
func (s *SMTPService) SendVerificationEmail(ctx context.Context, to, token, link string) error {
	subject := "Verify your email address"
	body := s.renderVerificationTemplate(link, token)

	return s.Send(ctx, to, subject, body)
}

// SendPasswordReset sends a password reset email
func (s *SMTPService) SendPasswordReset(ctx context.Context, to, token, link string) error {
	subject := "Reset your password"
	body := s.renderPasswordResetTemplate(link, token)

	return s.Send(ctx, to, subject, body)
}

// SendInvitationEmail sends an invitation email
func (s *SMTPService) SendInvitationEmail(ctx context.Context, to, inviterName, inviteLink string) error {
	subject := "You've been invited!"
	body := renderInvitationHTML(inviterName, inviteLink)

	return s.Send(ctx, to, subject, body)
}

// Send sends an email via SMTP
func (s *SMTPService) Send(ctx context.Context, to, subject, body string) error {
	if !s.config.Enabled {
		return fmt.Errorf("email service is disabled")
	}

	// Build the message
	message := s.buildMessage(to, subject, body)

	// Set up authentication (only if credentials are provided)
	var auth smtp.Auth
	if s.config.SMTPUsername != "" && s.config.SMTPPassword != "" {
		auth = smtp.PlainAuth(
			"",
			s.config.SMTPUsername,
			s.config.SMTPPassword,
			s.config.SMTPHost,
		)
	}

	// Determine server address
	addr := fmt.Sprintf("%s:%d", s.config.SMTPHost, s.config.SMTPPort)

	// Send email with TLS if enabled
	if s.config.SMTPTLS {
		return s.sendWithTLS(addr, auth, to, message)
	}

	// Send email without TLS
	return smtp.SendMail(addr, auth, s.config.FromAddress, []string{to}, message)
}

// sendWithTLS sends email with STARTTLS
func (s *SMTPService) sendWithTLS(addr string, auth smtp.Auth, to string, message []byte) error {
	// Connect to the server
	client, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server: %w", err)
	}
	defer func() { _ = client.Close() }()

	// Start TLS
	tlsConfig := &tls.Config{
		ServerName: s.config.SMTPHost,
	}
	if err := client.StartTLS(tlsConfig); err != nil {
		return fmt.Errorf("failed to start TLS: %w", err)
	}

	// Authenticate (if auth is provided)
	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("failed to authenticate: %w", err)
		}
	}

	// Set sender
	if err := client.Mail(s.config.FromAddress); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}

	// Set recipient
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("failed to set recipient: %w", err)
	}

	// Send message
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to get data writer: %w", err)
	}

	if _, err := w.Write(message); err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	if err := w.Close(); err != nil {
		return fmt.Errorf("failed to close data writer: %w", err)
	}

	// Quit
	return client.Quit()
}

// buildMessage builds an email message
func (s *SMTPService) buildMessage(to, subject, body string) []byte {
	var buf bytes.Buffer

	buf.WriteString(fmt.Sprintf("From: %s <%s>\r\n", s.config.FromName, s.config.FromAddress))
	buf.WriteString(fmt.Sprintf("To: %s\r\n", to))
	if s.config.ReplyToAddress != "" {
		buf.WriteString(fmt.Sprintf("Reply-To: %s\r\n", s.config.ReplyToAddress))
	}
	buf.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	buf.WriteString("MIME-Version: 1.0\r\n")
	buf.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	buf.WriteString("\r\n")
	buf.WriteString(body)

	return buf.Bytes()
}

// renderMagicLinkTemplate renders the magic link email template
func (s *SMTPService) renderMagicLinkTemplate(link, token string) string {
	return renderMagicLinkHTML(link, token, s.config.MagicLinkTemplate)
}

// renderVerificationTemplate renders the email verification template
func (s *SMTPService) renderVerificationTemplate(link, token string) string {
	return renderVerificationHTML(link, token, s.config.VerificationTemplate)
}

// renderPasswordResetTemplate renders the password reset email template
func (s *SMTPService) renderPasswordResetTemplate(link, token string) string {
	return renderPasswordResetHTML(link, token, s.config.PasswordResetTemplate)
}

// Default email templates
const defaultMagicLinkTemplate = `
<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<style>
		body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
		.container { max-width: 600px; margin: 0 auto; padding: 20px; }
		.button { display: inline-block; padding: 12px 24px; background-color: #007bff; color: white; text-decoration: none; border-radius: 4px; }
		.footer { margin-top: 30px; font-size: 12px; color: #666; }
	</style>
</head>
<body>
	<div class="container">
		<h2>Your Login Link</h2>
		<p>Click the button below to log in to your account:</p>
		<p><a href="{{.Link}}" class="button">Log In</a></p>
		<p>Or copy and paste this link into your browser:</p>
		<p><code>{{.Link}}</code></p>
		<p><strong>This link expires in {{.Expiry}}</strong></p>
		<div class="footer">
			<p>If you didn't request this login link, please ignore this email.</p>
			<p>For security reasons, this link can only be used once.</p>
		</div>
	</div>
</body>
</html>
`

const defaultVerificationTemplate = `
<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<style>
		body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
		.container { max-width: 600px; margin: 0 auto; padding: 20px; }
		.button { display: inline-block; padding: 12px 24px; background-color: #28a745; color: white; text-decoration: none; border-radius: 4px; }
		.footer { margin-top: 30px; font-size: 12px; color: #666; }
	</style>
</head>
<body>
	<div class="container">
		<h2>Verify Your Email Address</h2>
		<p>Thank you for signing up! Click the button below to verify your email address:</p>
		<p><a href="{{.Link}}" class="button">Verify Email</a></p>
		<p>Or copy and paste this link into your browser:</p>
		<p><code>{{.Link}}</code></p>
		<div class="footer">
			<p>If you didn't create an account, please ignore this email.</p>
		</div>
	</div>
</body>
</html>
`

const defaultPasswordResetTemplate = `
<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<style>
		body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
		.container { max-width: 600px; margin: 0 auto; padding: 20px; }
		.button { display: inline-block; padding: 12px 24px; background-color: #dc3545; color: white; text-decoration: none; border-radius: 4px; }
		.footer { margin-top: 30px; font-size: 12px; color: #666; }
		.warning { background-color: #fff3cd; border-left: 4px solid #ffc107; padding: 12px; margin: 20px 0; }
	</style>
</head>
<body>
	<div class="container">
		<h2>Reset Your Password</h2>
		<p>We received a request to reset your password. Click the button below to choose a new password:</p>
		<p><a href="{{.Link}}" class="button">Reset Password</a></p>
		<p>Or copy and paste this link into your browser:</p>
		<p><code>{{.Link}}</code></p>
		<p><strong>This link expires in {{.Expiry}}</strong></p>
		<div class="warning">
			<p><strong>Security Reminder:</strong></p>
			<ul>
				<li>This link can only be used once</li>
				<li>We will never ask for your password via email</li>
				<li>If you didn't request this reset, please ignore this email</li>
			</ul>
		</div>
		<div class="footer">
			<p>If you didn't request a password reset, your account is still secure. Someone may have entered your email address by mistake.</p>
		</div>
	</div>
</body>
</html>
`

const defaultInvitationTemplate = `
<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<style>
		body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
		.container { max-width: 600px; margin: 0 auto; padding: 20px; }
		.button { display: inline-block; padding: 12px 24px; background-color: #5c6bc0; color: white; text-decoration: none; border-radius: 4px; }
		.footer { margin-top: 30px; font-size: 12px; color: #666; }
	</style>
</head>
<body>
	<div class="container">
		<h2>You've Been Invited!</h2>
		<p>{{.InviterName}} has invited you to join. Click the button below to accept:</p>
		<p><a href="{{.InviteLink}}" class="button">Accept Invitation</a></p>
		<p>Or copy and paste this link into your browser:</p>
		<p><code>{{.InviteLink}}</code></p>
		<p><strong>This invitation expires in 7 days</strong></p>
		<div class="footer">
			<p>If you weren't expecting this invitation, you can safely ignore this email.</p>
		</div>
	</div>
</body>
</html>
`

// IsConfigured returns true if the SMTP service is properly configured
func (s *SMTPService) IsConfigured() bool {
	return s.config.Enabled && s.config.IsConfigured()
}
