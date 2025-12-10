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
	"github.com/fluxbase-eu/fluxbase/internal/observability"
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
)

// ChatHandler handles WebSocket chat connections
type ChatHandler struct {
	storage       *Storage
	conversations *ConversationManager
	schemaBuilder *SchemaBuilder
	executor      *Executor
	httpHandler   *HttpRequestHandler
	auditLogger   *AuditLogger
	metrics       *observability.Metrics
	config        *config.AIConfig
	providers     map[string]Provider
	providersMu   sync.RWMutex
}

// NewChatHandler creates a new chat handler
func NewChatHandler(
	db *database.Connection,
	storage *Storage,
	conversations *ConversationManager,
	metrics *observability.Metrics,
	cfg *config.AIConfig,
) *ChatHandler {
	return &ChatHandler{
		storage:       storage,
		conversations: conversations,
		schemaBuilder: NewSchemaBuilder(db),
		executor:      NewExecutor(db, metrics, cfg.MaxRowsPerQuery, cfg.QueryTimeout),
		httpHandler:   NewHttpRequestHandler(),
		auditLogger:   NewAuditLogger(db),
		metrics:       metrics,
		config:        cfg,
		providers:     make(map[string]Provider),
	}
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

	// Build messages for LLM
	messages := []Message{
		{Role: RoleSystem, Content: systemPrompt},
	}

	// Add conversation history
	for _, m := range state.Messages {
		messages = append(messages, m)
	}

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

	for iteration := 0; iteration < maxIterations; iteration++ {
		// Build tools list based on chatbot configuration
		tools := []Tool{ExecuteSQLTool}
		if len(chatbot.HTTPAllowedDomains) > 0 {
			tools = append(tools, HttpRequestTool)
		}

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
					if toolName == "execute_sql" || toolName == "http_request" {
						pendingToolCalls = append(pendingToolCalls, ToolCall{
							ID:   event.ToolCall.ID,
							Type: "function",
							Function: FunctionCall{
								Name:      toolName,
								Arguments: event.ToolCall.ArgumentsDelta,
							},
						})
					}
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
			toolResult, queryResult := h.executeToolCall(ctx, chatCtx, msg.ConversationID, chatbot, &tc, userID)

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

// executeToolCall executes a tool call and returns:
// - the result as a string for the AI
// - the QueryResult for persistence (nil if query failed or not a SQL query)
func (h *ChatHandler) executeToolCall(ctx context.Context, chatCtx *ChatContext, conversationID string, chatbot *Chatbot, toolCall *ToolCall, userID string) (string, *QueryResult) {
	// Dispatch based on tool name
	switch toolCall.Function.Name {
	case "execute_sql":
		return h.executeSQLTool(ctx, chatCtx, conversationID, chatbot, toolCall, userID)
	case "http_request":
		return h.executeHttpTool(ctx, chatCtx, conversationID, chatbot, toolCall)
	default:
		return fmt.Sprintf("Error: Unknown tool '%s'", toolCall.Function.Name), nil
	}
}

// executeSQLTool handles the execute_sql tool call
func (h *ChatHandler) executeSQLTool(ctx context.Context, chatCtx *ChatContext, conversationID string, chatbot *Chatbot, toolCall *ToolCall, userID string) (string, *QueryResult) {
	// Parse arguments
	var args struct {
		SQL         string `json:"sql"`
		Description string `json:"description"`
	}

	if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
		log.Error().Err(err).Str("args", toolCall.Function.Arguments).Msg("Failed to parse SQL tool call arguments")
		return fmt.Sprintf("Error: Failed to parse tool arguments: %v", err), nil
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

	// Log to audit
	_ = h.auditLogger.LogFromExecuteResult(
		ctx,
		chatbot.ID, conversationID, "", userID,
		args.SQL, result,
		chatCtx.Role, chatCtx.IPAddress, chatCtx.UserAgent,
	)

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

// executeHttpTool handles the http_request tool call
func (h *ChatHandler) executeHttpTool(ctx context.Context, chatCtx *ChatContext, conversationID string, chatbot *Chatbot, toolCall *ToolCall) (string, *QueryResult) {
	// Parse arguments
	var args struct {
		URL    string `json:"url"`
		Method string `json:"method"`
	}

	if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
		log.Error().Err(err).Str("args", toolCall.Function.Arguments).Msg("Failed to parse HTTP tool call arguments")
		return fmt.Sprintf("Error: Failed to parse tool arguments: %v", err), nil
	}

	// Send progress
	h.sendProgress(chatCtx, conversationID, "http_request", fmt.Sprintf("Making HTTP request to %s", args.URL))

	// Execute HTTP request
	result := h.httpHandler.Execute(ctx, args.URL, args.Method, chatbot.HTTPAllowedDomains)

	// Log to audit (domain + path only, no query params for privacy)
	logPath := args.URL
	if parsedURL, err := parseURLForLogging(args.URL); err == nil {
		logPath = parsedURL
	}

	log.Info().
		Str("chatbot_id", chatbot.ID).
		Str("conversation_id", conversationID).
		Str("request_path", logPath).
		Bool("success", result.Success).
		Int("status", result.Status).
		Str("error", result.Error).
		Msg("HTTP request tool executed")

	// Return result as JSON string for AI
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return fmt.Sprintf("Error: Failed to serialize result: %v", err), nil
	}

	return string(resultJSON), nil
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
	h.providersMu.RLock()
	if len(h.providers) > 0 {
		// Return first provider (TODO: handle chatbot-specific providers)
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

	// Create provider
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

	// Cache provider
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
