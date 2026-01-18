package mcp

import (
	"strings"
	"sync"
	"time"

	"github.com/fluxbase-eu/fluxbase/internal/config"
	"github.com/fluxbase-eu/fluxbase/internal/database"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
)

// rateLimiter tracks request counts per client key using a sliding window
type rateLimiter struct {
	mu       sync.Mutex
	requests map[string][]time.Time // client key -> list of request timestamps
	limit    int                    // requests per minute
}

// newRateLimiter creates a new rate limiter
func newRateLimiter(limitPerMin int) *rateLimiter {
	rl := &rateLimiter{
		requests: make(map[string][]time.Time),
		limit:    limitPerMin,
	}
	// Start cleanup goroutine
	go rl.cleanup()
	return rl
}

// allow checks if a request from the given client key should be allowed
func (rl *rateLimiter) allow(clientKey string) bool {
	if rl.limit <= 0 {
		return true // Rate limiting disabled
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-time.Minute)

	// Get existing requests and filter to sliding window
	existing := rl.requests[clientKey]
	var valid []time.Time
	for _, t := range existing {
		if t.After(windowStart) {
			valid = append(valid, t)
		}
	}

	// Check if we're at limit
	if len(valid) >= rl.limit {
		rl.requests[clientKey] = valid // Update with filtered list
		return false
	}

	// Allow and record
	rl.requests[clientKey] = append(valid, now)
	return true
}

// cleanup periodically removes old entries to prevent memory growth
func (rl *rateLimiter) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		windowStart := now.Add(-time.Minute)

		for key, times := range rl.requests {
			var valid []time.Time
			for _, t := range times {
				if t.After(windowStart) {
					valid = append(valid, t)
				}
			}
			if len(valid) == 0 {
				delete(rl.requests, key)
			} else {
				rl.requests[key] = valid
			}
		}
		rl.mu.Unlock()
	}
}

// Handler handles HTTP requests for the MCP server
type Handler struct {
	server      *Server
	config      *config.MCPConfig
	db          *database.Connection
	rateLimiter *rateLimiter
}

// NewHandler creates a new MCP HTTP handler
func NewHandler(cfg *config.MCPConfig, db *database.Connection) *Handler {
	return &Handler{
		server:      NewServer(cfg),
		config:      cfg,
		db:          db,
		rateLimiter: newRateLimiter(cfg.RateLimitPerMin),
	}
}

// Server returns the underlying MCP server for tool/resource registration
func (h *Handler) Server() *Server {
	return h.server
}

// RegisterRoutes registers the MCP routes
// Auth middleware should be applied before calling this
func (h *Handler) RegisterRoutes(app fiber.Router) {
	// Health check endpoint (no auth required)
	app.Get("/health", h.handleHealth)

	// Main MCP endpoint - Streamable HTTP transport
	// POST: Send JSON-RPC requests
	// GET: Initiate SSE stream for server notifications (optional)
	app.Post("/", h.handlePost)
	app.Get("/", h.handleGet)
}

// handleHealth handles health check requests
func (h *Handler) handleHealth(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status":          "healthy",
		"protocolVersion": MCPVersion,
		"serverVersion":   FluxbaseVersion,
	})
}

// handlePost handles JSON-RPC POST requests
func (h *Handler) handlePost(c *fiber.Ctx) error {
	// Check content type
	contentType := c.Get("Content-Type")
	if !strings.HasPrefix(contentType, "application/json") {
		return c.Status(fiber.StatusUnsupportedMediaType).JSON(fiber.Map{
			"error": "Content-Type must be application/json",
		})
	}

	// Check message size
	if h.config.MaxMessageSize > 0 && len(c.Body()) > h.config.MaxMessageSize {
		return c.Status(fiber.StatusRequestEntityTooLarge).JSON(fiber.Map{
			"error": "Request body too large",
		})
	}

	// Extract auth context from Fiber locals (set by auth middleware)
	authCtx := ExtractAuthContext(c)

	// Check rate limit using client key or user ID as the key
	rateLimitKey := authCtx.ClientKeyID
	if rateLimitKey == "" && authCtx.UserID != nil && *authCtx.UserID != "" {
		rateLimitKey = *authCtx.UserID
	}
	if rateLimitKey == "" {
		rateLimitKey = c.IP() // Fallback to IP for anonymous requests
	}
	if !h.rateLimiter.allow(rateLimitKey) {
		log.Warn().
			Str("client_key", rateLimitKey).
			Int("limit", h.config.RateLimitPerMin).
			Msg("MCP: Rate limit exceeded")
		return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
			"error": "Rate limit exceeded. Please try again later.",
		})
	}

	// Log the request
	log.Debug().
		Str("auth_type", authCtx.AuthType).
		Str("user_role", authCtx.UserRole).
		Bool("authenticated", authCtx.IsAuthenticated()).
		Msg("MCP: Handling POST request")

	// Process the request
	response := h.server.HandleRequest(c.Context(), c.Body(), authCtx)

	// Check Accept header for response format
	accept := c.Get("Accept")

	// For streaming responses, use SSE if client accepts it
	if strings.Contains(accept, "text/event-stream") {
		return h.sendSSEResponse(c, response)
	}

	// Default to JSON response
	return h.sendJSONResponse(c, response)
}

// handleGet handles GET requests for SSE stream initiation
func (h *Handler) handleGet(c *fiber.Ctx) error {
	// Check if client accepts SSE
	accept := c.Get("Accept")
	if !strings.Contains(accept, "text/event-stream") {
		return c.Status(fiber.StatusNotAcceptable).JSON(fiber.Map{
			"error": "GET requests require Accept: text/event-stream header",
		})
	}

	// For now, we don't support server-initiated notifications
	// This would be implemented with a long-lived SSE connection
	return c.Status(fiber.StatusNotImplemented).JSON(fiber.Map{
		"error": "Server-initiated notifications not yet implemented",
	})
}

// sendJSONResponse sends a JSON-RPC response
func (h *Handler) sendJSONResponse(c *fiber.Ctx, response *Response) error {
	c.Set("Content-Type", "application/json")
	return c.JSON(response)
}

// sendSSEResponse sends a response as an SSE event
func (h *Handler) sendSSEResponse(c *fiber.Ctx, response *Response) error {
	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")

	// Serialize the response
	data, err := h.server.SerializeResponse(response)
	if err != nil {
		log.Error().Err(err).Msg("MCP: Failed to serialize response")
		return c.SendString("data: {\"error\":\"serialization error\"}\n\n")
	}

	// Send as SSE event - single event for now (not a stream)
	// Format: "data: <json>\n\n"
	return c.SendString("data: " + string(data) + "\n\n")
}
