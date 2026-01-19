package bundler

import (
	"testing"
)

func TestIsDataFile(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"_shared/data/countries.geojson", true},
		{"data/file.geojson", true},
		{"file.geojson", true},
		{"file.json", false},
		{"file.ts", false},
		{"file.js", false},
		{"geojson.ts", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := isDataFile(tt.path)
			if result != tt.expected {
				t.Errorf("isDataFile(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestExtractDataFiles(t *testing.T) {
	sharedModules := map[string]string{
		"_shared/utils.ts":               "export const foo = 1;",
		"_shared/data/countries.geojson": `{"type":"FeatureCollection"}`,
		"_shared/data/timezones.geojson": `{"type":"FeatureCollection"}`,
		"_shared/services/geo.ts":        "import x from '../data/countries.geojson';",
	}

	dataFiles := extractDataFiles(sharedModules)

	if len(dataFiles) != 2 {
		t.Errorf("extractDataFiles returned %d files, want 2", len(dataFiles))
	}

	if _, ok := dataFiles["_shared/data/countries.geojson"]; !ok {
		t.Error("extractDataFiles missing _shared/data/countries.geojson")
	}

	if _, ok := dataFiles["_shared/data/timezones.geojson"]; !ok {
		t.Error("extractDataFiles missing _shared/data/timezones.geojson")
	}
}

func TestLookupDataFile(t *testing.T) {
	dataFiles := map[string]string{
		"_shared/data/countries.geojson": `{"type":"FeatureCollection","name":"countries"}`,
		"_shared/data/timezones.geojson": `{"type":"FeatureCollection","name":"timezones"}`,
	}

	tests := []struct {
		name           string
		importPath     string
		sourceFilePath string
		wantFound      bool
		wantContent    string
	}{
		{
			name:           "exact path",
			importPath:     "_shared/data/countries.geojson",
			sourceFilePath: "",
			wantFound:      true,
			wantContent:    `{"type":"FeatureCollection","name":"countries"}`,
		},
		{
			name:           "relative path from nested file",
			importPath:     "../../data/countries.geojson",
			sourceFilePath: "_shared/services/external/geo.ts",
			wantFound:      true,
			wantContent:    `{"type":"FeatureCollection","name":"countries"}`,
		},
		{
			name:           "relative path from service",
			importPath:     "../data/timezones.geojson",
			sourceFilePath: "_shared/services/geo.ts",
			wantFound:      true,
			wantContent:    `{"type":"FeatureCollection","name":"timezones"}`,
		},
		{
			name:           "relative path with ./",
			importPath:     "./data/countries.geojson",
			sourceFilePath: "_shared/index.ts",
			wantFound:      true,
			wantContent:    `{"type":"FeatureCollection","name":"countries"}`,
		},
		{
			name:           "not found",
			importPath:     "nonexistent.geojson",
			sourceFilePath: "",
			wantFound:      false,
			wantContent:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := lookupDataFile(tt.importPath, tt.sourceFilePath, dataFiles)
			found := content != ""

			if found != tt.wantFound {
				t.Errorf("lookupDataFile(%q, %q) found=%v, want found=%v",
					tt.importPath, tt.sourceFilePath, found, tt.wantFound)
			}

			if content != tt.wantContent {
				t.Errorf("lookupDataFile(%q, %q) = %q, want %q",
					tt.importPath, tt.sourceFilePath, content, tt.wantContent)
			}
		})
	}
}

func TestInlineDataFiles(t *testing.T) {
	dataFiles := map[string]string{
		"_shared/data/countries.geojson": `{"type":"FeatureCollection"}`,
	}

	tests := []struct {
		name           string
		code           string
		sourceFilePath string
		wantContains   string
		wantNotContain string
	}{
		{
			name:           "inline relative import",
			code:           `import countriesRaw from '../../data/countries.geojson';`,
			sourceFilePath: "_shared/services/external/geo.ts",
			wantContains:   `const countriesRaw = {"type":"FeatureCollection"}`,
			wantNotContain: "import",
		},
		{
			name:           "inline absolute import",
			code:           `import data from '_shared/data/countries.geojson';`,
			sourceFilePath: "main.ts",
			wantContains:   `const data = {"type":"FeatureCollection"}`,
			wantNotContain: "import",
		},
		{
			name:           "preserve non-data imports",
			code:           `import { foo } from './utils.ts';`,
			sourceFilePath: "_shared/index.ts",
			wantContains:   `import { foo } from './utils.ts'`,
			wantNotContain: "const foo",
		},
		{
			name:           "missing file left as-is",
			code:           `import missing from './nonexistent.geojson';`,
			sourceFilePath: "_shared/index.ts",
			wantContains:   `import missing from './nonexistent.geojson'`,
			wantNotContain: "const missing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := inlineDataFiles(tt.code, tt.sourceFilePath, dataFiles)

			if tt.wantContains != "" && !contains(result, tt.wantContains) {
				t.Errorf("inlineDataFiles result should contain %q, got:\n%s", tt.wantContains, result)
			}

			if tt.wantNotContain != "" && contains(result, tt.wantNotContain) {
				t.Errorf("inlineDataFiles result should NOT contain %q, got:\n%s", tt.wantNotContain, result)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && searchString(s, substr)))
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
