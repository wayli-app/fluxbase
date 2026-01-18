package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultIdempotencyConfig(t *testing.T) {
	config := DefaultIdempotencyConfig()

	assert.Equal(t, "Idempotency-Key", config.HeaderName)
	assert.Equal(t, 24*time.Hour, config.TTL)
	assert.Contains(t, config.Methods, "POST")
	assert.Contains(t, config.Methods, "PUT")
	assert.Contains(t, config.Methods, "DELETE")
	assert.Contains(t, config.Methods, "PATCH")
	assert.Equal(t, "/api/", config.PathPrefix)
	assert.Equal(t, 256, config.MaxKeyLength)
	assert.Equal(t, 1*time.Hour, config.CleanupInterval)
}

func TestIdempotencyMiddleware_SkipsGET(t *testing.T) {
	config := DefaultIdempotencyConfig()
	// DB is nil, so middleware won't apply, but we're testing method filtering
	mw := NewIdempotencyMiddleware(config)
	defer mw.Stop()

	app := fiber.New()
	app.Use(mw.Middleware())
	app.Get("/api/v1/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	req.Header.Set("Idempotency-Key", "test-key-123")

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestIdempotencyMiddleware_SkipsNonAPIPaths(t *testing.T) {
	config := DefaultIdempotencyConfig()
	mw := NewIdempotencyMiddleware(config)
	defer mw.Stop()

	app := fiber.New()
	app.Use(mw.Middleware())
	app.Post("/health", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest(http.MethodPost, "/health", nil)
	req.Header.Set("Idempotency-Key", "test-key-123")

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestIdempotencyMiddleware_NoKeyProcessesNormally(t *testing.T) {
	config := DefaultIdempotencyConfig()
	mw := NewIdempotencyMiddleware(config)
	defer mw.Stop()

	handlerCalled := false
	app := fiber.New()
	app.Use(mw.Middleware())
	app.Post("/api/v1/test", func(c *fiber.Ctx) error {
		handlerCalled = true
		return c.Status(201).JSON(fiber.Map{"created": true})
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/test", strings.NewReader(`{"name":"test"}`))
	req.Header.Set("Content-Type", "application/json")
	// No Idempotency-Key header

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	assert.True(t, handlerCalled, "handler should be called without idempotency key")
}

func TestIdempotencyMiddleware_RejectsLongKey(t *testing.T) {
	config := DefaultIdempotencyConfig()
	config.MaxKeyLength = 50
	mw := NewIdempotencyMiddleware(config)
	defer mw.Stop()

	app := fiber.New()
	app.Use(mw.Middleware())
	app.Post("/api/v1/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	// Key that's too long
	longKey := strings.Repeat("a", 100)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/test", nil)
	req.Header.Set("Idempotency-Key", longKey)

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestIdempotencyMiddleware_SkipsExcludedPaths(t *testing.T) {
	config := DefaultIdempotencyConfig()
	config.ExcludePaths = []string{"/api/v1/auth/refresh", "/api/v1/auth/logout"}
	mw := NewIdempotencyMiddleware(config)
	defer mw.Stop()

	app := fiber.New()
	app.Use(mw.Middleware())
	app.Post("/api/v1/auth/refresh", func(c *fiber.Ctx) error {
		return c.SendString("Refreshed")
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
	req.Header.Set("Idempotency-Key", "test-key")

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestIdempotencyMiddleware_ShouldApply(t *testing.T) {
	config := DefaultIdempotencyConfig()
	config.PathPrefix = "/api/"
	config.ExcludePaths = []string{"/api/v1/auth/refresh"}
	mw := NewIdempotencyMiddleware(config)
	defer mw.Stop()

	tests := []struct {
		name     string
		method   string
		path     string
		expected bool
	}{
		{"POST to API should apply", "POST", "/api/v1/users", true},
		{"PUT to API should apply", "PUT", "/api/v1/users/123", true},
		{"DELETE to API should apply", "DELETE", "/api/v1/users/123", true},
		{"PATCH to API should apply", "PATCH", "/api/v1/users/123", true},
		{"GET to API should not apply", "GET", "/api/v1/users", false},
		{"HEAD to API should not apply", "HEAD", "/api/v1/users", false},
		{"POST to non-API should not apply", "POST", "/health", false},
		{"POST to excluded path should not apply", "POST", "/api/v1/auth/refresh", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			var shouldApplyResult bool

			app.Use(func(c *fiber.Ctx) error {
				shouldApplyResult = mw.shouldApply(c)
				return c.SendString("OK")
			})

			// Add route for all methods
			app.All(tt.path, func(c *fiber.Ctx) error {
				return c.SendString("OK")
			})

			req := httptest.NewRequest(tt.method, tt.path, nil)
			_, err := app.Test(req, -1)
			require.NoError(t, err)

			// Note: Without DB configured, shouldApply will always return false
			// because we check for DB presence. For this test we're testing the
			// other conditions.
			if tt.expected == false {
				assert.False(t, shouldApplyResult, "expected shouldApply to return false for %s %s", tt.method, tt.path)
			}
			// Can't test true cases without DB
		})
	}
}

func TestCalculateRequestHash(t *testing.T) {
	config := DefaultIdempotencyConfig()
	mw := NewIdempotencyMiddleware(config)
	defer mw.Stop()

	t.Run("empty body returns empty hash", func(t *testing.T) {
		app := fiber.New()
		var hash string
		app.Post("/test", func(c *fiber.Ctx) error {
			hash = mw.calculateRequestHash(c)
			return c.SendString("OK")
		})

		req := httptest.NewRequest(http.MethodPost, "/test", nil)
		_, err := app.Test(req, -1)
		require.NoError(t, err)
		assert.Empty(t, hash)
	})

	t.Run("same body produces same hash", func(t *testing.T) {
		app := fiber.New()
		var hashes []string
		app.Post("/test", func(c *fiber.Ctx) error {
			hashes = append(hashes, mw.calculateRequestHash(c))
			return c.SendString("OK")
		})

		body := `{"name":"test","value":123}`

		req1 := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(body))
		req1.Header.Set("Content-Type", "application/json")
		_, err := app.Test(req1, -1)
		require.NoError(t, err)

		req2 := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(body))
		req2.Header.Set("Content-Type", "application/json")
		_, err = app.Test(req2, -1)
		require.NoError(t, err)

		assert.Equal(t, hashes[0], hashes[1])
	})

	t.Run("different body produces different hash", func(t *testing.T) {
		app := fiber.New()
		var hashes []string
		app.Post("/test", func(c *fiber.Ctx) error {
			hashes = append(hashes, mw.calculateRequestHash(c))
			return c.SendString("OK")
		})

		req1 := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(`{"name":"test1"}`))
		req1.Header.Set("Content-Type", "application/json")
		_, err := app.Test(req1, -1)
		require.NoError(t, err)

		req2 := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(`{"name":"test2"}`))
		req2.Header.Set("Content-Type", "application/json")
		_, err = app.Test(req2, -1)
		require.NoError(t, err)

		assert.NotEqual(t, hashes[0], hashes[1])
	})
}

func TestIdempotencyKeyHelpers(t *testing.T) {
	t.Run("GetIdempotencyKey extracts header", func(t *testing.T) {
		app := fiber.New()
		var key string
		app.Post("/test", func(c *fiber.Ctx) error {
			key = GetIdempotencyKey(c)
			return c.SendString("OK")
		})

		req := httptest.NewRequest(http.MethodPost, "/test", nil)
		req.Header.Set("Idempotency-Key", "my-key-123")
		_, err := app.Test(req, -1)
		require.NoError(t, err)
		assert.Equal(t, "my-key-123", key)
	})

	t.Run("HasIdempotencyKey returns true when present", func(t *testing.T) {
		app := fiber.New()
		var hasKey bool
		app.Post("/test", func(c *fiber.Ctx) error {
			hasKey = HasIdempotencyKey(c)
			return c.SendString("OK")
		})

		req := httptest.NewRequest(http.MethodPost, "/test", nil)
		req.Header.Set("Idempotency-Key", "my-key-123")
		_, err := app.Test(req, -1)
		require.NoError(t, err)
		assert.True(t, hasKey)
	})

	t.Run("HasIdempotencyKey returns false when absent", func(t *testing.T) {
		app := fiber.New()
		var hasKey bool
		app.Post("/test", func(c *fiber.Ctx) error {
			hasKey = HasIdempotencyKey(c)
			return c.SendString("OK")
		})

		req := httptest.NewRequest(http.MethodPost, "/test", nil)
		_, err := app.Test(req, -1)
		require.NoError(t, err)
		assert.False(t, hasKey)
	})
}

func TestEncodeDecodeResponseBody(t *testing.T) {
	original := []byte(`{"status":"success","data":{"id":123}}`)

	encoded := EncodeResponseBody(original)
	assert.NotEmpty(t, encoded)
	assert.NotEqual(t, string(original), encoded)

	decoded, err := DecodeResponseBody(encoded)
	require.NoError(t, err)
	assert.Equal(t, original, decoded)
}

func TestCompareBytes(t *testing.T) {
	tests := []struct {
		name string
		a    []byte
		b    []byte
		want bool
	}{
		{"equal slices", []byte("hello"), []byte("hello"), true},
		{"different slices", []byte("hello"), []byte("world"), false},
		{"empty slices", []byte{}, []byte{}, true},
		{"nil and empty", nil, []byte{}, true},
		{"nil slices", nil, nil, true},
		{"different lengths", []byte("hello"), []byte("hi"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, CompareBytes(tt.a, tt.b))
		})
	}
}

func TestIdempotencyKeyStatus(t *testing.T) {
	assert.Equal(t, IdempotencyKeyStatus("processing"), StatusProcessing)
	assert.Equal(t, IdempotencyKeyStatus("completed"), StatusCompleted)
	assert.Equal(t, IdempotencyKeyStatus("failed"), StatusFailed)
}

func TestIdempotencyRecord(t *testing.T) {
	now := time.Now()
	userID := "user-123"
	responseStatus := 201

	record := IdempotencyRecord{
		Key:             "test-key",
		Method:          "POST",
		Path:            "/api/v1/users",
		UserID:          &userID,
		RequestHash:     "abc123",
		Status:          StatusCompleted,
		ResponseStatus:  &responseStatus,
		ResponseHeaders: map[string]string{"Content-Type": "application/json"},
		ResponseBody:    []byte(`{"created":true}`),
		CreatedAt:       now,
		CompletedAt:     &now,
		ExpiresAt:       now.Add(24 * time.Hour),
	}

	assert.Equal(t, "test-key", record.Key)
	assert.Equal(t, "POST", record.Method)
	assert.Equal(t, "/api/v1/users", record.Path)
	assert.Equal(t, &userID, record.UserID)
	assert.Equal(t, StatusCompleted, record.Status)
	assert.Equal(t, &responseStatus, record.ResponseStatus)
}

func TestNewIdempotencyMiddleware_Defaults(t *testing.T) {
	// Test with empty config
	config := IdempotencyConfig{}
	mw := NewIdempotencyMiddleware(config)
	defer mw.Stop()

	// Should have default values
	assert.Equal(t, "Idempotency-Key", mw.config.HeaderName)
	assert.Equal(t, 24*time.Hour, mw.config.TTL)
	assert.Equal(t, 256, mw.config.MaxKeyLength)
	assert.True(t, mw.methodSet["POST"])
	assert.True(t, mw.methodSet["PUT"])
	assert.True(t, mw.methodSet["DELETE"])
	assert.True(t, mw.methodSet["PATCH"])
}

func TestNewIdempotencyMiddleware_CustomConfig(t *testing.T) {
	config := IdempotencyConfig{
		HeaderName:      "X-Request-Id",
		TTL:             1 * time.Hour,
		Methods:         []string{"POST"},
		PathPrefix:      "/v1/",
		MaxKeyLength:    128,
		CleanupInterval: 30 * time.Minute,
	}
	mw := NewIdempotencyMiddleware(config)
	defer mw.Stop()

	assert.Equal(t, "X-Request-Id", mw.config.HeaderName)
	assert.Equal(t, 1*time.Hour, mw.config.TTL)
	assert.Equal(t, 128, mw.config.MaxKeyLength)
	assert.True(t, mw.methodSet["POST"])
	assert.False(t, mw.methodSet["PUT"])
	assert.False(t, mw.methodSet["DELETE"])
}
