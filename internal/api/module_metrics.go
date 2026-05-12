package api

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/middleware"
	"github.com/nimbleflux/fluxbase/internal/observability"
)

type MetricsModule struct {
	Handlers *MetricsComponents
}

func (m *MetricsModule) Name() string { return "metrics" }

func (m *MetricsModule) Init(ctx context.Context, registry *ServiceRegistry) error {
	cfg := registry.Config

	if !cfg.Metrics.Enabled {
		return nil
	}

	m.Handlers.Server = observability.NewMetricsServer(cfg.Metrics.Port, cfg.Metrics.Path)
	if err := m.Handlers.Server.Start(); err != nil {
		log.Error().Err(err).Msg("Failed to start metrics server")
	}

	middleware.SetRateLimiterMetrics(m.Handlers.Metrics)

	m.Handlers.StopChan = make(chan struct{})
	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				m.Handlers.Metrics.UpdateUptime(m.Handlers.StartTime)
			case <-m.Handlers.StopChan:
				return
			}
		}
	}()

	return nil
}

func (m *MetricsModule) Shutdown(ctx context.Context) error {
	if m.Handlers != nil {
		if m.Handlers.StopChan != nil {
			close(m.Handlers.StopChan)
		}
		if m.Handlers.Server != nil {
			if err := m.Handlers.Server.Shutdown(ctx); err != nil {
				log.Warn().Err(err).Msg("Failed to shutdown metrics server")
			}
		}
	}
	return nil
}
