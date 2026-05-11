package api

import (
	"errors"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"github.com/nimbleflux/fluxbase/internal/config"
	"github.com/nimbleflux/fluxbase/internal/mcp/custom"
	"github.com/nimbleflux/fluxbase/internal/middleware"
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

func (h *CustomMCPHandler) requireStorage(c fiber.Ctx) error {
	if h.storage == nil {
		return fiber.NewError(fiber.StatusServiceUnavailable, "Storage service not initialized")
	}
	return nil
}

func (h *CustomMCPHandler) requireManager(c fiber.Ctx) error {
	if h.manager == nil {
		return fiber.NewError(fiber.StatusServiceUnavailable, "MCP manager not initialized")
	}
	return nil
}

// Configuration Handlers

// GetConfig returns the current MCP configuration.
func (h *CustomMCPHandler) GetConfig(c fiber.Ctx) error {
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
func (h *CustomMCPHandler) ListTools(c fiber.Ctx) error {
	filter := custom.ListToolsFilter{
		Namespace:   c.Query("namespace"),
		EnabledOnly: c.Query("enabled_only") == "true",
	}
	if filter.Namespace == "default" {
		filter.Namespace = ""
	}

	if limit := fiber.Query[int](c, "limit", 0); limit > 0 {
		filter.Limit = limit
	}
	if offset := fiber.Query[int](c, "offset", 0); offset > 0 {
		filter.Offset = offset
	}

	if err := h.requireStorage(c); err != nil {
		return err
	}

	tools, err := h.storage.ListTools(middleware.CtxWithTenant(c), filter)
	if err != nil {
		return SendInternalError(c, "Failed to list custom tools")
	}

	return c.JSON(fiber.Map{
		"tools": tools,
		"count": len(tools),
	})
}

// GetTool returns a custom MCP tool by ID.
func (h *CustomMCPHandler) GetTool(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return SendBadRequest(c, "Invalid tool ID", ErrCodeInvalidID)
	}

	if err := h.requireStorage(c); err != nil {
		return err
	}

	tool, err := h.storage.GetTool(middleware.CtxWithTenant(c), id)
	if err != nil {
		if errors.Is(err, custom.ErrToolNotFound) {
			return SendResourceNotFound(c, "Tool")
		}
		return SendInternalError(c, "Failed to get tool")
	}

	return c.JSON(tool)
}

// CreateTool creates a new custom MCP tool.
func (h *CustomMCPHandler) CreateTool(c fiber.Ctx) error {
	var req custom.CreateToolRequest
	if err := ParseBody(c, &req); err != nil {
		return err
	}

	// Validate required fields
	if req.Name == "" {
		return SendMissingField(c, "Name")
	}
	if req.Code == "" {
		return SendMissingField(c, "Code")
	}

	// Validate code
	if err := custom.ValidateToolCode(req.Code); err != nil {
		return SendBadRequest(c, "Invalid tool code: "+err.Error(), ErrCodeValidationFailed)
	}

	// Get user ID from context
	var createdBy *uuid.UUID
	if uid, err := uuid.Parse(middleware.GetUserID(c)); err == nil {
		createdBy = &uid
	}

	if err := h.requireStorage(c); err != nil {
		return err
	}

	tool, err := h.storage.CreateTool(middleware.CtxWithTenant(c), &req, createdBy)
	if err != nil {
		if errors.Is(err, custom.ErrToolAlreadyExists) {
			return SendConflict(c, "A tool with this name already exists in the namespace", ErrCodeAlreadyExists)
		}
		return SendInternalError(c, "Failed to create tool")
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
func (h *CustomMCPHandler) UpdateTool(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return SendBadRequest(c, "Invalid tool ID", ErrCodeInvalidID)
	}

	var req custom.UpdateToolRequest
	if err := ParseBody(c, &req); err != nil {
		return err
	}

	// Validate code if provided
	if req.Code != nil {
		if err := custom.ValidateToolCode(*req.Code); err != nil {
			return SendBadRequest(c, "Invalid tool code: "+err.Error(), ErrCodeValidationFailed)
		}
	}

	if err := h.requireStorage(c); err != nil {
		return err
	}

	tool, err := h.storage.UpdateTool(middleware.CtxWithTenant(c), id, &req)
	if err != nil {
		if errors.Is(err, custom.ErrToolNotFound) {
			return SendResourceNotFound(c, "Tool")
		}
		if errors.Is(err, custom.ErrToolAlreadyExists) {
			return SendConflict(c, "A tool with this name already exists in the namespace", ErrCodeAlreadyExists)
		}
		return SendInternalError(c, "Failed to update tool")
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
func (h *CustomMCPHandler) DeleteTool(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return SendBadRequest(c, "Invalid tool ID", ErrCodeInvalidID)
	}

	if err := h.requireStorage(c); err != nil {
		return err
	}

	tool, err := h.storage.GetTool(middleware.CtxWithTenant(c), id)
	if err != nil {
		if errors.Is(err, custom.ErrToolNotFound) {
			return SendResourceNotFound(c, "Tool")
		}
		return SendInternalError(c, "Failed to get tool")
	}

	if err := h.storage.DeleteTool(middleware.CtxWithTenant(c), id); err != nil {
		return SendInternalError(c, "Failed to delete tool")
	}

	// Unregister from MCP server
	if h.manager != nil {
		h.manager.UnregisterTool(tool.Name)
	}

	return c.Status(fiber.StatusNoContent).Send(nil)
}

// SyncTool creates or updates a tool by name (upsert).
func (h *CustomMCPHandler) SyncTool(c fiber.Ctx) error {
	var req custom.SyncToolRequest
	if err := ParseBody(c, &req); err != nil {
		return err
	}

	// Validate required fields
	if req.Name == "" {
		return SendMissingField(c, "Name")
	}
	if req.Code == "" {
		return SendMissingField(c, "Code")
	}

	// Validate code
	if err := custom.ValidateToolCode(req.Code); err != nil {
		return SendBadRequest(c, "Invalid tool code: "+err.Error(), ErrCodeValidationFailed)
	}

	// Default upsert to true for sync operation
	req.Upsert = true

	// Get user ID from context
	var createdBy *uuid.UUID
	if uid, err := uuid.Parse(middleware.GetUserID(c)); err == nil {
		createdBy = &uid
	}

	if err := h.requireStorage(c); err != nil {
		return err
	}

	tool, err := h.storage.SyncTool(middleware.CtxWithTenant(c), &req, createdBy)
	if err != nil {
		return SendInternalError(c, "Failed to sync tool")
	}

	// Register with MCP server
	if h.manager != nil && tool.Enabled {
		_ = h.manager.RegisterTool(tool)
	}

	return c.JSON(tool)
}

// TestTool tests a custom MCP tool execution.
func (h *CustomMCPHandler) TestTool(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return SendBadRequest(c, "Invalid tool ID", ErrCodeInvalidID)
	}

	var req struct {
		Args map[string]any `json:"args"`
	}
	if err := ParseBody(c, &req); err != nil {
		return err
	}

	if err := h.requireStorage(c); err != nil {
		return err
	}

	tool, err := h.storage.GetTool(middleware.CtxWithTenant(c), id)
	if err != nil {
		if errors.Is(err, custom.ErrToolNotFound) {
			return SendResourceNotFound(c, "Tool")
		}
		return SendInternalError(c, "Failed to get tool")
	}

	if err := h.requireManager(c); err != nil {
		return err
	}

	// Execute the tool (manager has the executor)
	// For testing, we'll create a simple auth context
	result, err := h.manager.ExecuteToolForTest(c.RequestCtx(), tool, req.Args)
	if err != nil {
		return SendErrorWithDetails(c, 500, "Tool execution failed", ErrCodeInternalError, "", "", result)
	}

	return c.JSON(fiber.Map{
		"success": !result.IsError,
		"result":  result,
	})
}

// Resource Handlers

// ListResources returns all custom MCP resources.
func (h *CustomMCPHandler) ListResources(c fiber.Ctx) error {
	filter := custom.ListResourcesFilter{
		Namespace:   c.Query("namespace"),
		EnabledOnly: c.Query("enabled_only") == "true",
	}
	if filter.Namespace == "default" {
		filter.Namespace = ""
	}

	if limit := fiber.Query[int](c, "limit", 0); limit > 0 {
		filter.Limit = limit
	}
	if offset := fiber.Query[int](c, "offset", 0); offset > 0 {
		filter.Offset = offset
	}

	if err := h.requireStorage(c); err != nil {
		return err
	}

	resources, err := h.storage.ListResources(middleware.CtxWithTenant(c), filter)
	if err != nil {
		return SendInternalError(c, "Failed to list custom resources")
	}

	return c.JSON(fiber.Map{
		"resources": resources,
		"count":     len(resources),
	})
}

// GetResource returns a custom MCP resource by ID.
func (h *CustomMCPHandler) GetResource(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return SendBadRequest(c, "Invalid resource ID", ErrCodeInvalidID)
	}

	if err := h.requireStorage(c); err != nil {
		return err
	}

	resource, err := h.storage.GetResource(middleware.CtxWithTenant(c), id)
	if err != nil {
		if errors.Is(err, custom.ErrResourceNotFound) {
			return SendResourceNotFound(c, "Resource")
		}
		return SendInternalError(c, "Failed to get resource")
	}

	return c.JSON(resource)
}

// CreateResource creates a new custom MCP resource.
func (h *CustomMCPHandler) CreateResource(c fiber.Ctx) error {
	var req custom.CreateResourceRequest
	if err := ParseBody(c, &req); err != nil {
		return err
	}

	// Validate required fields
	if req.URI == "" {
		return SendMissingField(c, "URI")
	}
	if req.Name == "" {
		return SendMissingField(c, "Name")
	}
	if req.Code == "" {
		return SendMissingField(c, "Code")
	}

	// Validate code
	if err := custom.ValidateResourceCode(req.Code); err != nil {
		return SendBadRequest(c, "Invalid resource code: "+err.Error(), ErrCodeValidationFailed)
	}

	// Get user ID from context
	var createdBy *uuid.UUID
	if uid, err := uuid.Parse(middleware.GetUserID(c)); err == nil {
		createdBy = &uid
	}

	if err := h.requireStorage(c); err != nil {
		return err
	}

	resource, err := h.storage.CreateResource(middleware.CtxWithTenant(c), &req, createdBy)
	if err != nil {
		if errors.Is(err, custom.ErrResourceExists) {
			return SendConflict(c, "A resource with this URI already exists in the namespace", ErrCodeAlreadyExists)
		}
		return SendInternalError(c, "Failed to create resource")
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
func (h *CustomMCPHandler) UpdateResource(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return SendBadRequest(c, "Invalid resource ID", ErrCodeInvalidID)
	}

	var req custom.UpdateResourceRequest
	if err := ParseBody(c, &req); err != nil {
		return err
	}

	// Validate code if provided
	if req.Code != nil {
		if err := custom.ValidateResourceCode(*req.Code); err != nil {
			return SendBadRequest(c, "Invalid resource code: "+err.Error(), ErrCodeValidationFailed)
		}
	}

	if err := h.requireStorage(c); err != nil {
		return err
	}

	resource, err := h.storage.UpdateResource(middleware.CtxWithTenant(c), id, &req)
	if err != nil {
		if errors.Is(err, custom.ErrResourceNotFound) {
			return SendResourceNotFound(c, "Resource")
		}
		if errors.Is(err, custom.ErrResourceExists) {
			return SendConflict(c, "A resource with this URI already exists in the namespace", ErrCodeAlreadyExists)
		}
		return SendInternalError(c, "Failed to update resource")
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
func (h *CustomMCPHandler) DeleteResource(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return SendBadRequest(c, "Invalid resource ID", ErrCodeInvalidID)
	}

	if err := h.requireStorage(c); err != nil {
		return err
	}

	resource, err := h.storage.GetResource(middleware.CtxWithTenant(c), id)
	if err != nil {
		if errors.Is(err, custom.ErrResourceNotFound) {
			return SendResourceNotFound(c, "Resource")
		}
		return SendInternalError(c, "Failed to get resource")
	}

	if err := h.storage.DeleteResource(middleware.CtxWithTenant(c), id); err != nil {
		return SendInternalError(c, "Failed to delete resource")
	}

	// Unregister from MCP server
	if h.manager != nil {
		h.manager.UnregisterResource(resource.URI)
	}

	return c.Status(fiber.StatusNoContent).Send(nil)
}

// SyncResource creates or updates a resource by URI (upsert).
func (h *CustomMCPHandler) SyncResource(c fiber.Ctx) error {
	var req custom.SyncResourceRequest
	if err := ParseBody(c, &req); err != nil {
		return err
	}

	// Validate required fields
	if req.URI == "" {
		return SendMissingField(c, "URI")
	}
	if req.Name == "" {
		return SendMissingField(c, "Name")
	}
	if req.Code == "" {
		return SendMissingField(c, "Code")
	}

	// Validate code
	if err := custom.ValidateResourceCode(req.Code); err != nil {
		return SendBadRequest(c, "Invalid resource code: "+err.Error(), ErrCodeValidationFailed)
	}

	// Default upsert to true for sync operation
	req.Upsert = true

	// Get user ID from context
	var createdBy *uuid.UUID
	if uid, err := uuid.Parse(middleware.GetUserID(c)); err == nil {
		createdBy = &uid
	}

	if err := h.requireStorage(c); err != nil {
		return err
	}

	resource, err := h.storage.SyncResource(middleware.CtxWithTenant(c), &req, createdBy)
	if err != nil {
		return SendInternalError(c, "Failed to sync resource")
	}

	// Register with MCP server
	if h.manager != nil && resource.Enabled {
		_ = h.manager.RegisterResource(resource)
	}

	return c.JSON(resource)
}

// TestResource tests a custom MCP resource read.
func (h *CustomMCPHandler) TestResource(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return SendBadRequest(c, "Invalid resource ID", ErrCodeInvalidID)
	}

	var req struct {
		Params map[string]string `json:"params"`
	}
	if err := ParseBody(c, &req); err != nil {
		return err
	}

	if err := h.requireStorage(c); err != nil {
		return err
	}

	resource, err := h.storage.GetResource(middleware.CtxWithTenant(c), id)
	if err != nil {
		if errors.Is(err, custom.ErrResourceNotFound) {
			return SendResourceNotFound(c, "Resource")
		}
		return SendInternalError(c, "Failed to get resource")
	}

	if err := h.requireManager(c); err != nil {
		return err
	}

	contents, err := h.manager.ExecuteResourceForTest(c.RequestCtx(), resource, req.Params)
	if err != nil {
		return SendInternalError(c, "Resource read failed")
	}

	return c.JSON(fiber.Map{
		"success":  true,
		"contents": contents,
	})
}

// fiber:context-methods migrated
