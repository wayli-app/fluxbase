package api

import (
	"github.com/nimbleflux/fluxbase/internal/api/routes"
)

func (s *Server) buildSettingsRouteDeps() *routes.SettingsDeps {
	return &routes.SettingsDeps{
		OptionalAuth:     s.optionalAuth,
		TenantMiddleware: s.Middleware.Tenant,
		GetSetting:       s.Settings.Handler.GetSetting,
		GetSettings:      s.Settings.Handler.GetSettings,
		BatchGet:         s.Settings.Handler.GetSettings,
	}
}

func (s *Server) buildUserSettingsRouteDeps() *routes.UserSettingsDeps {
	return &routes.UserSettingsDeps{
		RequireAuth:       s.requireAuth,
		TenantMiddleware:  s.Middleware.Tenant,
		ListSettings:      s.Settings.User.ListSettings,
		GetUserOwnSetting: s.Settings.User.GetUserOwnSetting,
		GetSystemSetting:  s.Settings.User.GetSystemSettingPublic,
		GetSetting:        s.Settings.User.GetSetting,
		SetSetting:        s.Settings.User.SetSetting,
		DeleteSetting:     s.Settings.User.DeleteSetting,
		CreateSecret:      s.Settings.User.CreateSecret,
		ListSecrets:       s.Settings.User.ListSecrets,
		GetSecret:         s.Settings.User.GetSecret,
		UpdateSecret:      s.Settings.User.UpdateSecret,
		DeleteSecret:      s.Settings.User.DeleteSecret,
	}
}
