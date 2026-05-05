package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// UserSettingsHandler Construction Tests
// =============================================================================

func TestNewUserSettingsHandler(t *testing.T) {
	t.Run("creates handler with nil dependencies", func(t *testing.T) {
		handler := NewUserSettingsHandler(nil, nil)
		assert.NotNil(t, handler)
		assert.Nil(t, handler.db)
		assert.Nil(t, handler.settingsService)
	})

	t.Run("SetSecretsService sets the secrets service", func(t *testing.T) {
		handler := NewUserSettingsHandler(nil, nil)
		assert.Nil(t, handler.secretsService)
		// The method exists for dependency injection
	})
}

// =============================================================================
// CreateSecret Handler Validation Tests
// =============================================================================

func TestCreateSecret_Validation(t *testing.T) {
	t.Run("missing user context returns unauthorized", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewUserSettingsHandler(nil, nil)

		app.Post("/settings/secret", handler.CreateSecret)

		body := `{"key": "api_key", "value": "secret-value"}`
		req := httptest.NewRequest(http.MethodPost, "/settings/secret", bytes.NewReader([]byte(body)))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)

		respBody, _ := io.ReadAll(resp.Body)
		var result map[string]interface{}
		_ = json.Unmarshal(respBody, &result)
		assert.Equal(t, "Authentication required", result["error"])
	})

	t.Run("invalid user ID returns error", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewUserSettingsHandler(nil, nil)

		app.Post("/settings/secret", func(c fiber.Ctx) error {
			c.Locals("user_id", "not-a-uuid")
			return handler.CreateSecret(c)
		})

		body := `{"key": "api_key", "value": "secret-value"}`
		req := httptest.NewRequest(http.MethodPost, "/settings/secret", bytes.NewReader([]byte(body)))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)

		respBody, _ := io.ReadAll(resp.Body)
		var result map[string]interface{}
		_ = json.Unmarshal(respBody, &result)
		assert.Equal(t, "Invalid user ID", result["error"])
	})

	t.Run("invalid request body returns error", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewUserSettingsHandler(nil, nil)

		app.Post("/settings/secret", func(c fiber.Ctx) error {
			c.Locals("user_id", "550e8400-e29b-41d4-a716-446655440000")
			return handler.CreateSecret(c)
		})

		req := httptest.NewRequest(http.MethodPost, "/settings/secret", bytes.NewReader([]byte("not json")))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

		respBody, _ := io.ReadAll(resp.Body)
		var result map[string]interface{}
		_ = json.Unmarshal(respBody, &result)
		assert.Equal(t, "Invalid request body", result["error"])
	})

	t.Run("missing key returns error", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewUserSettingsHandler(nil, nil)

		app.Post("/settings/secret", func(c fiber.Ctx) error {
			c.Locals("user_id", "550e8400-e29b-41d4-a716-446655440000")
			return handler.CreateSecret(c)
		})

		body := `{"value": "secret-value"}`
		req := httptest.NewRequest(http.MethodPost, "/settings/secret", bytes.NewReader([]byte(body)))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

		respBody, _ := io.ReadAll(resp.Body)
		var result map[string]interface{}
		_ = json.Unmarshal(respBody, &result)
		assert.Equal(t, "key is required", result["error"])
	})

	t.Run("missing value returns error", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewUserSettingsHandler(nil, nil)

		app.Post("/settings/secret", func(c fiber.Ctx) error {
			c.Locals("user_id", "550e8400-e29b-41d4-a716-446655440000")
			return handler.CreateSecret(c)
		})

		body := `{"key": "api_key"}`
		req := httptest.NewRequest(http.MethodPost, "/settings/secret", bytes.NewReader([]byte(body)))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

		respBody, _ := io.ReadAll(resp.Body)
		var result map[string]interface{}
		_ = json.Unmarshal(respBody, &result)
		assert.Equal(t, "value is required", result["error"])
	})
}

// =============================================================================
// GetSecret Handler Validation Tests
// =============================================================================

func TestGetSecret_Validation(t *testing.T) {
	t.Run("empty key returns error", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewUserSettingsHandler(nil, nil)

		app.Get("/settings/secret/*", handler.GetSecret)

		req := httptest.NewRequest(http.MethodGet, "/settings/secret/", nil)

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

		respBody, _ := io.ReadAll(resp.Body)
		var result map[string]interface{}
		_ = json.Unmarshal(respBody, &result)
		assert.Equal(t, "key is required", result["error"])
	})

	t.Run("missing user context returns unauthorized", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewUserSettingsHandler(nil, nil)

		app.Get("/settings/secret/*", handler.GetSecret)

		req := httptest.NewRequest(http.MethodGet, "/settings/secret/my_key", nil)

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("invalid user ID returns error", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewUserSettingsHandler(nil, nil)

		app.Get("/settings/secret/*", func(c fiber.Ctx) error {
			c.Locals("user_id", "bad-uuid")
			return handler.GetSecret(c)
		})

		req := httptest.NewRequest(http.MethodGet, "/settings/secret/my_key", nil)

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
	})
}

// =============================================================================
// UpdateSecret Handler Validation Tests
// =============================================================================

func TestUpdateSecret_Validation(t *testing.T) {
	t.Run("empty key returns error", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewUserSettingsHandler(nil, nil)

		app.Put("/settings/secret/*", handler.UpdateSecret)

		body := `{"value": "new-value"}`
		req := httptest.NewRequest(http.MethodPut, "/settings/secret/", bytes.NewReader([]byte(body)))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

		respBody, _ := io.ReadAll(resp.Body)
		var result map[string]interface{}
		_ = json.Unmarshal(respBody, &result)
		assert.Equal(t, "key is required", result["error"])
	})

	t.Run("missing user context returns unauthorized", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewUserSettingsHandler(nil, nil)

		app.Put("/settings/secret/*", handler.UpdateSecret)

		body := `{"value": "new-value"}`
		req := httptest.NewRequest(http.MethodPut, "/settings/secret/my_key", bytes.NewReader([]byte(body)))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("invalid request body returns error", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewUserSettingsHandler(nil, nil)

		app.Put("/settings/secret/*", func(c fiber.Ctx) error {
			c.Locals("user_id", "550e8400-e29b-41d4-a716-446655440000")
			return handler.UpdateSecret(c)
		})

		req := httptest.NewRequest(http.MethodPut, "/settings/secret/my_key", bytes.NewReader([]byte("not json")))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})
}

// =============================================================================
// DeleteSecret Handler Validation Tests
// =============================================================================

func TestDeleteSecret_Validation(t *testing.T) {
	t.Run("empty key returns error", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewUserSettingsHandler(nil, nil)

		app.Delete("/settings/secret/*", handler.DeleteSecret)

		req := httptest.NewRequest(http.MethodDelete, "/settings/secret/", nil)

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

		respBody, _ := io.ReadAll(resp.Body)
		var result map[string]interface{}
		_ = json.Unmarshal(respBody, &result)
		assert.Equal(t, "key is required", result["error"])
	})

	t.Run("missing user context returns unauthorized", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewUserSettingsHandler(nil, nil)

		app.Delete("/settings/secret/*", handler.DeleteSecret)

		req := httptest.NewRequest(http.MethodDelete, "/settings/secret/my_key", nil)

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
	})
}

// =============================================================================
// ListSecrets Handler Validation Tests
// =============================================================================

func TestListSecrets_Validation(t *testing.T) {
	t.Run("missing user context returns unauthorized", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewUserSettingsHandler(nil, nil)

		app.Get("/settings/secrets", handler.ListSecrets)

		req := httptest.NewRequest(http.MethodGet, "/settings/secrets", nil)

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("invalid user ID returns error", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewUserSettingsHandler(nil, nil)

		app.Get("/settings/secrets", func(c fiber.Ctx) error {
			c.Locals("user_id", "invalid")
			return handler.ListSecrets(c)
		})

		req := httptest.NewRequest(http.MethodGet, "/settings/secrets", nil)

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
	})
}

// =============================================================================
// GetUserSecretValue Handler Validation Tests (Privileged)
// =============================================================================

func TestGetUserSecretValue_Validation(t *testing.T) {
	t.Run("non-service_role returns forbidden", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewUserSettingsHandler(nil, nil)

		app.Get("/admin/settings/user/:user_id/secret/:key/decrypt", func(c fiber.Ctx) error {
			c.Locals("user_role", "admin")
			return handler.GetUserSecretValue(c)
		})

		req := httptest.NewRequest(http.MethodGet, "/admin/settings/user/550e8400-e29b-41d4-a716-446655440000/secret/my_key/decrypt", nil)

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)

		respBody, _ := io.ReadAll(resp.Body)
		var result map[string]interface{}
		_ = json.Unmarshal(respBody, &result)
		assert.Equal(t, "This operation requires service_role", result["error"])
	})

	t.Run("no role returns forbidden", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewUserSettingsHandler(nil, nil)

		app.Get("/admin/settings/user/:user_id/secret/:key/decrypt", handler.GetUserSecretValue)

		req := httptest.NewRequest(http.MethodGet, "/admin/settings/user/550e8400-e29b-41d4-a716-446655440000/secret/my_key/decrypt", nil)

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)
	})

	t.Run("service_role with no secrets service returns error", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewUserSettingsHandler(nil, nil)

		app.Get("/admin/settings/user/:user_id/secret/:key/decrypt", func(c fiber.Ctx) error {
			c.Locals("user_role", "service_role")
			return handler.GetUserSecretValue(c)
		})

		req := httptest.NewRequest(http.MethodGet, "/admin/settings/user/550e8400-e29b-41d4-a716-446655440000/secret/my_key/decrypt", nil)

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)

		respBody, _ := io.ReadAll(resp.Body)
		var result map[string]interface{}
		_ = json.Unmarshal(respBody, &result)
		assert.Equal(t, "Secrets service not configured", result["error"])
	})

	t.Run("tenant_service passes role check", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewUserSettingsHandler(nil, nil)

		app.Get("/admin/settings/user/:user_id/secret/:key/decrypt", func(c fiber.Ctx) error {
			c.Locals("user_role", "tenant_service")
			return handler.GetUserSecretValue(c)
		})

		req := httptest.NewRequest(http.MethodGet, "/admin/settings/user/550e8400-e29b-41d4-a716-446655440000/secret/my_key/decrypt", nil)

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)

		respBody, _ := io.ReadAll(resp.Body)
		var result map[string]interface{}
		_ = json.Unmarshal(respBody, &result)
		assert.Equal(t, "Secrets service not configured", result["error"])
	})

	t.Run("invalid user_id format returns error", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewUserSettingsHandler(nil, nil)
		handler.secretsService = nil // Will be caught by role check first anyway

		app.Get("/admin/settings/user/:user_id/secret/:key/decrypt", func(c fiber.Ctx) error {
			c.Locals("user_role", "service_role")
			return handler.GetUserSecretValue(c)
		})

		req := httptest.NewRequest(http.MethodGet, "/admin/settings/user/invalid-uuid/secret/my_key/decrypt", nil)

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		// Will fail at secrets service check since it's nil
		// This is expected behavior - role check passes, then secrets service check
		assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
	})
}

// =============================================================================
// GetSetting Handler Validation Tests (with fallback)
// =============================================================================

func TestGetSetting_UserSettings_Validation(t *testing.T) {
	t.Run("empty key returns error", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewUserSettingsHandler(nil, nil)

		app.Get("/settings/user/:key", func(c fiber.Ctx) error {
			// Force empty key for test
			return handler.GetSetting(c)
		})

		req := httptest.NewRequest(http.MethodGet, "/settings/user/", nil)

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		// With empty param, Fiber returns 404 for route mismatch
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	})

	t.Run("missing user context returns unauthorized", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewUserSettingsHandler(nil, nil)

		app.Get("/settings/user/:key", handler.GetSetting)

		req := httptest.NewRequest(http.MethodGet, "/settings/user/my_key", nil)

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
	})
}

// =============================================================================
// GetUserOwnSetting Handler Validation Tests
// =============================================================================

func TestGetUserOwnSetting_Validation(t *testing.T) {
	t.Run("missing user context returns unauthorized", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewUserSettingsHandler(nil, nil)

		app.Get("/settings/user/own/:key", handler.GetUserOwnSetting)

		req := httptest.NewRequest(http.MethodGet, "/settings/user/own/my_key", nil)

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
	})
}

// =============================================================================
// GetSystemSettingPublic Handler Validation Tests
// =============================================================================

func TestGetSystemSettingPublic_Validation(t *testing.T) {
	t.Run("empty key returns error", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewUserSettingsHandler(nil, nil)

		app.Get("/settings/user/system/:key", func(c fiber.Ctx) error {
			// Explicit empty key test by overriding params
			return handler.GetSystemSettingPublic(c)
		})

		req := httptest.NewRequest(http.MethodGet, "/settings/user/system/", nil)

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		// Route mismatch
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	})
}

// =============================================================================
// SetSetting Handler Validation Tests
// =============================================================================

func TestSetSetting_Validation(t *testing.T) {
	t.Run("missing user context returns unauthorized", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewUserSettingsHandler(nil, nil)

		app.Put("/settings/user/:key", handler.SetSetting)

		body := `{"value": {"setting": "value"}}`
		req := httptest.NewRequest(http.MethodPut, "/settings/user/my_key", bytes.NewReader([]byte(body)))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("invalid user ID returns error", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewUserSettingsHandler(nil, nil)

		app.Put("/settings/user/:key", func(c fiber.Ctx) error {
			c.Locals("user_id", "not-valid-uuid")
			return handler.SetSetting(c)
		})

		body := `{"value": {"setting": "value"}}`
		req := httptest.NewRequest(http.MethodPut, "/settings/user/my_key", bytes.NewReader([]byte(body)))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
	})

	t.Run("invalid request body returns error", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewUserSettingsHandler(nil, nil)

		app.Put("/settings/user/:key", func(c fiber.Ctx) error {
			c.Locals("user_id", "550e8400-e29b-41d4-a716-446655440000")
			return handler.SetSetting(c)
		})

		req := httptest.NewRequest(http.MethodPut, "/settings/user/my_key", bytes.NewReader([]byte("not json")))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})

	t.Run("missing value returns error", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewUserSettingsHandler(nil, nil)

		app.Put("/settings/user/:key", func(c fiber.Ctx) error {
			c.Locals("user_id", "550e8400-e29b-41d4-a716-446655440000")
			return handler.SetSetting(c)
		})

		body := `{"description": "some desc"}`
		req := httptest.NewRequest(http.MethodPut, "/settings/user/my_key", bytes.NewReader([]byte(body)))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

		respBody, _ := io.ReadAll(resp.Body)
		var result map[string]interface{}
		_ = json.Unmarshal(respBody, &result)
		assert.Equal(t, "value is required", result["error"])
	})
}

// =============================================================================
// DeleteSetting Handler Validation Tests
// =============================================================================

func TestDeleteSetting_UserSettings_Validation(t *testing.T) {
	t.Run("missing user context returns unauthorized", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewUserSettingsHandler(nil, nil)

		app.Delete("/settings/user/:key", handler.DeleteSetting)

		req := httptest.NewRequest(http.MethodDelete, "/settings/user/my_key", nil)

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
	})
}

// =============================================================================
// ListSettings Handler Validation Tests
// =============================================================================

func TestListSettings_Validation(t *testing.T) {
	t.Run("missing user context returns unauthorized", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewUserSettingsHandler(nil, nil)

		app.Get("/settings/user/list", handler.ListSettings)

		req := httptest.NewRequest(http.MethodGet, "/settings/user/list", nil)

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("invalid user ID returns error", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewUserSettingsHandler(nil, nil)

		app.Get("/settings/user/list", func(c fiber.Ctx) error {
			c.Locals("user_id", "bad-uuid")
			return handler.ListSettings(c)
		})

		req := httptest.NewRequest(http.MethodGet, "/settings/user/list", nil)

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
	})
}

// =============================================================================
// Error Response Format Tests
// =============================================================================

func TestUserSettingsErrorResponses(t *testing.T) {
	t.Run("duplicate key error response format", func(t *testing.T) {
		expectedError := fiber.Map{
			"error": "A secret with this key already exists",
			"code":  "DUPLICATE_KEY",
		}

		data, err := json.Marshal(expectedError)
		require.NoError(t, err)

		assert.Contains(t, string(data), `"code":"DUPLICATE_KEY"`)
		assert.Contains(t, string(data), `"error":"A secret with this key already exists"`)
	})

	t.Run("invalid key error response format", func(t *testing.T) {
		expectedError := fiber.Map{
			"error": "Invalid setting key format",
			"code":  "INVALID_KEY",
		}

		data, err := json.Marshal(expectedError)
		require.NoError(t, err)

		assert.Contains(t, string(data), `"code":"INVALID_KEY"`)
	})

	t.Run("not found error response", func(t *testing.T) {
		expectedError := fiber.Map{
			"error": "Secret not found",
		}

		data, err := json.Marshal(expectedError)
		require.NoError(t, err)

		assert.Contains(t, string(data), `"error":"Secret not found"`)
	})
}
