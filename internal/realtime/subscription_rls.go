package realtime

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// Default RLS cache settings (used when no config provided)
const (
	DefaultRLSCacheTTL     = 30 * time.Second // 30 seconds default
	DefaultRLSCacheMaxSize = 100000           // 100K entries default
)

// RLSCacheConfig holds configuration for the RLS cache
type RLSCacheConfig struct {
	MaxSize int           // Maximum number of entries (0 = use default)
	TTL     time.Duration // Cache entry TTL (0 = use default)
}

// rlsCacheEntry represents a cached RLS check result
type rlsCacheEntry struct {
	allowed   bool
	expiresAt time.Time
}

// rlsCache provides a simple time-based cache for RLS check results
type rlsCache struct {
	mu      sync.RWMutex
	entries map[string]*rlsCacheEntry
	maxSize int
	ttl     time.Duration
	cancel  context.CancelFunc
}

// newRLSCache creates a new RLS cache with default settings
func newRLSCache() *rlsCache {
	return newRLSCacheWithConfig(RLSCacheConfig{})
}

// newRLSCacheWithConfig creates a new RLS cache with custom configuration
func newRLSCacheWithConfig(config RLSCacheConfig) *rlsCache {
	maxSize := config.MaxSize
	if maxSize <= 0 {
		maxSize = DefaultRLSCacheMaxSize
	}

	ttl := config.TTL
	if ttl <= 0 {
		ttl = DefaultRLSCacheTTL
	}

	cache := &rlsCache{
		entries: make(map[string]*rlsCacheEntry),
		maxSize: maxSize,
		ttl:     ttl,
	}

	ctx, cancel := context.WithCancel(context.Background())
	cache.cancel = cancel
	go cache.cleanup(ctx)

	return cache
}

// generateCacheKey creates a unique cache key for an RLS check
func (c *rlsCache) generateCacheKey(schema, table, role string, recordID interface{}, claims map[string]interface{}) string {
	// Create a deterministic key from all parameters
	data := fmt.Sprintf("%s:%s:%s:%v", schema, table, role, recordID)
	// Include a hash of the claims to handle custom claims
	if claims != nil {
		claimsJSON, _ := json.Marshal(claims)
		hash := sha256.Sum256(claimsJSON)
		data += ":" + hex.EncodeToString(hash[:8]) // Use first 8 bytes of hash for brevity
	}
	return data
}

// get retrieves a cached result, returning (allowed, found)
func (c *rlsCache) get(key string) (bool, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.entries[key]
	if !exists {
		return false, false
	}

	if time.Now().After(entry.expiresAt) {
		return false, false // Entry expired
	}

	return entry.allowed, true
}

// set stores a result in the cache
func (c *rlsCache) set(key string, allowed bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Evict old entries if cache is too large
	if len(c.entries) >= c.maxSize {
		c.evictExpiredLocked()
	}

	c.entries[key] = &rlsCacheEntry{
		allowed:   allowed,
		expiresAt: time.Now().Add(c.ttl),
	}
}

// evictExpiredLocked removes expired entries (must be called with lock held)
func (c *rlsCache) evictExpiredLocked() {
	now := time.Now()
	for key, entry := range c.entries {
		if now.After(entry.expiresAt) {
			delete(c.entries, key)
		}
	}
}

// cleanup periodically removes expired entries
func (c *rlsCache) cleanup(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.mu.Lock()
			c.evictExpiredLocked()
			c.mu.Unlock()
		}
	}
}

func (c *rlsCache) stop() {
	if c.cancel != nil {
		c.cancel()
	}
}
