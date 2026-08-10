package functions

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Unit tests for the additional pure + FS-dependent helpers in bundler.go that
// were not covered by bundler_test.go: the functional options (WithNpmRegistry,
// WithJsrRegistry), inlineSharedModules, filterEnvVars, getGlobalDenoConfig,
// mergeDenoConfig, and inlineAllImports. None invoke Deno.
//
// Convention matches bundler_test.go: testify, package functions (white-box so
// the unexported struct fields are reachable), table-driven where it fits.

// =============================================================================
// Functional options
// =============================================================================

func TestWithNpmRegistry(t *testing.T) {
	t.Parallel()
	t.Run("sets npmRegistry field", func(t *testing.T) {
		t.Parallel()
		b := &Bundler{}
		WithNpmRegistry("https://npm.example.com/")(b)
		assert.Equal(t, "https://npm.example.com/", b.npmRegistry)
	})
	t.Run("default bundler has empty npmRegistry", func(t *testing.T) {
		t.Parallel()
		b := &Bundler{}
		assert.Empty(t, b.npmRegistry)
	})
	t.Run("empty url accepted (no validation)", func(t *testing.T) {
		t.Parallel()
		b := &Bundler{}
		WithNpmRegistry("")(b)
		assert.Empty(t, b.npmRegistry)
	})
}

func TestWithJsrRegistry(t *testing.T) {
	t.Parallel()
	t.Run("sets jsrRegistry field", func(t *testing.T) {
		t.Parallel()
		b := &Bundler{}
		WithJsrRegistry("https://jsr.example.com/")(b)
		assert.Equal(t, "https://jsr.example.com/", b.jsrRegistry)
	})
	t.Run("both options compose independently", func(t *testing.T) {
		t.Parallel()
		b := &Bundler{}
		WithNpmRegistry("https://npm.x/")(b)
		WithJsrRegistry("https://jsr.x/")(b)
		assert.Equal(t, "https://npm.x/", b.npmRegistry)
		assert.Equal(t, "https://jsr.x/", b.jsrRegistry)
	})
}

// =============================================================================
// filterEnvVars
// =============================================================================
//
// Contract source: bundler.go:1061. Returns a new slice of env entries excluding
// any whose prefix is exactly "<name>=" for a name in names. The "=" is required,
// so "HOME" never matches "HOMEDIR=...". Order is preserved; input is not mutated.

func TestFilterEnvVars(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		env   []string
		names []string
		want  []string
	}{
		{
			name:  "empty env returns empty",
			env:   nil,
			names: []string{"HOME"},
			want:  []string{},
		},
		{
			name:  "empty names returns copy unchanged",
			env:   []string{"A=1", "B=2"},
			names: nil,
			want:  []string{"A=1", "B=2"},
		},
		{
			name:  "exact prefix match filtered",
			env:   []string{"HOME=/root", "PATH=/bin", "HOMEOWNER=/x"},
			names: []string{"HOME"},
			want:  []string{"PATH=/bin", "HOMEOWNER=/x"},
		},
		{
			name:  "substring but not prefix is kept",
			env:   []string{"HOMEDIR=/x", "HOME=/root"},
			names: []string{"HOME"},
			want:  []string{"HOMEDIR=/x"},
		},
		{
			name:  "multiple names all filtered",
			env:   []string{"A=1", "B=2", "C=3", "D=4"},
			names: []string{"A", "C"},
			want:  []string{"B=2", "D=4"},
		},
		{
			name:  "names not present returns unchanged",
			env:   []string{"A=1", "B=2"},
			names: []string{"Z"},
			want:  []string{"A=1", "B=2"},
		},
		{
			name:  "multiple entries with same prefix all filtered",
			env:   []string{"X=1", "X=2", "Y=3"},
			names: []string{"X"},
			want:  []string{"Y=3"},
		},
		{
			name:  "entries without equals are kept",
			env:   []string{"MALFORMED", "A=1"},
			names: []string{"A"},
			want:  []string{"MALFORMED"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := filterEnvVars(tt.env, tt.names...)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFilterEnvVars_DoesNotMutateInput(t *testing.T) {
	t.Parallel()
	env := []string{"HOME=/root", "PATH=/bin"}
	filterEnvVars(env, "HOME")
	// Original slice content and length are unchanged.
	assert.Equal(t, []string{"HOME=/root", "PATH=/bin"}, env)
}

func TestFilterEnvVars_NewBackingArray(t *testing.T) {
	t.Parallel()
	env := []string{"HOME=/root", "PATH=/bin"}
	got := filterEnvVars(env, "HOME")
	// Mutating the result must not affect the input.
	got[0] = "MUTATED=1"
	assert.Equal(t, "PATH=/bin", env[1])
}

// =============================================================================
// inlineSharedModules
// =============================================================================
//
// Contract source: bundler.go:620. Concatenates shared-module bodies (stripping
// ALL import lines) with the main code (stripping only _shared/ imports). Each
// shared module gets a "// Inlined from _shared/<cleanPath>" header. A trailing
// newline is appended after every source line, so output always ends in "\n".
// Map iteration order is non-deterministic, so multi-module cases assert on
// substrings rather than exact output.

func TestInlineSharedModules_EmptyShared_StripsOnlySharedImportsFromMain(t *testing.T) {
	t.Parallel()
	main := strings.Join([]string{
		`import { cors } from "_shared/cors.ts";`,
		`import { z } from "npm:zod";`,
		`export function handler() { return cors(); }`,
	}, "\n")

	got := inlineSharedModules(main, nil)
	// _shared/ import removed; npm: import preserved.
	assert.NotContains(t, got, `"_shared/cors.ts"`)
	assert.Contains(t, got, `import { z } from "npm:zod"`)
	assert.Contains(t, got, "export function handler() { return cors(); }")
}

func TestInlineSharedModules_SingleModule_ExactOutput(t *testing.T) {
	t.Parallel()
	shared := map[string]string{
		"_shared/cors.ts": strings.Join([]string{
			`import { something } from "npm:zod";`,
			`export function cors() { return "ok"; }`,
		}, "\n"),
	}
	main := `import { cors } from "_shared/cors.ts";` + "\n" +
		`export function handler() { return cors(); }`

	got := inlineSharedModules(main, shared)
	want := strings.Join([]string{
		`// Inlined from _shared/cors.ts`,
		`export function cors() { return "ok"; }`,
		``,
		`export function handler() { return cors(); }`,
		``,
	}, "\n")
	assert.Equal(t, want, got)
}

// The exact-output case above was getting fiddly with the placeholder; rewrite cleanly.
func TestInlineSharedModules_SingleModule_Precise(t *testing.T) {
	t.Parallel()
	shared := map[string]string{
		"_shared/cors.ts": strings.Join([]string{
			`import { something } from "npm:zod";`,
			`export function cors() { return "ok"; }`,
		}, "\n"),
	}
	main := `import { cors } from "_shared/cors.ts";` + "\n" +
		`export function handler() { return cors(); }`

	got := inlineSharedModules(main, shared)
	expected := "// Inlined from _shared/cors.ts\n" +
		"export function cors() { return \"ok\"; }\n" +
		"\n" +
		"export function handler() { return cors(); }\n"
	assert.Equal(t, expected, got)
}

func TestInlineSharedModules_ModuleWithoutSharedPrefix(t *testing.T) {
	t.Parallel()
	// A module path without the "_shared/" prefix still gets inlined; the header
	// uses the path as-is (TrimPrefix is a no-op).
	shared := map[string]string{
		"utils.ts": `export const x = 1;`,
	}
	got := inlineSharedModules("const y = x;", shared)
	assert.Contains(t, got, "// Inlined from _shared/utils.ts\n")
	assert.Contains(t, got, "export const x = 1;")
}

func TestInlineSharedModules_StripsAllImportsFromSharedModules(t *testing.T) {
	t.Parallel()
	shared := map[string]string{
		"_shared/m.ts": strings.Join([]string{
			`import { a } from "npm:zod";`, // blocked-style import, still stripped
			`import { b } from "./local";`, // relative import, stripped
			`export const m = a + b;`,
		}, "\n"),
	}
	got := inlineSharedModules("", shared)
	assert.NotContains(t, got, "import")
	assert.Contains(t, got, "export const m = a + b;")
}

func TestInlineSharedModules_PreservesNonSharedImportsInMain(t *testing.T) {
	t.Parallel()
	main := strings.Join([]string{
		`import { cors } from "_shared/cors.ts";`,
		`import { z } from "npm:zod";`,
		`import { helper } from "./utils";`,
		`console.log("hi");`,
	}, "\n")
	got := inlineSharedModules(main, nil)
	// Only the _shared/ import is removed from main.
	assert.NotContains(t, got, "_shared/cors.ts")
	assert.Contains(t, got, `import { z } from "npm:zod"`)
	assert.Contains(t, got, `import { helper } from "./utils"`)
	assert.Contains(t, got, `console.log("hi")`)
}

func TestInlineSharedModules_SideEffectSharedImportStripped(t *testing.T) {
	t.Parallel()
	// A side-effect import `import "_shared/x"` starts with "import " and
	// contains "_shared/" so it is stripped from main.
	main := `import "_shared/polyfill"` + "\n" + `console.log("ran");`
	got := inlineSharedModules(main, nil)
	assert.NotContains(t, got, "_shared/polyfill")
	assert.Contains(t, got, `console.log("ran")`)
}

// =============================================================================
// getGlobalDenoConfig (FS-dependent — uses temp dirs)
// =============================================================================
//
// Contract source: bundler.go:134. Returns the dev config path if it exists,
// else the prod config path if it exists, else "".

func TestGetGlobalDenoConfig(t *testing.T) {
	t.Parallel()

	t.Run("dev config takes precedence when both exist", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		dev := filepath.Join(dir, "deno.dev.json")
		prod := filepath.Join(dir, "deno.json")
		require.NoError(t, os.WriteFile(dev, []byte("{}"), 0o600))
		require.NoError(t, os.WriteFile(prod, []byte("{}"), 0o600))

		b := &Bundler{globalDenoConfig: prod, globalDenoConfigDev: dev}
		assert.Equal(t, dev, b.getGlobalDenoConfig())
	})

	t.Run("prod config used when dev missing", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		dev := filepath.Join(dir, "deno.dev.json") // not created
		prod := filepath.Join(dir, "deno.json")
		require.NoError(t, os.WriteFile(prod, []byte("{}"), 0o600))

		b := &Bundler{globalDenoConfig: prod, globalDenoConfigDev: dev}
		assert.Equal(t, prod, b.getGlobalDenoConfig())
	})

	t.Run("empty string when neither exists", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		b := &Bundler{
			globalDenoConfig:    filepath.Join(dir, "nope.json"),
			globalDenoConfigDev: filepath.Join(dir, "nope.dev.json"),
		}
		assert.Empty(t, b.getGlobalDenoConfig())
	})
}

// =============================================================================
// mergeDenoConfig (FS-dependent — global config read from temp dir)
// =============================================================================
//
// Contract source: bundler.go:816. Merges global config imports + optional
// _shared/ mapping + function config imports (function takes precedence). Returns
// indented JSON with an "imports" object. Non-string import values are skipped.
// Malformed global config is silently ignored; malformed function config errors.

func parseImports(t *testing.T, configJSON string) map[string]string {
	t.Helper()
	var cfg struct {
		Imports map[string]string `json:"imports"`
	}
	require.NoError(t, json.Unmarshal([]byte(configJSON), &cfg))
	return cfg.Imports
}

func TestMergeDenoConfig_NoGlobalNoFunction(t *testing.T) {
	t.Parallel()
	b := &Bundler{globalDenoConfig: "/nonexistent", globalDenoConfigDev: "/nonexistent"}
	got, err := b.mergeDenoConfig("", false)
	require.NoError(t, err)
	assert.Equal(t, map[string]string{}, parseImports(t, got))
}

func TestMergeDenoConfig_HasSharedModulesInjectsMapping(t *testing.T) {
	t.Parallel()
	b := &Bundler{globalDenoConfig: "/nonexistent", globalDenoConfigDev: "/nonexistent"}
	got, err := b.mergeDenoConfig("", true)
	require.NoError(t, err)
	imports := parseImports(t, got)
	assert.Equal(t, "./_shared/", imports["_shared/"])
}

func TestMergeDenoConfig_GlobalImportsPreserved(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	globalPath := filepath.Join(dir, "deno.json")
	require.NoError(t, os.WriteFile(globalPath,
		[]byte(`{"imports": {"zod": "npm:zod", "std/": "jsr:@std/"}}`), 0o600))

	b := &Bundler{globalDenoConfig: globalPath, globalDenoConfigDev: "/nonexistent"}
	got, err := b.mergeDenoConfig("", false)
	require.NoError(t, err)
	imports := parseImports(t, got)
	assert.Equal(t, "npm:zod", imports["zod"])
	assert.Equal(t, "jsr:@std/", imports["std/"])
}

func TestMergeDenoConfig_FunctionOverridesGlobal(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	globalPath := filepath.Join(dir, "deno.json")
	require.NoError(t, os.WriteFile(globalPath,
		[]byte(`{"imports": {"zod": "npm:zod@3.0.0"}}`), 0o600))

	b := &Bundler{globalDenoConfig: globalPath, globalDenoConfigDev: "/nonexistent"}
	fnConfig := `{"imports": {"zod": "npm:zod@3.1.0"}}`
	got, err := b.mergeDenoConfig(fnConfig, false)
	require.NoError(t, err)
	imports := parseImports(t, got)
	assert.Equal(t, "npm:zod@3.1.0", imports["zod"], "function config must override global")
}

func TestMergeDenoConfig_FunctionOverridesSharedMapping(t *testing.T) {
	t.Parallel()
	b := &Bundler{globalDenoConfig: "/nonexistent", globalDenoConfigDev: "/nonexistent"}
	// Function provides its own _shared/ mapping; it must win over the injected one.
	fnConfig := `{"imports": {"_shared/": "./custom_shared/"}}`
	got, err := b.mergeDenoConfig(fnConfig, true)
	require.NoError(t, err)
	imports := parseImports(t, got)
	assert.Equal(t, "./custom_shared/", imports["_shared/"])
}

func TestMergeDenoConfig_NonStringImportValuesSkipped(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	globalPath := filepath.Join(dir, "deno.json")
	// "scopes" is a nested object (non-string); must be skipped, not panic.
	require.NoError(t, os.WriteFile(globalPath,
		[]byte(`{"imports": {"zod": "npm:zod", "scopes": {"nested": true}}}`), 0o600))

	b := &Bundler{globalDenoConfig: globalPath, globalDenoConfigDev: "/nonexistent"}
	got, err := b.mergeDenoConfig("", false)
	require.NoError(t, err)
	imports := parseImports(t, got)
	assert.Equal(t, "npm:zod", imports["zod"])
	_, present := imports["scopes"]
	assert.False(t, present, "non-string import value must be skipped")
}

func TestMergeDenoConfig_MalformedFunctionConfigErrors(t *testing.T) {
	t.Parallel()
	b := &Bundler{globalDenoConfig: "/nonexistent", globalDenoConfigDev: "/nonexistent"}
	_, err := b.mergeDenoConfig(`{not valid json`, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse function's deno.json")
}

func TestMergeDenoConfig_MalformedGlobalConfigIgnored(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	globalPath := filepath.Join(dir, "deno.json")
	require.NoError(t, os.WriteFile(globalPath, []byte(`{not valid json`), 0o600))

	b := &Bundler{globalDenoConfig: globalPath, globalDenoConfigDev: "/nonexistent"}
	// Malformed global config is silently ignored — no error, empty imports.
	got, err := b.mergeDenoConfig("", false)
	require.NoError(t, err)
	assert.Equal(t, map[string]string{}, parseImports(t, got))
}

// =============================================================================
// inlineAllImports
// =============================================================================
//
// Contract source: bundler.go:878. Inlines external modules (absolute-path
// imports from deno.json), shared modules, then the main code with ALL imports
// stripped (single- and multi-line). deno.json parse errors are returned;
// missing external-module files are warn-and-skip (no error).

func TestInlineAllImports_MalformedDenoJSONErrors(t *testing.T) {
	t.Parallel()
	_, err := inlineAllImports("const x = 1;", nil, "{bad json")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse deno.json")
}

func TestInlineAllImports_EmptyDenoJSONStripsAllMainImports(t *testing.T) {
	t.Parallel()
	main := strings.Join([]string{
		`import { z } from "npm:zod";`,
		`import { cors } from "./cors";`,
		`const handler = () => z.parse(cors());`,
		`export { handler };`,
	}, "\n")
	got, err := inlineAllImports(main, nil, "")
	require.NoError(t, err)
	// ALL import lines stripped from main code.
	assert.NotContains(t, got, "import")
	assert.Contains(t, got, "const handler = () => z.parse(cors());")
	assert.Contains(t, got, "export { handler };")
}

func TestInlineAllImports_MultilineImportStripped(t *testing.T) {
	t.Parallel()
	main := strings.Join([]string{
		`import {`,
		`  z,`,
		`} from "npm:zod";`,
		`const x = z;`,
	}, "\n")
	got, err := inlineAllImports(main, nil, "")
	require.NoError(t, err)
	// The entire multi-line import block (until the line containing " from ") is removed.
	assert.NotContains(t, got, "npm:zod")
	assert.NotContains(t, got, "  z,")
	assert.Contains(t, got, "const x = z;")
}

func TestInlineAllImports_SharedModulesInlined(t *testing.T) {
	t.Parallel()
	shared := map[string]string{
		"_shared/cors.ts": strings.Join([]string{
			`import { x } from "npm:zod";`,
			`export function cors() { return "ok"; }`,
		}, "\n"),
	}
	main := `const h = () => cors();`
	got, err := inlineAllImports(main, shared, "")
	require.NoError(t, err)
	assert.Contains(t, got, "// Inlined from _shared/cors.ts")
	assert.Contains(t, got, "export function cors() { return \"ok\"; }")
	// Shared-module imports are stripped.
	assert.NotContains(t, got, "npm:zod")
}

func TestInlineAllImports_RelativePathImportSkipped(t *testing.T) {
	t.Parallel()
	// A relative-path import value (not absolute) is not loaded as external.
	denoJSON := `{"imports": {"utils": "./utils.js"}}`
	got, err := inlineAllImports(`const x = 1;`, nil, denoJSON)
	require.NoError(t, err)
	assert.NotContains(t, got, "Inlined from utils")
}

func TestInlineAllImports_AbsolutePathMissingFileSkipped(t *testing.T) {
	t.Parallel()
	denoJSON := `{"imports": {"sdk": "/nonexistent/path/module.js"}}`
	got, err := inlineAllImports(`const x = 1;`, nil, denoJSON)
	require.NoError(t, err)
	// Missing file is warn-and-skip: no error, no inline.
	assert.NotContains(t, got, "Inlined from sdk")
}

func TestInlineAllImports_AbsolutePathExistingFileInlined(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	modPath := filepath.Join(dir, "sdk.js")
	require.NoError(t, os.WriteFile(modPath,
		[]byte(strings.Join([]string{
			`import { internal } from "npm:internal";`,
			`export const api = () => "v";`,
			`export { api as default };`,
		}, "\n")), 0o600))

	denoJSON := `{"imports": {"sdk": "` + modPath + `"}}`
	main := `const result = api();`
	got, err := inlineAllImports(main, nil, denoJSON)
	require.NoError(t, err)
	// IIFE wrapper present, exports destructured.
	assert.Contains(t, got, "// Inlined from sdk")
	assert.Contains(t, got, "const __external_module = (() => {")
	assert.Contains(t, got, "})();")
	// export names (pre-alias) appear in both the return and destructure.
	assert.Contains(t, got, "return { api }")
	assert.Contains(t, got, "const { api } = __external_module;")
	// import/export lines stripped from the module body.
	lines := strings.Split(got, "\n")
	for _, l := range lines {
		trimmed := strings.TrimSpace(l)
		assert.False(t, strings.HasPrefix(trimmed, "import "), "stray import in output: %q", l)
	}
}

func TestInlineAllImports_ExternalModuleWithoutExportsNoDestructure(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	modPath := filepath.Join(dir, "sideeffect.js")
	// No export statement -> extractExportNames returns nil.
	require.NoError(t, os.WriteFile(modPath,
		[]byte(`console.log("loaded");`), 0o600))

	denoJSON := `{"imports": {"se": "` + modPath + `"}}`
	got, err := inlineAllImports(`const x = 1;`, nil, denoJSON)
	require.NoError(t, err)
	assert.Contains(t, got, "// Inlined from se")
	assert.Contains(t, got, "console.log(\"loaded\");")
	// No return/destructure emitted when there are no exports.
	assert.NotContains(t, got, "return {")
	assert.NotContains(t, got, "__external_module;")
}

func TestInlineAllImports_SharedMappingInDenoJSONSkipped(t *testing.T) {
	t.Parallel()
	// The "_shared/" mapping in deno.json must not be treated as an external module.
	denoJSON := `{"imports": {"_shared/": "./_shared/"}}`
	got, err := inlineAllImports(`const x = 1;`, nil, denoJSON)
	require.NoError(t, err)
	assert.NotContains(t, got, "Inlined from _shared/")
}
