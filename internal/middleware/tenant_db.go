package middleware

import (
	"context"
	"errors"
	"fmt"

	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/auth"
	"github.com/nimbleflux/fluxbase/internal/database"
	"github.com/nimbleflux/fluxbase/internal/tenantdb"
)

var ErrTenantDBUnavailable = errors.New("tenant database unavailable")

// TenantRepository provides the storage methods the tenant DB middleware needs.
// Defined at the consumer side per Go convention. *tenantdb.Storage satisfies this.
type TenantRepository interface {
	GetTenant(ctx context.Context, id string) (*tenantdb.Tenant, error)
	GetTenantBySlug(ctx context.Context, slug string) (*tenantdb.Tenant, error)
	GetDefaultTenant(ctx context.Context) (*tenantdb.Tenant, error)
	IsUserAssignedToTenant(ctx context.Context, userID, tenantID string) (bool, error)
}

// UserTenantLister provides tenant membership lookups.
// *tenantdb.Storage satisfies this.
type UserTenantLister interface {
	GetTenantsForUser(ctx context.Context, userID string) ([]tenantdb.Tenant, error)
}

// Compile-time interface satisfaction checks
var (
	_ TenantRepository = (*tenantdb.Repository)(nil)
	_ UserTenantLister = (*tenantdb.Repository)(nil)
)

type TenantDBConfig struct {
	Router     *tenantdb.Router
	Repository TenantRepository
}

func TenantDBMiddleware(cfg TenantDBConfig) fiber.Handler {
	return func(c fiber.Ctx) error {
		userID, _ := c.Locals("user_id").(string)
		isInstanceAdmin, _ := c.Locals("is_instance_admin").(bool)
		userRole, _ := c.Locals("user_role").(string)
		tenantRole, _ := c.Locals("tenant_role").(string)

		var claims *auth.TokenClaims
		if cl, ok := c.Locals("claims").(*auth.TokenClaims); ok {
			claims = cl
		} else if cl, ok := c.Locals("jwt_claims").(*auth.TokenClaims); ok {
			claims = cl
		}

		// Respect prior explicit resolution from TenantMiddleware.
		// Only re-resolve if no explicit source ("header" or "jwt") was set.
		existingSource, _ := c.Locals("tenant_source").(string)
		existingTenantID, _ := c.Locals("tenant_id").(string)

		var tenantID, tenantSource string
		if existingTenantID != "" && (existingSource == "header" || existingSource == "jwt") {
			tenantID = existingTenantID
			tenantSource = existingSource
		} else {
			tenantID, tenantSource = resolveTenantID(c, userID, isInstanceAdmin, claims, cfg.Repository)
		}

		if tenantID != "" && !isInstanceAdmin && userRole != "tenant_service" && tenantRole != "tenant_service" && userID != "" {
			hasAccess, err := cfg.Repository.IsUserAssignedToTenant(c.Context(), userID, tenantID)
			if err != nil {
				log.Debug().Err(err).Msg("Failed to check tenant access")
				return fiber.NewError(fiber.StatusInternalServerError, "failed to verify tenant access")
			}
			if !hasAccess {
				return fiber.NewError(fiber.StatusForbidden, "access denied to this tenant")
			}
		}

		c.Locals("tenant_id", tenantID)
		c.Locals("tenant_source", tenantSource)
		c.SetContext(database.ContextWithTenant(c.RequestCtx(), tenantID))

		// Fetch tenant record and set DB context only for tenants with separate databases.
		// For default tenants (UsesMainDatabase), tenant_db stays nil so handlers fall back
		// to their default main pool — preserving backward compatibility.
		if tenantID != "" && cfg.Repository != nil {
			if tenant, err := cfg.Repository.GetTenant(c.Context(), tenantID); err == nil {
				c.Locals("tenant_slug", tenant.Slug)
				if !tenant.UsesMainDatabase() {
					c.Locals("tenant_db_name", *tenant.DBName)
					pool, err := cfg.Router.GetPool(tenantID)
					if err != nil {
						log.Error().Err(err).Str("tenant_id", tenantID).Msg("Failed to get tenant pool")
						return fiber.NewError(fiber.StatusInternalServerError, "tenant database unavailable")
					}
					c.Locals("tenant_db", pool)
				}
			}
		}

		if userID != "" && tenantID != "" && cfg.Repository != nil {
			if claims != nil && claims.TenantID != nil && *claims.TenantID == tenantID && claims.TenantRole != "" {
				c.Locals("tenant_role", claims.TenantRole)
			}
		}

		log.Debug().
			Str("tenant_id", tenantID).
			Str("tenant_source", tenantSource).
			Str("user_id", userID).
			Str("path", c.Path()).
			Bool("uses_main_db", c.Locals("tenant_db") == nil).
			Msg("TenantDBMiddleware: Set tenant context")

		return c.Next()
	}
}

func resolveTenantID(c fiber.Ctx, userID string, isInstanceAdmin bool, claims *auth.TokenClaims, storage TenantRepository) (string, string) {
	if headerTenant := c.Get("X-FB-Tenant"); headerTenant != "" {
		if storage != nil {
			if tenant, err := storage.GetTenantBySlug(c.Context(), headerTenant); err == nil {
				return tenant.ID, "header"
			}
			if _, err := storage.GetTenant(c.Context(), headerTenant); err == nil {
				return headerTenant, "header"
			}
			log.Debug().Str("tenant", headerTenant).Msg("resolveTenantID: X-FB-Tenant header value does not match any known tenant")
		}
		return "", ""
	}

	if claims != nil && claims.TenantID != nil && *claims.TenantID != "" {
		return *claims.TenantID, "jwt"
	}

	if storage != nil {
		if tenant, err := storage.GetDefaultTenant(c.Context()); err == nil {
			return tenant.ID, "default"
		}
	}

	return "", ""
}

func GetTenantPool(c fiber.Ctx) *pgxpool.Pool {
	pool, _ := c.Locals("tenant_db").(*pgxpool.Pool)
	return pool
}

// GetPoolForSchema returns the appropriate database pool based on the target schema.
// Priority: branch pool > tenant pool > main pool.
// This enables tenant-aware routing for REST, GraphQL, and other handlers.
// Tenant pools use FDW for cross-database joins (e.g., auth.users), so all schemas
// can be served from the tenant pool.
func GetPoolForSchema(c fiber.Ctx, schema string, mainPool *pgxpool.Pool) *pgxpool.Pool {
	// 1. Branch pool takes highest priority (for database branching feature)
	if pool := GetBranchPool(c); pool != nil {
		return pool
	}

	// 2. Tenant pool routes all queries (FDW handles cross-DB joins)
	if tenantPool := GetTenantPool(c); tenantPool != nil {
		return tenantPool
	}

	// 3. Fall back to main pool when no tenant pool exists
	return mainPool
}

// SetTargetSchema stores the target schema in fiber locals for pool routing.
// Handlers should call this before WrapWithRLS to enable schema-aware pool selection.
func SetTargetSchema(c fiber.Ctx, schema string) {
	c.Locals("target_schema", schema)
}

// GetTargetSchema retrieves the target schema from fiber locals.
// Returns empty string as default if not set. Handlers must explicitly call
// SetTargetSchema to enable tenant-aware pool routing. This prevents internal
// schema handlers (auth.*, app.*, etc.) from inadvertently routing to tenant pools.
func GetTargetSchema(c fiber.Ctx) string {
	if schema, ok := c.Locals("target_schema").(string); ok && schema != "" {
		return schema
	}
	return ""
}

func GetUserManagedTenantIDs(ctx context.Context, storage UserTenantLister, userID string) ([]string, error) {
	if storage == nil {
		return nil, errors.New("storage not initialized")
	}

	tenants, err := storage.GetTenantsForUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user tenants: %w", err)
	}

	ids := make([]string, len(tenants))
	for i, t := range tenants {
		ids[i] = t.ID
	}
	return ids, nil
}
