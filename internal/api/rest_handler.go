package api

import (
	"fmt"
	"strings"

	"github.com/fluxbase-eu/fluxbase/internal/auth"
	"github.com/fluxbase-eu/fluxbase/internal/database"
	"github.com/fluxbase-eu/fluxbase/internal/middleware"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
)

// RESTHandler handles dynamic REST API endpoints
type RESTHandler struct {
	db          *database.Connection
	parser      *QueryParser
	schemaCache *database.SchemaCache
}

// NewRESTHandler creates a new REST handler
func NewRESTHandler(db *database.Connection, parser *QueryParser, schemaCache *database.SchemaCache) *RESTHandler {
	return &RESTHandler{
		db:          db,
		parser:      parser,
		schemaCache: schemaCache,
	}
}

// SchemaCache returns the schema cache for external access (e.g., migrations handler)
func (h *RESTHandler) SchemaCache() *database.SchemaCache {
	return h.schemaCache
}

// parseTableFromPath extracts schema and table name from URL path parameters
// Handles both /tables/:table and /tables/:schema/:table patterns
func (h *RESTHandler) parseTableFromPath(c *fiber.Ctx) (schema, table string) {
	// Get path parameters
	// For paths like /tables/posts, Fiber sees schema="posts", table=""
	// For paths like /tables/auth/users, Fiber sees schema="auth", table="users"
	schemaParam := c.Params("schema")
	tableParam := c.Params("table")

	if tableParam == "" {
		// Single segment path: /tables/posts -> public.posts
		return "public", schemaParam
	}
	// Two segment path: /tables/auth/users -> auth.users
	return schemaParam, tableParam
}

// HandleDynamicTable handles REST operations for any table via dynamic lookup
// Supports GET (list), POST (create), PATCH (batch update), DELETE (batch delete)
func (h *RESTHandler) HandleDynamicTable(c *fiber.Ctx) error {
	ctx := c.Context()
	schema, tableName := h.parseTableFromPath(c)

	// Look up table in cache
	tableInfo, exists, err := h.schemaCache.GetTable(ctx, schema, tableName)
	if err != nil {
		log.Error().Err(err).Str("schema", schema).Str("table", tableName).Msg("Failed to lookup table")
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to lookup table metadata",
		})
	}
	if !exists {
		return c.Status(404).JSON(fiber.Map{
			"error": fmt.Sprintf("Table '%s.%s' not found", schema, tableName),
		})
	}

	// Check if table is writable for write operations
	isWritable, err := h.schemaCache.IsTableWritable(ctx, schema, tableName)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to check table permissions",
		})
	}

	// Dispatch based on HTTP method
	switch c.Method() {
	case "GET":
		// Scope check is handled by middleware before this handler
		return h.makeGetHandler(*tableInfo)(c)
	case "POST":
		// Check for /query suffix (POST-based query)
		if strings.HasSuffix(c.Path(), "/query") {
			return h.makePostQueryHandler(*tableInfo)(c)
		}
		if !isWritable {
			return c.Status(405).JSON(fiber.Map{
				"error": fmt.Sprintf("Table '%s.%s' is read-only (view or materialized view)", schema, tableName),
			})
		}
		return h.makePostHandler(*tableInfo)(c)
	case "PATCH":
		if !isWritable {
			return c.Status(405).JSON(fiber.Map{
				"error": fmt.Sprintf("Table '%s.%s' is read-only (view or materialized view)", schema, tableName),
			})
		}
		return h.makeBatchPatchHandler(*tableInfo)(c)
	case "DELETE":
		if !isWritable {
			return c.Status(405).JSON(fiber.Map{
				"error": fmt.Sprintf("Table '%s.%s' is read-only (view or materialized view)", schema, tableName),
			})
		}
		return h.makeBatchDeleteHandler(*tableInfo)(c)
	default:
		return c.Status(405).JSON(fiber.Map{
			"error": fmt.Sprintf("Method %s not allowed", c.Method()),
		})
	}
}

// HandleDynamicTableById handles REST operations for a specific record
// Supports GET (fetch), PUT (replace), PATCH (update), DELETE (remove)
func (h *RESTHandler) HandleDynamicTableById(c *fiber.Ctx) error {
	ctx := c.Context()
	schema, tableName := h.parseTableFromPath(c)

	// Look up table in cache
	tableInfo, exists, err := h.schemaCache.GetTable(ctx, schema, tableName)
	if err != nil {
		log.Error().Err(err).Str("schema", schema).Str("table", tableName).Msg("Failed to lookup table")
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to lookup table metadata",
		})
	}
	if !exists {
		return c.Status(404).JSON(fiber.Map{
			"error": fmt.Sprintf("Table '%s.%s' not found", schema, tableName),
		})
	}

	// Check if table is writable for write operations
	isWritable, err := h.schemaCache.IsTableWritable(ctx, schema, tableName)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to check table permissions",
		})
	}

	// Dispatch based on HTTP method
	switch c.Method() {
	case "GET":
		return h.makeGetByIdHandler(*tableInfo)(c)
	case "PUT":
		if !isWritable {
			return c.Status(405).JSON(fiber.Map{
				"error": fmt.Sprintf("Table '%s.%s' is read-only (view or materialized view)", schema, tableName),
			})
		}
		return h.makePutHandler(*tableInfo)(c)
	case "PATCH":
		if !isWritable {
			return c.Status(405).JSON(fiber.Map{
				"error": fmt.Sprintf("Table '%s.%s' is read-only (view or materialized view)", schema, tableName),
			})
		}
		return h.makePatchHandler(*tableInfo)(c)
	case "DELETE":
		if !isWritable {
			return c.Status(405).JSON(fiber.Map{
				"error": fmt.Sprintf("Table '%s.%s' is read-only (view or materialized view)", schema, tableName),
			})
		}
		return h.makeDeleteHandler(*tableInfo)(c)
	default:
		return c.Status(405).JSON(fiber.Map{
			"error": fmt.Sprintf("Method %s not allowed", c.Method()),
		})
	}
}

// HandleDynamicQuery handles POST-based query for complex filters
func (h *RESTHandler) HandleDynamicQuery(c *fiber.Ctx) error {
	ctx := c.Context()
	schema, tableName := h.parseTableFromPath(c)

	// Look up table in cache
	tableInfo, exists, err := h.schemaCache.GetTable(ctx, schema, tableName)
	if err != nil {
		log.Error().Err(err).Str("schema", schema).Str("table", tableName).Msg("Failed to lookup table")
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to lookup table metadata",
		})
	}
	if !exists {
		return c.Status(404).JSON(fiber.Map{
			"error": fmt.Sprintf("Table '%s.%s' not found", schema, tableName),
		})
	}

	return h.makePostQueryHandler(*tableInfo)(c)
}

// RegisterTableRoutes registers REST routes for a table
func (h *RESTHandler) RegisterTableRoutes(router fiber.Router, table database.TableInfo) {
	// Build the REST path for this table
	basePath := h.BuildTablePath(table)

	log.Info().
		Str("table", fmt.Sprintf("%s.%s", table.Schema, table.Name)).
		Str("path", basePath).
		Bool("rls_enabled", table.RLSEnabled).
		Msg("Registering REST endpoints")

	// Register routes with scope enforcement
	// Read operations require read:tables scope
	router.Get(basePath, middleware.RequireScope(auth.ScopeTablesRead), h.makeGetHandler(table))
	router.Get(basePath+"/:id", middleware.RequireScope(auth.ScopeTablesRead), h.makeGetByIdHandler(table))

	// POST-based query endpoint for complex filters (avoids URL length limits)
	router.Post(basePath+"/query", middleware.RequireScope(auth.ScopeTablesRead), h.makePostQueryHandler(table))

	// Write operations require write:tables scope
	router.Post(basePath, middleware.RequireScope(auth.ScopeTablesWrite), h.makePostHandler(table))
	router.Put(basePath+"/:id", middleware.RequireScope(auth.ScopeTablesWrite), h.makePutHandler(table))
	router.Patch(basePath+"/:id", middleware.RequireScope(auth.ScopeTablesWrite), h.makePatchHandler(table))   // Single record update
	router.Patch(basePath, middleware.RequireScope(auth.ScopeTablesWrite), h.makeBatchPatchHandler(table))     // Batch update with filters
	router.Delete(basePath+"/:id", middleware.RequireScope(auth.ScopeTablesWrite), h.makeDeleteHandler(table)) // Single record delete
	router.Delete(basePath, middleware.RequireScope(auth.ScopeTablesWrite), h.makeBatchDeleteHandler(table))   // Batch delete with filters
}

// RegisterViewRoutes registers read-only REST routes for a database view
func (h *RESTHandler) RegisterViewRoutes(router fiber.Router, view database.TableInfo) {
	// Build the REST path for this view
	basePath := h.BuildTablePath(view)

	log.Info().
		Str("view", fmt.Sprintf("%s.%s", view.Schema, view.Name)).
		Str("path", basePath).
		Msg("Registering read-only view endpoints")

	// Register only GET routes for views (read-only) with scope enforcement
	router.Get(basePath, middleware.RequireScope(auth.ScopeTablesRead), h.makeGetHandler(view))
	router.Get(basePath+"/:id", middleware.RequireScope(auth.ScopeTablesRead), h.makeGetByIdHandler(view))
}

// BuildTablePath builds the REST API path for a table (relative to router group)
// Used for registering routes on the /api/v1/tables router group
func (h *RESTHandler) BuildTablePath(table database.TableInfo) string {
	// Use table name as-is without pluralization
	tableName := table.Name

	// Paths are relative to the router group
	if table.Schema != "public" {
		return "/" + table.Schema + "/" + tableName
	}
	return "/" + tableName
}

// BuildFullTablePath builds the full REST API path for a table (including /api/v1/tables prefix)
// Used for client consumption in API responses
func (h *RESTHandler) BuildFullTablePath(table database.TableInfo) string {
	relativePath := h.BuildTablePath(table)
	return "/api/v1/tables" + relativePath
}

// HandleGetTables returns metadata about available tables
func (h *RESTHandler) HandleGetTables(c *fiber.Ctx) error {
	ctx := c.Context()

	// Get all tables from cache
	tables, err := h.schemaCache.GetAllTables(ctx)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to get tables",
		})
	}

	// Get all views from cache
	views, err := h.schemaCache.GetAllViews(ctx)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to get views")
	} else {
		tables = append(tables, views...)
	}

	// Get all materialized views from cache
	matViews, err := h.schemaCache.GetAllMaterializedViews(ctx)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to get materialized views")
	} else {
		tables = append(tables, matViews...)
	}

	// Filter by schema if requested
	schemaParam := c.Query("schema")
	var schemas map[string]bool
	if schemaParam != "" {
		schemas = make(map[string]bool)
		for _, s := range strings.Split(schemaParam, ",") {
			schemas[strings.TrimSpace(s)] = true
		}
	}

	// Format response
	response := make([]fiber.Map, 0, len(tables))
	for _, table := range tables {
		// Filter by schema if specified
		if schemas != nil && !schemas[table.Schema] {
			continue
		}

		response = append(response, fiber.Map{
			"schema":      table.Schema,
			"name":        table.Name,
			"type":        table.Type,
			"path":        h.BuildFullTablePath(table),
			"columns":     table.Columns,
			"primary_key": table.PrimaryKey,
			"rls_enabled": table.RLSEnabled,
		})
	}

	return c.JSON(response)
}
