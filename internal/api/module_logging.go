package api

import (
	"context"

	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/logging"
	"github.com/nimbleflux/fluxbase/internal/storage"
)

type LoggingModule struct {
	Handlers *LoggingHandlers
}

func (m *LoggingModule) Name() string { return "logging" }

func (m *LoggingModule) Init(ctx context.Context, registry *ServiceRegistry) error {
	m.Handlers = &LoggingHandlers{}
	cfg := registry.Config
	db := registry.DB

	if !(cfg.Logging.ConsoleEnabled || cfg.Logging.Backend != "") {
		return nil
	}

	storageSvc := GetService[*storage.Service](registry)
	var provider storage.Provider
	if storageSvc != nil {
		provider = storageSvc.Provider
	}

	loggingService, err := logging.New(&cfg.Logging, db, provider, registry.PubSub)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to initialize central logging service, continuing with default logging")
		return nil
	}
	m.Handlers.Service = loggingService

	log.Logger = log.Output(loggingService.Writer())
	log.Info().
		Str("backend", cfg.Logging.Backend).
		Bool("pubsub_enabled", cfg.Logging.PubSubEnabled).
		Int("batch_size", cfg.Logging.BatchSize).
		Msg("Central logging service initialized")

	m.Handlers.Handler = NewLoggingHandler(loggingService)

	if cfg.Logging.RetentionEnabled {
		m.Handlers.Retention = logging.NewRetentionService(&cfg.Logging, loggingService.Storage())
	}

	registry.Register(loggingService)
	return nil
}

func (m *LoggingModule) Shutdown(ctx context.Context) error {
	if m.Handlers.Retention != nil {
		log.Info().Msg("Stopping log retention cleanup service")
		m.Handlers.Retention.Stop()
	}
	if m.Handlers.Service != nil {
		log.Info().Msg("Closing central logging service")
		if err := m.Handlers.Service.Close(); err != nil {
			log.Warn().Err(err).Msg("Failed to close logging service")
		}
	}
	return nil
}
