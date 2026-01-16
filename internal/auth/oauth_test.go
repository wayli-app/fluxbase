package auth

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// GenerateState Tests
// =============================================================================

func TestGenerateState_NotEmpty(t *testing.T) {
	state, err := GenerateState()

	require.NoError(t, err)
	assert.NotEmpty(t, state)
}

func TestGenerateState_Uniqueness(t *testing.T) {
	states := make(map[string]bool)

	for i := 0; i < 100; i++ {
		state, err := GenerateState()
		require.NoError(t, err)

		assert.False(t, states[state], "State collision detected")
		states[state] = true
	}

	assert.Len(t, states, 100)
}

func TestGenerateState_Base64URLEncoded(t *testing.T) {
	state, err := GenerateState()
	require.NoError(t, err)

	// Base64 URL safe characters: A-Z, a-z, 0-9, -, _, =
	for _, c := range state {
		isAlphaNum := (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9')
		isUrlSafe := c == '-' || c == '_' || c == '='
		assert.True(t, isAlphaNum || isUrlSafe, "Invalid character in state: %c", c)
	}
}

func TestGenerateState_Length(t *testing.T) {
	state, err := GenerateState()
	require.NoError(t, err)

	// 32 bytes base64 encoded should be ~44 characters
	assert.GreaterOrEqual(t, len(state), 40)
	assert.LessOrEqual(t, len(state), 48)
}

// =============================================================================
// OAuthProvider Constants Tests
// =============================================================================

func TestOAuthProvider_Constants(t *testing.T) {
	providers := []struct {
		provider OAuthProvider
		expected string
	}{
		{ProviderGoogle, "google"},
		{ProviderGithub, "github"},
		{ProviderMicrosoft, "microsoft"},
		{ProviderApple, "apple"},
		{ProviderFacebook, "facebook"},
		{ProviderTwitter, "twitter"},
		{ProviderLinkedIn, "linkedin"},
		{ProviderGitLab, "gitlab"},
		{ProviderBitbucket, "bitbucket"},
	}

	for _, p := range providers {
		t.Run(string(p.provider), func(t *testing.T) {
			assert.Equal(t, OAuthProvider(p.expected), p.provider)
		})
	}
}

func TestOAuthProvider_Uniqueness(t *testing.T) {
	providers := []OAuthProvider{
		ProviderGoogle,
		ProviderGithub,
		ProviderMicrosoft,
		ProviderApple,
		ProviderFacebook,
		ProviderTwitter,
		ProviderLinkedIn,
		ProviderGitLab,
		ProviderBitbucket,
	}

	seen := make(map[OAuthProvider]bool)
	for _, p := range providers {
		assert.False(t, seen[p], "Duplicate provider: %s", p)
		seen[p] = true
	}
}

// =============================================================================
// Error Variable Tests
// =============================================================================

func TestOAuthErrors_Defined(t *testing.T) {
	assert.NotNil(t, ErrInvalidProvider)
	assert.NotNil(t, ErrInvalidState)
}

func TestOAuthErrors_Messages(t *testing.T) {
	assert.Equal(t, "invalid OAuth provider", ErrInvalidProvider.Error())
	assert.Equal(t, "invalid OAuth state", ErrInvalidState.Error())
}

func TestOAuthErrors_Distinct(t *testing.T) {
	assert.NotEqual(t, ErrInvalidProvider, ErrInvalidState)
}

// =============================================================================
// OAuthConfig Struct Tests
// =============================================================================

func TestOAuthConfig_Fields(t *testing.T) {
	config := OAuthConfig{
		ClientID:     "client-id-123",
		ClientSecret: "client-secret-456",
		RedirectURL:  "https://example.com/callback",
		Scopes:       []string{"email", "profile"},
	}

	assert.Equal(t, "client-id-123", config.ClientID)
	assert.Equal(t, "client-secret-456", config.ClientSecret)
	assert.Equal(t, "https://example.com/callback", config.RedirectURL)
	assert.Equal(t, []string{"email", "profile"}, config.Scopes)
}

func TestOAuthConfig_EmptyScopes(t *testing.T) {
	config := OAuthConfig{
		ClientID:     "client-id",
		ClientSecret: "client-secret",
		RedirectURL:  "https://example.com/callback",
		Scopes:       nil,
	}

	assert.Nil(t, config.Scopes)
}

// =============================================================================
// OAuthManager Tests
// =============================================================================

func TestNewOAuthManager(t *testing.T) {
	manager := NewOAuthManager()

	require.NotNil(t, manager)
	assert.NotNil(t, manager.configs)
	assert.Empty(t, manager.configs)
}

func TestOAuthManager_RegisterProvider(t *testing.T) {
	manager := NewOAuthManager()

	config := OAuthConfig{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		RedirectURL:  "https://example.com/callback",
		Scopes:       []string{"email", "profile"},
	}

	err := manager.RegisterProvider(ProviderGoogle, config)
	require.NoError(t, err)

	assert.Len(t, manager.configs, 1)
	assert.NotNil(t, manager.configs[ProviderGoogle])
}

func TestOAuthManager_RegisterMultipleProviders(t *testing.T) {
	manager := NewOAuthManager()

	providers := []OAuthProvider{
		ProviderGoogle,
		ProviderGithub,
		ProviderMicrosoft,
	}

	for _, p := range providers {
		config := OAuthConfig{
			ClientID:     string(p) + "-client-id",
			ClientSecret: string(p) + "-client-secret",
			RedirectURL:  "https://example.com/callback/" + string(p),
			Scopes:       []string{"email"},
		}

		err := manager.RegisterProvider(p, config)
		require.NoError(t, err)
	}

	assert.Len(t, manager.configs, 3)
}

// =============================================================================
// OAuthManager GetEndpoint Tests
// =============================================================================

func TestOAuthManager_GetEndpoint(t *testing.T) {
	manager := NewOAuthManager()

	tests := []struct {
		provider      OAuthProvider
		expectAuthURL bool
	}{
		{ProviderGoogle, true},
		{ProviderGithub, true},
		{ProviderMicrosoft, true},
		{ProviderApple, true},
		{ProviderFacebook, true},
		{ProviderTwitter, true},
		{ProviderLinkedIn, true},
		{ProviderGitLab, true},
		{ProviderBitbucket, true},
		{OAuthProvider("invalid"), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.provider), func(t *testing.T) {
			endpoint := manager.GetEndpoint(tt.provider)

			if tt.expectAuthURL {
				assert.NotEmpty(t, endpoint.AuthURL)
				assert.NotEmpty(t, endpoint.TokenURL)
			} else {
				assert.Empty(t, endpoint.AuthURL)
				assert.Empty(t, endpoint.TokenURL)
			}
		})
	}
}

func TestOAuthManager_GetEndpoint_URLs(t *testing.T) {
	manager := NewOAuthManager()

	tests := []struct {
		provider OAuthProvider
		authURL  string
		tokenURL string
	}{
		{
			ProviderMicrosoft,
			"https://login.microsoftonline.com/common/oauth2/v2.0/authorize",
			"https://login.microsoftonline.com/common/oauth2/v2.0/token",
		},
		{
			ProviderApple,
			"https://appleid.apple.com/auth/authorize",
			"https://appleid.apple.com/auth/token",
		},
		{
			ProviderFacebook,
			"https://www.facebook.com/v12.0/dialog/oauth",
			"https://graph.facebook.com/v12.0/oauth/access_token",
		},
		{
			ProviderTwitter,
			"https://twitter.com/i/oauth2/authorize",
			"https://api.twitter.com/2/oauth2/token",
		},
		{
			ProviderLinkedIn,
			"https://www.linkedin.com/oauth/v2/authorization",
			"https://www.linkedin.com/oauth/v2/accessToken",
		},
		{
			ProviderGitLab,
			"https://gitlab.com/oauth/authorize",
			"https://gitlab.com/oauth/token",
		},
		{
			ProviderBitbucket,
			"https://bitbucket.org/site/oauth2/authorize",
			"https://bitbucket.org/site/oauth2/access_token",
		},
	}

	for _, tt := range tests {
		t.Run(string(tt.provider), func(t *testing.T) {
			endpoint := manager.GetEndpoint(tt.provider)

			assert.Equal(t, tt.authURL, endpoint.AuthURL)
			assert.Equal(t, tt.tokenURL, endpoint.TokenURL)
		})
	}
}

// =============================================================================
// OAuthManager GetUserInfoURL Tests
// =============================================================================

func TestOAuthManager_GetUserInfoURL(t *testing.T) {
	manager := NewOAuthManager()

	tests := []struct {
		provider OAuthProvider
		expected string
	}{
		{ProviderGoogle, "https://www.googleapis.com/oauth2/v2/userinfo"},
		{ProviderGithub, "https://api.github.com/user"},
		{ProviderMicrosoft, "https://graph.microsoft.com/v1.0/me"},
		{ProviderApple, "https://appleid.apple.com/auth/keys"},
		{ProviderFacebook, "https://graph.facebook.com/me?fields=id,name,email,picture"},
		{ProviderTwitter, "https://api.twitter.com/2/users/me"},
		{ProviderLinkedIn, "https://api.linkedin.com/v2/me"},
		{ProviderGitLab, "https://gitlab.com/api/v4/user"},
		{ProviderBitbucket, "https://api.bitbucket.org/2.0/user"},
		{OAuthProvider("invalid"), ""},
	}

	for _, tt := range tests {
		t.Run(string(tt.provider), func(t *testing.T) {
			url := manager.GetUserInfoURL(tt.provider)
			assert.Equal(t, tt.expected, url)
		})
	}
}

// =============================================================================
// OAuthManager GetAuthURL Tests
// =============================================================================

func TestOAuthManager_GetAuthURL_UnregisteredProvider(t *testing.T) {
	manager := NewOAuthManager()

	url, err := manager.GetAuthURL(ProviderGoogle, "test-state")

	assert.Error(t, err)
	assert.Equal(t, ErrInvalidProvider, err)
	assert.Empty(t, url)
}

func TestOAuthManager_GetAuthURL_RegisteredProvider(t *testing.T) {
	manager := NewOAuthManager()

	config := OAuthConfig{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		RedirectURL:  "https://example.com/callback",
		Scopes:       []string{"email", "profile"},
	}

	err := manager.RegisterProvider(ProviderGoogle, config)
	require.NoError(t, err)

	url, err := manager.GetAuthURL(ProviderGoogle, "test-state")

	require.NoError(t, err)
	assert.NotEmpty(t, url)
	assert.Contains(t, url, "client_id=test-client-id")
	assert.Contains(t, url, "state=test-state")
	assert.Contains(t, url, "redirect_uri=")
}

// =============================================================================
// StateStore Tests
// =============================================================================

func TestNewStateStore(t *testing.T) {
	store := NewStateStore()

	require.NotNil(t, store)
	assert.NotNil(t, store.states)
	assert.Empty(t, store.states)
}

func TestStateStore_Set(t *testing.T) {
	store := NewStateStore()

	store.Set("test-state")

	store.mu.RLock()
	metadata, exists := store.states["test-state"]
	store.mu.RUnlock()

	assert.True(t, exists)
	assert.NotNil(t, metadata)
	assert.True(t, metadata.Expiry.After(time.Now()))
	assert.Empty(t, metadata.RedirectURI)
}

func TestStateStore_Set_WithRedirectURI(t *testing.T) {
	store := NewStateStore()

	store.Set("test-state", "https://example.com/custom-redirect")

	store.mu.RLock()
	metadata, exists := store.states["test-state"]
	store.mu.RUnlock()

	assert.True(t, exists)
	assert.NotNil(t, metadata)
	assert.Equal(t, "https://example.com/custom-redirect", metadata.RedirectURI)
}

func TestStateStore_Set_EmptyRedirectURI(t *testing.T) {
	store := NewStateStore()

	store.Set("test-state", "")

	store.mu.RLock()
	metadata, exists := store.states["test-state"]
	store.mu.RUnlock()

	assert.True(t, exists)
	assert.Empty(t, metadata.RedirectURI)
}

func TestStateStore_Validate_ValidState(t *testing.T) {
	store := NewStateStore()
	store.Set("test-state")

	valid := store.Validate("test-state")

	assert.True(t, valid)

	// State should be removed after validation
	store.mu.RLock()
	_, exists := store.states["test-state"]
	store.mu.RUnlock()

	assert.False(t, exists)
}

func TestStateStore_Validate_InvalidState(t *testing.T) {
	store := NewStateStore()

	valid := store.Validate("non-existent-state")

	assert.False(t, valid)
}

func TestStateStore_Validate_ExpiredState(t *testing.T) {
	store := NewStateStore()

	// Manually add an expired state
	store.mu.Lock()
	store.states["expired-state"] = &StateMetadata{
		Expiry: time.Now().Add(-1 * time.Minute), // Expired
	}
	store.mu.Unlock()

	valid := store.Validate("expired-state")

	assert.False(t, valid)

	// State should still be removed
	store.mu.RLock()
	_, exists := store.states["expired-state"]
	store.mu.RUnlock()

	assert.False(t, exists)
}

func TestStateStore_GetAndValidate_ValidState(t *testing.T) {
	store := NewStateStore()
	store.Set("test-state", "https://example.com/redirect")

	metadata, valid := store.GetAndValidate("test-state")

	assert.True(t, valid)
	assert.NotNil(t, metadata)
	assert.Equal(t, "https://example.com/redirect", metadata.RedirectURI)

	// State should be removed
	store.mu.RLock()
	_, exists := store.states["test-state"]
	store.mu.RUnlock()

	assert.False(t, exists)
}

func TestStateStore_GetAndValidate_InvalidState(t *testing.T) {
	store := NewStateStore()

	metadata, valid := store.GetAndValidate("non-existent-state")

	assert.False(t, valid)
	assert.Nil(t, metadata)
}

func TestStateStore_GetAndValidate_ExpiredState(t *testing.T) {
	store := NewStateStore()

	// Manually add an expired state
	store.mu.Lock()
	store.states["expired-state"] = &StateMetadata{
		Expiry:      time.Now().Add(-1 * time.Minute),
		RedirectURI: "https://example.com/redirect",
	}
	store.mu.Unlock()

	metadata, valid := store.GetAndValidate("expired-state")

	assert.False(t, valid)
	assert.Nil(t, metadata)
}

func TestStateStore_Cleanup(t *testing.T) {
	store := NewStateStore()

	// Add expired and valid states
	store.mu.Lock()
	store.states["expired-1"] = &StateMetadata{Expiry: time.Now().Add(-1 * time.Minute)}
	store.states["expired-2"] = &StateMetadata{Expiry: time.Now().Add(-1 * time.Hour)}
	store.states["valid-1"] = &StateMetadata{Expiry: time.Now().Add(5 * time.Minute)}
	store.states["valid-2"] = &StateMetadata{Expiry: time.Now().Add(10 * time.Minute)}
	store.mu.Unlock()

	store.Cleanup()

	store.mu.RLock()
	_, expired1Exists := store.states["expired-1"]
	_, expired2Exists := store.states["expired-2"]
	_, valid1Exists := store.states["valid-1"]
	_, valid2Exists := store.states["valid-2"]
	store.mu.RUnlock()

	assert.False(t, expired1Exists)
	assert.False(t, expired2Exists)
	assert.True(t, valid1Exists)
	assert.True(t, valid2Exists)
}

func TestStateStore_Cleanup_AllExpired(t *testing.T) {
	store := NewStateStore()

	store.mu.Lock()
	for i := 0; i < 10; i++ {
		store.states["expired-"+string(rune('0'+i))] = &StateMetadata{
			Expiry: time.Now().Add(-1 * time.Minute),
		}
	}
	store.mu.Unlock()

	store.Cleanup()

	store.mu.RLock()
	assert.Empty(t, store.states)
	store.mu.RUnlock()
}

func TestStateStore_Cleanup_AllValid(t *testing.T) {
	store := NewStateStore()

	store.mu.Lock()
	for i := 0; i < 10; i++ {
		store.states["valid-"+string(rune('0'+i))] = &StateMetadata{
			Expiry: time.Now().Add(5 * time.Minute),
		}
	}
	store.mu.Unlock()

	store.Cleanup()

	store.mu.RLock()
	assert.Len(t, store.states, 10)
	store.mu.RUnlock()
}

// =============================================================================
// StateMetadata Tests
// =============================================================================

func TestStateMetadata_Fields(t *testing.T) {
	expiry := time.Now().Add(10 * time.Minute)
	metadata := StateMetadata{
		Expiry:      expiry,
		RedirectURI: "https://example.com/redirect",
	}

	assert.Equal(t, expiry, metadata.Expiry)
	assert.Equal(t, "https://example.com/redirect", metadata.RedirectURI)
}

func TestStateMetadata_EmptyRedirectURI(t *testing.T) {
	metadata := StateMetadata{
		Expiry: time.Now().Add(10 * time.Minute),
	}

	assert.Empty(t, metadata.RedirectURI)
}

// =============================================================================
// Concurrent Access Tests
// =============================================================================

func TestStateStore_ConcurrentSetValidate(t *testing.T) {
	store := NewStateStore()
	var wg sync.WaitGroup

	// Concurrent setters
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				state := "state-" + string(rune('0'+id)) + "-" + string(rune('0'+j%10))
				store.Set(state)
			}
		}(i)
	}

	// Concurrent validators
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				state := "state-" + string(rune('0'+id)) + "-" + string(rune('0'+j%10))
				store.Validate(state)
			}
		}(i)
	}

	wg.Wait()
	// No race conditions means success
}

func TestStateStore_ConcurrentCleanup(t *testing.T) {
	store := NewStateStore()
	var wg sync.WaitGroup

	// Writer goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			store.Set("state-" + string(rune('0'+i%10)))
		}
	}()

	// Cleanup goroutines
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				store.Cleanup()
				time.Sleep(time.Millisecond)
			}
		}()
	}

	wg.Wait()
	// No race conditions means success
}

// =============================================================================
// Benchmark Tests
// =============================================================================

func BenchmarkGenerateState(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = GenerateState()
	}
}

func BenchmarkStateStore_Set(b *testing.B) {
	store := NewStateStore()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.Set("state-" + string(rune(i%256)))
	}
}

func BenchmarkStateStore_Validate(b *testing.B) {
	store := NewStateStore()

	// Pre-populate
	for i := 0; i < 1000; i++ {
		store.Set("state-" + string(rune(i)))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.Validate("state-" + string(rune(i%1000)))
	}
}

func BenchmarkStateStore_Cleanup(b *testing.B) {
	store := NewStateStore()

	for i := 0; i < b.N; i++ {
		// Pre-populate with mix of expired and valid
		store.mu.Lock()
		for j := 0; j < 100; j++ {
			if j%2 == 0 {
				store.states["state-"+string(rune(j))] = &StateMetadata{
					Expiry: time.Now().Add(-1 * time.Minute),
				}
			} else {
				store.states["state-"+string(rune(j))] = &StateMetadata{
					Expiry: time.Now().Add(5 * time.Minute),
				}
			}
		}
		store.mu.Unlock()

		store.Cleanup()
	}
}

func BenchmarkOAuthManager_GetEndpoint(b *testing.B) {
	manager := NewOAuthManager()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		manager.GetEndpoint(ProviderGoogle)
	}
}

func BenchmarkOAuthManager_GetUserInfoURL(b *testing.B) {
	manager := NewOAuthManager()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		manager.GetUserInfoURL(ProviderGithub)
	}
}
