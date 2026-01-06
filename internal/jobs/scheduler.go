package jobs

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/fluxbase-eu/fluxbase/internal/database"
	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog/log"
)

// ScheduleConfig contains schedule configuration including run parameters
type ScheduleConfig struct {
	CronExpression string                 `json:"cron_expression"`
	Params         map[string]interface{} `json:"params,omitempty"`
}

// Scheduler manages scheduled execution of jobs via cron
type Scheduler struct {
	cron          *cron.Cron
	storage       *Storage
	maxConcurrent int
	activeMu      sync.Mutex
	activeCount   int
	jobEntries    map[string]cron.EntryID // job function name -> cron entry ID
	jobsMu        sync.RWMutex
	ctx           context.Context
	cancel        context.CancelFunc
}

// NewScheduler creates a new scheduler for jobs
func NewScheduler(db *database.Connection) *Scheduler {
	ctx, cancel := context.WithCancel(context.Background())

	// Use a parser that supports both standard 5-field cron expressions
	// and 6-field expressions with optional seconds
	// 5-field: "*/5 * * * *" (every 5 minutes)
	// 6-field: "0 */5 * * * *" (every 5 minutes at second 0)
	parser := cron.NewParser(
		cron.SecondOptional | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor,
	)

	return &Scheduler{
		cron:          cron.New(cron.WithParser(parser)),
		storage:       NewStorage(db),
		maxConcurrent: 20, // Max concurrent scheduled job submissions
		jobEntries:    make(map[string]cron.EntryID),
		ctx:           ctx,
		cancel:        cancel,
	}
}

// Start initializes the scheduler and loads all enabled scheduled jobs
// It runs asynchronously to avoid blocking server startup and retries on database errors
func (s *Scheduler) Start() error {
	log.Info().Msg("Starting job scheduler")

	// Start the cron scheduler immediately
	s.cron.Start()

	// Load jobs asynchronously with retry logic to handle race conditions during startup
	go func() {
		maxRetries := 5
		retryDelay := 100 * time.Millisecond

		for attempt := 1; attempt <= maxRetries; attempt++ {
			// Load all namespaces to get all job functions
			namespaces, err := s.storage.ListJobNamespaces(s.ctx)
			if err != nil {
				if attempt < maxRetries {
					log.Debug().
						Err(err).
						Int("attempt", attempt).
						Int("max_retries", maxRetries).
						Dur("retry_delay", retryDelay).
						Msg("Failed to list job namespaces for scheduler, retrying")
					time.Sleep(retryDelay)
					retryDelay *= 2 // Exponential backoff
					continue
				}
				log.Error().Err(err).Msg("Failed to list job namespaces for scheduler after all retries")
				return
			}

			// If no namespaces exist, try with "default"
			if len(namespaces) == 0 {
				namespaces = []string{"default"}
			}

			// Load all job functions from each namespace
			for _, ns := range namespaces {
				functions, err := s.storage.ListJobFunctions(s.ctx, ns)
				if err != nil {
					log.Error().Err(err).Str("namespace", ns).Msg("Failed to load job functions for scheduler")
					continue
				}

				// Schedule each function that has a cron schedule
				for _, fn := range functions {
					if fn.Enabled && fn.Schedule != nil && *fn.Schedule != "" {
						if err := s.ScheduleJob(fn); err != nil {
							log.Error().
								Err(err).
								Str("job", fn.Name).
								Str("schedule", *fn.Schedule).
								Msg("Failed to schedule job")
						}
					}
				}
			}

			log.Info().Int("scheduled_jobs", len(s.jobEntries)).Msg("Job scheduler started successfully")
			return
		}
	}()

	return nil
}

// Stop gracefully shuts down the scheduler
func (s *Scheduler) Stop() {
	log.Info().Msg("Stopping job scheduler")
	s.cancel()

	// Stop accepting new jobs
	ctx := s.cron.Stop()

	// Wait for scheduled submissions to complete (with timeout)
	select {
	case <-ctx.Done():
		log.Info().Msg("All scheduled job submissions completed")
	case <-time.After(30 * time.Second):
		log.Warn().Msg("Scheduler shutdown timeout - some submissions may not have completed")
	}
}

// ScheduleJob adds or updates a job function's cron schedule
func (s *Scheduler) ScheduleJob(fn *JobFunctionSummary) error {
	if fn.Schedule == nil || *fn.Schedule == "" {
		return nil
	}

	// Parse the schedule which may include params
	scheduleConfig := s.parseScheduleConfig(*fn.Schedule)

	s.jobsMu.Lock()
	defer s.jobsMu.Unlock()

	// Create a unique key for the job
	jobKey := fn.Namespace + "/" + fn.Name

	// Remove existing schedule if present
	if existingID, exists := s.jobEntries[jobKey]; exists {
		s.cron.Remove(existingID)
		delete(s.jobEntries, jobKey)
		log.Debug().Str("job", fn.Name).Msg("Removed existing cron schedule")
	}

	// Capture job details for the closure
	jobName := fn.Name
	jobNamespace := fn.Namespace
	scheduleParams := scheduleConfig.Params

	// Add new schedule
	entryID, err := s.cron.AddFunc(scheduleConfig.CronExpression, func() {
		s.enqueueScheduledJob(jobName, jobNamespace, scheduleParams)
	})

	if err != nil {
		log.Error().
			Err(err).
			Str("job", fn.Name).
			Str("schedule", scheduleConfig.CronExpression).
			Msg("Failed to add cron schedule")
		return err
	}

	s.jobEntries[jobKey] = entryID
	log.Info().
		Str("job", fn.Name).
		Str("namespace", jobNamespace).
		Str("schedule", scheduleConfig.CronExpression).
		Interface("params", scheduleParams).
		Uint("entry_id", uint(entryID)).
		Msg("Job scheduled successfully")

	return nil
}

// ScheduleJobFunction schedules a full JobFunction (not just summary)
func (s *Scheduler) ScheduleJobFunction(fn *JobFunction) error {
	summary := &JobFunctionSummary{
		ID:                     fn.ID,
		Name:                   fn.Name,
		Namespace:              fn.Namespace,
		Enabled:                fn.Enabled,
		Schedule:               fn.Schedule,
		TimeoutSeconds:         fn.TimeoutSeconds,
		MemoryLimitMB:          fn.MemoryLimitMB,
		MaxRetries:             fn.MaxRetries,
		ProgressTimeoutSeconds: fn.ProgressTimeoutSeconds,
		AllowNet:               fn.AllowNet,
		AllowEnv:               fn.AllowEnv,
		AllowRead:              fn.AllowRead,
		AllowWrite:             fn.AllowWrite,
		RequireRoles:           fn.RequireRoles,
		Source:                 fn.Source,
	}
	return s.ScheduleJob(summary)
}

// UnscheduleJob removes a job's cron schedule
func (s *Scheduler) UnscheduleJob(namespace, jobName string) {
	s.jobsMu.Lock()
	defer s.jobsMu.Unlock()

	jobKey := namespace + "/" + jobName

	if entryID, exists := s.jobEntries[jobKey]; exists {
		s.cron.Remove(entryID)
		delete(s.jobEntries, jobKey)
		log.Info().Str("job", jobName).Str("namespace", namespace).Msg("Job unscheduled")
	}
}

// RescheduleJob updates a job's schedule (helper method)
func (s *Scheduler) RescheduleJob(fn *JobFunctionSummary) error {
	s.UnscheduleJob(fn.Namespace, fn.Name)
	if fn.Enabled && fn.Schedule != nil && *fn.Schedule != "" {
		return s.ScheduleJob(fn)
	}
	return nil
}

// parseScheduleConfig parses a schedule string that may contain params
// Format: "cron_expression" or "cron_expression|params_json"
// Example: "0 2 * * *" or "0 2 * * *|{\"type\":\"daily\"}"
func (s *Scheduler) parseScheduleConfig(schedule string) ScheduleConfig {
	config := ScheduleConfig{
		CronExpression: schedule,
		Params:         make(map[string]interface{}),
	}

	// Check if schedule contains params separated by |
	for i := len(schedule) - 1; i >= 0; i-- {
		if schedule[i] == '|' {
			config.CronExpression = schedule[:i]
			paramsJSON := schedule[i+1:]
			if err := json.Unmarshal([]byte(paramsJSON), &config.Params); err != nil {
				log.Warn().
					Err(err).
					Str("params", paramsJSON).
					Msg("Failed to parse schedule params, using cron expression only")
				config.CronExpression = schedule
				config.Params = make(map[string]interface{})
			}
			break
		}
	}

	return config
}

// enqueueScheduledJob enqueues a job triggered by cron
func (s *Scheduler) enqueueScheduledJob(jobName, jobNamespace string, params map[string]interface{}) {
	// Check concurrent submission limit
	s.activeMu.Lock()
	if s.activeCount >= s.maxConcurrent {
		s.activeMu.Unlock()
		log.Warn().
			Str("job", jobName).
			Int("active", s.activeCount).
			Int("max", s.maxConcurrent).
			Msg("Skipping scheduled job submission - concurrent limit reached")
		return
	}
	s.activeCount++
	s.activeMu.Unlock()

	// Decrement counter when done
	defer func() {
		s.activeMu.Lock()
		s.activeCount--
		s.activeMu.Unlock()
	}()

	// Fetch the current job function data from storage
	fn, err := s.storage.GetJobFunction(s.ctx, jobNamespace, jobName)
	if err != nil {
		log.Error().
			Err(err).
			Str("job", jobName).
			Str("namespace", jobNamespace).
			Msg("Failed to fetch job function for scheduled execution")
		return
	}

	// Check if job is still enabled
	if !fn.Enabled {
		log.Debug().
			Str("job", jobName).
			Msg("Skipping scheduled job - function is disabled")
		return
	}

	log.Info().
		Str("job", fn.Name).
		Str("namespace", fn.Namespace).
		Str("trigger", "cron").
		Interface("params", params).
		Msg("Enqueuing scheduled job")

	// Prepare payload with schedule params and trigger info
	payload := map[string]interface{}{
		"_trigger":      "cron",
		"_scheduled_at": time.Now().UTC().Format(time.RFC3339),
	}

	// Merge in any schedule params
	for k, v := range params {
		payload[k] = v
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		log.Error().
			Err(err).
			Str("job", fn.Name).
			Msg("Failed to marshal scheduled job payload")
		return
	}
	payloadStr := string(payloadJSON)

	// Create job instance
	job := &Job{
		ID:                     uuid.New(),
		Namespace:              fn.Namespace,
		JobFunctionID:          &fn.ID,
		JobName:                fn.Name,
		Status:                 JobStatusPending,
		Payload:                &payloadStr,
		Priority:               0, // Default priority for scheduled jobs
		MaxRetries:             fn.MaxRetries,
		ProgressTimeoutSeconds: &fn.ProgressTimeoutSeconds,
		MaxDurationSeconds:     &fn.TimeoutSeconds,
		// No user context for scheduled jobs (system-triggered)
	}

	// Enqueue the job
	if err := s.storage.EnqueueJob(s.ctx, job); err != nil {
		log.Error().
			Err(err).
			Str("job", fn.Name).
			Msg("Failed to enqueue scheduled job")
		return
	}

	log.Info().
		Str("job", fn.Name).
		Str("job_id", job.ID.String()).
		Str("namespace", fn.Namespace).
		Msg("Scheduled job enqueued successfully")
}

// GetScheduledJobs returns a list of all currently scheduled jobs
func (s *Scheduler) GetScheduledJobs() []ScheduledJobInfo {
	s.jobsMu.RLock()
	defer s.jobsMu.RUnlock()

	jobs := make([]ScheduledJobInfo, 0, len(s.jobEntries))
	for key, entryID := range s.jobEntries {
		entry := s.cron.Entry(entryID)
		info := ScheduledJobInfo{
			Key:     key,
			EntryID: int(entryID),
			NextRun: entry.Next,
			PrevRun: entry.Prev,
		}
		jobs = append(jobs, info)
	}
	return jobs
}

// GetScheduleInfo returns schedule information for a job
func (s *Scheduler) GetScheduleInfo(namespace, jobName string) (string, bool) {
	s.jobsMu.RLock()
	defer s.jobsMu.RUnlock()

	jobKey := namespace + "/" + jobName
	entryID, exists := s.jobEntries[jobKey]
	if !exists {
		return "", false
	}

	entry := s.cron.Entry(entryID)
	if entry.Next.IsZero() {
		return "Not scheduled", true
	}

	return entry.Next.Format(time.RFC3339), true
}

// IsScheduled checks if a job is currently scheduled
func (s *Scheduler) IsScheduled(namespace, jobName string) bool {
	s.jobsMu.RLock()
	defer s.jobsMu.RUnlock()

	jobKey := namespace + "/" + jobName
	_, exists := s.jobEntries[jobKey]
	return exists
}

// ScheduledJobInfo contains information about a scheduled job
type ScheduledJobInfo struct {
	Key     string    `json:"key"`
	EntryID int       `json:"entry_id"`
	NextRun time.Time `json:"next_run"`
	PrevRun time.Time `json:"prev_run"`
}
