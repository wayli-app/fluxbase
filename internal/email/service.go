package email

import (
	"context"
	"fmt"

	"github.com/fluxbase-eu/fluxbase/internal/config"
)

// Service defines the interface for email providers
type Service interface {
	// SendMagicLink sends a magic link email
	SendMagicLink(ctx context.Context, to, token, link string) error

	// SendVerificationEmail sends an email verification link
	SendVerificationEmail(ctx context.Context, to, token, link string) error

	// SendPasswordReset sends a password reset email
	SendPasswordReset(ctx context.Context, to, token, link string) error

	// SendInvitationEmail sends an invitation email
	SendInvitationEmail(ctx context.Context, to, inviterName, inviteLink string) error

	// Send sends a generic email
	Send(ctx context.Context, to, subject, body string) error
}

// NewService creates an email service based on configuration
func NewService(cfg *config.EmailConfig) (Service, error) {
	if !cfg.Enabled {
		return &NoOpService{reason: "email is disabled"}, nil
	}

	// Validate provider first before checking if fully configured
	switch cfg.Provider {
	case "smtp", "", "sendgrid", "mailgun", "ses":
		// Valid provider, continue
	default:
		return nil, fmt.Errorf("unsupported email provider: %s", cfg.Provider)
	}

	// If email is enabled but not fully configured, return a NoOpService
	// This allows the server to start and be configured via admin UI
	if !cfg.IsConfigured() {
		return &NoOpService{reason: "email provider is not fully configured"}, nil
	}

	switch cfg.Provider {
	case "smtp", "":
		return NewSMTPService(cfg), nil
	case "sendgrid":
		return NewSendGridService(cfg)
	case "mailgun":
		return NewMailgunService(cfg)
	case "ses":
		return NewSESService(cfg)
	default:
		return nil, fmt.Errorf("unsupported email provider: %s", cfg.Provider)
	}
}

// NoOpService is a no-op email service for when email is disabled or not configured
type NoOpService struct {
	reason string
}

// NewNoOpService creates a new NoOpService with the given reason
func NewNoOpService(reason string) *NoOpService {
	return &NoOpService{reason: reason}
}

func (s *NoOpService) SendMagicLink(ctx context.Context, to, token, link string) error {
	return fmt.Errorf("cannot send email: %s", s.reason)
}

func (s *NoOpService) SendVerificationEmail(ctx context.Context, to, token, link string) error {
	return fmt.Errorf("cannot send email: %s", s.reason)
}

func (s *NoOpService) SendPasswordReset(ctx context.Context, to, token, link string) error {
	return fmt.Errorf("cannot send email: %s", s.reason)
}

func (s *NoOpService) SendInvitationEmail(ctx context.Context, to, inviterName, inviteLink string) error {
	return fmt.Errorf("cannot send email: %s", s.reason)
}

func (s *NoOpService) Send(ctx context.Context, to, subject, body string) error {
	return fmt.Errorf("cannot send email: %s", s.reason)
}
