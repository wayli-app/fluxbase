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
	DDL         *DDLHandler
	Cache       *database.SchemaCache
	Migrations  *migrations.Handler
	Export      *SchemaExportHandler
	Internal    *InternalSchemaHandler
	RESTHandler *RESTHandler
}

func (m *SchemaModule) Name() string { return "schema" }

func (m *SchemaModule) Init(ctx context.Context, registry *ServiceRegistry) error {
	cfg := registry.Config
	db := registry.DB

	ddlHandler := NewDDLHandler(db, nil)
	m.DDL = ddlHandler

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
	m.Cache = schemaCache
	ddlHandler.SetSchemaCache(schemaCache)

	migrationsHandler := migrations.NewHandler(db, schemaCache)
	tenantManager := GetService[*tenantdb.Manager](registry)
	if tenantManager != nil && tenantManager.GetRouter() != nil {
		migrationsHandler.SetTenantPoolProvider(tenantManager.GetRouter())
	}
	m.Migrations = migrationsHandler

	m.Export = NewSchemaExportHandler(schemaCache, db.Inspector())

	internalSchemaHandler := NewInternalSchemaHandler()
	internalSchemaHandler.Initialize(cfg, db)
	m.Internal = internalSchemaHandler
	log.Info().Msg("Internal schema handler initialized")

	m.RESTHandler = NewRESTHandler(db, NewQueryParser(cfg), schemaCache, cfg)

	registry.Register(schemaCache)
	registry.Register(ddlHandler)
	return nil
}
