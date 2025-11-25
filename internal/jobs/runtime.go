package jobs

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// JobRuntime manages execution of Deno-based job functions
type JobRuntime struct {
	denoPath             string
	defaultTimeout       time.Duration
	defaultMemoryLimitMB int
	onProgress           func(jobID uuid.UUID, progress *Progress)
	onLog                func(jobID uuid.UUID, message string)
}

// NewJobRuntime creates a new job runtime
func NewJobRuntime(defaultTimeout time.Duration, defaultMemoryLimitMB int) *JobRuntime {
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

	return &JobRuntime{
		denoPath:             denoPath,
		defaultTimeout:       defaultTimeout,
		defaultMemoryLimitMB: defaultMemoryLimitMB,
	}
}

// SetProgressCallback sets the callback for progress updates
func (r *JobRuntime) SetProgressCallback(fn func(jobID uuid.UUID, progress *Progress)) {
	r.onProgress = fn
}

// SetLogCallback sets the callback for log messages
func (r *JobRuntime) SetLogCallback(fn func(jobID uuid.UUID, message string)) {
	r.onLog = fn
}

// JobExecutionRequest represents a job execution context
type JobExecutionRequest struct {
	JobID      uuid.UUID              `json:"job_id"`
	JobName    string                 `json:"job_name"`
	Payload    map[string]interface{} `json:"payload"`
	UserID     *uuid.UUID             `json:"user_id,omitempty"`
	UserEmail  *string                `json:"user_email,omitempty"`
	UserRole   *string                `json:"user_role,omitempty"`
	Namespace  string                 `json:"namespace"`
	RetryCount int                    `json:"retry_count"`
}

// JobExecutionResult represents the output of a job execution
type JobExecutionResult struct {
	Success    bool                   `json:"success"`
	Result     map[string]interface{} `json:"result,omitempty"`
	Error      string                 `json:"error,omitempty"`
	Logs       string                 `json:"logs"`
	DurationMs int64                  `json:"duration_ms"`
}

// Execute runs a job function with the given code and context
func (r *JobRuntime) Execute(
	ctx context.Context,
	job *Job,
	code string,
	permissions Permissions,
	cancelSignal *CancelSignal,
) (*JobExecutionResult, error) {
	start := time.Now()

	// Get timeout from job or use default
	timeout := r.defaultTimeout
	if job.MaxDurationSeconds != nil {
		timeout = time.Duration(*job.MaxDurationSeconds) * time.Second
	}

	// Create context with timeout
	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Prepare execution request
	execReq := JobExecutionRequest{
		JobID:      job.ID,
		JobName:    job.JobName,
		Namespace:  job.Namespace,
		RetryCount: job.RetryCount,
	}

	// Parse payload if present
	if job.Payload != nil {
		if err := json.Unmarshal([]byte(*job.Payload), &execReq.Payload); err != nil {
			return nil, fmt.Errorf("failed to parse job payload: %w", err)
		}
	}

	// Add user context if available
	if job.CreatedBy != nil {
		execReq.UserID = job.CreatedBy
	}
	if job.UserEmail != nil {
		execReq.UserEmail = job.UserEmail
	}
	if job.UserRole != nil {
		execReq.UserRole = job.UserRole
	}

	// Wrap user code with runtime bridge
	wrappedCode := r.wrapJobCode(code, execReq)

	// Write code to temporary file
	tmpFile, err := os.CreateTemp("", fmt.Sprintf("job-exec-%s-*.ts", job.ID))
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

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

	// Add temp file path
	args = append(args, tmpPath)

	// Create command
	cmd := exec.CommandContext(execCtx, r.denoPath, args...)

	// Set environment variables
	cmd.Env = r.buildEnvForJob(job, cancelSignal)

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
		defer wg.Done()
		scanner := bufio.NewScanner(stdoutPipe)
		for scanner.Scan() {
			line := scanner.Text()
			stdoutBuilder.WriteString(line + "\n")

			// Check for progress updates
			if strings.HasPrefix(line, "__PROGRESS__::") {
				progressJSON := strings.TrimPrefix(line, "__PROGRESS__::")
				var progress Progress
				if err := json.Unmarshal([]byte(progressJSON), &progress); err == nil {
					// Call progress callback
					if r.onProgress != nil {
						r.onProgress(job.ID, &progress)
					}
				}
			}
		}
	}()

	// Process stderr (logs)
	wg.Add(1)
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stderrPipe)
		for scanner.Scan() {
			line := scanner.Text()
			stderrBuilder.WriteString(line + "\n")

			// Call log callback
			if r.onLog != nil {
				r.onLog(job.ID, line)
			}
		}
	}()

	// Wait for command to complete
	cmdErr := cmd.Wait()

	// Wait for output processing to complete
	wg.Wait()

	duration := time.Since(start)

	// Build result
	result := &JobExecutionResult{
		Logs:       stderrBuilder.String(),
		DurationMs: duration.Milliseconds(),
	}

	// Check for timeout
	if execCtx.Err() == context.DeadlineExceeded {
		result.Success = false
		result.Error = fmt.Sprintf("Job execution timeout after %v", timeout)
		log.Warn().
			Str("job_id", job.ID.String()).
			Str("job_name", job.JobName).
			Int64("timeout_ms", timeout.Milliseconds()).
			Int64("duration_ms", duration.Milliseconds()).
			Msg("Job execution timeout")
		return result, fmt.Errorf("execution timeout")
	}

	// Check for cancellation
	if cancelSignal != nil && cancelSignal.IsCancelled() {
		result.Success = false
		result.Error = "Job was cancelled"
		return result, fmt.Errorf("job cancelled")
	}

	// Check for execution errors
	if cmdErr != nil {
		result.Success = false
		result.Error = fmt.Sprintf("Job execution failed: %v", cmdErr)
		log.Error().
			Err(cmdErr).
			Str("job_id", job.ID.String()).
			Str("job_name", job.JobName).
			Str("stderr", stderrBuilder.String()).
			Int64("duration_ms", duration.Milliseconds()).
			Msg("Job execution failed")
		return result, cmdErr
	}

	// Parse result from stdout
	stdout := strings.TrimSpace(stdoutBuilder.String())

	// Remove progress lines from stdout to get final result
	lines := strings.Split(stdout, "\n")
	var resultLines []string
	for _, line := range lines {
		if !strings.HasPrefix(line, "__PROGRESS__::") {
			resultLines = append(resultLines, line)
		}
	}
	resultOutput := strings.TrimSpace(strings.Join(resultLines, "\n"))

	if resultOutput == "" {
		// No output means success with no result
		result.Success = true
		return result, nil
	}

	// Try to parse as JSON result
	var jobResult struct {
		Success bool                   `json:"success"`
		Result  map[string]interface{} `json:"result,omitempty"`
		Error   string                 `json:"error,omitempty"`
	}

	if err := json.Unmarshal([]byte(resultOutput), &jobResult); err != nil {
		// Not valid JSON, treat as plain text success result
		result.Success = true
		result.Result = map[string]interface{}{
			"output": resultOutput,
		}
		return result, nil
	}

	result.Success = jobResult.Success
	result.Result = jobResult.Result
	if !jobResult.Success {
		result.Error = jobResult.Error
	}

	return result, nil
}

// wrapJobCode wraps user code with job runtime bridge
func (r *JobRuntime) wrapJobCode(userCode string, req JobExecutionRequest) string {
	reqJSON, _ := json.Marshal(req)

	// Extract imports (same pattern as functions)
	imports, codeWithoutImports := r.extractImports(userCode)

	return fmt.Sprintf(`
// Fluxbase Job Runtime Bridge
%s

// Global API for job functions
const Fluxbase = {
  // Report progress (0-100)
  reportProgress(percent, message, data) {
    const progress = { percent, message, data };
    console.log('__PROGRESS__::' + JSON.stringify(progress));
  },

  // Get job payload
  getJobPayload() {
    const jobContext = %s;
    return jobContext.payload || {};
  },

  // Check if job was cancelled
  checkCancellation() {
    return Deno.env.get('FLUXBASE_JOB_CANCELLED') === 'true';
  },

  // Get job context
  getJobContext() {
    const jobContext = %s;
    return {
      job_id: jobContext.job_id,
      job_name: jobContext.job_name,
      namespace: jobContext.namespace,
      retry_count: jobContext.retry_count,
      payload: jobContext.payload || {},
      user: jobContext.user_id ? {
        id: jobContext.user_id,
        email: jobContext.user_email,
        role: jobContext.user_role
      } : null
    };
  }
};

// Make Fluxbase globally available
globalThis.Fluxbase = Fluxbase;

// User job code (imports extracted)
%s

// Execute job handler
(async () => {
  try {
    // Get job context
    const jobContext = %s;

    // Create a Request object for compatibility with edge functions
    const request = new Request('http://localhost', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(jobContext.payload || {})
    });

    let result;

    // Try to call handler function (standard pattern)
    if (typeof handler === 'function') {
      result = await handler(request);
    }
    // Try to call default export
    else if (typeof default_handler === 'function') {
      result = await default_handler(request);
    }
    // Try to call main function
    else if (typeof main === 'function') {
      result = await main(jobContext.payload);
    }
    else {
      throw new Error("No handler function found. Export a 'handler', 'default', or 'main' function.");
    }

    // Normalize result
    let finalResult = result;
    if (result && typeof result === 'object' && result.status !== undefined) {
      // HTTP response format from handler
      try {
        finalResult = JSON.parse(result.body);
      } catch {
        finalResult = result.body;
      }
    }

    // Output final result as JSON
    console.log(JSON.stringify({
      success: true,
      result: finalResult
    }));

  } catch (error) {
    // Output error
    console.error('Job execution error:', error.message);
    console.log(JSON.stringify({
      success: false,
      error: error.message,
      stack: error.stack
    }));
  }
})();
`, imports, string(reqJSON), string(reqJSON), codeWithoutImports, string(reqJSON))
}

// extractImports separates import/export statements from code
func (r *JobRuntime) extractImports(code string) (imports string, remaining string) {
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
			braceCount += strings.Count(line, "{") - strings.Count(line, "}")
			if braceCount == 0 {
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

// buildEnvForJob creates the environment variable list for job execution
func (r *JobRuntime) buildEnvForJob(job *Job, cancelSignal *CancelSignal) []string {
	env := []string{}

	// Blocked secrets (same as functions)
	blockedVars := map[string]bool{
		"FLUXBASE_AUTH_JWT_SECRET":         true,
		"FLUXBASE_DATABASE_PASSWORD":       true,
		"FLUXBASE_DATABASE_ADMIN_PASSWORD": true,
		"FLUXBASE_STORAGE_S3_SECRET_KEY":   true,
		"FLUXBASE_STORAGE_S3_ACCESS_KEY":   true,
		"FLUXBASE_EMAIL_SMTP_PASSWORD":     true,
		"FLUXBASE_SECURITY_SETUP_TOKEN":    true,
	}

	// Pass all FLUXBASE_* environment variables except blocked ones
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, "FLUXBASE_") {
			parts := strings.SplitN(e, "=", 2)
			if len(parts) == 2 {
				key := parts[0]
				if !blockedVars[key] {
					env = append(env, e)
				}
			}
		}
	}

	// Add job-specific environment variables
	env = append(env, fmt.Sprintf("FLUXBASE_JOB_ID=%s", job.ID))
	env = append(env, fmt.Sprintf("FLUXBASE_JOB_NAME=%s", job.JobName))
	env = append(env, fmt.Sprintf("FLUXBASE_JOB_NAMESPACE=%s", job.Namespace))

	// Add cancellation signal
	if cancelSignal != nil && cancelSignal.IsCancelled() {
		env = append(env, "FLUXBASE_JOB_CANCELLED=true")
	} else {
		env = append(env, "FLUXBASE_JOB_CANCELLED=false")
	}

	return env
}

// CancelSignal is a signal that can be used to cancel a job
type CancelSignal struct {
	mu        sync.RWMutex
	cancelled bool
}

// NewCancelSignal creates a new cancel signal
func NewCancelSignal() *CancelSignal {
	return &CancelSignal{}
}

// Cancel marks the job as cancelled
func (s *CancelSignal) Cancel() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cancelled = true
}

// IsCancelled returns true if the job was cancelled
func (s *CancelSignal) IsCancelled() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cancelled
}
