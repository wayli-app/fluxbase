package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/fluxbase-eu/fluxbase/internal/ai"
	"github.com/fluxbase-eu/fluxbase/internal/mcp"
	"github.com/rs/zerolog/log"
)

// SearchVectorsTool implements the search_vectors MCP tool
type SearchVectorsTool struct {
	ragService *ai.RAGService
}

// NewSearchVectorsTool creates a new search_vectors tool
func NewSearchVectorsTool(ragService *ai.RAGService) *SearchVectorsTool {
	return &SearchVectorsTool{
		ragService: ragService,
	}
}

func (t *SearchVectorsTool) Name() string {
	return "search_vectors"
}

func (t *SearchVectorsTool) Description() string {
	return "Search for semantically similar content using vector embeddings. Requires a chatbot with linked knowledge bases."
}

func (t *SearchVectorsTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "The search query to find similar content",
			},
			"chatbot_id": map[string]any{
				"type":        "string",
				"description": "The chatbot ID (optional when called via chatbot context)",
			},
			"knowledge_bases": map[string]any{
				"type":        "array",
				"description": "Optional list of specific knowledge base names to search",
				"items": map[string]any{
					"type": "string",
				},
			},
			"limit": map[string]any{
				"type":        "integer",
				"description": "Maximum number of results (default: 5, max: 20)",
				"default":     5,
				"maximum":     20,
			},
			"threshold": map[string]any{
				"type":        "number",
				"description": "Minimum similarity threshold 0-1 (default: 0.7)",
				"default":     0.7,
			},
			"tags": map[string]any{
				"type":        "array",
				"description": "Optional tags to filter results",
				"items": map[string]any{
					"type": "string",
				},
			},
		},
		"required": []string{"query"}, // chatbot_id is optional - will be read from context metadata if not provided
	}
}

func (t *SearchVectorsTool) RequiredScopes() []string {
	return []string{mcp.ScopeReadVectors}
}

func (t *SearchVectorsTool) Execute(ctx context.Context, args map[string]any, authCtx *mcp.AuthContext) (*mcp.ToolResult, error) {
	if t.ragService == nil {
		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.ErrorContent("Vector search is not configured")},
			IsError: true,
		}, nil
	}

	// Parse arguments
	query, ok := args["query"].(string)
	if !ok || query == "" {
		return nil, fmt.Errorf("query is required")
	}

	// Try to get chatbot_id from args first, then fall back to metadata
	chatbotID, _ := args["chatbot_id"].(string)
	if chatbotID == "" {
		// Fall back to metadata (set by ChatbotAuthContext)
		chatbotID = authCtx.GetMetadataString(mcp.MetadataKeyChatbotID)
	}
	if chatbotID == "" {
		return nil, fmt.Errorf("chatbot_id is required (provide in args or context)")
	}

	limit := 5
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
		if limit > 20 {
			limit = 20
		}
	}

	threshold := 0.7
	if th, ok := args["threshold"].(float64); ok {
		threshold = th
	}

	// Parse knowledge bases
	var knowledgeBases []string
	if kbs, ok := args["knowledge_bases"].([]any); ok {
		for _, kb := range kbs {
			if kbStr, ok := kb.(string); ok {
				knowledgeBases = append(knowledgeBases, kbStr)
			}
		}
	}

	// Parse tags
	var tags []string
	if t, ok := args["tags"].([]any); ok {
		for _, tag := range t {
			if tagStr, ok := tag.(string); ok {
				tags = append(tags, tagStr)
			}
		}
	}

	log.Debug().
		Str("query", query).
		Str("chatbot_id", chatbotID).
		Int("limit", limit).
		Float64("threshold", threshold).
		Msg("MCP: Searching vectors")

	// Build search options
	opts := ai.VectorSearchOptions{
		Query:          query,
		ChatbotID:      chatbotID,
		KnowledgeBases: knowledgeBases,
		Limit:          limit,
		Threshold:      threshold,
		Tags:           tags,
	}

	// Add user context for filtering
	if authCtx.UserID != nil {
		opts.UserID = authCtx.UserID
	}

	// Check if user has admin access (service_role bypasses user filtering)
	if authCtx.UserRole == "service_role" || authCtx.UserRole == "dashboard_admin" {
		opts.IsAdmin = true
	}

	// Execute search
	results, err := t.ragService.VectorSearch(ctx, opts)
	if err != nil {
		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.ErrorContent(fmt.Sprintf("Vector search failed: %v", err))},
			IsError: true,
		}, nil
	}

	// Convert results
	resultList := make([]map[string]any, 0, len(results))
	for _, r := range results {
		item := map[string]any{
			"chunk_id":       r.ChunkID,
			"document_id":    r.DocumentID,
			"content":        r.Content,
			"similarity":     r.Similarity,
			"knowledge_base": r.KnowledgeBaseName,
		}
		if r.DocumentTitle != "" {
			item["document_title"] = r.DocumentTitle
		}
		if len(r.Tags) > 0 {
			item["tags"] = r.Tags
		}
		resultList = append(resultList, item)
	}

	response := map[string]any{
		"query":   query,
		"results": resultList,
		"count":   len(resultList),
	}

	resultJSON, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.ErrorContent(fmt.Sprintf("Failed to serialize result: %v", err))},
			IsError: true,
		}, nil
	}

	return &mcp.ToolResult{
		Content: []mcp.Content{mcp.TextContent(string(resultJSON))},
	}, nil
}
