package settings

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/fluxbase-eu/fluxbase/internal/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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
}

// UpdateCustomSettingRequest represents the request to update a custom setting
type UpdateCustomSettingRequest struct {
	Value       map[string]interface{} `json:"value"`
	Description *string                `json:"description,omitempty"`
	EditableBy  []string               `json:"editable_by,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// CustomSettingsService handles custom admin-managed settings
type CustomSettingsService struct {
	db *database.Connection
}

// NewCustomSettingsService creates a new custom settings service
func NewCustomSettingsService(db *database.Connection) *CustomSettingsService {
	return &CustomSettingsService{db: db}
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
