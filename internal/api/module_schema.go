package api

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/database"
	"github.com/nimbleflux/fluxbase/internal/migrations"
	"github.com/nimbleflux/fluxbase/internal/tenantdb"
)

type SchemaModule struct {
	Handlers    *SchemaHandlers
	RESTHandler *RESTHandler
	GraphCache  *schemaGraphCache
}

func (m *SchemaModule) Name() string { return "schema" }

func (m *SchemaModule) Init(ctx context.Context, registry *ServiceRegistry) error {
	cfg := registry.Config
	db := registry.DB

	m.Handlers = &SchemaHandlers{}
	ddlHandler := NewDDLHandler(db, nil)
	m.Handlers.DDL = ddlHandler

	schemaCache := database.NewSchemaCache(db.Inspector(), 5*time.Minute)
	if registry.PubSub != nil {
		schemaCache.SetPubSub(registry.PubSub)
		log.Info().Msg("Schema cache configured for cross-instance invalidation via pub/sub")
	}
	if err := schemaCache.Refresh(ctx); err != nil {
		log.Warn().Err(err).Msg("Failed to populate schema cache on startup")
	} else {
		log.Info().Int("tables", schemaCache.TableCount()).Int("views", schemaCache.ViewCount()).Msg("Schema cache populated")
	}
	m.Handlers.Cache = schemaCache
	ddlHandler.SetSchemaCache(schemaCache)

	migrationsHandler := migrations.NewHandler(db, schemaCache)
	tenantManager := GetService[*tenantdb.Manager](registry)
	if tenantManager != nil && tenantManager.GetRouter() != nil {
		migrationsHandler.SetTenantPoolProvider(tenantManager.GetRouter())
	}
	m.Handlers.Migrations = migrationsHandler

	m.Handlers.Export = NewSchemaExportHandler(schemaCache, db.Inspector())

	internalSchemaHandler := NewInternalSchemaHandler()
	internalSchemaHandler.Initialize(cfg, db)
	m.Handlers.InternalSchema = internalSchemaHandler
	log.Info().Msg("Internal schema handler initialized")

	appSchemaHandler := NewAppSchemaHandler()
	appSchemaHandler.Initialize(cfg, db)
	m.Handlers.AppSchema = appSchemaHandler
	log.Info().Msg("App schema handler initialized")

	if m.GraphCache != nil {
		m.Handlers.Graph = NewSchemaGraphHandlers(db, m.GraphCache)
	}

	m.Handlers.Policy = NewPolicyHandlers(db)

	m.RESTHandler = NewRESTHandler(db, NewQueryParser(cfg), schemaCache, cfg)

	m.Handlers.Admin = NewSchemaAdminHandlers(db, m.RESTHandler, m.GraphCache)

	registry.Register(schemaCache)
	registry.Register(ddlHandler)
	return nil
}

func (m *SchemaModule) Shutdown(ctx context.Context) error {
	if m.Handlers != nil && m.Handlers.Cache != nil {
		m.Handlers.Cache.Close()
	}
	return nil
}
