package realtime

import (
	"context"
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

// DefaultMessageQueueSize is the default size of the per-connection message queue
const DefaultMessageQueueSize = 256

// ErrSlowClient is returned when a client is too slow to receive messages
var ErrSlowClient = errors.New("client is too slow to receive messages")

// ErrQueueFull is returned when the message queue is full
var ErrQueueFull = errors.New("message queue is full")

// ErrConnectionClosed is returned when trying to send to a closed connection
var ErrConnectionClosed = errors.New("connection is closed")

// Connection represents a WebSocket client connection
type Connection struct {
	ID              string
	Conn            *websocket.Conn
	Subscriptions   map[string]bool        // channel -> subscribed
	UserID          *string                // Authenticated user ID (nil if anonymous)
	Role            string                 // User role (e.g., "authenticated", "anon", "dashboard_admin")
	Claims          map[string]interface{} // Full JWT claims for RLS (includes custom claims like meeting_id, player_id)
	ConnectedAt     time.Time              // Connection timestamp
	mu              sync.RWMutex
	slowClientCount atomic.Int32 // Count of slow client warnings
	lastSlowWarning time.Time    // Time of last slow client warning
	slowWarningMu   sync.Mutex   // Mutex for lastSlowWarning

	// Async message queue
	sendCh  chan interface{} // Message queue for async sending
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	closed  atomic.Bool
	useSync bool // If true, use synchronous sending (for backward compatibility in tests)

	// Metrics
	messagesSent    atomic.Uint64
	messagesDropped atomic.Uint64
	queueHighWater  atomic.Int32 // Highest queue length seen
}

// NewConnection creates a new WebSocket connection with async message queue
func NewConnection(id string, conn *websocket.Conn, userID *string, role string, claims map[string]interface{}) *Connection {
	return NewConnectionWithQueueSize(id, conn, userID, role, claims, DefaultMessageQueueSize)
}

// NewConnectionWithQueueSize creates a new WebSocket connection with custom queue size
func NewConnectionWithQueueSize(id string, conn *websocket.Conn, userID *string, role string, claims map[string]interface{}, queueSize int) *Connection {
	if queueSize <= 0 {
		queueSize = DefaultMessageQueueSize
	}

	ctx, cancel := context.WithCancel(context.Background())

	c := &Connection{
		ID:            id,
		Conn:          conn,
		Subscriptions: make(map[string]bool),
		UserID:        userID,
		Role:          role,
		Claims:        claims,
		ConnectedAt:   time.Now(),
		sendCh:        make(chan interface{}, queueSize),
		ctx:           ctx,
		cancel:        cancel,
	}

	// Start the async writer goroutine
	c.wg.Add(1)
	go c.writerLoop()

	return c
}

// NewConnectionSync creates a new connection with synchronous sending (for tests)
func NewConnectionSync(id string, conn *websocket.Conn, userID *string, role string, claims map[string]interface{}) *Connection {
	return &Connection{
		ID:            id,
		Conn:          conn,
		Subscriptions: make(map[string]bool),
		UserID:        userID,
		Role:          role,
		Claims:        claims,
		ConnectedAt:   time.Now(),
		useSync:       true,
	}
}

// writerLoop drains the message queue and sends messages to the WebSocket
func (c *Connection) writerLoop() {
	defer c.wg.Done()

	for {
		select {
		case <-c.ctx.Done():
			// Drain remaining messages before exit
			for {
				select {
				case msg := <-c.sendCh:
					_ = c.writeMessage(msg)
				default:
					return
				}
			}
		case msg, ok := <-c.sendCh:
			if !ok {
				return
			}

			// Track queue depth metrics
			queueLen := int32(len(c.sendCh))
			if highWater := c.queueHighWater.Load(); queueLen > highWater {
				c.queueHighWater.Store(queueLen)
			}

			if err := c.writeMessage(msg); err != nil {
				// Log error but continue - the connection will be cleaned up
				// by the reader loop if the WebSocket is truly dead
				log.Debug().
					Err(err).
					Str("connection_id", c.ID).
					Msg("Failed to write message in async writer")
			}
		}
	}
}

// writeMessage performs the actual WebSocket write with timeout
func (c *Connection) writeMessage(msg interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.Conn == nil {
		return ErrConnectionClosed
	}

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

		return err
	}

	// Reset slow client count on successful send
	c.slowClientCount.Store(0)
	c.messagesSent.Add(1)

	return nil
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

// SendMessage queues a message to be sent to the WebSocket client asynchronously.
// Returns ErrQueueFull if the message queue is full (client is too slow).
// Returns ErrSlowClient if the client has been consistently too slow.
// Returns ErrConnectionClosed if the connection has been closed.
func (c *Connection) SendMessage(msg interface{}) error {
	// Check if connection is closed
	if c.closed.Load() {
		return ErrConnectionClosed
	}

	// Check if client is marked as slow
	if c.slowClientCount.Load() >= MaxSlowClientWarnings {
		return ErrSlowClient
	}

	// For backward compatibility in tests, use sync sending if no queue
	if c.useSync {
		return c.writeMessage(msg)
	}

	// Try to queue the message (non-blocking)
	select {
	case c.sendCh <- msg:
		return nil
	default:
		// Queue is full - client is too slow
		c.messagesDropped.Add(1)

		// Track slow client warnings
		c.slowWarningMu.Lock()
		count := c.slowClientCount.Add(1)
		shouldLog := time.Since(c.lastSlowWarning) > time.Minute
		if shouldLog {
			c.lastSlowWarning = time.Now()
		}
		c.slowWarningMu.Unlock()

		if shouldLog {
			log.Warn().
				Str("connection_id", c.ID).
				Int32("slow_count", count).
				Uint64("dropped", c.messagesDropped.Load()).
				Msg("Message queue full - slow client detected")
		}

		if count >= MaxSlowClientWarnings {
			return ErrSlowClient
		}

		return ErrQueueFull
	}
}

// IsSlowClient returns true if this connection has been marked as a slow client
func (c *Connection) IsSlowClient() bool {
	return c.slowClientCount.Load() >= MaxSlowClientWarnings
}

// Close closes the WebSocket connection and stops the writer goroutine
func (c *Connection) Close() error {
	// Mark as closed to prevent new messages
	if c.closed.Swap(true) {
		// Already closed
		return nil
	}

	// Stop the writer goroutine
	if c.cancel != nil {
		c.cancel()
	}

	// Wait for writer to finish (with timeout)
	done := make(chan struct{})
	go func() {
		c.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Writer finished cleanly
	case <-time.After(5 * time.Second):
		// Writer didn't finish in time, continue anyway
		log.Warn().
			Str("connection_id", c.ID).
			Msg("Writer goroutine did not stop in time during close")
	}

	// Close the WebSocket
	if c.Conn != nil {
		return c.Conn.Close()
	}
	return nil
}

// GetQueueStats returns statistics about the message queue
func (c *Connection) GetQueueStats() ConnectionQueueStats {
	queueLen := 0
	queueCap := 0
	if c.sendCh != nil {
		queueLen = len(c.sendCh)
		queueCap = cap(c.sendCh)
	}

	return ConnectionQueueStats{
		QueueLength:     queueLen,
		QueueCapacity:   queueCap,
		QueueHighWater:  c.queueHighWater.Load(),
		MessagesSent:    c.messagesSent.Load(),
		MessagesDropped: c.messagesDropped.Load(),
		SlowClientCount: c.slowClientCount.Load(),
	}
}

// ConnectionQueueStats contains statistics about a connection's message queue
type ConnectionQueueStats struct {
	QueueLength     int
	QueueCapacity   int
	QueueHighWater  int32
	MessagesSent    uint64
	MessagesDropped uint64
	SlowClientCount int32
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
