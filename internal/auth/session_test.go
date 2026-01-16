package auth

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// hashToken Function Tests
// =============================================================================

func TestHashToken_Consistency(t *testing.T) {
	// Same input should always produce same hash
	token := "test-token-12345"

	hash1 := hashToken(token)
	hash2 := hashToken(token)

	assert.Equal(t, hash1, hash2)
	assert.NotEmpty(t, hash1)
}

func TestHashToken_DifferentInputs(t *testing.T) {
	// Different inputs should produce different hashes
	token1 := "token-1"
	token2 := "token-2"

	hash1 := hashToken(token1)
	hash2 := hashToken(token2)

	assert.NotEqual(t, hash1, hash2)
}

func TestHashToken_EmptyString(t *testing.T) {
	// Empty string should produce a valid hash
	hash := hashToken("")

	assert.NotEmpty(t, hash)
}

func TestHashToken_LongToken(t *testing.T) {
	// Long tokens should produce valid hashes
	longToken := "a"
	for i := 0; i < 1000; i++ {
		longToken += "a"
	}

	hash := hashToken(longToken)

	assert.NotEmpty(t, hash)
	// SHA-256 produces 32 bytes, base64 encoded = 43-44 chars
	assert.LessOrEqual(t, len(hash), 50)
}

func TestHashToken_SpecialCharacters(t *testing.T) {
	tokens := []string{
		"token!@#$%^&*()",
		"token with spaces",
		"token\nwith\nnewlines",
		"token\twith\ttabs",
		"émoji🎉token",
	}

	for _, token := range tokens {
		t.Run(token[:10], func(t *testing.T) {
			hash := hashToken(token)
			assert.NotEmpty(t, hash)

			// Verify consistency
			hash2 := hashToken(token)
			assert.Equal(t, hash, hash2)
		})
	}
}

// =============================================================================
// MockSessionRepository Tests
// =============================================================================

func TestMockSessionRepository_Create(t *testing.T) {
	repo := NewMockSessionRepository()
	ctx := context.Background()

	session, err := repo.Create(ctx, "user-123", "access-token", "refresh-token", time.Now().Add(time.Hour))

	require.NoError(t, err)
	assert.NotEmpty(t, session.ID)
	assert.Equal(t, "user-123", session.UserID)
	assert.Equal(t, "access-token", session.AccessToken)
	assert.Equal(t, "refresh-token", session.RefreshToken)
}

func TestMockSessionRepository_GetByAccessToken(t *testing.T) {
	repo := NewMockSessionRepository()
	ctx := context.Background()

	// Create session
	created, err := repo.Create(ctx, "user-123", "access-token", "refresh-token", time.Now().Add(time.Hour))
	require.NoError(t, err)

	// Get by access token
	retrieved, err := repo.GetByAccessToken(ctx, "access-token")

	require.NoError(t, err)
	assert.Equal(t, created.ID, retrieved.ID)
	assert.Equal(t, created.UserID, retrieved.UserID)
}

func TestMockSessionRepository_GetByAccessToken_NotFound(t *testing.T) {
	repo := NewMockSessionRepository()
	ctx := context.Background()

	retrieved, err := repo.GetByAccessToken(ctx, "nonexistent-token")

	assert.ErrorIs(t, err, ErrSessionNotFound)
	assert.Nil(t, retrieved)
}

func TestMockSessionRepository_GetByRefreshToken(t *testing.T) {
	repo := NewMockSessionRepository()
	ctx := context.Background()

	// Create session
	created, err := repo.Create(ctx, "user-123", "access-token", "refresh-token", time.Now().Add(time.Hour))
	require.NoError(t, err)

	// Get by refresh token
	retrieved, err := repo.GetByRefreshToken(ctx, "refresh-token")

	require.NoError(t, err)
	assert.Equal(t, created.ID, retrieved.ID)
}

func TestMockSessionRepository_GetByRefreshToken_NotFound(t *testing.T) {
	repo := NewMockSessionRepository()
	ctx := context.Background()

	retrieved, err := repo.GetByRefreshToken(ctx, "nonexistent-token")

	assert.ErrorIs(t, err, ErrSessionNotFound)
	assert.Nil(t, retrieved)
}

func TestMockSessionRepository_GetByUserID(t *testing.T) {
	repo := NewMockSessionRepository()
	ctx := context.Background()

	// Create multiple sessions for same user
	_, err := repo.Create(ctx, "user-123", "access-1", "refresh-1", time.Now().Add(time.Hour))
	require.NoError(t, err)
	_, err = repo.Create(ctx, "user-123", "access-2", "refresh-2", time.Now().Add(time.Hour))
	require.NoError(t, err)
	_, err = repo.Create(ctx, "other-user", "access-3", "refresh-3", time.Now().Add(time.Hour))
	require.NoError(t, err)

	// Get sessions for user-123
	sessions, err := repo.GetByUserID(ctx, "user-123")

	require.NoError(t, err)
	assert.Len(t, sessions, 2)
}

func TestMockSessionRepository_GetByUserID_Empty(t *testing.T) {
	repo := NewMockSessionRepository()
	ctx := context.Background()

	sessions, err := repo.GetByUserID(ctx, "nonexistent-user")

	require.NoError(t, err)
	assert.Empty(t, sessions)
}

func TestMockSessionRepository_UpdateTokens_WithValidation(t *testing.T) {
	repo := NewMockSessionRepository()
	ctx := context.Background()

	// Create session
	session, err := repo.Create(ctx, "user-123", "old-access", "old-refresh", time.Now().Add(time.Hour))
	require.NoError(t, err)

	// Update tokens
	newExpiry := time.Now().Add(2 * time.Hour)
	err = repo.UpdateTokens(ctx, session.ID, "new-access", "new-refresh", newExpiry)
	require.NoError(t, err)

	// Verify old tokens no longer work
	_, err = repo.GetByAccessToken(ctx, "old-access")
	assert.ErrorIs(t, err, ErrSessionNotFound)

	// Verify new tokens work
	retrieved, err := repo.GetByAccessToken(ctx, "new-access")
	require.NoError(t, err)
	assert.Equal(t, session.ID, retrieved.ID)
}

func TestMockSessionRepository_UpdateTokens_NotFound(t *testing.T) {
	repo := NewMockSessionRepository()
	ctx := context.Background()

	err := repo.UpdateTokens(ctx, "nonexistent-id", "access", "refresh", time.Now().Add(time.Hour))

	assert.ErrorIs(t, err, ErrSessionNotFound)
}

func TestMockSessionRepository_Delete_WithValidation(t *testing.T) {
	repo := NewMockSessionRepository()
	ctx := context.Background()

	// Create session
	session, err := repo.Create(ctx, "user-123", "access-token", "refresh-token", time.Now().Add(time.Hour))
	require.NoError(t, err)

	// Delete session
	err = repo.Delete(ctx, session.ID)
	require.NoError(t, err)

	// Verify it's deleted
	_, err = repo.GetByAccessToken(ctx, "access-token")
	assert.ErrorIs(t, err, ErrSessionNotFound)
}

func TestMockSessionRepository_Delete_NotFound(t *testing.T) {
	repo := NewMockSessionRepository()
	ctx := context.Background()

	err := repo.Delete(ctx, "nonexistent-id")

	assert.ErrorIs(t, err, ErrSessionNotFound)
}

func TestMockSessionRepository_DeleteByAccessToken(t *testing.T) {
	repo := NewMockSessionRepository()
	ctx := context.Background()

	// Create session
	_, err := repo.Create(ctx, "user-123", "access-token", "refresh-token", time.Now().Add(time.Hour))
	require.NoError(t, err)

	// Delete by access token
	err = repo.DeleteByAccessToken(ctx, "access-token")
	require.NoError(t, err)

	// Verify it's deleted
	_, err = repo.GetByAccessToken(ctx, "access-token")
	assert.ErrorIs(t, err, ErrSessionNotFound)
}

func TestMockSessionRepository_DeleteByUserID_WithValidation(t *testing.T) {
	repo := NewMockSessionRepository()
	ctx := context.Background()

	// Create multiple sessions
	_, err := repo.Create(ctx, "user-123", "access-1", "refresh-1", time.Now().Add(time.Hour))
	require.NoError(t, err)
	_, err = repo.Create(ctx, "user-123", "access-2", "refresh-2", time.Now().Add(time.Hour))
	require.NoError(t, err)

	// Delete all sessions for user
	err = repo.DeleteByUserID(ctx, "user-123")
	require.NoError(t, err)

	// Verify all deleted
	sessions, err := repo.GetByUserID(ctx, "user-123")
	require.NoError(t, err)
	assert.Empty(t, sessions)
}

func TestMockSessionRepository_DeleteExpired_WithValidation(t *testing.T) {
	repo := NewMockSessionRepository()
	ctx := context.Background()

	// Create expired session
	_, err := repo.Create(ctx, "user-123", "expired-access", "expired-refresh", time.Now().Add(-time.Hour))
	require.NoError(t, err)

	// Create valid session
	_, err = repo.Create(ctx, "user-456", "valid-access", "valid-refresh", time.Now().Add(time.Hour))
	require.NoError(t, err)

	// Delete expired
	count, err := repo.DeleteExpired(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)

	// Verify expired is deleted
	_, err = repo.GetByAccessToken(ctx, "expired-access")
	assert.ErrorIs(t, err, ErrSessionNotFound)

	// Verify valid still exists
	_, err = repo.GetByAccessToken(ctx, "valid-access")
	require.NoError(t, err)
}

func TestMockSessionRepository_Count(t *testing.T) {
	repo := NewMockSessionRepository()
	ctx := context.Background()

	// Initially empty
	count, err := repo.Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, count)

	// Add sessions
	_, _ = repo.Create(ctx, "user-1", "access-1", "refresh-1", time.Now().Add(time.Hour))
	_, _ = repo.Create(ctx, "user-2", "access-2", "refresh-2", time.Now().Add(time.Hour))

	count, err = repo.Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}

func TestMockSessionRepository_ConcurrentAccess(t *testing.T) {
	repo := NewMockSessionRepository()
	ctx := context.Background()

	// Create sessions concurrently
	done := make(chan bool, 100)
	for i := 0; i < 100; i++ {
		go func(idx int) {
			userID := "user-" + string(rune('0'+idx%10))
			accessToken := "access-" + string(rune('0'+idx))
			refreshToken := "refresh-" + string(rune('0'+idx))
			_, err := repo.Create(ctx, userID, accessToken, refreshToken, time.Now().Add(time.Hour))
			assert.NoError(t, err)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 100; i++ {
		<-done
	}

	// Verify count
	count, err := repo.Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 100, count)
}
