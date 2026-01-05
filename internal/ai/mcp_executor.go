package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/fluxbase-eu/fluxbase/internal/mcp"
	"github.com/rs/zerolog/log"
)

// MCPToolExecutor executes MCP tools on behalf of chatbots
type MCPToolExecutor struct {
	toolRegistry *mcp.ToolRegistry
}

// NewMCPToolExecutor creates a new MCP tool executor
func NewMCPToolExecutor(toolRegistry *mcp.ToolRegistry) *MCPToolExecutor {
	return &MCPToolExecutor{
		toolRegistry: toolRegistry,
	}
}

// ExecuteToolResult represents the result of an MCP tool execution
type ExecuteToolResult struct {
	Content string `json:"content"`
	IsError bool   `json:"is_error"`
}

// ExecuteTool executes an MCP tool on behalf of a chatbot
// It validates that:
// 1. The tool is in the chatbot's allowed tools list
// 2. The chatbot has the required scopes for the tool
// 3. For data tools, the table is in the chatbot's allowed tables
func (e *MCPToolExecutor) ExecuteTool(
	ctx context.Context,
	toolName string,
	args map[string]any,
	chatCtx *ChatContext,
	chatbot *Chatbot,
) (*ExecuteToolResult, error) {
	// Validate tool is allowed for this chatbot
	if err := ValidateChatbotToolAccess(chatbot, toolName); err != nil {
		return &ExecuteToolResult{
			Content: err.Error(),
			IsError: true,
		}, nil
	}

	// Validate table access for data tools
	if isDataTool(toolName) {
		if tableName, ok := args["table"].(string); ok && tableName != "" {
			if !IsTableAllowed(tableName, chatbot) {
				return &ExecuteToolResult{
					Content: fmt.Sprintf("Table '%s' is not allowed for this chatbot", tableName),
					IsError: true,
				}, nil
			}
		}
	}

	// Get the tool from registry
	tool := e.toolRegistry.GetTool(toolName)
	if tool == nil {
		return &ExecuteToolResult{
			Content: fmt.Sprintf("Tool '%s' not found", toolName),
			IsError: true,
		}, nil
	}

	// Create MCP auth context from chatbot context
	authCtx := ChatbotAuthContext(chatCtx, chatbot)

	log.Debug().
		Str("tool", toolName).
		Str("chatbot", chatbot.Name).
		Interface("args", args).
		Msg("Executing MCP tool for chatbot")

	// Execute the tool
	result, err := tool.Execute(ctx, args, authCtx)
	if err != nil {
		log.Error().
			Err(err).
			Str("tool", toolName).
			Str("chatbot", chatbot.Name).
			Msg("MCP tool execution failed")
		return &ExecuteToolResult{
			Content: fmt.Sprintf("Tool execution failed: %v", err),
			IsError: true,
		}, nil
	}

	// Convert MCP result to string content
	content := extractResultContent(result)

	return &ExecuteToolResult{
		Content: content,
		IsError: result.IsError,
	}, nil
}

// ExecuteVectorSearch executes a vector search on a table with embeddings
// This is called when query_table has a vector_search parameter
func (e *MCPToolExecutor) ExecuteVectorSearch(
	ctx context.Context,
	tableName string,
	vectorColumn string,
	query string,
	filters map[string]any,
	limit int,
	chatCtx *ChatContext,
	chatbot *Chatbot,
) (*ExecuteToolResult, error) {
	// Validate table access
	if !IsTableAllowed(tableName, chatbot) {
		return &ExecuteToolResult{
			Content: fmt.Sprintf("Table '%s' is not allowed for this chatbot", tableName),
			IsError: true,
		}, nil
	}

	// Check if search_vectors tool is available
	if !IsToolAllowed("search_vectors", chatbot.MCPTools) {
		return &ExecuteToolResult{
			Content: "Vector search is not enabled for this chatbot",
			IsError: true,
		}, nil
	}

	// Get the search_vectors tool
	tool := e.toolRegistry.GetTool("search_vectors")
	if tool == nil {
		return &ExecuteToolResult{
			Content: "Vector search tool not available",
			IsError: true,
		}, nil
	}

	// Build args for vector search
	args := map[string]any{
		"query":      query,
		"chatbot_id": chatbot.ID,
		"limit":      limit,
	}

	// Add table-specific context
	if tableName != "" {
		args["table"] = tableName
	}
	if vectorColumn != "" {
		args["vector_column"] = vectorColumn
	}

	// Create MCP auth context
	authCtx := ChatbotAuthContext(chatCtx, chatbot)

	// Execute vector search
	result, err := tool.Execute(ctx, args, authCtx)
	if err != nil {
		return &ExecuteToolResult{
			Content: fmt.Sprintf("Vector search failed: %v", err),
			IsError: true,
		}, nil
	}

	return &ExecuteToolResult{
		Content: extractResultContent(result),
		IsError: result.IsError,
	}, nil
}

// isDataTool returns true if the tool operates on tables
func isDataTool(toolName string) bool {
	dataTools := map[string]bool{
		"query_table":    true,
		"insert_record":  true,
		"update_record":  true,
		"delete_record":  true,
		"search_vectors": true,
	}
	return dataTools[toolName]
}

// extractResultContent converts MCP ToolResult content to a string
func extractResultContent(result *mcp.ToolResult) string {
	if result == nil || len(result.Content) == 0 {
		return ""
	}

	var parts []string
	for _, content := range result.Content {
		if content.Type == mcp.ContentTypeText && content.Text != "" {
			parts = append(parts, content.Text)
		}
		// Note: Resource content (URI-based) is not directly extractable as text
	}

	return strings.Join(parts, "\n")
}

// GetAvailableTools returns the tool definitions for tools available to a chatbot
func (e *MCPToolExecutor) GetAvailableTools(chatbot *Chatbot) []ToolDefinition {
	if !chatbot.HasMCPTools() {
		return nil
	}

	var tools []ToolDefinition
	for _, toolName := range chatbot.MCPTools {
		if info, exists := MCPToolInfoMap[toolName]; exists {
			tool := e.toolRegistry.GetTool(toolName)
			if tool == nil {
				continue
			}

			// Get schema from the actual tool
			schema := tool.InputSchema()

			// Add table restrictions to schema description for data tools
			if isDataTool(toolName) && len(chatbot.AllowedTables) > 0 {
				if props, ok := schema["properties"].(map[string]any); ok {
					if tableProp, ok := props["table"].(map[string]any); ok {
						desc := tableProp["description"].(string)
						tableProp["description"] = fmt.Sprintf("%s (allowed: %s)",
							desc, strings.Join(chatbot.AllowedTables, ", "))
					}
				}
			}

			tools = append(tools, ToolDefinition{
				Name:        toolName,
				Description: info.Description,
				Parameters:  schema,
			})
		}
	}

	return tools
}

// ToolDefinition represents a tool definition for the AI provider
type ToolDefinition struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

// ToAnthropicFormat converts tool definitions to Anthropic's tool format
func ToAnthropicFormat(tools []ToolDefinition) []map[string]any {
	result := make([]map[string]any, 0, len(tools))
	for _, tool := range tools {
		result = append(result, map[string]any{
			"name":         tool.Name,
			"description":  tool.Description,
			"input_schema": tool.Parameters,
		})
	}
	return result
}

// ToOpenAIFormat converts tool definitions to OpenAI's function calling format
func ToOpenAIFormat(tools []ToolDefinition) []map[string]any {
	result := make([]map[string]any, 0, len(tools))
	for _, tool := range tools {
		result = append(result, map[string]any{
			"type": "function",
			"function": map[string]any{
				"name":        tool.Name,
				"description": tool.Description,
				"parameters":  tool.Parameters,
			},
		})
	}
	return result
}

// ParseToolCall parses a tool call from the AI response
func ParseToolCall(name string, argsJSON string) (string, map[string]any, error) {
	var args map[string]any
	if argsJSON != "" {
		if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
			return name, nil, fmt.Errorf("failed to parse tool arguments: %w", err)
		}
	}
	return name, args, nil
}
