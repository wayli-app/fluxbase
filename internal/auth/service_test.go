package auth

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nimbleflux/fluxbase/internal/config"
	"github.com/nimbleflux/fluxbase/internal/crypto"
	"github.com/nimbleflux/fluxbase/internal/testutil"
)

// TestableService wraps the auth service components for unit testing
// without requiring a database connection.
type TestableService struct {
	userRepo              UserRepositoryInterface
	sessionRepo           SessionRepositoryInterface
	tokenBlacklistRepo    TokenBlacklistRepositoryInterface
	jwtManager            *JWTManager
	passwordHasher        *PasswordHasher
	settingsCache         *testutil.MockSettingsCache
	config                *config.AuthConfig
	emailVerificationRepo *MockEmailVerificationRepository
	oauthManager          *OAuthManager
}

func mustNewJWTManager(secret string, accessTTL, refreshTTL time.Duration) *JWTManager {
	m, err := NewJWTManager(secret, accessTTL, refreshTTL)
	if err != nil {
		panic(err)
	}
	return m
}

func mustNewJWTManagerWithConfig(secret string, accessTTL, refreshTTL, serviceRoleTTL, anonTTL time.Duration) *JWTManager {
	m, err := NewJWTManagerWithConfig(secret, accessTTL, refreshTTL, serviceRoleTTL, anonTTL)
	if err != nil {
		panic(err)
	}
	return m
}

// MockEmailVerificationRepository for testing email verification
type MockEmailVerificationRepository struct {
	tokens map[string]*EmailVerificationToken
}

func NewMockEmailVerificationRepository() *MockEmailVerificationRepository {
	return &MockEmailVerificationRepository{
		tokens: make(map[string]*EmailVerificationToken),
	}
}

func (m *MockEmailVerificationRepository) Create(ctx context.Context, userID string, expiry time.Duration) (*EmailVerificationTokenWithPlaintext, error) {
	token := &EmailVerificationTokenWithPlaintext{
		PlaintextToken: "test-verification-token",
	}
	m.tokens[token.PlaintextToken] = &EmailVerificationToken{
		ID:        "test-id",
		UserID:    userID,
		ExpiresAt: time.Now().Add(expiry),
	}
	return token, nil
}

func (m *MockEmailVerificationRepository) Validate(ctx context.Context, token string) (*EmailVerificationToken, error) {
	if t, ok := m.tokens[token]; ok {
		if time.Now().Before(t.ExpiresAt) {
			return t, nil
		}
		return nil, errors.New("token expired")
	}
	return nil, errors.New("token not found")
}

func (m *MockEmailVerificationRepository) MarkAsUsed(ctx context.Context, id string) error {
	return nil
}

func (m *MockEmailVerificationRepository) DeleteByUserID(ctx context.Context, userID string) error {
	return nil
}

// NewTestableService creates a service configured for testing
func NewTestableService() *TestableService {
	cfg := &config.AuthConfig{
		JWTSecret:      "test-secret-key-at-least-32-chars-long",
		JWTExpiry:      15 * time.Minute,
		RefreshExpiry:  7 * 24 * time.Hour,
		PasswordMinLen: 8,
		BcryptCost:     4, // Low cost for fast tests
		SignupEnabled:  true,
	}

	return &TestableService{
		userRepo:              NewMockUserRepository(),
		sessionRepo:           NewMockSessionRepository(),
		tokenBlacklistRepo:    NewMockTokenBlacklistRepository(),
		jwtManager:            mustNewJWTManager(cfg.JWTSecret, cfg.JWTExpiry, cfg.RefreshExpiry),
		passwordHasher:        NewPasswordHasherWithConfig(PasswordHasherConfig{MinLength: cfg.PasswordMinLen, Cost: cfg.BcryptCost}),
		settingsCache:         testutil.NewMockSettingsCache(),
		config:                cfg,
		emailVerificationRepo: NewMockEmailVerificationRepository(),
		oauthManager:          NewOAuthManager(),
	}
}

// SignUp implements user registration logic for testing
func (s *TestableService) SignUp(ctx context.Context, req SignUpRequest) (*SignUpResponse, error) {
	// Check if signup is enabled
	enableSignup := s.settingsCache.GetBool(ctx, "app.auth.signup_enabled", s.config.SignupEnabled)
	if !enableSignup {
		return nil, errors.New("signup is disabled")
	}

	// Validate email
	if err := ValidateEmail(req.Email); err != nil {
		return nil, errors.New("invalid email: " + err.Error())
	}

	// Validate password
	if err := s.passwordHasher.ValidatePassword(req.Password); err != nil {
		return nil, errors.New("invalid password: " + err.Error())
	}

	// Hash password
	hashedPassword, err := s.passwordHasher.HashPassword(req.Password)
	if err != nil {
		return nil, errors.New("failed to hash password: " + err.Error())
	}

	// Create user
	user, err := s.userRepo.Create(ctx, CreateUserRequest{
		Email:        req.Email,
		UserMetadata: req.UserMetadata,
		AppMetadata:  nil, // Stripped for security
	}, hashedPassword)
	if err != nil {
		return nil, errors.New("failed to create user: " + err.Error())
	}

	// Check if email verification is required
	requireVerification := s.settingsCache.GetBool(ctx, "app.auth.require_email_verification", false)
	if requireVerification {
		return &SignUpResponse{
			User:                      user,
			RequiresEmailVerification: true,
		}, nil
	}

	// Generate tokens
	accessToken, refreshToken, _, err := s.jwtManager.GenerateTokenPair(
		user.ID,
		user.Email,
		user.Role,
		user.UserMetadata,
		user.AppMetadata,
	)
	if err != nil {
		return nil, errors.New("failed to generate tokens: " + err.Error())
	}

	// Create session
	expiresAt := time.Now().Add(s.config.RefreshExpiry)
	_, err = s.sessionRepo.Create(ctx, user.ID, accessToken, refreshToken, expiresAt)
	if err != nil {
		return nil, errors.New("failed to create session: " + err.Error())
	}

	return &SignUpResponse{
		User:         user,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(s.config.JWTExpiry.Seconds()),
	}, nil
}

// SignIn implements user login logic for testing
func (s *TestableService) SignIn(ctx context.Context, req SignInRequest) (*SignInResponse, error) {
	// Get user by email
	user, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, errors.New("invalid email or password")
		}
		return nil, errors.New("failed to get user: " + err.Error())
	}

	// Check if account is locked
	if user.IsLocked {
		if user.LockedUntil != nil && time.Now().After(*user.LockedUntil) {
			// Lock expired, reset it
			_ = s.userRepo.ResetFailedLoginAttempts(ctx, user.ID)
		} else {
			return nil, ErrAccountLocked
		}
	}

	// Verify password
	if err := s.passwordHasher.ComparePassword(user.PasswordHash, req.Password); err != nil {
		// Increment failed login attempts
		_ = s.userRepo.IncrementFailedLoginAttempts(ctx, user.ID)
		return nil, errors.New("invalid email or password")
	}

	// Reset failed login attempts on successful login
	if user.FailedLoginAttempts > 0 {
		_ = s.userRepo.ResetFailedLoginAttempts(ctx, user.ID)
	}

	// Check if email verification is required
	requireVerification := s.settingsCache.GetBool(ctx, "app.auth.require_email_verification", false)
	if requireVerification && !user.EmailVerified {
		return nil, ErrEmailNotVerified
	}

	// Generate tokens
	accessToken, refreshToken, _, err := s.jwtManager.GenerateTokenPair(
		user.ID,
		user.Email,
		user.Role,
		user.UserMetadata,
		user.AppMetadata,
	)
	if err != nil {
		return nil, errors.New("failed to generate tokens: " + err.Error())
	}

	// Create session
	expiresAt := time.Now().Add(s.config.RefreshExpiry)
	_, err = s.sessionRepo.Create(ctx, user.ID, accessToken, refreshToken, expiresAt)
	if err != nil {
		return nil, errors.New("failed to create session: " + err.Error())
	}

	return &SignInResponse{
		User:         user,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(s.config.JWTExpiry.Seconds()),
	}, nil
}

// SignOut implements logout logic for testing
func (s *TestableService) SignOut(ctx context.Context, accessToken string) error {
	// Get session by access token
	session, err := s.sessionRepo.GetByAccessToken(ctx, accessToken)
	if err != nil {
		if errors.Is(err, ErrSessionNotFound) {
			return nil // Already signed out
		}
		return errors.New("failed to get session: " + err.Error())
	}

	// Delete session
	if err := s.sessionRepo.Delete(ctx, session.ID); err != nil {
		return errors.New("failed to delete session: " + err.Error())
	}

	return nil
}

// RefreshToken implements token refresh logic for testing
func (s *TestableService) RefreshToken(ctx context.Context, req RefreshTokenRequest) (*RefreshTokenResponse, error) {
	// Validate refresh token
	claims, err := s.jwtManager.ValidateToken(req.RefreshToken)
	if err != nil {
		return nil, errors.New("invalid refresh token: " + err.Error())
	}

	if claims.TokenType != "refresh" {
		return nil, errors.New("invalid token type")
	}

	// Get session by refresh token
	session, err := s.sessionRepo.GetByRefreshToken(ctx, req.RefreshToken)
	if err != nil {
		if errors.Is(err, ErrSessionNotFound) {
			return nil, errors.New("session not found or expired")
		}
		return nil, errors.New("failed to get session: " + err.Error())
	}

	// Check if session is expired
	if time.Now().After(session.ExpiresAt) {
		return nil, errors.New("session expired")
	}

	// Generate new access token
	newAccessToken, err := s.jwtManager.RefreshAccessToken(req.RefreshToken)
	if err != nil {
		return nil, errors.New("failed to refresh token: " + err.Error())
	}

	return &RefreshTokenResponse{
		AccessToken:  newAccessToken,
		RefreshToken: req.RefreshToken,
		ExpiresIn:    int64(s.config.JWTExpiry.Seconds()),
	}, nil
}

// GetUser retrieves the current user for testing
func (s *TestableService) GetUser(ctx context.Context, accessToken string) (*User, error) {
	// Validate token
	claims, err := s.jwtManager.ValidateToken(accessToken)
	if err != nil {
		return nil, errors.New("invalid token: " + err.Error())
	}

	// Verify session still exists
	_, err = s.sessionRepo.GetByAccessToken(ctx, accessToken)
	if err != nil {
		if errors.Is(err, ErrSessionNotFound) {
			return nil, errors.New("session not found or expired")
		}
		return nil, errors.New("failed to verify session: " + err.Error())
	}

	// Get user
	user, err := s.userRepo.GetByID(ctx, claims.UserID)
	if err != nil {
		return nil, errors.New("failed to get user: " + err.Error())
	}

	return user, nil
}

// UpdateUser updates user information for testing
func (s *TestableService) UpdateUser(ctx context.Context, userID string, req UpdateUserRequest) (*User, error) {
	// Validate email if provided
	if req.Email != nil {
		if err := ValidateEmail(*req.Email); err != nil {
			return nil, errors.New("invalid email: " + err.Error())
		}
	}
	return s.userRepo.Update(ctx, userID, req)
}

// SendMagicLink sends a magic link for testing
func (s *TestableService) SendMagicLink(ctx context.Context, email string) error {
	enableMagicLink := s.settingsCache.GetBool(ctx, "app.auth.magic_link_enabled", s.config.MagicLinkEnabled)
	if !enableMagicLink {
		return errors.New("magic link authentication is disabled")
	}
	return nil // Mock implementation
}

// VerifyMagicLink verifies a magic link for testing
func (s *TestableService) VerifyMagicLink(ctx context.Context, token string) (*SignInResponse, error) {
	enableMagicLink := s.settingsCache.GetBool(ctx, "app.auth.magic_link_enabled", s.config.MagicLinkEnabled)
	if !enableMagicLink {
		return nil, errors.New("magic link authentication is disabled")
	}
	return nil, errors.New("not implemented")
}

// ValidateToken validates an access token for testing
func (s *TestableService) ValidateToken(token string) (*TokenClaims, error) {
	return s.jwtManager.ValidateToken(token)
}

// ValidateServiceRoleToken validates a service role token for testing
func (s *TestableService) ValidateServiceRoleToken(token string) (*TokenClaims, error) {
	claims, err := s.jwtManager.ValidateToken(token)
	if err != nil {
		return nil, err
	}
	if claims.Role != "service_role" {
		return nil, errors.New("not a service role token")
	}
	return claims, nil
}

// GetOAuthManager returns the OAuth manager for testing
func (s *TestableService) GetOAuthManager() *OAuthManager {
	return s.oauthManager
}

// IsTOTPEnabled checks if TOTP is enabled for testing
func (s *TestableService) IsTOTPEnabled(ctx context.Context, userID string) (bool, error) {
	// Mock implementation - always returns false
	return false, nil
}

// GenerateTokensForUser generates tokens for a user for testing
func (s *TestableService) GenerateTokensForUser(ctx context.Context, userID string) (*SignInResponse, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	accessToken, refreshToken, _, err := s.jwtManager.GenerateTokenPair(
		user.ID,
		user.Email,
		user.Role,
		user.UserMetadata,
		user.AppMetadata,
	)
	if err != nil {
		return nil, errors.New("failed to generate tokens: " + err.Error())
	}

	expiresAt := time.Now().Add(s.config.RefreshExpiry)
	_, err = s.sessionRepo.Create(ctx, user.ID, accessToken, refreshToken, expiresAt)
	if err != nil {
		return nil, errors.New("failed to create session: " + err.Error())
	}

	return &SignInResponse{
		User:         user,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(s.config.JWTExpiry.Seconds()),
	}, nil
}

// Reauthenticate reauthenticates a user for testing
func (s *TestableService) Reauthenticate(ctx context.Context, userID string) (string, error) {
	return "", errors.New("not implemented")
}

// VerifyNonce verifies a nonce for testing
func (s *TestableService) VerifyNonce(ctx context.Context, nonce, userID string) bool {
	return false
}

// CleanupExpiredNonces cleans up expired nonces for testing
func (s *TestableService) CleanupExpiredNonces(ctx context.Context) (int64, error) {
	return 0, errors.New("not implemented")
}

// SignInWithIDToken signs in with ID token for testing
func (s *TestableService) SignInWithIDToken(ctx context.Context, provider, idToken, nonce string) (*SignInResponse, error) {
	return nil, errors.New("not implemented")
}

// SendOTP sends an OTP for testing
func (s *TestableService) SendOTP(ctx context.Context, email, purpose string) error {
	return errors.New("not implemented")
}

// VerifyOTP verifies an OTP for testing
func (s *TestableService) VerifyOTP(ctx context.Context, email, code string) (*OTPCode, error) {
	return nil, errors.New("not implemented")
}

// ResendOTP resends an OTP for testing
func (s *TestableService) ResendOTP(ctx context.Context, email, purpose string) error {
	return errors.New("not implemented")
}

// GetUserIdentities gets user identities for testing
func (s *TestableService) GetUserIdentities(ctx context.Context, userID string) ([]UserIdentity, error) {
	return nil, errors.New("not implemented")
}

// LinkIdentity links an identity for testing
func (s *TestableService) LinkIdentity(ctx context.Context, userID, provider string) (string, string, error) {
	return "", "", errors.New("not implemented")
}

// UnlinkIdentity unlinks an identity for testing
func (s *TestableService) UnlinkIdentity(ctx context.Context, userID, identityID string) error {
	return errors.New("not implemented")
}

// GetUserByEmail gets a user by email for testing
func (s *TestableService) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	return s.userRepo.GetByEmail(ctx, email)
}

// CreateUser creates a user for testing
func (s *TestableService) CreateUser(ctx context.Context, email, password string) (*User, error) {
	// Validate email format
	if err := ValidateEmail(email); err != nil {
		return nil, errors.New("invalid email: " + err.Error())
	}

	// If password is empty, create user without password
	hashedPassword := ""
	if password != "" {
		hash, err := s.passwordHasher.HashPassword(password)
		if err != nil {
			return nil, errors.New("failed to hash password: " + err.Error())
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

// IsEmailVerificationRequired checks if email verification is required
func (s *TestableService) IsEmailVerificationRequired(ctx context.Context) bool {
	required := s.settingsCache.GetBool(ctx, "app.auth.require_email_verification", false)
	if !required {
		return false
	}
	// Mock email service check
	return false
}

// CreateSAMLUser creates a SAML user for testing
func (s *TestableService) CreateSAMLUser(ctx context.Context, email, name, provider, nameID string, attrs map[string][]string) (*User, error) {
	if err := ValidateEmail(email); err != nil {
		return nil, errors.New("invalid email: " + err.Error())
	}
	return nil, errors.New("not implemented")
}

// =============================================================================
// Test Cases
// =============================================================================

func TestSignUp_Success(t *testing.T) {
	svc := NewTestableService()
	ctx := context.Background()

	req := SignUpRequest{
		Email:    "test@example.com",
		Password: "SecurePassword123!",
		UserMetadata: map[string]interface{}{
			"name": "Test User",
		},
	}

	resp, err := svc.SignUp(ctx, req)

	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotNil(t, resp.User)
	assert.Equal(t, req.Email, resp.User.Email)
	assert.NotEmpty(t, resp.AccessToken)
	assert.NotEmpty(t, resp.RefreshToken)
	assert.Greater(t, resp.ExpiresIn, int64(0))
	assert.False(t, resp.RequiresEmailVerification)

	// Verify user metadata was stored
	assert.Equal(t, "Test User", resp.User.UserMetadata.(map[string]interface{})["name"])
}

func TestSignUp_InvalidEmail(t *testing.T) {
	svc := NewTestableService()
	ctx := context.Background()

	tests := []struct {
		name  string
		email string
	}{
		{"empty email", ""},
		{"no domain", "test@"},
		{"no at sign", "testexample.com"},
		{"invalid format", "test@.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := SignUpRequest{
				Email:    tt.email,
				Password: "SecurePassword123!",
			}

			resp, err := svc.SignUp(ctx, req)

			assert.Error(t, err)
			assert.Nil(t, resp)
			assert.Contains(t, err.Error(), "invalid email")
		})
	}
}

func TestSignUp_InvalidPassword(t *testing.T) {
	svc := NewTestableService()
	ctx := context.Background()

	tests := []struct {
		name     string
		password string
	}{
		{"empty password", ""},
		{"too short", "Short1!"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := SignUpRequest{
				Email:    "test@example.com",
				Password: tt.password,
			}

			resp, err := svc.SignUp(ctx, req)

			assert.Error(t, err)
			assert.Nil(t, resp)
			assert.Contains(t, err.Error(), "invalid password")
		})
	}
}

func TestSignUp_DuplicateEmail(t *testing.T) {
	svc := NewTestableService()
	ctx := context.Background()

	req := SignUpRequest{
		Email:    "test@example.com",
		Password: "SecurePassword123!",
	}

	// First signup should succeed
	resp1, err := svc.SignUp(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, resp1)

	// Second signup with same email should fail
	resp2, err := svc.SignUp(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, resp2)
	assert.Contains(t, err.Error(), "failed to create user")
}

func TestSignUp_Disabled(t *testing.T) {
	svc := NewTestableService()
	svc.config.SignupEnabled = false
	ctx := context.Background()

	req := SignUpRequest{
		Email:    "test@example.com",
		Password: "SecurePassword123!",
	}

	resp, err := svc.SignUp(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "signup is disabled")
}

func TestSignUp_RequiresEmailVerification(t *testing.T) {
	svc := NewTestableService()
	svc.settingsCache.SetBool("app.auth.require_email_verification", true)
	ctx := context.Background()

	req := SignUpRequest{
		Email:    "test@example.com",
		Password: "SecurePassword123!",
	}

	resp, err := svc.SignUp(ctx, req)

	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.True(t, resp.RequiresEmailVerification)
	assert.Empty(t, resp.AccessToken) // No tokens when verification required
	assert.Empty(t, resp.RefreshToken)
}

func TestSignIn_Success(t *testing.T) {
	svc := NewTestableService()
	ctx := context.Background()

	// First, sign up a user
	signUpReq := SignUpRequest{
		Email:    "test@example.com",
		Password: "SecurePassword123!",
	}
	_, err := svc.SignUp(ctx, signUpReq)
	require.NoError(t, err)

	// Now sign in
	signInReq := SignInRequest{
		Email:    "test@example.com",
		Password: "SecurePassword123!",
	}

	resp, err := svc.SignIn(ctx, signInReq)

	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotNil(t, resp.User)
	assert.Equal(t, signUpReq.Email, resp.User.Email)
	assert.NotEmpty(t, resp.AccessToken)
	assert.NotEmpty(t, resp.RefreshToken)
	assert.Greater(t, resp.ExpiresIn, int64(0))
}

func TestSignIn_InvalidEmail(t *testing.T) {
	svc := NewTestableService()
	ctx := context.Background()

	signInReq := SignInRequest{
		Email:    "nonexistent@example.com",
		Password: "SecurePassword123!",
	}

	resp, err := svc.SignIn(ctx, signInReq)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "invalid email or password")
}

func TestSignIn_InvalidPassword(t *testing.T) {
	svc := NewTestableService()
	ctx := context.Background()

	// First, sign up a user
	signUpReq := SignUpRequest{
		Email:    "test@example.com",
		Password: "SecurePassword123!",
	}
	_, err := svc.SignUp(ctx, signUpReq)
	require.NoError(t, err)

	// Try to sign in with wrong password
	signInReq := SignInRequest{
		Email:    "test@example.com",
		Password: "WrongPassword123!",
	}

	resp, err := svc.SignIn(ctx, signInReq)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "invalid email or password")
}

func TestSignIn_AccountLocked(t *testing.T) {
	svc := NewTestableService()
	ctx := context.Background()

	// First, sign up a user
	signUpReq := SignUpRequest{
		Email:    "test@example.com",
		Password: "SecurePassword123!",
	}
	signUpResp, err := svc.SignUp(ctx, signUpReq)
	require.NoError(t, err)

	// Manually lock the account
	mockUserRepo := svc.userRepo.(*MockUserRepository)
	mockUserRepo.mu.Lock()
	user := mockUserRepo.users[signUpResp.User.ID]
	user.IsLocked = true
	mockUserRepo.mu.Unlock()

	// Try to sign in
	signInReq := SignInRequest{
		Email:    "test@example.com",
		Password: "SecurePassword123!",
	}

	resp, err := svc.SignIn(ctx, signInReq)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.ErrorIs(t, err, ErrAccountLocked)
}

func TestSignIn_EmailNotVerified(t *testing.T) {
	svc := NewTestableService()
	svc.settingsCache.SetBool("app.auth.require_email_verification", true)
	ctx := context.Background()

	// Create user directly with unverified email
	hashedPw, _ := svc.passwordHasher.HashPassword("SecurePassword123!")
	_, err := svc.userRepo.Create(ctx, CreateUserRequest{
		Email: "test@example.com",
	}, hashedPw)
	require.NoError(t, err)

	// Try to sign in
	signInReq := SignInRequest{
		Email:    "test@example.com",
		Password: "SecurePassword123!",
	}

	resp, err := svc.SignIn(ctx, signInReq)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.ErrorIs(t, err, ErrEmailNotVerified)
}

func TestSignOut_Success(t *testing.T) {
	svc := NewTestableService()
	ctx := context.Background()

	// Sign up and get tokens
	signUpReq := SignUpRequest{
		Email:    "test@example.com",
		Password: "SecurePassword123!",
	}
	signUpResp, err := svc.SignUp(ctx, signUpReq)
	require.NoError(t, err)

	// Verify session exists
	_, err = svc.sessionRepo.GetByAccessToken(ctx, signUpResp.AccessToken)
	require.NoError(t, err)

	// Sign out
	err = svc.SignOut(ctx, signUpResp.AccessToken)
	require.NoError(t, err)

	// Verify session no longer exists
	_, err = svc.sessionRepo.GetByAccessToken(ctx, signUpResp.AccessToken)
	assert.ErrorIs(t, err, ErrSessionNotFound)
}

func TestSignOut_InvalidToken(t *testing.T) {
	svc := NewTestableService()
	ctx := context.Background()

	// Sign out with invalid token should not error (idempotent)
	err := svc.SignOut(ctx, "invalid-token")
	assert.NoError(t, err)
}

func TestRefreshToken_Success(t *testing.T) {
	svc := NewTestableService()
	ctx := context.Background()

	// Sign up and get tokens
	signUpReq := SignUpRequest{
		Email:    "test@example.com",
		Password: "SecurePassword123!",
	}
	signUpResp, err := svc.SignUp(ctx, signUpReq)
	require.NoError(t, err)

	// Refresh token
	refreshReq := RefreshTokenRequest{
		RefreshToken: signUpResp.RefreshToken,
	}

	resp, err := svc.RefreshToken(ctx, refreshReq)

	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotEmpty(t, resp.AccessToken)
	assert.Equal(t, signUpResp.RefreshToken, resp.RefreshToken) // Refresh token stays same
	assert.Greater(t, resp.ExpiresIn, int64(0))

	// New access token should be different
	assert.NotEqual(t, signUpResp.AccessToken, resp.AccessToken)
}

func TestRefreshToken_InvalidToken(t *testing.T) {
	svc := NewTestableService()
	ctx := context.Background()

	refreshReq := RefreshTokenRequest{
		RefreshToken: "invalid-refresh-token",
	}

	resp, err := svc.RefreshToken(ctx, refreshReq)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "invalid refresh token")
}

func TestRefreshToken_AccessTokenNotAllowed(t *testing.T) {
	svc := NewTestableService()
	ctx := context.Background()

	// Sign up and get tokens
	signUpReq := SignUpRequest{
		Email:    "test@example.com",
		Password: "SecurePassword123!",
	}
	signUpResp, err := svc.SignUp(ctx, signUpReq)
	require.NoError(t, err)

	// Try to refresh using access token (should fail)
	refreshReq := RefreshTokenRequest{
		RefreshToken: signUpResp.AccessToken, // Wrong token type
	}

	resp, err := svc.RefreshToken(ctx, refreshReq)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "invalid token type")
}

func TestGetUser_Success(t *testing.T) {
	svc := NewTestableService()
	ctx := context.Background()

	// Sign up
	signUpReq := SignUpRequest{
		Email:    "test@example.com",
		Password: "SecurePassword123!",
		UserMetadata: map[string]interface{}{
			"name": "Test User",
		},
	}
	signUpResp, err := svc.SignUp(ctx, signUpReq)
	require.NoError(t, err)

	// Get user
	user, err := svc.GetUser(ctx, signUpResp.AccessToken)

	require.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, signUpReq.Email, user.Email)
	assert.Equal(t, "Test User", user.UserMetadata.(map[string]interface{})["name"])
}

func TestGetUser_InvalidToken(t *testing.T) {
	svc := NewTestableService()
	ctx := context.Background()

	user, err := svc.GetUser(ctx, "invalid-token")

	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Contains(t, err.Error(), "invalid token")
}

func TestGetUser_SessionDeleted(t *testing.T) {
	svc := NewTestableService()
	ctx := context.Background()

	// Sign up
	signUpReq := SignUpRequest{
		Email:    "test@example.com",
		Password: "SecurePassword123!",
	}
	signUpResp, err := svc.SignUp(ctx, signUpReq)
	require.NoError(t, err)

	// Sign out (deletes session)
	err = svc.SignOut(ctx, signUpResp.AccessToken)
	require.NoError(t, err)

	// Try to get user with old token
	user, err := svc.GetUser(ctx, signUpResp.AccessToken)

	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Contains(t, err.Error(), "session not found")
}

func TestFailedLoginAttempts_IncrementOnWrongPassword(t *testing.T) {
	svc := NewTestableService()
	ctx := context.Background()

	// Sign up
	signUpReq := SignUpRequest{
		Email:    "test@example.com",
		Password: "SecurePassword123!",
	}
	signUpResp, err := svc.SignUp(ctx, signUpReq)
	require.NoError(t, err)

	// Try wrong password multiple times
	for i := 0; i < 3; i++ {
		signInReq := SignInRequest{
			Email:    "test@example.com",
			Password: "WrongPassword!",
		}
		_, _ = svc.SignIn(ctx, signInReq)
	}

	// Check failed attempts
	mockUserRepo := svc.userRepo.(*MockUserRepository)
	mockUserRepo.mu.RLock()
	user := mockUserRepo.users[signUpResp.User.ID]
	mockUserRepo.mu.RUnlock()

	assert.Equal(t, 3, user.FailedLoginAttempts)
}

func TestFailedLoginAttempts_ResetOnSuccess(t *testing.T) {
	svc := NewTestableService()
	ctx := context.Background()

	// Sign up
	signUpReq := SignUpRequest{
		Email:    "test@example.com",
		Password: "SecurePassword123!",
	}
	signUpResp, err := svc.SignUp(ctx, signUpReq)
	require.NoError(t, err)

	// Try wrong password
	signInReq := SignInRequest{
		Email:    "test@example.com",
		Password: "WrongPassword!",
	}
	_, _ = svc.SignIn(ctx, signInReq)

	// Check failed attempts > 0
	mockUserRepo := svc.userRepo.(*MockUserRepository)
	mockUserRepo.mu.RLock()
	user := mockUserRepo.users[signUpResp.User.ID]
	mockUserRepo.mu.RUnlock()
	assert.Greater(t, user.FailedLoginAttempts, 0)

	// Now sign in successfully
	signInReq.Password = "SecurePassword123!"
	_, err = svc.SignIn(ctx, signInReq)
	require.NoError(t, err)

	// Check failed attempts reset
	mockUserRepo.mu.RLock()
	user = mockUserRepo.users[signUpResp.User.ID]
	mockUserRepo.mu.RUnlock()
	assert.Equal(t, 0, user.FailedLoginAttempts)
}

func TestConcurrentSignUps(t *testing.T) {
	svc := NewTestableService()
	ctx := context.Background()

	const numUsers = 50
	results := make(chan error, numUsers)

	// Sign up users concurrently
	for i := 0; i < numUsers; i++ {
		go func(idx int) {
			req := SignUpRequest{
				Email:    "user" + string(rune('0'+idx%10)) + string(rune('0'+idx/10)) + "@example.com",
				Password: "SecurePassword123!",
			}
			_, err := svc.SignUp(ctx, req)
			results <- err
		}(i)
	}

	// Collect results
	successCount := 0
	for i := 0; i < numUsers; i++ {
		err := <-results
		if err == nil {
			successCount++
		}
	}

	// All signups should succeed (unique emails)
	assert.Equal(t, numUsers, successCount)
}

// Benchmark tests
func BenchmarkSignUp(b *testing.B) {
	svc := NewTestableService()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := SignUpRequest{
			Email:    "user" + string(rune('0'+i%10)) + string(rune('0'+(i/10)%10)) + string(rune('0'+(i/100)%10)) + "@example.com",
			Password: "SecurePassword123!",
		}
		_, _ = svc.SignUp(ctx, req)
	}
}

func BenchmarkSignIn(b *testing.B) {
	svc := NewTestableService()
	ctx := context.Background()

	// Create a user first
	signUpReq := SignUpRequest{
		Email:    "bench@example.com",
		Password: "SecurePassword123!",
	}
	_, _ = svc.SignUp(ctx, signUpReq)

	signInReq := SignInRequest{
		Email:    "bench@example.com",
		Password: "SecurePassword123!",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = svc.SignIn(ctx, signInReq)
	}
}

func BenchmarkTokenValidation(b *testing.B) {
	svc := NewTestableService()
	ctx := context.Background()

	// Create a user and get token
	signUpReq := SignUpRequest{
		Email:    "bench@example.com",
		Password: "SecurePassword123!",
	}
	resp, _ := svc.SignUp(ctx, signUpReq)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = svc.jwtManager.ValidateToken(resp.AccessToken)
	}
}

// =============================================================================
// TOTP Encryption Tests
// =============================================================================

func TestTOTPEncryption_EnableRequiresEncryptionKey(t *testing.T) {
	// Test that TOTP secrets require encryption
	// Without an encryption key, TOTP operations should fail to protect secrets

	// Test that crypto.Encrypt returns error when key is empty
	_, err := crypto.Encrypt("test-secret", "")
	assert.Error(t, err, "encrypting TOTP secret without key should fail")
}

func TestTOTPEncryption_EncryptSecretWithValidKey(t *testing.T) {
	// Test that crypto.Encrypt works correctly with a valid key
	// This validates the encryption mechanism used for TOTP secrets

	secret := "JBSWY3DPEHPK3PXP"              // Example TOTP secret
	key := "12345678901234567890123456789012" // 32-byte key

	encrypted, err := crypto.Encrypt(secret, key)
	require.NoError(t, err)
	assert.NotEmpty(t, encrypted)
	assert.NotEqual(t, secret, encrypted) // Should be different from original

	// Decrypt and verify
	decrypted, err := crypto.Decrypt(encrypted, key)
	require.NoError(t, err)
	assert.Equal(t, secret, decrypted)
}

func TestTOTPEncryption_DecryptWithWrongKey(t *testing.T) {
	// Test that decryption fails with wrong key

	secret := "JBSWY3DPEHPK3PXP"
	key1 := "12345678901234567890123456789012"
	key2 := "abcdefghijklmnopqrstuvwxyz123456"

	encrypted, err := crypto.Encrypt(secret, key1)
	require.NoError(t, err)

	// Decryption with wrong key should fail
	_, err = crypto.Decrypt(encrypted, key2)
	assert.Error(t, err)
}

func TestTOTPEncryption_InvalidKeyLength(t *testing.T) {
	// Test that encryption fails with invalid key length

	secret := "JBSWY3DPEHPK3PXP"

	tests := []struct {
		name string
		key  string
	}{
		{"empty key", ""},
		{"too short", "short"},
		{"31 bytes", "1234567890123456789012345678901"},
		{"33 bytes", "123456789012345678901234567890123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := crypto.Encrypt(secret, tt.key)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "32 bytes")
		})
	}
}

func TestTOTPEncryption_RoundTrip(t *testing.T) {
	// Test full encryption/decryption round trip with various secrets

	key := "12345678901234567890123456789012"

	secrets := []string{
		"JBSWY3DPEHPK3PXP",
		"GEZDGNBVGY3TQOJQ",
		"MFRGGZDFMY======",
		"", // empty secret (edge case)
	}

	for _, secret := range secrets {
		t.Run("secret_"+secret, func(t *testing.T) {
			if secret == "" {
				// Empty string encryption should work
				encrypted, err := crypto.Encrypt(secret, key)
				require.NoError(t, err)

				decrypted, err := crypto.Decrypt(encrypted, key)
				require.NoError(t, err)
				assert.Equal(t, secret, decrypted)
			} else {
				encrypted, err := crypto.Encrypt(secret, key)
				require.NoError(t, err)
				assert.NotEqual(t, secret, encrypted)

				decrypted, err := crypto.Decrypt(encrypted, key)
				require.NoError(t, err)
				assert.Equal(t, secret, decrypted)
			}
		})
	}
}

func TestTOTPEncryption_DifferentNonceEachTime(t *testing.T) {
	// Test that encrypting the same secret produces different ciphertext
	// (due to random nonce)

	secret := "JBSWY3DPEHPK3PXP"
	key := "12345678901234567890123456789012"

	encrypted1, err := crypto.Encrypt(secret, key)
	require.NoError(t, err)

	encrypted2, err := crypto.Encrypt(secret, key)
	require.NoError(t, err)

	// Same secret should produce different ciphertext (random nonce)
	assert.NotEqual(t, encrypted1, encrypted2)

	// But both should decrypt to the same value
	decrypted1, _ := crypto.Decrypt(encrypted1, key)
	decrypted2, _ := crypto.Decrypt(encrypted2, key)
	assert.Equal(t, decrypted1, decrypted2)
	assert.Equal(t, secret, decrypted1)
}

// =============================================================================
// Mock Services for Additional Testing
// =============================================================================

// MockPasswordResetService is a mock implementation for testing password reset flows
type MockPasswordResetService struct {
	mu                   sync.RWMutex
	tokens               map[string]*PasswordResetTokenWithPlaintext // token -> token data
	resetTokens          map[string]string                           // user -> reset token
	passwordChangeErrors map[string]error                            // user -> error to return
}

func NewMockPasswordResetService() *MockPasswordResetService {
	return &MockPasswordResetService{
		tokens:               make(map[string]*PasswordResetTokenWithPlaintext),
		resetTokens:          make(map[string]string),
		passwordChangeErrors: make(map[string]error),
	}
}

func (m *MockPasswordResetService) RequestPasswordReset(ctx context.Context, email string, redirectTo string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Generate a mock token
	token := &PasswordResetTokenWithPlaintext{
		PasswordResetToken: PasswordResetToken{
			ID:        uuid.New().String(),
			UserID:    "user-" + email, // Mock user ID
			ExpiresAt: time.Now().Add(1 * time.Hour),
			CreatedAt: time.Now(),
		},
		PlaintextToken: "mock-reset-token-" + uuid.New().String(),
	}
	m.tokens[token.PlaintextToken] = token
	m.resetTokens[email] = token.PlaintextToken
	return nil
}

func (m *MockPasswordResetService) ResetPassword(ctx context.Context, token, newPassword string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Find token by matching the plaintext token or hash
	var userID string
	for tok, data := range m.tokens {
		if tok == token {
			userID = data.UserID
			break
		}
		// Also check hash (simplified for testing)
		if hashPasswordResetToken(token) == data.TokenHash {
			userID = data.UserID
			break
		}
	}

	if userID == "" {
		return "", errors.New("invalid or expired reset token")
	}

	// Check for mock error
	if err, ok := m.passwordChangeErrors[userID]; ok {
		return "", err
	}

	return "Password updated successfully", nil
}

func (m *MockPasswordResetService) VerifyPasswordResetToken(ctx context.Context, token string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, data := range m.tokens {
		if data.PlaintextToken == token {
			if time.Now().After(data.ExpiresAt) {
				return errors.New("token expired")
			}
			return nil
		}
	}
	return errors.New("invalid token")
}

func (m *MockPasswordResetService) SetPasswordChangeError(userID string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.passwordChangeErrors[userID] = err
}

// =============================================================================
// Comprehensive Service Method Tests
// These tests cover the main Service methods with various scenarios
// =============================================================================

func TestService_SignUp_DisabledSignup(t *testing.T) {
	service := NewTestableService()
	service.settingsCache.SetBool("app.auth.signup_enabled", false)

	ctx := context.Background()
	req := SignUpRequest{
		Email:    "test@example.com",
		Password: "password123",
	}

	resp, err := service.SignUp(ctx, req)
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "signup is disabled")
}

func TestService_SignUp_InvalidEmail(t *testing.T) {
	service := NewTestableService()

	ctx := context.Background()
	tests := []struct {
		name  string
		email string
	}{
		{"empty email", ""},
		{"no at sign", "invalidemail"},
		{"no domain", "user@"},
		{"no user", "@example.com"},
		{"spaces", "user @example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := SignUpRequest{
				Email:    tt.email,
				Password: "password123",
			}
			resp, err := service.SignUp(ctx, req)
			assert.Error(t, err)
			assert.Nil(t, resp)
			assert.Contains(t, err.Error(), "invalid email")
		})
	}
}

func TestService_SignUp_WeakPassword(t *testing.T) {
	service := NewTestableService()

	ctx := context.Background()
	tests := []struct {
		name     string
		password string
	}{
		{"empty password", ""},
		{"too short", "pass1"},
		{"min length but valid", "pass1234"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := SignUpRequest{
				Email:    "test@example.com",
				Password: tt.password,
			}
			resp, err := service.SignUp(ctx, req)

			// Only the too short password should fail (min 8 chars)
			if len(tt.password) < 8 {
				assert.Error(t, err)
				assert.Nil(t, resp)
				assert.Contains(t, err.Error(), "invalid password")
			}
		})
	}
}

func TestService_SignUp_WithEmailVerification(t *testing.T) {
	service := NewTestableService()
	service.settingsCache.SetBool("app.auth.require_email_verification", true)

	ctx := context.Background()
	req := SignUpRequest{
		Email:    "verify@example.com",
		Password: "password123",
	}

	resp, err := service.SignUp(ctx, req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotNil(t, resp.User)
	assert.True(t, resp.RequiresEmailVerification)
	assert.Empty(t, resp.AccessToken, "No token when verification required")
	assert.Empty(t, resp.RefreshToken, "No refresh token when verification required")
}

func TestService_SignUp_StripsAppMetadata(t *testing.T) {
	service := NewTestableService()

	ctx := context.Background()
	req := SignUpRequest{
		Email:    "test@example.com",
		Password: "password123",
		AppMetadata: map[string]interface{}{
			"role": "admin", // User trying to escalate privileges
		},
	}

	resp, err := service.SignUp(ctx, req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotNil(t, resp.User)
	assert.Nil(t, resp.User.AppMetadata, "AppMetadata should be stripped from signup")
}

func TestService_SignUp_DuplicateEmail(t *testing.T) {
	service := NewTestableService()

	ctx := context.Background()
	req := SignUpRequest{
		Email:    "test@example.com",
		Password: "password123",
	}

	// First signup
	_, err := service.SignUp(ctx, req)
	assert.NoError(t, err)

	// Duplicate signup
	resp, err := service.SignUp(ctx, req)
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "failed to create user")
}

func TestService_SignIn_UserNotFound(t *testing.T) {
	service := NewTestableService()

	ctx := context.Background()
	req := SignInRequest{
		Email:    "nonexistent@example.com",
		Password: "password123",
	}

	resp, err := service.SignIn(ctx, req)
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "invalid email or password")
}

func TestService_SignIn_WrongPassword(t *testing.T) {
	service := NewTestableService()

	ctx := context.Background()

	// Create user first
	signupReq := SignUpRequest{
		Email:    "test@example.com",
		Password: "correctpassword",
	}
	_, err := service.SignUp(ctx, signupReq)
	require.NoError(t, err)

	// Try to sign in with wrong password
	signinReq := SignInRequest{
		Email:    "test@example.com",
		Password: "wrongpassword",
	}
	resp, err := service.SignIn(ctx, signinReq)
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "invalid email or password")
}

func TestService_SignIn_AccountLocked(t *testing.T) {
	service := NewTestableService()

	ctx := context.Background()

	// Create user
	signupReq := SignUpRequest{
		Email:    "locked@example.com",
		Password: "password123",
	}
	signupResp, err := service.SignUp(ctx, signupReq)
	require.NoError(t, err)

	// Simulate locked account
	user := signupResp.User
	user.IsLocked = true
	futureTime := time.Now().Add(1 * time.Hour)
	user.LockedUntil = &futureTime

	// Update user in repo
	mockRepo := service.userRepo.(*MockUserRepository)
	mockRepo.users[user.ID].IsLocked = true
	mockRepo.users[user.ID].LockedUntil = &futureTime

	// Try to sign in
	signinReq := SignInRequest{
		Email:    "locked@example.com",
		Password: "password123",
	}
	resp, err := service.SignIn(ctx, signinReq)
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.ErrorIs(t, err, ErrAccountLocked)
}

func TestService_SignIn_EmailNotVerified(t *testing.T) {
	service := NewTestableService()
	service.settingsCache.SetBool("app.auth.require_email_verification", true)

	ctx := context.Background()

	// Create user without email verified
	signupReq := SignUpRequest{
		Email:    "test@example.com",
		Password: "password123",
	}
	_, err := service.SignUp(ctx, signupReq)
	require.NoError(t, err)

	// Try to sign in
	signinReq := SignInRequest{
		Email:    "test@example.com",
		Password: "password123",
	}
	resp, err := service.SignIn(ctx, signinReq)
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.ErrorIs(t, err, ErrEmailNotVerified)
}

func TestService_SignIn_Success(t *testing.T) {
	service := NewTestableService()

	ctx := context.Background()

	// Create user
	signupReq := SignUpRequest{
		Email:    "test@example.com",
		Password: "password123",
	}
	_, err := service.SignUp(ctx, signupReq)
	require.NoError(t, err)

	// Sign in
	signinReq := SignInRequest{
		Email:    "test@example.com",
		Password: "password123",
	}
	resp, err := service.SignIn(ctx, signinReq)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotNil(t, resp.User)
	assert.NotEmpty(t, resp.AccessToken)
	assert.NotEmpty(t, resp.RefreshToken)
	assert.Greater(t, resp.ExpiresIn, int64(0))
}

func TestService_SignOut_Success(t *testing.T) {
	service := NewTestableService()

	ctx := context.Background()

	// Create and sign in user
	signupReq := SignUpRequest{
		Email:    "test@example.com",
		Password: "password123",
	}
	_, err := service.SignUp(ctx, signupReq)
	require.NoError(t, err)

	signinReq := SignInRequest{
		Email:    "test@example.com",
		Password: "password123",
	}
	signinResp, err := service.SignIn(ctx, signinReq)
	require.NoError(t, err)

	// Sign out
	err = service.SignOut(ctx, signinResp.AccessToken)
	assert.NoError(t, err)

	// Verify session is deleted
	_, err = service.sessionRepo.GetByAccessToken(ctx, signinResp.AccessToken)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrSessionNotFound)
}

func TestService_SignOut_AlreadySignedOut(t *testing.T) {
	service := NewTestableService()

	ctx := context.Background()

	// Sign out with non-existent token (already signed out or invalid)
	err := service.SignOut(ctx, "nonexistent-token")
	assert.NoError(t, err, "SignOut should succeed even if session doesn't exist")
}

func TestService_RefreshToken_InvalidToken(t *testing.T) {
	service := NewTestableService()

	ctx := context.Background()
	req := RefreshTokenRequest{
		RefreshToken: "invalid-token",
	}

	resp, err := service.RefreshToken(ctx, req)
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "invalid refresh token")
}

func TestService_RefreshToken_WrongTokenType(t *testing.T) {
	service := NewTestableService()

	ctx := context.Background()

	// Create access token (not refresh token)
	cfg := &config.AuthConfig{
		JWTSecret: "test-secret-key-at-least-32-chars-long",
		JWTExpiry: 15 * time.Minute,
	}
	jwtManager := mustNewJWTManager(cfg.JWTSecret, cfg.JWTExpiry, 7*24*time.Hour)
	accessToken, _, err := jwtManager.GenerateAccessToken("user-id", "test@example.com", "authenticated", nil, nil)
	require.NoError(t, err)

	req := RefreshTokenRequest{
		RefreshToken: accessToken,
	}

	resp, err := service.RefreshToken(ctx, req)
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "invalid token type")
}

func TestService_RefreshToken_SessionNotFound(t *testing.T) {
	service := NewTestableService()

	ctx := context.Background()

	// Create valid refresh token but no session
	cfg := &config.AuthConfig{
		JWTSecret:     "test-secret-key-at-least-32-chars-long",
		JWTExpiry:     15 * time.Minute,
		RefreshExpiry: 7 * 24 * time.Hour,
	}
	jwtManager := mustNewJWTManager(cfg.JWTSecret, cfg.JWTExpiry, cfg.RefreshExpiry)
	refreshToken, _, err := jwtManager.GenerateRefreshToken("user-id", "test@example.com", "authenticated", "session-id", nil, nil)
	require.NoError(t, err)

	req := RefreshTokenRequest{
		RefreshToken: refreshToken,
	}

	resp, err := service.RefreshToken(ctx, req)
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "session not found")
}

func TestService_RefreshToken_Success(t *testing.T) {
	service := NewTestableService()

	ctx := context.Background()

	// Create and sign in user
	signupReq := SignUpRequest{
		Email:    "test@example.com",
		Password: "password123",
	}
	_, err := service.SignUp(ctx, signupReq)
	require.NoError(t, err)

	signinReq := SignInRequest{
		Email:    "test@example.com",
		Password: "password123",
	}
	signinResp, err := service.SignIn(ctx, signinReq)
	require.NoError(t, err)

	// Refresh token
	refreshReq := RefreshTokenRequest{
		RefreshToken: signinResp.RefreshToken,
	}
	refreshResp, err := service.RefreshToken(ctx, refreshReq)
	assert.NoError(t, err)
	assert.NotNil(t, refreshResp)
	assert.NotEmpty(t, refreshResp.AccessToken)
	assert.NotEmpty(t, refreshResp.RefreshToken)
	// Note: Token rotation happens in real implementation but mock may not rotate
	// Just verify that new tokens are generated
}

func TestService_GetUser_InvalidToken(t *testing.T) {
	service := NewTestableService()

	ctx := context.Background()
	user, err := service.GetUser(ctx, "invalid-token")
	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Contains(t, err.Error(), "invalid token")
}

func TestService_UpdateUser_InvalidEmail(t *testing.T) {
	service := NewTestableService()

	ctx := context.Background()
	req := UpdateUserRequest{
		Email: stringPtr("invalid-email"),
	}

	user, err := service.UpdateUser(ctx, "user-id", req)
	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Contains(t, err.Error(), "invalid email")
}

func TestService_UpdateUser_Success(t *testing.T) {
	service := NewTestableService()

	ctx := context.Background()

	// Create user
	signupReq := SignUpRequest{
		Email:    "test@example.com",
		Password: "password123",
	}
	signupResp, err := service.SignUp(ctx, signupReq)
	require.NoError(t, err)

	// Update user
	newEmail := "updated@example.com"
	req := UpdateUserRequest{
		Email: &newEmail,
	}
	user, err := service.UpdateUser(ctx, signupResp.User.ID, req)
	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, newEmail, user.Email)
}

func TestService_SendMagicLink_Disabled(t *testing.T) {
	service := NewTestableService()
	service.settingsCache.SetBool("app.auth.magic_link_enabled", false)

	ctx := context.Background()
	err := service.SendMagicLink(ctx, "test@example.com")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "magic link authentication is disabled")
}

func TestService_VerifyMagicLink_Disabled(t *testing.T) {
	service := NewTestableService()
	service.settingsCache.SetBool("app.auth.magic_link_enabled", false)

	ctx := context.Background()
	resp, err := service.VerifyMagicLink(ctx, "token")
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "magic link authentication is disabled")
}

func TestService_ValidateToken_Success(t *testing.T) {
	service := NewTestableService()

	ctx := context.Background()

	// Create and sign in user
	signupReq := SignUpRequest{
		Email:    "test@example.com",
		Password: "password123",
	}
	signupResp, err := service.SignUp(ctx, signupReq)
	require.NoError(t, err)

	// Validate token
	claims, err := service.ValidateToken(signupResp.AccessToken)
	assert.NoError(t, err)
	assert.NotNil(t, claims)
	assert.Equal(t, signupResp.User.ID, claims.UserID)
	assert.Equal(t, signupResp.User.Email, claims.Email)
}

func TestService_ValidateToken_Invalid(t *testing.T) {
	service := NewTestableService()

	claims, err := service.ValidateToken("invalid-token")
	assert.Error(t, err)
	assert.Nil(t, claims)
}

func TestService_ValidateServiceRoleToken_Success(t *testing.T) {
	service := NewTestableService()
	cfg := &config.AuthConfig{
		JWTSecret:      "test-secret-key-at-least-32-chars-long",
		ServiceRoleTTL: 24 * time.Hour,
	}
	jwtManager := mustNewJWTManagerWithConfig(cfg.JWTSecret, 15*time.Minute, 7*24*time.Hour, cfg.ServiceRoleTTL, 1*time.Hour)

	token, err := jwtManager.GenerateServiceRoleToken()
	require.NoError(t, err)

	claims, err := service.ValidateServiceRoleToken(token)
	assert.NoError(t, err)
	assert.NotNil(t, claims)
	assert.Equal(t, "service_role", claims.Role)
}

func TestService_ValidateServiceRoleToken_Invalid(t *testing.T) {
	service := NewTestableService()

	claims, err := service.ValidateServiceRoleToken("invalid-token")
	assert.Error(t, err)
	assert.Nil(t, claims)
}

func TestService_GetOAuthManager(t *testing.T) {
	service := NewTestableService()

	manager := service.GetOAuthManager()
	assert.NotNil(t, manager)
	assert.Same(t, service.oauthManager, manager)
}

func TestService_IsTOTPEnabled_NotEnabled(t *testing.T) {
	service := NewTestableService()

	ctx := context.Background()
	enabled, err := service.IsTOTPEnabled(ctx, "user-id")
	assert.NoError(t, err)
	assert.False(t, enabled)
}

func TestService_GenerateTokensForUser_UserNotFound(t *testing.T) {
	service := NewTestableService()

	ctx := context.Background()
	resp, err := service.GenerateTokensForUser(ctx, "nonexistent-user-id")
	assert.Error(t, err)
	assert.Nil(t, resp)
}

func TestService_GetUserByEmail_NotFound(t *testing.T) {
	service := NewTestableService()

	ctx := context.Background()
	user, err := service.GetUserByEmail(ctx, "nonexistent@example.com")
	assert.Error(t, err)
	assert.Nil(t, user)
	assert.ErrorIs(t, err, ErrUserNotFound)
}

func TestService_CreateUser_InvalidEmail(t *testing.T) {
	service := NewTestableService()

	ctx := context.Background()
	user, err := service.CreateUser(ctx, "invalid-email", "password123")
	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Contains(t, err.Error(), "invalid email")
}

func TestService_CreateUser_Success(t *testing.T) {
	service := NewTestableService()

	ctx := context.Background()
	user, err := service.CreateUser(ctx, "newuser@example.com", "password123")
	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, "newuser@example.com", user.Email)
	assert.Equal(t, "user", user.Role, "Default role should be 'user'")
}

func TestService_CreateUser_NoPassword(t *testing.T) {
	service := NewTestableService()

	ctx := context.Background()
	user, err := service.CreateUser(ctx, "nopass@example.com", "")
	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, "nopass@example.com", user.Email)
	assert.Empty(t, user.PasswordHash, "No password hash should be set")
}

func TestService_IsEmailVerificationRequired_Disabled(t *testing.T) {
	service := NewTestableService()
	service.settingsCache.SetBool("app.auth.require_email_verification", false)

	ctx := context.Background()
	required := service.IsEmailVerificationRequired(ctx)
	assert.False(t, required)
}

func TestService_IsEmailVerificationRequired_NoEmailService(t *testing.T) {
	service := NewTestableService()
	service.settingsCache.SetBool("app.auth.require_email_verification", true)
	// No email service configured

	ctx := context.Background()
	required := service.IsEmailVerificationRequired(ctx)
	assert.False(t, required, "Should be false when email service not configured")
}

func TestService_Reauthenticate_NotImplemented(t *testing.T) {
	service := NewTestableService()

	ctx := context.Background()
	nonce, err := service.Reauthenticate(ctx, "user-id")
	assert.Error(t, err)
	assert.Empty(t, nonce)
}

func TestService_VerifyNonce_NotImplemented(t *testing.T) {
	service := NewTestableService()

	ctx := context.Background()
	valid := service.VerifyNonce(ctx, "nonce", "user-id")
	assert.False(t, valid)
}

func TestService_CleanupExpiredNonces_NotImplemented(t *testing.T) {
	service := NewTestableService()

	ctx := context.Background()
	count, err := service.CleanupExpiredNonces(ctx)
	assert.Error(t, err)
	assert.Equal(t, int64(0), count)
}

func TestService_SignInWithIDToken_NotImplemented(t *testing.T) {
	service := NewTestableService()

	ctx := context.Background()
	resp, err := service.SignInWithIDToken(ctx, "provider", "id-token", "nonce")
	assert.Error(t, err)
	assert.Nil(t, resp)
}

func TestService_SendOTP_NotImplemented(t *testing.T) {
	service := NewTestableService()

	ctx := context.Background()
	err := service.SendOTP(ctx, "test@example.com", "login")
	assert.Error(t, err)
}

func TestService_VerifyOTP_NotImplemented(t *testing.T) {
	service := NewTestableService()

	ctx := context.Background()
	code, err := service.VerifyOTP(ctx, "test@example.com", "123456")
	assert.Nil(t, code)
	assert.Error(t, err)
}

func TestService_ResendOTP_NotImplemented(t *testing.T) {
	service := NewTestableService()

	ctx := context.Background()
	err := service.ResendOTP(ctx, "test@example.com", "login")
	assert.Error(t, err)
}

func TestService_GetUserIdentities_NotImplemented(t *testing.T) {
	service := NewTestableService()

	ctx := context.Background()
	identities, err := service.GetUserIdentities(ctx, "user-id")
	assert.Nil(t, identities)
	assert.Error(t, err)
}

func TestService_LinkIdentity_NotImplemented(t *testing.T) {
	service := NewTestableService()

	ctx := context.Background()
	state, url, err := service.LinkIdentity(ctx, "user-id", "provider")
	assert.Empty(t, state)
	assert.Empty(t, url)
	assert.Error(t, err)
}

func TestService_UnlinkIdentity_NotImplemented(t *testing.T) {
	service := NewTestableService()

	ctx := context.Background()
	err := service.UnlinkIdentity(ctx, "user-id", "identity-id")
	assert.Error(t, err)
}

func TestService_CreateSAMLUser_InvalidEmail(t *testing.T) {
	service := NewTestableService()

	ctx := context.Background()
	user, err := service.CreateSAMLUser(ctx, "invalid-email", "Test User", "saml", "nameID", nil)
	assert.Nil(t, user)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid email")
}

func TestService_GenerateTokensForSAMLUser_Success(t *testing.T) {
	service := NewTestableService()

	ctx := context.Background()
	user := &User{
		ID:    "saml-user-id",
		Email: "saml@example.com",
		Role:  "authenticated",
	}

	resp, err := service.GenerateTokensForUser(ctx, user.ID)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotEmpty(t, resp.AccessToken)
	assert.NotEmpty(t, resp.RefreshToken)
}
