package api

import (
	"context"

	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/database"
)

type GraphQLModule struct {
	Handler *GraphQLHandler
}

func (m *GraphQLModule) Name() string { return "graphql" }

func (m *GraphQLModule) Init(ctx context.Context, registry *ServiceRegistry) error {
	cfg := registry.Config
	db := registry.DB

	if !cfg.GraphQL.Enabled {
		return nil
	}

	schemaCache := GetService[*database.SchemaCache](registry)

	m.Handler = NewGraphQLHandler(db, schemaCache, &cfg.GraphQL, cfg)

	ddlHandler := GetService[*DDLHandler](registry)
	if ddlHandler != nil && m.Handler != nil {
		ddlHandler.SetGraphQLInvalidator(m.Handler.InvalidateSchema)
	}

	log.Info().
		Int("max_depth", cfg.GraphQL.MaxDepth).
		Int("max_complexity", cfg.GraphQL.MaxComplexity).
		Bool("introspection", cfg.GraphQL.Introspection).
		Msg("GraphQL API enabled")
	if cfg.GraphQL.Introspection {
		log.Warn().Msg("GraphQL introspection is enabled — consider setting graphql.introspection to false in production")
	}

	return nil
}
