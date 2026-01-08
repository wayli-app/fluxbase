package middleware

import (
	"net"

	"github.com/fluxbase-eu/fluxbase/internal/config"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
)

// RequireGlobalIPAllowlist restricts all server access to allowed IP ranges
// This is a global server-level restriction applied before any authentication
// If no IP ranges are configured, all IPs are allowed (backward compatible)
func RequireGlobalIPAllowlist(cfg *config.ServerConfig) fiber.Handler {
	// Parse allowed IP ranges at startup (CIDR notation)
	var allowedNets []*net.IPNet
	for _, ipRange := range cfg.AllowedIPRanges {
		_, network, err := net.ParseCIDR(ipRange)
		if err != nil {
			log.Error().Err(err).Str("range", ipRange).Msg("Invalid IP range in server global allowlist")
			continue
		}
		allowedNets = append(allowedNets, network)
	}

	return func(c *fiber.Ctx) error {
		// Skip if no IP ranges configured (allows all - backward compatible)
		if len(allowedNets) == 0 {
			return c.Next()
		}

		clientIP := getClientIP(c)

		// Check if IP is in any allowed range
		for _, network := range allowedNets {
			if network.Contains(clientIP) {
				log.Debug().
					Str("ip", clientIP.String()).
					Str("network", network.String()).
					Msg("Global IP allowlist: access allowed")
				return c.Next()
			}
		}

		// IP not in allowlist - reject request
		log.Warn().
			Str("ip", clientIP.String()).
			Str("path", c.Path()).
			Msg("Global IP allowlist: access denied - IP not in allowlist")

		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Access denied - IP not allowlisted",
		})
	}
}
