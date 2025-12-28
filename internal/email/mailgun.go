package email

import (
	"context"
	"fmt"
	"time"

	"github.com/fluxbase-eu/fluxbase/internal/config"
	"github.com/mailgun/mailgun-go/v4"
	"github.com/rs/zerolog/log"
)

// MailgunService handles email sending via Mailgun
type MailgunService struct {
	config *config.EmailConfig
	client *mailgun.MailgunImpl
}

// NewMailgunService creates a new Mailgun email service
func NewMailgunService(cfg *config.EmailConfig) (*MailgunService, error) {
	if cfg.MailgunAPIKey == "" {
		return nil, fmt.Errorf("mailgun API key is required")
	}
	if cfg.MailgunDomain == "" {
		return nil, fmt.Errorf("mailgun domain is required")
	}

	mg := mailgun.NewMailgun(cfg.MailgunDomain, cfg.MailgunAPIKey)

	return &MailgunService{
		config: cfg,
		client: mg,
	}, nil
}

// SendMagicLink sends a magic link email via Mailgun
func (s *MailgunService) SendMagicLink(ctx context.Context, to, token, link string) error {
	subject := "Your Login Link"
	body := renderMagicLinkHTML(link, token, s.config.MagicLinkTemplate)
	return s.Send(ctx, to, subject, body)
}

// SendVerificationEmail sends an email verification link via Mailgun
func (s *MailgunService) SendVerificationEmail(ctx context.Context, to, token, link string) error {
	subject := "Verify Your Email"
	body := renderVerificationHTML(link, token, s.config.VerificationTemplate)
	return s.Send(ctx, to, subject, body)
}

// SendPasswordReset sends a password reset email via Mailgun
func (s *MailgunService) SendPasswordReset(ctx context.Context, to, token, link string) error {
	subject := "Reset Your Password"
	body := renderPasswordResetHTML(link, token, s.config.PasswordResetTemplate)
	return s.Send(ctx, to, subject, body)
}

// SendInvitationEmail sends an invitation email via Mailgun
func (s *MailgunService) SendInvitationEmail(ctx context.Context, to, inviterName, inviteLink string) error {
	subject := "You've been invited!"
	body := renderInvitationHTML(inviterName, inviteLink)
	return s.Send(ctx, to, subject, body)
}

// Send sends a generic email via Mailgun
func (s *MailgunService) Send(ctx context.Context, to, subject, body string) error {
	message := mailgun.NewMessage(
		fmt.Sprintf("%s <%s>", s.config.FromName, s.config.FromAddress),
		subject,
		"", // Plain text body (optional)
		to,
	)

	// Set HTML body
	message.SetHTML(body)

	// Set reply-to if configured
	if s.config.ReplyToAddress != "" {
		message.SetReplyTo(s.config.ReplyToAddress)
	}

	// Send with timeout
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	resp, id, err := s.client.Send(ctx, message)
	if err != nil {
		log.Error().
			Err(err).
			Str("to", to).
			Str("subject", subject).
			Msg("Failed to send email via Mailgun")
		return fmt.Errorf("failed to send email: %w", err)
	}

	log.Info().
		Str("to", to).
		Str("subject", subject).
		Str("message_id", id).
		Str("response", resp).
		Msg("Email sent successfully via Mailgun")

	return nil
}

// IsConfigured returns true if the Mailgun service is properly configured
func (s *MailgunService) IsConfigured() bool {
	return s.config.Enabled && s.config.IsConfigured()
}
