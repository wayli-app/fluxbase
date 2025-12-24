package crypto

import (
	"testing"
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
			_, err := Encrypt("test", tt.key)
			if err != ErrInvalidKey {
				t.Errorf("Expected ErrInvalidKey, got %v", err)
			}

			_, err = Decrypt("test", tt.key)
			if err != ErrInvalidKey {
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
	if err != ErrDecryptionFailed {
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
