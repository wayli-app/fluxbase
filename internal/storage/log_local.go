package storage

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// LocalLogStorage implements LogStorage using the local filesystem.
// It stores logs as NDJSON (newline-delimited JSON) files organized by date and category.
// Structure: {basePath}/{category}/{YYYY}/{MM}/{DD}/{execution_id|batch_uuid}.ndjson
type LocalLogStorage struct {
	basePath string
	mu       sync.RWMutex // Protects file operations for execution logs
}

// NewLocalLogStorage creates a new local filesystem-backed log storage.
func NewLocalLogStorage(basePath string) (*LocalLogStorage, error) {
	if basePath == "" {
		basePath = "./logs"
	}

	// Ensure base path exists
	if err := os.MkdirAll(basePath, 0750); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	return &LocalLogStorage{
		basePath: basePath,
	}, nil
}

// Name returns the backend identifier.
func (s *LocalLogStorage) Name() string {
	return "local"
}

// Write writes a batch of log entries to the local filesystem.
// Entries are grouped by category and date, then written as NDJSON files.
func (s *LocalLogStorage) Write(ctx context.Context, entries []*LogEntry) error {
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
		var filePath string
		if strings.HasSuffix(groupKey, "/batch") {
			// For batch logs, add a UUID to make the filename unique
			filePath = filepath.Join(s.basePath, groupKey+"_"+uuid.New().String()[:8]+".ndjson")
		} else {
			// For execution logs, use the execution ID as the filename
			filePath = filepath.Join(s.basePath, groupKey+".ndjson")
		}

		if err := s.writeEntries(filePath, groupEntries, strings.Contains(groupKey, "exec_")); err != nil {
			return err
		}
	}

	return nil
}

// writeEntries writes entries to a file, optionally appending for execution logs.
func (s *LocalLogStorage) writeEntries(filePath string, entries []*LogEntry, append bool) error {
	// Ensure directory exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	// For execution logs, we need to lock to prevent concurrent writes
	if append {
		s.mu.Lock()
		defer s.mu.Unlock()
	}

	// Open file
	var flags int
	if append {
		flags = os.O_APPEND | os.O_CREATE | os.O_WRONLY
	} else {
		flags = os.O_CREATE | os.O_WRONLY | os.O_TRUNC
	}

	f, err := os.OpenFile(filePath, flags, 0600) //nolint:gosec // File path is constructed from trusted prefix
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer func() { _ = f.Close() }()

	// Write entries as NDJSON
	enc := json.NewEncoder(f)
	for _, entry := range entries {
		if err := enc.Encode(entry); err != nil {
			return fmt.Errorf("failed to encode log entry: %w", err)
		}
	}

	return nil
}

// Query retrieves logs matching the given options.
// Note: Local filesystem is not optimized for querying - this requires scanning files.
// For heavy querying, use PostgreSQL backend instead.
func (s *LocalLogStorage) Query(ctx context.Context, opts LogQueryOptions) (*LogQueryResult, error) {
	// Build directory path for scanning
	searchPath := s.basePath
	if opts.Category != "" {
		searchPath = filepath.Join(searchPath, string(opts.Category))
	}

	// If we have time range, narrow down the path
	if !opts.StartTime.IsZero() {
		searchPath = filepath.Join(searchPath, opts.StartTime.Format("2006/01"))
	}

	// Find all NDJSON files
	var files []string
	err := filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip inaccessible paths
		}
		if !info.IsDir() && strings.HasSuffix(path, ".ndjson") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to scan log directory: %w", err)
	}

	// Read and filter entries from each file
	var allEntries []*LogEntry
	for _, file := range files {
		entries, err := s.readAndFilterFile(file, opts)
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
func (s *LocalLogStorage) GetExecutionLogs(ctx context.Context, executionID string, afterLine int) ([]*LogEntry, error) {
	// Search for the execution log file
	searchPath := filepath.Join(s.basePath, string(LogCategoryExecution))

	var targetFile string
	err := filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && strings.Contains(path, "exec_"+executionID) {
			targetFile = path
			return filepath.SkipAll // Found it, stop walking
		}
		return nil
	})
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to search for execution log: %w", err)
	}

	if targetFile == "" {
		return []*LogEntry{}, nil
	}

	// Read the file
	s.mu.RLock()
	defer s.mu.RUnlock()

	f, err := os.Open(targetFile) //nolint:gosec // File path is constructed from trusted prefix
	if err != nil {
		return nil, fmt.Errorf("failed to open execution log file: %w", err)
	}
	defer func() { _ = f.Close() }()

	var entries []*LogEntry
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var entry LogEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			continue
		}
		if entry.LineNumber > afterLine {
			entries = append(entries, &entry)
		}
	}

	// Sort by line number
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].LineNumber < entries[j].LineNumber
	})

	return entries, nil
}

// Delete removes logs matching the given options.
func (s *LocalLogStorage) Delete(ctx context.Context, opts LogQueryOptions) (int64, error) {
	// Build directory path for scanning
	searchPath := s.basePath
	if opts.Category != "" {
		searchPath = filepath.Join(searchPath, string(opts.Category))
	}

	// Find all NDJSON files
	var deletedCount int64
	err := filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() || !strings.HasSuffix(path, ".ndjson") {
			return nil
		}

		// Check if file falls within time range (based on path)
		if !opts.EndTime.IsZero() {
			// Parse date from path (format: basePath/category/YYYY/MM/DD/...)
			relPath, _ := filepath.Rel(s.basePath, path)
			parts := strings.Split(relPath, string(filepath.Separator))
			if len(parts) >= 4 {
				dateStr := parts[1] + "/" + parts[2] + "/" + parts[3]
				fileDate, err := time.Parse("2006/01/02", dateStr)
				if err == nil && fileDate.After(opts.EndTime) {
					return nil
				}
			}
		}

		if err := os.Remove(path); err == nil {
			deletedCount++
		}
		return nil
	})

	if err != nil && !os.IsNotExist(err) {
		return deletedCount, fmt.Errorf("failed to scan log directory: %w", err)
	}

	// Clean up empty directories
	s.cleanEmptyDirs(s.basePath)

	return deletedCount, nil
}

// cleanEmptyDirs removes empty directories recursively.
func (s *LocalLogStorage) cleanEmptyDirs(path string) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			subPath := filepath.Join(path, entry.Name())
			s.cleanEmptyDirs(subPath)
		}
	}

	// Try to remove the directory (will fail if not empty)
	if path != s.basePath {
		_ = os.Remove(path)
	}
}

// Stats returns statistics about stored logs.
func (s *LocalLogStorage) Stats(ctx context.Context) (*LogStats, error) {
	stats := &LogStats{
		EntriesByCategory: make(map[LogCategory]int64),
		EntriesByLevel:    make(map[LogLevel]int64),
	}

	err := filepath.Walk(s.basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() || !strings.HasSuffix(path, ".ndjson") {
			return nil
		}

		// Parse category from path
		relPath, _ := filepath.Rel(s.basePath, path)
		parts := strings.Split(relPath, string(filepath.Separator))
		if len(parts) > 0 {
			category := LogCategory(parts[0])
			stats.EntriesByCategory[category]++
			stats.TotalEntries++
		}

		return nil
	})

	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to scan log directory: %w", err)
	}

	return stats, nil
}

// Health checks if the backend is operational.
func (s *LocalLogStorage) Health(ctx context.Context) error {
	// Check if base path is accessible
	_, err := os.Stat(s.basePath)
	if os.IsNotExist(err) {
		// Try to create it
		if err := os.MkdirAll(s.basePath, 0750); err != nil {
			return fmt.Errorf("log directory not accessible: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("log directory not accessible: %w", err)
	}

	// Try to write a test file
	testFile := filepath.Join(s.basePath, ".health_check")
	if err := os.WriteFile(testFile, []byte("ok"), 0600); err != nil {
		return fmt.Errorf("cannot write to log directory: %w", err)
	}
	_ = os.Remove(testFile)

	return nil
}

// Close releases resources.
func (s *LocalLogStorage) Close() error {
	return nil
}

// readAndFilterFile reads an NDJSON file and returns entries matching the filter.
func (s *LocalLogStorage) readAndFilterFile(filePath string, opts LogQueryOptions) ([]*LogEntry, error) {
	f, err := os.Open(filePath) //nolint:gosec // File path is constructed from trusted prefix
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	var entries []*LogEntry
	scanner := bufio.NewScanner(f)

	// Increase buffer size for potentially large log lines
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

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
func (s *LocalLogStorage) matchesFilter(entry *LogEntry, opts LogQueryOptions) bool {
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

// StreamExecutionLogs returns a channel that streams log entries as they're written.
// This uses file watching to detect new entries.
func (s *LocalLogStorage) StreamExecutionLogs(ctx context.Context, executionID string) (<-chan *LogEntry, error) {
	ch := make(chan *LogEntry, 100)

	go func() {
		defer close(ch)

		var lastLine int
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				entries, err := s.GetExecutionLogs(ctx, executionID, lastLine)
				if err != nil {
					continue
				}

				for _, entry := range entries {
					select {
					case ch <- entry:
						if entry.LineNumber > lastLine {
							lastLine = entry.LineNumber
						}
					case <-ctx.Done():
						return
					}
				}
			}
		}
	}()

	return ch, nil
}

// Compile-time check that LocalLogStorage implements LogStorage
var _ LogStorage = (*LocalLogStorage)(nil)
