package ai

import (
	"encoding/json"
	"time"
)

// KnowledgeBase represents a collection of documents for RAG retrieval
type KnowledgeBase struct {
	ID                  string    `json:"id"`
	Name                string    `json:"name"`
	Namespace           string    `json:"namespace"`
	Description         string    `json:"description,omitempty"`
	EmbeddingModel      string    `json:"embedding_model"`
	EmbeddingDimensions int       `json:"embedding_dimensions"`
	ChunkSize           int       `json:"chunk_size"`
	ChunkOverlap        int       `json:"chunk_overlap"`
	ChunkStrategy       string    `json:"chunk_strategy"`
	Enabled             bool      `json:"enabled"`
	DocumentCount       int       `json:"document_count"`
	TotalChunks         int       `json:"total_chunks"`
	Source              string    `json:"source"`
	CreatedBy           *string   `json:"created_by,omitempty"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

// KnowledgeBaseSummary is a lightweight version for listing
type KnowledgeBaseSummary struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Namespace     string `json:"namespace"`
	Description   string `json:"description,omitempty"`
	Enabled       bool   `json:"enabled"`
	DocumentCount int    `json:"document_count"`
	TotalChunks   int    `json:"total_chunks"`
	UpdatedAt     string `json:"updated_at"`
}

// ToSummary converts a KnowledgeBase to a summary
func (kb *KnowledgeBase) ToSummary() KnowledgeBaseSummary {
	return KnowledgeBaseSummary{
		ID:            kb.ID,
		Name:          kb.Name,
		Namespace:     kb.Namespace,
		Description:   kb.Description,
		Enabled:       kb.Enabled,
		DocumentCount: kb.DocumentCount,
		TotalChunks:   kb.TotalChunks,
		UpdatedAt:     kb.UpdatedAt.Format(time.RFC3339),
	}
}

// Document represents a source document in a knowledge base
type Document struct {
	ID              string          `json:"id"`
	KnowledgeBaseID string          `json:"knowledge_base_id"`
	Title           string          `json:"title,omitempty"`
	SourceURL       string          `json:"source_url,omitempty"`
	SourceType      string          `json:"source_type"`
	MimeType        string          `json:"mime_type,omitempty"`
	Content         string          `json:"content"`
	ContentHash     string          `json:"content_hash,omitempty"`
	Status          DocumentStatus  `json:"status"`
	ErrorMessage    string          `json:"error_message,omitempty"`
	ChunksCount     int             `json:"chunks_count"`
	Metadata        json.RawMessage `json:"metadata,omitempty"`
	Tags            []string        `json:"tags,omitempty"`
	CreatedBy       *string         `json:"created_by,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
	IndexedAt       *time.Time      `json:"indexed_at,omitempty"`
}

// DocumentStatus represents the processing status of a document
type DocumentStatus string

const (
	DocumentStatusPending    DocumentStatus = "pending"
	DocumentStatusProcessing DocumentStatus = "processing"
	DocumentStatusIndexed    DocumentStatus = "indexed"
	DocumentStatusFailed     DocumentStatus = "failed"
)

// DocumentSummary is a lightweight version for listing
type DocumentSummary struct {
	ID          string         `json:"id"`
	Title       string         `json:"title,omitempty"`
	SourceType  string         `json:"source_type"`
	Status      DocumentStatus `json:"status"`
	ChunksCount int            `json:"chunks_count"`
	Tags        []string       `json:"tags,omitempty"`
	UpdatedAt   string         `json:"updated_at"`
}

// ToSummary converts a Document to a summary
func (d *Document) ToSummary() DocumentSummary {
	return DocumentSummary{
		ID:          d.ID,
		Title:       d.Title,
		SourceType:  d.SourceType,
		Status:      d.Status,
		ChunksCount: d.ChunksCount,
		Tags:        d.Tags,
		UpdatedAt:   d.UpdatedAt.Format(time.RFC3339),
	}
}

// Chunk represents a document chunk with its embedding
type Chunk struct {
	ID              string          `json:"id"`
	DocumentID      string          `json:"document_id"`
	KnowledgeBaseID string          `json:"knowledge_base_id"`
	Content         string          `json:"content"`
	ChunkIndex      int             `json:"chunk_index"`
	StartOffset     *int            `json:"start_offset,omitempty"`
	EndOffset       *int            `json:"end_offset,omitempty"`
	TokenCount      *int            `json:"token_count,omitempty"`
	Embedding       []float32       `json:"embedding,omitempty"` // Not included in JSON by default
	Metadata        json.RawMessage `json:"metadata,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
}

// ChatbotKnowledgeBase links a chatbot to a knowledge base
type ChatbotKnowledgeBase struct {
	ID                  string    `json:"id"`
	ChatbotID           string    `json:"chatbot_id"`
	KnowledgeBaseID     string    `json:"knowledge_base_id"`
	Enabled             bool      `json:"enabled"`
	MaxChunks           int       `json:"max_chunks"`
	SimilarityThreshold float64   `json:"similarity_threshold"`
	Priority            int       `json:"priority"`
	CreatedAt           time.Time `json:"created_at"`
}

// RetrievalResult represents a single chunk retrieved during RAG
type RetrievalResult struct {
	ChunkID           string          `json:"chunk_id"`
	DocumentID        string          `json:"document_id"`
	KnowledgeBaseID   string          `json:"knowledge_base_id"`
	KnowledgeBaseName string          `json:"knowledge_base_name,omitempty"`
	DocumentTitle     string          `json:"document_title,omitempty"`
	Content           string          `json:"content"`
	Similarity        float64         `json:"similarity"`
	Metadata          json.RawMessage `json:"metadata,omitempty"`
	Tags              []string        `json:"tags,omitempty"`
}

// MetadataFilter for user isolation and tag filtering in vector search
type MetadataFilter struct {
	UserID        *string           // If set, filter to this user's content + global content
	Tags          []string          // Filter by tags (documents must have ALL these tags)
	IncludeGlobal bool              // Include content without user_id (default: true)
	Metadata      map[string]string // Arbitrary key-value filters on document metadata
}

// VectorSearchResult represents a single search result from the vector_search tool
type VectorSearchResult struct {
	ChunkID           string   `json:"chunk_id"`
	DocumentID        string   `json:"document_id"`
	DocumentTitle     string   `json:"document_title,omitempty"`
	KnowledgeBaseName string   `json:"knowledge_base_name"`
	Content           string   `json:"content"`
	Similarity        float64  `json:"similarity"`
	Tags              []string `json:"tags,omitempty"`
}

// VectorSearchOptions contains options for explicit vector search via the tool
type VectorSearchOptions struct {
	ChatbotID      string
	Query          string
	KnowledgeBases []string          // Specific KB names, or empty for all linked
	Limit          int
	Threshold      float64
	Tags           []string
	Metadata       map[string]string // Arbitrary key-value filters on document metadata
	UserID         *string           // For user isolation
	IsAdmin        bool              // Admin can bypass user filter
}

// RetrievalLog records a RAG retrieval operation
type RetrievalLog struct {
	ID                  string    `json:"id"`
	ChatbotID           *string   `json:"chatbot_id,omitempty"`
	ConversationID      *string   `json:"conversation_id,omitempty"`
	KnowledgeBaseID     *string   `json:"knowledge_base_id,omitempty"`
	UserID              *string   `json:"user_id,omitempty"`
	QueryText           string    `json:"query_text"`
	QueryEmbeddingModel string    `json:"query_embedding_model,omitempty"`
	ChunksRetrieved     int       `json:"chunks_retrieved"`
	ChunkIDs            []string  `json:"chunk_ids,omitempty"`
	SimilarityScores    []float64 `json:"similarity_scores,omitempty"`
	RetrievalDurationMs int       `json:"retrieval_duration_ms"`
	CreatedAt           time.Time `json:"created_at"`
}

// CreateKnowledgeBaseRequest is the request to create a knowledge base
type CreateKnowledgeBaseRequest struct {
	Name                string `json:"name"`
	Namespace           string `json:"namespace,omitempty"`
	Description         string `json:"description,omitempty"`
	EmbeddingModel      string `json:"embedding_model,omitempty"`
	EmbeddingDimensions int    `json:"embedding_dimensions,omitempty"`
	ChunkSize           int    `json:"chunk_size,omitempty"`
	ChunkOverlap        int    `json:"chunk_overlap,omitempty"`
	ChunkStrategy       string `json:"chunk_strategy,omitempty"`
}

// UpdateKnowledgeBaseRequest is the request to update a knowledge base
type UpdateKnowledgeBaseRequest struct {
	Name                *string `json:"name,omitempty"`
	Description         *string `json:"description,omitempty"`
	EmbeddingModel      *string `json:"embedding_model,omitempty"`
	EmbeddingDimensions *int    `json:"embedding_dimensions,omitempty"`
	ChunkSize           *int    `json:"chunk_size,omitempty"`
	ChunkOverlap        *int    `json:"chunk_overlap,omitempty"`
	ChunkStrategy       *string `json:"chunk_strategy,omitempty"`
	Enabled             *bool   `json:"enabled,omitempty"`
}

// CreateDocumentRequest is the request to add a document to a knowledge base
type CreateDocumentRequest struct {
	Title            string            `json:"title,omitempty"`
	Content          string            `json:"content"`
	SourceURL        string            `json:"source_url,omitempty"`
	SourceType       string            `json:"source_type,omitempty"`
	MimeType         string            `json:"mime_type,omitempty"`
	Metadata         map[string]string `json:"metadata,omitempty"`
	Tags             []string          `json:"tags,omitempty"`
	OriginalFilename string            `json:"original_filename,omitempty"`
}

// LinkKnowledgeBaseRequest is the request to link a knowledge base to a chatbot
type LinkKnowledgeBaseRequest struct {
	KnowledgeBaseID     string   `json:"knowledge_base_id"`
	MaxChunks           *int     `json:"max_chunks,omitempty"`
	SimilarityThreshold *float64 `json:"similarity_threshold,omitempty"`
	Priority            *int     `json:"priority,omitempty"`
}

// ChunkingStrategy defines the strategy for splitting documents
type ChunkingStrategy string

const (
	ChunkingStrategyRecursive ChunkingStrategy = "recursive"
	ChunkingStrategySentence  ChunkingStrategy = "sentence"
	ChunkingStrategyParagraph ChunkingStrategy = "paragraph"
	ChunkingStrategyFixed     ChunkingStrategy = "fixed"
)

// DefaultKnowledgeBaseConfig returns default configuration
func DefaultKnowledgeBaseConfig() CreateKnowledgeBaseRequest {
	return CreateKnowledgeBaseRequest{
		Namespace:           "default",
		EmbeddingModel:      "text-embedding-3-small",
		EmbeddingDimensions: 1536,
		ChunkSize:           512,
		ChunkOverlap:        50,
		ChunkStrategy:       string(ChunkingStrategyRecursive),
	}
}
