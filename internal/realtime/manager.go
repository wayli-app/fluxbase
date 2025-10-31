package realtime

import (
	"context"
	"fmt"
	"sync"

	"github.com/gofiber/contrib/websocket"
	"github.com/rs/zerolog/log"
)

// Manager manages all WebSocket connections and subscriptions
type Manager struct {
	connections map[string]*Connection     // connection ID -> connection
	channels    map[string]map[string]bool // channel -> connection IDs
	mu          sync.RWMutex
	ctx         context.Context
	cancel      context.CancelFunc
}

// NewManager creates a new connection manager
func NewManager(ctx context.Context) *Manager {
	ctx, cancel := context.WithCancel(ctx)
	return &Manager{
		connections: make(map[string]*Connection),
		channels:    make(map[string]map[string]bool),
		ctx:         ctx,
		cancel:      cancel,
	}
}

// AddConnection adds a new WebSocket connection
func (m *Manager) AddConnection(id string, conn *websocket.Conn, userID *string) *Connection {
	m.mu.Lock()
	defer m.mu.Unlock()

	connection := NewConnection(id, conn, userID)
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

	// Get a copy of subscription channels while holding the connection lock
	connection.mu.RLock()
	channels := make([]string, 0, len(connection.Subscriptions))
	for channel := range connection.Subscriptions {
		channels = append(channels, channel)
	}
	connection.mu.RUnlock()

	// Remove from all channel subscriptions
	for _, channel := range channels {
		if subscribers, exists := m.channels[channel]; exists {
			delete(subscribers, id)
			if len(subscribers) == 0 {
				delete(m.channels, channel)
			}
		}
	}

	delete(m.connections, id)

	// Release manager lock before closing connection
	m.mu.Unlock()

	connection.Close()

	log.Info().
		Str("connection_id", id).
		Msg("WebSocket connection closed")
}

// Subscribe subscribes a connection to a channel
func (m *Manager) Subscribe(connectionID string, channel string) error {
	m.mu.Lock()

	connection, exists := m.connections[connectionID]
	if !exists {
		m.mu.Unlock()
		log.Warn().Str("connection_id", connectionID).Msg("Connection not found")
		return fmt.Errorf("connection not found")
	}

	// Add to channel subscribers
	if _, exists := m.channels[channel]; !exists {
		m.channels[channel] = make(map[string]bool)
	}
	m.channels[channel][connectionID] = true

	// Release manager lock before acquiring connection lock to avoid deadlock
	m.mu.Unlock()

	// Update connection subscriptions (done after releasing manager lock)
	connection.Subscribe(channel)

	return nil
}

// Unsubscribe unsubscribes a connection from a channel
func (m *Manager) Unsubscribe(connectionID string, channel string) error {
	m.mu.Lock()

	connection, exists := m.connections[connectionID]
	if !exists {
		m.mu.Unlock()
		return fmt.Errorf("connection not found")
	}

	// Remove from channel subscribers
	if subscribers, exists := m.channels[channel]; exists {
		delete(subscribers, connectionID)
		if len(subscribers) == 0 {
			delete(m.channels, channel)
		}
	}

	// Release manager lock before acquiring connection lock to avoid deadlock
	m.mu.Unlock()

	// Update connection subscriptions (done after releasing manager lock)
	connection.Unsubscribe(channel)

	return nil
}

// Broadcast sends a message to all subscribers of a channel
func (m *Manager) Broadcast(channel string, message interface{}) {
	m.mu.RLock()
	subscribers, exists := m.channels[channel]
	if !exists {
		m.mu.RUnlock()
		return
	}

	// Create a copy of subscriber IDs to avoid holding the lock during sends
	subscriberIDs := make([]string, 0, len(subscribers))
	for id := range subscribers {
		subscriberIDs = append(subscriberIDs, id)
	}
	m.mu.RUnlock()

	// Send to each subscriber
	for _, id := range subscriberIDs {
		m.mu.RLock()
		connection, exists := m.connections[id]
		m.mu.RUnlock()

		if exists {
			if err := connection.SendMessage(message); err != nil {
				log.Error().
					Err(err).
					Str("connection_id", id).
					Str("channel", channel).
					Msg("Failed to send message")
				// Remove failed connection
				go m.RemoveConnection(id)
			}
		}
	}

	log.Debug().
		Str("channel", channel).
		Int("subscribers", len(subscriberIDs)).
		Msg("Broadcast message")
}

// GetConnectionCount returns the total number of active connections
func (m *Manager) GetConnectionCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.connections)
}

// GetChannelCount returns the total number of active channels
func (m *Manager) GetChannelCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.channels)
}

// ConnectionInfo represents detailed information about a connection
type ConnectionInfo struct {
	ID            string   `json:"id"`
	UserID        *string  `json:"user_id"`
	Subscriptions []string `json:"subscriptions"`
	RemoteAddr    string   `json:"remote_addr"`
	ConnectedAt   string   `json:"connected_at"`
}

// ChannelInfo represents detailed information about a channel
type ChannelInfo struct {
	Name            string `json:"name"`
	SubscriberCount int    `json:"subscriber_count"`
}

// GetDetailedStats returns detailed realtime statistics
func (m *Manager) GetDetailedStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Get connection details
	connections := make([]ConnectionInfo, 0, len(m.connections))
	for _, conn := range m.connections {
		// Acquire connection lock to safely read subscriptions
		conn.mu.RLock()
		subscriptions := make([]string, 0, len(conn.Subscriptions))
		for channel := range conn.Subscriptions {
			subscriptions = append(subscriptions, channel)
		}
		conn.mu.RUnlock()

		remoteAddr := "unknown"
		if conn.Conn != nil {
			remoteAddr = conn.Conn.RemoteAddr().String()
		}

		connections = append(connections, ConnectionInfo{
			ID:            conn.ID,
			UserID:        conn.UserID,
			Subscriptions: subscriptions,
			RemoteAddr:    remoteAddr,
			ConnectedAt:   conn.ConnectedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	// Get channel details
	channels := make([]ChannelInfo, 0, len(m.channels))
	for channel, subscribers := range m.channels {
		channels = append(channels, ChannelInfo{
			Name:            channel,
			SubscriberCount: len(subscribers),
		})
	}

	return map[string]interface{}{
		"total_connections": len(m.connections),
		"total_channels":    len(m.channels),
		"connections":       connections,
		"channels":          channels,
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

	// Clear maps while holding the lock
	m.connections = make(map[string]*Connection)
	m.channels = make(map[string]map[string]bool)

	m.mu.Unlock()

	// Close connections after releasing the lock to avoid deadlock
	for _, conn := range connsToClose {
		conn.Close()
		log.Info().Str("connection_id", conn.ID).Msg("Closed connection during shutdown")
	}
}
