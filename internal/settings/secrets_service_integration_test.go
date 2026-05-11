//go:build integration
// +build integration

package settings_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nimbleflux/fluxbase/internal/settings"
	"github.com/nimbleflux/fluxbase/internal/testutil"
)

// =============================================================================
// SecretsService Integration Tests
// =============================================================================

func createSecretsService(t *testing.T, tc *testutil.IntegrationTestContext) *settings.SecretsService {
	t.Helper()
	return settings.NewSecretsService(tc.DB, []byte(testEncryptionKey))
}

func createTestUser(t *testing.T, tc *testutil.IntegrationTestContext) uuid.UUID {
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

func setupSystemSecret(t *testing.T, tc *testutil.IntegrationTestContext, key, value string) {
	t.Helper()
	svc := createCustomSettingsService(t, tc)
	ctx := context.Background()
	createdBy := uuid.New()

	req := settings.CreateSecretSettingRequest{
		Key:         key,
		Value:       value,
		Description: "Test secret",
	}
	_, err := svc.CreateSecretSetting(ctx, req, nil, createdBy)
	require.NoError(t, err)
}

func setupUserSecret(t *testing.T, tc *testutil.IntegrationTestContext, userID uuid.UUID, key, value string) {
	t.Helper()
	svc := createCustomSettingsService(t, tc)
	ctx := context.Background()
	createdBy := uuid.New()

	req := settings.CreateSecretSettingRequest{
		Key:         key,
		Value:       value,
		Description: "Test user secret",
	}
	_, err := svc.CreateSecretSetting(ctx, req, &userID, createdBy)
	require.NoError(t, err)
}

// =============================================================================
// GetSystemSecret Tests
// =============================================================================

func TestSecretsService_GetSystemSecret_Success(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "settings")
	defer tc.CleanupTestData()

	setupSystemSecret(t, tc, "secret.system.test", "my-secret-value")

	svc := createSecretsService(t, tc)
	ctx := context.Background()

	decrypted, err := svc.GetSystemSecret(ctx, "secret.system.test")
	require.NoError(t, err)
	assert.Equal(t, "my-secret-value", decrypted)
}

func TestSecretsService_GetSystemSecret_NotFound(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "settings")
	defer tc.CleanupTestData()

	svc := createSecretsService(t, tc)
	ctx := context.Background()

	_, err := svc.GetSystemSecret(ctx, "secret.nonexistent")
	assert.Error(t, err)
	assert.ErrorIs(t, err, settings.ErrSecretNotFound)
}

func TestSecretsService_GetSystemSecret_ComplexValue(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "settings")
	defer tc.CleanupTestData()

	complexValue := "key\nwith\nnewlines\tand\ttabs\"quotes'"
	setupSystemSecret(t, tc, "secret.system.complex", complexValue)

	svc := createSecretsService(t, tc)
	ctx := context.Background()

	decrypted, err := svc.GetSystemSecret(ctx, "secret.system.complex")
	require.NoError(t, err)
	assert.Equal(t, complexValue, decrypted)
}

func TestSecretsService_GetSystemSecret_EmptyValue(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "settings")
	defer tc.CleanupTestData()

	setupSystemSecret(t, tc, "secret.system.empty", "")

	svc := createSecretsService(t, tc)
	ctx := context.Background()

	decrypted, err := svc.GetSystemSecret(ctx, "secret.system.empty")
	require.NoError(t, err)
	assert.Equal(t, "", decrypted)
}

func TestSecretsService_GetSystemSecret_UnicodeValue(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "settings")
	defer tc.CleanupTestData()

	unicodeValue := "secret 世界 🌍 全"
	setupSystemSecret(t, tc, "secret.system.unicode", unicodeValue)

	svc := createSecretsService(t, tc)
	ctx := context.Background()

	decrypted, err := svc.GetSystemSecret(ctx, "secret.system.unicode")
	require.NoError(t, err)
	assert.Equal(t, unicodeValue, decrypted)
}

// =============================================================================
// GetUserSecret Tests
// =============================================================================

func TestSecretsService_GetUserSecret_Success(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "settings")
	defer tc.CleanupTestData()

	userID := createTestUser(t, tc)
	setupUserSecret(t, tc, userID, "secret.user.test", "user-secret-value")

	svc := createSecretsService(t, tc)
	ctx := context.Background()

	decrypted, err := svc.GetUserSecret(ctx, userID, "secret.user.test")
	require.NoError(t, err)
	assert.Equal(t, "user-secret-value", decrypted)
}

func TestSecretsService_GetUserSecret_NotFound(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "settings")
	defer tc.CleanupTestData()

	userID := createTestUser(t, tc)

	svc := createSecretsService(t, tc)
	ctx := context.Background()

	_, err := svc.GetUserSecret(ctx, userID, "secret.user.nonexistent")
	assert.Error(t, err)
	assert.ErrorIs(t, err, settings.ErrSecretNotFound)
}

func TestSecretsService_GetUserSecret_WrongUser(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "settings")
	defer tc.CleanupTestData()

	// Create actual users in the database
	userID1 := createTestUser(t, tc)
	userID2 := createTestUser(t, tc)
	setupUserSecret(t, tc, userID1, "secret.user.isolated", "secret-value")

	svc := createSecretsService(t, tc)
	ctx := context.Background()

	// User 2 should not be able to access user 1's secret
	_, err := svc.GetUserSecret(ctx, userID2, "secret.user.isolated")
	assert.Error(t, err)
	assert.ErrorIs(t, err, settings.ErrSecretNotFound)
}

func TestSecretsService_GetUserSecret_LongValue(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "settings")
	defer tc.CleanupTestData()

	userID := createTestUser(t, tc)
	longValue := string(make([]byte, 10000))
	for i := range longValue {
		longValue = longValue[:i] + "a" + longValue[i+1:]
	}
	setupUserSecret(t, tc, userID, "secret.user.long", longValue)

	svc := createSecretsService(t, tc)
	ctx := context.Background()

	decrypted, err := svc.GetUserSecret(ctx, userID, "secret.user.long")
	require.NoError(t, err)
	assert.Len(t, decrypted, 10000)
}

// =============================================================================
// GetSystemSecrets Tests
// =============================================================================

func TestSecretsService_GetSystemSecrets_Multiple(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "settings")
	defer tc.CleanupTestData()

	// Setup multiple system secrets
	setupSystemSecret(t, tc, "secret.system.1", "value1")
	setupSystemSecret(t, tc, "secret.system.2", "value2")
	setupSystemSecret(t, tc, "secret.system.3", "value3")

	svc := createSecretsService(t, tc)
	ctx := context.Background()

	secrets, err := svc.GetSystemSecrets(ctx)
	require.NoError(t, err)
	assert.Len(t, secrets, 3)
	assert.Equal(t, "value1", secrets["secret.system.1"])
	assert.Equal(t, "value2", secrets["secret.system.2"])
	assert.Equal(t, "value3", secrets["secret.system.3"])
}

func TestSecretsService_GetSystemSecrets_Empty(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "settings")
	defer tc.CleanupTestData()

	svc := createSecretsService(t, tc)
	ctx := context.Background()

	secrets, err := svc.GetSystemSecrets(ctx)
	require.NoError(t, err)
	assert.Empty(t, secrets)
}

func TestSecretsService_GetSystemSecrets_WithNonSecretSettings(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "settings")
	defer tc.CleanupTestData()

	// Create a non-secret setting
	customSvc := createCustomSettingsService(t, tc)
	ctx := context.Background()
	createdBy := uuid.New()

	nonSecretReq := settings.CreateCustomSettingRequest{
		Key:   "system.nonsecret",
		Value: map[string]interface{}{"value": "not-secret"},
	}
	_, err := customSvc.CreateSetting(ctx, nonSecretReq, createdBy)
	require.NoError(t, err)

	// Create a secret
	setupSystemSecret(t, tc, "secret.system.mixed", "secret-value")

	// Get system secrets should only return secrets
	secretsSvc := createSecretsService(t, tc)
	secrets, err := secretsSvc.GetSystemSecrets(ctx)
	require.NoError(t, err)
	assert.Len(t, secrets, 1)
	assert.Equal(t, "secret-value", secrets["secret.system.mixed"])
	assert.NotContains(t, secrets, "system.nonsecret")
}

func TestSecretsService_GetUserSecrets_Multiple(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "settings")
	defer tc.CleanupTestData()

	userID := createTestUser(t, tc)
	setupUserSecret(t, tc, userID, "secret.user.1", "value1")
	setupUserSecret(t, tc, userID, "secret.user.2", "value2")
	setupUserSecret(t, tc, userID, "secret.user.3", "value3")

	svc := createSecretsService(t, tc)
	ctx := context.Background()

	secrets, err := svc.GetUserSecrets(ctx, userID)
	require.NoError(t, err)
	assert.Len(t, secrets, 3)
	assert.Equal(t, "value1", secrets["secret.user.1"])
	assert.Equal(t, "value2", secrets["secret.user.2"])
	assert.Equal(t, "value3", secrets["secret.user.3"])
}

func TestSecretsService_GetUserSecrets_UserIsolation(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "settings")
	defer tc.CleanupTestData()

	// Create actual users in the database
	userID1 := createTestUser(t, tc)
	userID2 := createTestUser(t, tc)

	setupUserSecret(t, tc, userID1, "secret.user.1a", "value1a")
	setupUserSecret(t, tc, userID2, "secret.user.2a", "value2a")

	svc := createSecretsService(t, tc)
	ctx := context.Background()

	// User 1 should only get their own secrets
	secrets1, err := svc.GetUserSecrets(ctx, userID1)
	require.NoError(t, err)
	assert.Len(t, secrets1, 1)
	assert.Contains(t, secrets1, "secret.user.1a")
	assert.NotContains(t, secrets1, "secret.user.2a")

	// User 2 should only get their own secrets
	secrets2, err := svc.GetUserSecrets(ctx, userID2)
	require.NoError(t, err)
	assert.Len(t, secrets2, 1)
	assert.Contains(t, secrets2, "secret.user.2a")
	assert.NotContains(t, secrets2, "secret.user.1a")
}

func TestSecretsService_GetUserSecrets_Empty(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "settings")
	defer tc.CleanupTestData()

	userID := createTestUser(t, tc)

	svc := createSecretsService(t, tc)
	ctx := context.Background()

	secrets, err := svc.GetUserSecrets(ctx, userID)
	require.NoError(t, err)
	assert.Empty(t, secrets)
}

// =============================================================================
// SetSystemSecret Tests
// =============================================================================

func TestSecretsService_SetSystemSecret_Create(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "settings")
	defer tc.CleanupTestData()

	svc := createSecretsService(t, tc)
	ctx := context.Background()

	err := svc.SetSystemSecret(ctx, "secret.system.create", "my-value", "Test secret")
	require.NoError(t, err)

	// Verify it was created
	decrypted, err := svc.GetSystemSecret(ctx, "secret.system.create")
	require.NoError(t, err)
	assert.Equal(t, "my-value", decrypted)
}

func TestSecretsService_SetSystemSecret_Update(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "settings")
	defer tc.CleanupTestData()

	setupSystemSecret(t, tc, "secret.system.update", "original-value")

	svc := createSecretsService(t, tc)
	ctx := context.Background()

	err := svc.SetSystemSecret(ctx, "secret.system.update", "updated-value", "Updated description")
	require.NoError(t, err)

	// Verify it was updated
	decrypted, err := svc.GetSystemSecret(ctx, "secret.system.update")
	require.NoError(t, err)
	assert.Equal(t, "updated-value", decrypted)
}

func TestSecretsService_SetSystemSecret_SpecialCharacters(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "settings")
	defer tc.CleanupTestData()

	svc := createSecretsService(t, tc)
	ctx := context.Background()

	specialValue := "value\nwith\nnewlines\tand\ttabs\"quotes'"
	err := svc.SetSystemSecret(ctx, "secret.system.special", specialValue, "Special chars")
	require.NoError(t, err)

	decrypted, err := svc.GetSystemSecret(ctx, "secret.system.special")
	require.NoError(t, err)
	assert.Equal(t, specialValue, decrypted)
}

// =============================================================================
// SetUserSecret Tests
// =============================================================================

func TestSecretsService_SetUserSecret_Create(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "settings")
	defer tc.CleanupTestData()

	svc := createSecretsService(t, tc)
	ctx := context.Background()
	userID := createTestUser(t, tc)

	err := svc.SetUserSecret(ctx, userID, "secret.user.create", "user-value", "User secret")
	require.NoError(t, err)

	// Verify it was created
	decrypted, err := svc.GetUserSecret(ctx, userID, "secret.user.create")
	require.NoError(t, err)
	assert.Equal(t, "user-value", decrypted)
}

func TestSecretsService_SetUserSecret_Update(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "settings")
	defer tc.CleanupTestData()

	userID := createTestUser(t, tc)
	setupUserSecret(t, tc, userID, "secret.user.update", "original")

	svc := createSecretsService(t, tc)
	ctx := context.Background()

	err := svc.SetUserSecret(ctx, userID, "secret.user.update", "updated", "Updated")
	require.NoError(t, err)

	decrypted, err := svc.GetUserSecret(ctx, userID, "secret.user.update")
	require.NoError(t, err)
	assert.Equal(t, "updated", decrypted)
}

func TestSecretsService_SetUserSecret_UserIsolation(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "settings")
	defer tc.CleanupTestData()

	svc := createSecretsService(t, tc)
	ctx := context.Background()
	userID1 := createTestUser(t, tc)
	userID2 := createTestUser(t, tc)

	// User 1 sets a secret
	err := svc.SetUserSecret(ctx, userID1, "secret.user.isolated", "user1-value", "User 1 secret")
	require.NoError(t, err)

	// User 2 sets a secret with the same key
	err = svc.SetUserSecret(ctx, userID2, "secret.user.isolated", "user2-value", "User 2 secret")
	require.NoError(t, err)

	// Verify they're isolated
	user1Value, err := svc.GetUserSecret(ctx, userID1, "secret.user.isolated")
	require.NoError(t, err)
	assert.Equal(t, "user1-value", user1Value)

	user2Value, err := svc.GetUserSecret(ctx, userID2, "secret.user.isolated")
	require.NoError(t, err)
	assert.Equal(t, "user2-value", user2Value)
}

// =============================================================================
// DeleteSystemSecret Tests
// =============================================================================

func TestSecretsService_DeleteSystemSecret_Success(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "settings")
	defer tc.CleanupTestData()

	setupSystemSecret(t, tc, "secret.system.delete", "value")

	svc := createSecretsService(t, tc)
	ctx := context.Background()

	err := svc.DeleteSystemSecret(ctx, "secret.system.delete")
	require.NoError(t, err)

	// Verify it's deleted
	_, err = svc.GetSystemSecret(ctx, "secret.system.delete")
	assert.Error(t, err)
	assert.ErrorIs(t, err, settings.ErrSecretNotFound)
}

func TestSecretsService_DeleteSystemSecret_NotFound(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "settings")
	defer tc.CleanupTestData()

	svc := createSecretsService(t, tc)
	ctx := context.Background()

	err := svc.DeleteSystemSecret(ctx, "secret.system.nonexistent")
	assert.Error(t, err)
	assert.ErrorIs(t, err, settings.ErrSecretNotFound)
}

// =============================================================================
// DeleteUserSecret Tests
// =============================================================================

func TestSecretsService_DeleteUserSecret_Success(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "settings")
	defer tc.CleanupTestData()

	userID := createTestUser(t, tc)
	setupUserSecret(t, tc, userID, "secret.user.delete", "value")

	svc := createSecretsService(t, tc)
	ctx := context.Background()

	err := svc.DeleteUserSecret(ctx, userID, "secret.user.delete")
	require.NoError(t, err)

	// Verify it's deleted
	_, err = svc.GetUserSecret(ctx, userID, "secret.user.delete")
	assert.Error(t, err)
	assert.ErrorIs(t, err, settings.ErrSecretNotFound)
}

func TestSecretsService_DeleteUserSecret_NotFound(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "settings")
	defer tc.CleanupTestData()

	svc := createSecretsService(t, tc)
	ctx := context.Background()
	userID := createTestUser(t, tc)

	err := svc.DeleteUserSecret(ctx, userID, "secret.user.nonexistent")
	assert.Error(t, err)
	assert.ErrorIs(t, err, settings.ErrSecretNotFound)
}

func TestSecretsService_DeleteUserSecret_CannotDeleteOtherUsers(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "settings")
	defer tc.CleanupTestData()

	userID1 := createTestUser(t, tc)
	userID2 := createTestUser(t, tc)
	setupUserSecret(t, tc, userID1, "secret.user.protected", "value")

	svc := createSecretsService(t, tc)
	ctx := context.Background()

	// User 2 tries to delete user 1's secret
	err := svc.DeleteUserSecret(ctx, userID2, "secret.user.protected")
	assert.Error(t, err)
	assert.ErrorIs(t, err, settings.ErrSecretNotFound)

	// Verify user 1's secret still exists
	value, err := svc.GetUserSecret(ctx, userID1, "secret.user.protected")
	require.NoError(t, err)
	assert.Equal(t, "value", value)
}

// =============================================================================
// GetUserSetting Tests
// =============================================================================

func TestSecretsService_GetUserSetting_Secret(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "settings")
	defer tc.CleanupTestData()

	userID := createTestUser(t, tc)
	setupUserSecret(t, tc, userID, "user.secret.setting", "secret-value")

	svc := createSecretsService(t, tc)
	ctx := context.Background()

	value, err := svc.GetUserSetting(ctx, userID, "user.secret.setting")
	require.NoError(t, err)
	assert.Equal(t, "secret-value", value)
}

func TestSecretsService_GetUserSetting_NonSecret(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "settings")
	defer tc.CleanupTestData()

	svc := createCustomSettingsService(t, tc)
	ctx := context.Background()
	userID := createTestUser(t, tc) // Create actual user in database

	// Create a non-secret user setting
	req := settings.CreateUserSettingRequest{
		Key:   "user.public.setting",
		Value: map[string]interface{}{"value": "public-value"},
	}
	_, err := svc.CreateUserSetting(ctx, userID, req)
	require.NoError(t, err)

	// Get via SecretsService
	secretsSvc := createSecretsService(t, tc)
	value, err := secretsSvc.GetUserSetting(ctx, userID, "user.public.setting")
	require.NoError(t, err)
	assert.Equal(t, "public-value", value)
}

func TestSecretsService_GetUserSetting_NotFound(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "settings")
	defer tc.CleanupTestData()

	svc := createSecretsService(t, tc)
	ctx := context.Background()
	userID := createTestUser(t, tc)

	_, err := svc.GetUserSetting(ctx, userID, "user.nonexistent")
	assert.Error(t, err)
	assert.ErrorIs(t, err, settings.ErrSettingNotFound)
}

// =============================================================================
// GetSystemSetting Tests (SecretsService variant)
// =============================================================================

func TestSecretsService_GetSystemSetting_Secret(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "settings")
	defer tc.CleanupTestData()

	setupSystemSecret(t, tc, "system.secret.setting.getsecret", "system-secret")

	svc := createSecretsService(t, tc)
	ctx := context.Background()

	value, err := svc.GetSystemSetting(ctx, "system.secret.setting.getsecret")
	require.NoError(t, err)
	assert.Equal(t, "system-secret", value)
}

func TestSecretsService_GetSystemSetting_NonSecret(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "settings")
	defer tc.CleanupTestData()

	svc := createCustomSettingsService(t, tc)
	ctx := context.Background()
	createdBy := uuid.New()

	// Create a non-secret system setting
	req := settings.CreateCustomSettingRequest{
		Key:   "system.public.setting.getnonsecret",
		Value: map[string]interface{}{"value": "system-public"},
	}
	_, err := svc.CreateSetting(ctx, req, createdBy)
	require.NoError(t, err)

	// Get via SecretsService
	secretsSvc := createSecretsService(t, tc)
	value, err := secretsSvc.GetSystemSetting(ctx, "system.public.setting.getnonsecret")
	require.NoError(t, err)
	assert.Equal(t, "system-public", value)
}

func TestSecretsService_GetSystemSetting_NotFound(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "settings")
	defer tc.CleanupTestData()

	svc := createSecretsService(t, tc)
	ctx := context.Background()

	_, err := svc.GetSystemSetting(ctx, "system.nonexistent")
	assert.Error(t, err)
	assert.ErrorIs(t, err, settings.ErrSettingNotFound)
}

// =============================================================================
// GetAllUserSettings Tests
// =============================================================================

func TestSecretsService_GetAllUserSettings_Mixed(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "settings")
	defer tc.CleanupTestData()

	customSvc := createCustomSettingsService(t, tc)
	ctx := context.Background()
	userID := createTestUser(t, tc) // Create actual user in database

	// Create secret setting
	secretReq := settings.CreateSecretSettingRequest{
		Key:   "user.mixed.secret",
		Value: "secret-value",
	}
	_, err := customSvc.CreateSecretSetting(ctx, secretReq, &userID, userID)
	require.NoError(t, err)

	// Create non-secret setting
	publicReq := settings.CreateUserSettingRequest{
		Key:   "user.mixed.public",
		Value: map[string]interface{}{"value": "public-value"},
	}
	_, err = customSvc.CreateUserSetting(ctx, userID, publicReq)
	require.NoError(t, err)

	// Get all settings
	secretsSvc := createSecretsService(t, tc)
	settings, err := secretsSvc.GetAllUserSettings(ctx, userID)
	require.NoError(t, err)
	assert.Len(t, settings, 2)
	assert.Equal(t, "secret-value", settings["user.mixed.secret"])
	assert.Equal(t, "public-value", settings["user.mixed.public"])
}

// =============================================================================
// GetAllSystemSettings Tests
// =============================================================================

func TestSecretsService_GetAllSystemSettings_Mixed(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "settings")
	defer tc.CleanupTestData()

	customSvc := createCustomSettingsService(t, tc)
	ctx := context.Background()
	createdBy := uuid.New()

	// Create secret setting
	secretReq := settings.CreateSecretSettingRequest{
		Key:   "system.mixed.secret.getall",
		Value: "system-secret",
	}
	_, err := customSvc.CreateSecretSetting(ctx, secretReq, nil, createdBy)
	require.NoError(t, err)

	// Create non-secret setting
	publicReq := settings.CreateCustomSettingRequest{
		Key:   "system.mixed.public.getall",
		Value: map[string]interface{}{"value": "system-public"},
	}
	_, err = customSvc.CreateSetting(ctx, publicReq, createdBy)
	require.NoError(t, err)

	// Get all settings
	secretsSvc := createSecretsService(t, tc)
	settings, err := secretsSvc.GetAllSystemSettings(ctx)
	require.NoError(t, err)
	// Note: CleanupTestData() doesn't clean custom settings, so we check for our specific items
	assert.GreaterOrEqual(t, len(settings), 2)
	assert.Equal(t, "system-secret", settings["system.mixed.secret.getall"])
	assert.Equal(t, "system-public", settings["system.mixed.public.getall"])
}

// =============================================================================
// GetSettingWithFallback Tests
// =============================================================================

func TestSecretsService_GetSettingWithFallback_UserSetting(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "settings")
	defer tc.CleanupTestData()

	userID := createTestUser(t, tc) // Create actual user in database
	setupUserSecret(t, tc, userID, "fallback.usersetting", "user-value")

	svc := createSecretsService(t, tc)
	ctx := context.Background()

	value, found, err := svc.GetSettingWithFallback(ctx, &userID, "fallback.usersetting")
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "user-value", value)
}

func TestSecretsService_GetSettingWithFallback_SystemFallback(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "settings")
	defer tc.CleanupTestData()

	uniqueKey := fmt.Sprintf("fallback.systemfallback.%s", uuid.New().String())
	setupSystemSecret(t, tc, uniqueKey, "system-value")

	svc := createSecretsService(t, tc)
	ctx := context.Background()
	userID := createTestUser(t, tc)

	// User doesn't have this setting, should fall back to system
	value, found, err := svc.GetSettingWithFallback(ctx, &userID, uniqueKey)
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "system-value", value)
}

func TestSecretsService_GetSettingWithFallback_NoFallback(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "settings")
	defer tc.CleanupTestData()

	svc := createSecretsService(t, tc)
	ctx := context.Background()
	userID := createTestUser(t, tc)

	value, found, err := svc.GetSettingWithFallback(ctx, &userID, "fallback.nonexistent")
	require.NoError(t, err)
	assert.False(t, found)
	assert.Empty(t, value)
}

func TestSecretsService_GetSettingWithFallback_NilUserID(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "settings")
	defer tc.CleanupTestData()

	uniqueKey := fmt.Sprintf("fallback.systemniluserid.%s", uuid.New().String())
	setupSystemSecret(t, tc, uniqueKey, "system-value")

	svc := createSecretsService(t, tc)
	ctx := context.Background()

	// nil userID means only check system settings
	value, found, err := svc.GetSettingWithFallback(ctx, nil, uniqueKey)
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "system-value", value)
}
