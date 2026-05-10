package api

import (
	"context"

	"github.com/gofiber/fiber/v3"

	"github.com/nimbleflux/fluxbase/internal/config"
	"github.com/nimbleflux/fluxbase/internal/middleware"
)

type contextKey string

const resolverLocalKey contextKey = "tenant_config_resolver"

func getResolverFromContext(c fiber.Ctx) *TenantConfigResolver {
	if resolver, ok := c.Locals(resolverLocalKey).(*TenantConfigResolver); ok {
		return resolver
	}
	return nil
}

func TenantConfigResolverMiddleware(resolver *TenantConfigResolver) fiber.Handler {
	return func(c fiber.Ctx) error {
		if resolver != nil {
			c.Locals(resolverLocalKey, resolver)
		}
		return c.Next()
	}
}

func GetTenantConfig(c fiber.Ctx, baseConfig *config.Config) *config.Config {
	if resolver := getResolverFromContext(c); resolver != nil {
		resolved := resolver.ResolveForRequest(context.Background(), c)
		return resolvedToFullConfig(resolved, baseConfig)
	}

	if tc, ok := c.Locals("tenant_config").(*config.Config); ok && tc != nil {
		return tc
	}
	return baseConfig
}

func resolvedToFullConfig(resolved *ResolvedConfig, baseConfig *config.Config) *config.Config {
	cfg := *baseConfig
	cfg.Auth = resolved.Auth
	cfg.Storage = resolved.Storage
	cfg.Email = resolved.Email
	cfg.Functions = resolved.Functions
	cfg.Jobs = resolved.Jobs
	cfg.AI = resolved.AI
	cfg.Realtime = resolved.Realtime
	cfg.RPC = resolved.RPC
	cfg.GraphQL = resolved.GraphQL
	cfg.API = resolved.API
	return &cfg
}

func GetTenantConfigFromLocals(c fiber.Ctx) *config.Config {
	if tc, ok := c.Locals("tenant_config").(*config.Config); ok {
		return tc
	}
	return nil
}

func GetTenantID(c fiber.Ctx) string {
	return middleware.GetTenantID(c)
}

func GetTenantSlug(c fiber.Ctx) string {
	if slug, ok := c.Locals("tenant_slug").(string); ok {
		return slug
	}
	return ""
}

func GetTenantSource(c fiber.Ctx) string {
	if source, ok := c.Locals("tenant_source").(string); ok {
		return source
	}
	return ""
}

func GetTenantRole(c fiber.Ctx) string {
	if role, ok := c.Locals("tenant_role").(string); ok {
		return role
	}
	return ""
}

func IsInstanceAdmin(c fiber.Ctx) bool {
	isAdmin, ok := c.Locals("is_instance_admin").(bool)
	return ok && isAdmin
}

func GetStorageConfig(c fiber.Ctx, baseConfig *config.Config) *config.StorageConfig {
	if resolver := getResolverFromContext(c); resolver != nil {
		resolved := resolver.ResolveForRequest(context.Background(), c)
		return &resolved.Storage
	}
	if tc, ok := c.Locals("tenant_config").(*config.Config); ok && tc != nil {
		return &tc.Storage
	}
	if baseConfig != nil {
		return &baseConfig.Storage
	}
	return nil
}

func GetAuthConfig(c fiber.Ctx, baseConfig *config.Config) *config.AuthConfig {
	if resolver := getResolverFromContext(c); resolver != nil {
		resolved := resolver.ResolveForRequest(context.Background(), c)
		return &resolved.Auth
	}
	if tc, ok := c.Locals("tenant_config").(*config.Config); ok && tc != nil {
		return &tc.Auth
	}
	return &baseConfig.Auth
}

func GetEmailConfig(c fiber.Ctx, baseConfig *config.Config) *config.EmailConfig {
	if resolver := getResolverFromContext(c); resolver != nil {
		resolved := resolver.ResolveForRequest(context.Background(), c)
		return &resolved.Email
	}
	if tc, ok := c.Locals("tenant_config").(*config.Config); ok && tc != nil {
		return &tc.Email
	}
	return &baseConfig.Email
}

func GetFunctionsConfig(c fiber.Ctx, baseConfig *config.Config) *config.FunctionsConfig {
	if resolver := getResolverFromContext(c); resolver != nil {
		resolved := resolver.ResolveForRequest(context.Background(), c)
		return &resolved.Functions
	}
	if tc, ok := c.Locals("tenant_config").(*config.Config); ok && tc != nil {
		return &tc.Functions
	}
	return &baseConfig.Functions
}

func GetJobsConfig(c fiber.Ctx, baseConfig *config.Config) *config.JobsConfig {
	if resolver := getResolverFromContext(c); resolver != nil {
		resolved := resolver.ResolveForRequest(context.Background(), c)
		return &resolved.Jobs
	}
	if tc, ok := c.Locals("tenant_config").(*config.Config); ok && tc != nil {
		return &tc.Jobs
	}
	return &baseConfig.Jobs
}

func GetAIConfig(c fiber.Ctx, baseConfig *config.Config) *config.AIConfig {
	if resolver := getResolverFromContext(c); resolver != nil {
		resolved := resolver.ResolveForRequest(context.Background(), c)
		return &resolved.AI
	}
	if tc, ok := c.Locals("tenant_config").(*config.Config); ok && tc != nil {
		return &tc.AI
	}
	return &baseConfig.AI
}

func GetRealtimeConfig(c fiber.Ctx, baseConfig *config.Config) *config.RealtimeConfig {
	if resolver := getResolverFromContext(c); resolver != nil {
		resolved := resolver.ResolveForRequest(context.Background(), c)
		return &resolved.Realtime
	}
	if tc, ok := c.Locals("tenant_config").(*config.Config); ok && tc != nil {
		return &tc.Realtime
	}
	return &baseConfig.Realtime
}

func GetRPCConfig(c fiber.Ctx, baseConfig *config.Config) *config.RPCConfig {
	if resolver := getResolverFromContext(c); resolver != nil {
		resolved := resolver.ResolveForRequest(context.Background(), c)
		return &resolved.RPC
	}
	if tc, ok := c.Locals("tenant_config").(*config.Config); ok && tc != nil {
		return &tc.RPC
	}
	return &baseConfig.RPC
}

func GetGraphQLConfig(c fiber.Ctx, baseConfig *config.Config) *config.GraphQLConfig {
	if resolver := getResolverFromContext(c); resolver != nil {
		resolved := resolver.ResolveForRequest(context.Background(), c)
		return &resolved.GraphQL
	}
	if tc, ok := c.Locals("tenant_config").(*config.Config); ok && tc != nil {
		return &tc.GraphQL
	}
	return &baseConfig.GraphQL
}

func GetAPIConfig(c fiber.Ctx, baseConfig *config.Config) *config.APIConfig {
	if resolver := getResolverFromContext(c); resolver != nil {
		resolved := resolver.ResolveForRequest(context.Background(), c)
		return &resolved.API
	}
	if tc, ok := c.Locals("tenant_config").(*config.Config); ok && tc != nil {
		return &tc.API
	}
	return &baseConfig.API
}
