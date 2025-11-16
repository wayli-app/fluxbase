package functions

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

// NewBundler creates a new bundler instance
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

	return &Bundler{denoPath: denoPath}, nil
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
	defer os.Remove(inputPath)

	// Write code to input file
	if _, err := inputFile.WriteString(code); err != nil {
		inputFile.Close()
		return nil, fmt.Errorf("failed to write code to temp file: %w", err)
	}
	inputFile.Close()

	// Create temporary output file
	outputFile, err := os.CreateTemp("", "bundled-*.js")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp output file: %w", err)
	}
	outputPath := outputFile.Name()
	outputFile.Close()
	defer os.Remove(outputPath)

	// Set timeout for bundling (30 seconds)
	bundleCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Run deno bundle
	// Note: deno bundle is deprecated in Deno 2.0+ but still works
	cmd := exec.CommandContext(bundleCtx, b.denoPath, "bundle", inputPath, outputPath)

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

	// Validate bundled size (5MB limit)
	if len(result.BundledCode) > 5*1024*1024 {
		result.Error = fmt.Sprintf("Bundled code exceeds 5MB limit (got %d bytes)", len(result.BundledCode))
		return nil, fmt.Errorf("%s", result.Error)
	}

	return result, nil
}

// cleanBundleError cleans up Deno error messages for better user experience
func cleanBundleError(errMsg string) string {
	// Remove file paths from temp files
	errMsg = regexp.MustCompile(`/tmp/function-[a-zA-Z0-9]+\.ts`).ReplaceAllString(errMsg, "function.ts")

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
