package logging

import (
	"testing"
	"time"

	"github.com/fluxbase-eu/fluxbase/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected storage.LogLevel
	}{
		{"trace", storage.LogLevelTrace},
		{"TRACE", storage.LogLevelTrace},
		{"Trace", storage.LogLevelTrace},
		{"debug", storage.LogLevelDebug},
		{"DEBUG", storage.LogLevelDebug},
		{"info", storage.LogLevelInfo},
		{"INFO", storage.LogLevelInfo},
		{"warn", storage.LogLevelWarn},
		{"WARN", storage.LogLevelWarn},
		{"warning", storage.LogLevelWarn},
		{"WARNING", storage.LogLevelWarn},
		{"error", storage.LogLevelError},
		{"ERROR", storage.LogLevelError},
		{"fatal", storage.LogLevelFatal},
		{"FATAL", storage.LogLevelFatal},
		{"panic", storage.LogLevelPanic},
		{"PANIC", storage.LogLevelPanic},
		{"", storage.LogLevelInfo},
		{"unknown", storage.LogLevelInfo},
		{"invalid", storage.LogLevelInfo},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			result := parseLogLevel(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestWriter_ParseZerologJSON(t *testing.T) {
	// Create a minimal writer for testing the parsing function
	w := &Writer{}

	t.Run("parses basic log entry", func(t *testing.T) {
		jsonLog := `{"level":"info","message":"Test message","time":"2024-01-15T10:30:00Z"}`

		entry, err := w.parseZerologJSON([]byte(jsonLog))
		require.NoError(t, err)

		assert.Equal(t, storage.LogLevelInfo, entry.Level)
		assert.Equal(t, "Test message", entry.Message)
		assert.Equal(t, storage.LogCategorySystem, entry.Category)
	})

	t.Run("parses msg field as message", func(t *testing.T) {
		jsonLog := `{"level":"debug","msg":"Debug message"}`

		entry, err := w.parseZerologJSON([]byte(jsonLog))
		require.NoError(t, err)

		assert.Equal(t, "Debug message", entry.Message)
	})

	t.Run("parses time field", func(t *testing.T) {
		jsonLog := `{"level":"info","message":"Test","time":"2024-06-15T14:30:45Z"}`

		entry, err := w.parseZerologJSON([]byte(jsonLog))
		require.NoError(t, err)

		expected := time.Date(2024, 6, 15, 14, 30, 45, 0, time.UTC)
		assert.Equal(t, expected, entry.Timestamp)
	})

	t.Run("parses correlation IDs", func(t *testing.T) {
		jsonLog := `{"level":"info","message":"Test","request_id":"req-123","trace_id":"trace-456"}`

		entry, err := w.parseZerologJSON([]byte(jsonLog))
		require.NoError(t, err)

		assert.Equal(t, "req-123", entry.RequestID)
		assert.Equal(t, "trace-456", entry.TraceID)
	})

	t.Run("parses component", func(t *testing.T) {
		jsonLog := `{"level":"info","message":"Test","component":"api"}`

		entry, err := w.parseZerologJSON([]byte(jsonLog))
		require.NoError(t, err)

		assert.Equal(t, "api", entry.Component)
	})

	t.Run("parses user and IP fields", func(t *testing.T) {
		jsonLog := `{"level":"info","message":"Test","user_id":"user-789","ip_address":"192.168.1.1"}`

		entry, err := w.parseZerologJSON([]byte(jsonLog))
		require.NoError(t, err)

		assert.Equal(t, "user-789", entry.UserID)
		assert.Equal(t, "192.168.1.1", entry.IPAddress)
	})

	t.Run("parses ip field alias", func(t *testing.T) {
		jsonLog := `{"level":"info","message":"Test","ip":"10.0.0.1"}`

		entry, err := w.parseZerologJSON([]byte(jsonLog))
		require.NoError(t, err)

		assert.Equal(t, "10.0.0.1", entry.IPAddress)
	})

	t.Run("parses execution fields and sets category", func(t *testing.T) {
		jsonLog := `{"level":"debug","message":"Function executing","execution_id":"exec-001","execution_type":"function"}`

		entry, err := w.parseZerologJSON([]byte(jsonLog))
		require.NoError(t, err)

		assert.Equal(t, storage.LogCategoryExecution, entry.Category)
		assert.Equal(t, "exec-001", entry.ExecutionID)
		assert.Equal(t, "function", entry.ExecutionType)
	})

	t.Run("detects security category from security_event field", func(t *testing.T) {
		jsonLog := `{"level":"warn","message":"Login failed","security_event":"login_failed"}`

		entry, err := w.parseZerologJSON([]byte(jsonLog))
		require.NoError(t, err)

		assert.Equal(t, storage.LogCategorySecurity, entry.Category)
	})

	t.Run("detects HTTP category from method and status", func(t *testing.T) {
		jsonLog := `{"level":"info","message":"GET /api","method":"GET","status":200,"path":"/api"}`

		entry, err := w.parseZerologJSON([]byte(jsonLog))
		require.NoError(t, err)

		assert.Equal(t, storage.LogCategoryHTTP, entry.Category)
	})

	t.Run("detects category from component=security", func(t *testing.T) {
		jsonLog := `{"level":"info","message":"Auth event","component":"security"}`

		entry, err := w.parseZerologJSON([]byte(jsonLog))
		require.NoError(t, err)

		assert.Equal(t, storage.LogCategorySecurity, entry.Category)
	})

	t.Run("detects category from component=http", func(t *testing.T) {
		jsonLog := `{"level":"info","message":"HTTP event","component":"http"}`

		entry, err := w.parseZerologJSON([]byte(jsonLog))
		require.NoError(t, err)

		assert.Equal(t, storage.LogCategoryHTTP, entry.Category)
	})

	t.Run("upgrades level to error when error field present", func(t *testing.T) {
		jsonLog := `{"level":"info","message":"Something went wrong","error":"connection refused"}`

		entry, err := w.parseZerologJSON([]byte(jsonLog))
		require.NoError(t, err)

		assert.Equal(t, storage.LogLevelError, entry.Level)
	})

	t.Run("preserves error level when already set", func(t *testing.T) {
		jsonLog := `{"level":"error","message":"Error message","error":"details"}`

		entry, err := w.parseZerologJSON([]byte(jsonLog))
		require.NoError(t, err)

		// Level should stay error, not be changed
		assert.Equal(t, storage.LogLevelError, entry.Level)
	})

	t.Run("stores remaining fields in Fields map", func(t *testing.T) {
		jsonLog := `{"level":"info","message":"Test","custom_field":"value","numeric":123,"nested":{"key":"val"}}`

		entry, err := w.parseZerologJSON([]byte(jsonLog))
		require.NoError(t, err)

		assert.Equal(t, "value", entry.Fields["custom_field"])
		assert.Equal(t, float64(123), entry.Fields["numeric"])
		nested := entry.Fields["nested"].(map[string]any)
		assert.Equal(t, "val", nested["key"])
	})

	t.Run("removes extracted fields from Fields", func(t *testing.T) {
		jsonLog := `{"level":"info","message":"Test","request_id":"req-1","custom":"value"}`

		entry, err := w.parseZerologJSON([]byte(jsonLog))
		require.NoError(t, err)

		// These should NOT be in Fields
		_, hasLevel := entry.Fields["level"]
		_, hasMessage := entry.Fields["message"]
		_, hasRequestID := entry.Fields["request_id"]

		assert.False(t, hasLevel)
		assert.False(t, hasMessage)
		assert.False(t, hasRequestID)

		// This should be in Fields
		assert.Equal(t, "value", entry.Fields["custom"])
	})

	t.Run("returns error for invalid JSON", func(t *testing.T) {
		jsonLog := `{invalid json}`

		_, err := w.parseZerologJSON([]byte(jsonLog))
		require.Error(t, err)
	})

	t.Run("handles empty JSON", func(t *testing.T) {
		jsonLog := `{}`

		entry, err := w.parseZerologJSON([]byte(jsonLog))
		require.NoError(t, err)

		assert.Equal(t, storage.LogCategorySystem, entry.Category)
		// Level stays empty when not specified in JSON (not explicitly set to info)
		assert.Empty(t, entry.Level)
		assert.Empty(t, entry.Message)
	})
}

func TestMultiWriter(t *testing.T) {
	t.Run("writes to all writers", func(t *testing.T) {
		var buf1, buf2 []byte

		w1 := &testWriter{buf: &buf1}
		w2 := &testWriter{buf: &buf2}

		mw := MultiWriter(w1, w2)

		data := []byte("test data")
		n, err := mw.Write(data)

		require.NoError(t, err)
		assert.Equal(t, len(data), n)
		assert.Equal(t, data, buf1)
		assert.Equal(t, data, buf2)
	})

	t.Run("returns error on write failure", func(t *testing.T) {
		w1 := &testWriter{err: assert.AnError}

		mw := MultiWriter(w1)

		_, err := mw.Write([]byte("test"))
		assert.Error(t, err)
	})

	t.Run("handles empty writers list", func(t *testing.T) {
		mw := MultiWriter()

		n, err := mw.Write([]byte("test"))
		require.NoError(t, err)
		assert.Equal(t, 4, n)
	})

	t.Run("returns short write error", func(t *testing.T) {
		w1 := &testWriter{shortWrite: true}

		mw := MultiWriter(w1)

		_, err := mw.Write([]byte("test data"))
		assert.Error(t, err)
	})
}

// testWriter is a simple io.Writer for testing
type testWriter struct {
	buf        *[]byte
	err        error
	shortWrite bool
}

func (w *testWriter) Write(p []byte) (int, error) {
	if w.err != nil {
		return 0, w.err
	}
	if w.shortWrite {
		return 1, nil // Return less than len(p)
	}
	if w.buf != nil {
		*w.buf = append(*w.buf, p...)
	}
	return len(p), nil
}

func TestNewWriter(t *testing.T) {
	// Note: We can't fully test NewWriter without a real Service,
	// but we can test the console configuration logic

	t.Run("console disabled", func(t *testing.T) {
		w := NewWriter(nil, false, "json")
		assert.False(t, w.consoleEnabled)
		assert.Nil(t, w.console)
	})
}
