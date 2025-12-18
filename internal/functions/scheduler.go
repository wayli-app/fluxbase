package functions

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/fluxbase-eu/fluxbase/internal/database"
	"github.com/fluxbase-eu/fluxbase/internal/runtime"
	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog/log"
)

// Scheduler manages scheduled execution of edge functions via cron
type Scheduler struct {
	cron          *cron.Cron
	storage       *Storage
	runtime       *runtime.DenoRuntime
	maxConcurrent int
	activeMu      sync.Mutex
	activeCount   int
	functionJobs  map[string]cron.EntryID // function name -> cron entry ID
	jobsMu        sync.RWMutex
	ctx           context.Context
	cancel        context.CancelFunc
	jwtSecret     string
	publicURL     string
	logCounters   sync.Map // map[uuid.UUID]*int for tracking log line numbers per execution
}

// NewScheduler creates a new scheduler for edge functions
func NewScheduler(db *database.Connection, jwtSecret, publicURL string) *Scheduler {
	ctx, cancel := context.WithCancel(context.Background())

	// Use a parser that supports both standard 5-field cron expressions
	// and 6-field expressions with optional seconds
	// 5-field: "*/5 * * * *" (every 5 minutes)
	// 6-field: "0 */5 * * * *" (every 5 minutes at second 0)
	parser := cron.NewParser(
		cron.SecondOptional | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor,
	)

	s := &Scheduler{
		cron:          cron.New(cron.WithParser(parser)),
		storage:       NewStorage(db),
		runtime:       runtime.NewRuntime(runtime.RuntimeTypeFunction, jwtSecret, publicURL),
		maxConcurrent: 10,
		functionJobs:  make(map[string]cron.EntryID),
		ctx:           ctx,
		cancel:        cancel,
		jwtSecret:     jwtSecret,
		publicURL:     publicURL,
	}

	// Set up log callback to capture console.log output
	s.runtime.SetLogCallback(s.handleLogMessage)

	return s
}

// handleLogMessage is called when a scheduled function outputs a log message
func (s *Scheduler) handleLogMessage(executionID uuid.UUID, level string, message string) {
	// Get and increment the line counter for this execution
	counterVal, ok := s.logCounters.Load(executionID)
	if !ok {
		log.Debug().
			Str("execution_id", executionID.String()).
			Str("level", level).
			Str("message", message).
			Msg("Scheduled function log (no counter)")
		return
	}

	counterPtr, ok := counterVal.(*int)
	if !ok {
		log.Warn().Str("execution_id", executionID.String()).Msg("Invalid log counter type")
		return
	}

	lineNumber := *counterPtr
	*counterPtr = lineNumber + 1

	// Insert log line into database
	ctx := context.Background()
	if err := s.storage.AppendExecutionLog(ctx, executionID, lineNumber, level, message); err != nil {
		log.Error().Err(err).Str("execution_id", executionID.String()).Msg("Failed to insert execution log")
	}
}

// Start initializes the scheduler and loads all enabled cron functions
// It runs asynchronously to avoid blocking server startup and retries on database errors
func (s *Scheduler) Start() error {
	log.Info().Msg("Starting edge functions scheduler")

	// Start the cron scheduler immediately
	s.cron.Start()

	// Load functions asynchronously with retry logic to handle race conditions during startup
	go func() {
		maxRetries := 5
		retryDelay := 100 * time.Millisecond

		for attempt := 1; attempt <= maxRetries; attempt++ {
			// Use a timeout context to prevent indefinite hanging if database pool has issues
			ctx, cancel := context.WithTimeout(s.ctx, 10*time.Second)

			functions, err := s.storage.ListFunctions(ctx)
			cancel()

			if err != nil {
				if attempt < maxRetries {
					log.Debug().
						Err(err).
						Int("attempt", attempt).
						Int("max_retries", maxRetries).
						Dur("retry_delay", retryDelay).
						Msg("Failed to load functions for scheduler, retrying")
					time.Sleep(retryDelay)
					retryDelay *= 2 // Exponential backoff
					continue
				}
				log.Error().Err(err).Msg("Failed to load functions for scheduler after all retries")
				return
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

			log.Info().Int("scheduled_functions", len(s.functionJobs)).Msg("Edge functions scheduler started successfully")
			return
		}
	}()

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

	// Prepare execution request (empty for cron triggers)
	executionID := uuid.New()
	req := runtime.ExecutionRequest{
		ID:        executionID,
		Name:      fn.Name,
		Namespace: fn.Namespace,
		Method:    "POST",
		URL:       "/scheduled",
		Headers:   make(map[string]string),
		Body:      "{}",
	}

	// Create execution record BEFORE running to enable real-time logging
	if err := s.storage.CreateExecution(s.ctx, executionID, fn.ID, "cron"); err != nil {
		log.Error().Err(err).Str("execution_id", executionID.String()).Msg("Failed to create execution record")
		// Continue anyway - logging will still work via stderr fallback
	}

	// Initialize log counter for this execution
	lineCounter := 0
	s.logCounters.Store(executionID, &lineCounter)
	defer s.logCounters.Delete(executionID)

	// Build permissions from function config
	perms := runtime.Permissions{
		AllowNet:   fn.AllowNet,
		AllowEnv:   fn.AllowEnv,
		AllowRead:  fn.AllowRead,
		AllowWrite: fn.AllowWrite,
	}

	// Build timeout override from function settings
	var timeoutOverride *time.Duration
	if fn.TimeoutSeconds > 0 {
		timeout := time.Duration(fn.TimeoutSeconds) * time.Second
		timeoutOverride = &timeout
	}

	// Execute (nil cancel signal for scheduled executions)
	result, err := s.runtime.Execute(s.ctx, fn.Code, req, perms, nil, timeoutOverride)
	duration := time.Since(start)

	// Determine final status
	status := "success"
	var errorMessage *string
	durationMs := int(duration.Milliseconds())

	if err != nil {
		status = "error"
		errorMsg := err.Error()
		errorMessage = &errorMsg
		log.Error().
			Err(err).
			Str("function", fn.Name).
			Dur("duration", duration).
			Msg("Scheduled function execution failed")
	} else {
		if result.Error != "" {
			status = "error"
			errorMessage = &result.Error
		}
		log.Info().
			Str("function", fn.Name).
			Str("status", status).
			Int("status_code", result.Status).
			Dur("duration", duration).
			Msg("Scheduled function execution completed")
	}

	// Serialize result to JSON
	var resultStr *string
	if result != nil {
		if resultJSON, jsonErr := json.Marshal(result); jsonErr == nil {
			rs := string(resultJSON)
			resultStr = &rs
		}
	}

	// Complete execution record asynchronously
	go func() {
		if updateErr := s.storage.CompleteExecution(context.Background(), executionID, status, &result.Status, &durationMs, resultStr, &result.Logs, errorMessage); updateErr != nil {
			log.Error().
				Err(updateErr).
				Str("function", fn.Name).
				Str("execution_id", executionID.String()).
				Msg("Failed to complete scheduled execution record")
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
