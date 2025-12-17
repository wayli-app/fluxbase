package ai

import (
	"context"
	"encoding/json"
	"fmt"
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

		batch.Queue(`
			INSERT INTO ai.chunks (
				id, document_id, knowledge_base_id, content,
				chunk_index, start_offset, end_offset, token_count,
				embedding, metadata
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		`,
			chunk.ID, chunk.DocumentID, chunk.KnowledgeBaseID, chunk.Content,
			chunk.ChunkIndex, chunk.StartOffset, chunk.EndOffset, chunk.TokenCount,
			chunk.Embedding, metadataJSON,
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
func formatEmbeddingLiteral(v []float32) string {
	parts := make([]string, len(v))
	for i, f := range v {
		parts[i] = fmt.Sprintf("%g", f)
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
