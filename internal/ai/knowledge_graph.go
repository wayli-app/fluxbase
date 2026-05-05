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

// KnowledgeGraph handles entity and relationship storage and queries
type KnowledgeGraph struct {
	database.TenantAware
	storage *KnowledgeBaseStorage
}

// NewKnowledgeGraph creates a new knowledge graph service
func NewKnowledgeGraph(storage *KnowledgeBaseStorage) *KnowledgeGraph {
	return &KnowledgeGraph{
		TenantAware: database.TenantAware{DB: storage.DB},
		storage:     storage,
	}
}

// ============================================================================
// Entity Operations
// ============================================================================

// AddEntity adds an entity to the knowledge graph
// Returns the actual entity ID from the database (may differ from input ID on conflict)
func (kg *KnowledgeGraph) AddEntity(ctx context.Context, entity *Entity) error {
	if entity.ID == "" {
		entity.ID = uuid.New().String()
	}
	entity.CreatedAt = time.Now()
	entity.UpdatedAt = time.Now()

	query := `
		INSERT INTO ai.entities (
			id, knowledge_base_id, entity_type, name, canonical_name,
			aliases, metadata, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (knowledge_base_id, entity_type, canonical_name)
		DO UPDATE SET
			name = EXCLUDED.name,
			aliases = EXCLUDED.aliases,
			metadata = EXCLUDED.metadata,
			updated_at = NOW()
		RETURNING id, created_at, updated_at
	`

	return kg.WithTenant(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(
			ctx, query,
			entity.ID, entity.KnowledgeBaseID, entity.EntityType, entity.Name,
			entity.CanonicalName, entity.Aliases, entity.Metadata,
			entity.CreatedAt, entity.UpdatedAt,
		).Scan(&entity.ID, &entity.CreatedAt, &entity.UpdatedAt)
	})
}

// GetEntity retrieves an entity by ID
func (kg *KnowledgeGraph) GetEntity(ctx context.Context, entityID string) (*Entity, error) {
	query := `
		SELECT id, knowledge_base_id, entity_type, name, canonical_name,
			aliases, metadata, created_at, updated_at
		FROM ai.entities
		WHERE id = $1
	`

	var entity Entity
	err := kg.WithTenant(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, query, entityID).Scan(
			&entity.ID, &entity.KnowledgeBaseID, &entity.EntityType, &entity.Name,
			&entity.CanonicalName, &entity.Aliases, &entity.Metadata,
			&entity.CreatedAt, &entity.UpdatedAt,
		)
	})

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("entity not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get entity: %w", err)
	}

	return &entity, nil
}

// ListEntities lists all entities in a knowledge base
func (kg *KnowledgeGraph) ListEntities(ctx context.Context, kbID string, entityType *EntityType) ([]Entity, error) {
	query := `
		SELECT id, knowledge_base_id, entity_type, name, canonical_name,
			aliases, metadata, created_at, updated_at
		FROM ai.entities
		WHERE knowledge_base_id = $1
	`
	args := []interface{}{kbID}

	if entityType != nil {
		query += " AND entity_type = $2"
		args = append(args, *entityType)
	}

	query += " ORDER BY canonical_name"

	entities := make([]Entity, 0)
	err := kg.WithTenant(ctx, func(tx pgx.Tx) error {
		rows, queryErr := tx.Query(ctx, query, args...)
		if queryErr != nil {
			return fmt.Errorf("failed to list entities: %w", queryErr)
		}
		defer rows.Close()

		for rows.Next() {
			var entity Entity
			if scanErr := rows.Scan(
				&entity.ID, &entity.KnowledgeBaseID, &entity.EntityType, &entity.Name,
				&entity.CanonicalName, &entity.Aliases, &entity.Metadata,
				&entity.CreatedAt, &entity.UpdatedAt,
			); scanErr != nil {
				log.Warn().Err(scanErr).Msg("Failed to scan entity")
				continue
			}
			entities = append(entities, entity)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return entities, nil
}

// SearchEntities searches for entities by name (fuzzy matching)
func (kg *KnowledgeGraph) SearchEntities(ctx context.Context, kbID string, query string, entityTypes []EntityType, limit int) ([]Entity, error) {
	sqlQuery := `
		SELECT e.id, e.knowledge_base_id, e.entity_type, e.name, e.canonical_name,
			e.aliases, e.metadata, e.created_at, e.updated_at
		FROM ai.search_entities($1::UUID, $2::TEXT, $3::TEXT[], $4::INTEGER)
		JOIN ai.entities e ON e.id = search_entities.entity_id
		ORDER BY search_entities.rank DESC
	`

	// Convert entity types to strings
	var typeStrings *[]string
	if len(entityTypes) > 0 {
		types := make([]string, len(entityTypes))
		for i, et := range entityTypes {
			types[i] = string(et)
		}
		typeStrings = &types
	}

	entities := make([]Entity, 0)
	err := kg.WithTenant(ctx, func(tx pgx.Tx) error {
		rows, queryErr := tx.Query(ctx, sqlQuery, kbID, query, typeStrings, limit)
		if queryErr != nil {
			return fmt.Errorf("failed to search entities: %w", queryErr)
		}
		defer rows.Close()

		for rows.Next() {
			var entity Entity
			if scanErr := rows.Scan(
				&entity.ID, &entity.KnowledgeBaseID, &entity.EntityType, &entity.Name,
				&entity.CanonicalName, &entity.Aliases, &entity.Metadata,
				&entity.CreatedAt, &entity.UpdatedAt,
			); scanErr != nil {
				log.Warn().Err(scanErr).Msg("Failed to scan search result")
				continue
			}
			entities = append(entities, entity)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return entities, nil
}

// ============================================================================
// Relationship Operations
// ============================================================================

// AddRelationship adds a relationship between two entities
func (kg *KnowledgeGraph) AddRelationship(ctx context.Context, rel *EntityRelationship) error {
	if rel.ID == "" {
		rel.ID = uuid.New().String()
	}
	rel.CreatedAt = time.Now()

	query := `
		INSERT INTO ai.entity_relationships (
			id, knowledge_base_id, source_entity_id, target_entity_id,
			relationship_type, direction, confidence, metadata, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (knowledge_base_id, source_entity_id, target_entity_id, relationship_type)
		DO UPDATE SET
			direction = EXCLUDED.direction,
			confidence = EXCLUDED.confidence,
			metadata = EXCLUDED.metadata
		RETURNING created_at
	`

	return kg.WithTenant(ctx, func(tx pgx.Tx) error {
		_, err := tx.Exec(
			ctx, query,
			rel.ID, rel.KnowledgeBaseID, rel.SourceEntityID, rel.TargetEntityID,
			rel.RelationshipType, rel.Direction, rel.Confidence, rel.Metadata,
			rel.CreatedAt,
		)
		return err
	})
}

// GetRelationships gets all relationships for an entity
func (kg *KnowledgeGraph) GetRelationships(ctx context.Context, kbID string, entityID string) ([]EntityRelationship, error) {
	query := `
		SELECT r.id, r.knowledge_base_id, r.source_entity_id, r.target_entity_id,
			r.relationship_type, r.direction, r.confidence, r.metadata, r.created_at,
			s_e.id as source_id, s_e.entity_type as source_type, s_e.name as source_name,
			t_e.id as target_id, t_e.entity_type as target_type, t_e.name as target_name
		FROM ai.entity_relationships r
		JOIN ai.entities s_e ON s_e.id = r.source_entity_id
		JOIN ai.entities t_e ON t_e.id = r.target_entity_id
		WHERE r.knowledge_base_id = $1
			AND (r.source_entity_id = $2 OR r.target_entity_id = $2)
		ORDER BY r.relationship_type, s_e.name, t_e.name
	`

	relationships := make([]EntityRelationship, 0)
	err := kg.WithTenant(ctx, func(tx pgx.Tx) error {
		rows, queryErr := tx.Query(ctx, query, kbID, entityID)
		if queryErr != nil {
			return fmt.Errorf("failed to get relationships: %w", queryErr)
		}
		defer rows.Close()

		for rows.Next() {
			var rel EntityRelationship
			var sourceID, sourceType, sourceName string
			var targetID, targetType, targetName string

			if scanErr := rows.Scan(
				&rel.ID, &rel.KnowledgeBaseID, &rel.SourceEntityID, &rel.TargetEntityID,
				&rel.RelationshipType, &rel.Direction, &rel.Confidence, &rel.Metadata,
				&rel.CreatedAt,
				&sourceID, &sourceType, &sourceName,
				&targetID, &targetType, &targetName,
			); scanErr != nil {
				log.Warn().Err(scanErr).Msg("Failed to scan relationship")
				continue
			}

			rel.SourceEntity = &Entity{
				ID:         sourceID,
				EntityType: EntityType(sourceType),
				Name:       sourceName,
			}
			rel.TargetEntity = &Entity{
				ID:         targetID,
				EntityType: EntityType(targetType),
				Name:       targetName,
			}

			relationships = append(relationships, rel)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return relationships, nil
}

// FindRelatedEntities finds entities related to a given entity using graph traversal
func (kg *KnowledgeGraph) FindRelatedEntities(ctx context.Context, kbID string, entityID string, maxDepth int, relationshipTypes []RelationshipType) ([]RelatedEntity, error) {
	query := `
		SELECT * FROM ai.find_related_entities($1::UUID, $2::UUID, $3::INTEGER, $4::TEXT[])
	`

	// Convert relationship types to strings
	var typeStrings *[]string
	if len(relationshipTypes) > 0 {
		types := make([]string, len(relationshipTypes))
		for i, rt := range relationshipTypes {
			types[i] = string(rt)
		}
		typeStrings = &types
	}

	related := make([]RelatedEntity, 0)
	err := kg.WithTenant(ctx, func(tx pgx.Tx) error {
		rows, queryErr := tx.Query(ctx, query, kbID, entityID, maxDepth, typeStrings)
		if queryErr != nil {
			return fmt.Errorf("failed to find related entities: %w", queryErr)
		}
		defer rows.Close()

		for rows.Next() {
			var r RelatedEntity
			if scanErr := rows.Scan(
				&r.EntityID, &r.EntityType, &r.Name, &r.CanonicalName,
				&r.RelationshipType, &r.Depth, &r.Path,
			); scanErr != nil {
				log.Warn().Err(scanErr).Msg("Failed to scan related entity")
				continue
			}

			related = append(related, r)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return related, nil
}

// ============================================================================
// Document-Entity Operations
// ============================================================================

// AddDocumentEntities adds document-entity mentions
func (kg *KnowledgeGraph) AddDocumentEntities(ctx context.Context, docEntities []DocumentEntity) error {
	if len(docEntities) == 0 {
		return nil
	}

	query := `
		INSERT INTO ai.document_entities (
			id, document_id, entity_id, mention_count,
			first_mention_offset, salience, context, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (document_id, entity_id)
		DO UPDATE SET
			mention_count = EXCLUDED.mention_count,
			salience = EXCLUDED.salience,
			context = EXCLUDED.context
	`

	return kg.WithTenant(ctx, func(tx pgx.Tx) error {
		for _, de := range docEntities {
			if de.ID == "" {
				de.ID = uuid.New().String()
			}
			de.CreatedAt = time.Now()

			_, err := tx.Exec(
				ctx, query,
				de.ID, de.DocumentID, de.EntityID, de.MentionCount,
				de.FirstMentionOffset, de.Salience, de.Context, de.CreatedAt,
			)
			if err != nil {
				log.Warn().Err(err).
					Str("document_id", de.DocumentID).
					Str("entity_id", de.EntityID).
					Msg("Failed to add document-entity link")
			}
		}

		return nil
	})
}

// GetDocumentEntities gets all entities mentioned in a document
func (kg *KnowledgeGraph) GetDocumentEntities(ctx context.Context, documentID string) ([]DocumentEntity, error) {
	query := `
		SELECT de.id, de.document_id, de.entity_id, de.mention_count,
			de.first_mention_offset, de.salience, de.context, de.created_at,
			e.id, e.entity_type, e.name, e.canonical_name
		FROM ai.document_entities de
		JOIN ai.entities e ON e.id = de.entity_id
		WHERE de.document_id = $1
		ORDER BY de.salience DESC, e.name
	`

	docEntities := make([]DocumentEntity, 0)
	err := kg.WithTenant(ctx, func(tx pgx.Tx) error {
		rows, queryErr := tx.Query(ctx, query, documentID)
		if queryErr != nil {
			return fmt.Errorf("failed to get document entities: %w", queryErr)
		}
		defer rows.Close()

		for rows.Next() {
			var de DocumentEntity
			var entity Entity

			if scanErr := rows.Scan(
				&de.ID, &de.DocumentID, &de.EntityID, &de.MentionCount,
				&de.FirstMentionOffset, &de.Salience, &de.Context, &de.CreatedAt,
				&entity.ID, &entity.EntityType, &entity.Name, &entity.CanonicalName,
			); scanErr != nil {
				log.Warn().Err(scanErr).Msg("Failed to scan document entity")
				continue
			}

			de.Entity = &entity
			docEntities = append(docEntities, de)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return docEntities, nil
}

// GetEntitiesByDocument retrieves all entities mentioned in a document
func (kg *KnowledgeGraph) GetEntitiesByDocument(ctx context.Context, documentID string) ([]Entity, error) {
	query := `
		SELECT DISTINCT e.id, e.knowledge_base_id, e.entity_type, e.name,
			   e.canonical_name, e.aliases, e.metadata, e.created_at, e.updated_at
		FROM ai.entities e
		JOIN ai.document_entities de ON de.entity_id = e.id
		WHERE de.document_id = $1
		ORDER BY de.salience DESC, e.name
	`

	entities := make([]Entity, 0)
	err := kg.WithTenant(ctx, func(tx pgx.Tx) error {
		rows, queryErr := tx.Query(ctx, query, documentID)
		if queryErr != nil {
			return fmt.Errorf("failed to get document entities: %w", queryErr)
		}
		defer rows.Close()

		for rows.Next() {
			var entity Entity
			if scanErr := rows.Scan(
				&entity.ID, &entity.KnowledgeBaseID, &entity.EntityType, &entity.Name,
				&entity.CanonicalName, &entity.Aliases, &entity.Metadata,
				&entity.CreatedAt, &entity.UpdatedAt,
			); scanErr != nil {
				log.Warn().Err(scanErr).Msg("Failed to scan entity")
				continue
			}
			entities = append(entities, entity)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return entities, nil
}

// BatchAddEntities adds multiple entities efficiently using batch operations
func (kg *KnowledgeGraph) BatchAddEntities(ctx context.Context, entities []Entity) error {
	if len(entities) == 0 {
		return nil
	}

	query := `
		INSERT INTO ai.entities (
			id, knowledge_base_id, entity_type, name, canonical_name,
			aliases, metadata, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (knowledge_base_id, entity_type, canonical_name)
		DO UPDATE SET
			name = EXCLUDED.name,
			aliases = EXCLUDED.aliases,
			metadata = EXCLUDED.metadata,
			updated_at = NOW()
	`

	return kg.WithTenant(ctx, func(tx pgx.Tx) error {
		batch := &pgx.Batch{}

		for _, entity := range entities {
			if entity.ID == "" {
				entity.ID = uuid.New().String()
			}
			entity.CreatedAt = time.Now()
			entity.UpdatedAt = time.Now()

			batch.Queue(
				query,
				entity.ID, entity.KnowledgeBaseID, entity.EntityType, entity.Name,
				entity.CanonicalName, entity.Aliases, entity.Metadata,
				entity.CreatedAt, entity.UpdatedAt,
			)
		}

		br := tx.SendBatch(ctx, batch)
		defer func() { _ = br.Close() }()

		for range entities {
			if _, err := br.Exec(); err != nil {
				return fmt.Errorf("failed to insert entity: %w", err)
			}
		}

		return nil
	})
}

// DeleteOrphanedEntitiesByDocument deletes entities that are only referenced by a specific document
// This is useful for cleaning up table export entities when the document is deleted.
// It only deletes entities that have exactly one document reference (the deleted one).
func (kg *KnowledgeGraph) DeleteOrphanedEntitiesByDocument(ctx context.Context, documentID string) error {
	// Delete entities that are only referenced by this document
	// The CASCADE will automatically clean up relationships and document_entities
	query := `
		DELETE FROM ai.entities
		WHERE id IN (
			SELECT e.id
			FROM ai.entities e
			JOIN ai.document_entities de ON e.id = de.entity_id
			WHERE de.document_id = $1
			GROUP BY e.id
			HAVING COUNT(DISTINCT de.document_id) = 1
		)
	`

	var rowsAffected int64
	err := kg.WithTenant(ctx, func(tx pgx.Tx) error {
		result, execErr := tx.Exec(ctx, query, documentID)
		if execErr != nil {
			return fmt.Errorf("failed to delete orphaned entities: %w", execErr)
		}
		rowsAffected = result.RowsAffected()
		return nil
	})
	if err != nil {
		return err
	}

	if rowsAffected > 0 {
		log.Info().
			Str("document_id", documentID).
			Int64("deleted_entities", rowsAffected).
			Msg("Deleted orphaned entities for document")
	}

	return nil
}
