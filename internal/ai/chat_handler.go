package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/gofiber/contrib/v3/websocket"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/ai/integrations"
	"github.com/nimbleflux/fluxbase/internal/auth"
	"github.com/nimbleflux/fluxbase/internal/config"
	"github.com/nimbleflux/fluxbase/internal/database"
	"github.com/nimbleflux/fluxbase/internal/logging"
	"github.com/nimbleflux/fluxbase/internal/mcp"
	"github.com/nimbleflux/fluxbase/internal/observability"
)

// ChatHandler handles WebSocket chat connections
type ChatHandler struct {
	storage         *Storage
	conversations   *ConversationManager
	schemaBuilder   *SchemaBuilder
	executor        *Executor
	auditLogger     *AuditLogger
	toolAuditLogger *ToolAuditLogger
	ragService      *RAGService
	loggingService *logging.Service
	metrics        *observability.Metrics
	config         *config.AIConfig
	providers      map[string]Provider
	providersMu    sync.RWMutex
	limiter        *ChatbotLimiter
	// MCP integration
	mcpExecutor *MCPToolExecutor
	// Tool integrations (Web Agent). nil when AI is disabled or no
	// integrations are configured — the supervisor silently excludes
	// "web" from the route in that case.
	integrationsStorage *integrations.Storage
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
		// Initialize knowledge graph and entity extractor for graph-boosted search
		knowledgeGraph := NewKnowledgeGraph(kbStorage)
		entityExtractor := NewRuleBasedExtractor()
		ragService = NewRAGService(kbStorage, embeddingService, knowledgeGraph, entityExtractor)
	}

	return &ChatHandler{
		storage:        storage,
		conversations:  conversations,
		schemaBuilder:   NewSchemaBuilder(db),
		executor:        NewExecutor(db, metrics, cfg.MaxRowsPerQuery, cfg.QueryTimeout),
		auditLogger:     NewAuditLogger(db),
		toolAuditLogger: NewToolAuditLogger(db),
		ragService:      ragService,
		loggingService: loggingService,
		metrics:        metrics,
		config:         cfg,
		providers:      make(map[string]Provider),
		limiter:        NewChatbotLimiter(),
	}
}

// SetSettingsResolver sets the settings resolver for template variable resolution in system prompts
func (h *ChatHandler) SetSettingsResolver(resolver *SettingsResolver) {
	h.schemaBuilder.SetSettingsResolver(resolver)
}

// SetIntegrationsStorage wires the tool integrations storage (Tavily,
// future Brave/Jina) into the chat handler. The supervisor's Web Agent
// reads from this to resolve credentials at turn start. nil-safe — when
// never called, the Web Agent is silently excluded from routes.
func (h *ChatHandler) SetIntegrationsStorage(s *integrations.Storage) {
	h.integrationsStorage = s
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

// InvalidateProvider drops a cached provider so the next request reloads it
// from the database. Called by the admin Handler when a provider is mutated
// (config update, deletion, default change). Without this, rotated API keys
// and changed models never take effect for the chat path.
func (h *ChatHandler) InvalidateProvider(providerID string) {
	if providerID == "" {
		return
	}
	h.providersMu.Lock()
	delete(h.providers, providerID)
	h.providersMu.Unlock()
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
	// PageContext is the optional page-context string sent per-message by
	// clients that embed a single chatbot across multiple app pages. When
	// present, the supervisor looks up the matching PageProfile (if any)
	// and uses it to bias routing and override per-page config. Missing or
	// unknown values fall back to the chatbot's global config.
	PageContext string `json:"page_context,omitempty"`
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
	// MatchedIntentRules surfaces the intent rules that fired for this turn
	// (Add 5). Empty/omitted when no rules match or no rules are configured.
	MatchedIntentRules []MatchedIntentRule `json:"matched_intent_rules,omitempty"`
	// DailyQuota carries the per-user remaining quota snapshot at turn end
	// (Ask 2). Omitted when no daily limits are configured for the chatbot.
	DailyQuota *DailyQuotaSnapshot `json:"daily_quota,omitempty"`
	// AgentTransition is set on agent_transition event types, emitted by
	// the supervisor graph when one agent hands off to another. Optional
	// observability for clients that want to render the routing flow.
	AgentTransition *AgentTransition `json:"agent_transition,omitempty"`
	// Agent names the currently-active specialist in agent_transition /
	// agent_complete events.
	Agent string `json:"agent,omitempty"`
	// PageContext echoes the page_context string from the client's message
	// back in agent_transition events, so multi-page clients can correlate.
	PageContext string `json:"page_context,omitempty"`
	// AgentThought carries one piece of agent reasoning (routing plan,
	// streamed thought chunk, tool call decision, tool result summary).
	// Emitted as agent_thought events; clients render them as the
	// thought-process stream alongside the final response. Suppressed
	// when chatbot.ShowReasoning is false (only reasoning kind —
	// tool_call/tool_result events still emit so users see actions).
	AgentThought *AgentThought `json:"agent_thought,omitempty"`
	Error        string        `json:"error,omitempty"`
	Code         string        `json:"code,omitempty"`
}

// DailyQuotaSnapshot is a point-in-time view of the per-user daily counters
// for a chatbot. Surfaced in the done event so clients can render
// "X requests/tokens left today" without an extra round-trip.
type DailyQuotaSnapshot struct {
	Requests Quota  `json:"requests"`
	Tokens   Quota  `json:"tokens"`
	ResetsAt string `json:"resets_at,omitempty"` // RFC3339; empty when unknown
}

// Quota is one counter half of a DailyQuotaSnapshot.
type Quota struct {
	Used  int `json:"used"`
	Limit int `json:"limit"`
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
func (h *ChatHandler) HandleWebSocket(c fiber.Ctx) error {
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

	if tenantID := extractStringDefault(c.Locals("tenant_id"), ""); tenantID != "" {
		ctx = database.ContextWithTenant(ctx, tenantID)
	}

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
		// Only instance_admin can impersonate
		if chatCtx.Role != "instance_admin" {
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

	// Check role-based access control
	if len(chatbot.RequireRoles) > 0 && chatCtx.Claims != nil {
		if !hasRequiredRole(chatCtx.Claims, chatbot.RequireRoles) {
			h.sendError(chatCtx, "", "FORBIDDEN", "Insufficient role to access this chatbot")
			return
		}
	}

	// Determine user identifier for rate limiting
	userIdentifier := "anonymous"
	if chatCtx.UserID != nil {
		userIdentifier = *chatCtx.UserID
	}

	// Check per-minute rate limit
	if !h.limiter.CheckRateLimit(chatbot.ID, userIdentifier, chatbot.RateLimitPerMinute) {
		h.sendError(chatCtx, "", "RATE_LIMITED", "Rate limit exceeded. Please try again later.")
		return
	}

	// Check daily request limit
	if !h.limiter.CheckDailyRequestLimit(chatbot.ID, userIdentifier, chatbot.DailyRequestLimit) {
		h.sendError(chatCtx, "", "DAILY_LIMIT", "Daily request limit exceeded.")
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
		switch {
		case err != nil:
			log.Warn().
				Err(err).
				Str("chatbot", chatbot.Name).
				Str("provider_id", providerID).
				Msg("Failed to get chatbot's configured provider, falling back to default")
		case providerRecord == nil:
			log.Warn().
				Str("chatbot", chatbot.Name).
				Str("provider_id", providerID).
				Msg("Chatbot's configured provider not found, falling back to default")
		case !providerRecord.Enabled:
			log.Warn().
				Str("chatbot", chatbot.Name).
				Str("provider_id", providerID).
				Str("provider_name", providerRecord.Name).
				Msg("Chatbot's configured provider is disabled, falling back to default")
		default:
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

// hasRequiredRole checks if the user's JWT claims contain any of the required roles (OR semantics).
// It checks the "role" claim directly and also looks for a "roles" key in app_metadata.
func hasRequiredRole(claims *auth.TokenClaims, requiredRoles []string) bool {
	if claims == nil || len(requiredRoles) == 0 {
		return false
	}

	roleSet := make(map[string]bool, len(requiredRoles))
	for _, r := range requiredRoles {
		roleSet[r] = true
	}

	if claims.Role != "" && roleSet[claims.Role] {
		return true
	}

	if claims.AppMetadata != nil {
		if metaMap, ok := claims.AppMetadata.(map[string]interface{}); ok {
			if rolesVal, exists := metaMap["roles"]; exists {
				switch rv := rolesVal.(type) {
				case []string:
					for _, r := range rv {
						if roleSet[r] {
							return true
						}
					}
				case []interface{}:
					for _, r := range rv {
						if rs, ok := r.(string); ok && roleSet[rs] {
							return true
						}
					}
				}
			}
		}
	}

	return false
}
