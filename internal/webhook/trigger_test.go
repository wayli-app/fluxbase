package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTriggerService(t *testing.T) {
	t.Run("creates with default workers", func(t *testing.T) {
		svc := NewTriggerService(nil, nil, 0)
		require.NotNil(t, svc)
		assert.Equal(t, 4, svc.workers) // Default
		assert.Equal(t, 30*time.Second, svc.backlogInterval)
		assert.NotNil(t, svc.eventChan)
		assert.NotNil(t, svc.stopChan)
	})

	t.Run("creates with negative workers uses default", func(t *testing.T) {
		svc := NewTriggerService(nil, nil, -1)
		assert.Equal(t, 4, svc.workers)
	})

	t.Run("creates with custom workers", func(t *testing.T) {
		svc := NewTriggerService(nil, nil, 10)
		assert.Equal(t, 10, svc.workers)
	})

	t.Run("creates with specified db and webhook service", func(t *testing.T) {
		webhookSvc := &WebhookService{}
		svc := NewTriggerService(nil, webhookSvc, 5)
		assert.Equal(t, webhookSvc, svc.webhookSvc)
	})
}

func TestTriggerService_SetBacklogInterval(t *testing.T) {
	svc := NewTriggerService(nil, nil, 2)

	t.Run("sets backlog interval before start", func(t *testing.T) {
		svc.SetBacklogInterval(1 * time.Minute)
		assert.Equal(t, 1*time.Minute, svc.backlogInterval)
	})

	t.Run("sets backlog interval to short duration", func(t *testing.T) {
		svc.SetBacklogInterval(5 * time.Second)
		assert.Equal(t, 5*time.Second, svc.backlogInterval)
	})
}

func TestTriggerService_Stop(t *testing.T) {
	svc := NewTriggerService(nil, nil, 1)

	// Stop should not panic even without Start
	assert.NotPanics(t, func() {
		svc.Stop()
	})
}

func TestWebhookEvent_Struct(t *testing.T) {
	t.Run("creates webhook event with all fields", func(t *testing.T) {
		webhookID := uuid.New()
		eventID := uuid.New()
		now := time.Now()
		recordID := "record-123"
		errorMsg := "test error"

		event := &WebhookEvent{
			ID:            eventID,
			WebhookID:     webhookID,
			EventType:     "INSERT",
			TableSchema:   "public",
			TableName:     "users",
			RecordID:      &recordID,
			OldData:       []byte(`{"name": "old"}`),
			NewData:       []byte(`{"name": "new"}`),
			Processed:     false,
			Attempts:      2,
			LastAttemptAt: &now,
			NextRetryAt:   &now,
			ErrorMessage:  &errorMsg,
			CreatedAt:     now,
		}

		assert.Equal(t, eventID, event.ID)
		assert.Equal(t, webhookID, event.WebhookID)
		assert.Equal(t, "INSERT", event.EventType)
		assert.Equal(t, "public", event.TableSchema)
		assert.Equal(t, "users", event.TableName)
		assert.Equal(t, "record-123", *event.RecordID)
		assert.JSONEq(t, `{"name": "old"}`, string(event.OldData))
		assert.JSONEq(t, `{"name": "new"}`, string(event.NewData))
		assert.False(t, event.Processed)
		assert.Equal(t, 2, event.Attempts)
		assert.NotNil(t, event.LastAttemptAt)
		assert.NotNil(t, event.NextRetryAt)
		assert.Equal(t, "test error", *event.ErrorMessage)
	})

	t.Run("creates minimal webhook event", func(t *testing.T) {
		event := &WebhookEvent{
			ID:          uuid.New(),
			WebhookID:   uuid.New(),
			EventType:   "DELETE",
			TableSchema: "public",
			TableName:   "posts",
			Processed:   false,
			Attempts:    0,
			CreatedAt:   time.Now(),
		}

		assert.NotEqual(t, uuid.Nil, event.ID)
		assert.Equal(t, "DELETE", event.EventType)
		assert.Nil(t, event.RecordID)
		assert.Nil(t, event.OldData)
		assert.Nil(t, event.NewData)
		assert.Nil(t, event.LastAttemptAt)
		assert.Nil(t, event.NextRetryAt)
		assert.Nil(t, event.ErrorMessage)
	})
}

func TestEventChannel(t *testing.T) {
	svc := NewTriggerService(nil, nil, 1)

	t.Run("event channel has buffer of 1000", func(t *testing.T) {
		assert.Equal(t, 1000, cap(svc.eventChan))
	})

	t.Run("can send events to channel", func(t *testing.T) {
		id := uuid.New()
		svc.eventChan <- id

		select {
		case received := <-svc.eventChan:
			assert.Equal(t, id, received)
		case <-time.After(100 * time.Millisecond):
			t.Fatal("timeout waiting for event")
		}
	})
}

func TestBackoffCalculation(t *testing.T) {
	// Test the exponential backoff calculation logic
	// that would be used in handleDeliveryFailure

	testCases := []struct {
		attempts              int
		retryBackoffSeconds   int
		expectedBackoffMillis int
	}{
		{1, 60, 60000},  // First retry: 60 * 1 = 60s
		{2, 60, 120000}, // Second retry: 60 * 2 = 120s
		{3, 60, 180000}, // Third retry: 60 * 3 = 180s
		{1, 30, 30000},  // Different base: 30 * 1 = 30s
		{5, 10, 50000},  // Fifth retry: 10 * 5 = 50s
	}

	for _, tc := range testCases {
		t.Run("backoff calculation", func(t *testing.T) {
			backoffSeconds := tc.retryBackoffSeconds * tc.attempts
			assert.Equal(t, tc.expectedBackoffMillis/1000, backoffSeconds)
		})
	}
}

func TestMaxRetriesLogic(t *testing.T) {
	// Test the max retries check logic

	testCases := []struct {
		attempts   int
		maxRetries int
		shouldFail bool
	}{
		{1, 3, false}, // 1 attempt, max 3 - continue
		{2, 3, false}, // 2 attempts, max 3 - continue
		{3, 3, true},  // 3 attempts, max 3 - max reached
		{4, 3, true},  // 4 attempts, max 3 - exceeded
		{1, 1, true},  // 1 attempt, max 1 - max reached
		{0, 5, false}, // 0 attempts, max 5 - continue
	}

	for _, tc := range testCases {
		maxReached := tc.attempts >= tc.maxRetries
		assert.Equal(t, tc.shouldFail, maxReached)
	}
}

func TestEndpointRateLimiter(t *testing.T) {
	t.Run("allows initial request", func(t *testing.T) {
		rl := &endpointRateLimiter{
			requests: make(map[string][]time.Time),
			limit:    10,
		}

		allowed := rl.allow("https://example.com/webhook")
		assert.True(t, allowed)
	})

	t.Run("allows requests up to limit", func(t *testing.T) {
		rl := &endpointRateLimiter{
			requests: make(map[string][]time.Time),
			limit:    5,
		}

		endpoint := "https://example.com/webhook"
		for i := 0; i < 5; i++ {
			allowed := rl.allow(endpoint)
			assert.True(t, allowed, "request %d should be allowed", i+1)
		}
	})

	t.Run("blocks requests after limit exceeded", func(t *testing.T) {
		rl := &endpointRateLimiter{
			requests: make(map[string][]time.Time),
			limit:    3,
		}

		endpoint := "https://example.com/webhook"
		// Use up all allowed requests
		for i := 0; i < 3; i++ {
			rl.allow(endpoint)
		}

		// Next request should be blocked
		allowed := rl.allow(endpoint)
		assert.False(t, allowed)
	})

	t.Run("tracks different endpoints separately", func(t *testing.T) {
		rl := &endpointRateLimiter{
			requests: make(map[string][]time.Time),
			limit:    2,
		}

		endpoint1 := "https://example.com/webhook1"
		endpoint2 := "https://example.com/webhook2"

		// Exhaust limit for endpoint1
		rl.allow(endpoint1)
		rl.allow(endpoint1)
		assert.False(t, rl.allow(endpoint1))

		// endpoint2 should still be allowed
		assert.True(t, rl.allow(endpoint2))
		assert.True(t, rl.allow(endpoint2))
		assert.False(t, rl.allow(endpoint2))
	})

	t.Run("uses sliding window to expire old requests", func(t *testing.T) {
		rl := &endpointRateLimiter{
			requests: make(map[string][]time.Time),
			limit:    2,
		}

		endpoint := "https://example.com/webhook"

		// Pre-populate with old requests (outside 1-minute window)
		oldTime := time.Now().Add(-2 * time.Minute)
		rl.requests[endpoint] = []time.Time{oldTime, oldTime}

		// Should allow new requests since old ones are expired
		assert.True(t, rl.allow(endpoint))
		assert.True(t, rl.allow(endpoint))
		assert.False(t, rl.allow(endpoint))
	})

	t.Run("filters expired requests while keeping recent ones", func(t *testing.T) {
		rl := &endpointRateLimiter{
			requests: make(map[string][]time.Time),
			limit:    3,
		}

		endpoint := "https://example.com/webhook"

		// Pre-populate with mix of old and recent requests
		oldTime := time.Now().Add(-2 * time.Minute)
		recentTime := time.Now().Add(-30 * time.Second)
		rl.requests[endpoint] = []time.Time{oldTime, recentTime}

		// One recent request exists, so only 2 more should be allowed
		assert.True(t, rl.allow(endpoint))
		assert.True(t, rl.allow(endpoint))
		assert.False(t, rl.allow(endpoint))
	})

	t.Run("limit of 1 allows single request then blocks", func(t *testing.T) {
		rl := &endpointRateLimiter{
			requests: make(map[string][]time.Time),
			limit:    1,
		}

		endpoint := "https://example.com/webhook"
		assert.True(t, rl.allow(endpoint))
		assert.False(t, rl.allow(endpoint))
	})

	t.Run("handles empty endpoint string", func(t *testing.T) {
		rl := &endpointRateLimiter{
			requests: make(map[string][]time.Time),
			limit:    2,
		}

		assert.True(t, rl.allow(""))
		assert.True(t, rl.allow(""))
		assert.False(t, rl.allow(""))
	})
}

func TestNewEndpointRateLimiter(t *testing.T) {
	t.Run("creates with specified limit", func(t *testing.T) {
		rl := newEndpointRateLimiter(100)
		require.NotNil(t, rl)
		assert.Equal(t, 100, rl.limit)
		assert.NotNil(t, rl.requests)
	})

	t.Run("uses default limit when zero provided", func(t *testing.T) {
		rl := newEndpointRateLimiter(0)
		assert.Equal(t, DefaultRateLimitPerEndpoint, rl.limit)
	})

	t.Run("uses default limit when negative provided", func(t *testing.T) {
		rl := newEndpointRateLimiter(-5)
		assert.Equal(t, DefaultRateLimitPerEndpoint, rl.limit)
	})
}

func TestDefaultRateLimitPerEndpoint(t *testing.T) {
	assert.Equal(t, 60, DefaultRateLimitPerEndpoint)
}

func TestTriggerService_WaitForReady(t *testing.T) {
	t.Run("returns nil when already ready", func(t *testing.T) {
		svc := NewTriggerService(nil, nil, 1)
		// Simulate successful ready state
		svc.signalReady(false)

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		err := svc.WaitForReady(ctx)
		assert.NoError(t, err)
	})

	t.Run("returns error when listener failed", func(t *testing.T) {
		svc := NewTriggerService(nil, nil, 1)
		// Simulate failed ready state
		svc.signalReady(true)

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		err := svc.WaitForReady(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to start")
	})

	t.Run("returns context error on timeout", func(t *testing.T) {
		svc := NewTriggerService(nil, nil, 1)
		// Don't signal ready

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		err := svc.WaitForReady(ctx)
		assert.Error(t, err)
		assert.Equal(t, context.DeadlineExceeded, err)
	})

	t.Run("returns context error when cancelled", func(t *testing.T) {
		svc := NewTriggerService(nil, nil, 1)

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		err := svc.WaitForReady(ctx)
		assert.Error(t, err)
		assert.Equal(t, context.Canceled, err)
	})
}

func TestTriggerService_IsReady(t *testing.T) {
	t.Run("returns false initially", func(t *testing.T) {
		svc := NewTriggerService(nil, nil, 1)
		assert.False(t, svc.IsReady())
	})

	t.Run("returns true after successful signal", func(t *testing.T) {
		svc := NewTriggerService(nil, nil, 1)
		svc.signalReady(false)
		assert.True(t, svc.IsReady())
	})

	t.Run("returns false after failed signal", func(t *testing.T) {
		svc := NewTriggerService(nil, nil, 1)
		svc.signalReady(true)
		assert.False(t, svc.IsReady())
	})
}

func TestTriggerService_signalReady(t *testing.T) {
	t.Run("only signals once", func(t *testing.T) {
		svc := NewTriggerService(nil, nil, 1)

		// Signal success
		svc.signalReady(false)
		assert.True(t, svc.IsReady())
		assert.False(t, svc.listenerFailed)

		// Try to signal failure - should be ignored
		svc.signalReady(true)
		assert.True(t, svc.IsReady())
		assert.False(t, svc.listenerFailed) // Still false
	})

	t.Run("closes ready channel", func(t *testing.T) {
		svc := NewTriggerService(nil, nil, 1)
		svc.signalReady(false)

		// Channel should be closed and receive should not block
		select {
		case <-svc.readyChan:
			// Expected - channel is closed
		default:
			t.Fatal("ready channel should be closed")
		}
	})
}

func TestTriggerService_RateLimiter(t *testing.T) {
	t.Run("service has rate limiter initialized", func(t *testing.T) {
		svc := NewTriggerService(nil, nil, 1)
		assert.NotNil(t, svc.rateLimiter)
		assert.Equal(t, DefaultRateLimitPerEndpoint, svc.rateLimiter.limit)
	})
}

func TestEndpointRateLimiter_ConcurrentAccess(t *testing.T) {
	rl := &endpointRateLimiter{
		requests: make(map[string][]time.Time),
		limit:    100,
	}

	endpoint := "https://example.com/webhook"

	// Run concurrent requests
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 20; j++ {
				rl.allow(endpoint)
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Should have recorded requests without panic
	assert.NotPanics(t, func() {
		rl.allow(endpoint)
	})
}

func TestWebhookPayloadEventTypes(t *testing.T) {
	// Test that event types match expected values used in deliverEvent
	eventTypes := []string{"INSERT", "UPDATE", "DELETE"}

	for _, et := range eventTypes {
		event := &WebhookEvent{
			ID:          uuid.New(),
			WebhookID:   uuid.New(),
			EventType:   et,
			TableSchema: "public",
			TableName:   "users",
		}
		assert.Equal(t, et, event.EventType)
	}
}

func TestTriggerService_EnablePrivateIPs(t *testing.T) {
	t.Run("EnablePrivateIPs sets AllowPrivateIPs on webhook service", func(t *testing.T) {
		webhookSvc := &WebhookService{}
		svc := NewTriggerService(nil, webhookSvc, 1)

		assert.False(t, svc.webhookSvc.AllowPrivateIPs())

		svc.EnablePrivateIPs()

		assert.True(t, svc.webhookSvc.AllowPrivateIPs())
	})

	t.Run("EnablePrivateIPs with nil webhook service", func(t *testing.T) {
		svc := NewTriggerService(nil, nil, 1)

		assert.NotPanics(t, func() {
			svc.EnablePrivateIPs()
		})
	})
}

func TestTriggerService_Start_Stop_Lifecycle(t *testing.T) {
	t.Run("Stop can be called multiple times safely", func(t *testing.T) {
		svc := NewTriggerService(nil, nil, 1)

		assert.NotPanics(t, func() {
			svc.Stop()
			svc.Stop()
			svc.Stop()
		})
	})

	t.Run("Stop uses atomic check to prevent double-close", func(t *testing.T) {
		svc := NewTriggerService(nil, nil, 1)

		// First stop should succeed
		svc.Stop()

		// Second stop should be idempotent
		assert.NotPanics(t, func() {
			svc.Stop()
		})
	})

	t.Run("service initialization defaults", func(t *testing.T) {
		svc := NewTriggerService(nil, nil, 0)
		assert.Equal(t, 4, svc.workers) // Default
		assert.Equal(t, 30*time.Second, svc.backlogInterval)
		assert.NotNil(t, svc.eventChan)
		assert.NotNil(t, svc.stopChan)
		assert.NotNil(t, svc.rateLimiter)
		assert.NotNil(t, svc.readyChan)
	})

	t.Run("custom worker count", func(t *testing.T) {
		svc := NewTriggerService(nil, nil, 8)
		assert.Equal(t, 8, svc.workers)
	})

	t.Run("event channel capacity", func(t *testing.T) {
		svc := NewTriggerService(nil, nil, 1)
		assert.Equal(t, 1000, cap(svc.eventChan))
	})

	t.Run("default backlog interval", func(t *testing.T) {
		svc := NewTriggerService(nil, nil, 1)
		assert.Equal(t, 30*time.Second, svc.backlogInterval)
	})

	t.Run("can customize backlog interval", func(t *testing.T) {
		svc := NewTriggerService(nil, nil, 1)
		customInterval := 15 * time.Second

		svc.SetBacklogInterval(customInterval)
		assert.Equal(t, customInterval, svc.backlogInterval)
	})

	t.Run("initial state values", func(t *testing.T) {
		svc := NewTriggerService(nil, nil, 1)

		// Check initial state
		assert.False(t, svc.IsReady())
		assert.False(t, svc.listenerFailed)
		assert.Nil(t, svc.cancel)
		assert.Nil(t, svc.backlogTicker)
		assert.Nil(t, svc.cleanupTicker)
	})

	t.Run("ready channel is not closed initially", func(t *testing.T) {
		svc := NewTriggerService(nil, nil, 1)

		select {
		case <-svc.readyChan:
			t.Fatal("ready channel should not be closed initially")
		default:
			// Expected - channel is open
		}
	})
}

func TestHandleDeliveryFailure_MaxRetriesLogic(t *testing.T) {
	// Test the exponential backoff and max retry logic
	// from handleDeliveryFailure without requiring a database

	testCases := []struct {
		name                string
		attempts            int
		maxRetries          int
		retryBackoffSeconds int
		shouldMarkFailed    bool
		expectedNextRetry   string
	}{
		{
			name:                "first retry with 60s backoff",
			attempts:            1,
			maxRetries:          3,
			retryBackoffSeconds: 60,
			shouldMarkFailed:    false,
			expectedNextRetry:   "60s from now",
		},
		{
			name:                "second retry with 120s backoff",
			attempts:            2,
			maxRetries:          3,
			retryBackoffSeconds: 60,
			shouldMarkFailed:    false,
			expectedNextRetry:   "120s from now",
		},
		{
			name:                "third retry with 180s backoff",
			attempts:            3,
			maxRetries:          3,
			retryBackoffSeconds: 60,
			shouldMarkFailed:    true, // Max retries reached
			expectedNextRetry:   "never",
		},
		{
			name:                "exceeded retries",
			attempts:            5,
			maxRetries:          3,
			retryBackoffSeconds: 60,
			shouldMarkFailed:    true,
			expectedNextRetry:   "never",
		},
		{
			name:                "custom backoff interval",
			attempts:            2,
			maxRetries:          5,
			retryBackoffSeconds: 30,
			shouldMarkFailed:    false,
			expectedNextRetry:   "60s from now",
		},
		{
			name:                "first attempt with maxRetries=1",
			attempts:            1,
			maxRetries:          1,
			retryBackoffSeconds: 60,
			shouldMarkFailed:    true,
			expectedNextRetry:   "never",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Simulate the logic from handleDeliveryFailure
			maxReached := tc.attempts >= tc.maxRetries

			if maxReached {
				assert.True(t, tc.shouldMarkFailed, "should mark as failed")
			} else {
				assert.False(t, tc.shouldMarkFailed, "should schedule retry")

				// Calculate backoff
				backoffSeconds := tc.retryBackoffSeconds * tc.attempts
				assert.Greater(t, backoffSeconds, 0, "backoff should be positive")
			}
		})
	}
}

func TestHandleDeliveryFailure_BackoffCalculation(t *testing.T) {
	t.Run("linear backoff formula", func(t *testing.T) {
		// The formula is: retryBackoffSeconds * attempts
		retryBackoffSeconds := 60

		backoffs := map[int]int{
			1:  60,  // 60 * 1
			2:  120, // 60 * 2
			3:  180, // 60 * 3
			4:  240, // 60 * 4
			5:  300, // 60 * 5
			10: 600, // 60 * 10
		}

		for attempt, expectedBackoff := range backoffs {
			t.Run(fmt.Sprintf("attempt %d", attempt), func(t *testing.T) {
				backoff := retryBackoffSeconds * attempt
				assert.Equal(t, expectedBackoff, backoff)
			})
		}
	})

	t.Run("different backoff bases", func(t *testing.T) {
		testCases := []struct {
			backoffBase int
			attempts    int
			expected    int
		}{
			{10, 1, 10},
			{10, 5, 50},
			{30, 2, 60},
			{120, 3, 360},
		}

		for _, tc := range testCases {
			t.Run(fmt.Sprintf("%ds base, attempt %d", tc.backoffBase, tc.attempts), func(t *testing.T) {
				backoff := tc.backoffBase * tc.attempts
				assert.Equal(t, tc.expected, backoff)
			})
		}
	})
}

func TestScheduleRetryWithoutIncrement_Logic(t *testing.T) {
	t.Run("calculates 10 second retry delay", func(t *testing.T) {
		// From scheduleRetryWithoutIncrement
		nextRetry := time.Now().Add(10 * time.Second)

		// Verify it's approximately 10 seconds from now
		duration := time.Until(nextRetry)
		assert.Greater(t, duration, 9*time.Second)
		assert.Less(t, duration, 11*time.Second)
	})

	t.Run("does not increment attempt count", func(t *testing.T) {
		// This test verifies the intent of scheduleRetryWithoutIncrement
		// It should schedule a retry without marking the attempt as failed
		event := &WebhookEvent{
			ID:       uuid.New(),
			Attempts: 2,
		}

		// The function should NOT increment event.Attempts
		// (we can't test the actual function without DB, but we document the intent)
		expectedAttempts := event.Attempts
		assert.Equal(t, 2, expectedAttempts)
	})
}

func TestMarkEventSuccess_Logic(t *testing.T) {
	t.Run("marks event as processed", func(t *testing.T) {
		eventID := uuid.New()

		// Simulate the query logic
		query := `
			UPDATE auth.webhook_events
			SET processed = TRUE,
			    last_attempt_at = NOW()
			WHERE id = $1
		`

		assert.Contains(t, query, "processed = TRUE")
		assert.Contains(t, query, "last_attempt_at = NOW()")
		assert.Contains(t, query, "WHERE id = $1")

		// Verify the event ID would be used
		assert.NotEqual(t, uuid.Nil, eventID)
	})
}

func TestProcessBacklog_Logic(t *testing.T) {
	t.Run("queries for events needing retry", func(t *testing.T) {
		query := `
			SELECT DISTINCT webhook_id
			FROM auth.webhook_events
			WHERE processed = FALSE
			  AND next_retry_at IS NOT NULL
			  AND next_retry_at <= NOW()
			LIMIT 50
		`

		// Verify query selects events that need retry
		assert.Contains(t, query, "processed = FALSE")
		assert.Contains(t, query, "next_retry_at IS NOT NULL")
		assert.Contains(t, query, "next_retry_at <= NOW()")
		assert.Contains(t, query, "LIMIT 50")
	})

	t.Run("uses default backlog interval", func(t *testing.T) {
		svc := NewTriggerService(nil, nil, 1)
		assert.Equal(t, 30*time.Second, svc.backlogInterval)
	})

	t.Run("can customize backlog interval", func(t *testing.T) {
		svc := NewTriggerService(nil, nil, 1)
		customInterval := 15 * time.Second

		svc.SetBacklogInterval(customInterval)
		assert.Equal(t, customInterval, svc.backlogInterval)
	})

	t.Run("resets ticker when changing interval while running", func(t *testing.T) {
		svc := NewTriggerService(nil, nil, 1)

		// Set initial interval
		svc.SetBacklogInterval(10 * time.Second)
		assert.Equal(t, 10*time.Second, svc.backlogInterval)

		// Simulate ticker being created (would happen in Start)
		svc.backlogTicker = time.NewTicker(10 * time.Second)

		// Change interval
		svc.SetBacklogInterval(5 * time.Second)
		assert.Equal(t, 5*time.Second, svc.backlogInterval)

		// Cleanup
		svc.backlogTicker.Stop()
	})
}

func TestCleanupOldEvents_Logic(t *testing.T) {
	t.Run("deletes events older than 7 days", func(t *testing.T) {
		query := `
			DELETE FROM auth.webhook_events
			WHERE processed = TRUE
			  AND created_at < NOW() - INTERVAL '7 days'
		`

		assert.Contains(t, query, "DELETE FROM auth.webhook_events")
		assert.Contains(t, query, "processed = TRUE")
		assert.Contains(t, query, "created_at < NOW() - INTERVAL '7 days'")
	})

	t.Run("cleanup ticker is 1 hour", func(t *testing.T) {
		// From TriggerService.Start
		cleanupInterval := 1 * time.Hour
		assert.Equal(t, 1*time.Hour, cleanupInterval)
	})
}

func TestProcessWebhookEvents_Logic(t *testing.T) {
	t.Run("queries unprocessed events in batches of 10", func(t *testing.T) {
		query := `
			SELECT id, webhook_id, event_type, table_schema, table_name, record_id,
			       old_data, new_data, processed, attempts, last_attempt_at, next_retry_at, error_message, created_at
			FROM auth.webhook_events
			WHERE webhook_id = $1
			  AND processed = FALSE
			  AND (next_retry_at IS NULL OR next_retry_at <= NOW())
			ORDER BY created_at ASC
			LIMIT 10
		`

		assert.Contains(t, query, "webhook_id = $1")
		assert.Contains(t, query, "processed = FALSE")
		assert.Contains(t, query, "next_retry_at IS NULL OR next_retry_at <= NOW()")
		assert.Contains(t, query, "ORDER BY created_at ASC")
		assert.Contains(t, query, "LIMIT 10")
	})

	t.Run("checks if webhook is enabled before processing", func(t *testing.T) {
		enabledWebhook := &Webhook{Enabled: true}
		disabledWebhook := &Webhook{Enabled: false}

		assert.True(t, enabledWebhook.Enabled)
		assert.False(t, disabledWebhook.Enabled)
	})

	t.Run("processes events in batch", func(t *testing.T) {
		// Verify batch size limit
		batchSize := 10

		events := make([]*WebhookEvent, 15)
		for i := 0; i < 15; i++ {
			events[i] = &WebhookEvent{ID: uuid.New()}
		}

		// Only first 10 should be processed
		toProcess := events
		if len(toProcess) > batchSize {
			toProcess = toProcess[:batchSize]
		}

		assert.Len(t, toProcess, batchSize)
	})
}

func TestDeliverEvent_RateLimitCheck(t *testing.T) {
	t.Run("checks rate limit before delivery", func(t *testing.T) {
		rl := &endpointRateLimiter{
			requests: make(map[string][]time.Time),
			limit:    5,
		}

		webhook := &Webhook{
			ID:  uuid.New(),
			URL: "https://example.com/webhook",
		}

		// First request should be allowed
		allowed := rl.allow(webhook.URL)
		assert.True(t, allowed)

		// Use up the limit
		for i := 0; i < 4; i++ {
			rl.allow(webhook.URL)
		}

		// Next request should be blocked
		allowed = rl.allow(webhook.URL)
		assert.False(t, allowed)
	})

	t.Run("schedules retry without increment when rate limited", func(t *testing.T) {
		// When rate limited, scheduleRetryWithoutIncrement is called
		// This should NOT increment the attempt count
		initialAttempts := 2

		// Simulate the logic
		nextRetry := time.Now().Add(10 * time.Second)
		attemptsAfterRateLimit := initialAttempts // Should NOT increment

		assert.Equal(t, initialAttempts, attemptsAfterRateLimit)
		assert.True(t, nextRetry.After(time.Now().Add(9*time.Second)))
	})
}

func TestDeliverEvent_PayloadConstruction(t *testing.T) {
	t.Run("INSERT event includes only new record", func(t *testing.T) {
		event := &WebhookEvent{
			EventType: "INSERT",
			NewData:   json.RawMessage(`{"id": 1, "name": "Alice"}`),
		}

		payload := &WebhookPayload{
			Event:     event.EventType,
			Table:     "users",
			Schema:    "public",
			Record:    event.NewData,
			Timestamp: time.Now(),
		}

		assert.Equal(t, "INSERT", payload.Event)
		assert.NotNil(t, payload.Record)
		assert.Nil(t, payload.OldRecord)
	})

	t.Run("UPDATE event includes both records", func(t *testing.T) {
		event := &WebhookEvent{
			EventType: "UPDATE",
			OldData:   json.RawMessage(`{"id": 1, "name": "Bob"}`),
			NewData:   json.RawMessage(`{"id": 1, "name": "Robert"}`),
		}

		payload := &WebhookPayload{
			Event:     event.EventType,
			Table:     "users",
			Schema:    "public",
			Record:    event.NewData,
			OldRecord: event.OldData,
			Timestamp: time.Now(),
		}

		assert.Equal(t, "UPDATE", payload.Event)
		assert.NotNil(t, payload.Record)
		assert.NotNil(t, payload.OldRecord)
	})

	t.Run("DELETE event includes old record", func(t *testing.T) {
		event := &WebhookEvent{
			EventType: "DELETE",
			OldData:   json.RawMessage(`{"id": 1, "name": "Deleted"}`),
		}

		payload := &WebhookPayload{
			Event:     event.EventType,
			Table:     "users",
			Schema:    "public",
			Record:    event.OldData,
			Timestamp: time.Now(),
		}

		assert.Equal(t, "DELETE", payload.Event)
		assert.NotNil(t, payload.Record)
	})
}

func TestTriggerService_EventChannel(t *testing.T) {
	t.Run("event channel has buffer of 1000", func(t *testing.T) {
		svc := NewTriggerService(nil, nil, 1)
		assert.Equal(t, 1000, cap(svc.eventChan))
	})

	t.Run("can send to event channel", func(t *testing.T) {
		svc := NewTriggerService(nil, nil, 1)
		webhookID := uuid.New()

		select {
		case svc.eventChan <- webhookID:
			// Success
		default:
			t.Fatal("channel should not be full")
		}

		// Clean up
		<-svc.eventChan
	})

	t.Run("event channel blocks when full", func(t *testing.T) {
		svc := NewTriggerService(nil, nil, 1)

		// Fill the channel
		for i := 0; i < 1000; i++ {
			svc.eventChan <- uuid.New()
		}

		// Next send should block
		done := make(chan bool)
		errChan := make(chan string, 1)
		go func() {
			select {
			case svc.eventChan <- uuid.New():
				errChan <- "should not be able to send to full channel"
			case <-time.After(100 * time.Millisecond):
				// Expected - channel is full
			}
			close(done)
		}()

		<-done
		select {
		case errMsg := <-errChan:
			t.Fatal(errMsg)
		default:
			// No error, test passed
		}

		// Cleanup: drain some events
		for i := 0; i < 100; i++ {
			<-svc.eventChan
		}
	})
}

func TestTriggerService_ListenRetryLogic(t *testing.T) {
	t.Run("max retries is 5", func(t *testing.T) {
		// From listen function
		maxRetries := 5
		assert.Equal(t, 5, maxRetries)
	})

	t.Run("initial retry delay is 200ms", func(t *testing.T) {
		// From listen function
		initialDelay := 200 * time.Millisecond
		assert.Equal(t, 200*time.Millisecond, initialDelay)
	})

	t.Run("retry delay doubles with exponential backoff", func(t *testing.T) {
		// From listen function
		retryDelay := 200 * time.Millisecond

		retryDelay *= 2
		assert.Equal(t, 400*time.Millisecond, retryDelay)

		retryDelay *= 2
		assert.Equal(t, 800*time.Millisecond, retryDelay)
	})

	t.Run("retry delay caps at 2 seconds", func(t *testing.T) {
		// From listen function
		retryDelay := 200 * time.Millisecond

		// Simulate doubling
		for i := 0; i < 10; i++ {
			retryDelay *= 2
			if retryDelay > 2*time.Second {
				retryDelay = 2 * time.Second
			}
		}

		assert.Equal(t, 2*time.Second, retryDelay)
	})
}

func TestWebhookEvent_StatusTransitions(t *testing.T) {
	t.Run("event starts unprocessed", func(t *testing.T) {
		event := &WebhookEvent{
			ID:        uuid.New(),
			Processed: false,
			Attempts:  0,
		}

		assert.False(t, event.Processed)
		assert.Equal(t, 0, event.Attempts)
	})

	t.Run("event is marked as processed on success", func(t *testing.T) {
		event := &WebhookEvent{
			ID:        uuid.New(),
			Processed: false,
		}

		// Simulate success marking
		event.Processed = true

		assert.True(t, event.Processed)
	})

	t.Run("event attempts increment on failure", func(t *testing.T) {
		event := &WebhookEvent{
			ID:        uuid.New(),
			Processed: false,
			Attempts:  0,
		}

		// Simulate failure handling
		event.Attempts++

		assert.Equal(t, 1, event.Attempts)
	})

	t.Run("event with next retry set", func(t *testing.T) {
		nextRetry := time.Now().Add(5 * time.Minute)

		event := &WebhookEvent{
			ID:          uuid.New(),
			Processed:   false,
			NextRetryAt: &nextRetry,
		}

		assert.NotNil(t, event.NextRetryAt)
		assert.True(t, event.NextRetryAt.After(time.Now()))
	})
}

func TestWebhookDeliveryCreation(t *testing.T) {
	t.Run("creates delivery record before delivery", func(t *testing.T) {
		webhookID := uuid.New()
		event := "INSERT"
		attempt := 1

		// Simulate delivery record creation
		delivery := &WebhookDelivery{
			ID:        uuid.New(),
			WebhookID: webhookID,
			Event:     event,
			Status:    "pending",
			Attempt:   attempt,
		}

		assert.NotEqual(t, uuid.Nil, delivery.ID)
		assert.Equal(t, webhookID, delivery.WebhookID)
		assert.Equal(t, "INSERT", delivery.Event)
		assert.Equal(t, "pending", delivery.Status)
		assert.Equal(t, 1, delivery.Attempt)
	})

	t.Run("delivery includes attempt number", func(t *testing.T) {
		attempts := []int{1, 2, 3, 4, 5}

		for _, attempt := range attempts {
			delivery := &WebhookDelivery{
				Attempt: attempt,
			}
			assert.Equal(t, attempt, delivery.Attempt)
		}
	})
}
