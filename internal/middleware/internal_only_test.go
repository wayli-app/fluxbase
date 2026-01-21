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
		remoteAddr     string
		expectedStatus int
	}{
		{
			name:           "localhost IPv4 allowed",
			remoteAddr:     "127.0.0.1:12345",
			expectedStatus: fiber.StatusOK,
		},
		{
			name:           "localhost IPv6 allowed",
			remoteAddr:     "[::1]:12345",
			expectedStatus: fiber.StatusOK,
		},
		{
			name:           "external IPv4 denied",
			remoteAddr:     "192.168.1.100:12345",
			expectedStatus: fiber.StatusForbidden,
		},
		{
			name:           "public IPv4 denied",
			remoteAddr:     "8.8.8.8:12345",
			expectedStatus: fiber.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()

			// Apply the middleware
			app.Use(RequireInternal())

			// Add a test handler
			app.Get("/test", func(c *fiber.Ctx) error {
				return c.SendString("OK")
			})

			// Create request
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = tt.remoteAddr

			resp, err := app.Test(req)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
		})
	}
}

func TestRequireInternal_IgnoresProxyHeaders(t *testing.T) {
	app := fiber.New()

	// Apply the middleware
	app.Use(RequireInternal())

	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	// Request from external IP but with spoofed X-Forwarded-For
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.100:12345"              // External IP
	req.Header.Set("X-Forwarded-For", "127.0.0.1")      // Spoofed localhost
	req.Header.Set("X-Real-IP", "127.0.0.1")            // Spoofed localhost

	resp, err := app.Test(req)
	assert.NoError(t, err)
	// Should be denied because we check actual connection IP, not headers
	assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)
}
