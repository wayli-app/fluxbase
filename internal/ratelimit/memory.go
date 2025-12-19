package ratelimit

import (
	"context"
	"encoding/binary"
	"sync"
	"time"
)

// MemoryStore implements Store using in-memory storage.
// This is the default store for single-instance deployments.
// It provides the fastest performance but doesn't share state across instances.
type MemoryStore struct {
	data       map[string]*entry
	mu         sync.RWMutex
	gcInterval time.Duration
	stopCh     chan struct{}
}

type entry struct {
	count     int64
	expiresAt time.Time
}

// NewMemoryStore creates a new in-memory rate limit store.
// gcInterval specifies how often to clean up expired entries.
func NewMemoryStore(gcInterval time.Duration) *MemoryStore {
	if gcInterval <= 0 {
		gcInterval = 10 * time.Minute
	}

	store := &MemoryStore{
		data:       make(map[string]*entry),
		gcInterval: gcInterval,
		stopCh:     make(chan struct{}),
	}

	// Start garbage collection goroutine
	go store.gc()

	return store
}

// Get retrieves the current count for a key.
func (s *MemoryStore) Get(ctx context.Context, key string) (int64, time.Time, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	e, exists := s.data[key]
	if !exists {
		return 0, time.Time{}, nil
	}

	// Check if expired
	if time.Now().After(e.expiresAt) {
		return 0, time.Time{}, nil
	}

	return e.count, e.expiresAt, nil
}

// Increment atomically increments the counter for a key.
func (s *MemoryStore) Increment(ctx context.Context, key string, expiration time.Duration) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	e, exists := s.data[key]

	if !exists || now.After(e.expiresAt) {
		// Create new entry or reset expired one
		s.data[key] = &entry{
			count:     1,
			expiresAt: now.Add(expiration),
		}
		return 1, nil
	}

	// Increment existing entry
	e.count++
	return e.count, nil
}

// Reset resets the counter for a key.
func (s *MemoryStore) Reset(ctx context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.data, key)
	return nil
}

// Close stops the garbage collection goroutine.
func (s *MemoryStore) Close() error {
	close(s.stopCh)
	return nil
}

// gc periodically removes expired entries.
func (s *MemoryStore) gc() {
	ticker := time.NewTicker(s.gcInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.cleanup()
		}
	}
}

// cleanup removes all expired entries.
func (s *MemoryStore) cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for key, e := range s.data {
		if now.After(e.expiresAt) {
			delete(s.data, key)
		}
	}
}

// encodeCount converts an int64 to bytes for storage
func encodeCount(count int64) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(count))
	return buf
}

// decodeCount converts bytes back to int64
func decodeCount(data []byte) int64 {
	if len(data) != 8 {
		return 0
	}
	return int64(binary.BigEndian.Uint64(data))
}
