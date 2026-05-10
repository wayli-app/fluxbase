package database

import (
	"context"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/logutil"
	"github.com/nimbleflux/fluxbase/internal/observability"
)

func getCallerFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if v, ok := ctx.Value(callerKey{}).(string); ok && v != "" {
		return v
	}
	return ""
}

func getCallerFromRuntime() string {
	for skip := 3; skip <= 8; skip++ {
		if _, file, _, ok := runtime.Caller(skip); ok {
			idx := strings.LastIndex(file, "/internal/")
			if idx >= 0 {
				return file[idx+1:]
			}
		}
	}
	return ""
}

type slowQueryEntry struct {
	count     int
	firstSeen time.Time
}

type slowQueryTracker struct {
	mu      sync.Mutex
	entries map[string]*slowQueryEntry
	maxAge  time.Duration
	stopCh  chan struct{}
}

func newSlowQueryTracker() *slowQueryTracker {
	t := &slowQueryTracker{
		entries: make(map[string]*slowQueryEntry),
		maxAge:  1 * time.Hour,
		stopCh:  make(chan struct{}),
	}
	go t.cleanupLoop()
	return t
}

func (t *slowQueryTracker) stop() {
	close(t.stopCh)
}

func (t *slowQueryTracker) record(queryKey string) int {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	e, ok := t.entries[queryKey]
	if !ok {
		t.entries[queryKey] = &slowQueryEntry{count: 1, firstSeen: now}
		return 1
	}
	e.count++
	return e.count
}

func (t *slowQueryTracker) cleanupLoop() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			t.mu.Lock()
			now := time.Now()
			for k, e := range t.entries {
				if now.Sub(e.firstSeen) > t.maxAge {
					delete(t.entries, k)
				}
			}
			t.mu.Unlock()
		case <-t.stopCh:
			return
		}
	}
}

const slowQueryTruncationLimit = 500

// SetMetrics sets the metrics instance for recording database metrics
func (c *Connection) SetMetrics(m *observability.Metrics) {
	c.metrics = m
}

func (c *Connection) logSlowQuery(ctx context.Context, sql string, duration time.Duration, opType string) {
	if duration <= c.slowQueryThreshold {
		return
	}

	operation := ExtractOperation(sql)
	table := ExtractTableName(sql)
	sanitizedQuery := truncateQuery(logutil.SanitizeSQL(sql), slowQueryTruncationLimit)

	queryKey := operation + ":" + table
	occurrences := 1
	if c.slowQueryTracker != nil {
		occurrences = c.slowQueryTracker.record(queryKey)
	}

	caller := getCallerFromContext(ctx)
	if caller == "" {
		caller = getCallerFromRuntime()
	}

	evt := log.Warn().
		Dur("duration", duration).
		Int64("duration_ms", duration.Milliseconds()).
		Str("operation", operation).
		Str("table", table).
		Str("query", sanitizedQuery).
		Int("occurrences", occurrences).
		Bool("slow_query", true)

	if caller != "" {
		evt = evt.Str("caller", caller)
	}

	evt.Msg("Slow query detected")
}

// truncateQuery truncates a SQL query to a maximum length for logging
func truncateQuery(query string, maxLen int) string {
	if len(query) <= maxLen {
		return query
	}
	return query[:maxLen] + "... (truncated)"
}
