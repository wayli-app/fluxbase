package ai

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestChatbotAuthContext_NilChatCtx_DoesNotPanic is the regression test for
// the production crash reported in user logs:
//
//	panic: runtime error: invalid memory address or nil pointer dereference
//	github.com/nimbleflux/fluxbase/internal/ai.ChatbotAuthContext(0x0, ...)
//	    /build/internal/ai/mcp_bridge.go:14 +0x1d4
//
// Root cause: ActionAgent passed nil for chatCtx; ChatbotAuthContext dereffed
// chatCtx.UserID on line 14. The fix added a nil-guard at the top of
// ChatbotAuthContext. This test pins that guard so a future refactor can't
// reintroduce the crash.
func TestChatbotAuthContext_NilChatCtx_DoesNotPanic(t *testing.T) {
	chatbot := &Chatbot{
		ID:                 "cb-test",
		AllowedTables:      []string{"orders"},
		AllowedSchemas:     []string{"public"},
		HTTPAllowedDomains: []string{"example.com"},
	}

	// Must not panic — produce a usable AuthContext instead.
	authCtx := ChatbotAuthContext(nil, chatbot)
	require.NotNil(t, authCtx)
	assert.Equal(t, "chatbot", authCtx.AuthType)
	assert.Nil(t, authCtx.UserID)
	assert.Equal(t, "", authCtx.UserRole)
	// Metadata should still carry chatbot config even with no user context
	assert.Equal(t, "cb-test", authCtx.Metadata["chatbot_id"])
}

// TestChatbotAuthContext_PopulatedChatCtx_PassesUserAndRole verifies the
// happy path still works after the nil-guard was added.
func TestChatbotAuthContext_PopulatedChatCtx_PassesUserAndRole(t *testing.T) {
	userID := "user-123"
	chatCtx := &ChatContext{
		UserID: &userID,
		Role:   "admin",
	}
	chatbot := &Chatbot{ID: "cb-test"}

	authCtx := ChatbotAuthContext(chatCtx, chatbot)
	require.NotNil(t, authCtx)
	require.NotNil(t, authCtx.UserID)
	assert.Equal(t, "user-123", *authCtx.UserID)
	assert.Equal(t, "admin", authCtx.UserRole)
}
