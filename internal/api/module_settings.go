package api

import (
	"context"

	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/auth"
	"github.com/nimbleflux/fluxbase/internal/email"
	"github.com/nimbleflux/fluxbase/internal/settings"
	"github.com/nimbleflux/fluxbase/internal/tenantdb"
)

type SettingsModule struct {
	Handlers *SettingsHandlers
	Email    *EmailHandlers
	Captcha  *CaptchaHandlers
}

func (m *SettingsModule) Name() string { return "settings" }

func (m *SettingsModule) Init(ctx context.Context, registry *ServiceRegistry) error {
	cfg := registry.Config
	db := registry.DB
	authService := GetService[*auth.Service](registry)
	systemSettingsService := GetService[*auth.SystemSettingsService](registry)
	captchaService := GetService[*auth.CaptchaService](registry)

	unifiedSettingsService := settings.NewUnifiedService(db, cfg, cfg.EncryptionKeyBytes)
	m.Handlers = &SettingsHandlers{}
	m.Handlers.Unified = unifiedSettingsService
	m.Handlers.Instance = NewInstanceSettingsHandler(unifiedSettingsService)

	var tenantStorage *tenantdb.Repository
	if ts := GetService[*tenantdb.Repository](registry); ts != nil {
		tenantStorage = ts
	}
	m.Handlers.Tenant = NewTenantSettingsHandler(unifiedSettingsService, tenantStorage)

	tenantConfigResolver := NewTenantConfigResolver(db, cfg, unifiedSettingsService)
	registry.Register(tenantConfigResolver)
	log.Info().Msg("Tenant config resolver initialized for dynamic settings")

	m.Handlers.System = NewSystemSettingsHandler(systemSettingsService, authService.GetSettingsCache())

	customSettingsService := settings.NewCustomSettingsService(db, cfg.EncryptionKeyBytes)
	m.Handlers.Custom = NewCustomSettingsHandler(customSettingsService)

	secretsService := settings.NewSecretsService(db, cfg.EncryptionKeyBytes)
	m.Handlers.Service = secretsService

	emailManager := email.NewManager(&cfg.Email, authService.GetSettingsCache(), secretsService, cfg)
	if err := emailManager.RefreshFromSettings(ctx); err != nil {
		log.Warn().Err(err).Msg("Failed to refresh email service from settings on startup")
	}
	registry.Register(emailManager)

	if lazyEmail := GetService[*email.LazyService](registry); lazyEmail != nil {
		lazyEmail.SetResolver(emailManager.WrapAsService)
	}

	m.Email = &EmailHandlers{
		Template: NewEmailTemplateHandler(db, emailManager.WrapAsService()),
	}

	m.Email.Settings = NewEmailSettingsHandler(
		systemSettingsService,
		authService.GetSettingsCache(),
		emailManager,
		secretsService,
		cfg,
		unifiedSettingsService,
	)

	userSettingsHandler := NewUserSettingsHandler(db, customSettingsService)
	userSettingsHandler.SetSecretsService(secretsService)
	m.Handlers.User = userSettingsHandler

	m.Handlers.App = NewAppSettingsHandler(systemSettingsService, authService.GetSettingsCache(), cfg)
	m.Handlers.Handler = NewSettingsHandler(db)

	m.Email = &EmailHandlers{
		Template: NewEmailTemplateHandler(db, emailManager.WrapAsService()),
	}

	m.Email.Settings = NewEmailSettingsHandler(
		systemSettingsService,
		authService.GetSettingsCache(),
		emailManager,
		secretsService,
		cfg,
		unifiedSettingsService,
	)

	m.Captcha = &CaptchaHandlers{
		Settings: NewCaptchaSettingsHandler(
			systemSettingsService,
			authService.GetSettingsCache(),
			secretsService,
			&cfg.Security,
			captchaService,
		),
	}

	if captchaService != nil {
		if err := captchaService.ReloadFromSettings(ctx, authService.GetSettingsCache(), &cfg.Security); err != nil {
			log.Warn().Err(err).Msg("Failed to refresh captcha service from settings on startup")
		}
	}

	registry.Register(secretsService)
	registry.Register(unifiedSettingsService)
	return nil
}
