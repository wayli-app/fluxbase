package routes

import "github.com/gofiber/fiber/v3"

type DashboardUserAuthDeps struct {
	RequireDashboardAuth fiber.Handler
	TenantMiddleware     fiber.Handler
	LoginLimiter         fiber.Handler
	RefreshLimiter       fiber.Handler
	PasswordResetLimiter fiber.Handler

	Signup                   fiber.Handler
	Login                    fiber.Handler
	RefreshToken             fiber.Handler
	VerifyTOTP               fiber.Handler
	RequestPasswordReset     fiber.Handler
	VerifyPasswordResetToken fiber.Handler
	ConfirmPasswordReset     fiber.Handler
	GetSSOProviders          fiber.Handler
	InitiateOAuthLogin       fiber.Handler
	OAuthCallback            fiber.Handler
	InitiateSAMLLogin        fiber.Handler
	SAMLACSCallback          fiber.Handler
	GetCurrentUser           fiber.Handler
	UpdateProfile            fiber.Handler
	ChangePassword           fiber.Handler
	DeleteAccount            fiber.Handler
	SetupTOTP                fiber.Handler
	EnableTOTP               fiber.Handler
	DisableTOTP              fiber.Handler
}

func BuildDashboardUserAuthRoutes(deps *DashboardUserAuthDeps) *RouteGroup {
	protected := []Middleware{
		{Name: "RequireDashboardAuth", Handler: deps.RequireDashboardAuth},
	}

	var groupMiddlewares []Middleware
	if deps.TenantMiddleware != nil {
		groupMiddlewares = append(groupMiddlewares, Middleware{
			Name:    "TenantContext",
			Handler: deps.TenantMiddleware,
		})
	}

	return &RouteGroup{
		Name:        "dashboard-user-auth",
		Prefix:      "/dashboard/auth",
		Middlewares: groupMiddlewares,
		Routes: []Route{
			{Method: "POST", Path: "/signup", Handler: deps.Signup, Summary: "Dashboard user signup", Auth: AuthNone, Public: true},
			{Method: "POST", Path: "/login", Handler: deps.Login, Summary: "Dashboard user login", Auth: AuthNone, Public: true, Middlewares: limiterMiddleware(deps.LoginLimiter, "LoginLimiter")},
			{Method: "POST", Path: "/refresh", Handler: deps.RefreshToken, Summary: "Refresh dashboard token", Auth: AuthNone, Public: true, Middlewares: limiterMiddleware(deps.RefreshLimiter, "RefreshLimiter")},
			{Method: "POST", Path: "/2fa/verify", Handler: deps.VerifyTOTP, Summary: "Verify 2FA TOTP", Auth: AuthNone, Public: true},
			{Method: "POST", Path: "/password/reset", Handler: deps.RequestPasswordReset, Summary: "Request password reset", Auth: AuthNone, Public: true, Middlewares: limiterMiddleware(deps.PasswordResetLimiter, "PasswordResetLimiter")},
			{Method: "POST", Path: "/password/reset/verify", Handler: deps.VerifyPasswordResetToken, Summary: "Verify password reset token", Auth: AuthNone, Public: true},
			{Method: "POST", Path: "/password/reset/confirm", Handler: deps.ConfirmPasswordReset, Summary: "Confirm password reset", Auth: AuthNone, Public: true, Middlewares: limiterMiddleware(deps.PasswordResetLimiter, "PasswordResetLimiter")},
			{Method: "GET", Path: "/sso/providers", Handler: deps.GetSSOProviders, Summary: "List SSO providers", Auth: AuthNone, Public: true},
			{Method: "GET", Path: "/sso/oauth/:provider", Handler: deps.InitiateOAuthLogin, Summary: "Initiate OAuth login", Auth: AuthNone, Public: true},
			{Method: "GET", Path: "/sso/oauth/:provider/callback", Handler: deps.OAuthCallback, Summary: "OAuth callback", Auth: AuthNone, Public: true},
			{Method: "GET", Path: "/sso/saml/:provider", Handler: deps.InitiateSAMLLogin, Summary: "Initiate SAML login", Auth: AuthNone, Public: true},
			{Method: "POST", Path: "/sso/saml/acs", Handler: deps.SAMLACSCallback, Summary: "SAML ACS callback", Auth: AuthNone, Public: true},
			{Method: "GET", Path: "/me", Handler: deps.GetCurrentUser, Summary: "Get current user", Auth: AuthRequired, Middlewares: protected},
			{Method: "PUT", Path: "/profile", Handler: deps.UpdateProfile, Summary: "Update profile", Auth: AuthRequired, Middlewares: protected},
			{Method: "POST", Path: "/password/change", Handler: deps.ChangePassword, Summary: "Change password", Auth: AuthRequired, Middlewares: protected},
			{Method: "DELETE", Path: "/account", Handler: deps.DeleteAccount, Summary: "Delete account", Auth: AuthRequired, Middlewares: protected},
			{Method: "POST", Path: "/2fa/setup", Handler: deps.SetupTOTP, Summary: "Setup 2FA", Auth: AuthRequired, Middlewares: protected},
			{Method: "POST", Path: "/2fa/enable", Handler: deps.EnableTOTP, Summary: "Enable 2FA", Auth: AuthRequired, Middlewares: protected},
			{Method: "POST", Path: "/2fa/disable", Handler: deps.DisableTOTP, Summary: "Disable 2FA", Auth: AuthRequired, Middlewares: protected},
		},
	}
}

func limiterMiddleware(h fiber.Handler, name string) []Middleware {
	if h == nil {
		return nil
	}
	return []Middleware{{Name: name, Handler: h}}
}
