package auth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateLogoutState(t *testing.T) {
	// Test that state is generated
	state1, err := GenerateLogoutState()
	if err != nil {
		t.Fatalf("GenerateLogoutState() returned error: %v", err)
	}
	if state1 == "" {
		t.Error("GenerateLogoutState() returned empty string")
	}

	// Test that each state is unique
	state2, err := GenerateLogoutState()
	if err != nil {
		t.Fatalf("GenerateLogoutState() returned error: %v", err)
	}
	if state1 == state2 {
		t.Error("GenerateLogoutState() returned same state twice")
	}

	// Test state length (base64 encoded 32 bytes = ~43 chars)
	if len(state1) < 40 {
		t.Errorf("State too short: %d chars", len(state1))
	}
}

func TestGenerateOIDCLogoutURL(t *testing.T) {
	service := &OAuthLogoutService{}

	tests := []struct {
		name                  string
		endSessionEndpoint    string
		idToken               string
		postLogoutRedirectURI string
		state                 string
		wantErr               bool
		wantContains          []string
	}{
		{
			name:               "empty endpoint returns error",
			endSessionEndpoint: "",
			wantErr:            true,
		},
		{
			name:                  "basic URL generation",
			endSessionEndpoint:    "https://accounts.google.com/o/oauth2/logout",
			idToken:               "test-id-token",
			postLogoutRedirectURI: "https://example.com/logged-out",
			state:                 "test-state",
			wantErr:               false,
			wantContains:          []string{"id_token_hint=test-id-token", "post_logout_redirect_uri=", "state=test-state"},
		},
		{
			name:               "URL without id_token",
			endSessionEndpoint: "https://accounts.google.com/o/oauth2/logout",
			idToken:            "",
			state:              "test-state",
			wantErr:            false,
			wantContains:       []string{"state=test-state"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url, err := service.GenerateOIDCLogoutURL(tt.endSessionEndpoint, tt.idToken, tt.postLogoutRedirectURI, tt.state)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			for _, want := range tt.wantContains {
				if !containsString(url, want) {
					t.Errorf("URL %q should contain %q", url, want)
				}
			}
		})
	}
}

func TestGetDefaultRevocationEndpoint(t *testing.T) {
	tests := []struct {
		provider OAuthProvider
		want     string
	}{
		{ProviderGoogle, "https://oauth2.googleapis.com/revoke"},
		{ProviderApple, "https://appleid.apple.com/auth/revoke"},
		{ProviderGitLab, "https://gitlab.com/oauth/revoke"},
		{ProviderTwitter, "https://api.twitter.com/2/oauth2/revoke"},
		{ProviderGithub, ""},   // GitHub doesn't support token revocation
		{ProviderFacebook, ""}, // Facebook uses different mechanism
		{ProviderLinkedIn, ""}, // LinkedIn doesn't support standard revocation
	}

	for _, tt := range tests {
		t.Run(string(tt.provider), func(t *testing.T) {
			got := GetDefaultRevocationEndpoint(tt.provider)
			if got != tt.want {
				t.Errorf("GetDefaultRevocationEndpoint(%q) = %q, want %q", tt.provider, got, tt.want)
			}
		})
	}
}

func TestGetDefaultEndSessionEndpoint(t *testing.T) {
	tests := []struct {
		provider OAuthProvider
		want     string
	}{
		{ProviderGoogle, "https://accounts.google.com/o/oauth2/logout"},
		{ProviderMicrosoft, "https://login.microsoftonline.com/common/oauth2/v2.0/logout"},
		{ProviderGitLab, "https://gitlab.com/oauth/logout"},
		{ProviderGithub, ""},   // GitHub doesn't support OIDC logout
		{ProviderApple, ""},    // Apple doesn't support OIDC logout
		{ProviderFacebook, ""}, // Facebook doesn't support OIDC logout
	}

	for _, tt := range tests {
		t.Run(string(tt.provider), func(t *testing.T) {
			got := GetDefaultEndSessionEndpoint(tt.provider)
			if got != tt.want {
				t.Errorf("GetDefaultEndSessionEndpoint(%q) = %q, want %q", tt.provider, got, tt.want)
			}
		})
	}
}

// Helper function to check if a string contains a substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// =============================================================================
// Error Constants Tests
// =============================================================================

func TestOAuthLogoutErrors(t *testing.T) {
	t.Run("error constants are defined", func(t *testing.T) {
		assert.NotNil(t, ErrOAuthLogoutStateNotFound)
		assert.NotNil(t, ErrOAuthLogoutStateExpired)
		assert.NotNil(t, ErrOAuthTokenNotFound)
		assert.NotNil(t, ErrOAuthProviderNoSLO)
	})

	t.Run("error messages are meaningful", func(t *testing.T) {
		assert.Contains(t, ErrOAuthLogoutStateNotFound.Error(), "not found")
		assert.Contains(t, ErrOAuthLogoutStateExpired.Error(), "expired")
		assert.Contains(t, ErrOAuthTokenNotFound.Error(), "not found")
		assert.Contains(t, ErrOAuthProviderNoSLO.Error(), "does not support")
	})

	t.Run("errors are distinct", func(t *testing.T) {
		errors := []error{
			ErrOAuthLogoutStateNotFound,
			ErrOAuthLogoutStateExpired,
			ErrOAuthTokenNotFound,
			ErrOAuthProviderNoSLO,
		}

		for i, err1 := range errors {
			for j, err2 := range errors {
				if i != j {
					assert.NotEqual(t, err1, err2)
				}
			}
		}
	})
}

// =============================================================================
// OAuthLogoutState Struct Tests
// =============================================================================

func TestOAuthLogoutState_Struct(t *testing.T) {
	t.Run("stores all fields", func(t *testing.T) {
		now := time.Now()
		state := OAuthLogoutState{
			ID:                    "state-123",
			UserID:                "user-456",
			Provider:              "google",
			State:                 "random-state-value",
			PostLogoutRedirectURI: "https://app.example.com/logged-out",
			CreatedAt:             now,
			ExpiresAt:             now.Add(10 * time.Minute),
		}

		assert.Equal(t, "state-123", state.ID)
		assert.Equal(t, "user-456", state.UserID)
		assert.Equal(t, "google", state.Provider)
		assert.Equal(t, "random-state-value", state.State)
		assert.Equal(t, "https://app.example.com/logged-out", state.PostLogoutRedirectURI)
		assert.Equal(t, now, state.CreatedAt)
	})

	t.Run("defaults to zero values", func(t *testing.T) {
		state := OAuthLogoutState{}

		assert.Empty(t, state.ID)
		assert.Empty(t, state.UserID)
		assert.Empty(t, state.Provider)
		assert.Empty(t, state.State)
		assert.Empty(t, state.PostLogoutRedirectURI)
	})

	t.Run("expiry check", func(t *testing.T) {
		// Not expired
		state := OAuthLogoutState{
			ExpiresAt: time.Now().Add(time.Hour),
		}
		assert.False(t, time.Now().After(state.ExpiresAt))

		// Expired
		expiredState := OAuthLogoutState{
			ExpiresAt: time.Now().Add(-time.Hour),
		}
		assert.True(t, time.Now().After(expiredState.ExpiresAt))
	})
}

// =============================================================================
// StoredOAuthToken Struct Tests
// =============================================================================

func TestStoredOAuthToken_Struct(t *testing.T) {
	t.Run("stores all fields", func(t *testing.T) {
		now := time.Now()
		token := StoredOAuthToken{
			ID:           "token-123",
			UserID:       "user-456",
			Provider:     "google",
			AccessToken:  "access-token-abc",
			RefreshToken: "refresh-token-xyz",
			IDToken:      "id-token-123",
			TokenExpiry:  now.Add(time.Hour),
			CreatedAt:    now,
			UpdatedAt:    now,
		}

		assert.Equal(t, "token-123", token.ID)
		assert.Equal(t, "user-456", token.UserID)
		assert.Equal(t, "google", token.Provider)
		assert.Equal(t, "access-token-abc", token.AccessToken)
		assert.Equal(t, "refresh-token-xyz", token.RefreshToken)
		assert.Equal(t, "id-token-123", token.IDToken)
	})

	t.Run("defaults to zero values", func(t *testing.T) {
		token := StoredOAuthToken{}

		assert.Empty(t, token.ID)
		assert.Empty(t, token.UserID)
		assert.Empty(t, token.Provider)
		assert.Empty(t, token.AccessToken)
		assert.Empty(t, token.RefreshToken)
		assert.Empty(t, token.IDToken)
	})

	t.Run("token without refresh or id token", func(t *testing.T) {
		token := StoredOAuthToken{
			ID:          "token-123",
			UserID:      "user-456",
			Provider:    "github",
			AccessToken: "access-token-only",
		}

		assert.NotEmpty(t, token.AccessToken)
		assert.Empty(t, token.RefreshToken)
		assert.Empty(t, token.IDToken)
	})
}

// =============================================================================
// OAuthLogoutResult Struct Tests
// =============================================================================

func TestOAuthLogoutResult_Struct(t *testing.T) {
	t.Run("complete logout result", func(t *testing.T) {
		result := OAuthLogoutResult{
			LocalLogoutComplete:  true,
			ProviderTokenRevoked: true,
			RequiresRedirect:     false,
			RedirectURL:          "",
			Provider:             "google",
			Warning:              "",
		}

		assert.True(t, result.LocalLogoutComplete)
		assert.True(t, result.ProviderTokenRevoked)
		assert.False(t, result.RequiresRedirect)
		assert.Equal(t, "google", result.Provider)
	})

	t.Run("logout with redirect required", func(t *testing.T) {
		result := OAuthLogoutResult{
			LocalLogoutComplete:  true,
			ProviderTokenRevoked: false,
			RequiresRedirect:     true,
			RedirectURL:          "https://accounts.google.com/logout",
			Provider:             "google",
		}

		assert.True(t, result.RequiresRedirect)
		assert.NotEmpty(t, result.RedirectURL)
	})

	t.Run("logout with warning", func(t *testing.T) {
		result := OAuthLogoutResult{
			LocalLogoutComplete:  true,
			ProviderTokenRevoked: false,
			Provider:             "github",
			Warning:              "Provider does not support token revocation",
		}

		assert.NotEmpty(t, result.Warning)
		assert.False(t, result.ProviderTokenRevoked)
	})

	t.Run("defaults to zero values", func(t *testing.T) {
		result := OAuthLogoutResult{}

		assert.False(t, result.LocalLogoutComplete)
		assert.False(t, result.ProviderTokenRevoked)
		assert.False(t, result.RequiresRedirect)
		assert.Empty(t, result.RedirectURL)
		assert.Empty(t, result.Provider)
		assert.Empty(t, result.Warning)
	})
}

// =============================================================================
// NewOAuthLogoutService Tests
// =============================================================================

func TestNewOAuthLogoutService(t *testing.T) {
	t.Run("creates service with nil database", func(t *testing.T) {
		svc := NewOAuthLogoutService(nil, []byte(""))
		assert.NotNil(t, svc)
		assert.Nil(t, svc.db)
		assert.Empty(t, svc.encryptionKey)
		assert.NotNil(t, svc.httpClient)
	})

	t.Run("creates service with encryption key", func(t *testing.T) {
		svc := NewOAuthLogoutService(nil, []byte("my-secret-key"))
		assert.NotNil(t, svc)
		assert.Equal(t, []byte("my-secret-key"), svc.encryptionKey)
	})

	t.Run("http client has timeout", func(t *testing.T) {
		svc := NewOAuthLogoutService(nil, []byte(""))
		assert.Equal(t, 10*time.Second, svc.httpClient.Timeout)
	})
}

// =============================================================================
// GenerateLogoutState Extended Tests
// =============================================================================

func TestGenerateLogoutState_Extended(t *testing.T) {
	t.Run("generates unique states", func(t *testing.T) {
		states := make(map[string]bool)
		for i := 0; i < 100; i++ {
			state, err := GenerateLogoutState()
			require.NoError(t, err)
			assert.False(t, states[state], "state should be unique")
			states[state] = true
		}
		assert.Len(t, states, 100)
	})

	t.Run("state is URL safe", func(t *testing.T) {
		state, err := GenerateLogoutState()
		require.NoError(t, err)

		// Base64 URL encoding should not contain + or /
		assert.NotContains(t, state, "+")
		assert.NotContains(t, state, "/")
	})
}

// =============================================================================
// GetDefaultRevocationEndpoint Extended Tests
// =============================================================================

func TestGetDefaultRevocationEndpoint_AllProviders(t *testing.T) {
	tests := []struct {
		provider    OAuthProvider
		wantEmpty   bool
		wantContain string
	}{
		{ProviderGoogle, false, "googleapis.com"},
		{ProviderMicrosoft, true, ""},
		{ProviderApple, false, "appleid.apple.com"},
		{ProviderGitLab, false, "gitlab.com"},
		{ProviderGithub, true, ""},
		{ProviderFacebook, true, ""},
		{ProviderTwitter, false, "twitter.com"},
		{ProviderLinkedIn, true, ""},
		{ProviderBitbucket, true, ""},
		{OAuthProvider("unknown"), true, ""},
	}

	for _, tt := range tests {
		t.Run(string(tt.provider), func(t *testing.T) {
			endpoint := GetDefaultRevocationEndpoint(tt.provider)
			if tt.wantEmpty {
				assert.Empty(t, endpoint)
			} else {
				assert.NotEmpty(t, endpoint)
				assert.Contains(t, endpoint, tt.wantContain)
			}
		})
	}
}

// =============================================================================
// GetDefaultEndSessionEndpoint Extended Tests
// =============================================================================

func TestGetDefaultEndSessionEndpoint_AllProviders(t *testing.T) {
	tests := []struct {
		provider    OAuthProvider
		wantEmpty   bool
		wantContain string
	}{
		{ProviderGoogle, false, "accounts.google.com"},
		{ProviderMicrosoft, false, "microsoftonline.com"},
		{ProviderApple, true, ""},
		{ProviderGitLab, false, "gitlab.com"},
		{ProviderGithub, true, ""},
		{ProviderFacebook, true, ""},
		{ProviderTwitter, true, ""},
		{ProviderLinkedIn, true, ""},
		{ProviderBitbucket, true, ""},
		{OAuthProvider("unknown"), true, ""},
	}

	for _, tt := range tests {
		t.Run(string(tt.provider), func(t *testing.T) {
			endpoint := GetDefaultEndSessionEndpoint(tt.provider)
			if tt.wantEmpty {
				assert.Empty(t, endpoint)
			} else {
				assert.NotEmpty(t, endpoint)
				assert.Contains(t, endpoint, tt.wantContain)
			}
		})
	}
}

// =============================================================================
// GenerateOIDCLogoutURL Extended Tests
// =============================================================================

func TestGenerateOIDCLogoutURL_Extended(t *testing.T) {
	service := &OAuthLogoutService{}

	t.Run("URL with all parameters", func(t *testing.T) {
		url, err := service.GenerateOIDCLogoutURL(
			"https://auth.example.com/logout",
			"id-token-123",
			"https://app.example.com/logged-out",
			"state-abc",
		)

		require.NoError(t, err)
		assert.Contains(t, url, "id_token_hint=id-token-123")
		assert.Contains(t, url, "post_logout_redirect_uri=")
		assert.Contains(t, url, "state=state-abc")
	})

	t.Run("URL with only state", func(t *testing.T) {
		url, err := service.GenerateOIDCLogoutURL(
			"https://auth.example.com/logout",
			"",
			"",
			"state-only",
		)

		require.NoError(t, err)
		assert.Contains(t, url, "state=state-only")
		assert.NotContains(t, url, "id_token_hint")
	})

	t.Run("URL with no optional params", func(t *testing.T) {
		url, err := service.GenerateOIDCLogoutURL(
			"https://auth.example.com/logout",
			"",
			"",
			"",
		)

		require.NoError(t, err)
		assert.Equal(t, "https://auth.example.com/logout", url)
	})

	t.Run("invalid endpoint URL", func(t *testing.T) {
		_, err := service.GenerateOIDCLogoutURL(
			"://invalid-url",
			"",
			"",
			"",
		)

		assert.Error(t, err)
	})
}

// =============================================================================
// Benchmark Tests
// =============================================================================

func BenchmarkGenerateLogoutState(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = GenerateLogoutState()
	}
}

func BenchmarkGenerateOIDCLogoutURL(b *testing.B) {
	service := &OAuthLogoutService{}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = service.GenerateOIDCLogoutURL(
			"https://auth.example.com/logout",
			"id-token-123",
			"https://app.example.com/logged-out",
			"state-abc",
		)
	}
}

func BenchmarkGetDefaultRevocationEndpoint(b *testing.B) {
	providers := []OAuthProvider{
		ProviderGoogle,
		ProviderMicrosoft,
		ProviderApple,
		ProviderGitLab,
		ProviderGithub,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GetDefaultRevocationEndpoint(providers[i%len(providers)])
	}
}
