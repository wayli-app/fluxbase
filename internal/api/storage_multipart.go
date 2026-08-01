package api

import (
	"errors"
	"fmt"
	"mime/multipart"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/storage"
)

// MultipartUpload handles multipart upload
// POST /api/v1/storage/:bucket/multipart
func (h *StorageHandler) MultipartUpload(c fiber.Ctx) error {
	// Get tenant-specific storage service
	svc, err := h.getService(c)
	if err != nil {
		return SendInternalError(c, "failed to get storage service")
	}

	bucket := c.Params("bucket")

	if bucket == "" {
		return SendBadRequest(c, "bucket is required", ErrCodeMissingField)
	}

	// H-19: Check if bucket exists before upload
	// Use SECURITY DEFINER function to bypass RLS when checking bucket existence
	var bucketExists bool
	err = h.db.Pool().QueryRow(
		c.RequestCtx(),
		`SELECT storage.bucket_exists($1::text, $2::uuid)`,
		bucket, getTenantIDArg(c),
	).Scan(&bucketExists)
	if err != nil {
		log.Error().Err(err).Str("bucket", bucket).Msg("Failed to check bucket existence")
		return SendInternalError(c, "failed to validate bucket")
	}
	if !bucketExists {
		return SendNotFound(c, fmt.Sprintf("bucket '%s' does not exist", bucket))
	}

	// C-3: Get bucket MIME type settings
	// Use SECURITY DEFINER function to bypass RLS when fetching bucket settings
	var bucketAllowedMimeTypes []string
	err = h.db.Pool().QueryRow(
		c.RequestCtx(),
		`SELECT allowed_mime_types FROM storage.get_bucket_settings($1::text, $2::uuid)`,
		bucket, getTenantIDArg(c),
	).Scan(&bucketAllowedMimeTypes)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		log.Error().Err(err).Str("bucket", bucket).Msg("Failed to get bucket settings")
		return SendInternalError(c, "failed to validate bucket settings")
	}

	// Parse multipart form
	form, err := c.MultipartForm()
	if err != nil {
		return SendBadRequest(c, "failed to parse multipart form", ErrCodeInvalidInput)
	}

	files := form.File["files"]
	if len(files) == 0 {
		return SendBadRequest(c, "no files provided", ErrCodeMissingField)
	}

	var uploaded []storage.Object
	var errors []string

	// Upload each file
	for _, file := range files {
		key := file.Filename

		// H-20: Sanitize filename
		key = sanitizeFilename(key)
		if key == "" {
			errors = append(errors, fmt.Sprintf("%s: invalid filename after sanitization", file.Filename))
			continue
		}

		// Validate file size
		if err := svc.ValidateUploadSize(file.Size); err != nil {
			errors = append(errors, fmt.Sprintf("%s: %s", file.Filename, err.Error()))
			continue
		}

		// C-3: Detect content type for MIME validation
		contentType := file.Header.Get("Content-Type")
		if contentType == "" {
			contentType = detectContentType(file.Filename)
		}

		// C-3: Validate MIME type against bucket-specific allowed types
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
				errors = append(errors, fmt.Sprintf("%s: file type %s is not allowed for this bucket", file.Filename, contentType))
				continue
			}
		}

		// Upload file
		if err := h.uploadMultipartFile(c, svc, bucket, key, file, contentType); err != nil {
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

// uploadMultipartFile uploads a single file from multipart form and records its
// metadata row in storage.objects. This mirrors StorageHandler.UploadFile in
// storage_files.go — without the metadata insert, the uploaded bytes would be
// invisible to the API (download/list/share/delete key off the objects row) and
// the file would have no owner_id, failing the storage_objects_insert RLS policy.
func (h *StorageHandler) uploadMultipartFile(c fiber.Ctx, svc *storage.Service, bucket, key string, file *multipart.FileHeader, contentType string) error {
	src, err := file.Open()
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer func() { _ = src.Close() }()

	opts := &storage.UploadOptions{
		ContentType: contentType,
	}

	ctx := c.RequestCtx()

	// Upload the file to the storage provider first
	object, err := svc.Provider.Upload(ctx, bucket, key, src, file.Size, opts)
	if err != nil {
		log.Error().Err(err).Str("bucket", bucket).Str("key", key).Msg("Failed to upload multipart file")
		return fmt.Errorf("failed to upload file")
	}

	// Get owner ID from authenticated user (same source as the regular upload path)
	ownerID := getUserID(c)
	var ownerUUID *string
	if ownerID != "" && ownerID != "anonymous" {
		ownerUUID = &ownerID
	}

	// Store object metadata in the database under RLS
	tx, err := h.getPool(c).Begin(ctx)
	if err != nil {
		_ = svc.Provider.Delete(ctx, bucket, key)
		log.Error().Err(err).Msg("Failed to start transaction for multipart file upload")
		return fmt.Errorf("failed to save file metadata")
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := h.setRLSContext(ctx, tx, c); err != nil {
		_ = svc.Provider.Delete(ctx, bucket, key)
		log.Error().Err(err).Msg("Failed to set RLS context for multipart file upload")
		return fmt.Errorf("failed to save file metadata")
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO storage.objects (bucket_id, path, mime_type, size, metadata, owner_id)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (bucket_id, path)
		DO UPDATE SET mime_type = $3, size = $4, owner_id = $6, updated_at = NOW()
	`, bucket, key, contentType, object.Size, nil, ownerUUID)
	if err != nil {
		_ = svc.Provider.Delete(ctx, bucket, key)
		errMsg := err.Error()
		log.Error().
			Err(err).
			Str("bucket", bucket).
			Str("key", key).
			Str("error_message", errMsg).
			Msg("Failed to insert multipart file metadata into database")
		if strings.Contains(errMsg, "permission denied") || strings.Contains(errMsg, "policy") {
			return fmt.Errorf("insufficient permissions to upload file")
		}
		return fmt.Errorf("failed to save file metadata")
	}

	if err := tx.Commit(ctx); err != nil {
		_ = svc.Provider.Delete(ctx, bucket, key)
		log.Error().Err(err).Str("bucket", bucket).Str("key", key).Msg("Failed to commit multipart file upload")
		return fmt.Errorf("failed to save file metadata")
	}

	log.Info().
		Str("bucket", bucket).
		Str("key", key).
		Int64("size", object.Size).
		Str("user_id", ownerID).
		Msg("Multipart file uploaded")

	return nil
}

// fiber:context-methods migrated
