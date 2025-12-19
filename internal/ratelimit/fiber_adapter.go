package ratelimit

import (
	"context"
	"encoding/binary"
	"time"
)

// FiberAdapter adapts our Store interface to Fiber's Storage interface.
// This allows using our pluggable rate limit stores with Fiber's limiter middleware.
type FiberAdapter struct {
	store Store
}

// NewFiberAdapter creates a new Fiber storage adapter wrapping our Store.
func NewFiberAdapter(store Store) *FiberAdapter {
	return &FiberAdapter{store: store}
}

// Get retrieves a value from the store.
// The returned byte slice contains the encoded count.
func (a *FiberAdapter) Get(key string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	count, _, err := a.store.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	if count == 0 {
		return nil, nil
	}

	return encodeInt64(count), nil
}

// Set stores a value in the store.
// The value is expected to be an encoded int64 count.
// exp is the expiration time.
func (a *FiberAdapter) Set(key string, val []byte, exp time.Duration) error {
	// The Fiber limiter calls Set with the new count after incrementing
	// Since our Increment already handles this atomically, we can ignore Set
	// This is called by Fiber's limiter but our store handles it differently
	return nil
}

// Delete removes a value from the store.
func (a *FiberAdapter) Delete(key string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return a.store.Reset(ctx, key)
}

// Reset clears all data (not implemented for distributed stores).
func (a *FiberAdapter) Reset() error {
	// Not supported for distributed stores
	return nil
}

// Close releases resources.
func (a *FiberAdapter) Close() error {
	return a.store.Close()
}

// IncrementAdapter is a specialized adapter for Fiber's limiter that provides
// atomic increment functionality.
type IncrementAdapter struct {
	store      Store
	expiration time.Duration
}

// NewIncrementAdapter creates an adapter that provides atomic increment.
func NewIncrementAdapter(store Store, expiration time.Duration) *IncrementAdapter {
	return &IncrementAdapter{
		store:      store,
		expiration: expiration,
	}
}

// Get retrieves and increments the counter for a key.
// Returns the new count as a byte slice.
func (a *IncrementAdapter) Get(key string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	count, err := a.store.Increment(ctx, key, a.expiration)
	if err != nil {
		return nil, err
	}

	return encodeInt64(count), nil
}

// Set is a no-op since Get already increments.
func (a *IncrementAdapter) Set(key string, val []byte, exp time.Duration) error {
	return nil
}

// Delete removes a value from the store.
func (a *IncrementAdapter) Delete(key string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return a.store.Reset(ctx, key)
}

// Reset clears all data (not implemented for distributed stores).
func (a *IncrementAdapter) Reset() error {
	return nil
}

// Close releases resources.
func (a *IncrementAdapter) Close() error {
	return a.store.Close()
}

// encodeInt64 converts an int64 to bytes.
func encodeInt64(n int64) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(n))
	return buf
}

// decodeInt64 converts bytes to int64.
func decodeInt64(data []byte) int64 {
	if len(data) != 8 {
		return 0
	}
	return int64(binary.BigEndian.Uint64(data))
}
