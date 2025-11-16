package functions

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/rs/zerolog/log"
)

// LoadFunctionCode loads function code from the filesystem
// Supports both flat files ({name}.ts) and directory-based ({name}/index.ts) patterns
func LoadFunctionCode(functionsDir, functionName string) (string, error) {
	// Resolve which pattern exists (priority: flat file > directory-based)
	functionPath, err := ResolveFunctionPath(functionsDir, functionName)
	if err != nil {
		return "", fmt.Errorf("invalid function path: %w", err)
	}

	// Check if file exists
	if _, err := os.Stat(functionPath); os.IsNotExist(err) {
		return "", fmt.Errorf("function file not found: %s", functionName)
	}

	// Read the file
	code, err := os.ReadFile(functionPath)
	if err != nil {
		return "", fmt.Errorf("failed to read function file: %w", err)
	}

	return string(code), nil
}

// SaveFunctionCode saves function code to the filesystem
func SaveFunctionCode(functionsDir, functionName, code string) error {
	// Validate function name and code
	if err := ValidateFunctionName(functionName); err != nil {
		return fmt.Errorf("invalid function name: %w", err)
	}

	if err := ValidateFunctionCode(code); err != nil {
		return fmt.Errorf("invalid function code: %w", err)
	}

	// Validate and get safe path
	functionPath, err := ValidateFunctionPath(functionsDir, functionName)
	if err != nil {
		return fmt.Errorf("invalid function path: %w", err)
	}

	// Ensure functions directory exists
	if err := os.MkdirAll(functionsDir, 0755); err != nil {
		return fmt.Errorf("failed to create functions directory: %w", err)
	}

	// Write the file with appropriate permissions (read/write for owner, read for others)
	if err := os.WriteFile(functionPath, []byte(code), 0644); err != nil {
		return fmt.Errorf("failed to write function file: %w", err)
	}

	log.Info().
		Str("function", functionName).
		Str("path", functionPath).
		Msg("Function code saved to filesystem")

	return nil
}

// DeleteFunctionCode removes a function file from the filesystem
// Supports both flat files ({name}.ts) and directory-based ({name}/index.ts) patterns
func DeleteFunctionCode(functionsDir, functionName string) error {
	// Resolve which pattern exists (priority: flat file > directory-based)
	functionPath, err := ResolveFunctionPath(functionsDir, functionName)
	if err != nil {
		return fmt.Errorf("invalid function path: %w", err)
	}

	// Check if file exists before attempting to delete
	if _, err := os.Stat(functionPath); os.IsNotExist(err) {
		return fmt.Errorf("function file not found: %s", functionName)
	}

	// Delete the file
	if err := os.Remove(functionPath); err != nil {
		return fmt.Errorf("failed to delete function file: %w", err)
	}

	log.Info().
		Str("function", functionName).
		Str("path", functionPath).
		Msg("Function code deleted from filesystem")

	return nil
}

// FunctionFileInfo contains information about a function file
type FunctionFileInfo struct {
	Name         string // Function name (without .ts extension)
	Path         string // Full path to the file
	Size         int64  // File size in bytes
	ModifiedTime int64  // Unix timestamp of last modification
}

// ListFunctionFiles scans the functions directory and returns all function files
func ListFunctionFiles(functionsDir string) ([]FunctionFileInfo, error) {
	// Check if directory exists
	if _, err := os.Stat(functionsDir); os.IsNotExist(err) {
		// Directory doesn't exist yet - return empty list
		log.Debug().Str("dir", functionsDir).Msg("Functions directory does not exist yet")
		return []FunctionFileInfo{}, nil
	}

	// Read directory entries
	entries, err := os.ReadDir(functionsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read functions directory: %w", err)
	}

	var functions []FunctionFileInfo
	processedNames := make(map[string]bool) // Track which function names we've seen

	// First pass: Process flat .ts files (these have priority)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".ts") {
			log.Debug().Str("file", name).Msg("Skipping non-.ts file in functions directory")
			continue
		}

		// Extract function name (remove .ts extension)
		functionName := strings.TrimSuffix(name, ".ts")

		// Validate function name
		if err := ValidateFunctionName(functionName); err != nil {
			log.Warn().
				Str("file", name).
				Err(err).
				Msg("Skipping file with invalid function name")
			continue
		}

		// Get file info
		fullPath := filepath.Join(functionsDir, name)
		info, err := os.Stat(fullPath)
		if err != nil {
			log.Warn().
				Str("file", name).
				Err(err).
				Msg("Failed to get file info")
			continue
		}

		functions = append(functions, FunctionFileInfo{
			Name:         functionName,
			Path:         fullPath,
			Size:         info.Size(),
			ModifiedTime: info.ModTime().Unix(),
		})
		processedNames[functionName] = true
	}

	// Second pass: Process directories with index.ts
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		functionName := entry.Name()

		// Skip if we already have a flat file for this function name
		if processedNames[functionName] {
			log.Debug().
				Str("function", functionName).
				Msg("Skipping directory-based function (flat file takes precedence)")
			continue
		}

		// Check if directory contains index.ts
		indexPath := filepath.Join(functionsDir, functionName, "index.ts")
		indexInfo, err := os.Stat(indexPath)
		if err != nil {
			if !os.IsNotExist(err) {
				log.Warn().
					Str("directory", functionName).
					Err(err).
					Msg("Failed to check for index.ts")
			}
			continue
		}

		// Validate function name
		if err := ValidateFunctionName(functionName); err != nil {
			log.Warn().
				Str("directory", functionName).
				Err(err).
				Msg("Skipping directory with invalid function name")
			continue
		}

		functions = append(functions, FunctionFileInfo{
			Name:         functionName,
			Path:         indexPath,
			Size:         indexInfo.Size(),
			ModifiedTime: indexInfo.ModTime().Unix(),
		})
		processedNames[functionName] = true
	}

	log.Debug().
		Str("dir", functionsDir).
		Int("count", len(functions)).
		Msg("Scanned functions directory")

	return functions, nil
}

// FunctionExists checks if a function file exists in the filesystem
// Supports both flat files ({name}.ts) and directory-based ({name}/index.ts) patterns
func FunctionExists(functionsDir, functionName string) (bool, error) {
	// Validate function name first
	if err := ValidateFunctionName(functionName); err != nil {
		return false, fmt.Errorf("invalid function name: %w", err)
	}

	// Check flat file pattern first ({name}.ts)
	flatPath := filepath.Join(functionsDir, functionName+".ts")
	if _, err := os.Stat(flatPath); err == nil {
		return true, nil
	}

	// Check directory-based pattern ({name}/index.ts)
	dirPath := filepath.Join(functionsDir, functionName, "index.ts")
	if _, err := os.Stat(dirPath); err == nil {
		return true, nil
	}

	// Neither pattern exists
	return false, nil
}

// FunctionConfig contains configuration parsed from function code comments
type FunctionConfig struct {
	AllowUnauthenticated bool
}

// ParseFunctionConfig parses special @fluxbase directives from function code comments
// Supported directives:
//   - // @fluxbase:allow-unauthenticated - Allows function invocation without authentication
func ParseFunctionConfig(code string) FunctionConfig {
	config := FunctionConfig{
		AllowUnauthenticated: false, // Secure by default
	}

	// Regex to match @fluxbase directives in comments
	// Matches: // @fluxbase:allow-unauthenticated or /* @fluxbase:allow-unauthenticated */
	allowUnauthPattern := regexp.MustCompile(`(?m)^\s*(?://|/\*)\s*@fluxbase:allow-unauthenticated`)

	if allowUnauthPattern.MatchString(code) {
		config.AllowUnauthenticated = true
		log.Debug().Msg("Found @fluxbase:allow-unauthenticated directive in function code")
	}

	return config
}
