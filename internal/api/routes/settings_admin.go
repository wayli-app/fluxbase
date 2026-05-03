package routes

import (
	"github.com/gofiber/fiber/v3"
)

// SettingsAdminDeps contains dependencies for settings admin routes.
// Auth middleware is inherited from the parent admin route group.
//
// Role Access:
//   - instance_admin: Full access to all settings including instance-level
//   - tenant_admin: Access to custom settings within their tenant
type SettingsAdminDeps struct {
	// System settings - instance admin only
	ListSystemSettings  fiber.Handler
	GetSystemSetting    fiber.Handler
	UpdateSystemSetting fiber.Handler
	DeleteSystemSetting fiber.Handler

	// Custom settings - tenant accessible
	CreateCustomSetting fiber.Handler
	ListCustomSettings  fiber.Handler
	CreateSecretSetting fiber.Handler
	ListSecretSettings  fiber.Handler
	GetSecretSetting    fiber.Handler
	UpdateSecretSetting fiber.Handler
	DeleteSecretSetting fiber.Handler
	GetUserSecretValue  fiber.Handler
	GetCustomSetting    fiber.Handler
	UpdateCustomSetting fiber.Handler
	DeleteCustomSetting fiber.Handler

	// App settings
	GetAppSettings    fiber.Handler
	UpdateAppSettings fiber.Handler

	// Email settings - instance admin only
	ListEmailSettings   fiber.Handler
	GetEmailSetting     fiber.Handler
	UpdateEmailSetting  fiber.Handler
	TestEmailSettings   fiber.Handler
	ListEmailTemplates  fiber.Handler
	GetEmailTemplate    fiber.Handler
	UpdateEmailTemplate fiber.Handler
	TestEmailTemplate   fiber.Handler
	ResetEmailTemplate  fiber.Handler

	// Email settings - tenant-scoped
	GetTenantEmailSettings    fiber.Handler
	UpdateTenantEmailSettings fiber.Handler
	DeleteTenantEmailSetting  fiber.Handler
	TestTenantEmailSettings   fiber.Handler

	// Captcha settings - instance admin only
	GetCaptchaSettings    fiber.Handler
	UpdateCaptchaSettings fiber.Handler

	// Instance settings - instance admin only
	GetInstanceSettings       fiber.Handler
	UpdateInstanceSettings    fiber.Handler
	GetOverridableSettings    fiber.Handler
	UpdateOverridableSettings fiber.Handler
}

// BuildSettingsAdminRoutes creates the settings admin route group.
func BuildSettingsAdminRoutes(deps *SettingsAdminDeps) *RouteGroup {
	if deps == nil {
		return nil
	}

	return &RouteGroup{
		Name:         "settings_admin",
		DefaultAuth:  AuthRequired,
		DefaultRoles: []string{"admin", "instance_admin"},
		Routes: []Route{
			// System Settings (uses default roles)
			{Method: "GET", Path: "/system/settings", Handler: deps.ListSystemSettings, Summary: "List system settings"},
			{Method: "GET", Path: "/system/settings/*", Handler: deps.GetSystemSetting, Summary: "Get system setting"},
			{Method: "PUT", Path: "/system/settings/*", Handler: deps.UpdateSystemSetting, Summary: "Update system setting"},
			{Method: "DELETE", Path: "/system/settings/*", Handler: deps.DeleteSystemSetting, Summary: "Delete system setting"},

			// Custom Settings - tenant accessible (override roles)
			{Method: "POST", Path: "/settings/custom", Handler: deps.CreateCustomSetting, Summary: "Create custom setting", Roles: []string{"admin", "instance_admin", "tenant_admin", "service_role", "tenant_service"}},
			{Method: "GET", Path: "/settings/custom", Handler: deps.ListCustomSettings, Summary: "List custom settings", Roles: []string{"admin", "instance_admin", "tenant_admin", "service_role", "tenant_service"}},
			{Method: "POST", Path: "/settings/custom/secret", Handler: deps.CreateSecretSetting, Summary: "Create secret setting", Roles: []string{"admin", "instance_admin", "tenant_admin", "service_role", "tenant_service"}},
			{Method: "GET", Path: "/settings/custom/secrets", Handler: deps.ListSecretSettings, Summary: "List secret settings", Roles: []string{"admin", "instance_admin", "tenant_admin", "service_role", "tenant_service"}},
			{Method: "GET", Path: "/settings/custom/secret/*", Handler: deps.GetSecretSetting, Summary: "Get secret setting", Roles: []string{"admin", "instance_admin", "tenant_admin", "service_role", "tenant_service"}},
			{Method: "PUT", Path: "/settings/custom/secret/*", Handler: deps.UpdateSecretSetting, Summary: "Update secret setting", Roles: []string{"admin", "instance_admin", "tenant_admin", "service_role", "tenant_service"}},
			{Method: "DELETE", Path: "/settings/custom/secret/*", Handler: deps.DeleteSecretSetting, Summary: "Delete secret setting", Roles: []string{"admin", "instance_admin", "tenant_admin", "service_role", "tenant_service"}},
			{Method: "GET", Path: "/settings/user/:user_id/secret/:key/decrypt", Handler: deps.GetUserSecretValue, Summary: "Decrypt user secret", Roles: []string{"service_role", "tenant_service"}},
			{Method: "GET", Path: "/settings/custom/*", Handler: deps.GetCustomSetting, Summary: "Get custom setting", Roles: []string{"admin", "instance_admin", "tenant_admin", "service_role", "tenant_service"}},
			{Method: "PUT", Path: "/settings/custom/*", Handler: deps.UpdateCustomSetting, Summary: "Update custom setting", Roles: []string{"admin", "instance_admin", "tenant_admin", "service_role", "tenant_service"}},
			{Method: "DELETE", Path: "/settings/custom/*", Handler: deps.DeleteCustomSetting, Summary: "Delete custom setting", Roles: []string{"admin", "instance_admin", "tenant_admin", "service_role", "tenant_service"}},

			// App Settings - tenant accessible (override roles)
			{Method: "GET", Path: "/app/settings", Handler: deps.GetAppSettings, Summary: "Get app settings", Roles: []string{"admin", "instance_admin", "tenant_admin"}},
			{Method: "PUT", Path: "/app/settings", Handler: deps.UpdateAppSettings, Summary: "Update app settings", Roles: []string{"admin", "instance_admin", "tenant_admin"}},

			// Email Settings (uses default roles)
			{Method: "GET", Path: "/email/settings", Handler: deps.ListEmailSettings, Summary: "List email settings"},
			{Method: "GET", Path: "/email/settings/:provider", Handler: deps.GetEmailSetting, Summary: "Get email settings"},
			{Method: "PUT", Path: "/email/settings/:provider", Handler: deps.UpdateEmailSetting, Summary: "Update email settings"},
			{Method: "POST", Path: "/email/settings/:provider/test", Handler: deps.TestEmailSettings, Summary: "Test email settings"},
			{Method: "GET", Path: "/email/templates", Handler: deps.ListEmailTemplates, Summary: "List email templates"},
			{Method: "GET", Path: "/email/templates/:name", Handler: deps.GetEmailTemplate, Summary: "Get email template"},
			{Method: "PUT", Path: "/email/templates/:name", Handler: deps.UpdateEmailTemplate, Summary: "Update email template"},
			{Method: "POST", Path: "/email/templates/:name/test", Handler: deps.TestEmailTemplate, Summary: "Test email template"},
			{Method: "POST", Path: "/email/templates/:name/reset", Handler: deps.ResetEmailTemplate, Summary: "Reset email template"},

			// Tenant-scoped Email Settings (tenant_admin+)
			{Method: "GET", Path: "/email/settings/tenant", Handler: deps.GetTenantEmailSettings, Summary: "Get tenant email settings", Roles: []string{"admin", "instance_admin", "tenant_admin"}},
			{Method: "PUT", Path: "/email/settings/tenant", Handler: deps.UpdateTenantEmailSettings, Summary: "Update tenant email settings", Roles: []string{"admin", "instance_admin", "tenant_admin"}},
			{Method: "DELETE", Path: "/email/settings/tenant/:field", Handler: deps.DeleteTenantEmailSetting, Summary: "Delete tenant email setting override", Roles: []string{"admin", "instance_admin", "tenant_admin"}},
			{Method: "POST", Path: "/email/settings/tenant/test", Handler: deps.TestTenantEmailSettings, Summary: "Test tenant email settings", Roles: []string{"admin", "instance_admin", "tenant_admin"}},

			// Captcha Settings (uses default roles)
			{Method: "GET", Path: "/settings/captcha", Handler: deps.GetCaptchaSettings, Summary: "Get captcha settings"},
			{Method: "PUT", Path: "/settings/captcha", Handler: deps.UpdateCaptchaSettings, Summary: "Update captcha settings"},

			// Instance Settings (uses default roles)
			{Method: "GET", Path: "/instance/settings", Handler: deps.GetInstanceSettings, Summary: "Get instance settings"},
			{Method: "PATCH", Path: "/instance/settings", Handler: deps.UpdateInstanceSettings, Summary: "Update instance settings"},
			{Method: "GET", Path: "/instance/settings/overridable", Handler: deps.GetOverridableSettings, Summary: "Get overridable settings"},
			{Method: "PUT", Path: "/instance/settings/overridable", Handler: deps.UpdateOverridableSettings, Summary: "Update overridable settings"},
		},
	}
}
