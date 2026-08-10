package ai

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Unit tests for pure helpers across internal/ai that were at 0% or partial
// coverage under -short: the Build* agent-prompt functions, BuildDynamicContextForAgent,
// generateTitle, GenerateSpanID, parseSupervisorPlan, and buildSupervisorDynamicContext.
// All are deterministic and I/O-free (time is stubbed via the currentTimeForPrompt var).
//
// Convention matches pure_helpers_test.go: testify, package ai (white-box),
// t.Parallel() for the deterministic cases.

// =============================================================================
// Build* agent-prompt functions (6 functions, all were 0%)
// =============================================================================
//
// Each returns a static package-level prompt constant and ignores its *Chatbot
// arg (schema/page details are injected dynamically elsewhere). They must not
// dereference the arg, so nil is a valid input.

func TestBuildAgentPrompts_ReturnStaticConstants(t *testing.T) {
	t.Parallel()
	// nil chatbot is intentional — these functions must not dereference it.
	assert.Equal(t, sqlAgentSystemPrompt, BuildSQLAgentPrompt(nil))
	assert.Equal(t, kbAgentSystemPrompt, BuildKBAgentPrompt(nil))
	assert.Equal(t, actionAgentSystemPrompt, BuildActionAgentPrompt(nil))
	assert.Equal(t, chatAgentSystemPrompt, BuildChatAgentPrompt(nil))
	assert.Equal(t, synthesizerSystemPrompt, BuildSynthesizerPrompt(nil))
	assert.Equal(t, webAgentSystemPrompt, BuildWebAgentPrompt(nil))
}

func TestBuildAgentPrompts_IgnoreChatbotArg(t *testing.T) {
	t.Parallel()
	// A populated chatbot must not change the output — the arg exists only for
	// forward compatibility / interface symmetry.
	cb := &Chatbot{Name: "x", Model: "gpt-4"}
	assert.Equal(t, sqlAgentSystemPrompt, BuildSQLAgentPrompt(cb))
	assert.Equal(t, kbAgentSystemPrompt, BuildKBAgentPrompt(cb))
}

func TestBuildAgentPrompts_NonEmpty(t *testing.T) {
	t.Parallel()
	// Guard against an accidental empty-string constant replacing a prompt.
	prompts := map[string]string{
		"sql":         BuildSQLAgentPrompt(nil),
		"kb":          BuildKBAgentPrompt(nil),
		"action":      BuildActionAgentPrompt(nil),
		"chat":        BuildChatAgentPrompt(nil),
		"synthesizer": BuildSynthesizerPrompt(nil),
		"web":         BuildWebAgentPrompt(nil),
	}
	for name, p := range prompts {
		assert.NotEmpty(t, p, "%s prompt must not be empty", name)
	}
}

// =============================================================================
// BuildDynamicContextForAgent (was 87.5%)
// =============================================================================
//
// Contract source: agent_prompts.go:333. Emits user ID + date, a hard language
// directive when ResponseLanguage is pinned, and a page-context focus suffix.
// Time is non-deterministic but stubbable via currentTimeForPrompt.

func TestBuildDynamicContextForAgent(t *testing.T) {
	// These tests stub the package time var; serialize them so they don't race.
	prev := currentTimeForPrompt
	t.Cleanup(func() { currentTimeForPrompt = prev })

	t.Run("includes user id and stubbed time", func(t *testing.T) {
		currentTimeForPrompt = func() string { return "FIXED-TIME" }
		got := BuildDynamicContextForAgent(&Chatbot{}, "user-123", "sql", nil)
		assert.Contains(t, got, "Current user ID: user-123")
		assert.Contains(t, got, "Current date and time: FIXED-TIME")
		// No language directive or page context by default.
		assert.NotContains(t, got, "IMPORTANT")
		assert.NotContains(t, got, "Page context focus")
	})

	t.Run("hard language directive when pinned", func(t *testing.T) {
		currentTimeForPrompt = func() string { return "T" }
		cb := &Chatbot{ResponseLanguage: "German"}
		got := BuildDynamicContextForAgent(cb, "u", "sql", nil)
		assert.Contains(t, got, "IMPORTANT: Always respond in German")
	})

	t.Run("auto language emits no directive", func(t *testing.T) {
		currentTimeForPrompt = func() string { return "T" }
		for _, lang := range []string{"", "auto"} {
			cb := &Chatbot{ResponseLanguage: lang}
			assert.NotContains(t, BuildDynamicContextForAgent(cb, "u", "sql", nil), "IMPORTANT",
				"ResponseLanguage=%q must not pin language", lang)
		}
	})

	t.Run("page profile suffix appended", func(t *testing.T) {
		currentTimeForPrompt = func() string { return "T" }
		profile := &PageProfile{Suffix: "Focus on orders table."}
		got := BuildDynamicContextForAgent(&Chatbot{}, "u", "sql", profile)
		assert.Contains(t, got, "Page context focus:")
		assert.Contains(t, got, "Focus on orders table.")
	})

	t.Run("nil page profile is safe", func(t *testing.T) {
		currentTimeForPrompt = func() string { return "T" }
		// Must not panic; output omits the page-context section.
		got := BuildDynamicContextForAgent(&Chatbot{}, "u", "sql", nil)
		assert.NotContains(t, got, "Page context focus")
	})

	t.Run("empty page profile suffix omitted", func(t *testing.T) {
		currentTimeForPrompt = func() string { return "T" }
		profile := &PageProfile{Suffix: ""}
		got := BuildDynamicContextForAgent(&Chatbot{}, "u", "sql", profile)
		assert.NotContains(t, got, "Page context focus")
	})
}

// =============================================================================
// generateTitle (was 0%)
// =============================================================================
//
// Contract source: conversation.go:564. Empty -> "New conversation";
// <=50 chars -> as-is; >50 chars -> truncate to 50, breaking on the last space
// if it's after position 30, else hard-cut at 50; append "...".

func TestGenerateTitle(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{"empty returns default", "", "New conversation"},
		{"whitespace only returns default", "   \n\t  ", "New conversation"},
		{"short content returned as-is", "Hello world", "Hello world"},
		{"exactly 50 chars", strings.Repeat("a", 50), strings.Repeat("a", 50)},
		{"under 50 with surrounding whitespace trimmed", "  hi there  ", "hi there"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, generateTitle(tt.content))
		})
	}

	// Long-content cases: verify truncation behavior with word boundaries.
	t.Run("long content with space after 30 breaks on word boundary", func(t *testing.T) {
		t.Parallel()
		// 50 chars: "word " * 10 = 50 chars; last space at index 49 (>30) -> cut there.
		content := strings.Repeat("word ", 11) // > 50 chars
		got := generateTitle(content)
		assert.True(t, strings.HasSuffix(got, "..."))
		assert.LessOrEqual(t, len(got), 53, "truncated title + '...' should be short")
		assert.NotContains(t, got[len(got)-3:], " ") // no trailing space before ...
	})

	t.Run("long content with no spaces hard-cuts at 50", func(t *testing.T) {
		t.Parallel()
		content := strings.Repeat("x", 80) // no spaces -> lastSpace = -1 (not > 30)
		got := generateTitle(content)
		assert.Equal(t, strings.Repeat("x", 50)+"...", got)
	})

	t.Run("long content with space before 30 hard-cuts at 50", func(t *testing.T) {
		t.Parallel()
		// One space at index 5, then no more spaces in the first 50 chars.
		content := "abcde f" + strings.Repeat("g", 60)
		got := generateTitle(content)
		// lastSpace in first 50 = 5, not > 30, so hard-cut at 50.
		assert.Equal(t, content[:50]+"...", got)
	})
}

// =============================================================================
// GenerateSpanID (was 0%)
// =============================================================================
//
// Contract source: chatbot.go:1055. Returns "span_" + hex(uuid). Non-deterministic
// value but deterministic prefix; two calls produce distinct IDs.

func TestGenerateSpanID(t *testing.T) {
	t.Parallel()
	gen := &DefaultTraceIDGenerator{}

	t.Run("has span_ prefix", func(t *testing.T) {
		t.Parallel()
		id := gen.GenerateSpanID()
		assert.True(t, strings.HasPrefix(id, "span_"), "span ID must start with 'span_': %q", id)
	})

	t.Run("non-empty body after prefix", func(t *testing.T) {
		t.Parallel()
		id := gen.GenerateSpanID()
		assert.Greater(t, len(id), len("span_"), "span ID must have a non-empty body")
	})

	t.Run("two calls produce distinct IDs", func(t *testing.T) {
		t.Parallel()
		a := gen.GenerateSpanID()
		b := gen.GenerateSpanID()
		assert.NotEqual(t, a, b, "successive span IDs must differ")
	})

	t.Run("trace and span IDs differ", func(t *testing.T) {
		t.Parallel()
		trace := gen.GenerateTraceID()
		span := gen.GenerateSpanID()
		assert.NotEqual(t, trace, span)
		assert.True(t, strings.HasPrefix(trace, "trace_"))
		assert.True(t, strings.HasPrefix(span, "span_"))
	})
}

// =============================================================================
// parseSupervisorPlan (was 90%, push to 100%)
// =============================================================================
//
// Contract source: agent_supervisor.go:188. Extracts the first {...} object,
// unmarshals to SupervisorPlan, and validates each route entry is a known agent.
// Errors: no JSON object, invalid JSON, unknown agent in route.

func TestParseSupervisorPlan(t *testing.T) {
	t.Parallel()

	t.Run("valid plan with known agents", func(t *testing.T) {
		t.Parallel()
		plan, err := parseSupervisorPlan(`{"route":["sql","kb"],"reasoning":"both"}`)
		require.NoError(t, err)
		require.NotNil(t, plan)
		assert.Equal(t, []string{"sql", "kb"}, plan.Route)
	})

	t.Run("tolerates surrounding prose", func(t *testing.T) {
		t.Parallel()
		plan, err := parseSupervisorPlan(`Here is the plan: {"route":["chat"]} hope it helps`)
		require.NoError(t, err)
		assert.Equal(t, []string{"chat"}, plan.Route)
	})

	t.Run("empty content errors", func(t *testing.T) {
		t.Parallel()
		_, err := parseSupervisorPlan("")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no JSON object")
	})

	t.Run("no object in content errors", func(t *testing.T) {
		t.Parallel()
		_, err := parseSupervisorPlan("just prose, no braces")
		require.Error(t, err)
	})

	t.Run("invalid JSON errors", func(t *testing.T) {
		t.Parallel()
		_, err := parseSupervisorPlan(`{not valid}`)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid JSON")
	})

	t.Run("unknown agent in route errors", func(t *testing.T) {
		t.Parallel()
		_, err := parseSupervisorPlan(`{"route":["sql","bogus"]}`)
		require.Error(t, err)
		assert.Contains(t, err.Error(), `unknown agent in route: "bogus"`)
	})

	t.Run("all valid agents accepted", func(t *testing.T) {
		t.Parallel()
		plan, err := parseSupervisorPlan(`{"route":["sql","kb","action","chat","web"]}`)
		require.NoError(t, err)
		assert.Len(t, plan.Route, 5)
	})
}

// =============================================================================
// buildSupervisorDynamicContext (was 95%, push to 100%)
// =============================================================================
//
// Contract source: agent_supervisor.go:226. Emits date, page context, and an
// excluded-agents list when a page profile whitelists a subset of agents.
// Time stubbed via currentTimeForPrompt.

func TestBuildSupervisorDynamicContext(t *testing.T) {
	prev := currentTimeForPrompt
	t.Cleanup(func() { currentTimeForPrompt = prev })

	t.Run("no page profile omits page context", func(t *testing.T) {
		currentTimeForPrompt = func() string { return "T" }
		got := buildSupervisorDynamicContext(&Chatbot{}, nil)
		assert.Contains(t, got, "Current date and time: T")
		assert.NotContains(t, got, "Page Context")
	})

	t.Run("page profile with full agent list lists no exclusions", func(t *testing.T) {
		currentTimeForPrompt = func() string { return "T" }
		profile := &PageProfile{
			Page:   "dashboard",
			Agents: []string{"sql", "kb", "action", "chat", "web"},
		}
		got := buildSupervisorDynamicContext(&Chatbot{}, profile)
		assert.Contains(t, got, "User is currently on page: dashboard")
		assert.Contains(t, got, "Agents available on this page: sql, kb, action, chat, web")
		assert.NotContains(t, got, "Agents NOT available")
	})

	t.Run("partial agent list lists excluded agents", func(t *testing.T) {
		currentTimeForPrompt = func() string { return "T" }
		profile := &PageProfile{
			Page:   "reports",
			Agents: []string{"sql"}, // excludes kb, action, chat, web
		}
		got := buildSupervisorDynamicContext(&Chatbot{}, profile)
		assert.Contains(t, got, "Agents NOT available on this page: kb, action, chat, web")
		assert.Contains(t, got, "do NOT include these in the route")
	})

	t.Run("suffix focus instruction included", func(t *testing.T) {
		currentTimeForPrompt = func() string { return "T" }
		profile := &PageProfile{Page: "p", Suffix: "Only answer about revenue."}
		got := buildSupervisorDynamicContext(&Chatbot{}, profile)
		assert.Contains(t, got, "Focus instruction: Only answer about revenue.")
	})
}
