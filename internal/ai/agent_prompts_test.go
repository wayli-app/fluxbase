package ai

import (
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
