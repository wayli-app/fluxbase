package api

import (
	"github.com/gofiber/fiber/v2"
	"github.com/wayli-app/fluxbase/internal/auth"
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

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Magic link sent to your email",
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

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "If an account with that email exists, a password reset link has been sent",
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

	// Reset password
	if err := h.authService.ResetPassword(c.Context(), req.Token, req.NewPassword); err != nil {
		log.Error().Err(err).Msg("Failed to reset password")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Password has been successfully reset",
	})
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
func (h *AuthHandler) RegisterRoutes(app *fiber.App, rateLimiters map[string]fiber.Handler) {
	auth := app.Group("/auth")

	// Public routes with rate limiting
	auth.Post("/signup", rateLimiters["signup"], h.SignUp)
	auth.Post("/signin", rateLimiters["login"], h.SignIn)
	auth.Post("/signin/anonymous", h.SignInAnonymous) // No rate limit - creates unique user each time
	auth.Post("/refresh", rateLimiters["refresh"], h.RefreshToken)
	auth.Post("/magiclink", rateLimiters["magiclink"], h.SendMagicLink)
	auth.Post("/magiclink/verify", h.VerifyMagicLink) // No rate limit on verification
	auth.Post("/password/reset", rateLimiters["password_reset"], h.RequestPasswordReset)
	auth.Post("/password/reset/confirm", h.ResetPassword) // No rate limit on actual reset (token is single-use)
	auth.Post("/password/reset/verify", h.VerifyPasswordResetToken) // No rate limit on verification

	// Protected routes (authentication required) - lighter rate limits
	auth.Post("/signout", h.SignOut)
	auth.Get("/user", h.GetUser)
	auth.Patch("/user", h.UpdateUser)

	// Admin impersonation routes (admin only)
	auth.Post("/impersonate", h.StartImpersonation)
	auth.Delete("/impersonate", h.StopImpersonation)
	auth.Get("/impersonate", h.GetActiveImpersonation)
	auth.Get("/impersonate/sessions", h.ListImpersonationSessions)
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
