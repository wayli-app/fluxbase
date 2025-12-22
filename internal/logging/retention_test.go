package logging

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fluxbase-eu/fluxbase/internal/config"
	"github.com/fluxbase-eu/fluxbase/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockLogStorage implements storage.LogStorage for testing
type mockLogStorage struct {
	mu          sync.Mutex
	deleteCalls []deleteCall
	deleteCount int64
	deleteErr   error
}

type deleteCall struct {
	opts storage.LogQueryOptions
}

func newMockLogStorage() *mockLogStorage {
	return &mockLogStorage{
		deleteCalls: make([]deleteCall, 0),
	}
}

func (m *mockLogStorage) Name() string {
	return "mock"
}

func (m *mockLogStorage) Health(ctx context.Context) error {
	return nil
}

func (m *mockLogStorage) Write(ctx context.Context, entries []*storage.LogEntry) error {
	return nil
}

func (m *mockLogStorage) Query(ctx context.Context, opts storage.LogQueryOptions) (*storage.LogQueryResult, error) {
	return &storage.LogQueryResult{}, nil
}

func (m *mockLogStorage) GetExecutionLogs(ctx context.Context, executionID string, afterLine int) ([]*storage.LogEntry, error) {
	return nil, nil
}

func (m *mockLogStorage) Stats(ctx context.Context) (*storage.LogStats, error) {
	return &storage.LogStats{}, nil
}

func (m *mockLogStorage) Delete(ctx context.Context, opts storage.LogQueryOptions) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Always track the call, even if we're going to return an error
	m.deleteCalls = append(m.deleteCalls, deleteCall{opts: opts})

	if m.deleteErr != nil {
		return 0, m.deleteErr
	}

	return m.deleteCount, nil
}

func (m *mockLogStorage) Close() error {
	return nil
}

func (m *mockLogStorage) getDeleteCalls() []deleteCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]deleteCall, len(m.deleteCalls))
	copy(result, m.deleteCalls)
	return result
}

func TestNewRetentionService(t *testing.T) {
	t.Run("uses default interval", func(t *testing.T) {
		cfg := &config.LoggingConfig{
			SystemRetentionDays: 30,
		}
		storage := newMockLogStorage()

		svc := NewRetentionService(cfg, storage)
		require.NotNil(t, svc)

		assert.Equal(t, 24*time.Hour, svc.interval)
	})

	t.Run("uses custom interval", func(t *testing.T) {
		cfg := &config.LoggingConfig{
			RetentionCheckInterval: 12 * time.Hour,
		}
		storage := newMockLogStorage()

		svc := NewRetentionService(cfg, storage)
		require.NotNil(t, svc)

		assert.Equal(t, 12*time.Hour, svc.interval)
	})

	t.Run("uses default interval for zero value", func(t *testing.T) {
		cfg := &config.LoggingConfig{
			RetentionCheckInterval: 0,
		}
		storage := newMockLogStorage()

		svc := NewRetentionService(cfg, storage)
		require.NotNil(t, svc)

		assert.Equal(t, 24*time.Hour, svc.interval)
	})
}

func TestRetentionService_StartStop(t *testing.T) {
	t.Run("start is idempotent", func(t *testing.T) {
		cfg := &config.LoggingConfig{
			RetentionCheckInterval: time.Hour,
		}
		mockStorage := newMockLogStorage()
		svc := NewRetentionService(cfg, mockStorage)

		// Start multiple times
		svc.Start()
		svc.Start()
		svc.Start()

		// Should still only have one goroutine running
		assert.True(t, svc.running)

		svc.Stop()
		assert.False(t, svc.running)
	})

	t.Run("stop is idempotent", func(t *testing.T) {
		cfg := &config.LoggingConfig{
			RetentionCheckInterval: time.Hour,
		}
		mockStorage := newMockLogStorage()
		svc := NewRetentionService(cfg, mockStorage)

		svc.Start()
		svc.Stop()
		svc.Stop()
		svc.Stop()

		// Should not panic or block
		assert.False(t, svc.running)
	})

	t.Run("cleanup runs on start", func(t *testing.T) {
		cfg := &config.LoggingConfig{
			RetentionCheckInterval: time.Hour,
			SystemRetentionDays:    30,
		}
		mockStorage := newMockLogStorage()
		mockStorage.deleteCount = 10

		svc := NewRetentionService(cfg, mockStorage)
		svc.Start()

		// Give time for cleanup to run
		time.Sleep(50 * time.Millisecond)

		svc.Stop()

		// Should have called delete at least once
		calls := mockStorage.getDeleteCalls()
		assert.Greater(t, len(calls), 0)
	})
}

func TestRetentionService_Cleanup(t *testing.T) {
	t.Run("deletes logs for categories with retention", func(t *testing.T) {
		cfg := &config.LoggingConfig{
			SystemRetentionDays:    30,
			HTTPRetentionDays:      7,
			SecurityRetentionDays:  90,
			ExecutionRetentionDays: 14,
			AIRetentionDays:        60,
			CustomRetentionDays:    30,
		}
		mockStorage := newMockLogStorage()
		mockStorage.deleteCount = 5

		svc := NewRetentionService(cfg, mockStorage)
		svc.RunOnce()

		calls := mockStorage.getDeleteCalls()
		// Should have 6 delete calls (one per category)
		assert.Len(t, calls, 6)

		// Verify categories
		categories := make(map[storage.LogCategory]bool)
		for _, call := range calls {
			categories[call.opts.Category] = true
		}

		assert.True(t, categories[storage.LogCategorySystem])
		assert.True(t, categories[storage.LogCategoryHTTP])
		assert.True(t, categories[storage.LogCategorySecurity])
		assert.True(t, categories[storage.LogCategoryExecution])
		assert.True(t, categories[storage.LogCategoryAI])
		assert.True(t, categories[storage.LogCategoryCustom])
	})

	t.Run("skips categories with zero retention", func(t *testing.T) {
		cfg := &config.LoggingConfig{
			SystemRetentionDays:    30,
			HTTPRetentionDays:      0, // Skip
			SecurityRetentionDays:  0, // Skip
			ExecutionRetentionDays: 14,
			AIRetentionDays:        0, // Skip
			CustomRetentionDays:    0, // Skip
		}
		mockStorage := newMockLogStorage()
		mockStorage.deleteCount = 5

		svc := NewRetentionService(cfg, mockStorage)
		svc.RunOnce()

		calls := mockStorage.getDeleteCalls()
		// Should only have 2 delete calls (system and execution)
		assert.Len(t, calls, 2)

		categories := make(map[storage.LogCategory]bool)
		for _, call := range calls {
			categories[call.opts.Category] = true
		}

		assert.True(t, categories[storage.LogCategorySystem])
		assert.True(t, categories[storage.LogCategoryExecution])
		assert.False(t, categories[storage.LogCategoryHTTP])
		assert.False(t, categories[storage.LogCategorySecurity])
	})

	t.Run("calculates correct cutoff time", func(t *testing.T) {
		cfg := &config.LoggingConfig{
			SystemRetentionDays: 7,
		}
		mockStorage := newMockLogStorage()

		svc := NewRetentionService(cfg, mockStorage)

		before := time.Now().AddDate(0, 0, -7)
		svc.RunOnce()
		after := time.Now().AddDate(0, 0, -7)

		calls := mockStorage.getDeleteCalls()
		require.Len(t, calls, 1)

		// Cutoff should be approximately 7 days ago
		cutoff := calls[0].opts.EndTime
		assert.True(t, cutoff.After(before.Add(-time.Second)))
		assert.True(t, cutoff.Before(after.Add(time.Second)))
	})

	t.Run("continues on delete error", func(t *testing.T) {
		cfg := &config.LoggingConfig{
			SystemRetentionDays:   30,
			HTTPRetentionDays:     30,
			SecurityRetentionDays: 30,
		}
		mockStorage := &mockLogStorage{
			deleteCalls: make([]deleteCall, 0),
			deleteErr:   assert.AnError,
		}

		svc := NewRetentionService(cfg, mockStorage)
		svc.RunOnce()

		// Should have attempted all 3 deletes despite errors
		calls := mockStorage.getDeleteCalls()
		assert.Len(t, calls, 3)
	})
}

func TestRetentionService_ConcurrentStartStop(t *testing.T) {
	cfg := &config.LoggingConfig{
		RetentionCheckInterval: 50 * time.Millisecond,
		SystemRetentionDays:    1,
	}
	mockStorage := newMockLogStorage()
	svc := NewRetentionService(cfg, mockStorage)

	var wg sync.WaitGroup
	startCount := int32(0)
	stopCount := int32(0)

	// Start multiple goroutines that start/stop
	for i := 0; i < 10; i++ {
		wg.Add(2)

		go func() {
			defer wg.Done()
			svc.Start()
			atomic.AddInt32(&startCount, 1)
		}()

		go func() {
			defer wg.Done()
			time.Sleep(10 * time.Millisecond)
			svc.Stop()
			atomic.AddInt32(&stopCount, 1)
		}()
	}

	wg.Wait()

	// All operations should complete without deadlock
	assert.Equal(t, int32(10), atomic.LoadInt32(&startCount))
	assert.Equal(t, int32(10), atomic.LoadInt32(&stopCount))
}

func TestRetentionService_RunOnce(t *testing.T) {
	t.Run("can be called without starting service", func(t *testing.T) {
		cfg := &config.LoggingConfig{
			SystemRetentionDays: 30,
		}
		mockStorage := newMockLogStorage()
		mockStorage.deleteCount = 5

		svc := NewRetentionService(cfg, mockStorage)

		// RunOnce should work without Start()
		svc.RunOnce()

		calls := mockStorage.getDeleteCalls()
		assert.Greater(t, len(calls), 0)
	})

	t.Run("can be called multiple times", func(t *testing.T) {
		cfg := &config.LoggingConfig{
			SystemRetentionDays: 30,
		}
		mockStorage := newMockLogStorage()

		svc := NewRetentionService(cfg, mockStorage)

		svc.RunOnce()
		svc.RunOnce()
		svc.RunOnce()

		// Should have accumulated delete calls
		calls := mockStorage.getDeleteCalls()
		assert.Equal(t, 3, len(calls))
	})
}
