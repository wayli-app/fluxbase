package config

import (
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
)

// JobsConfig contains long-running background jobs settings
type JobsConfig struct {
	Enabled                   bool          `mapstructure:"enabled"`
	JobsDir                   string        `mapstructure:"jobs_dir"`
	AutoLoadOnBoot            bool          `mapstructure:"auto_load_on_boot"`            // Load jobs from filesystem at boot
	WorkerMode                string        `mapstructure:"worker_mode"`                  // "embedded", "standalone", "disabled"
	EmbeddedWorkerCount       int           `mapstructure:"embedded_worker_count"`        // Number of embedded workers
	MaxConcurrentPerWorker    int           `mapstructure:"max_concurrent_per_worker"`    // Max concurrent jobs per worker
	MaxConcurrentPerNamespace int           `mapstructure:"max_concurrent_per_namespace"` // Max concurrent jobs per namespace
	DefaultMaxDuration        time.Duration `mapstructure:"default_max_duration"`         // Default job timeout
	MaxMaxDuration            time.Duration `mapstructure:"max_max_duration"`             // Maximum allowed job timeout
	DefaultProgressTimeout    time.Duration `mapstructure:"default_progress_timeout"`     // Default progress timeout
	PollInterval              time.Duration `mapstructure:"poll_interval"`                // Worker poll interval
	WorkerHeartbeatInterval   time.Duration `mapstructure:"worker_heartbeat_interval"`    // Worker heartbeat interval
	WorkerTimeout             time.Duration `mapstructure:"worker_timeout"`               // Worker considered dead after this
	SyncAllowedIPRanges       []string      `mapstructure:"sync_allowed_ip_ranges"`       // IP CIDR ranges allowed to sync jobs
	GracefulShutdownTimeout   time.Duration `mapstructure:"graceful_shutdown_timeout"`    // Time to wait for running jobs during shutdown (default: 5m)
}

// Validate validates jobs configuration
func (jc *JobsConfig) Validate() error {
	// Validate jobs directory
	if jc.JobsDir == "" {
		return fmt.Errorf("jobs_dir cannot be empty")
	}

	// Validate worker mode
	validModes := []string{"embedded", "standalone", "disabled"}
	modeValid := false
	for _, mode := range validModes {
		if jc.WorkerMode == mode {
			modeValid = true
			break
		}
	}
	if !modeValid {
		return fmt.Errorf("invalid worker_mode: %s (must be one of: %v)", jc.WorkerMode, validModes)
	}

	// Validate worker counts
	if jc.EmbeddedWorkerCount < 0 {
		return fmt.Errorf("embedded_worker_count cannot be negative, got: %d", jc.EmbeddedWorkerCount)
	}
	if jc.MaxConcurrentPerWorker <= 0 {
		return fmt.Errorf("max_concurrent_per_worker must be positive, got: %d", jc.MaxConcurrentPerWorker)
	}
	if jc.MaxConcurrentPerNamespace <= 0 {
		return fmt.Errorf("max_concurrent_per_namespace must be positive, got: %d", jc.MaxConcurrentPerNamespace)
	}

	// Validate timeout settings
	if jc.DefaultMaxDuration <= 0 {
		return fmt.Errorf("default_max_duration must be positive, got: %v", jc.DefaultMaxDuration)
	}
	if jc.MaxMaxDuration <= 0 {
		return fmt.Errorf("max_max_duration must be positive, got: %v", jc.MaxMaxDuration)
	}
	if jc.DefaultMaxDuration > jc.MaxMaxDuration {
		return fmt.Errorf("default_max_duration (%v) cannot be greater than max_max_duration (%v)", jc.DefaultMaxDuration, jc.MaxMaxDuration)
	}
	if jc.DefaultProgressTimeout <= 0 {
		return fmt.Errorf("default_progress_timeout must be positive, got: %v", jc.DefaultProgressTimeout)
	}

	// Validate intervals
	if jc.PollInterval <= 0 {
		return fmt.Errorf("poll_interval must be positive, got: %v", jc.PollInterval)
	}
	if jc.WorkerHeartbeatInterval <= 0 {
		return fmt.Errorf("worker_heartbeat_interval must be positive, got: %v", jc.WorkerHeartbeatInterval)
	}
	if jc.WorkerTimeout <= 0 {
		return fmt.Errorf("worker_timeout must be positive, got: %v", jc.WorkerTimeout)
	}
	// Worker timeout must comfortably exceed the heartbeat interval, otherwise
	// a worker can be reaped between heartbeats. Require at least 2x so a
	// single missed heartbeat never triggers cleanup.
	if jc.WorkerTimeout < 2*jc.WorkerHeartbeatInterval {
		return fmt.Errorf("worker_timeout (%v) must be at least 2x worker_heartbeat_interval (%v)", jc.WorkerTimeout, jc.WorkerHeartbeatInterval)
	}

	// Warn if max_max_duration is very high (over 1 hour)
	if jc.MaxMaxDuration > time.Hour {
		log.Warn().Dur("max_max_duration", jc.MaxMaxDuration).Msg("max_max_duration is over 1 hour - very long-running jobs may impact performance")
	}

	// Warn if worker count is 0 in embedded mode
	if jc.WorkerMode == "embedded" && jc.EmbeddedWorkerCount == 0 {
		log.Warn().Msg("worker_mode is 'embedded' but embedded_worker_count is 0 - no jobs will be processed")
	}

	return nil
}
