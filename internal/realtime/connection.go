package realtime

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gofiber/contrib/websocket"
	"github.com/rs/zerolog/log"
)

// WriteTimeout is the maximum time allowed to write a message to a WebSocket client
const WriteTimeout = 10 * time.Second

// MaxSlowClientWarnings is the number of slow client warnings before marking unhealthy
const MaxSlowClientWarnings = 3

// ErrSlowClient is returned when a client is too slow to receive messages
var ErrSlowClient = errors.New("client is too slow to receive messages")

// Connection represents a WebSocket client connection
type Connection struct {
	ID               string
	Conn             *websocket.Conn
	Subscriptions    map[string]bool        // channel -> subscribed
	UserID           *string                // Authenticated user ID (nil if anonymous)
	Role             string                 // User role (e.g., "authenticated", "anon", "dashboard_admin")
	Claims           map[string]interface{} // Full JWT claims for RLS (includes custom claims like meeting_id, player_id)
	ConnectedAt      time.Time              // Connection timestamp
	mu               sync.RWMutex
	slowClientCount  atomic.Int32 // Count of slow client warnings
	lastSlowWarning  time.Time    // Time of last slow client warning
	slowWarningMu    sync.Mutex   // Mutex for lastSlowWarning
}

// NewConnection creates a new WebSocket connection
func NewConnection(id string, conn *websocket.Conn, userID *string, role string, claims map[string]interface{}) *Connection {
	return &Connection{
		ID:            id,
		Conn:          conn,
		Subscriptions: make(map[string]bool),
		UserID:        userID,
		Role:          role,
		Claims:        claims,
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

// SendMessage sends a message to the WebSocket client with timeout protection
// Returns ErrSlowClient if the client is consistently too slow to receive messages
func (c *Connection) SendMessage(msg interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Set write deadline to prevent blocking on slow clients
	if err := c.Conn.SetWriteDeadline(time.Now().Add(WriteTimeout)); err != nil {
		return err
	}

	err := c.Conn.WriteJSON(msg)

	// Reset deadline after write
	_ = c.Conn.SetWriteDeadline(time.Time{})

	if err != nil {
		// Track slow client warnings
		c.slowWarningMu.Lock()
		count := c.slowClientCount.Add(1)

		// Only log warning once per minute to avoid log spam
		shouldLog := time.Since(c.lastSlowWarning) > time.Minute
		if shouldLog {
			c.lastSlowWarning = time.Now()
		}
		c.slowWarningMu.Unlock()

		if shouldLog {
			log.Warn().
				Str("connection_id", c.ID).
				Int32("slow_count", count).
				Err(err).
				Msg("Slow client detected - message write timeout or error")
		}

		// Return specific error if client is consistently slow
		if count >= MaxSlowClientWarnings {
			return ErrSlowClient
		}
	} else {
		// Reset slow client count on successful send
		c.slowClientCount.Store(0)
	}

	return err
}

// IsSlowClient returns true if this connection has been marked as a slow client
func (c *Connection) IsSlowClient() bool {
	return c.slowClientCount.Load() >= MaxSlowClientWarnings
}

// Close closes the WebSocket connection
func (c *Connection) Close() error {
	return c.Conn.Close()
}

// UpdateAuth updates the connection's authentication context
func (c *Connection) UpdateAuth(userID *string, role string, claims map[string]interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.UserID = userID
	c.Role = role
	c.Claims = claims
	log.Info().
		Str("connection_id", c.ID).
		Str("role", role).
		Msg("Updated connection auth")
}
