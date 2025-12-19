package ratelimit

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

// PostgresStore implements Store using PostgreSQL.
// This is suitable for multi-instance deployments without requiring Redis.
// It uses UPSERT with ON CONFLICT to atomically increment counters.
//
// Performance characteristics:
// - Slower than Redis but faster than hitting an external service
// - Uses the existing database connection pool
// - Good for deployments up to ~1000 requests/second
// - For higher scale, use RedisStore with Dragonfly
type PostgresStore struct {
	pool *pgxpool.Pool
}

// NewPostgresStore creates a new PostgreSQL-backed rate limit store.
// The store uses the system.rate_limits table which must be created via migration.
func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{
		pool: pool,
	}
}

// Get retrieves the current count for a key.
func (s *PostgresStore) Get(ctx context.Context, key string) (int64, time.Time, error) {
	var count int64
	var expiresAt time.Time

	err := s.pool.QueryRow(ctx, `
		SELECT count, expires_at
		FROM system.rate_limits
		WHERE key = $1 AND expires_at > NOW()
	`, key).Scan(&count, &expiresAt)

	if err != nil {
		// Not found is not an error - return zero count
		return 0, time.Time{}, nil
	}

	return count, expiresAt, nil
}

// Increment atomically increments the counter for a key.
// Uses PostgreSQL's UPSERT to handle concurrent access safely.
func (s *PostgresStore) Increment(ctx context.Context, key string, expiration time.Duration) (int64, error) {
	expiresAt := time.Now().Add(expiration)

	var count int64
	err := s.pool.QueryRow(ctx, `
		INSERT INTO system.rate_limits (key, count, expires_at)
		VALUES ($1, 1, $2)
		ON CONFLICT (key) DO UPDATE SET
			count = CASE
				WHEN system.rate_limits.expires_at <= NOW() THEN 1
				ELSE system.rate_limits.count + 1
			END,
			expires_at = CASE
				WHEN system.rate_limits.expires_at <= NOW() THEN $2
				ELSE system.rate_limits.expires_at
			END
		RETURNING count
	`, key, expiresAt).Scan(&count)

	if err != nil {
		log.Error().Err(err).Str("key", key).Msg("Failed to increment rate limit counter")
		return 0, err
	}

	return count, nil
}

// Reset resets the counter for a key.
func (s *PostgresStore) Reset(ctx context.Context, key string) error {
	_, err := s.pool.Exec(ctx, `
		DELETE FROM system.rate_limits WHERE key = $1
	`, key)
	return err
}

// Close is a no-op for PostgresStore as we don't own the connection pool.
func (s *PostgresStore) Close() error {
	return nil
}

// Cleanup removes expired entries from the rate_limits table.
// This should be called periodically (e.g., by a background job or cron).
func (s *PostgresStore) Cleanup(ctx context.Context) (int64, error) {
	result, err := s.pool.Exec(ctx, `
		DELETE FROM system.rate_limits WHERE expires_at <= NOW()
	`)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected(), nil
}

// EnsureTable creates the rate_limits table if it doesn't exist.
// This is called during startup to ensure the table exists.
// In production, the table should be created via a migration.
func (s *PostgresStore) EnsureTable(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS system.rate_limits (
			key TEXT PRIMARY KEY,
			count BIGINT NOT NULL DEFAULT 1,
			expires_at TIMESTAMPTZ NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);

		CREATE INDEX IF NOT EXISTS idx_rate_limits_expires_at
		ON system.rate_limits (expires_at);
	`)
	return err
}
