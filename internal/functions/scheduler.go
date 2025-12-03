package functions

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/fluxbase-eu/fluxbase/internal/database"
	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog/log"
)

// Scheduler manages scheduled execution of edge functions via cron
type Scheduler struct {
	cron          *cron.Cron
	storage       *Storage
	runtime       *DenoRuntime
	maxConcurrent int
	activeMu      sync.Mutex
	activeCount   int
	functionJobs  map[string]cron.EntryID // function name -> cron entry ID
	jobsMu        sync.RWMutex
	ctx           context.Context
	cancel        context.CancelFunc
}

// NewScheduler creates a new scheduler for edge functions
func NewScheduler(db *database.Connection) *Scheduler {
	ctx, cancel := context.WithCancel(context.Background())

	return &Scheduler{
		cron:          cron.New(cron.WithSeconds()),
		storage:       NewStorage(db),
		runtime:       NewDenoRuntime(),
		maxConcurrent: 10,
		functionJobs:  make(map[string]cron.EntryID),
		ctx:           ctx,
		cancel:        cancel,
	}
}

// Start initializes the scheduler and loads all enabled cron functions
func (s *Scheduler) Start() error {
	log.Info().Msg("Starting edge functions scheduler")

	// Load all functions with cron schedules
	functions, err := s.storage.ListFunctions(s.ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to load functions for scheduler")
		return err
	}

	// Schedule each function that has a cron schedule
	for _, fn := range functions {
		if fn.Enabled && fn.CronSchedule != nil && *fn.CronSchedule != "" {
			if err := s.ScheduleFunction(fn); err != nil {
				log.Error().
					Err(err).
					Str("function", fn.Name).
					Str("schedule", *fn.CronSchedule).
					Msg("Failed to schedule function")
			}
		}
	}

	// Start the cron scheduler
	s.cron.Start()
	log.Info().Int("scheduled_functions", len(s.functionJobs)).Msg("Edge functions scheduler started")

	return nil
}

// Stop gracefully shuts down the scheduler
func (s *Scheduler) Stop() {
	log.Info().Msg("Stopping edge functions scheduler")
	s.cancel()

	// Stop accepting new jobs
	ctx := s.cron.Stop()

	// Wait for running jobs to complete (with timeout)
	select {
	case <-ctx.Done():
		log.Info().Msg("All scheduled functions completed")
	case <-time.After(30 * time.Second):
		log.Warn().Msg("Scheduler shutdown timeout - some functions may not have completed")
	}
}

// ScheduleFunction adds or updates a function's cron schedule
func (s *Scheduler) ScheduleFunction(fn EdgeFunctionSummary) error {
	if fn.CronSchedule == nil || *fn.CronSchedule == "" {
		return nil
	}

	s.jobsMu.Lock()
	defer s.jobsMu.Unlock()

	// Remove existing schedule if present
	if existingID, exists := s.functionJobs[fn.Name]; exists {
		s.cron.Remove(existingID)
		delete(s.functionJobs, fn.Name)
		log.Debug().Str("function", fn.Name).Msg("Removed existing cron schedule")
	}

	// Capture only name and namespace - fetch fresh data at execution time
	funcName := fn.Name
	funcNamespace := fn.Namespace

	// Add new schedule
	entryID, err := s.cron.AddFunc(*fn.CronSchedule, func() {
		s.executeScheduledFunction(funcName, funcNamespace)
	})

	if err != nil {
		log.Error().
			Err(err).
			Str("function", fn.Name).
			Str("schedule", *fn.CronSchedule).
			Msg("Failed to add cron schedule")
		return err
	}

	s.functionJobs[fn.Name] = entryID
	log.Info().
		Str("function", fn.Name).
		Str("schedule", *fn.CronSchedule).
		Uint("entry_id", uint(entryID)).
		Msg("Function scheduled successfully")

	return nil
}

// UnscheduleFunction removes a function's cron schedule
func (s *Scheduler) UnscheduleFunction(functionName string) {
	s.jobsMu.Lock()
	defer s.jobsMu.Unlock()

	if entryID, exists := s.functionJobs[functionName]; exists {
		s.cron.Remove(entryID)
		delete(s.functionJobs, functionName)
		log.Info().Str("function", functionName).Msg("Function unscheduled")
	}
}

// RescheduleFunction updates a function's schedule (helper method)
func (s *Scheduler) RescheduleFunction(fn EdgeFunctionSummary) error {
	s.UnscheduleFunction(fn.Name)
	if fn.Enabled && fn.CronSchedule != nil && *fn.CronSchedule != "" {
		return s.ScheduleFunction(fn)
	}
	return nil
}

// executeScheduledFunction executes a function triggered by cron
func (s *Scheduler) executeScheduledFunction(funcName, funcNamespace string) {
	// Check concurrent execution limit
	s.activeMu.Lock()
	if s.activeCount >= s.maxConcurrent {
		s.activeMu.Unlock()
		log.Warn().
			Str("function", funcName).
			Int("active", s.activeCount).
			Int("max", s.maxConcurrent).
			Msg("Skipping scheduled execution - concurrent limit reached")
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

	// Fetch the current function data from storage
	fn, err := s.storage.GetFunctionByNamespace(s.ctx, funcName, funcNamespace)
	if err != nil {
		log.Error().
			Err(err).
			Str("function", funcName).
			Str("namespace", funcNamespace).
			Msg("Failed to fetch function for scheduled execution")
		return
	}

	// Check if function is still enabled
	if !fn.Enabled {
		log.Debug().
			Str("function", funcName).
			Msg("Skipping scheduled execution - function is disabled")
		return
	}

	log.Info().
		Str("function", fn.Name).
		Str("trigger", "cron").
		Msg("Executing scheduled function")

	start := time.Now()

	// Create execution record
	exec := &EdgeFunctionExecution{
		FunctionID:  fn.ID,
		TriggerType: "cron",
		Status:      "running",
		ExecutedAt:  start,
	}

	// Prepare execution request (empty for cron triggers)
	req := ExecutionRequest{
		Method:  "POST",
		URL:     "/scheduled",
		Headers: make(map[string]string),
		Body:    "{}",
	}

	// Build permissions from function config
	perms := Permissions{
		AllowNet:   fn.AllowNet,
		AllowEnv:   fn.AllowEnv,
		AllowRead:  fn.AllowRead,
		AllowWrite: fn.AllowWrite,
	}

	// Execute with timeout
	ctx, cancel := context.WithTimeout(s.ctx, time.Duration(fn.TimeoutSeconds)*time.Second)
	defer cancel()

	result, err := s.runtime.Execute(ctx, fn.Code, req, perms)
	duration := time.Since(start)

	// Update execution record
	completedAt := time.Now()
	exec.CompletedAt = &completedAt
	durationMs := int(duration.Milliseconds())
	exec.DurationMs = &durationMs

	if err != nil {
		exec.Status = "error"
		errorMsg := err.Error()
		exec.ErrorMessage = &errorMsg
		log.Error().
			Err(err).
			Str("function", fn.Name).
			Dur("duration", duration).
			Msg("Scheduled function execution failed")
	} else {
		if result.Error != "" {
			exec.Status = "error"
			exec.ErrorMessage = &result.Error
		} else {
			exec.Status = "success"
		}
		exec.StatusCode = &result.Status

		// Serialize result to JSON
		if resultJSON, err := json.Marshal(result); err == nil {
			resultStr := string(resultJSON)
			exec.Result = &resultStr
		}
		exec.Logs = &result.Logs

		log.Info().
			Str("function", fn.Name).
			Str("status", exec.Status).
			Int("status_code", result.Status).
			Dur("duration", duration).
			Msg("Scheduled function execution completed")
	}

	// Save execution log asynchronously
	go func() {
		if err := s.storage.LogExecution(context.Background(), exec); err != nil {
			log.Error().
				Err(err).
				Str("function", fn.Name).
				Msg("Failed to log scheduled execution")
		}
	}()
}

// GetScheduledFunctions returns a list of all currently scheduled functions
func (s *Scheduler) GetScheduledFunctions() []string {
	s.jobsMu.RLock()
	defer s.jobsMu.RUnlock()

	functions := make([]string, 0, len(s.functionJobs))
	for name := range s.functionJobs {
		functions = append(functions, name)
	}
	return functions
}

// GetScheduleInfo returns schedule information for a function
func (s *Scheduler) GetScheduleInfo(functionName string) (string, bool) {
	s.jobsMu.RLock()
	defer s.jobsMu.RUnlock()

	entryID, exists := s.functionJobs[functionName]
	if !exists {
		return "", false
	}

	entry := s.cron.Entry(entryID)
	if entry.Next.IsZero() {
		return "Not scheduled", true
	}

	return entry.Next.Format(time.RFC3339), true
}
