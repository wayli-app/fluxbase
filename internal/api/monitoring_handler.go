package api

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/database"
	"github.com/nimbleflux/fluxbase/internal/jobs"
	"github.com/nimbleflux/fluxbase/internal/logging"
	"github.com/nimbleflux/fluxbase/internal/middleware"
	"github.com/nimbleflux/fluxbase/internal/realtime"
	"github.com/nimbleflux/fluxbase/internal/storage"
)

// MonitoringHandler handles system monitoring and health check endpoints
type MonitoringHandler struct {
	db              *database.Connection
	realtimeHandler *realtime.RealtimeHandler
	storageProvider storage.Provider
	loggingService  *logging.Service // Optional - may be nil if logging not configured
	jobsStorage     *jobs.Storage    // Optional - may be nil if jobs not enabled
}

// NewMonitoringHandler creates a new monitoring handler
func NewMonitoringHandler(db *database.Connection, realtimeHandler *realtime.RealtimeHandler, storageProvider storage.Provider) *MonitoringHandler {
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

// SetJobsStorage sets the jobs storage for job health monitoring
func (h *MonitoringHandler) SetJobsStorage(jobsStorage *jobs.Storage) {
	h.jobsStorage = jobsStorage
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
func (h *MonitoringHandler) GetMetrics(c fiber.Ctx) error {
	if h.db == nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Database connection not initialized",
		})
	}

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// Database stats
	dbStats := h.db.Stats()
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

	// Storage stats - query database with RLS/tenant context for accurate counts
	storageStats, err := h.getStorageStats(c)
	if err == nil {
		metrics.StorageStats = storageStats
	}

	return c.JSON(metrics)
}

// getStorageStats queries storage.buckets and storage.objects tables with RLS/tenant
// context for accurate counts. Falls back to storage provider if DB query fails.
func (h *MonitoringHandler) getStorageStats(c fiber.Ctx) (*StorageStats, error) {
	ctx := c.RequestCtx()

	// Get tenant context from middleware
	tenantID := middleware.GetTenantIDFromContext(c)

	// Get user role for RLS
	role, _ := c.Locals("user_role").(string)
	userID := middleware.GetUserID(c)

	// Determine DB role for RLS
	dbRole := "authenticated"
	if role == "anon" {
		dbRole = "anon"
	}

	conn, err := h.db.Pool().Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire connection: %w", err)
	}
	defer conn.Release()

	tx, err := conn.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Set RLS context
	claimsMap := map[string]string{"sub": userID, "role": role}
	jwtClaimsBytes, err := json.Marshal(claimsMap)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal JWT claims: %w", err)
	}
	jwtClaims := string(jwtClaimsBytes)
	if _, err := tx.Exec(ctx, "SELECT set_config('request.jwt.claims', $1, true)", jwtClaims); err != nil {
		return nil, fmt.Errorf("failed to set JWT claims: %w", err)
	}
	if tenantID != "" {
		if _, err := tx.Exec(ctx, "SELECT set_config('app.current_tenant_id', $1, true)", tenantID); err != nil {
			return nil, fmt.Errorf("failed to set tenant context: %w", err)
		}
	}
	if _, err := tx.Exec(ctx, fmt.Sprintf("SET LOCAL ROLE %s", quoteIdentifier(dbRole))); err != nil {
		return nil, fmt.Errorf("failed to SET LOCAL ROLE: %w", err)
	}

	// Query bucket count
	var bucketCount int
	if err := tx.QueryRow(ctx, "SELECT count(*) FROM storage.buckets").Scan(&bucketCount); err != nil {
		return nil, fmt.Errorf("failed to count buckets: %w", err)
	}

	// Query file count and total size
	var totalFiles int
	var totalSizeGB float64
	if err := tx.QueryRow(ctx, "SELECT count(*), coalesce(sum(size)::float8 / 1024 / 1024 / 1024, 0) FROM storage.objects WHERE size IS NOT NULL").Scan(&totalFiles, &totalSizeGB); err != nil {
		return nil, fmt.Errorf("failed to count objects: %w", err)
	}

	return &StorageStats{
		TotalBuckets: bucketCount,
		TotalFiles:   totalFiles,
		TotalSizeGB:  totalSizeGB,
	}, nil
}

// GetHealth returns the health status of all system components
// Admin-only endpoint - non-admin users receive 403 Forbidden
func (h *MonitoringHandler) GetHealth(c fiber.Ctx) error {
	if h.db == nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Database connection not initialized",
		})
	}

	health := SystemHealth{
		Status:   "healthy",
		Services: make(map[string]HealthStatus),
	}

	// Check database health
	dbStart := time.Now()
	ctx, cancel := context.WithTimeout(c.RequestCtx(), 5*time.Second)
	defer cancel()

	err := h.db.Pool().Ping(ctx)
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
		_, err := h.storageProvider.ListBuckets(c.RequestCtx())
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

	// Check jobs health (if available)
	if h.jobsStorage != nil {
		stats, err := h.jobsStorage.GetJobStats(c.RequestCtx(), nil)
		if err != nil {
			health.Services["jobs"] = HealthStatus{
				Status:  "degraded",
				Message: err.Error(),
			}
			if health.Status == "healthy" {
				health.Status = "degraded"
			}
		} else {
			// Determine health based on pending/running job count
			pendingJobs := stats.PendingJobs + stats.RunningJobs

			jobsStatus := "healthy"
			jobsMessage := "Processing queue"

			// Consider degraded if too many pending jobs
			if pendingJobs > 100 {
				jobsStatus = "degraded"
				jobsMessage = fmt.Sprintf("%d pending jobs", pendingJobs)
			}

			health.Services["jobs"] = HealthStatus{
				Status:  jobsStatus,
				Message: jobsMessage,
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
func (h *MonitoringHandler) GetLogs(c fiber.Ctx) error {
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
	limit := fiber.Query[int](c, "limit", 100)
	if limit > 1000 {
		limit = 1000 // Cap at 1000
	}
	opts.Limit = limit
	opts.Offset = fiber.Query[int](c, "offset", 0)

	// Query logs from storage
	result, err := h.loggingService.Storage().Query(middleware.CtxWithTenant(c), opts)
	if err != nil {
		log.Error().Err(err).Msg("Failed to query logs")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal error",
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

// fiber:context-methods migrated
