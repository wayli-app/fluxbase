package logging

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"strings"
	"time"

	"github.com/fluxbase-eu/fluxbase/internal/storage"
	"github.com/rs/zerolog"
)

// Writer is a custom io.Writer that intercepts zerolog output and sends it
// to the central logging service while also writing to console.
type Writer struct {
	service        *Service
	console        io.Writer
	consoleEnabled bool
}

// NewWriter creates a new zerolog writer.
func NewWriter(service *Service, consoleEnabled bool, format string) *Writer {
	var console io.Writer
	if consoleEnabled {
		if format == "console" {
			// Pretty console output
			console = zerolog.ConsoleWriter{
				Out:        os.Stderr,
				TimeFormat: time.RFC3339,
			}
		} else {
			// JSON output
			console = os.Stderr
		}
	}

	return &Writer{
		service:        service,
		console:        console,
		consoleEnabled: consoleEnabled,
	}
}

// Write implements io.Writer. It parses zerolog JSON output and sends it
// to the central logging service.
func (w *Writer) Write(p []byte) (n int, err error) {
	n = len(p)

	// Always write to console if enabled
	if w.consoleEnabled && w.console != nil {
		// Ignore console write errors - they shouldn't prevent logging
		_, _ = w.console.Write(p)
	}

	// Parse the JSON log entry
	entry, parseErr := w.parseZerologJSON(p)
	if parseErr != nil {
		// If we can't parse, just skip sending to backend
		return n, nil
	}

	// Send to logging service
	w.service.Log(context.Background(), entry)

	return n, nil
}

// parseZerologJSON parses zerolog JSON output into a LogEntry.
func (w *Writer) parseZerologJSON(p []byte) (*storage.LogEntry, error) {
	var raw map[string]any
	if err := json.Unmarshal(p, &raw); err != nil {
		return nil, err
	}

	entry := &storage.LogEntry{
		Category:  storage.LogCategorySystem,
		Timestamp: time.Now(),
		Fields:    make(map[string]any),
	}

	// Extract standard zerolog fields
	if level, ok := raw["level"].(string); ok {
		entry.Level = parseLogLevel(level)
		delete(raw, "level")
	}

	if msg, ok := raw["message"].(string); ok {
		entry.Message = msg
		delete(raw, "message")
	} else if msg, ok := raw["msg"].(string); ok {
		entry.Message = msg
		delete(raw, "msg")
	}

	if ts, ok := raw["time"].(string); ok {
		if parsed, err := time.Parse(time.RFC3339, ts); err == nil {
			entry.Timestamp = parsed
		}
		delete(raw, "time")
	}

	// Extract correlation IDs
	if requestID, ok := raw["request_id"].(string); ok {
		entry.RequestID = requestID
		delete(raw, "request_id")
	}
	if traceID, ok := raw["trace_id"].(string); ok {
		entry.TraceID = traceID
		delete(raw, "trace_id")
	}

	// Extract component
	if component, ok := raw["component"].(string); ok {
		entry.Component = component
		delete(raw, "component")
	}

	// Extract user/IP
	if userID, ok := raw["user_id"].(string); ok {
		entry.UserID = userID
		delete(raw, "user_id")
	}
	if ipAddress, ok := raw["ip_address"].(string); ok {
		entry.IPAddress = ipAddress
		delete(raw, "ip_address")
	} else if ipAddress, ok := raw["ip"].(string); ok {
		entry.IPAddress = ipAddress
		delete(raw, "ip")
	}

	// Extract execution fields
	if executionID, ok := raw["execution_id"].(string); ok {
		entry.ExecutionID = executionID
		entry.Category = storage.LogCategoryExecution
		delete(raw, "execution_id")
	}
	if executionType, ok := raw["execution_type"].(string); ok {
		entry.ExecutionType = executionType
		delete(raw, "execution_type")
	}

	// Detect security logs (from SecurityLogger)
	if _, ok := raw["security_event"]; ok {
		entry.Category = storage.LogCategorySecurity
	}

	// Detect HTTP logs (from structured logger middleware)
	// HTTP logs have method and status fields
	if _, hasMethod := raw["method"]; hasMethod {
		if _, hasStatus := raw["status"]; hasStatus {
			entry.Category = storage.LogCategoryHTTP
		}
	}

	// Check component field for category hints
	switch entry.Component {
	case "security":
		entry.Category = storage.LogCategorySecurity
	case "http":
		entry.Category = storage.LogCategoryHTTP
	}

	// Check for error field - indicates this might be an error log
	if _, hasError := raw["error"]; hasError {
		if entry.Level == storage.LogLevelInfo {
			entry.Level = storage.LogLevelError
		}
	}

	// Store remaining fields
	for k, v := range raw {
		entry.Fields[k] = v
	}

	return entry, nil
}

// parseLogLevel converts a zerolog level string to LogLevel.
func parseLogLevel(level string) storage.LogLevel {
	switch strings.ToLower(level) {
	case "trace":
		return storage.LogLevelTrace
	case "debug":
		return storage.LogLevelDebug
	case "info":
		return storage.LogLevelInfo
	case "warn", "warning":
		return storage.LogLevelWarn
	case "error":
		return storage.LogLevelError
	case "fatal":
		return storage.LogLevelFatal
	case "panic":
		return storage.LogLevelPanic
	default:
		return storage.LogLevelInfo
	}
}

// MultiWriter creates an io.Writer that writes to multiple destinations.
// This is useful for writing to both the logging service and the original console.
func MultiWriter(writers ...io.Writer) io.Writer {
	return &multiWriter{writers: writers}
}

type multiWriter struct {
	writers []io.Writer
}

func (mw *multiWriter) Write(p []byte) (n int, err error) {
	for _, w := range mw.writers {
		n, err = w.Write(p)
		if err != nil {
			return
		}
		if n != len(p) {
			err = io.ErrShortWrite
			return
		}
	}
	return len(p), nil
}
