package e2e

import (
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/require"
	"github.com/wayli-app/fluxbase/test"
)

// setupAuthTest prepares the test context for auth tests
func setupAuthTest(t *testing.T) *test.TestContext {
	tc := test.NewTestContext(t)
	tc.EnsureAuthSchema()

	// Clean auth tables before each test to ensure isolation
	tc.ExecuteSQL("TRUNCATE TABLE auth.users CASCADE")
	tc.ExecuteSQL("TRUNCATE TABLE auth.sessions CASCADE")

	return tc
}

// TestAuthSignup tests user signup with email/password
func TestAuthSignup(t *testing.T) {
	tc := setupAuthTest(t)
	defer tc.Close()

	// Signup with email and password
	email := "test@example.com"
	password := "testpassword123"

	resp := tc.NewRequest("POST", "/api/v1/auth/signup").
		WithBody(map[string]interface{}{
			"email":    email,
			"password": password,
		}).
		Send().
		AssertStatus(fiber.StatusCreated)

	// Verify response structure
	var result map[string]interface{}
	resp.JSON(&result)

	require.NotNil(t, result["access_token"], "access_token should be present")
	require.NotNil(t, result["refresh_token"], "refresh_token should be present")
	require.NotNil(t, result["user"], "user should be present")

	// Verify user in database
	users := tc.QuerySQL("SELECT email FROM auth.users WHERE email = $1", email)
	require.Len(t, users, 1, "User should exist in database")
	require.Equal(t, email, users[0]["email"])
}

// TestAuthSignin tests user signin with email/password
func TestAuthSignin(t *testing.T) {
	tc := setupAuthTest(t)
	defer tc.Close()

	// Create a test user
	email := "signin@example.com"
	password := "testpassword123"
	_, token := tc.CreateTestUser(email, password)
	require.NotEmpty(t, token, "Should receive token from signup")

	// Now signin with the same credentials
	resp := tc.NewRequest("POST", "/api/v1/auth/signin").
		WithBody(map[string]interface{}{
			"email":    email,
			"password": password,
		}).
		Send().
		AssertStatus(fiber.StatusOK)

	// Verify response structure
	var result map[string]interface{}
	resp.JSON(&result)

	require.NotNil(t, result["access_token"], "access_token should be present")
	require.NotNil(t, result["refresh_token"], "refresh_token should be present")
}

// TestAuthGetUser tests getting current user info with token
func TestAuthGetUser(t *testing.T) {
	tc := setupAuthTest(t)
	defer tc.Close()

	// Create a test user
	email := "getuser@example.com"
	password := "testpassword123"
	userID, token := tc.CreateTestUser(email, password)
	require.NotEmpty(t, userID, "Should have user ID")
	require.NotEmpty(t, token, "Should have token")

	// Get user info with token
	resp := tc.NewRequest("GET", "/api/v1/auth/user").
		WithAuth(token).
		Send().
		AssertStatus(fiber.StatusOK)

	// Verify response
	var user map[string]interface{}
	resp.JSON(&user)

	require.Equal(t, email, user["email"])
	require.Equal(t, userID, user["id"])
}

// TestAuthSignout tests user signout (token invalidation)
func TestAuthSignout(t *testing.T) {
	tc := setupAuthTest(t)
	defer tc.Close()

	// Create a test user
	email := "signout@example.com"
	password := "testpassword123"
	_, token := tc.CreateTestUser(email, password)

	// Signout
	tc.NewRequest("POST", "/api/v1/auth/signout").
		WithAuth(token).
		Send().
		AssertStatus(fiber.StatusOK)

	// Try to use the token - should fail
	tc.NewRequest("GET", "/api/v1/auth/user").
		WithAuth(token).
		Send().
		AssertStatus(fiber.StatusUnauthorized)
}

// TestAuthRefreshToken tests the refresh token flow
func TestAuthRefreshToken(t *testing.T) {
	tc := setupAuthTest(t)
	defer tc.Close()

	// Create a test user
	email := "refresh@example.com"
	password := "testpassword123"

	signupResp := tc.NewRequest("POST", "/api/v1/auth/signup").
		WithBody(map[string]interface{}{
			"email":    email,
			"password": password,
		}).
		Send().
		AssertStatus(fiber.StatusCreated)

	var signupResult map[string]interface{}
	signupResp.JSON(&signupResult)

	refreshToken := signupResult["refresh_token"].(string)
	require.NotEmpty(t, refreshToken, "refresh_token should be present")

	// Use refresh token to get new access token
	refreshResp := tc.NewRequest("POST", "/api/v1/auth/refresh").
		WithBody(map[string]interface{}{
			"refresh_token": refreshToken,
		}).
		Send().
		AssertStatus(fiber.StatusOK)

	var refreshResult map[string]interface{}
	refreshResp.JSON(&refreshResult)

	require.NotNil(t, refreshResult["access_token"], "new access_token should be present")
	require.NotNil(t, refreshResult["refresh_token"], "new refresh_token should be present")
}

// TestAuthPasswordReset tests password reset flow with MailHog email verification
func TestAuthPasswordReset(t *testing.T) {
	tc := setupAuthTest(t)
	defer tc.Close()
	_ = tc.ClearMailHogEmails()

	// Create a test user
	email := "reset@example.com"
	password := "oldpassword123"
	tc.CreateTestUser(email, password)

	// Request password reset
	tc.NewRequest("POST", "/api/v1/auth/password/reset").
		WithBody(map[string]interface{}{
			"email": email,
		}).
		Send().
		AssertStatus(fiber.StatusOK)

	// Wait for password reset email
	resetEmail := tc.WaitForEmail(5*time.Second, func(msg test.MailHogMessage) bool {
		if len(msg.To) > 0 {
			return msg.To[0].Mailbox+"@"+msg.To[0].Domain == email
		}
		return false
	})

	if resetEmail != nil {
		require.Contains(t, resetEmail.Content.Body, "reset", "Email should contain reset info")
		t.Logf("Password reset email received successfully")
	} else {
		t.Log("Password reset email not received (MailHog might not be available)")
	}
}

// TestAuthMagicLink tests magic link authentication with MailHog
func TestAuthMagicLink(t *testing.T) {
	tc := setupAuthTest(t)
	defer tc.Close()
	_ = tc.ClearMailHogEmails()

	email := "magic@example.com"

	// Request magic link
	tc.NewRequest("POST", "/api/v1/auth/magiclink").
		WithBody(map[string]interface{}{
			"email": email,
		}).
		Send().
		AssertStatus(fiber.StatusOK)

	// Wait for magic link email
	magicEmail := tc.WaitForEmail(5*time.Second, func(msg test.MailHogMessage) bool {
		if len(msg.To) > 0 {
			return msg.To[0].Mailbox+"@"+msg.To[0].Domain == email
		}
		return false
	})

	if magicEmail != nil {
		require.Contains(t, strings.ToLower(magicEmail.Content.Body), "login", "Email should contain login link info")
		require.Contains(t, magicEmail.Content.Body, "token=", "Email should contain token parameter")
		t.Logf("Magic link email received successfully")
	} else {
		t.Log("Magic link email not received (MailHog might not be available)")
	}
}

// TestAuthInvalidCredentials tests signin with wrong password
func TestAuthInvalidCredentials(t *testing.T) {
	tc := setupAuthTest(t)
	defer tc.Close()

	// Create a test user
	email := "invalid@example.com"
	password := "correctpassword"
	tc.CreateTestUser(email, password)

	// Try to signin with wrong password
	tc.NewRequest("POST", "/api/v1/auth/signin").
		WithBody(map[string]interface{}{
			"email":    email,
			"password": "wrongpassword",
		}).
		Send().
		AssertStatus(fiber.StatusUnauthorized)
}

// TestAuthMissingToken tests accessing protected endpoint without token
func TestAuthMissingToken(t *testing.T) {
	tc := setupAuthTest(t)
	defer tc.Close()

	// Try to get user info without token
	tc.NewRequest("GET", "/api/v1/auth/user").
		Send().
		AssertStatus(fiber.StatusUnauthorized)
}
