package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/fluxbase-eu/fluxbase/internal/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"
)

// KnowledgeBaseStorage handles database operations for knowledge bases
type KnowledgeBaseStorage struct {
	db *database.Connection
}

// NewKnowledgeBaseStorage creates a new knowledge base storage
func NewKnowledgeBaseStorage(db *database.Connection) *KnowledgeBaseStorage {
	return &KnowledgeBaseStorage{db: db}
}

// ============================================================================
// Knowledge Base CRUD
// ============================================================================

// CreateKnowledgeBase creates a new knowledge base
func (s *KnowledgeBaseStorage) CreateKnowledgeBase(ctx context.Context, kb *KnowledgeBase) error {
	if kb.ID == "" {
		kb.ID = uuid.New().String()
	}
	kb.CreatedAt = time.Now()
	kb.UpdatedAt = time.Now()

	query := `
		INSERT INTO ai.knowledge_bases (
			id, name, namespace, description,
			embedding_model, embedding_dimensions,
			chunk_size, chunk_overlap, chunk_strategy,
			enabled, source, created_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING created_at, updated_at
	`

	return s.db.QueryRow(ctx, query,
		kb.ID, kb.Name, kb.Namespace, kb.Description,
		kb.EmbeddingModel, kb.EmbeddingDimensions,
		kb.ChunkSize, kb.ChunkOverlap, kb.ChunkStrategy,
		kb.Enabled, kb.Source, kb.CreatedBy,
	).Scan(&kb.CreatedAt, &kb.UpdatedAt)
}

// GetKnowledgeBase retrieves a knowledge base by ID
func (s *KnowledgeBaseStorage) GetKnowledgeBase(ctx context.Context, id string) (*KnowledgeBase, error) {
	query := `
		SELECT id, name, namespace, description,
			embedding_model, embedding_dimensions,
			chunk_size, chunk_overlap, chunk_strategy,
			enabled, document_count, total_chunks,
			source, created_by, created_at, updated_at
		FROM ai.knowledge_bases
		WHERE id = $1
	`

	var kb KnowledgeBase
	err := s.db.QueryRow(ctx, query, id).Scan(
		&kb.ID, &kb.Name, &kb.Namespace, &kb.Description,
		&kb.EmbeddingModel, &kb.EmbeddingDimensions,
		&kb.ChunkSize, &kb.ChunkOverlap, &kb.ChunkStrategy,
		&kb.Enabled, &kb.DocumentCount, &kb.TotalChunks,
		&kb.Source, &kb.CreatedBy, &kb.CreatedAt, &kb.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get knowledge base: %w", err)
	}
	return &kb, nil
}

// GetKnowledgeBaseByName retrieves a knowledge base by name and namespace
func (s *KnowledgeBaseStorage) GetKnowledgeBaseByName(ctx context.Context, name, namespace string) (*KnowledgeBase, error) {
	query := `
		SELECT id, name, namespace, description,
			embedding_model, embedding_dimensions,
			chunk_size, chunk_overlap, chunk_strategy,
			enabled, document_count, total_chunks,
			source, created_by, created_at, updated_at
		FROM ai.knowledge_bases
		WHERE name = $1 AND namespace = $2
	`

	var kb KnowledgeBase
	err := s.db.QueryRow(ctx, query, name, namespace).Scan(
		&kb.ID, &kb.Name, &kb.Namespace, &kb.Description,
		&kb.EmbeddingModel, &kb.EmbeddingDimensions,
		&kb.ChunkSize, &kb.ChunkOverlap, &kb.ChunkStrategy,
		&kb.Enabled, &kb.DocumentCount, &kb.TotalChunks,
		&kb.Source, &kb.CreatedBy, &kb.CreatedAt, &kb.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get knowledge base by name: %w", err)
	}
	return &kb, nil
}

// ListKnowledgeBases lists knowledge bases with optional filtering
func (s *KnowledgeBaseStorage) ListKnowledgeBases(ctx context.Context, namespace string, enabledOnly bool) ([]KnowledgeBase, error) {
	query := `
		SELECT id, name, namespace, description,
			embedding_model, embedding_dimensions,
			chunk_size, chunk_overlap, chunk_strategy,
			enabled, document_count, total_chunks,
			source, created_by, created_at, updated_at
		FROM ai.knowledge_bases
		WHERE ($1 = '' OR namespace = $1)
		  AND ($2 = false OR enabled = true)
		ORDER BY namespace, name
	`

	rows, err := s.db.Query(ctx, query, namespace, enabledOnly)
	if err != nil {
		return nil, fmt.Errorf("failed to list knowledge bases: %w", err)
	}
	defer rows.Close()

	var kbs []KnowledgeBase
	for rows.Next() {
		var kb KnowledgeBase
		if err := rows.Scan(
			&kb.ID, &kb.Name, &kb.Namespace, &kb.Description,
			&kb.EmbeddingModel, &kb.EmbeddingDimensions,
			&kb.ChunkSize, &kb.ChunkOverlap, &kb.ChunkStrategy,
			&kb.Enabled, &kb.DocumentCount, &kb.TotalChunks,
			&kb.Source, &kb.CreatedBy, &kb.CreatedAt, &kb.UpdatedAt,
		); err != nil {
			log.Warn().Err(err).Msg("Failed to scan knowledge base row")
			continue
		}
		kbs = append(kbs, kb)
	}

	return kbs, nil
}

// UpdateKnowledgeBase updates a knowledge base
func (s *KnowledgeBaseStorage) UpdateKnowledgeBase(ctx context.Context, kb *KnowledgeBase) error {
	query := `
		UPDATE ai.knowledge_bases SET
			name = $2, description = $3,
			embedding_model = $4, embedding_dimensions = $5,
			chunk_size = $6, chunk_overlap = $7, chunk_strategy = $8,
			enabled = $9, updated_at = NOW()
		WHERE id = $1
		RETURNING updated_at
	`

	return s.db.QueryRow(ctx, query,
		kb.ID, kb.Name, kb.Description,
		kb.EmbeddingModel, kb.EmbeddingDimensions,
		kb.ChunkSize, kb.ChunkOverlap, kb.ChunkStrategy,
		kb.Enabled,
	).Scan(&kb.UpdatedAt)
}

// DeleteKnowledgeBase deletes a knowledge base and all its documents/chunks
func (s *KnowledgeBaseStorage) DeleteKnowledgeBase(ctx context.Context, id string) error {
	_, err := s.db.Exec(ctx, "DELETE FROM ai.knowledge_bases WHERE id = $1", id)
	return err
}

// ============================================================================
// Document CRUD
// ============================================================================

// CreateDocument creates a new document in a knowledge base
func (s *KnowledgeBaseStorage) CreateDocument(ctx context.Context, doc *Document) error {
	if doc.ID == "" {
		doc.ID = uuid.New().String()
	}
	doc.CreatedAt = time.Now()
	doc.UpdatedAt = time.Now()
	doc.Status = DocumentStatusPending

	// Marshal metadata if present
	var metadataJSON []byte
	if doc.Metadata != nil {
		metadataJSON = doc.Metadata
	}

	query := `
		INSERT INTO ai.documents (
			id, knowledge_base_id, title, source_url, source_type,
			mime_type, content, content_hash, status, metadata, tags, created_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING created_at, updated_at
	`

	return s.db.QueryRow(ctx, query,
		doc.ID, doc.KnowledgeBaseID, doc.Title, doc.SourceURL, doc.SourceType,
		doc.MimeType, doc.Content, doc.ContentHash, doc.Status, metadataJSON, doc.Tags, doc.CreatedBy,
	).Scan(&doc.CreatedAt, &doc.UpdatedAt)
}

// GetDocument retrieves a document by ID
func (s *KnowledgeBaseStorage) GetDocument(ctx context.Context, id string) (*Document, error) {
	query := `
		SELECT id, knowledge_base_id, title, source_url, source_type,
			mime_type, content, content_hash, status, error_message,
			chunks_count, metadata, tags, created_by, created_at, updated_at, indexed_at
		FROM ai.documents
		WHERE id = $1
	`

	var doc Document
	err := s.db.QueryRow(ctx, query, id).Scan(
		&doc.ID, &doc.KnowledgeBaseID, &doc.Title, &doc.SourceURL, &doc.SourceType,
		&doc.MimeType, &doc.Content, &doc.ContentHash, &doc.Status, &doc.ErrorMessage,
		&doc.ChunksCount, &doc.Metadata, &doc.Tags, &doc.CreatedBy, &doc.CreatedAt, &doc.UpdatedAt, &doc.IndexedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get document: %w", err)
	}
	return &doc, nil
}

// ListDocuments lists documents in a knowledge base
func (s *KnowledgeBaseStorage) ListDocuments(ctx context.Context, knowledgeBaseID string) ([]Document, error) {
	query := `
		SELECT id, knowledge_base_id, title, source_url, source_type,
			mime_type, content, content_hash, status, error_message,
			chunks_count, metadata, tags, created_by, created_at, updated_at, indexed_at
		FROM ai.documents
		WHERE knowledge_base_id = $1
		ORDER BY created_at DESC
	`

	rows, err := s.db.Query(ctx, query, knowledgeBaseID)
	if err != nil {
		return nil, fmt.Errorf("failed to list documents: %w", err)
	}
	defer rows.Close()

	var docs []Document
	for rows.Next() {
		var doc Document
		if err := rows.Scan(
			&doc.ID, &doc.KnowledgeBaseID, &doc.Title, &doc.SourceURL, &doc.SourceType,
			&doc.MimeType, &doc.Content, &doc.ContentHash, &doc.Status, &doc.ErrorMessage,
			&doc.ChunksCount, &doc.Metadata, &doc.Tags, &doc.CreatedBy, &doc.CreatedAt, &doc.UpdatedAt, &doc.IndexedAt,
		); err != nil {
			log.Warn().Err(err).Msg("Failed to scan document row")
			continue
		}
		docs = append(docs, doc)
	}

	return docs, nil
}

// UpdateDocumentStatus updates a document's processing status
func (s *KnowledgeBaseStorage) UpdateDocumentStatus(ctx context.Context, id string, status DocumentStatus, errorMsg string) error {
	query := `
		UPDATE ai.documents SET
			status = $2, error_message = $3, updated_at = NOW()
		WHERE id = $1
	`
	_, err := s.db.Exec(ctx, query, id, status, errorMsg)
	return err
}

// MarkDocumentIndexed marks a document as indexed
func (s *KnowledgeBaseStorage) MarkDocumentIndexed(ctx context.Context, id string) error {
	query := `
		UPDATE ai.documents SET
			status = 'indexed', indexed_at = NOW(), updated_at = NOW()
		WHERE id = $1
	`
	_, err := s.db.Exec(ctx, query, id)
	return err
}

// DeleteDocument deletes a document and its chunks
func (s *KnowledgeBaseStorage) DeleteDocument(ctx context.Context, id string) error {
	_, err := s.db.Exec(ctx, "DELETE FROM ai.documents WHERE id = $1", id)
	return err
}

// UpdateDocumentMetadata updates a document's title, metadata, and tags
func (s *KnowledgeBaseStorage) UpdateDocumentMetadata(ctx context.Context, id string, title *string, metadata map[string]string, tags []string) (*Document, error) {
	// Build the metadata JSON
	var metadataJSON []byte
	if metadata != nil {
		var err error
		metadataJSON, err = json.Marshal(metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	query := `
		UPDATE ai.documents SET
			title = COALESCE($2, title),
			metadata = COALESCE($3, metadata),
			tags = COALESCE($4, tags),
			updated_at = NOW()
		WHERE id = $1
		RETURNING id, knowledge_base_id, title, source_url, source_type,
			mime_type, content, content_hash, status, error_message,
			chunks_count, metadata, tags, created_by, created_at, updated_at, indexed_at
	`

	var doc Document
	err := s.db.QueryRow(ctx, query, id, title, metadataJSON, tags).Scan(
		&doc.ID, &doc.KnowledgeBaseID, &doc.Title, &doc.SourceURL, &doc.SourceType,
		&doc.MimeType, &doc.Content, &doc.ContentHash, &doc.Status, &doc.ErrorMessage,
		&doc.ChunksCount, &doc.Metadata, &doc.Tags, &doc.CreatedBy, &doc.CreatedAt, &doc.UpdatedAt, &doc.IndexedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update document: %w", err)
	}

	return &doc, nil
}

// ============================================================================
// Chunk Operations
// ============================================================================

// CreateChunks creates multiple chunks for a document (batch insert)
func (s *KnowledgeBaseStorage) CreateChunks(ctx context.Context, chunks []Chunk) error {
	if len(chunks) == 0 {
		return nil
	}

	// Use COPY for efficient bulk insert
	batch := &pgx.Batch{}
	for _, chunk := range chunks {
		if chunk.ID == "" {
			chunk.ID = uuid.New().String()
		}

		var metadataJSON []byte
		if chunk.Metadata != nil {
			metadataJSON = chunk.Metadata
		}

		// Format embedding as PostgreSQL vector literal (pgx can't encode []float32 directly)
		var embeddingExpr string
		if chunk.Embedding != nil {
			embeddingExpr = fmt.Sprintf("'%s'::vector", formatEmbeddingLiteral(chunk.Embedding))
		} else {
			embeddingExpr = "NULL"
		}

		query := fmt.Sprintf(`
			INSERT INTO ai.chunks (
				id, document_id, knowledge_base_id, content,
				chunk_index, start_offset, end_offset, token_count,
				embedding, metadata
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, %s, $9)
		`, embeddingExpr)

		batch.Queue(query,
			chunk.ID, chunk.DocumentID, chunk.KnowledgeBaseID, chunk.Content,
			chunk.ChunkIndex, chunk.StartOffset, chunk.EndOffset, chunk.TokenCount,
			metadataJSON,
		)
	}

	br := s.db.Pool().SendBatch(ctx, batch)
	defer br.Close()

	for range chunks {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("failed to insert chunk: %w", err)
		}
	}

	return nil
}

// GetChunksByDocument retrieves all chunks for a document
func (s *KnowledgeBaseStorage) GetChunksByDocument(ctx context.Context, documentID string) ([]Chunk, error) {
	query := `
		SELECT id, document_id, knowledge_base_id, content,
			chunk_index, start_offset, end_offset, token_count, metadata, created_at
		FROM ai.chunks
		WHERE document_id = $1
		ORDER BY chunk_index
	`

	rows, err := s.db.Query(ctx, query, documentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get chunks: %w", err)
	}
	defer rows.Close()

	var chunks []Chunk
	for rows.Next() {
		var chunk Chunk
		if err := rows.Scan(
			&chunk.ID, &chunk.DocumentID, &chunk.KnowledgeBaseID, &chunk.Content,
			&chunk.ChunkIndex, &chunk.StartOffset, &chunk.EndOffset, &chunk.TokenCount,
			&chunk.Metadata, &chunk.CreatedAt,
		); err != nil {
			log.Warn().Err(err).Msg("Failed to scan chunk row")
			continue
		}
		chunks = append(chunks, chunk)
	}

	return chunks, nil
}

// DeleteChunksByDocument deletes all chunks for a document
func (s *KnowledgeBaseStorage) DeleteChunksByDocument(ctx context.Context, documentID string) error {
	_, err := s.db.Exec(ctx, "DELETE FROM ai.chunks WHERE document_id = $1", documentID)
	return err
}

// ============================================================================
// Chatbot Knowledge Base Links
// ============================================================================

// LinkChatbotKnowledgeBase links a chatbot to a knowledge base
func (s *KnowledgeBaseStorage) LinkChatbotKnowledgeBase(ctx context.Context, link *ChatbotKnowledgeBase) error {
	if link.ID == "" {
		link.ID = uuid.New().String()
	}
	link.CreatedAt = time.Now()

	query := `
		INSERT INTO ai.chatbot_knowledge_bases (
			id, chatbot_id, knowledge_base_id, enabled,
			max_chunks, similarity_threshold, priority
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (chatbot_id, knowledge_base_id) DO UPDATE SET
			enabled = EXCLUDED.enabled,
			max_chunks = EXCLUDED.max_chunks,
			similarity_threshold = EXCLUDED.similarity_threshold,
			priority = EXCLUDED.priority
		RETURNING created_at
	`

	return s.db.QueryRow(ctx, query,
		link.ID, link.ChatbotID, link.KnowledgeBaseID, link.Enabled,
		link.MaxChunks, link.SimilarityThreshold, link.Priority,
	).Scan(&link.CreatedAt)
}

// GetChatbotKnowledgeBases retrieves all knowledge base links for a chatbot
func (s *KnowledgeBaseStorage) GetChatbotKnowledgeBases(ctx context.Context, chatbotID string) ([]ChatbotKnowledgeBase, error) {
	query := `
		SELECT id, chatbot_id, knowledge_base_id, enabled,
			max_chunks, similarity_threshold, priority, created_at
		FROM ai.chatbot_knowledge_bases
		WHERE chatbot_id = $1
		ORDER BY priority DESC
	`

	rows, err := s.db.Query(ctx, query, chatbotID)
	if err != nil {
		return nil, fmt.Errorf("failed to get chatbot knowledge bases: %w", err)
	}
	defer rows.Close()

	var links []ChatbotKnowledgeBase
	for rows.Next() {
		var link ChatbotKnowledgeBase
		if err := rows.Scan(
			&link.ID, &link.ChatbotID, &link.KnowledgeBaseID, &link.Enabled,
			&link.MaxChunks, &link.SimilarityThreshold, &link.Priority, &link.CreatedAt,
		); err != nil {
			log.Warn().Err(err).Msg("Failed to scan chatbot knowledge base link")
			continue
		}
		links = append(links, link)
	}

	return links, nil
}

// UnlinkChatbotKnowledgeBase removes a link between chatbot and knowledge base
func (s *KnowledgeBaseStorage) UnlinkChatbotKnowledgeBase(ctx context.Context, chatbotID, knowledgeBaseID string) error {
	_, err := s.db.Exec(ctx,
		"DELETE FROM ai.chatbot_knowledge_bases WHERE chatbot_id = $1 AND knowledge_base_id = $2",
		chatbotID, knowledgeBaseID,
	)
	return err
}

// ============================================================================
// Vector Search / Retrieval
// ============================================================================

// SearchChunks searches for similar chunks in a knowledge base
func (s *KnowledgeBaseStorage) SearchChunks(ctx context.Context, knowledgeBaseID string, queryEmbedding []float32, limit int, threshold float64) ([]RetrievalResult, error) {
	// Format embedding as PostgreSQL vector literal
	embeddingStr := formatEmbeddingLiteral(queryEmbedding)

	// Log embedding info for debugging
	embeddingPreview := embeddingStr
	if len(embeddingPreview) > 100 {
		embeddingPreview = embeddingPreview[:100] + "..."
	}
	log.Debug().
		Int("embedding_length", len(queryEmbedding)).
		Str("kb_id", knowledgeBaseID).
		Float64("threshold", threshold).
		Int("limit", limit).
		Str("embedding_preview", embeddingPreview).
		Msg("SearchChunks starting")

	query := fmt.Sprintf(`
		SELECT
			c.id as chunk_id,
			c.document_id,
			c.content,
			1 - (c.embedding <=> '%s'::vector) as similarity,
			c.metadata,
			d.title as document_title
		FROM ai.chunks c
		JOIN ai.documents d ON d.id = c.document_id
		WHERE c.knowledge_base_id = $1
		  AND 1 - (c.embedding <=> '%s'::vector) >= $2
		ORDER BY c.embedding <=> '%s'::vector
		LIMIT $3
	`, embeddingStr, embeddingStr, embeddingStr)

	rows, err := s.db.Query(ctx, query, knowledgeBaseID, threshold, limit)
	if err != nil {
		log.Error().Err(err).Str("kb_id", knowledgeBaseID).Msg("SearchChunks query failed")
		return nil, fmt.Errorf("failed to search chunks: %w", err)
	}
	defer rows.Close()

	var results []RetrievalResult
	for rows.Next() {
		var r RetrievalResult
		var docTitle *string
		if err := rows.Scan(&r.ChunkID, &r.DocumentID, &r.Content, &r.Similarity, &r.Metadata, &docTitle); err != nil {
			log.Warn().Err(err).Msg("Failed to scan search result")
			continue
		}
		r.KnowledgeBaseID = knowledgeBaseID
		if docTitle != nil {
			r.DocumentTitle = *docTitle
		}
		results = append(results, r)
	}

	// Log results
	if len(results) > 0 {
		log.Debug().
			Int("results_count", len(results)).
			Float64("top_similarity", results[0].Similarity).
			Str("kb_id", knowledgeBaseID).
			Msg("SearchChunks completed")
	} else {
		log.Debug().
			Str("kb_id", knowledgeBaseID).
			Float64("threshold", threshold).
			Msg("SearchChunks returned no results")
	}

	return results, nil
}

// SearchMode defines how search should be performed
type SearchMode string

const (
	SearchModeSemantic SearchMode = "semantic" // Vector similarity only
	SearchModeKeyword  SearchMode = "keyword"  // Full-text search only
	SearchModeHybrid   SearchMode = "hybrid"   // Combined vector + full-text
)

// HybridSearchOptions contains options for hybrid search
type HybridSearchOptions struct {
	Query           string
	QueryEmbedding  []float32
	Limit           int
	Threshold       float64
	Mode            SearchMode
	SemanticWeight  float64         // Weight for semantic score (0-1), keyword weight = 1 - semantic
	KeywordBoost    float64         // Boost factor for exact keyword matches
	Filter          *MetadataFilter // Optional metadata filter for user isolation
}

// SearchChunksHybrid performs hybrid search combining vector similarity with full-text search
func (s *KnowledgeBaseStorage) SearchChunksHybrid(ctx context.Context, knowledgeBaseID string, opts HybridSearchOptions) ([]RetrievalResult, error) {
	// Default weights
	if opts.SemanticWeight == 0 {
		opts.SemanticWeight = 0.5 // 50/50 by default
	}
	if opts.KeywordBoost == 0 {
		opts.KeywordBoost = 0.3 // 30% boost for keyword matches
	}

	log.Debug().
		Str("mode", string(opts.Mode)).
		Str("query", opts.Query).
		Float64("semantic_weight", opts.SemanticWeight).
		Float64("threshold", opts.Threshold).
		Msg("SearchChunksHybrid starting")

	switch opts.Mode {
	case SearchModeKeyword:
		return s.searchKeywordOnly(ctx, knowledgeBaseID, opts)
	case SearchModeHybrid:
		return s.searchHybrid(ctx, knowledgeBaseID, opts)
	default: // SearchModeSemantic
		return s.SearchChunks(ctx, knowledgeBaseID, opts.QueryEmbedding, opts.Limit, opts.Threshold)
	}
}

// searchKeywordOnly performs full-text search only
func (s *KnowledgeBaseStorage) searchKeywordOnly(ctx context.Context, knowledgeBaseID string, opts HybridSearchOptions) ([]RetrievalResult, error) {
	// Prepare the search query for PostgreSQL full-text search
	// Use plainto_tsquery for simple word matching, or websearch_to_tsquery for more advanced
	query := `
		SELECT
			c.id as chunk_id,
			c.document_id,
			c.content,
			ts_rank_cd(to_tsvector('simple', c.content), plainto_tsquery('simple', $2)) as similarity,
			c.metadata,
			d.title as document_title
		FROM ai.chunks c
		JOIN ai.documents d ON d.id = c.document_id
		WHERE c.knowledge_base_id = $1
		  AND (
		    to_tsvector('simple', c.content) @@ plainto_tsquery('simple', $2)
		    OR c.content ILIKE '%' || $2 || '%'
		  )
		ORDER BY similarity DESC
		LIMIT $3
	`

	rows, err := s.db.Query(ctx, query, knowledgeBaseID, opts.Query, opts.Limit)
	if err != nil {
		log.Error().Err(err).Str("kb_id", knowledgeBaseID).Msg("Keyword search query failed")
		return nil, fmt.Errorf("failed to search chunks: %w", err)
	}
	defer rows.Close()

	var results []RetrievalResult
	for rows.Next() {
		var r RetrievalResult
		var docTitle *string
		if err := rows.Scan(&r.ChunkID, &r.DocumentID, &r.Content, &r.Similarity, &r.Metadata, &docTitle); err != nil {
			log.Warn().Err(err).Msg("Failed to scan keyword search result")
			continue
		}
		r.KnowledgeBaseID = knowledgeBaseID
		if docTitle != nil {
			r.DocumentTitle = *docTitle
		}
		// Normalize similarity to 0-1 range (ts_rank_cd can exceed 1)
		if r.Similarity > 1 {
			r.Similarity = 1
		}
		results = append(results, r)
	}

	log.Debug().
		Int("results_count", len(results)).
		Str("kb_id", knowledgeBaseID).
		Msg("Keyword search completed")

	return results, nil
}

// searchHybrid combines vector similarity with full-text search
func (s *KnowledgeBaseStorage) searchHybrid(ctx context.Context, knowledgeBaseID string, opts HybridSearchOptions) ([]RetrievalResult, error) {
	embeddingStr := formatEmbeddingLiteral(opts.QueryEmbedding)
	keywordWeight := 1 - opts.SemanticWeight

	// Build dynamic filter conditions for user isolation
	filterConditions := ""
	args := []interface{}{knowledgeBaseID, opts.Query, opts.SemanticWeight, keywordWeight, opts.KeywordBoost, opts.Threshold, opts.Limit}
	argIndex := 8

	if opts.Filter != nil && opts.Filter.UserID != nil {
		// Include user's content OR content without user_id (global)
		filterConditions += fmt.Sprintf(` AND (
			d.metadata->>'user_id' = $%d OR
			d.metadata->>'user_id' IS NULL OR
			NOT (d.metadata ? 'user_id')
		)`, argIndex)
		args = append(args, *opts.Filter.UserID)
		argIndex++
	}

	if opts.Filter != nil && len(opts.Filter.Tags) > 0 {
		filterConditions += fmt.Sprintf(" AND d.tags @> $%d", argIndex)
		args = append(args, opts.Filter.Tags)
		argIndex++
	}

	// Apply arbitrary metadata filters
	if opts.Filter != nil && len(opts.Filter.Metadata) > 0 {
		for key, value := range opts.Filter.Metadata {
			// Use parameterized value but key must be sanitized (alphanumeric + underscore only)
			safeKey := sanitizeMetadataKey(key)
			filterConditions += fmt.Sprintf(" AND d.metadata->>'%s' = $%d", safeKey, argIndex)
			args = append(args, value)
			argIndex++
		}
	}

	// Hybrid query combining vector similarity and full-text search
	// The final score is: (semantic_weight * vector_similarity) + (keyword_weight * text_rank) + keyword_boost_if_match
	query := fmt.Sprintf(`
		WITH vector_search AS (
			SELECT
				c.id as chunk_id,
				c.document_id,
				c.content,
				c.metadata,
				1 - (c.embedding <=> '%s'::vector) as vector_similarity
			FROM ai.chunks c
			WHERE c.knowledge_base_id = $1
			  AND c.embedding IS NOT NULL
		),
		text_search AS (
			SELECT
				c.id as chunk_id,
				ts_rank_cd(to_tsvector('simple', c.content), plainto_tsquery('simple', $2)) as text_rank,
				CASE
					WHEN c.content ILIKE '%%' || $2 || '%%' THEN $5::float
					ELSE 0
				END as keyword_boost
			FROM ai.chunks c
			WHERE c.knowledge_base_id = $1
		)
		SELECT
			v.chunk_id,
			v.document_id,
			v.content,
			(($3::float * v.vector_similarity) + ($4::float * COALESCE(t.text_rank, 0)) + COALESCE(t.keyword_boost, 0)) as similarity,
			v.metadata,
			d.title as document_title,
			d.tags,
			v.vector_similarity,
			COALESCE(t.text_rank, 0) as text_rank
		FROM vector_search v
		JOIN ai.documents d ON d.id = v.document_id
		LEFT JOIN text_search t ON t.chunk_id = v.chunk_id
		WHERE (($3::float * v.vector_similarity) + ($4::float * COALESCE(t.text_rank, 0)) + COALESCE(t.keyword_boost, 0)) >= $6
		%s
		ORDER BY similarity DESC
		LIMIT $7
	`, embeddingStr, filterConditions)

	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		log.Error().Err(err).Str("kb_id", knowledgeBaseID).Msg("Hybrid search query failed")
		return nil, fmt.Errorf("failed to search chunks: %w", err)
	}
	defer rows.Close()

	var results []RetrievalResult
	for rows.Next() {
		var r RetrievalResult
		var docTitle *string
		var tags []string
		var vectorSim, textRank float64
		if err := rows.Scan(&r.ChunkID, &r.DocumentID, &r.Content, &r.Similarity, &r.Metadata, &docTitle, &tags, &vectorSim, &textRank); err != nil {
			log.Warn().Err(err).Msg("Failed to scan hybrid search result")
			continue
		}
		r.KnowledgeBaseID = knowledgeBaseID
		if docTitle != nil {
			r.DocumentTitle = *docTitle
		}
		r.Tags = tags

		log.Debug().
			Str("chunk_id", r.ChunkID).
			Float64("vector_sim", vectorSim).
			Float64("text_rank", textRank).
			Float64("combined", r.Similarity).
			Msg("Hybrid result")

		results = append(results, r)
	}

	log.Debug().
		Int("results_count", len(results)).
		Str("kb_id", knowledgeBaseID).
		Msg("Hybrid search completed")

	return results, nil
}

// SearchChunksWithFilter searches for similar chunks with metadata filtering for user isolation
func (s *KnowledgeBaseStorage) SearchChunksWithFilter(
	ctx context.Context,
	knowledgeBaseID string,
	queryEmbedding []float32,
	limit int,
	threshold float64,
	filter *MetadataFilter,
) ([]RetrievalResult, error) {
	// Format embedding as PostgreSQL vector literal
	embeddingStr := formatEmbeddingLiteral(queryEmbedding)

	// Build dynamic WHERE clause for filtering
	whereConditions := []string{
		"c.knowledge_base_id = $1",
		fmt.Sprintf("1 - (c.embedding <=> '%s'::vector) >= $2", embeddingStr),
	}
	args := []interface{}{knowledgeBaseID, threshold, limit}
	argIndex := 4

	// User isolation filter
	if filter != nil && filter.UserID != nil {
		// Include user's content OR content without user_id (global)
		whereConditions = append(whereConditions, fmt.Sprintf(`(
			d.metadata->>'user_id' = $%d OR
			d.metadata->>'user_id' IS NULL OR
			NOT (d.metadata ? 'user_id')
		)`, argIndex))
		args = append(args, *filter.UserID)
		argIndex++
	}

	// Tag filter - documents must have ALL specified tags
	if filter != nil && len(filter.Tags) > 0 {
		whereConditions = append(whereConditions, fmt.Sprintf("d.tags @> $%d", argIndex))
		args = append(args, filter.Tags)
		argIndex++
	}

	whereClause := strings.Join(whereConditions, " AND ")

	query := fmt.Sprintf(`
		SELECT
			c.id as chunk_id,
			c.document_id,
			c.content,
			1 - (c.embedding <=> '%s'::vector) as similarity,
			c.metadata,
			d.title as document_title,
			d.tags
		FROM ai.chunks c
		JOIN ai.documents d ON d.id = c.document_id
		WHERE %s
		ORDER BY c.embedding <=> '%s'::vector
		LIMIT $3
	`, embeddingStr, whereClause, embeddingStr)

	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to search chunks with filter: %w", err)
	}
	defer rows.Close()

	var results []RetrievalResult
	for rows.Next() {
		var r RetrievalResult
		var docTitle *string
		var tags []string
		if err := rows.Scan(&r.ChunkID, &r.DocumentID, &r.Content, &r.Similarity, &r.Metadata, &docTitle, &tags); err != nil {
			log.Warn().Err(err).Msg("Failed to scan filtered search result")
			continue
		}
		r.KnowledgeBaseID = knowledgeBaseID
		if docTitle != nil {
			r.DocumentTitle = *docTitle
		}
		r.Tags = tags
		results = append(results, r)
	}

	return results, nil
}

// SearchChatbotKnowledge searches all knowledge bases linked to a chatbot
func (s *KnowledgeBaseStorage) SearchChatbotKnowledge(ctx context.Context, chatbotID string, queryEmbedding []float32) ([]RetrievalResult, error) {
	// Get linked knowledge bases
	links, err := s.GetChatbotKnowledgeBases(ctx, chatbotID)
	if err != nil {
		return nil, err
	}

	if len(links) == 0 {
		return nil, nil
	}

	// Search each knowledge base and combine results
	var allResults []RetrievalResult
	for _, link := range links {
		if !link.Enabled {
			continue
		}

		results, err := s.SearchChunks(ctx, link.KnowledgeBaseID, queryEmbedding, link.MaxChunks, link.SimilarityThreshold)
		if err != nil {
			log.Warn().Err(err).Str("kb_id", link.KnowledgeBaseID).Msg("Failed to search knowledge base")
			continue
		}

		// Get KB name for context
		kb, err := s.GetKnowledgeBase(ctx, link.KnowledgeBaseID)
		if err == nil && kb != nil {
			for i := range results {
				results[i].KnowledgeBaseName = kb.Name
			}
		}

		allResults = append(allResults, results...)
	}

	return allResults, nil
}

// ============================================================================
// Retrieval Logging
// ============================================================================

// LogRetrieval logs a RAG retrieval operation
func (s *KnowledgeBaseStorage) LogRetrieval(ctx context.Context, log *RetrievalLog) error {
	if log.ID == "" {
		log.ID = uuid.New().String()
	}
	log.CreatedAt = time.Now()

	query := `
		INSERT INTO ai.retrieval_log (
			id, chatbot_id, conversation_id, knowledge_base_id, user_id,
			query_text, query_embedding_model, chunks_retrieved,
			chunk_ids, similarity_scores, retrieval_duration_ms
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	_, err := s.db.Exec(ctx, query,
		log.ID, log.ChatbotID, log.ConversationID, log.KnowledgeBaseID, log.UserID,
		log.QueryText, log.QueryEmbeddingModel, log.ChunksRetrieved,
		log.ChunkIDs, log.SimilarityScores, log.RetrievalDurationMs,
	)
	return err
}

// formatEmbeddingLiteral formats a float32 slice as PostgreSQL vector literal
// Uses %v format to preserve full float32 precision (7 decimal digits)
func formatEmbeddingLiteral(v []float32) string {
	parts := make([]string, len(v))
	for i, f := range v {
		// Use %v for full float32 precision instead of %g which defaults to 6 significant digits
		parts[i] = fmt.Sprintf("%v", f)
	}
	return "[" + joinStrings(parts, ",") + "]"
}

func joinStrings(parts []string, sep string) string {
	if len(parts) == 0 {
		return ""
	}
	result := parts[0]
	for i := 1; i < len(parts); i++ {
		result += sep + parts[i]
	}
	return result
}

// sanitizeMetadataKey sanitizes a metadata key to prevent SQL injection
// Only allows alphanumeric characters and underscores
func sanitizeMetadataKey(key string) string {
	var result strings.Builder
	for _, r := range key {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// GetPendingDocuments retrieves documents pending processing
func (s *KnowledgeBaseStorage) GetPendingDocuments(ctx context.Context, limit int) ([]Document, error) {
	query := `
		SELECT id, knowledge_base_id, title, source_url, source_type,
			mime_type, content, content_hash, status, error_message,
			chunks_count, metadata, tags, created_by, created_at, updated_at, indexed_at
		FROM ai.documents
		WHERE status = 'pending'
		ORDER BY created_at
		LIMIT $1
	`

	rows, err := s.db.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending documents: %w", err)
	}
	defer rows.Close()

	var docs []Document
	for rows.Next() {
		var doc Document
		if err := rows.Scan(
			&doc.ID, &doc.KnowledgeBaseID, &doc.Title, &doc.SourceURL, &doc.SourceType,
			&doc.MimeType, &doc.Content, &doc.ContentHash, &doc.Status, &doc.ErrorMessage,
			&doc.ChunksCount, &doc.Metadata, &doc.Tags, &doc.CreatedBy, &doc.CreatedAt, &doc.UpdatedAt, &doc.IndexedAt,
		); err != nil {
			continue
		}
		docs = append(docs, doc)
	}

	return docs, nil
}

// UpdateChunkEmbedding updates the embedding for a single chunk
func (s *KnowledgeBaseStorage) UpdateChunkEmbedding(ctx context.Context, chunkID string, embedding []float32) error {
	embeddingJSON, err := json.Marshal(embedding)
	if err != nil {
		return err
	}

	query := `UPDATE ai.chunks SET embedding = $2::vector WHERE id = $1`
	_, err = s.db.Exec(ctx, query, chunkID, string(embeddingJSON))
	return err
}

// GetChunkEmbeddingPreview returns the first N values of a chunk's embedding for debugging
func (s *KnowledgeBaseStorage) GetChunkEmbeddingPreview(ctx context.Context, chunkID string, n int) ([]float32, error) {
	// Get the embedding as text and parse the first N values
	query := `SELECT left(embedding::text, 500) FROM ai.chunks WHERE id = $1`

	var embeddingText *string
	err := s.db.QueryRow(ctx, query, chunkID).Scan(&embeddingText)
	if err != nil {
		return nil, fmt.Errorf("failed to get embedding: %w", err)
	}

	if embeddingText == nil || *embeddingText == "" {
		return nil, fmt.Errorf("embedding is NULL for chunk %s", chunkID)
	}

	// Parse the vector literal format: [0.1,0.2,0.3,...]
	text := strings.TrimPrefix(*embeddingText, "[")
	parts := strings.Split(text, ",")

	result := make([]float32, 0, n)
	for i := 0; i < n && i < len(parts); i++ {
		var val float64
		_, err := fmt.Sscanf(parts[i], "%f", &val)
		if err != nil {
			break
		}
		result = append(result, float32(val))
	}

	return result, nil
}

// ChunkEmbeddingStats contains statistics about chunk embeddings in a knowledge base
type ChunkEmbeddingStats struct {
	TotalChunks            int `json:"total_chunks"`
	ChunksWithEmbedding    int `json:"chunks_with_embedding"`
	ChunksWithoutEmbedding int `json:"chunks_without_embedding"`
}

// GetChunkEmbeddingStats returns statistics about chunk embeddings for debugging
func (s *KnowledgeBaseStorage) GetChunkEmbeddingStats(ctx context.Context, knowledgeBaseID string) (*ChunkEmbeddingStats, error) {
	query := `
		SELECT
			COUNT(*) as total,
			COUNT(embedding) as with_embedding,
			COUNT(*) - COUNT(embedding) as without_embedding
		FROM ai.chunks
		WHERE knowledge_base_id = $1
	`

	var stats ChunkEmbeddingStats
	err := s.db.QueryRow(ctx, query, knowledgeBaseID).Scan(
		&stats.TotalChunks,
		&stats.ChunksWithEmbedding,
		&stats.ChunksWithoutEmbedding,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get chunk stats: %w", err)
	}

	return &stats, nil
}

// GetFirstChunkWithEmbedding returns the first chunk ID that has an embedding
func (s *KnowledgeBaseStorage) GetFirstChunkWithEmbedding(ctx context.Context, knowledgeBaseID string) (string, error) {
	query := `
		SELECT id FROM ai.chunks
		WHERE knowledge_base_id = $1 AND embedding IS NOT NULL
		LIMIT 1
	`

	var chunkID string
	err := s.db.QueryRow(ctx, query, knowledgeBaseID).Scan(&chunkID)
	if err != nil {
		return "", fmt.Errorf("no chunks with embeddings found: %w", err)
	}

	return chunkID, nil
}

// ============================================================================
// Convenience Methods for HTTP Handlers
// ============================================================================

// CreateKnowledgeBaseFromRequest creates a knowledge base from a request
func (s *KnowledgeBaseStorage) CreateKnowledgeBaseFromRequest(ctx context.Context, req CreateKnowledgeBaseRequest) (*KnowledgeBase, error) {
	defaults := DefaultKnowledgeBaseConfig()

	kb := &KnowledgeBase{
		Name:      req.Name,
		Namespace: req.Namespace,
		Enabled:   true,
		Source:    "api",
	}

	// Apply defaults where not specified
	if kb.Namespace == "" {
		kb.Namespace = defaults.Namespace
	}
	if req.Description != "" {
		kb.Description = req.Description
	}
	if req.EmbeddingModel != "" {
		kb.EmbeddingModel = req.EmbeddingModel
	} else {
		kb.EmbeddingModel = defaults.EmbeddingModel
	}
	if req.EmbeddingDimensions > 0 {
		kb.EmbeddingDimensions = req.EmbeddingDimensions
	} else {
		kb.EmbeddingDimensions = defaults.EmbeddingDimensions
	}
	if req.ChunkSize > 0 {
		kb.ChunkSize = req.ChunkSize
	} else {
		kb.ChunkSize = defaults.ChunkSize
	}
	if req.ChunkOverlap > 0 {
		kb.ChunkOverlap = req.ChunkOverlap
	} else {
		kb.ChunkOverlap = defaults.ChunkOverlap
	}
	if req.ChunkStrategy != "" {
		kb.ChunkStrategy = req.ChunkStrategy
	} else {
		kb.ChunkStrategy = defaults.ChunkStrategy
	}

	if err := s.CreateKnowledgeBase(ctx, kb); err != nil {
		return nil, err
	}

	return kb, nil
}

// UpdateKnowledgeBaseByID updates a knowledge base by ID from a request
func (s *KnowledgeBaseStorage) UpdateKnowledgeBaseByID(ctx context.Context, id string, req UpdateKnowledgeBaseRequest) (*KnowledgeBase, error) {
	// Get existing knowledge base
	kb, err := s.GetKnowledgeBase(ctx, id)
	if err != nil {
		return nil, err
	}
	if kb == nil {
		return nil, nil
	}

	// Apply updates
	if req.Name != nil {
		kb.Name = *req.Name
	}
	if req.Description != nil {
		kb.Description = *req.Description
	}
	if req.EmbeddingModel != nil {
		kb.EmbeddingModel = *req.EmbeddingModel
	}
	if req.EmbeddingDimensions != nil {
		kb.EmbeddingDimensions = *req.EmbeddingDimensions
	}
	if req.ChunkSize != nil {
		kb.ChunkSize = *req.ChunkSize
	}
	if req.ChunkOverlap != nil {
		kb.ChunkOverlap = *req.ChunkOverlap
	}
	if req.ChunkStrategy != nil {
		kb.ChunkStrategy = *req.ChunkStrategy
	}
	if req.Enabled != nil {
		kb.Enabled = *req.Enabled
	}

	if err := s.UpdateKnowledgeBase(ctx, kb); err != nil {
		return nil, err
	}

	return kb, nil
}

// ListAllKnowledgeBases lists all knowledge bases (no filtering)
func (s *KnowledgeBaseStorage) ListAllKnowledgeBases(ctx context.Context) ([]KnowledgeBase, error) {
	return s.ListKnowledgeBases(ctx, "", false)
}

// UpdateChatbotKnowledgeBaseOptions represents options for updating a link
type UpdateChatbotKnowledgeBaseOptions struct {
	Priority            *int
	MaxChunks           *int
	SimilarityThreshold *float64
	Enabled             *bool
}

// UpdateChatbotKnowledgeBaseLink updates a chatbot-knowledge base link
func (s *KnowledgeBaseStorage) UpdateChatbotKnowledgeBaseLink(ctx context.Context, chatbotID, kbID string, opts UpdateChatbotKnowledgeBaseOptions) (*ChatbotKnowledgeBase, error) {
	// First get the existing link
	links, err := s.GetChatbotKnowledgeBases(ctx, chatbotID)
	if err != nil {
		return nil, err
	}

	var existingLink *ChatbotKnowledgeBase
	for i := range links {
		if links[i].KnowledgeBaseID == kbID {
			existingLink = &links[i]
			break
		}
	}

	if existingLink == nil {
		return nil, nil
	}

	// Apply updates
	if opts.Priority != nil {
		existingLink.Priority = *opts.Priority
	}
	if opts.MaxChunks != nil {
		existingLink.MaxChunks = *opts.MaxChunks
	}
	if opts.SimilarityThreshold != nil {
		existingLink.SimilarityThreshold = *opts.SimilarityThreshold
	}
	if opts.Enabled != nil {
		existingLink.Enabled = *opts.Enabled
	}

	// Update using the existing link method (which handles upsert)
	if err := s.LinkChatbotKnowledgeBase(ctx, existingLink); err != nil {
		return nil, err
	}

	return existingLink, nil
}

// LinkChatbotKnowledgeBaseSimple is a convenience method for linking
func (s *KnowledgeBaseStorage) LinkChatbotKnowledgeBaseSimple(ctx context.Context, chatbotID, kbID string, priority, maxChunks int, similarityThreshold float64) (*ChatbotKnowledgeBase, error) {
	link := &ChatbotKnowledgeBase{
		ChatbotID:           chatbotID,
		KnowledgeBaseID:     kbID,
		Enabled:             true,
		Priority:            priority,
		MaxChunks:           maxChunks,
		SimilarityThreshold: similarityThreshold,
	}

	if err := s.LinkChatbotKnowledgeBase(ctx, link); err != nil {
		return nil, err
	}

	return link, nil
}
