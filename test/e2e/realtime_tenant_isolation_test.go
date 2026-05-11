//go:build integration

package e2e

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/nimbleflux/fluxbase/internal/realtime"
)

const (
	rtTenantA = "11111111-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	rtTenantB = "22222222-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
)

func TestRealtimeTenantIsolation_BroadcastFiltering(t *testing.T) {
	ctx := context.Background()
	manager := realtime.NewManager(ctx)

	connA1, _ := manager.AddConnection("conn-a1", nil, nil, "anon", nil, rtTenantA)
	connA2, _ := manager.AddConnection("conn-a2", nil, nil, "anon", nil, rtTenantA)
	manager.AddConnection("conn-b1", nil, nil, "anon", nil, rtTenantB)

	channel := "test-channel"
	connA1.Subscribe(channel)
	connA2.Subscribe(channel)

	message := realtime.ServerMessage{
		Type:    realtime.MessageTypeBroadcast,
		Channel: channel,
		Payload: map[string]interface{}{"hello": "world"},
	}

	sentCount := manager.BroadcastToChannel(channel, rtTenantA, message)
	assert.Equal(t, 2, sentCount, "Only tenant A connections should receive the message")

	sentToB := manager.BroadcastToChannel(channel, rtTenantB, message)
	assert.Equal(t, 0, sentToB, "Tenant B conn is not subscribed so sent should be 0")
}

func TestRealtimeTenantIsolation_TenantBDoesNotReceiveTenantAMessages(t *testing.T) {
	ctx := context.Background()
	manager := realtime.NewManager(ctx)

	connA, _ := manager.AddConnection("conn-a", nil, nil, "anon", nil, rtTenantA)
	connB, _ := manager.AddConnection("conn-b", nil, nil, "anon", nil, rtTenantB)

	channel := "shared-channel"
	connA.Subscribe(channel)
	connB.Subscribe(channel)

	msgA := realtime.ServerMessage{
		Type:    realtime.MessageTypeBroadcast,
		Channel: channel,
		Payload: map[string]interface{}{"data": "from-A"},
	}
	sentA := manager.BroadcastToChannel(channel, rtTenantA, msgA)
	assert.Equal(t, 1, sentA, "Only tenant A should receive A's message")

	msgB := realtime.ServerMessage{
		Type:    realtime.MessageTypeBroadcast,
		Channel: channel,
		Payload: map[string]interface{}{"data": "from-B"},
	}
	sentB := manager.BroadcastToChannel(channel, rtTenantB, msgB)
	assert.Equal(t, 1, sentB, "Only tenant B should receive B's message")
}

func TestRealtimeTenantIsolation_MultipleConnectionsPerTenant(t *testing.T) {
	ctx := context.Background()
	manager := realtime.NewManager(ctx)

	var connsA []*realtime.Connection
	for i := 0; i < 5; i++ {
		conn, _ := manager.AddConnection("conn-a-"+string(rune('0'+i)), nil, nil, "anon", nil, rtTenantA)
		connsA = append(connsA, conn)
	}
	connB, _ := manager.AddConnection("conn-b", nil, nil, "anon", nil, rtTenantB)

	channel := "fanout"
	for _, c := range connsA {
		c.Subscribe(channel)
	}
	connB.Subscribe(channel)

	message := realtime.ServerMessage{
		Type:    realtime.MessageTypeBroadcast,
		Channel: channel,
		Payload: map[string]interface{}{"fanout": true},
	}

	sentCount := manager.BroadcastToChannel(channel, rtTenantA, message)
	assert.Equal(t, 5, sentCount, "All 5 tenant A connections should receive the fanout")

	sentToB := manager.BroadcastToChannel(channel, rtTenantB, message)
	assert.Equal(t, 1, sentToB, "Tenant B should only receive its own broadcast")
}

func TestRealtimeTenantIsolation_SameChannelDifferentTenants(t *testing.T) {
	ctx := context.Background()
	manager := realtime.NewManager(ctx)

	connA, _ := manager.AddConnection("conn-a", nil, nil, "anon", nil, rtTenantA)
	connB, _ := manager.AddConnection("conn-b", nil, nil, "anon", nil, rtTenantB)
	connC, _ := manager.AddConnection("conn-c", nil, nil, "anon", nil, "unknown-tenant")

	channel := "shared"
	connA.Subscribe(channel)
	connB.Subscribe(channel)
	connC.Subscribe(channel)

	msg := realtime.ServerMessage{
		Type:    realtime.MessageTypeBroadcast,
		Channel: channel,
		Payload: map[string]interface{}{},
	}

	sentA := manager.BroadcastToChannel(channel, rtTenantA, msg)
	assert.Equal(t, 1, sentA, "Only tenant A receives on shared channel from A's broadcast")

	sentB := manager.BroadcastToChannel(channel, rtTenantB, msg)
	assert.Equal(t, 1, sentB, "Only tenant B receives on shared channel from B's broadcast")

	sentC := manager.BroadcastToChannel(channel, "unknown-tenant", msg)
	assert.Equal(t, 1, sentC, "Only unknown-tenant receives on shared channel from C's broadcast")
}

func TestRealtimeTenantIsolation_ConnectionCleanup(t *testing.T) {
	ctx := context.Background()
	manager := realtime.NewManager(ctx)

	connA, _ := manager.AddConnection("conn-a", nil, nil, "anon", nil, rtTenantA)
	connB, _ := manager.AddConnection("conn-b", nil, nil, "anon", nil, rtTenantB)

	channel := "test"
	connA.Subscribe(channel)
	connB.Subscribe(channel)

	manager.RemoveConnection("conn-a")

	msg := realtime.ServerMessage{
		Type:    realtime.MessageTypeBroadcast,
		Channel: channel,
		Payload: map[string]interface{}{},
	}

	sentA := manager.BroadcastToChannel(channel, rtTenantA, msg)
	assert.Equal(t, 0, sentA, "Removed connection should not receive messages")

	sentB := manager.BroadcastToChannel(channel, rtTenantB, msg)
	assert.Equal(t, 1, sentB, "Tenant B connection should still work after removing tenant A's connection")
}
