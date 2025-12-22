package logging

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/fluxbase-eu/fluxbase/internal/pubsub"
	"github.com/fluxbase-eu/fluxbase/internal/storage"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockPubSub is a simple mock for testing
type mockPubSub struct {
	mu        sync.Mutex
	published []publishedMessage
	closed    bool
	publishFn func(ctx context.Context, channel string, payload []byte) error
}

type publishedMessage struct {
	channel string
	payload []byte
}

func newMockPubSub() *mockPubSub {
	return &mockPubSub{
		published: make([]publishedMessage, 0),
	}
}

func (m *mockPubSub) Publish(ctx context.Context, channel string, payload []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.publishFn != nil {
		return m.publishFn(ctx, channel, payload)
	}

	m.published = append(m.published, publishedMessage{
		channel: channel,
		payload: payload,
	})
	return nil
}

func (m *mockPubSub) Subscribe(ctx context.Context, channel string) (<-chan pubsub.Message, error) {
	ch := make(chan pubsub.Message)
	return ch, nil
}

func (m *mockPubSub) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}

func (m *mockPubSub) getPublished() []publishedMessage {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]publishedMessage, len(m.published))
	copy(result, m.published)
	return result
}

func TestCategoryChannel(t *testing.T) {
	tests := []struct {
		category string
		expected string
	}{
		{"system", "fluxbase:logs:system"},
		{"http", "fluxbase:logs:http"},
		{"security", "fluxbase:logs:security"},
		{"execution", "fluxbase:logs:execution"},
		{"custom", "fluxbase:logs:custom"},
	}

	for _, tc := range tests {
		t.Run(tc.category, func(t *testing.T) {
			result := CategoryChannel(tc.category)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestNewPubSubNotifier(t *testing.T) {
	t.Run("uses default channel when empty", func(t *testing.T) {
		ps := newMockPubSub()
		notifier := NewPubSubNotifier(ps, "")

		assert.Equal(t, LogChannel, notifier.channel)
	})

	t.Run("uses provided channel", func(t *testing.T) {
		ps := newMockPubSub()
		notifier := NewPubSubNotifier(ps, "custom:channel")

		assert.Equal(t, "custom:channel", notifier.channel)
	})
}

func TestPubSubNotifier_Notify(t *testing.T) {
	t.Run("publishes system log to all-logs and category channels", func(t *testing.T) {
		ps := newMockPubSub()
		notifier := NewPubSubNotifier(ps, "")

		entry := &storage.LogEntry{
			ID:        uuid.New(),
			Timestamp: time.Now(),
			Category:  storage.LogCategorySystem,
			Level:     storage.LogLevelInfo,
			Message:   "Test system log",
			Component: "api",
		}

		err := notifier.Notify(context.Background(), entry)
		require.NoError(t, err)

		published := ps.getPublished()
		require.Len(t, published, 2) // all_logs + category

		// Check all-logs channel
		assert.Equal(t, AllLogsChannel, published[0].channel)

		// Check category channel
		assert.Equal(t, "fluxbase:logs:system", published[1].channel)

		// Verify payload
		var event storage.LogStreamEvent
		err = json.Unmarshal(published[0].payload, &event)
		require.NoError(t, err)
		assert.Equal(t, entry.Message, event.Message)
		assert.Equal(t, storage.LogCategorySystem, event.Category)
	})

	t.Run("publishes execution log to execution channel as well", func(t *testing.T) {
		ps := newMockPubSub()
		notifier := NewPubSubNotifier(ps, LogChannel)

		entry := &storage.LogEntry{
			ID:            uuid.New(),
			Timestamp:     time.Now(),
			Category:      storage.LogCategoryExecution,
			Level:         storage.LogLevelDebug,
			Message:       "Execution log message",
			ExecutionID:   "exec-123",
			ExecutionType: "function",
			LineNumber:    5,
		}

		err := notifier.Notify(context.Background(), entry)
		require.NoError(t, err)

		published := ps.getPublished()
		require.Len(t, published, 3) // all_logs + category + execution-specific

		// Check channels
		assert.Equal(t, AllLogsChannel, published[0].channel)
		assert.Equal(t, "fluxbase:logs:execution", published[1].channel)
		assert.Equal(t, LogChannel, published[2].channel)

		// Verify execution log event payload
		var execEvent storage.ExecutionLogEvent
		err = json.Unmarshal(published[2].payload, &execEvent)
		require.NoError(t, err)
		assert.Equal(t, "exec-123", execEvent.ExecutionID)
		assert.Equal(t, "function", execEvent.ExecutionType)
		assert.Equal(t, 5, execEvent.LineNumber)
		assert.Equal(t, entry.Message, execEvent.Message)
	})

	t.Run("does not publish execution log without execution ID", func(t *testing.T) {
		ps := newMockPubSub()
		notifier := NewPubSubNotifier(ps, LogChannel)

		entry := &storage.LogEntry{
			ID:        uuid.New(),
			Timestamp: time.Now(),
			Category:  storage.LogCategoryExecution,
			Level:     storage.LogLevelInfo,
			Message:   "Execution without ID",
			// ExecutionID is empty
		}

		err := notifier.Notify(context.Background(), entry)
		require.NoError(t, err)

		published := ps.getPublished()
		// Should only publish to all_logs and category, not execution channel
		require.Len(t, published, 2)
	})

	t.Run("publishes HTTP log correctly", func(t *testing.T) {
		ps := newMockPubSub()
		notifier := NewPubSubNotifier(ps, "")

		entry := &storage.LogEntry{
			ID:        uuid.New(),
			Timestamp: time.Now(),
			Category:  storage.LogCategoryHTTP,
			Level:     storage.LogLevelInfo,
			Message:   "GET /api/users 200",
			RequestID: "req-456",
			UserID:    "user-789",
			IPAddress: "192.168.1.1",
		}

		err := notifier.Notify(context.Background(), entry)
		require.NoError(t, err)

		published := ps.getPublished()
		require.Len(t, published, 2)

		// Verify stream event has all fields
		var event storage.LogStreamEvent
		err = json.Unmarshal(published[0].payload, &event)
		require.NoError(t, err)
		assert.Equal(t, storage.LogCategoryHTTP, event.Category)
		assert.Equal(t, "req-456", event.RequestID)
		assert.Equal(t, "user-789", event.UserID)
		assert.Equal(t, "192.168.1.1", event.IPAddress)
	})

	t.Run("publishes security log correctly", func(t *testing.T) {
		ps := newMockPubSub()
		notifier := NewPubSubNotifier(ps, "")

		entry := &storage.LogEntry{
			ID:        uuid.New(),
			Timestamp: time.Now(),
			Category:  storage.LogCategorySecurity,
			Level:     storage.LogLevelWarn,
			Message:   "Failed login attempt",
			UserID:    "user-123",
			IPAddress: "10.0.0.1",
		}

		err := notifier.Notify(context.Background(), entry)
		require.NoError(t, err)

		published := ps.getPublished()
		assert.Equal(t, "fluxbase:logs:security", published[1].channel)
	})

	t.Run("includes custom fields in payload", func(t *testing.T) {
		ps := newMockPubSub()
		notifier := NewPubSubNotifier(ps, "")

		entry := &storage.LogEntry{
			ID:        uuid.New(),
			Timestamp: time.Now(),
			Category:  storage.LogCategorySystem,
			Level:     storage.LogLevelInfo,
			Message:   "Test with fields",
			Fields: map[string]any{
				"custom_key":  "custom_value",
				"numeric_key": 123,
				"boolean_key": true,
			},
		}

		err := notifier.Notify(context.Background(), entry)
		require.NoError(t, err)

		published := ps.getPublished()
		var event storage.LogStreamEvent
		err = json.Unmarshal(published[0].payload, &event)
		require.NoError(t, err)

		assert.Equal(t, "custom_value", event.Fields["custom_key"])
		assert.Equal(t, float64(123), event.Fields["numeric_key"]) // JSON numbers are float64
		assert.Equal(t, true, event.Fields["boolean_key"])
	})
}

func TestPubSubNotifier_NotifyBatch(t *testing.T) {
	t.Run("notifies all entries in batch", func(t *testing.T) {
		ps := newMockPubSub()
		notifier := NewPubSubNotifier(ps, "")

		entries := []*storage.LogEntry{
			{
				ID:        uuid.New(),
				Timestamp: time.Now(),
				Category:  storage.LogCategorySystem,
				Level:     storage.LogLevelInfo,
				Message:   "First message",
			},
			{
				ID:        uuid.New(),
				Timestamp: time.Now(),
				Category:  storage.LogCategoryHTTP,
				Level:     storage.LogLevelInfo,
				Message:   "Second message",
			},
			{
				ID:        uuid.New(),
				Timestamp: time.Now(),
				Category:  storage.LogCategorySecurity,
				Level:     storage.LogLevelWarn,
				Message:   "Third message",
			},
		}

		err := notifier.NotifyBatch(context.Background(), entries)
		require.NoError(t, err)

		published := ps.getPublished()
		// Each entry publishes to 2 channels (all_logs + category)
		assert.Len(t, published, 6)
	})

	t.Run("continues on error", func(t *testing.T) {
		callCount := 0
		ps := &mockPubSub{
			publishFn: func(ctx context.Context, channel string, payload []byte) error {
				callCount++
				if callCount == 1 {
					return assert.AnError
				}
				return nil
			},
		}
		notifier := NewPubSubNotifier(ps, "")

		entries := []*storage.LogEntry{
			{
				ID:        uuid.New(),
				Timestamp: time.Now(),
				Category:  storage.LogCategorySystem,
				Level:     storage.LogLevelInfo,
				Message:   "First",
			},
			{
				ID:        uuid.New(),
				Timestamp: time.Now(),
				Category:  storage.LogCategorySystem,
				Level:     storage.LogLevelInfo,
				Message:   "Second",
			},
		}

		// Should not return error even if some publishes fail
		err := notifier.NotifyBatch(context.Background(), entries)
		assert.NoError(t, err)

		// Should have attempted to publish multiple times
		assert.Greater(t, callCount, 1)
	})

	t.Run("handles empty batch", func(t *testing.T) {
		ps := newMockPubSub()
		notifier := NewPubSubNotifier(ps, "")

		err := notifier.NotifyBatch(context.Background(), []*storage.LogEntry{})
		assert.NoError(t, err)

		published := ps.getPublished()
		assert.Len(t, published, 0)
	})
}

func TestChannelConstants(t *testing.T) {
	assert.Equal(t, "fluxbase:logs", LogChannel)
	assert.Equal(t, "fluxbase:all_logs", AllLogsChannel)
	assert.Equal(t, "fluxbase:logs:", CategoryChannelPrefix)
}
