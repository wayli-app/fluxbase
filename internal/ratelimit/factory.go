package ratelimit

import (
	"fmt"
	"time"

	"github.com/fluxbase-eu/fluxbase/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

// NewStore creates a rate limit store based on the scaling configuration.
//
// Backend options:
// - "local": In-memory store (default for single instance)
// - "postgres": PostgreSQL-backed store (for multi-instance without Redis)
// - "redis": Redis-compatible store (Dragonfly recommended for high scale)
//
// The pool parameter is required for "postgres" backend.
// The redisURL is required for "redis" backend (from config.Scaling.RedisURL).
func NewStore(cfg *config.ScalingConfig, pool *pgxpool.Pool) (Store, error) {
	switch cfg.Backend {
	case "local", "":
		log.Info().Msg("Using in-memory rate limit store (single instance mode)")
		return NewMemoryStore(10 * time.Minute), nil

	case "postgres":
		if pool == nil {
			return nil, fmt.Errorf("database pool is required for postgres rate limit backend")
		}
		log.Info().Msg("Using PostgreSQL rate limit store (multi-instance mode)")
		store := NewPostgresStore(pool)
		return store, nil

	case "redis":
		if cfg.RedisURL == "" {
			return nil, fmt.Errorf("redis_url is required for redis rate limit backend")
		}
		log.Info().Msg("Using Redis-compatible rate limit store (high-scale mode)")
		store, err := NewRedisStore(cfg.RedisURL)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to Redis: %w", err)
		}
		return store, nil

	default:
		return nil, fmt.Errorf("unknown rate limit backend: %s (valid options: local, postgres, redis)", cfg.Backend)
	}
}

// GlobalStore is a package-level store that can be used across the application.
// It is set during server initialization.
var GlobalStore Store

// SetGlobalStore sets the global rate limit store.
// This should be called once during server initialization.
func SetGlobalStore(store Store) {
	if GlobalStore != nil {
		log.Warn().Msg("Replacing existing global rate limit store")
		_ = GlobalStore.Close()
	}
	GlobalStore = store
}

// GetGlobalStore returns the global rate limit store.
// If no store has been set, it returns a memory store as fallback.
func GetGlobalStore() Store {
	if GlobalStore == nil {
		log.Warn().Msg("Global rate limit store not set, using fallback memory store")
		GlobalStore = NewMemoryStore(10 * time.Minute)
	}
	return GlobalStore
}
