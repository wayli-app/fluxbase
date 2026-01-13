package auth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Error Variable Tests
// =============================================================================

func TestIdentityErrors(t *testing.T) {
	t.Run("error types are defined", func(t *testing.T) {
		assert.NotNil(t, ErrIdentityNotFound)
		assert.NotNil(t, ErrIdentityAlreadyLinked)
	})

	t.Run("error messages are meaningful", func(t *testing.T) {
		assert.Contains(t, ErrIdentityNotFound.Error(), "not found")
		assert.Contains(t, ErrIdentityAlreadyLinked.Error(), "already linked")
	})

	t.Run("errors are distinct", func(t *testing.T) {
		assert.NotEqual(t, ErrIdentityNotFound, ErrIdentityAlreadyLinked)
	})

	t.Run("error messages are exact", func(t *testing.T) {
		assert.Equal(t, "identity not found", ErrIdentityNotFound.Error())
		assert.Equal(t, "identity is already linked to another user", ErrIdentityAlreadyLinked.Error())
	})
}

// =============================================================================
// UserIdentity Struct Tests
// =============================================================================

func TestUserIdentity_Struct(t *testing.T) {
	t.Run("creates identity with all fields", func(t *testing.T) {
		now := time.Now()
		email := "user@example.com"

		identity := UserIdentity{
			ID:             "identity-123",
			UserID:         "user-456",
			Provider:       "google",
			ProviderUserID: "google-user-789",
			Email:          &email,
			IdentityData: map[string]interface{}{
				"name":    "Test User",
				"picture": "https://example.com/avatar.jpg",
			},
			CreatedAt: now,
			UpdatedAt: now,
		}

		assert.Equal(t, "identity-123", identity.ID)
		assert.Equal(t, "user-456", identity.UserID)
		assert.Equal(t, "google", identity.Provider)
		assert.Equal(t, "google-user-789", identity.ProviderUserID)
		assert.Equal(t, "user@example.com", *identity.Email)
		assert.Equal(t, "Test User", identity.IdentityData["name"])
	})

	t.Run("handles nil optional fields", func(t *testing.T) {
		identity := UserIdentity{
			ID:             "identity-123",
			UserID:         "user-456",
			Provider:       "github",
			ProviderUserID: "github-user-789",
		}

		assert.Nil(t, identity.Email)
		assert.Nil(t, identity.IdentityData)
	})
}

func TestUserIdentity_Providers(t *testing.T) {
	providers := []string{
		"google",
		"github",
		"microsoft",
		"apple",
		"facebook",
		"twitter",
		"linkedin",
		"gitlab",
		"bitbucket",
		"saml",
		"oidc",
	}

	for _, provider := range providers {
		t.Run("provider_"+provider, func(t *testing.T) {
			identity := UserIdentity{
				ID:             "identity-123",
				UserID:         "user-456",
				Provider:       provider,
				ProviderUserID: provider + "-user-789",
			}

			assert.Equal(t, provider, identity.Provider)
		})
	}
}

func TestUserIdentity_IdentityData(t *testing.T) {
	t.Run("empty identity data", func(t *testing.T) {
		identity := UserIdentity{
			IdentityData: map[string]interface{}{},
		}

		assert.NotNil(t, identity.IdentityData)
		assert.Empty(t, identity.IdentityData)
	})

	t.Run("complex identity data", func(t *testing.T) {
		identity := UserIdentity{
			IdentityData: map[string]interface{}{
				"name":         "Test User",
				"email":        "test@example.com",
				"picture":      "https://example.com/avatar.jpg",
				"verified":     true,
				"login_count":  42,
				"last_login":   "2026-01-13T12:00:00Z",
				"permissions":  []string{"read", "write"},
				"organization": map[string]interface{}{"id": "org-123", "name": "Acme Corp"},
			},
		}

		assert.Equal(t, "Test User", identity.IdentityData["name"])
		assert.Equal(t, true, identity.IdentityData["verified"])
		assert.Equal(t, 42, identity.IdentityData["login_count"])
	})

	t.Run("nil vs empty identity data", func(t *testing.T) {
		identityNil := UserIdentity{IdentityData: nil}
		identityEmpty := UserIdentity{IdentityData: map[string]interface{}{}}

		assert.Nil(t, identityNil.IdentityData)
		assert.NotNil(t, identityEmpty.IdentityData)
	})
}

func TestUserIdentity_Timestamps(t *testing.T) {
	now := time.Now()
	past := now.Add(-24 * time.Hour)

	identity := UserIdentity{
		CreatedAt: past,
		UpdatedAt: now,
	}

	assert.True(t, identity.CreatedAt.Before(identity.UpdatedAt))
	assert.True(t, identity.UpdatedAt.After(identity.CreatedAt))
}

// =============================================================================
// Repository Tests
// =============================================================================

func TestNewIdentityRepository(t *testing.T) {
	// Test that it doesn't panic with nil db
	repo := NewIdentityRepository(nil)
	assert.NotNil(t, repo)
}

func TestNewIdentityRepository_Fields(t *testing.T) {
	repo := NewIdentityRepository(nil)

	require.NotNil(t, repo)
	assert.Nil(t, repo.db)
}

// =============================================================================
// Service Tests
// =============================================================================

func TestNewIdentityService(t *testing.T) {
	// Test that it doesn't panic with nil dependencies
	svc := NewIdentityService(nil, nil, nil)
	assert.NotNil(t, svc)
}

func TestNewIdentityService_Fields(t *testing.T) {
	oauthManager := NewOAuthManager()
	stateStore := NewStateStore()
	repo := NewIdentityRepository(nil)

	svc := NewIdentityService(repo, oauthManager, stateStore)

	require.NotNil(t, svc)
	assert.Equal(t, repo, svc.repo)
	assert.Equal(t, oauthManager, svc.oauthManager)
	assert.Equal(t, stateStore, svc.stateStore)
}

func TestNewIdentityService_WithAllNil(t *testing.T) {
	svc := NewIdentityService(nil, nil, nil)

	require.NotNil(t, svc)
	assert.Nil(t, svc.repo)
	assert.Nil(t, svc.oauthManager)
	assert.Nil(t, svc.stateStore)
}

// =============================================================================
// IdentityService Method Structure Tests (without DB)
// =============================================================================

func TestIdentityService_LinkIdentityProvider_NoOAuthManager(t *testing.T) {
	stateStore := NewStateStore()
	svc := NewIdentityService(nil, nil, stateStore)

	// This will fail because oauthManager is nil
	_, _, err := svc.LinkIdentityProvider(nil, "user-123", "google")

	// Should get a panic or error due to nil oauthManager
	// We're just testing it doesn't crash catastrophically
	assert.Error(t, err)
}

func TestIdentityService_LinkIdentityProvider_WithOAuthManager(t *testing.T) {
	oauthManager := NewOAuthManager()
	stateStore := NewStateStore()
	svc := NewIdentityService(nil, oauthManager, stateStore)

	// Register a provider first
	err := oauthManager.RegisterProvider(ProviderGoogle, OAuthConfig{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		RedirectURL:  "https://example.com/callback",
		Scopes:       []string{"email", "profile"},
	})
	require.NoError(t, err)

	// This should work now
	authURL, state, err := svc.LinkIdentityProvider(nil, "user-123", "google")

	require.NoError(t, err)
	assert.NotEmpty(t, authURL)
	assert.NotEmpty(t, state)
	assert.Contains(t, authURL, "client_id=test-client-id")
}

func TestIdentityService_LinkIdentityCallback_InvalidState(t *testing.T) {
	oauthManager := NewOAuthManager()
	stateStore := NewStateStore()
	svc := NewIdentityService(nil, oauthManager, stateStore)

	// Try to callback with invalid state
	_, err := svc.LinkIdentityCallback(nil, "user-123", "google", "auth-code", "invalid-state")

	assert.Error(t, err)
	assert.Equal(t, ErrInvalidState, err)
}

func TestIdentityService_StateStoreIntegration(t *testing.T) {
	oauthManager := NewOAuthManager()
	stateStore := NewStateStore()
	svc := NewIdentityService(nil, oauthManager, stateStore)

	err := oauthManager.RegisterProvider(ProviderGoogle, OAuthConfig{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		RedirectURL:  "https://example.com/callback",
		Scopes:       []string{"email", "profile"},
	})
	require.NoError(t, err)

	// Generate state via service
	_, state, err := svc.LinkIdentityProvider(nil, "user-123", "google")
	require.NoError(t, err)

	// State should be stored in state store
	// Note: Validate consumes the state, so we can only check once
	valid := stateStore.Validate(state)
	assert.True(t, valid)

	// After validation, state should be consumed
	validAgain := stateStore.Validate(state)
	assert.False(t, validAgain)
}

// =============================================================================
// Provider-specific Tests
// =============================================================================

func TestUserIdentity_GoogleProvider(t *testing.T) {
	email := "user@gmail.com"
	identity := UserIdentity{
		ID:             "identity-google-123",
		UserID:         "user-456",
		Provider:       "google",
		ProviderUserID: "114823456789012345678",
		Email:          &email,
		IdentityData: map[string]interface{}{
			"sub":            "114823456789012345678",
			"name":           "Google User",
			"given_name":     "Google",
			"family_name":    "User",
			"picture":        "https://lh3.googleusercontent.com/a/default-user",
			"email":          "user@gmail.com",
			"email_verified": true,
			"locale":         "en",
		},
	}

	assert.Equal(t, "google", identity.Provider)
	assert.Equal(t, "114823456789012345678", identity.ProviderUserID)
	assert.Equal(t, true, identity.IdentityData["email_verified"])
}

func TestUserIdentity_GithubProvider(t *testing.T) {
	email := "user@github.com"
	identity := UserIdentity{
		ID:             "identity-github-123",
		UserID:         "user-456",
		Provider:       "github",
		ProviderUserID: "12345678",
		Email:          &email,
		IdentityData: map[string]interface{}{
			"id":         12345678,
			"login":      "githubuser",
			"name":       "GitHub User",
			"email":      "user@github.com",
			"avatar_url": "https://avatars.githubusercontent.com/u/12345678?v=4",
			"company":    "Acme Corp",
			"location":   "San Francisco, CA",
		},
	}

	assert.Equal(t, "github", identity.Provider)
	assert.Equal(t, "githubuser", identity.IdentityData["login"])
}

func TestUserIdentity_MicrosoftProvider(t *testing.T) {
	email := "user@outlook.com"
	identity := UserIdentity{
		ID:             "identity-microsoft-123",
		UserID:         "user-456",
		Provider:       "microsoft",
		ProviderUserID: "abc123-def456-ghi789",
		Email:          &email,
		IdentityData: map[string]interface{}{
			"id":                "abc123-def456-ghi789",
			"displayName":       "Microsoft User",
			"mail":              "user@outlook.com",
			"userPrincipalName": "user@outlook.com",
		},
	}

	assert.Equal(t, "microsoft", identity.Provider)
	assert.Equal(t, "Microsoft User", identity.IdentityData["displayName"])
}

// =============================================================================
// Benchmark Tests
// =============================================================================

func BenchmarkUserIdentity_Creation(b *testing.B) {
	email := "user@example.com"

	for i := 0; i < b.N; i++ {
		_ = UserIdentity{
			ID:             "identity-123",
			UserID:         "user-456",
			Provider:       "google",
			ProviderUserID: "google-user-789",
			Email:          &email,
			IdentityData: map[string]interface{}{
				"name":    "Test User",
				"picture": "https://example.com/avatar.jpg",
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
	}
}

func BenchmarkNewIdentityService(b *testing.B) {
	oauthManager := NewOAuthManager()
	stateStore := NewStateStore()
	repo := NewIdentityRepository(nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewIdentityService(repo, oauthManager, stateStore)
	}
}

func BenchmarkNewIdentityRepository(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = NewIdentityRepository(nil)
	}
}
