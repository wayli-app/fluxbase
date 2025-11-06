package functions

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateFunctionName(t *testing.T) {
	tests := []struct {
		name      string
		funcName  string
		wantError bool
	}{
		{
			name:      "valid alphanumeric name",
			funcName:  "myfunction",
			wantError: false,
		},
		{
			name:      "valid name with hyphens",
			funcName:  "my-function",
			wantError: false,
		},
		{
			name:      "valid name with underscores",
			funcName:  "my_function",
			wantError: false,
		},
		{
			name:      "valid mixed alphanumeric with symbols",
			funcName:  "my-function_123",
			wantError: false,
		},
		{
			name:      "empty name",
			funcName:  "",
			wantError: true,
		},
		{
			name:      "name too long",
			funcName:  "this_is_a_very_long_function_name_that_exceeds_the_maximum_length_limit_of_64_characters",
			wantError: true,
		},
		{
			name:      "reserved name - dot",
			funcName:  ".",
			wantError: true,
		},
		{
			name:      "reserved name - double dot",
			funcName:  "..",
			wantError: true,
		},
		{
			name:      "reserved name - index",
			funcName:  "index",
			wantError: true,
		},
		{
			name:      "reserved name - main",
			funcName:  "main",
			wantError: true,
		},
		{
			name:      "reserved name - handler",
			funcName:  "handler",
			wantError: true,
		},
		{
			name:      "name with forward slash (path traversal)",
			funcName:  "../malicious",
			wantError: true,
		},
		{
			name:      "name with backslash (path traversal)",
			funcName:  "..\\malicious",
			wantError: true,
		},
		{
			name:      "name with special characters",
			funcName:  "my@function",
			wantError: true,
		},
		{
			name:      "name with spaces",
			funcName:  "my function",
			wantError: true,
		},
		{
			name:      "name with dots",
			funcName:  "my.function",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFunctionName(tt.funcName)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateFunctionName(%q) error = %v, wantError %v", tt.funcName, err, tt.wantError)
			}
		})
	}
}

func TestValidateFunctionPath(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "functions-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name         string
		functionsDir string
		functionName string
		wantError    bool
	}{
		{
			name:         "valid function path",
			functionsDir: tmpDir,
			functionName: "test-function",
			wantError:    false,
		},
		{
			name:         "invalid function name",
			functionsDir: tmpDir,
			functionName: "../traversal",
			wantError:    true,
		},
		{
			name:         "empty function name",
			functionsDir: tmpDir,
			functionName: "",
			wantError:    true,
		},
		{
			name:         "reserved function name",
			functionsDir: tmpDir,
			functionName: ".",
			wantError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, err := ValidateFunctionPath(tt.functionsDir, tt.functionName)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateFunctionPath() error = %v, wantError %v", err, tt.wantError)
			}
			if !tt.wantError && path == "" {
				t.Error("ValidateFunctionPath() returned empty path for valid input")
			}
			if !tt.wantError {
				// Verify path is within functions directory
				absDir, _ := filepath.Abs(tt.functionsDir)
				if !filepath.HasPrefix(path, absDir) {
					t.Errorf("ValidateFunctionPath() returned path outside functions directory: %s", path)
				}
			}
		})
	}
}

func TestValidateFunctionCode(t *testing.T) {
	tests := []struct {
		name      string
		code      string
		wantError bool
	}{
		{
			name:      "valid code",
			code:      "async function handler(req) { return { status: 200 }; }",
			wantError: false,
		},
		{
			name:      "empty code",
			code:      "",
			wantError: true,
		},
		{
			name:      "code too large",
			code:      string(make([]byte, 2*1024*1024)), // 2MB
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFunctionCode(tt.code)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateFunctionCode() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestValidateFunctionPathTraversal(t *testing.T) {
	// Create a temporary directory structure for testing path traversal
	tmpDir, err := os.MkdirTemp("", "functions-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	functionsDir := filepath.Join(tmpDir, "functions")
	if err := os.MkdirAll(functionsDir, 0755); err != nil {
		t.Fatalf("Failed to create functions dir: %v", err)
	}

	// Create a file outside the functions directory
	outsideFile := filepath.Join(tmpDir, "outside.ts")
	if err := os.WriteFile(outsideFile, []byte("malicious"), 0644); err != nil {
		t.Fatalf("Failed to create outside file: %v", err)
	}

	// Attempt to access the file outside using path traversal
	// This should fail validation
	_, err = ValidateFunctionPath(functionsDir, ".."+string(filepath.Separator)+"outside")
	if err == nil {
		t.Error("ValidateFunctionPath() should have rejected path traversal attempt")
	}
}
