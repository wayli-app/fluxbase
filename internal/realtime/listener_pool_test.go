package realtime

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDefaultListenerPoolConfig(t *testing.T) {
	config := DefaultListenerPoolConfig()

	assert.Equal(t, 2, config.PoolSize)
	assert.Equal(t, 4, config.WorkerCount)
	assert.Equal(t, 1000, config.QueueSize)
	assert.Equal(t, time.Second, config.RetryInterval)
	assert.Equal(t, 5, config.MaxRetries)
}

func TestListenerPoolConfig_DefaultsApplied(t *testing.T) {
	// Test that zero values get sensible defaults
	ctx := context.Background()
	manager := NewManager(ctx)
	handler := NewRealtimeHandler(manager, nil, nil)

	lp := NewListenerPool(nil, handler, nil, nil, ListenerPoolConfig{
		// All zero values
	})

	assert.Equal(t, 2, lp.config.PoolSize)
	assert.Equal(t, 4, lp.config.WorkerCount)
	assert.Equal(t, 1000, lp.config.QueueSize)
	assert.Equal(t, time.Second, lp.config.RetryInterval)
	assert.Equal(t, 5, lp.config.MaxRetries)
}

func TestListenerPoolConfig_CustomValues(t *testing.T) {
	ctx := context.Background()
	manager := NewManager(ctx)
	handler := NewRealtimeHandler(manager, nil, nil)

	lp := NewListenerPool(nil, handler, nil, nil, ListenerPoolConfig{
		PoolSize:      4,
		WorkerCount:   8,
		QueueSize:     2000,
		RetryInterval: 2 * time.Second,
		MaxRetries:    10,
	})

	assert.Equal(t, 4, lp.config.PoolSize)
	assert.Equal(t, 8, lp.config.WorkerCount)
	assert.Equal(t, 2000, lp.config.QueueSize)
	assert.Equal(t, 2*time.Second, lp.config.RetryInterval)
	assert.Equal(t, 10, lp.config.MaxRetries)
}

func TestListenerPoolConfig_NegativeValuesFallback(t *testing.T) {
	ctx := context.Background()
	manager := NewManager(ctx)
	handler := NewRealtimeHandler(manager, nil, nil)

	lp := NewListenerPool(nil, handler, nil, nil, ListenerPoolConfig{
		PoolSize:      -1,
		WorkerCount:   -5,
		QueueSize:     -100,
		RetryInterval: -time.Second,
		MaxRetries:    -3,
	})

	// Negative values should fall back to defaults
	assert.Equal(t, 2, lp.config.PoolSize)
	assert.Equal(t, 4, lp.config.WorkerCount)
	assert.Equal(t, 1000, lp.config.QueueSize)
	assert.Equal(t, time.Second, lp.config.RetryInterval)
	assert.Equal(t, 5, lp.config.MaxRetries)
}

func TestListenerPool_NotificationChannelCapacity(t *testing.T) {
	ctx := context.Background()
	manager := NewManager(ctx)
	handler := NewRealtimeHandler(manager, nil, nil)

	lp := NewListenerPool(nil, handler, nil, nil, ListenerPoolConfig{
		WorkerCount: 4,
		QueueSize:   500,
	})

	// Channel capacity should be QueueSize * WorkerCount
	assert.Equal(t, 500*4, cap(lp.notificationCh))
}

func TestListenerPool_StopBeforeStart(t *testing.T) {
	ctx := context.Background()
	manager := NewManager(ctx)
	handler := NewRealtimeHandler(manager, nil, nil)

	lp := NewListenerPool(nil, handler, nil, nil, DefaultListenerPoolConfig())

	// Should not panic
	lp.Stop()
}

func TestListenerPool_MetricsInitialValues(t *testing.T) {
	ctx := context.Background()
	manager := NewManager(ctx)
	handler := NewRealtimeHandler(manager, nil, nil)

	lp := NewListenerPool(nil, handler, nil, nil, DefaultListenerPoolConfig())

	metrics := lp.GetMetrics()

	assert.Equal(t, int32(0), metrics.ActiveConnections)
	assert.Equal(t, uint64(0), metrics.NotificationsReceived)
	assert.Equal(t, uint64(0), metrics.NotificationsProcessed)
	assert.Equal(t, uint64(0), metrics.ConnectionFailures)
	assert.Equal(t, uint64(0), metrics.Reconnections)
	assert.Equal(t, 0, metrics.QueueLength)
	assert.Greater(t, metrics.QueueCapacity, 0)
}

func TestListenerPool_MetricsQueueCapacity(t *testing.T) {
	ctx := context.Background()
	manager := NewManager(ctx)
	handler := NewRealtimeHandler(manager, nil, nil)

	config := ListenerPoolConfig{
		WorkerCount: 2,
		QueueSize:   100,
	}

	lp := NewListenerPool(nil, handler, nil, nil, config)

	metrics := lp.GetMetrics()

	// Capacity should match WorkerCount * QueueSize
	assert.Equal(t, 200, metrics.QueueCapacity)
}

func TestListenerPool_AtomicCountersSafe(t *testing.T) {
	// Test that atomic counters can be safely incremented
	var counter uint64

	// Simulate concurrent access
	done := make(chan bool, 100)
	for i := 0; i < 100; i++ {
		go func() {
			atomic.AddUint64(&counter, 1)
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 100; i++ {
		<-done
	}

	assert.Equal(t, uint64(100), atomic.LoadUint64(&counter))
}

func TestListenerPoolImplementsInterface(t *testing.T) {
	ctx := context.Background()
	manager := NewManager(ctx)
	handler := NewRealtimeHandler(manager, nil, nil)

	// ListenerPool should implement RealtimeListener
	var _ RealtimeListener = NewListenerPool(nil, handler, nil, nil, DefaultListenerPoolConfig())
}

func TestListenerImplementsInterface(t *testing.T) {
	ctx := context.Background()
	manager := NewManager(ctx)
	handler := NewRealtimeHandler(manager, nil, nil)

	// Listener should also implement RealtimeListener
	var _ RealtimeListener = NewListener(nil, handler, nil, nil)
}

func TestListenerPoolMetrics_Struct(t *testing.T) {
	metrics := ListenerPoolMetrics{
		ActiveConnections:      2,
		NotificationsReceived:  1000,
		NotificationsProcessed: 950,
		ConnectionFailures:     5,
		Reconnections:          3,
		QueueLength:            50,
		QueueCapacity:          4000,
	}

	assert.Equal(t, int32(2), metrics.ActiveConnections)
	assert.Equal(t, uint64(1000), metrics.NotificationsReceived)
	assert.Equal(t, uint64(950), metrics.NotificationsProcessed)
	assert.Equal(t, uint64(5), metrics.ConnectionFailures)
	assert.Equal(t, uint64(3), metrics.Reconnections)
	assert.Equal(t, 50, metrics.QueueLength)
	assert.Equal(t, 4000, metrics.QueueCapacity)
}

func TestListenerPool_EnrichJobWithETA_NoProgress(t *testing.T) {
	ctx := context.Background()
	manager := NewManager(ctx)
	handler := NewRealtimeHandler(manager, nil, nil)

	lp := NewListenerPool(nil, handler, nil, nil, DefaultListenerPoolConfig())

	// Event without progress should not panic
	event := &ChangeEvent{
		Schema: "jobs",
		Table:  "queue",
		Type:   "UPDATE",
		Record: map[string]interface{}{
			"id":     "123",
			"status": "running",
		},
	}

	// Should not panic
	lp.enrichJobWithETA(event)

	// No progress fields added
	_, hasPercent := event.Record["progress_percent"]
	assert.False(t, hasPercent)
}

func TestListenerPool_EnrichJobWithETA_WithProgress(t *testing.T) {
	ctx := context.Background()
	manager := NewManager(ctx)
	handler := NewRealtimeHandler(manager, nil, nil)

	lp := NewListenerPool(nil, handler, nil, nil, DefaultListenerPoolConfig())

	event := &ChangeEvent{
		Schema: "jobs",
		Table:  "queue",
		Type:   "UPDATE",
		Record: map[string]interface{}{
			"id":     "123",
			"status": "running",
			"progress": map[string]interface{}{
				"percent": float64(50),
				"message": "Processing items...",
			},
		},
	}

	lp.enrichJobWithETA(event)

	assert.Equal(t, 50, event.Record["progress_percent"])
	assert.Equal(t, "Processing items...", event.Record["progress_message"])
}

func TestListenerPool_EnrichJobWithETA_WithExistingETA(t *testing.T) {
	ctx := context.Background()
	manager := NewManager(ctx)
	handler := NewRealtimeHandler(manager, nil, nil)

	lp := NewListenerPool(nil, handler, nil, nil, DefaultListenerPoolConfig())

	event := &ChangeEvent{
		Schema: "jobs",
		Table:  "queue",
		Type:   "UPDATE",
		Record: map[string]interface{}{
			"id":     "123",
			"status": "running",
			"progress": map[string]interface{}{
				"percent":                float64(75),
				"estimated_seconds_left": float64(120),
			},
		},
	}

	lp.enrichJobWithETA(event)

	assert.Equal(t, 75, event.Record["progress_percent"])
	assert.Equal(t, 120, event.Record["estimated_seconds_left"])
}

func TestChangeEvent_Struct(t *testing.T) {
	event := ChangeEvent{
		Type:   "INSERT",
		Table:  "users",
		Schema: "public",
		Record: map[string]interface{}{
			"id":    "123",
			"email": "test@example.com",
		},
	}

	assert.Equal(t, "INSERT", event.Type)
	assert.Equal(t, "users", event.Table)
	assert.Equal(t, "public", event.Schema)
	assert.Equal(t, "123", event.Record["id"])
}

func TestChangeEvent_WithOldRecord(t *testing.T) {
	event := ChangeEvent{
		Type:   "UPDATE",
		Table:  "users",
		Schema: "public",
		Record: map[string]interface{}{
			"email": "new@example.com",
		},
		OldRecord: map[string]interface{}{
			"email": "old@example.com",
		},
	}

	assert.Equal(t, "UPDATE", event.Type)
	assert.Equal(t, "new@example.com", event.Record["email"])
	assert.Equal(t, "old@example.com", event.OldRecord["email"])
}

// Benchmarks

func BenchmarkListenerPool_GetMetrics(b *testing.B) {
	ctx := context.Background()
	manager := NewManager(ctx)
	handler := NewRealtimeHandler(manager, nil, nil)

	lp := NewListenerPool(nil, handler, nil, nil, DefaultListenerPoolConfig())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = lp.GetMetrics()
	}
}

func BenchmarkListenerPool_EnrichJobWithETA(b *testing.B) {
	ctx := context.Background()
	manager := NewManager(ctx)
	handler := NewRealtimeHandler(manager, nil, nil)

	lp := NewListenerPool(nil, handler, nil, nil, DefaultListenerPoolConfig())

	event := &ChangeEvent{
		Schema: "jobs",
		Table:  "queue",
		Type:   "UPDATE",
		Record: map[string]interface{}{
			"id":     "123",
			"status": "running",
			"progress": map[string]interface{}{
				"percent": float64(50),
				"message": "Processing...",
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lp.enrichJobWithETA(event)
	}
}
