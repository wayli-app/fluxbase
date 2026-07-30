package routes

import (
	"github.com/gofiber/fiber/v3"
)

// SchemaAdminDeps contains dependencies for schema admin routes.
// Auth middleware is inherited from the parent admin route group.
//
// Role Access:
//   - instance_admin: Full access to all schema operations (bypasses RLS)
//   - tenant_admin: Access to own tenant's schema operations (RLS enforced)
type SchemaAdminDeps struct {
	GetTables               fiber.Handler
	GetTableSchema          fiber.Handler
	GetSchemas              fiber.Handler
	ExecuteQuery            fiber.Handler
	ListSchemasDDL          fiber.Handler
	CreateSchemaDDL         fiber.Handler
	ListTablesDDL           fiber.Handler
	CreateTableDDL          fiber.Handler
	DeleteTableDDL          fiber.Handler
	RenameTableDDL          fiber.Handler
	AddColumnDDL            fiber.Handler
	DropColumnDDL           fiber.Handler
	EnableRealtime          fiber.Handler
	ListRealtimeTables      fiber.Handler
	GetRealtimeStatus       fiber.Handler
	UpdateRealtimeConfig    fiber.Handler
	DisableRealtime         fiber.Handler
	ExecuteSQL              fiber.Handler
	ExportTypeScript        fiber.Handler
	RefreshSchema           fiber.Handler
	GetSchemaGraph          fiber.Handler
	GetTableRelationships   fiber.Handler
	GetTablesWithRLS        fiber.Handler
	GetTableRLSStatus       fiber.Handler
	ToggleTableRLS          fiber.Handler
	ListPolicies            fiber.Handler
	CreatePolicy            fiber.Handler
	UpdatePolicy            fiber.Handler
	DeletePolicy            fiber.Handler
	GetPolicyTemplates      fiber.Handler
	GetSecurityWarnings     fiber.Handler
	DumpInternalSchema      fiber.Handler
	PlanInternalSchema      fiber.Handler
	ApplyInternalSchema     fiber.Handler
	ValidateInternalSchema  fiber.Handler
	GetInternalSchemaStatus fiber.Handler
	MigrateInternalSchema   fiber.Handler

	// App Schema (Declarative) — opt-in declarative management for application tables (public)
	SyncAppSchema      fiber.Handler
	ApplyAppSchema     fiber.Handler
	PlanAppSchema      fiber.Handler
	ValidateAppSchema  fiber.Handler
	GetAppSchemaStatus fiber.Handler
	DeleteAppSchema    fiber.Handler

	// Middleware for branch context
	BranchMiddleware fiber.Handler
}

// BuildSchemaAdminRoutes creates the schema admin route group.
func BuildSchemaAdminRoutes(deps *SchemaAdminDeps) *RouteGroup {
	if deps == nil {
		return nil
	}

	// Build middlewares for branch context
	var middlewares []Middleware
	if deps.BranchMiddleware != nil {
		middlewares = append(middlewares, Middleware{
			Name:    "BranchContext",
			Handler: deps.BranchMiddleware,
		})
	}

	return &RouteGroup{
		Name:         "schema_admin",
		DefaultAuth:  AuthRequired,
		DefaultRoles: []string{"admin", "instance_admin", "tenant_admin"},
		Middlewares:  middlewares,
		Routes: []Route{
			// Tables (uses default roles)
			{Method: "GET", Path: "/tables", Handler: deps.GetTables, Summary: "List all tables"},
			{Method: "GET", Path: "/tables/:schema/:table", Handler: deps.GetTableSchema, Summary: "Get table schema"},
			{Method: "GET", Path: "/schemas", Handler: deps.GetSchemas, Summary: "List schemas"},
			{Method: "POST", Path: "/query", Handler: deps.ExecuteQuery, Summary: "Execute SQL query"},

			// DDL (uses default roles)
			{Method: "GET", Path: "/ddl/schemas", Handler: deps.ListSchemasDDL, Summary: "List schemas for DDL"},
			{Method: "POST", Path: "/ddl/schemas", Handler: deps.CreateSchemaDDL, Summary: "Create schema"},
			{Method: "GET", Path: "/ddl/tables", Handler: deps.ListTablesDDL, Summary: "List tables for DDL"},
			{Method: "POST", Path: "/ddl/tables", Handler: deps.CreateTableDDL, Summary: "Create table"},
			{Method: "DELETE", Path: "/ddl/tables/:schema/:table", Handler: deps.DeleteTableDDL, Summary: "Delete table"},
			{Method: "POST", Path: "/schemas", Handler: deps.CreateSchemaDDL, Summary: "Create schema (legacy)"},
			{Method: "POST", Path: "/tables", Handler: deps.CreateTableDDL, Summary: "Create table (legacy)"},
			{Method: "DELETE", Path: "/tables/:schema/:table", Handler: deps.DeleteTableDDL, Summary: "Delete table (legacy)"},
			{Method: "PATCH", Path: "/tables/:schema/:table", Handler: deps.RenameTableDDL, Summary: "Rename table"},
			{Method: "POST", Path: "/tables/:schema/:table/columns", Handler: deps.AddColumnDDL, Summary: "Add column"},
			{Method: "DELETE", Path: "/tables/:schema/:table/columns/:column", Handler: deps.DropColumnDDL, Summary: "Drop column"},

			// Realtime (uses default roles)
			{Method: "POST", Path: "/realtime/tables", Handler: deps.EnableRealtime, Summary: "Enable realtime for table"},
			{Method: "GET", Path: "/realtime/tables", Handler: deps.ListRealtimeTables, Summary: "List realtime tables"},
			{Method: "GET", Path: "/realtime/tables/:schema/:table", Handler: deps.GetRealtimeStatus, Summary: "Get realtime status"},
			{Method: "PATCH", Path: "/realtime/tables/:schema/:table", Handler: deps.UpdateRealtimeConfig, Summary: "Update realtime config"},
			{Method: "DELETE", Path: "/realtime/tables/:schema/:table", Handler: deps.DisableRealtime, Summary: "Disable realtime for table"},

			// SQL & Schema Export (uses default roles)
			{Method: "POST", Path: "/sql/execute", Handler: deps.ExecuteSQL, Summary: "Execute SQL"},
			{Method: "GET", Path: "/schema/export/typescript", Handler: deps.ExportTypeScript, Summary: "Export TypeScript types"},
			{Method: "POST", Path: "/schema/refresh", Handler: deps.RefreshSchema, Summary: "Refresh schema cache"},

			// Schema Graph & Relationships (uses default roles)
			{Method: "GET", Path: "/schema/graph", Handler: deps.GetSchemaGraph, Summary: "Get schema graph"},
			{Method: "GET", Path: "/tables/:schema/:table/relationships", Handler: deps.GetTableRelationships, Summary: "Get table relationships"},

			// RLS (uses default roles)
			{Method: "GET", Path: "/tables/rls", Handler: deps.GetTablesWithRLS, Summary: "Get tables with RLS status"},
			{Method: "GET", Path: "/tables/:schema/:table/rls", Handler: deps.GetTableRLSStatus, Summary: "Get table RLS status"},
			// Toggle RLS is instance_admin only (override roles)
			{Method: "POST", Path: "/tables/:schema/:table/rls/toggle", Handler: deps.ToggleTableRLS, Summary: "Toggle table RLS", Roles: []string{"admin", "instance_admin"}},

			// Policies (mixed - most use default, CRUD is instance_admin only)
			{Method: "GET", Path: "/policies", Handler: deps.ListPolicies, Summary: "List RLS policies"},
			{Method: "POST", Path: "/policies", Handler: deps.CreatePolicy, Summary: "Create RLS policy", Roles: []string{"admin", "instance_admin"}},
			{Method: "GET", Path: "/policies/:schema/:table/:policy", Handler: deps.GetTableRLSStatus, Summary: "Get policies for table"},
			{Method: "PUT", Path: "/policies/:schema/:table/:policy", Handler: deps.UpdatePolicy, Summary: "Update RLS policy", Roles: []string{"admin", "instance_admin"}},
			{Method: "DELETE", Path: "/policies/:schema/:table/:policy", Handler: deps.DeletePolicy, Summary: "Delete RLS policy", Roles: []string{"admin", "instance_admin"}},
			{Method: "GET", Path: "/policies/templates", Handler: deps.GetPolicyTemplates, Summary: "Get policy templates"},
			{Method: "GET", Path: "/security/warnings", Handler: deps.GetSecurityWarnings, Summary: "Get security warnings"},

			// Internal Schema (Declarative) - instance admin only (override roles)
			{Method: "POST", Path: "/internal-schema/dump", Handler: deps.DumpInternalSchema, Summary: "Dump internal schema to SQL", Roles: []string{"admin", "instance_admin"}},
			{Method: "POST", Path: "/internal-schema/plan", Handler: deps.PlanInternalSchema, Summary: "Plan schema changes", Roles: []string{"admin", "instance_admin"}},
			{Method: "POST", Path: "/internal-schema/apply", Handler: deps.ApplyInternalSchema, Summary: "Apply schema changes", Roles: []string{"admin", "instance_admin"}},
			{Method: "GET", Path: "/internal-schema/validate", Handler: deps.ValidateInternalSchema, Summary: "Validate schema for drift", Roles: []string{"admin", "instance_admin"}},
			{Method: "GET", Path: "/internal-schema/status", Handler: deps.GetInternalSchemaStatus, Summary: "Get schema status", Roles: []string{"admin", "instance_admin"}},
			{Method: "POST", Path: "/internal-schema/migrate", Handler: deps.MigrateInternalSchema, Summary: "Migrate from imperative to declarative", Roles: []string{"admin", "instance_admin"}},

			// App Schema (Declarative) - opt-in declarative management for application tables - instance admin only
			{Method: "POST", Path: "/app-schema/sync", Handler: deps.SyncAppSchema, Summary: "Sync declarative app schema content", Roles: []string{"admin", "instance_admin"}},
			{Method: "POST", Path: "/app-schema/apply", Handler: deps.ApplyAppSchema, Summary: "Apply declarative app schema", Roles: []string{"admin", "instance_admin"}},
			{Method: "POST", Path: "/app-schema/plan", Handler: deps.PlanAppSchema, Summary: "Plan app schema changes", Roles: []string{"admin", "instance_admin"}},
			{Method: "GET", Path: "/app-schema/validate", Handler: deps.ValidateAppSchema, Summary: "Validate app schema for drift", Roles: []string{"admin", "instance_admin"}},
			{Method: "GET", Path: "/app-schema/status", Handler: deps.GetAppSchemaStatus, Summary: "Get app schema status", Roles: []string{"admin", "instance_admin"}},
			{Method: "DELETE", Path: "/app-schema", Handler: deps.DeleteAppSchema, Summary: "Remove stored app schema content", Roles: []string{"admin", "instance_admin"}},
		},
	}
}
