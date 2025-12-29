package storage

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// TransformCacheBucket is the internal bucket name for transform cache
const TransformCacheBucket = "_transform_cache"

// cacheEntryMeta stores metadata for a cached transform
type cacheEntryMeta struct {
	ContentType string    `json:"content_type"`
	Size        int64     `json:"size"`
	SourceKey   string    `json:"source_key"`
	AccessTime  time.Time `json:"access_time"`
	CreatedAt   time.Time `json:"created_at"`
}

// cacheEntry represents an in-memory cache entry for LRU tracking
type cacheEntry struct {
	key        string
	size       int64
	accessTime time.Time
}

// TransformCache provides caching for transformed images
// It uses a dedicated bucket in the storage provider and implements LRU eviction
type TransformCache struct {
	provider    Provider
	ttl         time.Duration
	maxSize     int64
	mu          sync.RWMutex
	currentSize int64
	entries     map[string]*cacheEntry // key -> entry for LRU tracking
}

// TransformCacheOptions configures the transform cache
type TransformCacheOptions struct {
	TTL     time.Duration // Cache entry TTL (default: 24 hours)
	MaxSize int64         // Max cache size in bytes (default: 1GB)
}

// NewTransformCache creates a new transform cache using the storage provider
func NewTransformCache(ctx context.Context, provider Provider, opts TransformCacheOptions) (*TransformCache, error) {
	if opts.TTL <= 0 {
		opts.TTL = 24 * time.Hour
	}
	if opts.MaxSize <= 0 {
		opts.MaxSize = 1024 * 1024 * 1024 // 1GB default
	}

	cache := &TransformCache{
		provider: provider,
		ttl:      opts.TTL,
		maxSize:  opts.MaxSize,
		entries:  make(map[string]*cacheEntry),
	}

	// Ensure cache bucket exists
	exists, err := provider.BucketExists(ctx, TransformCacheBucket)
	if err != nil {
		return nil, fmt.Errorf("failed to check cache bucket: %w", err)
	}

	if !exists {
		if err := provider.CreateBucket(ctx, TransformCacheBucket); err != nil {
			return nil, fmt.Errorf("failed to create cache bucket: %w", err)
		}
		log.Info().Str("bucket", TransformCacheBucket).Msg("Transform cache bucket created")
	}

	// Load existing cache entries
	if err := cache.loadExistingEntries(ctx); err != nil {
		log.Warn().Err(err).Msg("Failed to load existing cache entries")
	}

	return cache, nil
}

// loadExistingEntries scans the cache bucket and populates the entries map
func (c *TransformCache) loadExistingEntries(ctx context.Context) error {
	result, err := c.provider.List(ctx, TransformCacheBucket, &ListOptions{MaxKeys: 10000})
	if err != nil {
		return err
	}

	for _, obj := range result.Objects {
		// Skip metadata files
		if len(obj.Key) > 5 && obj.Key[len(obj.Key)-5:] == ".meta" {
			continue
		}

		c.entries[obj.Key] = &cacheEntry{
			key:        obj.Key,
			size:       obj.Size,
			accessTime: obj.LastModified,
		}
		c.currentSize += obj.Size
	}

	log.Info().
		Int64("size", c.currentSize).
		Int("entries", len(c.entries)).
		Msg("Transform cache loaded")

	return nil
}

// cacheKey generates a cache key from bucket, key, and transform options
func (c *TransformCache) cacheKey(bucket, key string, opts *TransformOptions) string {
	data := fmt.Sprintf("%s/%s:%d:%d:%s:%d:%s",
		bucket, key, opts.Width, opts.Height, opts.Format, opts.Quality, opts.Fit)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// Get retrieves a cached transform if it exists and is not expired
func (c *TransformCache) Get(ctx context.Context, bucket, key string, opts *TransformOptions) ([]byte, string, bool) {
	cacheKey := c.cacheKey(bucket, key, opts)

	c.mu.RLock()
	entry, exists := c.entries[cacheKey]
	c.mu.RUnlock()

	if !exists {
		return nil, "", false
	}

	// Check TTL expiration
	if time.Since(entry.accessTime) > c.ttl {
		c.evictEntry(ctx, cacheKey)
		return nil, "", false
	}

	// Download cached data
	reader, obj, err := c.provider.Download(ctx, TransformCacheBucket, cacheKey, nil)
	if err != nil {
		c.evictEntry(ctx, cacheKey)
		return nil, "", false
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, "", false
	}

	// Load metadata
	contentType := obj.ContentType
	metaReader, _, err := c.provider.Download(ctx, TransformCacheBucket, cacheKey+".meta", nil)
	if err == nil {
		defer metaReader.Close()
		var meta cacheEntryMeta
		if err := json.NewDecoder(metaReader).Decode(&meta); err == nil {
			contentType = meta.ContentType
		}
	}

	// Update access time for LRU tracking
	c.mu.Lock()
	if e, ok := c.entries[cacheKey]; ok {
		e.accessTime = time.Now()
	}
	c.mu.Unlock()

	return data, contentType, true
}

// Set stores a transformed image in the cache
func (c *TransformCache) Set(ctx context.Context, bucket, key string, opts *TransformOptions, data []byte, contentType string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	newSize := int64(len(data))

	// Check if we need to evict (target 80% capacity after eviction)
	if c.currentSize+newSize > c.maxSize {
		c.evictUntilSize(ctx, int64(float64(c.maxSize)*0.8)-newSize)
	}

	cacheKey := c.cacheKey(bucket, key, opts)

	// Upload the cached data
	_, err := c.provider.Upload(ctx, TransformCacheBucket, cacheKey, bytes.NewReader(data), newSize, &UploadOptions{
		ContentType: contentType,
	})
	if err != nil {
		return fmt.Errorf("failed to cache transform: %w", err)
	}

	// Upload metadata
	meta := cacheEntryMeta{
		ContentType: contentType,
		Size:        newSize,
		SourceKey:   fmt.Sprintf("%s/%s", bucket, key),
		AccessTime:  time.Now(),
		CreatedAt:   time.Now(),
	}
	metaData, _ := json.Marshal(meta)
	_, err = c.provider.Upload(ctx, TransformCacheBucket, cacheKey+".meta", bytes.NewReader(metaData), int64(len(metaData)), &UploadOptions{
		ContentType: "application/json",
	})
	if err != nil {
		// Clean up the cached data if metadata upload fails
		_ = c.provider.Delete(ctx, TransformCacheBucket, cacheKey)
		return fmt.Errorf("failed to cache transform metadata: %w", err)
	}

	c.entries[cacheKey] = &cacheEntry{
		key:        cacheKey,
		size:       newSize,
		accessTime: time.Now(),
	}
	c.currentSize += newSize

	return nil
}

// evictUntilSize removes oldest entries until currentSize <= targetSize
// Must be called with c.mu held
func (c *TransformCache) evictUntilSize(ctx context.Context, targetSize int64) {
	if targetSize < 0 {
		targetSize = 0
	}

	// Sort entries by access time (oldest first)
	var sortedEntries []*cacheEntry
	for _, entry := range c.entries {
		sortedEntries = append(sortedEntries, entry)
	}
	sort.Slice(sortedEntries, func(i, j int) bool {
		return sortedEntries[i].accessTime.Before(sortedEntries[j].accessTime)
	})

	// Evict oldest entries until we're under target
	evicted := 0
	for _, entry := range sortedEntries {
		if c.currentSize <= targetSize {
			break
		}
		c.evictEntryUnlocked(ctx, entry.key)
		evicted++
	}

	if evicted > 0 {
		log.Debug().
			Int("evicted", evicted).
			Int64("current_size", c.currentSize).
			Msg("Cache eviction completed")
	}
}

// evictEntry removes a cache entry
func (c *TransformCache) evictEntry(ctx context.Context, cacheKey string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.evictEntryUnlocked(ctx, cacheKey)
}

// evictEntryUnlocked removes a cache entry (must hold c.mu)
func (c *TransformCache) evictEntryUnlocked(ctx context.Context, cacheKey string) {
	entry, exists := c.entries[cacheKey]
	if !exists {
		return
	}

	// Delete from storage
	_ = c.provider.Delete(ctx, TransformCacheBucket, cacheKey)
	_ = c.provider.Delete(ctx, TransformCacheBucket, cacheKey+".meta")

	c.currentSize -= entry.size
	delete(c.entries, cacheKey)
}

// Invalidate removes all cached transforms for a source file
// This is called when the source file is updated or deleted
func (c *TransformCache) Invalidate(ctx context.Context, bucket, key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	sourceKey := fmt.Sprintf("%s/%s", bucket, key)

	// Find and remove all entries for this source
	// We need to check metadata to find related entries
	var keysToEvict []string

	// List all cache entries and check their metadata
	result, err := c.provider.List(ctx, TransformCacheBucket, &ListOptions{MaxKeys: 10000})
	if err != nil {
		return err
	}

	for _, obj := range result.Objects {
		// Only check .meta files
		if len(obj.Key) < 5 || obj.Key[len(obj.Key)-5:] != ".meta" {
			continue
		}

		// Download and check metadata
		reader, _, err := c.provider.Download(ctx, TransformCacheBucket, obj.Key, nil)
		if err != nil {
			continue
		}

		var meta cacheEntryMeta
		if err := json.NewDecoder(reader).Decode(&meta); err != nil {
			reader.Close()
			continue
		}
		reader.Close()

		if meta.SourceKey == sourceKey {
			// Remove .meta suffix to get the cache key
			cacheKey := obj.Key[:len(obj.Key)-5]
			keysToEvict = append(keysToEvict, cacheKey)
		}
	}

	// Evict all related entries
	for _, cacheKey := range keysToEvict {
		c.evictEntryUnlocked(ctx, cacheKey)
	}

	if len(keysToEvict) > 0 {
		log.Debug().
			Str("source", sourceKey).
			Int("evicted", len(keysToEvict)).
			Msg("Invalidated cache entries for source file")
	}

	return nil
}

// Cleanup removes expired entries (call periodically)
func (c *TransformCache) Cleanup(ctx context.Context) {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	var keysToEvict []string

	for key, entry := range c.entries {
		if now.Sub(entry.accessTime) > c.ttl {
			keysToEvict = append(keysToEvict, key)
		}
	}

	for _, key := range keysToEvict {
		c.evictEntryUnlocked(ctx, key)
	}

	if len(keysToEvict) > 0 {
		log.Debug().
			Int("evicted", len(keysToEvict)).
			Msg("Cleanup removed expired cache entries")
	}
}

// Stats returns cache statistics
func (c *TransformCache) Stats() (currentSize int64, entryCount int, maxSize int64) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.currentSize, len(c.entries), c.maxSize
}

// Clear removes all cache entries
func (c *TransformCache) Clear(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Delete all entries
	for key := range c.entries {
		_ = c.provider.Delete(ctx, TransformCacheBucket, key)
		_ = c.provider.Delete(ctx, TransformCacheBucket, key+".meta")
	}

	c.entries = make(map[string]*cacheEntry)
	c.currentSize = 0

	log.Info().Msg("Transform cache cleared")
	return nil
}
