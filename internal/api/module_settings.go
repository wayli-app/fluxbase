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
	Unified         *settings.UnifiedService
	Instance        *InstanceSettingsHandler
	Tenant          *TenantSettingsHandler
	System          *SystemSettingsHandler
	Custom          *CustomSettingsHandler
	Service         *settings.SecretsService
	User            *UserSettingsHandler
	App             *AppSettingsHandler
	Handler         *SettingsHandler
	EmailTemplate   *EmailTemplateHandler
	EmailSettings   *EmailSettingsHandler
	CaptchaSettings *CaptchaSettingsHandler
}

func (m *SettingsModule) Name() string { return "settings" }

func (m *SettingsModule) Init(ctx context.Context, registry *ServiceRegistry) error {
	cfg := registry.Config
	db := registry.DB
	authService := GetService[*auth.Service](registry)
	emailManager := GetService[*email.Manager](registry)
	systemSettingsService := GetService[*auth.SystemSettingsService](registry)
	captchaService := GetService[*auth.CaptchaService](registry)

	unifiedSettingsService := settings.NewUnifiedService(db, cfg, 	cfg.EncryptionKeyBytes)
	m.Unified = unifiedSettingsService
	m.Instance = NewInstanceSettingsHandler(unifiedSettingsService)

	var tenantStorage *tenantdb.Repository
	if ts := GetService[*tenantdb.Repository](registry); ts != nil {
		tenantStorage = ts
	}
	m.Tenant = NewTenantSettingsHandler(unifiedSettingsService, tenantStorage)

	tenantConfigResolver := NewTenantConfigResolver(db, cfg, unifiedSettingsService)
	registry.Register(tenantConfigResolver)
	log.Info().Msg("Tenant config resolver initialized for dynamic settings")

	m.System = NewSystemSettingsHandler(systemSettingsService, authService.GetSettingsCache())

	customSettingsService := settings.NewCustomSettingsService(db, 	cfg.EncryptionKeyBytes)
	m.Custom = NewCustomSettingsHandler(customSettingsService)

	secretsService := settings.NewSecretsService(db, 	cfg.EncryptionKeyBytes)
	m.Service = secretsService

	userSettingsHandler := NewUserSettingsHandler(db, customSettingsService)
	userSettingsHandler.SetSecretsService(secretsService)
	m.User = userSettingsHandler

	m.App = NewAppSettingsHandler(systemSettingsService, authService.GetSettingsCache(), cfg)
	m.Handler = NewSettingsHandler(db)

	m.EmailTemplate = NewEmailTemplateHandler(db, emailManager.WrapAsService())

	m.EmailSettings = NewEmailSettingsHandler(
		systemSettingsService,
		authService.GetSettingsCache(),
		emailManager,
		secretsService,
		cfg,
		unifiedSettingsService,
	)

	emailManager.SetSettingsCache(authService.GetSettingsCache())
	emailManager.SetSecretsService(secretsService)
	if err := emailManager.RefreshFromSettings(ctx); err != nil {
		log.Warn().Err(err).Msg("Failed to refresh email service from settings on startup")
	}

	m.CaptchaSettings = NewCaptchaSettingsHandler(
		systemSettingsService,
		authService.GetSettingsCache(),
		secretsService,
		&cfg.Security,
		captchaService,
	)

	if captchaService != nil {
		if err := captchaService.ReloadFromSettings(ctx, authService.GetSettingsCache(), &cfg.Security); err != nil {
			log.Warn().Err(err).Msg("Failed to refresh captcha service from settings on startup")
		}
	}

	registry.Register(secretsService)
	registry.Register(unifiedSettingsService)
	return nil
}
