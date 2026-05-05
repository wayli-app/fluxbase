package ai

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"
)

// ============================================================================
// Chatbot Knowledge Base Links
// ============================================================================

// LinkChatbotKnowledgeBase links a chatbot to a knowledge base
func (s *KnowledgeBaseStorage) LinkChatbotKnowledgeBase(ctx context.Context, link *ChatbotKnowledgeBase) error {
	if link.ID == "" {
		link.ID = uuid.New().String()
	}
	link.CreatedAt = time.Now()
	link.UpdatedAt = time.Now()

	// Set defaults
	if link.AccessLevel == "" {
		link.AccessLevel = "full"
	}
	if link.ContextWeight == 0 {
		link.ContextWeight = 1.0
	}
	if link.Priority == 0 {
		link.Priority = 100
	}

	return s.WithTenant(ctx, func(tx pgx.Tx) error {
		query := `
			INSERT INTO ai.chatbot_knowledge_bases (
				id, chatbot_id, knowledge_base_id,
				access_level, filter_expression, context_weight, priority,
				intent_keywords, max_chunks, similarity_threshold,
				enabled, metadata
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
			ON CONFLICT (chatbot_id, knowledge_base_id) DO UPDATE SET
				access_level = EXCLUDED.access_level,
				filter_expression = EXCLUDED.filter_expression,
				context_weight = EXCLUDED.context_weight,
				priority = EXCLUDED.priority,
				intent_keywords = EXCLUDED.intent_keywords,
				max_chunks = EXCLUDED.max_chunks,
				similarity_threshold = EXCLUDED.similarity_threshold,
				enabled = EXCLUDED.enabled,
				metadata = EXCLUDED.metadata,
				updated_at = NOW()
			RETURNING created_at, updated_at
		`

		return tx.QueryRow(
			ctx, query,
			link.ID, link.ChatbotID, link.KnowledgeBaseID,
			link.AccessLevel, link.FilterExpression, link.ContextWeight, link.Priority,
			link.IntentKeywords, link.MaxChunks, link.SimilarityThreshold,
			link.Enabled, link.Metadata,
		).Scan(&link.CreatedAt, &link.UpdatedAt)
	})
}

// GetChatbotKnowledgeBases retrieves all knowledge base links for a chatbot
func (s *KnowledgeBaseStorage) GetChatbotKnowledgeBases(ctx context.Context, chatbotID string) ([]ChatbotKnowledgeBase, error) {
	var links []ChatbotKnowledgeBase
	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		query := `
			SELECT ckb.id, ckb.chatbot_id, ckb.knowledge_base_id,
				ckb.access_level, ckb.filter_expression, ckb.context_weight,
				ckb.priority, ckb.intent_keywords, ckb.max_chunks,
				ckb.similarity_threshold, ckb.enabled, ckb.metadata,
				ckb.created_at, ckb.updated_at,
				kb.name as knowledge_base_name
			FROM ai.chatbot_knowledge_bases ckb
			JOIN ai.knowledge_bases kb ON kb.id = ckb.knowledge_base_id
			WHERE ckb.chatbot_id = $1
			ORDER BY ckb.priority DESC
		`

		rows, err := tx.Query(ctx, query, chatbotID)
		if err != nil {
			return fmt.Errorf("failed to get chatbot knowledge bases: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var link ChatbotKnowledgeBase
			if err := rows.Scan(
				&link.ID, &link.ChatbotID, &link.KnowledgeBaseID,
				&link.AccessLevel, &link.FilterExpression, &link.ContextWeight,
				&link.Priority, &link.IntentKeywords, &link.MaxChunks,
				&link.SimilarityThreshold, &link.Enabled, &link.Metadata,
				&link.CreatedAt, &link.UpdatedAt,
				&link.KnowledgeBaseName,
			); err != nil {
				log.Warn().Err(err).Msg("Failed to scan chatbot knowledge base link")
				continue
			}
			links = append(links, link)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return links, nil
}

// GetChatbotKnowledgeBaseLinks is an alias for GetChatbotKnowledgeBases for the query router
func (s *KnowledgeBaseStorage) GetChatbotKnowledgeBaseLinks(ctx context.Context, chatbotID string) ([]ChatbotKnowledgeBase, error) {
	return s.GetChatbotKnowledgeBases(ctx, chatbotID)
}

// GetKnowledgeBaseChatbots retrieves all chatbot links for a knowledge base (reverse lookup)
// This is used to show which chatbots are using a specific knowledge base
func (s *KnowledgeBaseStorage) GetKnowledgeBaseChatbots(ctx context.Context, knowledgeBaseID string) ([]ChatbotKnowledgeBase, error) {
	var links []ChatbotKnowledgeBase
	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		query := `
			SELECT ckb.id, ckb.chatbot_id, ckb.knowledge_base_id,
				ckb.access_level, ckb.filter_expression, ckb.context_weight,
				ckb.priority, ckb.intent_keywords, ckb.max_chunks,
				ckb.similarity_threshold, ckb.enabled, ckb.metadata,
				ckb.created_at, ckb.updated_at,
				c.name as chatbot_name
			FROM ai.chatbot_knowledge_bases ckb
			JOIN ai.chatbots c ON c.id = ckb.chatbot_id
			WHERE ckb.knowledge_base_id = $1
			ORDER BY ckb.priority ASC
		`

		rows, err := tx.Query(ctx, query, knowledgeBaseID)
		if err != nil {
			return fmt.Errorf("failed to get knowledge base chatbots: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var link ChatbotKnowledgeBase
			if err := rows.Scan(
				&link.ID, &link.ChatbotID, &link.KnowledgeBaseID,
				&link.AccessLevel, &link.FilterExpression, &link.ContextWeight,
				&link.Priority, &link.IntentKeywords, &link.MaxChunks,
				&link.SimilarityThreshold, &link.Enabled, &link.Metadata,
				&link.CreatedAt, &link.UpdatedAt,
				&link.ChatbotName,
			); err != nil {
				log.Warn().Err(err).Msg("Failed to scan knowledge base chatbot link")
				continue
			}
			links = append(links, link)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return links, nil
}

// UnlinkChatbotKnowledgeBase removes a link between chatbot and knowledge base
func (s *KnowledgeBaseStorage) UnlinkChatbotKnowledgeBase(ctx context.Context, chatbotID, knowledgeBaseID string) error {
	return s.WithTenant(ctx, func(tx pgx.Tx) error {
		_, err := tx.Exec(
			ctx,
			"DELETE FROM ai.chatbot_knowledge_bases WHERE chatbot_id = $1 AND knowledge_base_id = $2",
			chatbotID, knowledgeBaseID,
		)
		return err
	})
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

	return s.WithTenant(ctx, func(tx pgx.Tx) error {
		query := `
			INSERT INTO ai.retrieval_log (
				id, chatbot_id, conversation_id, knowledge_base_id, user_id,
				query_text, query_embedding_model, chunks_retrieved,
				chunk_ids, similarity_scores, retrieval_duration_ms
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		`

		_, err := tx.Exec(
			ctx, query,
			log.ID, log.ChatbotID, log.ConversationID, log.KnowledgeBaseID, log.UserID,
			log.QueryText, log.QueryEmbeddingModel, log.ChunksRetrieved,
			log.ChunkIDs, log.SimilarityScores, log.RetrievalDurationMs,
		)
		return err
	})
}

// SyncChatbotKnowledgeBaseLinks syncs knowledge base links for a chatbot based on KB names
// This is called when a chatbot is synced from the filesystem to ensure the links match the config
// It will create links for KBs in the config that don't have links, and remove links that aren't in the config
func (s *KnowledgeBaseStorage) SyncChatbotKnowledgeBaseLinks(ctx context.Context, chatbotID string, kbNames []string, maxChunks int, similarityThreshold float64) error {
	// Get knowledge bases by name
	knowledgeBases, err := s.ListKnowledgeBases(ctx, "", false)
	if err != nil {
		return fmt.Errorf("failed to list knowledge bases: %w", err)
	}

	// Build a map of KB name to ID
	kbNameToID := make(map[string]string)
	for _, kb := range knowledgeBases {
		kbNameToID[kb.Name] = kb.ID
	}

	// Get existing links for this chatbot
	existingLinks, err := s.GetChatbotKnowledgeBases(ctx, chatbotID)
	if err != nil {
		return fmt.Errorf("failed to get existing links: %w", err)
	}

	// Build a set of expected KB IDs
	expectedKbIDs := make(map[string]bool)
	for _, kbName := range kbNames {
		if kbID, exists := kbNameToID[kbName]; exists {
			expectedKbIDs[kbID] = true
		} else {
			log.Warn().Str("kb_name", kbName).Str("chatbot_id", chatbotID).Msg("Knowledge base not found for linking")
		}
	}

	// Create or update links for KBs in the config
	for kbID := range expectedKbIDs {
		link := &ChatbotKnowledgeBase{
			ChatbotID:           chatbotID,
			KnowledgeBaseID:     kbID,
			AccessLevel:         "full",
			Enabled:             true,
			Priority:            1,
			MaxChunks:           &maxChunks,
			SimilarityThreshold: &similarityThreshold,
		}

		if err := s.LinkChatbotKnowledgeBase(ctx, link); err != nil {
			log.Warn().Err(err).Str("chatbot_id", chatbotID).Str("kb_id", kbID).Msg("Failed to link knowledge base")
		}
	}

	// Remove links that aren't in the config
	for _, existingLink := range existingLinks {
		if !expectedKbIDs[existingLink.KnowledgeBaseID] {
			if err := s.UnlinkChatbotKnowledgeBase(ctx, chatbotID, existingLink.KnowledgeBaseID); err != nil {
				log.Warn().Err(err).Str("chatbot_id", chatbotID).Str("kb_id", existingLink.KnowledgeBaseID).Msg("Failed to unlink knowledge base")
			}
		}
	}

	return nil
}
