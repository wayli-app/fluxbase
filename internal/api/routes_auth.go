package api

import (
	"github.com/gofiber/fiber/v3"

	"github.com/nimbleflux/fluxbase/internal/api/routes"
	"github.com/nimbleflux/fluxbase/internal/middleware"
)

func (s *Server) buildDashboardAuthRouteDeps() *routes.DashboardAuthDeps {
	return &routes.DashboardAuthDeps{
		SetupLimiter:    middleware.AdminSetupLimiterWithConfig(s.config.Security.AdminSetupRateLimit, s.config.Security.AdminSetupRateWindow, s.sharedMiddlewareStorage),
		LoginLimiter:    middleware.AdminLoginLimiterWithConfig(s.config.Security.AdminLoginRateLimit, s.config.Security.AdminLoginRateWindow, s.sharedMiddlewareStorage),
		GetSetupStatus:  s.Auth.AdminHandler.GetSetupStatus,
		InitialSetup:    s.Auth.AdminHandler.InitialSetup,
		AdminLogin:      s.Auth.AdminHandler.AdminLogin,
		RefreshToken:    s.Auth.AdminHandler.AdminRefreshToken,
		UnifiedAuth:     UnifiedAuthMiddleware(s.Auth.Handler.authService, s.Auth.DashboardHandler.jwtManager, s.db),
		AdminLogout:     s.Auth.AdminHandler.AdminLogout,
		GetCurrentAdmin: s.Auth.AdminHandler.GetCurrentAdmin,
	}
}

func (s *Server) buildAuthRouteDeps() *routes.AuthDeps {
	rateLimiters := map[string]fiber.Handler{
		"signup":         middleware.AuthSignupLimiterWithConfig(s.config.Security.AuthSignupRateLimit, s.config.Security.AuthSignupRateWindow, s.sharedMiddlewareStorage),
		"login":          middleware.AuthLoginLimiterWithConfig(s.config.Security.AuthLoginRateLimit, s.config.Security.AuthLoginRateWindow, s.sharedMiddlewareStorage),
		"refresh":        middleware.AuthRefreshLimiterWithConfig(s.config.Security.AuthRefreshRateLimit, s.config.Security.AuthRefreshRateWindow, s.sharedMiddlewareStorage),
		"magiclink":      middleware.AuthMagicLinkLimiterWithConfig(s.config.Security.AuthMagicLinkRateLimit, s.config.Security.AuthMagicLinkRateWindow, s.sharedMiddlewareStorage),
		"password_reset": middleware.AuthPasswordResetLimiterWithConfig(s.config.Security.AuthPasswordResetRateLimit, s.config.Security.AuthPasswordResetRateWindow, s.sharedMiddlewareStorage),
		"otp":            middleware.AuthMagicLinkLimiterWithConfig(s.config.Security.AuthMagicLinkRateLimit, s.config.Security.AuthMagicLinkRateWindow, s.sharedMiddlewareStorage),
		"2fa":            middleware.Auth2FALimiterWithConfig(s.config.Security.Auth2FARateLimit, s.config.Security.Auth2FARateWindow, s.sharedMiddlewareStorage),
	}

	return &routes.AuthDeps{
		AuthMiddleware:            AuthMiddleware(s.Auth.Handler.authService),
		TenantMiddleware:          s.Middleware.Tenant,
		RequireRole:               RequireRole,
		RequireScope:              middleware.RequireScope,
		RateLimiters:              rateLimiters,
		GetCSRFToken:              s.Auth.Handler.GetCSRFToken,
		GetCaptchaConfig:          s.Auth.Handler.GetCaptchaConfig,
		CheckCaptcha:              s.Auth.Handler.CheckCaptcha,
		GetAuthConfig:             s.Auth.Handler.GetAuthConfig,
		SignUp:                    s.Auth.Handler.SignUp,
		SignIn:                    s.Auth.Handler.SignIn,
		RefreshToken:              s.Auth.Handler.RefreshToken,
		SendMagicLink:             s.Auth.Handler.SendMagicLink,
		VerifyMagicLink:           s.Auth.Handler.VerifyMagicLink,
		RequestPasswordReset:      s.Auth.Handler.RequestPasswordReset,
		ResetPassword:             s.Auth.Handler.ResetPassword,
		VerifyPasswordReset:       s.Auth.Handler.VerifyPasswordResetToken,
		VerifyEmail:               s.Auth.Handler.VerifyEmail,
		ResendVerification:        s.Auth.Handler.ResendVerificationEmail,
		VerifyTOTP:                s.Auth.Handler.VerifyTOTP,
		SendOTP:                   s.Auth.Handler.SendOTP,
		VerifyOTP:                 s.Auth.Handler.VerifyOTP,
		ResendOTP:                 s.Auth.Handler.ResendOTP,
		SignInWithIDToken:         s.Auth.Handler.SignInWithIDToken,
		SignOut:                   s.Auth.Handler.SignOut,
		GetUser:                   s.Auth.Handler.GetUser,
		UpdateUser:                s.Auth.Handler.UpdateUser,
		StartImpersonation:        s.Auth.Handler.StartImpersonation,
		StartAnonImpersonation:    s.Auth.Handler.StartAnonImpersonation,
		StopImpersonation:         s.Auth.Handler.StopImpersonation,
		GetActiveImpersonation:    s.Auth.Handler.GetActiveImpersonation,
		ListImpersonationSessions: s.Auth.Handler.ListImpersonationSessions,
		SetupTOTP:                 s.Auth.Handler.SetupTOTP,
		EnableTOTP:                s.Auth.Handler.EnableTOTP,
		DisableTOTP:               s.Auth.Handler.DisableTOTP,
		GetTOTPStatus:             s.Auth.Handler.GetTOTPStatus,
		GetUserIdentities:         s.Auth.Handler.GetUserIdentities,
		LinkIdentity:              s.Auth.Handler.LinkIdentity,
		UnlinkIdentity:            s.Auth.Handler.UnlinkIdentity,
		Reauthenticate:            s.Auth.Handler.Reauthenticate,
		ListOAuthProviders:        s.Auth.OAuth.ListEnabledProviders,
		OAuthAuthorize:            s.Auth.OAuth.Authorize,
		OAuthCallback:             s.Auth.OAuth.Callback,
		GetSPMetadata:             s.Auth.SAMLProvider.GetSPMetadata,
		StartServiceImpersonation: s.Auth.Handler.StartServiceImpersonation,
		GetProviderToken:          s.Auth.OAuth.GetProviderToken,
		OAuthLogout:               s.Auth.OAuth.Logout,
		OAuthLogoutCallback:       s.Auth.OAuth.LogoutCallback,
		ListSAMLProviders:         s.Auth.SAML.ListSAMLProviders,
		InitiateSAMLLogin:         s.Auth.SAML.InitiateSAMLLogin,
		HandleSAMLAssertion:       s.Auth.SAML.HandleSAMLAssertion,
		HandleSAMLLogout:          s.Auth.SAML.HandleSAMLLogout,
		InitiateSAMLLogout:        s.Auth.SAML.InitiateSAMLLogout,
	}
}

func (s *Server) buildClientKeysRouteDeps() *routes.ClientKeysDeps {
	return &routes.ClientKeysDeps{
		RequireAuth:                      s.requireAuth,
		RequireAdminIfClientKeysDisabled: middleware.RequireAdminIfClientKeysDisabled(s.Auth.Handler.authService.GetSettingsCache()),
		RequireScope:                     middleware.RequireScope,
		TenantMiddleware:                 s.Middleware.Tenant,
		ListClientKeys:                   s.Auth.ClientKeyHandler.ListClientKeys,
		GetClientKey:                     s.Auth.ClientKeyHandler.GetClientKey,
		CreateClientKey:                  s.Auth.ClientKeyHandler.CreateClientKey,
		UpdateClientKey:                  s.Auth.ClientKeyHandler.UpdateClientKey,
		DeleteClientKey:                  s.Auth.ClientKeyHandler.DeleteClientKey,
		RevokeClientKey:                  s.Auth.ClientKeyHandler.RevokeClientKey,
	}
}

func (s *Server) buildDashboardUserAuthRouteDeps() *routes.DashboardUserAuthDeps {
	return &routes.DashboardUserAuthDeps{
		RequireDashboardAuth:     s.Auth.DashboardHandler.RequireDashboardAuth,
		TenantMiddleware:         s.Middleware.Tenant,
		Signup:                   s.Auth.DashboardHandler.Signup,
		Login:                    s.Auth.DashboardHandler.Login,
		RefreshToken:             s.Auth.DashboardHandler.RefreshToken,
		VerifyTOTP:               s.Auth.DashboardHandler.VerifyTOTP,
		RequestPasswordReset:     s.Auth.DashboardHandler.RequestPasswordReset,
		VerifyPasswordResetToken: s.Auth.DashboardHandler.VerifyPasswordResetToken,
		ConfirmPasswordReset:     s.Auth.DashboardHandler.ConfirmPasswordReset,
		GetSSOProviders:          s.Auth.DashboardHandler.GetSSOProviders,
		InitiateOAuthLogin:       s.Auth.DashboardHandler.InitiateOAuthLogin,
		OAuthCallback:            s.Auth.DashboardHandler.OAuthCallback,
		InitiateSAMLLogin:        s.Auth.DashboardHandler.InitiateSAMLLogin,
		SAMLACSCallback:          s.Auth.DashboardHandler.SAMLACSCallback,
		GetCurrentUser:           s.Auth.DashboardHandler.GetCurrentUser,
		UpdateProfile:            s.Auth.DashboardHandler.UpdateProfile,
		ChangePassword:           s.Auth.DashboardHandler.ChangePassword,
		DeleteAccount:            s.Auth.DashboardHandler.DeleteAccount,
		SetupTOTP:                s.Auth.DashboardHandler.SetupTOTP,
		EnableTOTP:               s.Auth.DashboardHandler.EnableTOTP,
		DisableTOTP:              s.Auth.DashboardHandler.DisableTOTP,
	}
}

func (s *Server) buildInvitationRouteDeps() *routes.InvitationDeps {
	return &routes.InvitationDeps{
		ValidateInvitation: s.Auth.Invitation.ValidateInvitation,
		AcceptInvitation:   s.Auth.Invitation.AcceptInvitation,
	}
}
