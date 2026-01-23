package middleware

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
)

// BodyLimitConfig holds per-endpoint body size limit configuration
type BodyLimitConfig struct {
	// Default limit when no pattern matches (defaults to 1MB)
	DefaultLimit int64

	// Pattern-based limits (evaluated in order, first match wins)
	// Patterns support basic glob: * for single path segment, ** for any path
	Patterns []BodyLimitPattern

	// MaxJSONDepth limits nesting depth to prevent stack overflow (0 = unlimited)
	MaxJSONDepth int

	// ErrorHandler is called when the request body exceeds the limit
	// If nil, a default 413 response is sent
	ErrorHandler func(c *fiber.Ctx, limit int64) error
}

// BodyLimitPattern defines a limit for matching routes
type BodyLimitPattern struct {
	// Pattern is a glob-style path pattern (e.g., "/api/v1/storage/**", "/api/v1/rest/*")
	Pattern string

	// Limit is the maximum body size in bytes
	Limit int64

	// Description is used for logging and error messages
	Description string
}

// Default body limits for different endpoint types
const (
	// DefaultBodyLimit is the default for most endpoints (1MB)
	DefaultBodyLimit int64 = 1 * 1024 * 1024

	// RESTBodyLimit is for REST API CRUD operations (1MB)
	RESTBodyLimit int64 = 1 * 1024 * 1024

	// AuthBodyLimit is for authentication endpoints (64KB)
	AuthBodyLimit int64 = 64 * 1024

	// StorageUploadLimit is for file uploads (100MB per request body)
	// Note: Streaming uploads bypass this since body isn't buffered
	StorageUploadLimit int64 = 100 * 1024 * 1024

	// MultipartUploadLimit is for multipart form uploads (100MB)
	MultipartUploadLimit int64 = 100 * 1024 * 1024

	// LargePayloadLimit is for bulk operations (10MB)
	LargePayloadLimit int64 = 10 * 1024 * 1024

	// AdminLimit is for admin endpoints (5MB)
	AdminLimit int64 = 5 * 1024 * 1024

	// WebhookLimit is for incoming webhooks (1MB)
	WebhookLimit int64 = 1 * 1024 * 1024

	// DefaultMaxJSONDepth prevents deeply nested JSON attacks
	DefaultMaxJSONDepth = 64
)

// DefaultBodyLimitPatterns returns the default patterns for Fluxbase endpoints
func DefaultBodyLimitPatterns() []BodyLimitPattern {
	return []BodyLimitPattern{
		// Storage uploads - larger limits
		{Pattern: "/api/v1/storage/*/multipart", Limit: MultipartUploadLimit, Description: "multipart upload"},
		{Pattern: "/api/v1/storage/*/stream/**", Limit: StorageUploadLimit, Description: "stream upload"},
		{Pattern: "/api/v1/storage/*/chunked/**", Limit: StorageUploadLimit, Description: "chunked upload"},
		{Pattern: "/api/v1/storage/**", Limit: StorageUploadLimit, Description: "storage"},

		// Admin sync endpoints - need larger limits for bundled code (can be 100+ MB)
		{Pattern: "/api/v1/admin/functions/sync", Limit: StorageUploadLimit, Description: "functions sync"},
		{Pattern: "/api/v1/admin/jobs/sync", Limit: StorageUploadLimit, Description: "jobs sync"},
		{Pattern: "/api/v1/admin/ai/chatbots/sync", Limit: StorageUploadLimit, Description: "chatbots sync"},
		{Pattern: "/api/v1/admin/rpc/sync", Limit: StorageUploadLimit, Description: "RPC sync"},
		{Pattern: "/api/v1/admin/migrations/sync", Limit: StorageUploadLimit, Description: "migrations sync"},

		// Admin endpoints (general)
		{Pattern: "/api/v1/admin/**", Limit: AdminLimit, Description: "admin"},
		{Pattern: "/api/v1/ai/**", Limit: AdminLimit, Description: "AI/vectors"},

		// Auth endpoints - small limits
		{Pattern: "/api/v1/auth/**", Limit: AuthBodyLimit, Description: "auth"},

		// Webhooks
		{Pattern: "/api/v1/webhooks/**", Limit: WebhookLimit, Description: "webhooks"},
		{Pattern: "/api/v1/functions/webhooks/**", Limit: WebhookLimit, Description: "function webhooks"},

		// Bulk operations - larger limits
		{Pattern: "/api/v1/rest/*/bulk", Limit: LargePayloadLimit, Description: "bulk operations"},
		{Pattern: "/api/v1/rpc/**", Limit: LargePayloadLimit, Description: "RPC"},

		// REST endpoints - standard limit
		{Pattern: "/api/v1/rest/**", Limit: RESTBodyLimit, Description: "REST"},

		// GraphQL
		{Pattern: "/graphql", Limit: LargePayloadLimit, Description: "GraphQL"},

		// MCP endpoints
		{Pattern: "/mcp/**", Limit: LargePayloadLimit, Description: "MCP"},

		// Realtime (small payloads)
		{Pattern: "/api/v1/realtime/**", Limit: AuthBodyLimit, Description: "realtime"},

		// Default for any other API endpoint
		{Pattern: "/api/**", Limit: RESTBodyLimit, Description: "API"},
	}
}

// DefaultBodyLimitConfig returns the default body limit configuration
func DefaultBodyLimitConfig() BodyLimitConfig {
	return BodyLimitConfig{
		DefaultLimit: DefaultBodyLimit,
		Patterns:     DefaultBodyLimitPatterns(),
		MaxJSONDepth: DefaultMaxJSONDepth,
	}
}

// BodyLimitsFromConfig creates a BodyLimitConfig from configurable limits.
// This allows overriding the default limits per endpoint type via configuration.
func BodyLimitsFromConfig(defaultLimit, restLimit, authLimit, storageLimit, bulkLimit, adminLimit int64, maxJSONDepth int) BodyLimitConfig {
	// Use defaults if not specified
	if defaultLimit <= 0 {
		defaultLimit = DefaultBodyLimit
	}
	if restLimit <= 0 {
		restLimit = RESTBodyLimit
	}
	if authLimit <= 0 {
		authLimit = AuthBodyLimit
	}
	if storageLimit <= 0 {
		storageLimit = StorageUploadLimit
	}
	if bulkLimit <= 0 {
		bulkLimit = LargePayloadLimit
	}
	if adminLimit <= 0 {
		adminLimit = AdminLimit
	}
	if maxJSONDepth <= 0 {
		maxJSONDepth = DefaultMaxJSONDepth
	}

	patterns := []BodyLimitPattern{
		// Storage uploads - larger limits
		{Pattern: "/api/v1/storage/*/multipart", Limit: storageLimit, Description: "multipart upload"},
		{Pattern: "/api/v1/storage/*/stream/**", Limit: storageLimit, Description: "stream upload"},
		{Pattern: "/api/v1/storage/*/chunked/**", Limit: storageLimit, Description: "chunked upload"},
		{Pattern: "/api/v1/storage/**", Limit: storageLimit, Description: "storage"},

		// Admin sync endpoints - need larger limits for bundled code (can be 100+ MB)
		{Pattern: "/api/v1/admin/functions/sync", Limit: storageLimit, Description: "functions sync"},
		{Pattern: "/api/v1/admin/jobs/sync", Limit: storageLimit, Description: "jobs sync"},
		{Pattern: "/api/v1/admin/ai/chatbots/sync", Limit: storageLimit, Description: "chatbots sync"},
		{Pattern: "/api/v1/admin/rpc/sync", Limit: storageLimit, Description: "RPC sync"},
		{Pattern: "/api/v1/admin/migrations/sync", Limit: storageLimit, Description: "migrations sync"},

		// Admin endpoints (general)
		{Pattern: "/api/v1/admin/**", Limit: adminLimit, Description: "admin"},
		{Pattern: "/api/v1/ai/**", Limit: adminLimit, Description: "AI/vectors"},

		// Auth endpoints - small limits
		{Pattern: "/api/v1/auth/**", Limit: authLimit, Description: "auth"},

		// Webhooks
		{Pattern: "/api/v1/webhooks/**", Limit: restLimit, Description: "webhooks"},
		{Pattern: "/api/v1/functions/webhooks/**", Limit: restLimit, Description: "function webhooks"},

		// Bulk operations - larger limits
		{Pattern: "/api/v1/rest/*/bulk", Limit: bulkLimit, Description: "bulk operations"},
		{Pattern: "/api/v1/rpc/**", Limit: bulkLimit, Description: "RPC"},

		// REST endpoints - standard limit
		{Pattern: "/api/v1/rest/**", Limit: restLimit, Description: "REST"},

		// GraphQL
		{Pattern: "/graphql", Limit: bulkLimit, Description: "GraphQL"},

		// MCP endpoints
		{Pattern: "/mcp/**", Limit: bulkLimit, Description: "MCP"},

		// Realtime (small payloads)
		{Pattern: "/api/v1/realtime/**", Limit: authLimit, Description: "realtime"},

		// Default for any other API endpoint
		{Pattern: "/api/**", Limit: restLimit, Description: "API"},
	}

	return BodyLimitConfig{
		DefaultLimit: defaultLimit,
		Patterns:     patterns,
		MaxJSONDepth: maxJSONDepth,
	}
}

// compiledPattern holds a pre-compiled pattern for efficient matching
type compiledPattern struct {
	original         string
	parts            []string
	isWildcard       []bool // true for * or ** segments
	isDoubleWildcard []bool
	limit            int64
	description      string
}

// PatternBodyLimiter provides efficient route-based body limiting
type PatternBodyLimiter struct {
	config   BodyLimitConfig
	patterns []compiledPattern
	mu       sync.RWMutex
}

// NewPatternBodyLimiter creates a new pattern-based body limiter
func NewPatternBodyLimiter(config BodyLimitConfig) *PatternBodyLimiter {
	limiter := &PatternBodyLimiter{
		config: config,
	}
	limiter.compilePatterns()
	return limiter
}

// compilePatterns pre-processes patterns for efficient matching
func (l *PatternBodyLimiter) compilePatterns() {
	l.patterns = make([]compiledPattern, 0, len(l.config.Patterns))

	for _, p := range l.config.Patterns {
		parts := strings.Split(strings.Trim(p.Pattern, "/"), "/")
		isWildcard := make([]bool, len(parts))
		isDoubleWildcard := make([]bool, len(parts))

		for i, part := range parts {
			isWildcard[i] = part == "*" || part == "**"
			isDoubleWildcard[i] = part == "**"
		}

		l.patterns = append(l.patterns, compiledPattern{
			original:         p.Pattern,
			parts:            parts,
			isWildcard:       isWildcard,
			isDoubleWildcard: isDoubleWildcard,
			limit:            p.Limit,
			description:      p.Description,
		})
	}
}

// matchPattern checks if a path matches a compiled pattern
func (l *PatternBodyLimiter) matchPattern(path string, pattern compiledPattern) bool {
	pathParts := strings.Split(strings.Trim(path, "/"), "/")
	patternParts := pattern.parts

	pathIdx := 0
	patternIdx := 0

	for patternIdx < len(patternParts) && pathIdx < len(pathParts) {
		if pattern.isDoubleWildcard[patternIdx] {
			// ** matches zero or more segments
			// If this is the last pattern part, it matches everything
			if patternIdx == len(patternParts)-1 {
				return true
			}
			// Try to match the rest of the pattern against remaining path parts
			for i := pathIdx; i <= len(pathParts); i++ {
				if l.matchRemainingPattern(pathParts, i, patternParts, patternIdx+1, pattern) {
					return true
				}
			}
			return false
		} else if pattern.isWildcard[patternIdx] {
			// * matches exactly one segment
			pathIdx++
			patternIdx++
		} else if patternParts[patternIdx] == pathParts[pathIdx] {
			// Exact match
			pathIdx++
			patternIdx++
		} else {
			return false
		}
	}

	// Check if we consumed all parts
	return pathIdx == len(pathParts) && patternIdx == len(patternParts)
}

// matchRemainingPattern is a helper for ** matching
func (l *PatternBodyLimiter) matchRemainingPattern(pathParts []string, pathIdx int, patternParts []string, patternIdx int, pattern compiledPattern) bool {
	for patternIdx < len(patternParts) && pathIdx < len(pathParts) {
		if pattern.isDoubleWildcard[patternIdx] {
			// Another ** - recurse
			if patternIdx == len(patternParts)-1 {
				return true
			}
			for i := pathIdx; i <= len(pathParts); i++ {
				if l.matchRemainingPattern(pathParts, i, patternParts, patternIdx+1, pattern) {
					return true
				}
			}
			return false
		} else if pattern.isWildcard[patternIdx] {
			pathIdx++
			patternIdx++
		} else if patternParts[patternIdx] == pathParts[pathIdx] {
			pathIdx++
			patternIdx++
		} else {
			return false
		}
	}

	return pathIdx == len(pathParts) && patternIdx == len(patternParts)
}

// GetLimit returns the body limit for a given path
func (l *PatternBodyLimiter) GetLimit(path string) (limit int64, description string) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	for _, pattern := range l.patterns {
		if l.matchPattern(path, pattern) {
			return pattern.limit, pattern.description
		}
	}

	return l.config.DefaultLimit, "default"
}

// Middleware returns a Fiber middleware that enforces body limits
func (l *PatternBodyLimiter) Middleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Skip body limit check for GET, HEAD, OPTIONS requests
		method := c.Method()
		if method == fiber.MethodGet || method == fiber.MethodHead || method == fiber.MethodOptions {
			return c.Next()
		}

		path := c.Path()
		limit, description := l.GetLimit(path)

		// Get content length if available
		contentLength := c.Request().Header.ContentLength()
		if contentLength > 0 && contentLength > int(limit) {
			log.Debug().
				Str("path", path).
				Int64("content_length", int64(contentLength)).
				Int64("limit", limit).
				Str("endpoint_type", description).
				Msg("Request body exceeds limit (Content-Length)")

			return l.sendLimitExceeded(c, limit, description)
		}

		return c.Next()
	}
}

// sendLimitExceeded sends a 413 Payload Too Large response
func (l *PatternBodyLimiter) sendLimitExceeded(c *fiber.Ctx, limit int64, description string) error {
	if l.config.ErrorHandler != nil {
		return l.config.ErrorHandler(c, limit)
	}

	return c.Status(fiber.StatusRequestEntityTooLarge).JSON(fiber.Map{
		"error":   "Payload Too Large",
		"code":    "PAYLOAD_TOO_LARGE",
		"message": fmt.Sprintf("Request body exceeds maximum size of %s for %s endpoints", formatBytes(limit), description),
		"hint":    "Reduce the size of your request body or use chunked upload for large files",
		"details": fiber.Map{
			"max_bytes":     limit,
			"endpoint_type": description,
		},
	})
}

// formatBytes formats bytes into human-readable format
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// JSONDepthLimiter validates JSON body depth to prevent stack overflow attacks
type JSONDepthLimiter struct {
	maxDepth int
}

// NewJSONDepthLimiter creates a new JSON depth limiter
func NewJSONDepthLimiter(maxDepth int) *JSONDepthLimiter {
	if maxDepth <= 0 {
		maxDepth = DefaultMaxJSONDepth
	}
	return &JSONDepthLimiter{maxDepth: maxDepth}
}

// CheckDepth validates JSON depth for the request body and returns an error response if exceeded
func (l *JSONDepthLimiter) CheckDepth(c *fiber.Ctx) error {
	// Only check JSON content
	contentType := string(c.Request().Header.ContentType())
	if !strings.Contains(contentType, "application/json") {
		return nil
	}

	body := c.Body()
	if len(body) == 0 {
		return nil
	}

	// Check JSON depth
	depth, err := checkJSONDepth(body, l.maxDepth)
	if err != nil {
		log.Debug().
			Err(err).
			Int("depth", depth).
			Int("max_depth", l.maxDepth).
			Str("path", c.Path()).
			Msg("JSON depth validation failed")

		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Invalid JSON",
			"code":    "JSON_TOO_DEEP",
			"message": fmt.Sprintf("JSON nesting exceeds maximum depth of %d", l.maxDepth),
			"hint":    "Flatten your JSON structure or reduce nesting levels",
		})
	}

	return nil
}

// Middleware returns a Fiber middleware that validates JSON depth
func (l *JSONDepthLimiter) Middleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Skip for GET, HEAD, OPTIONS
		method := c.Method()
		if method == fiber.MethodGet || method == fiber.MethodHead || method == fiber.MethodOptions {
			return c.Next()
		}

		if err := l.CheckDepth(c); err != nil {
			return err
		}

		return c.Next()
	}
}

// checkJSONDepth validates that JSON doesn't exceed max nesting depth
// Returns the max depth encountered and error if exceeded
func checkJSONDepth(data []byte, maxDepth int) (int, error) {
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	depth := 0
	maxFound := 0

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			// Invalid JSON - let the handler deal with it
			return maxFound, nil
		}

		switch token {
		case json.Delim('['), json.Delim('{'):
			depth++
			if depth > maxFound {
				maxFound = depth
			}
			if depth > maxDepth {
				return maxFound, fmt.Errorf("JSON depth %d exceeds maximum %d", depth, maxDepth)
			}
		case json.Delim(']'), json.Delim('}'):
			depth--
		}
	}

	return maxFound, nil
}

// BodyLimitMiddleware creates a combined middleware for body size and JSON depth limits
func BodyLimitMiddleware(config BodyLimitConfig) fiber.Handler {
	bodyLimiter := NewPatternBodyLimiter(config)
	jsonLimiter := NewJSONDepthLimiter(config.MaxJSONDepth)

	return func(c *fiber.Ctx) error {
		// Skip body limit check for GET, HEAD, OPTIONS requests
		method := c.Method()
		if method == fiber.MethodGet || method == fiber.MethodHead || method == fiber.MethodOptions {
			return c.Next()
		}

		// First check body size limit
		path := c.Path()
		limit, description := bodyLimiter.GetLimit(path)

		// Get content length if available
		contentLength := c.Request().Header.ContentLength()
		if contentLength > 0 && contentLength > int(limit) {
			log.Debug().
				Str("path", path).
				Int64("content_length", int64(contentLength)).
				Int64("limit", limit).
				Str("endpoint_type", description).
				Msg("Request body exceeds limit (Content-Length)")

			return bodyLimiter.sendLimitExceeded(c, limit, description)
		}

		// Then check JSON depth (only for JSON requests)
		if err := jsonLimiter.CheckDepth(c); err != nil {
			return err
		}

		return c.Next()
	}
}
