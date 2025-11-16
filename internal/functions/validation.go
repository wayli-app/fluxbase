package functions

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	// Only allow alphanumeric characters, hyphens, and underscores
	validFunctionNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

	// Reserved names that cannot be used
	reservedNames = map[string]bool{
		".":       true,
		"..":      true,
		"index":   true,
		"main":    true,
		"handler": true,
		"_":       true,
		"-":       true,
	}
)

// ValidateFunctionName validates that a function name is safe and meets requirements
func ValidateFunctionName(name string) error {
	if name == "" {
		return fmt.Errorf("function name cannot be empty")
	}

	// Check length (max 64 characters for reasonable filesystem limits)
	if len(name) > 64 {
		return fmt.Errorf("function name too long (max 64 characters), got %d", len(name))
	}

	// Check for reserved names
	if reservedNames[name] {
		return fmt.Errorf("function name '%s' is reserved", name)
	}

	// Only allow alphanumeric, hyphen, and underscore to prevent path traversal
	if !validFunctionNameRegex.MatchString(name) {
		return fmt.Errorf("function name must contain only letters, numbers, hyphens, and underscores (got: %s)", name)
	}

	// Additional safety check: ensure no path separators
	if strings.Contains(name, "/") || strings.Contains(name, "\\") {
		return fmt.Errorf("function name cannot contain path separators")
	}

	return nil
}

// ValidateFunctionPath validates that a constructed function path is safe
// This provides defense-in-depth by ensuring the final path is within the functions directory
// Note: This always returns the flat file pattern ({name}.ts) for writing operations
func ValidateFunctionPath(functionsDir, functionName string) (string, error) {
	// First validate the function name
	if err := ValidateFunctionName(functionName); err != nil {
		return "", err
	}

	// Construct the path (flat file pattern)
	functionPath := filepath.Join(functionsDir, functionName+".ts")

	// Get absolute path of functions directory
	absFunctionsDir, err := filepath.Abs(functionsDir)
	if err != nil {
		return "", fmt.Errorf("failed to resolve functions directory: %w", err)
	}

	// Get absolute path of the function file
	absFunctionPath, err := filepath.Abs(functionPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve function path: %w", err)
	}

	// Ensure the function path is within the functions directory (prevent path traversal)
	if !strings.HasPrefix(absFunctionPath, absFunctionsDir+string(filepath.Separator)) {
		return "", fmt.Errorf("function path is outside of functions directory (potential path traversal attack)")
	}

	return absFunctionPath, nil
}

// ResolveFunctionPath resolves which function pattern exists (flat file or directory-based)
// Priority: {name}.ts takes precedence over {name}/index.ts
// This is used for reading operations to support both patterns
func ResolveFunctionPath(functionsDir, functionName string) (string, error) {
	// First validate the function name
	if err := ValidateFunctionName(functionName); err != nil {
		return "", err
	}

	// Get absolute path of functions directory for validation
	absFunctionsDir, err := filepath.Abs(functionsDir)
	if err != nil {
		return "", fmt.Errorf("failed to resolve functions directory: %w", err)
	}

	// Helper to validate a path is within functionsDir and exists
	validatePath := func(path string) (string, error) {
		absPath, err := filepath.Abs(path)
		if err != nil {
			return "", fmt.Errorf("failed to resolve function path: %w", err)
		}

		// Ensure the function path is within the functions directory
		if !strings.HasPrefix(absPath, absFunctionsDir+string(filepath.Separator)) {
			return "", fmt.Errorf("function path is outside of functions directory")
		}

		// Check if file exists
		if _, err := os.Stat(absPath); err != nil {
			return "", err
		}

		return absPath, nil
	}

	// Check pattern 1: flat file ({name}.ts) - has priority
	flatPath := filepath.Join(functionsDir, functionName+".ts")
	if absPath, err := validatePath(flatPath); err == nil {
		return absPath, nil
	}

	// Check pattern 2: directory-based ({name}/index.ts)
	dirPath := filepath.Join(functionsDir, functionName, "index.ts")
	if absPath, err := validatePath(dirPath); err == nil {
		return absPath, nil
	}

	// Neither pattern is valid
	return "", fmt.Errorf("could not resolve function path for: %s", functionName)
}

// ValidateFunctionCode performs basic validation on function code
func ValidateFunctionCode(code string) error {
	if code == "" {
		return fmt.Errorf("function code cannot be empty")
	}

	// Check for reasonable size limit (e.g., 1MB for function code)
	maxCodeSize := 1024 * 1024 // 1MB
	if len(code) > maxCodeSize {
		return fmt.Errorf("function code too large (max %d bytes), got %d", maxCodeSize, len(code))
	}

	return nil
}
