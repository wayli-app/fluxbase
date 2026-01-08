package middleware

import (
	"net/url"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// sensitiveQueryParams are query parameters that should be redacted from logs
var sensitiveQueryParams = []string{"token", "access_token", "refresh_token", "api_key", "apikey", "key", "secret", "password"}

// StructuredLoggerConfig holds configuration for structured logging
type StructuredLoggerConfig struct {
	// SkipPaths are paths that should not be logged (e.g., health checks)
	SkipPaths []string
	// SkipSuccessfulRequests skips logging successful requests (2xx status codes)
	SkipSuccessfulRequests bool
	// Logger is the zerolog logger to use (defaults to global log)
	Logger *zerolog.Logger
	// LogRequestBody logs the request body (be careful with sensitive data)
	LogRequestBody bool
	// LogResponseBody logs the response body (be careful with sensitive data)
	LogResponseBody bool
	// SlowRequestThreshold logs slow requests with WARN level (0 = disabled)
	SlowRequestThreshold time.Duration
}

// DefaultStructuredLoggerConfig returns default configuration
func DefaultStructuredLoggerConfig() StructuredLoggerConfig {
	return StructuredLoggerConfig{
		SkipPaths: []string{
			"/health",
			"/ready",
			"/metrics",
		},
		SkipSuccessfulRequests: false,
		Logger:                 nil, // Use global log
		LogRequestBody:         false,
		LogResponseBody:        false,
		SlowRequestThreshold:   1 * time.Second, // Warn on requests > 1s
	}
}

// redactQueryString redacts sensitive query parameters from a query string
func redactQueryString(queryString string) string {
	if queryString == "" {
		return ""
	}

	values, err := url.ParseQuery(queryString)
	if err != nil {
		// If we can't parse it, redact the whole thing to be safe
		return "[redacted]"
	}

	for _, param := range sensitiveQueryParams {
		if values.Has(param) {
			values.Set(param, "[redacted]")
		}
		// Also check case-insensitive
		for key := range values {
			if strings.EqualFold(key, param) && key != param {
				values.Set(key, "[redacted]")
			}
		}
	}

	return values.Encode()
}

// StructuredLogger returns a middleware that logs requests with structured logging
func StructuredLogger(config ...StructuredLoggerConfig) fiber.Handler {
	// Use default config if none provided
	cfg := DefaultStructuredLoggerConfig()
	if len(config) > 0 {
		cfg = config[0]
	}

	// Use provided logger or default to global
	logger := log.Logger
	if cfg.Logger != nil {
		logger = *cfg.Logger
	}

	return func(c *fiber.Ctx) error {
		// Check if path should be skipped
		path := c.Path()
		for _, skipPath := range cfg.SkipPaths {
			if path == skipPath {
				return c.Next()
			}
		}

		// Start timer
		start := time.Now()

		// Get request ID (should be set by requestid middleware)
		requestID := c.Locals("requestid")
		if requestID == nil {
			requestID = c.Get("X-Request-ID", "")
		}

		// Get user context if available
		userID := c.Locals("user_id")
		clientKeyID := c.Locals("client_key_id")

		// Process request
		err := c.Next()

		// Calculate duration
		duration := time.Since(start)
		durationMs := duration.Milliseconds()

		// Get response status
		status := c.Response().StatusCode()

		// Skip successful requests if configured
		if cfg.SkipSuccessfulRequests && status >= 200 && status < 300 {
			return err
		}

		// Determine log level based on status code and duration
		var logEvent *zerolog.Event
		if err != nil {
			logEvent = logger.Error().Err(err)
		} else if status >= 500 {
			logEvent = logger.Error()
		} else if status >= 400 {
			logEvent = logger.Warn()
		} else if cfg.SlowRequestThreshold > 0 && duration > cfg.SlowRequestThreshold {
			logEvent = logger.Warn().Bool("slow_request", true)
		} else {
			logEvent = logger.Info()
		}

		// Build structured log entry
		logEvent = logEvent.
			Str("request_id", toString(requestID)).
			Str("method", c.Method()).
			Str("path", path).
			Str("ip", c.IP()).
			Int("status", status).
			Int64("duration_ms", durationMs).
			Str("user_agent", c.Get("User-Agent")).
			Str("protocol", c.Protocol())

		// Add query string if present (with sensitive params redacted)
		if queryString := string(c.Request().URI().QueryString()); queryString != "" {
			logEvent = logEvent.Str("query", redactQueryString(queryString))
		}

		// Add user context if available
		if userID != nil {
			logEvent = logEvent.Str("user_id", toString(userID))
		}
		if clientKeyID != nil {
			logEvent = logEvent.Str("client_key_id", toString(clientKeyID))
		}

		// Add response size
		logEvent = logEvent.Int("response_bytes", len(c.Response().Body()))

		// Add referer if present
		if referer := c.Get("Referer"); referer != "" {
			logEvent = logEvent.Str("referer", referer)
		}

		// Log request body if configured (be careful with sensitive data)
		if cfg.LogRequestBody && len(c.Body()) > 0 {
			// Limit body size to prevent huge logs
			bodySize := len(c.Body())
			if bodySize > 1024 {
				logEvent = logEvent.Str("request_body", string(c.Body()[:1024])+"... (truncated)")
			} else {
				logEvent = logEvent.Str("request_body", string(c.Body()))
			}
		}

		// Add error details if present
		if err != nil {
			logEvent = logEvent.Str("error", err.Error())
		}

		// Send log
		logEvent.Msg("HTTP request")

		return err
	}
}

// toString safely converts interface{} to string
func toString(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// SlowQueryLogger logs slow database queries
func SlowQueryLogger(threshold time.Duration) func(query string, duration time.Duration, err error) {
	return func(query string, duration time.Duration, err error) {
		if duration > threshold {
			logEvent := log.Warn().
				Dur("duration", duration).
				Int64("duration_ms", duration.Milliseconds()).
				Bool("slow_query", true)

			if err != nil {
				logEvent = logEvent.Err(err)
			}

			// Truncate long queries
			if len(query) > 500 {
				logEvent = logEvent.Str("query", query[:500]+"... (truncated)")
			} else {
				logEvent = logEvent.Str("query", query)
			}

			logEvent.Msg("Slow database query detected")
		}
	}
}

// AuditLogger logs security-sensitive events (auth, user management, config changes)
type AuditLogger struct {
	logger zerolog.Logger
}

// NewAuditLogger creates a new audit logger
func NewAuditLogger(logger zerolog.Logger) *AuditLogger {
	return &AuditLogger{
		logger: logger.With().Str("log_type", "audit").Logger(),
	}
}

// LogAuth logs authentication events
func (al *AuditLogger) LogAuth(c *fiber.Ctx, event, userID, email string, success bool) {
	logEvent := al.logger.Info()
	if !success {
		logEvent = al.logger.Warn()
	}

	logEvent.
		Str("event", event).
		Str("user_id", userID).
		Str("email", email).
		Bool("success", success).
		Str("ip", c.IP()).
		Str("user_agent", c.Get("User-Agent")).
		Str("request_id", c.Get("X-Request-ID")).
		Msg("Authentication event")
}

// LogUserManagement logs user management events
func (al *AuditLogger) LogUserManagement(c *fiber.Ctx, action, targetUserID, performedBy string) {
	al.logger.Info().
		Str("action", action).
		Str("target_user_id", targetUserID).
		Str("performed_by", performedBy).
		Str("ip", c.IP()).
		Str("request_id", c.Get("X-Request-ID")).
		Msg("User management event")
}

// LogClientKeyOperation logs client key operations
func (al *AuditLogger) LogClientKeyOperation(c *fiber.Ctx, action, keyID, keyName, performedBy string) {
	al.logger.Info().
		Str("action", action).
		Str("key_id", keyID).
		Str("key_name", keyName).
		Str("performed_by", performedBy).
		Str("ip", c.IP()).
		Str("request_id", c.Get("X-Request-ID")).
		Msg("Client key operation")
}

// LogConfigChange logs configuration changes
func (al *AuditLogger) LogConfigChange(c *fiber.Ctx, setting, oldValue, newValue, performedBy string) {
	al.logger.Warn().
		Str("setting", setting).
		Str("old_value", oldValue).
		Str("new_value", newValue).
		Str("performed_by", performedBy).
		Str("ip", c.IP()).
		Str("request_id", c.Get("X-Request-ID")).
		Msg("Configuration change")
}

// LogSecurityEvent logs security-related events
func (al *AuditLogger) LogSecurityEvent(c *fiber.Ctx, event, description, severity string) {
	var logEvent *zerolog.Event
	switch severity {
	case "critical":
		logEvent = al.logger.Error()
	case "high":
		logEvent = al.logger.Warn()
	default:
		logEvent = al.logger.Info()
	}

	logEvent.
		Str("event", event).
		Str("description", description).
		Str("severity", severity).
		Str("ip", c.IP()).
		Str("user_agent", c.Get("User-Agent")).
		Str("request_id", c.Get("X-Request-ID")).
		Msg("Security event")
}
