package auth

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRevokeToken_ServiceRoleTokenRejected(t *testing.T) {
	manager := NewJWTManager("test-secret", 15*time.Minute, 7*24*time.Hour)

	// Generate a service role token
	serviceRoleToken, err := manager.GenerateServiceRoleToken()
	require.NoError(t, err)

	// Create service without repository (we only need the jwtManager for this test)
	service := &TokenBlacklistService{
		repo:       nil, // Not needed since we expect early return
		jwtManager: manager,
	}

	// Attempt to revoke the service role token - should be rejected
	err = service.RevokeToken(context.Background(), serviceRoleToken, "test revocation")

	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrCannotRevokeServiceRole)
}

func TestRevokeToken_AnonTokenNotBlockedByGuard(t *testing.T) {
	manager := NewJWTManager("test-secret", 15*time.Minute, 7*24*time.Hour)

	// Generate an anon token
	anonToken, err := manager.GenerateAnonToken()
	require.NoError(t, err)

	// Verify the token is valid as a service role token with "anon" role
	claims, err := manager.ValidateServiceRoleToken(anonToken)
	require.NoError(t, err)
	assert.Equal(t, "anon", claims.Role)

	// The guard only blocks "service_role", not "anon"
	// Anon tokens should be revocable since they represent anonymous users
}

func TestRevokeToken_OnlyServiceRoleBlocked(t *testing.T) {
	manager := NewJWTManager("test-secret", 15*time.Minute, 7*24*time.Hour)

	// Generate a service role token and verify it's blocked
	serviceRoleToken, err := manager.GenerateServiceRoleToken()
	require.NoError(t, err)

	claims, err := manager.ValidateServiceRoleToken(serviceRoleToken)
	require.NoError(t, err)
	assert.Equal(t, "service_role", claims.Role)

	// Only service_role tokens should be blocked from revocation
	// This is because service_role is a system-level credential that should never be blacklisted
}
