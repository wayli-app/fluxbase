package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/fluxbase-eu/fluxbase/internal/config"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// Worker executes jobs from the queue
type Worker struct {
	ID                    uuid.UUID
	Name                  string
	Config                *config.JobsConfig
	Storage               *Storage
	Runtime               *JobRuntime
	MaxConcurrent         int
	currentJobs           sync.Map // jobID -> *CancelSignal
	jobLogCounters        sync.Map // jobID -> *int (line counter)
	jobStartTimes         sync.Map // jobID -> time.Time (for ETA calculation)
	currentJobCount       int
	currentJobCountMutex  sync.RWMutex
	shutdownChan          chan struct{}
	shutdownComplete      chan struct{}
	progressTimeoutTicker *time.Ticker
}

// NewWorker creates a new worker
func NewWorker(cfg *config.JobsConfig, storage *Storage, jwtSecret, publicURL string) *Worker {
	workerID := uuid.New()
	hostname, _ := os.Hostname()

	// Create runtime with SDK credentials
	runtime := NewJobRuntime(
		cfg.DefaultMaxDuration,
		128, // Default memory limit
		jwtSecret,
		publicURL,
	)

	worker := &Worker{
		ID:               workerID,
		Name:             fmt.Sprintf("worker-%s@%s", workerID.String()[:8], hostname),
		Config:           cfg,
		Storage:          storage,
		Runtime:          runtime,
		MaxConcurrent:    cfg.MaxConcurrentPerWorker,
		shutdownChan:     make(chan struct{}),
		shutdownComplete: make(chan struct{}),
	}

	// Set up runtime callbacks
	runtime.SetProgressCallback(worker.handleProgressUpdate)
	runtime.SetLogCallback(worker.handleLogMessage)

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

	log.Info().Str("worker_id", w.ID.String()).Msg("Worker shutting down gracefully")

	// Wait for all jobs to complete (with timeout)
	shutdownTimeout := time.After(30 * time.Second)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-shutdownTimeout:
			log.Warn().
				Str("worker_id", w.ID.String()).
				Int("remaining_jobs", w.getCurrentJobCount()).
				Msg("Shutdown timeout reached, forcing shutdown")
			w.cancelAllJobs()
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
								w.Storage.FailJob(context.Background(), j.ID, fmt.Sprintf("Internal error: job execution panic: %v", rec))
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
	// Run cleanup every WorkerTimeout interval
	ticker := time.NewTicker(w.Config.WorkerTimeout)
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
	cancelSignal := NewCancelSignal()
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
			w.Storage.FailJob(ctx, job.ID, fmt.Sprintf("Job function not found: %v", err))
			return
		}
	} else {
		// Try to get by name and namespace
		jobFunction, err = w.Storage.GetJobFunction(ctx, job.Namespace, job.JobName)
		if err != nil {
			log.Error().Err(err).Str("job_id", job.ID.String()).Msg("Failed to get job function")
			w.Storage.FailJob(ctx, job.ID, fmt.Sprintf("Job function not found: %v", err))
			return
		}
	}

	// Check if job function is enabled
	if !jobFunction.Enabled {
		log.Warn().Str("job_id", job.ID.String()).Str("job_name", job.JobName).Msg("Job function is disabled")
		w.Storage.FailJob(ctx, job.ID, "Job function is disabled")
		return
	}

	// Check if code is available
	if jobFunction.Code == nil || *jobFunction.Code == "" {
		log.Error().Str("job_id", job.ID.String()).Msg("Job function has no code")
		w.Storage.FailJob(ctx, job.ID, "Job function has no code")
		return
	}

	// Build permissions
	permissions := Permissions{
		AllowNet:      jobFunction.AllowNet,
		AllowEnv:      jobFunction.AllowEnv,
		AllowRead:     jobFunction.AllowRead,
		AllowWrite:    jobFunction.AllowWrite,
		MemoryLimitMB: jobFunction.MemoryLimitMB,
	}

	// Execute job in runtime
	result, err := w.Runtime.Execute(ctx, job, *jobFunction.Code, permissions, cancelSignal)

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
				w.Storage.FailJob(ctx, job.ID, errorMsg)
			}
		} else {
			// Max retries reached, mark as failed
			w.Storage.FailJob(ctx, job.ID, errorMsg)
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
func (w *Worker) handleProgressUpdate(jobID uuid.UUID, progress *Progress) {
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

	// Update in database
	ctx := context.Background()
	if err := w.Storage.UpdateJobProgress(ctx, jobID, string(progressJSON)); err != nil {
		log.Error().Err(err).Str("job_id", jobID.String()).Msg("Failed to update job progress")
	}
}

// handleLogMessage is called when a job outputs a log message
func (w *Worker) handleLogMessage(jobID uuid.UUID, level string, message string) {
	log.Debug().
		Str("job_id", jobID.String()).
		Str("level", level).
		Str("message", message).
		Msg("Job log")

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

	// Insert log line
	ctx := context.Background()
	if err := w.Storage.InsertExecutionLog(ctx, jobID, lineNumber, level, message); err != nil {
		log.Error().Err(err).Str("job_id", jobID.String()).Msg("Failed to insert execution log")
	}
}

// cancelJob cancels a running job
func (w *Worker) cancelJob(jobID uuid.UUID) {
	if signal, ok := w.currentJobs.Load(jobID); ok {
		if cancelSignal, ok := signal.(*CancelSignal); ok {
			cancelSignal.Cancel()
			log.Info().Str("job_id", jobID.String()).Msg("Job cancelled")
		}
	}
}

// cancelAllJobs cancels all running jobs
func (w *Worker) cancelAllJobs() {
	w.currentJobs.Range(func(key, value interface{}) bool {
		if cancelSignal, ok := value.(*CancelSignal); ok {
			cancelSignal.Cancel()
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
