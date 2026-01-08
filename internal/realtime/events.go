package realtime

import (
	"time"
)

// ConnectionEventType represents the type of connection event
type ConnectionEventType string

const (
	ConnectionEventConnected    ConnectionEventType = "connected"
	ConnectionEventDisconnected ConnectionEventType = "disconnected"
)

// ConnectionEvent represents a connection lifecycle event
// This is broadcast to the admin channel when connections are established or terminated
type ConnectionEvent struct {
	Type        ConnectionEventType `json:"type"`
	ID          string              `json:"id"`
	UserID      *string             `json:"user_id"`
	Email       *string             `json:"email"`
	DisplayName *string             `json:"display_name,omitempty"`
	RemoteAddr  string              `json:"remote_addr"`
	ConnectedAt string              `json:"connected_at"` // ISO 8601 format
	Timestamp   string              `json:"timestamp"`    // Event timestamp
}

// NewConnectionEvent creates a new connection event
func NewConnectionEvent(eventType ConnectionEventType, conn *Connection, email *string, displayName *string) ConnectionEvent {
	remoteAddr := "unknown"
	if conn.Conn != nil {
		remoteAddr = conn.Conn.RemoteAddr().String()
	}

	return ConnectionEvent{
		Type:        eventType,
		ID:          conn.ID,
		UserID:      conn.UserID,
		Email:       email,
		DisplayName: displayName,
		RemoteAddr:  remoteAddr,
		ConnectedAt: conn.ConnectedAt.Format(time.RFC3339),
		Timestamp:   time.Now().Format(time.RFC3339),
	}
}

// ToServerMessage converts a ConnectionEvent to a ServerMessage for broadcasting
func (e ConnectionEvent) ToServerMessage() ServerMessage {
	// Use broadcast type so the SDK can handle it with the existing broadcast mechanism
	// Format matches the standard broadcast payload structure
	return ServerMessage{
		Type:    MessageTypeBroadcast,
		Channel: AdminConnectionsChannel,
		Payload: map[string]interface{}{
			"broadcast": map[string]interface{}{
				"event":   string(e.Type),
				"payload": e,
			},
		},
	}
}
