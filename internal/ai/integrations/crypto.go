package integrations

import (
	"fmt"

	"github.com/nimbleflux/fluxbase/internal/crypto"
)

// SecretPrefix tags encrypted values so read paths can distinguish them
// from legacy plaintext. Lazy migration: read paths try decrypt if the
// prefix is present, treat as plaintext otherwise. Write paths always
// encrypt.
//
// Without the prefix we couldn't tell a real API key from an encrypted
// blob — both look like base64-ish strings. The prefix makes the
// detection deterministic.
const SecretPrefix = "enc:"

// MaskPlaceholder is what API responses return in place of the real
// secret value. Updates that receive this placeholder preserve the
// existing encrypted value rather than overwriting it with the literal
// string.
const MaskPlaceholder = "***masked***"

// EncryptSecret encrypts plaintext with the master key and tags it
// with SecretPrefix so future reads know to decrypt. Returns "" for ""
// (preserves the "empty means empty" invariant — never encrypts nil).
// Returns plaintext unchanged if key is empty (encryption not configured).
func EncryptSecret(plaintext string, key []byte) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	if len(key) == 0 {
		// Encryption not configured — preserve old behavior. Log site is
		// the caller's responsibility; we don't log here to avoid noise
		// on every secret write.
		return plaintext, nil
	}
	if len(key) != 32 {
		return "", fmt.Errorf("EncryptSecret: key must be 32 bytes for AES-256, got %d", len(key))
	}
	ciphertext, err := crypto.Encrypt(plaintext, string(key))
	if err != nil {
		return "", fmt.Errorf("EncryptSecret: %w", err)
	}
	return SecretPrefix + ciphertext, nil
}

// DecryptIfEncrypted returns the plaintext for a value tagged with
// SecretPrefix. Values without the prefix are returned unchanged —
// this is the forward-compatible path that lets us read plaintext
// rows written before encryption shipped (lazy migration).
//
// On decryption failure (wrong key, corrupted ciphertext), returns
// the original value unchanged with the error. Callers decide whether
// to fail loudly or fall through; the default is fall through so a
// bad key doesn't lock users out of every integration at once.
func DecryptIfEncrypted(value string, key []byte) (string, error) {
	if value == "" || len(key) == 0 {
		return value, nil
	}
	if !IsEncrypted(value) {
		// Plaintext (pre-migration) — return as-is. Next write will encrypt.
		return value, nil
	}
	inner := value[len(SecretPrefix):]
	plaintext, err := crypto.Decrypt(inner, string(key))
	if err != nil {
		// ponytail: return original on failure rather than empty string.
		// Returning empty would silently corrupt the integration; the
		// caller can detect the error and surface it.
		return value, fmt.Errorf("DecryptIfEncrypted: %w", err)
	}
	return plaintext, nil
}

// IsEncrypted reports whether value carries the SecretPrefix tag.
// Used by read paths to decide whether to attempt decryption.
func IsEncrypted(value string) bool {
	return len(value) >= len(SecretPrefix) && value[:len(SecretPrefix)] == SecretPrefix
}

// IsMasked reports whether value is the API-response mask placeholder.
// Used by update handlers to detect "user didn't change the key" and
// preserve the existing encrypted value.
func IsMasked(value string) bool {
	return value == MaskPlaceholder
}

// MaskSecret returns the API-response placeholder for any secret.
// Empty input returns empty (so we don't leak "there used to be a key here"
// in responses for integrations that don't have one configured).
func MaskSecret(value string) string {
	if value == "" {
		return ""
	}
	return MaskPlaceholder
}
