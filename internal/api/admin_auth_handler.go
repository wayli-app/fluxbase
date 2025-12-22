package api

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/fluxbase-eu/fluxbase/internal/auth"
	"github.com/fluxbase-eu/fluxbase/internal/config"
	"github.com/gofiber/fiber/v2"
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

// SetupStatusResponse represents the setup status
type SetupStatusResponse struct {
	NeedsSetup bool `json:"needs_setup"`
	HasAdmin   bool `json:"has_admin"`
}

// InitialSetupRequest represents the initial setup request
type InitialSetupRequest struct {
	Email      string `json:"email" validate:"required,email"`
	Password   string `json:"password" validate:"required,min=12"`
	Name       string `json:"name" validate:"required,min=2"`
	SetupToken string `json:"setup_token" validate:"required"`
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
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
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
func (h *AdminAuthHandler) GetSetupStatus(c *fiber.Ctx) error {
	ctx := context.Background()

	// Check if setup has been completed using system settings
	setupComplete, err := h.systemSettings.IsSetupComplete(ctx)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to check setup status",
		})
	}

	return c.JSON(SetupStatusResponse{
		NeedsSetup: !setupComplete,
		HasAdmin:   setupComplete,
	})
}

// InitialSetup creates the first admin user
// POST /api/v1/admin/setup
func (h *AdminAuthHandler) InitialSetup(c *fiber.Ctx) error {
	ctx := context.Background()

	// Check if setup has already been completed using system settings
	setupComplete, err := h.systemSettings.IsSetupComplete(ctx)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to check setup status",
		})
	}

	if setupComplete {
		return c.Status(http.StatusForbidden).JSON(fiber.Map{
			"error": "Setup has already been completed",
		})
	}

	// Parse request
	var req InitialSetupRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate setup token using constant-time comparison to prevent timing attacks
	configuredToken := h.config.Security.SetupToken
	if configuredToken == "" {
		return c.Status(http.StatusForbidden).JSON(fiber.Map{
			"error": "Admin setup is disabled. Set FLUXBASE_SECURITY_SETUP_TOKEN to enable.",
		})
	}

	if req.SetupToken == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "setup_token is required",
		})
	}

	// Use constant-time comparison to prevent timing attacks
	if subtle.ConstantTimeCompare([]byte(req.SetupToken), []byte(configuredToken)) != 1 {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid setup token",
		})
	}

	// Validate password strength
	if err := auth.ValidateDashboardPassword(req.Password); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Create the first dashboard admin user
	user, err := h.dashboardAuth.CreateUser(ctx, req.Email, req.Password, req.Name)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to create admin user: %v", err),
		})
	}

	// Update user to be dashboard_admin with email verified
	// Direct database update since dashboard service doesn't have these methods yet
	_, err = h.dashboardAuth.GetDB().Exec(ctx, `
		UPDATE dashboard.users
		SET role = 'dashboard_admin', email_verified = true
		WHERE id = $1
	`, user.ID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to set admin role and verify email",
		})
	}

	// Mark setup as complete in system settings
	if err := h.systemSettings.MarkSetupComplete(ctx, user.ID, user.Email); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to mark setup as complete",
		})
	}

	// Log in the user to get access token
	loggedInUser, loginResp, err := h.dashboardAuth.Login(ctx, req.Email, req.Password, nil, c.Get("User-Agent"))
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "User created but failed to generate access token",
		})
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
func (h *AdminAuthHandler) AdminLogin(c *fiber.Ctx) error {
	ctx := context.Background()

	var req AdminLoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Use the dashboard auth service to sign in (dashboard.users, not auth.users)
	user, loginResp, err := h.dashboardAuth.Login(ctx, req.Email, req.Password, nil, c.Get("User-Agent"))
	if err != nil {
		if errors.Is(err, auth.ErrInvalidCredentials) {
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid email or password",
			})
		}
		if errors.Is(err, auth.ErrAccountLocked) {
			return c.Status(http.StatusForbidden).JSON(fiber.Map{
				"error": "Account is locked due to too many failed login attempts",
			})
		}
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Authentication failed: %v", err),
		})
	}

	// Query user's role from database (DashboardUser struct doesn't include role)
	var userRole string
	err = h.dashboardAuth.GetDB().QueryRow(ctx,
		"SELECT role FROM dashboard.users WHERE id = $1",
		user.ID,
	).Scan(&userRole)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to verify user role",
		})
	}

	// Check if user has dashboard_admin role
	if userRole != "dashboard_admin" {
		return c.Status(http.StatusForbidden).JSON(fiber.Map{
			"error": "Access denied. Admin role required.",
		})
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
func (h *AdminAuthHandler) AdminRefreshToken(c *fiber.Ctx) error {
	ctx := context.Background()

	var req struct {
		RefreshToken string `json:"refresh_token" validate:"required"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	refreshReq := auth.RefreshTokenRequest{
		RefreshToken: req.RefreshToken,
	}

	refreshResp, err := h.authService.RefreshToken(ctx, refreshReq)
	if err != nil {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid or expired refresh token",
		})
	}

	// Get user from the new access token to verify admin role
	claims, err := h.authService.ValidateToken(refreshResp.AccessToken)
	if err != nil {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"error": "Failed to validate refreshed token",
		})
	}

	// Fetch user details
	user, err := h.userRepo.GetByID(ctx, claims.UserID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch user",
		})
	}

	// Verify user still has admin role
	if user.Role != "admin" {
		return c.Status(http.StatusForbidden).JSON(fiber.Map{
			"error": "Admin role required",
		})
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
func (h *AdminAuthHandler) AdminLogout(c *fiber.Ctx) error {
	ctx := context.Background()

	// Get the access token from the Authorization header
	authHeader := c.Get("Authorization")
	if authHeader == "" {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"error": "No authorization token provided",
		})
	}

	// Extract token from "Bearer <token>"
	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid authorization header format",
		})
	}

	token := parts[1]

	// Sign out using the auth service
	if err := h.authService.SignOut(ctx, token); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to logout",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Logged out successfully",
	})
}

// GetCurrentAdmin returns the currently authenticated admin user
// GET /api/v1/admin/me
func (h *AdminAuthHandler) GetCurrentAdmin(c *fiber.Ctx) error {
	// Get user info from context (set by auth middleware)
	userID, ok := c.Locals("user_id").(string)
	if !ok {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"error": "User not authenticated",
		})
	}

	userEmail, _ := c.Locals("user_email").(string)
	userRole, _ := c.Locals("user_role").(string)

	// Verify admin role
	if userRole != "admin" {
		return c.Status(http.StatusForbidden).JSON(fiber.Map{
			"error": "Admin role required",
		})
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
