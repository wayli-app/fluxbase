package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/fluxbase-eu/fluxbase/internal/auth"
	"github.com/fluxbase-eu/fluxbase/internal/config"
	"github.com/fluxbase-eu/fluxbase/internal/database"
	"github.com/fluxbase-eu/fluxbase/internal/logging"
	"github.com/fluxbase-eu/fluxbase/internal/mcp"
	"github.com/fluxbase-eu/fluxbase/internal/observability"
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// ChatHandler handles WebSocket chat connections
type ChatHandler struct {
	storage        *Storage
	conversations  *ConversationManager
	schemaBuilder  *SchemaBuilder
	executor       *Executor
	auditLogger    *AuditLogger
	ragService     *RAGService
	loggingService *logging.Service
	metrics        *observability.Metrics
	config         *config.AIConfig
	providers      map[string]Provider
	providersMu    sync.RWMutex
	// MCP integration
	mcpExecutor *MCPToolExecutor
}

// NewChatHandler creates a new chat handler
func NewChatHandler(
	db *database.Connection,
	storage *Storage,
	conversations *ConversationManager,
	metrics *observability.Metrics,
	cfg *config.AIConfig,
	embeddingService *EmbeddingService,
	loggingService *logging.Service,
) *ChatHandler {
	// Initialize RAG service if embedding is available
	var ragService *RAGService
	if embeddingService != nil {
		kbStorage := NewKnowledgeBaseStorage(db)
		ragService = NewRAGService(kbStorage, embeddingService)
	}

	return &ChatHandler{
		storage:        storage,
		conversations:  conversations,
		schemaBuilder:  NewSchemaBuilder(db),
		executor:       NewExecutor(db, metrics, cfg.MaxRowsPerQuery, cfg.QueryTimeout),
		auditLogger:    NewAuditLogger(db),
		ragService:     ragService,
		loggingService: loggingService,
		metrics:        metrics,
		config:         cfg,
		providers:      make(map[string]Provider),
	}
}

// SetSettingsResolver sets the settings resolver for template variable resolution in system prompts
func (h *ChatHandler) SetSettingsResolver(resolver *SettingsResolver) {
	h.schemaBuilder.SetSettingsResolver(resolver)
}

// SetMCPToolRegistry sets the MCP tool registry for MCP-enabled chatbots
func (h *ChatHandler) SetMCPToolRegistry(registry *mcp.ToolRegistry) {
	if registry != nil {
		h.mcpExecutor = NewMCPToolExecutor(registry)
	}
}

// SetMCPResources sets the MCP resource registry for schema fetching
func (h *ChatHandler) SetMCPResources(resources MCPResourceReader) {
	h.schemaBuilder.SetMCPResources(resources)
}

// GetRAGService returns the RAG service (may be nil if not initialized)
func (h *ChatHandler) GetRAGService() *RAGService {
	return h.ragService
}

// ResolveChatbotTemplates resolves template variables in chatbot annotation values.
// This resolves {{key}}, {{user:key}}, and {{system:key}} placeholders in fields
// like HTTPAllowedDomains that are parsed from annotations.
func (h *ChatHandler) ResolveChatbotTemplates(ctx context.Context, chatbot *Chatbot, userID *string) error {
	resolver := h.schemaBuilder.GetSettingsResolver()
	if resolver == nil {
		return nil
	}

	// Convert userID string to uuid.UUID pointer for resolver
	var userUUID *uuid.UUID
	if userID != nil && *userID != "" {
		parsed, err := uuid.Parse(*userID)
		if err == nil {
			userUUID = &parsed
		}
	}

	// Resolve HTTP allowed domains
	if len(chatbot.HTTPAllowedDomains) > 0 {
		resolved := make([]string, 0, len(chatbot.HTTPAllowedDomains))
		for _, domain := range chatbot.HTTPAllowedDomains {
			if strings.Contains(domain, "{{") {
				resolvedDomain, err := resolver.ResolveTemplate(ctx, domain, userUUID)
				if err != nil {
					return fmt.Errorf("failed to resolve template in http-allowed-domains: %w", err)
				}
				// Only add non-empty resolved values
				if resolvedDomain != "" {
					resolved = append(resolved, resolvedDomain)
				}
			} else {
				resolved = append(resolved, domain)
			}
		}
		chatbot.HTTPAllowedDomains = resolved
	}

	return nil
}

// ClientMessage represents a message from the client
type ClientMessage struct {
	Type              string `json:"type"`
	Chatbot           string `json:"chatbot,omitempty"`
	Namespace         string `json:"namespace,omitempty"`
	ConversationID    string `json:"conversation_id,omitempty"`
	Content           string `json:"content,omitempty"`
	ImpersonateUserID string `json:"impersonate_user_id,omitempty"` // Admin-only: test as this user
}

// ServerMessage represents a message to the client
type ServerMessage struct {
	Type           string           `json:"type"`
	ConversationID string           `json:"conversation_id,omitempty"`
	MessageID      string           `json:"message_id,omitempty"`
	Chatbot        string           `json:"chatbot,omitempty"`
	Step           string           `json:"step,omitempty"`
	Message        string           `json:"message,omitempty"`
	Delta          string           `json:"delta,omitempty"`
	Query          string           `json:"query,omitempty"`
	Summary        string           `json:"summary,omitempty"`
	RowCount       int              `json:"row_count,omitempty"`
	Data           []map[string]any `json:"data,omitempty"`
	Usage          *UsageStats      `json:"usage,omitempty"`
	Error          string           `json:"error,omitempty"`
	Code           string           `json:"code,omitempty"`
}

// ChatContext holds the context for a chat session
type ChatContext struct {
	Conn          *websocket.Conn
	UserID        *string
	Role          string
	Claims        *auth.TokenClaims
	IPAddress     string
	UserAgent     string
	Conversations map[string]*ConversationState
	ActiveChatbot *Chatbot
	Cancel        context.CancelFunc
}

// HandleWebSocket handles a WebSocket chat connection upgrade
func (h *ChatHandler) HandleWebSocket(c *fiber.Ctx) error {
	// Check if WebSocket upgrade
	if !websocket.IsWebSocketUpgrade(c) {
		return fiber.ErrUpgradeRequired
	}

	// Upgrade to WebSocket
	return websocket.New(h.handleConnection)(c)
}

// handleConnection handles an individual WebSocket connection
func (h *ChatHandler) handleConnection(c *websocket.Conn) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Extract auth context from locals (set by auth middleware)
	userID := extractString(c.Locals("user_id"))
	role := extractStringDefault(c.Locals("rls_role"), "anon")
	claims, _ := c.Locals("jwt_claims").(*auth.TokenClaims)

	chatCtx := &ChatContext{
		Conn:          c,
		UserID:        userID,
		Role:          role,
		Claims:        claims,
		IPAddress:     c.RemoteAddr().String(),
		UserAgent:     c.Headers("User-Agent"),
		Conversations: make(map[string]*ConversationState),
		Cancel:        cancel,
	}

	log.Info().
		Interface("user_id", userID).
		Str("role", role).
		Msg("AI WebSocket connection established")

	// Update metrics
	if h.metrics != nil {
		h.metrics.UpdateAIWebSocketConnections(1) // Increment - should track actual count
	}

	defer func() {
		// Cleanup
		if h.metrics != nil {
			h.metrics.UpdateAIWebSocketConnections(-1) // Decrement
		}
		log.Info().Interface("user_id", userID).Msg("AI WebSocket connection closed")
	}()

	// Message loop
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		_, msgBytes, err := c.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				return
			}
			log.Error().Err(err).Msg("Error reading WebSocket message")
			return
		}

		var msg ClientMessage
		if err := json.Unmarshal(msgBytes, &msg); err != nil {
			h.sendError(chatCtx, "", "INVALID_MESSAGE", "Invalid message format")
			continue
		}

		// Handle message based on type
		switch msg.Type {
		case "start_chat":
			h.handleStartChat(ctx, chatCtx, &msg)
		case "message":
			h.handleMessage(ctx, chatCtx, &msg)
		case "cancel":
			h.handleCancel(chatCtx, &msg)
		default:
			h.sendError(chatCtx, msg.ConversationID, "UNKNOWN_TYPE", "Unknown message type")
		}
	}
}

// handleStartChat handles starting a new chat session
func (h *ChatHandler) handleStartChat(ctx context.Context, chatCtx *ChatContext, msg *ClientMessage) {
	namespace := msg.Namespace
	if namespace == "" {
		namespace = "default"
	}

	// Get chatbot
	chatbot, err := h.storage.GetChatbotByName(ctx, namespace, msg.Chatbot)
	if err != nil {
		log.Error().Err(err).Str("chatbot", msg.Chatbot).Msg("Failed to get chatbot")
		h.sendError(chatCtx, "", "CHATBOT_ERROR", "Failed to get chatbot")
		return
	}

	if chatbot == nil || !chatbot.Enabled {
		h.sendError(chatCtx, "", "CHATBOT_NOT_FOUND", "Chatbot not found or disabled")
		return
	}

	// Handle admin impersonation
	if msg.ImpersonateUserID != "" {
		// Only dashboard_admin can impersonate
		if chatCtx.Role != "dashboard_admin" {
			h.sendError(chatCtx, "", "FORBIDDEN", "Only admins can impersonate users")
			return
		}

		// Verify the impersonated user exists
		exists, verifyErr := h.storage.UserExists(ctx, msg.ImpersonateUserID)
		if verifyErr != nil {
			log.Error().Err(verifyErr).Str("user_id", msg.ImpersonateUserID).Msg("Failed to verify user")
			h.sendError(chatCtx, "", "USER_ERROR", "Failed to verify user")
			return
		}
		if !exists {
			h.sendError(chatCtx, "", "USER_NOT_FOUND", "User not found")
			return
		}

		// Log admin ID before overriding
		adminID := "unknown"
		if chatCtx.UserID != nil {
			adminID = *chatCtx.UserID
		}

		// Override context with impersonated user
		impersonatedID := msg.ImpersonateUserID
		chatCtx.UserID = &impersonatedID
		chatCtx.Role = "authenticated" // Impersonated users run as authenticated, not admin

		log.Info().
			Str("admin_id", adminID).
			Str("impersonated_user_id", msg.ImpersonateUserID).
			Msg("Admin impersonating user for chatbot test")
	}

	// Check access control
	if !chatbot.AllowUnauthenticated && chatCtx.UserID == nil {
		h.sendError(chatCtx, "", "AUTH_REQUIRED", "Authentication required")
		return
	}

	// Resume existing conversation or create new
	var state *ConversationState
	if msg.ConversationID != "" {
		state, err = h.conversations.GetConversation(ctx, msg.ConversationID)
		if err != nil {
			log.Error().Err(err).Msg("Failed to get conversation")
		}
	}

	if state == nil {
		state, err = h.conversations.CreateConversation(ctx, chatbot, chatCtx.UserID, nil)
		if err != nil {
			log.Error().Err(err).Msg("Failed to create conversation")
			h.sendError(chatCtx, "", "CONVERSATION_ERROR", "Failed to create conversation")
			return
		}
	}

	chatCtx.ActiveChatbot = chatbot
	chatCtx.Conversations[state.ID] = state

	// Send confirmation
	h.send(chatCtx, ServerMessage{
		Type:           "chat_started",
		ConversationID: state.ID,
		Chatbot:        chatbot.Name,
	})

	log.Debug().
		Str("conversation_id", state.ID).
		Str("chatbot", chatbot.Name).
		Msg("Chat session started")
}

// handleMessage handles a user message
func (h *ChatHandler) handleMessage(ctx context.Context, chatCtx *ChatContext, msg *ClientMessage) {
	start := time.Now()

	state := chatCtx.Conversations[msg.ConversationID]
	if state == nil {
		h.sendError(chatCtx, msg.ConversationID, "NO_SESSION", "No active chat session")
		return
	}

	chatbot := chatCtx.ActiveChatbot
	if chatbot == nil {
		h.sendError(chatCtx, msg.ConversationID, "NO_CHATBOT", "No active chatbot")
		return
	}

	// Resolve template variables in chatbot annotation values (e.g., http-allowed-domains)
	if err := h.ResolveChatbotTemplates(ctx, chatbot, chatCtx.UserID); err != nil {
		log.Warn().Err(err).Str("chatbot", chatbot.Name).Msg("Failed to resolve chatbot templates")
		// Continue with unresolved values - don't fail the request
	}

	// Check turn limit
	if state.TurnCount >= chatbot.MaxConversationTurns {
		h.sendError(chatCtx, msg.ConversationID, "TURN_LIMIT", "Conversation turn limit reached")
		return
	}

	// Send thinking progress
	h.sendProgress(chatCtx, msg.ConversationID, "thinking", "Thinking...")

	// Build system prompt with schema
	userID := ""
	if chatCtx.UserID != nil {
		userID = *chatCtx.UserID
	}

	systemPrompt, err := h.schemaBuilder.BuildSystemPrompt(ctx, chatbot, userID)
	if err != nil {
		log.Error().Err(err).Msg("Failed to build system prompt")
		h.sendError(chatCtx, msg.ConversationID, "PROMPT_ERROR", "Failed to build prompt")
		return
	}

	// Retrieve RAG context if available
	if h.ragService != nil {
		ragSection, err := h.ragService.BuildRAGSystemPromptSection(ctx, chatbot.ID, msg.Content)
		if err != nil {
			log.Warn().Err(err).Str("chatbot_id", chatbot.ID).Msg("Failed to retrieve RAG context")
			// Continue without RAG - don't fail the request
		} else if ragSection != "" {
			systemPrompt = systemPrompt + "\n\n" + ragSection
			log.Debug().
				Str("chatbot_id", chatbot.ID).
				Str("conversation_id", msg.ConversationID).
				Int("rag_section_len", len(ragSection)).
				Msg("RAG context added to system prompt")
		}
	}

	// Build messages for LLM
	messages := []Message{
		{Role: RoleSystem, Content: systemPrompt},
	}

	// Add conversation history
	messages = append(messages, state.Messages...)

	// Add user message
	userMsg := Message{Role: RoleUser, Content: msg.Content}
	messages = append(messages, userMsg)

	// Get provider
	provider, err := h.getProvider(ctx, chatbot)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get provider")
		h.sendError(chatCtx, msg.ConversationID, "PROVIDER_ERROR", "AI provider not available")
		return
	}

	// Save user message to conversation
	_ = h.conversations.AddMessage(ctx, msg.ConversationID, userMsg, 0, 0)

	// Tool calling loop - continue until AI generates content without tool calls
	var totalUsage UsageStats
	var accumulatedQueryResults []QueryResult // Accumulate query results for persistence
	maxIterations := 5                        // Prevent infinite loops

	// Track consecutive tool validation failures to detect stubborn LLM behavior
	var lastFailedTool string
	var consecutiveFailures int
	const maxConsecutiveFailures = 2

	for iteration := 0; iteration < maxIterations; iteration++ {
		// Determine forbidden tools based on user message and intent rules
		var forbiddenTools []string
		if len(chatbot.IntentRules) > 0 {
			intentValidator := NewIntentValidator(chatbot.IntentRules, chatbot.RequiredColumns, chatbot.DefaultTable)
			forbiddenTools = intentValidator.GetForbiddenTools(msg.Content)
			if len(forbiddenTools) > 0 {
				log.Debug().
					Strs("forbidden_tools", forbiddenTools).
					Str("user_message", msg.Content).
					Msg("Filtering out forbidden tools based on intent rules")
			}
		}

		// Helper to check if a tool is forbidden
		isToolForbidden := func(toolName string) bool {
			for _, ft := range forbiddenTools {
				if ft == toolName {
					return true
				}
			}
			return false
		}

		// Build tools list based on chatbot configuration
		var tools []Tool

		// Add MCP tools if configured (includes execute_sql as MCP tool now)
		if chatbot.HasMCPTools() && h.mcpExecutor != nil {
			mcpToolDefs := h.mcpExecutor.GetAvailableTools(chatbot)
			for _, def := range mcpToolDefs {
				// Skip forbidden tools
				if isToolForbidden(def.Name) {
					continue
				}
				tools = append(tools, Tool{
					Type:     "function",
					Function: ToolFunction(def),
				})
			}
		} else {
			// Fallback: add legacy execute_sql if no MCP tools configured
			if !isToolForbidden("execute_sql") {
				tools = append(tools, ExecuteSQLTool)
			}
		}

		log.Debug().
			Str("chatbot", chatbot.Name).
			Int("total_tools", len(tools)).
			Msg("Tools available for chatbot")

		// Create chat request
		chatReq := &ChatRequest{
			Messages:    messages,
			MaxTokens:   chatbot.MaxTokens,
			Temperature: chatbot.Temperature,
			Tools:       tools,
			Stream:      true,
		}

		// Track response for this iteration
		var responseContent strings.Builder
		var pendingToolCalls []ToolCall

		// Stream callback
		callback := func(event StreamEvent) error {
			switch event.Type {
			case "content":
				responseContent.WriteString(event.Delta)
				h.send(chatCtx, ServerMessage{
					Type:           "content",
					ConversationID: msg.ConversationID,
					Delta:          event.Delta,
				})

			case "tool_call":
				// Collect tool calls to execute after streaming completes
				if event.ToolCall != nil {
					toolName := event.ToolCall.FunctionName
					// Accept legacy tools, MCP tools, or any tool requested by the model
					pendingToolCalls = append(pendingToolCalls, ToolCall{
						ID:   event.ToolCall.ID,
						Type: "function",
						Function: FunctionCall{
							Name:      toolName,
							Arguments: event.ToolCall.ArgumentsDelta,
						},
					})
				}

			case "done":
				if event.Usage != nil {
					totalUsage.PromptTokens += event.Usage.PromptTokens
					totalUsage.CompletionTokens += event.Usage.CompletionTokens
					totalUsage.TotalTokens += event.Usage.TotalTokens
				}
			}
			return nil
		}

		// Stream the response
		h.sendProgress(chatCtx, msg.ConversationID, "generating", "Generating response...")

		if err := provider.ChatStream(ctx, chatReq, callback); err != nil {
			log.Error().Err(err).Msg("Chat stream error")
			h.sendError(chatCtx, msg.ConversationID, "STREAM_ERROR", "Error generating response")

			if h.metrics != nil {
				h.metrics.RecordAIChatRequest(chatbot.Name, "error", time.Since(start))
			}
			return
		}

		// If no tool calls, we're done
		if len(pendingToolCalls) == 0 {
			// Save assistant message with accumulated query results
			assistantMsg := Message{
				Role:         RoleAssistant,
				Content:      responseContent.String(),
				QueryResults: accumulatedQueryResults,
			}
			_ = h.conversations.AddMessage(ctx, msg.ConversationID, assistantMsg, totalUsage.PromptTokens, totalUsage.CompletionTokens)
			break
		}

		// Add assistant message with tool calls to conversation
		assistantMsg := Message{
			Role:      RoleAssistant,
			Content:   responseContent.String(),
			ToolCalls: pendingToolCalls,
		}
		messages = append(messages, assistantMsg)

		// Execute each tool call and add results
		for _, tc := range pendingToolCalls {
			toolName := tc.Function.Name

			// Validate tool call against intent rules (requiredTool/forbiddenTool)
			if len(chatbot.IntentRules) > 0 {
				intentValidator := NewIntentValidator(chatbot.IntentRules, chatbot.RequiredColumns, chatbot.DefaultTable)
				toolValidation := intentValidator.ValidateToolCall(msg.Content, toolName)

				log.Debug().
					Int("intent_rules_count", len(chatbot.IntentRules)).
					Str("tool", toolName).
					Str("user_message", msg.Content).
					Bool("valid", toolValidation.Valid).
					Strs("matched_keywords", toolValidation.MatchedKeywords).
					Msg("Tool validation check")

				if !toolValidation.Valid {
					// Track consecutive failures for the same tool
					if toolName == lastFailedTool {
						consecutiveFailures++
					} else {
						lastFailedTool = toolName
						consecutiveFailures = 1
					}

					// If the same tool fails too many times, break the loop
					if consecutiveFailures >= maxConsecutiveFailures {
						log.Warn().
							Str("tool", toolName).
							Int("failures", consecutiveFailures).
							Msg("Breaking loop due to repeated tool validation failures")

						h.send(chatCtx, ServerMessage{
							Type:  "error",
							Error: "Unable to process this request - the AI kept trying to use a tool that isn't allowed for this type of query. Please rephrase your question.",
						})
						return
					}

					// Build list of alternative tools (exclude the forbidden one)
					var alternativeTools []string
					for _, t := range chatbot.MCPTools {
						if t != toolName {
							alternativeTools = append(alternativeTools, t)
						}
					}

					errMsg := fmt.Sprintf("TOOL NOT ALLOWED: %s. %s Available tools: %s. Please use one of these tools instead.",
						strings.Join(toolValidation.Errors, "; "),
						strings.Join(toolValidation.Suggestions, " "),
						strings.Join(alternativeTools, ", "))

					log.Debug().
						Strs("errors", toolValidation.Errors).
						Strs("alternative_tools", alternativeTools).
						Str("error_message", errMsg).
						Msg("Tool validation failed, returning error to LLM")
					toolMsg := Message{
						Role:       RoleTool,
						Content:    errMsg,
						ToolCallID: tc.ID,
						Name:       toolName,
					}
					messages = append(messages, toolMsg)
					continue // Skip execution, let AI retry with correct tool
				}
			}

			toolResult, queryResult := h.executeToolCall(ctx, chatCtx, msg.ConversationID, chatbot, &tc, userID, msg.Content)

			// Accumulate successful query results for persistence
			if queryResult != nil {
				accumulatedQueryResults = append(accumulatedQueryResults, *queryResult)
			}

			// Add tool result message
			toolMsg := Message{
				Role:       RoleTool,
				Content:    toolResult,
				ToolCallID: tc.ID,
				Name:       tc.Function.Name,
			}
			messages = append(messages, toolMsg)
		}

		// Continue loop to get AI's response to tool results
		log.Debug().
			Int("iteration", iteration+1).
			Int("tool_calls", len(pendingToolCalls)).
			Msg("Processed tool calls, continuing conversation")
	}

	// Send completion
	h.send(chatCtx, ServerMessage{
		Type:           "done",
		ConversationID: msg.ConversationID,
		Usage:          &totalUsage,
	})

	// Record metrics
	if h.metrics != nil {
		h.metrics.RecordAIChatRequest(chatbot.Name, "success", time.Since(start))
		h.metrics.RecordAITokens(chatbot.Name, totalUsage.PromptTokens, totalUsage.CompletionTokens)
	}

	log.Debug().
		Str("conversation_id", msg.ConversationID).
		Int("prompt_tokens", totalUsage.PromptTokens).
		Int("completion_tokens", totalUsage.CompletionTokens).
		Msg("Message processed")
}

// / executeToolCall executes a tool call and returns:
// - the result as a string for the AI
// - the QueryResult for persistence (nil if query failed or not a SQL query)
func (h *ChatHandler) executeToolCall(ctx context.Context, chatCtx *ChatContext, conversationID string, chatbot *Chatbot, toolCall *ToolCall, userID string, userMessage string) (string, *QueryResult) {
	toolName := toolCall.Function.Name

	// Check if this is an MCP tool
	if chatbot.HasMCPTools() && h.mcpExecutor != nil && IsToolAllowed(toolName, chatbot.MCPTools) {
		return h.executeMCPTool(ctx, chatCtx, conversationID, chatbot, toolCall)
	}

	// Dispatch based on tool name for legacy tools
	switch toolName {
	case "execute_sql":
		return h.executeSQLTool(ctx, chatCtx, conversationID, chatbot, toolCall, userID, userMessage)
	default:
		return fmt.Sprintf("Error: Unknown tool '%s'", toolName), nil
	}
}

// executeSQLTool handles the execute_sql tool call
func (h *ChatHandler) executeSQLTool(ctx context.Context, chatCtx *ChatContext, conversationID string, chatbot *Chatbot, toolCall *ToolCall, userID string, userMessage string) (string, *QueryResult) {
	// Parse arguments
	var args struct {
		SQL         string `json:"sql"`
		Description string `json:"description"`
	}

	if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
		log.Error().Err(err).Str("args", toolCall.Function.Arguments).Msg("Failed to parse SQL tool call arguments")
		return fmt.Sprintf("Error: Failed to parse tool arguments: %v", err), nil
	}

	// Intent validation (before execution)
	if len(chatbot.IntentRules) > 0 || len(chatbot.RequiredColumns) > 0 {
		intentValidator := NewIntentValidator(
			chatbot.IntentRules,
			chatbot.RequiredColumns,
			chatbot.DefaultTable,
		)

		// Pre-validate SQL to get tables accessed
		preValidator := NewSQLValidator(chatbot.AllowedSchemas, chatbot.AllowedTables, chatbot.AllowedOperations)
		preResult := preValidator.Validate(args.SQL)

		// Validate intent matches query
		intentResult := intentValidator.ValidateIntent(userMessage, args.SQL, preResult.TablesAccessed)
		if !intentResult.Valid {
			errMsg := fmt.Sprintf("Intent validation failed: %s", strings.Join(intentResult.Errors, "; "))
			if len(intentResult.Suggestions) > 0 {
				errMsg += fmt.Sprintf(" Suggestions: %s", strings.Join(intentResult.Suggestions, "; "))
			}
			log.Warn().
				Str("user_message", userMessage).
				Str("sql", args.SQL).
				Strs("errors", intentResult.Errors).
				Msg("Intent validation failed")
			return errMsg, nil
		}

		// Validate required columns
		colResult := intentValidator.ValidateRequiredColumns(args.SQL, preResult.TablesAccessed)
		if !colResult.Valid {
			errMsg := fmt.Sprintf("Required columns missing: %s", strings.Join(colResult.Errors, "; "))
			if len(colResult.Suggestions) > 0 {
				errMsg += fmt.Sprintf(" Suggestions: %s", strings.Join(colResult.Suggestions, "; "))
			}
			log.Warn().
				Str("sql", args.SQL).
				Strs("errors", colResult.Errors).
				Msg("Required columns validation failed")
			return errMsg, nil
		}
	}

	// Send progress
	h.sendProgress(chatCtx, conversationID, "querying", fmt.Sprintf("Executing: %s", args.Description))

	// Execute SQL
	execReq := &ExecuteRequest{
		ChatbotName:       chatbot.Name,
		ChatbotID:         chatbot.ID,
		ConversationID:    conversationID,
		UserID:            userID,
		Role:              chatCtx.Role,
		Claims:            chatCtx.Claims,
		SQL:               args.SQL,
		Description:       args.Description,
		AllowedSchemas:    chatbot.AllowedSchemas,
		AllowedTables:     chatbot.AllowedTables,
		AllowedOperations: chatbot.AllowedOperations,
	}

	result, err := h.executor.Execute(ctx, execReq)
	if err != nil {
		log.Error().Err(err).Msg("SQL execution error")
		return fmt.Sprintf("Error executing query: %v", err), nil
	}

	// Log to audit (unless execution logs are disabled)
	if !chatbot.DisableExecutionLogs {
		_ = h.auditLogger.LogFromExecuteResult(
			ctx,
			chatbot.ID, conversationID, "", userID,
			args.SQL, result,
			chatCtx.Role, chatCtx.IPAddress, chatCtx.UserAgent,
		)

		// Log to central logging service
		if h.loggingService != nil {
			h.loggingService.LogAI(ctx, map[string]any{
				"tool":            "execute_sql",
				"chatbot_id":      chatbot.ID,
				"conversation_id": conversationID,
				"success":         result.Success,
				"rows_returned":   result.RowCount,
				"tables":          result.TablesAccessed,
				"duration_ms":     result.DurationMs,
			}, "", userID)
		}
	}

	// Send result to client for display
	h.send(chatCtx, ServerMessage{
		Type:           "query_result",
		ConversationID: conversationID,
		Query:          args.SQL,
		Summary:        result.Summary,
		RowCount:       result.RowCount,
		Data:           result.Rows,
	})

	// Return result as string for AI to interpret
	if !result.Success {
		return fmt.Sprintf("Query failed: %s", result.Error), nil
	}

	// Build QueryResult for persistence
	queryResult := &QueryResult{
		Query:    args.SQL,
		Summary:  result.Summary,
		RowCount: result.RowCount,
		Data:     result.Rows,
	}

	// Format result for AI - include summary and sample data
	resultStr := fmt.Sprintf("Query executed successfully. %s\n", result.Summary)
	if len(result.Rows) > 0 {
		// Include first few rows as JSON for context
		maxRows := 5
		if len(result.Rows) < maxRows {
			maxRows = len(result.Rows)
		}
		sampleData, _ := json.Marshal(result.Rows[:maxRows])
		resultStr += fmt.Sprintf("Sample data (first %d rows): %s", maxRows, string(sampleData))
	}

	return resultStr, queryResult
}

// executeMCPTool handles MCP tool execution for chatbots with MCP tools configured
func (h *ChatHandler) executeMCPTool(ctx context.Context, chatCtx *ChatContext, conversationID string, chatbot *Chatbot, toolCall *ToolCall) (string, *QueryResult) {
	toolName := toolCall.Function.Name

	// Parse tool arguments
	var args map[string]any
	if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
		log.Error().Err(err).Str("tool", toolName).Str("args", toolCall.Function.Arguments).Msg("Failed to parse MCP tool arguments")
		return fmt.Sprintf("Error: Failed to parse tool arguments: %v", err), nil
	}

	// Send progress to client
	progressMsg := fmt.Sprintf("Executing %s...", toolName)
	if tableName, ok := args["table"].(string); ok && tableName != "" {
		progressMsg = fmt.Sprintf("Executing %s on %s...", toolName, tableName)
	}
	h.sendProgress(chatCtx, conversationID, "executing", progressMsg)

	// Execute the MCP tool
	result, err := h.mcpExecutor.ExecuteTool(ctx, toolName, args, chatCtx, chatbot)
	if err != nil {
		log.Error().Err(err).Str("tool", toolName).Msg("MCP tool execution error")
		return fmt.Sprintf("Error executing %s: %v", toolName, err), nil
	}

	if result.IsError {
		log.Warn().Str("tool", toolName).Str("error", result.Content).Msg("MCP tool returned error")
		return fmt.Sprintf("Error: %s", result.Content), nil
	}

	// Log successful execution
	log.Debug().
		Str("chatbot", chatbot.Name).
		Str("tool", toolName).
		Int("result_length", len(result.Content)).
		Msg("MCP tool executed successfully")

	// Parse query results for data-returning tools
	var queryResult *QueryResult
	if toolName == "query_table" {
		queryResult = h.parseMCPQueryResult(toolName, args, result.Content)
	} else if toolName == "execute_sql" {
		queryResult = h.parseMCPExecuteSQLResult(args, result.Content)
	}

	// Build server message
	serverMsg := ServerMessage{
		Type:           "tool_result",
		ConversationID: conversationID,
		Message:        toolName,
	}

	// Add structured fields for execute_sql
	if toolName == "execute_sql" && queryResult != nil {
		serverMsg.Query = queryResult.Query
		serverMsg.Summary = queryResult.Summary
		serverMsg.RowCount = queryResult.RowCount
		serverMsg.Data = queryResult.Data
	} else {
		serverMsg.Data = []map[string]any{{"tool": toolName, "result": result.Content}}
	}

	h.send(chatCtx, serverMsg)

	return result.Content, queryResult
}

// parseMCPQueryResult attempts to parse MCP query results for persistence
func (h *ChatHandler) parseMCPQueryResult(toolName string, args map[string]any, resultContent string) *QueryResult {
	// Try to parse the result as JSON array
	var rows []map[string]any
	if err := json.Unmarshal([]byte(resultContent), &rows); err != nil {
		// Not valid JSON array, skip persistence
		return nil
	}

	// Build a description for the query
	tableName := ""
	if t, ok := args["table"].(string); ok {
		tableName = t
	}

	return &QueryResult{
		Query:    fmt.Sprintf("MCP %s on %s", toolName, tableName),
		Summary:  fmt.Sprintf("Query returned %d row(s)", len(rows)),
		RowCount: len(rows),
		Data:     rows,
	}
}

// parseMCPExecuteSQLResult parses execute_sql MCP tool results for persistence
func (h *ChatHandler) parseMCPExecuteSQLResult(args map[string]any, resultContent string) *QueryResult {
	// Parse the result JSON from the MCP tool
	var execResult struct {
		Success    bool             `json:"success"`
		RowCount   int              `json:"row_count"`
		Columns    []string         `json:"columns"`
		Rows       []map[string]any `json:"rows"`
		Summary    string           `json:"summary"`
		DurationMs int64            `json:"duration_ms"`
		Tables     []string         `json:"tables"`
	}

	if err := json.Unmarshal([]byte(resultContent), &execResult); err != nil {
		return nil
	}

	if !execResult.Success {
		return nil
	}

	// Extract SQL query from tool arguments
	sqlQuery := ""
	if sql, ok := args["sql"].(string); ok {
		sqlQuery = sql
	}

	return &QueryResult{
		Query:    sqlQuery,
		Summary:  execResult.Summary,
		RowCount: execResult.RowCount,
		Data:     execResult.Rows,
	}
}

// parseURLForLogging extracts domain and path from URL for safe logging (no query params)
func parseURLForLogging(rawURL string) (string, error) {
	// Simple extraction of scheme + host + path without query params
	if idx := strings.Index(rawURL, "?"); idx != -1 {
		return rawURL[:idx], nil
	}
	return rawURL, nil
}

// handleCancel handles cancellation of a generation
func (h *ChatHandler) handleCancel(chatCtx *ChatContext, msg *ClientMessage) {
	// Cancel current generation (if using cancellable context)
	if chatCtx.Cancel != nil {
		chatCtx.Cancel()
	}

	h.send(chatCtx, ServerMessage{
		Type:           "cancelled",
		ConversationID: msg.ConversationID,
	})
}

// Helper methods

func (h *ChatHandler) send(chatCtx *ChatContext, msg ServerMessage) {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Error().Err(err).Msg("Failed to marshal server message")
		return
	}

	if err := chatCtx.Conn.WriteMessage(websocket.TextMessage, data); err != nil {
		log.Error().Err(err).Msg("Failed to send WebSocket message")
	}
}

func (h *ChatHandler) sendError(chatCtx *ChatContext, conversationID, code, message string) {
	h.send(chatCtx, ServerMessage{
		Type:           "error",
		ConversationID: conversationID,
		Code:           code,
		Error:          message,
	})
}

func (h *ChatHandler) sendProgress(chatCtx *ChatContext, conversationID, step, message string) {
	h.send(chatCtx, ServerMessage{
		Type:           "progress",
		ConversationID: conversationID,
		Step:           step,
		Message:        message,
	})
}

func (h *ChatHandler) getProvider(ctx context.Context, chatbot *Chatbot) (Provider, error) {
	// Check if chatbot has a specific provider configured
	if chatbot != nil && chatbot.ProviderID != nil && *chatbot.ProviderID != "" {
		providerID := *chatbot.ProviderID

		// Check cache first
		h.providersMu.RLock()
		if provider, ok := h.providers[providerID]; ok {
			h.providersMu.RUnlock()
			return provider, nil
		}
		h.providersMu.RUnlock()

		// Load chatbot-specific provider from database
		providerRecord, err := h.storage.GetProvider(ctx, providerID)
		if err != nil {
			log.Warn().
				Err(err).
				Str("chatbot", chatbot.Name).
				Str("provider_id", providerID).
				Msg("Failed to get chatbot's configured provider, falling back to default")
		} else if providerRecord == nil {
			log.Warn().
				Str("chatbot", chatbot.Name).
				Str("provider_id", providerID).
				Msg("Chatbot's configured provider not found, falling back to default")
		} else if !providerRecord.Enabled {
			log.Warn().
				Str("chatbot", chatbot.Name).
				Str("provider_id", providerID).
				Str("provider_name", providerRecord.Name).
				Msg("Chatbot's configured provider is disabled, falling back to default")
		} else {
			// Create and cache the chatbot-specific provider
			return h.createAndCacheProvider(providerRecord)
		}
		// Fall through to default provider logic if chatbot's provider is unavailable
	}

	// Check if we have any cached providers (use as default)
	h.providersMu.RLock()
	if len(h.providers) > 0 {
		for _, p := range h.providers {
			h.providersMu.RUnlock()
			return p, nil
		}
	}
	h.providersMu.RUnlock()

	// Load default provider from database
	providerRecord, err := h.storage.GetDefaultProvider(ctx)
	if err != nil {
		return nil, err
	}

	if providerRecord == nil {
		// Fallback: if there's only one enabled provider, use it
		allProviders, listErr := h.storage.ListProviders(ctx, true)
		if listErr == nil && len(allProviders) == 1 {
			providerRecord = allProviders[0]
			log.Info().Str("provider", providerRecord.Name).Msg("Using only available provider as default")
		} else {
			return nil, fmt.Errorf("no default AI provider configured")
		}
	}

	return h.createAndCacheProvider(providerRecord)
}

// createAndCacheProvider creates a provider from a record and caches it
func (h *ChatHandler) createAndCacheProvider(providerRecord *ProviderRecord) (Provider, error) {
	providerConfig := ProviderConfig{
		Name:        providerRecord.Name,
		DisplayName: providerRecord.DisplayName,
		Type:        ProviderType(providerRecord.ProviderType),
		Config:      providerRecord.Config,
	}

	if providerRecord.Config != nil {
		if model, ok := providerRecord.Config["model"]; ok {
			providerConfig.Model = model
		}
	}

	provider, err := NewProvider(providerConfig)
	if err != nil {
		return nil, err
	}

	// Cache provider by ID
	h.providersMu.Lock()
	h.providers[providerRecord.ID] = provider
	h.providersMu.Unlock()

	return provider, nil
}

// Helper functions

func extractString(v interface{}) *string {
	if v == nil {
		return nil
	}
	if s, ok := v.(string); ok {
		return &s
	}
	return nil
}

func extractStringDefault(v interface{}, defaultVal string) string {
	if v == nil {
		return defaultVal
	}
	if s, ok := v.(string); ok {
		return s
	}
	return defaultVal
}
