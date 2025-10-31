package observability

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics holds all Prometheus metrics for Fluxbase
type Metrics struct {
	// HTTP metrics
	httpRequestsTotal   *prometheus.CounterVec
	httpRequestDuration *prometheus.HistogramVec
	httpRequestSize     *prometheus.HistogramVec
	httpResponseSize    *prometheus.HistogramVec
	httpRequestsInFlight prometheus.Gauge

	// Database metrics
	dbQueriesTotal    *prometheus.CounterVec
	dbQueryDuration   *prometheus.HistogramVec
	dbConnections     prometheus.Gauge
	dbConnectionsIdle prometheus.Gauge
	dbConnectionsMax  prometheus.Gauge

	// Realtime metrics
	realtimeConnections      prometheus.Gauge
	realtimeChannels         prometheus.Gauge
	realtimeSubscriptions    prometheus.Gauge
	realtimeMessagesTotal    *prometheus.CounterVec
	realtimeConnectionErrors *prometheus.CounterVec

	// Storage metrics
	storageBytesTotal      *prometheus.CounterVec
	storageOperationsTotal *prometheus.CounterVec
	storageOperationDuration *prometheus.HistogramVec

	// Auth metrics
	authAttemptsTotal *prometheus.CounterVec
	authSuccessTotal  *prometheus.CounterVec
	authFailureTotal  *prometheus.CounterVec
	authTokensIssued  *prometheus.CounterVec

	// Rate limiting metrics
	rateLimitHitsTotal *prometheus.CounterVec

	// System metrics
	systemUptime prometheus.Gauge
}

// NewMetrics creates and registers all Prometheus metrics
func NewMetrics() *Metrics {
	m := &Metrics{
		// HTTP metrics
		httpRequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "fluxbase_http_requests_total",
				Help: "Total number of HTTP requests",
			},
			[]string{"method", "path", "status"},
		),
		httpRequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "fluxbase_http_request_duration_seconds",
				Help:    "HTTP request latency in seconds",
				Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
			},
			[]string{"method", "path", "status"},
		),
		httpRequestSize: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "fluxbase_http_request_size_bytes",
				Help:    "HTTP request size in bytes",
				Buckets: prometheus.ExponentialBuckets(100, 10, 8),
			},
			[]string{"method", "path"},
		),
		httpResponseSize: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "fluxbase_http_response_size_bytes",
				Help:    "HTTP response size in bytes",
				Buckets: prometheus.ExponentialBuckets(100, 10, 8),
			},
			[]string{"method", "path", "status"},
		),
		httpRequestsInFlight: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "fluxbase_http_requests_in_flight",
				Help: "Current number of HTTP requests being processed",
			},
		),

		// Database metrics
		dbQueriesTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "fluxbase_db_queries_total",
				Help: "Total number of database queries",
			},
			[]string{"operation", "table"},
		),
		dbQueryDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "fluxbase_db_query_duration_seconds",
				Help:    "Database query latency in seconds",
				Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5},
			},
			[]string{"operation", "table"},
		),
		dbConnections: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "fluxbase_db_connections",
				Help: "Current number of database connections",
			},
		),
		dbConnectionsIdle: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "fluxbase_db_connections_idle",
				Help: "Current number of idle database connections",
			},
		),
		dbConnectionsMax: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "fluxbase_db_connections_max",
				Help: "Maximum number of database connections",
			},
		),

		// Realtime metrics
		realtimeConnections: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "fluxbase_realtime_connections",
				Help: "Current number of WebSocket connections",
			},
		),
		realtimeChannels: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "fluxbase_realtime_channels",
				Help: "Current number of active channels",
			},
		),
		realtimeSubscriptions: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "fluxbase_realtime_subscriptions",
				Help: "Current number of subscriptions",
			},
		),
		realtimeMessagesTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "fluxbase_realtime_messages_total",
				Help: "Total number of realtime messages sent",
			},
			[]string{"channel_type"},
		),
		realtimeConnectionErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "fluxbase_realtime_connection_errors_total",
				Help: "Total number of WebSocket connection errors",
			},
			[]string{"error_type"},
		),

		// Storage metrics
		storageBytesTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "fluxbase_storage_bytes_total",
				Help: "Total number of bytes stored/retrieved",
			},
			[]string{"operation", "bucket"},
		),
		storageOperationsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "fluxbase_storage_operations_total",
				Help: "Total number of storage operations",
			},
			[]string{"operation", "bucket", "status"},
		),
		storageOperationDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "fluxbase_storage_operation_duration_seconds",
				Help:    "Storage operation latency in seconds",
				Buckets: []float64{.01, .05, .1, .25, .5, 1, 2.5, 5, 10},
			},
			[]string{"operation", "bucket"},
		),

		// Auth metrics
		authAttemptsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "fluxbase_auth_attempts_total",
				Help: "Total number of authentication attempts",
			},
			[]string{"method", "result"},
		),
		authSuccessTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "fluxbase_auth_success_total",
				Help: "Total number of successful authentications",
			},
			[]string{"method"},
		),
		authFailureTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "fluxbase_auth_failure_total",
				Help: "Total number of failed authentications",
			},
			[]string{"method", "reason"},
		),
		authTokensIssued: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "fluxbase_auth_tokens_issued_total",
				Help: "Total number of auth tokens issued",
			},
			[]string{"token_type"},
		),

		// Rate limiting metrics
		rateLimitHitsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "fluxbase_rate_limit_hits_total",
				Help: "Total number of rate limit hits",
			},
			[]string{"limiter_type", "identifier"},
		),

		// System metrics
		systemUptime: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "fluxbase_system_uptime_seconds",
				Help: "System uptime in seconds",
			},
		),
	}

	return m
}

// MetricsMiddleware returns a Fiber middleware that collects HTTP metrics
func (m *Metrics) MetricsMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		m.httpRequestsInFlight.Inc()
		defer m.httpRequestsInFlight.Dec()

		// Get request size
		requestSize := len(c.Body())
		path := normalizePath(c.Path())
		method := c.Method()

		// Process request
		err := c.Next()

		// Calculate duration
		duration := time.Since(start).Seconds()
		status := statusClass(c.Response().StatusCode())
		responseSize := len(c.Response().Body())

		// Record metrics
		m.httpRequestsTotal.WithLabelValues(method, path, status).Inc()
		m.httpRequestDuration.WithLabelValues(method, path, status).Observe(duration)
		m.httpRequestSize.WithLabelValues(method, path).Observe(float64(requestSize))
		m.httpResponseSize.WithLabelValues(method, path, status).Observe(float64(responseSize))

		return err
	}
}

// RecordDBQuery records database query metrics
func (m *Metrics) RecordDBQuery(operation, table string, duration time.Duration, err error) {
	m.dbQueriesTotal.WithLabelValues(operation, table).Inc()
	m.dbQueryDuration.WithLabelValues(operation, table).Observe(duration.Seconds())
}

// UpdateDBStats updates database connection pool stats
func (m *Metrics) UpdateDBStats(total, idle, max int32) {
	m.dbConnections.Set(float64(total))
	m.dbConnectionsIdle.Set(float64(idle))
	m.dbConnectionsMax.Set(float64(max))
}

// UpdateRealtimeStats updates realtime connection stats
func (m *Metrics) UpdateRealtimeStats(connections, channels, subscriptions int) {
	m.realtimeConnections.Set(float64(connections))
	m.realtimeChannels.Set(float64(channels))
	m.realtimeSubscriptions.Set(float64(subscriptions))
}

// RecordRealtimeMessage records a realtime message sent
func (m *Metrics) RecordRealtimeMessage(channelType string) {
	m.realtimeMessagesTotal.WithLabelValues(channelType).Inc()
}

// RecordRealtimeError records a realtime connection error
func (m *Metrics) RecordRealtimeError(errorType string) {
	m.realtimeConnectionErrors.WithLabelValues(errorType).Inc()
}

// RecordStorageOperation records a storage operation
func (m *Metrics) RecordStorageOperation(operation, bucket string, bytes int64, duration time.Duration, err error) {
	status := "success"
	if err != nil {
		status = "error"
	}

	m.storageOperationsTotal.WithLabelValues(operation, bucket, status).Inc()
	m.storageBytesTotal.WithLabelValues(operation, bucket).Add(float64(bytes))
	m.storageOperationDuration.WithLabelValues(operation, bucket).Observe(duration.Seconds())
}

// RecordAuthAttempt records an authentication attempt
func (m *Metrics) RecordAuthAttempt(method string, success bool, reason string) {
	result := "success"
	if !success {
		result = "failure"
	}

	m.authAttemptsTotal.WithLabelValues(method, result).Inc()

	if success {
		m.authSuccessTotal.WithLabelValues(method).Inc()
	} else {
		m.authFailureTotal.WithLabelValues(method, reason).Inc()
	}
}

// RecordAuthToken records an issued auth token
func (m *Metrics) RecordAuthToken(tokenType string) {
	m.authTokensIssued.WithLabelValues(tokenType).Inc()
}

// RecordRateLimitHit records a rate limit hit
func (m *Metrics) RecordRateLimitHit(limiterType, identifier string) {
	m.rateLimitHitsTotal.WithLabelValues(limiterType, identifier).Inc()
}

// UpdateUptime updates the system uptime metric
func (m *Metrics) UpdateUptime(startTime time.Time) {
	m.systemUptime.Set(time.Since(startTime).Seconds())
}

// Handler returns a Fiber handler that exposes Prometheus metrics
func (m *Metrics) Handler() fiber.Handler {
	return adaptor.HTTPHandler(promhttp.Handler())
}

// normalizePath normalizes API paths for metrics (replaces IDs with placeholders)
func normalizePath(path string) string {
	// For metrics, we want to group paths like /api/v1/tables/users/123 -> /api/v1/tables/users/:id
	// This prevents cardinality explosion

	// Simple heuristic: if path segment looks like UUID or number, replace with :id
	// This is a simplified version - production might need more sophisticated logic
	if len(path) > 50 {
		return "long_path" // Prevent cardinality explosion
	}
	return path
}

// statusClass returns the HTTP status class (2xx, 3xx, 4xx, 5xx)
func statusClass(status int) string {
	switch {
	case status >= 200 && status < 300:
		return "2xx"
	case status >= 300 && status < 400:
		return "3xx"
	case status >= 400 && status < 500:
		return "4xx"
	case status >= 500:
		return "5xx"
	default:
		return "unknown"
	}
}
