package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"
)

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

	var results []RetrievalResult
	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
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

		rows, err := tx.Query(ctx, query, knowledgeBaseID, threshold, limit)
		if err != nil {
			log.Error().Err(err).Str("kb_id", knowledgeBaseID).Msg("SearchChunks query failed")
			return fmt.Errorf("failed to search chunks: %w", err)
		}
		defer rows.Close()

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

		return nil
	})
	if err != nil {
		return nil, err
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
	Query          string
	QueryEmbedding []float32
	Limit          int
	Threshold      float64
	Mode           SearchMode
	SemanticWeight float64         // Weight for semantic score (0-1), keyword weight = 1 - semantic
	KeywordBoost   float64         // Boost factor for exact keyword matches
	Filter         *MetadataFilter // Optional metadata filter for user isolation
}

// GraphBoostOptions contains options for graph-boosted search
type GraphBoostOptions struct {
	QueryEmbedding   []float32       // Query vector embedding
	QueryText        string          // Query text for entity extraction
	Limit            int             // Maximum number of results to return
	Threshold        float64         // Minimum similarity threshold (0-1)
	GraphBoostWeight float64         // How much to weight entity matches vs vector similarity (0.0-1.0)
	Filter           *MetadataFilter // Optional metadata filter for user isolation
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
	var results []RetrievalResult
	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
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

		rows, err := tx.Query(ctx, query, knowledgeBaseID, opts.Query, opts.Limit)
		if err != nil {
			log.Error().Err(err).Str("kb_id", knowledgeBaseID).Msg("Keyword search query failed")
			return fmt.Errorf("failed to search chunks: %w", err)
		}
		defer rows.Close()

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

		return nil
	})
	if err != nil {
		return nil, err
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

	var results []RetrievalResult
	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
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

		rows, err := tx.Query(ctx, query, args...)
		if err != nil {
			log.Error().Err(err).Str("kb_id", knowledgeBaseID).Msg("Hybrid search query failed")
			return fmt.Errorf("failed to search chunks: %w", err)
		}
		defer rows.Close()

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

		return nil
	})
	if err != nil {
		return nil, err
	}

	log.Debug().
		Int("results_count", len(results)).
		Str("kb_id", knowledgeBaseID).
		Msg("Hybrid search completed")

	return results, nil
}

// SearchChunksWithGraphBoost performs vector search with entity-based boosting
// This combines semantic similarity with knowledge graph entity salience
// Entities are extracted from the query and documents mentioning those entities receive a ranking boost
func (s *KnowledgeBaseStorage) SearchChunksWithGraphBoost(
	ctx context.Context,
	knowledgeBaseID string,
	knowledgeGraph *KnowledgeGraph,
	entityExtractor EntityExtractor,
	opts GraphBoostOptions,
) ([]RetrievalResult, error) {
	// Apply defaults and validate
	if opts.GraphBoostWeight < 0 {
		opts.GraphBoostWeight = 0
	} else if opts.GraphBoostWeight > 1 {
		opts.GraphBoostWeight = 1
	}

	// If no boosting requested, use regular search for efficiency
	if opts.GraphBoostWeight == 0 {
		return s.SearchChunks(ctx, knowledgeBaseID, opts.QueryEmbedding, opts.Limit, opts.Threshold)
	}

	log.Debug().
		Str("kb_id", knowledgeBaseID).
		Float64("boost_weight", opts.GraphBoostWeight).
		Str("query", opts.QueryText).
		Msg("SearchChunksWithGraphBoost starting")

	// Step 1: Extract entities from query text
	var queryEntities []Entity
	if entityExtractor != nil && opts.QueryText != "" {
		extracted, err := entityExtractor.ExtractEntities(opts.QueryText, knowledgeBaseID)
		if err != nil {
			log.Warn().Err(err).Msg("Failed to extract entities from query, using vector-only search")
			return s.SearchChunks(ctx, knowledgeBaseID, opts.QueryEmbedding, opts.Limit, opts.Threshold)
		}
		queryEntities = extracted.Entities
		log.Debug().Int("entity_count", len(queryEntities)).Msg("Extracted entities from query")
	}

	// Step 2: Over-fetch candidate chunks (3x) for re-ranking. The helper honors
	// the optional user-isolation filter.
	chunks, err := s.searchChunksForBoost(ctx, knowledgeBaseID, opts)
	if err != nil {
		return nil, err
	}

	if len(chunks) == 0 {
		return chunks, nil
	}

	// Step 3: Calculate entity salience per document
	documentEntitySalience := make(map[string]float64) // document_id -> salience sum

	if len(queryEntities) > 0 && knowledgeGraph != nil {
		// For each query entity, find documents that mention it
		for _, queryEntity := range queryEntities {
			// Search for exact entity matches in the knowledge base
			matchingEntities, err := knowledgeGraph.SearchEntities(ctx, knowledgeBaseID, queryEntity.CanonicalName, nil, 50)
			if err != nil {
				log.Warn().Err(err).Str("entity", queryEntity.CanonicalName).Msg("Failed to search for entity")
				continue
			}

			// For each matching entity, get documents mentioning it with salience
			for _, entity := range matchingEntities {
				docEntities, err := knowledgeGraph.GetDocumentsByEntity(ctx, entity.ID)
				if err != nil {
					log.Warn().Err(err).Str("entity_id", entity.ID).Msg("Failed to get documents for entity")
					continue
				}

				// Aggregate salience per document
				for _, de := range docEntities {
					documentEntitySalience[de.DocumentID] += de.Salience
				}
			}
		}
	}

	log.Debug().
		Int("documents_with_entities", len(documentEntitySalience)).
		Msg("Found documents with entity matches")

	// Step 4-5: Apply entity boost, re-rank, and return top N
	return applyGraphBoost(chunks, documentEntitySalience, opts), nil
}

// searchChunksForBoost over-fetches candidate chunks for graph-boost re-ranking.
// Uses SearchChunksWithFilter when a user-isolation filter is supplied (preserving
// per-user scoping); otherwise falls back to plain SearchChunks.
func (s *KnowledgeBaseStorage) searchChunksForBoost(ctx context.Context, knowledgeBaseID string, opts GraphBoostOptions) ([]RetrievalResult, error) {
	retrievalLimit := opts.Limit * 3
	if retrievalLimit < 10 {
		retrievalLimit = 10
	}
	if retrievalLimit > 100 {
		retrievalLimit = 100
	}
	if opts.Filter != nil {
		return s.SearchChunksWithFilter(ctx, knowledgeBaseID, opts.QueryEmbedding, retrievalLimit, opts.Threshold, opts.Filter)
	}
	return s.SearchChunks(ctx, knowledgeBaseID, opts.QueryEmbedding, retrievalLimit, opts.Threshold)
}

// applyGraphBoost re-ranks retrieval results based on entity salience scores.
// It combines vector similarity with entity-based boosting using a weighted average.
// documentEntitySalience maps document_id -> aggregated salience score.
func applyGraphBoost(chunks []RetrievalResult, documentEntitySalience map[string]float64, opts GraphBoostOptions) []RetrievalResult {
	type boostedResult struct {
		result      RetrievalResult
		entityBoost float64
		finalScore  float64
	}

	boosted := make([]boostedResult, 0, len(chunks))

	maxSalience := 0.0
	for _, salience := range documentEntitySalience {
		if salience > maxSalience {
			maxSalience = salience
		}
	}

	for _, chunk := range chunks {
		entityBoost := 0.0
		if salience, ok := documentEntitySalience[chunk.DocumentID]; ok && maxSalience > 0 {
			entityBoost = (salience / maxSalience)
		}

		vectorWeight := 1.0 - opts.GraphBoostWeight
		finalScore := (chunk.Similarity * vectorWeight) + (entityBoost * opts.GraphBoostWeight)

		boosted = append(boosted, boostedResult{
			result:      chunk,
			entityBoost: entityBoost,
			finalScore:  finalScore,
		})
	}

	sort.Slice(boosted, func(i, j int) bool {
		return boosted[i].finalScore > boosted[j].finalScore
	})

	resultCount := opts.Limit
	if resultCount > len(boosted) {
		resultCount = len(boosted)
	}
	if resultCount == 0 {
		return []RetrievalResult{}
	}

	results := make([]RetrievalResult, resultCount)
	for i := 0; i < resultCount; i++ {
		results[i] = boosted[i].result
		results[i].Similarity = boosted[i].finalScore
		metadataMap := make(map[string]any)
		if len(results[i].Metadata) > 0 {
			_ = json.Unmarshal(results[i].Metadata, &metadataMap)
		}
		metadataMap["entity_boost"] = boosted[i].entityBoost
		metadataMap["final_score"] = boosted[i].finalScore
		metadataJSON, _ := json.Marshal(metadataMap)
		results[i].Metadata = json.RawMessage(metadataJSON)
	}

	log.Debug().
		Int("results_count", len(results)).
		Float64("top_boost", boosted[0].entityBoost).
		Float64("top_final_score", boosted[0].finalScore).
		Msg("SearchChunksWithGraphBoost completed")

	return results
}

// buildMetadataFilterSQL builds SQL WHERE conditions and args from a MetadataFilterGroup
// Returns (WHERE clause fragment, args, error)
func buildMetadataFilterSQL(group MetadataFilterGroup, argIndex *int) (string, []interface{}, error) {
	var conditions []string
	var args []interface{}

	// Process all conditions in this group
	for _, cond := range group.Conditions {
		conditionSQL, conditionArgs, err := buildConditionSQL(cond, argIndex)
		if err != nil {
			return "", nil, fmt.Errorf("failed to build condition for key '%s': %w", cond.Key, err)
		}
		conditions = append(conditions, conditionSQL)
		args = append(args, conditionArgs...)
	}

	// Process nested groups recursively
	for _, nestedGroup := range group.Groups {
		nestedSQL, nestedArgs, err := buildMetadataFilterSQL(nestedGroup, argIndex)
		if err != nil {
			return "", nil, err
		}
		if nestedSQL != "" {
			conditions = append(conditions, fmt.Sprintf("(%s)", nestedSQL))
			args = append(args, nestedArgs...)
		}
	}

	if len(conditions) == 0 {
		return "", args, nil
	}

	logicalOp := string(group.LogicalOp)
	if logicalOp == "" {
		logicalOp = "AND" // Default to AND
	}

	whereClause := strings.Join(conditions, fmt.Sprintf(" %s ", logicalOp))
	return whereClause, args, nil
}

// buildConditionSQL builds SQL for a single MetadataCondition
func buildConditionSQL(cond MetadataCondition, argIndex *int) (string, []interface{}, error) {
	var args []interface{}
	var sqlCond string

	// Use d.metadata->>'key' syntax to extract metadata as text
	metadataRef := fmt.Sprintf("d.metadata->>'%s'", escapeStringLiteral(cond.Key))

	switch cond.Operator {
	case MetadataOpEquals:
		if cond.Value == nil {
			return "", nil, errors.New("value is required for equals operator")
		}
		sqlCond = fmt.Sprintf("%s = $%d", metadataRef, *argIndex)
		args = append(args, fmt.Sprintf("%v", cond.Value))
		*argIndex++

	case MetadataOpNotEquals:
		if cond.Value == nil {
			return "", nil, errors.New("value is required for not equals operator")
		}
		sqlCond = fmt.Sprintf("%s != $%d", metadataRef, *argIndex)
		args = append(args, fmt.Sprintf("%v", cond.Value))
		*argIndex++

	case MetadataOpILike:
		if cond.Value == nil {
			return "", nil, errors.New("value is required for ilike operator")
		}
		sqlCond = fmt.Sprintf("%s ILIKE $%d", metadataRef, *argIndex)
		args = append(args, fmt.Sprintf("%v", cond.Value))
		*argIndex++

	case MetadataOpLike:
		if cond.Value == nil {
			return "", nil, errors.New("value is required for like operator")
		}
		sqlCond = fmt.Sprintf("%s LIKE $%d", metadataRef, *argIndex)
		args = append(args, fmt.Sprintf("%v", cond.Value))
		*argIndex++

	case MetadataOpIn:
		if len(cond.Values) == 0 {
			return "", nil, errors.New("values are required for IN operator")
		}
		placeholders := make([]string, len(cond.Values))
		for i, v := range cond.Values {
			placeholders[i] = fmt.Sprintf("$%d", *argIndex)
			args = append(args, fmt.Sprintf("%v", v))
			*argIndex++
		}
		sqlCond = fmt.Sprintf("%s IN (%s)", metadataRef, strings.Join(placeholders, ", "))

	case MetadataOpNotIn:
		if len(cond.Values) == 0 {
			return "", nil, errors.New("values are required for NOT IN operator")
		}
		placeholders := make([]string, len(cond.Values))
		for i, v := range cond.Values {
			placeholders[i] = fmt.Sprintf("$%d", *argIndex)
			args = append(args, fmt.Sprintf("%v", v))
			*argIndex++
		}
		sqlCond = fmt.Sprintf("%s NOT IN (%s)", metadataRef, strings.Join(placeholders, ", "))

	case MetadataOpGreaterThan:
		if cond.Value == nil {
			return "", nil, errors.New("value is required for greater than operator")
		}
		sqlCond = fmt.Sprintf("%s > $%d", metadataRef, *argIndex)
		args = append(args, fmt.Sprintf("%v", cond.Value))
		*argIndex++

	case MetadataOpGreaterThanOr:
		if cond.Value == nil {
			return "", nil, errors.New("value is required for greater than or equal operator")
		}
		sqlCond = fmt.Sprintf("%s >= $%d", metadataRef, *argIndex)
		args = append(args, fmt.Sprintf("%v", cond.Value))
		*argIndex++

	case MetadataOpLessThan:
		if cond.Value == nil {
			return "", nil, errors.New("value is required for less than operator")
		}
		sqlCond = fmt.Sprintf("%s < $%d", metadataRef, *argIndex)
		args = append(args, fmt.Sprintf("%v", cond.Value))
		*argIndex++

	case MetadataOpLessThanOr:
		if cond.Value == nil {
			return "", nil, errors.New("value is required for less than or equal operator")
		}
		sqlCond = fmt.Sprintf("%s <= $%d", metadataRef, *argIndex)
		args = append(args, fmt.Sprintf("%v", cond.Value))
		*argIndex++

	case MetadataOpBetween:
		if cond.Min == nil || cond.Max == nil {
			return "", nil, errors.New("min and max are required for BETWEEN operator")
		}
		sqlCond = fmt.Sprintf("%s BETWEEN $%d AND $%d", metadataRef, *argIndex, *argIndex+1)
		args = append(args, fmt.Sprintf("%v", cond.Min), fmt.Sprintf("%v", cond.Max))
		*argIndex += 2

	case MetadataOpIsNull:
		sqlCond = fmt.Sprintf("%s IS NULL", metadataRef)

	case MetadataOpIsNotNull:
		sqlCond = fmt.Sprintf("%s IS NOT NULL", metadataRef)

	default:
		return "", nil, fmt.Errorf("unsupported operator: %s", cond.Operator)
	}

	return sqlCond, args, nil
}

// escapeStringLiteral escapes single quotes in a string literal for SQL
func escapeStringLiteral(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

// buildMetadataFilterSQLForTable builds SQL WHERE conditions and args from a MetadataFilterGroup
// tablePrefix is the table alias prefix (e.g., "d" for "d.metadata") or empty string for direct table access
func buildMetadataFilterSQLForTable(group MetadataFilterGroup, argIndex *int, tablePrefix string) (string, []interface{}, error) {
	// Reuse the existing function with proper table prefix
	// For backward compatibility, when tablePrefix is empty, we use "metadata" directly
	prefix := tablePrefix
	if prefix == "" {
		prefix = "" // No prefix needed
	} else if prefix != "" && !strings.HasSuffix(prefix, ".") {
		prefix += "."
	}

	// Build conditions using the modified prefix
	var conditions []string
	var args []interface{}

	for _, cond := range group.Conditions {
		conditionSQL, conditionArgs, err := buildConditionSQLForTable(cond, argIndex, prefix)
		if err != nil {
			return "", nil, fmt.Errorf("failed to build condition for key '%s': %w", cond.Key, err)
		}
		conditions = append(conditions, conditionSQL)
		args = append(args, conditionArgs...)
	}

	// Process nested groups recursively
	for _, nestedGroup := range group.Groups {
		nestedSQL, nestedArgs, err := buildMetadataFilterSQLForTable(nestedGroup, argIndex, tablePrefix)
		if err != nil {
			return "", nil, err
		}
		if nestedSQL != "" {
			conditions = append(conditions, fmt.Sprintf("(%s)", nestedSQL))
			args = append(args, nestedArgs...)
		}
	}

	if len(conditions) == 0 {
		return "", args, nil
	}

	logicalOp := string(group.LogicalOp)
	if logicalOp == "" {
		logicalOp = "AND"
	}

	whereClause := strings.Join(conditions, fmt.Sprintf(" %s ", logicalOp))
	return whereClause, args, nil
}

// buildConditionSQLForTable builds SQL for a single MetadataCondition with a table prefix
func buildConditionSQLForTable(cond MetadataCondition, argIndex *int, tablePrefix string) (string, []interface{}, error) {
	var args []interface{}
	var sqlCond string

	// Use prefix + metadata->>'key' syntax
	metadataRef := fmt.Sprintf("%smetadata->>'%s'", tablePrefix, escapeStringLiteral(cond.Key))

	switch cond.Operator {
	case MetadataOpEquals:
		if cond.Value == nil {
			return "", nil, errors.New("value is required for equals operator")
		}
		sqlCond = fmt.Sprintf("%s = $%d", metadataRef, *argIndex)
		args = append(args, fmt.Sprintf("%v", cond.Value))
		*argIndex++

	case MetadataOpNotEquals:
		if cond.Value == nil {
			return "", nil, errors.New("value is required for not equals operator")
		}
		sqlCond = fmt.Sprintf("%s != $%d", metadataRef, *argIndex)
		args = append(args, fmt.Sprintf("%v", cond.Value))
		*argIndex++

	case MetadataOpILike:
		if cond.Value == nil {
			return "", nil, errors.New("value is required for ilike operator")
		}
		sqlCond = fmt.Sprintf("%s ILIKE $%d", metadataRef, *argIndex)
		args = append(args, fmt.Sprintf("%v", cond.Value))
		*argIndex++

	case MetadataOpLike:
		if cond.Value == nil {
			return "", nil, errors.New("value is required for like operator")
		}
		sqlCond = fmt.Sprintf("%s LIKE $%d", metadataRef, *argIndex)
		args = append(args, fmt.Sprintf("%v", cond.Value))
		*argIndex++

	case MetadataOpIn:
		if len(cond.Values) == 0 {
			return "", nil, errors.New("values are required for IN operator")
		}
		placeholders := make([]string, len(cond.Values))
		for i, v := range cond.Values {
			placeholders[i] = fmt.Sprintf("$%d", *argIndex)
			args = append(args, fmt.Sprintf("%v", v))
			*argIndex++
		}
		sqlCond = fmt.Sprintf("%s IN (%s)", metadataRef, strings.Join(placeholders, ", "))

	case MetadataOpNotIn:
		if len(cond.Values) == 0 {
			return "", nil, errors.New("values are required for NOT IN operator")
		}
		placeholders := make([]string, len(cond.Values))
		for i, v := range cond.Values {
			placeholders[i] = fmt.Sprintf("$%d", *argIndex)
			args = append(args, fmt.Sprintf("%v", v))
			*argIndex++
		}
		sqlCond = fmt.Sprintf("%s NOT IN (%s)", metadataRef, strings.Join(placeholders, ", "))

	case MetadataOpGreaterThan:
		if cond.Value == nil {
			return "", nil, errors.New("value is required for greater than operator")
		}
		sqlCond = fmt.Sprintf("%s > $%d", metadataRef, *argIndex)
		args = append(args, fmt.Sprintf("%v", cond.Value))
		*argIndex++

	case MetadataOpGreaterThanOr:
		if cond.Value == nil {
			return "", nil, errors.New("value is required for greater than or equal operator")
		}
		sqlCond = fmt.Sprintf("%s >= $%d", metadataRef, *argIndex)
		args = append(args, fmt.Sprintf("%v", cond.Value))
		*argIndex++

	case MetadataOpLessThan:
		if cond.Value == nil {
			return "", nil, errors.New("value is required for less than operator")
		}
		sqlCond = fmt.Sprintf("%s < $%d", metadataRef, *argIndex)
		args = append(args, fmt.Sprintf("%v", cond.Value))
		*argIndex++

	case MetadataOpLessThanOr:
		if cond.Value == nil {
			return "", nil, errors.New("value is required for less than or equal operator")
		}
		sqlCond = fmt.Sprintf("%s <= $%d", metadataRef, *argIndex)
		args = append(args, fmt.Sprintf("%v", cond.Value))
		*argIndex++

	case MetadataOpBetween:
		if cond.Min == nil || cond.Max == nil {
			return "", nil, errors.New("min and max are required for BETWEEN operator")
		}
		sqlCond = fmt.Sprintf("%s BETWEEN $%d AND $%d", metadataRef, *argIndex, *argIndex+1)
		args = append(args, fmt.Sprintf("%v", cond.Min), fmt.Sprintf("%v", cond.Max))
		*argIndex += 2

	case MetadataOpIsNull:
		sqlCond = fmt.Sprintf("%s IS NULL", metadataRef)

	case MetadataOpIsNotNull:
		sqlCond = fmt.Sprintf("%s IS NOT NULL", metadataRef)

	default:
		return "", nil, fmt.Errorf("unsupported operator: %s", cond.Operator)
	}

	return sqlCond, args, nil
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

	// Advanced metadata filter with operators and logical combinations
	if filter != nil && filter.AdvancedFilter != nil {
		metadataSQL, metadataArgs, err := buildMetadataFilterSQL(*filter.AdvancedFilter, &argIndex)
		if err != nil {
			return nil, fmt.Errorf("failed to build metadata filter: %w", err)
		}
		if metadataSQL != "" {
			whereConditions = append(whereConditions, metadataSQL)
			args = append(args, metadataArgs...)
		}
	}

	// Legacy simple metadata filter (exact match only) - for backward compatibility
	if filter != nil && filter.AdvancedFilter == nil && len(filter.Metadata) > 0 {
		for key, value := range filter.Metadata {
			escapedKey := escapeStringLiteral(key)
			whereConditions = append(whereConditions, fmt.Sprintf("d.metadata->>'%s' = $%d", escapedKey, argIndex))
			args = append(args, value)
			argIndex++
		}
	}

	whereClause := strings.Join(whereConditions, " AND ")

	var results []RetrievalResult
	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
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

		rows, err := tx.Query(ctx, query, args...)
		if err != nil {
			return fmt.Errorf("failed to search chunks with filter: %w", err)
		}
		defer rows.Close()

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

		return nil
	})
	if err != nil {
		return nil, err
	}

	return results, nil
}

// SearchChatbotKnowledgeOptions contains options for chatbot knowledge search
type SearchChatbotKnowledgeOptions struct {
	UserID    *string
	MaxChunks int
	Threshold float64
}

// SearchChatbotKnowledge searches all knowledge bases linked to a chatbot
func (s *KnowledgeBaseStorage) SearchChatbotKnowledge(ctx context.Context, chatbotID string, queryEmbedding []float32) ([]RetrievalResult, error) {
	return s.SearchChatbotKnowledgeWithOptions(ctx, chatbotID, queryEmbedding, SearchChatbotKnowledgeOptions{})
}

// SearchChatbotKnowledgeWithOptions searches all knowledge bases linked to a chatbot with user context
func (s *KnowledgeBaseStorage) SearchChatbotKnowledgeWithOptions(ctx context.Context, chatbotID string, queryEmbedding []float32, opts SearchChatbotKnowledgeOptions) ([]RetrievalResult, error) {
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

		// Use link defaults or fall back to system defaults
		maxChunks := 5
		if link.MaxChunks != nil {
			maxChunks = *link.MaxChunks
		}
		if opts.MaxChunks > 0 {
			maxChunks = opts.MaxChunks
		}

		threshold := 0.7
		if link.SimilarityThreshold != nil {
			threshold = *link.SimilarityThreshold
		}
		if opts.Threshold > 0 {
			threshold = opts.Threshold
		}

		var results []RetrievalResult

		// Build filter for user isolation and access level
		var filter *MetadataFilter
		if opts.UserID != nil || link.AccessLevel == "filtered" {
			filter = &MetadataFilter{}

			// Apply user isolation if UserID provided
			if opts.UserID != nil {
				filter.UserID = opts.UserID
				filter.IncludeGlobal = true
			}

			// Apply FilterExpression for "filtered" access level
			if link.AccessLevel == "filtered" && link.FilterExpression != nil {
				advancedFilter := convertFilterExpression(link.FilterExpression, opts.UserID)
				if advancedFilter != nil {
					filter.AdvancedFilter = advancedFilter
				}
			}
		}

		if filter != nil {
			results, err = s.SearchChunksWithFilter(ctx, link.KnowledgeBaseID, queryEmbedding, maxChunks, threshold, filter)
		} else {
			results, err = s.SearchChunks(ctx, link.KnowledgeBaseID, queryEmbedding, maxChunks, threshold)
		}

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

// convertFilterExpression converts a FilterExpression map to a MetadataFilterGroup
// It also substitutes special variables like $user_id
func convertFilterExpression(expr map[string]interface{}, userID *string) *MetadataFilterGroup {
	if expr == nil {
		return nil
	}

	var conditions []MetadataCondition

	for key, value := range expr {
		// Handle special variable substitution
		strValue, ok := value.(string)
		if ok && strValue == "$user_id" {
			if userID != nil {
				conditions = append(conditions, MetadataCondition{
					Key:      key,
					Operator: MetadataOpEquals,
					Value:    *userID,
				})
			}
			continue
		}

		// Handle direct value matches
		conditions = append(conditions, MetadataCondition{
			Key:      key,
			Operator: MetadataOpEquals,
			Value:    value,
		})
	}

	if len(conditions) == 0 {
		return nil
	}

	return &MetadataFilterGroup{
		Conditions: conditions,
		LogicalOp:  LogicalOpAND,
	}
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

// ChunkEmbeddingStats contains statistics about chunk embeddings in a knowledge base
type ChunkEmbeddingStats struct {
	TotalChunks            int `json:"total_chunks"`
	ChunksWithEmbedding    int `json:"chunks_with_embedding"`
	ChunksWithoutEmbedding int `json:"chunks_without_embedding"`
}

// GetChunkEmbeddingStats returns statistics about chunk embeddings for debugging
func (s *KnowledgeBaseStorage) GetChunkEmbeddingStats(ctx context.Context, knowledgeBaseID string) (*ChunkEmbeddingStats, error) {
	var stats ChunkEmbeddingStats
	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		query := `
			SELECT
				COUNT(*) as total,
				COUNT(embedding) as with_embedding,
				COUNT(*) - COUNT(embedding) as without_embedding
			FROM ai.chunks
			WHERE knowledge_base_id = $1
		`

		return tx.QueryRow(ctx, query, knowledgeBaseID).Scan(
			&stats.TotalChunks,
			&stats.ChunksWithEmbedding,
			&stats.ChunksWithoutEmbedding,
		)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get chunk stats: %w", err)
	}

	return &stats, nil
}

// GetFirstChunkWithEmbedding returns the first chunk ID that has an embedding
func (s *KnowledgeBaseStorage) GetFirstChunkWithEmbedding(ctx context.Context, knowledgeBaseID string) (string, error) {
	var chunkID string
	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		query := `
			SELECT id FROM ai.chunks
			WHERE knowledge_base_id = $1 AND embedding IS NOT NULL
			LIMIT 1
		`

		return tx.QueryRow(ctx, query, knowledgeBaseID).Scan(&chunkID)
	})
	if err != nil {
		return "", fmt.Errorf("no chunks with embeddings found: %w", err)
	}

	return chunkID, nil
}
