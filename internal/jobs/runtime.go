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

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/shirou/gopsutil/v3/mem"
)

// JobRuntime manages execution of Deno-based job functions
type JobRuntime struct {
	denoPath             string
	defaultTimeout       time.Duration
	defaultMemoryLimitMB int
	jwtSecret            string
	publicURL            string
	onProgress           func(jobID uuid.UUID, progress *Progress)
	onLog                func(jobID uuid.UUID, message string)
}

// NewJobRuntime creates a new job runtime
func NewJobRuntime(defaultTimeout time.Duration, defaultMemoryLimitMB int, jwtSecret, publicURL string) *JobRuntime {
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
		jwtSecret:            jwtSecret,
		publicURL:            publicURL,
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

// generateJobToken generates a JWT token for the job's user context
// This token respects RLS policies based on the user who submitted the job
func (r *JobRuntime) generateJobToken(job *Job, timeout time.Duration) (string, error) {
	if r.jwtSecret == "" {
		return "", fmt.Errorf("JWT secret not configured")
	}

	now := time.Now()

	// Build claims matching the auth.TokenClaims format
	claims := jwt.MapClaims{
		"iss":        "fluxbase",
		"iat":        now.Unix(),
		"exp":        now.Add(timeout).Unix(),
		"nbf":        now.Unix(),
		"jti":        uuid.New().String(),
		"token_type": "access",
		"job_id":     job.ID.String(), // Audit trail
	}

	// Add user context if available
	if job.CreatedBy != nil {
		claims["sub"] = job.CreatedBy.String()
		claims["user_id"] = job.CreatedBy.String()
	}
	if job.UserEmail != nil {
		claims["email"] = *job.UserEmail
	}
	if job.UserRole != nil {
		claims["role"] = *job.UserRole
	} else {
		claims["role"] = "authenticated"
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(r.jwtSecret))
}

// generateServiceToken generates a JWT token with service_role that bypasses RLS
// This token allows jobs to access all data regardless of ownership
func (r *JobRuntime) generateServiceToken(job *Job, timeout time.Duration) (string, error) {
	if r.jwtSecret == "" {
		return "", fmt.Errorf("JWT secret not configured")
	}

	now := time.Now()

	claims := jwt.MapClaims{
		"iss":        "fluxbase",
		"sub":        "service_role",
		"role":       "service_role",
		"iat":        now.Unix(),
		"exp":        now.Add(timeout).Unix(),
		"nbf":        now.Unix(),
		"jti":        uuid.New().String(),
		"token_type": "access",
		"job_id":     job.ID.String(), // Audit trail
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(r.jwtSecret))
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

	// Generate SDK tokens for job execution (needed before building args)
	var jobToken, serviceToken string
	if r.jwtSecret != "" && r.publicURL != "" {
		var tokenErr error
		jobToken, tokenErr = r.generateJobToken(job, timeout)
		if tokenErr != nil {
			log.Warn().Err(tokenErr).Str("job_id", job.ID.String()).Msg("Failed to generate job token, SDK will not be available")
		}
		serviceToken, tokenErr = r.generateServiceToken(job, timeout)
		if tokenErr != nil {
			log.Warn().Err(tokenErr).Str("job_id", job.ID.String()).Msg("Failed to generate service token, SDK will not be available")
		}
	}

	// Build Deno command
	args := []string{"run"}

	// Apply memory limit via V8 flags
	// Default to 512MB if not specified, use configured value otherwise
	memoryLimitMB := permissions.MemoryLimitMB
	if memoryLimitMB <= 0 {
		memoryLimitMB = r.defaultMemoryLimitMB
	}
	if memoryLimitMB <= 0 {
		memoryLimitMB = 512 // Fallback default
	}

	// Check available system memory and warn if limit exceeds it
	var availableMemoryMB uint64
	if vmStat, err := mem.VirtualMemory(); err == nil {
		availableMemoryMB = vmStat.Available / 1024 / 1024
		totalMemoryMB := vmStat.Total / 1024 / 1024

		if uint64(memoryLimitMB) > availableMemoryMB {
			log.Warn().
				Str("job_id", job.ID.String()).
				Str("job_name", job.JobName).
				Int("requested_memory_mb", memoryLimitMB).
				Uint64("available_memory_mb", availableMemoryMB).
				Uint64("total_memory_mb", totalMemoryMB).
				Msg("Job memory limit exceeds available system memory - OOM kill is likely")
		}
	}

	args = append(args, fmt.Sprintf("--v8-flags=--max-old-space-size=%d", memoryLimitMB))

	// Apply permissions
	// Note: --allow-net is required for npm imports and SDK API calls
	// SDK requires network access, so if SDK tokens are being generated, we need network
	if permissions.AllowNet || (jobToken != "" || serviceToken != "") {
		args = append(args, "--allow-net")
	}
	if permissions.AllowEnv {
		// Allow all env vars
		args = append(args, "--allow-env")
	} else {
		// Always allow FLUXBASE_* env vars for SDK access
		args = append(args, "--allow-env=FLUXBASE_URL,FLUXBASE_JOB_TOKEN,FLUXBASE_SERVICE_TOKEN,FLUXBASE_JOB_ID,FLUXBASE_JOB_NAME,FLUXBASE_JOB_NAMESPACE,FLUXBASE_JOB_CANCELLED")
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
	cmd.Env = r.buildEnvForJob(job, cancelSignal, jobToken, serviceToken)

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
				log.Error().Interface("panic", rec).Str("job_id", job.ID.String()).Msg("Panic in stdout processing - recovered")
			}
			wg.Done()
		}()
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
			} else {
				// Regular console.log output - send to log callback
				if r.onLog != nil {
					r.onLog(job.ID, line)
				}
			}
		}
	}()

	// Process stderr (logs)
	wg.Add(1)
	go func() {
		defer func() {
			if rec := recover(); rec != nil {
				log.Error().Interface("panic", rec).Str("job_id", job.ID.String()).Msg("Panic in stderr processing - recovered")
			}
			wg.Done()
		}()
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
		errMsg := cmdErr.Error()

		// Check for OOM kill (signal: killed)
		if strings.Contains(errMsg, "signal: killed") {
			// Get current system memory info for better error message
			var currentAvailableMB, totalMB uint64
			if vmStat, err := mem.VirtualMemory(); err == nil {
				currentAvailableMB = vmStat.Available / 1024 / 1024
				totalMB = vmStat.Total / 1024 / 1024
			}

			// Determine if this is a system memory limit vs V8 heap issue
			if totalMB > 0 && uint64(memoryLimitMB) > totalMB {
				result.Error = fmt.Sprintf("Job killed (Out of Memory). Requested %dMB but system only has %dMB total RAM. Reduce @fluxbase:memory or use streaming for large data.", memoryLimitMB, totalMB)
			} else if availableMemoryMB > 0 && uint64(memoryLimitMB) > availableMemoryMB {
				result.Error = fmt.Sprintf("Job killed (Out of Memory). Requested %dMB but only %dMB was available (system total: %dMB). Free up memory or process data in smaller chunks.", memoryLimitMB, availableMemoryMB, totalMB)
			} else {
				result.Error = fmt.Sprintf("Job killed (Out of Memory). V8 heap limit: %dMB. The job may need more memory than configured, or you should process data in smaller chunks.", memoryLimitMB)
			}

			log.Error().
				Str("job_id", job.ID.String()).
				Str("job_name", job.JobName).
				Int("memory_limit_mb", memoryLimitMB).
				Uint64("available_at_start_mb", availableMemoryMB).
				Uint64("available_at_end_mb", currentAvailableMB).
				Uint64("total_system_mb", totalMB).
				Int64("duration_ms", duration.Milliseconds()).
				Msg("Job killed - OOM. Consider streaming data or reducing memory usage.")
		} else {
			result.Error = fmt.Sprintf("Job execution failed: %v", cmdErr)
			log.Error().
				Err(cmdErr).
				Str("job_id", job.ID.String()).
				Str("job_name", job.JobName).
				Str("stderr", stderrBuilder.String()).
				Int("memory_limit_mb", memoryLimitMB).
				Int64("duration_ms", duration.Milliseconds()).
				Msg("Job execution failed")
		}
		return result, cmdErr
	}

	// Parse result from stdout
	stdout := strings.TrimSpace(stdoutBuilder.String())
	stderr := strings.TrimSpace(stderrBuilder.String())

	// Look for result line with __RESULT__:: prefix (most reliable)
	lines := strings.Split(stdout, "\n")
	var resultLine string
	for _, line := range lines {
		if strings.HasPrefix(line, "__RESULT__::") {
			resultLine = strings.TrimPrefix(line, "__RESULT__::")
		}
	}

	// Try to parse as JSON result
	var jobResult struct {
		Success bool                   `json:"success"`
		Result  map[string]interface{} `json:"result,omitempty"`
		Error   string                 `json:"error,omitempty"`
	}

	if resultLine != "" {
		// Found a result line with prefix - parse it
		if err := json.Unmarshal([]byte(resultLine), &jobResult); err != nil {
			// Result line exists but couldn't be parsed - treat as error
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

	// No __RESULT__:: prefix found - fallback to legacy parsing
	// Remove progress lines from stdout
	var resultLines []string
	for _, line := range lines {
		if !strings.HasPrefix(line, "__PROGRESS__::") {
			resultLines = append(resultLines, line)
		}
	}
	resultOutput := strings.TrimSpace(strings.Join(resultLines, "\n"))

	if resultOutput == "" {
		// No output - check stderr for errors
		if stderr != "" && (strings.Contains(stderr, "error") || strings.Contains(stderr, "Error")) {
			result.Success = false
			result.Error = stderr
			return result, nil
		}
		// No output and no errors - assume success
		result.Success = true
		return result, nil
	}

	// Try to parse the entire output as JSON (legacy format)
	if err := json.Unmarshal([]byte(resultOutput), &jobResult); err != nil {
		// Not valid JSON - check stderr for error indicators
		if stderr != "" && (strings.Contains(stderr, "error") || strings.Contains(stderr, "Error")) {
			result.Success = false
			result.Error = stderr
			return result, nil
		}
		// No stderr errors, treat as plain text success result
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

// Inline Fluxbase SDK for job runtime (no external npm dependency)
const _fluxbaseUrl = Deno.env.get('FLUXBASE_URL') || '';
const _jobToken = Deno.env.get('FLUXBASE_JOB_TOKEN') || '';
const _serviceToken = Deno.env.get('FLUXBASE_SERVICE_TOKEN') || '';

// Minimal Fluxbase client implementation for jobs
class _FluxbaseClient {
  constructor(url, token) {
    this.url = url.replace(/\/$/, '');
    this.token = token;
    this.headers = {
      'Content-Type': 'application/json',
      'Authorization': 'Bearer ' + token,
      'apikey': token,
    };
    this.storage = new _FluxbaseStorage(this);
    this.jobs = new _FluxbaseJobs(this);
    this.auth = new _FluxbaseAuth(this);
  }

  async _request(path, options = {}) {
    const response = await fetch(this.url + path, {
      method: options.method || 'GET',
      headers: { ...this.headers, ...options.headers },
      body: options.body ? JSON.stringify(options.body) : undefined,
    });
    const contentType = response.headers.get('content-type');
    const data = contentType?.includes('application/json') ? await response.json() : await response.text();
    if (!response.ok) {
      return { data: null, error: { message: data?.error || data?.message || response.statusText, code: response.status } };
    }
    return { data, error: null };
  }

  from(table) {
    return new _QueryBuilder(this, table);
  }

  async rpc(fn, params) {
    return this._request('/api/v1/rpc/' + fn, { method: 'POST', body: params || {} });
  }
}

// Query builder for database operations
class _QueryBuilder {
  constructor(client, table) {
    this.client = client;
    this.table = table;
    this.query = {};
    this.filters = [];
    this.method = 'GET';
    this.body = null;
  }

  select(columns, options) {
    this.query.select = columns || '*';
    if (options?.count) this.query.count = options.count;
    return this;
  }

  insert(data, options) {
    this.method = 'POST';
    this.body = data;
    if (options?.onConflict) this.query.on_conflict = options.onConflict;
    return this;
  }

  upsert(data, options) {
    this.method = 'POST';
    this.body = data;
    this.query.upsert = 'true';
    if (options?.onConflict) this.query.on_conflict = options.onConflict;
    return this;
  }

  update(data) {
    this.method = 'PATCH';
    this.body = data;
    return this;
  }

  delete() {
    this.method = 'DELETE';
    return this;
  }

  eq(column, value) { this.filters.push(column + '=eq.' + encodeURIComponent(value)); return this; }
  neq(column, value) { this.filters.push(column + '=neq.' + encodeURIComponent(value)); return this; }
  gt(column, value) { this.filters.push(column + '=gt.' + encodeURIComponent(value)); return this; }
  gte(column, value) { this.filters.push(column + '=gte.' + encodeURIComponent(value)); return this; }
  lt(column, value) { this.filters.push(column + '=lt.' + encodeURIComponent(value)); return this; }
  lte(column, value) { this.filters.push(column + '=lte.' + encodeURIComponent(value)); return this; }
  like(column, pattern) { this.filters.push(column + '=like.' + encodeURIComponent(pattern)); return this; }
  ilike(column, pattern) { this.filters.push(column + '=ilike.' + encodeURIComponent(pattern)); return this; }
  in(column, values) { this.filters.push(column + '=in.(' + values.map(v => encodeURIComponent(v)).join(',') + ')'); return this; }
  is(column, value) { this.filters.push(column + '=is.' + value); return this; }
  not(column, op, value) { this.filters.push(column + '=not.' + op + '.' + encodeURIComponent(value)); return this; }
  or(filters) { this.filters.push('or=(' + filters + ')'); return this; }
  order(column, options) { this.query.order = column + (options?.ascending === false ? '.desc' : '.asc'); return this; }
  limit(count) { this.query.limit = count; return this; }
  offset(count) { this.query.offset = count; return this; }
  range(from, to) { this.query.offset = from; this.query.limit = to - from + 1; return this; }

  async single() {
    this.query.limit = 1;
    const result = await this._execute();
    if (result.error) return result;
    return { data: Array.isArray(result.data) ? result.data[0] || null : result.data, error: null };
  }

  async maybeSingle() {
    return this.single();
  }

  async then(resolve, reject) {
    try {
      const result = await this._execute();
      resolve(result);
    } catch (e) {
      if (reject) reject(e);
      else throw e;
    }
  }

  async _execute() {
    let path = '/api/v1/rest/' + this.table;
    const params = [];
    if (this.query.select) params.push('select=' + encodeURIComponent(this.query.select));
    for (const f of this.filters) params.push(f);
    for (const [k, v] of Object.entries(this.query)) {
      if (k !== 'select') params.push(k + '=' + encodeURIComponent(v));
    }
    if (params.length > 0) path += '?' + params.join('&');
    return this.client._request(path, { method: this.method, body: this.body });
  }
}

// Storage client
class _FluxbaseStorage {
  constructor(client) { this.client = client; }
  from(bucket) { return new _StorageBucket(this.client, bucket); }

  async listBuckets() {
    return this.client._request('/api/v1/storage/buckets');
  }

  async createBucket(name, options) {
    return this.client._request('/api/v1/storage/buckets/' + encodeURIComponent(name), {
      method: 'POST',
      body: { public: options?.public, max_file_size: options?.fileSizeLimit },
    });
  }

  async getBucket(name) {
    return this.client._request('/api/v1/storage/buckets/' + encodeURIComponent(name));
  }

  async deleteBucket(name) {
    return this.client._request('/api/v1/storage/buckets/' + encodeURIComponent(name), { method: 'DELETE' });
  }
}

class _StorageBucket {
  constructor(client, bucket) { this.client = client; this.bucket = bucket; }

  // Helper to build storage path: /api/v1/storage/:bucket/:path
  _storagePath(filePath) {
    return '/api/v1/storage/' + encodeURIComponent(this.bucket) + '/' + filePath;
  }

  async upload(path, data, options) {
    const formData = new FormData();
    const blob = data instanceof Blob ? data : new Blob([data]);
    formData.append('file', blob, path.split('/').pop());

    const params = [];
    if (options?.upsert) params.push('upsert=true');
    const queryStr = params.length > 0 ? '?' + params.join('&') : '';

    const response = await fetch(
      this.client.url + this._storagePath(path) + queryStr,
      {
        method: 'POST',
        headers: { 'Authorization': 'Bearer ' + this.client.token, 'apikey': this.client.token },
        body: formData,
      }
    );
    const resData = await response.json().catch(() => null);
    if (!response.ok) return { data: null, error: { message: resData?.error || response.statusText } };
    return { data: { path }, error: null };
  }

  async download(path, options) {
    const response = await fetch(
      this.client.url + this._storagePath(path),
      { headers: { 'Authorization': 'Bearer ' + this.client.token, 'apikey': this.client.token } }
    );
    if (!response.ok) {
      const errData = await response.json().catch(() => ({ message: response.statusText }));
      return { data: null, error: { message: errData?.error || errData?.message || response.statusText } };
    }
    // Return stream if requested, otherwise return Blob
    if (options?.stream) {
      if (!response.body) {
        return { data: null, error: { message: 'Response body is not available for streaming' } };
      }
      return { data: response.body, error: null };
    }
    return { data: await response.blob(), error: null };
  }

  async list(pathPrefix, options) {
    const params = [];
    if (pathPrefix) params.push('prefix=' + encodeURIComponent(pathPrefix));
    if (options?.limit) params.push('limit=' + options.limit);
    if (options?.offset) params.push('offset=' + options.offset);
    const queryStr = params.length > 0 ? '?' + params.join('&') : '';
    // List uses /api/v1/storage/:bucket (no file path)
    return this.client._request('/api/v1/storage/' + encodeURIComponent(this.bucket) + queryStr);
  }

  async remove(paths) {
    // Delete individual files - need to delete each path
    const results = [];
    for (const p of (Array.isArray(paths) ? paths : [paths])) {
      const result = await this.client._request(this._storagePath(p), { method: 'DELETE' });
      results.push(result);
    }
    return { data: results, error: null };
  }

  async move(from, to) {
    // Move is not directly supported - copy then delete
    const copyResult = await this.copy(from, to);
    if (copyResult.error) return copyResult;
    return this.remove([from]);
  }

  async copy(from, to) {
    // Copy by downloading and re-uploading
    const downloadResult = await this.download(from);
    if (downloadResult.error) return downloadResult;
    return this.upload(to, downloadResult.data);
  }

  async createSignedUrl(path, expiresIn) {
    return this.client._request(
      this._storagePath(path) + '/signed-url',
      { method: 'POST', body: { expires_in: expiresIn } }
    );
  }

  getPublicUrl(path) {
    // Public URL format for public buckets
    return { data: { publicUrl: this.client.url + this._storagePath(path) } };
  }
}

// Jobs client
class _FluxbaseJobs {
  constructor(client) { this.client = client; }

  async submit(jobName, payload, options) {
    return this.client._request('/api/v1/jobs', {
      method: 'POST',
      body: {
        job_name: jobName,
        payload,
        namespace: options?.namespace,
        priority: options?.priority,
        scheduled_at: options?.scheduledAt?.toISOString(),
      },
    });
  }

  async get(jobId) {
    return this.client._request('/api/v1/jobs/' + jobId);
  }

  async list(options) {
    const params = [];
    if (options?.status) params.push('status=' + options.status);
    if (options?.jobName) params.push('job_name=' + encodeURIComponent(options.jobName));
    if (options?.limit) params.push('limit=' + options.limit);
    const queryStr = params.length > 0 ? '?' + params.join('&') : '';
    return this.client._request('/api/v1/jobs' + queryStr);
  }

  async cancel(jobId) {
    return this.client._request('/api/v1/jobs/' + jobId + '/cancel', { method: 'POST' });
  }
}

// Auth client (limited in job context)
class _FluxbaseAuth {
  constructor(client) { this.client = client; }

  async getSession() {
    return { data: { session: null }, error: null };
  }

  async getUser() {
    return this.client._request('/api/v1/auth/user');
  }
}

// Create SDK client instances
function _createFluxbaseClient(url, token, clientName) {
  if (!url || !token) {
    console.error('[Fluxbase SDK] ' + clientName + ': Missing URL or token');
    return null;
  }
  try {
    return new _FluxbaseClient(url, token);
  } catch (e) {
    console.error('[Fluxbase SDK] ' + clientName + ': Failed to create client:', e.message);
    return null;
  }
}

// User client - respects RLS based on job submitter's permissions
const _fluxbase = _createFluxbaseClient(_fluxbaseUrl, _jobToken, 'UserClient');

// Service client - bypasses RLS for system-level operations
const _fluxbaseService = _createFluxbaseClient(_fluxbaseUrl, _serviceToken, 'ServiceClient');

// Job utilities object - using arrow functions for consistent behavior
const _jobUtils = {
  // Report progress (0-100)
  reportProgress: (percent, message, data) => {
    const progress = { percent, message, data };
    console.log('__PROGRESS__::' + JSON.stringify(progress));
  },

  // Check if job was cancelled
  checkCancellation: () => {
    return Deno.env.get('FLUXBASE_JOB_CANCELLED') === 'true';
  },

  // Alias for checkCancellation (matches types.d.ts)
  isCancelled: async () => {
    return Deno.env.get('FLUXBASE_JOB_CANCELLED') === 'true';
  },

  // Get job context
  getJobContext: () => {
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
  },

  // Get job payload (convenience method)
  getJobPayload: () => {
    const jobContext = %s;
    return jobContext.payload || {};
  }
};

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

    // Try to call handler function with SDK clients
    // New signature: handler(req, fluxbase, fluxbaseService, job)
    if (typeof handler === 'function') {
      result = await handler(request, _fluxbase, _fluxbaseService, _jobUtils);
    }
    // Try to call default export
    else if (typeof default_handler === 'function') {
      result = await default_handler(request, _fluxbase, _fluxbaseService, _jobUtils);
    }
    // Try to call main function
    else if (typeof main === 'function') {
      result = await main(jobContext.payload, _fluxbase, _fluxbaseService, _jobUtils);
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

    // Output final result as JSON with prefix for reliable parsing
    console.log('__RESULT__::' + JSON.stringify({
      success: true,
      result: finalResult
    }));

  } catch (error) {
    // Output error with prefix for reliable parsing
    console.error('Job execution error:', error.message);
    console.log('__RESULT__::' + JSON.stringify({
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
	inMultilineExport := false
	braceCount := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Check if we're starting a multi-line type/interface declaration
		if !inMultilineDeclaration && !inMultilineExport &&
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

		// Check if we're starting a multi-line export { ... } statement
		if !inMultilineExport && strings.HasPrefix(trimmed, "export {") {
			braceCount = strings.Count(line, "{") - strings.Count(line, "}")
			importLines = append(importLines, line)
			if braceCount > 0 {
				// Opening brace without closing - multi-line export
				inMultilineExport = true
			}
			continue
		}

		// If we're in a multi-line export, continue collecting lines
		if inMultilineExport {
			importLines = append(importLines, line)
			braceCount += strings.Count(line, "{") - strings.Count(line, "}")
			if braceCount <= 0 {
				inMultilineExport = false
			}
			continue
		}

		// Extract single-line import/export statements
		if strings.HasPrefix(trimmed, "import ") ||
			strings.HasPrefix(trimmed, "import{") ||
			strings.HasPrefix(trimmed, "export * ") {
			importLines = append(importLines, line)
		} else {
			codeLines = append(codeLines, line)
		}
	}

	return strings.Join(importLines, "\n"), strings.Join(codeLines, "\n")
}

// buildEnvForJob creates the environment variable list for job execution
func (r *JobRuntime) buildEnvForJob(job *Job, cancelSignal *CancelSignal, jobToken, serviceToken string) []string {
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

	// Add SDK client credentials
	if r.publicURL != "" {
		env = append(env, fmt.Sprintf("FLUXBASE_URL=%s", r.publicURL))
	}
	if jobToken != "" {
		env = append(env, fmt.Sprintf("FLUXBASE_JOB_TOKEN=%s", jobToken))
	}
	if serviceToken != "" {
		env = append(env, fmt.Sprintf("FLUXBASE_SERVICE_TOKEN=%s", serviceToken))
	}

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
