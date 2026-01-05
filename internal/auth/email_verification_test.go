package auth

import (
	"crypto/sha256"
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHashEmailVerificationToken(t *testing.T) {
	tests := []struct {
		name  string
		token string
	}{
		{"standard token", "abc123def456"},
		{"long token", "very-long-token-with-many-characters-for-email-verification"},
		{"short token", "abc"},
		{"empty token", ""},
		{"special characters", "token!@#$%^&*()"},
		{"unicode token", "token-用户-verification"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash := hashEmailVerificationToken(tt.token)

			// Verify hash is not empty
			assert.NotEmpty(t, hash)

			// Verify hash is base64 URL encoded
			_, err := base64.URLEncoding.DecodeString(hash)
			assert.NoError(t, err)

			// Verify hash is deterministic
			hash2 := hashEmailVerificationToken(tt.token)
			assert.Equal(t, hash, hash2)
		})
	}
}

func TestHashEmailVerificationToken_SHA256(t *testing.T) {
	token := "test-verification-token"
	hash := hashEmailVerificationToken(token)

	// Manually compute SHA-256
	expectedHash := sha256.Sum256([]byte(token))
	expectedBase64 := base64.URLEncoding.EncodeToString(expectedHash[:])

	assert.Equal(t, expectedBase64, hash)
}

func TestHashEmailVerificationToken_DifferentTokens(t *testing.T) {
	token1 := "token1"
	token2 := "token2"

	hash1 := hashEmailVerificationToken(token1)
	hash2 := hashEmailVerificationToken(token2)

	assert.NotEqual(t, hash1, hash2)
}

func TestGenerateEmailVerificationToken_Success(t *testing.T) {
	token, err := generateEmailVerificationToken()

	require.NoError(t, err)
	assert.NotEmpty(t, token)

	// Verify token is base64 URL encoded
	decoded, err := base64.URLEncoding.DecodeString(token)
	assert.NoError(t, err)
	assert.Equal(t, 32, len(decoded))
}

func TestGenerateEmailVerificationToken_Uniqueness(t *testing.T) {
	tokens := make(map[string]bool)
	iterations := 100

	for i := 0; i < iterations; i++ {
		token, err := generateEmailVerificationToken()
		require.NoError(t, err)

		assert.False(t, tokens[token], "tokens should be unique")
		tokens[token] = true
	}

	assert.Len(t, tokens, iterations)
}

func TestEmailVerificationToken_Integration(t *testing.T) {
	// Generate token
	token, err := generateEmailVerificationToken()
	require.NoError(t, err)

	// Hash it
	hash := hashEmailVerificationToken(token)

	// Verify properties
	assert.NotEqual(t, token, hash)
	assert.True(t, len(hash) > 40)

	// Verify same token produces same hash
	hash2 := hashEmailVerificationToken(token)
	assert.Equal(t, hash, hash2)
}
