package routes

import (
	"github.com/gofiber/fiber/v3"
)

type AuthDeps struct {
	AuthMiddleware   fiber.Handler
	TenantMiddleware fiber.Handler
	RequireRole      func(...string) fiber.Handler
	RequireScope     func(...string) fiber.Handler
	RateLimiters     map[string]fiber.Handler

	GetCSRFToken              fiber.Handler
	GetCaptchaConfig          fiber.Handler
	CheckCaptcha              fiber.Handler
	GetAuthConfig             fiber.Handler
	SignUp                    fiber.Handler
	SignIn                    fiber.Handler
	RefreshToken              fiber.Handler
	SendMagicLink             fiber.Handler
	VerifyMagicLink           fiber.Handler
	RequestPasswordReset      fiber.Handler
	ResetPassword             fiber.Handler
	VerifyPasswordReset       fiber.Handler
	VerifyEmail               fiber.Handler
	ResendVerification        fiber.Handler
	VerifyTOTP                fiber.Handler
	SendOTP                   fiber.Handler
	VerifyOTP                 fiber.Handler
	ResendOTP                 fiber.Handler
	SignInWithIDToken         fiber.Handler
	SignOut                   fiber.Handler
	GetUser                   fiber.Handler
	UpdateUser                fiber.Handler
	StartImpersonation        fiber.Handler
	StartAnonImpersonation    fiber.Handler
	StartServiceImpersonation fiber.Handler
	StopImpersonation         fiber.Handler
	GetActiveImpersonation    fiber.Handler
	ListImpersonationSessions fiber.Handler
	SetupTOTP                 fiber.Handler
	EnableTOTP                fiber.Handler
	DisableTOTP               fiber.Handler
	GetTOTPStatus             fiber.Handler
	GetUserIdentities         fiber.Handler
	LinkIdentity              fiber.Handler
	UnlinkIdentity            fiber.Handler
	Reauthenticate            fiber.Handler
	ListOAuthProviders        fiber.Handler
	OAuthAuthorize            fiber.Handler
	OAuthCallback             fiber.Handler
	GetProviderToken          fiber.Handler
	OAuthLogout               fiber.Handler
	OAuthLogoutCallback       fiber.Handler
	GetSPMetadata             fiber.Handler
	ListSAMLProviders         fiber.Handler
	InitiateSAMLLogin         fiber.Handler
	HandleSAMLAssertion       fiber.Handler
	HandleSAMLLogout          fiber.Handler
	InitiateSAMLLogout        fiber.Handler
}

func BuildAuthRoutes(deps *AuthDeps) *RouteGroup {
	r := []Route{
		{Method: "GET", Path: "/csrf", Handler: deps.GetCSRFToken, Summary: "Get CSRF token", Auth: AuthNone, Public: true},
		{Method: "GET", Path: "/captcha/config", Handler: deps.GetCaptchaConfig, Summary: "Get CAPTCHA config", Auth: AuthNone, Public: true},
		{Method: "POST", Path: "/captcha/check", Handler: deps.CheckCaptcha, Summary: "Check CAPTCHA required", Auth: AuthNone, Public: true},
		{Method: "GET", Path: "/config", Handler: deps.GetAuthConfig, Summary: "Get auth config", Auth: AuthNone, Public: true},
		{Method: "POST", Path: "/signup", Handler: deps.SignUp, Summary: "Register user", Auth: AuthNone, Public: true, Middlewares: limiter(deps, "signup")},
		{Method: "POST", Path: "/signin", Handler: deps.SignIn, Summary: "Authenticate user", Auth: AuthNone, Public: true, Middlewares: limiter(deps, "login")},
		{Method: "POST", Path: "/refresh", Handler: deps.RefreshToken, Summary: "Refresh token", Auth: AuthNone, Public: true, Middlewares: limiter(deps, "refresh")},
		{Method: "POST", Path: "/magiclink", Handler: deps.SendMagicLink, Summary: "Send magic link", Auth: AuthNone, Public: true, Middlewares: limiter(deps, "magiclink")},
		{Method: "POST", Path: "/magiclink/verify", Handler: deps.VerifyMagicLink, Summary: "Verify magic link", Auth: AuthNone, Public: true, Middlewares: limiter(deps, "login")},
		{Method: "POST", Path: "/password/reset", Handler: deps.RequestPasswordReset, Summary: "Request password reset", Auth: AuthNone, Public: true, Middlewares: limiter(deps, "password_reset")},
		{Method: "POST", Path: "/password/reset/confirm", Handler: deps.ResetPassword, Summary: "Reset password", Auth: AuthNone, Public: true, Middlewares: limiter(deps, "password_reset")},
		{Method: "POST", Path: "/password/reset/verify", Handler: deps.VerifyPasswordReset, Summary: "Verify reset token", Auth: AuthNone, Public: true},
		{Method: "POST", Path: "/verify-email", Handler: deps.VerifyEmail, Summary: "Verify email", Auth: AuthNone, Public: true},
		{Method: "POST", Path: "/verify-email/resend", Handler: deps.ResendVerification, Summary: "Resend verification", Auth: AuthNone, Public: true, Middlewares: limiter(deps, "magiclink")},
		{Method: "POST", Path: "/2fa/verify", Handler: deps.VerifyTOTP, Summary: "Verify 2FA", Auth: AuthNone, Public: true, Middlewares: limiter(deps, "2fa")},
		{Method: "POST", Path: "/otp/signin", Handler: deps.SendOTP, Summary: "Send OTP", Auth: AuthNone, Public: true, Middlewares: limiter(deps, "otp")},
		{Method: "POST", Path: "/otp/verify", Handler: deps.VerifyOTP, Summary: "Verify OTP", Auth: AuthNone, Public: true, Middlewares: limiter(deps, "2fa")},
		{Method: "POST", Path: "/otp/resend", Handler: deps.ResendOTP, Summary: "Resend OTP", Auth: AuthNone, Public: true, Middlewares: limiter(deps, "otp")},
		{Method: "POST", Path: "/signin/idtoken", Handler: deps.SignInWithIDToken, Summary: "Sign in with ID token", Auth: AuthNone, Public: true},
	}

	// Authenticated routes - auth middleware is auto-injected based on Auth: AuthRequired
	r = append(r, []Route{
		{Method: "POST", Path: "/signout", Handler: deps.SignOut, Summary: "Sign out", Auth: AuthRequired},
		{Method: "GET", Path: "/user", Handler: deps.GetUser, Summary: "Get user", Auth: AuthRequired},
		{Method: "PATCH", Path: "/user", Handler: deps.UpdateUser, Summary: "Update user", Auth: AuthRequired},
		{Method: "POST", Path: "/impersonate", Handler: deps.StartImpersonation, Summary: "Start impersonation", Auth: AuthRequired, Roles: []string{"admin", "instance_admin", "tenant_admin"}},
		{Method: "POST", Path: "/impersonate/anon", Handler: deps.StartAnonImpersonation, Summary: "Start anon impersonation", Auth: AuthRequired, Roles: []string{"admin", "instance_admin", "tenant_admin"}},
		{Method: "POST", Path: "/impersonate/service", Handler: deps.StartServiceImpersonation, Summary: "Start service impersonation", Auth: AuthRequired, Roles: []string{"admin", "instance_admin", "tenant_admin"}},
		{Method: "DELETE", Path: "/impersonate", Handler: deps.StopImpersonation, Summary: "Stop impersonation", Auth: AuthRequired},
		{Method: "GET", Path: "/impersonate", Handler: deps.GetActiveImpersonation, Summary: "Get active impersonation", Auth: AuthRequired},
		{Method: "GET", Path: "/impersonate/sessions", Handler: deps.ListImpersonationSessions, Summary: "List impersonation sessions", Auth: AuthRequired, Roles: []string{"admin", "instance_admin", "tenant_admin"}},
		{Method: "POST", Path: "/2fa/setup", Handler: deps.SetupTOTP, Summary: "Setup 2FA", Auth: AuthRequired},
		{Method: "POST", Path: "/2fa/enable", Handler: deps.EnableTOTP, Summary: "Enable 2FA", Auth: AuthRequired},
		{Method: "POST", Path: "/2fa/disable", Handler: deps.DisableTOTP, Summary: "Disable 2FA", Auth: AuthRequired},
		{Method: "GET", Path: "/2fa/status", Handler: deps.GetTOTPStatus, Summary: "Get 2FA status", Auth: AuthRequired},
		{Method: "GET", Path: "/user/identities", Handler: deps.GetUserIdentities, Summary: "Get identities", Auth: AuthRequired},
		{Method: "POST", Path: "/user/identities", Handler: deps.LinkIdentity, Summary: "Link identity", Auth: AuthRequired},
		{Method: "DELETE", Path: "/user/identities/:id", Handler: deps.UnlinkIdentity, Summary: "Unlink identity", Auth: AuthRequired},
		{Method: "POST", Path: "/reauthenticate", Handler: deps.Reauthenticate, Summary: "Reauthenticate", Auth: AuthRequired},
	}...)

	r = append(r, []Route{
		{Method: "GET", Path: "/oauth/providers", Handler: deps.ListOAuthProviders, Summary: "List OAuth providers", Auth: AuthNone, Public: true},
		{Method: "GET", Path: "/oauth/:provider/authorize", Handler: deps.OAuthAuthorize, Summary: "OAuth authorize", Auth: AuthNone, Public: true},
		{Method: "GET", Path: "/oauth/:provider/callback", Handler: deps.OAuthCallback, Summary: "OAuth callback", Auth: AuthNone, Public: true},
		{Method: "GET", Path: "/oauth/:provider/token", Handler: deps.GetProviderToken, Summary: "Get OAuth provider token", Auth: AuthRequired},
		{Method: "POST", Path: "/oauth/:provider/logout", Handler: deps.OAuthLogout, Summary: "OAuth logout", Auth: AuthRequired},
		{Method: "GET", Path: "/oauth/:provider/logout/callback", Handler: deps.OAuthLogoutCallback, Summary: "OAuth logout callback", Auth: AuthNone, Public: true},
		{Method: "GET", Path: "/saml/metadata/:provider", Handler: deps.GetSPMetadata, Summary: "Get SAML SP metadata", Auth: AuthNone, Public: true},
		{Method: "GET", Path: "/saml/providers", Handler: deps.ListSAMLProviders, Summary: "List SAML providers", Auth: AuthNone, Public: true},
		{Method: "GET", Path: "/saml/login/:provider", Handler: deps.InitiateSAMLLogin, Summary: "Initiate SAML login", Auth: AuthNone, Public: true},
		{Method: "POST", Path: "/saml/acs", Handler: deps.HandleSAMLAssertion, Summary: "SAML ACS callback", Auth: AuthNone, Public: true},
		{Method: "POST", Path: "/saml/slo", Handler: deps.HandleSAMLLogout, Summary: "SAML Single Logout", Auth: AuthNone, Public: true},
		{Method: "GET", Path: "/saml/slo", Handler: deps.HandleSAMLLogout, Summary: "SAML Single Logout (GET)", Auth: AuthNone, Public: true},
		{Method: "GET", Path: "/saml/logout/:provider", Handler: deps.InitiateSAMLLogout, Summary: "Initiate SAML logout", Auth: AuthRequired},
	}...)

	var middlewares []Middleware
	if deps.TenantMiddleware != nil {
		middlewares = append(middlewares, Middleware{
			Name:    "TenantContext",
			Handler: deps.TenantMiddleware,
		})
	}

	return &RouteGroup{
		Name:        "auth",
		Prefix:      "/api/v1/auth",
		Routes:      r,
		Middlewares: middlewares,
		AuthMiddlewares: &AuthMiddlewares{
			Required: deps.AuthMiddleware,
		},
		RequireRole: deps.RequireRole,
	}
}

func limiter(deps *AuthDeps, name string) []Middleware {
	if deps.RateLimiters == nil {
		return nil
	}
	if h, ok := deps.RateLimiters[name]; ok {
		return []Middleware{{Name: "RateLimiter:" + name, Handler: h}}
	}
	return nil
}
