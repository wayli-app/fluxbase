package api

import (
	"context"

	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/auth"
	"github.com/nimbleflux/fluxbase/internal/logging"
	"github.com/nimbleflux/fluxbase/internal/realtime"
	"github.com/nimbleflux/fluxbase/internal/storage"
)

type RealtimeModule struct {
	Handlers   *RealtimeHandlers
	Monitoring *MonitoringHandlers
}

func (m *RealtimeModule) Name() string { return "realtime" }

func (m *RealtimeModule) Init(ctx context.Context, registry *ServiceRegistry) error {
	cfg := registry.Config
	db := registry.DB

	m.Handlers = &RealtimeHandlers{}
	m.Handlers.Admin = NewRealtimeAdminHandler(db)

	realtimeManager := realtime.NewManagerWithConfig(ctx, realtime.ManagerConfig{
		MaxConnections:         cfg.Realtime.MaxConnections,
		MaxConnectionsPerUser:  cfg.Realtime.MaxConnectionsPerUser,
		MaxConnectionsPerIP:    cfg.Realtime.MaxConnectionsPerIP,
		ClientMessageQueueSize: cfg.Realtime.ClientMessageQueueSize,
		SlowClientThreshold:    cfg.Realtime.SlowClientThreshold,
		SlowClientTimeout:      cfg.Realtime.SlowClientTimeout,
	})
	realtimeManager.SetBaseConfig(cfg)

	if registry.PubSub != nil {
		realtimeManager.SetPubSub(registry.PubSub)
	}

	authService := GetService[*auth.Service](registry)
	realtimeAuthAdapter := realtime.NewAuthServiceAdapter(authService)
	realtimeSubManager := realtime.NewSubscriptionManagerWithConfig(
		realtime.NewPgxSubscriptionDB(db.Pool()),
		realtime.RLSCacheConfig{
			MaxSize: cfg.Realtime.RLSCacheSize,
			TTL:     cfg.Realtime.RLSCacheTTL,
		},
	)
	realtimeHandler := realtime.NewRealtimeHandler(realtimeManager, realtimeAuthAdapter, realtimeSubManager)
	realtimeListener := realtime.NewListenerPool(
		db.Pool(),
		realtimeHandler,
		realtimeSubManager,
		registry.PubSub,
		realtime.ListenerPoolConfig{
			PoolSize:    cfg.Realtime.ListenerPoolSize,
			WorkerCount: cfg.Realtime.NotificationWorkers,
			QueueSize:   1000,
		},
	)

	m.Handlers.Manager = realtimeManager
	m.Handlers.Handler = realtimeHandler
	m.Handlers.Listener = realtimeListener

	storageSvc := GetService[*storage.Service](registry)
	var storageProvider storage.Provider
	if storageSvc != nil {
		storageProvider = storageSvc.Provider
	}

	monitoringHandler := NewMonitoringHandler(db, realtimeHandler, storageProvider)
	loggingSvc := GetService[*logging.Service](registry)
	if loggingSvc != nil {
		monitoringHandler.SetLoggingService(loggingSvc)
	}
	m.Monitoring = &MonitoringHandlers{Handler: monitoringHandler}

	registry.Register(realtimeManager)
	registry.Register(realtimeListener)
	return nil
}

func (m *RealtimeModule) Shutdown(ctx context.Context) error {
	if m.Handlers != nil {
		if m.Handlers.Listener != nil {
			log.Info().Msg("Stopping realtime listener")
			m.Handlers.Listener.Stop()
		}
		if m.Handlers.Manager != nil {
			log.Info().Msg("Closing WebSocket connections")
			m.Handlers.Manager.Shutdown()
		}
	}
	return nil
}
