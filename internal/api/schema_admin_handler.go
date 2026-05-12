package api

import (
	"context"
	"fmt"

	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/database"
	"github.com/nimbleflux/fluxbase/internal/middleware"
)

type SchemaAdminHandlers struct {
	db         *database.Connection
	rest       *RESTHandler
	graphCache *schemaGraphCache
}

func NewSchemaAdminHandlers(db *database.Connection, rest *RESTHandler, graphCache *schemaGraphCache) *SchemaAdminHandlers {
	return &SchemaAdminHandlers{db: db, rest: rest, graphCache: graphCache}
}

func (h *SchemaAdminHandlers) GetTables(c fiber.Ctx) error {
	ctx := context.Context(c.RequestCtx())

	if userID := middleware.GetUserID(c); userID != "" {
		if userRole, ok := GetUserRole(c); ok {
			ctx = database.ContextWithAuth(ctx, userID, userRole, userRole == "admin" || userRole == "service_role" || userRole == "tenant_service")
		}
	}

	inspector := h.db.Inspector()
	tenantPool := middleware.GetTenantPool(c)

	var schemasToQuery []string
	schemaParam := c.Query("schema")

	if schemaParam != "" {
		schemasToQuery = []string{schemaParam}
	} else {
		var schemas []string
		var err error
		if tenantPool != nil {
			schemas, err = inspector.GetSchemasFromPool(ctx, tenantPool)
		} else {
			schemas, err = inspector.GetSchemas(ctx)
		}
		if err != nil {
			return SendOperationFailed(c, "list schemas")
		}

		for _, schema := range schemas {
			if schema == "information_schema" || schema == "pg_catalog" || schema == "pg_toast" {
				continue
			}
			schemasToQuery = append(schemasToQuery, schema)
		}
	}

	var allItems []database.TableInfo
	for _, schema := range schemasToQuery {
		var tables, views, matviews []database.TableInfo
		var err error

		if tenantPool != nil {
			tables, err = inspector.GetAllTablesFromPool(ctx, tenantPool, schema)
		} else {
			tables, err = inspector.GetAllTables(ctx, schema)
		}
		if err != nil {
			log.Warn().Err(err).Str("schema", schema).Msg("Failed to get tables from schema")
		} else {
			allItems = append(allItems, tables...)
		}

		if tenantPool != nil {
			views, err = inspector.GetAllViewsFromPool(ctx, tenantPool, schema)
		} else {
			views, err = inspector.GetAllViews(ctx, schema)
		}
		if err != nil {
			log.Warn().Err(err).Str("schema", schema).Msg("Failed to get views from schema")
		} else {
			allItems = append(allItems, views...)
		}

		if tenantPool != nil {
			matviews, err = inspector.GetAllMaterializedViewsFromPool(ctx, tenantPool, schema)
		} else {
			matviews, err = inspector.GetAllMaterializedViews(ctx, schema)
		}
		if err != nil {
			log.Warn().Err(err).Str("schema", schema).Msg("Failed to get materialized views from schema")
		} else {
			allItems = append(allItems, matviews...)
		}
	}

	return c.JSON(allItems)
}

func (h *SchemaAdminHandlers) GetTableSchema(c fiber.Ctx) error {
	ctx := c.RequestCtx()
	schema := c.Params("schema")
	table := c.Params("table")

	if schema == "" || table == "" {
		return SendBadRequest(c, "Schema and table parameters are required", ErrCodeMissingField)
	}

	var tableInfo *database.TableInfo
	var err error
	if pool := middleware.GetTenantPool(c); pool != nil {
		tableInfo, err = h.db.Inspector().GetTableInfoFromPool(ctx, pool, schema, table)
	} else {
		tableInfo, err = h.db.Inspector().GetTableInfo(ctx, schema, table)
	}
	if err != nil {
		return SendNotFound(c, fmt.Sprintf("Table not found: %s.%s", schema, table))
	}

	return c.JSON(tableInfo)
}

func (h *SchemaAdminHandlers) GetSchemas(c fiber.Ctx) error {
	ctx := context.Context(c.RequestCtx())

	if userID := middleware.GetUserID(c); userID != "" {
		if userRole, ok := GetUserRole(c); ok {
			ctx = database.ContextWithAuth(ctx, userID, userRole, userRole == "admin" || userRole == "service_role" || userRole == "tenant_service")
		}
	}

	var schemas []string
	var err error
	if pool := middleware.GetTenantPool(c); pool != nil {
		schemas, err = h.db.Inspector().GetSchemasFromPool(ctx, pool)
	} else {
		schemas, err = h.db.Inspector().GetSchemas(ctx)
	}
	if err != nil {
		return SendOperationFailed(c, "list schemas")
	}

	var userSchemas []string
	for _, schema := range schemas {
		if schema != "information_schema" && schema != "pg_catalog" && schema != "pg_toast" {
			userSchemas = append(userSchemas, schema)
		}
	}

	if userRole, ok := GetUserRole(c); ok {
		isInstanceAdmin := userRole == "admin" || userRole == "instance_admin" || userRole == "service_role" || userRole == "tenant_service"
		if !isInstanceAdmin {
			tenantVisible := map[string]bool{
				"public": true, "auth": true, "storage": true, "functions": true,
				"jobs": true, "ai": true, "rpc": true, "mcp": true,
				"realtime": true, "branching": true, "logging": true, "platform": true,
			}
			var filtered []string
			for _, schema := range userSchemas {
				if tenantVisible[schema] {
					filtered = append(filtered, schema)
				}
			}
			userSchemas = filtered
		}
	}

	return c.JSON(userSchemas)
}

func (h *SchemaAdminHandlers) ExecuteQuery(c fiber.Ctx) error {
	return SendError(c, fiber.StatusNotImplemented, "Not implemented")
}

func (h *SchemaAdminHandlers) RefreshSchema(c fiber.Ctx) error {
	log.Info().Msg("Schema refresh requested")

	schemaCache := h.rest.SchemaCache()
	if schemaCache == nil {
		return SendInternalError(c, "Schema cache not initialized")
	}

	if err := schemaCache.Refresh(c.RequestCtx()); err != nil {
		log.Error().Err(err).Msg("Failed to refresh schema cache")
		return SendInternalError(c, "Failed to refresh schema cache")
	}

	h.graphCache.Invalidate()

	log.Info().
		Int("tables", schemaCache.TableCount()).
		Int("views", schemaCache.ViewCount()).
		Msg("Schema cache refreshed successfully")

	return c.JSON(fiber.Map{
		"message": "Schema cache refreshed successfully",
		"tables":  schemaCache.TableCount(),
		"views":   schemaCache.ViewCount(),
	})
}

func (h *SchemaAdminHandlers) InvalidateSchemaCache(ctx context.Context) error {
	schemaCache := h.rest.SchemaCache()
	if schemaCache == nil {
		return fmt.Errorf("schema cache not initialized")
	}

	schemaCache.InvalidateAll(ctx)
	h.graphCache.Invalidate()
	log.Debug().Msg("Schema cache invalidated and refresh triggered")

	return nil
}
