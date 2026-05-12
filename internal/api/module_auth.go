package api

import (
	"context"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/auth"
	"github.com/nimbleflux/fluxbase/internal/email"
	"github.com/nimbleflux/fluxbase/internal/middleware"
)

type AuthModule struct {
	Service               *auth.Service
	CaptchaService        *auth.CaptchaService
	SystemSettingsService *auth.SystemSettingsService
	UserMgmtService       *auth.UserManagementService
	InvitationService     *auth.InvitationService
	Handlers              *AuthHandlers
	SQLHandler            *SQLHandler
	RequireAuth           fiber.Handler
	OptionalAuth          fiber.Handler
}

func (m *AuthModule) Name() string { return "auth" }

func (m *AuthModule) Init(ctx context.Context, registry *ServiceRegistry) error {
	cfg := registry.Config
	db := registry.DB

	lazyEmailService := GetService[*email.LazyService](registry)
	var emailService email.Service
	if lazyEmailService != nil {
		emailService = lazyEmailService
	}

	authService := auth.NewService(db, &cfg.Auth, emailService, cfg.GetPublicBaseURL(), registry.Metrics)
	authService.SetEncryptionKey(cfg.EncryptionKeyBytes)
	totpRateLimiter := auth.NewTOTPRateLimiter(db, auth.DefaultTOTPRateLimiterConfig())
	authService.SetTOTPRateLimiter(totpRateLimiter)
	m.Service = authService

	clientKeyService := auth.NewClientKeyService(db, authService.GetSettingsCache())

	m.UserMgmtService = auth.NewUserManagementService(
		auth.NewUserRepository(db),
		auth.NewSessionRepository(db),
		auth.NewPasswordHasherWithConfig(auth.PasswordHasherConfig{MinLength: cfg.Auth.PasswordMinLen, Cost: cfg.Auth.BcryptCost}),
		emailService,
		cfg.GetPublicBaseURL(),
	)

	captchaService, err := auth.NewCaptchaService(&cfg.Security.Captcha)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to initialize CAPTCHA service - CAPTCHA protection disabled")
		captchaService = nil
	}
	m.CaptchaService = captchaService

	authHandler := NewAuthHandler(db, authService, captchaService, cfg.GetPublicBaseURL())

	dashboardJWTManager, err := auth.NewJWTManager(cfg.Auth.JWTSecret, 24*time.Hour, 168*time.Hour)
	if err != nil {
		return fmt.Errorf("failed to create dashboard JWT manager: %w", err)
	}
	dashboardAuthService := auth.NewDashboardAuthService(db, dashboardJWTManager, cfg.Auth.TOTPIssuer)

	systemSettingsService := auth.NewSystemSettingsService(db)
	systemSettingsService.SetCache(authService.GetSettingsCache())
	m.SystemSettingsService = systemSettingsService

	adminAuthHandler := NewAdminAuthHandler(authService, auth.NewUserRepository(db), dashboardAuthService, systemSettingsService, cfg)
	clientKeyHandler := NewClientKeyHandler(clientKeyService)

	userMgmtHandler := NewUserManagementHandler(m.UserMgmtService, authService)
	invitationService := auth.NewInvitationService(db)
	m.InvitationService = invitationService
	invitationHandler := NewInvitationHandler(invitationService, dashboardAuthService, emailService, cfg.GetPublicBaseURL())

	oauthProviderHandler := NewOAuthProviderHandler(db, authService.GetSettingsCache(), cfg.EncryptionKeyBytes, cfg.GetPublicBaseURL(), cfg.Auth.OAuthProviders)
	jwtManager, err := auth.NewJWTManager(cfg.Auth.JWTSecret, cfg.Auth.JWTExpiry, cfg.Auth.RefreshExpiry)
	if err != nil {
		return fmt.Errorf("failed to create JWT manager: %w", err)
	}
	oauthHandler := NewOAuthHandler(db, authService, jwtManager, cfg.GetPublicBaseURL(), cfg.EncryptionKeyBytes, cfg.Auth.OAuthProviders)

	samlService, samlErr := auth.NewSAMLService(db, cfg.GetPublicBaseURL(), cfg.Auth.SAMLProviders)
	if samlErr != nil {
		log.Warn().Err(samlErr).Msg("Failed to initialize SAML service from config")
	}
	if samlService != nil {
		if err := samlService.LoadProvidersFromDB(ctx); err != nil {
			log.Warn().Err(err).Msg("Failed to load SAML providers from database")
		}
	}
	samlProviderHandler := NewSAMLProviderHandler(db, samlService)

	var samlHandler *SAMLHandler
	if samlService != nil {
		samlHandler = NewSAMLHandler(samlService, authService)
	}

	dashboardAuthHandler := NewDashboardAuthHandler(dashboardAuthService, dashboardJWTManager, db, samlService, emailService, cfg.GetPublicBaseURL(), cfg.EncryptionKeyBytes, oauthHandler)
	adminSessionHandler := NewAdminSessionHandler(auth.NewSessionRepository(db))

	if err := oauthProviderHandler.EncryptExistingSecrets(ctx); err != nil {
		log.Error().Err(err).Msg("Failed to encrypt existing OAuth provider secrets")
	}

	m.SQLHandler = NewSQLHandler(db, authService)

	m.Handlers = &AuthHandlers{
		Handler:          authHandler,
		AdminHandler:     adminAuthHandler,
		DashboardHandler: dashboardAuthHandler,
		ClientKeyHandler: clientKeyHandler,
		ClientKeyService: clientKeyService,
		OAuthProvider:    oauthProviderHandler,
		OAuth:            oauthHandler,
		SAMLProvider:     samlProviderHandler,
		SAML:             samlHandler,
		SAMLService:      samlService,
		AdminSession:     adminSessionHandler,
		UserManagement:   userMgmtHandler,
		Invitation:       invitationHandler,
	}

	m.RequireAuth = middleware.RequireAuthOrServiceKey(authService, clientKeyService, db.Pool(), &cfg.Security, dashboardJWTManager)
	m.OptionalAuth = middleware.OptionalAuthOrServiceKey(authService, clientKeyService, db.Pool(), &cfg.Security, dashboardJWTManager)

	registry.Register(m.Service)
	registry.Register(m.CaptchaService)
	registry.Register(m.SystemSettingsService)
	registry.Register(m.UserMgmtService)
	registry.Register(m.InvitationService)

	return nil
}

func (m *AuthModule) Shutdown(ctx context.Context) error {
	if m.Handlers != nil && m.Handlers.OAuth != nil {
		m.Handlers.OAuth.Stop()
	}
	return nil
}
