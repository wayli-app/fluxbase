package functions

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
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

// LoadFunctionCodeWithFiles loads function code and supporting files from the filesystem
// For flat files ({name}.ts): Returns main code only
// For directory-based ({name}/index.ts): Returns main code + all other .ts files in the directory
func LoadFunctionCodeWithFiles(functionsDir, functionName string) (mainCode string, supportingFiles map[string]string, err error) {
	supportingFiles = make(map[string]string)

	// Resolve which pattern exists (priority: flat file > directory-based)
	functionPath, err := ResolveFunctionPath(functionsDir, functionName)
	if err != nil {
		return "", nil, fmt.Errorf("invalid function path: %w", err)
	}

	// Check if file exists
	if _, err := os.Stat(functionPath); os.IsNotExist(err) {
		return "", nil, fmt.Errorf("function file not found: %s", functionName)
	}

	// Read the main file
	mainCodeBytes, err := os.ReadFile(functionPath)
	if err != nil {
		return "", nil, fmt.Errorf("failed to read function file: %w", err)
	}
	mainCode = string(mainCodeBytes)

	// Check if this is a directory-based function
	functionDir := filepath.Join(functionsDir, functionName)
	dirInfo, err := os.Stat(functionDir)
	if err != nil || !dirInfo.IsDir() {
		// Flat file pattern - no supporting files
		return mainCode, supportingFiles, nil
	}

	// Directory-based pattern - scan for supporting files
	entries, err := os.ReadDir(functionDir)
	if err != nil {
		log.Warn().
			Str("function", functionName).
			Err(err).
			Msg("Failed to read function directory for supporting files")
		return mainCode, supportingFiles, nil
	}

	// Read all .ts files except index.ts
	for _, entry := range entries {
		if entry.IsDir() {
			// Handle nested directories (e.g., utils/, helpers/)
			nestedFiles, err := loadNestedFiles(filepath.Join(functionDir, entry.Name()), entry.Name())
			if err != nil {
				log.Warn().
					Str("function", functionName).
					Str("dir", entry.Name()).
					Err(err).
					Msg("Failed to read nested directory")
				continue
			}
			// Merge nested files into supporting files
			for path, content := range nestedFiles {
				supportingFiles[path] = content
			}
			continue
		}

		fileName := entry.Name()

		// Skip index.ts (that's the main file)
		if fileName == "index.ts" {
			continue
		}

		// Process TypeScript/JavaScript files and deno.json
		isCodeFile := strings.HasSuffix(fileName, ".ts") || strings.HasSuffix(fileName, ".js") ||
			strings.HasSuffix(fileName, ".mts") || strings.HasSuffix(fileName, ".mjs")
		isDenoConfig := fileName == "deno.json" || fileName == "deno.jsonc"

		if !isCodeFile && !isDenoConfig {
			continue
		}

		// Read the supporting file
		filePath := filepath.Join(functionDir, fileName)
		content, err := os.ReadFile(filePath)
		if err != nil {
			log.Warn().
				Str("function", functionName).
				Str("file", fileName).
				Err(err).
				Msg("Failed to read supporting file")
			continue
		}

		supportingFiles[fileName] = string(content)
		log.Debug().
			Str("function", functionName).
			Str("file", fileName).
			Msg("Loaded supporting file")
	}

	return mainCode, supportingFiles, nil
}

// loadNestedFiles recursively loads .ts files from a nested directory
func loadNestedFiles(dirPath, relativePath string) (map[string]string, error) {
	files := make(map[string]string)

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			// Recursively handle nested directories
			nestedPath := filepath.Join(relativePath, entry.Name())
			nestedFiles, err := loadNestedFiles(filepath.Join(dirPath, entry.Name()), nestedPath)
			if err != nil {
				log.Warn().
					Str("dir", nestedPath).
					Err(err).
					Msg("Failed to read nested directory")
				continue
			}
			for path, content := range nestedFiles {
				files[path] = content
			}
			continue
		}

		fileName := entry.Name()

		// Only process TypeScript/JavaScript files
		if !strings.HasSuffix(fileName, ".ts") && !strings.HasSuffix(fileName, ".js") &&
			!strings.HasSuffix(fileName, ".mts") && !strings.HasSuffix(fileName, ".mjs") {
			continue
		}

		filePath := filepath.Join(dirPath, fileName)
		content, err := os.ReadFile(filePath)
		if err != nil {
			log.Warn().
				Str("file", fileName).
				Err(err).
				Msg("Failed to read nested file")
			continue
		}

		// Store with relative path (e.g., "utils/db.ts")
		relativeFilePath := filepath.Join(relativePath, fileName)
		files[relativeFilePath] = string(content)
		log.Debug().
			Str("file", relativeFilePath).
			Msg("Loaded nested file")
	}

	return files, nil
}

// LoadSharedModulesFromFilesystem loads all shared modules from the _shared/ directory
// Returns a map of module paths (e.g., "_shared/cors.ts") to content
func LoadSharedModulesFromFilesystem(functionsDir string) (map[string]string, error) {
	sharedModules := make(map[string]string)

	// Check if _shared directory exists
	sharedDir := filepath.Join(functionsDir, "_shared")
	dirInfo, err := os.Stat(sharedDir)
	if err != nil {
		if os.IsNotExist(err) {
			// No _shared directory - return empty map
			log.Debug().
				Str("dir", functionsDir).
				Msg("No _shared directory found")
			return sharedModules, nil
		}
		return nil, fmt.Errorf("failed to stat _shared directory: %w", err)
	}

	if !dirInfo.IsDir() {
		return nil, fmt.Errorf("_shared exists but is not a directory")
	}

	// Recursively load all .ts files from _shared/
	files, err := loadNestedFiles(sharedDir, "_shared")
	if err != nil {
		return nil, fmt.Errorf("failed to load shared modules: %w", err)
	}

	log.Info().
		Int("count", len(files)).
		Msg("Loaded shared modules from filesystem")

	return files, nil
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
	IsPublic             bool
	// CORS configuration (nil means use global defaults)
	CorsOrigins     *string
	CorsMethods     *string
	CorsHeaders     *string
	CorsCredentials *bool
	CorsMaxAge      *int
}

// ParseFunctionConfig parses special @fluxbase directives from function code comments
// Supported directives:
//   - // @fluxbase:allow-unauthenticated - Allows function invocation without authentication
//   - // @fluxbase:public [true|false] - Controls whether function is publicly listed (default: true)
//   - // @fluxbase:cors-origins <origins> - Comma-separated list of allowed origins
//   - // @fluxbase:cors-methods <methods> - Comma-separated list of allowed HTTP methods
//   - // @fluxbase:cors-headers <headers> - Comma-separated list of allowed headers
//   - // @fluxbase:cors-credentials <true|false> - Allow credentials (cookies, auth headers)
//   - // @fluxbase:cors-max-age <seconds> - Max age for preflight cache in seconds
func ParseFunctionConfig(code string) FunctionConfig {
	config := FunctionConfig{
		AllowUnauthenticated: false, // Secure by default
		IsPublic:             true,  // Public by default
	}

	// Regex to match @fluxbase directives in comments
	// Matches: // @fluxbase:allow-unauthenticated or /* @fluxbase:allow-unauthenticated */ or * @fluxbase:allow-unauthenticated (inside multi-line comments)
	allowUnauthPattern := regexp.MustCompile(`(?m)^\s*(?://|/\*|\*)\s*@fluxbase:allow-unauthenticated`)

	if allowUnauthPattern.MatchString(code) {
		config.AllowUnauthenticated = true
		log.Debug().Msg("Found @fluxbase:allow-unauthenticated directive in function code")
	}

	// Match @fluxbase:public with optional true/false value
	// Matches: // @fluxbase:public false or // @fluxbase:public true or just // @fluxbase:public
	publicPattern := regexp.MustCompile(`(?m)^\s*(?://|/\*|\*)\s*@fluxbase:public(?:\s+(true|false))?`)

	if matches := publicPattern.FindStringSubmatch(code); matches != nil {
		if len(matches) > 1 && matches[1] == "false" {
			config.IsPublic = false
			log.Debug().Msg("Found @fluxbase:public false directive in function code")
		} else {
			config.IsPublic = true
			log.Debug().Msg("Found @fluxbase:public directive in function code")
		}
	}

	// Parse CORS annotations
	// Match @fluxbase:cors-origins with value
	corsOriginsPattern := regexp.MustCompile(`(?m)^\s*(?://|/\*|\*)\s*@fluxbase:cors-origins\s+(.+?)\s*$`)
	if matches := corsOriginsPattern.FindStringSubmatch(code); matches != nil && len(matches) > 1 {
		value := strings.TrimSpace(matches[1])
		config.CorsOrigins = &value
		log.Debug().Str("origins", value).Msg("Found @fluxbase:cors-origins directive")
	}

	// Match @fluxbase:cors-methods with value
	corsMethodsPattern := regexp.MustCompile(`(?m)^\s*(?://|/\*|\*)\s*@fluxbase:cors-methods\s+(.+?)\s*$`)
	if matches := corsMethodsPattern.FindStringSubmatch(code); matches != nil && len(matches) > 1 {
		value := strings.TrimSpace(matches[1])
		config.CorsMethods = &value
		log.Debug().Str("methods", value).Msg("Found @fluxbase:cors-methods directive")
	}

	// Match @fluxbase:cors-headers with value
	corsHeadersPattern := regexp.MustCompile(`(?m)^\s*(?://|/\*|\*)\s*@fluxbase:cors-headers\s+(.+?)\s*$`)
	if matches := corsHeadersPattern.FindStringSubmatch(code); matches != nil && len(matches) > 1 {
		value := strings.TrimSpace(matches[1])
		config.CorsHeaders = &value
		log.Debug().Str("headers", value).Msg("Found @fluxbase:cors-headers directive")
	}

	// Match @fluxbase:cors-credentials with boolean value
	corsCredentialsPattern := regexp.MustCompile(`(?m)^\s*(?://|/\*|\*)\s*@fluxbase:cors-credentials\s+(true|false)\s*$`)
	if matches := corsCredentialsPattern.FindStringSubmatch(code); matches != nil && len(matches) > 1 {
		value := matches[1] == "true"
		config.CorsCredentials = &value
		log.Debug().Bool("credentials", value).Msg("Found @fluxbase:cors-credentials directive")
	}

	// Match @fluxbase:cors-max-age with integer value
	corsMaxAgePattern := regexp.MustCompile(`(?m)^\s*(?://|/\*|\*)\s*@fluxbase:cors-max-age\s+(\d+)\s*$`)
	if matches := corsMaxAgePattern.FindStringSubmatch(code); matches != nil && len(matches) > 1 {
		if value, err := strconv.Atoi(matches[1]); err == nil {
			config.CorsMaxAge = &value
			log.Debug().Int("max_age", value).Msg("Found @fluxbase:cors-max-age directive")
		}
	}

	return config
}
