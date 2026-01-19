// Package bundler provides client-side bundling for edge functions and jobs.
package bundler

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// dataFileExtensions lists file extensions that should be inlined as JSON data
// These are data files that esbuild/deno bundle don't have built-in loaders for
var dataFileExtensions = []string{".geojson"}

// Bundler handles bundling edge functions with npm dependencies
type Bundler struct {
	denoPath  string
	sourceDir string // Original source directory for resolving relative imports
}

// BundleResult contains the result of a bundling operation
type BundleResult struct {
	BundledCode  string
	OriginalCode string
	IsBundled    bool
	Error        string
}

// BlockedPackages contains npm packages that are not allowed for security reasons
var BlockedPackages = []string{
	"child_process",
	"node:child_process",
	"vm",
	"node:vm",
	"fs",
	"node:fs",
	"process",
	"node:process",
}

// NewBundler creates a new bundler instance.
// Returns an error if Deno is not installed.
// sourceDir is the original directory containing the source files, used for resolving relative imports.
func NewBundler(sourceDir string) (*Bundler, error) {
	// Try to find Deno executable
	denoPath, err := exec.LookPath("deno")
	if err != nil {
		// Try common installation paths
		commonPaths := []string{
			"/usr/local/bin/deno",
			"/usr/bin/deno",
			"/opt/homebrew/bin/deno",
			"/home/linuxbrew/.linuxbrew/bin/deno",
		}

		// Also check user home directory
		if home, err := os.UserHomeDir(); err == nil {
			commonPaths = append(commonPaths, home+"/.deno/bin/deno")
		}

		for _, path := range commonPaths {
			if _, err := os.Stat(path); err == nil {
				denoPath = path
				break
			}
		}

		if denoPath == "" {
			return nil, fmt.Errorf("deno is required for bundling functions with imports. Install from https://deno.land")
		}
	}

	return &Bundler{
		denoPath:  denoPath,
		sourceDir: sourceDir,
	}, nil
}

// NeedsBundle checks if code contains import statements that require bundling
func (b *Bundler) NeedsBundle(code string) bool {
	// Match various import patterns:
	// - import { foo } from "package"
	// - import foo from "package"
	// - import * as foo from "package"
	// - import "package"
	// - import type { foo } from "package" (TypeScript)
	importRegex := regexp.MustCompile(`(?m)^\s*import\s+(?:type\s+)?(?:[\w\s{},*]+\s+from\s+)?['"]`)
	return importRegex.MatchString(code)
}

// ValidateImports checks for dangerous imports before bundling
func (b *Bundler) ValidateImports(code string) error {
	for _, blocked := range BlockedPackages {
		// Check for npm: specifier
		if strings.Contains(code, fmt.Sprintf(`"npm:%s"`, blocked)) ||
			strings.Contains(code, fmt.Sprintf(`'npm:%s'`, blocked)) {
			return fmt.Errorf("import 'npm:%s' is not allowed for security reasons", blocked)
		}

		// Check for node: specifier
		if strings.Contains(code, fmt.Sprintf(`"%s"`, blocked)) && strings.HasPrefix(blocked, "node:") {
			return fmt.Errorf("import '%s' is not allowed for security reasons", blocked)
		}
	}
	return nil
}

// knownDenoBuiltins contains modules that Deno provides natively
var knownDenoBuiltins = map[string]bool{
	"Deno": true,
}

// transformBareImports converts bare npm imports to npm: specifiers
// and _shared/ imports to relative paths for esbuild compatibility.
// e.g., import JSZip from "jszip" -> import JSZip from "npm:jszip"
// e.g., import x from "_shared/utils" -> import x from "./_shared/utils"
func transformBareImports(code string) string {
	// Match import statements with bare specifiers (not starting with . / http npm: jsr: node:)
	// Captures: full match, quote char, module name
	importRegex := regexp.MustCompile(`(import\s+(?:type\s+)?(?:[\w\s{},*]+\s+from\s+)?['"])([^'"./][^'"]*?)(['"])`)

	return importRegex.ReplaceAllStringFunc(code, func(match string) string {
		submatch := importRegex.FindStringSubmatch(match)
		if len(submatch) != 4 {
			return match
		}

		prefix := submatch[1]
		moduleName := submatch[2]
		suffix := submatch[3]

		// Skip if already has a protocol prefix
		if strings.HasPrefix(moduleName, "npm:") ||
			strings.HasPrefix(moduleName, "jsr:") ||
			strings.HasPrefix(moduleName, "node:") ||
			strings.HasPrefix(moduleName, "http:") ||
			strings.HasPrefix(moduleName, "https:") {
			return match
		}

		// Skip Deno builtins
		if knownDenoBuiltins[moduleName] {
			return match
		}

		// Transform _shared/ imports to relative paths (esbuild compatibility)
		if strings.HasPrefix(moduleName, "_shared/") {
			return prefix + "./" + moduleName + suffix
		}

		// Transform to npm: specifier
		return prefix + "npm:" + moduleName + suffix
	})
}

// Bundle bundles TypeScript/JavaScript code with dependencies into a single file.
// sharedModules is a map of module paths (e.g., "_shared/cors.ts") to their content.
func (b *Bundler) Bundle(ctx context.Context, code string, sharedModules map[string]string) (*BundleResult, error) {
	result := &BundleResult{
		OriginalCode: code,
	}

	// Extract data files (.geojson, etc.) from shared modules before processing
	// These need to be inlined because deno bundle/esbuild don't have loaders for them
	dataFiles := extractDataFiles(sharedModules)
	if len(dataFiles) > 0 {
		// Inline data file imports in main code (main file is at root, use empty path)
		code = inlineDataFiles(code, "", dataFiles)

		// Also inline in shared modules (code files only)
		for path, content := range sharedModules {
			if !isDataFile(path) {
				sharedModules[path] = inlineDataFiles(content, path, dataFiles)
			}
		}
	}

	// Transform bare npm imports to npm: specifiers
	code = transformBareImports(code)

	// Validate imports for security
	if err := b.ValidateImports(code); err != nil {
		result.Error = err.Error()
		return nil, err
	}

	// Also validate and transform shared modules (skip data files)
	transformedSharedModules := make(map[string]string, len(sharedModules))
	for path, content := range sharedModules {
		if isDataFile(path) {
			// Skip data files - they've been inlined
			continue
		}
		content = transformBareImports(content)
		if err := b.ValidateImports(content); err != nil {
			result.Error = err.Error()
			return nil, err
		}
		transformedSharedModules[path] = content
	}
	sharedModules = transformedSharedModules

	// Always use a temp directory for bundling to ensure we can write transformed files
	// This is necessary because shared modules need bare imports transformed to npm: specifiers
	tmpDir, err := os.MkdirTemp("", "fluxbase-bundle-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	workDir := tmpDir

	// Write main file to working directory
	mainPath := filepath.Join(workDir, ".fluxbase-bundle-entry.ts")
	if err := os.WriteFile(mainPath, []byte(code), 0600); err != nil { //nolint:gosec // temp file for bundling
		return nil, fmt.Errorf("failed to write main file: %w", err)
	}

	// Write all shared modules with transformed imports
	if len(sharedModules) > 0 {
		for modulePath, content := range sharedModules {
			// Ensure path starts with _shared/
			if !strings.HasPrefix(modulePath, "_shared/") {
				modulePath = "_shared/" + modulePath
			}
			fullPath := filepath.Join(workDir, modulePath)

			// Create parent directory if needed
			dir := filepath.Dir(fullPath)
			if err := os.MkdirAll(dir, 0750); err != nil { //nolint:gosec // temp directory for bundling
				return nil, fmt.Errorf("failed to create directory for %s: %w", modulePath, err)
			}

			if err := os.WriteFile(fullPath, []byte(content), 0600); err != nil { //nolint:gosec // temp file for bundling
				return nil, fmt.Errorf("failed to write shared module %s: %w", modulePath, err)
			}
		}
	}

	// Set timeout for bundling (30 seconds)
	bundleCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Build deno bundle command (native bundler using esbuild internally)
	// Deno bundle automatically handles external imports (npm:*, https://, jsr:, etc.)
	// and supports JSON/GeoJSON imports natively
	// Output to stdout for maximum compatibility across Deno versions (1.x and 2.x)
	args := []string{
		"bundle",
		"--quiet",
		mainPath,
	}

	// Note: deno bundle automatically marks npm:*, https://* etc. as external imports.
	// The bundled code runs on the server where Deno can resolve those directly at runtime.

	// Run deno bundle command
	cmd := exec.CommandContext(bundleCtx, b.denoPath, args...) //nolint:gosec // denoPath is validated in NewBundler
	cmd.Dir = workDir

	// Build environment for Deno, ensuring DENO_DIR and HOME are set correctly
	// Filter out existing DENO_DIR/HOME and add them explicitly at the end
	cmd.Env = filterEnvVars(os.Environ(), "DENO_DIR", "HOME")
	denoDir := os.Getenv("DENO_DIR")
	if denoDir == "" {
		denoDir = "/tmp/deno"
	}
	home := os.Getenv("HOME")
	if home == "" {
		home = "/tmp"
	}
	cmd.Env = append(cmd.Env, "DENO_DIR="+denoDir, "HOME="+home)

	// Capture stdout and stderr
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	runErr := cmd.Run()

	// Check for timeout
	if bundleCtx.Err() == context.DeadlineExceeded {
		result.Error = "bundling timeout after 30s - package may be too large or network issue"
		return nil, fmt.Errorf("bundling timeout: %s", result.Error)
	}

	// Check for bundling errors
	if runErr != nil {
		errMsg := stderr.String()
		if errMsg == "" {
			errMsg = stdout.String()
		}
		if errMsg == "" {
			errMsg = runErr.Error()
		}

		// Clean up error message
		result.Error = cleanBundleError(errMsg)
		return nil, fmt.Errorf("bundle failed: %s", result.Error)
	}

	// Get bundled output from stdout
	result.BundledCode = stdout.String()
	result.IsBundled = true

	// Validate bundled size (50MB limit - allows for embedded GeoJSON data)
	if len(result.BundledCode) > 50*1024*1024 {
		result.Error = fmt.Sprintf("bundled code exceeds 50MB limit (got %d bytes)", len(result.BundledCode))
		return nil, fmt.Errorf("%s", result.Error)
	}

	return result, nil
}

// cleanBundleError cleans up error messages for better user experience
func cleanBundleError(errMsg string) string {
	// Remove file paths from temp files
	errMsg = regexp.MustCompile(`/tmp/fluxbase-bundle-[a-zA-Z0-9]+/`).ReplaceAllString(errMsg, "")
	// Remove references to our temporary entry file
	errMsg = strings.ReplaceAll(errMsg, ".fluxbase-bundle-entry.ts", "<entry>")

	// Extract the most relevant error message
	lines := strings.Split(errMsg, "\n")
	var relevantLines []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Include error messages but skip noise
		if strings.Contains(line, "error:") ||
			strings.Contains(line, "Module not found") ||
			strings.Contains(line, "Expected") ||
			strings.Contains(line, "Unexpected") ||
			strings.Contains(line, "Could not resolve") {
			relevantLines = append(relevantLines, line)
		}
	}

	if len(relevantLines) > 0 {
		return strings.Join(relevantLines, "\n")
	}

	// Fallback to full error if we couldn't extract anything
	return errMsg
}

// filterEnvVars returns a copy of env with the specified variable names removed
func filterEnvVars(env []string, names ...string) []string {
	result := make([]string, 0, len(env))
	for _, e := range env {
		skip := false
		for _, name := range names {
			if strings.HasPrefix(e, name+"=") {
				skip = true
				break
			}
		}
		if !skip {
			result = append(result, e)
		}
	}
	return result
}

// isDataFile checks if a file path has a data file extension
func isDataFile(path string) bool {
	for _, ext := range dataFileExtensions {
		if strings.HasSuffix(path, ext) {
			return true
		}
	}
	return false
}

// extractDataFiles collects all data files (.geojson, etc.) from shared modules
func extractDataFiles(sharedModules map[string]string) map[string]string {
	dataFiles := make(map[string]string)
	for path, content := range sharedModules {
		if isDataFile(path) {
			dataFiles[path] = content
		}
	}
	return dataFiles
}

// lookupDataFile finds a data file by path, trying various path normalizations
// sourceFilePath is the path of the file containing the import statement (used to resolve relative paths)
func lookupDataFile(importPath string, sourceFilePath string, dataFiles map[string]string) string {
	// If it's a relative path (starts with . or ..), resolve it relative to the source file
	if strings.HasPrefix(importPath, "./") || strings.HasPrefix(importPath, "../") {
		// Get the directory of the source file
		sourceDir := filepath.Dir(sourceFilePath)
		// Join and clean the path to resolve .. sequences
		resolvedPath := filepath.Clean(filepath.Join(sourceDir, importPath))
		// Convert back to forward slashes
		resolvedPath = filepath.ToSlash(resolvedPath)

		// Try the resolved path
		if content, ok := dataFiles[resolvedPath]; ok {
			return content
		}
		// Also try with _shared/ prefix if not already there
		if !strings.HasPrefix(resolvedPath, "_shared/") {
			if content, ok := dataFiles["_shared/"+resolvedPath]; ok {
				return content
			}
		}
	}

	// Try exact path
	if content, ok := dataFiles[importPath]; ok {
		return content
	}
	// Try with ./ prefix stripped
	cleanPath := strings.TrimPrefix(importPath, "./")
	if content, ok := dataFiles[cleanPath]; ok {
		return content
	}
	// Try with _shared/ prefix added
	if !strings.HasPrefix(cleanPath, "_shared/") {
		if content, ok := dataFiles["_shared/"+cleanPath]; ok {
			return content
		}
	}
	// Try with _shared/ prefix stripped
	if strings.HasPrefix(cleanPath, "_shared/") {
		stripped := strings.TrimPrefix(cleanPath, "_shared/")
		if content, ok := dataFiles[stripped]; ok {
			return content
		}
	}
	return ""
}

// buildDataFilePattern builds a regex pattern to match imports of data file extensions
func buildDataFilePattern() *regexp.Regexp {
	// Build alternation for extensions: (\.geojson)
	var extPatterns []string
	for _, ext := range dataFileExtensions {
		// Escape the dot for regex
		extPatterns = append(extPatterns, regexp.QuoteMeta(ext))
	}
	extAlternation := strings.Join(extPatterns, "|")

	// Match: import varName from './path/file.geojson' or "_shared/data/file.geojson"
	// Captures: (1) variable name, (2) full file path
	pattern := fmt.Sprintf(`import\s+(\w+)\s+from\s+['"]([^'"]+(?:%s))['"]`, extAlternation)
	return regexp.MustCompile(pattern)
}

// inlineDataFiles inlines data file imports (.geojson, etc.) by replacing them with const declarations
// This is needed because deno bundle/esbuild don't have loaders for these file types
// sourceFilePath is the path of the file being processed (used to resolve relative imports)
func inlineDataFiles(code string, sourceFilePath string, dataFiles map[string]string) string {
	if len(dataFiles) == 0 {
		return code
	}

	importRegex := buildDataFilePattern()

	return importRegex.ReplaceAllStringFunc(code, func(match string) string {
		matches := importRegex.FindStringSubmatch(match)
		if len(matches) < 3 {
			return match
		}
		varName := matches[1]
		importPath := matches[2]

		// Look up the data file content
		content := lookupDataFile(importPath, sourceFilePath, dataFiles)
		if content == "" {
			// File not found - leave import as-is, will fail at bundle time with clear error
			return match
		}

		return fmt.Sprintf("const %s = %s", varName, content)
	})
}
