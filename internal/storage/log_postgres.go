package storage

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/fluxbase-eu/fluxbase/internal/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// staticAssetExtensions contains file extensions to filter out when HideStaticAssets is enabled.
var staticAssetExtensions = []string{
	// Scripts
	".js", ".mjs", ".ts", ".jsx", ".tsx",
	// Styles
	".css",
	// Images
	".png", ".jpg", ".jpeg", ".gif", ".svg", ".webp", ".ico", ".avif",
	// Fonts
	".woff", ".woff2", ".ttf", ".eot", ".otf",
	// Source maps
	".map",
}

// PostgresLogStorage implements LogStorage using PostgreSQL.
type PostgresLogStorage struct {
	db *database.Connection
}

// NewPostgresLogStorage creates a new PostgreSQL-backed log storage.
func NewPostgresLogStorage(db *database.Connection) *PostgresLogStorage {
	return &PostgresLogStorage{db: db}
}

// Name returns the backend identifier.
func (s *PostgresLogStorage) Name() string {
	return "postgres"
}

// Write writes a batch of log entries to PostgreSQL.
func (s *PostgresLogStorage) Write(ctx context.Context, entries []*LogEntry) error {
	if len(entries) == 0 {
		return nil
	}

	// Build batch insert query
	// Using COPY would be faster for very large batches, but for typical log batches
	// (100-1000 entries), a multi-value INSERT is efficient enough
	query := `
		INSERT INTO logging.entries (
			id, timestamp, category, level, message, custom_category,
			request_id, trace_id, component, user_id, ip_address,
			fields, execution_id, line_number
		) VALUES `

	values := make([]string, 0, len(entries))
	args := make([]any, 0, len(entries)*14)

	for i, entry := range entries {
		base := i * 14
		values = append(values, fmt.Sprintf(
			"($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d)",
			base+1, base+2, base+3, base+4, base+5, base+6, base+7, base+8, base+9, base+10, base+11, base+12, base+13, base+14,
		))

		// Ensure ID is set
		if entry.ID == uuid.Nil {
			entry.ID = uuid.New()
		}

		// Ensure timestamp is set
		if entry.Timestamp.IsZero() {
			entry.Timestamp = time.Now()
		}

		// Convert user ID to UUID if present
		var userID *uuid.UUID
		if entry.UserID != "" {
			if parsed, err := uuid.Parse(entry.UserID); err == nil {
				userID = &parsed
			}
		}

		// Convert execution ID to UUID if present
		var executionID *uuid.UUID
		if entry.ExecutionID != "" {
			if parsed, err := uuid.Parse(entry.ExecutionID); err == nil {
				executionID = &parsed
			}
		}

		// Convert line number (0 means not set)
		var lineNumber *int
		if entry.LineNumber > 0 {
			lineNumber = &entry.LineNumber
		}

		args = append(args,
			entry.ID,
			entry.Timestamp,
			string(entry.Category),
			string(entry.Level),
			entry.Message,
			nullableString(entry.CustomCategory),
			nullableString(entry.RequestID),
			nullableString(entry.TraceID),
			nullableString(entry.Component),
			userID,
			nullableString(entry.IPAddress),
			entry.Fields,
			executionID,
			lineNumber,
		)
	}

	query += strings.Join(values, ", ")

	_, err := s.db.Pool().Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to insert log entries: %w", err)
	}

	return nil
}

// Query retrieves logs matching the given options.
func (s *PostgresLogStorage) Query(ctx context.Context, opts LogQueryOptions) (*LogQueryResult, error) {
	// Build WHERE clauses
	where, args := s.buildWhereClause(opts)

	// Count total matching entries
	countQuery := "SELECT COUNT(*) FROM logging.entries" + where
	var totalCount int64
	if err := s.db.Pool().QueryRow(ctx, countQuery, args...).Scan(&totalCount); err != nil {
		return nil, fmt.Errorf("failed to count log entries: %w", err)
	}

	// Build ORDER BY
	orderBy := " ORDER BY timestamp DESC"
	if opts.SortAsc {
		orderBy = " ORDER BY timestamp ASC"
	}

	// Build LIMIT/OFFSET
	limit := opts.Limit
	if limit <= 0 {
		limit = 100 // Default limit
	}
	offset := opts.Offset
	if offset < 0 {
		offset = 0
	}

	// Fetch entries
	selectQuery := `
		SELECT id, timestamp, category, level, message, custom_category,
		       request_id, trace_id, component, user_id, ip_address::text,
		       fields, execution_id, line_number
		FROM logging.entries` + where + orderBy + fmt.Sprintf(" LIMIT %d OFFSET %d", limit, offset)

	rows, err := s.db.Pool().Query(ctx, selectQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query log entries: %w", err)
	}
	defer rows.Close()

	entries, err := s.scanEntries(rows)
	if err != nil {
		return nil, err
	}

	return &LogQueryResult{
		Entries:    entries,
		TotalCount: totalCount,
		HasMore:    int64(offset+len(entries)) < totalCount,
	}, nil
}

// GetExecutionLogs retrieves logs for a specific execution.
func (s *PostgresLogStorage) GetExecutionLogs(ctx context.Context, executionID string, afterLine int) ([]*LogEntry, error) {
	execUUID, err := uuid.Parse(executionID)
	if err != nil {
		return nil, fmt.Errorf("invalid execution ID: %w", err)
	}

	query := `
		SELECT id, timestamp, category, level, message, custom_category,
		       request_id, trace_id, component, user_id, ip_address::text,
		       fields, execution_id, line_number
		FROM logging.entries
		WHERE execution_id = $1 AND line_number > $2
		ORDER BY line_number ASC`

	rows, err := s.db.Pool().Query(ctx, query, execUUID, afterLine)
	if err != nil {
		return nil, fmt.Errorf("failed to query execution logs: %w", err)
	}
	defer rows.Close()

	return s.scanEntries(rows)
}

// Delete removes logs matching the given options.
func (s *PostgresLogStorage) Delete(ctx context.Context, opts LogQueryOptions) (int64, error) {
	where, args := s.buildWhereClause(opts)
	if where == "" {
		return 0, fmt.Errorf("delete requires at least one filter condition")
	}

	query := "DELETE FROM logging.entries" + where

	result, err := s.db.Pool().Exec(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("failed to delete log entries: %w", err)
	}

	return result.RowsAffected(), nil
}

// Stats returns statistics about stored logs.
func (s *PostgresLogStorage) Stats(ctx context.Context) (*LogStats, error) {
	stats := &LogStats{
		EntriesByCategory: make(map[LogCategory]int64),
		EntriesByLevel:    make(map[LogLevel]int64),
	}

	// Get total count and time range
	err := s.db.Pool().QueryRow(ctx, `
		SELECT COUNT(*), MIN(timestamp), MAX(timestamp)
		FROM logging.entries
	`).Scan(&stats.TotalEntries, &stats.OldestEntry, &stats.NewestEntry)
	if err != nil {
		return nil, fmt.Errorf("failed to get log stats: %w", err)
	}

	// Get counts by category
	rows, err := s.db.Pool().Query(ctx, `
		SELECT category, COUNT(*) FROM logging.entries GROUP BY category
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to get category counts: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var category string
		var count int64
		if err := rows.Scan(&category, &count); err != nil {
			return nil, fmt.Errorf("failed to scan category count: %w", err)
		}
		stats.EntriesByCategory[LogCategory(category)] = count
	}

	// Get counts by level
	rows, err = s.db.Pool().Query(ctx, `
		SELECT level, COUNT(*) FROM logging.entries GROUP BY level
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to get level counts: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var level string
		var count int64
		if err := rows.Scan(&level, &count); err != nil {
			return nil, fmt.Errorf("failed to scan level count: %w", err)
		}
		stats.EntriesByLevel[LogLevel(level)] = count
	}

	return stats, nil
}

// Health checks if the backend is operational.
func (s *PostgresLogStorage) Health(ctx context.Context) error {
	return s.db.Pool().Ping(ctx)
}

// Close releases resources (no-op for PostgreSQL as we share the connection pool).
func (s *PostgresLogStorage) Close() error {
	// Don't close the shared database connection
	return nil
}

// buildWhereClause builds a WHERE clause from query options.
func (s *PostgresLogStorage) buildWhereClause(opts LogQueryOptions) (string, []any) {
	var conditions []string
	var args []any
	argNum := 1

	if opts.Category != "" {
		conditions = append(conditions, fmt.Sprintf("category = $%d", argNum))
		args = append(args, string(opts.Category))
		argNum++
	}

	if opts.CustomCategory != "" {
		conditions = append(conditions, fmt.Sprintf("custom_category = $%d", argNum))
		args = append(args, opts.CustomCategory)
		argNum++
	}

	if len(opts.Levels) > 0 {
		placeholders := make([]string, len(opts.Levels))
		for i, level := range opts.Levels {
			placeholders[i] = fmt.Sprintf("$%d", argNum)
			args = append(args, string(level))
			argNum++
		}
		conditions = append(conditions, fmt.Sprintf("level IN (%s)", strings.Join(placeholders, ", ")))
	}

	if opts.Component != "" {
		conditions = append(conditions, fmt.Sprintf("component = $%d", argNum))
		args = append(args, opts.Component)
		argNum++
	}

	if opts.RequestID != "" {
		conditions = append(conditions, fmt.Sprintf("request_id = $%d", argNum))
		args = append(args, opts.RequestID)
		argNum++
	}

	if opts.TraceID != "" {
		conditions = append(conditions, fmt.Sprintf("trace_id = $%d", argNum))
		args = append(args, opts.TraceID)
		argNum++
	}

	if opts.UserID != "" {
		if userUUID, err := uuid.Parse(opts.UserID); err == nil {
			conditions = append(conditions, fmt.Sprintf("user_id = $%d", argNum))
			args = append(args, userUUID)
			argNum++
		}
	}

	if opts.ExecutionID != "" {
		if execUUID, err := uuid.Parse(opts.ExecutionID); err == nil {
			conditions = append(conditions, fmt.Sprintf("execution_id = $%d", argNum))
			args = append(args, execUUID)
			argNum++
		}
	}

	if opts.ExecutionType != "" {
		// ExecutionType is stored in the fields JSONB
		conditions = append(conditions, fmt.Sprintf("fields->>'execution_type' = $%d", argNum))
		args = append(args, opts.ExecutionType)
		argNum++
	}

	if !opts.StartTime.IsZero() {
		conditions = append(conditions, fmt.Sprintf("timestamp >= $%d", argNum))
		args = append(args, opts.StartTime)
		argNum++
	}

	if !opts.EndTime.IsZero() {
		conditions = append(conditions, fmt.Sprintf("timestamp <= $%d", argNum))
		args = append(args, opts.EndTime)
		argNum++
	}

	if opts.Search != "" {
		conditions = append(conditions, fmt.Sprintf("to_tsvector('english', message) @@ plainto_tsquery('english', $%d)", argNum))
		args = append(args, opts.Search)
		argNum++
	}

	if opts.AfterLine > 0 {
		conditions = append(conditions, fmt.Sprintf("line_number > $%d", argNum))
		args = append(args, opts.AfterLine)
		argNum++
	}

	if opts.HideStaticAssets {
		// Exclude HTTP logs where the path ends with a static asset extension
		// This uses a NOT condition with multiple LIKE patterns on the JSONB path field
		var excludePatterns []string
		for _, ext := range staticAssetExtensions {
			excludePatterns = append(excludePatterns, fmt.Sprintf("fields->>'path' ILIKE $%d", argNum))
			args = append(args, "%"+ext)
			argNum++
		}
		// Only apply to HTTP category, or exclude matching HTTP logs from all categories
		conditions = append(conditions,
			fmt.Sprintf("(category != 'http' OR NOT (%s))", strings.Join(excludePatterns, " OR ")))
	}

	if len(conditions) == 0 {
		return "", nil
	}

	return " WHERE " + strings.Join(conditions, " AND "), args
}

// scanEntries scans rows into LogEntry structs.
func (s *PostgresLogStorage) scanEntries(rows pgx.Rows) ([]*LogEntry, error) {
	var entries []*LogEntry

	for rows.Next() {
		var entry LogEntry
		var customCategory, requestID, traceID, component, ipAddress *string
		var userID, executionID *uuid.UUID
		var lineNumber *int

		err := rows.Scan(
			&entry.ID,
			&entry.Timestamp,
			&entry.Category,
			&entry.Level,
			&entry.Message,
			&customCategory,
			&requestID,
			&traceID,
			&component,
			&userID,
			&ipAddress,
			&entry.Fields,
			&executionID,
			&lineNumber,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan log entry: %w", err)
		}

		if customCategory != nil {
			entry.CustomCategory = *customCategory
		}
		if requestID != nil {
			entry.RequestID = *requestID
		}
		if traceID != nil {
			entry.TraceID = *traceID
		}
		if component != nil {
			entry.Component = *component
		}
		if userID != nil {
			entry.UserID = userID.String()
		}
		if ipAddress != nil {
			entry.IPAddress = *ipAddress
		}
		if executionID != nil {
			entry.ExecutionID = executionID.String()
		}
		if lineNumber != nil {
			entry.LineNumber = *lineNumber
		}

		entries = append(entries, &entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating log entries: %w", err)
	}

	return entries, nil
}

// nullableString returns a pointer to the string if not empty, nil otherwise.
func nullableString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
