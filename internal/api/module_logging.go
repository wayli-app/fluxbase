package api

import (
	"context"

	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/logging"
	"github.com/nimbleflux/fluxbase/internal/pubsub"
	"github.com/nimbleflux/fluxbase/internal/storage"
)

type LoggingModule struct {
	Service   *logging.Service
	Handler   *LoggingHandler
	Retention *logging.RetentionService
	PubSub    pubsub.PubSub
}

func (m *LoggingModule) Name() string { return "logging" }

func (m *LoggingModule) Init(ctx context.Context, registry *ServiceRegistry) error {
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

	loggingService, err := logging.New(&cfg.Logging, db, provider, m.PubSub)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to initialize central logging service, continuing with default logging")
		return nil
	}
	m.Service = loggingService

	log.Logger = log.Output(loggingService.Writer())
	log.Info().
		Str("backend", cfg.Logging.Backend).
		Bool("pubsub_enabled", cfg.Logging.PubSubEnabled).
		Int("batch_size", cfg.Logging.BatchSize).
		Msg("Central logging service initialized")

	m.Handler = NewLoggingHandler(loggingService)

	if cfg.Logging.RetentionEnabled {
		m.Retention = logging.NewRetentionService(&cfg.Logging, loggingService.Storage())
	}

	registry.Register(loggingService)
	return nil
}
