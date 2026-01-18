package realtime

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/fluxbase-eu/fluxbase/internal/jobs"
	"github.com/fluxbase-eu/fluxbase/internal/pubsub"
	"github.com/fluxbase-eu/fluxbase/internal/storage"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

// ChangeEvent represents a database change event
type ChangeEvent struct {
	Type      string                 `json:"type"`                 // INSERT, UPDATE, DELETE
	Table     string                 `json:"table"`                // Table name
	Schema    string                 `json:"schema"`               // Schema name
	Record    map[string]interface{} `json:"record"`               // New record data
	OldRecord map[string]interface{} `json:"old_record,omitempty"` // Old record data (for UPDATE/DELETE)
}

// LogChannel is the PubSub channel for execution log notifications.
const LogChannel = "fluxbase:logs"

// AllLogsChannel is the PubSub channel for all log notifications (admin streaming).
const AllLogsChannel = "fluxbase:all_logs"

// RealtimeListener is the interface for PostgreSQL LISTEN/NOTIFY handlers.
// Both the simple Listener and the pooled ListenerPool implement this interface.
type RealtimeListener interface {
	Start() error
	Stop()
}

// Listener handles PostgreSQL LISTEN/NOTIFY and PubSub log events
type Listener struct {
	pool       *pgxpool.Pool
	handler    *RealtimeHandler
	subManager *SubscriptionManager
	pubsub     pubsub.PubSub
	ctx        context.Context
	cancel     context.CancelFunc
}

// NewListener creates a new PostgreSQL listener
func NewListener(pool *pgxpool.Pool, handler *RealtimeHandler, subManager *SubscriptionManager, ps pubsub.PubSub) *Listener {
	ctx, cancel := context.WithCancel(context.Background())
	return &Listener{
		pool:       pool,
		handler:    handler,
		subManager: subManager,
		pubsub:     ps,
		ctx:        ctx,
		cancel:     cancel,
	}
}

// Start begins listening for PostgreSQL notifications and PubSub log events
func (l *Listener) Start() error {
	// Start PostgreSQL listening loop in a goroutine
	go l.listen()

	log.Info().Msg("PostgreSQL LISTEN started on channel: fluxbase_changes")

	// Start PubSub log listener if available
	if l.pubsub != nil {
		go l.listenLogsPubSub()
		log.Info().Msg("PubSub log listener started on channel: " + LogChannel)

		// Start all-logs listener for admin dashboard streaming
		go l.listenAllLogsPubSub()
		log.Info().Msg("PubSub all-logs listener started on channel: " + AllLogsChannel)
	}

	return nil
}

// listen processes incoming PostgreSQL notifications
func (l *Listener) listen() {
	// Retry connection acquisition with exponential backoff
	var conn *pgxpool.Conn
	var err error
	maxRetries := 5
	baseDelay := 1 * time.Second

	for attempt := 1; attempt <= maxRetries; attempt++ {
		// Check if context is cancelled before attempting
		if l.ctx.Err() != nil {
			log.Info().Msg("Listener context cancelled, stopping connection attempts")
			return
		}

		// Use a timeout for connection acquisition
		acquireCtx, cancel := context.WithTimeout(l.ctx, 10*time.Second)
		conn, err = l.pool.Acquire(acquireCtx)
		cancel()

		if err == nil {
			break // Successfully acquired connection
		}

		log.Warn().
			Err(err).
			Int("attempt", attempt).
			Int("max_retries", maxRetries).
			Msg("Failed to acquire connection for LISTEN, retrying...")

		if attempt < maxRetries {
			// Exponential backoff: 1s, 2s, 4s, 8s
			delay := baseDelay * time.Duration(1<<(attempt-1))
			select {
			case <-time.After(delay):
			case <-l.ctx.Done():
				log.Info().Msg("Listener context cancelled during retry backoff")
				return
			}
		}
	}

	if err != nil {
		log.Error().Err(err).Msg("Failed to acquire connection for LISTEN after all retries")
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
	// Parse the notification payload
	var event ChangeEvent
	if err := json.Unmarshal([]byte(notification.Payload), &event); err != nil {
		log.Error().Err(err).Str("payload", notification.Payload).Msg("Failed to parse notification")
		return
	}

	// Skip debug logging for noisy events (e.g., worker heartbeats)
	isWorkerHeartbeat := event.Schema == "jobs" && event.Table == "workers" && event.Type == "UPDATE"
	if !isWorkerHeartbeat {
		log.Debug().
			Str("channel", notification.Channel).
			Str("payload", notification.Payload).
			Msg("Received notification")
	}

	// Compute ETA for job queue events
	if event.Schema == "jobs" && event.Table == "queue" && event.Record != nil {
		l.enrichJobWithETA(&event)
	}

	// Do RLS-aware filtering for table subscriptions
	if l.subManager != nil {
		filteredEvents := l.subManager.FilterEventForSubscribers(l.ctx, &event)

		// Send to each connection that has access
		for connID, filteredEvent := range filteredEvents {
			// Get connection from manager
			manager := l.handler.GetManager()
			manager.mu.RLock()
			conn, exists := manager.connections[connID]
			manager.mu.RUnlock()

			if exists {
				_ = conn.SendMessage(ServerMessage{
					Type:    MessageTypeChange,
					Payload: filteredEvent,
				})
			}
		}

		if !isWorkerHeartbeat {
			log.Debug().
				Str("table", fmt.Sprintf("%s.%s", event.Schema, event.Table)).
				Str("type", event.Type).
				Int("subscribers", len(filteredEvents)).
				Msg("Filtered and sent RLS-aware change event")
		}
	} else {
		if !isWorkerHeartbeat {
			log.Debug().
				Str("table", fmt.Sprintf("%s.%s", event.Schema, event.Table)).
				Str("type", event.Type).
				Msg("No subscription manager - change event not processed")
		}
	}
}

// enrichJobWithETA computes ETA fields for job queue events and adds them to the record
func (l *Listener) enrichJobWithETA(event *ChangeEvent) {
	// Parse progress directly from the record (it comes as a JSON object from pg_notify, not a string)
	progressData, ok := event.Record["progress"].(map[string]interface{})
	if !ok || progressData == nil {
		return
	}

	// Extract progress fields
	var progress jobs.Progress
	if percent, ok := progressData["percent"].(float64); ok {
		progress.Percent = int(percent)
	}
	if message, ok := progressData["message"].(string); ok {
		progress.Message = message
	}
	if etaSeconds, ok := progressData["estimated_seconds_left"].(float64); ok {
		eta := int(etaSeconds)
		progress.EstimatedSecondsLeft = &eta
	}

	// Get job status and started_at for ETA calculation
	status, _ := event.Record["status"].(string)
	startedAtStr, _ := event.Record["started_at"].(string)

	// Calculate ETA if not already present and job is running
	if progress.EstimatedSecondsLeft == nil && status == "running" && progress.Percent > 0 && progress.Percent < 100 {
		if startedAt, err := time.Parse(time.RFC3339, startedAtStr); err == nil {
			elapsed := time.Since(startedAt).Seconds()
			if elapsed > 0 {
				remainingPercent := float64(100 - progress.Percent)
				etaSeconds := int((elapsed / float64(progress.Percent)) * remainingPercent)
				progress.EstimatedSecondsLeft = &etaSeconds
			}
		}
	}

	// Add computed fields to the record
	event.Record["progress_percent"] = progress.Percent
	if progress.Message != "" {
		event.Record["progress_message"] = progress.Message
	}
	if progress.EstimatedSecondsLeft != nil {
		event.Record["estimated_seconds_left"] = *progress.EstimatedSecondsLeft
	}
}

// Stop stops the listener
func (l *Listener) Stop() {
	l.cancel()
}

// listenLogsPubSub listens for execution log events via PubSub
func (l *Listener) listenLogsPubSub() {
	msgChan, err := l.pubsub.Subscribe(l.ctx, LogChannel)
	if err != nil {
		log.Error().Err(err).Msg("Failed to subscribe to log channel")
		return
	}

	for {
		select {
		case <-l.ctx.Done():
			log.Info().Msg("Stopping PubSub log listener")
			return

		case msg, ok := <-msgChan:
			if !ok {
				log.Info().Msg("PubSub log channel closed")
				return
			}

			// Parse the log event
			var event storage.ExecutionLogEvent
			if err := json.Unmarshal(msg.Payload, &event); err != nil {
				log.Error().Err(err).Msg("Failed to parse log event")
				continue
			}

			// Forward to subscribers
			l.processLogEvent(&event)
		}
	}
}

// processLogEvent forwards a log event to subscribed connections
func (l *Listener) processLogEvent(event *storage.ExecutionLogEvent) {
	if l.subManager == nil || l.handler == nil {
		return
	}

	// Get connections subscribed to this execution's logs
	connIDs := l.subManager.GetLogSubscribers(event.ExecutionID)
	if len(connIDs) == 0 {
		return
	}

	// Send to each subscribed connection
	manager := l.handler.GetManager()
	for _, connID := range connIDs {
		manager.mu.RLock()
		conn, exists := manager.connections[connID]
		manager.mu.RUnlock()

		if exists {
			_ = conn.SendMessage(ServerMessage{
				Type:    MessageTypeExecutionLog,
				Payload: event,
			})
		}
	}

	// Note: We intentionally don't log here to avoid potential feedback loops
}

// listenAllLogsPubSub listens for all log events via PubSub (for admin dashboard streaming)
func (l *Listener) listenAllLogsPubSub() {
	msgChan, err := l.pubsub.Subscribe(l.ctx, AllLogsChannel)
	if err != nil {
		log.Error().Err(err).Msg("Failed to subscribe to all-logs channel")
		return
	}

	for {
		select {
		case <-l.ctx.Done():
			log.Info().Msg("Stopping PubSub all-logs listener")
			return

		case msg, ok := <-msgChan:
			if !ok {
				log.Info().Msg("PubSub all-logs channel closed")
				return
			}

			// Parse the log stream event
			var event storage.LogStreamEvent
			if err := json.Unmarshal(msg.Payload, &event); err != nil {
				log.Error().Err(err).Msg("Failed to parse all-logs event")
				continue
			}

			// Forward to subscribers
			l.processAllLogsEvent(&event)
		}
	}
}

// processAllLogsEvent forwards a log event to all-logs subscribers with filtering
func (l *Listener) processAllLogsEvent(event *storage.LogStreamEvent) {
	if l.subManager == nil || l.handler == nil {
		return
	}

	// Get all connections subscribed to all logs with their filter preferences
	subscribers := l.subManager.GetAllLogsSubscribers()
	if len(subscribers) == 0 {
		return
	}

	// Send to each subscribed connection that matches the filters
	manager := l.handler.GetManager()
	sentCount := 0

	for connID, sub := range subscribers {
		// Apply category filter if set
		if sub.Category != "" && string(event.Category) != sub.Category {
			continue
		}

		// Apply level filter if set
		if len(sub.Levels) > 0 {
			levelMatch := false
			for _, level := range sub.Levels {
				if string(event.Level) == level {
					levelMatch = true
					break
				}
			}
			if !levelMatch {
				continue
			}
		}

		// Send to this connection
		manager.mu.RLock()
		conn, exists := manager.connections[connID]
		manager.mu.RUnlock()

		if exists {
			_ = conn.SendMessage(ServerMessage{
				Type:    MessageTypeLogEntry,
				Payload: event,
			})
			sentCount++
		}
	}

	// Note: We intentionally don't log here to avoid a feedback loop
	// (logging would trigger another log event, which would be forwarded, etc.)
}
