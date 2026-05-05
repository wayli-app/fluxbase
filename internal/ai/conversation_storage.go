package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/database"
)

// ============================================================================
// USER CONVERSATION OPERATIONS
// ============================================================================

// UserConversationSummary represents a conversation for the user API (not admin)
type UserConversationSummary struct {
	ID           string    `json:"id"`
	ChatbotName  string    `json:"chatbot"`
	Namespace    string    `json:"namespace"`
	Title        *string   `json:"title"`
	Preview      string    `json:"preview"`
	MessageCount int       `json:"message_count"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// UserMessageDetail represents a message in user API response
type UserMessageDetail struct {
	ID           string            `json:"id"`
	Role         string            `json:"role"`
	Content      string            `json:"content"`
	Timestamp    time.Time         `json:"timestamp"`
	QueryResults []UserQueryResult `json:"query_results,omitempty"` // Array of query results for assistant messages
	Usage        *UserUsageStats   `json:"usage,omitempty"`
}

// UserQueryResult represents SQL query results for user API
type UserQueryResult struct {
	Query    string                   `json:"query"`
	Summary  string                   `json:"summary"`
	RowCount int                      `json:"row_count"`
	Data     []map[string]interface{} `json:"data,omitempty"`
}

// UserUsageStats represents token usage for user API
type UserUsageStats struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens,omitempty"`
}

// UserConversationDetail represents a full conversation with messages for user API
type UserConversationDetail struct {
	ID          string              `json:"id"`
	ChatbotName string              `json:"chatbot"`
	Namespace   string              `json:"namespace"`
	Title       *string             `json:"title"`
	CreatedAt   time.Time           `json:"created_at"`
	UpdatedAt   time.Time           `json:"updated_at"`
	Messages    []UserMessageDetail `json:"messages"`
}

// ListUserConversationsOptions contains options for listing user conversations
type ListUserConversationsOptions struct {
	UserID      string
	ChatbotName *string
	Namespace   *string
	Limit       int
	Offset      int
}

// ListUserConversationsResult contains the result of listing user conversations
type ListUserConversationsResult struct {
	Conversations []UserConversationSummary `json:"conversations"`
	Total         int                       `json:"total"`
	HasMore       bool                      `json:"has_more"`
}

// ListUserConversations lists conversations for a specific user with pagination
func (s *Storage) ListUserConversations(ctx context.Context, opts ListUserConversationsOptions) (*ListUserConversationsResult, error) {
	tenantID := database.TenantFromContext(ctx)

	var result *ListUserConversationsResult
	err := database.WrapWithTenantAwareRole(ctx, s.db, tenantID, func(tx pgx.Tx) error {
		query := `
		WITH conv_preview AS (
			SELECT DISTINCT ON (m.conversation_id)
				m.conversation_id,
				LEFT(m.content, 50) AS preview
			FROM ai.messages m
			WHERE m.role = 'user'
			ORDER BY m.conversation_id, m.sequence_number ASC
		),
		conv_count AS (
			SELECT conversation_id, COUNT(*) AS message_count
			FROM ai.messages
			GROUP BY conversation_id
		)
		SELECT
			c.id,
			cb.name AS chatbot_name,
			cb.namespace,
			c.title,
			COALESCE(cp.preview, '') AS preview,
			COALESCE(cc.message_count, 0) AS message_count,
			c.created_at,
			c.updated_at
		FROM ai.conversations c
		LEFT JOIN ai.chatbots cb ON cb.id = c.chatbot_id
		LEFT JOIN conv_preview cp ON cp.conversation_id = c.id
		LEFT JOIN conv_count cc ON cc.conversation_id = c.id
		WHERE c.user_id = $1 AND c.status = 'active'
			AND (c.tenant_id = $2 OR ($2 IS NULL AND c.tenant_id IS NULL))
	`

		args := []interface{}{opts.UserID, database.TenantOrNil(tenantID)}
		argIndex := 3

		if opts.ChatbotName != nil {
			query += fmt.Sprintf(" AND cb.name = $%d", argIndex)
			args = append(args, *opts.ChatbotName)
			argIndex++
		}

		if opts.Namespace != nil {
			query += fmt.Sprintf(" AND cb.namespace = $%d", argIndex)
			args = append(args, *opts.Namespace)
			argIndex++
		}

		query += fmt.Sprintf(" ORDER BY c.updated_at DESC LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
		args = append(args, opts.Limit, opts.Offset)

		rows, err := tx.Query(ctx, query, args...)
		if err != nil {
			return fmt.Errorf("failed to list user conversations: %w", err)
		}
		defer rows.Close()

		var conversations []UserConversationSummary
		for rows.Next() {
			var conv UserConversationSummary
			err := rows.Scan(
				&conv.ID,
				&conv.ChatbotName,
				&conv.Namespace,
				&conv.Title,
				&conv.Preview,
				&conv.MessageCount,
				&conv.CreatedAt,
				&conv.UpdatedAt,
			)
			if err != nil {
				log.Error().Err(err).Msg("Failed to scan conversation")
				continue
			}
			conversations = append(conversations, conv)
		}

		countQuery := `
		SELECT COUNT(*)
		FROM ai.conversations c
		LEFT JOIN ai.chatbots cb ON cb.id = c.chatbot_id
		WHERE c.user_id = $1 AND c.status = 'active'
			AND (c.tenant_id = $2 OR ($2 IS NULL AND c.tenant_id IS NULL))
	`
		countArgs := []interface{}{opts.UserID, database.TenantOrNil(tenantID)}
		countArgIndex := 3

		if opts.ChatbotName != nil {
			countQuery += fmt.Sprintf(" AND cb.name = $%d", countArgIndex)
			countArgs = append(countArgs, *opts.ChatbotName)
			countArgIndex++
		}

		if opts.Namespace != nil {
			countQuery += fmt.Sprintf(" AND cb.namespace = $%d", countArgIndex)
			countArgs = append(countArgs, *opts.Namespace)
		}

		var total int
		err = tx.QueryRow(ctx, countQuery, countArgs...).Scan(&total)
		if err != nil {
			log.Warn().Err(err).Msg("Failed to get conversation count")
			total = len(conversations)
		}

		if conversations == nil {
			conversations = []UserConversationSummary{}
		}

		result = &ListUserConversationsResult{
			Conversations: conversations,
			Total:         total,
			HasMore:       opts.Offset+len(conversations) < total,
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// GetUserConversation retrieves a single conversation with messages for a user
func (s *Storage) GetUserConversation(ctx context.Context, userID, conversationID string) (*UserConversationDetail, error) {
	tenantID := database.TenantFromContext(ctx)

	var result *UserConversationDetail
	err := database.WrapWithTenantAwareRole(ctx, s.db, tenantID, func(tx pgx.Tx) error {
		query := `
		SELECT
			c.id,
			cb.name AS chatbot_name,
			cb.namespace,
			c.title,
			c.created_at,
			c.updated_at
		FROM ai.conversations c
		LEFT JOIN ai.chatbots cb ON cb.id = c.chatbot_id
		WHERE c.id = $1 AND c.user_id = $2 AND c.status = 'active'
			AND (c.tenant_id = $3 OR ($3 IS NULL AND c.tenant_id IS NULL))
	`

		var conv UserConversationDetail
		err := tx.QueryRow(ctx, query, conversationID, userID, database.TenantOrNil(tenantID)).Scan(
			&conv.ID,
			&conv.ChatbotName,
			&conv.Namespace,
			&conv.Title,
			&conv.CreatedAt,
			&conv.UpdatedAt,
		)

		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("failed to get conversation: %w", err)
		}

		msgQuery := `
		SELECT
			id,
			role,
			content,
			query_results,
			executed_sql,
			sql_result_summary,
			sql_row_count,
			prompt_tokens,
			completion_tokens,
			created_at
		FROM ai.messages
		WHERE conversation_id = $1
		ORDER BY sequence_number ASC
	`

		rows, err := tx.Query(ctx, msgQuery, conversationID)
		if err != nil {
			return fmt.Errorf("failed to get messages: %w", err)
		}
		defer rows.Close()

		var messages []UserMessageDetail
		for rows.Next() {
			var msg UserMessageDetail
			var queryResultsJSON []byte
			var executedSQL *string
			var sqlSummary *string
			var sqlRowCount *int
			var promptTokens *int
			var completionTokens *int

			err := rows.Scan(
				&msg.ID,
				&msg.Role,
				&msg.Content,
				&queryResultsJSON,
				&executedSQL,
				&sqlSummary,
				&sqlRowCount,
				&promptTokens,
				&completionTokens,
				&msg.Timestamp,
			)
			if err != nil {
				log.Error().Err(err).Msg("Failed to scan message")
				continue
			}

			if queryResultsJSON != nil {
				var queryResults []UserQueryResult
				if err := json.Unmarshal(queryResultsJSON, &queryResults); err != nil {
					log.Warn().Err(err).Msg("Failed to parse query_results JSON")
				} else if len(queryResults) > 0 {
					msg.QueryResults = queryResults
				}
			}

			if msg.QueryResults == nil && (sqlSummary != nil && *sqlSummary != "") {
				legacyResult := UserQueryResult{
					Summary:  *sqlSummary,
					RowCount: 0,
				}
				if executedSQL != nil {
					legacyResult.Query = *executedSQL
				}
				if sqlRowCount != nil {
					legacyResult.RowCount = *sqlRowCount
				}
				msg.QueryResults = []UserQueryResult{legacyResult}
			}

			if promptTokens != nil || completionTokens != nil {
				msg.Usage = &UserUsageStats{}
				if promptTokens != nil {
					msg.Usage.PromptTokens = *promptTokens
				}
				if completionTokens != nil {
					msg.Usage.CompletionTokens = *completionTokens
				}
				msg.Usage.TotalTokens = msg.Usage.PromptTokens + msg.Usage.CompletionTokens
			}

			messages = append(messages, msg)
		}

		if messages == nil {
			messages = []UserMessageDetail{}
		}

		conv.Messages = messages
		result = &conv
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// DeleteUserConversation soft-deletes a conversation owned by the user
func (s *Storage) DeleteUserConversation(ctx context.Context, userID, conversationID string) error {
	tenantID := database.TenantFromContext(ctx)

	return database.WrapWithTenantAwareRole(ctx, s.db, tenantID, func(tx pgx.Tx) error {
		result, err := tx.Exec(ctx, `
			UPDATE ai.conversations
			SET status = 'deleted', updated_at = NOW()
			WHERE id = $1 AND user_id = $2 AND status = 'active'
		`, conversationID, userID)
		if err != nil {
			return fmt.Errorf("failed to delete conversation: %w", err)
		}

		if result.RowsAffected() == 0 {
			return fmt.Errorf("conversation not found")
		}

		log.Info().
			Str("conversation_id", conversationID).
			Str("user_id", userID).
			Msg("Deleted user conversation")

		return nil
	})
}

// UpdateConversationTitle updates the title of a conversation owned by the user
func (s *Storage) UpdateConversationTitle(ctx context.Context, userID, conversationID, title string) error {
	tenantID := database.TenantFromContext(ctx)

	return database.WrapWithTenantAwareRole(ctx, s.db, tenantID, func(tx pgx.Tx) error {
		result, err := tx.Exec(ctx, `
			UPDATE ai.conversations
			SET title = $3, updated_at = NOW()
			WHERE id = $1 AND user_id = $2 AND status = 'active'
		`, conversationID, userID, title)
		if err != nil {
			return fmt.Errorf("failed to update conversation title: %w", err)
		}

		if result.RowsAffected() == 0 {
			return fmt.Errorf("conversation not found")
		}

		log.Info().
			Str("conversation_id", conversationID).
			Str("title", title).
			Msg("Updated conversation title")

		return nil
	})
}

// SetConversationTitle sets the title of a conversation (internal use, no ownership check)
func (s *Storage) SetConversationTitle(ctx context.Context, conversationID, title string) error {
	tenantID := database.TenantFromContext(ctx)

	return database.WrapWithTenantAwareRole(ctx, s.db, tenantID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			UPDATE ai.conversations
			SET title = $2, updated_at = NOW()
			WHERE id = $1 AND title IS NULL
		`, conversationID, title)
		if err != nil {
			return fmt.Errorf("failed to set conversation title: %w", err)
		}

		return nil
	})
}
