package settings

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/nimbleflux/fluxbase/internal/crypto"
	"github.com/nimbleflux/fluxbase/internal/database"
)

// SecretSettingMetadata represents metadata for a secret setting (value is never exposed)
type SecretSettingMetadata struct {
	ID          uuid.UUID  `json:"id"`
	Key         string     `json:"key"`
	Description string     `json:"description,omitempty"`
	UserID      *uuid.UUID `json:"user_id,omitempty"`
	CreatedBy   *uuid.UUID `json:"created_by,omitempty"`
	UpdatedBy   *uuid.UUID `json:"updated_by,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// CreateSecretSettingRequest represents the request to create a secret setting
type CreateSecretSettingRequest struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	Description string `json:"description,omitempty"`
}

// UpdateSecretSettingRequest represents the request to update a secret setting
type UpdateSecretSettingRequest struct {
	Value       *string `json:"value,omitempty"`
	Description *string `json:"description,omitempty"`
}

// CreateSecretSetting creates a new encrypted secret setting
// For user-specific secrets, pass userID. For system secrets, pass nil.
func (s *CustomSettingsService) CreateSecretSetting(ctx context.Context, req CreateSecretSettingRequest, userID *uuid.UUID, createdBy uuid.UUID) (*SecretSettingMetadata, error) {
	if err := ValidateKey(req.Key); err != nil {
		return nil, err
	}

	// Determine encryption key (user-specific or system)
	encKey := s.encryptionKey
	if userID != nil {
		derivedKey, err := crypto.DeriveUserKeyWithBytesKey(s.encryptionKey, userID.String(), "fluxbase-user-settings-v1")
		if err != nil {
			return nil, fmt.Errorf("failed to derive user key: %w", err)
		}
		encKey = derivedKey
	}

	// Encrypt the value
	encryptedValue, err := crypto.EncryptWithBytesKey(req.Value, encKey)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt secret: %w", err)
	}

	// Store placeholder in value column (never expose real value)
	placeholderValue := map[string]interface{}{"value": "[ENCRYPTED]"}
	valueJSON, _ := json.Marshal(placeholderValue)

	var metadata SecretSettingMetadata
	err = s.WithTenant(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `
			INSERT INTO app.settings
			(key, value, value_type, description, is_secret, encrypted_value, user_id, editable_by, category, created_by, updated_by)
			VALUES ($1, $2, 'string', $3, true, $4, $5, ARRAY['instance_admin']::TEXT[], 'custom', $6, $6)
			RETURNING id, key, description, user_id, created_by, updated_by, created_at, updated_at
		`, req.Key, valueJSON, req.Description, encryptedValue, userID, createdBy).Scan(
			&metadata.ID,
			&metadata.Key,
			&metadata.Description,
			&metadata.UserID,
			&metadata.CreatedBy,
			&metadata.UpdatedBy,
			&metadata.CreatedAt,
			&metadata.UpdatedAt,
		)
	})
	if err != nil {
		if database.IsUniqueViolation(err) {
			return nil, ErrCustomSettingDuplicate
		}
		return nil, err
	}

	return &metadata, nil
}

// GetSecretSettingMetadata retrieves metadata for a secret setting (never returns the value)
func (s *CustomSettingsService) GetSecretSettingMetadata(ctx context.Context, key string, userID *uuid.UUID) (*SecretSettingMetadata, error) {
	var metadata SecretSettingMetadata

	query := `
		SELECT id, key, description, user_id, created_by, updated_by, created_at, updated_at
		FROM app.settings
		WHERE key = $1 AND is_secret = true
	`
	args := []interface{}{key}

	// Filter by user_id if provided (user-specific) or NULL (system)
	if userID != nil {
		query += " AND user_id = $2"
		args = append(args, *userID)
	} else {
		query += " AND user_id IS NULL"
	}

	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, query, args...).Scan(
			&metadata.ID,
			&metadata.Key,
			&metadata.Description,
			&metadata.UserID,
			&metadata.CreatedBy,
			&metadata.UpdatedBy,
			&metadata.CreatedAt,
			&metadata.UpdatedAt,
		)
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCustomSettingNotFound
		}
		return nil, err
	}

	return &metadata, nil
}

// UpdateSecretSetting updates an existing secret setting
func (s *CustomSettingsService) UpdateSecretSetting(ctx context.Context, key string, req UpdateSecretSettingRequest, userID *uuid.UUID, updatedBy uuid.UUID) (*SecretSettingMetadata, error) {
	// First check if the setting exists
	existing, err := s.GetSecretSettingMetadata(ctx, key, userID)
	if err != nil {
		return nil, err
	}

	// Build update query dynamically
	description := existing.Description
	if req.Description != nil {
		description = *req.Description
	}

	var encryptedValue *string
	if req.Value != nil {
		// Determine encryption key
		encKey := s.encryptionKey
		if userID != nil {
			derivedKey, err := crypto.DeriveUserKeyWithBytesKey(s.encryptionKey, userID.String(), "fluxbase-user-settings-v1")
			if err != nil {
				return nil, fmt.Errorf("failed to derive user key: %w", err)
			}
			encKey = derivedKey
		}

		encrypted, err := crypto.EncryptWithBytesKey(*req.Value, encKey)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt secret: %w", err)
		}
		encryptedValue = &encrypted
	}

	var metadata SecretSettingMetadata
	var query string
	var args []interface{}

	if encryptedValue != nil {
		query = `
			UPDATE app.settings
			SET description = $1, encrypted_value = $2, updated_by = $3, updated_at = NOW()
			WHERE id = $4
			RETURNING id, key, description, user_id, created_by, updated_by, created_at, updated_at
		`
		args = []interface{}{description, *encryptedValue, updatedBy, existing.ID}
	} else {
		query = `
			UPDATE app.settings
			SET description = $1, updated_by = $2, updated_at = NOW()
			WHERE id = $3
			RETURNING id, key, description, user_id, created_by, updated_by, created_at, updated_at
		`
		args = []interface{}{description, updatedBy, existing.ID}
	}

	err = s.WithTenant(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, query, args...).Scan(
			&metadata.ID,
			&metadata.Key,
			&metadata.Description,
			&metadata.UserID,
			&metadata.CreatedBy,
			&metadata.UpdatedBy,
			&metadata.CreatedAt,
			&metadata.UpdatedAt,
		)
	})
	if err != nil {
		return nil, err
	}

	return &metadata, nil
}

// DeleteSecretSetting removes a secret setting
func (s *CustomSettingsService) DeleteSecretSetting(ctx context.Context, key string, userID *uuid.UUID) error {
	query := `DELETE FROM app.settings WHERE key = $1 AND is_secret = true`
	args := []interface{}{key}

	if userID != nil {
		query += " AND user_id = $2"
		args = append(args, *userID)
	} else {
		query += " AND user_id IS NULL"
	}

	var result pgconn.CommandTag
	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		var err error
		result, err = tx.Exec(ctx, query, args...)
		return err
	})
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrCustomSettingNotFound
	}

	return nil
}

// ListSecretSettings retrieves metadata for all secret settings (never returns values)
func (s *CustomSettingsService) ListSecretSettings(ctx context.Context, userID *uuid.UUID) ([]SecretSettingMetadata, error) {
	query := `
		SELECT id, key, description, user_id, created_by, updated_by, created_at, updated_at
		FROM app.settings
		WHERE is_secret = true
	`
	args := []interface{}{}

	if userID != nil {
		query += " AND user_id = $1"
		args = append(args, *userID)
	} else {
		query += " AND user_id IS NULL"
	}

	query += " ORDER BY key"

	var secrets []SecretSettingMetadata
	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, query, args...)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var metadata SecretSettingMetadata
			err := rows.Scan(
				&metadata.ID,
				&metadata.Key,
				&metadata.Description,
				&metadata.UserID,
				&metadata.CreatedBy,
				&metadata.UpdatedBy,
				&metadata.CreatedAt,
				&metadata.UpdatedAt,
			)
			if err != nil {
				return err
			}
			secrets = append(secrets, metadata)
		}

		return rows.Err()
	})
	if err != nil {
		return nil, err
	}

	return secrets, nil
}

// ============================================================================
// Secret Settings Transaction-accepting method variants (*WithTx)
// ============================================================================

// CreateSecretSettingWithTx creates a new encrypted secret setting using a transaction
func (s *CustomSettingsService) CreateSecretSettingWithTx(ctx context.Context, tx Querier, req CreateSecretSettingRequest, userID *uuid.UUID, createdBy uuid.UUID) (*SecretSettingMetadata, error) {
	if err := ValidateKey(req.Key); err != nil {
		return nil, err
	}

	// Determine encryption key (user-specific or system)
	encKey := s.encryptionKey
	if userID != nil {
		derivedKey, err := crypto.DeriveUserKeyWithBytesKey(s.encryptionKey, userID.String(), "fluxbase-user-settings-v1")
		if err != nil {
			return nil, fmt.Errorf("failed to derive user key: %w", err)
		}
		encKey = derivedKey
	}

	// Encrypt the value
	encryptedValue, err := crypto.EncryptWithBytesKey(req.Value, encKey)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt secret: %w", err)
	}

	// Store placeholder in value column (never expose real value)
	placeholderValue := map[string]interface{}{"value": "[ENCRYPTED]"}
	valueJSON, _ := json.Marshal(placeholderValue)

	var metadata SecretSettingMetadata
	err = tx.QueryRow(ctx, `
		INSERT INTO app.settings
		(key, value, value_type, description, is_secret, encrypted_value, user_id, editable_by, category, created_by, updated_by)
		VALUES ($1, $2, 'string', $3, true, $4, $5, ARRAY['instance_admin']::TEXT[], 'custom', $6, $6)
		RETURNING id, key, description, user_id, created_by, updated_by, created_at, updated_at
	`, req.Key, valueJSON, req.Description, encryptedValue, userID, createdBy).Scan(
		&metadata.ID,
		&metadata.Key,
		&metadata.Description,
		&metadata.UserID,
		&metadata.CreatedBy,
		&metadata.UpdatedBy,
		&metadata.CreatedAt,
		&metadata.UpdatedAt,
	)
	if err != nil {
		if database.IsUniqueViolation(err) {
			return nil, ErrCustomSettingDuplicate
		}
		return nil, err
	}

	return &metadata, nil
}

// GetSecretSettingMetadataWithTx retrieves metadata for a secret setting using a transaction
func (s *CustomSettingsService) GetSecretSettingMetadataWithTx(ctx context.Context, tx Querier, key string, userID *uuid.UUID) (*SecretSettingMetadata, error) {
	var metadata SecretSettingMetadata

	query := `
		SELECT id, key, description, user_id, created_by, updated_by, created_at, updated_at
		FROM app.settings
		WHERE key = $1 AND is_secret = true
	`
	args := []interface{}{key}

	// Filter by user_id if provided (user-specific) or NULL (system)
	if userID != nil {
		query += " AND user_id = $2"
		args = append(args, *userID)
	} else {
		query += " AND user_id IS NULL"
	}

	err := tx.QueryRow(ctx, query, args...).Scan(
		&metadata.ID,
		&metadata.Key,
		&metadata.Description,
		&metadata.UserID,
		&metadata.CreatedBy,
		&metadata.UpdatedBy,
		&metadata.CreatedAt,
		&metadata.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCustomSettingNotFound
		}
		return nil, err
	}

	return &metadata, nil
}

// UpdateSecretSettingWithTx updates an existing secret setting using a transaction
func (s *CustomSettingsService) UpdateSecretSettingWithTx(ctx context.Context, tx Querier, key string, req UpdateSecretSettingRequest, userID *uuid.UUID, updatedBy uuid.UUID) (*SecretSettingMetadata, error) {
	// First check if the setting exists
	existing, err := s.GetSecretSettingMetadataWithTx(ctx, tx, key, userID)
	if err != nil {
		return nil, err
	}

	// Build update query dynamically
	description := existing.Description
	if req.Description != nil {
		description = *req.Description
	}

	var encryptedValue *string
	if req.Value != nil {
		// Determine encryption key
		encKey := s.encryptionKey
		if userID != nil {
			derivedKey, err := crypto.DeriveUserKeyWithBytesKey(s.encryptionKey, userID.String(), "fluxbase-user-settings-v1")
			if err != nil {
				return nil, fmt.Errorf("failed to derive user key: %w", err)
			}
			encKey = derivedKey
		}

		encrypted, err := crypto.EncryptWithBytesKey(*req.Value, encKey)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt secret: %w", err)
		}
		encryptedValue = &encrypted
	}

	var metadata SecretSettingMetadata
	var query string
	var args []interface{}

	if encryptedValue != nil {
		query = `
			UPDATE app.settings
			SET description = $1, encrypted_value = $2, updated_by = $3, updated_at = NOW()
			WHERE id = $4
			RETURNING id, key, description, user_id, created_by, updated_by, created_at, updated_at
		`
		args = []interface{}{description, *encryptedValue, updatedBy, existing.ID}
	} else {
		query = `
			UPDATE app.settings
			SET description = $1, updated_by = $2, updated_at = NOW()
			WHERE id = $3
			RETURNING id, key, description, user_id, created_by, updated_by, created_at, updated_at
		`
		args = []interface{}{description, updatedBy, existing.ID}
	}

	err = tx.QueryRow(ctx, query, args...).Scan(
		&metadata.ID,
		&metadata.Key,
		&metadata.Description,
		&metadata.UserID,
		&metadata.CreatedBy,
		&metadata.UpdatedBy,
		&metadata.CreatedAt,
		&metadata.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &metadata, nil
}

// DeleteSecretSettingWithTx removes a secret setting using a transaction
func (s *CustomSettingsService) DeleteSecretSettingWithTx(ctx context.Context, tx Querier, key string, userID *uuid.UUID) error {
	query := `DELETE FROM app.settings WHERE key = $1 AND is_secret = true`
	args := []interface{}{key}

	if userID != nil {
		query += " AND user_id = $2"
		args = append(args, *userID)
	} else {
		query += " AND user_id IS NULL"
	}

	result, err := tx.Exec(ctx, query, args...)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrCustomSettingNotFound
	}

	return nil
}

// ListSecretSettingsWithTx retrieves metadata for all secret settings using a transaction
func (s *CustomSettingsService) ListSecretSettingsWithTx(ctx context.Context, tx Querier, userID *uuid.UUID) ([]SecretSettingMetadata, error) {
	query := `
		SELECT id, key, description, user_id, created_by, updated_by, created_at, updated_at
		FROM app.settings
		WHERE is_secret = true
	`
	args := []interface{}{}

	if userID != nil {
		query += " AND user_id = $1"
		args = append(args, *userID)
	} else {
		query += " AND user_id IS NULL"
	}

	query += " ORDER BY key"

	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var secrets []SecretSettingMetadata
	for rows.Next() {
		var metadata SecretSettingMetadata
		err := rows.Scan(
			&metadata.ID,
			&metadata.Key,
			&metadata.Description,
			&metadata.UserID,
			&metadata.CreatedBy,
			&metadata.UpdatedBy,
			&metadata.CreatedAt,
			&metadata.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		secrets = append(secrets, metadata)
	}

	return secrets, rows.Err()
}
