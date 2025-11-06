package functions

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFunctionCode(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "functions-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test function file
	testCode := "async function handler(req) { return { status: 200 }; }"
	testFunctionName := "test-function"
	testFilePath := filepath.Join(tmpDir, testFunctionName+".ts")
	if err := os.WriteFile(testFilePath, []byte(testCode), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tests := []struct {
		name         string
		functionsDir string
		functionName string
		wantError    bool
		wantCode     string
	}{
		{
			name:         "load existing function",
			functionsDir: tmpDir,
			functionName: testFunctionName,
			wantError:    false,
			wantCode:     testCode,
		},
		{
			name:         "load non-existent function",
			functionsDir: tmpDir,
			functionName: "non-existent",
			wantError:    true,
			wantCode:     "",
		},
		{
			name:         "invalid function name",
			functionsDir: tmpDir,
			functionName: "../traversal",
			wantError:    true,
			wantCode:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, err := LoadFunctionCode(tt.functionsDir, tt.functionName)
			if (err != nil) != tt.wantError {
				t.Errorf("LoadFunctionCode() error = %v, wantError %v", err, tt.wantError)
			}
			if code != tt.wantCode {
				t.Errorf("LoadFunctionCode() code = %q, want %q", code, tt.wantCode)
			}
		})
	}
}

func TestSaveFunctionCode(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "functions-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	testCode := "async function handler(req) { return { status: 200 }; }"

	tests := []struct {
		name         string
		functionsDir string
		functionName string
		code         string
		wantError    bool
	}{
		{
			name:         "save valid function",
			functionsDir: tmpDir,
			functionName: "test-function",
			code:         testCode,
			wantError:    false,
		},
		{
			name:         "save with invalid name",
			functionsDir: tmpDir,
			functionName: "../traversal",
			code:         testCode,
			wantError:    true,
		},
		{
			name:         "save with empty code",
			functionsDir: tmpDir,
			functionName: "empty-function",
			code:         "",
			wantError:    true,
		},
		{
			name:         "save with code too large",
			functionsDir: tmpDir,
			functionName: "large-function",
			code:         string(make([]byte, 2*1024*1024)), // 2MB
			wantError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := SaveFunctionCode(tt.functionsDir, tt.functionName, tt.code)
			if (err != nil) != tt.wantError {
				t.Errorf("SaveFunctionCode() error = %v, wantError %v", err, tt.wantError)
			}

			if !tt.wantError {
				// Verify file was created and contains correct code
				filePath := filepath.Join(tt.functionsDir, tt.functionName+".ts")
				savedCode, err := os.ReadFile(filePath)
				if err != nil {
					t.Errorf("Failed to read saved file: %v", err)
				}
				if string(savedCode) != tt.code {
					t.Errorf("SaveFunctionCode() saved code = %q, want %q", string(savedCode), tt.code)
				}
			}
		})
	}
}

func TestDeleteFunctionCode(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "functions-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test function file to delete
	testFunctionName := "test-function"
	testFilePath := filepath.Join(tmpDir, testFunctionName+".ts")
	if err := os.WriteFile(testFilePath, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tests := []struct {
		name         string
		functionsDir string
		functionName string
		wantError    bool
	}{
		{
			name:         "delete existing function",
			functionsDir: tmpDir,
			functionName: testFunctionName,
			wantError:    false,
		},
		{
			name:         "delete non-existent function",
			functionsDir: tmpDir,
			functionName: "non-existent",
			wantError:    true,
		},
		{
			name:         "delete with invalid name",
			functionsDir: tmpDir,
			functionName: "../traversal",
			wantError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := DeleteFunctionCode(tt.functionsDir, tt.functionName)
			if (err != nil) != tt.wantError {
				t.Errorf("DeleteFunctionCode() error = %v, wantError %v", err, tt.wantError)
			}

			if !tt.wantError {
				// Verify file was deleted
				filePath := filepath.Join(tt.functionsDir, tt.functionName+".ts")
				if _, err := os.Stat(filePath); !os.IsNotExist(err) {
					t.Error("DeleteFunctionCode() did not delete the file")
				}
			}
		})
	}
}

func TestListFunctionFiles(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "functions-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test function files
	testFunctions := []string{"function1", "function2", "function-3"}
	for _, name := range testFunctions {
		filePath := filepath.Join(tmpDir, name+".ts")
		if err := os.WriteFile(filePath, []byte("test"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	// Create a non-.ts file (should be ignored)
	nonTsFile := filepath.Join(tmpDir, "readme.md")
	if err := os.WriteFile(nonTsFile, []byte("readme"), 0644); err != nil {
		t.Fatalf("Failed to create non-ts file: %v", err)
	}

	// Create a subdirectory (should be ignored)
	subDir := filepath.Join(tmpDir, "subdir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	// Create a file with invalid name (should be skipped)
	invalidFile := filepath.Join(tmpDir, "../invalid.ts")
	if err := os.WriteFile(invalidFile, []byte("test"), 0644); err == nil {
		// Only if we successfully created it (might fail due to path issues)
		defer os.Remove(invalidFile)
	}

	tests := []struct {
		name         string
		functionsDir string
		wantCount    int
		wantError    bool
	}{
		{
			name:         "list existing functions",
			functionsDir: tmpDir,
			wantCount:    len(testFunctions),
			wantError:    false,
		},
		{
			name:         "list from non-existent directory",
			functionsDir: filepath.Join(tmpDir, "non-existent"),
			wantCount:    0,
			wantError:    false, // Should return empty list, not error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			functions, err := ListFunctionFiles(tt.functionsDir)
			if (err != nil) != tt.wantError {
				t.Errorf("ListFunctionFiles() error = %v, wantError %v", err, tt.wantError)
			}
			if len(functions) != tt.wantCount {
				t.Errorf("ListFunctionFiles() returned %d functions, want %d", len(functions), tt.wantCount)
			}

			if !tt.wantError {
				// Verify function info is correct
				for _, fn := range functions {
					if fn.Name == "" {
						t.Error("ListFunctionFiles() returned function with empty name")
					}
					if fn.Path == "" {
						t.Error("ListFunctionFiles() returned function with empty path")
					}
					if fn.Size <= 0 {
						t.Error("ListFunctionFiles() returned function with invalid size")
					}
					if fn.ModifiedTime <= 0 {
						t.Error("ListFunctionFiles() returned function with invalid modified time")
					}
				}
			}
		})
	}
}

func TestFunctionExists(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "functions-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test function file
	testFunctionName := "test-function"
	testFilePath := filepath.Join(tmpDir, testFunctionName+".ts")
	if err := os.WriteFile(testFilePath, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tests := []struct {
		name         string
		functionsDir string
		functionName string
		wantExists   bool
		wantError    bool
	}{
		{
			name:         "existing function",
			functionsDir: tmpDir,
			functionName: testFunctionName,
			wantExists:   true,
			wantError:    false,
		},
		{
			name:         "non-existent function",
			functionsDir: tmpDir,
			functionName: "non-existent",
			wantExists:   false,
			wantError:    false,
		},
		{
			name:         "invalid function name",
			functionsDir: tmpDir,
			functionName: "../traversal",
			wantExists:   false,
			wantError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exists, err := FunctionExists(tt.functionsDir, tt.functionName)
			if (err != nil) != tt.wantError {
				t.Errorf("FunctionExists() error = %v, wantError %v", err, tt.wantError)
			}
			if exists != tt.wantExists {
				t.Errorf("FunctionExists() = %v, want %v", exists, tt.wantExists)
			}
		})
	}
}

func TestSaveAndLoadFunctionCodeIntegration(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "functions-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	testCode := `async function handler(req) {
	const data = JSON.parse(req.body || "{}");
	return {
		status: 200,
		headers: { "Content-Type": "application/json" },
		body: JSON.stringify({ message: "Hello " + data.name })
	};
}`

	functionName := "hello-world"

	// Save function code
	if err := SaveFunctionCode(tmpDir, functionName, testCode); err != nil {
		t.Fatalf("SaveFunctionCode() failed: %v", err)
	}

	// Load function code
	loadedCode, err := LoadFunctionCode(tmpDir, functionName)
	if err != nil {
		t.Fatalf("LoadFunctionCode() failed: %v", err)
	}

	if loadedCode != testCode {
		t.Errorf("Loaded code does not match saved code.\nGot: %q\nWant: %q", loadedCode, testCode)
	}

	// Verify function exists
	exists, err := FunctionExists(tmpDir, functionName)
	if err != nil {
		t.Fatalf("FunctionExists() failed: %v", err)
	}
	if !exists {
		t.Error("Function should exist after saving")
	}

	// List functions
	functions, err := ListFunctionFiles(tmpDir)
	if err != nil {
		t.Fatalf("ListFunctionFiles() failed: %v", err)
	}
	if len(functions) != 1 {
		t.Fatalf("ListFunctionFiles() returned %d functions, want 1", len(functions))
	}
	if functions[0].Name != functionName {
		t.Errorf("ListFunctionFiles() returned function with name %q, want %q", functions[0].Name, functionName)
	}

	// Delete function code
	if err := DeleteFunctionCode(tmpDir, functionName); err != nil {
		t.Fatalf("DeleteFunctionCode() failed: %v", err)
	}

	// Verify function no longer exists
	exists, err = FunctionExists(tmpDir, functionName)
	if err != nil {
		t.Fatalf("FunctionExists() failed after delete: %v", err)
	}
	if exists {
		t.Error("Function should not exist after deletion")
	}
}
