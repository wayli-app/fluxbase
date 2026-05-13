package auth

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/nimbleflux/fluxbase/internal/database"
	"github.com/nimbleflux/fluxbase/internal/settings"
)

// ErrSettingNotFound is returned when a system setting is not found
var ErrSettingNotFound = errors.New("system setting not found")

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
	cache *settings.SettingsCache
}

// NewSystemSettingsService creates a new system settings service
func NewSystemSettingsService(db *database.Connection) *SystemSettingsService {
	return &SystemSettingsService{db: db}
}

// SetCache sets the settings cache for invalidation on updates
func (s *SystemSettingsService) SetCache(cache *settings.SettingsCache) {
	s.cache = cache
}

type systemSettingsProvider struct {
	svc *SystemSettingsService
}

func (p *systemSettingsProvider) GetSetting(ctx context.Context, key string) (*settings.SettingEntry, error) {
	s, err := p.svc.GetSetting(ctx, key)
	if err != nil {
		return nil, err
	}
	return &settings.SettingEntry{Value: s.Value}, nil
}

func (p *systemSettingsProvider) GetSettings(ctx context.Context, keys []string) (map[string]*settings.SettingEntry, error) {
	m, err := p.svc.GetSettings(ctx, keys)
	if err != nil {
		return nil, err
	}
	result := make(map[string]*settings.SettingEntry, len(m))
	for k, v := range m {
		result[k] = &settings.SettingEntry{Value: v.Value}
	}
	return result, nil
}

func (s *SystemSettingsService) AsProvider() settings.SettingProvider {
	return &systemSettingsProvider{svc: s}
}

// IsSetupComplete checks if the initial setup has been completed
func (s *SystemSettingsService) IsSetupComplete(ctx context.Context) (bool, error) {
	var exists bool
	err := database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `
			SELECT EXISTS(
				SELECT 1 FROM app.settings
				WHERE key = 'setup_completed'
			)
		`).Scan(&exists)
	})
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

	return database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			INSERT INTO app.settings (key, value, description, category)
			VALUES ($1, $2, $3, 'system')
		`, "setup_completed", valueJSON, "Tracks initial setup completion")
		return err
	})
}

// GetSetupInfo retrieves setup completion information
func (s *SystemSettingsService) GetSetupInfo(ctx context.Context) (*SetupCompleteValue, error) {
	var valueJSON []byte
	err := database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `
			SELECT value FROM app.settings
			WHERE key = 'setup_completed'
		`).Scan(&valueJSON)
	})
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

	err := database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `
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
	})
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

	settings := make(map[string]*SystemSetting, len(keys))

	err := database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT id, key, value, description, created_at, updated_at
			FROM app.settings
			WHERE key = ANY($1)
		`, keys)
		if err != nil {
			return err
		}
		defer rows.Close()

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
				return err
			}

			if err := json.Unmarshal(valueJSON, &setting.Value); err != nil {
				// Handle legacy format where value is stored as raw primitive
				var rawValue interface{}
				if rawErr := json.Unmarshal(valueJSON, &rawValue); rawErr == nil {
					setting.Value = map[string]interface{}{"value": rawValue}
				} else {
					return err
				}
			}

			settings[setting.Key] = &setting
		}

		return rows.Err()
	})
	if err != nil {
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

	err = database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			INSERT INTO app.settings (key, value, description, category)
			VALUES ($1, $2, $3, 'system')
			ON CONFLICT (key) WHERE user_id IS NULL DO UPDATE
			SET value = EXCLUDED.value,
			    description = EXCLUDED.description,
			    updated_at = NOW()
		`, key, valueJSON, description)
		return err
	})
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
	var rowsAffected int64

	err := database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `
			DELETE FROM app.settings WHERE key = $1
		`, key)
		if err != nil {
			return err
		}
		rowsAffected = tag.RowsAffected()
		return nil
	})
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
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
	var settings []SystemSetting

	err := database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT id, key, value, description, created_at, updated_at
			FROM app.settings
			ORDER BY key
		`)
		if err != nil {
			return err
		}
		defer rows.Close()

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
				return err
			}

			if err := json.Unmarshal(valueJSON, &setting.Value); err != nil {
				// Handle legacy format where value is stored as raw primitive (e.g., "true", "false")
				// instead of the expected {"value": <primitive>} format
				var rawValue interface{}
				if rawErr := json.Unmarshal(valueJSON, &rawValue); rawErr == nil {
					setting.Value = map[string]interface{}{"value": rawValue}
				} else {
					return err
				}
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

// ErrInstanceSettingNotFound is returned when an instance-level setting is not found
var ErrInstanceSettingNotFound = errors.New("instance-level setting not found")

// ErrTenantSettingNotFound is returned when a tenant-level setting is not found
var ErrTenantSettingNotFound = errors.New("tenant-level setting not found")

// InstanceSetting represents an instance-level (platform-wide) configuration setting
type InstanceSetting struct {
	ID          uuid.UUID              `json:"id"`
	Key         string                 `json:"key"`
	Value       map[string]interface{} `json:"value"`
	Description *string                `json:"description,omitempty"`
	IsPublic    bool                   `json:"is_public"`
	Category    string                 `json:"category"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// GetInstanceSetting retrieves an instance-level setting from app.instance_settings
func (s *SystemSettingsService) GetInstanceSetting(ctx context.Context, key string) (*InstanceSetting, error) {
	var setting InstanceSetting
	var valueJSON []byte
	err := database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `
	        SELECT id, key, value, description, is_public, category, created_at, updated_at
	        FROM app.instance_settings
	        WHERE key = $1
	    `, key).Scan(
			&setting.ID,
			&setting.Key,
			&valueJSON,
			&setting.Description,
			&setting.IsPublic,
			&setting.Category,
			&setting.CreatedAt,
			&setting.UpdatedAt,
		)
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInstanceSettingNotFound
		}
		return nil, err
	}
	if err := json.Unmarshal(valueJSON, &setting.Value); err != nil {
		return nil, err
	}
	return &setting, nil
}

// SetInstanceSetting creates or updates an instance-level setting
func (s *SystemSettingsService) SetInstanceSetting(ctx context.Context, key string, value map[string]interface{}, description string) error {
	valueJSON, err := json.Marshal(value)
	if err != nil {
		return err
	}
	err = database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
	        INSERT INTO app.instance_settings (key, value, description, category)
	        VALUES ($1, $2, $3, 'system')
	        ON CONFLICT (key) DO UPDATE
	        SET value = EXCLUDED.value,
	            description = EXCLUDED.description,
	            updated_at = NOW()
	    `, key, valueJSON, description)
		return err
	})
	if err != nil {
		return err
	}
	// Invalidate cache for this key
	if s.cache != nil {
		s.cache.Invalidate(key)
	}
	return nil
}

// DeleteInstanceSetting removes an instance-level setting
func (s *SystemSettingsService) DeleteInstanceSetting(ctx context.Context, key string) error {
	var rowsAffected int64
	err := database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `
	        DELETE FROM app.instance_settings WHERE key = $1
	    `, key)
		if err != nil {
			return err
		}
		rowsAffected = tag.RowsAffected()
		return nil
	})
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrInstanceSettingNotFound
	}
	// Invalidate cache for this key
	if s.cache != nil {
		s.cache.Invalidate(key)
	}
	return nil
}

// GetSettingWithInheritance retrieves a setting with inheritance logic
// Priority: tenant settings > instance settings > error
func (s *SystemSettingsService) GetSettingWithInheritance(ctx context.Context, key string, tenantID string) (*SystemSetting, error) {
	// First try tenant settings if tenant context is provided
	if tenantID != "" {
		setting, err := s.GetTenantSetting(ctx, key, tenantID)
		if err == nil {
			return setting, nil
		}
		// If tenant setting not found, fall through to instance settings
		if !errors.Is(err, ErrTenantSettingNotFound) {
			return nil, err
		}
	}

	// Fall back to instance settings (no tenant context)
	instanceSetting, err := s.GetInstanceSetting(ctx, key)
	if err != nil {
		return nil, err
	}

	// Convert InstanceSetting to SystemSetting
	return &SystemSetting{
		ID:          instanceSetting.ID,
		Key:         instanceSetting.Key,
		Value:       instanceSetting.Value,
		Description: instanceSetting.Description,
		CreatedAt:   instanceSetting.CreatedAt,
		UpdatedAt:   instanceSetting.UpdatedAt,
	}, nil
}

// GetTenantSetting retrieves a tenant-level setting from app.settings
func (s *SystemSettingsService) GetTenantSetting(ctx context.Context, key string, tenantID string) (*SystemSetting, error) {
	var setting SystemSetting
	var valueJSON []byte
	err := database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `
	        SELECT id, key, value, description, created_at, updated_at
	        FROM app.settings
	        WHERE key = $1 AND tenant_id = $2
	    `, key, tenantID).Scan(
			&setting.ID,
			&setting.Key,
			&valueJSON,
			&setting.Description,
			&setting.CreatedAt,
			&setting.UpdatedAt,
		)
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTenantSettingNotFound
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

// SetTenantSetting creates or updates a tenant-level setting
func (s *SystemSettingsService) SetTenantSetting(ctx context.Context, key string, tenantID string, value map[string]interface{}, description string) error {
	valueJSON, err := json.Marshal(value)
	if err != nil {
		return err
	}
	err = database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
	        INSERT INTO app.settings (key, value, description, category, tenant_id)
	        VALUES ($1, $2, $3, 'custom', $4)
	        ON CONFLICT (key, tenant_id) DO UPDATE
	        SET value = EXCLUDED.value,
	            description = EXCLUDED.description,
	            updated_at = NOW()
	    `, key, valueJSON, description, tenantID)
		return err
	})
	if err != nil {
		return err
	}
	// Invalidate cache for this key
	if s.cache != nil {
		s.cache.Invalidate(key)
	}
	return nil
}
