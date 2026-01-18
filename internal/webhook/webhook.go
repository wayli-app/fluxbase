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
	"net"
	"net/http"
	"net/url"
	"strings"
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
	Scope               string            `json:"scope"` // "user" or "global"
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
	ID           uuid.UUID       `json:"id"`
	WebhookID    uuid.UUID       `json:"webhook_id"`
	Event        string          `json:"event"`
	Payload      json.RawMessage `json:"payload"`
	Status       string          `json:"status"` // pending, success, failed
	StatusCode   *int            `json:"status_code,omitempty"`
	ResponseBody *string         `json:"response_body,omitempty"`
	Error        *string         `json:"error,omitempty"`
	Attempt      int             `json:"attempt"`
	DeliveredAt  *time.Time      `json:"delivered_at,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
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
	db              *database.Connection
	client          *http.Client
	AllowPrivateIPs bool // Allow private IPs for testing purposes (SSRF protection bypass)
}

// isPrivateIP checks if an IP address is in a private range
func isPrivateIP(ip net.IP) bool {
	if ip == nil {
		return false
	}

	// Check for loopback
	if ip.IsLoopback() {
		return true
	}

	// Check for link-local
	if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}

	// Check for private ranges (RFC 1918)
	privateBlocks := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"169.254.0.0/16", // AWS metadata endpoint range
		"127.0.0.0/8",    // Loopback
		"::1/128",        // IPv6 loopback
		"fc00::/7",       // IPv6 unique local
		"fe80::/10",      // IPv6 link local
	}

	for _, block := range privateBlocks {
		_, cidr, err := net.ParseCIDR(block)
		if err != nil {
			continue
		}
		if cidr.Contains(ip) {
			return true
		}
	}

	return false
}

// validateWebhookHeaders validates that custom webhook headers are safe
// This prevents HTTP header injection attacks
func validateWebhookHeaders(headers map[string]string) error {
	// Blocklist of headers that shouldn't be overridden
	blockedHeaders := map[string]bool{
		"content-length":      true,
		"host":                true,
		"transfer-encoding":   true,
		"connection":          true,
		"keep-alive":          true,
		"proxy-authenticate":  true,
		"proxy-authorization": true,
		"te":                  true,
		"trailers":            true,
		"upgrade":             true,
	}

	for key, value := range headers {
		lowerKey := strings.ToLower(key)

		// Check for blocked headers
		if blockedHeaders[lowerKey] {
			return fmt.Errorf("header '%s' is not allowed to be overridden", key)
		}

		// Check for CRLF injection in header name
		if strings.ContainsAny(key, "\r\n") {
			return fmt.Errorf("header name '%s' contains invalid characters", key)
		}

		// Check for CRLF injection in header value
		if strings.ContainsAny(value, "\r\n") {
			return fmt.Errorf("header value for '%s' contains invalid characters", key)
		}

		// Limit header value length
		if len(value) > 8192 {
			return fmt.Errorf("header value for '%s' exceeds maximum length of 8192 bytes", key)
		}
	}

	return nil
}

// validateWebhookURL validates that a webhook URL is safe to call
// This prevents SSRF attacks by blocking internal/private IP addresses
func validateWebhookURL(webhookURL string) error {
	// Parse the URL
	parsedURL, err := url.Parse(webhookURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	// Only allow HTTP and HTTPS schemes
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("URL scheme must be http or https, got: %s", parsedURL.Scheme)
	}

	// Get hostname
	hostname := parsedURL.Hostname()
	if hostname == "" {
		return fmt.Errorf("URL must have a hostname")
	}

	// Check for localhost variants
	lowerHost := strings.ToLower(hostname)
	if lowerHost == "localhost" || lowerHost == "ip6-localhost" {
		return fmt.Errorf("localhost URLs are not allowed")
	}

	// Check for common internal hostnames
	blockedHostnames := []string{
		"metadata.google.internal",
		"metadata",
		"instance-data",
		"kubernetes.default",
		"kubernetes.default.svc",
	}
	for _, blocked := range blockedHostnames {
		if lowerHost == blocked || strings.HasSuffix(lowerHost, "."+blocked) {
			return fmt.Errorf("internal hostname '%s' is not allowed", hostname)
		}
	}

	// Resolve the hostname and check if it resolves to a private IP
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	resolver := net.Resolver{}
	ips, err := resolver.LookupIPAddr(ctx, hostname)
	if err != nil {
		// If DNS lookup fails, we can't verify - block it to be safe
		return fmt.Errorf("failed to resolve hostname: %w", err)
	}

	for _, ip := range ips {
		if isPrivateIP(ip.IP) {
			return fmt.Errorf("URL resolves to private IP address %s which is not allowed", ip.IP.String())
		}
	}

	return nil
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

// parseTableReference splits a table reference into schema and table name
// e.g., "auth.users" -> ("auth", "users"), "users" -> ("auth", "users")
func parseTableReference(tableRef string) (schema, table string) {
	if idx := strings.Index(tableRef, "."); idx > 0 {
		return tableRef[:idx], tableRef[idx+1:]
	}
	// Default to auth schema since most webhook targets are auth tables
	return "auth", tableRef
}

// ManageTriggersForWebhook ensures database triggers exist for all tables monitored by this webhook
func (s *WebhookService) ManageTriggersForWebhook(ctx context.Context, events []EventConfig) error {
	for _, event := range events {
		if event.Table == "*" {
			continue // Wildcard doesn't need specific trigger
		}
		schema, table := parseTableReference(event.Table)
		if err := s.incrementTableCount(ctx, schema, table); err != nil {
			return fmt.Errorf("failed to create trigger for %s.%s: %w", schema, table, err)
		}
	}
	return nil
}

// CleanupTriggersForWebhook decrements reference counts for monitored tables
func (s *WebhookService) CleanupTriggersForWebhook(ctx context.Context, events []EventConfig) error {
	for _, event := range events {
		if event.Table == "*" {
			continue
		}
		schema, table := parseTableReference(event.Table)
		if err := s.decrementTableCount(ctx, schema, table); err != nil {
			log.Error().Err(err).Str("schema", schema).Str("table", table).Msg("Failed to decrement table count")
		}
	}
	return nil
}

// incrementTableCount calls the database function to increment webhook count for a table
func (s *WebhookService) incrementTableCount(ctx context.Context, schema, table string) error {
	query := `SELECT auth.increment_webhook_table_count($1, $2)`
	return database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, query, schema, table)
		return err
	})
}

// decrementTableCount calls the database function to decrement webhook count for a table
func (s *WebhookService) decrementTableCount(ctx context.Context, schema, table string) error {
	query := `SELECT auth.decrement_webhook_table_count($1, $2)`
	return database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, query, schema, table)
		return err
	})
}

// Create creates a new webhook
func (s *WebhookService) Create(ctx context.Context, webhook *Webhook) error {
	// Validate webhook URL to prevent SSRF attacks (skip for tests with AllowPrivateIPs)
	if !s.AllowPrivateIPs {
		if err := validateWebhookURL(webhook.URL); err != nil {
			return fmt.Errorf("invalid webhook URL: %w", err)
		}
	}

	// Validate custom headers to prevent header injection
	if err := validateWebhookHeaders(webhook.Headers); err != nil {
		return fmt.Errorf("invalid webhook headers: %w", err)
	}

	// Set default scope if not provided
	if webhook.Scope == "" {
		webhook.Scope = "user"
	}

	eventsJSON, err := json.Marshal(webhook.Events)
	if err != nil {
		return fmt.Errorf("failed to marshal events: %w", err)
	}

	headersJSON, err := json.Marshal(webhook.Headers)
	if err != nil {
		return fmt.Errorf("failed to marshal headers: %w", err)
	}

	query := `
		INSERT INTO auth.webhooks (name, description, url, secret, enabled, events, max_retries, retry_backoff_seconds, timeout_seconds, headers, scope, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
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
			webhook.Scope,
			webhook.CreatedBy,
		).Scan(&webhook.ID, &webhook.CreatedAt, &webhook.UpdatedAt)
	})

	if err != nil {
		return fmt.Errorf("failed to create webhook: %w", err)
	}

	// Create triggers for monitored tables if webhook is enabled
	if webhook.Enabled {
		if err := s.ManageTriggersForWebhook(ctx, webhook.Events); err != nil {
			log.Error().Err(err).Str("webhook_id", webhook.ID.String()).Msg("Failed to create triggers for webhook")
			// Don't fail the webhook creation, just log the error
		}
	}

	return nil
}

// List lists all webhooks
func (s *WebhookService) List(ctx context.Context) ([]*Webhook, error) {
	query := `
		SELECT id, name, description, url, secret, enabled, events, max_retries, retry_backoff_seconds, timeout_seconds, headers, scope, created_by, created_at, updated_at
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
			var scope *string

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
				&scope,
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

			// Handle NULL scope (legacy webhooks)
			if scope != nil {
				webhook.Scope = *scope
			} else {
				webhook.Scope = "user"
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
		SELECT id, name, description, url, secret, enabled, events, max_retries, retry_backoff_seconds, timeout_seconds, headers, scope, created_by, created_at, updated_at
		FROM auth.webhooks
		WHERE id = $1
	`

	var webhook Webhook
	var eventsJSON, headersJSON []byte
	var scope *string

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
			&scope,
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

	// Handle NULL scope (legacy webhooks)
	if scope != nil {
		webhook.Scope = *scope
	} else {
		webhook.Scope = "user"
	}

	return &webhook, nil
}

// Update updates a webhook
func (s *WebhookService) Update(ctx context.Context, id uuid.UUID, webhook *Webhook) error {
	// Validate webhook URL to prevent SSRF attacks (skip for tests with AllowPrivateIPs)
	if !s.AllowPrivateIPs {
		if err := validateWebhookURL(webhook.URL); err != nil {
			return fmt.Errorf("invalid webhook URL: %w", err)
		}
	}

	// Validate custom headers to prevent header injection
	if err := validateWebhookHeaders(webhook.Headers); err != nil {
		return fmt.Errorf("invalid webhook headers: %w", err)
	}

	// Get the old webhook to compare events and enabled state
	oldWebhook, err := s.Get(ctx, id)
	if err != nil {
		return err
	}

	// Set default scope if not provided
	if webhook.Scope == "" {
		webhook.Scope = "user"
	}

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
		    max_retries = $7, retry_backoff_seconds = $8, timeout_seconds = $9, headers = $10, scope = $11
		WHERE id = $12
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
			webhook.Scope,
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

	// Handle trigger changes
	// If webhook was enabled and is now disabled, decrement counts
	if oldWebhook.Enabled && !webhook.Enabled {
		if err := s.CleanupTriggersForWebhook(ctx, oldWebhook.Events); err != nil {
			log.Error().Err(err).Str("webhook_id", id.String()).Msg("Failed to cleanup triggers after disabling webhook")
		}
	}

	// If webhook was disabled and is now enabled, increment counts
	if !oldWebhook.Enabled && webhook.Enabled {
		if err := s.ManageTriggersForWebhook(ctx, webhook.Events); err != nil {
			log.Error().Err(err).Str("webhook_id", id.String()).Msg("Failed to create triggers after enabling webhook")
		}
	}

	// If webhook was and is enabled, but events changed, handle the diff
	if oldWebhook.Enabled && webhook.Enabled {
		// Build maps of old and new tables
		oldTables := make(map[string]bool)
		for _, e := range oldWebhook.Events {
			if e.Table != "*" {
				oldTables[e.Table] = true
			}
		}
		newTables := make(map[string]bool)
		for _, e := range webhook.Events {
			if e.Table != "*" {
				newTables[e.Table] = true
			}
		}

		// Decrement counts for tables no longer monitored
		for t := range oldTables {
			if !newTables[t] {
				schema, table := parseTableReference(t)
				if err := s.decrementTableCount(ctx, schema, table); err != nil {
					log.Error().Err(err).Str("table", t).Msg("Failed to decrement table count")
				}
			}
		}

		// Increment counts for newly monitored tables
		for t := range newTables {
			if !oldTables[t] {
				schema, table := parseTableReference(t)
				if err := s.incrementTableCount(ctx, schema, table); err != nil {
					log.Error().Err(err).Str("table", t).Msg("Failed to increment table count")
				}
			}
		}
	}

	return nil
}

// Delete deletes a webhook
func (s *WebhookService) Delete(ctx context.Context, id uuid.UUID) error {
	// Get the webhook first to cleanup triggers
	webhook, err := s.Get(ctx, id)
	if err != nil {
		return err
	}

	query := `DELETE FROM auth.webhooks WHERE id = $1`

	err = database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
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

	// Cleanup triggers if webhook was enabled
	if webhook.Enabled {
		if err := s.CleanupTriggersForWebhook(ctx, webhook.Events); err != nil {
			log.Error().Err(err).Str("webhook_id", id.String()).Msg("Failed to cleanup triggers after deleting webhook")
		}
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
	// SECURITY FIX: Validate webhook URL at request time to prevent DNS rebinding attacks
	// An attacker could create a webhook with a URL that initially resolves to a public IP,
	// then change the DNS to point to a private IP (e.g., 169.254.169.254 for cloud metadata)
	if !s.AllowPrivateIPs {
		if err := validateWebhookURL(webhook.URL); err != nil {
			return fmt.Errorf("webhook URL validation failed (possible DNS rebinding attack): %w", err)
		}
	}

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
	defer func() { _ = resp.Body.Close() }()

	// Check status code
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// sendWebhook sends the actual HTTP request (runs asynchronously)
func (s *WebhookService) sendWebhook(ctx context.Context, deliveryID uuid.UUID, webhook *Webhook, payloadJSON []byte) {
	// SECURITY FIX: Validate webhook URL at request time to prevent DNS rebinding attacks
	if !s.AllowPrivateIPs {
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

// generateSignature generates HMAC SHA256 signature
func (s *WebhookService) generateSignature(payload []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}

// CreateDeliveryRecord creates a delivery record before attempting delivery
func (s *WebhookService) CreateDeliveryRecord(ctx context.Context, webhookID uuid.UUID, event string, payload []byte, attempt int) (uuid.UUID, error) {
	query := `
		INSERT INTO auth.webhook_deliveries (webhook_id, event, payload, status, attempt)
		VALUES ($1, $2, $3, 'pending', $4)
		RETURNING id
	`

	var deliveryID uuid.UUID
	err := database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
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

// ListDeliveries lists webhook deliveries
func (s *WebhookService) ListDeliveries(ctx context.Context, webhookID uuid.UUID, limit int) ([]*WebhookDelivery, error) {
	query := `
		SELECT id, webhook_id, event, payload, status, status_code, response_body, error, attempt, delivered_at, created_at
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
				&delivery.Event,
				&delivery.Payload,
				&delivery.Status,
				&delivery.StatusCode,
				&delivery.ResponseBody,
				&delivery.Error,
				&delivery.Attempt,
				&delivery.DeliveredAt,
				&delivery.CreatedAt,
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
