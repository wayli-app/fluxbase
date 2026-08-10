//go:build integration
// +build integration

package api_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nimbleflux/fluxbase/internal/testutil/e2e"
)

// TestAdminSession_ListSessions_Integration tests listing all admin sessions
func TestAdminSession_ListSessions_Integration(t *testing.T) {
	tc := e2e.NewIntegrationTestContextWithNamespace(t, "api")
	defer tc.Close()
	defer tc.CleanupTestData()

	// Create dashboard admin user
	adminEmail := randomEmail()
	_, adminToken := tc.CreateDashboardAdminUser(adminEmail, "password123")

	// Create a regular user and get their token (creates a session)
	userEmail := randomEmail()
	_, _ = tc.CreateTestUser(userEmail, "password123")

	// List admin sessions
	resp := tc.NewRequest("GET", "/api/v1/admin/auth/sessions").
		WithAuth(adminToken).
		Send().
		AssertStatus(200)

	var result map[string]interface{}
	resp.JSON(&result)

	// Verify response structure
	assert.Contains(t, result, "sessions")
	assert.Contains(t, result, "total")
	assert.Contains(t, result, "limit")
	assert.Contains(t, result, "offset")

	sessions := result["sessions"].([]interface{})
	total := int(result["total"].(float64))

	// Should have at least 1 session (admin)
	// Note: User signup may not create an auth session in the same way as dashboard login
	assert.GreaterOrEqual(t, total, 1, "Should have at least admin session")
	assert.GreaterOrEqual(t, len(sessions), 1, "Sessions array should have at least 1 entry")
}

// TestAdminSession_ListSessions_Pagination_Integration tests pagination for session listing
func TestAdminSession_ListSessions_Pagination_Integration(t *testing.T) {
	tc := e2e.NewIntegrationTestContextWithNamespace(t, "api")
	defer tc.Close()
	defer tc.CleanupTestData()

	// Create dashboard admin user
	adminEmail := randomEmail()
	_, adminToken := tc.CreateDashboardAdminUser(adminEmail, "password123")

	// Create multiple users (each creates a session via signup)
	for i := 0; i < 5; i++ {
		tc.CreateTestUser(randomEmail(), "password123")
	}

	// List sessions with limit=2
	resp := tc.NewRequest("GET", "/api/v1/admin/auth/sessions?limit=2&offset=0").
		WithAuth(adminToken).
		Send().
		AssertStatus(200)

	var result map[string]interface{}
	resp.JSON(&result)

	// Verify pagination parameters
	assert.Equal(t, float64(2), result["limit"])
	assert.Equal(t, float64(0), result["offset"])

	sessions, _ := result["sessions"].([]interface{})
	// Guard the type assertion: if "count" is missing/nil the response shape
	// differs from expected, but that should be an assertion failure, not a panic
	// that aborts the entire test package (killing coverage for all other tests).
	count := 0
	if countVal, ok := result["count"].(float64); ok {
		count = int(countVal)
	}

	// Should return at most 2 sessions with limit=2
	assert.LessOrEqual(t, count, 2, "Should return at most 2 sessions with limit=2")
	assert.Equal(t, count, len(sessions), "Count should match sessions array length")
	assert.NotNil(t, result["count"], "response should include count field")

	// Get second page
	resp2 := tc.NewRequest("GET", "/api/v1/admin/auth/sessions?limit=2&offset=2").
		WithAuth(adminToken).
		Send().
		AssertStatus(200)

	var result2 map[string]interface{}
	resp2.JSON(&result2)

	sessions2 := result2["sessions"].([]interface{})
	assert.LessOrEqual(t, len(sessions2), 2, "Second page should have at most 2 sessions")
}

// TestAdminSession_RevokeSession_Integration tests revoking a specific session
func TestAdminSession_RevokeSession_Integration(t *testing.T) {
	tc := e2e.NewIntegrationTestContextWithNamespace(t, "api")
	defer tc.Close()
	defer tc.CleanupTestData()

	// Create dashboard admin user
	adminEmail := randomEmail()
	_, adminToken := tc.CreateDashboardAdminUser(adminEmail, "password123")

	// Create a regular user (creates a session)
	userEmail := randomEmail()
	_, userToken := tc.CreateTestUser(userEmail, "password123")

	// Get session ID from token (extract from JWT)
	// For simplicity, we'll query the database to get the session ID
	sessions := tc.QuerySQL(`
		SELECT id, user_id
		FROM auth.sessions
		WHERE user_id = (SELECT id FROM auth.users WHERE email = $1)
		ORDER BY created_at DESC
		LIMIT 1
	`, userEmail)

	require.NotEmpty(t, sessions, "Should have at least one session for the user")
	sessionID := sessions[0]["id"].(string)

	// Revoke the session
	resp := tc.NewRequest("DELETE", "/api/v1/admin/auth/sessions/"+sessionID).
		WithAuth(adminToken).
		Send().
		AssertStatus(200)

	var result map[string]interface{}
	resp.JSON(&result)
	assert.Equal(t, "Session revoked successfully", result["message"])

	// Verify session is deleted
	sessionsAfter := tc.QuerySQL(`
		SELECT id FROM auth.sessions WHERE id = $1
	`, sessionID)
	assert.Empty(t, sessionsAfter, "Session should be deleted")

	// Verify the user token is no longer valid
	resp2 := tc.NewRequest("GET", "/api/v1/auth/user").
		WithAuth(userToken).
		Send()
	// The token should be invalid (401 Unauthorized) since session was revoked
	assert.Contains(t, []int{401, 403}, resp2.Status(), "User token should be invalid after session revocation")
}

// TestAdminSession_RevokeSession_NotFound_Integration tests revoking a non-existent session
func TestAdminSession_RevokeSession_NotFound_Integration(t *testing.T) {
	tc := e2e.NewIntegrationTestContextWithNamespace(t, "api")
	defer tc.Close()
	defer tc.CleanupTestData()

	// Create dashboard admin user
	adminEmail := randomEmail()
	_, adminToken := tc.CreateDashboardAdminUser(adminEmail, "password123")

	// Try to revoke a non-existent session
	fakeSessionID := uuid.New().String()
	resp := tc.NewRequest("DELETE", "/api/v1/admin/auth/sessions/"+fakeSessionID).
		WithAuth(adminToken).
		Send().
		AssertStatus(404)

	var result map[string]interface{}
	resp.JSON(&result)
	assert.Equal(t, "Session not found", result["error"])
}

// TestAdminSession_RevokeUserSessions_Integration tests revoking all sessions for a user
func TestAdminSession_RevokeUserSessions_Integration(t *testing.T) {
	tc := e2e.NewIntegrationTestContextWithNamespace(t, "api")
	defer tc.Close()
	defer tc.CleanupTestData()

	// Create dashboard admin user
	adminEmail := randomEmail()
	_, adminToken := tc.CreateDashboardAdminUser(adminEmail, "password123")

	// Create a regular user (creates one session via signup)
	userEmail := randomEmail()
	userID, userToken := tc.CreateTestUser(userEmail, "password123")

	// Create additional sessions for the user by signing in multiple times
	for i := 0; i < 3; i++ {
		tc.NewRequest("POST", "/api/v1/auth/signin").
			WithBody(map[string]interface{}{
				"email":    userEmail,
				"password": "password123",
			}).
			Send()
	}

	// Verify user has multiple sessions
	sessionsBefore := tc.QuerySQL(`
		SELECT COUNT(*) as count FROM auth.sessions WHERE user_id = $1
	`, userID)
	assert.GreaterOrEqual(t, sessionsBefore[0]["count"], int64(4), "User should have at least 4 sessions")

	// Revoke all user sessions
	resp := tc.NewRequest("DELETE", "/api/v1/admin/auth/sessions/user/"+userID).
		WithAuth(adminToken).
		Send().
		AssertStatus(200)

	var result map[string]interface{}
	resp.JSON(&result)
	assert.Equal(t, "All user sessions revoked successfully", result["message"])

	// Verify all sessions are deleted
	sessionsAfter := tc.QuerySQL(`
		SELECT COUNT(*) as count FROM auth.sessions WHERE user_id = $1
	`, userID)
	assert.Equal(t, int64(0), sessionsAfter[0]["count"], "All user sessions should be deleted")

	// Verify the user token is no longer valid
	resp2 := tc.NewRequest("GET", "/api/v1/auth/user").
		WithAuth(userToken).
		Send()
	// The token should be invalid (401 Unauthorized) since all sessions were revoked
	assert.Contains(t, []int{401, 403}, resp2.Status(), "User token should be invalid after revoking all sessions")
}

// TestAdminSession_Unauthorized_Integration tests that regular users cannot access admin session endpoints
func TestAdminSession_Unauthorized_Integration(t *testing.T) {
	tc := e2e.NewIntegrationTestContextWithNamespace(t, "api")
	defer tc.Close()
	defer tc.CleanupTestData()

	// Create a regular user (not admin)
	userEmail := randomEmail()
	_, userToken := tc.CreateTestUser(userEmail, "password123")

	// Try to list sessions (should fail - user is not admin)
	resp := tc.NewRequest("GET", "/api/v1/admin/auth/sessions").
		WithAuth(userToken).
		Send()
	assert.Contains(t, []int{401, 403}, resp.Status(), "Regular user should not be able to list admin sessions")

	// Try to revoke a session (should fail)
	resp2 := tc.NewRequest("DELETE", "/api/v1/admin/auth/sessions/"+uuid.New().String()).
		WithAuth(userToken).
		Send()
	assert.Contains(t, []int{401, 403}, resp2.Status(), "Regular user should not be able to revoke admin sessions")

	// Try to revoke user sessions (should fail)
	resp3 := tc.NewRequest("DELETE", "/api/v1/admin/auth/sessions/user/"+uuid.New().String()).
		WithAuth(userToken).
		Send()
	assert.Contains(t, []int{401, 403}, resp3.Status(), "Regular user should not be able to revoke user sessions")
}
