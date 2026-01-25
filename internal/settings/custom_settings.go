package settings

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/fluxbase-eu/fluxbase/internal/crypto"
	"github.com/fluxbase-eu/fluxbase/internal/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	// ErrCustomSettingNotFound is returned when a custom setting is not found
	ErrCustomSettingNotFound = errors.New("custom setting not found")
	// ErrCustomSettingPermissionDenied is returned when user lacks permission to modify a setting
	ErrCustomSettingPermissionDenied = errors.New("permission denied for this custom setting")
	// ErrCustomSettingInvalidKey is returned when the key format is invalid
	ErrCustomSettingInvalidKey = errors.New("invalid custom setting key format")
	// ErrCustomSettingDuplicate is returned when a setting with the same key already exists
	ErrCustomSettingDuplicate = errors.New("custom setting with this key already exists")
)

// CustomSetting represents a custom admin-managed configuration setting
type CustomSetting struct {
	ID          uuid.UUID              `json:"id"`
	Key         string                 `json:"key"`
	Value       map[string]interface{} `json:"value"`
	ValueType   string                 `json:"value_type"`
	Description string                 `json:"description,omitempty"`
	EditableBy  []string               `json:"editable_by"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	CreatedBy   *uuid.UUID             `json:"created_by,omitempty"`
	UpdatedBy   *uuid.UUID             `json:"updated_by,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// CreateCustomSettingRequest represents the request to create a custom setting
type CreateCustomSettingRequest struct {
	Key         string                 `json:"key"`
	Value       map[string]interface{} `json:"value"`
	ValueType   string                 `json:"value_type"`
	Description string                 `json:"description,omitempty"`
	EditableBy  []string               `json:"editable_by,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	IsSecret    bool                   `json:"is_secret,omitempty"`
}

// UpdateCustomSettingRequest represents the request to update a custom setting
type UpdateCustomSettingRequest struct {
	Value       map[string]interface{} `json:"value"`
	Description *string                `json:"description,omitempty"`
	EditableBy  []string               `json:"editable_by,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

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

// CustomSettingsService handles custom admin-managed settings
type CustomSettingsService struct {
	db            *database.Connection
	encryptionKey string
}

// NewCustomSettingsService creates a new custom settings service
func NewCustomSettingsService(db *database.Connection, encryptionKey string) *CustomSettingsService {
	return &CustomSettingsService{db: db, encryptionKey: encryptionKey}
}

// CanEditSetting checks if the given role can edit a specific setting
func CanEditSetting(editableBy []string, userRole string) bool {
	// dashboard_admin, admin, and service_role can edit everything
	if userRole == "dashboard_admin" || userRole == "admin" || userRole == "service_role" {
		return true
	}

	// Check if user's role is in the editable_by list
	for _, role := range editableBy {
		if role == userRole {
			return true
		}
	}
	return false
}

// ValidateKey validates the custom setting key format
// Keys should follow a pattern like "custom.*" to avoid conflicts
func ValidateKey(key string) error {
	if key == "" {
		return ErrCustomSettingInvalidKey
	}
	// You can add more validation rules here (e.g., regex patterns, reserved prefixes)
	return nil
}

// CreateSetting creates a new custom setting
func (s *CustomSettingsService) CreateSetting(ctx context.Context, req CreateCustomSettingRequest, createdBy uuid.UUID) (*CustomSetting, error) {
	if err := ValidateKey(req.Key); err != nil {
		return nil, err
	}

	// Set default value type if not provided
	if req.ValueType == "" {
		req.ValueType = "string"
	}

	// Validate value type
	validTypes := map[string]bool{"string": true, "number": true, "boolean": true, "json": true}
	if !validTypes[req.ValueType] {
		return nil, fmt.Errorf("invalid value_type: %s", req.ValueType)
	}

	// Set default editable_by if not provided
	if len(req.EditableBy) == 0 {
		req.EditableBy = []string{"dashboard_admin"}
	}

	// Default metadata to empty object if nil
	if req.Metadata == nil {
		req.Metadata = make(map[string]interface{})
	}

	valueJSON, err := json.Marshal(req.Value)
	if err != nil {
		return nil, err
	}

	metadataJSON, err := json.Marshal(req.Metadata)
	if err != nil {
		return nil, err
	}

	var setting CustomSetting
	var valueJSONResult, metadataJSONResult []byte
	var editableByResult []string

	err = s.db.QueryRow(ctx, `
		INSERT INTO app.settings
		(key, value, value_type, description, editable_by, metadata, created_by, updated_by, category)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $7, 'custom')
		RETURNING id, key, value, value_type, description, editable_by, metadata, created_by, updated_by, created_at, updated_at
	`, req.Key, valueJSON, req.ValueType, req.Description, req.EditableBy, metadataJSON, createdBy).Scan(
		&setting.ID,
		&setting.Key,
		&valueJSONResult,
		&setting.ValueType,
		&setting.Description,
		&editableByResult,
		&metadataJSONResult,
		&setting.CreatedBy,
		&setting.UpdatedBy,
		&setting.CreatedAt,
		&setting.UpdatedAt,
	)

	if err != nil {
		// Check for unique constraint violation
		if database.IsUniqueViolation(err) {
			return nil, ErrCustomSettingDuplicate
		}
		return nil, err
	}

	if err := json.Unmarshal(valueJSONResult, &setting.Value); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(metadataJSONResult, &setting.Metadata); err != nil {
		return nil, err
	}
	setting.EditableBy = editableByResult

	return &setting, nil
}

// GetSetting retrieves a custom setting by key
func (s *CustomSettingsService) GetSetting(ctx context.Context, key string) (*CustomSetting, error) {
	var setting CustomSetting
	var valueJSON, metadataJSON []byte
	var editableBy []string

	err := s.db.QueryRow(ctx, `
		SELECT id, key, value, value_type, description, editable_by, metadata, created_by, updated_by, created_at, updated_at
		FROM app.settings
		WHERE key = $1
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
	if err := json.Unmarshal(metadataJSON, &setting.Metadata); err != nil {
		return nil, err
	}
	setting.EditableBy = editableBy

	return &setting, nil
}

// UpdateSetting updates an existing custom setting
func (s *CustomSettingsService) UpdateSetting(ctx context.Context, key string, req UpdateCustomSettingRequest, updatedBy uuid.UUID, userRole string) (*CustomSetting, error) {
	// First, get the existing setting to check permissions
	existing, err := s.GetSetting(ctx, key)
	if err != nil {
		return nil, err
	}

	// Check if user has permission to edit this setting
	if !CanEditSetting(existing.EditableBy, userRole) {
		return nil, ErrCustomSettingPermissionDenied
	}

	valueJSON, err := json.Marshal(req.Value)
	if err != nil {
		return nil, err
	}

	// Build dynamic update query based on what's provided
	description := existing.Description
	if req.Description != nil {
		description = *req.Description
	}

	editableBy := existing.EditableBy
	if len(req.EditableBy) > 0 {
		editableBy = req.EditableBy
	}

	metadata := existing.Metadata
	if req.Metadata != nil {
		metadata = req.Metadata
	}

	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return nil, err
	}

	var setting CustomSetting
	var valueJSONResult, metadataJSONResult []byte
	var editableByResult []string

	err = s.db.QueryRow(ctx, `
		UPDATE app.settings
		SET value = $1,
		    description = $2,
		    editable_by = $3,
		    metadata = $4,
		    updated_by = $5,
		    updated_at = NOW()
		WHERE key = $6
		RETURNING id, key, value, value_type, description, editable_by, metadata, created_by, updated_by, created_at, updated_at
	`, valueJSON, description, editableBy, metadataJSON, updatedBy, key).Scan(
		&setting.ID,
		&setting.Key,
		&valueJSONResult,
		&setting.ValueType,
		&setting.Description,
		&editableByResult,
		&metadataJSONResult,
		&setting.CreatedBy,
		&setting.UpdatedBy,
		&setting.CreatedAt,
		&setting.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(valueJSONResult, &setting.Value); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(metadataJSONResult, &setting.Metadata); err != nil {
		return nil, err
	}
	setting.EditableBy = editableByResult

	return &setting, nil
}

// DeleteSetting removes a custom setting by key
func (s *CustomSettingsService) DeleteSetting(ctx context.Context, key string, userRole string) error {
	// First, get the existing setting to check permissions
	existing, err := s.GetSetting(ctx, key)
	if err != nil {
		return err
	}

	// Check if user has permission to delete this setting
	if !CanEditSetting(existing.EditableBy, userRole) {
		return ErrCustomSettingPermissionDenied
	}

	result, err := s.db.Exec(ctx, `
		DELETE FROM app.settings WHERE key = $1
	`, key)

	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrCustomSettingNotFound
	}

	return nil
}

// ListSettings retrieves all custom settings, optionally filtered by user role permissions
func (s *CustomSettingsService) ListSettings(ctx context.Context, userRole string) ([]CustomSetting, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, key, value, value_type, description, editable_by, metadata, created_by, updated_by, created_at, updated_at
		FROM app.settings
		WHERE category = 'custom'
		ORDER BY key
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var settings []CustomSetting
	for rows.Next() {
		var setting CustomSetting
		var valueJSON, metadataJSON []byte
		var editableBy []string

		err := rows.Scan(
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
			return nil, err
		}

		if err := json.Unmarshal(valueJSON, &setting.Value); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(metadataJSON, &setting.Metadata); err != nil {
			return nil, err
		}
		setting.EditableBy = editableBy

		settings = append(settings, setting)
	}

	return settings, rows.Err()
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
		derivedKey, err := crypto.DeriveUserKey(s.encryptionKey, *userID)
		if err != nil {
			return nil, fmt.Errorf("failed to derive user key: %w", err)
		}
		encKey = derivedKey
	}

	// Encrypt the value
	encryptedValue, err := crypto.Encrypt(req.Value, encKey)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt secret: %w", err)
	}

	// Store placeholder in value column (never expose real value)
	placeholderValue := map[string]interface{}{"value": "[ENCRYPTED]"}
	valueJSON, _ := json.Marshal(placeholderValue)

	var metadata SecretSettingMetadata
	err = s.db.QueryRow(ctx, `
		INSERT INTO app.settings
		(key, value, value_type, description, is_secret, encrypted_value, user_id, editable_by, category, created_by, updated_by)
		VALUES ($1, $2, 'string', $3, true, $4, $5, ARRAY['dashboard_admin']::TEXT[], 'custom', $6, $6)
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

	err := s.db.QueryRow(ctx, query, args...).Scan(
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
			derivedKey, err := crypto.DeriveUserKey(s.encryptionKey, *userID)
			if err != nil {
				return nil, fmt.Errorf("failed to derive user key: %w", err)
			}
			encKey = derivedKey
		}

		encrypted, err := crypto.Encrypt(*req.Value, encKey)
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

	err = s.db.QueryRow(ctx, query, args...).Scan(
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

	result, err := s.db.Exec(ctx, query, args...)
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

	rows, err := s.db.Query(ctx, query, args...)
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

	err = s.db.QueryRow(ctx, `
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

	err := s.db.QueryRow(ctx, `
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

// GetSystemSetting retrieves a system-level setting (user_id IS NULL)
// This is for public/system settings that any authenticated user can read
func (s *CustomSettingsService) GetSystemSetting(ctx context.Context, key string) (*CustomSetting, error) {
	var setting CustomSetting
	var valueJSON, metadataJSON []byte
	var editableBy []string

	err := s.db.QueryRow(ctx, `
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

	err = s.db.QueryRow(ctx, `
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

	err = s.db.QueryRow(ctx, `
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

// DeleteUserSetting removes a user's setting
func (s *CustomSettingsService) DeleteUserSetting(ctx context.Context, userID uuid.UUID, key string) error {
	result, err := s.db.Exec(ctx, `
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

// ListUserOwnSettings retrieves all non-encrypted settings for a user
func (s *CustomSettingsService) ListUserOwnSettings(ctx context.Context, userID uuid.UUID) ([]UserSetting, error) {
	rows, err := s.db.Query(ctx, `
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

// ============================================================================
// Transaction-accepting method variants (*WithTx)
// These methods accept a pgx.Tx for RLS context support
// ============================================================================

// Querier is an interface that both *database.Connection and pgx.Tx implement
type Querier interface {
	QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row
	Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
	Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error)
}

// CreateSecretSettingWithTx creates a new encrypted secret setting using a transaction
func (s *CustomSettingsService) CreateSecretSettingWithTx(ctx context.Context, tx Querier, req CreateSecretSettingRequest, userID *uuid.UUID, createdBy uuid.UUID) (*SecretSettingMetadata, error) {
	if err := ValidateKey(req.Key); err != nil {
		return nil, err
	}

	// Determine encryption key (user-specific or system)
	encKey := s.encryptionKey
	if userID != nil {
		derivedKey, err := crypto.DeriveUserKey(s.encryptionKey, *userID)
		if err != nil {
			return nil, fmt.Errorf("failed to derive user key: %w", err)
		}
		encKey = derivedKey
	}

	// Encrypt the value
	encryptedValue, err := crypto.Encrypt(req.Value, encKey)
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
		VALUES ($1, $2, 'string', $3, true, $4, $5, ARRAY['dashboard_admin']::TEXT[], 'custom', $6, $6)
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
			derivedKey, err := crypto.DeriveUserKey(s.encryptionKey, *userID)
			if err != nil {
				return nil, fmt.Errorf("failed to derive user key: %w", err)
			}
			encKey = derivedKey
		}

		encrypted, err := crypto.Encrypt(*req.Value, encKey)
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
