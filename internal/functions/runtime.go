package functions

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

// DenoRuntime manages execution of Deno-based edge functions
type DenoRuntime struct {
	denoPath string
	timeout  time.Duration
}

// NewDenoRuntime creates a new Deno runtime manager
func NewDenoRuntime() *DenoRuntime {
	// Auto-detect Deno path
	denoPath, err := exec.LookPath("deno")
	if err != nil {
		// Try common installation paths
		paths := []string{
			"/home/vscode/.deno/bin/deno",
			"/usr/local/bin/deno",
			"/usr/bin/deno",
			"$HOME/.deno/bin/deno",
		}
		for _, path := range paths {
			if _, err := exec.LookPath(path); err == nil {
				denoPath = path
				break
			}
		}
	}

	return &DenoRuntime{
		denoPath: denoPath,
		timeout:  30 * time.Second,
	}
}

// ExecutionRequest represents a function invocation request
type ExecutionRequest struct {
	Method    string            `json:"method"`
	URL       string            `json:"url"`
	Headers   map[string]string `json:"headers"`
	Body      string            `json:"body"`
	Params    map[string]string `json:"params"`
	UserID    string            `json:"user_id,omitempty"`
	UserEmail string            `json:"user_email,omitempty"`
	UserRole  string            `json:"user_role,omitempty"`
	SessionID string            `json:"session_id,omitempty"`
}

// ExecutionResult represents the output of a function execution
type ExecutionResult struct {
	Status     int               `json:"status"`
	Headers    map[string]string `json:"headers"`
	Body       string            `json:"body"`
	Logs       string            `json:"logs"`
	Error      string            `json:"error,omitempty"`
	DurationMs int64             `json:"duration_ms"`
}

// Execute runs a Deno function with the given code and request context
func (r *DenoRuntime) Execute(ctx context.Context, code string, req ExecutionRequest, permissions Permissions) (*ExecutionResult, error) {
	start := time.Now()

	// Create context with timeout
	execCtx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	// Wrap the user code with our runtime bridge
	wrappedCode := r.wrapCode(code, req)

	// Write code to temporary file to allow Deno to properly handle TypeScript
	// Using stdin doesn't work well with TypeScript type annotations
	tmpFile, err := os.CreateTemp("", "function-exec-*.ts")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath) // Clean up temp file after execution

	if _, err := tmpFile.WriteString(wrappedCode); err != nil {
		tmpFile.Close()
		return nil, fmt.Errorf("failed to write code to temp file: %w", err)
	}
	tmpFile.Close()

	// Build Deno command
	args := []string{"run"}

	// Apply permissions
	if permissions.AllowNet {
		args = append(args, "--allow-net")
	}
	if permissions.AllowEnv {
		args = append(args, "--allow-env")
	}
	if permissions.AllowRead {
		args = append(args, "--allow-read")
	}
	if permissions.AllowWrite {
		args = append(args, "--allow-write")
	}

	// Run from temp file instead of stdin
	args = append(args, tmpPath)

	// Create command
	cmd := exec.CommandContext(execCtx, r.denoPath, args...)

	// Pass environment variables to the Deno subprocess
	// This allows edge functions to access FLUXBASE_* environment variables
	// (with sensitive credentials filtered out by buildEnvForFunction)
	cmd.Env = buildEnvForFunction()

	// Capture stdout and stderr
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Execute
	err = cmd.Run()

	duration := time.Since(start)

	// Parse result
	result := &ExecutionResult{
		Logs:       stderr.String(),
		DurationMs: duration.Milliseconds(),
	}

	// Check for timeout
	if execCtx.Err() == context.DeadlineExceeded {
		result.Status = 504
		result.Error = "Function execution timeout"
		log.Warn().
			Str("error_type", "execution_timeout").
			Int64("timeout_ms", r.timeout.Milliseconds()).
			Int64("duration_ms", duration.Milliseconds()).
			Str("stderr", stderr.String()).
			Msg("Edge function execution timeout")
		return result, fmt.Errorf("execution timeout after %v", r.timeout)
	}

	// Check for execution errors
	if err != nil {
		result.Status = 500
		result.Error = fmt.Sprintf("Function execution failed: %v", err)
		result.Logs += "\n" + err.Error()
		log.Error().
			Err(err).
			Str("error_type", "deno_execution_failure").
			Str("stderr", stderr.String()).
			Str("stdout", stdout.String()).
			Int64("duration_ms", duration.Milliseconds()).
			Msg("Deno runtime execution failed")
		return result, err
	}

	// Parse function output (JSON response from stdout)
	outputStr := stdout.String()
	if outputStr == "" {
		result.Status = 200
		result.Body = ""
		return result, nil
	}

	// Try to parse as JSON response
	var response struct {
		Status  int               `json:"status"`
		Headers map[string]string `json:"headers"`
		Body    string            `json:"body"`
	}

	if err := json.Unmarshal([]byte(outputStr), &response); err != nil {
		// If not valid JSON response format, treat as plain text
		result.Status = 200
		result.Body = outputStr
		return result, nil
	}

	result.Status = response.Status
	result.Headers = response.Headers
	result.Body = response.Body

	return result, nil
}

// wrapCode wraps user code with runtime bridge
func (r *DenoRuntime) wrapCode(userCode string, req ExecutionRequest) string {
	reqJSON, _ := json.Marshal(req)

	// Extract import/export statements from user code
	// These must be at top level for Deno modules
	imports, codeWithoutImports := extractImports(userCode)

	return fmt.Sprintf(`
// Fluxbase Edge Function Runtime Bridge
%s

// User function code (imports extracted)
%s

// Execute handler
(async () => {
  try {
    // Redirect console.log to console.error so user logs go to stderr
    // This keeps stdout clean for the JSON response only
    const originalLog = console.log;
    console.log = console.error;

    // Parse request from environment
    const request = %s;

    // Set up Fluxbase API URL (if needed by user code)
    const FLUXBASE_URL = Deno.env.get("FLUXBASE_BASE_URL") || "http://localhost:8080";
    const FLUXBASE_TOKEN = Deno.env.get("FLUXBASE_TOKEN") || "";

    // If handler function exists, call it
    if (typeof handler === 'function') {
      const result = await handler(request);

      // Normalize response
      let response = result;
      if (typeof result === 'object' && !result.status) {
        response = {
          status: 200,
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(result)
        };
      }

      // Restore console.log and output response as JSON to stdout
      console.log = originalLog;
      console.log(JSON.stringify(response));
    } else {
      throw new Error("No 'handler' function exported");
    }
  } catch (error) {
    // Output error
    const errorResponse = {
      status: 500,
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ error: error.message, stack: error.stack })
    };
    console.log(JSON.stringify(errorResponse));
  }
})();
`, imports, codeWithoutImports, string(reqJSON))
}

// extractImports separates import/export statements from the rest of the code
// Import statements must be at the top level in ES modules
func extractImports(code string) (imports string, remaining string) {
	lines := strings.Split(code, "\n")
	var importLines []string
	var codeLines []string

	inMultilineDeclaration := false
	braceCount := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Check if we're starting a multi-line type/interface declaration
		if !inMultilineDeclaration &&
			(strings.HasPrefix(trimmed, "export type ") ||
				strings.HasPrefix(trimmed, "export interface ") ||
				strings.HasPrefix(trimmed, "export enum ")) {
			inMultilineDeclaration = true
			braceCount = 0
			importLines = append(importLines, line)
			// Count braces in this line
			braceCount += strings.Count(line, "{") - strings.Count(line, "}")
			if braceCount == 0 {
				// Single-line declaration
				inMultilineDeclaration = false
			}
			continue
		}

		// If we're in a multi-line declaration, continue collecting lines
		if inMultilineDeclaration {
			importLines = append(importLines, line)
			braceCount += strings.Count(line, "{") - strings.Count(line, "}")
			if braceCount == 0 {
				inMultilineDeclaration = false
			}
			continue
		}

		// Extract single-line import/export statements
		if strings.HasPrefix(trimmed, "import ") ||
			strings.HasPrefix(trimmed, "import{") ||
			(strings.HasPrefix(trimmed, "export ") &&
				(strings.HasPrefix(trimmed, "export {") ||
					strings.HasPrefix(trimmed, "export * "))) {
			importLines = append(importLines, line)
		} else {
			codeLines = append(codeLines, line)
		}
	}

	return strings.Join(importLines, "\n"), strings.Join(codeLines, "\n")
}

// Permissions represents Deno security permissions
type Permissions struct {
	AllowNet   bool
	AllowEnv   bool
	AllowRead  bool
	AllowWrite bool
}

// DefaultPermissions returns safe default permissions
func DefaultPermissions() Permissions {
	return Permissions{
		AllowNet:   true,
		AllowEnv:   true,
		AllowRead:  false,
		AllowWrite: false,
	}
}

// buildEnvForFunction creates the environment variable list for edge functions
// This includes all FLUXBASE_* variables except sensitive credentials
func buildEnvForFunction() []string {
	env := []string{}

	// Secrets that should NEVER be passed to edge functions
	// These could allow complete system compromise if exposed
	blockedVars := map[string]bool{
		"FLUXBASE_AUTH_JWT_SECRET":         true, // Master secret for JWT signing - would allow token forgery
		"FLUXBASE_DATABASE_PASSWORD":       true, // Direct DB access bypasses all security
		"FLUXBASE_DATABASE_ADMIN_PASSWORD": true, // Admin DB access
		"FLUXBASE_STORAGE_S3_SECRET_KEY":   true, // S3 credentials
		"FLUXBASE_STORAGE_S3_ACCESS_KEY":   true, // S3 credentials
		"FLUXBASE_EMAIL_SMTP_PASSWORD":     true, // Email credentials
		"FLUXBASE_SECURITY_SETUP_TOKEN":    true, // Initial setup token
	}

	// Pass all FLUXBASE_* environment variables except blocked ones
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, "FLUXBASE_") {
			// Extract the key name
			parts := strings.SplitN(e, "=", 2)
			if len(parts) == 2 {
				key := parts[0]
				// Only add if not blocked
				if !blockedVars[key] {
					env = append(env, e)
				} else {
					log.Debug().
						Str("key", key).
						Msg("Blocking sensitive environment variable from edge function")
				}
			}
		}
	}

	log.Debug().
		Int("env_var_count", len(env)).
		Msg("Prepared environment variables for edge function")

	return env
}
