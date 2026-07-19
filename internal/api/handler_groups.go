package api

import (
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/nimbleflux/fluxbase/internal/ai"
	"github.com/nimbleflux/fluxbase/internal/ai/integrations"
	"github.com/nimbleflux/fluxbase/internal/auth"
	"github.com/nimbleflux/fluxbase/internal/branching"
	"github.com/nimbleflux/fluxbase/internal/database"
	"github.com/nimbleflux/fluxbase/internal/extensions"
	"github.com/nimbleflux/fluxbase/internal/functions"
	"github.com/nimbleflux/fluxbase/internal/jobs"
	"github.com/nimbleflux/fluxbase/internal/logging"
	"github.com/nimbleflux/fluxbase/internal/mcp"
	"github.com/nimbleflux/fluxbase/internal/mcp/custom"
	"github.com/nimbleflux/fluxbase/internal/middleware"
	"github.com/nimbleflux/fluxbase/internal/migrations"
	"github.com/nimbleflux/fluxbase/internal/observability"
	"github.com/nimbleflux/fluxbase/internal/realtime"
	"github.com/nimbleflux/fluxbase/internal/rpc"
	"github.com/nimbleflux/fluxbase/internal/scaling"
	"github.com/nimbleflux/fluxbase/internal/secrets"
	"github.com/nimbleflux/fluxbase/internal/settings"
	"github.com/nimbleflux/fluxbase/internal/tenantdb"
	"github.com/nimbleflux/fluxbase/internal/webhook"
)

// AuthHandlers groups authentication-related handlers.
type AuthHandlers struct {
	Handler          *AuthHandler
	AdminHandler     *AdminAuthHandler
	DashboardHandler *DashboardAuthHandler
	ClientKeyHandler *ClientKeyHandler
	ClientKeyService *auth.ClientKeyService
	OAuthProvider    *OAuthProviderHandler
	OAuth            *OAuthHandler
	SAMLProvider     *SAMLProviderHandler
	SAML             *SAMLHandler
	SAMLService      *auth.SAMLService
	AdminSession     *AdminSessionHandler
	UserManagement   *UserManagementHandler
	Invitation       *InvitationHandler
	SettingsCache    *settings.SettingsCache
}

// StorageHandlers groups storage-related handlers.
type StorageHandlers struct {
	Handler *StorageHandler
}

// AIHandlers groups AI-related handlers and services.
type AIHandlers struct {
	Handler         *ai.Handler
	Chat            *ai.ChatHandler
	Conversations   *ai.ConversationManager
	Metrics         *observability.Metrics
	KnowledgeBase   *ai.KnowledgeBaseHandler
	KBStorage       *ai.KnowledgeBaseStorage
	KnowledgeGraph  *ai.KnowledgeGraph
	DocProcessor    *ai.DocumentProcessor
	TableExportSync *ai.TableExportSyncService
	VectorManager   *VectorManager
	VectorHandler   *VectorHandler
	Internal        *InternalAIHandler
	// IntegrationsHandler exposes CRUD + test endpoints for non-LLM tool
	// integrations (Web Search via Tavily, URL fetch). May be nil when
	// AI is disabled — wrapIntegrationHandler guards against that.
	Integrations *integrations.Handler
}

// wrapIntegrationHandler returns a fiber.Handler that delegates to the
// provided closure, or returns 503 when AI is disabled (h == nil at
// boot). The closure is constructed by the caller at registration time
// but only INVOKED at request time, so it's safe to capture h there —
// the nil-check inside the wrapper short-circuits before invocation.
//
// We use a closure rather than passing the method value directly because
// creating a method value on a nil pointer receiver is unsafe in Go.
// This pattern keeps request-time dispatch cheap (no reflect).
func wrapIntegrationHandler(h *integrations.Handler, fn func(h *integrations.Handler) fiber.Handler) fiber.Handler {
	if h == nil {
		return func(c fiber.Ctx) error {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"error": "Tool integrations are not available (AI disabled)",
			})
		}
	}
	return fn(h)
}

// FunctionsHandlers groups edge functions handlers.
type FunctionsHandlers struct {
	Handler   *functions.Handler
	Scheduler *functions.Scheduler
}

// JobsHandlers groups background jobs handlers.
type JobsHandlers struct {
	Handler   *jobs.Handler
	Manager   *jobs.Manager
	Scheduler *jobs.Scheduler
}

// RealtimeHandlers groups realtime/WebSocket handlers.
type RealtimeHandlers struct {
	Manager     *realtime.Manager
	Handler     *realtime.RealtimeHandler
	Listener    realtime.RealtimeListener
	Admin       *RealtimeAdminHandler
	HandleStats fiber.Handler
}

// MCPHandlers groups Model Context Protocol handlers.
type MCPHandlers struct {
	Handler       *mcp.Handler
	OAuth         *MCPOAuthHandler
	CustomManager *custom.Manager
	CustomHandler *CustomMCPHandler
}

// TenancyHandlers groups multi-tenancy handlers.
type TenancyHandlers struct {
	ServiceKey *ServiceKeyHandler
	Tenant     *TenantHandler
	Manager    *tenantdb.Manager
	Storage    *tenantdb.Repository
}

// BranchingHandlers groups database branching handlers.
type BranchingHandlers struct {
	Manager   *branching.Manager
	Router    *branching.Router
	Handler   *BranchHandler
	GitHub    *GitHubWebhookHandler
	Scheduler *branching.CleanupScheduler
}

// SettingsHandlers groups settings/configuration handlers.
type SettingsHandlers struct {
	System   *SystemSettingsHandler
	Custom   *CustomSettingsHandler
	User     *UserSettingsHandler
	App      *AppSettingsHandler
	Handler  *SettingsHandler
	Service  *settings.SecretsService
	Instance *InstanceSettingsHandler
	Tenant   *TenantSettingsHandler
	Unified  *settings.UnifiedService
}

// WebhookHandlers groups webhook handlers.
type WebhookHandlers struct {
	Handler *WebhookHandler
	Trigger *webhook.TriggerService
}

// LoggingHandlers groups logging handlers.
type LoggingHandlers struct {
	Service   *logging.Service
	Handler   *LoggingHandler
	Retention *logging.RetentionService
}

// SchemaHandlers groups schema/migration handlers.
type SchemaHandlers struct {
	DDL            *DDLHandler
	Migrations     *migrations.Handler
	Cache          *database.SchemaCache
	Export         *SchemaExportHandler
	InternalSchema *InternalSchemaHandler
	Graph          *SchemaGraphHandlers
	Policy         *PolicyHandlers
	Admin          *SchemaAdminHandlers
}

// RPCHandlers groups RPC handlers.
type RPCHandlers struct {
	Handler   *rpc.Handler
	Scheduler *rpc.Scheduler
}

// GraphQLHandlers groups GraphQL handlers.
type GraphQLHandlers struct {
	Handler *GraphQLHandler
}

// ExtensionsHandlers groups extension handlers.
type ExtensionsHandlers struct {
	Handler *extensions.Handler
}

// SecretsHandlers groups secret management handlers.
type SecretsHandlers struct {
	Handler *secrets.Handler
	Storage *secrets.Storage
}

// ScalingHandlers groups scaling/leader election handlers.
type ScalingHandlers struct {
	JobsLeader      *scaling.LeaderElector
	FunctionsLeader *scaling.LeaderElector
	RPCLeader       *scaling.LeaderElector
}

// MetricsComponents groups metrics-related components.
type MetricsComponents struct {
	Metrics   *observability.Metrics
	Server    *observability.MetricsServer
	StartTime time.Time
	StopChan  chan struct{}
}

// EmailHandlers groups email-related handlers.
type EmailHandlers struct {
	Template *EmailTemplateHandler
	Settings *EmailSettingsHandler
}

// CaptchaHandlers groups captcha-related handlers.
type CaptchaHandlers struct {
	Settings *CaptchaSettingsHandler
}

// MonitoringHandlers groups monitoring handlers.
type MonitoringHandlers struct {
	Handler *MonitoringHandler
}

// QuotaHandlers groups quota handlers.
type QuotaHandlers struct {
	Handler *QuotaHandler
}

// MiddlewareComponents groups middleware-related components.
type MiddlewareComponents struct {
	Tenant      fiber.Handler
	TenantDB    fiber.Handler
	Idempotency *middleware.IdempotencyMiddleware
}
