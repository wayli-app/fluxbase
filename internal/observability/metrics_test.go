package observability

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStatusClass(t *testing.T) {
	testCases := []struct {
		status   int
		expected string
	}{
		{200, "2xx"},
		{201, "2xx"},
		{204, "2xx"},
		{299, "2xx"},
		{300, "3xx"},
		{301, "3xx"},
		{304, "3xx"},
		{399, "3xx"},
		{400, "4xx"},
		{401, "4xx"},
		{403, "4xx"},
		{404, "4xx"},
		{499, "4xx"},
		{500, "5xx"},
		{502, "5xx"},
		{503, "5xx"},
		{599, "5xx"},
		{100, "unknown"},
		{0, "unknown"},
		{600, "5xx"}, // >= 500 returns 5xx
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("status_%d", tc.status), func(t *testing.T) {
			result := statusClass(tc.status)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestNormalizePath(t *testing.T) {
	t.Run("returns path unchanged for short paths", func(t *testing.T) {
		result := normalizePath("/api/v1/users")
		assert.Equal(t, "/api/v1/users", result)
	})

	t.Run("returns long_path for paths over 50 chars", func(t *testing.T) {
		longPath := "/api/v1/very/long/path/that/exceeds/fifty/characters/limit/here"
		result := normalizePath(longPath)
		assert.Equal(t, "long_path", result)
	})

	t.Run("handles empty path", func(t *testing.T) {
		result := normalizePath("")
		assert.Equal(t, "", result)
	})

	t.Run("handles root path", func(t *testing.T) {
		result := normalizePath("/")
		assert.Equal(t, "/", result)
	})
}

func TestMetrics_Struct(t *testing.T) {
	t.Run("metrics struct has expected fields", func(t *testing.T) {
		m := &Metrics{}
		// Just verify the struct can be created
		assert.NotNil(t, m)
	})
}

// TestMetrics_AllMethods tests all metrics methods using the singleton instance
// We use a single test to avoid duplicate metric registration issues
func TestMetrics_AllMethods(t *testing.T) {
	// Use the singleton pattern - NewMetrics returns the same instance
	m := NewMetrics()
	require.NotNil(t, m)

	t.Run("RecordDBQuery", func(t *testing.T) {
		assert.NotPanics(t, func() {
			m.RecordDBQuery("SELECT", "users", 100*time.Millisecond, nil)
		})
	})

	t.Run("UpdateDBStats", func(t *testing.T) {
		assert.NotPanics(t, func() {
			m.UpdateDBStats(10, 5, 100)
		})
	})

	t.Run("UpdateRealtimeStats", func(t *testing.T) {
		assert.NotPanics(t, func() {
			m.UpdateRealtimeStats(50, 10, 200)
		})
	})

	t.Run("RecordRealtimeMessage", func(t *testing.T) {
		assert.NotPanics(t, func() {
			m.RecordRealtimeMessage("broadcast")
		})
	})

	t.Run("RecordRealtimeError", func(t *testing.T) {
		assert.NotPanics(t, func() {
			m.RecordRealtimeError("connection_timeout")
		})
	})

	t.Run("RecordStorageOperation_success", func(t *testing.T) {
		assert.NotPanics(t, func() {
			m.RecordStorageOperation("upload", "avatars", 1024, 50*time.Millisecond, nil)
		})
	})

	t.Run("RecordStorageOperation_error", func(t *testing.T) {
		assert.NotPanics(t, func() {
			m.RecordStorageOperation("download", "documents", 0, 100*time.Millisecond, assert.AnError)
		})
	})

	t.Run("RecordAuthAttempt_success", func(t *testing.T) {
		assert.NotPanics(t, func() {
			m.RecordAuthAttempt("password", true, "")
		})
	})

	t.Run("RecordAuthAttempt_failure", func(t *testing.T) {
		assert.NotPanics(t, func() {
			m.RecordAuthAttempt("password", false, "invalid_credentials")
		})
	})

	t.Run("RecordAuthToken", func(t *testing.T) {
		assert.NotPanics(t, func() {
			m.RecordAuthToken("access_token")
			m.RecordAuthToken("refresh_token")
		})
	})

	t.Run("RecordRateLimitHit", func(t *testing.T) {
		assert.NotPanics(t, func() {
			m.RecordRateLimitHit("api", "192.168.1.1")
		})
	})

	t.Run("UpdateUptime", func(t *testing.T) {
		startTime := time.Now().Add(-time.Hour)
		assert.NotPanics(t, func() {
			m.UpdateUptime(startTime)
		})
	})

	t.Run("RecordAIChatRequest", func(t *testing.T) {
		assert.NotPanics(t, func() {
			m.RecordAIChatRequest("support-bot", "success", 2*time.Second)
		})
	})

	t.Run("RecordAITokens", func(t *testing.T) {
		assert.NotPanics(t, func() {
			m.RecordAITokens("support-bot", 100, 50)
		})
	})

	t.Run("RecordAISQLQuery_executed", func(t *testing.T) {
		assert.NotPanics(t, func() {
			m.RecordAISQLQuery("data-bot", "executed", 100*time.Millisecond)
		})
	})

	t.Run("RecordAISQLQuery_rejected", func(t *testing.T) {
		assert.NotPanics(t, func() {
			m.RecordAISQLQuery("data-bot", "rejected", 0)
		})
	})

	t.Run("UpdateAIConversations", func(t *testing.T) {
		assert.NotPanics(t, func() {
			m.UpdateAIConversations(25)
		})
	})

	t.Run("UpdateAIWebSocketConnections", func(t *testing.T) {
		assert.NotPanics(t, func() {
			m.UpdateAIWebSocketConnections(10)
		})
	})

	t.Run("RecordAIProviderRequest", func(t *testing.T) {
		assert.NotPanics(t, func() {
			m.RecordAIProviderRequest("openai", "success", 500*time.Millisecond)
		})
	})

	t.Run("RecordRPCExecution_success", func(t *testing.T) {
		assert.NotPanics(t, func() {
			m.RecordRPCExecution("get_user_stats", "success", 50*time.Millisecond)
		})
	})

	t.Run("RecordRPCExecution_error", func(t *testing.T) {
		assert.NotPanics(t, func() {
			m.RecordRPCExecution("create_order", "error", 0)
		})
	})

	// Job metrics tests
	t.Run("UpdateJobQueueDepth", func(t *testing.T) {
		assert.NotPanics(t, func() {
			m.UpdateJobQueueDepth("emails", "high", 10)
			m.UpdateJobQueueDepth("emails", "normal", 25)
			m.UpdateJobQueueDepth("emails", "low", 50)
		})
	})

	t.Run("UpdateJobQueueDepth_default_namespace", func(t *testing.T) {
		assert.NotPanics(t, func() {
			m.UpdateJobQueueDepth("", "normal", 15) // empty namespace should become "default"
		})
	})

	t.Run("UpdateJobsProcessing", func(t *testing.T) {
		assert.NotPanics(t, func() {
			m.UpdateJobsProcessing(5)
			m.UpdateJobsProcessing(0)
		})
	})

	t.Run("RecordJobCompleted", func(t *testing.T) {
		assert.NotPanics(t, func() {
			m.RecordJobCompleted("notifications", "send_email", 2*time.Second)
			m.RecordJobCompleted("notifications", "send_push", 500*time.Millisecond)
		})
	})

	t.Run("RecordJobCompleted_default_namespace", func(t *testing.T) {
		assert.NotPanics(t, func() {
			m.RecordJobCompleted("", "cleanup", 100*time.Millisecond) // empty namespace should become "default"
		})
	})

	t.Run("RecordJobFailed_timeout", func(t *testing.T) {
		assert.NotPanics(t, func() {
			m.RecordJobFailed("reports", "generate_pdf", "timeout", 5*time.Minute)
		})
	})

	t.Run("RecordJobFailed_error", func(t *testing.T) {
		assert.NotPanics(t, func() {
			m.RecordJobFailed("imports", "csv_import", "error", 30*time.Second)
		})
	})

	t.Run("RecordJobFailed_cancelled", func(t *testing.T) {
		assert.NotPanics(t, func() {
			m.RecordJobFailed("exports", "data_export", "cancelled", 10*time.Second)
		})
	})

	t.Run("RecordJobFailed_panic", func(t *testing.T) {
		assert.NotPanics(t, func() {
			m.RecordJobFailed("processing", "image_resize", "panic", 1*time.Second)
		})
	})

	t.Run("RecordJobFailed_default_namespace", func(t *testing.T) {
		assert.NotPanics(t, func() {
			m.RecordJobFailed("", "sync_data", "error", 2*time.Second) // empty namespace should become "default"
		})
	})

	t.Run("UpdateJobWorkers", func(t *testing.T) {
		assert.NotPanics(t, func() {
			m.UpdateJobWorkers(4, 0.75) // 4 active workers, 75% utilization
			m.UpdateJobWorkers(0, 0.0)  // no active workers
			m.UpdateJobWorkers(8, 1.0)  // full utilization
		})
	})

	t.Run("Handler", func(t *testing.T) {
		handler := m.Handler()
		assert.NotNil(t, handler)
	})

	t.Run("MetricsMiddleware", func(t *testing.T) {
		middleware := m.MetricsMiddleware()
		assert.NotNil(t, middleware)
	})
}
