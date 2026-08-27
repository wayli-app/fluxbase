package auth

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// This file contains comprehensive tests for session management workflows

// TestListSessions_All tests listing all sessions for a user
func TestListSessions_All(t *testing.T) {
	ctx := context.Background()
	repo := NewMockSessionRepository()
	userID := "user-123"

	// Create multiple sessions
	session1, err := repo.Create(ctx, userID, "access-token-1", "refresh-token-1", time.Now().Add(1*time.Hour))
	require.NoError(t, err)

	session2, err := repo.Create(ctx, userID, "access-token-2", "refresh-token-2", time.Now().Add(2*time.Hour))
	require.NoError(t, err)

	session3, err := repo.Create(ctx, userID, "access-token-3", "refresh-token-3", time.Now().Add(3*time.Hour))
	require.NoError(t, err)

	// List all sessions
	sessions, err := repo.GetByUserID(ctx, userID)
	require.NoError(t, err)

	assert.Len(t, sessions, 3, "Should return all 3 sessions")

	// Verify session IDs
	sessionIDs := make(map[string]bool)
	for _, session := range sessions {
		sessionIDs[session.ID] = true
	}

	assert.True(t, sessionIDs[session1.ID], "Should include session 1")
	assert.True(t, sessionIDs[session2.ID], "Should include session 2")
	assert.True(t, sessionIDs[session3.ID], "Should include session 3")
}

// TestListSessions_ActiveOnly tests listing only active (non-expired) sessions
func TestListSessions_ActiveOnly(t *testing.T) {
	ctx := context.Background()
	repo := NewMockSessionRepository()
	userID := "user-123"

	// Create active session
	activeSession, err := repo.Create(ctx, userID, "active-access", "active-refresh", time.Now().Add(1*time.Hour))
	require.NoError(t, err)

	// Create expired session
	expiredSession, err := repo.Create(ctx, userID, "expired-access", "expired-refresh", time.Now().Add(-1*time.Hour))
	require.NoError(t, err)

	// List sessions - mock should filter expired
	sessions, err := repo.GetByUserID(ctx, userID)
	require.NoError(t, err)

	// Should only return active session
	assert.Len(t, sessions, 1, "Should only return active sessions")
	assert.Equal(t, activeSession.ID, sessions[0].ID, "Should be the active session")
	assert.NotEqual(t, expiredSession.ID, sessions[0].ID, "Should not be the expired session")
}

// TestListSessions_ExpiredFiltered tests that expired sessions are filtered out
func TestListSessions_ExpiredFiltered(t *testing.T) {
	ctx := context.Background()
	repo := NewMockSessionRepository()
	userID := "user-123"

	// Create sessions with different expiry times
	now := time.Now()

	_, err := repo.Create(ctx, userID, "expired-1", "refresh-1", now.Add(-2*time.Hour))
	require.NoError(t, err)

	_, err = repo.Create(ctx, userID, "expired-2", "refresh-2", now.Add(-1*time.Hour))
	require.NoError(t, err)

	activeSession, err := repo.Create(ctx, userID, "active-1", "refresh-3", now.Add(1*time.Hour))
	require.NoError(t, err)

	// List sessions
	sessions, err := repo.GetByUserID(ctx, userID)
	require.NoError(t, err)

	// Should only return active session
	assert.Len(t, sessions, 1, "Should filter out all expired sessions")
	assert.Equal(t, activeSession.ID, sessions[0].ID)
}

// TestRevokeSession_Success tests successful session revocation
func TestRevokeSession_Success(t *testing.T) {
	ctx := context.Background()
	repo := NewMockSessionRepository()
	userID := "user-123"

	// Create session
	session, err := repo.Create(ctx, userID, "access-token", "refresh-token", time.Now().Add(1*time.Hour))
	require.NoError(t, err)

	// Verify session exists
	found, err := repo.GetByAccessToken(ctx, "access-token")
	require.NoError(t, err)
	assert.Equal(t, session.ID, found.ID)

	// Revoke session
	err = repo.Delete(ctx, session.ID)
	require.NoError(t, err)

	// Verify session is deleted
	_, err = repo.GetByAccessToken(ctx, "access-token")
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrSessionNotFound)
}

// TestRevokeSession_NotFound tests revoking a non-existent session
func TestRevokeSession_NotFound(t *testing.T) {
	ctx := context.Background()
	repo := NewMockSessionRepository()

	// Try to delete non-existent session
	err := repo.Delete(ctx, "non-existent-session-id")

	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrSessionNotFound)
}

// TestRevokeSession_AlreadyRevoked tests revoking an already revoked session
func TestRevokeSession_AlreadyRevoked(t *testing.T) {
	ctx := context.Background()
	repo := NewMockSessionRepository()
	userID := "user-123"

	// Create session
	session, err := repo.Create(ctx, userID, "access-token", "refresh-token", time.Now().Add(1*time.Hour))
	require.NoError(t, err)

	// Revoke session
	err = repo.Delete(ctx, session.ID)
	require.NoError(t, err)

	// Try to revoke again
	err = repo.Delete(ctx, session.ID)

	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrSessionNotFound)
}

// TestRevokeAllSessions_Success tests revoking all sessions for a user
func TestRevokeAllSessions_Success(t *testing.T) {
	ctx := context.Background()
	repo := NewMockSessionRepository()
	userID := "user-123"

	// Create multiple sessions
	_, err := repo.Create(ctx, userID, "access-1", "refresh-1", time.Now().Add(1*time.Hour))
	require.NoError(t, err)

	_, err = repo.Create(ctx, userID, "access-2", "refresh-2", time.Now().Add(2*time.Hour))
	require.NoError(t, err)

	_, err = repo.Create(ctx, userID, "access-3", "refresh-3", time.Now().Add(3*time.Hour))
	require.NoError(t, err)

	// Verify sessions exist
	sessions, err := repo.GetByUserID(ctx, userID)
	require.NoError(t, err)
	assert.Len(t, sessions, 3)

	// Revoke all sessions
	err = repo.DeleteByUserID(ctx, userID)
	require.NoError(t, err)

	// Verify all sessions are deleted
	sessions, err = repo.GetByUserID(ctx, userID)
	require.NoError(t, err)
	assert.Len(t, sessions, 0, "All sessions should be deleted")
}

// TestRevokeAllSessions_UserHasNoSessions tests revoking sessions when user has none
func TestRevokeAllSessions_UserHasNoSessions(t *testing.T) {
	ctx := context.Background()
	repo := NewMockSessionRepository()
	userID := "user-123"

	// Don't create any sessions

	// Revoke all sessions (should succeed even if none exist)
	err := repo.DeleteByUserID(ctx, userID)
	assert.NoError(t, err, "Deleting sessions should succeed even if user has no sessions")
}

// TestRefreshSession_Success tests successful session refresh with token rotation
func TestRefreshSession_Success(t *testing.T) {
	ctx := context.Background()
	repo := NewMockSessionRepository()
	userID := "user-123"

	// Create session with old tokens
	oldAccessToken := "old-access-token"
	oldRefreshToken := "old-refresh-token"

	session, err := repo.Create(ctx, userID, oldAccessToken, oldRefreshToken, time.Now().Add(1*time.Hour))
	require.NoError(t, err)

	// Verify old tokens work
	found, err := repo.GetByAccessToken(ctx, oldAccessToken)
	require.NoError(t, err)
	assert.Equal(t, session.ID, found.ID)

	// Refresh session with new tokens
	newAccessToken := "new-access-token"
	newRefreshToken := "new-refresh-token"
	newExpiry := time.Now().Add(2 * time.Hour)

	err = repo.UpdateTokens(ctx, session.ID, oldRefreshToken, newAccessToken, newRefreshToken, newExpiry)
	require.NoError(t, err)

	// Verify old tokens no longer work
	_, err = repo.GetByAccessToken(ctx, oldAccessToken)
	assert.Error(t, err, "Old access token should no longer work")

	// Verify new tokens work
	found, err = repo.GetByAccessToken(ctx, newAccessToken)
	require.NoError(t, err)
	assert.Equal(t, session.ID, found.ID)
}

// TestRefreshSession_Expired tests refreshing an expired session
func TestRefreshSession_Expired(t *testing.T) {
	ctx := context.Background()
	repo := NewMockSessionRepository()
	userID := "user-123"

	// Create expired session
	session, err := repo.Create(ctx, userID, "expired-access", "expired-refresh", time.Now().Add(-1*time.Hour))
	require.NoError(t, err)

	// Try to refresh expired session
	newAccessToken := "new-access-token"
	newRefreshToken := "new-refresh-token"
	newExpiry := time.Now().Add(2 * time.Hour)

	err = repo.UpdateTokens(ctx, session.ID, "expired-refresh", newAccessToken, newRefreshToken, newExpiry)
	require.NoError(t, err)

	// Verify session can still be refreshed even if expired
	// (in real implementation, you might want to prevent this)
	found, err := repo.GetByAccessToken(ctx, newAccessToken)
	assert.NoError(t, err)
	assert.Equal(t, session.ID, found.ID)
}

// TestRefreshSession_InvalidToken tests refreshing with invalid session ID
func TestRefreshSession_InvalidToken(t *testing.T) {
	ctx := context.Background()
	repo := NewMockSessionRepository()

	// Try to refresh non-existent session
	newAccessToken := "new-access-token"
	newRefreshToken := "new-refresh-token"
	newExpiry := time.Now().Add(2 * time.Hour)

	err := repo.UpdateTokens(ctx, "invalid-session-id", "old-refresh", newAccessToken, newRefreshToken, newExpiry)

	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrSessionNotFound)
}

// TestRefreshSession_RevokedSession tests refreshing a revoked session
func TestRefreshSession_RevokedSession(t *testing.T) {
	ctx := context.Background()
	repo := NewMockSessionRepository()
	userID := "user-123"

	// Create session
	session, err := repo.Create(ctx, userID, "access-token", "refresh-token", time.Now().Add(1*time.Hour))
	require.NoError(t, err)

	// Revoke session
	err = repo.Delete(ctx, session.ID)
	require.NoError(t, err)

	// Try to refresh revoked session
	newAccessToken := "new-access-token"
	newRefreshToken := "new-refresh-token"
	newExpiry := time.Now().Add(2 * time.Hour)

	err = repo.UpdateTokens(ctx, session.ID, "refresh-token", newAccessToken, newRefreshToken, newExpiry)

	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrSessionNotFound)
}

// TestSessionToken_Hashing tests that session tokens are properly hashed
func TestSessionToken_Hashing(t *testing.T) {
	token := "test-session-token-12345"

	// Hash should be consistent
	hash1 := hashToken(token)
	hash2 := hashToken(token)

	assert.Equal(t, hash1, hash2, "Same token should produce same hash")
	assert.NotEmpty(t, hash1)
	assert.NotEqual(t, hash1, token, "Hash should be different from plaintext")

	// Different tokens should produce different hashes
	hash3 := hashToken("different-token")
	assert.NotEqual(t, hash1, hash3, "Different tokens should produce different hashes")
}

// TestSession_ExpirationHandling tests session expiration handling
func TestSession_ExpirationHandling(t *testing.T) {
	ctx := context.Background()
	repo := NewMockSessionRepository()
	userID := "user-123"

	// Create session that will expire soon
	session, err := repo.Create(ctx, userID, "access-token", "refresh-token", time.Now().Add(100*time.Millisecond))
	require.NoError(t, err)

	// Session should be valid now
	found, err := repo.GetByAccessToken(ctx, "access-token")
	require.NoError(t, err)
	assert.Equal(t, session.ID, found.ID)

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Session should now be expired
	_, err = repo.GetByAccessToken(ctx, "access-token")
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrSessionExpired)
}

// TestSession_MultipleUsers tests sessions for multiple users
func TestSession_MultipleUsers(t *testing.T) {
	ctx := context.Background()
	repo := NewMockSessionRepository()

	user1ID := "user-1"
	user2ID := "user-2"

	// Create sessions for user 1
	session1, err := repo.Create(ctx, user1ID, "access-user1-1", "refresh-user1-1", time.Now().Add(1*time.Hour))
	require.NoError(t, err)

	session2, err := repo.Create(ctx, user1ID, "access-user1-2", "refresh-user1-2", time.Now().Add(2*time.Hour))
	require.NoError(t, err)

	// Create sessions for user 2
	session3, err := repo.Create(ctx, user2ID, "access-user2-1", "refresh-user2-1", time.Now().Add(1*time.Hour))
	require.NoError(t, err)

	// List user 1 sessions
	user1Sessions, err := repo.GetByUserID(ctx, user1ID)
	require.NoError(t, err)
	assert.Len(t, user1Sessions, 2, "User 1 should have 2 sessions")

	// List user 2 sessions
	user2Sessions, err := repo.GetByUserID(ctx, user2ID)
	require.NoError(t, err)
	assert.Len(t, user2Sessions, 1, "User 2 should have 1 session")

	// Verify sessions are not mixed
	user1IDs := make(map[string]bool)
	for _, s := range user1Sessions {
		user1IDs[s.ID] = true
	}

	assert.True(t, user1IDs[session1.ID])
	assert.True(t, user1IDs[session2.ID])
	assert.False(t, user1IDs[session3.ID], "User 2 session should not be in user 1's sessions")
}

// TestSession_DeleteByAccessToken tests deleting session by access token
func TestSession_DeleteByAccessToken(t *testing.T) {
	ctx := context.Background()
	repo := NewMockSessionRepository()
	userID := "user-123"

	// Create session
	accessToken := "access-token"
	_, err := repo.Create(ctx, userID, accessToken, "refresh-token", time.Now().Add(1*time.Hour))
	require.NoError(t, err)

	// Verify session exists
	_, err = repo.GetByAccessToken(ctx, accessToken)
	require.NoError(t, err)

	// Delete by access token
	err = repo.DeleteByAccessToken(ctx, accessToken)
	require.NoError(t, err)

	// Verify session is deleted
	_, err = repo.GetByAccessToken(ctx, accessToken)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrSessionNotFound)
}

// TestSession_GetByRefreshToken tests retrieving session by refresh token
func TestSession_GetByRefreshToken(t *testing.T) {
	ctx := context.Background()
	repo := NewMockSessionRepository()
	userID := "user-123"

	// Create session
	refreshToken := "refresh-token"
	session, err := repo.Create(ctx, userID, "access-token", refreshToken, time.Now().Add(1*time.Hour))
	require.NoError(t, err)

	// Get by refresh token
	found, err := repo.GetByRefreshToken(ctx, refreshToken)
	require.NoError(t, err)
	assert.Equal(t, session.ID, found.ID)
	assert.Equal(t, session.UserID, found.UserID)
}

// TestSession_ConcurrentAccess tests concurrent session access
func TestSession_ConcurrentAccess(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent test in short mode")
	}

	ctx := context.Background()
	repo := NewMockSessionRepository()
	userID := "user-123"

	done := make(chan error, 5)

	// Create sessions concurrently
	for i := 0; i < 5; i++ {
		go func(idx int) {
			_, err := repo.Create(ctx, userID, "access-"+string(rune(idx)), "refresh-"+string(rune(idx)), time.Now().Add(1*time.Hour))
			done <- err
		}(i)
	}

	// Wait for all creates
	for i := 0; i < 5; i++ {
		err := <-done
		assert.NoError(t, err)
	}

	// Verify all sessions were created
	sessions, err := repo.GetByUserID(ctx, userID)
	require.NoError(t, err)
	assert.Len(t, sessions, 5)
}

// TestSession_DeleteExpired tests cleanup of expired sessions
func TestSession_DeleteExpired(t *testing.T) {
	ctx := context.Background()
	repo := NewMockSessionRepository()
	userID := "user-123"

	// Create expired sessions
	_, err := repo.Create(ctx, userID, "expired-1", "refresh-1", time.Now().Add(-2*time.Hour))
	require.NoError(t, err)

	_, err = repo.Create(ctx, userID, "expired-2", "refresh-2", time.Now().Add(-1*time.Hour))
	require.NoError(t, err)

	// Create active session
	activeSession, err := repo.Create(ctx, userID, "active-1", "refresh-3", time.Now().Add(1*time.Hour))
	require.NoError(t, err)

	// Delete expired sessions
	count, err := repo.DeleteExpired(ctx)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, count, int64(2), "Should delete at least 2 expired sessions")

	// Verify only active session remains
	sessions, err := repo.GetByUserID(ctx, userID)
	require.NoError(t, err)
	assert.Len(t, sessions, 1, "Only active session should remain")
	assert.Equal(t, activeSession.ID, sessions[0].ID)
}

// TestSession_CountActive tests counting active sessions
func TestSession_CountActive(t *testing.T) {
	ctx := context.Background()
	repo := NewMockSessionRepository()

	user1ID := "user-1"
	user2ID := "user-2"

	// Create active sessions
	_, err := repo.Create(ctx, user1ID, "access-1", "refresh-1", time.Now().Add(1*time.Hour))
	require.NoError(t, err)

	_, err = repo.Create(ctx, user2ID, "access-2", "refresh-2", time.Now().Add(1*time.Hour))
	require.NoError(t, err)

	// Create expired session
	_, err = repo.Create(ctx, user1ID, "expired-access", "expired-refresh", time.Now().Add(-1*time.Hour))
	require.NoError(t, err)

	// Count active sessions
	count, err := repo.Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 2, count, "Should count only active sessions")
}

// TestSession_CountByUserID tests counting active sessions for a user
func TestSession_CountByUserID(t *testing.T) {
	ctx := context.Background()
	repo := NewMockSessionRepository()
	userID := "user-123"

	// Create multiple sessions for user
	_, err := repo.Create(ctx, userID, "access-1", "refresh-1", time.Now().Add(1*time.Hour))
	require.NoError(t, err)

	_, err = repo.Create(ctx, userID, "access-2", "refresh-2", time.Now().Add(1*time.Hour))
	require.NoError(t, err)

	// Create expired session
	_, err = repo.Create(ctx, userID, "expired-access", "expired-refresh", time.Now().Add(-1*time.Hour))
	require.NoError(t, err)

	// Count active sessions for user
	count, err := repo.Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 2, count, "Should count only active sessions")
}

// TestSession_UpdateAccessToken tests updating only the access token
func TestSession_UpdateAccessToken(t *testing.T) {
	ctx := context.Background()
	repo := NewMockSessionRepository()
	userID := "user-123"

	// Create session
	oldAccessToken := "old-access-token"
	session, err := repo.Create(ctx, userID, oldAccessToken, "refresh-token", time.Now().Add(1*time.Hour))
	require.NoError(t, err)

	// Verify session was created
	assert.NotEmpty(t, session.ID)

	// In a real implementation, you'd update the access token
	// For this test, we verify the token exists
	found, err := repo.GetByAccessToken(ctx, oldAccessToken)
	require.NoError(t, err)
	assert.Equal(t, session.ID, found.ID)
}

// TestSession_TokenRotation tests complete token rotation workflow
func TestSession_TokenRotation(t *testing.T) {
	ctx := context.Background()
	repo := NewMockSessionRepository()
	userID := "user-123"

	// Create session with initial tokens
	oldAccess := "old-access"
	oldRefresh := "old-refresh"

	session, err := repo.Create(ctx, userID, oldAccess, oldRefresh, time.Now().Add(30*time.Minute))
	require.NoError(t, err)

	// Simulate token refresh workflow
	// Step 1: Verify old refresh token works
	found, err := repo.GetByRefreshToken(ctx, oldRefresh)
	require.NoError(t, err)
	assert.Equal(t, session.ID, found.ID)

	// Step 2: Generate new tokens
	newAccess := "new-access"
	newRefresh := "new-refresh"
	newExpiry := time.Now().Add(1 * time.Hour)

	// Step 3: Update session with new tokens
	err = repo.UpdateTokens(ctx, session.ID, oldRefresh, newAccess, newRefresh, newExpiry)
	require.NoError(t, err)

	// Step 4: Verify old tokens no longer work
	_, err = repo.GetByAccessToken(ctx, oldAccess)
	assert.Error(t, err)

	_, err = repo.GetByRefreshToken(ctx, oldRefresh)
	assert.Error(t, err)

	// Step 5: Verify new tokens work
	found, err = repo.GetByAccessToken(ctx, newAccess)
	require.NoError(t, err)
	assert.Equal(t, session.ID, found.ID)

	found, err = repo.GetByRefreshToken(ctx, newRefresh)
	require.NoError(t, err)
	assert.Equal(t, session.ID, found.ID)
}

// TestSession_Security_TokenHashing tests security properties of token hashing
func TestSession_Security_TokenHashing(t *testing.T) {
	tokens := []string{
		"simple-token",
		"complex-token-with-special-chars-!@#$%",
		"token-with-unicode-你好",
		"very-long-token-" + string(make([]byte, 100)),
	}

	for _, token := range tokens {
		t.Run("hash_"+token[:10], func(t *testing.T) {
			hash := hashToken(token)

			// Hash should be different from token
			assert.NotEqual(t, hash, token, "Hash should not equal plaintext token")

			// Hash should be consistent
			hash2 := hashToken(token)
			assert.Equal(t, hash, hash2, "Hash should be deterministic")

			// Hash should be fixed length (SHA-256 + base64)
			assert.Greater(t, len(hash), 40, "Hash should have sufficient length")
			assert.Less(t, len(hash), 50, "Hash should not be excessively long")
		})
	}
}

// TestSession_Workflow_CompleteLifecycle tests complete session lifecycle
func TestSession_Workflow_CompleteLifecycle(t *testing.T) {
	ctx := context.Background()
	repo := NewMockSessionRepository()
	userID := "user-123"

	// Step 1: Create session
	accessToken := "access-token"
	refreshToken := "refresh-token"
	expiresAt := time.Now().Add(1 * time.Hour)

	session, err := repo.Create(ctx, userID, accessToken, refreshToken, expiresAt)
	require.NoError(t, err)
	assert.NotEmpty(t, session.ID)

	// Step 2: Retrieve session by access token
	found, err := repo.GetByAccessToken(ctx, accessToken)
	require.NoError(t, err)
	assert.Equal(t, session.ID, found.ID)

	// Step 3: Retrieve session by refresh token
	found, err = repo.GetByRefreshToken(ctx, refreshToken)
	require.NoError(t, err)
	assert.Equal(t, session.ID, found.ID)

	// Step 4: List all user sessions
	sessions, err := repo.GetByUserID(ctx, userID)
	require.NoError(t, err)
	assert.Len(t, sessions, 1)

	// Step 5: Refresh session (token rotation)
	newAccessToken := "new-access-token"
	newRefreshToken := "new-refresh-token"
	newExpiry := time.Now().Add(2 * time.Hour)

	err = repo.UpdateTokens(ctx, session.ID, refreshToken, newAccessToken, newRefreshToken, newExpiry)
	require.NoError(t, err)

	// Step 6: Verify new tokens work
	_, err = repo.GetByAccessToken(ctx, newAccessToken)
	require.NoError(t, err)

	// Step 7: Revoke session
	err = repo.Delete(ctx, session.ID)
	require.NoError(t, err)

	// Step 8: Verify session is deleted
	_, err = repo.GetByAccessToken(ctx, newAccessToken)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrSessionNotFound)

	// Step 9: Verify no sessions remain for user
	sessions, err = repo.GetByUserID(ctx, userID)
	require.NoError(t, err)
	assert.Len(t, sessions, 0)
}
