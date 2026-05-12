package storage

import (
	"context"
	"os"
	"testing"

	"github.com/nimbleflux/fluxbase/internal/config"
)

func TestNewManager(t *testing.T) {
	cfg := &config.StorageConfig{
		Provider:  "local",
		LocalPath: t.TempDir(),
	}

	manager, err := NewManager(cfg, "http://localhost:8080", "test-secret-at-least-32-chars!", nil)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	if manager == nil {
		t.Fatal("Expected non-nil manager")
	}

	if manager.ServiceCount() != 1 {
		t.Errorf("Expected 1 service (base), got %d", manager.ServiceCount())
	}
}

func TestManager_GetBaseService(t *testing.T) {
	cfg := &config.StorageConfig{
		Provider:  "local",
		LocalPath: t.TempDir(),
	}

	manager, err := NewManager(cfg, "http://localhost:8080", "test-secret-at-least-32-chars!", nil)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	svc := manager.GetBaseService()
	if svc == nil {
		t.Fatal("Expected non-nil base service")
	}

	if svc.GetProviderName() != "local" {
		t.Errorf("Expected provider 'local', got %s", svc.GetProviderName())
	}
}

func TestManager_GetService_SameConfig(t *testing.T) {
	cfg := &config.StorageConfig{
		Provider:  "local",
		LocalPath: t.TempDir(),
	}

	manager, err := NewManager(cfg, "http://localhost:8080", "test-secret-at-least-32-chars!", nil)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Request with same config should return base service
	svc, err := manager.GetService(cfg)
	if err != nil {
		t.Fatalf("Failed to get service: %v", err)
	}

	if svc != manager.GetBaseService() {
		t.Error("Expected same service instance for same config")
	}

	// Service count should still be 1
	if manager.ServiceCount() != 1 {
		t.Errorf("Expected 1 service, got %d", manager.ServiceCount())
	}
}

func TestManager_GetService_NilConfig(t *testing.T) {
	cfg := &config.StorageConfig{
		Provider:  "local",
		LocalPath: t.TempDir(),
	}

	manager, err := NewManager(cfg, "http://localhost:8080", "test-secret-at-least-32-chars!", nil)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Request with nil config should return base service
	svc, err := manager.GetService(nil)
	if err != nil {
		t.Fatalf("Failed to get service: %v", err)
	}

	if svc != manager.GetBaseService() {
		t.Error("Expected base service for nil config")
	}
}

func TestManager_GetService_DifferentConfig(t *testing.T) {
	baseCfg := &config.StorageConfig{
		Provider:  "local",
		LocalPath: t.TempDir(),
	}

	manager, err := NewManager(baseCfg, "http://localhost:8080", "test-secret-at-least-32-chars!", nil)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Request with different local path
	tenantCfg := &config.StorageConfig{
		Provider:  "local",
		LocalPath: t.TempDir(), // Different path
	}

	svc, err := manager.GetService(tenantCfg)
	if err != nil {
		t.Fatalf("Failed to get service: %v", err)
	}

	if svc == nil {
		t.Fatal("Expected non-nil service")
	}

	// Should be a different service instance
	if svc == manager.GetBaseService() {
		t.Error("Expected different service instance for different config")
	}

	// Service count should now be 2
	if manager.ServiceCount() != 2 {
		t.Errorf("Expected 2 services, got %d", manager.ServiceCount())
	}

	// Request same config again - should return cached service
	svc2, err := manager.GetService(tenantCfg)
	if err != nil {
		t.Fatalf("Failed to get cached service: %v", err)
	}

	if svc != svc2 {
		t.Error("Expected cached service instance")
	}

	// Service count should still be 2 (not 3)
	if manager.ServiceCount() != 2 {
		t.Errorf("Expected 2 services after cache hit, got %d", manager.ServiceCount())
	}
}

func TestManager_RefreshService(t *testing.T) {
	cfg := &config.StorageConfig{
		Provider:  "local",
		LocalPath: t.TempDir(),
	}

	manager, err := NewManager(cfg, "http://localhost:8080", "test-secret-at-least-32-chars!", nil)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Create a tenant service first
	tenantCfg := &config.StorageConfig{
		Provider:  "local",
		LocalPath: t.TempDir(),
	}

	_, err = manager.GetService(tenantCfg)
	if err != nil {
		t.Fatalf("Failed to get service: %v", err)
	}

	// Refresh the service
	err = manager.RefreshService(context.Background(), tenantCfg)
	if err != nil {
		t.Fatalf("Failed to refresh service: %v", err)
	}

	// Service count should still be 2 (refresh doesn't add new)
	if manager.ServiceCount() != 2 {
		t.Errorf("Expected 2 services after refresh, got %d", manager.ServiceCount())
	}
}

func TestManager_EnsureDefaultBuckets(t *testing.T) {
	cfg := &config.StorageConfig{
		Provider:       "local",
		LocalPath:      t.TempDir(),
		DefaultBuckets: []string{"test-bucket"},
	}

	manager, err := NewManager(cfg, "http://localhost:8080", "test-secret-at-least-32-chars!", nil)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	err = manager.EnsureDefaultBuckets(context.Background())
	if err != nil {
		t.Fatalf("Failed to ensure default buckets: %v", err)
	}

	// Verify bucket exists
	svc := manager.GetBaseService()
	exists, err := svc.Provider.BucketExists(context.Background(), "test-bucket")
	if err != nil {
		t.Fatalf("Failed to check bucket: %v", err)
	}

	if !exists {
		t.Error("Expected test-bucket to exist")
	}
}

func TestConfigKey(t *testing.T) {
	tests := []struct {
		name     string
		config   *config.StorageConfig
		expected string
	}{
		{
			name:     "nil config",
			config:   nil,
			expected: "",
		},
		{
			name: "s3 with region",
			config: &config.StorageConfig{
				Provider:   "s3",
				S3Bucket:   "my-bucket",
				S3Region:   "us-east-1",
				S3Endpoint: "",
			},
			expected: "s3:my-bucket:us-east-1:",
		},
		{
			name: "s3 with endpoint",
			config: &config.StorageConfig{
				Provider:   "s3",
				S3Bucket:   "my-bucket",
				S3Region:   "eu-west-1",
				S3Endpoint: "minio.example.com",
			},
			expected: "s3:my-bucket:eu-west-1:minio.example.com",
		},
		{
			name: "local storage",
			config: &config.StorageConfig{
				Provider:  "local",
				LocalPath: "/data/storage",
			},
			expected: "local:/data/storage",
		},
		{
			name: "unknown provider",
			config: &config.StorageConfig{
				Provider: "unknown",
			},
			expected: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := configKey(tt.config)
			if key != tt.expected {
				t.Errorf("Expected key %q, got %q", tt.expected, key)
			}
		})
	}
}

func TestConfigEqual(t *testing.T) {
	tests := []struct {
		name     string
		a        *config.StorageConfig
		b        *config.StorageConfig
		expected bool
	}{
		{
			name:     "both nil",
			a:        nil,
			b:        nil,
			expected: true,
		},
		{
			name:     "one nil",
			a:        &config.StorageConfig{Provider: "local"},
			b:        nil,
			expected: false,
		},
		{
			name:     "different providers",
			a:        &config.StorageConfig{Provider: "local"},
			b:        &config.StorageConfig{Provider: "s3"},
			expected: false,
		},
		{
			name:     "same local config",
			a:        &config.StorageConfig{Provider: "local", LocalPath: "/data"},
			b:        &config.StorageConfig{Provider: "local", LocalPath: "/data"},
			expected: true,
		},
		{
			name:     "different local paths",
			a:        &config.StorageConfig{Provider: "local", LocalPath: "/data1"},
			b:        &config.StorageConfig{Provider: "local", LocalPath: "/data2"},
			expected: false,
		},
		{
			name: "same s3 config",
			a: &config.StorageConfig{
				Provider:    "s3",
				S3Bucket:    "bucket",
				S3Region:    "us-east-1",
				S3AccessKey: "key",
				S3SecretKey: "secret",
			},
			b: &config.StorageConfig{
				Provider:    "s3",
				S3Bucket:    "bucket",
				S3Region:    "us-east-1",
				S3AccessKey: "key",
				S3SecretKey: "secret",
			},
			expected: true,
		},
		{
			name: "different s3 buckets",
			a: &config.StorageConfig{
				Provider: "s3",
				S3Bucket: "bucket1",
			},
			b: &config.StorageConfig{
				Provider: "s3",
				S3Bucket: "bucket2",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := configEqual(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestManager_S3Config(t *testing.T) {
	// Skip if no S3 credentials available
	if os.Getenv("AWS_ACCESS_KEY_ID") == "" {
		t.Skip("Skipping S3 test - no AWS credentials")
	}

	baseCfg := &config.StorageConfig{
		Provider:  "local",
		LocalPath: t.TempDir(),
	}

	manager, err := NewManager(baseCfg, "http://localhost:8080", "test-secret-at-least-32-chars!", nil)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Request S3 config (will fail without real credentials, but tests the key generation)
	s3Cfg := &config.StorageConfig{
		Provider:    "s3",
		S3Bucket:    "test-bucket",
		S3Region:    "us-east-1",
		S3AccessKey: os.Getenv("AWS_ACCESS_KEY_ID"),
		S3SecretKey: os.Getenv("AWS_SECRET_ACCESS_KEY"),
	}

	// This should create a new service (even if it fails to connect)
	svc, err := manager.GetService(s3Cfg)
	// We don't fail the test if it errors - we just want to verify the key is different
	if err == nil && svc != manager.GetBaseService() {
		// Successfully created a separate S3 service
		if manager.ServiceCount() != 2 {
			t.Errorf("Expected 2 services after S3 config, got %d", manager.ServiceCount())
		}
	}
}
