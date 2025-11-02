package webhook

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

// WebhookEvent represents an event waiting to be delivered
type WebhookEvent struct {
	ID            uuid.UUID       `db:"id"`
	WebhookID     uuid.UUID       `db:"webhook_id"`
	EventType     string          `db:"event_type"`
	TableSchema   string          `db:"table_schema"`
	TableName     string          `db:"table_name"`
	RecordID      *string         `db:"record_id"`
	OldData       json.RawMessage `db:"old_data"`
	NewData       json.RawMessage `db:"new_data"`
	Processed     bool            `db:"processed"`
	Attempts      int             `db:"attempts"`
	LastAttemptAt *time.Time      `db:"last_attempt_at"`
	NextRetryAt   *time.Time      `db:"next_retry_at"`
	ErrorMessage  *string         `db:"error_message"`
	CreatedAt     time.Time       `db:"created_at"`
}

// TriggerService manages webhook event processing
type TriggerService struct {
	db              *pgxpool.Pool
	webhookSvc      *WebhookService
	workers         int
	backlogInterval time.Duration
	eventChan       chan uuid.UUID
	stopChan        chan struct{}
	backlogTicker   *time.Ticker
	cleanupTicker   *time.Ticker
	cancel          context.CancelFunc
}

// NewTriggerService creates a new webhook trigger service
func NewTriggerService(db *pgxpool.Pool, webhookSvc *WebhookService, workers int) *TriggerService {
	if workers <= 0 {
		workers = 4
	}

	return &TriggerService{
		db:              db,
		webhookSvc:      webhookSvc,
		workers:         workers,
		backlogInterval: 30 * time.Second, // Default 30 seconds
		eventChan:       make(chan uuid.UUID, 1000),
		stopChan:        make(chan struct{}),
	}
}

// SetBacklogInterval allows customizing the backlog check interval (useful for testing)
// If the service is already running, it will reset the ticker with the new interval
func (s *TriggerService) SetBacklogInterval(interval time.Duration) {
	s.backlogInterval = interval

	// If the ticker is already running, reset it with the new interval
	if s.backlogTicker != nil {
		s.backlogTicker.Reset(interval)
	}
}

// Start begins processing webhook events
func (s *TriggerService) Start(ctx context.Context) error {
	log.Info().Int("workers", s.workers).Msg("Starting webhook trigger service")

	// Create a cancellable context
	ctx, cancel := context.WithCancel(ctx)
	s.cancel = cancel

	// Start worker pool
	for i := 0; i < s.workers; i++ {
		go s.worker(ctx, i)
	}

	// Start listening for new webhook events
	go s.listen(ctx)

	// Start backlog processor (processes events that need retry)
	go s.processBacklog(ctx)

	// Start cleanup goroutine (removes old processed events)
	s.cleanupTicker = time.NewTicker(1 * time.Hour)
	go s.cleanup(ctx)

	return nil
}

// Stop gracefully stops the trigger service
func (s *TriggerService) Stop() {
	log.Info().Msg("Stopping webhook trigger service")

	// Cancel the context to interrupt all goroutines
	if s.cancel != nil {
		s.cancel()
	}

	close(s.stopChan)
	if s.backlogTicker != nil {
		s.backlogTicker.Stop()
	}
	if s.cleanupTicker != nil {
		s.cleanupTicker.Stop()
	}
}

// listen listens for PostgreSQL notifications about new webhook events
func (s *TriggerService) listen(ctx context.Context) {
	conn, err := s.db.Acquire(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to acquire connection for webhook listener")
		return
	}
	defer conn.Release()

	// Listen on webhook_event channel
	_, err = conn.Exec(ctx, "LISTEN webhook_event")
	if err != nil {
		log.Error().Err(err).Msg("Failed to LISTEN on webhook_event channel")
		return
	}

	log.Info().Msg("Webhook trigger service listening for events")

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopChan:
			return
		default:
			// Wait for notification with timeout
			notification, err := conn.Conn().WaitForNotification(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				log.Error().Err(err).Msg("Error waiting for notification")
				time.Sleep(1 * time.Second)
				continue
			}

			// Parse webhook ID from notification payload
			webhookID, err := uuid.Parse(notification.Payload)
			if err != nil {
				log.Error().Err(err).Str("payload", notification.Payload).Msg("Invalid webhook ID in notification")
				continue
			}

			// Queue webhook for processing
			select {
			case s.eventChan <- webhookID:
				log.Debug().Str("webhook_id", webhookID.String()).Msg("Queued webhook for processing")
			default:
				log.Warn().Str("webhook_id", webhookID.String()).Msg("Event channel full, skipping")
			}
		}
	}
}

// worker processes webhook events from the queue
func (s *TriggerService) worker(ctx context.Context, workerID int) {
	log.Debug().Int("worker_id", workerID).Msg("Webhook worker started")

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopChan:
			return
		case webhookID := <-s.eventChan:
			s.processWebhookEvents(ctx, webhookID, workerID)
		}
	}
}

// processWebhookEvents processes all pending events for a webhook
func (s *TriggerService) processWebhookEvents(ctx context.Context, webhookID uuid.UUID, workerID int) {
	// Get webhook configuration
	webhook, err := s.webhookSvc.Get(ctx, webhookID)
	if err != nil {
		log.Error().Err(err).Str("webhook_id", webhookID.String()).Msg("Failed to get webhook")
		return
	}

	if !webhook.Enabled {
		log.Debug().Str("webhook_id", webhookID.String()).Msg("Webhook is disabled, skipping")
		return
	}

	// Get unprocessed events for this webhook (limit to 10 at a time)
	query := `
		SELECT id, webhook_id, event_type, table_schema, table_name, record_id,
		       old_data, new_data, processed, attempts, last_attempt_at, next_retry_at, error_message, created_at
		FROM auth.webhook_events
		WHERE webhook_id = $1
		  AND processed = FALSE
		  AND (next_retry_at IS NULL OR next_retry_at <= NOW())
		ORDER BY created_at ASC
		LIMIT 10
	`

	rows, err := s.db.Query(ctx, query, webhookID)
	if err != nil {
		log.Error().Err(err).Str("webhook_id", webhookID.String()).Msg("Failed to query webhook events")
		return
	}
	defer rows.Close()

	var events []WebhookEvent
	for rows.Next() {
		var event WebhookEvent
		err := rows.Scan(
			&event.ID,
			&event.WebhookID,
			&event.EventType,
			&event.TableSchema,
			&event.TableName,
			&event.RecordID,
			&event.OldData,
			&event.NewData,
			&event.Processed,
			&event.Attempts,
			&event.LastAttemptAt,
			&event.NextRetryAt,
			&event.ErrorMessage,
			&event.CreatedAt,
		)
		if err != nil {
			log.Error().Err(err).Msg("Failed to scan webhook event")
			continue
		}
		events = append(events, event)
	}

	if len(events) == 0 {
		return
	}

	log.Debug().
		Int("worker_id", workerID).
		Str("webhook_id", webhookID.String()).
		Int("event_count", len(events)).
		Msg("Processing webhook events")

	// Process each event
	for _, event := range events {
		s.deliverEvent(ctx, webhook, &event)
	}
}

// deliverEvent delivers a single webhook event
func (s *TriggerService) deliverEvent(ctx context.Context, webhook *Webhook, event *WebhookEvent) {
	// Create payload
	payload := &WebhookPayload{
		Event:     event.EventType,
		Table:     event.TableName,
		Schema:    event.TableSchema,
		Timestamp: time.Now(),
	}

	// Add record data based on event type
	switch event.EventType {
	case "INSERT":
		payload.Record = event.NewData
	case "UPDATE":
		payload.Record = event.NewData
		payload.OldRecord = event.OldData
	case "DELETE":
		payload.Record = event.OldData
	}

	// Deliver webhook
	err := s.webhookSvc.Deliver(ctx, webhook, payload)

	// Update event status
	if err != nil {
		s.handleDeliveryFailure(ctx, event, webhook, err.Error())
	} else {
		s.markEventSuccess(ctx, event.ID)
	}
}

// handleDeliveryFailure handles failed webhook delivery
func (s *TriggerService) handleDeliveryFailure(ctx context.Context, event *WebhookEvent, webhook *Webhook, errorMsg string) {
	attempts := event.Attempts + 1

	// Check if max retries reached
	if attempts >= webhook.MaxRetries {
		log.Warn().
			Str("event_id", event.ID.String()).
			Str("webhook_id", webhook.ID.String()).
			Int("attempts", attempts).
			Msg("Max retries reached, marking event as processed")

		// Mark as processed (failed)
		query := `
			UPDATE auth.webhook_events
			SET processed = TRUE,
			    attempts = $1,
			    last_attempt_at = NOW(),
			    error_message = $2
			WHERE id = $3
		`
		_, err := s.db.Exec(ctx, query, attempts, errorMsg, event.ID)
		if err != nil {
			log.Error().Err(err).Str("event_id", event.ID.String()).Msg("Failed to mark event as failed")
		}
		return
	}

	// Calculate next retry time with exponential backoff
	backoffSeconds := webhook.RetryBackoffSeconds * attempts
	nextRetry := time.Now().Add(time.Duration(backoffSeconds) * time.Second)

	log.Debug().
		Str("event_id", event.ID.String()).
		Int("attempts", attempts).
		Time("next_retry", nextRetry).
		Msg("Scheduling webhook retry")

	// Update event with retry info
	query := `
		UPDATE auth.webhook_events
		SET attempts = $1,
		    last_attempt_at = NOW(),
		    next_retry_at = $2,
		    error_message = $3
		WHERE id = $4
	`
	_, err := s.db.Exec(ctx, query, attempts, nextRetry, errorMsg, event.ID)
	if err != nil {
		log.Error().Err(err).Str("event_id", event.ID.String()).Msg("Failed to update event retry info")
	}
}

// markEventSuccess marks an event as successfully processed
func (s *TriggerService) markEventSuccess(ctx context.Context, eventID uuid.UUID) {
	query := `
		UPDATE auth.webhook_events
		SET processed = TRUE,
		    last_attempt_at = NOW()
		WHERE id = $1
	`
	_, err := s.db.Exec(ctx, query, eventID)
	if err != nil {
		log.Error().Err(err).Str("event_id", eventID.String()).Msg("Failed to mark event as success")
	}
}

// processBacklog periodically processes events that are ready for retry
func (s *TriggerService) processBacklog(ctx context.Context) {
	s.backlogTicker = time.NewTicker(s.backlogInterval)
	defer s.backlogTicker.Stop()

	// Run immediately on startup to check for any pending retries
	s.checkForRetries(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopChan:
			return
		case <-s.backlogTicker.C:
			s.checkForRetries(ctx)
		}
	}
}

// checkForRetries finds events that need to be retried
func (s *TriggerService) checkForRetries(ctx context.Context) {
	query := `
		SELECT DISTINCT webhook_id
		FROM auth.webhook_events
		WHERE processed = FALSE
		  AND next_retry_at IS NOT NULL
		  AND next_retry_at <= NOW()
		LIMIT 50
	`

	rows, err := s.db.Query(ctx, query)
	if err != nil {
		log.Error().Err(err).Msg("Failed to query webhooks needing retry")
		return
	}
	defer rows.Close()

	var count int
	for rows.Next() {
		var webhookID uuid.UUID
		if err := rows.Scan(&webhookID); err != nil {
			log.Error().Err(err).Msg("Failed to scan webhook ID")
			continue
		}

		// Queue webhook for processing
		select {
		case s.eventChan <- webhookID:
			count++
		default:
			log.Warn().Str("webhook_id", webhookID.String()).Msg("Event channel full, will retry next cycle")
		}
	}

	if count > 0 {
		log.Debug().Int("count", count).Msg("Queued webhooks for retry")
	}
}

// cleanup removes old processed webhook events to prevent table bloat
func (s *TriggerService) cleanup(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopChan:
			return
		case <-s.cleanupTicker.C:
			s.cleanupOldEvents(ctx)
		}
	}
}

// cleanupOldEvents deletes processed events older than 7 days
func (s *TriggerService) cleanupOldEvents(ctx context.Context) {
	query := `
		DELETE FROM auth.webhook_events
		WHERE processed = TRUE
		  AND created_at < NOW() - INTERVAL '7 days'
	`

	result, err := s.db.Exec(ctx, query)
	if err != nil {
		log.Error().Err(err).Msg("Failed to cleanup old webhook events")
		return
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected > 0 {
		log.Info().Int64("rows_deleted", rowsAffected).Msg("Cleaned up old webhook events")
	}
}
