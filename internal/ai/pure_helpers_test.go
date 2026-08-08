package ai

import (
	"errors"
	"strings"
	"testing"
)

// Unit tests for small pure helpers across internal/ai that were at 0.0%
// -short coverage. Each is deterministic with no DB/network/LLM/FS. The
// readFileContent helper takes an injected reader, so it needs no real file.

// =============================================================================
// splitCommaList / splitCommaSeparated (near-duplicate comma splitters)
// =============================================================================
//
// Both: empty -> nil; otherwise split on ",", trim each element, drop empties.
// They are behaviorally identical (independently defined in two files).

func TestSplitCommaList(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   string
		want []string
	}{
		{"empty returns nil", "", nil},
		{"single value", "a", []string{"a"}},
		{"two values", "a,b", []string{"a", "b"}},
		{"trims whitespace", "  a  ,  b  ", []string{"a", "b"}},
		{"drops empty parts", "a,,b,", []string{"a", "b"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := splitCommaList(tt.in)
			if !sliceEqual(got, tt.want) {
				t.Errorf("splitCommaList(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestSplitCommaSeparated(t *testing.T) {
	t.Parallel()
	// Same cases as splitCommaList — confirm the two helpers agree.
	tests := []struct {
		name string
		in   string
		want []string
	}{
		{"empty returns nil", "", nil},
		{"single value", "alpha", []string{"alpha"}},
		{"two values", "alpha,beta", []string{"alpha", "beta"}},
		{"trims whitespace", "  alpha  ,  beta  ", []string{"alpha", "beta"}},
		{"drops empty parts", "alpha,,beta,", []string{"alpha", "beta"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := splitCommaSeparated(tt.in)
			if !sliceEqual(got, tt.want) {
				t.Errorf("splitCommaSeparated(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func sliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	// Distinguish nil from empty (both helpers return nil for empty input).
	if a == nil || b == nil {
		return (a == nil) == (b == nil)
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// =============================================================================
// parseIntParam
// =============================================================================
//
// Contract: parse int, clamp to [min, max]. Non-numeric returns an error (and 0).
// Note: clamping returns min/max with nil error, NOT an error.

func TestParseIntParam(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		in      string
		min     int
		max     int
		want    int
		wantErr bool
	}{
		{"valid in range", "5", 1, 10, 5, false},
		{"below min clamps to min", "0", 1, 10, 1, false},
		{"above max clamps to max", "99", 1, 10, 10, false},
		{"exactly min", "1", 1, 10, 1, false},
		{"exactly max", "10", 1, 10, 10, false},
		{"non-numeric errors", "abc", 1, 10, 0, true},
		{"empty errors", "", 1, 10, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := parseIntParam(tt.in, tt.min, tt.max)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("got %d, want %d", got, tt.want)
			}
		})
	}
}

// =============================================================================
// parsePostgresArray
// =============================================================================
//
// Parses a PG array literal {a,b,...}. "{}"/""/NULL -> nil. Quote toggle is
// honored so commas inside quotes don't split, and the quote characters
// themselves are STRIPPED (the `case '"'` toggles inQuote without appending).

func TestParsePostgresArray(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   string
		want []string
	}{
		{"empty braces return nil", "{}", nil},
		{"empty string returns nil", "", nil},
		{"NULL returns nil", "NULL", nil},
		{"single element", "{a}", []string{"a"}},
		{"two elements", "{a,b}", []string{"a", "b"}},
		{"quoted comma preserved, quotes stripped", `{a,"b,c"}`, []string{`a`, `b,c`}},
		{"drops empty between commas", "{a,,b}", []string{"a", "b"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := parsePostgresArray(tt.in)
			if !sliceEqual(got, tt.want) {
				t.Errorf("parsePostgresArray(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

// =============================================================================
// readFileContent (injected reader — no FS needed)
// =============================================================================
//
// Reads all bytes from an injected reader up to maxSize (capped at 50MB).
// Oversized -> error. The reader's Read must return (0, err) to terminate.

type endlessReader struct{}

func (endlessReader) Read(p []byte) (int, error) {
	// Fill every call; never returns an error, never EOF.
	for i := range p {
		p[i] = 'x'
	}
	return len(p), nil
}

func TestReadFileContent(t *testing.T) {
	t.Parallel()

	t.Run("reads content under max", func(t *testing.T) {
		t.Parallel()
		got, err := readFileContent(strings.NewReader("hello world"), 1024)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if string(got) != "hello world" {
			t.Errorf("got %q, want %q", string(got), "hello world")
		}
	})

	t.Run("oversized returns error", func(t *testing.T) {
		t.Parallel()
		// An endless reader will exceed maxSize=10 and never EOF, so the size
		// guard must fire. maxSize=10 keeps the test fast.
		_, err := readFileContent(endlessReader{}, 10)
		if err == nil {
			t.Error("expected error for oversized input, got nil")
		}
		if !strings.Contains(err.Error(), "too large") {
			t.Errorf("expected 'too large' in error, got %v", err)
		}
	})

	t.Run("empty reader returns empty", func(t *testing.T) {
		t.Parallel()
		got, err := readFileContent(strings.NewReader(""), 1024)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if len(got) != 0 {
			t.Errorf("got %d bytes, want 0", len(got))
		}
	})
}

// =============================================================================
// tailMessages
// =============================================================================
//
// Returns the last n messages; if len(msgs) <= n, returns msgs unchanged.

func TestTailMessages(t *testing.T) {
	t.Parallel()
	mk := func(roles ...string) []Message {
		out := make([]Message, len(roles))
		for i, r := range roles {
			out[i] = Message{Content: r}
		}
		return out
	}
	t.Run("n larger than len returns all", func(t *testing.T) {
		t.Parallel()
		msgs := mk("a", "b")
		got := tailMessages(msgs, 5)
		if len(got) != 2 || got[0].Content != "a" || got[1].Content != "b" {
			t.Errorf("got %+v, want [a b]", got)
		}
	})
	t.Run("returns last n", func(t *testing.T) {
		t.Parallel()
		msgs := mk("a", "b", "c", "d")
		got := tailMessages(msgs, 2)
		if len(got) != 2 || got[0].Content != "c" || got[1].Content != "d" {
			t.Errorf("got %+v, want [c d]", got)
		}
	})
	t.Run("n zero returns empty", func(t *testing.T) {
		t.Parallel()
		msgs := mk("a", "b")
		got := tailMessages(msgs, 0)
		if len(got) != 0 {
			t.Errorf("got %d, want 0", len(got))
		}
	})
	t.Run("empty input", func(t *testing.T) {
		t.Parallel()
		got := tailMessages(nil, 3)
		if len(got) != 0 {
			t.Errorf("got %d, want 0", len(got))
		}
	})
}

// =============================================================================
// parseJSONArgs
// =============================================================================

func TestParseJSONArgs(t *testing.T) {
	t.Parallel()
	t.Run("valid json unmarshals", func(t *testing.T) {
		t.Parallel()
		var out map[string]string
		if err := parseJSONArgs(`{"a":"b"}`, &out); err != nil {
			t.Fatalf("err: %v", err)
		}
		if out["a"] != "b" {
			t.Errorf("got %q, want %q", out["a"], "b")
		}
	})
	t.Run("invalid json errors", func(t *testing.T) {
		t.Parallel()
		var out map[string]string
		if err := parseJSONArgs("{not json", &out); err == nil {
			t.Error("expected error for invalid JSON, got nil")
		}
	})
	// Keep the errors import meaningful for future error-typing cases.
	_ = errors.Is
}

// =============================================================================
// extractTableName
// =============================================================================
//
// Returns the last "."-delimited segment; if no dot, returns input unchanged.

func TestExtractTableName(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"schema-qualified", "public.users", "users"},
		{"two qualifiers", "db.public.users", "users"},
		{"no dot returns input", "users", "users"},
		{"empty returns empty", "", ""},
		{"trailing dot returns empty", "public.", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := extractTableName(tt.in); got != tt.want {
				t.Errorf("extractTableName(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
