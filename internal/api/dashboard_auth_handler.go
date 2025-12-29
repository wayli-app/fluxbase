package api

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/fluxbase-eu/fluxbase/internal/auth"
	"github.com/fluxbase-eu/fluxbase/internal/database"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"golang.org/x/oauth2"
)

// DashboardAuthHandler handles dashboard authentication endpoints
type DashboardAuthHandler struct {
	authService *auth.DashboardAuthService
	jwtManager  *auth.JWTManager
	db          *database.Connection
	samlService *auth.SAMLService
	baseURL     string

	// OAuth state storage (in production, use Redis or database)
	oauthStates    map[string]*dashboardOAuthState
	oauthStatesMu  sync.RWMutex
	oauthConfigs   map[string]*oauth2.Config
	oauthConfigsMu sync.RWMutex
}

// dashboardOAuthState holds OAuth state for dashboard SSO
type dashboardOAuthState struct {
	Provider   string
	CreatedAt  time.Time
	RedirectTo string
}

// NewDashboardAuthHandler creates a new dashboard auth handler
func NewDashboardAuthHandler(authService *auth.DashboardAuthService, jwtManager *auth.JWTManager, db *database.Connection, samlService *auth.SAMLService, baseURL string) *DashboardAuthHandler {
	return &DashboardAuthHandler{
		authService:  authService,
		jwtManager:   jwtManager,
		db:           db,
		samlService:  samlService,
		baseURL:      baseURL,
		oauthStates:  make(map[string]*dashboardOAuthState),
		oauthConfigs: make(map[string]*oauth2.Config),
	}
}

// RegisterRoutes registers dashboard auth routes
func (h *DashboardAuthHandler) RegisterRoutes(app *fiber.App) {
	dashboard := app.Group("/dashboard/auth")

	// Public routes
	dashboard.Post("/signup", h.Signup)
	dashboard.Post("/login", h.Login)
	dashboard.Post("/2fa/verify", h.VerifyTOTP)

	// SSO routes (public)
	dashboard.Get("/sso/providers", h.GetSSOProviders)
	dashboard.Get("/sso/oauth/:provider", h.InitiateOAuthLogin)
	dashboard.Get("/sso/oauth/:provider/callback", h.OAuthCallback)
	dashboard.Get("/sso/saml/:provider", h.InitiateSAMLLogin)
	dashboard.Post("/sso/saml/acs", h.SAMLACSCallback)

	// Protected routes (require dashboard JWT)
	dashboard.Get("/me", h.RequireDashboardAuth, h.GetCurrentUser)
	dashboard.Put("/profile", h.RequireDashboardAuth, h.UpdateProfile)
	dashboard.Post("/password/change", h.RequireDashboardAuth, h.ChangePassword)
	dashboard.Delete("/account", h.RequireDashboardAuth, h.DeleteAccount)

	// 2FA routes
	dashboard.Post("/2fa/setup", h.RequireDashboardAuth, h.SetupTOTP)
	dashboard.Post("/2fa/enable", h.RequireDashboardAuth, h.EnableTOTP)
	dashboard.Post("/2fa/disable", h.RequireDashboardAuth, h.DisableTOTP)
}

// Signup creates a new dashboard user account
// Only allowed if no dashboard users exist yet (first user self-registration)
func (h *DashboardAuthHandler) Signup(c *fiber.Ctx) error {
	// Check if any dashboard users exist
	hasUsers, err := h.authService.HasExistingUsers(c.Context())
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to check existing users")
	}

	// If users exist, signup is disabled (must use invite instead)
	if hasUsers {
		return fiber.NewError(fiber.StatusForbidden, "Sign-up is disabled. Please contact an administrator for an invitation.")
	}

	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		FullName string `json:"full_name"`
	}

	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("Invalid request body: %v", err))
	}

	if req.Email == "" || req.Password == "" || req.FullName == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Email, password, and full name are required")
	}

	user, err := h.authService.CreateUser(c.Context(), req.Email, req.Password, req.FullName)
	if err != nil {
		// Check for validation errors
		errMsg := err.Error()
		if strings.Contains(errMsg, "invalid email") ||
			strings.Contains(errMsg, "invalid name") ||
			strings.Contains(errMsg, "password must be") {
			return fiber.NewError(fiber.StatusBadRequest, errMsg)
		}
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to create user")
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"user":    user,
		"message": "Account created successfully",
	})
}

// Login authenticates a dashboard user
func (h *DashboardAuthHandler) Login(c *fiber.Ctx) error {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
	}

	if req.Email == "" || req.Password == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Email and password are required")
	}

	ipAddress := getIPAddress(c)
	userAgent := string(c.Request().Header.UserAgent())

	user, loginResp, err := h.authService.Login(c.Context(), req.Email, req.Password, ipAddress, userAgent)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || err.Error() == "invalid credentials" {
			return fiber.NewError(fiber.StatusUnauthorized, "Invalid email or password")
		}
		if err.Error() == "account is locked" {
			return fiber.NewError(fiber.StatusForbidden, "Account is locked due to too many failed login attempts")
		}
		if err.Error() == "account is inactive" {
			return fiber.NewError(fiber.StatusForbidden, "Account is inactive")
		}
		// Log the actual error for debugging
		fmt.Printf("Dashboard login error: %v\n", err)
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("Login failed: %v", err))
	}

	// Check if 2FA is enabled
	if user.TOTPEnabled {
		return c.JSON(fiber.Map{
			"requires_2fa": true,
			"user_id":      user.ID,
		})
	}

	return c.JSON(fiber.Map{
		"access_token":  loginResp.AccessToken,
		"refresh_token": loginResp.RefreshToken,
		"expires_in":    loginResp.ExpiresIn,
		"user":          user,
	})
}

// VerifyTOTP verifies a TOTP code during login
func (h *DashboardAuthHandler) VerifyTOTP(c *fiber.Ctx) error {
	var req struct {
		UserID string `json:"user_id"`
		Code   string `json:"code"`
	}

	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
	}

	if req.UserID == "" || req.Code == "" {
		return fiber.NewError(fiber.StatusBadRequest, "User ID and code are required")
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid user ID")
	}

	err = h.authService.VerifyTOTP(c.Context(), userID, req.Code)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "Invalid 2FA code")
	}

	// Get user after successful 2FA
	user, err := h.authService.GetUserByID(c.Context(), userID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to fetch user")
	}

	// Generate JWT tokens
	accessToken, refreshToken, _, err := h.jwtManager.GenerateTokenPair(user.ID.String(), user.Email, "dashboard_admin", nil, nil)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to generate tokens")
	}

	return c.JSON(fiber.Map{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"expires_in":    86400, // 24 hours
		"user":          user,
	})
}

// GetCurrentUser returns the currently authenticated dashboard user
func (h *DashboardAuthHandler) GetCurrentUser(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uuid.UUID)

	user, err := h.authService.GetUserByID(c.Context(), userID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "User not found")
	}

	// Set role from JWT (RequireDashboardAuth middleware validates this is "dashboard_admin")
	user.Role = "dashboard_admin"

	return c.JSON(user)
}

// UpdateProfile updates the current user's profile
func (h *DashboardAuthHandler) UpdateProfile(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uuid.UUID)

	var req struct {
		FullName  string  `json:"full_name"`
		AvatarURL *string `json:"avatar_url"`
	}

	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
	}

	if req.FullName == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Full name is required")
	}

	err := h.authService.UpdateProfile(c.Context(), userID, req.FullName, req.AvatarURL)
	if err != nil {
		// Check for validation errors
		errMsg := err.Error()
		if strings.Contains(errMsg, "invalid name") ||
			strings.Contains(errMsg, "invalid avatar URL") {
			return fiber.NewError(fiber.StatusBadRequest, errMsg)
		}
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to update profile")
	}

	user, _ := h.authService.GetUserByID(c.Context(), userID)
	return c.JSON(user)
}

// ChangePassword changes the current user's password
func (h *DashboardAuthHandler) ChangePassword(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uuid.UUID)

	var req struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}

	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
	}

	if req.CurrentPassword == "" || req.NewPassword == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Current password and new password are required")
	}

	ipAddress := getIPAddress(c)
	userAgent := string(c.Request().Header.UserAgent())

	err := h.authService.ChangePassword(c.Context(), userID, req.CurrentPassword, req.NewPassword, ipAddress, userAgent)
	if err != nil {
		errMsg := err.Error()
		if errMsg == "current password is incorrect" {
			return fiber.NewError(fiber.StatusUnauthorized, "Current password is incorrect")
		}
		// Check for password validation errors
		if strings.Contains(errMsg, "password must be") {
			return fiber.NewError(fiber.StatusBadRequest, errMsg)
		}
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to change password")
	}

	return c.JSON(fiber.Map{
		"message": "Password changed successfully",
	})
}

// DeleteAccount deletes the current user's account
func (h *DashboardAuthHandler) DeleteAccount(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uuid.UUID)

	var req struct {
		Password string `json:"password"`
	}

	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
	}

	if req.Password == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Password is required")
	}

	ipAddress := getIPAddress(c)
	userAgent := string(c.Request().Header.UserAgent())

	err := h.authService.DeleteAccount(c.Context(), userID, req.Password, ipAddress, userAgent)
	if err != nil {
		if err.Error() == "password is incorrect" {
			return fiber.NewError(fiber.StatusUnauthorized, "Password is incorrect")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to delete account")
	}

	return c.JSON(fiber.Map{
		"message": "Account deleted successfully",
	})
}

// SetupTOTP generates a new TOTP secret for 2FA
func (h *DashboardAuthHandler) SetupTOTP(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uuid.UUID)

	user, err := h.authService.GetUserByID(c.Context(), userID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "User not found")
	}

	// Parse optional issuer from request body
	var req struct {
		Issuer string `json:"issuer"` // Optional: custom issuer name for the QR code
	}
	// Ignore parse errors - issuer is optional and will default to config value
	_ = c.BodyParser(&req)

	secret, qrURL, err := h.authService.SetupTOTP(c.Context(), userID, user.Email, req.Issuer)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to setup 2FA")
	}

	return c.JSON(fiber.Map{
		"secret": secret,
		"qr_url": qrURL,
	})
}

// EnableTOTP enables 2FA after verifying the TOTP code
func (h *DashboardAuthHandler) EnableTOTP(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uuid.UUID)

	var req struct {
		Code string `json:"code"`
	}

	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
	}

	if req.Code == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Code is required")
	}

	ipAddress := getIPAddress(c)
	userAgent := string(c.Request().Header.UserAgent())

	backupCodes, err := h.authService.EnableTOTP(c.Context(), userID, req.Code, ipAddress, userAgent)
	if err != nil {
		if err.Error() == "invalid TOTP code" {
			return fiber.NewError(fiber.StatusUnauthorized, "Invalid 2FA code")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to enable 2FA")
	}

	return c.JSON(fiber.Map{
		"message":      "2FA enabled successfully",
		"backup_codes": backupCodes,
	})
}

// DisableTOTP disables 2FA for the current user
func (h *DashboardAuthHandler) DisableTOTP(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uuid.UUID)

	var req struct {
		Password string `json:"password"`
	}

	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
	}

	if req.Password == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Password is required")
	}

	ipAddress := getIPAddress(c)
	userAgent := string(c.Request().Header.UserAgent())

	err := h.authService.DisableTOTP(c.Context(), userID, req.Password, ipAddress, userAgent)
	if err != nil {
		if err.Error() == "password is incorrect" {
			return fiber.NewError(fiber.StatusUnauthorized, "Password is incorrect")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to disable 2FA")
	}

	return c.JSON(fiber.Map{
		"message": "2FA disabled successfully",
	})
}

// RequireDashboardAuth is a middleware that requires dashboard authentication
func (h *DashboardAuthHandler) RequireDashboardAuth(c *fiber.Ctx) error {
	authHeader := c.Get("Authorization")
	if authHeader == "" {
		return fiber.NewError(fiber.StatusUnauthorized, "Missing authorization header")
	}

	// Extract token from "Bearer <token>"
	var token string
	if strings.HasPrefix(authHeader, "Bearer ") {
		token = authHeader[7:]
	} else {
		return fiber.NewError(fiber.StatusUnauthorized, "Invalid authorization header")
	}

	// Verify token and extract claims
	claims, err := h.jwtManager.ValidateAccessToken(token)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "Invalid token")
	}

	// Verify role is dashboard_admin
	if claims.Role != "dashboard_admin" {
		return fiber.NewError(fiber.StatusForbidden, "Insufficient permissions")
	}

	// Extract user ID
	sub := claims.Subject

	userID, err := uuid.Parse(sub)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "Invalid user ID")
	}

	// Set user ID and role in locals
	// Using "user_role" to match RLS middleware expectations
	c.Locals("user_id", userID)
	c.Locals("user_role", claims.Role)

	return c.Next()
}

// getIPAddress extracts the client IP address from the request
func getIPAddress(c *fiber.Ctx) net.IP {
	// Try X-Forwarded-For header first (for proxies)
	xff := c.Get("X-Forwarded-For")
	if xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			ip := strings.TrimSpace(ips[0])
			return net.ParseIP(ip)
		}
	}

	// Try X-Real-IP header
	xri := c.Get("X-Real-IP")
	if xri != "" {
		return net.ParseIP(xri)
	}

	// Fall back to RemoteAddr (IP() method for Fiber)
	return net.ParseIP(c.IP())
}

// SSOProvider represents an SSO provider available for dashboard login
type SSOProvider struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Type     string `json:"type"`               // "oauth" or "saml"
	Provider string `json:"provider,omitempty"` // For OAuth: google, github, etc.
}

// GetSSOProviders returns the list of SSO providers available for dashboard login
func (h *DashboardAuthHandler) GetSSOProviders(c *fiber.Ctx) error {
	ctx := c.Context()
	providers := []SSOProvider{}

	// Get OAuth providers with allow_dashboard_login = true
	oauthProviders, err := h.getOAuthProvidersForDashboard(ctx)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to fetch OAuth providers")
	}
	providers = append(providers, oauthProviders...)

	// Get SAML providers with allow_dashboard_login = true
	if h.samlService != nil {
		samlProviders := h.samlService.GetProvidersForDashboard()
		for _, sp := range samlProviders {
			providers = append(providers, SSOProvider{
				ID:   sp.Name,
				Name: sp.Name,
				Type: "saml",
			})
		}
	}

	return c.JSON(fiber.Map{
		"providers": providers,
	})
}

// getOAuthProvidersForDashboard fetches OAuth providers enabled for dashboard login
func (h *DashboardAuthHandler) getOAuthProvidersForDashboard(ctx context.Context) ([]SSOProvider, error) {
	providers := []SSOProvider{}

	err := database.WrapWithServiceRole(ctx, h.db, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT id, name, provider
			FROM dashboard.oauth_providers
			WHERE enabled = true AND allow_dashboard_login = true
		`)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var id uuid.UUID
			var name, provider string
			if err := rows.Scan(&id, &name, &provider); err != nil {
				return err
			}
			providers = append(providers, SSOProvider{
				ID:       id.String(),
				Name:     name,
				Type:     "oauth",
				Provider: provider,
			})
		}
		return rows.Err()
	})
	if err != nil {
		return nil, err
	}

	return providers, nil
}

// InitiateOAuthLogin initiates an OAuth login flow for dashboard SSO
func (h *DashboardAuthHandler) InitiateOAuthLogin(c *fiber.Ctx) error {
	providerID := c.Params("provider")
	redirectTo := c.Query("redirect_to", "/")
	ctx := c.Context()

	// Fetch the OAuth provider configuration
	var clientID, clientSecret, provider string
	var scopes []string
	err := database.WrapWithServiceRole(ctx, h.db, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `
			SELECT client_id, client_secret, provider, scopes
			FROM dashboard.oauth_providers
			WHERE (id::text = $1 OR name = $1) AND enabled = true AND allow_dashboard_login = true
		`, providerID).Scan(&clientID, &clientSecret, &provider, &scopes)
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fiber.NewError(fiber.StatusNotFound, "OAuth provider not found or not enabled for dashboard login")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to fetch OAuth provider")
	}

	// Build OAuth config
	config := h.buildOAuthConfig(provider, clientID, clientSecret, scopes)
	if config == nil {
		return fiber.NewError(fiber.StatusBadRequest, "Unsupported OAuth provider")
	}

	// Generate state
	state, err := generateOAuthState()
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to generate state")
	}

	// Store state
	h.oauthStatesMu.Lock()
	h.oauthStates[state] = &dashboardOAuthState{
		Provider:   providerID,
		CreatedAt:  time.Now(),
		RedirectTo: redirectTo,
	}
	h.oauthStatesMu.Unlock()

	// Store config for callback
	h.oauthConfigsMu.Lock()
	h.oauthConfigs[state] = config
	h.oauthConfigsMu.Unlock()

	// Redirect to OAuth provider
	authURL := config.AuthCodeURL(state, oauth2.AccessTypeOffline)
	return c.Redirect(authURL)
}

// buildOAuthConfig creates an OAuth2 config for the given provider
func (h *DashboardAuthHandler) buildOAuthConfig(provider, clientID, clientSecret string, scopes []string) *oauth2.Config {
	callbackURL := h.baseURL + "/dashboard/auth/sso/oauth/" + provider + "/callback"

	var endpoint oauth2.Endpoint
	switch provider {
	case "google":
		endpoint = oauth2.Endpoint{
			AuthURL:  "https://accounts.google.com/o/oauth2/v2/auth",
			TokenURL: "https://oauth2.googleapis.com/token",
		}
		if len(scopes) == 0 {
			scopes = []string{"openid", "email", "profile"}
		}
	case "github":
		endpoint = oauth2.Endpoint{
			AuthURL:  "https://github.com/login/oauth/authorize",
			TokenURL: "https://github.com/login/oauth/access_token",
		}
		if len(scopes) == 0 {
			scopes = []string{"read:user", "user:email"}
		}
	case "microsoft":
		endpoint = oauth2.Endpoint{
			AuthURL:  "https://login.microsoftonline.com/common/oauth2/v2.0/authorize",
			TokenURL: "https://login.microsoftonline.com/common/oauth2/v2.0/token",
		}
		if len(scopes) == 0 {
			scopes = []string{"openid", "email", "profile"}
		}
	case "gitlab":
		endpoint = oauth2.Endpoint{
			AuthURL:  "https://gitlab.com/oauth/authorize",
			TokenURL: "https://gitlab.com/oauth/token",
		}
		if len(scopes) == 0 {
			scopes = []string{"read_user", "openid", "email"}
		}
	default:
		return nil
	}

	return &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  callbackURL,
		Scopes:       scopes,
		Endpoint:     endpoint,
	}
}

// OAuthCallback handles the OAuth callback for dashboard SSO
func (h *DashboardAuthHandler) OAuthCallback(c *fiber.Ctx) error {
	code := c.Query("code")
	state := c.Query("state")
	errorParam := c.Query("error")
	ctx := c.Context()

	if errorParam != "" {
		errorDesc := c.Query("error_description", errorParam)
		return c.Redirect("/login?error=" + url.QueryEscape(errorDesc))
	}

	if code == "" || state == "" {
		return c.Redirect("/login?error=" + url.QueryEscape("Missing authorization code or state"))
	}

	// Validate state
	h.oauthStatesMu.Lock()
	storedState, ok := h.oauthStates[state]
	if ok {
		delete(h.oauthStates, state)
	}
	h.oauthStatesMu.Unlock()

	if !ok || time.Since(storedState.CreatedAt) > 10*time.Minute {
		return c.Redirect("/login?error=" + url.QueryEscape("Invalid or expired state"))
	}

	// Get stored config
	h.oauthConfigsMu.Lock()
	config, ok := h.oauthConfigs[state]
	if ok {
		delete(h.oauthConfigs, state)
	}
	h.oauthConfigsMu.Unlock()

	if !ok {
		return c.Redirect("/login?error=" + url.QueryEscape("OAuth configuration not found"))
	}

	// Exchange code for token
	token, err := config.Exchange(ctx, code)
	if err != nil {
		return c.Redirect("/login?error=" + url.QueryEscape("Failed to exchange authorization code"))
	}

	// Get user info from provider
	userInfo, err := h.getUserInfoFromOAuth(ctx, config, token)
	if err != nil {
		return c.Redirect("/login?error=" + url.QueryEscape("Failed to get user info from provider"))
	}

	email, _ := userInfo["email"].(string)
	name, _ := userInfo["name"].(string)
	providerUserID, _ := userInfo["id"].(string)
	if providerUserID == "" {
		// Some providers use "sub" instead of "id"
		providerUserID, _ = userInfo["sub"].(string)
	}

	if email == "" {
		return c.Redirect("/login?error=" + url.QueryEscape("Email not provided by OAuth provider"))
	}

	// Find or create dashboard user
	providerName := "oauth:" + storedState.Provider
	user, _, err := h.authService.FindOrCreateUserBySSO(ctx, email, name, providerName, providerUserID)
	if err != nil {
		return c.Redirect("/login?error=" + url.QueryEscape("Failed to create or find user"))
	}

	// Login via SSO
	ipAddress := getIPAddress(c)
	userAgent := string(c.Request().Header.UserAgent())
	loginResp, err := h.authService.LoginViaSSO(ctx, user, ipAddress, userAgent)
	if err != nil {
		errMsg := "Login failed"
		if err.Error() == "account is locked" {
			errMsg = "Account is locked"
		} else if err.Error() == "account is inactive" {
			errMsg = "Account is inactive"
		}
		return c.Redirect("/login?error=" + url.QueryEscape(errMsg))
	}

	// Redirect with tokens in URL fragment (for SPA to capture)
	redirectURL := storedState.RedirectTo
	if redirectURL == "" || redirectURL == "/" {
		redirectURL = "/"
	}
	return c.Redirect(fmt.Sprintf("/login/callback?access_token=%s&refresh_token=%s&redirect_to=%s",
		url.QueryEscape(loginResp.AccessToken),
		url.QueryEscape(loginResp.RefreshToken),
		url.QueryEscape(redirectURL)))
}

// getUserInfoFromOAuth fetches user info from OAuth provider
func (h *DashboardAuthHandler) getUserInfoFromOAuth(ctx context.Context, config *oauth2.Config, token *oauth2.Token) (map[string]interface{}, error) {
	client := config.Client(ctx, token)

	// Determine user info URL based on endpoint
	var userInfoURL string
	switch {
	case strings.Contains(config.Endpoint.AuthURL, "google"):
		userInfoURL = "https://www.googleapis.com/oauth2/v2/userinfo"
	case strings.Contains(config.Endpoint.AuthURL, "github"):
		userInfoURL = "https://api.github.com/user"
	case strings.Contains(config.Endpoint.AuthURL, "microsoft"):
		userInfoURL = "https://graph.microsoft.com/v1.0/me"
	case strings.Contains(config.Endpoint.AuthURL, "gitlab"):
		userInfoURL = "https://gitlab.com/api/v4/user"
	default:
		return nil, errors.New("unsupported provider")
	}

	resp, err := client.Get(userInfoURL)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var userInfo map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, err
	}

	// For GitHub, we need to fetch email separately if not in profile
	if strings.Contains(config.Endpoint.AuthURL, "github") {
		if _, ok := userInfo["email"]; !ok || userInfo["email"] == nil {
			emailResp, err := client.Get("https://api.github.com/user/emails")
			if err == nil {
				defer func() { _ = emailResp.Body.Close() }()
				var emails []map[string]interface{}
				if err := json.NewDecoder(emailResp.Body).Decode(&emails); err == nil {
					for _, e := range emails {
						if primary, ok := e["primary"].(bool); ok && primary {
							userInfo["email"] = e["email"]
							break
						}
					}
				}
			}
		}
	}

	return userInfo, nil
}

// InitiateSAMLLogin initiates a SAML login flow for dashboard SSO
func (h *DashboardAuthHandler) InitiateSAMLLogin(c *fiber.Ctx) error {
	providerName := c.Params("provider")
	redirectTo := c.Query("redirect_to", "/")

	if h.samlService == nil {
		return fiber.NewError(fiber.StatusNotFound, "SAML is not configured")
	}

	// Get provider
	provider, err := h.samlService.GetProvider(providerName)
	if err != nil || provider == nil {
		return fiber.NewError(fiber.StatusNotFound, "SAML provider not found")
	}

	// Check if provider allows dashboard login
	if !provider.AllowDashboardLogin {
		return fiber.NewError(fiber.StatusForbidden, "SAML provider not enabled for dashboard login")
	}

	// Generate SAML AuthnRequest
	authURL, _, err := h.samlService.GenerateAuthRequest(providerName, redirectTo)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("Failed to create SAML request: %v", err))
	}

	return c.Redirect(authURL)
}

// SAMLACSCallback handles the SAML Assertion Consumer Service callback for dashboard SSO
func (h *DashboardAuthHandler) SAMLACSCallback(c *fiber.Ctx) error {
	ctx := c.Context()

	if h.samlService == nil {
		return c.Redirect("/login?error=" + url.QueryEscape("SAML is not configured"))
	}

	// Parse SAML response
	samlResponse := c.FormValue("SAMLResponse")
	relayState := c.FormValue("RelayState")

	if samlResponse == "" {
		return c.Redirect("/login?error=" + url.QueryEscape("Missing SAML response"))
	}

	// Find the provider from relay state or try all dashboard-enabled providers
	var assertion *auth.SAMLAssertion
	var providerName string
	var parseErr error

	// Get all dashboard-enabled SAML providers
	dashboardProviders := h.samlService.GetProvidersForDashboard()
	for _, provider := range dashboardProviders {
		assertion, parseErr = h.samlService.ParseAssertion(provider.Name, samlResponse)
		if parseErr == nil {
			providerName = provider.Name
			break
		}
	}

	if assertion == nil {
		errMsg := "SAML authentication failed"
		if parseErr != nil {
			errMsg = fmt.Sprintf("SAML authentication failed: %v", parseErr)
		}
		return c.Redirect("/login?error=" + url.QueryEscape(errMsg))
	}

	// Check if provider allows dashboard login
	provider, _ := h.samlService.GetProvider(providerName)
	if provider == nil || !provider.AllowDashboardLogin {
		return c.Redirect("/login?error=" + url.QueryEscape("SAML provider not enabled for dashboard login"))
	}

	// Extract user info using the service method
	email, name, err := h.samlService.ExtractUserInfo(providerName, assertion)
	if err != nil {
		// Fallback to manual extraction from attributes map
		email = getFirstAttribute(assertion.Attributes, "email")
		if email == "" {
			email = getFirstAttribute(assertion.Attributes, "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress")
		}
		if email == "" {
			email = assertion.NameID
		}

		name = getFirstAttribute(assertion.Attributes, "displayName")
		if name == "" {
			name = getFirstAttribute(assertion.Attributes, "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/name")
		}
		if name == "" {
			firstName := getFirstAttribute(assertion.Attributes, "firstName")
			lastName := getFirstAttribute(assertion.Attributes, "lastName")
			if firstName != "" || lastName != "" {
				name = strings.TrimSpace(firstName + " " + lastName)
			}
		}
	}

	providerUserID := assertion.NameID
	if providerUserID == "" {
		providerUserID = email
	}

	if email == "" {
		return c.Redirect("/login?error=" + url.QueryEscape("Email not provided in SAML assertion"))
	}

	// Find or create dashboard user
	samlProviderName := "saml:" + providerName
	user, _, err := h.authService.FindOrCreateUserBySSO(ctx, email, name, samlProviderName, providerUserID)
	if err != nil {
		return c.Redirect("/login?error=" + url.QueryEscape("Failed to create or find user"))
	}

	// Login via SSO
	ipAddress := getIPAddress(c)
	userAgent := string(c.Request().Header.UserAgent())
	loginResp, err := h.authService.LoginViaSSO(ctx, user, ipAddress, userAgent)
	if err != nil {
		errMsg := "Login failed"
		if err.Error() == "account is locked" {
			errMsg = "Account is locked"
		} else if err.Error() == "account is inactive" {
			errMsg = "Account is inactive"
		}
		return c.Redirect("/login?error=" + url.QueryEscape(errMsg))
	}

	// Redirect with tokens
	redirectURL := relayState
	if redirectURL == "" || redirectURL == "/" {
		redirectURL = "/"
	}
	return c.Redirect(fmt.Sprintf("/login/callback?access_token=%s&refresh_token=%s&redirect_to=%s",
		url.QueryEscape(loginResp.AccessToken),
		url.QueryEscape(loginResp.RefreshToken),
		url.QueryEscape(redirectURL)))
}

// generateOAuthState generates a random state string for OAuth
func generateOAuthState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// getFirstAttribute returns the first value for a SAML attribute or empty string
func getFirstAttribute(attributes map[string][]string, key string) string {
	if values, ok := attributes[key]; ok && len(values) > 0 {
		return values[0]
	}
	return ""
}
