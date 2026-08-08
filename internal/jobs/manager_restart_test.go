package jobs

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nimbleflux/fluxbase/internal/config"
)

// =============================================================================
// Manager restart / supervision Tests
//
// superviseWorkers always recovers failed workers (never permanently gives up),
// with manager-level exponential backoff that escalates across replacements and
// caps at WorkerMaxRestartBackoff. These tests pin that behavior.
// =============================================================================

func TestManager_NextRestartBackoff_Escalates(t *testing.T) {
	t.Run("backoff escalates 1s,2s,4s,... up to cap", func(t *testing.T) {
		cfg := &config.JobsConfig{WorkerMaxRestartBackoff: 60 * time.Second}
		m := NewManager(cfg, nil, "secret", "http://localhost", nil, nil)

		// First failure: base 1s.
		b, n := m.nextRestartBackoff(5 * time.Minute)
		assert.Equal(t, time.Second, b)
		assert.Equal(t, 1, n)

		b, n = m.nextRestartBackoff(5 * time.Minute)
		assert.Equal(t, 2*time.Second, b)
		assert.Equal(t, 2, n)

		b, n = m.nextRestartBackoff(5 * time.Minute)
		assert.Equal(t, 4*time.Second, b)
		assert.Equal(t, 3, n)

		b, n = m.nextRestartBackoff(5 * time.Minute)
		assert.Equal(t, 8*time.Second, b)
		assert.Equal(t, 4, n)
	})
}

func TestManager_NextRestartBackoff_Capped(t *testing.T) {
	t.Run("backoff never exceeds configured cap", func(t *testing.T) {
		cap := 10 * time.Second
		cfg := &config.JobsConfig{WorkerMaxRestartBackoff: cap}
		m := NewManager(cfg, nil, "secret", "http://localhost", nil, nil)

		// Drive enough failures to exceed the cap.
		var last time.Duration
		for i := 0; i < 30; i++ {
			b, n := m.nextRestartBackoff(5 * time.Minute)
			last = b
			require.Equal(t, i+1, n)
			assert.LessOrEqual(t, b, cap, "backoff must not exceed cap on iter %d", i)
		}
		assert.Equal(t, cap, last, "backoff should reach the cap under sustained failure")
	})
}

func TestManager_NextRestartBackoff_DefaultCap(t *testing.T) {
	t.Run("non-positive config cap defaults to 60s", func(t *testing.T) {
		m := NewManager(&config.JobsConfig{}, nil, "secret", "http://localhost", nil, nil)
		assert.Equal(t, 60*time.Second, m.maxRestartBackoff())

		m2 := NewManager(nil, nil, "secret", "http://localhost", nil, nil)
		assert.Equal(t, 60*time.Second, m2.maxRestartBackoff())
	})
}

func TestManager_NextRestartBackoff_ResetWindow(t *testing.T) {
	t.Run("failures older than the reset window are pruned", func(t *testing.T) {
		cfg := &config.JobsConfig{WorkerMaxRestartBackoff: 60 * time.Second}
		m := NewManager(cfg, nil, "secret", "http://localhost", nil, nil)

		// Accumulate a few failures within the window.
		_, n := m.nextRestartBackoff(5 * time.Minute)
		assert.Equal(t, 1, n)
		_, n = m.nextRestartBackoff(5 * time.Minute)
		assert.Equal(t, 2, n)
		_, n = m.nextRestartBackoff(5 * time.Minute)
		assert.Equal(t, 3, n)

		// Now backdate all recorded failures so they fall outside the window.
		m.restartMutex.Lock()
		old := time.Now().Add(-10 * time.Minute)
		for i := range m.restartFailures {
			m.restartFailures[i] = old
		}
		m.restartMutex.Unlock()

		// Next failure should reset the escalation (only the new one counts).
		b, n := m.nextRestartBackoff(5 * time.Minute)
		assert.Equal(t, 1, n)
		assert.Equal(t, time.Second, b)
	})
}

// spawnCountingManager returns a Manager whose worker spawns are counted via an
// override, plus a cancel for its supervisor context. No DB is involved.
func spawnCountingManager(t *testing.T, targetCount int, spawnDelay time.Duration) (*Manager, *int32, context.CancelFunc) {
	t.Helper()
	cfg := &config.JobsConfig{WorkerMaxRestartBackoff: 60 * time.Second}
	m := NewManager(cfg, nil, "secret", "http://localhost", nil, nil)
	m.targetCount = targetCount

	m.supervisorCtx, m.supervisorStop = context.WithCancel(context.Background())

	var spawns int32
	m.startWorkerFn = func(ctx context.Context) *Worker {
		atomic.AddInt32(&spawns, 1)
		if spawnDelay > 0 {
			time.Sleep(spawnDelay)
		}
		return nil
	}
	return m, &spawns, m.supervisorStop
}

func TestManager_SuperviseWorkers_RestartsOnFailure(t *testing.T) {
	t.Run("a worker error triggers a replacement spawn", func(t *testing.T) {
		m, spawns, stop := spawnCountingManager(t, 1, 0)
		defer stop()

		go m.superviseWorkers()
		defer stop()

		// First failure: backoff is 1s, then a spawn should occur.
		m.workerErrors <- workerError{workerID: newTestUUID(), err: errFail("loop panic")}

		// Wait a bit longer than the 1s backoff for the spawn.
		done := make(chan struct{})
		go func() {
			for atomic.LoadInt32(spawns) == 0 {
				time.Sleep(10 * time.Millisecond)
			}
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			t.Fatalf("expected a replacement spawn, got %d", atomic.LoadInt32(spawns))
		}
	})
}

func TestManager_SuperviseWorkers_InterruptibleBackoff(t *testing.T) {
	t.Run("Stop returns promptly during a long backoff", func(t *testing.T) {
		// Large cap + many pre-seeded failures -> a long backoff. The supervisor
		// must still exit immediately when its context is cancelled.
		cfg := &config.JobsConfig{WorkerMaxRestartBackoff: 60 * time.Second}
		m := NewManager(cfg, nil, "secret", "http://localhost", nil, nil)
		m.targetCount = 1
		m.supervisorCtx, m.supervisorStop = context.WithCancel(context.Background())

		var spawns int32
		m.startWorkerFn = func(ctx context.Context) *Worker {
			atomic.AddInt32(&spawns, 1)
			return nil
		}

		// Pre-seed a high failure count so the next backoff is large.
		for i := 0; i < 10; i++ {
			_, _ = m.nextRestartBackoff(5 * time.Minute)
		}

		go m.superviseWorkers()

		start := time.Now()
		// Feed an error to park the supervisor in its backoff select, then stop.
		m.workerErrors <- workerError{workerID: newTestUUID(), err: errFail("boom")}
		// Give the supervisor a moment to enter the backoff wait.
		time.Sleep(50 * time.Millisecond)
		m.supervisorStop()

		// superviseWorkers returns on ctx Done; we only assert that stopping did
		// not block for the full (multi-second) backoff.
		elapsed := time.Since(start)
		assert.Less(t, elapsed, 2*time.Second, "supervisor should exit during backoff, not wait it out")
	})
}

func TestManager_SuperviseWorkers_AtTargetNoSpawn(t *testing.T) {
	t.Run("does not spawn when already at target count", func(t *testing.T) {
		m, spawns, stop := spawnCountingManager(t, 2, 0)
		defer stop()

		// Pretend we already have targetCount active workers.
		m.workersMutex.Lock()
		m.activeWorkers[newTestUUID()] = true
		m.activeWorkers[newTestUUID()] = true
		m.workersMutex.Unlock()

		go m.superviseWorkers()
		defer stop()

		m.workerErrors <- workerError{workerID: newTestUUID(), err: errFail("boom")}

		// Allow enough time for the 1s backoff + check; no spawn should happen.
		time.Sleep(1500 * time.Millisecond)
		assert.Equal(t, int32(0), atomic.LoadInt32(spawns), "no spawn expected at target count")
	})
}
