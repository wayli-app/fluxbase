package rpc

import (
	"context"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog/log"
)

// Scheduler manages scheduled execution of RPC procedures via cron
type Scheduler struct {
	cron          *cron.Cron
	storage       *Storage
	executor      *Executor
	maxConcurrent int
	activeMu      sync.Mutex
	activeCount   int
	procedureJobs map[string]cron.EntryID // "namespace/name" -> cron entry ID
	jobsMu        sync.RWMutex
	ctx           context.Context
	cancel        context.CancelFunc
}

// NewScheduler creates a new RPC scheduler
func NewScheduler(storage *Storage, executor *Executor) *Scheduler {
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
		storage:       storage,
		executor:      executor,
		maxConcurrent: 10, // Max concurrent scheduled RPC executions
		procedureJobs: make(map[string]cron.EntryID),
		ctx:           ctx,
		cancel:        cancel,
	}
}

// Start initializes the scheduler and loads all enabled scheduled procedures
// It runs asynchronously to avoid blocking server startup and retries on database errors
func (s *Scheduler) Start() error {
	log.Info().Msg("Starting RPC scheduler")

	// Start the cron scheduler immediately
	s.cron.Start()

	// Load procedures asynchronously with retry logic to handle race conditions during startup
	go func() {
		maxRetries := 5
		retryDelay := 100 * time.Millisecond

		for attempt := 1; attempt <= maxRetries; attempt++ {
			// Load all scheduled procedures
			procedures, err := s.storage.ListScheduledProcedures(s.ctx)
			if err != nil {
				if attempt < maxRetries {
					log.Debug().
						Err(err).
						Int("attempt", attempt).
						Int("max_retries", maxRetries).
						Dur("retry_delay", retryDelay).
						Msg("Failed to load scheduled procedures, retrying")
					time.Sleep(retryDelay)
					retryDelay *= 2 // Exponential backoff
					continue
				}
				log.Error().Err(err).Msg("Failed to load scheduled procedures after all retries")
				return
			}

			// Schedule each procedure
			for _, proc := range procedures {
				if proc.Schedule != nil && *proc.Schedule != "" {
					if err := s.ScheduleProcedure(proc); err != nil {
						log.Error().
							Err(err).
							Str("procedure", proc.Name).
							Str("schedule", *proc.Schedule).
							Msg("Failed to schedule procedure")
					}
				}
			}

			log.Info().Int("scheduled_procedures", len(s.procedureJobs)).Msg("RPC scheduler started successfully")
			return
		}
	}()

	return nil
}

// Stop gracefully shuts down the scheduler
func (s *Scheduler) Stop() {
	log.Info().Msg("Stopping RPC scheduler")
	s.cancel()

	ctx := s.cron.Stop()
	select {
	case <-ctx.Done():
		log.Info().Msg("All scheduled RPC executions completed")
	case <-time.After(30 * time.Second):
		log.Warn().Msg("RPC scheduler shutdown timeout")
	}
}

// ScheduleProcedure adds or updates a procedure's cron schedule
func (s *Scheduler) ScheduleProcedure(proc *Procedure) error {
	if proc.Schedule == nil || *proc.Schedule == "" {
		return nil
	}

	s.jobsMu.Lock()
	defer s.jobsMu.Unlock()

	procKey := proc.Namespace + "/" + proc.Name

	// Remove existing schedule if present
	if existingID, exists := s.procedureJobs[procKey]; exists {
		s.cron.Remove(existingID)
		delete(s.procedureJobs, procKey)
	}

	// Capture procedure details for closure
	procName := proc.Name
	procNamespace := proc.Namespace

	// Add new schedule
	entryID, err := s.cron.AddFunc(*proc.Schedule, func() {
		s.executeScheduledProcedure(procName, procNamespace)
	})
	if err != nil {
		return err
	}

	s.procedureJobs[procKey] = entryID
	log.Info().
		Str("procedure", proc.Name).
		Str("namespace", proc.Namespace).
		Str("schedule", *proc.Schedule).
		Msg("Procedure scheduled successfully")

	return nil
}

// UnscheduleProcedure removes a procedure's cron schedule
func (s *Scheduler) UnscheduleProcedure(namespace, name string) {
	s.jobsMu.Lock()
	defer s.jobsMu.Unlock()

	procKey := namespace + "/" + name
	if entryID, exists := s.procedureJobs[procKey]; exists {
		s.cron.Remove(entryID)
		delete(s.procedureJobs, procKey)
		log.Info().Str("procedure", name).Str("namespace", namespace).Msg("Procedure unscheduled")
	}
}

// RescheduleProcedure updates a procedure's schedule
func (s *Scheduler) RescheduleProcedure(proc *Procedure) error {
	s.UnscheduleProcedure(proc.Namespace, proc.Name)
	if proc.Enabled && proc.Schedule != nil && *proc.Schedule != "" {
		return s.ScheduleProcedure(proc)
	}
	return nil
}

// executeScheduledProcedure executes a procedure triggered by cron
func (s *Scheduler) executeScheduledProcedure(procName, procNamespace string) {
	// Check concurrent execution limit
	s.activeMu.Lock()
	if s.activeCount >= s.maxConcurrent {
		s.activeMu.Unlock()
		log.Warn().
			Str("procedure", procName).
			Msg("Skipping scheduled execution - concurrent limit reached")
		return
	}
	s.activeCount++
	s.activeMu.Unlock()

	defer func() {
		s.activeMu.Lock()
		s.activeCount--
		s.activeMu.Unlock()
	}()

	// Fetch current procedure data
	proc, err := s.storage.GetProcedureByName(s.ctx, procNamespace, procName)
	if err != nil || proc == nil {
		log.Error().Err(err).Str("procedure", procName).Msg("Failed to fetch procedure for scheduled execution")
		return
	}

	if !proc.Enabled {
		log.Debug().Str("procedure", procName).Msg("Skipping scheduled execution - procedure disabled")
		return
	}

	log.Info().
		Str("procedure", proc.Name).
		Str("namespace", proc.Namespace).
		Str("trigger", "cron").
		Msg("Executing scheduled procedure")

	// Execute with system context (no user)
	// UserID is left empty for scheduled executions - the executor handles this
	execCtx := &ExecuteContext{
		Procedure: proc,
		Params: map[string]interface{}{
			"_trigger":      "cron",
			"_scheduled_at": time.Now().UTC().Format(time.RFC3339),
		},
		UserID:               "", // Empty for system/scheduled executions (stored as NULL)
		UserRole:             "service_role",
		IsAsync:              false,
		DisableExecutionLogs: proc.DisableExecutionLogs,
	}

	result, err := s.executor.Execute(s.ctx, execCtx)
	if err != nil {
		log.Error().Err(err).Str("procedure", proc.Name).Msg("Scheduled execution failed")
		return
	}

	log.Info().
		Str("procedure", proc.Name).
		Str("execution_id", result.ExecutionID).
		Str("status", string(result.Status)).
		Msg("Scheduled execution completed")
}

// GetScheduledProcedures returns info about all scheduled procedures
func (s *Scheduler) GetScheduledProcedures() []ScheduledProcedureInfo {
	s.jobsMu.RLock()
	defer s.jobsMu.RUnlock()

	procs := make([]ScheduledProcedureInfo, 0, len(s.procedureJobs))
	for key, entryID := range s.procedureJobs {
		entry := s.cron.Entry(entryID)
		procs = append(procs, ScheduledProcedureInfo{
			Key:     key,
			EntryID: int(entryID),
			NextRun: entry.Next,
			PrevRun: entry.Prev,
		})
	}
	return procs
}

// IsScheduled checks if a procedure is currently scheduled
func (s *Scheduler) IsScheduled(namespace, name string) bool {
	s.jobsMu.RLock()
	defer s.jobsMu.RUnlock()

	procKey := namespace + "/" + name
	_, exists := s.procedureJobs[procKey]
	return exists
}

// ScheduledProcedureInfo contains information about a scheduled procedure
type ScheduledProcedureInfo struct {
	Key     string    `json:"key"`
	EntryID int       `json:"entry_id"`
	NextRun time.Time `json:"next_run"`
	PrevRun time.Time `json:"prev_run"`
}
