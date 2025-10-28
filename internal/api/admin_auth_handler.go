package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/wayli-app/fluxbase/internal/auth"
)

// AdminAuthHandler handles admin-specific authentication
type AdminAuthHandler struct {
	authService *auth.Service
	userRepo    *auth.UserRepository
}

// NewAdminAuthHandler creates a new admin auth handler
func NewAdminAuthHandler(authService *auth.Service, userRepo *auth.UserRepository) *AdminAuthHandler {
	return &AdminAuthHandler{
		authService: authService,
		userRepo:    userRepo,
	}
}

// SetupStatusResponse represents the setup status
type SetupStatusResponse struct {
	NeedsSetup bool `json:"needs_setup"`
	HasAdmin   bool `json:"has_admin"`
}

// InitialSetupRequest represents the initial setup request
type InitialSetupRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=12"`
	Name     string `json:"name" validate:"required,min=2"`
}

// InitialSetupResponse represents the initial setup response
type InitialSetupResponse struct {
	User         *auth.User `json:"user"`
	AccessToken  string     `json:"access_token"`
	RefreshToken string     `json:"refresh_token"`
	ExpiresIn    int64      `json:"expires_in"`
}

// AdminLoginRequest represents an admin login request
type AdminLoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// AdminLoginResponse represents an admin login response
type AdminLoginResponse struct {
	User         *auth.User `json:"user"`
	AccessToken  string     `json:"access_token"`
	RefreshToken string     `json:"refresh_token"`
	ExpiresIn    int64      `json:"expires_in"`
}

// GetSetupStatus checks if initial setup is needed
// GET /api/v1/admin/setup/status
func (h *AdminAuthHandler) GetSetupStatus(c *fiber.Ctx) error {
	ctx := context.Background()

	// Count total users
	userCount, err := h.userRepo.Count(ctx)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to check setup status",
		})
	}

	// Count admin users
	adminCount := 0
	if userCount > 0 {
		// Check if any admin users exist
		users, err := h.userRepo.List(ctx, 0, 1000) // Get all users (limit for safety)
		if err == nil {
			for _, user := range users {
				if user.Role == "admin" {
					adminCount++
				}
			}
		}
	}

	return c.JSON(SetupStatusResponse{
		NeedsSetup: adminCount == 0, // Setup needed if no admin users exist
		HasAdmin:   adminCount > 0,
	})
}

// InitialSetup creates the first admin user
// POST /api/v1/admin/setup
func (h *AdminAuthHandler) InitialSetup(c *fiber.Ctx) error {
	ctx := context.Background()

	// Check if setup is still needed (i.e., no admin users exist)
	userCount, err := h.userRepo.Count(ctx)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to check setup status",
		})
	}

	// Check if any admin users already exist
	adminCount := 0
	if userCount > 0 {
		users, err := h.userRepo.List(ctx, 0, 1000)
		if err == nil {
			for _, user := range users {
				if user.Role == "admin" {
					adminCount++
				}
			}
		}
	}

	if adminCount > 0 {
		return c.Status(http.StatusForbidden).JSON(fiber.Map{
			"error": "Setup has already been completed - admin user exists",
		})
	}

	// Parse request
	var req InitialSetupRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate password strength
	if len(req.Password) < 12 {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Password must be at least 12 characters long",
		})
	}

	// Create the first admin user using the auth service
	signupReq := auth.SignUpRequest{
		Email:    req.Email,
		Password: req.Password,
		Metadata: map[string]interface{}{
			"name": req.Name,
		},
	}

	// Sign up the user
	signupResp, err := h.authService.SignUp(ctx, signupReq)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to create admin user: %v", err),
		})
	}

	// Update the user's role to admin
	adminRole := "admin"
	emailVerified := true
	updateReq := auth.UpdateUserRequest{
		Role:          &adminRole,
		EmailVerified: &emailVerified,
	}

	updatedUser, err := h.userRepo.Update(ctx, signupResp.User.ID, updateReq)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to set admin role",
		})
	}

	return c.Status(http.StatusCreated).JSON(InitialSetupResponse{
		User:         updatedUser,
		AccessToken:  signupResp.AccessToken,
		RefreshToken: signupResp.RefreshToken,
		ExpiresIn:    signupResp.ExpiresIn,
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

	// Use the auth service to sign in
	signInReq := auth.SignInRequest{
		Email:    req.Email,
		Password: req.Password,
	}

	signInResp, err := h.authService.SignIn(ctx, signInReq)
	if err != nil {
		if err == auth.ErrInvalidCredentials {
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid email or password",
			})
		}
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Authentication failed",
		})
	}

	// Check if user has admin role
	if signInResp.User.Role != "admin" {
		return c.Status(http.StatusForbidden).JSON(fiber.Map{
			"error": "Access denied. Admin role required.",
		})
	}

	return c.JSON(AdminLoginResponse{
		User:         signInResp.User,
		AccessToken:  signInResp.AccessToken,
		RefreshToken: signInResp.RefreshToken,
		ExpiresIn:    signInResp.ExpiresIn,
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
	// Get user from context (set by auth middleware)
	user := c.Locals("user").(*auth.User)

	// Verify admin role
	if user.Role != "admin" {
		return c.Status(http.StatusForbidden).JSON(fiber.Map{
			"error": "Admin role required",
		})
	}

	return c.JSON(fiber.Map{
		"user": user,
	})
}
