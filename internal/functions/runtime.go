package functions

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
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
	Method  string            `json:"method"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body"`
	UserID  string            `json:"user_id,omitempty"`
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

	// Build Deno command (using 'deno run -' to read from stdin)
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

	// Add '-' to read from stdin
	args = append(args, "-")

	// Create command
	cmd := exec.CommandContext(execCtx, r.denoPath, args...)

	// Pipe the code to stdin
	cmd.Stdin = strings.NewReader(wrappedCode)

	// Capture stdout and stderr
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Execute
	err := cmd.Run()

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
		return result, fmt.Errorf("execution timeout after %v", r.timeout)
	}

	// Check for execution errors
	if err != nil {
		result.Status = 500
		result.Error = fmt.Sprintf("Function execution failed: %v", err)
		result.Logs += "\n" + err.Error()
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

	return fmt.Sprintf(`
// Fluxbase Edge Function Runtime Bridge
(async () => {
  try {
    // Parse request from environment
    const request = %s;

    // Set up Fluxbase API URL (if needed by user code)
    const FLUXBASE_URL = Deno.env.get("FLUXBASE_URL") || "http://localhost:8080";
    const FLUXBASE_TOKEN = Deno.env.get("FLUXBASE_TOKEN") || "";

    // User function code starts here
    %s
    // User function code ends here

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

      // Output response as JSON
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
`, string(reqJSON), userCode)
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
