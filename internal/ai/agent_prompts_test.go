package ai

import (
	"fmt"
	"strings"
	"testing"
)

func TestBuildSupervisorPrompt_WebSearchEnabled(t *testing.T) {
	t.Run("web-enabled chatbot gets web-available suffix", func(t *testing.T) {
		c := &Chatbot{WebSearchEnabled: true}
		got := BuildSupervisorPrompt(c)

		// Static prompt is always present
		if !strings.Contains(got, "Available specialist agents") {
			t.Error("prompt missing standard agent list")
		}
		// Web-enabled suffix is appended
		if !strings.Contains(got, "web search IS ENABLED") {
			t.Error("web-enabled suffix not appended when WebSearchEnabled=true")
		}
		if !strings.Contains(got, "available RIGHT NOW") {
			t.Error("expected explicit 'available RIGHT NOW' directive")
		}
	})

	t.Run("web-disabled chatbot does NOT get suffix", func(t *testing.T) {
		c := &Chatbot{WebSearchEnabled: false}
		got := BuildSupervisorPrompt(c)

		if strings.Contains(got, "web search IS ENABLED") {
			t.Error("web suffix should NOT be appended when WebSearchEnabled=false")
		}
		// Static prompt still mentions "web" as a conditional agent — that's fine.
		if !strings.Contains(got, "Available specialist agents") {
			t.Error("prompt missing standard agent list")
		}
	})

	t.Run("nil chatbot falls back to static prompt", func(t *testing.T) {
		got := BuildSupervisorPrompt(nil)

		if strings.Contains(got, "web search IS ENABLED") {
			t.Error("nil chatbot should not get web suffix")
		}
		if !strings.Contains(got, "Available specialist agents") {
			t.Error("nil chatbot still gets the static prompt")
		}
	})

	t.Run("prompt is byte-stable per chatbot config (cacheable)", func(t *testing.T) {
		// Same input -> same output, byte for byte
		c1 := &Chatbot{WebSearchEnabled: true}
		c2 := &Chatbot{WebSearchEnabled: true}
		if BuildSupervisorPrompt(c1) != BuildSupervisorPrompt(c2) {
			t.Error("prompt differs between identical chatbot configs — breaks provider caching")
		}

		d1 := &Chatbot{WebSearchEnabled: false}
		d2 := &Chatbot{WebSearchEnabled: false}
		if BuildSupervisorPrompt(d1) != BuildSupervisorPrompt(d2) {
			t.Error("prompt differs between identical chatbot configs — breaks provider caching")
		}
	})
}

func TestBuildSupervisorPrompt_SupervisorWebTriggers(t *testing.T) {
	t.Run("triggers produce chatbot-specific section when present", func(t *testing.T) {
		c := &Chatbot{
			WebSearchEnabled:      true,
			SupervisorWebTriggers: []string{"this weekend", "next weekend", "what's happening"},
		}
		got := BuildSupervisorPrompt(c)

		if !strings.Contains(got, "CHATBOT-SPECIFIC WEB ROUTING TRIGGERS") {
			t.Error("expected chatbot-specific triggers section heading")
		}
		// Each trigger listed, numbered
		for _, phrase := range c.SupervisorWebTriggers {
			want := fmt.Sprintf("\"%s\"", phrase)
			if !strings.Contains(got, want) {
				t.Errorf("prompt missing trigger phrase %q", phrase)
			}
		}
		// MUST-include directive
		if !strings.Contains(got, "you MUST include \"web\" in the route") {
			t.Error("missing MUST-include directive")
		}
	})

	t.Run("triggers absent -> no chatbot-specific section", func(t *testing.T) {
		c := &Chatbot{WebSearchEnabled: true} // no triggers
		got := BuildSupervisorPrompt(c)

		if strings.Contains(got, "CHATBOT-SPECIFIC WEB ROUTING TRIGGERS") {
			t.Error("triggers section should not appear when SupervisorWebTriggers is empty")
		}
		// Still gets the general web-enabled suffix
		if !strings.Contains(got, "web search IS ENABLED") {
			t.Error("expected web-enabled suffix even without triggers")
		}
	})

	t.Run("triggers ignored when web disabled", func(t *testing.T) {
		// Even if triggers are set, web must be enabled for them to surface.
		// Otherwise we'd tell the LLM to route to web when web agent isn't
		// registered in routerAgents.
		c := &Chatbot{
			WebSearchEnabled:      false,
			SupervisorWebTriggers: []string{"this weekend"},
		}
		got := BuildSupervisorPrompt(c)

		if strings.Contains(got, "CHATBOT-SPECIFIC WEB ROUTING TRIGGERS") {
			t.Error("triggers must not surface when WebSearchEnabled=false")
		}
		if strings.Contains(got, "web search IS ENABLED") {
			t.Error("web-enabled suffix must not appear when WebSearchEnabled=false")
		}
	})

	t.Run("prompt is byte-stable with triggers (cacheable)", func(t *testing.T) {
		c1 := &Chatbot{
			WebSearchEnabled:      true,
			SupervisorWebTriggers: []string{"a", "b"},
		}
		c2 := &Chatbot{
			WebSearchEnabled:      true,
			SupervisorWebTriggers: []string{"a", "b"},
		}
		if BuildSupervisorPrompt(c1) != BuildSupervisorPrompt(c2) {
			t.Error("prompt differs between identical chatbot configs — breaks provider caching")
		}
	})
}

func TestParseChatbotConfig_SupervisorWebTriggers(t *testing.T) {
	t.Run("parses comma-separated triggers, optional double-quotes", func(t *testing.T) {
		code := "/**\n" +
			" * @fluxbase:web-search enabled\n" +
			" * @fluxbase:supervisor-web-triggers \"this weekend\",\"next weekend\",\"events in\",\"opening hours\"\n" +
			" */\n" +
			"export default `You are a bot.`;"
		cfg := ParseChatbotConfig(code)

		if len(cfg.SupervisorWebTriggers) != 4 {
			t.Fatalf("expected 4 triggers, got %d: %v", len(cfg.SupervisorWebTriggers), cfg.SupervisorWebTriggers)
		}
		want := []string{"this weekend", "next weekend", "events in", "opening hours"}
		for i, w := range want {
			if cfg.SupervisorWebTriggers[i] != w {
				t.Errorf("trigger[%d] = %q, want %q", i, cfg.SupervisorWebTriggers[i], w)
			}
		}
	})

	t.Run("handles unquoted comma-separated values", func(t *testing.T) {
		code := `-- @fluxbase:supervisor-web-triggers today,currently,this week
`
		cfg := ParseChatbotConfig(code)
		if len(cfg.SupervisorWebTriggers) != 3 {
			t.Fatalf("expected 3 triggers, got %d", len(cfg.SupervisorWebTriggers))
		}
	})

	t.Run("absent annotation -> nil triggers", func(t *testing.T) {
		code := `-- @fluxbase:web-search enabled
`
		cfg := ParseChatbotConfig(code)
		if cfg.SupervisorWebTriggers != nil && len(cfg.SupervisorWebTriggers) != 0 {
			t.Errorf("expected nil/empty triggers, got %v", cfg.SupervisorWebTriggers)
		}
	})

	t.Run("strips trailing */ if author puts annotation on a comment-closing line", func(t *testing.T) {
		// Defensive: JSDoc-style annotations sometimes share a line with */.
		code := ` * @fluxbase:supervisor-web-triggers "this weekend","next weekend" */`
		cfg := ParseChatbotConfig(code)
		// The */ must not appear in any parsed trigger value.
		for _, tr := range cfg.SupervisorWebTriggers {
			if strings.Contains(tr, "*/") {
				t.Errorf("trigger %q contains stray */", tr)
			}
		}
	})
}
