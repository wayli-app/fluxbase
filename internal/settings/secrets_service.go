package settings

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/fluxbase-eu/fluxbase/internal/crypto"
	"github.com/fluxbase-eu/fluxbase/internal/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var (
	// ErrSecretNotFound is returned when a secret setting is not found
	ErrSecretNotFound = errors.New("secret setting not found")
	// ErrDecryptionFailed is returned when decryption fails
	ErrDecryptionFailed = errors.New("failed to decrypt secret")
)

// SecretsService provides server-side access to encrypted secret settings.
// This service should ONLY be used internally by edge functions, background jobs,
// and custom handlers. It should NEVER be exposed via API endpoints.
type SecretsService struct {
	db            *database.Connection
	encryptionKey string
}

// NewSecretsService creates a new secrets service for server-side decryption
func NewSecretsService(db *database.Connection, encryptionKey string) *SecretsService {
	return &SecretsService{db: db, encryptionKey: encryptionKey}
}

// GetUserSecret retrieves and decrypts a user's secret setting.
// This should only be called from server-side code (handlers, jobs, edge functions).
func (s *SecretsService) GetUserSecret(ctx context.Context, userID uuid.UUID, key string) (string, error) {
	var encryptedValue string

	err := s.db.QueryRow(ctx, `
		SELECT encrypted_value
		FROM app.settings
		WHERE key = $1 AND user_id = $2 AND is_secret = true AND encrypted_value IS NOT NULL
	`, key, userID).Scan(&encryptedValue)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrSecretNotFound
		}
		return "", err
	}

	// Derive user-specific key
	derivedKey, err := crypto.DeriveUserKey(s.encryptionKey, userID)
	if err != nil {
		return "", fmt.Errorf("failed to derive user key: %w", err)
	}

	// Decrypt
	plaintext, err := crypto.Decrypt(encryptedValue, derivedKey)
	if err != nil {
		return "", ErrDecryptionFailed
	}

	return plaintext, nil
}

// GetSystemSecret retrieves and decrypts a system-level secret setting.
// This should only be called from server-side code (handlers, jobs, edge functions).
func (s *SecretsService) GetSystemSecret(ctx context.Context, key string) (string, error) {
	var encryptedValue string

	err := s.db.QueryRow(ctx, `
		SELECT encrypted_value
		FROM app.settings
		WHERE key = $1 AND user_id IS NULL AND is_secret = true AND encrypted_value IS NOT NULL
	`, key).Scan(&encryptedValue)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrSecretNotFound
		}
		return "", err
	}

	// Use master key for system secrets
	plaintext, err := crypto.Decrypt(encryptedValue, s.encryptionKey)
	if err != nil {
		return "", ErrDecryptionFailed
	}

	return plaintext, nil
}

// GetUserSecrets retrieves all decrypted secrets for a user.
// This is used for injecting secrets into edge functions as environment variables.
func (s *SecretsService) GetUserSecrets(ctx context.Context, userID uuid.UUID) (map[string]string, error) {
	rows, err := s.db.Query(ctx, `
		SELECT key, encrypted_value
		FROM app.settings
		WHERE user_id = $1 AND is_secret = true AND encrypted_value IS NOT NULL
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Derive user-specific key once
	derivedKey, err := crypto.DeriveUserKey(s.encryptionKey, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to derive user key: %w", err)
	}

	secrets := make(map[string]string)
	for rows.Next() {
		var key, encryptedValue string
		if err := rows.Scan(&key, &encryptedValue); err != nil {
			return nil, err
		}

		plaintext, err := crypto.Decrypt(encryptedValue, derivedKey)
		if err != nil {
			// Log warning but continue with other secrets
			continue
		}

		secrets[key] = plaintext
	}

	return secrets, rows.Err()
}

// GetSystemSecrets retrieves all decrypted system-level secrets.
// This is used for injecting secrets into edge functions as environment variables.
func (s *SecretsService) GetSystemSecrets(ctx context.Context) (map[string]string, error) {
	rows, err := s.db.Query(ctx, `
		SELECT key, encrypted_value
		FROM app.settings
		WHERE user_id IS NULL AND is_secret = true AND encrypted_value IS NOT NULL
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	secrets := make(map[string]string)
	for rows.Next() {
		var key, encryptedValue string
		if err := rows.Scan(&key, &encryptedValue); err != nil {
			return nil, err
		}

		plaintext, err := crypto.Decrypt(encryptedValue, s.encryptionKey)
		if err != nil {
			// Log warning but continue with other secrets
			continue
		}

		secrets[key] = plaintext
	}

	return secrets, rows.Err()
}

// SetSystemSecret creates or updates a system-level secret setting.
// This encrypts the value and stores it in the database.
func (s *SecretsService) SetSystemSecret(ctx context.Context, key, value, description string) error {
	// Encrypt with master key for system secrets
	encryptedValue, err := crypto.Encrypt(value, s.encryptionKey)
	if err != nil {
		return fmt.Errorf("failed to encrypt secret: %w", err)
	}

	// Upsert the secret
	_, err = s.db.Exec(ctx, `
		INSERT INTO app.settings (key, value, description, is_secret, encrypted_value, user_id, created_at, updated_at)
		VALUES ($1, '{"value": "[ENCRYPTED]"}', $2, true, $3, NULL, NOW(), NOW())
		ON CONFLICT (key, COALESCE(user_id, '00000000-0000-0000-0000-000000000000'::UUID))
		DO UPDATE SET
			encrypted_value = EXCLUDED.encrypted_value,
			description = COALESCE(EXCLUDED.description, app.settings.description),
			updated_at = NOW()
	`, key, description, encryptedValue)

	return err
}

// SetUserSecret creates or updates a user-specific secret setting.
// This encrypts the value with a user-derived key and stores it in the database.
func (s *SecretsService) SetUserSecret(ctx context.Context, userID uuid.UUID, key, value, description string) error {
	// Derive user-specific key
	derivedKey, err := crypto.DeriveUserKey(s.encryptionKey, userID)
	if err != nil {
		return fmt.Errorf("failed to derive user key: %w", err)
	}

	// Encrypt with user-derived key
	encryptedValue, err := crypto.Encrypt(value, derivedKey)
	if err != nil {
		return fmt.Errorf("failed to encrypt secret: %w", err)
	}

	// Upsert the secret
	_, err = s.db.Exec(ctx, `
		INSERT INTO app.settings (key, value, description, is_secret, encrypted_value, user_id, created_at, updated_at)
		VALUES ($1, '{"value": "[ENCRYPTED]"}', $2, true, $3, $4, NOW(), NOW())
		ON CONFLICT (key, COALESCE(user_id, '00000000-0000-0000-0000-000000000000'::UUID))
		DO UPDATE SET
			encrypted_value = EXCLUDED.encrypted_value,
			description = COALESCE(EXCLUDED.description, app.settings.description),
			updated_at = NOW()
	`, key, description, encryptedValue, userID)

	return err
}

// DeleteSystemSecret removes a system-level secret setting.
func (s *SecretsService) DeleteSystemSecret(ctx context.Context, key string) error {
	result, err := s.db.Exec(ctx, `
		DELETE FROM app.settings
		WHERE key = $1 AND user_id IS NULL AND is_secret = true
	`, key)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrSecretNotFound
	}

	return nil
}

// DeleteUserSecret removes a user-specific secret setting.
func (s *SecretsService) DeleteUserSecret(ctx context.Context, userID uuid.UUID, key string) error {
	result, err := s.db.Exec(ctx, `
		DELETE FROM app.settings
		WHERE key = $1 AND user_id = $2 AND is_secret = true
	`, key, userID)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrSecretNotFound
	}

	return nil
}

// ErrSettingNotFound is returned when a setting is not found
var ErrSettingNotFound = errors.New("setting not found")

// GetUserSetting retrieves a user's setting (secret or non-secret).
// For secrets, returns the decrypted value. For non-secrets, returns the JSON value as string.
func (s *SecretsService) GetUserSetting(ctx context.Context, userID uuid.UUID, key string) (string, error) {
	var isSecret bool
	var encryptedValue *string
	var value []byte

	err := s.db.QueryRow(ctx, `
		SELECT is_secret, encrypted_value, value
		FROM app.settings
		WHERE key = $1 AND user_id = $2
	`, key, userID).Scan(&isSecret, &encryptedValue, &value)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrSettingNotFound
		}
		return "", err
	}

	if isSecret && encryptedValue != nil {
		// Derive user-specific key and decrypt
		derivedKey, err := crypto.DeriveUserKey(s.encryptionKey, userID)
		if err != nil {
			return "", fmt.Errorf("failed to derive user key: %w", err)
		}
		return crypto.Decrypt(*encryptedValue, derivedKey)
	}

	// For non-secrets, extract string value from JSONB
	return extractJSONStringValue(value), nil
}

// GetSystemSetting retrieves a system-level setting (secret or non-secret).
// For secrets, returns the decrypted value. For non-secrets, returns the JSON value as string.
func (s *SecretsService) GetSystemSetting(ctx context.Context, key string) (string, error) {
	var isSecret bool
	var encryptedValue *string
	var value []byte

	err := s.db.QueryRow(ctx, `
		SELECT is_secret, encrypted_value, value
		FROM app.settings
		WHERE key = $1 AND user_id IS NULL
	`, key).Scan(&isSecret, &encryptedValue, &value)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrSettingNotFound
		}
		return "", err
	}

	if isSecret && encryptedValue != nil {
		// Use master key for system secrets
		return crypto.Decrypt(*encryptedValue, s.encryptionKey)
	}

	// For non-secrets, extract string value from JSONB
	return extractJSONStringValue(value), nil
}

// GetAllUserSettings retrieves all settings for a user (both secrets and non-secrets).
// Returns a map of key -> value. Secrets are decrypted.
func (s *SecretsService) GetAllUserSettings(ctx context.Context, userID uuid.UUID) (map[string]string, error) {
	rows, err := s.db.Query(ctx, `
		SELECT key, is_secret, encrypted_value, value
		FROM app.settings
		WHERE user_id = $1
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Derive user-specific key once for all secrets
	derivedKey, err := crypto.DeriveUserKey(s.encryptionKey, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to derive user key: %w", err)
	}

	settings := make(map[string]string)
	for rows.Next() {
		var key string
		var isSecret bool
		var encryptedValue *string
		var value []byte

		if err := rows.Scan(&key, &isSecret, &encryptedValue, &value); err != nil {
			return nil, err
		}

		if isSecret && encryptedValue != nil {
			plaintext, err := crypto.Decrypt(*encryptedValue, derivedKey)
			if err != nil {
				// Skip settings that fail to decrypt
				continue
			}
			settings[key] = plaintext
		} else {
			settings[key] = extractJSONStringValue(value)
		}
	}

	return settings, rows.Err()
}

// GetAllSystemSettings retrieves all system-level settings (both secrets and non-secrets).
// Returns a map of key -> value. Secrets are decrypted.
func (s *SecretsService) GetAllSystemSettings(ctx context.Context) (map[string]string, error) {
	rows, err := s.db.Query(ctx, `
		SELECT key, is_secret, encrypted_value, value
		FROM app.settings
		WHERE user_id IS NULL
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	settings := make(map[string]string)
	for rows.Next() {
		var key string
		var isSecret bool
		var encryptedValue *string
		var value []byte

		if err := rows.Scan(&key, &isSecret, &encryptedValue, &value); err != nil {
			return nil, err
		}

		if isSecret && encryptedValue != nil {
			plaintext, err := crypto.Decrypt(*encryptedValue, s.encryptionKey)
			if err != nil {
				// Skip settings that fail to decrypt
				continue
			}
			settings[key] = plaintext
		} else {
			settings[key] = extractJSONStringValue(value)
		}
	}

	return settings, rows.Err()
}

// GetSettingWithFallback retrieves a setting with user -> system fallback.
// Returns the value and a boolean indicating if found.
func (s *SecretsService) GetSettingWithFallback(ctx context.Context, userID *uuid.UUID, key string) (string, bool, error) {
	// Try user-specific setting first if userID provided
	if userID != nil {
		val, err := s.GetUserSetting(ctx, *userID, key)
		if err == nil {
			return val, true, nil
		}
		if !errors.Is(err, ErrSettingNotFound) {
			return "", false, err
		}
	}

	// Fall back to system setting
	val, err := s.GetSystemSetting(ctx, key)
	if err == nil {
		return val, true, nil
	}
	if errors.Is(err, ErrSettingNotFound) {
		return "", false, nil
	}
	return "", false, err
}

// extractJSONStringValue extracts a string value from a JSONB column.
// Handles both direct string values and {"value": "..."} objects.
func extractJSONStringValue(jsonBytes []byte) string {
	if len(jsonBytes) == 0 {
		return ""
	}

	// If it's a simple quoted string
	if jsonBytes[0] == '"' {
		var s string
		if err := json.Unmarshal(jsonBytes, &s); err == nil {
			return s
		}
	}

	// Try to extract from {"value": "..."} format
	var obj map[string]any
	if err := json.Unmarshal(jsonBytes, &obj); err == nil {
		if v, ok := obj["value"]; ok {
			switch val := v.(type) {
			case string:
				return val
			case float64:
				return fmt.Sprintf("%v", val)
			case bool:
				return fmt.Sprintf("%v", val)
			default:
				// For complex values, return JSON string
				if b, err := json.Marshal(val); err == nil {
					return string(b)
				}
			}
		}
	}

	// Return raw JSON as string
	return string(jsonBytes)
}
