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
		if err := uploadMultipartFile(c, svc, bucket, key, file, contentType); err != nil {
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

// uploadMultipartFile uploads a single file from multipart form
func uploadMultipartFile(c fiber.Ctx, svc *storage.Service, bucket, key string, file *multipart.FileHeader, contentType string) error {
	src, err := file.Open()
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer func() { _ = src.Close() }()

	opts := &storage.UploadOptions{
		ContentType: contentType,
	}

	_, err = svc.Provider.Upload(c.RequestCtx(), bucket, key, src, file.Size, opts)
	return err
}

// fiber:context-methods migrated
