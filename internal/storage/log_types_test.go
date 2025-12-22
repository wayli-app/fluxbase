package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsBuiltinCategory(t *testing.T) {
	tests := []struct {
		category LogCategory
		expected bool
	}{
		{LogCategorySystem, true},
		{LogCategoryHTTP, true},
		{LogCategorySecurity, true},
		{LogCategoryExecution, true},
		{LogCategoryAI, true},
		{LogCategoryCustom, false},
		{"unknown", false},
		{"", false},
	}

	for _, tc := range tests {
		t.Run(string(tc.category), func(t *testing.T) {
			result := IsBuiltinCategory(tc.category)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestAllBuiltinCategories(t *testing.T) {
	categories := AllBuiltinCategories()

	assert.Len(t, categories, 5)
	assert.Contains(t, categories, LogCategorySystem)
	assert.Contains(t, categories, LogCategoryHTTP)
	assert.Contains(t, categories, LogCategorySecurity)
	assert.Contains(t, categories, LogCategoryExecution)
	assert.Contains(t, categories, LogCategoryAI)
	assert.NotContains(t, categories, LogCategoryCustom)
}

func TestLogCategoryConstants(t *testing.T) {
	assert.Equal(t, LogCategory("system"), LogCategorySystem)
	assert.Equal(t, LogCategory("http"), LogCategoryHTTP)
	assert.Equal(t, LogCategory("security"), LogCategorySecurity)
	assert.Equal(t, LogCategory("execution"), LogCategoryExecution)
	assert.Equal(t, LogCategory("ai"), LogCategoryAI)
	assert.Equal(t, LogCategory("custom"), LogCategoryCustom)
}

func TestLogLevelConstants(t *testing.T) {
	assert.Equal(t, LogLevel("trace"), LogLevelTrace)
	assert.Equal(t, LogLevel("debug"), LogLevelDebug)
	assert.Equal(t, LogLevel("info"), LogLevelInfo)
	assert.Equal(t, LogLevel("warn"), LogLevelWarn)
	assert.Equal(t, LogLevel("error"), LogLevelError)
	assert.Equal(t, LogLevel("fatal"), LogLevelFatal)
	assert.Equal(t, LogLevel("panic"), LogLevelPanic)
}

func TestLogEntry_Fields(t *testing.T) {
	t.Run("entry with all fields", func(t *testing.T) {
		entry := &LogEntry{
			Category:       LogCategoryHTTP,
			Level:          LogLevelInfo,
			Message:        "Test message",
			CustomCategory: "",
			RequestID:      "req-123",
			TraceID:        "trace-456",
			Component:      "api",
			UserID:         "user-789",
			IPAddress:      "192.168.1.1",
			Fields: map[string]any{
				"key": "value",
			},
			ExecutionID:   "exec-001",
			ExecutionType: "function",
			LineNumber:    5,
		}

		assert.Equal(t, LogCategoryHTTP, entry.Category)
		assert.Equal(t, LogLevelInfo, entry.Level)
		assert.Equal(t, "Test message", entry.Message)
		assert.Equal(t, "req-123", entry.RequestID)
		assert.Equal(t, "trace-456", entry.TraceID)
		assert.Equal(t, "api", entry.Component)
		assert.Equal(t, "user-789", entry.UserID)
		assert.Equal(t, "192.168.1.1", entry.IPAddress)
		assert.Equal(t, "value", entry.Fields["key"])
		assert.Equal(t, "exec-001", entry.ExecutionID)
		assert.Equal(t, "function", entry.ExecutionType)
		assert.Equal(t, 5, entry.LineNumber)
	})

	t.Run("entry with custom category", func(t *testing.T) {
		entry := &LogEntry{
			Category:       LogCategoryCustom,
			CustomCategory: "metrics",
			Level:          LogLevelInfo,
			Message:        "Custom metric",
		}

		assert.Equal(t, LogCategoryCustom, entry.Category)
		assert.Equal(t, "metrics", entry.CustomCategory)
	})
}

func TestHTTPLogFields(t *testing.T) {
	fields := &HTTPLogFields{
		Method:        "GET",
		Path:          "/api/users",
		Query:         "limit=10",
		StatusCode:    200,
		DurationMs:    50,
		UserAgent:     "Mozilla/5.0",
		Referer:       "https://example.com",
		ResponseBytes: 1024,
		RequestBytes:  256,
	}

	assert.Equal(t, "GET", fields.Method)
	assert.Equal(t, "/api/users", fields.Path)
	assert.Equal(t, "limit=10", fields.Query)
	assert.Equal(t, 200, fields.StatusCode)
	assert.Equal(t, int64(50), fields.DurationMs)
	assert.Equal(t, "Mozilla/5.0", fields.UserAgent)
	assert.Equal(t, "https://example.com", fields.Referer)
	assert.Equal(t, 1024, fields.ResponseBytes)
	assert.Equal(t, 256, fields.RequestBytes)
}

func TestSecurityLogFields(t *testing.T) {
	fields := &SecurityLogFields{
		EventType: "login_success",
		Success:   true,
		Email:     "user@example.com",
		TargetID:  "target-123",
		Action:    "login",
		Details: map[string]any{
			"provider": "email",
		},
	}

	assert.Equal(t, "login_success", fields.EventType)
	assert.True(t, fields.Success)
	assert.Equal(t, "user@example.com", fields.Email)
	assert.Equal(t, "target-123", fields.TargetID)
	assert.Equal(t, "login", fields.Action)
	assert.Equal(t, "email", fields.Details["provider"])
}

func TestExecutionLogFields(t *testing.T) {
	fields := &ExecutionLogFields{
		ExecutionType: "function",
		FunctionName:  "myFunction",
		Namespace:     "default",
		JobType:       "",
		Status:        "completed",
		DurationMs:    1500,
	}

	assert.Equal(t, "function", fields.ExecutionType)
	assert.Equal(t, "myFunction", fields.FunctionName)
	assert.Equal(t, "default", fields.Namespace)
	assert.Equal(t, "completed", fields.Status)
	assert.Equal(t, int64(1500), fields.DurationMs)
}

func TestLogQueryOptions(t *testing.T) {
	t.Run("default options", func(t *testing.T) {
		opts := LogQueryOptions{}

		assert.Empty(t, opts.Category)
		assert.Empty(t, opts.Levels)
		assert.Empty(t, opts.Search)
		assert.Equal(t, 0, opts.Limit)
		assert.Equal(t, 0, opts.Offset)
		assert.False(t, opts.SortAsc)
		assert.False(t, opts.HideStaticAssets)
	})

	t.Run("populated options", func(t *testing.T) {
		opts := LogQueryOptions{
			Category:         LogCategoryHTTP,
			CustomCategory:   "metrics",
			Levels:           []LogLevel{LogLevelInfo, LogLevelWarn},
			Component:        "api",
			RequestID:        "req-123",
			TraceID:          "trace-456",
			UserID:           "user-789",
			ExecutionID:      "exec-001",
			ExecutionType:    "function",
			Search:           "error",
			Limit:            100,
			Offset:           50,
			AfterLine:        10,
			SortAsc:          true,
			HideStaticAssets: true,
		}

		assert.Equal(t, LogCategoryHTTP, opts.Category)
		assert.Equal(t, "metrics", opts.CustomCategory)
		assert.Len(t, opts.Levels, 2)
		assert.Equal(t, "api", opts.Component)
		assert.Equal(t, "req-123", opts.RequestID)
		assert.Equal(t, "trace-456", opts.TraceID)
		assert.Equal(t, "user-789", opts.UserID)
		assert.Equal(t, "exec-001", opts.ExecutionID)
		assert.Equal(t, "function", opts.ExecutionType)
		assert.Equal(t, "error", opts.Search)
		assert.Equal(t, 100, opts.Limit)
		assert.Equal(t, 50, opts.Offset)
		assert.Equal(t, 10, opts.AfterLine)
		assert.True(t, opts.SortAsc)
		assert.True(t, opts.HideStaticAssets)
	})
}

func TestLogQueryResult(t *testing.T) {
	result := &LogQueryResult{
		Entries: []*LogEntry{
			{Message: "Entry 1"},
			{Message: "Entry 2"},
		},
		TotalCount: 100,
		HasMore:    true,
	}

	assert.Len(t, result.Entries, 2)
	assert.Equal(t, int64(100), result.TotalCount)
	assert.True(t, result.HasMore)
}

func TestLogStats(t *testing.T) {
	stats := &LogStats{
		TotalEntries: 1000,
		EntriesByCategory: map[LogCategory]int64{
			LogCategorySystem:    400,
			LogCategoryHTTP:      300,
			LogCategorySecurity:  100,
			LogCategoryExecution: 150,
			LogCategoryAI:        50,
		},
		EntriesByLevel: map[LogLevel]int64{
			LogLevelInfo:  700,
			LogLevelWarn:  200,
			LogLevelError: 100,
		},
	}

	assert.Equal(t, int64(1000), stats.TotalEntries)
	assert.Equal(t, int64(400), stats.EntriesByCategory[LogCategorySystem])
	assert.Equal(t, int64(300), stats.EntriesByCategory[LogCategoryHTTP])
	assert.Equal(t, int64(700), stats.EntriesByLevel[LogLevelInfo])
}

func TestExecutionLogEvent(t *testing.T) {
	event := ExecutionLogEvent{
		ExecutionID:   "exec-123",
		ExecutionType: "function",
		LineNumber:    15,
		Level:         LogLevelInfo,
		Message:       "Processing request",
	}

	assert.Equal(t, "exec-123", event.ExecutionID)
	assert.Equal(t, "function", event.ExecutionType)
	assert.Equal(t, 15, event.LineNumber)
	assert.Equal(t, LogLevelInfo, event.Level)
	assert.Equal(t, "Processing request", event.Message)
}

func TestLogStreamEvent(t *testing.T) {
	event := LogStreamEvent{
		ID:             "550e8400-e29b-41d4-a716-446655440000",
		Category:       LogCategoryHTTP,
		Level:          LogLevelInfo,
		Message:        "GET /api/users",
		CustomCategory: "",
		RequestID:      "req-123",
		TraceID:        "trace-456",
		Component:      "api",
		UserID:         "user-789",
		IPAddress:      "192.168.1.1",
		Fields: map[string]any{
			"status": 200,
		},
		ExecutionID:   "",
		ExecutionType: "",
		LineNumber:    0,
	}

	assert.Equal(t, "550e8400-e29b-41d4-a716-446655440000", event.ID)
	assert.Equal(t, LogCategoryHTTP, event.Category)
	assert.Equal(t, LogLevelInfo, event.Level)
	assert.Equal(t, "GET /api/users", event.Message)
	assert.Equal(t, "req-123", event.RequestID)
	assert.Equal(t, 200, event.Fields["status"])
}
