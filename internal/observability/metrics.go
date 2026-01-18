package observability

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
)

var (
	metricsOnce     sync.Once
	metricsInstance *Metrics
)

// Metrics holds all Prometheus metrics for Fluxbase
type Metrics struct {
	// HTTP metrics
	httpRequestsTotal    *prometheus.CounterVec
	httpRequestDuration  *prometheus.HistogramVec
	httpRequestSize      *prometheus.HistogramVec
	httpResponseSize     *prometheus.HistogramVec
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
	storageBytesTotal        *prometheus.CounterVec
	storageOperationsTotal   *prometheus.CounterVec
	storageOperationDuration *prometheus.HistogramVec

	// Auth metrics
	authAttemptsTotal *prometheus.CounterVec
	authSuccessTotal  *prometheus.CounterVec
	authFailureTotal  *prometheus.CounterVec
	authTokensIssued  *prometheus.CounterVec

	// Rate limiting metrics
	rateLimitHitsTotal *prometheus.CounterVec

	// Job metrics
	jobsQueueDepth       *prometheus.GaugeVec
	jobsProcessing       prometheus.Gauge
	jobsCompletedTotal   *prometheus.CounterVec
	jobsFailedTotal      *prometheus.CounterVec
	jobExecutionDuration *prometheus.HistogramVec
	jobWorkersActive     prometheus.Gauge
	jobWorkerUtilization prometheus.Gauge

	// AI Chatbot metrics
	aiChatRequestsTotal     *prometheus.CounterVec
	aiChatRequestDuration   *prometheus.HistogramVec
	aiTokensUsedTotal       *prometheus.CounterVec
	aiSQLQueriesTotal       *prometheus.CounterVec
	aiSQLQueryDuration      *prometheus.HistogramVec
	aiActiveConversations   prometheus.Gauge
	aiWebSocketConnections  prometheus.Gauge
	aiProviderRequestsTotal *prometheus.CounterVec
	aiProviderLatency       *prometheus.HistogramVec

	// System metrics
	systemUptime prometheus.Gauge
}

// NewMetrics creates and registers all Prometheus metrics (singleton)
func NewMetrics() *Metrics {
	metricsOnce.Do(func() {
		metricsInstance = createMetrics()
	})
	return metricsInstance
}

// createMetrics creates all Prometheus metrics
func createMetrics() *Metrics {
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

		// Job metrics
		jobsQueueDepth: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "fluxbase_jobs_queue_depth",
				Help: "Current number of jobs waiting in queue",
			},
			[]string{"namespace", "priority"},
		),
		jobsProcessing: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "fluxbase_jobs_processing",
				Help: "Current number of jobs being processed",
			},
		),
		jobsCompletedTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "fluxbase_jobs_completed_total",
				Help: "Total number of jobs completed successfully",
			},
			[]string{"namespace", "name"},
		),
		jobsFailedTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "fluxbase_jobs_failed_total",
				Help: "Total number of jobs that failed",
			},
			[]string{"namespace", "name", "reason"},
		),
		jobExecutionDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "fluxbase_job_execution_duration_seconds",
				Help:    "Job execution duration in seconds",
				Buckets: []float64{.1, .5, 1, 5, 10, 30, 60, 120, 300, 600},
			},
			[]string{"namespace", "name"},
		),
		jobWorkersActive: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "fluxbase_job_workers_active",
				Help: "Current number of active job workers",
			},
		),
		jobWorkerUtilization: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "fluxbase_job_worker_utilization",
				Help: "Job worker utilization (0.0-1.0)",
			},
		),

		// AI Chatbot metrics
		aiChatRequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "fluxbase_ai_chat_requests_total",
				Help: "Total AI chat requests",
			},
			[]string{"chatbot", "status"}, // status: success, error, rate_limited
		),
		aiChatRequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "fluxbase_ai_chat_request_duration_seconds",
				Help:    "AI chat request duration in seconds",
				Buckets: []float64{0.1, 0.5, 1, 2.5, 5, 10, 30, 60},
			},
			[]string{"chatbot"},
		),
		aiTokensUsedTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "fluxbase_ai_tokens_total",
				Help: "Total AI tokens used",
			},
			[]string{"chatbot", "token_type"}, // token_type: prompt, completion
		),
		aiSQLQueriesTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "fluxbase_ai_sql_queries_total",
				Help: "Total SQL queries generated by AI",
			},
			[]string{"chatbot", "status"}, // status: executed, rejected, error
		),
		aiSQLQueryDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "fluxbase_ai_sql_query_duration_seconds",
				Help:    "AI-generated SQL query execution duration in seconds",
				Buckets: []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5},
			},
			[]string{"chatbot"},
		),
		aiActiveConversations: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "fluxbase_ai_active_conversations",
				Help: "Currently active AI conversations",
			},
		),
		aiWebSocketConnections: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "fluxbase_ai_websocket_connections",
				Help: "Active AI WebSocket connections",
			},
		),
		aiProviderRequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "fluxbase_ai_provider_requests_total",
				Help: "Requests to AI providers",
			},
			[]string{"provider", "status"}, // provider: openai, azure, ollama
		),
		aiProviderLatency: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "fluxbase_ai_provider_latency_seconds",
				Help:    "AI provider response latency in seconds",
				Buckets: []float64{0.1, 0.5, 1, 2, 5, 10, 30},
			},
			[]string{"provider"},
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

// UpdateJobQueueDepth updates the job queue depth metric
// priority should be "high", "normal", or "low"
func (m *Metrics) UpdateJobQueueDepth(namespace, priority string, count int) {
	if namespace == "" {
		namespace = "default"
	}
	m.jobsQueueDepth.WithLabelValues(namespace, priority).Set(float64(count))
}

// UpdateJobsProcessing updates the number of jobs currently being processed
func (m *Metrics) UpdateJobsProcessing(count int) {
	m.jobsProcessing.Set(float64(count))
}

// RecordJobCompleted records a successfully completed job
func (m *Metrics) RecordJobCompleted(namespace, name string, duration time.Duration) {
	if namespace == "" {
		namespace = "default"
	}
	m.jobsCompletedTotal.WithLabelValues(namespace, name).Inc()
	m.jobExecutionDuration.WithLabelValues(namespace, name).Observe(duration.Seconds())
}

// RecordJobFailed records a failed job
// reason should be descriptive like "timeout", "error", "cancelled", "panic"
func (m *Metrics) RecordJobFailed(namespace, name, reason string, duration time.Duration) {
	if namespace == "" {
		namespace = "default"
	}
	m.jobsFailedTotal.WithLabelValues(namespace, name, reason).Inc()
	m.jobExecutionDuration.WithLabelValues(namespace, name).Observe(duration.Seconds())
}

// UpdateJobWorkers updates job worker metrics
// utilization is the ratio of active jobs to total capacity (0.0 to 1.0)
func (m *Metrics) UpdateJobWorkers(activeWorkers int, utilization float64) {
	m.jobWorkersActive.Set(float64(activeWorkers))
	m.jobWorkerUtilization.Set(utilization)
}

// UpdateUptime updates the system uptime metric
func (m *Metrics) UpdateUptime(startTime time.Time) {
	m.systemUptime.Set(time.Since(startTime).Seconds())
}

// RecordAIChatRequest records an AI chat request
func (m *Metrics) RecordAIChatRequest(chatbot, status string, duration time.Duration) {
	m.aiChatRequestsTotal.WithLabelValues(chatbot, status).Inc()
	m.aiChatRequestDuration.WithLabelValues(chatbot).Observe(duration.Seconds())
}

// RecordAITokens records AI token usage
func (m *Metrics) RecordAITokens(chatbot string, promptTokens, completionTokens int) {
	m.aiTokensUsedTotal.WithLabelValues(chatbot, "prompt").Add(float64(promptTokens))
	m.aiTokensUsedTotal.WithLabelValues(chatbot, "completion").Add(float64(completionTokens))
}

// RecordAISQLQuery records an AI-generated SQL query
func (m *Metrics) RecordAISQLQuery(chatbot, status string, duration time.Duration) {
	m.aiSQLQueriesTotal.WithLabelValues(chatbot, status).Inc()
	if status == "executed" {
		m.aiSQLQueryDuration.WithLabelValues(chatbot).Observe(duration.Seconds())
	}
}

// UpdateAIConversations updates the active AI conversations gauge
func (m *Metrics) UpdateAIConversations(count int) {
	m.aiActiveConversations.Set(float64(count))
}

// UpdateAIWebSocketConnections updates the AI WebSocket connections gauge
func (m *Metrics) UpdateAIWebSocketConnections(count int) {
	m.aiWebSocketConnections.Set(float64(count))
}

// RecordAIProviderRequest records an AI provider request
func (m *Metrics) RecordAIProviderRequest(provider, status string, duration time.Duration) {
	m.aiProviderRequestsTotal.WithLabelValues(provider, status).Inc()
	m.aiProviderLatency.WithLabelValues(provider).Observe(duration.Seconds())
}

// RecordRPCExecution records an RPC procedure execution
func (m *Metrics) RecordRPCExecution(procedure, status string, duration time.Duration) {
	// Reuse AI SQL query metrics for RPC since they track similar SQL execution patterns
	m.aiSQLQueriesTotal.WithLabelValues("rpc:"+procedure, status).Inc()
	if status == "success" {
		m.aiSQLQueryDuration.WithLabelValues("rpc:" + procedure).Observe(duration.Seconds())
	}
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

// MetricsServer is a dedicated HTTP server for Prometheus metrics
type MetricsServer struct {
	server *http.Server
	port   int
	path   string
}

// NewMetricsServer creates a new metrics server
func NewMetricsServer(port int, path string) *MetricsServer {
	return &MetricsServer{
		port: port,
		path: path,
	}
}

// Start starts the metrics server on the configured port
func (ms *MetricsServer) Start() error {
	mux := http.NewServeMux()
	mux.Handle(ms.path, promhttp.Handler())

	// Add a simple health check for the metrics server
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	ms.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", ms.port),
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
	}

	log.Info().
		Int("port", ms.port).
		Str("path", ms.path).
		Msg("Starting Prometheus metrics server")

	go func() {
		if err := ms.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error().Err(err).Msg("Metrics server error")
		}
	}()

	return nil
}

// Shutdown gracefully shuts down the metrics server
func (ms *MetricsServer) Shutdown(ctx context.Context) error {
	if ms.server == nil {
		return nil
	}

	log.Info().Msg("Shutting down metrics server")
	return ms.server.Shutdown(ctx)
}
