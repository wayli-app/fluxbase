//go:build integration
// +build integration

package integration

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nimbleflux/fluxbase/internal/pubsub"
	"github.com/nimbleflux/fluxbase/internal/realtime"
	test "github.com/nimbleflux/fluxbase/test"
)

// =============================================================================
// Mock Subscription DB for Testing
// =============================================================================

// mockSubscriptionDB is a mock implementation of SubscriptionDB for testing
type mockSubscriptionDB struct {
	mu               sync.RWMutex
	EnabledTables    map[string]bool
	RLSResults       map[string]bool
	OwnershipResults map[uuid.UUID]struct {
		IsOwner bool
		Exists  bool
	}
}

// newMockSubscriptionDB creates a new mock subscription database
func newMockSubscriptionDB() *mockSubscriptionDB {
	return &mockSubscriptionDB{
		EnabledTables: make(map[string]bool),
		RLSResults:    make(map[string]bool),
		OwnershipResults: make(map[uuid.UUID]struct {
			IsOwner bool
			Exists  bool
		}),
	}
}

// IsTableEnabled checks if a table is enabled for realtime
func (m *mockSubscriptionDB) IsTableEnabled(ctx context.Context, schema, table string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := schema + "." + table
	enabled, exists := m.EnabledTables[key]
	if !exists {
		return false, nil
	}
	return enabled, nil
}

// IsTableRealtimeEnabled checks if a table is enabled for realtime (alias for IsTableEnabled)
func (m *mockSubscriptionDB) IsTableRealtimeEnabled(ctx context.Context, schema, table string) (bool, error) {
	return m.IsTableEnabled(ctx, schema, table)
}

// CheckRLSAccess always returns true for testing (bypasses RLS)
func (m *mockSubscriptionDB) CheckRLSAccess(ctx context.Context, schema, table, role string, claims map[string]interface{}, recordID interface{}) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := schema + "." + table + "." + fmt.Sprintf("%v", recordID)
	if result, exists := m.RLSResults[key]; exists {
		return result, nil
	}
	// Default: allow access
	return true, nil
}

// SetTableEnabled marks a table as enabled for realtime
func (m *mockSubscriptionDB) SetTableEnabled(schema, table string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := schema + "." + table
	m.EnabledTables[key] = true
}

// CheckRPCOwnership is a stub for testing
func (m *mockSubscriptionDB) CheckRPCOwnership(ctx context.Context, execID, userID uuid.UUID) (bool, bool, error) {
	return true, true, nil
}

// CheckJobOwnership is a stub for testing
func (m *mockSubscriptionDB) CheckJobOwnership(ctx context.Context, execID, userID uuid.UUID) (bool, bool, error) {
	return true, true, nil
}

// CheckFunctionOwnership is a stub for testing
func (m *mockSubscriptionDB) CheckFunctionOwnership(ctx context.Context, execID, userID uuid.UUID) (bool, bool, error) {
	return true, true, nil
}

// =============================================================================
// Test Setup Helpers
// =============================================================================

// setupRealtimeTest creates a test environment for realtime integration tests
func setupRealtimeTest(t *testing.T) (*test.TestContext, *realtime.Manager, *realtime.SubscriptionManager, *mockSubscriptionDB) {
	tc := test.NewTestContext(t)

	// Create pubsub for testing
	ps := pubsub.NewLocalPubSub()

	// Create realtime manager
	manager := realtime.NewManagerWithConfig(context.Background(), realtime.ManagerConfig{
		MaxConnections:         100,
		MaxConnectionsPerUser:  10,
		ClientMessageQueueSize: 256,
	})

	// Create mock subscription DB for testing (bypasses actual RLS checks)
	mockDB := newMockSubscriptionDB()
	// Enable common test tables
	mockDB.SetTableEnabled("public", "products")

	// Create subscription manager with mock DB
	subManager := realtime.NewSubscriptionManager(mockDB)

	// Set pubsub for cross-instance broadcasting
	manager.SetPubSub(ps)

	return tc, manager, subManager, mockDB
}

// cleanupRealtimeTest cleans up test resources
func cleanupRealtimeTest(t *testing.T, tc *test.TestContext, manager *realtime.Manager) {
	manager.Shutdown()
	// Note: Don't close tc as it's shared across tests in the package
}

// =============================================================================
// WebSocket Connection Tests
// =============================================================================

func TestRealtimeWebSocketProtocol_Integration_Connection(t *testing.T) {
	tc, manager, _, _ := setupRealtimeTest(t)
	defer cleanupRealtimeTest(t, tc, manager)

	t.Run("WebSocket connection establishment", func(t *testing.T) {
		connID := uuid.New().String()
		userID := uuid.New().String()

		// Create a connection (using nil WebSocket conn for manager testing)
		conn, err := manager.AddConnectionWithIP(
			connID,
			nil, // WebSocket conn - nil for testing manager logic
			&userID,
			"authenticated",
			nil,
			"127.0.0.1",
		)

		require.NoError(t, err)
		assert.NotNil(t, conn)
		assert.Equal(t, connID, conn.ID)
		assert.Equal(t, userID, *conn.UserID)

		// Verify connection is tracked
		assert.Equal(t, 1, manager.GetConnectionCount())

		// Cleanup
		manager.RemoveConnection(connID)
		assert.Equal(t, 0, manager.GetConnectionCount())
	})

	t.Run("Anonymous WebSocket connection", func(t *testing.T) {
		connID := uuid.New().String()

		conn, err := manager.AddConnectionWithIP(
			connID,
			nil,
			nil,
			"anon",
			nil,
			"127.0.0.1",
		)

		require.NoError(t, err)
		assert.NotNil(t, conn)
		assert.Nil(t, conn.UserID)
		assert.Equal(t, "anon", conn.Role)

		manager.RemoveConnection(connID)
	})

	t.Run("Connection limit enforcement", func(t *testing.T) {
		// Create manager with small limit
		limitedManager := realtime.NewManagerWithConfig(context.Background(), realtime.ManagerConfig{
			MaxConnections: 2,
		})
		defer limitedManager.Shutdown()

		conn1 := uuid.New().String()
		conn2 := uuid.New().String()
		conn3 := uuid.New().String()

		// First two connections should succeed
		_, err1 := limitedManager.AddConnectionWithIP(conn1, nil, nil, "anon", nil, "127.0.0.1")
		assert.NoError(t, err1)

		_, err2 := limitedManager.AddConnectionWithIP(conn2, nil, nil, "anon", nil, "127.0.0.1")
		assert.NoError(t, err2)

		// Third connection should fail
		_, err3 := limitedManager.AddConnectionWithIP(conn3, nil, nil, "anon", nil, "127.0.0.1")
		assert.Error(t, err3)
		assert.Equal(t, realtime.ErrMaxConnectionsReached, err3)
	})
}

// =============================================================================
// Subscription Tests
// =============================================================================

func TestRealtimeWebSocketProtocol_Integration_Subscription(t *testing.T) {
	tc, manager, subManager, _ := setupRealtimeTest(t)
	defer cleanupRealtimeTest(t, tc, manager)

	// Enable realtime for products table (this table exists in e2e tests)
	tc.ExecuteSQL(`
		INSERT INTO realtime.schema_registry (schema_name, table_name, realtime_enabled)
		VALUES ('public', 'products', true)
		ON CONFLICT (schema_name, table_name) DO UPDATE
		SET realtime_enabled = true
	`)

	t.Run("Subscribe to table changes", func(t *testing.T) {
		connID := uuid.New().String()
		userID := uuid.New().String()

		// Add connection
		_, err := manager.AddConnectionWithIP(
			connID,
			nil,
			&userID,
			"authenticated",
			nil,
			"127.0.0.1",
		)
		require.NoError(t, err)

		// Subscribe to products table
		subID := uuid.New().String()
		sub, err := subManager.CreateSubscription(
			subID,
			connID,
			userID,
			"authenticated",
			nil,
			"public",
			"products",
			"*",
			"",
		)

		require.NoError(t, err)
		assert.NotNil(t, sub)
		assert.Equal(t, subID, sub.ID)
		assert.Equal(t, "products", sub.Table)
		assert.Equal(t, userID, sub.UserID)

		// Verify subscription is tracked
		subs := subManager.GetSubscriptionsByConnection(connID)
		assert.Len(t, subs, 1)
		assert.Equal(t, subID, subs[0].ID)

		// Cleanup
		_ = subManager.RemoveSubscription(subID)
		manager.RemoveConnection(connID)
	})

	t.Run("Subscribe with specific event type", func(t *testing.T) {
		connID := uuid.New().String()
		userID := uuid.New().String()

		_, err := manager.AddConnectionWithIP(
			connID,
			nil,
			&userID,
			"authenticated",
			nil,
			"127.0.0.1",
		)
		require.NoError(t, err)

		// Subscribe to INSERT events only
		subID := uuid.New().String()
		sub, err := subManager.CreateSubscription(
			subID,
			connID,
			userID,
			"authenticated",
			nil,
			"public",
			"products",
			"INSERT",
			"",
		)

		require.NoError(t, err)
		assert.Equal(t, "INSERT", sub.Event)

		_ = subManager.RemoveSubscription(subID)
		manager.RemoveConnection(connID)
	})

	t.Run("Subscribe with filter", func(t *testing.T) {
		connID := uuid.New().String()
		userID := uuid.New().String()

		_, err := manager.AddConnectionWithIP(
			connID,
			nil,
			&userID,
			"authenticated",
			nil,
			"127.0.0.1",
		)
		require.NoError(t, err)

		// Subscribe with filter
		subID := uuid.New().String()
		sub, err := subManager.CreateSubscription(
			subID,
			connID,
			userID,
			"authenticated",
			nil,
			"public",
			"products",
			"*",
			"id=eq.1",
		)

		require.NoError(t, err)
		assert.NotNil(t, sub.Filter)
		assert.Equal(t, "id", sub.Filter.Column)
		assert.Equal(t, "eq", sub.Filter.Operator)

		_ = subManager.RemoveSubscription(subID)
		manager.RemoveConnection(connID)
	})

	t.Run("Unsubscribe from table", func(t *testing.T) {
		connID := uuid.New().String()
		userID := uuid.New().String()

		_, err := manager.AddConnectionWithIP(
			connID,
			nil,
			&userID,
			"authenticated",
			nil,
			"127.0.0.1",
		)
		require.NoError(t, err)

		// Subscribe
		subID := uuid.New().String()
		_, err = subManager.CreateSubscription(
			subID,
			connID,
			userID,
			"authenticated",
			nil,
			"public",
			"products",
			"*",
			"",
		)
		require.NoError(t, err)

		// Verify subscription exists
		subs := subManager.GetSubscriptionsByConnection(connID)
		assert.Len(t, subs, 1)

		// Unsubscribe
		err = subManager.RemoveSubscription(subID)
		require.NoError(t, err)

		// Verify subscription is removed
		subs = subManager.GetSubscriptionsByConnection(connID)
		assert.Len(t, subs, 0)

		manager.RemoveConnection(connID)
	})

	t.Run("Subscribe to non-enabled table fails", func(t *testing.T) {
		connID := uuid.New().String()
		userID := uuid.New().String()

		_, err := manager.AddConnectionWithIP(
			connID,
			nil,
			&userID,
			"authenticated",
			nil,
			"127.0.0.1",
		)
		require.NoError(t, err)

		// Try to subscribe to table that's not enabled for realtime
		subID := uuid.New().String()
		_, err = subManager.CreateSubscription(
			subID,
			connID,
			userID,
			"authenticated",
			nil,
			"public",
			"nonexistent_table_xyz",
			"*",
			"",
		)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not enabled for realtime")

		manager.RemoveConnection(connID)
	})
}

// =============================================================================
// Message Broadcasting Tests
// =============================================================================

func TestRealtimeWebSocketProtocol_Integration_Broadcasting(t *testing.T) {
	tc, manager, _, _ := setupRealtimeTest(t)
	defer cleanupRealtimeTest(t, tc, manager)

	t.Run("Broadcast message to all subscribed connections", func(t *testing.T) {
		// Create multiple connections
		connID1 := uuid.New().String()
		connID2 := uuid.New().String()
		connID3 := uuid.New().String()

		conn1, _ := manager.AddConnectionWithIP(connID1, nil, nil, "anon", nil, "127.0.0.1")
		conn2, _ := manager.AddConnectionWithIP(connID2, nil, nil, "anon", nil, "127.0.0.2")
		_, _ = manager.AddConnectionWithIP(connID3, nil, nil, "anon", nil, "127.0.0.3")

		// Subscribe conn1 and conn2 to a channel
		conn1.Subscribe("test:channel")
		conn2.Subscribe("test:channel")
		// conn3 is not subscribed

		// Broadcast message
		message := realtime.ServerMessage{
			Type:    "broadcast",
			Channel: "test:channel",
			Payload: map[string]interface{}{
				"event": "test-event",
				"data":  "test-data",
			},
		}

		sentCount := manager.BroadcastToChannel("test:channel", "", message)
		assert.Equal(t, 2, sentCount)

		// Cleanup
		manager.RemoveConnection(connID1)
		manager.RemoveConnection(connID2)
		manager.RemoveConnection(connID3)
	})

	t.Run("Broadcast to channel with no subscribers", func(t *testing.T) {
		connID := uuid.New().String()
		_, _ = manager.AddConnectionWithIP(connID, nil, nil, "anon", nil, "127.0.0.1")

		// Broadcast to channel with no subscribers
		message := realtime.ServerMessage{
			Type:    "broadcast",
			Channel: "empty:channel",
			Payload: map[string]interface{}{},
		}

		sentCount := manager.BroadcastToChannel("empty:channel", "", message)
		assert.Equal(t, 0, sentCount)

		manager.RemoveConnection(connID)
	})
}

// =============================================================================
// Database Change Event Tests
// =============================================================================

func TestRealtimeWebSocketProtocol_Integration_DatabaseChanges(t *testing.T) {
	tc, manager, subManager, mockDB := setupRealtimeTest(t)
	defer cleanupRealtimeTest(t, tc, manager)

	// Create test table
	tableName := fmt.Sprintf("realtime_test_%s", uuid.New().String()[:8])
	createRealtimeTestTable(t, tc, tableName)
	defer dropRealtimeTestTable(t, tc, tableName)

	// Enable table in mock DB
	mockDB.SetTableEnabled("public", tableName)

	t.Run("Filter event for subscribers", func(t *testing.T) {
		// Create two connections with different users
		userID1 := uuid.New().String()
		userID2 := uuid.New().String()

		connID1 := uuid.New().String()
		connID2 := uuid.New().String()

		_, _ = manager.AddConnectionWithIP(connID1, nil, &userID1, "authenticated", nil, "127.0.0.1")
		_, _ = manager.AddConnectionWithIP(connID2, nil, &userID2, "authenticated", nil, "127.0.0.2")

		// Subscribe both to the table
		subID1 := uuid.New().String()
		subID2 := uuid.New().String()

		_, err := subManager.CreateSubscription(
			subID1,
			connID1,
			userID1,
			"authenticated",
			nil,
			"public",
			tableName,
			"*",
			"",
		)
		require.NoError(t, err)

		_, err = subManager.CreateSubscription(
			subID2,
			connID2,
			userID2,
			"authenticated",
			nil,
			"public",
			tableName,
			"INSERT",
			"",
		)
		require.NoError(t, err)

		// Simulate database INSERT event
		event := &realtime.ChangeEvent{
			Type:   "INSERT",
			Schema: "public",
			Table:  tableName,
			Record: map[string]interface{}{
				"id":     1,
				"name":   "test-product",
				"status": "active",
			},
		}

		// Filter event for subscribers
		filteredEvents := subManager.FilterEventForSubscribers(context.Background(), event)

		// Both connections should receive the event (no RLS in test mode)
		assert.Len(t, filteredEvents, 2)
		assert.Contains(t, filteredEvents, connID1)
		assert.Contains(t, filteredEvents, connID2)

		// Cleanup
		_ = subManager.RemoveSubscription(subID1)
		_ = subManager.RemoveSubscription(subID2)
		manager.RemoveConnection(connID1)
		manager.RemoveConnection(connID2)
	})

	t.Run("Event type filtering", func(t *testing.T) {
		userID := uuid.New().String()
		connID := uuid.New().String()

		_, _ = manager.AddConnectionWithIP(connID, nil, &userID, "authenticated", nil, "127.0.0.1")

		// Subscribe to INSERT events only
		subID := uuid.New().String()
		_, err := subManager.CreateSubscription(
			subID,
			connID,
			userID,
			"authenticated",
			nil,
			"public",
			tableName,
			"INSERT",
			"",
		)
		require.NoError(t, err)

		// Simulate UPDATE event
		updateEvent := &realtime.ChangeEvent{
			Type:   "UPDATE",
			Schema: "public",
			Table:  tableName,
			Record: map[string]interface{}{
				"id":     1,
				"status": "updated",
			},
		}

		// Filter should not match (subscribed to INSERT only)
		filteredEvents := subManager.FilterEventForSubscribers(context.Background(), updateEvent)
		assert.Len(t, filteredEvents, 0)

		// Simulate INSERT event
		insertEvent := &realtime.ChangeEvent{
			Type:   "INSERT",
			Schema: "public",
			Table:  tableName,
			Record: map[string]interface{}{
				"id":   2,
				"name": "new-product",
			},
		}

		// Filter should match
		filteredEvents = subManager.FilterEventForSubscribers(context.Background(), insertEvent)
		assert.Len(t, filteredEvents, 1)
		assert.Contains(t, filteredEvents, connID)

		_ = subManager.RemoveSubscription(subID)
		manager.RemoveConnection(connID)
	})
}

// =============================================================================
// Connection Management Tests
// =============================================================================

func TestRealtimeWebSocketProtocol_Integration_ConnectionManagement(t *testing.T) {
	tc, manager, _, _ := setupRealtimeTest(t)
	defer cleanupRealtimeTest(t, tc, manager)

	t.Run("Multiple concurrent connections", func(t *testing.T) {
		const numConnections = 10

		connIDs := make([]string, numConnections)
		for i := 0; i < numConnections; i++ {
			connIDs[i] = uuid.New().String()
			_, err := manager.AddConnectionWithIP(
				connIDs[i],
				nil,
				nil,
				"anon",
				nil,
				fmt.Sprintf("127.0.0.%d", i+1),
			)
			require.NoError(t, err)
		}

		// Verify all connections are tracked
		assert.Equal(t, numConnections, manager.GetConnectionCount())

		// Remove all connections
		for _, connID := range connIDs {
			manager.RemoveConnection(connID)
		}

		assert.Equal(t, 0, manager.GetConnectionCount())
	})

	t.Run("Per-user connection limit", func(t *testing.T) {
		limitedManager := realtime.NewManagerWithConfig(context.Background(), realtime.ManagerConfig{
			MaxConnectionsPerUser: 2,
		})
		defer limitedManager.Shutdown()

		userID := uuid.New().String()

		conn1 := uuid.New().String()
		conn2 := uuid.New().String()
		conn3 := uuid.New().String()

		// First two connections should succeed
		_, err1 := limitedManager.AddConnectionWithIP(conn1, nil, &userID, "authenticated", nil, "127.0.0.1")
		assert.NoError(t, err1)

		_, err2 := limitedManager.AddConnectionWithIP(conn2, nil, &userID, "authenticated", nil, "127.0.0.2")
		assert.NoError(t, err2)

		// Third connection for same user should fail
		_, err3 := limitedManager.AddConnectionWithIP(conn3, nil, &userID, "authenticated", nil, "127.0.0.3")
		assert.Error(t, err3)
		assert.Equal(t, realtime.ErrMaxUserConnectionsReached, err3)
	})

	t.Run("Connection stats", func(t *testing.T) {
		stats := manager.GetDetailedStats()

		assert.NotNil(t, stats)
		assert.Contains(t, stats, "total_connections")
		assert.Contains(t, stats, "connections")
	})

	t.Run("Per-IP connection limit for anonymous users", func(t *testing.T) {
		limitedManager := realtime.NewManagerWithConfig(context.Background(), realtime.ManagerConfig{
			MaxConnectionsPerIP: 2,
		})
		defer limitedManager.Shutdown()

		ip := "192.168.1.100"

		conn1 := uuid.New().String()
		conn2 := uuid.New().String()
		conn3 := uuid.New().String()

		// First two anonymous connections from same IP should succeed
		_, err1 := limitedManager.AddConnectionWithIP(conn1, nil, nil, "anon", nil, ip)
		assert.NoError(t, err1)

		_, err2 := limitedManager.AddConnectionWithIP(conn2, nil, nil, "anon", nil, ip)
		assert.NoError(t, err2)

		// Third anonymous connection from same IP should fail
		_, err3 := limitedManager.AddConnectionWithIP(conn3, nil, nil, "anon", nil, ip)
		assert.Error(t, err3)
		assert.Equal(t, realtime.ErrMaxIPConnectionsReached, err3)
	})
}

// =============================================================================
// Filter Tests
// =============================================================================

func TestRealtimeWebSocketProtocol_Integration_Filters(t *testing.T) {
	tc, manager, subManager, mockDB := setupRealtimeTest(t)
	defer cleanupRealtimeTest(t, tc, manager)

	// Create test table
	tableName := fmt.Sprintf("realtime_test_%s", uuid.New().String()[:8])
	createRealtimeTestTable(t, tc, tableName)
	defer dropRealtimeTestTable(t, tc, tableName)

	// Enable table in mock DB
	mockDB.SetTableEnabled("public", tableName)

	t.Run("Filter by id=eq.1", func(t *testing.T) {
		userID := uuid.New().String()
		connID := uuid.New().String()

		_, _ = manager.AddConnectionWithIP(connID, nil, &userID, "authenticated", nil, "127.0.0.1")

		// Subscribe with filter id=eq.1
		subID := uuid.New().String()
		_, err := subManager.CreateSubscription(
			subID,
			connID,
			userID,
			"authenticated",
			nil,
			"public",
			tableName,
			"*",
			"id=eq.1",
		)
		require.NoError(t, err)

		// Test filter matching
		event := &realtime.ChangeEvent{
			Type:   "INSERT",
			Schema: "public",
			Table:  tableName,
			Record: map[string]interface{}{
				"id":   1,
				"name": "matching-record",
			},
		}

		// Should match (id=1)
		filteredEvents := subManager.FilterEventForSubscribers(context.Background(), event)
		assert.Len(t, filteredEvents, 1)

		// Test non-matching event
		event2 := &realtime.ChangeEvent{
			Type:   "INSERT",
			Schema: "public",
			Table:  tableName,
			Record: map[string]interface{}{
				"id":   2,
				"name": "non-matching-record",
			},
		}

		// Should not match (id=2)
		filteredEvents = subManager.FilterEventForSubscribers(context.Background(), event2)
		assert.Len(t, filteredEvents, 0)

		_ = subManager.RemoveSubscription(subID)
		manager.RemoveConnection(connID)
	})

	t.Run("Filter by status=eq.active", func(t *testing.T) {
		userID := uuid.New().String()
		connID := uuid.New().String()

		_, _ = manager.AddConnectionWithIP(connID, nil, &userID, "authenticated", nil, "127.0.0.1")

		// Subscribe with status filter
		subID := uuid.New().String()
		_, err := subManager.CreateSubscription(
			subID,
			connID,
			userID,
			"authenticated",
			nil,
			"public",
			tableName,
			"*",
			"status=eq.active",
		)
		require.NoError(t, err)

		// Test matching event
		event := &realtime.ChangeEvent{
			Type:   "INSERT",
			Schema: "public",
			Table:  tableName,
			Record: map[string]interface{}{
				"id":     1,
				"status": "active",
			},
		}

		filteredEvents := subManager.FilterEventForSubscribers(context.Background(), event)
		assert.Len(t, filteredEvents, 1)

		// Test non-matching event
		event2 := &realtime.ChangeEvent{
			Type:   "INSERT",
			Schema: "public",
			Table:  tableName,
			Record: map[string]interface{}{
				"id":     2,
				"status": "inactive",
			},
		}

		filteredEvents = subManager.FilterEventForSubscribers(context.Background(), event2)
		assert.Len(t, filteredEvents, 0)

		_ = subManager.RemoveSubscription(subID)
		manager.RemoveConnection(connID)
	})

	t.Run("Multiple subscriptions with different filters", func(t *testing.T) {
		userID1 := uuid.New().String()
		userID2 := uuid.New().String()

		connID1 := uuid.New().String()
		connID2 := uuid.New().String()

		_, _ = manager.AddConnectionWithIP(connID1, nil, &userID1, "authenticated", nil, "127.0.0.1")
		_, _ = manager.AddConnectionWithIP(connID2, nil, &userID2, "authenticated", nil, "127.0.0.2")

		// Connection 1 subscribes with filter id=eq.1
		subID1 := uuid.New().String()
		_, err := subManager.CreateSubscription(
			subID1,
			connID1,
			userID1,
			"authenticated",
			nil,
			"public",
			tableName,
			"INSERT",
			"id=eq.1",
		)
		require.NoError(t, err)

		// Connection 2 subscribes with filter id=eq.2
		subID2 := uuid.New().String()
		_, err = subManager.CreateSubscription(
			subID2,
			connID2,
			userID2,
			"authenticated",
			nil,
			"public",
			tableName,
			"INSERT",
			"id=eq.2",
		)
		require.NoError(t, err)

		// Simulate INSERT with id=1
		event1 := &realtime.ChangeEvent{
			Type:   "INSERT",
			Schema: "public",
			Table:  tableName,
			Record: map[string]interface{}{
				"id":   1,
				"name": "record-1",
			},
		}

		filteredEvents := subManager.FilterEventForSubscribers(context.Background(), event1)
		assert.Len(t, filteredEvents, 1)
		assert.Contains(t, filteredEvents, connID1)
		assert.NotContains(t, filteredEvents, connID2)

		// Simulate INSERT with id=2
		event2 := &realtime.ChangeEvent{
			Type:   "INSERT",
			Schema: "public",
			Table:  tableName,
			Record: map[string]interface{}{
				"id":   2,
				"name": "record-2",
			},
		}

		filteredEvents = subManager.FilterEventForSubscribers(context.Background(), event2)
		assert.Len(t, filteredEvents, 1)
		assert.NotContains(t, filteredEvents, connID1)
		assert.Contains(t, filteredEvents, connID2)

		_ = subManager.RemoveSubscription(subID1)
		_ = subManager.RemoveSubscription(subID2)
		manager.RemoveConnection(connID1)
		manager.RemoveConnection(connID2)
	})
}

// =============================================================================
// Subscription Statistics Tests
// =============================================================================

func TestRealtimeWebSocketProtocol_Integration_SubscriptionStats(t *testing.T) {
	tc, manager, subManager, mockDB := setupRealtimeTest(t)
	defer cleanupRealtimeTest(t, tc, manager)

	t.Run("Get subscription statistics", func(t *testing.T) {
		// Initially no subscriptions
		stats := subManager.GetStats()
		assert.Equal(t, 0, stats["total_subscriptions"])
		assert.Equal(t, 0, stats["users_with_subs"])
		assert.Equal(t, 0, stats["tables_with_subs"])

		// Create test table
		tableName := fmt.Sprintf("realtime_test_%s", uuid.New().String()[:8])
		createRealtimeTestTable(t, tc, tableName)
		defer dropRealtimeTestTable(t, tc, tableName)

		// Enable table in mock DB
		mockDB.SetTableEnabled("public", tableName)

		// Add connection and subscribe
		userID := uuid.New().String()
		connID := uuid.New().String()

		_, _ = manager.AddConnectionWithIP(connID, nil, &userID, "authenticated", nil, "127.0.0.1")

		subID := uuid.New().String()
		_, err := subManager.CreateSubscription(
			subID,
			connID,
			userID,
			"authenticated",
			nil,
			"public",
			tableName,
			"*",
			"",
		)
		require.NoError(t, err)

		// Check stats after subscription
		stats = subManager.GetStats()
		assert.Equal(t, 1, stats["total_subscriptions"])
		assert.Equal(t, 1, stats["users_with_subs"])
		assert.Equal(t, 1, stats["tables_with_subs"])

		// Cleanup
		_ = subManager.RemoveSubscription(subID)
		manager.RemoveConnection(connID)
	})
}

// =============================================================================
// Helper Functions
// =============================================================================

// createRealtimeTestTable creates a test table for realtime notifications
func createRealtimeTestTable(t *testing.T, tc *test.TestContext, tableName string) {
	tc.ExecuteSQL(fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS public.%s (
			id SERIAL PRIMARY KEY,
			name TEXT NOT NULL,
			status TEXT DEFAULT 'pending'
		)
	`, tableName))

	// Disable RLS for test tables (simplifies testing)
	tc.ExecuteSQL(fmt.Sprintf(`ALTER TABLE public.%s DISABLE ROW LEVEL SECURITY`, tableName))

	// Grant necessary permissions
	tc.ExecuteSQL(fmt.Sprintf(`GRANT SELECT ON public.%s TO fluxbase_app`, tableName))
	tc.ExecuteSQL(fmt.Sprintf(`GRANT INSERT ON public.%s TO fluxbase_app`, tableName))
	tc.ExecuteSQL(fmt.Sprintf(`GRANT UPDATE ON public.%s TO fluxbase_app`, tableName))
	tc.ExecuteSQL(fmt.Sprintf(`GRANT DELETE ON public.%s TO fluxbase_app`, tableName))

	// Enable realtime for this table
	tc.ExecuteSQL(fmt.Sprintf(`
		INSERT INTO realtime.schema_registry (schema_name, table_name, realtime_enabled)
		VALUES ('public', '%s', true)
		ON CONFLICT (schema_name, table_name) DO UPDATE
		SET realtime_enabled = true
	`, tableName))
}

// dropRealtimeTestTable drops a test table
func dropRealtimeTestTable(t *testing.T, tc *test.TestContext, tableName string) {
	tc.ExecuteSQL(fmt.Sprintf(`DROP TABLE IF EXISTS public.%s CASCADE`, tableName))
}
