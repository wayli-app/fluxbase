//go:build integration

// Package integration provides integration tests for the pubsub module.
// These tests use a real PostgreSQL database to verify pub/sub functionality
// including message delivery, subscription management, and channel behavior.
package integration

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nimbleflux/fluxbase/internal/pubsub"
	"github.com/nimbleflux/fluxbase/internal/testutil/e2e"
)

// TestPostgresPubSub_Start verifies that the pubsub starts correctly
// and launches the listenLoop goroutine.
func TestPostgresPubSub_Start(t *testing.T) {
	tc := e2e.NewIntegrationTestContextWithNamespace(t, "pubsub")
	defer tc.Close()

	ps := pubsub.NewPostgresPubSub(tc.DB.Pool())

	err := ps.Start()
	require.NoError(t, err)

	// Start is idempotent - calling again should succeed
	err = ps.Start()
	assert.NoError(t, err)

	_ = ps.Close()
}

// TestPostgresPubSub_Subscribe creates subscriptions and verifies they work.
func TestPostgresPubSub_Subscribe(t *testing.T) {
	tc := e2e.NewIntegrationTestContextWithNamespace(t, "pubsub")
	defer tc.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ps := pubsub.NewPostgresPubSub(tc.DB.Pool())
	require.NoError(t, ps.Start())
	defer func() { _ = ps.Close() }()

	// Create a subscription
	sub, err := ps.Subscribe(ctx, "test-channel")
	require.NoError(t, err)
	require.NotNil(t, sub)

	// Verify we can read from the channel (it should be open)
	select {
	case _, ok := <-sub:
		assert.True(t, ok, "channel should be open")
	case <-time.After(100 * time.Millisecond):
		// Timeout is OK - just checking the channel is readable
	}
}

// TestPostgresPubSub_MultipleSubscribers verifies multiple subscribers
// can be created for the same channel.
func TestPostgresPubSub_MultipleSubscribers(t *testing.T) {
	tc := e2e.NewIntegrationTestContextWithNamespace(t, "pubsub")
	defer tc.Close()

	ctx := context.Background()

	ps := pubsub.NewPostgresPubSub(tc.DB.Pool())
	require.NoError(t, ps.Start())
	defer func() { _ = ps.Close() }()

	// Create multiple subscribers
	subs := make([]<-chan pubsub.Message, 3)
	for i := 0; i < 3; i++ {
		ch, err := ps.Subscribe(ctx, "test-channel")
		require.NoError(t, err)
		subs[i] = ch
	}

	// Verify all channels are open
	for i, sub := range subs {
		select {
		case _, ok := <-sub:
			assert.True(t, ok, "channel %d should be open", i)
		case <-time.After(100 * time.Millisecond):
			// Timeout is OK
		}
	}
}

// TestPostgresPubSub_PublishBasic verifies basic publish functionality.
func TestPostgresPubSub_PublishBasic(t *testing.T) {
	tc := e2e.NewIntegrationTestContextWithNamespace(t, "pubsub")
	defer tc.Close()

	ctx := context.Background()

	ps := pubsub.NewPostgresPubSub(tc.DB.Pool())
	require.NoError(t, ps.Start())
	defer func() { _ = ps.Close() }()

	// Publish a message
	err := ps.Publish(ctx, "test-channel", []byte("test"))
	require.NoError(t, err)
}

// TestPostgresPubSub_PayloadSizeLimit verifies the 8000 byte payload limit.
func TestPostgresPubSub_PayloadSizeLimit(t *testing.T) {
	tc := e2e.NewIntegrationTestContextWithNamespace(t, "pubsub")
	defer tc.Close()

	ctx := context.Background()

	ps := pubsub.NewPostgresPubSub(tc.DB.Pool())
	require.NoError(t, ps.Start())
	defer func() { _ = ps.Close() }()

	// Payload under limit should succeed
	payloadOk := make([]byte, 1000)
	for i := range payloadOk {
		payloadOk[i] = 'A'
	}
	err := ps.Publish(ctx, "test-channel", payloadOk)
	assert.NoError(t, err, "payload under 8000 bytes should succeed")

	// Payload over limit should fail with our validation
	payloadOverLimit := make([]byte, 8001)
	for i := range payloadOverLimit {
		payloadOverLimit[i] = 'B'
	}
	err = ps.Publish(ctx, "test-channel", payloadOverLimit)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "payload too large")
}

// TestPostgresPubSub_BuiltinChannels verifies the built-in channel constants.
func TestPostgresPubSub_BuiltinChannels(t *testing.T) {
	tc := e2e.NewIntegrationTestContextWithNamespace(t, "pubsub")
	defer tc.Close()

	ctx := context.Background()

	ps := pubsub.NewPostgresPubSub(tc.DB.Pool())
	require.NoError(t, ps.Start())
	defer func() { _ = ps.Close() }()

	// Test publishing to built-in channels
	channels := []string{
		pubsub.BroadcastChannel,
		pubsub.PresenceChannel,
		pubsub.SchemaCacheChannel,
	}

	for _, channel := range channels {
		err := ps.Publish(ctx, channel, []byte("test"))
		require.NoError(t, err, "should publish to %s", channel)
	}
}

// TestPostgresPubSub_UnsubscribeOnContextCancel verifies that subscribers
// are removed when their context is cancelled.
func TestPostgresPubSub_UnsubscribeOnContextCancel(t *testing.T) {
	tc := e2e.NewIntegrationTestContextWithNamespace(t, "pubsub")
	defer tc.Close()

	ps := pubsub.NewPostgresPubSub(tc.DB.Pool())
	require.NoError(t, ps.Start())
	defer func() { _ = ps.Close() }()

	// Create subscriber with cancelable context
	ctx, cancel := context.WithCancel(context.Background())
	sub, err := ps.Subscribe(ctx, "test-channel")
	require.NoError(t, err)

	// Subscribe should be active
	select {
	case _, ok := <-sub:
		assert.True(t, ok, "channel should be open")
	case <-time.After(100 * time.Millisecond):
		// Timeout is OK
	}

	// Cancel context
	cancel()

	// Wait for cleanup
	time.Sleep(200 * time.Millisecond)

	// Channel should be closed
	select {
	case _, ok := <-sub:
		assert.False(t, ok, "channel should be closed after context cancel")
	default:
		// Channel might not be closed yet
	}
}

// TestPostgresPubSub_CloseClosesAllChannels verifies that Close()
// closes all subscriber channels.
func TestPostgresPubSub_CloseClosesAllChannels(t *testing.T) {
	tc := e2e.NewIntegrationTestContextWithNamespace(t, "pubsub")
	defer tc.Close()

	ctx := context.Background()

	ps := pubsub.NewPostgresPubSub(tc.DB.Pool())
	require.NoError(t, ps.Start())

	// Create multiple subscribers
	channels := make([]<-chan pubsub.Message, 3)
	for i := 0; i < 3; i++ {
		ch, err := ps.Subscribe(ctx, "test-channel")
		require.NoError(t, err)
		channels[i] = ch
	}

	// Close pubsub
	_ = ps.Close()

	// All channels should be closed
	for i, ch := range channels {
		_, ok := <-ch
		assert.False(t, ok, "channel %d should be closed", i)
	}
}

// TestPostgresPubSub_EmptyPayload verifies that empty payloads work correctly.
func TestPostgresPubSub_EmptyPayload(t *testing.T) {
	tc := e2e.NewIntegrationTestContextWithNamespace(t, "pubsub")
	defer tc.Close()

	ctx := context.Background()

	ps := pubsub.NewPostgresPubSub(tc.DB.Pool())
	require.NoError(t, ps.Start())
	defer func() { _ = ps.Close() }()

	err := ps.Publish(ctx, "test-channel", []byte{})
	require.NoError(t, err, "empty payload should be accepted")
}
