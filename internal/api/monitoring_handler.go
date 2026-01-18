package api

import (
	"context"
	"runtime"
	"time"

	"github.com/fluxbase-eu/fluxbase/internal/auth"
	"github.com/fluxbase-eu/fluxbase/internal/logging"
	"github.com/fluxbase-eu/fluxbase/internal/middleware"
	"github.com/fluxbase-eu/fluxbase/internal/realtime"
	"github.com/fluxbase-eu/fluxbase/internal/storage"
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
)

// MonitoringHandler handles system monitoring and health check endpoints
type MonitoringHandler struct {
	db              *pgxpool.Pool
	realtimeHandler *realtime.RealtimeHandler
	storageProvider storage.Provider
	loggingService  *logging.Service // Optional - may be nil if logging not configured
}

// NewMonitoringHandler creates a new monitoring handler
func NewMonitoringHandler(db *pgxpool.Pool, realtimeHandler *realtime.RealtimeHandler, storageProvider storage.Provider) *MonitoringHandler {
	return &MonitoringHandler{
		db:              db,
		realtimeHandler: realtimeHandler,
		storageProvider: storageProvider,
	}
}

// SetLoggingService sets the logging service for log queries
func (h *MonitoringHandler) SetLoggingService(loggingService *logging.Service) {
	h.loggingService = loggingService
}

// RegisterRoutes registers monitoring routes with authentication
func (h *MonitoringHandler) RegisterRoutes(app *fiber.App, authService *auth.Service, clientKeyService *auth.ClientKeyService, db *pgxpool.Pool, jwtManager *auth.JWTManager) {
	// Apply authentication middleware to all monitoring routes
	monitoring := app.Group("/api/v1/monitoring",
		middleware.RequireAuthOrServiceKey(authService, clientKeyService, db, jwtManager),
	)

	// All monitoring routes require read:monitoring scope
	monitoring.Get("/metrics", middleware.RequireScope(auth.ScopeMonitoringRead), h.GetMetrics)
	monitoring.Get("/health", middleware.RequireScope(auth.ScopeMonitoringRead), h.GetHealth)
	monitoring.Get("/logs", middleware.RequireScope(auth.ScopeMonitoringRead), h.GetLogs)
}

// SystemMetrics represents system-wide metrics
type SystemMetrics struct {
	// System info
	Uptime       int64  `json:"uptime_seconds"`
	GoVersion    string `json:"go_version"`
	NumGoroutine int    `json:"num_goroutines"`

	// Memory stats
	MemoryAllocMB      uint64  `json:"memory_alloc_mb"`
	MemoryTotalAllocMB uint64  `json:"memory_total_alloc_mb"`
	MemorySysMB        uint64  `json:"memory_sys_mb"`
	NumGC              uint32  `json:"num_gc"`
	GCPauseMS          float64 `json:"gc_pause_ms"`

	// Database stats
	DatabaseStats DatabaseStats `json:"database"`

	// Realtime stats
	RealtimeStats RealtimeStats `json:"realtime"`

	// Storage stats (if available)
	StorageStats *StorageStats `json:"storage,omitempty"`
}

// DatabaseStats represents database connection pool stats
type DatabaseStats struct {
	AcquireCount            int64   `json:"acquire_count"`
	AcquiredConns           int32   `json:"acquired_conns"`
	CanceledAcquireCount    int64   `json:"canceled_acquire_count"`
	ConstructingConns       int32   `json:"constructing_conns"`
	EmptyAcquireCount       int64   `json:"empty_acquire_count"`
	IdleConns               int32   `json:"idle_conns"`
	MaxConns                int32   `json:"max_conns"`
	TotalConns              int32   `json:"total_conns"`
	NewConnsCount           int64   `json:"new_conns_count"`
	MaxLifetimeDestroyCount int64   `json:"max_lifetime_destroy_count"`
	MaxIdleDestroyCount     int64   `json:"max_idle_destroy_count"`
	AcquireDurationMS       float64 `json:"acquire_duration_ms"`
}

// RealtimeStats represents realtime connection stats
type RealtimeStats struct {
	TotalConnections   int `json:"total_connections"`
	ActiveChannels     int `json:"active_channels"`
	TotalSubscriptions int `json:"total_subscriptions"`
}

// StorageStats represents storage usage stats
type StorageStats struct {
	TotalBuckets int     `json:"total_buckets"`
	TotalFiles   int     `json:"total_files"`
	TotalSizeGB  float64 `json:"total_size_gb"`
}

// HealthStatus represents the health status of a component
type HealthStatus struct {
	Status  string `json:"status"` // "healthy", "degraded", "unhealthy"
	Message string `json:"message,omitempty"`
	Latency int64  `json:"latency_ms,omitempty"`
}

// SystemHealth represents the health of all system components
type SystemHealth struct {
	Status   string                  `json:"status"` // "healthy", "degraded", "unhealthy"
	Services map[string]HealthStatus `json:"services"`
}

var startTime = time.Now()

// GetMetrics returns system metrics
// Admin-only endpoint - non-admin users receive 403 Forbidden
func (h *MonitoringHandler) GetMetrics(c *fiber.Ctx) error {
	// Check if user has admin role
	role, _ := c.Locals("user_role").(string)
	if role != "admin" && role != "dashboard_admin" && role != "service_role" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Admin access required to view system metrics",
		})
	}

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// Database stats
	dbStats := h.db.Stat()
	dbAcquireDuration := dbStats.AcquireDuration()

	// Realtime stats
	realtimeStats := h.realtimeHandler.GetStats()
	totalConnections := 0
	activeChannels := 0
	totalSubscriptions := 0

	if conns, ok := realtimeStats["connections"].(int); ok {
		totalConnections = conns
	}
	if channels, ok := realtimeStats["channels"].(int); ok {
		activeChannels = channels
	}
	if subs, ok := realtimeStats["subscriptions"].(int); ok {
		totalSubscriptions = subs
	}

	metrics := SystemMetrics{
		Uptime:       int64(time.Since(startTime).Seconds()),
		GoVersion:    runtime.Version(),
		NumGoroutine: runtime.NumGoroutine(),

		MemoryAllocMB:      m.Alloc / 1024 / 1024,
		MemoryTotalAllocMB: m.TotalAlloc / 1024 / 1024,
		MemorySysMB:        m.Sys / 1024 / 1024,
		NumGC:              m.NumGC,
		GCPauseMS:          float64(m.PauseNs[(m.NumGC+255)%256]) / 1000000,

		DatabaseStats: DatabaseStats{
			AcquireCount:            dbStats.AcquireCount(),
			AcquiredConns:           dbStats.AcquiredConns(),
			CanceledAcquireCount:    dbStats.CanceledAcquireCount(),
			ConstructingConns:       dbStats.ConstructingConns(),
			EmptyAcquireCount:       dbStats.EmptyAcquireCount(),
			IdleConns:               dbStats.IdleConns(),
			MaxConns:                dbStats.MaxConns(),
			TotalConns:              dbStats.TotalConns(),
			NewConnsCount:           dbStats.NewConnsCount(),
			MaxLifetimeDestroyCount: dbStats.MaxLifetimeDestroyCount(),
			MaxIdleDestroyCount:     dbStats.MaxIdleDestroyCount(),
			AcquireDurationMS:       float64(dbAcquireDuration.Milliseconds()),
		},

		RealtimeStats: RealtimeStats{
			TotalConnections:   totalConnections,
			ActiveChannels:     activeChannels,
			TotalSubscriptions: totalSubscriptions,
		},
	}

	// Storage stats (if available)
	if h.storageProvider != nil {
		buckets, err := h.storageProvider.ListBuckets(c.Context())
		if err == nil {
			totalFiles := 0
			var totalSize int64

			for _, bucket := range buckets {
				result, err := h.storageProvider.List(c.Context(), bucket, &storage.ListOptions{MaxKeys: 10000})
				if err == nil && result != nil {
					totalFiles += len(result.Objects)
					for _, file := range result.Objects {
						totalSize += file.Size
					}
				}
			}

			metrics.StorageStats = &StorageStats{
				TotalBuckets: len(buckets),
				TotalFiles:   totalFiles,
				TotalSizeGB:  float64(totalSize) / 1024 / 1024 / 1024,
			}
		}
	}

	return c.JSON(metrics)
}

// GetHealth returns the health status of all system components
// Admin-only endpoint - non-admin users receive 403 Forbidden
func (h *MonitoringHandler) GetHealth(c *fiber.Ctx) error {
	// Check if user has admin role
	role, _ := c.Locals("user_role").(string)
	if role != "admin" && role != "dashboard_admin" && role != "service_role" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Admin access required to view system health",
		})
	}

	health := SystemHealth{
		Status:   "healthy",
		Services: make(map[string]HealthStatus),
	}

	// Check database health
	dbStart := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := h.db.Ping(ctx)
	dbLatency := time.Since(dbStart).Milliseconds()

	if err != nil {
		health.Services["database"] = HealthStatus{
			Status:  "unhealthy",
			Message: err.Error(),
			Latency: dbLatency,
		}
		health.Status = "unhealthy"
	} else {
		health.Services["database"] = HealthStatus{
			Status:  "healthy",
			Latency: dbLatency,
		}
	}

	// Check realtime health
	health.Services["realtime"] = HealthStatus{
		Status:  "healthy",
		Message: "WebSocket server running",
		Latency: 0,
	}

	// Check storage health (if available)
	if h.storageProvider != nil {
		storageStart := time.Now()
		_, err := h.storageProvider.ListBuckets(c.Context())
		storageLatency := time.Since(storageStart).Milliseconds()

		if err != nil {
			health.Services["storage"] = HealthStatus{
				Status:  "degraded",
				Message: err.Error(),
				Latency: storageLatency,
			}
			if health.Status == "healthy" {
				health.Status = "degraded"
			}
		} else {
			health.Services["storage"] = HealthStatus{
				Status:  "healthy",
				Latency: storageLatency,
			}
		}
	}

	// Overall health based on individual services
	if health.Status == "unhealthy" {
		c.Status(fiber.StatusServiceUnavailable)
	}

	return c.JSON(health)
}

// LogEntry represents a log entry
type LogEntry struct {
	Timestamp time.Time              `json:"timestamp"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Module    string                 `json:"module,omitempty"`
	Error     string                 `json:"error,omitempty"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
}

// GetLogs returns recent application logs
// Admin-only endpoint - non-admin users receive 403 Forbidden
func (h *MonitoringHandler) GetLogs(c *fiber.Ctx) error {
	// Check if user has admin role
	role, _ := c.Locals("user_role").(string)
	if role != "admin" && role != "dashboard_admin" && role != "service_role" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Admin access required to view logs",
		})
	}

	// Check if logging service is available
	if h.loggingService == nil {
		return c.JSON(fiber.Map{
			"message": "Logging service not configured. Enable logging in configuration to view logs.",
			"logs":    []LogEntry{},
		})
	}

	// Parse query parameters
	opts := storage.LogQueryOptions{}

	// Parse level filter
	if level := c.Query("level"); level != "" {
		opts.Levels = []storage.LogLevel{storage.LogLevel(level)}
	}

	// Parse category filter
	if category := c.Query("category"); category != "" {
		opts.Category = storage.LogCategory(category)
	}

	// Parse component filter
	if component := c.Query("component"); component != "" {
		opts.Component = component
	}

	// Parse search text
	if search := c.Query("search"); search != "" {
		opts.Search = search
	}

	// Parse time range - default to last hour
	if startTime := c.Query("start_time"); startTime != "" {
		if t, err := time.Parse(time.RFC3339, startTime); err == nil {
			opts.StartTime = t
		}
	} else {
		opts.StartTime = time.Now().Add(-1 * time.Hour)
	}

	if endTime := c.Query("end_time"); endTime != "" {
		if t, err := time.Parse(time.RFC3339, endTime); err == nil {
			opts.EndTime = t
		}
	}

	// Parse pagination
	limit := c.QueryInt("limit", 100)
	if limit > 1000 {
		limit = 1000 // Cap at 1000
	}
	opts.Limit = limit
	opts.Offset = c.QueryInt("offset", 0)

	// Query logs from storage
	result, err := h.loggingService.Storage().Query(c.Context(), opts)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to query logs: " + err.Error(),
		})
	}

	// Convert to response format
	logs := make([]LogEntry, 0, len(result.Entries))
	for _, entry := range result.Entries {
		// Extract error from fields if present
		errStr := ""
		if entry.Fields != nil {
			if e, ok := entry.Fields["error"].(string); ok {
				errStr = e
			}
		}
		logs = append(logs, LogEntry{
			Timestamp: entry.Timestamp,
			Level:     string(entry.Level),
			Message:   entry.Message,
			Module:    entry.Component,
			Error:     errStr,
			Fields:    entry.Fields,
		})
	}

	return c.JSON(fiber.Map{
		"logs":    logs,
		"total":   result.TotalCount,
		"limit":   limit,
		"offset":  opts.Offset,
		"hasMore": result.TotalCount > int64(opts.Offset+len(logs)),
	})
}
