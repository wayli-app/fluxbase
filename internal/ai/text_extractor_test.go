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

func TestGetMimeTypeFromExtension(t *testing.T) {
	tests := []struct {
		ext      string
		expected string
	}{
		// Standard cases
		{"pdf", "application/pdf"},
		{"txt", "text/plain"},
		{"md", "text/markdown"},
		{"html", "text/html"},
		{"htm", "text/html"},
		{"csv", "text/csv"},
		{"docx", "application/vnd.openxmlformats-officedocument.wordprocessingml.document"},
		{"xlsx", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"},
		{"rtf", "application/rtf"},
		{"epub", "application/epub+zip"},
		{"json", "application/json"},

		// With leading dot
		{".pdf", "application/pdf"},
		{".txt", "text/plain"},

		// Case insensitivity
		{"PDF", "application/pdf"},
		{".PDF", "application/pdf"},
		{"TXT", "text/plain"},
		{"Html", "text/html"},

		// Unknown extensions
		{"unknown", "application/octet-stream"},
		{"exe", "application/octet-stream"},
		{"", "application/octet-stream"},
	}

	for _, tt := range tests {
		t.Run(tt.ext, func(t *testing.T) {
			result := GetMimeTypeFromExtension(tt.ext)
			if result != tt.expected {
				t.Errorf("GetMimeTypeFromExtension(%q) = %q, want %q", tt.ext, result, tt.expected)
			}
		})
	}
}

func TestGetExtensionFromMimeType(t *testing.T) {
	tests := []struct {
		mimeType string
		expected string
	}{
		{"application/pdf", ".pdf"},
		{"text/plain", ".txt"},
		{"text/markdown", ".md"},
		{"text/html", ".html"},
		{"text/csv", ".csv"},
		{"application/vnd.openxmlformats-officedocument.wordprocessingml.document", ".docx"},
		{"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", ".xlsx"},
		{"application/rtf", ".rtf"},
		{"application/epub+zip", ".epub"},
		{"application/json", ".json"},

		// Unknown MIME types
		{"unknown/type", ""},
		{"application/octet-stream", ""},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.mimeType, func(t *testing.T) {
			result := GetExtensionFromMimeType(tt.mimeType)
			if result != tt.expected {
				t.Errorf("GetExtensionFromMimeType(%q) = %q, want %q", tt.mimeType, result, tt.expected)
			}
		})
	}
}

func TestNewTextExtractor(t *testing.T) {
	t.Run("without OCR", func(t *testing.T) {
		extractor := NewTextExtractor()
		if extractor == nil {
			t.Error("NewTextExtractor() returned nil")
		}
		if extractor.ocrService != nil {
			t.Error("Expected ocrService to be nil")
		}
	})

	t.Run("with OCR service", func(t *testing.T) {
		ocrService := &OCRService{}
		extractor := NewTextExtractorWithOCR(ocrService)
		if extractor == nil {
			t.Error("NewTextExtractorWithOCR() returned nil")
		}
		if extractor.ocrService != ocrService {
			t.Error("Expected ocrService to be set")
		}
	})
}

func TestTextExtractor_SupportedMimeTypes(t *testing.T) {
	extractor := NewTextExtractor()
	mimeTypes := extractor.SupportedMimeTypes()

	expectedMimeTypes := []string{
		"application/pdf",
		"text/plain",
		"text/markdown",
		"text/html",
		"text/csv",
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		"application/rtf",
		"application/epub+zip",
		"application/json",
	}

	for _, expected := range expectedMimeTypes {
		found := false
		for _, mime := range mimeTypes {
			if mime == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected MIME type %q not found in SupportedMimeTypes()", expected)
		}
	}
}

func TestTextExtractor_ExtractFromText(t *testing.T) {
	extractor := NewTextExtractor()

	tests := []struct {
		name     string
		data     []byte
		expected string
	}{
		{
			name:     "plain text",
			data:     []byte("Hello, World!"),
			expected: "Hello, World!",
		},
		{
			name:     "empty data",
			data:     []byte(""),
			expected: "",
		},
		{
			name:     "unicode text",
			data:     []byte("日本語テキスト"),
			expected: "日本語テキスト",
		},
		{
			name:     "multiline text",
			data:     []byte("Line 1\nLine 2\nLine 3"),
			expected: "Line 1\nLine 2\nLine 3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := extractor.ExtractFromText(tt.data)
			if err != nil {
				t.Errorf("ExtractFromText() error = %v", err)
			}
			if result != tt.expected {
				t.Errorf("ExtractFromText() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestTextExtractor_ExtractFromCSV(t *testing.T) {
	extractor := NewTextExtractor()

	t.Run("basic CSV", func(t *testing.T) {
		data := []byte("name,age,city\nAlice,30,NYC\nBob,25,LA")
		result, err := extractor.ExtractFromCSV(data)
		if err != nil {
			t.Errorf("ExtractFromCSV() error = %v", err)
		}
		if result == "" {
			t.Error("ExtractFromCSV() returned empty string")
		}
	})

	t.Run("empty CSV", func(t *testing.T) {
		data := []byte("")
		result, err := extractor.ExtractFromCSV(data)
		if err != nil {
			t.Errorf("ExtractFromCSV() error = %v", err)
		}
		if result != "" {
			t.Errorf("ExtractFromCSV() = %q, want empty string", result)
		}
	})
}

func TestTextExtractor_Extract_UnsupportedMimeType(t *testing.T) {
	extractor := NewTextExtractor()

	data := []byte("some data")
	_, err := extractor.Extract(data, "application/unsupported")
	if err == nil {
		t.Error("Expected error for unsupported MIME type")
	}
}
