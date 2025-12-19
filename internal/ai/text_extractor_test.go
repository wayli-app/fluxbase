package ai

import (
	"testing"
)

func TestSanitizeText(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "normal text unchanged",
			input:    "Hello, World!",
			expected: "Hello, World!",
		},
		{
			name:     "removes null bytes",
			input:    "Hello\x00World",
			expected: "HelloWorld",
		},
		{
			name:     "preserves newlines and tabs",
			input:    "Line 1\nLine 2\tTabbed",
			expected: "Line 1\nLine 2\tTabbed",
		},
		{
			name:     "preserves carriage returns",
			input:    "Line 1\r\nLine 2",
			expected: "Line 1\r\nLine 2",
		},
		{
			name:     "removes control characters",
			input:    "Hello\x01\x02\x03World",
			expected: "HelloWorld",
		},
		{
			name:     "removes DEL character",
			input:    "Hello\x7FWorld",
			expected: "HelloWorld",
		},
		{
			name:     "handles binary garbage from PDF",
			input:    "*02\x01\x1D*)\x011\x1C) ¹\x06\x02\x13\x15",
			expected: "*02*)1) ¹",
		},
		{
			name:     "preserves unicode",
			input:    "こんにちは世界",
			expected: "こんにちは世界",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "only null bytes",
			input:    "\x00\x00\x00",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeText(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeText(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
