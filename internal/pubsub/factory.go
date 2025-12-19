package pubsub

import (
	"fmt"

	"github.com/fluxbase-eu/fluxbase/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

// NewPubSub creates a pub/sub based on the scaling configuration.
//
// Backend options:
// - "local": Local in-process pub/sub (default for single instance)
// - "postgres": PostgreSQL LISTEN/NOTIFY (for multi-instance without Redis)
// - "redis": Redis pub/sub (Dragonfly recommended for high scale)
//
// The pool parameter is required for "postgres" backend.
// The redisURL is required for "redis" backend (from config.Scaling.RedisURL).
func NewPubSub(cfg *config.ScalingConfig, pool *pgxpool.Pool) (PubSub, error) {
	switch cfg.Backend {
	case "local", "":
		log.Info().Msg("Using local pub/sub (single instance mode)")
		return NewLocalPubSub(), nil

	case "postgres":
		if pool == nil {
			return nil, fmt.Errorf("database pool is required for postgres pub/sub backend")
		}
		log.Info().Msg("Using PostgreSQL pub/sub (multi-instance mode)")
		ps := NewPostgresPubSub(pool)
		if err := ps.Start(); err != nil {
			return nil, fmt.Errorf("failed to start PostgreSQL pub/sub: %w", err)
		}
		return ps, nil

	case "redis":
		if cfg.RedisURL == "" {
			return nil, fmt.Errorf("redis_url is required for redis pub/sub backend")
		}
		log.Info().Msg("Using Redis-compatible pub/sub (high-scale mode)")
		ps, err := NewRedisPubSub(cfg.RedisURL)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to Redis for pub/sub: %w", err)
		}
		return ps, nil

	default:
		return nil, fmt.Errorf("unknown pub/sub backend: %s (valid options: local, postgres, redis)", cfg.Backend)
	}
}

// GlobalPubSub is a package-level pub/sub that can be used across the application.
var GlobalPubSub PubSub

// SetGlobalPubSub sets the global pub/sub instance.
func SetGlobalPubSub(ps PubSub) {
	if GlobalPubSub != nil {
		log.Warn().Msg("Replacing existing global pub/sub")
		_ = GlobalPubSub.Close()
	}
	GlobalPubSub = ps
}

// GetGlobalPubSub returns the global pub/sub instance.
// If no pub/sub has been set, it returns a local pub/sub as fallback.
func GetGlobalPubSub() PubSub {
	if GlobalPubSub == nil {
		log.Warn().Msg("Global pub/sub not set, using fallback local pub/sub")
		GlobalPubSub = NewLocalPubSub()
	}
	return GlobalPubSub
}
