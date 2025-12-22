package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSecurityEventTypes(t *testing.T) {
	// Test that all security event types are defined
	eventTypes := []SecurityEventType{
		// Login events
		SecurityEventLoginSuccess,
		SecurityEventLoginFailed,
		SecurityEventAccountLocked,
		SecurityEventAccountUnlocked,
		SecurityEventLogout,

		// Token events
		SecurityEventTokenRefresh,
		SecurityEventTokenRevoked,
		SecurityEventInvalidToken,

		// Password events
		SecurityEventPasswordReset,
		SecurityEventPasswordChanged,

		// 2FA events
		SecurityEvent2FAEnabled,
		SecurityEvent2FADisabled,
		SecurityEvent2FAVerified,
		SecurityEvent2FAFailed,

		// Impersonation events
		SecurityEventImpersonationStart,
		SecurityEventImpersonationEnd,

		// Suspicious activity
		SecurityEventSuspiciousActivity,
		SecurityEventRateLimitExceeded,
	}

	for _, eventType := range eventTypes {
		assert.NotEmpty(t, string(eventType))
	}
}

func TestNewSecurityLogger(t *testing.T) {
	logger := NewSecurityLogger()
	require.NotNil(t, logger)
}

func TestSecurityLogger_Log(t *testing.T) {
	// Capture log output
	var buf bytes.Buffer
	oldLogger := log.Logger
	log.Logger = zerolog.New(&buf)
	defer func() { log.Logger = oldLogger }()

	logger := NewSecurityLogger()
	ctx := context.Background()

	t.Run("logs basic security event", func(t *testing.T) {
		buf.Reset()

		event := SecurityEvent{
			Type: SecurityEventLoginSuccess,
		}
		logger.Log(ctx, event)

		var logEntry map[string]interface{}
		err := json.Unmarshal(buf.Bytes(), &logEntry)
		require.NoError(t, err)

		assert.Equal(t, "security", logEntry["component"])
		assert.Equal(t, "login_success", logEntry["security_event"])
		assert.Equal(t, "login_success", logEntry["event_type"])
		assert.Equal(t, "Security event", logEntry["message"])
	})

	t.Run("logs event with user ID", func(t *testing.T) {
		buf.Reset()

		event := SecurityEvent{
			Type:   SecurityEventLoginSuccess,
			UserID: "user-123",
		}
		logger.Log(ctx, event)

		var logEntry map[string]interface{}
		err := json.Unmarshal(buf.Bytes(), &logEntry)
		require.NoError(t, err)

		assert.Equal(t, "user-123", logEntry["user_id"])
	})

	t.Run("logs event with email", func(t *testing.T) {
		buf.Reset()

		event := SecurityEvent{
			Type:  SecurityEventLoginSuccess,
			Email: "user@example.com",
		}
		logger.Log(ctx, event)

		var logEntry map[string]interface{}
		err := json.Unmarshal(buf.Bytes(), &logEntry)
		require.NoError(t, err)

		assert.Equal(t, "user@example.com", logEntry["email"])
	})

	t.Run("logs event with IP address", func(t *testing.T) {
		buf.Reset()

		event := SecurityEvent{
			Type:      SecurityEventLoginFailed,
			IPAddress: "192.168.1.1",
		}
		logger.Log(ctx, event)

		var logEntry map[string]interface{}
		err := json.Unmarshal(buf.Bytes(), &logEntry)
		require.NoError(t, err)

		assert.Equal(t, "192.168.1.1", logEntry["ip_address"])
	})

	t.Run("logs event with user agent", func(t *testing.T) {
		buf.Reset()

		event := SecurityEvent{
			Type:      SecurityEventLoginSuccess,
			UserAgent: "Mozilla/5.0 Test Browser",
		}
		logger.Log(ctx, event)

		var logEntry map[string]interface{}
		err := json.Unmarshal(buf.Bytes(), &logEntry)
		require.NoError(t, err)

		assert.Equal(t, "Mozilla/5.0 Test Browser", logEntry["user_agent"])
	})

	t.Run("logs event with details", func(t *testing.T) {
		buf.Reset()

		event := SecurityEvent{
			Type: SecurityEventAccountLocked,
			Details: map[string]interface{}{
				"reason":         "too_many_failed_attempts",
				"failed_attempts": 5,
			},
		}
		logger.Log(ctx, event)

		var logEntry map[string]interface{}
		err := json.Unmarshal(buf.Bytes(), &logEntry)
		require.NoError(t, err)

		details := logEntry["details"].(map[string]interface{})
		assert.Equal(t, "too_many_failed_attempts", details["reason"])
		assert.Equal(t, float64(5), details["failed_attempts"])
	})

	t.Run("logs full security event", func(t *testing.T) {
		buf.Reset()

		event := SecurityEvent{
			Type:      SecurityEventLoginSuccess,
			UserID:    "user-456",
			Email:     "admin@example.com",
			IPAddress: "10.0.0.1",
			UserAgent: "Custom/1.0",
			Details: map[string]interface{}{
				"auth_method": "password",
			},
		}
		logger.Log(ctx, event)

		var logEntry map[string]interface{}
		err := json.Unmarshal(buf.Bytes(), &logEntry)
		require.NoError(t, err)

		assert.Equal(t, "security", logEntry["component"])
		assert.Equal(t, "login_success", logEntry["security_event"])
		assert.Equal(t, "user-456", logEntry["user_id"])
		assert.Equal(t, "admin@example.com", logEntry["email"])
		assert.Equal(t, "10.0.0.1", logEntry["ip_address"])
		assert.Equal(t, "Custom/1.0", logEntry["user_agent"])
	})
}

func TestSecurityLogger_LogWarning(t *testing.T) {
	// Capture log output
	var buf bytes.Buffer
	oldLogger := log.Logger
	log.Logger = zerolog.New(&buf)
	defer func() { log.Logger = oldLogger }()

	logger := NewSecurityLogger()
	ctx := context.Background()

	t.Run("logs warning level security event", func(t *testing.T) {
		buf.Reset()

		event := SecurityEvent{
			Type:      SecurityEventSuspiciousActivity,
			UserID:    "user-789",
			IPAddress: "1.2.3.4",
			Details: map[string]interface{}{
				"reason": "multiple_failed_logins_from_different_ips",
			},
		}
		logger.LogWarning(ctx, event)

		var logEntry map[string]interface{}
		err := json.Unmarshal(buf.Bytes(), &logEntry)
		require.NoError(t, err)

		assert.Equal(t, "security", logEntry["component"])
		assert.Equal(t, "suspicious_activity", logEntry["security_event"])
		assert.Equal(t, "user-789", logEntry["user_id"])
		assert.Equal(t, "1.2.3.4", logEntry["ip_address"])
		assert.Equal(t, "Security warning", logEntry["message"])
		assert.Equal(t, "warn", logEntry["level"])
	})

	t.Run("logs rate limit exceeded warning", func(t *testing.T) {
		buf.Reset()

		event := SecurityEvent{
			Type:      SecurityEventRateLimitExceeded,
			IPAddress: "5.6.7.8",
			Details: map[string]interface{}{
				"endpoint": "/api/login",
				"limit":    100,
			},
		}
		logger.LogWarning(ctx, event)

		var logEntry map[string]interface{}
		err := json.Unmarshal(buf.Bytes(), &logEntry)
		require.NoError(t, err)

		assert.Equal(t, "rate_limit_exceeded", logEntry["security_event"])
		assert.Equal(t, "warn", logEntry["level"])
	})
}

func TestLogSecurityEvent(t *testing.T) {
	// Capture log output
	var buf bytes.Buffer
	oldLogger := log.Logger
	log.Logger = zerolog.New(&buf)
	defer func() { log.Logger = oldLogger }()

	ctx := context.Background()

	event := SecurityEvent{
		Type:   SecurityEventLoginSuccess,
		UserID: "user-global",
	}
	LogSecurityEvent(ctx, event)

	var logEntry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)

	assert.Equal(t, "login_success", logEntry["security_event"])
	assert.Equal(t, "user-global", logEntry["user_id"])
}

func TestLogSecurityWarning(t *testing.T) {
	// Capture log output
	var buf bytes.Buffer
	oldLogger := log.Logger
	log.Logger = zerolog.New(&buf)
	defer func() { log.Logger = oldLogger }()

	ctx := context.Background()

	event := SecurityEvent{
		Type:   SecurityEventSuspiciousActivity,
		UserID: "user-warning",
	}
	LogSecurityWarning(ctx, event)

	var logEntry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)

	assert.Equal(t, "suspicious_activity", logEntry["security_event"])
	assert.Equal(t, "user-warning", logEntry["user_id"])
	assert.Equal(t, "warn", logEntry["level"])
}

func TestSecurityEventStruct(t *testing.T) {
	event := SecurityEvent{
		Type:      SecurityEventLoginSuccess,
		UserID:    "user-123",
		Email:     "test@example.com",
		IPAddress: "192.168.1.1",
		UserAgent: "TestAgent/1.0",
		Details: map[string]interface{}{
			"key": "value",
		},
	}

	assert.Equal(t, SecurityEventLoginSuccess, event.Type)
	assert.Equal(t, "user-123", event.UserID)
	assert.Equal(t, "test@example.com", event.Email)
	assert.Equal(t, "192.168.1.1", event.IPAddress)
	assert.Equal(t, "TestAgent/1.0", event.UserAgent)
	assert.Equal(t, "value", event.Details["key"])
}
