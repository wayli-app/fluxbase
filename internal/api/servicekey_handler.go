package api

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/bcrypt"

	"github.com/nimbleflux/fluxbase/internal/auth"
	"github.com/nimbleflux/fluxbase/internal/database"
	"github.com/nimbleflux/fluxbase/internal/middleware"
)

// ServiceKeyHandler handles service key management requests
type ServiceKeyHandler struct {
	db *database.Connection
}

// NewServiceKeyHandler creates a new service key handler
func NewServiceKeyHandler(db *database.Connection) *ServiceKeyHandler {
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
	KeyType            string     `json:"key_type"`
	Scopes             []string   `json:"scopes"`
	AllowedNamespaces  []string   `json:"allowed_namespaces,omitempty"`
	Enabled            bool       `json:"enabled"`
	RateLimitPerMinute *int       `json:"rate_limit_per_minute,omitempty"`
	RateLimitPerHour   *int       `json:"rate_limit_per_hour,omitempty"`
	CreatedBy          *uuid.UUID `json:"created_by,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	LastUsedAt         *time.Time `json:"last_used_at,omitempty"`
	ExpiresAt          *time.Time `json:"expires_at,omitempty"`
	RevokedAt          *time.Time `json:"revoked_at,omitempty"`
	DeprecatedAt       *time.Time `json:"deprecated_at,omitempty"`
	GracePeriodEndsAt  *time.Time `json:"grace_period_ends_at,omitempty"`
	ReplacedBy         *uuid.UUID `json:"replaced_by,omitempty"`
}

// ServiceKeyWithKey is returned only on creation, includes the plaintext key
type ServiceKeyWithKey struct {
	ServiceKey
	Key string `json:"key"`
}

// CreateServiceKeyRequest represents a request to create a service key
type CreateServiceKeyRequest struct {
	Name               string     `json:"name"`
	Description        *string    `json:"description,omitempty"`
	KeyType            string     `json:"key_type"`
	Scopes             []string   `json:"scopes,omitempty"`
	AllowedNamespaces  []string   `json:"allowed_namespaces,omitempty"`
	RateLimitPerMinute *int       `json:"rate_limit_per_minute,omitempty"`
	RateLimitPerHour   *int       `json:"rate_limit_per_hour,omitempty"`
	ExpiresAt          *time.Time `json:"expires_at,omitempty"`
}

// UpdateServiceKeyRequest represents a request to update a service key
type UpdateServiceKeyRequest struct {
	Name               *string  `json:"name,omitempty"`
	Description        *string  `json:"description,omitempty"`
	Scopes             []string `json:"scopes,omitempty"`
	AllowedNamespaces  []string `json:"allowed_namespaces,omitempty"`
	Enabled            *bool    `json:"enabled,omitempty"`
	RateLimitPerMinute *int     `json:"rate_limit_per_minute,omitempty"`
	RateLimitPerHour   *int     `json:"rate_limit_per_hour,omitempty"`
}

// RevokeServiceKeyRequest represents a request to revoke a service key
type RevokeServiceKeyRequest struct {
	Reason string `json:"reason"`
}

// DeprecateServiceKeyRequest represents a request to deprecate a service key
type DeprecateServiceKeyRequest struct {
	Reason           string `json:"reason"`
	GracePeriodHours int    `json:"grace_period_hours"`
}

// RotateServiceKeyRequest represents a request to rotate a service key
type RotateServiceKeyRequest struct {
	NewName          *string  `json:"new_name,omitempty"`
	NewKeyName       *string  `json:"new_key_name,omitempty"`
	NewScopes        []string `json:"new_scopes,omitempty"`
	GracePeriodHours int      `json:"grace_period_hours,omitempty"`
}

// errDBNotInitialized is returned when database connection is not available
var errDBNotInitialized = fmt.Errorf("database connection not initialized")

// checkDB checks if database connection is available.
// Always uses the main pool since auth.service_keys only exists in the main database.
func (h *ServiceKeyHandler) checkDB(c fiber.Ctx) (*pgxpool.Pool, error) {
	if h.db == nil {
		return nil, errDBNotInitialized
	}
	return h.db.Pool(), nil
}

// tenantFilterForServiceKey returns a WHERE clause fragment and args for tenant-scoped service key queries.
// Returns ("", nil) when no tenant context is available.
func tenantFilterForServiceKey(c fiber.Ctx, nextArgIdx int) (string, []interface{}) {
	if tenantID := middleware.GetTenantID(c); tenantID != "" {
		return fmt.Sprintf(" AND tenant_id = $%d", nextArgIdx), []interface{}{uuid.MustParse(tenantID)}
	}
	return "", nil
}

// ListServiceKeys lists all service keys
func (h *ServiceKeyHandler) ListServiceKeys(c fiber.Ctx) error {
	pool, err := h.checkDB(c)
	if err != nil {
		return SendInternalError(c, "Database connection not initialized")
	}

	query := `
		SELECT id, name, description, key_prefix, COALESCE(key_type, 'service'), scopes, allowed_namespaces, enabled,
		       rate_limit_per_minute, rate_limit_per_hour,
		       created_by, created_at, last_used_at, expires_at, revoked_at, deprecated_at, grace_period_ends_at, replaced_by
		FROM auth.service_keys
		WHERE 1=1
	`
	args := []interface{}{}
	argIdx := 1

	if tenantID := middleware.GetTenantID(c); tenantID != "" {
		query += fmt.Sprintf(" AND tenant_id = $%d", argIdx)
		args = append(args, tenantID)
		argIdx++
	}

	query += " ORDER BY created_at DESC"

	rows, err := pool.Query(c.RequestCtx(), query, args...)
	if err != nil {
		log.Error().Err(err).Msg("Failed to list service keys")
		return SendInternalError(c, "Failed to list service keys")
	}
	defer rows.Close()

	var keys []ServiceKey
	for rows.Next() {
		var key ServiceKey
		err := rows.Scan(
			&key.ID, &key.Name, &key.Description, &key.KeyPrefix, &key.KeyType, &key.Scopes,
			&key.AllowedNamespaces, &key.Enabled, &key.RateLimitPerMinute, &key.RateLimitPerHour,
			&key.CreatedBy, &key.CreatedAt, &key.LastUsedAt, &key.ExpiresAt,
			&key.RevokedAt, &key.DeprecatedAt, &key.GracePeriodEndsAt, &key.ReplacedBy,
		)
		if err != nil {
			log.Error().Err(err).Msg("Failed to scan service key")
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
func (h *ServiceKeyHandler) GetServiceKey(c fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return SendBadRequest(c, "Invalid service key ID", ErrCodeInvalidID)
	}

	pool, err := h.checkDB(c)
	if err != nil {
		return SendInternalError(c, "Database connection not initialized")
	}

	var key ServiceKey
	query := `
		SELECT id, name, description, key_prefix, COALESCE(key_type, 'service'), scopes, allowed_namespaces, enabled,
		       rate_limit_per_minute, rate_limit_per_hour,
		       created_by, created_at, last_used_at, expires_at, revoked_at, deprecated_at, grace_period_ends_at, replaced_by
		FROM auth.service_keys
		WHERE id = $1
	`
	args := []interface{}{id}

	if tenantID := middleware.GetTenantID(c); tenantID != "" {
		query += fmt.Sprintf(" AND tenant_id = $2")
		args = append(args, uuid.MustParse(tenantID))
	}

	err = pool.QueryRow(c.RequestCtx(), query, args...).Scan(
		&key.ID, &key.Name, &key.Description, &key.KeyPrefix, &key.KeyType, &key.Scopes,
		&key.AllowedNamespaces, &key.Enabled, &key.RateLimitPerMinute, &key.RateLimitPerHour,
		&key.CreatedBy, &key.CreatedAt, &key.LastUsedAt, &key.ExpiresAt,
		&key.RevokedAt, &key.DeprecatedAt, &key.GracePeriodEndsAt, &key.ReplacedBy,
	)
	if err != nil {
		return SendNotFound(c, "Service key not found")
	}

	return c.JSON(key)
}

// CreateServiceKey creates a new service key
func (h *ServiceKeyHandler) CreateServiceKey(c fiber.Ctx) error {
	var req CreateServiceKeyRequest
	if err := ParseBody(c, &req); err != nil {
		return err
	}

	if req.Name == "" {
		return SendBadRequest(c, "Name is required", ErrCodeMissingField)
	}

	if req.KeyType == "" {
		req.KeyType = "service"
	}

	if req.KeyType != "anon" && req.KeyType != "service" {
		return SendBadRequest(c, "key_type must be 'anon' or 'service'", ErrCodeInvalidInput)
	}

	pool, err := h.checkDB(c)
	if err != nil {
		return SendInternalError(c, "Database connection not initialized")
	}

	keyBytes := make([]byte, 32)
	if _, err := rand.Read(keyBytes); err != nil {
		return SendInternalError(c, "Failed to generate key")
	}

	prefix := "sk_live_"
	if req.KeyType == "anon" {
		prefix = "pk_live_"
	}
	fullKey := prefix + base64.URLEncoding.EncodeToString(keyBytes)
	keyPrefix := fullKey[:16]

	keyHash, err := bcrypt.GenerateFromPassword([]byte(fullKey), bcrypt.DefaultCost)
	if err != nil {
		return SendInternalError(c, "Failed to hash key")
	}

	var scopes []string
	if req.Scopes != nil {
		scopes = req.Scopes
	} else {
		if req.KeyType == "service" {
			scopes = []string{"*"}
		} else {
			scopes = []string{"read"}
		}
	}

	// created_by references auth.users(id), but dashboard users authenticate
	// against platform.users — use typed nil to ensure pgx encodes as SQL NULL.
	createdBy := (*uuid.UUID)(nil)
	var createdByUUID *uuid.UUID
	tenantID := middleware.GetTenantID(c)
	tenantUUID := uuid.Nil
	if tenantID != "" {
		tenantUUID = uuid.MustParse(tenantID)
	}

	var keyID uuid.UUID
	err = pool.QueryRow(c.RequestCtx(), `
		INSERT INTO auth.service_keys (name, description, key_hash, key_prefix, key_type, scopes, allowed_namespaces, enabled, rate_limit_per_minute, rate_limit_per_hour, created_by, expires_at, tenant_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, true, $8, $9, $10, $11, $12)
		RETURNING id
	`, req.Name, req.Description, string(keyHash), keyPrefix, req.KeyType, scopes, req.AllowedNamespaces, req.RateLimitPerMinute, req.RateLimitPerHour, createdBy, req.ExpiresAt, tenantUUID).Scan(&keyID)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create service key")
		return SendInternalError(c, "Failed to create service key")
	}

	log.Info().Str("key_id", keyID.String()).Str("key_type", req.KeyType).Str("name", req.Name).Msg("Service key created")

	return c.Status(fiber.StatusCreated).JSON(ServiceKeyWithKey{
		ServiceKey: ServiceKey{
			ID:                 keyID,
			Name:               req.Name,
			Description:        req.Description,
			KeyPrefix:          keyPrefix,
			KeyType:            req.KeyType,
			Scopes:             scopes,
			AllowedNamespaces:  req.AllowedNamespaces,
			Enabled:            true,
			RateLimitPerMinute: req.RateLimitPerMinute,
			RateLimitPerHour:   req.RateLimitPerHour,
			CreatedBy:          createdByUUID,
			CreatedAt:          time.Now(),
			ExpiresAt:          req.ExpiresAt,
		},
		Key: fullKey,
	})
}

// UpdateServiceKey updates a service key
func (h *ServiceKeyHandler) UpdateServiceKey(c fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return SendBadRequest(c, "Invalid service key ID", ErrCodeInvalidID)
	}

	var req UpdateServiceKeyRequest
	if err := ParseBody(c, &req); err != nil {
		return err
	}

	// Validate that at least one field is provided for update
	if req.Name == nil && req.Description == nil && req.Scopes == nil &&
		req.AllowedNamespaces == nil && req.Enabled == nil &&
		req.RateLimitPerMinute == nil && req.RateLimitPerHour == nil {
		return SendBadRequest(c, "No fields to update", ErrCodeInvalidInput)
	}

	pool, err := h.checkDB(c)
	if err != nil {
		return SendInternalError(c, "Database connection not initialized")
	}

	result, err := pool.Exec(c.RequestCtx(), `
		UPDATE auth.service_keys
		SET name = COALESCE($1, name),
		    description = COALESCE($2, description),
		    scopes = COALESCE($3, scopes),
		    allowed_namespaces = COALESCE($4, allowed_namespaces),
		    enabled = COALESCE($5, enabled),
		    rate_limit_per_minute = COALESCE($6, rate_limit_per_minute),
		    rate_limit_per_hour = COALESCE($7, rate_limit_per_hour),
		    updated_at = NOW()
		WHERE id = $8
	`, req.Name, req.Description, req.Scopes, req.AllowedNamespaces, req.Enabled, req.RateLimitPerMinute, req.RateLimitPerHour, id)
	if err != nil {
		log.Error().Err(err).Msg("Failed to update service key")
		return SendInternalError(c, "Failed to update service key")
	}

	if result.RowsAffected() == 0 {
		return SendNotFound(c, "Service key not found")
	}

	return h.GetServiceKey(c)
}

// DeleteServiceKey deletes a service key
func (h *ServiceKeyHandler) DeleteServiceKey(c fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return SendBadRequest(c, "Invalid service key ID", ErrCodeInvalidID)
	}

	pool, err := h.checkDB(c)
	if err != nil {
		return SendInternalError(c, "Database connection not initialized")
	}

	delQuery := `DELETE FROM auth.service_keys WHERE id = $1`
	delArgs := []interface{}{id}
	if filter, filterArgs := tenantFilterForServiceKey(c, 2); filter != "" {
		delQuery += filter
		delArgs = append(delArgs, filterArgs...)
	}
	result, err := pool.Exec(c.RequestCtx(), delQuery, delArgs...)
	if err != nil {
		log.Error().Err(err).Msg("Failed to delete service key")
		return SendInternalError(c, "Failed to delete service key")
	}

	if result.RowsAffected() == 0 {
		return SendNotFound(c, "Service key not found")
	}

	log.Info().Str("key_id", id.String()).Msg("Service key deleted")

	return c.SendStatus(fiber.StatusNoContent)
}

// DisableServiceKey disables a service key
func (h *ServiceKeyHandler) DisableServiceKey(c fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return SendBadRequest(c, "Invalid service key ID", ErrCodeInvalidID)
	}

	pool, err := h.checkDB(c)
	if err != nil {
		return SendInternalError(c, "Database connection not initialized")
	}

	disQuery := `UPDATE auth.service_keys SET enabled = false WHERE id = $1`
	disArgs := []interface{}{id}
	if filter, filterArgs := tenantFilterForServiceKey(c, 2); filter != "" {
		disQuery += filter
		disArgs = append(disArgs, filterArgs...)
	}
	result, err := pool.Exec(c.RequestCtx(), disQuery, disArgs...)
	if err != nil {
		log.Error().Err(err).Msg("Failed to disable service key")
		return SendInternalError(c, "Failed to disable service key")
	}

	if result.RowsAffected() == 0 {
		return SendNotFound(c, "Service key not found")
	}

	log.Info().Str("key_id", id.String()).Msg("Service key disabled")

	return c.SendStatus(fiber.StatusNoContent)
}

// EnableServiceKey enables a service key
func (h *ServiceKeyHandler) EnableServiceKey(c fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return SendBadRequest(c, "Invalid service key ID", ErrCodeInvalidID)
	}

	pool, err := h.checkDB(c)
	if err != nil {
		return SendInternalError(c, "Database connection not initialized")
	}

	enQuery := `UPDATE auth.service_keys SET enabled = true WHERE id = $1`
	enArgs := []interface{}{id}
	if filter, filterArgs := tenantFilterForServiceKey(c, 2); filter != "" {
		enQuery += filter
		enArgs = append(enArgs, filterArgs...)
	}
	result, err := pool.Exec(c.RequestCtx(), enQuery, enArgs...)
	if err != nil {
		log.Error().Err(err).Msg("Failed to enable service key")
		return SendInternalError(c, "Failed to enable service key")
	}

	if result.RowsAffected() == 0 {
		return SendNotFound(c, "Service key not found")
	}

	log.Info().Str("key_id", id.String()).Msg("Service key enabled")

	return c.SendStatus(fiber.StatusNoContent)
}

// RevokeServiceKey revokes a service key
func (h *ServiceKeyHandler) RevokeServiceKey(c fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return SendBadRequest(c, "Invalid service key ID", ErrCodeInvalidID)
	}

	// revoked_by references platform.users(id)
	var userID uuid.UUID
	if userIDStr := middleware.GetUserID(c); userIDStr != "" {
		userID, _ = uuid.Parse(userIDStr)
	}
	reason := c.FormValue("reason", "")
	tenantIDStr := middleware.GetTenantID(c)
	var tenantArg interface{}
	if tenantIDStr != "" {
		if tid, perr := uuid.Parse(tenantIDStr); perr == nil {
			tenantArg = tid
		}
	}
	var revokedByArg interface{}
	if userID != uuid.Nil {
		revokedByArg = userID
	}

	pool, err := h.checkDB(c)
	if err != nil {
		return SendInternalError(c, "Database connection not initialized")
	}

	tx, err := pool.Begin(c.RequestCtx())
	if err != nil {
		return SendInternalError(c, "Failed to begin transaction")
	}
	defer func() { _ = tx.Rollback(c.RequestCtx()) }()

	var keyPrefix string
	err = tx.QueryRow(c.RequestCtx(), `
		UPDATE auth.service_keys
		SET revoked_at = NOW(),
		    revoked_by = $1,
		    revocation_reason = $2,
		    enabled = false
		WHERE id = $3
		RETURNING key_prefix
	`, userID, reason, id).Scan(&keyPrefix)
	if err != nil {
		return SendNotFound(c, "Service key not found")
	}

	if _, err := tx.Exec(c.RequestCtx(), `
		INSERT INTO auth.service_key_revocations (key_id, key_prefix, revoked_by, reason, revocation_type, tenant_id)
		VALUES ($1, $2, $3, $4, 'emergency', $5)
	`, id, keyPrefix, revokedByArg, reason, tenantArg); err != nil {
		log.Error().Err(err).Msg("Failed to record service key revocation audit")
		return SendInternalError(c, "Failed to record revocation audit")
	}

	if err := tx.Commit(c.RequestCtx()); err != nil {
		return SendInternalError(c, "Failed to commit transaction")
	}

	log.Warn().Str("key_id", id.String()).Str("reason", reason).Msg("Service key revoked")

	return c.SendStatus(fiber.StatusNoContent)
}

// DeprecateServiceKey marks a service key for rotation
func (h *ServiceKeyHandler) DeprecateServiceKey(c fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return SendBadRequest(c, "Invalid service key ID", ErrCodeInvalidID)
	}

	gracePeriodHours := 24
	if gp := c.FormValue("grace_period_hours"); gp != "" {
		if _, err := fmt.Sscanf(gp, "%d", &gracePeriodHours); err != nil {
			gracePeriodHours = 24
		}
	}

	gracePeriodEndsAt := time.Now().Add(time.Duration(gracePeriodHours) * time.Hour)

	pool, err := h.checkDB(c)
	if err != nil {
		return SendInternalError(c, "Database connection not initialized")
	}

	result, err := pool.Exec(c.RequestCtx(), `
		UPDATE auth.service_keys
		SET deprecated_at = NOW(),
		    grace_period_ends_at = $1
		WHERE id = $2
	`, gracePeriodEndsAt, id)
	if err != nil {
		log.Error().Err(err).Msg("Failed to deprecate service key")
		return SendInternalError(c, "Failed to deprecate service key")
	}

	if result.RowsAffected() == 0 {
		return SendNotFound(c, "Service key not found")
	}

	log.Info().Str("key_id", id.String()).Time("grace_period_ends", gracePeriodEndsAt).Msg("Service key deprecated")

	return c.JSON(fiber.Map{
		"deprecated_at":        time.Now(),
		"grace_period_ends_at": gracePeriodEndsAt,
	})
}

// RotateServiceKey rotates a service key, creating a new one and deprecating the old
func (h *ServiceKeyHandler) RotateServiceKey(c fiber.Ctx) error {
	idStr := c.Params("id")
	oldID, err := uuid.Parse(idStr)
	if err != nil {
		return SendBadRequest(c, "Invalid service key ID", ErrCodeInvalidID)
	}

	pool, err := h.checkDB(c)
	if err != nil {
		return SendInternalError(c, "Database connection not initialized")
	}

	var oldKey ServiceKey
	err = pool.QueryRow(c.RequestCtx(), `
		SELECT id, name, description, key_prefix, COALESCE(key_type, 'service'), scopes, allowed_namespaces,
		       rate_limit_per_minute, rate_limit_per_hour, expires_at
		FROM auth.service_keys
		WHERE id = $1
	`, oldID).Scan(
		&oldKey.ID, &oldKey.Name, &oldKey.Description, &oldKey.KeyPrefix, &oldKey.KeyType,
		&oldKey.Scopes, &oldKey.AllowedNamespaces, &oldKey.RateLimitPerMinute, &oldKey.RateLimitPerHour, &oldKey.ExpiresAt,
	)
	if err != nil {
		return SendNotFound(c, "Service key not found")
	}

	keyBytes := make([]byte, 32)
	if _, err := rand.Read(keyBytes); err != nil {
		return SendInternalError(c, "Failed to generate key")
	}

	prefix := "sk_live_"
	if oldKey.KeyType == "anon" {
		prefix = "pk_live_"
	}
	fullKey := prefix + base64.URLEncoding.EncodeToString(keyBytes)
	keyPrefix := fullKey[:16]

	keyHash, err := bcrypt.GenerateFromPassword([]byte(fullKey), bcrypt.DefaultCost)
	if err != nil {
		return SendInternalError(c, "Failed to hash key")
	}

	// created_by references auth.users(id), but dashboard users authenticate
	// against platform.users — pass nil to avoid FK violation
	// Dashboard users authenticate against platform.users, not auth.users.
	// Always pass nil for created_by to avoid FK violation (column is nullable).
	var createdBy interface{}
	var createdByUUID *uuid.UUID
	tenantID := middleware.GetTenantID(c)
	tenantUUID := uuid.Nil
	if tenantID != "" {
		tenantUUID = uuid.MustParse(tenantID)
	}

	tx, err := pool.Begin(c.RequestCtx())
	if err != nil {
		return SendInternalError(c, "Failed to begin transaction")
	}
	defer func() { _ = tx.Rollback(c.RequestCtx()) }()

	var newID uuid.UUID
	err = tx.QueryRow(c.RequestCtx(), `
		INSERT INTO auth.service_keys (name, description, key_hash, key_prefix, key_type, scopes, allowed_namespaces, enabled, rate_limit_per_minute, rate_limit_per_hour, created_by, expires_at, tenant_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, true, $8, $9, $10, $11, $12)
		RETURNING id
	`, oldKey.Name+" (rotated)", oldKey.Description, string(keyHash), keyPrefix, oldKey.KeyType,
		oldKey.Scopes, oldKey.AllowedNamespaces, oldKey.RateLimitPerMinute, oldKey.RateLimitPerHour, createdBy, oldKey.ExpiresAt, tenantUUID).Scan(&newID)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create rotated service key")
		return SendInternalError(c, "Failed to create rotated key")
	}

	_, err = tx.Exec(c.RequestCtx(), `
		UPDATE auth.service_keys
		SET deprecated_at = NOW(),
		    replaced_by = $1
		WHERE id = $2
	`, newID, oldID)
	if err != nil {
		log.Error().Err(err).Msg("Failed to deprecate old service key")
		return SendInternalError(c, "Failed to deprecate old key")
	}

	// Record the rotation in the revocation audit log
	var rotUserID uuid.UUID
	if uidStr := middleware.GetUserID(c); uidStr != "" {
		rotUserID, _ = uuid.Parse(uidStr)
	}
	var rotRevokedByArg interface{}
	if rotUserID != uuid.Nil {
		rotRevokedByArg = rotUserID
	}
	var rotTenantArg interface{}
	if tenantID != "" {
		rotTenantArg = tenantUUID
	}
	if _, err := tx.Exec(c.RequestCtx(), `
		INSERT INTO auth.service_key_revocations (key_id, key_prefix, revoked_by, reason, revocation_type, tenant_id)
		VALUES ($1, $2, $3, 'rotated', 'rotation', $4)
	`, oldID, oldKey.KeyPrefix, rotRevokedByArg, rotTenantArg); err != nil {
		log.Error().Err(err).Msg("Failed to record service key rotation audit")
		return SendInternalError(c, "Failed to record rotation audit")
	}

	if err := tx.Commit(c.RequestCtx()); err != nil {
		return SendInternalError(c, "Failed to commit transaction")
	}

	log.Warn().Str("old_key_id", oldID.String()).Str("new_key_id", newID.String()).Msg("Service key rotated")

	return c.Status(fiber.StatusCreated).JSON(ServiceKeyWithKey{
		ServiceKey: ServiceKey{
			ID:                 newID,
			Name:               oldKey.Name + " (rotated)",
			Description:        oldKey.Description,
			KeyPrefix:          keyPrefix,
			KeyType:            oldKey.KeyType,
			Scopes:             oldKey.Scopes,
			AllowedNamespaces:  oldKey.AllowedNamespaces,
			Enabled:            true,
			RateLimitPerMinute: oldKey.RateLimitPerMinute,
			RateLimitPerHour:   oldKey.RateLimitPerHour,
			CreatedBy:          createdByUUID,
			CreatedAt:          time.Now(),
			ExpiresAt:          oldKey.ExpiresAt,
		},
		Key: fullKey,
	})
}

// GetRevocationHistory returns the revocation history for a service key
func (h *ServiceKeyHandler) GetRevocationHistory(c fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return SendBadRequest(c, "Invalid service key ID", ErrCodeInvalidID)
	}

	pool, err := h.checkDB(c)
	if err != nil {
		return SendInternalError(c, "Database connection not initialized")
	}

	rows, err := pool.Query(c.RequestCtx(), `
		SELECT id, key_id, key_prefix, revoked_by, reason, revocation_type, created_at, tenant_id
		FROM auth.service_key_revocations
		WHERE key_id = $1
		ORDER BY created_at DESC
	`, id)
	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch service key revocation history")
		return SendInternalError(c, "Failed to fetch revocation history")
	}
	defer rows.Close()

	type revocationEvent struct {
		ID             uuid.UUID  `json:"id"`
		KeyID          uuid.UUID  `json:"key_id"`
		KeyPrefix      string     `json:"key_prefix"`
		RevokedBy      *uuid.UUID `json:"revoked_by"`
		Reason         string     `json:"reason"`
		RevocationType string     `json:"revocation_type"`
		CreatedAt      time.Time  `json:"created_at"`
		TenantID       *uuid.UUID `json:"tenant_id"`
	}

	history := []revocationEvent{}
	for rows.Next() {
		var r revocationEvent
		var revokedBy, tenantID uuid.UUID
		if err := rows.Scan(&r.ID, &r.KeyID, &r.KeyPrefix, &revokedBy, &r.Reason, &r.RevocationType, &r.CreatedAt, &tenantID); err != nil {
			return SendInternalError(c, "Failed to scan revocation history")
		}
		if revokedBy != uuid.Nil {
			r.RevokedBy = &revokedBy
		}
		if tenantID != uuid.Nil {
			r.TenantID = &tenantID
		}
		history = append(history, r)
	}

	return c.JSON(fiber.Map{
		"key_id":  id,
		"history": history,
	})
}

var _ = auth.Service{}
