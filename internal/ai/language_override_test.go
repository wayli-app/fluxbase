package ai

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSupervisorAgent_ResponseLanguage_OverridesDetected is the regression
// test for the "asked in English, got German" bug on a chatbot configured
// with @fluxbase:response-language German.
//
// The supervisor used to ignore chatbot.ResponseLanguage and trust its own
// detected language. When detection was wrong (said "German" for an English
// message), every downstream agent and the synthesizer wrote German. The
// fix overrides plan.UserLanguage with chatbot.ResponseLanguage when set.
func TestSupervisorAgent_ResponseLanguage_OverridesDetected(t *testing.T) {
	provider := newFakeProvider("test")
	// Supervisor "detects" English (wrong for a German-pinned chatbot)
	provider.QueueResponse(newSimpleResponse(`{"user_language":"English","route":["chat"]}`))

	chatbot := &Chatbot{
		Name:             "test",
		Model:            "gpt-4",
		ResponseLanguage: "German", // pinned: should win over detected
	}
	deps := &AgentDeps{Chatbot: chatbot, Provider: provider}
	state := NewState()
	state.SetUserMessage("Hello there")

	require.NoError(t, NewSupervisorAgent(deps).Run(context.Background(), state))

	planVal, _ := state.Get(SupervisorPlanKey)
	plan := planVal.(*SupervisorPlan)
	assert.Equal(t, "German", plan.UserLanguage,
		"configured ResponseLanguage must override supervisor's detected language")
	assert.Equal(t, "German", state.UserLanguage())
}

// TestSupervisorAgent_NoResponseLanguage_UsesDetected verifies that when
// the chatbot is on default behavior (no ResponseLanguage or "auto"), the
// supervisor's detected language is used as before.
func TestSupervisorAgent_NoResponseLanguage_UsesDetected(t *testing.T) {
	for _, tc := range []struct {
		name  string
		value string
	}{
		{"empty", ""},
		{"auto", "auto"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Fresh provider per subtest so each has its own queued response.
			provider := newFakeProvider("test")
			provider.QueueResponse(newSimpleResponse(`{"user_language":"French","route":["chat"]}`))

			chatbot := &Chatbot{Name: "test", Model: "gpt-4", ResponseLanguage: tc.value}
			deps := &AgentDeps{Chatbot: chatbot, Provider: provider}
			state := NewState()
			state.SetUserMessage("Bonjour")

			require.NoError(t, NewSupervisorAgent(deps).Run(context.Background(), state))

			planVal, _ := state.Get(SupervisorPlanKey)
			plan := planVal.(*SupervisorPlan)
			assert.Equal(t, "French", plan.UserLanguage,
				"detected language should be used when ResponseLanguage is %q", tc.value)
		})
	}
}

// TestBuildDynamicContextForAgent_ResponseLanguage_HardDirective verifies
// that the per-agent dynamic context includes the hard language directive
// when ResponseLanguage is set. This is the second half of Bug B's fix —
// even if the supervisor somehow mis-detected, the agents' prompts force
// the configured language.
func TestBuildDynamicContextForAgent_ResponseLanguage_HardDirective(t *testing.T) {
	chatbot := &Chatbot{ResponseLanguage: "German"}
	out := BuildDynamicContextForAgent(chatbot, "user-1", "sql", nil)
	assert.Contains(t, out, "IMPORTANT: Always respond in German")
}

func TestBuildDynamicContextForAgent_NoResponseLanguage_NoDirective(t *testing.T) {
	chatbot := &Chatbot{}
	out := BuildDynamicContextForAgent(chatbot, "user-1", "sql", nil)
	assert.NotContains(t, out, "IMPORTANT: Always respond in")
}

// TestVerifier_NeedsLLMLanguageCheck verifies the gating logic: only
// Latin-script responses with a known expected language AND non-trivial
// length trigger the LLM check. CJK, Arabic etc. are conclusive from
// script alone.
func TestVerifier_NeedsLLMLanguageCheck(t *testing.T) {
	tests := []struct {
		name     string
		userMsg  string
		response string
		language string
		want     bool
	}{
		{"latin both, with language, long enough", "Hello there", "I am doing well, thank you for asking.", "English", true},
		{"latin both, short response, skip", "Hi", "Hello", "English", false},
		{"cjk response, skip", "Hello there", "你好，我很好，谢谢你。", "Chinese", false},
		{"no expected language, skip", "Hello there", "I am doing well, thank you for asking.", "", false},
		{"arabic response, skip", "Hello", "مرحبا، أنا بخير، شكرا لك.", "Arabic", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := needsLLMLanguageCheck(tc.userMsg, tc.response, tc.language)
			assert.Equal(t, tc.want, got)
		})
	}
}

// TestVerifier_LanguageCheck_HappyPath verifies the LLM language call
// parses a yes/no JSON response correctly.
func TestVerifier_LanguageCheck_HappyPath(t *testing.T) {
	provider := newFakeProvider("test")
	provider.QueueResponse(newSimpleResponse(`{"yes": true}`))

	chatbot := &Chatbot{Name: "test", Model: "gpt-4"}
	deps := &AgentDeps{Chatbot: chatbot, Provider: provider}

	verifier := NewVerifierAgent(deps)
	ok, err := verifier.languageCheck(context.Background(), "This is clearly English text.", "English")
	require.NoError(t, err)
	assert.True(t, ok)
}

// TestVerifier_LanguageCheck_RejectsWrongLanguage verifies the LLM call
// returns false when the response is in the wrong language.
func TestVerifier_LanguageCheck_RejectsWrongLanguage(t *testing.T) {
	provider := newFakeProvider("test")
	provider.QueueResponse(newSimpleResponse(`{"yes": false}`))

	chatbot := &Chatbot{Name: "test", Model: "gpt-4"}
	deps := &AgentDeps{Chatbot: chatbot, Provider: provider}

	verifier := NewVerifierAgent(deps)
	ok, err := verifier.languageCheck(context.Background(), "Das ist auf Deutsch geschrieben.", "English")
	require.NoError(t, err)
	assert.False(t, ok)
}
