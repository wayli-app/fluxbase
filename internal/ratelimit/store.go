// Package ratelimit provides pluggable rate limiting backends for distributed deployments.
package ratelimit

import (
	"context"
	"time"
)

// Store is the interface for rate limit storage backends.
// It supports different backends for different deployment scenarios:
// - Memory: Single instance deployments (fastest, no external dependencies)
// - PostgreSQL: Multi-instance deployments without additional infrastructure
// - Redis: High-scale deployments (works with Dragonfly, Redis, Valkey, KeyDB)
type Store interface {
	// Get retrieves the current count for a key.
	// Returns the count and expiration time.
	Get(ctx context.Context, key string) (int64, time.Time, error)

	// Increment atomically increments the counter for a key.
	// If the key doesn't exist, it creates it with count=1 and the given expiration.
	// Returns the new count after incrementing.
	Increment(ctx context.Context, key string, expiration time.Duration) (int64, error)

	// Reset resets the counter for a key.
	Reset(ctx context.Context, key string) error

	// Close closes the store and releases resources.
	Close() error
}

// Result contains the rate limit check result
type Result struct {
	// Allowed indicates whether the request is allowed
	Allowed bool

	// Remaining is the number of requests remaining in the current window
	Remaining int64

	// ResetAt is when the rate limit window resets
	ResetAt time.Time

	// Limit is the maximum number of requests allowed in the window
	Limit int64
}

// Check performs a rate limit check using the store.
// It increments the counter and returns whether the request is allowed.
func Check(ctx context.Context, store Store, key string, limit int64, window time.Duration) (*Result, error) {
	count, err := store.Increment(ctx, key, window)
	if err != nil {
		return nil, err
	}

	result := &Result{
		Allowed:   count <= limit,
		Remaining: limit - count,
		Limit:     limit,
		ResetAt:   time.Now().Add(window),
	}

	if result.Remaining < 0 {
		result.Remaining = 0
	}

	return result, nil
}
