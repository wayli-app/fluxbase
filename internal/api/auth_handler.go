package api

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/auth"
	"github.com/nimbleflux/fluxbase/internal/database"
	"github.com/nimbleflux/fluxbase/internal/middleware"
)

// Cookie names for authentication tokens
const (
	AccessTokenCookieName  = "fluxbase_access_token"
	RefreshTokenCookieName = "fluxbase_refresh_token"
)

// AuthHandler handles authentication HTTP requests
type AuthHandler struct {
	db                  *database.Connection
	authService         *auth.Service
	captchaService      *auth.CaptchaService
	captchaTrustService *auth.CaptchaTrustService
	samlService         *auth.SAMLService
	baseURL             string
	secureCookie        bool // Whether to set Secure flag on cookies (true in production)
}

// NewAuthHandler creates a new authentication handler
func NewAuthHandler(db *database.Connection, authService *auth.Service, captchaService *auth.CaptchaService, baseURL string) *AuthHandler {
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

// SetCaptchaTrustService sets the CAPTCHA trust service for adaptive verification
func (h *AuthHandler) SetCaptchaTrustService(trustService *auth.CaptchaTrustService) {
	h.captchaTrustService = trustService
}

// AuthConfigResponse represents the public authentication configuration
type AuthConfigResponse struct {
	SignupEnabled            bool                        `json:"signup_enabled"`
	RequireEmailVerification bool                        `json:"require_email_verification"`
	MagicLinkEnabled         bool                        `json:"magic_link_enabled"`
	PasswordLoginEnabled     bool                        `json:"password_login_enabled"`
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
func (h *AuthHandler) setAuthCookies(c fiber.Ctx, accessToken, refreshToken string, expiresIn int64) {
	// Access token cookie - shorter expiry
	// SameSite=Lax allows the cookie to be sent during top-level navigation
	// which is required for OAuth authorization flows from external clients
	c.Cookie(&fiber.Cookie{
		Name:     AccessTokenCookieName,
		Value:    accessToken,
		Path:     "/",
		MaxAge:   int(expiresIn), // seconds
		Secure:   h.secureCookie,
		HTTPOnly: true,
		SameSite: "Lax",
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
func (h *AuthHandler) clearAuthCookies(c fiber.Ctx) {
	c.Cookie(&fiber.Cookie{
		Name:     AccessTokenCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1, // Expire immediately
		Secure:   h.secureCookie,
		HTTPOnly: true,
		SameSite: "Lax",
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
func (h *AuthHandler) getAccessToken(c fiber.Ctx) string {
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
func (h *AuthHandler) getRefreshToken(c fiber.Ctx) string {
	// First try cookie
	if token := c.Cookies(RefreshTokenCookieName); token != "" {
		return token
	}
	return ""
}

// SignUp handles user registration
// POST /auth/signup
func (h *AuthHandler) SignUp(c fiber.Ctx) error {
	// Check if signup is enabled
	if !h.authService.IsSignupEnabled() {
		return SendErrorWithCode(c, 403, "User registration is currently disabled", "SIGNUP_DISABLED")
	}

	var req auth.SignUpRequest
	if err := ParseBody(c, &req); err != nil {
		return err
	}

	// CAPTCHA verification with adaptive trust support
	captchaVerified := false
	if h.captchaService != nil && h.captchaService.IsEnabled() {
		// If challenge_id is provided, validate the challenge first
		if req.ChallengeID != "" && h.captchaTrustService != nil {
			// Verify CAPTCHA token if one was provided
			if req.CaptchaToken != "" {
				if err := h.captchaService.Verify(middleware.CtxWithTenant(c), req.CaptchaToken, c.IP()); err != nil {
					log.Warn().Err(err).Str("email", req.Email).Msg("CAPTCHA verification failed for signup")
					return SendBadRequest(c, "CAPTCHA verification failed", "CAPTCHA_INVALID")
				}
				captchaVerified = true
			}

			// Validate the challenge (checks if CAPTCHA was required and if it was verified)
			if err := h.captchaTrustService.ValidateChallenge(middleware.CtxWithTenant(c), req.ChallengeID, "signup", c.IP(), captchaVerified); err != nil {
				if errors.Is(err, auth.ErrCaptchaRequired) {
					return SendBadRequest(c, "CAPTCHA verification required", "CAPTCHA_REQUIRED")
				}
				if errors.Is(err, auth.ErrChallengeExpired) {
					return SendBadRequest(c, "Challenge expired, please request a new one", "CHALLENGE_EXPIRED")
				}
				if errors.Is(err, auth.ErrChallengeConsumed) {
					return SendBadRequest(c, "Challenge already used, please request a new one", "CHALLENGE_CONSUMED")
				}
				log.Warn().Err(err).Str("email", req.Email).Msg("Challenge validation failed for signup")
				return SendBadRequest(c, "Invalid challenge", "CHALLENGE_INVALID")
			}
		} else {
			// Fall back to static CAPTCHA verification (no challenge_id provided)
			if err := h.captchaService.VerifyForEndpoint(middleware.CtxWithTenant(c), "signup", req.CaptchaToken, c.IP()); err != nil {
				if errors.Is(err, auth.ErrCaptchaRequired) {
					return SendBadRequest(c, "CAPTCHA verification required", "CAPTCHA_REQUIRED")
				}
				log.Warn().Err(err).Str("email", req.Email).Msg("CAPTCHA verification failed for signup")
				return SendBadRequest(c, "CAPTCHA verification failed", "CAPTCHA_INVALID")
			}
			captchaVerified = req.CaptchaToken != ""
		}
	}

	// Validate required fields
	if req.Email == "" {
		return SendMissingField(c, "Email")
	}
	if req.Password == "" {
		return SendMissingField(c, "Password")
	}

	// Create user
	resp, err := h.authService.SignUp(middleware.CtxWithTenant(c), req)
	if err != nil {
		log.Error().Err(err).Str("email", req.Email).Msg("Failed to sign up user")
		return SendBadRequest(c, "Registration failed", ErrCodeInvalidInput)
	}

	// Issue trust token if CAPTCHA was verified (for use in subsequent requests)
	var trustToken string
	if captchaVerified && h.captchaTrustService != nil && h.captchaTrustService.IsEnabled() {
		trustToken, _ = h.captchaTrustService.IssueTrustToken(middleware.CtxWithTenant(c), c.IP(), req.DeviceFingerprint, c.Get("User-Agent"))
	}

	// Check if email verification is required (don't set cookies, no tokens returned)
	if resp.RequiresEmailVerification {
		response := fiber.Map{
			"user":                        resp.User,
			"requires_email_verification": true,
			"message":                     "Please check your email to verify your account before signing in.",
		}
		if trustToken != "" {
			response["trust_token"] = trustToken
		}
		return c.Status(fiber.StatusCreated).JSON(response)
	}

	// Set httpOnly cookies for tokens
	h.setAuthCookies(c, resp.AccessToken, resp.RefreshToken, resp.ExpiresIn)

	// Add trust token to response if available
	if trustToken != "" {
		return c.Status(fiber.StatusCreated).JSON(fiber.Map{
			"user":          resp.User,
			"access_token":  resp.AccessToken,
			"refresh_token": resp.RefreshToken,
			"expires_in":    resp.ExpiresIn,
			"trust_token":   trustToken,
		})
	}

	return c.Status(fiber.StatusCreated).JSON(resp)
}

// SignIn handles user login
// POST /auth/signin
func (h *AuthHandler) SignIn(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)

	// Check if password login is disabled for app users
	if h.isPasswordLoginDisabled(ctx) {
		return SendErrorWithCode(c, 403, "Password login is disabled. Please use an OAuth or SAML provider to sign in.", "PASSWORD_LOGIN_DISABLED")
	}

	var req auth.SignInRequest
	if err := ParseBody(c, &req); err != nil {
		return err
	}

	// CAPTCHA verification with adaptive trust support
	captchaVerified := false
	if h.captchaService != nil && h.captchaService.IsEnabled() {
		// If challenge_id is provided, validate the challenge first
		if req.ChallengeID != "" && h.captchaTrustService != nil {
			// Verify CAPTCHA token if one was provided
			if req.CaptchaToken != "" {
				if err := h.captchaService.Verify(middleware.CtxWithTenant(c), req.CaptchaToken, c.IP()); err != nil {
					log.Warn().Err(err).Str("email", req.Email).Msg("CAPTCHA verification failed for login")
					return SendBadRequest(c, "CAPTCHA verification failed", "CAPTCHA_INVALID")
				}
				captchaVerified = true
			}

			// Validate the challenge (checks if CAPTCHA was required and if it was verified)
			if err := h.captchaTrustService.ValidateChallenge(middleware.CtxWithTenant(c), req.ChallengeID, "login", c.IP(), captchaVerified); err != nil {
				if errors.Is(err, auth.ErrCaptchaRequired) {
					return SendBadRequest(c, "CAPTCHA verification required", "CAPTCHA_REQUIRED")
				}
				if errors.Is(err, auth.ErrChallengeExpired) {
					return SendBadRequest(c, "Challenge expired, please request a new one", "CHALLENGE_EXPIRED")
				}
				if errors.Is(err, auth.ErrChallengeConsumed) {
					return SendBadRequest(c, "Challenge already used, please request a new one", "CHALLENGE_CONSUMED")
				}
				log.Warn().Err(err).Str("email", req.Email).Msg("Challenge validation failed for login")
				return SendBadRequest(c, "Invalid challenge", "CHALLENGE_INVALID")
			}
		} else {
			// Fall back to static CAPTCHA verification (no challenge_id provided)
			if err := h.captchaService.VerifyForEndpoint(middleware.CtxWithTenant(c), "login", req.CaptchaToken, c.IP()); err != nil {
				if errors.Is(err, auth.ErrCaptchaRequired) {
					return SendBadRequest(c, "CAPTCHA verification required", "CAPTCHA_REQUIRED")
				}
				log.Warn().Err(err).Str("email", req.Email).Msg("CAPTCHA verification failed for login")
				return SendBadRequest(c, "CAPTCHA verification failed", "CAPTCHA_INVALID")
			}
			captchaVerified = req.CaptchaToken != ""
		}
	}

	// Validate required fields
	if req.Email == "" || req.Password == "" {
		return SendBadRequest(c, "Email and password are required", ErrCodeInvalidInput)
	}

	// Authenticate user
	resp, err := h.authService.SignIn(middleware.CtxWithTenant(c), req)
	if err != nil {
		// Record failed attempt for trust tracking
		if h.captchaTrustService != nil {
			_ = h.captchaTrustService.RecordFailedAttempt(ctx, nil, c.IP(), req.DeviceFingerprint, c.Get("User-Agent"))
		}

		// Check for locked account
		if errors.Is(err, auth.ErrAccountLocked) {
			log.Warn().Str("email", req.Email).Msg("Login attempt on locked account")
			return SendErrorWithCode(c, 403, "Account locked due to too many failed login attempts. Please contact support.", "ACCOUNT_LOCKED")
		}
		// Check for email not verified
		if errors.Is(err, auth.ErrEmailNotVerified) {
			log.Warn().Str("email", req.Email).Msg("Login attempt with unverified email")
			return SendErrorWithDetails(c, 403, "Please verify your email address before signing in. Check your inbox for the verification link.", "EMAIL_NOT_VERIFIED", "", "", map[string]bool{"requires_email_verification": true})
		}
		log.Error().Err(err).Str("email", req.Email).Msg("Failed to sign in user")
		return SendUnauthorized(c, "Invalid email or password", ErrCodeInvalidCredentials)
	}

	// Record successful login for trust tracking
	if h.captchaTrustService != nil {
		if userUUID, err := uuid.Parse(resp.User.ID); err == nil {
			_ = h.captchaTrustService.RecordSuccessfulLogin(ctx, userUUID, c.IP(), req.DeviceFingerprint, c.Get("User-Agent"))
			if captchaVerified {
				_ = h.captchaTrustService.RecordCaptchaSolved(ctx, &userUUID, c.IP(), req.DeviceFingerprint, c.Get("User-Agent"))
			}
		}
	}

	// Issue trust token if CAPTCHA was verified (for use in subsequent requests)
	var trustToken string
	if captchaVerified && h.captchaTrustService != nil && h.captchaTrustService.IsEnabled() {
		trustToken, _ = h.captchaTrustService.IssueTrustToken(ctx, c.IP(), req.DeviceFingerprint, c.Get("User-Agent"))
	}

	// Check if user has 2FA enabled
	twoFAEnabled, err := h.authService.IsTOTPEnabled(middleware.CtxWithTenant(c), resp.User.ID)
	if err != nil {
		log.Error().Err(err).Str("user_id", resp.User.ID).Msg("Failed to check 2FA status")
		// Continue with login - don't block if 2FA check fails
		// Set httpOnly cookies for tokens
		h.setAuthCookies(c, resp.AccessToken, resp.RefreshToken, resp.ExpiresIn)
		if trustToken != "" {
			return c.Status(fiber.StatusOK).JSON(fiber.Map{
				"user":          resp.User,
				"access_token":  resp.AccessToken,
				"refresh_token": resp.RefreshToken,
				"expires_in":    resp.ExpiresIn,
				"trust_token":   trustToken,
			})
		}
		return c.Status(fiber.StatusOK).JSON(resp)
	}

	// If 2FA is enabled, return special response requiring 2FA verification
	if twoFAEnabled {
		response := fiber.Map{
			"requires_2fa": true,
			"user_id":      resp.User.ID,
			"message":      "2FA verification required. Please provide your 2FA code.",
		}
		if trustToken != "" {
			response["trust_token"] = trustToken
		}
		return c.Status(fiber.StatusOK).JSON(response)
	}

	// Set httpOnly cookies for tokens
	h.setAuthCookies(c, resp.AccessToken, resp.RefreshToken, resp.ExpiresIn)

	// Add trust token to response if available
	if trustToken != "" {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"user":          resp.User,
			"access_token":  resp.AccessToken,
			"refresh_token": resp.RefreshToken,
			"expires_in":    resp.ExpiresIn,
			"trust_token":   trustToken,
		})
	}

	return c.Status(fiber.StatusOK).JSON(resp)
}

// SignOut handles user logout
// POST /auth/signout
func (h *AuthHandler) SignOut(c fiber.Ctx) error {
	// Get token from cookie or Authorization header
	token := h.getAccessToken(c)
	if token == "" {
		return SendBadRequest(c, "No authentication token provided", ErrCodeMissingAuth)
	}

	ctx := middleware.CtxWithTenant(c)

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
		return SendInternalError(c, "Failed to sign out")
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
func (h *AuthHandler) RefreshToken(c fiber.Ctx) error {
	var req auth.RefreshTokenRequest
	if err := c.Bind().Body(&req); err != nil {
		// Body parsing failed, try to get refresh token from cookie
		req.RefreshToken = h.getRefreshToken(c)
	}

	// If no refresh token in body, try cookie
	if req.RefreshToken == "" {
		req.RefreshToken = h.getRefreshToken(c)
	}

	// Validate required fields
	if req.RefreshToken == "" {
		return SendMissingField(c, "Refresh token")
	}

	// Refresh token
	resp, err := h.authService.RefreshToken(middleware.CtxWithTenant(c), req)
	if err != nil {
		log.Error().Err(err).Msg("Failed to refresh token")
		// Clear cookies on refresh failure
		h.clearAuthCookies(c)
		return SendUnauthorized(c, "Invalid or expired refresh token", ErrCodeInvalidToken)
	}

	// Set httpOnly cookies for new tokens
	h.setAuthCookies(c, resp.AccessToken, resp.RefreshToken, resp.ExpiresIn)

	return c.Status(fiber.StatusOK).JSON(resp)
}

// GetUser handles getting current user profile
// GET /auth/user
func (h *AuthHandler) GetUser(c fiber.Ctx) error {
	// Get token from Authorization header
	token := c.Get("Authorization")
	if token == "" {
		return SendUnauthorized(c, "Authorization header is required", ErrCodeMissingAuth)
	}

	// Remove "Bearer " prefix if present
	if len(token) > 7 && token[:7] == "Bearer " {
		token = token[7:]
	}

	// Get user
	user, err := h.authService.GetUser(middleware.CtxWithTenant(c), token)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get user")
		return SendInvalidToken(c)
	}

	return c.Status(fiber.StatusOK).JSON(user)
}

// UpdateUser handles updating user profile
// PATCH /auth/user
func (h *AuthHandler) UpdateUser(c fiber.Ctx) error {
	// Get user ID from context (set by auth middleware)
	userID := middleware.GetUserID(c)
	if userID == "" {
		return SendMissingAuth(c)
	}

	var req auth.UpdateUserRequest
	if err := ParseBody(c, &req); err != nil {
		return err
	}

	user, err := h.authService.UpdateUser(middleware.CtxWithTenant(c), userID, req)
	if err != nil {
		log.Error().Err(err).Str("user_id", userID).Msg("Failed to update user")
		return SendBadRequest(c, "Failed to update user", ErrCodeInvalidInput)
	}

	return c.Status(fiber.StatusOK).JSON(user)
}

// SendMagicLink handles sending magic link
// POST /auth/magiclink
func (h *AuthHandler) SendMagicLink(c fiber.Ctx) error {
	var req struct {
		Email        string `json:"email"`
		CaptchaToken string `json:"captcha_token,omitempty"`
	}
	if err := ParseBody(c, &req); err != nil {
		return err
	}

	// Verify CAPTCHA if enabled for magic_link
	if h.captchaService != nil {
		if err := h.captchaService.VerifyForEndpoint(middleware.CtxWithTenant(c), "magic_link", req.CaptchaToken, c.IP()); err != nil {
			if errors.Is(err, auth.ErrCaptchaRequired) {
				return SendBadRequest(c, "CAPTCHA verification required", "CAPTCHA_REQUIRED")
			}
			log.Warn().Err(err).Str("email", req.Email).Msg("CAPTCHA verification failed for magic link")
			return SendBadRequest(c, "CAPTCHA verification failed", "CAPTCHA_INVALID")
		}
	}

	// Validate email
	if req.Email == "" {
		return SendMissingField(c, "Email")
	}

	// Send magic link
	if err := h.authService.SendMagicLink(middleware.CtxWithTenant(c), req.Email); err != nil {
		log.Error().Err(err).Str("email", req.Email).Msg("Failed to send magic link")
		return SendBadRequest(c, "Failed to send magic link", ErrCodeInvalidInput)
	}

	// Return standard OTP response
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"user":    nil,
		"session": nil,
	})
}

// VerifyMagicLink handles magic link verification
// POST /auth/magiclink/verify
func (h *AuthHandler) VerifyMagicLink(c fiber.Ctx) error {
	var req struct {
		Token string `json:"token"`
	}
	if err := ParseBody(c, &req); err != nil {
		return err
	}

	// Validate token
	if req.Token == "" {
		return SendMissingField(c, "Token")
	}

	// Verify magic link
	resp, err := h.authService.VerifyMagicLink(middleware.CtxWithTenant(c), req.Token)
	if err != nil {
		log.Error().Err(err).Msg("Failed to verify magic link")
		return SendBadRequest(c, "Invalid or expired magic link token", ErrCodeInvalidInput)
	}

	return c.Status(fiber.StatusOK).JSON(resp)
}

// RequestPasswordReset handles password reset requests
// POST /auth/password/reset
func (h *AuthHandler) RequestPasswordReset(c fiber.Ctx) error {
	var req struct {
		Email        string `json:"email"`
		RedirectTo   string `json:"redirect_to,omitempty"`
		CaptchaToken string `json:"captcha_token,omitempty"`
	}

	if err := ParseBody(c, &req); err != nil {
		return err
	}

	// Verify CAPTCHA if enabled for password_reset
	if h.captchaService != nil {
		if err := h.captchaService.VerifyForEndpoint(middleware.CtxWithTenant(c), "password_reset", req.CaptchaToken, c.IP()); err != nil {
			if errors.Is(err, auth.ErrCaptchaRequired) {
				return SendBadRequest(c, "CAPTCHA verification required", "CAPTCHA_REQUIRED")
			}
			log.Warn().Err(err).Str("email", req.Email).Msg("CAPTCHA verification failed for password reset")
			return SendBadRequest(c, "CAPTCHA verification failed", "CAPTCHA_INVALID")
		}
	}

	// Validate email
	if req.Email == "" {
		return SendMissingField(c, "Email")
	}

	// Request password reset (this won't reveal if user exists)
	if err := h.authService.RequestPasswordReset(middleware.CtxWithTenant(c), req.Email, req.RedirectTo); err != nil {
		// Check for SMTP not configured error - this should be returned to the user
		if errors.Is(err, auth.ErrSMTPNotConfigured) {
			return SendBadRequest(c, "SMTP is not configured. Please configure an email provider to enable password reset.", "SMTP_NOT_CONFIGURED")
		}
		// Check for invalid redirect URL - return error to prevent misuse
		if errors.Is(err, auth.ErrInvalidRedirectURL) {
			return SendBadRequest(c, "Invalid redirect_to URL. Must be a valid HTTP or HTTPS URL.", "INVALID_REDIRECT_URL")
		}
		// Check for rate limiting - user requested reset too soon
		if errors.Is(err, auth.ErrPasswordResetTooSoon) {
			return SendErrorWithCode(c, 429, "Password reset requested too recently. Please wait 60 seconds before trying again.", ErrCodeRateLimited)
		}
		// Check for email sending failure - this should be returned to the user
		if errors.Is(err, auth.ErrEmailSendFailed) {
			log.Error().Err(err).Str("email", req.Email).Msg("Failed to send password reset email")
			return SendInternalError(c, "Failed to send password reset email. Please try again later.")
		}
		log.Error().Err(err).Str("email", req.Email).Msg("Failed to request password reset")
		// Don't reveal if user exists - always return success
	}

	// Return standard OTP response
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"user":    nil,
		"session": nil,
	})
}

// ResetPassword handles password reset with token
// POST /auth/password/reset/confirm
func (h *AuthHandler) ResetPassword(c fiber.Ctx) error {
	var req struct {
		Token       string `json:"token"`
		NewPassword string `json:"new_password"`
	}

	if err := ParseBody(c, &req); err != nil {
		return err
	}

	// Validate required fields
	if req.Token == "" {
		return SendMissingField(c, "Token")
	}
	if req.NewPassword == "" {
		return SendMissingField(c, "New password")
	}

	// Reset password and get user ID
	userID, err := h.authService.ResetPassword(middleware.CtxWithTenant(c), req.Token, req.NewPassword)
	if err != nil {
		log.Error().Err(err).Msg("Failed to reset password")
		return SendBadRequest(c, "Invalid or expired reset token", ErrCodeInvalidInput)
	}

	// Generate new tokens for the user
	resp, err := h.authService.GenerateTokensForUser(middleware.CtxWithTenant(c), userID)
	if err != nil {
		log.Error().Err(err).Msg("Failed to generate tokens after password reset")
		return SendInternalError(c, "Failed to generate authentication tokens")
	}

	return c.Status(fiber.StatusOK).JSON(resp)
}

// VerifyPasswordResetToken handles password reset token verification
// POST /auth/password/reset/verify
func (h *AuthHandler) VerifyPasswordResetToken(c fiber.Ctx) error {
	var req struct {
		Token string `json:"token"`
	}

	if err := ParseBody(c, &req); err != nil {
		return err
	}

	// Validate token
	if req.Token == "" {
		return SendMissingField(c, "Token")
	}

	// Verify token
	if err := h.authService.VerifyPasswordResetToken(middleware.CtxWithTenant(c), req.Token); err != nil {
		log.Error().Err(err).Msg("Failed to verify password reset token")
		return SendBadRequest(c, "Invalid or expired reset token", ErrCodeInvalidInput)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Token is valid",
	})
}

// VerifyEmail verifies a user's email address using a verification token
// POST /auth/verify-email
func (h *AuthHandler) VerifyEmail(c fiber.Ctx) error {
	var req struct {
		Token string `json:"token"`
	}
	if err := ParseBody(c, &req); err != nil {
		return err
	}

	if req.Token == "" {
		return SendMissingField(c, "Token")
	}

	user, err := h.authService.VerifyEmailToken(middleware.CtxWithTenant(c), req.Token)
	if err != nil {
		// Check for specific token errors
		if errors.Is(err, auth.ErrEmailVerificationTokenNotFound) {
			return SendBadRequest(c, "Invalid or expired verification token", "INVALID_TOKEN")
		}
		if errors.Is(err, auth.ErrEmailVerificationTokenExpired) {
			return SendBadRequest(c, "Verification token has expired. Please request a new one.", "TOKEN_EXPIRED")
		}
		if errors.Is(err, auth.ErrEmailVerificationTokenUsed) {
			return SendBadRequest(c, "This verification token has already been used", "TOKEN_USED")
		}
		log.Error().Err(err).Msg("Failed to verify email")
		return SendBadRequest(c, "Email verification failed", ErrCodeInvalidInput)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Email verified successfully. You can now sign in.",
		"user":    user,
	})
}

// ResendVerificationEmail resends the verification email to a user
// POST /auth/verify-email/resend
func (h *AuthHandler) ResendVerificationEmail(c fiber.Ctx) error {
	var req struct {
		Email string `json:"email"`
	}
	if err := ParseBody(c, &req); err != nil {
		return err
	}

	if req.Email == "" {
		return SendMissingField(c, "Email")
	}

	// Get user by email
	user, err := h.authService.GetUserByEmail(middleware.CtxWithTenant(c), req.Email)
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
	if err := h.authService.SendEmailVerification(middleware.CtxWithTenant(c), user.ID, user.Email); err != nil {
		log.Error().Err(err).Str("email", req.Email).Msg("Failed to resend verification email")
		return SendInternalError(c, "Failed to send verification email. Please try again later.")
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Verification email sent. Please check your inbox.",
	})
}

// GetCSRFToken returns the current CSRF token for the client
// Clients should call this endpoint first, then include the token in the X-CSRF-Token header
// GET /auth/csrf
func (h *AuthHandler) GetCSRFToken(c fiber.Ctx) error {
	// The CSRF middleware has already set the cookie
	// Return the token value so clients can use it in the X-CSRF-Token header
	token := c.Cookies("csrf_token")
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"csrf_token": token,
	})
}

// StartImpersonation starts an admin impersonation session
func (h *AuthHandler) StartImpersonation(c fiber.Ctx) error {
	adminUserID := middleware.GetUserID(c)
	if adminUserID == "" {
		return SendMissingAuth(c)
	}

	var req auth.StartImpersonationRequest
	if err := ParseBody(c, &req); err != nil {
		return err
	}

	req.IPAddress = c.IP()
	req.UserAgent = c.Get("User-Agent")

	tenantID := c.Get("X-FB-Tenant")

	resp, err := h.authService.StartImpersonation(middleware.CtxWithTenant(c), adminUserID, tenantID, req)
	if err != nil {
		if errors.Is(err, auth.ErrNotAdmin) || errors.Is(err, auth.ErrNotTenantAdmin) {
			return SendForbidden(c, "Insufficient permissions", ErrCodeAccessDenied)
		} else if errors.Is(err, auth.ErrSelfImpersonation) {
			return SendBadRequest(c, "Cannot impersonate yourself", ErrCodeInvalidInput)
		} else if errors.Is(err, auth.ErrTargetUserNotInTenant) {
			return SendForbidden(c, "Target user is not in this tenant", ErrCodeAccessDenied)
		}
		return SendInternalError(c, "Failed to start impersonation")
	}

	return c.Status(fiber.StatusOK).JSON(resp)
}

// StopImpersonation stops the active impersonation session
func (h *AuthHandler) StopImpersonation(c fiber.Ctx) error {
	adminUserID := middleware.GetUserID(c)
	if adminUserID == "" {
		return SendMissingAuth(c)
	}

	err := h.authService.StopImpersonation(middleware.CtxWithTenant(c), adminUserID)
	if err != nil {
		if errors.Is(err, auth.ErrNoActiveImpersonation) {
			return SendNotFound(c, "No active impersonation session found")
		}
		return SendInternalError(c, "Failed to stop impersonation")
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Impersonation session ended",
	})
}

// GetActiveImpersonation gets the active impersonation session
func (h *AuthHandler) GetActiveImpersonation(c fiber.Ctx) error {
	adminUserID := middleware.GetUserID(c)
	if adminUserID == "" {
		return SendMissingAuth(c)
	}

	session, err := h.authService.GetActiveImpersonation(middleware.CtxWithTenant(c), adminUserID)
	if err != nil {
		if errors.Is(err, auth.ErrNoActiveImpersonation) {
			return SendNotFound(c, "No active impersonation session found")
		}
		return SendInternalError(c, "Failed to get active impersonation")
	}

	return c.Status(fiber.StatusOK).JSON(session)
}

// ListImpersonationSessions lists impersonation sessions for audit
func (h *AuthHandler) ListImpersonationSessions(c fiber.Ctx) error {
	adminUserID := middleware.GetUserID(c)
	if adminUserID == "" {
		return SendMissingAuth(c)
	}

	limit := fiber.Query[int](c, "limit", 50)
	offset := fiber.Query[int](c, "offset", 0)

	sessions, err := h.authService.ListImpersonationSessions(middleware.CtxWithTenant(c), adminUserID, limit, offset)
	if err != nil {
		return SendInternalError(c, "Failed to list impersonation sessions")
	}

	return c.Status(fiber.StatusOK).JSON(sessions)
}

// StartAnonImpersonation starts impersonation as anonymous user
func (h *AuthHandler) StartAnonImpersonation(c fiber.Ctx) error {
	adminUserID := middleware.GetUserID(c)
	if adminUserID == "" {
		return SendMissingAuth(c)
	}

	var req struct {
		Reason string `json:"reason"`
	}
	if err := ParseBody(c, &req); err != nil {
		return err
	}

	if req.Reason == "" {
		return SendMissingField(c, "Reason")
	}

	ipAddress := c.IP()
	userAgent := c.Get("User-Agent")
	tenantID := c.Get("X-FB-Tenant")

	resp, err := h.authService.StartAnonImpersonation(middleware.CtxWithTenant(c), adminUserID, tenantID, req.Reason, ipAddress, userAgent)
	if err != nil {
		if errors.Is(err, auth.ErrNotAdmin) || errors.Is(err, auth.ErrNotTenantAdmin) {
			return SendForbidden(c, "Insufficient permissions", ErrCodeAccessDenied)
		}
		return SendInternalError(c, "Failed to start anonymous impersonation")
	}

	return c.Status(fiber.StatusOK).JSON(resp)
}

func (h *AuthHandler) StartServiceImpersonation(c fiber.Ctx) error {
	adminUserID := middleware.GetUserID(c)
	if adminUserID == "" {
		return SendMissingAuth(c)
	}

	var req struct {
		Reason string `json:"reason"`
	}
	if err := ParseBody(c, &req); err != nil {
		return err
	}

	if req.Reason == "" {
		return SendMissingField(c, "Reason")
	}

	ipAddress := c.IP()
	userAgent := c.Get("User-Agent")
	tenantID := c.Get("X-FB-Tenant")

	resp, err := h.authService.StartServiceImpersonation(middleware.CtxWithTenant(c), adminUserID, tenantID, req.Reason, ipAddress, userAgent)
	if err != nil {
		if errors.Is(err, auth.ErrNotAdmin) || errors.Is(err, auth.ErrNotTenantAdmin) {
			return SendForbidden(c, "Insufficient permissions", ErrCodeAccessDenied)
		}
		return SendInternalError(c, "Failed to start service impersonation")
	}

	return c.Status(fiber.StatusOK).JSON(resp)
}

// SetupTOTP initiates 2FA setup by generating a TOTP secret
// POST /auth/2fa/setup
func (h *AuthHandler) SetupTOTP(c fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if userID == "" {
		return SendMissingAuth(c)
	}

	var req struct {
		Issuer string `json:"issuer"`
	}
	_ = c.Bind().Body(&req)

	response, err := h.authService.SetupTOTP(middleware.CtxWithTenant(c), userID, req.Issuer)
	if err != nil {
		log.Error().Err(err).Str("user_id", userID).Msg("Failed to setup TOTP")
		return SendInternalError(c, "Failed to setup 2FA")
	}

	return c.Status(fiber.StatusOK).JSON(response)
}

// EnableTOTP enables 2FA after verifying the TOTP code
// POST /auth/2fa/enable
func (h *AuthHandler) EnableTOTP(c fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if userID == "" {
		return SendMissingAuth(c)
	}

	var req struct {
		Code string `json:"code"`
	}
	if err := ParseBody(c, &req); err != nil {
		return err
	}

	if req.Code == "" {
		return SendMissingField(c, "Code")
	}

	backupCodes, err := h.authService.EnableTOTP(middleware.CtxWithTenant(c), userID, req.Code)
	if err != nil {
		log.Error().Err(err).Str("user_id", userID).Msg("Failed to enable TOTP")
		return SendBadRequest(c, "Invalid 2FA code", ErrCodeInvalidInput)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success":      true,
		"backup_codes": backupCodes,
		"message":      "2FA enabled successfully. Please save your backup codes in a secure location.",
	})
}

// VerifyTOTP verifies a TOTP code during login and issues JWT tokens
// POST /auth/2fa/verify
func (h *AuthHandler) VerifyTOTP(c fiber.Ctx) error {
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

	// Verify the 2FA code
	err := h.authService.VerifyTOTP(middleware.CtxWithTenant(c), req.UserID, req.Code)
	if err != nil {
		log.Warn().Err(err).Str("user_id", req.UserID).Msg("Failed to verify TOTP")
		return SendBadRequest(c, "Invalid 2FA code", ErrCodeInvalidCredentials)
	}

	// Generate a complete sign-in response with tokens
	resp, err := h.authService.GenerateTokensForUser(middleware.CtxWithTenant(c), req.UserID)
	if err != nil {
		log.Error().Err(err).Str("user_id", req.UserID).Msg("Failed to generate tokens after 2FA verification")
		return SendInternalError(c, "Failed to complete authentication")
	}

	return c.Status(fiber.StatusOK).JSON(resp)
}

// DisableTOTP disables 2FA for a user
// POST /auth/2fa/disable
func (h *AuthHandler) DisableTOTP(c fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if userID == "" {
		return SendMissingAuth(c)
	}

	var req struct {
		Password string `json:"password"`
	}
	if err := ParseBody(c, &req); err != nil {
		return err
	}

	if req.Password == "" {
		return SendMissingField(c, "Password")
	}

	err := h.authService.DisableTOTP(middleware.CtxWithTenant(c), userID, req.Password)
	if err != nil {
		log.Error().Err(err).Str("user_id", userID).Msg("Failed to disable TOTP")
		return SendBadRequest(c, "Failed to disable 2FA", ErrCodeInvalidCredentials)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"message": "2FA disabled successfully",
	})
}

// GetTOTPStatus checks if 2FA is enabled for a user
// GET /auth/2fa/status
func (h *AuthHandler) GetTOTPStatus(c fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if userID == "" {
		return SendMissingAuth(c)
	}

	enabled, err := h.authService.IsTOTPEnabled(middleware.CtxWithTenant(c), userID)
	if err != nil {
		log.Error().Err(err).Str("user_id", userID).Msg("Failed to check TOTP status")
		return SendInternalError(c, "Failed to check 2FA status")
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"totp_enabled": enabled,
	})
}

// SendOTP sends an OTP code via email or SMS
// POST /auth/otp/signin
func (h *AuthHandler) SendOTP(c fiber.Ctx) error {
	var req struct {
		Email   *string                 `json:"email,omitempty"`
		Phone   *string                 `json:"phone,omitempty"`
		Options *map[string]interface{} `json:"options,omitempty"`
	}

	if err := ParseBody(c, &req); err != nil {
		return err
	}

	// Validate that either email or phone is provided
	if err := auth.ValidateOTPContact(req.Email, req.Phone); err != nil {
		return SendBadRequest(c, "Email or phone is required", ErrCodeMissingField)
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
		err = h.authService.SendOTP(middleware.CtxWithTenant(c), *req.Email, purpose)
	} else if req.Phone != nil {
		// SMS OTP not yet fully implemented
		err = fmt.Errorf("SMS OTP not yet implemented")
	}

	if err != nil {
		log.Error().Str("error", err.Error()).Msg("Failed to send OTP")
		return SendInternalError(c, "Failed to send OTP code")
	}

	// Return standard OTP response
	// For send requests, user and session are both nil (OTP delivered but not verified yet)
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"user":    nil,
		"session": nil,
	})
}

// VerifyOTP verifies an OTP code and creates a session
// POST /auth/otp/verify
func (h *AuthHandler) VerifyOTP(c fiber.Ctx) error {
	var req struct {
		Email *string `json:"email,omitempty"`
		Phone *string `json:"phone,omitempty"`
		Token string  `json:"token"`
		Type  string  `json:"type"`
	}

	if err := ParseBody(c, &req); err != nil {
		return err
	}

	if req.Token == "" {
		return SendMissingField(c, "OTP token")
	}

	// Verify OTP
	var otpCode *auth.OTPCode
	var err error

	// Validate that either email or phone is provided
	if err := auth.ValidateOTPContact(req.Email, req.Phone); err != nil {
		return SendBadRequest(c, "Email or phone is required", ErrCodeMissingField)
	}

	if req.Email != nil {
		otpCode, err = h.authService.VerifyOTP(middleware.CtxWithTenant(c), *req.Email, req.Token)
	} else if req.Phone != nil {
		// Phone OTP not yet fully implemented
		return SendErrorWithCode(c, 501, "Phone-based OTP authentication not yet implemented", "NOT_IMPLEMENTED")
	}

	if err != nil {
		log.Warn().Err(err).Msg("Failed to verify OTP")
		return SendUnauthorized(c, "Invalid or expired OTP code", ErrCodeInvalidCredentials)
	}

	// Get existing user - auto-creation is disabled for security
	// Users must register via signup endpoint first
	var user *auth.User
	if req.Email != nil && otpCode.Email != nil {
		user, err = h.authService.GetUserByEmail(middleware.CtxWithTenant(c), *otpCode.Email)
		if err != nil {
			log.Warn().Str("email", *otpCode.Email).Msg("OTP verification for non-existent user")
			return SendNotFound(c, "No account found for this email - please sign up first")
		}
	}

	// Generate tokens
	resp, err := h.authService.GenerateTokensForUser(middleware.CtxWithTenant(c), user.ID)
	if err != nil {
		log.Error().Err(err).Msg("Failed to generate tokens")
		return SendInternalError(c, "Failed to complete authentication")
	}

	return c.Status(fiber.StatusOK).JSON(resp)
}

// ResendOTP resends an OTP code
// POST /auth/otp/resend
func (h *AuthHandler) ResendOTP(c fiber.Ctx) error {
	var req struct {
		Type    string                  `json:"type"`
		Email   *string                 `json:"email,omitempty"`
		Phone   *string                 `json:"phone,omitempty"`
		Options *map[string]interface{} `json:"options,omitempty"`
	}

	if err := ParseBody(c, &req); err != nil {
		return err
	}

	// Validate that either email or phone is provided
	if err := auth.ValidateOTPContact(req.Email, req.Phone); err != nil {
		return SendBadRequest(c, "Email or phone is required", ErrCodeMissingField)
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
		err = h.authService.ResendOTP(middleware.CtxWithTenant(c), *req.Email, purpose)
	} else if req.Phone != nil {
		// SMS OTP not yet fully implemented
		err = fmt.Errorf("SMS OTP not yet implemented")
	}

	if err != nil {
		log.Error().Err(err).Msg("Failed to resend OTP")
		return SendInternalError(c, "Failed to resend OTP code")
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"user":    nil,
		"session": nil,
	})
}

// GetUserIdentities gets all OAuth identities linked to a user
// GET /auth/user/identities
func (h *AuthHandler) GetUserIdentities(c fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if userID == "" {
		return SendMissingAuth(c)
	}

	identities, err := h.authService.GetUserIdentities(middleware.CtxWithTenant(c), userID)
	if err != nil {
		log.Error().Err(err).Str("user_id", userID).Msg("Failed to get user identities")
		return SendInternalError(c, "Failed to retrieve identities")
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"identities": identities,
	})
}

// LinkIdentity initiates OAuth flow to link a provider
// POST /auth/user/identities
func (h *AuthHandler) LinkIdentity(c fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if userID == "" {
		return SendMissingAuth(c)
	}

	var req struct {
		Provider string `json:"provider"`
	}

	if err := ParseBody(c, &req); err != nil {
		return err
	}

	if req.Provider == "" {
		return SendMissingField(c, "Provider")
	}

	authURL, state, err := h.authService.LinkIdentity(middleware.CtxWithTenant(c), userID, req.Provider)
	if err != nil {
		log.Error().Err(err).Str("provider", req.Provider).Msg("Failed to initiate identity linking")
		return SendBadRequest(c, "Failed to link identity", ErrCodeInvalidInput)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"url":      authURL,
		"provider": req.Provider,
		"state":    state,
	})
}

// UnlinkIdentity removes an OAuth identity from a user
// DELETE /auth/user/identities/:id
func (h *AuthHandler) UnlinkIdentity(c fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if userID == "" {
		return SendMissingAuth(c)
	}

	identityID := c.Params("id")
	if identityID == "" {
		return SendMissingField(c, "Identity ID")
	}

	err := h.authService.UnlinkIdentity(middleware.CtxWithTenant(c), userID, identityID)
	if err != nil {
		log.Error().Err(err).Str("identity_id", identityID).Msg("Failed to unlink identity")
		return SendBadRequest(c, "Failed to unlink identity", ErrCodeInvalidInput)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
	})
}

// Reauthenticate generates a security nonce
// POST /auth/reauthenticate
func (h *AuthHandler) Reauthenticate(c fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if userID == "" {
		return SendMissingAuth(c)
	}

	nonce, err := h.authService.Reauthenticate(middleware.CtxWithTenant(c), userID)
	if err != nil {
		log.Error().Err(err).Str("user_id", userID).Msg("Failed to reauthenticate")
		return SendInternalError(c, "Failed to generate security nonce")
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"nonce": nonce,
	})
}

// SignInWithIDToken handles OAuth ID token authentication (Google, Apple)
// POST /auth/signin/idtoken
func (h *AuthHandler) SignInWithIDToken(c fiber.Ctx) error {
	var req struct {
		Provider string  `json:"provider"`
		Token    string  `json:"token"`
		Nonce    *string `json:"nonce,omitempty"`
	}

	if err := ParseBody(c, &req); err != nil {
		return err
	}

	if req.Provider == "" || req.Token == "" {
		return SendBadRequest(c, "Provider and token are required", ErrCodeMissingField)
	}

	nonce := ""
	if req.Nonce != nil {
		nonce = *req.Nonce
	}

	resp, err := h.authService.SignInWithIDToken(middleware.CtxWithTenant(c), req.Provider, req.Token, nonce)
	if err != nil {
		log.Error().Err(err).Str("provider", req.Provider).Msg("Failed to sign in with ID token")
		return SendBadRequest(c, "Invalid ID token", ErrCodeInvalidCredentials)
	}

	return c.Status(fiber.StatusOK).JSON(resp)
}

// GetCaptchaConfig returns the public CAPTCHA configuration for clients
// GET /auth/captcha/config
func (h *AuthHandler) GetCaptchaConfig(c fiber.Ctx) error {
	if h.captchaService == nil {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"enabled": false,
		})
	}

	config := h.captchaService.GetConfig()
	return c.Status(fiber.StatusOK).JSON(config)
}

// CheckCaptcha performs a pre-flight check to determine if CAPTCHA is required
// POST /auth/captcha/check
//
// This endpoint evaluates trust signals and returns whether CAPTCHA verification
// is needed for the subsequent auth action. It issues a challenge_id that must
// be included in the actual auth request.
//
// Request body:
//
//	{
//	  "endpoint": "login",                    // Required: signup, login, password_reset, magic_link
//	  "email": "user@example.com",            // Optional: for trust lookup
//	  "device_fingerprint": "abc123",         // Optional: browser fingerprint
//	  "trust_token": "tt_..."                 // Optional: token from previous CAPTCHA
//	}
//
// Response:
//
//	{
//	  "captcha_required": true,
//	  "reason": "new_ip_address",
//	  "trust_score": 35,
//	  "provider": "hcaptcha",
//	  "site_key": "...",
//	  "challenge_id": "ch_abc123...",
//	  "expires_at": "2024-01-15T10:05:00Z"
//	}
func (h *AuthHandler) CheckCaptcha(c fiber.Ctx) error {
	// Parse request
	var req auth.CaptchaCheckRequest
	if err := ParseBody(c, &req); err != nil {
		return err
	}

	// Validate endpoint
	validEndpoints := map[string]bool{
		"signup":         true,
		"login":          true,
		"password_reset": true,
		"magic_link":     true,
	}
	if !validEndpoints[req.Endpoint] {
		return SendBadRequest(c, "Invalid endpoint. Must be one of: signup, login, password_reset, magic_link", "INVALID_ENDPOINT")
	}

	// If CAPTCHA is not enabled at all, return early
	if h.captchaService == nil || !h.captchaService.IsEnabled() {
		return c.Status(fiber.StatusOK).JSON(auth.CaptchaCheckResponse{
			CaptchaRequired: false,
			Reason:          "captcha_disabled",
			ChallengeID:     "", // No challenge needed
		})
	}

	// If adaptive trust service is available, use it
	if h.captchaTrustService != nil {
		response, err := h.captchaTrustService.CheckCaptchaRequired(middleware.CtxWithTenant(c), req, c.IP(), c.Get("User-Agent"))
		if err != nil {
			log.Error().Err(err).Msg("Failed to check CAPTCHA requirement")
			// Fall back to requiring CAPTCHA on error
			return c.Status(fiber.StatusOK).JSON(auth.CaptchaCheckResponse{
				CaptchaRequired: true,
				Reason:          "trust_check_error",
				Provider:        h.captchaService.GetProvider(),
				SiteKey:         h.captchaService.GetSiteKey(),
			})
		}
		return c.Status(fiber.StatusOK).JSON(response)
	}

	// Fall back to static check (adaptive trust not configured)
	required := h.captchaService.IsEnabledForEndpoint(req.Endpoint)
	response := auth.CaptchaCheckResponse{
		CaptchaRequired: required,
		ChallengeID:     "", // No challenge tracking without trust service
	}
	if required {
		response.Reason = "captcha_enabled_for_endpoint"
		response.Provider = h.captchaService.GetProvider()
		response.SiteKey = h.captchaService.GetSiteKey()
	} else {
		response.Reason = "captcha_not_required_for_endpoint"
	}

	return c.Status(fiber.StatusOK).JSON(response)
}

// GetAuthConfig returns the public authentication configuration for clients
// GET /auth/config
func (h *AuthHandler) GetAuthConfig(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)
	settingsCache := h.authService.GetSettingsCache()

	// Build response
	response := AuthConfigResponse{
		SignupEnabled:            h.authService.IsSignupEnabled(),
		RequireEmailVerification: settingsCache.GetBool(ctx, "app.auth.require_email_verification", false),
		MagicLinkEnabled:         settingsCache.GetBool(ctx, "app.auth.magic_link_enabled", false),
		PasswordLoginEnabled:     !settingsCache.GetBool(ctx, "app.auth.disable_app_password_login", false), // Inverted: disabled=false means enabled=true
		MFAAvailable:             true,                                                                      // MFA is always available, users opt-in
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
		FROM platform.oauth_providers
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

// isPasswordLoginDisabled checks if password login is disabled for app users
func (h *AuthHandler) isPasswordLoginDisabled(ctx context.Context) bool {
	// Emergency override via environment variable
	if os.Getenv("FLUXBASE_APP_FORCE_PASSWORD_LOGIN") == "true" {
		return false // Password login forced enabled
	}

	settingsCache := h.authService.GetSettingsCache()
	return settingsCache.GetBool(ctx, "app.auth.disable_app_password_login", false)
}

// fiber:context-methods migrated
