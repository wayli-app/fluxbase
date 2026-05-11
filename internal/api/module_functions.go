package api

import (
	"context"

	"github.com/nimbleflux/fluxbase/internal/auth"
	"github.com/nimbleflux/fluxbase/internal/functions"
	"github.com/nimbleflux/fluxbase/internal/logging"
	"github.com/nimbleflux/fluxbase/internal/secrets"
	"github.com/nimbleflux/fluxbase/internal/settings"
)

type FunctionsModule struct {
	Handlers *FunctionsHandlers
}

func (m *FunctionsModule) Name() string { return "functions" }

func (m *FunctionsModule) Init(ctx context.Context, registry *ServiceRegistry) error {
	cfg := registry.Config
	db := registry.DB

	functionsInternalURL := cfg.BaseURL
	if functionsInternalURL == "" {
		functionsInternalURL = "http://localhost" + cfg.Server.Address
	}

	authService := GetService[*auth.Service](registry)
	loggingService := GetService[*logging.Service](registry)
	secretsStorage := GetService[*secrets.Storage](registry)

	functionsHandler := functions.NewHandler(db, cfg.Functions.FunctionsDir, cfg.CORS, cfg.Auth.JWTSecret, functionsInternalURL, cfg.Deno.NpmRegistry, cfg.Deno.JsrRegistry, authService, loggingService, secretsStorage, cfg)

	settingsSecrets := GetService[*settings.SecretsService](registry)
	if settingsSecrets != nil {
		functionsHandler.SetSettingsSecretsService(settingsSecrets)
	}

	functionsScheduler := functions.NewScheduler(db, cfg.Auth.JWTSecret, functionsInternalURL, secretsStorage, cfg)
	functionsHandler.SetScheduler(functionsScheduler)

	if registry.RateLimiter != nil {
		functionsHandler.SetRateLimiter(registry.RateLimiter)
	}

	m.Handlers = &FunctionsHandlers{
		Handler:   functionsHandler,
		Scheduler: functionsScheduler,
	}

	registry.Register(functionsHandler)
	registry.Register(functionsScheduler)
	return nil
}

func (m *FunctionsModule) Shutdown(ctx context.Context) error {
	if m.Handlers != nil && m.Handlers.Scheduler != nil {
		m.Handlers.Scheduler.Stop()
	}
	return nil
}
