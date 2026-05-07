package ai

import (
	"github.com/gofiber/fiber/v3"

	"github.com/nimbleflux/fluxbase/internal/middleware"
)

// ============================================================================
// DOCUMENT PERMISSION ENDPOINTS
// ============================================================================

// GrantDocumentPermission grants permission on a document to a user
// POST /api/v1/admin/ai/knowledge-bases/:kb_id/documents/:doc_id/permissions
func (h *KnowledgeBaseHandler) GrantDocumentPermission(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)
	userID := middleware.GetUserID(c)
	_ = c.Params("kb_id") // kb_id is part of the route but not used directly
	docID := c.Params("doc_id")

	var req GrantDocumentPermissionRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	grant, err := h.storage.GrantDocumentPermission(ctx, docID, req.UserID, string(req.Permission), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to grant permission: " + err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(grant)
}

// ListDocumentPermissions lists permissions for a document
// GET /api/v1/admin/ai/knowledge-bases/:kb_id/documents/:doc_id/permissions
func (h *KnowledgeBaseHandler) ListDocumentPermissions(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)
	docID := c.Params("doc_id")

	perms, err := h.storage.ListDocumentPermissions(ctx, docID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to list permissions",
		})
	}

	return c.JSON(perms)
}

// RevokeDocumentPermission revokes permission from a user on a document
// DELETE /api/v1/admin/ai/knowledge-bases/:kb_id/documents/:doc_id/permissions/:user_id
func (h *KnowledgeBaseHandler) RevokeDocumentPermission(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)
	docID := c.Params("doc_id")
	targetUserID := c.Params("user_id")

	err := h.storage.RevokeDocumentPermission(ctx, docID, targetUserID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to revoke permission",
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// SetTableExporter sets the table exporter for database schema export
func (h *KnowledgeBaseHandler) SetTableExporter(exporter *TableExporter) {
	h.tableExporter = exporter
}

// SetSyncService sets the table export sync service
func (h *KnowledgeBaseHandler) SetSyncService(svc *TableExportSyncService) {
	h.syncService = svc
}
