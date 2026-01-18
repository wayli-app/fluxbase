package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

// ErrTOTPRateLimitExceeded is returned when a user has exceeded the TOTP attempt limit
var ErrTOTPRateLimitExceeded = errors.New("too many 2FA attempts, please try again later")

// TOTPRateLimiter provides rate limiting for TOTP verification attempts
type TOTPRateLimiter struct {
	db              *pgxpool.Pool
	maxAttempts     int           // Maximum failed attempts allowed within the window
	windowDuration  time.Duration // Time window for counting attempts
	lockoutDuration time.Duration // How long to lock out after exceeding limit
}

// TOTPRateLimiterConfig holds configuration for the TOTP rate limiter
type TOTPRateLimiterConfig struct {
	MaxAttempts     int           // Maximum failed attempts allowed (default: 5)
	WindowDuration  time.Duration // Time window for counting attempts (default: 5 minutes)
	LockoutDuration time.Duration // How long to lock out after exceeding limit (default: 15 minutes)
}

// DefaultTOTPRateLimiterConfig returns the default rate limiter configuration
func DefaultTOTPRateLimiterConfig() TOTPRateLimiterConfig {
	return TOTPRateLimiterConfig{
		MaxAttempts:     5,
		WindowDuration:  5 * time.Minute,
		LockoutDuration: 15 * time.Minute,
	}
}

// NewTOTPRateLimiter creates a new TOTP rate limiter
func NewTOTPRateLimiter(db *pgxpool.Pool, config TOTPRateLimiterConfig) *TOTPRateLimiter {
	if config.MaxAttempts <= 0 {
		config.MaxAttempts = 5
	}
	if config.WindowDuration <= 0 {
		config.WindowDuration = 5 * time.Minute
	}
	if config.LockoutDuration <= 0 {
		config.LockoutDuration = 15 * time.Minute
	}

	return &TOTPRateLimiter{
		db:              db,
		maxAttempts:     config.MaxAttempts,
		windowDuration:  config.WindowDuration,
		lockoutDuration: config.LockoutDuration,
	}
}

// CheckRateLimit checks if the user has exceeded the TOTP attempt limit.
// Returns nil if the user is allowed to attempt, or ErrTOTPRateLimitExceeded if blocked.
func (r *TOTPRateLimiter) CheckRateLimit(ctx context.Context, userID string) error {
	// Count failed attempts within the window
	query := `
		SELECT COUNT(*)
		FROM auth.two_factor_recovery_attempts
		WHERE user_id = $1
		  AND success = FALSE
		  AND attempted_at > $2
	`

	windowStart := time.Now().Add(-r.windowDuration)
	var failedAttempts int

	err := r.db.QueryRow(ctx, query, userID, windowStart).Scan(&failedAttempts)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		// Log error but don't block the user - fail open for availability
		log.Warn().Err(err).Str("user_id", userID).Msg("Failed to check TOTP rate limit")
		return nil
	}

	if failedAttempts >= r.maxAttempts {
		// Check if the lockout period has passed since the last failed attempt
		var lastAttemptTime time.Time
		lastAttemptQuery := `
			SELECT MAX(attempted_at)
			FROM auth.two_factor_recovery_attempts
			WHERE user_id = $1 AND success = FALSE
		`
		err := r.db.QueryRow(ctx, lastAttemptQuery, userID).Scan(&lastAttemptTime)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			log.Warn().Err(err).Str("user_id", userID).Msg("Failed to get last TOTP attempt time")
			return nil
		}

		lockoutEnd := lastAttemptTime.Add(r.lockoutDuration)
		if time.Now().Before(lockoutEnd) {
			// Still in lockout period
			log.Warn().
				Str("user_id", userID).
				Int("failed_attempts", failedAttempts).
				Time("lockout_ends", lockoutEnd).
				Msg("TOTP rate limit exceeded")

			// Log security event
			LogSecurityEvent(SecurityEvent{
				Type:   SecurityEventRateLimitExceeded,
				UserID: userID,
				Details: map[string]interface{}{
					"reason":          "totp_rate_limit",
					"failed_attempts": failedAttempts,
					"window_minutes":  r.windowDuration.Minutes(),
					"lockout_ends":    lockoutEnd.Format(time.RFC3339),
				},
			})

			return ErrTOTPRateLimitExceeded
		}
	}

	return nil
}

// RecordAttempt records a TOTP verification attempt in the database.
// This should be called after every verification attempt (success or failure).
func (r *TOTPRateLimiter) RecordAttempt(ctx context.Context, userID string, success bool, ipAddress, userAgent string) error {
	query := `
		INSERT INTO auth.two_factor_recovery_attempts (user_id, code_used, success, ip_address, user_agent)
		VALUES ($1, $2, $3, $4::inet, $5)
	`

	// Don't store the actual code for security reasons
	codeUsed := "totp_code"
	if success {
		codeUsed = "totp_code_success"
	}

	_, err := r.db.Exec(ctx, query, userID, codeUsed, success, nilIfEmpty(ipAddress), userAgent)
	if err != nil {
		return fmt.Errorf("failed to record TOTP attempt: %w", err)
	}

	// If successful, clear the failed attempt counter by recording a success
	// The rate limiter will see the success and allow future attempts
	if success {
		log.Debug().Str("user_id", userID).Msg("TOTP verification successful, rate limit counter effectively reset")
	}

	return nil
}

// ClearFailedAttempts clears all failed TOTP attempts for a user.
// This can be called after a successful password reset or by an admin.
func (r *TOTPRateLimiter) ClearFailedAttempts(ctx context.Context, userID string) error {
	query := `
		DELETE FROM auth.two_factor_recovery_attempts
		WHERE user_id = $1 AND success = FALSE
	`

	_, err := r.db.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to clear TOTP failed attempts: %w", err)
	}

	log.Info().Str("user_id", userID).Msg("Cleared TOTP failed attempts")
	return nil
}

// GetFailedAttemptCount returns the number of recent failed TOTP attempts for a user.
func (r *TOTPRateLimiter) GetFailedAttemptCount(ctx context.Context, userID string) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM auth.two_factor_recovery_attempts
		WHERE user_id = $1
		  AND success = FALSE
		  AND attempted_at > $2
	`

	windowStart := time.Now().Add(-r.windowDuration)
	var count int

	err := r.db.QueryRow(ctx, query, userID, windowStart).Scan(&count)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return 0, fmt.Errorf("failed to get TOTP attempt count: %w", err)
	}

	return count, nil
}

// nilIfEmpty returns nil if the string is empty, otherwise returns a pointer to the string
func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
