package webhook

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Webhook Service Delivery Tests
// =============================================================================

func TestWebhookService_Deliver(t *testing.T) {
	t.Run("deliver webhook with AllowPrivateIPs", func(t *testing.T) {
		service := &WebhookService{
			client:          &http.Client{Timeout: 5 * time.Second},
			allowPrivateIPs: true, // Bypass SSRF protection for testing
		}

		// Create test server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "POST", r.Method)
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
			assert.Equal(t, "Fluxbase-Webhooks/1.0", r.Header.Get("User-Agent"))

			var payload WebhookPayload
			err := json.NewDecoder(r.Body).Decode(&payload)
			require.NoError(t, err)

			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		webhook := &Webhook{
			ID:             uuid.New(),
			Name:           "test-webhook",
			URL:            server.URL,
			MaxRetries:     3,
			TimeoutSeconds: 30,
			Headers: map[string]string{
				"X-Custom": "test-value",
			},
			Events: []EventConfig{
				{Table: "users", Operations: []string{"INSERT"}},
			},
		}

		payload := &WebhookPayload{
			Event:     "INSERT",
			Table:     "users",
			Schema:    "public",
			Record:    json.RawMessage(`{"id":1,"name":"test"}`),
			Timestamp: time.Now(),
		}

		err := service.Deliver(context.Background(), webhook, payload)
		assert.NoError(t, err)
	})

	t.Run("deliver webhook with secret signature", func(t *testing.T) {
		secret := "test-secret-key"
		service := &WebhookService{
			client:          &http.Client{Timeout: 5 * time.Second},
			allowPrivateIPs: true,
		}

		receivedSignature := ""
		receivedLegacySig := ""

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			receivedSignature = r.Header.Get("X-Fluxbase-Signature")
			receivedLegacySig = r.Header.Get("X-Webhook-Signature")
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		webhook := &Webhook{
			ID:             uuid.New(),
			Name:           "signed-webhook",
			URL:            server.URL,
			Secret:         &secret,
			MaxRetries:     3,
			TimeoutSeconds: 30,
		}

		payload := &WebhookPayload{
			Event:     "INSERT",
			Table:     "users",
			Schema:    "public",
			Record:    json.RawMessage(`{"id":1}`),
			Timestamp: time.Now(),
		}

		err := service.Deliver(context.Background(), webhook, payload)
		assert.NoError(t, err)

		// Verify signature headers were sent
		assert.NotEmpty(t, receivedSignature, "X-Fluxbase-Signature should be set")
		assert.Contains(t, receivedSignature, "t=", "Signature should contain timestamp")
		assert.Contains(t, receivedSignature, "v1=", "Signature should contain v1")
		assert.NotEmpty(t, receivedLegacySig, "X-Webhook-Signature should be set")
	})

	t.Run("deliver webhook with timeout", func(t *testing.T) {
		service := &WebhookService{
			client:          &http.Client{Timeout: 5 * time.Second},
			allowPrivateIPs: true,
		}

		// Create server that delays response
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(100 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		webhook := &Webhook{
			ID:             uuid.New(),
			Name:           "timeout-webhook",
			URL:            server.URL,
			TimeoutSeconds: 1, // 1 second timeout (more than 100ms delay)
		}

		payload := &WebhookPayload{
			Event:  "INSERT",
			Table:  "users",
			Schema: "public",
			Record: json.RawMessage(`{"id":1}`),
		}

		err := service.Deliver(context.Background(), webhook, payload)
		// Should succeed since delay (100ms) < timeout (1000ms)
		assert.NoError(t, err)
	})

	t.Run("deliver webhook handles error response", func(t *testing.T) {
		service := &WebhookService{
			client:          &http.Client{Timeout: 5 * time.Second},
			allowPrivateIPs: true,
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("Internal Server Error"))
		}))
		defer server.Close()

		webhook := &Webhook{
			ID:             uuid.New(),
			Name:           "error-webhook",
			URL:            server.URL,
			MaxRetries:     3,
			TimeoutSeconds: 30,
		}

		payload := &WebhookPayload{
			Event:  "INSERT",
			Table:  "users",
			Schema: "public",
			Record: json.RawMessage(`{"id":1}`),
		}

		err := service.Deliver(context.Background(), webhook, payload)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "HTTP 500")
	})

	t.Run("deliver webhook with custom headers", func(t *testing.T) {
		service := &WebhookService{
			client:          &http.Client{Timeout: 5 * time.Second},
			allowPrivateIPs: true,
		}

		receivedHeaders := make(map[string]string)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for key := range r.Header {
				receivedHeaders[key] = r.Header.Get(key)
			}
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		webhook := &Webhook{
			ID:             uuid.New(),
			Name:           "headers-webhook",
			URL:            server.URL,
			MaxRetries:     3,
			TimeoutSeconds: 30,
			Headers: map[string]string{
				"X-API-Key":      "secret-key",
				"X-Request-ID":   "test-123",
				"Authorization":  "Bearer token",
				"X-Custom-Value": "custom",
			},
		}

		payload := &WebhookPayload{
			Event:  "INSERT",
			Table:  "users",
			Schema: "public",
			Record: json.RawMessage(`{"id":1}`),
		}

		err := service.Deliver(context.Background(), webhook, payload)
		assert.NoError(t, err)

		assert.Equal(t, "secret-key", receivedHeaders["X-Api-Key"])
		assert.Equal(t, "test-123", receivedHeaders["X-Request-Id"])
		assert.Equal(t, "Bearer token", receivedHeaders["Authorization"])
		assert.Equal(t, "custom", receivedHeaders["X-Custom-Value"])
	})
}

// =============================================================================
// Webhook Service Context Tests
// =============================================================================

func TestWebhookService_ContextCancellation(t *testing.T) {
	t.Run("context cancellation stops delivery", func(t *testing.T) {
		service := &WebhookService{
			client:          &http.Client{Timeout: 5 * time.Second},
			allowPrivateIPs: true,
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(2 * time.Second)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		webhook := &Webhook{
			ID:             uuid.New(),
			Name:           "cancel-webhook",
			URL:            server.URL,
			TimeoutSeconds: 30,
		}

		payload := &WebhookPayload{
			Event:  "INSERT",
			Table:  "users",
			Schema: "public",
			Record: json.RawMessage(`{"id":1}`),
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		err := service.Deliver(ctx, webhook, payload)
		assert.Error(t, err)
	})
}

// =============================================================================
// Webhook Service Error Handling Tests
// =============================================================================

func TestWebhookService_ErrorHandling(t *testing.T) {
	t.Run("invalid URL is rejected", func(t *testing.T) {
		service := &WebhookService{
			client:          &http.Client{Timeout: 5 * time.Second},
			allowPrivateIPs: false, // Enforce SSRF protection
		}

		webhook := &Webhook{
			ID:             uuid.New(),
			Name:           "invalid-url-webhook",
			URL:            "http://localhost:8080/webhook", // localhost is rejected
			MaxRetries:     3,
			TimeoutSeconds: 30,
		}

		payload := &WebhookPayload{
			Event:  "INSERT",
			Table:  "users",
			Schema: "public",
			Record: json.RawMessage(`{"id":1}`),
		}

		err := service.Deliver(context.Background(), webhook, payload)
		assert.Error(t, err)
		assert.True(t, strings.Contains(err.Error(), "webhook URL validation") || strings.Contains(err.Error(), "localhost"))
	})

	t.Run("connection error is propagated", func(t *testing.T) {
		service := &WebhookService{
			client:          &http.Client{Timeout: 100 * time.Millisecond},
			allowPrivateIPs: true,
		}

		webhook := &Webhook{
			ID:             uuid.New(),
			Name:           "conn-error-webhook",
			URL:            "http://localhost:9999/nonexistent", // Non-existent server
			MaxRetries:     3,
			TimeoutSeconds: 1,
		}

		payload := &WebhookPayload{
			Event:  "INSERT",
			Table:  "users",
			Schema: "public",
			Record: json.RawMessage(`{"id":1}`),
		}

		err := service.Deliver(context.Background(), webhook, payload)
		assert.Error(t, err)
	})
}

// =============================================================================
// Webhook Payload Tests
// =============================================================================

func TestWebhookService_PayloadHandling(t *testing.T) {
	t.Run("payload with old record", func(t *testing.T) {
		service := &WebhookService{
			client:          &http.Client{Timeout: 5 * time.Second},
			allowPrivateIPs: true,
		}

		var receivedPayload WebhookPayload

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			err := json.NewDecoder(r.Body).Decode(&receivedPayload)
			require.NoError(t, err)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		webhook := &Webhook{
			ID:             uuid.New(),
			Name:           "update-webhook",
			URL:            server.URL,
			MaxRetries:     3,
			TimeoutSeconds: 30,
		}

		payload := &WebhookPayload{
			Event:     "UPDATE",
			Table:     "users",
			Schema:    "public",
			Record:    json.RawMessage(`{"id":1,"name":"updated"}`),
			OldRecord: json.RawMessage(`{"id":1,"name":"original"}`),
			Timestamp: time.Now(),
		}

		err := service.Deliver(context.Background(), webhook, payload)
		assert.NoError(t, err)

		assert.Equal(t, "UPDATE", receivedPayload.Event)
		assert.NotNil(t, receivedPayload.OldRecord)
		assert.Contains(t, string(receivedPayload.OldRecord), "original")
	})

	t.Run("payload with complex JSON", func(t *testing.T) {
		service := &WebhookService{
			client:          &http.Client{Timeout: 5 * time.Second},
			allowPrivateIPs: true,
		}

		var receivedPayload WebhookPayload

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			err := json.NewDecoder(r.Body).Decode(&receivedPayload)
			require.NoError(t, err)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		webhook := &Webhook{
			ID:             uuid.New(),
			Name:           "complex-webhook",
			URL:            server.URL,
			MaxRetries:     3,
			TimeoutSeconds: 30,
		}

		complexRecord := map[string]interface{}{
			"id":      1,
			"name":    "test",
			"nested":  map[string]string{"key": "value"},
			"array":   []int{1, 2, 3},
			"boolean": true,
			"null":    nil,
			"number":  42.5,
		}
		recordJSON, _ := json.Marshal(complexRecord)

		payload := &WebhookPayload{
			Event:     "INSERT",
			Table:     "users",
			Schema:    "public",
			Record:    json.RawMessage(recordJSON),
			Timestamp: time.Now(),
		}

		err := service.Deliver(context.Background(), webhook, payload)
		assert.NoError(t, err)

		var decodedRecord map[string]interface{}
		err = json.Unmarshal(receivedPayload.Record, &decodedRecord)
		require.NoError(t, err)
		assert.Equal(t, float64(1), decodedRecord["id"])
		assert.Equal(t, "test", decodedRecord["name"])
	})
}

// =============================================================================
// Webhook Service Retry Logic Tests
// =============================================================================

func TestWebhookService_RetryLogic(t *testing.T) {
	t.Run("max retries configuration", func(t *testing.T) {
		testCases := []struct {
			name       string
			maxRetries int
			expected   int
		}{
			{"no retries", 0, 0},
			{"one retry", 1, 1},
			{"three retries", 3, 3},
			{"five retries", 5, 5},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				webhook := &Webhook{
					ID:             uuid.New(),
					Name:           "retry-test",
					URL:            "http://example.com",
					MaxRetries:     tc.maxRetries,
					TimeoutSeconds: 30,
				}

				assert.Equal(t, tc.expected, webhook.MaxRetries)
			})
		}
	})

	t.Run("retry backoff configuration", func(t *testing.T) {
		testCases := []struct {
			name             string
			retryBackoff     int
			expectedDuration time.Duration
		}{
			{"no backoff", 0, 0},
			{"1 second", 1, time.Second},
			{"30 seconds", 30, 30 * time.Second},
			{"60 seconds", 60, time.Minute},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				webhook := &Webhook{
					ID:                  uuid.New(),
					Name:                "backoff-test",
					URL:                 "http://example.com",
					RetryBackoffSeconds: tc.retryBackoff,
					TimeoutSeconds:      30,
				}

				assert.Equal(t, tc.expectedDuration, time.Duration(webhook.RetryBackoffSeconds)*time.Second)
			})
		}
	})
}

// =============================================================================
// Webhook Service SSRF Protection Tests
// =============================================================================

func TestWebhookService_SSRFProtection(t *testing.T) {
	t.Run("SSRF protection blocks private IPs", func(t *testing.T) {
		service := &WebhookService{
			client:          &http.Client{Timeout: 5 * time.Second},
			allowPrivateIPs: false, // Enforce SSRF protection
		}

		// Use a test server that simulates being on a private IP
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		webhook := &Webhook{
			ID:             uuid.New(),
			Name:           "ssrf-test",
			URL:            server.URL, // httptest.NewServer uses 127.0.0.1
			MaxRetries:     3,
			TimeoutSeconds: 30,
		}

		payload := &WebhookPayload{
			Event:  "INSERT",
			Table:  "users",
			Schema: "public",
			Record: json.RawMessage(`{"id":1}`),
		}

		err := service.Deliver(context.Background(), webhook, payload)
		// Should fail because 127.0.0.1 is a private IP
		assert.Error(t, err)
		// Error should be about URL validation or DNS rebinding
		assert.True(t, strings.Contains(err.Error(), "webhook URL validation") || strings.Contains(err.Error(), "DNS rebinding"))
	})

	t.Run("SSRF protection allows public IPs when AllowPrivateIPs is true", func(t *testing.T) {
		service := &WebhookService{
			client:          &http.Client{Timeout: 5 * time.Second},
			allowPrivateIPs: true, // Bypass for testing
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		webhook := &Webhook{
			ID:             uuid.New(),
			Name:           "ssrf-bypass-test",
			URL:            server.URL,
			MaxRetries:     3,
			TimeoutSeconds: 30,
		}

		payload := &WebhookPayload{
			Event:  "INSERT",
			Table:  "users",
			Schema: "public",
			Record: json.RawMessage(`{"id":1}`),
		}

		err := service.Deliver(context.Background(), webhook, payload)
		assert.NoError(t, err)
	})
}

// =============================================================================
// Webhook Event Config Tests
// =============================================================================

func TestWebhookService_EventConfig(t *testing.T) {
	t.Run("wildcard table", func(t *testing.T) {
		webhook := &Webhook{
			ID:             uuid.New(),
			Name:           "wildcard-webhook",
			URL:            "http://example.com",
			MaxRetries:     3,
			TimeoutSeconds: 30,
			Events: []EventConfig{
				{Table: "*", Operations: []string{"INSERT", "UPDATE", "DELETE"}},
			},
		}

		assert.Len(t, webhook.Events, 1)
		assert.Equal(t, "*", webhook.Events[0].Table)
		assert.Len(t, webhook.Events[0].Operations, 3)
	})

	t.Run("multiple event configs", func(t *testing.T) {
		webhook := &Webhook{
			ID:             uuid.New(),
			Name:           "multi-event-webhook",
			URL:            "http://example.com",
			MaxRetries:     3,
			TimeoutSeconds: 30,
			Events: []EventConfig{
				{Table: "users", Operations: []string{"INSERT", "UPDATE"}},
				{Table: "products", Operations: []string{"INSERT", "UPDATE", "DELETE"}},
				{Table: "orders", Operations: []string{"DELETE"}},
			},
		}

		assert.Len(t, webhook.Events, 3)

		usersOps := webhook.Events[0].Operations
		assert.Len(t, usersOps, 2)

		productsOps := webhook.Events[1].Operations
		assert.Len(t, productsOps, 3)

		ordersOps := webhook.Events[2].Operations
		assert.Len(t, ordersOps, 1)
	})
}

// =============================================================================
// Webhook Service Timestamp Tests
// =============================================================================

func TestWebhookService_Timestamps(t *testing.T) {
	t.Run("payload timestamp is set", func(t *testing.T) {
		service := &WebhookService{
			client:          &http.Client{Timeout: 5 * time.Second},
			allowPrivateIPs: true,
		}

		var receivedPayload WebhookPayload

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			err := json.NewDecoder(r.Body).Decode(&receivedPayload)
			require.NoError(t, err)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		webhook := &Webhook{
			ID:             uuid.New(),
			Name:           "timestamp-webhook",
			URL:            server.URL,
			MaxRetries:     3,
			TimeoutSeconds: 30,
		}

		now := time.Now()
		payload := &WebhookPayload{
			Event:     "INSERT",
			Table:     "users",
			Schema:    "public",
			Record:    json.RawMessage(`{"id":1}`),
			Timestamp: now,
		}

		err := service.Deliver(context.Background(), webhook, payload)
		assert.NoError(t, err)

		// Verify timestamp was sent (with some tolerance for network delay)
		assert.False(t, receivedPayload.Timestamp.IsZero())
		assert.WithinDuration(t, now, receivedPayload.Timestamp, time.Second)
	})
}

// =============================================================================
// Benchmark Tests
// =============================================================================

func BenchmarkWebhookService_Deliver(b *testing.B) {
	service := &WebhookService{
		client:          &http.Client{Timeout: 5 * time.Second},
		allowPrivateIPs: true,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload WebhookPayload
		_ = json.NewDecoder(r.Body).Decode(&payload)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	webhook := &Webhook{
		ID:             uuid.New(),
		Name:           "bench-webhook",
		URL:            server.URL,
		MaxRetries:     3,
		TimeoutSeconds: 30,
	}

	payload := &WebhookPayload{
		Event:     "INSERT",
		Table:     "users",
		Schema:    "public",
		Record:    json.RawMessage(`{"id":1,"name":"test"}`),
		Timestamp: time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = service.Deliver(context.Background(), webhook, payload)
	}
}

func BenchmarkGenerateSignature(b *testing.B) {
	service := &WebhookService{}
	payload := []byte(`{"event":"INSERT","table":"users","record":{"id":1,"name":"test"}}`)
	secret := "test-secret-key"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.generateSignature(payload, secret)
	}
}

func BenchmarkGenerateTimestampedSignature(b *testing.B) {
	payload := []byte(`{"event":"INSERT","table":"users","record":{"id":1}}`)
	secret := "test-secret"
	timestamp := time.Now().Unix()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		generateTimestampedSignature(payload, secret, timestamp)
	}
}

func BenchmarkValidateWebhookHeaders(b *testing.B) {
	headers := map[string]string{
		"Authorization":   "Bearer token",
		"X-Custom-Header": "custom-value",
		"X-Request-ID":    "req-123",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = validateWebhookHeaders(headers)
	}
}
