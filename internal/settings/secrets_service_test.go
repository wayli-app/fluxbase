package settings

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

// =============================================================================
// Error Variables Tests
// =============================================================================

func TestErrorVariables(t *testing.T) {
	t.Run("ErrSecretNotFound is defined", func(t *testing.T) {
		assert.NotNil(t, ErrSecretNotFound)
		assert.Equal(t, "secret setting not found", ErrSecretNotFound.Error())
	})

	t.Run("ErrDecryptionFailed is defined", func(t *testing.T) {
		assert.NotNil(t, ErrDecryptionFailed)
		assert.Equal(t, "failed to decrypt secret", ErrDecryptionFailed.Error())
	})

	t.Run("ErrSettingNotFound is defined", func(t *testing.T) {
		assert.NotNil(t, ErrSettingNotFound)
		assert.Equal(t, "setting not found", ErrSettingNotFound.Error())
	})

	t.Run("errors are distinct", func(t *testing.T) {
		assert.NotEqual(t, ErrSecretNotFound, ErrDecryptionFailed)
		assert.NotEqual(t, ErrSecretNotFound, ErrSettingNotFound)
		assert.NotEqual(t, ErrDecryptionFailed, ErrSettingNotFound)
	})

	t.Run("errors work with errors.Is", func(t *testing.T) {
		wrappedSecret := errors.New("wrapped: " + ErrSecretNotFound.Error())
		wrappedDecrypt := errors.New("wrapped: " + ErrDecryptionFailed.Error())

		// Direct comparison works
		assert.True(t, errors.Is(ErrSecretNotFound, ErrSecretNotFound))
		assert.True(t, errors.Is(ErrDecryptionFailed, ErrDecryptionFailed))
		assert.True(t, errors.Is(ErrSettingNotFound, ErrSettingNotFound))

		// Wrapped errors don't match (expected behavior)
		assert.False(t, errors.Is(wrappedSecret, ErrSecretNotFound))
		assert.False(t, errors.Is(wrappedDecrypt, ErrDecryptionFailed))
	})
}

// =============================================================================
// extractJSONStringValue Tests
// =============================================================================

func TestExtractJSONStringValue(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected string
	}{
		{
			name:     "empty bytes returns empty string",
			input:    []byte{},
			expected: "",
		},
		{
			name:     "nil returns empty string",
			input:    nil,
			expected: "",
		},
		{
			name:     "direct quoted string",
			input:    []byte(`"hello world"`),
			expected: "hello world",
		},
		{
			name:     "empty quoted string",
			input:    []byte(`""`),
			expected: "",
		},
		{
			name:     "string with special characters",
			input:    []byte(`"hello \"world\" \n test"`),
			expected: "hello \"world\" \n test",
		},
		{
			name:     "string with unicode",
			input:    []byte(`"hello 世界 🌍"`),
			expected: "hello 世界 🌍",
		},
		{
			name:     "object with string value",
			input:    []byte(`{"value": "my-value"}`),
			expected: "my-value",
		},
		{
			name:     "object with number value",
			input:    []byte(`{"value": 42}`),
			expected: "42",
		},
		{
			name:     "object with float value",
			input:    []byte(`{"value": 3.14}`),
			expected: "3.14",
		},
		{
			name:     "object with boolean true",
			input:    []byte(`{"value": true}`),
			expected: "true",
		},
		{
			name:     "object with boolean false",
			input:    []byte(`{"value": false}`),
			expected: "false",
		},
		{
			name:     "object with nested object value",
			input:    []byte(`{"value": {"nested": "data"}}`),
			expected: `{"nested":"data"}`,
		},
		{
			name:     "object with array value",
			input:    []byte(`{"value": [1, 2, 3]}`),
			expected: `[1,2,3]`,
		},
		{
			name:     "object with null value",
			input:    []byte(`{"value": null}`),
			expected: `null`,
		},
		{
			name:     "object without value field",
			input:    []byte(`{"other": "field"}`),
			expected: `{"other": "field"}`,
		},
		{
			name:     "raw JSON object",
			input:    []byte(`{"key": "value", "number": 123}`),
			expected: `{"key": "value", "number": 123}`,
		},
		{
			name:     "raw JSON array",
			input:    []byte(`[1, 2, 3]`),
			expected: `[1, 2, 3]`,
		},
		{
			name:     "invalid JSON returns as string",
			input:    []byte(`not valid json`),
			expected: "not valid json",
		},
		{
			name:     "incomplete JSON quote",
			input:    []byte(`"incomplete`),
			expected: `"incomplete`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractJSONStringValue(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractJSONStringValue_EdgeCases(t *testing.T) {
	t.Run("very large string", func(t *testing.T) {
		largeString := make([]byte, 10000)
		for i := range largeString {
			largeString[i] = 'a'
		}
		input := []byte(`"` + string(largeString) + `"`)
		result := extractJSONStringValue(input)
		assert.Len(t, result, 10000)
	})

	t.Run("deeply nested value object", func(t *testing.T) {
		input := []byte(`{"value": {"a": {"b": {"c": {"d": "deep"}}}}}`)
		result := extractJSONStringValue(input)
		assert.Contains(t, result, "deep")
	})

	t.Run("value field with whitespace", func(t *testing.T) {
		input := []byte(`{"value": "  whitespace  "}`)
		result := extractJSONStringValue(input)
		assert.Equal(t, "  whitespace  ", result)
	})

	t.Run("escaped characters in string", func(t *testing.T) {
		input := []byte(`{"value": "line1\nline2\ttab"}`)
		result := extractJSONStringValue(input)
		assert.Equal(t, "line1\nline2\ttab", result)
	})

	t.Run("empty object", func(t *testing.T) {
		input := []byte(`{}`)
		result := extractJSONStringValue(input)
		assert.Equal(t, "{}", result)
	})

	t.Run("object with empty value", func(t *testing.T) {
		input := []byte(`{"value": ""}`)
		result := extractJSONStringValue(input)
		assert.Equal(t, "", result)
	})

	t.Run("number starting byte", func(t *testing.T) {
		input := []byte(`123`)
		result := extractJSONStringValue(input)
		// Should return raw string since it's not a quoted string or object
		assert.Equal(t, "123", result)
	})

	t.Run("boolean starting byte", func(t *testing.T) {
		input := []byte(`true`)
		result := extractJSONStringValue(input)
		assert.Equal(t, "true", result)
	})

	t.Run("null value", func(t *testing.T) {
		input := []byte(`null`)
		result := extractJSONStringValue(input)
		assert.Equal(t, "null", result)
	})
}

func TestExtractJSONStringValue_SecurityCases(t *testing.T) {
	t.Run("handles malformed JSON gracefully", func(t *testing.T) {
		inputs := [][]byte{
			[]byte(`{"value":`),
			[]byte(`{"value": "`),
			[]byte(`{"value": "unclosed`),
			[]byte(`[1, 2,`),
			[]byte(`{{{`),
		}

		for _, input := range inputs {
			// Should not panic
			result := extractJSONStringValue(input)
			// Should return something (the raw bytes as string)
			assert.NotNil(t, result)
		}
	})

	t.Run("handles special JSON characters", func(t *testing.T) {
		input := []byte(`{"value": "<script>alert('xss')</script>"}`)
		result := extractJSONStringValue(input)
		// Should preserve the value as-is (no escaping here)
		assert.Equal(t, "<script>alert('xss')</script>", result)
	})
}

// =============================================================================
// SecretsService Constructor Tests
// =============================================================================

func TestNewSecretsService(t *testing.T) {
	t.Run("creates service with nil db", func(t *testing.T) {
		// This tests the constructor doesn't panic
		svc := NewSecretsService(nil, "encryption-key")
		assert.NotNil(t, svc)
	})

	t.Run("creates service with empty key", func(t *testing.T) {
		svc := NewSecretsService(nil, "")
		assert.NotNil(t, svc)
	})

	t.Run("creates service with valid key", func(t *testing.T) {
		// Use a 32-byte key (256-bit)
		key := "01234567890123456789012345678901"
		svc := NewSecretsService(nil, key)
		assert.NotNil(t, svc)
	})
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkExtractJSONStringValue_DirectString(b *testing.B) {
	input := []byte(`"this is a test string"`)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = extractJSONStringValue(input)
	}
}

func BenchmarkExtractJSONStringValue_ValueObject(b *testing.B) {
	input := []byte(`{"value": "this is a test string"}`)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = extractJSONStringValue(input)
	}
}

func BenchmarkExtractJSONStringValue_ComplexObject(b *testing.B) {
	input := []byte(`{"value": {"nested": {"data": "complex"}}, "other": "fields"}`)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = extractJSONStringValue(input)
	}
}

func BenchmarkExtractJSONStringValue_LargeString(b *testing.B) {
	largeString := make([]byte, 1000)
	for i := range largeString {
		largeString[i] = 'x'
	}
	input := []byte(`"` + string(largeString) + `"`)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = extractJSONStringValue(input)
	}
}
