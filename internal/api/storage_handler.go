package api

import (
	"context"
	"fmt"
	"mime/multipart"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/fluxbase-eu/fluxbase/internal/database"
	"github.com/fluxbase-eu/fluxbase/internal/storage"
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"
)

// StorageHandler handles file storage operations
type StorageHandler struct {
	storage *storage.Service
	db      *database.Connection
}

// NewStorageHandler creates a new storage handler
func NewStorageHandler(storage *storage.Service, db *database.Connection) *StorageHandler {
	return &StorageHandler{
		storage: storage,
		db:      db,
	}
}

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

	// Validate file size
	if err := h.storage.ValidateUploadSize(file.Size); err != nil {
		return c.Status(fiber.StatusRequestEntityTooLarge).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Open the uploaded file
	src, err := file.Open()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to open uploaded file",
		})
	}
	defer src.Close()

	// Detect content type
	contentType := file.Header.Get("Content-Type")
	if contentType == "" {
		contentType = detectContentType(file.Filename)
	}

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
	defer tx.Rollback(ctx)

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
				"detail": errMsg, // Include error detail for debugging
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
	defer tx.Rollback(ctx)

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
	// Note: Don't defer reader.Close() here - SendStream handles closing the reader

	// Set response headers
	c.Set("Content-Type", object.ContentType)
	c.Set("Content-Length", strconv.FormatInt(object.Size, 10))
	c.Set("Last-Modified", object.LastModified.Format(time.RFC1123))
	c.Set("ETag", object.ETag)
	c.Set("Accept-Ranges", "bytes")

	// Handle range request response
	if rangeHeader := c.Get("Range"); rangeHeader != "" {
		// Parse range to set Content-Range header
		var start, end int64
		if _, err := fmt.Sscanf(rangeHeader, "bytes=%d-%d", &start, &end); err == nil {
			// Note: object.Size now contains the chunk size, not total file size
			// We set Content-Range with the actual bytes returned
			c.Set("Content-Range", fmt.Sprintf("bytes %d-%d/*", start, start+object.Size-1))
			c.Status(fiber.StatusPartialContent)
		}
	}

	// Set content disposition (for downloads)
	if c.Query("download") == "true" {
		filename := filepath.Base(key)
		c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	}

	log.Debug().
		Str("bucket", bucket).
		Str("key", key).
		Str("user_id", getUserID(c)).
		Msg("File downloaded")

	// Stream the file (SendStream will close the reader)
	return c.SendStream(reader)
}

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
	defer tx.Rollback(ctx)

	// Set RLS context
	if err := h.setRLSContext(ctx, tx, c); err != nil {
		log.Error().Err(err).Msg("Failed to set RLS context")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to delete file",
		})
	}

	// First check if file exists (with superuser context to bypass RLS)
	var fileExists bool
	err = h.db.Pool().QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM storage.objects WHERE bucket_id = $1 AND path = $2)
	`, bucket, key).Scan(&fileExists)

	if err != nil {
		log.Error().Err(err).Str("bucket", bucket).Str("key", key).Msg("Failed to check file existence")
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
		// File exists but RLS prevented delete - return 403 instead of 404
		if fileExists {
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
		// Log error but don't fail - database is source of truth
		log.Warn().Err(err).Str("bucket", bucket).Str("key", key).Msg("Failed to delete file from provider (metadata already deleted)")
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

	// If key is empty, forward to ListFiles
	if key == "" {
		return h.ListFiles(c)
	}

	if bucket == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "bucket is required",
		})
	}

	ctx := c.Context()

	// Start database transaction
	tx, err := h.db.Pool().Begin(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to start transaction for getting file info")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get file info",
		})
	}
	defer tx.Rollback(ctx)

	// Set RLS context
	if err := h.setRLSContext(ctx, tx, c); err != nil {
		log.Error().Err(err).Msg("Failed to set RLS context")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get file info",
		})
	}

	// Query object metadata from database (RLS will filter based on permissions)
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
		log.Error().Err(err).Str("bucket", bucket).Str("key", key).Msg("Failed to get file metadata from database")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get file info",
		})
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		log.Error().Err(err).Msg("Failed to commit get file info transaction")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get file info",
		})
	}

	// Set HTTP headers (needed for HEAD requests and resumable downloads)
	contentType := "application/octet-stream"
	if mimeType != nil {
		contentType = *mimeType
	}

	// For HEAD requests, set headers directly on the response to prevent Fiber from overwriting
	// This is critical for downloadResumable which needs accurate Content-Length header
	if c.Method() == "HEAD" {
		log.Debug().
			Str("bucket", bucket).
			Str("key", key).
			Int64("size", size).
			Str("content_type", contentType).
			Msg("HEAD request - returning file info headers")

		c.Response().Header.SetContentType(contentType)
		c.Response().Header.SetContentLength(int(size))
		c.Response().Header.Set("Accept-Ranges", "bytes")
		c.Response().Header.Set("Last-Modified", updatedAt.Format(time.RFC1123))
		c.Status(fiber.StatusOK)
		return nil
	}

	// For non-HEAD requests, use Fiber's c.Set which is fine for JSON responses
	c.Set("Content-Type", contentType)
	c.Set("Content-Length", strconv.FormatInt(size, 10))
	c.Set("Accept-Ranges", "bytes")
	c.Set("Last-Modified", updatedAt.Format(time.RFC1123))

	// For other methods (e.g., if this handler is called via GET for metadata),
	// return full metadata as JSON
	response := map[string]interface{}{
		"id":         id,
		"bucket":     bucket,
		"path":       key,
		"size":       size,
		"created_at": createdAt,
		"updated_at": updatedAt,
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

	// Parse query parameters
	prefix := c.Query("prefix", "")
	delimiter := c.Query("delimiter", "")
	limit := c.QueryInt("limit", 1000)
	offset := c.QueryInt("offset", 0)

	ctx := c.Context()

	// Start database transaction
	tx, err := h.db.Pool().Begin(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to start transaction for listing files")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to list files",
		})
	}
	defer tx.Rollback(ctx)

	// Set RLS context
	if err := h.setRLSContext(ctx, tx, c); err != nil {
		log.Error().Err(err).Msg("Failed to set RLS context")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to list files",
		})
	}

	// Parse results
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
		// Efficient SQL-based delimiter logic for folder simulation
		// Query 1: Get objects at current level (paths without delimiter after prefix)
		objectsQuery := `
			SELECT id, bucket_id, path, mime_type, size, metadata, owner_id, created_at, updated_at
			FROM storage.objects
			WHERE bucket_id = $1
			  AND path LIKE $2 || '%'
			  AND position($3 in substring(path from length($2)+1)) = 0
			ORDER BY path ASC
			LIMIT $4 OFFSET $5
		`
		rows, err := tx.Query(ctx, objectsQuery, bucket, prefix, delimiter, limit, offset)
		if err != nil {
			log.Error().Err(err).Str("bucket", bucket).Msg("Failed to query files from database")
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to list files",
			})
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

		if err := rows.Err(); err != nil {
			log.Error().Err(err).Msg("Error iterating object rows")
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to list files",
			})
		}

		// Query 2: Get distinct folder prefixes (common prefixes)
		prefixesQuery := `
			SELECT DISTINCT
			  $2 || split_part(substring(path from length($2)+1), $3, 1) || $3 as prefix
			FROM storage.objects
			WHERE bucket_id = $1
			  AND path LIKE $2 || '%'
			  AND position($3 in substring(path from length($2)+1)) > 0
			ORDER BY prefix ASC
		`
		prefixRows, err := tx.Query(ctx, prefixesQuery, bucket, prefix, delimiter)
		if err != nil {
			log.Error().Err(err).Str("bucket", bucket).Msg("Failed to query prefixes from database")
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to list files",
			})
		}
		defer prefixRows.Close()

		for prefixRows.Next() {
			var p string
			if err := prefixRows.Scan(&p); err != nil {
				log.Error().Err(err).Msg("Failed to scan prefix row")
				continue
			}
			prefixes = append(prefixes, p)
		}

		if err := prefixRows.Err(); err != nil {
			log.Error().Err(err).Msg("Error iterating prefix rows")
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to list files",
			})
		}
	} else {
		// Original query without delimiter (flat list)
		query := `
			SELECT id, bucket_id, path, mime_type, size, metadata, owner_id, created_at, updated_at
			FROM storage.objects
			WHERE bucket_id = $1
		`
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
			log.Error().Err(err).Str("bucket", bucket).Msg("Failed to query files from database")
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to list files",
			})
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

		if err := rows.Err(); err != nil {
			log.Error().Err(err).Msg("Error iterating object rows")
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to list files",
			})
		}
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		log.Error().Err(err).Msg("Failed to commit file list transaction")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to list files",
		})
	}

	// Build response
	response := fiber.Map{
		"bucket":  bucket,
		"objects": objects,
		"count":   len(objects),
	}

	if delimiter != "" {
		response["prefix"] = prefix
		response["prefixes"] = prefixes
	}

	return c.JSON(response)
}

// CreateBucket handles bucket creation
// POST /api/v1/storage/buckets/:bucket
func (h *StorageHandler) CreateBucket(c *fiber.Ctx) error {
	bucket := c.Params("bucket")

	if bucket == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "bucket name is required",
		})
	}

	// Parse request body for bucket configuration
	var req struct {
		Public           bool     `json:"public"`
		AllowedMimeTypes []string `json:"allowed_mime_types"`
		MaxFileSize      *int64   `json:"max_file_size"`
	}
	// Try to parse body, but allow empty body (use defaults)
	_ = c.BodyParser(&req)

	// Start database transaction
	ctx := c.Context()
	tx, err := h.db.Pool().Begin(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to start transaction for bucket creation")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to create bucket",
		})
	}
	defer tx.Rollback(ctx)

	// Set RLS context
	if err := h.setRLSContext(ctx, tx, c); err != nil {
		log.Error().Err(err).Msg("Failed to set RLS context")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to create bucket",
		})
	}

	// Insert bucket into database (RLS will check permissions)
	_, err = tx.Exec(ctx, `
		INSERT INTO storage.buckets (id, name, public, allowed_mime_types, max_file_size)
		VALUES ($1, $2, $3, $4, $5)
	`, bucket, bucket, req.Public, req.AllowedMimeTypes, req.MaxFileSize)

	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "already exists") {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": "bucket already exists",
			})
		}
		if strings.Contains(err.Error(), "permission denied") || strings.Contains(err.Error(), "policy") {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "insufficient permissions to create bucket",
			})
		}
		log.Error().Err(err).Str("bucket", bucket).Msg("Failed to insert bucket into database")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to create bucket",
		})
	}

	// Create the bucket in storage provider
	if err := h.storage.Provider.CreateBucket(ctx, bucket); err != nil {
		// Rollback will happen via defer
		if strings.Contains(err.Error(), "already exists") {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": "bucket already exists in storage",
			})
		}
		log.Error().Err(err).Str("bucket", bucket).Msg("Failed to create bucket in provider")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to create bucket in storage provider",
		})
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		log.Error().Err(err).Str("bucket", bucket).Msg("Failed to commit bucket creation")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to create bucket",
		})
	}

	log.Info().
		Str("bucket", bucket).
		Bool("public", req.Public).
		Str("user_id", getUserID(c)).
		Msg("Bucket created")

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"bucket":             bucket,
		"id":                 bucket,
		"name":               bucket,
		"public":             req.Public,
		"allowed_mime_types": req.AllowedMimeTypes,
		"max_file_size":      req.MaxFileSize,
		"message":            "bucket created successfully",
	})
}

// UpdateBucketSettings handles updating bucket settings
// PUT /api/v1/storage/buckets/:bucket
func (h *StorageHandler) UpdateBucketSettings(c *fiber.Ctx) error {
	bucket := c.Params("bucket")

	if bucket == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "bucket name is required",
		})
	}

	// Parse request body
	var req struct {
		Public           *bool    `json:"public"`
		AllowedMimeTypes []string `json:"allowed_mime_types"`
		MaxFileSize      *int64   `json:"max_file_size"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	ctx := c.Context()

	// Start database transaction
	tx, err := h.db.Pool().Begin(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to start transaction for bucket update")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to update bucket",
		})
	}
	defer tx.Rollback(ctx)

	// Set RLS context
	if err := h.setRLSContext(ctx, tx, c); err != nil {
		log.Error().Err(err).Msg("Failed to set RLS context")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to update bucket",
		})
	}

	// Build dynamic UPDATE query based on provided fields
	updates := []string{}
	args := []interface{}{bucket}
	argCount := 1

	if req.Public != nil {
		argCount++
		updates = append(updates, fmt.Sprintf("public = $%d", argCount))
		args = append(args, *req.Public)
	}

	if req.AllowedMimeTypes != nil {
		argCount++
		updates = append(updates, fmt.Sprintf("allowed_mime_types = $%d", argCount))
		args = append(args, req.AllowedMimeTypes)
	}

	if req.MaxFileSize != nil {
		argCount++
		updates = append(updates, fmt.Sprintf("max_file_size = $%d", argCount))
		args = append(args, req.MaxFileSize)
	}

	if len(updates) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "no fields to update",
		})
	}

	updates = append(updates, "updated_at = NOW()")
	query := fmt.Sprintf(`
		UPDATE storage.buckets
		SET %s
		WHERE id = $1
	`, strings.Join(updates, ", "))

	// Execute update (RLS will check permissions - only admins can update buckets)
	result, err := tx.Exec(ctx, query, args...)
	if err != nil {
		if strings.Contains(err.Error(), "permission denied") || strings.Contains(err.Error(), "policy") {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "insufficient permissions to update bucket",
			})
		}
		log.Error().Err(err).Str("bucket", bucket).Msg("Failed to update bucket in database")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to update bucket",
		})
	}

	// Check if any rows were affected
	if result.RowsAffected() == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "bucket not found or insufficient permissions",
		})
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		log.Error().Err(err).Str("bucket", bucket).Msg("Failed to commit bucket update")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to update bucket",
		})
	}

	log.Info().
		Str("bucket", bucket).
		Str("user_id", getUserID(c)).
		Interface("updates", req).
		Msg("Bucket settings updated")

	return c.JSON(fiber.Map{
		"message": "bucket settings updated successfully",
	})
}

// DeleteBucket handles bucket deletion
// DELETE /api/v1/storage/buckets/:bucket
func (h *StorageHandler) DeleteBucket(c *fiber.Ctx) error {
	bucket := c.Params("bucket")

	if bucket == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "bucket name is required",
		})
	}

	// Delete the bucket
	if err := h.storage.Provider.DeleteBucket(c.Context(), bucket); err != nil {
		if strings.Contains(err.Error(), "not found") {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "bucket not found",
			})
		}
		if strings.Contains(err.Error(), "not empty") {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": "bucket is not empty",
			})
		}
		log.Error().Err(err).Str("bucket", bucket).Msg("Failed to delete bucket")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to delete bucket",
		})
	}

	log.Info().
		Str("bucket", bucket).
		Str("user_id", getUserID(c)).
		Msg("Bucket deleted")

	return c.Status(fiber.StatusNoContent).Send(nil)
}

// ListBuckets handles listing all buckets
// GET /api/v1/storage/buckets
func (h *StorageHandler) ListBuckets(c *fiber.Ctx) error {
	ctx := c.Context()

	// Start database transaction
	tx, err := h.db.Pool().Begin(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to start transaction for listing buckets")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to list buckets",
		})
	}
	defer tx.Rollback(ctx)

	// Set RLS context
	if err := h.setRLSContext(ctx, tx, c); err != nil {
		log.Error().Err(err).Msg("Failed to set RLS context")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to list buckets",
		})
	}

	// Query buckets from database (RLS will filter based on permissions)
	rows, err := tx.Query(ctx, `
		SELECT id, name, public, allowed_mime_types, max_file_size, created_at, updated_at
		FROM storage.buckets
		ORDER BY created_at DESC
	`)
	if err != nil {
		log.Error().Err(err).Msg("Failed to query buckets from database")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to list buckets",
		})
	}
	defer rows.Close()

	// Parse results
	type Bucket struct {
		ID               string    `json:"id"`
		Name             string    `json:"name"`
		Public           bool      `json:"public"`
		AllowedMimeTypes []string  `json:"allowed_mime_types"`
		MaxFileSize      *int64    `json:"max_file_size"`
		CreatedAt        time.Time `json:"created_at"`
		UpdatedAt        time.Time `json:"updated_at"`
	}

	var buckets []Bucket
	for rows.Next() {
		var b Bucket
		if err := rows.Scan(&b.ID, &b.Name, &b.Public, &b.AllowedMimeTypes, &b.MaxFileSize, &b.CreatedAt, &b.UpdatedAt); err != nil {
			log.Error().Err(err).Msg("Failed to scan bucket row")
			continue
		}
		buckets = append(buckets, b)
	}

	if err := rows.Err(); err != nil {
		log.Error().Err(err).Msg("Error iterating bucket rows")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to list buckets",
		})
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		log.Error().Err(err).Msg("Failed to commit bucket list transaction")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to list buckets",
		})
	}

	return c.JSON(fiber.Map{
		"buckets": buckets,
	})
}

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

// ShareObject handles sharing a file with another user
// POST /api/v1/storage/:bucket/:path/share
func (h *StorageHandler) ShareObject(c *fiber.Ctx) error {
	bucket := c.Params("bucket")
	key := c.Params("*")

	if bucket == "" || key == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "bucket and key are required",
		})
	}

	// Parse request body
	var req struct {
		UserID     string `json:"user_id"`
		Permission string `json:"permission"` // "read" or "write"
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	// Validate permission
	if req.Permission != "read" && req.Permission != "write" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "permission must be 'read' or 'write'",
		})
	}

	if req.UserID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "user_id is required",
		})
	}

	ctx := c.Context()

	// Start database transaction
	tx, err := h.db.Pool().Begin(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to start transaction for sharing file")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to share file",
		})
	}
	defer tx.Rollback(ctx)

	// Set RLS context
	if err := h.setRLSContext(ctx, tx, c); err != nil {
		log.Error().Err(err).Msg("Failed to set RLS context")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to share file",
		})
	}

	// Get object ID (also verifies user has access to this file)
	var objectID string
	err = tx.QueryRow(ctx, `
		SELECT id FROM storage.objects
		WHERE bucket_id = $1 AND path = $2
	`, bucket, key).Scan(&objectID)

	if err != nil {
		if err == pgx.ErrNoRows {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "file not found or insufficient permissions",
			})
		}
		log.Error().Err(err).Str("bucket", bucket).Str("key", key).Msg("Failed to find file")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to share file",
		})
	}

	// Insert or update permission (RLS will check if current user owns the file)
	_, err = tx.Exec(ctx, `
		INSERT INTO storage.object_permissions (object_id, user_id, permission)
		VALUES ($1, $2, $3)
		ON CONFLICT (object_id, user_id)
		DO UPDATE SET permission = $3
	`, objectID, req.UserID, req.Permission)

	if err != nil {
		if strings.Contains(err.Error(), "permission denied") || strings.Contains(err.Error(), "policy") {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "only file owners can share files",
			})
		}
		log.Error().Err(err).Str("object_id", objectID).Msg("Failed to create file permission")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to share file",
		})
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		log.Error().Err(err).Msg("Failed to commit share transaction")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to share file",
		})
	}

	log.Info().
		Str("bucket", bucket).
		Str("key", key).
		Str("shared_with", req.UserID).
		Str("permission", req.Permission).
		Str("user_id", getUserID(c)).
		Msg("File shared")

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message":    "file shared successfully",
		"user_id":    req.UserID,
		"permission": req.Permission,
	})
}

// RevokeShare handles revoking file access from a user
// DELETE /api/v1/storage/:bucket/:path/share/:user_id
func (h *StorageHandler) RevokeShare(c *fiber.Ctx) error {
	bucket := c.Params("bucket")
	key := c.Params("*1") // Everything before "/share"
	sharedUserID := c.Params("user_id")

	if bucket == "" || key == "" || sharedUserID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "bucket, key, and user_id are required",
		})
	}

	ctx := c.Context()

	// Start database transaction
	tx, err := h.db.Pool().Begin(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to start transaction for revoking share")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to revoke share",
		})
	}
	defer tx.Rollback(ctx)

	// Set RLS context
	if err := h.setRLSContext(ctx, tx, c); err != nil {
		log.Error().Err(err).Msg("Failed to set RLS context")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to revoke share",
		})
	}

	// Get object ID (also verifies user has access to this file)
	var objectID string
	err = tx.QueryRow(ctx, `
		SELECT id FROM storage.objects
		WHERE bucket_id = $1 AND path = $2
	`, bucket, key).Scan(&objectID)

	if err != nil {
		if err == pgx.ErrNoRows {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "file not found or insufficient permissions",
			})
		}
		log.Error().Err(err).Str("bucket", bucket).Str("key", key).Msg("Failed to find file")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to revoke share",
		})
	}

	// Delete permission (RLS will check if current user owns the file)
	result, err := tx.Exec(ctx, `
		DELETE FROM storage.object_permissions
		WHERE object_id = $1 AND user_id = $2
	`, objectID, sharedUserID)

	if err != nil {
		if strings.Contains(err.Error(), "permission denied") || strings.Contains(err.Error(), "policy") {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "only file owners can revoke shares",
			})
		}
		log.Error().Err(err).Str("object_id", objectID).Msg("Failed to delete file permission")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to revoke share",
		})
	}

	// Check if any rows were affected
	if result.RowsAffected() == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "share not found or insufficient permissions",
		})
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		log.Error().Err(err).Msg("Failed to commit revoke share transaction")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to revoke share",
		})
	}

	log.Info().
		Str("bucket", bucket).
		Str("key", key).
		Str("revoked_from", sharedUserID).
		Str("user_id", getUserID(c)).
		Msg("File share revoked")

	return c.Status(fiber.StatusNoContent).Send(nil)
}

// ListShares handles listing users a file is shared with
// GET /api/v1/storage/:bucket/:path/shares
func (h *StorageHandler) ListShares(c *fiber.Ctx) error {
	bucket := c.Params("bucket")
	key := c.Params("*")

	if bucket == "" || key == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "bucket and key are required",
		})
	}

	ctx := c.Context()

	// Start database transaction
	tx, err := h.db.Pool().Begin(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to start transaction for listing shares")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to list shares",
		})
	}
	defer tx.Rollback(ctx)

	// Set RLS context
	if err := h.setRLSContext(ctx, tx, c); err != nil {
		log.Error().Err(err).Msg("Failed to set RLS context")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to list shares",
		})
	}

	// Get object ID and verify access
	var objectID string
	err = tx.QueryRow(ctx, `
		SELECT id FROM storage.objects
		WHERE bucket_id = $1 AND path = $2
	`, bucket, key).Scan(&objectID)

	if err != nil {
		if err == pgx.ErrNoRows {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "file not found or insufficient permissions",
			})
		}
		log.Error().Err(err).Str("bucket", bucket).Str("key", key).Msg("Failed to find file")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to list shares",
		})
	}

	// Query shares (RLS will filter - only owners can see shares)
	rows, err := tx.Query(ctx, `
		SELECT user_id, permission, created_at
		FROM storage.object_permissions
		WHERE object_id = $1
		ORDER BY created_at DESC
	`, objectID)
	if err != nil {
		log.Error().Err(err).Str("object_id", objectID).Msg("Failed to query shares")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to list shares",
		})
	}
	defer rows.Close()

	type Share struct {
		UserID     string    `json:"user_id"`
		Permission string    `json:"permission"`
		CreatedAt  time.Time `json:"created_at"`
	}

	var shares []Share
	for rows.Next() {
		var share Share
		if err := rows.Scan(&share.UserID, &share.Permission, &share.CreatedAt); err != nil {
			log.Error().Err(err).Msg("Failed to scan share row")
			continue
		}
		shares = append(shares, share)
	}

	if err := rows.Err(); err != nil {
		log.Error().Err(err).Msg("Error iterating share rows")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to list shares",
		})
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		log.Error().Err(err).Msg("Failed to commit list shares transaction")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to list shares",
		})
	}

	return c.JSON(fiber.Map{
		"shares": shares,
		"count":  len(shares),
	})
}

// Helper functions

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

func parseMetadata(c *fiber.Ctx) map[string]string {
	metadata := make(map[string]string)

	// Parse metadata from form fields starting with "metadata_"
	c.Request().PostArgs().VisitAll(func(key, value []byte) {
		keyStr := string(key)
		if strings.HasPrefix(keyStr, "metadata_") {
			metaKey := strings.TrimPrefix(keyStr, "metadata_")
			metadata[metaKey] = string(value)
		}
	})

	return metadata
}

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

// MultipartUpload handles multipart upload
// POST /api/v1/storage/:bucket/multipart
func (h *StorageHandler) MultipartUpload(c *fiber.Ctx) error {
	bucket := c.Params("bucket")

	if bucket == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "bucket is required",
		})
	}

	// Parse multipart form
	form, err := c.MultipartForm()
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "failed to parse multipart form",
		})
	}

	files := form.File["files"]
	if len(files) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "no files provided",
		})
	}

	var uploaded []storage.Object
	var errors []string

	// Upload each file
	for _, file := range files {
		key := file.Filename

		// Validate file size
		if err := h.storage.ValidateUploadSize(file.Size); err != nil {
			errors = append(errors, fmt.Sprintf("%s: %s", file.Filename, err.Error()))
			continue
		}

		// Upload file
		if err := uploadMultipartFile(c, h.storage, bucket, key, file); err != nil {
			errors = append(errors, fmt.Sprintf("%s: %s", file.Filename, err.Error()))
			continue
		}

		uploaded = append(uploaded, storage.Object{
			Key:    key,
			Bucket: bucket,
			Size:   file.Size,
		})
	}

	response := fiber.Map{
		"uploaded": uploaded,
		"count":    len(uploaded),
	}

	if len(errors) > 0 {
		response["errors"] = errors
	}

	return c.Status(fiber.StatusCreated).JSON(response)
}

func uploadMultipartFile(c *fiber.Ctx, svc *storage.Service, bucket, key string, file *multipart.FileHeader) error {
	src, err := file.Open()
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer src.Close()

	contentType := file.Header.Get("Content-Type")
	if contentType == "" {
		contentType = detectContentType(file.Filename)
	}

	opts := &storage.UploadOptions{
		ContentType: contentType,
	}

	_, err = svc.Provider.Upload(c.Context(), bucket, key, src, file.Size, opts)
	return err
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
