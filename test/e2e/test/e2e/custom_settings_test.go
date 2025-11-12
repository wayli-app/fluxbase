package e2e

import (
	"fmt"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/require"
	"github.com/wayli-app/fluxbase/test"
)

// setupCustomSettingsTest prepares the test context for custom settings tests
func setupCustomSettingsTest(t *testing.T) (*test.TestContext, string) {
	tc := test.NewTestContext(t)
	tc.EnsureAuthSchema()

	// Clean custom settings and auth tables before each test
	tc.ExecuteSQL("TRUNCATE TABLE dashboard.custom_settings CASCADE")
	tc.ExecuteSQL("TRUNCATE TABLE dashboard.users CASCADE")
	tc.ExecuteSQL("TRUNCATE TABLE auth.users CASCADE")

	// Create dashboard admin user
	timestamp := time.Now().UnixNano()
	email := fmt.Sprintf("admin-%d@test.com", timestamp)
	password := "adminpass123456"
	_, token := tc.CreateDashboardAdminUser(email, password)

	return tc, token
}

// TestCustomSettingsCreateAsAdmin tests creating a custom setting as admin
func TestCustomSettingsCreateAsAdmin(t *testing.T) {
	tc, token := setupCustomSettingsTest(t)
	defer tc.Close()

	// Create a custom setting
	resp := tc.NewRequest("POST", "/api/v1/admin/settings/custom").
		WithAuth(token).
		WithBody(map[string]interface{}{
			"key":         "custom.test.feature_flag",
			"value":       map[string]interface{}{"enabled": true, "threshold": float64(10)},
			"value_type":  "json",
			"description": "Test feature flag",
			"editable_by": []string{"dashboard_admin", "admin"},
		}).
		Send().
		AssertStatus(fiber.StatusCreated)

	// Verify response structure
	var result map[string]interface{}
	resp.JSON(&result)

	require.Equal(t, "custom.test.feature_flag", result["key"])
	require.Equal(t, "json", result["value_type"])
	require.Equal(t, "Test feature flag", result["description"])
	require.NotNil(t, result["id"])
	require.NotNil(t, result["created_at"])
	require.NotNil(t, result["value"])

	// Verify value content
	value := result["value"].(map[string]interface{})
	require.Equal(t, true, value["enabled"])
	require.Equal(t, float64(10), value["threshold"])

	// Verify in database
	settings := tc.QuerySQL("SELECT key, value_type, description FROM dashboard.custom_settings WHERE key = $1", "custom.test.feature_flag")
	require.Len(t, settings, 1)
	require.Equal(t, "custom.test.feature_flag", settings[0]["key"])
	require.Equal(t, "json", settings[0]["value_type"])
}

// TestCustomSettingsCreateWithDefaults tests creating a setting with minimal fields
func TestCustomSettingsCreateWithDefaults(t *testing.T) {
	tc, token := setupCustomSettingsTest(t)
	defer tc.Close()

	// Create with minimal fields
	resp := tc.NewRequest("POST", "/api/v1/admin/settings/custom").
		WithAuth(token).
		WithBody(map[string]interface{}{
			"key":   "custom.test.minimal",
			"value": map[string]interface{}{"setting": "value"},
		}).
		Send().
		AssertStatus(fiber.StatusCreated)

	var result map[string]interface{}
	resp.JSON(&result)

	// Should have default value_type and editable_by
	require.Equal(t, "string", result["value_type"])
	require.Equal(t, []interface{}{"dashboard_admin"}, result["editable_by"])
}

// TestCustomSettingsList tests listing all custom settings
func TestCustomSettingsList(t *testing.T) {
	tc, token := setupCustomSettingsTest(t)
	defer tc.Close()

	// Create multiple settings
	keys := []string{"custom.test.one", "custom.test.two", "custom.test.three"}
	for _, key := range keys {
		tc.NewRequest("POST", "/api/v1/admin/settings/custom").
			WithAuth(token).
			WithBody(map[string]interface{}{
				"key":   key,
				"value": map[string]interface{}{"index": key},
			}).
			Send().
			AssertStatus(fiber.StatusCreated)
	}

	// List all settings
	resp := tc.NewRequest("GET", "/api/v1/admin/settings/custom").
		WithAuth(token).
		Send().
		AssertStatus(fiber.StatusOK)

	var results []map[string]interface{}
	resp.JSON(&results)

	require.GreaterOrEqual(t, len(results), 3)

	// Verify our keys are present
	foundKeys := make(map[string]bool)
	for _, setting := range results {
		foundKeys[setting["key"].(string)] = true
	}
	for _, key := range keys {
		require.True(t, foundKeys[key], "Should find key: "+key)
	}
}

// TestCustomSettingsGet tests getting a specific custom setting
func TestCustomSettingsGet(t *testing.T) {
	tc, token := setupCustomSettingsTest(t)
	defer tc.Close()

	// Create a setting
	key := "custom.test.get"
	tc.NewRequest("POST", "/api/v1/admin/settings/custom").
		WithAuth(token).
		WithBody(map[string]interface{}{
			"key":         key,
			"value":       map[string]interface{}{"test": "data"},
			"description": "Get test",
		}).
		Send().
		AssertStatus(fiber.StatusCreated)

	// Get the setting
	resp := tc.NewRequest("GET", "/api/v1/admin/settings/custom/"+key).
		WithAuth(token).
		Send().
		AssertStatus(fiber.StatusOK)

	var result map[string]interface{}
	resp.JSON(&result)

	require.Equal(t, key, result["key"])
	require.Equal(t, "Get test", result["description"])
	value := result["value"].(map[string]interface{})
	require.Equal(t, "data", value["test"])
}

// TestCustomSettingsGetNotFound tests getting a non-existent setting
func TestCustomSettingsGetNotFound(t *testing.T) {
	tc, token := setupCustomSettingsTest(t)
	defer tc.Close()

	// Try to get non-existent setting
	tc.NewRequest("GET", "/api/v1/admin/settings/custom/nonexistent.key").
		WithAuth(token).
		Send().
		AssertStatus(fiber.StatusNotFound)
}

// TestCustomSettingsUpdate tests updating a custom setting
func TestCustomSettingsUpdate(t *testing.T) {
	tc, token := setupCustomSettingsTest(t)
	defer tc.Close()

	// Create a setting
	key := "custom.test.update"
	tc.NewRequest("POST", "/api/v1/admin/settings/custom").
		WithAuth(token).
		WithBody(map[string]interface{}{
			"key":   key,
			"value": map[string]interface{}{"count": float64(1)},
		}).
		Send().
		AssertStatus(fiber.StatusCreated)

	// Update the setting
	resp := tc.NewRequest("PUT", "/api/v1/admin/settings/custom/"+key).
		WithAuth(token).
		WithBody(map[string]interface{}{
			"value": map[string]interface{}{"count": float64(2), "updated": true},
		}).
		Send().
		AssertStatus(fiber.StatusOK)

	var result map[string]interface{}
	resp.JSON(&result)

	value := result["value"].(map[string]interface{})
	require.Equal(t, float64(2), value["count"])
	require.Equal(t, true, value["updated"])
}

// TestCustomSettingsUpdateWithDescription tests updating description and editable_by
func TestCustomSettingsUpdateWithDescription(t *testing.T) {
	tc, token := setupCustomSettingsTest(t)
	defer tc.Close()

	// Create a setting
	key := "custom.test.update_desc"
	tc.NewRequest("POST", "/api/v1/admin/settings/custom").
		WithAuth(token).
		WithBody(map[string]interface{}{
			"key":         key,
			"value":       map[string]interface{}{"test": "value"},
			"description": "Original description",
		}).
		Send().
		AssertStatus(fiber.StatusCreated)

	// Update with new description and editable_by
	resp := tc.NewRequest("PUT", "/api/v1/admin/settings/custom/"+key).
		WithAuth(token).
		WithBody(map[string]interface{}{
			"value":       map[string]interface{}{"test": "updated"},
			"description": "Updated description",
			"editable_by": []string{"dashboard_admin", "admin"},
		}).
		Send().
		AssertStatus(fiber.StatusOK)

	var result map[string]interface{}
	resp.JSON(&result)

	require.Equal(t, "Updated description", result["description"])
	require.Equal(t, []interface{}{"dashboard_admin", "admin"}, result["editable_by"])
}

// TestCustomSettingsDelete tests deleting a custom setting
func TestCustomSettingsDelete(t *testing.T) {
	tc, token := setupCustomSettingsTest(t)
	defer tc.Close()

	// Create a setting
	key := "custom.test.delete"
	tc.NewRequest("POST", "/api/v1/admin/settings/custom").
		WithAuth(token).
		WithBody(map[string]interface{}{
			"key":   key,
			"value": map[string]interface{}{"test": "data"},
		}).
		Send().
		AssertStatus(fiber.StatusCreated)

	// Delete the setting
	tc.NewRequest("DELETE", "/api/v1/admin/settings/custom/"+key).
		WithAuth(token).
		Send().
		AssertStatus(fiber.StatusNoContent)

	// Verify it's gone
	tc.NewRequest("GET", "/api/v1/admin/settings/custom/"+key).
		WithAuth(token).
		Send().
		AssertStatus(fiber.StatusNotFound)

	// Verify in database
	settings := tc.QuerySQL("SELECT * FROM dashboard.custom_settings WHERE key = $1", key)
	require.Len(t, settings, 0)
}

// TestCustomSettingsDuplicateKeyFails tests that duplicate keys are rejected
func TestCustomSettingsDuplicateKeyFails(t *testing.T) {
	tc, token := setupCustomSettingsTest(t)
	defer tc.Close()

	key := "custom.test.duplicate"

	// Create first setting
	tc.NewRequest("POST", "/api/v1/admin/settings/custom").
		WithAuth(token).
		WithBody(map[string]interface{}{
			"key":   key,
			"value": map[string]interface{}{"test": "first"},
		}).
		Send().
		AssertStatus(fiber.StatusCreated)

	// Try to create with same key
	resp := tc.NewRequest("POST", "/api/v1/admin/settings/custom").
		WithAuth(token).
		WithBody(map[string]interface{}{
			"key":   key,
			"value": map[string]interface{}{"test": "second"},
		}).
		Send().
		AssertStatus(fiber.StatusConflict)

	var result map[string]interface{}
	resp.JSON(&result)
	require.Contains(t, result["error"], "already exists")
}

// TestCustomSettingsUnauthorized tests that unauthenticated requests fail
func TestCustomSettingsUnauthorized(t *testing.T) {
	tc, _ := setupCustomSettingsTest(t)
	defer tc.Close()

	// Try to create without auth
	tc.NewRequest("POST", "/api/v1/admin/settings/custom").
		WithBody(map[string]interface{}{
			"key":   "custom.test.unauth",
			"value": map[string]interface{}{"test": "value"},
		}).
		Send().
		AssertStatus(fiber.StatusUnauthorized)

	// Try to list without auth
	tc.NewRequest("GET", "/api/v1/admin/settings/custom").
		Send().
		AssertStatus(fiber.StatusUnauthorized)
}

// TestCustomSettingsEmptyKeyFails tests that empty key is rejected
func TestCustomSettingsEmptyKeyFails(t *testing.T) {
	tc, token := setupCustomSettingsTest(t)
	defer tc.Close()

	// Try to create with empty key
	tc.NewRequest("POST", "/api/v1/admin/settings/custom").
		WithAuth(token).
		WithBody(map[string]interface{}{
			"key":   "",
			"value": map[string]interface{}{"test": "value"},
		}).
		Send().
		AssertStatus(fiber.StatusBadRequest)
}

// TestCustomSettingsEmptyValueFails tests that empty value is rejected
func TestCustomSettingsEmptyValueFails(t *testing.T) {
	tc, token := setupCustomSettingsTest(t)
	defer tc.Close()

	// Try to create with nil value
	tc.NewRequest("POST", "/api/v1/admin/settings/custom").
		WithAuth(token).
		WithBody(map[string]interface{}{
			"key": "custom.test.novalue",
		}).
		Send().
		AssertStatus(fiber.StatusBadRequest)
}

// TestCustomSettingsInvalidValueType tests that invalid value_type is rejected
func TestCustomSettingsInvalidValueType(t *testing.T) {
	tc, token := setupCustomSettingsTest(t)
	defer tc.Close()

	// Try to create with invalid value_type
	tc.NewRequest("POST", "/api/v1/admin/settings/custom").
		WithAuth(token).
		WithBody(map[string]interface{}{
			"key":        "custom.test.badtype",
			"value":      map[string]interface{}{"test": "value"},
			"value_type": "invalid_type",
		}).
		Send().
		AssertStatus(fiber.StatusInternalServerError)
}

// TestCustomSettingsMetadata tests storing metadata with a setting
func TestCustomSettingsMetadata(t *testing.T) {
	tc, token := setupCustomSettingsTest(t)
	defer tc.Close()

	// Create with metadata
	resp := tc.NewRequest("POST", "/api/v1/admin/settings/custom").
		WithAuth(token).
		WithBody(map[string]interface{}{
			"key":   "custom.test.metadata",
			"value": map[string]interface{}{"enabled": true},
			"metadata": map[string]interface{}{
				"category": "features",
				"tags":     []string{"beta", "experimental"},
				"owner":    "platform-team",
			},
		}).
		Send().
		AssertStatus(fiber.StatusCreated)

	var result map[string]interface{}
	resp.JSON(&result)

	metadata := result["metadata"].(map[string]interface{})
	require.Equal(t, "features", metadata["category"])
	require.Equal(t, "platform-team", metadata["owner"])
}

// TestCustomSettingsComplexValues tests storing various JSON structures
func TestCustomSettingsComplexValues(t *testing.T) {
	tc, token := setupCustomSettingsTest(t)
	defer tc.Close()

	testCases := []struct {
		name      string
		key       string
		value     interface{}
		valueType string
	}{
		{
			name:      "nested object",
			key:       "custom.test.nested",
			value:     map[string]interface{}{"level1": map[string]interface{}{"level2": map[string]interface{}{"level3": "deep"}}},
			valueType: "json",
		},
		{
			name:      "array of objects",
			key:       "custom.test.array",
			value:     map[string]interface{}{"items": []interface{}{map[string]interface{}{"id": float64(1)}, map[string]interface{}{"id": float64(2)}}},
			valueType: "json",
		},
		{
			name:      "boolean value",
			key:       "custom.test.boolean",
			value:     map[string]interface{}{"enabled": true, "disabled": false},
			valueType: "boolean",
		},
		{
			name:      "numeric value",
			key:       "custom.test.number",
			value:     map[string]interface{}{"count": float64(42), "percentage": 99.9},
			valueType: "number",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			resp := tc.NewRequest("POST", "/api/v1/admin/settings/custom").
				WithAuth(token).
				WithBody(map[string]interface{}{
					"key":        testCase.key,
					"value":      testCase.value,
					"value_type": testCase.valueType,
				}).
				Send().
				AssertStatus(fiber.StatusCreated)

			var result map[string]interface{}
			resp.JSON(&result)

			require.Equal(t, testCase.key, result["key"])
			require.Equal(t, testCase.valueType, result["value_type"])
			require.NotNil(t, result["value"])
		})
	}
}
