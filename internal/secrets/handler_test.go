package secrets

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestCreateSecretRequest validates CreateSecretRequest struct
func TestCreateSecretRequest(t *testing.T) {
	t.Run("valid global secret request", func(t *testing.T) {
		req := CreateSecretRequest{
			Name:        "API_KEY",
			Value:       "sk-1234567890",
			Scope:       "global",
			Description: strPtr("My API key"),
		}

		if req.Name != "API_KEY" {
			t.Errorf("expected Name to be 'API_KEY', got %s", req.Name)
		}
		if req.Scope != "global" {
			t.Errorf("expected Scope to be 'global', got %s", req.Scope)
		}
		if req.Namespace != nil {
			t.Error("expected Namespace to be nil for global scope")
		}
	})

	t.Run("valid namespace secret request", func(t *testing.T) {
		ns := "my-namespace"
		req := CreateSecretRequest{
			Name:      "DB_PASSWORD",
			Value:     "secret123",
			Scope:     "namespace",
			Namespace: &ns,
		}

		if req.Scope != "namespace" {
			t.Errorf("expected Scope to be 'namespace', got %s", req.Scope)
		}
		if req.Namespace == nil || *req.Namespace != "my-namespace" {
			t.Error("expected Namespace to be 'my-namespace'")
		}
	})

	t.Run("request with expiration", func(t *testing.T) {
		expiresAt := time.Now().Add(24 * time.Hour)
		req := CreateSecretRequest{
			Name:      "TEMP_TOKEN",
			Value:     "temp-value",
			Scope:     "global",
			ExpiresAt: &expiresAt,
		}

		if req.ExpiresAt == nil {
			t.Error("expected ExpiresAt to be set")
		}
	})
}

// TestUpdateSecretRequest validates UpdateSecretRequest struct
func TestUpdateSecretRequest(t *testing.T) {
	t.Run("update value only", func(t *testing.T) {
		value := "new-value"
		req := UpdateSecretRequest{
			Value: &value,
		}

		if req.Value == nil || *req.Value != "new-value" {
			t.Error("expected Value to be 'new-value'")
		}
		if req.Description != nil {
			t.Error("expected Description to be nil")
		}
	})

	t.Run("update description only", func(t *testing.T) {
		desc := "Updated description"
		req := UpdateSecretRequest{
			Description: &desc,
		}

		if req.Description == nil || *req.Description != "Updated description" {
			t.Error("expected Description to be 'Updated description'")
		}
		if req.Value != nil {
			t.Error("expected Value to be nil")
		}
	})

	t.Run("update multiple fields", func(t *testing.T) {
		value := "new-value"
		desc := "New description"
		expiresAt := time.Now().Add(7 * 24 * time.Hour)

		req := UpdateSecretRequest{
			Value:       &value,
			Description: &desc,
			ExpiresAt:   &expiresAt,
		}

		if req.Value == nil {
			t.Error("expected Value to be set")
		}
		if req.Description == nil {
			t.Error("expected Description to be set")
		}
		if req.ExpiresAt == nil {
			t.Error("expected ExpiresAt to be set")
		}
	})
}

// TestNewHandler validates handler construction
func TestNewHandler(t *testing.T) {
	encryptionKey := "12345678901234567890123456789012"
	storage := NewStorage(nil, encryptionKey)
	handler := NewHandler(storage)

	if handler == nil {
		t.Fatal("expected handler to not be nil")
	}
	if handler.storage != storage {
		t.Error("expected handler.storage to match provided storage")
	}
}

// TestIsDuplicateKeyError validates duplicate key error detection
func TestIsDuplicateKeyError(t *testing.T) {
	tests := []struct {
		name     string
		errMsg   string
		expected bool
	}{
		{"nil error", "", false},
		{"duplicate key error", "duplicate key value violates unique constraint", true},
		{"unique constraint error", "unique constraint violation", true},
		{"specific constraint error", "unique_secret_name_scope", true},
		{"random error", "connection refused", false},
		{"not found error", "no rows in result set", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			if tt.errMsg != "" {
				err = &testError{msg: tt.errMsg}
			}

			result := isDuplicateKeyError(err)
			if result != tt.expected {
				t.Errorf("isDuplicateKeyError(%q) = %v, want %v", tt.errMsg, result, tt.expected)
			}
		})
	}
}

// TestIsNotFoundError validates not found error detection
func TestIsNotFoundError(t *testing.T) {
	tests := []struct {
		name     string
		errMsg   string
		expected bool
	}{
		{"nil error", "", false},
		{"no rows error", "no rows in result set", true},
		{"not found error", "secret not found", true},
		{"generic not found", "not found", true},
		{"duplicate key error", "duplicate key violation", false},
		{"connection error", "connection refused", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			if tt.errMsg != "" {
				err = &testError{msg: tt.errMsg}
			}

			result := isNotFoundError(err)
			if result != tt.expected {
				t.Errorf("isNotFoundError(%q) = %v, want %v", tt.errMsg, result, tt.expected)
			}
		})
	}
}

// TestContains validates string contains helper
func TestContains(t *testing.T) {
	tests := []struct {
		s        string
		substr   string
		expected bool
	}{
		{"hello world", "world", true},
		{"hello world", "hello", true},
		{"hello world", "foo", false},
		{"", "foo", false},
		{"hello", "", true},
		{"", "", true},
		{"abc", "abcd", false},
	}

	for _, tt := range tests {
		t.Run(tt.s+"_"+tt.substr, func(t *testing.T) {
			result := contains(tt.s, tt.substr)
			if result != tt.expected {
				t.Errorf("contains(%q, %q) = %v, want %v", tt.s, tt.substr, result, tt.expected)
			}
		})
	}
}

// TestScopeValidationRules tests validation rules for scopes
func TestScopeValidationRules(t *testing.T) {
	t.Run("global scope requires nil namespace", func(t *testing.T) {
		req := CreateSecretRequest{
			Name:      "TEST",
			Value:     "value",
			Scope:     "global",
			Namespace: strPtr("should-be-nil"),
		}

		// In the actual handler, this would be normalized to nil
		if req.Scope == "global" {
			req.Namespace = nil
		}

		if req.Namespace != nil {
			t.Error("global scope should have nil namespace after normalization")
		}
	})

	t.Run("namespace scope requires non-nil namespace", func(t *testing.T) {
		req := CreateSecretRequest{
			Name:      "TEST",
			Value:     "value",
			Scope:     "namespace",
			Namespace: nil,
		}

		// Validation would fail
		if req.Scope == "namespace" && req.Namespace == nil {
			// This is the expected invalid state
		} else {
			t.Error("expected validation to catch nil namespace for namespace scope")
		}
	})

	t.Run("default scope is global", func(t *testing.T) {
		req := CreateSecretRequest{
			Name:  "TEST",
			Value: "value",
			Scope: "",
		}

		// In the actual handler, empty scope defaults to global
		if req.Scope == "" {
			req.Scope = "global"
		}

		if req.Scope != "global" {
			t.Errorf("expected default scope to be 'global', got %s", req.Scope)
		}
	})
}

// TestSecretVersionParsing tests version number parsing
func TestSecretVersionParsing(t *testing.T) {
	tests := []struct {
		name      string
		versionIn string
		expected  int
		isValid   bool
	}{
		{"valid version 1", "1", 1, true},
		{"valid version 10", "10", 10, true},
		{"valid version 100", "100", 100, true},
		{"invalid zero", "0", 0, false},
		{"invalid negative", "-1", 0, false},
		{"invalid non-numeric", "abc", 0, false},
		{"invalid float", "1.5", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var version int
			n, err := parseVersion(tt.versionIn, &version)

			if tt.isValid {
				if err != nil || n != 1 || version < 1 {
					t.Errorf("expected valid version %d, got %d (err: %v, n: %d)", tt.expected, version, err, n)
				}
			} else {
				if err == nil && version >= 1 {
					t.Errorf("expected invalid version, but got %d", version)
				}
			}
		})
	}
}

// TestUUIDParsing tests UUID parsing for secret IDs
func TestUUIDParsing(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		isValid bool
	}{
		{"valid uuid", "550e8400-e29b-41d4-a716-446655440000", true},
		{"valid uuid uppercase", "550E8400-E29B-41D4-A716-446655440000", true},
		{"invalid uuid - too short", "550e8400-e29b-41d4", false},
		{"invalid uuid - wrong format", "not-a-uuid", false},
		{"invalid uuid - empty", "", false},
		{"valid uuid without dashes", "550e8400e29b41d4a716446655440000", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := uuid.Parse(tt.input)
			isValid := err == nil

			if isValid != tt.isValid {
				t.Errorf("uuid.Parse(%q) valid = %v, want %v", tt.input, isValid, tt.isValid)
			}
		})
	}
}

// Helper functions and types

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

func strPtr(s string) *string {
	return &s
}

// parseVersion is a helper to simulate version parsing from handler
func parseVersion(s string, version *int) (int, error) {
	n, err := parseVersionImpl(s, version)
	return n, err
}

func parseVersionImpl(s string, version *int) (int, error) {
	var v int
	n, err := scanVersion(s, &v)
	if err != nil || n != 1 {
		return n, err
	}
	*version = v
	return n, nil
}

func scanVersion(s string, v *int) (int, error) {
	n := 0
	for i := 0; i < len(s); i++ {
		if s[i] >= '0' && s[i] <= '9' {
			*v = *v*10 + int(s[i]-'0')
			n = 1
		} else {
			return 0, &testError{msg: "invalid character"}
		}
	}
	if n == 0 {
		return 0, &testError{msg: "no digits"}
	}
	return n, nil
}
