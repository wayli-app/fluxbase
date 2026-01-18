package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/fluxbase-eu/fluxbase/internal/config"
	"github.com/fluxbase-eu/fluxbase/internal/runtime"
	"github.com/fluxbase-eu/fluxbase/internal/secrets"
	"github.com/fluxbase-eu/fluxbase/internal/settings"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// Worker executes jobs from the queue
type Worker struct {
	ID                     uuid.UUID
	Name                   string
	Config                 *config.JobsConfig
	Storage                *Storage
	Runtime                *runtime.DenoRuntime
	SecretsStorage         *secrets.Storage
	SettingsSecretsService *settings.SecretsService
	MaxConcurrent          int
	publicURL              string
	currentJobs            sync.Map // jobID -> *runtime.CancelSignal
	jobLogCounters         sync.Map // jobID -> *int (line counter)
	jobStartTimes          sync.Map // jobID -> time.Time (for ETA calculation)
	jobLogsDisabled        sync.Map // jobID -> bool (whether execution logs are disabled)
	currentJobCount        int
	currentJobCountMutex   sync.RWMutex
	shutdownChan           chan struct{}
	shutdownComplete       chan struct{}
	draining               bool       // True when worker is draining (not accepting new jobs)
	drainingMutex          sync.RWMutex
}

// NewWorker creates a new worker
func NewWorker(cfg *config.JobsConfig, storage *Storage, jwtSecret, publicURL string, secretsStorage *secrets.Storage) *Worker {
	workerID := uuid.New()
	hostname, _ := os.Hostname()

	// Create runtime with SDK credentials
	jobRuntime := runtime.NewRuntime(
		runtime.RuntimeTypeJob,
		jwtSecret,
		publicURL,
		runtime.WithTimeout(cfg.DefaultMaxDuration),
		runtime.WithMemoryLimit(128), // Default memory limit
	)

	worker := &Worker{
		ID:               workerID,
		Name:             fmt.Sprintf("worker-%s@%s", workerID.String()[:8], hostname),
		Config:           cfg,
		Storage:          storage,
		Runtime:          jobRuntime,
		SecretsStorage:   secretsStorage,
		MaxConcurrent:    cfg.MaxConcurrentPerWorker,
		publicURL:        publicURL,
		shutdownChan:     make(chan struct{}),
		shutdownComplete: make(chan struct{}),
	}

	// Set up runtime callbacks
	jobRuntime.SetProgressCallback(worker.handleProgressUpdate)
	jobRuntime.SetLogCallback(worker.handleLogMessage)

	return worker
}

// Start starts the worker
func (w *Worker) Start(ctx context.Context) error {
	log.Info().
		Str("worker_id", w.ID.String()).
		Str("worker_name", w.Name).
		Int("max_concurrent", w.MaxConcurrent).
		Msg("Starting job worker")

	// Register worker in database
	hostname, _ := os.Hostname()
	metadata := map[string]interface{}{
		"hostname": hostname,
		"pid":      os.Getpid(),
	}
	metadataJSON, _ := json.Marshal(metadata)
	metadataStr := string(metadataJSON)

	workerName := w.Name
	workerRecord := &WorkerRecord{
		ID:                w.ID,
		Name:              &workerName,
		Hostname:          &hostname,
		Status:            WorkerStatusActive,
		MaxConcurrentJobs: w.MaxConcurrent,
		CurrentJobCount:   0,
		Metadata:          &metadataStr,
	}

	if err := w.Storage.RegisterWorker(ctx, workerRecord); err != nil {
		return fmt.Errorf("failed to register worker: %w", err)
	}

	// Start heartbeat goroutine
	go w.heartbeatLoop(ctx)

	// Start progress timeout monitor
	go w.progressTimeoutLoop(ctx)

	// Start stale worker cleanup loop
	go w.staleWorkerCleanupLoop(ctx)

	// Start job poll loop
	go w.pollLoop(ctx)

	// Wait for shutdown signal
	<-w.shutdownChan

	// Set draining mode to stop accepting new jobs
	w.setDraining(true)

	log.Info().
		Str("worker_id", w.ID.String()).
		Dur("timeout", w.Config.GracefulShutdownTimeout).
		Msg("Worker shutting down gracefully, waiting for running jobs")

	// Update worker status in database to draining
	if err := w.Storage.UpdateWorkerStatus(ctx, w.ID, WorkerStatusDraining); err != nil {
		log.Warn().Err(err).Msg("Failed to update worker status to draining")
	}

	// Use configured timeout (default 5m) instead of hard-coded 30s
	timeout := w.Config.GracefulShutdownTimeout
	if timeout <= 0 {
		timeout = 5 * time.Minute
	}
	shutdownTimeout := time.After(timeout)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-shutdownTimeout:
			remainingJobs := w.getCurrentJobCount()
			log.Warn().
				Str("worker_id", w.ID.String()).
				Int("remaining_jobs", remainingJobs).
				Dur("timeout", timeout).
				Msg("Shutdown timeout reached, interrupting remaining jobs")
			w.interruptAllJobs()
			close(w.shutdownComplete)
			return nil
		case <-ticker.C:
			if w.getCurrentJobCount() == 0 {
				log.Info().Str("worker_id", w.ID.String()).Msg("All jobs completed, worker stopped")
				close(w.shutdownComplete)
				return nil
			}
		}
	}
}

// isDraining returns true if the worker is draining (not accepting new jobs)
func (w *Worker) isDraining() bool {
	w.drainingMutex.RLock()
	defer w.drainingMutex.RUnlock()
	return w.draining
}

// setDraining sets the draining state
func (w *Worker) setDraining(draining bool) {
	w.drainingMutex.Lock()
	defer w.drainingMutex.Unlock()
	w.draining = draining
}

// Stop stops the worker gracefully
func (w *Worker) Stop() {
	log.Info().Str("worker_id", w.ID.String()).Msg("Stopping worker")
	close(w.shutdownChan)
	<-w.shutdownComplete
}

// pollLoop continuously polls for and executes jobs
func (w *Worker) pollLoop(ctx context.Context) {
	ticker := time.NewTicker(w.Config.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Don't accept new jobs if draining
			if w.isDraining() {
				continue
			}

			// Check if we have capacity
			if w.hasCapacity() {
				// Try to claim a job
				job, err := w.Storage.ClaimNextJob(ctx, w.ID)
				if err != nil {
					log.Error().Err(err).Msg("Failed to claim job")
					continue
				}

				if job != nil {
					// Execute job in goroutine with panic recovery
					go func(j *Job) {
						defer func() {
							if rec := recover(); rec != nil {
								log.Error().
									Interface("panic", rec).
									Str("job_id", j.ID.String()).
									Str("job_name", j.JobName).
									Msg("Panic in job execution - recovered, marking job as failed")
								// Mark job as failed (defers in executeJob will have already cleaned up job count)
								_ = w.Storage.FailJob(context.Background(), j.ID, fmt.Sprintf("Internal error: job execution panic: %v", rec))
							}
						}()
						w.executeJob(ctx, j)
					}(job)
				}
			}
		case <-w.shutdownChan:
			return
		}
	}
}

// heartbeatLoop sends periodic heartbeats
func (w *Worker) heartbeatLoop(ctx context.Context) {
	ticker := time.NewTicker(w.Config.WorkerHeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			count := w.getCurrentJobCount()
			if err := w.Storage.UpdateWorkerHeartbeat(ctx, w.ID, count); err != nil {
				log.Error().Err(err).Str("worker_id", w.ID.String()).Msg("Failed to send heartbeat")
			}
		case <-w.shutdownChan:
			// Send final heartbeat with stopped status
			if err := w.Storage.UpdateWorkerStatus(ctx, w.ID, WorkerStatusStopped); err != nil {
				log.Error().Err(err).Msg("Failed to update worker status to stopped")
			}
			return
		}
	}
}

// staleWorkerCleanupLoop periodically removes workers that haven't sent heartbeats
func (w *Worker) staleWorkerCleanupLoop(ctx context.Context) {
	// Run cleanup at half the WorkerTimeout interval for faster detection
	// This reduces worst-case detection time from 2*timeout to 1.5*timeout
	cleanupInterval := w.Config.WorkerTimeout / 2
	if cleanupInterval < 5*time.Second {
		cleanupInterval = 5 * time.Second // Minimum 5 seconds to avoid excessive DB queries
	}
	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			deleted, err := w.Storage.CleanupStaleWorkers(ctx, w.Config.WorkerTimeout)
			if err != nil {
				log.Error().Err(err).Msg("Failed to cleanup stale workers")
			} else if deleted > 0 {
				log.Info().Int64("deleted", deleted).Dur("timeout", w.Config.WorkerTimeout).Msg("Cleaned up stale workers")
			}

			// Reset any orphaned jobs (running but worker_id is NULL due to worker deletion)
			reset, err := w.Storage.ResetOrphanedJobs(ctx)
			if err != nil {
				log.Error().Err(err).Msg("Failed to reset orphaned jobs")
			} else if reset > 0 {
				log.Info().Int64("reset", reset).Msg("Reset orphaned jobs to pending")
			}
		case <-w.shutdownChan:
			return
		}
	}
}

// progressTimeoutLoop monitors running jobs for progress timeouts
func (w *Worker) progressTimeoutLoop(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Check all running jobs for progress timeout
			filters := &JobFilters{
				WorkerID: &w.ID,
			}
			status := JobStatusRunning
			filters.Status = &status

			jobs, err := w.Storage.ListJobs(ctx, filters)
			if err != nil {
				log.Error().Err(err).Msg("Failed to list running jobs for progress check")
				continue
			}

			now := time.Now()
			for _, job := range jobs {
				if job.LastProgressAt == nil {
					continue
				}

				// Calculate timeout
				progressTimeout := time.Duration(w.Config.DefaultProgressTimeout)
				if job.ProgressTimeoutSeconds != nil {
					progressTimeout = time.Duration(*job.ProgressTimeoutSeconds) * time.Second
				}

				// Check if progress timeout exceeded
				if now.Sub(*job.LastProgressAt) > progressTimeout {
					log.Warn().
						Str("job_id", job.ID.String()).
						Str("job_name", job.JobName).
						Dur("timeout", progressTimeout).
						Dur("elapsed", now.Sub(*job.LastProgressAt)).
						Msg("Job progress timeout exceeded, cancelling")

					// Cancel the job
					w.cancelJob(job.ID)

					// Mark as failed
					errorMsg := fmt.Sprintf("Progress timeout exceeded (%v)", progressTimeout)
					if err := w.Storage.FailJob(ctx, job.ID, errorMsg); err != nil {
						log.Error().Err(err).Str("job_id", job.ID.String()).Msg("Failed to mark job as failed")
					}
				}
			}
		case <-w.shutdownChan:
			return
		}
	}
}

// executeJob executes a single job
func (w *Worker) executeJob(ctx context.Context, job *Job) {
	log.Info().
		Str("job_id", job.ID.String()).
		Str("job_name", job.JobName).
		Int("retry_count", job.RetryCount).
		Msg("Executing job")

	// Increment job count
	w.incrementJobCount()
	defer w.decrementJobCount()

	// Create cancel signal
	cancelSignal := runtime.NewCancelSignal()
	w.currentJobs.Store(job.ID, cancelSignal)
	defer w.currentJobs.Delete(job.ID)

	// Initialize log line counter
	lineCounter := 0
	w.jobLogCounters.Store(job.ID, &lineCounter)
	defer w.jobLogCounters.Delete(job.ID)

	// Store job start time for ETA calculation
	w.jobStartTimes.Store(job.ID, time.Now())
	defer w.jobStartTimes.Delete(job.ID)

	// Get job function
	var jobFunction *JobFunction
	var err error

	if job.JobFunctionID != nil {
		jobFunction, err = w.Storage.GetJobFunctionByID(ctx, *job.JobFunctionID)
		if err != nil {
			log.Error().Err(err).Str("job_id", job.ID.String()).Msg("Failed to get job function")
			_ = w.Storage.FailJob(ctx, job.ID, fmt.Sprintf("Job function not found: %v", err))
			return
		}
	} else {
		// Try to get by name and namespace
		jobFunction, err = w.Storage.GetJobFunction(ctx, job.Namespace, job.JobName)
		if err != nil {
			log.Error().Err(err).Str("job_id", job.ID.String()).Msg("Failed to get job function")
			_ = w.Storage.FailJob(ctx, job.ID, fmt.Sprintf("Job function not found: %v", err))
			return
		}
	}

	// Track whether execution logs are disabled for this job
	if jobFunction.DisableExecutionLogs {
		w.jobLogsDisabled.Store(job.ID, true)
		defer w.jobLogsDisabled.Delete(job.ID)
	}

	// Check if job function is enabled
	if !jobFunction.Enabled {
		log.Warn().Str("job_id", job.ID.String()).Str("job_name", job.JobName).Msg("Job function is disabled")
		_ = w.Storage.FailJob(ctx, job.ID, "Job function is disabled")
		return
	}

	// Check if code is available
	if jobFunction.Code == nil || *jobFunction.Code == "" {
		log.Error().Str("job_id", job.ID.String()).Msg("Job function has no code")
		_ = w.Storage.FailJob(ctx, job.ID, "Job function has no code")
		return
	}

	// Build permissions
	permissions := runtime.Permissions{
		AllowNet:      jobFunction.AllowNet,
		AllowEnv:      jobFunction.AllowEnv,
		AllowRead:     jobFunction.AllowRead,
		AllowWrite:    jobFunction.AllowWrite,
		MemoryLimitMB: jobFunction.MemoryLimitMB,
	}

	// Build execution request from job
	execReq := jobToExecutionRequest(job, w.publicURL)

	// Build timeout override from job settings
	var timeoutOverride *time.Duration
	if job.MaxDurationSeconds != nil && *job.MaxDurationSeconds > 0 {
		timeout := time.Duration(*job.MaxDurationSeconds) * time.Second
		timeoutOverride = &timeout
	}

	// Load secrets for job's namespace
	var jobSecrets map[string]string
	if w.SecretsStorage != nil {
		var err error
		jobSecrets, err = w.SecretsStorage.GetSecretsForNamespace(ctx, job.Namespace)
		if err != nil {
			log.Warn().Err(err).Str("namespace", job.Namespace).Msg("Failed to load secrets for job execution")
			// Continue without secrets - don't fail the job
		}
	}

	// Load settings secrets (user-specific and system-level)
	// These are injected as FLUXBASE_USER_* and FLUXBASE_SETTING_* env vars
	settingsSecrets := w.loadSettingsSecrets(ctx, job.CreatedBy)

	// Merge all secrets: job secrets first, then settings secrets (which include the env var prefix already)
	allSecrets := make(map[string]string)
	for k, v := range jobSecrets {
		allSecrets[k] = v
	}
	for k, v := range settingsSecrets {
		allSecrets[k] = v
	}

	// Execute job in runtime
	result, err := w.Runtime.Execute(ctx, *jobFunction.Code, execReq, permissions, cancelSignal, timeoutOverride, allSecrets)

	// Check if job was cancelled
	if cancelSignal.IsCancelled() {
		log.Info().Str("job_id", job.ID.String()).Msg("Job was cancelled")
		return // Already marked as cancelled in database
	}

	// Handle result
	if err != nil || !result.Success {
		errorMsg := result.Error
		if err != nil {
			errorMsg = err.Error()
		}

		log.Error().
			Str("job_id", job.ID.String()).
			Str("error", errorMsg).
			Msg("Job execution failed")

		// Check if should retry
		if job.RetryCount < job.MaxRetries {
			log.Info().
				Str("job_id", job.ID.String()).
				Int("retry_count", job.RetryCount).
				Int("max_retries", job.MaxRetries).
				Msg("Requeueing job for retry")

			if err := w.Storage.RequeueJob(ctx, job.ID); err != nil {
				log.Error().Err(err).Str("job_id", job.ID.String()).Msg("Failed to requeue job")
				_ = w.Storage.FailJob(ctx, job.ID, errorMsg)
			}
		} else {
			// Max retries reached, mark as failed
			_ = w.Storage.FailJob(ctx, job.ID, errorMsg)
		}
		return
	}

	// Job succeeded
	resultJSON := "{}"
	if result.Result != nil {
		resultBytes, _ := json.Marshal(result.Result)
		resultJSON = string(resultBytes)
	}

	if err := w.Storage.CompleteJob(ctx, job.ID, resultJSON); err != nil {
		log.Error().Err(err).Str("job_id", job.ID.String()).Msg("Failed to mark job as completed")
	} else {
		log.Info().
			Str("job_id", job.ID.String()).
			Str("job_name", job.JobName).
			Int64("duration_ms", result.DurationMs).
			Msg("Job completed successfully")
	}
}

// handleProgressUpdate is called when a job reports progress
func (w *Worker) handleProgressUpdate(jobID uuid.UUID, progress *runtime.Progress) {
	// Calculate ETA if we have valid progress (between 1-99%)
	if progress.Percent > 0 && progress.Percent < 100 {
		if startTimeVal, ok := w.jobStartTimes.Load(jobID); ok {
			startTime, ok := startTimeVal.(time.Time)
			if !ok {
				log.Warn().Str("job_id", jobID.String()).Msg("Invalid start time type in progress calculation")
			} else {
				elapsed := time.Since(startTime).Seconds()
				if elapsed > 0 {
					// ETA = (elapsed / percent) * remaining_percent
					remainingPercent := float64(100 - progress.Percent)
					etaSeconds := int((elapsed / float64(progress.Percent)) * remainingPercent)
					progress.EstimatedSecondsLeft = &etaSeconds
				}
			}
		}
	}

	log.Debug().
		Str("job_id", jobID.String()).
		Int("percent", progress.Percent).
		Str("message", progress.Message).
		Msg("Job progress update")

	// Convert progress to JSON
	progressJSON, err := json.Marshal(progress)
	if err != nil {
		log.Error().Err(err).Msg("Failed to marshal progress")
		return
	}

	// Update in database with a short timeout to avoid blocking on slow DB
	// Using a timeout context instead of job context since progress updates
	// are async and should complete even if job is finishing
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := w.Storage.UpdateJobProgress(ctx, jobID, string(progressJSON)); err != nil {
		log.Error().Err(err).Str("job_id", jobID.String()).Msg("Failed to update job progress")
	}
}

// handleLogMessage is called when a job outputs a log message
// Note: Execution logs are now stored in the central logging schema (logging.entries)
func (w *Worker) handleLogMessage(jobID uuid.UUID, level string, message string) {
	// Check if execution logs are disabled for this job
	if _, disabled := w.jobLogsDisabled.Load(jobID); disabled {
		return
	}

	// Get and increment the line counter for this job
	counterVal, ok := w.jobLogCounters.Load(jobID)
	if !ok {
		log.Warn().Str("job_id", jobID.String()).Msg("Log counter not found for job")
		return
	}
	counterPtr, ok := counterVal.(*int)
	if !ok {
		log.Warn().Str("job_id", jobID.String()).Msg("Invalid log counter type for job")
		return
	}
	lineNumber := *counterPtr
	*counterPtr = lineNumber + 1

	// Log to zerolog - central logging service will capture this via execution_id field
	log.Info().
		Str("execution_id", jobID.String()).
		Str("execution_type", "job").
		Str("level", level).
		Int("line_number", lineNumber).
		Msg(message)
}

// cancelJob cancels a running job
func (w *Worker) cancelJob(jobID uuid.UUID) {
	if signal, ok := w.currentJobs.Load(jobID); ok {
		if cancelSignal, ok := signal.(*runtime.CancelSignal); ok {
			cancelSignal.Cancel()
			log.Info().Str("job_id", jobID.String()).Msg("Job cancelled")
		}
	}
}

// cancelAllJobs cancels all running jobs
func (w *Worker) cancelAllJobs() {
	w.currentJobs.Range(func(key, value interface{}) bool {
		if cancelSignal, ok := value.(*runtime.CancelSignal); ok {
			cancelSignal.Cancel()
		}
		return true
	})
}

// interruptAllJobs cancels all running jobs and marks them as interrupted.
// This is called during graceful shutdown timeout, marking jobs as "interrupted"
// rather than "failed" so they can be distinguished from actual failures.
func (w *Worker) interruptAllJobs() {
	ctx := context.Background()
	reason := "Worker shutdown timeout - job interrupted"

	w.currentJobs.Range(func(key, value interface{}) bool {
		jobID, ok := key.(uuid.UUID)
		if !ok {
			return true
		}

		// Cancel the job execution
		if cancelSignal, ok := value.(*runtime.CancelSignal); ok {
			cancelSignal.Cancel()
		}

		// Mark the job as interrupted in the database
		if err := w.Storage.InterruptJob(ctx, jobID, reason); err != nil {
			log.Warn().
				Err(err).
				Str("job_id", jobID.String()).
				Msg("Failed to mark job as interrupted, will be marked as failed by cleanup")
		} else {
			log.Info().
				Str("job_id", jobID.String()).
				Msg("Job marked as interrupted due to worker shutdown")
		}

		return true
	})
}

// hasCapacity returns true if the worker can accept more jobs
func (w *Worker) hasCapacity() bool {
	return w.getCurrentJobCount() < w.MaxConcurrent
}

// getCurrentJobCount returns the current number of jobs being executed
func (w *Worker) getCurrentJobCount() int {
	w.currentJobCountMutex.RLock()
	defer w.currentJobCountMutex.RUnlock()
	return w.currentJobCount
}

// incrementJobCount increments the job count
func (w *Worker) incrementJobCount() {
	w.currentJobCountMutex.Lock()
	defer w.currentJobCountMutex.Unlock()
	w.currentJobCount++
}

// decrementJobCount decrements the job count
func (w *Worker) decrementJobCount() {
	w.currentJobCountMutex.Lock()
	defer w.currentJobCountMutex.Unlock()
	w.currentJobCount--
}

// loadSettingsSecrets loads settings secrets (user-specific and system-level) for job execution.
// Returns a map of environment variable name -> decrypted value.
// User secrets use prefix FLUXBASE_USER_, system secrets use prefix FLUXBASE_SETTING_.
func (w *Worker) loadSettingsSecrets(ctx context.Context, userID *uuid.UUID) map[string]string {
	if w.SettingsSecretsService == nil {
		return nil
	}

	envVars := make(map[string]string)

	// Load system-level settings secrets
	systemSecrets, err := w.SettingsSecretsService.GetSystemSecrets(ctx)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to load system settings secrets for job execution")
	} else {
		for key, value := range systemSecrets {
			envName := "FLUXBASE_SETTING_" + normalizeSettingsKey(key)
			envVars[envName] = value
		}
	}

	// Load user-specific settings secrets (if user is authenticated)
	if userID != nil {
		userSecrets, err := w.SettingsSecretsService.GetUserSecrets(ctx, *userID)
		if err != nil {
			log.Warn().Err(err).Str("user_id", userID.String()).Msg("Failed to load user settings secrets for job execution")
		} else {
			for key, value := range userSecrets {
				envName := "FLUXBASE_USER_" + normalizeSettingsKey(key)
				envVars[envName] = value
			}
		}
	}

	return envVars
}

// normalizeSettingsKey converts a settings key to an environment variable suffix.
// Example: "openai_api_key" -> "OPENAI_API_KEY", "ai.openai.api_key" -> "AI_OPENAI_API_KEY"
func normalizeSettingsKey(key string) string {
	// Replace dots with underscores, then uppercase
	normalized := strings.ReplaceAll(key, ".", "_")
	return strings.ToUpper(normalized)
}

// jobToExecutionRequest converts a Job to a runtime.ExecutionRequest
func jobToExecutionRequest(job *Job, publicURL string) runtime.ExecutionRequest {
	req := runtime.ExecutionRequest{
		ID:         job.ID,
		Name:       job.JobName,
		Namespace:  job.Namespace,
		RetryCount: job.RetryCount,
		BaseURL:    publicURL,
	}

	// Parse payload if present
	if job.Payload != nil {
		var payload map[string]interface{}
		if err := json.Unmarshal([]byte(*job.Payload), &payload); err == nil {
			req.Payload = payload
		}
	}

	// Add user context if available
	if job.CreatedBy != nil {
		req.UserID = job.CreatedBy.String()
	}
	if job.UserEmail != nil {
		req.UserEmail = *job.UserEmail
	}
	if job.UserRole != nil {
		req.UserRole = *job.UserRole
	}

	return req
}
