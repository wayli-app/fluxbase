package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/config"
	"github.com/nimbleflux/fluxbase/internal/database"
	"github.com/nimbleflux/fluxbase/internal/observability"
)

// Service provides a high-level authentication API
type Service struct {
	userRepo                 *UserRepository
	sessionRepo              *SessionRepository
	magicLinkRepo            *MagicLinkRepository
	emailVerificationRepo    *EmailVerificationRepository
	jwtManager               *JWTManager
	passwordHasher           *PasswordHasher
	oauthManager             *OAuthManager
	magicLinkService         *MagicLinkService
	passwordResetService     *PasswordResetService
	tokenBlacklistService    *TokenBlacklistService
	impersonationService     *ImpersonationService
	otpService               *OTPService
	identityService          *IdentityService
	systemSettings           *SystemSettingsService
	settingsCache            *SettingsCache
	nonceRepo                *NonceRepository
	oidcVerifier             *OIDCVerifier
	config                   *config.AuthConfig
	emailService             EmailService
	baseURL                  string
	emailVerificationExpiry  time.Duration
	metrics                  *observability.Metrics
	mfaService               *MFAService
	nonceService             *NonceService
	emailVerificationService *EmailVerificationService
}

// SetEncryptionKey sets the encryption key for encrypting sensitive data at rest
func (s *Service) SetEncryptionKey(key string) {
	if s.mfaService != nil {
		s.mfaService.SetEncryptionKey(key)
	}
}

// SetTOTPRateLimiter sets the TOTP rate limiter for protecting against brute force attacks
func (s *Service) SetTOTPRateLimiter(limiter *TOTPRateLimiter) {
	if s.mfaService != nil {
		s.mfaService.SetTOTPRateLimiter(limiter)
	}
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

	jwtManager, err := NewJWTManagerWithConfig(cfg.JWTSecret, cfg.JWTExpiry, cfg.RefreshExpiry, cfg.ServiceRoleTTL, cfg.AnonTTL)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create JWT manager")
	}
	passwordHasher := NewPasswordHasherWithConfig(PasswordHasherConfig{MinLength: cfg.PasswordMinLen, Cost: cfg.BcryptCost})
	oauthManager := NewOAuthManager()

	emailSvc, _ := emailService.(EmailService)

	// Use configured expiry times with sensible fallbacks
	magicLinkExpiry := cfg.MagicLinkExpiry
	if magicLinkExpiry == 0 {
		magicLinkExpiry = 15 * time.Minute
	}

	magicLinkService := NewMagicLinkService(
		magicLinkRepo,
		userRepo,
		emailSvc,
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
		emailSvc,
		passwordResetExpiry,
		baseURL,
	)

	tokenBlacklistRepo := NewTokenBlacklistRepository(db)
	tokenBlacklistService := NewTokenBlacklistService(tokenBlacklistRepo, jwtManager)

	impersonationRepo := NewImpersonationRepository(db)
	impersonationService := NewImpersonationService(impersonationRepo, userRepo, jwtManager, db)
	impersonationService.SetTokenBlacklistService(tokenBlacklistService)

	// OTP service for passwordless authentication
	otpExpiry := cfg.MagicLinkExpiry // Reuse magic link expiry for OTP (typically 10-15 minutes)
	if otpExpiry == 0 {
		otpExpiry = 10 * time.Minute
	}
	otpRepo := NewOTPRepository(db)
	// Create OTP sender that uses the email service
	// If email service doesn't support Send method, use NoOpOTPSender
	var otpSender OTPSender
	if emailSvc != nil {
		otpSender = NewDefaultOTPSender(emailSvc, "", "")
	} else {
		otpSender = &NoOpOTPSender{}
	}
	otpService := NewOTPService(otpRepo, userRepo, otpSender, otpExpiry)

	// Identity linking service
	// SECURITY: Use database-backed state store for multi-instance deployments
	// In-memory state store fails in multi-instance setups (OAuth callback may hit different instance)
	var stateStore StateStorer
	if cfg.OAuthStateStorage == "database" {
		log.Info().Msg("Using database-backed OAuth state storage for multi-instance deployments")
		stateStore = NewDBStateStore(db, DefaultDBStateStoreConfig())
	} else {
		if cfg.OAuthStateStorage != "" && cfg.OAuthStateStorage != "memory" {
			log.Warn().Str("storage", cfg.OAuthStateStorage).Msg("Unknown oauth_state_storage value, using default (memory)")
		}
		stateStore = NewStateStore()
	}
	identityRepo := NewIdentityRepository(db)
	identityService := NewIdentityService(identityRepo, oauthManager, stateStore)

	systemSettingsService := NewSystemSettingsService(db)
	settingsCache := NewSettingsCache(systemSettingsService, 30*time.Second)

	// Wire up cache to settings service for cache invalidation on updates
	systemSettingsService.SetCache(settingsCache)

	// Create nonce repository for distributed reauthentication
	nonceRepo := NewNonceRepository(db)
	nonceService := NewNonceService(nonceRepo, userRepo)

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
	emailVerificationService := NewEmailVerificationService(
		emailVerificationRepo,
		userRepo,
		settingsCache,
		emailSvc,
		baseURL,
		emailVerificationExpiry,
	)

	// Create MFA service
	mfaService := NewMFAService(userRepo, sessionRepo, jwtManager, passwordHasher, db, cfg)

	return &Service{
		userRepo:                 userRepo,
		sessionRepo:              sessionRepo,
		magicLinkRepo:            magicLinkRepo,
		emailVerificationRepo:    emailVerificationRepo,
		jwtManager:               jwtManager,
		passwordHasher:           passwordHasher,
		oauthManager:             oauthManager,
		magicLinkService:         magicLinkService,
		passwordResetService:     passwordResetService,
		tokenBlacklistService:    tokenBlacklistService,
		impersonationService:     impersonationService,
		otpService:               otpService,
		identityService:          identityService,
		systemSettings:           systemSettingsService,
		settingsCache:            settingsCache,
		nonceRepo:                nonceRepo,
		oidcVerifier:             oidcVerifier,
		config:                   cfg,
		emailService:             emailSvc,
		baseURL:                  baseURL,
		emailVerificationExpiry:  emailVerificationExpiry,
		mfaService:               mfaService,
		nonceService:             nonceService,
		emailVerificationService: emailVerificationService,
	}
}

// SignUpRequest represents a user registration request
type SignUpRequest struct {
	Email             string                 `json:"email"`
	Password          string                 `json:"password"`
	UserMetadata      map[string]interface{} `json:"user_metadata,omitempty"`      // User-editable metadata
	AppMetadata       map[string]interface{} `json:"app_metadata,omitempty"`       // Application/admin-only metadata
	CaptchaToken      string                 `json:"captcha_token,omitempty"`      // CAPTCHA verification token
	ChallengeID       string                 `json:"challenge_id,omitempty"`       // Challenge ID from pre-flight check
	DeviceFingerprint string                 `json:"device_fingerprint,omitempty"` // Optional device fingerprint for trust tracking
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
	Email             string `json:"email"`
	Password          string `json:"password"`
	CaptchaToken      string `json:"captcha_token,omitempty"`      // CAPTCHA verification token
	ChallengeID       string `json:"challenge_id,omitempty"`       // Challenge ID from pre-flight check
	DeviceFingerprint string `json:"device_fingerprint,omitempty"` // Optional device fingerprint for trust tracking
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

	tenantID := database.TenantFromContext(ctx)
	if tenantID != "" && user.TenantID != "" && user.TenantID != tenantID {
		LogSecurityEvent(ctx, SecurityEvent{
			Type:  SecurityEventLoginFailed,
			Email: req.Email,
			Details: map[string]interface{}{
				"reason":         "tenant_mismatch",
				"user_tenant_id": user.TenantID,
				"request_tenant": tenantID,
			},
		})
		return nil, fmt.Errorf("invalid email or password")
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
