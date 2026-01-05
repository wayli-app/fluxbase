package auth

import (
	"encoding/base32"
	"strings"
	"testing"
	"time"

	"github.com/pquerna/otp/totp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func TestGenerateTOTPSecret_Success(t *testing.T) {
	issuer := "Fluxbase"
	accountName := "user@example.com"

	secret, qrCodeDataURI, otpauthURI, err := GenerateTOTPSecret(issuer, accountName)

	require.NoError(t, err)
	assert.NotEmpty(t, secret)
	assert.NotEmpty(t, qrCodeDataURI)
	assert.NotEmpty(t, otpauthURI)

	// Verify secret is valid base32
	_, err = base32.StdEncoding.DecodeString(secret)
	assert.NoError(t, err, "secret should be valid base32")

	// Verify QR code data URI format
	assert.True(t, strings.HasPrefix(qrCodeDataURI, "data:image/png;base64,"))

	// Verify otpauth URI contains issuer and account name
	assert.Contains(t, otpauthURI, issuer)
	assert.Contains(t, otpauthURI, accountName)
	assert.True(t, strings.HasPrefix(otpauthURI, "otpauth://totp/"))
}

func TestGenerateTOTPSecret_UniquenessPerCall(t *testing.T) {
	issuer := "Fluxbase"
	accountName := "user@example.com"

	secret1, _, _, err1 := GenerateTOTPSecret(issuer, accountName)
	secret2, _, _, err2 := GenerateTOTPSecret(issuer, accountName)

	require.NoError(t, err1)
	require.NoError(t, err2)

	// Each call should generate a unique secret
	assert.NotEqual(t, secret1, secret2)
}

func TestGenerateTOTPSecret_DifferentAccounts(t *testing.T) {
	tests := []struct {
		name        string
		issuer      string
		accountName string
	}{
		{
			name:        "standard email",
			issuer:      "Fluxbase",
			accountName: "user@example.com",
		},
		{
			name:        "email with plus sign",
			issuer:      "Fluxbase",
			accountName: "user+test@example.com",
		},
		{
			name:        "email with subdomain",
			issuer:      "Fluxbase",
			accountName: "admin@staging.example.com",
		},
		{
			name:        "different issuer",
			issuer:      "MyApp",
			accountName: "user@example.com",
		},
		{
			name:        "special characters",
			issuer:      "Fluxbase",
			accountName: "user-test@example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			secret, qrCodeDataURI, otpauthURI, err := GenerateTOTPSecret(tt.issuer, tt.accountName)

			require.NoError(t, err)
			assert.NotEmpty(t, secret)
			assert.NotEmpty(t, qrCodeDataURI)
			assert.NotEmpty(t, otpauthURI)
			assert.Contains(t, otpauthURI, tt.issuer)
			assert.Contains(t, otpauthURI, tt.accountName)
		})
	}
}

func TestVerifyTOTPCode_ValidCode(t *testing.T) {
	// Generate a secret
	issuer := "Fluxbase"
	accountName := "user@example.com"
	secret, _, _, err := GenerateTOTPSecret(issuer, accountName)
	require.NoError(t, err)

	// Generate a valid TOTP code for current time
	code, err := totp.GenerateCode(secret, time.Now())
	require.NoError(t, err)

	// Verify the code
	valid, err := VerifyTOTPCode(code, secret)

	require.NoError(t, err)
	assert.True(t, valid, "valid TOTP code should be verified successfully")
}

func TestVerifyTOTPCode_InvalidCode(t *testing.T) {
	// Generate a secret
	secret, _, _, err := GenerateTOTPSecret("Fluxbase", "user@example.com")
	require.NoError(t, err)

	tests := []struct {
		name string
		code string
	}{
		{
			name: "wrong code",
			code: "000000",
		},
		{
			name: "invalid format",
			code: "abcdef",
		},
		{
			name: "too short",
			code: "123",
		},
		{
			name: "too long",
			code: "12345678",
		},
		{
			name: "empty code",
			code: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid, err := VerifyTOTPCode(tt.code, secret)

			require.NoError(t, err)
			assert.False(t, valid, "invalid TOTP code should not be verified")
		})
	}
}

func TestVerifyTOTPCode_ExpiredCode(t *testing.T) {
	// Generate a secret
	secret, _, _, err := GenerateTOTPSecret("Fluxbase", "user@example.com")
	require.NoError(t, err)

	// Generate a code for a past time (more than 30 seconds ago - outside TOTP window)
	pastTime := time.Now().Add(-2 * time.Minute)
	oldCode, err := totp.GenerateCode(secret, pastTime)
	require.NoError(t, err)

	// Verify the old code should fail
	valid, err := VerifyTOTPCode(oldCode, secret)

	require.NoError(t, err)
	assert.False(t, valid, "expired TOTP code should not be verified")
}

func TestGenerateBackupCodes_Success(t *testing.T) {
	count := 10

	plainCodes, hashedCodes, err := GenerateBackupCodes(count)

	require.NoError(t, err)
	assert.Len(t, plainCodes, count)
	assert.Len(t, hashedCodes, count)

	// Verify all codes are unique
	codeMap := make(map[string]bool)
	for _, code := range plainCodes {
		assert.NotEmpty(t, code)
		assert.False(t, codeMap[code], "backup codes should be unique")
		codeMap[code] = true
	}

	// Verify all hashes are different
	hashMap := make(map[string]bool)
	for _, hash := range hashedCodes {
		assert.NotEmpty(t, hash)
		assert.False(t, hashMap[hash], "backup code hashes should be unique due to salt")
		hashMap[hash] = true
	}

	// Verify codes are valid base32 (from generateAppBackupCode)
	for _, code := range plainCodes {
		_, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(code)
		assert.NoError(t, err, "backup code should be valid base32")
	}

	// Verify each plain code matches its hash
	for i := 0; i < count; i++ {
		err := bcrypt.CompareHashAndPassword([]byte(hashedCodes[i]), []byte(plainCodes[i]))
		assert.NoError(t, err, "plain code should match its hashed version")
	}
}

func TestGenerateBackupCodes_DifferentCounts(t *testing.T) {
	tests := []struct {
		name  string
		count int
	}{
		{
			name:  "single code",
			count: 1,
		},
		{
			name:  "typical count",
			count: 10,
		},
		{
			name:  "large count",
			count: 20,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plainCodes, hashedCodes, err := GenerateBackupCodes(tt.count)

			require.NoError(t, err)
			assert.Len(t, plainCodes, tt.count)
			assert.Len(t, hashedCodes, tt.count)
		})
	}
}

func TestGenerateBackupCodes_ZeroCount(t *testing.T) {
	plainCodes, hashedCodes, err := GenerateBackupCodes(0)

	require.NoError(t, err)
	assert.Empty(t, plainCodes)
	assert.Empty(t, hashedCodes)
}

func TestVerifyBackupCode_ValidCode(t *testing.T) {
	// Generate backup codes
	plainCodes, hashedCodes, err := GenerateBackupCodes(5)
	require.NoError(t, err)

	// Verify each code against its hash
	for i := 0; i < len(plainCodes); i++ {
		valid, err := VerifyBackupCode(plainCodes[i], hashedCodes[i])

		require.NoError(t, err)
		assert.True(t, valid, "plain backup code should verify against its hash")
	}
}

func TestVerifyBackupCode_InvalidCode(t *testing.T) {
	// Generate a backup code
	plainCodes, hashedCodes, err := GenerateBackupCodes(1)
	require.NoError(t, err)

	tests := []struct {
		name string
		code string
	}{
		{
			name: "wrong code",
			code: "WRONGCODE123",
		},
		{
			name: "empty code",
			code: "",
		},
		{
			name: "different case (if applicable)",
			code: strings.ToLower(plainCodes[0]),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid, err := VerifyBackupCode(tt.code, hashedCodes[0])

			require.NoError(t, err)
			assert.False(t, valid, "invalid backup code should not verify")
		})
	}
}

func TestVerifyBackupCode_WrongHash(t *testing.T) {
	// Generate two sets of backup codes
	plainCodes1, _, err := GenerateBackupCodes(1)
	require.NoError(t, err)

	_, hashedCodes2, err := GenerateBackupCodes(1)
	require.NoError(t, err)

	// Try to verify code1 against hash2 (should fail)
	valid, err := VerifyBackupCode(plainCodes1[0], hashedCodes2[0])

	require.NoError(t, err)
	assert.False(t, valid, "backup code should not verify against different code's hash")
}

func TestGenerateAppBackupCode_Format(t *testing.T) {
	// Generate multiple codes to test consistency
	for i := 0; i < 100; i++ {
		code, err := generateAppBackupCode()

		require.NoError(t, err)
		assert.NotEmpty(t, code)

		// Verify it's valid base32
		_, err = base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(code)
		assert.NoError(t, err, "backup code should be valid base32")

		// Code should be 8 characters (5 bytes base32 encoded without padding)
		assert.Equal(t, 8, len(code), "backup code should be 8 characters")
	}
}

func TestGenerateAppBackupCode_Uniqueness(t *testing.T) {
	// Generate many codes to verify uniqueness
	codeMap := make(map[string]bool)
	iterations := 1000

	for i := 0; i < iterations; i++ {
		code, err := generateAppBackupCode()
		require.NoError(t, err)

		assert.False(t, codeMap[code], "backup codes should be unique")
		codeMap[code] = true
	}

	// All codes should be unique
	assert.Len(t, codeMap, iterations)
}

func TestTOTPIntegration_FullFlow(t *testing.T) {
	// Simulate a complete 2FA setup and verification flow
	issuer := "Fluxbase"
	accountName := "user@example.com"

	// Step 1: Generate TOTP secret for user
	secret, qrCodeDataURI, otpauthURI, err := GenerateTOTPSecret(issuer, accountName)
	require.NoError(t, err)
	assert.NotEmpty(t, secret)
	assert.NotEmpty(t, qrCodeDataURI)
	assert.NotEmpty(t, otpauthURI)

	// Step 2: Generate backup codes
	plainCodes, hashedCodes, err := GenerateBackupCodes(10)
	require.NoError(t, err)
	assert.Len(t, plainCodes, 10)
	assert.Len(t, hashedCodes, 10)

	// Step 3: User enters TOTP code from authenticator app
	totpCode, err := totp.GenerateCode(secret, time.Now())
	require.NoError(t, err)

	valid, err := VerifyTOTPCode(totpCode, secret)
	require.NoError(t, err)
	assert.True(t, valid)

	// Step 4: User can also use backup code
	backupValid, err := VerifyBackupCode(plainCodes[0], hashedCodes[0])
	require.NoError(t, err)
	assert.True(t, backupValid)

	// Step 5: After using backup code, it should be marked as used (not tested here, but simulated)
	// In real implementation, you'd remove the used hash from storage

	// Step 6: Other backup codes should still work
	backupValid2, err := VerifyBackupCode(plainCodes[1], hashedCodes[1])
	require.NoError(t, err)
	assert.True(t, backupValid2)
}
