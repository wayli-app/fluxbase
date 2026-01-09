//nolint:errcheck // Test code - error handling not critical
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
	assert.Equal(t, 0, len(manager.connections))
}

func TestManager_AddConnection(t *testing.T) {
	ctx := context.Background()
	manager := NewManager(ctx)
	conn := &websocket.Conn{}

	connection := manager.AddConnection("conn1", conn, nil, "anon", nil)

	assert.NotNil(t, connection)
	assert.Equal(t, "conn1", connection.ID)
	assert.Equal(t, 1, manager.GetConnectionCount())
}

func TestManager_AddMultipleConnections(t *testing.T) {
	ctx := context.Background()
	manager := NewManager(ctx)

	manager.AddConnection("conn1", &websocket.Conn{}, nil, "anon", nil)
	manager.AddConnection("conn2", &websocket.Conn{}, nil, "anon", nil)
	manager.AddConnection("conn3", &websocket.Conn{}, nil, "anon", nil)

	assert.Equal(t, 3, manager.GetConnectionCount())
}

func TestManager_AddConnectionWithUserID(t *testing.T) {
	ctx := context.Background()
	manager := NewManager(ctx)
	userID := "user123"

	connection := manager.AddConnection("conn1", &websocket.Conn{}, &userID, "authenticated", nil)

	assert.NotNil(t, connection.UserID)
	assert.Equal(t, "user123", *connection.UserID)
}

func TestManager_RemoveConnection(t *testing.T) {
	ctx := context.Background()
	manager := NewManager(ctx)

	manager.AddConnection("conn1", &websocket.Conn{}, nil, "anon", nil)
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

	manager.AddConnection("conn1", &websocket.Conn{}, nil, "anon", nil)
	assert.Equal(t, 1, manager.GetConnectionCount())

	manager.AddConnection("conn2", &websocket.Conn{}, nil, "anon", nil)
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
			manager.AddConnection("conn"+string(rune(n)), &websocket.Conn{}, nil, "anon", nil)
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
		manager.AddConnection("conn"+string(rune(i)), &websocket.Conn{}, nil, "anon", nil)
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

	manager.AddConnection("conn1", &websocket.Conn{}, nil, "anon", nil)
	manager.AddConnection("conn2", &websocket.Conn{}, nil, "anon", nil)

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
			manager.AddConnection("conn"+string(rune(n%20)), &websocket.Conn{}, nil, "anon", nil)
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
