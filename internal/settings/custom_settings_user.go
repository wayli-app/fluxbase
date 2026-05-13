package settings

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/nimbleflux/fluxbase/internal/database"
)

// ============================================================================
// User Settings (non-encrypted, with system fallback support)
// These methods mirror the edge function secrets helper pattern for regular settings
// ============================================================================

// UserSetting represents a user's non-encrypted setting
type UserSetting struct {
	ID          uuid.UUID              `json:"id"`
	Key         string                 `json:"key"`
	Value       map[string]interface{} `json:"value"`
	Description string                 `json:"description,omitempty"`
	UserID      uuid.UUID              `json:"user_id"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// UserSettingWithSource represents a setting with its source (user or system)
type UserSettingWithSource struct {
	Key    string                 `json:"key"`
	Value  map[string]interface{} `json:"value"`
	Source string                 `json:"source"` // "user" or "system"
}

// CreateUserSettingRequest represents the request to create a user setting
type CreateUserSettingRequest struct {
	Key         string                 `json:"key"`
	Value       map[string]interface{} `json:"value"`
	Description string                 `json:"description,omitempty"`
}

// UpdateUserSettingRequest represents the request to update a user setting
type UpdateUserSettingRequest struct {
	Value       map[string]interface{} `json:"value"`
	Description *string                `json:"description,omitempty"`
}

// CreateUserSetting creates a new non-encrypted user setting
func (s *CustomSettingsService) CreateUserSetting(ctx context.Context, userID uuid.UUID, req CreateUserSettingRequest) (*UserSetting, error) {
	if err := ValidateKey(req.Key); err != nil {
		return nil, err
	}

	valueJSON, err := json.Marshal(req.Value)
	if err != nil {
		return nil, err
	}

	var setting UserSetting
	var valueJSONResult []byte

	err = s.WithTenant(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `
			INSERT INTO app.settings
			(key, value, value_type, description, is_secret, user_id, editable_by, category, created_by, updated_by)
			VALUES ($1, $2, 'json', $3, false, $4, ARRAY['authenticated']::TEXT[], 'custom', $4, $4)
			RETURNING id, key, value, description, user_id, created_at, updated_at
		`, req.Key, valueJSON, req.Description, userID).Scan(
			&setting.ID,
			&setting.Key,
			&valueJSONResult,
			&setting.Description,
			&setting.UserID,
			&setting.CreatedAt,
			&setting.UpdatedAt,
		)
	})
	if err != nil {
		if database.IsUniqueViolation(err) {
			return nil, ErrCustomSettingDuplicate
		}
		return nil, err
	}

	if err := json.Unmarshal(valueJSONResult, &setting.Value); err != nil {
		return nil, err
	}

	return &setting, nil
}

// GetUserOwnSetting retrieves a user's own setting only (no fallback)
func (s *CustomSettingsService) GetUserOwnSetting(ctx context.Context, userID uuid.UUID, key string) (*UserSetting, error) {
	var setting UserSetting
	var valueJSON []byte

	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `
			SELECT id, key, value, description, user_id, created_at, updated_at
			FROM app.settings
			WHERE key = $1 AND user_id = $2 AND is_secret = false
		`, key, userID).Scan(
			&setting.ID,
			&setting.Key,
			&valueJSON,
			&setting.Description,
			&setting.UserID,
			&setting.CreatedAt,
			&setting.UpdatedAt,
		)
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCustomSettingNotFound
		}
		return nil, err
	}

	if err := json.Unmarshal(valueJSON, &setting.Value); err != nil {
		return nil, err
	}

	return &setting, nil
}

// GetSystemSetting retrieves a system-level setting (user_id IS NULL)
// This is for public/system settings that any authenticated user can read
func (s *CustomSettingsService) GetSystemSetting(ctx context.Context, key string) (*CustomSetting, error) {
	var setting CustomSetting
	var valueJSON, metadataJSON []byte
	var editableBy []string

	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `
			SELECT id, key, value, value_type, description, editable_by, metadata, created_by, updated_by, created_at, updated_at
			FROM app.settings
			WHERE key = $1 AND user_id IS NULL AND is_secret = false
		`, key).Scan(
			&setting.ID,
			&setting.Key,
			&valueJSON,
			&setting.ValueType,
			&setting.Description,
			&editableBy,
			&metadataJSON,
			&setting.CreatedBy,
			&setting.UpdatedBy,
			&setting.CreatedAt,
			&setting.UpdatedAt,
		)
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCustomSettingNotFound
		}
		return nil, err
	}

	if err := json.Unmarshal(valueJSON, &setting.Value); err != nil {
		return nil, err
	}
	if metadataJSON != nil {
		if err := json.Unmarshal(metadataJSON, &setting.Metadata); err != nil {
			return nil, err
		}
	}
	setting.EditableBy = editableBy

	return &setting, nil
}

// GetUserSettingWithFallback retrieves a setting with user -> system fallback
// Returns the value and whether it came from user or system
func (s *CustomSettingsService) GetUserSettingWithFallback(ctx context.Context, userID uuid.UUID, key string) (*UserSettingWithSource, error) {
	// Try user's own setting first
	userSetting, err := s.GetUserOwnSetting(ctx, userID, key)
	if err == nil {
		return &UserSettingWithSource{
			Key:    userSetting.Key,
			Value:  userSetting.Value,
			Source: "user",
		}, nil
	}

	// If not found, fall back to system setting
	if errors.Is(err, ErrCustomSettingNotFound) {
		systemSetting, err := s.GetSystemSetting(ctx, key)
		if err != nil {
			return nil, err
		}
		return &UserSettingWithSource{
			Key:    systemSetting.Key,
			Value:  systemSetting.Value,
			Source: "system",
		}, nil
	}

	return nil, err
}

// UpdateUserSetting updates an existing user setting
func (s *CustomSettingsService) UpdateUserSetting(ctx context.Context, userID uuid.UUID, key string, req UpdateUserSettingRequest) (*UserSetting, error) {
	// First check if the setting exists and belongs to the user
	existing, err := s.GetUserOwnSetting(ctx, userID, key)
	if err != nil {
		return nil, err
	}

	valueJSON, err := json.Marshal(req.Value)
	if err != nil {
		return nil, err
	}

	description := existing.Description
	if req.Description != nil {
		description = *req.Description
	}

	var setting UserSetting
	var valueJSONResult []byte

	err = s.WithTenant(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `
			UPDATE app.settings
			SET value = $1, description = $2, updated_by = $3, updated_at = NOW()
			WHERE key = $4 AND user_id = $5 AND is_secret = false
			RETURNING id, key, value, description, user_id, created_at, updated_at
		`, valueJSON, description, userID, key, userID).Scan(
			&setting.ID,
			&setting.Key,
			&valueJSONResult,
			&setting.Description,
			&setting.UserID,
			&setting.CreatedAt,
			&setting.UpdatedAt,
		)
	})
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(valueJSONResult, &setting.Value); err != nil {
		return nil, err
	}

	return &setting, nil
}

// UpsertUserSetting creates or updates a user setting
func (s *CustomSettingsService) UpsertUserSetting(ctx context.Context, userID uuid.UUID, req CreateUserSettingRequest) (*UserSetting, error) {
	if err := ValidateKey(req.Key); err != nil {
		return nil, err
	}

	valueJSON, err := json.Marshal(req.Value)
	if err != nil {
		return nil, err
	}

	var setting UserSetting
	var valueJSONResult []byte

	err = s.WithTenant(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `
			INSERT INTO app.settings
			(key, value, value_type, description, is_secret, user_id, editable_by, category, created_by, updated_by)
			VALUES ($1, $2, 'json', $3, false, $4, ARRAY['authenticated']::TEXT[], 'custom', $4, $4)
			ON CONFLICT (key, COALESCE(user_id, '00000000-0000-0000-0000-000000000000'::UUID))
			DO UPDATE SET
				value = EXCLUDED.value,
				description = COALESCE(EXCLUDED.description, app.settings.description),
				updated_by = EXCLUDED.updated_by,
				updated_at = NOW()
			RETURNING id, key, value, description, user_id, created_at, updated_at
		`, req.Key, valueJSON, req.Description, userID).Scan(
			&setting.ID,
			&setting.Key,
			&valueJSONResult,
			&setting.Description,
			&setting.UserID,
			&setting.CreatedAt,
			&setting.UpdatedAt,
		)
	})
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(valueJSONResult, &setting.Value); err != nil {
		return nil, err
	}

	return &setting, nil
}

// DeleteUserSetting removes a user's setting
func (s *CustomSettingsService) DeleteUserSetting(ctx context.Context, userID uuid.UUID, key string) error {
	var result pgconn.CommandTag
	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		var err error
		result, err = tx.Exec(ctx, `
			DELETE FROM app.settings
			WHERE key = $1 AND user_id = $2 AND is_secret = false
		`, key, userID)
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

// ListUserOwnSettings retrieves all non-encrypted settings for a user
func (s *CustomSettingsService) ListUserOwnSettings(ctx context.Context, userID uuid.UUID) ([]UserSetting, error) {
	var settings []UserSetting
	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT id, key, value, description, user_id, created_at, updated_at
			FROM app.settings
			WHERE user_id = $1 AND is_secret = false
			ORDER BY key
		`, userID)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var setting UserSetting
			var valueJSON []byte

			err := rows.Scan(
				&setting.ID,
				&setting.Key,
				&valueJSON,
				&setting.Description,
				&setting.UserID,
				&setting.CreatedAt,
				&setting.UpdatedAt,
			)
			if err != nil {
				return err
			}

			if err := json.Unmarshal(valueJSON, &setting.Value); err != nil {
				return err
			}

			settings = append(settings, setting)
		}

		return rows.Err()
	})
	if err != nil {
		return nil, err
	}

	return settings, nil
}

// ============================================================================
// User Settings Transaction-accepting method variants
// ============================================================================

// GetUserOwnSettingWithTx retrieves a user's own setting using a transaction
func (s *CustomSettingsService) GetUserOwnSettingWithTx(ctx context.Context, tx Querier, userID uuid.UUID, key string) (*UserSetting, error) {
	var setting UserSetting
	var valueJSON []byte

	err := tx.QueryRow(ctx, `
		SELECT id, key, value, description, user_id, created_at, updated_at
		FROM app.settings
		WHERE key = $1 AND user_id = $2 AND is_secret = false
	`, key, userID).Scan(
		&setting.ID,
		&setting.Key,
		&valueJSON,
		&setting.Description,
		&setting.UserID,
		&setting.CreatedAt,
		&setting.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCustomSettingNotFound
		}
		return nil, err
	}

	if err := json.Unmarshal(valueJSON, &setting.Value); err != nil {
		return nil, err
	}

	return &setting, nil
}

// GetSystemSettingWithTx retrieves a system-level setting using a transaction
func (s *CustomSettingsService) GetSystemSettingWithTx(ctx context.Context, tx Querier, key string) (*CustomSetting, error) {
	var setting CustomSetting
	var valueJSON, metadataJSON []byte
	var editableBy []string

	err := tx.QueryRow(ctx, `
		SELECT id, key, value, value_type, description, editable_by, metadata, created_by, updated_by, created_at, updated_at
		FROM app.settings
		WHERE key = $1 AND user_id IS NULL AND is_secret = false
	`, key).Scan(
		&setting.ID,
		&setting.Key,
		&valueJSON,
		&setting.ValueType,
		&setting.Description,
		&editableBy,
		&metadataJSON,
		&setting.CreatedBy,
		&setting.UpdatedBy,
		&setting.CreatedAt,
		&setting.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCustomSettingNotFound
		}
		return nil, err
	}

	if err := json.Unmarshal(valueJSON, &setting.Value); err != nil {
		return nil, err
	}
	if metadataJSON != nil {
		if err := json.Unmarshal(metadataJSON, &setting.Metadata); err != nil {
			return nil, err
		}
	}
	setting.EditableBy = editableBy

	return &setting, nil
}

// GetUserSettingWithFallbackWithTx retrieves a setting with user -> system fallback using a transaction
func (s *CustomSettingsService) GetUserSettingWithFallbackWithTx(ctx context.Context, tx Querier, userID uuid.UUID, key string) (*UserSettingWithSource, error) {
	// Try user's own setting first
	userSetting, err := s.GetUserOwnSettingWithTx(ctx, tx, userID, key)
	if err == nil {
		return &UserSettingWithSource{
			Key:    userSetting.Key,
			Value:  userSetting.Value,
			Source: "user",
		}, nil
	}

	// If not found, fall back to system setting
	if errors.Is(err, ErrCustomSettingNotFound) {
		systemSetting, err := s.GetSystemSettingWithTx(ctx, tx, key)
		if err != nil {
			return nil, err
		}
		return &UserSettingWithSource{
			Key:    systemSetting.Key,
			Value:  systemSetting.Value,
			Source: "system",
		}, nil
	}

	return nil, err
}

// UpsertUserSettingWithTx creates or updates a user setting using a transaction
func (s *CustomSettingsService) UpsertUserSettingWithTx(ctx context.Context, tx Querier, userID uuid.UUID, req CreateUserSettingRequest) (*UserSetting, error) {
	if err := ValidateKey(req.Key); err != nil {
		return nil, err
	}

	valueJSON, err := json.Marshal(req.Value)
	if err != nil {
		return nil, err
	}

	var setting UserSetting
	var valueJSONResult []byte

	err = tx.QueryRow(ctx, `
		INSERT INTO app.settings
		(key, value, value_type, description, is_secret, user_id, editable_by, category, created_by, updated_by)
		VALUES ($1, $2, 'json', $3, false, $4, ARRAY['authenticated']::TEXT[], 'custom', $4, $4)
		ON CONFLICT (key, COALESCE(user_id, '00000000-0000-0000-0000-000000000000'::UUID))
		DO UPDATE SET
			value = EXCLUDED.value,
			description = COALESCE(EXCLUDED.description, app.settings.description),
			updated_by = EXCLUDED.updated_by,
			updated_at = NOW()
		RETURNING id, key, value, description, user_id, created_at, updated_at
	`, req.Key, valueJSON, req.Description, userID).Scan(
		&setting.ID,
		&setting.Key,
		&valueJSONResult,
		&setting.Description,
		&setting.UserID,
		&setting.CreatedAt,
		&setting.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(valueJSONResult, &setting.Value); err != nil {
		return nil, err
	}

	return &setting, nil
}

// DeleteUserSettingWithTx removes a user's setting using a transaction
func (s *CustomSettingsService) DeleteUserSettingWithTx(ctx context.Context, tx Querier, userID uuid.UUID, key string) error {
	result, err := tx.Exec(ctx, `
		DELETE FROM app.settings
		WHERE key = $1 AND user_id = $2 AND is_secret = false
	`, key, userID)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrCustomSettingNotFound
	}

	return nil
}

// ListUserOwnSettingsWithTx retrieves all non-encrypted settings for a user using a transaction
func (s *CustomSettingsService) ListUserOwnSettingsWithTx(ctx context.Context, tx Querier, userID uuid.UUID) ([]UserSetting, error) {
	rows, err := tx.Query(ctx, `
		SELECT id, key, value, description, user_id, created_at, updated_at
		FROM app.settings
		WHERE user_id = $1 AND is_secret = false
		ORDER BY key
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var settings []UserSetting
	for rows.Next() {
		var setting UserSetting
		var valueJSON []byte

		err := rows.Scan(
			&setting.ID,
			&setting.Key,
			&valueJSON,
			&setting.Description,
			&setting.UserID,
			&setting.CreatedAt,
			&setting.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		if err := json.Unmarshal(valueJSON, &setting.Value); err != nil {
			return nil, err
		}

		settings = append(settings, setting)
	}

	return settings, rows.Err()
}
