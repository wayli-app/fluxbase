package email

import (
	"context"
	"fmt"

	"github.com/wayli-app/fluxbase/internal/config"
)

// Service defines the interface for email providers
type Service interface {
	// SendMagicLink sends a magic link email
	SendMagicLink(ctx context.Context, to, token, link string) error

	// SendVerificationEmail sends an email verification link
	SendVerificationEmail(ctx context.Context, to, token, link string) error

	// SendPasswordReset sends a password reset email
	SendPasswordReset(ctx context.Context, to, token, link string) error

	// Send sends a generic email
	Send(ctx context.Context, to, subject, body string) error
}

// NewService creates an email service based on configuration
func NewService(cfg *config.EmailConfig) (Service, error) {
	if !cfg.Enabled {
		return &NoOpService{}, nil
	}

	switch cfg.Provider {
	case "smtp", "":
		return NewSMTPService(cfg), nil
	case "sendgrid":
		// TODO: Implement SendGrid
		return nil, fmt.Errorf("sendgrid provider not yet implemented")
	case "mailgun":
		// TODO: Implement Mailgun
		return nil, fmt.Errorf("mailgun provider not yet implemented")
	case "ses":
		// TODO: Implement AWS SES
		return nil, fmt.Errorf("ses provider not yet implemented")
	default:
		return nil, fmt.Errorf("unsupported email provider: %s", cfg.Provider)
	}
}

// NoOpService is a no-op email service for when email is disabled
type NoOpService struct{}

func (s *NoOpService) SendMagicLink(ctx context.Context, to, token, link string) error {
	return fmt.Errorf("email service is disabled")
}

func (s *NoOpService) SendVerificationEmail(ctx context.Context, to, token, link string) error {
	return fmt.Errorf("email service is disabled")
}

func (s *NoOpService) SendPasswordReset(ctx context.Context, to, token, link string) error {
	return fmt.Errorf("email service is disabled")
}

func (s *NoOpService) Send(ctx context.Context, to, subject, body string) error {
	return fmt.Errorf("email service is disabled")
}
