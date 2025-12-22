package ratelimit

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheck(t *testing.T) {
	store := NewMemoryStore(time.Minute)
	defer store.Close()

	ctx := context.Background()

	t.Run("allows requests under limit", func(t *testing.T) {
		result, err := Check(ctx, store, "check-under-limit", 10, time.Minute)
		require.NoError(t, err)

		assert.True(t, result.Allowed)
		assert.Equal(t, int64(10), result.Limit)
		assert.Equal(t, int64(9), result.Remaining)
		assert.False(t, result.ResetAt.IsZero())
	})

	t.Run("tracks remaining correctly", func(t *testing.T) {
		for i := 1; i <= 5; i++ {
			result, err := Check(ctx, store, "check-remaining", 10, time.Minute)
			require.NoError(t, err)

			assert.True(t, result.Allowed)
			assert.Equal(t, int64(10-i), result.Remaining)
		}
	})

	t.Run("denies requests at limit", func(t *testing.T) {
		key := "check-at-limit"

		// Use up all requests
		for i := 0; i < 5; i++ {
			_, err := Check(ctx, store, key, 5, time.Minute)
			require.NoError(t, err)
		}

		// Next request should be denied
		result, err := Check(ctx, store, key, 5, time.Minute)
		require.NoError(t, err)

		assert.False(t, result.Allowed)
		assert.Equal(t, int64(0), result.Remaining)
		assert.Equal(t, int64(5), result.Limit)
	})

	t.Run("remaining never goes negative", func(t *testing.T) {
		key := "check-not-negative"

		// Use up all requests and then some
		for i := 0; i < 15; i++ {
			result, err := Check(ctx, store, key, 10, time.Minute)
			require.NoError(t, err)

			// Remaining should never be negative
			assert.GreaterOrEqual(t, result.Remaining, int64(0))
		}
	})

	t.Run("reset time is in the future", func(t *testing.T) {
		result, err := Check(ctx, store, "check-reset-time", 10, time.Minute)
		require.NoError(t, err)

		assert.True(t, result.ResetAt.After(time.Now()))
		// Should be approximately 1 minute from now
		expectedReset := time.Now().Add(time.Minute)
		assert.WithinDuration(t, expectedReset, result.ResetAt, 5*time.Second)
	})
}

func TestResult(t *testing.T) {
	t.Run("result with all fields", func(t *testing.T) {
		resetAt := time.Now().Add(time.Minute)
		result := &Result{
			Allowed:   true,
			Remaining: 5,
			ResetAt:   resetAt,
			Limit:     10,
		}

		assert.True(t, result.Allowed)
		assert.Equal(t, int64(5), result.Remaining)
		assert.Equal(t, resetAt, result.ResetAt)
		assert.Equal(t, int64(10), result.Limit)
	})

	t.Run("result when denied", func(t *testing.T) {
		result := &Result{
			Allowed:   false,
			Remaining: 0,
			Limit:     10,
		}

		assert.False(t, result.Allowed)
		assert.Equal(t, int64(0), result.Remaining)
	})
}

func TestCheckWithExpiration(t *testing.T) {
	store := NewMemoryStore(time.Minute)
	defer store.Close()

	ctx := context.Background()
	key := "check-expiration"

	// Use up all requests
	for i := 0; i < 5; i++ {
		_, err := Check(ctx, store, key, 5, 100*time.Millisecond)
		require.NoError(t, err)
	}

	// Should be denied
	result, err := Check(ctx, store, key, 5, 100*time.Millisecond)
	require.NoError(t, err)
	assert.False(t, result.Allowed)

	// Wait for window to expire
	time.Sleep(150 * time.Millisecond)

	// Should be allowed again
	result, err = Check(ctx, store, key, 5, 100*time.Millisecond)
	require.NoError(t, err)
	assert.True(t, result.Allowed)
	assert.Equal(t, int64(4), result.Remaining)
}
