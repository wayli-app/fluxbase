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

	"github.com/fluxbase-eu/fluxbase/internal/auth"
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

	// Validate scopes
	if err := auth.ValidateScopes(scopes); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("Invalid scopes: %v", err),
		})
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

	// Validate scopes if provided
	if len(req.Scopes) > 0 {
		if err := auth.ValidateScopes(req.Scopes); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": fmt.Sprintf("Invalid scopes: %v", err),
			})
		}
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

// RevokeServiceKeyRequest represents a request to revoke a service key
type RevokeServiceKeyRequest struct {
	Reason string `json:"reason"`
}

// RevokeServiceKey immediately revokes a service key (emergency revocation)
// POST /api/v1/admin/service-keys/:id/revoke
func (h *ServiceKeyHandler) RevokeServiceKey(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return SendInvalidID(c, "service key ID")
	}

	var req RevokeServiceKeyRequest
	if err := c.BodyParser(&req); err != nil {
		return SendInvalidBody(c)
	}

	if req.Reason == "" {
		return SendMissingField(c, "reason")
	}

	// Get admin user ID from context
	adminID, ok := c.Locals("user_id").(string)
	if !ok {
		return SendUnauthorized(c, "User not authenticated", ErrCodeAuthRequired)
	}
	adminUUID, err := uuid.Parse(adminID)
	if err != nil {
		return SendUnauthorized(c, "Invalid user ID", ErrCodeInvalidUserID)
	}

	// Get the key prefix for audit log
	var keyPrefix string
	err = h.db.QueryRow(c.Context(), `
		SELECT key_prefix FROM auth.service_keys WHERE id = $1
	`, id).Scan(&keyPrefix)
	if err != nil {
		return SendResourceNotFound(c, "Service key")
	}

	// Start transaction
	tx, err := h.db.Begin(c.Context())
	if err != nil {
		return SendOperationFailed(c, "start transaction")
	}
	defer tx.Rollback(c.Context())

	// Revoke the key
	result, err := tx.Exec(c.Context(), `
		UPDATE auth.service_keys
		SET enabled = false, revoked_at = NOW(), revoked_by = $2, revocation_reason = $3
		WHERE id = $1 AND revoked_at IS NULL
	`, id, adminUUID, req.Reason)
	if err != nil {
		log.Error().Err(err).Msg("Failed to revoke service key")
		return SendOperationFailed(c, "revoke service key")
	}

	if result.RowsAffected() == 0 {
		return SendConflict(c, "Service key not found or already revoked", ErrCodeAlreadyExists)
	}

	// Log to audit table
	_, err = tx.Exec(c.Context(), `
		INSERT INTO auth.service_key_revocations (key_id, key_prefix, revoked_by, reason, revocation_type)
		VALUES ($1, $2, $3, $4, 'emergency')
	`, id, keyPrefix, adminUUID, req.Reason)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to create revocation audit log")
		// Don't fail the operation for audit log failure
	}

	if err := tx.Commit(c.Context()); err != nil {
		return SendOperationFailed(c, "commit transaction")
	}

	log.Warn().
		Str("key_id", id.String()).
		Str("key_prefix", keyPrefix).
		Str("revoked_by", adminID).
		Str("reason", req.Reason).
		Msg("Service key emergency revoked")

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Service key revoked successfully",
		"key_id":  id,
	})
}

// DeprecateServiceKeyRequest represents a request to deprecate a service key for rotation
type DeprecateServiceKeyRequest struct {
	Reason           string `json:"reason"`
	GracePeriodHours int    `json:"grace_period_hours"` // Default: 24 hours
}

// DeprecateServiceKey marks a service key for rotation with a grace period
// During the grace period, the key still works but is flagged for replacement
// POST /api/v1/admin/service-keys/:id/deprecate
func (h *ServiceKeyHandler) DeprecateServiceKey(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return SendInvalidID(c, "service key ID")
	}

	var req DeprecateServiceKeyRequest
	if err := c.BodyParser(&req); err != nil {
		return SendInvalidBody(c)
	}

	if req.Reason == "" {
		req.Reason = "Key rotation"
	}

	gracePeriod := req.GracePeriodHours
	if gracePeriod <= 0 {
		gracePeriod = 24 // Default: 24 hours
	}
	if gracePeriod > 720 { // Max 30 days
		gracePeriod = 720
	}

	graceEndTime := time.Now().Add(time.Duration(gracePeriod) * time.Hour)

	result, err := h.db.Exec(c.Context(), `
		UPDATE auth.service_keys
		SET deprecated_at = NOW(), grace_period_ends_at = $2
		WHERE id = $1 AND deprecated_at IS NULL AND revoked_at IS NULL
	`, id, graceEndTime)
	if err != nil {
		log.Error().Err(err).Msg("Failed to deprecate service key")
		return SendOperationFailed(c, "deprecate service key")
	}

	if result.RowsAffected() == 0 {
		return SendConflict(c, "Service key not found, already deprecated, or revoked", ErrCodeConflict)
	}

	log.Info().
		Str("key_id", id.String()).
		Int("grace_period_hours", gracePeriod).
		Time("grace_ends", graceEndTime).
		Str("reason", req.Reason).
		Msg("Service key deprecated for rotation")

	return c.JSON(fiber.Map{
		"success":            true,
		"message":            "Service key deprecated for rotation",
		"key_id":             id,
		"grace_period_ends":  graceEndTime,
		"grace_period_hours": gracePeriod,
	})
}

// RotateServiceKeyRequest represents a request to rotate a service key
type RotateServiceKeyRequest struct {
	GracePeriodHours int      `json:"grace_period_hours"` // Default: 24 hours
	NewKeyName       *string  `json:"new_key_name,omitempty"`
	NewScopes        []string `json:"new_scopes,omitempty"`
}

// RotateServiceKey creates a new replacement key and deprecates the old one
// POST /api/v1/admin/service-keys/:id/rotate
func (h *ServiceKeyHandler) RotateServiceKey(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return SendInvalidID(c, "service key ID")
	}

	var req RotateServiceKeyRequest
	if err := c.BodyParser(&req); err != nil {
		return SendInvalidBody(c)
	}

	// Get the old key details
	var oldKey ServiceKey
	err = h.db.QueryRow(c.Context(), `
		SELECT id, name, description, key_prefix, scopes, enabled,
		       rate_limit_per_minute, rate_limit_per_hour,
		       created_by, created_at, last_used_at, expires_at
		FROM auth.service_keys
		WHERE id = $1 AND revoked_at IS NULL
	`, id).Scan(
		&oldKey.ID, &oldKey.Name, &oldKey.Description, &oldKey.KeyPrefix, &oldKey.Scopes, &oldKey.Enabled,
		&oldKey.RateLimitPerMinute, &oldKey.RateLimitPerHour,
		&oldKey.CreatedBy, &oldKey.CreatedAt, &oldKey.LastUsedAt, &oldKey.ExpiresAt,
	)
	if err != nil {
		return SendResourceNotFound(c, "Service key")
	}

	gracePeriod := req.GracePeriodHours
	if gracePeriod <= 0 {
		gracePeriod = 24
	}
	if gracePeriod > 720 {
		gracePeriod = 720
	}

	// Get admin user ID
	createdBy, _ := c.Locals("user_id").(string)
	var createdByUUID *uuid.UUID
	if uid, err := uuid.Parse(createdBy); err == nil {
		createdByUUID = &uid
	}

	// Generate new key
	keyBytes := make([]byte, 32)
	if _, err := rand.Read(keyBytes); err != nil {
		return SendOperationFailed(c, "generate secure key")
	}
	plainKey := "sk_" + base64.URLEncoding.EncodeToString(keyBytes)
	keyHash, err := bcrypt.GenerateFromPassword([]byte(plainKey), bcrypt.DefaultCost)
	if err != nil {
		return SendOperationFailed(c, "hash key")
	}
	keyPrefix := plainKey[:16]

	// Determine new key properties
	newName := oldKey.Name + " (rotated)"
	if req.NewKeyName != nil && *req.NewKeyName != "" {
		newName = *req.NewKeyName
	}
	scopes := oldKey.Scopes
	if req.NewScopes != nil && len(req.NewScopes) > 0 {
		if err := auth.ValidateScopes(req.NewScopes); err != nil {
			return SendBadRequest(c, fmt.Sprintf("Invalid scopes: %v", err), ErrCodeValidationFailed)
		}
		scopes = req.NewScopes
	}

	graceEndTime := time.Now().Add(time.Duration(gracePeriod) * time.Hour)

	// Start transaction
	tx, err := h.db.Begin(c.Context())
	if err != nil {
		return SendOperationFailed(c, "start transaction")
	}
	defer tx.Rollback(c.Context())

	// Create new key
	var newKey ServiceKey
	err = tx.QueryRow(c.Context(), `
		INSERT INTO auth.service_keys (
			name, description, key_hash, key_prefix, scopes, enabled,
			rate_limit_per_minute, rate_limit_per_hour, created_by, expires_at
		) VALUES ($1, $2, $3, $4, $5, true, $6, $7, $8, $9)
		RETURNING id, name, description, key_prefix, scopes, enabled,
		          rate_limit_per_minute, rate_limit_per_hour,
		          created_by, created_at, last_used_at, expires_at
	`,
		newName, oldKey.Description, string(keyHash), keyPrefix, scopes,
		oldKey.RateLimitPerMinute, oldKey.RateLimitPerHour, createdByUUID, oldKey.ExpiresAt,
	).Scan(
		&newKey.ID, &newKey.Name, &newKey.Description, &newKey.KeyPrefix, &newKey.Scopes, &newKey.Enabled,
		&newKey.RateLimitPerMinute, &newKey.RateLimitPerHour,
		&newKey.CreatedBy, &newKey.CreatedAt, &newKey.LastUsedAt, &newKey.ExpiresAt,
	)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create replacement key")
		return SendOperationFailed(c, "create replacement key")
	}

	// Deprecate old key and link to new one
	_, err = tx.Exec(c.Context(), `
		UPDATE auth.service_keys
		SET deprecated_at = NOW(), grace_period_ends_at = $2, replaced_by = $3
		WHERE id = $1
	`, id, graceEndTime, newKey.ID)
	if err != nil {
		log.Error().Err(err).Msg("Failed to deprecate old key")
		return SendOperationFailed(c, "deprecate old key")
	}

	// Log rotation to audit table
	if createdByUUID != nil {
		_, _ = tx.Exec(c.Context(), `
			INSERT INTO auth.service_key_revocations (key_id, key_prefix, revoked_by, reason, revocation_type)
			VALUES ($1, $2, $3, $4, 'rotation')
		`, id, oldKey.KeyPrefix, createdByUUID, "Key rotation")
	}

	if err := tx.Commit(c.Context()); err != nil {
		return SendOperationFailed(c, "commit transaction")
	}

	log.Info().
		Str("old_key_id", id.String()).
		Str("new_key_id", newKey.ID.String()).
		Str("new_prefix", keyPrefix).
		Int("grace_period_hours", gracePeriod).
		Msg("Service key rotated")

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success":           true,
		"message":           "Service key rotated successfully",
		"old_key_id":        id,
		"grace_period_ends": graceEndTime,
		"new_key": ServiceKeyWithKey{
			ServiceKey: newKey,
			Key:        plainKey,
		},
	})
}

// GetRevocationHistory returns the revocation history for a service key
// GET /api/v1/admin/service-keys/:id/revocations
func (h *ServiceKeyHandler) GetRevocationHistory(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return SendInvalidID(c, "service key ID")
	}

	rows, err := h.db.Query(c.Context(), `
		SELECT id, key_id, key_prefix, revoked_by, reason, revocation_type, created_at
		FROM auth.service_key_revocations
		WHERE key_id = $1
		ORDER BY created_at DESC
	`, id)
	if err != nil {
		return SendOperationFailed(c, "fetch revocation history")
	}
	defer rows.Close()

	type RevocationEntry struct {
		ID             uuid.UUID  `json:"id"`
		KeyID          uuid.UUID  `json:"key_id"`
		KeyPrefix      string     `json:"key_prefix"`
		RevokedBy      *uuid.UUID `json:"revoked_by,omitempty"`
		Reason         string     `json:"reason"`
		RevocationType string     `json:"revocation_type"`
		CreatedAt      time.Time  `json:"created_at"`
	}

	var entries []RevocationEntry
	for rows.Next() {
		var entry RevocationEntry
		if err := rows.Scan(
			&entry.ID, &entry.KeyID, &entry.KeyPrefix,
			&entry.RevokedBy, &entry.Reason, &entry.RevocationType, &entry.CreatedAt,
		); err != nil {
			continue
		}
		entries = append(entries, entry)
	}

	if entries == nil {
		entries = []RevocationEntry{}
	}

	return c.JSON(entries)
}
