package pubsub

import (
	"testing"

	"github.com/fluxbase-eu/fluxbase/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPubSub(t *testing.T) {
	t.Run("creates local pubsub for empty backend", func(t *testing.T) {
		cfg := &config.ScalingConfig{
			Backend: "",
		}

		ps, err := NewPubSub(cfg, nil)
		require.NoError(t, err)
		require.NotNil(t, ps)
		defer ps.Close()

		// Verify it's a local pubsub
		_, ok := ps.(*LocalPubSub)
		assert.True(t, ok, "should be LocalPubSub")
	})

	t.Run("creates local pubsub for local backend", func(t *testing.T) {
		cfg := &config.ScalingConfig{
			Backend: "local",
		}

		ps, err := NewPubSub(cfg, nil)
		require.NoError(t, err)
		require.NotNil(t, ps)
		defer ps.Close()

		_, ok := ps.(*LocalPubSub)
		assert.True(t, ok, "should be LocalPubSub")
	})

	t.Run("errors for postgres backend without pool", func(t *testing.T) {
		cfg := &config.ScalingConfig{
			Backend: "postgres",
		}

		ps, err := NewPubSub(cfg, nil)
		require.Error(t, err)
		assert.Nil(t, ps)
		assert.Contains(t, err.Error(), "database pool is required")
	})

	t.Run("errors for redis backend without url", func(t *testing.T) {
		cfg := &config.ScalingConfig{
			Backend:  "redis",
			RedisURL: "",
		}

		ps, err := NewPubSub(cfg, nil)
		require.Error(t, err)
		assert.Nil(t, ps)
		assert.Contains(t, err.Error(), "redis_url is required")
	})

	t.Run("errors for redis backend with invalid url", func(t *testing.T) {
		cfg := &config.ScalingConfig{
			Backend:  "redis",
			RedisURL: "invalid://url",
		}

		ps, err := NewPubSub(cfg, nil)
		require.Error(t, err)
		assert.Nil(t, ps)
		assert.Contains(t, err.Error(), "failed to connect to Redis")
	})

	t.Run("errors for unknown backend", func(t *testing.T) {
		cfg := &config.ScalingConfig{
			Backend: "unknown",
		}

		ps, err := NewPubSub(cfg, nil)
		require.Error(t, err)
		assert.Nil(t, ps)
		assert.Contains(t, err.Error(), "unknown pub/sub backend")
		assert.Contains(t, err.Error(), "valid options: local, postgres, redis")
	})
}

func TestGlobalPubSub(t *testing.T) {
	// Save original global pubsub
	originalPubSub := GlobalPubSub

	t.Cleanup(func() {
		// Restore original global pubsub
		GlobalPubSub = originalPubSub
	})

	t.Run("GetGlobalPubSub returns fallback when nil", func(t *testing.T) {
		GlobalPubSub = nil

		ps := GetGlobalPubSub()
		require.NotNil(t, ps)

		// Should have set it as the new global
		assert.Equal(t, ps, GlobalPubSub)

		// Should be a LocalPubSub
		_, ok := ps.(*LocalPubSub)
		assert.True(t, ok, "fallback should be LocalPubSub")
	})

	t.Run("SetGlobalPubSub sets the global instance", func(t *testing.T) {
		GlobalPubSub = nil

		newPS := NewLocalPubSub()
		SetGlobalPubSub(newPS)

		assert.Equal(t, newPS, GlobalPubSub)
	})

	t.Run("SetGlobalPubSub closes existing pubsub", func(t *testing.T) {
		// Set an initial pubsub
		oldPS := NewLocalPubSub()
		GlobalPubSub = oldPS

		// Set a new one
		newPS := NewLocalPubSub()
		SetGlobalPubSub(newPS)

		// New one should be set (comparing pointers, not values)
		assert.Same(t, newPS, GlobalPubSub)
		// The old pubsub should have been replaced (pointer comparison)
		assert.NotSame(t, oldPS, GlobalPubSub)
	})

	t.Run("GetGlobalPubSub returns set pubsub", func(t *testing.T) {
		newPS := NewLocalPubSub()
		GlobalPubSub = newPS

		ps := GetGlobalPubSub()
		assert.Equal(t, newPS, ps)
	})
}

func TestMessageStruct(t *testing.T) {
	t.Run("message with all fields", func(t *testing.T) {
		msg := Message{
			Channel: "test-channel",
			Payload: []byte("test payload"),
		}

		assert.Equal(t, "test-channel", msg.Channel)
		assert.Equal(t, []byte("test payload"), msg.Payload)
	})

	t.Run("empty message", func(t *testing.T) {
		msg := Message{}

		assert.Empty(t, msg.Channel)
		assert.Nil(t, msg.Payload)
	})

	t.Run("message with empty payload", func(t *testing.T) {
		msg := Message{
			Channel: "test",
			Payload: []byte{},
		}

		assert.Equal(t, "test", msg.Channel)
		assert.Empty(t, msg.Payload)
	})
}

func TestChannelConstants(t *testing.T) {
	assert.Equal(t, "fluxbase:broadcast", BroadcastChannel)
	assert.Equal(t, "fluxbase:presence", PresenceChannel)
}
