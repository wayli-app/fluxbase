package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/fluxbase-eu/fluxbase/internal/auth"
	"github.com/fluxbase-eu/fluxbase/internal/config"
	"github.com/fluxbase-eu/fluxbase/internal/database"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupAuthTestServer creates a test server with auth routes
func setupAuthTestServer(t *testing.T) (*fiber.App, *auth.Service, *database.Connection) {
	t.Helper()

	// Skip integration tests when running with -short flag
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Get database configuration from environment variables
	dbHost := os.Getenv("FLUXBASE_DATABASE_HOST")
	if dbHost == "" {
		dbHost = os.Getenv("DB_HOST")
	}
	if dbHost == "" {
		dbHost = "localhost"
	}

	dbUser := os.Getenv("FLUXBASE_DATABASE_USER")
	if dbUser == "" {
		dbUser = "fluxbase_app"
	}

	dbPassword := os.Getenv("FLUXBASE_DATABASE_PASSWORD")
	if dbPassword == "" {
		dbPassword = "fluxbase_app_password"
	}

	dbDatabase := os.Getenv("FLUXBASE_DATABASE_DATABASE")
	if dbDatabase == "" {
		dbDatabase = "fluxbase_test"
	}

	// Create database configuration for testing
	dbConfig := config.DatabaseConfig{
		Host:            dbHost,
		Port:            5432,
		User:            dbUser,
		Password:        dbPassword,
		Database:        dbDatabase,
		SSLMode:         "disable",
		MaxConnections:  5,
		MinConnections:  1,
		MaxConnLifetime: 5 * time.Minute,
		MaxConnIdleTime: 5 * time.Minute,
		HealthCheck:     30 * time.Second,
	}

	// Connect to database
	db, err := database.NewConnection(dbConfig)
	require.NoError(t, err)

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = db.Health(ctx)
	require.NoError(t, err)

	// Create auth configuration
	authConfig := &config.AuthConfig{
		JWTSecret:           "test-secret-key-for-testing-only",
		JWTExpiry:           15 * time.Minute,
		RefreshExpiry:       7 * 24 * time.Hour,
		MagicLinkExpiry:     15 * time.Minute,
		PasswordResetExpiry: 1 * time.Hour,
		PasswordMinLen:      8,
		BcryptCost:          4, // Lower cost for faster tests
		SignupEnabled:       true,
	}

	// Create a no-op email service for testing
	emailService := &auth.NoOpOTPSender{}

	// Create auth service
	authService := auth.NewService(db, authConfig, emailService, "http://localhost:3000")

	// Create Fiber app
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return c.Status(code).JSON(fiber.Map{
				"error": err.Error(),
			})
		},
	})

	// Create auth handler (no captcha service in tests)
	authHandler := NewAuthHandler(db.Pool(), authService, nil, "http://localhost:3000")

	// Setup auth routes
	auth := app.Group("/api/v1/auth")
	rateLimiters := map[string]fiber.Handler{
		"signup":         func(c *fiber.Ctx) error { return c.Next() }, // No-op for tests
		"login":          func(c *fiber.Ctx) error { return c.Next() },
		"refresh":        func(c *fiber.Ctx) error { return c.Next() },
		"magiclink":      func(c *fiber.Ctx) error { return c.Next() },
		"password_reset": func(c *fiber.Ctx) error { return c.Next() },
		"otp":            func(c *fiber.Ctx) error { return c.Next() },
	}
	authHandler.RegisterRoutes(auth, rateLimiters)

	return app, authService, db
}

func TestOTPFlow(t *testing.T) {
	app, _, db := setupAuthTestServer(t)
	defer db.Close()

	// Create a test user first
	ctx := context.Background()
	testEmail := "otp-test@example.com"

	// Sign up a user
	signupReq := map[string]interface{}{
		"email":    testEmail,
		"password": "testpassword123",
	}
	body, _ := json.Marshal(signupReq)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/signup", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, 30000)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusCreated, resp.StatusCode)

	t.Run("SendOTP", func(t *testing.T) {
		// Send OTP
		otpReq := map[string]interface{}{
			"email": testEmail,
		}
		body, _ := json.Marshal(otpReq)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/otp/signin", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req, 30000)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("VerifyOTP", func(t *testing.T) {
		// First, generate an OTP code directly using the repository
		otpRepo := auth.NewOTPRepository(db)
		otpCode, err := otpRepo.Create(ctx, &testEmail, nil, "email", "signin", 10*time.Minute)
		require.NoError(t, err)

		// Verify OTP
		verifyReq := map[string]interface{}{
			"email": testEmail,
			"token": otpCode.Code,
			"type":  "email",
		}
		body, _ := json.Marshal(verifyReq)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/otp/verify", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req, 30000)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Parse response to check for tokens
		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		assert.NotNil(t, result["access_token"])
		assert.NotNil(t, result["refresh_token"])
	})

	t.Run("ResendOTP", func(t *testing.T) {
		// Resend OTP
		resendReq := map[string]interface{}{
			"email": testEmail,
			"type":  "email",
		}
		body, _ := json.Marshal(resendReq)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/otp/resend", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req, 30000)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	// Cleanup
	userRepo := auth.NewUserRepository(db)
	user, _ := userRepo.GetByEmail(ctx, testEmail)
	if user != nil {
		userRepo.Delete(ctx, user.ID)
	}
}

func TestReauthenticate(t *testing.T) {
	app, _, db := setupAuthTestServer(t)
	defer db.Close()

	ctx := context.Background()
	testEmail := "reauth-test@example.com"
	testPassword := "testpassword123"

	// Create and sign in a user
	signupReq := map[string]interface{}{
		"email":    testEmail,
		"password": testPassword,
	}
	body, _ := json.Marshal(signupReq)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/signup", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, 30000)
	require.NoError(t, err)
	resp.Body.Close()

	// Sign in to get token
	signinReq := map[string]interface{}{
		"email":    testEmail,
		"password": testPassword,
	}
	body, _ = json.Marshal(signinReq)

	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/signin", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err = app.Test(req, 30000)
	require.NoError(t, err)

	var signinResult map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&signinResult)
	resp.Body.Close()

	accessToken := signinResult["access_token"].(string)

	t.Run("Reauthenticate with valid token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/reauthenticate", nil)
		req.Header.Set("Authorization", "Bearer "+accessToken)
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req, 30000)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		assert.NotNil(t, result["nonce"])
	})

	t.Run("Reauthenticate without token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/reauthenticate", nil)
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req, 30000)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	// Cleanup
	userRepo := auth.NewUserRepository(db)
	user, _ := userRepo.GetByEmail(ctx, testEmail)
	if user != nil {
		userRepo.Delete(ctx, user.ID)
	}
}

func TestIdentityRoutes(t *testing.T) {
	app, _, db := setupAuthTestServer(t)
	defer db.Close()

	ctx := context.Background()
	testEmail := "identity-test@example.com"
	testPassword := "testpassword123"

	// Create and sign in a user
	signupReq := map[string]interface{}{
		"email":    testEmail,
		"password": testPassword,
	}
	body, _ := json.Marshal(signupReq)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/signup", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, 30000)
	require.NoError(t, err)
	resp.Body.Close()

	// Sign in to get token
	signinReq := map[string]interface{}{
		"email":    testEmail,
		"password": testPassword,
	}
	body, _ = json.Marshal(signinReq)

	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/signin", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err = app.Test(req, 30000)
	require.NoError(t, err)

	var signinResult map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&signinResult)
	resp.Body.Close()

	accessToken := signinResult["access_token"].(string)

	t.Run("GetUserIdentities", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/user/identities", nil)
		req.Header.Set("Authorization", "Bearer "+accessToken)
		resp, err := app.Test(req, 30000)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		assert.NotNil(t, result["identities"])
	})

	t.Run("LinkIdentity", func(t *testing.T) {
		linkReq := map[string]interface{}{
			"provider": "google",
		}
		body, _ := json.Marshal(linkReq)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/user/identities", bytes.NewBuffer(body))
		req.Header.Set("Authorization", "Bearer "+accessToken)
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req, 30000)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Should return OAuth URL
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		assert.NotNil(t, result["url"])
	})

	t.Run("GetUserIdentities without auth", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/user/identities", nil)
		resp, err := app.Test(req, 30000)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	// Cleanup
	userRepo := auth.NewUserRepository(db)
	user, _ := userRepo.GetByEmail(ctx, testEmail)
	if user != nil {
		userRepo.Delete(ctx, user.ID)
	}
}
