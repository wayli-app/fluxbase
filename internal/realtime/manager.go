package realtime

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/fluxbase-eu/fluxbase/internal/observability"
	"github.com/fluxbase-eu/fluxbase/internal/pubsub"
	"github.com/gofiber/contrib/websocket"
	"github.com/rs/zerolog/log"
)

// Manager manages all WebSocket connections
type Manager struct {
	connections map[string]*Connection // connection ID -> connection
	mu          sync.RWMutex
	ctx         context.Context
	cancel      context.CancelFunc
	ps          pubsub.PubSub // For cross-instance broadcasting
	metrics     *observability.Metrics
}

// SetMetrics sets the metrics instance for recording realtime metrics
func (m *Manager) SetMetrics(metrics *observability.Metrics) {
	m.metrics = metrics
}

// updateMetrics updates the realtime metrics gauges
func (m *Manager) updateMetrics() {
	if m.metrics == nil {
		return
	}

	m.mu.RLock()
	connections := len(m.connections)
	channels := 0
	subscriptions := 0
	channelSet := make(map[string]struct{})

	for _, conn := range m.connections {
		for ch := range conn.Subscriptions {
			channelSet[ch] = struct{}{}
			subscriptions++
		}
	}
	channels = len(channelSet)
	m.mu.RUnlock()

	m.metrics.UpdateRealtimeStats(connections, channels, subscriptions)
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

// SetPubSub sets the pub/sub backend for cross-instance broadcasting.
// If set, BroadcastGlobal will publish messages to the pub/sub channel
// and this manager will subscribe to receive messages from other instances.
func (m *Manager) SetPubSub(ps pubsub.PubSub) {
	m.ps = ps

	// Subscribe to broadcast channel to receive messages from other instances
	if ps != nil {
		go m.handleGlobalBroadcasts()
	}
}

// handleGlobalBroadcasts listens for broadcast messages from other instances
func (m *Manager) handleGlobalBroadcasts() {
	ch, err := m.ps.Subscribe(m.ctx, pubsub.BroadcastChannel)
	if err != nil {
		log.Error().Err(err).Msg("Failed to subscribe to broadcast channel")
		return
	}

	log.Info().Msg("Subscribed to global broadcast channel for cross-instance messages")

	for {
		select {
		case <-m.ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			m.handleGlobalMessage(msg)
		}
	}
}

// GlobalBroadcast represents a message broadcast across instances
type GlobalBroadcast struct {
	Channel string        `json:"channel"` // The realtime channel to broadcast to
	Message ServerMessage `json:"message"` // The message to send
}

// handleGlobalMessage processes a message received from another instance
func (m *Manager) handleGlobalMessage(msg pubsub.Message) {
	var broadcast GlobalBroadcast
	if err := json.Unmarshal(msg.Payload, &broadcast); err != nil {
		log.Error().Err(err).Msg("Failed to unmarshal global broadcast message")
		return
	}

	// Deliver to local connections subscribed to the channel
	m.BroadcastToChannel(broadcast.Channel, broadcast.Message)
}

// BroadcastGlobal sends a message to all connections across all instances.
// If pub/sub is configured, it publishes to the broadcast channel.
// Otherwise, it only broadcasts to local connections.
func (m *Manager) BroadcastGlobal(channel string, message ServerMessage) error {
	if m.ps == nil {
		// No pub/sub configured - broadcast locally only
		m.BroadcastToChannel(channel, message)
		return nil
	}

	// Publish to pub/sub for cross-instance delivery
	broadcast := GlobalBroadcast{
		Channel: channel,
		Message: message,
	}

	payload, err := json.Marshal(broadcast)
	if err != nil {
		return err
	}

	return m.ps.Publish(m.ctx, pubsub.BroadcastChannel, payload)
}

// AddConnection adds a new WebSocket connection
func (m *Manager) AddConnection(id string, conn *websocket.Conn, userID *string, role string) *Connection {
	m.mu.Lock()
	connection := NewConnection(id, conn, userID, role)
	m.connections[id] = connection
	m.mu.Unlock()

	// Update metrics after releasing lock
	m.updateMetrics()

	log.Info().
		Str("connection_id", id).
		Str("user_id", func() string {
			if userID != nil {
				return *userID
			}
			return "anonymous"
		}()).
		Msg("New WebSocket connection")

	// Broadcast connection event to admin channel
	// Email and DisplayName are nil here since we don't query the database on connect
	// The admin UI should already have this info from the initial stats call
	event := NewConnectionEvent(ConnectionEventConnected, connection, nil, nil)
	m.BroadcastConnectionEvent(event)

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

	// Broadcast disconnection event before closing
	event := NewConnectionEvent(ConnectionEventDisconnected, connection, nil, nil)
	m.BroadcastConnectionEvent(event)

	_ = connection.Close()

	// Update metrics after releasing lock
	m.updateMetrics()

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
				// Record error metric
				if m.metrics != nil {
					m.metrics.RecordRealtimeError("send_failed")
				}
			} else {
				sentCount++
				// Record message sent metric
				if m.metrics != nil {
					m.metrics.RecordRealtimeMessage(string(message.Type))
				}
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
	Email       *string `json:"email"`
	DisplayName *string `json:"display_name,omitempty"`
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

// GetConnectionsForStats returns all connections as ConnectionInfo slice
// This is used by the stats handler which will enrich with emails
func (m *Manager) GetConnectionsForStats() []ConnectionInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

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

	return connections
}

// AdminConnectionsChannel is the channel name for broadcasting connection events to admins
const AdminConnectionsChannel = "realtime:admin:connections"

// BroadcastConnectionEvent broadcasts a connection event to the admin channel
// This allows admins to monitor connection lifecycle in real-time
func (m *Manager) BroadcastConnectionEvent(event ConnectionEvent) {
	message := event.ToServerMessage()

	// Use global broadcast to ensure all instances receive the event
	if err := m.BroadcastGlobal(AdminConnectionsChannel, message); err != nil {
		log.Error().
			Err(err).
			Str("event_type", string(event.Type)).
			Str("connection_id", event.ID).
			Msg("Failed to broadcast connection event")
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
