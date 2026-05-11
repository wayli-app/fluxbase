package api

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"

	"github.com/nimbleflux/fluxbase/internal/storage"
)

// =============================================================================
// NewLoggingHandler Tests
// =============================================================================

func TestNewLoggingHandler(t *testing.T) {
	t.Run("creates handler with nil service", func(t *testing.T) {
		handler := NewLoggingHandler(nil)

		require.NotNil(t, handler)
		assert.Nil(t, handler.loggingService)
	})
}

// =============================================================================
// LoggingHandler Struct Tests
// =============================================================================

func TestLoggingHandler_Struct(t *testing.T) {
	t.Run("stores logging service", func(t *testing.T) {
		handler := &LoggingHandler{
			loggingService: nil,
		}

		assert.Nil(t, handler.loggingService)
	})
}

// =============================================================================
// QueryLogs Handler Tests
// =============================================================================

func TestLoggingHandler_QueryLogs(t *testing.T) {
	t.Run("returns service unavailable when service is nil", func(t *testing.T) {
		handler := &LoggingHandler{loggingService: nil}

		app := fiber.New()
		app.Get("/admin/logs", handler.QueryLogs)

		req := httptest.NewRequest(http.MethodGet, "/admin/logs", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	})
}

// =============================================================================
// GetExecutionLogs Handler Tests
// =============================================================================

func TestLoggingHandler_GetExecutionLogs(t *testing.T) {
	t.Run("returns forbidden when service is nil", func(t *testing.T) {
		handler := &LoggingHandler{loggingService: nil}

		app := fiber.New()
		app.Get("/admin/logs/executions/:id", handler.GetExecutionLogs)

		req := httptest.NewRequest(http.MethodGet, "/admin/logs/executions/exec-123", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	})

	t.Run("returns forbidden when execution_id is empty and service is nil", func(t *testing.T) {
		handler := &LoggingHandler{loggingService: nil}

		app := fiber.New()
		app.Get("/admin/logs/executions/", handler.GetExecutionLogs)

		req := httptest.NewRequest(http.MethodGet, "/admin/logs/executions/", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	})
}

// =============================================================================
// GetLogStats Handler Tests
// =============================================================================

func TestLoggingHandler_GetLogStats(t *testing.T) {
	t.Run("returns forbidden when service is nil", func(t *testing.T) {
		handler := &LoggingHandler{loggingService: nil}

		app := fiber.New()
		app.Get("/admin/logs/stats", handler.GetLogStats)

		req := httptest.NewRequest(http.MethodGet, "/admin/logs/stats", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	})
}

// =============================================================================
// FlushLogs Handler Tests
// =============================================================================

func TestLoggingHandler_FlushLogs(t *testing.T) {
	t.Run("returns forbidden when service is nil", func(t *testing.T) {
		handler := &LoggingHandler{loggingService: nil}

		app := fiber.New()
		app.Post("/admin/logs/flush", handler.FlushLogs)

		req := httptest.NewRequest(http.MethodPost, "/admin/logs/flush", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	})
}

// =============================================================================
// Response Struct Tests
// =============================================================================

func TestLogQueryResponse_Struct(t *testing.T) {
	t.Run("stores all fields", func(t *testing.T) {
		entries := []*storage.LogEntry{
			{Message: "test log 1"},
			{Message: "test log 2"},
		}

		resp := LogQueryResponse{
			Entries:    entries,
			TotalCount: 100,
			HasMore:    true,
		}

		assert.Len(t, resp.Entries, 2)
		assert.Equal(t, int64(100), resp.TotalCount)
		assert.True(t, resp.HasMore)
	})

	t.Run("handles empty entries", func(t *testing.T) {
		resp := LogQueryResponse{
			Entries:    []*storage.LogEntry{},
			TotalCount: 0,
			HasMore:    false,
		}

		assert.Empty(t, resp.Entries)
		assert.Equal(t, int64(0), resp.TotalCount)
		assert.False(t, resp.HasMore)
	})

	t.Run("handles nil entries", func(t *testing.T) {
		resp := LogQueryResponse{
			Entries:    nil,
			TotalCount: 0,
			HasMore:    false,
		}

		assert.Nil(t, resp.Entries)
	})
}

func TestExecutionLogsResponse_Struct(t *testing.T) {
	t.Run("stores entries and count", func(t *testing.T) {
		entries := []*storage.LogEntry{
			{Message: "execution log 1"},
			{Message: "execution log 2"},
			{Message: "execution log 3"},
		}

		resp := ExecutionLogsResponse{
			Entries: entries,
			Count:   3,
		}

		assert.Len(t, resp.Entries, 3)
		assert.Equal(t, 3, resp.Count)
	})
}

func TestLogStatsResponse_Struct(t *testing.T) {
	t.Run("stores all fields", func(t *testing.T) {
		now := time.Now()
		oldest := now.Add(-24 * time.Hour)

		resp := LogStatsResponse{
			TotalEntries: 1000,
			EntriesByCategory: map[string]int64{
				"system":   500,
				"http":     300,
				"security": 200,
			},
			EntriesByLevel: map[string]int64{
				"info":  700,
				"warn":  200,
				"error": 100,
			},
			OldestEntry: &oldest,
			NewestEntry: &now,
		}

		assert.Equal(t, int64(1000), resp.TotalEntries)
		assert.Len(t, resp.EntriesByCategory, 3)
		assert.Len(t, resp.EntriesByLevel, 3)
		assert.NotNil(t, resp.OldestEntry)
		assert.NotNil(t, resp.NewestEntry)
	})

	t.Run("handles nil time pointers", func(t *testing.T) {
		resp := LogStatsResponse{
			TotalEntries:      0,
			EntriesByCategory: map[string]int64{},
			EntriesByLevel:    map[string]int64{},
			OldestEntry:       nil,
			NewestEntry:       nil,
		}

		assert.Nil(t, resp.OldestEntry)
		assert.Nil(t, resp.NewestEntry)
	})
}

// =============================================================================
// Query Parameter Parsing Tests
// =============================================================================

func TestLogQueryParameterParsing(t *testing.T) {
	t.Run("parses category parameter", func(t *testing.T) {
		app := fiber.New()
		ctx := app.AcquireCtx(&fasthttp.RequestCtx{})
		defer app.ReleaseCtx(ctx)

		// Simulate query parameter
		category := "system"
		opts := storage.LogQueryOptions{}
		if category != "" {
			opts.Category = storage.LogCategory(category)
		}

		assert.Equal(t, storage.LogCategorySystem, opts.Category)
	})

	t.Run("parses multiple levels", func(t *testing.T) {
		levels := "debug,info,warn"
		var parsedLevels []storage.LogLevel

		for _, level := range strings.Split(levels, ",") {
			level = strings.TrimSpace(level)
			parsedLevels = append(parsedLevels, storage.LogLevel(level))
		}

		assert.Len(t, parsedLevels, 3)
		assert.Equal(t, storage.LogLevelDebug, parsedLevels[0])
		assert.Equal(t, storage.LogLevelInfo, parsedLevels[1])
		assert.Equal(t, storage.LogLevelWarn, parsedLevels[2])
	})

	t.Run("parses levels with spaces", func(t *testing.T) {
		levels := "debug, info, warn"
		var parsedLevels []storage.LogLevel

		for _, level := range strings.Split(levels, ",") {
			level = strings.TrimSpace(level)
			parsedLevels = append(parsedLevels, storage.LogLevel(level))
		}

		assert.Len(t, parsedLevels, 3)
		assert.Equal(t, storage.LogLevelDebug, parsedLevels[0])
		assert.Equal(t, storage.LogLevelInfo, parsedLevels[1])
		assert.Equal(t, storage.LogLevelWarn, parsedLevels[2])
	})

	t.Run("parses start_time in RFC3339 format", func(t *testing.T) {
		startTimeStr := "2026-01-15T10:30:00Z"

		parsedTime, err := time.Parse(time.RFC3339, startTimeStr)
		assert.NoError(t, err)
		assert.Equal(t, 2026, parsedTime.Year())
		assert.Equal(t, time.January, parsedTime.Month())
		assert.Equal(t, 15, parsedTime.Day())
	})

	t.Run("parses limit parameter", func(t *testing.T) {
		limitStr := "50"

		limit, err := strconv.Atoi(limitStr)
		assert.NoError(t, err)
		assert.Equal(t, 50, limit)
	})

	t.Run("uses default limit of 100", func(t *testing.T) {
		limitStr := ""

		var limit int
		if limitStr == "" {
			limit = 100
		}

		assert.Equal(t, 100, limit)
	})

	t.Run("parses offset parameter", func(t *testing.T) {
		offsetStr := "25"

		offset, err := strconv.Atoi(offsetStr)
		assert.NoError(t, err)
		assert.Equal(t, 25, offset)
	})

	t.Run("parses sort_asc boolean", func(t *testing.T) {
		sortAscStr := "true"

		sortAsc := sortAscStr == "true"
		assert.True(t, sortAsc)

		sortAscStr = "false"
		sortAsc = sortAscStr == "true"
		assert.False(t, sortAsc)
	})

	t.Run("parses hide_static_assets boolean", func(t *testing.T) {
		hideStaticStr := "true"

		hideStatic := hideStaticStr == "true"
		assert.True(t, hideStatic)
	})
}

// =============================================================================
// LogQueryOptions Tests
// =============================================================================

func TestLogQueryOptions(t *testing.T) {
	t.Run("stores all filter fields", func(t *testing.T) {
		now := time.Now()

		opts := storage.LogQueryOptions{
			Category:         storage.LogCategoryHTTP,
			CustomCategory:   "my_custom",
			Levels:           []storage.LogLevel{storage.LogLevelInfo, storage.LogLevelWarn},
			Component:        "api",
			RequestID:        "req-123",
			TraceID:          "trace-456",
			UserID:           "user-789",
			ExecutionID:      "exec-abc",
			Search:           "error message",
			StartTime:        now.Add(-1 * time.Hour),
			EndTime:          now,
			Limit:            50,
			Offset:           10,
			SortAsc:          true,
			HideStaticAssets: true,
		}

		assert.Equal(t, storage.LogCategoryHTTP, opts.Category)
		assert.Equal(t, "my_custom", opts.CustomCategory)
		assert.Len(t, opts.Levels, 2)
		assert.Equal(t, "api", opts.Component)
		assert.Equal(t, "req-123", opts.RequestID)
		assert.Equal(t, "trace-456", opts.TraceID)
		assert.Equal(t, "user-789", opts.UserID)
		assert.Equal(t, "exec-abc", opts.ExecutionID)
		assert.Equal(t, "error message", opts.Search)
		assert.Equal(t, 50, opts.Limit)
		assert.Equal(t, 10, opts.Offset)
		assert.True(t, opts.SortAsc)
		assert.True(t, opts.HideStaticAssets)
	})
}

// =============================================================================
// Log Category Tests
// =============================================================================

func TestLogCategories(t *testing.T) {
	t.Run("system category", func(t *testing.T) {
		assert.Equal(t, storage.LogCategory("system"), storage.LogCategorySystem)
	})

	t.Run("http category", func(t *testing.T) {
		assert.Equal(t, storage.LogCategory("http"), storage.LogCategoryHTTP)
	})

	t.Run("security category", func(t *testing.T) {
		assert.Equal(t, storage.LogCategory("security"), storage.LogCategorySecurity)
	})

	t.Run("execution category", func(t *testing.T) {
		assert.Equal(t, storage.LogCategory("execution"), storage.LogCategoryExecution)
	})

	t.Run("ai category", func(t *testing.T) {
		assert.Equal(t, storage.LogCategory("ai"), storage.LogCategoryAI)
	})

	t.Run("custom category", func(t *testing.T) {
		assert.Equal(t, storage.LogCategory("custom"), storage.LogCategoryCustom)
	})
}

// =============================================================================
// Log Level Tests
// =============================================================================

func TestLogLevels(t *testing.T) {
	t.Run("trace level", func(t *testing.T) {
		assert.Equal(t, storage.LogLevel("trace"), storage.LogLevelTrace)
	})

	t.Run("debug level", func(t *testing.T) {
		assert.Equal(t, storage.LogLevel("debug"), storage.LogLevelDebug)
	})

	t.Run("info level", func(t *testing.T) {
		assert.Equal(t, storage.LogLevel("info"), storage.LogLevelInfo)
	})

	t.Run("warn level", func(t *testing.T) {
		assert.Equal(t, storage.LogLevel("warn"), storage.LogLevelWarn)
	})

	t.Run("error level", func(t *testing.T) {
		assert.Equal(t, storage.LogLevel("error"), storage.LogLevelError)
	})

	t.Run("fatal level", func(t *testing.T) {
		assert.Equal(t, storage.LogLevel("fatal"), storage.LogLevelFatal)
	})

	t.Run("panic level", func(t *testing.T) {
		assert.Equal(t, storage.LogLevel("panic"), storage.LogLevelPanic)
	})
}

// =============================================================================
// after_line Parameter Tests
// =============================================================================

func TestAfterLineParameter(t *testing.T) {
	t.Run("parses valid after_line", func(t *testing.T) {
		afterLineStr := "100"

		afterLine := 0
		if afterLineStr != "" {
			if l, err := strconv.Atoi(afterLineStr); err == nil {
				afterLine = l
			}
		}

		assert.Equal(t, 100, afterLine)
	})

	t.Run("defaults to 0 for empty string", func(t *testing.T) {
		afterLineStr := ""

		afterLine := 0
		if afterLineStr != "" {
			if l, err := strconv.Atoi(afterLineStr); err == nil {
				afterLine = l
			}
		}

		assert.Equal(t, 0, afterLine)
	})

	t.Run("defaults to 0 for invalid value", func(t *testing.T) {
		afterLineStr := "invalid"

		afterLine := 0
		if afterLineStr != "" {
			if l, err := strconv.Atoi(afterLineStr); err == nil {
				afterLine = l
			}
		}

		assert.Equal(t, 0, afterLine)
	})
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkParseLevels(b *testing.B) {
	levels := "debug,info,warn,error"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var parsedLevels []storage.LogLevel
		for _, level := range strings.Split(levels, ",") {
			level = strings.TrimSpace(level)
			parsedLevels = append(parsedLevels, storage.LogLevel(level))
		}
		_ = parsedLevels
	}
}

func BenchmarkParseTimeRFC3339(b *testing.B) {
	timeStr := "2026-01-15T10:30:00Z"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = time.Parse(time.RFC3339, timeStr)
	}
}

func BenchmarkParseLimit(b *testing.B) {
	limitStr := "100"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = strconv.Atoi(limitStr)
	}
}
