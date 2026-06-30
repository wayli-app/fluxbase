package ai

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestAnthropicProvider() *anthropicProvider {
	return &anthropicProvider{
		name: "test",
		config: AnthropicConfig{
			APIKey:     "test-key",
			Model:      "claude-sonnet-4-5-20250929",
			BaseURL:    "https://api.anthropic.com",
			APIVersion: "2023-06-01",
		},
	}
}

func TestAnthropicProvider_BuildRequest_SystemGoesToTopLevel(t *testing.T) {
	p := newTestAnthropicProvider()

	req := &ChatRequest{
		Messages: []Message{
			{Role: RoleSystem, Content: "static system prompt"},
			{Role: RoleSystem, Content: "dynamic per-turn context"},
			{Role: RoleUser, Content: "hello"},
		},
		MaxTokens: 1024,
	}

	out := p.buildRequest(req)

	// Two system blocks preserved in order
	require.Len(t, out.System, 2)
	assert.Equal(t, "static system prompt", out.System[0].Text)
	assert.Equal(t, "dynamic per-turn context", out.System[1].Text)

	// Messages slice must NOT contain any system messages
	for _, m := range out.Messages {
		assert.NotEqual(t, "system", m.Role)
	}
}

func TestAnthropicProvider_BuildRequest_CacheControlMarksAllButLastSystemBlock(t *testing.T) {
	p := newTestAnthropicProvider()

	req := &ChatRequest{
		Messages: []Message{
			{Role: RoleSystem, Content: "static"},       // should be cached
			{Role: RoleSystem, Content: "dynamic tail"}, // should NOT be cached
		},
	}

	out := p.buildRequest(req)

	require.Len(t, out.System, 2)
	assert.NotNil(t, out.System[0].CacheControl, "first (static) system block must be marked cacheable")
	assert.Equal(t, "ephemeral", out.System[0].CacheControl.Type)
	assert.Nil(t, out.System[1].CacheControl, "last (dynamic) system block must NOT be marked cacheable")
}

func TestAnthropicProvider_BuildRequest_SingleSystemBlockNotMarked(t *testing.T) {
	// Edge case: with only one system block, the loop "0 .. len-1" doesn't
	// execute, so no breakpoint is set. Anthropic would still cache nothing
	// in this case; callers should ensure the static/dynamic split happens
	// upstream (chat_handler_message.go does this as of Step 2).
	p := newTestAnthropicProvider()

	req := &ChatRequest{
		Messages: []Message{
			{Role: RoleSystem, Content: "only one"},
		},
	}

	out := p.buildRequest(req)
	require.Len(t, out.System, 1)
	assert.Nil(t, out.System[0].CacheControl)
}

func TestAnthropicProvider_BuildRequest_ToolsGetCacheBreakpointOnLast(t *testing.T) {
	p := newTestAnthropicProvider()

	req := &ChatRequest{
		Messages: []Message{
			{Role: RoleUser, Content: "hi"},
		},
		Tools: []Tool{
			{Type: "function", Function: ToolFunction{Name: "tool_a", Parameters: map[string]any{"type": "object"}}},
			{Type: "function", Function: ToolFunction{Name: "tool_b", Parameters: map[string]any{"type": "object"}}},
		},
	}

	out := p.buildRequest(req)

	require.Len(t, out.Tools, 2)
	assert.Nil(t, out.Tools[0].CacheControl, "only the last tool should carry the cache breakpoint")
	assert.NotNil(t, out.Tools[1].CacheControl)
	assert.Equal(t, "ephemeral", out.Tools[1].CacheControl.Type)
}

func TestAnthropicProvider_BuildRequest_AssistantToolCallsBecomeToolUseBlocks(t *testing.T) {
	p := newTestAnthropicProvider()

	req := &ChatRequest{
		Messages: []Message{
			{Role: RoleUser, Content: "what's the weather?"},
			{
				Role:    RoleAssistant,
				Content: "Let me check.",
				ToolCalls: []ToolCall{{
					ID:   "toolu_123",
					Type: "function",
					Function: FunctionCall{
						Name:      "get_weather",
						Arguments: `{"city":"Paris"}`,
					},
				}},
			},
			{
				Role:       RoleTool,
				Content:    "sunny, 22C",
				ToolCallID: "toolu_123",
			},
		},
	}

	out := p.buildRequest(req)

	// user → assistant(tool_use) → user(tool_result)
	require.Len(t, out.Messages, 3)

	// Assistant message: text block + tool_use block
	asstMsg := out.Messages[1]
	require.Equal(t, "assistant", asstMsg.Role)
	require.Len(t, asstMsg.Content, 2)
	assert.Equal(t, "text", asstMsg.Content[0].Type)
	assert.Equal(t, "Let me check.", asstMsg.Content[0].Text)
	assert.Equal(t, "tool_use", asstMsg.Content[1].Type)
	assert.Equal(t, "toolu_123", asstMsg.Content[1].ID)
	assert.Equal(t, "get_weather", asstMsg.Content[1].Name)
	assert.Equal(t, "Paris", asstMsg.Content[1].Input["city"])

	// Tool result is a user message with a tool_result content block
	toolMsg := out.Messages[2]
	require.Equal(t, "user", toolMsg.Role)
	require.Len(t, toolMsg.Content, 1)
	assert.Equal(t, "tool_result", toolMsg.Content[0].Type)
	assert.Equal(t, "toolu_123", toolMsg.Content[0].ToolUseID)
	assert.Equal(t, "sunny, 22C", toolMsg.Content[0].Content)
}

func TestAnthropicProvider_BuildRequest_ModelFallback(t *testing.T) {
	p := newTestAnthropicProvider()

	t.Run("uses request model when set", func(t *testing.T) {
		req := &ChatRequest{
			Model:    "claude-opus-4-1-20250805",
			Messages: []Message{{Role: RoleUser, Content: "hi"}},
		}
		out := p.buildRequest(req)
		assert.Equal(t, "claude-opus-4-1-20250805", out.Model)
	})

	t.Run("falls back to provider default", func(t *testing.T) {
		req := &ChatRequest{
			Messages: []Message{{Role: RoleUser, Content: "hi"}},
		}
		out := p.buildRequest(req)
		assert.Equal(t, "claude-sonnet-4-5-20250929", out.Model)
	})
}

func TestAnthropicUsageToStats(t *testing.T) {
	t.Run("all four counters populated", func(t *testing.T) {
		u := &anthropicUsage{
			InputTokens:              100,
			OutputTokens:             50,
			CacheCreationInputTokens: 200,
			CacheReadInputTokens:     300,
		}
		stats := anthropicUsageToStats(u)

		// PromptTokens = 100 + 200 + 300 = 600 (all input billed)
		assert.Equal(t, 600, stats.PromptTokens)
		assert.Equal(t, 50, stats.CompletionTokens)
		assert.Equal(t, 650, stats.TotalTokens)
		assert.Equal(t, 300, stats.CachedTokens, "only cache reads are surfaced as cached")
	})

	t.Run("no caching active", func(t *testing.T) {
		u := &anthropicUsage{
			InputTokens:  100,
			OutputTokens: 50,
		}
		stats := anthropicUsageToStats(u)
		assert.Equal(t, 100, stats.PromptTokens)
		assert.Equal(t, 50, stats.CompletionTokens)
		assert.Equal(t, 0, stats.CachedTokens)
	})

	t.Run("nil input", func(t *testing.T) {
		assert.Nil(t, anthropicUsageToStats(nil))
	})
}

func TestAnthropicProvider_ConvertResponse_ToolUse(t *testing.T) {
	p := newTestAnthropicProvider()

	// Simulated Anthropic response with mixed text + tool_use content blocks.
	anthropicJSON := `{
		"id": "msg_123",
		"model": "claude-sonnet-4-5-20250929",
		"role": "assistant",
		"type": "message",
		"content": [
			{"type": "text", "text": "Let me check the weather."},
			{"type": "tool_use", "id": "toolu_456", "name": "get_weather", "input": {"city": "Tokyo"}}
		],
		"stop_reason": "tool_use",
		"usage": {
			"input_tokens": 50,
			"output_tokens": 30,
			"cache_creation_input_tokens": 0,
			"cache_read_input_tokens": 25
		}
	}`

	var resp anthropicResponse
	require.NoError(t, json.Unmarshal([]byte(anthropicJSON), &resp))

	out := p.convertResponse(&resp)

	require.Len(t, out.Choices, 1)
	c := out.Choices[0]
	assert.Equal(t, "tool_calls", c.FinishReason)
	assert.Contains(t, c.Message.Content, "Let me check the weather.")
	require.Len(t, c.Message.ToolCalls, 1)
	assert.Equal(t, "toolu_456", c.Message.ToolCalls[0].ID)
	assert.Equal(t, "get_weather", c.Message.ToolCalls[0].Function.Name)
	assert.JSONEq(t, `{"city":"Tokyo"}`, c.Message.ToolCalls[0].Function.Arguments)

	require.NotNil(t, out.Usage)
	assert.Equal(t, 75, out.Usage.PromptTokens) // 50 + 0 + 25
	assert.Equal(t, 30, out.Usage.CompletionTokens)
	assert.Equal(t, 25, out.Usage.CachedTokens)
}

func TestMapAnthropicStopReason(t *testing.T) {
	cases := []struct{ in, out string }{
		{"end_turn", "stop"},
		{"tool_use", "tool_calls"},
		{"max_tokens", "length"},
		{"stop_sequence", "stop"},
		{"", ""},
		{"unknown_reason", "unknown_reason"},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.out, mapAnthropicStopReason(tc.in))
	}
}
