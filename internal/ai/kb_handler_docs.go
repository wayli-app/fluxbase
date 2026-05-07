package ai

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/middleware"
	"github.com/nimbleflux/fluxbase/internal/storage"
)

// ============================================================================
// DOCUMENT ENDPOINTS
// ============================================================================

// ListDocuments returns all documents in a knowledge base
// GET /api/v1/admin/ai/knowledge-bases/:id/documents
func (h *KnowledgeBaseHandler) ListDocuments(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)
	kbID := c.Params("id")

	if kbID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Knowledge base ID is required",
		})
	}

	docs, err := h.storage.ListDocuments(ctx, kbID)
	if err != nil {
		log.Error().Err(err).Str("kb_id", kbID).Msg("Failed to list documents")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to list documents",
		})
	}

	return c.JSON(fiber.Map{
		"documents": docs,
		"count":     len(docs),
	})
}

// GetDocument returns a specific document
// GET /api/v1/admin/ai/knowledge-bases/:id/documents/:doc_id
func (h *KnowledgeBaseHandler) GetDocument(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)
	docID := c.Params("doc_id")

	if docID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Document ID is required",
		})
	}

	doc, err := h.storage.GetDocument(ctx, docID)
	if err != nil {
		log.Error().Err(err).Str("doc_id", docID).Msg("Failed to get document")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get document",
		})
	}
	if doc == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Document not found",
		})
	}

	return c.JSON(doc)
}

// AddDocumentRequest represents a request to add a document
type AddDocumentRequest struct {
	Title    string            `json:"title"`
	Content  string            `json:"content"`
	Source   string            `json:"source,omitempty"`
	MimeType string            `json:"mime_type,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// AddDocument adds a document to a knowledge base
// POST /api/v1/admin/ai/knowledge-bases/:id/documents
func (h *KnowledgeBaseHandler) AddDocument(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)
	kbID := c.Params("id")

	if kbID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Knowledge base ID is required",
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

	// Check if processor is available
	if h.processor == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "Document processing not available (embedding service not configured)",
		})
	}

	// Get knowledge base to check it exists
	kb, err := h.storage.GetKnowledgeBase(ctx, kbID)
	if err != nil {
		log.Error().Err(err).Str("kb_id", kbID).Msg("Failed to get knowledge base")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get knowledge base",
		})
	}
	if kb == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Knowledge base not found",
		})
	}

	// Auto-set user_id in metadata for user isolation
	metadata := req.Metadata
	if uid := middleware.GetUserID(c); uid != "" {
		if metadata == nil {
			metadata = make(map[string]string)
		}
		metadata["user_id"] = uid
	}

	// Add document asynchronously
	docReq := CreateDocumentRequest{
		Title:     req.Title,
		Content:   req.Content,
		SourceURL: req.Source,
		MimeType:  req.MimeType,
		Metadata:  metadata,
	}

	doc, err := h.processor.AddDocument(ctx, kbID, docReq, nil)
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

// UploadDocument uploads a file and extracts text for a knowledge base document
// POST /api/v1/admin/ai/knowledge-bases/:id/documents/upload
func (h *KnowledgeBaseHandler) UploadDocument(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)
	kbID := c.Params("id")

	if kbID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Knowledge base ID is required",
		})
	}

	// Check if storage service is available
	if h.storageService == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "File upload not available (storage service not configured)",
		})
	}

	// Check if processor is available
	if h.processor == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "Document processing not available (embedding service not configured)",
		})
	}

	// Get knowledge base to check it exists
	kb, err := h.storage.GetKnowledgeBase(ctx, kbID)
	if err != nil {
		log.Error().Err(err).Str("kb_id", kbID).Msg("Failed to get knowledge base")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get knowledge base",
		})
	}
	if kb == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Knowledge base not found",
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

	fileData, err := io.ReadAll(fileReader)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to read file content",
		})
	}

	// Get optional OCR language from form (comma-separated, e.g., "eng,deu")
	var ocrLanguages []string
	if langStr := c.FormValue("language"); langStr != "" {
		for _, lang := range strings.Split(langStr, ",") {
			lang = strings.TrimSpace(lang)
			if lang != "" {
				ocrLanguages = append(ocrLanguages, lang)
			}
		}
	}

	// Extract text from file (with OCR fallback if needed)
	extractedText, err := h.textExtractor.ExtractWithLanguages(fileData, mimeType, ocrLanguages)
	if err != nil {
		log.Error().Err(err).Str("filename", file.Filename).Str("mime_type", mimeType).Msg("Failed to extract text from file")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to extract text from file: %v", err),
		})
	}

	if strings.TrimSpace(extractedText) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "No text content could be extracted from the file",
		})
	}

	// Get optional title from form
	title := c.FormValue("title")
	if title == "" {
		// Use filename without extension as title
		title = strings.TrimSuffix(file.Filename, ext)
	}

	// Store the file in the knowledge-base bucket
	storagePath := fmt.Sprintf("kb-%s/%s", kbID, file.Filename)

	// Store file (we need to recreate the reader since we already read it)
	fileReader2, err := file.Open()
	if err != nil {
		log.Error().Err(err).Str("filename", file.Filename).Msg("Failed to reopen file for storage")
	}
	defer func() { _ = fileReader2.Close() }()

	var sourceURL string
	uploadOpts := &storage.UploadOptions{
		ContentType: mimeType,
	}
	_, err = h.storageService.Provider.Upload(ctx, "knowledge-base", storagePath, fileReader2, file.Size, uploadOpts)
	if err != nil {
		log.Error().Err(err).Str("path", storagePath).Str("bucket", "knowledge-base").Msg("Failed to store file in bucket")
		// Continue without storing - the text has been extracted
		sourceURL = "" // No storage URL since upload failed
	} else {
		sourceURL = fmt.Sprintf("storage://knowledge-base/%s", storagePath)
		log.Info().Str("path", storagePath).Str("bucket", "knowledge-base").Msg("File stored successfully")
	}

	// Auto-set user_id in metadata for user isolation
	var metadata map[string]string
	if uid := middleware.GetUserID(c); uid != "" {
		metadata = map[string]string{"user_id": uid}
	}

	// Create document with extracted content
	docReq := CreateDocumentRequest{
		Title:            title,
		Content:          extractedText,
		SourceURL:        sourceURL,
		MimeType:         mimeType,
		OriginalFilename: file.Filename,
		Metadata:         metadata,
	}

	doc, err := h.processor.AddDocument(ctx, kbID, docReq, nil)
	if err != nil {
		log.Error().Err(err).Str("kb_id", kbID).Msg("Failed to add document")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to add document",
		})
	}

	return c.Status(fiber.StatusAccepted).JSON(fiber.Map{
		"document_id":      doc.ID,
		"status":           "processing",
		"message":          "Document is being processed and will be available shortly",
		"filename":         file.Filename,
		"extracted_length": len(extractedText),
		"mime_type":        mimeType,
	})
}

// DeleteDocument deletes a document
// DELETE /api/v1/admin/ai/knowledge-bases/:id/documents/:doc_id
func (h *KnowledgeBaseHandler) DeleteDocument(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)
	docID := c.Params("doc_id")

	if docID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Document ID is required",
		})
	}

	// First, clean up orphaned entities (those only referenced by this document)
	// This is important for table exports where entities are 1:1 with documents
	if h.knowledgeGraph != nil {
		if err := h.knowledgeGraph.DeleteOrphanedEntitiesByDocument(ctx, docID); err != nil {
			log.Warn().Err(err).Str("doc_id", docID).Msg("Failed to delete orphaned entities (continuing)")
		}
	}

	err := h.storage.DeleteDocument(ctx, docID)
	if err != nil {
		log.Error().Err(err).Str("doc_id", docID).Msg("Failed to delete document")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete document",
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// DeleteDocumentsByFilter deletes documents matching a metadata filter
// POST /api/v1/admin/ai/knowledge-bases/:id/documents/delete-by-filter
func (h *KnowledgeBaseHandler) DeleteDocumentsByFilter(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)
	kbID := c.Params("id")

	if kbID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Knowledge base ID is required",
		})
	}

	var req struct {
		Tags           []string             `json:"tags"`
		Metadata       map[string]string    `json:"metadata"`
		MetadataFilter *MetadataFilterGroup `json:"metadata_filter"`
	}

	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Get user ID from context (for user isolation)
	var userID *string
	if uid := middleware.GetUserID(c); uid != "" {
		userID = &uid
	}

	// Build the filter
	filter := &MetadataFilter{
		Tags:           req.Tags,
		Metadata:       req.Metadata,
		AdvancedFilter: req.MetadataFilter,
		IncludeGlobal:  false, // Only delete user's own documents
	}

	// Only non-admin users are filtered by user_id
	if userID != nil {
		isAdmin := false
		if role := c.Locals("role"); role != nil {
			isAdmin = role == "service_role" || role == "instance_admin" || role == "tenant_service"
		}
		if !isAdmin {
			filter.UserID = userID
		}
	}

	// Delete documents
	count, err := h.storage.DeleteDocumentsByFilter(ctx, kbID, filter)
	if err != nil {
		log.Error().Err(err).Str("kb_id", kbID).Msg("Failed to delete documents by filter")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete documents",
		})
	}

	return c.JSON(fiber.Map{
		"deleted_count": count,
	})
}

// UpdateDocument updates a document's metadata and tags
// PATCH /api/v1/admin/ai/knowledge-bases/:id/documents/:doc_id
func (h *KnowledgeBaseHandler) UpdateDocument(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)
	docID := c.Params("doc_id")

	if docID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Document ID is required",
		})
	}

	var req struct {
		Title    *string           `json:"title"`
		Metadata map[string]string `json:"metadata"`
		Tags     []string          `json:"tags"`
	}

	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Get existing document first
	doc, err := h.storage.GetDocument(ctx, docID)
	if err != nil {
		log.Error().Err(err).Str("doc_id", docID).Msg("Failed to get document")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get document",
		})
	}
	if doc == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Document not found",
		})
	}

	// Update document metadata
	updatedDoc, err := h.storage.UpdateDocumentMetadata(ctx, docID, req.Title, req.Metadata, req.Tags)
	if err != nil {
		log.Error().Err(err).Str("doc_id", docID).Msg("Failed to update document")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update document",
		})
	}

	return c.JSON(updatedDoc)
}
