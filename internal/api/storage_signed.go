package api

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/fluxbase-eu/fluxbase/internal/storage"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
)

// GenerateSignedURL generates a presigned URL for temporary access
// POST /api/v1/storage/:bucket/sign/*
func (h *StorageHandler) GenerateSignedURL(c *fiber.Ctx) error {
	bucket := c.Params("bucket")
	key := c.Params("*")

	if bucket == "" || key == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "bucket and key are required",
		})
	}

	// Parse request body
	var req struct {
		ExpiresIn int    `json:"expires_in"` // seconds
		Method    string `json:"method"`     // GET, PUT, DELETE
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	// Default values
	if req.ExpiresIn == 0 {
		req.ExpiresIn = 900 // 15 minutes
	}
	if req.Method == "" {
		req.Method = "GET"
	}

	// Generate signed URL
	opts := &storage.SignedURLOptions{
		ExpiresIn: time.Duration(req.ExpiresIn) * time.Second,
		Method:    req.Method,
	}

	url, err := h.storage.Provider.GenerateSignedURL(c.Context(), bucket, key, opts)
	if err != nil {
		log.Error().Err(err).Str("bucket", bucket).Str("key", key).Msg("Failed to generate signed URL")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to generate signed URL",
		})
	}

	return c.JSON(fiber.Map{
		"signed_url": url,
		"expires_in": req.ExpiresIn,
		"method":     req.Method,
	})
}

// DownloadSignedObject handles file downloads via signed URL tokens
// GET /api/v1/storage/object?token=...
// This is a PUBLIC endpoint - authentication is provided by the signed token
func (h *StorageHandler) DownloadSignedObject(c *fiber.Ctx) error {
	token := c.Query("token")
	if token == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "token is required",
		})
	}

	// Only local storage supports signed URL validation
	localStorage, ok := h.storage.Provider.(*storage.LocalStorage)
	if !ok {
		// For S3, the signed URL is handled directly by S3
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "this endpoint is only for local storage signed URLs",
		})
	}

	// Validate the token
	bucket, key, method, err := localStorage.ValidateSignedToken(token)
	if err != nil {
		log.Warn().Err(err).Msg("Invalid signed URL token")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "invalid or expired token",
		})
	}

	// Verify the request method matches the token
	if method != c.Method() {
		return c.Status(fiber.StatusMethodNotAllowed).JSON(fiber.Map{
			"error": fmt.Sprintf("token is only valid for %s requests", method),
		})
	}

	// Download the file (no RLS check - token is the authorization)
	opts := &storage.DownloadOptions{}
	if rangeHeader := c.Get("Range"); rangeHeader != "" {
		opts.Range = rangeHeader
	}

	reader, object, err := h.storage.Provider.Download(c.Context(), bucket, key, opts)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "file not found",
			})
		}
		log.Error().Err(err).Str("bucket", bucket).Str("key", key).Msg("Failed to download file via signed URL")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to download file",
		})
	}

	// Set response headers
	c.Set("Content-Type", object.ContentType)
	c.Set("Content-Length", strconv.FormatInt(object.Size, 10))
	c.Set("Last-Modified", object.LastModified.UTC().Format("Mon, 02 Jan 2006 15:04:05 GMT"))

	// Set Content-Disposition for download
	filename := filepath.Base(key)
	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))

	// Stream the file
	return c.SendStream(reader)
}
