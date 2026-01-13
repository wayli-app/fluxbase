package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// generateSecurePassword Tests
// =============================================================================

func TestGenerateSecurePassword_Length(t *testing.T) {
	tests := []int{8, 12, 16, 24, 32}

	for _, length := range tests {
		t.Run("length_"+string(rune('0'+length/10))+string(rune('0'+length%10)), func(t *testing.T) {
			password, err := generateSecurePassword(length)

			require.NoError(t, err)
			assert.Len(t, password, length)
		})
	}
}

func TestGenerateSecurePassword_Uniqueness(t *testing.T) {
	// Generate multiple passwords and ensure they're different
	passwords := make(map[string]bool)

	for i := 0; i < 100; i++ {
		password, err := generateSecurePassword(16)
		require.NoError(t, err)

		// Should not have seen this password before
		assert.False(t, passwords[password], "Password collision detected")
		passwords[password] = true
	}

	// Should have 100 unique passwords
	assert.Len(t, passwords, 100)
}

func TestGenerateSecurePassword_NotEmpty(t *testing.T) {
	password, err := generateSecurePassword(8)

	require.NoError(t, err)
	assert.NotEmpty(t, password)
}

func TestGenerateSecurePassword_Printable(t *testing.T) {
	// Base64 URL encoding should produce printable characters
	for i := 0; i < 10; i++ {
		password, err := generateSecurePassword(16)
		require.NoError(t, err)

		for _, c := range password {
			// Base64 URL safe characters: A-Z, a-z, 0-9, -, _
			isAlphaNum := (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9')
			isUrlSafe := c == '-' || c == '_'
			assert.True(t, isAlphaNum || isUrlSafe, "Non-printable character found: %c", c)
		}
	}
}

func TestGenerateSecurePassword_MinimumLength(t *testing.T) {
	// Even with length 1, should work
	password, err := generateSecurePassword(1)

	require.NoError(t, err)
	assert.Len(t, password, 1)
}

// =============================================================================
// EnrichedUser Type Tests
// =============================================================================

func TestEnrichedUser_FieldsExist(t *testing.T) {
	// Test that EnrichedUser has all expected fields
	user := EnrichedUser{
		ID:             "user-123",
		Email:          "test@example.com",
		EmailVerified:  true,
		Role:           "admin",
		Provider:       "email",
		ActiveSessions: 2,
		LastSignIn:     nil,
		IsLocked:       false,
		UserMetadata:   map[string]interface{}{"name": "Test"},
		AppMetadata:    map[string]interface{}{"plan": "pro"},
	}

	assert.Equal(t, "user-123", user.ID)
	assert.Equal(t, "test@example.com", user.Email)
	assert.True(t, user.EmailVerified)
	assert.Equal(t, "admin", user.Role)
	assert.Equal(t, "email", user.Provider)
	assert.Equal(t, 2, user.ActiveSessions)
	assert.False(t, user.IsLocked)
}

// =============================================================================
// InviteUserRequest Tests
// =============================================================================

func TestInviteUserRequest_Defaults(t *testing.T) {
	req := InviteUserRequest{
		Email: "new@example.com",
	}

	// Email should be set, role should be empty (defaults applied in service)
	assert.Equal(t, "new@example.com", req.Email)
	assert.Empty(t, req.Role)
	assert.Empty(t, req.Password)
}

func TestInviteUserRequest_WithPassword(t *testing.T) {
	req := InviteUserRequest{
		Email:    "new@example.com",
		Role:     "admin",
		Password: "custom-password",
	}

	assert.Equal(t, "new@example.com", req.Email)
	assert.Equal(t, "admin", req.Role)
	assert.Equal(t, "custom-password", req.Password)
}

// =============================================================================
// UpdateAdminUserRequest Tests
// =============================================================================

func TestUpdateAdminUserRequest_AllFields(t *testing.T) {
	email := "updated@example.com"
	role := "superadmin"
	password := "new-password"

	req := UpdateAdminUserRequest{
		Email:        &email,
		Role:         &role,
		Password:     &password,
		UserMetadata: map[string]interface{}{"key": "value"},
	}

	assert.NotNil(t, req.Email)
	assert.Equal(t, "updated@example.com", *req.Email)
	assert.NotNil(t, req.Role)
	assert.Equal(t, "superadmin", *req.Role)
	assert.NotNil(t, req.Password)
	assert.Equal(t, "new-password", *req.Password)
	assert.NotNil(t, req.UserMetadata)
}

func TestUpdateAdminUserRequest_PartialUpdate(t *testing.T) {
	email := "updated@example.com"

	req := UpdateAdminUserRequest{
		Email: &email,
		// Other fields are nil
	}

	assert.NotNil(t, req.Email)
	assert.Nil(t, req.Role)
	assert.Nil(t, req.Password)
	assert.Nil(t, req.UserMetadata)
}

// =============================================================================
// Benchmark Tests
// =============================================================================

func BenchmarkGenerateSecurePassword(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = generateSecurePassword(16)
	}
}

func BenchmarkGenerateSecurePassword_Long(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = generateSecurePassword(64)
	}
}
