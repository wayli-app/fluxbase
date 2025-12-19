package storage

import (
	"context"
)

// LogStorage defines the interface for log storage backends.
// Implementations can store logs in PostgreSQL, S3, local filesystem, etc.
type LogStorage interface {
	// Name returns the backend identifier (e.g., "postgres", "s3", "local").
	Name() string

	// Write writes a batch of log entries to the backend.
	// Implementations should handle batching efficiently.
	Write(ctx context.Context, entries []*LogEntry) error

	// Query retrieves logs matching the given options.
	// Returns a QueryResult with entries, total count, and pagination info.
	Query(ctx context.Context, opts LogQueryOptions) (*LogQueryResult, error)

	// GetExecutionLogs retrieves logs for a specific execution.
	// This is optimized for streaming execution logs with line number ordering.
	// Use afterLine to get logs after a specific line number for pagination.
	GetExecutionLogs(ctx context.Context, executionID string, afterLine int) ([]*LogEntry, error)

	// Delete removes logs matching the given options.
	// Used for retention cleanup. Returns the number of deleted entries.
	Delete(ctx context.Context, opts LogQueryOptions) (int64, error)

	// Stats returns statistics about stored logs.
	Stats(ctx context.Context) (*LogStats, error)

	// Health checks if the backend is operational.
	Health(ctx context.Context) error

	// Close releases resources held by the backend.
	Close() error
}

// LogStorageConfig contains configuration for creating a LogStorage instance.
type LogStorageConfig struct {
	// Backend type: "postgres", "s3", "local"
	Backend string `mapstructure:"backend"`

	// PostgreSQL settings (used when backend is "postgres")
	// Uses the main database connection

	// S3 settings (used when backend is "s3")
	S3Bucket string `mapstructure:"s3_bucket"`
	S3Prefix string `mapstructure:"s3_prefix"`

	// Local filesystem settings (used when backend is "local")
	LocalPath string `mapstructure:"local_path"`

	// Batching configuration
	BatchSize     int `mapstructure:"batch_size"`
	FlushInterval int `mapstructure:"flush_interval_ms"` // milliseconds

	// Buffer size for async writes
	BufferSize int `mapstructure:"buffer_size"`
}

// DefaultLogStorageConfig returns a LogStorageConfig with sensible defaults.
func DefaultLogStorageConfig() LogStorageConfig {
	return LogStorageConfig{
		Backend:       "postgres",
		S3Prefix:      "logs",
		LocalPath:     "./logs",
		BatchSize:     100,
		FlushInterval: 1000, // 1 second
		BufferSize:    10000,
	}
}
