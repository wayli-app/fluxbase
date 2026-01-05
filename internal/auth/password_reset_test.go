package auth

import (
	"crypto/sha256"
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHashPasswordResetToken(t *testing.T) {
	tests := []struct {
		name  string
		token string
	}{
		{"standard token", "reset123token456"},
		{"long token", "very-long-password-reset-token-with-many-characters"},
		{"short token", "xyz"},
		{"empty token", ""},
		{"special characters", "reset!@#$%^&*()"},
		{"unicode token", "reset-密码-token"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash := hashPasswordResetToken(tt.token)

			// Verify hash is not empty
			assert.NotEmpty(t, hash)

			// Verify hash is base64 URL encoded
			_, err := base64.URLEncoding.DecodeString(hash)
			assert.NoError(t, err)

			// Verify hash is deterministic
			hash2 := hashPasswordResetToken(tt.token)
			assert.Equal(t, hash, hash2)
		})
	}
}

func TestHashPasswordResetToken_SHA256(t *testing.T) {
	token := "test-password-reset-token"
	hash := hashPasswordResetToken(token)

	// Manually compute SHA-256
	expectedHash := sha256.Sum256([]byte(token))
	expectedBase64 := base64.URLEncoding.EncodeToString(expectedHash[:])

	assert.Equal(t, expectedBase64, hash)
}

func TestHashPasswordResetToken_DifferentTokens(t *testing.T) {
	token1 := "reset1"
	token2 := "reset2"

	hash1 := hashPasswordResetToken(token1)
	hash2 := hashPasswordResetToken(token2)

	assert.NotEqual(t, hash1, hash2)
}

func TestHashPasswordResetToken_CaseSensitive(t *testing.T) {
	token1 := "ResetToken"
	token2 := "resettoken"

	hash1 := hashPasswordResetToken(token1)
	hash2 := hashPasswordResetToken(token2)

	assert.NotEqual(t, hash1, hash2)
}

func TestGeneratePasswordResetToken_Success(t *testing.T) {
	token, err := GeneratePasswordResetToken()

	require.NoError(t, err)
	assert.NotEmpty(t, token)

	// Verify token is base64 URL encoded
	decoded, err := base64.URLEncoding.DecodeString(token)
	assert.NoError(t, err)
	assert.Equal(t, 32, len(decoded), "token should be 32 bytes when decoded")
}

func TestGeneratePasswordResetToken_Uniqueness(t *testing.T) {
	tokens := make(map[string]bool)
	iterations := 100

	for i := 0; i < iterations; i++ {
		token, err := GeneratePasswordResetToken()
		require.NoError(t, err)

		assert.False(t, tokens[token], "tokens should be unique")
		tokens[token] = true
	}

	assert.Len(t, tokens, iterations)
}

func TestGeneratePasswordResetToken_URLSafe(t *testing.T) {
	for i := 0; i < 50; i++ {
		token, err := GeneratePasswordResetToken()
		require.NoError(t, err)

		// URL-safe base64 should not contain + or /
		assert.NotContains(t, token, "+")
		assert.NotContains(t, token, "/")
	}
}

func TestPasswordResetToken_Integration(t *testing.T) {
	// Generate token
	token, err := GeneratePasswordResetToken()
	require.NoError(t, err)

	// Hash it
	hash := hashPasswordResetToken(token)

	// Verify properties
	assert.NotEqual(t, token, hash)
	assert.True(t, len(hash) > 40)

	// Verify same token produces same hash
	hash2 := hashPasswordResetToken(token)
	assert.Equal(t, hash, hash2)

	// Verify hash is URL-safe
	_, err = base64.URLEncoding.DecodeString(hash)
	assert.NoError(t, err)
}

func TestPasswordResetToken_SecurityProperties(t *testing.T) {
	// Verify avalanche effect: small change in input = large change in hash
	token1 := "reset1"
	token2 := "reset2"

	hash1 := hashPasswordResetToken(token1)
	hash2 := hashPasswordResetToken(token2)

	// Count different characters
	diffCount := 0
	minLen := len(hash1)
	if len(hash2) < minLen {
		minLen = len(hash2)
	}
	for i := 0; i < minLen; i++ {
		if hash1[i] != hash2[i] {
			diffCount++
		}
	}

	percentDifferent := float64(diffCount) / float64(minLen) * 100
	assert.True(t, percentDifferent > 40, "small input change should cause large hash change")
}
