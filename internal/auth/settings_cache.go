package auth

import (
	"time"

	"github.com/nimbleflux/fluxbase/internal/settings"
)

type SettingsCache = settings.SettingsCache

func NewSettingsCache(provider *SystemSettingsService, ttl time.Duration) *SettingsCache {
	var p settings.SettingProvider
	if provider != nil {
		p = provider.AsProvider()
	}
	return settings.NewSettingsCache(p, ttl)
}
