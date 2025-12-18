package ai

import (
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
)

// KnowledgeBaseHandler handles knowledge base management endpoints
type KnowledgeBaseHandler struct {
	storage   *KnowledgeBaseStorage
	processor *DocumentProcessor
}

// NewKnowledgeBaseHandler creates a new knowledge base handler
func NewKnowledgeBaseHandler(storage *KnowledgeBaseStorage, processor *DocumentProcessor) *KnowledgeBaseHandler {
	return &KnowledgeBaseHandler{
		storage:   storage,
		processor: processor,
	}
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

	// Add document asynchronously
	docID, err := h.processor.AddDocument(ctx, kbID, req.Title, req.Content, req.Source, req.MimeType, req.Metadata)
	if err != nil {
		log.Error().Err(err).Str("kb_id", kbID).Msg("Failed to add document")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to add document",
		})
	}

	return c.Status(fiber.StatusAccepted).JSON(fiber.Map{
		"document_id": docID,
		"status":      "processing",
		"message":     "Document is being processed and will be available shortly",
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

	link, err := h.storage.UpdateChatbotKnowledgeBaseLink(ctx, chatbotID, kbID, UpdateChatbotKnowledgeBaseOptions{
		Priority:            req.Priority,
		MaxChunks:           req.MaxChunks,
		SimilarityThreshold: req.SimilarityThreshold,
		Enabled:             req.Enabled,
	})
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
	Query     string  `json:"query"`
	MaxChunks int     `json:"max_chunks,omitempty"`
	Threshold float64 `json:"threshold,omitempty"`
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
	if h.processor == nil || h.processor.embedding == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "Search not available (embedding service not configured)",
		})
	}

	// Set defaults
	if req.MaxChunks == 0 {
		req.MaxChunks = 10
	}
	if req.Threshold == 0 {
		req.Threshold = 0.5
	}

	// Generate embedding for query
	embedding, err := h.processor.embedding.EmbedSingle(ctx, req.Query, "")
	if err != nil {
		log.Error().Err(err).Msg("Failed to embed query")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to embed query",
		})
	}

	// Search
	results, err := h.storage.SearchChunks(ctx, kbID, embedding, req.MaxChunks, req.Threshold)
	if err != nil {
		log.Error().Err(err).Str("kb_id", kbID).Msg("Failed to search knowledge base")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to search knowledge base",
		})
	}

	return c.JSON(fiber.Map{
		"results": results,
		"count":   len(results),
		"query":   req.Query,
	})
}
