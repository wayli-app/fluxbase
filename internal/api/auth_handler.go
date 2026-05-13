package api

import (
	"errors"
	"fmt"

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
