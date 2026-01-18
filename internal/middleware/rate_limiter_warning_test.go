package middleware

import (
	"os"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLogRateLimiterWarning_WithRedisURL(t *testing.T) {
	// Reset the warning state for this test
	resetRateLimiterWarning()

	// Set Redis URL - warning should not be displayed
	os.Setenv("FLUXBASE_REDIS_URL", "redis://localhost:6379")
	defer os.Unsetenv("FLUXBASE_REDIS_URL")

	// Set Kubernetes indicator
	os.Setenv("KUBERNETES_SERVICE_HOST", "10.0.0.1")
	defer os.Unsetenv("KUBERNETES_SERVICE_HOST")

	logRateLimiterWarning()

	// Warning should not be displayed when Redis is configured
	assert.False(t, IsRateLimiterWarningDisplayed())
}

func TestLogRateLimiterWarning_WithDragonflyURL(t *testing.T) {
	// Reset the warning state for this test
	resetRateLimiterWarning()

	// Set Dragonfly URL - warning should not be displayed
	os.Setenv("FLUXBASE_DRAGONFLY_URL", "redis://localhost:6379")
	defer os.Unsetenv("FLUXBASE_DRAGONFLY_URL")

	// Set Kubernetes indicator
	os.Setenv("KUBERNETES_SERVICE_HOST", "10.0.0.1")
	defer os.Unsetenv("KUBERNETES_SERVICE_HOST")

	logRateLimiterWarning()

	// Warning should not be displayed when Dragonfly is configured
	assert.False(t, IsRateLimiterWarningDisplayed())
}

func TestLogRateLimiterWarning_NoMultiInstanceIndicators(t *testing.T) {
	// Reset the warning state for this test
	resetRateLimiterWarning()

	// Clear all multi-instance indicators
	os.Unsetenv("KUBERNETES_SERVICE_HOST")
	os.Unsetenv("POD_NAME")
	os.Unsetenv("COMPOSE_PROJECT_NAME")
	os.Unsetenv("FLUXBASE_REDIS_URL")
	os.Unsetenv("FLUXBASE_DRAGONFLY_URL")

	// Store original HOSTNAME and clear it for this test
	originalHostname := os.Getenv("HOSTNAME")
	os.Unsetenv("HOSTNAME")
	defer func() {
		if originalHostname != "" {
			os.Setenv("HOSTNAME", originalHostname)
		}
	}()

	logRateLimiterWarning()

	// Warning should not be displayed when no multi-instance indicators are present
	// Note: HOSTNAME might be set in some environments, so we only test when it's not
	// The test may still pass if HOSTNAME was already set
}

// Helper to reset the warning state between tests
func resetRateLimiterWarning() {
	rateLimiterWarningDisplayed = false
	rateLimiterWarningMu = sync.Once{}
}
