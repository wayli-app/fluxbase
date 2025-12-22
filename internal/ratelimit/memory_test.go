package ratelimit

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMemoryStore(t *testing.T) {
	t.Run("creates store with default gc interval", func(t *testing.T) {
		store := NewMemoryStore(0)
		require.NotNil(t, store)
		assert.NotNil(t, store.data)
		assert.Equal(t, 10*time.Minute, store.gcInterval)
		store.Close()
	})

	t.Run("creates store with custom gc interval", func(t *testing.T) {
		store := NewMemoryStore(5 * time.Minute)
		require.NotNil(t, store)
		assert.Equal(t, 5*time.Minute, store.gcInterval)
		store.Close()
	})

	t.Run("creates store with negative gc interval uses default", func(t *testing.T) {
		store := NewMemoryStore(-1 * time.Minute)
		require.NotNil(t, store)
		assert.Equal(t, 10*time.Minute, store.gcInterval)
		store.Close()
	})
}

func TestMemoryStore_Increment(t *testing.T) {
	store := NewMemoryStore(time.Minute)
	defer store.Close()

	ctx := context.Background()

	t.Run("first increment returns 1", func(t *testing.T) {
		count, err := store.Increment(ctx, "key1", time.Minute)
		require.NoError(t, err)
		assert.Equal(t, int64(1), count)
	})

	t.Run("subsequent increments increase count", func(t *testing.T) {
		count, err := store.Increment(ctx, "key2", time.Minute)
		require.NoError(t, err)
		assert.Equal(t, int64(1), count)

		count, err = store.Increment(ctx, "key2", time.Minute)
		require.NoError(t, err)
		assert.Equal(t, int64(2), count)

		count, err = store.Increment(ctx, "key2", time.Minute)
		require.NoError(t, err)
		assert.Equal(t, int64(3), count)
	})

	t.Run("different keys have independent counters", func(t *testing.T) {
		store.Increment(ctx, "keyA", time.Minute)
		store.Increment(ctx, "keyA", time.Minute)
		store.Increment(ctx, "keyA", time.Minute)

		countA, err := store.Increment(ctx, "keyA", time.Minute)
		require.NoError(t, err)
		assert.Equal(t, int64(4), countA)

		countB, err := store.Increment(ctx, "keyB", time.Minute)
		require.NoError(t, err)
		assert.Equal(t, int64(1), countB)
	})
}

func TestMemoryStore_IncrementExpiration(t *testing.T) {
	store := NewMemoryStore(time.Minute)
	defer store.Close()

	ctx := context.Background()

	// Increment with short expiration
	count, err := store.Increment(ctx, "expiring-key", 50*time.Millisecond)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)

	count, err = store.Increment(ctx, "expiring-key", 50*time.Millisecond)
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	// After expiration, should reset to 1
	count, err = store.Increment(ctx, "expiring-key", 50*time.Millisecond)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
}

func TestMemoryStore_Get(t *testing.T) {
	store := NewMemoryStore(time.Minute)
	defer store.Close()

	ctx := context.Background()

	t.Run("non-existent key returns zero", func(t *testing.T) {
		count, expiry, err := store.Get(ctx, "non-existent")
		require.NoError(t, err)
		assert.Equal(t, int64(0), count)
		assert.True(t, expiry.IsZero())
	})

	t.Run("existing key returns count and expiry", func(t *testing.T) {
		// Increment to create the key
		_, err := store.Increment(ctx, "get-test", time.Minute)
		require.NoError(t, err)
		_, err = store.Increment(ctx, "get-test", time.Minute)
		require.NoError(t, err)

		count, expiry, err := store.Get(ctx, "get-test")
		require.NoError(t, err)
		assert.Equal(t, int64(2), count)
		assert.False(t, expiry.IsZero())
		assert.True(t, expiry.After(time.Now()))
	})

	t.Run("expired key returns zero", func(t *testing.T) {
		// Create a key with short expiration
		_, err := store.Increment(ctx, "expiring-get", 50*time.Millisecond)
		require.NoError(t, err)

		// Wait for expiration
		time.Sleep(100 * time.Millisecond)

		count, expiry, err := store.Get(ctx, "expiring-get")
		require.NoError(t, err)
		assert.Equal(t, int64(0), count)
		assert.True(t, expiry.IsZero())
	})
}

func TestMemoryStore_Reset(t *testing.T) {
	store := NewMemoryStore(time.Minute)
	defer store.Close()

	ctx := context.Background()

	// Create some entries
	store.Increment(ctx, "reset-test", time.Minute)
	store.Increment(ctx, "reset-test", time.Minute)

	count, _, err := store.Get(ctx, "reset-test")
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)

	// Reset the key
	err = store.Reset(ctx, "reset-test")
	require.NoError(t, err)

	// Should now return zero
	count, _, err = store.Get(ctx, "reset-test")
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)
}

func TestMemoryStore_ResetNonExistent(t *testing.T) {
	store := NewMemoryStore(time.Minute)
	defer store.Close()

	ctx := context.Background()

	// Resetting non-existent key should not error
	err := store.Reset(ctx, "non-existent")
	require.NoError(t, err)
}

func TestMemoryStore_Close(t *testing.T) {
	store := NewMemoryStore(time.Minute)

	err := store.Close()
	require.NoError(t, err)

	// Close should be idempotent (though it may panic on double close of channel)
	// This is testing that Close doesn't error
}

func TestMemoryStore_Cleanup(t *testing.T) {
	store := NewMemoryStore(time.Minute)
	defer store.Close()

	ctx := context.Background()

	// Create some entries with short expiration
	store.Increment(ctx, "cleanup-1", 50*time.Millisecond)
	store.Increment(ctx, "cleanup-2", 50*time.Millisecond)
	store.Increment(ctx, "cleanup-3", time.Hour) // This one should survive

	// Verify entries exist
	store.mu.RLock()
	assert.Len(t, store.data, 3)
	store.mu.RUnlock()

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	// Trigger cleanup manually
	store.cleanup()

	// Only the non-expired entry should remain
	store.mu.RLock()
	assert.Len(t, store.data, 1)
	_, exists := store.data["cleanup-3"]
	assert.True(t, exists)
	store.mu.RUnlock()
}

func TestMemoryStore_ConcurrentAccess(t *testing.T) {
	store := NewMemoryStore(time.Minute)
	defer store.Close()

	ctx := context.Background()
	var wg sync.WaitGroup
	numGoroutines := 50
	incrementsPerGoroutine := 100

	// Concurrent increments on the same key
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < incrementsPerGoroutine; j++ {
				_, err := store.Increment(ctx, "concurrent-key", time.Minute)
				assert.NoError(t, err)
			}
		}()
	}

	wg.Wait()

	// Final count should be exactly numGoroutines * incrementsPerGoroutine
	count, _, err := store.Get(ctx, "concurrent-key")
	require.NoError(t, err)
	assert.Equal(t, int64(numGoroutines*incrementsPerGoroutine), count)
}

func TestMemoryStore_ConcurrentMixedOperations(t *testing.T) {
	store := NewMemoryStore(time.Minute)
	defer store.Close()

	ctx := context.Background()
	var wg sync.WaitGroup

	// Concurrent increments, gets, and resets
	for i := 0; i < 20; i++ {
		wg.Add(3)

		// Incrementer
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				store.Increment(ctx, "mixed-ops", time.Minute)
			}
		}()

		// Getter
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				store.Get(ctx, "mixed-ops")
			}
		}()

		// Occasional resetter
		go func() {
			defer wg.Done()
			time.Sleep(10 * time.Millisecond)
			store.Reset(ctx, "mixed-ops")
		}()
	}

	wg.Wait()
	// If we got here without deadlock or panic, the test passes
}

func TestMemoryStore_MultipleKeys(t *testing.T) {
	store := NewMemoryStore(time.Minute)
	defer store.Close()

	ctx := context.Background()

	// Create many different keys
	numKeys := 100
	for i := 0; i < numKeys; i++ {
		key := string(rune('A' + i%26))
		_, err := store.Increment(ctx, key, time.Minute)
		require.NoError(t, err)
	}

	// Verify we have at least some entries
	store.mu.RLock()
	assert.Greater(t, len(store.data), 0)
	store.mu.RUnlock()
}

func TestMemoryStore_RateLimitScenario(t *testing.T) {
	store := NewMemoryStore(time.Minute)
	defer store.Close()

	ctx := context.Background()
	limit := int64(10)
	window := 100 * time.Millisecond

	// Simulate rate limiting
	for i := 0; i < 15; i++ {
		count, err := store.Increment(ctx, "rate-limit-test", window)
		require.NoError(t, err)

		if count > limit {
			// Should be rate limited
			assert.Greater(t, int64(i+1), limit, "should be rate limited after exceeding limit")
		}
	}

	// Verify we're over the limit
	count, _, err := store.Get(ctx, "rate-limit-test")
	require.NoError(t, err)
	assert.Equal(t, int64(15), count)

	// Wait for window to expire
	time.Sleep(150 * time.Millisecond)

	// Should be able to make requests again
	count, err = store.Increment(ctx, "rate-limit-test", window)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
}

func TestMemoryStore_ExpiryTiming(t *testing.T) {
	store := NewMemoryStore(time.Minute)
	defer store.Close()

	ctx := context.Background()
	expiration := 200 * time.Millisecond

	// Create entry
	_, err := store.Increment(ctx, "timing-test", expiration)
	require.NoError(t, err)

	// Check expiry time
	_, expiry, err := store.Get(ctx, "timing-test")
	require.NoError(t, err)

	expectedExpiry := time.Now().Add(expiration)
	// Allow 50ms tolerance
	assert.WithinDuration(t, expectedExpiry, expiry, 50*time.Millisecond)
}

func TestMemoryStore_ContextCancellation(t *testing.T) {
	store := NewMemoryStore(time.Minute)
	defer store.Close()

	// Operations should still work even with cancelled context
	// (the in-memory store doesn't actually check context for basic ops)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// These operations should still succeed since they're in-memory
	count, err := store.Increment(ctx, "cancelled-ctx", time.Minute)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)

	count, _, err = store.Get(ctx, "cancelled-ctx")
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)

	err = store.Reset(ctx, "cancelled-ctx")
	require.NoError(t, err)
}
