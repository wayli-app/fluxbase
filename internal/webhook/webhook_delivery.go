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

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/database"
)

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
	// SECURITY FIX: Validate webhook URL at request time to prevent DNS rebinding attacks
	// An attacker could create a webhook with a URL that initially resolves to a public IP,
	// then change the DNS to point to a private IP (e.g., 169.254.169.254 for cloud metadata)
	if !s.AllowPrivateIPs() {
		if err := validateWebhookURL(webhook.URL); err != nil {
			return fmt.Errorf("webhook URL validation failed (possible DNS rebinding attack): %w", err)
		}
	}

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

	// Add HMAC signature with timestamp if secret is provided
	// Format: t=timestamp,v1=signature
	// This enables replay protection and is similar to Stripe's webhook signing
	if webhook.Secret != nil && *webhook.Secret != "" {
		timestamp := time.Now().Unix()
		signature := generateTimestampedSignature(payloadJSON, *webhook.Secret, timestamp)
		signatureHeader := fmt.Sprintf("t=%d,v1=%s", timestamp, signature)
		req.Header.Set("X-Fluxbase-Signature", signatureHeader)
		// Also keep legacy header for backwards compatibility
		legacySignature := s.generateSignature(payloadJSON, *webhook.Secret)
		req.Header.Set("X-Webhook-Signature", legacySignature)
	}

	// Send request with timeout
	client := &http.Client{
		Timeout: time.Duration(webhook.TimeoutSeconds) * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Check status code
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// sendWebhook sends the actual HTTP request (runs asynchronously).
// Note: Currently unused but kept for potential future async webhook delivery implementation.
/*
func (s *WebhookService) sendWebhook(ctx context.Context, deliveryID uuid.UUID, webhook *Webhook, payloadJSON []byte) {
	// SECURITY FIX: Validate webhook URL at request time to prevent DNS rebinding attacks
	if !s.AllowPrivateIPs() {
		if err := validateWebhookURL(webhook.URL); err != nil {
			s.markDeliveryFailed(ctx, deliveryID, 0, nil, fmt.Sprintf("webhook URL validation failed (possible DNS rebinding): %v", err))
			return
		}
	}

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

	// Add HMAC signature with timestamp if secret is provided
	if webhook.Secret != nil && *webhook.Secret != "" {
		timestamp := time.Now().Unix()
		signature := generateTimestampedSignature(payloadJSON, *webhook.Secret, timestamp)
		signatureHeader := fmt.Sprintf("t=%d,v1=%s", timestamp, signature)
		req.Header.Set("X-Fluxbase-Signature", signatureHeader)
		// Also keep legacy header for backwards compatibility
		legacySignature := s.generateSignature(payloadJSON, *webhook.Secret)
		req.Header.Set("X-Webhook-Signature", legacySignature)
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
	defer func() { _ = resp.Body.Close() }()

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
*/

// generateSignature generates HMAC SHA256 signature (legacy, without timestamp)
func (s *WebhookService) generateSignature(payload []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}

// generateTimestampedSignature generates HMAC SHA256 signature with timestamp
// The signature is computed over: timestamp + "." + payload
// This prevents replay attacks by including the timestamp in the signed data
func generateTimestampedSignature(payload []byte, secret string, timestamp int64) string {
	// Create the signed payload: timestamp.payload
	signedPayload := fmt.Sprintf("%d.%s", timestamp, string(payload))

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signedPayload))
	return hex.EncodeToString(mac.Sum(nil))
}

// CreateDeliveryRecord creates a delivery record before attempting delivery
func (s *WebhookService) CreateDeliveryRecord(ctx context.Context, webhookID uuid.UUID, event string, payload []byte, attempt int, tenantID string) (uuid.UUID, error) {
	query := `
		INSERT INTO auth.webhook_deliveries (webhook_id, event, payload, status, attempt)
		VALUES ($1, $2, $3, 'pending', $4)
		RETURNING id
	`

	var deliveryID uuid.UUID
	err := database.WrapWithServiceRoleAndTenant(ctx, s.db, tenantID, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, query, webhookID, event, payload, attempt).Scan(&deliveryID)
	})
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to create delivery record: %w", err)
	}

	return deliveryID, nil
}

// markDeliverySuccess marks a delivery as successful
func (s *WebhookService) markDeliverySuccess(ctx context.Context, deliveryID uuid.UUID, statusCode int, responseBody *string) {
	query := `
		UPDATE auth.webhook_deliveries
		SET status = 'success', status_code = $1, response_body = $2, delivered_at = NOW()
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
		SET status = 'failed', status_code = $1, response_body = $2, error = $3
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
