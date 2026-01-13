package auth

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Helper Function Tests
// =============================================================================

func TestJoinStrings_Empty(t *testing.T) {
	result := joinStrings([]string{}, ", ")
	assert.Equal(t, "", result)
}

func TestJoinStrings_Single(t *testing.T) {
	result := joinStrings([]string{"one"}, ", ")
	assert.Equal(t, "one", result)
}

func TestJoinStrings_Multiple(t *testing.T) {
	result := joinStrings([]string{"one", "two", "three"}, ", ")
	assert.Equal(t, "one, two, three", result)
}

func TestJoinStrings_DifferentSeparators(t *testing.T) {
	tests := []struct {
		input    []string
		sep      string
		expected string
	}{
		{[]string{"a", "b"}, " AND ", "a AND b"},
		{[]string{"x", "y", "z"}, "-", "x-y-z"},
		{[]string{"1", "2"}, "", "12"},
	}

	for _, tt := range tests {
		result := joinStrings(tt.input, tt.sep)
		assert.Equal(t, tt.expected, result)
	}
}

func TestFormatPlaceholder(t *testing.T) {
	tests := []struct {
		column   string
		argNum   int
		expected string
	}{
		{"email", 1, "email = $1"},
		{"name", 2, "name = $2"},
		{"updated_at", 10, "updated_at = $10"},
	}

	for _, tt := range tests {
		result := formatPlaceholder(tt.column, tt.argNum)
		assert.Equal(t, tt.expected, result)
	}
}

// =============================================================================
// MockUserRepository Tests
// =============================================================================

func TestMockUserRepository_Create(t *testing.T) {
	repo := NewMockUserRepository()
	ctx := context.Background()

	req := CreateUserRequest{
		Email: "test@example.com",
		Role:  "authenticated",
		UserMetadata: map[string]interface{}{
			"name": "Test User",
		},
	}

	user, err := repo.Create(ctx, req, "hashed-password")

	require.NoError(t, err)
	assert.NotEmpty(t, user.ID)
	assert.Equal(t, req.Email, user.Email)
	assert.Equal(t, "hashed-password", user.PasswordHash)
	assert.Equal(t, req.Role, user.Role)
	assert.False(t, user.EmailVerified)
}

func TestMockUserRepository_Create_DuplicateEmail(t *testing.T) {
	repo := NewMockUserRepository()
	ctx := context.Background()

	req := CreateUserRequest{Email: "test@example.com"}

	// First creation succeeds
	_, err := repo.Create(ctx, req, "hash1")
	require.NoError(t, err)

	// Second creation with same email fails
	_, err = repo.Create(ctx, req, "hash2")
	assert.ErrorIs(t, err, ErrUserAlreadyExists)
}

func TestMockUserRepository_Create_CustomFn(t *testing.T) {
	repo := NewMockUserRepository()
	ctx := context.Background()

	// Set custom create function
	repo.CreateFn = func(ctx context.Context, req CreateUserRequest, passwordHash string) (*User, error) {
		return nil, assert.AnError
	}

	req := CreateUserRequest{Email: "test@example.com"}
	_, err := repo.Create(ctx, req, "hash")

	assert.Error(t, err)
}

func TestMockUserRepository_GetByID(t *testing.T) {
	repo := NewMockUserRepository()
	ctx := context.Background()

	// Create user
	req := CreateUserRequest{Email: "test@example.com"}
	created, err := repo.Create(ctx, req, "hash")
	require.NoError(t, err)

	// Get by ID
	retrieved, err := repo.GetByID(ctx, created.ID)

	require.NoError(t, err)
	assert.Equal(t, created.ID, retrieved.ID)
	assert.Equal(t, created.Email, retrieved.Email)
}

func TestMockUserRepository_GetByID_NotFound(t *testing.T) {
	repo := NewMockUserRepository()
	ctx := context.Background()

	retrieved, err := repo.GetByID(ctx, "nonexistent-id")

	assert.ErrorIs(t, err, ErrUserNotFound)
	assert.Nil(t, retrieved)
}

func TestMockUserRepository_GetByEmail(t *testing.T) {
	repo := NewMockUserRepository()
	ctx := context.Background()

	// Create user
	req := CreateUserRequest{Email: "test@example.com"}
	created, err := repo.Create(ctx, req, "hash")
	require.NoError(t, err)

	// Get by email
	retrieved, err := repo.GetByEmail(ctx, "test@example.com")

	require.NoError(t, err)
	assert.Equal(t, created.ID, retrieved.ID)
}

func TestMockUserRepository_GetByEmail_NotFound(t *testing.T) {
	repo := NewMockUserRepository()
	ctx := context.Background()

	retrieved, err := repo.GetByEmail(ctx, "nonexistent@example.com")

	assert.ErrorIs(t, err, ErrUserNotFound)
	assert.Nil(t, retrieved)
}

func TestMockUserRepository_List(t *testing.T) {
	repo := NewMockUserRepository()
	ctx := context.Background()

	// Create multiple users
	for i := 0; i < 5; i++ {
		email := "user" + string(rune('0'+i)) + "@example.com"
		req := CreateUserRequest{Email: email}
		_, err := repo.Create(ctx, req, "hash")
		require.NoError(t, err)
	}

	// List with pagination
	users, err := repo.List(ctx, 3, 0)
	require.NoError(t, err)
	assert.Len(t, users, 3)

	// List with offset
	users, err = repo.List(ctx, 10, 3)
	require.NoError(t, err)
	assert.Len(t, users, 2)
}

func TestMockUserRepository_List_Empty(t *testing.T) {
	repo := NewMockUserRepository()
	ctx := context.Background()

	users, err := repo.List(ctx, 10, 0)

	require.NoError(t, err)
	assert.Empty(t, users)
}

func TestMockUserRepository_Update_WithValidation(t *testing.T) {
	repo := NewMockUserRepository()
	ctx := context.Background()

	// Create user
	req := CreateUserRequest{Email: "test@example.com", Role: "user"}
	created, err := repo.Create(ctx, req, "hash")
	require.NoError(t, err)

	// Update user
	newEmail := "updated@example.com"
	newRole := "admin"
	verified := true
	updateReq := UpdateUserRequest{
		Email:         &newEmail,
		Role:          &newRole,
		EmailVerified: &verified,
	}

	updated, err := repo.Update(ctx, created.ID, updateReq)

	require.NoError(t, err)
	assert.Equal(t, newEmail, updated.Email)
	assert.Equal(t, newRole, updated.Role)
	assert.True(t, updated.EmailVerified)
}

func TestMockUserRepository_Update_NotFound(t *testing.T) {
	repo := NewMockUserRepository()
	ctx := context.Background()

	newEmail := "test@example.com"
	updateReq := UpdateUserRequest{Email: &newEmail}

	_, err := repo.Update(ctx, "nonexistent-id", updateReq)

	assert.ErrorIs(t, err, ErrUserNotFound)
}

func TestMockUserRepository_UpdatePassword(t *testing.T) {
	repo := NewMockUserRepository()
	ctx := context.Background()

	// Create user
	req := CreateUserRequest{Email: "test@example.com"}
	created, err := repo.Create(ctx, req, "old-hash")
	require.NoError(t, err)

	// Update password
	err = repo.UpdatePassword(ctx, created.ID, "new-hash")
	require.NoError(t, err)

	// Verify password updated
	retrieved, err := repo.GetByID(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, "new-hash", retrieved.PasswordHash)
}

func TestMockUserRepository_UpdatePassword_NotFound(t *testing.T) {
	repo := NewMockUserRepository()
	ctx := context.Background()

	err := repo.UpdatePassword(ctx, "nonexistent-id", "new-hash")

	assert.ErrorIs(t, err, ErrUserNotFound)
}

func TestMockUserRepository_VerifyEmail(t *testing.T) {
	repo := NewMockUserRepository()
	ctx := context.Background()

	// Create user (email not verified by default)
	req := CreateUserRequest{Email: "test@example.com"}
	created, err := repo.Create(ctx, req, "hash")
	require.NoError(t, err)
	assert.False(t, created.EmailVerified)

	// Verify email
	err = repo.VerifyEmail(ctx, created.ID)
	require.NoError(t, err)

	// Check email is verified
	retrieved, err := repo.GetByID(ctx, created.ID)
	require.NoError(t, err)
	assert.True(t, retrieved.EmailVerified)
}

func TestMockUserRepository_VerifyEmail_NotFound(t *testing.T) {
	repo := NewMockUserRepository()
	ctx := context.Background()

	err := repo.VerifyEmail(ctx, "nonexistent-id")

	assert.ErrorIs(t, err, ErrUserNotFound)
}

func TestMockUserRepository_IncrementFailedLoginAttempts(t *testing.T) {
	repo := NewMockUserRepository()
	ctx := context.Background()

	// Create user
	req := CreateUserRequest{Email: "test@example.com"}
	created, err := repo.Create(ctx, req, "hash")
	require.NoError(t, err)

	// Increment failed attempts
	for i := 0; i < 3; i++ {
		err = repo.IncrementFailedLoginAttempts(ctx, created.ID)
		require.NoError(t, err)
	}

	// Check count
	retrieved, err := repo.GetByID(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, 3, retrieved.FailedLoginAttempts)
}

func TestMockUserRepository_ResetFailedLoginAttempts(t *testing.T) {
	repo := NewMockUserRepository()
	ctx := context.Background()

	// Create user and increment attempts
	req := CreateUserRequest{Email: "test@example.com"}
	created, err := repo.Create(ctx, req, "hash")
	require.NoError(t, err)

	_ = repo.IncrementFailedLoginAttempts(ctx, created.ID)
	_ = repo.IncrementFailedLoginAttempts(ctx, created.ID)

	// Reset
	err = repo.ResetFailedLoginAttempts(ctx, created.ID)
	require.NoError(t, err)

	// Check count is reset
	retrieved, err := repo.GetByID(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, 0, retrieved.FailedLoginAttempts)
}

func TestMockUserRepository_UnlockUser(t *testing.T) {
	repo := NewMockUserRepository()
	ctx := context.Background()

	// Create user and lock it
	req := CreateUserRequest{Email: "test@example.com"}
	created, err := repo.Create(ctx, req, "hash")
	require.NoError(t, err)

	// Manually lock
	repo.mu.Lock()
	repo.users[created.ID].IsLocked = true
	repo.mu.Unlock()

	// Unlock
	err = repo.UnlockUser(ctx, created.ID)
	require.NoError(t, err)

	// Check unlocked
	retrieved, err := repo.GetByID(ctx, created.ID)
	require.NoError(t, err)
	assert.False(t, retrieved.IsLocked)
}

func TestMockUserRepository_Delete_WithValidation(t *testing.T) {
	repo := NewMockUserRepository()
	ctx := context.Background()

	// Create user
	req := CreateUserRequest{Email: "test@example.com"}
	created, err := repo.Create(ctx, req, "hash")
	require.NoError(t, err)

	// Delete
	err = repo.Delete(ctx, created.ID)
	require.NoError(t, err)

	// Verify deleted
	_, err = repo.GetByID(ctx, created.ID)
	assert.ErrorIs(t, err, ErrUserNotFound)

	// Also verify by email
	_, err = repo.GetByEmail(ctx, "test@example.com")
	assert.ErrorIs(t, err, ErrUserNotFound)
}

func TestMockUserRepository_Delete_NotFound(t *testing.T) {
	repo := NewMockUserRepository()
	ctx := context.Background()

	err := repo.Delete(ctx, "nonexistent-id")

	assert.ErrorIs(t, err, ErrUserNotFound)
}

func TestMockUserRepository_Count_WithValidation(t *testing.T) {
	repo := NewMockUserRepository()
	ctx := context.Background()

	// Initially empty
	count, err := repo.Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, count)

	// Add users
	for i := 0; i < 5; i++ {
		email := "user" + string(rune('0'+i)) + "@example.com"
		req := CreateUserRequest{Email: email}
		_, _ = repo.Create(ctx, req, "hash")
	}

	count, err = repo.Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 5, count)
}

func TestMockUserRepository_ConcurrentAccess(t *testing.T) {
	repo := NewMockUserRepository()
	ctx := context.Background()

	// Create users concurrently
	done := make(chan bool, 50)
	for i := 0; i < 50; i++ {
		go func(idx int) {
			email := "user" + string(rune('a'+idx%26)) + string(rune('0'+idx/26)) + "@example.com"
			req := CreateUserRequest{Email: email}
			_, err := repo.Create(ctx, req, "hash")
			assert.NoError(t, err)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 50; i++ {
		<-done
	}

	// Verify count
	count, err := repo.Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 50, count)
}

// =============================================================================
// MockTokenBlacklistRepository Tests
// =============================================================================

func TestMockTokenBlacklistRepository_Add(t *testing.T) {
	repo := NewMockTokenBlacklistRepository()
	ctx := context.Background()

	userID := "user-123"
	err := repo.Add(ctx, "jti-123", &userID, "manual revocation", expiresInHour())

	require.NoError(t, err)
}

func TestMockTokenBlacklistRepository_IsBlacklisted(t *testing.T) {
	repo := NewMockTokenBlacklistRepository()
	ctx := context.Background()

	// Add to blacklist
	userID := "user-123"
	err := repo.Add(ctx, "jti-123", &userID, "reason", expiresInHour())
	require.NoError(t, err)

	// Check if blacklisted
	blacklisted, err := repo.IsBlacklisted(ctx, "jti-123")
	require.NoError(t, err)
	assert.True(t, blacklisted)

	// Non-blacklisted
	blacklisted, err = repo.IsBlacklisted(ctx, "nonexistent-jti")
	require.NoError(t, err)
	assert.False(t, blacklisted)
}

func TestMockTokenBlacklistRepository_IsBlacklisted_Expired(t *testing.T) {
	repo := NewMockTokenBlacklistRepository()
	ctx := context.Background()

	// Add expired entry
	userID := "user-123"
	err := repo.Add(ctx, "jti-expired", &userID, "reason", expiredHourAgo())
	require.NoError(t, err)

	// Should not be blacklisted (expired)
	blacklisted, err := repo.IsBlacklisted(ctx, "jti-expired")
	require.NoError(t, err)
	assert.False(t, blacklisted)
}

func TestMockTokenBlacklistRepository_GetByJTI(t *testing.T) {
	repo := NewMockTokenBlacklistRepository()
	ctx := context.Background()

	// Add to blacklist
	userID := "user-123"
	err := repo.Add(ctx, "jti-123", &userID, "test reason", expiresInHour())
	require.NoError(t, err)

	// Get entry
	entry, err := repo.GetByJTI(ctx, "jti-123")

	require.NoError(t, err)
	assert.Equal(t, "jti-123", entry.TokenJTI)
	assert.Equal(t, userID, entry.RevokedBy)
	assert.Equal(t, "test reason", entry.Reason)
}

func TestMockTokenBlacklistRepository_GetByJTI_NotFound(t *testing.T) {
	repo := NewMockTokenBlacklistRepository()
	ctx := context.Background()

	_, err := repo.GetByJTI(ctx, "nonexistent-jti")

	assert.Error(t, err)
}

func TestMockTokenBlacklistRepository_DeleteExpired_WithValidation(t *testing.T) {
	repo := NewMockTokenBlacklistRepository()
	ctx := context.Background()

	userID := "user-123"

	// Add expired entry
	err := repo.Add(ctx, "jti-expired", &userID, "reason", expiredHourAgo())
	require.NoError(t, err)

	// Add valid entry
	err = repo.Add(ctx, "jti-valid", &userID, "reason", expiresInHour())
	require.NoError(t, err)

	// Delete expired
	count, err := repo.DeleteExpired(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)

	// Verify valid still exists
	_, err = repo.GetByJTI(ctx, "jti-valid")
	require.NoError(t, err)
}

func TestMockTokenBlacklistRepository_DeleteByUser(t *testing.T) {
	repo := NewMockTokenBlacklistRepository()
	ctx := context.Background()

	userID := "user-123"

	// Add entries for user
	_ = repo.Add(ctx, "jti-1", &userID, "reason", expiresInHour())
	_ = repo.Add(ctx, "jti-2", &userID, "reason", expiresInHour())

	// Delete by user
	err := repo.DeleteByUser(ctx, userID)
	require.NoError(t, err)

	// Verify deleted
	blacklisted, _ := repo.IsBlacklisted(ctx, "jti-1")
	assert.False(t, blacklisted)
}

func TestMockTokenBlacklistRepository_AddWithNilUser(t *testing.T) {
	repo := NewMockTokenBlacklistRepository()
	ctx := context.Background()

	// Add with nil user (anonymous token)
	err := repo.Add(ctx, "jti-anon", nil, "anonymous revocation", expiresInHour())
	require.NoError(t, err)

	// Check it's blacklisted
	blacklisted, err := repo.IsBlacklisted(ctx, "jti-anon")
	require.NoError(t, err)
	assert.True(t, blacklisted)

	// Get entry - revokedBy should be empty string
	entry, err := repo.GetByJTI(ctx, "jti-anon")
	require.NoError(t, err)
	assert.Empty(t, entry.RevokedBy)
}

// Helper functions
func expiresInHour() time.Time {
	return time.Now().Add(time.Hour)
}

func expiredHourAgo() time.Time {
	return time.Now().Add(-time.Hour)
}
