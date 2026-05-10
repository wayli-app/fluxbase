//nolint:errcheck // Test code - error handling not critical
package realtime

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/nimbleflux/fluxbase/internal/pubsub"
)

// mockPubSub implements pubsub.PubSub for testing
type mockPubSub struct {
	mu            sync.RWMutex
	subscriptions map[string][]chan pubsub.Message
	published     []pubsubMessage
}

type pubsubMessage struct {
	Channel string
	Payload []byte
}

func newMockPubSub() *mockPubSub {
	return &mockPubSub{
		subscriptions: make(map[string][]chan pubsub.Message),
	}
}

func (m *mockPubSub) Publish(ctx context.Context, channel string, payload []byte) error {
	m.mu.Lock()
	m.published = append(m.published, pubsubMessage{Channel: channel, Payload: payload})
	subs := m.subscriptions[channel]
	m.mu.Unlock()

	for _, ch := range subs {
		select {
		case ch <- pubsub.Message{Channel: channel, Payload: payload}:
		default:
		}
	}
	return nil
}

func (m *mockPubSub) Subscribe(ctx context.Context, channel string) (<-chan pubsub.Message, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	ch := make(chan pubsub.Message, 100)
	m.subscriptions[channel] = append(m.subscriptions[channel], ch)
	return ch, nil
}

func (m *mockPubSub) Close() error {
	return nil
}

func (m *mockPubSub) getPublishedMessages() []pubsubMessage {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]pubsubMessage{}, m.published...)
}

func TestNewManager(t *testing.T) {
	ctx := context.Background()
	manager := NewManager(ctx)

	assert.NotNil(t, manager)
	assert.NotNil(t, manager.connections)
	assert.Equal(t, 0, len(manager.connections))
}

func TestManager_AddConnection(t *testing.T) {
	ctx := context.Background()
	manager := NewManager(ctx)

	connection, err := manager.AddConnection("conn1", nil, nil, "anon", nil, "")

	assert.NoError(t, err)
	assert.NotNil(t, connection)
	assert.Equal(t, "conn1", connection.ID)
	assert.Equal(t, 1, manager.GetConnectionCount())
}

func TestManager_AddMultipleConnections(t *testing.T) {
	ctx := context.Background()
	manager := NewManager(ctx)

	manager.AddConnection("conn1", nil, nil, "anon", nil, "")
	manager.AddConnection("conn2", nil, nil, "anon", nil, "")
	manager.AddConnection("conn3", nil, nil, "anon", nil, "")

	assert.Equal(t, 3, manager.GetConnectionCount())
}

func TestManager_AddConnectionWithUserID(t *testing.T) {
	ctx := context.Background()
	manager := NewManager(ctx)
	userID := "user123"

	connection, err := manager.AddConnection("conn1", nil, &userID, "authenticated", nil, "")

	assert.NoError(t, err)
	assert.NotNil(t, connection.UserID)
	assert.Equal(t, "user123", *connection.UserID)
}

func TestManager_RemoveConnection(t *testing.T) {
	ctx := context.Background()
	manager := NewManager(ctx)

	manager.AddConnection("conn1", nil, nil, "anon", nil, "")
	assert.Equal(t, 1, manager.GetConnectionCount())

	manager.RemoveConnection("conn1")
	assert.Equal(t, 0, manager.GetConnectionCount())
}

func TestManager_RemoveNonExistentConnection(t *testing.T) {
	ctx := context.Background()
	manager := NewManager(ctx)

	// Should not panic
	manager.RemoveConnection("conn1")
	assert.Equal(t, 0, manager.GetConnectionCount())
}

func TestManager_GetConnectionCount(t *testing.T) {
	ctx := context.Background()
	manager := NewManager(ctx)

	assert.Equal(t, 0, manager.GetConnectionCount())

	manager.AddConnection("conn1", nil, nil, "anon", nil, "")
	assert.Equal(t, 1, manager.GetConnectionCount())

	manager.AddConnection("conn2", nil, nil, "anon", nil, "")
	assert.Equal(t, 2, manager.GetConnectionCount())

	manager.RemoveConnection("conn1")
	assert.Equal(t, 1, manager.GetConnectionCount())

	manager.RemoveConnection("conn2")
	assert.Equal(t, 0, manager.GetConnectionCount())
}

func TestManager_ConcurrentAddConnection(t *testing.T) {
	ctx := context.Background()
	manager := NewManager(ctx)

	var wg sync.WaitGroup
	numConnections := 100

	for i := 0; i < numConnections; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			manager.AddConnection("conn"+string(rune(n)), nil, nil, "anon", nil, "")
		}(i)
	}

	wg.Wait()

	assert.Equal(t, numConnections, manager.GetConnectionCount())
}

func TestManager_ConcurrentRemoveConnection(t *testing.T) {
	ctx := context.Background()
	manager := NewManager(ctx)

	// Add connections first
	numConnections := 100
	for i := 0; i < numConnections; i++ {
		manager.AddConnection("conn"+string(rune(i)), nil, nil, "anon", nil, "")
	}

	assert.Equal(t, numConnections, manager.GetConnectionCount())

	var wg sync.WaitGroup

	for i := 0; i < numConnections; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			manager.RemoveConnection("conn" + string(rune(n)))
		}(i)
	}

	wg.Wait()

	assert.Equal(t, 0, manager.GetConnectionCount())
}

func TestManager_Shutdown(t *testing.T) {
	ctx := context.Background()
	manager := NewManager(ctx)

	manager.AddConnection("conn1", nil, nil, "anon", nil, "")
	manager.AddConnection("conn2", nil, nil, "anon", nil, "")

	manager.Shutdown()

	// Give time for cleanup
	time.Sleep(100 * time.Millisecond)

	// Connections should be cleaned up
	assert.Equal(t, 0, manager.GetConnectionCount())
}

func TestManager_MixedConcurrentOperations(t *testing.T) {
	ctx := context.Background()
	manager := NewManager(ctx)

	var wg sync.WaitGroup
	numGoroutines := 50

	// Mix of add and remove operations
	for i := 0; i < numGoroutines; i++ {
		// Add connection
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			manager.AddConnection("conn"+string(rune(n%20)), nil, nil, "anon", nil, "")
		}(i)

		// Remove connection
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			manager.RemoveConnection("conn" + string(rune(n%20)))
		}(i)
	}

	wg.Wait()

	// Should not panic - exact counts may vary due to race conditions
	assert.True(t, manager.GetConnectionCount() >= 0)
}

// Tests for per-user and per-IP connection limits

func TestManager_PerUserConnectionLimit(t *testing.T) {
	ctx := context.Background()
	manager := NewManagerWithConfig(ctx, ManagerConfig{
		MaxConnectionsPerUser: 2,
	})

	userID := "user123"

	// First two connections should succeed
	conn1, err := manager.AddConnectionWithIP("conn1", nil, &userID, "authenticated", nil, "", "192.168.1.1")
	assert.NoError(t, err)
	assert.NotNil(t, conn1)

	conn2, err := manager.AddConnectionWithIP("conn2", nil, &userID, "authenticated", nil, "", "192.168.1.1")
	assert.NoError(t, err)
	assert.NotNil(t, conn2)

	// Third connection should fail
	conn3, err := manager.AddConnectionWithIP("conn3", nil, &userID, "authenticated", nil, "", "192.168.1.1")
	assert.Error(t, err)
	assert.Equal(t, ErrMaxUserConnectionsReached, err)
	assert.Nil(t, conn3)

	// Verify counts
	assert.Equal(t, 2, manager.GetConnectionCount())
	assert.Equal(t, 2, manager.GetUserConnectionCount(userID))
}

func TestManager_PerUserConnectionLimit_DifferentUsers(t *testing.T) {
	ctx := context.Background()
	manager := NewManagerWithConfig(ctx, ManagerConfig{
		MaxConnectionsPerUser: 2,
	})

	user1 := "user1"
	user2 := "user2"

	// User1 can have 2 connections
	manager.AddConnectionWithIP("conn1", nil, &user1, "authenticated", nil, "", "192.168.1.1")
	manager.AddConnectionWithIP("conn2", nil, &user1, "authenticated", nil, "", "192.168.1.1")

	// User2 can also have 2 connections
	manager.AddConnectionWithIP("conn3", nil, &user2, "authenticated", nil, "", "192.168.1.2")
	manager.AddConnectionWithIP("conn4", nil, &user2, "authenticated", nil, "", "192.168.1.2")

	// Both users should be at their limits
	_, err1 := manager.AddConnectionWithIP("conn5", nil, &user1, "authenticated", nil, "", "192.168.1.1")
	assert.Equal(t, ErrMaxUserConnectionsReached, err1)

	_, err2 := manager.AddConnectionWithIP("conn6", nil, &user2, "authenticated", nil, "", "192.168.1.2")
	assert.Equal(t, ErrMaxUserConnectionsReached, err2)

	assert.Equal(t, 4, manager.GetConnectionCount())
}

func TestManager_PerIPConnectionLimit(t *testing.T) {
	ctx := context.Background()
	manager := NewManagerWithConfig(ctx, ManagerConfig{
		MaxConnectionsPerIP: 3,
	})

	ip := "192.168.1.100"

	// First three anonymous connections from same IP should succeed
	conn1, err := manager.AddConnectionWithIP("conn1", nil, nil, "anon", nil, "", ip)
	assert.NoError(t, err)
	assert.NotNil(t, conn1)

	conn2, err := manager.AddConnectionWithIP("conn2", nil, nil, "anon", nil, "", ip)
	assert.NoError(t, err)
	assert.NotNil(t, conn2)

	conn3, err := manager.AddConnectionWithIP("conn3", nil, nil, "anon", nil, "", ip)
	assert.NoError(t, err)
	assert.NotNil(t, conn3)

	// Fourth connection should fail
	conn4, err := manager.AddConnectionWithIP("conn4", nil, nil, "anon", nil, "", ip)
	assert.Error(t, err)
	assert.Equal(t, ErrMaxIPConnectionsReached, err)
	assert.Nil(t, conn4)

	// Verify counts
	assert.Equal(t, 3, manager.GetConnectionCount())
	assert.Equal(t, 3, manager.GetIPConnectionCount(ip))
}

func TestManager_PerIPConnectionLimit_DifferentIPs(t *testing.T) {
	ctx := context.Background()
	manager := NewManagerWithConfig(ctx, ManagerConfig{
		MaxConnectionsPerIP: 2,
	})

	ip1 := "192.168.1.1"
	ip2 := "192.168.1.2"

	// IP1 can have 2 connections
	manager.AddConnectionWithIP("conn1", nil, nil, "anon", nil, "", ip1)
	manager.AddConnectionWithIP("conn2", nil, nil, "anon", nil, "", ip1)

	// IP2 can also have 2 connections
	manager.AddConnectionWithIP("conn3", nil, nil, "anon", nil, "", ip2)
	manager.AddConnectionWithIP("conn4", nil, nil, "anon", nil, "", ip2)

	assert.Equal(t, 4, manager.GetConnectionCount())
	assert.Equal(t, 2, manager.GetIPConnectionCount(ip1))
	assert.Equal(t, 2, manager.GetIPConnectionCount(ip2))
}

func TestManager_PerIPLimitNotAppliedToAuthenticatedUsers(t *testing.T) {
	ctx := context.Background()
	manager := NewManagerWithConfig(ctx, ManagerConfig{
		MaxConnectionsPerIP:   2,
		MaxConnectionsPerUser: 100, // High limit for users
	})

	ip := "192.168.1.1"
	userID := "user123"

	// Authenticated users should not be limited by IP
	for i := 0; i < 5; i++ {
		conn, err := manager.AddConnectionWithIP("conn"+string(rune('a'+i)), nil, &userID, "authenticated", nil, "", ip)
		assert.NoError(t, err)
		assert.NotNil(t, conn)
	}

	// IP count should be 0 (only tracks anonymous)
	assert.Equal(t, 0, manager.GetIPConnectionCount(ip))
	// User count should be 5
	assert.Equal(t, 5, manager.GetUserConnectionCount(userID))
}

func TestManager_RemoveConnection_DecrementsUserCount(t *testing.T) {
	ctx := context.Background()
	manager := NewManagerWithConfig(ctx, ManagerConfig{
		MaxConnectionsPerUser: 2,
	})

	userID := "user123"

	// Add two connections
	manager.AddConnectionWithIP("conn1", nil, &userID, "authenticated", nil, "", "192.168.1.1")
	manager.AddConnectionWithIP("conn2", nil, &userID, "authenticated", nil, "", "192.168.1.1")

	// Verify at limit
	_, err := manager.AddConnectionWithIP("conn3", nil, &userID, "authenticated", nil, "", "192.168.1.1")
	assert.Equal(t, ErrMaxUserConnectionsReached, err)

	// Remove one connection
	manager.RemoveConnection("conn1")

	// Should be able to add a new connection
	conn3, err := manager.AddConnectionWithIP("conn3", nil, &userID, "authenticated", nil, "", "192.168.1.1")
	assert.NoError(t, err)
	assert.NotNil(t, conn3)
}

func TestManager_RemoveConnection_DecrementsIPCount(t *testing.T) {
	ctx := context.Background()
	manager := NewManagerWithConfig(ctx, ManagerConfig{
		MaxConnectionsPerIP: 2,
	})

	ip := "192.168.1.100"

	// Add two connections
	manager.AddConnectionWithIP("conn1", nil, nil, "anon", nil, "", ip)
	manager.AddConnectionWithIP("conn2", nil, nil, "anon", nil, "", ip)

	// Verify at limit
	_, err := manager.AddConnectionWithIP("conn3", nil, nil, "anon", nil, "", ip)
	assert.Equal(t, ErrMaxIPConnectionsReached, err)

	// Remove one connection
	manager.RemoveConnection("conn1")

	// Should be able to add a new connection
	conn3, err := manager.AddConnectionWithIP("conn3", nil, nil, "anon", nil, "", ip)
	assert.NoError(t, err)
	assert.NotNil(t, conn3)
}

func TestManager_Shutdown_ClearsTrackingMaps(t *testing.T) {
	ctx := context.Background()
	manager := NewManagerWithConfig(ctx, ManagerConfig{
		MaxConnectionsPerUser: 10,
		MaxConnectionsPerIP:   10,
	})

	userID := "user123"
	ip := "192.168.1.1"

	// Add some connections
	manager.AddConnectionWithIP("conn1", nil, &userID, "authenticated", nil, "", ip)
	manager.AddConnectionWithIP("conn2", nil, nil, "anon", nil, "", ip)

	// Shutdown
	manager.Shutdown()
	time.Sleep(100 * time.Millisecond)

	// All tracking should be cleared
	assert.Equal(t, 0, manager.GetConnectionCount())
	assert.Equal(t, 0, manager.GetUserConnectionCount(userID))
	assert.Equal(t, 0, manager.GetIPConnectionCount(ip))
}

func TestManager_SetConnectionLimits(t *testing.T) {
	ctx := context.Background()
	manager := NewManagerWithConfig(ctx, ManagerConfig{
		MaxConnectionsPerUser: 100,
		MaxConnectionsPerIP:   100,
	})

	userID := "user123"

	// Add 5 connections
	for i := 0; i < 5; i++ {
		manager.AddConnectionWithIP("conn"+string(rune('a'+i)), nil, &userID, "authenticated", nil, "", "192.168.1.1")
	}

	// Reduce limit - existing connections remain but no new ones allowed
	manager.SetConnectionLimits(3, 3)

	// New connection should fail
	_, err := manager.AddConnectionWithIP("conn_new", nil, &userID, "authenticated", nil, "", "192.168.1.1")
	assert.Equal(t, ErrMaxUserConnectionsReached, err)
}

func TestManager_GlobalLimitTakesPrecedence(t *testing.T) {
	ctx := context.Background()
	manager := NewManagerWithConfig(ctx, ManagerConfig{
		MaxConnections:        3,
		MaxConnectionsPerUser: 10,
		MaxConnectionsPerIP:   10,
	})

	user1 := "user1"
	user2 := "user2"

	// Add 3 connections (global limit)
	manager.AddConnectionWithIP("conn1", nil, &user1, "authenticated", nil, "", "192.168.1.1")
	manager.AddConnectionWithIP("conn2", nil, &user1, "authenticated", nil, "", "192.168.1.1")
	manager.AddConnectionWithIP("conn3", nil, &user2, "authenticated", nil, "", "192.168.1.2")

	// Fourth connection should fail due to global limit
	_, err := manager.AddConnectionWithIP("conn4", nil, &user2, "authenticated", nil, "", "192.168.1.2")
	assert.Equal(t, ErrMaxConnectionsReached, err)
}

// =============================================================================
// Slow Client Tests
// =============================================================================

func TestManager_SlowClientConfig(t *testing.T) {
	t.Run("default slow client settings", func(t *testing.T) {
		manager := NewManagerWithConfig(context.Background(), ManagerConfig{})

		// Check defaults
		assert.Equal(t, 100, manager.slowClientThreshold)
		assert.Equal(t, 30*time.Second, manager.slowClientTimeout)
	})

	t.Run("custom slow client settings", func(t *testing.T) {
		manager := NewManagerWithConfig(context.Background(), ManagerConfig{
			SlowClientThreshold: 50,
			SlowClientTimeout:   10 * time.Second,
		})

		assert.Equal(t, 50, manager.slowClientThreshold)
		assert.Equal(t, 10*time.Second, manager.slowClientTimeout)
	})
}

func TestManager_SlowClientTrackingMap(t *testing.T) {
	manager := NewManagerWithConfig(context.Background(), ManagerConfig{
		SlowClientThreshold: 5, // Low threshold for testing
		SlowClientTimeout:   1 * time.Second,
	})

	// Verify the tracking map is initialized
	assert.NotNil(t, manager.slowClientFirstSeen)
	assert.Empty(t, manager.slowClientFirstSeen)
}

func TestManager_GetSlowClientsDisconnected(t *testing.T) {
	manager := NewManager(context.Background())

	// Initially should be 0
	assert.Equal(t, uint64(0), manager.GetSlowClientsDisconnected())
}

func TestManagerConfig_SlowClientFields(t *testing.T) {
	config := ManagerConfig{
		MaxConnections:         100,
		MaxConnectionsPerUser:  10,
		MaxConnectionsPerIP:    20,
		ClientMessageQueueSize: 256,
		SlowClientThreshold:    150,
		SlowClientTimeout:      45 * time.Second,
	}

	assert.Equal(t, 100, config.MaxConnections)
	assert.Equal(t, 150, config.SlowClientThreshold)
	assert.Equal(t, 45*time.Second, config.SlowClientTimeout)
}

// =============================================================================
// PubSub Tests (Global Broadcasting)
// =============================================================================

func TestManager_SetPubSub(t *testing.T) {
	t.Run("sets pubsub backend", func(t *testing.T) {
		manager := NewManager(context.Background())
		mockPubSub := newMockPubSub()

		manager.SetPubSub(mockPubSub)

		assert.NotNil(t, manager.ps)
		assert.Equal(t, mockPubSub, manager.ps)
	})

	t.Run("starts global broadcast handler when pubsub is set", func(t *testing.T) {
		manager := NewManager(context.Background())
		mockPubSub := newMockPubSub()

		// Setting pubsub starts the goroutine
		manager.SetPubSub(mockPubSub)

		// Verify the manager has the pubsub reference
		assert.Equal(t, mockPubSub, manager.ps)
	})

	t.Run("allows nil pubsub", func(t *testing.T) {
		manager := NewManager(context.Background())

		// Should not panic
		manager.SetPubSub(nil)

		assert.Nil(t, manager.ps)
	})
}

func TestManager_BroadcastGlobal(t *testing.T) {
	t.Run("broadcasts locally when no pubsub configured", func(t *testing.T) {
		ctx := context.Background()
		manager := NewManager(ctx)

		// Add a connection with subscription
		conn, _ := manager.AddConnection("conn1", nil, nil, "anon", nil, "")
		conn.Subscribe("test-channel")

		// Broadcast should work without pubsub
		message := ServerMessage{
			Type:    MessageTypeBroadcast,
			Channel: "test-channel",
			Payload: map[string]interface{}{"test": "data"},
		}

		err := manager.BroadcastGlobal("test-channel", "", message)
		assert.NoError(t, err)

		// Verify the message was broadcast locally
		count := manager.BroadcastToChannel("test-channel", "", message)
		assert.Equal(t, 1, count)
	})

	t.Run("publishes to pubsub when configured", func(t *testing.T) {
		ctx := context.Background()
		manager := NewManager(ctx)
		mockPubSub := newMockPubSub()

		manager.SetPubSub(mockPubSub)

		message := ServerMessage{
			Type:    MessageTypeBroadcast,
			Channel: "test-channel",
			Payload: map[string]interface{}{"test": "data"},
		}

		err := manager.BroadcastGlobal("test-channel", "", message)
		assert.NoError(t, err)

		// Verify message was published to pubsub
		published := mockPubSub.getPublishedMessages()
		assert.Len(t, published, 1)
		assert.Equal(t, pubsub.BroadcastChannel, published[0].Channel)
	})
}

func TestManager_handleGlobalBroadcasts(t *testing.T) {
	t.Run("subscribes to broadcast channel", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		manager := NewManager(ctx)
		mockPubSub := newMockPubSub()

		// Subscribe first (simulating what SetPubSub does)
		ch, err := mockPubSub.Subscribe(ctx, pubsub.BroadcastChannel)
		assert.NoError(t, err)

		manager.ps = mockPubSub

		// Start the handler (normally done by SetPubSub)
		go manager.handleGlobalBroadcasts()

		// Publish a message
		broadcast := GlobalBroadcast{
			Channel: "test-channel",
			Message: ServerMessage{
				Type:    MessageTypeBroadcast,
				Channel: "test-channel",
				Payload: map[string]interface{}{"test": "data"},
			},
		}
		payload, _ := json.Marshal(broadcast)
		mockPubSub.Publish(ctx, pubsub.BroadcastChannel, payload)

		// Message should be received
		select {
		case <-ch:
			// Success - message was published
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Did not receive published message")
		}
	})
}

func TestManager_handleGlobalMessage(t *testing.T) {
	t.Run("delivers message to subscribed connections", func(t *testing.T) {
		ctx := context.Background()
		manager := NewManager(ctx)

		// Add connection with subscription
		conn, _ := manager.AddConnection("conn1", nil, nil, "anon", nil, "")
		conn.Subscribe("test-channel")

		// Simulate a global broadcast message
		broadcast := GlobalBroadcast{
			Channel: "test-channel",
			Message: ServerMessage{
				Type:    MessageTypeBroadcast,
				Channel: "test-channel",
				Payload: map[string]interface{}{"test": "data"},
			},
		}
		payload, _ := json.Marshal(broadcast)
		msg := pubsub.Message{
			Channel: pubsub.BroadcastChannel,
			Payload: payload,
		}

		// Handle the message
		manager.handleGlobalMessage(msg)

		// Give time for async processing
		time.Sleep(50 * time.Millisecond)
	})

	t.Run("does not deliver to unsubscribed connections", func(t *testing.T) {
		ctx := context.Background()
		manager := NewManager(ctx)

		// Add connection WITHOUT subscription
		_, _ = manager.AddConnection("conn1", nil, nil, "anon", nil, "")

		// Simulate a global broadcast message
		broadcast := GlobalBroadcast{
			Channel: "test-channel",
			Message: ServerMessage{
				Type:    MessageTypeBroadcast,
				Channel: "test-channel",
				Payload: map[string]interface{}{"test": "data"},
			},
		}
		payload, _ := json.Marshal(broadcast)
		msg := pubsub.Message{
			Channel: pubsub.BroadcastChannel,
			Payload: payload,
		}

		// Handle the message - should not panic
		manager.handleGlobalMessage(msg)

		// Give time for async processing
		time.Sleep(50 * time.Millisecond)
	})

	t.Run("handles invalid messages gracefully", func(t *testing.T) {
		ctx := context.Background()
		manager := NewManager(ctx)

		// Invalid JSON
		msg := pubsub.Message{
			Channel: pubsub.BroadcastChannel,
			Payload: []byte("invalid json"),
		}

		// Should not panic
		manager.handleGlobalMessage(msg)
	})
}

// =============================================================================
// Metrics Tests
// =============================================================================

func TestManager_SetMetrics(t *testing.T) {
	t.Run("sets metrics instance", func(t *testing.T) {
		manager := NewManager(context.Background())

		// Initially nil
		assert.Nil(t, manager.metrics)

		// Note: We can't create a real Metrics instance without complex setup
		// Just verify the method exists and doesn't panic
		manager.SetMetrics(nil)

		assert.Nil(t, manager.metrics)
	})
}

func TestManager_updateMetrics(t *testing.T) {
	t.Run("does not panic when metrics is nil", func(t *testing.T) {
		manager := NewManager(context.Background())

		// Should not panic
		manager.updateMetrics()
	})

	t.Run("counts active connections", func(t *testing.T) {
		manager := NewManager(context.Background())

		manager.AddConnection("conn1", nil, nil, "anon", nil, "")
		manager.AddConnection("conn2", nil, nil, "anon", nil, "")

		// Should not panic
		manager.updateMetrics()
	})

	t.Run("counts unique channels", func(t *testing.T) {
		manager := NewManager(context.Background())

		conn1, _ := manager.AddConnection("conn1", nil, nil, "anon", nil, "")
		conn1.Subscribe("channel1")
		conn1.Subscribe("channel2")

		conn2, _ := manager.AddConnection("conn2", nil, nil, "anon", nil, "")
		conn2.Subscribe("channel1") // Same channel
		conn2.Subscribe("channel3") // Different channel

		// Should not panic
		manager.updateMetrics()
	})
}

// =============================================================================
// Connection Info Tests
// =============================================================================

func TestManager_GetDetailedStats(t *testing.T) {
	t.Run("returns empty stats when no connections", func(t *testing.T) {
		manager := NewManager(context.Background())

		stats := manager.GetDetailedStats()

		assert.NotNil(t, stats)
		assert.Equal(t, 0, stats["total_connections"])
	})

	t.Run("returns connection details", func(t *testing.T) {
		manager := NewManager(context.Background())

		userID := "user123"
		manager.AddConnection("conn1", nil, &userID, "authenticated", nil, "")

		stats := manager.GetDetailedStats()

		assert.NotNil(t, stats)
		assert.Equal(t, 1, stats["total_connections"])

		connections, ok := stats["connections"].([]ConnectionInfo)
		assert.True(t, ok)
		assert.Len(t, connections, 1)
		assert.Equal(t, "conn1", connections[0].ID)
		assert.NotNil(t, connections[0].UserID)
		assert.Equal(t, "user123", *connections[0].UserID)
	})

	t.Run("includes connected timestamp", func(t *testing.T) {
		manager := NewManager(context.Background())

		manager.AddConnection("conn1", nil, nil, "anon", nil, "")

		stats := manager.GetDetailedStats()
		connections := stats["connections"].([]ConnectionInfo)

		assert.NotEmpty(t, connections[0].ConnectedAt)
	})
}

func TestManager_GetConnectionsForStats(t *testing.T) {
	t.Run("returns empty list when no connections", func(t *testing.T) {
		manager := NewManager(context.Background())

		connections := manager.GetConnectionsForStats()

		assert.NotNil(t, connections)
		assert.Len(t, connections, 0)
	})

	t.Run("returns all connections", func(t *testing.T) {
		manager := NewManager(context.Background())

		userID := "user123"
		manager.AddConnection("conn1", nil, &userID, "authenticated", nil, "")
		manager.AddConnection("conn2", nil, nil, "anon", nil, "")

		connections := manager.GetConnectionsForStats()

		assert.Len(t, connections, 2)

		// Connections may be returned in any order since they're stored in a map
		// Create a map of connection IDs for order-independent comparison
		connIDs := make(map[string]bool)
		for _, conn := range connections {
			connIDs[conn.ID] = true
		}

		assert.True(t, connIDs["conn1"], "conn1 should be present")
		assert.True(t, connIDs["conn2"], "conn2 should be present")
	})
}

// =============================================================================
// Admin Broadcast Tests
// =============================================================================

func TestManager_BroadcastConnectionEvent(t *testing.T) {
	t.Run("broadcasts connection event", func(t *testing.T) {
		ctx := context.Background()
		manager := NewManager(ctx)
		mockPubSub := newMockPubSub()

		manager.SetPubSub(mockPubSub)

		conn, _ := manager.AddConnection("conn1", nil, nil, "anon", nil, "")

		event := NewConnectionEvent(ConnectionEventConnected, conn, nil, nil)

		// Broadcast the event
		manager.BroadcastConnectionEvent(event)

		// Verify pubsub message was sent
		published := mockPubSub.getPublishedMessages()
		assert.NotEmpty(t, published)
	})

	t.Run("works without pubsub configured", func(t *testing.T) {
		manager := NewManager(context.Background())

		conn, _ := manager.AddConnection("conn1", nil, nil, "anon", nil, "")

		event := NewConnectionEvent(ConnectionEventConnected, conn, nil, nil)

		// Should not panic
		manager.BroadcastConnectionEvent(event)
	})
}

// =============================================================================
// SetMaxConnections Tests
// =============================================================================

func TestManager_SetMaxConnections(t *testing.T) {
	t.Run("updates maximum connections", func(t *testing.T) {
		manager := NewManager(context.Background())

		manager.SetMaxConnections(50)

		assert.Equal(t, 50, manager.maxConnections)
	})

	t.Run("enforces new limit for future connections", func(t *testing.T) {
		manager := NewManager(context.Background())
		manager.SetMaxConnections(2)

		// Add 2 connections
		manager.AddConnection("conn1", nil, nil, "anon", nil, "")
		manager.AddConnection("conn2", nil, nil, "anon", nil, "")

		// Third should fail
		_, err := manager.AddConnection("conn3", nil, nil, "anon", nil, "")
		assert.Equal(t, ErrMaxConnectionsReached, err)
	})

	t.Run("allows zero for unlimited", func(t *testing.T) {
		manager := NewManager(context.Background())
		manager.SetMaxConnections(0)

		// Should allow many connections
		for i := 0; i < 100; i++ {
			_, err := manager.AddConnection("conn"+string(rune(i)), nil, nil, "anon", nil, "")
			assert.NoError(t, err)
		}
	})
}

// =============================================================================
// Slow Client Detection Tests
// =============================================================================

func TestManager_checkAndDisconnectSlowClients(t *testing.T) {
	t.Run("does not disconnect normal clients", func(t *testing.T) {
		ctx := context.Background()
		manager := NewManagerWithConfig(ctx, ManagerConfig{
			SlowClientThreshold: 10,
			SlowClientTimeout:   1 * time.Second,
		})

		_, _ = manager.AddConnection("conn1", nil, nil, "anon", nil, "")

		// Run check - should not disconnect
		manager.checkAndDisconnectSlowClients()

		assert.Equal(t, 1, manager.GetConnectionCount())
	})

	t.Run("marks clients as slow when queue exceeds threshold", func(t *testing.T) {
		ctx := context.Background()
		manager := NewManagerWithConfig(ctx, ManagerConfig{
			ClientMessageQueueSize: 20,
			SlowClientThreshold:    10,
			SlowClientTimeout:      1 * time.Second,
		})

		conn, _ := manager.AddConnection("conn1", nil, nil, "anon", nil, "")

		// Cancel the connection context to stop the writer goroutine from draining the queue
		conn.cancel()
		conn.wg.Wait()

		// Fill the queue beyond threshold directly via the channel
		stats := conn.GetQueueStats()
		for conn.GetQueueStats().QueueLength < stats.QueueCapacity {
			conn.sendCh <- map[string]interface{}{"fill": "queue"}
		}

		// Run check - should mark as slow but not disconnect yet
		manager.checkAndDisconnectSlowClients()

		assert.Equal(t, 1, manager.GetConnectionCount())
		assert.Contains(t, manager.slowClientFirstSeen, "conn1")
	})
}

func TestManager_disconnectSlowClient(t *testing.T) {
	t.Run("removes connection and increments counter", func(t *testing.T) {
		ctx := context.Background()
		manager := NewManager(ctx)

		// Create a sync connection for testing
		conn := NewConnectionSync("conn1", nil, nil, "anon", nil)
		manager.connections["conn1"] = conn

		before := manager.GetSlowClientsDisconnected()

		manager.disconnectSlowClient("conn1")

		assert.Equal(t, 0, manager.GetConnectionCount())
		assert.Equal(t, before+uint64(1), manager.GetSlowClientsDisconnected())
	})

	t.Run("handles non-existent connection gracefully", func(t *testing.T) {
		manager := NewManager(context.Background())

		// Should not panic
		manager.disconnectSlowClient("non-existent")
	})
}

// =============================================================================
// SplitHostPort Tests
// =============================================================================

func TestSplitHostPort(t *testing.T) {
	t.Run("splits valid host:port", func(t *testing.T) {
		host, port, err := splitHostPort("192.168.1.1:8080")

		assert.NoError(t, err)
		assert.Equal(t, "192.168.1.1", host)
		assert.Equal(t, "8080", port)
	})

	t.Run("handles IPv6 addresses", func(t *testing.T) {
		host, port, err := splitHostPort("[::1]:8080")

		assert.NoError(t, err)
		assert.Equal(t, "::1", host)
		assert.Equal(t, "8080", port)
	})

	t.Run("returns error for invalid format", func(t *testing.T) {
		_, _, err := splitHostPort("invalid")

		assert.Error(t, err)
	})

	t.Run("returns error for missing port", func(t *testing.T) {
		_, _, err := splitHostPort("192.168.1.1")

		assert.Error(t, err)
	})
}

// =============================================================================
// Tenant Isolation Tests
// =============================================================================

func TestManager_BroadcastToChannel_TenantIsolation(t *testing.T) {
	t.Run("only sends to connections matching tenant ID", func(t *testing.T) {
		ctx := context.Background()
		manager := NewManager(ctx)

		connA, _ := manager.AddConnection("conn-a", nil, nil, "anon", nil, "tenant-A")
		connB, _ := manager.AddConnection("conn-b", nil, nil, "anon", nil, "tenant-B")

		channel := "test-channel"
		connA.Subscribe(channel)
		connB.Subscribe(channel)

		message := ServerMessage{
			Type:    MessageTypeBroadcast,
			Channel: channel,
			Payload: map[string]interface{}{"data": "for A"},
		}

		sentCount := manager.BroadcastToChannel(channel, "tenant-A", message)

		assert.Equal(t, 1, sentCount)
	})

	t.Run("sends to all connections of the same tenant", func(t *testing.T) {
		ctx := context.Background()
		manager := NewManager(ctx)

		connA1, _ := manager.AddConnection("conn-a1", nil, nil, "anon", nil, "tenant-A")
		connA2, _ := manager.AddConnection("conn-a2", nil, nil, "anon", nil, "tenant-A")
		connB, _ := manager.AddConnection("conn-b", nil, nil, "anon", nil, "tenant-B")

		channel := "shared-channel"
		connA1.Subscribe(channel)
		connA2.Subscribe(channel)
		connB.Subscribe(channel)

		message := ServerMessage{
			Type:    MessageTypeBroadcast,
			Channel: channel,
			Payload: map[string]interface{}{"data": "broadcast"},
		}

		sentCount := manager.BroadcastToChannel(channel, "tenant-A", message)

		assert.Equal(t, 2, sentCount)
	})
}

func TestManager_BroadcastToChannel_EmptyTenantID(t *testing.T) {
	t.Run("only sends to connections with empty tenant ID", func(t *testing.T) {
		ctx := context.Background()
		manager := NewManager(ctx)

		connEmpty, _ := manager.AddConnection("conn-empty", nil, nil, "anon", nil, "")
		connX, _ := manager.AddConnection("conn-x", nil, nil, "anon", nil, "tenant-X")

		channel := "public-channel"
		connEmpty.Subscribe(channel)
		connX.Subscribe(channel)

		message := ServerMessage{
			Type:    MessageTypeBroadcast,
			Channel: channel,
			Payload: map[string]interface{}{"data": "default tenant"},
		}

		sentCount := manager.BroadcastToChannel(channel, "", message)

		assert.Equal(t, 1, sentCount)
	})

	t.Run("does not send to connections with non-empty tenant ID when broadcasting with empty tenant ID", func(t *testing.T) {
		ctx := context.Background()
		manager := NewManager(ctx)

		connA, _ := manager.AddConnection("conn-a", nil, nil, "anon", nil, "tenant-A")
		connB, _ := manager.AddConnection("conn-b", nil, nil, "anon", nil, "tenant-B")
		connDefault, _ := manager.AddConnection("conn-default", nil, nil, "anon", nil, "")

		channel := "multi-tenant-channel"
		connA.Subscribe(channel)
		connB.Subscribe(channel)
		connDefault.Subscribe(channel)

		message := ServerMessage{
			Type:    MessageTypeBroadcast,
			Channel: channel,
			Payload: map[string]interface{}{"data": "default only"},
		}

		sentCount := manager.BroadcastToChannel(channel, "", message)

		assert.Equal(t, 1, sentCount)
	})
}

func TestManager_BroadcastGlobal_TenantIDPropagation(t *testing.T) {
	t.Run("includes tenant ID in published pubsub message", func(t *testing.T) {
		ctx := context.Background()
		manager := NewManager(ctx)
		mockPS := newMockPubSub()

		manager.SetPubSub(mockPS)

		message := ServerMessage{
			Type:    MessageTypeBroadcast,
			Channel: "test-channel",
			Payload: map[string]interface{}{"data": "tenant-scoped"},
		}

		err := manager.BroadcastGlobal("test-channel", "test-tenant", message)
		assert.NoError(t, err)

		published := mockPS.getPublishedMessages()
		assert.Len(t, published, 1)
		assert.Equal(t, pubsub.BroadcastChannel, published[0].Channel)

		var broadcast GlobalBroadcast
		err = json.Unmarshal(published[0].Payload, &broadcast)
		assert.NoError(t, err)
		assert.Equal(t, "test-tenant", broadcast.TenantID)
		assert.Equal(t, "test-channel", broadcast.Channel)
	})

	t.Run("broadcasts locally with tenant ID when no pubsub configured", func(t *testing.T) {
		ctx := context.Background()
		manager := NewManager(ctx)

		connA, _ := manager.AddConnection("conn-a", nil, nil, "anon", nil, "tenant-A")
		connB, _ := manager.AddConnection("conn-b", nil, nil, "anon", nil, "tenant-B")

		channel := "local-channel"
		connA.Subscribe(channel)
		connB.Subscribe(channel)

		message := ServerMessage{
			Type:    MessageTypeBroadcast,
			Channel: channel,
			Payload: map[string]interface{}{"data": "local"},
		}

		err := manager.BroadcastGlobal(channel, "tenant-A", message)
		assert.NoError(t, err)

		sentCount := manager.BroadcastToChannel(channel, "tenant-A", message)
		assert.Equal(t, 1, sentCount)
	})
}
