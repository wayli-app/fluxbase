package ai

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSupervisorPlan_ValidJSON(t *testing.T) {
	content := `{"user_language":"German","route":["sql"],"requires_synthesis":false,"is_investigative":true,"min_tool_calls":2}`
	plan, err := parseSupervisorPlan(content)
	require.NoError(t, err)
	require.NotNil(t, plan)
	assert.Equal(t, "German", plan.UserLanguage)
	assert.Equal(t, []string{"sql"}, plan.Route)
	assert.True(t, plan.IsInvestigative)
	assert.Equal(t, 2, plan.MinToolCalls)
}

func TestParseSupervisorPlan_ToleratesMarkdownFences(t *testing.T) {
	content := "```json\n" + `{"user_language":"English","route":["chat"]}` + "\n```"
	plan, err := parseSupervisorPlan(content)
	require.NoError(t, err)
	assert.Equal(t, "English", plan.UserLanguage)
	assert.Equal(t, []string{"chat"}, plan.Route)
}

func TestParseSupervisorPlan_ToleratesStrayProse(t *testing.T) {
	content := `Here is my plan: {"user_language":"English","route":["chat"]}. Done.`
	plan, err := parseSupervisorPlan(content)
	require.NoError(t, err)
	assert.Equal(t, "English", plan.UserLanguage)
}

func TestParseSupervisorPlan_RejectsUnknownAgent(t *testing.T) {
	content := `{"user_language":"English","route":["nonsense"]}`
	_, err := parseSupervisorPlan(content)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown agent")
}

func TestParseSupervisorPlan_RejectsMissingJSON(t *testing.T) {
	_, err := parseSupervisorPlan("no json here")
	require.Error(t, err)
}

func TestLooksInvestigative_MatchesKeywords(t *testing.T) {
	cases := map[string]bool{
		"how many users":      true,
		"count the orders":    true,
		"show me top 10":      true,
		"list all customers":  true,
		"what is the total":   true,
		"sum of revenue":      true,
		"average order value": true,
		"hello":               false,
		"thanks":              false,
		"what is love":        false, // "what is the" wouldn't match
	}
	for msg, want := range cases {
		assert.Equal(t, want, looksInvestigative(msg), "msg=%q", msg)
	}
}

func TestSupervisorAgent_RunsAndStoresPlan(t *testing.T) {
	provider := newFakeProvider("test")
	provider.QueueResponse(newSimpleResponse(`{"user_language":"English","route":["chat"],"requires_synthesis":false,"is_investigative":false,"min_tool_calls":0}`))

	chatbot := &Chatbot{Name: "test", Model: "gpt-4"}
	deps := &AgentDeps{Chatbot: chatbot, Provider: provider}
	supervisor := NewSupervisorAgent(deps)

	state := NewState()
	state.SetUserMessage("hi")
	err := supervisor.Run(context.Background(), state)
	require.NoError(t, err)

	planVal, ok := state.Get(SupervisorPlanKey)
	require.True(t, ok)
	plan, ok := planVal.(*SupervisorPlan)
	require.True(t, ok)
	assert.Equal(t, "English", plan.UserLanguage)
	assert.Equal(t, []string{"chat"}, plan.Route)
	assert.Equal(t, "English", state.UserLanguage())
}

func TestSupervisorAgent_PageProfileWhitelist_FiltersRoute(t *testing.T) {
	provider := newFakeProvider("test")
	// LLM routes to sql+kb, but page profile only allows sql
	provider.QueueResponse(newSimpleResponse(`{"user_language":"English","route":["sql","kb"],"requires_synthesis":true,"is_investigative":true,"min_tool_calls":1}`))

	chatbot := &Chatbot{Name: "test", Model: "gpt-4"}
	deps := &AgentDeps{
		Chatbot:     chatbot,
		Provider:    provider,
		PageProfile: &PageProfile{Page: "orders", Agents: []string{"sql"}},
	}
	supervisor := NewSupervisorAgent(deps)

	state := NewState()
	state.SetUserMessage("show me the orders")
	state.SetPageProfile(deps.PageProfile)
	err := supervisor.Run(context.Background(), state)
	require.NoError(t, err)

	planVal, _ := state.Get(SupervisorPlanKey)
	plan := planVal.(*SupervisorPlan)
	// kb should have been filtered out
	assert.Equal(t, []string{"sql"}, plan.Route)
}

func TestSupervisorAgent_PageProfileFiltersAll_FallsBackToChat(t *testing.T) {
	provider := newFakeProvider("test")
	// LLM routes to sql, but page profile only allows chat
	provider.QueueResponse(newSimpleResponse(`{"user_language":"English","route":["sql"],"is_investigative":true,"min_tool_calls":1}`))

	chatbot := &Chatbot{Name: "test", Model: "gpt-4"}
	deps := &AgentDeps{
		Chatbot:     chatbot,
		Provider:    provider,
		PageProfile: &PageProfile{Page: "chat-only", Agents: []string{"chat"}},
	}
	supervisor := NewSupervisorAgent(deps)

	state := NewState()
	state.SetUserMessage("query something")
	state.SetPageProfile(deps.PageProfile)
	err := supervisor.Run(context.Background(), state)
	require.NoError(t, err)

	planVal, _ := state.Get(SupervisorPlanKey)
	plan := planVal.(*SupervisorPlan)
	assert.Equal(t, []string{"chat"}, plan.Route)
}

func TestSupervisorAgent_InvestigativeWithoutMinCalls_DefaultsTo1(t *testing.T) {
	provider := newFakeProvider("test")
	provider.QueueResponse(newSimpleResponse(`{"user_language":"English","route":["sql"],"is_investigative":true,"min_tool_calls":0}`))

	chatbot := &Chatbot{Name: "test", Model: "gpt-4"}
	deps := &AgentDeps{Chatbot: chatbot, Provider: provider}
	state := NewState()
	state.SetUserMessage("count users")
	require.NoError(t, NewSupervisorAgent(deps).Run(context.Background(), state))

	planVal, _ := state.Get(SupervisorPlanKey)
	plan := planVal.(*SupervisorPlan)
	assert.Equal(t, 1, plan.MinToolCalls)
}

func TestSupervisorAgent_ProviderError_ReturnsError(t *testing.T) {
	provider := newFakeProvider("test")
	provider.QueueError(assertError("provider down"))

	chatbot := &Chatbot{Name: "test", Model: "gpt-4"}
	deps := &AgentDeps{Chatbot: chatbot, Provider: provider}
	state := NewState()
	state.SetUserMessage("hi")
	err := NewSupervisorAgent(deps).Run(context.Background(), state)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "provider call failed")
}

// assertError is a tiny helper to construct an error inline.
func assertError(msg string) error { return &simpleError{msg: msg} }

type simpleError struct{ msg string }

func (e *simpleError) Error() string { return e.msg }
