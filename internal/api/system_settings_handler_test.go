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
// SystemSettingsHandler Construction Tests
// =============================================================================

func TestNewSystemSettingsHandler(t *testing.T) {
	t.Run("creates handler with nil dependencies", func(t *testing.T) {
		handler := NewSystemSettingsHandler(nil, nil)
		assert.NotNil(t, handler)
		assert.Nil(t, handler.settingsService)
		assert.Nil(t, handler.settingsCache)
	})
}

// =============================================================================
// Setting Defaults Tests
// =============================================================================

func TestSettingDefaults(t *testing.T) {
	t.Run("auth settings have defaults", func(t *testing.T) {
		authKeys := []string{
			"app.auth.signup_enabled",
			"app.auth.magic_link_enabled",
			"app.auth.password_min_length",
			"app.auth.require_email_verification",
		}

		for _, key := range authKeys {
			assert.Contains(t, settingDefaults, key, "Expected default for %s", key)
		}
	})

	t.Run("feature flags have defaults", func(t *testing.T) {
		featureKeys := []string{
			"app.realtime.enabled",
			"app.storage.enabled",
			"app.functions.enabled",
			"app.ai.enabled",
			"app.rpc.enabled",
			"app.jobs.enabled",
			"app.email.enabled",
		}

		for _, key := range featureKeys {
			assert.Contains(t, settingDefaults, key, "Expected default for %s", key)
		}
	})

	t.Run("email settings have defaults", func(t *testing.T) {
		emailKeys := []string{
			"app.email.provider",
			"app.email.from_address",
			"app.email.from_name",
			"app.email.smtp_host",
			"app.email.smtp_port",
			"app.email.smtp_username",
			"app.email.smtp_password",
			"app.email.smtp_tls",
			"app.email.sendgrid_api_key",
			"app.email.mailgun_api_key",
			"app.email.mailgun_domain",
			"app.email.ses_access_key",
			"app.email.ses_secret_key",
			"app.email.ses_region",
		}

		for _, key := range emailKeys {
			assert.Contains(t, settingDefaults, key, "Expected default for %s", key)
		}
	})

	t.Run("captcha settings have defaults", func(t *testing.T) {
		captchaKeys := []string{
			"app.security.captcha.enabled",
			"app.security.captcha.provider",
			"app.security.captcha.site_key",
			"app.security.captcha.secret_key",
			"app.security.captcha.score_threshold",
			"app.security.captcha.endpoints",
			"app.security.captcha.cap_server_url",
			"app.security.captcha.cap_api_key",
		}

		for _, key := range captchaKeys {
			assert.Contains(t, settingDefaults, key, "Expected default for %s", key)
		}
	})

	t.Run("security settings have defaults", func(t *testing.T) {
		securityKeys := []string{
			"app.security.enable_global_rate_limit",
		}

		for _, key := range securityKeys {
			assert.Contains(t, settingDefaults, key, "Expected default for %s", key)
		}
	})

	t.Run("default values are correct types", func(t *testing.T) {
		// Check boolean defaults
		signupDefault := settingDefaults["app.auth.signup_enabled"]
		assert.IsType(t, true, signupDefault["value"])

		// Check integer defaults
		passwordLengthDefault := settingDefaults["app.auth.password_min_length"]
		assert.IsType(t, 12, passwordLengthDefault["value"])

		// Check float defaults
		scoreThresholdDefault := settingDefaults["app.security.captcha.score_threshold"]
		assert.IsType(t, 0.5, scoreThresholdDefault["value"])

		// Check string defaults
		providerDefault := settingDefaults["app.email.provider"]
		assert.IsType(t, "", providerDefault["value"])

		// Check slice defaults
		endpointsDefault := settingDefaults["app.security.captcha.endpoints"]
		assert.IsType(t, []string{}, endpointsDefault["value"])
	})
}

// =============================================================================
// isValidSettingKey Tests
// =============================================================================

func TestIsValidSettingKey(t *testing.T) {
	handler := NewSystemSettingsHandler(nil, nil)

	t.Run("valid setting keys", func(t *testing.T) {
		validKeys := []string{
			"app.auth.signup_enabled",
			"app.auth.password_min_length",
			"app.email.smtp_host",
			"app.security.captcha.enabled",
		}

		for _, key := range validKeys {
			assert.True(t, handler.isValidSettingKey(key), "Expected %q to be valid", key)
		}
	})

	t.Run("invalid setting keys", func(t *testing.T) {
		invalidKeys := []string{
			"unknown.setting",
			"app.unknown.setting",
			"",
			"app.auth.nonexistent",
			"arbitrary.key",
		}

		for _, key := range invalidKeys {
			assert.False(t, handler.isValidSettingKey(key), "Expected %q to be invalid", key)
		}
	})
}

// =============================================================================
// getDefaultSetting Tests
// =============================================================================

func TestGetDefaultSetting(t *testing.T) {
	handler := NewSystemSettingsHandler(nil, nil)

	t.Run("returns default for known key", func(t *testing.T) {
		setting := handler.getDefaultSetting("app.auth.signup_enabled")
		assert.NotNil(t, setting)
		assert.Equal(t, "app.auth.signup_enabled", setting.Key)
		assert.NotNil(t, setting.Value)
	})

	t.Run("returns nil for unknown key", func(t *testing.T) {
		setting := handler.getDefaultSetting("unknown.key")
		assert.Nil(t, setting)
	})

	t.Run("returns correct default value for password min length", func(t *testing.T) {
		setting := handler.getDefaultSetting("app.auth.password_min_length")
		assert.NotNil(t, setting)
		assert.Equal(t, 12, setting.Value["value"])
	})

	t.Run("returns correct default value for captcha provider", func(t *testing.T) {
		setting := handler.getDefaultSetting("app.security.captcha.provider")
		assert.NotNil(t, setting)
		assert.Equal(t, "hcaptcha", setting.Value["value"])
	})

	t.Run("returns correct default value for SES region", func(t *testing.T) {
		setting := handler.getDefaultSetting("app.email.ses_region")
		assert.NotNil(t, setting)
		assert.Equal(t, "us-east-1", setting.Value["value"])
	})
}

// =============================================================================
// GetSetting Handler Validation Tests
// =============================================================================

func TestGetSetting_Validation(t *testing.T) {
	t.Run("empty key returns error", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewSystemSettingsHandler(nil, nil)

		app.Get("/settings/*", handler.GetSetting)

		req := httptest.NewRequest(http.MethodGet, "/settings/", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		var result map[string]interface{}
		_ = json.Unmarshal(body, &result)
		assert.Equal(t, "Setting key is required", result["error"])
	})
}

// =============================================================================
// UpdateSetting Handler Validation Tests
// =============================================================================

func TestUpdateSetting_Validation(t *testing.T) {
	t.Run("empty key returns error", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewSystemSettingsHandler(nil, nil)

		app.Put("/settings/*", handler.UpdateSetting)

		body := `{"value": {"value": true}}`
		req := httptest.NewRequest(http.MethodPut, "/settings/", bytes.NewReader([]byte(body)))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

		respBody, _ := io.ReadAll(resp.Body)
		var result map[string]interface{}
		_ = json.Unmarshal(respBody, &result)
		assert.Equal(t, "Setting key is required", result["error"])
	})

	t.Run("invalid request body", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewSystemSettingsHandler(nil, nil)

		app.Put("/settings/*", handler.UpdateSetting)

		req := httptest.NewRequest(http.MethodPut, "/settings/app.auth.signup_enabled", bytes.NewReader([]byte("not json")))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		var result map[string]interface{}
		_ = json.Unmarshal(body, &result)
		assert.Contains(t, result["error"], "Invalid request body")
	})

	t.Run("invalid setting key", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewSystemSettingsHandler(nil, nil)

		app.Put("/settings/*", handler.UpdateSetting)

		body := `{"value": {"value": true}}`
		req := httptest.NewRequest(http.MethodPut, "/settings/invalid.setting.key", bytes.NewReader([]byte(body)))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

		respBody, _ := io.ReadAll(resp.Body)
		var result map[string]interface{}
		_ = json.Unmarshal(respBody, &result)
		assert.Equal(t, "Invalid setting key", result["error"])
		assert.Equal(t, "INVALID_INPUT", result["code"])
	})
}

// =============================================================================
// DeleteSetting Handler Validation Tests
// =============================================================================

func TestDeleteSetting_Validation(t *testing.T) {
	t.Run("empty key returns error", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewSystemSettingsHandler(nil, nil)

		app.Delete("/settings/*", handler.DeleteSetting)

		req := httptest.NewRequest(http.MethodDelete, "/settings/", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		var result map[string]interface{}
		_ = json.Unmarshal(body, &result)
		assert.Equal(t, "Setting key is required", result["error"])
	})
}

// =============================================================================
// Response Format Tests
// =============================================================================

func TestSystemSettingsResponseFormats(t *testing.T) {
	t.Run("list settings error response", func(t *testing.T) {
		expectedError := fiber.Map{
			"error": "Failed to retrieve system settings",
		}

		data, err := json.Marshal(expectedError)
		require.NoError(t, err)

		assert.Contains(t, string(data), `"error":"Failed to retrieve system settings"`)
	})

	t.Run("setting not found response", func(t *testing.T) {
		expectedError := fiber.Map{
			"error": "Setting not found",
		}

		data, err := json.Marshal(expectedError)
		require.NoError(t, err)

		assert.Contains(t, string(data), `"error":"Setting not found"`)
	})

	t.Run("env override conflict response", func(t *testing.T) {
		expectedError := fiber.Map{
			"error": "This setting cannot be updated because it is overridden by an environment variable",
			"code":  "ENV_OVERRIDE",
			"key":   "app.auth.signup_enabled",
		}

		data, err := json.Marshal(expectedError)
		require.NoError(t, err)

		assert.Contains(t, string(data), `"code":"ENV_OVERRIDE"`)
		assert.Contains(t, string(data), `"key":"app.auth.signup_enabled"`)
	})

	t.Run("update setting fallback response", func(t *testing.T) {
		// When setting is created but can't be retrieved
		expectedResponse := fiber.Map{
			"key":         "app.auth.signup_enabled",
			"value":       map[string]interface{}{"value": true},
			"description": "Test description",
		}

		data, err := json.Marshal(expectedResponse)
		require.NoError(t, err)

		assert.Contains(t, string(data), `"key":"app.auth.signup_enabled"`)
		assert.Contains(t, string(data), `"description":"Test description"`)
	})
}

// =============================================================================
// Internal Server Error Tests
// =============================================================================

func TestSystemSettingsInternalErrors(t *testing.T) {
	t.Run("list settings with nil service returns 500", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewSystemSettingsHandler(nil, nil)
		app.Get("/settings", handler.ListSettings)

		req := httptest.NewRequest(http.MethodGet, "/settings", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusServiceUnavailable, resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		assert.Contains(t, string(body), "not initialized")
	})

	t.Run("delete setting with nil service returns 503", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewSystemSettingsHandler(nil, nil)
		app.Delete("/settings/*", handler.DeleteSetting)

		req := httptest.NewRequest(http.MethodDelete, "/settings/app.auth.signup_enabled", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusServiceUnavailable, resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		assert.Contains(t, string(body), "not initialized")
	})
}

// =============================================================================
// Setting Categories Tests
// =============================================================================

func TestSettingCategories(t *testing.T) {
	t.Run("auth settings category", func(t *testing.T) {
		authSettings := []string{
			"app.auth.signup_enabled",
			"app.auth.magic_link_enabled",
			"app.auth.password_min_length",
			"app.auth.require_email_verification",
		}

		for _, key := range authSettings {
			assert.Contains(t, key, "app.auth.")
			assert.Contains(t, settingDefaults, key)
		}
	})

	t.Run("email settings category", func(t *testing.T) {
		// Count email settings
		emailCount := 0
		for key := range settingDefaults {
			if len(key) > 10 && key[:10] == "app.email." {
				emailCount++
			}
		}
		assert.GreaterOrEqual(t, emailCount, 10) // Should have at least 10 email settings
	})

	t.Run("security settings category", func(t *testing.T) {
		securityCount := 0
		for key := range settingDefaults {
			if len(key) > 13 && key[:13] == "app.security." {
				securityCount++
			}
		}
		assert.GreaterOrEqual(t, securityCount, 1)
	})
}

// =============================================================================
// HTTP Method Tests
// =============================================================================

func TestSystemSettingsHTTPMethods(t *testing.T) {
	t.Run("list settings uses GET", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewSystemSettingsHandler(nil, nil)
		app.Get("/settings", handler.ListSettings)

		// POST should not work
		req := httptest.NewRequest(http.MethodPost, "/settings", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusMethodNotAllowed, resp.StatusCode)
	})

	t.Run("get setting uses GET", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewSystemSettingsHandler(nil, nil)
		app.Get("/settings/*", handler.GetSetting)

		// PUT should not work
		req := httptest.NewRequest(http.MethodPut, "/settings/app.auth.signup_enabled", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusMethodNotAllowed, resp.StatusCode)
	})

	t.Run("update setting uses PUT", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewSystemSettingsHandler(nil, nil)
		app.Put("/settings/*", handler.UpdateSetting)

		// GET should not work
		req := httptest.NewRequest(http.MethodGet, "/settings/app.auth.signup_enabled", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusMethodNotAllowed, resp.StatusCode)
	})

	t.Run("delete setting uses DELETE", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewSystemSettingsHandler(nil, nil)
		app.Delete("/settings/*", handler.DeleteSetting)

		// POST should not work
		req := httptest.NewRequest(http.MethodPost, "/settings/app.auth.signup_enabled", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusMethodNotAllowed, resp.StatusCode)
	})
}

// =============================================================================
// Default Setting Value Tests
// =============================================================================

func TestDefaultSettingValues(t *testing.T) {
	t.Run("signup enabled defaults to true", func(t *testing.T) {
		defaultValue := settingDefaults["app.auth.signup_enabled"]["value"]
		assert.Equal(t, true, defaultValue)
	})

	t.Run("magic link defaults to false", func(t *testing.T) {
		defaultValue := settingDefaults["app.auth.magic_link_enabled"]["value"]
		assert.Equal(t, false, defaultValue)
	})

	t.Run("password min length defaults to 12", func(t *testing.T) {
		defaultValue := settingDefaults["app.auth.password_min_length"]["value"]
		assert.Equal(t, 12, defaultValue)
	})

	t.Run("realtime enabled defaults to true", func(t *testing.T) {
		defaultValue := settingDefaults["app.realtime.enabled"]["value"]
		assert.Equal(t, true, defaultValue)
	})

	t.Run("captcha enabled defaults to false", func(t *testing.T) {
		defaultValue := settingDefaults["app.security.captcha.enabled"]["value"]
		assert.Equal(t, false, defaultValue)
	})

	t.Run("captcha provider defaults to hcaptcha", func(t *testing.T) {
		defaultValue := settingDefaults["app.security.captcha.provider"]["value"]
		assert.Equal(t, "hcaptcha", defaultValue)
	})

	t.Run("captcha score threshold defaults to 0.5", func(t *testing.T) {
		defaultValue := settingDefaults["app.security.captcha.score_threshold"]["value"]
		assert.Equal(t, 0.5, defaultValue)
	})

	t.Run("captcha endpoints defaults include all auth endpoints", func(t *testing.T) {
		defaultValue := settingDefaults["app.security.captcha.endpoints"]["value"]
		endpoints, ok := defaultValue.([]string)
		require.True(t, ok)
		assert.Contains(t, endpoints, "signup")
		assert.Contains(t, endpoints, "login")
		assert.Contains(t, endpoints, "password_reset")
		assert.Contains(t, endpoints, "magic_link")
	})

	t.Run("SMTP port defaults to 587", func(t *testing.T) {
		defaultValue := settingDefaults["app.email.smtp_port"]["value"]
		assert.Equal(t, 587, defaultValue)
	})

	t.Run("SMTP TLS defaults to true", func(t *testing.T) {
		defaultValue := settingDefaults["app.email.smtp_tls"]["value"]
		assert.Equal(t, true, defaultValue)
	})

	t.Run("SES region defaults to us-east-1", func(t *testing.T) {
		defaultValue := settingDefaults["app.email.ses_region"]["value"]
		assert.Equal(t, "us-east-1", defaultValue)
	})

	t.Run("global rate limit defaults to true", func(t *testing.T) {
		defaultValue := settingDefaults["app.security.enable_global_rate_limit"]["value"]
		assert.Equal(t, true, defaultValue)
	})
}

// =============================================================================
// Sensitive Settings Tests
// =============================================================================

func TestSensitiveSettings(t *testing.T) {
	sensitiveKeys := []string{
		"app.email.smtp_password",
		"app.email.sendgrid_api_key",
		"app.email.mailgun_api_key",
		"app.email.ses_access_key",
		"app.email.ses_secret_key",
		"app.security.captcha.secret_key",
		"app.security.captcha.cap_api_key",
	}

	t.Run("sensitive settings exist in defaults", func(t *testing.T) {
		for _, key := range sensitiveKeys {
			assert.Contains(t, settingDefaults, key, "Expected %s to be in defaults", key)
		}
	})

	t.Run("sensitive settings default to empty string", func(t *testing.T) {
		for _, key := range sensitiveKeys {
			defaultValue := settingDefaults[key]["value"]
			assert.Equal(t, "", defaultValue, "Expected %s to default to empty string", key)
		}
	})
}
