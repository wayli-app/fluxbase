package realtime

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fluxbase-eu/fluxbase/internal/observability"
	"github.com/fluxbase-eu/fluxbase/internal/pubsub"
	"github.com/gofiber/contrib/websocket"
	"github.com/rs/zerolog/log"
)

// splitHostPort is a wrapper around net.SplitHostPort
func splitHostPort(hostport string) (host, port string, err error) {
	return net.SplitHostPort(hostport)
}

// ErrMaxConnectionsReached is returned when the maximum number of connections is reached
var ErrMaxConnectionsReached = errors.New("maximum number of websocket connections reached")

// ErrMaxUserConnectionsReached is returned when a user has exceeded their connection limit
var ErrMaxUserConnectionsReached = errors.New("maximum number of websocket connections per user reached")

// ErrMaxIPConnectionsReached is returned when an IP has exceeded its connection limit
var ErrMaxIPConnectionsReached = errors.New("maximum number of websocket connections per IP reached")

// Manager manages all WebSocket connections
type Manager struct {
	connections            map[string]*Connection // connection ID -> connection
	userConnections        map[string]int         // user ID -> connection count
	ipConnections          map[string]int         // IP address -> connection count
	connectionUserMap      map[string]string      // connection ID -> user ID (for tracking)
	connectionIPMap        map[string]string      // connection ID -> IP address (for tracking)
	slowClientFirstSeen    map[string]time.Time   // connection ID -> when first marked slow
	mu                     sync.RWMutex
	ctx                    context.Context
	cancel                 context.CancelFunc
	ps                     pubsub.PubSub // For cross-instance broadcasting
	metrics                *observability.Metrics
	maxConnections         int           // Maximum allowed connections (0 = unlimited)
	maxConnectionsPerUser  int           // Maximum connections per user (0 = unlimited)
	maxConnectionsPerIP    int           // Maximum connections per IP for anonymous (0 = unlimited)
	clientMessageQueueSize int           // Size of per-client message queue (0 = default)
	slowClientThreshold    int           // Queue length threshold for slow client (default: 100)
	slowClientTimeout      time.Duration // Duration before disconnecting slow clients (default: 30s)

	// Metrics
	slowClientsDisconnected atomic.Uint64
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

// ManagerConfig holds configuration for the connection manager
type ManagerConfig struct {
	MaxConnections         int           // Maximum total connections (0 = unlimited)
	MaxConnectionsPerUser  int           // Maximum connections per user (0 = unlimited)
	MaxConnectionsPerIP    int           // Maximum connections per IP for anonymous (0 = unlimited)
	ClientMessageQueueSize int           // Size of per-client message queue for async sending (0 = default)
	SlowClientThreshold    int           // Queue length threshold for slow client detection (default: 100)
	SlowClientTimeout      time.Duration // Duration before disconnecting slow clients (default: 30s)
}

// NewManager creates a new connection manager
func NewManager(ctx context.Context) *Manager {
	return NewManagerWithConfig(ctx, ManagerConfig{}) // All defaults (unlimited)
}

// NewManagerWithLimit creates a new connection manager with a connection limit
// Deprecated: Use NewManagerWithConfig for more control
func NewManagerWithLimit(ctx context.Context, maxConnections int) *Manager {
	return NewManagerWithConfig(ctx, ManagerConfig{MaxConnections: maxConnections})
}

// NewManagerWithConfig creates a new connection manager with full configuration
func NewManagerWithConfig(ctx context.Context, config ManagerConfig) *Manager {
	ctx, cancel := context.WithCancel(ctx)

	// Apply defaults for slow client settings
	slowClientThreshold := config.SlowClientThreshold
	if slowClientThreshold <= 0 {
		slowClientThreshold = 100 // Default: 100 pending messages
	}
	slowClientTimeout := config.SlowClientTimeout
	if slowClientTimeout <= 0 {
		slowClientTimeout = 30 * time.Second // Default: 30 seconds
	}

	m := &Manager{
		connections:            make(map[string]*Connection),
		userConnections:        make(map[string]int),
		ipConnections:          make(map[string]int),
		connectionUserMap:      make(map[string]string),
		connectionIPMap:        make(map[string]string),
		slowClientFirstSeen:    make(map[string]time.Time),
		ctx:                    ctx,
		cancel:                 cancel,
		maxConnections:         config.MaxConnections,
		maxConnectionsPerUser:  config.MaxConnectionsPerUser,
		maxConnectionsPerIP:    config.MaxConnectionsPerIP,
		clientMessageQueueSize: config.ClientMessageQueueSize,
		slowClientThreshold:    slowClientThreshold,
		slowClientTimeout:      slowClientTimeout,
	}

	// Start slow client checker goroutine
	go m.slowClientChecker()

	return m
}

// SetMaxConnections sets the maximum number of allowed connections
func (m *Manager) SetMaxConnections(max int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.maxConnections = max
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
// Returns nil and ErrMaxConnectionsReached if the connection limit is exceeded
// Returns nil and ErrMaxUserConnectionsReached if the per-user limit is exceeded
// Returns nil and ErrMaxIPConnectionsReached if the per-IP limit is exceeded (for anonymous)
func (m *Manager) AddConnection(id string, conn *websocket.Conn, userID *string, role string, claims map[string]interface{}) (*Connection, error) {
	// Get remote IP for tracking
	remoteIP := ""
	if conn != nil {
		remoteIP = conn.RemoteAddr().String()
		// Extract just the IP part (remove port)
		if host, _, err := splitHostPort(remoteIP); err == nil {
			remoteIP = host
		}
	}

	return m.AddConnectionWithIP(id, conn, userID, role, claims, remoteIP)
}

// AddConnectionWithIP adds a new WebSocket connection with explicit IP address
// This is useful when the IP is already known (e.g., from X-Forwarded-For header)
func (m *Manager) AddConnectionWithIP(id string, conn *websocket.Conn, userID *string, role string, claims map[string]interface{}, remoteIP string) (*Connection, error) {
	m.mu.Lock()

	// Check global connection limit before adding
	if m.maxConnections > 0 && len(m.connections) >= m.maxConnections {
		m.mu.Unlock()
		log.Warn().
			Int("current_connections", len(m.connections)).
			Int("max_connections", m.maxConnections).
			Str("connection_id", id).
			Msg("Rejecting WebSocket connection: maximum connections reached")
		if m.metrics != nil {
			m.metrics.RecordRealtimeError("max_connections_reached")
		}
		return nil, ErrMaxConnectionsReached
	}

	// Check per-user limit for authenticated users
	if userID != nil && m.maxConnectionsPerUser > 0 {
		currentUserConns := m.userConnections[*userID]
		if currentUserConns >= m.maxConnectionsPerUser {
			m.mu.Unlock()
			log.Warn().
				Str("user_id", *userID).
				Int("current_connections", currentUserConns).
				Int("max_per_user", m.maxConnectionsPerUser).
				Str("connection_id", id).
				Msg("Rejecting WebSocket connection: per-user limit exceeded")
			if m.metrics != nil {
				m.metrics.RecordRealtimeError("max_user_connections_reached")
			}
			return nil, ErrMaxUserConnectionsReached
		}
	}

	// Check per-IP limit for anonymous connections
	if userID == nil && remoteIP != "" && m.maxConnectionsPerIP > 0 {
		currentIPConns := m.ipConnections[remoteIP]
		if currentIPConns >= m.maxConnectionsPerIP {
			m.mu.Unlock()
			log.Warn().
				Str("remote_ip", remoteIP).
				Int("current_connections", currentIPConns).
				Int("max_per_ip", m.maxConnectionsPerIP).
				Str("connection_id", id).
				Msg("Rejecting WebSocket connection: per-IP limit exceeded")
			if m.metrics != nil {
				m.metrics.RecordRealtimeError("max_ip_connections_reached")
			}
			return nil, ErrMaxIPConnectionsReached
		}
	}

	// Create and track the connection with configured queue size
	var connection *Connection
	if m.clientMessageQueueSize > 0 {
		connection = NewConnectionWithQueueSize(id, conn, userID, role, claims, m.clientMessageQueueSize)
	} else {
		connection = NewConnection(id, conn, userID, role, claims)
	}
	m.connections[id] = connection

	// Track per-user connections
	if userID != nil {
		m.userConnections[*userID]++
		m.connectionUserMap[id] = *userID
	}

	// Track per-IP connections for anonymous users
	if userID == nil && remoteIP != "" {
		m.ipConnections[remoteIP]++
		m.connectionIPMap[id] = remoteIP
	}

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
		Str("remote_ip", remoteIP).
		Msg("New WebSocket connection")

	// Broadcast connection event to admin channel
	// Email and DisplayName are nil here since we don't query the database on connect
	// The admin UI should already have this info from the initial stats call
	event := NewConnectionEvent(ConnectionEventConnected, connection, nil, nil)
	m.BroadcastConnectionEvent(event)

	return connection, nil
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

	// Decrement per-user connection count
	if userID, ok := m.connectionUserMap[id]; ok {
		m.userConnections[userID]--
		if m.userConnections[userID] <= 0 {
			delete(m.userConnections, userID)
		}
		delete(m.connectionUserMap, id)
	}

	// Decrement per-IP connection count
	if ip, ok := m.connectionIPMap[id]; ok {
		m.ipConnections[ip]--
		if m.ipConnections[ip] <= 0 {
			delete(m.ipConnections, ip)
		}
		delete(m.connectionIPMap, id)
	}

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

	// Clear all maps while holding the lock
	m.connections = make(map[string]*Connection)
	m.userConnections = make(map[string]int)
	m.ipConnections = make(map[string]int)
	m.connectionUserMap = make(map[string]string)
	m.connectionIPMap = make(map[string]string)

	m.mu.Unlock()

	// Close connections after releasing the lock to avoid deadlock
	for _, conn := range connsToClose {
		_ = conn.Close()
		log.Info().Str("connection_id", conn.ID).Msg("Closed connection during shutdown")
	}
}

// GetUserConnectionCount returns the number of connections for a specific user
func (m *Manager) GetUserConnectionCount(userID string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.userConnections[userID]
}

// GetIPConnectionCount returns the number of connections for a specific IP
func (m *Manager) GetIPConnectionCount(ip string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.ipConnections[ip]
}

// SetConnectionLimits updates the per-user and per-IP connection limits
func (m *Manager) SetConnectionLimits(maxPerUser, maxPerIP int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.maxConnectionsPerUser = maxPerUser
	m.maxConnectionsPerIP = maxPerIP
}

// slowClientChecker periodically checks for slow clients and disconnects them
func (m *Manager) slowClientChecker() {
	// Check every 5 seconds
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.checkAndDisconnectSlowClients()
		}
	}
}

// checkAndDisconnectSlowClients scans connections and disconnects slow ones
func (m *Manager) checkAndDisconnectSlowClients() {
	m.mu.Lock()
	now := time.Now()
	toDisconnect := []string{}

	for id, conn := range m.connections {
		// Get queue stats
		stats := conn.GetQueueStats()
		isSlowNow := stats.QueueLength >= m.slowClientThreshold || conn.IsSlowClient()

		if isSlowNow {
			// Check if we've seen this client as slow before
			firstSeen, exists := m.slowClientFirstSeen[id]
			if !exists {
				// First time seeing this client as slow
				m.slowClientFirstSeen[id] = now
				log.Debug().
					Str("connection_id", id).
					Int("queue_length", stats.QueueLength).
					Int("threshold", m.slowClientThreshold).
					Msg("Client marked as slow")
			} else if now.Sub(firstSeen) >= m.slowClientTimeout {
				// Client has been slow for too long
				toDisconnect = append(toDisconnect, id)
			}
		} else {
			// Client is no longer slow - remove from tracking
			delete(m.slowClientFirstSeen, id)
		}
	}
	m.mu.Unlock()

	// Disconnect slow clients outside the lock
	for _, id := range toDisconnect {
		m.disconnectSlowClient(id)
	}
}

// disconnectSlowClient closes a slow client connection with a proper close frame
func (m *Manager) disconnectSlowClient(id string) {
	m.mu.RLock()
	conn, exists := m.connections[id]
	m.mu.RUnlock()

	if !exists {
		return
	}

	m.slowClientsDisconnected.Add(1)

	log.Warn().
		Str("connection_id", id).
		Msg("Disconnecting slow client - exceeded timeout")

	// Send close frame with 1008 Policy Violation before disconnect
	if conn.Conn != nil {
		// 1008 = Policy Violation
		_ = conn.Conn.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(1008, "Connection too slow"),
			time.Now().Add(time.Second),
		)
	}

	// Close and remove the connection
	conn.Close()
	m.RemoveConnection(id)
}

// GetSlowClientsDisconnected returns the count of slow clients disconnected
func (m *Manager) GetSlowClientsDisconnected() uint64 {
	return m.slowClientsDisconnected.Load()
}
