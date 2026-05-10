package api

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/auth"
	"github.com/nimbleflux/fluxbase/internal/email"
	"github.com/nimbleflux/fluxbase/internal/middleware"
	"github.com/nimbleflux/fluxbase/internal/tenantdb"
)

type TenancyModule struct {
	Manager  *tenantdb.Manager
	Storage  *tenantdb.Repository
	Tenant   *TenantHandler
	Key      *ServiceKeyHandler
	TenantDB fiber.Handler
}

func (m *TenancyModule) Name() string { return "tenancy" }

func (m *TenancyModule) Init(ctx context.Context, registry *ServiceRegistry) error {
	cfg := registry.Config
	db := registry.DB

	m.Key = NewServiceKeyHandler(db)

	if !cfg.Tenants.Enabled {
		m.Tenant = NewTenantHandler(db, nil, nil, nil, nil, cfg)
		return nil
	}

	tenantStorage := tenantdb.NewRepository(db.Pool())
	dbURL := cfg.Database.RuntimeConnectionString()
	tenantCfg := tenantdb.Config{
		Enabled:        cfg.Tenants.Enabled,
		DatabasePrefix: cfg.Tenants.DatabasePrefix,
		MaxTenants:     cfg.Tenants.MaxTenants,
		Pool: tenantdb.PoolConfig{
			MaxTotalConnections: cfg.Tenants.Pool.MaxTotalConnections,
			EvictionAge:         cfg.Tenants.Pool.EvictionAge,
		},
		Migrations: tenantdb.MigrationsConfig{
			CheckInterval: cfg.Tenants.Migrations.CheckInterval,
			OnCreate:      cfg.Tenants.Migrations.OnCreate,
			OnAccess:      cfg.Tenants.Migrations.OnAccess,
			Background:    cfg.Tenants.Migrations.Background,
		},
	}
	tenantManager := tenantdb.NewManager(tenantStorage, tenantCfg, db.Pool(), dbURL)
	tenantManager.SetAdminDBURL(cfg.Database.AdminConnectionString())

	if adminDBURL := cfg.Database.AdminConnectionString(); adminDBURL != "" {
		fdwCfg, fdwErr := tenantdb.ParseFDWConfig(adminDBURL)
		if fdwErr != nil {
			log.Warn().Err(fdwErr).Msg("Failed to parse FDW config, FDW disabled for tenant databases")
		} else {
			tenantManager.SetFDWConfig(fdwCfg)
			log.Info().Msg("FDW enabled for tenant databases")

			go func() {
				time.Sleep(5 * time.Second)
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
				defer cancel()
				tenantManager.UpgradeAllTenantsFDW(ctx)
			}()
		}
	}

	tenantRouter := tenantdb.NewRouter(tenantStorage, tenantCfg, db.Pool(), db.Pool(), dbURL)
	tenantRouter.SetManager(tenantManager)
	tenantManager.SetRouter(tenantRouter)

	log.Info().Msg("Multi-tenancy enabled")

	if cfg.Tenants.Declarative.Enabled && cfg.Tenants.Declarative.SchemaDir != "" {
		declarativeCfg := tenantdb.DeclarativeConfig{
			Enabled:          cfg.Tenants.Declarative.Enabled,
			SchemaDir:        cfg.Tenants.Declarative.SchemaDir,
			OnCreate:         cfg.Tenants.Declarative.OnCreate,
			OnStartup:        cfg.Tenants.Declarative.OnStartup,
			AllowDestructive: cfg.Tenants.Declarative.AllowDestructive,
		}
		declarativeSvc := tenantdb.NewDeclarativeService(
			declarativeCfg,
			"pgschema",
			cfg.Database.Host,
			cfg.Database.Port,
			cfg.Database.AdminUser,
			cfg.Database.AdminPassword,
			db.Pool(),
		)
		tenantManager.SetDeclarativeService(declarativeSvc)
		tenantManager.SetDeclarativeConfig(declarativeCfg)
		log.Info().
			Str("schema_dir", cfg.Tenants.Declarative.SchemaDir).
			Bool("on_create", cfg.Tenants.Declarative.OnCreate).
			Bool("on_startup", cfg.Tenants.Declarative.OnStartup).
			Msg("Tenant declarative schema service initialized")

		if cfg.Tenants.Declarative.OnStartup {
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
				defer cancel()
				if err := tenantManager.ApplyDeclarativeSchemas(ctx); err != nil {
					log.Error().Err(err).Msg("Failed to apply tenant declarative schemas on startup")
				}
			}()
		}
	}

	m.Manager = tenantManager
	m.Storage = tenantStorage

	invitationService := GetService[*auth.InvitationService](registry)
	emailService := GetService[*email.Manager](registry)
	var emailSvc email.Service
	if emailService != nil {
		emailSvc = emailService.WrapAsService()
	}
	m.Tenant = NewTenantHandler(db, tenantManager, tenantStorage, invitationService, emailSvc, cfg)

	if tenantManager.GetRouter() != nil {
		m.TenantDB = middleware.TenantDBMiddleware(middleware.TenantDBConfig{
			Router:     tenantManager.GetRouter(),
			Repository: tenantStorage,
		})
	}

	registry.Register(tenantManager)
	registry.Register(tenantStorage)
	return nil
}
