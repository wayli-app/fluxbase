package api

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/database"
	apperrors "github.com/nimbleflux/fluxbase/internal/errors"
	"github.com/nimbleflux/fluxbase/internal/logging"
	"github.com/nimbleflux/fluxbase/internal/middleware"
	"github.com/nimbleflux/fluxbase/internal/storage"
)

func ctxForLogs(c fiber.Ctx) context.Context {
	if middleware.IsInstanceAdminFromContext(c) && middleware.GetTenantSourceFromContext(c) == "default" {
		return database.ContextWithTenant(c.RequestCtx(), "")
	}
	return middleware.CtxWithTenant(c)
}

// LoggingHandler handles logging-related API endpoints
type LoggingHandler struct {
	loggingService *logging.Service
}

// NewLoggingHandler creates a new logging handler
func NewLoggingHandler(loggingService *logging.Service) *LoggingHandler {
	return &LoggingHandler{
		loggingService: loggingService,
	}
}

// QueryLogs handles GET /admin/logs
// @Summary Query logs
// @Description Query logs with filters
// @Tags Admin/Logging
// @Accept json
// @Produce json
// @Param category query string false "Log category (system, http, security, execution, ai, custom)"
// @Param custom_category query string false "Custom category name (only used when category=custom)"
// @Param level query string false "Log levels (comma-separated: debug, info, warn, error)"
// @Param component query string false "Component name"
// @Param request_id query string false "Request ID"
// @Param trace_id query string false "Trace ID"
// @Param user_id query string false "User ID"
// @Param execution_id query string false "Execution ID"
// @Param search query string false "Search text in message"
// @Param start_time query string false "Start time (RFC3339)"
// @Param end_time query string false "End time (RFC3339)"
// @Param limit query int false "Max results (default 100)"
// @Param offset query int false "Offset for pagination"
// @Param sort_asc query bool false "Sort ascending by timestamp"
// @Param hide_static_assets query bool false "Hide HTTP logs for static assets (js, css, images, fonts)"
// @Success 200 {object} LogQueryResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /admin/logs [get]
func (h *LoggingHandler) QueryLogs(c fiber.Ctx) error {
	if h.loggingService == nil {
		return SendFeatureDisabled(c, "Logging")
	}

	opts := storage.LogQueryOptions{}

	// Parse category
	if category := c.Query("category"); category != "" {
		opts.Category = storage.LogCategory(category)
	}

	// Parse custom category (for category=custom)
	if customCategory := c.Query("custom_category"); customCategory != "" {
		opts.CustomCategory = customCategory
	}

	// Parse levels
	if levels := c.Query("level"); levels != "" {
		for _, level := range strings.Split(levels, ",") {
			level = strings.TrimSpace(level)
			opts.Levels = append(opts.Levels, storage.LogLevel(level))
		}
	}

	// Parse other filters
	opts.Component = c.Query("component")
	opts.RequestID = c.Query("request_id")
	opts.TraceID = c.Query("trace_id")
	opts.UserID = c.Query("user_id")
	opts.ExecutionID = c.Query("execution_id")
	opts.Search = c.Query("search")

	// Parse time range
	if startTime := c.Query("start_time"); startTime != "" {
		if t, err := time.Parse(time.RFC3339, startTime); err == nil {
			opts.StartTime = t
		}
	}
	if endTime := c.Query("end_time"); endTime != "" {
		if t, err := time.Parse(time.RFC3339, endTime); err == nil {
			opts.EndTime = t
		}
	}

	// Parse pagination
	if limit := c.Query("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil && l > 0 {
			opts.Limit = l
		}
	} else {
		opts.Limit = 100
	}
	if offset := c.Query("offset"); offset != "" {
		if o, err := strconv.Atoi(offset); err == nil && o >= 0 {
			opts.Offset = o
		}
	}

	// Parse sort order
	opts.SortAsc = c.Query("sort_asc") == "true"

	// Parse hide static assets filter
	opts.HideStaticAssets = c.Query("hide_static_assets") == "true"

	// Query logs
	result, err := h.loggingService.Query(ctxForLogs(c), opts)
	if err != nil {
		log.Error().Err(err).Msg("Failed to query logs")
		return SendInternalError(c, "Internal error")
	}

	return c.JSON(fiber.Map{
		"entries":     result.Entries,
		"total_count": result.TotalCount,
		"has_more":    result.HasMore,
	})
}

// GetExecutionLogs handles GET /admin/logs/executions/:id
// @Summary Get execution logs
// @Description Get logs for a specific execution
// @Tags Admin/Logging
// @Accept json
// @Produce json
// @Param id path string true "Execution ID"
// @Param after_line query int false "Return logs after this line number"
// @Success 200 {object} ExecutionLogsResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /admin/logs/executions/{execution_id} [get]
func (h *LoggingHandler) GetExecutionLogs(c fiber.Ctx) error {
	if h.loggingService == nil {
		return SendFeatureDisabled(c, "Logging")
	}

	executionID := c.Params("id")
	if executionID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "execution_id is required",
		})
	}

	afterLine := 0
	if afterLineStr := c.Query("after_line"); afterLineStr != "" {
		if l, err := strconv.Atoi(afterLineStr); err == nil {
			afterLine = l
		}
	}

	entries, err := h.loggingService.GetExecutionLogs(middleware.CtxWithTenant(c), executionID, afterLine)
	if err != nil {
		log.Error().Err(err).Str("execution_id", executionID).Msg("Failed to get execution logs")
		return SendInternalError(c, "Internal error")
	}

	return c.JSON(fiber.Map{
		"entries": entries,
		"count":   len(entries),
	})
}

// GetLogStats handles GET /admin/logs/stats
// @Summary Get log statistics
// @Description Get statistics about stored logs
// @Tags Admin/Logging
// @Accept json
// @Produce json
// @Success 200 {object} LogStatsResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /admin/logs/stats [get]
func (h *LoggingHandler) GetLogStats(c fiber.Ctx) error {
	if h.loggingService == nil {
		return SendFeatureDisabled(c, "Logging")
	}

	stats, err := h.loggingService.Stats(ctxForLogs(c))
	if err != nil {
		log.Error().Err(err).Msg("Failed to get log stats")
		return SendInternalError(c, "Internal error")
	}

	return c.JSON(stats)
}

// FlushLogs handles POST /admin/logs/flush
// @Summary Flush buffered logs
// @Description Force flush any buffered log entries to storage
// @Tags Admin/Logging
// @Accept json
// @Produce json
// @Success 200 {object} SuccessResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /admin/logs/flush [post]
func (h *LoggingHandler) FlushLogs(c fiber.Ctx) error {
	if h.loggingService == nil {
		return SendFeatureDisabled(c, "Logging")
	}

	if err := h.loggingService.Flush(middleware.CtxWithTenant(c)); err != nil {
		log.Error().Err(err).Msg("Failed to flush logs")
		return SendInternalError(c, "Internal error")
	}

	return apperrors.SendSuccess(c, "Logs flushed successfully")
}

// GenerateTestLogs handles POST /admin/logs/test
// @Summary Generate test logs
// @Description Generates test log entries for diagnostic purposes
// @Tags Admin/Logging
// @Accept json
// @Produce json
// @Success 200 {object} SuccessResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /admin/logs/test [post]
func (h *LoggingHandler) GenerateTestLogs(c fiber.Ctx) error {
	if h.loggingService == nil {
		return SendFeatureDisabled(c, "Logging")
	}

	// Generate test logs of different types and levels
	testLogs := []*storage.LogEntry{
		{
			Category: storage.LogCategorySystem,
			Level:    storage.LogLevelInfo,
			Message:  "Test info log from admin endpoint",
			Fields:   map[string]any{"test": true, "component": "admin_test"},
		},
		{
			Category: storage.LogCategorySystem,
			Level:    storage.LogLevelWarn,
			Message:  "Test warn log from admin endpoint",
			Fields:   map[string]any{"test": true, "component": "admin_test"},
		},
		{
			Category: storage.LogCategorySystem,
			Level:    storage.LogLevelError,
			Message:  "Test error log from admin endpoint",
			Fields:   map[string]any{"test": true, "component": "admin_test"},
		},
		{
			Category: storage.LogCategoryHTTP,
			Level:    storage.LogLevelInfo,
			Message:  "Test HTTP log - page navigation simulation",
			Fields: map[string]any{
				"test":        true,
				"method":      "GET",
				"path":        "/admin/logs/test",
				"status_code": 200,
				"duration_ms": 45,
			},
		},
	}

	// Log each test entry
	ctx := middleware.CtxWithTenant(c)
	for _, logEntry := range testLogs {
		h.loggingService.Log(ctx, logEntry)
	}

	return c.JSON(fiber.Map{
		"message":      "Test logs generated",
		"count":        len(testLogs),
		"check_page":   "/admin/logs",
		"check_db":     "Query logging.logs table directly",
		"check_stream": "Look for WebSocket log_entry messages",
	})
}

// LogQueryResponse represents the response from log query
type LogQueryResponse struct {
	Entries    []*storage.LogEntry `json:"entries"`
	TotalCount int64               `json:"total_count"`
	HasMore    bool                `json:"has_more"`
}

// ExecutionLogsResponse represents the response from execution logs query
type ExecutionLogsResponse struct {
	Entries []*storage.LogEntry `json:"entries"`
	Count   int                 `json:"count"`
}

// LogStatsResponse represents the response from log stats
type LogStatsResponse struct {
	TotalEntries      int64            `json:"total_entries"`
	EntriesByCategory map[string]int64 `json:"entries_by_category"`
	EntriesByLevel    map[string]int64 `json:"entries_by_level"`
	OldestEntry       *time.Time       `json:"oldest_entry,omitempty"`
	NewestEntry       *time.Time       `json:"newest_entry,omitempty"`
}

// fiber:context-methods migrated
