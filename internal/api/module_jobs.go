package api

import (
	"context"

	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/auth"
	"github.com/nimbleflux/fluxbase/internal/jobs"
	"github.com/nimbleflux/fluxbase/internal/logging"
	"github.com/nimbleflux/fluxbase/internal/secrets"
	"github.com/nimbleflux/fluxbase/internal/settings"
)

type JobsModule struct {
	Handlers *JobsHandlers
}

func (m *JobsModule) Name() string { return "jobs" }

func (m *JobsModule) Init(ctx context.Context, registry *ServiceRegistry) error {
	cfg := registry.Config
	db := registry.DB

	if !cfg.Jobs.Enabled {
		return nil
	}

	jobsInternalURL := cfg.BaseURL
	if jobsInternalURL == "" {
		jobsInternalURL = "http://localhost" + cfg.Server.Address
	}
	log.Info().
		Str("jobs_internal_url", jobsInternalURL).
		Bool("jwt_secret_set", cfg.Auth.JWTSecret != "").
		Msg("Initializing jobs manager with SDK credentials")

	authService := GetService[*auth.Service](registry)
	loggingService := GetService[*logging.Service](registry)
	secretsStorage := GetService[*secrets.Storage](registry)

	jobsManager := jobs.NewManager(&cfg.Jobs, db, cfg.Auth.JWTSecret, jobsInternalURL, secretsStorage, cfg)
	settingsSecrets := GetService[*settings.SecretsService](registry)
	if settingsSecrets != nil {
		jobsManager.SetSettingsSecretsService(settingsSecrets)
	}
	jobsHandler, err := jobs.NewHandler(db, &cfg.Jobs, jobsManager, authService, loggingService, cfg.Deno.NpmRegistry, cfg.Deno.JsrRegistry)
	if err != nil {
		return err
	}

	jobsScheduler := jobs.NewScheduler(db)
	jobsHandler.SetScheduler(jobsScheduler)

	m.Handlers = &JobsHandlers{
		Manager:   jobsManager,
		Handler:   jobsHandler,
		Scheduler: jobsScheduler,
	}

	registry.Register(jobsManager)
	registry.Register(jobsScheduler)
	return nil
}

func (m *JobsModule) Shutdown(ctx context.Context) error {
	if m.Handlers != nil {
		if m.Handlers.Scheduler != nil {
			m.Handlers.Scheduler.Stop()
		}
		if m.Handlers.Manager != nil {
			m.Handlers.Manager.Stop()
		}
	}
	return nil
}
