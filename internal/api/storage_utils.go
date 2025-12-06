package api

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"
)

// detectContentType detects content type from file extension
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
