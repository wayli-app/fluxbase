package logging

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/fluxbase-eu/fluxbase/internal/config"
	"github.com/fluxbase-eu/fluxbase/internal/storage"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// serviceTestMockStorage implements storage.LogStorage for testing
type serviceTestMockStorage struct {
	mu          sync.Mutex
	entries     []*storage.LogEntry
	writeErr    error
	queryResult *storage.LogQueryResult
	stats       *storage.LogStats
	closed      bool
}

func newServiceTestMockStorage() *serviceTestMockStorage {
	return &serviceTestMockStorage{
		entries: make([]*storage.LogEntry, 0),
		queryResult: &storage.LogQueryResult{
			Entries:    make([]*storage.LogEntry, 0),
			TotalCount: 0,
			HasMore:    false,
		},
		stats: &storage.LogStats{
			TotalEntries:      0,
			EntriesByCategory: make(map[storage.LogCategory]int64),
			EntriesByLevel:    make(map[storage.LogLevel]int64),
		},
	}
}

func (m *serviceTestMockStorage) Name() string {
	return "mock"
}

func (m *serviceTestMockStorage) Health(ctx context.Context) error {
	return nil
}

func (m *serviceTestMockStorage) Write(ctx context.Context, entries []*storage.LogEntry) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.writeErr != nil {
		return m.writeErr
	}
	m.entries = append(m.entries, entries...)
	return nil
}

func (m *serviceTestMockStorage) Query(ctx context.Context, opts storage.LogQueryOptions) (*storage.LogQueryResult, error) {
	return m.queryResult, nil
}

func (m *serviceTestMockStorage) GetExecutionLogs(ctx context.Context, executionID string, afterLine int) ([]*storage.LogEntry, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []*storage.LogEntry
	for _, e := range m.entries {
		if e.ExecutionID == executionID && e.LineNumber > afterLine {
			result = append(result, e)
		}
	}
	return result, nil
}

func (m *serviceTestMockStorage) Stats(ctx context.Context) (*storage.LogStats, error) {
	return m.stats, nil
}

func (m *serviceTestMockStorage) Delete(ctx context.Context, opts storage.LogQueryOptions) (int64, error) {
	return 0, nil
}

func (m *serviceTestMockStorage) Close() error {
	m.closed = true
	return nil
}

func (m *serviceTestMockStorage) getEntries() []*storage.LogEntry {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]*storage.LogEntry, len(m.entries))
	copy(result, m.entries)
	return result
}

// mockPubSubForService implements pubsub.PubSub for testing
type mockPubSubForService struct {
	mu       sync.Mutex
	messages []struct {
		channel string
		payload []byte
	}
	closed bool
}

func newMockPubSubForService() *mockPubSubForService {
	return &mockPubSubForService{
		messages: make([]struct {
			channel string
			payload []byte
		}, 0),
	}
}

func (m *mockPubSubForService) Publish(ctx context.Context, channel string, payload []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = append(m.messages, struct {
		channel string
		payload []byte
	}{channel: channel, payload: payload})
	return nil
}

type pubsubMessage struct {
	Channel string
	Payload []byte
}

func (m *mockPubSubForService) Subscribe(ctx context.Context, channel string) (<-chan pubsubMessage, error) {
	ch := make(chan pubsubMessage)
	go func() {
		<-ctx.Done()
		close(ch)
	}()
	return ch, nil
}

func (m *mockPubSubForService) Close() error {
	m.closed = true
	return nil
}

// createTestService creates a Service for testing with mocked dependencies
func createTestService(cfg *config.LoggingConfig) (*Service, *serviceTestMockStorage) {
	mockStorage := newServiceTestMockStorage()

	// Create service manually to bypass the database/storage dependencies
	s := &Service{
		config:     cfg,
		storage:    mockStorage,
		lineNumber: make(map[string]int),
	}

	// Create a batcher with our mock
	s.batcher = NewBatcher(
		10,             // batch size
		100*time.Millisecond, // flush interval
		1000,           // buffer size
		func(ctx context.Context, entries []*storage.LogEntry) error {
			return mockStorage.Write(ctx, entries)
		},
	)

	return s, mockStorage
}

func TestService_Log(t *testing.T) {
	t.Run("logs entry with generated ID", func(t *testing.T) {
		cfg := &config.LoggingConfig{}
		svc, mockStorage := createTestService(cfg)
		defer svc.Close()

		entry := &storage.LogEntry{
			Category: storage.LogCategorySystem,
			Level:    storage.LogLevelInfo,
			Message:  "test message",
		}

		svc.Log(context.Background(), entry)

		// Force flush
		err := svc.Flush(context.Background())
		require.NoError(t, err)

		entries := mockStorage.getEntries()
		require.Len(t, entries, 1)
		assert.NotEqual(t, uuid.Nil, entries[0].ID)
		assert.Equal(t, "test message", entries[0].Message)
	})

	t.Run("preserves existing ID", func(t *testing.T) {
		cfg := &config.LoggingConfig{}
		svc, mockStorage := createTestService(cfg)
		defer svc.Close()

		existingID := uuid.New()
		entry := &storage.LogEntry{
			ID:       existingID,
			Category: storage.LogCategorySystem,
			Level:    storage.LogLevelInfo,
			Message:  "test",
		}

		svc.Log(context.Background(), entry)
		svc.Flush(context.Background())

		entries := mockStorage.getEntries()
		require.Len(t, entries, 1)
		assert.Equal(t, existingID, entries[0].ID)
	})

	t.Run("sets timestamp if zero", func(t *testing.T) {
		cfg := &config.LoggingConfig{}
		svc, mockStorage := createTestService(cfg)
		defer svc.Close()

		entry := &storage.LogEntry{
			Category: storage.LogCategorySystem,
			Level:    storage.LogLevelInfo,
			Message:  "test",
		}

		before := time.Now()
		svc.Log(context.Background(), entry)
		svc.Flush(context.Background())
		after := time.Now()

		entries := mockStorage.getEntries()
		require.Len(t, entries, 1)
		assert.True(t, entries[0].Timestamp.After(before.Add(-time.Second)))
		assert.True(t, entries[0].Timestamp.Before(after.Add(time.Second)))
	})

	t.Run("does not log when closed", func(t *testing.T) {
		cfg := &config.LoggingConfig{}
		svc, mockStorage := createTestService(cfg)

		svc.Close()

		entry := &storage.LogEntry{
			Category: storage.LogCategorySystem,
			Level:    storage.LogLevelInfo,
			Message:  "should not be logged",
		}

		svc.Log(context.Background(), entry)

		// Even with flush, nothing should be added after close
		entries := mockStorage.getEntries()
		assert.Empty(t, entries)
	})
}

func TestService_LogSystem(t *testing.T) {
	cfg := &config.LoggingConfig{}
	svc, mockStorage := createTestService(cfg)
	defer svc.Close()

	fields := map[string]any{
		"component":  "test-component",
		"request_id": "req-123",
		"trace_id":   "trace-456",
		"custom_key": "custom_value",
	}

	svc.LogSystem(context.Background(), storage.LogLevelInfo, "System log test", fields)
	svc.Flush(context.Background())

	entries := mockStorage.getEntries()
	require.Len(t, entries, 1)

	entry := entries[0]
	assert.Equal(t, storage.LogCategorySystem, entry.Category)
	assert.Equal(t, storage.LogLevelInfo, entry.Level)
	assert.Equal(t, "System log test", entry.Message)
	assert.Equal(t, "test-component", entry.Component)
	assert.Equal(t, "req-123", entry.RequestID)
	assert.Equal(t, "trace-456", entry.TraceID)
}

func TestService_LogHTTP(t *testing.T) {
	t.Run("logs HTTP request with info level for 2xx", func(t *testing.T) {
		cfg := &config.LoggingConfig{}
		svc, mockStorage := createTestService(cfg)
		defer svc.Close()

		fields := &storage.HTTPLogFields{
			Method:        "GET",
			Path:          "/api/users",
			Query:         "limit=10",
			StatusCode:    200,
			DurationMs:    50,
			UserAgent:     "TestClient/1.0",
			Referer:       "https://example.com",
			ResponseBytes: 1024,
			RequestBytes:  100,
		}

		svc.LogHTTP(context.Background(), fields, "req-123", "trace-456", "user-789", "192.168.1.1")
		svc.Flush(context.Background())

		entries := mockStorage.getEntries()
		require.Len(t, entries, 1)

		entry := entries[0]
		assert.Equal(t, storage.LogCategoryHTTP, entry.Category)
		assert.Equal(t, storage.LogLevelInfo, entry.Level)
		assert.Contains(t, entry.Message, "GET /api/users")
		assert.Contains(t, entry.Message, "200")
		assert.Equal(t, "req-123", entry.RequestID)
		assert.Equal(t, "user-789", entry.UserID)
		assert.Equal(t, "192.168.1.1", entry.IPAddress)
	})

	t.Run("logs HTTP request with warn level for 4xx", func(t *testing.T) {
		cfg := &config.LoggingConfig{}
		svc, mockStorage := createTestService(cfg)
		defer svc.Close()

		fields := &storage.HTTPLogFields{
			Method:     "POST",
			Path:       "/api/login",
			StatusCode: 401,
			DurationMs: 10,
		}

		svc.LogHTTP(context.Background(), fields, "", "", "", "")
		svc.Flush(context.Background())

		entries := mockStorage.getEntries()
		require.Len(t, entries, 1)
		assert.Equal(t, storage.LogLevelWarn, entries[0].Level)
	})

	t.Run("logs HTTP request with error level for 5xx", func(t *testing.T) {
		cfg := &config.LoggingConfig{}
		svc, mockStorage := createTestService(cfg)
		defer svc.Close()

		fields := &storage.HTTPLogFields{
			Method:     "GET",
			Path:       "/api/data",
			StatusCode: 500,
			DurationMs: 100,
		}

		svc.LogHTTP(context.Background(), fields, "", "", "", "")
		svc.Flush(context.Background())

		entries := mockStorage.getEntries()
		require.Len(t, entries, 1)
		assert.Equal(t, storage.LogLevelError, entries[0].Level)
	})
}

func TestService_LogSecurity(t *testing.T) {
	t.Run("logs successful security event", func(t *testing.T) {
		cfg := &config.LoggingConfig{}
		svc, mockStorage := createTestService(cfg)
		defer svc.Close()

		fields := &storage.SecurityLogFields{
			EventType: "login_success",
			Success:   true,
			Email:     "user@example.com",
			TargetID:  "target-123",
			Action:    "login",
			Details: map[string]any{
				"provider": "email",
			},
		}

		svc.LogSecurity(context.Background(), fields, "req-123", "user-456", "192.168.1.1")
		svc.Flush(context.Background())

		entries := mockStorage.getEntries()
		require.Len(t, entries, 1)

		entry := entries[0]
		assert.Equal(t, storage.LogCategorySecurity, entry.Category)
		assert.Equal(t, storage.LogLevelInfo, entry.Level)
		assert.Equal(t, "login_success", entry.Message)
		assert.Equal(t, "user@example.com", entry.Fields["email"])
	})

	t.Run("logs failed security event with warn level", func(t *testing.T) {
		cfg := &config.LoggingConfig{}
		svc, mockStorage := createTestService(cfg)
		defer svc.Close()

		fields := &storage.SecurityLogFields{
			EventType: "login_failed",
			Success:   false,
			Email:     "attacker@example.com",
		}

		svc.LogSecurity(context.Background(), fields, "", "", "")
		svc.Flush(context.Background())

		entries := mockStorage.getEntries()
		require.Len(t, entries, 1)
		assert.Equal(t, storage.LogLevelWarn, entries[0].Level)
	})
}

func TestService_LogExecution(t *testing.T) {
	cfg := &config.LoggingConfig{}
	svc, mockStorage := createTestService(cfg)
	defer svc.Close()

	fields := map[string]any{
		"request_id": "req-123",
		"user_id":    "user-456",
		"extra":      "value",
	}

	svc.LogExecution(context.Background(), "exec-001", "function", storage.LogLevelDebug, "Function started", fields)
	svc.Flush(context.Background())

	entries := mockStorage.getEntries()
	require.Len(t, entries, 1)

	entry := entries[0]
	assert.Equal(t, storage.LogCategoryExecution, entry.Category)
	assert.Equal(t, storage.LogLevelDebug, entry.Level)
	assert.Equal(t, "Function started", entry.Message)
	assert.Equal(t, "exec-001", entry.ExecutionID)
	assert.Equal(t, "function", entry.ExecutionType)
	assert.Equal(t, "req-123", entry.RequestID)
	assert.Equal(t, "user-456", entry.UserID)
	assert.Equal(t, 1, entry.LineNumber)
}

func TestService_LogAI(t *testing.T) {
	cfg := &config.LoggingConfig{}
	svc, mockStorage := createTestService(cfg)
	defer svc.Close()

	fields := map[string]any{
		"model":       "gpt-4",
		"tokens_in":   100,
		"tokens_out":  200,
		"duration_ms": 1500,
	}

	svc.LogAI(context.Background(), fields, "req-123", "user-456")
	svc.Flush(context.Background())

	entries := mockStorage.getEntries()
	require.Len(t, entries, 1)

	entry := entries[0]
	assert.Equal(t, storage.LogCategoryAI, entry.Category)
	assert.Equal(t, storage.LogLevelInfo, entry.Level)
	assert.Equal(t, "AI query", entry.Message)
	assert.Equal(t, "req-123", entry.RequestID)
	assert.Equal(t, "user-456", entry.UserID)
	assert.Equal(t, "gpt-4", entry.Fields["model"])
}

func TestService_LogCustom(t *testing.T) {
	t.Run("logs custom category when allowed", func(t *testing.T) {
		cfg := &config.LoggingConfig{
			CustomCategories: []string{"metrics", "audit", "analytics"},
		}
		svc, mockStorage := createTestService(cfg)
		defer svc.Close()

		fields := map[string]any{
			"component":  "metrics",
			"request_id": "req-123",
			"user_id":    "user-456",
		}

		svc.LogCustom(context.Background(), "metrics", storage.LogLevelInfo, "Custom metric log", fields)
		svc.Flush(context.Background())

		entries := mockStorage.getEntries()
		require.Len(t, entries, 1)

		entry := entries[0]
		assert.Equal(t, storage.LogCategoryCustom, entry.Category)
		assert.Equal(t, "metrics", entry.CustomCategory)
		assert.Equal(t, "Custom metric log", entry.Message)
	})

	t.Run("logs as unknown when category not in allowed list", func(t *testing.T) {
		cfg := &config.LoggingConfig{
			CustomCategories: []string{"metrics", "audit"},
		}
		svc, mockStorage := createTestService(cfg)
		defer svc.Close()

		svc.LogCustom(context.Background(), "invalid", storage.LogLevelInfo, "Test", nil)
		svc.Flush(context.Background())

		entries := mockStorage.getEntries()
		require.Len(t, entries, 1)
		assert.Equal(t, "unknown", entries[0].CustomCategory)
	})

	t.Run("allows any category when no restrictions configured", func(t *testing.T) {
		cfg := &config.LoggingConfig{
			CustomCategories: []string{}, // No restrictions
		}
		svc, mockStorage := createTestService(cfg)
		defer svc.Close()

		svc.LogCustom(context.Background(), "any-category", storage.LogLevelInfo, "Test", nil)
		svc.Flush(context.Background())

		entries := mockStorage.getEntries()
		require.Len(t, entries, 1)
		assert.Equal(t, "any-category", entries[0].CustomCategory)
	})
}

func TestService_IsValidCustomCategory(t *testing.T) {
	t.Run("returns true when no restrictions", func(t *testing.T) {
		cfg := &config.LoggingConfig{
			CustomCategories: []string{},
		}
		svc, _ := createTestService(cfg)
		defer svc.Close()

		assert.True(t, svc.IsValidCustomCategory("anything"))
		assert.True(t, svc.IsValidCustomCategory("random"))
	})

	t.Run("returns true for allowed categories", func(t *testing.T) {
		cfg := &config.LoggingConfig{
			CustomCategories: []string{"metrics", "audit"},
		}
		svc, _ := createTestService(cfg)
		defer svc.Close()

		assert.True(t, svc.IsValidCustomCategory("metrics"))
		assert.True(t, svc.IsValidCustomCategory("audit"))
	})

	t.Run("returns false for disallowed categories", func(t *testing.T) {
		cfg := &config.LoggingConfig{
			CustomCategories: []string{"metrics", "audit"},
		}
		svc, _ := createTestService(cfg)
		defer svc.Close()

		assert.False(t, svc.IsValidCustomCategory("invalid"))
		assert.False(t, svc.IsValidCustomCategory("other"))
	})
}

func TestService_GetCustomCategories(t *testing.T) {
	cfg := &config.LoggingConfig{
		CustomCategories: []string{"metrics", "audit", "analytics"},
	}
	svc, _ := createTestService(cfg)
	defer svc.Close()

	categories := svc.GetCustomCategories()
	assert.Len(t, categories, 3)
	assert.Contains(t, categories, "metrics")
	assert.Contains(t, categories, "audit")
	assert.Contains(t, categories, "analytics")
}

func TestService_LineNumbers(t *testing.T) {
	t.Run("increments line numbers per execution", func(t *testing.T) {
		cfg := &config.LoggingConfig{}
		svc, mockStorage := createTestService(cfg)
		defer svc.Close()

		// Log multiple entries for same execution
		for i := 0; i < 5; i++ {
			entry := &storage.LogEntry{
				Category:    storage.LogCategoryExecution,
				ExecutionID: "exec-001",
				Message:     "Log line",
			}
			svc.Log(context.Background(), entry)
		}

		svc.Flush(context.Background())

		entries := mockStorage.getEntries()
		assert.Len(t, entries, 5)
		for i, entry := range entries {
			assert.Equal(t, i+1, entry.LineNumber)
		}
	})

	t.Run("separate line numbers per execution", func(t *testing.T) {
		cfg := &config.LoggingConfig{}
		svc, mockStorage := createTestService(cfg)
		defer svc.Close()

		// Log for exec-001
		svc.Log(context.Background(), &storage.LogEntry{
			Category:    storage.LogCategoryExecution,
			ExecutionID: "exec-001",
			Message:     "Line 1",
		})
		svc.Log(context.Background(), &storage.LogEntry{
			Category:    storage.LogCategoryExecution,
			ExecutionID: "exec-001",
			Message:     "Line 2",
		})

		// Log for exec-002
		svc.Log(context.Background(), &storage.LogEntry{
			Category:    storage.LogCategoryExecution,
			ExecutionID: "exec-002",
			Message:     "Line 1",
		})

		svc.Flush(context.Background())

		entries := mockStorage.getEntries()
		assert.Len(t, entries, 3)

		// exec-001 should have lines 1, 2
		assert.Equal(t, 1, entries[0].LineNumber)
		assert.Equal(t, 2, entries[1].LineNumber)
		// exec-002 should have line 1
		assert.Equal(t, 1, entries[2].LineNumber)
	})

	t.Run("clears line numbers", func(t *testing.T) {
		cfg := &config.LoggingConfig{}
		svc, mockStorage := createTestService(cfg)
		defer svc.Close()

		// Log some entries
		svc.Log(context.Background(), &storage.LogEntry{
			Category:    storage.LogCategoryExecution,
			ExecutionID: "exec-001",
			Message:     "Line 1",
		})
		svc.Log(context.Background(), &storage.LogEntry{
			Category:    storage.LogCategoryExecution,
			ExecutionID: "exec-001",
			Message:     "Line 2",
		})

		// Clear line numbers
		svc.ClearLineNumbers("exec-001")

		// New entries should start from 1
		svc.Log(context.Background(), &storage.LogEntry{
			Category:    storage.LogCategoryExecution,
			ExecutionID: "exec-001",
			Message:     "New Line 1",
		})

		svc.Flush(context.Background())

		entries := mockStorage.getEntries()
		assert.Len(t, entries, 3)
		assert.Equal(t, 1, entries[2].LineNumber) // Should restart from 1
	})
}

func TestService_GetRetentionPolicy(t *testing.T) {
	cfg := &config.LoggingConfig{
		SystemRetentionDays:    30,
		HTTPRetentionDays:      7,
		SecurityRetentionDays:  90,
		ExecutionRetentionDays: 14,
		AIRetentionDays:        60,
		CustomRetentionDays:    45,
	}
	svc, _ := createTestService(cfg)
	defer svc.Close()

	tests := []struct {
		category storage.LogCategory
		expected int
	}{
		{storage.LogCategorySystem, 30},
		{storage.LogCategoryHTTP, 7},
		{storage.LogCategorySecurity, 90},
		{storage.LogCategoryExecution, 14},
		{storage.LogCategoryAI, 60},
		{storage.LogCategoryCustom, 45},
		{"unknown", 30}, // Default
	}

	for _, tc := range tests {
		t.Run(string(tc.category), func(t *testing.T) {
			result := svc.GetRetentionPolicy(tc.category)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestService_Query(t *testing.T) {
	cfg := &config.LoggingConfig{}
	svc, mockStorage := createTestService(cfg)
	defer svc.Close()

	// Set up mock result
	mockStorage.queryResult = &storage.LogQueryResult{
		Entries: []*storage.LogEntry{
			{Message: "Entry 1"},
			{Message: "Entry 2"},
		},
		TotalCount: 2,
		HasMore:    false,
	}

	result, err := svc.Query(context.Background(), storage.LogQueryOptions{
		Category: storage.LogCategorySystem,
		Limit:    10,
	})

	require.NoError(t, err)
	assert.Len(t, result.Entries, 2)
	assert.Equal(t, int64(2), result.TotalCount)
	assert.False(t, result.HasMore)
}

func TestService_Stats(t *testing.T) {
	cfg := &config.LoggingConfig{}
	svc, mockStorage := createTestService(cfg)
	defer svc.Close()

	// Set up mock stats
	mockStorage.stats = &storage.LogStats{
		TotalEntries: 1000,
		EntriesByCategory: map[storage.LogCategory]int64{
			storage.LogCategorySystem: 500,
			storage.LogCategoryHTTP:   500,
		},
		EntriesByLevel: map[storage.LogLevel]int64{
			storage.LogLevelInfo:  800,
			storage.LogLevelError: 200,
		},
	}

	stats, err := svc.Stats(context.Background())

	require.NoError(t, err)
	assert.Equal(t, int64(1000), stats.TotalEntries)
	assert.Equal(t, int64(500), stats.EntriesByCategory[storage.LogCategorySystem])
}

func TestService_Close(t *testing.T) {
	cfg := &config.LoggingConfig{}
	svc, mockStorage := createTestService(cfg)

	// Log some entries
	svc.Log(context.Background(), &storage.LogEntry{
		Category: storage.LogCategorySystem,
		Message:  "Before close",
	})

	err := svc.Close()
	require.NoError(t, err)

	// Storage should be closed
	assert.True(t, mockStorage.closed)

	// Service should be marked closed
	assert.True(t, svc.closed)
}

func TestService_Writer(t *testing.T) {
	cfg := &config.LoggingConfig{
		ConsoleEnabled: true,
		ConsoleFormat:  "json",
	}
	svc, _ := createTestService(cfg)
	defer svc.Close()

	// Create writer manually since we're using mocks
	svc.writer = NewWriter(svc, true, "json")

	writer := svc.Writer()
	assert.NotNil(t, writer)
}

func TestService_Storage(t *testing.T) {
	cfg := &config.LoggingConfig{}
	svc, mockStorage := createTestService(cfg)
	defer svc.Close()

	storage := svc.Storage()
	assert.Same(t, mockStorage, storage)
}
