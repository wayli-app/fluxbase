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
)

var (
	// ErrInvalidAPIKey is returned when API key is invalid
	ErrInvalidAPIKey = errors.New("invalid API key")
	// ErrAPIKeyExpired is returned when API key has expired
	ErrAPIKeyExpired = errors.New("API key has expired")
	// ErrAPIKeyRevoked is returned when API key has been revoked
	ErrAPIKeyRevoked = errors.New("API key has been revoked")
)

// APIKey represents an API key
type APIKey struct {
	ID                 uuid.UUID  `json:"id"`
	Name               string     `json:"name"`
	Description        *string    `json:"description,omitempty"`
	KeyHash            string     `json:"-"` // Never expose the hash
	KeyPrefix          string     `json:"key_prefix"`
	UserID             *uuid.UUID `json:"user_id,omitempty"`
	Scopes             []string   `json:"scopes"`
	RateLimitPerMinute int        `json:"rate_limit_per_minute"`
	LastUsedAt         *time.Time `json:"last_used_at,omitempty"`
	ExpiresAt          *time.Time `json:"expires_at,omitempty"`
	RevokedAt          *time.Time `json:"revoked_at,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

// APIKeyWithPlaintext includes the plaintext key (only returned once during creation)
type APIKeyWithPlaintext struct {
	APIKey
	PlaintextKey string `json:"key"` // Full key, only shown once
}

// APIKeyService handles API key operations
type APIKeyService struct {
	db *pgxpool.Pool
}

// NewAPIKeyService creates a new API key service
func NewAPIKeyService(db *pgxpool.Pool) *APIKeyService {
	return &APIKeyService{db: db}
}

// GenerateAPIKey generates a new API key with format: fbk_<random_string>
func (s *APIKeyService) GenerateAPIKey(ctx context.Context, name string, description *string, userID *uuid.UUID, scopes []string, rateLimitPerMinute int, expiresAt *time.Time) (*APIKeyWithPlaintext, error) {
	// Generate random bytes for the key
	keyBytes := make([]byte, 32)
	if _, err := rand.Read(keyBytes); err != nil {
		return nil, fmt.Errorf("failed to generate random key: %w", err)
	}

	// Create the plaintext key with prefix
	plaintextKey := "fbk_" + base64.URLEncoding.EncodeToString(keyBytes)

	// Hash the key for storage
	keyHash := hashAPIKey(plaintextKey)

	// Extract prefix (first 12 chars: "fbk_" + 8 chars)
	keyPrefix := plaintextKey[:12]

	// Set default scopes if none provided
	if len(scopes) == 0 {
		scopes = []string{"read:tables", "write:tables", "read:storage", "write:storage", "read:functions", "execute:functions"}
	}

	// Set default rate limit
	if rateLimitPerMinute == 0 {
		rateLimitPerMinute = 100
	}

	// Insert into database
	var apiKey APIKey
	query := `
		INSERT INTO auth.api_keys (name, description, key_hash, key_prefix, user_id, scopes, rate_limit_per_minute, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, name, description, key_hash, key_prefix, user_id, scopes, rate_limit_per_minute, last_used_at, expires_at, revoked_at, created_at, updated_at
	`

	err := s.db.QueryRow(ctx, query, name, description, keyHash, keyPrefix, userID, scopes, rateLimitPerMinute, expiresAt).Scan(
		&apiKey.ID,
		&apiKey.Name,
		&apiKey.Description,
		&apiKey.KeyHash,
		&apiKey.KeyPrefix,
		&apiKey.UserID,
		&apiKey.Scopes,
		&apiKey.RateLimitPerMinute,
		&apiKey.LastUsedAt,
		&apiKey.ExpiresAt,
		&apiKey.RevokedAt,
		&apiKey.CreatedAt,
		&apiKey.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create API key: %w", err)
	}

	return &APIKeyWithPlaintext{
		APIKey:       apiKey,
		PlaintextKey: plaintextKey,
	}, nil
}

// ValidateAPIKey validates an API key and returns the associated API key info
func (s *APIKeyService) ValidateAPIKey(ctx context.Context, plaintextKey string) (*APIKey, error) {
	// Hash the provided key
	keyHash := hashAPIKey(plaintextKey)

	// Query the database
	var apiKey APIKey
	query := `
		SELECT id, name, description, key_hash, key_prefix, user_id, scopes, rate_limit_per_minute, last_used_at, expires_at, revoked_at, created_at, updated_at
		FROM auth.api_keys
		WHERE key_hash = $1
	`

	err := s.db.QueryRow(ctx, query, keyHash).Scan(
		&apiKey.ID,
		&apiKey.Name,
		&apiKey.Description,
		&apiKey.KeyHash,
		&apiKey.KeyPrefix,
		&apiKey.UserID,
		&apiKey.Scopes,
		&apiKey.RateLimitPerMinute,
		&apiKey.LastUsedAt,
		&apiKey.ExpiresAt,
		&apiKey.RevokedAt,
		&apiKey.CreatedAt,
		&apiKey.UpdatedAt,
	)
	if err != nil {
		return nil, ErrInvalidAPIKey
	}

	// Check if revoked
	if apiKey.RevokedAt != nil {
		return nil, ErrAPIKeyRevoked
	}

	// Check if expired
	if apiKey.ExpiresAt != nil && apiKey.ExpiresAt.Before(time.Now()) {
		return nil, ErrAPIKeyExpired
	}

	// Update last used timestamp
	now := time.Now()
	_, err = s.db.Exec(ctx, "UPDATE auth.api_keys SET last_used_at = $1 WHERE id = $2", now, apiKey.ID)
	if err != nil {
		// Log but don't fail validation
		fmt.Printf("Failed to update last_used_at: %v\n", err)
	} else {
		// Update the struct with the new timestamp
		apiKey.LastUsedAt = &now
	}

	return &apiKey, nil
}

// ListAPIKeys lists all API keys (optionally filtered by user)
func (s *APIKeyService) ListAPIKeys(ctx context.Context, userID *uuid.UUID) ([]APIKey, error) {
	var query string
	var args []interface{}

	if userID != nil {
		query = `
			SELECT id, name, description, key_hash, key_prefix, user_id, scopes, rate_limit_per_minute, last_used_at, expires_at, revoked_at, created_at, updated_at
			FROM auth.api_keys
			WHERE user_id = $1
			ORDER BY created_at DESC
		`
		args = []interface{}{userID}
	} else {
		query = `
			SELECT id, name, description, key_hash, key_prefix, user_id, scopes, rate_limit_per_minute, last_used_at, expires_at, revoked_at, created_at, updated_at
			FROM auth.api_keys
			ORDER BY created_at DESC
		`
	}

	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list API keys: %w", err)
	}
	defer rows.Close()

	var apiKeys []APIKey
	for rows.Next() {
		var apiKey APIKey
		err := rows.Scan(
			&apiKey.ID,
			&apiKey.Name,
			&apiKey.Description,
			&apiKey.KeyHash,
			&apiKey.KeyPrefix,
			&apiKey.UserID,
			&apiKey.Scopes,
			&apiKey.RateLimitPerMinute,
			&apiKey.LastUsedAt,
			&apiKey.ExpiresAt,
			&apiKey.RevokedAt,
			&apiKey.CreatedAt,
			&apiKey.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan API key: %w", err)
		}
		apiKeys = append(apiKeys, apiKey)
	}

	return apiKeys, nil
}

// RevokeAPIKey revokes an API key
func (s *APIKeyService) RevokeAPIKey(ctx context.Context, id uuid.UUID) error {
	query := "UPDATE auth.api_keys SET revoked_at = NOW() WHERE id = $1"
	result, err := s.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to revoke API key: %w", err)
	}

	if result.RowsAffected() == 0 {
		return errors.New("API key not found")
	}

	return nil
}

// DeleteAPIKey permanently deletes an API key
func (s *APIKeyService) DeleteAPIKey(ctx context.Context, id uuid.UUID) error {
	query := "DELETE FROM auth.api_keys WHERE id = $1"
	result, err := s.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete API key: %w", err)
	}

	if result.RowsAffected() == 0 {
		return errors.New("API key not found")
	}

	return nil
}

// UpdateAPIKey updates an API key's metadata
func (s *APIKeyService) UpdateAPIKey(ctx context.Context, id uuid.UUID, name *string, description *string, scopes []string, rateLimitPerMinute *int) error {
	query := `
		UPDATE auth.api_keys
		SET name = COALESCE($2, name),
		    description = COALESCE($3, description),
		    scopes = COALESCE($4, scopes),
		    rate_limit_per_minute = COALESCE($5, rate_limit_per_minute)
		WHERE id = $1
	`

	result, err := s.db.Exec(ctx, query, id, name, description, scopes, rateLimitPerMinute)
	if err != nil {
		return fmt.Errorf("failed to update API key: %w", err)
	}

	if result.RowsAffected() == 0 {
		return errors.New("API key not found")
	}

	return nil
}

// hashAPIKey hashes an API key using SHA-256
func hashAPIKey(key string) string {
	hash := sha256.Sum256([]byte(key))
	return base64.URLEncoding.EncodeToString(hash[:])
}
