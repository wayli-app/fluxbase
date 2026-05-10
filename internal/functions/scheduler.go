package functions

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/config"
	"github.com/nimbleflux/fluxbase/internal/database"
	"github.com/nimbleflux/fluxbase/internal/runtime"
	"github.com/nimbleflux/fluxbase/internal/scheduler"
	"github.com/nimbleflux/fluxbase/internal/secrets"
)

type Scheduler struct {
	inner          *scheduler.CronScheduler
	storage        *Storage
	runtime        *runtime.DenoRuntime
	secretsStorage *secrets.Storage
	jwtSecret      string
	publicURL      string
	logCounters    sync.Map
}

func NewScheduler(db *database.Connection, jwtSecret, publicURL string, secretsStorage *secrets.Storage, baseConfig *config.Config) *Scheduler {
	opts := []runtime.Option{}
	if baseConfig != nil && baseConfig.Functions.MaxOutputSize > 0 {
		opts = append(opts, runtime.WithMaxOutputSize(baseConfig.Functions.MaxOutputSize))
	}
	s := &Scheduler{
		inner:          scheduler.NewCronScheduler(10),
		storage:        NewStorage(db),
		runtime:        runtime.NewRuntime(runtime.RuntimeTypeFunction, jwtSecret, publicURL, opts...),
		secretsStorage: secretsStorage,
		jwtSecret:      jwtSecret,
		publicURL:      publicURL,
	}

	s.runtime.SetLogCallback(s.handleLogMessage)

	return s
}

func (s *Scheduler) handleLogMessage(executionID uuid.UUID, level string, message string) {
	counterVal, ok := s.logCounters.Load(executionID)
	if !ok {
		log.Debug().
			Str("execution_id", executionID.String()).
			Str("level", level).
			Str("message", message).
			Msg("Scheduled function log (no counter)")
		return
	}

	ctx, ok := counterVal.(*executionLogContext)
	if !ok {
		log.Warn().Str("execution_id", executionID.String()).Msg("Invalid log counter type")
		return
	}

	lineNumber := ctx.lineCounter
	ctx.lineCounter = lineNumber + 1

	event := log.Debug().
		Str("execution_id", executionID.String()).
		Str("level", level).
		Int("line_number", lineNumber).
		Str("message", message)

	if ctx.tenantID != "" {
		event = event.Str("tenant_id", ctx.tenantID)
	}

	event.Msg("Scheduled function execution log")
}

func (s *Scheduler) Start() error {
	return s.inner.Start(func(ctx context.Context) ([]scheduler.Schedulable, error) {
		functions, err := s.storage.ListAllFunctionsAllTenants(ctx)
		if err != nil {
			return nil, err
		}

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

		return nil, nil
	}, "edge functions")
}

func (s *Scheduler) Stop() {
	s.inner.Stop("edge functions")
}

func (s *Scheduler) ScheduleFunction(fn EdgeFunctionSummary) error {
	if fn.CronSchedule == nil || *fn.CronSchedule == "" {
		return nil
	}

	funcName := fn.Name
	funcNamespace := fn.Namespace
	var funcTenantID string
	if fn.TenantID != nil {
		funcTenantID = *fn.TenantID
	}

	entryID, err := s.inner.AddFunc(fn.Name, *fn.CronSchedule, func() {
		s.executeScheduledFunction(funcName, funcNamespace, funcTenantID)
	})
	if err != nil {
		log.Error().
			Err(err).
			Str("function", fn.Name).
			Str("schedule", *fn.CronSchedule).
			Msg("Failed to add cron schedule")
		return err
	}

	log.Info().
		Str("function", fn.Name).
		Str("schedule", *fn.CronSchedule).
		Uint("entry_id", uint(entryID)).
		Msg("Function scheduled successfully")

	return nil
}

func (s *Scheduler) UnscheduleFunction(functionName string) {
	if s.inner.IsScheduled(functionName) {
		s.inner.Remove(functionName)
		log.Info().Str("function", functionName).Msg("Function unscheduled")
	}
}

func (s *Scheduler) RescheduleFunction(fn EdgeFunctionSummary) error {
	s.UnscheduleFunction(fn.Name)
	if fn.Enabled && fn.CronSchedule != nil && *fn.CronSchedule != "" {
		return s.ScheduleFunction(fn)
	}
	return nil
}

func (s *Scheduler) executeScheduledFunction(funcName, funcNamespace, tenantID string) {
	if !s.inner.Guard.Acquire(funcName) {
		return
	}
	defer s.inner.Guard.Release()

	ctx := s.inner.Context()
	if tenantID != "" {
		ctx = database.ContextWithTenant(ctx, tenantID)
	}

	fn, err := s.storage.GetFunctionByNamespace(ctx, funcName, funcNamespace)
	if err != nil {
		log.Error().
			Err(err).
			Str("function", funcName).
			Str("namespace", funcNamespace).
			Msg("Failed to fetch function for scheduled execution")
		return
	}

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

	executionID := uuid.New()
	req := runtime.ExecutionRequest{
		ID:        executionID,
		Name:      fn.Name,
		Namespace: fn.Namespace,
		Method:    "POST",
		URL:       "/scheduled",
		Headers:   make(map[string]string),
		Body:      "{}",
		TenantID:  tenantID,
	}

	if !fn.DisableExecutionLogs {
		if err := s.storage.CreateExecution(ctx, executionID, fn.ID, "cron"); err != nil {
			log.Error().Err(err).Str("execution_id", executionID.String()).Msg("Failed to create execution record")
		}
	}

	s.logCounters.Store(executionID, &executionLogContext{tenantID: tenantID})
	defer s.logCounters.Delete(executionID)

	perms := runtime.DefaultJobPermissions()
	perms.AllowNet = fn.AllowNet
	perms.AllowEnv = fn.AllowEnv
	perms.AllowRead = fn.AllowRead
	perms.AllowWrite = fn.AllowWrite

	var timeoutOverride *time.Duration
	if fn.TimeoutSeconds > 0 {
		timeout := time.Duration(fn.TimeoutSeconds) * time.Second
		timeoutOverride = &timeout
	}

	var functionSecrets map[string]string
	if s.secretsStorage != nil {
		var err error
		functionSecrets, err = s.secretsStorage.GetSecretsForNamespace(ctx, fn.Namespace)
		if err != nil {
			log.Warn().Err(err).Str("namespace", fn.Namespace).Msg("Failed to load secrets for scheduled function execution")
		}
	}

	result, err := s.runtime.Execute(ctx, fn.Code, req, perms, nil, timeoutOverride, functionSecrets)
	duration := time.Since(start)

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
		} else if result.Status >= 400 {
			status = "error"
			errMsg := fmt.Sprintf("Function returned HTTP %d", result.Status)
			errorMessage = &errMsg
		}
		log.Info().
			Str("function", fn.Name).
			Str("status", status).
			Int("status_code", result.Status).
			Dur("duration", duration).
			Msg("Scheduled function execution completed")
	}

	var resultStr *string
	if result != nil {
		if resultJSON, jsonErr := json.Marshal(result); jsonErr == nil {
			rs := string(resultJSON)
			resultStr = &rs
		}
	}

	if !fn.DisableExecutionLogs {
		go func() {
			defer func() {
				if rec := recover(); rec != nil {
					log.Error().
						Interface("panic", rec).
						Str("function", fn.Name).
						Str("execution_id", executionID.String()).
						Msg("Panic in scheduled function execution record completion - recovered")
				}
			}()
			completeCtx := context.Background()
			if tenantID != "" {
				completeCtx = database.ContextWithTenant(completeCtx, tenantID)
			}
			if updateErr := s.storage.CompleteExecution(completeCtx, executionID, status, &result.Status, &durationMs, resultStr, &result.Logs, errorMessage); updateErr != nil {
				log.Error().
					Err(updateErr).
					Str("function", fn.Name).
					Str("execution_id", executionID.String()).
					Msg("Failed to complete scheduled execution record")
			}
		}()
	}
}

func (s *Scheduler) GetScheduledFunctions() []string {
	entries := s.inner.GetScheduledEntries()
	functions := make([]string, 0, len(entries))
	for _, entry := range entries {
		functions = append(functions, entry.Key)
	}
	return functions
}

func (s *Scheduler) GetScheduleInfo(functionName string) (string, bool) {
	return s.inner.GetScheduleInfo(functionName)
}

func (s *Scheduler) IsScheduled(functionName string) bool {
	return s.inner.IsScheduled(functionName)
}
