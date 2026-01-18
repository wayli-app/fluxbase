package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/fluxbase-eu/fluxbase/internal/config"
	"github.com/fluxbase-eu/fluxbase/internal/crypto"
	"github.com/fluxbase-eu/fluxbase/internal/database"
	"github.com/fluxbase-eu/fluxbase/internal/observability"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// Service provides a high-level authentication API
type Service struct {
	userRepo                *UserRepository
	sessionRepo             *SessionRepository
	magicLinkRepo           *MagicLinkRepository
	emailVerificationRepo   *EmailVerificationRepository
	jwtManager              *JWTManager
	passwordHasher          *PasswordHasher
	oauthManager            *OAuthManager
	magicLinkService        *MagicLinkService
	passwordResetService    *PasswordResetService
	tokenBlacklistService   *TokenBlacklistService
	impersonationService    *ImpersonationService
	otpService              *OTPService
	identityService         *IdentityService
	systemSettings          *SystemSettingsService
	settingsCache           *SettingsCache
	nonceRepo               *NonceRepository
	oidcVerifier            *OIDCVerifier
	config                  *config.AuthConfig
	emailService            RealEmailService
	baseURL                 string
	emailVerificationExpiry time.Duration
	metrics                 *observability.Metrics
	encryptionKey           string // 32-byte key for encrypting sensitive data (TOTP secrets)
	totpRateLimiter         *TOTPRateLimiter
}

// SetEncryptionKey sets the encryption key for encrypting sensitive data at rest
func (s *Service) SetEncryptionKey(key string) {
	s.encryptionKey = key
}

// SetTOTPRateLimiter sets the TOTP rate limiter for protecting against brute force attacks
func (s *Service) SetTOTPRateLimiter(limiter *TOTPRateLimiter) {
	s.totpRateLimiter = limiter
}

// SetMetrics sets the metrics instance for recording auth metrics
func (s *Service) SetMetrics(m *observability.Metrics) {
	s.metrics = m
}

// recordAuthAttempt records an authentication attempt to metrics
func (s *Service) recordAuthAttempt(method string, success bool, reason string) {
	if s.metrics != nil {
		s.metrics.RecordAuthAttempt(method, success, reason)
	}
}

// recordAuthToken records an issued auth token to metrics
func (s *Service) recordAuthToken(tokenType string) {
	if s.metrics != nil {
		s.metrics.RecordAuthToken(tokenType)
	}
}

// FullEmailService is a complete email service interface
// that includes both the basic EmailSender methods and a generic Send method
type FullEmailService interface {
	EmailSender
	Send(ctx context.Context, to, subject, body string) error
}

// NewService creates a new authentication service
func NewService(
	db *database.Connection,
	cfg *config.AuthConfig,
	emailService interface{},
	baseURL string,
) *Service {
	userRepo := NewUserRepository(db)
	sessionRepo := NewSessionRepository(db)
	magicLinkRepo := NewMagicLinkRepository(db)

	jwtManager := NewJWTManagerWithConfig(cfg.JWTSecret, cfg.JWTExpiry, cfg.RefreshExpiry, cfg.ServiceRoleTTL, cfg.AnonTTL)
	passwordHasher := NewPasswordHasherWithConfig(PasswordHasherConfig{MinLength: cfg.PasswordMinLen, Cost: cfg.BcryptCost})
	oauthManager := NewOAuthManager()

	// Cast email service to appropriate interfaces
	emailSender, _ := emailService.(EmailSender)
	realEmailService, _ := emailService.(RealEmailService)

	// Use configured expiry times with sensible fallbacks
	magicLinkExpiry := cfg.MagicLinkExpiry
	if magicLinkExpiry == 0 {
		magicLinkExpiry = 15 * time.Minute
	}

	magicLinkService := NewMagicLinkService(
		magicLinkRepo,
		userRepo,
		emailSender,
		magicLinkExpiry,
		baseURL,
	)

	passwordResetExpiry := cfg.PasswordResetExpiry
	if passwordResetExpiry == 0 {
		passwordResetExpiry = 1 * time.Hour
	}

	passwordResetRepo := NewPasswordResetRepository(db)
	passwordResetService := NewPasswordResetService(
		passwordResetRepo,
		userRepo,
		emailSender,
		passwordResetExpiry,
		baseURL,
	)

	tokenBlacklistRepo := NewTokenBlacklistRepository(db)
	tokenBlacklistService := NewTokenBlacklistService(tokenBlacklistRepo, jwtManager)

	impersonationRepo := NewImpersonationRepository(db)
	impersonationService := NewImpersonationService(impersonationRepo, userRepo, jwtManager, db)

	// OTP service for passwordless authentication
	otpExpiry := cfg.MagicLinkExpiry // Reuse magic link expiry for OTP (typically 10-15 minutes)
	if otpExpiry == 0 {
		otpExpiry = 10 * time.Minute
	}
	otpRepo := NewOTPRepository(db)
	// Create OTP sender that uses the email service
	// If email service doesn't support Send method, use NoOpOTPSender
	var otpSender OTPSender
	if realEmailService != nil {
		otpSender = NewDefaultOTPSender(realEmailService, "", "")
	} else {
		otpSender = &NoOpOTPSender{}
	}
	otpService := NewOTPService(otpRepo, userRepo, otpSender, otpExpiry)

	// Identity linking service
	stateStore := NewStateStore()
	identityRepo := NewIdentityRepository(db)
	identityService := NewIdentityService(identityRepo, oauthManager, stateStore)

	systemSettingsService := NewSystemSettingsService(db)
	settingsCache := NewSettingsCache(systemSettingsService, 30*time.Second)

	// Wire up cache to settings service for cache invalidation on updates
	systemSettingsService.SetCache(settingsCache)

	// Create nonce repository for distributed reauthentication
	nonceRepo := NewNonceRepository(db)

	// Create OIDC verifier for ID token authentication
	oidcVerifier, err := NewOIDCVerifier(context.Background(), cfg)
	if err != nil {
		// Log warning but continue - OIDC is optional
		// The error is already logged in NewOIDCVerifier
		oidcVerifier = &OIDCVerifier{
			verifiers: make(map[string]*oidc.IDTokenVerifier),
			providers: make(map[string]*oidc.Provider),
			clientIDs: make(map[string]string),
		}
	}

	// Email verification token expiry (default 24 hours)
	emailVerificationExpiry := 24 * time.Hour

	// Create email verification repository
	emailVerificationRepo := NewEmailVerificationRepository(db)

	return &Service{
		userRepo:                userRepo,
		sessionRepo:             sessionRepo,
		magicLinkRepo:           magicLinkRepo,
		emailVerificationRepo:   emailVerificationRepo,
		jwtManager:              jwtManager,
		passwordHasher:          passwordHasher,
		oauthManager:            oauthManager,
		magicLinkService:        magicLinkService,
		passwordResetService:    passwordResetService,
		tokenBlacklistService:   tokenBlacklistService,
		impersonationService:    impersonationService,
		otpService:              otpService,
		identityService:         identityService,
		systemSettings:          systemSettingsService,
		settingsCache:           settingsCache,
		nonceRepo:               nonceRepo,
		oidcVerifier:            oidcVerifier,
		config:                  cfg,
		emailService:            realEmailService,
		baseURL:                 baseURL,
		emailVerificationExpiry: emailVerificationExpiry,
	}
}

// SignUpRequest represents a user registration request
type SignUpRequest struct {
	Email        string                 `json:"email"`
	Password     string                 `json:"password"`
	UserMetadata map[string]interface{} `json:"user_metadata,omitempty"` // User-editable metadata
	AppMetadata  map[string]interface{} `json:"app_metadata,omitempty"`  // Application/admin-only metadata
	CaptchaToken string                 `json:"captcha_token,omitempty"` // CAPTCHA verification token
}

// SignUpResponse represents a successful registration response
type SignUpResponse struct {
	User                      *User  `json:"user"`
	AccessToken               string `json:"access_token,omitempty"`
	RefreshToken              string `json:"refresh_token,omitempty"`
	ExpiresIn                 int64  `json:"expires_in,omitempty"` // seconds
	RequiresEmailVerification bool   `json:"requires_email_verification,omitempty"`
}

// SignUp registers a new user with email and password
func (s *Service) SignUp(ctx context.Context, req SignUpRequest) (*SignUpResponse, error) {
	// Check if signup is enabled from database settings (with fallback to config)
	enableSignup := s.settingsCache.GetBool(ctx, "app.auth.signup_enabled", s.config.SignupEnabled)
	if !enableSignup {
		return nil, fmt.Errorf("signup is disabled")
	}

	// Validate email format and length
	if err := ValidateEmail(req.Email); err != nil {
		return nil, fmt.Errorf("invalid email: %w", err)
	}

	// Validate password
	if err := s.passwordHasher.ValidatePassword(req.Password); err != nil {
		return nil, fmt.Errorf("invalid password: %w", err)
	}

	// Hash password
	hashedPassword, err := s.passwordHasher.HashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user with metadata
	// NOTE: app_metadata is stripped from signup requests to prevent privilege escalation
	// Only admins can set app_metadata via user management endpoints
	user, err := s.userRepo.Create(ctx, CreateUserRequest{
		Email:        req.Email,
		UserMetadata: req.UserMetadata, // User-editable metadata
		AppMetadata:  nil,              // Stripped for security - admin-only field
	}, hashedPassword)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Check if email verification is required
	if s.IsEmailVerificationRequired(ctx) {
		// Send verification email (don't fail signup if email fails)
		if err := s.SendEmailVerification(ctx, user.ID, user.Email); err != nil {
			// Log error but don't fail the signup - user was created successfully
			LogSecurityEvent(ctx, SecurityEvent{
				Type:   SecurityEventLoginFailed,
				UserID: user.ID,
				Email:  user.Email,
				Details: map[string]interface{}{
					"reason": "failed_to_send_verification_email",
					"error":  err.Error(),
				},
			})
		}

		// Return response WITHOUT tokens - user needs to verify email first
		return &SignUpResponse{
			User:                      user,
			RequiresEmailVerification: true,
		}, nil
	}

	// Generate tokens with metadata
	accessToken, refreshToken, _, err := s.jwtManager.GenerateTokenPair(
		user.ID,
		user.Email,
		user.Role,
		user.UserMetadata,
		user.AppMetadata,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	// Create session
	expiresAt := time.Now().Add(s.config.RefreshExpiry)
	_, err = s.sessionRepo.Create(ctx, user.ID, accessToken, refreshToken, expiresAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return &SignUpResponse{
		User:         user,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(s.config.JWTExpiry.Seconds()),
	}, nil
}

// SignInRequest represents a login request
type SignInRequest struct {
	Email        string `json:"email"`
	Password     string `json:"password"`
	CaptchaToken string `json:"captcha_token,omitempty"` // CAPTCHA verification token
}

// SignInResponse represents a successful login response
type SignInResponse struct {
	User         *User  `json:"user"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"` // seconds
}

// SignIn authenticates a user with email and password
func (s *Service) SignIn(ctx context.Context, req SignInRequest) (*SignInResponse, error) {
	// Get user by email
	user, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			// Log failed login attempt for non-existent user
			LogSecurityEvent(ctx, SecurityEvent{
				Type:  SecurityEventLoginFailed,
				Email: req.Email,
				Details: map[string]interface{}{
					"reason": "user_not_found",
				},
			})
			s.recordAuthAttempt("password", false, "user_not_found")
			return nil, fmt.Errorf("invalid email or password")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Check if account is locked
	if user.IsLocked {
		// Check if lock has expired (if locked_until is set)
		if user.LockedUntil != nil && time.Now().After(*user.LockedUntil) {
			// Lock expired, reset it
			if err := s.userRepo.ResetFailedLoginAttempts(ctx, user.ID); err != nil {
				// Log error but continue - worst case user stays locked
				_ = err
			}
			LogSecurityEvent(ctx, SecurityEvent{
				Type:   SecurityEventAccountUnlocked,
				UserID: user.ID,
				Email:  user.Email,
				Details: map[string]interface{}{
					"reason": "lock_expired",
				},
			})
		} else {
			// Log locked account access attempt
			LogSecurityWarning(ctx, SecurityEvent{
				Type:   SecurityEventLoginFailed,
				UserID: user.ID,
				Email:  user.Email,
				Details: map[string]interface{}{
					"reason": "account_locked",
				},
			})
			s.recordAuthAttempt("password", false, "account_locked")
			return nil, ErrAccountLocked
		}
	}

	// Verify password
	if err := s.passwordHasher.ComparePassword(user.PasswordHash, req.Password); err != nil {
		// Increment failed login attempts
		if incErr := s.userRepo.IncrementFailedLoginAttempts(ctx, user.ID); incErr != nil {
			// Log error but return generic invalid credentials
			_ = incErr
		}

		// Check if account is now locked (after 5 failed attempts)
		failedAttempts := user.FailedLoginAttempts + 1
		if failedAttempts >= 5 {
			LogSecurityWarning(ctx, SecurityEvent{
				Type:   SecurityEventAccountLocked,
				UserID: user.ID,
				Email:  user.Email,
				Details: map[string]interface{}{
					"failed_attempts": failedAttempts,
				},
			})
		} else {
			LogSecurityEvent(ctx, SecurityEvent{
				Type:   SecurityEventLoginFailed,
				UserID: user.ID,
				Email:  user.Email,
				Details: map[string]interface{}{
					"reason":          "invalid_password",
					"failed_attempts": failedAttempts,
				},
			})
		}

		s.recordAuthAttempt("password", false, "invalid_password")
		return nil, fmt.Errorf("invalid email or password")
	}

	// Reset failed login attempts on successful login
	if user.FailedLoginAttempts > 0 {
		if err := s.userRepo.ResetFailedLoginAttempts(ctx, user.ID); err != nil {
			// Log error but continue with login
			_ = err
		}
	}

	// Check if email verification is required and user's email is not verified
	if s.IsEmailVerificationRequired(ctx) && !user.EmailVerified {
		LogSecurityEvent(ctx, SecurityEvent{
			Type:   SecurityEventLoginFailed,
			UserID: user.ID,
			Email:  user.Email,
			Details: map[string]interface{}{
				"reason": "email_not_verified",
			},
		})
		s.recordAuthAttempt("password", false, "email_not_verified")
		return nil, ErrEmailNotVerified
	}

	// Log successful login
	LogSecurityEvent(ctx, SecurityEvent{
		Type:   SecurityEventLoginSuccess,
		UserID: user.ID,
		Email:  user.Email,
	})

	// Generate tokens with metadata
	accessToken, refreshToken, _, err := s.jwtManager.GenerateTokenPair(
		user.ID,
		user.Email,
		user.Role,
		user.UserMetadata,
		user.AppMetadata,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	// Create session
	expiresAt := time.Now().Add(s.config.RefreshExpiry)
	_, err = s.sessionRepo.Create(ctx, user.ID, accessToken, refreshToken, expiresAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	// Record successful auth and token issuance
	s.recordAuthAttempt("password", true, "")
	s.recordAuthToken("access")
	s.recordAuthToken("refresh")

	return &SignInResponse{
		User:         user,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(s.config.JWTExpiry.Seconds()),
	}, nil
}

// SignOut logs out a user by invalidating their session
func (s *Service) SignOut(ctx context.Context, accessToken string) error {
	// Blacklist the access token first
	if err := s.tokenBlacklistService.RevokeToken(ctx, accessToken, "logout"); err != nil {
		// Log error but continue with session deletion
		// Revocation failure shouldn't block logout
		_ = err // nolint:staticcheck // Intentionally ignored
	}

	// Get session by access token
	session, err := s.sessionRepo.GetByAccessToken(ctx, accessToken)
	if err != nil {
		if errors.Is(err, ErrSessionNotFound) {
			// Already signed out or invalid token
			return nil
		}
		return fmt.Errorf("failed to get session: %w", err)
	}

	// Delete session
	if err := s.sessionRepo.Delete(ctx, session.ID); err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	return nil
}

// RefreshTokenRequest represents a token refresh request
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// RefreshTokenResponse represents a successful token refresh
type RefreshTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"` // seconds
}

// RefreshToken generates new access and refresh tokens using a refresh token (token rotation)
// SECURITY: Implements refresh token rotation - each refresh generates a new refresh token
// and invalidates the old one. This limits the window of opportunity for stolen tokens.
func (s *Service) RefreshToken(ctx context.Context, req RefreshTokenRequest) (*RefreshTokenResponse, error) {
	// Validate refresh token
	claims, err := s.jwtManager.ValidateToken(req.RefreshToken)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}

	if claims.TokenType != "refresh" {
		return nil, fmt.Errorf("invalid token type")
	}

	// Get session by refresh token
	session, err := s.sessionRepo.GetByRefreshToken(ctx, req.RefreshToken)
	if err != nil {
		if errors.Is(err, ErrSessionNotFound) {
			// SECURITY: If the session is not found but the token is valid, it may indicate
			// that a stolen token was used after the legitimate user rotated it.
			// Log this as a potential security event.
			log.Warn().
				Str("user_id", claims.UserID).
				Str("session_id", claims.SessionID).
				Msg("Valid refresh token used but session not found - possible token theft detected")
			return nil, fmt.Errorf("session not found or expired")
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	// Check if session is expired
	if time.Now().After(session.ExpiresAt) {
		return nil, fmt.Errorf("session expired")
	}

	// Get user to include metadata in new tokens
	user, err := s.userRepo.GetByID(ctx, claims.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Generate new access token
	newAccessToken, _, err := s.jwtManager.GenerateAccessToken(
		claims.UserID,
		claims.Email,
		user.Role,
		claims.UserMetadata,
		claims.AppMetadata,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	// Generate new refresh token (rotation)
	newRefreshToken, _, err := s.jwtManager.GenerateRefreshToken(
		claims.UserID,
		claims.Email,
		user.Role,
		claims.SessionID,
		claims.UserMetadata,
		claims.AppMetadata,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Calculate new expiry (extend session)
	newExpiresAt := time.Now().Add(s.config.RefreshExpiry)

	// Update session with new tokens (rotation)
	if err := s.sessionRepo.UpdateTokens(ctx, session.ID, newAccessToken, newRefreshToken, newExpiresAt); err != nil {
		return nil, fmt.Errorf("failed to update session: %w", err)
	}

	return &RefreshTokenResponse{
		AccessToken:  newAccessToken,
		RefreshToken: newRefreshToken, // New rotated refresh token
		ExpiresIn:    int64(s.config.JWTExpiry.Seconds()),
	}, nil
}

// GetUser retrieves the current user by access token
func (s *Service) GetUser(ctx context.Context, accessToken string) (*User, error) {
	// Validate token
	claims, err := s.jwtManager.ValidateToken(accessToken)
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	// Verify session still exists (not signed out)
	_, err = s.sessionRepo.GetByAccessToken(ctx, accessToken)
	if err != nil {
		if errors.Is(err, ErrSessionNotFound) {
			return nil, fmt.Errorf("session not found or expired")
		}
		return nil, fmt.Errorf("failed to verify session: %w", err)
	}

	// Get user
	user, err := s.userRepo.GetByID(ctx, claims.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

// UpdateUser updates user information
func (s *Service) UpdateUser(ctx context.Context, userID string, req UpdateUserRequest) (*User, error) {
	// Validate email if provided
	if req.Email != nil {
		if err := ValidateEmail(*req.Email); err != nil {
			return nil, fmt.Errorf("invalid email: %w", err)
		}
	}
	return s.userRepo.Update(ctx, userID, req)
}

// SendMagicLink sends a magic link to the specified email
func (s *Service) SendMagicLink(ctx context.Context, email string) error {
	// Check if magic link is enabled from database settings (with fallback to config)
	enableMagicLink := s.settingsCache.GetBool(ctx, "app.auth.magic_link_enabled", s.config.MagicLinkEnabled)
	if !enableMagicLink {
		return fmt.Errorf("magic link authentication is disabled")
	}

	return s.magicLinkService.SendMagicLink(ctx, email)
}

// VerifyMagicLink verifies a magic link and returns tokens
func (s *Service) VerifyMagicLink(ctx context.Context, token string) (*SignInResponse, error) {
	// Check if magic link is enabled from database settings (with fallback to config)
	enableMagicLink := s.settingsCache.GetBool(ctx, "app.auth.magic_link_enabled", s.config.MagicLinkEnabled)
	if !enableMagicLink {
		return nil, fmt.Errorf("magic link authentication is disabled")
	}

	// Verify the magic link
	email, err := s.magicLinkService.VerifyMagicLink(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("failed to verify magic link: %w", err)
	}

	// Get existing user - auto-creation is disabled for security
	// Users must register via signup endpoint first
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, fmt.Errorf("no account found for this email - please sign up first")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Generate tokens with metadata
	accessToken, refreshToken, _, err := s.jwtManager.GenerateTokenPair(
		user.ID,
		user.Email,
		user.Role,
		user.UserMetadata,
		user.AppMetadata,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	// Create session
	expiresAt := time.Now().Add(s.config.RefreshExpiry)
	_, err = s.sessionRepo.Create(ctx, user.ID, accessToken, refreshToken, expiresAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return &SignInResponse{
		User:         user,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(s.config.JWTExpiry.Seconds()),
	}, nil
}

// ValidateToken validates an access token and returns the claims
func (s *Service) ValidateToken(token string) (*TokenClaims, error) {
	return s.jwtManager.ValidateToken(token)
}

// ValidateServiceRoleToken validates a JWT containing a role claim (anon, service_role, authenticated)
// This is used for Supabase-compatible client keys which are JWTs with role claims.
// Unlike user tokens, these don't require user lookup or revocation checks.
func (s *Service) ValidateServiceRoleToken(token string) (*TokenClaims, error) {
	return s.jwtManager.ValidateServiceRoleToken(token)
}

// GetOAuthManager returns the OAuth manager for configuring providers
func (s *Service) GetOAuthManager() *OAuthManager {
	return s.oauthManager
}

// RequestPasswordReset sends a password reset email
// If redirectTo is provided, the email link will point to that URL instead of the default.
func (s *Service) RequestPasswordReset(ctx context.Context, email string, redirectTo string) error {
	return s.passwordResetService.RequestPasswordReset(ctx, email, redirectTo)
}

// ResetPassword resets a user's password using a valid reset token
func (s *Service) ResetPassword(ctx context.Context, token, newPassword string) (string, error) {
	return s.passwordResetService.ResetPassword(ctx, token, newPassword)
}

// VerifyPasswordResetToken verifies if a password reset token is valid
func (s *Service) VerifyPasswordResetToken(ctx context.Context, token string) error {
	return s.passwordResetService.VerifyPasswordResetToken(ctx, token)
}

// RevokeToken revokes a specific JWT token
func (s *Service) RevokeToken(ctx context.Context, token, reason string) error {
	return s.tokenBlacklistService.RevokeToken(ctx, token, reason)
}

// IsTokenRevoked checks if a JWT token has been revoked
func (s *Service) IsTokenRevoked(ctx context.Context, jti string) (bool, error) {
	return s.tokenBlacklistService.IsTokenRevoked(ctx, jti)
}

// RevokeAllUserTokens revokes all tokens for a specific user
func (s *Service) RevokeAllUserTokens(ctx context.Context, userID, reason string) error {
	return s.tokenBlacklistService.RevokeAllUserTokens(ctx, userID, reason)
}

// SignInAnonymousResponse represents an anonymous user sign-in response
type SignInAnonymousResponse struct {
	UserID       string `json:"user_id"` // Temporary anonymous user ID
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`   // seconds
	IsAnonymous  bool   `json:"is_anonymous"` // Always true for anonymous users
}

// SignInAnonymous creates JWT tokens for an anonymous user (no database record)
func (s *Service) SignInAnonymous(ctx context.Context) (*SignInAnonymousResponse, error) {
	// Generate a random UUID for the anonymous user
	// This ID exists only in the JWT token, not in the database
	anonymousUserID := uuid.New().String()

	// Generate JWT tokens with is_anonymous flag in claims
	accessToken, err := s.jwtManager.GenerateAnonymousAccessToken(anonymousUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := s.jwtManager.GenerateAnonymousRefreshToken(anonymousUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return &SignInAnonymousResponse{
		UserID:       anonymousUserID,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(s.config.JWTExpiry.Seconds()),
		IsAnonymous:  true,
	}, nil
}

// Impersonation wrapper methods

// StartImpersonation starts an admin impersonation session
func (s *Service) StartImpersonation(ctx context.Context, adminUserID string, req StartImpersonationRequest) (*StartImpersonationResponse, error) {
	return s.impersonationService.StartImpersonation(ctx, adminUserID, req)
}

// StopImpersonation stops the active impersonation session for an admin
func (s *Service) StopImpersonation(ctx context.Context, adminUserID string) error {
	return s.impersonationService.StopImpersonation(ctx, adminUserID)
}

// GetActiveImpersonation gets the active impersonation session for an admin
func (s *Service) GetActiveImpersonation(ctx context.Context, adminUserID string) (*ImpersonationSession, error) {
	return s.impersonationService.GetActiveSession(ctx, adminUserID)
}

// ListImpersonationSessions lists impersonation sessions for audit purposes
func (s *Service) ListImpersonationSessions(ctx context.Context, adminUserID string, limit, offset int) ([]*ImpersonationSession, error) {
	return s.impersonationService.ListSessions(ctx, adminUserID, limit, offset)
}

// StartAnonImpersonation starts an impersonation session as anonymous user
func (s *Service) StartAnonImpersonation(ctx context.Context, adminUserID string, reason string, ipAddress string, userAgent string) (*StartImpersonationResponse, error) {
	return s.impersonationService.StartAnonImpersonation(ctx, adminUserID, reason, ipAddress, userAgent)
}

// StartServiceImpersonation starts an impersonation session with service role
func (s *Service) StartServiceImpersonation(ctx context.Context, adminUserID string, reason string, ipAddress string, userAgent string) (*StartImpersonationResponse, error) {
	return s.impersonationService.StartServiceImpersonation(ctx, adminUserID, reason, ipAddress, userAgent)
}

// IsSignupEnabled returns whether user signup is enabled
func (s *Service) IsSignupEnabled() bool {
	// Use background context for health check endpoint
	ctx := context.Background()
	return s.settingsCache.GetBool(ctx, "app.auth.signup_enabled", s.config.SignupEnabled)
}

// GetSettingsCache returns the settings cache
func (s *Service) GetSettingsCache() *SettingsCache {
	return s.settingsCache
}

// GetAccessTokenExpirySeconds returns the configured JWT access token expiry in seconds
func (s *Service) GetAccessTokenExpirySeconds() int64 {
	return int64(s.config.JWTExpiry.Seconds())
}

// TOTPSetupResponse represents the TOTP setup data
type TOTPSetupResponse struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	TOTP struct {
		QRCode string `json:"qr_code"`
		Secret string `json:"secret"`
		URI    string `json:"uri"`
	} `json:"totp"`
}

// SetupTOTP generates a new TOTP secret for 2FA setup
// If issuer is empty, uses the configured default from AuthConfig.TOTPIssuer
func (s *Service) SetupTOTP(ctx context.Context, userID string, issuer string) (*TOTPSetupResponse, error) {
	// Use provided issuer, or fall back to configured default
	if issuer == "" {
		issuer = s.config.TOTPIssuer
	}

	// Generate TOTP secret and QR code
	secret, qrCodeDataURI, otpauthURI, err := GenerateTOTPSecret(issuer, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate TOTP secret: %w", err)
	}

	// Generate a unique factor ID
	factorID := uuid.New().String()

	// Store the secret in a temporary setup table (expires in 10 minutes)
	query := `
		INSERT INTO auth.two_factor_setups (user_id, factor_id, secret, qr_code_data_uri, otpauth_uri, expires_at)
		VALUES ($1, $2, $3, $4, $5, NOW() + INTERVAL '10 minutes')
		ON CONFLICT (user_id) DO UPDATE
			SET factor_id = EXCLUDED.factor_id,
			    secret = EXCLUDED.secret,
			    qr_code_data_uri = EXCLUDED.qr_code_data_uri,
			    otpauth_uri = EXCLUDED.otpauth_uri,
			    expires_at = EXCLUDED.expires_at,
			    verified = FALSE
	`

	_, err = s.userRepo.db.Pool().Exec(ctx, query, userID, factorID, secret, qrCodeDataURI, otpauthURI)
	if err != nil {
		return nil, fmt.Errorf("failed to store TOTP setup: %w", err)
	}

	// Build response in Supabase-compatible format
	response := &TOTPSetupResponse{
		ID:   factorID,
		Type: "totp",
	}
	response.TOTP.QRCode = qrCodeDataURI
	response.TOTP.Secret = secret
	response.TOTP.URI = otpauthURI

	return response, nil
}

// EnableTOTP enables 2FA after verifying the TOTP code
func (s *Service) EnableTOTP(ctx context.Context, userID, code string) ([]string, error) {
	// Fetch the pending TOTP setup
	var secret string
	var expiresAt time.Time
	query := `
		SELECT secret, expires_at
		FROM auth.two_factor_setups
		WHERE user_id = $1 AND verified = FALSE
	`

	err := s.userRepo.db.Pool().QueryRow(ctx, query, userID).Scan(&secret, &expiresAt)
	if err != nil {
		return nil, fmt.Errorf("2FA setup not found or expired: %w", err)
	}

	// Check if setup has expired
	if time.Now().After(expiresAt) {
		return nil, errors.New("2FA setup has expired, please start again")
	}

	// Verify the TOTP code
	valid, err := VerifyTOTPCode(code, secret)
	if err != nil {
		return nil, fmt.Errorf("failed to verify TOTP code: %w", err)
	}

	if !valid {
		return nil, errors.New("invalid TOTP code")
	}

	// Generate backup codes
	backupCodes, hashedCodes, err := GenerateBackupCodes(10)
	if err != nil {
		return nil, fmt.Errorf("failed to generate backup codes: %w", err)
	}

	// Encrypt the TOTP secret before storing (encryption is required)
	if s.encryptionKey == "" {
		return nil, errors.New("TOTP encryption key not configured - cannot store TOTP secrets securely")
	}
	encryptedSecret, err := crypto.Encrypt(secret, s.encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt TOTP secret: %w", err)
	}
	secretToStore := encryptedSecret

	// Enable TOTP for the user
	updateQuery := `
		UPDATE auth.users
		SET totp_secret = $1, totp_enabled = TRUE, backup_codes = $2, updated_at = NOW()
		WHERE id = $3
	`

	_, err = s.userRepo.db.Pool().Exec(ctx, updateQuery, secretToStore, hashedCodes, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to enable TOTP: %w", err)
	}

	// Mark setup as verified
	_, _ = s.userRepo.db.Pool().Exec(ctx, `
		UPDATE auth.two_factor_setups
		SET verified = TRUE
		WHERE user_id = $1
	`, userID)

	return backupCodes, nil
}

// VerifyTOTP verifies a TOTP code during login
func (s *Service) VerifyTOTP(ctx context.Context, userID, code string) error {
	return s.VerifyTOTPWithContext(ctx, userID, code, "", "")
}

// VerifyTOTPWithContext verifies a TOTP code with IP address and user agent for rate limiting
func (s *Service) VerifyTOTPWithContext(ctx context.Context, userID, code, ipAddress, userAgent string) error {
	// Check rate limit before attempting verification
	if s.totpRateLimiter != nil {
		if err := s.totpRateLimiter.CheckRateLimit(ctx, userID); err != nil {
			return err
		}
	}

	// Fetch user's TOTP secret and backup codes
	var storedSecret string
	var backupCodes []string
	query := `
		SELECT totp_secret, COALESCE(backup_codes, ARRAY[]::text[])
		FROM auth.users
		WHERE id = $1 AND totp_enabled = TRUE
	`

	err := s.userRepo.db.Pool().QueryRow(ctx, query, userID).Scan(&storedSecret, &backupCodes)
	if err != nil {
		return fmt.Errorf("2FA not enabled for this user: %w", err)
	}

	// Decrypt the TOTP secret
	secret := storedSecret
	if s.encryptionKey == "" {
		// This should not happen in production - encryption key should always be set
		log.Warn().Str("user_id", userID).Msg("TOTP encryption key not configured - TOTP secrets may be stored insecurely")
	} else {
		decrypted, err := crypto.Decrypt(storedSecret, s.encryptionKey)
		if err != nil {
			// Log but don't fail - might be a legacy unencrypted secret
			log.Warn().Err(err).Str("user_id", userID).Msg("Failed to decrypt TOTP secret, trying as plaintext (legacy secret)")
		} else {
			secret = decrypted
		}
	}

	// Try TOTP code first
	valid, err := VerifyTOTPCode(code, secret)
	if err == nil && valid {
		// Record successful attempt (clears rate limit counter effectively)
		if s.totpRateLimiter != nil {
			_ = s.totpRateLimiter.RecordAttempt(ctx, userID, true, ipAddress, userAgent)
		}
		return nil
	}

	// Try backup codes
	for i, hashedCode := range backupCodes {
		match, err := VerifyBackupCode(code, hashedCode)
		if err == nil && match {
			// Remove used backup code
			newBackupCodes := append(backupCodes[:i], backupCodes[i+1:]...)

			_, err = s.userRepo.db.Pool().Exec(ctx, `
				UPDATE auth.users
				SET backup_codes = $1, updated_at = NOW()
				WHERE id = $2
			`, newBackupCodes, userID)

			if err != nil {
				return fmt.Errorf("failed to update backup codes: %w", err)
			}

			// Log successful recovery code usage
			_, _ = s.userRepo.db.Pool().Exec(ctx, `
				INSERT INTO auth.two_factor_recovery_attempts (user_id, code_used, success)
				VALUES ($1, $2, TRUE)
			`, userID, "backup_code")

			// Record successful attempt
			if s.totpRateLimiter != nil {
				_ = s.totpRateLimiter.RecordAttempt(ctx, userID, true, ipAddress, userAgent)
			}

			return nil
		}
	}

	// Record failed attempt for rate limiting
	if s.totpRateLimiter != nil {
		_ = s.totpRateLimiter.RecordAttempt(ctx, userID, false, ipAddress, userAgent)
	} else {
		// Fallback: Log failed attempt directly if no rate limiter configured
		_, _ = s.userRepo.db.Pool().Exec(ctx, `
			INSERT INTO auth.two_factor_recovery_attempts (user_id, code_used, success)
			VALUES ($1, $2, FALSE)
		`, userID, "totp_code")
	}

	return errors.New("invalid 2FA code")
}

// DisableTOTP disables 2FA for a user
func (s *Service) DisableTOTP(ctx context.Context, userID, password string) error {
	// Verify password first
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	if user.PasswordHash != "" {
		err := s.passwordHasher.ComparePassword(user.PasswordHash, password)
		if err != nil {
			return errors.New("invalid password")
		}
	}

	// Disable TOTP
	query := `
		UPDATE auth.users
		SET totp_enabled = FALSE, totp_secret = NULL, backup_codes = NULL, updated_at = NOW()
		WHERE id = $1
	`

	_, err = s.userRepo.db.Pool().Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to disable 2FA: %w", err)
	}

	// Clean up pending setups
	_, _ = s.userRepo.db.Pool().Exec(ctx, `
		DELETE FROM auth.two_factor_setups WHERE user_id = $1
	`, userID)

	return nil
}

// IsTOTPEnabled checks if 2FA is enabled for a user
func (s *Service) IsTOTPEnabled(ctx context.Context, userID string) (bool, error) {
	var enabled bool
	query := `SELECT COALESCE(totp_enabled, FALSE) FROM auth.users WHERE id = $1`

	err := s.userRepo.db.Pool().QueryRow(ctx, query, userID).Scan(&enabled)
	if err != nil {
		return false, fmt.Errorf("failed to check 2FA status: %w", err)
	}

	return enabled, nil
}

// GenerateTokensForUser generates JWT tokens for a user after successful 2FA verification
func (s *Service) GenerateTokensForUser(ctx context.Context, userID string) (*SignInResponse, error) {
	// Get user details
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// Generate JWT token pair with metadata
	accessToken, refreshToken, _, err := s.jwtManager.GenerateTokenPair(user.ID, user.Email, user.Role, user.UserMetadata, user.AppMetadata)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	// Create session
	expiresAt := time.Now().Add(s.config.RefreshExpiry)
	_, err = s.sessionRepo.Create(ctx, user.ID, accessToken, refreshToken, expiresAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return &SignInResponse{
		User:         user,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(s.config.JWTExpiry.Seconds()),
	}, nil
}

// Reauthenticate generates a security nonce for sensitive operations.
// The nonce is stored with a 5-minute TTL and can only be used once.
func (s *Service) Reauthenticate(ctx context.Context, userID string) (string, error) {
	// Verify user exists
	_, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("user not found: %w", err)
	}

	// Generate a secure random nonce
	nonce := uuid.New().String()

	// Store nonce with 5-minute expiry for later verification (distributed across instances)
	if err := s.nonceRepo.Set(ctx, nonce, userID, 5*time.Minute); err != nil {
		return "", fmt.Errorf("failed to store nonce: %w", err)
	}

	return nonce, nil
}

// VerifyNonce validates a nonce for sensitive operations.
// The nonce is single-use and will be invalidated after verification.
// Returns true if the nonce is valid and belongs to the specified user.
func (s *Service) VerifyNonce(ctx context.Context, nonce, userID string) bool {
	valid, err := s.nonceRepo.Validate(ctx, nonce, userID)
	if err != nil {
		// Log error but return false to maintain backward compatibility
		return false
	}
	return valid
}

// CleanupExpiredNonces removes expired nonces from the database.
// This is optional maintenance - nonces are single-use and deleted on validation.
// Expired but unused nonces will simply fail validation and can be cleaned up periodically.
func (s *Service) CleanupExpiredNonces(ctx context.Context) (int64, error) {
	return s.nonceRepo.Cleanup(ctx)
}

// SignInWithIDToken signs in a user with an OAuth ID token (Google, Apple, Microsoft, or custom OIDC)
func (s *Service) SignInWithIDToken(ctx context.Context, provider, idToken, nonce string) (*SignInResponse, error) {
	// Check if the provider is configured
	if !s.oidcVerifier.IsProviderConfigured(provider) {
		return nil, fmt.Errorf("OIDC provider not configured: %s", provider)
	}

	// Verify the ID token and extract claims
	claims, err := s.oidcVerifier.Verify(ctx, provider, idToken, nonce)
	if err != nil {
		return nil, fmt.Errorf("invalid ID token: %w", err)
	}

	// Require email for user lookup/creation
	if claims.Email == "" {
		return nil, fmt.Errorf("ID token does not contain email claim")
	}

	// Look up existing user by email
	user, err := s.userRepo.GetByEmail(ctx, claims.Email)
	if err != nil && !errors.Is(err, ErrUserNotFound) {
		return nil, fmt.Errorf("failed to lookup user: %w", err)
	}

	if user == nil {
		// Create new user from OIDC claims
		user, err = s.createOIDCUser(ctx, provider, claims)
		if err != nil {
			return nil, fmt.Errorf("failed to create user: %w", err)
		}
	} else {
		// Update user info from OIDC claims if changed
		if err := s.updateUserFromOIDCClaims(ctx, user, claims); err != nil {
			// Log but don't fail the sign-in
			fmt.Printf("warning: failed to update user from OIDC claims: %v\n", err)
		}
	}

	// Generate JWT tokens
	accessToken, refreshToken, _, err := s.jwtManager.GenerateTokenPair(
		user.ID,
		user.Email,
		user.Role,
		user.UserMetadata,
		user.AppMetadata,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	return &SignInResponse{
		User:         user,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(s.config.JWTExpiry.Seconds()),
	}, nil
}

// createOIDCUser creates a new user from OIDC claims
func (s *Service) createOIDCUser(ctx context.Context, provider string, claims *IDTokenClaims) (*User, error) {
	req := CreateUserRequest{
		Email:    claims.Email,
		Password: "", // No password for OIDC users
		Role:     "authenticated",
		UserMetadata: map[string]interface{}{
			"name":    claims.Name,
			"picture": claims.Picture,
		},
		AppMetadata: map[string]interface{}{
			"provider":         provider,
			"provider_user_id": claims.Subject,
		},
	}

	// Create user with empty password hash (OIDC users don't have passwords)
	user, err := s.userRepo.Create(ctx, req, "")
	if err != nil {
		return nil, err
	}

	// Update email_verified if the OIDC provider verified it
	if claims.EmailVerified {
		emailVerified := true
		_, err = s.userRepo.Update(ctx, user.ID, UpdateUserRequest{
			EmailVerified: &emailVerified,
		})
		if err != nil {
			// Log but don't fail - user was created
			fmt.Printf("warning: failed to update email_verified: %v\n", err)
		}
		user.EmailVerified = true
	}

	return user, nil
}

// updateUserFromOIDCClaims updates user info from OIDC claims if changed
func (s *Service) updateUserFromOIDCClaims(ctx context.Context, user *User, claims *IDTokenClaims) error {
	updateReq := UpdateUserRequest{}
	needsUpdate := false

	// Update email verification status if changed
	if claims.EmailVerified && !user.EmailVerified {
		emailVerified := true
		updateReq.EmailVerified = &emailVerified
		needsUpdate = true
	}

	// Update user metadata if name or picture changed
	currentMetadata, _ := user.UserMetadata.(map[string]interface{})
	if currentMetadata == nil {
		currentMetadata = make(map[string]interface{})
	}

	newMetadata := make(map[string]interface{})
	for k, v := range currentMetadata {
		newMetadata[k] = v
	}

	if claims.Name != "" {
		if currentName, _ := currentMetadata["name"].(string); currentName != claims.Name {
			newMetadata["name"] = claims.Name
			needsUpdate = true
		}
	}

	if claims.Picture != "" {
		if currentPic, _ := currentMetadata["picture"].(string); currentPic != claims.Picture {
			newMetadata["picture"] = claims.Picture
			needsUpdate = true
		}
	}

	if needsUpdate {
		updateReq.UserMetadata = newMetadata
		_, err := s.userRepo.Update(ctx, user.ID, updateReq)
		return err
	}

	return nil
}

// SendOTP sends an OTP code via email
func (s *Service) SendOTP(ctx context.Context, email, purpose string) error {
	return s.otpService.SendEmailOTP(ctx, email, purpose)
}

// VerifyOTP verifies an OTP code sent via email
func (s *Service) VerifyOTP(ctx context.Context, email, code string) (*OTPCode, error) {
	return s.otpService.VerifyEmailOTP(ctx, email, code)
}

// ResendOTP resends an OTP code to an email
func (s *Service) ResendOTP(ctx context.Context, email, purpose string) error {
	return s.otpService.ResendEmailOTP(ctx, email, purpose)
}

// GetUserIdentities retrieves all OAuth identities linked to a user
func (s *Service) GetUserIdentities(ctx context.Context, userID string) ([]UserIdentity, error) {
	return s.identityService.GetUserIdentities(ctx, userID)
}

// LinkIdentity initiates OAuth flow to link a new provider
func (s *Service) LinkIdentity(ctx context.Context, userID, provider string) (string, string, error) {
	return s.identityService.LinkIdentityProvider(ctx, userID, provider)
}

// UnlinkIdentity removes an OAuth identity from a user
func (s *Service) UnlinkIdentity(ctx context.Context, userID, identityID string) error {
	return s.identityService.UnlinkIdentity(ctx, userID, identityID)
}

// GetUserByEmail retrieves a user by email
func (s *Service) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	return s.userRepo.GetByEmail(ctx, email)
}

// CreateUser creates a new user with email and optional password
func (s *Service) CreateUser(ctx context.Context, email, password string) (*User, error) {
	// Validate email format and length
	if err := ValidateEmail(email); err != nil {
		return nil, fmt.Errorf("invalid email: %w", err)
	}

	// If password is empty, create user without password (for OTP/OAuth flows)
	hashedPassword := ""
	if password != "" {
		hash, err := s.passwordHasher.HashPassword(password)
		if err != nil {
			return nil, fmt.Errorf("failed to hash password: %w", err)
		}
		hashedPassword = hash
	}

	req := CreateUserRequest{
		Email:    email,
		Password: password,
		Role:     "user",
	}
	return s.userRepo.Create(ctx, req, hashedPassword)
}

// IsEmailVerificationRequired checks if email verification is required based on settings and email configuration
func (s *Service) IsEmailVerificationRequired(ctx context.Context) bool {
	// Check if the setting is enabled
	required := s.settingsCache.GetBool(ctx, "app.auth.require_email_verification", false)
	if !required {
		return false
	}

	// Also check if email is configured - can't require verification without email
	if s.emailService == nil {
		return false
	}
	return s.emailService.IsConfigured()
}

// SendEmailVerification sends a verification email to the user
func (s *Service) SendEmailVerification(ctx context.Context, userID, email string) error {
	if s.emailService == nil || !s.emailService.IsConfigured() {
		return fmt.Errorf("email service is not configured")
	}

	// Delete any existing tokens for this user
	_ = s.emailVerificationRepo.DeleteByUserID(ctx, userID)

	// Create new verification token
	tokenWithPlaintext, err := s.emailVerificationRepo.Create(ctx, userID, s.emailVerificationExpiry)
	if err != nil {
		return fmt.Errorf("failed to create verification token: %w", err)
	}

	// Build verification link
	link := fmt.Sprintf("%s/auth/verify-email?token=%s", s.baseURL, tokenWithPlaintext.PlaintextToken)

	// Send verification email
	if err := s.emailService.SendVerificationEmail(ctx, email, tokenWithPlaintext.PlaintextToken, link); err != nil {
		return fmt.Errorf("failed to send verification email: %w", err)
	}

	return nil
}

// VerifyEmailToken validates the verification token and marks the user's email as verified
func (s *Service) VerifyEmailToken(ctx context.Context, token string) (*User, error) {
	// Validate the token
	emailToken, err := s.emailVerificationRepo.Validate(ctx, token)
	if err != nil {
		return nil, err
	}

	// Mark token as used
	if err := s.emailVerificationRepo.MarkAsUsed(ctx, emailToken.ID); err != nil {
		return nil, fmt.Errorf("failed to mark token as used: %w", err)
	}

	// Mark user's email as verified
	if err := s.userRepo.VerifyEmail(ctx, emailToken.UserID); err != nil {
		return nil, fmt.Errorf("failed to verify email: %w", err)
	}

	// Get updated user
	user, err := s.userRepo.GetByID(ctx, emailToken.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

// =============================================================================
// SAML SSO Methods
// =============================================================================

// CreateSAMLUser creates a new user from a SAML assertion
func (s *Service) CreateSAMLUser(ctx context.Context, email, name, provider, nameID string, attrs map[string][]string) (*User, error) {
	// Validate email format
	if err := ValidateEmail(email); err != nil {
		return nil, fmt.Errorf("invalid email: %w", err)
	}

	// Build user metadata with name if provided
	userMetadata := make(map[string]interface{})
	if name != "" {
		userMetadata["full_name"] = name
	}

	// Create user without password (SAML users authenticate via IdP)
	req := CreateUserRequest{
		Email:        email,
		Password:     "",
		Role:         "authenticated",
		UserMetadata: userMetadata,
	}

	user, err := s.userRepo.Create(ctx, req, "")
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Link SAML identity using identity service
	if err := s.LinkSAMLIdentity(ctx, user.ID, provider, nameID, attrs); err != nil {
		// Log warning but don't fail - user was created successfully
		_ = err // Ignore error, user is still valid
	}

	// Refresh user to get updated data
	user, err = s.userRepo.GetByID(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get created user: %w", err)
	}

	return user, nil
}

// LinkSAMLIdentity links or updates a SAML identity for a user
func (s *Service) LinkSAMLIdentity(ctx context.Context, userID, provider, nameID string, attrs map[string][]string) error {
	// Create identity data that includes SAML-specific fields
	identityData := map[string]interface{}{
		"saml_name_id":    nameID,
		"saml_attributes": attrs,
	}

	// Extract email from SAML attributes if present
	var email *string
	if emails, ok := attrs["email"]; ok && len(emails) > 0 {
		email = &emails[0]
	}

	// Use the identity service to link the SAML identity
	// Provider format: "saml:{provider_name}"
	_, err := s.identityService.LinkIdentity(ctx, userID, "saml:"+provider, nameID, email, identityData)
	return err
}

// GenerateTokensForSAMLUser generates tokens for a SAML-authenticated user
// This is a wrapper around GenerateTokensForUser that takes a User object
func (s *Service) GenerateTokensForSAMLUser(ctx context.Context, user *User) (*SignInResponse, error) {
	return s.GenerateTokensForUser(ctx, user.ID)
}
