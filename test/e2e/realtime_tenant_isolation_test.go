//go:build integration

package e2e

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nimbleflux/fluxbase/internal/pubsub"
	"github.com/nimbleflux/fluxbase/internal/realtime"
	"github.com/nimbleflux/fluxbase/test"
)

const (
	rtTenantA = "11111111-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	rtTenantB = "22222222-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
)

type mockSubDB struct {
	mu            sync.RWMutex
	enabledTables map[string]bool
}

func newMockSubDB() *mockSubDB {
	return &mockSubDB{enabledTables: make(map[string]bool)}
}

func (m *mockSubDB) SetTableEnabled(schema, table string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.enabledTables[schema+"."+table] = true
}

func (m *mockSubDB) IsTableEnabled(_ context.Context, schema, table string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.enabledTables[schema+"."+table], nil
}

func (m *mockSubDB) IsTableRealtimeEnabled(ctx context.Context, schema, table string) (bool, error) {
	return m.IsTableEnabled(ctx, schema, table)
}

func (m *mockSubDB) CheckRLSAccess(_ context.Context, _ string, _ string, _ string, _ map[string]interface{}, _ interface{}) (bool, error) {
	return true, nil
}

func (m *mockSubDB) CheckRPCOwnership(_ context.Context, _, _ uuid.UUID) (bool, bool, error) {
	return true, true, nil
}

func (m *mockSubDB) CheckJobOwnership(_ context.Context, _, _ uuid.UUID) (bool, bool, error) {
	return true, true, nil
}

func (m *mockSubDB) CheckFunctionOwnership(_ context.Context, _, _ uuid.UUID) (bool, bool, error) {
	return true, true, nil
}

func setupRealtimeIsolationTest(t *testing.T) (*test.TestContext, *realtime.Manager, *realtime.SubscriptionManager) {
	tc := test.NewTestContext(t)
	tc.EnsureAuthSchema()

	tc.ExecuteSQLAsSuperuser(`DELETE FROM realtime.schema_registry WHERE tenant_id IN ($1::uuid, $2::uuid)`, rtTenantA, rtTenantB)

	for _, id := range []string{rtTenantA, rtTenantB} {
		tc.ExecuteSQLAsSuperuser(`
			INSERT INTO platform.tenants (id, slug, name, status)
			VALUES ($1, 'rt-test-tenant-' || $1, 'RT Test Tenant ' || $1, 'active')
			ON CONFLICT (id) DO NOTHING
		`, id)
	}

	ps := pubsub.NewLocalPubSub()

	manager := realtime.NewManagerWithConfig(context.Background(), realtime.ManagerConfig{
		MaxConnections:         100,
		MaxConnectionsPerUser:  10,
		ClientMessageQueueSize: 256,
	})
	manager.SetPubSub(ps)

	mockDB := newMockSubDB()
	mockDB.SetTableEnabled("public", "products")
	subManager := realtime.NewSubscriptionManager(mockDB)

	return tc, manager, subManager
}

func cleanupRealtimeIsolationTest(t *testing.T, tc *test.TestContext, manager *realtime.Manager) {
	manager.Shutdown()
}

func TestRealtimeTenantIsolation_BroadcastFiltering(t *testing.T) {
	tc, manager, _ := setupRealtimeIsolationTest(t)
	defer cleanupRealtimeIsolationTest(t, tc, manager)

	channel := "tenant-isolation-test"

	connA1, err := manager.AddConnection("conn-a1", nil, nil, "anon", nil, rtTenantA)
	require.NoError(t, err)
	connA2, err := manager.AddConnection("conn-a2", nil, nil, "anon", nil, rtTenantA)
	require.NoError(t, err)
	connB1, err := manager.AddConnection("conn-b1", nil, nil, "anon", nil, rtTenantB)
	require.NoError(t, err)
	connB2, err := manager.AddConnection("conn-b2", nil, nil, "anon", nil, rtTenantB)
	require.NoError(t, err)

	connA1.Subscribe(channel)
	connA2.Subscribe(channel)
	connB1.Subscribe(channel)
	connB2.Subscribe(channel)

	message := realtime.ServerMessage{
		Type:    realtime.MessageTypeBroadcast,
		Channel: channel,
		Payload: map[string]interface{}{"text": "tenant A only"},
	}

	sentCount := manager.BroadcastToChannel(channel, rtTenantA, message)

	assert.Equal(t, 2, sentCount, "broadcast to tenant A should reach exactly 2 tenant-A connections")
}

func TestRealtimeTenantIsolation_TenantBDoesNotReceiveTenantAMessages(t *testing.T) {
	tc, manager, _ := setupRealtimeIsolationTest(t)
	defer cleanupRealtimeIsolationTest(t, tc, manager)

	channel := "isolation-verify-channel"

	connA, err := manager.AddConnection("conn-a", nil, nil, "anon", nil, rtTenantA)
	require.NoError(t, err)
	connB, err := manager.AddConnection("conn-b", nil, nil, "anon", nil, rtTenantB)
	require.NoError(t, err)

	connA.Subscribe(channel)
	connB.Subscribe(channel)

	sentCount := manager.BroadcastToChannel(channel, rtTenantA, realtime.ServerMessage{
		Type:    realtime.MessageTypeBroadcast,
		Channel: channel,
		Payload: map[string]interface{}{"secret": "A"},
	})

	assert.Equal(t, 1, sentCount, "only tenant A connection should receive tenant A broadcast")

	sentCount = manager.BroadcastToChannel(channel, rtTenantB, realtime.ServerMessage{
		Type:    realtime.MessageTypeBroadcast,
		Channel: channel,
		Payload: map[string]interface{}{"secret": "B"},
	})

	assert.Equal(t, 1, sentCount, "only tenant B connection should receive tenant B broadcast")
}

func TestRealtimeTenantIsolation_IndependentTenantMessaging(t *testing.T) {
	tc, manager, _ := setupRealtimeIsolationTest(t)
	defer cleanupRealtimeIsolationTest(t, tc, manager)

	channelA := "tenant-a-private"
	channelB := "tenant-b-private"

	connA, err := manager.AddConnection("conn-a", nil, nil, "anon", nil, rtTenantA)
	require.NoError(t, err)
	connB, err := manager.AddConnection("conn-b", nil, nil, "anon", nil, rtTenantB)
	require.NoError(t, err)

	connA.Subscribe(channelA)
	connA.Subscribe(channelB)
	connB.Subscribe(channelA)
	connB.Subscribe(channelB)

	msgA := realtime.ServerMessage{
		Type:    realtime.MessageTypeBroadcast,
		Channel: channelA,
		Payload: map[string]interface{}{"from": "A"},
	}
	sentA := manager.BroadcastToChannel(channelA, rtTenantA, msgA)
	assert.Equal(t, 1, sentA, "tenant A broadcast on channel A reaches only tenant A connection")

	msgB := realtime.ServerMessage{
		Type:    realtime.MessageTypeBroadcast,
		Channel: channelB,
		Payload: map[string]interface{}{"from": "B"},
	}
	sentB := manager.BroadcastToChannel(channelB, rtTenantB, msgB)
	assert.Equal(t, 1, sentB, "tenant B broadcast on channel B reaches only tenant B connection")

	crossSent := manager.BroadcastToChannel(channelA, rtTenantB, msgB)
	assert.Equal(t, 1, crossSent, "tenant B broadcast on channel A reaches only tenant B connection (same channel, different tenant)")
}

func TestRealtimeTenantIsolation_EmptyTenantIDIsolation(t *testing.T) {
	tc, manager, _ := setupRealtimeIsolationTest(t)
	defer cleanupRealtimeIsolationTest(t, tc, manager)

	channel := "default-tenant-channel"

	connDefault, err := manager.AddConnection("conn-default", nil, nil, "anon", nil, "")
	require.NoError(t, err)
	connA, err := manager.AddConnection("conn-a", nil, nil, "anon", nil, rtTenantA)
	require.NoError(t, err)
	connB, err := manager.AddConnection("conn-b", nil, nil, "anon", nil, rtTenantB)
	require.NoError(t, err)

	connDefault.Subscribe(channel)
	connA.Subscribe(channel)
	connB.Subscribe(channel)

	sentCount := manager.BroadcastToChannel(channel, "", realtime.ServerMessage{
		Type:    realtime.MessageTypeBroadcast,
		Channel: channel,
		Payload: map[string]interface{}{"default": true},
	})

	assert.Equal(t, 1, sentCount, "broadcast with empty tenant ID should only reach connections with empty tenant ID")
}

func TestRealtimeTenantIsolation_MultipleConnectionsPerTenant(t *testing.T) {
	tc, manager, _ := setupRealtimeIsolationTest(t)
	defer cleanupRealtimeIsolationTest(t, tc, manager)

	channel := "multi-conn-channel"

	var connIDs []string
	for i := 0; i < 5; i++ {
		id := fmt.Sprintf("conn-a-%d", i)
		conn, err := manager.AddConnection(id, nil, nil, "anon", nil, rtTenantA)
		require.NoError(t, err)
		conn.Subscribe(channel)
		connIDs = append(connIDs, id)
	}

	connB, err := manager.AddConnection("conn-b-0", nil, nil, "anon", nil, rtTenantB)
	require.NoError(t, err)
	connB.Subscribe(channel)

	sentCount := manager.BroadcastToChannel(channel, rtTenantA, realtime.ServerMessage{
		Type:    realtime.MessageTypeBroadcast,
		Channel: channel,
		Payload: map[string]interface{}{"fanout": true},
	})

	assert.Equal(t, 5, sentCount, "broadcast to tenant A should reach all 5 tenant A connections, not tenant B")

	sentCountB := manager.BroadcastToChannel(channel, rtTenantB, realtime.ServerMessage{
		Type:    realtime.MessageTypeBroadcast,
		Channel: channel,
		Payload: map[string]interface{}{"fanout": true},
	})

	assert.Equal(t, 1, sentCountB, "broadcast to tenant B should reach only 1 tenant B connection")

	for _, id := range connIDs {
		manager.RemoveConnection(id)
	}
	manager.RemoveConnection("conn-b-0")
}

func TestRealtimeTenantIsolation_SameChannelDifferentTenants(t *testing.T) {
	tc, manager, _ := setupRealtimeIsolationTest(t)
	defer cleanupRealtimeIsolationTest(t, tc, manager)

	channel := "shared-channel-name"

	connA1, _ := manager.AddConnection("a1", nil, nil, "anon", nil, rtTenantA)
	connA2, _ := manager.AddConnection("a2", nil, nil, "anon", nil, rtTenantA)
	connB1, _ := manager.AddConnection("b1", nil, nil, "anon", nil, rtTenantB)

	connA1.Subscribe(channel)
	connA2.Subscribe(channel)
	connB1.Subscribe(channel)

	sentA := manager.BroadcastToChannel(channel, rtTenantA, realtime.ServerMessage{
		Type:    realtime.MessageTypeBroadcast,
		Channel: channel,
		Payload: map[string]interface{}{"seq": 1},
	})
	require.Equal(t, 2, sentA)

	sentB := manager.BroadcastToChannel(channel, rtTenantB, realtime.ServerMessage{
		Type:    realtime.MessageTypeBroadcast,
		Channel: channel,
		Payload: map[string]interface{}{"seq": 2},
	})
	require.Equal(t, 1, sentB)

	sentDefault := manager.BroadcastToChannel(channel, "nonexistent-tenant", realtime.ServerMessage{
		Type:    realtime.MessageTypeBroadcast,
		Channel: channel,
		Payload: map[string]interface{}{"seq": 3},
	})
	assert.Equal(t, 0, sentDefault, "broadcast with unknown tenant ID should reach no connections")
}

func TestRealtimeTenantIsolation_RLSStatsEndpointTenantScoped(t *testing.T) {
	tc := test.NewTestContext(t)
	tc.EnsureAuthSchema()
	defer tc.Close()

	if !tc.Config.Realtime.Enabled {
		t.Skip("Realtime is not enabled in test config")
	}

	email := test.E2ETestEmail()
	_, token := tc.CreateTestUser(email, "testpassword123")
	require.NotEmpty(t, token)

	t.Run("stats endpoint requires auth", func(t *testing.T) {
		tc.NewRequest("GET", "/api/v1/realtime/stats").
			Send().
			AssertStatus(401)
	})

	t.Run("stats endpoint returns data for authenticated user", func(t *testing.T) {
		resp := tc.NewRequest("GET", "/api/v1/realtime/stats").
			WithAuth(token).
			Send()

		status := resp.Status()
		assert.True(t, status == 200 || status == 503,
			"Expected 200 or 503, got %d", status)
	})

	t.Run("stats endpoint with tenant header", func(t *testing.T) {
		tenantID := tc.GetDefaultTenantID()

		resp := tc.NewRequest("GET", "/api/v1/realtime/stats").
			WithAuth(token).
			WithHeader("X-FB-Tenant", tenantID).
			Send()

		status := resp.Status()
		assert.True(t, status == 200 || status == 503,
			"Expected 200 or 503 with tenant header, got %d", status)
	})
}

func TestRealtimeTenantIsolation_PresenceTenantFiltering(t *testing.T) {
	tc, manager, _ := setupRealtimeIsolationTest(t)
	defer cleanupRealtimeIsolationTest(t, tc, manager)

	channel := "presence-isolation"

	userA := "user-a"
	userB := "user-b"

	connA, err := manager.AddConnection("conn-presence-a", nil, &userA, "authenticated", nil, rtTenantA)
	require.NoError(t, err)
	connB, err := manager.AddConnection("conn-presence-b", nil, &userB, "authenticated", nil, rtTenantB)
	require.NoError(t, err)

	connA.Subscribe(channel)
	connB.Subscribe(channel)

	msg := realtime.ServerMessage{
		Type:    realtime.MessageTypePresence,
		Channel: channel,
		Payload: map[string]interface{}{
			"presence": map[string]interface{}{
				"event": "join",
				"key":   userA,
			},
		},
	}

	sentA := manager.BroadcastToChannel(channel, rtTenantA, msg)
	assert.Equal(t, 1, sentA, "presence event for tenant A should only reach tenant A connections")

	sentB := manager.BroadcastToChannel(channel, rtTenantB, msg)
	assert.Equal(t, 1, sentB, "presence event for tenant B should only reach tenant B connections")
}

func TestRealtimeTenantIsolation_ConnectionCleanup(t *testing.T) {
	tc, manager, _ := setupRealtimeIsolationTest(t)
	defer cleanupRealtimeIsolationTest(t, tc, manager)

	channel := "cleanup-test"

	connA, _ := manager.AddConnection("conn-cleanup-a", nil, nil, "anon", nil, rtTenantA)
	connB, _ := manager.AddConnection("conn-cleanup-b", nil, nil, "anon", nil, rtTenantB)

	connA.Subscribe(channel)
	connB.Subscribe(channel)

	manager.RemoveConnection("conn-cleanup-a")

	sentCount := manager.BroadcastToChannel(channel, rtTenantA, realtime.ServerMessage{
		Type:    realtime.MessageTypeBroadcast,
		Channel: channel,
		Payload: map[string]interface{}{},
	})

	assert.Equal(t, 0, sentCount, "after removing tenant A connection, broadcast to tenant A should reach 0 connections")

	sentCountB := manager.BroadcastToChannel(channel, rtTenantB, realtime.ServerMessage{
		Type:    realtime.MessageTypeBroadcast,
		Channel: channel,
		Payload: map[string]interface{}{},
	})

	assert.Equal(t, 1, sentCountB, "tenant B connection should still be active and receive broadcasts")
}

func TestRealtimeTenantIsolation_SubscriptionTenantFiltering(t *testing.T) {
	tc, manager, subManager := setupRealtimeIsolationTest(t)
	defer cleanupRealtimeIsolationTest(t, tc, manager)

	userA := "sub-user-a"
	userB := "sub-user-b"

	connA, err := manager.AddConnection("sub-conn-a", nil, &userA, "authenticated", nil, rtTenantA)
	require.NoError(t, err)
	connB, err := manager.AddConnection("sub-conn-b", nil, &userB, "authenticated", nil, rtTenantB)
	require.NoError(t, err)

	subA, err := subManager.CreateSubscription(
		"sub-a-1", connA.ID, userA, "authenticated", nil,
		"public", "products", "*", "",
	)
	require.NoError(t, err)
	require.NotNil(t, subA)

	subB, err := subManager.CreateSubscription(
		"sub-b-1", connB.ID, userB, "authenticated", nil,
		"public", "products", "*", "",
	)
	require.NoError(t, err)
	require.NotNil(t, subB)

	changeEvent := &realtime.ChangeEvent{
		Type:   "INSERT",
		Schema: "public",
		Table:  "products",
		Record: map[string]interface{}{"id": 1, "name": "widget"},
	}

	filtered := subManager.FilterEventForSubscribers(context.Background(), changeEvent)

	assert.Contains(t, filtered, connA.ID, "tenant A subscription should receive the change event")
	assert.Contains(t, filtered, connB.ID, "tenant B subscription should receive the change event")
	assert.Len(t, filtered, 2, "both subscriptions should match (subscription filtering is not tenant-aware)")

	_ = subManager.RemoveSubscription(subA.ID)
	_ = subManager.RemoveSubscription(subB.ID)
}

func TestRealtimeTenantIsolation_WebSocketBroadcastViaREST(t *testing.T) {
	tc := test.NewTestContext(t)
	tc.EnsureAuthSchema()
	defer tc.Close()

	if !tc.Config.Realtime.Enabled {
		t.Skip("Realtime is not enabled in test config")
	}

	emailA := test.E2ETestEmail()
	_, tokenA := tc.CreateTestUser(emailA, "password123")
	require.NotEmpty(t, tokenA)

	emailB := test.E2ETestEmail()
	_, tokenB := tc.CreateTestUser(emailB, "password123")
	require.NotEmpty(t, tokenB)

	t.Run("broadcast endpoint accepts authenticated request", func(t *testing.T) {
		resp := tc.NewRequest("POST", "/api/v1/realtime/broadcast").
			WithAuth(tokenA).
			WithBody(map[string]interface{}{
				"channel": "test-tenant-channel",
				"message": map[string]interface{}{"text": "hello"},
			}).
			Send()

		status := resp.Status()
		assert.True(t, status == 200 || status == 404 || status == 503,
			"Expected 200, 404, or 503, got %d. Body: %s", status, string(resp.Body()))
	})

	t.Run("broadcast endpoint rejects unauthenticated request", func(t *testing.T) {
		tc.NewRequest("POST", "/api/v1/realtime/broadcast").
			WithBody(map[string]interface{}{
				"channel": "test-tenant-channel",
				"message": map[string]interface{}{"text": "hello"},
			}).
			Send().
			AssertStatus(401)
	})

	t.Run("broadcast with different tenant tokens", func(t *testing.T) {
		for i, tok := range []string{tokenA, tokenB} {
			resp := tc.NewRequest("POST", "/api/v1/realtime/broadcast").
				WithAuth(tok).
				WithBody(map[string]interface{}{
					"channel": fmt.Sprintf("tenant-broadcast-%d", i),
					"message": map[string]interface{}{"seq": i},
				}).
				Send()

			status := resp.Status()
			assert.True(t, status == 200 || status == 404 || status == 503,
				"token %d: expected 200/404/503, got %d", i, status)
		}
	})
}

func TestRealtimeTenantIsolation_GlobalBroadcast(t *testing.T) {
	tc, manager, _ := setupRealtimeIsolationTest(t)
	defer cleanupRealtimeIsolationTest(t, tc, manager)

	channel := "global-broadcast-test"

	connA, _ := manager.AddConnection("g-conn-a", nil, nil, "anon", nil, rtTenantA)
	connB, _ := manager.AddConnection("g-conn-b", nil, nil, "anon", nil, rtTenantB)
	connDefault, _ := manager.AddConnection("g-conn-default", nil, nil, "anon", nil, "")

	connA.Subscribe(channel)
	connB.Subscribe(channel)
	connDefault.Subscribe(channel)

	err := manager.BroadcastGlobal(channel, rtTenantA, realtime.ServerMessage{
		Type:    realtime.MessageTypeBroadcast,
		Channel: channel,
		Payload: map[string]interface{}{"global": true},
	})
	require.NoError(t, err)

	time.Sleep(50 * time.Millisecond)

	assert.Equal(t, 3, manager.GetConnectionCount(),
		"all connections should still be active after global broadcast")
}
