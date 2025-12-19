package auth

import (
	"context"

	"github.com/rs/zerolog/log"
)

// SecurityEventType represents the type of security event
type SecurityEventType string

const (
	// Login events
	SecurityEventLoginSuccess    SecurityEventType = "login_success"
	SecurityEventLoginFailed     SecurityEventType = "login_failed"
	SecurityEventAccountLocked   SecurityEventType = "account_locked"
	SecurityEventAccountUnlocked SecurityEventType = "account_unlocked"
	SecurityEventLogout          SecurityEventType = "logout"

	// Token events
	SecurityEventTokenRefresh SecurityEventType = "token_refresh"
	SecurityEventTokenRevoked SecurityEventType = "token_revoked"
	SecurityEventInvalidToken SecurityEventType = "invalid_token"

	// Password events
	SecurityEventPasswordReset   SecurityEventType = "password_reset_requested"
	SecurityEventPasswordChanged SecurityEventType = "password_changed"

	// 2FA events
	SecurityEvent2FAEnabled  SecurityEventType = "2fa_enabled"
	SecurityEvent2FADisabled SecurityEventType = "2fa_disabled"
	SecurityEvent2FAVerified SecurityEventType = "2fa_verified"
	SecurityEvent2FAFailed   SecurityEventType = "2fa_failed"

	// Impersonation events
	SecurityEventImpersonationStart SecurityEventType = "impersonation_start"
	SecurityEventImpersonationEnd   SecurityEventType = "impersonation_end"

	// Suspicious activity
	SecurityEventSuspiciousActivity SecurityEventType = "suspicious_activity"
	SecurityEventRateLimitExceeded  SecurityEventType = "rate_limit_exceeded"
)

// SecurityEvent represents a security event to be logged
type SecurityEvent struct {
	Type      SecurityEventType
	UserID    string
	Email     string
	IPAddress string
	UserAgent string
	Details   map[string]interface{}
}

// SecurityLogger handles logging of security events
type SecurityLogger struct{}

// NewSecurityLogger creates a new security logger
func NewSecurityLogger() *SecurityLogger {
	return &SecurityLogger{}
}

// Log logs a security event
// All security events are logged at INFO level with a "security_event" marker
// so they can be easily filtered and monitored
func (s *SecurityLogger) Log(ctx context.Context, event SecurityEvent) {
	// Use log.Info() directly to ensure we use the current global logger writer
	// (which may have been replaced by the logging service after package init)
	logEvent := log.Info().
		Str("component", "security").
		Str("security_event", string(event.Type)).
		Str("event_type", string(event.Type))

	if event.UserID != "" {
		logEvent = logEvent.Str("user_id", event.UserID)
	}
	if event.Email != "" {
		logEvent = logEvent.Str("email", event.Email)
	}
	if event.IPAddress != "" {
		logEvent = logEvent.Str("ip_address", event.IPAddress)
	}
	if event.UserAgent != "" {
		logEvent = logEvent.Str("user_agent", event.UserAgent)
	}
	if event.Details != nil {
		logEvent = logEvent.Interface("details", event.Details)
	}

	logEvent.Msg("Security event")
}

// LogWarning logs a warning-level security event (suspicious activity)
func (s *SecurityLogger) LogWarning(ctx context.Context, event SecurityEvent) {
	// Use log.Warn() directly to ensure we use the current global logger writer
	logEvent := log.Warn().
		Str("component", "security").
		Str("security_event", string(event.Type)).
		Str("event_type", string(event.Type))

	if event.UserID != "" {
		logEvent = logEvent.Str("user_id", event.UserID)
	}
	if event.Email != "" {
		logEvent = logEvent.Str("email", event.Email)
	}
	if event.IPAddress != "" {
		logEvent = logEvent.Str("ip_address", event.IPAddress)
	}
	if event.UserAgent != "" {
		logEvent = logEvent.Str("user_agent", event.UserAgent)
	}
	if event.Details != nil {
		logEvent = logEvent.Interface("details", event.Details)
	}

	logEvent.Msg("Security warning")
}

// Global security logger instance
var securityLogger = NewSecurityLogger()

// LogSecurityEvent logs a security event using the global logger
func LogSecurityEvent(ctx context.Context, event SecurityEvent) {
	securityLogger.Log(ctx, event)
}

// LogSecurityWarning logs a warning-level security event using the global logger
func LogSecurityWarning(ctx context.Context, event SecurityEvent) {
	securityLogger.LogWarning(ctx, event)
}
