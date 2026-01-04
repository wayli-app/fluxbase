package api

import (
	"errors"
	"fmt"

	"github.com/fluxbase-eu/fluxbase/internal/auth"
	"github.com/fluxbase-eu/fluxbase/internal/middleware"
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

// Cookie names for authentication tokens
const (
	AccessTokenCookieName  = "fluxbase_access_token"
	RefreshTokenCookieName = "fluxbase_refresh_token"
)

// AuthHandler handles authentication HTTP requests
type AuthHandler struct {
	db             *pgxpool.Pool
	authService    *auth.Service
	captchaService *auth.CaptchaService
	samlService    *auth.SAMLService
	baseURL        string
	secureCookie   bool // Whether to set Secure flag on cookies (true in production)
}

// NewAuthHandler creates a new authentication handler
func NewAuthHandler(db *pgxpool.Pool, authService *auth.Service, captchaService *auth.CaptchaService, baseURL string) *AuthHandler {
	return &AuthHandler{
		db:             db,
		authService:    authService,
		captchaService: captchaService,
		baseURL:        baseURL,
		secureCookie:   false, // Will be set based on environment
	}
}

// SetSAMLService sets the SAML service for SLO integration
func (h *AuthHandler) SetSAMLService(samlService *auth.SAMLService) {
	h.samlService = samlService
}

// SetSecureCookie sets whether cookies should have the Secure flag
func (h *AuthHandler) SetSecureCookie(secure bool) {
	h.secureCookie = secure
}

// AuthConfigResponse represents the public authentication configuration
type AuthConfigResponse struct {
	SignupEnabled            bool                        `json:"signup_enabled"`
	RequireEmailVerification bool                        `json:"require_email_verification"`
	MagicLinkEnabled         bool                        `json:"magic_link_enabled"`
	MFAAvailable             bool                        `json:"mfa_available"`
	PasswordMinLength        int                         `json:"password_min_length"`
	PasswordRequireUppercase bool                        `json:"password_require_uppercase"`
	PasswordRequireLowercase bool                        `json:"password_require_lowercase"`
	PasswordRequireNumber    bool                        `json:"password_require_number"`
	PasswordRequireSpecial   bool                        `json:"password_require_special"`
	OAuthProviders           []OAuthProviderPublic       `json:"oauth_providers"`
	SAMLProviders            []SAMLProviderPublic        `json:"saml_providers"`
	Captcha                  *auth.CaptchaConfigResponse `json:"captcha"`
}

// OAuthProviderPublic represents public OAuth provider information
type OAuthProviderPublic struct {
	Provider     string `json:"provider"`
	DisplayName  string `json:"display_name"`
	AuthorizeURL string `json:"authorize_url"`
}

// SAMLProviderPublic represents public SAML provider information
type SAMLProviderPublic struct {
	Provider    string `json:"provider"`
	DisplayName string `json:"display_name"`
}

// setAuthCookies sets httpOnly cookies for access and refresh tokens
func (h *AuthHandler) setAuthCookies(c *fiber.Ctx, accessToken, refreshToken string, expiresIn int64) {
	// Access token cookie - shorter expiry
	c.Cookie(&fiber.Cookie{
		Name:     AccessTokenCookieName,
		Value:    accessToken,
		Path:     "/",
		MaxAge:   int(expiresIn), // seconds
		Secure:   h.secureCookie,
		HTTPOnly: true,
		SameSite: "Strict",
	})

	// Refresh token cookie - longer expiry (7 days default)
	c.Cookie(&fiber.Cookie{
		Name:     RefreshTokenCookieName,
		Value:    refreshToken,
		Path:     "/api/v1/auth",   // Only sent to auth endpoints
		MaxAge:   7 * 24 * 60 * 60, // 7 days
		Secure:   h.secureCookie,
		HTTPOnly: true,
		SameSite: "Strict",
	})
}

// clearAuthCookies removes authentication cookies
func (h *AuthHandler) clearAuthCookies(c *fiber.Ctx) {
	c.Cookie(&fiber.Cookie{
		Name:     AccessTokenCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1, // Expire immediately
		Secure:   h.secureCookie,
		HTTPOnly: true,
		SameSite: "Strict",
	})

	c.Cookie(&fiber.Cookie{
		Name:     RefreshTokenCookieName,
		Value:    "",
		Path:     "/api/v1/auth",
		MaxAge:   -1,
		Secure:   h.secureCookie,
		HTTPOnly: true,
		SameSite: "Strict",
	})
}

// getAccessToken gets the access token from cookie or Authorization header
func (h *AuthHandler) getAccessToken(c *fiber.Ctx) string {
	// First try cookie
	if token := c.Cookies(AccessTokenCookieName); token != "" {
		return token
	}

	// Fall back to Authorization header for API clients
	token := c.Get("Authorization")
	if len(token) > 7 && token[:7] == "Bearer " {
		return token[7:]
	}
	return token
}

// getRefreshToken gets the refresh token from cookie or request body
func (h *AuthHandler) getRefreshToken(c *fiber.Ctx) string {
	// First try cookie
	if token := c.Cookies(RefreshTokenCookieName); token != "" {
		return token
	}
	return ""
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

	// Verify CAPTCHA if enabled for signup
	if h.captchaService != nil {
		if err := h.captchaService.VerifyForEndpoint(c.Context(), "signup", req.CaptchaToken, c.IP()); err != nil {
			if errors.Is(err, auth.ErrCaptchaRequired) {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": "CAPTCHA verification required",
					"code":  "CAPTCHA_REQUIRED",
				})
			}
			log.Warn().Err(err).Str("email", req.Email).Msg("CAPTCHA verification failed for signup")
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "CAPTCHA verification failed",
				"code":  "CAPTCHA_INVALID",
			})
		}
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

	// Check if email verification is required (don't set cookies, no tokens returned)
	if resp.RequiresEmailVerification {
		return c.Status(fiber.StatusCreated).JSON(fiber.Map{
			"user":                        resp.User,
			"requires_email_verification": true,
			"message":                     "Please check your email to verify your account before signing in.",
		})
	}

	// Set httpOnly cookies for tokens
	h.setAuthCookies(c, resp.AccessToken, resp.RefreshToken, resp.ExpiresIn)

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

	// Verify CAPTCHA if enabled for login
	if h.captchaService != nil {
		if err := h.captchaService.VerifyForEndpoint(c.Context(), "login", req.CaptchaToken, c.IP()); err != nil {
			if errors.Is(err, auth.ErrCaptchaRequired) {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": "CAPTCHA verification required",
					"code":  "CAPTCHA_REQUIRED",
				})
			}
			log.Warn().Err(err).Str("email", req.Email).Msg("CAPTCHA verification failed for login")
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "CAPTCHA verification failed",
				"code":  "CAPTCHA_INVALID",
			})
		}
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
		// Check for locked account
		if errors.Is(err, auth.ErrAccountLocked) {
			log.Warn().Str("email", req.Email).Msg("Login attempt on locked account")
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Account locked due to too many failed login attempts. Please contact support.",
				"code":  "ACCOUNT_LOCKED",
			})
		}
		// Check for email not verified
		if errors.Is(err, auth.ErrEmailNotVerified) {
			log.Warn().Str("email", req.Email).Msg("Login attempt with unverified email")
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error":                       "Please verify your email address before signing in. Check your inbox for the verification link.",
				"code":                        "EMAIL_NOT_VERIFIED",
				"requires_email_verification": true,
			})
		}
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
		// Set httpOnly cookies for tokens
		h.setAuthCookies(c, resp.AccessToken, resp.RefreshToken, resp.ExpiresIn)
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

	// Set httpOnly cookies for tokens
	h.setAuthCookies(c, resp.AccessToken, resp.RefreshToken, resp.ExpiresIn)

	return c.Status(fiber.StatusOK).JSON(resp)
}

// SignOut handles user logout
// POST /auth/signout
func (h *AuthHandler) SignOut(c *fiber.Ctx) error {
	// Get token from cookie or Authorization header
	token := h.getAccessToken(c)
	if token == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "No authentication token provided",
		})
	}

	ctx := c.Context()

	// Get user ID from token before signing out
	var userID string
	if claims, err := h.authService.ValidateToken(token); err == nil {
		userID = claims.UserID
	}

	// Check if user has an active SAML session
	var samlLogoutInfo *fiber.Map
	if userID != "" && h.samlService != nil {
		samlSession, err := h.samlService.GetSAMLSessionByUserID(ctx, userID)
		if err == nil && samlSession != nil {
			// Check if provider has SLO support
			idpSloURL, _ := h.samlService.GetIdPSloURL(samlSession.ProviderName)
			if idpSloURL != "" && h.samlService.HasSigningKey(samlSession.ProviderName) {
				// SAML SLO is available - return the logout URL
				samlLogoutInfo = &fiber.Map{
					"saml_logout": true,
					"provider":    samlSession.ProviderName,
					"slo_url":     fmt.Sprintf("/auth/saml/logout/%s", samlSession.ProviderName),
				}
			} else {
				// No SLO support - clean up SAML session locally
				if err := h.samlService.DeleteSAMLSession(ctx, samlSession.ID); err != nil {
					log.Warn().Err(err).Msg("Failed to delete SAML session during signout")
				}
			}
		}
	}

	// Sign out user (invalidates JWT)
	if err := h.authService.SignOut(ctx, token); err != nil {
		log.Error().Err(err).Msg("Failed to sign out user")
		// Clear cookies even if sign out fails
		h.clearAuthCookies(c)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to sign out",
		})
	}

	// Clear authentication cookies
	h.clearAuthCookies(c)

	// Return response with SAML logout info if applicable
	if samlLogoutInfo != nil {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"message":     "Successfully signed out locally",
			"saml_logout": (*samlLogoutInfo)["saml_logout"],
			"provider":    (*samlLogoutInfo)["provider"],
			"slo_url":     (*samlLogoutInfo)["slo_url"],
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
		// Body parsing failed, try to get refresh token from cookie
		req.RefreshToken = h.getRefreshToken(c)
	}

	// If no refresh token in body, try cookie
	if req.RefreshToken == "" {
		req.RefreshToken = h.getRefreshToken(c)
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
		// Clear cookies on refresh failure
		h.clearAuthCookies(c)
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid or expired refresh token",
		})
	}

	// Set httpOnly cookies for new tokens
	h.setAuthCookies(c, resp.AccessToken, resp.RefreshToken, resp.ExpiresIn)

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
		Email        string `json:"email"`
		CaptchaToken string `json:"captcha_token,omitempty"`
	}
	if err := c.BodyParser(&req); err != nil {
		log.Error().Err(err).Msg("Failed to parse magic link request")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Verify CAPTCHA if enabled for magic_link
	if h.captchaService != nil {
		if err := h.captchaService.VerifyForEndpoint(c.Context(), "magic_link", req.CaptchaToken, c.IP()); err != nil {
			if errors.Is(err, auth.ErrCaptchaRequired) {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": "CAPTCHA verification required",
					"code":  "CAPTCHA_REQUIRED",
				})
			}
			log.Warn().Err(err).Str("email", req.Email).Msg("CAPTCHA verification failed for magic link")
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "CAPTCHA verification failed",
				"code":  "CAPTCHA_INVALID",
			})
		}
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
		Email        string `json:"email"`
		RedirectTo   string `json:"redirect_to,omitempty"`
		CaptchaToken string `json:"captcha_token,omitempty"`
	}

	if err := c.BodyParser(&req); err != nil {
		log.Error().Err(err).Msg("Failed to parse password reset request")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Verify CAPTCHA if enabled for password_reset
	if h.captchaService != nil {
		if err := h.captchaService.VerifyForEndpoint(c.Context(), "password_reset", req.CaptchaToken, c.IP()); err != nil {
			if errors.Is(err, auth.ErrCaptchaRequired) {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": "CAPTCHA verification required",
					"code":  "CAPTCHA_REQUIRED",
				})
			}
			log.Warn().Err(err).Str("email", req.Email).Msg("CAPTCHA verification failed for password reset")
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "CAPTCHA verification failed",
				"code":  "CAPTCHA_INVALID",
			})
		}
	}

	// Validate email
	if req.Email == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Email is required",
		})
	}

	// Request password reset (this won't reveal if user exists)
	if err := h.authService.RequestPasswordReset(c.Context(), req.Email, req.RedirectTo); err != nil {
		// Check for SMTP not configured error - this should be returned to the user
		if errors.Is(err, auth.ErrSMTPNotConfigured) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "SMTP is not configured. Please configure an email provider to enable password reset.",
				"code":  "SMTP_NOT_CONFIGURED",
			})
		}
		// Check for invalid redirect URL - return error to prevent misuse
		if errors.Is(err, auth.ErrInvalidRedirectURL) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid redirect_to URL. Must be a valid HTTP or HTTPS URL.",
				"code":  "INVALID_REDIRECT_URL",
			})
		}
		// Check for rate limiting - user requested reset too soon
		if errors.Is(err, auth.ErrPasswordResetTooSoon) {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": "Password reset requested too recently. Please wait 60 seconds before trying again.",
				"code":  "RATE_LIMITED",
			})
		}
		// Check for email sending failure - this should be returned to the user
		if errors.Is(err, auth.ErrEmailSendFailed) {
			log.Error().Err(err).Str("email", req.Email).Msg("Failed to send password reset email")
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to send password reset email. Please try again later.",
				"code":  "EMAIL_SEND_FAILED",
			})
		}
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

// VerifyEmail verifies a user's email address using a verification token
// POST /auth/verify-email
func (h *AuthHandler) VerifyEmail(c *fiber.Ctx) error {
	var req struct {
		Token string `json:"token"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.Token == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Token is required",
		})
	}

	user, err := h.authService.VerifyEmailToken(c.Context(), req.Token)
	if err != nil {
		// Check for specific token errors
		if errors.Is(err, auth.ErrEmailVerificationTokenNotFound) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid or expired verification token",
				"code":  "INVALID_TOKEN",
			})
		}
		if errors.Is(err, auth.ErrEmailVerificationTokenExpired) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Verification token has expired. Please request a new one.",
				"code":  "TOKEN_EXPIRED",
			})
		}
		if errors.Is(err, auth.ErrEmailVerificationTokenUsed) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "This verification token has already been used",
				"code":  "TOKEN_USED",
			})
		}
		log.Error().Err(err).Msg("Failed to verify email")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Email verified successfully. You can now sign in.",
		"user":    user,
	})
}

// ResendVerificationEmail resends the verification email to a user
// POST /auth/verify-email/resend
func (h *AuthHandler) ResendVerificationEmail(c *fiber.Ctx) error {
	var req struct {
		Email string `json:"email"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.Email == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Email is required",
		})
	}

	// Get user by email
	user, err := h.authService.GetUserByEmail(c.Context(), req.Email)
	if err != nil {
		// Don't reveal if email exists - return generic success message
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"message": "If an account exists with this email, a verification link has been sent.",
		})
	}

	// Check if already verified
	if user.EmailVerified {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"message": "Email is already verified. You can sign in.",
		})
	}

	// Send verification email
	if err := h.authService.SendEmailVerification(c.Context(), user.ID, user.Email); err != nil {
		log.Error().Err(err).Str("email", req.Email).Msg("Failed to resend verification email")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to send verification email. Please try again later.",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Verification email sent. Please check your inbox.",
	})
}

// RegisterRoutes registers all authentication routes with rate limiting
func (h *AuthHandler) RegisterRoutes(router fiber.Router, rateLimiters map[string]fiber.Handler) {
	// Register routes directly on the provided router (which should already be /api/v1/auth or similar)

	// CSRF token endpoint - clients should call this first to get a CSRF token
	// The CSRF middleware will set the csrf_token cookie on this request
	router.Get("/csrf", h.GetCSRFToken)

	// CAPTCHA configuration endpoint - returns public config (provider, site key)
	router.Get("/captcha/config", h.GetCaptchaConfig)

	// Auth configuration endpoint - returns all public auth config (signup, OAuth, SAML, CAPTCHA, password requirements)
	router.Get("/config", h.GetAuthConfig)

	// Public routes with rate limiting
	router.Post("/signup", rateLimiters["signup"], h.SignUp)
	router.Post("/signin", rateLimiters["login"], h.SignIn)
	// NOTE: Anonymous sign-in endpoint removed for security - reduces attack surface
	router.Post("/refresh", rateLimiters["refresh"], h.RefreshToken)
	router.Post("/magiclink", rateLimiters["magiclink"], h.SendMagicLink)
	router.Post("/magiclink/verify", h.VerifyMagicLink) // No rate limit on verification
	router.Post("/password/reset", rateLimiters["password_reset"], h.RequestPasswordReset)
	router.Post("/password/reset/confirm", h.ResetPassword)           // No rate limit on actual reset (token is single-use)
	router.Post("/password/reset/verify", h.VerifyPasswordResetToken) // No rate limit on verification

	// Email verification routes (public)
	router.Post("/verify-email", h.VerifyEmail)                                               // No rate limit on verification (token is single-use)
	router.Post("/verify-email/resend", rateLimiters["magiclink"], h.ResendVerificationEmail) // Use magiclink rate limiter

	// 2FA verification (public - used during login) with rate limiting
	// Rate limited to prevent brute-force attacks on 6-digit TOTP codes
	router.Post("/2fa/verify", rateLimiters["2fa"], h.VerifyTOTP)

	// OTP routes (public)
	router.Post("/otp/signin", rateLimiters["otp"], h.SendOTP)
	router.Post("/otp/verify", rateLimiters["2fa"], h.VerifyOTP) // Use 2FA rate limiter to prevent brute-force
	router.Post("/otp/resend", rateLimiters["otp"], h.ResendOTP)

	// ID token signin (public - for mobile OAuth)
	router.Post("/signin/idtoken", h.SignInWithIDToken)

	// Protected routes (authentication required) - lighter rate limits
	// Apply auth middleware to all routes below
	authMiddleware := AuthMiddleware(h.authService)
	router.Post("/signout", authMiddleware, h.SignOut)

	// User profile routes with scope enforcement
	router.Get("/user", authMiddleware, middleware.RequireScope(auth.ScopeAuthRead), h.GetUser)
	router.Patch("/user", authMiddleware, middleware.RequireScope(auth.ScopeAuthWrite), h.UpdateUser)

	// Admin impersonation routes (admin only) - no API key scope enforcement (admin-only feature)
	router.Post("/impersonate", authMiddleware, h.StartImpersonation)
	router.Post("/impersonate/anon", authMiddleware, h.StartAnonImpersonation)
	router.Post("/impersonate/service", authMiddleware, h.StartServiceImpersonation)
	router.Delete("/impersonate", authMiddleware, h.StopImpersonation)
	router.Get("/impersonate", authMiddleware, h.GetActiveImpersonation)
	router.Get("/impersonate/sessions", authMiddleware, h.ListImpersonationSessions)

	// 2FA routes (protected - authentication required) with scope enforcement
	router.Post("/2fa/setup", authMiddleware, middleware.RequireScope(auth.ScopeAuthWrite), h.SetupTOTP)
	router.Post("/2fa/enable", authMiddleware, middleware.RequireScope(auth.ScopeAuthWrite), h.EnableTOTP)
	router.Post("/2fa/disable", authMiddleware, middleware.RequireScope(auth.ScopeAuthWrite), h.DisableTOTP)
	router.Get("/2fa/status", authMiddleware, middleware.RequireScope(auth.ScopeAuthRead), h.GetTOTPStatus)

	// Identity linking routes (protected - authentication required) with scope enforcement
	router.Get("/user/identities", authMiddleware, middleware.RequireScope(auth.ScopeAuthRead), h.GetUserIdentities)
	router.Post("/user/identities", authMiddleware, middleware.RequireScope(auth.ScopeAuthWrite), h.LinkIdentity)
	router.Delete("/user/identities/:id", authMiddleware, middleware.RequireScope(auth.ScopeAuthWrite), h.UnlinkIdentity)

	// Reauthentication route (protected - authentication required)
	router.Post("/reauthenticate", authMiddleware, middleware.RequireScope(auth.ScopeAuthWrite), h.Reauthenticate)
}

// SignInAnonymous is deprecated and disabled for security reasons
// Anonymous sign-in reduces security by allowing anyone to get tokens
// Use regular signup/signin flow instead
func (h *AuthHandler) SignInAnonymous(c *fiber.Ctx) error {
	return c.Status(fiber.StatusGone).JSON(fiber.Map{
		"error": "Anonymous sign-in has been disabled for security reasons",
	})
}

// GetCSRFToken returns the current CSRF token for the client
// Clients should call this endpoint first, then include the token in the X-CSRF-Token header
// GET /auth/csrf
func (h *AuthHandler) GetCSRFToken(c *fiber.Ctx) error {
	// The CSRF middleware has already set the cookie
	// Return the token value so clients can use it in the X-CSRF-Token header
	token := c.Cookies("csrf_token")
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"csrf_token": token,
	})
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
		switch err {
		case auth.ErrNotAdmin:
			statusCode = fiber.StatusForbidden
		case auth.ErrSelfImpersonation:
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
	if err := auth.ValidateOTPContact(req.Email, req.Phone); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
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

	// Validate that either email or phone is provided
	if err := auth.ValidateOTPContact(req.Email, req.Phone); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	if req.Email != nil {
		otpCode, err = h.authService.VerifyOTP(c.Context(), *req.Email, req.Token)
	} else if req.Phone != nil {
		// Phone OTP not yet fully implemented
		return c.Status(fiber.StatusNotImplemented).JSON(fiber.Map{
			"error": "Phone-based OTP authentication not yet implemented",
		})
	}

	if err != nil {
		log.Warn().Err(err).Msg("Failed to verify OTP")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid or expired OTP code",
		})
	}

	// Get existing user - auto-creation is disabled for security
	// Users must register via signup endpoint first
	var user *auth.User
	if req.Email != nil && otpCode.Email != nil {
		user, err = h.authService.GetUserByEmail(c.Context(), *otpCode.Email)
		if err != nil {
			log.Warn().Str("email", *otpCode.Email).Msg("OTP verification for non-existent user")
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "No account found for this email - please sign up first",
			})
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

	// Validate that either email or phone is provided
	if err := auth.ValidateOTPContact(req.Email, req.Phone); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
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

// GetCaptchaConfig returns the public CAPTCHA configuration for clients
// GET /auth/captcha/config
func (h *AuthHandler) GetCaptchaConfig(c *fiber.Ctx) error {
	if h.captchaService == nil {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"enabled": false,
		})
	}

	config := h.captchaService.GetConfig()
	return c.Status(fiber.StatusOK).JSON(config)
}

// GetAuthConfig returns the public authentication configuration for clients
// GET /auth/config
func (h *AuthHandler) GetAuthConfig(c *fiber.Ctx) error {
	ctx := c.Context()
	settingsCache := h.authService.GetSettingsCache()

	// Build response
	response := AuthConfigResponse{
		SignupEnabled:            h.authService.IsSignupEnabled(),
		RequireEmailVerification: settingsCache.GetBool(ctx, "app.auth.require_email_verification", false),
		MagicLinkEnabled:         settingsCache.GetBool(ctx, "app.auth.magic_link_enabled", false),
		MFAAvailable:             true, // MFA is always available, users opt-in
		PasswordMinLength:        settingsCache.GetInt(ctx, "app.auth.password_min_length", 8),
		PasswordRequireUppercase: settingsCache.GetBool(ctx, "app.auth.password_require_uppercase", false),
		PasswordRequireLowercase: settingsCache.GetBool(ctx, "app.auth.password_require_lowercase", false),
		PasswordRequireNumber:    settingsCache.GetBool(ctx, "app.auth.password_require_number", false),
		PasswordRequireSpecial:   settingsCache.GetBool(ctx, "app.auth.password_require_special", false),
		OAuthProviders:           []OAuthProviderPublic{},
		SAMLProviders:            []SAMLProviderPublic{},
	}

	// Fetch OAuth providers
	oauthQuery := `
		SELECT provider_name, display_name, redirect_url
		FROM dashboard.oauth_providers
		WHERE enabled = TRUE AND allow_app_login = TRUE
		ORDER BY display_name
	`
	rows, err := h.db.Query(ctx, oauthQuery)
	if err != nil {
		log.Error().Err(err).Msg("Failed to list OAuth providers for auth config")
	} else {
		defer rows.Close()
		for rows.Next() {
			var providerName, displayName, redirectURL string
			if err := rows.Scan(&providerName, &displayName, &redirectURL); err != nil {
				log.Error().Err(err).Msg("Failed to scan OAuth provider")
				continue
			}
			response.OAuthProviders = append(response.OAuthProviders, OAuthProviderPublic{
				Provider:     providerName,
				DisplayName:  displayName,
				AuthorizeURL: fmt.Sprintf("%s/api/v1/auth/oauth/%s/authorize", h.baseURL, providerName),
			})
		}
	}

	// Fetch SAML providers
	if h.samlService != nil {
		samlProviders := h.samlService.GetProvidersForApp()
		for _, provider := range samlProviders {
			response.SAMLProviders = append(response.SAMLProviders, SAMLProviderPublic{
				Provider:    provider.Name,
				DisplayName: provider.Name, // SAML providers use Name as display name
			})
		}
	}

	// Get CAPTCHA config
	if h.captchaService != nil {
		captchaConfig := h.captchaService.GetConfig()
		response.Captcha = &captchaConfig
	} else {
		response.Captcha = &auth.CaptchaConfigResponse{
			Enabled: false,
		}
	}

	return c.Status(fiber.StatusOK).JSON(response)
}
