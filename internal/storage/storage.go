package storage

import (
	"context"
	"io"
	"time"
)

// Object represents a stored file
type Object struct {
	Key          string            `json:"key"`
	Bucket       string            `json:"bucket"`
	Size         int64             `json:"size"`
	ContentType  string            `json:"content_type"`
	LastModified time.Time         `json:"last_modified"`
	ETag         string            `json:"etag,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// UploadOptions contains options for uploading files
type UploadOptions struct {
	ContentType     string
	Metadata        map[string]string
	CacheControl    string
	ContentEncoding string
}

// DownloadOptions contains options for downloading files
type DownloadOptions struct {
	IfModifiedSince   *time.Time
	IfUnmodifiedSince *time.Time
	IfMatch           string
	IfNoneMatch       string
	Range             string
}

// SignedURLOptions contains options for generating signed URLs
type SignedURLOptions struct {
	ExpiresIn   time.Duration
	Method      string // GET, PUT, DELETE
	ContentType string
}

// ListOptions contains options for listing objects
type ListOptions struct {
	Prefix     string
	MaxKeys    int
	Delimiter  string
	StartAfter string
}

// ListResult contains the result of a list operation
type ListResult struct {
	Objects        []Object
	CommonPrefixes []string
	IsTruncated    bool
	NextMarker     string
}

// Storage defines the interface for file storage operations
type Storage interface {
	// Upload uploads a file to storage
	Upload(ctx context.Context, bucket, key string, data io.Reader, size int64, opts *UploadOptions) (*Object, error)

	// Download downloads a file from storage
	Download(ctx context.Context, bucket, key string, opts *DownloadOptions) (io.ReadCloser, *Object, error)

	// Delete deletes a file from storage
	Delete(ctx context.Context, bucket, key string) error

	// Exists checks if a file exists
	Exists(ctx context.Context, bucket, key string) (bool, error)

	// GetObject gets object metadata without downloading the file
	GetObject(ctx context.Context, bucket, key string) (*Object, error)

	// List lists objects in a bucket
	List(ctx context.Context, bucket string, opts *ListOptions) (*ListResult, error)

	// CreateBucket creates a new bucket
	CreateBucket(ctx context.Context, bucket string) error

	// DeleteBucket deletes a bucket (must be empty)
	DeleteBucket(ctx context.Context, bucket string) error

	// BucketExists checks if a bucket exists
	BucketExists(ctx context.Context, bucket string) (bool, error)

	// ListBuckets lists all buckets
	ListBuckets(ctx context.Context) ([]string, error)

	// GenerateSignedURL generates a signed URL for temporary access
	GenerateSignedURL(ctx context.Context, bucket, key string, opts *SignedURLOptions) (string, error)

	// CopyObject copies an object within storage
	CopyObject(ctx context.Context, srcBucket, srcKey, destBucket, destKey string) error

	// MoveObject moves an object (copy + delete)
	MoveObject(ctx context.Context, srcBucket, srcKey, destBucket, destKey string) error
}

// Provider is the interface that storage providers must implement
type Provider interface {
	Storage
	Name() string
	Health(ctx context.Context) error
}
