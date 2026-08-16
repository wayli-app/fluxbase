package api

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/nimbleflux/fluxbase/internal/config"
)

// maxRedirectURLs caps how many redirect URLs a single OAuth provider may configure.
const maxRedirectURLs = 50

// forbiddenRedirectSchemes lists schemes that must never appear in a redirect
// URL allowlist. Web schemes (http/https) are validated separately; any other
// scheme is treated as a private-use app scheme (RFC 8252, e.g.
// "wayli://oauth/callback") and allowed unless listed here. This is safe
// because authorize/callback enforce an exact-string match against the
// stored allowlist, so entries only redirect where an admin explicitly
// configured.
var forbiddenRedirectSchemes = map[string]bool{
	"javascript": true,
	"data":       true,
	"file":       true,
	"blob":       true,
	"vbscript":   true,
	"ftp":        true,
	"ftps":       true,
	"ws":         true,
	"wss":        true,
	"mailto":     true,
	"urn":        true,
	"about":      true,
	"intent":     true,
}

// normalizeRedirectURLs validates and normalizes a redirect URL list for storage.
// Entries are trimmed, must be absolute http/https URLs or private-use app
// schemes (RFC 8252, e.g. "wayli://oauth/callback"), duplicates are removed
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
		if err != nil || parsed.Scheme == "" {
			return nil, fmt.Errorf("invalid redirect URL '%s': must be an absolute http/https URL or app scheme URI", u)
		}
		scheme := strings.ToLower(parsed.Scheme)
		if scheme == "http" || scheme == "https" {
			if parsed.Host == "" {
				return nil, fmt.Errorf("invalid redirect URL '%s': http/https URLs must include a host", u)
			}
		} else {
			if forbiddenRedirectSchemes[scheme] {
				return nil, fmt.Errorf("invalid redirect URL '%s': scheme '%s' is not allowed", u, parsed.Scheme)
			}
			// Reject scheme-only URIs with nothing to match against (e.g. "wayli:")
			if parsed.Host == "" && parsed.Opaque == "" && parsed.Path == "" {
				return nil, fmt.Errorf("invalid redirect URL '%s': app scheme URIs must include a host or path", u)
			}
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
