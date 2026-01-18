package logging

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/fluxbase-eu/fluxbase/internal/config"
	"github.com/fluxbase-eu/fluxbase/internal/database"
	"github.com/fluxbase-eu/fluxbase/internal/pubsub"
	"github.com/fluxbase-eu/fluxbase/internal/storage"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// Service is the central logging service that orchestrates log collection,
// batching, storage, and realtime notifications.
type Service struct {
	config       *config.LoggingConfig
	storage      storage.LogStorage
	batcher      *Batcher
	notifier     *PubSubNotifier
	writer       *Writer
	mu           sync.RWMutex
	closed       bool
	lineNumber   map[string]int       // Track line numbers per execution
	lineLastUsed map[string]time.Time // Track last access time for cleanup
	lineMu       sync.Mutex
}

// New creates a new logging service based on configuration.
func New(cfg *config.LoggingConfig, db *database.Connection, fileStorage storage.Provider, ps pubsub.PubSub) (*Service, error) {
	// Create the log storage service
	storageCfg := storage.LogStorageConfig{
		Backend:       cfg.Backend,
		S3Bucket:      cfg.S3Bucket,
		S3Prefix:      cfg.S3Prefix,
		LocalPath:     cfg.LocalPath,
		BatchSize:     cfg.BatchSize,
		FlushInterval: int(cfg.FlushInterval.Milliseconds()),
		BufferSize:    cfg.BufferSize,
	}

	logService, err := storage.NewLogService(storageCfg, db, fileStorage)
	if err != nil {
		return nil, err
	}

	s := &Service{
		config:       cfg,
		storage:      logService.Storage,
		lineNumber:   make(map[string]int),
		lineLastUsed: make(map[string]time.Time),
	}

	// Start background cleanup goroutine for stale line number entries
	go s.cleanupStaleLineNumbers()

	// Create PubSub notifier if enabled
	if cfg.PubSubEnabled && ps != nil {
		s.notifier = NewPubSubNotifier(ps, "fluxbase:logs")
	}

	// Create batcher
	batchSize := cfg.BatchSize
	if batchSize <= 0 {
		batchSize = 100
	}
	flushInterval := cfg.FlushInterval
	if flushInterval <= 0 {
		flushInterval = time.Second
	}
	bufferSize := cfg.BufferSize
	if bufferSize <= 0 {
		bufferSize = 10000
	}

	s.batcher = NewBatcher(batchSize, flushInterval, bufferSize, s.writeBatch)

	// Create zerolog writer
	s.writer = NewWriter(s, cfg.ConsoleEnabled, cfg.ConsoleFormat)

	log.Info().
		Str("backend", logService.GetBackendName()).
		Int("batch_size", batchSize).
		Dur("flush_interval", flushInterval).
		Bool("pubsub_enabled", cfg.PubSubEnabled).
		Msg("Central logging service initialized")

	return s, nil
}

// Writer returns the io.Writer for zerolog integration.
func (s *Service) Writer() *Writer {
	return s.writer
}

// Storage returns the underlying log storage.
func (s *Service) Storage() storage.LogStorage {
	return s.storage
}

// Log sends a log entry to the logging pipeline.
// The entry is batched and written asynchronously.
func (s *Service) Log(ctx context.Context, entry *storage.LogEntry) {
	s.mu.RLock()
	if s.closed {
		s.mu.RUnlock()
		return
	}
	s.mu.RUnlock()

	// Ensure ID and timestamp are set
	if entry.ID == uuid.Nil {
		entry.ID = uuid.New()
	}
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}

	// For execution logs, assign line numbers
	if entry.Category == storage.LogCategoryExecution && entry.ExecutionID != "" {
		entry.LineNumber = s.nextLineNumber(entry.ExecutionID)
	}

	// Add to batch
	s.batcher.Add(entry)

	// Send PubSub notification for all logs (for realtime streaming)
	if s.notifier != nil {
		go func(e *storage.LogEntry) {
			if err := s.notifier.Notify(context.Background(), e); err != nil {
				log.Debug().Err(err).Msg("Failed to send log notification")
			}
		}(entry)
	}
}

// LogSystem logs a system/application log entry.
func (s *Service) LogSystem(ctx context.Context, level storage.LogLevel, message string, fields map[string]any) {
	entry := &storage.LogEntry{
		Category:  storage.LogCategorySystem,
		Level:     level,
		Message:   message,
		Fields:    fields,
		Timestamp: time.Now(),
	}

	// Extract common fields
	if fields != nil {
		if v, ok := fields["component"].(string); ok {
			entry.Component = v
		}
		if v, ok := fields["request_id"].(string); ok {
			entry.RequestID = v
		}
		if v, ok := fields["trace_id"].(string); ok {
			entry.TraceID = v
		}
	}

	s.Log(ctx, entry)
}

// LogHTTP logs an HTTP access log entry.
func (s *Service) LogHTTP(ctx context.Context, fields *storage.HTTPLogFields, requestID, traceID, userID, ipAddress string) {
	fieldsMap := map[string]any{
		"method":         fields.Method,
		"path":           fields.Path,
		"status_code":    fields.StatusCode,
		"duration_ms":    fields.DurationMs,
		"response_bytes": fields.ResponseBytes,
	}
	if fields.Query != "" {
		fieldsMap["query"] = fields.Query
	}
	if fields.UserAgent != "" {
		fieldsMap["user_agent"] = fields.UserAgent
	}
	if fields.Referer != "" {
		fieldsMap["referer"] = fields.Referer
	}
	if fields.RequestBytes > 0 {
		fieldsMap["request_bytes"] = fields.RequestBytes
	}

	// Determine log level based on status code
	level := storage.LogLevelInfo
	if fields.StatusCode >= 500 {
		level = storage.LogLevelError
	} else if fields.StatusCode >= 400 {
		level = storage.LogLevelWarn
	}

	entry := &storage.LogEntry{
		Category:  storage.LogCategoryHTTP,
		Level:     level,
		Message:   fmt.Sprintf("%s %s → %d (%dms)", fields.Method, fields.Path, fields.StatusCode, fields.DurationMs),
		RequestID: requestID,
		TraceID:   traceID,
		UserID:    userID,
		IPAddress: ipAddress,
		Fields:    fieldsMap,
		Timestamp: time.Now(),
	}

	s.Log(ctx, entry)
}

// LogSecurity logs a security/audit log entry.
func (s *Service) LogSecurity(ctx context.Context, fields *storage.SecurityLogFields, requestID, userID, ipAddress string) {
	fieldsMap := map[string]any{
		"event_type": fields.EventType,
		"success":    fields.Success,
	}
	if fields.Email != "" {
		fieldsMap["email"] = fields.Email
	}
	if fields.TargetID != "" {
		fieldsMap["target_id"] = fields.TargetID
	}
	if fields.Action != "" {
		fieldsMap["action"] = fields.Action
	}
	for k, v := range fields.Details {
		fieldsMap[k] = v
	}

	// Determine log level
	level := storage.LogLevelInfo
	if !fields.Success {
		level = storage.LogLevelWarn
	}

	entry := &storage.LogEntry{
		Category:  storage.LogCategorySecurity,
		Level:     level,
		Message:   fields.EventType,
		RequestID: requestID,
		UserID:    userID,
		IPAddress: ipAddress,
		Fields:    fieldsMap,
		Timestamp: time.Now(),
	}

	s.Log(ctx, entry)
}

// LogExecution logs an execution log entry (function, job, or RPC).
func (s *Service) LogExecution(ctx context.Context, executionID, executionType string, level storage.LogLevel, message string, fields map[string]any) {
	if fields == nil {
		fields = make(map[string]any)
	}
	fields["execution_type"] = executionType

	entry := &storage.LogEntry{
		Category:      storage.LogCategoryExecution,
		Level:         level,
		Message:       message,
		ExecutionID:   executionID,
		ExecutionType: executionType,
		Fields:        fields,
		Timestamp:     time.Now(),
	}

	// Extract common fields
	if v, ok := fields["request_id"].(string); ok {
		entry.RequestID = v
	}
	if v, ok := fields["user_id"].(string); ok {
		entry.UserID = v
	}

	s.Log(ctx, entry)
}

// LogAI logs an AI query audit log entry.
func (s *Service) LogAI(ctx context.Context, fields map[string]any, requestID, userID string) {
	entry := &storage.LogEntry{
		Category:  storage.LogCategoryAI,
		Level:     storage.LogLevelInfo,
		Message:   "AI query",
		RequestID: requestID,
		UserID:    userID,
		Fields:    fields,
		Timestamp: time.Now(),
	}

	s.Log(ctx, entry)
}

// LogCustom logs an entry with a user-defined custom category.
func (s *Service) LogCustom(ctx context.Context, customCategory string, level storage.LogLevel, message string, fields map[string]any) {
	// Validate custom category if configured
	if len(s.config.CustomCategories) > 0 && !s.IsValidCustomCategory(customCategory) {
		log.Warn().
			Str("category", customCategory).
			Msg("Invalid custom log category - logging with category 'custom'")
		customCategory = "unknown"
	}

	entry := &storage.LogEntry{
		Category:       storage.LogCategoryCustom,
		CustomCategory: customCategory,
		Level:          level,
		Message:        message,
		Fields:         fields,
		Timestamp:      time.Now(),
	}

	// Extract common fields
	if fields != nil {
		if v, ok := fields["component"].(string); ok {
			entry.Component = v
		}
		if v, ok := fields["request_id"].(string); ok {
			entry.RequestID = v
		}
		if v, ok := fields["trace_id"].(string); ok {
			entry.TraceID = v
		}
		if v, ok := fields["user_id"].(string); ok {
			entry.UserID = v
		}
	}

	s.Log(ctx, entry)
}

// IsValidCustomCategory checks if a custom category name is allowed.
func (s *Service) IsValidCustomCategory(category string) bool {
	if len(s.config.CustomCategories) == 0 {
		// If no custom categories are configured, allow any
		return true
	}
	for _, allowed := range s.config.CustomCategories {
		if allowed == category {
			return true
		}
	}
	return false
}

// GetCustomCategories returns the list of configured custom categories.
func (s *Service) GetCustomCategories() []string {
	return s.config.CustomCategories
}

// Query retrieves logs matching the given options.
func (s *Service) Query(ctx context.Context, opts storage.LogQueryOptions) (*storage.LogQueryResult, error) {
	return s.storage.Query(ctx, opts)
}

// GetExecutionLogs retrieves logs for a specific execution.
func (s *Service) GetExecutionLogs(ctx context.Context, executionID string, afterLine int) ([]*storage.LogEntry, error) {
	return s.storage.GetExecutionLogs(ctx, executionID, afterLine)
}

// Stats returns statistics about stored logs.
func (s *Service) Stats(ctx context.Context) (*storage.LogStats, error) {
	return s.storage.Stats(ctx)
}

// Flush forces a flush of any buffered log entries.
func (s *Service) Flush(ctx context.Context) error {
	return s.batcher.Flush(ctx)
}

// Close shuts down the logging service gracefully.
func (s *Service) Close() error {
	s.mu.Lock()
	s.closed = true
	s.mu.Unlock()

	// Flush remaining entries
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.batcher.Close(ctx); err != nil {
		log.Warn().Err(err).Msg("Error closing log batcher")
	}

	return s.storage.Close()
}

// writeBatch is called by the batcher to write a batch of entries.
func (s *Service) writeBatch(ctx context.Context, entries []*storage.LogEntry) error {
	if len(entries) == 0 {
		return nil
	}

	if err := s.storage.Write(ctx, entries); err != nil {
		log.Error().Err(err).Int("count", len(entries)).Msg("Failed to write log batch")
		return err
	}

	// log.Debug().Int("count", len(entries)).Msg("Wrote log batch")
	return nil
}

// nextLineNumber returns the next line number for an execution.
func (s *Service) nextLineNumber(executionID string) int {
	s.lineMu.Lock()
	defer s.lineMu.Unlock()

	s.lineNumber[executionID]++
	s.lineLastUsed[executionID] = time.Now()
	return s.lineNumber[executionID]
}

// ClearLineNumbers clears the line number counter for an execution.
// Call this when an execution completes.
func (s *Service) ClearLineNumbers(executionID string) {
	s.lineMu.Lock()
	defer s.lineMu.Unlock()

	delete(s.lineNumber, executionID)
	delete(s.lineLastUsed, executionID)
}

// cleanupStaleLineNumbers periodically removes stale line number entries
// to prevent memory leaks from executions that never called ClearLineNumbers.
func (s *Service) cleanupStaleLineNumbers() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.RLock()
		if s.closed {
			s.mu.RUnlock()
			return
		}
		s.mu.RUnlock()

		s.lineMu.Lock()
		now := time.Now()
		staleThreshold := 30 * time.Minute
		for execID, lastUsed := range s.lineLastUsed {
			if now.Sub(lastUsed) > staleThreshold {
				delete(s.lineNumber, execID)
				delete(s.lineLastUsed, execID)
			}
		}
		s.lineMu.Unlock()
	}
}

// GetRetentionPolicy returns the retention days for a category.
func (s *Service) GetRetentionPolicy(category storage.LogCategory) int {
	switch category {
	case storage.LogCategorySystem:
		return s.config.SystemRetentionDays
	case storage.LogCategoryHTTP:
		return s.config.HTTPRetentionDays
	case storage.LogCategorySecurity:
		return s.config.SecurityRetentionDays
	case storage.LogCategoryExecution:
		return s.config.ExecutionRetentionDays
	case storage.LogCategoryAI:
		return s.config.AIRetentionDays
	case storage.LogCategoryCustom:
		return s.config.CustomRetentionDays
	default:
		return 30 // Default retention
	}
}
