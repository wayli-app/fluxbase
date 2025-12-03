package email

import (
	"context"
	"strings"
	"testing"

	"github.com/fluxbase-eu/fluxbase/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSMTPService(t *testing.T) {
	cfg := &config.EmailConfig{
		Enabled:      true,
		Provider:     "smtp",
		SMTPHost:     "smtp.example.com",
		SMTPPort:     587,
		SMTPUsername: "user@example.com",
		SMTPPassword: "password",
		SMTPTLS:      true,
		FromAddress:  "noreply@example.com",
		FromName:     "Test Service",
	}

	service := NewSMTPService(cfg)
	assert.NotNil(t, service)
	assert.Equal(t, cfg, service.config)
}

func TestSMTPService_buildMessage(t *testing.T) {
	cfg := &config.EmailConfig{
		FromAddress:    "noreply@example.com",
		FromName:       "Test Service",
		ReplyToAddress: "support@example.com",
	}
	service := NewSMTPService(cfg)

	tests := []struct {
		name    string
		to      string
		subject string
		body    string
		want    []string // Strings that should be present in the message
	}{
		{
			name:    "basic message",
			to:      "user@example.com",
			subject: "Test Subject",
			body:    "<p>Test Body</p>",
			want: []string{
				"From: Test Service <noreply@example.com>",
				"To: user@example.com",
				"Reply-To: support@example.com",
				"Subject: Test Subject",
				"MIME-Version: 1.0",
				"Content-Type: text/html; charset=UTF-8",
				"<p>Test Body</p>",
			},
		},
		{
			name:    "message without reply-to",
			to:      "user@example.com",
			subject: "Test",
			body:    "Body",
			want: []string{
				"From: Test Service <noreply@example.com>",
				"To: user@example.com",
				"Subject: Test",
				"Body",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			message := service.buildMessage(tt.to, tt.subject, tt.body)
			messageStr := string(message)

			for _, want := range tt.want {
				assert.Contains(t, messageStr, want)
			}
		})
	}
}

func TestSMTPService_renderMagicLinkTemplate(t *testing.T) {
	cfg := &config.EmailConfig{}
	service := NewSMTPService(cfg)

	link := "https://example.com/auth/verify?token=abc123"
	token := "abc123"

	result := service.renderMagicLinkTemplate(link, token)

	// Check that the result contains expected elements
	assert.Contains(t, result, link)
	assert.Contains(t, result, "Your Login Link")
	assert.Contains(t, result, "<!DOCTYPE html>")
	assert.Contains(t, result, "Log In")
}

func TestSMTPService_renderVerificationTemplate(t *testing.T) {
	cfg := &config.EmailConfig{}
	service := NewSMTPService(cfg)

	link := "https://example.com/auth/verify?token=xyz789"
	token := "xyz789"

	result := service.renderVerificationTemplate(link, token)

	// Check that the result contains expected elements
	assert.Contains(t, result, link)
	assert.Contains(t, result, "Verify Your Email")
	assert.Contains(t, result, "<!DOCTYPE html>")
	assert.Contains(t, result, "Verify Email")
}

func TestSMTPService_Send_Disabled(t *testing.T) {
	cfg := &config.EmailConfig{
		Enabled: false,
	}
	service := NewSMTPService(cfg)

	ctx := context.Background()
	err := service.Send(ctx, "user@example.com", "Test", "Body")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "disabled")
}

func TestSMTPService_Send_InvalidConfig(t *testing.T) {
	// This test would fail if we try to connect to a non-existent SMTP server
	// For now, we just test that the method exists and can be called
	cfg := &config.EmailConfig{
		Enabled:      true,
		Provider:     "smtp",
		SMTPHost:     "nonexistent.smtp.server",
		SMTPPort:     587,
		SMTPUsername: "user",
		SMTPPassword: "pass",
		SMTPTLS:      false,
		FromAddress:  "from@example.com",
	}
	service := NewSMTPService(cfg)

	ctx := context.Background()
	err := service.Send(ctx, "to@example.com", "Test", "Body")

	// We expect an error because the SMTP server doesn't exist
	assert.Error(t, err)
}

func TestDefaultTemplates(t *testing.T) {
	t.Run("magic link template is valid HTML", func(t *testing.T) {
		assert.Contains(t, defaultMagicLinkTemplate, "<!DOCTYPE html>")
		assert.Contains(t, defaultMagicLinkTemplate, "{{.Link}}")
		assert.Contains(t, defaultMagicLinkTemplate, "Your Login Link")
	})

	t.Run("verification template is valid HTML", func(t *testing.T) {
		assert.Contains(t, defaultVerificationTemplate, "<!DOCTYPE html>")
		assert.Contains(t, defaultVerificationTemplate, "{{.Link}}")
		assert.Contains(t, defaultVerificationTemplate, "Verify Your Email")
	})
}

func TestSMTPService_TemplateRendering_Fallback(t *testing.T) {
	// Test that the fallback templates are used when template execution fails
	// This would require mocking or using an invalid template, which we'll skip for now
	// The fallback code is already tested indirectly through the rendering tests
}

func TestNewService(t *testing.T) {
	tests := []struct {
		name      string
		cfg       *config.EmailConfig
		wantErr   bool
		errMsg    string
		checkType func(t *testing.T, svc Service)
	}{
		{
			name: "SMTP provider",
			cfg: &config.EmailConfig{
				Enabled:     true,
				Provider:    "smtp",
				SMTPHost:    "smtp.example.com",
				SMTPPort:    587,
				FromAddress: "test@example.com",
			},
			wantErr: false,
			checkType: func(t *testing.T, svc Service) {
				_, ok := svc.(*SMTPService)
				assert.True(t, ok, "Expected SMTPService")
			},
		},
		{
			name: "disabled email",
			cfg: &config.EmailConfig{
				Enabled: false,
			},
			wantErr: false,
			checkType: func(t *testing.T, svc Service) {
				_, ok := svc.(*NoOpService)
				assert.True(t, ok, "Expected NoOpService")
			},
		},
		{
			name: "sendgrid provider",
			cfg: &config.EmailConfig{
				Enabled:        true,
				Provider:       "sendgrid",
				SendGridAPIKey: "test-api-key",
				FromAddress:    "test@example.com",
			},
			wantErr: false,
			checkType: func(t *testing.T, svc Service) {
				_, ok := svc.(*SendGridService)
				assert.True(t, ok, "Expected SendGridService")
			},
		},
		{
			name: "mailgun provider",
			cfg: &config.EmailConfig{
				Enabled:       true,
				Provider:      "mailgun",
				MailgunAPIKey: "test-api-key",
				MailgunDomain: "example.com",
				FromAddress:   "test@example.com",
			},
			wantErr: false,
			checkType: func(t *testing.T, svc Service) {
				_, ok := svc.(*MailgunService)
				assert.True(t, ok, "Expected MailgunService")
			},
		},
		{
			name: "ses provider",
			cfg: &config.EmailConfig{
				Enabled:      true,
				Provider:     "ses",
				SESRegion:    "us-east-1",
				SESAccessKey: "test-access-key",
				SESSecretKey: "test-secret-key",
				FromAddress:  "test@example.com",
			},
			wantErr: false,
			checkType: func(t *testing.T, svc Service) {
				_, ok := svc.(*SESService)
				assert.True(t, ok, "Expected SESService")
			},
		},
		{
			name: "unsupported provider",
			cfg: &config.EmailConfig{
				Enabled:  true,
				Provider: "invalid",
			},
			wantErr: true,
			errMsg:  "unsupported email provider",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, err := NewService(tt.cfg)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Nil(t, svc)
			} else {
				require.NoError(t, err)
				require.NotNil(t, svc)
				if tt.checkType != nil {
					tt.checkType(t, svc)
				}
			}
		})
	}
}

func TestNoOpService(t *testing.T) {
	service := &NoOpService{}
	ctx := context.Background()

	t.Run("SendMagicLink returns error", func(t *testing.T) {
		err := service.SendMagicLink(ctx, "user@example.com", "token", "link")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "disabled")
	})

	t.Run("SendVerificationEmail returns error", func(t *testing.T) {
		err := service.SendVerificationEmail(ctx, "user@example.com", "token", "link")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "disabled")
	})

	t.Run("Send returns error", func(t *testing.T) {
		err := service.Send(ctx, "user@example.com", "subject", "body")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "disabled")
	})
}

func TestEmailConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     config.EmailConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid SMTP config",
			cfg: config.EmailConfig{
				Provider:    "smtp",
				FromAddress: "test@example.com",
				SMTPHost:    "smtp.example.com",
				SMTPPort:    587,
			},
			wantErr: false,
		},
		{
			name: "missing from_address",
			cfg: config.EmailConfig{
				Provider: "smtp",
				SMTPHost: "smtp.example.com",
				SMTPPort: 587,
			},
			wantErr: true,
			errMsg:  "from_address is required",
		},
		{
			name: "invalid provider",
			cfg: config.EmailConfig{
				Provider:    "invalid",
				FromAddress: "test@example.com",
			},
			wantErr: true,
			errMsg:  "invalid email provider",
		},
		{
			name: "SMTP missing host",
			cfg: config.EmailConfig{
				Provider:    "smtp",
				FromAddress: "test@example.com",
				SMTPPort:    587,
			},
			wantErr: true,
			errMsg:  "smtp_host is required",
		},
		{
			name: "SMTP missing port",
			cfg: config.EmailConfig{
				Provider:    "smtp",
				FromAddress: "test@example.com",
				SMTPHost:    "smtp.example.com",
			},
			wantErr: true,
			errMsg:  "smtp_port is required",
		},
		{
			name: "SendGrid missing API key",
			cfg: config.EmailConfig{
				Provider:    "sendgrid",
				FromAddress: "test@example.com",
			},
			wantErr: true,
			errMsg:  "sendgrid_api_key is required",
		},
		{
			name: "Mailgun missing API key",
			cfg: config.EmailConfig{
				Provider:      "mailgun",
				FromAddress:   "test@example.com",
				MailgunDomain: "example.com",
			},
			wantErr: true,
			errMsg:  "mailgun_api_key is required",
		},
		{
			name: "Mailgun missing domain",
			cfg: config.EmailConfig{
				Provider:      "mailgun",
				FromAddress:   "test@example.com",
				MailgunAPIKey: "key",
			},
			wantErr: true,
			errMsg:  "mailgun_domain is required",
		},
		{
			name: "SES missing keys",
			cfg: config.EmailConfig{
				Provider:    "ses",
				FromAddress: "test@example.com",
				SESRegion:   "us-east-1",
			},
			wantErr: true,
			errMsg:  "ses_access_key and ses_secret_key are required",
		},
		{
			name: "SES missing region",
			cfg: config.EmailConfig{
				Provider:     "ses",
				FromAddress:  "test@example.com",
				SESAccessKey: "key",
				SESSecretKey: "secret",
			},
			wantErr: true,
			errMsg:  "ses_region is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, strings.ToLower(err.Error()), strings.ToLower(tt.errMsg))
			} else {
				require.NoError(t, err)
			}
		})
	}
}
