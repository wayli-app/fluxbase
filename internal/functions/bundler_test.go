package functions

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBundler(t *testing.T) {
	bundler, err := NewBundler()

	// This test might fail if Deno is not installed
	if err != nil {
		t.Skip("Deno not installed, skipping bundler tests")
	}

	assert.NotNil(t, bundler)
	assert.NotEmpty(t, bundler.denoPath)
}

func TestNeedsBundle(t *testing.T) {
	bundler := &Bundler{}

	tests := []struct {
		name     string
		code     string
		expected bool
	}{
		{
			name:     "simple import",
			code:     `import { z } from "npm:zod"`,
			expected: true,
		},
		{
			name:     "default import",
			code:     `import lodash from "npm:lodash"`,
			expected: true,
		},
		{
			name:     "namespace import",
			code:     `import * as dayjs from "npm:dayjs"`,
			expected: true,
		},
		{
			name:     "side-effect import",
			code:     `import "npm:dotenv/config"`,
			expected: true,
		},
		{
			name:     "type-only import",
			code:     `import type { Schema } from "npm:zod"`,
			expected: true,
		},
		{
			name:     "URL import (esm.sh)",
			code:     `import { z } from "https://esm.sh/zod@3.22.4"`,
			expected: true,
		},
		{
			name: "no imports",
			code: `
export async function handler(req) {
  return { status: 200, body: "Hello" }
}`,
			expected: false,
		},
		{
			name: "commented import",
			code: `
// import { z } from "npm:zod"
export async function handler(req) {}`,
			expected: false,
		},
		{
			name: "import in string",
			code: `
const code = 'import { z } from "npm:zod"'`,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := bundler.NeedsBundle(tt.code)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateImports(t *testing.T) {
	bundler := &Bundler{}

	tests := []struct {
		name          string
		code          string
		expectError   bool
		errorContains string
	}{
		{
			name:        "safe import - zod",
			code:        `import { z } from "npm:zod"`,
			expectError: false,
		},
		{
			name:        "safe import - lodash",
			code:        `import _ from "npm:lodash"`,
			expectError: false,
		},
		{
			name:          "blocked import - child_process",
			code:          `import { exec } from "npm:child_process"`,
			expectError:   true,
			errorContains: "child_process",
		},
		{
			name:          "blocked import - node:child_process",
			code:          `import { exec } from "npm:node:child_process"`,
			expectError:   true,
			errorContains: "node:child_process",
		},
		{
			name:          "blocked import - vm",
			code:          `import vm from "npm:vm"`,
			expectError:   true,
			errorContains: "vm",
		},
		{
			name:          "blocked import - fs",
			code:          `import fs from "npm:fs"`,
			expectError:   true,
			errorContains: "fs",
		},
		{
			name:          "blocked import - process",
			code:          `import process from "npm:process"`,
			expectError:   true,
			errorContains: "process",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := bundler.ValidateImports(tt.code)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestBundle(t *testing.T) {
	bundler, err := NewBundler()
	if err != nil {
		t.Skip("Deno not installed, skipping bundle tests")
	}

	ctx := context.Background()

	t.Run("bundle with npm import", func(t *testing.T) {
		// npm: imports are marked as external by esbuild and resolved at runtime by Deno
		code := `import { z } from "npm:zod@3.22.4"

export async function handler(req) {
  const schema = z.object({ name: z.string() })
  const result = schema.parse({ name: "test" })
  return { status: 200, body: JSON.stringify(result) }
}`

		result, err := bundler.Bundle(ctx, code)

		// This might fail if esbuild has issues
		if err != nil {
			if strings.Contains(err.Error(), "network") ||
				strings.Contains(err.Error(), "timeout") {
				t.Skip("Network unavailable, skipping npm import test")
			}
		}

		require.NoError(t, err)
		assert.True(t, result.IsBundled)
		assert.NotEmpty(t, result.BundledCode)
		assert.Equal(t, code, result.OriginalCode)
		assert.Empty(t, result.Error)

		// Bundled code should contain the import (npm: imports are external, resolved at runtime)
		assert.Contains(t, result.BundledCode, "npm:zod")
	})

	t.Run("no imports - passthrough", func(t *testing.T) {
		code := `export async function handler(req) {
  return { status: 200, body: "Hello World" }
}`

		result, err := bundler.Bundle(ctx, code)

		require.NoError(t, err)
		assert.False(t, result.IsBundled)
		assert.Equal(t, code, result.BundledCode)
		assert.Equal(t, code, result.OriginalCode)
		assert.Empty(t, result.Error)
	})

	t.Run("syntax error", func(t *testing.T) {
		code := `import { z } from "npm:zod"

export async function handler(req) {
  const x = // syntax error here
  return { status: 200 }
}`

		result, err := bundler.Bundle(ctx, code)

		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("blocked import", func(t *testing.T) {
		code := `import { exec } from "npm:child_process"

export async function handler(req) {
  exec("ls")
  return { status: 200 }
}`

		result, err := bundler.Bundle(ctx, code)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "child_process")
		assert.Nil(t, result)
	})

	t.Run("invalid package", func(t *testing.T) {
		// Note: With esbuild bundling, npm: imports are marked as external and validated at runtime
		// by Deno, not at bundle time. This test verifies the bundle succeeds (runtime will fail)
		code := `import { foo } from "npm:this-package-definitely-does-not-exist-12345"

export async function handler(req) {
  return { status: 200 }
}`

		result, err := bundler.Bundle(ctx, code)

		// Bundle should succeed - npm: imports are external and resolved at runtime by Deno
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.True(t, result.IsBundled)
	})

	t.Run("timeout handling", func(t *testing.T) {
		// Create a context with very short timeout
		timeoutCtx, cancel := context.WithTimeout(ctx, 1*time.Millisecond)
		defer cancel()

		code := `import { z } from "npm:zod"

export async function handler(req) {
  return { status: 200 }
}`

		// Sleep to ensure timeout
		time.Sleep(10 * time.Millisecond)

		result, err := bundler.Bundle(timeoutCtx, code)

		// Should fail due to timeout
		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestCleanBundleError(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "removes temp file paths",
			input:    "error: Unexpected token at /tmp/function-abc123.ts:5:10",
			expected: "error: Unexpected token at function.ts:5:10",
		},
		{
			name:     "extracts relevant error lines",
			input:    "error: Module not found \"npm:invalid-pkg\"\n    at file:///tmp/function-xyz.ts:1:1\n\nSome other noise",
			expected: "error: Module not found \"npm:invalid-pkg\"",
		},
		{
			name:     "handles empty error",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cleanBundleError(tt.input)
			assert.Contains(t, result, tt.expected)
		})
	}
}
