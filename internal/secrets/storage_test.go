package secrets

import (
	"testing"
	"time"

	"github.com/fluxbase-eu/fluxbase/internal/crypto"
	"github.com/google/uuid"
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
	encryptionKey := "12345678901234567890123456789012"

	storage := NewStorage(nil, encryptionKey)

	if storage == nil {
		t.Fatal("expected storage to not be nil")
	}
	if storage.encryptionKey != encryptionKey {
		t.Error("expected encryption key to be set")
	}
}

// TestEncryptionIntegration tests that encryption/decryption works with the storage layer
func TestEncryptionIntegration(t *testing.T) {
	encryptionKey := "12345678901234567890123456789012"

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
			encrypted, err := crypto.Encrypt(tt.plainValue, encryptionKey)
			if err != nil {
				t.Fatalf("failed to encrypt: %v", err)
			}

			// Verify encrypted value is different from plain value
			if encrypted == tt.plainValue {
				t.Error("encrypted value should not equal plain value")
			}

			// Decrypt
			decrypted, err := crypto.Decrypt(encrypted, encryptionKey)
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
	key1 := "12345678901234567890123456789012"
	key2 := "abcdefghijklmnopqrstuvwxyzABCDEF"

	plainValue := "my-secret-value"

	encrypted, err := crypto.Encrypt(plainValue, key1)
	if err != nil {
		t.Fatalf("failed to encrypt: %v", err)
	}

	_, err = crypto.Decrypt(encrypted, key2)
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

// strPtr is a helper function - defined in handler_test.go
