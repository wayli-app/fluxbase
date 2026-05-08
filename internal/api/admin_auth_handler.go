package api

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/auth"
	"github.com/nimbleflux/fluxbase/internal/config"
	apperrors "github.com/nimbleflux/fluxbase/internal/errors"
	"github.com/nimbleflux/fluxbase/internal/middleware"
)

// AdminAuthHandler handles admin-specific authentication
type AdminAuthHandler struct {
	authService    *auth.Service
	userRepo       *auth.UserRepository
	dashboardAuth  *auth.DashboardAuthService
	systemSettings *auth.SystemSettingsService
	config         *config.Config
}

// NewAdminAuthHandler creates a new admin auth handler
func NewAdminAuthHandler(
	authService *auth.Service,
	userRepo *auth.UserRepository,
	dashboardAuth *auth.DashboardAuthService,
	systemSettings *auth.SystemSettingsService,
	cfg *config.Config,
) *AdminAuthHandler {
	return &AdminAuthHandler{
		authService:    authService,
		userRepo:       userRepo,
		dashboardAuth:  dashboardAuth,
		systemSettings: systemSettings,
		config:         cfg,
	}
}

func (h *AdminAuthHandler) requireService(c fiber.Ctx) error {
	if h.systemSettings == nil {
		return fiber.NewError(fiber.StatusInternalServerError, "not_initialized")
	}
	return nil
}

func (h *AdminAuthHandler) requireDashboardAuth(c fiber.Ctx) error {
	if h.dashboardAuth == nil {
		return fiber.NewError(fiber.StatusInternalServerError, "not_initialized")
	}
	return nil
}

func (h *AdminAuthHandler) requireAuthService(c fiber.Ctx) error {
	if h.authService == nil {
		return fiber.NewError(fiber.StatusInternalServerError, "not_initialized")
	}
	return nil
}

// SetupStatusResponse represents the setup status
type SetupStatusResponse struct {
	NeedsSetup bool `json:"needs_setup"`
	HasAdmin   bool `json:"has_admin"`
}

// InitialSetupRequest represents the initial setup request
type InitialSetupRequest struct {
	Email      string `json:"email"`
	Password   string `json:"password"`
	Name       string `json:"name"`
	SetupToken string `json:"setup_token"`
}

// InitialSetupResponse represents the initial setup response
type InitialSetupResponse struct {
	User         *auth.DashboardUser `json:"user"`
	AccessToken  string              `json:"access_token"`
	RefreshToken string              `json:"refresh_token"`
	ExpiresIn    int64               `json:"expires_in"`
}

// AdminLoginRequest represents an admin login request
type AdminLoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// AdminLoginResponse represents an admin login response
type AdminLoginResponse struct {
	User         *auth.DashboardUser `json:"user"`
	AccessToken  string              `json:"access_token"`
	RefreshToken string              `json:"refresh_token"`
	ExpiresIn    int64               `json:"expires_in"`
}

// GetSetupStatus checks if initial setup is needed
// GET /api/v1/admin/setup/status
func (h *AdminAuthHandler) GetSetupStatus(c fiber.Ctx) error {
	ctx := context.Background()

	if err := h.requireService(c); err != nil {
		return err
	}

	setupComplete, err := h.systemSettings.IsSetupComplete(ctx)
	if err != nil {
		// If settings table doesn't exist yet (e.g., during bootstrap),
		// treat as needing setup rather than returning a 500.
		log.Debug().Err(err).Msg("Failed to check setup status, assuming setup needed")
		return c.JSON(SetupStatusResponse{
			NeedsSetup: true,
			HasAdmin:   false,
		})
	}

	return c.JSON(SetupStatusResponse{
		NeedsSetup: !setupComplete,
		HasAdmin:   setupComplete,
	})
}

// InitialSetup creates the first admin user
// POST /api/v1/admin/setup
func (h *AdminAuthHandler) InitialSetup(c fiber.Ctx) error {
	log.Debug().Str("path", c.Path()).Str("method", c.Method()).Msg("InitialSetup handler called")

	ctx := context.Background()

	if err := h.requireService(c); err != nil {
		return err
	}

	if err := h.requireDashboardAuth(c); err != nil {
		return err
	}

	setupComplete, err := h.systemSettings.IsSetupComplete(ctx)
	if err != nil {
		return SendOperationFailed(c, "check setup status")
	}

	if setupComplete {
		return SendForbidden(c, "Setup has already been completed", ErrCodeSetupCompleted)
	}

	// Parse request
	var req InitialSetupRequest
	if err := ParseBody(c, &req); err != nil {
		return err
	}

	// Validate setup token using constant-time comparison to prevent timing attacks
	configuredToken := h.config.Security.SetupToken
	if configuredToken == "" {
		return SendForbidden(c, "Admin setup is disabled. Set FLUXBASE_SECURITY_SETUP_TOKEN to enable.", ErrCodeSetupDisabled)
	}

	if req.SetupToken == "" {
		return SendMissingField(c, "setup_token")
	}

	// Use constant-time comparison to prevent timing attacks
	if subtle.ConstantTimeCompare([]byte(req.SetupToken), []byte(configuredToken)) != 1 {
		return SendUnauthorized(c, "Invalid setup token", ErrCodeInvalidSetupToken)
	}

	// Validate password strength
	if err := auth.ValidateDashboardPassword(req.Password); err != nil {
		return SendBadRequest(c, err.Error(), ErrCodeValidationFailed)
	}

	// Create the first dashboard admin user
	user, err := h.dashboardAuth.CreateUser(ctx, req.Email, req.Password, req.Name)
	if err != nil {
		return SendInternalError(c, fmt.Sprintf("Failed to create admin user: %v", err))
	}

	_, err = h.dashboardAuth.GetDB().Exec(ctx, `
		UPDATE platform.users
		SET role = 'instance_admin', email_verified = true
		WHERE id = $1
	`, user.ID)
	if err != nil {
		return SendOperationFailed(c, "set admin role and verify email")
	}

	var defaultTenantID string
	err = h.dashboardAuth.GetDB().QueryRow(ctx, `SELECT id FROM platform.tenants WHERE is_default = true LIMIT 1`).Scan(&defaultTenantID)
	if err == nil {
		_, _ = h.dashboardAuth.GetDB().Exec(ctx, `
			INSERT INTO platform.tenant_admin_assignments (user_id, tenant_id, assigned_by)
			VALUES ($1, $2, $1)
			ON CONFLICT (user_id, tenant_id) DO NOTHING
		`, user.ID, defaultTenantID)
	}

	// Mark setup as complete in system settings
	if err := h.systemSettings.MarkSetupComplete(ctx, user.ID, user.Email); err != nil {
		return SendOperationFailed(c, "mark setup as complete")
	}

	// Log in the user to get access token
	loggedInUser, loginResp, err := h.dashboardAuth.Login(ctx, req.Email, req.Password, nil, c.Get("User-Agent"))
	if err != nil {
		return SendInternalError(c, "User created but failed to generate access token")
	}

	return c.Status(http.StatusCreated).JSON(InitialSetupResponse{
		User:         loggedInUser,
		AccessToken:  loginResp.AccessToken,
		RefreshToken: loginResp.RefreshToken,
		ExpiresIn:    loginResp.ExpiresIn,
	})
}

// AdminLogin authenticates an admin user
// POST /api/v1/admin/login
func (h *AdminAuthHandler) AdminLogin(c fiber.Ctx) error {
	ctx := context.Background()

	var req AdminLoginRequest
	if err := ParseBody(c, &req); err != nil {
		return err
	}

	if err := h.requireDashboardAuth(c); err != nil {
		return err
	}

	user, loginResp, err := h.dashboardAuth.Login(ctx, req.Email, req.Password, nil, c.Get("User-Agent"))
	if err != nil {
		if errors.Is(err, auth.ErrInvalidCredentials) {
			return SendUnauthorized(c, "Invalid email or password", ErrCodeInvalidCredentials)
		}
		if errors.Is(err, auth.ErrAccountLocked) {
			return SendForbidden(c, "Account is locked due to too many failed login attempts", ErrCodeAccountLocked)
		}
		return SendInternalError(c, fmt.Sprintf("Authentication failed: %v", err))
	}

	// Query user's role from database (DashboardUser struct doesn't include role)
	var userRole string
	err = h.dashboardAuth.GetDB().QueryRow(
		ctx,
		"SELECT role FROM platform.users WHERE id = $1",
		user.ID,
	).Scan(&userRole)
	if err != nil {
		return SendOperationFailed(c, "verify user role")
	}

	// Check if user has instance_admin role
	if userRole != "instance_admin" {
		return SendForbidden(c, "Access denied. Admin role required.", ErrCodeAdminRequired)
	}

	return c.JSON(AdminLoginResponse{
		User:         user,
		AccessToken:  loginResp.AccessToken,
		RefreshToken: loginResp.RefreshToken,
		ExpiresIn:    loginResp.ExpiresIn,
	})
}

// AdminRefreshToken refreshes an admin's access token
// POST /api/v1/admin/refresh
func (h *AdminAuthHandler) AdminRefreshToken(c fiber.Ctx) error {
	ctx := context.Background()

	var req struct {
		RefreshToken string `json:"refresh_token"`
	}

	if err := ParseBody(c, &req); err != nil {
		return err
	}

	if err := h.requireDashboardAuth(c); err != nil {
		return err
	}

	refreshResp, err := h.dashboardAuth.RefreshToken(ctx, req.RefreshToken)
	if err != nil {
		return SendUnauthorized(c, "Invalid or expired refresh token", ErrCodeInvalidToken)
	}

	// Validate the new access token to get user ID
	claims, err := h.dashboardAuth.ValidateToken(refreshResp.AccessToken)
	if err != nil {
		return SendUnauthorized(c, "Failed to validate refreshed token", ErrCodeInvalidToken)
	}

	// Parse user ID from claims
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return SendUnauthorized(c, "Invalid user ID in token", ErrCodeInvalidToken)
	}

	// Fetch platform user details (not auth.users)
	user, err := h.dashboardAuth.GetUserByID(ctx, userID)
	if err != nil {
		return SendOperationFailed(c, "fetch user")
	}

	// Verify user still has a valid dashboard role
	validRoles := map[string]bool{
		"instance_admin": true,
		"tenant_admin":   true,
	}
	if !validRoles[user.Role] {
		return SendAdminRequired(c)
	}

	return c.JSON(fiber.Map{
		"access_token":  refreshResp.AccessToken,
		"refresh_token": refreshResp.RefreshToken,
		"expires_in":    refreshResp.ExpiresIn,
		"user":          user,
	})
}

// AdminLogout logs out an admin user
// POST /api/v1/admin/logout
func (h *AdminAuthHandler) AdminLogout(c fiber.Ctx) error {
	ctx := context.Background()

	// Get the access token from the Authorization header
	authHeader := c.Get("Authorization")
	if authHeader == "" {
		return SendMissingAuth(c)
	}

	// Extract token from "Bearer <token>"
	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return SendUnauthorized(c, "Invalid authorization header format", ErrCodeInvalidFormat)
	}

	token := parts[1]

	if err := h.requireAuthService(c); err != nil {
		return err
	}

	if err := h.authService.SignOut(ctx, token); err != nil {
		return SendOperationFailed(c, "logout")
	}

	return apperrors.SendSuccess(c, "Logged out successfully")
}

// GetCurrentAdmin returns the currently authenticated admin user
// GET /api/v1/admin/me
func (h *AdminAuthHandler) GetCurrentAdmin(c fiber.Ctx) error {
	// Get user info from context (set by auth middleware)
	userID := middleware.GetUserID(c)
	if userID == "" {
		return SendUnauthorized(c, "User not authenticated", ErrCodeAuthRequired)
	}

	userEmail, _ := c.Locals("user_email").(string)
	userRole, _ := c.Locals("user_role").(string)

	// Verify admin role
	if userRole != "admin" && userRole != "tenant_service" {
		return SendAdminRequired(c)
	}

	// Return user info from JWT claims (sufficient for UI needs)
	// We could fetch full user from DB but JWT claims have what we need
	return c.JSON(fiber.Map{
		"user": fiber.Map{
			"id":    userID,
			"email": userEmail,
			"role":  userRole,
		},
	})
}
