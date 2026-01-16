package runtime

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// =============================================================================
// extractImports Tests
// =============================================================================

func TestExtractImports_SingleLineImports(t *testing.T) {
	tests := []struct {
		name           string
		code           string
		expectedImport string
		expectedCode   string
	}{
		{
			name: "basic import statement",
			code: `import { something } from 'module';
const x = 1;`,
			expectedImport: "import { something } from 'module';",
			expectedCode:   "const x = 1;",
		},
		{
			name: "import without braces",
			code: `import module from 'module';
const x = 1;`,
			expectedImport: "import module from 'module';",
			expectedCode:   "const x = 1;",
		},
		{
			name: "import star",
			code: `import * as mod from 'module';
function test() {}`,
			expectedImport: "import * as mod from 'module';",
			expectedCode:   "function test() {}",
		},
		{
			name: "import{ (no space)",
			code: `import{a, b} from 'module';
const x = 1;`,
			expectedImport: "import{a, b} from 'module';",
			expectedCode:   "const x = 1;",
		},
		{
			name: "export star",
			code: `export * from 'module';
const x = 1;`,
			expectedImport: "export * from 'module';",
			expectedCode:   "const x = 1;",
		},
		{
			name: "multiple imports",
			code: `import { a } from 'module1';
import { b } from 'module2';
import * as c from 'module3';
const x = 1;`,
			expectedImport: "import { a } from 'module1';\nimport { b } from 'module2';\nimport * as c from 'module3';",
			expectedCode:   "const x = 1;",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			imports, remaining := extractImports(tt.code)
			assert.Equal(t, tt.expectedImport, imports)
			assert.Equal(t, tt.expectedCode, remaining)
		})
	}
}

func TestExtractImports_ExportTypes(t *testing.T) {
	tests := []struct {
		name           string
		code           string
		expectedImport string
		expectedCode   string
	}{
		{
			name: "single line export type",
			code: `export type MyType = string;
const x = 1;`,
			expectedImport: "export type MyType = string;",
			expectedCode:   "const x = 1;",
		},
		{
			name: "multi-line export type",
			code: `export type MyType = {
  name: string;
  age: number;
};
const x = 1;`,
			expectedImport: "export type MyType = {\n  name: string;\n  age: number;\n};",
			expectedCode:   "const x = 1;",
		},
		{
			name: "export interface",
			code: `export interface Person {
  name: string;
}
const x = 1;`,
			expectedImport: "export interface Person {\n  name: string;\n}",
			expectedCode:   "const x = 1;",
		},
		{
			name: "export enum",
			code: `export enum Status {
  Active = 'active',
  Inactive = 'inactive'
}
const x = 1;`,
			expectedImport: "export enum Status {\n  Active = 'active',\n  Inactive = 'inactive'\n}",
			expectedCode:   "const x = 1;",
		},
		{
			name: "nested braces in type",
			code: `export type Complex = {
  nested: {
    value: number;
  };
};
const x = 1;`,
			expectedImport: "export type Complex = {\n  nested: {\n    value: number;\n  };\n};",
			expectedCode:   "const x = 1;",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			imports, remaining := extractImports(tt.code)
			assert.Equal(t, tt.expectedImport, imports)
			assert.Equal(t, tt.expectedCode, remaining)
		})
	}
}

func TestExtractImports_ExportBraces(t *testing.T) {
	tests := []struct {
		name           string
		code           string
		expectedImport string
		expectedCode   string
	}{
		{
			name: "single line export braces",
			code: `export { a, b, c };
const x = 1;`,
			expectedImport: "export { a, b, c };",
			expectedCode:   "const x = 1;",
		},
		{
			name: "multi-line export braces",
			code: `export {
  functionA,
  functionB
};
const x = 1;`,
			expectedImport: "export {\n  functionA,\n  functionB\n};",
			expectedCode:   "const x = 1;",
		},
		{
			name: "export with as keyword",
			code: `export { a as aliasA, b as aliasB };
const x = 1;`,
			expectedImport: "export { a as aliasA, b as aliasB };",
			expectedCode:   "const x = 1;",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			imports, remaining := extractImports(tt.code)
			assert.Equal(t, tt.expectedImport, imports)
			assert.Equal(t, tt.expectedCode, remaining)
		})
	}
}

func TestExtractImports_NoImports(t *testing.T) {
	tests := []struct {
		name         string
		code         string
		expectedCode string
	}{
		{
			name:         "no imports at all",
			code:         "const x = 1;\nconst y = 2;",
			expectedCode: "const x = 1;\nconst y = 2;",
		},
		{
			name:         "empty code",
			code:         "",
			expectedCode: "",
		},
		{
			name:         "only whitespace",
			code:         "   \n   \n   ",
			expectedCode: "   \n   \n   ",
		},
		{
			name:         "only comments",
			code:         "// comment\n/* block comment */",
			expectedCode: "// comment\n/* block comment */",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			imports, remaining := extractImports(tt.code)
			assert.Empty(t, imports)
			assert.Equal(t, tt.expectedCode, remaining)
		})
	}
}

func TestExtractImports_MixedCode(t *testing.T) {
	t.Run("imports and exports mixed with code", func(t *testing.T) {
		code := `import { Database } from './types';
import * as helpers from './helpers';
export type Config = { host: string; port: number; };
export interface Handler {
  handle(req: Request): Response;
}

const DEFAULT_PORT = 8080;

export function createHandler() {
  return null;
}

export { createHandler as handler };`

		imports, remaining := extractImports(code)

		// Check that imports contains the import/export declarations
		assert.Contains(t, imports, "import { Database }")
		assert.Contains(t, imports, "import * as helpers")
		assert.Contains(t, imports, "export type Config")
		assert.Contains(t, imports, "export interface Handler")
		assert.Contains(t, imports, "export { createHandler as handler }")

		// Check that remaining code is the function code
		assert.Contains(t, remaining, "const DEFAULT_PORT = 8080")
		assert.Contains(t, remaining, "export function createHandler()")
	})
}

func TestExtractImports_EdgeCases(t *testing.T) {
	t.Run("import statement in string literal is not extracted", func(t *testing.T) {
		code := `const str = "import { something } from 'module'";
const x = 1;`

		imports, remaining := extractImports(code)

		// The string containing import should NOT be extracted
		assert.Empty(t, imports)
		assert.Contains(t, remaining, `"import { something } from 'module'"`)
	})

	t.Run("import comment is not extracted", func(t *testing.T) {
		code := `// import { something } from 'module';
const x = 1;`

		imports, remaining := extractImports(code)

		// Comments are not imports
		assert.Empty(t, imports)
		assert.Contains(t, remaining, "// import { something }")
	})

	t.Run("import after code is extracted", func(t *testing.T) {
		// Note: This tests current behavior - imports anywhere are extracted
		code := `const x = 1;
import { late } from 'module';`

		imports, remaining := extractImports(code)

		// Import is extracted regardless of position
		assert.Equal(t, "import { late } from 'module';", imports)
		assert.Contains(t, remaining, "const x = 1;")
	})

	t.Run("deeply nested braces in type", func(t *testing.T) {
		code := `export type DeepType = {
  level1: {
    level2: {
      level3: {
        value: number;
      };
    };
  };
};
const after = true;`

		imports, remaining := extractImports(code)

		// All nested braces should be captured
		assert.Contains(t, imports, "export type DeepType")
		assert.Contains(t, imports, "level3")
		assert.Contains(t, remaining, "const after = true")
	})

	t.Run("unclosed brace at end", func(t *testing.T) {
		// Edge case: what happens with invalid syntax
		code := `export type Incomplete = {
  name: string;`

		imports, remaining := extractImports(code)

		// Should still extract what it can
		assert.Contains(t, imports, "export type Incomplete")
		// Remaining might be empty since all lines were captured
		_ = remaining // Just testing no panic
	})
}

func TestExtractImports_PreservesOrder(t *testing.T) {
	t.Run("preserves line order in imports", func(t *testing.T) {
		code := `import { first } from 'first';
import { second } from 'second';
import { third } from 'third';
const x = 1;`

		imports, _ := extractImports(code)

		// Check order is preserved
		firstIdx := strings.Index(imports, "first")
		secondIdx := strings.Index(imports, "second")
		thirdIdx := strings.Index(imports, "third")

		assert.Less(t, firstIdx, secondIdx)
		assert.Less(t, secondIdx, thirdIdx)
	})

	t.Run("preserves line order in remaining code", func(t *testing.T) {
		code := `import { x } from 'x';
const a = 1;
const b = 2;
const c = 3;`

		_, remaining := extractImports(code)

		// Check order is preserved
		aIdx := strings.Index(remaining, "const a")
		bIdx := strings.Index(remaining, "const b")
		cIdx := strings.Index(remaining, "const c")

		assert.Less(t, aIdx, bIdx)
		assert.Less(t, bIdx, cIdx)
	})
}

func TestExtractImports_RealWorldExamples(t *testing.T) {
	t.Run("typical edge function code", func(t *testing.T) {
		code := `import { createClient } from '@supabase/supabase-js';
import type { Database } from './types';

export interface Config {
  apiKey: string;
  baseUrl: string;
}

const supabase = createClient<Database>(
  Deno.env.get('SUPABASE_URL')!,
  Deno.env.get('SUPABASE_ANON_KEY')!
);

export async function handler(req: Request) {
  const { data, error } = await supabase.from('users').select('*');
  return new Response(JSON.stringify({ data, error }));
}`

		imports, remaining := extractImports(code)

		// Imports should have the import statements and type exports
		assert.Contains(t, imports, "@supabase/supabase-js")
		assert.Contains(t, imports, "./types")
		assert.Contains(t, imports, "export interface Config")

		// Remaining should have the actual code
		assert.Contains(t, remaining, "const supabase = createClient")
		assert.Contains(t, remaining, "export async function handler")
	})

	t.Run("job function code", func(t *testing.T) {
		code := `import { z } from 'zod';

export type JobPayload = {
  userId: string;
  action: 'sync' | 'cleanup';
};

export async function handler(
  req: Request,
  fluxbase: any,
  fluxbaseService: any,
  job: any
) {
  const payload = await job.getJobPayload();
  job.reportProgress(50, 'Processing...');
  return { success: true };
}`

		imports, remaining := extractImports(code)

		assert.Contains(t, imports, "import { z }")
		assert.Contains(t, imports, "export type JobPayload")
		assert.Contains(t, remaining, "export async function handler")
	})
}

func TestExtractImports_WhitespaceHandling(t *testing.T) {
	tests := []struct {
		name         string
		code         string
		expectImport bool
	}{
		{
			name:         "import with leading spaces",
			code:         "   import { x } from 'x';",
			expectImport: true,
		},
		{
			name:         "import with tabs",
			code:         "\timport { x } from 'x';",
			expectImport: true,
		},
		{
			name:         "import with mixed whitespace",
			code:         "  \t  import { x } from 'x';",
			expectImport: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			imports, _ := extractImports(tt.code)
			if tt.expectImport {
				assert.NotEmpty(t, imports)
				assert.Contains(t, imports, "import")
			}
		})
	}
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkExtractImports_Simple(b *testing.B) {
	code := `import { a } from 'a';
import { b } from 'b';
const x = 1;`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = extractImports(code)
	}
}

func BenchmarkExtractImports_Complex(b *testing.B) {
	code := `import { createClient } from '@supabase/supabase-js';
import type { Database } from './types';
export type Config = { apiKey: string; baseUrl: string; };
export interface Handler {
  handle(req: Request): Response;
}
const client = createClient();
export function handler(req: Request) { return new Response(); }`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = extractImports(code)
	}
}

func BenchmarkExtractImports_DeepNesting(b *testing.B) {
	code := `export type DeepType = {
  a: { b: { c: { d: { e: { f: number; }; }; }; }; };
};
const x = 1;`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = extractImports(code)
	}
}

func BenchmarkExtractImports_ManyImports(b *testing.B) {
	var builder strings.Builder
	for i := 0; i < 50; i++ {
		builder.WriteString("import { x" + string(rune('a'+i%26)) + " } from 'module" + string(rune('a'+i%26)) + "';\n")
	}
	builder.WriteString("const code = 1;")
	code := builder.String()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = extractImports(code)
	}
}

func BenchmarkExtractImports_LargeFile(b *testing.B) {
	var builder strings.Builder
	// Add some imports
	builder.WriteString("import { a } from 'a';\nimport { b } from 'b';\n")
	// Add large amount of code
	for i := 0; i < 1000; i++ {
		builder.WriteString("const variable" + string(rune('a'+i%26)) + " = " + string(rune('0'+i%10)) + ";\n")
	}
	code := builder.String()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = extractImports(code)
	}
}
