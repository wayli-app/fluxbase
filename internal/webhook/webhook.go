package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/fluxbase-eu/fluxbase/internal/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"
)

// Webhook represents a webhook configuration
type Webhook struct {
	ID                  uuid.UUID         `json:"id"`
	Name                string            `json:"name"`
	Description         *string           `json:"description,omitempty"`
	URL                 string            `json:"url"`
	Secret              *string           `json:"secret,omitempty"`
	Enabled             bool              `json:"enabled"`
	Events              []EventConfig     `json:"events"`
	MaxRetries          int               `json:"max_retries"`
	RetryBackoffSeconds int               `json:"retry_backoff_seconds"`
	TimeoutSeconds      int               `json:"timeout_seconds"`
	Headers             map[string]string `json:"headers"`
	CreatedBy           *uuid.UUID        `json:"created_by,omitempty"`
	CreatedAt           time.Time         `json:"created_at"`
	UpdatedAt           time.Time         `json:"updated_at"`
}

// EventConfig represents events a webhook subscribes to
type EventConfig struct {
	Table      string   `json:"table"`      // e.g., "products", "users"
	Operations []string `json:"operations"` // INSERT, UPDATE, DELETE
}

// WebhookDelivery represents a webhook delivery attempt
type WebhookDelivery struct {
	ID             uuid.UUID       `json:"id"`
	WebhookID      uuid.UUID       `json:"webhook_id"`
	EventType      string          `json:"event_type"`
	TableName      string          `json:"table_name"`
	RecordID       *string         `json:"record_id,omitempty"`
	Payload        json.RawMessage `json:"payload"`
	AttemptNumber  int             `json:"attempt_number"`
	Status         string          `json:"status"` // pending, success, failed, retrying
	HTTPStatusCode *int            `json:"http_status_code,omitempty"`
	ResponseBody   *string         `json:"response_body,omitempty"`
	ErrorMessage   *string         `json:"error_message,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
	DeliveredAt    *time.Time      `json:"delivered_at,omitempty"`
	NextRetryAt    *time.Time      `json:"next_retry_at,omitempty"`
}

// WebhookPayload represents the payload sent to webhooks
type WebhookPayload struct {
	Event     string          `json:"event"`                // INSERT, UPDATE, DELETE
	Table     string          `json:"table"`                // table name
	Schema    string          `json:"schema"`               // schema name
	Record    json.RawMessage `json:"record"`               // new record data
	OldRecord json.RawMessage `json:"old_record,omitempty"` // old record (for UPDATE/DELETE)
	Timestamp time.Time       `json:"timestamp"`
}

// WebhookService manages webhooks
type WebhookService struct {
	db     *database.Connection
	client *http.Client
}

// NewWebhookService creates a new webhook service
func NewWebhookService(db *database.Connection) *WebhookService {
	return &WebhookService{
		db: db,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Create creates a new webhook
func (s *WebhookService) Create(ctx context.Context, webhook *Webhook) error {
	eventsJSON, err := json.Marshal(webhook.Events)
	if err != nil {
		return fmt.Errorf("failed to marshal events: %w", err)
	}

	headersJSON, err := json.Marshal(webhook.Headers)
	if err != nil {
		return fmt.Errorf("failed to marshal headers: %w", err)
	}

	query := `
		INSERT INTO auth.webhooks (name, description, url, secret, enabled, events, max_retries, retry_backoff_seconds, timeout_seconds, headers, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, created_at, updated_at
	`

	err = database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, query,
			webhook.Name,
			webhook.Description,
			webhook.URL,
			webhook.Secret,
			webhook.Enabled,
			eventsJSON,
			webhook.MaxRetries,
			webhook.RetryBackoffSeconds,
			webhook.TimeoutSeconds,
			headersJSON,
			webhook.CreatedBy,
		).Scan(&webhook.ID, &webhook.CreatedAt, &webhook.UpdatedAt)
	})

	if err != nil {
		return fmt.Errorf("failed to create webhook: %w", err)
	}

	return nil
}

// List lists all webhooks
func (s *WebhookService) List(ctx context.Context) ([]*Webhook, error) {
	query := `
		SELECT id, name, description, url, secret, enabled, events, max_retries, retry_backoff_seconds, timeout_seconds, headers, created_by, created_at, updated_at
		FROM auth.webhooks
		ORDER BY created_at DESC
	`

	var webhooks []*Webhook
	err := database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, query)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var webhook Webhook
			var eventsJSON, headersJSON []byte

			err := rows.Scan(
				&webhook.ID,
				&webhook.Name,
				&webhook.Description,
				&webhook.URL,
				&webhook.Secret,
				&webhook.Enabled,
				&eventsJSON,
				&webhook.MaxRetries,
				&webhook.RetryBackoffSeconds,
				&webhook.TimeoutSeconds,
				&headersJSON,
				&webhook.CreatedBy,
				&webhook.CreatedAt,
				&webhook.UpdatedAt,
			)
			if err != nil {
				return err
			}

			if err := json.Unmarshal(eventsJSON, &webhook.Events); err != nil {
				return err
			}

			if err := json.Unmarshal(headersJSON, &webhook.Headers); err != nil {
				return err
			}

			webhooks = append(webhooks, &webhook)
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list webhooks: %w", err)
	}

	return webhooks, nil
}

// Get retrieves a webhook by ID
func (s *WebhookService) Get(ctx context.Context, id uuid.UUID) (*Webhook, error) {
	query := `
		SELECT id, name, description, url, secret, enabled, events, max_retries, retry_backoff_seconds, timeout_seconds, headers, created_by, created_at, updated_at
		FROM auth.webhooks
		WHERE id = $1
	`

	var webhook Webhook
	var eventsJSON, headersJSON []byte

	err := database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, query, id).Scan(
			&webhook.ID,
			&webhook.Name,
			&webhook.Description,
			&webhook.URL,
			&webhook.Secret,
			&webhook.Enabled,
			&eventsJSON,
			&webhook.MaxRetries,
			&webhook.RetryBackoffSeconds,
			&webhook.TimeoutSeconds,
			&headersJSON,
			&webhook.CreatedBy,
			&webhook.CreatedAt,
			&webhook.UpdatedAt,
		)
	})

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("webhook not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get webhook: %w", err)
	}

	if err := json.Unmarshal(eventsJSON, &webhook.Events); err != nil {
		return nil, fmt.Errorf("failed to unmarshal events: %w", err)
	}

	if err := json.Unmarshal(headersJSON, &webhook.Headers); err != nil {
		return nil, fmt.Errorf("failed to unmarshal headers: %w", err)
	}

	return &webhook, nil
}

// Update updates a webhook
func (s *WebhookService) Update(ctx context.Context, id uuid.UUID, webhook *Webhook) error {
	eventsJSON, err := json.Marshal(webhook.Events)
	if err != nil {
		return fmt.Errorf("failed to marshal events: %w", err)
	}

	headersJSON, err := json.Marshal(webhook.Headers)
	if err != nil {
		return fmt.Errorf("failed to marshal headers: %w", err)
	}

	query := `
		UPDATE auth.webhooks
		SET name = $1, description = $2, url = $3, secret = $4, enabled = $5, events = $6,
		    max_retries = $7, retry_backoff_seconds = $8, timeout_seconds = $9, headers = $10
		WHERE id = $11
	`

	err = database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		result, err := tx.Exec(ctx, query,
			webhook.Name,
			webhook.Description,
			webhook.URL,
			webhook.Secret,
			webhook.Enabled,
			eventsJSON,
			webhook.MaxRetries,
			webhook.RetryBackoffSeconds,
			webhook.TimeoutSeconds,
			headersJSON,
			id,
		)

		if err != nil {
			return err
		}

		if result.RowsAffected() == 0 {
			return fmt.Errorf("webhook not found")
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to update webhook: %w", err)
	}

	return nil
}

// Delete deletes a webhook
func (s *WebhookService) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM auth.webhooks WHERE id = $1`

	err := database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		result, err := tx.Exec(ctx, query, id)
		if err != nil {
			return err
		}

		if result.RowsAffected() == 0 {
			return fmt.Errorf("webhook not found")
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to delete webhook: %w", err)
	}

	return nil
}

// Deliver sends a webhook payload to the configured URL
func (s *WebhookService) Deliver(ctx context.Context, webhook *Webhook, payload *WebhookPayload) error {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Send HTTP request synchronously and return error if it fails
	// The trigger service will handle retries via webhook_events table
	return s.sendWebhookSync(ctx, webhook, payloadJSON)
}

// sendWebhookSync sends an HTTP request synchronously and returns any error
func (s *WebhookService) sendWebhookSync(ctx context.Context, webhook *Webhook, payloadJSON []byte) error {
	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", webhook.URL, bytes.NewReader(payloadJSON))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Fluxbase-Webhooks/1.0")

	// Add custom headers
	for key, value := range webhook.Headers {
		req.Header.Set(key, value)
	}

	// Add HMAC signature if secret is provided
	if webhook.Secret != nil && *webhook.Secret != "" {
		signature := s.generateSignature(payloadJSON, *webhook.Secret)
		req.Header.Set("X-Webhook-Signature", signature)
	}

	// Send request with timeout
	client := &http.Client{
		Timeout: time.Duration(webhook.TimeoutSeconds) * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// sendWebhook sends the actual HTTP request (runs asynchronously)
func (s *WebhookService) sendWebhook(ctx context.Context, deliveryID uuid.UUID, webhook *Webhook, payloadJSON []byte) {
	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", webhook.URL, bytes.NewReader(payloadJSON))
	if err != nil {
		s.markDeliveryFailed(ctx, deliveryID, 0, nil, fmt.Sprintf("failed to create request: %v", err))
		return
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Fluxbase-Webhooks/1.0")

	// Add custom headers
	for key, value := range webhook.Headers {
		req.Header.Set(key, value)
	}

	// Add HMAC signature if secret is provided
	if webhook.Secret != nil && *webhook.Secret != "" {
		signature := s.generateSignature(payloadJSON, *webhook.Secret)
		req.Header.Set("X-Webhook-Signature", signature)
	}

	// Send request with timeout
	client := &http.Client{
		Timeout: time.Duration(webhook.TimeoutSeconds) * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		s.markDeliveryFailed(ctx, deliveryID, 0, nil, fmt.Sprintf("failed to send request: %v", err))
		return
	}
	defer resp.Body.Close()

	// Read response body
	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)

	// Check status code
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		s.markDeliverySuccess(ctx, deliveryID, resp.StatusCode, &bodyStr)
	} else {
		s.markDeliveryFailed(ctx, deliveryID, resp.StatusCode, &bodyStr, fmt.Sprintf("HTTP %d", resp.StatusCode))
	}
}

// generateSignature generates HMAC SHA256 signature
func (s *WebhookService) generateSignature(payload []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}

// markDeliverySuccess marks a delivery as successful
func (s *WebhookService) markDeliverySuccess(ctx context.Context, deliveryID uuid.UUID, statusCode int, responseBody *string) {
	query := `
		UPDATE auth.webhook_deliveries
		SET status = 'success', http_status_code = $1, response_body = $2, delivered_at = NOW()
		WHERE id = $3
	`

	err := database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, query, statusCode, responseBody, deliveryID)
		return err
	})
	if err != nil {
		log.Error().Err(err).Str("delivery_id", deliveryID.String()).Msg("Failed to mark delivery as success")
	}
}

// markDeliveryFailed marks a delivery as failed
func (s *WebhookService) markDeliveryFailed(ctx context.Context, deliveryID uuid.UUID, statusCode int, responseBody *string, errorMsg string) {
	query := `
		UPDATE auth.webhook_deliveries
		SET status = 'failed', http_status_code = $1, response_body = $2, error_message = $3
		WHERE id = $4
	`

	err := database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, query, statusCode, responseBody, errorMsg, deliveryID)
		return err
	})
	if err != nil {
		log.Error().Err(err).Str("delivery_id", deliveryID.String()).Msg("Failed to mark delivery as failed")
	}
}

// ListDeliveries lists webhook deliveries
func (s *WebhookService) ListDeliveries(ctx context.Context, webhookID uuid.UUID, limit int) ([]*WebhookDelivery, error) {
	query := `
		SELECT id, webhook_id, event_type, table_name, record_id, payload, attempt_number, status,
		       http_status_code, response_body, error_message, created_at, delivered_at, next_retry_at
		FROM auth.webhook_deliveries
		WHERE webhook_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	var deliveries []*WebhookDelivery
	err := database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, query, webhookID, limit)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var delivery WebhookDelivery

			err := rows.Scan(
				&delivery.ID,
				&delivery.WebhookID,
				&delivery.EventType,
				&delivery.TableName,
				&delivery.RecordID,
				&delivery.Payload,
				&delivery.AttemptNumber,
				&delivery.Status,
				&delivery.HTTPStatusCode,
				&delivery.ResponseBody,
				&delivery.ErrorMessage,
				&delivery.CreatedAt,
				&delivery.DeliveredAt,
				&delivery.NextRetryAt,
			)
			if err != nil {
				return err
			}

			deliveries = append(deliveries, &delivery)
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list deliveries: %w", err)
	}

	return deliveries, nil
}
