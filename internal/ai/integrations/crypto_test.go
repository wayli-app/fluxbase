package integrations

import (
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// randomKey generates a 32-byte key for tests (AES-256 size requirement).
func randomKey(t *testing.T) []byte {
	t.Helper()
	key := make([]byte, 32)
	_, err := rand.Read(key)
	require.NoError(t, err)
	return key
}

func TestEncryptSecret_RoundTrip(t *testing.T) {
	key := randomKey(t)
	plaintext := "tvly-abc123secret"

	enc, err := EncryptSecret(plaintext, key)
	require.NoError(t, err)
	assert.NotEqual(t, plaintext, enc, "ciphertext must differ from plaintext")
	assert.True(t, IsEncrypted(enc), "ciphertext must carry the enc: prefix")

	dec, err := DecryptIfEncrypted(enc, key)
	require.NoError(t, err)
	assert.Equal(t, plaintext, dec, "decrypted value must match input")
}

func TestEncryptSecret_EmptyValue_PassesThrough(t *testing.T) {
	key := randomKey(t)
	enc, err := EncryptSecret("", key)
	require.NoError(t, err)
	assert.Equal(t, "", enc, "empty input must not be encrypted")
}

func TestEncryptSecret_NoKey_ReturnsPlaintext(t *testing.T) {
	enc, err := EncryptSecret("plaintext", nil)
	require.NoError(t, err)
	assert.Equal(t, "plaintext", enc, "nil key means encryption disabled; return plaintext unchanged")
}

func TestEncryptSecret_BadKeyLength_ReturnsError(t *testing.T) {
	_, err := EncryptSecret("plaintext", []byte("too-short"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "32 bytes")
}

func TestDecryptIfEncrypted_PlaintextPassesThrough(t *testing.T) {
	key := randomKey(t)
	out, err := DecryptIfEncrypted("tvly-plaintext-key", key)
	require.NoError(t, err)
	assert.Equal(t, "tvly-plaintext-key", out, "values without enc: prefix must pass through unchanged — lazy migration path")
}

func TestDecryptIfEncrypted_EmptyValue(t *testing.T) {
	out, err := DecryptIfEncrypted("", randomKey(t))
	require.NoError(t, err)
	assert.Equal(t, "", out)
}

func TestDecryptIfEncrypted_NilKey_ReturnsValue(t *testing.T) {
	enc, _ := EncryptSecret("secret", randomKey(t))
	out, err := DecryptIfEncrypted(enc, nil)
	require.NoError(t, err)
	assert.Equal(t, enc, out, "nil key = encryption disabled; return value unchanged")
}

func TestIsEncrypted(t *testing.T) {
	assert.True(t, IsEncrypted("enc:abc123"))
	assert.True(t, IsEncrypted("enc:"))
	assert.False(t, IsEncrypted("tvly-abc123"))
	assert.False(t, IsEncrypted(""))
	assert.False(t, IsEncrypted("en")) // too short
}

func TestMaskSecret(t *testing.T) {
	assert.Equal(t, MaskPlaceholder, MaskSecret("anything"))
	assert.Equal(t, "", MaskSecret(""), "empty input must not surface a mask — preserves 'no key configured' semantics")
}

func TestIsMasked(t *testing.T) {
	assert.True(t, IsMasked(MaskPlaceholder))
	assert.False(t, IsMasked("tvly-real-key"))
	assert.False(t, IsMasked(""))
}
