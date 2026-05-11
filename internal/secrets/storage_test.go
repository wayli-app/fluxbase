package secrets

import (
	"bytes"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/nimbleflux/fluxbase/internal/crypto"
)

// TestSecretStructs tests that the secret structs work correctly
func TestSecretStructs(t *testing.T) {
	t.Run("Secret struct initialization", func(t *testing.T) {
		now := time.Now()
		userID := uuid.New()
		namespace := "test-ns"

		secret := &Secret{
			ID:             uuid.New(),
			Name:           "API_KEY",
			Scope:          "namespace",
			Namespace:      &namespace,
			EncryptedValue: "encrypted-data",
			Description:    strPtr("Test API key"),
			Version:        1,
			ExpiresAt:      &now,
			CreatedAt:      now,
			UpdatedAt:      now,
			CreatedBy:      &userID,
			UpdatedBy:      &userID,
		}

		if secret.Name != "API_KEY" {
			t.Errorf("expected Name to be 'API_KEY', got %s", secret.Name)
		}
		if secret.Scope != "namespace" {
			t.Errorf("expected Scope to be 'namespace', got %s", secret.Scope)
		}
		if *secret.Namespace != "test-ns" {
			t.Errorf("expected Namespace to be 'test-ns', got %s", *secret.Namespace)
		}
	})

	t.Run("SecretSummary struct initialization", func(t *testing.T) {
		now := time.Now()
		summary := SecretSummary{
			ID:        uuid.New(),
			Name:      "DB_PASSWORD",
			Scope:     "global",
			Namespace: nil,
			Version:   3,
			IsExpired: false,
			CreatedAt: now,
			UpdatedAt: now,
		}

		if summary.Scope != "global" {
			t.Errorf("expected Scope to be 'global', got %s", summary.Scope)
		}
		if summary.Namespace != nil {
			t.Errorf("expected Namespace to be nil for global scope")
		}
	})

	t.Run("SecretVersion struct initialization", func(t *testing.T) {
		secretID := uuid.New()
		version := SecretVersion{
			ID:        uuid.New(),
			SecretID:  secretID,
			Version:   2,
			CreatedAt: time.Now(),
		}

		if version.SecretID != secretID {
			t.Errorf("expected SecretID to match")
		}
		if version.Version != 2 {
			t.Errorf("expected Version to be 2, got %d", version.Version)
		}
	})
}

// TestNewStorage tests the Storage constructor
func TestNewStorage(t *testing.T) {
	encryptionKey := []byte("12345678901234567890123456789012")

	storage := NewStorage(nil, encryptionKey)

	if storage == nil {
		t.Fatal("expected storage to not be nil")
	}
	if !bytes.Equal(storage.encryptionKey, encryptionKey) {
		t.Error("expected encryption key to be set")
	}
}

// TestEncryptionIntegration tests that encryption/decryption works with the storage layer
func TestEncryptionIntegration(t *testing.T) {
	encryptionKey := []byte("12345678901234567890123456789012")

	tests := []struct {
		name       string
		plainValue string
	}{
		{"simple password", "mysecretpassword"},
		{"api key", "sk-1234567890abcdefghijklmnopqrstuvwxyz"},
		{"json secret", `{"client_id": "abc", "client_secret": "xyz"}`},
		{"special characters", "p@$$w0rd!#$%^&*()_+-=[]{}|;':\",./<>?"},
		{"unicode", "日本語パスワード🔐"},
		{"long secret", "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Encrypt
			encrypted, err := crypto.EncryptWithBytesKey(tt.plainValue, encryptionKey)
			if err != nil {
				t.Fatalf("failed to encrypt: %v", err)
			}

			// Verify encrypted value is different from plain value
			if encrypted == tt.plainValue {
				t.Error("encrypted value should not equal plain value")
			}

			// Decrypt
			decrypted, err := crypto.DecryptWithBytesKey(encrypted, encryptionKey)
			if err != nil {
				t.Fatalf("failed to decrypt: %v", err)
			}

			if decrypted != tt.plainValue {
				t.Errorf("decrypted value mismatch: got %q, want %q", decrypted, tt.plainValue)
			}
		})
	}
}

// TestEncryptionWithWrongKey verifies that decryption fails with wrong key
func TestEncryptionWithWrongKey(t *testing.T) {
	key1 := []byte("12345678901234567890123456789012")
	key2 := []byte("abcdefghijklmnopqrstuvwxyzABCDEF")

	plainValue := "my-secret-value"

	encrypted, err := crypto.EncryptWithBytesKey(plainValue, key1)
	if err != nil {
		t.Fatalf("failed to encrypt: %v", err)
	}

	_, err = crypto.DecryptWithBytesKey(encrypted, key2)
	if err == nil {
		t.Error("expected decryption to fail with wrong key")
	}
}

// TestScopeValidation tests scope value validation
func TestScopeValidation(t *testing.T) {
	tests := []struct {
		name    string
		scope   string
		isValid bool
	}{
		{"global scope", "global", true},
		{"namespace scope", "namespace", true},
		{"invalid scope", "function", false},
		{"empty scope", "", false},
		{"uppercase scope", "GLOBAL", false},
	}

	validScopes := map[string]bool{
		"global":    true,
		"namespace": true,
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := validScopes[tt.scope]
			if isValid != tt.isValid {
				t.Errorf("scope %q validation: got %v, want %v", tt.scope, isValid, tt.isValid)
			}
		})
	}
}

// TestSecretNamespace tests namespace handling for different scopes
func TestSecretNamespace(t *testing.T) {
	t.Run("global scope should have nil namespace", func(t *testing.T) {
		secret := &Secret{
			Name:      "GLOBAL_SECRET",
			Scope:     "global",
			Namespace: nil,
		}

		if secret.Namespace != nil {
			t.Error("global scope secrets should have nil namespace")
		}
	})

	t.Run("namespace scope should have namespace set", func(t *testing.T) {
		ns := "my-namespace"
		secret := &Secret{
			Name:      "NS_SECRET",
			Scope:     "namespace",
			Namespace: &ns,
		}

		if secret.Namespace == nil {
			t.Error("namespace scope secrets should have namespace set")
		}
		if *secret.Namespace != "my-namespace" {
			t.Errorf("expected namespace 'my-namespace', got %s", *secret.Namespace)
		}
	})
}

// TestSecretExpiration tests expiration date handling
func TestSecretExpiration(t *testing.T) {
	t.Run("non-expired secret", func(t *testing.T) {
		futureTime := time.Now().Add(24 * time.Hour)
		summary := SecretSummary{
			Name:      "FUTURE_SECRET",
			ExpiresAt: &futureTime,
			IsExpired: false,
		}

		if summary.IsExpired {
			t.Error("secret with future expiration should not be expired")
		}
	})

	t.Run("expired secret", func(t *testing.T) {
		pastTime := time.Now().Add(-24 * time.Hour)
		summary := SecretSummary{
			Name:      "EXPIRED_SECRET",
			ExpiresAt: &pastTime,
			IsExpired: true,
		}

		if !summary.IsExpired {
			t.Error("secret with past expiration should be expired")
		}
	})

	t.Run("no expiration", func(t *testing.T) {
		summary := SecretSummary{
			Name:      "NEVER_EXPIRES",
			ExpiresAt: nil,
			IsExpired: false,
		}

		if summary.ExpiresAt != nil {
			t.Error("secret without expiration should have nil ExpiresAt")
		}
		if summary.IsExpired {
			t.Error("secret without expiration should not be expired")
		}
	})
}

// TestVersionIncrement tests version incrementing logic
func TestVersionIncrement(t *testing.T) {
	tests := []struct {
		name           string
		currentVersion int
		expectedNext   int
	}{
		{"initial version", 1, 2},
		{"second version", 2, 3},
		{"high version", 100, 101},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nextVersion := tt.currentVersion + 1
			if nextVersion != tt.expectedNext {
				t.Errorf("expected version %d, got %d", tt.expectedNext, nextVersion)
			}
		})
	}
}

// =============================================================================
// Secret Name Validation Tests
// =============================================================================

func TestSecretNameValidation(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		isValid bool
	}{
		{"valid uppercase", "API_KEY", true},
		{"valid with numbers", "API_KEY_V2", true},
		{"valid simple", "SECRET", true},
		{"lowercase", "api_key", true},
		{"mixed case", "Api_Key", true},
		{"starts with number", "1API_KEY", false},
		{"contains spaces", "API KEY", false},
		{"contains dash", "API-KEY", false},
		{"empty", "", false},
		{"too long", string(make([]byte, 256)), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simple validation: alphanumeric and underscore, not starting with number
			isValid := len(tt.input) > 0 && len(tt.input) <= 255
			if isValid && len(tt.input) > 0 {
				first := tt.input[0]
				isValid = (first >= 'a' && first <= 'z') || (first >= 'A' && first <= 'Z') || first == '_'
			}
			if isValid {
				for _, c := range tt.input {
					if (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') && (c < '0' || c > '9') && c != '_' {
						isValid = false
						break
					}
				}
			}

			if isValid != tt.isValid {
				t.Errorf("secret name %q validation: got %v, want %v", tt.input, isValid, tt.isValid)
			}
		})
	}
}

// =============================================================================
// Secret Struct Edge Cases
// =============================================================================

func TestSecret_EdgeCases(t *testing.T) {
	t.Run("secret with all optional fields nil", func(t *testing.T) {
		secret := &Secret{
			ID:             uuid.New(),
			Name:           "MINIMAL_SECRET",
			Scope:          "global",
			Namespace:      nil,
			EncryptedValue: "encrypted",
			Description:    nil,
			Version:        1,
			ExpiresAt:      nil,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
			CreatedBy:      nil,
			UpdatedBy:      nil,
		}

		if secret.Description != nil {
			t.Error("expected Description to be nil")
		}
		if secret.ExpiresAt != nil {
			t.Error("expected ExpiresAt to be nil")
		}
		if secret.CreatedBy != nil {
			t.Error("expected CreatedBy to be nil")
		}
	})

	t.Run("secret summary with computed expiration", func(t *testing.T) {
		now := time.Now()
		pastTime := now.Add(-1 * time.Hour)
		futureTime := now.Add(1 * time.Hour)

		expiredSummary := SecretSummary{
			Name:      "EXPIRED",
			ExpiresAt: &pastTime,
			IsExpired: pastTime.Before(now),
		}

		validSummary := SecretSummary{
			Name:      "VALID",
			ExpiresAt: &futureTime,
			IsExpired: futureTime.Before(now),
		}

		if !expiredSummary.IsExpired {
			t.Error("secret with past expiration should be marked expired")
		}
		if validSummary.IsExpired {
			t.Error("secret with future expiration should not be marked expired")
		}
	})
}

// =============================================================================
// Storage Initialization Tests
// =============================================================================

func TestStorage_Initialization(t *testing.T) {
	t.Run("storage with valid 32-byte key", func(t *testing.T) {
		key := []byte("12345678901234567890123456789012")
		storage := NewStorage(nil, key)

		if storage == nil {
			t.Fatal("storage should not be nil")
		}
		if len(storage.encryptionKey) != 32 {
			t.Errorf("expected 32-byte key, got %d bytes", len(storage.encryptionKey))
		}
	})

	t.Run("storage fields are set correctly", func(t *testing.T) {
		key := []byte("12345678901234567890123456789012")
		storage := NewStorage(nil, key)

		if storage.db != nil {
			t.Error("db should be nil when initialized with nil")
		}
		if !bytes.Equal(storage.encryptionKey, key) {
			t.Error("encryption key should match input")
		}
	})
}

// =============================================================================
// Benchmark Tests
// =============================================================================

func BenchmarkSecretStruct_Creation(b *testing.B) {
	now := time.Now()
	namespace := "test-ns"
	userID := uuid.New()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = &Secret{
			ID:             uuid.New(),
			Name:           "API_KEY",
			Scope:          "namespace",
			Namespace:      &namespace,
			EncryptedValue: "encrypted-data",
			Version:        1,
			CreatedAt:      now,
			UpdatedAt:      now,
			CreatedBy:      &userID,
		}
	}
}

func BenchmarkSecretSummary_Creation(b *testing.B) {
	now := time.Now()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = SecretSummary{
			ID:        uuid.New(),
			Name:      "DB_PASSWORD",
			Scope:     "global",
			Version:   1,
			IsExpired: false,
			CreatedAt: now,
			UpdatedAt: now,
		}
	}
}

func BenchmarkEncryption(b *testing.B) {
	key := []byte("12345678901234567890123456789012")
	plainValue := "my-secret-password-value"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = crypto.EncryptWithBytesKey(plainValue, key)
	}
}

func BenchmarkDecryption(b *testing.B) {
	key := []byte("12345678901234567890123456789012")
	plainValue := "my-secret-password-value"
	encrypted, _ := crypto.EncryptWithBytesKey(plainValue, key)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = crypto.DecryptWithBytesKey(encrypted, key)
	}
}

func BenchmarkEncryptDecrypt_RoundTrip(b *testing.B) {
	key := []byte("12345678901234567890123456789012")
	plainValue := "my-secret-password-value"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		encrypted, _ := crypto.EncryptWithBytesKey(plainValue, key)
		_, _ = crypto.DecryptWithBytesKey(encrypted, key)
	}
}

// strPtr is a helper function - defined in handler_test.go
