package api

import (
	"context"

	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/config"
	"github.com/nimbleflux/fluxbase/internal/database"
	"github.com/nimbleflux/fluxbase/internal/functions"
	"github.com/nimbleflux/fluxbase/internal/jobs"
	"github.com/nimbleflux/fluxbase/internal/realtime"
	"github.com/nimbleflux/fluxbase/internal/rpc"
	"github.com/nimbleflux/fluxbase/internal/scaling"
)

type BackgroundServicesModule struct {
	FunctionsLeader *scaling.LeaderElector
	JobsLeader      *scaling.LeaderElector
	RPCLeader       *scaling.LeaderElector
}

func (m *BackgroundServicesModule) Name() string { return "background-services" }

func (m *BackgroundServicesModule) Init(ctx context.Context, registry *ServiceRegistry) error {
	cfg := registry.Config
	db := registry.DB

	realtimeListener := GetService[*realtime.ListenerPool](registry)
	if realtimeListener != nil {
		if !cfg.Scaling.DisableRealtime && !cfg.Scaling.WorkerOnly {
			if err := realtimeListener.Start(); err != nil {
				log.Error().Err(err).Msg("Failed to start realtime listener")
			}
		} else {
			log.Info().
				Bool("disable_realtime", cfg.Scaling.DisableRealtime).
				Bool("worker_only", cfg.Scaling.WorkerOnly).
				Msg("Realtime listener disabled by scaling configuration")
		}
	}

	functionsScheduler := GetService[*functions.Scheduler](registry)
	if functionsScheduler != nil {
		m.FunctionsLeader = leaderElect(
			db, &cfg.Scaling,
			"functions-scheduler", scaling.FunctionsSchedulerLockID,
			func() {
				log.Info().Msg("This instance is now the functions scheduler leader")
				if err := functionsScheduler.Start(); err != nil {
					log.Error().Err(err).Msg("Failed to start edge functions scheduler")
				}
			},
			func() {
				log.Warn().Msg("Lost functions scheduler leadership - stopping scheduler")
				functionsScheduler.Stop()
			},
		)
	}

	jobsManager := GetService[*jobs.Manager](registry)
	if cfg.Jobs.Enabled && jobsManager != nil {
		workerCount := cfg.Jobs.EmbeddedWorkerCount
		if workerCount == 0 {
			log.Info().Msg("Jobs manager created but no workers (embedded_worker_count=0)")
		} else {
			if err := jobsManager.Start(ctx, workerCount); err != nil {
				log.Error().Err(err).Msg("Failed to start jobs manager")
			} else {
				log.Info().Int("workers", workerCount).Msg("Jobs manager started successfully")
			}
		}

		jobsScheduler := GetService[*jobs.Scheduler](registry)
		if jobsScheduler != nil {
			m.JobsLeader = leaderElect(
				db, &cfg.Scaling,
				"jobs-scheduler", scaling.JobsSchedulerLockID,
				func() {
					log.Info().Msg("This instance is now the jobs scheduler leader")
					if err := jobsScheduler.Start(); err != nil {
						log.Error().Err(err).Msg("Failed to start jobs scheduler")
					}
				},
				func() {
					log.Warn().Msg("Lost jobs scheduler leadership - stopping scheduler")
					jobsScheduler.Stop()
				},
			)
		}
	}

	rpcScheduler := GetService[*rpc.Scheduler](registry)
	if cfg.RPC.Enabled && rpcScheduler != nil {
		m.RPCLeader = leaderElect(
			db, &cfg.Scaling,
			"rpc-scheduler", scaling.RPCSchedulerLockID,
			func() {
				log.Info().Msg("This instance is now the RPC scheduler leader")
				if err := rpcScheduler.Start(); err != nil {
					log.Error().Err(err).Msg("Failed to start RPC scheduler")
				}
			},
			func() {
				log.Warn().Msg("Lost RPC scheduler leadership - stopping scheduler")
				rpcScheduler.Stop()
			},
		)
	}

	return nil
}

func leaderElect(db *database.Connection, cfg *config.ScalingConfig, name string, lockID int64, startFn, stopFn func()) *scaling.LeaderElector {
	if cfg.DisableScheduler || cfg.WorkerOnly {
		log.Info().
			Bool("disable_scheduler", cfg.DisableScheduler).
			Bool("worker_only", cfg.WorkerOnly).
			Msgf("%s disabled by scaling configuration", name)
		return nil
	}
	if cfg.EnableSchedulerLeaderElection {
		elector := scaling.NewLeaderElector(db.Pool(), lockID, name)
		elector.Start(startFn, stopFn)
		return elector
	}
	startFn()
	return nil
}

func (m *BackgroundServicesModule) Shutdown(ctx context.Context) error {
	if m.FunctionsLeader != nil {
		log.Info().Msg("Stopping functions scheduler leader election")
		m.FunctionsLeader.Stop()
	}
	if m.JobsLeader != nil {
		log.Info().Msg("Stopping jobs scheduler leader election")
		m.JobsLeader.Stop()
	}
	if m.RPCLeader != nil {
		log.Info().Msg("Stopping RPC scheduler leader election")
		m.RPCLeader.Stop()
	}
	return nil
}
