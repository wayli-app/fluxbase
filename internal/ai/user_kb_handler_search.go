package ai

import (
	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/middleware"
)

// SearchMyKB searches a knowledge base (requires viewer permission)
// POST /api/v1/ai/knowledge-bases/:id/search
func (h *UserKnowledgeBaseHandler) SearchMyKB(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)
	userID := middleware.GetUserID(c)
	kbID := c.Params("id")

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

	var req SearchRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.Query == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Query is required",
		})
	}

	// Set defaults
	if req.Limit == 0 {
		req.Limit = 10
	}

	// Perform search using hybrid search (keyword-only if embeddings not available)
	opts := HybridSearchOptions{
		Query: req.Query,
		Limit: req.Limit,
		Mode:  SearchModeKeyword, // Default to keyword search for user endpoint
	}

	// If processor has embedding service, use hybrid search
	if h.processor != nil && h.processor.embeddingService != nil {
		embedding, err := h.processor.embeddingService.EmbedSingle(ctx, req.Query, "")
		if err == nil && len(embedding) > 0 {
			opts.QueryEmbedding = embedding
			opts.Mode = SearchModeHybrid
			opts.SemanticWeight = 0.7
		}
	}

	results, err := h.storage.SearchChunksHybrid(ctx, kbID, opts)
	if err != nil {
		log.Error().Err(err).Str("kb_id", kbID).Msg("Search failed")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Search failed",
		})
	}

	return c.JSON(fiber.Map{
		"results": results,
		"query":   req.Query,
		"limit":   req.Limit,
		"count":   len(results),
	})
}

// DebugSearchMyKB performs a debug search with detailed diagnostic information
// POST /api/v1/ai/knowledge-bases/:id/debug-search
func (h *UserKnowledgeBaseHandler) DebugSearchMyKB(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)
	userID := middleware.GetUserID(c)
	kbID := c.Params("id")

	// Check viewer permission
	hasPermission, err := h.storage.CheckKBPermission(ctx, kbID, userID, string(KBPermissionViewer))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to check permission",
		})
	}
	if !hasPermission {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Viewer permission required",
		})
	}

	var req struct {
		Query string `json:"query"`
	}
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.Query == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Query is required",
		})
	}

	// Perform search with debug info
	opts := HybridSearchOptions{
		Query:          req.Query,
		Limit:          10,
		SemanticWeight: 0.7,
	}

	results, err := h.storage.SearchChunksHybrid(ctx, kbID, opts)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Search failed",
		})
	}

	// Get KB info for context
	kb, _ := h.storage.GetKnowledgeBase(ctx, kbID)

	return c.JSON(fiber.Map{
		"query":          req.Query,
		"results":        results,
		"result_count":   len(results),
		"search_options": opts,
		"knowledge_base": fiber.Map{
			"id":   kbID,
			"name": kb.Name,
		},
		"debug_info": fiber.Map{
			"search_type":      "hybrid",
			"semantic_weight":  opts.SemanticWeight,
			"keyword_weight":   1 - opts.SemanticWeight,
			"embedding_status": "available",
		},
	})
}

// SearchRequest represents a search request
type SearchRequest struct {
	Query string `json:"query"`
	Limit int    `json:"limit,omitempty"`
}
