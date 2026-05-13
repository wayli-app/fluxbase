package config

import (
	"fmt"

	"github.com/rs/zerolog/log"
)

// APIConfig contains REST API settings
type APIConfig struct {
	MaxPageSize     int `mapstructure:"max_page_size"`     // Max rows per request (-1 = unlimited)
	MaxTotalResults int `mapstructure:"max_total_results"` // Max total retrievable rows via offset+limit (-1 = unlimited)
	DefaultPageSize int `mapstructure:"default_page_size"` // Auto-applied when no limit specified (-1 = no default)
	MaxBatchSize    int `mapstructure:"max_batch_size"`    // Max records in batch insert/update (-1 = unlimited, default: 1000)
}

// Validate validates API configuration
func (ac *APIConfig) Validate() error {
	// Validate max_page_size (-1 is allowed for unlimited)
	if ac.MaxPageSize == 0 || ac.MaxPageSize < -1 {
		return fmt.Errorf("max_page_size must be positive or -1 for unlimited, got: %d", ac.MaxPageSize)
	}

	// Validate max_total_results (-1 is allowed for unlimited)
	if ac.MaxTotalResults == 0 || ac.MaxTotalResults < -1 {
		return fmt.Errorf("max_total_results must be positive or -1 for unlimited, got: %d", ac.MaxTotalResults)
	}

	// Validate default_page_size (-1 is allowed for no default)
	if ac.DefaultPageSize == 0 || ac.DefaultPageSize < -1 {
		return fmt.Errorf("default_page_size must be positive or -1 for no default, got: %d", ac.DefaultPageSize)
	}

	// Validate that default_page_size doesn't exceed max_page_size (unless either is -1)
	if ac.DefaultPageSize > 0 && ac.MaxPageSize > 0 && ac.DefaultPageSize > ac.MaxPageSize {
		return fmt.Errorf("default_page_size (%d) cannot exceed max_page_size (%d)", ac.DefaultPageSize, ac.MaxPageSize)
	}

	// Warn if limits are disabled
	if ac.MaxPageSize == -1 {
		log.Warn().Msg("max_page_size is set to -1 (unlimited) - this may allow expensive queries")
	}
	if ac.MaxTotalResults == -1 {
		log.Warn().Msg("max_total_results is set to -1 (unlimited) - this may allow deep pagination attacks")
	}
	if ac.DefaultPageSize == -1 {
		log.Warn().Msg("default_page_size is set to -1 (no default) - queries without limit parameter will return all rows")
	}

	return nil
}
