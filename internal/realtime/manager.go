package realtime

import (
	"context"
	"sync"

	"github.com/gofiber/contrib/websocket"
	"github.com/rs/zerolog/log"
)

// Manager manages all WebSocket connections
type Manager struct {
	connections map[string]*Connection // connection ID -> connection
	mu          sync.RWMutex
	ctx         context.Context
	cancel      context.CancelFunc
}

// NewManager creates a new connection manager
func NewManager(ctx context.Context) *Manager {
	ctx, cancel := context.WithCancel(ctx)
	return &Manager{
		connections: make(map[string]*Connection),
		ctx:         ctx,
		cancel:      cancel,
	}
}

// AddConnection adds a new WebSocket connection
func (m *Manager) AddConnection(id string, conn *websocket.Conn, userID *string, role string) *Connection {
	m.mu.Lock()
	defer m.mu.Unlock()

	connection := NewConnection(id, conn, userID, role)
	m.connections[id] = connection

	log.Info().
		Str("connection_id", id).
		Str("user_id", func() string {
			if userID != nil {
				return *userID
			}
			return "anonymous"
		}()).
		Msg("New WebSocket connection")

	return connection
}

// RemoveConnection removes a WebSocket connection
func (m *Manager) RemoveConnection(id string) {
	m.mu.Lock()

	connection, exists := m.connections[id]
	if !exists {
		m.mu.Unlock()
		return
	}

	delete(m.connections, id)

	// Release manager lock before closing connection
	m.mu.Unlock()

	_ = connection.Close()

	log.Info().
		Str("connection_id", id).
		Msg("WebSocket connection closed")
}

// GetConnectionCount returns the total number of active connections
func (m *Manager) GetConnectionCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.connections)
}

// BroadcastToChannel sends a message to all connections subscribed to a channel
func (m *Manager) BroadcastToChannel(channel string, message ServerMessage) int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sentCount := 0
	for _, conn := range m.connections {
		if conn.IsSubscribed(channel) {
			if err := conn.SendMessage(message); err != nil {
				log.Error().
					Err(err).
					Str("connection_id", conn.ID).
					Str("channel", channel).
					Msg("Failed to broadcast message to connection")
			} else {
				sentCount++
			}
		}
	}

	log.Debug().
		Str("channel", channel).
		Int("recipients", sentCount).
		Msg("Broadcast message sent")

	return sentCount
}

// ConnectionInfo represents detailed information about a connection
type ConnectionInfo struct {
	ID          string  `json:"id"`
	UserID      *string `json:"user_id"`
	RemoteAddr  string  `json:"remote_addr"`
	ConnectedAt string  `json:"connected_at"`
}

// GetDetailedStats returns detailed realtime statistics
func (m *Manager) GetDetailedStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Get connection details
	connections := make([]ConnectionInfo, 0, len(m.connections))
	for _, conn := range m.connections {
		remoteAddr := "unknown"
		if conn.Conn != nil {
			remoteAddr = conn.Conn.RemoteAddr().String()
		}

		connections = append(connections, ConnectionInfo{
			ID:          conn.ID,
			UserID:      conn.UserID,
			RemoteAddr:  remoteAddr,
			ConnectedAt: conn.ConnectedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	return map[string]interface{}{
		"total_connections": len(m.connections),
		"connections":       connections,
	}
}

// Shutdown gracefully shuts down the manager
func (m *Manager) Shutdown() {
	m.cancel()

	m.mu.Lock()

	// Collect all connections to close
	connsToClose := make([]*Connection, 0, len(m.connections))
	for _, conn := range m.connections {
		connsToClose = append(connsToClose, conn)
	}

	// Clear connections map while holding the lock
	m.connections = make(map[string]*Connection)

	m.mu.Unlock()

	// Close connections after releasing the lock to avoid deadlock
	for _, conn := range connsToClose {
		_ = conn.Close()
		log.Info().Str("connection_id", conn.ID).Msg("Closed connection during shutdown")
	}
}
