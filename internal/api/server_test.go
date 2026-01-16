package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// NormalizePaginationParams Tests
// =============================================================================

func TestNormalizePaginationParams(t *testing.T) {
	const defaultLimit = 25
	const maxLimit = 100

	tests := []struct {
		name           string
		inputLimit     int
		inputOffset    int
		expectedLimit  int
		expectedOffset int
	}{
		{
			name:           "valid limit and offset",
			inputLimit:     50,
			inputOffset:    10,
			expectedLimit:  50,
			expectedOffset: 10,
		},
		{
			name:           "zero limit uses default",
			inputLimit:     0,
			inputOffset:    0,
			expectedLimit:  defaultLimit,
			expectedOffset: 0,
		},
		{
			name:           "negative limit uses default",
			inputLimit:     -1,
			inputOffset:    0,
			expectedLimit:  defaultLimit,
			expectedOffset: 0,
		},
		{
			name:           "limit exceeds max uses default",
			inputLimit:     150,
			inputOffset:    0,
			expectedLimit:  defaultLimit,
			expectedOffset: 0,
		},
		{
			name:           "limit equals max is valid",
			inputLimit:     100,
			inputOffset:    0,
			expectedLimit:  100,
			expectedOffset: 0,
		},
		{
			name:           "negative offset becomes zero",
			inputLimit:     25,
			inputOffset:    -10,
			expectedLimit:  25,
			expectedOffset: 0,
		},
		{
			name:           "both invalid",
			inputLimit:     -5,
			inputOffset:    -5,
			expectedLimit:  defaultLimit,
			expectedOffset: 0,
		},
		{
			name:           "minimum valid values",
			inputLimit:     1,
			inputOffset:    0,
			expectedLimit:  1,
			expectedOffset: 0,
		},
		{
			name:           "large valid offset",
			inputLimit:     25,
			inputOffset:    1000,
			expectedLimit:  25,
			expectedOffset: 1000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limit, offset := NormalizePaginationParams(tt.inputLimit, tt.inputOffset, defaultLimit, maxLimit)
			assert.Equal(t, tt.expectedLimit, limit)
			assert.Equal(t, tt.expectedOffset, offset)
		})
	}
}

func TestNormalizePaginationParams_DifferentDefaults(t *testing.T) {
	t.Run("different default limit", func(t *testing.T) {
		limit, offset := NormalizePaginationParams(0, 0, 10, 50)
		assert.Equal(t, 10, limit)
		assert.Equal(t, 0, offset)
	})

	t.Run("different max limit", func(t *testing.T) {
		limit, offset := NormalizePaginationParams(200, 0, 25, 50)
		assert.Equal(t, 25, limit)
		assert.Equal(t, 0, offset)
	})

	t.Run("small max limit", func(t *testing.T) {
		limit, offset := NormalizePaginationParams(15, 0, 5, 10)
		assert.Equal(t, 5, limit)
		assert.Equal(t, 0, offset)
	})
}

// =============================================================================
// customErrorHandler Tests
// =============================================================================

func TestCustomErrorHandler(t *testing.T) {
	tests := []struct {
		name          string
		err           error
		expectedCode  int
		expectedError string
	}{
		{
			name:          "generic error returns 500",
			err:           errors.New("something went wrong"),
			expectedCode:  500,
			expectedError: "Internal Server Error",
		},
		{
			name:          "fiber 400 error",
			err:           fiber.NewError(fiber.StatusBadRequest, "Invalid request"),
			expectedCode:  400,
			expectedError: "Invalid request",
		},
		{
			name:          "fiber 401 error",
			err:           fiber.NewError(fiber.StatusUnauthorized, "Unauthorized"),
			expectedCode:  401,
			expectedError: "Unauthorized",
		},
		{
			name:          "fiber 403 error",
			err:           fiber.NewError(fiber.StatusForbidden, "Forbidden"),
			expectedCode:  403,
			expectedError: "Forbidden",
		},
		{
			name:          "fiber 404 error",
			err:           fiber.NewError(fiber.StatusNotFound, "Not found"),
			expectedCode:  404,
			expectedError: "Not found",
		},
		{
			name:          "fiber 429 error",
			err:           fiber.NewError(fiber.StatusTooManyRequests, "Rate limit exceeded"),
			expectedCode:  429,
			expectedError: "Rate limit exceeded",
		},
		{
			name:          "fiber 502 error",
			err:           fiber.NewError(fiber.StatusBadGateway, "Bad gateway"),
			expectedCode:  502,
			expectedError: "Bad gateway",
		},
		{
			name:          "fiber 503 error",
			err:           fiber.NewError(fiber.StatusServiceUnavailable, "Service unavailable"),
			expectedCode:  503,
			expectedError: "Service unavailable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New(fiber.Config{
				ErrorHandler: customErrorHandler,
			})

			app.Get("/test", func(c *fiber.Ctx) error {
				return tt.err
			})

			req := httptest.NewRequest("GET", "/test", nil)
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()

			assert.Equal(t, tt.expectedCode, resp.StatusCode)

			body, _ := io.ReadAll(resp.Body)
			var result map[string]interface{}
			err = json.Unmarshal(body, &result)
			require.NoError(t, err)

			assert.Equal(t, tt.expectedError, result["error"])
			assert.Equal(t, float64(tt.expectedCode), result["code"])
		})
	}
}

// =============================================================================
// Admin Role Checking Pattern Tests
// =============================================================================

func TestAdminRoleChecking(t *testing.T) {
	tests := []struct {
		name    string
		role    string
		isAdmin bool
	}{
		{"admin role", "admin", true},
		{"dashboard_admin role", "dashboard_admin", true},
		{"service_role role", "service_role", true},
		{"authenticated role", "authenticated", false},
		{"anon role", "anon", false},
		{"empty role", "", false},
		{"unknown role", "unknown", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isAdmin := tt.role == "admin" || tt.role == "dashboard_admin" || tt.role == "service_role"
			assert.Equal(t, tt.isAdmin, isAdmin)
		})
	}
}

// Test the role checking in a Fiber context
func TestAdminRoleCheckingInFiberContext(t *testing.T) {
	tests := []struct {
		name           string
		role           interface{}
		expectedStatus int
	}{
		{"admin access granted", "admin", 200},
		{"dashboard_admin access granted", "dashboard_admin", 200},
		{"service_role access granted", "service_role", 200},
		{"authenticated denied", "authenticated", 403},
		{"anon denied", "anon", 403},
		{"nil role denied", nil, 403},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			app.Use(func(c *fiber.Ctx) error {
				if tt.role != nil {
					c.Locals("user_role", tt.role)
				}
				return c.Next()
			})
			app.Get("/admin-only", func(c *fiber.Ctx) error {
				role, _ := c.Locals("user_role").(string)
				if role != "admin" && role != "dashboard_admin" && role != "service_role" {
					return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
						"error": "Admin access required",
					})
				}
				return c.JSON(fiber.Map{"message": "success"})
			})

			req := httptest.NewRequest("GET", "/admin-only", nil)
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
		})
	}
}

// =============================================================================
// Fiber App Configuration Tests
// =============================================================================

func TestFiberAppConfiguration(t *testing.T) {
	t.Run("default error handler returns JSON", func(t *testing.T) {
		app := fiber.New(fiber.Config{
			ErrorHandler: customErrorHandler,
		})

		app.Get("/error", func(c *fiber.Ctx) error {
			return errors.New("test error")
		})

		req := httptest.NewRequest("GET", "/error", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		contentType := resp.Header.Get("Content-Type")
		assert.Contains(t, contentType, "application/json")
	})
}

// =============================================================================
// Health Check Response Tests
// =============================================================================

func TestHealthCheckResponseFormat(t *testing.T) {
	t.Run("healthy response format", func(t *testing.T) {
		response := fiber.Map{
			"status": "ok",
			"services": fiber.Map{
				"database": true,
				"realtime": true,
			},
		}

		assert.Equal(t, "ok", response["status"])
		services := response["services"].(fiber.Map)
		assert.Equal(t, true, services["database"])
		assert.Equal(t, true, services["realtime"])
	})

	t.Run("degraded response format", func(t *testing.T) {
		response := fiber.Map{
			"status": "degraded",
			"services": fiber.Map{
				"database": false,
				"realtime": true,
			},
		}

		assert.Equal(t, "degraded", response["status"])
		services := response["services"].(fiber.Map)
		assert.Equal(t, false, services["database"])
	})
}

// =============================================================================
// Query Parameter Parsing Tests
// =============================================================================

func TestSchemaQueryParsing(t *testing.T) {
	app := fiber.New()

	var capturedSchema string
	app.Get("/tables", func(c *fiber.Ctx) error {
		capturedSchema = c.Query("schema")
		return c.SendStatus(200)
	})

	tests := []struct {
		name           string
		queryParam     string
		expectedSchema string
	}{
		{"no schema param", "/tables", ""},
		{"public schema", "/tables?schema=public", "public"},
		{"auth schema", "/tables?schema=auth", "auth"},
		{"storage schema", "/tables?schema=storage", "storage"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.queryParam, nil)
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()

			assert.Equal(t, tt.expectedSchema, capturedSchema)
		})
	}
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkNormalizePaginationParams(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = NormalizePaginationParams(50, 10, 25, 100)
	}
}

func BenchmarkNormalizePaginationParams_Invalid(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = NormalizePaginationParams(-1, -1, 25, 100)
	}
}

func BenchmarkAdminRoleCheck(b *testing.B) {
	role := "authenticated"

	for i := 0; i < b.N; i++ {
		_ = (role == "admin" || role == "dashboard_admin" || role == "service_role")
	}
}

func BenchmarkCustomErrorHandler(b *testing.B) {
	app := fiber.New(fiber.Config{
		ErrorHandler: customErrorHandler,
	})

	app.Get("/test", func(c *fiber.Ctx) error {
		return fiber.NewError(fiber.StatusBadRequest, "Test error")
	})

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		resp, _ := app.Test(req)
		if resp != nil {
			_ = resp.Body.Close()
		}
	}
}
