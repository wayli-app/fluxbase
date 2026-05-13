package ai

import (
	"github.com/gofiber/fiber/v3"

	"github.com/nimbleflux/fluxbase/internal/middleware"
	"github.com/nimbleflux/fluxbase/internal/storage"
)

// UserKnowledgeBaseHandler handles user-facing KB endpoints
type UserKnowledgeBaseHandler struct {
	storage        *KnowledgeBaseStorage
	knowledgeGraph *KnowledgeGraph
	processor      *DocumentProcessor
	storageService *storage.Service
	textExtractor  *TextExtractor
}

// NewUserKnowledgeBaseHandler creates a new user KB handler
func NewUserKnowledgeBaseHandler(storage *KnowledgeBaseStorage) *UserKnowledgeBaseHandler {
	return &UserKnowledgeBaseHandler{
		storage:       storage,
		textExtractor: NewTextExtractor(),
	}
}

// NewUserKnowledgeBaseHandlerWithProcessor creates a handler with document processing support
func NewUserKnowledgeBaseHandlerWithProcessor(storage *KnowledgeBaseStorage, processor *DocumentProcessor) *UserKnowledgeBaseHandler {
	return &UserKnowledgeBaseHandler{
		storage:       storage,
		processor:     processor,
		textExtractor: NewTextExtractor(),
	}
}

// NewUserKnowledgeBaseHandlerWithGraph creates a handler with knowledge graph support
func NewUserKnowledgeBaseHandlerWithGraph(storage *KnowledgeBaseStorage, kg *KnowledgeGraph) *UserKnowledgeBaseHandler {
	return &UserKnowledgeBaseHandler{
		storage:        storage,
		knowledgeGraph: kg,
		textExtractor:  NewTextExtractor(),
	}
}

// NewUserKnowledgeBaseHandlerWithProcessorAndGraph creates a handler with both processor and graph support
func NewUserKnowledgeBaseHandlerWithProcessorAndGraph(storage *KnowledgeBaseStorage, processor *DocumentProcessor, kg *KnowledgeGraph) *UserKnowledgeBaseHandler {
	return &UserKnowledgeBaseHandler{
		storage:        storage,
		knowledgeGraph: kg,
		processor:      processor,
		textExtractor:  NewTextExtractor(),
	}
}

// SetStorageService sets the storage service for file uploads
func (h *UserKnowledgeBaseHandler) SetStorageService(svc *storage.Service) {
	h.storageService = svc
}

// ListMyKnowledgeBases returns KBs accessible to current user
// GET /api/v1/ai/knowledge-bases
func (h *UserKnowledgeBaseHandler) ListMyKnowledgeBases(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)

	// Safely check if user_id exists in context
	userID := middleware.GetUserID(c)
	if userID == "" {
		userRole := c.Locals("user_role")
		if userRole == "instance_admin" || userRole == "service_role" || userRole == "tenant_service" {
			return c.JSON(fiber.Map{
				"knowledge_bases": []interface{}{},
				"count":           0,
				"message":         "Select a tenant to view knowledge bases",
			})
		}
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "User not authenticated",
		})
	}

	kbs, err := h.storage.ListUserKnowledgeBases(ctx, userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to list knowledge bases",
		})
	}

	return c.JSON(fiber.Map{
		"knowledge_bases": kbs,
		"count":           len(kbs),
	})
}

// CreateMyKnowledgeBase creates a user-owned KB
// POST /api/v1/ai/knowledge-bases
func (h *UserKnowledgeBaseHandler) CreateMyKnowledgeBase(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)
	userID := middleware.GetUserID(c)

	var req CreateKnowledgeBaseRequest
	if err := c.Bind().Body(&req); err != nil {
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

	// Set owner on request so it's included in the INSERT
	req.OwnerID = &userID

	// Create KB using the shared method (handles defaults including embedding model)
	kb, err := h.storage.CreateKnowledgeBaseFromRequest(ctx, req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create knowledge base",
		})
	}

	// Grant initial permissions if specified
	for _, perm := range req.InitialPermissions {
		_, err := h.storage.GrantKBPermission(ctx, kb.ID, perm.UserID, string(perm.Permission), &userID)
		if err != nil {
			// Log error but don't fail the entire request
			// The KB was created successfully, just permission grant failed
			continue
		}
	}

	return c.Status(fiber.StatusCreated).JSON(kb)
}

// GetMyKnowledgeBase returns a specific KB if user has access
// GET /api/v1/ai/knowledge-bases/:id
func (h *UserKnowledgeBaseHandler) GetMyKnowledgeBase(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)
	userID := middleware.GetUserID(c)
	kbID := c.Params("id")

	if !h.storage.CanUserAccessKB(ctx, kbID, userID) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Access denied",
		})
	}

	kb, err := h.storage.GetKnowledgeBase(ctx, kbID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Knowledge base not found",
		})
	}

	return c.JSON(kb)
}

// ShareKnowledgeBase grants permission to another user
// POST /api/v1/ai/knowledge-bases/:id/share
func (h *UserKnowledgeBaseHandler) ShareKnowledgeBase(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)
	userID := middleware.GetUserID(c)
	kbID := c.Params("id")

	kb, err := h.storage.GetKnowledgeBase(ctx, kbID)
	if err != nil || kb.OwnerID == nil || *kb.OwnerID != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Only owner can share knowledge base",
		})
	}

	var req struct {
		UserID     string `json:"user_id"`
		Permission string `json:"permission"`
	}
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	grant, err := h.storage.GrantKBPermission(ctx, kbID, req.UserID, req.Permission, &userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to grant permission",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(grant)
}

// ListPermissions lists permissions for a KB
// GET /api/v1/ai/knowledge-bases/:id/permissions
func (h *UserKnowledgeBaseHandler) ListPermissions(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)
	userID := middleware.GetUserID(c)
	kbID := c.Params("id")

	kb, err := h.storage.GetKnowledgeBase(ctx, kbID)
	if err != nil || kb.OwnerID == nil || *kb.OwnerID != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Only owner can view permissions",
		})
	}

	perms, err := h.storage.ListKBPermissions(ctx, kbID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to list permissions",
		})
	}

	return c.JSON(perms)
}

// RevokePermission revokes a permission
// DELETE /api/v1/ai/knowledge-bases/:id/permissions/:user_id
func (h *UserKnowledgeBaseHandler) RevokePermission(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)
	userID := middleware.GetUserID(c)
	kbID := c.Params("id")
	targetUserID := c.Params("user_id")

	kb, err := h.storage.GetKnowledgeBase(ctx, kbID)
	if err != nil || kb.OwnerID == nil || *kb.OwnerID != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Only owner can revoke permissions",
		})
	}

	err = h.storage.RevokeKBPermission(ctx, kbID, targetUserID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to revoke permission",
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}
