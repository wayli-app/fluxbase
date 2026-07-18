package ai

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/auth"
)

// canRunSupervisor returns true if the chatbot has the minimum dependencies
// required to run the supervisor pipeline (a provider, at minimum). Used to
// decide whether to attempt the supervisor path or fall back to ReAct.
func (h *ChatHandler) canRunSupervisor(chatbot *Chatbot) bool {
	if chatbot == nil {
		return false
	}
	if chatbot.ReasoningMode != "supervisor" {
		return false
	}
	// Provider check happens lazily inside runSupervisorTurn — if it can't
	// resolve a provider, it returns supervisorRan=false and we fall through.
	return true
}

// runSupervisorTurn executes one user turn via the supervisor graph.
//
// Returns (ran, err):
//   - ran=true, err=nil: supervisor handled the turn completely. Caller returns.
//   - ran=true, err!=nil: supervisor started but a node failed. Caller logs
//     and falls back to the legacy ReAct loop for safety.
//   - ran=false: supervisor couldn't start (e.g., no provider). Caller
//     falls through to the legacy path silently.
func (h *ChatHandler) runSupervisorTurn(ctx context.Context, chatCtx *ChatContext, chatbot *Chatbot, msg *ClientMessage, userID string, provider Provider) (bool, error) {
	if provider == nil {
		return false, nil
	}

	// Build initial state
	state := NewState()
	state.SetUserMessage(msg.Content)
	state.SetPageContext(msg.PageContext)
	state.SetConversationHistory(state.ConversationHistory()) // placeholder, real history below

	// Get conversation history (last few messages) — bounded so token costs
	// don't explode for long conversations.
	if state2, ok := chatCtx.Conversations[msg.ConversationID]; ok && len(state2.Messages) > 0 {
		state.SetConversationHistory(tailMessages(state2.Messages, 6))
	}

	// Resolve page profile from chatbot config (if any)
	if profile := chatbot.PageProfiles.Resolve(msg.PageContext); profile != nil {
		state.SetPageProfile(profile)
	}

	// Build deps bundle for this turn
	deps := &AgentDeps{
		Chatbot:        chatbot,
		Provider:       provider,
		UserID:         userID,
		Role:           chatCtx.Role,
		Claims:         chatCtx.Claims,
		PageProfile:    state.PageProfile(),
		SQLExecutor:    h.executor,
		SchemaBuilder:  h.schemaBuilder,
		RAGService:     h.ragService,
		MCPExecutor:    h.mcpExecutor,
		ConversationID: msg.ConversationID,
		Sender: &chatHandlerSender{
			h:              h,
			chatCtx:        chatCtx,
			conversationID: msg.ConversationID,
			pageContext:    msg.PageContext,
		},
	}

	graph := NewSupervisorGraph(deps)
	if err := graph.Run(ctx, state); err != nil {
		return true, fmt.Errorf("supervisor graph failed: %w", err)
	}

	// Stream the final response to the client. The supervisor path doesn't
	// stream token-by-token (each node uses non-streaming calls for
	// simplicity). If token-streaming is needed in the future, the
	// synthesizer / single-agent path would switch to ChatStream.
	finalResponse := state.FinalResponse()
	if finalResponse == "" {
		finalResponse = "I wasn't able to generate a response. Please try again."
	}

	// Send content to client as one chunk (no streaming).
	h.send(chatCtx, ServerMessage{
		Type:           "content",
		ConversationID: msg.ConversationID,
		Delta:          finalResponse,
	})

	// Save assistant message with query results
	assistantMsg := Message{
		Role:         RoleAssistant,
		Content:      finalResponse,
		QueryResults: state.ToolResults(),
	}
	totalUsage := state.Usage()
	_ = h.conversations.AddMessage(ctx, msg.ConversationID, assistantMsg, totalUsage.PromptTokens, totalUsage.CompletionTokens)

	// Track token usage for daily budget enforcement
	if chatbot.DailyTokenBudget > 0 {
		userIdentifier := "anonymous"
		if chatCtx.UserID != nil {
			userIdentifier = *chatCtx.UserID
		}
		effective := totalUsage.PromptTokens - totalUsage.CachedTokens + totalUsage.CompletionTokens
		if effective < 0 {
			effective = 0
		}
		h.limiter.AddTokenUsage(chatbot.ID, userIdentifier, effective)
	}

	// Build per-user daily quota snapshot for the done event
	var dailyQuota *DailyQuotaSnapshot
	if chatbot.DailyRequestLimit > 0 || chatbot.DailyTokenBudget > 0 {
		userIdentifier := "anonymous"
		if chatCtx.UserID != nil {
			userIdentifier = *chatCtx.UserID
		}
		usage := h.limiter.GetDailyUsage(chatbot.ID, userIdentifier, chatbot.DailyRequestLimit, chatbot.DailyTokenBudget)
		dailyQuota = &DailyQuotaSnapshot{
			Requests: Quota{Used: usage.RequestsUsed, Limit: usage.RequestsLimit},
			Tokens:   Quota{Used: usage.TokensUsed, Limit: usage.TokensLimit},
			ResetsAt: usage.ResetsAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	h.send(chatCtx, ServerMessage{
		Type:           "done",
		ConversationID: msg.ConversationID,
		Usage:          &totalUsage,
		DailyQuota:     dailyQuota,
		PageContext:    msg.PageContext,
	})

	// Record metrics
	if h.metrics != nil {
		h.metrics.RecordAIChatRequest(chatbot.Name, "success", 0)
		h.metrics.RecordAITokens(chatbot.Name, totalUsage.PromptTokens, totalUsage.CompletionTokens)
	}

	log.Debug().
		Str("conversation_id", msg.ConversationID).
		Int("prompt_tokens", totalUsage.PromptTokens).
		Int("completion_tokens", totalUsage.CompletionTokens).
		Msg("Supervisor turn processed")

	return true, nil
}

// chatHandlerSender bridges AgentEventSender to the ChatHandler's WS send.
// Each agent calls these methods to emit events to the connected client.
type chatHandlerSender struct {
	h              *ChatHandler
	chatCtx        *ChatContext
	conversationID string
	pageContext    string
}

// SendProgress emits a progress event.
func (s *chatHandlerSender) SendProgress(ctx context.Context, conversationID, step, message string) {
	if s == nil || s.h == nil {
		return
	}
	s.h.sendProgress(s.chatCtx, conversationID, step, message)
}

// SendContent streams a content delta to the client.
func (s *chatHandlerSender) SendContent(ctx context.Context, conversationID, delta string) {
	if s == nil || s.h == nil {
		return
	}
	s.h.send(s.chatCtx, ServerMessage{
		Type:           "content",
		ConversationID: conversationID,
		Delta:          delta,
	})
}

// SendQueryResult emits a structured query_result event.
func (s *chatHandlerSender) SendQueryResult(ctx context.Context, conversationID string, result QueryResult) {
	if s == nil || s.h == nil {
		return
	}
	s.h.send(s.chatCtx, ServerMessage{
		Type:           "query_result",
		ConversationID: conversationID,
		Query:          result.Query,
		Summary:        result.Summary,
		RowCount:       result.RowCount,
		Data:           result.Data,
	})
}

// SendAgentTransition emits an agent_transition event with the optional
// page_context echo so multi-page clients can correlate.
func (s *chatHandlerSender) SendAgentTransition(ctx context.Context, conversationID string, transition AgentTransition) {
	if s == nil || s.h == nil {
		return
	}
	transition.PageContext = s.pageContext
	s.h.send(s.chatCtx, ServerMessage{
		Type:            "agent_transition",
		ConversationID:  conversationID,
		Agent:           transition.To,
		AgentTransition: &transition,
		PageContext:     s.pageContext,
	})
}

// Compile-time assertion that chatHandlerSender satisfies AgentEventSender.
var _ AgentEventSender = (*chatHandlerSender)(nil)

// Unused import guard — auth is imported for the AgentDeps.Claims field
// type. Keep the import even when other code paths don't reference it
// directly here.
var _ = (*auth.TokenClaims)(nil)
