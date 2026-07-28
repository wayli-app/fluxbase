package ai

import (
	"context"
	"errors"
	"testing"
)

// TestToolAuditLogger_NilSafe verifies that a nil logger (tests / unconfigured
// deployments) never panics — agents call Log unconditionally, so this is a
// load-bearing safety property.
func TestToolAuditLogger_NilSafe(t *testing.T) {
	var l *ToolAuditLogger // nil receiver

	// Must not panic.
	l.Log(context.Background(), &ToolAuditEntry{
		ToolName: "web_search",
		ToolType: "web",
		Agent:    "web",
	})

	// Nil entry must not panic either.
	l.Log(context.Background(), nil)

	// Empty tool name must not panic.
	l.Log(context.Background(), &ToolAuditEntry{})
}

// TestToolAuditLogger_LogToolCallFinish verifies the finish closure captures
// success/error correctly (without a DB, Log itself is a no-op, but the closure
// must still build the entry without panicking).
func TestToolAuditLogger_LogToolCallFinish(t *testing.T) {
	l := &ToolAuditLogger{} // non-nil, nil DB → Log is a no-op

	finish := l.LogToolCall(
		context.Background(),
		"chatbot-1", "conv-1", "msg-1", "user-1", "web",
		"web_search", "web",
		[]byte(`{"query":"events in berlin"}`),
		200,
	)

	// Success path.
	finish("3 results", nil)

	// Error path must also be safe.
	finish2 := l.LogToolCall(
		context.Background(),
		"chatbot-1", "conv-1", "msg-1", "user-1", "web",
		"web_search", "web",
		[]byte(`{"query":"x"}`),
		200,
	)
	finish2("", errors.New("tavily timeout"))
}

// TestTruncateBytes verifies the argument-truncation helper bounds stored
// payloads so a huge tool-call argument can't bloat the audit table.
func TestTruncateBytes(t *testing.T) {
	cases := []struct {
		name  string
		in    []byte
		limit int
		want  int
	}{
		{"under limit", []byte("short"), 100, 5},
		{"over limit", []byte("hello world"), 5, 5},
		{"zero limit (no truncation)", []byte("keep all"), 0, 8},
		{"empty", []byte{}, 100, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := truncateBytes(c.in, c.limit)
			if len(got) != c.want {
				t.Errorf("truncateBytes(%q, %d): len=%d, want %d", string(c.in), c.limit, len(got), c.want)
			}
		})
	}
}

// TestPtrHelpers verifies the inline pointer helpers used to build nullable
// audit fields.
func TestPtrHelpers(t *testing.T) {
	b := boolPtr(true)
	if *b != true {
		t.Error("boolPtr(true) returned wrong value")
	}
	s := strPtr("ok")
	if *s != "ok" {
		t.Error("strPtr returned wrong value")
	}
}
