package email

import (
	"context"
	"sync"

	"github.com/fluxbase-eu/fluxbase/internal/auth"
	"github.com/fluxbase-eu/fluxbase/internal/config"
	"github.com/fluxbase-eu/fluxbase/internal/crypto"
	"github.com/rs/zerolog/log"
)

// Manager manages the email service with support for dynamic configuration refresh
type Manager struct {
	mu            sync.RWMutex
	service       Service
	settingsCache *auth.SettingsCache
	encryptionKey string
	envConfig     *config.EmailConfig // Fallback to env config
}

// NewManager creates a new email service manager
func NewManager(envConfig *config.EmailConfig, settingsCache *auth.SettingsCache, encryptionKey string) *Manager {
	m := &Manager{
		settingsCache: settingsCache,
		encryptionKey: encryptionKey,
		envConfig:     envConfig,
	}

	// Initialize with env config first
	service, err := NewService(envConfig)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to initialize email service from config, using NoOpService")
		service = NewNoOpService("initialization failed: " + err.Error())
	}
	m.service = service

	return m
}

// GetService returns the current email service
func (m *Manager) GetService() Service {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.service
}

// SetSettingsCache sets the settings cache for dynamic configuration
func (m *Manager) SetSettingsCache(cache *auth.SettingsCache) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.settingsCache = cache
}

// RefreshFromSettings rebuilds the email service from database settings
func (m *Manager) RefreshFromSettings(ctx context.Context) error {
	// Build config from settings cache
	cfg := m.buildConfigFromSettings(ctx)

	// Create new service
	service, err := NewService(cfg)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to create email service from settings, keeping current service")
		return err
	}

	// Swap service
	m.mu.Lock()
	m.service = service
	m.mu.Unlock()

	log.Info().
		Str("provider", cfg.Provider).
		Bool("enabled", cfg.Enabled).
		Bool("configured", cfg.IsConfigured()).
		Msg("Email service refreshed from settings")

	return nil
}

// buildConfigFromSettings creates an EmailConfig from the settings cache
func (m *Manager) buildConfigFromSettings(ctx context.Context) *config.EmailConfig {
	// Start with env config as base (for defaults and overrides)
	cfg := &config.EmailConfig{}
	if m.envConfig != nil {
		*cfg = *m.envConfig
	}

	// If no settings cache, use env config only
	if m.settingsCache == nil {
		return cfg
	}

	// Override with database settings (only if not overridden by env)
	// The settings cache handles the override logic

	cfg.Enabled = m.settingsCache.GetBool(ctx, "app.email.enabled", cfg.Enabled)
	cfg.Provider = m.settingsCache.GetString(ctx, "app.email.provider", cfg.Provider)
	cfg.FromAddress = m.settingsCache.GetString(ctx, "app.email.from_address", cfg.FromAddress)
	cfg.FromName = m.settingsCache.GetString(ctx, "app.email.from_name", cfg.FromName)

	// SMTP settings
	cfg.SMTPHost = m.settingsCache.GetString(ctx, "app.email.smtp_host", cfg.SMTPHost)
	cfg.SMTPPort = m.settingsCache.GetInt(ctx, "app.email.smtp_port", cfg.SMTPPort)
	cfg.SMTPUsername = m.settingsCache.GetString(ctx, "app.email.smtp_username", cfg.SMTPUsername)
	cfg.SMTPTLS = m.settingsCache.GetBool(ctx, "app.email.smtp_tls", cfg.SMTPTLS)

	// Decrypt SMTP password if encrypted
	smtpPassword := m.settingsCache.GetString(ctx, "app.email.smtp_password", cfg.SMTPPassword)
	if smtpPassword != "" && m.encryptionKey != "" {
		decrypted, err := crypto.DecryptIfNotEmpty(smtpPassword, m.encryptionKey)
		if err != nil {
			// If decryption fails, it might be a plaintext password from env
			log.Debug().Err(err).Msg("SMTP password decryption failed, using as-is")
		} else {
			smtpPassword = decrypted
		}
	}
	cfg.SMTPPassword = smtpPassword

	// SendGrid
	sendgridKey := m.settingsCache.GetString(ctx, "app.email.sendgrid_api_key", cfg.SendGridAPIKey)
	if sendgridKey != "" && m.encryptionKey != "" {
		decrypted, err := crypto.DecryptIfNotEmpty(sendgridKey, m.encryptionKey)
		if err != nil {
			log.Debug().Err(err).Msg("SendGrid API key decryption failed, using as-is")
		} else {
			sendgridKey = decrypted
		}
	}
	cfg.SendGridAPIKey = sendgridKey

	// Mailgun
	mailgunKey := m.settingsCache.GetString(ctx, "app.email.mailgun_api_key", cfg.MailgunAPIKey)
	if mailgunKey != "" && m.encryptionKey != "" {
		decrypted, err := crypto.DecryptIfNotEmpty(mailgunKey, m.encryptionKey)
		if err != nil {
			log.Debug().Err(err).Msg("Mailgun API key decryption failed, using as-is")
		} else {
			mailgunKey = decrypted
		}
	}
	cfg.MailgunAPIKey = mailgunKey
	cfg.MailgunDomain = m.settingsCache.GetString(ctx, "app.email.mailgun_domain", cfg.MailgunDomain)

	// AWS SES
	sesAccessKey := m.settingsCache.GetString(ctx, "app.email.ses_access_key", cfg.SESAccessKey)
	if sesAccessKey != "" && m.encryptionKey != "" {
		decrypted, err := crypto.DecryptIfNotEmpty(sesAccessKey, m.encryptionKey)
		if err != nil {
			log.Debug().Err(err).Msg("SES access key decryption failed, using as-is")
		} else {
			sesAccessKey = decrypted
		}
	}
	cfg.SESAccessKey = sesAccessKey

	sesSecretKey := m.settingsCache.GetString(ctx, "app.email.ses_secret_key", cfg.SESSecretKey)
	if sesSecretKey != "" && m.encryptionKey != "" {
		decrypted, err := crypto.DecryptIfNotEmpty(sesSecretKey, m.encryptionKey)
		if err != nil {
			log.Debug().Err(err).Msg("SES secret key decryption failed, using as-is")
		} else {
			sesSecretKey = decrypted
		}
	}
	cfg.SESSecretKey = sesSecretKey
	cfg.SESRegion = m.settingsCache.GetString(ctx, "app.email.ses_region", cfg.SESRegion)

	return cfg
}

// ServiceWrapper wraps the manager to implement the Service interface
// This allows the manager to be used wherever a Service is expected
type ServiceWrapper struct {
	manager *Manager
}

// WrapAsService creates a Service wrapper around the manager
func (m *Manager) WrapAsService() Service {
	return &ServiceWrapper{manager: m}
}

// SendMagicLink implements Service
func (w *ServiceWrapper) SendMagicLink(ctx context.Context, to, token, link string) error {
	return w.manager.GetService().SendMagicLink(ctx, to, token, link)
}

// SendVerificationEmail implements Service
func (w *ServiceWrapper) SendVerificationEmail(ctx context.Context, to, token, link string) error {
	return w.manager.GetService().SendVerificationEmail(ctx, to, token, link)
}

// SendPasswordReset implements Service
func (w *ServiceWrapper) SendPasswordReset(ctx context.Context, to, token, link string) error {
	return w.manager.GetService().SendPasswordReset(ctx, to, token, link)
}

// SendInvitationEmail implements Service
func (w *ServiceWrapper) SendInvitationEmail(ctx context.Context, to, inviterName, inviteLink string) error {
	return w.manager.GetService().SendInvitationEmail(ctx, to, inviterName, inviteLink)
}

// Send implements Service
func (w *ServiceWrapper) Send(ctx context.Context, to, subject, body string) error {
	return w.manager.GetService().Send(ctx, to, subject, body)
}
