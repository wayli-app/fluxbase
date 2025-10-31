package api

import (
	"errors"
	"fmt"
	"net"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/wayli-app/fluxbase/internal/auth"
)

// DashboardAuthHandler handles dashboard authentication endpoints
type DashboardAuthHandler struct {
	authService *auth.DashboardAuthService
	jwtManager  *auth.JWTManager
}

// NewDashboardAuthHandler creates a new dashboard auth handler
func NewDashboardAuthHandler(authService *auth.DashboardAuthService, jwtManager *auth.JWTManager) *DashboardAuthHandler {
	return &DashboardAuthHandler{
		authService: authService,
		jwtManager:  jwtManager,
	}
}

// RegisterRoutes registers dashboard auth routes
func (h *DashboardAuthHandler) RegisterRoutes(app *fiber.App) {
	dashboard := app.Group("/dashboard/auth")

	// Public routes
	dashboard.Post("/signup", h.Signup)
	dashboard.Post("/login", h.Login)
	dashboard.Post("/2fa/verify", h.VerifyTOTP)

	// Protected routes (require dashboard JWT)
	dashboard.Get("/me", h.requireDashboardAuth, h.GetCurrentUser)
	dashboard.Put("/profile", h.requireDashboardAuth, h.UpdateProfile)
	dashboard.Post("/password/change", h.requireDashboardAuth, h.ChangePassword)
	dashboard.Delete("/account", h.requireDashboardAuth, h.DeleteAccount)

	// 2FA routes
	dashboard.Post("/2fa/setup", h.requireDashboardAuth, h.SetupTOTP)
	dashboard.Post("/2fa/enable", h.requireDashboardAuth, h.EnableTOTP)
	dashboard.Post("/2fa/disable", h.requireDashboardAuth, h.DisableTOTP)
}

// Signup creates a new dashboard user account
func (h *DashboardAuthHandler) Signup(c *fiber.Ctx) error {
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

	user, token, err := h.authService.Login(c.Context(), req.Email, req.Password, ipAddress, userAgent)
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
		return fiber.NewError(fiber.StatusInternalServerError, "Login failed")
	}

	// Check if 2FA is enabled
	if user.TOTPEnabled {
		return c.JSON(fiber.Map{
			"requires_2fa": true,
			"user_id":      user.ID,
		})
	}

	return c.JSON(fiber.Map{
		"access_token": token,
		"user":         user,
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

	// Generate token after successful 2FA
	user, err := h.authService.GetUserByID(c.Context(), userID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to fetch user")
	}

	// Generate JWT token - need to get secret from config
	// For now, return success
	return c.JSON(fiber.Map{
		"message": "2FA verified successfully",
		"user":    user,
	})
}

// GetCurrentUser returns the currently authenticated dashboard user
func (h *DashboardAuthHandler) GetCurrentUser(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uuid.UUID)

	user, err := h.authService.GetUserByID(c.Context(), userID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "User not found")
	}

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

	if len(req.NewPassword) < 8 {
		return fiber.NewError(fiber.StatusBadRequest, "New password must be at least 8 characters")
	}

	ipAddress := getIPAddress(c)
	userAgent := string(c.Request().Header.UserAgent())

	err := h.authService.ChangePassword(c.Context(), userID, req.CurrentPassword, req.NewPassword, ipAddress, userAgent)
	if err != nil {
		if err.Error() == "current password is incorrect" {
			return fiber.NewError(fiber.StatusUnauthorized, "Current password is incorrect")
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

	secret, qrURL, err := h.authService.SetupTOTP(c.Context(), userID, user.Email)
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

// requireDashboardAuth is a middleware that requires dashboard authentication
func (h *DashboardAuthHandler) requireDashboardAuth(c *fiber.Ctx) error {
	authHeader := c.Get("Authorization")
	if authHeader == "" {
		return fiber.NewError(fiber.StatusUnauthorized, "Missing authorization header")
	}

	// Extract token from "Bearer <token>"
	token := ""
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

	// Set user ID in locals
	c.Locals("user_id", userID)
	c.Locals("role", claims.Role)

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
