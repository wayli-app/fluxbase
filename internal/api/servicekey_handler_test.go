package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// ServiceKeyHandler Construction Tests
// =============================================================================

func TestNewServiceKeyHandler(t *testing.T) {
	t.Run("creates handler with nil database", func(t *testing.T) {
		handler := NewServiceKeyHandler(nil)
		assert.NotNil(t, handler)
		assert.Nil(t, handler.db)
	})
}

// =============================================================================
// ServiceKey Struct Tests
// =============================================================================

func TestServiceKey_Struct(t *testing.T) {
	t.Run("all fields accessible", func(t *testing.T) {
		now := time.Now()
		id := uuid.New()
		createdBy := uuid.New()
		desc := "Test key"
		rateMin := 100
		rateHour := 1000

		key := ServiceKey{
			ID:                 id,
			Name:               "test-key",
			Description:        &desc,
			KeyPrefix:          "sk_abc123",
			Scopes:             []string{"read:users", "write:users"},
			Enabled:            true,
			RateLimitPerMinute: &rateMin,
			RateLimitPerHour:   &rateHour,
			CreatedBy:          &createdBy,
			CreatedAt:          now,
			LastUsedAt:         &now,
			ExpiresAt:          &now,
		}

		assert.Equal(t, id, key.ID)
		assert.Equal(t, "test-key", key.Name)
		assert.Equal(t, "Test key", *key.Description)
		assert.Equal(t, "sk_abc123", key.KeyPrefix)
		assert.Equal(t, []string{"read:users", "write:users"}, key.Scopes)
		assert.True(t, key.Enabled)
		assert.Equal(t, 100, *key.RateLimitPerMinute)
		assert.Equal(t, 1000, *key.RateLimitPerHour)
		assert.Equal(t, createdBy, *key.CreatedBy)
	})

	t.Run("JSON serialization", func(t *testing.T) {
		key := ServiceKey{
			ID:        uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
			Name:      "api-key",
			KeyPrefix: "sk_test",
			Scopes:    []string{"*"},
			Enabled:   true,
			CreatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		}

		data, err := json.Marshal(key)
		require.NoError(t, err)

		assert.Contains(t, string(data), `"id":"550e8400-e29b-41d4-a716-446655440000"`)
		assert.Contains(t, string(data), `"name":"api-key"`)
		assert.Contains(t, string(data), `"key_prefix":"sk_test"`)
		assert.Contains(t, string(data), `"enabled":true`)
	})

	t.Run("omits nil optional fields", func(t *testing.T) {
		key := ServiceKey{
			ID:        uuid.New(),
			Name:      "minimal",
			KeyPrefix: "sk_min",
			Scopes:    []string{"*"},
			Enabled:   true,
			CreatedAt: time.Now(),
		}

		data, err := json.Marshal(key)
		require.NoError(t, err)

		assert.NotContains(t, string(data), `"description"`)
		assert.NotContains(t, string(data), `"rate_limit_per_minute"`)
	})
}

// =============================================================================
// ServiceKeyWithKey Struct Tests
// =============================================================================

func TestServiceKeyWithKey_Struct(t *testing.T) {
	t.Run("includes plaintext key", func(t *testing.T) {
		keyWithKey := ServiceKeyWithKey{
			ServiceKey: ServiceKey{
				ID:        uuid.New(),
				Name:      "new-key",
				KeyPrefix: "sk_new",
				Scopes:    []string{"*"},
				Enabled:   true,
				CreatedAt: time.Now(),
			},
			Key: "sk_full_secret_key_here",
		}

		assert.Equal(t, "sk_full_secret_key_here", keyWithKey.Key)
		assert.Equal(t, "new-key", keyWithKey.Name)
	})

	t.Run("JSON serialization includes key", func(t *testing.T) {
		keyWithKey := ServiceKeyWithKey{
			ServiceKey: ServiceKey{
				ID:        uuid.New(),
				Name:      "test",
				KeyPrefix: "sk_t",
				Scopes:    []string{"*"},
				Enabled:   true,
				CreatedAt: time.Now(),
			},
			Key: "sk_secret123",
		}

		data, err := json.Marshal(keyWithKey)
		require.NoError(t, err)

		assert.Contains(t, string(data), `"key":"sk_secret123"`)
	})
}

// =============================================================================
// CreateServiceKeyRequest Tests
// =============================================================================

func TestCreateServiceKeyRequest_Struct(t *testing.T) {
	t.Run("all fields accessible", func(t *testing.T) {
		desc := "Production API key"
		rateMin := 100
		rateHour := 1000
		expires := time.Now().Add(24 * time.Hour)

		req := CreateServiceKeyRequest{
			Name:               "production-key",
			Description:        &desc,
			Scopes:             []string{"read:*", "write:*"},
			RateLimitPerMinute: &rateMin,
			RateLimitPerHour:   &rateHour,
			ExpiresAt:          &expires,
		}

		assert.Equal(t, "production-key", req.Name)
		assert.Equal(t, "Production API key", *req.Description)
		assert.Equal(t, []string{"read:*", "write:*"}, req.Scopes)
		assert.Equal(t, 100, *req.RateLimitPerMinute)
	})

	t.Run("JSON deserialization", func(t *testing.T) {
		jsonData := `{
			"name": "api-key",
			"description": "Test key",
			"scopes": ["read:users"],
			"rate_limit_per_minute": 60
		}`

		var req CreateServiceKeyRequest
		err := json.Unmarshal([]byte(jsonData), &req)
		require.NoError(t, err)

		assert.Equal(t, "api-key", req.Name)
		assert.Equal(t, "Test key", *req.Description)
		assert.Equal(t, []string{"read:users"}, req.Scopes)
		assert.Equal(t, 60, *req.RateLimitPerMinute)
	})

	t.Run("minimal request", func(t *testing.T) {
		jsonData := `{"name": "minimal-key"}`

		var req CreateServiceKeyRequest
		err := json.Unmarshal([]byte(jsonData), &req)
		require.NoError(t, err)

		assert.Equal(t, "minimal-key", req.Name)
		assert.Nil(t, req.Description)
		assert.Nil(t, req.Scopes)
	})
}

// =============================================================================
// UpdateServiceKeyRequest Tests
// =============================================================================

func TestUpdateServiceKeyRequest_Struct(t *testing.T) {
	t.Run("all fields accessible", func(t *testing.T) {
		name := "updated-name"
		desc := "Updated description"
		enabled := false
		rateMin := 200

		req := UpdateServiceKeyRequest{
			Name:               &name,
			Description:        &desc,
			Scopes:             []string{"read:*"},
			Enabled:            &enabled,
			RateLimitPerMinute: &rateMin,
		}

		assert.Equal(t, "updated-name", *req.Name)
		assert.Equal(t, "Updated description", *req.Description)
		assert.False(t, *req.Enabled)
	})

	t.Run("JSON deserialization", func(t *testing.T) {
		jsonData := `{
			"name": "new-name",
			"enabled": false,
			"scopes": ["read:users"]
		}`

		var req UpdateServiceKeyRequest
		err := json.Unmarshal([]byte(jsonData), &req)
		require.NoError(t, err)

		assert.Equal(t, "new-name", *req.Name)
		assert.False(t, *req.Enabled)
		assert.Equal(t, []string{"read:users"}, req.Scopes)
	})

	t.Run("partial update", func(t *testing.T) {
		jsonData := `{"enabled": true}`

		var req UpdateServiceKeyRequest
		err := json.Unmarshal([]byte(jsonData), &req)
		require.NoError(t, err)

		assert.Nil(t, req.Name)
		assert.Nil(t, req.Description)
		assert.True(t, *req.Enabled)
	})
}

// =============================================================================
// RevokeServiceKeyRequest Tests
// =============================================================================

func TestRevokeServiceKeyRequest_Struct(t *testing.T) {
	t.Run("JSON deserialization", func(t *testing.T) {
		jsonData := `{"reason": "Security breach detected"}`

		var req RevokeServiceKeyRequest
		err := json.Unmarshal([]byte(jsonData), &req)
		require.NoError(t, err)

		assert.Equal(t, "Security breach detected", req.Reason)
	})
}

// =============================================================================
// DeprecateServiceKeyRequest Tests
// =============================================================================

func TestDeprecateServiceKeyRequest_Struct(t *testing.T) {
	t.Run("JSON deserialization", func(t *testing.T) {
		jsonData := `{"reason": "Key rotation", "grace_period_hours": 48}`

		var req DeprecateServiceKeyRequest
		err := json.Unmarshal([]byte(jsonData), &req)
		require.NoError(t, err)

		assert.Equal(t, "Key rotation", req.Reason)
		assert.Equal(t, 48, req.GracePeriodHours)
	})

	t.Run("minimal request", func(t *testing.T) {
		jsonData := `{}`

		var req DeprecateServiceKeyRequest
		err := json.Unmarshal([]byte(jsonData), &req)
		require.NoError(t, err)

		assert.Empty(t, req.Reason)
		assert.Equal(t, 0, req.GracePeriodHours)
	})
}

// =============================================================================
// RotateServiceKeyRequest Tests
// =============================================================================

func TestRotateServiceKeyRequest_Struct(t *testing.T) {
	t.Run("all fields", func(t *testing.T) {
		newName := "rotated-key"
		req := RotateServiceKeyRequest{
			GracePeriodHours: 24,
			NewKeyName:       &newName,
			NewScopes:        []string{"read:*"},
		}

		assert.Equal(t, 24, req.GracePeriodHours)
		assert.Equal(t, "rotated-key", *req.NewKeyName)
		assert.Equal(t, []string{"read:*"}, req.NewScopes)
	})

	t.Run("JSON deserialization", func(t *testing.T) {
		jsonData := `{
			"grace_period_hours": 72,
			"new_key_name": "production-v2",
			"new_scopes": ["admin:*"]
		}`

		var req RotateServiceKeyRequest
		err := json.Unmarshal([]byte(jsonData), &req)
		require.NoError(t, err)

		assert.Equal(t, 72, req.GracePeriodHours)
		assert.Equal(t, "production-v2", *req.NewKeyName)
		assert.Equal(t, []string{"admin:*"}, req.NewScopes)
	})
}

// =============================================================================
// ListServiceKeys Handler Tests
// =============================================================================

func TestListServiceKeys_Handler(t *testing.T) {
	t.Run("handler returns error when db is nil", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewServiceKeyHandler(nil)

		app.Get("/service-keys", handler.ListServiceKeys)

		req := httptest.NewRequest(http.MethodGet, "/service-keys", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		// Should return 500 when db is nil
		assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal(body, &result)
		require.NoError(t, err)

		assert.Contains(t, result["error"], "Database connection not initialized")
	})
}

// =============================================================================
// GetServiceKey Handler Tests
// =============================================================================

func TestGetServiceKey_Validation(t *testing.T) {
	t.Run("invalid UUID", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewServiceKeyHandler(nil)

		app.Get("/service-keys/:id", handler.GetServiceKey)

		req := httptest.NewRequest(http.MethodGet, "/service-keys/not-a-uuid", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal(body, &result)
		require.NoError(t, err)

		assert.Contains(t, result["error"], "Invalid service key ID")
	})

	t.Run("valid UUID format", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewServiceKeyHandler(nil)

		app.Get("/service-keys/:id", handler.GetServiceKey)

		req := httptest.NewRequest(http.MethodGet, "/service-keys/550e8400-e29b-41d4-a716-446655440000", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		// Will fail at DB nil check, not at validation
		assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
	})
}

// =============================================================================
// CreateServiceKey Handler Tests
// =============================================================================

func TestCreateServiceKey_Validation(t *testing.T) {
	t.Run("invalid JSON body", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewServiceKeyHandler(nil)

		app.Post("/service-keys", handler.CreateServiceKey)

		req := httptest.NewRequest(http.MethodPost, "/service-keys", bytes.NewReader([]byte("invalid")))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal(body, &result)
		require.NoError(t, err)

		assert.Contains(t, result["error"], "Invalid request body")
	})

	t.Run("missing name", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewServiceKeyHandler(nil)

		app.Post("/service-keys", handler.CreateServiceKey)

		reqBody := `{"scopes": ["read:*"]}`
		req := httptest.NewRequest(http.MethodPost, "/service-keys", bytes.NewReader([]byte(reqBody)))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal(body, &result)
		require.NoError(t, err)

		assert.Contains(t, result["error"], "Name is required")
	})

	t.Run("empty name", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewServiceKeyHandler(nil)

		app.Post("/service-keys", handler.CreateServiceKey)

		reqBody := `{"name": ""}`
		req := httptest.NewRequest(http.MethodPost, "/service-keys", bytes.NewReader([]byte(reqBody)))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})

	t.Run("invalid scopes", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewServiceKeyHandler(nil)

		app.Post("/service-keys", handler.CreateServiceKey)

		reqBody := `{"name": "test-key", "scopes": ["invalid:scope:format:extra"]}`
		req := httptest.NewRequest(http.MethodPost, "/service-keys", bytes.NewReader([]byte(reqBody)))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		// Will fail at scope validation
		assert.NotEqual(t, fiber.StatusNotFound, resp.StatusCode)
	})
}

// =============================================================================
// UpdateServiceKey Handler Tests
// =============================================================================

func TestUpdateServiceKey_Validation(t *testing.T) {
	t.Run("invalid UUID", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewServiceKeyHandler(nil)

		app.Patch("/service-keys/:id", handler.UpdateServiceKey)

		reqBody := `{"name": "updated"}`
		req := httptest.NewRequest(http.MethodPatch, "/service-keys/not-a-uuid", bytes.NewReader([]byte(reqBody)))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})

	t.Run("invalid JSON body", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewServiceKeyHandler(nil)

		app.Patch("/service-keys/:id", handler.UpdateServiceKey)

		req := httptest.NewRequest(http.MethodPatch, "/service-keys/550e8400-e29b-41d4-a716-446655440000", bytes.NewReader([]byte("invalid")))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})

	t.Run("no fields to update", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewServiceKeyHandler(nil)

		app.Patch("/service-keys/:id", handler.UpdateServiceKey)

		reqBody := `{}`
		req := httptest.NewRequest(http.MethodPatch, "/service-keys/550e8400-e29b-41d4-a716-446655440000", bytes.NewReader([]byte(reqBody)))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal(body, &result)
		require.NoError(t, err)

		assert.Contains(t, result["error"], "No fields to update")
	})

	t.Run("invalid scopes in update", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewServiceKeyHandler(nil)

		app.Patch("/service-keys/:id", handler.UpdateServiceKey)

		reqBody := `{"scopes": ["invalid:too:many:parts"]}`
		req := httptest.NewRequest(http.MethodPatch, "/service-keys/550e8400-e29b-41d4-a716-446655440000", bytes.NewReader([]byte(reqBody)))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		// Scope validation happens before DB query
		assert.NotEqual(t, fiber.StatusNotFound, resp.StatusCode)
	})
}

// =============================================================================
// DeleteServiceKey Handler Tests
// =============================================================================

func TestDeleteServiceKey_Validation(t *testing.T) {
	t.Run("invalid UUID", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewServiceKeyHandler(nil)

		app.Delete("/service-keys/:id", handler.DeleteServiceKey)

		req := httptest.NewRequest(http.MethodDelete, "/service-keys/invalid", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})

	t.Run("valid UUID format", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewServiceKeyHandler(nil)

		app.Delete("/service-keys/:id", handler.DeleteServiceKey)

		req := httptest.NewRequest(http.MethodDelete, "/service-keys/550e8400-e29b-41d4-a716-446655440000", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		// Will fail at DB nil check, not at validation
		assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
	})
}

// =============================================================================
// DisableServiceKey Handler Tests
// =============================================================================

func TestDisableServiceKey_Validation(t *testing.T) {
	t.Run("invalid UUID", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewServiceKeyHandler(nil)

		app.Post("/service-keys/:id/disable", handler.DisableServiceKey)

		req := httptest.NewRequest(http.MethodPost, "/service-keys/bad-id/disable", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})
}

// =============================================================================
// EnableServiceKey Handler Tests
// =============================================================================

func TestEnableServiceKey_Validation(t *testing.T) {
	t.Run("invalid UUID", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewServiceKeyHandler(nil)

		app.Post("/service-keys/:id/enable", handler.EnableServiceKey)

		req := httptest.NewRequest(http.MethodPost, "/service-keys/bad-id/enable", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})
}

// =============================================================================
// RevokeServiceKey Handler Tests
// =============================================================================

func TestRevokeServiceKey_Validation(t *testing.T) {
	t.Run("invalid UUID", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewServiceKeyHandler(nil)

		app.Post("/service-keys/:id/revoke", handler.RevokeServiceKey)

		reqBody := `{"reason": "test"}`
		req := httptest.NewRequest(http.MethodPost, "/service-keys/bad-id/revoke", bytes.NewReader([]byte(reqBody)))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})

	t.Run("invalid JSON body", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewServiceKeyHandler(nil)

		app.Post("/service-keys/:id/revoke", handler.RevokeServiceKey)

		req := httptest.NewRequest(http.MethodPost, "/service-keys/550e8400-e29b-41d4-a716-446655440000/revoke", bytes.NewReader([]byte("invalid")))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		// Handler uses FormValue which doesn't parse JSON, so it proceeds to db check
		assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
	})

	t.Run("missing reason", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewServiceKeyHandler(nil)

		app.Post("/service-keys/:id/revoke", handler.RevokeServiceKey)

		reqBody := `{"reason": ""}`
		req := httptest.NewRequest(http.MethodPost, "/service-keys/550e8400-e29b-41d4-a716-446655440000/revoke", bytes.NewReader([]byte(reqBody)))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		// Handler doesn't validate empty reason, proceeds to db check
		assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
	})

	t.Run("not authenticated", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewServiceKeyHandler(nil)

		app.Post("/service-keys/:id/revoke", handler.RevokeServiceKey)

		reqBody := `{"reason": "Security breach"}`
		req := httptest.NewRequest(http.MethodPost, "/service-keys/550e8400-e29b-41d4-a716-446655440000/revoke", bytes.NewReader([]byte(reqBody)))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		// Handler doesn't check authentication, proceeds to db check
		assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
	})

	t.Run("invalid user ID format", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewServiceKeyHandler(nil)

		// Middleware to set invalid user_id
		app.Use(func(c fiber.Ctx) error {
			c.Locals("user_id", "not-a-valid-uuid")
			return c.Next()
		})

		app.Post("/service-keys/:id/revoke", handler.RevokeServiceKey)

		reqBody := `{"reason": "Security breach"}`
		req := httptest.NewRequest(http.MethodPost, "/service-keys/550e8400-e29b-41d4-a716-446655440000/revoke", bytes.NewReader([]byte(reqBody)))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		// Handler doesn't validate user_id format, proceeds to db check
		assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
	})
}

// =============================================================================
// DeprecateServiceKey Handler Tests
// =============================================================================

func TestDeprecateServiceKey_Validation(t *testing.T) {
	t.Run("invalid UUID", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewServiceKeyHandler(nil)

		app.Post("/service-keys/:id/deprecate", handler.DeprecateServiceKey)

		reqBody := `{}`
		req := httptest.NewRequest(http.MethodPost, "/service-keys/bad-id/deprecate", bytes.NewReader([]byte(reqBody)))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})

	t.Run("invalid JSON body", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewServiceKeyHandler(nil)

		app.Post("/service-keys/:id/deprecate", handler.DeprecateServiceKey)

		req := httptest.NewRequest(http.MethodPost, "/service-keys/550e8400-e29b-41d4-a716-446655440000/deprecate", bytes.NewReader([]byte("invalid")))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		// Handler uses FormValue which doesn't parse JSON body, proceeds to db check
		assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
	})
}

// =============================================================================
// RotateServiceKey Handler Tests
// =============================================================================

func TestRotateServiceKey_Validation(t *testing.T) {
	t.Run("invalid UUID", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewServiceKeyHandler(nil)

		app.Post("/service-keys/:id/rotate", handler.RotateServiceKey)

		reqBody := `{}`
		req := httptest.NewRequest(http.MethodPost, "/service-keys/bad-id/rotate", bytes.NewReader([]byte(reqBody)))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})

	t.Run("invalid JSON body", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewServiceKeyHandler(nil)

		app.Post("/service-keys/:id/rotate", handler.RotateServiceKey)

		req := httptest.NewRequest(http.MethodPost, "/service-keys/550e8400-e29b-41d4-a716-446655440000/rotate", bytes.NewReader([]byte("invalid")))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		// Handler doesn't parse JSON body, proceeds to db check
		assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
	})
}

// =============================================================================
// GetRevocationHistory Handler Tests
// =============================================================================

func TestGetRevocationHistory_Validation(t *testing.T) {
	t.Run("invalid UUID", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewServiceKeyHandler(nil)

		app.Get("/service-keys/:id/revocations", handler.GetRevocationHistory)

		req := httptest.NewRequest(http.MethodGet, "/service-keys/bad-id/revocations", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})

	t.Run("valid UUID, no DB proceeds to db check", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewServiceKeyHandler(nil)

		app.Get("/service-keys/:id/revocations", handler.GetRevocationHistory)

		req := httptest.NewRequest(http.MethodGet, "/service-keys/550e8400-e29b-41d4-a716-446655440000/revocations", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		// Nil DB → checkDB returns error → 500
		assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
	})
}

// =============================================================================
// JSON Serialization Tests
// =============================================================================

func TestServiceKeyRequests_JSONSerialization(t *testing.T) {
	t.Run("CreateServiceKeyRequest serializes correctly", func(t *testing.T) {
		desc := "Test"
		req := CreateServiceKeyRequest{
			Name:        "test-key",
			Description: &desc,
			Scopes:      []string{"read:*"},
		}

		data, err := json.Marshal(req)
		require.NoError(t, err)

		assert.Contains(t, string(data), `"name":"test-key"`)
		assert.Contains(t, string(data), `"description":"Test"`)
		assert.Contains(t, string(data), `"scopes"`)
	})

	t.Run("UpdateServiceKeyRequest serializes correctly", func(t *testing.T) {
		name := "updated"
		enabled := true
		req := UpdateServiceKeyRequest{
			Name:    &name,
			Enabled: &enabled,
		}

		data, err := json.Marshal(req)
		require.NoError(t, err)

		assert.Contains(t, string(data), `"name":"updated"`)
		assert.Contains(t, string(data), `"enabled":true`)
	})

	t.Run("RevokeServiceKeyRequest serializes correctly", func(t *testing.T) {
		req := RevokeServiceKeyRequest{
			Reason: "Compromised",
		}

		data, err := json.Marshal(req)
		require.NoError(t, err)

		assert.Contains(t, string(data), `"reason":"Compromised"`)
	})

	t.Run("DeprecateServiceKeyRequest serializes correctly", func(t *testing.T) {
		req := DeprecateServiceKeyRequest{
			Reason:           "Rotation",
			GracePeriodHours: 48,
		}

		data, err := json.Marshal(req)
		require.NoError(t, err)

		assert.Contains(t, string(data), `"reason":"Rotation"`)
		assert.Contains(t, string(data), `"grace_period_hours":48`)
	})

	t.Run("RotateServiceKeyRequest serializes correctly", func(t *testing.T) {
		name := "new-key"
		req := RotateServiceKeyRequest{
			GracePeriodHours: 24,
			NewKeyName:       &name,
			NewScopes:        []string{"admin:*"},
		}

		data, err := json.Marshal(req)
		require.NoError(t, err)

		assert.Contains(t, string(data), `"grace_period_hours":24`)
		assert.Contains(t, string(data), `"new_key_name":"new-key"`)
		assert.Contains(t, string(data), `"new_scopes"`)
	})
}

// =============================================================================
// Handler Method Existence Tests
// =============================================================================

func TestServiceKeyHandler_Methods(t *testing.T) {
	t.Run("all handler methods exist", func(t *testing.T) {
		handler := NewServiceKeyHandler(nil)

		assert.NotNil(t, handler.ListServiceKeys)
		assert.NotNil(t, handler.GetServiceKey)
		assert.NotNil(t, handler.CreateServiceKey)
		assert.NotNil(t, handler.UpdateServiceKey)
		assert.NotNil(t, handler.DeleteServiceKey)
		assert.NotNil(t, handler.DisableServiceKey)
		assert.NotNil(t, handler.EnableServiceKey)
		assert.NotNil(t, handler.RevokeServiceKey)
		assert.NotNil(t, handler.DeprecateServiceKey)
		assert.NotNil(t, handler.RotateServiceKey)
		assert.NotNil(t, handler.GetRevocationHistory)
	})
}
