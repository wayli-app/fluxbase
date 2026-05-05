package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"
)

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

	return s.WithTenant(ctx, func(tx pgx.Tx) error {
		query := `
			INSERT INTO ai.documents (
				id, knowledge_base_id, title, source_url, source_type,
				mime_type, content, content_hash, status, metadata, tags, created_by, owner_id
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
			RETURNING created_at, updated_at
		`

		return tx.QueryRow(
			ctx, query,
			doc.ID, doc.KnowledgeBaseID, doc.Title, doc.SourceURL, doc.SourceType,
			doc.MimeType, doc.Content, doc.ContentHash, doc.Status, metadataJSON, doc.Tags, doc.CreatedBy, doc.OwnerID,
		).Scan(&doc.CreatedAt, &doc.UpdatedAt)
	})
}

// GetDocument retrieves a document by ID
func (s *KnowledgeBaseStorage) GetDocument(ctx context.Context, id string) (*Document, error) {
	var doc Document
	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		query := `
			SELECT id, knowledge_base_id, title, source_url, source_type,
				mime_type, content, content_hash, status, error_message,
				chunks_count, metadata, tags, created_by, created_at, updated_at, indexed_at
			FROM ai.documents
			WHERE id = $1
		`

		return tx.QueryRow(ctx, query, id).Scan(
			&doc.ID, &doc.KnowledgeBaseID, &doc.Title, &doc.SourceURL, &doc.SourceType,
			&doc.MimeType, &doc.Content, &doc.ContentHash, &doc.Status, &doc.ErrorMessage,
			&doc.ChunksCount, &doc.Metadata, &doc.Tags, &doc.CreatedBy, &doc.CreatedAt, &doc.UpdatedAt, &doc.IndexedAt,
		)
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get document: %w", err)
	}
	return &doc, nil
}

// ListDocuments lists documents in a knowledge base
func (s *KnowledgeBaseStorage) ListDocuments(ctx context.Context, knowledgeBaseID string) ([]Document, error) {
	var docs []Document
	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		query := `
			SELECT id, knowledge_base_id, title, source_url, source_type,
				mime_type, content, content_hash, status, error_message,
				chunks_count, metadata, tags, created_by, created_at, updated_at, indexed_at
			FROM ai.documents
			WHERE knowledge_base_id = $1
			ORDER BY created_at DESC
		`

		rows, err := tx.Query(ctx, query, knowledgeBaseID)
		if err != nil {
			return fmt.Errorf("failed to list documents: %w", err)
		}
		defer rows.Close()

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

		return nil
	})
	if err != nil {
		return nil, err
	}

	return docs, nil
}

// UpdateDocumentStatus updates a document's processing status
func (s *KnowledgeBaseStorage) UpdateDocumentStatus(ctx context.Context, id string, status DocumentStatus, errorMsg string) error {
	return s.WithTenant(ctx, func(tx pgx.Tx) error {
		query := `
			UPDATE ai.documents SET
				status = $2, error_message = $3, updated_at = NOW()
			WHERE id = $1
		`
		_, err := tx.Exec(ctx, query, id, status, errorMsg)
		return err
	})
}

// MarkDocumentIndexed marks a document as indexed
func (s *KnowledgeBaseStorage) MarkDocumentIndexed(ctx context.Context, id string) error {
	return s.WithTenant(ctx, func(tx pgx.Tx) error {
		query := `
			UPDATE ai.documents SET
				status = 'indexed', indexed_at = NOW(), updated_at = NOW()
			WHERE id = $1
		`
		_, err := tx.Exec(ctx, query, id)
		return err
	})
}

// DeleteDocument deletes a document and its chunks
func (s *KnowledgeBaseStorage) DeleteDocument(ctx context.Context, id string) error {
	return s.WithTenant(ctx, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, "DELETE FROM ai.documents WHERE id = $1", id)
		return err
	})
}

// DeleteDocumentsByFilter deletes documents matching the given metadata filter
// Returns the number of documents deleted
func (s *KnowledgeBaseStorage) DeleteDocumentsByFilter(
	ctx context.Context,
	knowledgeBaseID string,
	filter *MetadataFilter,
) (int, error) {
	// Build WHERE clause for filtering
	whereConditions := []string{
		"knowledge_base_id = $1",
	}
	args := []interface{}{knowledgeBaseID}
	argIndex := 2

	// User isolation filter
	if filter != nil && filter.UserID != nil {
		whereConditions = append(whereConditions, fmt.Sprintf(`(
			metadata->>'user_id' = $%d OR
			metadata->>'user_id' IS NULL OR
			NOT (metadata ? 'user_id')
		)`, argIndex))
		args = append(args, *filter.UserID)
		argIndex++
	}

	// Tag filter - documents must have ALL specified tags
	if filter != nil && len(filter.Tags) > 0 {
		whereConditions = append(whereConditions, fmt.Sprintf("tags @> $%d", argIndex))
		args = append(args, filter.Tags)
		argIndex++
	}

	// Advanced metadata filter with operators and logical combinations
	if filter != nil && filter.AdvancedFilter != nil {
		// We need to use 'd' as the table alias for consistency, but here we're querying documents directly
		// So we need to adjust the SQL builder or prefix the table name
		metadataSQL, metadataArgs, err := buildMetadataFilterSQLForTable(*filter.AdvancedFilter, &argIndex, "")
		if err != nil {
			return 0, fmt.Errorf("failed to build metadata filter: %w", err)
		}
		if metadataSQL != "" {
			whereConditions = append(whereConditions, metadataSQL)
			args = append(args, metadataArgs...)
		}
	}

	// Legacy simple metadata filter (exact match only)
	if filter != nil && filter.AdvancedFilter == nil && len(filter.Metadata) > 0 {
		for key, value := range filter.Metadata {
			escapedKey := escapeStringLiteral(key)
			whereConditions = append(whereConditions, fmt.Sprintf("metadata->>'%s' = $%d", escapedKey, argIndex))
			args = append(args, value)
			argIndex++
		}
	}

	whereClause := strings.Join(whereConditions, " AND ")

	var rowsAffected int64
	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		query := fmt.Sprintf("DELETE FROM ai.documents WHERE %s", whereClause)

		result, err := tx.Exec(ctx, query, args...)
		if err != nil {
			return fmt.Errorf("failed to delete documents by filter: %w", err)
		}

		rowsAffected = result.RowsAffected()
		return nil
	})
	if err != nil {
		return 0, err
	}

	return int(rowsAffected), nil
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

	var doc Document
	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
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

		return tx.QueryRow(ctx, query, id, title, metadataJSON, tags).Scan(
			&doc.ID, &doc.KnowledgeBaseID, &doc.Title, &doc.SourceURL, &doc.SourceType,
			&doc.MimeType, &doc.Content, &doc.ContentHash, &doc.Status, &doc.ErrorMessage,
			&doc.ChunksCount, &doc.Metadata, &doc.Tags, &doc.CreatedBy, &doc.CreatedAt, &doc.UpdatedAt, &doc.IndexedAt,
		)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update document: %w", err)
	}

	return &doc, nil
}

// UpdateDocument updates a document entity in the database
func (s *KnowledgeBaseStorage) UpdateDocument(ctx context.Context, doc *Document) error {
	var metadataJSON []byte
	var err error
	if doc.Metadata != nil {
		metadataJSON, err = json.Marshal(doc.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	return s.WithTenant(ctx, func(tx pgx.Tx) error {
		query := `
			UPDATE ai.documents SET
				title = COALESCE($2, title),
				metadata = COALESCE($3, metadata),
				tags = COALESCE($4, tags),
				updated_at = NOW()
			WHERE id = $1
		`

		result, err := tx.Exec(ctx, query, doc.ID, doc.Title, metadataJSON, doc.Tags)
		if err != nil {
			return fmt.Errorf("failed to update document: %w", err)
		}

		if result.RowsAffected() == 0 {
			return fmt.Errorf("document not found")
		}

		return nil
	})
}

// FindDocumentByMetadata finds a document by knowledge base ID and metadata fields
// This is used for idempotent operations like table exports where we want to find
// an existing document for a specific schema.table combination.
func (s *KnowledgeBaseStorage) FindDocumentByMetadata(ctx context.Context, knowledgeBaseID string, metadata map[string]string) (*Document, error) {
	if len(metadata) == 0 {
		return nil, fmt.Errorf("at least one metadata field is required")
	}

	// Build WHERE conditions for each metadata field
	argIndex := 1
	var whereConditions []string
	var args []interface{}

	for key, value := range metadata {
		// Escape key to prevent SQL injection
		escapedKey := strings.ReplaceAll(key, "'", "''")
		whereConditions = append(whereConditions, fmt.Sprintf("metadata->>'%s' = $%d", escapedKey, argIndex))
		args = append(args, value)
		argIndex++
	}

	// Add knowledge base ID filter
	whereConditions = append(whereConditions, fmt.Sprintf("knowledge_base_id = $%d", argIndex))
	args = append(args, knowledgeBaseID)

	var doc Document
	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		query := fmt.Sprintf(`
			SELECT id, knowledge_base_id, title, source_url, source_type,
				mime_type, content, content_hash, status, error_message,
				chunks_count, metadata, tags, created_by, created_at, updated_at, indexed_at
			FROM ai.documents
			WHERE %s
			LIMIT 1
		`, strings.Join(whereConditions, " AND "))

		return tx.QueryRow(ctx, query, args...).Scan(
			&doc.ID, &doc.KnowledgeBaseID, &doc.Title, &doc.SourceURL, &doc.SourceType,
			&doc.MimeType, &doc.Content, &doc.ContentHash, &doc.Status, &doc.ErrorMessage,
			&doc.ChunksCount, &doc.Metadata, &doc.Tags, &doc.CreatedBy, &doc.CreatedAt, &doc.UpdatedAt, &doc.IndexedAt,
		)
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil // Not found
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find document by metadata: %w", err)
	}

	return &doc, nil
}

// UpdateDocumentContent updates a document's content, title, and metadata
// This is used for idempotent operations like re-exporting tables.
func (s *KnowledgeBaseStorage) UpdateDocumentContent(ctx context.Context, id string, content string, title string, metadataJSON []byte) error {
	// Calculate new content hash
	contentHash := hashContent(content)

	return s.WithTenant(ctx, func(tx pgx.Tx) error {
		query := `
			UPDATE ai.documents SET
				content = $2,
				content_hash = $3,
				title = $4,
				metadata = COALESCE($5, metadata),
				status = 'pending',
				updated_at = NOW()
			WHERE id = $1
		`
		_, err := tx.Exec(ctx, query, id, content, contentHash, title, metadataJSON)
		if err != nil {
			return fmt.Errorf("failed to update document content: %w", err)
		}

		return nil
	})
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

		batch.Queue(
			query,
			chunk.ID, chunk.DocumentID, chunk.KnowledgeBaseID, chunk.Content,
			chunk.ChunkIndex, chunk.StartOffset, chunk.EndOffset, chunk.TokenCount,
			metadataJSON,
		)
	}

	return s.WithTenant(ctx, func(tx pgx.Tx) error {
		br := tx.SendBatch(ctx, batch)
		defer func() { _ = br.Close() }()

		for range chunks {
			if _, err := br.Exec(); err != nil {
				return fmt.Errorf("failed to insert chunk: %w", err)
			}
		}

		return nil
	})
}

// GetChunksByDocument retrieves all chunks for a document
func (s *KnowledgeBaseStorage) GetChunksByDocument(ctx context.Context, documentID string) ([]Chunk, error) {
	var chunks []Chunk
	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		query := `
			SELECT id, document_id, knowledge_base_id, content,
				chunk_index, start_offset, end_offset, token_count, metadata, created_at
			FROM ai.chunks
			WHERE document_id = $1
			ORDER BY chunk_index
		`

		rows, err := tx.Query(ctx, query, documentID)
		if err != nil {
			return fmt.Errorf("failed to get chunks: %w", err)
		}
		defer rows.Close()

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

		return nil
	})
	if err != nil {
		return nil, err
	}

	return chunks, nil
}

// DeleteChunksByDocument deletes all chunks for a document
func (s *KnowledgeBaseStorage) DeleteChunksByDocument(ctx context.Context, documentID string) error {
	return s.WithTenant(ctx, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, "DELETE FROM ai.chunks WHERE document_id = $1", documentID)
		return err
	})
}

// GetPendingDocuments retrieves documents pending processing
func (s *KnowledgeBaseStorage) GetPendingDocuments(ctx context.Context, limit int) ([]Document, error) {
	var docs []Document
	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		query := `
			SELECT id, knowledge_base_id, title, source_url, source_type,
				mime_type, content, content_hash, status, error_message,
				chunks_count, metadata, tags, created_by, created_at, updated_at, indexed_at
			FROM ai.documents
			WHERE status = 'pending'
			ORDER BY created_at
			LIMIT $1
		`

		rows, err := tx.Query(ctx, query, limit)
		if err != nil {
			return fmt.Errorf("failed to get pending documents: %w", err)
		}
		defer rows.Close()

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

		return nil
	})
	if err != nil {
		return nil, err
	}

	return docs, nil
}

// UpdateChunkEmbedding updates the embedding for a single chunk
func (s *KnowledgeBaseStorage) UpdateChunkEmbedding(ctx context.Context, chunkID string, embedding []float32) error {
	embeddingJSON, err := json.Marshal(embedding)
	if err != nil {
		return err
	}

	return s.WithTenant(ctx, func(tx pgx.Tx) error {
		query := `UPDATE ai.chunks SET embedding = $2::vector WHERE id = $1`
		_, err := tx.Exec(ctx, query, chunkID, string(embeddingJSON))
		return err
	})
}

// GetChunkEmbeddingPreview returns the first N values of a chunk's embedding for debugging
func (s *KnowledgeBaseStorage) GetChunkEmbeddingPreview(ctx context.Context, chunkID string, n int) ([]float32, error) {
	var embeddingText *string
	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		// Get the embedding as text and parse the first N values
		query := `SELECT left(embedding::text, 500) FROM ai.chunks WHERE id = $1`

		return tx.QueryRow(ctx, query, chunkID).Scan(&embeddingText)
	})
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

// ============================================================================
// DOCUMENT PERMISSIONS
// ============================================================================

// GrantDocumentPermission grants permission on a document to a user
func (s *KnowledgeBaseStorage) GrantDocumentPermission(ctx context.Context, documentID, userID, permission, grantedBy string) (*DocumentPermissionGrant, error) {
	var grant DocumentPermissionGrant
	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		// First check if the requester owns the document
		var ownerID string
		checkQuery := `SELECT owner_id FROM ai.documents WHERE id = $1`
		err := tx.QueryRow(ctx, checkQuery, documentID).Scan(&ownerID)
		if err != nil {
			return fmt.Errorf("document not found: %w", err)
		}

		// Verify ownership (service role and dashboard admins bypass this check in the handler)
		if ownerID != grantedBy {
			// Check if grantedBy is dashboard admin or service role
			// This is a simple check - in production you'd want proper auth context
			return fmt.Errorf("only document owner can grant permissions")
		}

		// Upsert permission
		query := `
			INSERT INTO ai.document_permissions (document_id, user_id, permission, granted_by)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (document_id, user_id)
			DO UPDATE SET permission = $3, granted_by = $4, granted_at = NOW()
			RETURNING id, document_id, user_id, permission, granted_by, granted_at
		`

		return tx.QueryRow(ctx, query, documentID, userID, permission, grantedBy).Scan(
			&grant.ID, &grant.DocumentID, &grant.UserID, &grant.Permission, &grant.GrantedBy, &grant.GrantedAt,
		)
	})
	if err != nil {
		return nil, err
	}

	return &grant, nil
}

// ListDocumentPermissions lists all permissions for a document
func (s *KnowledgeBaseStorage) ListDocumentPermissions(ctx context.Context, documentID string) ([]DocumentPermissionGrant, error) {
	var grants []DocumentPermissionGrant
	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		query := `
			SELECT id, document_id, user_id, permission, granted_by, granted_at
			FROM ai.document_permissions
			WHERE document_id = $1
			ORDER BY granted_at DESC
		`

		rows, err := tx.Query(ctx, query, documentID)
		if err != nil {
			return fmt.Errorf("failed to list document permissions: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var grant DocumentPermissionGrant
			if err := rows.Scan(
				&grant.ID, &grant.DocumentID, &grant.UserID, &grant.Permission, &grant.GrantedBy, &grant.GrantedAt,
			); err != nil {
				log.Warn().Err(err).Msg("Failed to scan document permission row")
				continue
			}
			grants = append(grants, grant)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return grants, nil
}

// RevokeDocumentPermission revokes permission from a user on a document
func (s *KnowledgeBaseStorage) RevokeDocumentPermission(ctx context.Context, documentID, userID string) error {
	return s.WithTenant(ctx, func(tx pgx.Tx) error {
		query := `DELETE FROM ai.document_permissions WHERE document_id = $1 AND user_id = $2`
		_, err := tx.Exec(ctx, query, documentID, userID)
		if err != nil {
			return fmt.Errorf("failed to revoke document permission: %w", err)
		}
		return nil
	})
}

// CanUserAccessDocument checks if a user can access a document
func (s *KnowledgeBaseStorage) CanUserAccessDocument(ctx context.Context, documentID, userID string) (bool, error) {
	var hasAccess bool
	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		// Check if user owns the document
		var ownerID *string
		checkQuery := `SELECT owner_id FROM ai.documents WHERE id = $1`
		err := tx.QueryRow(ctx, checkQuery, documentID).Scan(&ownerID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil // hasAccess stays false
			}
			return fmt.Errorf("failed to check document ownership: %w", err)
		}

		// User owns the document
		if ownerID != nil && *ownerID == userID {
			hasAccess = true
			return nil
		}

		// Check if user has been granted permission
		permQuery := `
			SELECT EXISTS(
				SELECT 1 FROM ai.document_permissions
				WHERE document_id = $1 AND user_id = $2
			)
		`
		return tx.QueryRow(ctx, permQuery, documentID, userID).Scan(&hasAccess)
	})
	if err != nil {
		return false, err
	}

	return hasAccess, nil
}
