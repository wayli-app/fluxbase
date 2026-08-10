//go:build integration
// +build integration

package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nimbleflux/fluxbase/internal/auth"
	"github.com/nimbleflux/fluxbase/internal/config"
	"github.com/nimbleflux/fluxbase/internal/database"
)

// setupDashboardAuthTestServer creates a test server with dashboard auth routes
func setupDashboardAuthTestServer(t *testing.T) (*fiber.App, *DashboardAuthHandler, *database.Connection) {
	t.Helper()

	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

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

	dbDatabase := os.Getenv("FLUXBASE_TEST_DATABASE")
	if dbDatabase == "" {
		dbDatabase = "fluxbase_test"
	}

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

	db, err := database.NewConnection(dbConfig)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = db.Health(ctx)
	require.NoError(t, err)

	jwtManager, err := auth.NewJWTManager("test-dashboard-jwt-secret-at-least-32!", 15*time.Minute, 7*24*time.Hour)
	require.NoError(t, err)
	dashboardAuth := auth.NewDashboardAuthService(db, jwtManager, "FluxbaseTest")

	app := fiber.New(fiber.Config{
		ErrorHandler: func(c fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			var e *fiber.Error
			if errors.As(err, &e) {
				code = e.Code
			}
			return c.Status(code).JSON(fiber.Map{
				"error": err.Error(),
			})
		},
	})

	handler := NewDashboardAuthHandler(dashboardAuth, jwtManager, db, nil, nil, "http://localhost:3000", nil, nil)

	dashboard := app.Group("/dashboard/auth")
	dashboard.Post("/login", handler.Login)
	dashboard.Get("/sso/providers", handler.GetSSOProviders)

	return app, handler, db
}

func TestDashboardPasswordLoginDisabled(t *testing.T) {
	app, _, db := setupDashboardAuthTestServer(t)
	defer db.Close()

	ctx := context.Background()

	// Create a test admin user
	testEmail := "sso-test-admin@example.com"
	testPassword := "testpassword123"

	// Clean up any existing user first
	_, err := db.Exec(ctx, "DELETE FROM platform.users WHERE email = $1", testEmail)
	require.NoError(t, err)

	// Create admin user
	_, err = db.Exec(ctx, `
		INSERT INTO platform.users (id, email, password_hash, role, email_verified, email_verified_at, is_active, created_at, updated_at)
		VALUES (gen_random_uuid(), $1, crypt($2, gen_salt('bf', 4)), 'instance_admin', true, now(), true, now(), now())
	`, testEmail, testPassword)
	require.NoError(t, err)

	t.Run("LoginAllowedWhenPasswordLoginEnabled", func(t *testing.T) {
		// Ensure password login is NOT disabled
		_, err := db.Exec(ctx, `DELETE FROM app.settings WHERE key = 'disable_dashboard_password_login' AND category = 'auth'`)
		require.NoError(t, err)

		loginReq := map[string]interface{}{
			"email":    testEmail,
			"password": testPassword,
		}
		body, _ := json.Marshal(loginReq)

		req := httptest.NewRequest(http.MethodPost, "/dashboard/auth/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req, fiber.TestConfig{Timeout: 30 * time.Second})
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("LoginBlockedWhenPasswordLoginDisabled", func(t *testing.T) {
		// Disable password login in settings
		// Note: Using DELETE + INSERT instead of ON CONFLICT because the table
		// may not have a unique constraint on (key, category)
		_, err := db.Exec(ctx, `DELETE FROM app.settings WHERE key = 'disable_dashboard_password_login'`)
		require.NoError(t, err)
		_, err = db.Exec(ctx, `
			INSERT INTO app.settings (key, category, value, description)
			VALUES ('disable_dashboard_password_login', 'auth', 'true', 'Disable password login for dashboard')
		`)
		require.NoError(t, err)

		loginReq := map[string]interface{}{
			"email":    testEmail,
			"password": testPassword,
		}
		body, _ := json.Marshal(loginReq)

		req := httptest.NewRequest(http.MethodPost, "/dashboard/auth/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req, fiber.TestConfig{Timeout: 30 * time.Second})
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusForbidden, resp.StatusCode)

		var result map[string]interface{}
		_ = json.NewDecoder(resp.Body).Decode(&result)
		assert.Contains(t, result["error"], "Password login is disabled")
	})

	t.Run("EnvironmentVariableOverridesDisabledLogin", func(t *testing.T) {
		// Ensure password login is disabled in DB
		_, err := db.Exec(ctx, `DELETE FROM app.settings WHERE key = 'disable_dashboard_password_login'`)
		require.NoError(t, err)
		_, err = db.Exec(ctx, `
			INSERT INTO app.settings (key, category, value, description)
			VALUES ('disable_dashboard_password_login', 'auth', 'true', 'Disable password login for dashboard')
		`)
		require.NoError(t, err)

		// Set environment variable to force password login
		t.Setenv("FLUXBASE_DASHBOARD_FORCE_PASSWORD_LOGIN", "true")

		loginReq := map[string]interface{}{
			"email":    testEmail,
			"password": testPassword,
		}
		body, _ := json.Marshal(loginReq)

		req := httptest.NewRequest(http.MethodPost, "/dashboard/auth/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req, fiber.TestConfig{Timeout: 30 * time.Second})
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		// Should succeed because env var overrides
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	// Cleanup
	_, _ = db.Exec(ctx, "DELETE FROM platform.users WHERE email = $1", testEmail)
	_, _ = db.Exec(ctx, `DELETE FROM app.settings WHERE key = 'disable_dashboard_password_login' AND category = 'auth'`)
}

func TestGetSSOProvidersEndpoint(t *testing.T) {
	app, _, db := setupDashboardAuthTestServer(t)
	defer db.Close()

	ctx := context.Background()

	t.Run("ReturnsPasswordLoginDisabledStatus", func(t *testing.T) {
		// First test: password login NOT disabled
		_, err := db.Exec(ctx, `DELETE FROM app.settings WHERE key = 'disable_dashboard_password_login' AND category = 'auth'`)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/dashboard/auth/sso/providers", nil)
		resp, err := app.Test(req, fiber.TestConfig{Timeout: 30 * time.Second})
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		_ = json.NewDecoder(resp.Body).Decode(&result)

		assert.NotNil(t, result["providers"])
		assert.Equal(t, false, result["password_login_disabled"])
	})

	t.Run("ReturnsPasswordLoginDisabledTrue", func(t *testing.T) {
		// Enable password login disabled
		_, err := db.Exec(ctx, `DELETE FROM app.settings WHERE key = 'disable_dashboard_password_login'`)
		require.NoError(t, err)
		_, err = db.Exec(ctx, `
			INSERT INTO app.settings (key, category, value, description)
			VALUES ('disable_dashboard_password_login', 'auth', 'true', 'Disable password login for dashboard')
		`)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/dashboard/auth/sso/providers", nil)
		resp, err := app.Test(req, fiber.TestConfig{Timeout: 30 * time.Second})
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		_ = json.NewDecoder(resp.Body).Decode(&result)

		assert.NotNil(t, result["providers"])
		assert.Equal(t, true, result["password_login_disabled"])
	})

	// Cleanup
	_, _ = db.Exec(ctx, `DELETE FROM app.settings WHERE key = 'disable_dashboard_password_login' AND category = 'auth'`)
}
