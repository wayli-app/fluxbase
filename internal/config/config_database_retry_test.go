package config

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Verifies that the connection string always carries a connect_timeout so a hung
// TCP connect fails fast instead of waiting on the OS default.
func TestDatabaseConfig_ConnectionStringIncludesConnectTimeout(t *testing.T) {
	dc := DatabaseConfig{
		Host:     "localhost",
		Port:     5432,
		User:     "postgres",
		Password: "p@ss w0rd!",
		Database: "fluxbase",
		SSLMode:  "disable",
	}

	got := dc.RuntimeConnectionString()
	assert.Contains(t, got, "connect_timeout=10", "default connect_timeout=10 should be present")

	// Custom timeout.
	dc.ConnectTimeout = 7 * time.Second
	got = dc.RuntimeConnectionString()
	assert.Contains(t, got, "connect_timeout=7", "custom connect_timeout should be honored")

	// The password contains spaces/special chars that must be percent-encoded,
	// proving the builder uses url.UserPassword rather than naive concatenation.
	assert.True(t, strings.HasPrefix(got, "postgres://"))
	assert.NotContains(t, got, "p@ss w0rd!", "raw password must not appear; it should be percent-encoded")
	assert.Contains(t, got, "p%40ss", "special char '@' should be percent-encoded")
}

func TestDatabaseConfig_RetryDefaultsValidate(t *testing.T) {
	// Zero/negative retry values are allowed (treated as defaults); validation
	// only rejects explicit negatives.
	dc := DatabaseConfig{
		Host:            "localhost",
		Port:            5432,
		User:            "postgres",
		Database:        "fluxbase",
		SSLMode:         "disable",
		MaxConnections:  10,
		MaxConnLifetime: time.Hour,
		MaxConnIdleTime: 30 * time.Minute,
		HealthCheck:     time.Minute,
	}
	require.NoError(t, dc.Validate())

	dc.RetryAttempts = -1
	assert.Error(t, dc.Validate(), "negative retry_attempts must be rejected")
}
