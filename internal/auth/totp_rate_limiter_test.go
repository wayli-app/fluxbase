package auth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDefaultTOTPRateLimiterConfig(t *testing.T) {
	config := DefaultTOTPRateLimiterConfig()

	assert.Equal(t, 5, config.MaxAttempts, "Default max attempts should be 5")
	assert.Equal(t, 5*time.Minute, config.WindowDuration, "Default window should be 5 minutes")
	assert.Equal(t, 15*time.Minute, config.LockoutDuration, "Default lockout should be 15 minutes")
}

func TestNewTOTPRateLimiter_DefaultValues(t *testing.T) {
	// Test with zero values - should use defaults
	limiter := NewTOTPRateLimiter(nil, TOTPRateLimiterConfig{})

	assert.Equal(t, 5, limiter.maxAttempts)
	assert.Equal(t, 5*time.Minute, limiter.windowDuration)
	assert.Equal(t, 15*time.Minute, limiter.lockoutDuration)
}

func TestNewTOTPRateLimiter_CustomValues(t *testing.T) {
	config := TOTPRateLimiterConfig{
		MaxAttempts:     10,
		WindowDuration:  10 * time.Minute,
		LockoutDuration: 30 * time.Minute,
	}

	limiter := NewTOTPRateLimiter(nil, config)

	assert.Equal(t, 10, limiter.maxAttempts)
	assert.Equal(t, 10*time.Minute, limiter.windowDuration)
	assert.Equal(t, 30*time.Minute, limiter.lockoutDuration)
}

func TestNewTOTPRateLimiter_NegativeValues(t *testing.T) {
	// Negative values should be replaced with defaults
	config := TOTPRateLimiterConfig{
		MaxAttempts:     -1,
		WindowDuration:  -1,
		LockoutDuration: -1,
	}

	limiter := NewTOTPRateLimiter(nil, config)

	assert.Equal(t, 5, limiter.maxAttempts)
	assert.Equal(t, 5*time.Minute, limiter.windowDuration)
	assert.Equal(t, 15*time.Minute, limiter.lockoutDuration)
}

func TestNilIfEmpty(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *string
	}{
		{
			name:     "empty string returns nil",
			input:    "",
			expected: nil,
		},
		{
			name:     "non-empty string returns pointer",
			input:    "192.168.1.1",
			expected: strPtr("192.168.1.1"),
		},
		{
			name:     "whitespace string returns pointer",
			input:    "  ",
			expected: strPtr("  "),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := nilIfEmpty(tt.input)
			if tt.expected == nil {
				assert.Nil(t, result)
			} else {
				assert.NotNil(t, result)
				assert.Equal(t, *tt.expected, *result)
			}
		})
	}
}

func TestErrTOTPRateLimitExceeded(t *testing.T) {
	// Ensure the error message is user-friendly
	assert.Contains(t, ErrTOTPRateLimitExceeded.Error(), "too many")
	assert.Contains(t, ErrTOTPRateLimitExceeded.Error(), "try again")
}

// Helper function
func strPtr(s string) *string {
	return &s
}

// Note: Integration tests for CheckRateLimit, RecordAttempt, ClearFailedAttempts,
// and GetFailedAttemptCount require a database connection and are in the E2E test suite.
// The unit tests above verify the configuration and helper logic.
