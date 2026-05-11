//go:build integration
// +build integration

package settings_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nimbleflux/fluxbase/internal/settings"
	"github.com/nimbleflux/fluxbase/internal/testutil"
	test "github.com/nimbleflux/fluxbase/test"
)

// =============================================================================
// Test Helpers
// =============================================================================

const testEncryptionKey = "01234567890123456789012345678901" // 32 bytes for AES-256

func createCustomSettingsService(t *testing.T, tc *testutil.IntegrationTestContext) *settings.CustomSettingsService {
	t.Helper()
	return settings.NewCustomSettingsService(tc.DB, []byte(testEncryptionKey))
}

func createTestUserForCustomSettings(t *testing.T, tc *testutil.IntegrationTestContext) uuid.UUID {
	t.Helper()
	ctx := context.Background()
	userID := uuid.New()

	// Create a test user in the database
	_, err := tc.DB.Exec(ctx, `
		INSERT INTO auth.users (id, email, password_hash, email_verified, role)
		VALUES ($1, $2 || '@test.local', $3, true, 'authenticated')
	`, userID, userID.String(), "hashed_password")
	require.NoError(t, err, "Failed to create test user")

	return userID
}

// =============================================================================
// Custom Settings Service Integration Tests
// =============================================================================

func TestCustomSettingsService_CreateSetting_Success(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "settings")
	defer tc.CleanupTestData()

	svc := createCustomSettingsService(t, tc)
	ctx := context.Background()
	createdBy := uuid.New()

	req := settings.CreateCustomSettingRequest{
		Key:         "custom.test.setting",
		Value:       map[string]interface{}{"enabled": true, "count": 42},
		ValueType:   "json",
		Description: "Test setting",
		EditableBy:  []string{"instance_admin", "admin"},
		Metadata:    map[string]interface{}{"category": "test"},
	}

	setting, err := svc.CreateSetting(ctx, req, createdBy)
	require.NoError(t, err)
	assert.NotNil(t, setting)
	assert.Equal(t, "custom.test.setting", setting.Key)
	assert.Equal(t, "json", setting.ValueType)
	assert.Equal(t, "Test setting", setting.Description)
	assert.Equal(t, &createdBy, setting.CreatedBy)
	assert.Equal(t, &createdBy, setting.UpdatedBy)
	assert.Len(t, setting.EditableBy, 2)
}

func TestCustomSettingsService_CreateSetting_DefaultValues(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "settings")
	defer tc.CleanupTestData()

	svc := createCustomSettingsService(t, tc)
	ctx := context.Background()
	createdBy := uuid.New()

	req := settings.CreateCustomSettingRequest{
		Key:   "custom.minimal",
		Value: map[string]interface{}{"value": "test"},
	}

	setting, err := svc.CreateSetting(ctx, req, createdBy)
	require.NoError(t, err)
	assert.Equal(t, "string", setting.ValueType) // Default value type
	assert.Len(t, setting.EditableBy, 1)
	assert.Contains(t, setting.EditableBy, "instance_admin")
}

func TestCustomSettingsService_CreateSetting_DuplicateKey(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "settings")
	defer tc.CleanupTestData()

	svc := createCustomSettingsService(t, tc)
	ctx := context.Background()
	createdBy := uuid.New()

	req := settings.CreateCustomSettingRequest{
		Key:   "custom.duplicate",
		Value: map[string]interface{}{"value": "test"},
	}

	// Create first setting
	_, err := svc.CreateSetting(ctx, req, createdBy)
	require.NoError(t, err)

	// Try to create duplicate
	_, err = svc.CreateSetting(ctx, req, createdBy)
	assert.Error(t, err)
	assert.ErrorIs(t, err, settings.ErrCustomSettingDuplicate)
}

func TestCustomSettingsService_CreateSetting_InvalidKey(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "settings")
	defer tc.CleanupTestData()

	svc := createCustomSettingsService(t, tc)
	ctx := context.Background()
	createdBy := uuid.New()

	req := settings.CreateCustomSettingRequest{
		Key:   "", // Empty key
		Value: map[string]interface{}{"value": "test"},
	}

	_, err := svc.CreateSetting(ctx, req, createdBy)
	assert.Error(t, err)
	assert.ErrorIs(t, err, settings.ErrCustomSettingInvalidKey)
}

func TestCustomSettingsService_CreateSetting_InvalidValueType(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "settings")
	defer tc.CleanupTestData()

	svc := createCustomSettingsService(t, tc)
	ctx := context.Background()
	createdBy := uuid.New()

	req := settings.CreateCustomSettingRequest{
		Key:       "custom.invalid",
		Value:     map[string]interface{}{"value": "test"},
		ValueType: "invalid_type",
	}

	_, err := svc.CreateSetting(ctx, req, createdBy)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid value_type")
}

func TestCustomSettingsService_GetSetting_Success(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "settings")
	defer tc.CleanupTestData()

	svc := createCustomSettingsService(t, tc)
	ctx := context.Background()
	createdBy := uuid.New()

	// Create setting first
	createReq := settings.CreateCustomSettingRequest{
		Key:         "custom.get.test",
		Value:       map[string]interface{}{"enabled": true},
		ValueType:   "json",
		Description: "Test setting",
		EditableBy:  []string{"admin"},
	}
	created, err := svc.CreateSetting(ctx, createReq, createdBy)
	require.NoError(t, err)

	// Get setting
	setting, err := svc.GetSetting(ctx, "custom.get.test")
	require.NoError(t, err)
	assert.Equal(t, created.ID, setting.ID)
	assert.Equal(t, "custom.get.test", setting.Key)
	assert.Equal(t, true, setting.Value["enabled"])
	assert.Equal(t, "Test setting", setting.Description)
}

func TestCustomSettingsService_GetSetting_NotFound(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "settings")
	defer tc.CleanupTestData()

	svc := createCustomSettingsService(t, tc)
	ctx := context.Background()

	_, err := svc.GetSetting(ctx, "custom.nonexistent")
	assert.Error(t, err)
	assert.ErrorIs(t, err, settings.ErrCustomSettingNotFound)
}

func TestCustomSettingsService_UpdateSetting_Success(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "settings")
	defer tc.CleanupTestData()

	svc := createCustomSettingsService(t, tc)
	ctx := context.Background()
	createdBy := uuid.New()

	// Create setting first
	createReq := settings.CreateCustomSettingRequest{
		Key:         "custom.update.test",
		Value:       map[string]interface{}{"enabled": false},
		ValueType:   "boolean",
		Description: "Original description",
		EditableBy:  []string{"admin"},
	}
	_, err := svc.CreateSetting(ctx, createReq, createdBy)
	require.NoError(t, err)

	// Update setting
	newDesc := "Updated description"
	updateReq := settings.UpdateCustomSettingRequest{
		Value:       map[string]interface{}{"enabled": true},
		Description: &newDesc,
		EditableBy:  []string{"admin", "instance_admin"},
	}

	updated, err := svc.UpdateSetting(ctx, "custom.update.test", updateReq, createdBy, "instance_admin")
	require.NoError(t, err)
	assert.Equal(t, true, updated.Value["enabled"])
	assert.Equal(t, "Updated description", updated.Description)
	assert.Len(t, updated.EditableBy, 2)
}

func TestCustomSettingsService_UpdateSetting_PermissionDenied(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "settings")
	defer tc.CleanupTestData()

	svc := createCustomSettingsService(t, tc)
	ctx := context.Background()
	createdBy := uuid.New()

	// Create setting with admin-only access
	createReq := settings.CreateCustomSettingRequest{
		Key:        "custom.restricted",
		Value:      map[string]interface{}{"value": "test"},
		EditableBy: []string{"admin"},
	}
	_, err := svc.CreateSetting(ctx, createReq, createdBy)
	require.NoError(t, err)

	// Try to update as non-admin user
	updateReq := settings.UpdateCustomSettingRequest{
		Value: map[string]interface{}{"value": "updated"},
	}

	_, err = svc.UpdateSetting(ctx, "custom.restricted", updateReq, createdBy, "authenticated")
	assert.Error(t, err)
	assert.ErrorIs(t, err, settings.ErrCustomSettingPermissionDenied)
}

func TestCustomSettingsService_UpdateSetting_AdminCanEdit(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "settings")
	defer tc.CleanupTestData()

	svc := createCustomSettingsService(t, tc)
	ctx := context.Background()
	createdBy := uuid.New()

	// Create setting with restricted editable_by
	createReq := settings.CreateCustomSettingRequest{
		Key:        "custom.admin.edit",
		Value:      map[string]interface{}{"value": "test"},
		EditableBy: []string{"moderator"},
	}
	_, err := svc.CreateSetting(ctx, createReq, createdBy)
	require.NoError(t, err)

	// Admin should be able to edit anyway
	updateReq := settings.UpdateCustomSettingRequest{
		Value: map[string]interface{}{"value": "updated"},
	}

	updated, err := svc.UpdateSetting(ctx, "custom.admin.edit", updateReq, createdBy, "admin")
	require.NoError(t, err)
	assert.Equal(t, "updated", updated.Value["value"])
}

func TestCustomSettingsService_DeleteSetting_Success(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "settings")
	defer tc.CleanupTestData()

	svc := createCustomSettingsService(t, tc)
	ctx := context.Background()
	createdBy := uuid.New()

	// Create setting first
	createReq := settings.CreateCustomSettingRequest{
		Key:        "custom.delete.test",
		Value:      map[string]interface{}{"value": "test"},
		EditableBy: []string{"admin"},
	}
	_, err := svc.CreateSetting(ctx, createReq, createdBy)
	require.NoError(t, err)

	// Delete setting
	err = svc.DeleteSetting(ctx, "custom.delete.test", "instance_admin")
	require.NoError(t, err)

	// Verify it's deleted
	_, err = svc.GetSetting(ctx, "custom.delete.test")
	assert.Error(t, err)
	assert.ErrorIs(t, err, settings.ErrCustomSettingNotFound)
}

func TestCustomSettingsService_DeleteSetting_PermissionDenied(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "settings")
	defer tc.CleanupTestData()

	svc := createCustomSettingsService(t, tc)
	ctx := context.Background()
	createdBy := uuid.New()

	// Create admin-only setting
	createReq := settings.CreateCustomSettingRequest{
		Key:        "custom.restricted.delete",
		Value:      map[string]interface{}{"value": "test"},
		EditableBy: []string{"admin"},
	}
	_, err := svc.CreateSetting(ctx, createReq, createdBy)
	require.NoError(t, err)

	// Try to delete as non-admin
	err = svc.DeleteSetting(ctx, "custom.restricted.delete", "authenticated")
	assert.Error(t, err)
	assert.ErrorIs(t, err, settings.ErrCustomSettingPermissionDenied)
}

func TestCustomSettingsService_ListSettings_Success(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "settings")
	defer tc.CleanupTestData()

	svc := createCustomSettingsService(t, tc)
	ctx := context.Background()
	createdBy := uuid.New()

	// Create multiple settings
	settingReqs := []settings.CreateCustomSettingRequest{
		{Key: "custom.list.one", Value: map[string]interface{}{"value": 1}},
		{Key: "custom.list.two", Value: map[string]interface{}{"value": 2}},
		{Key: "custom.list.three", Value: map[string]interface{}{"value": 3}},
	}

	for _, req := range settingReqs {
		_, err := svc.CreateSetting(ctx, req, createdBy)
		require.NoError(t, err)
	}

	// List all settings
	list, err := svc.ListSettings(ctx, "instance_admin")
	require.NoError(t, err)
	assert.Len(t, list, 3)

	// Verify they're sorted by key
	assert.Equal(t, "custom.list.one", list[0].Key)
	assert.Equal(t, "custom.list.two", list[1].Key)
	assert.Equal(t, "custom.list.three", list[2].Key)
}

// =============================================================================
// Secret Settings Integration Tests
// =============================================================================

func TestCustomSettingsService_CreateSecretSetting_System(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "settings")
	defer tc.CleanupTestData()

	svc := createCustomSettingsService(t, tc)
	ctx := context.Background()
	createdBy := uuid.New()

	createReq := settings.CreateSecretSettingRequest{
		Key:         "secret.system.api",
		Value:       "my-secret-api-key",
		Description: "System API key",
	}

	metadata, err := svc.CreateSecretSetting(ctx, createReq, nil, createdBy)
	require.NoError(t, err)
	assert.NotNil(t, metadata)
	assert.Equal(t, "secret.system.api", metadata.Key)
	assert.Equal(t, "System API key", metadata.Description)
	assert.Nil(t, metadata.UserID) // System secret has no user
	assert.Equal(t, &createdBy, metadata.CreatedBy)

	// Verify value is encrypted in database
	var encryptedValue string
	err = tc.DB.QueryRow(ctx, `
		SELECT encrypted_value FROM app.settings WHERE key = $1 AND user_id IS NULL
	`, "secret.system.api").Scan(&encryptedValue)
	require.NoError(t, err)
	assert.NotEmpty(t, encryptedValue)
	assert.NotEqual(t, "my-secret-api-key", encryptedValue) // Should be encrypted
}

func TestCustomSettingsService_CreateSecretSetting_User(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "settings")
	defer tc.CleanupTestData()

	svc := createCustomSettingsService(t, tc)
	ctx := context.Background()
	userID := createTestUserForCustomSettings(t, tc)
	createdBy := uuid.New()

	createReq := settings.CreateSecretSettingRequest{
		Key:         "secret.user.token",
		Value:       "user-secret-token",
		Description: "User's personal token",
	}

	metadata, err := svc.CreateSecretSetting(ctx, createReq, &userID, createdBy)
	require.NoError(t, err)
	assert.NotNil(t, metadata)
	assert.Equal(t, "secret.user.token", metadata.Key)
	assert.Equal(t, &userID, metadata.UserID)
}

func TestCustomSettingsService_CreateSecretSetting_Duplicate(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "settings")
	defer tc.CleanupTestData()

	svc := createCustomSettingsService(t, tc)
	ctx := context.Background()
	createdBy := uuid.New()

	createReq := settings.CreateSecretSettingRequest{
		Key:   "secret.duplicate",
		Value: "secret-value",
	}

	// Create first secret
	_, err := svc.CreateSecretSetting(ctx, createReq, nil, createdBy)
	require.NoError(t, err)

	// Try to create duplicate
	_, err = svc.CreateSecretSetting(ctx, createReq, nil, createdBy)
	assert.Error(t, err)
	assert.ErrorIs(t, err, settings.ErrCustomSettingDuplicate)
}

func TestCustomSettingsService_GetSecretSettingMetadata_Success(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "settings")
	defer tc.CleanupTestData()

	svc := createCustomSettingsService(t, tc)
	ctx := context.Background()
	createdBy := uuid.New()

	createReq := settings.CreateSecretSettingRequest{
		Key:         "secret.metadata.test",
		Value:       "secret-value",
		Description: "Test secret",
	}
	created, err := svc.CreateSecretSetting(ctx, createReq, nil, createdBy)
	require.NoError(t, err)

	// Get metadata
	metadata, err := svc.GetSecretSettingMetadata(ctx, "secret.metadata.test", nil)
	require.NoError(t, err)
	assert.Equal(t, created.ID, metadata.ID)
	assert.Equal(t, "secret.metadata.test", metadata.Key)
	assert.Equal(t, "Test secret", metadata.Description)
	assert.Nil(t, metadata.UserID)
}

func TestCustomSettingsService_GetSecretSettingMetadata_NotFound(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "settings")
	defer tc.CleanupTestData()

	svc := createCustomSettingsService(t, tc)
	ctx := context.Background()

	_, err := svc.GetSecretSettingMetadata(ctx, "secret.nonexistent", nil)
	assert.Error(t, err)
	assert.ErrorIs(t, err, settings.ErrCustomSettingNotFound)
}

func TestCustomSettingsService_UpdateSecretSetting_Value(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "settings")
	defer tc.CleanupTestData()

	svc := createCustomSettingsService(t, tc)
	ctx := context.Background()
	createdBy := uuid.New()

	createReq := settings.CreateSecretSettingRequest{
		Key:         "secret.update.value",
		Value:       "original-value",
		Description: "Original description",
	}
	_, err := svc.CreateSecretSetting(ctx, createReq, nil, createdBy)
	require.NoError(t, err)

	// Update value
	newValue := "updated-value"
	updateReq := settings.UpdateSecretSettingRequest{
		Value: &newValue,
	}

	metadata, err := svc.UpdateSecretSetting(ctx, "secret.update.value", updateReq, nil, createdBy)
	require.NoError(t, err)
	assert.Equal(t, "Original description", metadata.Description)

	// Verify encrypted value changed
	var encryptedValue string
	err = tc.DB.QueryRow(ctx, `
		SELECT encrypted_value FROM app.settings WHERE key = $1
	`, "secret.update.value").Scan(&encryptedValue)
	require.NoError(t, err)
	assert.NotEmpty(t, encryptedValue)
}

func TestCustomSettingsService_UpdateSecretSetting_Description(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "settings")
	defer tc.CleanupTestData()

	svc := createCustomSettingsService(t, tc)
	ctx := context.Background()
	createdBy := uuid.New()

	createReq := settings.CreateSecretSettingRequest{
		Key:         "secret.update.desc",
		Value:       "secret-value",
		Description: "Original description",
	}
	_, err := svc.CreateSecretSetting(ctx, createReq, nil, createdBy)
	require.NoError(t, err)

	// Update description only
	newDesc := "Updated description"
	updateReq := settings.UpdateSecretSettingRequest{
		Description: &newDesc,
	}

	metadata, err := svc.UpdateSecretSetting(ctx, "secret.update.desc", updateReq, nil, createdBy)
	require.NoError(t, err)
	assert.Equal(t, "Updated description", metadata.Description)
}

func TestCustomSettingsService_DeleteSecretSetting_Success(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "settings")
	defer tc.CleanupTestData()

	svc := createCustomSettingsService(t, tc)
	ctx := context.Background()
	createdBy := uuid.New()

	createReq := settings.CreateSecretSettingRequest{
		Key:   "secret.delete.test",
		Value: "secret-value",
	}
	_, err := svc.CreateSecretSetting(ctx, createReq, nil, createdBy)
	require.NoError(t, err)

	// Delete secret
	err = svc.DeleteSecretSetting(ctx, "secret.delete.test", nil)
	require.NoError(t, err)

	// Verify it's deleted
	_, err = svc.GetSecretSettingMetadata(ctx, "secret.delete.test", nil)
	assert.Error(t, err)
	assert.ErrorIs(t, err, settings.ErrCustomSettingNotFound)
}

func TestCustomSettingsService_DeleteSecretSetting_NotFound(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "settings")
	defer tc.CleanupTestData()

	svc := createCustomSettingsService(t, tc)
	ctx := context.Background()

	err := svc.DeleteSecretSetting(ctx, "secret.nonexistent", nil)
	assert.Error(t, err)
	assert.ErrorIs(t, err, settings.ErrCustomSettingNotFound)
}

func TestCustomSettingsService_ListSecretSettings_Success(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "settings")
	defer tc.CleanupTestData()

	svc := createCustomSettingsService(t, tc)
	ctx := context.Background()
	createdBy := uuid.New()

	// Create multiple secrets
	secretReqs := []settings.CreateSecretSettingRequest{
		{Key: "secret.list.one", Value: "value1"},
		{Key: "secret.list.two", Value: "value2"},
		{Key: "secret.list.three", Value: "value3"},
	}

	for _, createReq := range secretReqs {
		_, err := svc.CreateSecretSetting(ctx, createReq, nil, createdBy)
		require.NoError(t, err)
	}

	// List all secrets
	list, err := svc.ListSecretSettings(ctx, nil)
	require.NoError(t, err)
	assert.Len(t, list, 3)

	// Verify sorted by key
	assert.Equal(t, "secret.list.one", list[0].Key)
	assert.Equal(t, "secret.list.three", list[2].Key)
}

// =============================================================================
// User Settings Integration Tests
// =============================================================================

func TestCustomSettingsService_CreateUserSetting_Success(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "settings")
	defer tc.CleanupTestData()

	svc := createCustomSettingsService(t, tc)
	ctx := context.Background()
	userID := createTestUserForCustomSettings(t, tc)

	req := settings.CreateUserSettingRequest{
		Key:         "user.preferences.theme",
		Value:       map[string]interface{}{"theme": "dark", "fontSize": 14},
		Description: "User UI preferences",
	}

	setting, err := svc.CreateUserSetting(ctx, userID, req)
	require.NoError(t, err)
	assert.NotNil(t, setting)
	assert.Equal(t, "user.preferences.theme", setting.Key)
	assert.Equal(t, userID, setting.UserID)
	assert.Equal(t, "dark", setting.Value["theme"])
	assert.Equal(t, float64(14), setting.Value["fontSize"]) // JSON numbers are float64
}

func TestCustomSettingsService_CreateUserSetting_Duplicate(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "settings")
	defer tc.CleanupTestData()

	svc := createCustomSettingsService(t, tc)
	ctx := context.Background()
	userID := createTestUserForCustomSettings(t, tc)

	req := settings.CreateUserSettingRequest{
		Key:   "user.duplicate",
		Value: map[string]interface{}{"value": "test"},
	}

	// Create first setting
	_, err := svc.CreateUserSetting(ctx, userID, req)
	require.NoError(t, err)

	// Try to create duplicate
	_, err = svc.CreateUserSetting(ctx, userID, req)
	assert.Error(t, err)
	assert.ErrorIs(t, err, settings.ErrCustomSettingDuplicate)
}

func TestCustomSettingsService_GetUserOwnSetting_Success(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "settings")
	defer tc.CleanupTestData()

	svc := createCustomSettingsService(t, tc)
	ctx := context.Background()
	userID := createTestUserForCustomSettings(t, tc)

	createReq := settings.CreateUserSettingRequest{
		Key:         "user.get.test",
		Value:       map[string]interface{}{"enabled": true},
		Description: "Test setting",
	}
	_, err := svc.CreateUserSetting(ctx, userID, createReq)
	require.NoError(t, err)

	// Get setting
	setting, err := svc.GetUserOwnSetting(ctx, userID, "user.get.test")
	require.NoError(t, err)
	assert.Equal(t, "user.get.test", setting.Key)
	assert.Equal(t, userID, setting.UserID)
	assert.Equal(t, true, setting.Value["enabled"])
}

func TestCustomSettingsService_GetUserOwnSetting_NotFound(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "settings")
	defer tc.CleanupTestData()

	svc := createCustomSettingsService(t, tc)
	ctx := context.Background()
	userID := createTestUserForCustomSettings(t, tc)

	_, err := svc.GetUserOwnSetting(ctx, userID, "user.nonexistent")
	assert.Error(t, err)
	assert.ErrorIs(t, err, settings.ErrCustomSettingNotFound)
}

func TestCustomSettingsService_GetSystemSetting_Success(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "settings")
	defer tc.CleanupTestData()

	svc := createCustomSettingsService(t, tc)
	ctx := context.Background()
	createdBy := uuid.New()

	// Create a system setting
	createReq := settings.CreateCustomSettingRequest{
		Key:         "system.defaults.theme",
		Value:       map[string]interface{}{"theme": "light"},
		Description: "Default theme",
	}
	_, err := svc.CreateSetting(ctx, createReq, createdBy)
	require.NoError(t, err)

	// Get system setting
	setting, err := svc.GetSystemSetting(ctx, "system.defaults.theme")
	require.NoError(t, err)
	assert.Equal(t, "system.defaults.theme", setting.Key)
	assert.Equal(t, "light", setting.Value["theme"])
}

func TestCustomSettingsService_GetUserSettingWithFallback_UserSource(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "settings")
	defer tc.CleanupTestData()

	svc := createCustomSettingsService(t, tc)
	ctx := context.Background()
	userID := createTestUserForCustomSettings(t, tc)

	// Create user-specific setting
	userReq := settings.CreateUserSettingRequest{
		Key:   "user.fallback.test",
		Value: map[string]interface{}{"source": "user"},
	}
	_, err := svc.CreateUserSetting(ctx, userID, userReq)
	require.NoError(t, err)

	// Get with fallback
	setting, err := svc.GetUserSettingWithFallback(ctx, userID, "user.fallback.test")
	require.NoError(t, err)
	assert.Equal(t, "user", setting.Source)
	assert.Equal(t, "user", setting.Value["source"])
}

func TestCustomSettingsService_GetUserSettingWithFallback_SystemSource(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "settings")
	defer tc.CleanupTestData()

	svc := createCustomSettingsService(t, tc)
	ctx := context.Background()
	userID := createTestUserForCustomSettings(t, tc)
	createdBy := uuid.New()

	// Create system setting
	systemReq := settings.CreateCustomSettingRequest{
		Key:   "system.fallback.test",
		Value: map[string]interface{}{"source": "system"},
	}
	_, err := svc.CreateSetting(ctx, systemReq, createdBy)
	require.NoError(t, err)

	// User doesn't have their own setting, should fall back to system
	setting, err := svc.GetUserSettingWithFallback(ctx, userID, "system.fallback.test")
	require.NoError(t, err)
	assert.Equal(t, "system", setting.Source)
	assert.Equal(t, "system", setting.Value["source"])
}

func TestCustomSettingsService_UpdateUserSetting_Success(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "settings")
	defer tc.CleanupTestData()

	svc := createCustomSettingsService(t, tc)
	ctx := context.Background()
	userID := createTestUserForCustomSettings(t, tc)

	createReq := settings.CreateUserSettingRequest{
		Key:         "user.update.test",
		Value:       map[string]interface{}{"value": "original"},
		Description: "Original description",
	}
	_, err := svc.CreateUserSetting(ctx, userID, createReq)
	require.NoError(t, err)

	// Update setting
	newDesc := "Updated description"
	updateReq := settings.UpdateUserSettingRequest{
		Value:       map[string]interface{}{"value": "updated"},
		Description: &newDesc,
	}

	setting, err := svc.UpdateUserSetting(ctx, userID, "user.update.test", updateReq)
	require.NoError(t, err)
	assert.Equal(t, "updated", setting.Value["value"])
	assert.Equal(t, "Updated description", setting.Description)
}

func TestCustomSettingsService_UpsertUserSetting_Create(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "settings")
	defer tc.CleanupTestData()

	svc := createCustomSettingsService(t, tc)
	ctx := context.Background()
	userID := createTestUserForCustomSettings(t, tc)

	req := settings.CreateUserSettingRequest{
		Key:   "user.upsert.test",
		Value: map[string]interface{}{"value": "test"},
	}

	// Create new setting
	setting, err := svc.UpsertUserSetting(ctx, userID, req)
	require.NoError(t, err)
	assert.Equal(t, "test", setting.Value["value"])
}

func TestCustomSettingsService_UpsertUserSetting_Update(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "settings")
	defer tc.CleanupTestData()

	svc := createCustomSettingsService(t, tc)
	ctx := context.Background()
	userID := createTestUserForCustomSettings(t, tc)

	req := settings.CreateUserSettingRequest{
		Key:   "user.upsert.update",
		Value: map[string]interface{}{"value": "original"},
	}

	// Create
	_, err := svc.UpsertUserSetting(ctx, userID, req)
	require.NoError(t, err)

	// Update using upsert
	req.Value = map[string]interface{}{"value": "updated"}
	setting, err := svc.UpsertUserSetting(ctx, userID, req)
	require.NoError(t, err)
	assert.Equal(t, "updated", setting.Value["value"])
}

func TestCustomSettingsService_DeleteUserSetting_Success(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "settings")
	defer tc.CleanupTestData()

	svc := createCustomSettingsService(t, tc)
	ctx := context.Background()
	userID := createTestUserForCustomSettings(t, tc)

	createReq := settings.CreateUserSettingRequest{
		Key:   "user.delete.test",
		Value: map[string]interface{}{"value": "test"},
	}
	_, err := svc.CreateUserSetting(ctx, userID, createReq)
	require.NoError(t, err)

	// Delete setting
	err = svc.DeleteUserSetting(ctx, userID, "user.delete.test")
	require.NoError(t, err)

	// Verify it's deleted
	_, err = svc.GetUserOwnSetting(ctx, userID, "user.delete.test")
	assert.Error(t, err)
	assert.ErrorIs(t, err, settings.ErrCustomSettingNotFound)
}

func TestCustomSettingsService_DeleteUserSetting_NotFound(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "settings")
	defer tc.CleanupTestData()

	svc := createCustomSettingsService(t, tc)
	ctx := context.Background()
	userID := createTestUserForCustomSettings(t, tc)

	err := svc.DeleteUserSetting(ctx, userID, "user.nonexistent")
	assert.Error(t, err)
	assert.ErrorIs(t, err, settings.ErrCustomSettingNotFound)
}

func TestCustomSettingsService_ListUserOwnSettings_Success(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "settings")
	defer tc.CleanupTestData()

	svc := createCustomSettingsService(t, tc)
	ctx := context.Background()
	userID := createTestUserForCustomSettings(t, tc)

	// Create multiple settings for user
	userSettingReqs := []settings.CreateUserSettingRequest{
		{Key: "user.list.alpha", Value: map[string]interface{}{"value": 1}},
		{Key: "user.list.beta", Value: map[string]interface{}{"value": 2}},
		{Key: "user.list.gamma", Value: map[string]interface{}{"value": 3}},
	}

	for _, req := range userSettingReqs {
		_, err := svc.CreateUserSetting(ctx, userID, req)
		require.NoError(t, err)
	}

	// List user's settings
	list, err := svc.ListUserOwnSettings(ctx, userID)
	require.NoError(t, err)
	assert.Len(t, list, 3)

	// Verify sorted by key
	assert.Equal(t, "user.list.alpha", list[0].Key)
	assert.Equal(t, "user.list.beta", list[1].Key)
	assert.Equal(t, "user.list.gamma", list[2].Key)
}

// =============================================================================
// Transaction-Accepting Methods Tests
// =============================================================================

func TestCustomSettingsService_CreateSecretSettingWithTx_Success(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "settings")
	defer tc.CleanupTestData()

	svc := createCustomSettingsService(t, tc)
	ctx := context.Background()
	createdBy := uuid.New()

	// Begin transaction
	tx, err := tc.DB.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }()

	req := settings.CreateSecretSettingRequest{
		Key:         "secret.tx.test",
		Value:       "tx-secret-value",
		Description: "Created with transaction",
	}

	metadata, err := svc.CreateSecretSettingWithTx(ctx, tx, req, nil, createdBy)
	require.NoError(t, err)
	assert.NotNil(t, metadata)
	assert.Equal(t, "secret.tx.test", metadata.Key)
}

func TestCustomSettingsService_GetSecretSettingMetadataWithTx_Success(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "settings")
	defer tc.CleanupTestData()

	svc := createCustomSettingsService(t, tc)
	ctx := context.Background()
	createdBy := uuid.New()

	// Create secret first
	createReq := settings.CreateSecretSettingRequest{
		Key:   "secret.tx.get",
		Value: "secret-value",
	}
	_, err := svc.CreateSecretSetting(ctx, createReq, nil, createdBy)
	require.NoError(t, err)

	// Begin transaction
	tx, err := tc.DB.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }()

	// Get with transaction
	metadata, err := svc.GetSecretSettingMetadataWithTx(ctx, tx, "secret.tx.get", nil)
	require.NoError(t, err)
	assert.Equal(t, "secret.tx.get", metadata.Key)
}

func TestCustomSettingsService_UpdateSecretSettingWithTx_Success(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "settings")
	defer tc.CleanupTestData()

	svc := createCustomSettingsService(t, tc)
	ctx := context.Background()
	createdBy := uuid.New()

	// Create secret first
	createReq := settings.CreateSecretSettingRequest{
		Key:         "secret.tx.update",
		Value:       "original",
		Description: "Original",
	}
	_, err := svc.CreateSecretSetting(ctx, createReq, nil, createdBy)
	require.NoError(t, err)

	// Begin transaction
	tx, err := tc.DB.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }()

	// Update with transaction
	newDesc := "Updated via transaction"
	updateReq := settings.UpdateSecretSettingRequest{
		Description: &newDesc,
	}

	metadata, err := svc.UpdateSecretSettingWithTx(ctx, tx, "secret.tx.update", updateReq, nil, createdBy)
	require.NoError(t, err)
	assert.Equal(t, "Updated via transaction", metadata.Description)
}

func TestCustomSettingsService_DeleteSecretSettingWithTx_Success(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "settings")
	defer tc.CleanupTestData()

	svc := createCustomSettingsService(t, tc)
	ctx := context.Background()
	createdBy := uuid.New()

	// Create secret first
	createReq := settings.CreateSecretSettingRequest{
		Key:   "secret.tx.delete",
		Value: "secret-value",
	}
	_, err := svc.CreateSecretSetting(ctx, createReq, nil, createdBy)
	require.NoError(t, err)

	// Begin transaction
	tx, err := tc.DB.BeginTx(ctx)
	require.NoError(t, err)

	// Delete with transaction
	err = svc.DeleteSecretSettingWithTx(ctx, tx, "secret.tx.delete", nil)
	require.NoError(t, err)

	// Rollback to verify deletion was within transaction
	_ = tx.Rollback(ctx)

	// Setting should still exist (transaction was rolled back)
	_, err = svc.GetSecretSettingMetadata(ctx, "secret.tx.delete", nil)
	assert.NoError(t, err)
}

func TestCustomSettingsService_ListSecretSettingsWithTx_Success(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "settings")
	defer tc.CleanupTestData()

	svc := createCustomSettingsService(t, tc)
	ctx := context.Background()
	createdBy := uuid.New()

	// Create secrets with unique keys to avoid conflicts
	uniqueSuffix := uuid.New().String()
	secretReqs := []settings.CreateSecretSettingRequest{
		{Key: fmt.Sprintf("secret.tx.list.%s.1", uniqueSuffix), Value: "value1"},
		{Key: fmt.Sprintf("secret.tx.list.%s.2", uniqueSuffix), Value: "value2"},
	}
	for _, createReq := range secretReqs {
		_, err := svc.CreateSecretSetting(ctx, createReq, nil, createdBy)
		require.NoError(t, err)
	}

	// Begin transaction
	tx, err := tc.DB.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }()

	// List with transaction - Note: will include all existing secrets, not just our test data
	list, err := svc.ListSecretSettingsWithTx(ctx, tx, nil)
	require.NoError(t, err)
	// Check that our test secrets are in the list (there may be more from other tests)
	listKeys := make(map[string]bool)
	for _, s := range list {
		listKeys[s.Key] = true
	}
	assert.True(t, listKeys[secretReqs[0].Key], "First test secret not found")
	assert.True(t, listKeys[secretReqs[1].Key], "Second test secret not found")
}

func TestCustomSettingsService_UpsertUserSettingWithTx_Create(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "settings")
	defer tc.CleanupTestData()

	svc := createCustomSettingsService(t, tc)
	ctx := context.Background()
	userID := createTestUserForCustomSettings(t, tc) // Create actual user in database

	// Begin transaction
	tx, err := tc.DB.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }()

	req := settings.CreateUserSettingRequest{
		Key:   "user.tx.upsert",
		Value: map[string]interface{}{"value": "test"},
	}

	setting, err := svc.UpsertUserSettingWithTx(ctx, tx, userID, req)
	require.NoError(t, err)
	assert.Equal(t, "test", setting.Value["value"])
}

func TestCustomSettingsService_GetUserSettingWithFallbackWithTx_UserSource(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "settings")
	defer tc.CleanupTestData()

	svc := createCustomSettingsService(t, tc)
	ctx := context.Background()
	userID := createTestUserForCustomSettings(t, tc) // Create actual user in database

	// Create user setting
	req := settings.CreateUserSettingRequest{
		Key:   "user.tx.fallback",
		Value: map[string]interface{}{"source": "user"},
	}
	_, err := svc.CreateUserSetting(ctx, userID, req)
	require.NoError(t, err)

	// Begin transaction
	tx, err := tc.DB.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }()

	// Get with fallback using transaction
	setting, err := svc.GetUserSettingWithFallbackWithTx(ctx, tx, userID, "user.tx.fallback")
	require.NoError(t, err)
	assert.Equal(t, "user", setting.Source)
}

// TestMain is the entry point for running tests in this package when using go test
// It ensures that the shared test context is properly cleaned up after all tests run,
// which prevents goroutine leaks from Fiber's middleware (CSRF, rate limiter, etc.)
func TestMain(m *testing.M) {
	// Initialize shared test context before running any tests
	test.InitSharedTestContext()

	// Run all tests
	code := m.Run()

	// Clean up shared test context after all tests complete
	// This shuts down the server and stops all background goroutines
	test.CleanupSharedTestContext()

	// Exit with the test result code
	os.Exit(code)
}
