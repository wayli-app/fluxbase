package api

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"
	"golang.org/x/oauth2"

	"github.com/nimbleflux/fluxbase/internal/auth"
	"github.com/nimbleflux/fluxbase/internal/crypto"
	"github.com/nimbleflux/fluxbase/internal/database"
	"github.com/nimbleflux/fluxbase/internal/email"
	apperrors "github.com/nimbleflux/fluxbase/internal/errors"
	"github.com/nimbleflux/fluxbase/internal/middleware"
)

// DashboardAuthHandler handles platform authentication endpoints
type DashboardAuthHandler struct {
	authService   *auth.DashboardAuthService
	jwtManager    *auth.JWTManager
	db            *database.Connection
	samlService   *auth.SAMLService
	emailService  email.Service
	baseURL       string
	encryptionKey []byte
	oauthHandler  *OAuthHandler // Reference to app OAuth handler for state validation

	// OAuth state storage (in production, use Redis or database)
	oauthStates    map[string]*dashboardOAuthState
	oauthStatesMu  sync.RWMutex
	oauthConfigs   map[string]*oauth2.Config
	oauthConfigsMu sync.RWMutex
}

// dashboardOAuthState holds OAuth state for dashboard SSO
type dashboardOAuthState struct {
	Provider    string
	CreatedAt   time.Time
	RedirectTo  string
	UserInfoURL *string
}

// NewDashboardAuthHandler creates a new dashboard auth handler
func NewDashboardAuthHandler(authService *auth.DashboardAuthService, jwtManager *auth.JWTManager, db *database.Connection, samlService *auth.SAMLService, emailService email.Service, baseURL string, encryptionKey []byte, oauthHandler *OAuthHandler) *DashboardAuthHandler {
	return &DashboardAuthHandler{
		authService:   authService,
		jwtManager:    jwtManager,
		db:            db,
		samlService:   samlService,
		emailService:  emailService,
		baseURL:       baseURL,
		encryptionKey: encryptionKey,
		oauthHandler:  oauthHandler,
		oauthStates:   make(map[string]*dashboardOAuthState),
		oauthConfigs:  make(map[string]*oauth2.Config),
	}
}

func (h *DashboardAuthHandler) requireAuthService(c fiber.Ctx) error {
	if h.authService == nil {
		return SendNotInitialized(c, "Auth service")
	}
	return nil
}

func (h *DashboardAuthHandler) requireJWTManager(c fiber.Ctx) error {
	if h.jwtManager == nil {
		return SendNotInitialized(c, "JWT manager")
	}
	return nil
}

func (h *DashboardAuthHandler) requireDB(c fiber.Ctx) error {
	if h.db == nil {
		return SendNotInitialized(c, "Database")
	}
	return nil
}

// Signup creates a new dashboard user account
// Only allowed if no dashboard users exist yet (first user self-registration)
func (h *DashboardAuthHandler) Signup(c fiber.Ctx) error {
	if err := h.requireAuthService(c); err != nil {
		return err
	}

	hasUsers, err := h.authService.HasExistingUsers(c.RequestCtx())
	if err != nil {
		return SendInternalError(c, "Failed to check existing users")
	}

	// If users exist, signup is disabled (must use invite instead)
	if hasUsers {
		return SendForbidden(c, "Sign-up is disabled. Please contact an administrator for an invitation.", ErrCodeAccessDenied)
	}

	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		FullName string `json:"full_name"`
	}

	if err := ParseBody(c, &req); err != nil {
		return err
	}

	if req.Email == "" || req.Password == "" || req.FullName == "" {
		return SendBadRequest(c, "Email, password, and full name are required", ErrCodeMissingField)
	}

	user, err := h.authService.CreateUser(c.RequestCtx(), req.Email, req.Password, req.FullName)
	if err != nil {
		// Check for validation errors
		errMsg := err.Error()
		if strings.Contains(errMsg, "invalid email") ||
			strings.Contains(errMsg, "invalid name") ||
			strings.Contains(errMsg, "password must be") {
			return SendBadRequest(c, errMsg, ErrCodeValidationFailed)
		}
		return SendInternalError(c, "Failed to create user")
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"user":    user,
		"message": "Account created successfully",
	})
}

// Login authenticates a dashboard user
func (h *DashboardAuthHandler) Login(c fiber.Ctx) error {
	// Check if password login is disabled
	if h.isPasswordLoginDisabled(c.RequestCtx()) {
		return SendForbidden(c, "Password login is disabled. Please use SSO to sign in.", ErrCodeFeatureDisabled)
	}

	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := ParseBody(c, &req); err != nil {
		return err
	}

	if err := h.requireAuthService(c); err != nil {
		return err
	}

	if err := h.requireDB(c); err != nil {
		return err
	}

	ipAddress := getIPAddress(c)
	userAgent := string(c.Request().Header.UserAgent())

	user, loginResp, err := h.authService.Login(c.RequestCtx(), req.Email, req.Password, ipAddress, userAgent)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || err.Error() == "invalid credentials" {
			return SendUnauthorized(c, "Invalid email or password", ErrCodeInvalidCredentials)
		}
		if err.Error() == "account is locked" {
			return SendForbidden(c, "Account is locked due to too many failed login attempts", ErrCodeAccountLocked)
		}
		if err.Error() == "account is inactive" {
			return SendForbidden(c, "Account is inactive", ErrCodeAccessDenied)
		}
		// Log the actual error for debugging
		fmt.Printf("Dashboard login error: %v\n", err)
		return SendInternalError(c, "Login failed")
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

// RefreshToken handles token refresh for dashboard users
func (h *DashboardAuthHandler) RefreshToken(c fiber.Ctx) error {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}

	if err := ParseBody(c, &req); err != nil {
		return err
	}

	if req.RefreshToken == "" {
		return SendBadRequest(c, "Refresh token is required", ErrCodeMissingField)
	}

	if err := h.requireAuthService(c); err != nil {
		return err
	}

	loginResp, err := h.authService.RefreshToken(c.RequestCtx(), req.RefreshToken)
	if err != nil {
		return SendUnauthorized(c, "Invalid or expired refresh token", ErrCodeInvalidToken)
	}

	return c.JSON(fiber.Map{
		"access_token":  loginResp.AccessToken,
		"refresh_token": loginResp.RefreshToken,
		"expires_in":    loginResp.ExpiresIn,
	})
}

// VerifyTOTP verifies a TOTP code during login
func (h *DashboardAuthHandler) VerifyTOTP(c fiber.Ctx) error {
	var req struct {
		UserID string `json:"user_id"`
		Code   string `json:"code"`
	}

	if err := ParseBody(c, &req); err != nil {
		return err
	}

	if req.UserID == "" || req.Code == "" {
		return SendBadRequest(c, "User ID and code are required", ErrCodeMissingField)
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		return SendBadRequest(c, "Invalid user ID", ErrCodeInvalidID)
	}

	if err := h.requireAuthService(c); err != nil {
		return err
	}

	if err := h.requireJWTManager(c); err != nil {
		return err
	}

	err = h.authService.VerifyTOTP(c.RequestCtx(), userID, req.Code)
	if err != nil {
		return SendUnauthorized(c, "Invalid 2FA code", ErrCodeInvalidCredentials)
	}

	// Get user after successful 2FA
	user, err := h.authService.GetUserByID(c.RequestCtx(), userID)
	if err != nil {
		return SendInternalError(c, "Failed to fetch user")
	}

	// Generate JWT tokens - platform admins get instance_admin role
	accessToken, refreshToken, _, err := h.jwtManager.GenerateTokenPair(user.ID.String(), user.Email, "instance_admin", nil, nil)
	if err != nil {
		return SendInternalError(c, "Failed to generate tokens")
	}

	return c.JSON(fiber.Map{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"expires_in":    86400, // 24 hours
		"user":          user,
	})
}

// GetCurrentUser returns the currently authenticated dashboard user
func (h *DashboardAuthHandler) GetCurrentUser(c fiber.Ctx) error {
	userID, _ := uuid.Parse(middleware.GetUserID(c))

	user, err := h.authService.GetUserByID(c.RequestCtx(), userID)
	if err != nil {
		return SendNotFound(c, "User not found")
	}

	// Set role from JWT (RequireDashboardAuth middleware validates this is "instance_admin")
	user.Role = "instance_admin"

	return c.JSON(user)
}

// UpdateProfile updates the current user's profile
func (h *DashboardAuthHandler) UpdateProfile(c fiber.Ctx) error {
	userID, _ := uuid.Parse(middleware.GetUserID(c))

	var req struct {
		FullName  string  `json:"full_name"`
		AvatarURL *string `json:"avatar_url"`
	}

	if err := ParseBody(c, &req); err != nil {
		return err
	}

	if req.FullName == "" {
		return SendBadRequest(c, "Full name is required", ErrCodeMissingField)
	}

	if err := h.requireAuthService(c); err != nil {
		return err
	}

	err := h.authService.UpdateProfile(c.RequestCtx(), userID, req.FullName, req.AvatarURL)
	if err != nil {
		// Check for validation errors
		errMsg := err.Error()
		if strings.Contains(errMsg, "invalid name") ||
			strings.Contains(errMsg, "invalid avatar URL") {
			return SendBadRequest(c, errMsg, ErrCodeValidationFailed)
		}
		return SendInternalError(c, "Failed to update profile")
	}

	user, _ := h.authService.GetUserByID(c.RequestCtx(), userID)
	return c.JSON(user)
}

// ChangePassword changes the current user's password
func (h *DashboardAuthHandler) ChangePassword(c fiber.Ctx) error {
	userID, _ := uuid.Parse(middleware.GetUserID(c))

	var req struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}

	if err := ParseBody(c, &req); err != nil {
		return err
	}

	if req.CurrentPassword == "" || req.NewPassword == "" {
		return SendBadRequest(c, "Current password and new password are required", ErrCodeMissingField)
	}

	if err := h.requireAuthService(c); err != nil {
		return err
	}

	ipAddress := getIPAddress(c)
	userAgent := string(c.Request().Header.UserAgent())

	err := h.authService.ChangePassword(c.RequestCtx(), userID, req.CurrentPassword, req.NewPassword, ipAddress, userAgent)
	if err != nil {
		errMsg := err.Error()
		if errMsg == "current password is incorrect" {
			return SendUnauthorized(c, "Current password is incorrect", ErrCodeInvalidCredentials)
		}
		// Check for password validation errors
		if strings.Contains(errMsg, "password must be") {
			return SendBadRequest(c, errMsg, ErrCodeValidationFailed)
		}
		return SendInternalError(c, "Failed to change password")
	}

	return apperrors.SendSuccess(c, "Password changed successfully")
}

// DeleteAccount deletes the current user's account
func (h *DashboardAuthHandler) DeleteAccount(c fiber.Ctx) error {
	userID, _ := uuid.Parse(middleware.GetUserID(c))

	var req struct {
		Password string `json:"password"`
	}

	if err := ParseBody(c, &req); err != nil {
		return err
	}

	if req.Password == "" {
		return SendBadRequest(c, "Password is required", ErrCodeMissingField)
	}

	if err := h.requireAuthService(c); err != nil {
		return err
	}

	ipAddress := getIPAddress(c)
	userAgent := string(c.Request().Header.UserAgent())

	err := h.authService.DeleteAccount(c.RequestCtx(), userID, req.Password, ipAddress, userAgent)
	if err != nil {
		if err.Error() == "password is incorrect" {
			return SendUnauthorized(c, "Password is incorrect", ErrCodeInvalidCredentials)
		}
		return SendInternalError(c, "Failed to delete account")
	}

	return apperrors.SendSuccess(c, "Account deleted successfully")
}

// SetupTOTP generates a new TOTP secret for 2FA
func (h *DashboardAuthHandler) SetupTOTP(c fiber.Ctx) error {
	userID, _ := uuid.Parse(middleware.GetUserID(c))

	if err := h.requireAuthService(c); err != nil {
		return err
	}

	user, err := h.authService.GetUserByID(c.RequestCtx(), userID)
	if err != nil {
		return SendNotFound(c, "User not found")
	}

	// Parse optional issuer from request body
	var req struct {
		Issuer string `json:"issuer"` // Optional: custom issuer name for the QR code
	}
	// Ignore parse errors - issuer is optional and will default to config value
	_ = c.Bind().Body(&req)

	secret, qrURL, err := h.authService.SetupTOTP(c.RequestCtx(), userID, user.Email, req.Issuer)
	if err != nil {
		return SendInternalError(c, "Failed to setup 2FA")
	}

	return c.JSON(fiber.Map{
		"secret": secret,
		"qr_url": qrURL,
	})
}

// EnableTOTP enables 2FA after verifying the TOTP code
func (h *DashboardAuthHandler) EnableTOTP(c fiber.Ctx) error {
	userID, _ := uuid.Parse(middleware.GetUserID(c))

	var req struct {
		Code string `json:"code"`
	}

	if err := ParseBody(c, &req); err != nil {
		return err
	}

	if req.Code == "" {
		return SendBadRequest(c, "Code is required", ErrCodeMissingField)
	}

	if err := h.requireAuthService(c); err != nil {
		return err
	}

	ipAddress := getIPAddress(c)
	userAgent := string(c.Request().Header.UserAgent())

	backupCodes, err := h.authService.EnableTOTP(c.RequestCtx(), userID, req.Code, ipAddress, userAgent)
	if err != nil {
		if err.Error() == "invalid TOTP code" {
			return SendUnauthorized(c, "Invalid 2FA code", ErrCodeInvalidCredentials)
		}
		return SendInternalError(c, "Failed to enable 2FA")
	}

	return c.JSON(fiber.Map{
		"message":      "2FA enabled successfully",
		"backup_codes": backupCodes,
	})
}

// DisableTOTP disables 2FA for the current user
func (h *DashboardAuthHandler) DisableTOTP(c fiber.Ctx) error {
	userID, _ := uuid.Parse(middleware.GetUserID(c))

	var req struct {
		Password string `json:"password"`
	}

	if err := ParseBody(c, &req); err != nil {
		return err
	}

	if req.Password == "" {
		return SendBadRequest(c, "Password is required", ErrCodeMissingField)
	}

	if err := h.requireAuthService(c); err != nil {
		return err
	}

	ipAddress := getIPAddress(c)
	userAgent := string(c.Request().Header.UserAgent())

	err := h.authService.DisableTOTP(c.RequestCtx(), userID, req.Password, ipAddress, userAgent)
	if err != nil {
		if err.Error() == "password is incorrect" {
			return SendUnauthorized(c, "Password is incorrect", ErrCodeInvalidCredentials)
		}
		return SendInternalError(c, "Failed to disable 2FA")
	}

	return apperrors.SendSuccess(c, "2FA disabled successfully")
}

// RequestPasswordReset initiates a password reset for a dashboard user
func (h *DashboardAuthHandler) RequestPasswordReset(c fiber.Ctx) error {
	// Check if email service is configured
	if h.emailService == nil {
		return SendBadRequest(c, "Email service is not configured. Please configure an email provider to enable password reset.", ErrCodeFeatureDisabled)
	}

	var req struct {
		Email string `json:"email"`
	}

	if err := ParseBody(c, &req); err != nil {
		return err
	}

	if req.Email == "" {
		return SendBadRequest(c, "Email is required", ErrCodeMissingField)
	}

	if err := h.requireAuthService(c); err != nil {
		return err
	}

	token, err := h.authService.RequestPasswordReset(c.RequestCtx(), req.Email)
	if err != nil {
		// Log the error but don't reveal details to user
		log.Error().Err(err).Str("email", req.Email).Msg("Failed to request password reset")
		// Still return success to prevent email enumeration
	}

	// If we got a token, send the password reset email
	if token != "" {
		resetLink := h.baseURL + "/admin/reset-password?token=" + token
		if err := h.emailService.SendPasswordReset(c.RequestCtx(), req.Email, token, resetLink); err != nil {
			log.Error().Err(err).Str("email", req.Email).Msg("Failed to send password reset email")
			// Don't return error to prevent email enumeration
		} else {
			log.Info().Str("email", req.Email).Msg("Password reset email sent")
		}
	}

	// Always return success to prevent email enumeration
	return c.JSON(fiber.Map{
		"message": "If an account with that email exists, a password reset link has been sent.",
	})
}

// VerifyPasswordResetToken verifies a password reset token is valid
func (h *DashboardAuthHandler) VerifyPasswordResetToken(c fiber.Ctx) error {
	var req struct {
		Token string `json:"token"`
	}

	if err := ParseBody(c, &req); err != nil {
		return err
	}

	if req.Token == "" {
		return SendBadRequest(c, "Token is required", ErrCodeMissingField)
	}

	if err := h.requireAuthService(c); err != nil {
		return err
	}

	valid, err := h.authService.VerifyPasswordResetToken(c.RequestCtx(), req.Token)
	if err != nil {
		return SendInternalError(c, "Failed to verify token")
	}

	if !valid {
		return c.JSON(fiber.Map{
			"valid":   false,
			"message": "Invalid or expired token",
		})
	}

	return c.JSON(fiber.Map{
		"valid":   true,
		"message": "Token is valid",
	})
}

// ConfirmPasswordReset resets the password using a valid reset token
func (h *DashboardAuthHandler) ConfirmPasswordReset(c fiber.Ctx) error {
	var req struct {
		Token       string `json:"token"`
		NewPassword string `json:"new_password"`
	}

	if err := ParseBody(c, &req); err != nil {
		return err
	}

	if req.Token == "" || req.NewPassword == "" {
		return SendBadRequest(c, "Token and new password are required", ErrCodeMissingField)
	}

	if err := h.requireAuthService(c); err != nil {
		return err
	}

	err := h.authService.ResetPassword(c.RequestCtx(), req.Token, req.NewPassword)
	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "invalid or expired") {
			return SendBadRequest(c, "Invalid or expired password reset token", ErrCodeInvalidToken)
		}
		if strings.Contains(errMsg, "password must be") {
			return SendBadRequest(c, errMsg, ErrCodeValidationFailed)
		}
		return SendInternalError(c, "Failed to reset password")
	}

	return apperrors.SendSuccess(c, "Password reset successfully")
}

// RequireDashboardAuth is a middleware that requires dashboard authentication
func (h *DashboardAuthHandler) RequireDashboardAuth(c fiber.Ctx) error {
	authHeader := c.Get("Authorization")
	if authHeader == "" {
		return SendUnauthorized(c, "Missing authorization header", ErrCodeMissingAuth)
	}

	// Extract token from "Bearer <token>"
	var token string
	if strings.HasPrefix(authHeader, "Bearer ") {
		token = authHeader[7:]
	} else {
		return SendUnauthorized(c, "Invalid authorization header", ErrCodeMissingAuth)
	}

	if err := h.requireJWTManager(c); err != nil {
		return err
	}

	claims, err := h.jwtManager.ValidateAccessToken(token)
	if err != nil {
		return SendUnauthorized(c, "Invalid token", ErrCodeInvalidToken)
	}

	// Verify role is instance_admin
	if claims.Role != "instance_admin" {
		return SendForbidden(c, "Insufficient permissions", ErrCodeInsufficientPermissions)
	}

	// Extract user ID
	sub := claims.Subject

	userID, err := uuid.Parse(sub)
	if err != nil {
		return SendUnauthorized(c, "Invalid user ID", ErrCodeInvalidUserID)
	}

	// Set user ID and role in locals
	// Using "user_role" to match RLS middleware expectations
	c.Locals("user_id", userID.String())
	c.Locals("user_role", claims.Role)

	return c.Next()
}

// getIPAddress extracts the client IP address from the request.
// Security note: This function only trusts X-Forwarded-For and X-Real-IP headers
// when the request comes from a private IP range (likely a trusted proxy/load balancer).
// For direct connections, it uses the actual connection IP to prevent spoofing.
func getIPAddress(c fiber.Ctx) net.IP {
	// Get the direct connection IP first
	directIP := net.ParseIP(c.IP())

	// Only trust proxy headers if the connection is from a private/trusted IP range
	// This is a heuristic for detecting trusted proxies (internal load balancers, etc.)
	if isPrivateOrLocalIP(directIP) {
		// Try X-Forwarded-For header first (for proxies)
		xff := c.Get("X-Forwarded-For")
		if xff != "" {
			ips := strings.Split(xff, ",")
			if len(ips) > 0 {
				// Take the leftmost IP (original client)
				// Subsequent IPs are proxies the request passed through
				ip := strings.TrimSpace(ips[0])
				if parsed := net.ParseIP(ip); parsed != nil {
					return parsed
				}
			}
		}

		// Try X-Real-IP header
		xri := c.Get("X-Real-IP")
		if xri != "" {
			if parsed := net.ParseIP(xri); parsed != nil {
				return parsed
			}
		}
	}

	// Fall back to direct connection IP
	return directIP
}

// isPrivateOrLocalIP checks if an IP is in a private range or is localhost
func isPrivateOrLocalIP(ip net.IP) bool {
	if ip == nil {
		return false
	}

	// Check for loopback
	if ip.IsLoopback() {
		return true
	}

	// Check for private ranges
	if ip.IsPrivate() {
		return true
	}

	// Check for link-local addresses
	if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}

	// Check for unspecified address (0.0.0.0 or ::) - used in test environments
	if ip.IsUnspecified() {
		return true
	}

	return false
}

// isPasswordLoginDisabled checks if password login is disabled for the dashboard
// This can be overridden by the FLUXBASE_DASHBOARD_FORCE_PASSWORD_LOGIN environment variable
func (h *DashboardAuthHandler) isPasswordLoginDisabled(ctx context.Context) bool {
	// Emergency override via environment variable
	if os.Getenv("FLUXBASE_DASHBOARD_FORCE_PASSWORD_LOGIN") == "true" {
		return false // Password login forced enabled
	}

	// Check database setting
	var disabled bool
	err := database.WrapWithServiceRole(ctx, h.db, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `
			SELECT COALESCE(value::boolean, false)
			FROM app.settings
			WHERE key = 'disable_dashboard_password_login' AND category = 'auth'
		`).Scan(&disabled)
	})
	if err != nil {
		// If setting doesn't exist or error, default to allowing password login
		return false
	}

	return disabled
}

// SSOProvider represents an SSO provider available for dashboard login
type SSOProvider struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Type     string `json:"type"`               // "oauth" or "saml"
	Provider string `json:"provider,omitempty"` // For OAuth: google, github, etc.
}

// GetSSOProviders returns the list of SSO providers available for dashboard login
func (h *DashboardAuthHandler) GetSSOProviders(c fiber.Ctx) error {
	ctx := c.RequestCtx()
	providers := []SSOProvider{}

	if err := h.requireDB(c); err != nil {
		return err
	}

	tenantID := middleware.GetTenantIDFromContext(c)
	oauthProviders, err := h.getOAuthProvidersForDashboard(ctx, tenantID)
	if err != nil {
		return SendInternalError(c, "Failed to fetch OAuth providers")
	}
	providers = append(providers, oauthProviders...)

	// Get SAML providers with allow_dashboard_login = true
	if h.samlService != nil {
		samlProviders := h.samlService.GetProvidersForDashboardWithTenant(c.RequestCtx(), middleware.GetTenantIDFromContext(c))
		for _, sp := range samlProviders {
			providers = append(providers, SSOProvider{
				ID:   sp.Name,
				Name: sp.Name,
				Type: "saml",
			})
		}
	}

	// Check if password login is disabled
	passwordLoginDisabled := h.isPasswordLoginDisabled(ctx)

	return c.JSON(fiber.Map{
		"providers":               providers,
		"password_login_disabled": passwordLoginDisabled,
	})
}

// getOAuthProvidersForDashboard fetches OAuth providers enabled for dashboard login
func (h *DashboardAuthHandler) getOAuthProvidersForDashboard(ctx context.Context, tenantID string) ([]SSOProvider, error) {
	providers := []SSOProvider{}

	err := database.WrapWithServiceRole(ctx, h.db, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT id, display_name, provider_name
		FROM platform.oauth_providers
		WHERE enabled = true AND allow_dashboard_login = true
		`)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var id uuid.UUID
			var displayName, providerName string
			if err := rows.Scan(&id, &displayName, &providerName); err != nil {
				return err
			}
			providers = append(providers, SSOProvider{
				ID:       providerName, // Use provider_name as ID for URL routing
				Name:     displayName,
				Type:     "oauth",
				Provider: providerName,
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
func (h *DashboardAuthHandler) InitiateOAuthLogin(c fiber.Ctx) error {
	providerID := c.Params("provider")
	redirectTo := c.Query("redirect_to", "/")
	ctx := c.RequestCtx()

	// Fetch the OAuth provider configuration
	var clientID, clientSecret, providerName string
	var scopes []string
	var isCustom bool
	var isEncrypted bool
	var authURL, tokenURL, userInfoURL *string
	err := database.WrapWithServiceRole(ctx, h.db, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `
			SELECT client_id, client_secret, provider_name, scopes,
			       is_custom, authorization_url, token_url, user_info_url,
			       COALESCE(is_encrypted, false) AS is_encrypted
		FROM platform.oauth_providers
		WHERE (id::text = $1 OR provider_name = $1) AND enabled = true AND allow_dashboard_login = true
		`, providerID).Scan(&clientID, &clientSecret, &providerName, &scopes, &isCustom, &authURL, &tokenURL, &userInfoURL, &isEncrypted)
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			log.Warn().
				Str("provider_id", providerID).
				Msg("OAuth provider not found or not enabled for dashboard login")
			return SendNotFound(c, "OAuth provider not found or not enabled for dashboard login")
		}
		log.Error().Err(err).Str("provider_id", providerID).Msg("Failed to fetch OAuth provider")
		return SendInternalError(c, "Failed to fetch OAuth provider")
	}

	log.Debug().
		Str("provider_id", providerID).
		Str("provider_name", providerName).
		Bool("is_custom", isCustom).
		Bool("has_auth_url", authURL != nil).
		Bool("has_token_url", tokenURL != nil).
		Msg("OAuth provider fetched for dashboard login")

	// Decrypt client secret if encrypted
	if isEncrypted && clientSecret != "" {
		decryptedSecret, decErr := crypto.DecryptWithBytesKey(clientSecret, h.encryptionKey)
		if decErr != nil {
			log.Error().Err(decErr).Str("provider", providerName).Msg("Failed to decrypt client secret")
			return SendInternalError(c, "Failed to decrypt client secret")
		}
		clientSecret = decryptedSecret
	}

	// Build OAuth config
	config := h.buildOAuthConfig(providerName, clientID, clientSecret, scopes, isCustom, authURL, tokenURL)
	if config == nil {
		log.Warn().
			Str("provider_name", providerName).
			Bool("is_custom", isCustom).
			Msg("Failed to build OAuth config - unsupported provider")
		return SendBadRequest(c, "Unsupported OAuth provider", ErrCodeInvalidInput)
	}

	// Generate state
	state, err := generateOAuthState()
	if err != nil {
		return SendInternalError(c, "Failed to generate state")
	}

	// Store state
	h.oauthStatesMu.Lock()
	h.oauthStates[state] = &dashboardOAuthState{
		Provider:    providerID,
		CreatedAt:   time.Now(),
		RedirectTo:  redirectTo,
		UserInfoURL: userInfoURL,
	}
	h.oauthStatesMu.Unlock()

	// Store config for callback
	h.oauthConfigsMu.Lock()
	h.oauthConfigs[state] = config
	h.oauthConfigsMu.Unlock()

	// Build auth URL options
	authURLOpts := []oauth2.AuthCodeOption{oauth2.AccessTypeOffline}

	// Add prompt=consent for Google to ensure refresh tokens on subsequent logins
	if strings.ToLower(providerName) == "google" {
		authURLOpts = append(authURLOpts, oauth2.SetAuthURLParam("prompt", "consent"))
	}

	// Redirect to OAuth provider
	authorizeURL := config.AuthCodeURL(state, authURLOpts...)

	log.Debug().
		Str("state", state).
		Str("provider", providerName).
		Str("authorize_url", authorizeURL).
		Msg("Dashboard OAuth login initiated")

	// Return JSON with authorization URL (client handles the redirect)
	return c.JSON(fiber.Map{
		"url":      authorizeURL,
		"provider": providerID,
	})
}

// buildOAuthConfig creates an OAuth2 config for the given provider
func (h *DashboardAuthHandler) buildOAuthConfig(provider, clientID, clientSecret string, scopes []string, isCustom bool, customAuthURL, customTokenURL *string) *oauth2.Config {
	callbackURL := h.baseURL + "/dashboard/auth/sso/oauth/" + provider + "/callback"

	var endpoint oauth2.Endpoint

	// If custom provider with URLs, use them
	if isCustom && customAuthURL != nil && customTokenURL != nil {
		endpoint = oauth2.Endpoint{
			AuthURL:  *customAuthURL,
			TokenURL: *customTokenURL,
		}
	} else {
		// Fall back to standard providers
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
				scopes = []string{"openid", "email", "profile", "offline_access"}
			}
		case "gitlab":
			endpoint = oauth2.Endpoint{
				AuthURL:  "https://gitlab.com/oauth/authorize",
				TokenURL: "https://gitlab.com/oauth/token",
			}
			if len(scopes) == 0 {
				scopes = []string{"read_user", "openid", "email", "offline_access"}
			}
		default:
			return nil
		}
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
func (h *DashboardAuthHandler) OAuthCallback(c fiber.Ctx) error {
	code := c.Query("code")
	state := c.Query("state")
	errorParam := c.Query("error")
	ctx := c.RequestCtx()

	codePreview := code
	if len(code) > 10 {
		codePreview = code[:10] + "..."
	}
	providerID := c.Params("provider")
	log.Debug().
		Str("state", state).
		Str("code", codePreview).
		Str("provider", providerID).
		Msg("Dashboard OAuth callback received")

	// Validate state from dashboard's own state store
	h.oauthStatesMu.Lock()
	dashState, stateExists := h.oauthStates[state]
	if stateExists {
		delete(h.oauthStates, state)
	}
	h.oauthStatesMu.Unlock()

	if !stateExists || dashState == nil {
		log.Warn().
			Str("state", state).
			Msg("Invalid or missing OAuth state in dashboard callback")
		return c.Redirect().To("/admin/login?error=" + url.QueryEscape("Invalid or expired state"))
	}

	// Retrieve stored OAuth config
	h.oauthConfigsMu.Lock()
	config, configExists := h.oauthConfigs[state]
	if configExists {
		delete(h.oauthConfigs, state)
	}
	h.oauthConfigsMu.Unlock()

	if !configExists || config == nil {
		log.Warn().
			Str("state", state).
			Msg("Missing OAuth config for dashboard callback")
		return c.Redirect().To("/admin/login?error=" + url.QueryEscape("OAuth configuration not found"))
	}

	// Verify provider matches the one from the initiation
	if providerID != "" && dashState.Provider != providerID {
		log.Warn().
			Str("url_provider", providerID).
			Str("state_provider", dashState.Provider).
			Msg("Provider mismatch in dashboard OAuth callback")
		return c.Redirect().To("/admin/login?error=" + url.QueryEscape("Provider mismatch"))
	}

	// This is a dashboard OAuth callback, process it
	if errorParam != "" {
		errorDesc := c.Query("error_description", errorParam)
		return c.Redirect().To("/admin/login?error=" + url.QueryEscape(errorDesc))
	}

	if code == "" {
		return c.Redirect().To("/admin/login?error=" + url.QueryEscape("Missing authorization code"))
	}

	userInfoURL := dashState.UserInfoURL

	// Log OAuth config details for debugging
	log.Debug().
		Str("provider", providerID).
		Str("redirect_uri", config.RedirectURL).
		Str("client_id", config.ClientID).
		Str("auth_url", config.Endpoint.AuthURL).
		Str("token_url", config.Endpoint.TokenURL).
		Msg("OAuth config for token exchange")

	// Exchange code for token
	token, err := config.Exchange(ctx, code)
	if err != nil {
		log.Error().
			Err(err).
			Str("provider", providerID).
			Str("redirect_uri", config.RedirectURL).
			Str("config_redirect_uri", config.RedirectURL).
			Msg("Failed to exchange OAuth authorization code")
		return c.Redirect().To("/admin/login?error=" + url.QueryEscape("Failed to exchange authorization code"))
	}

	// Fetch provider configuration for RBAC validation
	var requiredClaimsJSON, deniedClaimsJSON []byte
	var providerDisplayName string
	err = database.WrapWithServiceRole(ctx, h.db, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `
			SELECT display_name, required_claims, denied_claims
		FROM platform.oauth_providers
		WHERE (id::text = $1 OR provider_name = $1) AND enabled = true AND allow_dashboard_login = true
		`, providerID).Scan(&providerDisplayName, &requiredClaimsJSON, &deniedClaimsJSON)
	})
	if err != nil {
		log.Warn().
			Err(err).
			Str("provider", providerID).
			Msg("Failed to fetch OAuth provider config for RBAC validation")
		return c.Redirect().To("/admin/login?error=" + url.QueryEscape("OAuth provider configuration error"))
	}

	// Get user info from provider (includes ID token claims)
	userInfo, err := h.getUserInfoFromOAuth(ctx, config, token, userInfoURL)
	if err != nil {
		return c.Redirect().To("/admin/login?error=" + url.QueryEscape("Failed to get user info from provider"))
	}

	// Extract ID token claims (if available)
	var idTokenClaims map[string]interface{}
	if idTokenRaw, ok := token.Extra("id_token").(string); ok && idTokenRaw != "" {
		// Parse ID token (simple base64 decode of payload)
		idTokenClaims, err = parseIDTokenClaims(idTokenRaw)
		if err != nil {
			log.Warn().
				Err(err).
				Str("provider", providerID).
				Msg("Failed to parse ID token claims")
			// Use userInfo as fallback
			idTokenClaims = userInfo
		}
	} else {
		// Use userInfo as fallback if no ID token
		idTokenClaims = userInfo
	}

	// RBAC: Validate OAuth claims if configured
	if requiredClaimsJSON != nil || deniedClaimsJSON != nil {
		var requiredClaims, deniedClaims map[string][]string
		if requiredClaimsJSON != nil {
			if err := json.Unmarshal(requiredClaimsJSON, &requiredClaims); err != nil {
				log.Warn().Err(err).Msg("Failed to parse required_claims JSON")
			}
		}
		if deniedClaimsJSON != nil {
			if err := json.Unmarshal(deniedClaimsJSON, &deniedClaims); err != nil {
				log.Warn().Err(err).Msg("Failed to parse denied_claims JSON")
			}
		}

		provider := &auth.OAuthProviderRBAC{
			Name:           providerDisplayName,
			RequiredClaims: requiredClaims,
			DeniedClaims:   deniedClaims,
		}

		if err := auth.ValidateOAuthClaims(provider, idTokenClaims); err != nil {
			log.Warn().
				Err(err).
				Str("provider", providerID).
				Interface("claims", idTokenClaims).
				Msg("Dashboard OAuth access denied due to claims validation")
			return c.Redirect().To("/admin/login?error=" + url.QueryEscape(err.Error()))
		}
	}

	email, _ := userInfo["email"].(string)
	name, _ := userInfo["name"].(string)
	// Capitalize the first letter of each word in the name
	name = capitalizeWords(name)
	providerUserID, _ := userInfo["id"].(string)
	if providerUserID == "" {
		// Some providers use "sub" instead of "id"
		providerUserID, _ = userInfo["sub"].(string)
	}

	if email == "" {
		return c.Redirect().To("/admin/login?error=" + url.QueryEscape("Email not provided by OAuth provider"))
	}

	// Find or create dashboard user
	providerName := "oauth:" + providerID
	user, _, err := h.authService.FindOrCreateUserBySSO(ctx, email, name, providerName, providerUserID)
	if err != nil {
		log.Error().
			Err(err).
			Str("email", email).
			Str("provider", providerName).
			Str("provider_user_id", providerUserID).
			Msg("Failed to create or find dashboard user via SSO")
		return c.Redirect().To("/admin/login?error=" + url.QueryEscape("Failed to create or find user"))
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
		return c.Redirect().To("/admin/login?error=" + url.QueryEscape(errMsg))
	}

	// Redirect with tokens in URL fragment (for SPA to capture)
	redirectURL := dashState.RedirectTo
	if redirectURL == "" || redirectURL == "/" {
		redirectURL = "/admin"
	}
	return c.Redirect().To(fmt.Sprintf("/admin/login/callback#access_token=%s&refresh_token=%s&redirect_to=%s",
		url.QueryEscape(loginResp.AccessToken),
		url.QueryEscape(loginResp.RefreshToken),
		url.QueryEscape(redirectURL)))
}

// parseIDTokenClaims parses JWT ID token and extracts claims
// This is a simple implementation without signature verification (already verified by OAuth provider)
func parseIDTokenClaims(idToken string) (map[string]interface{}, error) {
	parts := strings.Split(idToken, ".")
	if len(parts) != 3 {
		return nil, errors.New("invalid ID token format")
	}

	// Decode payload (second part)
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("failed to decode ID token payload: %w", err)
	}

	var claims map[string]interface{}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, fmt.Errorf("failed to unmarshal ID token claims: %w", err)
	}

	return claims, nil
}

// getUserInfoFromOAuth fetches user info from OAuth provider
func (h *DashboardAuthHandler) getUserInfoFromOAuth(ctx context.Context, config *oauth2.Config, token *oauth2.Token, customUserInfoURL *string) (map[string]interface{}, error) {
	client := config.Client(ctx, token)

	// Determine user info URL - use custom URL if provided, otherwise use standard provider URLs
	var userInfoURL string
	if customUserInfoURL != nil && *customUserInfoURL != "" {
		userInfoURL = *customUserInfoURL
	} else {
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
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, userInfoURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
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
			emailReq, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/user/emails", nil)
			if err == nil {
				emailResp, err := client.Do(emailReq)
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
	}

	return userInfo, nil
}

// InitiateSAMLLogin initiates a SAML login flow for dashboard SSO
func (h *DashboardAuthHandler) InitiateSAMLLogin(c fiber.Ctx) error {
	providerIDOrName := c.Params("provider")
	redirectTo := c.Query("redirect_to", "/")
	ctx := c.RequestCtx()

	if h.samlService == nil {
		return SendNotInitialized(c, "SAML service")
	}

	if err := h.requireDB(c); err != nil {
		return err
	}

	var providerName string
	var allowDashboardLogin bool
	err := database.WrapWithServiceRole(ctx, h.db, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `
			SELECT name, COALESCE(allow_dashboard_login, false)
			FROM auth.saml_providers
			WHERE (id::text = $1 OR name = $1) AND enabled = true
		`, providerIDOrName).Scan(&providerName, &allowDashboardLogin)
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			log.Warn().
				Str("provider_id", providerIDOrName).
				Msg("SAML provider not found for dashboard login")
			return SendNotFound(c, "SAML provider not found or not enabled for dashboard login")
		}
		return SendInternalError(c, "Failed to fetch SAML provider")
	}

	// Check if provider allows dashboard login
	if !allowDashboardLogin {
		log.Warn().
			Str("provider", providerName).
			Msg("SAML provider not enabled for dashboard login")
		return SendForbidden(c, "SAML provider not enabled for dashboard login", ErrCodeAccessDenied)
	}

	// Get provider from service (by name)
	provider, err := h.samlService.GetProvider(providerName)
	if err != nil || provider == nil {
		return SendNotFound(c, "SAML provider not found")
	}

	// Generate SAML AuthnRequest
	authURL, _, err := h.samlService.GenerateAuthRequest(providerName, redirectTo)
	if err != nil {
		return SendInternalError(c, "Failed to create SAML request")
	}

	return c.Redirect().To(authURL)
}

// SAMLACSCallback handles the SAML Assertion Consumer Service callback for dashboard SSO
func (h *DashboardAuthHandler) SAMLACSCallback(c fiber.Ctx) error {
	ctx := c.RequestCtx()

	if h.samlService == nil {
		return SendNotInitialized(c, "SAML service")
	}

	samlResponse := c.FormValue("SAMLResponse")
	relayState := c.FormValue("RelayState")

	if samlResponse == "" {
		return c.Redirect().To("/admin/login?error=" + url.QueryEscape("Missing SAML response"))
	}

	// Find the provider from relay state or try all dashboard-enabled providers
	var assertion *auth.SAMLAssertion
	var providerName string
	var parseErr error

	// Get all dashboard-enabled SAML providers
	dashboardProviders := h.samlService.GetProvidersForDashboardWithTenant(c.RequestCtx(), middleware.GetTenantIDFromContext(c))

	// If no dashboard providers configured
	if len(dashboardProviders) == 0 {
		log.Warn().Msg("No SAML providers enabled for dashboard login")
		return c.Redirect().To("/admin/login?error=" + url.QueryEscape("No SAML providers configured for dashboard"))
	}

	for _, provider := range dashboardProviders {
		assertion, parseErr = h.samlService.ParseAssertion(provider.Name, samlResponse)
		if parseErr == nil {
			providerName = provider.Name
			break
		}
	}

	if assertion == nil {
		log.Warn().Err(parseErr).Msg("Could not parse SAML assertion with any dashboard provider")
		return c.Redirect().To("/admin/login?error=" + url.QueryEscape("Invalid SAML assertion"))
	}

	// Check if provider allows dashboard login
	provider, _ := h.samlService.GetProvider(providerName)
	if provider == nil || !provider.AllowDashboardLogin {
		log.Warn().Str("provider", providerName).Msg("SAML provider not enabled for dashboard login")
		return c.Redirect().To("/admin/login?error=" + url.QueryEscape("SAML provider not enabled for dashboard login"))
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

	// Capitalize the first letter of each word in the name
	name = capitalizeWords(name)

	providerUserID := assertion.NameID
	if providerUserID == "" {
		providerUserID = email
	}

	if email == "" {
		return c.Redirect().To("/admin/login?error=" + url.QueryEscape("Email not provided in SAML assertion"))
	}

	// RBAC: Validate group membership if configured
	if len(provider.RequiredGroups) > 0 || len(provider.RequiredGroupsAll) > 0 || len(provider.DeniedGroups) > 0 {
		groups := h.samlService.ExtractGroups(providerName, assertion)
		if err := h.samlService.ValidateGroupMembership(provider, groups); err != nil {
			log.Warn().
				Err(err).
				Str("provider", providerName).
				Str("email", email).
				Strs("groups", groups).
				Msg("Dashboard SSO access denied due to group membership")
			return c.Redirect().To("/admin/login?error=" + url.QueryEscape(err.Error()))
		}
	}

	// Find or create dashboard user
	samlProviderName := "saml:" + providerName
	user, _, err := h.authService.FindOrCreateUserBySSO(ctx, email, name, samlProviderName, providerUserID)
	if err != nil {
		log.Error().
			Err(err).
			Str("email", email).
			Str("provider", samlProviderName).
			Str("provider_user_id", providerUserID).
			Msg("Failed to create or find dashboard user via SAML SSO")
		return c.Redirect().To("/admin/login?error=" + url.QueryEscape("Failed to create or find user"))
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
		return c.Redirect().To("/admin/login?error=" + url.QueryEscape(errMsg))
	}

	// Create SAML session for SLO support
	samlSession := &auth.SAMLSession{
		ID:           uuid.New().String(),
		UserID:       user.ID.String(),
		ProviderName: providerName,
		NameID:       assertion.NameID,
		NameIDFormat: assertion.NameIDFormat,
		SessionIndex: assertion.SessionIndex,
		Attributes:   convertSAMLAttributesToMap(assertion.Attributes),
		ExpiresAt:    &assertion.NotOnOrAfter,
		CreatedAt:    time.Now(),
	}

	if err := h.samlService.CreateSAMLSession(ctx, samlSession); err != nil {
		log.Warn().Err(err).Str("user_id", user.ID.String()).Msg("Failed to create SAML session for dashboard user")
	}

	// Redirect with tokens
	redirectURL := relayState
	if redirectURL == "" || redirectURL == "/" {
		redirectURL = "/admin"
	}
	return c.Redirect().To(fmt.Sprintf("/admin/login/callback#access_token=%s&refresh_token=%s&redirect_to=%s",
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

// convertSAMLAttributesToMap converts SAML attributes to a map[string]interface{} for storage
func convertSAMLAttributesToMap(attrs map[string][]string) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range attrs {
		if len(v) == 1 {
			result[k] = v[0]
		} else {
			result[k] = v
		}
	}
	return result
}

// capitalizeWords capitalizes the first letter of each word in a string
func capitalizeWords(s string) string {
	if s == "" {
		return s
	}
	words := strings.Fields(s)
	for i, word := range words {
		if len(word) > 0 {
			// Capitalize first character and lowercase the rest
			words[i] = strings.ToUpper(string(word[0])) + strings.ToLower(word[1:])
		}
	}
	return strings.Join(words, " ")
}

// fiber:context-methods migrated
