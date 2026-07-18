package ai

import (
	"context"

	"github.com/nimbleflux/fluxbase/internal/auth"
	"github.com/nimbleflux/fluxbase/internal/mcp"
)

// AgentDeps is the shared dependency bundle passed to every agent in the
// supervisor graph. It bundles the chatbot being served, the LLM provider,
// and all the side-effecting services agents might need (executor for SQL,
// RAG for KB, MCP for actions, WebSocket for streaming output to client).
//
// One AgentDeps is constructed per turn and shared across all agents in
// that turn. Agents must not retain references to it past their Run() call.
type AgentDeps struct {
	Chatbot *Chatbot

	// Provider is the LLM provider for the chatbot's configured model.
	Provider Provider

	// UserID is the authenticated user (may be empty for anonymous chatbots).
	UserID string

	// Role and Claims carry the RLS context for SQL execution.
	Role   string
	Claims *auth.TokenClaims

	// PageProfile is the resolved per-page profile for this turn (may be nil).
	PageProfile *PageProfile

	// ── Service dependencies (any may be nil if not configured) ──

	// Executor runs SQL. Required for SQL agent.
	SQLExecutor *Executor

	// SchemaBuilder builds the schema description injected into the SQL
	// agent's dynamic context.
	SchemaBuilder *SchemaBuilder

	// RAGService retrieves knowledge-base context. Required for KB agent.
	RAGService *RAGService

	// MCPExecutor dispatches MCP tool calls (query_table, invoke_function,
	// rpc_call, etc.). Required for Action agent.
	MCPExecutor *MCPToolExecutor

	// MCPAuthCtx is the auth context for MCP tool execution.
	MCPAuthCtx *mcp.AuthContext

	// ConversationID is the active conversation (for audit logging,
	// progress events, etc.).
	ConversationID string

	// ── Streaming output to client ──

	// Sender emits ServerMessages to the WebSocket client. nil in tests
	// that don't care about streaming output.
	Sender AgentEventSender

	// CtxCancel cancels the parent turn (e.g., user clicked "stop").
	// Agents should respect ctx.Done() — that's the primary signal.
}

// AgentEventSender is the interface the chat handler implements so agents
// can emit progress, content, and transition events to the WebSocket
// client without depending on the concrete ChatHandler type.
type AgentEventSender interface {
	// SendProgress emits a step/status event.
	SendProgress(ctx context.Context, conversationID, step, message string)
	// SendContent streams a content delta to the client.
	SendContent(ctx context.Context, conversationID, delta string)
	// SendQueryResult emits a structured query_result event so the client
	// can render data tables alongside the chat.
	SendQueryResult(ctx context.Context, conversationID string, result QueryResult)
	// SendAgentTransition emits an agent_transition event when one agent
	// hands off to another (for client-side UI observability).
	SendAgentTransition(ctx context.Context, conversationID string, transition AgentTransition)
}

// AgentTransition is the wire payload for the agent_transition event.
type AgentTransition struct {
	From        string   `json:"from,omitempty"`
	To          string   `json:"to,omitempty"`
	Route       []string `json:"route,omitempty"`
	Reason      string   `json:"reason,omitempty"`
	PageContext string   `json:"page_context,omitempty"`
}
