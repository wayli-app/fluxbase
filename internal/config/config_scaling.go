package config

import (
	"fmt"

	"github.com/rs/zerolog/log"
)

// ScalingConfig contains horizontal scaling settings for multi-instance deployments
type ScalingConfig struct {
	// WorkerOnly mode disables the API server and only runs job workers
	// Use this for dedicated worker containers that only process background jobs
	WorkerOnly bool `mapstructure:"worker_only"`

	// DisableScheduler prevents cron schedulers from running on this instance
	// Use this when running multiple instances to prevent duplicate scheduled jobs
	// Only one instance should run the scheduler (use leader election or manual config)
	DisableScheduler bool `mapstructure:"disable_scheduler"`

	// DisableRealtime prevents the realtime listener from starting
	// Useful for worker-only instances or when using an external realtime service
	DisableRealtime bool `mapstructure:"disable_realtime"`

	// EnableSchedulerLeaderElection enables automatic leader election for schedulers
	// When enabled, only one instance will run schedulers using PostgreSQL advisory locks
	// This is the recommended setting for multi-instance deployments
	EnableSchedulerLeaderElection bool `mapstructure:"enable_scheduler_leader_election"`

	// Backend for distributed state (rate limiting, pub/sub, sessions)
	// Options: "local" (single instance), "postgres", "redis"
	// "redis" works with Dragonfly (recommended), Redis, Valkey, KeyDB
	Backend string `mapstructure:"backend"`

	// RedisURL is the connection URL for Redis-compatible backends (Dragonfly recommended)
	// Only used when Backend is "redis"
	// Format: redis://[password@]host:port[/db]
	RedisURL string `mapstructure:"redis_url"`
}

// Validate validates scaling configuration
func (sc *ScalingConfig) Validate() error {
	// Validate backend
	validBackends := []string{"local", "postgres", "redis"}
	backendValid := false
	for _, b := range validBackends {
		if sc.Backend == b {
			backendValid = true
			break
		}
	}
	if !backendValid {
		return fmt.Errorf("invalid scaling backend: %s (must be one of: %v)", sc.Backend, validBackends)
	}

	// Validate redis_url is set when backend is redis
	if sc.Backend == "redis" && sc.RedisURL == "" {
		return fmt.Errorf("redis_url is required when scaling backend is 'redis'")
	}

	// Warn about conflicting settings
	if sc.WorkerOnly && !sc.DisableScheduler {
		log.Warn().Msg("Worker-only mode is enabled but scheduler is not disabled - consider setting disable_scheduler=true for worker containers")
	}

	if sc.WorkerOnly && !sc.DisableRealtime {
		log.Warn().Msg("Worker-only mode is enabled but realtime is not disabled - realtime will be skipped in worker-only mode anyway")
	}

	return nil
}
