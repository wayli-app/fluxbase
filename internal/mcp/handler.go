package mcp

import (
	"strings"

	"github.com/fluxbase-eu/fluxbase/internal/config"
	"github.com/fluxbase-eu/fluxbase/internal/database"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
)

// Handler handles HTTP requests for the MCP server
type Handler struct {
	server *Server
	config *config.MCPConfig
	db     *database.Connection
}

// NewHandler creates a new MCP HTTP handler
func NewHandler(cfg *config.MCPConfig, db *database.Connection) *Handler {
	return &Handler{
		server: NewServer(cfg),
		config: cfg,
		db:     db,
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
