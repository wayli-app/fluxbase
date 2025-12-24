package runtime

import (
	"bufio"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/shirou/gopsutil/v3/mem"
)

// embeddedSDK contains the JavaScript SDK for runtime execution
// Generated from: sdk/src/*.ts via `npm run generate:embedded-sdk`
//
//go:embed embedded_sdk.js
var embeddedSDK string

// DenoRuntime manages execution of Deno-based functions and jobs
type DenoRuntime struct {
	denoPath       string
	defaultTimeout time.Duration
	memoryLimitMB  int // V8 heap limit in MB
	jwtSecret      string
	publicURL      string
	runtimeType    RuntimeType
	onProgress     func(id uuid.UUID, progress *Progress)
	onLog          func(id uuid.UUID, level string, message string)
}

// Option is a functional option for configuring DenoRuntime
type Option func(*DenoRuntime)

// WithTimeout sets the default timeout
func WithTimeout(timeout time.Duration) Option {
	return func(r *DenoRuntime) {
		r.defaultTimeout = timeout
	}
}

// WithMemoryLimit sets the V8 heap memory limit in MB
func WithMemoryLimit(mb int) Option {
	return func(r *DenoRuntime) {
		r.memoryLimitMB = mb
	}
}

// NewRuntime creates a new Deno runtime for the specified type
func NewRuntime(runtimeType RuntimeType, jwtSecret, publicURL string, opts ...Option) *DenoRuntime {
	r := &DenoRuntime{
		denoPath:    detectDenoPath(),
		jwtSecret:   jwtSecret,
		publicURL:   publicURL,
		runtimeType: runtimeType,
	}

	// Apply defaults based on type
	switch runtimeType {
	case RuntimeTypeFunction:
		r.defaultTimeout = 30 * time.Second
		r.memoryLimitMB = 512 // Same limit as jobs
	case RuntimeTypeJob:
		r.defaultTimeout = 300 * time.Second
		r.memoryLimitMB = 512
	}

	// Apply options
	for _, opt := range opts {
		opt(r)
	}

	return r
}

// detectDenoPath finds the Deno executable
func detectDenoPath() string {
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
				return path
			}
		}
	}
	return denoPath
}

// SetProgressCallback sets the callback for progress updates
func (r *DenoRuntime) SetProgressCallback(fn func(id uuid.UUID, progress *Progress)) {
	r.onProgress = fn
}

// SetLogCallback sets the callback for log messages
func (r *DenoRuntime) SetLogCallback(fn func(id uuid.UUID, level string, message string)) {
	r.onLog = fn
}

// RuntimeType returns the runtime type
func (r *DenoRuntime) RuntimeType() RuntimeType {
	return r.runtimeType
}

// Execute runs user code with the given request context
// timeoutOverride allows callers to specify a custom timeout; if nil, defaultTimeout is used
func (r *DenoRuntime) Execute(
	ctx context.Context,
	code string,
	req ExecutionRequest,
	permissions Permissions,
	cancelSignal *CancelSignal,
	timeoutOverride *time.Duration,
) (*ExecutionResult, error) {
	start := time.Now()

	// Get timeout - use override if provided, otherwise use default
	timeout := r.defaultTimeout
	if timeoutOverride != nil && *timeoutOverride > 0 {
		timeout = *timeoutOverride
	}

	// Create context with timeout that's also cancelled by the cancel signal
	timeoutCtx, timeoutCancel := context.WithTimeout(ctx, timeout)
	defer timeoutCancel()

	// Merge timeout context with cancel signal context
	execCtx, execCancel := context.WithCancel(timeoutCtx)
	defer execCancel()

	// Watch for cancel signal and cancel exec context
	if cancelSignal != nil {
		go func() {
			select {
			case <-cancelSignal.Context().Done():
				execCancel() // This will kill the Deno process
			case <-execCtx.Done():
				// Already done (timeout or normal completion)
			}
		}()
	}

	// Generate SDK tokens for execution
	var userToken, serviceToken string
	if r.jwtSecret != "" && r.publicURL != "" {
		var tokenErr error
		userToken, tokenErr = generateUserToken(r.jwtSecret, req, r.runtimeType, timeout)
		if tokenErr != nil {
			log.Warn().Err(tokenErr).Str("id", req.ID.String()).Msg("Failed to generate user token, SDK will not be available")
		}
		serviceToken, tokenErr = generateServiceToken(r.jwtSecret, req, r.runtimeType, timeout)
		if tokenErr != nil {
			log.Warn().Err(tokenErr).Str("id", req.ID.String()).Msg("Failed to generate service token, SDK will not be available")
		}
	}

	// Wrap the user code with our runtime bridge
	wrappedCode := r.wrapCode(code, req)

	// Ensure Deno cache directory exists (required for Deno to run)
	if err := os.MkdirAll("/tmp/deno", 0755); err != nil {
		log.Warn().Err(err).Msg("Failed to create Deno cache directory")
	}

	// Write code to temporary file to allow Deno to properly handle TypeScript
	tmpFile, err := os.CreateTemp("", fmt.Sprintf("%s-exec-%s-*.ts", r.runtimeType.String(), req.ID))
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer func() { _ = os.Remove(tmpPath) }()

	if _, err := tmpFile.WriteString(wrappedCode); err != nil {
		_ = tmpFile.Close()
		return nil, fmt.Errorf("failed to write code to temp file: %w", err)
	}
	_ = tmpFile.Close()

	// Build Deno command
	args := []string{"run"}

	// Apply memory limit via V8 flags (jobs only)
	memoryLimitMB := permissions.MemoryLimitMB
	if memoryLimitMB <= 0 {
		memoryLimitMB = r.memoryLimitMB
	}

	var availableMemoryMB uint64
	if r.runtimeType == RuntimeTypeJob && memoryLimitMB > 0 {
		// Check available system memory and warn if limit exceeds it
		if vmStat, err := mem.VirtualMemory(); err == nil {
			availableMemoryMB = vmStat.Available / 1024 / 1024
			totalMemoryMB := vmStat.Total / 1024 / 1024

			if uint64(memoryLimitMB) > availableMemoryMB {
				log.Warn().
					Str("id", req.ID.String()).
					Str("name", req.Name).
					Int("requested_memory_mb", memoryLimitMB).
					Uint64("available_memory_mb", availableMemoryMB).
					Uint64("total_memory_mb", totalMemoryMB).
					Msg("Memory limit exceeds available system memory - OOM kill is likely")
			}
		}

		args = append(args, fmt.Sprintf("--v8-flags=--max-old-space-size=%d", memoryLimitMB))
	}

	// Apply permissions - always allow net for SDK API calls
	if permissions.AllowNet || (userToken != "" || serviceToken != "") {
		args = append(args, "--allow-net")
	}
	if permissions.AllowEnv {
		args = append(args, "--allow-env")
	} else {
		// Always allow specific env vars for SDK access
		args = append(args, fmt.Sprintf("--allow-env=%s", allowedEnvVars(r.runtimeType)))
	}
	if permissions.AllowRead {
		args = append(args, "--allow-read")
	}
	if permissions.AllowWrite {
		args = append(args, "--allow-write")
	}

	args = append(args, tmpPath)

	// Create command
	cmd := exec.CommandContext(execCtx, r.denoPath, args...)

	// Set environment variables
	cmd.Env = buildEnv(req, r.runtimeType, r.publicURL, userToken, serviceToken, cancelSignal)

	// Capture stdout and stderr with streaming
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start command
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start deno: %w", err)
	}

	// Process output streams concurrently
	var wg sync.WaitGroup
	var stdoutBuilder, stderrBuilder strings.Builder

	// Process stdout (progress updates and final result)
	wg.Add(1)
	go func() {
		defer func() {
			if rec := recover(); rec != nil {
				log.Error().Interface("panic", rec).Str("id", req.ID.String()).Msg("Panic in stdout processing - recovered")
			}
			wg.Done()
		}()
		scanner := bufio.NewScanner(stdoutPipe)
		// Increase buffer size to handle large results (1MB max per line)
		const maxLineSize = 1024 * 1024
		scanner.Buffer(make([]byte, maxLineSize), maxLineSize)

		for scanner.Scan() {
			line := scanner.Text()
			stdoutBuilder.WriteString(line + "\n")

			// Check for progress updates
			if strings.HasPrefix(line, "__PROGRESS__::") {
				progressJSON := strings.TrimPrefix(line, "__PROGRESS__::")
				var progress Progress
				if err := json.Unmarshal([]byte(progressJSON), &progress); err == nil {
					if r.onProgress != nil {
						r.onProgress(req.ID, &progress)
					}
				}
			} else if line != "" {
				// Regular console.log output - send to log callback
				if r.onLog != nil {
					r.onLog(req.ID, "info", line)
				}
			}
		}

		// Check for scanner errors
		if err := scanner.Err(); err != nil {
			log.Warn().
				Err(err).
				Str("id", req.ID.String()).
				Msg("Scanner error while reading stdout - result line may be truncated")
		}
	}()

	// Process stderr (logs)
	wg.Add(1)
	go func() {
		defer func() {
			if rec := recover(); rec != nil {
				log.Error().Interface("panic", rec).Str("id", req.ID.String()).Msg("Panic in stderr processing - recovered")
			}
			wg.Done()
		}()
		scanner := bufio.NewScanner(stderrPipe)
		// Increase buffer size to handle large error messages (1MB max per line)
		const maxLineSize = 1024 * 1024
		scanner.Buffer(make([]byte, maxLineSize), maxLineSize)

		for scanner.Scan() {
			line := scanner.Text()
			stderrBuilder.WriteString(line + "\n")

			if r.onLog != nil && line != "" {
				r.onLog(req.ID, "error", line)
			}
		}

		// Check for scanner errors
		if err := scanner.Err(); err != nil {
			log.Warn().
				Err(err).
				Str("id", req.ID.String()).
				Msg("Scanner error while reading stderr")
		}
	}()

	// Wait for command to complete
	cmdErr := cmd.Wait()

	// Wait for output processing to complete
	wg.Wait()

	duration := time.Since(start)

	// Build result
	result := &ExecutionResult{
		Logs:       stderrBuilder.String(),
		DurationMs: duration.Milliseconds(),
	}

	// Check for timeout
	if execCtx.Err() == context.DeadlineExceeded {
		result.Success = false
		result.Error = fmt.Sprintf("Execution timeout after %v", timeout)
		if r.runtimeType == RuntimeTypeFunction {
			result.Status = 504
		}
		log.Warn().
			Str("id", req.ID.String()).
			Str("name", req.Name).
			Int64("timeout_ms", timeout.Milliseconds()).
			Int64("duration_ms", duration.Milliseconds()).
			Msg("Execution timeout")
		return result, fmt.Errorf("execution timeout after %v", timeout)
	}

	// Check for cancellation
	if cancelSignal != nil && cancelSignal.IsCancelled() {
		result.Success = false
		result.Error = "Execution was cancelled"
		if r.runtimeType == RuntimeTypeFunction {
			result.Status = 499 // Client Closed Request
		}
		return result, fmt.Errorf("execution cancelled")
	}

	// Check for execution errors
	if cmdErr != nil {
		result.Success = false

		// Check for OOM kill (jobs only)
		if r.runtimeType == RuntimeTypeJob && strings.Contains(cmdErr.Error(), "signal: killed") {
			result.Error = r.buildOOMErrorMessage(memoryLimitMB, availableMemoryMB)
			log.Error().
				Str("id", req.ID.String()).
				Str("name", req.Name).
				Int("memory_limit_mb", memoryLimitMB).
				Uint64("available_at_start_mb", availableMemoryMB).
				Int64("duration_ms", duration.Milliseconds()).
				Msg("Execution killed - OOM")
		} else {
			result.Error = fmt.Sprintf("Execution failed: %v", cmdErr)
			if r.runtimeType == RuntimeTypeFunction {
				result.Status = 500
			}
			log.Error().
				Err(cmdErr).
				Str("id", req.ID.String()).
				Str("name", req.Name).
				Str("stderr", stderrBuilder.String()).
				Int64("duration_ms", duration.Milliseconds()).
				Msg("Execution failed")
		}
		return result, cmdErr
	}

	// Parse result from stdout
	return r.parseResult(stdoutBuilder.String(), stderrBuilder.String(), result)
}

// buildOOMErrorMessage constructs an informative OOM error message
func (r *DenoRuntime) buildOOMErrorMessage(memoryLimitMB int, availableMemoryMB uint64) string {
	var totalMB uint64
	if vmStat, err := mem.VirtualMemory(); err == nil {
		totalMB = vmStat.Total / 1024 / 1024
	}

	if totalMB > 0 && uint64(memoryLimitMB) > totalMB {
		return fmt.Sprintf("Killed (Out of Memory). Requested %dMB but system only has %dMB total RAM. Reduce memory limit or use streaming for large data.", memoryLimitMB, totalMB)
	} else if availableMemoryMB > 0 && uint64(memoryLimitMB) > availableMemoryMB {
		return fmt.Sprintf("Killed (Out of Memory). Requested %dMB but only %dMB was available (system total: %dMB). Free up memory or process data in smaller chunks.", memoryLimitMB, availableMemoryMB, totalMB)
	}
	return fmt.Sprintf("Killed (Out of Memory). V8 heap limit: %dMB. May need more memory than configured, or should process data in smaller chunks.", memoryLimitMB)
}

// parseResult parses the execution result from stdout
func (r *DenoRuntime) parseResult(stdout, stderr string, result *ExecutionResult) (*ExecutionResult, error) {
	stdout = strings.TrimSpace(stdout)

	// Look for result line with __RESULT__:: prefix (most reliable)
	lines := strings.Split(stdout, "\n")
	var resultLine string
	for _, line := range lines {
		if strings.HasPrefix(line, "__RESULT__::") {
			resultLine = strings.TrimPrefix(line, "__RESULT__::")
		}
	}

	switch r.runtimeType {
	case RuntimeTypeFunction:
		return r.parseFunctionResult(resultLine, stdout, lines, result)
	case RuntimeTypeJob:
		return r.parseJobResult(resultLine, stdout, stderr, lines, result)
	default:
		result.Success = true
		return result, nil
	}
}

// parseFunctionResult parses the result for edge functions
func (r *DenoRuntime) parseFunctionResult(resultLine, stdout string, lines []string, result *ExecutionResult) (*ExecutionResult, error) {
	if resultLine != "" {
		var response struct {
			Status  int               `json:"status"`
			Headers map[string]string `json:"headers"`
			Body    string            `json:"body"`
		}
		if err := json.Unmarshal([]byte(resultLine), &response); err != nil {
			result.Status = 500
			result.Success = false
			result.Error = fmt.Sprintf("Failed to parse function result: %v", err)
			return result, nil
		}
		result.Status = response.Status
		result.Headers = response.Headers
		result.Body = response.Body
		result.Success = response.Status >= 200 && response.Status < 400
		return result, nil
	}

	// Fallback to legacy parsing
	var resultLines []string
	for _, line := range lines {
		if !strings.HasPrefix(line, "__PROGRESS__::") {
			resultLines = append(resultLines, line)
		}
	}
	resultOutput := strings.TrimSpace(strings.Join(resultLines, "\n"))

	if resultOutput == "" {
		result.Status = 200
		result.Body = ""
		result.Success = true
		return result, nil
	}

	var response struct {
		Status  int               `json:"status"`
		Headers map[string]string `json:"headers"`
		Body    string            `json:"body"`
	}

	if err := json.Unmarshal([]byte(resultOutput), &response); err != nil {
		result.Status = 200
		result.Body = resultOutput
		result.Success = true
		return result, nil
	}

	result.Status = response.Status
	result.Headers = response.Headers
	result.Body = response.Body
	result.Success = response.Status >= 200 && response.Status < 400

	return result, nil
}

// parseJobResult parses the result for job functions
func (r *DenoRuntime) parseJobResult(resultLine, stdout, stderr string, lines []string, result *ExecutionResult) (*ExecutionResult, error) {
	var jobResult struct {
		Success bool                   `json:"success"`
		Result  map[string]interface{} `json:"result,omitempty"`
		Error   string                 `json:"error,omitempty"`
	}

	if resultLine != "" {
		if err := json.Unmarshal([]byte(resultLine), &jobResult); err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("Failed to parse job result: %v", err)
			return result, nil
		}
		result.Success = jobResult.Success
		result.Result = jobResult.Result
		if !jobResult.Success {
			result.Error = jobResult.Error
		}
		return result, nil
	}

	// Fallback to legacy parsing - log warning since __RESULT__:: prefix was not found
	log.Warn().
		Str("stdout_preview", truncateString(stdout, 200)).
		Msg("Job result not found with __RESULT__:: prefix - handler may have exited early or returned non-serializable value")

	var resultLines []string
	for _, line := range lines {
		if !strings.HasPrefix(line, "__PROGRESS__::") {
			resultLines = append(resultLines, line)
		}
	}
	resultOutput := strings.TrimSpace(strings.Join(resultLines, "\n"))

	if resultOutput == "" {
		if stderr != "" && (strings.Contains(stderr, "error") || strings.Contains(stderr, "Error")) {
			result.Success = false
			result.Error = stderr
			return result, nil
		}
		result.Success = true
		return result, nil
	}

	if err := json.Unmarshal([]byte(resultOutput), &jobResult); err != nil {
		if stderr != "" && (strings.Contains(stderr, "error") || strings.Contains(stderr, "Error")) {
			result.Success = false
			result.Error = stderr
			return result, nil
		}
		// Don't wrap stdout in {"output": ...} - just return nil result
		result.Success = true
		result.Result = nil
		return result, nil
	}

	result.Success = jobResult.Success
	result.Result = jobResult.Result
	if !jobResult.Success {
		result.Error = jobResult.Error
	}

	return result, nil
}

// truncateString truncates a string to maxLen characters, adding "..." if truncated
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
