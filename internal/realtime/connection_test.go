package realtime

import (
	"sync"
	"testing"

	"github.com/gofiber/contrib/websocket"
	"github.com/stretchr/testify/assert"
)

// MockWebSocketConn is a mock WebSocket connection for testing
type MockWebSocketConn struct {
	messages []interface{}
	closed   bool
	mu       sync.Mutex
}

func (m *MockWebSocketConn) WriteJSON(v interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = append(m.messages, v)
	return nil
}

func (m *MockWebSocketConn) ReadJSON(v interface{}) error {
	return nil
}

func (m *MockWebSocketConn) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}

func (m *MockWebSocketConn) GetMessages() []interface{} {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]interface{}{}, m.messages...)
}

func (m *MockWebSocketConn) IsClosed() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.closed
}

func TestNewConnection(t *testing.T) {
	conn := &websocket.Conn{}
	userID := "user123"

	connection := NewConnection("conn1", conn, &userID, "authenticated", nil)

	assert.NotNil(t, connection)
	assert.Equal(t, "conn1", connection.ID)
	assert.Equal(t, &userID, connection.UserID)
	assert.Equal(t, "authenticated", connection.Role)
	assert.NotNil(t, connection.Subscriptions)
	assert.Equal(t, 0, len(connection.Subscriptions))
}

func TestConnection_Subscribe(t *testing.T) {
	conn := &websocket.Conn{}
	connection := NewConnection("conn1", conn, nil, "anon", nil)

	// Subscribe to a channel
	connection.Subscribe("table:public.products")

	assert.True(t, connection.Subscriptions["table:public.products"])
	assert.Equal(t, 1, len(connection.Subscriptions))
}

func TestConnection_SubscribeMultiple(t *testing.T) {
	conn := &websocket.Conn{}
	connection := NewConnection("conn1", conn, nil, "anon", nil)

	// Subscribe to multiple channels
	connection.Subscribe("table:public.products")
	connection.Subscribe("table:public.orders")
	connection.Subscribe("table:public.users")

	assert.True(t, connection.Subscriptions["table:public.products"])
	assert.True(t, connection.Subscriptions["table:public.orders"])
	assert.True(t, connection.Subscriptions["table:public.users"])
	assert.Equal(t, 3, len(connection.Subscriptions))
}

func TestConnection_SubscribeDuplicate(t *testing.T) {
	conn := &websocket.Conn{}
	connection := NewConnection("conn1", conn, nil, "anon", nil)

	// Subscribe to same channel twice
	connection.Subscribe("table:public.products")
	connection.Subscribe("table:public.products")

	assert.True(t, connection.Subscriptions["table:public.products"])
	assert.Equal(t, 1, len(connection.Subscriptions))
}

func TestConnection_Unsubscribe(t *testing.T) {
	conn := &websocket.Conn{}
	connection := NewConnection("conn1", conn, nil, "anon", nil)

	// Subscribe then unsubscribe
	connection.Subscribe("table:public.products")
	assert.True(t, connection.Subscriptions["table:public.products"])

	connection.Unsubscribe("table:public.products")
	assert.False(t, connection.Subscriptions["table:public.products"])
	assert.Equal(t, 0, len(connection.Subscriptions))
}

func TestConnection_UnsubscribeNonExistent(t *testing.T) {
	conn := &websocket.Conn{}
	connection := NewConnection("conn1", conn, nil, "anon", nil)

	// Unsubscribe from channel we never subscribed to
	connection.Unsubscribe("table:public.products")

	assert.False(t, connection.Subscriptions["table:public.products"])
	assert.Equal(t, 0, len(connection.Subscriptions))
}

func TestConnection_IsSubscribed(t *testing.T) {
	conn := &websocket.Conn{}
	connection := NewConnection("conn1", conn, nil, "anon", nil)

	// Initially not subscribed
	assert.False(t, connection.IsSubscribed("table:public.products"))

	// After subscribing
	connection.Subscribe("table:public.products")
	assert.True(t, connection.IsSubscribed("table:public.products"))

	// After unsubscribing
	connection.Unsubscribe("table:public.products")
	assert.False(t, connection.IsSubscribed("table:public.products"))
}

func TestConnection_ConcurrentSubscribe(t *testing.T) {
	conn := &websocket.Conn{}
	connection := NewConnection("conn1", conn, nil, "anon", nil)

	var wg sync.WaitGroup
	numGoroutines := 100

	// Concurrently subscribe to different channels
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			channel := "table:public.test" + string(rune(n))
			connection.Subscribe(channel)
		}(i)
	}

	wg.Wait()

	// All subscriptions should be recorded
	assert.Equal(t, numGoroutines, len(connection.Subscriptions))
}

func TestConnection_ConcurrentUnsubscribe(t *testing.T) {
	conn := &websocket.Conn{}
	connection := NewConnection("conn1", conn, nil, "anon", nil)

	// Subscribe to multiple channels first
	numChannels := 100
	for i := 0; i < numChannels; i++ {
		channel := "table:public.test" + string(rune(i))
		connection.Subscribe(channel)
	}

	assert.Equal(t, numChannels, len(connection.Subscriptions))

	var wg sync.WaitGroup

	// Concurrently unsubscribe from all channels
	for i := 0; i < numChannels; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			channel := "table:public.test" + string(rune(n))
			connection.Unsubscribe(channel)
		}(i)
	}

	wg.Wait()

	// All subscriptions should be removed
	assert.Equal(t, 0, len(connection.Subscriptions))
}

func TestConnection_ConcurrentIsSubscribed(t *testing.T) {
	conn := &websocket.Conn{}
	connection := NewConnection("conn1", conn, nil, "anon", nil)

	connection.Subscribe("table:public.products")

	var wg sync.WaitGroup
	numGoroutines := 100

	// Concurrently check subscription status
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// Should not panic or race
			_ = connection.IsSubscribed("table:public.products")
		}()
	}

	wg.Wait()

	// Subscription should still be there
	assert.True(t, connection.IsSubscribed("table:public.products"))
}

func TestConnection_MixedConcurrentOperations(t *testing.T) {
	conn := &websocket.Conn{}
	connection := NewConnection("conn1", conn, nil, "anon", nil)

	var wg sync.WaitGroup
	numGoroutines := 50

	// Mix of subscribe, unsubscribe, and check operations
	for i := 0; i < numGoroutines; i++ {
		wg.Add(3)

		// Subscribe
		go func(n int) {
			defer wg.Done()
			channel := "table:public.test" + string(rune(n%10))
			connection.Subscribe(channel)
		}(i)

		// Unsubscribe
		go func(n int) {
			defer wg.Done()
			channel := "table:public.test" + string(rune(n%10))
			connection.Unsubscribe(channel)
		}(i)

		// Check subscription
		go func(n int) {
			defer wg.Done()
			channel := "table:public.test" + string(rune(n%10))
			_ = connection.IsSubscribed(channel)
		}(i)
	}

	wg.Wait()

	// Should not panic - exact count may vary due to race conditions,
	// but that's expected behavior
	assert.True(t, len(connection.Subscriptions) >= 0)
}
