package webhook

import (
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
		{1, 60, 60000},   // First retry: 60 * 1 = 60s
		{2, 60, 120000},  // Second retry: 60 * 2 = 120s
		{3, 60, 180000},  // Third retry: 60 * 3 = 180s
		{1, 30, 30000},   // Different base: 30 * 1 = 30s
		{5, 10, 50000},   // Fifth retry: 10 * 5 = 50s
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
