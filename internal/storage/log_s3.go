package storage

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

// S3LogStorage implements LogStorage using S3-compatible object storage.
// It stores logs as NDJSON (newline-delimited JSON) files organized by date and category.
// Structure: {prefix}/{category}/{YYYY}/{MM}/{DD}/{execution_id|batch_uuid}.ndjson
type S3LogStorage struct {
	storage Provider
	bucket  string
	prefix  string
}

// NewS3LogStorage creates a new S3-backed log storage.
func NewS3LogStorage(storage Provider, bucket, prefix string) *S3LogStorage {
	if prefix == "" {
		prefix = "logs"
	}
	return &S3LogStorage{
		storage: storage,
		bucket:  bucket,
		prefix:  prefix,
	}
}

// Name returns the backend identifier.
func (s *S3LogStorage) Name() string {
	return "s3"
}

// Write writes a batch of log entries to S3.
// Entries are grouped by category and date, then written as NDJSON files.
func (s *S3LogStorage) Write(ctx context.Context, entries []*LogEntry) error {
	if len(entries) == 0 {
		return nil
	}

	// Group entries by category and execution ID (for execution logs) or batch
	groups := make(map[string][]*LogEntry)

	for _, entry := range entries {
		// Ensure ID and timestamp are set
		if entry.ID == uuid.Nil {
			entry.ID = uuid.New()
		}
		if entry.Timestamp.IsZero() {
			entry.Timestamp = time.Now()
		}

		var groupKey string
		if entry.Category == LogCategoryExecution && entry.ExecutionID != "" {
			// Group execution logs by execution ID
			groupKey = fmt.Sprintf("%s/%s/exec_%s",
				string(entry.Category),
				entry.Timestamp.Format("2006/01/02"),
				entry.ExecutionID,
			)
		} else {
			// Group other logs by category and date with a batch UUID
			groupKey = fmt.Sprintf("%s/%s/batch",
				string(entry.Category),
				entry.Timestamp.Format("2006/01/02"),
			)
		}

		groups[groupKey] = append(groups[groupKey], entry)
	}

	// Write each group to a separate file
	for groupKey, groupEntries := range groups {
		var key string
		if strings.HasSuffix(groupKey, "/batch") {
			// For batch logs, add a UUID to make the key unique
			key = fmt.Sprintf("%s/%s_%s.ndjson", s.prefix, groupKey, uuid.New().String()[:8])
		} else {
			// For execution logs, use the execution ID as the key
			key = fmt.Sprintf("%s/%s.ndjson", s.prefix, groupKey)
		}

		// For execution logs, use chunked writes instead of download-append-upload
		// to avoid memory exhaustion with large log files.
		// Each batch gets a unique suffix based on timestamp to prevent overwrites.
		if strings.Contains(groupKey, "exec_") {
			// Add timestamp suffix to create unique chunk files
			// Format: exec_{id}_{timestamp}.ndjson
			key = fmt.Sprintf("%s/%s_%d.ndjson", s.prefix, groupKey, time.Now().UnixNano())
		}

		// Serialize entries to NDJSON
		var buf bytes.Buffer
		enc := json.NewEncoder(&buf)
		for _, entry := range groupEntries {
			if err := enc.Encode(entry); err != nil {
				return fmt.Errorf("failed to encode log entry: %w", err)
			}
		}

		// Upload to S3
		data := buf.Bytes()
		if _, err := s.storage.Upload(ctx, s.bucket, key, bytes.NewReader(data), int64(len(data)), nil); err != nil {
			return fmt.Errorf("failed to upload log file: %w", err)
		}
	}

	return nil
}

// Query retrieves logs matching the given options.
// Note: S3 is not optimized for querying - this requires downloading and parsing files.
// For heavy querying, use PostgreSQL backend instead.
func (s *S3LogStorage) Query(ctx context.Context, opts LogQueryOptions) (*LogQueryResult, error) {
	// Build prefix for listing
	prefix := s.prefix + "/"
	if opts.Category != "" {
		prefix += string(opts.Category) + "/"
	}

	// If we have time range, add date prefix
	if !opts.StartTime.IsZero() {
		prefix += opts.StartTime.Format("2006/01/")
	}

	// List objects matching the prefix
	objects, err := s.storage.List(ctx, s.bucket, &ListOptions{
		Prefix: prefix,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list log files: %w", err)
	}

	// Download and parse each file
	var allEntries []*LogEntry
	for _, obj := range objects.Objects {
		if !strings.HasSuffix(obj.Key, ".ndjson") {
			continue
		}

		entries, err := s.downloadAndParseFile(ctx, obj.Key, opts)
		if err != nil {
			// Log error but continue with other files
			continue
		}
		allEntries = append(allEntries, entries...)
	}

	// Sort entries by timestamp
	sort.Slice(allEntries, func(i, j int) bool {
		if opts.SortAsc {
			return allEntries[i].Timestamp.Before(allEntries[j].Timestamp)
		}
		return allEntries[i].Timestamp.After(allEntries[j].Timestamp)
	})

	// Apply pagination
	totalCount := int64(len(allEntries))
	start := opts.Offset
	if start < 0 {
		start = 0
	}
	if start >= len(allEntries) {
		return &LogQueryResult{
			Entries:    []*LogEntry{},
			TotalCount: totalCount,
			HasMore:    false,
		}, nil
	}

	limit := opts.Limit
	if limit <= 0 {
		limit = 100
	}
	end := start + limit
	if end > len(allEntries) {
		end = len(allEntries)
	}

	return &LogQueryResult{
		Entries:    allEntries[start:end],
		TotalCount: totalCount,
		HasMore:    end < len(allEntries),
	}, nil
}

// GetExecutionLogs retrieves logs for a specific execution.
// Supports chunked storage where logs are stored in multiple files.
func (s *S3LogStorage) GetExecutionLogs(ctx context.Context, executionID string, afterLine int) ([]*LogEntry, error) {
	// Search for execution log files (may be multiple chunks)
	// We need to search across multiple dates since we don't know when the execution happened
	prefix := fmt.Sprintf("%s/%s/", s.prefix, string(LogCategoryExecution))

	objects, err := s.storage.List(ctx, s.bucket, &ListOptions{
		Prefix: prefix,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list execution log files: %w", err)
	}

	// Find all files for this execution (may be multiple chunks)
	var targetKeys []string
	execPrefix := "exec_" + executionID
	for _, obj := range objects.Objects {
		if strings.Contains(obj.Key, execPrefix) {
			targetKeys = append(targetKeys, obj.Key)
		}
	}

	if len(targetKeys) == 0 {
		return []*LogEntry{}, nil
	}

	// Download and parse all chunk files
	var entries []*LogEntry
	for _, targetKey := range targetKeys {
		reader, _, err := s.storage.Download(ctx, s.bucket, targetKey, nil)
		if err != nil {
			// Log error but continue with other files
			continue
		}

		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			var entry LogEntry
			if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
				continue
			}
			if entry.LineNumber > afterLine {
				entries = append(entries, &entry)
			}
		}
		_ = reader.Close()
	}

	// Sort by line number
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].LineNumber < entries[j].LineNumber
	})

	return entries, nil
}

// Delete removes logs matching the given options.
func (s *S3LogStorage) Delete(ctx context.Context, opts LogQueryOptions) (int64, error) {
	// Build prefix for listing
	prefix := s.prefix + "/"
	if opts.Category != "" {
		prefix += string(opts.Category) + "/"
	}

	// List objects matching the prefix
	objects, err := s.storage.List(ctx, s.bucket, &ListOptions{
		Prefix: prefix,
	})
	if err != nil {
		return 0, fmt.Errorf("failed to list log files: %w", err)
	}

	var deletedCount int64
	for _, obj := range objects.Objects {
		if !strings.HasSuffix(obj.Key, ".ndjson") {
			continue
		}

		// Check if file falls within time range (based on path)
		if !opts.EndTime.IsZero() {
			// Parse date from path (format: prefix/category/YYYY/MM/DD/...)
			parts := strings.Split(obj.Key, "/")
			if len(parts) >= 5 {
				dateStr := parts[2] + "/" + parts[3] + "/" + parts[4]
				fileDate, err := time.Parse("2006/01/02", dateStr)
				if err == nil && fileDate.After(opts.EndTime) {
					continue
				}
			}
		}

		if err := s.storage.Delete(ctx, s.bucket, obj.Key); err != nil {
			continue
		}
		deletedCount++
	}

	return deletedCount, nil
}

// Stats returns statistics about stored logs.
// Note: This is expensive for S3 as it requires listing and counting all files.
func (s *S3LogStorage) Stats(ctx context.Context) (*LogStats, error) {
	stats := &LogStats{
		EntriesByCategory: make(map[LogCategory]int64),
		EntriesByLevel:    make(map[LogLevel]int64),
	}

	// List all log files
	objects, err := s.storage.List(ctx, s.bucket, &ListOptions{
		Prefix: s.prefix + "/",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list log files: %w", err)
	}

	// Count entries per category from file paths
	for _, obj := range objects.Objects {
		if !strings.HasSuffix(obj.Key, ".ndjson") {
			continue
		}

		// Parse category from path
		parts := strings.Split(strings.TrimPrefix(obj.Key, s.prefix+"/"), "/")
		if len(parts) > 0 {
			category := LogCategory(parts[0])
			stats.EntriesByCategory[category]++
			stats.TotalEntries++
		}
	}

	return stats, nil
}

// Health checks if the backend is operational.
func (s *S3LogStorage) Health(ctx context.Context) error {
	return s.storage.Health(ctx)
}

// Close releases resources.
func (s *S3LogStorage) Close() error {
	// Don't close the shared storage provider
	return nil
}

// downloadAndParseFile downloads an NDJSON file and parses entries matching the filter.
func (s *S3LogStorage) downloadAndParseFile(ctx context.Context, key string, opts LogQueryOptions) ([]*LogEntry, error) {
	reader, _, err := s.storage.Download(ctx, s.bucket, key, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = reader.Close() }()

	var entries []*LogEntry
	scanner := bufio.NewScanner(reader)

	for scanner.Scan() {
		var entry LogEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			continue
		}

		// Apply filters
		if !s.matchesFilter(&entry, opts) {
			continue
		}

		entries = append(entries, &entry)
	}

	return entries, nil
}

// matchesFilter checks if an entry matches the query options.
func (s *S3LogStorage) matchesFilter(entry *LogEntry, opts LogQueryOptions) bool {
	if opts.Category != "" && entry.Category != opts.Category {
		return false
	}

	if len(opts.Levels) > 0 {
		found := false
		for _, level := range opts.Levels {
			if entry.Level == level {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	if opts.Component != "" && entry.Component != opts.Component {
		return false
	}

	if opts.RequestID != "" && entry.RequestID != opts.RequestID {
		return false
	}

	if opts.TraceID != "" && entry.TraceID != opts.TraceID {
		return false
	}

	if opts.UserID != "" && entry.UserID != opts.UserID {
		return false
	}

	if opts.ExecutionID != "" && entry.ExecutionID != opts.ExecutionID {
		return false
	}

	if !opts.StartTime.IsZero() && entry.Timestamp.Before(opts.StartTime) {
		return false
	}

	if !opts.EndTime.IsZero() && entry.Timestamp.After(opts.EndTime) {
		return false
	}

	if opts.Search != "" && !strings.Contains(strings.ToLower(entry.Message), strings.ToLower(opts.Search)) {
		return false
	}

	return true
}
