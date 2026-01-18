package middleware

import (
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestGenerateETag(t *testing.T) {
	body := []byte(`{"id": 1, "name": "test"}`)

	t.Run("weak ETag", func(t *testing.T) {
		etag := generateETag(body, true)
		if etag == "" {
			t.Error("Expected non-empty ETag")
		}
		if etag[:3] != `W/"` {
			t.Errorf("Expected weak ETag to start with W/\", got %s", etag)
		}
		if etag[len(etag)-1] != '"' {
			t.Errorf("Expected ETag to end with \", got %s", etag)
		}
	})

	t.Run("strong ETag", func(t *testing.T) {
		etag := generateETag(body, false)
		if etag == "" {
			t.Error("Expected non-empty ETag")
		}
		if etag[0] != '"' {
			t.Errorf("Expected strong ETag to start with \", got %s", etag)
		}
		if etag[len(etag)-1] != '"' {
			t.Errorf("Expected ETag to end with \", got %s", etag)
		}
	})

	t.Run("same body same ETag", func(t *testing.T) {
		etag1 := generateETag(body, true)
		etag2 := generateETag(body, true)
		if etag1 != etag2 {
			t.Errorf("Expected same body to produce same ETag, got %s and %s", etag1, etag2)
		}
	})

	t.Run("different body different ETag", func(t *testing.T) {
		body2 := []byte(`{"id": 2, "name": "test2"}`)
		etag1 := generateETag(body, true)
		etag2 := generateETag(body2, true)
		if etag1 == etag2 {
			t.Errorf("Expected different body to produce different ETag")
		}
	})
}

func TestEtagMatches(t *testing.T) {
	tests := []struct {
		name        string
		etag        string
		ifNoneMatch string
		expected    bool
	}{
		{
			name:        "exact match",
			etag:        `"abc123"`,
			ifNoneMatch: `"abc123"`,
			expected:    true,
		},
		{
			name:        "weak match",
			etag:        `W/"abc123"`,
			ifNoneMatch: `W/"abc123"`,
			expected:    true,
		},
		{
			name:        "weak vs strong match (weak comparison)",
			etag:        `W/"abc123"`,
			ifNoneMatch: `"abc123"`,
			expected:    true,
		},
		{
			name:        "strong vs weak match (weak comparison)",
			etag:        `"abc123"`,
			ifNoneMatch: `W/"abc123"`,
			expected:    true,
		},
		{
			name:        "no match",
			etag:        `"abc123"`,
			ifNoneMatch: `"xyz789"`,
			expected:    false,
		},
		{
			name:        "wildcard match",
			etag:        `"abc123"`,
			ifNoneMatch: `*`,
			expected:    true,
		},
		{
			name:        "multiple ETags - first matches",
			etag:        `"abc123"`,
			ifNoneMatch: `"abc123", "xyz789"`,
			expected:    true,
		},
		{
			name:        "multiple ETags - second matches",
			etag:        `"xyz789"`,
			ifNoneMatch: `"abc123", "xyz789"`,
			expected:    true,
		},
		{
			name:        "multiple ETags - none match",
			etag:        `"def456"`,
			ifNoneMatch: `"abc123", "xyz789"`,
			expected:    false,
		},
		{
			name:        "empty If-None-Match",
			etag:        `"abc123"`,
			ifNoneMatch: ``,
			expected:    false,
		},
		{
			name:        "whitespace handling",
			etag:        `"abc123"`,
			ifNoneMatch: `  "abc123"  `,
			expected:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := etagMatches(tt.etag, tt.ifNoneMatch)
			if result != tt.expected {
				t.Errorf("etagMatches(%q, %q) = %v, want %v", tt.etag, tt.ifNoneMatch, result, tt.expected)
			}
		})
	}
}

func TestNormalizeETag(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`"abc123"`, `"abc123"`},
		{`W/"abc123"`, `"abc123"`},
		{`  "abc123"  `, `"abc123"`},
		{`  W/"abc123"  `, `"abc123"`},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizeETag(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeETag(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestETagMiddleware(t *testing.T) {
	t.Run("adds ETag header to GET response", func(t *testing.T) {
		app := fiber.New()
		app.Use(ETag())
		app.Get("/test", func(c *fiber.Ctx) error {
			return c.JSON(fiber.Map{"id": 1, "name": "test"})
		})

		req := httptest.NewRequest("GET", "/test", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Failed to test request: %v", err)
		}

		etag := resp.Header.Get("ETag")
		if etag == "" {
			t.Error("Expected ETag header to be set")
		}
	})

	t.Run("skips ETag for non-GET methods", func(t *testing.T) {
		app := fiber.New()
		app.Use(ETag())
		app.Post("/test", func(c *fiber.Ctx) error {
			return c.JSON(fiber.Map{"id": 1})
		})

		req := httptest.NewRequest("POST", "/test", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Failed to test request: %v", err)
		}

		etag := resp.Header.Get("ETag")
		if etag != "" {
			t.Error("Expected no ETag header for POST")
		}
	})

	t.Run("returns 304 when ETag matches", func(t *testing.T) {
		app := fiber.New()
		app.Use(ETagWithConfig(ETagConfig{
			Weak:              true,
			EnableConditional: true,
		}))
		app.Get("/test", func(c *fiber.Ctx) error {
			return c.JSON(fiber.Map{"id": 1, "name": "test"})
		})

		// First request to get ETag
		req1 := httptest.NewRequest("GET", "/test", nil)
		resp1, _ := app.Test(req1)
		etag := resp1.Header.Get("ETag")

		// Second request with If-None-Match
		req2 := httptest.NewRequest("GET", "/test", nil)
		req2.Header.Set("If-None-Match", etag)
		resp2, err := app.Test(req2)
		if err != nil {
			t.Fatalf("Failed to test request: %v", err)
		}

		if resp2.StatusCode != 304 {
			t.Errorf("Expected 304 status, got %d", resp2.StatusCode)
		}

		// Body should be empty
		body, _ := io.ReadAll(resp2.Body)
		if len(body) > 0 {
			t.Errorf("Expected empty body for 304, got %s", string(body))
		}
	})

	t.Run("returns full response when ETag doesn't match", func(t *testing.T) {
		app := fiber.New()
		app.Use(ETag())
		app.Get("/test", func(c *fiber.Ctx) error {
			return c.JSON(fiber.Map{"id": 1, "name": "test"})
		})

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("If-None-Match", `"non-matching-etag"`)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Failed to test request: %v", err)
		}

		if resp.StatusCode != 200 {
			t.Errorf("Expected 200 status, got %d", resp.StatusCode)
		}

		body, _ := io.ReadAll(resp.Body)
		if len(body) == 0 {
			t.Error("Expected non-empty body")
		}
	})

	t.Run("skips configured paths", func(t *testing.T) {
		app := fiber.New()
		app.Use(ETagWithConfig(ETagConfig{
			Weak:      true,
			SkipPaths: []string{"/health"},
		}))
		app.Get("/health", func(c *fiber.Ctx) error {
			return c.JSON(fiber.Map{"status": "ok"})
		})

		req := httptest.NewRequest("GET", "/health", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Failed to test request: %v", err)
		}

		etag := resp.Header.Get("ETag")
		if etag != "" {
			t.Error("Expected no ETag for skipped path")
		}
	})

	t.Run("no ETag for error responses", func(t *testing.T) {
		app := fiber.New()
		app.Use(ETag())
		app.Get("/error", func(c *fiber.Ctx) error {
			return c.Status(500).JSON(fiber.Map{"error": "internal error"})
		})

		req := httptest.NewRequest("GET", "/error", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Failed to test request: %v", err)
		}

		etag := resp.Header.Get("ETag")
		if etag != "" {
			t.Error("Expected no ETag for error response")
		}
	})
}

func TestCacheControlMiddleware(t *testing.T) {
	t.Run("sets max-age", func(t *testing.T) {
		app := fiber.New()
		app.Use(CacheControl(CacheControlConfig{MaxAge: 3600}))
		app.Get("/test", func(c *fiber.Ctx) error {
			return c.JSON(fiber.Map{"data": "test"})
		})

		req := httptest.NewRequest("GET", "/test", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Failed to test request: %v", err)
		}

		cc := resp.Header.Get("Cache-Control")
		if cc == "" {
			t.Error("Expected Cache-Control header")
		}
		if cc != "public, max-age=3600" {
			t.Errorf("Expected 'public, max-age=3600', got %q", cc)
		}
	})

	t.Run("sets no-store", func(t *testing.T) {
		app := fiber.New()
		app.Use(CacheControl(CacheControlConfig{NoStore: true}))
		app.Get("/test", func(c *fiber.Ctx) error {
			return c.JSON(fiber.Map{"data": "test"})
		})

		req := httptest.NewRequest("GET", "/test", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Failed to test request: %v", err)
		}

		cc := resp.Header.Get("Cache-Control")
		if cc != "no-store" {
			t.Errorf("Expected 'no-store', got %q", cc)
		}
	})

	t.Run("sets private with max-age", func(t *testing.T) {
		app := fiber.New()
		app.Use(CacheControl(CacheControlConfig{
			Private:        true,
			MaxAge:         300,
			MustRevalidate: true,
		}))
		app.Get("/test", func(c *fiber.Ctx) error {
			return c.JSON(fiber.Map{"data": "test"})
		})

		req := httptest.NewRequest("GET", "/test", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Failed to test request: %v", err)
		}

		cc := resp.Header.Get("Cache-Control")
		if cc != "private, max-age=300, must-revalidate" {
			t.Errorf("Expected 'private, max-age=300, must-revalidate', got %q", cc)
		}
	})

	t.Run("skips non-GET methods", func(t *testing.T) {
		app := fiber.New()
		app.Use(CacheControl(CacheControlConfig{MaxAge: 3600}))
		app.Post("/test", func(c *fiber.Ctx) error {
			return c.JSON(fiber.Map{"data": "test"})
		})

		req := httptest.NewRequest("POST", "/test", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Failed to test request: %v", err)
		}

		cc := resp.Header.Get("Cache-Control")
		if cc != "" {
			t.Error("Expected no Cache-Control for POST")
		}
	})
}

func TestItoa(t *testing.T) {
	tests := []struct {
		input    int
		expected string
	}{
		{0, "0"},
		{1, "1"},
		{10, "10"},
		{100, "100"},
		{123, "123"},
		{3600, "3600"},
		{-1, "-1"},
		{-100, "-100"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := itoa(tt.input)
			if result != tt.expected {
				t.Errorf("itoa(%d) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
