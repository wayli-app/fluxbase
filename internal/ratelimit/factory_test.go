package ratelimit

import (
	"testing"

	"github.com/fluxbase-eu/fluxbase/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStore(t *testing.T) {
	t.Run("creates memory store for empty backend", func(t *testing.T) {
		cfg := &config.ScalingConfig{
			Backend: "",
		}

		store, err := NewStore(cfg, nil)
		require.NoError(t, err)
		require.NotNil(t, store)
		defer store.Close()

		// Verify it's a memory store
		_, ok := store.(*MemoryStore)
		assert.True(t, ok, "should be MemoryStore")
	})

	t.Run("creates memory store for local backend", func(t *testing.T) {
		cfg := &config.ScalingConfig{
			Backend: "local",
		}

		store, err := NewStore(cfg, nil)
		require.NoError(t, err)
		require.NotNil(t, store)
		defer store.Close()

		_, ok := store.(*MemoryStore)
		assert.True(t, ok, "should be MemoryStore")
	})

	t.Run("errors for postgres backend without pool", func(t *testing.T) {
		cfg := &config.ScalingConfig{
			Backend: "postgres",
		}

		store, err := NewStore(cfg, nil)
		require.Error(t, err)
		assert.Nil(t, store)
		assert.Contains(t, err.Error(), "database pool is required")
	})

	t.Run("errors for redis backend without url", func(t *testing.T) {
		cfg := &config.ScalingConfig{
			Backend:  "redis",
			RedisURL: "",
		}

		store, err := NewStore(cfg, nil)
		require.Error(t, err)
		assert.Nil(t, store)
		assert.Contains(t, err.Error(), "redis_url is required")
	})

	t.Run("errors for redis backend with invalid url", func(t *testing.T) {
		cfg := &config.ScalingConfig{
			Backend:  "redis",
			RedisURL: "invalid://url",
		}

		store, err := NewStore(cfg, nil)
		require.Error(t, err)
		assert.Nil(t, store)
		assert.Contains(t, err.Error(), "failed to connect to Redis")
	})

	t.Run("errors for unknown backend", func(t *testing.T) {
		cfg := &config.ScalingConfig{
			Backend: "memcached",
		}

		store, err := NewStore(cfg, nil)
		require.Error(t, err)
		assert.Nil(t, store)
		assert.Contains(t, err.Error(), "unknown rate limit backend")
		assert.Contains(t, err.Error(), "valid options: local, postgres, redis")
	})
}

func TestGlobalStore(t *testing.T) {
	// Save original global store
	originalStore := GlobalStore

	t.Cleanup(func() {
		// Restore original global store
		GlobalStore = originalStore
	})

	t.Run("GetGlobalStore returns fallback when nil", func(t *testing.T) {
		GlobalStore = nil

		store := GetGlobalStore()
		require.NotNil(t, store)

		// Should have set it as the new global
		assert.Equal(t, store, GlobalStore)

		// Should be a MemoryStore
		_, ok := store.(*MemoryStore)
		assert.True(t, ok, "fallback should be MemoryStore")
	})

	t.Run("SetGlobalStore sets the global instance", func(t *testing.T) {
		GlobalStore = nil

		newStore := NewMemoryStore(0)
		SetGlobalStore(newStore)

		assert.Same(t, newStore, GlobalStore)
	})

	t.Run("SetGlobalStore closes existing store", func(t *testing.T) {
		// Set an initial store
		oldStore := NewMemoryStore(0)
		GlobalStore = oldStore

		// Set a new one
		newStore := NewMemoryStore(0)
		SetGlobalStore(newStore)

		// New one should be set (pointer comparison)
		assert.Same(t, newStore, GlobalStore)
		assert.NotSame(t, oldStore, GlobalStore)
	})

	t.Run("GetGlobalStore returns set store", func(t *testing.T) {
		newStore := NewMemoryStore(0)
		GlobalStore = newStore

		store := GetGlobalStore()
		assert.Same(t, newStore, store)
	})
}
