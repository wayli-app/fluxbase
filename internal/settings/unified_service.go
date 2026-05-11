package settings

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/config"
	"github.com/nimbleflux/fluxbase/internal/crypto"
	"github.com/nimbleflux/fluxbase/internal/database"
)

// Errors for unified settings service
var (
	ErrNotOverridable     = errors.New("setting is not overridable at tenant level")
	ErrTenantRequired     = errors.New("tenant context required")
	ErrSecretKeyRequired  = errors.New("encryption key required for secret settings")
	ErrInvalidSettingPath = errors.New("invalid setting path")
)

// ResolvedSetting represents a setting with its source information
type ResolvedSetting struct {
	Value         any    `json:"value,omitempty"`
	Source        string `json:"source"` // "config", "instance", "tenant", "default"
	IsSecret      bool   `json:"is_secret,omitempty"`
	IsOverridable bool   `json:"is_overridable,omitempty"`
	IsReadOnly    bool   `json:"is_read_only,omitempty"` // True if from config file (cannot be changed in dashboard)
	DataType      string `json:"data_type,omitempty"`    // "string", "number", "boolean", "object", "array"
}

// InstanceSettings represents instance-level configuration
type InstanceSettings struct {
	Settings            map[string]any `json:"settings"`
	OverridableSettings []string       `json:"overridable_settings,omitempty"`
	CreatedAt           time.Time      `json:"created_at"`
	UpdatedAt           time.Time      `json:"updated_at"`
}

// TenantSettings represents tenant-specific overrides
type TenantSettings struct {
	TenantID  string         `json:"tenant_id"`
	Settings  map[string]any `json:"settings"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

// cachedSetting represents a cached setting entry
type cachedSetting struct {
	value     any
	source    string
	expiresAt time.Time
}

// UnifiedService provides settings resolution with tenant/instance/config fallback
type UnifiedService struct {
	database.TenantAware
	config             *config.Config
	encryptionKey      []byte
	tenantConfigLoader *config.TenantConfigLoader
	cache              map[string]map[string]*cachedSetting // tenantID -> path -> setting
	overridable        map[string]bool                      // cached overridable settings
	cacheMu            sync.RWMutex
	cacheDuration      time.Duration
}

// NewUnifiedService creates a new unified settings service
func NewUnifiedService(db *database.Connection, cfg *config.Config, encryptionKey []byte) *UnifiedService {
	return &UnifiedService{
		TenantAware:   database.TenantAware{DB: db},
		config:        cfg,
		encryptionKey: encryptionKey,
		cache:         make(map[string]map[string]*cachedSetting),
		overridable:   make(map[string]bool),
		cacheDuration: 5 * time.Minute,
	}
}

// SetTenantConfigLoader sets the tenant configuration loader for per-tenant config resolution
func (s *UnifiedService) SetTenantConfigLoader(loader *config.TenantConfigLoader) {
	s.tenantConfigLoader = loader
}

// ResolveSetting resolves a setting using the cascade: tenant -> instance -> config.
// For the default tenant (isDefaultTenant=true), the full cascade applies:
//
//	tenant DB → instance DB → config (YAML/env) → hardcoded default.
//
// For non-default tenants, the config layer is skipped:
//
//	tenant DB → instance DB → hardcoded default.
func (s *UnifiedService) ResolveSetting(ctx context.Context, tenantID, path string, isDefaultTenant bool, tenantSlug ...string) (*ResolvedSetting, error) {
	if path == "" {
		return nil, ErrInvalidSettingPath
	}

	// Check cache first
	if cached := s.getFromCache(tenantID, path); cached != nil {
		return cached, nil
	}

	// Check if tenant override exists
	if tenantID != "" {
		tenantValue, err := s.getTenantSetting(ctx, tenantID, path)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("failed to get tenant setting: %w", err)
		}
		if tenantValue != nil {
			// Check if setting is overridable
			overridable, err := s.IsSettingOverridable(ctx, path)
			if err != nil {
				return nil, fmt.Errorf("failed to check overridable: %w", err)
			}
			if overridable {
				s.addToCache(tenantID, path, tenantValue, "tenant")
				return &ResolvedSetting{
					Value:         tenantValue,
					Source:        "tenant",
					IsOverridable: overridable,
				}, nil
			}
		}
	}

	// Only fall back to instance settings for the default tenant.
	// Non-default tenants must not inherit instance-level values.
	if isDefaultTenant {
		instanceValue, err := s.getInstanceSetting(ctx, path)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("failed to get instance setting: %w", err)
		}
		if instanceValue != nil {
			overridable, _ := s.IsSettingOverridable(ctx, path)
			s.addToCache(tenantID, path, instanceValue, "instance")
			return &ResolvedSetting{
				Value:         instanceValue,
				Source:        "instance",
				IsOverridable: overridable,
			}, nil
		}
	}

	// Resolve config values from the appropriate config source:
	// - Default tenant: use the base config (YAML + env vars)
	// - Non-default tenant: use tenant-specific config overrides (YAML + env vars for that tenant)
	var cfg *config.Config
	if isDefaultTenant {
		cfg = s.config
	} else if s.tenantConfigLoader != nil && len(tenantSlug) > 0 && tenantSlug[0] != "" {
		cfg = s.tenantConfigLoader.GetConfigForSlug(tenantSlug[0], false)
	}
	if cfg != nil {
		configValue := s.getConfigValue(cfg, path)
		if configValue != nil {
			s.addToCache(tenantID, path, configValue, "config")
			return &ResolvedSetting{
				Value:         configValue,
				Source:        "config",
				IsOverridable: false,
				IsReadOnly:    true,
				DataType:      getDataType(configValue),
			}, nil
		}
	}

	return nil, ErrSettingNotFound
}

// ResolveSettingWithDefault resolves a setting with a default fallback
func (s *UnifiedService) ResolveSettingWithDefault(ctx context.Context, tenantID, path string, defaultValue any, isDefaultTenant bool, tenantSlug ...string) (*ResolvedSetting, error) {
	result, err := s.ResolveSetting(ctx, tenantID, path, isDefaultTenant, tenantSlug...)
	if err != nil {
		if errors.Is(err, ErrSettingNotFound) {
			return &ResolvedSetting{
				Value:  defaultValue,
				Source: "default",
			}, nil
		}
		return nil, err
	}
	return result, nil
}

// getFromCache retrieves a cached setting
func (s *UnifiedService) getFromCache(tenantID, path string) *ResolvedSetting {
	s.cacheMu.RLock()
	defer s.cacheMu.RUnlock()

	cacheKey := s.cacheKey(tenantID)
	tenantCache, ok := s.cache[cacheKey]
	if !ok {
		return nil
	}

	cached, exists := tenantCache[path]
	if !exists {
		return nil
	}

	if time.Now().After(cached.expiresAt) {
		return nil
	}

	return &ResolvedSetting{
		Value:         cached.value,
		Source:        cached.source,
		IsOverridable: s.isPathOverridableCached(path),
	}
}

// addToCache adds a setting to cache
func (s *UnifiedService) addToCache(tenantID, path string, value any, source string) {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()

	cacheKey := s.cacheKey(tenantID)
	if _, ok := s.cache[cacheKey]; !ok {
		s.cache[cacheKey] = make(map[string]*cachedSetting)
	}

	s.cache[cacheKey][path] = &cachedSetting{
		value:     value,
		source:    source,
		expiresAt: time.Now().Add(s.cacheDuration),
	}
}

// cacheKey generates a cache key for tenant
func (s *UnifiedService) cacheKey(tenantID string) string {
	if tenantID == "" {
		return "__instance__"
	}
	return tenantID
}

// isPathOverridableCached checks if path is overridable using cached data
func (s *UnifiedService) isPathOverridableCached(path string) bool {
	// Check exact match first
	if overridable, ok := s.overridable[path]; ok {
		return overridable
	}

	// Check prefix match (for nested settings)
	for prefix, overridable := range s.overridable {
		if strings.HasPrefix(path, prefix+".") {
			return overridable
		}
	}

	// Default overridable settings
	defaultOverridable := []string{
		"ai", "auth.oidc", "auth.saml", "email", "storage",
	}
	for _, prefix := range defaultOverridable {
		if path == prefix || strings.HasPrefix(path, prefix+".") {
			return true
		}
	}

	return false
}

// getTenantSetting gets a tenant-specific setting value
func (s *UnifiedService) getTenantSetting(ctx context.Context, tenantID, path string) (any, error) {
	var settingsJSON []byte
	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `
			SELECT settings FROM platform.instance_settings
			WHERE tenant_id = $1
		`, tenantID).Scan(&settingsJSON)
	})
	if err != nil {
		return nil, err
	}

	var settings map[string]any
	if err := json.Unmarshal(settingsJSON, &settings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal tenant settings: %w", err)
	}

	return getNestedValue(settings, path), nil
}

// getInstanceSetting gets an instance-level setting value.
// Uses WrapWithServiceRole to bypass RLS and avoid tenant context interfering
// with the WHERE tenant_id IS NULL query.
func (s *UnifiedService) getInstanceSetting(ctx context.Context, path string) (any, error) {
	var settingsJSON []byte
	err := database.WrapWithServiceRole(ctx, s.DB, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `
			SELECT settings FROM platform.instance_settings
			WHERE tenant_id IS NULL
			LIMIT 1
		`).Scan(&settingsJSON)
	})
	if err != nil {
		return nil, err
	}

	var settings map[string]any
	if err := json.Unmarshal(settingsJSON, &settings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal instance settings: %w", err)
	}

	return getNestedValue(settings, path), nil
}

// getConfigValue gets a value from the given config
func (s *UnifiedService) getConfigValue(cfg *config.Config, path string) any {
	parts := strings.Split(path, ".")
	if len(parts) < 1 {
		return nil
	}

	switch parts[0] {
	case "ai":
		return getConfigAIValue(cfg, parts[1:])
	case "auth":
		return getConfigAuthValue(cfg, parts[1:])
	case "email":
		return getConfigEmailValue(cfg, parts[1:])
	case "storage":
		return getConfigStorageValue(cfg, parts[1:])
	default:
		return nil
	}
}

func getConfigAIValue(cfg *config.Config, parts []string) any {
	if len(parts) < 1 {
		return nil
	}
	switch parts[0] {
	case "enabled":
		return cfg.AI.Enabled
	case "default_model":
		return cfg.AI.DefaultModel
	case "provider_type":
		return cfg.AI.ProviderType
	case "provider_name":
		return cfg.AI.ProviderName
	case "provider_model":
		return cfg.AI.ProviderModel
	case "embedding_enabled":
		return cfg.AI.EmbeddingEnabled
	case "embedding_provider":
		return cfg.AI.EmbeddingProvider
	case "embedding_model":
		return cfg.AI.EmbeddingModel
	case "openai_api_key":
		return cfg.AI.OpenAIAPIKey
	case "openai_base_url":
		return cfg.AI.OpenAIBaseURL
	case "azure_api_key":
		return cfg.AI.AzureAPIKey
	case "azure_endpoint":
		return cfg.AI.AzureEndpoint
	}
	return nil
}

func getConfigAuthValue(cfg *config.Config, parts []string) any {
	if len(parts) < 1 {
		return nil
	}
	switch parts[0] {
	case "signup_enabled":
		return cfg.Auth.SignupEnabled
	case "magic_link_enabled":
		return cfg.Auth.MagicLinkEnabled
	case "oauth_providers":
		return cfg.Auth.OAuthProviders
	case "saml_providers":
		return cfg.Auth.SAMLProviders
	}
	return nil
}

func getConfigEmailValue(cfg *config.Config, parts []string) any {
	if len(parts) < 1 {
		return nil
	}
	switch parts[0] {
	case "enabled":
		return cfg.Email.Enabled
	case "provider":
		return cfg.Email.Provider
	case "from_address":
		return cfg.Email.FromAddress
	case "from_name":
		return cfg.Email.FromName
	case "smtp_host":
		return cfg.Email.SMTPHost
	case "smtp_port":
		return cfg.Email.SMTPPort
	case "smtp_username":
		return cfg.Email.SMTPUsername
	case "smtp_tls":
		return cfg.Email.SMTPTLS
	}
	return nil
}

func getConfigStorageValue(cfg *config.Config, parts []string) any {
	if len(parts) < 1 {
		return nil
	}
	switch parts[0] {
	case "enabled":
		return cfg.Storage.Enabled
	case "provider":
		return cfg.Storage.Provider
	case "max_upload_size":
		return cfg.Storage.MaxUploadSize
	}
	return nil
}

// getNestedValue retrieves a value from a nested map using dot notation
func getNestedValue(data map[string]any, path string) any {
	parts := strings.Split(path, ".")
	current := data

	for i, part := range parts {
		if part == "" {
			return nil
		}

		val, exists := current[part]
		if !exists {
			return nil
		}

		if i == len(parts)-1 {
			return val
		}

		next, ok := val.(map[string]any)
		if !ok {
			return nil
		}
		current = next
	}

	return nil
}

// setNestedValue sets a value in a nested map using dot notation
func setNestedValue(data map[string]any, path string, value any) {
	parts := strings.Split(path, ".")
	current := data

	for i, part := range parts {
		if part == "" {
			return
		}

		if i == len(parts)-1 {
			current[part] = value
			return
		}

		next, exists := current[part]
		if !exists {
			next = make(map[string]any)
			current[part] = next
		}

		nextMap, ok := next.(map[string]any)
		if !ok {
			nextMap = make(map[string]any)
			current[part] = nextMap
		}
		current = nextMap
	}
}

// SetInstanceSetting sets an instance-level setting value
func (s *UnifiedService) SetInstanceSetting(ctx context.Context, path string, value any, isSecret bool) error {
	// Get or create instance settings
	settings, err := s.getOrCreateInstanceSettingsMap(ctx)
	if err != nil {
		return fmt.Errorf("failed to get instance settings: %w", err)
	}

	// Encrypt if secret
	if isSecret {
		if len(s.encryptionKey) == 0 {
			return ErrSecretKeyRequired
		}
		encrypted, err := crypto.EncryptWithBytesKey(fmt.Sprintf("%v", value), s.encryptionKey)
		if err != nil {
			return fmt.Errorf("failed to encrypt secret value: %w", err)
		}
		value = encrypted
	}

	// Set the value
	setNestedValue(settings, path, value)

	// Marshal and save
	settingsJSON, err := json.Marshal(settings)
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	err = database.WrapWithServiceRole(ctx, s.DB, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			UPDATE platform.instance_settings
			SET settings = $1, updated_at = NOW()
			WHERE tenant_id IS NULL
		`, settingsJSON)
		return err
	})
	if err != nil {
		return fmt.Errorf("failed to update instance settings: %w", err)
	}

	// Invalidate cache
	s.InvalidateCache("", path)

	log.Info().Str("path", path).Bool("is_secret", isSecret).Msg("Set instance setting")
	return nil
}

// getOrCreateInstanceSettingsMap gets instance settings or creates empty map.
// Uses WrapWithServiceRole (not WithTenant) to avoid the set_tenant_id_from_context()
// trigger overriding tenant_id=NULL with the session tenant UUID.
func (s *UnifiedService) getOrCreateInstanceSettingsMap(ctx context.Context) (map[string]any, error) {
	var settingsJSON []byte
	err := database.WrapWithServiceRole(ctx, s.DB, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `
			SELECT settings FROM platform.instance_settings WHERE tenant_id IS NULL LIMIT 1
		`).Scan(&settingsJSON)
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			settings := make(map[string]any)
			settingsJSON, _ := json.Marshal(settings)
			err = database.WrapWithServiceRole(ctx, s.DB, func(tx pgx.Tx) error {
				_, err := tx.Exec(ctx, `
					INSERT INTO platform.instance_settings (tenant_id, settings, created_at, updated_at)
					VALUES (NULL, $1, NOW(), NOW())
				`, settingsJSON)
				return err
			})
			if err != nil {
				return nil, fmt.Errorf("failed to create instance settings: %w", err)
			}
			return settings, nil
		}
		return nil, err
	}

	var settings map[string]any
	if err := json.Unmarshal(settingsJSON, &settings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal instance settings: %w", err)
	}

	return settings, nil
}

// SetTenantSetting sets a tenant-specific setting value
func (s *UnifiedService) SetTenantSetting(ctx context.Context, tenantID, path string, value any, isSecret bool) error {
	if tenantID == "" {
		return ErrTenantRequired
	}

	// Check if setting is overridable
	overridable, err := s.IsSettingOverridable(ctx, path)
	if err != nil {
		return fmt.Errorf("failed to check overridable: %w", err)
	}
	if !overridable {
		return ErrNotOverridable
	}

	// Get or create tenant settings
	settings, err := s.getOrCreateTenantSettingsMap(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("failed to get tenant settings: %w", err)
	}

	// Encrypt if secret
	if isSecret {
		if len(s.encryptionKey) == 0 {
			return ErrSecretKeyRequired
		}
		encrypted, err := crypto.EncryptWithBytesKey(fmt.Sprintf("%v", value), s.encryptionKey)
		if err != nil {
			return fmt.Errorf("failed to encrypt secret value: %w", err)
		}
		value = encrypted
	}

	// Set the value
	setNestedValue(settings, path, value)

	// Marshal and save
	settingsJSON, err := json.Marshal(settings)
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	err = s.WithTenant(ctx, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			UPDATE platform.instance_settings
			SET settings = $1, updated_at = NOW()
			WHERE tenant_id = $2
		`, settingsJSON, tenantID)
		return err
	})
	if err != nil {
		return fmt.Errorf("failed to update tenant settings: %w", err)
	}

	// Update cache
	s.addToCache(tenantID, path, value, "tenant")

	log.Info().Str("tenant_id", tenantID).Str("path", path).Bool("is_secret", isSecret).Msg("Set tenant setting")
	return nil
}

// getOrCreateTenantSettingsMap gets tenant settings or creates empty map
func (s *UnifiedService) getOrCreateTenantSettingsMap(ctx context.Context, tenantID string) (map[string]any, error) {
	var settingsJSON []byte
	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `
			SELECT settings FROM platform.instance_settings WHERE tenant_id = $1
		`, tenantID).Scan(&settingsJSON)
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Create new tenant settings row in instance_settings
			settings := make(map[string]any)
			settingsJSON, _ := json.Marshal(settings)
			err = s.WithTenant(ctx, func(tx pgx.Tx) error {
				_, err := tx.Exec(ctx, `
					INSERT INTO platform.instance_settings (tenant_id, settings, created_at, updated_at)
					VALUES ($1, $2, NOW(), NOW())
				`, tenantID, settingsJSON)
				return err
			})
			if err != nil {
				return nil, fmt.Errorf("failed to create tenant settings: %w", err)
			}
			return settings, nil
		}
		return nil, err
	}

	var settings map[string]any
	if err := json.Unmarshal(settingsJSON, &settings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal tenant settings: %w", err)
	}

	return settings, nil
}

// DeleteTenantSetting removes a tenant-specific setting
func (s *UnifiedService) DeleteTenantSetting(ctx context.Context, tenantID, path string) error {
	if tenantID == "" {
		return ErrTenantRequired
	}

	// Get current settings
	settings, err := s.getOrCreateTenantSettingsMap(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("failed to get tenant settings: %w", err)
	}

	// Remove the path
	deleteNestedValue(settings, path)

	// Save updated settings
	settingsJSON, err := json.Marshal(settings)
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	err = s.WithTenant(ctx, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			UPDATE platform.instance_settings
			SET settings = $1, updated_at = NOW()
			WHERE tenant_id = $2
		`, settingsJSON, tenantID)
		return err
	})
	if err != nil {
		return fmt.Errorf("failed to update tenant settings: %w", err)
	}

	// Invalidate cache for this path
	s.cacheMu.Lock()
	if tenantCache, ok := s.cache[tenantID]; ok {
		delete(tenantCache, path)
	}
	s.cacheMu.Unlock()

	log.Info().Str("tenant_id", tenantID).Str("path", path).Msg("Deleted tenant setting")
	return nil
}

// deleteNestedValue removes a value from a nested map using dot notation
func deleteNestedValue(data map[string]any, path string) {
	parts := strings.Split(path, ".")
	current := data

	for i, part := range parts {
		if part == "" {
			return
		}

		if i == len(parts)-1 {
			delete(current, part)
			return
		}

		next, exists := current[part]
		if !exists {
			return
		}

		nextMap, ok := next.(map[string]any)
		if !ok {
			return
		}
		current = nextMap
	}
}

// GetInstanceSettings returns all instance-level settings
func (s *UnifiedService) GetInstanceSettings(ctx context.Context) (*InstanceSettings, error) {
	var settingsJSON []byte
	var overridableJSON []byte
	var createdAt, updatedAt time.Time

	err := database.WrapWithServiceRole(ctx, s.DB, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `
			SELECT settings, overridable_settings, created_at, updated_at
			FROM platform.instance_settings
			WHERE tenant_id IS NULL
			LIMIT 1
		`).Scan(&settingsJSON, &overridableJSON, &createdAt, &updatedAt)
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return &InstanceSettings{
				Settings:            make(map[string]any),
				OverridableSettings: []string{},
				CreatedAt:           time.Now(),
				UpdatedAt:           time.Now(),
			}, nil
		}
		return nil, fmt.Errorf("failed to get instance settings: %w", err)
	}

	var settings map[string]any
	if len(settingsJSON) > 0 {
		if err := json.Unmarshal(settingsJSON, &settings); err != nil {
			return nil, fmt.Errorf("failed to unmarshal instance settings: %w", err)
		}
	} else {
		settings = make(map[string]any)
	}

	var overridableSettings []string
	if len(overridableJSON) > 0 {
		if err := json.Unmarshal(overridableJSON, &overridableSettings); err != nil {
			return nil, fmt.Errorf("failed to unmarshal overridable settings: %w", err)
		}
	}

	return &InstanceSettings{
		Settings:            settings,
		OverridableSettings: overridableSettings,
		CreatedAt:           createdAt,
		UpdatedAt:           updatedAt,
	}, nil
}

// GetTenantSettings returns all tenant-specific settings
func (s *UnifiedService) GetTenantSettings(ctx context.Context, tenantID string) (*TenantSettings, error) {
	if tenantID == "" {
		return nil, ErrTenantRequired
	}

	var settingsJSON []byte
	var createdAt, updatedAt time.Time

	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `
			SELECT settings, created_at, updated_at
			FROM platform.instance_settings
			WHERE tenant_id = $1
		`, tenantID).Scan(&settingsJSON, &createdAt, &updatedAt)
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return &TenantSettings{
				TenantID:  tenantID,
				Settings:  make(map[string]any),
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}, nil
		}
		return nil, fmt.Errorf("failed to get tenant settings: %w", err)
	}

	var settings map[string]any
	if len(settingsJSON) > 0 {
		if err := json.Unmarshal(settingsJSON, &settings); err != nil {
			return nil, fmt.Errorf("failed to unmarshal tenant settings: %w", err)
		}
	} else {
		settings = make(map[string]any)
	}

	return &TenantSettings{
		TenantID:  tenantID,
		Settings:  settings,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}, nil
}

// IsSettingOverridable checks if a setting can be overridden at tenant level
func (s *UnifiedService) IsSettingOverridable(ctx context.Context, path string) (bool, error) {
	// Check cache first
	s.cacheMu.RLock()
	if overridable, ok := s.overridable[path]; ok {
		s.cacheMu.RUnlock()
		return overridable, nil
	}
	s.cacheMu.RUnlock()

	// Get overridable settings list from database
	var overridableJSON []byte
	err := database.WrapWithServiceRole(ctx, s.DB, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `
			SELECT overridable_settings FROM platform.instance_settings WHERE tenant_id IS NULL LIMIT 1
		`).Scan(&overridableJSON)
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// No instance settings row, use defaults
			return s.isPathOverridableCached(path), nil
		}
		return false, fmt.Errorf("failed to get overridable settings: %w", err)
	}

	// If NULL, all settings are overridable
	if len(overridableJSON) == 0 {
		return true, nil
	}

	var overridableSettings []string
	if err := json.Unmarshal(overridableJSON, &overridableSettings); err != nil {
		return false, fmt.Errorf("failed to unmarshal overridable settings: %w", err)
	}

	// Check if path matches any overridable setting (with wildcard support)
	for _, allowed := range overridableSettings {
		if path == allowed || strings.HasPrefix(path, allowed+".") {
			// Cache the result
			s.cacheMu.Lock()
			s.overridable[path] = true
			s.cacheMu.Unlock()
			return true, nil
		}
	}

	// Cache negative result
	s.cacheMu.Lock()
	s.overridable[path] = false
	s.cacheMu.Unlock()
	return false, nil
}

// SetOverridableSettings sets which settings can be overridden at tenant level
func (s *UnifiedService) SetOverridableSettings(ctx context.Context, paths []string) error {
	pathsJSON, err := json.Marshal(paths)
	if err != nil {
		return fmt.Errorf("failed to marshal overridable settings: %w", err)
	}

	err = database.WrapWithServiceRole(ctx, s.DB, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			UPDATE platform.instance_settings
			SET overridable_settings = $1, updated_at = NOW()
			WHERE tenant_id IS NULL
		`, pathsJSON)
		return err
	})
	if err != nil {
		return fmt.Errorf("failed to update overridable settings: %w", err)
	}

	// Clear overridable cache
	s.cacheMu.Lock()
	s.overridable = make(map[string]bool)
	s.cacheMu.Unlock()

	log.Info().Int("count", len(paths)).Msg("Updated overridable settings")
	return nil
}

// InvalidateCache clears cache entries
func (s *UnifiedService) InvalidateCache(tenantID, path string) {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()

	if tenantID == "" {
		if path == "" {
			// Clear all cache
			s.cache = make(map[string]map[string]*cachedSetting)
			log.Info().Msg("Cleared all settings cache")
		} else {
			// Clear specific path from all tenants
			for _, tenantCache := range s.cache {
				delete(tenantCache, path)
			}
			log.Info().Str("path", path).Msg("Cleared setting from all caches")
		}
		return
	}

	cacheKey := s.cacheKey(tenantID)
	if path == "" {
		// Clear all cache for tenant
		delete(s.cache, cacheKey)
		log.Info().Str("tenant_id", tenantID).Msg("Cleared tenant settings cache")
		return
	}

	// Clear specific path
	if tenantCache, ok := s.cache[cacheKey]; ok {
		delete(tenantCache, path)
		log.Info().Str("tenant_id", tenantID).Str("path", path).Msg("Cleared specific setting from cache")
	}
}

// DecryptSecret decrypts an encrypted secret value
func (s *UnifiedService) DecryptSecret(encryptedValue string) (string, error) {
	if len(s.encryptionKey) == 0 {
		return "", ErrSecretKeyRequired
	}

	decrypted, err := crypto.DecryptWithBytesKey(encryptedValue, s.encryptionKey)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	return decrypted, nil
}

// getDataType returns the JSON type name for a value
func getDataType(value any) string {
	if value == nil {
		return "null"
	}
	switch value.(type) {
	case string:
		return "string"
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return "number"
	case bool:
		return "boolean"
	case []any:
		return "array"
	case map[string]any:
		return "object"
	default:
		return "unknown"
	}
}
