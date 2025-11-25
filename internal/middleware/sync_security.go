package middleware

import (
	"fmt"
	"net"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
)

// RequireSyncIPAllowlist creates middleware that restricts sync endpoints to allowed IPs
// Reusable for both functions and jobs sync endpoints
// If no IP ranges are configured, all IPs are allowed
func RequireSyncIPAllowlist(allowedRanges []string, featureName string) fiber.Handler {
	// Parse allowed IP ranges at startup (CIDR notation)
	var allowedNets []*net.IPNet
	for _, ipRange := range allowedRanges {
		_, network, err := net.ParseCIDR(ipRange)
		if err != nil {
			log.Error().Err(err).Str("range", ipRange).Msgf("Invalid IP range in %s sync config", featureName)
			continue
		}
		allowedNets = append(allowedNets, network)
	}

	return func(c *fiber.Ctx) error {
		// Skip if no IP ranges configured (allows all)
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
					Msgf("%s sync access allowed - IP in allowlist", featureName)
				return c.Next()
			}
		}

		// IP not in allowlist
		log.Warn().
			Str("ip", clientIP.String()).
			Str("path", c.Path()).
			Msgf("%s sync access denied - IP not in allowlist", featureName)

		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": fmt.Sprintf("Access denied - IP not allowlisted for %s sync", featureName),
		})
	}
}
