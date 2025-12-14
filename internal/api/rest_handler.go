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
	db     *database.Connection
	parser *QueryParser
}

// NewRESTHandler creates a new REST handler
func NewRESTHandler(db *database.Connection, parser *QueryParser) *RESTHandler {
	return &RESTHandler{
		db:     db,
		parser: parser,
	}
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

	// Get schema parameter
	schemas := []string{"public"}
	if schemaParam := c.Query("schema"); schemaParam != "" {
		schemas = strings.Split(schemaParam, ",")
	}

	// Get all tables
	tables, err := h.db.Inspector().GetAllTables(ctx, schemas...)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to get tables",
		})
	}

	// Format response
	response := make([]fiber.Map, 0, len(tables))
	for _, table := range tables {
		response = append(response, fiber.Map{
			"schema":      table.Schema,
			"name":        table.Name,
			"path":        h.BuildFullTablePath(table),
			"columns":     table.Columns,
			"primary_key": table.PrimaryKey,
			"rls_enabled": table.RLSEnabled,
		})
	}

	return c.JSON(response)
}
