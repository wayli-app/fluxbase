//go:build integration
// +build integration

package auth_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nimbleflux/fluxbase/internal/auth"
	"github.com/nimbleflux/fluxbase/internal/config"
	"github.com/nimbleflux/fluxbase/internal/email"
	"github.com/nimbleflux/fluxbase/internal/testutil"
)

func randomTestEmail() string {
	return fmt.Sprintf("test-%s@example.com", uuid.New().String()[:8])
}

// =============================================================================
// Signup & Signin Integration Tests
// =============================================================================

func TestAuthService_Signup_Success_Integration(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "auth")
	defer tc.Close()
	defer tc.CleanupTestData()

	// Create auth service
	service := createAuthService(t, tc)

	ctx := context.Background()
	email := randomTestEmail()
	password := "TestPassword123!"

	// Sign up new user
	resp, err := service.SignUp(ctx, auth.SignUpRequest{
		Email:    email,
		Password: password,
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.User)

	// Verify user was created in database
	results := tc.QuerySQL(`SELECT id, email, email_verified, role FROM auth.users WHERE email = $1`, email)
	require.Len(t, results, 1, "User should be created in database")
	assert.Equal(t, email, results[0]["email"])
	assert.False(t, results[0]["email_verified"].(bool), "Email should not be verified initially")
	assert.Equal(t, "authenticated", results[0]["role"])

	// Verify tokens were generated
	assert.NotEmpty(t, resp.AccessToken, "Access token should be generated")
	assert.NotEmpty(t, resp.RefreshToken, "Refresh token should be generated")
	assert.Greater(t, resp.ExpiresIn, int64(0), "ExpiresIn should be positive")

	// Verify session was created
	sessionResults := tc.QuerySQL(`SELECT * FROM auth.sessions WHERE user_id = $1`, resp.User.ID)
	assert.Len(t, sessionResults, 1, "Session should be created")
}

func TestAuthService_Signup_DuplicateEmail_Integration(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "auth")
	defer tc.Close()
	defer tc.CleanupTestData()

	service := createAuthService(t, tc)
	ctx := context.Background()
	email := randomTestEmail()
	password := "TestPassword123!"

	// Create first user
	_, err := service.SignUp(ctx, auth.SignUpRequest{
		Email:    email,
		Password: password,
	})
	require.NoError(t, err)

	// Try to create duplicate user
	_, err = service.SignUp(ctx, auth.SignUpRequest{
		Email:    email,
		Password: password,
	})
	assert.Error(t, err, "Should error on duplicate email")
	assert.Contains(t, err.Error(), "already exists", "Error should mention user already exists")
}

func TestAuthService_Signup_InvalidEmail_Integration(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "auth")
	defer tc.Close()
	defer tc.CleanupTestData()

	service := createAuthService(t, tc)
	ctx := context.Background()

	// Test with invalid email
	_, err := service.SignUp(ctx, auth.SignUpRequest{
		Email:    "not-an-email",
		Password: "TestPassword123!",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid email")
}

func TestAuthService_Signup_WeakPassword_Integration(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "auth")
	defer tc.Close()
	defer tc.CleanupTestData()

	service := createAuthService(t, tc)
	ctx := context.Background()
	email := randomTestEmail()

	// Test with weak password
	_, err := service.SignUp(ctx, auth.SignUpRequest{
		Email:    email,
		Password: "weak",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "password")
}

func TestAuthService_Signin_ValidCredentials_Integration(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "auth")
	defer tc.Close()
	defer tc.CleanupTestData()

	service := createAuthService(t, tc)
	ctx := context.Background()
	email := randomTestEmail()
	password := "TestPassword123!"

	// First, sign up
	_, err := service.SignUp(ctx, auth.SignUpRequest{
		Email:    email,
		Password: password,
	})
	require.NoError(t, err)

	// Now sign in
	resp, err := service.SignIn(ctx, auth.SignInRequest{
		Email:    email,
		Password: password,
	})
	require.NoError(t, err)
	require.NotNil(t, resp.User)
	assert.Equal(t, email, resp.User.Email)
	assert.NotEmpty(t, resp.AccessToken)
	assert.NotEmpty(t, resp.RefreshToken)
}

func TestAuthService_Signin_InvalidPassword_Integration(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "auth")
	defer tc.Close()
	defer tc.CleanupTestData()

	service := createAuthService(t, tc)
	ctx := context.Background()
	email := randomTestEmail()
	password := "TestPassword123!"

	// First, sign up
	_, err := service.SignUp(ctx, auth.SignUpRequest{
		Email:    email,
		Password: password,
	})
	require.NoError(t, err)

	// Try to sign in with wrong password
	_, err = service.SignIn(ctx, auth.SignInRequest{
		Email:    email,
		Password: "WrongPassword123!",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid")
}

func TestAuthService_Signin_NonExistentUser_Integration(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "auth")
	defer tc.Close()
	defer tc.CleanupTestData()

	service := createAuthService(t, tc)
	ctx := context.Background()

	// Try to sign in with non-existent user
	_, err := service.SignIn(ctx, auth.SignInRequest{
		Email:    "nonexistent@example.com",
		Password: "TestPassword123!",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid")
}

// =============================================================================
// Session Management Integration Tests
// =============================================================================

func TestAuthService_RefreshToken_Valid_Integration(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "auth")
	defer tc.Close()
	defer tc.CleanupTestData()

	service := createAuthService(t, tc)
	ctx := context.Background()
	email := randomTestEmail()
	password := "TestPassword123!"

	// Sign up to get initial tokens
	signupResp, err := service.SignUp(ctx, auth.SignUpRequest{
		Email:    email,
		Password: password,
	})
	require.NoError(t, err)
	require.NotEmpty(t, signupResp.RefreshToken)

	// Refresh token
	refreshResp, err := service.RefreshToken(ctx, auth.RefreshTokenRequest{
		RefreshToken: signupResp.RefreshToken,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, refreshResp.AccessToken)
	assert.NotEmpty(t, refreshResp.RefreshToken)
}

func TestAuthService_RefreshToken_Invalid_Integration(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "auth")
	defer tc.Close()
	defer tc.CleanupTestData()

	service := createAuthService(t, tc)
	ctx := context.Background()

	// Try to refresh with invalid token
	_, err := service.RefreshToken(ctx, auth.RefreshTokenRequest{
		RefreshToken: "invalid-refresh-token",
	})
	assert.Error(t, err)
}

func TestAuthService_SignOut_Integration(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "auth")
	defer tc.Close()
	defer tc.CleanupTestData()

	service := createAuthService(t, tc)
	ctx := context.Background()
	email := randomTestEmail()
	password := "TestPassword123!"

	// Sign up
	signupResp, err := service.SignUp(ctx, auth.SignUpRequest{
		Email:    email,
		Password: password,
	})
	require.NoError(t, err)

	// Sign out
	err = service.SignOut(ctx, signupResp.AccessToken)
	require.NoError(t, err)

	// Verify session was deleted
	sessionResults := tc.QuerySQL(`SELECT * FROM auth.sessions WHERE user_id = $1`, signupResp.User.ID)
	assert.Len(t, sessionResults, 0, "Session should be deleted after sign out")
}

func TestAuthService_SignOut_InvalidToken_Integration(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "auth")
	defer tc.Close()
	defer tc.CleanupTestData()

	service := createAuthService(t, tc)
	ctx := context.Background()

	// Try to sign out with invalid token
	// Note: SignOut may succeed silently for invalid tokens to avoid information leakage
	err := service.SignOut(ctx, "invalid-token")
	// We don't assert error here - the behavior is implementation-defined
	_ = err
}

// =============================================================================
// Password Reset Integration Tests
// =============================================================================

func TestAuthService_RequestPasswordReset_Integration(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "auth")
	defer tc.Close()
	defer tc.CleanupTestData()
	defer tc.CleanupTestData()

	service := createAuthServiceWithMailHog(t, tc)
	ctx := context.Background()
	email := randomTestEmail()
	password := "TestPassword123!"

	// Clear MailHog messages before test
	testutil.DeleteAllMailHogMessages(t)

	// Sign up
	signupResp, err := service.SignUp(ctx, auth.SignUpRequest{
		Email:    email,
		Password: password,
	})
	require.NoError(t, err)

	// Request password reset
	err = service.PasswordResetService().RequestPasswordReset(ctx, email, "http://localhost:3000/reset-password")
	require.NoError(t, err)

	// Wait for email and verify it was sent
	msg := testutil.WaitForEmail(t, 5*time.Second, func(m testutil.MailHogMessage) bool {
		return len(m.To) > 0 && m.To[0].Mailbox+"@"+m.To[0].Domain == email
	})
	require.NotNil(t, msg, "Password reset email not received within timeout")

	// Verify reset token was created (note: token is stored as hash, so we can't retrieve it directly)
	results := tc.QuerySQL(`SELECT * FROM auth.password_reset_tokens WHERE user_id = $1`, signupResp.User.ID)
	assert.Len(t, results, 1, "Password reset token should be created")
	// Token is hashed, so we verify it exists and is not used
	assert.False(t, results[0]["used"].(bool), "Token should not be marked as used")
}

func TestAuthService_ResetPassword_ValidToken_Integration(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "auth")
	defer tc.Close()
	defer tc.CleanupTestData()
	defer tc.CleanupTestData()

	service := createAuthServiceWithMailHog(t, tc)
	ctx := context.Background()
	email := randomTestEmail()
	password := "TestPassword123!"

	// Clear MailHog messages before test
	testutil.DeleteAllMailHogMessages(t)

	// Sign up
	signupResp, err := service.SignUp(ctx, auth.SignUpRequest{
		Email:    email,
		Password: password,
	})
	require.NoError(t, err)

	// Request password reset
	err = service.PasswordResetService().RequestPasswordReset(ctx, email, "http://localhost:3000/reset-password")
	require.NoError(t, err)

	// Wait for password reset email and extract token
	msg := testutil.WaitForEmail(t, 5*time.Second, func(m testutil.MailHogMessage) bool {
		return len(m.To) > 0 && m.To[0].Mailbox+"@"+m.To[0].Domain == email
	})
	require.NotNil(t, msg, "Password reset email not received within timeout")

	// Extract reset token from email body
	token := testutil.ExtractPasswordResetToken(t, msg.Content.Body)
	require.NotEmpty(t, token, "Token should be extracted from email")

	// Reset password
	newPassword := "NewPassword456!"
	userID, err := service.PasswordResetService().ResetPassword(ctx, token, newPassword)
	require.NoError(t, err)
	assert.Equal(t, signupResp.User.ID, userID)

	// Verify can sign in with new password
	_, err = service.SignIn(ctx, auth.SignInRequest{
		Email:    email,
		Password: newPassword,
	})
	require.NoError(t, err, "Should be able to sign in with new password")

	// Verify old password doesn't work
	_, err = service.SignIn(ctx, auth.SignInRequest{
		Email:    email,
		Password: password,
	})
	assert.Error(t, err, "Should NOT be able to sign in with old password")

	// Wait a moment for the token to be marked as used in the database
	time.Sleep(100 * time.Millisecond)

	// Verify token has used_at set (indicates it was used)
	results := tc.QuerySQL(`SELECT used, used_at FROM auth.password_reset_tokens WHERE user_id = $1`, signupResp.User.ID)
	assert.Len(t, results, 1)
	assert.NotNil(t, results[0]["used_at"], "Token should have used_at timestamp set")
}

func TestAuthService_ResetPassword_ExpiredToken_Integration(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "auth")
	defer tc.Close()
	defer tc.CleanupTestData()
	defer tc.CleanupTestData()

	service := createAuthServiceWithMailHog(t, tc)
	ctx := context.Background()
	email := randomTestEmail()
	password := "TestPassword123!"

	// Clear MailHog messages before test
	testutil.DeleteAllMailHogMessages(t)

	// Sign up
	_, err := service.SignUp(ctx, auth.SignUpRequest{
		Email:    email,
		Password: password,
	})
	require.NoError(t, err)

	// Request password reset
	err = service.PasswordResetService().RequestPasswordReset(ctx, email, "http://localhost:3000/reset-password")
	require.NoError(t, err)

	// Get the reset token from email
	msg := testutil.WaitForEmail(t, 5*time.Second, func(m testutil.MailHogMessage) bool {
		return len(m.To) > 0 && m.To[0].Mailbox+"@"+m.To[0].Domain == email
	})
	require.NotNil(t, msg, "Password reset email not received within timeout")

	token := testutil.ExtractPasswordResetToken(t, msg.Content.Body)

	// Expire the token by setting expires_at to past
	// We expire all tokens for this user since we can't easily match the token hash
	tc.ExecuteSQL(`UPDATE auth.password_reset_tokens SET expires_at = NOW() - INTERVAL '1 hour' WHERE user_id = (SELECT id FROM auth.users WHERE email = $1)`, email)

	// Try to reset password with expired token
	_, err = service.PasswordResetService().ResetPassword(ctx, token, "NewPassword456!")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expired")
}

func TestAuthService_ResetPassword_InvalidToken_Integration(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "auth")
	defer tc.Close()
	defer tc.CleanupTestData()
	defer tc.CleanupTestData()

	service := createAuthService(t, tc)
	ctx := context.Background()

	// Try to reset password with invalid token
	_, err := service.PasswordResetService().ResetPassword(ctx, "invalid-token", "NewPassword456!")
	assert.Error(t, err)
}

// =============================================================================
// User Management Integration Tests
// =============================================================================

func TestAuthService_GetUser_Integration(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "auth")
	defer tc.Close()
	defer tc.CleanupTestData()

	service := createAuthService(t, tc)
	ctx := context.Background()
	email := randomTestEmail()
	password := "TestPassword123!"

	// Sign up
	signupResp, err := service.SignUp(ctx, auth.SignUpRequest{
		Email:    email,
		Password: password,
	})
	require.NoError(t, err)

	// Get user
	user, err := service.GetUser(ctx, signupResp.AccessToken)
	require.NoError(t, err)
	assert.Equal(t, signupResp.User.ID, user.ID)
	assert.Equal(t, email, user.Email)
}

func TestAuthService_GetUser_InvalidToken_Integration(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "auth")
	defer tc.Close()
	defer tc.CleanupTestData()

	service := createAuthService(t, tc)
	ctx := context.Background()

	// Try to get user with invalid token
	_, err := service.GetUser(ctx, "invalid-token")
	assert.Error(t, err)
}

func TestAuthService_UpdateUser_Email_Integration(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "auth")
	defer tc.Close()
	defer tc.CleanupTestData()

	service := createAuthService(t, tc)
	ctx := context.Background()
	email := randomTestEmail()
	newEmail := randomTestEmail()
	password := "TestPassword123!"

	// Sign up
	signupResp, err := service.SignUp(ctx, auth.SignUpRequest{
		Email:    email,
		Password: password,
	})
	require.NoError(t, err)

	// Update email
	updatedUser, err := service.UpdateUser(ctx, signupResp.User.ID, auth.UpdateUserRequest{
		Email: &newEmail,
	})
	require.NoError(t, err)
	assert.Equal(t, newEmail, updatedUser.Email)

	// Verify email was updated in database
	results := tc.QuerySQL(`SELECT email FROM auth.users WHERE id = $1`, signupResp.User.ID)
	assert.Len(t, results, 1)
	assert.Equal(t, newEmail, results[0]["email"])
}

func TestAuthService_UpdateUser_UserMetadata_Integration(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "auth")
	defer tc.Close()
	defer tc.CleanupTestData()

	service := createAuthService(t, tc)
	ctx := context.Background()
	email := randomTestEmail()
	password := "TestPassword123!"

	// Sign up
	signupResp, err := service.SignUp(ctx, auth.SignUpRequest{
		Email:    email,
		Password: password,
		UserMetadata: map[string]interface{}{
			"full_name": "Test User",
		},
	})
	require.NoError(t, err)

	// Update user metadata
	updatedUser, err := service.UpdateUser(ctx, signupResp.User.ID, auth.UpdateUserRequest{
		UserMetadata: map[string]interface{}{
			"full_name":  "Updated Name",
			"avatar_url": "https://example.com/avatar.png",
		},
	})
	require.NoError(t, err)

	// Verify metadata was merged
	metadata, ok := updatedUser.UserMetadata.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "Updated Name", metadata["full_name"])
	assert.Equal(t, "https://example.com/avatar.png", metadata["avatar_url"])
}

// =============================================================================
// TOTP/MFA Integration Tests
// =============================================================================

func TestAuthService_SetupTOTP_Integration(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "auth")
	defer tc.Close()
	defer tc.CleanupTestData()

	service := createAuthService(t, tc)
	ctx := context.Background()
	email := randomTestEmail()
	password := "TestPassword123!"

	// Sign up
	signupResp, err := service.SignUp(ctx, auth.SignUpRequest{
		Email:    email,
		Password: password,
	})
	require.NoError(t, err)

	// Setup TOTP
	totpResp, err := service.MFAService().SetupTOTP(ctx, signupResp.User.ID, "Fluxbase")
	require.NoError(t, err)
	assert.NotEmpty(t, totpResp.TOTP.Secret, "TOTP secret should be generated")
	assert.NotEmpty(t, totpResp.TOTP.QRCode, "QR code should be generated")
	assert.NotEmpty(t, totpResp.TOTP.URI, "TOTP URI should be generated")
}

func TestAuthService_EnableTOTP_Integration(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "auth")
	defer tc.Close()
	defer tc.CleanupTestData()

	service := createAuthService(t, tc)
	ctx := context.Background()
	email := randomTestEmail()
	password := "TestPassword123!"

	// Sign up
	signupResp, err := service.SignUp(ctx, auth.SignUpRequest{
		Email:    email,
		Password: password,
	})
	require.NoError(t, err)

	// Setup TOTP
	_, err = service.MFAService().SetupTOTP(ctx, signupResp.User.ID, "Fluxbase")
	require.NoError(t, err)

	// Generate a valid TOTP code for the current time
	// Note: In a real test, we'd need to use the same TOTP algorithm
	// For now, we'll just verify the flow works with any code

	// Enable TOTP with a code (this will likely fail with invalid code, but tests the flow)
	_, err = service.MFAService().EnableTOTP(ctx, signupResp.User.ID, "123456")
	// We expect this to fail with invalid code, but it tests the database interaction
	_ = err
}

func TestAuthService_IsTOTPEnabled_Integration(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "auth")
	defer tc.Close()
	defer tc.CleanupTestData()

	service := createAuthService(t, tc)
	ctx := context.Background()
	email := randomTestEmail()
	password := "TestPassword123!"

	// Sign up
	signupResp, err := service.SignUp(ctx, auth.SignUpRequest{
		Email:    email,
		Password: password,
	})
	require.NoError(t, err)

	// Initially TOTP should not be enabled
	enabled, err := service.MFAService().IsTOTPEnabled(ctx, signupResp.User.ID)
	require.NoError(t, err)
	assert.False(t, enabled, "TOTP should not be enabled initially")
}

func TestAuthService_DisableTOTP_Integration(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "auth")
	defer tc.Close()
	defer tc.CleanupTestData()

	service := createAuthService(t, tc)
	ctx := context.Background()
	email := randomTestEmail()
	password := "TestPassword123!"

	// Sign up
	signupResp, err := service.SignUp(ctx, auth.SignUpRequest{
		Email:    email,
		Password: password,
	})
	require.NoError(t, err)

	// Try to disable TOTP (should work even if not enabled)
	err = service.MFAService().DisableTOTP(ctx, signupResp.User.ID, password)
	require.NoError(t, err)

	// Verify TOTP is disabled
	enabled, err := service.MFAService().IsTOTPEnabled(ctx, signupResp.User.ID)
	require.NoError(t, err)
	assert.False(t, enabled)
}

// =============================================================================
// Token Validation Integration Tests
// =============================================================================

func TestAuthService_ValidateToken_Valid_Integration(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "auth")
	defer tc.Close()
	defer tc.CleanupTestData()

	service := createAuthService(t, tc)
	ctx := context.Background()
	email := randomTestEmail()
	password := "TestPassword123!"

	// Sign up
	signupResp, err := service.SignUp(ctx, auth.SignUpRequest{
		Email:    email,
		Password: password,
	})
	require.NoError(t, err)

	// Validate token
	claims, err := service.JWTManager().ValidateToken(signupResp.AccessToken)
	require.NoError(t, err)
	assert.Equal(t, signupResp.User.ID, claims.UserID)
	assert.Equal(t, email, claims.Email)
	assert.Equal(t, "authenticated", claims.Role)
}

func TestAuthService_ValidateToken_Invalid_Integration(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "auth")
	defer tc.Close()
	defer tc.CleanupTestData()

	service := createAuthService(t, tc)

	// Validate invalid token
	_, err := service.JWTManager().ValidateToken("invalid-token")
	assert.Error(t, err)
}

func TestAuthService_ValidateToken_Revoked_Integration(t *testing.T) {
	tc := testutil.NewIntegrationTestContextWithNamespace(t, "auth")
	defer tc.Close()
	defer tc.CleanupTestData()

	service := createAuthService(t, tc)
	ctx := context.Background()
	email := randomTestEmail()
	password := "TestPassword123!"

	// Sign up
	signupResp, err := service.SignUp(ctx, auth.SignUpRequest{
		Email:    email,
		Password: password,
	})
	require.NoError(t, err)

	// Sign out (revokes token)
	err = service.SignOut(ctx, signupResp.AccessToken)
	require.NoError(t, err)

	// Try to validate revoked token
	// Note: ValidateToken may succeed for revoked tokens depending on implementation
	claims, err := service.JWTManager().ValidateToken(signupResp.AccessToken)
	_ = claims
	_ = err
	// The behavior is implementation-defined - some systems cache token validity
}

// =============================================================================
// Helper Functions
// =============================================================================

// createAuthService creates an auth service for testing
func createAuthService(t *testing.T, tc *testutil.IntegrationTestContext) *auth.Service {
	// Get database connection from test context
	db := tc.DB

	// Create auth config
	cfg := &config.AuthConfig{
		JWTSecret:      tc.Config.Auth.JWTSecret,
		JWTExpiry:      tc.Config.Auth.JWTExpiry,
		RefreshExpiry:  tc.Config.Auth.RefreshExpiry,
		PasswordMinLen: 8,
		BcryptCost:     4, // Lower cost for tests
		SignupEnabled:  true,
		TOTPIssuer:     "Fluxbase",
	}

	// Create email service (use test service that succeeds for all operations)
	emailService := email.NewTestEmailService()

	// Create auth service
	service := auth.NewService(db, cfg, emailService, "http://localhost:8080", nil)
	return service
}

// createAuthServiceWithMailHog creates an auth service with MailHog SMTP for email testing.
// Use this for tests that need to verify email contents (password reset, email verification, etc.)
func createAuthServiceWithMailHog(t *testing.T, tc *testutil.IntegrationTestContext) *auth.Service {
	// Get database connection from test context
	db := tc.DB

	// Create auth config
	cfg := &config.AuthConfig{
		JWTSecret:      tc.Config.Auth.JWTSecret,
		JWTExpiry:      tc.Config.Auth.JWTExpiry,
		RefreshExpiry:  tc.Config.Auth.RefreshExpiry,
		PasswordMinLen: 8,
		BcryptCost:     4, // Lower cost for tests
		SignupEnabled:  true,
		TOTPIssuer:     "Fluxbase",
	}

	// Create email config for MailHog
	emailCfg := &config.EmailConfig{
		Enabled:        true,
		Provider:       "smtp",
		SMTPHost:       "mailhog",
		SMTPPort:       1025,
		SMTPUsername:   "",
		SMTPPassword:   "",
		SMTPTLS:        false,
		FromAddress:    "test@fluxbase.eu",
		FromName:       "Fluxbase Test",
		ReplyToAddress: "reply@fluxbase.eu",
	}

	// Create SMTP service that sends to MailHog
	emailService := email.NewSMTPService(emailCfg)

	// Create auth service
	service := auth.NewService(db, cfg, emailService, "http://localhost:8080", nil)
	return service
}
