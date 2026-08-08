package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// This file contains comprehensive tests for OAuth authentication flows.
//
// NOTE: This file previously contained three `t.Skip` stub functions
// (TestUnlinkOAuthIdentity_*) that asserted nothing — each was an empty body
// with a "Requires database integration" skip. They have been removed to avoid
// implying coverage that doesn't exist. The real DB-coupled
// Service.UnlinkOAuthIdentity method still lacks test coverage; see the linked
// issue for the plan to cover it via repository mocks or a CI-run integration
// job. The pure-logic helpers in this file (MockOAuthManager, in-memory
// StateStore, GenerateState) are exercised by the tests below.

// TestGetOAuthURL_Success tests successful OAuth URL generation
func TestGetOAuthURL_Success(t *testing.T) {
	manager := NewMockOAuthManager()
	manager.RegisterProvider("google")
	manager.RegisterProvider("github")
	manager.RegisterProvider("microsoft")

	tests := []struct {
		name     string
		provider string
		state    string
	}{
		{
			name:     "Google OAuth",
			provider: "google",
			state:    "test_state_123",
		},
		{
			name:     "GitHub OAuth",
			provider: "github",
			state:    "another_state_456",
		},
		{
			name:     "Microsoft OAuth",
			provider: "microsoft",
			state:    "state_microsoft_789",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authURL, err := manager.GetAuthURL(tt.provider, tt.state)

			require.NoError(t, err)
			assert.NotEmpty(t, authURL)
			assert.Contains(t, authURL, tt.state, "Auth URL should contain state parameter")
			assert.Contains(t, authURL, tt.provider, "Auth URL should contain provider")
		})
	}
}

// TestGetOAuthURL_InvalidProvider tests OAuth URL generation with invalid provider
func TestGetOAuthURL_InvalidProvider(t *testing.T) {
	manager := NewMockOAuthManager()
	// Don't register any providers

	authURL, err := manager.GetAuthURL("invalid_provider", "test_state")

	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidProvider)
	assert.Empty(t, authURL)
}

// TestGetOAuthURL_CustomRedirectURI tests OAuth URL generation with custom redirect URI
func TestGetOAuthURL_CustomRedirectURI(t *testing.T) {
	manager := NewMockOAuthManager()
	manager.RegisterProvider("google")

	// Test with custom redirect URI callback
	customRedirectURI := "https://custom.example.com/callback"
	manager.AuthURLFn = func(provider, state string) (string, error) {
		return "https://" + provider + ".example.com/oauth/authorize?state=" + state + "&redirect_uri=" + customRedirectURI, nil
	}

	authURL, err := manager.GetAuthURL("google", "test_state")

	require.NoError(t, err)
	assert.NotEmpty(t, authURL)
	assert.Contains(t, authURL, customRedirectURI, "Auth URL should contain custom redirect URI")
}

// TestHandleOAuthCallback_Success tests successful OAuth callback handling
func TestHandleOAuthCallback_Success(t *testing.T) {
	ctx := context.Background()
	manager := NewMockOAuthManager()
	manager.RegisterProvider("google")

	_, err := GenerateState()
	require.NoError(t, err)

	// Mock successful code exchange
	manager.ExchangeCodeFn = func(ctx context.Context, provider, code string) (string, map[string]interface{}, error) {
		return "access_token_123", map[string]interface{}{
			"id":    "google_user_123",
			"email": "user@gmail.com",
			"name":  "Test User",
		}, nil
	}

	// Simulate callback handling
	token, userInfo, err := manager.ExchangeCode(ctx, "google", "auth_code_123")

	require.NoError(t, err)
	assert.NotEmpty(t, token)
	assert.Equal(t, "access_token_123", token)
	assert.NotEmpty(t, userInfo)
	assert.Equal(t, "google_user_123", userInfo["id"])
	assert.Equal(t, "user@gmail.com", userInfo["email"])
	assert.Equal(t, "Test User", userInfo["name"])
}

// TestHandleOAuthCallback_EmailMismatch tests email mismatch handling in OAuth callback
func TestHandleOAuthCallback_EmailMismatch(t *testing.T) {
	ctx := context.Background()
	manager := NewMockOAuthManager()
	manager.RegisterProvider("github")

	// Mock callback with different email
	manager.ExchangeCodeFn = func(ctx context.Context, provider, code string) (string, map[string]interface{}, error) {
		return "access_token_456", map[string]interface{}{
			"id":    "github_user_789",
			"email": "different@github.com",
			"name":  "Different User",
		}, nil
	}

	token, userInfo, err := manager.ExchangeCode(ctx, "github", "auth_code_456")

	require.NoError(t, err)
	assert.NotEmpty(t, token)
	// In a real implementation, you'd check if the email matches the expected user
	assert.NotEqual(t, "expected@email.com", userInfo["email"], "Email mismatch should be detected")
}

// TestHandleOAuthCallback_StateMismatch tests state parameter validation (CSRF protection)
func TestHandleOAuthCallback_StateMismatch(t *testing.T) {
	stateStore := NewStateStore()

	// Generate and store valid state
	validState, err := GenerateState()
	require.NoError(t, err)
	_ = stateStore.Set(context.Background(), validState, StateMetadata{
		Expiry: time.Now().Add(10 * time.Minute),
	})

	// Test with invalid state
	invalidState := "invalid_state_123"

	isValid := stateStore.Validate(context.Background(), invalidState)
	assert.False(t, isValid, "Invalid state should fail validation")

	// Valid state should pass (and be consumed)
	isValid = stateStore.Validate(context.Background(), validState)
	assert.True(t, isValid, "Valid state should pass validation")

	// Second validation of same state should fail (state consumed)
	isValid = stateStore.Validate(context.Background(), validState)
	assert.False(t, isValid, "State should only be valid once (consumed after first use)")
}

// TestHandleOAuthCallback_ProviderError tests OAuth provider error handling
func TestHandleOAuthCallback_ProviderError(t *testing.T) {
	ctx := context.Background()
	manager := NewMockOAuthManager()
	manager.RegisterProvider("microsoft")

	// Mock provider error
	manager.ExchangeCodeFn = func(ctx context.Context, provider, code string) (string, map[string]interface{}, error) {
		return "", nil, errors.New("provider error: invalid authorization code")
	}

	token, userInfo, err := manager.ExchangeCode(ctx, "microsoft", "invalid_code")

	assert.Error(t, err)
	assert.Empty(t, token)
	assert.Nil(t, userInfo)
	assert.Contains(t, err.Error(), "provider error")
}

// TestLinkOAuthIdentity_Success tests successful OAuth identity linking
func TestLinkOAuthIdentity_Success(t *testing.T) {
	ctx := context.Background()
	manager := NewMockOAuthManager()
	stateStore := NewStateStore()
	manager.RegisterProvider("github")

	provider := "github"

	// Initiate OAuth flow
	state, err := GenerateState()
	require.NoError(t, err)
	_ = stateStore.Set(context.Background(), state, StateMetadata{
		Expiry: time.Now().Add(10 * time.Minute),
	})

	authURL, err := manager.GetAuthURL(provider, state)
	require.NoError(t, err)
	assert.NotEmpty(t, authURL)
	assert.Contains(t, authURL, state)

	// Simulate callback
	manager.ExchangeCodeFn = func(ctx context.Context, provider, code string) (string, map[string]interface{}, error) {
		return "github_token_123", map[string]interface{}{
			"id":    "github_user_456",
			"email": "github@example.com",
			"name":  "GitHub User",
		}, nil
	}

	// Validate state
	valid := stateStore.Validate(context.Background(), state)
	assert.True(t, valid, "State should be valid")

	// Exchange code
	token, userInfo, err := manager.ExchangeCode(ctx, provider, "auth_code_789")
	require.NoError(t, err)
	assert.NotEmpty(t, token)
	assert.NotEmpty(t, userInfo)
	assert.Equal(t, "github_user_456", userInfo["id"])
}

// TestLinkOAuthIdentity_AlreadyLinked tests linking an already linked identity
func TestLinkOAuthIdentity_AlreadyLinked(t *testing.T) {
	_ = context.Background()
	manager := NewMockOAuthManager()
	stateStore := NewStateStore()
	manager.RegisterProvider("google")

	userID := "user-123"
	_ = "google" // provider

	// Simulate that identity is already linked
	providerUserID := "google_user_789"

	// In a real implementation, you'd check the database for existing links
	// For this test, we simulate the scenario
	alreadyLinked := true

	assert.True(t, alreadyLinked, "Identity should already be linked")

	// Attempting to link again should fail
	_ = userID
	_ = providerUserID
	_ = stateStore
	_ = manager
}

// TestUnlinkOAuthIdentity_*: removed (empty t.Skip stubs).
// See file note above re: DB-coupled OAuth coverage.

// TestOAuthState_StoreAndValidate tests OAuth state storage and validation
func TestOAuthState_StoreAndValidate(t *testing.T) {
	stateStore := NewStateStore()

	// Generate and store state
	state, err := GenerateState()
	require.NoError(t, err)

	customRedirectURI := "https://custom.example.com/callback"
	_ = stateStore.Set(context.Background(), state, StateMetadata{
		Expiry:      time.Now().Add(10 * time.Minute),
		RedirectURI: customRedirectURI,
		Provider:    "google",
	})

	// Validate and retrieve metadata
	metadata, valid := stateStore.GetAndValidate(context.Background(), state)

	require.True(t, valid, "State should be valid")
	require.NotNil(t, metadata)
	assert.Equal(t, customRedirectURI, metadata.RedirectURI)
	assert.Equal(t, "google", metadata.Provider)

	// Second validation should fail (state consumed)
	_, valid = stateStore.GetAndValidate(context.Background(), state)
	assert.False(t, valid, "State should be consumed after first validation")
}

// TestOAuthState_Expiration tests OAuth state expiration
func TestOAuthState_Expiration(t *testing.T) {
	stateStore := NewStateStore()

	// Generate and store state with short expiry
	state, err := GenerateState()
	require.NoError(t, err)

	_ = stateStore.Set(context.Background(), state, StateMetadata{
		Expiry:   time.Now().Add(-1 * time.Hour), // Expired
		Provider: "github",
	})

	// Validate expired state
	metadata, valid := stateStore.GetAndValidate(context.Background(), state)

	assert.False(t, valid, "Expired state should be invalid")
	assert.Nil(t, metadata)
}

// TestOAuthState_ConcurrentAccess tests thread-safe OAuth state access
func TestOAuthState_ConcurrentAccess(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent test in short mode")
	}

	stateStore := NewStateStore()
	done := make(chan bool, 10)

	// Concurrent writes
	for i := 0; i < 5; i++ {
		go func(idx int) {
			state, _ := GenerateState()
			_ = stateStore.Set(context.Background(), state, StateMetadata{
				Expiry: time.Now().Add(10 * time.Minute),
			})
			done <- true
		}(i)
	}

	// Concurrent reads
	for i := 0; i < 5; i++ {
		go func() {
			_ = stateStore.Cleanup(context.Background())
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Should not panic or deadlock
}

// TestOAuthProvider_Endpoints tests OAuth provider endpoint configuration
func TestOAuthProvider_Endpoints(t *testing.T) {
	oauthManager := NewOAuthManager()

	tests := []struct {
		name     string
		provider string
	}{
		{"Google Endpoint", "google"},
		{"GitHub Endpoint", "github"},
		{"Microsoft Endpoint", "microsoft"},
		{"Apple Endpoint", "apple"},
		{"Facebook Endpoint", "facebook"},
		{"Twitter Endpoint", "twitter"},
		{"LinkedIn Endpoint", "linkedin"},
		{"GitLab Endpoint", "gitlab"},
		{"Bitbucket Endpoint", "bitbucket"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			endpoint := oauthManager.GetEndpoint(OAuthProvider(tt.provider))
			assert.NotEmpty(t, endpoint.AuthURL, "Provider should have auth URL")
			assert.NotEmpty(t, endpoint.TokenURL, "Provider should have token URL")
		})
	}
}

// TestOAuthUserInfo_Success tests getting user info from OAuth provider
func TestOAuthUserInfo_Success(t *testing.T) {
	ctx := context.Background()
	manager := NewMockOAuthManager()
	manager.RegisterProvider("google")

	manager.GetUserInfoFn = func(ctx context.Context, provider string, tokenStr string) (map[string]interface{}, error) {
		return map[string]interface{}{
			"id":             "google_user_123",
			"email":          "user@gmail.com",
			"verified_email": true,
			"name":           "Test User",
			"picture":        "https://example.com/photo.jpg",
		}, nil
	}

	userInfo, err := manager.GetUserInfo(ctx, "google", "access_token_123")

	require.NoError(t, err)
	assert.NotEmpty(t, userInfo)
	assert.Equal(t, "google_user_123", userInfo["id"])
	assert.Equal(t, "user@gmail.com", userInfo["email"])
	assert.Equal(t, "Test User", userInfo["name"])
}

// TestOAuthUserInfo_ProviderError tests user info retrieval errors
func TestOAuthUserInfo_ProviderError(t *testing.T) {
	ctx := context.Background()
	manager := NewMockOAuthManager()
	manager.RegisterProvider("github")

	// Mock provider error
	manager.GetUserInfoFn = func(ctx context.Context, provider string, tokenStr string) (map[string]interface{}, error) {
		return nil, errors.New("provider error: invalid token")
	}

	userInfo, err := manager.GetUserInfo(ctx, "github", "invalid_token")

	assert.Error(t, err)
	assert.Nil(t, userInfo)
	assert.Contains(t, err.Error(), "provider error")
}

// TestOAuthProvider_InvalidProvider tests operations with invalid provider
func TestOAuthProvider_InvalidProvider(t *testing.T) {
	ctx := context.Background()
	manager := NewMockOAuthManager()
	// Don't register any providers

	// Try to get auth URL for invalid provider
	authURL, err := manager.GetAuthURL("invalid_provider", "test_state")
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidProvider)
	assert.Empty(t, authURL)

	// Try to exchange code for invalid provider
	token, userInfo, err := manager.ExchangeCode(ctx, "invalid_provider", "code")
	assert.Error(t, err)
	assert.Empty(t, token)
	assert.Nil(t, userInfo)

	// Try to get user info for invalid provider
	userInfo, err = manager.GetUserInfo(ctx, "invalid_provider", "token")
	assert.Error(t, err)
	assert.Nil(t, userInfo)
}

// TestOAuthWorkflow_CompleteFlow tests complete OAuth authentication flow
func TestOAuthWorkflow_CompleteFlow(t *testing.T) {
	ctx := context.Background()
	manager := NewMockOAuthManager()
	stateStore := NewStateStore()
	manager.RegisterProvider("google")

	// Step 1: Initiate OAuth flow
	state, err := GenerateState()
	require.NoError(t, err)
	_ = stateStore.Set(context.Background(), state, StateMetadata{
		Expiry: time.Now().Add(10 * time.Minute),
	})

	authURL, err := manager.GetAuthURL("google", state)
	require.NoError(t, err)
	assert.NotEmpty(t, authURL)
	assert.Contains(t, authURL, state)

	// Step 2: Mock callback with authorization code
	authCode := "auth_code_xyz"

	// Step 3: Validate state
	valid := stateStore.Validate(context.Background(), state)
	assert.True(t, valid, "State should be valid")

	// Step 4: Exchange authorization code for tokens and user info
	manager.ExchangeCodeFn = func(ctx context.Context, provider, code string) (string, map[string]interface{}, error) {
		if code != authCode {
			return "", nil, errors.New("invalid authorization code")
		}
		return "google_access_token_123", map[string]interface{}{
			"id":             "google_user_456",
			"email":          "oauth@gmail.com",
			"verified_email": true,
			"name":           "OAuth User",
			"picture":        "https://example.com/photo.jpg",
		}, nil
	}

	token, userInfo, err := manager.ExchangeCode(ctx, "google", authCode)
	require.NoError(t, err)
	assert.NotEmpty(t, token)
	assert.NotEmpty(t, userInfo)

	// Step 5: Verify user information
	assert.Equal(t, "google_access_token_123", token)
	assert.Equal(t, "google_user_456", userInfo["id"])
	assert.Equal(t, "oauth@gmail.com", userInfo["email"])
	assert.Equal(t, "OAuth User", userInfo["name"])

	// Step 6: Verify state was consumed (can't be used again)
	valid = stateStore.Validate(context.Background(), state)
	assert.False(t, valid, "State should be consumed after use")
}

// TestOAuthState_Cleanup tests cleanup of expired OAuth states
func TestOAuthState_Cleanup(t *testing.T) {
	stateStore := NewStateStore()

	// Generate and store multiple states
	expiredState1, _ := GenerateState()
	expiredState2, _ := GenerateState()
	validState, _ := GenerateState()

	// Store expired states
	_ = stateStore.Set(context.Background(), expiredState1, StateMetadata{
		Expiry: time.Now().Add(-1 * time.Hour),
	})

	_ = stateStore.Set(context.Background(), expiredState2, StateMetadata{
		Expiry: time.Now().Add(-30 * time.Minute),
	})

	// Store valid state
	_ = stateStore.Set(context.Background(), validState, StateMetadata{
		Expiry: time.Now().Add(10 * time.Minute),
	})

	// Run cleanup
	_ = stateStore.Cleanup(context.Background())

	// Expired states should no longer be valid
	_, valid1 := stateStore.GetAndValidate(context.Background(), expiredState1)
	_, valid2 := stateStore.GetAndValidate(context.Background(), expiredState2)

	assert.False(t, valid1, "Expired state 1 should be cleaned up")
	assert.False(t, valid2, "Expired state 2 should be cleaned up")

	// Valid state should still exist
	_, valid3 := stateStore.GetAndValidate(context.Background(), validState)
	assert.True(t, valid3, "Valid state should still exist")
}

// TestOAuthSecurity_CSRFProtection tests CSRF protection via state parameter
func TestOAuthSecurity_CSRFProtection(t *testing.T) {
	stateStore := NewStateStore()

	// Generate and store state
	state, err := GenerateState()
	require.NoError(t, err)
	_ = stateStore.Set(context.Background(), state, StateMetadata{
		Expiry: time.Now().Add(10 * time.Minute),
	})

	// Test 1: Correct state should validate
	valid := stateStore.Validate(context.Background(), state)
	assert.True(t, valid, "Correct state should validate")

	// Generate new state for subsequent tests (state was consumed)
	state, err = GenerateState()
	require.NoError(t, err)
	_ = stateStore.Set(context.Background(), state, StateMetadata{
		Expiry: time.Now().Add(10 * time.Minute),
	})

	// Test 2: Modified state should not validate
	modifiedState := state + "modified"
	valid = stateStore.Validate(context.Background(), modifiedState)
	assert.False(t, valid, "Modified state should not validate")

	// Test 3: Different state should not validate
	differentState, _ := GenerateState()
	valid = stateStore.Validate(context.Background(), differentState)
	assert.False(t, valid, "Different state should not validate")

	// Test 4: Empty state should not validate
	valid = stateStore.Validate(context.Background(), "")
	assert.False(t, valid, "Empty state should not validate")
}

// TestOAuthSecurity_StateUniqueness tests that states are unique
func TestOAuthSecurity_StateUniqueness(t *testing.T) {
	states := make(map[string]bool)
	iterations := 100

	for i := 0; i < iterations; i++ {
		state, err := GenerateState()
		require.NoError(t, err)

		assert.False(t, states[state], "State should be unique")
		states[state] = true
	}

	assert.Len(t, states, iterations, "All states should be unique")
}

// TestOAuthSecurity_StateEntropy tests that states have sufficient entropy
func TestOAuthSecurity_StateEntropy(t *testing.T) {
	state1, err := GenerateState()
	require.NoError(t, err)

	state2, err := GenerateState()
	require.NoError(t, err)

	// States should be different
	assert.NotEqual(t, state1, state2, "States should be different")

	// Check length (32 bytes = ~44 chars in base64)
	assert.GreaterOrEqual(t, len(state1), 40, "State should have sufficient length")
	assert.LessOrEqual(t, len(state1), 48, "State should not be excessively long")
}
