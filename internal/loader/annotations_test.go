package loader

import (
	"reflect"
	"testing"
)

// =============================================================================
// ParseAnnotations Tests
// =============================================================================
//
// Contract source: annotations.go doc comment.
//   - commentStyles selects which line-comment prefixes to recognize ("//", "--", "/*").
//   - Block comment bodies (lines starting with "*") are always recognized.
//   - Keys are lowercased; values are raw trimmed strings; flag-only → "".
//   - First occurrence of a key wins (duplicates do not overwrite).

func TestParseAnnotations_EmptyInput(t *testing.T) {
	t.Parallel()
	got := ParseAnnotations("", []string{"//"})
	if len(got) != 0 {
		t.Errorf("empty code: expected empty map, got %v", got)
	}
	// Even with no matches the result must be non-nil per the implementation
	// (it is always `make(map...)`).
	if got == nil {
		t.Error("empty code: expected non-nil map")
	}
}

func TestParseAnnotations_NilCommentStyles_StillRecognizesBlockLines(t *testing.T) {
	t.Parallel()
	// Contract: block comment bodies are recognized regardless of commentStyles.
	code := "* @fluxbase:flag\n"
	got := ParseAnnotations(code, nil)
	want := map[string]string{"flag": ""}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("nil commentStyles: got %v, want %v", got, want)
	}
}

func TestParseAnnotations_UnknownCommentStyleIgnored(t *testing.T) {
	t.Parallel()
	// "#" is not a supported style and must be silently ignored (no error path
	// exists in this function).
	code := "# @fluxbase:flag\n"
	got := ParseAnnotations(code, []string{"#"})
	if len(got) != 0 {
		t.Errorf("unknown style #: expected no matches, got %v", got)
	}
}

func TestParseAnnotations_LineStyles(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		code  string
		style string
		key   string
		value string
	}{
		{"slash-slash flag", "// @fluxbase:readonly\n", "//", "readonly", ""},
		{"slash-slash value", "// @fluxbase:table users\n", "//", "table", "users"},
		{"sql dash flag", "-- @fluxbase:readonly\n", "--", "readonly", ""},
		{"sql dash value", "-- @fluxbase:table users\n", "--", "table", "users"},
		{"c-open value", "/* @fluxbase:table users */\n", "/*", "table", "users"},
		{"c-open flag no close", "/* @fluxbase:readonly\n", "/*", "readonly", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ParseAnnotations(tt.code, []string{tt.style})
			val, ok := got[tt.key]
			if !ok {
				t.Fatalf("expected key %q in result, got %v", tt.key, got)
			}
			if val != tt.value {
				t.Errorf("key %q: got value %q, want %q", tt.key, val, tt.value)
			}
		})
	}
}

func TestParseAnnotations_BlockLineAlwaysRecognized(t *testing.T) {
	t.Parallel()
	// Even with a non-matching commentStyles set, the block pattern applies.
	code := "* @fluxbase:table users\n"
	got := ParseAnnotations(code, []string{"//"})
	if val, ok := got["table"]; !ok || val != "users" {
		t.Errorf("block line: got %v, want {table: users}", got)
	}
}

func TestParseAnnotations_FlagOnlyYieldsEmptyValue(t *testing.T) {
	t.Parallel()
	got := ParseAnnotations("// @fluxbase:readonly\n", []string{"//"})
	val, ok := got["readonly"]
	if !ok {
		t.Fatalf("flag-only: expected key %q, got %v", "readonly", got)
	}
	if val != "" {
		t.Errorf("flag-only: expected empty value, got %q", val)
	}
}

func TestParseAnnotations_KeyLowercasing(t *testing.T) {
	t.Parallel()
	// Contract: keys are lowercased.
	got := ParseAnnotations("// @fluxbase:MyKey value\n", []string{"//"})
	if _, ok := got["mykey"]; !ok {
		t.Errorf("expected lowercased key %q, got %v", "mykey", got)
	}
	if _, ok := got["MyKey"]; ok {
		t.Errorf("did not expect original-case key %q in %v", "MyKey", got)
	}
}

func TestParseAnnotations_FirstKeyWins(t *testing.T) {
	t.Parallel()
	// Contract: first occurrence wins; duplicates do not overwrite.
	code := "// @fluxbase:table first\n// @fluxbase:table second\n"
	got := ParseAnnotations(code, []string{"//"})
	if val := got["table"]; val != "first" {
		t.Errorf("first-key-wins: got %q, want %q", val, "first")
	}
}

func TestParseAnnotations_ValueTrimming(t *testing.T) {
	t.Parallel()
	// Leading/trailing whitespace around the value is trimmed.
	code := "// @fluxbase:table    padded value    \n"
	got := ParseAnnotations(code, []string{"//"})
	if val := got["table"]; val != "padded value" {
		t.Errorf("trimming: got %q, want %q", val, "padded value")
	}
}

func TestParseAnnotations_LineWithTrailingCommentClose(t *testing.T) {
	t.Parallel()
	// A `*/` terminator after the value must not bleed into the value.
	code := "/* @fluxbase:table users */\n"
	got := ParseAnnotations(code, []string{"/*"})
	if val := got["table"]; val != "users" {
		t.Errorf("trailing */: got %q, want %q", val, "users")
	}
}

func TestParseAnnotations_NoPrefixYieldsNoMatch(t *testing.T) {
	t.Parallel()
	code := "// @fluxbase table users\n" // missing the colon → no match
	got := ParseAnnotations(code, []string{"//"})
	if len(got) != 0 {
		t.Errorf("missing colon: expected no matches, got %v", got)
	}
}

// =============================================================================
// ParseCommaList Tests
// =============================================================================
//
// Contract source: annotations.go doc comment and observed behavior.
//   - Empty string → nil slice.
//   - Otherwise: split on ",", trim each element, drop empties.
//
// NOTE: the `""` → nil vs `",,"` → empty-non-nil distinction is an observed
// behavior, not documented. Per the testing discipline (test-vs-code) this is
// treated as a pin of current behavior, not a sensible-contract assertion.
// If this distinction proves load-bearing for callers it should be promoted to
// a documented contract; if not, it is a latent inconsistency.

func TestParseCommaList(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   string
		want []string
	}{
		{"empty returns nil", "", nil},
		{"single value", "a", []string{"a"}},
		{"two values", "a,b", []string{"a", "b"}},
		{"drops empty parts", "a,,b", []string{"a", "b"}},
		{"trims whitespace", "  a  ,  b  ", []string{"a", "b"}},
		{"all-commas returns empty non-nil slice", ",,", []string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ParseCommaList(tt.in)
			// Compare with reflect so nil vs empty-non-nil is distinguished.
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseCommaList(%q) = %v (nil=%v), want %v (nil=%v)",
					tt.in, got, got == nil, tt.want, tt.want == nil)
			}
		})
	}
}

// =============================================================================
// ParseRoleList Tests
// =============================================================================
//
// Contract source: annotations.go doc comment.
//   - Delegate to ParseCommaList, then lowercase each element.
//   - No dedup is documented; "Admin,admin" stays as two entries.

func TestParseRoleList(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   string
		want []string
	}{
		{"empty returns nil", "", nil},
		{"lowercases roles", "Admin,User", []string{"admin", "user"}},
		{"no dedup", "Admin,admin", []string{"admin", "admin"}},
		{"trims and lowercases", "  Admin  ,  USER ", []string{"admin", "user"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ParseRoleList(tt.in)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseRoleList(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}
