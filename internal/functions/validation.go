package functions

import (
	"fmt"
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
func ValidateFunctionPath(functionsDir, functionName string) (string, error) {
	// First validate the function name
	if err := ValidateFunctionName(functionName); err != nil {
		return "", err
	}

	// Construct the path
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
