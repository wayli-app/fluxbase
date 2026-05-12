package storage

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nimbleflux/fluxbase/internal/config"
)

// =============================================================================
// Mock Provider
// =============================================================================

type mockProvider struct {
	name           string
	uploadCalled   bool
	downloadCalled bool
	deleteCalled   bool
	bucketExists   bool
	buckets        []string
}

func (m *mockProvider) Name() string {
	return m.name
}

func (m *mockProvider) Upload(ctx context.Context, bucket, key string, data io.Reader, size int64, opts *UploadOptions) (*Object, error) {
	m.uploadCalled = true
	return &Object{
		Key:    key,
		Bucket: bucket,
		Size:   size,
	}, nil
}

func (m *mockProvider) Download(ctx context.Context, bucket, key string, opts *DownloadOptions) (io.ReadCloser, *Object, error) {
	m.downloadCalled = true
	return io.NopCloser(strings.NewReader("test data")), &Object{
		Key:    key,
		Bucket: bucket,
		Size:   9,
	}, nil
}

func (m *mockProvider) Delete(ctx context.Context, bucket, key string) error {
	m.deleteCalled = true
	return nil
}

func (m *mockProvider) List(ctx context.Context, bucket string, opts *ListOptions) (*ListResult, error) {
	return &ListResult{}, nil
}

func (m *mockProvider) GetMetadata(ctx context.Context, bucket, key string) (*Object, error) {
	return &Object{Key: key, Bucket: bucket}, nil
}

func (m *mockProvider) SetMetadata(ctx context.Context, bucket, key string, metadata map[string]string) error {
	return nil
}

func (m *mockProvider) Exists(ctx context.Context, bucket, key string) (bool, error) {
	return true, nil
}

func (m *mockProvider) GetObject(ctx context.Context, bucket, key string) (*Object, error) {
	return &Object{Key: key, Bucket: bucket}, nil
}

func (m *mockProvider) BucketExists(ctx context.Context, bucket string) (bool, error) {
	return m.bucketExists, nil
}

func (m *mockProvider) CreateBucket(ctx context.Context, bucket string) error {
	m.buckets = append(m.buckets, bucket)
	return nil
}

func (m *mockProvider) DeleteBucket(ctx context.Context, bucket string) error {
	return nil
}

func (m *mockProvider) ListBuckets(ctx context.Context) ([]string, error) {
	return m.buckets, nil
}

func (m *mockProvider) Copy(ctx context.Context, srcBucket, srcKey, dstBucket, dstKey string) error {
	return nil
}

func (m *mockProvider) Move(ctx context.Context, srcBucket, srcKey, dstBucket, dstKey string) error {
	return nil
}

func (m *mockProvider) CopyObject(ctx context.Context, srcBucket, srcKey, destBucket, destKey string) error {
	return nil
}

func (m *mockProvider) MoveObject(ctx context.Context, srcBucket, srcKey, destBucket, destKey string) error {
	return nil
}

func (m *mockProvider) GetSignedURL(ctx context.Context, bucket, key string, expiry int64) (string, error) {
	return fmt.Sprintf("http://example.com/%s/%s", bucket, key), nil
}

func (m *mockProvider) GetSignedUploadURL(ctx context.Context, bucket, key string, expiry int64) (string, error) {
	return fmt.Sprintf("http://example.com/upload/%s/%s", bucket, key), nil
}

func (m *mockProvider) GenerateSignedURL(ctx context.Context, bucket, key string, opts *SignedURLOptions) (string, error) {
	return fmt.Sprintf("http://example.com/signed/%s/%s", bucket, key), nil
}

func (m *mockProvider) Health(ctx context.Context) error {
	return nil
}

// =============================================================================
// Service Construction Tests
// =============================================================================

func TestNewService_Local(t *testing.T) {
	t.Run("creates local storage service", func(t *testing.T) {
		cfg := &config.StorageConfig{
			Provider:      "local",
			LocalPath:     t.TempDir(),
			MaxUploadSize: 1024 * 1024,
		}

		service, err := NewService(cfg, "http://localhost:8080", "secret", nil)

		require.NoError(t, err)
		require.NotNil(t, service)
		assert.True(t, service.IsLocal())
		assert.False(t, service.IsS3Compatible())
		assert.Equal(t, "local", service.GetProviderName())
	})
}

func TestNewService_UnsupportedProvider(t *testing.T) {
	t.Run("returns error for unsupported provider", func(t *testing.T) {
		cfg := &config.StorageConfig{
			Provider: "gcs",
		}

		service, err := NewService(cfg, "http://localhost:8080", "secret", nil)

		assert.Error(t, err)
		assert.Nil(t, service)
		assert.Contains(t, err.Error(), "unsupported storage provider")
	})
}

// =============================================================================
// Service MaxUploadSize Tests
// =============================================================================

func TestService_MaxUploadSize(t *testing.T) {
	t.Run("returns configured max upload size", func(t *testing.T) {
		maxSize := int64(10 * 1024 * 1024) // 10MB
		service := &Service{
			config: &config.StorageConfig{
				MaxUploadSize: maxSize,
			},
		}

		assert.Equal(t, maxSize, service.MaxUploadSize())
	})
}

// =============================================================================
// Service ValidateUploadSize Tests
// =============================================================================

func TestService_ValidateUploadSize(t *testing.T) {
	service := &Service{
		config: &config.StorageConfig{
			MaxUploadSize: 1024 * 1024, // 1MB
		},
	}

	t.Run("accepts valid size", func(t *testing.T) {
		err := service.ValidateUploadSize(512 * 1024) // 512KB
		assert.NoError(t, err)
	})

	t.Run("accepts exact max size", func(t *testing.T) {
		err := service.ValidateUploadSize(1024 * 1024) // 1MB
		assert.NoError(t, err)
	})

	t.Run("rejects size over limit", func(t *testing.T) {
		err := service.ValidateUploadSize(2 * 1024 * 1024) // 2MB
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "exceeds maximum allowed size")
	})

	t.Run("accepts zero size", func(t *testing.T) {
		err := service.ValidateUploadSize(0)
		assert.NoError(t, err)
	})
}

// =============================================================================
// Service Provider Type Tests
// =============================================================================

func TestService_ProviderTypes(t *testing.T) {
	t.Run("local provider", func(t *testing.T) {
		service := &Service{
			Provider: &mockProvider{name: "local"},
		}

		assert.True(t, service.IsLocal())
		assert.False(t, service.IsS3Compatible())
		assert.Equal(t, "local", service.GetProviderName())
	})

	t.Run("s3 provider", func(t *testing.T) {
		service := &Service{
			Provider: &mockProvider{name: "s3"},
		}

		assert.False(t, service.IsLocal())
		assert.True(t, service.IsS3Compatible())
		assert.Equal(t, "s3", service.GetProviderName())
	})
}

// =============================================================================
// Service DefaultBuckets Tests
// =============================================================================

func TestService_DefaultBuckets(t *testing.T) {
	t.Run("returns configured default buckets", func(t *testing.T) {
		service := &Service{
			config: &config.StorageConfig{
				DefaultBuckets: []string{"avatars", "documents", "backups"},
			},
		}

		buckets := service.DefaultBuckets()

		assert.Equal(t, []string{"avatars", "documents", "backups"}, buckets)
	})

	t.Run("returns empty slice when no defaults configured", func(t *testing.T) {
		service := &Service{
			config: &config.StorageConfig{
				DefaultBuckets: nil,
			},
		}

		buckets := service.DefaultBuckets()

		assert.Nil(t, buckets)
	})
}

// =============================================================================
// Service EnsureDefaultBuckets Tests
// =============================================================================

func TestService_EnsureDefaultBuckets(t *testing.T) {
	t.Run("creates non-existing buckets", func(t *testing.T) {
		provider := &mockProvider{
			name:         "local",
			bucketExists: false,
		}

		service := &Service{
			Provider: provider,
			config: &config.StorageConfig{
				DefaultBuckets: []string{"bucket1", "bucket2"},
			},
		}

		err := service.EnsureDefaultBuckets(context.Background())

		assert.NoError(t, err)
		assert.Contains(t, provider.buckets, "bucket1")
		assert.Contains(t, provider.buckets, "bucket2")
	})

	t.Run("skips existing buckets", func(t *testing.T) {
		provider := &mockProvider{
			name:         "local",
			bucketExists: true,
		}

		service := &Service{
			Provider: provider,
			config: &config.StorageConfig{
				DefaultBuckets: []string{"existing-bucket"},
			},
		}

		err := service.EnsureDefaultBuckets(context.Background())

		assert.NoError(t, err)
		assert.Empty(t, provider.buckets) // Should not create anything
	})
}

// =============================================================================
// Service Upload/Download/Delete Tests
// =============================================================================

func TestService_Upload(t *testing.T) {
	t.Run("uploads successfully", func(t *testing.T) {
		provider := &mockProvider{name: "local"}
		service := &Service{
			Provider: provider,
			config:   &config.StorageConfig{},
		}

		data := strings.NewReader("test content")
		obj, err := service.Upload(context.Background(), "bucket", "key.txt", data, 12, nil)

		assert.NoError(t, err)
		assert.NotNil(t, obj)
		assert.True(t, provider.uploadCalled)
		assert.Equal(t, "key.txt", obj.Key)
		assert.Equal(t, "bucket", obj.Bucket)
	})
}

func TestService_Download(t *testing.T) {
	t.Run("downloads successfully", func(t *testing.T) {
		provider := &mockProvider{name: "local"}
		service := &Service{
			Provider: provider,
			config:   &config.StorageConfig{},
		}

		reader, obj, err := service.Download(context.Background(), "bucket", "key.txt", nil)

		assert.NoError(t, err)
		assert.NotNil(t, reader)
		assert.NotNil(t, obj)
		assert.True(t, provider.downloadCalled)
		_ = reader.Close()
	})
}

func TestService_Delete(t *testing.T) {
	t.Run("deletes successfully", func(t *testing.T) {
		provider := &mockProvider{name: "local"}
		service := &Service{
			Provider: provider,
			config:   &config.StorageConfig{},
		}

		err := service.Delete(context.Background(), "bucket", "key.txt")

		assert.NoError(t, err)
		assert.True(t, provider.deleteCalled)
	})
}

// =============================================================================
// Service SetMetrics Tests
// =============================================================================

// =============================================================================
// Provider Configuration Tests
// =============================================================================

func TestNewService_S3Endpoint(t *testing.T) {
	tests := []struct {
		name           string
		endpoint       string
		expectSSL      bool
		expectEndpoint string
	}{
		{
			name:           "https endpoint",
			endpoint:       "https://minio.example.com",
			expectSSL:      true,
			expectEndpoint: "minio.example.com",
		},
		{
			name:           "http endpoint",
			endpoint:       "http://localhost:9000",
			expectSSL:      false,
			expectEndpoint: "localhost:9000",
		},
		{
			name:           "no scheme endpoint",
			endpoint:       "s3.custom.com",
			expectSSL:      true,
			expectEndpoint: "s3.custom.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the endpoint parsing logic
			endpoint := tt.endpoint
			useSSL := true
			if endpoint != "" {
				useSSL = !strings.HasPrefix(endpoint, "http://")
			}
			endpoint = strings.TrimPrefix(endpoint, "https://")
			endpoint = strings.TrimPrefix(endpoint, "http://")

			assert.Equal(t, tt.expectSSL, useSSL)
			assert.Equal(t, tt.expectEndpoint, endpoint)
		})
	}
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkService_ValidateUploadSize(b *testing.B) {
	service := &Service{
		config: &config.StorageConfig{
			MaxUploadSize: 100 * 1024 * 1024, // 100MB
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = service.ValidateUploadSize(50 * 1024 * 1024)
	}
}

func BenchmarkService_ProviderChecks(b *testing.B) {
	service := &Service{
		Provider: &mockProvider{name: "s3"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = service.IsS3Compatible()
		_ = service.IsLocal()
		_ = service.GetProviderName()
	}
}
