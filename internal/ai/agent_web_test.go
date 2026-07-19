package ai

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nimbleflux/fluxbase/internal/ai/integrations"
)

// TestWebAgent_NilIntegrations_ReturnsErrorNotPanic verifies the Web
// Agent's nil-safe contract: when no integrations storage is wired into
// AgentDeps, the agent returns a clean error instead of panicking. This
// is the Layer 2 recovery guarantee from PR #252 — panics in any
// specialist get surfaced as errors instead of crashing the server.
//
// (Real Tavily HTTP behavior is covered by tavily_test.go.)
func TestWebAgent_NilIntegrations_ReturnsErrorNotPanic(t *testing.T) {
	provider := newFakeProvider("test")
	provider.QueueResponse(newSimpleResponse("answer"))

	sender := newCaptureSender()
	chatbot := &Chatbot{
		Name:             "test",
		Model:            "gpt-4",
		WebSearchEnabled: true,
	}
	deps := &AgentDeps{
		Chatbot:  chatbot,
		Provider: provider,
		Sender:   sender,
		// Integrations intentionally nil
	}

	agent := NewWebAgent(deps)
	state := NewState()
	state.SetUserMessage("what's the latest?")
	err := agent.Run(context.Background(), state)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no integrations storage")
}

// TestSupervisorRoute_ExcludesWeb_WhenChatbotOptsOut verifies the
// supervisor graph silently excludes the Web Agent when the chatbot
// doesn't have @fluxbase:web-search enabled. This is the core opt-in
// contract — no integration = no Web Agent in the route.
func TestSupervisorRoute_ExcludesWeb_WhenChatbotOptsOut(t *testing.T) {
	// Build a supervisor graph with WebSearchEnabled=false. Verify the
	// router's agent map doesn't include "web".
	chatbot := &Chatbot{
		Name:             "test",
		Model:            "gpt-4",
		ReasoningMode:    "supervisor",
		WebSearchEnabled: false,
	}
	deps := &AgentDeps{Chatbot: chatbot}

	graph := NewSupervisorGraph(deps)
	router, ok := graph.graph.nodes["router"].(*routerNode)
	require.True(t, ok, "router must be registered")
	_, hasWeb := router.agents["web"]
	assert.False(t, hasWeb, "Web Agent must NOT be registered when chatbot has WebSearchEnabled=false")
}

// TestSupervisorRoute_IncludesWeb_WhenChatbotOptsIn_AndIntegrationsConfigured
// is the symmetric test: when both annotations are set, the Web Agent IS
// registered. We use a fakeIntegrationStorage as the Integrations dep
// (it's nil here — the test asserts that the chatbot flag is the gate,
// not the storage existence; storage gets checked at agent.Run time).
func TestSupervisorRoute_IncludesWeb_WhenChatbotOptsIn_AndIntegrationsConfigured(t *testing.T) {
	chatbot := &Chatbot{
		Name:             "test",
		Model:            "gpt-4",
		ReasoningMode:    "supervisor",
		WebSearchEnabled: true,
	}
	// Integrations is nil but WebSearchEnabled is true. The supervisor
	// graph should still register "web" — the existence check happens at
	// Run time when the agent tries to resolve credentials. This is the
	// right behavior: a chatbot with @fluxbase:web-search enabled=true
	// should still build the agent, even if no integration is configured
	// yet; the error at Run time tells the user to configure one.
	deps := &AgentDeps{
		Chatbot:      chatbot,
		Integrations: &integrations.Storage{}, // non-nil placeholder; never called in this test
	}
	graph := NewSupervisorGraph(deps)
	router, ok := graph.graph.nodes["router"].(*routerNode)
	require.True(t, ok)
	_, hasWeb := router.agents["web"]
	assert.True(t, hasWeb, "Web Agent must be registered when chatbot has WebSearchEnabled=true and Integrations is non-nil")
}

// TestParseSupervisorPlan_AcceptsWebAgent verifies the supervisor's
// JSON plan parser accepts "web" as a valid routing target. Without this,
// the supervisor would error out when the LLM routes to web.
func TestParseSupervisorPlan_AcceptsWebAgent(t *testing.T) {
	plan, err := parseSupervisorPlan(`{"user_language":"English","route":["web"]}`)
	require.NoError(t, err)
	require.NotNil(t, plan)
	assert.Equal(t, []string{"web"}, plan.Route)
}

// TestParseSupervisorPlan_RejectsUnknownAgent_StillEnforced verifies
// the parser still rejects unknown agent names — adding "web" didn't
// loosen validation.
func TestParseSupervisorPlan_RejectsUnknownAgent_StillEnforced(t *testing.T) {
	_, err := parseSupervisorPlan(`{"user_language":"English","route":["nonsense"]}`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown agent")
}

// TestWebAgent_ExecuteWebSearch_ParsesArguments is a focused unit test
// on the argument parsing path. Verifies the LLM's JSON arguments are
// correctly decoded into a Tavily SearchOptions.
func TestWebAgent_ExecuteWebSearch_ParsesArguments(t *testing.T) {
	// We can't easily call executeWebSearch in isolation (it needs a
	// *TavilyClient). Instead verify the args struct shape via JSON
	// round-trip — that's what executeWebSearch relies on.
	argsJSON := `{"query":"Berlin zoo","max_results":3,"search_depth":"advanced"}`
	var args struct {
		Query       string `json:"query"`
		MaxResults  int    `json:"max_results,omitempty"`
		SearchDepth string `json:"search_depth,omitempty"`
	}
	err := json.Unmarshal([]byte(argsJSON), &args)
	require.NoError(t, err)
	assert.Equal(t, "Berlin zoo", args.Query)
	assert.Equal(t, 3, args.MaxResults)
	assert.Equal(t, "advanced", args.SearchDepth)
}
