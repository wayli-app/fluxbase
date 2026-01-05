package auth

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewHCaptchaProvider(t *testing.T) {
	secretKey := "test-secret"
	client := &http.Client{}

	provider := NewHCaptchaProvider(secretKey, client)

	assert.NotNil(t, provider)
	assert.Equal(t, secretKey, provider.secretKey)
	assert.Equal(t, client, provider.httpClient)
}

func TestHCaptchaProvider_Name(t *testing.T) {
	provider := NewHCaptchaProvider("test-secret", &http.Client{})
	assert.Equal(t, "hcaptcha", provider.Name())
}

func TestHCaptchaProvider_TranslateErrorCode(t *testing.T) {
	provider := NewHCaptchaProvider("test-secret", &http.Client{})

	tests := []struct {
		code     string
		expected string
	}{
		{"missing-input-secret", "missing secret key"},
		{"invalid-input-secret", "invalid secret key"},
		{"missing-input-response", "missing captcha response"},
		{"invalid-input-response", "invalid captcha response"},
		{"bad-request", "bad request"},
		{"invalid-or-already-seen-response", "captcha token already used"},
		{"not-using-dummy-passcode", "test mode misconfigured"},
		{"sitekey-secret-mismatch", "site key and secret key mismatch"},
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

func TestHCaptchaProvider_AllErrorCodes(t *testing.T) {
	provider := NewHCaptchaProvider("test-secret", &http.Client{})

	// Test all known hCaptcha error codes
	errorCodes := []string{
		"missing-input-secret",
		"invalid-input-secret",
		"missing-input-response",
		"invalid-input-response",
		"bad-request",
		"invalid-or-already-seen-response",
		"not-using-dummy-passcode",
		"sitekey-secret-mismatch",
	}

	for _, code := range errorCodes {
		result := provider.translateErrorCode(code)
		assert.NotEmpty(t, result, "error code %s should have translation", code)
		assert.NotContains(t, result, code, "translation should not contain raw error code")
	}
}
