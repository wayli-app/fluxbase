package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// SettingsHandler Construction Tests
// =============================================================================

func TestNewSettingsHandler(t *testing.T) {
	t.Run("creates handler with nil database", func(t *testing.T) {
		handler := NewSettingsHandler(nil)
		assert.NotNil(t, handler)
		assert.Nil(t, handler.db)
	})
}

// =============================================================================
// SettingResponse Struct Tests
// =============================================================================

func TestSettingResponse_Struct(t *testing.T) {
	t.Run("string value", func(t *testing.T) {
		resp := SettingResponse{
			Value: "hello",
		}

		assert.Equal(t, "hello", resp.Value)
	})

	t.Run("integer value", func(t *testing.T) {
		resp := SettingResponse{
			Value: 42,
		}

		assert.Equal(t, 42, resp.Value)
	})

	t.Run("boolean value", func(t *testing.T) {
		resp := SettingResponse{
			Value: true,
		}

		assert.Equal(t, true, resp.Value)
	})

	t.Run("map value", func(t *testing.T) {
		resp := SettingResponse{
			Value: map[string]interface{}{
				"nested": "value",
				"count":  123,
			},
		}

		valueMap, ok := resp.Value.(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "value", valueMap["nested"])
		assert.Equal(t, 123, valueMap["count"])
	})

	t.Run("array value", func(t *testing.T) {
		resp := SettingResponse{
			Value: []string{"a", "b", "c"},
		}

		valueArr, ok := resp.Value.([]string)
		require.True(t, ok)
		assert.Len(t, valueArr, 3)
	})

	t.Run("nil value", func(t *testing.T) {
		resp := SettingResponse{
			Value: nil,
		}

		assert.Nil(t, resp.Value)
	})

	t.Run("JSON serialization - string", func(t *testing.T) {
		resp := SettingResponse{Value: "test value"}

		data, err := json.Marshal(resp)
		require.NoError(t, err)

		assert.Contains(t, string(data), `"value":"test value"`)
	})

	t.Run("JSON serialization - number", func(t *testing.T) {
		resp := SettingResponse{Value: 123}

		data, err := json.Marshal(resp)
		require.NoError(t, err)

		assert.Contains(t, string(data), `"value":123`)
	})

	t.Run("JSON serialization - boolean", func(t *testing.T) {
		resp := SettingResponse{Value: false}

		data, err := json.Marshal(resp)
		require.NoError(t, err)

		assert.Contains(t, string(data), `"value":false`)
	})
}

// =============================================================================
// BatchSettingsRequest Struct Tests
// =============================================================================

func TestBatchSettingsRequest_Struct(t *testing.T) {
	t.Run("all fields accessible", func(t *testing.T) {
		req := BatchSettingsRequest{
			Keys: []string{"setting1", "setting2", "setting3"},
		}

		assert.Len(t, req.Keys, 3)
		assert.Contains(t, req.Keys, "setting1")
		assert.Contains(t, req.Keys, "setting2")
		assert.Contains(t, req.Keys, "setting3")
	})

	t.Run("empty keys array", func(t *testing.T) {
		req := BatchSettingsRequest{
			Keys: []string{},
		}

		assert.Empty(t, req.Keys)
	})

	t.Run("single key", func(t *testing.T) {
		req := BatchSettingsRequest{
			Keys: []string{"only_one"},
		}

		assert.Len(t, req.Keys, 1)
		assert.Equal(t, "only_one", req.Keys[0])
	})

	t.Run("JSON deserialization", func(t *testing.T) {
		jsonData := `{"keys":["app.name","app.version","app.debug"]}`

		var req BatchSettingsRequest
		err := json.Unmarshal([]byte(jsonData), &req)
		require.NoError(t, err)

		assert.Len(t, req.Keys, 3)
		assert.Contains(t, req.Keys, "app.name")
		assert.Contains(t, req.Keys, "app.version")
		assert.Contains(t, req.Keys, "app.debug")
	})

	t.Run("JSON serialization", func(t *testing.T) {
		req := BatchSettingsRequest{
			Keys: []string{"key1", "key2"},
		}

		data, err := json.Marshal(req)
		require.NoError(t, err)

		assert.Contains(t, string(data), `"keys"`)
		assert.Contains(t, string(data), `"key1"`)
		assert.Contains(t, string(data), `"key2"`)
	})
}

// =============================================================================
// BatchSettingsResponse Struct Tests
// =============================================================================

func TestBatchSettingsResponse_Struct(t *testing.T) {
	t.Run("all fields accessible", func(t *testing.T) {
		resp := BatchSettingsResponse{
			Key:   "app.name",
			Value: "My Application",
		}

		assert.Equal(t, "app.name", resp.Key)
		assert.Equal(t, "My Application", resp.Value)
	})

	t.Run("various value types", func(t *testing.T) {
		testCases := []struct {
			name  string
			key   string
			value interface{}
		}{
			{"string", "app.name", "Test App"},
			{"int", "app.timeout", 30},
			{"bool", "app.debug", true},
			{"float", "app.ratio", 0.75},
			{"nil", "app.optional", nil},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				resp := BatchSettingsResponse{
					Key:   tc.key,
					Value: tc.value,
				}

				assert.Equal(t, tc.key, resp.Key)
				assert.Equal(t, tc.value, resp.Value)
			})
		}
	})

	t.Run("JSON serialization", func(t *testing.T) {
		resp := BatchSettingsResponse{
			Key:   "theme.color",
			Value: "#ff0000",
		}

		data, err := json.Marshal(resp)
		require.NoError(t, err)

		assert.Contains(t, string(data), `"key":"theme.color"`)
		assert.Contains(t, string(data), `"value":"#ff0000"`)
	})

	t.Run("JSON serialization with complex value", func(t *testing.T) {
		resp := BatchSettingsResponse{
			Key: "feature.flags",
			Value: map[string]bool{
				"feature1": true,
				"feature2": false,
			},
		}

		data, err := json.Marshal(resp)
		require.NoError(t, err)

		assert.Contains(t, string(data), `"key":"feature.flags"`)
		assert.Contains(t, string(data), `"value"`)
	})
}

// =============================================================================
// GetSetting Handler Tests
// =============================================================================

func TestGetSetting_ParameterValidation(t *testing.T) {
	t.Run("empty key parameter", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewSettingsHandler(nil)

		app.Get("/settings/:key", handler.GetSetting)

		// Empty key should be treated as route not found by Fiber
		req := httptest.NewRequest(http.MethodGet, "/settings/", nil)

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		// Fiber treats empty param as route not found
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	})

	t.Run("valid key parameter", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewSettingsHandler(nil)

		app.Get("/settings/:key", handler.GetSetting)

		req := httptest.NewRequest(http.MethodGet, "/settings/app.name", nil)

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		// Handler was reached, fails at DB operation
		assert.NotEqual(t, fiber.StatusNotFound, resp.StatusCode)
	})

	t.Run("key with dots", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewSettingsHandler(nil)

		app.Get("/settings/:key", handler.GetSetting)

		req := httptest.NewRequest(http.MethodGet, "/settings/app.feature.enabled", nil)

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		// Dots in key should be valid
		assert.NotEqual(t, fiber.StatusNotFound, resp.StatusCode)
	})

	t.Run("key with underscores", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewSettingsHandler(nil)

		app.Get("/settings/:key", handler.GetSetting)

		req := httptest.NewRequest(http.MethodGet, "/settings/my_setting_key", nil)

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.NotEqual(t, fiber.StatusNotFound, resp.StatusCode)
	})

	t.Run("key with hyphens", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewSettingsHandler(nil)

		app.Get("/settings/:key", handler.GetSetting)

		req := httptest.NewRequest(http.MethodGet, "/settings/my-setting-key", nil)

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.NotEqual(t, fiber.StatusNotFound, resp.StatusCode)
	})
}

// =============================================================================
// GetSettings (Batch) Handler Tests
// =============================================================================

func TestGetSettings_Validation(t *testing.T) {
	t.Run("invalid request body", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewSettingsHandler(nil)

		app.Post("/settings/batch", handler.GetSettings)

		req := httptest.NewRequest(http.MethodPost, "/settings/batch", bytes.NewReader([]byte("invalid json")))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

		respBody, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal(respBody, &result)
		require.NoError(t, err)

		assert.Contains(t, result["error"], "Invalid request body")
	})

	t.Run("empty keys array and no prefix", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewSettingsHandler(nil)

		app.Post("/settings/batch", handler.GetSettings)

		body := `{"keys":[]}`
		req := httptest.NewRequest(http.MethodPost, "/settings/batch", bytes.NewReader([]byte(body)))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

		respBody, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal(respBody, &result)
		require.NoError(t, err)

		assert.Contains(t, result["error"], "keys or prefix")
	})

	t.Run("missing keys and prefix fields", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewSettingsHandler(nil)

		app.Post("/settings/batch", handler.GetSettings)

		body := `{}`
		req := httptest.NewRequest(http.MethodPost, "/settings/batch", bytes.NewReader([]byte(body)))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

		respBody, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal(respBody, &result)
		require.NoError(t, err)

		assert.Contains(t, result["error"], "keys or prefix")
	})

	t.Run("prefix without trailing dot is rejected", func(t *testing.T) {
		// Namespace hygiene: a prefix must end with '.' so callers can't probe
		// for a specific key's existence (e.g. prefix "wayli.secret") and can't
		// accidentally over-match across namespaces.
		app := newTestApp(t)
		handler := NewSettingsHandler(nil)

		app.Post("/settings/batch", handler.GetSettings)

		body := `{"prefix":"wayli.secret"}`
		req := httptest.NewRequest(http.MethodPost, "/settings/batch", bytes.NewReader([]byte(body)))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

		respBody, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal(respBody, &result)
		require.NoError(t, err)

		assert.Contains(t, result["error"], "prefix must end with a '.'")
	})

	t.Run("valid prefix passes validation", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewSettingsHandler(nil)

		app.Post("/settings/batch", handler.GetSettings)

		body := `{"prefix":"wayli."}`
		req := httptest.NewRequest(http.MethodPost, "/settings/batch", bytes.NewReader([]byte(body)))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		// Validation passes (fails later at the nil-DB operation, not at 400).
		assert.NotEqual(t, fiber.StatusBadRequest, resp.StatusCode)
	})

	t.Run("too many keys", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewSettingsHandler(nil)

		app.Post("/settings/batch", handler.GetSettings)

		// Create 101 keys
		keys := make([]string, 101)
		for i := 0; i < 101; i++ {
			keys[i] = "key" + string(rune(i))
		}

		reqBody := BatchSettingsRequest{Keys: keys}
		bodyJSON, err := json.Marshal(reqBody)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/settings/batch", bytes.NewReader(bodyJSON))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

		respBody, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal(respBody, &result)
		require.NoError(t, err)

		assert.Contains(t, result["error"], "Maximum 100 keys")
	})

	t.Run("exactly 100 keys allowed", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewSettingsHandler(nil)

		app.Post("/settings/batch", handler.GetSettings)

		// Create exactly 100 keys
		keys := make([]string, 100)
		for i := 0; i < 100; i++ {
			keys[i] = "key" + string(rune(i))
		}

		reqBody := BatchSettingsRequest{Keys: keys}
		bodyJSON, err := json.Marshal(reqBody)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/settings/batch", bytes.NewReader(bodyJSON))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		// Should not be bad request (100 is exactly the limit)
		// Will fail at DB operation, but validation passes
		assert.NotEqual(t, fiber.StatusBadRequest, resp.StatusCode)
	})

	t.Run("valid request with single key", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewSettingsHandler(nil)

		app.Post("/settings/batch", handler.GetSettings)

		body := `{"keys":["app.name"]}`
		req := httptest.NewRequest(http.MethodPost, "/settings/batch", bytes.NewReader([]byte(body)))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		// Validation passes, fails at DB operation
		assert.NotEqual(t, fiber.StatusBadRequest, resp.StatusCode)
	})

	t.Run("valid request with multiple keys", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewSettingsHandler(nil)

		app.Post("/settings/batch", handler.GetSettings)

		body := `{"keys":["app.name","app.version","app.debug","theme.color"]}`
		req := httptest.NewRequest(http.MethodPost, "/settings/batch", bytes.NewReader([]byte(body)))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		// Validation passes, fails at DB operation
		assert.NotEqual(t, fiber.StatusBadRequest, resp.StatusCode)
	})
}

// =============================================================================
// Setting Key Format Tests
// =============================================================================

func TestSettingKeyFormats(t *testing.T) {
	validKeys := []string{
		"simple",
		"app.name",
		"app.feature.enabled",
		"my_setting",
		"my-setting",
		"app_v2_setting",
		"UPPERCASE",
		"MixedCase",
		"with123numbers",
		"a", // single character
	}

	for _, key := range validKeys {
		t.Run("valid key: "+key, func(t *testing.T) {
			req := BatchSettingsRequest{
				Keys: []string{key},
			}
			assert.Len(t, req.Keys, 1)
			assert.Equal(t, key, req.Keys[0])
		})
	}
}

// =============================================================================
// Batch Response Formatting Tests
// =============================================================================

func TestBatchResponseFormatting(t *testing.T) {
	t.Run("multiple settings in response", func(t *testing.T) {
		responses := []BatchSettingsResponse{
			{Key: "app.name", Value: "Test App"},
			{Key: "app.version", Value: "1.0.0"},
			{Key: "app.debug", Value: false},
			{Key: "app.timeout", Value: 30},
		}

		data, err := json.Marshal(responses)
		require.NoError(t, err)

		var parsed []BatchSettingsResponse
		err = json.Unmarshal(data, &parsed)
		require.NoError(t, err)

		assert.Len(t, parsed, 4)

		// Find each expected setting
		keyValueMap := make(map[string]interface{})
		for _, resp := range parsed {
			keyValueMap[resp.Key] = resp.Value
		}

		assert.Equal(t, "Test App", keyValueMap["app.name"])
		assert.Equal(t, "1.0.0", keyValueMap["app.version"])
		assert.Equal(t, false, keyValueMap["app.debug"])
		assert.Equal(t, float64(30), keyValueMap["app.timeout"]) // JSON numbers are float64
	})

	t.Run("empty response array", func(t *testing.T) {
		responses := []BatchSettingsResponse{}

		data, err := json.Marshal(responses)
		require.NoError(t, err)

		assert.Equal(t, "[]", string(data))
	})
}
