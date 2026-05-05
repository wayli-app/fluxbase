package middleware

import (
	"context"
	"errors"
	"fmt"

	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/auth"
	"github.com/nimbleflux/fluxbase/internal/config"
	"github.com/nimbleflux/fluxbase/internal/database"
)

var (
	// ErrTenantNotFound is returned when tenant cannot be found
	ErrTenantNotFound = errors.New("tenant not found")
	// ErrNotTenantMember is returned when user is not a member of the tenant
	ErrNotTenantMember = errors.New("user is not a member of this tenant")
)

// TenantConfig holds configuration for tenant middleware
type TenantConfig struct {
	// DB is the database connection pool
	DB *database.Connection
	// ConfigLoader is the tenant configuration loader (optional)
	ConfigLoader *config.TenantConfigLoader
}

// TenantMiddleware extracts tenant context from request
// Precedence: X-FB-Tenant header > JWT claim > default tenant
func TenantMiddleware(cfg TenantConfig) fiber.Handler {
	return func(c fiber.Ctx) error {
		// Get user ID from context (set by auth middleware)
		userID, _ := c.Locals("user_id").(string)

		// Get claims if available
		var claims *auth.TokenClaims
		if cl, ok := c.Locals("claims").(*auth.TokenClaims); ok {
			claims = cl
		} else if cl, ok := c.Locals("jwt_claims").(*auth.TokenClaims); ok {
			claims = cl
		}

		var tenantID string
		var tenantSource string

		// 1. Check X-FB-Tenant header (explicit override)
		if headerTenant := c.Get("X-FB-Tenant"); headerTenant != "" {
			// Validate user is member of this tenant (if authenticated)
			if userID != "" {
				isMember, err := ValidateTenantMembership(c.Context(), cfg.DB, userID, headerTenant)
				if err != nil {
					log.Debug().Err(err).Str("tenant_id", headerTenant).Msg("Failed to validate tenant membership")
				} else if isMember {
					tenantID = headerTenant
					tenantSource = "header"
				}
			} else {
				// For anonymous/unauthenticated requests, validate the tenant exists
				if headerTenant != "" {
					var exists bool
					err := cfg.DB.Pool().QueryRow(
						c.Context(),
						`SELECT EXISTS(SELECT 1 FROM platform.tenants WHERE (id::text = $1 OR slug = $1) AND deleted_at IS NULL)`,
						headerTenant,
					).Scan(&exists)
					if err != nil {
						log.Debug().Err(err).Str("tenant", headerTenant).Msg("Failed to validate tenant for anonymous request")
					} else if exists {
						tenantID = headerTenant
						tenantSource = "header"
					} else {
						return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
							"error": "tenant not found",
						})
					}
				}
			}
		}

		// 2. Check JWT claim
		if tenantID == "" && claims != nil && claims.TenantID != nil {
			tenantID = *claims.TenantID
			tenantSource = "jwt"
		}

		// 3. Fall back to default tenant
		if tenantID == "" {
			defaultID, err := GetDefaultTenantID(c.Context(), cfg.DB)
			if err != nil {
				log.Debug().Err(err).Msg("Failed to get default tenant")
			} else {
				tenantID = defaultID
				tenantSource = "default"
			}
		}

		// Store tenant context
		c.Locals("tenant_id", tenantID)
		c.Locals("tenant_source", tenantSource)

		// Also store in request context for storage layer
		// This ensures database.TenantFromContext(ctx) works when handlers pass c.RequestCtx()
		if tenantID != "" {
			ctx := database.ContextWithTenant(c.RequestCtx(), tenantID)
			c.SetContext(ctx)
		}

		// Look up tenant slug, is_default flag, and store tenant-specific config
		if tenantID != "" {
			var slug string
			var isDefault bool
			err := cfg.DB.Pool().QueryRow(
				c.Context(),
				"SELECT slug, is_default FROM platform.tenants WHERE id = $1::uuid",
				tenantID,
			).Scan(&slug, &isDefault)
			if err != nil {
				log.Debug().Err(err).Str("tenant_id", tenantID).Msg("Failed to get tenant info")
			} else {
				c.Locals("tenant_slug", slug)
				c.Locals("is_default_tenant", isDefault)

				// Get tenant-specific config (only applies to default tenant via YAML/env)
				if cfg.ConfigLoader != nil {
					tenantConfig := cfg.ConfigLoader.GetConfigForSlug(slug, isDefault)
					c.Locals("tenant_config", tenantConfig)
				}
			}
		}

		// Get tenant role if we have a user and tenant
		if userID != "" && tenantID != "" && claims != nil {
			// Use tenant role from claims if available and tenant matches
			if claims.TenantID != nil && *claims.TenantID == tenantID && claims.TenantRole != "" {
				c.Locals("tenant_role", claims.TenantRole)
			} else {
				// Fetch tenant role from database
				role, err := GetUserTenantRole(c.Context(), cfg.DB, userID, tenantID)
				if err != nil {
					log.Debug().Err(err).Msg("Failed to get tenant role")
				} else if role != "" {
					c.Locals("tenant_role", role)
				}
			}
		}

		// Check if user is instance admin
		if claims != nil && claims.IsInstanceAdmin {
			c.Locals("is_instance_admin", true)
		} else if userID != "" {
			isAdmin, err := IsInstanceAdmin(c.Context(), cfg.DB, userID)
			if err != nil {
				log.Debug().Err(err).Msg("Failed to check instance admin status")
			} else if isAdmin {
				c.Locals("is_instance_admin", true)
			}
		}

		log.Debug().
			Str("tenant_id", tenantID).
			Str("tenant_source", tenantSource).
			Str("user_id", userID).
			Str("path", c.Path()).
			Msg("TenantMiddleware: Set tenant context")

		return c.Next()
	}
}

// ValidateTenantMembership checks if user is assigned to manage the specified tenant
func ValidateTenantMembership(ctx context.Context, db *database.Connection, userID, tenantID string) (bool, error) {
	var isMember bool
	err := db.Pool().QueryRow(
		ctx,
		`SELECT EXISTS(
			SELECT 1 FROM platform.tenant_admin_assignments taa
			INNER JOIN platform.tenants t ON t.id = taa.tenant_id
			WHERE taa.user_id = $1::uuid
			AND taa.tenant_id = $2::uuid
			AND t.deleted_at IS NULL
		) OR EXISTS (
			SELECT 1 FROM platform.users pu
			WHERE pu.id = $1::uuid
			AND pu.role = 'instance_admin'
			AND pu.deleted_at IS NULL
			AND pu.is_active = true
		)`,
		userID, tenantID,
	).Scan(&isMember)
	if err != nil {
		return false, fmt.Errorf("failed to check tenant membership: %w", err)
	}

	return isMember, nil
}

// GetUserTenantRole gets the user's role for the specified tenant
// Returns 'tenant_admin' if assigned to the tenant, 'instance_admin' if instance admin
func GetUserTenantRole(ctx context.Context, db *database.Connection, userID, tenantID string) (string, error) {
	if userID == auth.TenantServiceUserID {
		return "tenant_service", nil
	}

	var isAdmin bool
	err := db.Pool().QueryRow(
		ctx,
		`SELECT EXISTS(
			SELECT 1 FROM platform.users
			WHERE id = $1::uuid
			AND role = 'instance_admin'
			AND deleted_at IS NULL
			AND is_active = true
		)`,
		userID,
	).Scan(&isAdmin)
	if err != nil {
		return "", fmt.Errorf("failed to check instance admin: %w", err)
	}
	if isAdmin {
		return "instance_admin", nil
	}

	var isAssigned bool
	err = db.Pool().QueryRow(
		ctx,
		`SELECT EXISTS(
			SELECT 1 FROM platform.tenant_admin_assignments taa
			INNER JOIN platform.tenants t ON t.id = taa.tenant_id
			WHERE taa.user_id = $1::uuid
			AND taa.tenant_id = $2::uuid
			AND t.deleted_at IS NULL
		)`,
		userID, tenantID,
	).Scan(&isAssigned)
	if err != nil {
		return "", fmt.Errorf("failed to check tenant assignment: %w", err)
	}
	if isAssigned {
		return "tenant_admin", nil
	}

	return "", nil
}

// GetDefaultTenantID gets the default tenant ID
func GetDefaultTenantID(ctx context.Context, db *database.Connection) (string, error) {
	var id string
	err := db.Pool().QueryRow(
		ctx,
		`SELECT id::text FROM platform.tenants WHERE is_default = true AND deleted_at IS NULL LIMIT 1`,
	).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrTenantNotFound
		}
		return "", fmt.Errorf("failed to get default tenant: %w", err)
	}

	return id, nil
}

// IsInstanceAdmin checks if the user is an instance-level admin
func IsInstanceAdmin(ctx context.Context, db *database.Connection, userID string) (bool, error) {
	var isAdmin bool
	err := db.Pool().QueryRow(
		ctx,
		`SELECT EXISTS(
			SELECT 1 FROM platform.users
			WHERE id = $1::uuid
			AND role = 'instance_admin'
			AND deleted_at IS NULL
			AND is_active = true
		)`,
		userID,
	).Scan(&isAdmin)
	if err != nil {
		return false, fmt.Errorf("failed to check instance admin: %w", err)
	}

	return isAdmin, nil
}

// SetTenantSessionContext sets the PostgreSQL session variable for tenant context
// This should be called at the beginning of each database transaction
func SetTenantSessionContext(ctx context.Context, tx pgx.Tx, tenantID string) error {
	if tenantID == "" {
		return nil
	}

	_, err := tx.Exec(ctx, "SELECT set_config('app.current_tenant_id', $1, true)", tenantID)
	if err != nil {
		return fmt.Errorf("failed to set tenant session context: %w", err)
	}

	return nil
}

// RequireTenantRole creates a middleware that requires a specific tenant role
// If the user is an instance admin AND has a tenant context set, they are treated as a tenant admin
func RequireTenantRole(requiredRole string) fiber.Handler {
	return func(c fiber.Ctx) error {
		isInstanceAdmin, _ := c.Locals("is_instance_admin").(bool)
		tenantID, _ := c.Locals("tenant_id").(string)
		tenantSource, _ := c.Locals("tenant_source").(string)

		// Check if instance admin is acting as tenant admin (has explicit tenant context)
		actingAsTenantAdmin := tenantID != "" && (tenantSource == "header" || tenantSource == "jwt")

		// Instance admin without tenant context can bypass tenant role checks
		if isInstanceAdmin && !actingAsTenantAdmin {
			return c.Next()
		}

		// Otherwise, require proper tenant role
		tenantRole, _ := c.Locals("tenant_role").(string)
		if tenantRole == "" {
			return fiber.NewError(fiber.StatusForbidden, "tenant membership required")
		}

		if tenantRole != requiredRole && tenantRole != "tenant_admin" {
			return fiber.NewError(fiber.StatusForbidden, fmt.Sprintf("tenant %s role required", requiredRole))
		}

		return c.Next()
	}
}

// RequireInstanceAdmin creates a middleware that requires instance admin role
// This will DENY access if the user is acting as a tenant admin (has a tenant context set)
func RequireInstanceAdmin() fiber.Handler {
	return func(c fiber.Ctx) error {
		isInstanceAdmin, _ := c.Locals("is_instance_admin").(bool)
		if !isInstanceAdmin {
			return fiber.NewError(fiber.StatusForbidden, "instance admin role required")
		}

		// Check if user is acting as tenant admin (has a tenant context set)
		// If so, deny access to instance-admin-only endpoints
		tenantID, _ := c.Locals("tenant_id").(string)
		tenantSource, _ := c.Locals("tenant_source").(string)

		// If tenant context was explicitly set (via header or JWT), user is acting as tenant admin
		if tenantID != "" && (tenantSource == "header" || tenantSource == "jwt") {
			return fiber.NewError(fiber.StatusForbidden, "instance admin access not available when acting as tenant admin")
		}

		return c.Next()
	}
}

// GetUserTenantIDs gets all tenant IDs that a user can manage
// Instance admins get all tenants, others get only their assigned tenants
func GetUserTenantIDs(ctx context.Context, db *database.Connection, userID string) ([]string, error) {
	var isAdmin bool
	err := db.Pool().QueryRow(
		ctx,
		`SELECT EXISTS(
			SELECT 1 FROM platform.users
			WHERE id = $1::uuid
			AND role = 'instance_admin'
			AND deleted_at IS NULL
			AND is_active = true
		)`,
		userID,
	).Scan(&isAdmin)
	if err != nil {
		return nil, fmt.Errorf("failed to check instance admin: %w", err)
	}

	if isAdmin {
		rows, err := db.Pool().Query(
			ctx,
			`SELECT id::text FROM platform.tenants
			WHERE deleted_at IS NULL
			ORDER BY name`,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to get all tenants: %w", err)
		}
		defer rows.Close()

		var tenantIDs []string
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil {
				return nil, fmt.Errorf("failed to scan tenant ID: %w", err)
			}
			tenantIDs = append(tenantIDs, id)
		}
		return tenantIDs, nil
	}

	rows, err := db.Pool().Query(
		ctx,
		`SELECT taa.tenant_id::text FROM platform.tenant_admin_assignments taa
		INNER JOIN platform.tenants t ON t.id = taa.tenant_id
		WHERE taa.user_id = $1::uuid
		AND t.deleted_at IS NULL
		ORDER BY t.name`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get user tenants: %w", err)
	}
	defer rows.Close()

	var tenantIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to scan tenant ID: %w", err)
		}
		tenantIDs = append(tenantIDs, id)
	}

	return tenantIDs, nil
}

// GetTenantIDFromContext extracts the tenant ID from fiber context locals
// Returns empty string if not set
func GetTenantIDFromContext(c fiber.Ctx) string {
	tenantID, _ := c.Locals("tenant_id").(string)
	return tenantID
}

// GetTenantSourceFromContext extracts the tenant source from fiber context locals
// Returns empty string if not set (possible values: "header", "jwt", "default")
func GetTenantSourceFromContext(c fiber.Ctx) string {
	source, _ := c.Locals("tenant_source").(string)
	return source
}

// GetTenantRoleFromContext extracts the user's tenant role from fiber context locals
// Returns empty string if not set (possible values: "tenant_admin", "tenant_member")
func GetTenantRoleFromContext(c fiber.Ctx) string {
	role, _ := c.Locals("tenant_role").(string)
	return role
}

// IsInstanceAdminFromContext checks if the user is an instance admin from fiber context
func IsInstanceAdminFromContext(c fiber.Ctx) bool {
	isAdmin, _ := c.Locals("is_instance_admin").(bool)
	return isAdmin
}

// IsDefaultTenantFromContext checks if the current tenant is the default tenant.
func IsDefaultTenantFromContext(c fiber.Ctx) bool {
	isDefault, _ := c.Locals("is_default_tenant").(bool)
	return isDefault
}

// CtxWithTenant wraps the request context with the tenant ID from fiber locals.
// It prefers JWT claims (set by auth middleware) over TenantMiddleware's default
// fallback, since TenantMiddleware runs before auth and can't read JWT claims.
func CtxWithTenant(c fiber.Ctx) context.Context {
	tenantID := GetTenantIDFromContext(c)
	tenantSource := GetTenantSourceFromContext(c)

	if tenantSource == "default" || tenantID == "" {
		var claims *auth.TokenClaims
		if cl, ok := c.Locals("claims").(*auth.TokenClaims); ok && cl != nil {
			claims = cl
		} else if cl, ok := c.Locals("jwt_claims").(*auth.TokenClaims); ok && cl != nil {
			claims = cl
		}
		if claims != nil && claims.TenantID != nil {
			tenantID = *claims.TenantID
		}
	}
	return database.ContextWithTenant(c.RequestCtx(), tenantID)
}
