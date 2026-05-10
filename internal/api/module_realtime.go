package api

import (
	"context"

	"github.com/nimbleflux/fluxbase/internal/auth"
	"github.com/nimbleflux/fluxbase/internal/logging"
	"github.com/nimbleflux/fluxbase/internal/realtime"
	"github.com/nimbleflux/fluxbase/internal/storage"
)

type RealtimeModule struct {
	Admin    *RealtimeAdminHandler
	Manager  *realtime.Manager
	Handler  *realtime.RealtimeHandler
	Listener *realtime.ListenerPool
	Monitor  *MonitoringHandler
}

func (m *RealtimeModule) Name() string { return "realtime" }

func (m *RealtimeModule) Init(ctx context.Context, registry *ServiceRegistry) error {
	cfg := registry.Config
	db := registry.DB

	m.Admin = NewRealtimeAdminHandler(db)

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

	m.Manager = realtimeManager
	m.Handler = realtimeHandler
	m.Listener = realtimeListener

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
	m.Monitor = monitoringHandler

	registry.Register(realtimeManager)
	registry.Register(realtimeHandler)
	registry.Register(realtimeListener)
	return nil
}
