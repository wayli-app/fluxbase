package api

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nimbleflux/fluxbase/internal/config"
)

func TestNormalizeRedirectURLs(t *testing.T) {
	t.Run("valid list is trimmed and deduplicated", func(t *testing.T) {
		urls, err := normalizeRedirectURLs([]string{
			"  https://app.example.com/api/v1/auth/oauth/google/callback  ",
			"https://custom.example.com/api/v1/auth/oauth/google/callback",
			"https://app.example.com/api/v1/auth/oauth/google/callback",
		})
		require.NoError(t, err)
		assert.Equal(t, []string{
			"https://app.example.com/api/v1/auth/oauth/google/callback",
			"https://custom.example.com/api/v1/auth/oauth/google/callback",
		}, urls)
	})

	t.Run("single legacy URL", func(t *testing.T) {
		urls, err := normalizeRedirectURLs([]string{"http://localhost:3000/callback"})
		require.NoError(t, err)
		assert.Equal(t, []string{"http://localhost:3000/callback"}, urls)
	})

	t.Run("empty list is allowed (checked by callers)", func(t *testing.T) {
		urls, err := normalizeRedirectURLs(nil)
		require.NoError(t, err)
		assert.Empty(t, urls)
	})

	t.Run("rejects empty entries", func(t *testing.T) {
		_, err := normalizeRedirectURLs([]string{"https://app.example.com/cb", "  "})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "empty")
	})

	t.Run("rejects relative URLs", func(t *testing.T) {
		_, err := normalizeRedirectURLs([]string{"/api/v1/auth/oauth/google/callback"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "absolute")
	})

	t.Run("rejects scheme-less URLs", func(t *testing.T) {
		_, err := normalizeRedirectURLs([]string{"app.example.com/callback"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "absolute")
	})

	t.Run("rejects http/https URLs without a host", func(t *testing.T) {
		_, err := normalizeRedirectURLs([]string{"https:/callback"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "host")
	})

	t.Run("allows private-use app schemes", func(t *testing.T) {
		urls, err := normalizeRedirectURLs([]string{"wayli://oauth/callback"})
		require.NoError(t, err)
		assert.Equal(t, []string{"wayli://oauth/callback"}, urls)
	})

	t.Run("allows private-use schemes without an authority", func(t *testing.T) {
		urls, err := normalizeRedirectURLs([]string{"com.example.app:/oauth2redirect"})
		require.NoError(t, err)
		assert.Equal(t, []string{"com.example.app:/oauth2redirect"}, urls)
	})

	t.Run("rejects scheme-only app URIs", func(t *testing.T) {
		_, err := normalizeRedirectURLs([]string{"wayli:"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "host or path")
	})

	t.Run("rejects dangerous and non-redirect schemes", func(t *testing.T) {
		for _, u := range []string{
			"javascript:alert(1)",
			"data:text/html;base64,PHNjcmlwdD4=",
			"ftp://app.example.com/callback",
			"mailto:admin@example.com",
		} {
			_, err := normalizeRedirectURLs([]string{u})
			require.Error(t, err, "expected %q to be rejected", u)
			assert.Contains(t, err.Error(), "not allowed")
		}
	})

	t.Run("rejects malformed URLs", func(t *testing.T) {
		_, err := normalizeRedirectURLs([]string{"https://[::invalid"})
		require.Error(t, err)
	})

	t.Run("rejects lists over the cap", func(t *testing.T) {
		urls := make([]string, maxRedirectURLs+1)
		for i := range urls {
			urls[i] = fmt.Sprintf("https://app%d.example.com/callback", i)
		}
		_, err := normalizeRedirectURLs(urls)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "too many redirect URLs")
	})
}

func TestMatchRedirectURL(t *testing.T) {
	allowlist := []string{
		"https://app.example.com/api/v1/auth/oauth/google/callback",
		"https://custom.example.com/api/v1/auth/oauth/google/callback",
	}
	baseURL := "https://app.example.com"

	t.Run("exact match", func(t *testing.T) {
		assert.True(t, matchRedirectURL(allowlist, "https://custom.example.com/api/v1/auth/oauth/google/callback", baseURL))
	})

	t.Run("relative path is resolved against baseURL", func(t *testing.T) {
		assert.True(t, matchRedirectURL(allowlist, "/api/v1/auth/oauth/google/callback", baseURL))
	})

	t.Run("mismatch is rejected", func(t *testing.T) {
		assert.False(t, matchRedirectURL(allowlist, "https://evil.example.com/api/v1/auth/oauth/google/callback", baseURL))
	})

	t.Run("prefix of an allowed URL is rejected", func(t *testing.T) {
		assert.False(t, matchRedirectURL(allowlist, "https://app.example.com/api/v1/auth/oauth/google/callback/extra", baseURL))
	})

	t.Run("empty candidate means no override and matches", func(t *testing.T) {
		assert.True(t, matchRedirectURL(allowlist, "", baseURL))
	})

	t.Run("deep link exact match", func(t *testing.T) {
		withDeepLink := append(allowlist, "wayli://oauth/callback")
		assert.True(t, matchRedirectURL(withDeepLink, "wayli://oauth/callback", baseURL))
		assert.False(t, matchRedirectURL(withDeepLink, "wayli://other", baseURL))
	})

	t.Run("empty allowlist rejects any override", func(t *testing.T) {
		assert.False(t, matchRedirectURL(nil, "https://app.example.com/cb", baseURL))
	})
}

func TestResolveConfigRedirectURLs(t *testing.T) {
	t.Run("derived URL only by default", func(t *testing.T) {
		cp := config.OAuthProviderConfig{Name: "google"}
		assert.Equal(t, []string{"https://flux.example.com/api/v1/auth/oauth/google/callback"},
			resolveConfigRedirectURLs(cp, "https://flux.example.com/api/v1/auth/oauth/google/callback"))
	})

	t.Run("configured extras are appended without duplicates", func(t *testing.T) {
		cp := config.OAuthProviderConfig{
			Name:         "google",
			RedirectURLs: []string{"https://alt.example.com/cb", "https://flux.example.com/api/v1/auth/oauth/google/callback", "  "},
		}
		assert.Equal(t, []string{
			"https://flux.example.com/api/v1/auth/oauth/google/callback",
			"https://alt.example.com/cb",
		}, resolveConfigRedirectURLs(cp, "https://flux.example.com/api/v1/auth/oauth/google/callback"))
	})
}

func TestFirstRedirectURL(t *testing.T) {
	assert.Equal(t, "https://a.example.com/cb", firstRedirectURL([]string{"https://a.example.com/cb", "https://b.example.com/cb"}))
	assert.Empty(t, firstRedirectURL(nil))
}
