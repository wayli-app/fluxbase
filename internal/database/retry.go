package database

import (
	"context"
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/config"
)

// RetryConfig controls the retry behavior of RetryTransient and ConnectWithRetry.
type RetryConfig struct {
	MaxAttempts    int           // total attempts (>=1)
	InitialBackoff time.Duration // backoff for the first retry
	MaxBackoff     time.Duration // cap on each backoff
}

// DefaultRetryConfig returns sensible defaults: 8 attempts, 1s..30s backoff.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:    8,
		InitialBackoff: 1 * time.Second,
		MaxBackoff:     30 * time.Second,
	}
}

// backoffFor returns the (jittered) backoff to wait before attempt n+1, where n
// is the number of failures so far (n>=1). It is exponential with full jitter,
// capped at MaxBackoff. The returned value is suitable for selecting against a
// context so callers can be interrupted during the wait.
func (rc RetryConfig) backoffFor(failures int) time.Duration {
	if failures <= 0 || rc.InitialBackoff <= 0 {
		return 0
	}
	// 2^(failures-1) * initial, capped at MaxBackoff.
	shift := uint(failures - 1)
	if shift > 20 {
		shift = 20
	}
	d := rc.InitialBackoff << shift
	if rc.MaxBackoff > 0 && d > rc.MaxBackoff {
		d = rc.MaxBackoff
	}
	// Full jitter in [d/2, d].
	jitter := time.Duration(rand.Int64N(int64(d/2 + 1)))
	return d/2 + jitter
}

// sleepCtx waits for d or until ctx is cancelled. Returns ctx.Err() if cancelled.
func sleepCtx(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// RetryTransient runs fn, retrying only while it returns a transient error (per
// IsTransientError). Non-transient errors and nil return immediately. Between
// retries it sleeps with exponential backoff (jittered, capped) that is
// interruptible by ctx. name is used in log lines to identify the step.
//
// This wraps DB-dependent startup steps (bootstrap, schema apply, initial
// connection) so a transient blip after the initial connect doesn't abort
// startup. On persistent real errors the original error is returned unchanged.
func RetryTransient(ctx context.Context, name string, rc RetryConfig, fn func() error) error {
	if rc.MaxAttempts < 1 {
		rc.MaxAttempts = 1
	}
	var lastErr error
	for attempt := 1; attempt <= rc.MaxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}

		err := fn()
		if err == nil {
			if attempt > 1 {
				log.Info().Str("step", name).Int("attempt", attempt).Msg("Step succeeded after retry")
			}
			return nil
		}
		lastErr = err

		// Non-transient errors fail fast — don't retry a real permission/syntax error.
		if !IsTransientError(err) {
			return err
		}

		if attempt >= rc.MaxAttempts {
			break
		}
		backoff := rc.backoffFor(attempt)
		log.Warn().
			Err(err).
			Str("step", name).
			Int("attempt", attempt).
			Int("max_attempts", rc.MaxAttempts).
			Dur("retry_in", backoff).
			Msg("Transient error, retrying")
		if werr := sleepCtx(ctx, backoff); werr != nil {
			return werr
		}
	}
	return fmt.Errorf("%s failed after %d attempts: %w", name, rc.MaxAttempts, lastErr)
}

// ConnectWithRetry attempts to connect to the database with exponential backoff.
//
// It retries while NewConnection returns a transient error (per IsTransientError)
// and fails fast on non-transient errors. The backoff is interruptible by ctx so
// SIGTERM during startup isn't delayed. maxAttempts <= 0 falls back to a default.
func ConnectWithRetry(cfg config.DatabaseConfig, maxAttempts int) (*Connection, error) {
	if maxAttempts <= 0 {
		maxAttempts = 8
	}
	rc := RetryConfig{
		MaxAttempts:    maxAttempts,
		InitialBackoff: 1 * time.Second,
		MaxBackoff:     30 * time.Second,
	}

	var db *Connection
	// RetryTransient has no cancellation channel from the caller (main blocks on
	// this returning), so use a background context; backoff is still bounded.
	err := RetryTransient(context.Background(), "database connect", rc, func() error {
		log.Info().
			Str("host", cfg.Host).
			Int("port", cfg.Port).
			Msg("Attempting to connect to database...")
		conn, err := NewConnection(cfg)
		if err != nil {
			return err
		}
		db = conn
		return nil
	})
	if err != nil {
		return nil, err
	}
	log.Info().Msg("Successfully connected to database")
	return db, nil
}
