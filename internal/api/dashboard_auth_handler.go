package api

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"golang.org/x/oauth2"

	"github.com/nimbleflux/fluxbase/internal/auth"
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
		return fiber.NewError(fiber.StatusServiceUnavailable, "Auth service not initialized")
	}
	return nil
}

func (h *DashboardAuthHandler) requireJWTManager(c fiber.Ctx) error {
	if h.jwtManager == nil {
		return fiber.NewError(fiber.StatusServiceUnavailable, "JWT manager not initialized")
	}
	return nil
}

func (h *DashboardAuthHandler) requireDB(c fiber.Ctx) error {
	if h.db == nil {
		return fiber.NewError(fiber.StatusServiceUnavailable, "Database not initialized")
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
