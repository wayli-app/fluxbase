package realtime

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// =============================================================================
// ConnectionEventType Tests
// =============================================================================

func TestConnectionEventType_Constants(t *testing.T) {
	t.Run("ConnectionEventConnected has expected value", func(t *testing.T) {
		assert.Equal(t, ConnectionEventType("connected"), ConnectionEventConnected)
	})

	t.Run("ConnectionEventDisconnected has expected value", func(t *testing.T) {
		assert.Equal(t, ConnectionEventType("disconnected"), ConnectionEventDisconnected)
	})

	t.Run("event types are distinct", func(t *testing.T) {
		assert.NotEqual(t, ConnectionEventConnected, ConnectionEventDisconnected)
	})

	t.Run("event types can be used as strings", func(t *testing.T) {
		assert.Equal(t, "connected", string(ConnectionEventConnected))
		assert.Equal(t, "disconnected", string(ConnectionEventDisconnected))
	})
}

// =============================================================================
// ConnectionEvent Tests
// =============================================================================

func TestConnectionEvent_Struct(t *testing.T) {
	t.Run("zero value", func(t *testing.T) {
		var event ConnectionEvent
		assert.Empty(t, event.Type)
		assert.Empty(t, event.ID)
		assert.Nil(t, event.UserID)
		assert.Nil(t, event.Email)
		assert.Nil(t, event.DisplayName)
		assert.Empty(t, event.RemoteAddr)
		assert.Empty(t, event.ConnectedAt)
		assert.Empty(t, event.Timestamp)
	})

	t.Run("connected event with all fields", func(t *testing.T) {
		userID := "user-123"
		email := "user@example.com"
		displayName := "John Doe"

		event := ConnectionEvent{
			Type:        ConnectionEventConnected,
			ID:          "conn-456",
			UserID:      &userID,
			Email:       &email,
			DisplayName: &displayName,
			RemoteAddr:  "192.168.1.100:5432",
			ConnectedAt: "2024-06-15T10:30:00Z",
			Timestamp:   "2024-06-15T10:30:01Z",
		}

		assert.Equal(t, ConnectionEventConnected, event.Type)
		assert.Equal(t, "conn-456", event.ID)
		assert.Equal(t, "user-123", *event.UserID)
		assert.Equal(t, "user@example.com", *event.Email)
		assert.Equal(t, "John Doe", *event.DisplayName)
		assert.Equal(t, "192.168.1.100:5432", event.RemoteAddr)
	})

	t.Run("disconnected event", func(t *testing.T) {
		event := ConnectionEvent{
			Type:        ConnectionEventDisconnected,
			ID:          "conn-789",
			RemoteAddr:  "10.0.0.1:8080",
			ConnectedAt: "2024-06-15T09:00:00Z",
			Timestamp:   "2024-06-15T10:00:00Z",
		}

		assert.Equal(t, ConnectionEventDisconnected, event.Type)
		assert.Nil(t, event.UserID) // Anonymous connection
	})

	t.Run("anonymous connection event", func(t *testing.T) {
		event := ConnectionEvent{
			Type:        ConnectionEventConnected,
			ID:          "anon-conn",
			RemoteAddr:  "172.16.0.1:9090",
			ConnectedAt: "2024-06-15T11:00:00Z",
			Timestamp:   "2024-06-15T11:00:00Z",
		}

		assert.Nil(t, event.UserID)
		assert.Nil(t, event.Email)
		assert.Nil(t, event.DisplayName)
	})
}

func TestConnectionEvent_JSONSerialization(t *testing.T) {
	t.Run("serializes all fields correctly", func(t *testing.T) {
		userID := "user-123"
		email := "user@example.com"
		displayName := "John Doe"

		event := ConnectionEvent{
			Type:        ConnectionEventConnected,
			ID:          "conn-456",
			UserID:      &userID,
			Email:       &email,
			DisplayName: &displayName,
			RemoteAddr:  "192.168.1.100:5432",
			ConnectedAt: "2024-06-15T10:30:00Z",
			Timestamp:   "2024-06-15T10:30:01Z",
		}

		data, err := json.Marshal(event)
		assert.NoError(t, err)

		var parsed map[string]interface{}
		err = json.Unmarshal(data, &parsed)
		assert.NoError(t, err)

		assert.Equal(t, "connected", parsed["type"])
		assert.Equal(t, "conn-456", parsed["id"])
		assert.Equal(t, "user-123", parsed["user_id"])
		assert.Equal(t, "user@example.com", parsed["email"])
		assert.Equal(t, "John Doe", parsed["display_name"])
		assert.Equal(t, "192.168.1.100:5432", parsed["remote_addr"])
	})

	t.Run("omits display_name when nil", func(t *testing.T) {
		event := ConnectionEvent{
			Type:        ConnectionEventConnected,
			ID:          "conn-456",
			RemoteAddr:  "192.168.1.100:5432",
			ConnectedAt: "2024-06-15T10:30:00Z",
			Timestamp:   "2024-06-15T10:30:01Z",
		}

		data, err := json.Marshal(event)
		assert.NoError(t, err)

		var parsed map[string]interface{}
		err = json.Unmarshal(data, &parsed)
		assert.NoError(t, err)

		_, hasDisplayName := parsed["display_name"]
		assert.False(t, hasDisplayName, "display_name should be omitted when nil")
	})

	t.Run("deserializes correctly", func(t *testing.T) {
		jsonData := `{
			"type": "disconnected",
			"id": "conn-123",
			"user_id": "user-456",
			"email": "test@test.com",
			"remote_addr": "10.0.0.1:1234",
			"connected_at": "2024-01-01T00:00:00Z",
			"timestamp": "2024-01-01T01:00:00Z"
		}`

		var event ConnectionEvent
		err := json.Unmarshal([]byte(jsonData), &event)
		assert.NoError(t, err)

		assert.Equal(t, ConnectionEventDisconnected, event.Type)
		assert.Equal(t, "conn-123", event.ID)
		assert.Equal(t, "user-456", *event.UserID)
		assert.Equal(t, "test@test.com", *event.Email)
	})
}

func TestConnectionEvent_ToServerMessage(t *testing.T) {
	t.Run("converts connected event to server message", func(t *testing.T) {
		userID := "user-123"
		event := ConnectionEvent{
			Type:        ConnectionEventConnected,
			ID:          "conn-456",
			UserID:      &userID,
			RemoteAddr:  "192.168.1.100:5432",
			ConnectedAt: "2024-06-15T10:30:00Z",
			Timestamp:   "2024-06-15T10:30:01Z",
		}

		msg := event.ToServerMessage()

		assert.Equal(t, MessageTypeBroadcast, msg.Type)
		assert.Equal(t, AdminConnectionsChannel, msg.Channel)
		assert.NotNil(t, msg.Payload)

		// Check payload structure
		payload, ok := msg.Payload.(map[string]interface{})
		assert.True(t, ok)

		broadcast, ok := payload["broadcast"].(map[string]interface{})
		assert.True(t, ok)

		assert.Equal(t, "connected", broadcast["event"])
		assert.Equal(t, event, broadcast["payload"])
	})

	t.Run("converts disconnected event to server message", func(t *testing.T) {
		event := ConnectionEvent{
			Type:        ConnectionEventDisconnected,
			ID:          "conn-789",
			RemoteAddr:  "10.0.0.1:8080",
			ConnectedAt: "2024-06-15T09:00:00Z",
			Timestamp:   "2024-06-15T10:00:00Z",
		}

		msg := event.ToServerMessage()

		assert.Equal(t, MessageTypeBroadcast, msg.Type)
		assert.Equal(t, AdminConnectionsChannel, msg.Channel)

		payload, ok := msg.Payload.(map[string]interface{})
		assert.True(t, ok)

		broadcast, ok := payload["broadcast"].(map[string]interface{})
		assert.True(t, ok)

		assert.Equal(t, "disconnected", broadcast["event"])
	})

	t.Run("server message can be serialized to JSON", func(t *testing.T) {
		event := ConnectionEvent{
			Type:        ConnectionEventConnected,
			ID:          "conn-123",
			RemoteAddr:  "127.0.0.1:5555",
			ConnectedAt: "2024-06-15T10:00:00Z",
			Timestamp:   "2024-06-15T10:00:00Z",
		}

		msg := event.ToServerMessage()

		data, err := json.Marshal(msg)
		assert.NoError(t, err)
		assert.Contains(t, string(data), "broadcast")
		assert.Contains(t, string(data), "connected")
	})
}

// =============================================================================
// MessageType Tests
// =============================================================================

func TestMessageType_Constants(t *testing.T) {
	tests := []struct {
		name     string
		msgType  MessageType
		expected string
	}{
		{"subscribe", MessageTypeSubscribe, "subscribe"},
		{"unsubscribe", MessageTypeUnsubscribe, "unsubscribe"},
		{"heartbeat", MessageTypeHeartbeat, "heartbeat"},
		{"broadcast", MessageTypeBroadcast, "broadcast"},
		{"presence", MessageTypePresence, "presence"},
		{"error", MessageTypeError, "error"},
		{"ack", MessageTypeAck, "ack"},
		{"postgres_changes", MessageTypeChange, "postgres_changes"},
		{"access_token", MessageTypeAccessToken, "access_token"},
		{"subscribe_logs", MessageTypeSubscribeLogs, "subscribe_logs"},
		{"execution_log", MessageTypeExecutionLog, "execution_log"},
		{"subscribe_all_logs", MessageTypeSubscribeAllLogs, "subscribe_all_logs"},
		{"log_entry", MessageTypeLogEntry, "log_entry"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.msgType))
		})
	}

	t.Run("all message types are distinct", func(t *testing.T) {
		types := []MessageType{
			MessageTypeSubscribe,
			MessageTypeUnsubscribe,
			MessageTypeHeartbeat,
			MessageTypeBroadcast,
			MessageTypePresence,
			MessageTypeError,
			MessageTypeAck,
			MessageTypeChange,
			MessageTypeAccessToken,
			MessageTypeSubscribeLogs,
			MessageTypeExecutionLog,
			MessageTypeSubscribeAllLogs,
			MessageTypeLogEntry,
		}

		seen := make(map[MessageType]bool)
		for _, mt := range types {
			assert.False(t, seen[mt], "duplicate message type: %s", mt)
			seen[mt] = true
		}
	})
}

// =============================================================================
// ClientMessage Tests
// =============================================================================

func TestClientMessage_Struct(t *testing.T) {
	t.Run("zero value", func(t *testing.T) {
		var msg ClientMessage
		assert.Empty(t, msg.Type)
		assert.Empty(t, msg.Channel)
		assert.Empty(t, msg.Event)
		assert.Empty(t, msg.Schema)
		assert.Empty(t, msg.Table)
		assert.Empty(t, msg.Filter)
		assert.Nil(t, msg.Payload)
		assert.Nil(t, msg.Config)
		assert.Empty(t, msg.SubscriptionID)
		assert.Empty(t, msg.MessageID)
		assert.Empty(t, msg.Token)
	})

	t.Run("subscribe message", func(t *testing.T) {
		msg := ClientMessage{
			Type:    MessageTypeSubscribe,
			Channel: "public:users",
			Event:   "INSERT",
			Schema:  "public",
			Table:   "users",
		}

		assert.Equal(t, MessageTypeSubscribe, msg.Type)
		assert.Equal(t, "public:users", msg.Channel)
		assert.Equal(t, "INSERT", msg.Event)
		assert.Equal(t, "public", msg.Schema)
		assert.Equal(t, "users", msg.Table)
	})

	t.Run("broadcast message with payload", func(t *testing.T) {
		payload := json.RawMessage(`{"message": "hello"}`)
		msg := ClientMessage{
			Type:      MessageTypeBroadcast,
			Channel:   "room:123",
			Event:     "chat",
			Payload:   payload,
			MessageID: "msg-456",
		}

		assert.Equal(t, MessageTypeBroadcast, msg.Type)
		assert.Equal(t, "room:123", msg.Channel)
		assert.Equal(t, "msg-456", msg.MessageID)
	})

	t.Run("access token message", func(t *testing.T) {
		msg := ClientMessage{
			Type:  MessageTypeAccessToken,
			Token: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
		}

		assert.Equal(t, MessageTypeAccessToken, msg.Type)
		assert.NotEmpty(t, msg.Token)
	})
}

func TestClientMessage_JSONSerialization(t *testing.T) {
	t.Run("deserializes subscribe message", func(t *testing.T) {
		jsonData := `{
			"type": "subscribe",
			"channel": "public:orders",
			"event": "*",
			"schema": "public",
			"table": "orders",
			"filter": "status=eq.pending"
		}`

		var msg ClientMessage
		err := json.Unmarshal([]byte(jsonData), &msg)
		assert.NoError(t, err)

		assert.Equal(t, MessageTypeSubscribe, msg.Type)
		assert.Equal(t, "public:orders", msg.Channel)
		assert.Equal(t, "*", msg.Event)
		assert.Equal(t, "public", msg.Schema)
		assert.Equal(t, "orders", msg.Table)
		assert.Equal(t, "status=eq.pending", msg.Filter)
	})

	t.Run("serializes correctly", func(t *testing.T) {
		msg := ClientMessage{
			Type:    MessageTypeUnsubscribe,
			Channel: "public:users",
		}

		data, err := json.Marshal(msg)
		assert.NoError(t, err)

		var parsed map[string]interface{}
		err = json.Unmarshal(data, &parsed)
		assert.NoError(t, err)

		assert.Equal(t, "unsubscribe", parsed["type"])
		assert.Equal(t, "public:users", parsed["channel"])
	})
}

// =============================================================================
// ServerMessage Tests
// =============================================================================

func TestServerMessage_Struct(t *testing.T) {
	t.Run("zero value", func(t *testing.T) {
		var msg ServerMessage
		assert.Empty(t, msg.Type)
		assert.Empty(t, msg.Channel)
		assert.Nil(t, msg.Payload)
		assert.Empty(t, msg.Error)
	})

	t.Run("success message", func(t *testing.T) {
		msg := ServerMessage{
			Type:    MessageTypeAck,
			Channel: "public:users",
			Payload: map[string]interface{}{"status": "ok"},
		}

		assert.Equal(t, MessageTypeAck, msg.Type)
		assert.NotNil(t, msg.Payload)
		assert.Empty(t, msg.Error)
	})

	t.Run("error message", func(t *testing.T) {
		msg := ServerMessage{
			Type:  MessageTypeError,
			Error: "Unauthorized access",
		}

		assert.Equal(t, MessageTypeError, msg.Type)
		assert.Equal(t, "Unauthorized access", msg.Error)
	})

	t.Run("broadcast message", func(t *testing.T) {
		msg := ServerMessage{
			Type:    MessageTypeBroadcast,
			Channel: "room:123",
			Payload: map[string]interface{}{
				"event":   "user_joined",
				"user_id": "user-456",
			},
		}

		assert.Equal(t, MessageTypeBroadcast, msg.Type)
		assert.Equal(t, "room:123", msg.Channel)

		payload, ok := msg.Payload.(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, "user_joined", payload["event"])
	})
}

// =============================================================================
// Config Struct Tests
// =============================================================================

func TestLogSubscriptionConfig_Struct(t *testing.T) {
	t.Run("zero value", func(t *testing.T) {
		var config LogSubscriptionConfig
		assert.Empty(t, config.ExecutionID)
		assert.Empty(t, config.Type)
	})

	t.Run("function log subscription", func(t *testing.T) {
		config := LogSubscriptionConfig{
			ExecutionID: "exec-123",
			Type:        "function",
		}

		assert.Equal(t, "exec-123", config.ExecutionID)
		assert.Equal(t, "function", config.Type)
	})

	t.Run("job log subscription", func(t *testing.T) {
		config := LogSubscriptionConfig{
			ExecutionID: "job-456",
			Type:        "job",
		}

		assert.Equal(t, "job-456", config.ExecutionID)
		assert.Equal(t, "job", config.Type)
	})

	t.Run("rpc log subscription", func(t *testing.T) {
		config := LogSubscriptionConfig{
			ExecutionID: "rpc-789",
			Type:        "rpc",
		}

		assert.Equal(t, "rpc-789", config.ExecutionID)
		assert.Equal(t, "rpc", config.Type)
	})

	t.Run("JSON serialization", func(t *testing.T) {
		config := LogSubscriptionConfig{
			ExecutionID: "exec-123",
			Type:        "function",
		}

		data, err := json.Marshal(config)
		assert.NoError(t, err)
		assert.Contains(t, string(data), `"execution_id":"exec-123"`)
		assert.Contains(t, string(data), `"type":"function"`)
	})
}

func TestPostgresChangesConfig_Struct(t *testing.T) {
	t.Run("zero value", func(t *testing.T) {
		var config PostgresChangesConfig
		assert.Empty(t, config.Event)
		assert.Empty(t, config.Schema)
		assert.Empty(t, config.Table)
		assert.Empty(t, config.Filter)
	})

	t.Run("insert subscription", func(t *testing.T) {
		config := PostgresChangesConfig{
			Event:  "INSERT",
			Schema: "public",
			Table:  "users",
		}

		assert.Equal(t, "INSERT", config.Event)
		assert.Equal(t, "public", config.Schema)
		assert.Equal(t, "users", config.Table)
		assert.Empty(t, config.Filter)
	})

	t.Run("wildcard subscription with filter", func(t *testing.T) {
		config := PostgresChangesConfig{
			Event:  "*",
			Schema: "public",
			Table:  "orders",
			Filter: "user_id=eq.123",
		}

		assert.Equal(t, "*", config.Event)
		assert.Equal(t, "user_id=eq.123", config.Filter)
	})

	t.Run("JSON serialization", func(t *testing.T) {
		config := PostgresChangesConfig{
			Event:  "UPDATE",
			Schema: "public",
			Table:  "products",
			Filter: "price=gt.100",
		}

		data, err := json.Marshal(config)
		assert.NoError(t, err)
		assert.Contains(t, string(data), `"event":"UPDATE"`)
		assert.Contains(t, string(data), `"schema":"public"`)
		assert.Contains(t, string(data), `"table":"products"`)
		assert.Contains(t, string(data), `"filter":"price=gt.100"`)
	})

	t.Run("filter is omitted when empty", func(t *testing.T) {
		config := PostgresChangesConfig{
			Event:  "DELETE",
			Schema: "public",
			Table:  "logs",
		}

		data, err := json.Marshal(config)
		assert.NoError(t, err)
		// Filter should be omitted due to omitempty
		assert.NotContains(t, string(data), `"filter"`)
	})
}

// =============================================================================
// TokenClaims Tests
// =============================================================================

func TestTokenClaims_Struct(t *testing.T) {
	t.Run("zero value", func(t *testing.T) {
		var claims TokenClaims
		assert.Empty(t, claims.UserID)
		assert.Empty(t, claims.Email)
		assert.Empty(t, claims.Role)
		assert.Empty(t, claims.SessionID)
		assert.Nil(t, claims.RawClaims)
	})

	t.Run("authenticated user claims", func(t *testing.T) {
		claims := TokenClaims{
			UserID:    "user-123",
			Email:     "user@example.com",
			Role:      "authenticated",
			SessionID: "sess-456",
			RawClaims: map[string]interface{}{
				"sub":   "user-123",
				"email": "user@example.com",
				"role":  "authenticated",
			},
		}

		assert.Equal(t, "user-123", claims.UserID)
		assert.Equal(t, "user@example.com", claims.Email)
		assert.Equal(t, "authenticated", claims.Role)
		assert.Equal(t, "sess-456", claims.SessionID)
		assert.Equal(t, "user-123", claims.RawClaims["sub"])
	})

	t.Run("admin user claims", func(t *testing.T) {
		claims := TokenClaims{
			UserID: "admin-789",
			Email:  "admin@example.com",
			Role:   "admin",
		}

		assert.Equal(t, "admin", claims.Role)
	})

	t.Run("service role claims", func(t *testing.T) {
		claims := TokenClaims{
			Role: "service_role",
		}

		assert.Equal(t, "service_role", claims.Role)
		assert.Empty(t, claims.UserID) // Service role may not have user ID
	})

	t.Run("custom claims for RLS", func(t *testing.T) {
		claims := TokenClaims{
			UserID: "player-123",
			Role:   "authenticated",
			RawClaims: map[string]interface{}{
				"sub":        "player-123",
				"role":       "authenticated",
				"meeting_id": "meeting-456",
				"player_id":  "player-123",
				"team":       "blue",
			},
		}

		assert.Equal(t, "meeting-456", claims.RawClaims["meeting_id"])
		assert.Equal(t, "blue", claims.RawClaims["team"])
	})
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkConnectionEvent_ToServerMessage(b *testing.B) {
	userID := "user-123"
	event := ConnectionEvent{
		Type:        ConnectionEventConnected,
		ID:          "conn-456",
		UserID:      &userID,
		RemoteAddr:  "192.168.1.100:5432",
		ConnectedAt: time.Now().Format(time.RFC3339),
		Timestamp:   time.Now().Format(time.RFC3339),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = event.ToServerMessage()
	}
}

func BenchmarkConnectionEvent_JSONMarshal(b *testing.B) {
	userID := "user-123"
	event := ConnectionEvent{
		Type:        ConnectionEventConnected,
		ID:          "conn-456",
		UserID:      &userID,
		RemoteAddr:  "192.168.1.100:5432",
		ConnectedAt: time.Now().Format(time.RFC3339),
		Timestamp:   time.Now().Format(time.RFC3339),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(event)
	}
}

func BenchmarkClientMessage_JSONUnmarshal(b *testing.B) {
	data := []byte(`{"type":"subscribe","channel":"public:users","event":"*","schema":"public","table":"users"}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var msg ClientMessage
		_ = json.Unmarshal(data, &msg)
	}
}

func BenchmarkServerMessage_JSONMarshal(b *testing.B) {
	msg := ServerMessage{
		Type:    MessageTypeBroadcast,
		Channel: "room:123",
		Payload: map[string]interface{}{"event": "message", "data": "hello"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(msg)
	}
}
