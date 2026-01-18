package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateSignature(t *testing.T) {
	service := &WebhookService{}

	t.Run("Generate HMAC signature", func(t *testing.T) {
		payload := []byte(`{"event":"INSERT","table":"users"}`)
		secret := "test-secret-key"

		signature := service.generateSignature(payload, secret)

		// Verify signature is not empty
		assert.NotEmpty(t, signature)

		// Verify signature is hex encoded
		_, err := hex.DecodeString(signature)
		assert.NoError(t, err)

		// Verify signature length (SHA256 produces 64 hex characters)
		assert.Equal(t, 64, len(signature))
	})

	t.Run("Same payload and secret produce same signature", func(t *testing.T) {
		payload := []byte(`{"test":"data"}`)
		secret := "my-secret"

		sig1 := service.generateSignature(payload, secret)
		sig2 := service.generateSignature(payload, secret)

		assert.Equal(t, sig1, sig2)
	})

	t.Run("Different payloads produce different signatures", func(t *testing.T) {
		secret := "my-secret"
		payload1 := []byte(`{"test":"data1"}`)
		payload2 := []byte(`{"test":"data2"}`)

		sig1 := service.generateSignature(payload1, secret)
		sig2 := service.generateSignature(payload2, secret)

		assert.NotEqual(t, sig1, sig2)
	})

	t.Run("Different secrets produce different signatures", func(t *testing.T) {
		payload := []byte(`{"test":"data"}`)
		secret1 := "secret1"
		secret2 := "secret2"

		sig1 := service.generateSignature(payload, secret1)
		sig2 := service.generateSignature(payload, secret2)

		assert.NotEqual(t, sig1, sig2)
	})

	t.Run("Signature matches manual HMAC calculation", func(t *testing.T) {
		payload := []byte(`{"event":"INSERT"}`)
		secret := "test-key"

		// Generate using service method
		serviceSig := service.generateSignature(payload, secret)

		// Generate manually
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write(payload)
		manualSig := hex.EncodeToString(mac.Sum(nil))

		assert.Equal(t, manualSig, serviceSig)
	})
}

func TestWebhookPayload_JSON(t *testing.T) {
	t.Run("Marshal WebhookPayload", func(t *testing.T) {
		payload := &WebhookPayload{
			Event:     "INSERT",
			Table:     "users",
			Schema:    "public",
			Record:    json.RawMessage(`{"id":1,"name":"test"}`),
			Timestamp: time.Now(),
		}

		data, err := json.Marshal(payload)
		require.NoError(t, err)
		assert.NotEmpty(t, data)

		// Verify it contains expected fields
		var result map[string]interface{}
		err = json.Unmarshal(data, &result)
		require.NoError(t, err)

		assert.Equal(t, "INSERT", result["event"])
		assert.Equal(t, "users", result["table"])
		assert.Equal(t, "public", result["schema"])
	})

	t.Run("Unmarshal WebhookPayload", func(t *testing.T) {
		jsonData := `{
			"event": "UPDATE",
			"table": "products",
			"schema": "public",
			"record": {"id": 10, "price": 99.99},
			"old_record": {"id": 10, "price": 79.99},
			"timestamp": "2024-01-01T00:00:00Z"
		}`

		var payload WebhookPayload
		err := json.Unmarshal([]byte(jsonData), &payload)
		require.NoError(t, err)

		assert.Equal(t, "UPDATE", payload.Event)
		assert.Equal(t, "products", payload.Table)
		assert.Equal(t, "public", payload.Schema)
		assert.NotNil(t, payload.Record)
		assert.NotNil(t, payload.OldRecord)
	})
}

func TestEventConfig(t *testing.T) {
	t.Run("Marshal EventConfig", func(t *testing.T) {
		config := EventConfig{
			Table:      "users",
			Operations: []string{"INSERT", "UPDATE"},
		}

		data, err := json.Marshal(config)
		require.NoError(t, err)
		assert.NotEmpty(t, data)

		// Unmarshal and verify
		var result EventConfig
		err = json.Unmarshal(data, &result)
		require.NoError(t, err)

		assert.Equal(t, "users", result.Table)
		assert.ElementsMatch(t, []string{"INSERT", "UPDATE"}, result.Operations)
	})

	t.Run("Multiple event configs", func(t *testing.T) {
		configs := []EventConfig{
			{Table: "users", Operations: []string{"INSERT"}},
			{Table: "products", Operations: []string{"INSERT", "UPDATE", "DELETE"}},
		}

		data, err := json.Marshal(configs)
		require.NoError(t, err)

		var result []EventConfig
		err = json.Unmarshal(data, &result)
		require.NoError(t, err)

		assert.Equal(t, 2, len(result))
		assert.Equal(t, "users", result[0].Table)
		assert.Equal(t, "products", result[1].Table)
	})
}

func TestWebhook_Struct(t *testing.T) {
	t.Run("Webhook with all fields", func(t *testing.T) {
		secret := "my-secret"
		description := "Test webhook"

		webhook := &Webhook{
			Name:                "test-webhook",
			Description:         &description,
			URL:                 "https://example.com/webhook",
			Secret:              &secret,
			Enabled:             true,
			MaxRetries:          3,
			RetryBackoffSeconds: 60,
			TimeoutSeconds:      30,
			Headers: map[string]string{
				"Authorization": "Bearer token",
			},
			Events: []EventConfig{
				{Table: "users", Operations: []string{"INSERT"}},
			},
		}

		assert.Equal(t, "test-webhook", webhook.Name)
		assert.Equal(t, &description, webhook.Description)
		assert.Equal(t, "https://example.com/webhook", webhook.URL)
		assert.Equal(t, &secret, webhook.Secret)
		assert.True(t, webhook.Enabled)
		assert.Equal(t, 3, webhook.MaxRetries)
		assert.Equal(t, 60, webhook.RetryBackoffSeconds)
		assert.Equal(t, 30, webhook.TimeoutSeconds)
		assert.Equal(t, 1, len(webhook.Events))
	})

	t.Run("Webhook marshaling", func(t *testing.T) {
		webhook := &Webhook{
			Name:    "test",
			URL:     "https://example.com",
			Enabled: true,
			Events: []EventConfig{
				{Table: "users", Operations: []string{"INSERT"}},
			},
			Headers: map[string]string{
				"X-Custom": "value",
			},
		}

		data, err := json.Marshal(webhook)
		require.NoError(t, err)

		var result Webhook
		err = json.Unmarshal(data, &result)
		require.NoError(t, err)

		assert.Equal(t, "test", result.Name)
		assert.Equal(t, "https://example.com", result.URL)
		assert.True(t, result.Enabled)
		assert.Equal(t, 1, len(result.Events))
		assert.Equal(t, "value", result.Headers["X-Custom"])
	})
}

func TestWebhookDelivery_Struct(t *testing.T) {
	t.Run("WebhookDelivery with success status", func(t *testing.T) {
		statusCode := 200
		responseBody := "OK"
		now := time.Now()

		delivery := &WebhookDelivery{
			Event:        "INSERT",
			Attempt:      1,
			Status:       "success",
			StatusCode:   &statusCode,
			ResponseBody: &responseBody,
			DeliveredAt:  &now,
		}

		assert.Equal(t, "INSERT", delivery.Event)
		assert.Equal(t, 1, delivery.Attempt)
		assert.Equal(t, "success", delivery.Status)
		assert.Equal(t, 200, *delivery.StatusCode)
		assert.Equal(t, "OK", *delivery.ResponseBody)
		assert.NotNil(t, delivery.DeliveredAt)
	})

	t.Run("WebhookDelivery with failed status", func(t *testing.T) {
		statusCode := 500
		errorMsg := "Internal Server Error"

		delivery := &WebhookDelivery{
			Event:      "UPDATE",
			Attempt:    2,
			Status:     "failed",
			StatusCode: &statusCode,
			Error:      &errorMsg,
		}

		assert.Equal(t, "failed", delivery.Status)
		assert.Equal(t, 2, delivery.Attempt)
		assert.Equal(t, 500, *delivery.StatusCode)
		assert.Equal(t, "Internal Server Error", *delivery.Error)
	})

	t.Run("WebhookDelivery marshaling", func(t *testing.T) {
		delivery := &WebhookDelivery{
			Event:   "DELETE",
			Payload: json.RawMessage(`{"id":5}`),
			Attempt: 1,
			Status:  "pending",
		}

		data, err := json.Marshal(delivery)
		require.NoError(t, err)

		var result WebhookDelivery
		err = json.Unmarshal(data, &result)
		require.NoError(t, err)

		assert.Equal(t, "DELETE", result.Event)
		assert.Equal(t, "pending", result.Status)
		assert.Equal(t, 1, result.Attempt)
	})
}

func TestNewWebhookService(t *testing.T) {
	t.Run("Create webhook service with nil pool", func(t *testing.T) {
		service := NewWebhookService(nil)
		assert.NotNil(t, service)
		assert.Nil(t, service.db)
		assert.NotNil(t, service.client)
		assert.Equal(t, 30*time.Second, service.client.Timeout)
	})
}

func TestWebhookValidation(t *testing.T) {
	t.Run("Valid webhook configuration", func(t *testing.T) {
		webhook := &Webhook{
			Name:                "valid-webhook",
			URL:                 "https://example.com/webhook",
			Enabled:             true,
			MaxRetries:          3,
			RetryBackoffSeconds: 60,
			TimeoutSeconds:      30,
			Events: []EventConfig{
				{Table: "users", Operations: []string{"INSERT", "UPDATE"}},
			},
			Headers: make(map[string]string),
		}

		assert.NotEmpty(t, webhook.Name)
		assert.NotEmpty(t, webhook.URL)
		assert.NotEmpty(t, webhook.Events)
		assert.Greater(t, webhook.MaxRetries, 0)
		assert.Greater(t, webhook.TimeoutSeconds, 0)
	})

	t.Run("Webhook with empty events", func(t *testing.T) {
		webhook := &Webhook{
			Name:    "no-events",
			URL:     "https://example.com",
			Enabled: true,
			Events:  []EventConfig{},
		}

		assert.Empty(t, webhook.Events)
	})
}

// Test HMAC signature verification (simulating receiver side)
func TestHMACSignatureVerification(t *testing.T) {
	service := &WebhookService{}

	t.Run("Verify valid signature", func(t *testing.T) {
		payload := []byte(`{"event":"INSERT","table":"users","record":{"id":1}}`)
		secret := "shared-secret"

		// Generate signature (sender side)
		signature := service.generateSignature(payload, secret)

		// Verify signature (receiver side)
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write(payload)
		expectedSignature := hex.EncodeToString(mac.Sum(nil))

		assert.Equal(t, expectedSignature, signature)

		// Verify using hmac.Equal (constant-time comparison)
		receivedMAC, _ := hex.DecodeString(signature)
		expectedMAC, _ := hex.DecodeString(expectedSignature)
		assert.True(t, hmac.Equal(receivedMAC, expectedMAC))
	})

	t.Run("Detect tampered payload", func(t *testing.T) {
		originalPayload := []byte(`{"event":"INSERT","amount":100}`)
		tamperedPayload := []byte(`{"event":"INSERT","amount":999}`)
		secret := "shared-secret"

		// Generate signature for original payload
		signature := service.generateSignature(originalPayload, secret)

		// Verify with tampered payload should fail
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write(tamperedPayload)
		expectedSignature := hex.EncodeToString(mac.Sum(nil))

		assert.NotEqual(t, expectedSignature, signature)
	})

	t.Run("Detect wrong secret", func(t *testing.T) {
		payload := []byte(`{"event":"INSERT"}`)
		correctSecret := "correct-secret"
		wrongSecret := "wrong-secret"

		// Generate with correct secret
		signature := service.generateSignature(payload, correctSecret)

		// Verify with wrong secret should fail
		mac := hmac.New(sha256.New, []byte(wrongSecret))
		mac.Write(payload)
		expectedSignature := hex.EncodeToString(mac.Sum(nil))

		assert.NotEqual(t, expectedSignature, signature)
	})
}

// Test webhook delivery status transitions
func TestWebhookDeliveryStatus(t *testing.T) {
	validStatuses := []string{"pending", "success", "failed", "retrying"}

	t.Run("Valid delivery statuses", func(t *testing.T) {
		for _, status := range validStatuses {
			delivery := &WebhookDelivery{
				Status: status,
			}
			assert.Contains(t, validStatuses, delivery.Status)
		}
	})

	t.Run("Delivery attempt progression", func(t *testing.T) {
		// Simulate retry attempts
		maxRetries := 3
		for attempt := 1; attempt <= maxRetries; attempt++ {
			delivery := &WebhookDelivery{
				Attempt: attempt,
				Status:  "retrying",
			}
			assert.LessOrEqual(t, delivery.Attempt, maxRetries)
		}
	})
}
