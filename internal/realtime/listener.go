package realtime

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

// ChangeEvent represents a database change event
type ChangeEvent struct {
	Type      string                 `json:"type"`      // INSERT, UPDATE, DELETE
	Table     string                 `json:"table"`     // Table name
	Schema    string                 `json:"schema"`    // Schema name
	Record    map[string]interface{} `json:"record"`    // New record data
	OldRecord map[string]interface{} `json:"old_record,omitempty"` // Old record data (for UPDATE/DELETE)
}

// Listener handles PostgreSQL LISTEN/NOTIFY
type Listener struct {
	pool    *pgxpool.Pool
	handler *RealtimeHandler
	ctx     context.Context
	cancel  context.CancelFunc
}

// NewListener creates a new PostgreSQL listener
func NewListener(pool *pgxpool.Pool, handler *RealtimeHandler) *Listener {
	ctx, cancel := context.WithCancel(context.Background())
	return &Listener{
		pool:    pool,
		handler: handler,
		ctx:     ctx,
		cancel:  cancel,
	}
}

// Start begins listening for PostgreSQL notifications
func (l *Listener) Start() error {
	// Start listening loop in a goroutine
	go l.listen()

	log.Info().Msg("PostgreSQL LISTEN started on channel: fluxbase_changes")

	return nil
}

// listen processes incoming PostgreSQL notifications
func (l *Listener) listen() {
	// Get a dedicated connection for LISTEN
	conn, err := l.pool.Acquire(l.ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to acquire connection for LISTEN")
		return
	}
	defer conn.Release()

	// Listen on the realtime channel
	_, err = conn.Exec(l.ctx, "LISTEN fluxbase_changes")
	if err != nil {
		log.Error().Err(err).Msg("Failed to execute LISTEN")
		return
	}

	log.Debug().Msg("LISTEN command executed successfully")

	// Listen for notifications
	for {
		select {
		case <-l.ctx.Done():
			log.Info().Msg("Stopping PostgreSQL listener")
			return

		default:
			// Wait for notification with timeout
			ctx, cancel := context.WithTimeout(l.ctx, 5*time.Second)
			notification, err := conn.Conn().WaitForNotification(ctx)
			cancel()

			if err != nil {
				// Check if context was cancelled
				if l.ctx.Err() != nil {
					return
				}

				// Timeout is expected, continue
				if err == context.DeadlineExceeded {
					continue
				}

				// Check if it's a context error
				if ctx.Err() == context.DeadlineExceeded {
					continue
				}

				log.Error().Err(err).Msg("Error waiting for notification")
				time.Sleep(1 * time.Second)
				continue
			}

			// Process notification
			l.processNotification(notification)
		}
	}
}

// processNotification handles a PostgreSQL notification
func (l *Listener) processNotification(notification *pgconn.Notification) {
	log.Debug().
		Str("channel", notification.Channel).
		Str("payload", notification.Payload).
		Msg("Received notification")

	// Parse the notification payload
	var event ChangeEvent
	if err := json.Unmarshal([]byte(notification.Payload), &event); err != nil {
		log.Error().Err(err).Str("payload", notification.Payload).Msg("Failed to parse notification")
		return
	}

	// Determine the broadcast channel based on the table
	channel := fmt.Sprintf("table:%s.%s", event.Schema, event.Table)

	// Broadcast to all subscribers
	l.handler.Broadcast(channel, event)

	log.Debug().
		Str("channel", channel).
		Str("type", event.Type).
		Str("table", event.Table).
		Msg("Broadcasted change event")
}

// Stop stops the listener
func (l *Listener) Stop() {
	l.cancel()
}
