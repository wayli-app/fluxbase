package auth

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/fluxbase-eu/fluxbase/internal/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var (
	// ErrSettingNotFound is returned when a system setting is not found
	ErrSettingNotFound = errors.New("system setting not found")
)

// SystemSetting represents a system-wide configuration setting
type SystemSetting struct {
	ID             uuid.UUID              `json:"id"`
	Key            string                 `json:"key"`
	Value          map[string]interface{} `json:"value"`
	Description    *string                `json:"description,omitempty"`
	IsOverridden   bool                   `json:"is_overridden"`
	OverrideSource string                 `json:"override_source,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
}

// SetupCompleteValue represents the value stored for setup_completed setting
type SetupCompleteValue struct {
	Completed       bool       `json:"completed"`
	CompletedAt     time.Time  `json:"completed_at"`
	FirstAdminID    *uuid.UUID `json:"first_admin_id,omitempty"`
	FirstAdminEmail *string    `json:"first_admin_email,omitempty"`
}

// SystemSettingsService handles system-wide settings
type SystemSettingsService struct {
	db    *database.Connection
	cache *SettingsCache
}

// NewSystemSettingsService creates a new system settings service
func NewSystemSettingsService(db *database.Connection) *SystemSettingsService {
	return &SystemSettingsService{db: db}
}

// SetCache sets the settings cache for invalidation on updates
func (s *SystemSettingsService) SetCache(cache *SettingsCache) {
	s.cache = cache
}

// IsSetupComplete checks if the initial setup has been completed
func (s *SystemSettingsService) IsSetupComplete(ctx context.Context) (bool, error) {
	var exists bool
	err := s.db.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM app.settings
			WHERE key = 'setup_completed'
		)
	`).Scan(&exists)

	if err != nil {
		return false, err
	}

	return exists, nil
}

// MarkSetupComplete marks the setup as completed
func (s *SystemSettingsService) MarkSetupComplete(ctx context.Context, adminID uuid.UUID, adminEmail string) error {
	// Check if already marked complete
	complete, err := s.IsSetupComplete(ctx)
	if err != nil {
		return err
	}
	if complete {
		return errors.New("setup already marked as completed")
	}

	// Create setup completion record
	value := SetupCompleteValue{
		Completed:       true,
		CompletedAt:     time.Now(),
		FirstAdminID:    &adminID,
		FirstAdminEmail: &adminEmail,
	}

	valueJSON, err := json.Marshal(value)
	if err != nil {
		return err
	}

	_, err = s.db.Exec(ctx, `
		INSERT INTO app.settings (key, value, description, category)
		VALUES ($1, $2, $3, 'system')
	`, "setup_completed", valueJSON, "Tracks initial setup completion")

	return err
}

// GetSetupInfo retrieves setup completion information
func (s *SystemSettingsService) GetSetupInfo(ctx context.Context) (*SetupCompleteValue, error) {
	var valueJSON []byte
	err := s.db.QueryRow(ctx, `
		SELECT value FROM app.settings
		WHERE key = 'setup_completed'
	`).Scan(&valueJSON)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSettingNotFound
		}
		return nil, err
	}

	var value SetupCompleteValue
	if err := json.Unmarshal(valueJSON, &value); err != nil {
		return nil, err
	}

	return &value, nil
}

// GetSetting retrieves a system setting by key
func (s *SystemSettingsService) GetSetting(ctx context.Context, key string) (*SystemSetting, error) {
	var setting SystemSetting
	var valueJSON []byte

	err := s.db.QueryRow(ctx, `
		SELECT id, key, value, description, created_at, updated_at
		FROM app.settings
		WHERE key = $1
	`, key).Scan(
		&setting.ID,
		&setting.Key,
		&valueJSON,
		&setting.Description,
		&setting.CreatedAt,
		&setting.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSettingNotFound
		}
		return nil, err
	}

	if err := json.Unmarshal(valueJSON, &setting.Value); err != nil {
		// Handle legacy format where value is stored as raw primitive
		var rawValue interface{}
		if rawErr := json.Unmarshal(valueJSON, &rawValue); rawErr == nil {
			setting.Value = map[string]interface{}{"value": rawValue}
		} else {
			return nil, err
		}
	}

	return &setting, nil
}

// GetSettings retrieves multiple settings at once using a batch query
// Returns a map of key -> setting for all found settings
func (s *SystemSettingsService) GetSettings(ctx context.Context, keys []string) (map[string]*SystemSetting, error) {
	if len(keys) == 0 {
		return make(map[string]*SystemSetting), nil
	}

	rows, err := s.db.Query(ctx, `
		SELECT id, key, value, description, created_at, updated_at
		FROM app.settings
		WHERE key = ANY($1)
	`, keys)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	settings := make(map[string]*SystemSetting, len(keys))
	for rows.Next() {
		var setting SystemSetting
		var valueJSON []byte

		if err := rows.Scan(
			&setting.ID,
			&setting.Key,
			&valueJSON,
			&setting.Description,
			&setting.CreatedAt,
			&setting.UpdatedAt,
		); err != nil {
			return nil, err
		}

		if err := json.Unmarshal(valueJSON, &setting.Value); err != nil {
			// Handle legacy format where value is stored as raw primitive
			var rawValue interface{}
			if rawErr := json.Unmarshal(valueJSON, &rawValue); rawErr == nil {
				setting.Value = map[string]interface{}{"value": rawValue}
			} else {
				return nil, err
			}
		}

		settings[setting.Key] = &setting
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return settings, nil
}

// SetSetting creates or updates a system setting
func (s *SystemSettingsService) SetSetting(ctx context.Context, key string, value map[string]interface{}, description string) error {
	valueJSON, err := json.Marshal(value)
	if err != nil {
		return err
	}

	_, err = s.db.Exec(ctx, `
		INSERT INTO app.settings (key, value, description, category)
		VALUES ($1, $2, $3, 'system')
		ON CONFLICT (key) DO UPDATE
		SET value = EXCLUDED.value,
		    description = EXCLUDED.description,
		    updated_at = NOW()
	`, key, valueJSON, description)

	if err != nil {
		return err
	}

	// Invalidate cache for this key
	if s.cache != nil {
		s.cache.Invalidate(key)
	}

	return nil
}

// DeleteSetting removes a system setting by key
func (s *SystemSettingsService) DeleteSetting(ctx context.Context, key string) error {
	result, err := s.db.Exec(ctx, `
		DELETE FROM app.settings WHERE key = $1
	`, key)

	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrSettingNotFound
	}

	// Invalidate cache for this key
	if s.cache != nil {
		s.cache.Invalidate(key)
	}

	return nil
}

// ListSettings retrieves all system settings
func (s *SystemSettingsService) ListSettings(ctx context.Context) ([]SystemSetting, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, key, value, description, created_at, updated_at
		FROM app.settings
		ORDER BY key
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var settings []SystemSetting
	for rows.Next() {
		var setting SystemSetting
		var valueJSON []byte

		err := rows.Scan(
			&setting.ID,
			&setting.Key,
			&valueJSON,
			&setting.Description,
			&setting.CreatedAt,
			&setting.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		if err := json.Unmarshal(valueJSON, &setting.Value); err != nil {
			// Handle legacy format where value is stored as raw primitive (e.g., "true", "false")
			// instead of the expected {"value": <primitive>} format
			var rawValue interface{}
			if rawErr := json.Unmarshal(valueJSON, &rawValue); rawErr == nil {
				setting.Value = map[string]interface{}{"value": rawValue}
			} else {
				return nil, err
			}
		}

		settings = append(settings, setting)
	}

	return settings, rows.Err()
}
