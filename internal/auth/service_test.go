package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/fluxbase-eu/fluxbase/internal/config"
	"github.com/fluxbase-eu/fluxbase/internal/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestableService wraps the auth service components for unit testing
// without requiring a database connection.
type TestableService struct {
	userRepo              UserRepositoryInterface
	sessionRepo           SessionRepositoryInterface
	tokenBlacklistRepo    TokenBlacklistRepositoryInterface
	jwtManager            *JWTManager
	passwordHasher        *PasswordHasher
	settingsCache         *MockSettingsCache
	config                *config.AuthConfig
	emailVerificationRepo *MockEmailVerificationRepository
}

// MockSettingsCache implements a simple settings cache for testing
type MockSettingsCache struct {
	boolSettings   map[string]bool
	stringSettings map[string]string
	intSettings    map[string]int
}

func NewMockSettingsCache() *MockSettingsCache {
	return &MockSettingsCache{
		boolSettings:   make(map[string]bool),
		stringSettings: make(map[string]string),
		intSettings:    make(map[string]int),
	}
}

func (m *MockSettingsCache) GetBool(ctx context.Context, key string, defaultValue bool) bool {
	if val, ok := m.boolSettings[key]; ok {
		return val
	}
	return defaultValue
}

func (m *MockSettingsCache) GetString(ctx context.Context, key string, defaultValue string) string {
	if val, ok := m.stringSettings[key]; ok {
		return val
	}
	return defaultValue
}

func (m *MockSettingsCache) GetInt(ctx context.Context, key string, defaultValue int) int {
	if val, ok := m.intSettings[key]; ok {
		return val
	}
	return defaultValue
}

func (m *MockSettingsCache) SetBool(key string, value bool) {
	m.boolSettings[key] = value
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
		jwtManager:            NewJWTManager(cfg.JWTSecret, cfg.JWTExpiry, cfg.RefreshExpiry),
		passwordHasher:        NewPasswordHasherWithConfig(PasswordHasherConfig{MinLength: cfg.PasswordMinLen, Cost: cfg.BcryptCost}),
		settingsCache:         NewMockSettingsCache(),
		config:                cfg,
		emailVerificationRepo: NewMockEmailVerificationRepository(),
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
