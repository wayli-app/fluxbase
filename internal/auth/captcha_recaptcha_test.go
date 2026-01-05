package auth

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewReCaptchaProvider(t *testing.T) {
	tests := []struct {
		name               string
		secretKey          string
		scoreThreshold     float64
		expectedThreshold  float64
	}{
		{
			name:              "valid threshold",
			secretKey:         "test-secret",
			scoreThreshold:    0.7,
			expectedThreshold: 0.7,
		},
		{
			name:              "zero threshold uses default",
			secretKey:         "test-secret",
			scoreThreshold:    0.0,
			expectedThreshold: 0.5,
		},
		{
			name:              "negative threshold uses default",
			secretKey:         "test-secret",
			scoreThreshold:    -0.1,
			expectedThreshold: 0.5,
		},
		{
			name:              "threshold above 1.0 uses default",
			secretKey:         "test-secret",
			scoreThreshold:    1.5,
			expectedThreshold: 0.5,
		},
		{
			name:              "minimum valid threshold",
			secretKey:         "test-secret",
			scoreThreshold:    0.1,
			expectedThreshold: 0.1,
		},
		{
			name:              "maximum valid threshold",
			secretKey:         "test-secret",
			scoreThreshold:    1.0,
			expectedThreshold: 1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &http.Client{}
			provider := NewReCaptchaProvider(tt.secretKey, tt.scoreThreshold, client)

			assert.NotNil(t, provider)
			assert.Equal(t, tt.secretKey, provider.secretKey)
			assert.Equal(t, tt.expectedThreshold, provider.scoreThreshold)
			assert.Equal(t, client, provider.httpClient)
		})
	}
}

func TestReCaptchaProvider_Name(t *testing.T) {
	provider := NewReCaptchaProvider("test-secret", 0.5, &http.Client{})
	assert.Equal(t, "recaptcha_v3", provider.Name())
}

func TestReCaptchaProvider_TranslateErrorCode(t *testing.T) {
	provider := NewReCaptchaProvider("test-secret", 0.5, &http.Client{})

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
		{"custom-error", "verification failed: custom-error"},
		{"", "verification failed: "},
		{"UNKNOWN_ERROR", "verification failed: UNKNOWN_ERROR"},
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			result := provider.translateErrorCode(tt.code)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestReCaptchaProvider_ThresholdBoundaries(t *testing.T) {
	tests := []struct {
		name      string
		threshold float64
		expected  float64
	}{
		{"exactly zero", 0.0, 0.5},
		{"just below zero", -0.000001, 0.5},
		{"exactly one", 1.0, 1.0},
		{"just above one", 1.000001, 0.5},
		{"very small valid", 0.000001, 0.000001},
		{"very close to one", 0.999999, 0.999999},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := NewReCaptchaProvider("secret", tt.threshold, &http.Client{})
			assert.Equal(t, tt.expected, provider.scoreThreshold)
		})
	}
}
