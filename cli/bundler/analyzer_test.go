package bundler

import (
	"bytes"
	"context"
	"testing"
)

func TestAnalyzeBundle_Simple(t *testing.T) {
	code := `
import { helper } from "./_shared/utils.ts";
export default function handler(req) {
    return helper(req);
}
`
	sharedModules := map[string]string{
		"_shared/utils.ts": `export function helper(x) { return x; }`,
	}

	analyzer := NewAnalyzer("/tmp")
	result, err := analyzer.AnalyzeBundle(context.Background(), code, "test-fn", sharedModules)

	if err != nil {
		t.Fatalf("AnalyzeBundle failed: %v", err)
	}

	if result.FunctionName != "test-fn" {
		t.Errorf("Expected function name 'test-fn', got '%s'", result.FunctionName)
	}

	if result.TotalBytes <= 0 {
		t.Errorf("Expected positive TotalBytes, got %d", result.TotalBytes)
	}

	if len(result.InputFiles) == 0 {
		t.Error("Expected at least one input file")
	}
}

func TestAnalyzeBundle_WithNpmImports(t *testing.T) {
	code := `
import { z } from "npm:zod";
export default function handler(req) {
    return z.string().parse(req.body);
}
`
	analyzer := NewAnalyzer("/tmp")
	result, err := analyzer.AnalyzeBundle(context.Background(), code, "test-fn", nil)

	if err != nil {
		t.Fatalf("AnalyzeBundle failed: %v", err)
	}

	found := false
	for _, ext := range result.ExternalImports {
		if ext == "npm:zod" {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Expected 'npm:zod' in external imports, got: %v", result.ExternalImports)
	}
}

func TestAnalyzeBundle_BareImportTransform(t *testing.T) {
	// Test that bare imports like "lodash" get transformed to "npm:lodash"
	code := `
import lodash from "lodash";
export default function handler(req) {
    return lodash.get(req, "foo");
}
`
	analyzer := NewAnalyzer("/tmp")
	result, err := analyzer.AnalyzeBundle(context.Background(), code, "test-fn", nil)

	if err != nil {
		t.Fatalf("AnalyzeBundle failed: %v", err)
	}

	found := false
	for _, ext := range result.ExternalImports {
		if ext == "npm:lodash" {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Expected 'npm:lodash' in external imports (transformed from bare 'lodash'), got: %v", result.ExternalImports)
	}
}

func TestDisplayAnalysis(t *testing.T) {
	result := &AnalysisResult{
		FunctionName: "my-function",
		TotalBytes:   10240,
		InputFiles: []FileAnalysis{
			{Path: "<entry>", Bytes: 1024, BytesInOutput: 2048, Percentage: 20.0},
			{Path: "_shared/utils.ts", Bytes: 512, BytesInOutput: 1024, Percentage: 10.0},
		},
		ExternalImports: []string{"npm:zod"},
	}

	var buf bytes.Buffer
	DisplayAnalysis(&buf, result, false)

	output := buf.String()

	if !bytes.Contains([]byte(output), []byte("my-function")) {
		t.Error("Expected function name in output")
	}

	if !bytes.Contains([]byte(output), []byte("npm:zod")) {
		t.Error("Expected external import in output")
	}

	if !bytes.Contains([]byte(output), []byte("<entry>")) {
		t.Error("Expected entry file in output")
	}
}

func TestDisplaySummary(t *testing.T) {
	results := []*AnalysisResult{
		{FunctionName: "func-a", TotalBytes: 10240, InputFiles: []FileAnalysis{{}, {}}, ExternalImports: []string{"npm:zod"}},
		{FunctionName: "func-b", TotalBytes: 5120, InputFiles: []FileAnalysis{{}}, ExternalImports: nil},
	}

	var buf bytes.Buffer
	DisplaySummary(&buf, results)

	output := buf.String()

	if !bytes.Contains([]byte(output), []byte("func-a")) {
		t.Error("Expected func-a in output")
	}

	if !bytes.Contains([]byte(output), []byte("func-b")) {
		t.Error("Expected func-b in output")
	}

	if !bytes.Contains([]byte(output), []byte("TOTAL")) {
		t.Error("Expected TOTAL in output")
	}
}
