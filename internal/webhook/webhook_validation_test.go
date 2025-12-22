package webhook

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsPrivateIP(t *testing.T) {
	testCases := []struct {
		name      string
		ip        string
		isPrivate bool
	}{
		// Loopback addresses
		{"IPv4 loopback 127.0.0.1", "127.0.0.1", true},
		{"IPv4 loopback 127.0.0.2", "127.0.0.2", true},
		{"IPv4 loopback 127.255.255.255", "127.255.255.255", true},
		{"IPv6 loopback", "::1", true},

		// RFC 1918 private ranges
		{"10.0.0.0/8 start", "10.0.0.0", true},
		{"10.0.0.0/8 mid", "10.50.100.200", true},
		{"10.0.0.0/8 end", "10.255.255.255", true},
		{"172.16.0.0/12 start", "172.16.0.0", true},
		{"172.16.0.0/12 mid", "172.20.50.100", true},
		{"172.16.0.0/12 end", "172.31.255.255", true},
		{"192.168.0.0/16 start", "192.168.0.0", true},
		{"192.168.0.0/16 mid", "192.168.1.100", true},
		{"192.168.0.0/16 end", "192.168.255.255", true},

		// AWS metadata endpoint range (SSRF target)
		{"AWS metadata 169.254.169.254", "169.254.169.254", true},
		{"Link-local start", "169.254.0.1", true},
		{"Link-local end", "169.254.255.255", true},

		// IPv6 private ranges
		{"IPv6 unique local fc00::", "fc00::1", true},
		{"IPv6 unique local fd00::", "fd00::1", true},
		{"IPv6 link-local", "fe80::1", true},

		// Public IPs - should NOT be private
		{"Public IP 8.8.8.8", "8.8.8.8", false},
		{"Public IP 1.1.1.1", "1.1.1.1", false},
		{"Public IP 93.184.216.34", "93.184.216.34", false},
		{"Public IP 172.32.0.1 (outside 172.16/12)", "172.32.0.1", false},
		{"Public IP 192.167.1.1 (not 192.168)", "192.167.1.1", false},
		{"Public IPv6", "2001:4860:4860::8888", false},

		// Edge cases
		{"Zero IP", "0.0.0.0", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ip := net.ParseIP(tc.ip)
			require.NotNil(t, ip, "Failed to parse IP: %s", tc.ip)
			result := isPrivateIP(ip)
			assert.Equal(t, tc.isPrivate, result, "IP %s: expected isPrivate=%v, got %v", tc.ip, tc.isPrivate, result)
		})
	}

	t.Run("nil IP returns false", func(t *testing.T) {
		result := isPrivateIP(nil)
		assert.False(t, result)
	})
}

func TestValidateWebhookHeaders(t *testing.T) {
	t.Run("valid headers pass validation", func(t *testing.T) {
		headers := map[string]string{
			"Authorization": "Bearer token123",
			"X-Custom-Header": "custom-value",
			"X-API-Key": "api-key-value",
		}
		err := validateWebhookHeaders(headers)
		assert.NoError(t, err)
	})

	t.Run("empty headers pass validation", func(t *testing.T) {
		err := validateWebhookHeaders(map[string]string{})
		assert.NoError(t, err)
	})

	t.Run("nil headers pass validation", func(t *testing.T) {
		err := validateWebhookHeaders(nil)
		assert.NoError(t, err)
	})

	t.Run("blocked headers are rejected", func(t *testing.T) {
		blockedHeaders := []string{
			"Content-Length",
			"Host",
			"Transfer-Encoding",
			"Connection",
			"Keep-Alive",
			"Proxy-Authenticate",
			"Proxy-Authorization",
			"TE",
			"Trailers",
			"Upgrade",
		}

		for _, header := range blockedHeaders {
			t.Run(header, func(t *testing.T) {
				headers := map[string]string{header: "some-value"}
				err := validateWebhookHeaders(headers)
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "not allowed to be overridden")
			})
		}
	})

	t.Run("blocked headers case insensitive", func(t *testing.T) {
		testCases := []string{
			"content-length",
			"CONTENT-LENGTH",
			"Content-Length",
			"CoNtEnT-lEnGtH",
		}

		for _, header := range testCases {
			t.Run(header, func(t *testing.T) {
				headers := map[string]string{header: "123"}
				err := validateWebhookHeaders(headers)
				assert.Error(t, err)
			})
		}
	})

	t.Run("CRLF injection in header name rejected", func(t *testing.T) {
		testCases := []struct {
			name   string
			header string
		}{
			{"carriage return", "X-Evil\rHeader"},
			{"newline", "X-Evil\nHeader"},
			{"CRLF", "X-Evil\r\nHeader"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				headers := map[string]string{tc.header: "value"}
				err := validateWebhookHeaders(headers)
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "contains invalid characters")
			})
		}
	})

	t.Run("CRLF injection in header value rejected", func(t *testing.T) {
		testCases := []struct {
			name  string
			value string
		}{
			{"carriage return", "evil\rvalue"},
			{"newline", "evil\nvalue"},
			{"CRLF injection attempt", "value\r\nX-Injected: malicious"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				headers := map[string]string{"X-Custom": tc.value}
				err := validateWebhookHeaders(headers)
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "contains invalid characters")
			})
		}
	})

	t.Run("header value length limit enforced", func(t *testing.T) {
		// Create a string longer than 8192 bytes
		longValue := make([]byte, 8193)
		for i := range longValue {
			longValue[i] = 'a'
		}

		headers := map[string]string{"X-Custom": string(longValue)}
		err := validateWebhookHeaders(headers)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "exceeds maximum length")
	})

	t.Run("header value at limit passes", func(t *testing.T) {
		// Create a string exactly at 8192 bytes
		value := make([]byte, 8192)
		for i := range value {
			value[i] = 'a'
		}

		headers := map[string]string{"X-Custom": string(value)}
		err := validateWebhookHeaders(headers)
		assert.NoError(t, err)
	})
}

func TestValidateWebhookURL(t *testing.T) {
	t.Run("valid HTTPS URLs pass", func(t *testing.T) {
		// Note: These tests will fail if DNS resolution fails for the domain
		// In CI, you might want to skip these or use mock DNS
		validURLs := []string{
			"https://example.com/webhook",
			"https://api.example.com/v1/webhooks",
			"https://webhook.site/test",
		}

		for _, url := range validURLs {
			t.Run(url, func(t *testing.T) {
				err := validateWebhookURL(url)
				// The URL validation might fail due to DNS resolution in test environment
				// but the scheme and hostname validation should pass
				if err != nil {
					assert.Contains(t, err.Error(), "resolve")
				}
			})
		}
	})

	t.Run("HTTP URLs are allowed", func(t *testing.T) {
		err := validateWebhookURL("http://example.com/webhook")
		// May fail DNS in test, but scheme should pass
		if err != nil {
			assert.NotContains(t, err.Error(), "scheme")
		}
	})

	t.Run("non-HTTP schemes rejected", func(t *testing.T) {
		invalidSchemes := []string{
			"ftp://ftp.example.com/file",
			"file:///etc/passwd",
			"javascript:alert(1)",
			"data:text/html,<script>alert(1)</script>",
			"gopher://gopher.example.com",
		}

		for _, url := range invalidSchemes {
			t.Run(url, func(t *testing.T) {
				err := validateWebhookURL(url)
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "scheme")
			})
		}
	})

	t.Run("localhost URLs rejected", func(t *testing.T) {
		localhostURLs := []string{
			"http://localhost/webhook",
			"http://localhost:8080/webhook",
			"https://localhost/webhook",
			"http://LOCALHOST/webhook",
			"http://ip6-localhost/webhook",
		}

		for _, url := range localhostURLs {
			t.Run(url, func(t *testing.T) {
				err := validateWebhookURL(url)
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "localhost")
			})
		}
	})

	t.Run("cloud metadata hostnames rejected", func(t *testing.T) {
		metadataURLs := []string{
			"http://metadata.google.internal/computeMetadata/v1/",
			"http://metadata/latest/meta-data/",
			"http://instance-data/latest/meta-data/",
			"http://kubernetes.default/api",
			"http://kubernetes.default.svc/api",
		}

		for _, url := range metadataURLs {
			t.Run(url, func(t *testing.T) {
				err := validateWebhookURL(url)
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "internal hostname")
			})
		}
	})

	t.Run("invalid URLs rejected", func(t *testing.T) {
		invalidURLs := []string{
			"not-a-url",
			"://missing-scheme.com",
			"",
		}

		for _, url := range invalidURLs {
			t.Run(url, func(t *testing.T) {
				err := validateWebhookURL(url)
				assert.Error(t, err)
			})
		}
	})

	t.Run("URL without hostname rejected", func(t *testing.T) {
		err := validateWebhookURL("http:///path/only")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "hostname")
	})

	// These tests verify IP-based URLs are validated
	t.Run("private IP in URL rejected", func(t *testing.T) {
		privateIPURLs := []string{
			"http://127.0.0.1/webhook",
			"http://10.0.0.1/webhook",
			"http://172.16.0.1/webhook",
			"http://192.168.1.1/webhook",
			"http://169.254.169.254/latest/meta-data/", // AWS metadata
		}

		for _, url := range privateIPURLs {
			t.Run(url, func(t *testing.T) {
				err := validateWebhookURL(url)
				assert.Error(t, err)
				// Either rejected as private IP or localhost
				assert.True(t,
					contains(err.Error(), "private IP") ||
					contains(err.Error(), "localhost"),
					"Expected private IP or localhost error, got: %v", err)
			})
		}
	})
}

// Note: TestGenerateSignature is defined in webhook_test.go

func TestParseTableReference(t *testing.T) {
	testCases := []struct {
		input          string
		expectedSchema string
		expectedTable  string
	}{
		{"users", "auth", "users"},
		{"auth.users", "auth", "users"},
		{"public.products", "public", "products"},
		{"my_schema.my_table", "my_schema", "my_table"},
		{"schema.table.extra", "schema", "table.extra"}, // Only splits on first dot
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			schema, table := parseTableReference(tc.input)
			assert.Equal(t, tc.expectedSchema, schema)
			assert.Equal(t, tc.expectedTable, table)
		})
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
