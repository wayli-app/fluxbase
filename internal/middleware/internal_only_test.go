package middleware

import (
	"net"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

func TestIsLoopback(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		expected bool
	}{
		{"IPv4 localhost", "127.0.0.1", true},
		{"IPv4 loopback range", "127.0.0.2", true},
		{"IPv4 loopback range end", "127.255.255.255", true},
		{"IPv6 localhost", "::1", true},
		{"IPv4 external", "192.168.1.1", false},
		{"IPv4 public", "8.8.8.8", false},
		{"IPv6 external", "2001:db8::1", false},
		{"nil IP", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ip net.IP
			if tt.ip != "" {
				ip = net.ParseIP(tt.ip)
			}
			result := isLoopback(ip)
			assert.Equal(t, tt.expected, result, "IP: %s", tt.ip)
		})
	}
}

func TestRequireInternal(t *testing.T) {
	tests := []struct {
		name           string
		ip             string
		expectedStatus int
	}{
		{
			name:           "localhost IPv4 allowed",
			ip:             "127.0.0.1",
			expectedStatus: fiber.StatusOK,
		},
		{
			name:           "localhost IPv6 allowed",
			ip:             "::1",
			expectedStatus: fiber.StatusOK,
		},
		{
			name:           "external IPv4 denied",
			ip:             "192.168.1.100",
			expectedStatus: fiber.StatusForbidden,
		},
		{
			name:           "public IPv4 denied",
			ip:             "8.8.8.8",
			expectedStatus: fiber.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()

			// Create a mock IP extractor that returns the test IP
			// This is needed because Fiber's app.Test() doesn't properly propagate
			// req.RemoteAddr to fasthttp's context
			mockExtractor := func(_ *fiber.Ctx) net.IP {
				return net.ParseIP(tt.ip)
			}

			// Apply the middleware with mock extractor
			app.Use(RequireInternalWithExtractor(mockExtractor))

			// Add a test handler
			app.Get("/test", func(c *fiber.Ctx) error {
				return c.SendString("OK")
			})

			// Create request
			req := httptest.NewRequest("GET", "/test", nil)

			resp, err := app.Test(req)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
		})
	}
}

func TestRequireInternal_IgnoresProxyHeaders(t *testing.T) {
	app := fiber.New()

	// Create a mock IP extractor that simulates an external connection
	// The headers should be ignored - only the "connection" IP matters
	mockExtractor := func(_ *fiber.Ctx) net.IP {
		// Simulate external IP from real connection (not from headers)
		return net.ParseIP("192.168.1.100")
	}

	// Apply the middleware with mock extractor
	app.Use(RequireInternalWithExtractor(mockExtractor))

	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	// Request with spoofed X-Forwarded-For headers
	// These should be ignored since we use the mock extractor (simulating real getDirectIP)
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Forwarded-For", "127.0.0.1") // Spoofed localhost
	req.Header.Set("X-Real-IP", "127.0.0.1")       // Spoofed localhost

	resp, err := app.Test(req)
	assert.NoError(t, err)
	// Should be denied because we check actual connection IP, not headers
	assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)
}

// TestGetDirectIP tests the actual IP extraction function
func TestGetDirectIP(t *testing.T) {
	// This test verifies the getDirectIP function behavior.
	// Note: In Fiber's test mode, c.Context().RemoteIP() always returns 0.0.0.0
	// so this test primarily ensures the function doesn't panic and handles edge cases.
	app := fiber.New()

	var extractedIP net.IP
	app.Get("/test", func(c *fiber.Ctx) error {
		extractedIP = getDirectIP(c)
		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	_, err := app.Test(req)
	assert.NoError(t, err)

	// In test mode, we get 0.0.0.0 (unspecified) because there's no real connection
	// The function should return a valid IP object (even if it's the unspecified address)
	assert.NotNil(t, extractedIP)
}
