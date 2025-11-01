package email

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	"github.com/wayli-app/fluxbase/internal/config"
)

// SendGridService handles email sending via SendGrid
type SendGridService struct {
	config *config.EmailConfig
	client *sendgrid.Client
}

// NewSendGridService creates a new SendGrid email service
func NewSendGridService(cfg *config.EmailConfig) (*SendGridService, error) {
	if cfg.SendGridAPIKey == "" {
		return nil, fmt.Errorf("SendGrid API key is required")
	}

	client := sendgrid.NewSendClient(cfg.SendGridAPIKey)

	return &SendGridService{
		config: cfg,
		client: client,
	}, nil
}

// SendMagicLink sends a magic link email via SendGrid
func (s *SendGridService) SendMagicLink(ctx context.Context, to, token, link string) error {
	subject := "Your Login Link"
	body := renderMagicLinkHTML(link, token, s.config.MagicLinkTemplate)
	return s.Send(ctx, to, subject, body)
}

// SendVerificationEmail sends an email verification link via SendGrid
func (s *SendGridService) SendVerificationEmail(ctx context.Context, to, token, link string) error {
	subject := "Verify Your Email"
	body := renderVerificationHTML(link, token, s.config.VerificationTemplate)
	return s.Send(ctx, to, subject, body)
}

// SendPasswordReset sends a password reset email via SendGrid
func (s *SendGridService) SendPasswordReset(ctx context.Context, to, token, link string) error {
	subject := "Reset Your Password"
	body := renderPasswordResetHTML(link, token, s.config.PasswordResetTemplate)
	return s.Send(ctx, to, subject, body)
}

// Send sends a generic email via SendGrid
func (s *SendGridService) Send(ctx context.Context, to, subject, body string) error {
	from := mail.NewEmail(s.config.FromName, s.config.FromAddress)
	toEmail := mail.NewEmail("", to)
	message := mail.NewSingleEmail(from, subject, toEmail, "", body)

	// Set reply-to if configured
	if s.config.ReplyToAddress != "" {
		message.SetReplyTo(mail.NewEmail("", s.config.ReplyToAddress))
	}

	response, err := s.client.SendWithContext(ctx, message)
	if err != nil {
		log.Error().
			Err(err).
			Str("to", to).
			Str("subject", subject).
			Msg("Failed to send email via SendGrid")
		return fmt.Errorf("failed to send email: %w", err)
	}

	if response.StatusCode >= 400 {
		log.Error().
			Int("status_code", response.StatusCode).
			Str("body", response.Body).
			Str("to", to).
			Msg("SendGrid API returned error")
		return fmt.Errorf("SendGrid API error: %s (status %d)", response.Body, response.StatusCode)
	}

	log.Info().
		Str("to", to).
		Str("subject", subject).
		Int("status_code", response.StatusCode).
		Msg("Email sent successfully via SendGrid")

	return nil
}
