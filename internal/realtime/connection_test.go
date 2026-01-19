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
	var conn *websocket.Conn // nil connection for testing
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
	var conn *websocket.Conn // nil connection for testing
	connection := NewConnection("conn1", conn, nil, "anon", nil)

	// Subscribe to a channel
	connection.Subscribe("table:public.products")

	assert.True(t, connection.Subscriptions["table:public.products"])
	assert.Equal(t, 1, len(connection.Subscriptions))
}

func TestConnection_SubscribeMultiple(t *testing.T) {
	var conn *websocket.Conn // nil connection for testing
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
	var conn *websocket.Conn // nil connection for testing
	connection := NewConnection("conn1", conn, nil, "anon", nil)

	// Subscribe to same channel twice
	connection.Subscribe("table:public.products")
	connection.Subscribe("table:public.products")

	assert.True(t, connection.Subscriptions["table:public.products"])
	assert.Equal(t, 1, len(connection.Subscriptions))
}

func TestConnection_Unsubscribe(t *testing.T) {
	var conn *websocket.Conn // nil connection for testing
	connection := NewConnection("conn1", conn, nil, "anon", nil)

	// Subscribe then unsubscribe
	connection.Subscribe("table:public.products")
	assert.True(t, connection.Subscriptions["table:public.products"])

	connection.Unsubscribe("table:public.products")
	assert.False(t, connection.Subscriptions["table:public.products"])
	assert.Equal(t, 0, len(connection.Subscriptions))
}

func TestConnection_UnsubscribeNonExistent(t *testing.T) {
	var conn *websocket.Conn // nil connection for testing
	connection := NewConnection("conn1", conn, nil, "anon", nil)

	// Unsubscribe from channel we never subscribed to
	connection.Unsubscribe("table:public.products")

	assert.False(t, connection.Subscriptions["table:public.products"])
	assert.Equal(t, 0, len(connection.Subscriptions))
}

func TestConnection_IsSubscribed(t *testing.T) {
	var conn *websocket.Conn // nil connection for testing
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
	var conn *websocket.Conn // nil connection for testing
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
	var conn *websocket.Conn // nil connection for testing
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
	var conn *websocket.Conn // nil connection for testing
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
	var conn *websocket.Conn // nil connection for testing
	connection := NewConnection("conn1", conn, nil, "anon", nil)
	defer connection.Close()

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

// =============================================================================
// Async Message Queue Tests
// =============================================================================

func TestNewConnectionWithQueueSize(t *testing.T) {
	var conn *websocket.Conn // nil connection for testing
	connection := NewConnectionWithQueueSize("conn1", conn, nil, "anon", nil, 128)
	defer connection.Close()

	assert.NotNil(t, connection)
	assert.Equal(t, 128, cap(connection.sendCh))
}

func TestNewConnectionWithQueueSize_DefaultOnZero(t *testing.T) {
	var conn *websocket.Conn // nil connection for testing
	connection := NewConnectionWithQueueSize("conn1", conn, nil, "anon", nil, 0)
	defer connection.Close()

	assert.Equal(t, DefaultMessageQueueSize, cap(connection.sendCh))
}

func TestNewConnectionWithQueueSize_DefaultOnNegative(t *testing.T) {
	var conn *websocket.Conn // nil connection for testing
	connection := NewConnectionWithQueueSize("conn1", conn, nil, "anon", nil, -10)
	defer connection.Close()

	assert.Equal(t, DefaultMessageQueueSize, cap(connection.sendCh))
}

func TestNewConnectionSync(t *testing.T) {
	var conn *websocket.Conn // nil connection for testing
	connection := NewConnectionSync("conn1", conn, nil, "anon", nil)

	assert.NotNil(t, connection)
	assert.True(t, connection.useSync)
	assert.Nil(t, connection.sendCh) // No queue for sync connections
}

func TestConnection_SendMessage_ToClosedConnection(t *testing.T) {
	var conn *websocket.Conn // nil connection for testing
	connection := NewConnection("conn1", conn, nil, "anon", nil)

	// Close the connection first
	connection.Close()

	// Sending should return error
	err := connection.SendMessage(ServerMessage{Type: "test"})
	assert.Equal(t, ErrConnectionClosed, err)
}

func TestConnection_GetQueueStats(t *testing.T) {
	var conn *websocket.Conn // nil connection for testing
	connection := NewConnectionWithQueueSize("conn1", conn, nil, "anon", nil, 100)
	defer connection.Close()

	stats := connection.GetQueueStats()

	assert.Equal(t, 0, stats.QueueLength)
	assert.Equal(t, 100, stats.QueueCapacity)
	assert.Equal(t, int32(0), stats.QueueHighWater)
	assert.Equal(t, uint64(0), stats.MessagesSent)
	assert.Equal(t, uint64(0), stats.MessagesDropped)
	assert.Equal(t, int32(0), stats.SlowClientCount)
}

func TestConnection_Close_MultipleTimes(t *testing.T) {
	connection := NewConnection("conn1", nil, nil, "anon", nil)

	// First close should succeed
	err := connection.Close()
	assert.NoError(t, err)

	// Second close should return nil without error
	err = connection.Close()
	assert.NoError(t, err)
}

func TestConnection_IsSlowClient_Initial(t *testing.T) {
	var conn *websocket.Conn // nil connection for testing
	connection := NewConnection("conn1", conn, nil, "anon", nil)
	defer connection.Close()

	assert.False(t, connection.IsSlowClient())
}

func TestConnection_IsSlowClient_AfterWarnings(t *testing.T) {
	var conn *websocket.Conn // nil connection for testing
	connection := NewConnection("conn1", conn, nil, "anon", nil)
	defer connection.Close()

	// Manually increment slow client count
	for i := 0; i < MaxSlowClientWarnings; i++ {
		connection.slowClientCount.Add(1)
	}

	assert.True(t, connection.IsSlowClient())
}

func TestConnectionQueueStats_Struct(t *testing.T) {
	stats := ConnectionQueueStats{
		QueueLength:     50,
		QueueCapacity:   256,
		QueueHighWater:  128,
		MessagesSent:    1000,
		MessagesDropped: 5,
		SlowClientCount: 2,
	}

	assert.Equal(t, 50, stats.QueueLength)
	assert.Equal(t, 256, stats.QueueCapacity)
	assert.Equal(t, int32(128), stats.QueueHighWater)
	assert.Equal(t, uint64(1000), stats.MessagesSent)
	assert.Equal(t, uint64(5), stats.MessagesDropped)
	assert.Equal(t, int32(2), stats.SlowClientCount)
}

func TestConnection_SendMessage_WithSlowClientMarked(t *testing.T) {
	var conn *websocket.Conn // nil connection for testing
	connection := NewConnection("conn1", conn, nil, "anon", nil)
	defer connection.Close()

	// Mark as slow client
	for i := 0; i < MaxSlowClientWarnings; i++ {
		connection.slowClientCount.Add(1)
	}

	// Sending should return ErrSlowClient immediately
	err := connection.SendMessage(ServerMessage{Type: "test"})
	assert.Equal(t, ErrSlowClient, err)
}

// Benchmarks

func BenchmarkConnection_Subscribe(b *testing.B) {
	var conn *websocket.Conn // nil connection for testing
	connection := NewConnectionSync("conn1", conn, nil, "anon", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		connection.Subscribe("table:public.test")
		connection.Unsubscribe("table:public.test")
	}
}

func BenchmarkConnection_IsSubscribed(b *testing.B) {
	var conn *websocket.Conn // nil connection for testing
	connection := NewConnectionSync("conn1", conn, nil, "anon", nil)
	connection.Subscribe("table:public.products")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = connection.IsSubscribed("table:public.products")
	}
}

func BenchmarkConnection_GetQueueStats(b *testing.B) {
	var conn *websocket.Conn // nil connection for testing
	connection := NewConnectionWithQueueSize("conn1", conn, nil, "anon", nil, 256)
	defer connection.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = connection.GetQueueStats()
	}
}
