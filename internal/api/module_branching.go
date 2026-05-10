package api

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/branching"
	"github.com/nimbleflux/fluxbase/internal/tenantdb"
)

type BranchingModule struct {
	Manager   *branching.Manager
	Router    *branching.Router
	Handler   *BranchHandler
	GitHub    *GitHubWebhookHandler
	Scheduler *branching.CleanupScheduler
}

func (m *BranchingModule) Name() string { return "branching" }

func (m *BranchingModule) Init(ctx context.Context, registry *ServiceRegistry) error {
	cfg := registry.Config
	db := registry.DB

	if !cfg.Branching.Enabled {
		return nil
	}

	branchStorage := branching.NewStorage(db, cfg.EncryptionKey)
	dbURL := cfg.Database.RuntimeConnectionString()
	branchManager, err := branching.NewManager(branchStorage, cfg.Branching, db.Pool(), dbURL)
	if err != nil {
		return err
	}
	branchRouter := branching.NewRouter(branchStorage, cfg.Branching, db.Pool(), dbURL)

	m.Manager = branchManager
	m.Router = branchRouter
	m.Handler = NewBranchHandler(branchManager, branchRouter, cfg.Branching)
	m.GitHub = NewGitHubWebhookHandler(branchManager, branchRouter, cfg.Branching)

	tenantManager := GetService[*tenantdb.Manager](registry)
	if tenantManager != nil {
		branchManager.SetTenantResolver(&branchTenantResolver{manager: tenantManager})
		branchManager.SetFDWRepairer(&branchFDWRepairer{manager: tenantManager})
	}

	if cfg.Branching.AutoDeleteAfter > 0 {
		cleanupInterval := cfg.Branching.AutoDeleteAfter
		if cleanupInterval < time.Hour {
			cleanupInterval = time.Hour
		}
		m.Scheduler = branching.NewCleanupScheduler(branchManager, branchRouter, cleanupInterval)
		log.Info().
			Dur("interval", cleanupInterval).
			Dur("auto_delete_after", cfg.Branching.AutoDeleteAfter).
			Msg("Branch cleanup scheduler initialized")
	}

	log.Info().
		Int("max_branches", cfg.Branching.MaxTotalBranches).
		Str("default_clone_mode", cfg.Branching.DefaultDataCloneMode).
		Msg("Database Branching enabled")

	registry.Register(branchManager)
	registry.Register(branchRouter)
	return nil
}
