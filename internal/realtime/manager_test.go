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

	connection, err := manager.AddConnection("conn1", conn, nil, "anon", nil)

	assert.NoError(t, err)
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

	connection, err := manager.AddConnection("conn1", &websocket.Conn{}, &userID, "authenticated", nil)

	assert.NoError(t, err)
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

// Tests for per-user and per-IP connection limits

func TestManager_PerUserConnectionLimit(t *testing.T) {
	ctx := context.Background()
	manager := NewManagerWithConfig(ctx, ManagerConfig{
		MaxConnectionsPerUser: 2,
	})

	userID := "user123"

	// First two connections should succeed
	conn1, err := manager.AddConnectionWithIP("conn1", &websocket.Conn{}, &userID, "authenticated", nil, "192.168.1.1")
	assert.NoError(t, err)
	assert.NotNil(t, conn1)

	conn2, err := manager.AddConnectionWithIP("conn2", &websocket.Conn{}, &userID, "authenticated", nil, "192.168.1.1")
	assert.NoError(t, err)
	assert.NotNil(t, conn2)

	// Third connection should fail
	conn3, err := manager.AddConnectionWithIP("conn3", &websocket.Conn{}, &userID, "authenticated", nil, "192.168.1.1")
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
	manager.AddConnectionWithIP("conn1", &websocket.Conn{}, &user1, "authenticated", nil, "192.168.1.1")
	manager.AddConnectionWithIP("conn2", &websocket.Conn{}, &user1, "authenticated", nil, "192.168.1.1")

	// User2 can also have 2 connections
	manager.AddConnectionWithIP("conn3", &websocket.Conn{}, &user2, "authenticated", nil, "192.168.1.2")
	manager.AddConnectionWithIP("conn4", &websocket.Conn{}, &user2, "authenticated", nil, "192.168.1.2")

	// Both users should be at their limits
	_, err1 := manager.AddConnectionWithIP("conn5", &websocket.Conn{}, &user1, "authenticated", nil, "192.168.1.1")
	assert.Equal(t, ErrMaxUserConnectionsReached, err1)

	_, err2 := manager.AddConnectionWithIP("conn6", &websocket.Conn{}, &user2, "authenticated", nil, "192.168.1.2")
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
	conn1, err := manager.AddConnectionWithIP("conn1", &websocket.Conn{}, nil, "anon", nil, ip)
	assert.NoError(t, err)
	assert.NotNil(t, conn1)

	conn2, err := manager.AddConnectionWithIP("conn2", &websocket.Conn{}, nil, "anon", nil, ip)
	assert.NoError(t, err)
	assert.NotNil(t, conn2)

	conn3, err := manager.AddConnectionWithIP("conn3", &websocket.Conn{}, nil, "anon", nil, ip)
	assert.NoError(t, err)
	assert.NotNil(t, conn3)

	// Fourth connection should fail
	conn4, err := manager.AddConnectionWithIP("conn4", &websocket.Conn{}, nil, "anon", nil, ip)
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
	manager.AddConnectionWithIP("conn1", &websocket.Conn{}, nil, "anon", nil, ip1)
	manager.AddConnectionWithIP("conn2", &websocket.Conn{}, nil, "anon", nil, ip1)

	// IP2 can also have 2 connections
	manager.AddConnectionWithIP("conn3", &websocket.Conn{}, nil, "anon", nil, ip2)
	manager.AddConnectionWithIP("conn4", &websocket.Conn{}, nil, "anon", nil, ip2)

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
		conn, err := manager.AddConnectionWithIP("conn"+string(rune('a'+i)), &websocket.Conn{}, &userID, "authenticated", nil, ip)
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
	manager.AddConnectionWithIP("conn1", &websocket.Conn{}, &userID, "authenticated", nil, "192.168.1.1")
	manager.AddConnectionWithIP("conn2", &websocket.Conn{}, &userID, "authenticated", nil, "192.168.1.1")

	// Verify at limit
	_, err := manager.AddConnectionWithIP("conn3", &websocket.Conn{}, &userID, "authenticated", nil, "192.168.1.1")
	assert.Equal(t, ErrMaxUserConnectionsReached, err)

	// Remove one connection
	manager.RemoveConnection("conn1")

	// Should be able to add a new connection
	conn3, err := manager.AddConnectionWithIP("conn3", &websocket.Conn{}, &userID, "authenticated", nil, "192.168.1.1")
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
	manager.AddConnectionWithIP("conn1", &websocket.Conn{}, nil, "anon", nil, ip)
	manager.AddConnectionWithIP("conn2", &websocket.Conn{}, nil, "anon", nil, ip)

	// Verify at limit
	_, err := manager.AddConnectionWithIP("conn3", &websocket.Conn{}, nil, "anon", nil, ip)
	assert.Equal(t, ErrMaxIPConnectionsReached, err)

	// Remove one connection
	manager.RemoveConnection("conn1")

	// Should be able to add a new connection
	conn3, err := manager.AddConnectionWithIP("conn3", &websocket.Conn{}, nil, "anon", nil, ip)
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
	manager.AddConnectionWithIP("conn1", &websocket.Conn{}, &userID, "authenticated", nil, ip)
	manager.AddConnectionWithIP("conn2", &websocket.Conn{}, nil, "anon", nil, ip)

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
		manager.AddConnectionWithIP("conn"+string(rune('a'+i)), &websocket.Conn{}, &userID, "authenticated", nil, "192.168.1.1")
	}

	// Reduce limit - existing connections remain but no new ones allowed
	manager.SetConnectionLimits(3, 3)

	// New connection should fail
	_, err := manager.AddConnectionWithIP("conn_new", &websocket.Conn{}, &userID, "authenticated", nil, "192.168.1.1")
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
	manager.AddConnectionWithIP("conn1", &websocket.Conn{}, &user1, "authenticated", nil, "192.168.1.1")
	manager.AddConnectionWithIP("conn2", &websocket.Conn{}, &user1, "authenticated", nil, "192.168.1.1")
	manager.AddConnectionWithIP("conn3", &websocket.Conn{}, &user2, "authenticated", nil, "192.168.1.2")

	// Fourth connection should fail due to global limit
	_, err := manager.AddConnectionWithIP("conn4", &websocket.Conn{}, &user2, "authenticated", nil, "192.168.1.2")
	assert.Equal(t, ErrMaxConnectionsReached, err)
}
