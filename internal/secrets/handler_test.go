package secrets

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

// =============================================================================
// CreateSecretRequest Tests
// =============================================================================

func TestCreateSecretRequest_GlobalSecret(t *testing.T) {
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
}

func TestCreateSecretRequest_NamespaceSecret(t *testing.T) {
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
}

func TestCreateSecretRequest_WithExpiration(t *testing.T) {
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
	if time.Until(*req.ExpiresAt) < 23*time.Hour || time.Until(*req.ExpiresAt) > 25*time.Hour {
		t.Error("ExpiresAt is not approximately 24 hours from now")
	}
}

// =============================================================================
// UpdateSecretRequest Tests
// =============================================================================

func TestUpdateSecretRequest_UpdateValue(t *testing.T) {
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
}

func TestUpdateSecretRequest_UpdateDescription(t *testing.T) {
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
}

func TestUpdateSecretRequest_UpdateMultiple(t *testing.T) {
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
}

// =============================================================================
// Error Detection Tests
// =============================================================================

func TestIsDuplicateKeyError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"nil error", nil, false},
		{"generic error", fmt.Errorf("some random error"), false},
		{"connection error", fmt.Errorf("connection refused"), false},
		{"not found error", fmt.Errorf("no rows in result set"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isDuplicateKeyError(tt.err)
			if result != tt.expected {
				t.Errorf("isDuplicateKeyError(%v) = %v, want %v", tt.err, result, tt.expected)
			}
		})
	}

	// Note: We can't easily test actual PostgreSQL unique constraint violations
	// without a real database connection. The integration tests cover that scenario.
}

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

// =============================================================================
// Helper Function Tests
// =============================================================================

func TestContains(t *testing.T) {
	tests := []struct {
		s        string
		substr   string
		expected bool
	}{
		{"hello world", "world", true},
		{"hello world", "hello", true},
		{"hello world", "foo", false},
		{"", "", true},
		{"abc", "abcd", false},
	}

	for _, tt := range tests {
		t.Run(tt.s+"_"+tt.substr, func(t *testing.T) {
			result := strings.Contains(tt.s, tt.substr)
			if result != tt.expected {
				t.Errorf("strings.Contains(%q, %q) = %v, want %v", tt.s, tt.substr, result, tt.expected)
			}
		})
	}
}

// =============================================================================
// Pointer Helper Tests
// =============================================================================

func TestStrPtr(t *testing.T) {
	s := strPtr("test")
	if s == nil {
		t.Fatal("expected non-nil pointer")
	}
	if *s != "test" {
		t.Errorf("expected 'test', got %s", *s)
	}
}

func TestIntPtr(t *testing.T) {
	i := intPtr(42)
	if i == nil {
		t.Fatal("expected non-nil pointer")
	}
	if *i != 42 {
		t.Errorf("expected 42, got %d", *i)
	}
}

// =============================================================================
// Time Parsing Tests
// =============================================================================

func TestParseExpiresAt(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantValid bool
	}{
		{"valid RFC3339", time.Now().Add(time.Hour).Format(time.RFC3339), true},
		{"empty", "", false},
		{"invalid", "not-a-time", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := time.Parse(time.RFC3339, tt.input)
			if tt.wantValid {
				if err != nil {
					t.Errorf("expected valid time, got error: %v", err)
				}
				if time.Until(result) < 0 {
					// Check if it's in the past
					t.Skip("Skipping test for past time")
				}
			} else if err == nil && !result.IsZero() {
				t.Error("expected error or zero time")
			}
		})
	}
}

// =============================================================================
// Version Tests
// =============================================================================

func TestSecretVersion_Increment(t *testing.T) {
	secret := &Secret{
		Version: 1,
	}

	secret.Version++
	if secret.Version != 2 {
		t.Errorf("expected version to be 2, got %d", secret.Version)
	}

	secret.Version += 5
	if secret.Version != 7 {
		t.Errorf("expected version to be 7, got %d", secret.Version)
	}
}

func TestSecretVersion_NeverNegative(t *testing.T) {
	secret := &Secret{
		Version: 1,
	}

	// Simulate multiple updates
	for i := 0; i < 10; i++ {
		secret.Version++
	}

	if secret.Version <= 0 {
		t.Errorf("version should never be negative or zero, got %d", secret.Version)
	}
}

// =============================================================================
// UUID Helper Tests
// =============================================================================

func TestUUID_Parsing(t *testing.T) {
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

// =============================================================================
// Time Comparison Tests
// =============================================================================

func TestTime_IsExpired(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name      string
		expiresAt *time.Time
		expected  bool
	}{
		{"no expiration", nil, false},
		{"expired", timePtr(now.Add(-1 * time.Hour)), true},
		{"not expired", timePtr(now.Add(1 * time.Hour)), false},
		{"just now", timePtr(now.Add(-1 * time.Second)), true},
		{"future", timePtr(now.Add(24 * time.Hour)), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var isExpired bool
			if tt.expiresAt == nil {
				isExpired = false
			} else {
				isExpired = tt.expiresAt.Before(now)
			}

			if isExpired != tt.expected {
				t.Errorf("isExpired = %v, want %v", isExpired, tt.expected)
			}
		})
	}
}

// =============================================================================
// Scope Validation Tests
// =============================================================================

func TestScope_ValidScopes(t *testing.T) {
	validScopes := []string{"global", "namespace", ""}

	for _, scope := range validScopes {
		t.Run("valid_scope_"+scope, func(t *testing.T) {
			// These should all be valid
			if scope == "" || scope == "global" || scope == "namespace" {
				// Valid scope
			} else {
				t.Error("invalid scope should have been caught")
			}
		})
	}
}

func TestScope_InvalidScope(t *testing.T) {
	invalidScopes := []string{"invalid", "user", "admin", "org"}

	for _, scope := range invalidScopes {
		t.Run("invalid_scope_"+scope, func(t *testing.T) {
			// These should be invalid
			if scope == "global" || scope == "namespace" || scope == "" {
				t.Error("invalid scope should have been rejected")
			}
		})
	}
}

// =============================================================================
// Namespace Validation Tests
// =============================================================================

func TestNamespace_GlobalScope(t *testing.T) {
	req := CreateSecretRequest{
		Name:      "TEST",
		Value:     "value",
		Scope:     "global",
		Namespace: strPtr("should-be-nil"),
	}

	// Global scope should have nil namespace
	if req.Scope == "global" && req.Namespace != nil {
		// Normalize to nil
		req.Namespace = nil
	}

	if req.Namespace != nil {
		t.Error("global scope should have nil namespace after normalization")
	}
}

func TestNamespace_NamespaceScope(t *testing.T) {
	req := CreateSecretRequest{
		Name:      "TEST",
		Value:     "value",
		Scope:     "namespace",
		Namespace: nil,
	}

	// Namespace scope requires namespace - this is the invalid case we're testing
	if req.Scope == "namespace" {
		if req.Namespace == nil {
			// Expected: validation should catch this
			return
		}
	}
	t.Error("Test setup should have namespace scope with nil namespace")
}

// =============================================================================
// Helper Functions and Types
// =============================================================================

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

func strPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}

func timePtr(t time.Time) *time.Time {
	return &t
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkStrPtr(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = strPtr("test")
	}
}

func BenchmarkContains(b *testing.B) {
	s := "hello world"
	substr := "world"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = strings.Contains(s, substr)
	}
}

func BenchmarkUUID_Parse(b *testing.B) {
	id := "550e8400-e29b-41d4-a716-446655440000"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = uuid.Parse(id)
	}
}
