// Package ai: tool_audit.go
//
// ToolAuditLogger records non-SQL AI tool calls (web_search, fetch_url,
// invoke_rpc, invoke_function, custom tools) to ai.tool_audit_log.
//
// The pre-existing AuditLogger only covers SQL (query_audit_log), so web/RPC/
// function calls were invisible after the fact — the root cause of "Tavily
// isn't firing, but I can't tell why". This logger makes every tool call
// inspectable. It is best-effort: logging failures never block a turn.

package ai

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/database"
)

// ToolAuditLogger logs non-SQL AI tool calls for observability and debugging.
type ToolAuditLogger struct {
	database.TenantAware
}

// NewToolAuditLogger creates a new tool-call audit logger.
func NewToolAuditLogger(db *database.Connection) *ToolAuditLogger {
	return &ToolAuditLogger{TenantAware: database.TenantAware{DB: db}}
}

// ToolAuditEntry is one tool-call record.
//
// ToolType categorises the call for filtering ("web", "rpc", "function",
// "custom"). Arguments is the raw tool-call argument JSON (truncated upstream
// if very large). ResultSummary is a short human-readable outcome;
// ResultMeta carries structured detail (e.g. {"results":3,"top_url":"..."}).
type ToolAuditEntry struct {
	ChatbotID      *string
	ConversationID *string
	MessageID      *string
	UserID         *string
	ToolName       string
	ToolType       string
	Arguments      []byte // raw JSON
	Agent          string // which specialist invoked it ("web", "action", "react")
	Success        *bool
	ErrorMessage   *string
	ResultSummary  *string
	ResultMeta     []byte // raw JSON
	DurationMs     *int
}

// Log writes one tool-call audit entry. Errors are logged but not returned to
// callers — audit logging must never break an agent turn. A nil logger is a
// safe no-op so agents can call it unconditionally even when auditing isn't
// configured (e.g., in tests).
func (l *ToolAuditLogger) Log(ctx context.Context, entry *ToolAuditEntry) {
	if l == nil || l.DB == nil {
		return
	}
	if entry == nil || entry.ToolName == "" {
		return
	}

	id := uuid.New().String()
	created := time.Now()

	query := `
		INSERT INTO tool_audit_log (
			id, chatbot_id, conversation_id, message_id, user_id,
			tool_name, tool_type, arguments, agent,
			success, error_message, result_summary, result_meta, duration_ms,
			created_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9,
			$10, $11, $12, $13, $14,
			$15
		)
	`
	err := l.WithTenant(ctx, func(tx pgx.Tx) error {
		_, execErr := tx.Exec(
			ctx, query,
			id, entry.ChatbotID, entry.ConversationID, entry.MessageID, entry.UserID,
			entry.ToolName, entry.ToolType, entry.Arguments, entry.Agent,
			entry.Success, entry.ErrorMessage, entry.ResultSummary, entry.ResultMeta, entry.DurationMs,
			created,
		)
		return execErr
	})
	if err != nil {
		log.Warn().Err(err).Str("tool", entry.ToolName).Msg("Failed to log tool call to audit table")
	}
}

// LogToolCall is the convenience wrapper agents use. It measures duration,
// captures success/error, and truncates the result summary to a sane length.
// `fn` is the actual tool execution; the logger wraps it so timing is always
// recorded even on panic-free happy/sad paths.
func (l *ToolAuditLogger) LogToolCall(
	ctx context.Context,
	chatbotID, conversationID, messageID, userID, agent string,
	toolName, toolType string,
	arguments []byte,
	resultSummaryLimit int,
) func(result string, execErr error) {
	// We return a "finish" closure: the agent calls the tool itself, then calls
	// the closure with the outcome. This keeps the logger out of the critical
	// path's control flow while still capturing timing.
	start := time.Now()
	return func(result string, execErr error) {
		if l == nil || l.DB == nil {
			return
		}
		durationMs := int(time.Since(start).Milliseconds())

		entry := &ToolAuditEntry{
			ToolName:   toolName,
			ToolType:   toolType,
			Arguments:  truncateBytes(arguments, 4000),
			Agent:      agent,
			DurationMs: &durationMs,
		}
		if chatbotID != "" {
			entry.ChatbotID = &chatbotID
		}
		if conversationID != "" {
			entry.ConversationID = &conversationID
		}
		if messageID != "" {
			entry.MessageID = &messageID
		}
		if userID != "" {
			entry.UserID = &userID
		}
		ok := execErr == nil
		entry.Success = &ok
		if execErr != nil {
			msg := execErr.Error()
			entry.ErrorMessage = &msg
		}
		if result != "" {
			summary := result
			if resultSummaryLimit > 0 && len(summary) > resultSummaryLimit {
				summary = summary[:resultSummaryLimit] + "..."
			}
			entry.ResultSummary = &summary
		}
		l.Log(ctx, entry)
	}
}

func truncateBytes(b []byte, limit int) []byte {
	if limit > 0 && len(b) > limit {
		return b[:limit]
	}
	return b
}

// boolPtr / strPtr are small helpers for building nullable audit fields inline.
func boolPtr(b bool) *bool    { return &b }
func strPtr(s string) *string { return &s }
