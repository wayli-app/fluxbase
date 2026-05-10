package scaling

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Lock ID Constants Tests
// =============================================================================

func TestLeaderLockID_Constants(t *testing.T) {
	t.Run("JobsSchedulerLockID has expected value", func(t *testing.T) {
		// 0x466C7578_00000001 = "Flux" + 1
		expected := int64(0x466C7578_00000001)
		assert.Equal(t, expected, JobsSchedulerLockID)
	})

	t.Run("FunctionsSchedulerLockID has expected value", func(t *testing.T) {
		// 0x466C7578_00000002 = "Flux" + 2
		expected := int64(0x466C7578_00000002)
		assert.Equal(t, expected, FunctionsSchedulerLockID)
	})

	t.Run("RPCSchedulerLockID has expected value", func(t *testing.T) {
		// 0x466C7578_00000003 = "Flux" + 3
		expected := int64(0x466C7578_00000003)
		assert.Equal(t, expected, RPCSchedulerLockID)
	})

	t.Run("all lock IDs are unique", func(t *testing.T) {
		lockIDs := []int64{
			JobsSchedulerLockID,
			FunctionsSchedulerLockID,
			RPCSchedulerLockID,
		}

		seen := make(map[int64]bool)
		for _, id := range lockIDs {
			assert.False(t, seen[id], "duplicate lock ID: %d", id)
			seen[id] = true
		}
	})

	t.Run("lock IDs share common prefix", func(t *testing.T) {
		// All lock IDs should share the "Flux" prefix (0x466C7578)
		prefix := int64(0x466C7578_00000000)
		mask := int64(-1) << 32 // Upper 32 bits mask (0xFFFFFFFF_00000000)

		assert.Equal(t, prefix, JobsSchedulerLockID&mask)
		assert.Equal(t, prefix, FunctionsSchedulerLockID&mask)
		assert.Equal(t, prefix, RPCSchedulerLockID&mask)
	})

	t.Run("lock IDs are positive", func(t *testing.T) {
		assert.Greater(t, JobsSchedulerLockID, int64(0))
		assert.Greater(t, FunctionsSchedulerLockID, int64(0))
		assert.Greater(t, RPCSchedulerLockID, int64(0))
	})
}

// =============================================================================
// NewLeaderElector Tests
// =============================================================================

func TestNewLeaderElector(t *testing.T) {
	t.Run("creates with nil pool", func(t *testing.T) {
		le := NewLeaderElector(nil, JobsSchedulerLockID, "jobs-scheduler")

		require.NotNil(t, le)
		assert.Nil(t, le.pool)
		assert.Equal(t, JobsSchedulerLockID, le.lockID)
		assert.Equal(t, "jobs-scheduler", le.lockName)
		assert.False(t, le.isLeader)
		assert.Equal(t, 5*time.Second, le.checkInterval)
		assert.NotNil(t, le.ctx)
		assert.NotNil(t, le.cancel)
	})

	t.Run("initializes with correct lock ID", func(t *testing.T) {
		le := NewLeaderElector(nil, FunctionsSchedulerLockID, "functions-scheduler")

		assert.Equal(t, FunctionsSchedulerLockID, le.lockID)
		assert.Equal(t, "functions-scheduler", le.lockName)
	})

	t.Run("initializes with custom lock ID", func(t *testing.T) {
		customLockID := int64(12345)
		le := NewLeaderElector(nil, customLockID, "custom-scheduler")

		assert.Equal(t, customLockID, le.lockID)
	})

	t.Run("starts not as leader", func(t *testing.T) {
		le := NewLeaderElector(nil, JobsSchedulerLockID, "test")

		assert.False(t, le.IsLeader())
	})

	t.Run("context is cancelable", func(t *testing.T) {
		le := NewLeaderElector(nil, JobsSchedulerLockID, "test")

		// Context should not be done initially
		select {
		case <-le.ctx.Done():
			t.Error("context should not be done initially")
		default:
			// Good
		}

		// Cancel should work
		le.cancel()

		select {
		case <-le.ctx.Done():
			// Good
		case <-time.After(100 * time.Millisecond):
			t.Error("context should be done after cancel")
		}
	})

	t.Run("empty lock name is allowed", func(t *testing.T) {
		le := NewLeaderElector(nil, JobsSchedulerLockID, "")

		assert.Empty(t, le.lockName)
	})
}

// =============================================================================
// LeaderElector Struct Tests
// =============================================================================

func TestLeaderElector_Struct(t *testing.T) {
	t.Run("zero value has safe defaults", func(t *testing.T) {
		var le LeaderElector

		assert.Nil(t, le.pool)
		assert.Zero(t, le.lockID)
		assert.Empty(t, le.lockName)
		assert.False(t, le.isLeader)
		assert.Zero(t, le.checkInterval)
		assert.Nil(t, le.ctx)
		assert.Nil(t, le.cancel)
	})

	t.Run("isLeader is protected by mutex", func(t *testing.T) {
		le := NewLeaderElector(nil, JobsSchedulerLockID, "test")

		// Verify mutex exists
		assert.NotNil(t, &le.isLeaderMu)
	})
}

// =============================================================================
// IsLeader Tests
// =============================================================================

func TestIsLeader(t *testing.T) {
	t.Run("returns false initially", func(t *testing.T) {
		le := NewLeaderElector(nil, JobsSchedulerLockID, "test")

		assert.False(t, le.IsLeader())
	})

	t.Run("returns true when set to leader", func(t *testing.T) {
		le := NewLeaderElector(nil, JobsSchedulerLockID, "test")

		// Directly set isLeader (simulating lock acquisition)
		le.isLeaderMu.Lock()
		le.isLeader = true
		le.isLeaderMu.Unlock()

		assert.True(t, le.IsLeader())
	})

	t.Run("returns false when leadership is lost", func(t *testing.T) {
		le := NewLeaderElector(nil, JobsSchedulerLockID, "test")

		// Set to leader
		le.isLeaderMu.Lock()
		le.isLeader = true
		le.isLeaderMu.Unlock()

		assert.True(t, le.IsLeader())

		// Lose leadership
		le.isLeaderMu.Lock()
		le.isLeader = false
		le.isLeaderMu.Unlock()

		assert.False(t, le.IsLeader())
	})

	t.Run("is thread-safe", func(t *testing.T) {
		le := NewLeaderElector(nil, JobsSchedulerLockID, "test")

		var wg sync.WaitGroup
		const iterations = 1000

		// Start multiple goroutines reading and writing
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(writerID int) {
				defer wg.Done()
				for j := 0; j < iterations; j++ {
					le.isLeaderMu.Lock()
					le.isLeader = (writerID+j)%2 == 0
					le.isLeaderMu.Unlock()
				}
			}(i)
		}

		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < iterations; j++ {
					_ = le.IsLeader()
				}
			}()
		}

		wg.Wait()
		// Test passes if no race condition panics occur
	})
}

// =============================================================================
// Check Interval Tests
// =============================================================================

func TestLeaderElector_CheckInterval(t *testing.T) {
	t.Run("default interval is 5 seconds", func(t *testing.T) {
		le := NewLeaderElector(nil, JobsSchedulerLockID, "test")

		assert.Equal(t, 5*time.Second, le.checkInterval)
	})

	t.Run("interval can be modified", func(t *testing.T) {
		le := NewLeaderElector(nil, JobsSchedulerLockID, "test")
		le.checkInterval = 10 * time.Second

		assert.Equal(t, 10*time.Second, le.checkInterval)
	})
}

// =============================================================================
// Multiple Elector Tests
// =============================================================================

func TestMultipleElectors(t *testing.T) {
	t.Run("multiple electors for same lock ID", func(t *testing.T) {
		le1 := NewLeaderElector(nil, JobsSchedulerLockID, "jobs-1")
		le2 := NewLeaderElector(nil, JobsSchedulerLockID, "jobs-2")

		// Both should have same lock ID
		assert.Equal(t, le1.lockID, le2.lockID)

		// But different lock names
		assert.NotEqual(t, le1.lockName, le2.lockName)

		// And independent contexts
		le1.cancel()
		select {
		case <-le1.ctx.Done():
			// Good - le1 context is done
		default:
			t.Error("le1 context should be done")
		}

		select {
		case <-le2.ctx.Done():
			t.Error("le2 context should not be affected by le1 cancel")
		default:
			// Good
		}
	})

	t.Run("multiple electors for different lock IDs", func(t *testing.T) {
		le1 := NewLeaderElector(nil, JobsSchedulerLockID, "jobs")
		le2 := NewLeaderElector(nil, FunctionsSchedulerLockID, "functions")
		le3 := NewLeaderElector(nil, RPCSchedulerLockID, "rpc")

		// All should have different lock IDs
		assert.NotEqual(t, le1.lockID, le2.lockID)
		assert.NotEqual(t, le2.lockID, le3.lockID)
		assert.NotEqual(t, le1.lockID, le3.lockID)

		// All should start as non-leaders
		assert.False(t, le1.IsLeader())
		assert.False(t, le2.IsLeader())
		assert.False(t, le3.IsLeader())
	})
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkNewLeaderElector(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		le := NewLeaderElector(nil, JobsSchedulerLockID, "benchmark")
		le.cancel() // Clean up context
	}
}

func BenchmarkIsLeader(b *testing.B) {
	le := NewLeaderElector(nil, JobsSchedulerLockID, "benchmark")
	defer le.cancel()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = le.IsLeader()
	}
}

func BenchmarkIsLeader_Concurrent(b *testing.B) {
	le := NewLeaderElector(nil, JobsSchedulerLockID, "benchmark")
	defer le.cancel()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = le.IsLeader()
		}
	})
}

// =============================================================================
// Start/Stop Idempotency and Safety Tests
// =============================================================================

func TestLeaderElector_StartIdempotent(t *testing.T) {
	t.Run("Start is idempotent", func(t *testing.T) {
		elector := NewLeaderElector(nil, JobsSchedulerLockID, "test")

		assert.False(t, elector.started, "should not be started initially")

		elector.Start(func() {}, func() {})
		assert.True(t, elector.started, "started should be true after first Start")

		elector.Start(func() {}, func() {})
		assert.True(t, elector.started, "started should still be true after second Start")

		elector.cancel()
	})
}

func TestLeaderElector_StopSafety(t *testing.T) {
	t.Run("Stop is safe when not started", func(t *testing.T) {
		elector := NewLeaderElector(nil, JobsSchedulerLockID, "test")

		assert.NotPanics(t, func() {
			elector.Stop()
		})
	})

	t.Run("Stop clears leadership state", func(t *testing.T) {
		elector := NewLeaderElector(nil, JobsSchedulerLockID, "test")

		elector.isLeaderMu.Lock()
		elector.isLeader = true
		elector.isLeaderMu.Unlock()
		assert.True(t, elector.IsLeader())

		elector.Stop()

		assert.False(t, elector.IsLeader(), "Stop should clear leadership")
	})

	t.Run("Stop is safe to call multiple times", func(t *testing.T) {
		elector := NewLeaderElector(nil, JobsSchedulerLockID, "test")

		assert.NotPanics(t, func() {
			elector.Stop()
			elector.Stop()
			elector.Stop()
		})
	})
}

func TestLeaderElector_ReleaseLockSafety(t *testing.T) {
	t.Run("releaseLock is safe with nil dedicatedConn", func(t *testing.T) {
		elector := NewLeaderElector(nil, JobsSchedulerLockID, "test")

		assert.NotPanics(t, func() {
			elector.releaseLock()
		})
	})

	t.Run("releaseLock clears isLeader", func(t *testing.T) {
		elector := NewLeaderElector(nil, JobsSchedulerLockID, "test")

		elector.isLeaderMu.Lock()
		elector.isLeader = true
		elector.isLeaderMu.Unlock()

		elector.releaseLock()

		assert.False(t, elector.IsLeader())
	})

	t.Run("releaseLock is safe to call multiple times", func(t *testing.T) {
		elector := NewLeaderElector(nil, JobsSchedulerLockID, "test")

		assert.NotPanics(t, func() {
			elector.releaseLock()
			elector.releaseLock()
			elector.releaseLock()
		})
	})
}
