package realtime

import (
	"sync"
	"time"

	"github.com/gofiber/contrib/websocket"
	"github.com/rs/zerolog/log"
)

// Connection represents a WebSocket client connection
type Connection struct {
	ID            string
	Conn          *websocket.Conn
	Subscriptions map[string]bool // channel -> subscribed
	UserID        *string         // Authenticated user ID (nil if anonymous)
	Role          string          // User role (e.g., "authenticated", "anon", "dashboard_admin")
	ConnectedAt   time.Time       // Connection timestamp
	mu            sync.RWMutex
}

// NewConnection creates a new WebSocket connection
func NewConnection(id string, conn *websocket.Conn, userID *string, role string) *Connection {
	return &Connection{
		ID:            id,
		Conn:          conn,
		Subscriptions: make(map[string]bool),
		UserID:        userID,
		Role:          role,
		ConnectedAt:   time.Now(),
	}
}

// Subscribe adds a channel subscription for this connection
func (c *Connection) Subscribe(channel string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Subscriptions[channel] = true
	log.Info().
		Str("connection_id", c.ID).
		Str("channel", channel).
		Msg("Subscribed to channel")
}

// Unsubscribe removes a channel subscription for this connection
func (c *Connection) Unsubscribe(channel string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.Subscriptions, channel)
	log.Info().
		Str("connection_id", c.ID).
		Str("channel", channel).
		Msg("Unsubscribed from channel")
}

// IsSubscribed checks if the connection is subscribed to a channel
func (c *Connection) IsSubscribed(channel string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Subscriptions[channel]
}

// SendMessage sends a message to the WebSocket client
func (c *Connection) SendMessage(msg interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.Conn.WriteJSON(msg)
}

// Close closes the WebSocket connection
func (c *Connection) Close() error {
	return c.Conn.Close()
}

// UpdateAuth updates the connection's authentication context
func (c *Connection) UpdateAuth(userID *string, role string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.UserID = userID
	c.Role = role
	log.Info().
		Str("connection_id", c.ID).
		Str("role", role).
		Msg("Updated connection auth")
}
