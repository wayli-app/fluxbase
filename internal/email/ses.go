package email

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/aws/aws-sdk-go-v2/service/ses/types"
	"github.com/fluxbase-eu/fluxbase/internal/config"
	"github.com/rs/zerolog/log"
)

// SESService handles email sending via AWS SES
type SESService struct {
	config *config.EmailConfig
	client *ses.Client
}

// NewSESService creates a new AWS SES email service
func NewSESService(cfg *config.EmailConfig) (*SESService, error) {
	if cfg.SESRegion == "" {
		return nil, fmt.Errorf("AWS SES region is required")
	}

	// Create AWS config
	awsConfig := aws.Config{
		Region: cfg.SESRegion,
	}

	// Add credentials if provided
	if cfg.SESAccessKey != "" && cfg.SESSecretKey != "" {
		awsConfig.Credentials = credentials.NewStaticCredentialsProvider(
			cfg.SESAccessKey,
			cfg.SESSecretKey,
			"", // Session token (empty for static credentials)
		)
	}
	// If credentials are not provided, SDK will use default credential chain
	// (environment variables, IAM role, etc.)

	client := ses.NewFromConfig(awsConfig)

	return &SESService{
		config: cfg,
		client: client,
	}, nil
}

// SendMagicLink sends a magic link email via AWS SES
func (s *SESService) SendMagicLink(ctx context.Context, to, token, link string) error {
	subject := "Your Login Link"
	body := renderMagicLinkHTML(link, token, s.config.MagicLinkTemplate)
	return s.Send(ctx, to, subject, body)
}

// SendVerificationEmail sends an email verification link via AWS SES
func (s *SESService) SendVerificationEmail(ctx context.Context, to, token, link string) error {
	subject := "Verify Your Email"
	body := renderVerificationHTML(link, token, s.config.VerificationTemplate)
	return s.Send(ctx, to, subject, body)
}

// SendPasswordReset sends a password reset email via AWS SES
func (s *SESService) SendPasswordReset(ctx context.Context, to, token, link string) error {
	subject := "Reset Your Password"
	body := renderPasswordResetHTML(link, token, s.config.PasswordResetTemplate)
	return s.Send(ctx, to, subject, body)
}

// Send sends a generic email via AWS SES
func (s *SESService) Send(ctx context.Context, to, subject, body string) error {
	input := &ses.SendEmailInput{
		Source: aws.String(fmt.Sprintf("%s <%s>", s.config.FromName, s.config.FromAddress)),
		Destination: &types.Destination{
			ToAddresses: []string{to},
		},
		Message: &types.Message{
			Subject: &types.Content{
				Data:    aws.String(subject),
				Charset: aws.String("UTF-8"),
			},
			Body: &types.Body{
				Html: &types.Content{
					Data:    aws.String(body),
					Charset: aws.String("UTF-8"),
				},
			},
		},
	}

	// Set reply-to if configured
	if s.config.ReplyToAddress != "" {
		input.ReplyToAddresses = []string{s.config.ReplyToAddress}
	}

	output, err := s.client.SendEmail(ctx, input)
	if err != nil {
		log.Error().
			Err(err).
			Str("to", to).
			Str("subject", subject).
			Msg("Failed to send email via AWS SES")
		return fmt.Errorf("failed to send email: %w", err)
	}

	log.Info().
		Str("to", to).
		Str("subject", subject).
		Str("message_id", aws.ToString(output.MessageId)).
		Msg("Email sent successfully via AWS SES")

	return nil
}
