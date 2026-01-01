package storage

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/fluxbase-eu/fluxbase/internal/config"
	"github.com/fluxbase-eu/fluxbase/internal/observability"
	"github.com/rs/zerolog/log"
)

// Service wraps a storage provider and provides additional functionality
type Service struct {
	Provider Provider
	config   *config.StorageConfig
	metrics  *observability.Metrics
}

// SetMetrics sets the metrics instance for recording storage metrics
func (s *Service) SetMetrics(m *observability.Metrics) {
	s.metrics = m
}

// Upload wraps the provider's Upload method with metrics
func (s *Service) Upload(ctx context.Context, bucket, key string, data io.Reader, size int64, opts *UploadOptions) (*Object, error) {
	start := time.Now()
	obj, err := s.Provider.Upload(ctx, bucket, key, data, size, opts)
	duration := time.Since(start)

	if s.metrics != nil {
		s.metrics.RecordStorageOperation("upload", bucket, size, duration, err)
	}

	return obj, err
}

// Download wraps the provider's Download method with metrics
func (s *Service) Download(ctx context.Context, bucket, key string, opts *DownloadOptions) (io.ReadCloser, *Object, error) {
	start := time.Now()
	reader, obj, err := s.Provider.Download(ctx, bucket, key, opts)
	duration := time.Since(start)

	var size int64
	if obj != nil {
		size = obj.Size
	}

	if s.metrics != nil {
		s.metrics.RecordStorageOperation("download", bucket, size, duration, err)
	}

	return reader, obj, err
}

// Delete wraps the provider's Delete method with metrics
func (s *Service) Delete(ctx context.Context, bucket, key string) error {
	start := time.Now()
	err := s.Provider.Delete(ctx, bucket, key)
	duration := time.Since(start)

	if s.metrics != nil {
		s.metrics.RecordStorageOperation("delete", bucket, 0, duration, err)
	}

	return err
}

// NewService creates a new storage service based on configuration
// baseURL is used for generating signed URLs (e.g., "http://localhost:8080")
// signingSecret is used for signing local storage URLs (typically the JWT secret)
func NewService(cfg *config.StorageConfig, baseURL, signingSecret string) (*Service, error) {
	var provider Provider
	var err error

	switch strings.ToLower(cfg.Provider) {
	case "local":
		provider, err = NewLocalStorage(cfg.LocalPath, baseURL, signingSecret)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize local storage: %w", err)
		}

	case "s3":
		// Determine if using SSL based on endpoint
		useSSL := true
		if cfg.S3Endpoint != "" {
			// If endpoint is specified (MinIO), check if it's http or https
			useSSL = !strings.HasPrefix(cfg.S3Endpoint, "http://")
		}

		// Remove http:// or https:// prefix if present
		endpoint := cfg.S3Endpoint
		endpoint = strings.TrimPrefix(endpoint, "https://")
		endpoint = strings.TrimPrefix(endpoint, "http://")

		// If no endpoint specified, use default S3 endpoint
		if endpoint == "" {
			endpoint = "s3.amazonaws.com"
			useSSL = true
		}

		provider, err = NewS3Storage(
			endpoint,
			cfg.S3AccessKey,
			cfg.S3SecretKey,
			cfg.S3Region,
			useSSL,
			cfg.S3ForcePathStyle,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize S3 storage: %w", err)
		}

	default:
		return nil, fmt.Errorf("unsupported storage provider: %s", cfg.Provider)
	}

	return &Service{
		Provider: provider,
		config:   cfg,
	}, nil
}

// MaxUploadSize returns the maximum allowed upload size
func (s *Service) MaxUploadSize() int64 {
	return s.config.MaxUploadSize
}

// ValidateUploadSize checks if the upload size is within limits
func (s *Service) ValidateUploadSize(size int64) error {
	if size > s.config.MaxUploadSize {
		return fmt.Errorf("file size %d exceeds maximum allowed size %d", size, s.config.MaxUploadSize)
	}
	return nil
}

// GetProviderName returns the name of the storage provider
func (s *Service) GetProviderName() string {
	return s.Provider.Name()
}

// IsS3Compatible returns true if the storage provider is S3-compatible
func (s *Service) IsS3Compatible() bool {
	return s.Provider.Name() == "s3"
}

// IsLocal returns true if the storage provider is local filesystem
func (s *Service) IsLocal() bool {
	return s.Provider.Name() == "local"
}

// DefaultBuckets returns the list of default buckets from config
func (s *Service) DefaultBuckets() []string {
	return s.config.DefaultBuckets
}

// EnsureDefaultBuckets creates the default buckets if they don't exist
func (s *Service) EnsureDefaultBuckets(ctx context.Context) error {
	for _, bucket := range s.config.DefaultBuckets {
		exists, err := s.Provider.BucketExists(ctx, bucket)
		if err != nil {
			log.Warn().Err(err).Str("bucket", bucket).Msg("Failed to check if bucket exists")
			continue
		}

		if !exists {
			if err := s.Provider.CreateBucket(ctx, bucket); err != nil {
				log.Warn().Err(err).Str("bucket", bucket).Msg("Failed to create default bucket")
				continue
			}
			log.Info().Str("bucket", bucket).Msg("Created default bucket")
		}
	}
	return nil
}
