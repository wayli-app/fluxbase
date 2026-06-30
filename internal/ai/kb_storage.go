package ai

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/database"
)

// KnowledgeBaseStorage handles database operations for knowledge bases
type KnowledgeBaseStorage struct {
	database.TenantAware
}

// NewKnowledgeBaseStorage creates a new knowledge base storage
func NewKnowledgeBaseStorage(db *database.Connection) *KnowledgeBaseStorage {
	return &KnowledgeBaseStorage{TenantAware: database.TenantAware{DB: db}}
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

	return s.WithTenant(ctx, func(tx pgx.Tx) error {
		query := `
			INSERT INTO ai.knowledge_bases (
				id, name, namespace, description,
				embedding_model, embedding_dimensions,
				chunk_size, chunk_overlap, chunk_strategy,
				enabled, source, created_by, visibility, owner_id,
				entity_extraction_enabled
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
			RETURNING created_at, updated_at
		`

		return tx.QueryRow(
			ctx, query,
			kb.ID, kb.Name, kb.Namespace, kb.Description,
			kb.EmbeddingModel, kb.EmbeddingDimensions,
			kb.ChunkSize, kb.ChunkOverlap, kb.ChunkStrategy,
			kb.Enabled, kb.Source, kb.CreatedBy, kb.Visibility, kb.OwnerID,
			kb.EntityExtractionEnabled,
		).Scan(&kb.CreatedAt, &kb.UpdatedAt)
	})
}

// GetKnowledgeBase retrieves a knowledge base by ID
func (s *KnowledgeBaseStorage) GetKnowledgeBase(ctx context.Context, id string) (*KnowledgeBase, error) {
	tenantID := database.TenantFromContext(ctx)
	var kb KnowledgeBase
	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		query := `
			SELECT id, name, namespace, description,
				embedding_model, embedding_dimensions,
				chunk_size, chunk_overlap, chunk_strategy,
				enabled, document_count, total_chunks,
				source, created_by, created_at, updated_at,
				visibility, owner_id, entity_extraction_enabled
			FROM ai.knowledge_bases
			WHERE id = $1
			  AND (tenant_id = $2 OR ($2 IS NULL AND tenant_id IS NULL))
		`

		return tx.QueryRow(ctx, query, id, database.TenantOrNil(tenantID)).Scan(
			&kb.ID, &kb.Name, &kb.Namespace, &kb.Description,
			&kb.EmbeddingModel, &kb.EmbeddingDimensions,
			&kb.ChunkSize, &kb.ChunkOverlap, &kb.ChunkStrategy,
			&kb.Enabled, &kb.DocumentCount, &kb.TotalChunks,
			&kb.Source, &kb.CreatedBy, &kb.CreatedAt, &kb.UpdatedAt,
			&kb.Visibility, &kb.OwnerID, &kb.EntityExtractionEnabled,
		)
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get knowledge base: %w", err)
	}
	return &kb, nil
}

// GetKnowledgeBaseByName retrieves a knowledge base by name and namespace
func (s *KnowledgeBaseStorage) GetKnowledgeBaseByName(ctx context.Context, name, namespace string) (*KnowledgeBase, error) {
	tenantID := database.TenantFromContext(ctx)
	var kb KnowledgeBase
	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		query := `
			SELECT id, name, namespace, description,
				embedding_model, embedding_dimensions,
				chunk_size, chunk_overlap, chunk_strategy,
				enabled, document_count, total_chunks,
				source, created_by, created_at, updated_at, visibility,
				entity_extraction_enabled
			FROM ai.knowledge_bases
			WHERE name = $1 AND namespace = $2
			  AND (tenant_id = $3 OR ($3 IS NULL AND tenant_id IS NULL))
		`

		return tx.QueryRow(ctx, query, name, namespace, database.TenantOrNil(tenantID)).Scan(
			&kb.ID, &kb.Name, &kb.Namespace, &kb.Description,
			&kb.EmbeddingModel, &kb.EmbeddingDimensions,
			&kb.ChunkSize, &kb.ChunkOverlap, &kb.ChunkStrategy,
			&kb.Enabled, &kb.DocumentCount, &kb.TotalChunks,
			&kb.Source, &kb.CreatedBy, &kb.CreatedAt, &kb.UpdatedAt, &kb.Visibility,
			&kb.EntityExtractionEnabled,
		)
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get knowledge base by name: %w", err)
	}
	return &kb, nil
}

// ListKnowledgeBases lists knowledge bases with optional filtering
func (s *KnowledgeBaseStorage) ListKnowledgeBases(ctx context.Context, namespace string, enabledOnly bool) ([]KnowledgeBase, error) {
	tenantID := database.TenantFromContext(ctx)
	var kbs []KnowledgeBase
	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		query := `
			SELECT id, name, namespace, description,
				embedding_model, embedding_dimensions,
				chunk_size, chunk_overlap, chunk_strategy,
				enabled, document_count, total_chunks,
				source, created_by, created_at, updated_at,
				entity_extraction_enabled
			FROM ai.knowledge_bases
			WHERE (tenant_id = $1 OR ($1 IS NULL AND tenant_id IS NULL))
			  AND ($2 = '' OR namespace = $2)
			  AND ($3 = false OR enabled = true)
			ORDER BY namespace, name
		`

		rows, err := tx.Query(ctx, query, database.TenantOrNil(tenantID), namespace, enabledOnly)
		if err != nil {
			return fmt.Errorf("failed to list knowledge bases: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var kb KnowledgeBase
			if err := rows.Scan(
				&kb.ID, &kb.Name, &kb.Namespace, &kb.Description,
				&kb.EmbeddingModel, &kb.EmbeddingDimensions,
				&kb.ChunkSize, &kb.ChunkOverlap, &kb.ChunkStrategy,
				&kb.Enabled, &kb.DocumentCount, &kb.TotalChunks,
				&kb.Source, &kb.CreatedBy, &kb.CreatedAt, &kb.UpdatedAt,
				&kb.EntityExtractionEnabled,
			); err != nil {
				log.Warn().Err(err).Msg("Failed to scan knowledge base row")
				continue
			}
			kbs = append(kbs, kb)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return kbs, nil
}

// UpdateKnowledgeBase updates a knowledge base
func (s *KnowledgeBaseStorage) UpdateKnowledgeBase(ctx context.Context, kb *KnowledgeBase) error {
	return s.WithTenant(ctx, func(tx pgx.Tx) error {
		query := `
			UPDATE ai.knowledge_bases SET
				name = $2,
				description = $3,
				embedding_model = $4,
				embedding_dimensions = $5,
				chunk_size = $6,
				chunk_overlap = $7,
				chunk_strategy = $8,
				enabled = $9,
				visibility = $10,
				created_by = $11,
				owner_id = $12,
				entity_extraction_enabled = $13,
				updated_at = NOW()
			WHERE id = $1
			RETURNING updated_at
		`

		return tx.QueryRow(
			ctx, query,
			kb.ID, kb.Name, kb.Description,
			kb.EmbeddingModel, kb.EmbeddingDimensions,
			kb.ChunkSize, kb.ChunkOverlap, kb.ChunkStrategy,
			kb.Enabled, kb.Visibility, kb.CreatedBy, kb.OwnerID,
			kb.EntityExtractionEnabled,
		).Scan(&kb.UpdatedAt)
	})
}

// DeleteKnowledgeBase deletes a knowledge base and all its documents/chunks
func (s *KnowledgeBaseStorage) DeleteKnowledgeBase(ctx context.Context, id string) error {
	tenantID := database.TenantFromContext(ctx)
	return s.WithTenant(ctx, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, "DELETE FROM ai.knowledge_bases WHERE id = $1 AND (tenant_id = $2 OR ($2 IS NULL AND tenant_id IS NULL))", id, database.TenantOrNil(tenantID))
		return err
	})
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
	if req.Visibility != nil {
		kb.Visibility = *req.Visibility
	} else {
		kb.Visibility = KBVisibilityPrivate
	}
	if req.EmbeddingModel != "" {
		kb.EmbeddingModel = req.EmbeddingModel
	} else {
		kb.EmbeddingModel = defaults.EmbeddingModel
	}
	if req.EmbeddingDimensions > 0 {
		if req.EmbeddingDimensions != SupportedEmbeddingDimension {
			return nil, fmt.Errorf("unsupported embedding_dimensions: %d (only %d is supported by the current database schema)", req.EmbeddingDimensions, SupportedEmbeddingDimension)
		}
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

	// Set owner from request if provided
	kb.OwnerID = req.OwnerID

	// Entity extraction defaults to true; explicit false opts out.
	if req.EntityExtractionEnabled != nil {
		kb.EntityExtractionEnabled = *req.EntityExtractionEnabled
	} else {
		kb.EntityExtractionEnabled = true
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
	if req.Visibility != nil {
		kb.Visibility = *req.Visibility
	}
	if req.EmbeddingModel != nil {
		kb.EmbeddingModel = *req.EmbeddingModel
	}
	if req.EmbeddingDimensions != nil {
		if *req.EmbeddingDimensions != SupportedEmbeddingDimension {
			return nil, fmt.Errorf("unsupported embedding_dimensions: %d (only %d is supported by the current database schema)", *req.EmbeddingDimensions, SupportedEmbeddingDimension)
		}
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
	if req.EntityExtractionEnabled != nil {
		kb.EntityExtractionEnabled = *req.EntityExtractionEnabled
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
	AccessLevel         *string
	FilterExpression    map[string]interface{}
	ContextWeight       *float64
	Priority            *int
	IntentKeywords      []string
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
	if opts.AccessLevel != nil {
		existingLink.AccessLevel = *opts.AccessLevel
	}
	if opts.FilterExpression != nil {
		existingLink.FilterExpression = opts.FilterExpression
	}
	if opts.ContextWeight != nil {
		existingLink.ContextWeight = *opts.ContextWeight
	}
	if opts.Priority != nil {
		existingLink.Priority = *opts.Priority
	}
	if opts.IntentKeywords != nil {
		existingLink.IntentKeywords = opts.IntentKeywords
	}
	if opts.MaxChunks != nil {
		existingLink.MaxChunks = opts.MaxChunks
	}
	if opts.SimilarityThreshold != nil {
		existingLink.SimilarityThreshold = opts.SimilarityThreshold
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
		AccessLevel:         "full",
		Enabled:             true,
		Priority:            priority,
		MaxChunks:           &maxChunks,
		SimilarityThreshold: &similarityThreshold,
	}

	if err := s.LinkChatbotKnowledgeBase(ctx, link); err != nil {
		return nil, err
	}

	return link, nil
}

// ============================================================================
// Knowledge Base Ownership and Permissions
// ============================================================================

// ListUserKnowledgeBases returns KBs accessible to user
func (s *KnowledgeBaseStorage) ListUserKnowledgeBases(ctx context.Context, userID string) ([]KnowledgeBaseSummary, error) {
	var kbs []KnowledgeBaseSummary
	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		query := `
			SELECT kb.id, kb.name, kb.namespace, kb.description, kb.enabled,
				   kb.document_count, kb.total_chunks, kb.visibility,
				   kb.updated_at,
				   CASE
					   WHEN kb.owner_id = $1 THEN 'owner'
					   WHEN kbp.permission IS NOT NULL THEN kbp.permission
					   WHEN kb.visibility = 'public' THEN 'viewer'
					   ELSE NULL
				   END as user_permission
			FROM ai.knowledge_bases kb
			LEFT JOIN ai.knowledge_base_permissions kbp
				   ON kbp.knowledge_base_id = kb.id AND kbp.user_id = $1
			WHERE kb.enabled = true
			  AND (kb.owner_id = $1 OR kbp.user_id = $1 OR kb.visibility = 'public')
			ORDER BY kb.name
		`

		rows, err := tx.Query(ctx, query, userID)
		if err != nil {
			return fmt.Errorf("failed to list user knowledge bases: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var kb KnowledgeBase
			var userPermission string
			if err := rows.Scan(
				&kb.ID, &kb.Name, &kb.Namespace, &kb.Description,
				&kb.Enabled, &kb.DocumentCount, &kb.TotalChunks, &kb.Visibility,
				&kb.UpdatedAt,
				&userPermission,
			); err != nil {
				log.Warn().Err(err).Msg("Failed to scan knowledge base row")
				continue
			}
			summary := kb.ToSummary()
			summary.UserPermission = userPermission
			if kb.Visibility != "" {
				summary.Visibility = string(kb.Visibility)
			}
			kbs = append(kbs, summary)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return kbs, nil
}

// CanUserAccessKB checks if user has access
func (s *KnowledgeBaseStorage) CanUserAccessKB(ctx context.Context, kbID, userID string) bool {
	var hasAccess bool
	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		query := `
			SELECT EXISTS (
				SELECT 1 FROM ai.knowledge_bases kb
				LEFT JOIN ai.knowledge_base_permissions kbp
					   ON kbp.knowledge_base_id = kb.id AND kbp.user_id = $2
				WHERE kb.id = $1
				  AND kb.enabled = true
				  AND (kbp.user_id = $2 OR kb.visibility = 'public')
			)
		`
		return tx.QueryRow(ctx, query, kbID, userID).Scan(&hasAccess)
	})
	return err == nil && hasAccess
}

// CheckKBPermission checks if a user has the required permission level on a KB.
// The permission hierarchy is: viewer < editor < owner
// - If required is "viewer": user needs any permission (viewer, editor, or owner)
// - If required is "editor": user needs editor or owner permission
// - If required is "owner": user must be the KB owner or have owner permission
func (s *KnowledgeBaseStorage) CheckKBPermission(ctx context.Context, kbID, userID, requiredPermission string) (bool, error) {
	// Build the permission check based on required level
	var permissionCheck string
	switch requiredPermission {
	case string(KBPermissionViewer):
		// Any permission level allows read access
		permissionCheck = "kbp.permission IN ('viewer', 'editor', 'owner')"
	case string(KBPermissionEditor):
		// Editor or owner required for write operations
		permissionCheck = "kbp.permission IN ('editor', 'owner')"
	case string(KBPermissionOwner):
		// Only owner permission (or being the KB owner) allows full control
		permissionCheck = "kbp.permission = 'owner'"
	default:
		// Unknown permission level, deny access
		return false, nil
	}

	var hasPermission bool
	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		query := `
			SELECT EXISTS (
				SELECT 1 FROM ai.knowledge_bases kb
				LEFT JOIN ai.knowledge_base_permissions kbp
					ON kbp.knowledge_base_id = kb.id AND kbp.user_id = $2
				WHERE kb.id = $1
				  AND kb.enabled = true
				  AND (kb.owner_id = $2 OR ` + permissionCheck + `)
			)
		`

		return tx.QueryRow(ctx, query, kbID, userID).Scan(&hasPermission)
	})
	if err != nil {
		return false, fmt.Errorf("failed to check KB permission: %w", err)
	}
	return hasPermission, nil
}

// GetUserKBPermission gets the user's effective permission level on a KB.
// Returns the permission level or empty string if no access.
// For public KBs, returns 'viewer' if no explicit permission exists.
func (s *KnowledgeBaseStorage) GetUserKBPermission(ctx context.Context, kbID, userID string) (string, error) {
	var permission string
	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		query := `
			SELECT
				CASE
					WHEN kb.owner_id = $2 THEN 'owner'
					WHEN kbp.permission IS NOT NULL THEN kbp.permission
					WHEN kb.visibility = 'public' THEN 'viewer'
					ELSE ''
				END as permission
			FROM ai.knowledge_bases kb
			LEFT JOIN ai.knowledge_base_permissions kbp
				ON kbp.knowledge_base_id = kb.id AND kbp.user_id = $2
			WHERE kb.id = $1 AND kb.enabled = true
		`

		return tx.QueryRow(ctx, query, kbID, userID).Scan(&permission)
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil
		}
		return "", fmt.Errorf("failed to get user KB permission: %w", err)
	}
	return permission, nil
}

// GrantKBPermission grants permission to user
func (s *KnowledgeBaseStorage) GrantKBPermission(ctx context.Context, kbID, userID, permission string, grantedBy *string) (*KBPermissionGrant, error) {
	var grant KBPermissionGrant
	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		// Upsert permission
		query := `
			INSERT INTO ai.knowledge_base_permissions (knowledge_base_id, user_id, permission, granted_by)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (knowledge_base_id, user_id)
			DO UPDATE SET permission = $3, granted_by = $4, granted_at = NOW()
			RETURNING id, knowledge_base_id, user_id, permission, granted_by, granted_at
		`

		return tx.QueryRow(ctx, query, kbID, userID, permission, grantedBy).Scan(
			&grant.ID, &grant.KnowledgeBaseID, &grant.UserID, &grant.Permission, &grant.GrantedBy, &grant.GrantedAt,
		)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to grant permission: %w", err)
	}

	return &grant, nil
}

// ListKBPermissions lists all permissions for a KB
func (s *KnowledgeBaseStorage) ListKBPermissions(ctx context.Context, kbID string) ([]KBPermissionGrant, error) {
	var grants []KBPermissionGrant
	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		query := `
			SELECT id, knowledge_base_id, user_id, permission, granted_by, granted_at
			FROM ai.knowledge_base_permissions
			WHERE knowledge_base_id = $1
			ORDER BY granted_at DESC
		`

		rows, err := tx.Query(ctx, query, kbID)
		if err != nil {
			return fmt.Errorf("failed to list permissions: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var grant KBPermissionGrant
			if err := rows.Scan(
				&grant.ID, &grant.KnowledgeBaseID, &grant.UserID, &grant.Permission, &grant.GrantedBy, &grant.GrantedAt,
			); err != nil {
				log.Warn().Err(err).Msg("Failed to scan permission row")
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

// RevokeKBPermission revokes permission from user
func (s *KnowledgeBaseStorage) RevokeKBPermission(ctx context.Context, kbID, userID string) error {
	return s.WithTenant(ctx, func(tx pgx.Tx) error {
		query := `DELETE FROM ai.knowledge_base_permissions WHERE knowledge_base_id = $1 AND user_id = $2`
		_, err := tx.Exec(ctx, query, kbID, userID)
		if err != nil {
			return fmt.Errorf("failed to revoke permission: %w", err)
		}
		return nil
	})
}

// GetUserQuota retrieves quota information for a user
func (s *KnowledgeBaseStorage) GetUserQuota(ctx context.Context, userID string) (*UserQuota, error) {
	var quota UserQuota
	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		query := `
			SELECT user_id, max_documents, max_chunks, max_storage_bytes,
			       used_documents, used_chunks, used_storage_bytes,
			       created_at, updated_at
			FROM ai.user_quotas
			WHERE user_id = $1
		`

		return tx.QueryRow(ctx, query, userID).Scan(
			&quota.UserID,
			&quota.MaxDocuments,
			&quota.MaxChunks,
			&quota.MaxStorageBytes,
			&quota.UsedDocuments,
			&quota.UsedChunks,
			&quota.UsedStorageBytes,
			&quota.CreatedAt,
			&quota.UpdatedAt,
		)
	})
	if err != nil {
		return nil, err
	}

	return &quota, nil
}

// SetUserQuota creates or updates quota for a user
func (s *KnowledgeBaseStorage) SetUserQuota(ctx context.Context, quota *UserQuota) error {
	return s.WithTenant(ctx, func(tx pgx.Tx) error {
		query := `
			INSERT INTO ai.user_quotas (user_id, max_documents, max_chunks, max_storage_bytes)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (user_id) DO UPDATE
			SET max_documents = COALESCE(EXCLUDED.max_documents, ai.user_quotas.max_documents),
			    max_chunks = COALESCE(EXCLUDED.max_chunks, ai.user_quotas.max_chunks),
			    max_storage_bytes = COALESCE(EXCLUDED.max_storage_bytes, ai.user_quotas.max_storage_bytes),
			    updated_at = NOW()
		`

		_, err := tx.Exec(
			ctx, query,
			quota.UserID,
			quota.MaxDocuments,
			quota.MaxChunks,
			quota.MaxStorageBytes,
		)
		if err != nil {
			return fmt.Errorf("failed to set user quota: %w", err)
		}

		return nil
	})
}

// UpdateUserQuotaUsage updates quota usage counters for a user
func (s *KnowledgeBaseStorage) UpdateUserQuotaUsage(ctx context.Context, userID string, docsDelta int, chunksDelta int, storageDelta int64) error {
	return s.WithTenant(ctx, func(tx pgx.Tx) error {
		query := `
			INSERT INTO ai.user_quotas (user_id, used_documents, used_chunks, used_storage_bytes)
			VALUES ($1, GREATEST(0, $2), GREATEST(0, $3), GREATEST(0, $4))
			ON CONFLICT (user_id) DO UPDATE
			SET used_documents = GREATEST(0, ai.user_quotas.used_documents + $2),
			    used_chunks = GREATEST(0, ai.user_quotas.used_chunks + $3),
			    used_storage_bytes = GREATEST(0, ai.user_quotas.used_storage_bytes + $4),
			    updated_at = NOW()
		`

		_, err := tx.Exec(ctx, query, userID, docsDelta, chunksDelta, storageDelta)
		if err != nil {
			return fmt.Errorf("failed to update quota usage: %w", err)
		}

		return nil
	})
}
