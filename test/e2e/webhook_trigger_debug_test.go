package e2e

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/require"
)

// TestWebhookTriggerDebug is a simplified test to debug webhook triggering
func TestWebhookTriggerDebug(t *testing.T) {
	tc := setupWebhookTriggerTest(t)
	defer tc.Close()

	// Create a test webhook server with mutex for thread-safe access
	var mu sync.Mutex
	var receivedPayload map[string]interface{}
	webhookServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Logf("Webhook received: %s %s", r.Method, r.URL.Path)
		t.Logf("Headers: %v", r.Header)

		var payload map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&payload)
		if err != nil {
			t.Logf("Error decoding payload: %v", err)
		} else {
			payloadJSON, _ := json.MarshalIndent(payload, "", "  ")
			t.Logf("Payload: %s", string(payloadJSON))

			mu.Lock()
			receivedPayload = payload
			mu.Unlock()
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer webhookServer.Close()

	t.Logf("Webhook server URL: %s", webhookServer.URL)

	// Sign up user
	email := "debug@example.com"
	password := "debugpass123"
	authResp := tc.NewRequest("POST", "/api/v1/auth/signup").
		WithBody(map[string]interface{}{
			"email":    email,
			"password": password,
		}).
		Send()

	t.Logf("Signup status: %d", authResp.Status())
	require.Equal(t, fiber.StatusCreated, authResp.Status(), "Signup should succeed")

	var authResult map[string]interface{}
	authResp.JSON(&authResult)
	token := authResult["access_token"].(string)
	t.Logf("Got auth token: %s...", token[:20])

	// Create webhook
	webhookPayload := map[string]interface{}{
		"name": "Debug Webhook",
		"url":  webhookServer.URL,
		"events": []map[string]interface{}{
			{
				"table":      "users",
				"operations": []string{"INSERT"},
			},
		},
		"secret":  "debug-secret",
		"enabled": true,
	}

	webhookJSON, _ := json.MarshalIndent(webhookPayload, "", "  ")
	t.Logf("Creating webhook with payload: %s", string(webhookJSON))

	createResp := tc.NewRequest("POST", "/api/v1/webhooks").
		WithAuth(token).
		WithBody(webhookPayload).
		Send()

	t.Logf("Webhook creation status: %d", createResp.Status())
	t.Logf("Webhook creation response: %s", string(createResp.Body()))

	if createResp.Status() != fiber.StatusCreated {
		t.Fatalf("Failed to create webhook: status=%d, body=%s", createResp.Status(), string(createResp.Body()))
	}

	var webhook map[string]interface{}
	createResp.JSON(&webhook)
	webhookID := webhook["id"].(string)
	t.Logf("Created webhook ID: %s", webhookID)

	// Verify webhook in database
	results := tc.QuerySQL("SELECT id, name, url, enabled FROM auth.webhooks WHERE id = $1", webhookID)
	if len(results) > 0 {
		t.Logf("Webhook in DB: %+v", results[0])
	} else {
		t.Fatal("Webhook not found in database!")
	}

	// Check if webhook trigger exists on users table
	triggerResults := tc.QuerySQL(`
		SELECT tgname
		FROM pg_trigger
		WHERE tgrelid = 'auth.users'::regclass
		AND tgname LIKE '%webhook%'
	`)
	t.Logf("Webhook triggers on auth.users: %+v", triggerResults)

	// Create a new user to trigger the webhook
	newEmail := "trigger@example.com"
	t.Logf("Creating new user: %s", newEmail)

	signupResp := tc.NewRequest("POST", "/api/v1/auth/signup").
		WithBody(map[string]interface{}{
			"email":    newEmail,
			"password": "triggerpass123",
		}).
		Send()

	t.Logf("New user signup status: %d", signupResp.Status())
	require.Equal(t, fiber.StatusCreated, signupResp.Status(), "New user signup should succeed")

	// Wait for webhook event to be created
	success := tc.WaitForCondition(3*time.Second, 100*time.Millisecond, func() bool {
		results := tc.QuerySQL("SELECT COUNT(*) FROM auth.webhook_events")
		return len(results) > 0 && results[0]["count"].(int64) > 0
	})
	require.True(t, success, "Webhook event should be created within 3 seconds")

	eventResults := tc.QuerySQL("SELECT id, webhook_id, event_type, table_name, processed FROM auth.webhook_events ORDER BY created_at DESC LIMIT 5")
	t.Logf("Recent webhook events: %+v", eventResults)

	// Wait for webhook delivery (thread-safe check)
	t.Log("Waiting for webhook delivery...")
	success = tc.WaitForCondition(5*time.Second, 200*time.Millisecond, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return receivedPayload != nil
	})
	if !success {
		t.Log("Webhook delivery timed out after 5 seconds")
	}

	// Check if webhook was delivered (thread-safe access)
	mu.Lock()
	payloadCopy := receivedPayload
	mu.Unlock()

	if payloadCopy != nil {
		t.Log("✓ Webhook was delivered successfully!")
		payloadJSON, _ := json.MarshalIndent(payloadCopy, "", "  ")
		t.Logf("Final payload: %s", string(payloadJSON))
	} else {
		t.Error("✗ Webhook was NOT delivered")

		// Check webhook_events for errors
		errorResults := tc.QuerySQL(`
			SELECT id, webhook_id, event_type, processed, attempts, error_message
			FROM auth.webhook_events
			WHERE webhook_id = $1
			ORDER BY created_at DESC
		`, webhookID)
		t.Logf("Webhook events for this webhook: %+v", errorResults)
	}
}
