package storage

import (
	"time"

	"github.com/google/uuid"
)

// LogCategory identifies the type of log entry.
type LogCategory string

const (
	// LogCategorySystem represents application/system logs (zerolog output).
	LogCategorySystem LogCategory = "system"
	// LogCategoryHTTP represents HTTP access logs.
	LogCategoryHTTP LogCategory = "http"
	// LogCategorySecurity represents authentication and audit events.
	LogCategorySecurity LogCategory = "security"
	// LogCategoryExecution represents function/job/RPC execution logs.
	LogCategoryExecution LogCategory = "execution"
	// LogCategoryAI represents AI query audit logs.
	LogCategoryAI LogCategory = "ai"
	// LogCategoryCustom represents user-defined log categories.
	// Custom category name is stored in the CustomCategory field.
	LogCategoryCustom LogCategory = "custom"
)

// IsBuiltinCategory returns true if the category is a built-in category.
func IsBuiltinCategory(cat LogCategory) bool {
	switch cat {
	case LogCategorySystem, LogCategoryHTTP, LogCategorySecurity,
		LogCategoryExecution, LogCategoryAI:
		return true
	}
	return false
}

// AllBuiltinCategories returns all built-in log categories.
func AllBuiltinCategories() []LogCategory {
	return []LogCategory{
		LogCategorySystem,
		LogCategoryHTTP,
		LogCategorySecurity,
		LogCategoryExecution,
		LogCategoryAI,
	}
}

// LogLevel represents the severity level of a log entry.
type LogLevel string

const (
	LogLevelTrace LogLevel = "trace"
	LogLevelDebug LogLevel = "debug"
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
	LogLevelFatal LogLevel = "fatal"
	LogLevelPanic LogLevel = "panic"
)

// LogEntry represents a unified log entry across all categories.
type LogEntry struct {
	// Core fields (all categories)
	ID        uuid.UUID   `json:"id"`
	Timestamp time.Time   `json:"timestamp"`
	Category  LogCategory `json:"category"`
	Level     LogLevel    `json:"level"`
	Message   string      `json:"message"`

	// Custom category name (only used when Category is LogCategoryCustom)
	CustomCategory string `json:"custom_category,omitempty"`

	// Correlation fields
	RequestID string `json:"request_id,omitempty"`
	TraceID   string `json:"trace_id,omitempty"`

	// Context fields
	Component string `json:"component,omitempty"` // e.g., "auth", "functions", "api"
	UserID    string `json:"user_id,omitempty"`
	IPAddress string `json:"ip_address,omitempty"`

	// Structured data (category-specific fields stored as JSON)
	Fields map[string]any `json:"fields,omitempty"`

	// Execution-specific fields (for streaming logs)
	ExecutionID   string `json:"execution_id,omitempty"`
	ExecutionType string `json:"execution_type,omitempty"` // "function", "job", "rpc"
	LineNumber    int    `json:"line_number,omitempty"`
}

// HTTPLogFields contains HTTP-specific log fields.
type HTTPLogFields struct {
	Method        string `json:"method"`
	Path          string `json:"path"`
	Query         string `json:"query,omitempty"`
	StatusCode    int    `json:"status_code"`
	DurationMs    int64  `json:"duration_ms"`
	UserAgent     string `json:"user_agent,omitempty"`
	Referer       string `json:"referer,omitempty"`
	ResponseBytes int    `json:"response_bytes,omitempty"`
	RequestBytes  int    `json:"request_bytes,omitempty"`
}

// SecurityLogFields contains security-specific log fields.
type SecurityLogFields struct {
	EventType string         `json:"event_type"` // login_success, login_failed, token_refresh, etc.
	Success   bool           `json:"success"`
	Email     string         `json:"email,omitempty"`
	TargetID  string         `json:"target_id,omitempty"` // Target user/resource ID
	Action    string         `json:"action,omitempty"`    // create, update, delete, etc.
	Details   map[string]any `json:"details,omitempty"`
}

// ExecutionLogFields contains execution-specific log fields.
type ExecutionLogFields struct {
	ExecutionType string `json:"execution_type"` // "function", "job", "rpc"
	FunctionName  string `json:"function_name,omitempty"`
	Namespace     string `json:"namespace,omitempty"`
	JobType       string `json:"job_type,omitempty"`
	Status        string `json:"status,omitempty"`
	DurationMs    int64  `json:"duration_ms,omitempty"`
}

// LogQueryOptions defines filters for log queries.
type LogQueryOptions struct {
	// Filter by category
	Category LogCategory `json:"category,omitempty"`

	// Filter by custom category name (only used when Category is "custom")
	CustomCategory string `json:"custom_category,omitempty"`

	// Filter by levels (multiple allowed)
	Levels []LogLevel `json:"levels,omitempty"`

	// Filter by component
	Component string `json:"component,omitempty"`

	// Filter by correlation IDs
	RequestID string `json:"request_id,omitempty"`
	TraceID   string `json:"trace_id,omitempty"`

	// Filter by user
	UserID string `json:"user_id,omitempty"`

	// Filter by execution
	ExecutionID   string `json:"execution_id,omitempty"`
	ExecutionType string `json:"execution_type,omitempty"`

	// Time range
	StartTime time.Time `json:"start_time,omitempty"`
	EndTime   time.Time `json:"end_time,omitempty"`

	// Full-text search in message
	Search string `json:"search,omitempty"`

	// Pagination
	Limit  int `json:"limit,omitempty"`
	Offset int `json:"offset,omitempty"`

	// For execution log streaming (get logs after this line number)
	AfterLine int `json:"after_line,omitempty"`

	// Sort order (default: descending by timestamp)
	SortAsc bool `json:"sort_asc,omitempty"`
}

// LogQueryResult contains the result of a log query.
type LogQueryResult struct {
	Entries    []*LogEntry `json:"entries"`
	TotalCount int64       `json:"total_count"`
	HasMore    bool        `json:"has_more"`
}

// LogRetentionPolicy defines how long to keep logs for a category.
type LogRetentionPolicy struct {
	Category   LogCategory   `json:"category"`
	MaxAge     time.Duration `json:"max_age"`
	MaxEntries int64         `json:"max_entries,omitempty"` // 0 = unlimited
}

// LogStats contains statistics about stored logs.
type LogStats struct {
	TotalEntries      int64                 `json:"total_entries"`
	EntriesByCategory map[LogCategory]int64 `json:"entries_by_category"`
	EntriesByLevel    map[LogLevel]int64    `json:"entries_by_level"`
	OldestEntry       time.Time             `json:"oldest_entry,omitempty"`
	NewestEntry       time.Time             `json:"newest_entry,omitempty"`
}

// ExecutionLogEvent is the event sent via PubSub for real-time log streaming.
type ExecutionLogEvent struct {
	ExecutionID   string    `json:"execution_id"`
	ExecutionType string    `json:"execution_type"` // "function", "job", "rpc"
	LineNumber    int       `json:"line_number"`
	Level         LogLevel  `json:"level"`
	Message       string    `json:"message"`
	Timestamp     time.Time `json:"timestamp"`
}

// LogStreamEvent is the event sent via PubSub for real-time log streaming (all categories).
type LogStreamEvent struct {
	ID             string         `json:"id"`
	Timestamp      time.Time      `json:"timestamp"`
	Category       LogCategory    `json:"category"`
	Level          LogLevel       `json:"level"`
	Message        string         `json:"message"`
	CustomCategory string         `json:"custom_category,omitempty"`
	RequestID      string         `json:"request_id,omitempty"`
	TraceID        string         `json:"trace_id,omitempty"`
	Component      string         `json:"component,omitempty"`
	UserID         string         `json:"user_id,omitempty"`
	IPAddress      string         `json:"ip_address,omitempty"`
	Fields         map[string]any `json:"fields,omitempty"`
	ExecutionID    string         `json:"execution_id,omitempty"`
	ExecutionType  string         `json:"execution_type,omitempty"`
	LineNumber     int            `json:"line_number,omitempty"`
}
