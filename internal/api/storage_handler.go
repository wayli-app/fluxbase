package api

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
	"golang.org/x/time/rate"

	"github.com/nimbleflux/fluxbase/internal/config"
	"github.com/nimbleflux/fluxbase/internal/database"
	"github.com/nimbleflux/fluxbase/internal/middleware"
	"github.com/nimbleflux/fluxbase/internal/storage"
)

// StorageHandler handles file storage operations
// Methods are split across multiple files:
// - storage_files.go: UploadFile, DownloadFile, DeleteFile, GetFileInfo, ListFiles
// - storage_buckets.go: CreateBucket, UpdateBucketSettings, DeleteBucket, ListBuckets
// - storage_signed.go: GenerateSignedURL, DownloadSignedObject
// - storage_multipart.go: MultipartUpload
// - storage_sharing.go: ShareObject, RevokeShare, ListShares
// - storage_utils.go: helper functions (detectContentType, parseMetadata, getUserID, setRLSContext)
type StorageHandler struct {
	storageManager  *storage.Manager
	baseConfig      *config.Config
	db              *database.Connection
	transformer     *storage.ImageTransformer
	transformConfig *config.TransformConfig
	transformCache  *storage.TransformCache

	transformLimiters   map[string]*rate.Limiter
	transformLimitersMu sync.Mutex
	transformRateLimit  rate.Limit
	transformBurst      int

	transformSem       chan struct{}
	signedURLLimiter   *ipRateLimiter
}

// NewStorageHandler creates a new storage handler with automatic cache initialization
func NewStorageHandler(storageMgr *storage.Manager, db *database.Connection, baseConfig *config.Config, transformCfg *config.TransformConfig) *StorageHandler {
	var cache *storage.TransformCache

	// Get base service for cache initialization
	storageSvc := storageMgr.GetBaseService()

	// Initialize transform cache if transforms are enabled
	if transformCfg != nil && transformCfg.Enabled && storageSvc != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		cacheOpts := storage.TransformCacheOptions{
			TTL:     transformCfg.CacheTTL,
			MaxSize: transformCfg.CacheMaxSize,
		}
		// Use defaults if not configured
		if cacheOpts.TTL <= 0 {
			cacheOpts.TTL = 24 * time.Hour
		}
		if cacheOpts.MaxSize <= 0 {
			cacheOpts.MaxSize = 1024 * 1024 * 1024 // 1GB
		}

		var err error
		cache, err = storage.NewTransformCache(ctx, storageSvc.Provider, cacheOpts)
		if err != nil {
			// Log error but don't fail - transforms will work without caching
			log.Warn().Err(err).Msg("Failed to initialize transform cache, transforms will not be cached")
		} else {
			log.Info().Msg("Transform cache initialized")
		}
	}

	return NewStorageHandlerWithCache(storageMgr, db, baseConfig, transformCfg, cache)
}

// NewStorageHandlerWithCache creates a new storage handler with optional transform cache
func NewStorageHandlerWithCache(storageMgr *storage.Manager, db *database.Connection, baseConfig *config.Config, transformCfg *config.TransformConfig, cache *storage.TransformCache) *StorageHandler {
	var transformer *storage.ImageTransformer
	var transformSem chan struct{}
	var rateLimit rate.Limit
	var burst int

	if transformCfg != nil && transformCfg.Enabled {
		transformer = storage.NewImageTransformerWithOptions(storage.TransformerOptions{
			MaxWidth:       transformCfg.MaxWidth,
			MaxHeight:      transformCfg.MaxHeight,
			MaxTotalPixels: transformCfg.MaxTotalPixels,
			BucketSize:     transformCfg.BucketSize,
		})

		// Initialize concurrency limiter
		maxConcurrent := transformCfg.MaxConcurrent
		if maxConcurrent <= 0 {
			maxConcurrent = 4
		}
		transformSem = make(chan struct{}, maxConcurrent)

		// Initialize rate limit (transforms per minute per user)
		rateLimitPerMin := transformCfg.RateLimit
		if rateLimitPerMin <= 0 {
			rateLimitPerMin = 60
		}
		rateLimit = rate.Limit(float64(rateLimitPerMin) / 60.0) // Convert to per-second
		burst = rateLimitPerMin / 10                            // Allow burst of 10% of per-minute limit
		if burst < 1 {
			burst = 1
		}
	}

	return &StorageHandler{
		storageManager:     storageMgr,
		baseConfig:         baseConfig,
		db:                 db,
		transformer:        transformer,
		transformConfig:    transformCfg,
		transformCache:     cache,
		transformLimiters:  make(map[string]*rate.Limiter),
		transformRateLimit: rateLimit,
		transformBurst:     burst,
		transformSem:       transformSem,
		signedURLLimiter: &ipRateLimiter{
			requests: make(map[string]*rateLimitEntry),
			limit:    100,
			window:   time.Minute,
		},
	}
}

// getService returns the storage service for the current request context.
// It uses the tenant-specific configuration if available, otherwise returns the base service.
func (h *StorageHandler) getService(c fiber.Ctx) (*storage.Service, error) {
	if h.storageManager == nil {
		return nil, fmt.Errorf("storage manager not initialized")
	}
	cfg := GetStorageConfig(c, h.baseConfig)
	return h.storageManager.GetService(cfg)
}

// getPool returns the database pool for storage operations.
// When a tenant pool is available (non-default tenant with a separate database),
// it routes through the tenant pool which uses FDW to access storage tables
// in the main database. For the default tenant, it falls back to the main pool.
func (h *StorageHandler) getPool(c fiber.Ctx) *pgxpool.Pool {
	if tenantPool := middleware.GetTenantPool(c); tenantPool != nil {
		return tenantPool
	}
	return h.db.Pool()
}

// getTransformLimiter returns the rate limiter for a given key (IP:userID)
func (h *StorageHandler) getTransformLimiter(key string) *rate.Limiter {
	h.transformLimitersMu.Lock()
	defer h.transformLimitersMu.Unlock()

	limiter, exists := h.transformLimiters[key]
	if !exists {
		limiter = rate.NewLimiter(h.transformRateLimit, h.transformBurst)
		h.transformLimiters[key] = limiter
	}
	return limiter
}

// acquireTransformSlot attempts to acquire a slot for transform processing
// Returns false if the system is at capacity
func (h *StorageHandler) acquireTransformSlot(timeout time.Duration) bool {
	if h.transformSem == nil {
		return true // No limit configured
	}

	select {
	case h.transformSem <- struct{}{}:
		return true
	case <-time.After(timeout):
		return false
	}
}

// releaseTransformSlot releases a transform slot
func (h *StorageHandler) releaseTransformSlot() {
	if h.transformSem != nil {
		select {
		case <-h.transformSem:
			// Successfully released
		default:
			// No slot to release (already released or never acquired)
			// This prevents blocking if release is called without acquire
		}
	}
}

// TransformConfigResponse represents the response for the transform config endpoint
type TransformConfigResponse struct {
	Enabled        bool     `json:"enabled"`
	DefaultQuality int      `json:"default_quality"`
	MaxWidth       int      `json:"max_width"`
	MaxHeight      int      `json:"max_height"`
	AllowedFormats []string `json:"allowed_formats,omitempty"`
}

// GetTransformConfig returns the image transformation configuration
// This is a public endpoint that returns configuration info for the admin dashboard
func (h *StorageHandler) GetTransformConfig(c fiber.Ctx) error {
	if h.transformConfig == nil {
		return c.JSON(TransformConfigResponse{
			Enabled: false,
		})
	}

	return c.JSON(TransformConfigResponse{
		Enabled:        h.transformConfig.Enabled,
		DefaultQuality: h.transformConfig.DefaultQuality,
		MaxWidth:       h.transformConfig.MaxWidth,
		MaxHeight:      h.transformConfig.MaxHeight,
		AllowedFormats: h.transformConfig.AllowedFormats,
	})
}
