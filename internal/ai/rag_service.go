package ai

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

// RAGService handles retrieval-augmented generation for chatbots
type RAGService struct {
	storage          *KnowledgeBaseStorage
	embeddingService *EmbeddingService
	knowledgeGraph   *KnowledgeGraph // For graph-boosted search
	entityExtractor  EntityExtractor // For extracting entities from queries
}

// NewRAGService creates a new RAG service
func NewRAGService(
	storage *KnowledgeBaseStorage,
	embeddingService *EmbeddingService,
	knowledgeGraph *KnowledgeGraph,
	entityExtractor EntityExtractor,
) *RAGService {
	return &RAGService{
		storage:          storage,
		embeddingService: embeddingService,
		knowledgeGraph:   knowledgeGraph,
		entityExtractor:  entityExtractor,
	}
}

// RetrieveContextOptions contains options for retrieval
type RetrieveContextOptions struct {
	ChatbotID        string
	ConversationID   string
	UserID           string
	Query            string
	MaxChunks        int     // Override max chunks (optional)
	Threshold        float64 // Override threshold (optional)
	GraphBoostWeight float64 // Optional: >0 enables entity-based re-ranking
}

// RetrieveContextResult contains the retrieval results
type RetrieveContextResult struct {
	Chunks           []RetrievalResult
	FormattedContext string
	TotalRetrieved   int
	DurationMs       int64
	EmbeddingModel   string
}

// RetrieveContext retrieves relevant context for a user query
func (r *RAGService) RetrieveContext(ctx context.Context, opts RetrieveContextOptions) (*RetrieveContextResult, error) {
	if r.embeddingService == nil {
		return nil, fmt.Errorf("embedding service not configured")
	}

	start := time.Now()

	// Generate embedding for the query
	queryEmbedding, err := r.embeddingService.EmbedSingle(ctx, opts.Query, "")
	if err != nil {
		return nil, fmt.Errorf("failed to embed query: %w", err)
	}

	// Build search options with user context for isolation
	searchOpts := SearchChatbotKnowledgeOptions{
		MaxChunks: opts.MaxChunks,
		Threshold: opts.Threshold,
	}
	if opts.UserID != "" {
		searchOpts.UserID = &opts.UserID
	}

	// Search for relevant chunks. When graph boost is requested and the
	// knowledge graph + entity extractor are wired, take the graph-boosted
	// path which over-fetches and re-ranks by entity salience. Otherwise use
	// the standard cross-KB search.
	var chunks []RetrievalResult
	if opts.GraphBoostWeight > 0 && r.knowledgeGraph != nil && r.entityExtractor != nil {
		chunks, err = r.retrieveContextWithGraphBoost(ctx, opts, queryEmbedding)
	} else {
		chunks, err = r.storage.SearchChatbotKnowledgeWithOptions(ctx, opts.ChatbotID, queryEmbedding, searchOpts)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to search knowledge: %w", err)
	}

	// Apply optional overrides
	if opts.MaxChunks > 0 && len(chunks) > opts.MaxChunks {
		chunks = chunks[:opts.MaxChunks]
	}
	if opts.Threshold > 0 {
		var filtered []RetrievalResult
		for _, chunk := range chunks {
			if chunk.Similarity >= opts.Threshold {
				filtered = append(filtered, chunk)
			}
		}
		chunks = filtered
	}

	duration := time.Since(start)

	// Format context for LLM
	formattedContext := r.formatContext(chunks)

	// Log retrieval
	chunkIDs := make([]string, len(chunks))
	scores := make([]float64, len(chunks))
	for i, chunk := range chunks {
		chunkIDs[i] = chunk.ChunkID
		scores[i] = chunk.Similarity
	}

	_ = r.storage.LogRetrieval(ctx, &RetrievalLog{
		ChatbotID:           &opts.ChatbotID,
		ConversationID:      optString(opts.ConversationID),
		UserID:              optString(opts.UserID),
		QueryText:           opts.Query,
		QueryEmbeddingModel: r.embeddingService.DefaultModel(),
		ChunksRetrieved:     len(chunks),
		ChunkIDs:            chunkIDs,
		SimilarityScores:    scores,
		RetrievalDurationMs: int(duration.Milliseconds()),
	})

	return &RetrieveContextResult{
		Chunks:           chunks,
		FormattedContext: formattedContext,
		TotalRetrieved:   len(chunks),
		DurationMs:       duration.Milliseconds(),
		EmbeddingModel:   r.embeddingService.DefaultModel(),
	}, nil
}

// retrieveContextWithGraphBoost runs the chat retrieval path with graph-boosted
// re-ranking. Mirrors the standard cross-KB search but calls SearchChunksWithGraphBoost
// per linked KB, preserving user isolation via the MetadataFilter.
func (r *RAGService) retrieveContextWithGraphBoost(ctx context.Context, opts RetrieveContextOptions, queryEmbedding []float32) ([]RetrievalResult, error) {
	links, err := r.storage.GetChatbotKnowledgeBases(ctx, opts.ChatbotID)
	if err != nil {
		return nil, fmt.Errorf("failed to get linked knowledge bases: %w", err)
	}
	if len(links) == 0 {
		return nil, nil
	}

	maxChunks := opts.MaxChunks
	if maxChunks <= 0 {
		maxChunks = 5
	}
	threshold := opts.Threshold
	if threshold <= 0 {
		threshold = 0.7
	}

	// Build the same user-isolation filter SearchChatbotKnowledgeWithOptions uses.
	var userFilter *MetadataFilter
	if opts.UserID != "" {
		userFilter = &MetadataFilter{
			UserID:        &opts.UserID,
			IncludeGlobal: true,
		}
	}

	var allResults []RetrievalResult
	for _, link := range links {
		if !link.Enabled {
			continue
		}

		graphOpts := GraphBoostOptions{
			QueryEmbedding:   queryEmbedding,
			QueryText:        opts.Query,
			Limit:            maxChunks,
			Threshold:        threshold,
			GraphBoostWeight: opts.GraphBoostWeight,
			Filter:           userFilter,
		}

		results, err := r.storage.SearchChunksWithGraphBoost(ctx, link.KnowledgeBaseID, r.knowledgeGraph, r.entityExtractor, graphOpts)
		if err != nil {
			log.Warn().Err(err).Str("kb_id", link.KnowledgeBaseID).Msg("Graph-boosted search failed, skipping KB")
			continue
		}

		// Stamp KB name onto results for context formatting.
		if kb, getErr := r.storage.GetKnowledgeBase(ctx, link.KnowledgeBaseID); getErr == nil && kb != nil {
			for i := range results {
				results[i].KnowledgeBaseName = kb.Name
			}
		}

		allResults = append(allResults, results...)
	}

	return allResults, nil
}

// formatContext formats retrieved chunks into a string for the LLM prompt
func (r *RAGService) formatContext(chunks []RetrievalResult) string {
	if len(chunks) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("## Relevant Knowledge\n\n")
	sb.WriteString("The following information was retrieved from the knowledge base and may be relevant to the user's question:\n\n")

	for i, chunk := range chunks {
		fmt.Fprintf(&sb, "### Source %d", i+1)
		if chunk.DocumentTitle != "" {
			fmt.Fprintf(&sb, ": %s", chunk.DocumentTitle)
		}
		if chunk.KnowledgeBaseName != "" {
			fmt.Fprintf(&sb, " (from %s)", chunk.KnowledgeBaseName)
		}
		fmt.Fprintf(&sb, " [similarity: %.2f]\n\n", chunk.Similarity)
		sb.WriteString(chunk.Content)
		sb.WriteString("\n\n---\n\n")
	}

	return sb.String()
}

// RetrieveForKnowledgeBase retrieves from a specific knowledge base
func (r *RAGService) RetrieveForKnowledgeBase(ctx context.Context, kbID string, query string, maxChunks int, threshold float64) ([]RetrievalResult, error) {
	if r.embeddingService == nil {
		return nil, fmt.Errorf("embedding service not configured")
	}

	// Generate embedding
	queryEmbedding, err := r.embeddingService.EmbedSingle(ctx, query, "")
	if err != nil {
		return nil, fmt.Errorf("failed to embed query: %w", err)
	}

	// Search
	chunks, err := r.storage.SearchChunks(ctx, kbID, queryEmbedding, maxChunks, threshold)
	if err != nil {
		return nil, fmt.Errorf("failed to search chunks: %w", err)
	}

	return chunks, nil
}

// VectorSearch performs an explicit search for the vector_search tool with user isolation
// Uses hybrid search combining vector similarity with full-text search for better results
func (r *RAGService) VectorSearch(ctx context.Context, opts VectorSearchOptions) ([]VectorSearchResult, error) {
	if r.embeddingService == nil {
		return nil, fmt.Errorf("embedding service not configured")
	}

	// Apply defaults
	if opts.Limit <= 0 {
		opts.Limit = 5
	}
	if opts.Limit > 20 {
		opts.Limit = 20
	}
	if opts.Threshold <= 0 {
		opts.Threshold = 0.7
	}

	// Generate embedding for the query
	queryEmbedding, err := r.embeddingService.EmbedSingle(ctx, opts.Query, "")
	if err != nil {
		return nil, fmt.Errorf("failed to embed query: %w", err)
	}

	// Get linked knowledge bases for this chatbot
	links, err := r.storage.GetChatbotKnowledgeBases(ctx, opts.ChatbotID)
	if err != nil {
		return nil, fmt.Errorf("failed to get linked knowledge bases: %w", err)
	}

	if len(links) == 0 {
		return nil, fmt.Errorf("no knowledge bases linked to this chatbot")
	}

	// Build metadata filter for user isolation and custom filters
	var filter *MetadataFilter
	if opts.UserID != nil && !opts.IsAdmin {
		filter = &MetadataFilter{
			UserID:         opts.UserID,
			Tags:           opts.Tags,
			Metadata:       opts.Metadata,
			AdvancedFilter: opts.MetadataFilter,
			IncludeGlobal:  true,
		}
	} else if len(opts.Tags) > 0 || len(opts.Metadata) > 0 || opts.MetadataFilter != nil {
		filter = &MetadataFilter{
			Tags:           opts.Tags,
			Metadata:       opts.Metadata,
			AdvancedFilter: opts.MetadataFilter,
		}
	}

	// Resolve which KBs to search
	kbsToSearch := make(map[string]*KnowledgeBase)
	for _, link := range links {
		if !link.Enabled {
			continue
		}
		kb, err := r.storage.GetKnowledgeBase(ctx, link.KnowledgeBaseID)
		if err != nil || kb == nil || !kb.Enabled {
			continue
		}

		// If specific KBs requested, filter to those
		if len(opts.KnowledgeBases) > 0 {
			for _, name := range opts.KnowledgeBases {
				if kb.Name == name {
					kbsToSearch[kb.ID] = kb
					break
				}
			}
		} else {
			kbsToSearch[kb.ID] = kb
		}
	}

	if len(kbsToSearch) == 0 {
		if len(opts.KnowledgeBases) > 0 {
			return nil, fmt.Errorf("no matching knowledge bases found for the specified names")
		}
		return nil, fmt.Errorf("no enabled knowledge bases available")
	}

	// Search each KB using hybrid search and aggregate results
	var allResults []VectorSearchResult
	perKBLimit := opts.Limit // Could distribute across KBs if needed

	for kbID, kb := range kbsToSearch {
		var results []RetrievalResult
		var err error

		// Use graph-boosted search if requested and available
		if opts.GraphBoostWeight > 0 && r.knowledgeGraph != nil && r.entityExtractor != nil {
			graphOpts := GraphBoostOptions{
				QueryEmbedding:   queryEmbedding,
				QueryText:        opts.Query,
				Limit:            perKBLimit,
				Threshold:        opts.Threshold,
				GraphBoostWeight: opts.GraphBoostWeight,
			}
			results, err = r.storage.SearchChunksWithGraphBoost(ctx, kbID, r.knowledgeGraph, r.entityExtractor, graphOpts)
			if err != nil {
				log.Warn().Err(err).Str("kb_id", kbID).Str("kb_name", kb.Name).Msg("Failed to search with graph boost, falling back to hybrid")
				// Fall back to hybrid search
			}
		}

		// Fallback to hybrid search if graph search wasn't used or failed
		if len(results) == 0 {
			hybridOpts := HybridSearchOptions{
				Query:          opts.Query,
				QueryEmbedding: queryEmbedding,
				Limit:          perKBLimit,
				Threshold:      opts.Threshold,
				Mode:           SearchModeHybrid,
				SemanticWeight: 0.7, // 70% semantic, 30% keyword
				KeywordBoost:   0.2, // 20% boost for exact keyword matches
				Filter:         filter,
			}

			results, err = r.storage.SearchChunksHybrid(ctx, kbID, hybridOpts)
			if err != nil {
				log.Warn().Err(err).Str("kb_id", kbID).Str("kb_name", kb.Name).Msg("Failed to search knowledge base")
				continue
			}
		}

		for _, result := range results {
			allResults = append(allResults, VectorSearchResult{
				ChunkID:           result.ChunkID,
				DocumentID:        result.DocumentID,
				DocumentTitle:     result.DocumentTitle,
				KnowledgeBaseName: kb.Name,
				Content:           result.Content,
				Similarity:        result.Similarity,
				Tags:              result.Tags,
			})
		}
	}

	// Sort by similarity (highest first) and limit
	sortVectorSearchResults(allResults)

	if len(allResults) > opts.Limit {
		allResults = allResults[:opts.Limit]
	}

	// Log the retrieval
	chunkIDs := make([]string, len(allResults))
	scores := make([]float64, len(allResults))
	for i, result := range allResults {
		chunkIDs[i] = result.ChunkID
		scores[i] = result.Similarity
	}

	_ = r.storage.LogRetrieval(ctx, &RetrievalLog{
		ChatbotID:           &opts.ChatbotID,
		UserID:              opts.UserID,
		QueryText:           opts.Query,
		QueryEmbeddingModel: r.embeddingService.DefaultModel(),
		ChunksRetrieved:     len(allResults),
		ChunkIDs:            chunkIDs,
		SimilarityScores:    scores,
	})

	return allResults, nil
}

// sortVectorSearchResults sorts results by similarity descending
func sortVectorSearchResults(results []VectorSearchResult) {
	for i := 0; i < len(results)-1; i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].Similarity > results[i].Similarity {
				results[i], results[j] = results[j], results[i]
			}
		}
	}
}

// GetChatbotRAGConfig returns the RAG configuration for a chatbot
func (r *RAGService) GetChatbotRAGConfig(ctx context.Context, chatbotID string) (*ChatbotRAGConfig, error) {
	links, err := r.storage.GetChatbotKnowledgeBases(ctx, chatbotID)
	if err != nil {
		return nil, err
	}

	if len(links) == 0 {
		return nil, nil
	}

	// Calculate totals
	var totalMaxChunks int
	var knowledgeBases []KnowledgeBaseSummary

	for _, link := range links {
		if !link.Enabled {
			continue
		}

		// Use configured max chunks or default to 5
		maxChunks := 5
		if link.MaxChunks != nil {
			maxChunks = *link.MaxChunks
		}
		totalMaxChunks += maxChunks

		kb, err := r.storage.GetKnowledgeBase(ctx, link.KnowledgeBaseID)
		if err == nil && kb != nil && kb.Enabled {
			knowledgeBases = append(knowledgeBases, kb.ToSummary())
		}
	}

	return &ChatbotRAGConfig{
		Enabled:        len(knowledgeBases) > 0,
		KnowledgeBases: knowledgeBases,
		TotalMaxChunks: totalMaxChunks,
	}, nil
}

// ChatbotRAGConfig represents RAG configuration for a chatbot
type ChatbotRAGConfig struct {
	Enabled        bool                   `json:"enabled"`
	KnowledgeBases []KnowledgeBaseSummary `json:"knowledge_bases"`
	TotalMaxChunks int                    `json:"total_max_chunks"`
}

// IsRAGEnabled checks if a chatbot has RAG enabled
func (r *RAGService) IsRAGEnabled(ctx context.Context, chatbotID string) bool {
	links, err := r.storage.GetChatbotKnowledgeBases(ctx, chatbotID)
	if err != nil {
		return false
	}

	for _, link := range links {
		if link.Enabled {
			return true
		}
	}

	return false
}

// BuildRAGSystemPromptSection builds the RAG section for a system prompt
func (r *RAGService) BuildRAGSystemPromptSection(ctx context.Context, chatbotID, userQuery string) (string, error) {
	return r.BuildRAGSystemPromptSectionWithUser(ctx, chatbotID, userQuery, "")
}

// BuildRAGSystemPromptSectionWithUser builds the RAG section for a system prompt with user context
func (r *RAGService) BuildRAGSystemPromptSectionWithUser(ctx context.Context, chatbotID, userQuery, userID string) (string, error) {
	if !r.IsRAGEnabled(ctx, chatbotID) {
		return "", nil
	}

	result, err := r.RetrieveContext(ctx, RetrieveContextOptions{
		ChatbotID: chatbotID,
		Query:     userQuery,
		UserID:    userID,
	})
	if err != nil {
		log.Warn().Err(err).Str("chatbot_id", chatbotID).Msg("Failed to retrieve RAG context")
		return "", nil // Don't fail the request, just skip RAG
	}

	if result.TotalRetrieved == 0 {
		return "", nil
	}

	return result.FormattedContext, nil
}

// optString returns a pointer to a string, or nil if empty
func optString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// GetKnowledgeBaseStats returns statistics about a knowledge base
func (r *RAGService) GetKnowledgeBaseStats(ctx context.Context, kbID string) (*KnowledgeBaseStats, error) {
	kb, err := r.storage.GetKnowledgeBase(ctx, kbID)
	if err != nil {
		return nil, err
	}
	if kb == nil {
		return nil, fmt.Errorf("knowledge base not found")
	}

	docs, err := r.storage.ListDocuments(ctx, kbID)
	if err != nil {
		return nil, err
	}

	var pendingDocs, indexedDocs, failedDocs int
	for _, doc := range docs {
		switch doc.Status {
		case DocumentStatusPending, DocumentStatusProcessing:
			pendingDocs++
		case DocumentStatusIndexed:
			indexedDocs++
		case DocumentStatusFailed:
			failedDocs++
		}
	}

	return &KnowledgeBaseStats{
		ID:             kb.ID,
		Name:           kb.Name,
		DocumentCount:  kb.DocumentCount,
		TotalChunks:    kb.TotalChunks,
		PendingDocs:    pendingDocs,
		IndexedDocs:    indexedDocs,
		FailedDocs:     failedDocs,
		EmbeddingModel: kb.EmbeddingModel,
		ChunkSize:      kb.ChunkSize,
		ChunkOverlap:   kb.ChunkOverlap,
		ChunkStrategy:  kb.ChunkStrategy,
		Enabled:        kb.Enabled,
		CreatedAt:      kb.CreatedAt,
		UpdatedAt:      kb.UpdatedAt,
	}, nil
}

// KnowledgeBaseStats contains statistics about a knowledge base
type KnowledgeBaseStats struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	DocumentCount  int       `json:"document_count"`
	TotalChunks    int       `json:"total_chunks"`
	PendingDocs    int       `json:"pending_docs"`
	IndexedDocs    int       `json:"indexed_docs"`
	FailedDocs     int       `json:"failed_docs"`
	EmbeddingModel string    `json:"embedding_model"`
	ChunkSize      int       `json:"chunk_size"`
	ChunkOverlap   int       `json:"chunk_overlap"`
	ChunkStrategy  string    `json:"chunk_strategy"`
	Enabled        bool      `json:"enabled"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}
