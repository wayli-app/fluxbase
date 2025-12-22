package pubsub

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLocalPubSub(t *testing.T) {
	ps := NewLocalPubSub()
	require.NotNil(t, ps)
	assert.NotNil(t, ps.subscribers)
	assert.Empty(t, ps.subscribers)

	err := ps.Close()
	require.NoError(t, err)
}

func TestLocalPubSub_PublishSubscribe(t *testing.T) {
	ps := NewLocalPubSub()
	defer ps.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Subscribe to a channel
	msgCh, err := ps.Subscribe(ctx, "test-channel")
	require.NoError(t, err)
	require.NotNil(t, msgCh)

	// Publish a message
	payload := []byte(`{"test": "data"}`)
	err = ps.Publish(ctx, "test-channel", payload)
	require.NoError(t, err)

	// Receive the message
	select {
	case msg := <-msgCh:
		assert.Equal(t, "test-channel", msg.Channel)
		assert.Equal(t, payload, msg.Payload)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for message")
	}
}

func TestLocalPubSub_MultipleSubscribers(t *testing.T) {
	ps := NewLocalPubSub()
	defer ps.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create multiple subscribers
	sub1, err := ps.Subscribe(ctx, "test-channel")
	require.NoError(t, err)

	sub2, err := ps.Subscribe(ctx, "test-channel")
	require.NoError(t, err)

	sub3, err := ps.Subscribe(ctx, "test-channel")
	require.NoError(t, err)

	// Publish a message
	payload := []byte("broadcast message")
	err = ps.Publish(ctx, "test-channel", payload)
	require.NoError(t, err)

	// All subscribers should receive the message
	for i, sub := range []<-chan Message{sub1, sub2, sub3} {
		select {
		case msg := <-sub:
			assert.Equal(t, payload, msg.Payload, "subscriber %d", i)
		case <-time.After(time.Second):
			t.Fatalf("subscriber %d timed out", i)
		}
	}
}

func TestLocalPubSub_DifferentChannels(t *testing.T) {
	ps := NewLocalPubSub()
	defer ps.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Subscribe to different channels
	sub1, err := ps.Subscribe(ctx, "channel-1")
	require.NoError(t, err)

	sub2, err := ps.Subscribe(ctx, "channel-2")
	require.NoError(t, err)

	// Publish to channel-1
	err = ps.Publish(ctx, "channel-1", []byte("message for channel 1"))
	require.NoError(t, err)

	// Only sub1 should receive the message
	select {
	case msg := <-sub1:
		assert.Equal(t, "channel-1", msg.Channel)
	case <-time.After(time.Second):
		t.Fatal("sub1 timed out")
	}

	// sub2 should not receive anything
	select {
	case msg := <-sub2:
		t.Fatalf("sub2 should not receive message, got: %v", msg)
	case <-time.After(100 * time.Millisecond):
		// Expected - no message for sub2
	}
}

func TestLocalPubSub_UnsubscribeOnContextCancel(t *testing.T) {
	ps := NewLocalPubSub()
	defer ps.Close()

	ctx, cancel := context.WithCancel(context.Background())

	// Subscribe
	msgCh, err := ps.Subscribe(ctx, "test-channel")
	require.NoError(t, err)

	// Verify subscription exists
	ps.mu.RLock()
	assert.Len(t, ps.subscribers["test-channel"], 1)
	ps.mu.RUnlock()

	// Cancel context
	cancel()

	// Wait for unsubscription to process
	time.Sleep(100 * time.Millisecond)

	// Verify subscription is removed
	ps.mu.RLock()
	assert.Len(t, ps.subscribers["test-channel"], 0)
	ps.mu.RUnlock()

	// Channel should be closed
	_, ok := <-msgCh
	assert.False(t, ok, "channel should be closed")
}

func TestLocalPubSub_Close(t *testing.T) {
	ps := NewLocalPubSub()

	ctx := context.Background()

	// Create multiple subscriptions
	sub1, err := ps.Subscribe(ctx, "channel-1")
	require.NoError(t, err)

	sub2, err := ps.Subscribe(ctx, "channel-2")
	require.NoError(t, err)

	// Close the pubsub
	err = ps.Close()
	require.NoError(t, err)

	// All channels should be closed
	_, ok := <-sub1
	assert.False(t, ok, "sub1 should be closed")

	_, ok = <-sub2
	assert.False(t, ok, "sub2 should be closed")

	// Subscribers map should be empty
	ps.mu.RLock()
	assert.Empty(t, ps.subscribers)
	ps.mu.RUnlock()
}

func TestLocalPubSub_PublishToNonExistentChannel(t *testing.T) {
	ps := NewLocalPubSub()
	defer ps.Close()

	ctx := context.Background()

	// Publishing to a channel with no subscribers should not error
	err := ps.Publish(ctx, "non-existent", []byte("message"))
	require.NoError(t, err)
}

func TestLocalPubSub_FullChannelBuffer(t *testing.T) {
	ps := NewLocalPubSub()
	defer ps.Close()

	ctx := context.Background()

	// Subscribe but don't read messages
	_, err := ps.Subscribe(ctx, "test-channel")
	require.NoError(t, err)

	// Publish more than buffer size (100) messages
	for i := 0; i < 150; i++ {
		err := ps.Publish(ctx, "test-channel", []byte("message"))
		require.NoError(t, err) // Should not error even when buffer is full
	}
}

func TestLocalPubSub_ConcurrentOperations(t *testing.T) {
	ps := NewLocalPubSub()
	defer ps.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	numSubscribers := 10
	numPublishers := 5
	messagesPerPublisher := 20

	// Track received messages
	received := make([]int, numSubscribers)
	var mu sync.Mutex

	// Start subscribers
	for i := 0; i < numSubscribers; i++ {
		subCtx, subCancel := context.WithCancel(ctx)
		defer subCancel()

		msgCh, err := ps.Subscribe(subCtx, "concurrent-test")
		require.NoError(t, err)

		wg.Add(1)
		go func(idx int, ch <-chan Message) {
			defer wg.Done()
			for {
				select {
				case _, ok := <-ch:
					if !ok {
						return
					}
					mu.Lock()
					received[idx]++
					mu.Unlock()
				case <-ctx.Done():
					return
				}
			}
		}(i, msgCh)
	}

	// Start publishers
	for i := 0; i < numPublishers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < messagesPerPublisher; j++ {
				err := ps.Publish(ctx, "concurrent-test", []byte("message"))
				assert.NoError(t, err)
			}
		}()
	}

	// Wait for publishers to finish, then wait a bit for messages to be delivered
	time.Sleep(500 * time.Millisecond)
	cancel()
	wg.Wait()

	// Verify messages were received (at least some, as channel buffer may drop some)
	mu.Lock()
	totalReceived := 0
	for _, count := range received {
		totalReceived += count
	}
	mu.Unlock()

	assert.Greater(t, totalReceived, 0, "at least some messages should be received")
}

func TestLocalPubSub_MessageContent(t *testing.T) {
	ps := NewLocalPubSub()
	defer ps.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	channel := "test-channel"
	msgCh, err := ps.Subscribe(ctx, channel)
	require.NoError(t, err)

	testCases := []struct {
		name    string
		payload []byte
	}{
		{"empty payload", []byte{}},
		{"simple string", []byte("hello world")},
		{"json object", []byte(`{"key": "value", "num": 123}`)},
		{"binary data", []byte{0x00, 0x01, 0x02, 0xFF}},
		{"large payload", make([]byte, 10000)},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := ps.Publish(ctx, channel, tc.payload)
			require.NoError(t, err)

			select {
			case msg := <-msgCh:
				assert.Equal(t, channel, msg.Channel)
				assert.Equal(t, tc.payload, msg.Payload)
			case <-time.After(time.Second):
				t.Fatal("timed out waiting for message")
			}
		})
	}
}

func TestLocalPubSub_SubscribeMultipleTimes(t *testing.T) {
	ps := NewLocalPubSub()
	defer ps.Close()

	ctx := context.Background()

	// Subscribe to the same channel multiple times
	channels := make([]<-chan Message, 3)
	for i := 0; i < 3; i++ {
		ch, err := ps.Subscribe(ctx, "shared-channel")
		require.NoError(t, err)
		channels[i] = ch
	}

	// Verify all subscriptions are independent
	ps.mu.RLock()
	assert.Len(t, ps.subscribers["shared-channel"], 3)
	ps.mu.RUnlock()
}

func TestLocalPubSub_BroadcastChannel(t *testing.T) {
	ps := NewLocalPubSub()
	defer ps.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Test using the defined BroadcastChannel constant
	msgCh, err := ps.Subscribe(ctx, BroadcastChannel)
	require.NoError(t, err)

	err = ps.Publish(ctx, BroadcastChannel, []byte("broadcast"))
	require.NoError(t, err)

	select {
	case msg := <-msgCh:
		assert.Equal(t, BroadcastChannel, msg.Channel)
	case <-time.After(time.Second):
		t.Fatal("timed out")
	}
}

func TestLocalPubSub_PresenceChannel(t *testing.T) {
	ps := NewLocalPubSub()
	defer ps.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Test using the defined PresenceChannel constant
	msgCh, err := ps.Subscribe(ctx, PresenceChannel)
	require.NoError(t, err)

	err = ps.Publish(ctx, PresenceChannel, []byte("presence update"))
	require.NoError(t, err)

	select {
	case msg := <-msgCh:
		assert.Equal(t, PresenceChannel, msg.Channel)
	case <-time.After(time.Second):
		t.Fatal("timed out")
	}
}
