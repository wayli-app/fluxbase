package ai

import (
	"context"
	"net"
	"time"

	"github.com/fluxbase-eu/fluxbase/internal/database"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// AuditLogger logs AI query execution for compliance and debugging
type AuditLogger struct {
	db *database.Connection
}

// NewAuditLogger creates a new audit logger
func NewAuditLogger(db *database.Connection) *AuditLogger {
	return &AuditLogger{
		db: db,
	}
}

// AuditEntry represents a query audit log entry
type AuditEntry struct {
	ID                  string
	ChatbotID           *string
	ConversationID      *string
	MessageID           *string
	UserID              *string
	GeneratedSQL        string
	SanitizedSQL        *string
	Executed            bool
	ValidationPassed    *bool
	ValidationErrors    []string
	Success             *bool
	ErrorMessage        *string
	RowsReturned        *int
	ExecutionDurationMs *int
	TablesAccessed      []string
	OperationsUsed      []string
	RLSUserID           *string
	RLSRole             *string
	IPAddress           *net.IP
	UserAgent           *string
	CreatedAt           time.Time
}

// LogQuery logs a query execution to the audit table
func (l *AuditLogger) LogQuery(ctx context.Context, entry *AuditEntry) error {
	if entry.ID == "" {
		entry.ID = uuid.New().String()
	}
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = time.Now()
	}

	// Validate user_id exists in auth.users before inserting
	// Admin users are in dashboard.users, not auth.users, so we need to check
	validUserID := entry.UserID
	if validUserID != nil && *validUserID != "" {
		var exists bool
		err := l.db.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM auth.users WHERE id = $1)", *validUserID).Scan(&exists)
		if err != nil {
			log.Warn().Err(err).Str("user_id", *validUserID).Msg("Failed to check if user exists for audit log, setting user_id to NULL")
			validUserID = nil
		} else if !exists {
			log.Debug().Str("user_id", *validUserID).Msg("User not found in auth.users for audit log (likely admin user), setting user_id to NULL")
			validUserID = nil
		}
	}

	query := `
		INSERT INTO ai.query_audit_log (
			id, chatbot_id, conversation_id, message_id, user_id,
			generated_sql, sanitized_sql, executed,
			validation_passed, validation_errors,
			success, error_message, rows_returned, execution_duration_ms,
			tables_accessed, operations_used,
			rls_user_id, rls_role,
			ip_address, user_agent,
			created_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8,
			$9, $10,
			$11, $12, $13, $14,
			$15, $16,
			$17, $18,
			$19, $20,
			$21
		)
	`

	_, err := l.db.Exec(ctx, query,
		entry.ID, entry.ChatbotID, entry.ConversationID, entry.MessageID, validUserID,
		entry.GeneratedSQL, entry.SanitizedSQL, entry.Executed,
		entry.ValidationPassed, entry.ValidationErrors,
		entry.Success, entry.ErrorMessage, entry.RowsReturned, entry.ExecutionDurationMs,
		entry.TablesAccessed, entry.OperationsUsed,
		entry.RLSUserID, entry.RLSRole,
		entry.IPAddress, entry.UserAgent,
		entry.CreatedAt,
	)

	if err != nil {
		log.Error().Err(err).Str("id", entry.ID).Msg("Failed to log AI query to audit table")
		return err
	}

	log.Debug().
		Str("id", entry.ID).
		Bool("executed", entry.Executed).
		Msg("Logged AI query to audit table")

	return nil
}

// LogFromExecuteResult creates and logs an audit entry from an execute result
func (l *AuditLogger) LogFromExecuteResult(
	ctx context.Context,
	chatbotID, conversationID, messageID, userID string,
	generatedSQL string,
	result *ExecuteResult,
	rlsRole string,
	ipAddress, userAgent string,
) error {
	entry := &AuditEntry{
		GeneratedSQL:   generatedSQL,
		Executed:       result.Success || result.Error != "",
		TablesAccessed: result.TablesAccessed,
		OperationsUsed: result.OperationsUsed,
	}

	// Set nullable fields
	if chatbotID != "" {
		entry.ChatbotID = &chatbotID
	}
	if conversationID != "" {
		entry.ConversationID = &conversationID
	}
	if messageID != "" {
		entry.MessageID = &messageID
	}
	if userID != "" {
		entry.UserID = &userID
		entry.RLSUserID = &userID
	}
	if rlsRole != "" {
		entry.RLSRole = &rlsRole
	}
	if ipAddress != "" {
		ip := net.ParseIP(ipAddress)
		if ip != nil {
			entry.IPAddress = &ip
		}
	}
	if userAgent != "" {
		entry.UserAgent = &userAgent
	}

	// Set validation result
	if result.ValidationResult != nil {
		validPassed := result.ValidationResult.Valid
		entry.ValidationPassed = &validPassed
		entry.ValidationErrors = result.ValidationResult.Errors
		if result.ValidationResult.NormalizedQuery != "" {
			entry.SanitizedSQL = &result.ValidationResult.NormalizedQuery
		}
	}

	// Set execution result
	entry.Success = &result.Success
	if result.Error != "" {
		entry.ErrorMessage = &result.Error
	}
	if result.RowCount > 0 {
		entry.RowsReturned = &result.RowCount
	}
	if result.DurationMs > 0 {
		durationMs := int(result.DurationMs)
		entry.ExecutionDurationMs = &durationMs
	}

	return l.LogQuery(ctx, entry)
}

// GetRecentQueries retrieves recent audit log entries
func (l *AuditLogger) GetRecentQueries(ctx context.Context, limit int) ([]*AuditEntry, error) {
	query := `
		SELECT
			id, chatbot_id, conversation_id, message_id, user_id,
			generated_sql, sanitized_sql, executed,
			validation_passed, validation_errors,
			success, error_message, rows_returned, execution_duration_ms,
			tables_accessed, operations_used,
			rls_user_id, rls_role,
			ip_address, user_agent,
			created_at
		FROM ai.query_audit_log
		ORDER BY created_at DESC
		LIMIT $1
	`

	rows, err := l.db.Query(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []*AuditEntry
	for rows.Next() {
		entry := &AuditEntry{}
		err := rows.Scan(
			&entry.ID, &entry.ChatbotID, &entry.ConversationID, &entry.MessageID, &entry.UserID,
			&entry.GeneratedSQL, &entry.SanitizedSQL, &entry.Executed,
			&entry.ValidationPassed, &entry.ValidationErrors,
			&entry.Success, &entry.ErrorMessage, &entry.RowsReturned, &entry.ExecutionDurationMs,
			&entry.TablesAccessed, &entry.OperationsUsed,
			&entry.RLSUserID, &entry.RLSRole,
			&entry.IPAddress, &entry.UserAgent,
			&entry.CreatedAt,
		)
		if err != nil {
			log.Warn().Err(err).Msg("Failed to scan audit entry")
			continue
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

// GetQueriesByChatbot retrieves audit entries for a specific chatbot
func (l *AuditLogger) GetQueriesByChatbot(ctx context.Context, chatbotID string, limit int) ([]*AuditEntry, error) {
	query := `
		SELECT
			id, chatbot_id, conversation_id, message_id, user_id,
			generated_sql, sanitized_sql, executed,
			validation_passed, validation_errors,
			success, error_message, rows_returned, execution_duration_ms,
			tables_accessed, operations_used,
			rls_user_id, rls_role,
			ip_address, user_agent,
			created_at
		FROM ai.query_audit_log
		WHERE chatbot_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	rows, err := l.db.Query(ctx, query, chatbotID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []*AuditEntry
	for rows.Next() {
		entry := &AuditEntry{}
		err := rows.Scan(
			&entry.ID, &entry.ChatbotID, &entry.ConversationID, &entry.MessageID, &entry.UserID,
			&entry.GeneratedSQL, &entry.SanitizedSQL, &entry.Executed,
			&entry.ValidationPassed, &entry.ValidationErrors,
			&entry.Success, &entry.ErrorMessage, &entry.RowsReturned, &entry.ExecutionDurationMs,
			&entry.TablesAccessed, &entry.OperationsUsed,
			&entry.RLSUserID, &entry.RLSRole,
			&entry.IPAddress, &entry.UserAgent,
			&entry.CreatedAt,
		)
		if err != nil {
			log.Warn().Err(err).Msg("Failed to scan audit entry")
			continue
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

// GetFailedQueries retrieves failed queries for monitoring
func (l *AuditLogger) GetFailedQueries(ctx context.Context, since time.Time, limit int) ([]*AuditEntry, error) {
	query := `
		SELECT
			id, chatbot_id, conversation_id, message_id, user_id,
			generated_sql, sanitized_sql, executed,
			validation_passed, validation_errors,
			success, error_message, rows_returned, execution_duration_ms,
			tables_accessed, operations_used,
			rls_user_id, rls_role,
			ip_address, user_agent,
			created_at
		FROM ai.query_audit_log
		WHERE (success = false OR validation_passed = false)
		  AND created_at >= $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	rows, err := l.db.Query(ctx, query, since, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []*AuditEntry
	for rows.Next() {
		entry := &AuditEntry{}
		err := rows.Scan(
			&entry.ID, &entry.ChatbotID, &entry.ConversationID, &entry.MessageID, &entry.UserID,
			&entry.GeneratedSQL, &entry.SanitizedSQL, &entry.Executed,
			&entry.ValidationPassed, &entry.ValidationErrors,
			&entry.Success, &entry.ErrorMessage, &entry.RowsReturned, &entry.ExecutionDurationMs,
			&entry.TablesAccessed, &entry.OperationsUsed,
			&entry.RLSUserID, &entry.RLSRole,
			&entry.IPAddress, &entry.UserAgent,
			&entry.CreatedAt,
		)
		if err != nil {
			log.Warn().Err(err).Msg("Failed to scan audit entry")
			continue
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

// AuditStats represents aggregate audit statistics
type AuditStats struct {
	TotalQueries      int64   `json:"total_queries"`
	ExecutedQueries   int64   `json:"executed_queries"`
	FailedQueries     int64   `json:"failed_queries"`
	RejectedQueries   int64   `json:"rejected_queries"`
	AverageDurationMs float64 `json:"average_duration_ms"`
}

// GetStats retrieves aggregate statistics for a time period
func (l *AuditLogger) GetStats(ctx context.Context, since time.Time) (*AuditStats, error) {
	query := `
		SELECT
			COUNT(*) as total_queries,
			COUNT(*) FILTER (WHERE executed = true) as executed_queries,
			COUNT(*) FILTER (WHERE success = false) as failed_queries,
			COUNT(*) FILTER (WHERE validation_passed = false) as rejected_queries,
			COALESCE(AVG(execution_duration_ms) FILTER (WHERE executed = true), 0) as avg_duration_ms
		FROM ai.query_audit_log
		WHERE created_at >= $1
	`

	stats := &AuditStats{}
	err := l.db.QueryRow(ctx, query, since).Scan(
		&stats.TotalQueries,
		&stats.ExecutedQueries,
		&stats.FailedQueries,
		&stats.RejectedQueries,
		&stats.AverageDurationMs,
	)

	if err != nil {
		return nil, err
	}

	return stats, nil
}
