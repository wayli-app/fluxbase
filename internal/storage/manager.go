package storage

import (
	"context"
	"sync"

	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/config"
	"github.com/nimbleflux/fluxbase/internal/observability"
)

// Manager manages tenant-specific storage services
type Manager struct {
	mu         sync.RWMutex
	services   map[string]*Service // slug -> service (empty string = base)
	baseConfig *config.StorageConfig
	baseURL    string
	jwtSecret  string
	metrics    *observability.Metrics
}

// NewManager creates a storage manager
func NewManager(baseCfg *config.StorageConfig, baseURL, jwtSecret string, metrics *observability.Metrics) (*Manager, error) {
	baseService, err := NewService(baseCfg, baseURL, jwtSecret, metrics)
	if err != nil {
		return nil, err
	}

	return &Manager{
		services:   map[string]*Service{"": baseService},
		baseConfig: baseCfg,
		baseURL:    baseURL,
		jwtSecret:  jwtSecret,
		metrics:    metrics,
	}, nil
}

// SetMetrics sets the metrics instance for all storage services
func (m *Manager) SetMetrics(metrics *observability.Metrics) {
	m.mu.Lock()
	m.metrics = metrics
	for _, svc := range m.services {
		svc.metrics = metrics
	}
	m.mu.Unlock()
}

// GetService returns the storage service for a tenant configuration.
// If tenantCfg is nil or matches the base config, the base service is returned.
func (m *Manager) GetService(tenantCfg *config.StorageConfig) (*Service, error) {
	// If no tenant config or same as base, use base service
	if tenantCfg == nil || configEqual(tenantCfg, m.baseConfig) {
		m.mu.RLock()
		svc := m.services[""]
		m.mu.RUnlock()
		return svc, nil
	}

	// Generate a cache key based on config
	key := configKey(tenantCfg)

	// Check cache
	m.mu.RLock()
	svc, exists := m.services[key]
	m.mu.RUnlock()

	if exists {
		return svc, nil
	}

	// Create new tenant-specific service
	return m.createService(key, tenantCfg)
}

// GetBaseService returns the base storage service
func (m *Manager) GetBaseService() *Service {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.services[""]
}

// createService creates and caches a new storage service for the given config
func (m *Manager) createService(key string, cfg *config.StorageConfig) (*Service, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check after acquiring write lock
	if svc, exists := m.services[key]; exists {
		return svc, nil
	}

	svc, err := NewService(cfg, m.baseURL, m.jwtSecret, m.metrics)
	if err != nil {
		log.Warn().Err(err).Str("key", key).Msg("Failed to create tenant storage service")
		return nil, err
	}

	m.services[key] = svc

	log.Info().
		Str("provider", cfg.Provider).
		Str("key", key).
		Msg("Created tenant-specific storage service")

	return svc, nil
}

// configKey generates a cache key from storage config
func configKey(cfg *config.StorageConfig) string {
	if cfg == nil {
		return ""
	}

	// Use provider + specific fields as key
	// This ensures different configs get different services
	switch cfg.Provider {
	case "s3":
		return "s3:" + cfg.S3Bucket + ":" + cfg.S3Region + ":" + cfg.S3Endpoint
	case "local":
		return "local:" + cfg.LocalPath
	default:
		return cfg.Provider
	}
}

// configEqual checks if two storage configs are equivalent
func configEqual(a, b *config.StorageConfig) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}

	// Compare key fields that affect service creation
	if a.Provider != b.Provider {
		return false
	}

	switch a.Provider {
	case "s3":
		return a.S3Bucket == b.S3Bucket &&
			a.S3Region == b.S3Region &&
			a.S3Endpoint == b.S3Endpoint &&
			a.S3AccessKey == b.S3AccessKey &&
			a.S3SecretKey == b.S3SecretKey
	case "local":
		return a.LocalPath == b.LocalPath
	default:
		return true
	}
}

// RefreshService recreates a service for a specific config key
// This is useful when tenant config changes
func (m *Manager) RefreshService(ctx context.Context, cfg *config.StorageConfig) error {
	key := configKey(cfg)

	m.mu.Lock()
	defer m.mu.Unlock()

	svc, err := NewService(cfg, m.baseURL, m.jwtSecret, m.metrics)
	if err != nil {
		return err
	}

	m.services[key] = svc

	log.Info().
		Str("provider", cfg.Provider).
		Str("key", key).
		Msg("Refreshed tenant storage service")

	return nil
}

// EnsureDefaultBuckets creates default buckets for all cached services
func (m *Manager) EnsureDefaultBuckets(ctx context.Context) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for key, svc := range m.services {
		if err := svc.EnsureDefaultBuckets(ctx); err != nil {
			log.Warn().Err(err).Str("key", key).Msg("Failed to ensure default buckets")
		}
	}
	return nil
}

// ServiceCount returns the number of cached services
func (m *Manager) ServiceCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.services)
}
