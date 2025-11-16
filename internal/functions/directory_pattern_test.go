package functions

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestDirectoryBasedFunctions tests the directory-based function pattern
func TestDirectoryBasedFunctions(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "test-dir-functions-")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create flat file function
	flatCode := `async function handler(req) { return { status: 200, body: "flat" }; }`
	err = os.WriteFile(filepath.Join(tmpDir, "flat-func.ts"), []byte(flatCode), 0644)
	require.NoError(t, err)

	// Create directory-based function
	dirPath := filepath.Join(tmpDir, "dir-func")
	err = os.Mkdir(dirPath, 0755)
	require.NoError(t, err)
	dirCode := `async function handler(req) { return { status: 200, body: "directory" }; }`
	err = os.WriteFile(filepath.Join(dirPath, "index.ts"), []byte(dirCode), 0644)
	require.NoError(t, err)

	// Test: List all functions
	funcs, err := ListFunctionFiles(tmpDir)
	require.NoError(t, err)
	require.Len(t, funcs, 2, "Should find both flat and directory-based functions")

	// Verify function names
	funcNames := make(map[string]bool)
	for _, f := range funcs {
		funcNames[f.Name] = true
	}
	require.True(t, funcNames["flat-func"], "Should find flat-func")
	require.True(t, funcNames["dir-func"], "Should find dir-func")

	// Test: Load flat file function
	flatLoadedCode, err := LoadFunctionCode(tmpDir, "flat-func")
	require.NoError(t, err)
	require.Equal(t, flatCode, flatLoadedCode, "Flat function code should match")

	// Test: Load directory-based function
	dirLoadedCode, err := LoadFunctionCode(tmpDir, "dir-func")
	require.NoError(t, err)
	require.Equal(t, dirCode, dirLoadedCode, "Directory function code should match")

	// Test: FunctionExists for both patterns
	exists, err := FunctionExists(tmpDir, "flat-func")
	require.NoError(t, err)
	require.True(t, exists, "FunctionExists should return true for flat file")

	exists, err = FunctionExists(tmpDir, "dir-func")
	require.NoError(t, err)
	require.True(t, exists, "FunctionExists should return true for directory-based function")

	exists, err = FunctionExists(tmpDir, "nonexistent")
	require.NoError(t, err)
	require.False(t, exists, "FunctionExists should return false for nonexistent function")
}

// TestFlatFilePriority tests that flat files take precedence over directory-based functions
func TestFlatFilePriority(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "test-priority-")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	functionName := "priority"

	// Create directory-based version
	dirPath := filepath.Join(tmpDir, functionName)
	err = os.Mkdir(dirPath, 0755)
	require.NoError(t, err)
	dirCode := `async function handler(req) { return { status: 200, body: "directory" }; }`
	err = os.WriteFile(filepath.Join(dirPath, "index.ts"), []byte(dirCode), 0644)
	require.NoError(t, err)

	// Create flat file version (should take priority)
	flatCode := `async function handler(req) { return { status: 200, body: "flat" }; }`
	err = os.WriteFile(filepath.Join(tmpDir, functionName+".ts"), []byte(flatCode), 0644)
	require.NoError(t, err)

	// Test: List functions should only return one
	funcs, err := ListFunctionFiles(tmpDir)
	require.NoError(t, err)
	require.Len(t, funcs, 1, "Should only find one function when both patterns exist")
	require.Equal(t, functionName, funcs[0].Name)

	// Test: Load should return flat file code
	loadedCode, err := LoadFunctionCode(tmpDir, functionName)
	require.NoError(t, err)
	require.Equal(t, flatCode, loadedCode, "Should load flat file when both patterns exist")

	// Verify the path is the flat file
	expectedFlatPath := filepath.Join(tmpDir, functionName+".ts")
	require.Contains(t, funcs[0].Path, functionName+".ts", "Path should be flat file")
	absExpectedPath, _ := filepath.Abs(expectedFlatPath)
	require.Equal(t, absExpectedPath, funcs[0].Path, "Should use flat file path")
}

// TestDirectoryWithoutIndexTs tests that directories without index.ts are skipped
func TestDirectoryWithoutIndexTs(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test-no-index-")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create directory without index.ts
	dirPath := filepath.Join(tmpDir, "no-index")
	err = os.Mkdir(dirPath, 0755)
	require.NoError(t, err)

	// Create some other file in the directory
	err = os.WriteFile(filepath.Join(dirPath, "helper.ts"), []byte("// helper"), 0644)
	require.NoError(t, err)

	// Test: Should not find any functions
	funcs, err := ListFunctionFiles(tmpDir)
	require.NoError(t, err)
	require.Len(t, funcs, 0, "Should not find directory without index.ts")

	// Test: FunctionExists should return false
	exists, err := FunctionExists(tmpDir, "no-index")
	require.NoError(t, err)
	require.False(t, exists, "Should return false for directory without index.ts")
}

// TestResolveFunctionPath tests the ResolveFunctionPath function
func TestResolveFunctionPath(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test-resolve-")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create flat file
	flatCode := "flat"
	err = os.WriteFile(filepath.Join(tmpDir, "flat.ts"), []byte(flatCode), 0644)
	require.NoError(t, err)

	// Create directory-based function
	dirPath := filepath.Join(tmpDir, "dir")
	err = os.Mkdir(dirPath, 0755)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(dirPath, "index.ts"), []byte("dir"), 0644)
	require.NoError(t, err)

	// Test: Resolve flat file
	path, err := ResolveFunctionPath(tmpDir, "flat")
	require.NoError(t, err)
	require.Contains(t, path, "flat.ts")

	// Test: Resolve directory-based
	path, err = ResolveFunctionPath(tmpDir, "dir")
	require.NoError(t, err)
	require.Contains(t, path, "index.ts")

	// Test: Nonexistent function
	_, err = ResolveFunctionPath(tmpDir, "nonexistent")
	require.Error(t, err)
	require.Contains(t, err.Error(), "could not resolve function path")
}
