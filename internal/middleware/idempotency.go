package middleware

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

// IdempotencyConfig configures the idempotency middleware
type IdempotencyConfig struct {
	// DB is the database connection pool
	DB *pgxpool.Pool

	// HeaderName is the header to look for idempotency keys (default: Idempotency-Key)
	HeaderName string

	// TTL is how long to keep idempotency records (default: 24h)
	TTL time.Duration

	// Methods to apply idempotency to (default: POST, PUT, DELETE, PATCH)
	Methods []string

	// PathPrefix filters which paths to apply idempotency to (default: /api/)
	PathPrefix string

	// ExcludePaths are paths to exclude from idempotency (e.g., /api/v1/auth/refresh)
	ExcludePaths []string

	// MaxKeyLength is the maximum allowed key length (default: 256)
	MaxKeyLength int

	// CleanupInterval is how often to run cleanup (default: 1h)
	CleanupInterval time.Duration
}

// DefaultIdempotencyConfig returns the default configuration
func DefaultIdempotencyConfig() IdempotencyConfig {
	return IdempotencyConfig{
		HeaderName:      "Idempotency-Key",
		TTL:             24 * time.Hour,
		Methods:         []string{"POST", "PUT", "DELETE", "PATCH"},
		PathPrefix:      "/api/",
		ExcludePaths:    []string{"/api/v1/auth/refresh", "/api/v1/auth/logout"},
		MaxKeyLength:    256,
		CleanupInterval: 1 * time.Hour,
	}
}

// IdempotencyKeyStatus represents the status of an idempotency key
type IdempotencyKeyStatus string

const (
	StatusProcessing IdempotencyKeyStatus = "processing"
	StatusCompleted  IdempotencyKeyStatus = "completed"
	StatusFailed     IdempotencyKeyStatus = "failed"
)

// IdempotencyRecord represents a stored idempotency key record
type IdempotencyRecord struct {
	Key             string               `json:"key"`
	Method          string               `json:"method"`
	Path            string               `json:"path"`
	UserID          *string              `json:"user_id,omitempty"`
	RequestHash     string               `json:"request_hash"`
	Status          IdempotencyKeyStatus `json:"status"`
	ResponseStatus  *int                 `json:"response_status,omitempty"`
	ResponseHeaders map[string]string    `json:"response_headers,omitempty"`
	ResponseBody    []byte               `json:"response_body,omitempty"`
	CreatedAt       time.Time            `json:"created_at"`
	CompletedAt     *time.Time           `json:"completed_at,omitempty"`
	ExpiresAt       time.Time            `json:"expires_at"`
}

// IdempotencyMiddleware provides idempotency key handling for safe request retries
type IdempotencyMiddleware struct {
	config      IdempotencyConfig
	methodSet   map[string]bool
	excludeSet  map[string]bool
	stopCleanup chan struct{}
}

// NewIdempotencyMiddleware creates a new idempotency middleware
func NewIdempotencyMiddleware(config IdempotencyConfig) *IdempotencyMiddleware {
	if config.HeaderName == "" {
		config.HeaderName = "Idempotency-Key"
	}
	if config.TTL == 0 {
		config.TTL = 24 * time.Hour
	}
	if len(config.Methods) == 0 {
		config.Methods = []string{"POST", "PUT", "DELETE", "PATCH"}
	}
	if config.MaxKeyLength == 0 {
		config.MaxKeyLength = 256
	}
	if config.CleanupInterval == 0 {
		config.CleanupInterval = 1 * time.Hour
	}

	methodSet := make(map[string]bool)
	for _, m := range config.Methods {
		methodSet[strings.ToUpper(m)] = true
	}

	excludeSet := make(map[string]bool)
	for _, p := range config.ExcludePaths {
		excludeSet[p] = true
	}

	mw := &IdempotencyMiddleware{
		config:      config,
		methodSet:   methodSet,
		excludeSet:  excludeSet,
		stopCleanup: make(chan struct{}),
	}

	// Start cleanup goroutine if DB is configured
	if config.DB != nil {
		go mw.cleanupLoop()
	}

	return mw
}

// Stop stops the cleanup goroutine
func (m *IdempotencyMiddleware) Stop() {
	close(m.stopCleanup)
}

// cleanupLoop periodically removes expired idempotency keys
func (m *IdempotencyMiddleware) cleanupLoop() {
	ticker := time.NewTicker(m.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.cleanupExpiredKeys()
		case <-m.stopCleanup:
			return
		}
	}
}

// cleanupExpiredKeys removes expired idempotency keys from the database
func (m *IdempotencyMiddleware) cleanupExpiredKeys() {
	if m.config.DB == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := m.config.DB.Exec(ctx,
		"DELETE FROM api.idempotency_keys WHERE expires_at < NOW()")
	if err != nil {
		log.Error().Err(err).Msg("Failed to cleanup expired idempotency keys")
		return
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected > 0 {
		log.Debug().Int64("deleted", rowsAffected).Msg("Cleaned up expired idempotency keys")
	}
}

// Middleware returns a Fiber middleware handler
func (m *IdempotencyMiddleware) Middleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Check if idempotency should be applied
		if !m.shouldApply(c) {
			return c.Next()
		}

		// Get idempotency key from header
		key := c.Get(m.config.HeaderName)
		if key == "" {
			// No key provided - process normally
			return c.Next()
		}

		// Validate key
		if len(key) > m.config.MaxKeyLength {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":   "Invalid Idempotency-Key",
				"code":    "IDEMPOTENCY_KEY_TOO_LONG",
				"message": fmt.Sprintf("Idempotency key exceeds maximum length of %d characters", m.config.MaxKeyLength),
			})
		}

		// Get user ID from context if available
		var userID *string
		if uid, ok := c.Locals("user_id").(string); ok && uid != "" {
			userID = &uid
		}

		// Calculate request hash for validation
		requestHash := m.calculateRequestHash(c)

		// Check if key exists
		record, err := m.getRecord(c.Context(), key)
		if err != nil && err != pgx.ErrNoRows {
			log.Error().Err(err).Str("key", key).Msg("Failed to check idempotency key")
			// Continue without idempotency on DB error to avoid blocking
			return c.Next()
		}

		if record != nil {
			// Key exists - handle based on status
			return m.handleExistingKey(c, record, requestHash)
		}

		// Key doesn't exist - create new record and process
		if err := m.createRecord(c.Context(), key, c.Method(), c.Path(), userID, requestHash); err != nil {
			log.Error().Err(err).Str("key", key).Msg("Failed to create idempotency record")
			// Continue without idempotency on DB error
			return c.Next()
		}

		// Create response capture
		originalBody := c.Response().Body()

		// Process the request
		err = c.Next()

		// Capture response
		responseStatus := c.Response().StatusCode()
		responseBody := c.Response().Body()
		responseHeaders := make(map[string]string)
		c.Response().Header.VisitAll(func(key, value []byte) {
			responseHeaders[string(key)] = string(value)
		})

		// Update record with response
		status := StatusCompleted
		if err != nil || responseStatus >= 500 {
			status = StatusFailed
		}

		if updateErr := m.updateRecord(c.Context(), key, status, responseStatus, responseHeaders, responseBody); updateErr != nil {
			log.Error().Err(updateErr).Str("key", key).Msg("Failed to update idempotency record")
		}

		// Restore original body if it was modified
		if len(originalBody) > 0 && len(responseBody) == 0 {
			c.Response().SetBody(originalBody)
		}

		return err
	}
}

// shouldApply checks if idempotency should be applied to this request
func (m *IdempotencyMiddleware) shouldApply(c *fiber.Ctx) bool {
	// Check method
	if !m.methodSet[c.Method()] {
		return false
	}

	// Check path prefix
	path := c.Path()
	if m.config.PathPrefix != "" && !strings.HasPrefix(path, m.config.PathPrefix) {
		return false
	}

	// Check exclusions
	if m.excludeSet[path] {
		return false
	}

	// Check if DB is configured
	if m.config.DB == nil {
		return false
	}

	return true
}

// calculateRequestHash creates a hash of the request body for validation
func (m *IdempotencyMiddleware) calculateRequestHash(c *fiber.Ctx) string {
	body := c.Body()
	if len(body) == 0 {
		return ""
	}

	hash := sha256.Sum256(body)
	return hex.EncodeToString(hash[:])
}

// getRecord retrieves an existing idempotency record
func (m *IdempotencyMiddleware) getRecord(ctx context.Context, key string) (*IdempotencyRecord, error) {
	var record IdempotencyRecord
	var responseHeaders []byte

	err := m.config.DB.QueryRow(ctx, `
		SELECT key, method, path, user_id, request_hash, status,
		       response_status, response_headers, response_body,
		       created_at, completed_at, expires_at
		FROM api.idempotency_keys
		WHERE key = $1 AND expires_at > NOW()
	`, key).Scan(
		&record.Key,
		&record.Method,
		&record.Path,
		&record.UserID,
		&record.RequestHash,
		&record.Status,
		&record.ResponseStatus,
		&responseHeaders,
		&record.ResponseBody,
		&record.CreatedAt,
		&record.CompletedAt,
		&record.ExpiresAt,
	)

	if err != nil {
		return nil, err
	}

	// Unmarshal response headers
	if len(responseHeaders) > 0 {
		if err := json.Unmarshal(responseHeaders, &record.ResponseHeaders); err != nil {
			log.Warn().Err(err).Str("key", key).Msg("Failed to unmarshal response headers")
		}
	}

	return &record, nil
}

// createRecord creates a new idempotency record
func (m *IdempotencyMiddleware) createRecord(ctx context.Context, key, method, path string, userID *string, requestHash string) error {
	_, err := m.config.DB.Exec(ctx, `
		INSERT INTO api.idempotency_keys (key, method, path, user_id, request_hash, status, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW() + $7::interval)
		ON CONFLICT (key) DO NOTHING
	`, key, method, path, userID, requestHash, StatusProcessing, fmt.Sprintf("%d seconds", int(m.config.TTL.Seconds())))

	return err
}

// updateRecord updates an idempotency record with the response
func (m *IdempotencyMiddleware) updateRecord(ctx context.Context, key string, status IdempotencyKeyStatus, responseStatus int, responseHeaders map[string]string, responseBody []byte) error {
	headersJSON, err := json.Marshal(responseHeaders)
	if err != nil {
		headersJSON = []byte("{}")
	}

	_, err = m.config.DB.Exec(ctx, `
		UPDATE api.idempotency_keys
		SET status = $2,
		    response_status = $3,
		    response_headers = $4,
		    response_body = $5,
		    completed_at = NOW()
		WHERE key = $1
	`, key, status, responseStatus, headersJSON, responseBody)

	return err
}

// handleExistingKey handles a request with an existing idempotency key
func (m *IdempotencyMiddleware) handleExistingKey(c *fiber.Ctx, record *IdempotencyRecord, requestHash string) error {
	// Verify request matches original (same method, path, body hash)
	if record.Method != c.Method() || record.Path != c.Path() {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"error":   "Idempotency Key Mismatch",
			"code":    "IDEMPOTENCY_KEY_MISMATCH",
			"message": "This idempotency key was used for a different request",
			"details": fiber.Map{
				"original_method": record.Method,
				"original_path":   record.Path,
				"current_method":  c.Method(),
				"current_path":    c.Path(),
			},
		})
	}

	// Optionally verify request body hash
	if record.RequestHash != "" && requestHash != "" && record.RequestHash != requestHash {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"error":   "Idempotency Key Mismatch",
			"code":    "IDEMPOTENCY_REQUEST_BODY_MISMATCH",
			"message": "This idempotency key was used with a different request body",
		})
	}

	switch record.Status {
	case StatusProcessing:
		// Request is still being processed
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error":   "Request In Progress",
			"code":    "IDEMPOTENCY_REQUEST_IN_PROGRESS",
			"message": "A request with this idempotency key is currently being processed",
			"hint":    "Please wait and retry in a few seconds",
		})

	case StatusCompleted:
		// Return cached response
		log.Debug().
			Str("key", record.Key).
			Int("status", *record.ResponseStatus).
			Msg("Returning cached idempotent response")

		// Set Idempotency-Replayed header
		c.Set("Idempotency-Replayed", "true")

		// Restore response headers
		for key, value := range record.ResponseHeaders {
			// Skip certain headers that shouldn't be replayed
			if key == "Content-Length" || key == "Date" {
				continue
			}
			c.Set(key, value)
		}

		// Return cached response
		c.Status(*record.ResponseStatus)
		return c.Send(record.ResponseBody)

	case StatusFailed:
		// Previous request failed - allow retry
		// Delete the old record and process as new
		if _, err := m.config.DB.Exec(c.Context(),
			"DELETE FROM api.idempotency_keys WHERE key = $1", record.Key); err != nil {
			log.Warn().Err(err).Str("key", record.Key).Msg("Failed to delete failed idempotency record")
		}
		return c.Next()

	default:
		// Unknown status - process normally
		return c.Next()
	}
}

// IdempotencyMiddlewareFunc creates a simple middleware function from config
func IdempotencyMiddlewareFunc(db *pgxpool.Pool) fiber.Handler {
	config := DefaultIdempotencyConfig()
	config.DB = db
	mw := NewIdempotencyMiddleware(config)
	return mw.Middleware()
}

// IdempotencyKeyReplayedHeader is the header set when returning a cached response
const IdempotencyKeyReplayedHeader = "Idempotency-Replayed"

// GetIdempotencyKey extracts the idempotency key from a request
func GetIdempotencyKey(c *fiber.Ctx) string {
	return c.Get("Idempotency-Key")
}

// HasIdempotencyKey checks if the request has an idempotency key
func HasIdempotencyKey(c *fiber.Ctx) bool {
	return c.Get("Idempotency-Key") != ""
}

// IsReplayedResponse checks if the response was replayed from cache
func IsReplayedResponse(c *fiber.Ctx) bool {
	return c.Get(IdempotencyKeyReplayedHeader) == "true"
}

// EncodeResponseBody encodes binary response body for storage
func EncodeResponseBody(body []byte) string {
	return base64.StdEncoding.EncodeToString(body)
}

// DecodeResponseBody decodes stored response body
func DecodeResponseBody(encoded string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(encoded)
}

// CompareBytes safely compares two byte slices
func CompareBytes(a, b []byte) bool {
	return bytes.Equal(a, b)
}
