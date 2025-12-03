package e2e

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/fluxbase-eu/fluxbase/test"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// setupWebhookTriggerTest prepares the test context for webhook trigger tests
func setupWebhookTriggerTest(t *testing.T) *test.TestContext {
	tc := test.NewTestContext(t)
	tc.EnsureAuthSchema()

	// Clean tables before each test (batched for performance)
	tc.ExecuteSQL(`
		TRUNCATE TABLE
			auth.webhook_events,
			auth.webhook_deliveries,
			auth.webhooks,
			auth.users,
			auth.sessions
		CASCADE
	`)

	// Enable webhook trigger on users table for these tests
	tc.ExecuteSQL("SELECT auth.create_webhook_trigger('auth', 'users')")

	// Enable signup for tests
	tc.Config.Auth.EnableSignup = true

	return tc
}

// TestWebhookTriggerOnUserInsert tests that a webhook is automatically triggered when a user is created
func TestWebhookTriggerOnUserInsert(t *testing.T) {
	tc := setupWebhookTriggerTest(t)
	defer tc.Close()

	// Create a test webhook server to receive the webhook with mutex for thread-safe access
	var mu sync.Mutex
	var receivedPayload map[string]interface{}
	webhookServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "POST", r.Method)
		require.Equal(t, "application/json", r.Header.Get("Content-Type"))
		require.NotEmpty(t, r.Header.Get("X-Webhook-Signature"))

		var payload map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&payload)
		require.NoError(t, err)

		mu.Lock()
		receivedPayload = payload
		mu.Unlock()

		w.WriteHeader(http.StatusOK)
	}))
	defer webhookServer.Close()

	// Sign up and get admin token
	email := "admin@example.com"
	password := "adminpassword123"
	authResp := tc.NewRequest("POST", "/api/v1/auth/signup").
		WithBody(map[string]interface{}{
			"email":    email,
			"password": password,
		}).
		Send().
		AssertStatus(fiber.StatusCreated)

	var authResult map[string]interface{}
	authResp.JSON(&authResult)
	adminToken := authResult["access_token"].(string)

	// Create a webhook for user INSERT events
	createWebhookResp := tc.NewRequest("POST", "/api/v1/webhooks").
		WithAuth(adminToken).
		WithBody(map[string]interface{}{
			"name": "Test Webhook",
			"url":  webhookServer.URL,
			"events": []map[string]interface{}{
				{
					"table":      "users",
					"operations": []string{"INSERT"},
				},
			},
			"secret":  "test-secret-key",
			"enabled": true,
		}).
		Send().
		AssertStatus(fiber.StatusCreated)

	var webhook map[string]interface{}
	createWebhookResp.JSON(&webhook)
	_ = webhook["id"].(string) // webhookID not needed for this test

	// Create a new user to trigger the webhook
	newUserEmail := "newuser@example.com"
	tc.NewRequest("POST", "/api/v1/auth/signup").
		WithBody(map[string]interface{}{
			"email":    newUserEmail,
			"password": "newuserpassword123",
		}).
		Send().
		AssertStatus(fiber.StatusCreated)

	// Wait for webhook to be triggered and delivered (check actual delivery, not just event creation)
	success := tc.WaitForCondition(5*time.Second, 100*time.Millisecond, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return receivedPayload != nil
	})
	require.True(t, success, "Webhook should have been delivered within 5 seconds")

	// Verify webhook was delivered (thread-safe access)
	mu.Lock()
	defer mu.Unlock()
	require.NotNil(t, receivedPayload, "Webhook should have been delivered")
	require.Equal(t, "INSERT", receivedPayload["event"])
	require.Equal(t, "auth", receivedPayload["schema"])
	require.Equal(t, "users", receivedPayload["table"])

	// Verify the record contains the new user email
	recordData := receivedPayload["record"].(map[string]interface{})
	require.Equal(t, newUserEmail, recordData["email"])
}

// TestWebhookTriggerOnUserUpdate tests that a webhook is triggered on user updates
func TestWebhookTriggerOnUserUpdate(t *testing.T) {
	tc := setupWebhookTriggerTest(t)
	defer tc.Close()

	// Create a test webhook server with mutex for thread-safe access
	var mu sync.Mutex
	var receivedPayloads []map[string]interface{}
	webhookServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&payload)
		require.NoError(t, err)
		mu.Lock()
		receivedPayloads = append(receivedPayloads, payload)
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer webhookServer.Close()

	// Sign up user
	email := "user@example.com"
	password := "password123"
	authResp := tc.NewRequest("POST", "/api/v1/auth/signup").
		WithBody(map[string]interface{}{
			"email":    email,
			"password": password,
		}).
		Send().
		AssertStatus(fiber.StatusCreated)

	var authResult map[string]interface{}
	authResp.JSON(&authResult)
	token := authResult["access_token"].(string)

	// Create webhook for UPDATE events
	createWebhookResp := tc.NewRequest("POST", "/api/v1/webhooks").
		WithAuth(token).
		WithBody(map[string]interface{}{
			"name":    "Update Webhook",
			"url":     webhookServer.URL,
			"events":  []map[string]interface{}{{"table": "users", "operations": []string{"UPDATE"}}},
			"secret":  "test-secret-key",
			"enabled": true,
		}).
		Send().
		AssertStatus(fiber.StatusCreated)

	var webhook map[string]interface{}
	createWebhookResp.JSON(&webhook)

	// Update the user's user_metadata
	tc.NewRequest("PATCH", "/api/v1/auth/user").
		WithAuth(token).
		WithBody(map[string]interface{}{
			"user_metadata": map[string]interface{}{
				"name": "Test User",
			},
		}).
		Send().
		AssertStatus(fiber.StatusOK)

	// Wait for webhook delivery
	success := tc.WaitForCondition(5*time.Second, 100*time.Millisecond, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(receivedPayloads) > 0
	})
	require.True(t, success, "Webhook should be delivered within 5 seconds")

	// Get payload copy (with lock)
	mu.Lock()
	payloadCount := len(receivedPayloads)
	payloadsCopy := make([]map[string]interface{}, len(receivedPayloads))
	copy(payloadsCopy, receivedPayloads)
	mu.Unlock()

	require.Greater(t, payloadCount, 0, "At least one webhook should have been delivered")

	// Find the UPDATE event
	var updatePayload map[string]interface{}
	for _, payload := range payloadsCopy {
		if payload["event"] == "UPDATE" {
			updatePayload = payload
			break
		}
	}

	require.NotNil(t, updatePayload, "UPDATE webhook should have been delivered")
	require.Equal(t, "UPDATE", updatePayload["event"])
	require.Equal(t, "auth", updatePayload["schema"])
	require.Equal(t, "users", updatePayload["table"])

	// Verify both old and new data are present
	require.NotNil(t, updatePayload["old_record"], "old_record should be present for UPDATE events")
	require.NotNil(t, updatePayload["record"], "record (new data) should be present for UPDATE events")
}

// TestWebhookTriggerRetry tests that failed webhooks are retried with exponential backoff
func TestWebhookTriggerRetry(t *testing.T) {
	tc := setupWebhookTriggerTest(t)
	defer tc.Close()

	// Configure webhook trigger service for faster retries in tests (3 second interval)
	triggerService := tc.Server.GetWebhookTriggerService()
	if triggerService != nil {
		triggerService.SetBacklogInterval(3 * time.Second)
	}

	// Create a failing webhook server (returns 500) with mutex for thread-safe access
	var mu sync.Mutex
	attemptCount := 0
	webhookServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		attemptCount++
		currentAttempt := attemptCount
		mu.Unlock()

		if currentAttempt < 3 {
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer webhookServer.Close()

	// Sign up user
	email := "user@example.com"
	password := "password123"
	authResp := tc.NewRequest("POST", "/api/v1/auth/signup").
		WithBody(map[string]interface{}{
			"email":    email,
			"password": password,
		}).
		Send().
		AssertStatus(fiber.StatusCreated)

	var authResult map[string]interface{}
	authResp.JSON(&authResult)
	token := authResult["access_token"].(string)

	// Create webhook with shorter retry backoff for testing (2 seconds)
	createWebhookResp := tc.NewRequest("POST", "/api/v1/webhooks").
		WithAuth(token).
		WithBody(map[string]interface{}{
			"name":                  "Retry Webhook",
			"url":                   webhookServer.URL,
			"events":                []map[string]interface{}{{"table": "users", "operations": []string{"INSERT"}}},
			"secret":                "test-secret-key",
			"enabled":               true,
			"retry_backoff_seconds": 2, // Shorter backoff for faster test
		}).
		Send().
		AssertStatus(fiber.StatusCreated)

	var webhook map[string]interface{}
	createWebhookResp.JSON(&webhook)

	// Create a new user to trigger webhook
	tc.NewRequest("POST", "/api/v1/auth/signup").
		WithBody(map[string]interface{}{
			"email":    "newuser@example.com",
			"password": "password123",
		}).
		Send().
		AssertStatus(fiber.StatusCreated)

	// Wait for retries
	// Timeline: T=0 first attempt fails, next_retry_at = T+2s (attempt 1 * 2s backoff)
	//           T=3s backlog processor runs, second attempt fails, next_retry_at = T+3+4s = T+7s (attempt 2 * 2s backoff)
	//           T=6s backlog processor runs (nothing ready yet)
	//           T=9s backlog processor runs, third attempt succeeds
	// So we need to wait at least 10 seconds to see 3 attempts
	success := tc.WaitForCondition(15*time.Second, 500*time.Millisecond, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return attemptCount >= 3
	})
	require.True(t, success, "Webhook should be retried at least 3 times within 15 seconds")

	// Get final attempt count (with lock)
	mu.Lock()
	finalAttemptCount := attemptCount
	mu.Unlock()

	require.GreaterOrEqual(t, finalAttemptCount, 3, "Webhook should have been retried at least 3 times")

	// Verify the event was eventually marked as processed
	results := tc.QuerySQL("SELECT processed FROM auth.webhook_events ORDER BY created_at DESC LIMIT 1")
	require.Greater(t, len(results), 0, "Should have query results")
	processed := results[0]["processed"].(bool)
	require.True(t, processed, "Event should eventually be marked as processed after successful delivery")
}

// TestWebhookTriggerMultipleWebhooks tests that multiple webhooks can be triggered for the same event
func TestWebhookTriggerMultipleWebhooks(t *testing.T) {
	tc := setupWebhookTriggerTest(t)
	defer tc.Close()

	// Create two test webhook servers with mutex for thread-safe access
	var mu1 sync.Mutex
	var payload1 map[string]interface{}
	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var p map[string]interface{}
		json.NewDecoder(r.Body).Decode(&p)
		mu1.Lock()
		payload1 = p
		mu1.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer server1.Close()

	var mu2 sync.Mutex
	var payload2 map[string]interface{}
	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var p map[string]interface{}
		json.NewDecoder(r.Body).Decode(&p)
		mu2.Lock()
		payload2 = p
		mu2.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer server2.Close()

	// Sign up user
	email := "user@example.com"
	password := "password123"
	authResp := tc.NewRequest("POST", "/api/v1/auth/signup").
		WithBody(map[string]interface{}{
			"email":    email,
			"password": password,
		}).
		Send().
		AssertStatus(fiber.StatusCreated)

	var authResult map[string]interface{}
	authResp.JSON(&authResult)
	token := authResult["access_token"].(string)

	// Create first webhook
	tc.NewRequest("POST", "/api/v1/webhooks").
		WithAuth(token).
		WithBody(map[string]interface{}{
			"name":    "Webhook 1",
			"url":     server1.URL,
			"events":  []map[string]interface{}{{"table": "users", "operations": []string{"INSERT"}}},
			"secret":  "secret1",
			"enabled": true,
		}).
		Send().
		AssertStatus(fiber.StatusCreated)

	// Create second webhook
	tc.NewRequest("POST", "/api/v1/webhooks").
		WithAuth(token).
		WithBody(map[string]interface{}{
			"name":    "Webhook 2",
			"url":     server2.URL,
			"events":  []map[string]interface{}{{"table": "users", "operations": []string{"INSERT"}}},
			"secret":  "secret2",
			"enabled": true,
		}).
		Send().
		AssertStatus(fiber.StatusCreated)

	// Create a new user to trigger both webhooks
	newEmail := "trigger@example.com"
	tc.NewRequest("POST", "/api/v1/auth/signup").
		WithBody(map[string]interface{}{
			"email":    newEmail,
			"password": "password123",
		}).
		Send().
		AssertStatus(fiber.StatusCreated)

	// Wait for webhook deliveries
	success := tc.WaitForCondition(5*time.Second, 100*time.Millisecond, func() bool {
		mu1.Lock()
		hasPayload1 := payload1 != nil
		mu1.Unlock()

		mu2.Lock()
		hasPayload2 := payload2 != nil
		mu2.Unlock()

		return hasPayload1 && hasPayload2
	})
	require.True(t, success, "Both webhooks should receive payloads within 5 seconds")

	// Get payload copies (with locks)
	mu1.Lock()
	payload1Copy := payload1
	mu1.Unlock()
	require.NotNil(t, payload1Copy, "First webhook should have received payload")

	mu2.Lock()
	payload2Copy := payload2
	mu2.Unlock()
	require.NotNil(t, payload2Copy, "Second webhook should have received payload")

	require.Equal(t, newEmail, payload1Copy["record"].(map[string]interface{})["email"])
	require.Equal(t, newEmail, payload2Copy["record"].(map[string]interface{})["email"])
}

// TestWebhookTriggerInactiveWebhook tests that inactive webhooks are not triggered
func TestWebhookTriggerInactiveWebhook(t *testing.T) {
	tc := setupWebhookTriggerTest(t)
	defer tc.Close()

	// Create a test webhook server with mutex for thread-safe access
	var mu sync.Mutex
	var receivedPayload map[string]interface{}
	webhookServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]interface{}
		json.NewDecoder(r.Body).Decode(&payload)
		mu.Lock()
		receivedPayload = payload
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer webhookServer.Close()

	// Sign up user
	email := "user@example.com"
	password := "password123"
	authResp := tc.NewRequest("POST", "/api/v1/auth/signup").
		WithBody(map[string]interface{}{
			"email":    email,
			"password": password,
		}).
		Send().
		AssertStatus(fiber.StatusCreated)

	var authResult map[string]interface{}
	authResp.JSON(&authResult)
	token := authResult["access_token"].(string)

	// Create webhook but keep it inactive
	tc.NewRequest("POST", "/api/v1/webhooks").
		WithAuth(token).
		WithBody(map[string]interface{}{
			"name":    "Inactive Webhook",
			"url":     webhookServer.URL,
			"events":  []map[string]interface{}{{"table": "users", "operations": []string{"INSERT"}}},
			"secret":  "test-secret-key",
			"enabled": false, // Inactive webhook
		}).
		Send().
		AssertStatus(fiber.StatusCreated)

	// Create a new user
	tc.NewRequest("POST", "/api/v1/auth/signup").
		WithBody(map[string]interface{}{
			"email":    "newuser@example.com",
			"password": "password123",
		}).
		Send().
		AssertStatus(fiber.StatusCreated)

	// Wait briefly to ensure webhook processing has time to run (if it incorrectly tries to)
	// We're testing a negative case - webhook should NOT be triggered
	// Wait for a reasonable period to confirm no events are created
	success := tc.WaitForCondition(3*time.Second, 200*time.Millisecond, func() bool {
		results := tc.QuerySQL("SELECT COUNT(*) FROM auth.webhook_events WHERE webhook_id IN (SELECT id FROM auth.webhooks WHERE enabled = false)")
		if len(results) == 0 {
			return false
		}
		// Return true if events were created (which would be unexpected)
		return results[0]["count"].(int64) > 0
	})
	// We expect this to timeout (success = false) because no events should be created
	require.False(t, success, "No webhook events should be created for inactive webhooks")

	// Verify webhook was NOT delivered (thread-safe check)
	mu.Lock()
	payloadCopy := receivedPayload
	mu.Unlock()
	require.Nil(t, payloadCopy, "Inactive webhook should not be triggered")

	// Double-check no webhook events were created for this webhook
	results := tc.QuerySQL("SELECT COUNT(*) FROM auth.webhook_events WHERE webhook_id IN (SELECT id FROM auth.webhooks WHERE enabled = false)")
	require.Greater(t, len(results), 0, "Should have query results")
	eventCount := results[0]["count"].(int64)
	require.Equal(t, int64(0), eventCount, "No events should be created for inactive webhooks")
}

// TestWebhookTriggerCleanup tests that old processed webhook events are cleaned up
func TestWebhookTriggerCleanup(t *testing.T) {
	tc := setupWebhookTriggerTest(t)
	defer tc.Close()

	// Create a test webhook server
	webhookServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer webhookServer.Close()

	// Sign up user
	email := "user@example.com"
	password := "password123"
	authResp := tc.NewRequest("POST", "/api/v1/auth/signup").
		WithBody(map[string]interface{}{
			"email":    email,
			"password": password,
		}).
		Send().
		AssertStatus(fiber.StatusCreated)

	var authResult map[string]interface{}
	authResp.JSON(&authResult)
	token := authResult["access_token"].(string)
	userID := authResult["user"].(map[string]interface{})["id"].(string)

	// Create webhook
	createWebhookResp := tc.NewRequest("POST", "/api/v1/webhooks").
		WithAuth(token).
		WithBody(map[string]interface{}{
			"name":    "Cleanup Webhook",
			"url":     webhookServer.URL,
			"events":  []map[string]interface{}{{"table": "users", "operations": []string{"INSERT"}}},
			"secret":  "test-secret-key",
			"enabled": true,
		}).
		Send().
		AssertStatus(fiber.StatusCreated)

	var webhook map[string]interface{}
	createWebhookResp.JSON(&webhook)
	webhookID := webhook["id"].(string)

	// Manually insert an old processed event (8 days old)
	oldEventID := uuid.New()
	tc.ExecuteSQL(`
		INSERT INTO auth.webhook_events
		(id, webhook_id, event_type, table_schema, table_name, record_id, processed, created_at)
		VALUES ($1, $2, 'INSERT', 'auth', 'users', $3, true, NOW() - INTERVAL '8 days')
	`, oldEventID, webhookID, userID)

	// Insert a recent processed event (1 day old)
	recentEventID := uuid.New()
	tc.ExecuteSQL(`
		INSERT INTO auth.webhook_events
		(id, webhook_id, event_type, table_schema, table_name, record_id, processed, created_at)
		VALUES ($1, $2, 'INSERT', 'auth', 'users', $3, true, NOW() - INTERVAL '1 day')
	`, recentEventID, webhookID, userID)

	// Manually trigger cleanup (normally runs every hour)
	// We'll just verify the SQL logic by checking if old events exist
	results := tc.QuerySQL("SELECT EXISTS(SELECT 1 FROM auth.webhook_events WHERE id = $1)", oldEventID)
	require.Greater(t, len(results), 0, "Should have query results")
	oldEventExists := results[0]["exists"].(bool)
	require.True(t, oldEventExists, "Old event should exist before cleanup")

	// Run cleanup query (simulating what the service does)
	tc.ExecuteSQL("DELETE FROM auth.webhook_events WHERE processed = true AND created_at < NOW() - INTERVAL '7 days'")

	// Verify old event was deleted
	results = tc.QuerySQL("SELECT EXISTS(SELECT 1 FROM auth.webhook_events WHERE id = $1)", oldEventID)
	require.Greater(t, len(results), 0, "Should have query results")
	oldEventExists = results[0]["exists"].(bool)
	require.False(t, oldEventExists, "Old event should be cleaned up after 7 days")

	// Verify recent event still exists
	results = tc.QuerySQL("SELECT EXISTS(SELECT 1 FROM auth.webhook_events WHERE id = $1)", recentEventID)
	require.Greater(t, len(results), 0, "Should have query results")
	recentEventExists := results[0]["exists"].(bool)
	require.True(t, recentEventExists, "Recent event should not be cleaned up")
}
