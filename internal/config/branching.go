package config

import (
	"fmt"
	"time"
)

// BranchingConfig contains database branching settings for isolated development/testing environments
type BranchingConfig struct {
	Enabled              bool          `mapstructure:"enabled"`                 // Enable database branching feature
	MaxTotalBranches     int           `mapstructure:"max_total_branches"`      // Maximum total branches across all users (default: 50)
	DefaultDataCloneMode string        `mapstructure:"default_data_clone_mode"` // Default data clone mode: schema_only, full_clone, seed_data
	AutoDeleteAfter      time.Duration `mapstructure:"auto_delete_after"`       // Auto-delete preview branches after this duration (0 = never)
	DatabasePrefix       string        `mapstructure:"database_prefix"`         // Prefix for branch database names (default: "branch_")
	SeedsPath            string        `mapstructure:"seeds_path"`              // Path to seed data files directory (default: "./seeds")
}

// DataCloneModes are the valid values for DefaultDataCloneMode
const (
	DataCloneModeSchemaOnly = "schema_only" // Clone schema only (tables, indexes, etc.)
	DataCloneModeFullClone  = "full_clone"  // Clone schema and all data
	DataCloneModeSeedData   = "seed_data"   // Clone schema and run seed data scripts
)

// Validate validates branching configuration
func (bc *BranchingConfig) Validate() error {
	if !bc.Enabled {
		return nil // No validation needed if disabled
	}

	if bc.MaxTotalBranches < 0 {
		return fmt.Errorf("branching max_total_branches cannot be negative, got: %d", bc.MaxTotalBranches)
	}

	// Validate data clone mode
	switch bc.DefaultDataCloneMode {
	case DataCloneModeSchemaOnly, DataCloneModeFullClone, DataCloneModeSeedData, "":
		// Valid modes (empty defaults to schema_only)
	default:
		return fmt.Errorf("branching default_data_clone_mode must be one of: %s, %s, %s, got: %s",
			DataCloneModeSchemaOnly, DataCloneModeFullClone, DataCloneModeSeedData, bc.DefaultDataCloneMode)
	}

	if bc.AutoDeleteAfter < 0 {
		return fmt.Errorf("branching auto_delete_after cannot be negative, got: %v", bc.AutoDeleteAfter)
	}

	if bc.DatabasePrefix == "" {
		return fmt.Errorf("branching database_prefix cannot be empty when enabled")
	}

	// Set default seeds path if not specified
	if bc.SeedsPath == "" {
		bc.SeedsPath = "./seeds"
	}

	return nil
}
