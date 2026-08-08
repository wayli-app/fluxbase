package database

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// RetryConfig.backoffFor Tests
// =============================================================================

func TestRetryConfig_BackoffFor_EscalatesAndCaps(t *testing.T) {
	rc := RetryConfig{InitialBackoff: 1 * time.Second, MaxBackoff: 10 * time.Second}

	// First failure -> base of initial backoff, jittered into [base/2, base].
	d := rc.backoffFor(1)
	assert.GreaterOrEqual(t, d, 500*time.Millisecond)
	assert.LessOrEqual(t, d, time.Second)

	// Higher failure counts escalate but must never exceed the cap.
	for i := 1; i <= 30; i++ {
		d := rc.backoffFor(i)
		assert.LessOrEqual(t, d, rc.MaxBackoff, "iter %d exceeded cap", i)
	}
}

func TestRetryConfig_BackoffFor_ZeroAndNegative(t *testing.T) {
	rc := RetryConfig{InitialBackoff: 1 * time.Second, MaxBackoff: 10 * time.Second}
	assert.Equal(t, time.Duration(0), rc.backoffFor(0))
	assert.Equal(t, time.Duration(0), rc.backoffFor(-3))

	// Zero initial backoff always yields zero.
	rc0 := RetryConfig{InitialBackoff: 0, MaxBackoff: 10 * time.Second}
	assert.Equal(t, time.Duration(0), rc0.backoffFor(5))
}

// =============================================================================
// RetryTransient Tests
// =============================================================================

func TestRetryTransient_SuccessFirstTry(t *testing.T) {
	var calls int32
	rc := RetryConfig{MaxAttempts: 5, InitialBackoff: 10 * time.Millisecond, MaxBackoff: 100 * time.Millisecond}
	err := RetryTransient(context.Background(), "step", rc, func() error {
		atomic.AddInt32(&calls, 1)
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, int32(1), atomic.LoadInt32(&calls))
}

func TestRetryTransient_RetriesUntilSuccess(t *testing.T) {
	var calls int32
	rc := RetryConfig{MaxAttempts: 5, InitialBackoff: 1 * time.Millisecond, MaxBackoff: 10 * time.Millisecond}
	err := RetryTransient(context.Background(), "step", rc, func() error {
		n := atomic.AddInt32(&calls, 1)
		if n < 3 {
			return errors.New("dial tcp: connection refused") // transient
		}
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, int32(3), atomic.LoadInt32(&calls), "should succeed on the 3rd call")
}

func TestRetryTransient_ExhaustsAttemptsOnPersistentTransient(t *testing.T) {
	var calls int32
	rc := RetryConfig{MaxAttempts: 4, InitialBackoff: 1 * time.Millisecond, MaxBackoff: 5 * time.Millisecond}
	err := RetryTransient(context.Background(), "step", rc, func() error {
		atomic.AddInt32(&calls, 1)
		return errors.New("connection refused")
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "step")
	assert.Equal(t, int32(4), atomic.LoadInt32(&calls))
}

func TestRetryTransient_NonTransientFailsFast(t *testing.T) {
	var calls int32
	rc := RetryConfig{MaxAttempts: 10, InitialBackoff: 1 * time.Millisecond, MaxBackoff: 10 * time.Millisecond}
	sentinel := errors.New("permission denied")
	err := RetryTransient(context.Background(), "step", rc, func() error {
		atomic.AddInt32(&calls, 1)
		return sentinel
	})
	// Non-transient must return the original error immediately, single call.
	require.ErrorIs(t, err, sentinel)
	assert.Equal(t, int32(1), atomic.LoadInt32(&calls), "non-transient should not be retried")
}

func TestRetryTransient_RespectsContextCancellation(t *testing.T) {
	var calls int32
	rc := RetryConfig{MaxAttempts: 100, InitialBackoff: 50 * time.Millisecond, MaxBackoff: 50 * time.Millisecond}

	ctx, cancel := context.WithCancel(context.Background())
	// Cancel shortly after the first attempt so the backoff sleep is interrupted.
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	err := RetryTransient(ctx, "step", rc, func() error {
		atomic.AddInt32(&calls, 1)
		return errors.New("connection refused") // transient -> would retry
	})
	elapsed := time.Since(start)

	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
	// Must NOT have waited through many 50ms backoffs.
	assert.Less(t, elapsed, 500*time.Millisecond, "should abort promptly on ctx cancel")
}

func TestRetryTransient_PreCancelledContext(t *testing.T) {
	rc := RetryConfig{MaxAttempts: 5, InitialBackoff: 1 * time.Millisecond, MaxBackoff: 10 * time.Millisecond}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var calls int32
	err := RetryTransient(ctx, "step", rc, func() error {
		atomic.AddInt32(&calls, 1)
		return nil
	})
	require.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, int32(0), atomic.LoadInt32(&calls), "should not run fn on pre-cancelled ctx")
}
