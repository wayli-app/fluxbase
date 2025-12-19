package ratelimit

import (
	"context"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

// RedisStore implements Store using Redis (or Redis-compatible backends like Dragonfly).
// This is the recommended store for high-scale deployments.
//
// Supported backends (all use the same go-redis library):
// - Dragonfly (recommended): 25x faster than Redis, 80% less memory
// - Redis: The original Redis server
// - Valkey: Redis fork by Linux Foundation
// - KeyDB: Multi-threaded Redis fork
//
// Performance characteristics:
// - In-memory operations are 10-100x faster than PostgreSQL
// - No WAL writes means no disk I/O for rate limiting
// - Handles 100,000+ requests/second per instance
// - Perfect for rate limiting, pub/sub, and ephemeral data
type RedisStore struct {
	client *redis.Client
}

// NewRedisStore creates a new Redis-backed rate limit store.
// url should be in the format: redis://[password@]host:port[/db]
// Examples:
//   - redis://localhost:6379
//   - redis://password@dragonfly:6379
//   - redis://:password@redis:6379/1
func NewRedisStore(url string) (*RedisStore, error) {
	opts, err := redis.ParseURL(url)
	if err != nil {
		return nil, err
	}

	client := redis.NewClient(opts)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	log.Info().Str("addr", opts.Addr).Msg("Connected to Redis-compatible backend for rate limiting")

	return &RedisStore{
		client: client,
	}, nil
}

// Get retrieves the current count for a key.
func (s *RedisStore) Get(ctx context.Context, key string) (int64, time.Time, error) {
	prefixedKey := "ratelimit:" + key

	pipe := s.client.Pipeline()
	getCmd := pipe.Get(ctx, prefixedKey)
	ttlCmd := pipe.TTL(ctx, prefixedKey)

	_, err := pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		return 0, time.Time{}, err
	}

	countStr, err := getCmd.Result()
	if err == redis.Nil {
		return 0, time.Time{}, nil
	}
	if err != nil {
		return 0, time.Time{}, err
	}

	count, err := strconv.ParseInt(countStr, 10, 64)
	if err != nil {
		return 0, time.Time{}, err
	}

	ttl, _ := ttlCmd.Result()
	expiresAt := time.Now().Add(ttl)

	return count, expiresAt, nil
}

// Increment atomically increments the counter for a key.
// Uses INCR + EXPIRE for atomic increment with TTL.
func (s *RedisStore) Increment(ctx context.Context, key string, expiration time.Duration) (int64, error) {
	prefixedKey := "ratelimit:" + key

	// Use a Lua script for atomic increment with conditional expiration
	// This ensures the expiration is only set on the first increment
	script := redis.NewScript(`
		local current = redis.call('INCR', KEYS[1])
		if current == 1 then
			redis.call('PEXPIRE', KEYS[1], ARGV[1])
		end
		return current
	`)

	result, err := script.Run(ctx, s.client, []string{prefixedKey}, expiration.Milliseconds()).Int64()
	if err != nil {
		log.Error().Err(err).Str("key", key).Msg("Failed to increment rate limit counter in Redis")
		return 0, err
	}

	return result, nil
}

// Reset resets the counter for a key.
func (s *RedisStore) Reset(ctx context.Context, key string) error {
	prefixedKey := "ratelimit:" + key
	return s.client.Del(ctx, prefixedKey).Err()
}

// Close closes the Redis client connection.
func (s *RedisStore) Close() error {
	return s.client.Close()
}

// Client returns the underlying Redis client for advanced use cases.
func (s *RedisStore) Client() *redis.Client {
	return s.client
}
