package api

import (
	"errors"
	"fmt"

	"github.com/fluxbase-eu/fluxbase/internal/auth"
	"github.com/fluxbase-eu/fluxbase/internal/config"
	"github.com/fluxbase-eu/fluxbase/internal/mcp/custom"
	"github.com/fluxbase-eu/fluxbase/internal/middleware"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CustomMCPHandler handles custom MCP tool and resource management requests.
type CustomMCPHandler struct {
	storage   *custom.Storage
	manager   *custom.Manager
	mcpConfig *config.MCPConfig
}

// NewCustomMCPHandler creates a new custom MCP handler.
func NewCustomMCPHandler(storage *custom.Storage, manager *custom.Manager, mcpConfig *config.MCPConfig) *CustomMCPHandler {
	return &CustomMCPHandler{
		storage:   storage,
		manager:   manager,
		mcpConfig: mcpConfig,
	}
}

// RegisterRoutes registers custom MCP routes.
func (h *CustomMCPHandler) RegisterRoutes(
	app *fiber.App,
	authService *auth.Service,
	clientKeyService *auth.ClientKeyService,
	db *pgxpool.Pool,
	jwtManager *auth.JWTManager,
) {
	// Custom MCP tools and resources require admin access
	mcpAdmin := app.Group("/api/v1/mcp",
		middleware.RequireAuthOrServiceKey(authService, clientKeyService, db, jwtManager),
		middleware.RequireAdmin(),
	)

	// MCP Configuration
	mcpAdmin.Get("/config", h.GetConfig)

	// Custom Tools CRUD
	// Note: Static routes (/sync) must be registered before parameterized routes (/:id)
	// to ensure correct route matching
	mcpAdmin.Get("/tools", h.ListTools)
	mcpAdmin.Post("/tools", h.CreateTool)
	mcpAdmin.Post("/tools/sync", h.SyncTool)
	mcpAdmin.Get("/tools/:id", h.GetTool)
	mcpAdmin.Put("/tools/:id", h.UpdateTool)
	mcpAdmin.Delete("/tools/:id", h.DeleteTool)
	mcpAdmin.Post("/tools/:id/test", h.TestTool)

	// Custom Resources CRUD
	// Note: Static routes (/sync) must be registered before parameterized routes (/:id)
	mcpAdmin.Get("/resources", h.ListResources)
	mcpAdmin.Post("/resources", h.CreateResource)
	mcpAdmin.Post("/resources/sync", h.SyncResource)
	mcpAdmin.Get("/resources/:id", h.GetResource)
	mcpAdmin.Put("/resources/:id", h.UpdateResource)
	mcpAdmin.Delete("/resources/:id", h.DeleteResource)
	mcpAdmin.Post("/resources/:id/test", h.TestResource)
}

// Configuration Handlers

// GetConfig returns the current MCP configuration.
func (h *CustomMCPHandler) GetConfig(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"enabled":            h.mcpConfig.Enabled,
		"base_path":          h.mcpConfig.BasePath,
		"tools_dir":          h.mcpConfig.ToolsDir,
		"auto_load_on_boot":  h.mcpConfig.AutoLoadOnBoot,
		"rate_limit_per_min": h.mcpConfig.RateLimitPerMin,
	})
}

// Tool Handlers

// ListTools returns all custom MCP tools.
func (h *CustomMCPHandler) ListTools(c *fiber.Ctx) error {
	filter := custom.ListToolsFilter{
		Namespace:   c.Query("namespace"),
		EnabledOnly: c.Query("enabled_only") == "true",
	}

	if limit := c.QueryInt("limit", 0); limit > 0 {
		filter.Limit = limit
	}
	if offset := c.QueryInt("offset", 0); offset > 0 {
		filter.Offset = offset
	}

	tools, err := h.storage.ListTools(c.Context(), filter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to list custom tools: %v", err),
		})
	}

	return c.JSON(fiber.Map{
		"tools": tools,
		"count": len(tools),
	})
}

// GetTool returns a custom MCP tool by ID.
func (h *CustomMCPHandler) GetTool(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid tool ID",
		})
	}

	tool, err := h.storage.GetTool(c.Context(), id)
	if err != nil {
		if errors.Is(err, custom.ErrToolNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Tool not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to get tool: %v", err),
		})
	}

	return c.JSON(tool)
}

// CreateTool creates a new custom MCP tool.
func (h *CustomMCPHandler) CreateTool(c *fiber.Ctx) error {
	var req custom.CreateToolRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate required fields
	if req.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Name is required",
		})
	}
	if req.Code == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Code is required",
		})
	}

	// Validate code
	if err := custom.ValidateToolCode(req.Code); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("Invalid tool code: %v", err),
		})
	}

	// Get user ID from context
	var createdBy *uuid.UUID
	if userID, ok := c.Locals("user_id").(uuid.UUID); ok {
		createdBy = &userID
	}

	tool, err := h.storage.CreateTool(c.Context(), &req, createdBy)
	if err != nil {
		if errors.Is(err, custom.ErrToolAlreadyExists) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": "A tool with this name already exists in the namespace",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to create tool: %v", err),
		})
	}

	// Register with MCP server
	if h.manager != nil {
		if err := h.manager.RegisterTool(tool); err != nil {
			// Log but don't fail - tool is created, just not registered yet
			c.Set("X-MCP-Registration-Warning", err.Error())
		}
	}

	return c.Status(fiber.StatusCreated).JSON(tool)
}

// UpdateTool updates an existing custom MCP tool.
func (h *CustomMCPHandler) UpdateTool(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid tool ID",
		})
	}

	var req custom.UpdateToolRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate code if provided
	if req.Code != nil {
		if err := custom.ValidateToolCode(*req.Code); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": fmt.Sprintf("Invalid tool code: %v", err),
			})
		}
	}

	tool, err := h.storage.UpdateTool(c.Context(), id, &req)
	if err != nil {
		if errors.Is(err, custom.ErrToolNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Tool not found",
			})
		}
		if errors.Is(err, custom.ErrToolAlreadyExists) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": "A tool with this name already exists in the namespace",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to update tool: %v", err),
		})
	}

	// Re-register with MCP server
	if h.manager != nil {
		if tool.Enabled {
			_ = h.manager.RegisterTool(tool)
		} else {
			h.manager.UnregisterTool(tool.Name)
		}
	}

	return c.JSON(tool)
}

// DeleteTool deletes a custom MCP tool.
func (h *CustomMCPHandler) DeleteTool(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid tool ID",
		})
	}

	// Get tool first to get name for unregistering
	tool, err := h.storage.GetTool(c.Context(), id)
	if err != nil {
		if errors.Is(err, custom.ErrToolNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Tool not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to get tool: %v", err),
		})
	}

	if err := h.storage.DeleteTool(c.Context(), id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to delete tool: %v", err),
		})
	}

	// Unregister from MCP server
	if h.manager != nil {
		h.manager.UnregisterTool(tool.Name)
	}

	return c.Status(fiber.StatusNoContent).Send(nil)
}

// SyncTool creates or updates a tool by name (upsert).
func (h *CustomMCPHandler) SyncTool(c *fiber.Ctx) error {
	var req custom.SyncToolRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate required fields
	if req.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Name is required",
		})
	}
	if req.Code == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Code is required",
		})
	}

	// Validate code
	if err := custom.ValidateToolCode(req.Code); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("Invalid tool code: %v", err),
		})
	}

	// Default upsert to true for sync operation
	req.Upsert = true

	// Get user ID from context
	var createdBy *uuid.UUID
	if userID, ok := c.Locals("user_id").(uuid.UUID); ok {
		createdBy = &userID
	}

	tool, err := h.storage.SyncTool(c.Context(), &req, createdBy)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to sync tool: %v", err),
		})
	}

	// Register with MCP server
	if h.manager != nil && tool.Enabled {
		_ = h.manager.RegisterTool(tool)
	}

	return c.JSON(tool)
}

// TestTool tests a custom MCP tool execution.
func (h *CustomMCPHandler) TestTool(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid tool ID",
		})
	}

	var req struct {
		Args map[string]any `json:"args"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	tool, err := h.storage.GetTool(c.Context(), id)
	if err != nil {
		if errors.Is(err, custom.ErrToolNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Tool not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to get tool: %v", err),
		})
	}

	if h.manager == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "MCP manager not initialized",
		})
	}

	// Execute the tool (manager has the executor)
	// For testing, we'll create a simple auth context
	result, err := h.manager.ExecuteToolForTest(c.Context(), tool, req.Args)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":  fmt.Sprintf("Tool execution failed: %v", err),
			"result": result,
		})
	}

	return c.JSON(fiber.Map{
		"success": !result.IsError,
		"result":  result,
	})
}

// Resource Handlers

// ListResources returns all custom MCP resources.
func (h *CustomMCPHandler) ListResources(c *fiber.Ctx) error {
	filter := custom.ListResourcesFilter{
		Namespace:   c.Query("namespace"),
		EnabledOnly: c.Query("enabled_only") == "true",
	}

	if limit := c.QueryInt("limit", 0); limit > 0 {
		filter.Limit = limit
	}
	if offset := c.QueryInt("offset", 0); offset > 0 {
		filter.Offset = offset
	}

	resources, err := h.storage.ListResources(c.Context(), filter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to list custom resources: %v", err),
		})
	}

	return c.JSON(fiber.Map{
		"resources": resources,
		"count":     len(resources),
	})
}

// GetResource returns a custom MCP resource by ID.
func (h *CustomMCPHandler) GetResource(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid resource ID",
		})
	}

	resource, err := h.storage.GetResource(c.Context(), id)
	if err != nil {
		if errors.Is(err, custom.ErrResourceNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Resource not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to get resource: %v", err),
		})
	}

	return c.JSON(resource)
}

// CreateResource creates a new custom MCP resource.
func (h *CustomMCPHandler) CreateResource(c *fiber.Ctx) error {
	var req custom.CreateResourceRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate required fields
	if req.URI == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "URI is required",
		})
	}
	if req.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Name is required",
		})
	}
	if req.Code == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Code is required",
		})
	}

	// Validate code
	if err := custom.ValidateResourceCode(req.Code); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("Invalid resource code: %v", err),
		})
	}

	// Get user ID from context
	var createdBy *uuid.UUID
	if userID, ok := c.Locals("user_id").(uuid.UUID); ok {
		createdBy = &userID
	}

	resource, err := h.storage.CreateResource(c.Context(), &req, createdBy)
	if err != nil {
		if errors.Is(err, custom.ErrResourceExists) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": "A resource with this URI already exists in the namespace",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to create resource: %v", err),
		})
	}

	// Register with MCP server
	if h.manager != nil {
		if err := h.manager.RegisterResource(resource); err != nil {
			c.Set("X-MCP-Registration-Warning", err.Error())
		}
	}

	return c.Status(fiber.StatusCreated).JSON(resource)
}

// UpdateResource updates an existing custom MCP resource.
func (h *CustomMCPHandler) UpdateResource(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid resource ID",
		})
	}

	var req custom.UpdateResourceRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate code if provided
	if req.Code != nil {
		if err := custom.ValidateResourceCode(*req.Code); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": fmt.Sprintf("Invalid resource code: %v", err),
			})
		}
	}

	resource, err := h.storage.UpdateResource(c.Context(), id, &req)
	if err != nil {
		if errors.Is(err, custom.ErrResourceNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Resource not found",
			})
		}
		if errors.Is(err, custom.ErrResourceExists) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": "A resource with this URI already exists in the namespace",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to update resource: %v", err),
		})
	}

	// Re-register with MCP server
	if h.manager != nil {
		if resource.Enabled {
			_ = h.manager.RegisterResource(resource)
		} else {
			h.manager.UnregisterResource(resource.URI)
		}
	}

	return c.JSON(resource)
}

// DeleteResource deletes a custom MCP resource.
func (h *CustomMCPHandler) DeleteResource(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid resource ID",
		})
	}

	// Get resource first to get URI for unregistering
	resource, err := h.storage.GetResource(c.Context(), id)
	if err != nil {
		if errors.Is(err, custom.ErrResourceNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Resource not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to get resource: %v", err),
		})
	}

	if err := h.storage.DeleteResource(c.Context(), id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to delete resource: %v", err),
		})
	}

	// Unregister from MCP server
	if h.manager != nil {
		h.manager.UnregisterResource(resource.URI)
	}

	return c.Status(fiber.StatusNoContent).Send(nil)
}

// SyncResource creates or updates a resource by URI (upsert).
func (h *CustomMCPHandler) SyncResource(c *fiber.Ctx) error {
	var req custom.SyncResourceRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate required fields
	if req.URI == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "URI is required",
		})
	}
	if req.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Name is required",
		})
	}
	if req.Code == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Code is required",
		})
	}

	// Validate code
	if err := custom.ValidateResourceCode(req.Code); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("Invalid resource code: %v", err),
		})
	}

	// Default upsert to true for sync operation
	req.Upsert = true

	// Get user ID from context
	var createdBy *uuid.UUID
	if userID, ok := c.Locals("user_id").(uuid.UUID); ok {
		createdBy = &userID
	}

	resource, err := h.storage.SyncResource(c.Context(), &req, createdBy)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to sync resource: %v", err),
		})
	}

	// Register with MCP server
	if h.manager != nil && resource.Enabled {
		_ = h.manager.RegisterResource(resource)
	}

	return c.JSON(resource)
}

// TestResource tests a custom MCP resource read.
func (h *CustomMCPHandler) TestResource(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid resource ID",
		})
	}

	var req struct {
		Params map[string]string `json:"params"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	resource, err := h.storage.GetResource(c.Context(), id)
	if err != nil {
		if errors.Is(err, custom.ErrResourceNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Resource not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to get resource: %v", err),
		})
	}

	if h.manager == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "MCP manager not initialized",
		})
	}

	// Execute the resource
	contents, err := h.manager.ExecuteResourceForTest(c.Context(), resource, req.Params)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Resource read failed: %v", err),
		})
	}

	return c.JSON(fiber.Map{
		"success":  true,
		"contents": contents,
	})
}
