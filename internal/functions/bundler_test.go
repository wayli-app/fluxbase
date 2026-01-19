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

		// This might fail if Deno config is not set up or network is unavailable
		if err != nil {
			if strings.Contains(err.Error(), "network") ||
				strings.Contains(err.Error(), "timeout") ||
				strings.Contains(err.Error(), "node_modules") ||
				strings.Contains(err.Error(), "deno.json") ||
				strings.Contains(err.Error(), "deno install") {
				t.Skip("Deno configuration or network unavailable, skipping npm import test")
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
			name:     "empty error message",
			input:    "",
			expected: "",
		},
		{
			name:     "removes temp file paths",
			input:    "error: Unexpected token at /tmp/function-abc123.ts:5:10",
			expected: "error: Unexpected token at function.ts:5:10",
		},
		{
			name:     "removes bundle directory paths",
			input:    "error: Failed to read /tmp/function-bundle-xyz789/module.ts",
			expected: "error: Failed to read module.ts",
		},
		{
			name:     "removes both temp file and bundle paths",
			input:    "error: Import failed from /tmp/function-abc123.ts in /tmp/function-bundle-xyz789/",
			expected: "error: Import failed from function.ts in",
		},
		{
			name:     "extracts module not found errors",
			input:    "Module not found: cannot find 'react'\n    at file:///tmp/function-xyz.ts:1:1\n\nSome other noise",
			expected: "Module not found: cannot find 'react'",
		},
		{
			name:     "extracts expected keyword errors",
			input:    "Some noise\nExpected semicolon at line 42\nMore noise",
			expected: "Expected semicolon at line 42",
		},
		{
			name:     "extracts unexpected token errors",
			input:    "Build started\nUnexpected token '}' at line 15\nBuild failed",
			expected: "Unexpected token '}' at line 15",
		},
		{
			name: "extracts multiple relevant errors",
			input: `Some noise line
error: Syntax error on line 10
More noise
Module not found: 'missing-module'
Even more noise`,
			expected: "error: Syntax error on line 10\nModule not found: 'missing-module'",
		},
		{
			name: "filters empty lines",
			input: `

error: First error

error: Second error


`,
			expected: "error: First error\nerror: Second error",
		},
		{
			name:     "returns full message when no keywords found",
			input:    "Some generic message without keywords",
			expected: "Some generic message without keywords",
		},
		{
			name: "handles complex real-world error",
			input: `Build process started
Processing /tmp/function-abc123.ts
error: Cannot find module 'specifier:@types/node'
  at file:///tmp/function-bundle-xyz789/deps.ts:1:23
Build failed
error: Expected identifier but got '}'
  at /tmp/function-abc123.ts:42:5
Process exited with code 1`,
			expected: "error: Cannot find module 'specifier:@types/node'\nerror: Expected identifier but got '}'",
		},
		{
			name:     "replaces multiple temp file paths",
			input:    "/tmp/function-aaa111.ts imported by /tmp/function-bbb222.ts",
			expected: "function.ts imported by function.ts",
		},
		{
			name: "handles all error keywords",
			input: `error: Syntax error
Module not found: 'foo'
Expected token
Unexpected end of file`,
			expected: "error: Syntax error\nModule not found: 'foo'\nExpected token\nUnexpected end of file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cleanBundleError(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCleanBundleError_PathReplacementRegex(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		contains    string
		notContains string
	}{
		{
			name:        "function temp file replaced",
			input:       "error: Failed in /tmp/function-xyz123ABC.ts",
			contains:    "function.ts",
			notContains: "/tmp/function-",
		},
		{
			name:        "bundle directory removed",
			input:       "error: /tmp/function-bundle-abc123XYZ/file.ts not found",
			contains:    "file.ts",
			notContains: "function-bundle-",
		},
		{
			name:        "alphanumeric IDs handled",
			input:       "error: /tmp/function-a1b2c3d4e5.ts and /tmp/function-bundle-9z8y7x6w/",
			contains:    "function.ts",
			notContains: "a1b2c3d4e5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cleanBundleError(tt.input)
			if tt.contains != "" {
				assert.Contains(t, result, tt.contains)
			}
			if tt.notContains != "" {
				assert.NotContains(t, result, tt.notContains)
			}
		})
	}
}

func TestExtractExportNames(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected []string
	}{
		{
			name:     "empty code",
			code:     "",
			expected: nil,
		},
		{
			name:     "no exports",
			code:     "const foo = 42;\nconsole.log(foo);",
			expected: nil,
		},
		{
			name:     "single export",
			code:     "export { handler }",
			expected: []string{"handler"},
		},
		{
			name:     "multiple exports",
			code:     "export { foo, bar, baz }",
			expected: []string{"foo", "bar", "baz"},
		},
		{
			name:     "export with whitespace",
			code:     "export {  foo  ,  bar  ,  baz  }",
			expected: []string{"foo", "bar", "baz"},
		},
		{
			name:     "export with newlines",
			code:     "export {\n  foo,\n  bar,\n  baz\n}",
			expected: []string{"foo", "bar", "baz"},
		},
		{
			name:     "export with tabs",
			code:     "export {\n\tfoo,\n\tbar,\n\tbaz\n}",
			expected: []string{"foo", "bar", "baz"},
		},
		{
			name:     "export with alias - single",
			code:     "export { handler as default }",
			expected: []string{"handler"},
		},
		{
			name:     "export with alias - multiple",
			code:     "export { foo as bar, baz as qux }",
			expected: []string{"foo", "baz"},
		},
		{
			name:     "export with mixed aliases",
			code:     "export { foo, bar as baz, qux }",
			expected: []string{"foo", "bar", "qux"},
		},
		{
			name:     "export with trailing comma",
			code:     "export { foo, bar, }",
			expected: []string{"foo", "bar"},
		},
		{
			name: "export in complex code",
			code: `import { something } from 'lib';
const foo = 1;
function bar() {}
export { foo, bar }
const unused = 2;`,
			expected: []string{"foo", "bar"},
		},
		{
			name:     "export with extra whitespace around braces",
			code:     "export   {   foo   }",
			expected: []string{"foo"},
		},
		{
			name:     "camelCase and snake_case names",
			code:     "export { myHandler, another_handler, HTTPServer }",
			expected: []string{"myHandler", "another_handler", "HTTPServer"},
		},
		{
			name: "multiline with aliases and whitespace",
			code: `export {
  handler as default,
  helper as utilityFunction,
  validator
}`,
			expected: []string{"handler", "helper", "validator"},
		},
		{
			name:     "single letter exports",
			code:     "export { a, b, c }",
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "underscore prefixed exports",
			code:     "export { _private, __internal }",
			expected: []string{"_private", "__internal"},
		},
		{
			name:     "dollar sign in names",
			code:     "export { $element, jQuery$ }",
			expected: []string{"$element", "jQuery$"},
		},
		{
			name:     "empty export braces",
			code:     "export { }",
			expected: nil,
		},
		{
			name:     "export with only whitespace",
			code:     "export {   \n\t   }",
			expected: nil,
		},
		{
			name:     "export with only commas",
			code:     "export { , , , }",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractExportNames(tt.code)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractExportNames_OnlyFirstExportStatement(t *testing.T) {
	// The function uses FindStringSubmatch which only finds the first match
	code := `export { foo }
export { bar }
export { baz }`

	result := extractExportNames(code)

	// Should only extract from the first export statement
	assert.Equal(t, []string{"foo"}, result)
	assert.NotContains(t, result, "bar")
	assert.NotContains(t, result, "baz")
}

func TestExtractExportNames_RealWorldExamples(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected []string
	}{
		{
			name: "Deno function with default handler",
			code: `import { serve } from "https://deno.land/std/http/server.ts";

async function handler(req: Request): Promise<Response> {
  return new Response("Hello, World!");
}

export { handler as default };`,
			expected: []string{"handler"},
		},
		{
			name: "multiple utilities exported",
			code: `const validate = (input: string) => input.length > 0;
const sanitize = (input: string) => input.trim();
const process = (input: string) => sanitize(validate(input));

export { validate, sanitize, process };`,
			expected: []string{"validate", "sanitize", "process"},
		},
		{
			name: "TypeScript with types and values",
			code: `interface Config {
  endpoint: string;
}

const defaultConfig: Config = { endpoint: "/api" };
function createClient(config: Config) {}

export { defaultConfig, createClient };`,
			expected: []string{"defaultConfig", "createClient"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractExportNames(tt.code)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsDataFile(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"countries.geojson", true},
		{"_shared/data/countries.geojson", true},
		{"./data/map.geojson", true},
		{"config.json", false},
		{"index.ts", false},
		{"helper.js", false},
		{"deno.json", false},
		{"file.GEOJSON", false}, // case-sensitive
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := isDataFile(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractDataFiles(t *testing.T) {
	supportingFiles := map[string]string{
		"utils.ts":        "export const x = 1",
		"data.geojson":    `{"type":"FeatureCollection"}`,
		"config.json":     `{"key":"value"}`,
		"helpers/db.ts":   "export const db = {}",
		"maps/us.geojson": `{"type":"Feature"}`,
	}

	sharedModules := map[string]string{
		"_shared/cors.ts":                "export const cors = {}",
		"_shared/data/countries.geojson": `{"type":"FeatureCollection","features":[]}`,
	}

	result := extractDataFiles(supportingFiles, sharedModules)

	assert.Len(t, result, 3)
	assert.Contains(t, result, "data.geojson")
	assert.Contains(t, result, "maps/us.geojson")
	assert.Contains(t, result, "_shared/data/countries.geojson")

	// Should not include .json files (they have native loader support)
	assert.NotContains(t, result, "config.json")
	// Should not include .ts files
	assert.NotContains(t, result, "utils.ts")
}

func TestLookupDataFile(t *testing.T) {
	dataFiles := map[string]string{
		"data.geojson":                   `{"exact":true}`,
		"_shared/data/countries.geojson": `{"shared":true}`,
		"nested/map.geojson":             `{"nested":true}`,
	}

	tests := []struct {
		name     string
		filePath string
		expected string
	}{
		{"exact match", "data.geojson", `{"exact":true}`},
		{"with ./ prefix", "./data.geojson", `{"exact":true}`},
		{"shared path", "_shared/data/countries.geojson", `{"shared":true}`},
		{"shared with ./", "./_shared/data/countries.geojson", `{"shared":true}`},
		{"nested path", "nested/map.geojson", `{"nested":true}`},
		{"not found", "missing.geojson", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := lookupDataFile(tt.filePath, dataFiles)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestInlineDataFiles(t *testing.T) {
	t.Run("geojson import inlined", func(t *testing.T) {
		dataFiles := map[string]string{
			"_shared/data/countries.geojson": `{"type":"FeatureCollection","features":[]}`,
		}

		code := `import countries from "_shared/data/countries.geojson"
console.log(countries)`

		result := inlineDataFiles(code, dataFiles)

		assert.Contains(t, result, `const countries = {"type":"FeatureCollection","features":[]}`)
		assert.NotContains(t, result, "import countries")
	})

	t.Run("multiple geojson imports", func(t *testing.T) {
		dataFiles := map[string]string{
			"data/us.geojson":     `{"type":"Feature","id":"US"}`,
			"data/europe.geojson": `{"type":"Feature","id":"EU"}`,
		}

		code := `import us from "./data/us.geojson"
import europe from "./data/europe.geojson"

export function getRegions() {
  return [us, europe]
}`

		result := inlineDataFiles(code, dataFiles)

		assert.Contains(t, result, `const us = {"type":"Feature","id":"US"}`)
		assert.Contains(t, result, `const europe = {"type":"Feature","id":"EU"}`)
		assert.NotContains(t, result, "import us from")
		assert.NotContains(t, result, "import europe from")
		assert.Contains(t, result, "export function getRegions")
	})

	t.Run("mixed imports preserved", func(t *testing.T) {
		dataFiles := map[string]string{
			"data.geojson": `{"type":"FeatureCollection"}`,
		}

		code := `import { z } from "npm:zod"
import data from "./data.geojson"

export function validate() {}`

		result := inlineDataFiles(code, dataFiles)

		// npm import should be preserved
		assert.Contains(t, result, `import { z } from "npm:zod"`)
		// geojson should be inlined
		assert.Contains(t, result, `const data = {"type":"FeatureCollection"}`)
		assert.NotContains(t, result, `import data from`)
	})

	t.Run("missing file leaves import unchanged", func(t *testing.T) {
		dataFiles := map[string]string{}

		code := `import missing from "./missing.geojson"
console.log(missing)`

		result := inlineDataFiles(code, dataFiles)

		// Import should be preserved since file wasn't found
		assert.Contains(t, result, `import missing from "./missing.geojson"`)
	})

	t.Run("no data files returns code unchanged", func(t *testing.T) {
		code := `import { z } from "npm:zod"
console.log(z)`

		result := inlineDataFiles(code, nil)

		assert.Equal(t, code, result)

		result = inlineDataFiles(code, map[string]string{})
		assert.Equal(t, code, result)
	})

	t.Run("single quotes supported", func(t *testing.T) {
		dataFiles := map[string]string{
			"data.geojson": `{"single":true}`,
		}

		code := `import data from './data.geojson'`

		result := inlineDataFiles(code, dataFiles)

		assert.Contains(t, result, `const data = {"single":true}`)
	})
}

func TestBuildDataFilePattern(t *testing.T) {
	pattern := buildDataFilePattern()

	tests := []struct {
		name     string
		input    string
		matches  bool
		varName  string
		filePath string
	}{
		{
			name:     "geojson double quotes",
			input:    `import countries from "_shared/data/countries.geojson"`,
			matches:  true,
			varName:  "countries",
			filePath: "_shared/data/countries.geojson",
		},
		{
			name:     "geojson single quotes",
			input:    `import data from './map.geojson'`,
			matches:  true,
			varName:  "data",
			filePath: "./map.geojson",
		},
		{
			name:    "json file not matched",
			input:   `import config from "./config.json"`,
			matches: false,
		},
		{
			name:    "ts file not matched",
			input:   `import { helper } from "./helper.ts"`,
			matches: false,
		},
		{
			name:    "npm import not matched",
			input:   `import { z } from "npm:zod"`,
			matches: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := pattern.FindStringSubmatch(tt.input)

			if tt.matches {
				require.NotNil(t, matches, "expected pattern to match")
				assert.Equal(t, tt.varName, matches[1])
				assert.Equal(t, tt.filePath, matches[2])
			} else {
				assert.Nil(t, matches, "expected pattern not to match")
			}
		})
	}
}
