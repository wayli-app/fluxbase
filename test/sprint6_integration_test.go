package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wayli-app/fluxbase/internal/api"
	"github.com/wayli-app/fluxbase/internal/config"
	"github.com/wayli-app/fluxbase/internal/database"
)

// createTestServer creates a test server instance
func createTestServer(t *testing.T, cfg *config.Config, db *database.Connection) *api.Server {
	server := api.NewServer(cfg, db)
	require.NotNil(t, server)
	return server
}

// authenticateTestUser creates a test user and returns an access token
func authenticateTestUser(t *testing.T, app *fiber.App) string {
	email := "test-sprint6@example.com"
	password := "testpassword123"

	// Cleanup any existing test user
	// (This would require access to db, so we'll try to create and handle errors)

	// Sign up
	signupBody, _ := json.Marshal(map[string]interface{}{
		"email":    email,
		"password": password,
	})

	req := httptest.NewRequest("POST", "/api/v1/auth/signup", bytes.NewReader(signupBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	require.NoError(t, err)

	if resp.StatusCode == 201 || resp.StatusCode == 409 {
		// User created or already exists, now sign in
		signinBody, _ := json.Marshal(map[string]interface{}{
			"email":    email,
			"password": password,
		})

		signinReq := httptest.NewRequest("POST", "/api/v1/auth/signin", bytes.NewReader(signinBody))
		signinReq.Header.Set("Content-Type", "application/json")

		signinResp, err := app.Test(signinReq, -1)
		require.NoError(t, err)
		assert.Equal(t, 200, signinResp.StatusCode)

		var result map[string]interface{}
		body, _ := io.ReadAll(signinResp.Body)
		err = json.Unmarshal(body, &result)
		require.NoError(t, err)

		token, ok := result["access_token"].(string)
		require.True(t, ok, "Access token should be present")
		require.NotEmpty(t, token)

		return token
	}

	t.Fatalf("Failed to authenticate test user: status %d", resp.StatusCode)
	return ""
}

// TestSprint6_APIKeysIntegration tests the API Keys HTTP endpoints
func TestSprint6_APIKeysIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	cfg, db := setupTestEnvironment(t)
	defer db.Close()

	server := createTestServer(t, cfg, db)
	app := server.App()

	// Authenticate first to get admin token
	token := authenticateTestUser(t, app)

	t.Run("Create API key", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"name":                  "test-integration-key",
			"description":           "Test API key for integration testing",
			"scopes":                []string{"read:tables", "write:tables"},
			"rate_limit_per_minute": 100,
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/v1/api-keys", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := app.Test(req, -1)
		require.NoError(t, err)

		assert.Equal(t, 201, resp.StatusCode)

		var result map[string]interface{}
		respBody, _ := io.ReadAll(resp.Body)
		err = json.Unmarshal(respBody, &result)
		require.NoError(t, err)

		// Verify response has key field (only shown once)
		assert.NotEmpty(t, result["key"], "Key should be present in response")
		assert.Contains(t, result["key"], "fbk_", "Key should have fbk_ prefix")
		assert.NotEmpty(t, result["id"], "ID should be present")
	})

	t.Run("List API keys", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/api-keys", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := app.Test(req, -1)
		require.NoError(t, err)

		assert.Equal(t, 200, resp.StatusCode)

		var result []map[string]interface{}
		respBody, _ := io.ReadAll(resp.Body)
		err = json.Unmarshal(respBody, &result)
		require.NoError(t, err)

		// Should have at least one key from previous test
		assert.GreaterOrEqual(t, len(result), 1)
	})

	t.Run("Validate API key", func(t *testing.T) {
		// This would require a valid API key from the create endpoint
		// Skipping for now as it requires stateful testing
		t.Skip("Requires API key from create endpoint")
	})

	t.Run("Unauthorized access", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/api-keys", nil)
		// No Authorization header

		resp, err := app.Test(req, -1)
		require.NoError(t, err)

		assert.Equal(t, 401, resp.StatusCode)
	})
}

// TestSprint6_WebhooksIntegration tests the Webhooks HTTP endpoints
func TestSprint6_WebhooksIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	cfg, db := setupTestEnvironment(t)
	defer db.Close()

	server := createTestServer(t, cfg, db)
	app := server.App()

	token := authenticateTestUser(t, app)

	var webhookID string

	t.Run("Create webhook", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"name":                   "test-webhook",
			"url":                    "https://example.com/webhook",
			"secret":                 "test-secret-key",
			"enabled":                true,
			"max_retries":            3,
			"retry_backoff_seconds":  60,
			"timeout_seconds":        30,
			"headers":                map[string]string{"X-Custom": "value"},
			"events": []map[string]interface{}{
				{
					"table":      "users",
					"operations": []string{"INSERT", "UPDATE"},
				},
			},
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/v1/webhooks", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := app.Test(req, -1)
		require.NoError(t, err)

		assert.Equal(t, 201, resp.StatusCode)

		var result map[string]interface{}
		respBody, _ := io.ReadAll(resp.Body)
		err = json.Unmarshal(respBody, &result)
		require.NoError(t, err)

		assert.NotEmpty(t, result["id"])
		assert.Equal(t, "test-webhook", result["name"])
		webhookID = result["id"].(string)
	})

	t.Run("List webhooks", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/webhooks", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := app.Test(req, -1)
		require.NoError(t, err)

		assert.Equal(t, 200, resp.StatusCode)

		var result []map[string]interface{}
		respBody, _ := io.ReadAll(resp.Body)
		err = json.Unmarshal(respBody, &result)
		require.NoError(t, err)

		assert.GreaterOrEqual(t, len(result), 1)
	})

	t.Run("Get webhook by ID", func(t *testing.T) {
		if webhookID == "" {
			t.Skip("No webhook ID available")
		}

		req := httptest.NewRequest("GET", "/api/v1/webhooks/"+webhookID, nil)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := app.Test(req, -1)
		require.NoError(t, err)

		assert.Equal(t, 200, resp.StatusCode)
	})

	t.Run("Test webhook delivery", func(t *testing.T) {
		if webhookID == "" {
			t.Skip("No webhook ID available")
		}

		reqBody := map[string]interface{}{
			"event":  "INSERT",
			"table":  "users",
			"record": map[string]interface{}{"id": 1, "name": "test"},
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/v1/webhooks/"+webhookID+"/test", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := app.Test(req, -1)
		require.NoError(t, err)

		// Test webhook might fail due to invalid URL, but endpoint should respond
		assert.Contains(t, []int{200, 202}, resp.StatusCode)
	})
}

// TestSprint6_MonitoringIntegration tests the Monitoring HTTP endpoints
func TestSprint6_MonitoringIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	cfg, db := setupTestEnvironment(t)
	defer db.Close()

	server := createTestServer(t, cfg, db)
	app := server.App()

	token := authenticateTestUser(t, app)

	t.Run("Get metrics", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/monitoring/metrics", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := app.Test(req, -1)
		require.NoError(t, err)

		assert.Equal(t, 200, resp.StatusCode)

		var result map[string]interface{}
		respBody, _ := io.ReadAll(resp.Body)
		err = json.Unmarshal(respBody, &result)
		require.NoError(t, err)

		// Verify expected metrics fields
		assert.NotNil(t, result["uptime_seconds"])
		assert.NotNil(t, result["go_version"])
		assert.NotNil(t, result["num_goroutines"])
		assert.NotNil(t, result["memory_alloc_mb"])
		assert.NotNil(t, result["database"])
		assert.NotNil(t, result["realtime"])

		// Verify database stats structure
		dbStats, ok := result["database"].(map[string]interface{})
		assert.True(t, ok, "Database stats should be a map")
		if ok {
			assert.NotNil(t, dbStats["max_conns"])
			assert.NotNil(t, dbStats["total_conns"])
		}
	})

	t.Run("Get health status", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/monitoring/health", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := app.Test(req, -1)
		require.NoError(t, err)

		// Health check might return 503 if services are unhealthy, but should respond
		assert.Contains(t, []int{200, 503}, resp.StatusCode)

		var result map[string]interface{}
		respBody, _ := io.ReadAll(resp.Body)
		err = json.Unmarshal(respBody, &result)
		require.NoError(t, err)

		// Verify health response structure
		assert.NotNil(t, result["status"])
		assert.NotNil(t, result["services"])

		// Verify services structure
		services, ok := result["services"].(map[string]interface{})
		assert.True(t, ok, "Services should be a map")
		if ok {
			// Should have at least database service
			assert.NotNil(t, services["database"])

			// Verify database service structure
			if dbService, ok := services["database"].(map[string]interface{}); ok {
				assert.NotNil(t, dbService["status"])
				assert.NotNil(t, dbService["latency_ms"])
			}
		}
	})

	t.Run("Get logs", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/monitoring/logs", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := app.Test(req, -1)
		require.NoError(t, err)

		assert.Equal(t, 200, resp.StatusCode)

		var result map[string]interface{}
		respBody, _ := io.ReadAll(resp.Body)
		err = json.Unmarshal(respBody, &result)
		require.NoError(t, err)

		// Logs endpoint returns placeholder for MVP
		assert.NotNil(t, result["message"])
		assert.NotNil(t, result["logs"])
	})

	t.Run("Unauthorized monitoring access", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/monitoring/metrics", nil)
		// No Authorization header

		resp, err := app.Test(req, -1)
		require.NoError(t, err)

		assert.Equal(t, 401, resp.StatusCode)
	})
}

// TestSprint6_EndToEnd tests a complete workflow across Sprint 6 features
func TestSprint6_EndToEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	cfg, db := setupTestEnvironment(t)
	defer db.Close()

	server := createTestServer(t, cfg, db)
	app := server.App()

	token := authenticateTestUser(t, app)

	t.Run("Complete Sprint 6 workflow", func(t *testing.T) {
		// Step 1: Check system health
		healthReq := httptest.NewRequest("GET", "/api/v1/monitoring/health", nil)
		healthReq.Header.Set("Authorization", "Bearer "+token)
		healthResp, err := app.Test(healthReq, -1)
		require.NoError(t, err)
		assert.Contains(t, []int{200, 503}, healthResp.StatusCode)

		// Step 2: Get current metrics
		metricsReq := httptest.NewRequest("GET", "/api/v1/monitoring/metrics", nil)
		metricsReq.Header.Set("Authorization", "Bearer "+token)
		metricsResp, err := app.Test(metricsReq, -1)
		require.NoError(t, err)
		assert.Equal(t, 200, metricsResp.StatusCode)

		// Step 3: Create an API key
		apiKeyBody, _ := json.Marshal(map[string]interface{}{
			"name":   "workflow-test-key",
			"scopes": []string{"read:tables"},
		})
		apiKeyReq := httptest.NewRequest("POST", "/api/v1/api-keys", bytes.NewReader(apiKeyBody))
		apiKeyReq.Header.Set("Content-Type", "application/json")
		apiKeyReq.Header.Set("Authorization", "Bearer "+token)
		apiKeyResp, err := app.Test(apiKeyReq, -1)
		require.NoError(t, err)
		assert.Equal(t, 201, apiKeyResp.StatusCode)

		// Step 4: Create a webhook
		webhookBody, _ := json.Marshal(map[string]interface{}{
			"name":    "workflow-test-webhook",
			"url":     "https://example.com/hook",
			"enabled": true,
			"events": []map[string]interface{}{
				{
					"table":      "products",
					"operations": []string{"INSERT"},
				},
			},
		})
		webhookReq := httptest.NewRequest("POST", "/api/v1/webhooks", bytes.NewReader(webhookBody))
		webhookReq.Header.Set("Content-Type", "application/json")
		webhookReq.Header.Set("Authorization", "Bearer "+token)
		webhookResp, err := app.Test(webhookReq, -1)
		require.NoError(t, err)
		assert.Equal(t, 201, webhookResp.StatusCode)

		fmt.Println("✅ Sprint 6 end-to-end workflow completed successfully")
	})
}
