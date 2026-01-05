package auth

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewTurnstileProvider(t *testing.T) {
	secretKey := "test-secret"
	client := &http.Client{}

	provider := NewTurnstileProvider(secretKey, client)

	assert.NotNil(t, provider)
	assert.Equal(t, secretKey, provider.secretKey)
	assert.Equal(t, client, provider.httpClient)
}

func TestTurnstileProvider_Name(t *testing.T) {
	provider := NewTurnstileProvider("test-secret", &http.Client{})
	assert.Equal(t, "turnstile", provider.Name())
}

func TestTurnstileProvider_TranslateErrorCode(t *testing.T) {
	provider := NewTurnstileProvider("test-secret", &http.Client{})

	tests := []struct {
		code     string
		expected string
	}{
		{"missing-input-secret", "missing secret key"},
		{"invalid-input-secret", "invalid secret key"},
		{"missing-input-response", "missing captcha response"},
		{"invalid-input-response", "invalid captcha response"},
		{"bad-request", "bad request"},
		{"timeout-or-duplicate", "captcha token expired or already used"},
		{"internal-error", "internal server error"},
		{"custom-error", "verification failed: custom-error"},
		{"", "verification failed: "},
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			result := provider.translateErrorCode(tt.code)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTurnstileProvider_AllErrorCodes(t *testing.T) {
	provider := NewTurnstileProvider("test-secret", &http.Client{})

	// Test all known Turnstile error codes
	errorCodes := []string{
		"missing-input-secret",
		"invalid-input-secret",
		"missing-input-response",
		"invalid-input-response",
		"bad-request",
		"timeout-or-duplicate",
		"internal-error",
	}

	for _, code := range errorCodes {
		result := provider.translateErrorCode(code)
		assert.NotEmpty(t, result, "error code %s should have translation", code)
		assert.NotContains(t, result, code, "translation should not contain raw error code with hyphens")
	}
}
