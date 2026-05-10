package runtime

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/shirou/gopsutil/v4/mem"

	"github.com/nimbleflux/fluxbase/internal/util"
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
	maxOutputSize  int // Max output size in bytes (0 = unlimited)
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

// WithMaxOutputSize sets the maximum output size in bytes (0 = unlimited)
func WithMaxOutputSize(bytes int) Option {
	return func(r *DenoRuntime) {
		r.maxOutputSize = bytes
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
		r.memoryLimitMB = 512              // Same limit as jobs
		r.maxOutputSize = 10 * 1024 * 1024 // 10MB default
	case RuntimeTypeJob:
		r.defaultTimeout = 300 * time.Second
		r.memoryLimitMB = 512
		r.maxOutputSize = 50 * 1024 * 1024 // 50MB default for jobs (more output expected)
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
// secrets is a map of secret name -> decrypted value that will be injected as FLUXBASE_SECRET_<NAME>
func (r *DenoRuntime) Execute(
	ctx context.Context,
	code string,
	req ExecutionRequest,
	permissions Permissions,
	cancelSignal *CancelSignal,
	timeoutOverride *time.Duration,
	secrets map[string]string,
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
			defer func() {
				if rec := recover(); rec != nil {
					log.Error().
						Interface("panic", rec).
						Str("id", req.ID.String()).
						Msg("Panic in cancel signal watcher - recovered")
				}
			}()
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
			log.Error().Err(tokenErr).Str("id", req.ID.String()).Msg("Failed to generate service token — tenant context is required")
			return nil, fmt.Errorf("cannot execute without tenant context: %w", tokenErr)
		}
		log.Debug().
			Str("id", req.ID.String()).
			Bool("user_token_generated", userToken != "").
			Bool("service_token_generated", serviceToken != "").
			Str("public_url", r.publicURL).
			Msg("SDK tokens generated for edge function execution")
	} else {
		log.Warn().
			Str("id", req.ID.String()).
			Bool("has_jwt_secret", r.jwtSecret != "").
			Bool("has_public_url", r.publicURL != "").
			Msg("SDK tokens NOT generated - missing jwtSecret or publicURL")
	}

	// Wrap the user code with our runtime bridge
	wrappedCode := r.wrapCode(code, req)

	// Ensure Deno cache directory exists (required for Deno to run)
	if err := os.MkdirAll("/tmp/deno", 0o750); err != nil {
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
	argsConfig := denoArgsConfig{
		RuntimeType:   r.runtimeType,
		DenoPath:      r.denoPath,
		PublicURL:     r.publicURL,
		MemoryLimitMB: r.memoryLimitMB,
		UserToken:     userToken,
		ServiceToken:  serviceToken,
	}
	args, memoryLimitMB, availableMemoryMB := buildDenoArgs(argsConfig, permissions, secrets, tmpPath)

	// Create command
	cmd := exec.CommandContext(execCtx, r.denoPath, args...)

	// Set environment variables (including secrets)
	cmd.Env = buildEnv(req, r.runtimeType, r.publicURL, userToken, serviceToken, cancelSignal, secrets)

	// Start command and stream output
	out, cmdErr := startAndStreamOutput(cmd, req.ID, r.maxOutputSize, r.onProgress, r.onLog)

	duration := time.Since(start)

	result := &ExecutionResult{
		Logs:       out.stderr.String(),
		DurationMs: duration.Milliseconds(),
	}

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

	if cancelSignal != nil && cancelSignal.IsCancelled() {
		result.Success = false
		result.Error = "Execution was cancelled"
		if r.runtimeType == RuntimeTypeFunction {
			result.Status = 499
		}
		return result, fmt.Errorf("execution cancelled")
	}

	if cmdErr != nil {
		result.Success = false

		if strings.Contains(cmdErr.Error(), "signal: killed") {
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
				Str("stderr", out.stderr.String()).
				Int64("duration_ms", duration.Milliseconds()).
				Msg("Execution failed")
		}
		return result, cmdErr
	}

	return r.parseResult(out.stdout.String(), out.stderr.String(), result)
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

		// Check if the nested result indicates failure (e.g., result.success: false)
		// This handles cases where the wrapper reports success but the actual
		// business logic returned a failure
		if result.Success && jobResult.Result != nil {
			if nestedSuccess, ok := jobResult.Result["success"]; ok {
				if success, isBool := nestedSuccess.(bool); isBool && !success {
					result.Success = false
					// Extract error from nested result if available
					if nestedError, ok := jobResult.Result["error"]; ok {
						if errStr, isString := nestedError.(string); isString {
							result.Error = errStr
						}
					}
				}
			}
		}

		return result, nil
	}

	// Fallback to legacy parsing - log warning since __RESULT__:: prefix was not found
	log.Warn().
		Str("stdout_preview", util.TruncateString(stdout, 200)).
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

	// Check if the nested result indicates failure (legacy parsing)
	if result.Success && jobResult.Result != nil {
		if nestedSuccess, ok := jobResult.Result["success"]; ok {
			if success, isBool := nestedSuccess.(bool); isBool && !success {
				result.Success = false
				if nestedError, ok := jobResult.Result["error"]; ok {
					if errStr, isString := nestedError.(string); isString {
						result.Error = errStr
					}
				}
			}
		}
	}

	return result, nil
}

// classifyStderrLine determines the appropriate log level for a stderr line.
// Deno writes informational messages (like download progress, warnings) to stderr,
// so we need to classify them appropriately rather than treating all as errors.
func classifyStderrLine(line string) string {
	// Strip ANSI color codes for pattern matching
	stripped := stripAnsiCodes(line)

	// Deno informational messages (not errors)
	infoPatterns := []string{
		"Download ",
		"Downloading ",
		"Check ",
		"Checking ",
		"Compile ",
		"Compiling ",
	}
	for _, pattern := range infoPatterns {
		if strings.Contains(stripped, pattern) {
			return "info"
		}
	}

	// Deno warnings
	if strings.Contains(stripped, "Warning") || strings.Contains(stripped, "warning:") {
		return "warn"
	}

	// Default to error for other stderr content
	return "error"
}

// buildNetworkAllowList constructs the list of allowed domains for Deno's --allow-net flag.
// It combines explicitly allowed domains (if any) with the self-host for SDK calls,
// then removes blocked domains. Returns an empty slice to indicate unrestricted net access
// (Deno default with --allow-net without domains), or a specific domain list for --allow-net=domain1,domain2,...
func buildNetworkAllowList(permissions Permissions, selfURL string) []string {
	// If no explicit allowlist is provided, return nil for unrestricted --allow-net.
	// Deno doesn't support per-domain blocking, only per-domain allowing,
	// so an empty AllowedDomains means "allow all".
	if len(permissions.AllowedDomains) == 0 {
		return nil
	}

	// Build domain list from explicit allowlist
	domains := make([]string, 0, len(permissions.AllowedDomains)+1)
	domains = append(domains, permissions.AllowedDomains...)

	// Add self-host for SDK calls (essential for internal communication)
	if selfURL != "" {
		if host := extractHost(selfURL); host != "" {
			domains = append(domains, host)
		}
	}

	// If no blocked domains, return the allowlist as-is
	if len(permissions.BlockedDomains) == 0 {
		return domains
	}

	// Build blocked domain set for efficient lookup
	blocked := make(map[string]bool, len(permissions.BlockedDomains))
	for _, d := range permissions.BlockedDomains {
		blocked[d] = true
	}

	// Filter out blocked domains
	filtered := make([]string, 0, len(domains))
	for _, d := range domains {
		if !blocked[d] {
			filtered = append(filtered, d)
		}
	}

	return filtered
}

// extractHost extracts the hostname from a URL, returning empty string if parsing fails
func extractHost(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return parsed.Hostname()
}

// stripAnsiCodes removes ANSI escape sequences from a string
func stripAnsiCodes(s string) string {
	// Simple state machine to strip ANSI codes
	var result strings.Builder
	inEscape := false
	for i := 0; i < len(s); i++ {
		if s[i] == '\x1b' {
			inEscape = true
			continue
		}
		if inEscape {
			// ANSI sequences end with a letter
			if (s[i] >= 'a' && s[i] <= 'z') || (s[i] >= 'A' && s[i] <= 'Z') {
				inEscape = false
			}
			continue
		}
		result.WriteByte(s[i])
	}
	return result.String()
}
