package realtime

import (
	"context"
	"encoding/json"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fluxbase-eu/fluxbase/internal/pubsub"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

// ListenerPoolConfig holds configuration for the listener pool.
type ListenerPoolConfig struct {
	PoolSize      int // Number of LISTEN connections (default: 2)
	WorkerCount   int // Number of workers for processing (default: 4)
	QueueSize     int // Size of notification queue per worker (default: 1000)
	RetryInterval time.Duration
	MaxRetries    int
}

// DefaultListenerPoolConfig returns sensible defaults.
func DefaultListenerPoolConfig() ListenerPoolConfig {
	return ListenerPoolConfig{
		PoolSize:      2,
		WorkerCount:   4,
		QueueSize:     1000,
		RetryInterval: time.Second,
		MaxRetries:    5,
	}
}

// ListenerPool manages a pool of PostgreSQL LISTEN connections with parallel processing.
type ListenerPool struct {
	config     ListenerPoolConfig
	pool       *pgxpool.Pool
	handler    *RealtimeHandler
	subManager *SubscriptionManager
	pubsub     pubsub.PubSub

	ctx    context.Context
	cancel context.CancelFunc

	// Notification queue and workers
	notificationCh chan *pgconn.Notification
	workerWg       sync.WaitGroup

	// Connection tracking
	activeConnections int32
	mu                sync.RWMutex
	connWg            sync.WaitGroup

	// Metrics
	notificationsReceived  uint64
	notificationsProcessed uint64
	connectionFailures     uint64
	reconnections          uint64
}

// NewListenerPool creates a new listener pool with the given configuration.
func NewListenerPool(
	pool *pgxpool.Pool,
	handler *RealtimeHandler,
	subManager *SubscriptionManager,
	ps pubsub.PubSub,
	config ListenerPoolConfig,
) *ListenerPool {
	// Apply defaults for zero values
	if config.PoolSize <= 0 {
		config.PoolSize = 2
	}
	if config.WorkerCount <= 0 {
		config.WorkerCount = 4
	}
	if config.QueueSize <= 0 {
		config.QueueSize = 1000
	}
	if config.RetryInterval <= 0 {
		config.RetryInterval = time.Second
	}
	if config.MaxRetries <= 0 {
		config.MaxRetries = 5
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &ListenerPool{
		config:         config,
		pool:           pool,
		handler:        handler,
		subManager:     subManager,
		pubsub:         ps,
		ctx:            ctx,
		cancel:         cancel,
		notificationCh: make(chan *pgconn.Notification, config.QueueSize*config.WorkerCount),
	}
}

// Start begins the listener pool.
func (lp *ListenerPool) Start() error {
	// Start worker goroutines
	for i := 0; i < lp.config.WorkerCount; i++ {
		lp.workerWg.Add(1)
		go lp.worker(i)
	}

	log.Info().
		Int("workers", lp.config.WorkerCount).
		Int("queue_size", lp.config.QueueSize).
		Msg("Listener pool workers started")

	// Start listener connections
	for i := 0; i < lp.config.PoolSize; i++ {
		lp.connWg.Add(1)
		go lp.runListener(i)
	}

	log.Info().
		Int("pool_size", lp.config.PoolSize).
		Msg("Listener pool connections started")

	// Start PubSub listeners if available
	if lp.pubsub != nil {
		go lp.listenLogsPubSub()
		go lp.listenAllLogsPubSub()
		log.Info().Msg("PubSub listeners started")
	}

	return nil
}

// Stop gracefully shuts down the listener pool.
func (lp *ListenerPool) Stop() {
	log.Info().Msg("Stopping listener pool...")

	// Signal all goroutines to stop
	lp.cancel()

	// Wait for listener connections to close
	lp.connWg.Wait()

	// Close the notification channel to signal workers to stop
	close(lp.notificationCh)

	// Wait for workers to finish
	lp.workerWg.Wait()

	log.Info().
		Uint64("notifications_received", atomic.LoadUint64(&lp.notificationsReceived)).
		Uint64("notifications_processed", atomic.LoadUint64(&lp.notificationsProcessed)).
		Uint64("connection_failures", atomic.LoadUint64(&lp.connectionFailures)).
		Uint64("reconnections", atomic.LoadUint64(&lp.reconnections)).
		Msg("Listener pool stopped")
}

// runListener manages a single LISTEN connection with reconnection logic.
func (lp *ListenerPool) runListener(id int) {
	defer lp.connWg.Done()

	for {
		select {
		case <-lp.ctx.Done():
			log.Debug().Int("listener_id", id).Msg("Listener shutting down")
			return
		default:
			if err := lp.listen(id); err != nil {
				if lp.ctx.Err() != nil {
					return // Context cancelled, exit cleanly
				}
				atomic.AddUint64(&lp.connectionFailures, 1)
				log.Error().Err(err).Int("listener_id", id).Msg("Listener error, reconnecting...")

				// Wait before reconnecting
				select {
				case <-time.After(lp.config.RetryInterval):
					atomic.AddUint64(&lp.reconnections, 1)
				case <-lp.ctx.Done():
					return
				}
			}
		}
	}
}

// listen runs the LISTEN loop for a single connection.
func (lp *ListenerPool) listen(id int) error {
	// Acquire connection with retry
	var conn *pgxpool.Conn
	var err error

	for attempt := 1; attempt <= lp.config.MaxRetries; attempt++ {
		if lp.ctx.Err() != nil {
			return lp.ctx.Err()
		}

		acquireCtx, cancel := context.WithTimeout(lp.ctx, 10*time.Second)
		conn, err = lp.pool.Acquire(acquireCtx)
		cancel()

		if err == nil {
			break
		}

		log.Warn().
			Err(err).
			Int("listener_id", id).
			Int("attempt", attempt).
			Int("max_retries", lp.config.MaxRetries).
			Msg("Failed to acquire connection for LISTEN")

		if attempt < lp.config.MaxRetries {
			delay := lp.config.RetryInterval * time.Duration(1<<(attempt-1))
			select {
			case <-time.After(delay):
			case <-lp.ctx.Done():
				return lp.ctx.Err()
			}
		}
	}

	if err != nil {
		return err
	}
	defer conn.Release()

	atomic.AddInt32(&lp.activeConnections, 1)
	defer atomic.AddInt32(&lp.activeConnections, -1)

	// Execute LISTEN
	_, err = conn.Exec(lp.ctx, "LISTEN fluxbase_changes")
	if err != nil {
		return err
	}

	log.Debug().Int("listener_id", id).Msg("LISTEN started")

	// Listen loop
	for {
		select {
		case <-lp.ctx.Done():
			return nil

		default:
			waitCtx, cancel := context.WithTimeout(lp.ctx, 5*time.Second)
			notification, err := conn.Conn().WaitForNotification(waitCtx)
			cancel()

			if err != nil {
				if lp.ctx.Err() != nil {
					return nil
				}
				if err == context.DeadlineExceeded || waitCtx.Err() == context.DeadlineExceeded {
					continue // Timeout is expected
				}
				return err
			}

			atomic.AddUint64(&lp.notificationsReceived, 1)

			// Send to worker queue (non-blocking with timeout)
			select {
			case lp.notificationCh <- notification:
			case <-time.After(100 * time.Millisecond):
				log.Warn().Int("listener_id", id).Msg("Notification queue full, dropping notification")
			case <-lp.ctx.Done():
				return nil
			}
		}
	}
}

// worker processes notifications from the queue.
func (lp *ListenerPool) worker(id int) {
	defer lp.workerWg.Done()

	for notification := range lp.notificationCh {
		lp.processNotification(notification)
		atomic.AddUint64(&lp.notificationsProcessed, 1)
	}

	log.Debug().Int("worker_id", id).Msg("Worker stopped")
}

// processNotification handles a single notification (same logic as original Listener).
func (lp *ListenerPool) processNotification(notification *pgconn.Notification) {
	var event ChangeEvent
	if err := json.Unmarshal([]byte(notification.Payload), &event); err != nil {
		log.Error().Err(err).Str("payload", notification.Payload).Msg("Failed to parse notification")
		return
	}

	// Skip debug logging for noisy events
	isWorkerHeartbeat := event.Schema == "jobs" && event.Table == "workers" && event.Type == "UPDATE"
	if !isWorkerHeartbeat {
		log.Debug().
			Str("channel", notification.Channel).
			Str("table", event.Schema+"."+event.Table).
			Str("type", event.Type).
			Msg("Processing notification")
	}

	// Compute ETA for job queue events
	if event.Schema == "jobs" && event.Table == "queue" && event.Record != nil {
		lp.enrichJobWithETA(&event)
	}

	// Do RLS-aware filtering
	if lp.subManager != nil {
		filteredEvents := lp.subManager.FilterEventForSubscribers(lp.ctx, &event)

		manager := lp.handler.GetManager()
		for connID, filteredEvent := range filteredEvents {
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
				Str("table", event.Schema+"."+event.Table).
				Int("subscribers", len(filteredEvents)).
				Msg("Delivered filtered change event")
		}
	}
}

// enrichJobWithETA adds ETA fields to job events (copied from original Listener).
func (lp *ListenerPool) enrichJobWithETA(event *ChangeEvent) {
	progressData, ok := event.Record["progress"].(map[string]interface{})
	if !ok || progressData == nil {
		return
	}

	var percent int
	var message string
	var etaSeconds *int

	if p, ok := progressData["percent"].(float64); ok {
		percent = int(p)
	}
	if m, ok := progressData["message"].(string); ok {
		message = m
	}
	if e, ok := progressData["estimated_seconds_left"].(float64); ok {
		eta := int(e)
		etaSeconds = &eta
	}

	status, _ := event.Record["status"].(string)
	startedAtStr, _ := event.Record["started_at"].(string)

	if etaSeconds == nil && status == "running" && percent > 0 && percent < 100 {
		if startedAt, err := time.Parse(time.RFC3339, startedAtStr); err == nil {
			elapsed := time.Since(startedAt).Seconds()
			if elapsed > 0 {
				remainingPercent := float64(100 - percent)
				eta := int((elapsed / float64(percent)) * remainingPercent)
				etaSeconds = &eta
			}
		}
	}

	event.Record["progress_percent"] = percent
	if message != "" {
		event.Record["progress_message"] = message
	}
	if etaSeconds != nil {
		event.Record["estimated_seconds_left"] = *etaSeconds
	}
}

// GetMetrics returns current metrics.
func (lp *ListenerPool) GetMetrics() ListenerPoolMetrics {
	return ListenerPoolMetrics{
		ActiveConnections:      atomic.LoadInt32(&lp.activeConnections),
		NotificationsReceived:  atomic.LoadUint64(&lp.notificationsReceived),
		NotificationsProcessed: atomic.LoadUint64(&lp.notificationsProcessed),
		ConnectionFailures:     atomic.LoadUint64(&lp.connectionFailures),
		Reconnections:          atomic.LoadUint64(&lp.reconnections),
		QueueLength:            len(lp.notificationCh),
		QueueCapacity:          cap(lp.notificationCh),
	}
}

// ListenerPoolMetrics contains metrics for the listener pool.
type ListenerPoolMetrics struct {
	ActiveConnections      int32
	NotificationsReceived  uint64
	NotificationsProcessed uint64
	ConnectionFailures     uint64
	Reconnections          uint64
	QueueLength            int
	QueueCapacity          int
}

// listenLogsPubSub listens for execution log events via PubSub (copied from original Listener).
func (lp *ListenerPool) listenLogsPubSub() {
	msgChan, err := lp.pubsub.Subscribe(lp.ctx, LogChannel)
	if err != nil {
		log.Error().Err(err).Msg("Failed to subscribe to log channel")
		return
	}

	for {
		select {
		case <-lp.ctx.Done():
			return
		case msg, ok := <-msgChan:
			if !ok {
				return
			}
			lp.processLogEvent(msg.Payload)
		}
	}
}

// listenAllLogsPubSub listens for all log events (copied from original Listener).
func (lp *ListenerPool) listenAllLogsPubSub() {
	msgChan, err := lp.pubsub.Subscribe(lp.ctx, AllLogsChannel)
	if err != nil {
		log.Error().Err(err).Msg("Failed to subscribe to all-logs channel")
		return
	}

	for {
		select {
		case <-lp.ctx.Done():
			return
		case msg, ok := <-msgChan:
			if !ok {
				return
			}
			lp.processAllLogsEvent(msg.Payload)
		}
	}
}

// processLogEvent forwards log events to subscribers.
func (lp *ListenerPool) processLogEvent(payload []byte) {
	if lp.subManager == nil || lp.handler == nil {
		return
	}

	// Import avoided - parse directly
	var event struct {
		ExecutionID string `json:"execution_id"`
	}
	if err := json.Unmarshal(payload, &event); err != nil {
		log.Error().Err(err).Msg("Failed to parse log event")
		return
	}

	connIDs := lp.subManager.GetLogSubscribers(event.ExecutionID)
	if len(connIDs) == 0 {
		return
	}

	// Re-parse full event for sending
	var fullEvent interface{}
	_ = json.Unmarshal(payload, &fullEvent)

	manager := lp.handler.GetManager()
	for _, connID := range connIDs {
		manager.mu.RLock()
		conn, exists := manager.connections[connID]
		manager.mu.RUnlock()

		if exists {
			_ = conn.SendMessage(ServerMessage{
				Type:    MessageTypeExecutionLog,
				Payload: fullEvent,
			})
		}
	}
}

// processAllLogsEvent forwards all-logs events with filtering.
func (lp *ListenerPool) processAllLogsEvent(payload []byte) {
	if lp.subManager == nil || lp.handler == nil {
		return
	}

	// Parse event for filtering
	var event struct {
		Category string `json:"category"`
		Level    string `json:"level"`
	}
	if err := json.Unmarshal(payload, &event); err != nil {
		log.Error().Err(err).Msg("Failed to parse all-logs event")
		return
	}

	subscribers := lp.subManager.GetAllLogsSubscribers()
	if len(subscribers) == 0 {
		return
	}

	// Re-parse full event for sending
	var fullEvent interface{}
	_ = json.Unmarshal(payload, &fullEvent)

	manager := lp.handler.GetManager()
	for connID, sub := range subscribers {
		// Apply filters
		if sub.Category != "" && event.Category != sub.Category {
			continue
		}
		if len(sub.Levels) > 0 {
			match := false
			for _, level := range sub.Levels {
				if event.Level == level {
					match = true
					break
				}
			}
			if !match {
				continue
			}
		}

		manager.mu.RLock()
		conn, exists := manager.connections[connID]
		manager.mu.RUnlock()

		if exists {
			_ = conn.SendMessage(ServerMessage{
				Type:    MessageTypeLogEntry,
				Payload: fullEvent,
			})
		}
	}
}
