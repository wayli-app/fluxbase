package ai

import (
	"fmt"
	"path/filepath"

	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/middleware"
)

// ListMyDocuments lists documents in a KB (requires viewer permission)
// GET /api/v1/ai/knowledge-bases/:id/documents
func (h *UserKnowledgeBaseHandler) ListMyDocuments(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)
	userID := middleware.GetUserID(c)
	kbID := c.Params("id")

	// Check read permission (viewer or higher)
	hasPermission, err := h.storage.CheckKBPermission(ctx, kbID, userID, string(KBPermissionViewer))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to check permission",
		})
	}
	if !hasPermission {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Access denied",
		})
	}

	// Get documents (the storage layer will filter by user's access)
	documents, err := h.storage.ListDocuments(ctx, kbID)
	if err != nil {
		log.Error().Err(err).Str("kb_id", kbID).Msg("Failed to list documents")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to list documents",
		})
	}

	return c.JSON(fiber.Map{
		"documents": documents,
		"count":     len(documents),
	})
}

// GetMyDocument gets a specific document (requires viewer permission)
// GET /api/v1/ai/knowledge-bases/:id/documents/:doc_id
func (h *UserKnowledgeBaseHandler) GetMyDocument(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)
	userID := middleware.GetUserID(c)
	kbID := c.Params("id")
	docID := c.Params("doc_id")

	// Check read permission
	hasPermission, err := h.storage.CheckKBPermission(ctx, kbID, userID, string(KBPermissionViewer))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to check permission",
		})
	}
	if !hasPermission {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Access denied",
		})
	}

	doc, err := h.storage.GetDocument(ctx, docID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Document not found",
		})
	}

	// Verify document belongs to the KB
	if doc.KnowledgeBaseID != kbID {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Document not found",
		})
	}

	return c.JSON(doc)
}

// AddMyDocument adds a document to a KB (requires editor permission)
// POST /api/v1/ai/knowledge-bases/:id/documents
func (h *UserKnowledgeBaseHandler) AddMyDocument(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)
	userID := middleware.GetUserID(c)
	kbID := c.Params("id")

	// Check write permission (editor or higher)
	hasPermission, err := h.storage.CheckKBPermission(ctx, kbID, userID, string(KBPermissionEditor))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to check permission",
		})
	}
	if !hasPermission {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Editor permission required to add documents",
		})
	}

	// Check if processor is available
	if h.processor == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "Document processing not available (embedding service not configured)",
		})
	}

	var req AddDocumentRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.Content == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Content is required",
		})
	}

	// Auto-set user_id in metadata for user isolation
	metadata := req.Metadata
	if metadata == nil {
		metadata = make(map[string]string)
	}
	metadata["user_id"] = userID

	// Add document
	docReq := CreateDocumentRequest{
		Title:     req.Title,
		Content:   req.Content,
		SourceURL: req.Source,
		MimeType:  req.MimeType,
		Metadata:  metadata,
	}

	doc, err := h.processor.AddDocument(ctx, kbID, docReq, &userID)
	if err != nil {
		log.Error().Err(err).Str("kb_id", kbID).Msg("Failed to add document")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to add document",
		})
	}

	return c.Status(fiber.StatusAccepted).JSON(fiber.Map{
		"document_id": doc.ID,
		"status":      "processing",
		"message":     "Document is being processed and will be available shortly",
	})
}

// UploadMyDocument uploads a file to a KB (requires editor permission)
// POST /api/v1/ai/knowledge-bases/:id/documents/upload
func (h *UserKnowledgeBaseHandler) UploadMyDocument(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)
	userID := middleware.GetUserID(c)
	kbID := c.Params("id")

	// Check write permission (editor or higher)
	hasPermission, err := h.storage.CheckKBPermission(ctx, kbID, userID, string(KBPermissionEditor))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to check permission",
		})
	}
	if !hasPermission {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Editor permission required to upload documents",
		})
	}

	// Check if processor is available
	if h.processor == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "Document processing not available (embedding service not configured)",
		})
	}

	// Get the uploaded file
	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "No file uploaded",
		})
	}

	// Check file size (max 50MB)
	maxSize := int64(50 * 1024 * 1024)
	if file.Size > maxSize {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("File too large. Maximum size is %dMB", maxSize/(1024*1024)),
		})
	}

	// Determine MIME type from file extension
	ext := filepath.Ext(file.Filename)
	mimeType := GetMimeTypeFromExtension(ext)

	// Check if MIME type is supported
	supported := h.textExtractor.SupportedMimeTypes()
	isSupported := false
	for _, s := range supported {
		if s == mimeType {
			isSupported = true
			break
		}
	}
	if !isSupported {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":           fmt.Sprintf("Unsupported file type: %s", ext),
			"supported_types": supported,
		})
	}

	// Read file content
	fileReader, err := file.Open()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to read uploaded file",
		})
	}
	defer func() { _ = fileReader.Close() }()

	fileContent, err := readFileContent(fileReader, int(file.Size))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to read file content",
		})
	}

	// Extract text from file
	extractedText, err := h.textExtractor.Extract(fileContent, mimeType)
	if err != nil {
		log.Error().Err(err).Str("mime_type", mimeType).Msg("Failed to extract text from file")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to extract text from file: %v", err),
		})
	}

	// Prepare metadata with user isolation
	metadata := map[string]string{"user_id": userID}

	// Create document request
	docReq := CreateDocumentRequest{
		Title:    file.Filename,
		Content:  extractedText,
		MimeType: mimeType,
		Metadata: metadata,
	}

	// Add document
	doc, err := h.processor.AddDocument(ctx, kbID, docReq, &userID)
	if err != nil {
		log.Error().Err(err).Str("kb_id", kbID).Msg("Failed to add document from upload")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to add document",
		})
	}

	return c.Status(fiber.StatusAccepted).JSON(fiber.Map{
		"document_id": doc.ID,
		"status":      "processing",
		"message":     "Document is being processed and will be available shortly",
	})
}

// DeleteMyDocument deletes a document from a KB (requires editor permission)
// DELETE /api/v1/ai/knowledge-bases/:id/documents/:doc_id
func (h *UserKnowledgeBaseHandler) DeleteMyDocument(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)
	userID := middleware.GetUserID(c)
	kbID := c.Params("id")
	docID := c.Params("doc_id")

	// Check write permission (editor or higher)
	hasPermission, err := h.storage.CheckKBPermission(ctx, kbID, userID, string(KBPermissionEditor))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to check permission",
		})
	}
	if !hasPermission {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Editor permission required to delete documents",
		})
	}

	// Get document to verify it belongs to this KB
	doc, err := h.storage.GetDocument(ctx, docID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Document not found",
		})
	}
	if doc.KnowledgeBaseID != kbID {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Document not found",
		})
	}

	// Delete document
	if err := h.storage.DeleteDocument(ctx, docID); err != nil {
		log.Error().Err(err).Str("doc_id", docID).Msg("Failed to delete document")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete document",
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// UpdateMyDocument updates a document's metadata
// PATCH /api/v1/ai/knowledge-bases/:id/documents/:doc_id
func (h *UserKnowledgeBaseHandler) UpdateMyDocument(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)
	userID := middleware.GetUserID(c)
	kbID := c.Params("id")
	docID := c.Params("doc_id")

	// Check editor permission
	hasPermission, err := h.storage.CheckKBPermission(ctx, kbID, userID, string(KBPermissionEditor))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to check permission",
		})
	}
	if !hasPermission {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Editor permission required",
		})
	}

	var req struct {
		Title    *string           `json:"title,omitempty"`
		Metadata map[string]string `json:"metadata,omitempty"`
		Tags     []string          `json:"tags,omitempty"`
	}
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Get existing document
	doc, err := h.storage.GetDocument(ctx, docID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Document not found",
		})
	}

	// Verify document belongs to KB
	if doc.KnowledgeBaseID != kbID {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Document not found",
		})
	}

	// Use UpdateDocumentMetadata for updating
	updatedDoc, err := h.storage.UpdateDocumentMetadata(ctx, docID, req.Title, req.Metadata, req.Tags)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update document",
		})
	}

	return c.JSON(updatedDoc)
}

// DeleteMyDocumentsByFilter deletes documents matching a filter
// POST /api/v1/ai/knowledge-bases/:id/documents/delete-by-filter
func (h *UserKnowledgeBaseHandler) DeleteMyDocumentsByFilter(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)
	userID := middleware.GetUserID(c)
	kbID := c.Params("id")

	// Check editor permission
	hasPermission, err := h.storage.CheckKBPermission(ctx, kbID, userID, string(KBPermissionEditor))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to check permission",
		})
	}
	if !hasPermission {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Editor permission required",
		})
	}

	var req struct {
		Tags     []string          `json:"tags,omitempty"`
		Metadata map[string]string `json:"metadata,omitempty"`
	}
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	filter := &MetadataFilter{
		Tags:     req.Tags,
		Metadata: req.Metadata,
	}

	deletedCount, err := h.storage.DeleteDocumentsByFilter(ctx, kbID, filter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete documents",
		})
	}

	return c.JSON(fiber.Map{
		"deleted_count": deletedCount,
	})
}

// readFileContent reads file content from reader with size limit
func readFileContent(reader interface{ Read([]byte) (int, error) }, maxSize int) ([]byte, error) {
	size := maxSize
	if size > 50*1024*1024 {
		size = 50 * 1024 * 1024 // Cap at 50MB
	}
	buf := make([]byte, 0, size)
	tmp := make([]byte, 1024)
	for {
		n, err := reader.Read(tmp)
		if err != nil {
			break
		}
		buf = append(buf, tmp[:n]...)
		if len(buf) > size {
			return nil, fmt.Errorf("file too large")
		}
	}
	return buf, nil
}
