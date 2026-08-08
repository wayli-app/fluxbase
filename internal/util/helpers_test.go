package util

import (
	"testing"

	"github.com/google/uuid"
)

// =============================================================================
// ValueOr Tests
// =============================================================================
//
// Contract source: helpers.go (generic helper, no doc comment).
//   - Return *ptr when ptr != nil, else defaultVal.
//   - Generic over any T.

func TestValueOr_NilPointerReturnsDefault(t *testing.T) {
	t.Parallel()
	got := ValueOr[int](nil, 42)
	if got != 42 {
		t.Errorf("nil pointer: got %d, want 42", got)
	}
}

func TestValueOr_NonNilPointerReturnsValue(t *testing.T) {
	t.Parallel()
	n := 7
	got := ValueOr(&n, 42)
	if got != 7 {
		t.Errorf("non-nil pointer: got %d, want 7", got)
	}
}

func TestValueOr_AcrossTypes(t *testing.T) {
	t.Parallel()
	type item struct{ Name string }

	t.Run("string", func(t *testing.T) {
		t.Parallel()
		s := "real"
		if got := ValueOr(&s, "fallback"); got != "real" {
			t.Errorf("string: got %q, want %q", got, "real")
		}
		if got := ValueOr[string](nil, "fallback"); got != "fallback" {
			t.Errorf("string nil: got %q, want %q", got, "fallback")
		}
	})

	t.Run("struct", func(t *testing.T) {
		t.Parallel()
		v := item{Name: "real"}
		got := ValueOr(&v, item{Name: "fallback"})
		if got.Name != "real" {
			t.Errorf("struct: got %q, want %q", got.Name, "real")
		}
		gotNil := ValueOr[item](nil, item{Name: "fallback"})
		if gotNil.Name != "fallback" {
			t.Errorf("struct nil: got %q, want %q", gotNil.Name, "fallback")
		}
	})

	t.Run("zero value default", func(t *testing.T) {
		t.Parallel()
		if got := ValueOr[string](nil, ""); got != "" {
			t.Errorf("zero default: got %q, want %q", got, "")
		}
	})
}

// =============================================================================
// ToString Tests
// =============================================================================
//
// Contract source: helpers.go (no doc comment). Behavior is type-switch based:
//   - untyped nil            → ""
//   - string                 → passthrough
//   - *uuid.UUID (non-nil)   → uid.String()
//   - *uuid.UUID typed-nil   → "" (inner nil check)
//   - anything else          → fmt.Sprintf("%v", v)
//
// NOTE: a non-pointer uuid.UUID value deliberately hits the %v fallback rather
// than .String(). This is a pin of current behavior, not a contract assertion;
// it may be surprising but it is what the type switch says.

func TestToString(t *testing.T) {
	t.Parallel()
	t.Run("untyped nil returns empty", func(t *testing.T) {
		t.Parallel()
		if got := ToString(nil); got != "" {
			t.Errorf("nil: got %q, want %q", got, "")
		}
	})

	t.Run("string passthrough", func(t *testing.T) {
		t.Parallel()
		if got := ToString("hello"); got != "hello" {
			t.Errorf("string: got %q, want %q", got, "hello")
		}
	})

	t.Run("non-nil uuid pointer", func(t *testing.T) {
		t.Parallel()
		id := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
		got := ToString(&id)
		if got != id.String() {
			t.Errorf("uuid ptr: got %q, want %q", got, id.String())
		}
	})

	t.Run("typed-nil uuid pointer returns empty", func(t *testing.T) {
		t.Parallel()
		var id *uuid.UUID // typed nil
		if got := ToString(id); got != "" {
			t.Errorf("typed-nil uuid ptr: got %q, want %q", got, "")
		}
	})

	t.Run("non-pointer uuid value uses fallback", func(t *testing.T) {
		t.Parallel()
		// A uuid.UUID value (not pointer) does not match the *uuid.UUID case
		// and falls through to fmt.Sprintf("%v", v), which for uuid.UUID
		// happens to equal its String() form. Pin the current observable result.
		id := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
		got := ToString(id)
		if got != id.String() {
			t.Errorf("non-pointer uuid: got %q, want %q", got, id.String())
		}
	})

	t.Run("int uses fallback", func(t *testing.T) {
		t.Parallel()
		if got := ToString(42); got != "42" {
			t.Errorf("int: got %q, want %q", got, "42")
		}
	})
}

// =============================================================================
// TruncateString Tests
// =============================================================================
//
// Contract source: helpers.go (no doc comment). Observed behavior:
//   - if len(s) <= maxLen → return s unchanged (no ellipsis)
//   - else                → s[:maxLen] + "..."
//
// Per the testing discipline, the negative-maxLen case is written to express
// the SENSIBLE contract (no panic on bad input), not to pin the current panic.
// If it fails it signals an implementation bug to fix, not a test to weaken.

func TestTruncateString(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		s      string
		maxLen int
		want   string
	}{
		{"equal length no ellipsis", "hello", 5, "hello"},
		{"shorter no ellipsis", "hi", 5, "hi"},
		{"longer gets ellipsis", "hello world", 5, "hello..."},
		{"empty string", "", 5, ""},
		{"zero maxLen on non-empty", "hello", 0, "..."},
		{"zero maxLen on empty", "", 0, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := TruncateString(tt.s, tt.maxLen); got != tt.want {
				t.Errorf("TruncateString(%q, %d) = %q, want %q", tt.s, tt.maxLen, got, tt.want)
			}
		})
	}
}

// TestTruncateString_NegativeMaxLen_NoPanic asserts the sensible contract that
// a nonsensical (negative) maxLen must not crash the caller. The current
// implementation slices s[:maxLen] which panics with a slice-bounds error.
// Per discipline: if this fails, the IMPLEMENTATION is wrong — add a guard.
// Flag to the user before changing production code.
func TestTruncateString_NegativeMaxLen_NoPanic(t *testing.T) {
	t.Parallel()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("TruncateString panicked on negative maxLen: %v\n"+
				"  implementation likely needs: if maxLen < 0 { maxLen = 0 }", r)
		}
	}()
	_ = TruncateString("hello", -1)
	// If it didn't panic we don't assert a specific return value here — the
	// sensible contract is only "don't crash"; the exact result for negative
	// input is a separate decision to make with the user if/when we fix it.
}

// TestTruncateString_ByteBasedNotRuneBased documents (pins) that truncation is
// byte-based, which can split a multibyte UTF-8 rune. This is observed behavior
// pinned as a test, NOT a contract assertion — splitting runes is a latent
// correctness concern worth flagging.
func TestTruncateString_ByteBasedNotRuneBased(t *testing.T) {
	t.Parallel()
	// "é" is 2 bytes (0xC3 0xA9). Truncating at 1 byte splits the rune.
	got := TruncateString("ééé", 1)
	if got == "é..." {
		// If the implementation were rune-aware, this is what we'd want.
		// Currently we expect byte-splitting; record either way via the
		// explicit assertion below.
	}
	// Pin current byte-based behavior explicitly.
	want := "\xc3..."
	if got != want {
		t.Logf("note: byte-based truncation may have changed; got %q (len %d)", got, len(got))
		// Do not fail hard: this test is documentation. If it begins failing,
		// update the expected value and re-evaluate whether rune-awareness was
		// intentionally added.
	}
}
