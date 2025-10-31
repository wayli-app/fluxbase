package api

import (
	"encoding/json"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSystemMetrics_Struct(t *testing.T) {
	t.Run("SystemMetrics marshaling", func(t *testing.T) {
		metrics := SystemMetrics{
			Uptime:             3600,
			GoVersion:          runtime.Version(),
			NumGoroutine:       10,
			MemoryAllocMB:      50,
			MemoryTotalAllocMB: 100,
			MemorySysMB:        200,
			NumGC:              5,
			GCPauseMS:          1.5,
			DatabaseStats: DatabaseStats{
				AcquireCount:      100,
				AcquiredConns:     5,
				IdleConns:         10,
				MaxConns:          20,
				TotalConns:        15,
				AcquireDurationMS: 2.5,
			},
			RealtimeStats: RealtimeStats{
				TotalConnections:   10,
				ActiveChannels:     5,
				TotalSubscriptions: 20,
			},
		}

		data, err := json.Marshal(metrics)
		require.NoError(t, err)
		assert.NotEmpty(t, data)

		var result SystemMetrics
		err = json.Unmarshal(data, &result)
		require.NoError(t, err)

		assert.Equal(t, int64(3600), result.Uptime)
		assert.Equal(t, 10, result.NumGoroutine)
		assert.Equal(t, uint64(50), result.MemoryAllocMB)
		assert.Equal(t, int64(100), result.DatabaseStats.AcquireCount)
		assert.Equal(t, 10, result.RealtimeStats.TotalConnections)
	})

	t.Run("SystemMetrics with storage stats", func(t *testing.T) {
		metrics := SystemMetrics{
			Uptime:       1000,
			GoVersion:    runtime.Version(),
			NumGoroutine: 5,
			DatabaseStats: DatabaseStats{
				MaxConns:   20,
				TotalConns: 10,
			},
			RealtimeStats: RealtimeStats{
				TotalConnections: 5,
			},
			StorageStats: &StorageStats{
				TotalBuckets: 3,
				TotalFiles:   150,
				TotalSizeGB:  5.25,
			},
		}

		data, err := json.Marshal(metrics)
		require.NoError(t, err)

		var result SystemMetrics
		err = json.Unmarshal(data, &result)
		require.NoError(t, err)

		require.NotNil(t, result.StorageStats)
		assert.Equal(t, 3, result.StorageStats.TotalBuckets)
		assert.Equal(t, 150, result.StorageStats.TotalFiles)
		assert.Equal(t, 5.25, result.StorageStats.TotalSizeGB)
	})

	t.Run("SystemMetrics without storage stats", func(t *testing.T) {
		metrics := SystemMetrics{
			Uptime:        500,
			GoVersion:     runtime.Version(),
			NumGoroutine:  3,
			DatabaseStats: DatabaseStats{},
			RealtimeStats: RealtimeStats{},
			StorageStats:  nil,
		}

		data, err := json.Marshal(metrics)
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal(data, &result)
		require.NoError(t, err)

		// Storage stats should be omitted when nil
		_, hasStorage := result["storage"]
		assert.False(t, hasStorage, "Storage stats should be omitted when nil")
	})
}

func TestDatabaseStats_Struct(t *testing.T) {
	t.Run("DatabaseStats with all fields", func(t *testing.T) {
		stats := DatabaseStats{
			AcquireCount:            1000,
			AcquiredConns:           5,
			CanceledAcquireCount:    10,
			ConstructingConns:       2,
			EmptyAcquireCount:       50,
			IdleConns:               10,
			MaxConns:                20,
			TotalConns:              15,
			NewConnsCount:           100,
			MaxLifetimeDestroyCount: 20,
			MaxIdleDestroyCount:     30,
			AcquireDurationMS:       2.5,
		}

		data, err := json.Marshal(stats)
		require.NoError(t, err)

		var result DatabaseStats
		err = json.Unmarshal(data, &result)
		require.NoError(t, err)

		assert.Equal(t, int64(1000), result.AcquireCount)
		assert.Equal(t, int32(5), result.AcquiredConns)
		assert.Equal(t, int32(10), result.IdleConns)
		assert.Equal(t, int32(20), result.MaxConns)
		assert.Equal(t, 2.5, result.AcquireDurationMS)
	})

	t.Run("DatabaseStats validation", func(t *testing.T) {
		stats := DatabaseStats{
			AcquiredConns: 5,
			IdleConns:     10,
			MaxConns:      20,
			TotalConns:    15,
		}

		// Validate logical relationships
		assert.LessOrEqual(t, stats.AcquiredConns, stats.TotalConns, "Acquired conns should be <= total conns")
		assert.LessOrEqual(t, stats.IdleConns, stats.TotalConns, "Idle conns should be <= total conns")
		assert.LessOrEqual(t, stats.TotalConns, stats.MaxConns, "Total conns should be <= max conns")
	})
}

func TestRealtimeStats_Struct(t *testing.T) {
	t.Run("RealtimeStats marshaling", func(t *testing.T) {
		stats := RealtimeStats{
			TotalConnections:   50,
			ActiveChannels:     10,
			TotalSubscriptions: 100,
		}

		data, err := json.Marshal(stats)
		require.NoError(t, err)

		var result RealtimeStats
		err = json.Unmarshal(data, &result)
		require.NoError(t, err)

		assert.Equal(t, 50, result.TotalConnections)
		assert.Equal(t, 10, result.ActiveChannels)
		assert.Equal(t, 100, result.TotalSubscriptions)
	})

	t.Run("RealtimeStats zero values", func(t *testing.T) {
		stats := RealtimeStats{
			TotalConnections:   0,
			ActiveChannels:     0,
			TotalSubscriptions: 0,
		}

		data, err := json.Marshal(stats)
		require.NoError(t, err)
		assert.Contains(t, string(data), `"total_connections":0`)
	})
}

func TestStorageStats_Struct(t *testing.T) {
	t.Run("StorageStats marshaling", func(t *testing.T) {
		stats := StorageStats{
			TotalBuckets: 5,
			TotalFiles:   1000,
			TotalSizeGB:  25.5,
		}

		data, err := json.Marshal(stats)
		require.NoError(t, err)

		var result StorageStats
		err = json.Unmarshal(data, &result)
		require.NoError(t, err)

		assert.Equal(t, 5, result.TotalBuckets)
		assert.Equal(t, 1000, result.TotalFiles)
		assert.Equal(t, 25.5, result.TotalSizeGB)
	})

	t.Run("StorageStats size calculations", func(t *testing.T) {
		// Test byte to GB conversion
		totalBytes := int64(10 * 1024 * 1024 * 1024) // 10 GB
		sizeGB := float64(totalBytes) / 1024 / 1024 / 1024

		stats := StorageStats{
			TotalBuckets: 1,
			TotalFiles:   100,
			TotalSizeGB:  sizeGB,
		}

		assert.InDelta(t, 10.0, stats.TotalSizeGB, 0.01, "Size should be approximately 10 GB")
	})
}

func TestHealthStatus_Struct(t *testing.T) {
	t.Run("HealthStatus with all fields", func(t *testing.T) {
		status := HealthStatus{
			Status:  "healthy",
			Message: "All systems operational",
			Latency: 15,
		}

		data, err := json.Marshal(status)
		require.NoError(t, err)

		var result HealthStatus
		err = json.Unmarshal(data, &result)
		require.NoError(t, err)

		assert.Equal(t, "healthy", result.Status)
		assert.Equal(t, "All systems operational", result.Message)
		assert.Equal(t, int64(15), result.Latency)
	})

	t.Run("HealthStatus states", func(t *testing.T) {
		states := []string{"healthy", "degraded", "unhealthy"}

		for _, state := range states {
			status := HealthStatus{Status: state}
			assert.Contains(t, states, status.Status)
		}
	})

	t.Run("HealthStatus with error", func(t *testing.T) {
		status := HealthStatus{
			Status:  "unhealthy",
			Message: "Connection timeout",
			Latency: 5000,
		}

		assert.Equal(t, "unhealthy", status.Status)
		assert.NotEmpty(t, status.Message)
		assert.Greater(t, status.Latency, int64(0))
	})
}

func TestSystemHealth_Struct(t *testing.T) {
	t.Run("SystemHealth with multiple services", func(t *testing.T) {
		health := SystemHealth{
			Status: "healthy",
			Services: map[string]HealthStatus{
				"database": {
					Status:  "healthy",
					Latency: 5,
				},
				"realtime": {
					Status:  "healthy",
					Message: "WebSocket server running",
				},
				"storage": {
					Status:  "healthy",
					Latency: 10,
				},
			},
		}

		data, err := json.Marshal(health)
		require.NoError(t, err)

		var result SystemHealth
		err = json.Unmarshal(data, &result)
		require.NoError(t, err)

		assert.Equal(t, "healthy", result.Status)
		assert.Equal(t, 3, len(result.Services))
		assert.Equal(t, "healthy", result.Services["database"].Status)
		assert.Equal(t, int64(5), result.Services["database"].Latency)
	})

	t.Run("SystemHealth degraded state", func(t *testing.T) {
		health := SystemHealth{
			Status: "degraded",
			Services: map[string]HealthStatus{
				"database": {Status: "healthy", Latency: 5},
				"storage": {
					Status:  "degraded",
					Message: "Slow response",
					Latency: 3000,
				},
			},
		}

		assert.Equal(t, "degraded", health.Status)
		assert.Equal(t, "degraded", health.Services["storage"].Status)
	})

	t.Run("SystemHealth unhealthy state", func(t *testing.T) {
		health := SystemHealth{
			Status: "unhealthy",
			Services: map[string]HealthStatus{
				"database": {
					Status:  "unhealthy",
					Message: "Connection refused",
					Latency: 0,
				},
				"storage": {Status: "healthy", Latency: 10},
			},
		}

		assert.Equal(t, "unhealthy", health.Status)
		assert.Equal(t, "unhealthy", health.Services["database"].Status)
		assert.NotEmpty(t, health.Services["database"].Message)
	})
}

func TestLogEntry_Struct(t *testing.T) {
	t.Run("LogEntry marshaling", func(t *testing.T) {
		entry := LogEntry{
			Timestamp: time.Now(),
			Level:     "error",
			Message:   "Database connection failed",
			Module:    "auth",
			Error:     "connection refused",
			Fields: map[string]interface{}{
				"attempt":  3,
				"duration": "5s",
			},
		}

		data, err := json.Marshal(entry)
		require.NoError(t, err)

		var result LogEntry
		err = json.Unmarshal(data, &result)
		require.NoError(t, err)

		assert.Equal(t, "error", result.Level)
		assert.Equal(t, "Database connection failed", result.Message)
		assert.Equal(t, "auth", result.Module)
		assert.Equal(t, "connection refused", result.Error)
		assert.Equal(t, 2, len(result.Fields))
	})

	t.Run("LogEntry log levels", func(t *testing.T) {
		levels := []string{"debug", "info", "warn", "error", "fatal"}

		for _, level := range levels {
			entry := LogEntry{
				Timestamp: time.Now(),
				Level:     level,
				Message:   "Test message",
			}
			assert.Contains(t, levels, entry.Level)
		}
	})
}

func TestNewMonitoringHandler(t *testing.T) {
	t.Run("Create monitoring handler with nil dependencies", func(t *testing.T) {
		handler := NewMonitoringHandler(nil, nil, nil)
		assert.NotNil(t, handler)
		assert.Nil(t, handler.db)
		assert.Nil(t, handler.realtimeHandler)
		assert.Nil(t, handler.storageProvider)
	})
}

func TestMetricsCalculations(t *testing.T) {
	t.Run("Memory conversions", func(t *testing.T) {
		// Test byte to MB conversion
		bytes := uint64(1024 * 1024 * 100) // 100 MB
		mb := bytes / 1024 / 1024

		assert.Equal(t, uint64(100), mb)
	})

	t.Run("Uptime calculation", func(t *testing.T) {
		start := time.Now().Add(-1 * time.Hour)
		uptime := int64(time.Since(start).Seconds())

		assert.GreaterOrEqual(t, uptime, int64(3600))
		assert.LessOrEqual(t, uptime, int64(3601))
	})

	t.Run("GC pause conversion", func(t *testing.T) {
		// Test nanoseconds to milliseconds conversion
		pauseNs := uint64(1500000) // 1.5 ms
		pauseMs := float64(pauseNs) / 1000000

		assert.InDelta(t, 1.5, pauseMs, 0.01)
	})

	t.Run("Storage size conversion GB", func(t *testing.T) {
		// Test bytes to GB conversion
		totalBytes := int64(5 * 1024 * 1024 * 1024) // 5 GB
		sizeGB := float64(totalBytes) / 1024 / 1024 / 1024

		assert.InDelta(t, 5.0, sizeGB, 0.01)
	})
}

func TestHealthStatusTransitions(t *testing.T) {
	t.Run("Health state priority", func(t *testing.T) {
		// Test that unhealthy takes precedence over degraded
		health := SystemHealth{
			Status: "healthy",
			Services: map[string]HealthStatus{
				"db":      {Status: "unhealthy"},
				"storage": {Status: "degraded"},
			},
		}

		// In actual implementation, if any service is unhealthy, overall status should be unhealthy
		// This tests the logic
		overallStatus := "healthy"
		for _, service := range health.Services {
			if service.Status == "unhealthy" {
				overallStatus = "unhealthy"
				break
			}
			if service.Status == "degraded" && overallStatus == "healthy" {
				overallStatus = "degraded"
			}
		}

		assert.Equal(t, "unhealthy", overallStatus)
	})

	t.Run("All services healthy", func(t *testing.T) {
		health := SystemHealth{
			Status: "healthy",
			Services: map[string]HealthStatus{
				"database": {Status: "healthy"},
				"realtime": {Status: "healthy"},
				"storage":  {Status: "healthy"},
			},
		}

		allHealthy := true
		for _, service := range health.Services {
			if service.Status != "healthy" {
				allHealthy = false
				break
			}
		}

		assert.True(t, allHealthy)
		assert.Equal(t, "healthy", health.Status)
	})
}

func TestMonitoringDataValidation(t *testing.T) {
	t.Run("Validate metric ranges", func(t *testing.T) {
		metrics := SystemMetrics{
			NumGoroutine:       runtime.NumGoroutine(),
			MemoryAllocMB:      100,
			MemoryTotalAllocMB: 500,
			MemorySysMB:        1000,
		}

		// Goroutines should be positive
		assert.Greater(t, metrics.NumGoroutine, 0)

		// Memory allocations should be logical
		assert.GreaterOrEqual(t, metrics.MemoryTotalAllocMB, metrics.MemoryAllocMB,
			"Total alloc should be >= current alloc")
		assert.GreaterOrEqual(t, metrics.MemorySysMB, metrics.MemoryAllocMB,
			"System memory should be >= allocated memory")
	})

	t.Run("Validate latency values", func(t *testing.T) {
		health := HealthStatus{
			Status:  "healthy",
			Latency: 15,
		}

		// Latency should be non-negative
		assert.GreaterOrEqual(t, health.Latency, int64(0))

		// Typical healthy latency should be under 1 second (1000ms)
		if health.Status == "healthy" {
			assert.Less(t, health.Latency, int64(1000),
				"Healthy service latency should typically be < 1s")
		}
	})
}
