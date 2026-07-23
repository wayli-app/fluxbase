package routes

import (
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog/log"
)

type RouteValidator func(group *RouteGroup, route Route, fullpath string) error

type Registry struct {
	groups     []*RouteGroup
	validators []RouteValidator
	strict     bool
}

type RegistryOption func(*Registry)

func NewRegistry(opts ...RegistryOption) *Registry {
	r := &Registry{
		groups: []*RouteGroup{},
		validators: []RouteValidator{
			ValidateAuthConsistency,
			ValidateMiddlewareDependencies,
			ValidatePublicRoutes,
		},
		strict: false,
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

func WithStrictValidation() RegistryOption {
	return func(r *Registry) { r.strict = true }
}

func WithValidators(validators ...RouteValidator) RegistryOption {
	return func(r *Registry) { r.validators = append(r.validators, validators...) }
}

func (r *Registry) Register(group *RouteGroup) error {
	if group == nil {
		return fmt.Errorf("cannot register nil route group")
	}
	if err := r.validateGroup(group); err != nil {
		return fmt.Errorf("invalid route group %q: %w", group.Name, err)
	}
	r.groups = append(r.groups, group)
	return nil
}

func (r *Registry) MustRegister(groups ...*RouteGroup) {
	for _, group := range groups {
		if err := r.Register(group); err != nil {
			panic(err)
		}
	}
}

func (r *Registry) Apply(app *fiber.App) error {
	for _, group := range r.groups {
		if err := r.applyGroup(app, group, nil, nil, nil, nil, AuthNone, nil); err != nil {
			return err
		}
	}
	return nil
}

func (r *Registry) ApplyTo(router fiber.Router) error {
	for _, group := range r.groups {
		if err := r.applyGroup(router, group, nil, nil, nil, nil, AuthNone, nil); err != nil {
			return err
		}
	}
	return nil
}

func (r *Registry) applyGroup(router fiber.Router, group *RouteGroup, parentMiddlewares []Middleware, parentAuth *AuthMiddlewares, parentRequireRole func(...string) fiber.Handler, parentRequireScope func(...string) fiber.Handler, parentDefaultAuth AuthRequirement, parentDefaultRoles []string) error {
	if group == nil {
		return nil
	}

	combined := append([]Middleware{}, parentMiddlewares...)
	combined = append(combined, group.Middlewares...)

	// Inherit auth middlewares from parent if not overridden
	authMiddlewares := parentAuth
	if group.AuthMiddlewares != nil {
		authMiddlewares = group.AuthMiddlewares
	}

	// Inherit RequireRole from parent if not overridden
	requireRole := parentRequireRole
	if group.RequireRole != nil {
		requireRole = group.RequireRole
	}

	// Inherit RequireScope from parent if not overridden
	requireScope := parentRequireScope
	if group.RequireScope != nil {
		requireScope = group.RequireScope
	}

	// Inherit DefaultAuth from parent if not overridden
	defaultAuth := group.DefaultAuth
	if defaultAuth == "" {
		defaultAuth = parentDefaultAuth
	}

	// Inherit DefaultRoles from parent if not overridden
	defaultRoles := group.DefaultRoles
	if len(defaultRoles) == 0 {
		defaultRoles = parentDefaultRoles
	}

	grp := router
	if group.Prefix != "" {
		grp = router.Group(group.Prefix)
	}

	for i := range group.Routes {
		if err := r.applyRoute(grp, &group.Routes[i], combined, authMiddlewares, requireRole, requireScope, defaultAuth, defaultRoles); err != nil {
			return err
		}
	}

	for _, sub := range group.SubGroups {
		if err := r.applyGroup(grp, sub, combined, authMiddlewares, requireRole, requireScope, defaultAuth, defaultRoles); err != nil {
			return err
		}
	}

	return nil
}

func (r *Registry) applyRoute(router fiber.Router, route *Route, middlewares []Middleware, authMiddlewares *AuthMiddlewares, requireRole func(...string) fiber.Handler, requireScope func(...string) fiber.Handler, defaultAuth AuthRequirement, defaultRoles []string) error {
	if route.Handler == nil {
		return fmt.Errorf("route %s %s has nil handler", route.Method, route.Path)
	}

	var handlers []fiber.Handler

	// Auth middleware must run BEFORE group middlewares (e.g. TenantContext).
	// If TenantContext runs first, c.Locals("user_id") is empty and the
	// tenant middleware falls into its anonymous branch — returning 404
	// when X-FB-Tenant references a valid-but-different tenant, because
	// the authenticated membership check (ValidateTenantMembership) is
	// never reached. UnifiedAuth sets user_id/role so TenantContext can
	// properly validate membership and gracefully fall back.
	effectiveAuth := route.Auth
	if effectiveAuth == "" {
		effectiveAuth = defaultAuth
	}

	if authMiddlewares != nil && effectiveAuth != AuthNone && effectiveAuth != "" {
		if authHandler := authMiddlewares.MiddlewareFor(effectiveAuth); authHandler != nil {
			handlers = append(handlers, authHandler)
		}
	}

	// Then add inherited/group middlewares (TenantContext, TenantDBContext, etc.)
	for _, m := range middlewares {
		if m.Handler != nil {
			handlers = append(handlers, m.Handler)
		}
	}

	// Determine effective roles: route.Roles overrides defaultRoles (no merge)
	effectiveRoles := route.Roles
	if len(effectiveRoles) == 0 && len(defaultRoles) > 0 {
		effectiveRoles = defaultRoles
	}

	// Auto-inject role middleware based on effective Roles field
	if requireRole != nil && len(effectiveRoles) > 0 {
		handlers = append(handlers, requireRole(effectiveRoles...))
	}

	// Auto-inject scope middleware based on Scopes field
	if requireScope != nil && len(route.Scopes) > 0 {
		handlers = append(handlers, requireScope(route.Scopes...))
	}

	// Then, add route-specific middlewares (non-auth middlewares like rate limiters)
	for _, m := range route.Middlewares {
		if m.Handler != nil {
			handlers = append(handlers, m.Handler)
		}
	}

	// Finally, add the route handler
	handlers = append(handlers, route.Handler)

	if len(handlers) == 0 {
		return fmt.Errorf("route %s %s has no handlers", route.Method, route.Path)
	}

	args := make([]any, len(handlers))
	for i, h := range handlers {
		args[i] = h
	}

	method := strings.ToUpper(route.Method)
	if method == "ALL" {
		router.All(route.Path, args[0], args[1:]...)
	} else {
		router.Add([]string{method}, route.Path, args[0], args[1:]...)
	}

	return nil
}

func (r *Registry) validateGroup(group *RouteGroup) error {
	if group.Name == "" {
		return fmt.Errorf("group name is required")
	}
	return r.walkRoutes(group, "", func(g *RouteGroup, route Route, fullpath string) error {
		for _, v := range r.validators {
			if err := v(g, route, fullpath); err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *Registry) walkRoutes(group *RouteGroup, parentPath string, fn func(*RouteGroup, Route, string) error) error {
	groupPath := joinPath(parentPath, group.Prefix)

	for _, route := range group.Routes {
		fullpath := joinPath(groupPath, route.Path)
		if err := fn(group, route, fullpath); err != nil {
			return err
		}
	}

	for _, sub := range group.SubGroups {
		if err := r.walkRoutes(sub, groupPath, fn); err != nil {
			return err
		}
	}

	return nil
}

func (r *Registry) Audit() []RouteAuditEntry {
	var entries []RouteAuditEntry
	for _, group := range r.groups {
		_ = r.walkRoutes(group, "", func(g *RouteGroup, route Route, fullpath string) error {
			entries = append(entries, RouteAuditEntry{
				Method:       route.Method,
				Path:         fullpath,
				Group:        g.Name,
				Summary:      route.Summary,
				Auth:         route.Auth,
				Roles:        route.Roles,
				Scopes:       route.Scopes,
				Public:       route.Public,
				Internal:     route.Internal,
				TenantScoped: route.TenantScoped,
				Middlewares:  route.MiddlewareNames(),
			})
			return nil
		})
	}
	return entries
}

func (r *Registry) Stats() *RegistryStats {
	stats := &RegistryStats{ByMethod: make(map[string]int)}
	for _, entry := range r.Audit() {
		stats.TotalRoutes++
		stats.ByMethod[entry.Method]++
		if entry.Public {
			stats.PublicRoutes++
		}
		if entry.Internal {
			stats.InternalRoutes++
		}
		if len(entry.Roles) > 0 {
			stats.RoleProtectedRoutes++
		}
		if len(entry.Scopes) > 0 {
			stats.ScopeProtectedRoutes++
		}
	}
	return stats
}

func joinPath(parent, child string) string {
	if child == "" {
		return parent
	}
	if parent == "" {
		return child
	}
	p := strings.TrimSuffix(parent, "/")
	if !strings.HasPrefix(child, "/") {
		p += "/"
	}
	return p + child
}

type AllDeps struct {
	Health            *HealthDeps
	Realtime          *RealtimeDeps
	Storage           *StorageDeps
	REST              *RESTDeps
	GraphQL           *GraphQLDeps
	Vector            *VectorDeps
	RPC               *RPCDeps
	AI                *AIDeps
	Settings          *SettingsDeps
	UserSettings      *UserSettingsDeps
	Dashboard         *DashboardAuthDeps
	OpenAPI           *OpenAPIDeps
	Auth              *AuthDeps
	InternalAI        *InternalAIDeps
	GitHubWebhook     *GitHubWebhookDeps
	Invitation        *InvitationDeps
	Webhook           *WebhookDeps
	Monitoring        *MonitoringDeps
	Functions         *FunctionsDeps
	Jobs              *JobsDeps
	ClientKeys        *ClientKeysDeps
	Secrets           *SecretsDeps
	Sync              *SyncDeps
	Admin             *AdminDeps
	DashboardUserAuth *DashboardUserAuthDeps
	CustomMCP         *CustomMCPDeps
	MCP               *MCPDeps
	MCPOAuth          *MCPOAuthDeps
	Migrations        *MigrationsDeps
	KnowledgeBase     *KnowledgeBaseDeps
	Root              fiber.Handler
}

type HealthDeps struct {
	Handler      fiber.Handler
	OptionalAuth fiber.Handler
}

// RegisterAllRoutes registers all routes and applies them to the Fiber app.
func RegisterAllRoutes(app *fiber.App, deps *AllDeps) error {
	registry := NewRegistry(WithStrictValidation())
	registerAllGroups(registry, deps)

	if err := registry.Apply(app); err != nil {
		return err
	}

	stats := registry.Stats()
	log.Info().
		Int("total", stats.TotalRoutes).
		Int("public", stats.PublicRoutes).
		Int("internal", stats.InternalRoutes).
		Msg("Routes registered via registry")

	return nil
}

// AuditRoutes returns audit entries for all routes without applying them.
// Useful for security reviews and documentation generation.
func AuditRoutes(deps *AllDeps) []RouteAuditEntry {
	registry := NewRegistry()
	registerAllGroups(registry, deps)
	return registry.Audit()
}

// registerAllGroups registers all route groups to the given registry.
// This is the single source of truth for which route groups exist.
func registerAllGroups(registry *Registry, deps *AllDeps) {
	if deps.Health != nil {
		registry.MustRegister(BuildHealthRoutes(deps.Health.Handler, deps.Health.OptionalAuth))
	}
	if deps.Realtime != nil {
		registry.MustRegister(BuildRealtimeRoutes(deps.Realtime))
	}
	if deps.Storage != nil {
		registry.MustRegister(BuildStorageRoutes(deps.Storage))
	}
	if deps.REST != nil {
		registry.MustRegister(BuildRESTRoutes(deps.REST))
	}
	if deps.GraphQL != nil {
		registry.MustRegister(BuildGraphQLRoutes(deps.GraphQL))
	}
	if deps.Vector != nil {
		registry.MustRegister(BuildVectorRoutes(deps.Vector))
	}
	if deps.RPC != nil {
		registry.MustRegister(BuildRPCRoutes(deps.RPC))
	}
	if deps.AI != nil {
		registry.MustRegister(BuildAIRoutes(deps.AI))
	}
	if deps.Settings != nil {
		registry.MustRegister(BuildSettingsRoutes(deps.Settings))
	}
	if deps.UserSettings != nil {
		registry.MustRegister(BuildUserSettingsRoutes(deps.UserSettings))
		registry.MustRegister(BuildUserSecretsRoutes(deps.UserSettings))
	}
	if deps.Dashboard != nil {
		registry.MustRegister(BuildDashboardAuthRoutes(deps.Dashboard))
	}
	if deps.OpenAPI != nil {
		registry.MustRegister(BuildOpenAPIRoutes(deps.OpenAPI))
	}
	if deps.Auth != nil {
		registry.MustRegister(BuildAuthRoutes(deps.Auth))
	}
	if deps.InternalAI != nil {
		registry.MustRegister(BuildInternalAIRoutes(deps.InternalAI))
	}
	if deps.GitHubWebhook != nil {
		registry.MustRegister(BuildGitHubWebhookRoutes(deps.GitHubWebhook))
	}
	if deps.Invitation != nil {
		registry.MustRegister(BuildInvitationRoutes(deps.Invitation))
	}
	if deps.Webhook != nil {
		registry.MustRegister(BuildWebhookRoutes(deps.Webhook))
	}
	if deps.Monitoring != nil {
		registry.MustRegister(BuildMonitoringRoutes(deps.Monitoring))
	}
	if deps.Functions != nil {
		registry.MustRegister(BuildFunctionsRoutes(deps.Functions))
	}
	if deps.Jobs != nil {
		registry.MustRegister(BuildJobsRoutes(deps.Jobs))
	}
	if deps.ClientKeys != nil {
		registry.MustRegister(BuildClientKeysRoutes(deps.ClientKeys))
	}
	if deps.Secrets != nil {
		registry.MustRegister(BuildSecretsRoutes(deps.Secrets))
	}
	if deps.Sync != nil {
		if routes := BuildSyncRoutes(deps.Sync); routes != nil {
			registry.MustRegister(routes)
		}
	}

	// Admin routes
	if deps.Admin != nil {
		registry.MustRegister(BuildAdminRoutes(deps.Admin))
	}

	// Dashboard user auth routes
	if deps.DashboardUserAuth != nil {
		registry.MustRegister(BuildDashboardUserAuthRoutes(deps.DashboardUserAuth))
	}

	// MCP routes
	if deps.CustomMCP != nil {
		registry.MustRegister(BuildCustomMCPRoutes(deps.CustomMCP))
	}
	if deps.MCP != nil {
		registry.MustRegister(BuildMCPRoutes(deps.MCP))
	}
	if deps.MCPOAuth != nil {
		registry.MustRegister(BuildMCPOAuthRoutes(deps.MCPOAuth))
	}

	// Migrations routes
	if deps.Migrations != nil {
		registry.MustRegister(BuildMigrationsRoutes(deps.Migrations))
	}

	// Knowledge base routes
	if deps.KnowledgeBase != nil {
		if routes := BuildKnowledgeBaseRoutes(deps.KnowledgeBase); routes != nil {
			registry.MustRegister(routes)
		}
	}

	// Root route
	if deps.Root != nil {
		registry.MustRegister(&RouteGroup{
			Name:   "root",
			Prefix: "/",
			Routes: []Route{
				{Method: "GET", Path: "/", Handler: deps.Root, Summary: "Root health check", Auth: AuthNone, Public: true},
			},
		})
	}
}
