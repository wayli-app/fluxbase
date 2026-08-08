package auth

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/pquerna/otp/totp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// This file contains comprehensive tests for MFA/TOTP authentication flows

// NOTE: This file previously contained several `t.Skip` stub functions
// (TestEnableMFA_AlreadyEnabled, TestVerifyMFA_DisabledUser,
// TestDisableMFA_*, TestRegenerateBackupCodes_MFANotEnabled) that asserted
// nothing — each was an empty body with a "Requires database integration"
// skip. They have been removed to avoid implying coverage that doesn't exist.
// The real DB-coupled MFA service methods (EnableMFA/DisableMFA/VerifyMFA on
// MFAService) still lack test coverage; see the linked issue for the plan to
// cover them via repository mocks or a CI-run integration job. The pure-logic
// helpers below (GenerateTOTPSecret, VerifyTOTPCode, GenerateBackupCodes,
// VerifyBackupCode, MockTOTPRateLimiter) are exercised by the tests in this
// file.

// TestEnableMFA_Success tests successful MFA enrollment with TOTP
func TestEnableMFA_Success(t *testing.T) {
	// Note: This test demonstrates the expected flow
	// Full integration tests would require database setup

	// Generate a TOTP secret for setup
	secret, qrCodeDataURI, otpauthURI, err := GenerateTOTPSecret("Fluxbase", "user@example.com")
	require.NoError(t, err)
	assert.NotEmpty(t, secret, "TOTP secret should be generated")
	assert.NotEmpty(t, qrCodeDataURI, "QR code should be generated")
	assert.NotEmpty(t, otpauthURI, "OTPAuth URI should be generated")

	// Generate backup codes
	backupCodes, hashedCodes, err := GenerateBackupCodes(3)
	require.NoError(t, err)
	assert.Len(t, backupCodes, 3, "Should generate 3 backup codes")
	assert.Len(t, hashedCodes, 3, "Should hash all backup codes")

	// Verify backup codes are unique
	codeMap := make(map[string]bool)
	for _, code := range backupCodes {
		assert.False(t, codeMap[code], "Backup codes should be unique")
		codeMap[code] = true
	}

	// Generate a valid TOTP code
	validCode, err := totp.GenerateCode(secret, time.Now())
	require.NoError(t, err)

	// Verify the code
	valid, err := VerifyTOTPCode(validCode, secret)
	require.NoError(t, err)
	assert.True(t, valid, "Valid TOTP code should verify successfully")

	// Verify backup codes work
	for i := 0; i < len(backupCodes); i++ {
		match, err := VerifyBackupCode(backupCodes[i], hashedCodes[i])
		require.NoError(t, err)
		assert.True(t, match, "Backup code should verify against its hash")
	}
}

// TestEnableMFA_AlreadyEnabled: removed (was an empty t.Skip stub).
// See file note above re: DB-coupled MFA coverage.

// TestEnableMFA_InvalidSecret tests MFA setup with invalid TOTP secret
func TestEnableMFA_InvalidSecret(t *testing.T) {
	invalidSecret := "invalid-secret-not-base32"
	code := "123456"

	// Should handle invalid secret gracefully
	// Note: totp.Validate returns false for invalid secrets but no error
	valid, err := VerifyTOTPCode(code, invalidSecret)
	assert.NoError(t, err, "No error returned for invalid secret")
	assert.False(t, valid, "Invalid secret should not verify codes")
}

// TestEnableMFA_BackupCodesGeneration tests backup code generation
func TestEnableMFA_BackupCodesGeneration(t *testing.T) {
	tests := []struct {
		name  string
		count int
	}{
		{"single backup code", 1},
		{"standard backup codes", 10},
		{"large backup code set", 20},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plainCodes, hashedCodes, err := GenerateBackupCodes(tt.count)

			require.NoError(t, err)
			assert.Len(t, plainCodes, tt.count)
			assert.Len(t, hashedCodes, tt.count)

			// All codes should be unique
			codeSet := make(map[string]bool)
			for _, code := range plainCodes {
				assert.NotEmpty(t, code)
				assert.False(t, codeSet[code], "Codes should be unique")
				codeSet[code] = true
				assert.Len(t, code, 8, "Backup codes should be 8 characters")
			}

			// All hashes should be unique (bcrypt uses random salt)
			hashSet := make(map[string]bool)
			for _, hash := range hashedCodes {
				assert.NotEmpty(t, hash)
				assert.False(t, hashSet[hash], "Hashes should be unique due to salt")
				hashSet[hash] = true
			}
		})
	}
}

// TestVerifyMFA_ValidCode tests TOTP verification with valid code
func TestVerifyMFA_ValidCode(t *testing.T) {
	// Generate a TOTP secret
	secret, _, _, err := GenerateTOTPSecret("Fluxbase", "user@example.com")
	require.NoError(t, err)

	// Generate a valid TOTP code for current time
	validCode, err := totp.GenerateCode(secret, time.Now())
	require.NoError(t, err)

	// Verify the code
	valid, err := VerifyTOTPCode(validCode, secret)

	require.NoError(t, err)
	assert.True(t, valid, "Valid TOTP code should verify successfully")
}

// TestVerifyMFA_InvalidCode tests TOTP verification with invalid codes
func TestVerifyMFA_InvalidCode(t *testing.T) {
	// Generate a TOTP secret
	secret, _, _, err := GenerateTOTPSecret("Fluxbase", "user@example.com")
	require.NoError(t, err)

	invalidCodes := []string{
		"000000",
		"111111",
		"222222",
		"333333",
		"invalid",
		"12345",
		"",
	}

	for _, invalidCode := range invalidCodes {
		t.Run("code_"+invalidCode, func(t *testing.T) {
			valid, err := VerifyTOTPCode(invalidCode, secret)

			require.NoError(t, err)
			assert.False(t, valid, "Invalid TOTP code should not verify")
		})
	}
}

// TestVerifyMFA_RateLimited tests TOTP verification rate limiting
func TestVerifyMFA_RateLimited(t *testing.T) {
	ctx := context.Background()
	userID := "user-123"

	// Create mock rate limiter
	rateLimiter := NewMockTOTPRateLimiter()
	rateLimiter.SetMaxAttempts(3)

	// Simulate failed attempts
	for i := 0; i < 3; i++ {
		err := rateLimiter.RecordAttempt(ctx, userID, false, "127.0.0.1", "test-agent")
		require.NoError(t, err)
	}

	// Should be rate limited now
	err := rateLimiter.CheckRateLimit(ctx, userID)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrTOTPRateLimitExceeded)

	// Verify failed attempt count
	assert.Equal(t, 3, rateLimiter.GetFailedAttempts(userID))
}

// TestVerifyMFA_RateLimitExpires tests that rate limit expires after lockout period
func TestVerifyMFA_RateLimitExpires(t *testing.T) {
	ctx := context.Background()
	userID := "user-123"

	// Create mock rate limiter with short lockout
	rateLimiter := NewMockTOTPRateLimiter()
	rateLimiter.SetMaxAttempts(3)

	// Simulate failed attempts to reach limit
	for i := 0; i < 3; i++ {
		err := rateLimiter.RecordAttempt(ctx, userID, false, "127.0.0.1", "test-agent")
		require.NoError(t, err)
	}

	// Should be rate limited
	err := rateLimiter.CheckRateLimit(ctx, userID)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrTOTPRateLimitExceeded)

	// Reset the failed attempts to simulate time passage
	// In real implementation, the window would expire and reset
	rateLimiter.Reset(userID)

	// Should no longer be rate limited
	err = rateLimiter.CheckRateLimit(ctx, userID)
	assert.NoError(t, err)
}

// TestVerifyMFA_BackupCode tests backup code verification
func TestVerifyMFA_BackupCode(t *testing.T) {
	// Generate backup codes
	plainCodes, hashedCodes, err := GenerateBackupCodes(3)
	require.NoError(t, err)

	// Test each backup code
	for i := 0; i < len(plainCodes); i++ {
		valid, err := VerifyBackupCode(plainCodes[i], hashedCodes[i])

		require.NoError(t, err)
		assert.True(t, valid, "Valid backup code should verify")
	}

	// Test that wrong code doesn't verify
	valid, err := VerifyBackupCode("WRONGCODE", hashedCodes[0])
	require.NoError(t, err)
	assert.False(t, valid, "Wrong backup code should not verify")
}

// TestVerifyMFA_BackupCode_OneTimeUse tests that backup codes can only be used once
func TestVerifyMFA_BackupCode_OneTimeUse(t *testing.T) {
	// In a real implementation, backup codes are removed after use
	// This test simulates that behavior

	// Generate backup codes
	plainCodes, hashedCodes, err := GenerateBackupCodes(3)
	require.NoError(t, err)

	// Use first backup code
	usedCode := plainCodes[0]
	usedHash := hashedCodes[0]

	// Simulate removing used code from storage
	remainingPlainCodes := plainCodes[1:]
	remainingHashedCodes := hashedCodes[1:]

	// Verify used code is no longer in remaining codes
	assert.NotContains(t, remainingPlainCodes, usedCode)
	assert.NotContains(t, remainingHashedCodes, usedHash)

	// Verify other codes still work
	for i := 0; i < len(remainingPlainCodes); i++ {
		valid, err := VerifyBackupCode(remainingPlainCodes[i], remainingHashedCodes[i])
		require.NoError(t, err)
		assert.True(t, valid, "Remaining backup codes should still verify")
	}

	// Used code shouldn't verify against remaining hashes
	valid := false
	for _, hash := range remainingHashedCodes {
		match, _ := VerifyBackupCode(usedCode, hash)
		if match {
			valid = true
			break
		}
	}
	assert.False(t, valid, "Used backup code should not verify against any remaining hash")
}

// TestVerifyMFA_DisabledUser / TestDisableMFA_*: removed (empty t.Skip stubs).
// See file note above re: DB-coupled MFA coverage.

// TestGenerateBackupCodes_Success tests backup code generation
func TestGenerateBackupCodes_Success(t *testing.T) {
	// Test various counts
	tests := []struct {
		name  string
		count int
	}{
		{"minimum codes", 1},
		{"standard codes", 10},
		{"maximum codes", 20},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plainCodes, hashedCodes, err := GenerateBackupCodes(tt.count)

			require.NoError(t, err)
			assert.Len(t, plainCodes, tt.count, "Should generate exact number of codes")
			assert.Len(t, hashedCodes, tt.count, "Should hash all codes")

			// Verify code format
			for _, code := range plainCodes {
				assert.Len(t, code, 8, "Backup codes should be 8 characters")
			}
		})
	}
}

// TestGenerateBackupCodes_AlreadyExists tests regenerating backup codes
func TestGenerateBackupCodes_AlreadyExists(t *testing.T) {
	// Use smaller code count for faster tests (especially with -race flag)
	// Testing the "already exists" scenario doesn't require many codes
	firstSet, firstHashes, err := GenerateBackupCodes(3)
	require.NoError(t, err)

	// Generate second set of codes
	secondSet, secondHashes, err := GenerateBackupCodes(3)
	require.NoError(t, err)

	// Verify they're different
	assert.NotEqual(t, firstSet, secondSet, "New codes should be different from old codes")
	assert.NotEqual(t, firstHashes, secondHashes, "New hashes should be different from old hashes")

	// Verify cross-compatibility fails (codes from set1 shouldn't verify against hashes from set2)
	allCompatible := true
	for i, code := range firstSet {
		match, _ := VerifyBackupCode(code, secondHashes[i])
		if !match {
			allCompatible = false
			break
		}
	}
	assert.False(t, allCompatible, "Codes should not verify against different code set hashes")
}

// TestVerifyBackupCode_Success tests successful backup code verification
func TestVerifyBackupCode_Success(t *testing.T) {
	// Use smaller code count for faster tests (especially with -race flag)
	plainCodes, hashedCodes, err := GenerateBackupCodes(3)
	require.NoError(t, err)

	// Verify each code works
	for i := 0; i < len(plainCodes); i++ {
		valid, err := VerifyBackupCode(plainCodes[i], hashedCodes[i])
		require.NoError(t, err)
		assert.True(t, valid, "Backup code should verify against its hash")
	}
}

// TestVerifyBackupCode_AlreadyUsed tests that used backup codes can't be reused
func TestVerifyBackupCode_AlreadyUsed(t *testing.T) {
	plainCodes, hashedCodes, err := GenerateBackupCodes(3)
	require.NoError(t, err)

	// Use a backup code
	usedCode := plainCodes[0]
	usedIndex := -1
	for i, code := range plainCodes {
		if code == usedCode {
			usedIndex = i
			break
		}
	}

	require.Equal(t, 0, usedIndex)

	// Simulate removal (as would happen in real implementation)
	plainCodes = append(plainCodes[:usedIndex], plainCodes[usedIndex+1:]...)
	hashedCodes = append(hashedCodes[:usedIndex], hashedCodes[usedIndex+1:]...)

	// Verify used code is not in remaining codes
	assert.NotContains(t, plainCodes, usedCode)
	assert.Len(t, plainCodes, 2)
	assert.Len(t, hashedCodes, 2)
}

// TestVerifyBackupCode_InvalidCode tests invalid backup codes
func TestVerifyBackupCode_InvalidCode(t *testing.T) {
	_, hashedCodes, err := GenerateBackupCodes(3)
	require.NoError(t, err)

	invalidCodes := []string{
		"",
		"short",
		"verylongcode123",
		"INVALID",
		"1234567",
	}

	for _, invalidCode := range invalidCodes {
		t.Run("code_"+invalidCode, func(t *testing.T) {
			// Test against first hash
			valid, err := VerifyBackupCode(invalidCode, hashedCodes[0])

			require.NoError(t, err)
			assert.False(t, valid, "Invalid backup code should not verify")
		})
	}
}

// TestRegenerateBackupCodes_Success tests regenerating backup codes
func TestRegenerateBackupCodes_Success(t *testing.T) {
	// Generate initial codes
	initialCodes, initialHashes, err := GenerateBackupCodes(3)
	require.NoError(t, err)
	assert.Len(t, initialCodes, 3)

	// "Regenerate" - generate new codes
	newCodes, newHashes, err := GenerateBackupCodes(3)
	require.NoError(t, err)
	assert.Len(t, newCodes, 3)

	// Verify new codes are different
	assert.NotEqual(t, initialCodes, newCodes, "Regenerated codes should be different")
	assert.NotEqual(t, initialHashes, newHashes, "Regenerated hashes should be different")

	// Verify old codes don't work with new hashes
	allValid := true
	for i, code := range initialCodes {
		valid, _ := VerifyBackupCode(code, newHashes[i])
		if !valid {
			allValid = false
			break
		}
	}
	assert.False(t, allValid, "Old codes should not verify against new hashes")
}

// TestRegenerateBackupCodes_MFANotEnabled: removed (empty t.Skip stub).
// See file note above re: DB-coupled MFA coverage.

// TestTOTPCode_TimeWindow tests TOTP code validation within time window
func TestTOTPCode_TimeWindow(t *testing.T) {
	secret, _, _, err := GenerateTOTPSecret("Fluxbase", "user@example.com")
	require.NoError(t, err)

	// Generate codes for different times within valid window
	now := time.Now()
	times := []time.Time{
		now.Add(-30 * time.Second), // Previous time window
		now,                        // Current time
		now.Add(30 * time.Second),  // Next time window
	}

	for _, testTime := range times {
		code, err := totp.GenerateCode(secret, testTime)
		require.NoError(t, err)

		// Codes from adjacent time windows should still verify
		valid, err := VerifyTOTPCode(code, secret)
		require.NoError(t, err)
		assert.True(t, valid, "TOTP code from valid time window should verify")
	}
}

// TestTOTPCode_OutsideTimeWindow tests TOTP code validation outside time window
func TestTOTPCode_OutsideTimeWindow(t *testing.T) {
	secret, _, _, err := GenerateTOTPSecret("Fluxbase", "user@example.com")
	require.NoError(t, err)

	// Generate code from far in the past (outside valid window)
	oldTime := time.Now().Add(-5 * time.Minute)
	oldCode, err := totp.GenerateCode(secret, oldTime)
	require.NoError(t, err)

	// Old code should not verify
	valid, err := VerifyTOTPCode(oldCode, secret)
	require.NoError(t, err)
	assert.False(t, valid, "TOTP code from outside time window should not verify")
}

// TestTOTPSetupResponse_Format tests TOTP setup response format
func TestTOTPSetupResponse_Format(t *testing.T) {
	secret, qrCode, uri, err := GenerateTOTPSecret("MyApp", "user@example.com")
	require.NoError(t, err)

	// Verify secret format
	assert.NotEmpty(t, secret)
	assert.Len(t, secret, 32, "Secret should be 32 characters (base32 encoded)")

	// Verify QR code format
	assert.True(t, strings.HasPrefix(qrCode, "data:image/png;base64,"),
		"QR code should be base64-encoded PNG data URI")

	// Verify otpauth URI format
	assert.True(t, strings.HasPrefix(uri, "otpauth://totp/"),
		"URI should start with otpauth://totp/")
	assert.Contains(t, uri, "MyApp", "URI should contain issuer")
	assert.Contains(t, uri, "user@example.com", "URI should contain account name")
	assert.Contains(t, uri, "secret=", "URI should contain secret parameter")
}

// TestBackupCodeSecurity tests backup code security properties
func TestBackupCodeSecurity(t *testing.T) {
	plainCodes, hashedCodes, err := GenerateBackupCodes(3)
	require.NoError(t, err)

	// Test 1: Plain codes should be reversible
	for _, code := range plainCodes {
		assert.NotEmpty(t, code, "Plain code should not be empty")
		assert.Len(t, code, 8, "Plain code should be 8 characters")
	}

	// Test 2: Hashed codes should not reveal plain codes
	for i, hash := range hashedCodes {
		assert.NotEmpty(t, hash, "Hashed code should not be empty")
		assert.NotEqual(t, hash, plainCodes[i], "Hash should not equal plain code")
		assert.Greater(t, len(hash), 20, "Bcrypt hash should be longer than plain code")
	}

	// Test 3: Same code produces different hashes due to salt
	_, hash1, err := GenerateBackupCodes(1)
	require.NoError(t, err)
	_, hash2, err := GenerateBackupCodes(1)
	require.NoError(t, err)

	// Hash the same code twice using bcrypt directly
	// (This simulates what would happen if we hashed the same code again)
	// Since bcrypt uses random salt, hashes should be different
	// Note: In our GenerateBackupCodes, each code is unique, so we can't test exact same code
	// But we can verify that all hashes are unique
	hashMap := make(map[string]bool)
	for _, hash := range append(hash1, hash2...) {
		assert.False(t, hashMap[hash], "Each bcrypt hash should be unique due to salt")
		hashMap[hash] = true
	}
}

// TestMFAWorkflow_CompleteFlow tests complete MFA enrollment and verification flow
func TestMFAWorkflow_CompleteFlow(t *testing.T) {
	// Step 1: User requests MFA setup
	secret, qrCode, uri, err := GenerateTOTPSecret("Fluxbase", "user@example.com")
	require.NoError(t, err)

	// Step 2: User scans QR code and enters verification code
	validCode, err := totp.GenerateCode(secret, time.Now())
	require.NoError(t, err)

	// Step 3: Verify TOTP code
	totpValid, err := VerifyTOTPCode(validCode, secret)
	require.NoError(t, err)
	assert.True(t, totpValid, "TOTP code should verify")

	// Step 4: Generate backup codes
	backupCodes, hashedCodes, err := GenerateBackupCodes(3)
	require.NoError(t, err)

	// Step 5: User can use backup code instead of TOTP
	backupValid, err := VerifyBackupCode(backupCodes[0], hashedCodes[0])
	require.NoError(t, err)
	assert.True(t, backupValid, "Backup code should verify")

	// Step 6: Verify all components are present
	assert.NotEmpty(t, secret, "Secret should be set")
	assert.NotEmpty(t, qrCode, "QR code should be generated")
	assert.NotEmpty(t, uri, "OTPAuth URI should be generated")
	assert.NotEmpty(t, backupCodes, "Backup codes should be generated")

	// Step 7: Verify QR code can be decoded from data URI
	assert.True(t, strings.HasPrefix(qrCode, "data:image/png;base64,"),
		"QR code should be base64 data URI")
}

// TestTOTPRateLimiter_AttemptTracking tests rate limiter attempt tracking
func TestTOTPRateLimiter_AttemptTracking(t *testing.T) {
	ctx := context.Background()
	userID := "user-123"
	ipAddress := "127.0.0.1"
	userAgent := "test-agent"

	rateLimiter := NewMockTOTPRateLimiter()
	rateLimiter.SetMaxAttempts(5)

	// Record failed attempts
	for i := 0; i < 3; i++ {
		err := rateLimiter.RecordAttempt(ctx, userID, false, ipAddress, userAgent)
		require.NoError(t, err)
	}

	// Verify attempt count
	assert.Equal(t, 3, rateLimiter.GetFailedAttempts(userID))

	// Record success - should reset counter
	err := rateLimiter.RecordAttempt(ctx, userID, true, ipAddress, userAgent)
	require.NoError(t, err)

	// Counter should be reset
	assert.Equal(t, 0, rateLimiter.GetFailedAttempts(userID))

	// Should not be rate limited after successful attempt
	err = rateLimiter.CheckRateLimit(ctx, userID)
	assert.NoError(t, err)
}
