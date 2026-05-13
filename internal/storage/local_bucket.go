package storage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog/log"
)

// CreateBucket creates a new bucket
func (ls *LocalStorage) CreateBucket(ctx context.Context, bucket string) error {
	bucketPath := filepath.Join(ls.basePath, bucket)

	// Check if bucket already exists
	if _, err := os.Stat(bucketPath); err == nil {
		return fmt.Errorf("bucket already exists")
	}

	// Create bucket directory
	if err := os.MkdirAll(bucketPath, 0o755); err != nil {
		return fmt.Errorf("failed to create bucket: %w", err)
	}

	log.Info().Str("bucket", bucket).Msg("Bucket created")
	return nil
}

// DeleteBucket deletes a bucket (must be empty)
func (ls *LocalStorage) DeleteBucket(ctx context.Context, bucket string) error {
	bucketPath := filepath.Join(ls.basePath, bucket)

	// Check if bucket exists
	if _, err := os.Stat(bucketPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("bucket not found")
		}
		return err
	}

	// Check if bucket contains any files (not just directories)
	hasFiles := false
	err := filepath.Walk(bucketPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// Skip the bucket directory itself and any metadata files
		if path != bucketPath && !info.IsDir() && !strings.HasSuffix(path, ".meta") {
			hasFiles = true
			return filepath.SkipDir // Stop walking once we find a file
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to check bucket contents: %w", err)
	}

	if hasFiles {
		return fmt.Errorf("bucket is not empty")
	}

	// Delete bucket directory and all empty subdirectories
	if err := os.RemoveAll(bucketPath); err != nil {
		return fmt.Errorf("failed to delete bucket: %w", err)
	}

	log.Info().Str("bucket", bucket).Msg("Bucket deleted")
	return nil
}

// BucketExists checks if a bucket exists
func (ls *LocalStorage) BucketExists(ctx context.Context, bucket string) (bool, error) {
	bucketPath := filepath.Join(ls.basePath, bucket)
	info, err := os.Stat(bucketPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return info.IsDir(), nil
}

// ListBuckets lists all buckets
func (ls *LocalStorage) ListBuckets(ctx context.Context) ([]string, error) {
	entries, err := os.ReadDir(ls.basePath)
	if err != nil {
		return nil, fmt.Errorf("failed to list buckets: %w", err)
	}

	var buckets []string
	for _, entry := range entries {
		if entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
			buckets = append(buckets, entry.Name())
		}
	}

	return buckets, nil
}
