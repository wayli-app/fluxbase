package api

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"
)

// detectContentType detects content type from file extension
// SECURITY NOTE: This function only checks file extension, which can be spoofed.
// For enhanced security, consider using detectContentTypeFromBytes() which validates
// magic bytes. However, the primary security control should be:
// 1. Never execute uploaded files
// 2. Serve files with Content-Disposition: attachment
// 3. Use strict CSP headers on storage endpoints
// 4. Implement bucket-level MIME type whitelists
func detectContentType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	contentTypes := map[string]string{
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".png":  "image/png",
		".gif":  "image/gif",
		".pdf":  "application/pdf",
		".txt":  "text/plain",
		".html": "text/html",
		".json": "application/json",
		".xml":  "application/xml",
		".zip":  "application/zip",
		".mp4":  "video/mp4",
		".mp3":  "audio/mpeg",
	}

	if ct, ok := contentTypes[ext]; ok {
		return ct
	}
	return "application/octet-stream"
}

// detectContentTypeFromBytes detects content type from file content using magic bytes
// This is more secure than extension-based detection as it validates actual file content
func detectContentTypeFromBytes(data []byte, filename string) string {
	if len(data) == 0 {
		return detectContentType(filename)
	}

	// Use Go's built-in magic byte detection (reads up to 512 bytes)
	detected := http.DetectContentType(data)

	// Get extension-based type for comparison
	extensionType := detectContentType(filename)

	// For dangerous executable types, always use detected type
	if isDangerousMimeType(extensionType) && detected != extensionType {
		log.Warn().
			Str("filename", filename).
			Str("claimed_type", extensionType).
			Str("detected_type", detected).
			Msg("MIME type mismatch detected - using magic byte detection")
		return detected
	}

	// For non-dangerous types, prefer extension-based type as magic bytes
	// can be less accurate for text-based formats
	return extensionType
}

// isDangerousMimeType checks if a MIME type is potentially dangerous for inline display
func isDangerousMimeType(mimeType string) bool {
	dangerousTypes := map[string]bool{
		"text/html":               true, // Could contain scripts
		"application/javascript":  true,
		"text/javascript":         true,
		"application/x-httpd-php": true,
		"application/xhtml+xml":   true,
		"image/svg+xml":           true, // SVG can contain scripts
	}
	return dangerousTypes[mimeType]
}

// parseMetadata parses metadata from form fields starting with "metadata_"
func parseMetadata(c *fiber.Ctx) map[string]string {
	metadata := make(map[string]string)

	c.Request().PostArgs().VisitAll(func(key, value []byte) {
		keyStr := string(key)
		if strings.HasPrefix(keyStr, "metadata_") {
			metaKey := strings.TrimPrefix(keyStr, "metadata_")
			metadata[metaKey] = string(value)
		}
	})

	return metadata
}

// getUserID gets the user ID from Fiber context
func getUserID(c *fiber.Ctx) string {
	if userID := c.Locals("user_id"); userID != nil {
		if id, ok := userID.(string); ok {
			return id
		}
	}
	return "anonymous"
}

// setRLSContext sets PostgreSQL session variables for RLS enforcement in a transaction
func (h *StorageHandler) setRLSContext(ctx context.Context, tx pgx.Tx, c *fiber.Ctx) error {
	// Get user ID and role from context
	userID := c.Locals("user_id")
	role := c.Locals("user_role")

	// Determine the role
	var roleStr string
	if role != nil {
		if r, ok := role.(string); ok {
			roleStr = r
		}
	}

	// Default role based on authentication state
	if roleStr == "" {
		if userID != nil {
			roleStr = "authenticated"
		} else {
			roleStr = "anon"
		}
	}

	// Convert userID to string
	var userIDStr string
	if userID != nil {
		userIDStr = fmt.Sprintf("%v", userID)
	}

	// Set request.jwt.claims with user ID and role (Supabase/Fluxbase format)
	// This is read by auth.current_user_id() and auth.current_user_role() functions
	var jwtClaims string
	if userIDStr != "" {
		jwtClaims = fmt.Sprintf(`{"sub":"%s","role":"%s"}`, userIDStr, roleStr)
	} else {
		jwtClaims = fmt.Sprintf(`{"role":"%s"}`, roleStr)
	}

	if _, err := tx.Exec(ctx, "SELECT set_config('request.jwt.claims', $1, true)", jwtClaims); err != nil {
		return fmt.Errorf("failed to set request.jwt.claims: %w", err)
	}

	log.Debug().Str("user_id", userIDStr).Str("role", roleStr).Msg("Set RLS context for storage operation")
	return nil
}
