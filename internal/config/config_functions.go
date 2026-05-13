package config

import (
	"fmt"

	"github.com/rs/zerolog/log"
)

// FunctionsConfig contains edge functions settings
type FunctionsConfig struct {
	Enabled             bool     `mapstructure:"enabled"`
	FunctionsDir        string   `mapstructure:"functions_dir"`
	AutoLoadOnBoot      bool     `mapstructure:"auto_load_on_boot"`      // Load functions from filesystem at boot
	DefaultTimeout      int      `mapstructure:"default_timeout"`        // seconds
	MaxTimeout          int      `mapstructure:"max_timeout"`            // seconds
	DefaultMemoryLimit  int      `mapstructure:"default_memory_limit"`   // MB
	MaxMemoryLimit      int      `mapstructure:"max_memory_limit"`       // MB
	MaxOutputSize       int      `mapstructure:"max_output_size"`        // Max output size in bytes (0 = unlimited, default: 10MB)
	SyncAllowedIPRanges []string `mapstructure:"sync_allowed_ip_ranges"` // IP CIDR ranges allowed to sync functions
}

// Validate validates functions configuration
func (fc *FunctionsConfig) Validate() error {
	// Validate functions directory
	if fc.FunctionsDir == "" {
		return fmt.Errorf("functions_dir cannot be empty")
	}

	// Validate timeout settings
	if fc.DefaultTimeout <= 0 {
		return fmt.Errorf("default_timeout must be positive, got: %d", fc.DefaultTimeout)
	}
	if fc.MaxTimeout <= 0 {
		return fmt.Errorf("max_timeout must be positive, got: %d", fc.MaxTimeout)
	}
	if fc.DefaultTimeout > fc.MaxTimeout {
		return fmt.Errorf("default_timeout (%d) cannot be greater than max_timeout (%d)", fc.DefaultTimeout, fc.MaxTimeout)
	}

	// Validate memory limit settings
	if fc.DefaultMemoryLimit <= 0 {
		return fmt.Errorf("default_memory_limit must be positive, got: %d", fc.DefaultMemoryLimit)
	}
	if fc.MaxMemoryLimit <= 0 {
		return fmt.Errorf("max_memory_limit must be positive, got: %d", fc.MaxMemoryLimit)
	}
	if fc.DefaultMemoryLimit > fc.MaxMemoryLimit {
		return fmt.Errorf("default_memory_limit (%d) cannot be greater than max_memory_limit (%d)", fc.DefaultMemoryLimit, fc.MaxMemoryLimit)
	}

	// Warn if max_timeout is very high (over 5 minutes)
	if fc.MaxTimeout > 300 {
		log.Warn().Int("max_timeout", fc.MaxTimeout).Msg("max_timeout is over 5 minutes - long-running functions may impact performance")
	}

	// Warn if max_memory_limit is very high (over 1GB)
	if fc.MaxMemoryLimit > 1024 {
		log.Warn().Int("max_memory_limit", fc.MaxMemoryLimit).Msg("max_memory_limit is over 1GB - high memory functions may impact performance")
	}

	return nil
}
