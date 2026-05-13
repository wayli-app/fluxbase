package config

import "fmt"

// EmailConfig contains email/SMTP settings
type EmailConfig struct {
	Enabled        bool   `mapstructure:"enabled"`
	Provider       string `mapstructure:"provider"` // smtp, sendgrid, mailgun, ses
	FromAddress    string `mapstructure:"from_address"`
	FromName       string `mapstructure:"from_name"`
	ReplyToAddress string `mapstructure:"reply_to_address"`

	// SMTP Settings
	SMTPHost     string `mapstructure:"smtp_host"`
	SMTPPort     int    `mapstructure:"smtp_port"`
	SMTPUsername string `mapstructure:"smtp_username"`
	SMTPPassword string `mapstructure:"smtp_password"`
	SMTPTLS      bool   `mapstructure:"smtp_tls"`

	// SendGrid Settings
	SendGridAPIKey string `mapstructure:"sendgrid_api_key"`

	// Mailgun Settings
	MailgunAPIKey string `mapstructure:"mailgun_api_key"`
	MailgunDomain string `mapstructure:"mailgun_domain"`

	// AWS SES Settings
	SESAccessKey string `mapstructure:"ses_access_key"`
	SESSecretKey string `mapstructure:"ses_secret_key"`
	SESRegion    string `mapstructure:"ses_region"`

	// Templates
	MagicLinkTemplate     string `mapstructure:"magic_link_template"`
	VerificationTemplate  string `mapstructure:"verification_template"`
	PasswordResetTemplate string `mapstructure:"password_reset_template"`
}

// Validate validates email configuration
func (ec *EmailConfig) Validate() error {
	// Validate provider if specified
	if ec.Provider != "" {
		validProviders := []string{"smtp", "sendgrid", "mailgun", "ses"}
		providerValid := false
		for _, p := range validProviders {
			if ec.Provider == p {
				providerValid = true
				break
			}
		}
		if !providerValid {
			return fmt.Errorf("invalid email provider: %s (must be one of: %v)", ec.Provider, validProviders)
		}
	}

	// Provider-specific settings are validated at runtime when sending emails,
	// allowing configuration via admin UI after startup

	return nil
}

// IsConfigured returns true if the email provider is fully configured and ready to send emails
func (ec *EmailConfig) IsConfigured() bool {
	if !ec.Enabled || ec.FromAddress == "" {
		return false
	}

	switch ec.Provider {
	case "smtp", "":
		return ec.SMTPHost != "" && ec.SMTPPort != 0
	case "sendgrid":
		return ec.SendGridAPIKey != ""
	case "mailgun":
		return ec.MailgunAPIKey != "" && ec.MailgunDomain != ""
	case "ses":
		// SES credentials are optional (can use AWS default credential chain)
		return ec.SESRegion != ""
	default:
		return false
	}
}
