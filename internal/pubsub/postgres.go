package pubsub

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

// PostgresPubSub implements PubSub using PostgreSQL LISTEN/NOTIFY.
// This is the default pub/sub backend for multi-instance deployments
// without requiring additional infrastructure.
//
// Performance characteristics:
// - Good for up to ~100 instances and moderate message rates
// - Messages are not persisted - only delivered to currently listening connections
// - Payload size limit: 8000 bytes (PostgreSQL NOTIFY limit)
// - For higher scale, use RedisPubSub with Dragonfly
type PostgresPubSub struct {
	pool        *pgxpool.Pool
	subscribers map[string][]chan Message
	mu          sync.RWMutex
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	started     bool
}

// NewPostgresPubSub creates a new PostgreSQL-backed pub/sub.
func NewPostgresPubSub(pool *pgxpool.Pool) *PostgresPubSub {
	ctx, cancel := context.WithCancel(context.Background())
	return &PostgresPubSub{
		pool:        pool,
		subscribers: make(map[string][]chan Message),
		ctx:         ctx,
		cancel:      cancel,
	}
}

// Start begins listening for notifications on all subscribed channels.
// This must be called after creating the pub/sub instance.
func (p *PostgresPubSub) Start() error {
	p.mu.Lock()
	if p.started {
		p.mu.Unlock()
		return nil
	}
	p.started = true
	p.mu.Unlock()

	p.wg.Add(1)
	go p.listenLoop()

	log.Info().Msg("PostgreSQL pub/sub started")
	return nil
}

// listenLoop handles PostgreSQL LISTEN/NOTIFY
func (p *PostgresPubSub) listenLoop() {
	defer p.wg.Done()

	for {
		// Check if we should stop
		if p.ctx.Err() != nil {
			return
		}

		// Acquire a connection for listening
		conn, err := p.pool.Acquire(p.ctx)
		if err != nil {
			if p.ctx.Err() != nil {
				return
			}
			log.Error().Err(err).Msg("Failed to acquire connection for pub/sub LISTEN")
			time.Sleep(time.Second)
			continue
		}

		// Listen on the pub/sub channels
		channels := []string{BroadcastChannel, PresenceChannel}
		for _, ch := range channels {
			// PostgreSQL channel names can't contain colons, replace with underscore
			pgChannel := sanitizeChannelName(ch)
			if _, err := conn.Exec(p.ctx, fmt.Sprintf("LISTEN %s", pgChannel)); err != nil {
				log.Error().Err(err).Str("channel", ch).Msg("Failed to LISTEN on channel")
			}
		}

		log.Debug().Msg("Listening for pub/sub notifications")

		// Process notifications
		for {
			ctx, cancel := context.WithTimeout(p.ctx, 5*time.Second)
			notification, err := conn.Conn().WaitForNotification(ctx)
			cancel()

			if err != nil {
				// Check if we should stop
				if p.ctx.Err() != nil {
					conn.Release()
					return
				}

				// Timeout is expected
				if ctx.Err() == context.DeadlineExceeded {
					continue
				}

				log.Error().Err(err).Msg("Error waiting for pub/sub notification")
				break // Exit inner loop to reconnect
			}

			// Convert PostgreSQL channel name back to our format
			channel := unsanitizeChannelName(notification.Channel)

			// Deliver message to subscribers
			msg := Message{
				Channel: channel,
				Payload: []byte(notification.Payload),
			}
			p.deliverMessage(msg)
		}

		conn.Release()
		time.Sleep(time.Second) // Brief delay before reconnecting
	}
}

// deliverMessage sends a message to all subscribers of the channel
func (p *PostgresPubSub) deliverMessage(msg Message) {
	p.mu.RLock()
	subs := p.subscribers[msg.Channel]
	p.mu.RUnlock()

	for _, ch := range subs {
		select {
		case ch <- msg:
		default:
			// Channel full, skip this subscriber
			log.Warn().Str("channel", msg.Channel).Msg("Pub/sub subscriber channel full, dropping message")
		}
	}
}

// Publish sends a message to all subscribers of a channel.
func (p *PostgresPubSub) Publish(ctx context.Context, channel string, payload []byte) error {
	// PostgreSQL NOTIFY payload limit is ~8000 bytes
	if len(payload) > 8000 {
		return fmt.Errorf("payload too large for PostgreSQL NOTIFY: %d bytes (max 8000)", len(payload))
	}

	// Sanitize channel name for PostgreSQL
	pgChannel := sanitizeChannelName(channel)

	// Escape the payload for SQL
	payloadJSON, err := json.Marshal(string(payload))
	if err != nil {
		return fmt.Errorf("failed to escape payload: %w", err)
	}

	_, err = p.pool.Exec(ctx, fmt.Sprintf("SELECT pg_notify('%s', %s)", pgChannel, payloadJSON))
	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	return nil
}

// Subscribe returns a channel that receives messages published to the given channel.
func (p *PostgresPubSub) Subscribe(ctx context.Context, channel string) (<-chan Message, error) {
	ch := make(chan Message, 100)

	p.mu.Lock()
	p.subscribers[channel] = append(p.subscribers[channel], ch)
	p.mu.Unlock()

	// Start listener if not already started
	if err := p.Start(); err != nil {
		return nil, err
	}

	// Remove subscription when context is cancelled
	go func() {
		<-ctx.Done()
		p.unsubscribe(channel, ch)
	}()

	return ch, nil
}

// unsubscribe removes a subscriber channel
func (p *PostgresPubSub) unsubscribe(channel string, ch chan Message) {
	p.mu.Lock()
	defer p.mu.Unlock()

	subs := p.subscribers[channel]
	for i, sub := range subs {
		if sub == ch {
			p.subscribers[channel] = append(subs[:i], subs[i+1:]...)
			close(ch)
			break
		}
	}
}

// Close releases all resources and closes all subscriptions.
func (p *PostgresPubSub) Close() error {
	p.cancel()
	p.wg.Wait()

	p.mu.Lock()
	defer p.mu.Unlock()

	for _, subs := range p.subscribers {
		for _, ch := range subs {
			close(ch)
		}
	}
	p.subscribers = make(map[string][]chan Message)

	log.Info().Msg("PostgreSQL pub/sub closed")
	return nil
}

// sanitizeChannelName converts our channel names to PostgreSQL-safe names
func sanitizeChannelName(channel string) string {
	// Replace colons with double underscores
	result := ""
	for _, c := range channel {
		if c == ':' {
			result += "__"
		} else {
			result += string(c)
		}
	}
	return result
}

// unsanitizeChannelName converts PostgreSQL channel names back to our format
func unsanitizeChannelName(pgChannel string) string {
	// Replace double underscores with colons
	result := ""
	for i := 0; i < len(pgChannel); i++ {
		if i < len(pgChannel)-1 && pgChannel[i] == '_' && pgChannel[i+1] == '_' {
			result += ":"
			i++ // Skip the second underscore
		} else {
			result += string(pgChannel[i])
		}
	}
	return result
}
