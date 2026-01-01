package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

var (
	// ErrInvalidClientKey is returned when client key is invalid
	ErrInvalidClientKey = errors.New("invalid client key")
	// ErrClientKeyExpired is returned when client key has expired
	ErrClientKeyExpired = errors.New("client key has expired")
	// ErrClientKeyRevoked is returned when client key has been revoked
	ErrClientKeyRevoked = errors.New("client key has been revoked")
	// ErrUserClientKeysDisabled is returned when user client keys are disabled via settings
	ErrUserClientKeysDisabled = errors.New("user client keys are disabled")
)

// ClientKey represents a client key
type ClientKey struct {
	ID                 uuid.UUID  `json:"id"`
	Name               string     `json:"name"`
	Description        *string    `json:"description,omitempty"`
	KeyHash            string     `json:"-"` // Never expose the hash
	KeyPrefix          string     `json:"key_prefix"`
	UserID             *uuid.UUID `json:"user_id,omitempty"`
	Scopes             []string   `json:"scopes"`
	AllowedNamespaces  []string   `json:"allowed_namespaces,omitempty"` // nil = all namespaces, empty = default only
	RateLimitPerMinute int        `json:"rate_limit_per_minute"`
	LastUsedAt         *time.Time `json:"last_used_at,omitempty"`
	ExpiresAt          *time.Time `json:"expires_at,omitempty"`
	RevokedAt          *time.Time `json:"revoked_at,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

// ClientKeyWithPlaintext includes the plaintext key (only returned once during creation)
type ClientKeyWithPlaintext struct {
	ClientKey
	PlaintextKey string `json:"key"` // Full key, only shown once
}

// ClientKeyService handles client key operations
type ClientKeyService struct {
	db            *pgxpool.Pool
	settingsCache *SettingsCache
}

// NewClientKeyService creates a new client key service
func NewClientKeyService(db *pgxpool.Pool, settingsCache *SettingsCache) *ClientKeyService {
	return &ClientKeyService{
		db:            db,
		settingsCache: settingsCache,
	}
}

// SetSettingsCache injects the settings cache after initialization
// This is used to break the circular dependency during server startup
func (s *ClientKeyService) SetSettingsCache(cache *SettingsCache) {
	s.settingsCache = cache
}

// GenerateClientKey generates a new client key with format: fbk_<random_string>
func (s *ClientKeyService) GenerateClientKey(ctx context.Context, name string, description *string, userID *uuid.UUID, scopes []string, rateLimitPerMinute int, expiresAt *time.Time) (*ClientKeyWithPlaintext, error) {
	// Generate random bytes for the key
	keyBytes := make([]byte, 32)
	if _, err := rand.Read(keyBytes); err != nil {
		return nil, fmt.Errorf("failed to generate random key: %w", err)
	}

	// Create the plaintext key with prefix
	plaintextKey := "fbk_" + base64.URLEncoding.EncodeToString(keyBytes)

	// Hash the key for storage
	keyHash := hashClientKey(plaintextKey)

	// Extract prefix (first 12 chars: "fbk_" + 8 chars)
	keyPrefix := plaintextKey[:12]

	// Validate scopes - at least one scope is required
	if err := ValidateScopes(scopes); err != nil {
		return nil, fmt.Errorf("invalid scopes: %w", err)
	}

	// Set default rate limit
	if rateLimitPerMinute == 0 {
		rateLimitPerMinute = 100
	}

	// Insert into database
	var clientKey ClientKey
	query := `
		INSERT INTO auth.client_keys (name, description, key_hash, key_prefix, user_id, scopes, rate_limit_per_minute, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, name, description, key_hash, key_prefix, user_id, scopes, rate_limit_per_minute, last_used_at, expires_at, revoked_at, created_at, updated_at
	`

	err := s.db.QueryRow(ctx, query, name, description, keyHash, keyPrefix, userID, scopes, rateLimitPerMinute, expiresAt).Scan(
		&clientKey.ID,
		&clientKey.Name,
		&clientKey.Description,
		&clientKey.KeyHash,
		&clientKey.KeyPrefix,
		&clientKey.UserID,
		&clientKey.Scopes,
		&clientKey.RateLimitPerMinute,
		&clientKey.LastUsedAt,
		&clientKey.ExpiresAt,
		&clientKey.RevokedAt,
		&clientKey.CreatedAt,
		&clientKey.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create client key: %w", err)
	}

	return &ClientKeyWithPlaintext{
		ClientKey:    clientKey,
		PlaintextKey: plaintextKey,
	}, nil
}

// ValidateClientKey validates a client key and returns the associated client key info
func (s *ClientKeyService) ValidateClientKey(ctx context.Context, plaintextKey string) (*ClientKey, error) {
	// Hash the provided key
	keyHash := hashClientKey(plaintextKey)

	// Query the database
	var clientKey ClientKey
	query := `
		SELECT id, name, description, key_hash, key_prefix, user_id, scopes, rate_limit_per_minute, last_used_at, expires_at, revoked_at, created_at, updated_at
		FROM auth.client_keys
		WHERE key_hash = $1
	`

	err := s.db.QueryRow(ctx, query, keyHash).Scan(
		&clientKey.ID,
		&clientKey.Name,
		&clientKey.Description,
		&clientKey.KeyHash,
		&clientKey.KeyPrefix,
		&clientKey.UserID,
		&clientKey.Scopes,
		&clientKey.RateLimitPerMinute,
		&clientKey.LastUsedAt,
		&clientKey.ExpiresAt,
		&clientKey.RevokedAt,
		&clientKey.CreatedAt,
		&clientKey.UpdatedAt,
	)
	if err != nil {
		return nil, ErrInvalidClientKey
	}

	// Check if revoked
	if clientKey.RevokedAt != nil {
		return nil, ErrClientKeyRevoked
	}

	// Check if expired
	if clientKey.ExpiresAt != nil && clientKey.ExpiresAt.Before(time.Now()) {
		return nil, ErrClientKeyExpired
	}

	// Check if user-created keys are allowed
	// Keys with user_id are user-created; keys without user_id are admin/system-created
	if clientKey.UserID != nil && s.settingsCache != nil {
		allowUserKeys := s.settingsCache.GetBool(ctx, "app.auth.allow_user_client_keys", true)
		if !allowUserKeys {
			return nil, ErrUserClientKeysDisabled
		}
	}

	// Update last used timestamp
	now := time.Now()
	_, err = s.db.Exec(ctx, "UPDATE auth.client_keys SET last_used_at = $1 WHERE id = $2", now, clientKey.ID)
	if err != nil {
		// Log but don't fail validation
		log.Warn().Err(err).Str("client_key_id", clientKey.ID.String()).Msg("Failed to update last_used_at")
	} else {
		// Update the struct with the new timestamp
		clientKey.LastUsedAt = &now
	}

	return &clientKey, nil
}

// ListClientKeys lists all client keys (optionally filtered by user)
func (s *ClientKeyService) ListClientKeys(ctx context.Context, userID *uuid.UUID) ([]ClientKey, error) {
	var query string
	var args []interface{}

	if userID != nil {
		query = `
			SELECT id, name, description, key_hash, key_prefix, user_id, scopes, rate_limit_per_minute, last_used_at, expires_at, revoked_at, created_at, updated_at
			FROM auth.client_keys
			WHERE user_id = $1
			ORDER BY created_at DESC
		`
		args = []interface{}{userID}
	} else {
		query = `
			SELECT id, name, description, key_hash, key_prefix, user_id, scopes, rate_limit_per_minute, last_used_at, expires_at, revoked_at, created_at, updated_at
			FROM auth.client_keys
			ORDER BY created_at DESC
		`
	}

	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list client keys: %w", err)
	}
	defer rows.Close()

	var clientKeys []ClientKey
	for rows.Next() {
		var clientKey ClientKey
		err := rows.Scan(
			&clientKey.ID,
			&clientKey.Name,
			&clientKey.Description,
			&clientKey.KeyHash,
			&clientKey.KeyPrefix,
			&clientKey.UserID,
			&clientKey.Scopes,
			&clientKey.RateLimitPerMinute,
			&clientKey.LastUsedAt,
			&clientKey.ExpiresAt,
			&clientKey.RevokedAt,
			&clientKey.CreatedAt,
			&clientKey.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan client key: %w", err)
		}
		clientKeys = append(clientKeys, clientKey)
	}

	return clientKeys, nil
}

// RevokeClientKey revokes a client key
func (s *ClientKeyService) RevokeClientKey(ctx context.Context, id uuid.UUID) error {
	query := "UPDATE auth.client_keys SET revoked_at = NOW() WHERE id = $1"
	result, err := s.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to revoke client key: %w", err)
	}

	if result.RowsAffected() == 0 {
		return errors.New("client key not found")
	}

	return nil
}

// DeleteClientKey permanently deletes a client key
func (s *ClientKeyService) DeleteClientKey(ctx context.Context, id uuid.UUID) error {
	query := "DELETE FROM auth.client_keys WHERE id = $1"
	result, err := s.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete client key: %w", err)
	}

	if result.RowsAffected() == 0 {
		return errors.New("client key not found")
	}

	return nil
}

// UpdateClientKey updates a client key's metadata
func (s *ClientKeyService) UpdateClientKey(ctx context.Context, id uuid.UUID, name *string, description *string, scopes []string, rateLimitPerMinute *int) error {
	// Validate scopes if provided
	if len(scopes) > 0 {
		if err := ValidateScopes(scopes); err != nil {
			return fmt.Errorf("invalid scopes: %w", err)
		}
	}

	query := `
		UPDATE auth.client_keys
		SET name = COALESCE($2, name),
		    description = COALESCE($3, description),
		    scopes = COALESCE($4, scopes),
		    rate_limit_per_minute = COALESCE($5, rate_limit_per_minute)
		WHERE id = $1
	`

	result, err := s.db.Exec(ctx, query, id, name, description, scopes, rateLimitPerMinute)
	if err != nil {
		return fmt.Errorf("failed to update client key: %w", err)
	}

	if result.RowsAffected() == 0 {
		return errors.New("client key not found")
	}

	return nil
}

// hashClientKey hashes a client key using SHA-256
func hashClientKey(key string) string {
	hash := sha256.Sum256([]byte(key))
	return base64.URLEncoding.EncodeToString(hash[:])
}
