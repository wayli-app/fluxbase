package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPatternBodyLimiter_GetLimit(t *testing.T) {
	config := BodyLimitConfig{
		DefaultLimit: 1024,
		Patterns: []BodyLimitPattern{
			{Pattern: "/api/v1/storage/**", Limit: 100 * 1024 * 1024, Description: "storage"},
			{Pattern: "/api/v1/auth/**", Limit: 64 * 1024, Description: "auth"},
			{Pattern: "/api/v1/rest/*", Limit: 1024 * 1024, Description: "REST"},
			{Pattern: "/api/v1/rest/*/bulk", Limit: 10 * 1024 * 1024, Description: "bulk"},
		},
	}

	limiter := NewPatternBodyLimiter(config)

	tests := []struct {
		name        string
		path        string
		wantLimit   int64
		wantDesc    string
	}{
		{
			name:      "storage upload",
			path:      "/api/v1/storage/bucket/file.txt",
			wantLimit: 100 * 1024 * 1024,
			wantDesc:  "storage",
		},
		{
			name:      "storage nested path",
			path:      "/api/v1/storage/bucket/folder/subfolder/file.txt",
			wantLimit: 100 * 1024 * 1024,
			wantDesc:  "storage",
		},
		{
			name:      "auth endpoint",
			path:      "/api/v1/auth/login",
			wantLimit: 64 * 1024,
			wantDesc:  "auth",
		},
		{
			name:      "auth nested",
			path:      "/api/v1/auth/2fa/verify",
			wantLimit: 64 * 1024,
			wantDesc:  "auth",
		},
		{
			name:      "REST endpoint single segment",
			path:      "/api/v1/rest/users",
			wantLimit: 1024 * 1024,
			wantDesc:  "REST",
		},
		{
			name:      "bulk operation - more specific match",
			path:      "/api/v1/rest/users/bulk",
			wantLimit: 10 * 1024 * 1024,
			wantDesc:  "bulk",
		},
		{
			name:      "unmatched path uses default",
			path:      "/health",
			wantLimit: 1024,
			wantDesc:  "default",
		},
		{
			name:      "unmatched nested path",
			path:      "/other/endpoint/deep/path",
			wantLimit: 1024,
			wantDesc:  "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limit, desc := limiter.GetLimit(tt.path)
			assert.Equal(t, tt.wantLimit, limit, "limit mismatch for path %s", tt.path)
			assert.Equal(t, tt.wantDesc, desc, "description mismatch for path %s", tt.path)
		})
	}
}

func TestPatternBodyLimiter_Middleware_AcceptsUnderLimit(t *testing.T) {
	config := BodyLimitConfig{
		DefaultLimit: 1024, // 1KB default
		Patterns: []BodyLimitPattern{
			{Pattern: "/api/**", Limit: 1024, Description: "API"},
		},
	}

	app := fiber.New()
	limiter := NewPatternBodyLimiter(config)
	app.Use(limiter.Middleware())
	app.Post("/api/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	// Request with body under limit
	body := bytes.Repeat([]byte("a"), 500) // 500 bytes
	req := httptest.NewRequest(http.MethodPost, "/api/test", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.ContentLength = int64(len(body))

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestPatternBodyLimiter_Middleware_RejectsOverLimit(t *testing.T) {
	config := BodyLimitConfig{
		DefaultLimit: 1024, // 1KB default
		Patterns: []BodyLimitPattern{
			{Pattern: "/api/**", Limit: 1024, Description: "API"},
		},
	}

	app := fiber.New()
	limiter := NewPatternBodyLimiter(config)
	app.Use(limiter.Middleware())
	app.Post("/api/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	// Request with body over limit
	body := bytes.Repeat([]byte("a"), 2048) // 2KB
	req := httptest.NewRequest(http.MethodPost, "/api/test", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.ContentLength = int64(len(body))

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusRequestEntityTooLarge, resp.StatusCode)
}

func TestPatternBodyLimiter_Middleware_SkipsGET(t *testing.T) {
	config := BodyLimitConfig{
		DefaultLimit: 100, // Very small limit
		Patterns:     []BodyLimitPattern{},
	}

	app := fiber.New()
	limiter := NewPatternBodyLimiter(config)
	app.Use(limiter.Middleware())
	app.Get("/api/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestPatternBodyLimiter_DifferentEndpointsDifferentLimits(t *testing.T) {
	config := BodyLimitConfig{
		DefaultLimit: 1024,
		Patterns: []BodyLimitPattern{
			{Pattern: "/api/v1/storage/**", Limit: 10 * 1024, Description: "storage"},
			{Pattern: "/api/v1/auth/**", Limit: 512, Description: "auth"},
		},
	}

	app := fiber.New()
	limiter := NewPatternBodyLimiter(config)
	app.Use(limiter.Middleware())
	app.Post("/api/v1/storage/upload", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})
	app.Post("/api/v1/auth/login", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	// Storage endpoint should accept 5KB
	storageBody := bytes.Repeat([]byte("a"), 5*1024)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/storage/upload", bytes.NewReader(storageBody))
	req.ContentLength = int64(len(storageBody))
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "storage should accept 5KB")

	// Auth endpoint should reject 1KB
	authBody := bytes.Repeat([]byte("a"), 1024)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(authBody))
	req.ContentLength = int64(len(authBody))
	resp, err = app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusRequestEntityTooLarge, resp.StatusCode, "auth should reject 1KB")
}

func TestJSONDepthLimiter_AcceptsShallowJSON(t *testing.T) {
	app := fiber.New()
	limiter := NewJSONDepthLimiter(5)
	app.Use(limiter.Middleware())
	app.Post("/api/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	// Shallow JSON (depth 2)
	body := `{"user": {"name": "test"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/test", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestJSONDepthLimiter_RejectsDeepJSON(t *testing.T) {
	app := fiber.New()
	limiter := NewJSONDepthLimiter(3)
	app.Use(limiter.Middleware())
	app.Post("/api/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	// Deep JSON (depth 5)
	body := `{"a": {"b": {"c": {"d": {"e": "value"}}}}}`
	req := httptest.NewRequest(http.MethodPost, "/api/test", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestJSONDepthLimiter_SkipsNonJSON(t *testing.T) {
	app := fiber.New()
	limiter := NewJSONDepthLimiter(1) // Very strict
	app.Use(limiter.Middleware())
	app.Post("/api/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	// Non-JSON content
	body := "this is plain text"
	req := httptest.NewRequest(http.MethodPost, "/api/test", strings.NewReader(body))
	req.Header.Set("Content-Type", "text/plain")

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestJSONDepthLimiter_SkipsGETRequests(t *testing.T) {
	app := fiber.New()
	limiter := NewJSONDepthLimiter(1)
	app.Use(limiter.Middleware())
	app.Get("/api/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestJSONDepthLimiter_HandlesArrays(t *testing.T) {
	app := fiber.New()
	limiter := NewJSONDepthLimiter(3)
	app.Use(limiter.Middleware())
	app.Post("/api/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	// Nested arrays (depth 4)
	body := `[[[["deep"]]]]`
	req := httptest.NewRequest(http.MethodPost, "/api/test", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestCheckJSONDepth(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		maxDepth int
		wantErr  bool
		wantMax  int
	}{
		{
			name:     "empty object",
			json:     `{}`,
			maxDepth: 10,
			wantErr:  false,
			wantMax:  1,
		},
		{
			name:     "nested object within limit",
			json:     `{"a": {"b": "c"}}`,
			maxDepth: 3,
			wantErr:  false,
			wantMax:  2,
		},
		{
			name:     "nested object exceeds limit",
			json:     `{"a": {"b": {"c": "d"}}}`,
			maxDepth: 2,
			wantErr:  true,
			wantMax:  3,
		},
		{
			name:     "array within limit",
			json:     `[[1, 2, 3]]`,
			maxDepth: 3,
			wantErr:  false,
			wantMax:  2,
		},
		{
			name:     "mixed nesting",
			json:     `{"arr": [{"nested": true}]}`,
			maxDepth: 5,
			wantErr:  false,
			wantMax:  3,
		},
		{
			name:     "deeply nested array",
			json:     `[[[[[[[[[[1]]]]]]]]]]`,
			maxDepth: 5,
			wantErr:  true,
			wantMax:  6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			depth, err := checkJSONDepth([]byte(tt.json), tt.maxDepth)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.wantMax, depth)
		})
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		bytes int64
		want  string
	}{
		{500, "500 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{10485760, "10.0 MB"},
		{1073741824, "1.0 GB"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := formatBytes(tt.bytes)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDefaultBodyLimitConfig(t *testing.T) {
	config := DefaultBodyLimitConfig()

	assert.Equal(t, DefaultBodyLimit, config.DefaultLimit)
	assert.Equal(t, DefaultMaxJSONDepth, config.MaxJSONDepth)
	assert.NotEmpty(t, config.Patterns)

	// Verify some key patterns exist
	hasStoragePattern := false
	hasAuthPattern := false
	hasRESTPattern := false

	for _, p := range config.Patterns {
		if strings.Contains(p.Pattern, "storage") {
			hasStoragePattern = true
		}
		if strings.Contains(p.Pattern, "auth") {
			hasAuthPattern = true
		}
		if strings.Contains(p.Pattern, "rest") {
			hasRESTPattern = true
		}
	}

	assert.True(t, hasStoragePattern, "should have storage pattern")
	assert.True(t, hasAuthPattern, "should have auth pattern")
	assert.True(t, hasRESTPattern, "should have REST pattern")
}

func TestBodyLimitMiddleware_Combined(t *testing.T) {
	config := BodyLimitConfig{
		DefaultLimit: 10 * 1024, // 10KB
		Patterns:     []BodyLimitPattern{},
		MaxJSONDepth: 3,
	}

	app := fiber.New()
	app.Use(BodyLimitMiddleware(config))
	app.Post("/api/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	t.Run("accepts valid request", func(t *testing.T) {
		body := `{"name": "test"}`
		req := httptest.NewRequest(http.MethodPost, "/api/test", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.ContentLength = int64(len(body))

		resp, err := app.Test(req, -1)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("rejects oversized body", func(t *testing.T) {
		body := bytes.Repeat([]byte("a"), 20*1024) // 20KB
		req := httptest.NewRequest(http.MethodPost, "/api/test", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.ContentLength = int64(len(body))

		resp, err := app.Test(req, -1)
		require.NoError(t, err)
		assert.Equal(t, http.StatusRequestEntityTooLarge, resp.StatusCode)
	})

	t.Run("rejects deep JSON", func(t *testing.T) {
		body := `{"a": {"b": {"c": {"d": "value"}}}}`
		req := httptest.NewRequest(http.MethodPost, "/api/test", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

func TestPatternMatching_EdgeCases(t *testing.T) {
	config := BodyLimitConfig{
		DefaultLimit: 100,
		Patterns: []BodyLimitPattern{
			{Pattern: "/api/v1/exact", Limit: 200, Description: "exact"},
			{Pattern: "/api/v1/wild/*", Limit: 300, Description: "single wild"},
			{Pattern: "/api/v1/double/**", Limit: 400, Description: "double wild"},
			{Pattern: "/api/v1/mixed/*/end", Limit: 500, Description: "mixed"},
			{Pattern: "/api/v1/complex/**/final", Limit: 600, Description: "complex"},
		},
	}

	limiter := NewPatternBodyLimiter(config)

	tests := []struct {
		path      string
		wantLimit int64
	}{
		// Exact match
		{"/api/v1/exact", 200},
		{"/api/v1/exact/extra", 100}, // No match - has extra segment

		// Single wildcard
		{"/api/v1/wild/anything", 300},
		{"/api/v1/wild/anything/more", 100}, // No match - too many segments

		// Double wildcard
		{"/api/v1/double/", 400},
		{"/api/v1/double/one", 400},
		{"/api/v1/double/one/two", 400},
		{"/api/v1/double/one/two/three/four", 400},

		// Mixed pattern
		{"/api/v1/mixed/anything/end", 500},
		{"/api/v1/mixed/something/end", 500},
		{"/api/v1/mixed/x/y/end", 100}, // No match - * only matches one segment

		// Complex pattern with ** in middle
		{"/api/v1/complex/final", 600},
		{"/api/v1/complex/a/final", 600},
		{"/api/v1/complex/a/b/c/final", 600},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			limit, _ := limiter.GetLimit(tt.path)
			assert.Equal(t, tt.wantLimit, limit, "limit mismatch for %s", tt.path)
		})
	}
}
