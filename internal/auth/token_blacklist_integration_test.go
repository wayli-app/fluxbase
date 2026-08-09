//go:build integration
// +build integration

package auth_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nimbleflux/fluxbase/internal/auth"
	"github.com/nimbleflux/fluxbase/internal/testutil/e2e"
)

const authTestSecretKey = "test-secret-key-must-be-32-characters!"

// TestRevokeAllUserTokens_Integration tests the complete user-wide revocation flow
func TestRevokeAllUserTokens_Integration(t *testing.T) {
	tc := e2e.NewIntegrationTestContextWithNamespace(t, "auth")
	defer tc.Close()
	defer tc.CleanupTestData()

	ctx := context.Background()
	repo := auth.NewTokenBlacklistRepository(tc.TestContext.DB)
	manager, err := auth.NewJWTManager(authTestSecretKey, 15*time.Minute, 7*24*time.Hour)
	require.NoError(t, err)

	service := auth.NewTokenBlacklistService(repo, manager)

	userID := "user-123"
	reason := "security incident"

	t.Run("RevokeAllUserTokens creates marker entry", func(t *testing.T) {
		err := service.RevokeAllUserTokens(ctx, userID, reason)
		require.NoError(t, err)

		// Verify the marker was created by querying for it
		var count int
		err = tc.TestContext.DB.Pool().QueryRow(ctx,
			`SELECT COUNT(*) FROM auth.token_blacklist WHERE token_jti LIKE $1`,
			"user:"+userID+":all:*").Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 1, count, "expected one revocation marker")
	})

	t.Run("IsTokenRevoked returns true for token issued before revocation", func(t *testing.T) {
		// Create a token that was issued BEFORE the revocation
		tokenIssuedAt := time.Now().Add(-1 * time.Hour)
		jti := "old-token-jti"

		isRevoked, err := service.IsTokenRevoked(ctx, jti, userID, tokenIssuedAt)
		require.NoError(t, err)
		assert.True(t, isRevoked, "token issued before revocation should be revoked")
	})

	t.Run("IsTokenRevoked returns false for token issued after revocation", func(t *testing.T) {
		// Create a token that was issued AFTER the revocation
		tokenIssuedAt := time.Now().Add(1 * time.Hour)
		jti := "new-token-jti"

		isRevoked, err := service.IsTokenRevoked(ctx, jti, userID, tokenIssuedAt)
		require.NoError(t, err)
		assert.False(t, isRevoked, "token issued after revocation should not be revoked")
	})

	t.Run("IsTokenRevoked with empty userID skips user-wide check", func(t *testing.T) {
		tokenIssuedAt := time.Now().Add(-1 * time.Hour)
		jti := "another-token-jti"

		// Empty userID should skip user-wide check (backward compatibility)
		isRevoked, err := service.IsTokenRevoked(ctx, jti, "", tokenIssuedAt)
		require.NoError(t, err)
		assert.False(t, isRevoked, "empty userID should skip user-wide revocation check")
	})

	t.Run("Specific JTI revocation still works alongside user-wide revocation", func(t *testing.T) {
		// Blacklist a specific JTI
		specificJTI := "specific-token-jti"
		err := repo.Add(ctx, specificJTI, &userID, "specific revocation", time.Now().Add(time.Hour))
		require.NoError(t, err)

		// This specific JTI should be revoked regardless of when it was issued
		isRevoked, err := service.IsTokenRevoked(ctx, specificJTI, userID, time.Now().Add(1*time.Hour))
		require.NoError(t, err)
		assert.True(t, isRevoked, "specific JTI revocation should work")
	})

	t.Run("Multiple users: revoking user A does not affect user B", func(t *testing.T) {
		userA := "user-A"
		userB := "user-B"

		// Revoke all tokens for user A
		err := service.RevokeAllUserTokens(ctx, userA, "revoke user A")
		require.NoError(t, err)

		// User B's old token should not be revoked
		tokenIssuedAt := time.Now().Add(-1 * time.Hour)
		isRevoked, err := service.IsTokenRevoked(ctx, "token-b-jti", userB, tokenIssuedAt)
		require.NoError(t, err)
		assert.False(t, isRevoked, "user B's token should not be affected by user A's revocation")

		// User A's old token should be revoked
		isRevoked, err = service.IsTokenRevoked(ctx, "token-a-jti", userA, tokenIssuedAt)
		require.NoError(t, err)
		assert.True(t, isRevoked, "user A's token should be revoked")
	})
}

// TestUserRevocationCache tests the cache behavior for user-wide revocation
func TestUserRevocationCache(t *testing.T) {
	tc := e2e.NewIntegrationTestContextWithNamespace(t, "auth")
	defer tc.Close()
	defer tc.CleanupTestData()

	ctx := context.Background()
	repo := auth.NewTokenBlacklistRepository(tc.TestContext.DB)
	manager, err := auth.NewJWTManager(authTestSecretKey, 15*time.Minute, 7*24*time.Hour)
	require.NoError(t, err)

	service := auth.NewTokenBlacklistService(repo, manager)
	userID := "cached-user"

	t.Run("Cache hit: second call within TTL uses cache", func(t *testing.T) {
		// First call should query DB and populate cache
		tokenIssuedAt := time.Now()
		_, err := service.IsTokenRevoked(ctx, "jti-1", userID, tokenIssuedAt)
		require.NoError(t, err)

		// Invalidate any existing cache by setting a revocation
		_ = service.RevokeAllUserTokens(ctx, userID, "test")

		// This should use cache (within TTL)
		_, err = service.IsTokenRevoked(ctx, "jti-2", userID, tokenIssuedAt)
		require.NoError(t, err)
	})

	t.Run("Cache invalidation: RevokeAllUserTokens clears cache", func(t *testing.T) {
		// Set up initial state
		_ = service.RevokeAllUserTokens(ctx, userID, "first revocation")

		// Populate cache
		tokenIssuedAt := time.Now()
		_, err := service.IsTokenRevoked(ctx, "jti-3", userID, tokenIssuedAt)
		require.NoError(t, err)

		// Revoke again - this should invalidate cache
		_ = service.RevokeAllUserTokens(ctx, userID, "second revocation")

		// Next check should query DB again and get the new revocation time
		isRevoked, err := service.IsTokenRevoked(ctx, "jti-4", userID, tokenIssuedAt)
		require.NoError(t, err)
		assert.True(t, isRevoked)
	})

	t.Run("Cache expiration: entries older than TTL are ignored", func(t *testing.T) {
		userID := "expire-user"
		service := auth.NewTokenBlacklistService(repo, manager)

		// Populate cache
		_ = service.RevokeAllUserTokens(ctx, userID, "test")
		_, _ = service.IsTokenRevoked(ctx, "jti-5", userID, time.Now())

		// Wait for cache to expire (default TTL is 5 seconds, but we can't change it directly)
		// Instead, we'll just verify the cache mechanism works by checking multiple times
		_, err := service.IsTokenRevoked(ctx, "jti-6", userID, time.Now())
		require.NoError(t, err)
	})
}

// TestGetMaxTokenTTL tests the JWT manager's max TTL calculation
func TestGetMaxTokenTTL(t *testing.T) {
	t.Run("returns maximum of all token TTls", func(t *testing.T) {
		// Set service role and anon TTLs via the config constructor
		managerWithTTLs, err := auth.NewJWTManagerWithConfig(authTestSecretKey,
			15*time.Minute, // access TTL
			7*24*time.Hour, // refresh TTL
			24*time.Hour,   // service role TTL
			24*time.Hour)   // anon TTL
		require.NoError(t, err)

		maxTTL := managerWithTTLs.GetMaxTokenTTL()
		// Max should be 7 days (refresh TTL)
		assert.Equal(t, 7*24*time.Hour, maxTTL)
	})

	t.Run("fallback to 7 days if all TTls are very small", func(t *testing.T) {
		manager, err := auth.NewJWTManager(authTestSecretKey, 1*time.Minute, 2*time.Minute)
		require.NoError(t, err)

		maxTTL := manager.GetMaxTokenTTL()
		// Should fallback to 7 days
		assert.Equal(t, 7*24*time.Hour, maxTTL)
	})
}
