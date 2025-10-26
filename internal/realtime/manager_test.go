package realtime

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/contrib/websocket"
	"github.com/stretchr/testify/assert"
)

func TestNewManager(t *testing.T) {
	ctx := context.Background()
	manager := NewManager(ctx)

	assert.NotNil(t, manager)
	assert.NotNil(t, manager.connections)
	assert.NotNil(t, manager.channels)
	assert.Equal(t, 0, len(manager.connections))
	assert.Equal(t, 0, len(manager.channels))
}

func TestManager_AddConnection(t *testing.T) {
	ctx := context.Background()
	manager := NewManager(ctx)
	conn := &websocket.Conn{}

	connection := manager.AddConnection("conn1", conn, nil)

	assert.NotNil(t, connection)
	assert.Equal(t, "conn1", connection.ID)
	assert.Equal(t, 1, manager.GetConnectionCount())
}

func TestManager_AddMultipleConnections(t *testing.T) {
	ctx := context.Background()
	manager := NewManager(ctx)

	manager.AddConnection("conn1", &websocket.Conn{}, nil)
	manager.AddConnection("conn2", &websocket.Conn{}, nil)
	manager.AddConnection("conn3", &websocket.Conn{}, nil)

	assert.Equal(t, 3, manager.GetConnectionCount())
}

func TestManager_AddConnectionWithUserID(t *testing.T) {
	ctx := context.Background()
	manager := NewManager(ctx)
	userID := "user123"

	connection := manager.AddConnection("conn1", &websocket.Conn{}, &userID)

	assert.NotNil(t, connection.UserID)
	assert.Equal(t, "user123", *connection.UserID)
}

func TestManager_RemoveConnection(t *testing.T) {
	ctx := context.Background()
	manager := NewManager(ctx)

	manager.AddConnection("conn1", &websocket.Conn{}, nil)
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

func TestManager_RemoveConnectionCleansUpSubscriptions(t *testing.T) {
	ctx := context.Background()
	manager := NewManager(ctx)

	manager.AddConnection("conn1", &websocket.Conn{}, nil)
	manager.Subscribe("conn1", "table:public.products")

	assert.Equal(t, 1, manager.GetChannelCount())

	manager.RemoveConnection("conn1")

	// Channel should be removed if no subscribers
	assert.Equal(t, 0, manager.GetChannelCount())
}

func TestManager_Subscribe(t *testing.T) {
	ctx := context.Background()
	manager := NewManager(ctx)

	manager.AddConnection("conn1", &websocket.Conn{}, nil)
	err := manager.Subscribe("conn1", "table:public.products")

	assert.NoError(t, err)
	assert.Equal(t, 1, manager.GetChannelCount())
}

func TestManager_SubscribeMultipleChannels(t *testing.T) {
	ctx := context.Background()
	manager := NewManager(ctx)

	manager.AddConnection("conn1", &websocket.Conn{}, nil)
	manager.Subscribe("conn1", "table:public.products")
	manager.Subscribe("conn1", "table:public.orders")
	manager.Subscribe("conn1", "table:public.users")

	assert.Equal(t, 3, manager.GetChannelCount())
}

func TestManager_SubscribeMultipleConnectionsToSameChannel(t *testing.T) {
	ctx := context.Background()
	manager := NewManager(ctx)

	manager.AddConnection("conn1", &websocket.Conn{}, nil)
	manager.AddConnection("conn2", &websocket.Conn{}, nil)
	manager.AddConnection("conn3", &websocket.Conn{}, nil)

	manager.Subscribe("conn1", "table:public.products")
	manager.Subscribe("conn2", "table:public.products")
	manager.Subscribe("conn3", "table:public.products")

	assert.Equal(t, 1, manager.GetChannelCount())
	assert.Equal(t, 3, manager.GetConnectionCount())
}

func TestManager_SubscribeNonExistentConnection(t *testing.T) {
	ctx := context.Background()
	manager := NewManager(ctx)

	err := manager.Subscribe("conn1", "table:public.products")

	assert.Error(t, err)
	assert.Equal(t, "connection not found", err.Error())
}

func TestManager_Unsubscribe(t *testing.T) {
	ctx := context.Background()
	manager := NewManager(ctx)

	manager.AddConnection("conn1", &websocket.Conn{}, nil)
	manager.Subscribe("conn1", "table:public.products")

	err := manager.Unsubscribe("conn1", "table:public.products")

	assert.NoError(t, err)
	assert.Equal(t, 0, manager.GetChannelCount())
}

func TestManager_UnsubscribeFromNonExistentChannel(t *testing.T) {
	ctx := context.Background()
	manager := NewManager(ctx)

	manager.AddConnection("conn1", &websocket.Conn{}, nil)

	// Unsubscribing from a channel we never subscribed to should be fine
	err := manager.Unsubscribe("conn1", "table:public.products")

	assert.NoError(t, err) // Should not error - connection exists, just not subscribed to this channel
}

func TestManager_UnsubscribeNonExistentConnection(t *testing.T) {
	ctx := context.Background()
	manager := NewManager(ctx)

	err := manager.Unsubscribe("conn1", "table:public.products")

	assert.Error(t, err)
	assert.Equal(t, "connection not found", err.Error())
}

func TestManager_UnsubscribeKeepsChannelIfOtherSubscribers(t *testing.T) {
	ctx := context.Background()
	manager := NewManager(ctx)

	manager.AddConnection("conn1", &websocket.Conn{}, nil)
	manager.AddConnection("conn2", &websocket.Conn{}, nil)

	manager.Subscribe("conn1", "table:public.products")
	manager.Subscribe("conn2", "table:public.products")

	assert.Equal(t, 1, manager.GetChannelCount())

	manager.Unsubscribe("conn1", "table:public.products")

	// Channel should still exist because conn2 is still subscribed
	assert.Equal(t, 1, manager.GetChannelCount())
}

func TestManager_GetConnectionCount(t *testing.T) {
	ctx := context.Background()
	manager := NewManager(ctx)

	assert.Equal(t, 0, manager.GetConnectionCount())

	manager.AddConnection("conn1", &websocket.Conn{}, nil)
	assert.Equal(t, 1, manager.GetConnectionCount())

	manager.AddConnection("conn2", &websocket.Conn{}, nil)
	assert.Equal(t, 2, manager.GetConnectionCount())

	manager.RemoveConnection("conn1")
	assert.Equal(t, 1, manager.GetConnectionCount())

	manager.RemoveConnection("conn2")
	assert.Equal(t, 0, manager.GetConnectionCount())
}

func TestManager_GetChannelCount(t *testing.T) {
	ctx := context.Background()
	manager := NewManager(ctx)

	assert.Equal(t, 0, manager.GetChannelCount())

	manager.AddConnection("conn1", &websocket.Conn{}, nil)
	manager.Subscribe("conn1", "table:public.products")
	assert.Equal(t, 1, manager.GetChannelCount())

	manager.Subscribe("conn1", "table:public.orders")
	assert.Equal(t, 2, manager.GetChannelCount())

	manager.Unsubscribe("conn1", "table:public.products")
	assert.Equal(t, 1, manager.GetChannelCount())

	manager.Unsubscribe("conn1", "table:public.orders")
	assert.Equal(t, 0, manager.GetChannelCount())
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
			manager.AddConnection("conn"+string(rune(n)), &websocket.Conn{}, nil)
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
		manager.AddConnection("conn"+string(rune(i)), &websocket.Conn{}, nil)
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

func TestManager_ConcurrentSubscribe(t *testing.T) {
	ctx := context.Background()
	manager := NewManager(ctx)

	// Add connections first
	numConnections := 50
	for i := 0; i < numConnections; i++ {
		manager.AddConnection("conn"+string(rune(i)), &websocket.Conn{}, nil)
	}

	var wg sync.WaitGroup

	// Each connection subscribes to multiple channels concurrently
	for i := 0; i < numConnections; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			manager.Subscribe("conn"+string(rune(n)), "table:public.products")
			manager.Subscribe("conn"+string(rune(n)), "table:public.orders")
		}(i)
	}

	wg.Wait()

	// Should have 2 channels
	assert.Equal(t, 2, manager.GetChannelCount())
}

func TestManager_ConcurrentUnsubscribe(t *testing.T) {
	ctx := context.Background()
	manager := NewManager(ctx)

	// Setup: add connections and subscriptions
	numConnections := 50
	for i := 0; i < numConnections; i++ {
		manager.AddConnection("conn"+string(rune(i)), &websocket.Conn{}, nil)
		manager.Subscribe("conn"+string(rune(i)), "table:public.products")
		manager.Subscribe("conn"+string(rune(i)), "table:public.orders")
	}

	var wg sync.WaitGroup

	// Concurrent unsubscribe
	for i := 0; i < numConnections; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			manager.Unsubscribe("conn"+string(rune(n)), "table:public.products")
			manager.Unsubscribe("conn"+string(rune(n)), "table:public.orders")
		}(i)
	}

	wg.Wait()

	// All channels should be removed
	assert.Equal(t, 0, manager.GetChannelCount())
}

func TestManager_Shutdown(t *testing.T) {
	ctx := context.Background()
	manager := NewManager(ctx)

	manager.AddConnection("conn1", &websocket.Conn{}, nil)
	manager.AddConnection("conn2", &websocket.Conn{}, nil)
	manager.Subscribe("conn1", "table:public.products")

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

	// Mix of add, remove, subscribe, unsubscribe operations
	for i := 0; i < numGoroutines; i++ {
		// Add connection
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			manager.AddConnection("conn"+string(rune(n%20)), &websocket.Conn{}, nil)
		}(i)

		// Subscribe
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			manager.Subscribe("conn"+string(rune(n%20)), "table:public.products")
		}(i)

		// Unsubscribe
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			manager.Unsubscribe("conn"+string(rune(n%20)), "table:public.products")
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
	assert.True(t, manager.GetChannelCount() >= 0)
}
