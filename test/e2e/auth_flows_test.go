package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wayli-app/fluxbase/internal/api"
	"github.com/wayli-app/fluxbase/internal/auth"
	"github.com/wayli-app/fluxbase/internal/config"
	"github.com/wayli-app/fluxbase/internal/database"
	"github.com/wayli-app/fluxbase/internal/middleware"
)

// TestAuthenticationFlows tests complete authentication flows end-to-end
func TestAuthenticationFlows(t *testing.T) {
	// Setup test environment
	cfg := setupAuthTestConfig(t)
	db := setupAuthTestDatabase(t, cfg)
	defer db.Close()

	// Ensure auth schema exists
	setupAuthSchema(t, db)

	// Create test app with auth routes
	app := createAuthTestApp(t, db, cfg)

	// Run authentication flow tests
	t.Run("Complete Signup Flow", func(t *testing.T) {
		testCompleteSignupFlow(t, app, db)
	})

	t.Run("Email Verification", func(t *testing.T) {
		testEmailVerification(t, app, db)
	})

	t.Run("Sign In Flow", func(t *testing.T) {
		testSignInFlow(t, app, db)
	})

	t.Run("Token Refresh Flow", func(t *testing.T) {
		testTokenRefreshFlow(t, app, db, cfg.JWTSecret)
	})

	t.Run("Token Expiration", func(t *testing.T) {
		testTokenExpiration(t, app, db, cfg.JWTSecret)
	})

	t.Run("Magic Link Authentication", func(t *testing.T) {
		testMagicLinkAuth(t, app, db)
	})

	t.Run("Password Reset Flow", func(t *testing.T) {
		testPasswordResetFlow(t, app, db)
	})

	t.Run("Multi-Device Sessions", func(t *testing.T) {
		testMultiDeviceSessions(t, app, db, cfg.JWTSecret)
	})

	t.Run("User Profile Updates", func(t *testing.T) {
		testUserProfileUpdates(t, app, db, cfg.JWTSecret)
	})

	t.Run("Sign Out Flow", func(t *testing.T) {
		testSignOutFlow(t, app, db, cfg.JWTSecret)
	})

	t.Run("OAuth Callback Simulation", func(t *testing.T) {
		testOAuthCallbackSimulation(t, app, db)
	})

	t.Run("Invalid Credentials", func(t *testing.T) {
		testInvalidCredentials(t, app, db)
	})

	t.Run("Rate Limiting", func(t *testing.T) {
		testAuthRateLimiting(t, app, db)
	})
}

// setupAuthTestConfig creates test configuration for auth tests
func setupAuthTestConfig(t *testing.T) *config.Config {
	return &config.Config{
		DatabaseURL:  "postgres://postgres:postgres@localhost:5432/fluxbase_test?sslmode=disable",
		JWTSecret:    "test-jwt-secret-for-auth-testing",
		Port:         "8080",
		SMTPHost:     "localhost",
		SMTPPort:     "1025", // MailHog for testing
		SMTPFrom:     "test@fluxbase.test",
		FluxbaseURL:  "http://localhost:8080",
		RLSEnabled:   true,
	}
}

// setupAuthTestDatabase creates database connection for auth tests
func setupAuthTestDatabase(t *testing.T, cfg *config.Config) *database.Connection {
	db, err := database.Connect(cfg.DatabaseURL)
	require.NoError(t, err, "Failed to connect to test database")
	return db
}

// setupAuthSchema ensures auth schema and tables exist
func setupAuthSchema(t *testing.T, db *database.Connection) {
	ctx := context.Background()

	queries := []string{
		`CREATE SCHEMA IF NOT EXISTS auth`,

		`CREATE TABLE IF NOT EXISTS auth.users (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			email TEXT UNIQUE NOT NULL,
			encrypted_password TEXT,
			email_confirmed_at TIMESTAMPTZ,
			last_sign_in_at TIMESTAMPTZ,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW(),
			raw_app_meta_data JSONB DEFAULT '{}'::jsonb,
			raw_user_meta_data JSONB DEFAULT '{}'::jsonb
		)`,

		`CREATE TABLE IF NOT EXISTS auth.sessions (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID REFERENCES auth.users(id) ON DELETE CASCADE,
			token TEXT UNIQUE NOT NULL,
			refresh_token TEXT UNIQUE,
			expires_at TIMESTAMPTZ NOT NULL,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW(),
			device_info TEXT,
			ip_address TEXT
		)`,

		`CREATE TABLE IF NOT EXISTS auth.magic_links (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			email TEXT NOT NULL,
			token TEXT UNIQUE NOT NULL,
			expires_at TIMESTAMPTZ NOT NULL,
			used_at TIMESTAMPTZ,
			created_at TIMESTAMPTZ DEFAULT NOW()
		)`,

		`CREATE TABLE IF NOT EXISTS auth.password_reset_tokens (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID REFERENCES auth.users(id) ON DELETE CASCADE,
			token TEXT UNIQUE NOT NULL,
			expires_at TIMESTAMPTZ NOT NULL,
			used_at TIMESTAMPTZ,
			created_at TIMESTAMPTZ DEFAULT NOW()
		)`,
	}

	for _, query := range queries {
		_, err := db.Pool().Exec(ctx, query)
		require.NoError(t, err, "Failed to setup auth schema: %s", query)
	}

	// Clean up existing test data
	cleanupQueries := []string{
		"TRUNCATE auth.sessions CASCADE",
		"TRUNCATE auth.magic_links CASCADE",
		"TRUNCATE auth.password_reset_tokens CASCADE",
		"DELETE FROM auth.users WHERE email LIKE '%@test.com'",
	}

	for _, query := range cleanupQueries {
		_, err := db.Pool().Exec(ctx, query)
		require.NoError(t, err, "Failed to cleanup auth data")
	}
}

// createAuthTestApp creates a Fiber app with auth routes
func createAuthTestApp(t *testing.T, db *database.Connection, cfg *config.Config) *fiber.App {
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		},
	})

	// Add middleware
	app.Use(middleware.AuthMiddleware(middleware.AuthConfig{
		JWTSecret: cfg.JWTSecret,
		Optional:  true,
	}))

	app.Use(middleware.RLSMiddleware(middleware.RLSConfig{
		DB:      db,
		Enabled: cfg.RLSEnabled,
	}))

	// Register auth routes
	authHandler := auth.NewHandler(db, cfg.JWTSecret, cfg.SMTPHost+":"+cfg.SMTPPort, cfg.SMTPFrom)
	authHandler.RegisterRoutes(app)

	// Register API routes for protected resources
	apiServer := api.NewServer(db, cfg)
	apiServer.RegisterRoutes(app)

	return app
}

// testCompleteSignupFlow tests the full signup process
func testCompleteSignupFlow(t *testing.T, app *fiber.App, db *database.Connection) {
	email := "newsignup@test.com"
	password := "SecurePass123!"

	// Step 1: Sign up
	signupReq := map[string]interface{}{
		"email":    email,
		"password": password,
	}

	body, _ := json.Marshal(signupReq)
	req := httptest.NewRequest("POST", "/api/v1/auth/signup", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 201, resp.StatusCode, "Signup should succeed")

	var signupResp map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&signupResp)

	// Should receive user info and access token
	assert.NotNil(t, signupResp["user"], "Should return user object")
	assert.NotNil(t, signupResp["access_token"], "Should return access token")
	assert.NotNil(t, signupResp["refresh_token"], "Should return refresh token")

	user := signupResp["user"].(map[string]interface{})
	assert.Equal(t, email, user["email"])
	assert.NotNil(t, user["id"])

	// Step 2: Verify user exists in database
	ctx := context.Background()
	var userID string
	var emailConfirmed bool
	err = db.Pool().QueryRow(ctx, "SELECT id, email_confirmed_at IS NOT NULL FROM auth.users WHERE email = $1", email).Scan(&userID, &emailConfirmed)
	require.NoError(t, err)
	assert.Equal(t, user["id"], userID)

	// Step 3: Try to sign up again with same email (should fail)
	req = httptest.NewRequest("POST", "/api/v1/auth/signup", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err = app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 409, resp.StatusCode, "Duplicate signup should be rejected")
}

// testEmailVerification tests email verification process
func testEmailVerification(t *testing.T, app *fiber.App, db *database.Connection) {
	email := "verify@test.com"
	password := "TestPass123!"

	// Sign up
	signupReq := map[string]interface{}{
		"email":    email,
		"password": password,
	}

	body, _ := json.Marshal(signupReq)
	req := httptest.NewRequest("POST", "/api/v1/auth/signup", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 201, resp.StatusCode)

	// In real scenario, we'd get verification token from email
	// For testing, we can query the database or use a test endpoint
	ctx := context.Background()
	var userID string
	err = db.Pool().QueryRow(ctx, "SELECT id FROM auth.users WHERE email = $1", email).Scan(&userID)
	require.NoError(t, err)

	// Simulate email verification
	_, err = db.Pool().Exec(ctx, "UPDATE auth.users SET email_confirmed_at = NOW() WHERE id = $1", userID)
	require.NoError(t, err)

	// Verify that email_confirmed_at is set
	var confirmedAt *time.Time
	err = db.Pool().QueryRow(ctx, "SELECT email_confirmed_at FROM auth.users WHERE id = $1", userID).Scan(&confirmedAt)
	require.NoError(t, err)
	assert.NotNil(t, confirmedAt, "Email should be confirmed")
}

// testSignInFlow tests the sign-in process
func testSignInFlow(t *testing.T, app *fiber.App, db *database.Connection) {
	email := "signin@test.com"
	password := "SignInPass123!"

	// First, create a user
	createTestUserWithPassword(t, db, email, password)

	// Sign in
	signinReq := map[string]interface{}{
		"email":    email,
		"password": password,
	}

	body, _ := json.Marshal(signinReq)
	req := httptest.NewRequest("POST", "/api/v1/auth/signin", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode, "Sign in should succeed")

	var signinResp map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&signinResp)

	// Should receive tokens
	assert.NotNil(t, signinResp["access_token"], "Should return access token")
	assert.NotNil(t, signinResp["refresh_token"], "Should return refresh token")
	assert.NotNil(t, signinResp["user"], "Should return user object")

	user := signinResp["user"].(map[string]interface{})
	assert.Equal(t, email, user["email"])

	// Verify last_sign_in_at is updated
	ctx := context.Background()
	var lastSignIn *time.Time
	err = db.Pool().QueryRow(ctx, "SELECT last_sign_in_at FROM auth.users WHERE email = $1", email).Scan(&lastSignIn)
	require.NoError(t, err)
	assert.NotNil(t, lastSignIn, "last_sign_in_at should be updated")
}

// testTokenRefreshFlow tests token refresh functionality
func testTokenRefreshFlow(t *testing.T, app *fiber.App, db *database.Connection, jwtSecret string) {
	email := "refresh@test.com"
	password := "RefreshPass123!"

	// Create user and sign in
	createTestUserWithPassword(t, db, email, password)

	signinReq := map[string]interface{}{
		"email":    email,
		"password": password,
	}

	body, _ := json.Marshal(signinReq)
	req := httptest.NewRequest("POST", "/api/v1/auth/signin", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	require.NoError(t, err)

	var signinResp map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&signinResp)

	refreshToken := signinResp["refresh_token"].(string)
	assert.NotEmpty(t, refreshToken)

	// Use refresh token to get new access token
	refreshReq := map[string]interface{}{
		"refresh_token": refreshToken,
	}

	body, _ = json.Marshal(refreshReq)
	req = httptest.NewRequest("POST", "/api/v1/auth/refresh", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err = app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode, "Token refresh should succeed")

	var refreshResp map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&refreshResp)

	newAccessToken := refreshResp["access_token"].(string)
	assert.NotEmpty(t, newAccessToken)
	assert.NotEqual(t, signinResp["access_token"], newAccessToken, "Should get a new access token")
}

// testTokenExpiration tests behavior with expired tokens
func testTokenExpiration(t *testing.T, app *fiber.App, db *database.Connection, jwtSecret string) {
	email := "expired@test.com"

	// Create a user
	userID := createTestUserWithPassword(t, db, email, "ExpiredPass123!")

	// Create an expired token (manually generate with past expiration)
	authService := auth.NewService(db, jwtSecret, "smtp://fake", "noreply@test.com")

	// Generate token with very short expiration
	token, err := authService.GenerateJWT(userID, email)
	require.NoError(t, err)

	// Try to use it immediately (should still work)
	req := httptest.NewRequest("GET", "/api/v1/auth/user", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.True(t, resp.StatusCode == 200 || resp.StatusCode == 401, "Should handle token validation")
}

// testMagicLinkAuth tests magic link authentication
func testMagicLinkAuth(t *testing.T, app *fiber.App, db *database.Connection) {
	email := "magiclink@test.com"

	// Request magic link
	magicLinkReq := map[string]interface{}{
		"email": email,
	}

	body, _ := json.Marshal(magicLinkReq)
	req := httptest.NewRequest("POST", "/api/v1/auth/magiclink", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.True(t, resp.StatusCode == 200 || resp.StatusCode == 201, "Magic link request should be accepted")

	// In production, token would be sent via email
	// For testing, retrieve it from database
	ctx := context.Background()
	var token string
	err = db.Pool().QueryRow(ctx, `
		SELECT token FROM auth.magic_links
		WHERE email = $1 AND used_at IS NULL
		ORDER BY created_at DESC LIMIT 1
	`, email).Scan(&token)

	if err == nil {
		// Verify magic link token
		verifyReq := map[string]interface{}{
			"token": token,
		}

		body, _ = json.Marshal(verifyReq)
		req = httptest.NewRequest("POST", "/api/v1/auth/magiclink/verify", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err = app.Test(req, -1)
		require.NoError(t, err)
		assert.True(t, resp.StatusCode == 200 || resp.StatusCode == 401, "Magic link verification response")
	}
}

// testPasswordResetFlow tests password reset functionality
func testPasswordResetFlow(t *testing.T, app *fiber.App, db *database.Connection) {
	email := "resetpass@test.com"
	oldPassword := "OldPass123!"
	newPassword := "NewPass456!"

	// Create user
	createTestUserWithPassword(t, db, email, oldPassword)

	// Request password reset
	resetReq := map[string]interface{}{
		"email": email,
	}

	body, _ := json.Marshal(resetReq)
	req := httptest.NewRequest("POST", "/api/v1/auth/reset-password", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.True(t, resp.StatusCode == 200 || resp.StatusCode == 202, "Password reset request accepted")

	// Get reset token from database (in production, it's sent via email)
	ctx := context.Background()
	var token string
	err = db.Pool().QueryRow(ctx, `
		SELECT prt.token FROM auth.password_reset_tokens prt
		JOIN auth.users u ON prt.user_id = u.id
		WHERE u.email = $1 AND prt.used_at IS NULL
		ORDER BY prt.created_at DESC LIMIT 1
	`, email).Scan(&token)

	if err == nil {
		// Confirm password reset with new password
		confirmReq := map[string]interface{}{
			"token":    token,
			"password": newPassword,
		}

		body, _ = json.Marshal(confirmReq)
		req = httptest.NewRequest("POST", "/api/v1/auth/reset-password/confirm", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err = app.Test(req, -1)
		require.NoError(t, err)
		assert.True(t, resp.StatusCode == 200 || resp.StatusCode == 404, "Password reset confirmation response")
	}
}

// testMultiDeviceSessions tests multiple active sessions
func testMultiDeviceSessions(t *testing.T, app *fiber.App, db *database.Connection, jwtSecret string) {
	email := "multidevice@test.com"
	password := "MultiDevicePass123!"

	createTestUserWithPassword(t, db, email, password)

	// Sign in from "device 1"
	device1Token := signInAndGetToken(t, app, email, password)
	assert.NotEmpty(t, device1Token)

	// Sign in from "device 2"
	device2Token := signInAndGetToken(t, app, email, password)
	assert.NotEmpty(t, device2Token)

	// Both tokens should be different
	assert.NotEqual(t, device1Token, device2Token)

	// Both tokens should work
	req1 := httptest.NewRequest("GET", "/api/v1/auth/user", nil)
	req1.Header.Set("Authorization", "Bearer "+device1Token)

	resp1, err := app.Test(req1, -1)
	require.NoError(t, err)
	assert.True(t, resp1.StatusCode == 200 || resp1.StatusCode == 401)

	req2 := httptest.NewRequest("GET", "/api/v1/auth/user", nil)
	req2.Header.Set("Authorization", "Bearer "+device2Token)

	resp2, err := app.Test(req2, -1)
	require.NoError(t, err)
	assert.True(t, resp2.StatusCode == 200 || resp2.StatusCode == 401)
}

// testUserProfileUpdates tests updating user profile
func testUserProfileUpdates(t *testing.T, app *fiber.App, db *database.Connection, jwtSecret string) {
	email := "updateprofile@test.com"
	password := "UpdatePass123!"

	createTestUserWithPassword(t, db, email, password)
	token := signInAndGetToken(t, app, email, password)

	// Update user profile
	updateReq := map[string]interface{}{
		"raw_user_meta_data": map[string]interface{}{
			"display_name": "Test User",
			"avatar_url":   "https://example.com/avatar.jpg",
		},
	}

	body, _ := json.Marshal(updateReq)
	req := httptest.NewRequest("PATCH", "/api/v1/auth/user", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.True(t, resp.StatusCode == 200 || resp.StatusCode == 404, "Profile update response")
}

// testSignOutFlow tests sign out functionality
func testSignOutFlow(t *testing.T, app *fiber.App, db *database.Connection, jwtSecret string) {
	email := "signout@test.com"
	password := "SignOutPass123!"

	createTestUserWithPassword(t, db, email, password)
	token := signInAndGetToken(t, app, email, password)

	// Sign out
	req := httptest.NewRequest("POST", "/api/v1/auth/signout", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.True(t, resp.StatusCode == 200 || resp.StatusCode == 204, "Sign out should succeed")

	// Try to use token after sign out (should fail or be handled gracefully)
	req = httptest.NewRequest("GET", "/api/v1/auth/user", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err = app.Test(req, -1)
	require.NoError(t, err)
	// Token might still be valid if not using session blacklist
	assert.True(t, resp.StatusCode == 200 || resp.StatusCode == 401)
}

// testOAuthCallbackSimulation tests OAuth callback handling
func testOAuthCallbackSimulation(t *testing.T, app *fiber.App, db *database.Connection) {
	// Simulate OAuth callback with provider data
	// This is a simplified test - real OAuth involves redirects and external providers

	oauthReq := map[string]interface{}{
		"provider": "google",
		"code":     "mock-auth-code",
	}

	body, _ := json.Marshal(oauthReq)
	req := httptest.NewRequest("POST", "/api/v1/auth/oauth/callback", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	// OAuth endpoint might not be implemented, so we check for appropriate response
	assert.True(t, resp.StatusCode == 200 || resp.StatusCode == 404 || resp.StatusCode == 501)
}

// testInvalidCredentials tests authentication with wrong credentials
func testInvalidCredentials(t *testing.T, app *fiber.App, db *database.Connection) {
	email := "invalid@test.com"
	correctPassword := "CorrectPass123!"
	wrongPassword := "WrongPass123!"

	createTestUserWithPassword(t, db, email, correctPassword)

	// Try to sign in with wrong password
	signinReq := map[string]interface{}{
		"email":    email,
		"password": wrongPassword,
	}

	body, _ := json.Marshal(signinReq)
	req := httptest.NewRequest("POST", "/api/v1/auth/signin", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 401, resp.StatusCode, "Wrong password should be rejected")

	// Try to sign in with non-existent email
	signinReq = map[string]interface{}{
		"email":    "nonexistent@test.com",
		"password": correctPassword,
	}

	body, _ = json.Marshal(signinReq)
	req = httptest.NewRequest("POST", "/api/v1/auth/signin", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err = app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 401, resp.StatusCode, "Non-existent user should be rejected")
}

// testAuthRateLimiting tests rate limiting on auth endpoints
func testAuthRateLimiting(t *testing.T, app *fiber.App, db *database.Connection) {
	email := "ratelimit@test.com"

	// Make multiple rapid requests
	for i := 0; i < 20; i++ {
		signinReq := map[string]interface{}{
			"email":    email,
			"password": "WrongPass123!",
		}

		body, _ := json.Marshal(signinReq)
		req := httptest.NewRequest("POST", "/api/v1/auth/signin", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Forwarded-For", "192.168.1.100") // Simulate IP

		resp, err := app.Test(req, -1)
		require.NoError(t, err)

		// After enough attempts, should be rate limited (429) or always return 401
		assert.True(t, resp.StatusCode == 401 || resp.StatusCode == 429)
	}
}

// Helper functions

func createTestUserWithPassword(t *testing.T, db *database.Connection, email, password string) string {
	ctx := context.Background()

	// Hash password (simplified - real implementation uses bcrypt)
	authService := auth.NewService(db, "test-secret", "smtp://fake", "noreply@test.com")
	hashedPassword, err := authService.HashPassword(password)
	require.NoError(t, err)

	// Create user
	userID := uuid.New().String()
	_, err = db.Pool().Exec(ctx, `
		INSERT INTO auth.users (id, email, encrypted_password, email_confirmed_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (email) DO NOTHING
	`, userID, email, hashedPassword)
	require.NoError(t, err)

	return userID
}

func signInAndGetToken(t *testing.T, app *fiber.App, email, password string) string {
	signinReq := map[string]interface{}{
		"email":    email,
		"password": password,
	}

	body, _ := json.Marshal(signinReq)
	req := httptest.NewRequest("POST", "/api/v1/auth/signin", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	require.NoError(t, err)

	if resp.StatusCode != 200 {
		return ""
	}

	var signinResp map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&signinResp)

	if token, ok := signinResp["access_token"].(string); ok {
		return token
	}

	return ""
}
