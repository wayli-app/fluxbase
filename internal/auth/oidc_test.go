package auth

import (
	"testing"
)

func TestNormalizeIssuerURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "base URL without trailing slash",
			input:    "https://auth.domain.com",
			expected: "https://auth.domain.com/.well-known/openid-configuration",
		},
		{
			name:     "base URL with trailing slash",
			input:    "https://auth.domain.com/",
			expected: "https://auth.domain.com/.well-known/openid-configuration",
		},
		{
			name:     "base URL with path",
			input:    "https://example.com/auth",
			expected: "https://example.com/auth/.well-known/openid-configuration",
		},
		{
			name:     "base URL with path and trailing slash",
			input:    "https://example.com/auth/",
			expected: "https://example.com/auth/.well-known/openid-configuration",
		},
		{
			name:     "already contains .well-known endpoint",
			input:    "https://auth.domain.com/.well-known/openid-configuration",
			expected: "https://auth.domain.com/.well-known/openid-configuration",
		},
		{
			name:     "custom .well-known path",
			input:    "https://auth.domain.com/.well-known/custom-oidc",
			expected: "https://auth.domain.com/.well-known/custom-oidc",
		},
		{
			name:     "Keycloak-style URL",
			input:    "https://keycloak.example.com/realms/myrealm",
			expected: "https://keycloak.example.com/realms/myrealm/.well-known/openid-configuration",
		},
		{
			name:     "Auth0-style URL",
			input:    "https://tenant.auth0.com",
			expected: "https://tenant.auth0.com/.well-known/openid-configuration",
		},
		{
			name:     "localhost for development",
			input:    "http://localhost:8080",
			expected: "http://localhost:8080/.well-known/openid-configuration",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeIssuerURL(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeIssuerURL(%q) = %q; want %q", tt.input, result, tt.expected)
			}
		})
	}
}
