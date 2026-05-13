package config

import (
	"fmt"
	"time"
)

// ServerConfig contains HTTP server settings
type ServerConfig struct {
	Address         string        `mapstructure:"address"`
	ReadTimeout     time.Duration `mapstructure:"read_timeout"`
	WriteTimeout    time.Duration `mapstructure:"write_timeout"`
	IdleTimeout     time.Duration `mapstructure:"idle_timeout"`
	BodyLimit       int           `mapstructure:"body_limit"`
	AllowedIPRanges []string      `mapstructure:"allowed_ip_ranges"` // Global IP CIDR ranges allowed to access server (empty = allow all)
	TrustedProxies  []string      `mapstructure:"trusted_proxies"`   // Trusted proxy IP ranges for X-Forwarded-For header validation (empty = trust none)

	// Per-endpoint body limits (if not specified, uses defaults from middleware)
	BodyLimits BodyLimitsConfig `mapstructure:"body_limits"`
}

// BodyLimitsConfig contains per-endpoint body size limits
type BodyLimitsConfig struct {
	// Enabled controls whether per-endpoint limits are enforced (default: true)
	Enabled bool `mapstructure:"enabled"`
	// DefaultLimit is used when no pattern matches (default: 1MB)
	DefaultLimit int64 `mapstructure:"default_limit"`
	// RESTLimit for REST API CRUD operations (default: 1MB)
	RESTLimit int64 `mapstructure:"rest_limit"`
	// AuthLimit for authentication endpoints (default: 64KB)
	AuthLimit int64 `mapstructure:"auth_limit"`
	// StorageLimit for file uploads (default: 500MB)
	StorageLimit int64 `mapstructure:"storage_limit"`
	// BulkLimit for bulk operations and RPC (default: 10MB)
	BulkLimit int64 `mapstructure:"bulk_limit"`
	// AdminLimit for admin endpoints (default: 5MB)
	AdminLimit int64 `mapstructure:"admin_limit"`
	// MaxJSONDepth limits nesting depth to prevent stack overflow (default: 64)
	MaxJSONDepth int `mapstructure:"max_json_depth"`
}

// Validate validates server configuration
func (sc *ServerConfig) Validate() error {
	if sc.Address == "" {
		return fmt.Errorf("server address cannot be empty")
	}

	// Validate timeouts are positive
	if sc.ReadTimeout <= 0 {
		return fmt.Errorf("read_timeout must be positive, got: %v", sc.ReadTimeout)
	}
	if sc.WriteTimeout <= 0 {
		return fmt.Errorf("write_timeout must be positive, got: %v", sc.WriteTimeout)
	}
	if sc.IdleTimeout <= 0 {
		return fmt.Errorf("idle_timeout must be positive, got: %v", sc.IdleTimeout)
	}

	// Validate body limit
	if sc.BodyLimit <= 0 {
		return fmt.Errorf("body_limit must be positive, got: %d", sc.BodyLimit)
	}

	return nil
}
