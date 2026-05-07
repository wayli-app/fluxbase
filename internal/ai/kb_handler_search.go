package ai

import (
	"fmt"

	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/middleware"
)

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
func (h *KnowledgeBaseHandler) SearchKnowledgeBase(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)
	kbID := c.Params("id")

	if kbID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Knowledge base ID is required",
		})
	}

	var req SearchKnowledgeBaseRequest
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
func (h *KnowledgeBaseHandler) GetCapabilities(c fiber.Ctx) error {
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
func (h *KnowledgeBaseHandler) DebugSearch(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)
	kbID := c.Params("id")

	if kbID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Knowledge base ID is required",
		})
	}

	var req DebugSearchRequest
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
		switch {
		case stats.TotalChunks == 0:
			response.ErrorMessage = "No chunks in knowledge base"
		case stats.ChunksWithEmbedding == 0:
			response.ErrorMessage = "All chunks have NULL embeddings - document processing may have failed"
		case stats.ChunksWithoutEmbedding > 0:
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
