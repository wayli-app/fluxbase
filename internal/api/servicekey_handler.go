package api

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/bcrypt"
)

// ServiceKeyHandler handles service key management requests
type ServiceKeyHandler struct {
	db *pgxpool.Pool
}

// NewServiceKeyHandler creates a new service key handler
func NewServiceKeyHandler(db *pgxpool.Pool) *ServiceKeyHandler {
	return &ServiceKeyHandler{
		db: db,
	}
}

// ServiceKey represents a service key in the database
type ServiceKey struct {
	ID                 uuid.UUID  `json:"id"`
	Name               string     `json:"name"`
	Description        *string    `json:"description,omitempty"`
	KeyPrefix          string     `json:"key_prefix"`
	Scopes             []string   `json:"scopes"`
	Enabled            bool       `json:"enabled"`
	RateLimitPerMinute *int       `json:"rate_limit_per_minute,omitempty"`
	RateLimitPerHour   *int       `json:"rate_limit_per_hour,omitempty"`
	CreatedBy          *uuid.UUID `json:"created_by,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	LastUsedAt         *time.Time `json:"last_used_at,omitempty"`
	ExpiresAt          *time.Time `json:"expires_at,omitempty"`
}

// ServiceKeyWithKey is returned only on creation, includes the plaintext key
type ServiceKeyWithKey struct {
	ServiceKey
	Key string `json:"key"` // Only returned on creation
}

// CreateServiceKeyRequest represents a request to create a service key
type CreateServiceKeyRequest struct {
	Name               string     `json:"name"`
	Description        *string    `json:"description,omitempty"`
	Scopes             []string   `json:"scopes,omitempty"`
	RateLimitPerMinute *int       `json:"rate_limit_per_minute,omitempty"`
	RateLimitPerHour   *int       `json:"rate_limit_per_hour,omitempty"`
	ExpiresAt          *time.Time `json:"expires_at,omitempty"`
}

// UpdateServiceKeyRequest represents a request to update a service key
type UpdateServiceKeyRequest struct {
	Name               *string  `json:"name,omitempty"`
	Description        *string  `json:"description,omitempty"`
	Scopes             []string `json:"scopes,omitempty"`
	Enabled            *bool    `json:"enabled,omitempty"`
	RateLimitPerMinute *int     `json:"rate_limit_per_minute,omitempty"`
	RateLimitPerHour   *int     `json:"rate_limit_per_hour,omitempty"`
}

// ListServiceKeys lists all service keys
func (h *ServiceKeyHandler) ListServiceKeys(c *fiber.Ctx) error {
	rows, err := h.db.Query(c.Context(), `
		SELECT id, name, description, key_prefix, scopes, enabled,
		       rate_limit_per_minute, rate_limit_per_hour,
		       created_by, created_at, last_used_at, expires_at
		FROM auth.service_keys
		ORDER BY created_at DESC
	`)
	if err != nil {
		log.Error().Err(err).Msg("Failed to list service keys")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to list service keys",
		})
	}
	defer rows.Close()

	var keys []ServiceKey
	for rows.Next() {
		var key ServiceKey
		err := rows.Scan(
			&key.ID, &key.Name, &key.Description, &key.KeyPrefix, &key.Scopes, &key.Enabled,
			&key.RateLimitPerMinute, &key.RateLimitPerHour,
			&key.CreatedBy, &key.CreatedAt, &key.LastUsedAt, &key.ExpiresAt,
		)
		if err != nil {
			log.Error().Err(err).Msg("Failed to scan service key row")
			continue
		}
		keys = append(keys, key)
	}

	if keys == nil {
		keys = []ServiceKey{}
	}

	return c.JSON(keys)
}

// GetServiceKey retrieves a single service key
func (h *ServiceKeyHandler) GetServiceKey(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid service key ID",
		})
	}

	var key ServiceKey
	err = h.db.QueryRow(c.Context(), `
		SELECT id, name, description, key_prefix, scopes, enabled,
		       rate_limit_per_minute, rate_limit_per_hour,
		       created_by, created_at, last_used_at, expires_at
		FROM auth.service_keys
		WHERE id = $1
	`, id).Scan(
		&key.ID, &key.Name, &key.Description, &key.KeyPrefix, &key.Scopes, &key.Enabled,
		&key.RateLimitPerMinute, &key.RateLimitPerHour,
		&key.CreatedBy, &key.CreatedAt, &key.LastUsedAt, &key.ExpiresAt,
	)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Service key not found",
		})
	}

	return c.JSON(key)
}

// CreateServiceKey creates a new service key
func (h *ServiceKeyHandler) CreateServiceKey(c *fiber.Ctx) error {
	var req CreateServiceKeyRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Name is required",
		})
	}

	// Generate a secure random key
	keyBytes := make([]byte, 32)
	if _, err := rand.Read(keyBytes); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to generate key",
		})
	}
	randomPart := base64.RawURLEncoding.EncodeToString(keyBytes)
	plainKey := fmt.Sprintf("sk_%s", randomPart)
	keyPrefix := plainKey[:16]

	// Hash the key
	keyHash, err := bcrypt.GenerateFromPassword([]byte(plainKey), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to hash key",
		})
	}

	// Default scopes if not provided
	scopes := req.Scopes
	if scopes == nil {
		scopes = []string{"*"}
	}

	// Get creator user ID if available
	var createdBy *uuid.UUID
	if userID, ok := c.Locals("user_id").(uuid.UUID); ok {
		createdBy = &userID
	}

	// Insert the key
	var key ServiceKey
	err = h.db.QueryRow(c.Context(), `
		INSERT INTO auth.service_keys (
			name, description, key_hash, key_prefix, scopes, enabled,
			rate_limit_per_minute, rate_limit_per_hour, created_by, expires_at
		) VALUES ($1, $2, $3, $4, $5, true, $6, $7, $8, $9)
		RETURNING id, name, description, key_prefix, scopes, enabled,
		          rate_limit_per_minute, rate_limit_per_hour,
		          created_by, created_at, last_used_at, expires_at
	`,
		req.Name, req.Description, string(keyHash), keyPrefix, scopes,
		req.RateLimitPerMinute, req.RateLimitPerHour, createdBy, req.ExpiresAt,
	).Scan(
		&key.ID, &key.Name, &key.Description, &key.KeyPrefix, &key.Scopes, &key.Enabled,
		&key.RateLimitPerMinute, &key.RateLimitPerHour,
		&key.CreatedBy, &key.CreatedAt, &key.LastUsedAt, &key.ExpiresAt,
	)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create service key")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to create service key: %v", err),
		})
	}

	// Return the key with the plaintext (only time it's shown)
	result := ServiceKeyWithKey{
		ServiceKey: key,
		Key:        plainKey,
	}

	log.Info().
		Str("key_id", key.ID.String()).
		Str("name", key.Name).
		Str("prefix", keyPrefix).
		Msg("Service key created")

	return c.Status(fiber.StatusCreated).JSON(result)
}

// UpdateServiceKey updates a service key
func (h *ServiceKeyHandler) UpdateServiceKey(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid service key ID",
		})
	}

	var req UpdateServiceKeyRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Build dynamic update query
	updates := make(map[string]interface{})
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.Scopes != nil {
		updates["scopes"] = req.Scopes
	}
	if req.Enabled != nil {
		updates["enabled"] = *req.Enabled
	}
	if req.RateLimitPerMinute != nil {
		updates["rate_limit_per_minute"] = *req.RateLimitPerMinute
	}
	if req.RateLimitPerHour != nil {
		updates["rate_limit_per_hour"] = *req.RateLimitPerHour
	}

	if len(updates) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "No fields to update",
		})
	}

	// Build SET clause dynamically
	setClause := ""
	args := make([]interface{}, 0, len(updates)+1)
	i := 1
	for field, value := range updates {
		if i > 1 {
			setClause += ", "
		}
		setClause += fmt.Sprintf("%s = $%d", field, i)
		args = append(args, value)
		i++
	}
	args = append(args, id)

	query := fmt.Sprintf(`
		UPDATE auth.service_keys
		SET %s
		WHERE id = $%d
	`, setClause, i)

	result, err := h.db.Exec(c.Context(), query, args...)
	if err != nil {
		log.Error().Err(err).Msg("Failed to update service key")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update service key",
		})
	}

	if result.RowsAffected() == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Service key not found",
		})
	}

	log.Info().
		Str("key_id", id.String()).
		Msg("Service key updated")

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Service key updated successfully",
	})
}

// DeleteServiceKey deletes a service key
func (h *ServiceKeyHandler) DeleteServiceKey(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid service key ID",
		})
	}

	result, err := h.db.Exec(c.Context(), `
		DELETE FROM auth.service_keys WHERE id = $1
	`, id)
	if err != nil {
		log.Error().Err(err).Msg("Failed to delete service key")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete service key",
		})
	}

	if result.RowsAffected() == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Service key not found",
		})
	}

	log.Info().
		Str("key_id", id.String()).
		Msg("Service key deleted")

	return c.Status(fiber.StatusNoContent).Send(nil)
}

// DisableServiceKey disables a service key without deleting it
func (h *ServiceKeyHandler) DisableServiceKey(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid service key ID",
		})
	}

	result, err := h.db.Exec(c.Context(), `
		UPDATE auth.service_keys SET enabled = false WHERE id = $1
	`, id)
	if err != nil {
		log.Error().Err(err).Msg("Failed to disable service key")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to disable service key",
		})
	}

	if result.RowsAffected() == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Service key not found",
		})
	}

	log.Info().
		Str("key_id", id.String()).
		Msg("Service key disabled")

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Service key disabled successfully",
	})
}

// EnableServiceKey enables a service key
func (h *ServiceKeyHandler) EnableServiceKey(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid service key ID",
		})
	}

	result, err := h.db.Exec(c.Context(), `
		UPDATE auth.service_keys SET enabled = true WHERE id = $1
	`, id)
	if err != nil {
		log.Error().Err(err).Msg("Failed to enable service key")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to enable service key",
		})
	}

	if result.RowsAffected() == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Service key not found",
		})
	}

	log.Info().
		Str("key_id", id.String()).
		Msg("Service key enabled")

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Service key enabled successfully",
	})
}
