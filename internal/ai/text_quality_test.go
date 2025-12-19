package ai

import (
	"testing"
)

func TestIsValidTextContent(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected bool
	}{
		{
			name:     "empty string",
			text:     "",
			expected: false,
		},
		{
			name:     "whitespace only",
			text:     "   \n\t  ",
			expected: false,
		},
		{
			name:     "valid english text",
			text:     "This is a valid English sentence with normal text content.",
			expected: true,
		},
		{
			name:     "valid multilingual text",
			text:     "Hello World. Bonjour le monde. Hallo Welt. こんにちは世界",
			expected: true,
		},
		{
			name:     "garbage binary data",
			text:     "*02\x01\x1D*)\x011\x1C) ¹\x06\x02\x13\x15",
			expected: false,
		},
		{
			name:     "mostly control characters",
			text:     "\x00\x01\x02\x03\x04\x05\x06\x07\x08\x0B\x0C\x0E\x0F",
			expected: false,
		},
		{
			name:     "short text (lenient)",
			text:     "OK",
			expected: true,
		},
		{
			name:     "numbers and punctuation",
			text:     "Order #12345: Total $99.99 - Payment received on 2024-01-15",
			expected: true,
		},
		{
			name:     "receipt-like garbage (original issue)",
			text:     "*02\x01\x1D*)\x011\x1C) ¹\x06\x02\x13\x15\u0098 \x05\x1C/0(\x01/-\x1C).\x1C\x1E/$ \u009A",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidTextContent(tt.text)
			if result != tt.expected {
				t.Errorf("IsValidTextContent(%q) = %v, want %v", tt.text, result, tt.expected)
			}
		})
	}
}

func TestTextQualityScore(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		minScore float64
		maxScore float64
	}{
		{
			name:     "empty string",
			text:     "",
			minScore: 0,
			maxScore: 0,
		},
		{
			name:     "high quality text",
			text:     "This is perfectly normal English text.",
			minScore: 0.9,
			maxScore: 1.0,
		},
		{
			name:     "garbage data",
			text:     "\x00\x01\x02\x03\x04\x05",
			minScore: 0,
			maxScore: 0.3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := TextQualityScore(tt.text)
			if score < tt.minScore || score > tt.maxScore {
				t.Errorf("TextQualityScore(%q) = %v, want between %v and %v",
					tt.text, score, tt.minScore, tt.maxScore)
			}
		})
	}
}
