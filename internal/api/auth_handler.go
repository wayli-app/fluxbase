package api

import (
	"fmt"

	"github.com/fluxbase-eu/fluxbase/internal/auth"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
)

// AuthHandler handles authentication HTTP requests
type AuthHandler struct {
	authService *auth.Service
}

// NewAuthHandler creates a new authentication handler
func NewAuthHandler(authService *auth.Service) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

// SignUp handles user registration
// POST /auth/signup
func (h *AuthHandler) SignUp(c *fiber.Ctx) error {
	// Check if signup is enabled
	if !h.authService.IsSignupEnabled() {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "User registration is currently disabled",
			"code":  "SIGNUP_DISABLED",
		})
	}

	var req auth.SignUpRequest
	if err := c.BodyParser(&req); err != nil {
		log.Error().Err(err).Msg("Failed to parse signup request")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate required fields
	if req.Email == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Email is required",
		})
	}
	if req.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Password is required",
		})
	}

	// Create user
	resp, err := h.authService.SignUp(c.Context(), req)
	if err != nil {
		log.Error().Err(err).Str("email", req.Email).Msg("Failed to sign up user")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(resp)
}

// SignIn handles user login
// POST /auth/signin
func (h *AuthHandler) SignIn(c *fiber.Ctx) error {
	var req auth.SignInRequest
	if err := c.BodyParser(&req); err != nil {
		log.Error().Err(err).Msg("Failed to parse signin request")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate required fields
	if req.Email == "" || req.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Email and password are required",
		})
	}

	// Authenticate user
	resp, err := h.authService.SignIn(c.Context(), req)
	if err != nil {
		log.Error().Err(err).Str("email", req.Email).Msg("Failed to sign in user")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid email or password",
		})
	}

	// Check if user has 2FA enabled
	twoFAEnabled, err := h.authService.IsTOTPEnabled(c.Context(), resp.User.ID)
	if err != nil {
		log.Error().Err(err).Str("user_id", resp.User.ID).Msg("Failed to check 2FA status")
		// Continue with login - don't block if 2FA check fails
		return c.Status(fiber.StatusOK).JSON(resp)
	}

	// If 2FA is enabled, return special response requiring 2FA verification
	if twoFAEnabled {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"requires_2fa": true,
			"user_id":      resp.User.ID,
			"message":      "2FA verification required. Please provide your 2FA code.",
		})
	}

	return c.Status(fiber.StatusOK).JSON(resp)
}

// SignOut handles user logout
// POST /auth/signout
func (h *AuthHandler) SignOut(c *fiber.Ctx) error {
	// Get token from Authorization header
	token := c.Get("Authorization")
	if token == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Authorization header is required",
		})
	}

	// Remove "Bearer " prefix if present
	if len(token) > 7 && token[:7] == "Bearer " {
		token = token[7:]
	}

	// Sign out user
	if err := h.authService.SignOut(c.Context(), token); err != nil {
		log.Error().Err(err).Msg("Failed to sign out user")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to sign out",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Successfully signed out",
	})
}

// RefreshToken handles token refresh
// POST /auth/refresh
func (h *AuthHandler) RefreshToken(c *fiber.Ctx) error {
	var req auth.RefreshTokenRequest
	if err := c.BodyParser(&req); err != nil {
		log.Error().Err(err).Msg("Failed to parse refresh token request")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate required fields
	if req.RefreshToken == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Refresh token is required",
		})
	}

	// Refresh token
	resp, err := h.authService.RefreshToken(c.Context(), req)
	if err != nil {
		log.Error().Err(err).Msg("Failed to refresh token")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid or expired refresh token",
		})
	}

	return c.Status(fiber.StatusOK).JSON(resp)
}

// GetUser handles getting current user profile
// GET /auth/user
func (h *AuthHandler) GetUser(c *fiber.Ctx) error {
	// Get token from Authorization header
	token := c.Get("Authorization")
	if token == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authorization header is required",
		})
	}

	// Remove "Bearer " prefix if present
	if len(token) > 7 && token[:7] == "Bearer " {
		token = token[7:]
	}

	// Get user
	user, err := h.authService.GetUser(c.Context(), token)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get user")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid or expired token",
		})
	}

	return c.Status(fiber.StatusOK).JSON(user)
}

// UpdateUser handles updating user profile
// PATCH /auth/user
func (h *AuthHandler) UpdateUser(c *fiber.Ctx) error {
	// Get user ID from context (set by auth middleware)
	userID := c.Locals("user_id")
	if userID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	var req auth.UpdateUserRequest
	if err := c.BodyParser(&req); err != nil {
		log.Error().Err(err).Msg("Failed to parse update user request")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Update user
	user, err := h.authService.UpdateUser(c.Context(), userID.(string), req)
	if err != nil {
		log.Error().Err(err).Str("user_id", userID.(string)).Msg("Failed to update user")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(user)
}

// SendMagicLink handles sending magic link
// POST /auth/magiclink
func (h *AuthHandler) SendMagicLink(c *fiber.Ctx) error {
	var req struct {
		Email string `json:"email"`
	}
	if err := c.BodyParser(&req); err != nil {
		log.Error().Err(err).Msg("Failed to parse magic link request")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate email
	if req.Email == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Email is required",
		})
	}

	// Send magic link
	if err := h.authService.SendMagicLink(c.Context(), req.Email); err != nil {
		log.Error().Err(err).Str("email", req.Email).Msg("Failed to send magic link")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Return Supabase-compatible OTP response
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"user":    nil,
		"session": nil,
	})
}

// VerifyMagicLink handles magic link verification
// POST /auth/magiclink/verify
func (h *AuthHandler) VerifyMagicLink(c *fiber.Ctx) error {
	var req struct {
		Token string `json:"token"`
	}
	if err := c.BodyParser(&req); err != nil {
		log.Error().Err(err).Msg("Failed to parse verify magic link request")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate token
	if req.Token == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Token is required",
		})
	}

	// Verify magic link
	resp, err := h.authService.VerifyMagicLink(c.Context(), req.Token)
	if err != nil {
		log.Error().Err(err).Msg("Failed to verify magic link")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(resp)
}

// RequestPasswordReset handles password reset requests
// POST /auth/password/reset
func (h *AuthHandler) RequestPasswordReset(c *fiber.Ctx) error {
	var req struct {
		Email string `json:"email"`
	}

	if err := c.BodyParser(&req); err != nil {
		log.Error().Err(err).Msg("Failed to parse password reset request")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate email
	if req.Email == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Email is required",
		})
	}

	// Request password reset (this won't reveal if user exists)
	if err := h.authService.RequestPasswordReset(c.Context(), req.Email); err != nil {
		log.Error().Err(err).Str("email", req.Email).Msg("Failed to request password reset")
		// Don't reveal if user exists - always return success
	}

	// Return Supabase-compatible OTP response
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"user":    nil,
		"session": nil,
	})
}

// ResetPassword handles password reset with token
// POST /auth/password/reset/confirm
func (h *AuthHandler) ResetPassword(c *fiber.Ctx) error {
	var req struct {
		Token       string `json:"token"`
		NewPassword string `json:"new_password"`
	}

	if err := c.BodyParser(&req); err != nil {
		log.Error().Err(err).Msg("Failed to parse reset password request")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate required fields
	if req.Token == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Token is required",
		})
	}
	if req.NewPassword == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "New password is required",
		})
	}

	// Reset password and get user ID
	userID, err := h.authService.ResetPassword(c.Context(), req.Token, req.NewPassword)
	if err != nil {
		log.Error().Err(err).Msg("Failed to reset password")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Generate new tokens for the user (Supabase-compatible)
	resp, err := h.authService.GenerateTokensForUser(c.Context(), userID)
	if err != nil {
		log.Error().Err(err).Msg("Failed to generate tokens after password reset")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to generate authentication tokens",
		})
	}

	return c.Status(fiber.StatusOK).JSON(resp)
}

// VerifyPasswordResetToken handles password reset token verification
// POST /auth/password/reset/verify
func (h *AuthHandler) VerifyPasswordResetToken(c *fiber.Ctx) error {
	var req struct {
		Token string `json:"token"`
	}

	if err := c.BodyParser(&req); err != nil {
		log.Error().Err(err).Msg("Failed to parse verify token request")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate token
	if req.Token == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Token is required",
		})
	}

	// Verify token
	if err := h.authService.VerifyPasswordResetToken(c.Context(), req.Token); err != nil {
		log.Error().Err(err).Msg("Failed to verify password reset token")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Token is valid",
	})
}

// RegisterRoutes registers all authentication routes with rate limiting
func (h *AuthHandler) RegisterRoutes(router fiber.Router, rateLimiters map[string]fiber.Handler) {
	// Register routes directly on the provided router (which should already be /api/v1/auth or similar)

	// Public routes with rate limiting
	router.Post("/signup", rateLimiters["signup"], h.SignUp)
	router.Post("/signin", rateLimiters["login"], h.SignIn)
	router.Post("/signin/anonymous", h.SignInAnonymous) // No rate limit - creates unique user each time
	router.Post("/refresh", rateLimiters["refresh"], h.RefreshToken)
	router.Post("/magiclink", rateLimiters["magiclink"], h.SendMagicLink)
	router.Post("/magiclink/verify", h.VerifyMagicLink) // No rate limit on verification
	router.Post("/password/reset", rateLimiters["password_reset"], h.RequestPasswordReset)
	router.Post("/password/reset/confirm", h.ResetPassword)           // No rate limit on actual reset (token is single-use)
	router.Post("/password/reset/verify", h.VerifyPasswordResetToken) // No rate limit on verification

	// 2FA verification (public - used during login)
	router.Post("/2fa/verify", h.VerifyTOTP)

	// OTP routes (public)
	router.Post("/otp/signin", rateLimiters["otp"], h.SendOTP)
	router.Post("/otp/verify", h.VerifyOTP)
	router.Post("/otp/resend", rateLimiters["otp"], h.ResendOTP)

	// ID token signin (public - for mobile OAuth)
	router.Post("/signin/idtoken", h.SignInWithIDToken)

	// Protected routes (authentication required) - lighter rate limits
	// Apply auth middleware to all routes below
	authMiddleware := AuthMiddleware(h.authService)
	router.Post("/signout", authMiddleware, h.SignOut)
	router.Get("/user", authMiddleware, h.GetUser)
	router.Patch("/user", authMiddleware, h.UpdateUser)

	// Admin impersonation routes (admin only)
	router.Post("/impersonate", authMiddleware, h.StartImpersonation)
	router.Post("/impersonate/anon", authMiddleware, h.StartAnonImpersonation)
	router.Post("/impersonate/service", authMiddleware, h.StartServiceImpersonation)
	router.Delete("/impersonate", authMiddleware, h.StopImpersonation)
	router.Get("/impersonate", authMiddleware, h.GetActiveImpersonation)
	router.Get("/impersonate/sessions", authMiddleware, h.ListImpersonationSessions)

	// 2FA routes (protected - authentication required)
	router.Post("/2fa/setup", authMiddleware, h.SetupTOTP)
	router.Post("/2fa/enable", authMiddleware, h.EnableTOTP)
	router.Post("/2fa/disable", authMiddleware, h.DisableTOTP)
	router.Get("/2fa/status", authMiddleware, h.GetTOTPStatus)

	// Identity linking routes (protected - authentication required)
	router.Get("/user/identities", authMiddleware, h.GetUserIdentities)
	router.Post("/user/identities", authMiddleware, h.LinkIdentity)
	router.Delete("/user/identities/:id", authMiddleware, h.UnlinkIdentity)

	// Reauthentication route (protected - authentication required)
	router.Post("/reauthenticate", authMiddleware, h.Reauthenticate)
}

// SignInAnonymous creates JWT tokens for an anonymous user (no database record)
func (h *AuthHandler) SignInAnonymous(c *fiber.Ctx) error {
	resp, err := h.authService.SignInAnonymous(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(resp)
}

// StartImpersonation starts an admin impersonation session
func (h *AuthHandler) StartImpersonation(c *fiber.Ctx) error {
	// Get admin user ID from context (must be authenticated)
	adminUserID := c.Locals("user_id")
	if adminUserID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authentication required",
		})
	}

	var req auth.StartImpersonationRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Set IP and user agent from request
	req.IPAddress = c.IP()
	req.UserAgent = c.Get("User-Agent")

	resp, err := h.authService.StartImpersonation(c.Context(), adminUserID.(string), req)
	if err != nil {
		statusCode := fiber.StatusInternalServerError
		if err == auth.ErrNotAdmin {
			statusCode = fiber.StatusForbidden
		} else if err == auth.ErrSelfImpersonation {
			statusCode = fiber.StatusBadRequest
		}
		return c.Status(statusCode).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(resp)
}

// StopImpersonation stops the active impersonation session
func (h *AuthHandler) StopImpersonation(c *fiber.Ctx) error {
	// Get admin user ID from context
	adminUserID := c.Locals("user_id")
	if adminUserID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authentication required",
		})
	}

	err := h.authService.StopImpersonation(c.Context(), adminUserID.(string))
	if err != nil {
		if err == auth.ErrNoActiveImpersonation {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Impersonation session ended",
	})
}

// GetActiveImpersonation gets the active impersonation session
func (h *AuthHandler) GetActiveImpersonation(c *fiber.Ctx) error {
	// Get admin user ID from context
	adminUserID := c.Locals("user_id")
	if adminUserID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authentication required",
		})
	}

	session, err := h.authService.GetActiveImpersonation(c.Context(), adminUserID.(string))
	if err != nil {
		if err == auth.ErrNoActiveImpersonation {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(session)
}

// ListImpersonationSessions lists impersonation sessions for audit
func (h *AuthHandler) ListImpersonationSessions(c *fiber.Ctx) error {
	// Get admin user ID from context
	adminUserID := c.Locals("user_id")
	if adminUserID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authentication required",
		})
	}

	limit := c.QueryInt("limit", 50)
	offset := c.QueryInt("offset", 0)

	sessions, err := h.authService.ListImpersonationSessions(c.Context(), adminUserID.(string), limit, offset)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(sessions)
}

// StartAnonImpersonation starts impersonation as anonymous user
func (h *AuthHandler) StartAnonImpersonation(c *fiber.Ctx) error {
	// Get admin user ID from context (must be authenticated)
	adminUserID := c.Locals("user_id")
	if adminUserID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authentication required",
		})
	}

	var req struct {
		Reason string `json:"reason"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.Reason == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Reason is required",
		})
	}

	// Set IP and user agent from request
	ipAddress := c.IP()
	userAgent := c.Get("User-Agent")

	resp, err := h.authService.StartAnonImpersonation(c.Context(), adminUserID.(string), req.Reason, ipAddress, userAgent)
	if err != nil {
		statusCode := fiber.StatusInternalServerError
		if err == auth.ErrNotAdmin {
			statusCode = fiber.StatusForbidden
		}
		return c.Status(statusCode).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(resp)
}

// StartServiceImpersonation starts impersonation with service role
func (h *AuthHandler) StartServiceImpersonation(c *fiber.Ctx) error {
	// Get admin user ID from context (must be authenticated)
	adminUserID := c.Locals("user_id")
	if adminUserID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authentication required",
		})
	}

	var req struct {
		Reason string `json:"reason"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.Reason == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Reason is required",
		})
	}

	// Set IP and user agent from request
	ipAddress := c.IP()
	userAgent := c.Get("User-Agent")

	resp, err := h.authService.StartServiceImpersonation(c.Context(), adminUserID.(string), req.Reason, ipAddress, userAgent)
	if err != nil {
		statusCode := fiber.StatusInternalServerError
		if err == auth.ErrNotAdmin {
			statusCode = fiber.StatusForbidden
		}
		return c.Status(statusCode).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(resp)
}

// SetupTOTP initiates 2FA setup by generating a TOTP secret
// POST /auth/2fa/setup
func (h *AuthHandler) SetupTOTP(c *fiber.Ctx) error {
	// Get user ID from JWT token
	userID := c.Locals("user_id")
	if userID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	// Parse optional issuer from request body
	var req struct {
		Issuer string `json:"issuer"` // Optional: custom issuer name for the QR code
	}
	// Ignore parse errors - issuer is optional and will default to config value
	_ = c.BodyParser(&req)

	response, err := h.authService.SetupTOTP(c.Context(), userID.(string), req.Issuer)
	if err != nil {
		log.Error().Err(err).Str("user_id", userID.(string)).Msg("Failed to setup TOTP")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to setup 2FA",
		})
	}

	return c.Status(fiber.StatusOK).JSON(response)
}

// EnableTOTP enables 2FA after verifying the TOTP code
// POST /auth/2fa/enable
func (h *AuthHandler) EnableTOTP(c *fiber.Ctx) error {
	// Get user ID from JWT token
	userID := c.Locals("user_id")
	if userID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	var req struct {
		Code string `json:"code"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.Code == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Code is required",
		})
	}

	backupCodes, err := h.authService.EnableTOTP(c.Context(), userID.(string), req.Code)
	if err != nil {
		log.Error().Err(err).Str("user_id", userID.(string)).Msg("Failed to enable TOTP")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success":      true,
		"backup_codes": backupCodes,
		"message":      "2FA enabled successfully. Please save your backup codes in a secure location.",
	})
}

// VerifyTOTP verifies a TOTP code during login and issues JWT tokens
// POST /auth/2fa/verify
func (h *AuthHandler) VerifyTOTP(c *fiber.Ctx) error {
	var req struct {
		UserID string `json:"user_id"`
		Code   string `json:"code"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.UserID == "" || req.Code == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "User ID and code are required",
		})
	}

	// Verify the 2FA code
	err := h.authService.VerifyTOTP(c.Context(), req.UserID, req.Code)
	if err != nil {
		log.Warn().Err(err).Str("user_id", req.UserID).Msg("Failed to verify TOTP")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Generate a complete sign-in response with tokens
	resp, err := h.authService.GenerateTokensForUser(c.Context(), req.UserID)
	if err != nil {
		log.Error().Err(err).Str("user_id", req.UserID).Msg("Failed to generate tokens after 2FA verification")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to complete authentication",
		})
	}

	return c.Status(fiber.StatusOK).JSON(resp)
}

// DisableTOTP disables 2FA for a user
// POST /auth/2fa/disable
func (h *AuthHandler) DisableTOTP(c *fiber.Ctx) error {
	// Get user ID from JWT token
	userID := c.Locals("user_id")
	if userID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	var req struct {
		Password string `json:"password"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Password is required to disable 2FA",
		})
	}

	err := h.authService.DisableTOTP(c.Context(), userID.(string), req.Password)
	if err != nil {
		log.Error().Err(err).Str("user_id", userID.(string)).Msg("Failed to disable TOTP")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"message": "2FA disabled successfully",
	})
}

// GetTOTPStatus checks if 2FA is enabled for a user
// GET /auth/2fa/status
func (h *AuthHandler) GetTOTPStatus(c *fiber.Ctx) error {
	// Get user ID from JWT token
	userID := c.Locals("user_id")
	if userID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	enabled, err := h.authService.IsTOTPEnabled(c.Context(), userID.(string))
	if err != nil {
		log.Error().Err(err).Str("user_id", userID.(string)).Msg("Failed to check TOTP status")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to check 2FA status",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"totp_enabled": enabled,
	})
}

// SendOTP sends an OTP code via email or SMS
// POST /auth/otp/signin
func (h *AuthHandler) SendOTP(c *fiber.Ctx) error {
	var req struct {
		Email   *string                 `json:"email,omitempty"`
		Phone   *string                 `json:"phone,omitempty"`
		Options *map[string]interface{} `json:"options,omitempty"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate that either email or phone is provided
	if req.Email == nil && req.Phone == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Either email or phone must be provided",
		})
	}

	// Send OTP
	var err error
	purpose := "signin" // Default purpose
	if req.Options != nil {
		if p, ok := (*req.Options)["purpose"].(string); ok {
			purpose = p
		}
	}

	if req.Email != nil {
		err = h.authService.SendOTP(c.Context(), *req.Email, purpose)
	} else if req.Phone != nil {
		// SMS OTP not yet fully implemented
		err = fmt.Errorf("SMS OTP not yet implemented")
	}

	if err != nil {
		log.Error().Err(err).Msg("Failed to send OTP")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to send OTP code",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"user":    nil,
		"session": nil,
	})
}

// VerifyOTP verifies an OTP code and creates a session
// POST /auth/otp/verify
func (h *AuthHandler) VerifyOTP(c *fiber.Ctx) error {
	var req struct {
		Email *string `json:"email,omitempty"`
		Phone *string `json:"phone,omitempty"`
		Token string  `json:"token"`
		Type  string  `json:"type"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.Token == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "OTP token is required",
		})
	}

	// Verify OTP
	var otpCode *auth.OTPCode
	var err error

	if req.Email != nil {
		otpCode, err = h.authService.VerifyOTP(c.Context(), *req.Email, req.Token)
	} else if req.Phone != nil {
		// Phone OTP not yet fully implemented
		return c.Status(fiber.StatusNotImplemented).JSON(fiber.Map{
			"error": "Phone-based OTP authentication not yet implemented",
		})
	} else {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Either email or phone must be provided",
		})
	}

	if err != nil {
		log.Warn().Err(err).Msg("Failed to verify OTP")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid or expired OTP code",
		})
	}

	// Get or create user
	var user *auth.User
	if req.Email != nil && otpCode.Email != nil {
		user, err = h.authService.GetUserByEmail(c.Context(), *otpCode.Email)
		if err != nil {
			// Create user if doesn't exist (for signup flow)
			user, err = h.authService.CreateUser(c.Context(), *otpCode.Email, "")
			if err != nil {
				log.Error().Err(err).Msg("Failed to create user")
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Failed to create user",
				})
			}
		}
	}

	// Generate tokens
	resp, err := h.authService.GenerateTokensForUser(c.Context(), user.ID)
	if err != nil {
		log.Error().Err(err).Msg("Failed to generate tokens")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to complete authentication",
		})
	}

	return c.Status(fiber.StatusOK).JSON(resp)
}

// ResendOTP resends an OTP code
// POST /auth/otp/resend
func (h *AuthHandler) ResendOTP(c *fiber.Ctx) error {
	var req struct {
		Type    string                  `json:"type"`
		Email   *string                 `json:"email,omitempty"`
		Phone   *string                 `json:"phone,omitempty"`
		Options *map[string]interface{} `json:"options,omitempty"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.Email == nil && req.Phone == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Either email or phone must be provided",
		})
	}

	purpose := "signin" // Default purpose
	if req.Options != nil {
		if p, ok := (*req.Options)["purpose"].(string); ok {
			purpose = p
		}
	}

	// Resend OTP
	var err error
	if req.Email != nil {
		err = h.authService.ResendOTP(c.Context(), *req.Email, purpose)
	} else if req.Phone != nil {
		// SMS OTP not yet fully implemented
		err = fmt.Errorf("SMS OTP not yet implemented")
	}

	if err != nil {
		log.Error().Err(err).Msg("Failed to resend OTP")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to resend OTP code",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"user":    nil,
		"session": nil,
	})
}

// GetUserIdentities gets all OAuth identities linked to a user
// GET /auth/user/identities
func (h *AuthHandler) GetUserIdentities(c *fiber.Ctx) error {
	userID := c.Locals("user_id")
	if userID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	identities, err := h.authService.GetUserIdentities(c.Context(), userID.(string))
	if err != nil {
		log.Error().Err(err).Str("user_id", userID.(string)).Msg("Failed to get user identities")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve identities",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"identities": identities,
	})
}

// LinkIdentity initiates OAuth flow to link a provider
// POST /auth/user/identities
func (h *AuthHandler) LinkIdentity(c *fiber.Ctx) error {
	userID := c.Locals("user_id")
	if userID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	var req struct {
		Provider string `json:"provider"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.Provider == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Provider is required",
		})
	}

	authURL, state, err := h.authService.LinkIdentity(c.Context(), userID.(string), req.Provider)
	if err != nil {
		log.Error().Err(err).Str("provider", req.Provider).Msg("Failed to initiate identity linking")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"url":      authURL,
		"provider": req.Provider,
		"state":    state,
	})
}

// UnlinkIdentity removes an OAuth identity from a user
// DELETE /auth/user/identities/:id
func (h *AuthHandler) UnlinkIdentity(c *fiber.Ctx) error {
	userID := c.Locals("user_id")
	if userID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	identityID := c.Params("id")
	if identityID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Identity ID is required",
		})
	}

	err := h.authService.UnlinkIdentity(c.Context(), userID.(string), identityID)
	if err != nil {
		log.Error().Err(err).Str("identity_id", identityID).Msg("Failed to unlink identity")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
	})
}

// Reauthenticate generates a security nonce
// POST /auth/reauthenticate
func (h *AuthHandler) Reauthenticate(c *fiber.Ctx) error {
	userID := c.Locals("user_id")
	if userID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	nonce, err := h.authService.Reauthenticate(c.Context(), userID.(string))
	if err != nil {
		log.Error().Err(err).Str("user_id", userID.(string)).Msg("Failed to reauthenticate")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to generate security nonce",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"nonce": nonce,
	})
}

// SignInWithIDToken handles OAuth ID token authentication (Google, Apple)
// POST /auth/signin/idtoken
func (h *AuthHandler) SignInWithIDToken(c *fiber.Ctx) error {
	var req struct {
		Provider string  `json:"provider"`
		Token    string  `json:"token"`
		Nonce    *string `json:"nonce,omitempty"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.Provider == "" || req.Token == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Provider and token are required",
		})
	}

	nonce := ""
	if req.Nonce != nil {
		nonce = *req.Nonce
	}

	resp, err := h.authService.SignInWithIDToken(c.Context(), req.Provider, req.Token, nonce)
	if err != nil {
		log.Error().Err(err).Str("provider", req.Provider).Msg("Failed to sign in with ID token")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(resp)
}
