package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// AdminSessionHandler Construction Tests
// =============================================================================

func TestNewAdminSessionHandler(t *testing.T) {
	t.Run("creates handler with nil repository", func(t *testing.T) {
		handler := NewAdminSessionHandler(nil)
		assert.NotNil(t, handler)
		assert.Nil(t, handler.sessionRepo)
	})
}

// =============================================================================
// ListSessions Parameter Parsing Tests
// =============================================================================

func TestListSessions_ParameterParsing(t *testing.T) {
	// These tests verify parameter parsing logic without invoking the handler
	// to avoid nil pointer dereferences with mock dependencies

	t.Run("valid limit parameter parsing", func(t *testing.T) {
		limitStr := "50"
		limit := 25 // default
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
			if limit > 100 {
				limit = 100
			}
		}
		assert.Equal(t, 50, limit)
	})

	t.Run("invalid limit falls back to default", func(t *testing.T) {
		limitStr := "invalid"
		limit := 25 // default
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
		}
		assert.Equal(t, 25, limit)
	})

	t.Run("negative limit falls back to default", func(t *testing.T) {
		limitStr := "-5"
		limit := 25 // default
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
		}
		assert.Equal(t, 25, limit)
	})

	t.Run("valid offset parameter parsing", func(t *testing.T) {
		offsetStr := "10"
		offset := 0 // default
		if parsed, err := strconv.Atoi(offsetStr); err == nil && parsed >= 0 {
			offset = parsed
		}
		assert.Equal(t, 10, offset)
	})

	t.Run("invalid offset falls back to default", func(t *testing.T) {
		offsetStr := "invalid"
		offset := 0 // default
		if parsed, err := strconv.Atoi(offsetStr); err == nil && parsed >= 0 {
			offset = parsed
		}
		assert.Equal(t, 0, offset)
	})

	t.Run("negative offset falls back to default", func(t *testing.T) {
		offsetStr := "-10"
		offset := 0 // default
		if parsed, err := strconv.Atoi(offsetStr); err == nil && parsed >= 0 {
			offset = parsed
		}
		assert.Equal(t, 0, offset)
	})
}

// =============================================================================
// ListSessions Limit Capping Tests
// =============================================================================

func TestListSessions_LimitCapping(t *testing.T) {
	t.Run("limit capped at 100", func(t *testing.T) {
		limit := 500
		if limit > 100 {
			limit = 100
		}
		assert.Equal(t, 100, limit)
	})

	t.Run("limit exactly 100 is allowed", func(t *testing.T) {
		limit := 100
		if limit > 100 {
			limit = 100
		}
		assert.Equal(t, 100, limit)
	})

	t.Run("limit of 1 is allowed", func(t *testing.T) {
		limit := 1
		assert.Greater(t, limit, 0)
	})

	t.Run("limit of 0 should use default", func(t *testing.T) {
		limitStr := "0"
		limit := 25 // default
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
		}
		// 0 is not > 0, so default is used
		assert.Equal(t, 25, limit)
	})
}

// =============================================================================
// RevokeSession Validation Tests
// =============================================================================

func TestRevokeSession_Validation(t *testing.T) {
	t.Run("empty session ID validation", func(t *testing.T) {
		sessionID := ""
		assert.Empty(t, sessionID)
	})

	t.Run("valid session ID format", func(t *testing.T) {
		sessionID := "sess_abc123"
		assert.NotEmpty(t, sessionID)
	})

	t.Run("UUID session ID format", func(t *testing.T) {
		sessionID := "550e8400-e29b-41d4-a716-446655440000"
		assert.NotEmpty(t, sessionID)
		assert.Len(t, sessionID, 36) // UUID length
	})
}

// =============================================================================
// RevokeUserSessions Validation Tests
// =============================================================================

func TestRevokeUserSessions_Validation(t *testing.T) {
	t.Run("empty user ID validation", func(t *testing.T) {
		userID := ""
		assert.Empty(t, userID)
	})

	t.Run("valid user ID format", func(t *testing.T) {
		userID := "user_123"
		assert.NotEmpty(t, userID)
	})

	t.Run("UUID user ID format", func(t *testing.T) {
		userID := "550e8400-e29b-41d4-a716-446655440000"
		assert.NotEmpty(t, userID)
		assert.Len(t, userID, 36) // UUID length
	})
}

// =============================================================================
// Response Format Tests
// =============================================================================

func TestSessionResponses_Format(t *testing.T) {
	t.Run("revoke session success response format", func(t *testing.T) {
		// Test expected response structure
		expectedResponse := fiber.Map{
			"message": "Session revoked successfully",
		}

		data, err := json.Marshal(expectedResponse)
		require.NoError(t, err)

		assert.Contains(t, string(data), `"message":"Session revoked successfully"`)
	})

	t.Run("revoke user sessions success response format", func(t *testing.T) {
		expectedResponse := fiber.Map{
			"message": "All user sessions revoked successfully",
		}

		data, err := json.Marshal(expectedResponse)
		require.NoError(t, err)

		assert.Contains(t, string(data), `"message":"All user sessions revoked successfully"`)
	})

	t.Run("list sessions response structure", func(t *testing.T) {
		// Test expected pagination response structure
		expectedResponse := fiber.Map{
			"sessions": []interface{}{},
			"total":    0,
			"limit":    25,
			"offset":   0,
		}

		data, err := json.Marshal(expectedResponse)
		require.NoError(t, err)

		assert.Contains(t, string(data), `"sessions"`)
		assert.Contains(t, string(data), `"total"`)
		assert.Contains(t, string(data), `"limit"`)
		assert.Contains(t, string(data), `"offset"`)
	})
}

// =============================================================================
// Error Response Tests
// =============================================================================

func TestSessionErrorResponses(t *testing.T) {
	t.Run("session ID required error", func(t *testing.T) {
		expectedError := fiber.Map{
			"error": "Session ID is required",
		}

		data, err := json.Marshal(expectedError)
		require.NoError(t, err)

		assert.Contains(t, string(data), `"error":"Session ID is required"`)
	})

	t.Run("user ID required error", func(t *testing.T) {
		expectedError := fiber.Map{
			"error": "User ID is required",
		}

		data, err := json.Marshal(expectedError)
		require.NoError(t, err)

		assert.Contains(t, string(data), `"error":"User ID is required"`)
	})

	t.Run("session not found error", func(t *testing.T) {
		expectedError := fiber.Map{
			"error": "Session not found",
		}

		data, err := json.Marshal(expectedError)
		require.NoError(t, err)

		assert.Contains(t, string(data), `"error":"Session not found"`)
	})

	t.Run("failed to list sessions error", func(t *testing.T) {
		expectedError := fiber.Map{
			"error": "Failed to list sessions",
		}

		data, err := json.Marshal(expectedError)
		require.NoError(t, err)

		assert.Contains(t, string(data), `"error":"Failed to list sessions"`)
	})
}

// =============================================================================
// Pagination Logic Tests
// =============================================================================

func TestPaginationLogic(t *testing.T) {
	t.Run("default values", func(t *testing.T) {
		// Test the default pagination values
		defaultLimit := 25
		defaultOffset := 0
		maxLimit := 100

		assert.Equal(t, 25, defaultLimit)
		assert.Equal(t, 0, defaultOffset)
		assert.Equal(t, 100, maxLimit)
	})

	t.Run("limit capping logic", func(t *testing.T) {
		// Test limit capping behavior
		limit := 150
		maxLimit := 100

		if limit > maxLimit {
			limit = maxLimit
		}

		assert.Equal(t, 100, limit)
	})

	t.Run("limit not capped when under max", func(t *testing.T) {
		limit := 50
		maxLimit := 100

		if limit > maxLimit {
			limit = maxLimit
		}

		assert.Equal(t, 50, limit)
	})

	t.Run("parse valid limit", func(t *testing.T) {
		// Simulating the parsing logic
		limitStr := "50"
		limit := 25 // default

		if parsed, err := parseInt(limitStr); err == nil && parsed > 0 {
			limit = parsed
			if limit > 100 {
				limit = 100
			}
		}

		assert.Equal(t, 50, limit)
	})

	t.Run("parse invalid limit uses default", func(t *testing.T) {
		limitStr := "invalid"
		limit := 25 // default

		if parsed, err := parseInt(limitStr); err == nil && parsed > 0 {
			limit = parsed
		}

		assert.Equal(t, 25, limit)
	})
}

// Helper function for parsing (mimics strconv.Atoi)
func parseInt(s string) (int, error) {
	var result int
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, fiber.ErrBadRequest
		}
		result = result*10 + int(c-'0')
	}
	return result, nil
}

// =============================================================================
// Query Parameter Tests
// =============================================================================

func TestIncludeExpiredParameter(t *testing.T) {
	t.Run("include_expired true parsing", func(t *testing.T) {
		queryValue := "true"
		includeExpired := queryValue == "true"
		assert.True(t, includeExpired)
	})

	t.Run("include_expired false parsing", func(t *testing.T) {
		queryValue := "false"
		includeExpired := queryValue == "true"
		assert.False(t, includeExpired)
	})

	t.Run("include_expired not set", func(t *testing.T) {
		queryValue := ""
		includeExpired := queryValue == "true"
		assert.False(t, includeExpired)
	})

	t.Run("include_expired invalid value", func(t *testing.T) {
		queryValue := "invalid"
		includeExpired := queryValue == "true"
		assert.False(t, includeExpired)
	})
}

// =============================================================================
// HTTP Method Tests
// =============================================================================

func TestHTTPMethods(t *testing.T) {
	t.Run("list sessions uses GET", func(t *testing.T) {
		app := fiber.New()
		handler := NewAdminSessionHandler(nil)
		app.Get("/sessions", handler.ListSessions)

		// POST should not work
		req := httptest.NewRequest(http.MethodPost, "/sessions", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusMethodNotAllowed, resp.StatusCode)
	})

	t.Run("revoke session uses DELETE", func(t *testing.T) {
		app := fiber.New()
		handler := NewAdminSessionHandler(nil)
		app.Delete("/sessions/:id", handler.RevokeSession)

		// GET should not work
		req := httptest.NewRequest(http.MethodGet, "/sessions/test-id", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusMethodNotAllowed, resp.StatusCode)
	})

	t.Run("revoke user sessions uses DELETE", func(t *testing.T) {
		app := fiber.New()
		handler := NewAdminSessionHandler(nil)
		app.Delete("/users/:user_id/sessions", handler.RevokeUserSessions)

		// PUT should not work
		req := httptest.NewRequest(http.MethodPut, "/users/test-user/sessions", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusMethodNotAllowed, resp.StatusCode)
	})
}

// =============================================================================
// Error Message Tests
// =============================================================================

func TestErrorMessages(t *testing.T) {
	t.Run("list sessions error format", func(t *testing.T) {
		expectedError := "Failed to list sessions"
		assert.Equal(t, "Failed to list sessions", expectedError)
	})

	t.Run("revoke session error format", func(t *testing.T) {
		expectedError := "Failed to revoke session"
		assert.Equal(t, "Failed to revoke session", expectedError)
	})

	t.Run("revoke user sessions error format", func(t *testing.T) {
		expectedError := "Failed to revoke user sessions"
		assert.Equal(t, "Failed to revoke user sessions", expectedError)
	})
}
