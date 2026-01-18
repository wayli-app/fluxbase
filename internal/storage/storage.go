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
	// Transform options (for image downloads)
	TransformWidth   int    // Target width in pixels
	TransformHeight  int    // Target height in pixels
	TransformFormat  string // Output format: webp, jpg, png, avif
	TransformQuality int    // Output quality 1-100
	TransformFit     string // Fit mode: cover, contain, fill, inside, outside
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

// ChunkedUploadSession represents an in-progress chunked upload
type ChunkedUploadSession struct {
	UploadID        string            `json:"upload_id"`
	Bucket          string            `json:"bucket"`
	Key             string            `json:"key"`
	TotalSize       int64             `json:"total_size"`
	ChunkSize       int64             `json:"chunk_size"`
	TotalChunks     int               `json:"total_chunks"`
	CompletedChunks []int             `json:"completed_chunks"`
	ContentType     string            `json:"content_type,omitempty"`
	Metadata        map[string]string `json:"metadata,omitempty"`
	CacheControl    string            `json:"cache_control,omitempty"`
	OwnerID         string            `json:"owner_id,omitempty"`

	// S3 multipart specific fields
	S3UploadID  string         `json:"s3_upload_id,omitempty"`
	S3PartETags map[int]string `json:"s3_part_etags,omitempty"`

	// Session lifecycle
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// ChunkResult represents the result of uploading a chunk
type ChunkResult struct {
	ChunkIndex int    `json:"chunk_index"`
	ETag       string `json:"etag,omitempty"`
	Size       int64  `json:"size"`
}

// ChunkedUploader defines the interface for chunked upload operations
type ChunkedUploader interface {
	// InitChunkedUpload starts a new chunked upload session
	InitChunkedUpload(ctx context.Context, bucket, key string, totalSize int64, chunkSize int64, opts *UploadOptions) (*ChunkedUploadSession, error)

	// UploadChunk uploads a single chunk of data
	UploadChunk(ctx context.Context, session *ChunkedUploadSession, chunkIndex int, data io.Reader, size int64) (*ChunkResult, error)

	// CompleteChunkedUpload finalizes the upload and assembles the file
	CompleteChunkedUpload(ctx context.Context, session *ChunkedUploadSession) (*Object, error)

	// AbortChunkedUpload cancels the upload and cleans up chunks
	AbortChunkedUpload(ctx context.Context, session *ChunkedUploadSession) error
}
