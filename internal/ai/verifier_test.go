package ai

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDominantScript_LatinString(t *testing.T) {
	assert.Equal(t, "latin", dominantScript("Hello world"))
}

func TestDominantScript_CyrillicString(t *testing.T) {
	assert.Equal(t, "cyrillic", dominantScript("Привет мир"))
}

func TestDominantScript_CJKString(t *testing.T) {
	assert.Equal(t, "han", dominantScript("你好世界"))
}

func TestDominantScript_ArabicString(t *testing.T) {
	assert.Equal(t, "arabic", dominantScript("مرحبا بالعالم"))
}

func TestDominantScript_EmptyString(t *testing.T) {
	assert.Equal(t, "", dominantScript(""))
}

func TestDominantScript_NoLetters(t *testing.T) {
	assert.Equal(t, "", dominantScript("123 !@#"))
}

func TestDominantScript_MixedScripts_ReturnsMixed(t *testing.T) {
	// 50/50 latin + cyrillic, should be "mixed"
	assert.Equal(t, "mixed", dominantScript("Hello Привет"))
}

func TestCheckLanguageScriptMatch_LatinBoth_ReturnsTrue(t *testing.T) {
	assert.True(t, checkLanguageScriptMatch("Hello there", "Hi there"))
}

func TestCheckLanguageScriptMatch_LatinUserCyrillicReply_ReturnsFalse(t *testing.T) {
	assert.False(t, checkLanguageScriptMatch("Привет", "Hello there, this is my answer in English"))
}

func TestCheckLanguageScriptMatch_CyrillicBoth_ReturnsTrue(t *testing.T) {
	assert.True(t, checkLanguageScriptMatch("Привет", "Привет, ответ"))
}

func TestCheckLanguageScriptMatch_MixedUserLatinReply_ReturnsTrue(t *testing.T) {
	// Permissive: mixed-script user message accepts any reply
	assert.True(t, checkLanguageScriptMatch("Hello Привет", "Hi there"))
}

func TestVerifier_NonInvestigative_SkipsGroundingCheck(t *testing.T) {
	// Build state for a non-investigative turn
	state := NewState()
	state.SetUserMessage("hello")
	state.SetFinalResponse("hi there")
	state.Set(SupervisorPlanKey, &SupervisorPlan{
		Route:           []string{"chat"},
		IsInvestigative: false,
	})

	// Provide a fake provider that should NOT be called
	provider := newFakeProvider("test")
	chatbot := &Chatbot{Name: "test", Model: "gpt-4"}
	deps := &AgentDeps{Chatbot: chatbot, Provider: provider}

	ctx := ContextWithState(context.Background(), state)
	require.NoError(t, NewVerifierAgent(deps).Run(ctx, state))

	reportVal, ok := state.Get(VerifyReportKey)
	require.True(t, ok)
	report := reportVal.(*VerifyReport)
	assert.True(t, report.GroundingOK, "non-investigative should pass grounding by default")
	assert.Equal(t, 0, provider.Calls(), "LLM should not be called for non-investigative")
}

func TestVerifier_LanguageMismatch_ReturnsLanguageOKFalse(t *testing.T) {
	state := NewState()
	state.SetUserMessage("Привет, как дела?")
	state.SetFinalResponse("Hello! I am doing well, thank you.")
	state.Set(SupervisorPlanKey, &SupervisorPlan{
		Route:           []string{"chat"},
		IsInvestigative: true,
		MinToolCalls:    1,
	})

	provider := newFakeProvider("test")
	chatbot := &Chatbot{Name: "test", Model: "gpt-4"}
	deps := &AgentDeps{Chatbot: chatbot, Provider: provider}
	ctx := ContextWithState(context.Background(), state)
	require.NoError(t, NewVerifierAgent(deps).Run(ctx, state))

	reportVal, _ := state.Get(VerifyReportKey)
	report := reportVal.(*VerifyReport)
	assert.False(t, report.LanguageOK)
	// LLM check should not have run (language already failed)
	assert.Equal(t, 0, provider.Calls())
}

func TestVerifier_InvestigativeWithResults_CallsLLM(t *testing.T) {
	state := NewState()
	state.SetUserMessage("how many users")
	state.SetFinalResponse("There are 42 users.")
	state.Set(SupervisorPlanKey, &SupervisorPlan{
		Route:           []string{"sql"},
		IsInvestigative: true,
		MinToolCalls:    1,
	})
	state.AppendToolResult(QueryResult{
		Query:    "SELECT COUNT(*) FROM users",
		Summary:  "1 row, count=42",
		RowCount: 1,
		Data:     []map[string]any{{"count": 42}},
	})

	provider := newFakeProvider("test")
	provider.QueueResponse(newSimpleResponse(`{"ok":true,"issues":[]}`))
	chatbot := &Chatbot{Name: "test", Model: "gpt-4"}
	deps := &AgentDeps{Chatbot: chatbot, Provider: provider}

	ctx := ContextWithState(context.Background(), state)
	require.NoError(t, NewVerifierAgent(deps).Run(ctx, state))

	assert.Equal(t, 1, provider.Calls(), "LLM should be called for grounding check")
	reportVal, _ := state.Get(VerifyReportKey)
	report := reportVal.(*VerifyReport)
	assert.True(t, report.GroundingOK)
}

func TestVerifier_LLMReportsIssues_PassesThrough(t *testing.T) {
	state := NewState()
	state.SetUserMessage("how many users")
	state.SetFinalResponse("There are 999 users.") // doesn't match tool result
	state.Set(SupervisorPlanKey, &SupervisorPlan{
		Route:           []string{"sql"},
		IsInvestigative: true,
		MinToolCalls:    1,
	})
	state.AppendToolResult(QueryResult{
		Query:    "SELECT COUNT(*) FROM users",
		Summary:  "1 row, count=42",
		RowCount: 1,
	})

	provider := newFakeProvider("test")
	provider.QueueResponse(newSimpleResponse(`{"ok":false,"issues":["count 999 not supported by tool results"]}`))
	chatbot := &Chatbot{Name: "test", Model: "gpt-4"}
	deps := &AgentDeps{Chatbot: chatbot, Provider: provider}

	ctx := ContextWithState(context.Background(), state)
	require.NoError(t, NewVerifierAgent(deps).Run(ctx, state))

	reportVal, _ := state.Get(VerifyReportKey)
	report := reportVal.(*VerifyReport)
	assert.False(t, report.GroundingOK)
	require.Len(t, report.Issues, 1)
}
