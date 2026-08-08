package jobs

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nimbleflux/fluxbase/internal/config"
)

// =============================================================================
// Worker fail() Tests
//
// fail() lets a background loop report an unrecoverable error so Worker.Start
// surfaces it to the manager's supervisor (which starts a replacement). Without
// it, a panicked loop would leave the worker silently idle forever. These tests
// pin the contract: fail() delivers exactly once, and signalShutdown() is a
// safe double-close guard.
// =============================================================================

func TestWorker_Fail_Idempotent(t *testing.T) {
	t.Run("delivers exactly one fatal error", func(t *testing.T) {
		cfg := &config.JobsConfig{MaxConcurrentPerWorker: 5}
		worker := NewWorker(cfg, nil, "secret", "http://localhost", nil, nil, nil)

		// First call delivers.
		worker.fail("boom-1")
		select {
		case err := <-worker.fatalErr:
			require.Error(t, err)
			assert.Contains(t, err.Error(), "boom-1")
			assert.Contains(t, err.Error(), worker.ID.String())
		default:
			t.Fatal("expected a fatal error to be delivered")
		}

		// Subsequent calls must not deliver again.
		worker.fail("boom-2")
		worker.fail("boom-3")
		select {
		case err := <-worker.fatalErr:
			t.Fatalf("fail() delivered a second error: %v", err)
		default:
			// expected: channel empty
		}

		assert.True(t, worker.failed.Load(), "failed flag should remain set")
	})
}

func TestWorker_Fail_PrePopulatedChannel(t *testing.T) {
	t.Run("does not block when channel already holds an error", func(t *testing.T) {
		cfg := &config.JobsConfig{MaxConcurrentPerWorker: 5}
		worker := NewWorker(cfg, nil, "secret", "http://localhost", nil, nil, nil)

		worker.fail("first")
		// Channel is now full (buffered size 1). A second fail() must not block.
		done := make(chan struct{})
		go func() {
			worker.fail("second") // non-blocking via select/default
			close(done)
		}()
		select {
		case <-done:
			// expected
		case <-time.After(time.Second):
			t.Fatal("second fail() blocked instead of dropping")
		}
	})
}

func TestWorker_SignalShutdown_Idempotent(t *testing.T) {
	t.Run("closes shutdownChan exactly once", func(t *testing.T) {
		cfg := &config.JobsConfig{MaxConcurrentPerWorker: 5}
		worker := NewWorker(cfg, nil, "secret", "http://localhost", nil, nil, nil)

		worker.signalShutdown()
		worker.signalShutdown() // must not panic (double close guard)
		worker.signalShutdown()

		select {
		case <-worker.shutdownChan:
			// expected: closed
		default:
			t.Fatal("shutdownChan should be closed after signalShutdown")
		}
	})
}
