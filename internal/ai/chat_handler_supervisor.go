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

	// Read conversation history from the live chat session. Depth respects
	// the chatbot's MaxConversationTurns (each turn = 1 user + 1 assistant
	// message, so multiply by 2). Fallback to 6 when unset.
	historyDepth := 6
	if chatbot.MaxConversationTurns > 0 {
		historyDepth = chatbot.MaxConversationTurns * 2
	}
	if state2, ok := chatCtx.Conversations[msg.ConversationID]; ok && len(state2.Messages) > 0 {
		state.SetConversationHistory(tailMessages(state2.Messages, historyDepth))
	}

	// Resolve page profile from chatbot config (if any)
	if profile := chatbot.PageProfiles.Resolve(msg.PageContext); profile != nil {
		state.SetPageProfile(profile)
	}

	// Build deps bundle for this turn
	deps := &AgentDeps{
		Chatbot:         chatbot,
		Provider:        provider,
		UserID:          userID,
		Role:            chatCtx.Role,
		Claims:          chatCtx.Claims,
		PageProfile:     state.PageProfile(),
		SQLExecutor:     h.executor,
		SchemaBuilder:   h.schemaBuilder,
		RAGService:      h.ragService,
		MCPExecutor:     h.mcpExecutor,
		Integrations:    h.integrationsStorage,
		ChatCtx:         chatCtx,
		ConversationID:  msg.ConversationID,
		ToolAuditLogger: h.toolAuditLogger,
		Sender: &chatHandlerSender{
			h:              h,
			chatCtx:        chatCtx,
			conversationID: msg.ConversationID,
			pageContext:    msg.PageContext,
			showReasoning:  chatbot.ShowReasoning,
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

	// Save assistant message with query results + supervisor turn metadata.
	// Metadata captures per-agent outputs and the supervisor plan so later
	// turns (and debugging tools) can recover what each specialist concluded
	// without re-reading the whole conversation.
	assistantMsg := Message{
		Role:         RoleAssistant,
		Content:      finalResponse,
		QueryResults: state.ToolResults(),
		Metadata:     buildSupervisorTurnMetadata(state),
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

// buildSupervisorTurnMetadata captures the supervisor's routing decision
// and each specialist agent's output for the turn. Stored on the persisted
// assistant message's metadata field so subsequent turns (and debugging
// tools) can recover what each agent concluded.
func buildSupervisorTurnMetadata(state *State) map[string]any {
	meta := map[string]any{}

	// Supervisor plan
	if v, ok := state.Get(SupervisorPlanKey); ok {
		if plan, ok := v.(*SupervisorPlan); ok && plan != nil {
			meta["supervisor_plan"] = plan
		}
	}

	// Per-agent outputs
	outputs := state.AgentOutputs()
	if len(outputs) > 0 {
		agentOutputs := make(map[string]string, len(outputs))
		for _, o := range outputs {
			// Last write wins per agent name — specialists only Run once
			// per turn, so this is well-defined.
			agentOutputs[o.Name] = o.Content
		}
		meta["agent_outputs"] = agentOutputs
	}

	// Detected language
	if lang := state.UserLanguage(); lang != "" {
		meta["user_language"] = lang
	}

	// Verifier report
	if v, ok := state.Get(VerifyReportKey); ok {
		if report, ok := v.(*VerifyReport); ok && report != nil {
			meta["verify_report"] = map[string]any{
				"language_ok":  report.LanguageOK,
				"grounding_ok": report.GroundingOK,
				"issues":       report.Issues,
			}
		}
	}

	if len(meta) == 0 {
		return nil
	}
	return meta
}

// chatHandlerSender bridges AgentEventSender to the ChatHandler's WS send.
// Each agent calls these methods to emit events to the connected client.
//
// ShowReasoning controls whether kind=reasoning agent_thought events are
// emitted. When false (the chatbot's @fluxbase:show-reasoning false), only
// structural events (plan / tool_call / tool_result) are emitted — clients
// still see what tools the agent ran, but not the streamed reasoning text.
type chatHandlerSender struct {
	h              *ChatHandler
	chatCtx        *ChatContext
	conversationID string
	pageContext    string
	showReasoning  bool
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

// SendAgentThought emits an agent_thought event. Reasoning chunks are
// suppressed when the chatbot has @fluxbase:show-reasoning false. Other
// kinds (plan / tool_call / tool_result) always emit so clients can
// still render the action flow.
func (s *chatHandlerSender) SendAgentThought(ctx context.Context, conversationID string, thought AgentThought) {
	if s == nil || s.h == nil {
		return
	}
	if thought.Kind == "reasoning" && !s.showReasoning {
		return
	}
	s.h.send(s.chatCtx, ServerMessage{
		Type:           "agent_thought",
		ConversationID: conversationID,
		AgentThought:   &thought,
		PageContext:    s.pageContext,
	})
}

// Compile-time assertion that chatHandlerSender satisfies AgentEventSender.
var _ AgentEventSender = (*chatHandlerSender)(nil)

// Unused import guard — auth is imported for the AgentDeps.Claims field
// type. Keep the import even when other code paths don't reference it
// directly here.
var _ = (*auth.TokenClaims)(nil)
