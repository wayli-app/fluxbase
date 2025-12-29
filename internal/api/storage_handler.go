package api

import (
	"sync"
	"time"

	"github.com/fluxbase-eu/fluxbase/internal/config"
	"github.com/fluxbase-eu/fluxbase/internal/database"
	"github.com/fluxbase-eu/fluxbase/internal/storage"
	"github.com/gofiber/fiber/v2"
	"golang.org/x/time/rate"
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
	storage         *storage.Service
	db              *database.Connection
	transformer     *storage.ImageTransformer
	transformConfig *config.TransformConfig
	transformCache  *storage.TransformCache

	// Rate limiting for transforms
	transformLimiters   map[string]*rate.Limiter
	transformLimitersMu sync.Mutex
	transformRateLimit  rate.Limit
	transformBurst      int

	// Concurrency limiting for transforms
	transformSem chan struct{}
}

// NewStorageHandler creates a new storage handler
func NewStorageHandler(storageSvc *storage.Service, db *database.Connection, transformCfg *config.TransformConfig) *StorageHandler {
	return NewStorageHandlerWithCache(storageSvc, db, transformCfg, nil)
}

// NewStorageHandlerWithCache creates a new storage handler with optional transform cache
func NewStorageHandlerWithCache(storageSvc *storage.Service, db *database.Connection, transformCfg *config.TransformConfig, cache *storage.TransformCache) *StorageHandler {
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
		storage:            storageSvc,
		db:                 db,
		transformer:        transformer,
		transformConfig:    transformCfg,
		transformCache:     cache,
		transformLimiters:  make(map[string]*rate.Limiter),
		transformRateLimit: rateLimit,
		transformBurst:     burst,
		transformSem:       transformSem,
	}
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
		<-h.transformSem
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
func (h *StorageHandler) GetTransformConfig(c *fiber.Ctx) error {
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
