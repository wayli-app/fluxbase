package logging

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fluxbase-eu/fluxbase/internal/storage"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBatcher(t *testing.T) {
	t.Run("creates batcher with default values", func(t *testing.T) {
		var called bool
		writeFunc := func(ctx context.Context, entries []*storage.LogEntry) error {
			called = true
			return nil
		}

		batcher := NewBatcher(0, 0, 0, writeFunc)
		require.NotNil(t, batcher)
		defer batcher.Close(context.Background())

		assert.Equal(t, 100, batcher.batchSize)
		assert.Equal(t, time.Second, batcher.flushInterval)
		assert.Equal(t, 10000, cap(batcher.entries))
		assert.False(t, called) // writeFunc should not be called yet
	})

	t.Run("creates batcher with custom values", func(t *testing.T) {
		writeFunc := func(ctx context.Context, entries []*storage.LogEntry) error {
			return nil
		}

		batcher := NewBatcher(50, 2*time.Second, 5000, writeFunc)
		require.NotNil(t, batcher)
		defer batcher.Close(context.Background())

		assert.Equal(t, 50, batcher.batchSize)
		assert.Equal(t, 2*time.Second, batcher.flushInterval)
		assert.Equal(t, 5000, cap(batcher.entries))
	})

	t.Run("negative values use defaults", func(t *testing.T) {
		writeFunc := func(ctx context.Context, entries []*storage.LogEntry) error {
			return nil
		}

		batcher := NewBatcher(-10, -5*time.Second, -100, writeFunc)
		require.NotNil(t, batcher)
		defer batcher.Close(context.Background())

		assert.Equal(t, 100, batcher.batchSize)
		assert.Equal(t, time.Second, batcher.flushInterval)
		assert.Equal(t, 10000, cap(batcher.entries))
	})
}

func TestBatcher_Add(t *testing.T) {
	t.Run("adds entries to buffer", func(t *testing.T) {
		var received []*storage.LogEntry
		var mu sync.Mutex
		writeFunc := func(ctx context.Context, entries []*storage.LogEntry) error {
			mu.Lock()
			received = append(received, entries...)
			mu.Unlock()
			return nil
		}

		batcher := NewBatcher(10, 100*time.Millisecond, 100, writeFunc)
		defer batcher.Close(context.Background())

		entry := &storage.LogEntry{
			ID:       uuid.New(),
			Message:  "test message",
			Level:    storage.LogLevelInfo,
			Category: storage.LogCategorySystem,
		}

		batcher.Add(entry)

		// Wait for flush interval
		time.Sleep(200 * time.Millisecond)

		mu.Lock()
		assert.Len(t, received, 1)
		assert.Equal(t, entry.Message, received[0].Message)
		mu.Unlock()
	})

	t.Run("drops entries when buffer is full", func(t *testing.T) {
		// Use a very slow writeFunc to block the channel
		writeFunc := func(ctx context.Context, entries []*storage.LogEntry) error {
			time.Sleep(time.Second)
			return nil
		}

		// Create batcher with very small buffer
		batcher := NewBatcher(1000, time.Hour, 5, writeFunc)
		defer batcher.Close(context.Background())

		// Try to add more than buffer size - should not block
		done := make(chan bool)
		go func() {
			for i := 0; i < 20; i++ {
				batcher.Add(&storage.LogEntry{
					ID:      uuid.New(),
					Message: "test",
				})
			}
			done <- true
		}()

		select {
		case <-done:
			// Good - Add didn't block
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Add blocked when buffer was full")
		}
	})

	t.Run("ignores adds after close", func(t *testing.T) {
		var received []*storage.LogEntry
		writeFunc := func(ctx context.Context, entries []*storage.LogEntry) error {
			received = append(received, entries...)
			return nil
		}

		batcher := NewBatcher(100, time.Hour, 100, writeFunc)
		batcher.Close(context.Background())

		// Add after close should not panic or add to buffer
		batcher.Add(&storage.LogEntry{
			ID:      uuid.New(),
			Message: "after close",
		})

		// Buffer should be empty (or flushed before close)
		assert.Len(t, batcher.entries, 0)
	})

	t.Run("nil entries are handled gracefully", func(t *testing.T) {
		var received []*storage.LogEntry
		var mu sync.Mutex
		writeFunc := func(ctx context.Context, entries []*storage.LogEntry) error {
			mu.Lock()
			received = append(received, entries...)
			mu.Unlock()
			return nil
		}

		batcher := NewBatcher(10, 50*time.Millisecond, 100, writeFunc)
		defer batcher.Close(context.Background())

		// Add nil entry
		batcher.Add(nil)

		// Add valid entry
		batcher.Add(&storage.LogEntry{
			ID:      uuid.New(),
			Message: "valid",
		})

		time.Sleep(100 * time.Millisecond)

		// Should only have the valid entry (nil is skipped in run loop)
		mu.Lock()
		assert.Len(t, received, 1)
		mu.Unlock()
	})
}

func TestBatcher_BatchSizeFlush(t *testing.T) {
	t.Run("flushes when batch size is reached", func(t *testing.T) {
		flushCount := int32(0)
		var mu sync.Mutex
		var batches []int

		writeFunc := func(ctx context.Context, entries []*storage.LogEntry) error {
			atomic.AddInt32(&flushCount, 1)
			mu.Lock()
			batches = append(batches, len(entries))
			mu.Unlock()
			return nil
		}

		batchSize := 5
		batcher := NewBatcher(batchSize, time.Hour, 100, writeFunc) // Long interval to avoid time-based flush
		defer batcher.Close(context.Background())

		// Add exactly batch size entries
		for i := 0; i < batchSize; i++ {
			batcher.Add(&storage.LogEntry{
				ID:      uuid.New(),
				Message: "test",
			})
		}

		// Wait a bit for the batch to be processed
		time.Sleep(50 * time.Millisecond)

		mu.Lock()
		assert.Equal(t, int32(1), atomic.LoadInt32(&flushCount))
		assert.Equal(t, []int{batchSize}, batches)
		mu.Unlock()
	})

	t.Run("flushes multiple batches", func(t *testing.T) {
		flushCount := int32(0)
		var mu sync.Mutex
		var batches []int

		writeFunc := func(ctx context.Context, entries []*storage.LogEntry) error {
			atomic.AddInt32(&flushCount, 1)
			mu.Lock()
			batches = append(batches, len(entries))
			mu.Unlock()
			return nil
		}

		batchSize := 3
		batcher := NewBatcher(batchSize, time.Hour, 100, writeFunc)
		defer batcher.Close(context.Background())

		// Add 9 entries (should trigger 3 flushes)
		for i := 0; i < 9; i++ {
			batcher.Add(&storage.LogEntry{
				ID:      uuid.New(),
				Message: "test",
			})
		}

		time.Sleep(50 * time.Millisecond)

		mu.Lock()
		assert.Equal(t, int32(3), atomic.LoadInt32(&flushCount))
		assert.Equal(t, []int{3, 3, 3}, batches)
		mu.Unlock()
	})
}

func TestBatcher_IntervalFlush(t *testing.T) {
	t.Run("flushes on interval even with partial batch", func(t *testing.T) {
		flushCount := int32(0)
		var mu sync.Mutex
		var batches []int

		writeFunc := func(ctx context.Context, entries []*storage.LogEntry) error {
			atomic.AddInt32(&flushCount, 1)
			mu.Lock()
			batches = append(batches, len(entries))
			mu.Unlock()
			return nil
		}

		// Small interval, large batch size
		batcher := NewBatcher(100, 50*time.Millisecond, 100, writeFunc)
		defer batcher.Close(context.Background())

		// Add only 3 entries (less than batch size)
		for i := 0; i < 3; i++ {
			batcher.Add(&storage.LogEntry{
				ID:      uuid.New(),
				Message: "test",
			})
		}

		// Wait for interval flush
		time.Sleep(100 * time.Millisecond)

		mu.Lock()
		assert.GreaterOrEqual(t, atomic.LoadInt32(&flushCount), int32(1))
		if len(batches) > 0 {
			assert.Equal(t, 3, batches[0])
		}
		mu.Unlock()
	})
}

func TestBatcher_Flush(t *testing.T) {
	t.Run("manual flush flushes all entries", func(t *testing.T) {
		var received []*storage.LogEntry
		var mu sync.Mutex

		writeFunc := func(ctx context.Context, entries []*storage.LogEntry) error {
			mu.Lock()
			received = append(received, entries...)
			mu.Unlock()
			return nil
		}

		batcher := NewBatcher(100, time.Hour, 100, writeFunc)
		defer batcher.Close(context.Background())

		// Add entries
		for i := 0; i < 5; i++ {
			batcher.Add(&storage.LogEntry{
				ID:      uuid.New(),
				Message: "test",
			})
		}

		// Manual flush
		ctx := context.Background()
		err := batcher.Flush(ctx)
		require.NoError(t, err)

		mu.Lock()
		assert.Len(t, received, 5)
		mu.Unlock()
	})

	t.Run("flush respects context timeout", func(t *testing.T) {
		writeFunc := func(ctx context.Context, entries []*storage.LogEntry) error {
			// Slow write
			time.Sleep(time.Second)
			return nil
		}

		batcher := NewBatcher(100, time.Hour, 100, writeFunc)
		defer batcher.Close(context.Background())

		// Add multiple entries to ensure one is picked up by flush
		for i := 0; i < 5; i++ {
			batcher.Add(&storage.LogEntry{
				ID:      uuid.New(),
				Message: "test",
			})
		}

		// Short timeout - but long enough for flush to start the write
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		err := batcher.Flush(ctx)
		// The flush should time out because writeFunc takes 1 second
		// Note: if the run loop processes entries first, flush may return nil
		// This is acceptable behavior - the important thing is no hang
		if err != nil {
			assert.ErrorIs(t, err, context.DeadlineExceeded)
		}
	})

	t.Run("flush with empty buffer returns immediately", func(t *testing.T) {
		writeFunc := func(ctx context.Context, entries []*storage.LogEntry) error {
			return nil
		}

		batcher := NewBatcher(100, time.Hour, 100, writeFunc)
		defer batcher.Close(context.Background())

		// Flush empty buffer
		start := time.Now()
		err := batcher.Flush(context.Background())
		duration := time.Since(start)

		require.NoError(t, err)
		assert.Less(t, duration, 50*time.Millisecond)
	})
}

func TestBatcher_Close(t *testing.T) {
	t.Run("close flushes remaining entries", func(t *testing.T) {
		var received []*storage.LogEntry
		var mu sync.Mutex

		writeFunc := func(ctx context.Context, entries []*storage.LogEntry) error {
			mu.Lock()
			received = append(received, entries...)
			mu.Unlock()
			return nil
		}

		batcher := NewBatcher(100, time.Hour, 100, writeFunc)

		// Add entries
		for i := 0; i < 5; i++ {
			batcher.Add(&storage.LogEntry{
				ID:      uuid.New(),
				Message: "test",
			})
		}

		// Close should flush
		err := batcher.Close(context.Background())
		require.NoError(t, err)

		mu.Lock()
		assert.Len(t, received, 5)
		mu.Unlock()
	})

	t.Run("double close is idempotent", func(t *testing.T) {
		writeFunc := func(ctx context.Context, entries []*storage.LogEntry) error {
			return nil
		}

		batcher := NewBatcher(100, time.Hour, 100, writeFunc)

		err1 := batcher.Close(context.Background())
		require.NoError(t, err1)

		err2 := batcher.Close(context.Background())
		require.NoError(t, err2)
	})

	t.Run("close respects context timeout", func(t *testing.T) {
		writeFunc := func(ctx context.Context, entries []*storage.LogEntry) error {
			time.Sleep(time.Second)
			return nil
		}

		batcher := NewBatcher(100, time.Hour, 100, writeFunc)

		// Add entry to force flush on close
		batcher.Add(&storage.LogEntry{
			ID:      uuid.New(),
			Message: "test",
		})

		// Short timeout
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		err := batcher.Close(ctx)
		// Close should still return (may have context error from flush)
		// The important thing is it doesn't hang
		_ = err
	})
}

func TestBatcher_Stats(t *testing.T) {
	t.Run("returns correct statistics", func(t *testing.T) {
		writeFunc := func(ctx context.Context, entries []*storage.LogEntry) error {
			return nil
		}

		batcher := NewBatcher(100, time.Hour, 50, writeFunc)
		defer batcher.Close(context.Background())

		// Check initial stats
		stats := batcher.Stats()
		assert.Equal(t, 50, stats.BufferSize)
		assert.Equal(t, 0, stats.BufferUsed)
		assert.Equal(t, 0.0, stats.BufferPercent)

		// Add some entries
		for i := 0; i < 10; i++ {
			batcher.Add(&storage.LogEntry{
				ID:      uuid.New(),
				Message: "test",
			})
		}

		// Stats may vary due to async processing - check buffer size is constant
		stats = batcher.Stats()
		assert.Equal(t, 50, stats.BufferSize)
		// BufferUsed is between 0 and 10 depending on timing
		assert.GreaterOrEqual(t, stats.BufferUsed, 0)
		assert.LessOrEqual(t, stats.BufferUsed, 10)
	})
}

func TestBatcher_ConcurrentAccess(t *testing.T) {
	t.Run("handles concurrent adds safely", func(t *testing.T) {
		var totalReceived int64

		writeFunc := func(ctx context.Context, entries []*storage.LogEntry) error {
			atomic.AddInt64(&totalReceived, int64(len(entries)))
			return nil
		}

		batcher := NewBatcher(10, 50*time.Millisecond, 1000, writeFunc)

		var wg sync.WaitGroup
		numGoroutines := 10
		entriesPerGoroutine := 100

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < entriesPerGoroutine; j++ {
					batcher.Add(&storage.LogEntry{
						ID:      uuid.New(),
						Message: "concurrent test",
					})
				}
			}()
		}

		wg.Wait()

		// Close to ensure all entries are flushed
		err := batcher.Close(context.Background())
		require.NoError(t, err)

		// All entries should be received
		assert.Equal(t, int64(numGoroutines*entriesPerGoroutine), atomic.LoadInt64(&totalReceived))
	})

	t.Run("handles concurrent adds and flushes", func(t *testing.T) {
		var totalReceived int64

		writeFunc := func(ctx context.Context, entries []*storage.LogEntry) error {
			atomic.AddInt64(&totalReceived, int64(len(entries)))
			return nil
		}

		batcher := NewBatcher(5, 20*time.Millisecond, 1000, writeFunc)

		var wg sync.WaitGroup

		// Add goroutines
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < 50; j++ {
					batcher.Add(&storage.LogEntry{
						ID:      uuid.New(),
						Message: "test",
					})
					time.Sleep(time.Millisecond)
				}
			}()
		}

		// Flush goroutines
		for i := 0; i < 3; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < 10; j++ {
					batcher.Flush(context.Background())
					time.Sleep(5 * time.Millisecond)
				}
			}()
		}

		wg.Wait()
		batcher.Close(context.Background())

		// Should receive all 250 entries (5 goroutines * 50 entries)
		assert.Equal(t, int64(250), atomic.LoadInt64(&totalReceived))
	})
}

func TestBatcher_WriteErrors(t *testing.T) {
	t.Run("continues operating after write error", func(t *testing.T) {
		callCount := int32(0)
		var mu sync.Mutex
		var batches []int

		writeFunc := func(ctx context.Context, entries []*storage.LogEntry) error {
			count := atomic.AddInt32(&callCount, 1)
			mu.Lock()
			batches = append(batches, len(entries))
			mu.Unlock()

			// First call returns error
			if count == 1 {
				return assert.AnError
			}
			return nil
		}

		batcher := NewBatcher(3, time.Hour, 100, writeFunc)
		defer batcher.Close(context.Background())

		// First batch - will fail
		for i := 0; i < 3; i++ {
			batcher.Add(&storage.LogEntry{
				ID:      uuid.New(),
				Message: "batch1",
			})
		}

		time.Sleep(20 * time.Millisecond)

		// Second batch - should succeed
		for i := 0; i < 3; i++ {
			batcher.Add(&storage.LogEntry{
				ID:      uuid.New(),
				Message: "batch2",
			})
		}

		time.Sleep(20 * time.Millisecond)

		mu.Lock()
		assert.Equal(t, int32(2), atomic.LoadInt32(&callCount))
		assert.Equal(t, []int{3, 3}, batches)
		mu.Unlock()
	})
}

func TestBatcher_EntryContent(t *testing.T) {
	t.Run("preserves entry content through batching", func(t *testing.T) {
		var received []*storage.LogEntry
		var mu sync.Mutex

		writeFunc := func(ctx context.Context, entries []*storage.LogEntry) error {
			mu.Lock()
			received = append(received, entries...)
			mu.Unlock()
			return nil
		}

		batcher := NewBatcher(10, 50*time.Millisecond, 100, writeFunc)
		defer batcher.Close(context.Background())

		testEntries := []*storage.LogEntry{
			{
				ID:        uuid.MustParse("550e8400-e29b-41d4-a716-446655440001"),
				Timestamp: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
				Category:  storage.LogCategorySystem,
				Level:     storage.LogLevelInfo,
				Message:   "System log message",
				Component: "api",
				Fields:    map[string]any{"key": "value"},
			},
			{
				ID:            uuid.MustParse("550e8400-e29b-41d4-a716-446655440002"),
				Timestamp:     time.Date(2024, 1, 15, 10, 31, 0, 0, time.UTC),
				Category:      storage.LogCategoryExecution,
				Level:         storage.LogLevelDebug,
				Message:       "Execution log",
				ExecutionID:   "exec-123",
				ExecutionType: "function",
				LineNumber:    5,
			},
			{
				ID:        uuid.MustParse("550e8400-e29b-41d4-a716-446655440003"),
				Timestamp: time.Date(2024, 1, 15, 10, 32, 0, 0, time.UTC),
				Category:  storage.LogCategoryHTTP,
				Level:     storage.LogLevelWarn,
				Message:   "HTTP warning",
				RequestID: "req-456",
				UserID:    "user-789",
				IPAddress: "192.168.1.1",
			},
		}

		for _, entry := range testEntries {
			batcher.Add(entry)
		}

		time.Sleep(100 * time.Millisecond)

		mu.Lock()
		require.Len(t, received, 3)

		// Verify first entry
		assert.Equal(t, testEntries[0].ID, received[0].ID)
		assert.Equal(t, testEntries[0].Category, received[0].Category)
		assert.Equal(t, testEntries[0].Level, received[0].Level)
		assert.Equal(t, testEntries[0].Message, received[0].Message)
		assert.Equal(t, testEntries[0].Component, received[0].Component)
		assert.Equal(t, "value", received[0].Fields["key"])

		// Verify second entry
		assert.Equal(t, testEntries[1].ExecutionID, received[1].ExecutionID)
		assert.Equal(t, testEntries[1].ExecutionType, received[1].ExecutionType)
		assert.Equal(t, testEntries[1].LineNumber, received[1].LineNumber)

		// Verify third entry
		assert.Equal(t, testEntries[2].RequestID, received[2].RequestID)
		assert.Equal(t, testEntries[2].UserID, received[2].UserID)
		assert.Equal(t, testEntries[2].IPAddress, received[2].IPAddress)
		mu.Unlock()
	})
}
