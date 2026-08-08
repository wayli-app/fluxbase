package auth

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Unit tests for the pure MFA decision helpers extracted from MFAService
// methods (issue #313 option C). These cover the security-relevant branches
// that were previously reachable only via DB-backed integration tests:
//   - validateTOTPEnable: setup-expiry + encryption-key-presence (EnableTOTP)
//   - resolveTOTPSecret: plaintext-fallback policy (VerifyTOTPWithContext)
//   - verifyTOTPDisablePassword: password-reverification gate (DisableTOTP)
//
// Each helper takes its dependencies (clock, decrypt fn, comparator) as
// parameters, so these tests need no database.

// =============================================================================
// validateTOTPEnable Tests
// =============================================================================

func TestValidateTOTPEnable(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC)
	validKey := []byte("0123456789abcdef0123456789abcdef") // 32 bytes

	tests := []struct {
		name          string
		expiresAt     time.Time
		encryptionKey []byte
		wantErr       string // substring; "" means no error expected
	}{
		{
			name:          "valid setup with key",
			expiresAt:     now.Add(5 * time.Minute),
			encryptionKey: validKey,
			wantErr:       "",
		},
		{
			name:          "expired setup rejected",
			expiresAt:     now.Add(-1 * time.Minute),
			encryptionKey: validKey,
			wantErr:       "expired",
		},
		{
			name:          "exactly now is not after → rejected? (boundary: After is strict)",
			expiresAt:     now,
			encryptionKey: validKey,
			// now.After(expiresAt) where expiresAt==now is false, so this PASSES.
			// Pin the boundary: equal timestamps do not count as expired.
			wantErr: "",
		},
		{
			name:          "missing encryption key rejected even if not expired",
			expiresAt:     now.Add(5 * time.Minute),
			encryptionKey: nil,
			wantErr:       "encryption key not configured",
		},
		{
			name:          "empty encryption key rejected",
			expiresAt:     now.Add(5 * time.Minute),
			encryptionKey: []byte{},
			wantErr:       "encryption key not configured",
		},
		{
			name:          "expiry checked before key (expired + no key → expired wins)",
			expiresAt:     now.Add(-1 * time.Minute),
			encryptionKey: nil,
			wantErr:       "expired",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validateTOTPEnable(tt.expiresAt, now, tt.encryptionKey)
			if tt.wantErr == "" {
				assert.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			}
		})
	}
}

// =============================================================================
// resolveTOTPSecret Tests
// =============================================================================
//
// Contract: the plaintext fallback exists for backward compatibility with
// secrets stored before encryption was enabled, and for decryption failures
// (corruption / never-encrypted). fellBackToPlaintext must be true in exactly
// those cases and false only on a clean decrypt.

func TestResolveTOTPSecret(t *testing.T) {
	t.Parallel()
	key := []byte("0123456789abcdef0123456789abcdef")

	t.Run("no key → plaintext fallback, nil err", func(t *testing.T) {
		t.Parallel()
		secret, fellBack, err := resolveTOTPSecret("rawsecret", nil, failingDecrypt)
		assert.Equal(t, "rawsecret", secret)
		assert.True(t, fellBack, "absent key must trigger plaintext fallback")
		assert.NoError(t, err, "absent-key fallback has no decrypt error")
	})

	t.Run("successful decrypt → no fallback, nil err", func(t *testing.T) {
		t.Parallel()
		decrypt := func(stored string, k []byte) (string, error) {
			assert.Equal(t, key, k)
			return "decrypted:" + stored, nil
		}
		secret, fellBack, err := resolveTOTPSecret("ciphertext", key, decrypt)
		require.NoError(t, err)
		assert.Equal(t, "decrypted:ciphertext", secret)
		assert.False(t, fellBack, "clean decrypt must not fall back")
	})

	t.Run("decrypt error → plaintext fallback, err preserved for logging", func(t *testing.T) {
		t.Parallel()
		decryptErr := errors.New("bad ciphertext")
		decrypt := func(string, []byte) (string, error) { return "", decryptErr }
		secret, fellBack, err := resolveTOTPSecret("ciphertext", key, decrypt)
		assert.Equal(t, "ciphertext", secret, "stored value used as-is on decrypt failure")
		assert.True(t, fellBack, "decrypt failure must trigger plaintext fallback")
		assert.ErrorIs(t, err, decryptErr, "decrypt error surfaced for caller logging")
	})
}

// failingDecrypt is a decrypt stub that always errors; used to prove the
// absent-key path never calls decrypt.
func failingDecrypt(string, []byte) (string, error) {
	return "", errors.New("decrypt should not have been called")
}

// =============================================================================
// verifyTOTPDisablePassword Tests
// =============================================================================

func TestVerifyTOTPDisablePassword(t *testing.T) {
	t.Parallel()
	t.Run("passwordless account bypasses check", func(t *testing.T) {
		t.Parallel()
		// cmp must NOT be called for passwordless accounts.
		err := verifyTOTPDisablePassword("", "anything", func(string, string) error {
			t.Fatal("comparator must not be called for passwordless account")
			return nil
		})
		assert.NoError(t, err)
	})

	t.Run("correct password accepted", func(t *testing.T) {
		t.Parallel()
		called := false
		err := verifyTOTPDisablePassword("hash", "pw", func(hash, pw string) error {
			called = true
			assert.Equal(t, "hash", hash)
			assert.Equal(t, "pw", pw)
			return nil
		})
		assert.NoError(t, err)
		assert.True(t, called, "comparator must be called when a password hash exists")
	})

	t.Run("wrong password rejected with generic message", func(t *testing.T) {
		t.Parallel()
		err := verifyTOTPDisablePassword("hash", "wrong", func(string, string) error {
			return errors.New("bcrypt mismatch")
		})
		require.Error(t, err)
		assert.Equal(t, "invalid password", err.Error(),
			"must not leak the underlying comparator error")
		// CRITICAL: the returned error must NOT contain the comparator's detail
		// (avoids leaking bcrypt timing/format info to an attacker).
		assert.False(t, strings.Contains(err.Error(), "bcrypt"))
	})
}
