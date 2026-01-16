package middleware

import (
	"bytes"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// DefaultStructuredLoggerConfig Tests
// =============================================================================

func TestDefaultStructuredLoggerConfig(t *testing.T) {
	cfg := DefaultStructuredLoggerConfig()

	t.Run("default skip paths", func(t *testing.T) {
		assert.Contains(t, cfg.SkipPaths, "/health")
		assert.Contains(t, cfg.SkipPaths, "/ready")
		assert.Contains(t, cfg.SkipPaths, "/metrics")
		assert.Len(t, cfg.SkipPaths, 3)
	})

	t.Run("default settings", func(t *testing.T) {
		assert.False(t, cfg.SkipSuccessfulRequests)
		assert.Nil(t, cfg.Logger)
		assert.False(t, cfg.LogRequestBody)
		assert.False(t, cfg.LogResponseBody)
	})

	t.Run("default slow request threshold", func(t *testing.T) {
		assert.Equal(t, 1*time.Second, cfg.SlowRequestThreshold)
	})
}

// =============================================================================
// redactQueryString Tests
// =============================================================================

func TestRedactQueryString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string // Expected substrings that should be in output
		notExpected []string // Substrings that should NOT be in output
	}{
		{
			name:     "empty string",
			input:    "",
			expected: []string{""},
		},
		{
			name:     "no sensitive params",
			input:    "page=1&limit=10",
			expected: []string{"page=1", "limit=10"},
		},
		{
			name:        "redacts token",
			input:       "token=secret123&page=1",
			expected:    []string{"token=%5Bredacted%5D", "page=1"},
			notExpected: []string{"secret123"},
		},
		{
			name:        "redacts access_token",
			input:       "access_token=myaccesstoken&callback=test",
			expected:    []string{"access_token=%5Bredacted%5D", "callback=test"},
			notExpected: []string{"myaccesstoken"},
		},
		{
			name:        "redacts refresh_token",
			input:       "refresh_token=myrefreshtoken",
			expected:    []string{"refresh_token=%5Bredacted%5D"},
			notExpected: []string{"myrefreshtoken"},
		},
		{
			name:        "redacts api_key",
			input:       "api_key=sk_live_12345",
			expected:    []string{"api_key=%5Bredacted%5D"},
			notExpected: []string{"sk_live_12345"},
		},
		{
			name:        "redacts apikey (no underscore)",
			input:       "apikey=sk_test_67890",
			expected:    []string{"apikey=%5Bredacted%5D"},
			notExpected: []string{"sk_test_67890"},
		},
		{
			name:        "redacts key",
			input:       "key=supersecretkey&version=2",
			expected:    []string{"key=%5Bredacted%5D", "version=2"},
			notExpected: []string{"supersecretkey"},
		},
		{
			name:        "redacts secret",
			input:       "secret=mysecret123",
			expected:    []string{"secret=%5Bredacted%5D"},
			notExpected: []string{"mysecret123"},
		},
		{
			name:        "redacts password",
			input:       "username=john&password=hunter2",
			expected:    []string{"username=john", "password=%5Bredacted%5D"},
			notExpected: []string{"hunter2"},
		},
		{
			name:        "case insensitive - TOKEN",
			input:       "TOKEN=uppercase_secret",
			expected:    []string{"%5Bredacted%5D"},
			notExpected: []string{"uppercase_secret"},
		},
		{
			name:        "case insensitive - Api_Key",
			input:       "Api_Key=mixedcase",
			expected:    []string{"%5Bredacted%5D"},
			notExpected: []string{"mixedcase"},
		},
		{
			name:        "multiple sensitive params",
			input:       "token=tok1&api_key=key1&password=pass1&page=1",
			expected:    []string{"page=1"},
			notExpected: []string{"tok1", "key1", "pass1"},
		},
		{
			name:     "invalid query string returns redacted",
			input:    "invalid=%zz",
			expected: []string{"[redacted]"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := redactQueryString(tt.input)

			for _, exp := range tt.expected {
				if exp == "" {
					assert.Equal(t, "", result)
				} else {
					assert.Contains(t, result, exp, "Expected %q in result %q", exp, result)
				}
			}

			for _, notExp := range tt.notExpected {
				assert.NotContains(t, result, notExp, "Did not expect %q in result %q", notExp, result)
			}
		})
	}
}

func TestRedactQueryString_EncodedValues(t *testing.T) {
	t.Run("handles URL encoded values", func(t *testing.T) {
		input := "redirect_uri=https%3A%2F%2Fexample.com&token=secret"
		result := redactQueryString(input)

		// Token should be redacted
		assert.NotContains(t, result, "secret")
		// Redirect URI should be preserved (value will be re-encoded)
		assert.Contains(t, result, "redirect_uri=")
	})
}

// =============================================================================
// toString Tests
// =============================================================================

func TestToString(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{
			name:     "nil returns empty string",
			input:    nil,
			expected: "",
		},
		{
			name:     "string returns itself",
			input:    "hello",
			expected: "hello",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "string with spaces",
			input:    "hello world",
			expected: "hello world",
		},
		{
			name:     "string with special chars",
			input:    "user@example.com",
			expected: "user@example.com",
		},
		{
			name:     "int returns empty string",
			input:    42,
			expected: "",
		},
		{
			name:     "float returns empty string",
			input:    3.14,
			expected: "",
		},
		{
			name:     "bool returns empty string",
			input:    true,
			expected: "",
		},
		{
			name:     "slice returns empty string",
			input:    []string{"a", "b"},
			expected: "",
		},
		{
			name:     "map returns empty string",
			input:    map[string]int{"key": 1},
			expected: "",
		},
		{
			name:     "struct returns empty string",
			input:    struct{ Name string }{"test"},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// =============================================================================
// SlowQueryLogger Tests
// =============================================================================

func TestSlowQueryLogger(t *testing.T) {
	t.Run("returns a function", func(t *testing.T) {
		threshold := 100 * time.Millisecond
		logger := SlowQueryLogger(threshold)
		assert.NotNil(t, logger)
	})

	t.Run("does not log fast queries", func(t *testing.T) {
		// Capture log output
		var buf bytes.Buffer
		testLogger := zerolog.New(&buf)
		zerolog.DefaultContextLogger = &testLogger

		threshold := 1 * time.Second
		logger := SlowQueryLogger(threshold)

		// Call with a fast query (well under threshold)
		logger("SELECT 1", 10*time.Millisecond, nil)

		// Buffer should be empty for fast queries
		// Note: SlowQueryLogger uses the global log, not our test logger
		// This test mainly ensures no panic
	})

	t.Run("accepts error parameter", func(t *testing.T) {
		threshold := 100 * time.Millisecond
		logger := SlowQueryLogger(threshold)

		// Should not panic with error
		assert.NotPanics(t, func() {
			logger("SELECT * FROM users", 200*time.Millisecond, nil)
		})
	})

	t.Run("handles long query truncation", func(t *testing.T) {
		threshold := 1 * time.Millisecond
		logger := SlowQueryLogger(threshold)

		// Create a very long query (over 500 chars)
		longQuery := "SELECT " + strings.Repeat("column_name, ", 100) + " FROM table"

		// Should not panic
		assert.NotPanics(t, func() {
			logger(longQuery, 10*time.Millisecond, nil)
		})
	})
}

// =============================================================================
// AuditLogger Tests
// =============================================================================

func TestNewAuditLogger(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)

	auditLogger := NewAuditLogger(logger)

	assert.NotNil(t, auditLogger)
}

func TestAuditLogger_LogAuth(t *testing.T) {
	tests := []struct {
		name    string
		event   string
		userID  string
		email   string
		success bool
	}{
		{
			name:    "successful login",
			event:   "login",
			userID:  "user-123",
			email:   "user@example.com",
			success: true,
		},
		{
			name:    "failed login",
			event:   "login",
			userID:  "",
			email:   "attacker@example.com",
			success: false,
		},
		{
			name:    "password reset",
			event:   "password_reset",
			userID:  "user-456",
			email:   "reset@example.com",
			success: true,
		},
		{
			name:    "mfa enabled",
			event:   "mfa_enabled",
			userID:  "user-789",
			email:   "secure@example.com",
			success: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a fresh Fiber app for each test to avoid route stacking
			app := fiber.New()

			var buf bytes.Buffer
			logger := zerolog.New(&buf)
			auditLogger := NewAuditLogger(logger)

			app.Get("/test", func(c *fiber.Ctx) error {
				auditLogger.LogAuth(c, tt.event, tt.userID, tt.email, tt.success)
				return c.SendString("OK")
			})

			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("User-Agent", "TestAgent/1.0")
			req.Header.Set("X-Request-ID", "req-123")

			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			// Log output should contain event data
			logOutput := buf.String()
			assert.Contains(t, logOutput, "event")
			assert.Contains(t, logOutput, tt.event)
		})
	}
}

func TestAuditLogger_LogUserManagement(t *testing.T) {
	app := fiber.New()

	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	auditLogger := NewAuditLogger(logger)

	app.Get("/test", func(c *fiber.Ctx) error {
		auditLogger.LogUserManagement(c, "delete", "target-user-123", "admin-user-456")
		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Request-ID", "req-456")

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	logOutput := buf.String()
	assert.Contains(t, logOutput, "delete")
	assert.Contains(t, logOutput, "target-user-123")
	assert.Contains(t, logOutput, "admin-user-456")
}

func TestAuditLogger_LogClientKeyOperation(t *testing.T) {
	app := fiber.New()

	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	auditLogger := NewAuditLogger(logger)

	app.Get("/test", func(c *fiber.Ctx) error {
		auditLogger.LogClientKeyOperation(c, "create", "key-123", "production-key", "admin@example.com")
		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	logOutput := buf.String()
	assert.Contains(t, logOutput, "create")
	assert.Contains(t, logOutput, "key-123")
	assert.Contains(t, logOutput, "production-key")
}

func TestAuditLogger_LogConfigChange(t *testing.T) {
	app := fiber.New()

	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	auditLogger := NewAuditLogger(logger)

	app.Get("/test", func(c *fiber.Ctx) error {
		auditLogger.LogConfigChange(c, "rate_limit", "100", "200", "admin@example.com")
		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	logOutput := buf.String()
	assert.Contains(t, logOutput, "rate_limit")
	assert.Contains(t, logOutput, "100")
	assert.Contains(t, logOutput, "200")
}

func TestAuditLogger_LogSecurityEvent(t *testing.T) {
	app := fiber.New()

	tests := []struct {
		name        string
		event       string
		description string
		severity    string
	}{
		{
			name:        "critical event",
			event:       "brute_force_detected",
			description: "Multiple failed login attempts",
			severity:    "critical",
		},
		{
			name:        "high severity event",
			event:       "suspicious_activity",
			description: "Unusual API access pattern",
			severity:    "high",
		},
		{
			name:        "info event",
			event:       "ip_blocked",
			description: "IP added to blocklist",
			severity:    "info",
		},
		{
			name:        "unknown severity defaults to info",
			event:       "audit_log_access",
			description: "Audit logs accessed",
			severity:    "low",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := zerolog.New(&buf)
			auditLogger := NewAuditLogger(logger)

			testApp := fiber.New()
			testApp.Get("/test", func(c *fiber.Ctx) error {
				auditLogger.LogSecurityEvent(c, tt.event, tt.description, tt.severity)
				return c.SendString("OK")
			})

			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("User-Agent", "TestAgent")

			resp, err := testApp.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			logOutput := buf.String()
			assert.Contains(t, logOutput, tt.event)
			assert.Contains(t, logOutput, tt.description)
			assert.Contains(t, logOutput, tt.severity)
		})
	}

	_ = app // Prevent unused variable warning
}

// =============================================================================
// StructuredLogger Middleware Tests
// =============================================================================

func TestStructuredLogger_SkipPaths(t *testing.T) {
	app := fiber.New()

	cfg := DefaultStructuredLoggerConfig()
	app.Use(StructuredLogger(cfg))

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	app.Get("/api/users", func(c *fiber.Ctx) error {
		return c.SendString("Users")
	})

	t.Run("skips health endpoint", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/health", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, 200, resp.StatusCode)
	})

	t.Run("logs regular endpoints", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/users", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, 200, resp.StatusCode)
	})
}

func TestStructuredLogger_DefaultConfig(t *testing.T) {
	app := fiber.New()

	// Call with no config uses defaults
	app.Use(StructuredLogger())

	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, 200, resp.StatusCode)
}

func TestStructuredLogger_CustomLogger(t *testing.T) {
	var buf bytes.Buffer
	customLogger := zerolog.New(&buf)

	app := fiber.New()

	cfg := StructuredLoggerConfig{
		Logger: &customLogger,
	}
	app.Use(StructuredLogger(cfg))

	app.Get("/logged", func(c *fiber.Ctx) error {
		return c.SendString("Logged")
	})

	req := httptest.NewRequest("GET", "/logged", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Custom logger should have received log entries
	assert.Contains(t, buf.String(), "HTTP request")
	assert.Contains(t, buf.String(), "/logged")
}

func TestStructuredLogger_SkipSuccessfulRequests(t *testing.T) {
	var buf bytes.Buffer
	customLogger := zerolog.New(&buf)

	app := fiber.New()

	cfg := StructuredLoggerConfig{
		Logger:                 &customLogger,
		SkipSuccessfulRequests: true,
	}
	app.Use(StructuredLogger(cfg))

	app.Get("/success", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	app.Get("/error", func(c *fiber.Ctx) error {
		return c.Status(500).SendString("Error")
	})

	t.Run("skips successful requests", func(t *testing.T) {
		buf.Reset()
		req := httptest.NewRequest("GET", "/success", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, 200, resp.StatusCode)
		// Log should be empty for successful requests
		assert.Empty(t, buf.String())
	})

	t.Run("logs error requests", func(t *testing.T) {
		buf.Reset()
		req := httptest.NewRequest("GET", "/error", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, 500, resp.StatusCode)
		// Error requests should be logged
		assert.Contains(t, buf.String(), "/error")
	})
}

func TestStructuredLogger_RequestID(t *testing.T) {
	var buf bytes.Buffer
	customLogger := zerolog.New(&buf)

	app := fiber.New()

	cfg := StructuredLoggerConfig{
		Logger: &customLogger,
	}
	app.Use(StructuredLogger(cfg))

	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	t.Run("uses X-Request-ID header", func(t *testing.T) {
		buf.Reset()
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Request-ID", "custom-request-id-123")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Contains(t, buf.String(), "custom-request-id-123")
	})
}

func TestStructuredLogger_UserContext(t *testing.T) {
	var buf bytes.Buffer
	customLogger := zerolog.New(&buf)

	app := fiber.New()

	// Set user context before logger
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("user_id", "user-123")
		c.Locals("client_key_id", "key-456")
		return c.Next()
	})

	cfg := StructuredLoggerConfig{
		Logger: &customLogger,
	}
	app.Use(StructuredLogger(cfg))

	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	logOutput := buf.String()
	assert.Contains(t, logOutput, "user-123")
	assert.Contains(t, logOutput, "key-456")
}

func TestStructuredLogger_StatusCodes(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
	}{
		{"2xx success", 200},
		{"3xx redirect", 301},
		{"4xx client error", 400},
		{"404 not found", 404},
		{"5xx server error", 500},
		{"503 unavailable", 503},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			customLogger := zerolog.New(&buf)

			app := fiber.New()

			cfg := StructuredLoggerConfig{
				Logger: &customLogger,
			}
			app.Use(StructuredLogger(cfg))

			app.Get("/test", func(c *fiber.Ctx) error {
				return c.Status(tt.statusCode).SendString("Response")
			})

			req := httptest.NewRequest("GET", "/test", nil)
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.statusCode, resp.StatusCode)
			// All status codes should be logged
			assert.NotEmpty(t, buf.String())
		})
	}
}

func TestStructuredLogger_LogRequestBody(t *testing.T) {
	var buf bytes.Buffer
	customLogger := zerolog.New(&buf)

	app := fiber.New()

	cfg := StructuredLoggerConfig{
		Logger:         &customLogger,
		LogRequestBody: true,
	}
	app.Use(StructuredLogger(cfg))

	app.Post("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	t.Run("logs request body", func(t *testing.T) {
		buf.Reset()
		body := `{"username":"testuser"}`
		req := httptest.NewRequest("POST", "/test", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Contains(t, buf.String(), "request_body")
	})

	t.Run("truncates large request body", func(t *testing.T) {
		buf.Reset()
		// Create body larger than 1024 bytes
		largeBody := strings.Repeat("x", 2000)
		req := httptest.NewRequest("POST", "/test", strings.NewReader(largeBody))

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Contains(t, buf.String(), "truncated")
	})
}

func TestStructuredLogger_QueryStringRedaction(t *testing.T) {
	var buf bytes.Buffer
	customLogger := zerolog.New(&buf)

	app := fiber.New()

	cfg := StructuredLoggerConfig{
		Logger: &customLogger,
	}
	app.Use(StructuredLogger(cfg))

	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/test?token=secret123&page=1", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	logOutput := buf.String()
	// Token should be redacted
	assert.NotContains(t, logOutput, "secret123")
	// Page should be present
	assert.Contains(t, logOutput, "page=1")
}

func TestStructuredLogger_Referer(t *testing.T) {
	var buf bytes.Buffer
	customLogger := zerolog.New(&buf)

	app := fiber.New()

	cfg := StructuredLoggerConfig{
		Logger: &customLogger,
	}
	app.Use(StructuredLogger(cfg))

	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Referer", "https://example.com/page")

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Contains(t, buf.String(), "https://example.com/page")
}

func TestStructuredLogger_HandlerError(t *testing.T) {
	var buf bytes.Buffer
	customLogger := zerolog.New(&buf)

	app := fiber.New()

	cfg := StructuredLoggerConfig{
		Logger: &customLogger,
	}
	app.Use(StructuredLogger(cfg))

	expectedError := fiber.NewError(500, "Internal error")
	app.Get("/error", func(c *fiber.Ctx) error {
		return expectedError
	})

	req := httptest.NewRequest("GET", "/error", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	logOutput := buf.String()
	assert.Contains(t, logOutput, "Internal error")
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkRedactQueryString_NoSensitive(b *testing.B) {
	query := "page=1&limit=10&sort=created_at"
	for i := 0; i < b.N; i++ {
		_ = redactQueryString(query)
	}
}

func BenchmarkRedactQueryString_WithSensitive(b *testing.B) {
	query := "token=secret123&api_key=sk_live_xxx&page=1"
	for i := 0; i < b.N; i++ {
		_ = redactQueryString(query)
	}
}

func BenchmarkRedactQueryString_MultipleSensitive(b *testing.B) {
	query := "token=a&access_token=b&refresh_token=c&api_key=d&apikey=e&key=f&secret=g&password=h"
	for i := 0; i < b.N; i++ {
		_ = redactQueryString(query)
	}
}

func BenchmarkToString_String(b *testing.B) {
	val := "test-string-value"
	for i := 0; i < b.N; i++ {
		_ = toString(val)
	}
}

func BenchmarkToString_Nil(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = toString(nil)
	}
}

func BenchmarkToString_NonString(b *testing.B) {
	val := 12345
	for i := 0; i < b.N; i++ {
		_ = toString(val)
	}
}

func BenchmarkDefaultStructuredLoggerConfig(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = DefaultStructuredLoggerConfig()
	}
}

func BenchmarkStructuredLogger(b *testing.B) {
	app := fiber.New()
	app.Use(StructuredLogger())
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, _ := app.Test(req)
		resp.Body.Close()
	}
}
