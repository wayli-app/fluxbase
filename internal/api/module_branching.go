package api

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
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

	branchStorage := branching.NewStorage(db, cfg.EncryptionKeyBytes)
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

type branchTenantResolver struct {
	manager *tenantdb.Manager
}

func (r *branchTenantResolver) GetTenantDatabase(ctx context.Context, tenantID uuid.UUID) (*branching.TenantDatabaseInfo, error) {
	tenant, err := r.manager.GetRepository().GetTenant(ctx, tenantID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant: %w", err)
	}
	info := &branching.TenantDatabaseInfo{
		Slug:      tenant.Slug,
		IsDefault: tenant.IsDefault,
	}
	if tenant.DBName != nil {
		info.DBName = *tenant.DBName
	}
	return info, nil
}

type branchFDWRepairer struct {
	manager *tenantdb.Manager
}

func (r *branchFDWRepairer) RepairFDWForBranch(ctx context.Context, branchDBURL string, tenantID uuid.UUID) error {
	return r.manager.RepairFDWForBranch(ctx, branchDBURL, tenantID)
}

func (m *BranchingModule) Shutdown(ctx context.Context) error {
	if m.Scheduler != nil {
		m.Scheduler.Stop()
	}
	if m.Router != nil {
		log.Info().Msg("Closing branch connection pools")
		m.Router.CloseAllPools()
	}
	if m.Manager != nil {
		log.Info().Msg("Closing branch manager")
		m.Manager.Close()
	}
	return nil
}
