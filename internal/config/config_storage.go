package config

import (
	"fmt"
	"time"
)

// StorageConfig contains file storage settings
type StorageConfig struct {
	Enabled          bool     `mapstructure:"enabled"`  // Enable storage functionality
	Provider         string   `mapstructure:"provider"` // local or s3
	LocalPath        string   `mapstructure:"local_path"`
	S3Endpoint       string   `mapstructure:"s3_endpoint"`
	S3AccessKey      string   `mapstructure:"s3_access_key"`
	S3SecretKey      string   `mapstructure:"s3_secret_key"`
	S3Bucket         string   `mapstructure:"s3_bucket"`
	S3Region         string   `mapstructure:"s3_region"`
	S3ForcePathStyle bool     `mapstructure:"s3_force_path_style"` // Use path-style addressing (required for MinIO, R2, Spaces, etc.)
	DefaultBuckets   []string `mapstructure:"default_buckets"`     // Buckets to auto-create on startup
	MaxUploadSize    int64    `mapstructure:"max_upload_size"`

	// Image transformation settings
	Transforms TransformConfig `mapstructure:"transforms"`
}

// TransformConfig contains image transformation settings
type TransformConfig struct {
	Enabled        bool     `mapstructure:"enabled"`         // Enable on-the-fly image transformations
	DefaultQuality int      `mapstructure:"default_quality"` // Default output quality (1-100)
	MaxWidth       int      `mapstructure:"max_width"`       // Maximum output width in pixels
	MaxHeight      int      `mapstructure:"max_height"`      // Maximum output height in pixels
	AllowedFormats []string `mapstructure:"allowed_formats"` // Allowed output formats (webp, jpg, png, avif)

	// Security settings
	MaxTotalPixels int           `mapstructure:"max_total_pixels"` // Maximum total pixels (width * height), default 16M
	BucketSize     int           `mapstructure:"bucket_size"`      // Dimension bucketing size (default 50px)
	RateLimit      int           `mapstructure:"rate_limit"`       // Transforms per minute per user (default 60)
	Timeout        time.Duration `mapstructure:"timeout"`          // Max transform duration (default 30s)
	MaxConcurrent  int           `mapstructure:"max_concurrent"`   // Max concurrent transforms (default 4)

	// Caching settings
	CacheEnabled bool          `mapstructure:"cache_enabled"`  // Enable transform caching
	CacheTTL     time.Duration `mapstructure:"cache_ttl"`      // Cache TTL (default 24h)
	CacheMaxSize int64         `mapstructure:"cache_max_size"` // Max cache size in bytes (default 1GB)
}

// Validate validates storage configuration
func (sc *StorageConfig) Validate() error {
	if sc.Provider != "local" && sc.Provider != "s3" {
		return fmt.Errorf("storage provider must be 'local' or 's3', got: %s", sc.Provider)
	}

	if sc.Provider == "local" {
		if sc.LocalPath == "" {
			return fmt.Errorf("local_path is required when using local storage provider")
		}
	}

	if sc.Provider == "s3" {
		if sc.S3Endpoint == "" {
			return fmt.Errorf("s3_endpoint is required when using S3 storage provider")
		}
		if sc.S3AccessKey == "" {
			return fmt.Errorf("s3_access_key is required when using S3 storage provider")
		}
		if sc.S3SecretKey == "" {
			return fmt.Errorf("s3_secret_key is required when using S3 storage provider")
		}
		if sc.S3Bucket == "" {
			return fmt.Errorf("s3_bucket is required when using S3 storage provider")
		}
		// S3Region is optional for some S3-compatible services
	}

	// Validate max upload size
	if sc.MaxUploadSize <= 0 {
		return fmt.Errorf("max_upload_size must be positive, got: %d", sc.MaxUploadSize)
	}

	return nil
}
