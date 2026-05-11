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
// ClientKeyHandler Construction Tests
// =============================================================================

func TestNewClientKeyHandler(t *testing.T) {
	t.Run("creates handler with nil service", func(t *testing.T) {
		handler := NewClientKeyHandler(nil)
		assert.NotNil(t, handler)
		assert.Nil(t, handler.clientKeyService)
	})
}

// =============================================================================
// CreateClientKeyRequest Struct Tests
// =============================================================================

func TestCreateClientKeyRequest_Struct(t *testing.T) {
	t.Run("all fields accessible", func(t *testing.T) {
		description := "Test key for API"
		expiresAt := time.Now().Add(24 * time.Hour)

		req := CreateClientKeyRequest{
			Name:               "My API Key",
			Description:        &description,
			Scopes:             []string{"read:*", "write:posts"},
			RateLimitPerMinute: 100,
			ExpiresAt:          &expiresAt,
		}

		assert.Equal(t, "My API Key", req.Name)
		assert.Equal(t, "Test key for API", *req.Description)
		assert.Len(t, req.Scopes, 2)
		assert.Equal(t, 100, req.RateLimitPerMinute)
		assert.NotNil(t, req.ExpiresAt)
	})

	t.Run("minimal request without optional fields", func(t *testing.T) {
		req := CreateClientKeyRequest{
			Name:   "Basic Key",
			Scopes: []string{"read:*"},
		}

		assert.Equal(t, "Basic Key", req.Name)
		assert.Nil(t, req.Description)
		assert.Len(t, req.Scopes, 1)
		assert.Equal(t, 0, req.RateLimitPerMinute)
		assert.Nil(t, req.ExpiresAt)
	})

	t.Run("JSON deserialization", func(t *testing.T) {
		jsonData := `{
			"name": "Production Key",
			"description": "Key for production use",
			"scopes": ["read:users", "write:users"],
			"rate_limit_per_minute": 500,
			"expires_at": "2025-12-31T23:59:59Z"
		}`

		var req CreateClientKeyRequest
		err := json.Unmarshal([]byte(jsonData), &req)
		require.NoError(t, err)

		assert.Equal(t, "Production Key", req.Name)
		assert.Equal(t, "Key for production use", *req.Description)
		assert.Len(t, req.Scopes, 2)
		assert.Equal(t, 500, req.RateLimitPerMinute)
		assert.NotNil(t, req.ExpiresAt)
	})

	t.Run("JSON deserialization without optional fields", func(t *testing.T) {
		jsonData := `{
			"name": "Simple Key",
			"scopes": ["read:*"]
		}`

		var req CreateClientKeyRequest
		err := json.Unmarshal([]byte(jsonData), &req)
		require.NoError(t, err)

		assert.Equal(t, "Simple Key", req.Name)
		assert.Nil(t, req.Description)
		assert.Nil(t, req.ExpiresAt)
	})

	t.Run("JSON serialization", func(t *testing.T) {
		description := "API Key"
		req := CreateClientKeyRequest{
			Name:               "Test Key",
			Description:        &description,
			Scopes:             []string{"read:all"},
			RateLimitPerMinute: 60,
		}

		data, err := json.Marshal(req)
		require.NoError(t, err)

		assert.Contains(t, string(data), `"name":"Test Key"`)
		assert.Contains(t, string(data), `"description":"API Key"`)
		assert.Contains(t, string(data), `"scopes":["read:all"]`)
		assert.Contains(t, string(data), `"rate_limit_per_minute":60`)
	})
}

// =============================================================================
// UpdateClientKeyRequest Struct Tests
// =============================================================================

func TestUpdateClientKeyRequest_Struct(t *testing.T) {
	t.Run("all fields accessible", func(t *testing.T) {
		name := "Updated Key"
		description := "Updated description"
		rateLimit := 200

		req := UpdateClientKeyRequest{
			Name:               &name,
			Description:        &description,
			Scopes:             []string{"read:*", "write:*"},
			RateLimitPerMinute: &rateLimit,
		}

		assert.Equal(t, "Updated Key", *req.Name)
		assert.Equal(t, "Updated description", *req.Description)
		assert.Len(t, req.Scopes, 2)
		assert.Equal(t, 200, *req.RateLimitPerMinute)
	})

	t.Run("partial update - name only", func(t *testing.T) {
		name := "New Name"
		req := UpdateClientKeyRequest{
			Name: &name,
		}

		assert.Equal(t, "New Name", *req.Name)
		assert.Nil(t, req.Description)
		assert.Nil(t, req.Scopes)
		assert.Nil(t, req.RateLimitPerMinute)
	})

	t.Run("partial update - scopes only", func(t *testing.T) {
		req := UpdateClientKeyRequest{
			Scopes: []string{"read:users"},
		}

		assert.Nil(t, req.Name)
		assert.Len(t, req.Scopes, 1)
	})

	t.Run("JSON deserialization - partial update", func(t *testing.T) {
		jsonData := `{"name":"Updated Name"}`

		var req UpdateClientKeyRequest
		err := json.Unmarshal([]byte(jsonData), &req)
		require.NoError(t, err)

		assert.Equal(t, "Updated Name", *req.Name)
		assert.Nil(t, req.Description)
		assert.Nil(t, req.Scopes)
		assert.Nil(t, req.RateLimitPerMinute)
	})

	t.Run("JSON deserialization - full update", func(t *testing.T) {
		jsonData := `{
			"name": "Full Update",
			"description": "New description",
			"scopes": ["admin"],
			"rate_limit_per_minute": 1000
		}`

		var req UpdateClientKeyRequest
		err := json.Unmarshal([]byte(jsonData), &req)
		require.NoError(t, err)

		assert.Equal(t, "Full Update", *req.Name)
		assert.Equal(t, "New description", *req.Description)
		assert.Equal(t, []string{"admin"}, req.Scopes)
		assert.Equal(t, 1000, *req.RateLimitPerMinute)
	})
}

// =============================================================================
// CreateClientKey Handler Tests
// =============================================================================

func TestCreateClientKey_Validation(t *testing.T) {
	t.Run("invalid request body", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewClientKeyHandler(nil)

		app.Post("/client-keys", handler.CreateClientKey)

		req := httptest.NewRequest(http.MethodPost, "/client-keys", bytes.NewReader([]byte("invalid json")))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

		respBody, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal(respBody, &result)
		require.NoError(t, err)

		assert.Contains(t, result["error"], "Invalid request body")
	})

	t.Run("missing name", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewClientKeyHandler(nil)

		app.Post("/client-keys", handler.CreateClientKey)

		body := `{"scopes": ["read:*"]}`
		req := httptest.NewRequest(http.MethodPost, "/client-keys", bytes.NewReader([]byte(body)))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

		respBody, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal(respBody, &result)
		require.NoError(t, err)

		assert.Contains(t, result["error"], "Name is required")
	})

	t.Run("empty name", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewClientKeyHandler(nil)

		app.Post("/client-keys", handler.CreateClientKey)

		body := `{"name": "", "scopes": ["read:*"]}`
		req := httptest.NewRequest(http.MethodPost, "/client-keys", bytes.NewReader([]byte(body)))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

		respBody, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal(respBody, &result)
		require.NoError(t, err)

		assert.Contains(t, result["error"], "Name is required")
	})

	t.Run("valid body but nil service", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewClientKeyHandler(nil)

		app.Post("/client-keys", handler.CreateClientKey)

		body := `{"name": "Test Key", "scopes": ["read:*"]}`
		req := httptest.NewRequest(http.MethodPost, "/client-keys", bytes.NewReader([]byte(body)))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		// Guard returns 503 when service is nil
		assert.Equal(t, fiber.StatusServiceUnavailable, resp.StatusCode)
	})

	t.Run("valid body with user_id in context", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewClientKeyHandler(nil)

		// Middleware to set user_id
		app.Use(func(c fiber.Ctx) error {
			c.Locals("user_id", uuid.New())
			return c.Next()
		})

		app.Post("/client-keys", handler.CreateClientKey)

		body := `{"name": "User Key", "scopes": ["read:*"], "rate_limit_per_minute": 60}`
		req := httptest.NewRequest(http.MethodPost, "/client-keys", bytes.NewReader([]byte(body)))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		// Guard returns 503 when service is nil
		assert.Equal(t, fiber.StatusServiceUnavailable, resp.StatusCode)
	})
}

// =============================================================================
// ListClientKeys Handler Tests
// =============================================================================

func TestListClientKeys_ParameterParsing(t *testing.T) {
	t.Run("without parameters", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewClientKeyHandler(nil)

		app.Get("/client-keys", handler.ListClientKeys)

		req := httptest.NewRequest(http.MethodGet, "/client-keys", nil)

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		// Fails at service call
		assert.NotEqual(t, fiber.StatusNotFound, resp.StatusCode)
		assert.NotEqual(t, fiber.StatusBadRequest, resp.StatusCode)
	})

	t.Run("with valid user_id filter", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewClientKeyHandler(nil)

		// Set admin role to allow filtering
		app.Use(func(c fiber.Ctx) error {
			c.Locals("user_role", "admin")
			return c.Next()
		})

		app.Get("/client-keys", handler.ListClientKeys)

		userID := uuid.New().String()
		req := httptest.NewRequest(http.MethodGet, "/client-keys?user_id="+userID, nil)

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		// Parameter parsing works, fails at service call
		assert.NotEqual(t, fiber.StatusBadRequest, resp.StatusCode)
	})

	t.Run("with invalid user_id filter", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewClientKeyHandler(nil)

		app.Get("/client-keys", handler.ListClientKeys)

		req := httptest.NewRequest(http.MethodGet, "/client-keys?user_id=invalid-uuid", nil)

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

		respBody, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal(respBody, &result)
		require.NoError(t, err)

		assert.Contains(t, result["error"], "Invalid user ID")
	})

	t.Run("non-admin trying to list other user's keys", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewClientKeyHandler(nil)

		currentUserID := uuid.New().String()
		otherUserID := uuid.New().String()

		// Set non-admin role
		app.Use(func(c fiber.Ctx) error {
			c.Locals("user_id", currentUserID)
			c.Locals("user_role", "user")
			return c.Next()
		})

		app.Get("/client-keys", handler.ListClientKeys)

		req := httptest.NewRequest(http.MethodGet, "/client-keys?user_id="+otherUserID, nil)

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)

		respBody, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal(respBody, &result)
		require.NoError(t, err)

		assert.Contains(t, result["error"], "Cannot view other users' client keys")
	})

	t.Run("admin can list other user's keys", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewClientKeyHandler(nil)

		otherUserID := uuid.New().String()

		// Set admin role
		app.Use(func(c fiber.Ctx) error {
			c.Locals("user_id", uuid.New().String())
			c.Locals("user_role", "admin")
			return c.Next()
		})

		app.Get("/client-keys", handler.ListClientKeys)

		req := httptest.NewRequest(http.MethodGet, "/client-keys?user_id="+otherUserID, nil)

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		// Parameter parsing works, only fails at service call (no forbidden)
		assert.NotEqual(t, fiber.StatusForbidden, resp.StatusCode)
	})

	t.Run("instance_admin can list other user's keys", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewClientKeyHandler(nil)

		// Set instance_admin role
		app.Use(func(c fiber.Ctx) error {
			c.Locals("user_id", uuid.New().String())
			c.Locals("user_role", "instance_admin")
			return c.Next()
		})

		app.Get("/client-keys", handler.ListClientKeys)

		req := httptest.NewRequest(http.MethodGet, "/client-keys?user_id="+uuid.New().String(), nil)

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		// Not forbidden
		assert.NotEqual(t, fiber.StatusForbidden, resp.StatusCode)
	})

	t.Run("service_role can list other user's keys", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewClientKeyHandler(nil)

		// Set service_role
		app.Use(func(c fiber.Ctx) error {
			c.Locals("user_id", uuid.New().String())
			c.Locals("user_role", "service_role")
			return c.Next()
		})

		app.Get("/client-keys", handler.ListClientKeys)

		req := httptest.NewRequest(http.MethodGet, "/client-keys?user_id="+uuid.New().String(), nil)

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		// Not forbidden
		assert.NotEqual(t, fiber.StatusForbidden, resp.StatusCode)
	})
}

// =============================================================================
// GetClientKey Handler Tests
// =============================================================================

func TestGetClientKey_Validation(t *testing.T) {
	t.Run("invalid client key ID", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewClientKeyHandler(nil)

		app.Get("/client-keys/:id", handler.GetClientKey)

		req := httptest.NewRequest(http.MethodGet, "/client-keys/invalid-uuid", nil)

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

		respBody, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal(respBody, &result)
		require.NoError(t, err)

		assert.Contains(t, result["error"], "Invalid client key ID")
	})

	t.Run("valid client key ID format", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewClientKeyHandler(nil)

		app.Get("/client-keys/:id", handler.GetClientKey)

		keyID := uuid.New().String()
		req := httptest.NewRequest(http.MethodGet, "/client-keys/"+keyID, nil)

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		// ID is valid, fails at service call
		assert.NotEqual(t, fiber.StatusBadRequest, resp.StatusCode)
	})
}

// =============================================================================
// UpdateClientKey Handler Tests
// =============================================================================

func TestUpdateClientKey_Validation(t *testing.T) {
	t.Run("invalid client key ID", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewClientKeyHandler(nil)

		app.Patch("/client-keys/:id", handler.UpdateClientKey)

		body := `{"name": "Updated"}`
		req := httptest.NewRequest(http.MethodPatch, "/client-keys/invalid-uuid", bytes.NewReader([]byte(body)))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

		respBody, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal(respBody, &result)
		require.NoError(t, err)

		assert.Contains(t, result["error"], "Invalid client key ID")
	})

	t.Run("invalid request body", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewClientKeyHandler(nil)

		app.Patch("/client-keys/:id", handler.UpdateClientKey)

		keyID := uuid.New().String()
		req := httptest.NewRequest(http.MethodPatch, "/client-keys/"+keyID, bytes.NewReader([]byte("invalid json")))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

		respBody, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal(respBody, &result)
		require.NoError(t, err)

		assert.Contains(t, result["error"], "Invalid request body")
	})

	t.Run("valid request", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewClientKeyHandler(nil)

		app.Patch("/client-keys/:id", handler.UpdateClientKey)

		keyID := uuid.New().String()
		body := `{"name": "Updated Key", "scopes": ["admin"]}`
		req := httptest.NewRequest(http.MethodPatch, "/client-keys/"+keyID, bytes.NewReader([]byte(body)))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		// Fails at service call, but validation passes
		assert.NotEqual(t, fiber.StatusBadRequest, resp.StatusCode)
	})
}

// =============================================================================
// RevokeClientKey Handler Tests
// =============================================================================

func TestRevokeClientKey_Validation(t *testing.T) {
	t.Run("invalid client key ID", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewClientKeyHandler(nil)

		app.Post("/client-keys/:id/revoke", handler.RevokeClientKey)

		req := httptest.NewRequest(http.MethodPost, "/client-keys/invalid-uuid/revoke", nil)

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

		respBody, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal(respBody, &result)
		require.NoError(t, err)

		assert.Contains(t, result["error"], "Invalid client key ID")
	})

	t.Run("valid client key ID", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewClientKeyHandler(nil)

		app.Post("/client-keys/:id/revoke", handler.RevokeClientKey)

		keyID := uuid.New().String()
		req := httptest.NewRequest(http.MethodPost, "/client-keys/"+keyID+"/revoke", nil)

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		// ID valid, fails at service call
		assert.NotEqual(t, fiber.StatusBadRequest, resp.StatusCode)
	})
}

// =============================================================================
// DeleteClientKey Handler Tests
// =============================================================================

func TestDeleteClientKey_Validation(t *testing.T) {
	t.Run("invalid client key ID", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewClientKeyHandler(nil)

		app.Delete("/client-keys/:id", handler.DeleteClientKey)

		req := httptest.NewRequest(http.MethodDelete, "/client-keys/invalid-uuid", nil)

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

		respBody, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal(respBody, &result)
		require.NoError(t, err)

		assert.Contains(t, result["error"], "Invalid client key ID")
	})

	t.Run("valid client key ID", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewClientKeyHandler(nil)

		app.Delete("/client-keys/:id", handler.DeleteClientKey)

		keyID := uuid.New().String()
		req := httptest.NewRequest(http.MethodDelete, "/client-keys/"+keyID, nil)

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		// ID valid, fails at service call
		assert.NotEqual(t, fiber.StatusBadRequest, resp.StatusCode)
	})
}

// =============================================================================
// Scope Tests
// =============================================================================

func TestClientKeyScopes(t *testing.T) {
	t.Run("common scope values", func(t *testing.T) {
		scopes := []string{
			"read:*",
			"write:*",
			"read:users",
			"write:users",
			"read:posts",
			"write:posts",
			"clientkeys:read",
			"clientkeys:write",
			"admin",
		}

		req := CreateClientKeyRequest{
			Name:   "Multi-scope Key",
			Scopes: scopes,
		}

		assert.Len(t, req.Scopes, len(scopes))
		for _, scope := range scopes {
			assert.Contains(t, req.Scopes, scope)
		}
	})

	t.Run("empty scopes array", func(t *testing.T) {
		req := CreateClientKeyRequest{
			Name:   "No Scopes Key",
			Scopes: []string{},
		}

		assert.Empty(t, req.Scopes)
	})
}

// =============================================================================
// Rate Limit Tests
// =============================================================================

func TestClientKeyRateLimits(t *testing.T) {
	t.Run("various rate limits", func(t *testing.T) {
		testCases := []struct {
			name      string
			rateLimit int
		}{
			{"zero rate limit", 0},
			{"low rate limit", 10},
			{"default rate limit", 60},
			{"high rate limit", 1000},
			{"very high rate limit", 10000},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				req := CreateClientKeyRequest{
					Name:               "Rate Limited Key",
					Scopes:             []string{"read:*"},
					RateLimitPerMinute: tc.rateLimit,
				}

				assert.Equal(t, tc.rateLimit, req.RateLimitPerMinute)
			})
		}
	})

	t.Run("update rate limit", func(t *testing.T) {
		rateLimit := 500
		req := UpdateClientKeyRequest{
			RateLimitPerMinute: &rateLimit,
		}

		assert.Equal(t, 500, *req.RateLimitPerMinute)
	})
}

// =============================================================================
// Admin Role Verification Tests
// =============================================================================

func TestAdminRoleVerification(t *testing.T) {
	adminRoles := []string{"admin", "instance_admin", "service_role"}
	nonAdminRoles := []string{"user", "authenticated", "anon", ""}

	t.Run("admin roles allow cross-user access", func(t *testing.T) {
		for _, role := range adminRoles {
			t.Run(role, func(t *testing.T) {
				app := newTestApp(t)
				handler := NewClientKeyHandler(nil)

				app.Use(func(c fiber.Ctx) error {
					c.Locals("user_id", uuid.New().String())
					c.Locals("user_role", role)
					return c.Next()
				})

				app.Get("/client-keys", handler.ListClientKeys)

				// Try to list another user's keys
				otherUserID := uuid.New().String()
				req := httptest.NewRequest(http.MethodGet, "/client-keys?user_id="+otherUserID, nil)

				resp, err := app.Test(req)
				require.NoError(t, err)
				defer func() { _ = resp.Body.Close() }()

				// Should not return forbidden for admin roles
				assert.NotEqual(t, fiber.StatusForbidden, resp.StatusCode)
			})
		}
	})

	t.Run("non-admin roles deny cross-user access", func(t *testing.T) {
		for _, role := range nonAdminRoles {
			t.Run(role, func(t *testing.T) {
				app := newTestApp(t)
				handler := NewClientKeyHandler(nil)

				app.Use(func(c fiber.Ctx) error {
					c.Locals("user_id", uuid.New().String())
					c.Locals("user_role", role)
					return c.Next()
				})

				app.Get("/client-keys", handler.ListClientKeys)

				// Try to list another user's keys
				otherUserID := uuid.New().String()
				req := httptest.NewRequest(http.MethodGet, "/client-keys?user_id="+otherUserID, nil)

				resp, err := app.Test(req)
				require.NoError(t, err)
				defer func() { _ = resp.Body.Close() }()

				// Should return forbidden for non-admin roles
				assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)
			})
		}
	})
}
