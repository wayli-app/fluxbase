package api

import (
	"context"

	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/auth"
	"github.com/nimbleflux/fluxbase/internal/logging"
	"github.com/nimbleflux/fluxbase/internal/observability"
	"github.com/nimbleflux/fluxbase/internal/rpc"
)

type RPCModule struct {
	Handlers *RPCHandlers
}

func (m *RPCModule) Name() string { return "rpc" }

func (m *RPCModule) Init(ctx context.Context, registry *ServiceRegistry) error {
	cfg := registry.Config
	db := registry.DB

	if !cfg.RPC.Enabled {
		return nil
	}

	authService := GetService[*auth.Service](registry)
	loggingService := GetService[*logging.Service](registry)

	rpcStorage := rpc.NewStorage(db)
	rpcLoader := rpc.NewLoader(cfg.RPC.ProceduresDir)
	rpcMetrics := observability.NewMetrics()
	rpcHandler := rpc.NewHandler(db, rpcStorage, rpcLoader, rpcMetrics, &cfg.RPC, authService, loggingService, cfg)

	rpcScheduler := rpc.NewScheduler(rpcStorage, rpcHandler.GetExecutor())
	rpcHandler.SetScheduler(rpcScheduler)

	m.Handlers = &RPCHandlers{
		Handler:   rpcHandler,
		Scheduler: rpcScheduler,
	}

	registry.Register(rpcHandler)
	registry.Register(rpcScheduler)

	log.Info().
		Str("procedures_dir", cfg.RPC.ProceduresDir).
		Bool("auto_load", cfg.RPC.AutoLoadOnBoot).
		Msg("RPC components initialized")
	return nil
}

func (m *RPCModule) Shutdown(ctx context.Context) error {
	if m.Handlers != nil {
		if m.Handlers.Scheduler != nil {
			m.Handlers.Scheduler.Stop()
		}
		if m.Handlers.Handler != nil {
			m.Handlers.Handler.GetExecutor().Stop()
		}
	}
	return nil
}
