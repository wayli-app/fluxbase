package test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wayli-app/fluxbase/internal/api"
	"github.com/wayli-app/fluxbase/internal/config"
	"github.com/wayli-app/fluxbase/internal/database"
)

// TestAuthFlow_SignupSigninSignout tests the complete auth flow
func TestAuthFlow_SignupSigninSignout(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Setup
	cfg, db := setupTestEnvironment(t)
	defer db.Close()

	// Clean up test data
	cleanupTestUsers(t, db, "test@example.com")

	// Create server
	server := api.NewServer(cfg, db)
	app := server.App()

	// Test data
	email := "test@example.com"
	password := "testpassword123"

	// Step 1: Sign up
	t.Run("Signup", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"email":    email,
			"password": password,
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/auth/signup", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		require.NoError(t, err)

		assert.Equal(t, 201, resp.StatusCode)

		// Parse response
		var signupResp map[string]interface{}
		respBody, _ := io.ReadAll(resp.Body)
		err = json.Unmarshal(respBody, &signupResp)
		require.NoError(t, err)

		// Verify response has tokens
		assert.NotEmpty(t, signupResp["access_token"])
		assert.NotEmpty(t, signupResp["refresh_token"])
		assert.NotEmpty(t, signupResp["user"])
	})

	// Step 2: Sign in
	var accessToken string
	var refreshToken string
	t.Run("Signin", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"email":    email,
			"password": password,
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/auth/signin", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		require.NoError(t, err)

		assert.Equal(t, 200, resp.StatusCode)

		// Parse response
		var signinResp map[string]interface{}
		respBody, _ := io.ReadAll(resp.Body)
		err = json.Unmarshal(respBody, &signinResp)
		require.NoError(t, err)

		// Store tokens for next test
		accessToken = signinResp["access_token"].(string)
		refreshToken = signinResp["refresh_token"].(string)

		assert.NotEmpty(t, accessToken)
		assert.NotEmpty(t, refreshToken)
	})

	// Step 3: Get user
	t.Run("GetUser", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/auth/user", nil)
		req.Header.Set("Authorization", "Bearer "+accessToken)

		resp, err := app.Test(req, -1)
		require.NoError(t, err)

		assert.Equal(t, 200, resp.StatusCode)

		// Parse response
		var user map[string]interface{}
		respBody, _ := io.ReadAll(resp.Body)
		err = json.Unmarshal(respBody, &user)
		require.NoError(t, err)

		assert.Equal(t, email, user["email"])
	})

	// Step 4: Refresh token
	t.Run("RefreshToken", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"refresh_token": refreshToken,
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/auth/refresh", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		require.NoError(t, err)

		assert.Equal(t, 200, resp.StatusCode)

		// Parse response
		var refreshResp map[string]interface{}
		respBody, _ := io.ReadAll(resp.Body)
		err = json.Unmarshal(respBody, &refreshResp)
		require.NoError(t, err)

		// New access token should be returned
		assert.NotEmpty(t, refreshResp["access_token"])
		assert.NotEqual(t, accessToken, refreshResp["access_token"])

		// Update access token for next test
		accessToken = refreshResp["access_token"].(string)
	})

	// Step 5: Sign out
	t.Run("Signout", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/auth/signout", nil)
		req.Header.Set("Authorization", "Bearer "+accessToken)

		resp, err := app.Test(req, -1)
		require.NoError(t, err)

		assert.Equal(t, 200, resp.StatusCode)
	})

	// Step 6: Verify token is invalid after signout
	t.Run("GetUser_AfterSignout", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/auth/user", nil)
		req.Header.Set("Authorization", "Bearer "+accessToken)

		resp, err := app.Test(req, -1)
		require.NoError(t, err)

		// Should be unauthorized because session was deleted
		assert.Equal(t, 401, resp.StatusCode)
	})

	// Cleanup
	cleanupTestUsers(t, db, email)
}

// setupTestEnvironment sets up a test configuration and database connection
func setupTestEnvironment(t *testing.T) (*config.Config, *database.Connection) {
	// Load test configuration
	cfg, err := config.Load()
	require.NoError(t, err)

	// Override some settings for testing
	cfg.Debug = true
	cfg.BaseURL = "http://localhost:8080"
	cfg.Email.Enabled = false // Disable email for auth tests

	// Connect to database
	db, err := database.NewConnection(cfg.Database)
	require.NoError(t, err)

	// Run migrations
	err = db.Migrate()
	require.NoError(t, err)

	return cfg, db
}

// cleanupTestUsers removes test users from the database
func cleanupTestUsers(t *testing.T, db *database.Connection, email string) {
	ctx := context.Background()
	query := `DELETE FROM auth.users WHERE email = $1`
	_, err := db.Exec(ctx, query, email)
	if err != nil {
		t.Logf("Warning: failed to cleanup test user: %v", err)
	}
}
