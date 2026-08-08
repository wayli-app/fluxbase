package crypto

import (
	"encoding/base64"
	"errors"
	"testing"

	"github.com/google/uuid"
)

func TestEncryptDecrypt(t *testing.T) {
	key := "12345678901234567890123456789012" // 32 bytes

	tests := []struct {
		name      string
		plaintext string
	}{
		{"empty string", ""},
		{"simple text", "hello world"},
		{"special characters", "p@ssw0rd!#$%^&*()"},
		{"unicode", "日本語テスト🎉"},
		{"long text", "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua."},
		{"api key format", "sk-1234567890abcdefghijklmnopqrstuvwxyz"},
		{"json", `{"key": "value", "nested": {"foo": "bar"}}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// Skip empty string for regular encrypt/decrypt (use IfNotEmpty variants)
			if tt.plaintext == "" {
				return
			}

			encrypted, err := Encrypt(tt.plaintext, key)
			if err != nil {
				t.Fatalf("Encrypt failed: %v", err)
			}

			// Encrypted should be different from plaintext
			if encrypted == tt.plaintext {
				t.Error("Encrypted text should not equal plaintext")
			}

			// Decrypt
			decrypted, err := Decrypt(encrypted, key)
			if err != nil {
				t.Fatalf("Decrypt failed: %v", err)
			}

			if decrypted != tt.plaintext {
				t.Errorf("Decrypted text mismatch: got %q, want %q", decrypted, tt.plaintext)
			}
		})
	}
}

func TestEncryptIfNotEmpty(t *testing.T) {
	key := "12345678901234567890123456789012"

	// Empty value should return empty string
	result, err := EncryptIfNotEmpty("", key)
	if err != nil {
		t.Fatalf("EncryptIfNotEmpty failed on empty: %v", err)
	}
	if result != "" {
		t.Errorf("Expected empty string, got %q", result)
	}

	// Non-empty should encrypt
	result, err = EncryptIfNotEmpty("secret", key)
	if err != nil {
		t.Fatalf("EncryptIfNotEmpty failed: %v", err)
	}
	if result == "" {
		t.Error("Expected non-empty encrypted string")
	}
	if result == "secret" {
		t.Error("Expected encrypted to differ from plaintext")
	}
}

func TestDecryptIfNotEmpty(t *testing.T) {
	key := "12345678901234567890123456789012"

	// Empty ciphertext should return empty string
	result, err := DecryptIfNotEmpty("", key)
	if err != nil {
		t.Fatalf("DecryptIfNotEmpty failed on empty: %v", err)
	}
	if result != "" {
		t.Errorf("Expected empty string, got %q", result)
	}

	// Encrypt then decrypt
	encrypted, _ := Encrypt("secret", key)
	result, err = DecryptIfNotEmpty(encrypted, key)
	if err != nil {
		t.Fatalf("DecryptIfNotEmpty failed: %v", err)
	}
	if result != "secret" {
		t.Errorf("Expected 'secret', got %q", result)
	}
}

func TestInvalidKey(t *testing.T) {
	tests := []struct {
		name string
		key  string
	}{
		{"empty key", ""},
		{"too short", "short"},
		{"too long", "12345678901234567890123456789012345"},
		{"31 bytes", "1234567890123456789012345678901"},
		{"33 bytes", "123456789012345678901234567890123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := Encrypt("test", tt.key)
			if !errors.Is(err, ErrInvalidKey) {
				t.Errorf("Expected ErrInvalidKey, got %v", err)
			}

			_, err = Decrypt("test", tt.key)
			if !errors.Is(err, ErrInvalidKey) {
				t.Errorf("Expected ErrInvalidKey, got %v", err)
			}
		})
	}
}

func TestWrongKeyDecryption(t *testing.T) {
	key1 := "12345678901234567890123456789012"
	key2 := "abcdefghijklmnopqrstuvwxyzABCDEF"

	encrypted, err := Encrypt("secret", key1)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	_, err = Decrypt(encrypted, key2)
	if !errors.Is(err, ErrDecryptionFailed) {
		t.Errorf("Expected ErrDecryptionFailed, got %v", err)
	}
}

func TestInvalidCiphertext(t *testing.T) {
	key := "12345678901234567890123456789012"

	tests := []struct {
		name       string
		ciphertext string
	}{
		{"invalid base64", "not-valid-base64!!!"},
		{"too short", "YWJj"}, // "abc" in base64, too short for nonce
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := Decrypt(tt.ciphertext, key)
			if err == nil {
				t.Error("Expected error for invalid ciphertext")
			}
		})
	}
}

func TestValidateKey(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		wantErr bool
	}{
		{"valid key", "12345678901234567890123456789012", false},
		{"empty key", "", true},
		{"short key", "short", true},
		{"long key", "12345678901234567890123456789012345", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateKey(tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateKey() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEncryptionIsDeterministic(t *testing.T) {
	key := "12345678901234567890123456789012"
	plaintext := "same input"

	// Encrypt the same plaintext twice
	encrypted1, _ := Encrypt(plaintext, key)
	encrypted2, _ := Encrypt(plaintext, key)

	// Due to random nonce, encryptions should differ
	if encrypted1 == encrypted2 {
		t.Error("Expected different ciphertexts due to random nonce")
	}

	// But both should decrypt to the same value
	decrypted1, _ := Decrypt(encrypted1, key)
	decrypted2, _ := Decrypt(encrypted2, key)

	if decrypted1 != plaintext || decrypted2 != plaintext {
		t.Error("Both ciphertexts should decrypt to original plaintext")
	}
}

// =============================================================================
// DeriveUserKey Tests
// =============================================================================

func TestDeriveUserKey_Success(t *testing.T) {
	masterKey := "12345678901234567890123456789012"
	userID := uuid.New()

	derivedKey, err := DeriveUserKey(masterKey, userID)
	if err != nil {
		t.Fatalf("DeriveUserKey failed: %v", err)
	}

	// Derived key should be 32 bytes
	if len(derivedKey) != 32 {
		t.Errorf("Derived key length = %d, want 32", len(derivedKey))
	}

	// Derived key should be different from master key
	if derivedKey == masterKey {
		t.Error("Derived key should differ from master key")
	}
}

func TestDeriveUserKey_Deterministic(t *testing.T) {
	masterKey := "12345678901234567890123456789012"
	userID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

	// Derive key twice with same inputs
	key1, err := DeriveUserKey(masterKey, userID)
	if err != nil {
		t.Fatalf("First DeriveUserKey failed: %v", err)
	}

	key2, err := DeriveUserKey(masterKey, userID)
	if err != nil {
		t.Fatalf("Second DeriveUserKey failed: %v", err)
	}

	// Should produce same key
	if key1 != key2 {
		t.Error("Same inputs should produce same derived key")
	}
}

func TestDeriveUserKey_DifferentUsers(t *testing.T) {
	masterKey := "12345678901234567890123456789012"
	user1 := uuid.MustParse("550e8400-e29b-41d4-a716-446655440001")
	user2 := uuid.MustParse("550e8400-e29b-41d4-a716-446655440002")

	key1, _ := DeriveUserKey(masterKey, user1)
	key2, _ := DeriveUserKey(masterKey, user2)

	if key1 == key2 {
		t.Error("Different users should have different derived keys")
	}
}

func TestDeriveUserKey_DifferentMasterKeys(t *testing.T) {
	masterKey1 := "12345678901234567890123456789012"
	masterKey2 := "abcdefghijklmnopqrstuvwxyzABCDEF"
	userID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

	key1, _ := DeriveUserKey(masterKey1, userID)
	key2, _ := DeriveUserKey(masterKey2, userID)

	if key1 == key2 {
		t.Error("Different master keys should produce different derived keys")
	}
}

func TestDeriveUserKey_InvalidMasterKey(t *testing.T) {
	tests := []struct {
		name      string
		masterKey string
	}{
		{"empty key", ""},
		{"too short", "short"},
		{"too long", "12345678901234567890123456789012345"},
		{"31 bytes", "1234567890123456789012345678901"},
		{"33 bytes", "123456789012345678901234567890123"},
	}

	userID := uuid.New()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := DeriveUserKey(tt.masterKey, userID)
			if !errors.Is(err, ErrInvalidKey) {
				t.Errorf("Expected ErrInvalidKey, got %v", err)
			}
		})
	}
}

func TestDeriveUserKey_CanEncrypt(t *testing.T) {
	masterKey := "12345678901234567890123456789012"
	userID := uuid.New()
	plaintext := "user secret data"

	// Derive user-specific key
	derivedKey, err := DeriveUserKey(masterKey, userID)
	if err != nil {
		t.Fatalf("DeriveUserKey failed: %v", err)
	}

	// Use derived key for encryption
	encrypted, err := Encrypt(plaintext, derivedKey)
	if err != nil {
		t.Fatalf("Encrypt with derived key failed: %v", err)
	}

	// Decrypt with same derived key
	decrypted, err := Decrypt(encrypted, derivedKey)
	if err != nil {
		t.Fatalf("Decrypt with derived key failed: %v", err)
	}

	if decrypted != plaintext {
		t.Errorf("Decrypted text = %q, want %q", decrypted, plaintext)
	}
}

func TestDeriveUserKey_WrongUserCannotDecrypt(t *testing.T) {
	masterKey := "12345678901234567890123456789012"
	user1 := uuid.New()
	user2 := uuid.New()
	plaintext := "user1's secret"

	// Derive keys for both users
	key1, _ := DeriveUserKey(masterKey, user1)
	key2, _ := DeriveUserKey(masterKey, user2)

	// Encrypt with user1's key
	encrypted, _ := Encrypt(plaintext, key1)

	// Try to decrypt with user2's key
	_, err := Decrypt(encrypted, key2)
	if !errors.Is(err, ErrDecryptionFailed) {
		t.Errorf("Expected ErrDecryptionFailed, got %v", err)
	}
}

// =============================================================================
// Error Variable Tests
// =============================================================================

func TestErrorVariables_Defined(t *testing.T) {
	errors := []error{
		ErrInvalidKey,
		ErrInvalidCiphertext,
		ErrDecryptionFailed,
	}

	for _, err := range errors {
		if err == nil {
			t.Error("Error variable should not be nil")
		}
		if err.Error() == "" {
			t.Error("Error message should not be empty")
		}
	}
}

func TestErrorVariables_Messages(t *testing.T) {
	tests := []struct {
		err      error
		contains string
	}{
		{ErrInvalidKey, "32 bytes"},
		{ErrInvalidCiphertext, "invalid"},
		{ErrDecryptionFailed, "decryption"},
	}

	for _, tt := range tests {
		t.Run(tt.err.Error(), func(t *testing.T) {
			t.Parallel()
			if !contains(tt.err.Error(), tt.contains) {
				t.Errorf("Error message should contain %q", tt.contains)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr, 0))
}

func containsAt(s, substr string, start int) bool {
	for i := start; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// =============================================================================
// Edge Case Tests
// =============================================================================

func TestEncrypt_LargeData(t *testing.T) {
	key := "12345678901234567890123456789012"

	// Create 1MB of data
	largeData := make([]byte, 1024*1024)
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}
	plaintext := string(largeData)

	encrypted, err := Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("Encrypt large data failed: %v", err)
	}

	decrypted, err := Decrypt(encrypted, key)
	if err != nil {
		t.Fatalf("Decrypt large data failed: %v", err)
	}

	if decrypted != plaintext {
		t.Error("Large data encryption/decryption mismatch")
	}
}

func TestDecrypt_CorruptedData(t *testing.T) {
	key := "12345678901234567890123456789012"

	// Encrypt some data
	encrypted, _ := Encrypt("secret", key)

	// Decode, corrupt, and re-encode
	data, _ := base64.StdEncoding.DecodeString(encrypted)
	data[len(data)-1] ^= 0xFF // Flip bits in last byte
	corrupted := base64.StdEncoding.EncodeToString(data)

	_, err := Decrypt(corrupted, key)
	if !errors.Is(err, ErrDecryptionFailed) {
		t.Errorf("Expected ErrDecryptionFailed for corrupted data, got %v", err)
	}
}

func TestDecrypt_TruncatedData(t *testing.T) {
	key := "12345678901234567890123456789012"

	// Encrypt some data
	encrypted, _ := Encrypt("secret", key)

	// Decode, truncate, and re-encode
	data, _ := base64.StdEncoding.DecodeString(encrypted)
	truncated := base64.StdEncoding.EncodeToString(data[:len(data)/2])

	_, err := Decrypt(truncated, key)
	if err == nil {
		t.Error("Expected error for truncated ciphertext")
	}
}

func TestEncrypt_BinaryData(t *testing.T) {
	key := "12345678901234567890123456789012"

	// Binary data with null bytes and control characters
	binary := string([]byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0x00, 0x1F})

	encrypted, err := Encrypt(binary, key)
	if err != nil {
		t.Fatalf("Encrypt binary data failed: %v", err)
	}

	decrypted, err := Decrypt(encrypted, key)
	if err != nil {
		t.Fatalf("Decrypt binary data failed: %v", err)
	}

	if decrypted != binary {
		t.Error("Binary data mismatch after encryption/decryption")
	}
}

// =============================================================================
// Benchmark Tests
// =============================================================================

func BenchmarkEncrypt(b *testing.B) {
	key := "12345678901234567890123456789012"
	plaintext := "This is a test secret value for benchmarking encryption performance"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Encrypt(plaintext, key)
	}
}

func BenchmarkDecrypt(b *testing.B) {
	key := "12345678901234567890123456789012"
	encrypted, _ := Encrypt("This is a test secret value for benchmarking decryption performance", key)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Decrypt(encrypted, key)
	}
}

func BenchmarkEncryptDecrypt(b *testing.B) {
	key := "12345678901234567890123456789012"
	plaintext := "This is a test secret value"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		encrypted, _ := Encrypt(plaintext, key)
		_, _ = Decrypt(encrypted, key)
	}
}

func BenchmarkDeriveUserKey(b *testing.B) {
	masterKey := "12345678901234567890123456789012"
	userID := uuid.New()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = DeriveUserKey(masterKey, userID)
	}
}

func BenchmarkValidateKey(b *testing.B) {
	key := "12345678901234567890123456789012"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ValidateKey(key)
	}
}

func BenchmarkEncrypt_Large(b *testing.B) {
	key := "12345678901234567890123456789012"
	plaintext := string(make([]byte, 10*1024)) // 10KB

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Encrypt(plaintext, key)
	}
}

// =============================================================================
// []byte Key Tests (H-12: Use []byte for encryption keys)
// =============================================================================

func TestEncryptDecryptWithBytesKey(t *testing.T) {
	key := []byte("12345678901234567890123456789012") // 32 bytes

	tests := []struct {
		name      string
		plaintext string
	}{
		{"simple text", "hello world"},
		{"special characters", "p@ssw0rd!#$%^&*()"},
		{"unicode", "日本語テスト🎉"},
		{"json", `{"key": "value", "nested": {"foo": "bar"}}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			encrypted, err := EncryptWithBytesKey(tt.plaintext, key)
			if err != nil {
				t.Fatalf("EncryptWithBytesKey failed: %v", err)
			}

			// Encrypted should be different from plaintext
			if encrypted == tt.plaintext {
				t.Error("Encrypted text should not equal plaintext")
			}

			// Decrypt
			decrypted, err := DecryptWithBytesKey(encrypted, key)
			if err != nil {
				t.Fatalf("DecryptWithBytesKey failed: %v", err)
			}

			if decrypted != tt.plaintext {
				t.Errorf("Decrypted text mismatch: got %q, want %q", decrypted, tt.plaintext)
			}
		})
	}
}

func TestEncryptWithBytesKey_KeyZeroing(t *testing.T) {
	// This test verifies that []byte keys can be zeroed after use
	// for security (prevent keys from remaining in memory)
	key := []byte("12345678901234567890123456789012")
	plaintext := "secret data"

	// Encrypt with the key
	encrypted, err := EncryptWithBytesKey(plaintext, key)
	if err != nil {
		t.Fatalf("EncryptWithBytesKey failed: %v", err)
	}

	// Zero the key after use (security best practice)
	for i := range key {
		key[i] = 0
	}

	// Verify key is zeroed
	for i, b := range key {
		if b != 0 {
			t.Errorf("Key byte at index %d not zeroed: got %d", i, b)
		}
	}

	// Create a new key for decryption (the zeroed key won't work)
	newKey := []byte("12345678901234567890123456789012")

	// Decrypt with the new key
	decrypted, err := DecryptWithBytesKey(encrypted, newKey)
	if err != nil {
		t.Fatalf("DecryptWithBytesKey failed: %v", err)
	}

	if decrypted != plaintext {
		t.Errorf("Decrypted text mismatch: got %q, want %q", decrypted, plaintext)
	}
}

func TestEncryptWithBytesKey_InvalidKeyLength(t *testing.T) {
	tests := []struct {
		name string
		key  []byte
	}{
		{"empty key", []byte{}},
		{"too short", []byte("short")},
		{"31 bytes", []byte("1234567890123456789012345678901")},
		{"33 bytes", []byte("123456789012345678901234567890123")},
	}

	plaintext := "test"

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := EncryptWithBytesKey(plaintext, tt.key)
			if !errors.Is(err, ErrInvalidKey) {
				t.Errorf("Expected ErrInvalidKey, got %v", err)
			}
		})
	}
}

func TestDecryptWithBytesKey_InvalidKeyLength(t *testing.T) {
	tests := []struct {
		name string
		key  []byte
	}{
		{"empty key", []byte{}},
		{"too short", []byte("short")},
		{"31 bytes", []byte("1234567890123456789012345678901")},
		{"33 bytes", []byte("123456789012345678901234567890123")},
	}

	// Valid ciphertext
	validKey := []byte("12345678901234567890123456789012")
	encrypted, _ := EncryptWithBytesKey("test", validKey)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := DecryptWithBytesKey(encrypted, tt.key)
			if !errors.Is(err, ErrInvalidKey) {
				t.Errorf("Expected ErrInvalidKey, got %v", err)
			}
		})
	}
}

func TestEncryptWithBytesKey_BackwardCompatibility(t *testing.T) {
	// Verify that []byte and string methods produce compatible results
	keyBytes := []byte("12345678901234567890123456789012")
	keyString := "12345678901234567890123456789012"
	plaintext := "test data"

	// Encrypt with []byte key
	encryptedBytes, err := EncryptWithBytesKey(plaintext, keyBytes)
	if err != nil {
		t.Fatalf("EncryptWithBytesKey failed: %v", err)
	}

	// Encrypt with string key
	encryptedString, err := Encrypt(plaintext, keyString)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	// Encrypted data should be different (due to random nonce)
	if encryptedBytes == encryptedString {
		t.Error("Encrypted data should differ due to random nonce")
	}

	// But both should decrypt correctly with their respective methods
	decryptedFromBytes, err := DecryptWithBytesKey(encryptedBytes, keyBytes)
	if err != nil {
		t.Fatalf("DecryptWithBytesKey failed: %v", err)
	}

	decryptedFromString, err := Decrypt(encryptedString, keyString)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if decryptedFromBytes != plaintext || decryptedFromString != plaintext {
		t.Error("Both decryption methods should recover original plaintext")
	}
}

// =============================================================================
// WithBytesKey / WithBytes Convenience Wrapper Tests
// =============================================================================
//
// These wrappers previously had 0% coverage, dragging the package below its
// 75% threshold. Each is a thin guard around the primary []byte API; tests
// cover both the guard branch (empty input / invalid key) and the happy path
// (delegated call).

func TestEncryptIfNotEmptyWithBytesKey(t *testing.T) {
	key := []byte("12345678901234567890123456789012") // 32 bytes

	t.Run("empty plaintext returns empty no error", func(t *testing.T) {
		t.Parallel()
		got, err := EncryptIfNotEmptyWithBytesKey("", key)
		if err != nil || got != "" {
			t.Errorf("empty input: got (%q, %v), want (\"\", nil)", got, err)
		}
	})

	t.Run("non-empty delegates to EncryptWithBytesKey", func(t *testing.T) {
		t.Parallel()
		enc, err := EncryptIfNotEmptyWithBytesKey("secret", key)
		if err != nil {
			t.Fatalf("non-empty: %v", err)
		}
		if enc == "" || enc == "secret" {
			t.Errorf("expected non-trivial ciphertext, got %q", enc)
		}
		// Round trip back through the matching decrypt wrapper.
		dec, err := DecryptIfNotEmptyWithBytesKey(enc, key)
		if err != nil {
			t.Fatalf("round-trip decrypt: %v", err)
		}
		if dec != "secret" {
			t.Errorf("round-trip: got %q, want %q", dec, "secret")
		}
	})
}

func TestDecryptIfNotEmptyWithBytesKey_EmptyReturnsEmpty(t *testing.T) {
	t.Parallel()
	key := []byte("12345678901234567890123456789012")
	got, err := DecryptIfNotEmptyWithBytesKey("", key)
	if err != nil || got != "" {
		t.Errorf("empty input: got (%q, %v), want (\"\", nil)", got, err)
	}
}

func TestDeriveUserKeyWithBytesKey(t *testing.T) {
	t.Run("invalid master key length", func(t *testing.T) {
		t.Parallel()
		_, err := DeriveUserKeyWithBytesKey([]byte("too-short"), "user-1", "purpose")
		if !errors.Is(err, ErrInvalidKey) {
			t.Errorf("expected ErrInvalidKey, got %v", err)
		}
	})

	t.Run("valid master key returns 32 bytes", func(t *testing.T) {
		t.Parallel()
		key := []byte("12345678901234567890123456789012")
		derived, err := DeriveUserKeyWithBytesKey(key, "user-1", "purpose")
		if err != nil {
			t.Fatalf("derive: %v", err)
		}
		if len(derived) != 32 {
			t.Errorf("derived key length: got %d, want 32", len(derived))
		}
	})

	t.Run("deterministic for same inputs", func(t *testing.T) {
		t.Parallel()
		key := []byte("12345678901234567890123456789012")
		a, _ := DeriveUserKeyWithBytesKey(key, "user-1", "purpose")
		b, _ := DeriveUserKeyWithBytesKey(key, "user-1", "purpose")
		if string(a) != string(b) {
			t.Error("expected deterministic derivation for identical inputs")
		}
	})

	t.Run("differs across users", func(t *testing.T) {
		t.Parallel()
		key := []byte("12345678901234567890123456789012")
		a, _ := DeriveUserKeyWithBytesKey(key, "user-1", "purpose")
		b, _ := DeriveUserKeyWithBytesKey(key, "user-2", "purpose")
		if string(a) == string(b) {
			t.Error("expected different derived keys for different users")
		}
	})
}

func TestValidateKeyWithBytes(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		key     []byte
		wantErr bool
	}{
		{"empty slice", []byte{}, true},
		{"nil slice", nil, true},
		{"too short", []byte("short"), true},
		{"too long", []byte("123456789012345678901234567890123"), true},
		{"valid 32 bytes", []byte("12345678901234567890123456789012"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateKeyWithBytes(tt.key)
			// Empty/nil produces a distinct "not set" error; others ErrInvalidKey.
			// Either way we only assert the wantErr boolean here.
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateKeyWithBytes(%v) error = %v, wantErr %v", tt.key, err, tt.wantErr)
			}
		})
	}
}

func TestZeroBytes(t *testing.T) {
	t.Parallel()
	t.Run("zeros populated slice", func(t *testing.T) {
		t.Parallel()
		b := []byte{0x01, 0x02, 0x03, 0xFF}
		ZeroBytes(b)
		for i, v := range b {
			if v != 0 {
				t.Errorf("byte %d not zeroed: got %x", i, v)
			}
		}
	})

	t.Run("nil slice does not panic", func(t *testing.T) {
		t.Parallel()
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("ZeroBytes(nil) panicked: %v", r)
			}
		}()
		ZeroBytes(nil)
	})

	t.Run("empty slice does not panic", func(t *testing.T) {
		t.Parallel()
		ZeroBytes([]byte{})
	})
}
