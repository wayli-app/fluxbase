package ai

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nimbleflux/fluxbase/internal/mcp"
)

// fakeCustomTool is a minimal ToolHandler standing in for a registered
// "custom:foo" tool, so we can exercise ActionAgent.buildToolList without the
// Deno runtime / DB-backed custom-tool loader.
type fakeCustomTool struct {
	name string
}

func (f *fakeCustomTool) Name() string                { return f.name }
func (f *fakeCustomTool) Description() string         { return "fake custom tool" }
func (f *fakeCustomTool) InputSchema() map[string]any { return map[string]any{"type": "object"} }
func (f *fakeCustomTool) RequiredScopes() []string    { return []string{"execute:custom"} }
func (f *fakeCustomTool) Execute(_ context.Context, _ map[string]any, _ *mcp.AuthContext) (*mcp.ToolResult, error) {
	return &mcp.ToolResult{}, nil
}

// TestActionAgent_IncludesCustomTools is the regression guard for the bug where
// custom tools were silently filtered out of the ActionAgent's tool list in
// supervisor mode (MCPToolInfoMap has no custom:* entries). A chatbot that
// enables a custom tool MUST see it exposed to the LLM.
func TestActionAgent_IncludesCustomTools(t *testing.T) {
	registry := mcp.NewToolRegistry()
	registry.Register(&fakeCustomTool{name: "custom:wayli:search_feed_posts"})
	registry.Register(&fakeCustomTool{name: "custom:standalone_tool"})
	// Register a stand-in for the built-in invoke_rpc (an execution-category
	// MCP tool) so the built-in path of GetAvailableTools also resolves. The
	// real handlers are registered by the MCP module init.
	registry.Register(&fakeCustomTool{name: "invoke_rpc"})

	executor := NewMCPToolExecutor(registry)

	chatbot := &Chatbot{
		Name:     "wayli",
		Model:    "gpt-4",
		MCPTools: []string{"custom:wayli:search_feed_posts", "custom:standalone_tool", "invoke_rpc"},
	}
	require.True(t, chatbot.HasMCPTools(), "chatbot should report it has MCP tools")

	deps := &AgentDeps{Chatbot: chatbot, MCPExecutor: executor}
	action := NewActionAgent(deps)

	tools := action.buildToolList(chatbot)
	names := toolNames(tools)

	// Both custom tools must be exposed.
	assert.Contains(t, names, "custom:wayli:search_feed_posts",
		"custom namespaced tool must be exposed to the action agent")
	assert.Contains(t, names, "custom:standalone_tool",
		"custom standalone tool must be exposed to the action agent")
	// The built-in invoke_rpc (an execution-category MCP tool) must still be present.
	assert.Contains(t, names, "invoke_rpc",
		"built-in execution tool must still be exposed")
}

// TestDeriveScopes_IncludesCustomScope verifies DeriveScopes emits the
// execute:custom scope for custom tools, so the AuthContext permits executing
// them (the custom handler's RequiredScopes demands it).
func TestDeriveScopes_IncludesCustomScope(t *testing.T) {
	scopes := DeriveScopes([]string{
		"custom:wayli:search_feed_posts",
		"execute_sql", // built-in → its mapped scopes
	})
	assert.Contains(t, scopes, "execute:custom",
		"custom tools must contribute the execute:custom scope")
}

func toolNames(tools []Tool) []string {
	out := make([]string, 0, len(tools))
	for _, t := range tools {
		out = append(out, t.Function.Name)
	}
	return out
}
