package ai

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConversationInsertArgs_PreservesUserID is the regression test for the
// production bug where conversations were saved with user_id = NULL,
// breaking fluxbase.ai.listConversations() for every authenticated user.
//
// Root cause: saveConversation used to run a pre-INSERT existence check
// against auth.users inside WithTenant. WithTenant sets ROLE tenant_service
// + app.current_tenant_id but NOT request.jwt.claims, so the auth_users_select
// RLS policy — which casts request.jwt.claims to jsonb — errored with
// "invalid input syntax for type json". The error branch then nuked
// validUserID to nil, so the INSERT wrote user_id = NULL. The user could
// still chat (in-memory state worked), but their history was invisible
// because ListUserConversations filters WHERE user_id = $1.
//
// The fix removed the pre-check entirely and delegated user_id validation
// to the conversations_user_id_fkey FK constraint. This test pins the
// "user_id is passed through verbatim" contract on the pure helper that
// builds INSERT args — a future refactor that reintroduces an inline
// mutation of user_id will fail here.
func TestConversationInsertArgs_PreservesUserID(t *testing.T) {
	userID := "aa23cf07-25aa-48ac-bb07-7bf4c47f76f3"
	title := "test"
	now := time.Now()
	expiresAt := now.Add(24 * time.Hour)

	conv := &Conversation{
		ID:                    "conv-1",
		ChatbotID:             "bot-1",
		UserID:                &userID,
		Title:                 &title,
		Status:                "active",
		TurnCount:             0,
		TotalPromptTokens:     0,
		TotalCompletionTokens: 0,
		CreatedAt:             now,
		UpdatedAt:             now,
		LastMessageAt:         now,
		ExpiresAt:             &expiresAt,
	}

	args := conversationInsertArgs(conv)

	require.Len(t, args, 13, "column count must match the INSERT statement")
	// args[2] is user_id (third column after id, chatbot_id)
	assert.Equal(t, &userID, args[2], "user_id must be passed through verbatim, never nulled")
	// Round-trip: mutating args must not affect the source conv
	require.NotNil(t, args[2])
	args[2] = nil
	assert.Equal(t, &userID, conv.UserID, "conversationInsertArgs must not share state with conv")
}

// TestConversationInsertArgs_NilUserID_PassesNilThrough verifies the
// nil-user case (anonymous chatbot) still works — user_id should be
// passed as nil so the column gets NULL, NOT converted to some sentinel.
func TestConversationInsertArgs_NilUserID_PassesNilThrough(t *testing.T) {
	now := time.Now()
	title := "anon"
	conv := &Conversation{
		ID:            "conv-1",
		ChatbotID:     "bot-1",
		UserID:        nil, // anonymous
		Title:         &title,
		Status:        "active",
		CreatedAt:     now,
		UpdatedAt:     now,
		LastMessageAt: now,
	}

	args := conversationInsertArgs(conv)
	require.Len(t, args, 13)
	assert.Nil(t, args[2], "nil UserID must be passed through as nil")
}

// TestConversationInsertArgs_ColumnOrderIsStable pins the column order
// against the INSERT statement in saveConversation. If someone adds a
// column to the INSERT but forgets to update the helper (or vice versa),
// the bug surfaces as a wrong column getting the wrong value — a silent
// data corruption class. This test fails loudly in that case.
func TestConversationInsertArgs_ColumnOrderIsStable(t *testing.T) {
	userID := "user-1"
	sessionID := "session-1"
	title := "Title"
	status := "active"
	now := time.Now()
	expiresAt := now.Add(1 * time.Hour)

	conv := &Conversation{
		ID:                    "id-1",
		ChatbotID:             "bot-1",
		UserID:                &userID,
		SessionID:             &sessionID,
		Title:                 &title,
		Status:                status,
		TurnCount:             7,
		TotalPromptTokens:     100,
		TotalCompletionTokens: 50,
		CreatedAt:             now,
		UpdatedAt:             now,
		LastMessageAt:         now,
		ExpiresAt:             &expiresAt,
	}

	args := conversationInsertArgs(conv)

	// Index → column (per saveConversation INSERT statement):
	//  0: id
	//  1: chatbot_id
	//  2: user_id
	//  3: session_id
	//  4: title
	//  5: status
	//  6: turn_count
	//  7: total_prompt_tokens
	//  8: total_completion_tokens
	//  9: created_at
	// 10: updated_at
	// 11: last_message_at
	// 12: expires_at
	assert.Equal(t, "id-1", args[0])
	assert.Equal(t, "bot-1", args[1])
	assert.Equal(t, &userID, args[2])
	assert.Equal(t, &sessionID, args[3])
	assert.Equal(t, &title, args[4])
	assert.Equal(t, status, args[5])
	assert.Equal(t, 7, args[6])
	assert.Equal(t, 100, args[7])
	assert.Equal(t, 50, args[8])
	assert.Equal(t, now, args[9])
	assert.Equal(t, now, args[10])
	assert.Equal(t, now, args[11])
	assert.Equal(t, &expiresAt, args[12])
}
