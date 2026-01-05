package auth

import (
	"crypto/sha256"
	"encoding/base64"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHashMagicLinkToken(t *testing.T) {
	tests := []struct {
		name  string
		token string
	}{
		{
			name:  "standard token",
			token: "abcdef123456",
		},
		{
			name:  "long token",
			token: "this-is-a-very-long-token-with-many-characters-1234567890",
		},
		{
			name:  "short token",
			token: "abc",
		},
		{
			name:  "token with special characters",
			token: "token!@#$%^&*()",
		},
		{
			name:  "base64 encoded token",
			token: "YWJjZGVmMTIzNDU2",
		},
		{
			name:  "empty string",
			token: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash := hashMagicLinkToken(tt.token)

			// Verify hash is not empty
			assert.NotEmpty(t, hash)

			// Verify hash is base64 URL encoded
			_, err := base64.URLEncoding.DecodeString(hash)
			assert.NoError(t, err, "hash should be valid base64 URL encoding")

			// Verify hash is deterministic (same input produces same output)
			hash2 := hashMagicLinkToken(tt.token)
			assert.Equal(t, hash, hash2, "hash should be deterministic")

			// Verify hash length is consistent (SHA-256 produces 32 bytes = 44 base64 chars with padding)
			assert.True(t, len(hash) > 40, "hash should be at least 40 characters")
		})
	}
}

func TestHashMagicLinkToken_DifferentInputs(t *testing.T) {
	// Verify that different inputs produce different hashes
	token1 := "token1"
	token2 := "token2"

	hash1 := hashMagicLinkToken(token1)
	hash2 := hashMagicLinkToken(token2)

	assert.NotEqual(t, hash1, hash2, "different tokens should produce different hashes")
}

func TestHashMagicLinkToken_SHA256Verification(t *testing.T) {
	// Verify the hash is actually SHA-256
	token := "test-token-123"
	hash := hashMagicLinkToken(token)

	// Manually compute SHA-256
	expectedHash := sha256.Sum256([]byte(token))
	expectedBase64 := base64.URLEncoding.EncodeToString(expectedHash[:])

	assert.Equal(t, expectedBase64, hash, "hash should be SHA-256 encoded as base64 URL")
}

func TestHashMagicLinkToken_CaseSensitive(t *testing.T) {
	// Verify hashing is case-sensitive
	token1 := "TestToken"
	token2 := "testtoken"

	hash1 := hashMagicLinkToken(token1)
	hash2 := hashMagicLinkToken(token2)

	assert.NotEqual(t, hash1, hash2, "hashing should be case-sensitive")
}

func TestGenerateMagicLinkToken_Success(t *testing.T) {
	token, err := GenerateMagicLinkToken()

	require.NoError(t, err)
	assert.NotEmpty(t, token)

	// Verify token is base64 URL encoded
	decoded, err := base64.URLEncoding.DecodeString(token)
	assert.NoError(t, err, "token should be valid base64 URL encoding")

	// Verify decoded length is 32 bytes
	assert.Equal(t, 32, len(decoded), "token should be 32 bytes when decoded")
}

func TestGenerateMagicLinkToken_Uniqueness(t *testing.T) {
	// Generate multiple tokens and verify they are all unique
	tokens := make(map[string]bool)
	iterations := 100

	for i := 0; i < iterations; i++ {
		token, err := GenerateMagicLinkToken()
		require.NoError(t, err)

		assert.False(t, tokens[token], "each token should be unique")
		tokens[token] = true
	}

	assert.Len(t, tokens, iterations, "all generated tokens should be unique")
}

func TestGenerateMagicLinkToken_Length(t *testing.T) {
	// Verify token length is consistent
	var lengths []int

	for i := 0; i < 10; i++ {
		token, err := GenerateMagicLinkToken()
		require.NoError(t, err)

		lengths = append(lengths, len(token))
	}

	// All tokens should have the same length
	firstLength := lengths[0]
	for _, length := range lengths {
		assert.Equal(t, firstLength, length, "all tokens should have consistent length")
	}

	// 32 bytes base64 encoded should be 44 characters (with padding) or 43 (URL encoding without padding)
	assert.True(t, firstLength >= 43 && firstLength <= 44, "token length should be 43-44 characters")
}

func TestGenerateMagicLinkToken_URLSafe(t *testing.T) {
	// Verify tokens are URL-safe (no +, /, or = characters that need escaping)
	for i := 0; i < 50; i++ {
		token, err := GenerateMagicLinkToken()
		require.NoError(t, err)

		// URL-safe base64 should not contain + or /
		assert.False(t, strings.Contains(token, "+"), "token should not contain + (URL-safe)")
		assert.False(t, strings.Contains(token, "/"), "token should not contain / (URL-safe)")
	}
}

func TestGenerateMagicLinkToken_Randomness(t *testing.T) {
	// Basic statistical test: verify tokens have good entropy
	// Generate many tokens and check they don't follow patterns

	tokens := make([]string, 100)
	for i := 0; i < 100; i++ {
		token, err := GenerateMagicLinkToken()
		require.NoError(t, err)
		tokens[i] = token
	}

	// Count unique characters across all tokens
	charSet := make(map[rune]bool)
	for _, token := range tokens {
		for _, char := range token {
			charSet[char] = true
		}
	}

	// Base64 URL alphabet has 64 characters (A-Z, a-z, 0-9, -, _)
	// We should see a good variety
	assert.True(t, len(charSet) > 20, "tokens should use a variety of characters for good entropy")
}

func TestMagicLinkTokenHashing_Integration(t *testing.T) {
	// Integration test: generate token, hash it, verify properties
	token, err := GenerateMagicLinkToken()
	require.NoError(t, err)

	// Hash the token
	hash := hashMagicLinkToken(token)

	// Verify hash properties
	assert.NotEqual(t, token, hash, "hash should be different from original token")
	assert.True(t, len(hash) > len(token)/2, "hash should be substantial length")

	// Verify same token produces same hash
	hash2 := hashMagicLinkToken(token)
	assert.Equal(t, hash, hash2, "same token should produce same hash")

	// Verify hash is URL-safe
	_, err = base64.URLEncoding.DecodeString(hash)
	assert.NoError(t, err, "hash should be valid base64 URL encoding")
}

func TestHashMagicLinkToken_SecurityProperties(t *testing.T) {
	// Verify security properties of hashing

	// 1. One-way: given a hash, you cannot determine the original token
	// (This is a property of SHA-256, we're just documenting it)
	token := "secret-token-12345"
	hash := hashMagicLinkToken(token)
	assert.NotContains(t, hash, "secret", "hash should not contain parts of the original token")
	assert.NotContains(t, hash, "12345", "hash should not contain parts of the original token")

	// 2. Small changes produce completely different hashes (avalanche effect)
	token1 := "token1"
	token2 := "token2" // Only 1 character different
	hash1 := hashMagicLinkToken(token1)
	hash2 := hashMagicLinkToken(token2)

	// Count how many characters are different
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

	// Should be significantly different (avalanche effect)
	percentDifferent := float64(diffCount) / float64(minLen) * 100
	assert.True(t, percentDifferent > 40, "small input change should cause large hash change (avalanche effect)")
}

func TestHashMagicLinkToken_EmptyString(t *testing.T) {
	// Verify empty string produces consistent hash
	hash1 := hashMagicLinkToken("")
	hash2 := hashMagicLinkToken("")

	assert.Equal(t, hash1, hash2, "empty string should produce consistent hash")
	assert.NotEmpty(t, hash1, "empty string should still produce a hash")
}

func TestHashMagicLinkToken_UnicodeSupport(t *testing.T) {
	// Verify hashing works with Unicode characters
	tests := []struct {
		name  string
		token string
	}{
		{
			name:  "emoji",
			token: "token-🔒-secure",
		},
		{
			name:  "chinese characters",
			token: "密码令牌",
		},
		{
			name:  "mixed unicode",
			token: "tøkén-señor-日本",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash := hashMagicLinkToken(tt.token)

			assert.NotEmpty(t, hash)
			_, err := base64.URLEncoding.DecodeString(hash)
			assert.NoError(t, err, "hash should be valid base64 URL encoding")
		})
	}
}
