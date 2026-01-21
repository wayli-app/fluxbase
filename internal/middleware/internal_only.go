package middleware

import (
	"net"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
)

// RequireInternal restricts access to requests originating from localhost only.
// This is used for internal service endpoints that should not be exposed externally,
// such as the AI endpoints used by MCP tools, edge functions, and jobs.
//
// The middleware checks the actual connection IP, ignoring X-Forwarded-For and
// X-Real-IP headers to prevent header spoofing attacks.
func RequireInternal() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get the actual connection IP (ignore proxy headers for security)
		// We use c.IP() with Fiber's trusted proxy config, but for internal
		// endpoints we want the direct connection IP
		clientIP := getDirectIP(c)

		if !isLoopback(clientIP) {
			log.Warn().
				Str("ip", clientIP.String()).
				Str("path", c.Path()).
				Msg("Internal endpoint access denied - not from localhost")

			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Access denied - internal endpoint",
			})
		}

		return c.Next()
	}
}

// getDirectIP returns the direct connection IP, ignoring proxy headers.
// This is more secure for internal endpoints where we want to verify
// the request truly comes from localhost.
func getDirectIP(c *fiber.Ctx) net.IP {
	// Get the raw IP from the connection, ignoring proxy headers
	// Fiber's c.Context().RemoteIP() gives us the actual connection IP
	ipStr := c.Context().RemoteIP().String()

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
