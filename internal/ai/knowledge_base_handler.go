package ai

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/fluxbase-eu/fluxbase/internal/storage"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
)

// KnowledgeBaseHandler handles knowledge base management endpoints
type KnowledgeBaseHandler struct {
	storage        *KnowledgeBaseStorage
	processor      *DocumentProcessor
	storageService *storage.Service
	textExtractor  *TextExtractor
	ocrService     *OCRService
}

// NewKnowledgeBaseHandler creates a new knowledge base handler
func NewKnowledgeBaseHandler(storage *KnowledgeBaseStorage, processor *DocumentProcessor) *KnowledgeBaseHandler {
	return &KnowledgeBaseHandler{
		storage:       storage,
		processor:     processor,
		textExtractor: NewTextExtractor(),
	}
}

// NewKnowledgeBaseHandlerWithOCR creates a new knowledge base handler with OCR support
func NewKnowledgeBaseHandlerWithOCR(storage *KnowledgeBaseStorage, processor *DocumentProcessor, ocrService *OCRService) *KnowledgeBaseHandler {
	return &KnowledgeBaseHandler{
		storage:       storage,
		processor:     processor,
		textExtractor: NewTextExtractorWithOCR(ocrService),
		ocrService:    ocrService,
	}
}

// SetStorageService sets the storage service for file uploads
func (h *KnowledgeBaseHandler) SetStorageService(svc *storage.Service) {
	h.storageService = svc
}

// ============================================================================
// KNOWLEDGE BASE ENDPOINTS
// ============================================================================

// ListKnowledgeBases returns all knowledge bases
// GET /api/v1/admin/ai/knowledge-bases
func (h *KnowledgeBaseHandler) ListKnowledgeBases(c *fiber.Ctx) error {
	ctx := c.Context()

	kbs, err := h.storage.ListAllKnowledgeBases(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to list knowledge bases")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to list knowledge bases",
		})
	}

	// Convert to summaries
	summaries := make([]KnowledgeBaseSummary, len(kbs))
	for i, kb := range kbs {
		summaries[i] = kb.ToSummary()
	}

	return c.JSON(fiber.Map{
		"knowledge_bases": summaries,
		"count":           len(summaries),
	})
}

// GetKnowledgeBase returns a specific knowledge base
// GET /api/v1/admin/ai/knowledge-bases/:id
func (h *KnowledgeBaseHandler) GetKnowledgeBase(c *fiber.Ctx) error {
	ctx := c.Context()
	id := c.Params("id")

	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Knowledge base ID is required",
		})
	}

	kb, err := h.storage.GetKnowledgeBase(ctx, id)
	if err != nil {
		log.Error().Err(err).Str("id", id).Msg("Failed to get knowledge base")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get knowledge base",
		})
	}
	if kb == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Knowledge base not found",
		})
	}

	return c.JSON(kb)
}

// CreateKnowledgeBase creates a new knowledge base
// POST /api/v1/admin/ai/knowledge-bases
func (h *KnowledgeBaseHandler) CreateKnowledgeBase(c *fiber.Ctx) error {
	ctx := c.Context()

	var req CreateKnowledgeBaseRequest
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

	kb, err := h.storage.CreateKnowledgeBaseFromRequest(ctx, req)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create knowledge base")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create knowledge base",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(kb)
}

// UpdateKnowledgeBase updates an existing knowledge base
// PUT /api/v1/admin/ai/knowledge-bases/:id
func (h *KnowledgeBaseHandler) UpdateKnowledgeBase(c *fiber.Ctx) error {
	ctx := c.Context()
	id := c.Params("id")

	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Knowledge base ID is required",
		})
	}

	var req UpdateKnowledgeBaseRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	kb, err := h.storage.UpdateKnowledgeBaseByID(ctx, id, req)
	if err != nil {
		log.Error().Err(err).Str("id", id).Msg("Failed to update knowledge base")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update knowledge base",
		})
	}
	if kb == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Knowledge base not found",
		})
	}

	return c.JSON(kb)
}

// DeleteKnowledgeBase deletes a knowledge base
// DELETE /api/v1/admin/ai/knowledge-bases/:id
func (h *KnowledgeBaseHandler) DeleteKnowledgeBase(c *fiber.Ctx) error {
	ctx := c.Context()
	id := c.Params("id")

	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Knowledge base ID is required",
		})
	}

	err := h.storage.DeleteKnowledgeBase(ctx, id)
	if err != nil {
		log.Error().Err(err).Str("id", id).Msg("Failed to delete knowledge base")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete knowledge base",
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// ============================================================================
// DOCUMENT ENDPOINTS
// ============================================================================

// ListDocuments returns all documents in a knowledge base
// GET /api/v1/admin/ai/knowledge-bases/:id/documents
func (h *KnowledgeBaseHandler) ListDocuments(c *fiber.Ctx) error {
	ctx := c.Context()
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
func (h *KnowledgeBaseHandler) GetDocument(c *fiber.Ctx) error {
	ctx := c.Context()
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
func (h *KnowledgeBaseHandler) AddDocument(c *fiber.Ctx) error {
	ctx := c.Context()
	kbID := c.Params("id")

	if kbID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Knowledge base ID is required",
		})
	}

	var req AddDocumentRequest
	if err := c.BodyParser(&req); err != nil {
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
	if uid, ok := c.Locals("user_id").(string); ok && uid != "" {
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
func (h *KnowledgeBaseHandler) UploadDocument(c *fiber.Ctx) error {
	ctx := c.Context()
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
	if uid, ok := c.Locals("user_id").(string); ok && uid != "" {
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
func (h *KnowledgeBaseHandler) DeleteDocument(c *fiber.Ctx) error {
	ctx := c.Context()
	docID := c.Params("doc_id")

	if docID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Document ID is required",
		})
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

// UpdateDocument updates a document's metadata and tags
// PATCH /api/v1/admin/ai/knowledge-bases/:id/documents/:doc_id
func (h *KnowledgeBaseHandler) UpdateDocument(c *fiber.Ctx) error {
	ctx := c.Context()
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

	if err := c.BodyParser(&req); err != nil {
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

// ============================================================================
// CHATBOT-KNOWLEDGE BASE LINKING ENDPOINTS
// ============================================================================

// ListChatbotKnowledgeBases returns knowledge bases linked to a chatbot
// GET /api/v1/admin/ai/chatbots/:id/knowledge-bases
func (h *KnowledgeBaseHandler) ListChatbotKnowledgeBases(c *fiber.Ctx) error {
	ctx := c.Context()
	chatbotID := c.Params("id")

	if chatbotID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Chatbot ID is required",
		})
	}

	links, err := h.storage.GetChatbotKnowledgeBases(ctx, chatbotID)
	if err != nil {
		log.Error().Err(err).Str("chatbot_id", chatbotID).Msg("Failed to get chatbot knowledge bases")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get chatbot knowledge bases",
		})
	}

	return c.JSON(fiber.Map{
		"knowledge_bases": links,
		"count":           len(links),
	})
}

// LinkKnowledgeBase links a knowledge base to a chatbot
// POST /api/v1/admin/ai/chatbots/:id/knowledge-bases
func (h *KnowledgeBaseHandler) LinkKnowledgeBase(c *fiber.Ctx) error {
	ctx := c.Context()
	chatbotID := c.Params("id")

	if chatbotID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Chatbot ID is required",
		})
	}

	var req LinkKnowledgeBaseRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.KnowledgeBaseID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Knowledge base ID is required",
		})
	}

	// Set defaults
	priority := 1
	maxChunks := 5
	similarityThreshold := 0.7

	if req.Priority != nil {
		priority = *req.Priority
	}
	if req.MaxChunks != nil {
		maxChunks = *req.MaxChunks
	}
	if req.SimilarityThreshold != nil {
		similarityThreshold = *req.SimilarityThreshold
	}

	link, err := h.storage.LinkChatbotKnowledgeBaseSimple(ctx, chatbotID, req.KnowledgeBaseID, priority, maxChunks, similarityThreshold)
	if err != nil {
		log.Error().Err(err).
			Str("chatbot_id", chatbotID).
			Str("kb_id", req.KnowledgeBaseID).
			Msg("Failed to link knowledge base")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to link knowledge base",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(link)
}

// UpdateChatbotKnowledgeBaseRequest represents a request to update a link
type UpdateChatbotKnowledgeBaseRequest struct {
	Priority            *int     `json:"priority,omitempty"`
	MaxChunks           *int     `json:"max_chunks,omitempty"`
	SimilarityThreshold *float64 `json:"similarity_threshold,omitempty"`
	Enabled             *bool    `json:"enabled,omitempty"`
}

// UpdateChatbotKnowledgeBase updates a chatbot-knowledge base link
// PUT /api/v1/admin/ai/chatbots/:id/knowledge-bases/:kb_id
func (h *KnowledgeBaseHandler) UpdateChatbotKnowledgeBase(c *fiber.Ctx) error {
	ctx := c.Context()
	chatbotID := c.Params("id")
	kbID := c.Params("kb_id")

	if chatbotID == "" || kbID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Chatbot ID and knowledge base ID are required",
		})
	}

	var req UpdateChatbotKnowledgeBaseRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	link, err := h.storage.UpdateChatbotKnowledgeBaseLink(ctx, chatbotID, kbID, UpdateChatbotKnowledgeBaseOptions(req))
	if err != nil {
		log.Error().Err(err).
			Str("chatbot_id", chatbotID).
			Str("kb_id", kbID).
			Msg("Failed to update chatbot knowledge base link")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update chatbot knowledge base link",
		})
	}
	if link == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Chatbot-knowledge base link not found",
		})
	}

	return c.JSON(link)
}

// UnlinkKnowledgeBase removes a knowledge base from a chatbot
// DELETE /api/v1/admin/ai/chatbots/:id/knowledge-bases/:kb_id
func (h *KnowledgeBaseHandler) UnlinkKnowledgeBase(c *fiber.Ctx) error {
	ctx := c.Context()
	chatbotID := c.Params("id")
	kbID := c.Params("kb_id")

	if chatbotID == "" || kbID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Chatbot ID and knowledge base ID are required",
		})
	}

	err := h.storage.UnlinkChatbotKnowledgeBase(ctx, chatbotID, kbID)
	if err != nil {
		log.Error().Err(err).
			Str("chatbot_id", chatbotID).
			Str("kb_id", kbID).
			Msg("Failed to unlink knowledge base")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to unlink knowledge base",
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// ============================================================================
// SEARCH/TEST ENDPOINTS
// ============================================================================

// SearchKnowledgeBaseRequest represents a search request
type SearchKnowledgeBaseRequest struct {
	Query          string  `json:"query"`
	MaxChunks      int     `json:"max_chunks,omitempty"`
	Threshold      float64 `json:"threshold,omitempty"`
	Mode           string  `json:"mode,omitempty"`            // "semantic", "keyword", or "hybrid"
	SemanticWeight float64 `json:"semantic_weight,omitempty"` // For hybrid mode: 0-1, default 0.5
}

// SearchKnowledgeBase searches a specific knowledge base
// POST /api/v1/admin/ai/knowledge-bases/:id/search
func (h *KnowledgeBaseHandler) SearchKnowledgeBase(c *fiber.Ctx) error {
	ctx := c.Context()
	kbID := c.Params("id")

	if kbID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Knowledge base ID is required",
		})
	}

	var req SearchKnowledgeBaseRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.Query == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Query is required",
		})
	}

	// Check if processor is available (has embedding service)
	if h.processor == nil || h.processor.embeddingService == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "Search not available (embedding service not configured)",
		})
	}

	// Set defaults
	if req.MaxChunks == 0 {
		req.MaxChunks = 10
	}
	if req.Threshold == 0 {
		req.Threshold = 0.2 // Lower default for hybrid/keyword search
	}
	if req.SemanticWeight == 0 {
		req.SemanticWeight = 0.5 // Default 50/50 for hybrid
	}

	// Determine search mode
	searchMode := SearchModeSemantic
	switch req.Mode {
	case "keyword":
		searchMode = SearchModeKeyword
	case "hybrid":
		searchMode = SearchModeHybrid
	}

	// For keyword-only mode, we don't need embeddings
	var embedding []float32
	if searchMode != SearchModeKeyword {
		var err error
		embedding, err = h.processor.embeddingService.EmbedSingle(ctx, req.Query, "")
		if err != nil {
			log.Error().Err(err).Msg("Failed to embed query")
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to embed query",
			})
		}
	}

	log.Debug().
		Str("kb_id", kbID).
		Str("query", req.Query).
		Str("mode", string(searchMode)).
		Int("embedding_dims", len(embedding)).
		Float64("threshold", req.Threshold).
		Int("max_chunks", req.MaxChunks).
		Float64("semantic_weight", req.SemanticWeight).
		Msg("Searching knowledge base")

	// Search using hybrid search
	results, err := h.storage.SearchChunksHybrid(ctx, kbID, HybridSearchOptions{
		Query:          req.Query,
		QueryEmbedding: embedding,
		Limit:          req.MaxChunks,
		Threshold:      req.Threshold,
		Mode:           searchMode,
		SemanticWeight: req.SemanticWeight,
	})
	if err != nil {
		log.Error().Err(err).Str("kb_id", kbID).Msg("Failed to search knowledge base")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to search knowledge base",
		})
	}

	// Log results for debugging
	if len(results) > 0 {
		log.Debug().
			Str("kb_id", kbID).
			Int("result_count", len(results)).
			Float64("top_similarity", results[0].Similarity).
			Str("mode", string(searchMode)).
			Msg("Search completed")
	} else {
		log.Debug().
			Str("kb_id", kbID).
			Float64("threshold", req.Threshold).
			Str("mode", string(searchMode)).
			Msg("Search returned no results")
	}

	return c.JSON(fiber.Map{
		"results": results,
		"count":   len(results),
		"query":   req.Query,
		"mode":    string(searchMode),
	})
}

// KnowledgeBaseCapabilities represents the capabilities of the knowledge base system
type KnowledgeBaseCapabilities struct {
	OCREnabled         bool     `json:"ocr_enabled"`
	OCRAvailable       bool     `json:"ocr_available"`
	OCRLanguages       []string `json:"ocr_languages"`
	SupportedFileTypes []string `json:"supported_file_types"`
}

// GetCapabilities returns the capabilities of the knowledge base system
// GET /api/v1/admin/ai/knowledge-bases/capabilities
func (h *KnowledgeBaseHandler) GetCapabilities(c *fiber.Ctx) error {
	// Check if OCR is enabled and available
	ocrEnabled := h.ocrService != nil
	ocrAvailable := ocrEnabled && h.ocrService.IsEnabled()

	var ocrLanguages []string
	if ocrAvailable {
		ocrLanguages = h.ocrService.GetDefaultLanguages()
	}

	// Get supported file types from text extractor
	supportedTypes := h.textExtractor.SupportedMimeTypes()

	// Convert MIME types to file extensions for the UI
	fileExtensions := []string{}
	for _, mimeType := range supportedTypes {
		ext := GetExtensionFromMimeType(mimeType)
		if ext != "" {
			fileExtensions = append(fileExtensions, ext)
		}
	}

	return c.JSON(KnowledgeBaseCapabilities{
		OCREnabled:         ocrEnabled,
		OCRAvailable:       ocrAvailable,
		OCRLanguages:       ocrLanguages,
		SupportedFileTypes: fileExtensions,
	})
}

// DebugSearchRequest represents a debug search request
type DebugSearchRequest struct {
	Query string `json:"query"`
}

// DebugSearchResponse contains detailed debug information about similarity search
type DebugSearchResponse struct {
	Query                  string    `json:"query"`
	QueryEmbeddingPreview  []float32 `json:"query_embedding_preview"`
	QueryEmbeddingDims     int       `json:"query_embedding_dims"`
	StoredEmbeddingPreview []float32 `json:"stored_embedding_preview,omitempty"`
	RawSimilarities        []float64 `json:"raw_similarities"`
	EmbeddingModel         string    `json:"embedding_model"`
	KBEmbeddingModel       string    `json:"kb_embedding_model"`
	ChunksFound            int       `json:"chunks_found"`
	TopChunkContentPreview string    `json:"top_chunk_content_preview,omitempty"`
	// Chunk statistics
	TotalChunks            int    `json:"total_chunks"`
	ChunksWithEmbedding    int    `json:"chunks_with_embedding"`
	ChunksWithoutEmbedding int    `json:"chunks_without_embedding"`
	ErrorMessage           string `json:"error_message,omitempty"`
}

// DebugSearch provides detailed debugging information for similarity search
// POST /api/v1/admin/ai/knowledge-bases/:id/debug-search
func (h *KnowledgeBaseHandler) DebugSearch(c *fiber.Ctx) error {
	ctx := c.Context()
	kbID := c.Params("id")

	if kbID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Knowledge base ID is required",
		})
	}

	var req DebugSearchRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.Query == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Query is required",
		})
	}

	// Check if processor is available
	if h.processor == nil || h.processor.embeddingService == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "Embedding service not configured",
		})
	}

	// Get KB info
	kb, err := h.storage.GetKnowledgeBase(ctx, kbID)
	if err != nil || kb == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Knowledge base not found",
		})
	}

	// Generate embedding for query
	queryEmbedding, err := h.processor.embeddingService.EmbedSingle(ctx, req.Query, "")
	if err != nil {
		log.Error().Err(err).Msg("Failed to embed query")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to embed query",
		})
	}

	// Get embedding preview (first 10 values)
	queryPreview := queryEmbedding
	if len(queryPreview) > 10 {
		queryPreview = queryPreview[:10]
	}

	// Log query embedding details
	log.Info().
		Int("query_embedding_dims", len(queryEmbedding)).
		Float32("first_value", queryEmbedding[0]).
		Float32("second_value", queryEmbedding[1]).
		Str("query", req.Query).
		Msg("Debug search - query embedding generated")

	// Get chunk embedding statistics first
	stats, statsErr := h.storage.GetChunkEmbeddingStats(ctx, kbID)
	if statsErr != nil {
		log.Warn().Err(statsErr).Msg("Failed to get chunk stats")
	}

	// Search with negative threshold to get ALL results (including negative similarity)
	// Using -2.0 ensures we get everything since cosine similarity range is [-1, 1]
	results, err := h.storage.SearchChunks(ctx, kbID, queryEmbedding, 10, -2.0)
	if err != nil {
		log.Error().Err(err).Msg("Failed to search chunks")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to search: " + err.Error(),
		})
	}

	log.Info().
		Int("results_count", len(results)).
		Msg("Debug search - search completed")

	// Extract similarities
	similarities := make([]float64, len(results))
	for i, r := range results {
		similarities[i] = r.Similarity
	}

	response := DebugSearchResponse{
		Query:                 req.Query,
		QueryEmbeddingPreview: queryPreview,
		QueryEmbeddingDims:    len(queryEmbedding),
		RawSimilarities:       similarities,
		EmbeddingModel:        h.processor.embeddingService.DefaultModel(),
		KBEmbeddingModel:      kb.EmbeddingModel,
		ChunksFound:           len(results),
	}

	// Add chunk stats
	if stats != nil {
		response.TotalChunks = stats.TotalChunks
		response.ChunksWithEmbedding = stats.ChunksWithEmbedding
		response.ChunksWithoutEmbedding = stats.ChunksWithoutEmbedding

		// Check for problematic state
		if stats.TotalChunks == 0 {
			response.ErrorMessage = "No chunks in knowledge base"
		} else if stats.ChunksWithEmbedding == 0 {
			response.ErrorMessage = "All chunks have NULL embeddings - document processing may have failed"
		} else if stats.ChunksWithoutEmbedding > 0 {
			response.ErrorMessage = fmt.Sprintf("%d chunks have NULL embeddings", stats.ChunksWithoutEmbedding)
		}
	}

	// Get stored embedding preview from top result or first chunk with embedding
	if len(results) > 0 {
		// Get content preview
		content := results[0].Content
		if len(content) > 200 {
			content = content[:200] + "..."
		}
		response.TopChunkContentPreview = content

		// Get stored embedding preview
		storedPreview, err := h.storage.GetChunkEmbeddingPreview(ctx, results[0].ChunkID, 10)
		if err == nil {
			response.StoredEmbeddingPreview = storedPreview
		} else {
			log.Warn().Err(err).Str("chunk_id", results[0].ChunkID).Msg("Failed to get embedding preview")
		}
	} else if stats != nil && stats.ChunksWithEmbedding > 0 {
		// No results but there are chunks with embeddings - try to get one
		chunkID, err := h.storage.GetFirstChunkWithEmbedding(ctx, kbID)
		if err == nil {
			storedPreview, err := h.storage.GetChunkEmbeddingPreview(ctx, chunkID, 10)
			if err == nil {
				response.StoredEmbeddingPreview = storedPreview
			}
		}
	}

	return c.JSON(response)
}
