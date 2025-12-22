package auth

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewNonceStore(t *testing.T) {
	store := NewNonceStore()
	require.NotNil(t, store)
	assert.NotNil(t, store.nonces)
	assert.Empty(t, store.nonces)
}

func TestNonceStore_Set(t *testing.T) {
	store := NewNonceStore()

	t.Run("stores nonce with user ID and TTL", func(t *testing.T) {
		store.Set("nonce123", "user456", 5*time.Minute)

		store.mu.RLock()
		entry, exists := store.nonces["nonce123"]
		store.mu.RUnlock()

		assert.True(t, exists)
		assert.Equal(t, "user456", entry.userID)
		assert.True(t, entry.expiry.After(time.Now()))
	})

	t.Run("overwrites existing nonce", func(t *testing.T) {
		store.Set("nonce-overwrite", "user1", time.Minute)
		store.Set("nonce-overwrite", "user2", time.Minute)

		store.mu.RLock()
		entry := store.nonces["nonce-overwrite"]
		store.mu.RUnlock()

		assert.Equal(t, "user2", entry.userID)
	})
}

func TestNonceStore_Validate(t *testing.T) {
	store := NewNonceStore()

	t.Run("returns true for valid nonce", func(t *testing.T) {
		store.Set("valid-nonce", "user123", 5*time.Minute)

		result := store.Validate("valid-nonce", "user123")
		assert.True(t, result)
	})

	t.Run("removes nonce after validation (single-use)", func(t *testing.T) {
		store.Set("single-use-nonce", "user123", 5*time.Minute)

		// First validation succeeds
		result1 := store.Validate("single-use-nonce", "user123")
		assert.True(t, result1)

		// Second validation fails (nonce already used)
		result2 := store.Validate("single-use-nonce", "user123")
		assert.False(t, result2)
	})

	t.Run("returns false for non-existent nonce", func(t *testing.T) {
		result := store.Validate("nonexistent", "user123")
		assert.False(t, result)
	})

	t.Run("returns false for wrong user", func(t *testing.T) {
		store.Set("user-specific-nonce", "user123", 5*time.Minute)

		result := store.Validate("user-specific-nonce", "wrong-user")
		assert.False(t, result)

		// Nonce should be deleted even on failed validation
		store.mu.RLock()
		_, exists := store.nonces["user-specific-nonce"]
		store.mu.RUnlock()
		assert.False(t, exists)
	})

	t.Run("returns false for expired nonce", func(t *testing.T) {
		store.Set("expired-nonce", "user123", 10*time.Millisecond)

		// Wait for expiry
		time.Sleep(50 * time.Millisecond)

		result := store.Validate("expired-nonce", "user123")
		assert.False(t, result)
	})
}

func TestNonceStore_Cleanup(t *testing.T) {
	store := NewNonceStore()

	// Add some nonces with different expiries
	store.Set("expires-soon", "user1", 10*time.Millisecond)
	store.Set("expires-later", "user2", 5*time.Minute)
	store.Set("also-expires-soon", "user3", 10*time.Millisecond)

	// Wait for some to expire
	time.Sleep(50 * time.Millisecond)

	// Run cleanup
	store.Cleanup()

	store.mu.RLock()
	defer store.mu.RUnlock()

	// Only the non-expired nonce should remain
	assert.Len(t, store.nonces, 1)
	_, exists := store.nonces["expires-later"]
	assert.True(t, exists)
}

func TestNonceStore_StartCleanup(t *testing.T) {
	store := NewNonceStore()

	// Add a nonce with very short expiry
	store.Set("auto-cleanup-nonce", "user1", 10*time.Millisecond)

	// Start cleanup with short interval
	stop := store.StartCleanup(50 * time.Millisecond)

	// Wait for expiry and cleanup to run
	time.Sleep(100 * time.Millisecond)

	store.mu.RLock()
	_, exists := store.nonces["auto-cleanup-nonce"]
	store.mu.RUnlock()

	assert.False(t, exists, "expired nonce should have been cleaned up")

	// Stop the cleanup goroutine
	close(stop)

	// Give goroutine time to exit
	time.Sleep(10 * time.Millisecond)
}

func TestNonceStore_ConcurrentAccess(t *testing.T) {
	store := NewNonceStore()
	var wg sync.WaitGroup
	numGoroutines := 50

	// Concurrent Set operations
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				store.Set(string(rune('A'+id)), "user", time.Minute)
			}
		}(i)
	}

	// Concurrent Validate operations
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				store.Validate(string(rune('A'+id)), "user")
			}
		}(i)
	}

	// Concurrent Cleanup operations
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 5; j++ {
				store.Cleanup()
				time.Sleep(time.Millisecond)
			}
		}()
	}

	wg.Wait()
	// If we got here without deadlock or panic, the test passes
}

func TestNonceStore_EdgeCases(t *testing.T) {
	store := NewNonceStore()

	t.Run("empty nonce string", func(t *testing.T) {
		store.Set("", "user123", time.Minute)
		result := store.Validate("", "user123")
		assert.True(t, result)
	})

	t.Run("empty user ID", func(t *testing.T) {
		store.Set("nonce-empty-user", "", time.Minute)
		result := store.Validate("nonce-empty-user", "")
		assert.True(t, result)
	})

	t.Run("zero TTL", func(t *testing.T) {
		store.Set("zero-ttl-nonce", "user123", 0)
		// With zero TTL, the nonce expires immediately
		result := store.Validate("zero-ttl-nonce", "user123")
		assert.False(t, result)
	})

	t.Run("negative TTL", func(t *testing.T) {
		store.Set("negative-ttl-nonce", "user123", -time.Minute)
		// With negative TTL, the nonce is already expired
		result := store.Validate("negative-ttl-nonce", "user123")
		assert.False(t, result)
	})
}
