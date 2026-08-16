package api

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/nimbleflux/fluxbase/internal/config"
)

// maxRedirectURLs caps how many redirect URLs a single OAuth provider may configure.
const maxRedirectURLs = 50

// normalizeRedirectURLs validates and normalizes a redirect URL list for storage.
// Entries are trimmed, must be absolute http/https URLs, duplicates are removed
// (order preserved), and the list is capped at maxRedirectURLs entries.
func normalizeRedirectURLs(urls []string) ([]string, error) {
	if len(urls) > maxRedirectURLs {
		return nil, fmt.Errorf("too many redirect URLs (max %d)", maxRedirectURLs)
	}
	normalized := make([]string, 0, len(urls))
	seen := make(map[string]bool, len(urls))
	for _, raw := range urls {
		u := strings.TrimSpace(raw)
		if u == "" {
			return nil, fmt.Errorf("redirect URLs cannot contain empty entries")
		}
		parsed, err := url.Parse(u)
		if err != nil || parsed.Host == "" {
			return nil, fmt.Errorf("invalid redirect URL '%s': must be an absolute http/https URL", u)
		}
		if parsed.Scheme != "http" && parsed.Scheme != "https" {
			return nil, fmt.Errorf("invalid redirect URL '%s': scheme must be http or https", u)
		}
		if !seen[u] {
			seen[u] = true
			normalized = append(normalized, u)
		}
	}
	return normalized, nil
}

// resolveRedirectURICandidate absolutizes a redirect URI the same way the
// authorize and callback handlers do: relative paths are prefixed with baseURL.
func resolveRedirectURICandidate(candidate, baseURL string) string {
	if strings.HasPrefix(candidate, "/") {
		return baseURL + candidate
	}
	return candidate
}

// matchRedirectURL reports whether candidate (after relative→absolute
// normalization) exactly matches one of the configured redirect URLs.
// An empty candidate means no override was requested and always matches.
func matchRedirectURL(allowlist []string, candidate, baseURL string) bool {
	if candidate == "" {
		return true
	}
	c := resolveRedirectURICandidate(candidate, baseURL)
	for _, allowed := range allowlist {
		if c == allowed {
			return true
		}
	}
	return false
}

// resolveConfigRedirectURLs builds the redirect URL list for a config-file
// provider: the derived callback URL first, followed by any explicitly
// configured redirect_urls (duplicates removed).
func resolveConfigRedirectURLs(cp config.OAuthProviderConfig, derivedURL string) []string {
	urls := []string{derivedURL}
	seen := map[string]bool{derivedURL: true}
	for _, u := range cp.RedirectURLs {
		u = strings.TrimSpace(u)
		if u != "" && !seen[u] {
			seen[u] = true
			urls = append(urls, u)
		}
	}
	return urls
}
