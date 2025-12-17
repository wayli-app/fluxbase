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
}

// NewRAGService creates a new RAG service
func NewRAGService(storage *KnowledgeBaseStorage, embeddingService *EmbeddingService) *RAGService {
	return &RAGService{
		storage:          storage,
		embeddingService: embeddingService,
	}
}

// RetrieveContextOptions contains options for retrieval
type RetrieveContextOptions struct {
	ChatbotID      string
	ConversationID string
	UserID         string
	Query          string
	MaxChunks      int     // Override max chunks (optional)
	Threshold      float64 // Override threshold (optional)
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

	// Search for relevant chunks
	chunks, err := r.storage.SearchChatbotKnowledge(ctx, opts.ChatbotID, queryEmbedding)
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

	r.storage.LogRetrieval(ctx, &RetrievalLog{
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

// formatContext formats retrieved chunks into a string for the LLM prompt
func (r *RAGService) formatContext(chunks []RetrievalResult) string {
	if len(chunks) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("## Relevant Knowledge\n\n")
	sb.WriteString("The following information was retrieved from the knowledge base and may be relevant to the user's question:\n\n")

	for i, chunk := range chunks {
		sb.WriteString(fmt.Sprintf("### Source %d", i+1))
		if chunk.DocumentTitle != "" {
			sb.WriteString(fmt.Sprintf(": %s", chunk.DocumentTitle))
		}
		if chunk.KnowledgeBaseName != "" {
			sb.WriteString(fmt.Sprintf(" (from %s)", chunk.KnowledgeBaseName))
		}
		sb.WriteString(fmt.Sprintf(" [similarity: %.2f]\n\n", chunk.Similarity))
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
		totalMaxChunks += link.MaxChunks

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
	if !r.IsRAGEnabled(ctx, chatbotID) {
		return "", nil
	}

	result, err := r.RetrieveContext(ctx, RetrieveContextOptions{
		ChatbotID: chatbotID,
		Query:     userQuery,
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
		ID:              kb.ID,
		Name:            kb.Name,
		DocumentCount:   kb.DocumentCount,
		TotalChunks:     kb.TotalChunks,
		PendingDocs:     pendingDocs,
		IndexedDocs:     indexedDocs,
		FailedDocs:      failedDocs,
		EmbeddingModel:  kb.EmbeddingModel,
		ChunkSize:       kb.ChunkSize,
		ChunkOverlap:    kb.ChunkOverlap,
		ChunkStrategy:   kb.ChunkStrategy,
		Enabled:         kb.Enabled,
		CreatedAt:       kb.CreatedAt,
		UpdatedAt:       kb.UpdatedAt,
	}, nil
}

// KnowledgeBaseStats contains statistics about a knowledge base
type KnowledgeBaseStats struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	DocumentCount   int       `json:"document_count"`
	TotalChunks     int       `json:"total_chunks"`
	PendingDocs     int       `json:"pending_docs"`
	IndexedDocs     int       `json:"indexed_docs"`
	FailedDocs      int       `json:"failed_docs"`
	EmbeddingModel  string    `json:"embedding_model"`
	ChunkSize       int       `json:"chunk_size"`
	ChunkOverlap    int       `json:"chunk_overlap"`
	ChunkStrategy   string    `json:"chunk_strategy"`
	Enabled         bool      `json:"enabled"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}
