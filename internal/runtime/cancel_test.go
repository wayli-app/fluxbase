package runtime

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCancelSignal(t *testing.T) {
	signal := NewCancelSignal()

	assert.NotNil(t, signal)
	assert.NotNil(t, signal.ctx)
	assert.NotNil(t, signal.cancel)
	assert.False(t, signal.cancelled)
}

func TestCancelSignal_Cancel(t *testing.T) {
	signal := NewCancelSignal()

	// Initially not cancelled
	assert.False(t, signal.IsCancelled())

	// Cancel the signal
	signal.Cancel()

	// Now it should be cancelled
	assert.True(t, signal.IsCancelled())

	// Context should be cancelled
	select {
	case <-signal.Context().Done():
		// Expected - context was cancelled
	case <-time.After(100 * time.Millisecond):
		t.Fatal("context should be cancelled")
	}
}

func TestCancelSignal_IsCancelled(t *testing.T) {
	t.Run("initially not cancelled", func(t *testing.T) {
		signal := NewCancelSignal()
		assert.False(t, signal.IsCancelled())
	})

	t.Run("returns true after cancel", func(t *testing.T) {
		signal := NewCancelSignal()
		signal.Cancel()
		assert.True(t, signal.IsCancelled())
	})

	t.Run("remains true on multiple calls", func(t *testing.T) {
		signal := NewCancelSignal()
		signal.Cancel()
		assert.True(t, signal.IsCancelled())
		assert.True(t, signal.IsCancelled())
		assert.True(t, signal.IsCancelled())
	})
}

func TestCancelSignal_Context(t *testing.T) {
	t.Run("returns valid context", func(t *testing.T) {
		signal := NewCancelSignal()
		ctx := signal.Context()

		assert.NotNil(t, ctx)
		// Context values should return nil for non-existent keys
		assert.Nil(t, ctx.Value("nonexistent"))
	})

	t.Run("context not cancelled initially", func(t *testing.T) {
		signal := NewCancelSignal()
		ctx := signal.Context()

		select {
		case <-ctx.Done():
			t.Fatal("context should not be cancelled initially")
		default:
			// Expected
		}
	})

	t.Run("context cancelled after Cancel", func(t *testing.T) {
		signal := NewCancelSignal()
		ctx := signal.Context()

		signal.Cancel()

		select {
		case <-ctx.Done():
			// Expected
			assert.Equal(t, context.Canceled, ctx.Err())
		case <-time.After(100 * time.Millisecond):
			t.Fatal("context should be cancelled")
		}
	})
}

func TestCancelSignal_ConcurrentAccess(t *testing.T) {
	t.Run("concurrent IsCancelled calls", func(t *testing.T) {
		signal := NewCancelSignal()
		var wg sync.WaitGroup

		// Spawn multiple goroutines checking cancellation
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_ = signal.IsCancelled()
			}()
		}

		wg.Wait()
	})

	t.Run("concurrent Cancel and IsCancelled", func(t *testing.T) {
		signal := NewCancelSignal()
		var wg sync.WaitGroup

		// Goroutines calling Cancel
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				signal.Cancel()
			}()
		}

		// Goroutines checking IsCancelled
		for i := 0; i < 50; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_ = signal.IsCancelled()
			}()
		}

		wg.Wait()

		// After all goroutines complete, should be cancelled
		assert.True(t, signal.IsCancelled())
	})
}

func TestCancelSignal_IdempotentCancel(t *testing.T) {
	signal := NewCancelSignal()

	// Cancel multiple times
	signal.Cancel()
	signal.Cancel()
	signal.Cancel()

	// Should still be cancelled (no panic or errors)
	assert.True(t, signal.IsCancelled())
}

func TestCancelSignal_ContextPropagation(t *testing.T) {
	t.Run("context can be used in select", func(t *testing.T) {
		signal := NewCancelSignal()
		ctx := signal.Context()

		done := make(chan bool)
		go func() {
			select {
			case <-ctx.Done():
				done <- true
			case <-time.After(1 * time.Second):
				done <- false
			}
		}()

		// Give goroutine time to start
		time.Sleep(10 * time.Millisecond)

		// Cancel the signal
		signal.Cancel()

		// Should receive true (context was cancelled)
		result := <-done
		assert.True(t, result, "context should have been cancelled")
	})

	t.Run("can derive child context", func(t *testing.T) {
		signal := NewCancelSignal()
		parentCtx := signal.Context()

		childCtx, childCancel := context.WithCancel(parentCtx)
		defer childCancel()

		// Cancel parent
		signal.Cancel()

		// Child should also be cancelled
		select {
		case <-childCtx.Done():
			// Expected
			assert.Equal(t, context.Canceled, childCtx.Err())
		case <-time.After(100 * time.Millisecond):
			t.Fatal("child context should be cancelled when parent is cancelled")
		}
	})
}

func TestCancelSignal_NilCancelFunc(t *testing.T) {
	// This tests the safety check in Cancel() for nil cancel func
	signal := &CancelSignal{
		ctx:       context.Background(),
		cancel:    nil, // Explicitly set to nil
		cancelled: false,
	}

	// Should not panic
	signal.Cancel()

	// Should still mark as cancelled
	assert.True(t, signal.IsCancelled())
}

func TestCancelSignal_Integration(t *testing.T) {
	t.Run("typical usage pattern", func(t *testing.T) {
		// Create signal
		signal := NewCancelSignal()
		require.NotNil(t, signal)

		// Check initial state
		assert.False(t, signal.IsCancelled())

		// Use context in a goroutine
		processed := make(chan bool)
		go func() {
			ticker := time.NewTicker(10 * time.Millisecond)
			defer ticker.Stop()

			for {
				select {
				case <-signal.Context().Done():
					processed <- false
					return
				case <-ticker.C:
					if signal.IsCancelled() {
						processed <- false
						return
					}
				}
			}
		}()

		// Let it run briefly
		time.Sleep(25 * time.Millisecond)

		// Cancel execution
		signal.Cancel()

		// Verify cancellation propagated
		select {
		case result := <-processed:
			assert.False(t, result) // Goroutine detected cancellation
		case <-time.After(200 * time.Millisecond):
			t.Fatal("goroutine did not detect cancellation")
		}

		// Verify state
		assert.True(t, signal.IsCancelled())
	})
}
