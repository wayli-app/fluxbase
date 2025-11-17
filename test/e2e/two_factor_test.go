package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/pquerna/otp/totp"
	"github.com/stretchr/testify/require"
	"github.com/wayli-app/fluxbase/test"
)

// cleanup2FATestUsers removes any existing test users created by 2FA tests
func cleanup2FATestUsers(t *testing.T, tc *test.TestContext) {
	_, err := tc.DB.Pool().Exec(context.Background(),
		"DELETE FROM auth.users WHERE email LIKE '%2fa_%@example.com'")
	if err != nil {
		t.Logf("Warning: failed to cleanup test users: %v", err)
	}
}

// Test2FASetup tests initiating 2FA setup
func Test2FASetup(t *testing.T) {
	tc := test.NewTestContext(t)
	defer tc.Close()

	tc.EnsureAuthSchema()
	cleanup2FATestUsers(t, tc)

	// Create a test user
	user, token := createTestUser(t, tc, "2fa_setup@example.com", "TestPass123!")

	// Setup 2FA
	resp := tc.NewRequest("POST", "/api/v1/auth/2fa/setup").
		WithAuth(token).
		Send().
		AssertStatus(fiber.StatusOK)

	var result map[string]interface{}
	resp.JSON(&result)

	require.Contains(t, result, "id", "Should have factor ID")
	require.Contains(t, result, "type", "Should have type")
	require.Contains(t, result, "totp", "Should have totp object")

	totpData := result["totp"].(map[string]interface{})
	require.Contains(t, totpData, "secret", "Should have secret in totp object")
	require.Contains(t, totpData, "qr_code", "Should have qr_code in totp object")
	require.Contains(t, totpData, "uri", "Should have uri in totp object")

	secret := totpData["secret"].(string)
	require.NotEmpty(t, secret, "Secret should not be empty")

	t.Logf("2FA setup successful for user: %s, secret: %s", user["id"], secret)
}

// Test2FAEnable tests enabling 2FA after setup
func Test2FAEnable(t *testing.T) {
	tc := test.NewTestContext(t)
	defer tc.Close()

	tc.EnsureAuthSchema()
	cleanup2FATestUsers(t, tc)

	// Create a test user
	user, token := createTestUser(t, tc, "2fa_enable@example.com", "TestPass123!")

	// Setup 2FA
	setupResp := tc.NewRequest("POST", "/api/v1/auth/2fa/setup").
		WithAuth(token).
		Send().
		AssertStatus(fiber.StatusOK)

	var setupResult map[string]interface{}
	setupResp.JSON(&setupResult)

	// Extract secret from nested totp object
	totpData := setupResult["totp"].(map[string]interface{})
	secret := totpData["secret"].(string)

	// Generate a valid TOTP code
	code, err := totp.GenerateCode(secret, time.Now())
	require.NoError(t, err, "Should generate TOTP code")

	// Enable 2FA with the code
	enableReq := map[string]interface{}{
		"code": code,
	}

	enableResp := tc.NewRequest("POST", "/api/v1/auth/2fa/enable").
		WithAuth(token).
		WithBody(enableReq).
		Send().
		AssertStatus(fiber.StatusOK)

	var enableResult map[string]interface{}
	enableResp.JSON(&enableResult)

	require.True(t, enableResult["success"].(bool), "Should be successful")
	require.Contains(t, enableResult, "backup_codes", "Should have backup codes")

	backupCodes := enableResult["backup_codes"].([]interface{})
	require.Len(t, backupCodes, 10, "Should have 10 backup codes")

	t.Logf("2FA enabled for user: %s, received %d backup codes", user["id"], len(backupCodes))
}

// Test2FAStatusCheck tests checking 2FA status
func Test2FAStatusCheck(t *testing.T) {
	tc := test.NewTestContext(t)
	defer tc.Close()

	tc.EnsureAuthSchema()
	cleanup2FATestUsers(t, tc)

	// Create a test user
	_, token := createTestUser(t, tc, "2fa_status@example.com", "TestPass123!")

	// Check status before enabling
	resp := tc.NewRequest("GET", "/api/v1/auth/2fa/status").
		WithAuth(token).
		Send().
		AssertStatus(fiber.StatusOK)

	var result map[string]interface{}
	resp.JSON(&result)

	require.False(t, result["totp_enabled"].(bool), "2FA should not be enabled initially")

	// Enable 2FA
	enable2FAForUser(t, tc, token)

	// Check status after enabling
	resp2 := tc.NewRequest("GET", "/api/v1/auth/2fa/status").
		WithAuth(token).
		Send().
		AssertStatus(fiber.StatusOK)

	var result2 map[string]interface{}
	resp2.JSON(&result2)

	require.True(t, result2["totp_enabled"].(bool), "2FA should be enabled after setup")

	t.Logf("2FA status check successful")
}

// Test2FALoginFlow tests the complete login flow with 2FA
func Test2FALoginFlow(t *testing.T) {
	tc := test.NewTestContext(t)
	defer tc.Close()

	tc.EnsureAuthSchema()
	cleanup2FATestUsers(t, tc)

	email := "2fa_login@example.com"
	password := "TestPass123!"

	// Create a test user
	_, token := createTestUser(t, tc, email, password)

	// Enable 2FA
	secret := enable2FAForUser(t, tc, token)

	// Try to login - should require 2FA
	loginReq := map[string]interface{}{
		"email":    email,
		"password": password,
	}

	loginResp := tc.NewRequest("POST", "/api/v1/auth/signin").
		WithBody(loginReq).
		Send().
		AssertStatus(fiber.StatusOK)

	var loginResult map[string]interface{}
	loginResp.JSON(&loginResult)

	require.True(t, loginResult["requires_2fa"].(bool), "Should require 2FA")
	require.Contains(t, loginResult, "user_id", "Should have user_id")
	userID := loginResult["user_id"].(string)

	// Generate a valid TOTP code
	code, err := totp.GenerateCode(secret, time.Now())
	require.NoError(t, err, "Should generate TOTP code")

	// Verify 2FA code
	verifyReq := map[string]interface{}{
		"user_id": userID,
		"code":    code,
	}

	verifyResp := tc.NewRequest("POST", "/api/v1/auth/2fa/verify").
		WithBody(verifyReq).
		Send().
		AssertStatus(fiber.StatusOK)

	var verifyResult map[string]interface{}
	verifyResp.JSON(&verifyResult)

	require.Contains(t, verifyResult, "access_token", "Should have access_token")
	require.Contains(t, verifyResult, "refresh_token", "Should have refresh_token")
	require.Contains(t, verifyResult, "user", "Should have user")

	t.Logf("2FA login flow successful for user: %s", userID)
}

// Test2FALoginWithBackupCode tests logging in with a backup code
func Test2FALoginWithBackupCode(t *testing.T) {
	tc := test.NewTestContext(t)
	defer tc.Close()

	tc.EnsureAuthSchema()
	cleanup2FATestUsers(t, tc)

	email := "2fa_backup@example.com"
	password := "TestPass123!"

	// Create a test user
	_, token := createTestUser(t, tc, email, password)

	// Enable 2FA and get backup codes
	backupCodes := enable2FAWithBackupCodes(t, tc, token)
	require.NotEmpty(t, backupCodes, "Should have backup codes")

	// Try to login - should require 2FA
	loginReq := map[string]interface{}{
		"email":    email,
		"password": password,
	}

	loginResp := tc.NewRequest("POST", "/api/v1/auth/signin").
		WithBody(loginReq).
		Send().
		AssertStatus(fiber.StatusOK)

	var loginResult map[string]interface{}
	loginResp.JSON(&loginResult)

	require.True(t, loginResult["requires_2fa"].(bool), "Should require 2FA")
	userID := loginResult["user_id"].(string)

	// Use first backup code
	backupCode := backupCodes[0].(string)

	verifyReq := map[string]interface{}{
		"user_id": userID,
		"code":    backupCode,
	}

	verifyResp := tc.NewRequest("POST", "/api/v1/auth/2fa/verify").
		WithBody(verifyReq).
		Send().
		AssertStatus(fiber.StatusOK)

	var verifyResult map[string]interface{}
	verifyResp.JSON(&verifyResult)

	require.Contains(t, verifyResult, "access_token", "Should have access_token")

	t.Logf("2FA login with backup code successful for user: %s", userID)

	// Try to use the same backup code again - should fail
	loginResp2 := tc.NewRequest("POST", "/api/v1/auth/signin").
		WithBody(loginReq).
		Send().
		AssertStatus(fiber.StatusOK)

	var loginResult2 map[string]interface{}
	loginResp2.JSON(&loginResult2)

	userID2 := loginResult2["user_id"].(string)

	verifyReq2 := map[string]interface{}{
		"user_id": userID2,
		"code":    backupCode,
	}

	tc.NewRequest("POST", "/api/v1/auth/2fa/verify").
		WithBody(verifyReq2).
		Send().
		AssertStatus(fiber.StatusBadRequest)

	t.Logf("Backup code cannot be reused - test passed")
}

// Test2FADisable tests disabling 2FA
func Test2FADisable(t *testing.T) {
	tc := test.NewTestContext(t)
	defer tc.Close()

	tc.EnsureAuthSchema()
	cleanup2FATestUsers(t, tc)

	email := "2fa_disable@example.com"
	password := "TestPass123!"

	// Create a test user
	_, token := createTestUser(t, tc, email, password)

	// Enable 2FA
	enable2FAForUser(t, tc, token)

	// Check that 2FA is enabled
	statusResp := tc.NewRequest("GET", "/api/v1/auth/2fa/status").
		WithAuth(token).
		Send().
		AssertStatus(fiber.StatusOK)

	var statusResult map[string]interface{}
	statusResp.JSON(&statusResult)
	require.True(t, statusResult["totp_enabled"].(bool), "2FA should be enabled")

	// Disable 2FA
	disableReq := map[string]interface{}{
		"password": password,
	}

	disableResp := tc.NewRequest("POST", "/api/v1/auth/2fa/disable").
		WithAuth(token).
		WithBody(disableReq).
		Send().
		AssertStatus(fiber.StatusOK)

	var disableResult map[string]interface{}
	disableResp.JSON(&disableResult)
	require.True(t, disableResult["success"].(bool), "Should be successful")

	// Check that 2FA is disabled
	statusResp2 := tc.NewRequest("GET", "/api/v1/auth/2fa/status").
		WithAuth(token).
		Send().
		AssertStatus(fiber.StatusOK)

	var statusResult2 map[string]interface{}
	statusResp2.JSON(&statusResult2)
	require.False(t, statusResult2["totp_enabled"].(bool), "2FA should be disabled")

	t.Logf("2FA disable successful")
}

// Test2FAInvalidCode tests 2FA verification with invalid code
func Test2FAInvalidCode(t *testing.T) {
	tc := test.NewTestContext(t)
	defer tc.Close()

	tc.EnsureAuthSchema()
	cleanup2FATestUsers(t, tc)

	email := "2fa_invalid@example.com"
	password := "TestPass123!"

	// Create a test user
	_, token := createTestUser(t, tc, email, password)

	// Enable 2FA
	enable2FAForUser(t, tc, token)

	// Try to login
	loginReq := map[string]interface{}{
		"email":    email,
		"password": password,
	}

	loginResp := tc.NewRequest("POST", "/api/v1/auth/signin").
		WithBody(loginReq).
		Send().
		AssertStatus(fiber.StatusOK)

	var loginResult map[string]interface{}
	loginResp.JSON(&loginResult)

	userID := loginResult["user_id"].(string)

	// Try to verify with invalid code
	verifyReq := map[string]interface{}{
		"user_id": userID,
		"code":    "000000",
	}

	tc.NewRequest("POST", "/api/v1/auth/2fa/verify").
		WithBody(verifyReq).
		Send().
		AssertStatus(fiber.StatusBadRequest)

	t.Logf("Invalid 2FA code correctly rejected")
}

// Test2FASetupExpiry tests that 2FA setup expires
func Test2FASetupExpiry(t *testing.T) {
	tc := test.NewTestContext(t)
	defer tc.Close()

	tc.EnsureAuthSchema()
	cleanup2FATestUsers(t, tc)

	// Create a test user
	_, token := createTestUser(t, tc, "2fa_expiry@example.com", "TestPass123!")

	// Setup 2FA
	setupResp := tc.NewRequest("POST", "/api/v1/auth/2fa/setup").
		WithAuth(token).
		Send().
		AssertStatus(fiber.StatusOK)

	var setupResult map[string]interface{}
	setupResp.JSON(&setupResult)

	// Extract secret from nested totp object
	totpData := setupResult["totp"].(map[string]interface{})
	secret := totpData["secret"].(string)

	// Simulate expiry by deleting the setup record
	_, err := tc.DB.Pool().Exec(context.Background(),
		"UPDATE auth.two_factor_setups SET expires_at = NOW() - INTERVAL '1 second' WHERE secret = $1",
		secret)
	require.NoError(t, err, "Should update expiry")

	// Try to enable with expired setup
	code, err := totp.GenerateCode(secret, time.Now())
	require.NoError(t, err, "Should generate TOTP code")

	enableReq := map[string]interface{}{
		"code": code,
	}

	tc.NewRequest("POST", "/api/v1/auth/2fa/enable").
		WithAuth(token).
		WithBody(enableReq).
		Send().
		AssertStatus(fiber.StatusBadRequest)

	t.Logf("Expired 2FA setup correctly rejected")
}

// Helper functions

func createTestUser(t *testing.T, tc *test.TestContext, email, password string) (map[string]interface{}, string) {
	signupReq := map[string]interface{}{
		"email":    email,
		"password": password,
	}

	resp := tc.NewRequest("POST", "/api/v1/auth/signup").
		WithBody(signupReq).
		Send().
		AssertStatus(fiber.StatusCreated)

	var result map[string]interface{}
	resp.JSON(&result)

	token := result["access_token"].(string)
	user := result["user"].(map[string]interface{})

	return user, token
}

func enable2FAForUser(t *testing.T, tc *test.TestContext, token string) string {
	// Setup 2FA
	setupResp := tc.NewRequest("POST", "/api/v1/auth/2fa/setup").
		WithAuth(token).
		Send().
		AssertStatus(fiber.StatusOK)

	var setupResult map[string]interface{}
	setupResp.JSON(&setupResult)

	// Extract secret from nested totp object
	totpData := setupResult["totp"].(map[string]interface{})
	secret := totpData["secret"].(string)

	// Generate a valid TOTP code
	code, err := totp.GenerateCode(secret, time.Now())
	require.NoError(t, err, "Should generate TOTP code")

	// Enable 2FA
	enableReq := map[string]interface{}{
		"code": code,
	}

	tc.NewRequest("POST", "/api/v1/auth/2fa/enable").
		WithAuth(token).
		WithBody(enableReq).
		Send().
		AssertStatus(fiber.StatusOK)

	return secret
}

func enable2FAWithBackupCodes(t *testing.T, tc *test.TestContext, token string) []interface{} {
	// Setup 2FA
	setupResp := tc.NewRequest("POST", "/api/v1/auth/2fa/setup").
		WithAuth(token).
		Send().
		AssertStatus(fiber.StatusOK)

	var setupResult map[string]interface{}
	setupResp.JSON(&setupResult)

	// Extract secret from nested totp object
	totpData := setupResult["totp"].(map[string]interface{})
	secret := totpData["secret"].(string)

	// Generate a valid TOTP code
	code, err := totp.GenerateCode(secret, time.Now())
	require.NoError(t, err, "Should generate TOTP code")

	// Enable 2FA
	enableReq := map[string]interface{}{
		"code": code,
	}

	enableResp := tc.NewRequest("POST", "/api/v1/auth/2fa/enable").
		WithAuth(token).
		WithBody(enableReq).
		Send().
		AssertStatus(fiber.StatusOK)

	var enableResult map[string]interface{}
	enableResp.JSON(&enableResult)

	backupCodes := enableResult["backup_codes"].([]interface{})
	return backupCodes
}
