package middleware

import (
	"net"
	"strings"
	"sync"

	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/config"
	apperrors "github.com/nimbleflux/fluxbase/internal/errors"
)

// trustedProxyNets caches parsed trusted proxy networks
var (
	trustedProxyCache   []*net.IPNet
	trustedProxyCacheMu sync.RWMutex
)

// IPExtractor is a function that extracts the client IP from a Fiber context.
// This allows for custom IP extraction strategies and easier testing.
type IPExtractor func(c fiber.Ctx) net.IP

// RequireInternal restricts access to requests originating from localhost only.
// This is used for internal service endpoints that should not be exposed externally,
// such as the AI endpoints used by MCP tools, edge functions, and jobs.
//
// The middleware checks the actual connection IP, ignoring X-Forwarded-For and
// X-Real-IP headers to prevent header spoofing attacks.
func RequireInternal() fiber.Handler {
	return RequireInternalWithExtractor(getDirectIP)
}

// RequireInternalWithExtractor is like RequireInternal but allows specifying
// a custom IP extractor function. This is primarily useful for testing.
func RequireInternalWithExtractor(extractor IPExtractor) fiber.Handler {
	return func(c fiber.Ctx) error {
		// Get the actual connection IP (ignore proxy headers for security)
		clientIP := extractor(c)

		if !isLoopback(clientIP) {
			ipStr := ""
			if clientIP != nil {
				ipStr = clientIP.String()
			}
			log.Warn().
				Str("ip", ipStr).
				Str("path", c.Path()).
				Msg("Internal endpoint access denied - not from localhost")

			return apperrors.SendErrorWithCode(c, fiber.StatusForbidden, "Access denied - internal endpoint", apperrors.ErrCodeAccessDenied)
		}

		return c.Next()
	}
}

// getDirectIP returns the direct connection IP, ignoring proxy headers.
// This is more secure for internal endpoints where we want to verify
// the request truly comes from localhost.
func getDirectIP(c fiber.Ctx) net.IP {
	// Get the raw IP from the connection, ignoring proxy headers
	// Fiber's c.Context().RemoteIP() gives us the actual connection IP
	ipStr := c.RequestCtx().RemoteIP().String()

	// Handle IPv6 zone suffix (e.g., "::1%lo0")
	if idx := strings.Index(ipStr, "%"); idx != -1 {
		ipStr = ipStr[:idx]
	}

	// Parse and return
	ip := net.ParseIP(ipStr)
	if ip == nil {
		// Fallback: try to parse from Fiber's IP method
		ip = net.ParseIP(c.IP())
	}

	return ip
}

// isLoopback checks if an IP address is a loopback address (localhost).
func isLoopback(ip net.IP) bool {
	if ip == nil {
		return false
	}

	// Check standard loopback
	if ip.IsLoopback() {
		return true
	}

	// Also check for IPv4 127.x.x.x range explicitly
	if ip4 := ip.To4(); ip4 != nil {
		return ip4[0] == 127
	}

	return false
}

// GetTrustedClientIP extracts the real client IP address safely.
// It only trusts X-Forwarded-For and X-Real-IP headers when the request
// comes from a configured trusted proxy. Otherwise, it returns the direct
// connection IP to prevent IP spoofing attacks.
//
// Security note: Always use this function instead of directly reading
// X-Forwarded-For or X-Real-IP headers, as those can be spoofed by attackers.
func GetTrustedClientIP(c fiber.Ctx, cfg *config.ServerConfig) net.IP {
	// Get the direct connection IP first
	directIP := getDirectIP(c)

	// If no trusted proxies configured, never trust proxy headers
	if len(cfg.TrustedProxies) == 0 {
		return directIP
	}

	// Check if the request comes from a trusted proxy
	if !isTrustedProxy(directIP, cfg.TrustedProxies) {
		// Request is not from a trusted proxy, don't trust headers
		return directIP
	}

	// Request is from a trusted proxy, try to get the real client IP from headers
	// Try X-Forwarded-For header first
	xff := c.Get("X-Forwarded-For")
	if xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			// The leftmost IP is the original client IP
			// Subsequent IPs are proxies the request passed through
			// We want the original client IP for access control
			ip := strings.TrimSpace(ips[0])
			parsed := net.ParseIP(ip)
			if parsed != nil {
				return parsed
			}
		}
	}

	// Try X-Real-IP header
	xri := c.Get("X-Real-IP")
	if xri != "" {
		parsed := net.ParseIP(xri)
		if parsed != nil {
			return parsed
		}
	}

	// Fall back to direct IP
	return directIP
}

// isTrustedProxy checks if an IP address is in the trusted proxy list
func isTrustedProxy(ip net.IP, trustedProxies []string) bool {
	if ip == nil {
		return false
	}

	// Parse trusted proxy ranges (with caching)
	trustedProxyCacheMu.RLock()
	nets := trustedProxyCache
	trustedProxyCacheMu.RUnlock()

	if nets == nil {
		// Parse and cache the networks
		var parsedNets []*net.IPNet
		for _, proxyRange := range trustedProxies {
			// Handle single IPs by converting to CIDR
			if !strings.Contains(proxyRange, "/") {
				// It's a single IP, convert to /32 or /128
				parsedIP := net.ParseIP(proxyRange)
				if parsedIP != nil {
					if parsedIP.To4() != nil {
						proxyRange += "/32"
					} else {
						proxyRange += "/128"
					}
				}
			}

			_, network, err := net.ParseCIDR(proxyRange)
			if err != nil {
				log.Error().Err(err).Str("range", proxyRange).Msg("Invalid trusted proxy range")
				continue
			}
			parsedNets = append(parsedNets, network)
		}

		trustedProxyCacheMu.Lock()
		trustedProxyCache = parsedNets
		nets = parsedNets
		trustedProxyCacheMu.Unlock()
	}

	// Check if IP is in any trusted proxy range
	for _, network := range nets {
		if network.Contains(ip) {
			return true
		}
	}

	return false
}

// fiber:context-methods migrated
