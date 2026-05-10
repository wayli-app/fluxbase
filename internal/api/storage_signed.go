package api

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/storage"
)

type ipRateLimiter struct {
	mu       sync.Mutex
	requests map[string]*rateLimitEntry
	limit    int
	window   time.Duration
}

type rateLimitEntry struct {
	count     int
	windowEnd time.Time
}

func (r *ipRateLimiter) allow(ip string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	entry, exists := r.requests[ip]

	if !exists || now.After(entry.windowEnd) {
		// New window
		r.requests[ip] = &rateLimitEntry{
			count:     1,
			windowEnd: now.Add(r.window),
		}
		return true
	}

	if entry.count >= r.limit {
		return false
	}

	entry.count++
	return true
}

// GenerateSignedURL generates a presigned URL for temporary access
// POST /api/v1/storage/:bucket/sign/*
func (h *StorageHandler) GenerateSignedURL(c fiber.Ctx) error {
	// Get tenant-specific storage service
	svc, err := h.getService(c)
	if err != nil {
		return SendInternalError(c, "Failed to get storage service")
	}

	bucket := c.Params("bucket")
	key := c.Params("*")

	if bucket == "" || key == "" {
		return SendBadRequest(c, "Bucket and key are required", ErrCodeMissingField)
	}

	// Parse request body
	var req struct {
		ExpiresIn int    `json:"expires_in"` // seconds
		Method    string `json:"method"`     // GET, PUT, DELETE
		// Transform options (for image downloads)
		Transform *struct {
			Width   int    `json:"width"`
			Height  int    `json:"height"`
			Format  string `json:"format"`
			Quality int    `json:"quality"`
			Fit     string `json:"fit"`
		} `json:"transform,omitempty"`
	}
	if err := ParseBody(c, &req); err != nil {
		return err
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

	// Include transform options if specified
	if req.Transform != nil {
		opts.TransformWidth = req.Transform.Width
		opts.TransformHeight = req.Transform.Height
		opts.TransformFormat = req.Transform.Format
		opts.TransformQuality = req.Transform.Quality
		opts.TransformFit = req.Transform.Fit
	}

	url, err := svc.Provider.GenerateSignedURL(c.RequestCtx(), bucket, key, opts)
	if err != nil {
		log.Error().Err(err).Str("bucket", bucket).Str("key", key).Msg("Failed to generate signed URL")
		return SendInternalError(c, "Failed to generate signed URL")
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
func (h *StorageHandler) DownloadSignedObject(c fiber.Ctx) error {
	// Rate limit by IP to prevent DoS via shared signed URLs
	clientIP := c.IP()
	if !h.signedURLLimiter.allow(clientIP) {
		log.Warn().Str("ip", clientIP).Msg("Rate limit exceeded for signed URL download")
		return SendErrorWithCode(c, fiber.StatusTooManyRequests, "Rate limit exceeded, please try again later", ErrCodeRateLimited)
	}

	token := c.Query("token")
	if token == "" {
		return SendBadRequest(c, "Token is required", ErrCodeMissingField)
	}

	// Get tenant-specific storage service
	svc, err := h.getService(c)
	if err != nil {
		return SendInternalError(c, "Failed to get storage service")
	}

	// Only local storage supports signed URL validation
	localStorage, ok := svc.Provider.(*storage.LocalStorage)
	if !ok {
		// For S3, the signed URL is handled directly by S3
		return SendBadRequest(c, "This endpoint is only for local storage signed URLs", ErrCodeInvalidInput)
	}

	// Validate the token (use full validation to get transform options)
	tokenResult, err := localStorage.ValidateSignedTokenFull(token)
	if err != nil {
		log.Warn().Err(err).Msg("Invalid signed URL token")
		return SendUnauthorized(c, "Invalid or expired token", ErrCodeInvalidToken)
	}

	// Verify the request method matches the token
	if tokenResult.Method != c.Method() {
		return SendErrorWithCode(c, fiber.StatusMethodNotAllowed,
			fmt.Sprintf("Token is only valid for %s requests", tokenResult.Method),
			ErrCodeInvalidInput)
	}

	// Download the file (no RLS check - token is the authorization)
	opts := &storage.DownloadOptions{}
	if rangeHeader := c.Get("Range"); rangeHeader != "" {
		opts.Range = rangeHeader
	}

	reader, object, err := svc.Provider.Download(c.RequestCtx(), tokenResult.Bucket, tokenResult.Key, opts)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return SendNotFound(c, "File not found")
		}
		log.Error().Err(err).Str("bucket", tokenResult.Bucket).Str("key", tokenResult.Key).Msg("Failed to download file via signed URL")
		return SendInternalError(c, "Failed to download file")
	}
	defer func() { _ = reader.Close() }()

	contentType := object.ContentType
	contentLength := object.Size

	// Check if transforms are requested and applicable
	hasTransform := tokenResult.TransformWidth > 0 || tokenResult.TransformHeight > 0 ||
		tokenResult.TransformFormat != "" || tokenResult.TransformQuality > 0
	canTransform := h.transformer != nil && storage.CanTransform(object.ContentType)

	if hasTransform && canTransform {
		// Apply image transformation
		transformOpts := storage.ParseTransformOptions(
			tokenResult.TransformWidth,
			tokenResult.TransformHeight,
			tokenResult.TransformFormat,
			tokenResult.TransformQuality,
			tokenResult.TransformFit,
		)

		if transformOpts != nil {
			transformedReader, newContentType, newSize, err := h.transformer.TransformReader(reader, object.ContentType, transformOpts)
			if err != nil {
				log.Error().Err(err).Msg("Failed to transform image for signed URL")
				// Fall back to original file
			} else if transformedReader != nil {
				// Use transformed result
				contentType = newContentType
				contentLength = newSize

				// Set response headers
				c.Set("Content-Type", contentType)
				c.Set("Content-Length", strconv.FormatInt(contentLength, 10))
				c.Set("Last-Modified", object.LastModified.UTC().Format("Mon, 02 Jan 2006 15:04:05 GMT"))
				c.Set("X-Image-Transformed", "true")

				// Set Content-Disposition for download
				filename := filepath.Base(tokenResult.Key)
				// Update extension if format changed
				if tokenResult.TransformFormat != "" {
					ext := "." + tokenResult.TransformFormat
					if tokenResult.TransformFormat == "jpeg" {
						ext = ".jpg"
					}
					filename = strings.TrimSuffix(filename, filepath.Ext(filename)) + ext
				}
				c.Set("Content-Disposition", fmt.Sprintf("inline; filename=\"%s\"", filename))

				return c.SendStream(transformedReader)
			}
		}
	}

	// No transform or transform not applicable - serve original file
	c.Set("Content-Type", contentType)
	c.Set("Content-Length", strconv.FormatInt(contentLength, 10))
	c.Set("Last-Modified", object.LastModified.UTC().Format("Mon, 02 Jan 2006 15:04:05 GMT"))

	// Set Content-Disposition for download
	filename := filepath.Base(tokenResult.Key)
	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))

	// Stream the file
	return c.SendStream(reader)
}

// fiber:context-methods migrated
