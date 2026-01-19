package api

import (
	"bytes"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/fluxbase-eu/fluxbase/internal/storage"
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"
)

// UploadFile handles file upload
// POST /api/v1/storage/:bucket/:key
func (h *StorageHandler) UploadFile(c *fiber.Ctx) error {
	bucket := c.Params("bucket")
	key := c.Params("*") // Capture the rest of the path

	if bucket == "" || key == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "bucket and key are required",
		})
	}

	// Get file from form data
	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "file is required",
		})
	}

	// Validate file size against global limit
	if err := h.storage.ValidateUploadSize(file.Size); err != nil {
		return c.Status(fiber.StatusRequestEntityTooLarge).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Get bucket settings for additional validation
	var bucketMaxFileSize *int64
	var bucketAllowedMimeTypes []string
	err = h.db.Pool().QueryRow(c.Context(),
		`SELECT max_file_size, allowed_mime_types FROM storage.buckets WHERE name = $1`,
		bucket,
	).Scan(&bucketMaxFileSize, &bucketAllowedMimeTypes)
	if err != nil && err != pgx.ErrNoRows {
		log.Error().Err(err).Str("bucket", bucket).Msg("Failed to get bucket settings")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to validate bucket settings",
		})
	}

	// Validate file size against bucket-specific limit
	if bucketMaxFileSize != nil && *bucketMaxFileSize > 0 && file.Size > *bucketMaxFileSize {
		return c.Status(fiber.StatusRequestEntityTooLarge).JSON(fiber.Map{
			"error": fmt.Sprintf("file size %d exceeds bucket maximum of %d bytes", file.Size, *bucketMaxFileSize),
		})
	}

	// Detect content type early for MIME validation
	contentType := file.Header.Get("Content-Type")
	if contentType == "" {
		contentType = detectContentType(file.Filename)
	}

	// Validate MIME type against bucket-specific allowed types
	if len(bucketAllowedMimeTypes) > 0 {
		mimeAllowed := false
		for _, allowedType := range bucketAllowedMimeTypes {
			if allowedType == contentType || allowedType == "*/*" {
				mimeAllowed = true
				break
			}
			// Support wildcard matching (e.g., "image/*")
			if strings.HasSuffix(allowedType, "/*") {
				prefix := strings.TrimSuffix(allowedType, "/*")
				if strings.HasPrefix(contentType, prefix+"/") {
					mimeAllowed = true
					break
				}
			}
		}
		if !mimeAllowed {
			return c.Status(fiber.StatusUnsupportedMediaType).JSON(fiber.Map{
				"error": fmt.Sprintf("file type %s is not allowed for this bucket", contentType),
			})
		}
	}

	// Open the uploaded file
	src, err := file.Open()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to open uploaded file",
		})
	}
	defer func() { _ = src.Close() }()

	// Parse metadata from form
	metadata := parseMetadata(c)

	// Upload options
	opts := &storage.UploadOptions{
		ContentType: contentType,
		Metadata:    metadata,
	}

	// Get owner ID from authenticated user
	ownerID := getUserID(c)
	var ownerUUID *string
	if ownerID != "" && ownerID != "anonymous" {
		ownerUUID = &ownerID
	}

	ctx := c.Context()

	// Upload the file to storage provider first
	object, err := h.storage.Provider.Upload(ctx, bucket, key, src, file.Size, opts)
	if err != nil {
		log.Error().Err(err).Str("bucket", bucket).Str("key", key).Msg("Failed to upload file")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to upload file",
		})
	}

	// Start database transaction to store metadata
	tx, err := h.db.Pool().Begin(ctx)
	if err != nil {
		// Delete from provider since DB insert failed
		_ = h.storage.Provider.Delete(ctx, bucket, key)
		log.Error().Err(err).Msg("Failed to start transaction for file upload")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to save file metadata",
		})
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Set RLS context
	if err := h.setRLSContext(ctx, tx, c); err != nil {
		_ = h.storage.Provider.Delete(ctx, bucket, key)
		log.Error().Err(err).Msg("Failed to set RLS context")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to save file metadata",
		})
	}

	// Convert metadata map to JSONB
	var metadataJSON map[string]interface{}
	if len(metadata) > 0 {
		metadataJSON = make(map[string]interface{})
		for k, v := range metadata {
			metadataJSON[k] = v
		}
	}

	// Insert object metadata into database (RLS will check permissions)
	_, err = tx.Exec(ctx, `
		INSERT INTO storage.objects (bucket_id, path, mime_type, size, metadata, owner_id)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (bucket_id, path)
		DO UPDATE SET mime_type = $3, size = $4, metadata = $5, owner_id = $6, updated_at = NOW()
	`, bucket, key, contentType, file.Size, metadataJSON, ownerUUID)

	if err != nil {
		// Delete from provider since DB insert failed
		_ = h.storage.Provider.Delete(ctx, bucket, key)

		// Log the full error for debugging
		errMsg := err.Error()
		log.Error().
			Err(err).
			Str("bucket", bucket).
			Str("key", key).
			Str("owner_id", fmt.Sprintf("%v", ownerUUID)).
			Str("error_message", errMsg).
			Msg("Failed to insert file metadata into database")

		if strings.Contains(errMsg, "permission denied") || strings.Contains(errMsg, "policy") {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error":  "insufficient permissions to upload file",
				"detail": errMsg,
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to save file metadata",
		})
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		_ = h.storage.Provider.Delete(ctx, bucket, key)
		log.Error().Err(err).Str("bucket", bucket).Str("key", key).Msg("Failed to commit file upload")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to save file metadata",
		})
	}

	log.Info().
		Str("bucket", bucket).
		Str("key", key).
		Int64("size", object.Size).
		Str("user_id", ownerID).
		Msg("File uploaded")

	// Add owner_id to response
	response := map[string]interface{}{
		"key":           object.Key,
		"bucket":        object.Bucket,
		"size":          object.Size,
		"content_type":  object.ContentType,
		"last_modified": object.LastModified,
	}
	if ownerUUID != nil {
		response["owner_id"] = *ownerUUID
	}

	return c.Status(fiber.StatusCreated).JSON(response)
}

// DownloadFile handles file download and HEAD requests for file info
// GET /api/v1/storage/:bucket/:key
// HEAD /api/v1/storage/:bucket/:key (for downloadResumable to get Content-Length)
func (h *StorageHandler) DownloadFile(c *fiber.Ctx) error {
	bucket := c.Params("bucket")
	key := c.Params("*")

	// For HEAD requests, delegate to GetFileInfo which returns proper headers
	if c.Method() == "HEAD" {
		return h.GetFileInfo(c)
	}

	// If key is empty, this might be a list files request
	// Forward to ListFiles handler
	if key == "" {
		return h.ListFiles(c)
	}

	if bucket == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "bucket is required",
		})
	}

	ctx := c.Context()

	// Start database transaction to check permissions
	tx, err := h.db.Pool().Begin(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to start transaction for file download")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to download file",
		})
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Set RLS context
	if err := h.setRLSContext(ctx, tx, c); err != nil {
		log.Error().Err(err).Msg("Failed to set RLS context")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to download file",
		})
	}

	// Check if user has permission to access this file (RLS will filter)
	var objectID string
	var mimeType string
	var fileSize int64
	err = tx.QueryRow(ctx, `
		SELECT id, mime_type, size
		FROM storage.objects
		WHERE bucket_id = $1 AND path = $2
	`, bucket, key).Scan(&objectID, &mimeType, &fileSize)

	if err != nil {
		if err == pgx.ErrNoRows {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "file not found or insufficient permissions",
			})
		}
		log.Error().Err(err).Str("bucket", bucket).Str("key", key).Msg("Failed to check file permissions")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to download file",
		})
	}

	// Commit transaction (permission check passed)
	if err := tx.Commit(ctx); err != nil {
		log.Error().Err(err).Msg("Failed to commit file download transaction")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to download file",
		})
	}

	// Parse download options
	opts := &storage.DownloadOptions{}

	// Support range requests
	if rangeHeader := c.Get("Range"); rangeHeader != "" {
		opts.Range = rangeHeader
	}

	// Download the file from provider
	reader, object, err := h.storage.Provider.Download(ctx, bucket, key, opts)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "file not found",
			})
		}
		log.Error().Err(err).Str("bucket", bucket).Str("key", key).Msg("Failed to download file from provider")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to download file",
		})
	}

	// Parse transform options from query parameters
	transformOpts := storage.ParseTransformOptions(
		c.QueryInt("w", c.QueryInt("width", 0)),
		c.QueryInt("h", c.QueryInt("height", 0)),
		c.Query("fmt", c.Query("format", "")),
		c.QueryInt("q", c.QueryInt("quality", 0)),
		c.Query("fit", ""),
	)

	// Apply image transformation if enabled and requested
	responseReader := reader
	responseContentType := object.ContentType
	responseSize := object.Size

	if transformOpts != nil && h.transformer != nil && storage.CanTransform(object.ContentType) {
		// Check rate limit for transforms
		limiterKey := c.IP() + ":" + getUserID(c)
		if limiter := h.getTransformLimiter(limiterKey); limiter != nil && !limiter.Allow() {
			_ = reader.Close()
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": "transform rate limit exceeded",
			})
		}

		// Check cache first (before acquiring transform slot)
		if h.transformCache != nil {
			if cached, contentType, ok := h.transformCache.Get(ctx, bucket, key, transformOpts); ok {
				_ = reader.Close() // Close original reader since we're using cache
				responseReader = io.NopCloser(bytes.NewReader(cached))
				responseContentType = contentType
				responseSize = int64(len(cached))
				log.Debug().Str("bucket", bucket).Str("key", key).Msg("Serving transformed image from cache")
				goto sendResponse
			}
		}

		// Acquire transform slot (concurrency limit)
		if !h.acquireTransformSlot(5 * time.Second) {
			_ = reader.Close()
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"error": "transform service busy, try again later",
			})
		}
		defer h.releaseTransformSlot()

		// Apply transformation
		result, err := h.transformer.Transform(reader, object.ContentType, transformOpts)
		_ = reader.Close() // Close original reader since we read all data

		if err != nil {
			// Log the error but return the original if transform fails
			log.Warn().Err(err).Str("bucket", bucket).Str("key", key).Msg("Image transform failed, returning original")

			// Re-download original since we consumed the reader
			reader, object, err = h.storage.Provider.Download(ctx, bucket, key, opts)
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "failed to download file",
				})
			}
			responseReader = reader
		} else if result != nil {
			// Cache the transformed result
			if h.transformCache != nil {
				if err := h.transformCache.Set(ctx, bucket, key, transformOpts, result.Data, result.ContentType); err != nil {
					log.Warn().Err(err).Str("bucket", bucket).Str("key", key).Msg("Failed to cache transformed image")
				}
			}

			responseReader = io.NopCloser(bytes.NewReader(result.Data))
			responseContentType = result.ContentType
			responseSize = int64(len(result.Data))
		} else {
			// No transformation was applied (result is nil)
			// Re-download original since we consumed the reader
			reader, object, err = h.storage.Provider.Download(ctx, bucket, key, opts)
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "failed to download file",
				})
			}
			responseReader = reader
		}
	}

sendResponse:
	// Note: Don't defer responseReader.Close() here - SendStream handles closing the reader

	// Set response headers
	c.Set("Content-Type", responseContentType)
	c.Set("Content-Length", strconv.FormatInt(responseSize, 10))
	c.Set("Last-Modified", object.LastModified.Format(time.RFC1123))
	c.Set("ETag", object.ETag)
	c.Set("Accept-Ranges", "bytes")

	// Disable range requests for transformed images (size is different)
	if transformOpts != nil && h.transformer != nil && storage.CanTransform(object.ContentType) {
		c.Set("Accept-Ranges", "none")
	} else {
		// Handle range request response for non-transformed images
		if rangeHeader := c.Get("Range"); rangeHeader != "" {
			// Parse range to set Content-Range header
			var start, end int64
			if _, err := fmt.Sscanf(rangeHeader, "bytes=%d-%d", &start, &end); err == nil {
				c.Set("Content-Range", fmt.Sprintf("bytes %d-%d/*", start, start+object.Size-1))
				c.Status(fiber.StatusPartialContent)
			}
		}
	}

	// Set content disposition - default to attachment for security
	// Only allow inline for safe MIME types when explicitly requested
	filename := filepath.Base(key)
	// If format was changed, update the filename extension
	if transformOpts != nil && transformOpts.Format != "" {
		ext := filepath.Ext(filename)
		if ext != "" {
			filename = strings.TrimSuffix(filename, ext) + "." + transformOpts.Format
		}
	}

	disposition := "attachment"
	if c.Query("inline") == "true" {
		// Only allow inline for safe content types
		safeTypes := map[string]bool{
			"image/jpeg":      true,
			"image/png":       true,
			"image/gif":       true,
			"image/webp":      true,
			"application/pdf": true,
			"video/mp4":       true,
			"audio/mpeg":      true,
		}
		if safeTypes[object.ContentType] {
			disposition = "inline"
		}
	}
	c.Set("Content-Disposition", fmt.Sprintf("%s; filename=\"%s\"", disposition, filename))

	log.Debug().
		Str("bucket", bucket).
		Str("key", key).
		Str("user_id", getUserID(c)).
		Bool("transformed", transformOpts != nil && h.transformer != nil).
		Msg("File downloaded")

	// Stream the file (SendStream will close the reader)
	return c.SendStream(responseReader)
}

// Suppress unused import warning for bytes package (used for transformed image streaming)
var _ = bytes.NewReader

// DeleteFile handles file deletion
// DELETE /api/v1/storage/:bucket/:key
func (h *StorageHandler) DeleteFile(c *fiber.Ctx) error {
	bucket := c.Params("bucket")
	key := c.Params("*")

	if bucket == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "bucket is required",
		})
	}

	if key == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "key is required",
		})
	}

	ctx := c.Context()

	// Start database transaction
	tx, err := h.db.Pool().Begin(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to start transaction for file deletion")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to delete file",
		})
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Set RLS context
	if err := h.setRLSContext(ctx, tx, c); err != nil {
		log.Error().Err(err).Msg("Failed to set RLS context")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to delete file",
		})
	}

	// Delete from database (RLS will check permissions)
	result, err := tx.Exec(ctx, `
		DELETE FROM storage.objects
		WHERE bucket_id = $1 AND path = $2
	`, bucket, key)

	if err != nil {
		if strings.Contains(err.Error(), "permission denied") || strings.Contains(err.Error(), "policy") {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "insufficient permissions to delete file",
			})
		}
		log.Error().Err(err).Str("bucket", bucket).Str("key", key).Msg("Failed to delete file from database")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to delete file",
		})
	}

	// Check if any rows were affected (file existed and was deleted)
	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		// No rows affected - need to determine if it's 403 (RLS blocked) or 404 (not found)
		// Check if file exists using superuser context to bypass RLS
		var fileExists bool
		err = h.db.Pool().QueryRow(ctx, `
			SELECT EXISTS(SELECT 1 FROM storage.objects WHERE bucket_id = $1 AND path = $2)
		`, bucket, key).Scan(&fileExists)

		if err != nil {
			// If we can't check existence, log it but still return 404
			// This is safer than returning 500 for a delete operation
			log.Warn().Err(err).Str("bucket", bucket).Str("key", key).Msg("Failed to check file existence after delete returned 0 rows")
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "file not found",
			})
		}

		if fileExists {
			// File exists but RLS prevented delete - return 403
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "insufficient permissions to delete file",
			})
		}
		// File doesn't exist at all
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "file not found",
		})
	}

	// Delete from storage provider
	if err := h.storage.Provider.Delete(ctx, bucket, key); err != nil {
		log.Warn().Err(err).Str("bucket", bucket).Str("key", key).Msg("Failed to delete file from provider (metadata already deleted)")
	}

	// Invalidate transform cache for this file
	if h.transformCache != nil {
		if err := h.transformCache.Invalidate(ctx, bucket, key); err != nil {
			log.Warn().Err(err).Str("bucket", bucket).Str("key", key).Msg("Failed to invalidate transform cache")
		}
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		log.Error().Err(err).Str("bucket", bucket).Str("key", key).Msg("Failed to commit file deletion")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to delete file",
		})
	}

	log.Info().
		Str("bucket", bucket).
		Str("key", key).
		Str("user_id", getUserID(c)).
		Msg("File deleted")

	return c.Status(fiber.StatusNoContent).Send(nil)
}

// GetFileInfo handles getting file metadata
// HEAD /api/v1/storage/:bucket/:key
func (h *StorageHandler) GetFileInfo(c *fiber.Ctx) error {
	bucket := c.Params("bucket")
	key := c.Params("*")

	if key == "" {
		return h.ListFiles(c)
	}

	if bucket == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "bucket is required",
		})
	}

	ctx := c.Context()

	tx, err := h.db.Pool().Begin(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to start transaction for getting file info")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get file info",
		})
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := h.setRLSContext(ctx, tx, c); err != nil {
		log.Error().Err(err).Msg("Failed to set RLS context")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get file info",
		})
	}

	var id string
	var mimeType *string
	var size int64
	var metadata map[string]interface{}
	var ownerID *string
	var createdAt, updatedAt time.Time

	err = tx.QueryRow(ctx, `
		SELECT id, mime_type, size, metadata, owner_id, created_at, updated_at
		FROM storage.objects
		WHERE bucket_id = $1 AND path = $2
	`, bucket, key).Scan(&id, &mimeType, &size, &metadata, &ownerID, &createdAt, &updatedAt)

	if err != nil {
		if err == pgx.ErrNoRows {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "file not found or insufficient permissions",
			})
		}
		log.Error().Err(err).Str("bucket", bucket).Str("key", key).Msg("Failed to get file metadata")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get file info",
		})
	}

	if err := tx.Commit(ctx); err != nil {
		log.Error().Err(err).Msg("Failed to commit get file info transaction")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get file info",
		})
	}

	contentType := "application/octet-stream"
	if mimeType != nil {
		contentType = *mimeType
	}

	if c.Method() == "HEAD" {
		log.Debug().Str("bucket", bucket).Str("key", key).Int64("size", size).Msg("HEAD request")
		c.Response().Header.SetContentType(contentType)
		c.Response().Header.SetContentLength(int(size))
		c.Response().Header.Set("Accept-Ranges", "bytes")
		c.Response().Header.Set("Last-Modified", updatedAt.Format(time.RFC1123))
		c.Status(fiber.StatusOK)
		return nil
	}

	c.Set("Content-Type", contentType)
	c.Set("Content-Length", strconv.FormatInt(size, 10))
	c.Set("Accept-Ranges", "bytes")
	c.Set("Last-Modified", updatedAt.Format(time.RFC1123))

	response := map[string]interface{}{
		"id": id, "bucket": bucket, "path": key, "size": size,
		"created_at": createdAt, "updated_at": updatedAt,
	}
	if mimeType != nil {
		response["mime_type"] = *mimeType
	}
	if metadata != nil {
		response["metadata"] = metadata
	}
	if ownerID != nil {
		response["owner_id"] = *ownerID
	}

	return c.JSON(response)
}

// ListFiles handles listing files in a bucket
// GET /api/v1/storage/:bucket
func (h *StorageHandler) ListFiles(c *fiber.Ctx) error {
	bucket := c.Params("bucket")

	if bucket == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "bucket is required",
		})
	}

	prefix := c.Query("prefix", "")
	delimiter := c.Query("delimiter", "")
	limit := c.QueryInt("limit", 1000)
	offset := c.QueryInt("offset", 0)

	ctx := c.Context()

	tx, err := h.db.Pool().Begin(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to start transaction for listing files")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to list files",
		})
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := h.setRLSContext(ctx, tx, c); err != nil {
		log.Error().Err(err).Msg("Failed to set RLS context")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to list files",
		})
	}

	type StorageObject struct {
		ID        string                 `json:"id"`
		Bucket    string                 `json:"bucket"`
		Path      string                 `json:"path"`
		MimeType  *string                `json:"mime_type"`
		Size      int64                  `json:"size"`
		Metadata  map[string]interface{} `json:"metadata"`
		OwnerID   *string                `json:"owner_id"`
		CreatedAt time.Time              `json:"created_at"`
		UpdatedAt time.Time              `json:"updated_at"`
	}

	var objects []StorageObject
	var prefixes []string

	if delimiter != "" {
		objectsQuery := `
			SELECT id, bucket_id, path, mime_type, size, metadata, owner_id, created_at, updated_at
			FROM storage.objects
			WHERE bucket_id = $1 AND path LIKE $2 || '%'
			  AND position($3 in substring(path from length($2)+1)) = 0
			ORDER BY path ASC LIMIT $4 OFFSET $5
		`
		rows, err := tx.Query(ctx, objectsQuery, bucket, prefix, delimiter, limit, offset)
		if err != nil {
			log.Error().Err(err).Str("bucket", bucket).Msg("Failed to query files")
			return SendOperationFailed(c, "list files")
		}
		defer rows.Close()

		for rows.Next() {
			var obj StorageObject
			if err := rows.Scan(&obj.ID, &obj.Bucket, &obj.Path, &obj.MimeType, &obj.Size, &obj.Metadata, &obj.OwnerID, &obj.CreatedAt, &obj.UpdatedAt); err != nil {
				log.Error().Err(err).Msg("Failed to scan object row")
				continue
			}
			objects = append(objects, obj)
		}

		prefixesQuery := `
			SELECT DISTINCT $2 || split_part(substring(path from length($2)+1), $3, 1) || $3 as prefix
			FROM storage.objects
			WHERE bucket_id = $1 AND path LIKE $2 || '%' AND position($3 in substring(path from length($2)+1)) > 0
			ORDER BY prefix ASC
		`
		prefixRows, err := tx.Query(ctx, prefixesQuery, bucket, prefix, delimiter)
		if err != nil {
			log.Error().Err(err).Str("bucket", bucket).Msg("Failed to query prefixes")
			return SendOperationFailed(c, "list files")
		}
		defer prefixRows.Close()

		for prefixRows.Next() {
			var p string
			if err := prefixRows.Scan(&p); err != nil {
				continue
			}
			prefixes = append(prefixes, p)
		}
	} else {
		query := `SELECT id, bucket_id, path, mime_type, size, metadata, owner_id, created_at, updated_at FROM storage.objects WHERE bucket_id = $1`
		args := []interface{}{bucket}
		argCount := 1

		if prefix != "" {
			argCount++
			query += fmt.Sprintf(" AND path LIKE $%d", argCount)
			args = append(args, prefix+"%")
		}
		query += " ORDER BY path ASC"
		if limit > 0 {
			argCount++
			query += fmt.Sprintf(" LIMIT $%d", argCount)
			args = append(args, limit)
		}
		if offset > 0 {
			argCount++
			query += fmt.Sprintf(" OFFSET $%d", argCount)
			args = append(args, offset)
		}

		rows, err := tx.Query(ctx, query, args...)
		if err != nil {
			log.Error().Err(err).Str("bucket", bucket).Msg("Failed to query files")
			return SendOperationFailed(c, "list files")
		}
		defer rows.Close()

		for rows.Next() {
			var obj StorageObject
			if err := rows.Scan(&obj.ID, &obj.Bucket, &obj.Path, &obj.MimeType, &obj.Size, &obj.Metadata, &obj.OwnerID, &obj.CreatedAt, &obj.UpdatedAt); err != nil {
				continue
			}
			objects = append(objects, obj)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		log.Error().Err(err).Msg("Failed to commit file list transaction")
		return SendOperationFailed(c, "list files")
	}

	response := fiber.Map{"bucket": bucket, "objects": objects, "count": len(objects)}
	if delimiter != "" {
		response["prefix"] = prefix
		response["prefixes"] = prefixes
	}

	return c.JSON(response)
}
