package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/wayli-app/fluxbase/internal/config"
	"github.com/wayli-app/fluxbase/internal/database"
)

// Service provides a high-level authentication API
type Service struct {
	userRepo              *UserRepository
	sessionRepo           *SessionRepository
	magicLinkRepo         *MagicLinkRepository
	jwtManager            *JWTManager
	passwordHasher        *PasswordHasher
	oauthManager          *OAuthManager
	magicLinkService      *MagicLinkService
	passwordResetService  *PasswordResetService
	tokenBlacklistService *TokenBlacklistService
	impersonationService  *ImpersonationService
	systemSettings        *SystemSettingsService
	settingsCache         *SettingsCache
	config                *config.AuthConfig
}

// NewService creates a new authentication service
func NewService(
	db *database.Connection,
	cfg *config.AuthConfig,
	emailService EmailSender,
	baseURL string,
) *Service {
	userRepo := NewUserRepository(db)
	sessionRepo := NewSessionRepository(db)
	magicLinkRepo := NewMagicLinkRepository(db)

	jwtManager := NewJWTManager(cfg.JWTSecret, cfg.JWTExpiry, cfg.RefreshExpiry)
	passwordHasher := NewPasswordHasherWithConfig(PasswordHasherConfig{MinLength: cfg.PasswordMinLen, Cost: cfg.BcryptCost})
	oauthManager := NewOAuthManager()

	// Use configured expiry times with sensible fallbacks
	magicLinkExpiry := cfg.MagicLinkExpiry
	if magicLinkExpiry == 0 {
		magicLinkExpiry = 15 * time.Minute
	}

	magicLinkService := NewMagicLinkService(
		magicLinkRepo,
		userRepo,
		emailService,
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
		emailService,
		passwordResetExpiry,
		baseURL,
	)

	tokenBlacklistRepo := NewTokenBlacklistRepository(db)
	tokenBlacklistService := NewTokenBlacklistService(tokenBlacklistRepo, jwtManager)

	impersonationRepo := NewImpersonationRepository(db)
	impersonationService := NewImpersonationService(impersonationRepo, userRepo, jwtManager)

	systemSettingsService := NewSystemSettingsService(db)
	settingsCache := NewSettingsCache(systemSettingsService, 30*time.Second)

	return &Service{
		userRepo:              userRepo,
		sessionRepo:           sessionRepo,
		magicLinkRepo:         magicLinkRepo,
		jwtManager:            jwtManager,
		passwordHasher:        passwordHasher,
		oauthManager:          oauthManager,
		magicLinkService:      magicLinkService,
		passwordResetService:  passwordResetService,
		tokenBlacklistService: tokenBlacklistService,
		impersonationService:  impersonationService,
		systemSettings:        systemSettingsService,
		settingsCache:         settingsCache,
		config:                cfg,
	}
}

// SignUpRequest represents a user registration request
type SignUpRequest struct {
	Email        string                 `json:"email"`
	Password     string                 `json:"password"`
	UserMetadata map[string]interface{} `json:"user_metadata,omitempty"` // User-editable metadata
	AppMetadata  map[string]interface{} `json:"app_metadata,omitempty"`  // Application/admin-only metadata
}

// SignUpResponse represents a successful registration response
type SignUpResponse struct {
	User         *User  `json:"user"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"` // seconds
}

// SignUp registers a new user with email and password
func (s *Service) SignUp(ctx context.Context, req SignUpRequest) (*SignUpResponse, error) {
	// Check if signup is enabled from database settings (with fallback to config)
	enableSignup := s.settingsCache.GetBool(ctx, "app.auth.enable_signup", s.config.EnableSignup)
	if !enableSignup {
		return nil, fmt.Errorf("signup is disabled")
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
	user, err := s.userRepo.Create(ctx, CreateUserRequest{
		Email:        req.Email,
		UserMetadata: req.UserMetadata, // User-editable metadata
		AppMetadata:  req.AppMetadata,  // Application/admin-only metadata
	}, hashedPassword)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
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
	Email    string `json:"email"`
	Password string `json:"password"`
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
			return nil, fmt.Errorf("invalid email or password")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Verify password
	if err := s.passwordHasher.ComparePassword(user.PasswordHash, req.Password); err != nil {
		return nil, fmt.Errorf("invalid email or password")
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

// RefreshToken generates a new access token using a refresh token
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
			return nil, fmt.Errorf("session not found or expired")
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	// Check if session is expired
	if time.Now().After(session.ExpiresAt) {
		return nil, fmt.Errorf("session expired")
	}

	// Generate new access token
	newAccessToken, err := s.jwtManager.RefreshAccessToken(req.RefreshToken)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}

	// Update session with new access token
	if err := s.sessionRepo.UpdateAccessToken(ctx, session.ID, newAccessToken); err != nil {
		return nil, fmt.Errorf("failed to update session: %w", err)
	}

	return &RefreshTokenResponse{
		AccessToken:  newAccessToken,
		RefreshToken: req.RefreshToken, // Refresh token stays the same
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
	return s.userRepo.Update(ctx, userID, req)
}

// SendMagicLink sends a magic link to the specified email
func (s *Service) SendMagicLink(ctx context.Context, email string) error {
	// Check if magic link is enabled from database settings (with fallback to config)
	enableMagicLink := s.settingsCache.GetBool(ctx, "app.auth.enable_magic_link", s.config.EnableMagicLink)
	if !enableMagicLink {
		return fmt.Errorf("magic link authentication is disabled")
	}

	return s.magicLinkService.SendMagicLink(ctx, email)
}

// VerifyMagicLink verifies a magic link and returns tokens
func (s *Service) VerifyMagicLink(ctx context.Context, token string) (*SignInResponse, error) {
	// Check if magic link is enabled from database settings (with fallback to config)
	enableMagicLink := s.settingsCache.GetBool(ctx, "app.auth.enable_magic_link", s.config.EnableMagicLink)
	if !enableMagicLink {
		return nil, fmt.Errorf("magic link authentication is disabled")
	}

	// Verify the magic link
	email, err := s.magicLinkService.VerifyMagicLink(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("failed to verify magic link: %w", err)
	}

	// Get or create user
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			// Create new user for magic link (no metadata by default)
			user, err = s.userRepo.Create(ctx, CreateUserRequest{
				Email: email,
			}, "") // No password for magic link users
			if err != nil {
				return nil, fmt.Errorf("failed to create user: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to get user: %w", err)
		}
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

// GetOAuthManager returns the OAuth manager for configuring providers
func (s *Service) GetOAuthManager() *OAuthManager {
	return s.oauthManager
}

// RequestPasswordReset sends a password reset email
func (s *Service) RequestPasswordReset(ctx context.Context, email string) error {
	return s.passwordResetService.RequestPasswordReset(ctx, email)
}

// ResetPassword resets a user's password using a valid reset token
func (s *Service) ResetPassword(ctx context.Context, token, newPassword string) error {
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
	return s.settingsCache.GetBool(ctx, "app.auth.enable_signup", s.config.EnableSignup)
}

// SetupTOTP generates a new TOTP secret for 2FA setup
func (s *Service) SetupTOTP(ctx context.Context, userID string) (string, string, error) {
	// Generate TOTP secret
	secret, qrCodeURL, err := GenerateTOTPSecret("Fluxbase", userID)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate TOTP secret: %w", err)
	}

	// Store the secret in a temporary setup table (expires in 10 minutes)
	query := `
		INSERT INTO auth.two_factor_setups (user_id, secret, qr_code_url, expires_at)
		VALUES ($1, $2, $3, NOW() + INTERVAL '10 minutes')
		ON CONFLICT (user_id) DO UPDATE
			SET secret = EXCLUDED.secret,
			    qr_code_url = EXCLUDED.qr_code_url,
			    expires_at = EXCLUDED.expires_at,
			    verified = FALSE
	`

	_, err = s.userRepo.db.Pool().Exec(ctx, query, userID, secret, qrCodeURL)
	if err != nil {
		return "", "", fmt.Errorf("failed to store TOTP setup: %w", err)
	}

	return secret, qrCodeURL, nil
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

	// Enable TOTP for the user
	updateQuery := `
		UPDATE auth.users
		SET totp_secret = $1, totp_enabled = TRUE, backup_codes = $2, updated_at = NOW()
		WHERE id = $3
	`

	_, err = s.userRepo.db.Pool().Exec(ctx, updateQuery, secret, hashedCodes, userID)
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
	// Fetch user's TOTP secret and backup codes
	var secret string
	var backupCodes []string
	query := `
		SELECT totp_secret, COALESCE(backup_codes, ARRAY[]::text[])
		FROM auth.users
		WHERE id = $1 AND totp_enabled = TRUE
	`

	err := s.userRepo.db.Pool().QueryRow(ctx, query, userID).Scan(&secret, &backupCodes)
	if err != nil {
		return fmt.Errorf("2FA not enabled for this user: %w", err)
	}

	// Try TOTP code first
	valid, err := VerifyTOTPCode(code, secret)
	if err == nil && valid {
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

			return nil
		}
	}

	// Log failed attempt
	_, _ = s.userRepo.db.Pool().Exec(ctx, `
		INSERT INTO auth.two_factor_recovery_attempts (user_id, code_used, success)
		VALUES ($1, $2, FALSE)
	`, userID, code)

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
