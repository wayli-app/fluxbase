package auth

import (
	"sync"
	"time"
)

// nonceEntry stores a nonce with its associated user and expiry time
type nonceEntry struct {
	userID string
	expiry time.Time
}

// NonceStore manages reauthentication nonces for sensitive operations.
// Uses a mutex to protect concurrent access from multiple goroutines.
// Nonces are single-use and expire after a configured TTL.
type NonceStore struct {
	mu     sync.RWMutex
	nonces map[string]nonceEntry
}

// NewNonceStore creates a new nonce store
func NewNonceStore() *NonceStore {
	return &NonceStore{
		nonces: make(map[string]nonceEntry),
	}
}

// Set stores a nonce with its associated user ID and TTL
func (s *NonceStore) Set(nonce, userID string, ttl time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nonces[nonce] = nonceEntry{
		userID: userID,
		expiry: time.Now().Add(ttl),
	}
}

// Validate checks if a nonce is valid for the given user and removes it (single-use).
// Returns true if the nonce exists, belongs to the user, and hasn't expired.
func (s *NonceStore) Validate(nonce, userID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, exists := s.nonces[nonce]
	if !exists {
		return false
	}

	// Always delete the nonce (single-use)
	delete(s.nonces, nonce)

	// Check if it belongs to the correct user and hasn't expired
	return entry.userID == userID && time.Now().Before(entry.expiry)
}

// Cleanup removes expired nonces
func (s *NonceStore) Cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for nonce, entry := range s.nonces {
		if now.After(entry.expiry) {
			delete(s.nonces, nonce)
		}
	}
}

// StartCleanup starts a background goroutine that periodically cleans up expired nonces.
// Returns a stop channel that can be closed to stop the cleanup goroutine.
func (s *NonceStore) StartCleanup(interval time.Duration) chan struct{} {
	stop := make(chan struct{})
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				s.Cleanup()
			case <-stop:
				return
			}
		}
	}()
	return stop
}
