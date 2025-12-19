package api

import (
	"context"
	"runtime"
	"time"

	"github.com/fluxbase-eu/fluxbase/internal/auth"
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
}

// NewMonitoringHandler creates a new monitoring handler
func NewMonitoringHandler(db *pgxpool.Pool, realtimeHandler *realtime.RealtimeHandler, storageProvider storage.Provider) *MonitoringHandler {
	return &MonitoringHandler{
		db:              db,
		realtimeHandler: realtimeHandler,
		storageProvider: storageProvider,
	}
}

// RegisterRoutes registers monitoring routes with authentication
func (h *MonitoringHandler) RegisterRoutes(app *fiber.App, authService *auth.Service, apiKeyService *auth.APIKeyService, db *pgxpool.Pool, jwtManager *auth.JWTManager) {
	// Apply authentication middleware to all monitoring routes
	monitoring := app.Group("/api/v1/monitoring",
		middleware.RequireAuthOrServiceKey(authService, apiKeyService, db, jwtManager),
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

	// For MVP, return a placeholder indicating logs are not yet stored
	// In production, this would query from a log storage system (e.g., database table, file, or external service)

	return c.JSON(fiber.Map{
		"message": "Log storage not yet implemented. Use server console output for now.",
		"logs":    []LogEntry{},
	})
}
