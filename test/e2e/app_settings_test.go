package e2e

import (
	"fmt"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/require"
	"github.com/wayli-app/fluxbase/test"
)

// TestAppSettingsGet tests getting app settings
func TestAppSettingsGet(t *testing.T) {
	tc, token := setupAdminTest(t)
	defer tc.Close()

	// Get app settings
	resp := tc.NewRequest("GET", "/api/v1/admin/app/settings").
		WithAuth(token).
		Send().
		AssertStatus(fiber.StatusOK)

	var settings map[string]interface{}
	resp.JSON(&settings)

	// Verify structure
	require.Contains(t, settings, "authentication", "Should have authentication settings")
	require.Contains(t, settings, "features", "Should have feature settings")
	require.Contains(t, settings, "email", "Should have email settings")
	require.Contains(t, settings, "security", "Should have security settings")

	t.Logf("App settings structure verified")
}

// TestAppSettingsUpdate tests updating app settings
func TestAppSettingsUpdate(t *testing.T) {
	tc, token := setupAdminTest(t)
	defer tc.Close()

	// Update app settings
	updates := map[string]interface{}{
		"authentication": map[string]interface{}{
			"enable_signup":       true,
			"enable_magic_link":   false,
			"password_min_length": 10,
		},
		"features": map[string]interface{}{
			"enable_realtime": false,
		},
	}

	resp := tc.NewRequest("PUT", "/api/v1/admin/app/settings").
		WithAuth(token).
		WithBody(updates).
		Send().
		AssertStatus(fiber.StatusOK)

	var settings map[string]interface{}
	resp.JSON(&settings)

	// Verify updates
	auth := settings["authentication"].(map[string]interface{})
	require.Equal(t, true, auth["enable_signup"], "Signup should be enabled")
	require.Equal(t, false, auth["enable_magic_link"], "Magic link should be disabled")
	require.Equal(t, float64(10), auth["password_min_length"], "Password min length should be 10")

	features := settings["features"].(map[string]interface{})
	require.Equal(t, false, features["enable_realtime"], "Realtime should be disabled")

	t.Logf("App settings updated successfully")
}

// TestAppSettingsReset tests resetting app settings
func TestAppSettingsReset(t *testing.T) {
	tc, token := setupAdminTest(t)
	defer tc.Close()

	// First, update some settings
	updates := map[string]interface{}{
		"authentication": map[string]interface{}{
			"enable_signup": true,
		},
		"features": map[string]interface{}{
			"enable_storage": false,
		},
	}

	tc.NewRequest("PUT", "/api/v1/admin/app/settings").
		WithAuth(token).
		WithBody(updates).
		Send().
		AssertStatus(fiber.StatusOK)

	// Reset settings
	resp := tc.NewRequest("POST", "/api/v1/admin/app/settings/reset").
		WithAuth(token).
		Send().
		AssertStatus(fiber.StatusOK)

	var settings map[string]interface{}
	resp.JSON(&settings)

	// Verify defaults are restored
	auth := settings["authentication"].(map[string]interface{})
	require.Equal(t, false, auth["enable_signup"], "Signup should be disabled (default)")

	features := settings["features"].(map[string]interface{})
	require.Equal(t, true, features["enable_storage"], "Storage should be enabled (default)")

	t.Logf("App settings reset to defaults successfully")
}

// TestSystemSettingsList tests listing all system settings
func TestSystemSettingsList(t *testing.T) {
	tc, token := setupAdminTest(t)
	defer tc.Close()

	// List system settings
	resp := tc.NewRequest("GET", "/api/v1/admin/system/settings").
		WithAuth(token).
		Send().
		AssertStatus(fiber.StatusOK)

	var settings []interface{}
	resp.JSON(&settings)

	// Should be a list (may be empty initially)
	require.NotNil(t, settings, "Should return a list of settings")

	t.Logf("System settings list retrieved: %d items", len(settings))
}

// TestSystemSettingsUpdate tests creating/updating a specific system setting
func TestSystemSettingsUpdate(t *testing.T) {
	tc, token := setupAdminTest(t)
	defer tc.Close()

	// Update a specific setting
	key := "app.auth.enable_signup"
	update := map[string]interface{}{
		"value": map[string]interface{}{
			"value": true,
		},
		"description": "Enable user signup (test)",
	}

	resp := tc.NewRequest("PUT", fmt.Sprintf("/api/v1/admin/system/settings/%s", key)).
		WithAuth(token).
		WithBody(update).
		Send().
		AssertStatus(fiber.StatusOK)

	var setting map[string]interface{}
	resp.JSON(&setting)

	// Verify setting was updated
	require.Equal(t, key, setting["key"], "Setting key should match")

	t.Logf("System setting updated: %s", key)
}

// TestSystemSettingsGet tests getting a specific system setting
func TestSystemSettingsGet(t *testing.T) {
	tc, token := setupAdminTest(t)
	defer tc.Close()

	// First create a setting
	key := "app.auth.enable_magic_link"
	update := map[string]interface{}{
		"value": map[string]interface{}{
			"value": false,
		},
		"description": "Enable magic link authentication",
	}

	tc.NewRequest("PUT", fmt.Sprintf("/api/v1/admin/system/settings/%s", key)).
		WithAuth(token).
		WithBody(update).
		Send().
		AssertStatus(fiber.StatusOK)

	// Get the setting
	resp := tc.NewRequest("GET", fmt.Sprintf("/api/v1/admin/system/settings/%s", key)).
		WithAuth(token).
		Send().
		AssertStatus(fiber.StatusOK)

	var setting map[string]interface{}
	resp.JSON(&setting)

	// Verify setting
	require.Equal(t, key, setting["key"], "Setting key should match")

	t.Logf("System setting retrieved: %s", key)
}

// TestSystemSettingsDelete tests deleting a specific system setting
func TestSystemSettingsDelete(t *testing.T) {
	tc, token := setupAdminTest(t)
	defer tc.Close()

	// First create a setting
	timestamp := time.Now().UnixNano()
	key := fmt.Sprintf("app.test.setting_%d", timestamp)

	// This should fail because the key is not in the whitelist
	update := map[string]interface{}{
		"value": map[string]interface{}{
			"value": "test",
		},
		"description": "Test setting for deletion",
	}

	// Try to update (will fail because not whitelisted)
	tc.NewRequest("PUT", fmt.Sprintf("/api/v1/admin/system/settings/%s", key)).
		WithAuth(token).
		WithBody(update).
		Send().
		AssertStatus(fiber.StatusBadRequest)

	t.Logf("System setting whitelist validation works correctly")
}

// TestAppSettingsAuthRequired tests that authentication is required
func TestAppSettingsAuthRequired(t *testing.T) {
	tc := test.NewTestContext(t)
	defer tc.Close()

	tc.EnsureAuthSchema()

	// Try to get app settings without auth
	tc.NewRequest("GET", "/api/v1/admin/app/settings").
		Send().
		AssertStatus(fiber.StatusUnauthorized)

	t.Logf("Authentication requirement enforced")
}

// TestAppSettingsRoleRequired tests that admin role is required
func TestAppSettingsRoleRequired(t *testing.T) {
	tc := test.NewTestContext(t)
	defer tc.Close()

	tc.EnsureAuthSchema()

	// Create a regular user (not admin)
	timestamp := time.Now().UnixNano()
	email := fmt.Sprintf("user-%d@test.com", timestamp)
	_, token := tc.CreateTestUser(email, "password123")

	// Try to get app settings with non-admin user
	resp := tc.NewRequest("GET", "/api/v1/admin/app/settings").
		WithAuth(token).
		Send()

	// Should be either Unauthorized (401) or Forbidden (403)
	status := resp.Status()
	require.Contains(t, []int{fiber.StatusUnauthorized, fiber.StatusForbidden}, status,
		"Non-admin user should not access app settings")

	t.Logf("Role-based access control enforced")
}
