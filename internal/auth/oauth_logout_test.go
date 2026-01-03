package auth

import (
	"testing"
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
		name                 string
		endSessionEndpoint   string
		idToken              string
		postLogoutRedirectURI string
		state                string
		wantErr              bool
		wantContains         []string
	}{
		{
			name:               "empty endpoint returns error",
			endSessionEndpoint: "",
			wantErr:            true,
		},
		{
			name:                 "basic URL generation",
			endSessionEndpoint:   "https://accounts.google.com/o/oauth2/logout",
			idToken:              "test-id-token",
			postLogoutRedirectURI: "https://example.com/logged-out",
			state:                "test-state",
			wantErr:              false,
			wantContains:         []string{"id_token_hint=test-id-token", "post_logout_redirect_uri=", "state=test-state"},
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
		{ProviderGithub, ""},  // GitHub doesn't support token revocation
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
		{ProviderGithub, ""},  // GitHub doesn't support OIDC logout
		{ProviderApple, ""},   // Apple doesn't support OIDC logout
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
