// Package bundler provides client-side bundling for edge functions and jobs.
package bundler

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// Bundler handles bundling edge functions with npm dependencies
type Bundler struct {
	denoPath string
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
func NewBundler() (*Bundler, error) {
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
		denoPath: denoPath,
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

// Bundle bundles TypeScript/JavaScript code with dependencies into a single file.
// sharedModules is a map of module paths (e.g., "_shared/cors.ts") to their content.
func (b *Bundler) Bundle(ctx context.Context, code string, sharedModules map[string]string) (*BundleResult, error) {
	result := &BundleResult{
		OriginalCode: code,
	}

	// Validate imports for security
	if err := b.ValidateImports(code); err != nil {
		result.Error = err.Error()
		return nil, err
	}

	// Also validate shared modules
	for _, content := range sharedModules {
		if err := b.ValidateImports(content); err != nil {
			result.Error = err.Error()
			return nil, err
		}
	}

	// Create temporary directory for bundling
	tmpDir, err := os.MkdirTemp("", "fluxbase-bundle-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Write main file
	mainPath := fmt.Sprintf("%s/index.ts", tmpDir)
	if err := os.WriteFile(mainPath, []byte(code), 0600); err != nil { //nolint:gosec // temp file for bundling
		return nil, fmt.Errorf("failed to write main file: %w", err)
	}

	// Write shared modules
	if len(sharedModules) > 0 {
		sharedDir := fmt.Sprintf("%s/_shared", tmpDir)
		if err := os.MkdirAll(sharedDir, 0750); err != nil { //nolint:gosec // temp directory for bundling
			return nil, fmt.Errorf("failed to create _shared directory: %w", err)
		}

		for modulePath, content := range sharedModules {
			// Remove "_shared/" prefix if present
			cleanPath := strings.TrimPrefix(modulePath, "_shared/")
			fullPath := fmt.Sprintf("%s/_shared/%s", tmpDir, cleanPath)

			// Create parent directory if needed
			dir := fullPath[:strings.LastIndex(fullPath, "/")]
			if err := os.MkdirAll(dir, 0750); err != nil { //nolint:gosec // temp directory for bundling
				return nil, fmt.Errorf("failed to create directory for %s: %w", modulePath, err)
			}

			if err := os.WriteFile(fullPath, []byte(content), 0600); err != nil { //nolint:gosec // temp file for bundling
				return nil, fmt.Errorf("failed to write shared module %s: %w", modulePath, err)
			}
		}
	}

	// Create output file
	outputPath := fmt.Sprintf("%s/bundled.js", tmpDir)

	// Set timeout for bundling (30 seconds)
	bundleCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Build esbuild command via Deno
	args := []string{
		"run", "--allow-all", "--quiet", "npm:esbuild@0.24.0",
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
	}

	// Run esbuild via Deno
	cmd := exec.CommandContext(bundleCtx, b.denoPath, args...) //nolint:gosec // denoPath is validated in NewBundler
	cmd.Dir = tmpDir

	// Capture stdout and stderr
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()

	// Check for timeout
	if bundleCtx.Err() == context.DeadlineExceeded {
		result.Error = "bundling timeout after 30s - package may be too large or network issue"
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
	bundled, err := os.ReadFile(outputPath) //nolint:gosec // Reading from temp file we created
	if err != nil {
		return nil, fmt.Errorf("failed to read bundled output: %w", err)
	}

	result.BundledCode = string(bundled)
	result.IsBundled = true

	// Validate bundled size (20MB limit)
	if len(result.BundledCode) > 20*1024*1024 {
		result.Error = fmt.Sprintf("bundled code exceeds 20MB limit (got %d bytes)", len(result.BundledCode))
		return nil, fmt.Errorf("%s", result.Error)
	}

	return result, nil
}

// cleanBundleError cleans up error messages for better user experience
func cleanBundleError(errMsg string) string {
	// Remove file paths from temp files
	errMsg = regexp.MustCompile(`/tmp/fluxbase-bundle-[a-zA-Z0-9]+/`).ReplaceAllString(errMsg, "")

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
