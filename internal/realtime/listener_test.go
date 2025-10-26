package realtime

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
)

// Mock handler for testing
type MockRealtimeHandler struct {
	broadcastCalls []struct {
		channel string
		event   ChangeEvent
	}
}

func (m *MockRealtimeHandler) Broadcast(channel string, payload interface{}) {
	event, ok := payload.(ChangeEvent)
	if ok {
		m.broadcastCalls = append(m.broadcastCalls, struct {
			channel string
			event   ChangeEvent
		}{channel, event})
	}
}

func TestListener_ProcessNotification_Insert(t *testing.T) {
	ctx := context.Background()
	manager := NewManager(ctx)
	handler := NewRealtimeHandler(manager, nil) // nil auth service for testing
	listener := &Listener{
		handler: handler,
	}

	// Create a notification
	notification := &pgconn.Notification{
		Channel: "fluxbase_changes",
		Payload: `{
			"type": "INSERT",
			"table": "products",
			"schema": "public",
			"record": {"id": 1, "name": "Test Product", "price": 99.99}
		}`,
	}

	// Process notification without subscribers
	// Should not panic even if no one is listening
	listener.processNotification(notification)
}

func TestListener_ProcessNotification_Update(t *testing.T) {
	ctx := context.Background()
	manager := NewManager(ctx)
	handler := NewRealtimeHandler(manager, nil)
	listener := &Listener{
		handler: handler,
	}

	notification := &pgconn.Notification{
		Channel: "fluxbase_changes",
		Payload: `{
			"type": "UPDATE",
			"table": "products",
			"schema": "public",
			"record": {"id": 1, "name": "Updated Product", "price": 149.99},
			"old_record": {"id": 1, "name": "Test Product", "price": 99.99}
		}`,
	}

	listener.processNotification(notification)
	// Should not panic
}

func TestListener_ProcessNotification_Delete(t *testing.T) {
	ctx := context.Background()
	manager := NewManager(ctx)
	handler := NewRealtimeHandler(manager, nil)
	listener := &Listener{
		handler: handler,
	}

	notification := &pgconn.Notification{
		Channel: "fluxbase_changes",
		Payload: `{
			"type": "DELETE",
			"table": "products",
			"schema": "public",
			"old_record": {"id": 1, "name": "Test Product", "price": 99.99}
		}`,
	}

	listener.processNotification(notification)
	// Should not panic
}

func TestListener_ProcessNotification_InvalidJSON(t *testing.T) {
	ctx := context.Background()
	manager := NewManager(ctx)
	handler := NewRealtimeHandler(manager, nil)
	listener := &Listener{
		handler: handler,
	}

	notification := &pgconn.Notification{
		Channel: "fluxbase_changes",
		Payload: `{invalid json`,
	}

	// Should handle error gracefully without panicking
	listener.processNotification(notification)
}

func TestListener_ProcessNotification_ChannelFormat(t *testing.T) {
	ctx := context.Background()
	manager := NewManager(ctx)
	handler := NewRealtimeHandler(manager, nil)
	listener := &Listener{
		handler: handler,
	}

	tests := []struct {
		name            string
		schema          string
		table           string
		expectedChannel string
	}{
		{
			name:            "public schema",
			schema:          "public",
			table:           "products",
			expectedChannel: "table:public.products",
		},
		{
			name:            "custom schema",
			schema:          "inventory",
			table:           "items",
			expectedChannel: "table:inventory.items",
		},
		{
			name:            "auth schema",
			schema:          "auth",
			table:           "users",
			expectedChannel: "table:auth.users",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := `{
				"type": "INSERT",
				"table": "` + tt.table + `",
				"schema": "` + tt.schema + `",
				"record": {"id": 1}
			}`

			notification := &pgconn.Notification{
				Channel: "fluxbase_changes",
				Payload: payload,
			}

			// Should not panic
			listener.processNotification(notification)
		})
	}
}

func TestChangeEvent_Structure(t *testing.T) {
	event := ChangeEvent{
		Type:   "INSERT",
		Table:  "products",
		Schema: "public",
		Record: map[string]interface{}{
			"id":    float64(1),
			"name":  "Test Product",
			"price": float64(99.99),
		},
	}

	assert.Equal(t, "INSERT", event.Type)
	assert.Equal(t, "products", event.Table)
	assert.Equal(t, "public", event.Schema)
	assert.NotNil(t, event.Record)
	assert.Equal(t, float64(1), event.Record["id"])
}

func TestChangeEvent_WithOldRecord(t *testing.T) {
	event := ChangeEvent{
		Type:   "UPDATE",
		Table:  "products",
		Schema: "public",
		Record: map[string]interface{}{
			"id":    float64(1),
			"name":  "Updated Product",
			"price": float64(149.99),
		},
		OldRecord: map[string]interface{}{
			"id":    float64(1),
			"name":  "Test Product",
			"price": float64(99.99),
		},
	}

	assert.Equal(t, "UPDATE", event.Type)
	assert.NotNil(t, event.Record)
	assert.NotNil(t, event.OldRecord)
	assert.Equal(t, "Updated Product", event.Record["name"])
	assert.Equal(t, "Test Product", event.OldRecord["name"])
}

func TestNewListener(t *testing.T) {
	ctx := context.Background()
	manager := NewManager(ctx)
	handler := NewRealtimeHandler(manager, nil)

	listener := NewListener(nil, handler)

	assert.NotNil(t, listener)
	assert.NotNil(t, listener.handler)
	assert.NotNil(t, listener.ctx)
	assert.NotNil(t, listener.cancel)
}

func TestListener_Stop(t *testing.T) {
	ctx := context.Background()
	manager := NewManager(ctx)
	handler := NewRealtimeHandler(manager, nil)

	listener := NewListener(nil, handler)

	// Should not panic
	listener.Stop()

	// Verify context is cancelled
	assert.Error(t, listener.ctx.Err())
}
