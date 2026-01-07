package functions

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// esbuildCacheOnce ensures esbuild is downloaded only once to avoid race conditions
// when multiple bundling operations run in parallel
var esbuildCacheOnce sync.Once
var esbuildCacheErr error

// esbuildVersion is the version of esbuild used for bundling
const esbuildVersion = "0.24.0"

// ensureEsbuildCached downloads and caches esbuild if not already cached.
// This must be called before parallel bundling operations to avoid race conditions.
func ensureEsbuildCached(denoPath string) error {
	esbuildCacheOnce.Do(func() {
		log.Debug().Msg("Pre-caching esbuild to avoid parallel download race conditions")

		// Build environment with DENO_DIR set
		env := filterEnvVars(os.Environ(), "DENO_DIR", "HOME")
		denoDir := os.Getenv("DENO_DIR")
		if denoDir == "" {
			denoDir = "/tmp/deno"
		}
		home := os.Getenv("HOME")
		if home == "" {
			home = "/tmp"
		}
		env = append(env, "DENO_DIR="+denoDir, "HOME="+home)

		// Use deno cache to download esbuild
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, denoPath, "cache", "npm:esbuild@"+esbuildVersion)
		cmd.Env = env

		output, err := cmd.CombinedOutput()
		if err != nil {
			esbuildCacheErr = fmt.Errorf("failed to cache esbuild: %w: %s", err, string(output))
			log.Error().Err(esbuildCacheErr).Msg("Failed to pre-cache esbuild")
			return
		}

		log.Info().Str("version", esbuildVersion).Msg("Successfully pre-cached esbuild")
	})

	return esbuildCacheErr
}

// Bundler handles bundling edge functions with npm dependencies
type Bundler struct {
	denoPath            string
	globalDenoConfig    string // Path to global deno.json
	globalDenoConfigDev string // Path to global deno.dev.json
}

// NewBundler creates a new bundler instance
func NewBundler() (*Bundler, error) {
	// Try to find Deno executable
	denoPath, err := exec.LookPath("deno")
	if err != nil {
		// Try common installation paths (system paths first, then user-installed)
		commonPaths := []string{
			"/usr/local/bin/deno",                 // System-wide installation (preferred)
			"/usr/bin/deno",                       // System package manager
			"/opt/homebrew/bin/deno",              // Homebrew on macOS
			"/home/linuxbrew/.linuxbrew/bin/deno", // Linuxbrew
			"/home/vscode/.deno/bin/deno",         // User install.sh (fallback)
		}

		for _, path := range commonPaths {
			if _, err := os.Stat(path); err == nil {
				denoPath = path
				break
			}
		}

		if denoPath == "" {
			return nil, fmt.Errorf("deno executable not found in PATH or common locations")
		}
	}

	return &Bundler{
		denoPath:            denoPath,
		globalDenoConfig:    "/functions/deno.json",
		globalDenoConfigDev: "/functions/deno.dev.json",
	}, nil
}

// getGlobalDenoConfig returns the path to the global deno config file
// Prefers deno.dev.json for development, falls back to deno.json
// Returns empty string if no global config exists
func (b *Bundler) getGlobalDenoConfig() string {
	// Check for development config first
	if _, err := os.Stat(b.globalDenoConfigDev); err == nil {
		log.Debug().Str("config", b.globalDenoConfigDev).Msg("Using development deno config")
		return b.globalDenoConfigDev
	}

	// Fall back to production config
	if _, err := os.Stat(b.globalDenoConfig); err == nil {
		log.Debug().Str("config", b.globalDenoConfig).Msg("Using production deno config")
		return b.globalDenoConfig
	}

	// No global config found
	log.Debug().Msg("No global deno config found")
	return ""
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

// BundleResult contains the result of a bundling operation
type BundleResult struct {
	BundledCode  string
	OriginalCode string
	IsBundled    bool
	Error        string
}

// Bundle bundles TypeScript/JavaScript code with dependencies into a single file
func (b *Bundler) Bundle(ctx context.Context, code string) (*BundleResult, error) {
	result := &BundleResult{
		OriginalCode: code,
	}

	// Check if bundling is needed
	if !b.NeedsBundle(code) {
		// No imports - return code as-is
		result.BundledCode = code
		result.IsBundled = false
		return result, nil
	}

	// Ensure esbuild is cached before bundling to avoid race conditions
	if err := ensureEsbuildCached(b.denoPath); err != nil {
		return nil, err
	}

	// Validate imports for security
	if err := b.ValidateImports(code); err != nil {
		result.Error = err.Error()
		return nil, err
	}

	// Create temporary input file
	inputFile, err := os.CreateTemp("", "function-*.ts")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp input file: %w", err)
	}
	inputPath := inputFile.Name()
	defer func() { _ = os.Remove(inputPath) }()

	// Write code to input file
	if _, err := inputFile.WriteString(code); err != nil {
		_ = inputFile.Close()
		return nil, fmt.Errorf("failed to write code to temp file: %w", err)
	}
	_ = inputFile.Close()

	// Create temporary output file
	outputFile, err := os.CreateTemp("", "bundled-*.js")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp output file: %w", err)
	}
	outputPath := outputFile.Name()
	_ = outputFile.Close()
	defer func() { _ = os.Remove(outputPath) }()

	// Set timeout for bundling (30 seconds)
	bundleCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Build esbuild command via Deno (replaces deprecated deno bundle)
	args := []string{
		"run", "--allow-all", "--quiet", "npm:esbuild@" + esbuildVersion,
		inputPath,
		"--bundle",
		"--format=esm",
		"--platform=neutral",
		"--target=esnext",
		"--outfile=" + outputPath,
		// Mark Deno-specific imports as external - Deno resolves them at runtime
		"--external:npm:*",
		"--external:https://*",
		"--external:http://*",
		"--external:jsr:*",
	}

	// Run esbuild via Deno
	cmd := exec.CommandContext(bundleCtx, b.denoPath, args...)

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

	err = cmd.Run()

	// Check for timeout
	if bundleCtx.Err() == context.DeadlineExceeded {
		result.Error = "Bundling timeout after 30s - package may be too large or network issue"
		return nil, fmt.Errorf("bundling timeout: %s", result.Error)
	}

	// Check for bundling errors
	if err != nil {
		errMsg := stderr.String()
		if errMsg == "" {
			errMsg = stdout.String()
		}
		if errMsg == "" {
			errMsg = err.Error()
		}

		// Clean up error message for better user experience
		result.Error = cleanBundleError(errMsg)
		return nil, fmt.Errorf("bundle failed: %s", result.Error)
	}

	// Read bundled output
	bundled, err := os.ReadFile(outputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read bundled output: %w", err)
	}

	result.BundledCode = string(bundled)
	result.IsBundled = true

	// Validate bundled size (50MB limit - allows for embedded GeoJSON data)
	if len(result.BundledCode) > 50*1024*1024 {
		result.Error = fmt.Sprintf("Bundled code exceeds 50MB limit (got %d bytes)", len(result.BundledCode))
		return nil, fmt.Errorf("%s", result.Error)
	}

	return result, nil
}

// BundleWithFiles bundles code with supporting files and shared modules
// mainCode: The main function code (index.ts)
// supportingFiles: Map of file paths to content (e.g., {"utils.ts": "...", "helpers/db.ts": "..."})
// sharedModules: Map of module paths to content (e.g., {"_shared/cors.ts": "..."})
func (b *Bundler) BundleWithFiles(ctx context.Context, mainCode string, supportingFiles map[string]string, sharedModules map[string]string) (*BundleResult, error) {
	result := &BundleResult{
		OriginalCode: mainCode,
	}

	// Ensure esbuild is cached before bundling to avoid race conditions
	if err := ensureEsbuildCached(b.denoPath); err != nil {
		return nil, err
	}

	// Validate imports for security
	if err := b.ValidateImports(mainCode); err != nil {
		result.Error = err.Error()
		return nil, err
	}

	// Also validate supporting files
	for _, content := range supportingFiles {
		if err := b.ValidateImports(content); err != nil {
			result.Error = err.Error()
			return nil, err
		}
	}

	// Count actual code files (exclude deno.json from count)
	codeFileCount := 0
	hasDenoJSON := false
	for filePath := range supportingFiles {
		if filePath == "deno.json" || filePath == "deno.jsonc" {
			hasDenoJSON = true
		} else {
			codeFileCount++
		}
	}

	// For Deno 2.x compatibility: Instead of using deno bundle (which is broken),
	// we'll inline shared modules directly into the code
	// This creates a single-file bundle manually
	// However, if there's a deno.json, we need to use the full bundling path to preserve import maps
	if len(sharedModules) > 0 && codeFileCount == 0 && !hasDenoJSON {
		// Simple case: only shared modules, no other files, no deno.json
		// We can inline them directly
		inlinedCode := inlineSharedModules(mainCode, sharedModules)

		// Try to bundle the inlined code (will handle npm: imports if any)
		if b.NeedsBundle(inlinedCode) {
			// Has npm imports, need to bundle
			bundleResult, err := b.Bundle(ctx, inlinedCode)
			if err != nil {
				// Bundling failed, but we can still use the inlined code
				result.BundledCode = inlinedCode
				result.IsBundled = true // We manually inlined it
				result.Error = ""       // Clear error, inlining worked
				return result, nil
			}
			return bundleResult, nil
		}

		// No npm imports, inlined code is ready to use
		result.BundledCode = inlinedCode
		result.IsBundled = true
		return result, nil
	}

	// Extract deno.json content before we start (we'll need it for inlining)
	var functionDenoJSON string
	for filePath, content := range supportingFiles {
		if filePath == "deno.json" || filePath == "deno.jsonc" {
			functionDenoJSON = content
			break
		}
	}

	// If function has deno.json with external imports (like @fluxbase/sdk),
	// we need to inline them manually because deno bundle doesn't work with import maps in Deno 2.x
	// Do this BEFORE writing any files so we write the fully inlined code
	if functionDenoJSON != "" {
		// Check if deno.json has npm: or URL imports that require deno bundle
		hasNpmOrUrlImports := strings.Contains(functionDenoJSON, "npm:") ||
			strings.Contains(functionDenoJSON, "https://") ||
			strings.Contains(functionDenoJSON, "http://")

		if hasNpmOrUrlImports {
			// Has npm/URL imports - must use deno bundle with config, can't inline these
			log.Debug().Msg("deno.json has npm/URL imports, will use deno bundle with config")
		} else {
			// Try to inline deno.json imports and shared modules together
			inlinedCode, err := inlineAllImports(mainCode, sharedModules, functionDenoJSON)
			if err != nil {
				// If inlining fails, fall back to original approach
				log.Warn().Err(err).Msg("Failed to inline imports, falling back to deno bundle")
			} else {
				// Successfully inlined, use the inlined code
				log.Info().Str("function", "unknown").Msg("Successfully inlined all imports, skipping deno bundle")
				mainCode = inlinedCode
				// Remove deno.json from supporting files since imports are resolved
				delete(supportingFiles, "deno.json")
				delete(supportingFiles, "deno.jsonc")

				// Return the inlined code directly without bundling
				// (deno bundle doesn't work well with inlined code anyway in Deno 2.x)
				result.BundledCode = mainCode
				result.IsBundled = true
				return result, nil
			}
		}
	}

	// Create temporary directory for function files
	tmpDir, err := os.MkdirTemp("", "function-bundle-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Write main file (index.ts) with possibly inlined code
	mainPath := fmt.Sprintf("%s/index.ts", tmpDir)
	if err := os.WriteFile(mainPath, []byte(mainCode), 0644); err != nil {
		return nil, fmt.Errorf("failed to write main file: %w", err)
	}

	// Write supporting files (excluding deno.json if it was used for inlining)
	for filePath, content := range supportingFiles {
		fullPath := fmt.Sprintf("%s/%s", tmpDir, filePath)

		// Create parent directory if needed
		dir := fullPath[:strings.LastIndex(fullPath, "/")]
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory for %s: %w", filePath, err)
		}

		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			return nil, fmt.Errorf("failed to write file %s: %w", filePath, err)
		}
	}

	// Write shared modules under _shared/ (if not already inlined)
	if len(sharedModules) > 0 {
		sharedDir := fmt.Sprintf("%s/_shared", tmpDir)
		if err := os.MkdirAll(sharedDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create _shared directory: %w", err)
		}

		for modulePath, content := range sharedModules {
			// Remove "_shared/" prefix if present
			cleanPath := strings.TrimPrefix(modulePath, "_shared/")
			fullPath := fmt.Sprintf("%s/_shared/%s", tmpDir, cleanPath)

			// Create parent directory if needed (for nested modules like _shared/utils/db.ts)
			dir := fullPath[:strings.LastIndex(fullPath, "/")]
			if err := os.MkdirAll(dir, 0755); err != nil {
				return nil, fmt.Errorf("failed to create directory for %s: %w", modulePath, err)
			}

			if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
				return nil, fmt.Errorf("failed to write shared module %s: %w", modulePath, err)
			}
		}

		// Create merged deno.json with import maps
		mergedConfig, err := b.mergeDenoConfig(functionDenoJSON, len(sharedModules) > 0)
		if err != nil {
			return nil, fmt.Errorf("failed to merge deno.json: %w", err)
		}

		denoConfigPath := fmt.Sprintf("%s/deno.json", tmpDir)
		if err := os.WriteFile(denoConfigPath, []byte(mergedConfig), 0644); err != nil {
			return nil, fmt.Errorf("failed to write deno.json: %w", err)
		}
	} else if functionDenoJSON != "" {
		// No shared modules, but function has deno.json - write it as-is
		denoConfigPath := fmt.Sprintf("%s/deno.json", tmpDir)
		if err := os.WriteFile(denoConfigPath, []byte(functionDenoJSON), 0644); err != nil {
			return nil, fmt.Errorf("failed to write deno.json: %w", err)
		}
		log.Debug().Str("path", denoConfigPath).Msg("Wrote function deno.json to temp directory")
	} else {
		log.Debug().Msg("No deno.json found for function")
	}

	// Create temporary output file
	outputFile, err := os.CreateTemp("", "bundled-*.js")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp output file: %w", err)
	}
	outputPath := outputFile.Name()
	_ = outputFile.Close()
	defer func() { _ = os.Remove(outputPath) }()

	// Set timeout for bundling (30 seconds)
	bundleCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Build esbuild command via Deno (replaces deprecated deno bundle)
	args := []string{
		"run", "--allow-all", "--quiet", "npm:esbuild@" + esbuildVersion,
		mainPath,
		"--bundle",
		"--format=esm",
		"--platform=neutral",
		"--target=esnext",
		"--outfile=" + outputPath,
		// Mark Deno-specific imports as external - Deno resolves them at runtime
		"--external:npm:*",
		"--external:https://*",
		"--external:http://*",
		"--external:jsr:*",
		// Enable JSON loader for .geojson files (used for embedded geodata)
		"--loader:.geojson=json",
	}

	log.Debug().
		Str("command", b.denoPath).
		Strs("args", args).
		Str("dir", tmpDir).
		Msg("Running esbuild for multi-file bundle")

	// Run esbuild via Deno on the main file (index.ts)
	// esbuild will automatically resolve local imports from the directory
	cmd := exec.CommandContext(bundleCtx, b.denoPath, args...)
	cmd.Dir = tmpDir

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

	err = cmd.Run()

	// Check for timeout
	if bundleCtx.Err() == context.DeadlineExceeded {
		result.Error = "Bundling timeout after 30s - package may be too large or network issue"
		return nil, fmt.Errorf("bundling timeout: %s", result.Error)
	}

	// Check for bundling errors
	if err != nil {
		errMsg := stderr.String()
		if errMsg == "" {
			errMsg = stdout.String()
		}
		if errMsg == "" {
			errMsg = err.Error()
		}

		// Clean up error message
		result.Error = cleanBundleError(errMsg)
		return nil, fmt.Errorf("bundle failed: %s", result.Error)
	}

	// Read bundled output
	bundled, err := os.ReadFile(outputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read bundled output: %w", err)
	}

	result.BundledCode = string(bundled)
	result.IsBundled = true

	// Validate bundled size (50MB limit - allows for embedded GeoJSON data)
	if len(result.BundledCode) > 50*1024*1024 {
		result.Error = fmt.Sprintf("Bundled code exceeds 50MB limit (got %d bytes)", len(result.BundledCode))
		return nil, fmt.Errorf("%s", result.Error)
	}

	return result, nil
}

// inlineSharedModules manually inlines shared module imports into the main code
// This is a workaround for Deno 2.x where deno bundle is broken for multi-file projects
func inlineSharedModules(mainCode string, sharedModules map[string]string) string {
	// Build a map of module imports to their content
	// Format: import { x, y } from "_shared/cors.ts" -> actual cors.ts content

	var result strings.Builder

	// Don't collect external imports from inlined code - they should remain in main code
	// Only inline the shared modules' code without any import statements

	// Second pass: add all shared modules' code (minus imports)
	for modulePath, content := range sharedModules {
		// Clean the path - remove _shared/ prefix if present
		cleanPath := strings.TrimPrefix(modulePath, "_shared/")

		// Write the module content with a comment header
		result.WriteString(fmt.Sprintf("// Inlined from _shared/%s\n", cleanPath))

		// Remove ALL imports from this shared module
		lines := strings.Split(content, "\n")
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)

			// Skip all import statements
			if strings.HasPrefix(trimmed, "import ") {
				continue
			}

			result.WriteString(line)
			result.WriteString("\n")
		}
		result.WriteString("\n")
	}

	// Now add the main code, but only remove _shared/ imports
	// Keep external imports in the main code
	lines := strings.Split(mainCode, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Only skip _shared/ imports
		if strings.HasPrefix(trimmed, "import ") &&
			(strings.Contains(line, "'_shared/") || strings.Contains(line, "\"_shared/")) {
			continue
		}

		result.WriteString(line)
		result.WriteString("\n")
	}

	return result.String()
}

// cleanBundleError cleans up Deno error messages for better user experience
func cleanBundleError(errMsg string) string {
	// Remove file paths from temp files
	errMsg = regexp.MustCompile(`/tmp/function-[a-zA-Z0-9]+\.ts`).ReplaceAllString(errMsg, "function.ts")
	errMsg = regexp.MustCompile(`/tmp/function-bundle-[a-zA-Z0-9]+/`).ReplaceAllString(errMsg, "")

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
			strings.Contains(line, "Unexpected") {
			relevantLines = append(relevantLines, line)
		}
	}

	if len(relevantLines) > 0 {
		return strings.Join(relevantLines, "\n")
	}

	// Fallback to full error if we couldn't extract anything
	return errMsg
}

// mergeDenoConfig merges global deno.json, function's deno.json, and _shared/ import map
func (b *Bundler) mergeDenoConfig(functionDenoJSON string, hasSharedModules bool) (string, error) {
	imports := make(map[string]string)

	// First, load global config if it exists
	globalConfigPath := b.getGlobalDenoConfig()
	if globalConfigPath != "" {
		globalConfigData, err := os.ReadFile(globalConfigPath)
		if err == nil {
			var globalConfig map[string]interface{}
			if err := json.Unmarshal(globalConfigData, &globalConfig); err == nil {
				// Extract imports from global config
				if importsField, ok := globalConfig["imports"].(map[string]interface{}); ok {
					for key, value := range importsField {
						if strValue, ok := value.(string); ok {
							imports[key] = strValue
						}
					}
				}
				log.Debug().Int("count", len(imports)).Msg("Loaded imports from global deno config")
			}
		}
	}

	// Add _shared/ mapping if we have shared modules (overrides global if present)
	if hasSharedModules {
		imports["_shared/"] = "./_shared/"
	}

	// Parse and merge function's deno.json if provided (function config takes precedence)
	if functionDenoJSON != "" {
		var config map[string]interface{}
		if err := json.Unmarshal([]byte(functionDenoJSON), &config); err != nil {
			return "", fmt.Errorf("failed to parse function's deno.json: %w", err)
		}

		// Extract imports from function's config
		if importsField, ok := config["imports"].(map[string]interface{}); ok {
			for key, value := range importsField {
				if strValue, ok := value.(string); ok {
					imports[key] = strValue
				}
			}
		}
	}

	// Build merged config
	mergedConfig := map[string]interface{}{
		"imports": imports,
	}

	// Marshal back to JSON
	configJSON, err := json.MarshalIndent(mergedConfig, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal merged config: %w", err)
	}

	log.Debug().Int("total_imports", len(imports)).Msg("Merged deno config created")
	return string(configJSON), nil
}

// inlineAllImports inlines both shared modules and deno.json import map imports into the main code
// This is needed because deno bundle doesn't work with import maps in Deno 2.x
func inlineAllImports(mainCode string, sharedModules map[string]string, denoJSON string) (string, error) {
	var result strings.Builder
	externalModules := make(map[string]string)

	// Parse deno.json to get import mappings
	if denoJSON != "" {
		var config map[string]interface{}
		if err := json.Unmarshal([]byte(denoJSON), &config); err != nil {
			return "", fmt.Errorf("failed to parse deno.json: %w", err)
		}

		// Extract imports from config
		if importsField, ok := config["imports"].(map[string]interface{}); ok {
			for key, value := range importsField {
				if strValue, ok := value.(string); ok {
					// Skip _shared/ mapping (handled separately)
					if key == "_shared/" {
						continue
					}

					// For absolute paths like /fluxbase-sdk/dist/index.js, read the file
					if strings.HasPrefix(strValue, "/") {
						content, err := os.ReadFile(strValue)
						if err != nil {
							log.Warn().
								Str("import", key).
								Str("path", strValue).
								Err(err).
								Msg("Failed to read external module, skipping inline")
							continue
						}
						externalModules[key] = string(content)
						log.Debug().
							Str("import", key).
							Str("path", strValue).
							Msg("Loaded external module for inlining")
					}
				}
			}
		}
	}

	// First, inline all external modules (like @fluxbase/sdk)
	// Wrap them in IIFEs to avoid scope conflicts and syntax errors
	for moduleName, content := range externalModules {
		result.WriteString(fmt.Sprintf("// Inlined from %s\n", moduleName))

		// Extract export names from the module to create proper variable bindings
		exportNames := extractExportNames(content)

		// Start IIFE wrapper
		result.WriteString("const __external_module = (() => {\n")

		// Remove import/export statements from the external module
		lines := strings.Split(content, "\n")
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)

			// Skip import statements
			if strings.HasPrefix(trimmed, "import ") {
				continue
			}

			// Skip export statements (we'll handle them in the return)
			if strings.HasPrefix(trimmed, "export ") {
				continue
			}

			result.WriteString(line)
			result.WriteString("\n")
		}

		// Return the exports from the IIFE
		if len(exportNames) > 0 {
			result.WriteString("\nreturn { ")
			result.WriteString(strings.Join(exportNames, ", "))
			result.WriteString(" };\n")
		}

		result.WriteString("})();\n")

		// Destructure the exports
		if len(exportNames) > 0 {
			result.WriteString("const { ")
			result.WriteString(strings.Join(exportNames, ", "))
			result.WriteString(" } = __external_module;\n")
		}

		result.WriteString("\n")
	}

	// Second, inline all shared modules
	for modulePath, content := range sharedModules {
		cleanPath := strings.TrimPrefix(modulePath, "_shared/")
		result.WriteString(fmt.Sprintf("// Inlined from _shared/%s\n", cleanPath))

		// Remove ALL import statements from shared modules since all dependencies are being inlined
		lines := strings.Split(content, "\n")
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)

			// Skip all import statements in shared modules
			// Since we're inlining everything, all dependencies should already be available
			if strings.HasPrefix(trimmed, "import ") {
				continue
			}

			result.WriteString(line)
			result.WriteString("\n")
		}
		result.WriteString("\n")
	}

	// Finally, add the main code but skip ALL import statements
	// Since we've already inlined external modules and shared modules,
	// any remaining imports in the main code are for inlined dependencies
	lines := strings.Split(mainCode, "\n")
	inImport := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Check if we're starting an import statement
		if strings.HasPrefix(trimmed, "import ") {
			inImport = true

			// Check if this is a single-line import (contains 'from')
			if strings.Contains(line, " from ") {
				inImport = false
			}
			continue // Skip this line
		}

		// If we're in a multi-line import, skip until we find the end
		if inImport {
			// Check if this is the last line (contains 'from')
			if strings.Contains(line, " from ") {
				inImport = false
			}
			continue // Skip this line
		}

		result.WriteString(line)
		result.WriteString("\n")
	}

	return result.String(), nil
}

// extractExportNames extracts exported names from ES6 export statements
// Handles: export { name1, name2, ... }
func extractExportNames(code string) []string {
	var exportNames []string

	// Regex to match: export { name1, name2, name3 }
	// This handles multi-line exports and various whitespace
	exportRegex := regexp.MustCompile(`export\s*\{\s*([^}]+)\s*\}`)
	matches := exportRegex.FindStringSubmatch(code)

	if len(matches) > 1 {
		// Split by comma and clean up whitespace
		namesStr := matches[1]
		parts := strings.Split(namesStr, ",")

		for _, part := range parts {
			name := strings.TrimSpace(part)
			if name != "" {
				// Handle "name as alias" syntax - we only want the original name
				if strings.Contains(name, " as ") {
					fields := strings.Fields(name)
					if len(fields) >= 3 {
						name = fields[0] // Take the name before "as"
					}
				}
				exportNames = append(exportNames, name)
			}
		}
	}

	return exportNames
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
