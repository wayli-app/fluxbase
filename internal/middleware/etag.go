package middleware

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

// ETagConfig defines the configuration for ETag middleware
type ETagConfig struct {
	// Weak determines if the ETag should be a weak validator (W/"...")
	// Weak ETags indicate semantic equivalence, not byte-for-byte equality
	Weak bool

	// SkipPaths are path prefixes that should not have ETags
	SkipPaths []string

	// IncludeMethods are HTTP methods that should include ETags
	// Defaults to GET, HEAD
	IncludeMethods []string

	// EnableConditional enables checking If-None-Match header
	// and returning 304 Not Modified when ETag matches
	EnableConditional bool
}

// DefaultETagConfig returns the default configuration
func DefaultETagConfig() ETagConfig {
	return ETagConfig{
		Weak:              true,
		SkipPaths:         []string{"/health", "/metrics", "/realtime"},
		IncludeMethods:    []string{"GET", "HEAD"},
		EnableConditional: true,
	}
}

// ETag creates a middleware that adds ETag headers to responses
// and handles conditional requests (If-None-Match)
func ETag() fiber.Handler {
	return ETagWithConfig(DefaultETagConfig())
}

// ETagWithConfig creates an ETag middleware with custom configuration
func ETagWithConfig(config ETagConfig) fiber.Handler {
	// Apply defaults
	if len(config.IncludeMethods) == 0 {
		config.IncludeMethods = []string{"GET", "HEAD"}
	}

	// Create method lookup for O(1) checks
	methodSet := make(map[string]bool)
	for _, m := range config.IncludeMethods {
		methodSet[strings.ToUpper(m)] = true
	}

	return func(c *fiber.Ctx) error {
		// Check if method should have ETag
		if !methodSet[c.Method()] {
			return c.Next()
		}

		// Check if path should be skipped
		path := c.Path()
		for _, skipPath := range config.SkipPaths {
			if strings.HasPrefix(path, skipPath) {
				return c.Next()
			}
		}

		// Process the request first
		if err := c.Next(); err != nil {
			return err
		}

		// Only add ETag for successful responses
		status := c.Response().StatusCode()
		if status < 200 || status >= 300 {
			return nil
		}

		// Get response body
		body := c.Response().Body()
		if len(body) == 0 {
			return nil
		}

		// Generate ETag from body hash
		etag := generateETag(body, config.Weak)

		// Set ETag header
		c.Set("ETag", etag)

		// Handle conditional request if enabled
		if config.EnableConditional {
			ifNoneMatch := c.Get("If-None-Match")
			if ifNoneMatch != "" && etagMatches(etag, ifNoneMatch) {
				// Return 304 Not Modified
				c.Status(304)
				c.Response().ResetBody()
				return nil
			}
		}

		return nil
	}
}

// generateETag creates an ETag from response body
func generateETag(body []byte, weak bool) string {
	hash := sha256.Sum256(body)
	// Use first 16 bytes of hash (32 hex chars)
	hashStr := hex.EncodeToString(hash[:16])

	if weak {
		return `W/"` + hashStr + `"`
	}
	return `"` + hashStr + `"`
}

// etagMatches checks if the current ETag matches any in the If-None-Match header
// Handles multiple ETags separated by commas and the * wildcard
func etagMatches(etag, ifNoneMatch string) bool {
	// Trim whitespace
	ifNoneMatch = strings.TrimSpace(ifNoneMatch)

	// Handle * wildcard
	if ifNoneMatch == "*" {
		return true
	}

	// Parse multiple ETags
	candidates := strings.Split(ifNoneMatch, ",")
	for _, candidate := range candidates {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}

		// Normalize for comparison (weak vs strong comparison)
		normalizedCandidate := normalizeETag(candidate)
		normalizedETag := normalizeETag(etag)

		if normalizedCandidate == normalizedETag {
			return true
		}
	}

	return false
}

// normalizeETag removes weak indicator for comparison
// RFC 7232: For weak comparison, ETags are equivalent if their opaque-tags match
func normalizeETag(etag string) string {
	etag = strings.TrimSpace(etag)
	// Remove W/ prefix for weak comparison
	if strings.HasPrefix(etag, "W/") {
		etag = etag[2:]
	}
	return etag
}

// LastModifiedMiddleware adds Last-Modified header based on response data
// This is a helper that can be used alongside ETag middleware
func LastModifiedMiddleware(timestampField string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Process the request
		if err := c.Next(); err != nil {
			return err
		}

		// Only for GET requests with successful responses
		if c.Method() != "GET" {
			return nil
		}
		status := c.Response().StatusCode()
		if status < 200 || status >= 300 {
			return nil
		}

		// Check If-Modified-Since header
		ifModifiedSince := c.Get("If-Modified-Since")
		if ifModifiedSince != "" {
			// Parse the If-Modified-Since date
			modSince, err := time.Parse(time.RFC1123, ifModifiedSince)
			if err == nil {
				// Get Last-Modified from response if set
				lastModified := c.Get("Last-Modified")
				if lastModified != "" {
					lastMod, err := time.Parse(time.RFC1123, lastModified)
					if err == nil && !lastMod.After(modSince) {
						// Not modified since
						c.Status(304)
						c.Response().ResetBody()
						return nil
					}
				}
			}
		}

		return nil
	}
}

// CacheControlMiddleware sets Cache-Control headers based on configuration
type CacheControlConfig struct {
	// MaxAge in seconds for public caching
	MaxAge int

	// SMaxAge in seconds for shared/proxy caching
	SMaxAge int

	// Private if true, response is for single user only
	Private bool

	// NoCache requires revalidation before using cached response
	NoCache bool

	// NoStore prevents any caching
	NoStore bool

	// MustRevalidate requires revalidation after max-age expires
	MustRevalidate bool
}

// CacheControl creates a middleware that sets Cache-Control headers
func CacheControl(config CacheControlConfig) fiber.Handler {
	// Build Cache-Control header value
	var directives []string

	if config.NoStore {
		directives = append(directives, "no-store")
	} else {
		if config.Private {
			directives = append(directives, "private")
		} else if config.MaxAge > 0 || config.SMaxAge > 0 {
			directives = append(directives, "public")
		}

		if config.NoCache {
			directives = append(directives, "no-cache")
		}

		if config.MaxAge > 0 {
			directives = append(directives, "max-age="+itoa(config.MaxAge))
		}

		if config.SMaxAge > 0 {
			directives = append(directives, "s-maxage="+itoa(config.SMaxAge))
		}

		if config.MustRevalidate {
			directives = append(directives, "must-revalidate")
		}
	}

	cacheControl := strings.Join(directives, ", ")

	return func(c *fiber.Ctx) error {
		// Only set for GET and HEAD requests
		if c.Method() != "GET" && c.Method() != "HEAD" {
			return c.Next()
		}

		// Process the request
		if err := c.Next(); err != nil {
			return err
		}

		// Only set for successful responses
		status := c.Response().StatusCode()
		if status >= 200 && status < 300 && cacheControl != "" {
			c.Set("Cache-Control", cacheControl)
		}

		return nil
	}
}

// itoa is a simple int to string conversion
func itoa(i int) string {
	if i == 0 {
		return "0"
	}

	var negative bool
	if i < 0 {
		negative = true
		i = -i
	}

	var buf [20]byte
	pos := len(buf)

	for i > 0 {
		pos--
		buf[pos] = byte('0' + i%10)
		i /= 10
	}

	if negative {
		pos--
		buf[pos] = '-'
	}

	return string(buf[pos:])
}
