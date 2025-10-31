package storage

import (
	"fmt"
	"strings"

	"github.com/wayli-app/fluxbase/internal/config"
)

// Service wraps a storage provider and provides additional functionality
type Service struct {
	Provider Provider
	config   *config.StorageConfig
}

// NewService creates a new storage service based on configuration
func NewService(cfg *config.StorageConfig) (*Service, error) {
	var provider Provider
	var err error

	switch strings.ToLower(cfg.Provider) {
	case "local":
		provider, err = NewLocalStorage(cfg.LocalPath)
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
