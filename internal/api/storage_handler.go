package api

import (
	"fmt"
	"mime/multipart"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
	"github.com/wayli-app/fluxbase/internal/storage"
)

// StorageHandler handles file storage operations
type StorageHandler struct {
	storage *storage.Service
}

// NewStorageHandler creates a new storage handler
func NewStorageHandler(storage *storage.Service) *StorageHandler {
	return &StorageHandler{
		storage: storage,
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

	// Upload the file
	object, err := h.storage.Provider.Upload(c.Context(), bucket, key, src, file.Size, opts)
	if err != nil {
		log.Error().Err(err).Str("bucket", bucket).Str("key", key).Msg("Failed to upload file")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to upload file",
		})
	}

	log.Info().
		Str("bucket", bucket).
		Str("key", key).
		Int64("size", object.Size).
		Str("user_id", getUserID(c)).
		Msg("File uploaded")

	return c.Status(fiber.StatusCreated).JSON(object)
}

// DownloadFile handles file download
// GET /api/v1/storageage/:bucket/:key
func (h *StorageHandler) DownloadFile(c *fiber.Ctx) error {
	bucket := c.Params("bucket")
	key := c.Params("*")

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

	// Parse download options
	opts := &storage.DownloadOptions{}

	// Support range requests
	if rangeHeader := c.Get("Range"); rangeHeader != "" {
		opts.Range = rangeHeader
	}

	// Download the file
	reader, object, err := h.storage.Provider.Download(c.Context(), bucket, key, opts)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "file not found",
			})
		}
		log.Error().Err(err).Str("bucket", bucket).Str("key", key).Msg("Failed to download file")
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

	// Set content disposition (for downloads)
	if c.Query("download") == "true" {
		filename := filepath.Base(key)
		c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	}

	// Stream the file (SendStream will close the reader)
	return c.SendStream(reader)
}

// DeleteFile handles file deletion
// DELETE /api/v1/storageage/:bucket/:key
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

	// Delete the file
	if err := h.storage.Provider.Delete(c.Context(), bucket, key); err != nil {
		if strings.Contains(err.Error(), "not found") {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "file not found",
			})
		}
		log.Error().Err(err).Str("bucket", bucket).Str("key", key).Msg("Failed to delete file")
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
// HEAD /api/v1/storageage/:bucket/:key
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

	// Get object metadata
	object, err := h.storage.Provider.GetObject(c.Context(), bucket, key)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "file not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get file info",
		})
	}

	return c.JSON(object)
}

// ListFiles handles listing files in a bucket
// GET /api/v1/storageage/:bucket
func (h *StorageHandler) ListFiles(c *fiber.Ctx) error {
	bucket := c.Params("bucket")

	if bucket == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "bucket is required",
		})
	}

	// Parse query parameters
	opts := &storage.ListOptions{
		Prefix:    c.Query("prefix", ""),
		Delimiter: c.Query("delimiter", ""),
		MaxKeys:   c.QueryInt("limit", 1000),
	}

	// List objects
	result, err := h.storage.Provider.List(c.Context(), bucket, opts)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "bucket not found",
			})
		}
		log.Error().Err(err).Str("bucket", bucket).Msg("Failed to list files")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to list files",
		})
	}

	return c.JSON(fiber.Map{
		"bucket":    bucket,
		"objects":   result.Objects,
		"prefixes":  result.CommonPrefixes,
		"truncated": result.IsTruncated,
	})
}

// CreateBucket handles bucket creation
// POST /api/v1/storageage/buckets/:bucket
func (h *StorageHandler) CreateBucket(c *fiber.Ctx) error {
	bucket := c.Params("bucket")

	if bucket == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "bucket name is required",
		})
	}

	// Create the bucket
	if err := h.storage.Provider.CreateBucket(c.Context(), bucket); err != nil {
		if strings.Contains(err.Error(), "already exists") {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": "bucket already exists",
			})
		}
		log.Error().Err(err).Str("bucket", bucket).Msg("Failed to create bucket")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to create bucket",
		})
	}

	log.Info().
		Str("bucket", bucket).
		Str("user_id", getUserID(c)).
		Msg("Bucket created")

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"bucket":  bucket,
		"message": "bucket created successfully",
	})
}

// DeleteBucket handles bucket deletion
// DELETE /api/v1/storageage/buckets/:bucket
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
// GET /api/v1/storageage/buckets
func (h *StorageHandler) ListBuckets(c *fiber.Ctx) error {
	// List all buckets
	buckets, err := h.storage.Provider.ListBuckets(c.Context())
	if err != nil {
		log.Error().Err(err).Msg("Failed to list buckets")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to list buckets",
		})
	}

	return c.JSON(fiber.Map{
		"buckets": buckets,
	})
}

// GenerateSignedURL generates a presigned URL for temporary access
// POST /api/v1/storageage/:bucket/:key/signed-url
func (h *StorageHandler) GenerateSignedURL(c *fiber.Ctx) error {
	// Check if provider supports signed URLs
	if h.storage.IsLocal() {
		return c.Status(fiber.StatusNotImplemented).JSON(fiber.Map{
			"error": "signed URLs not supported for local storage",
		})
	}

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
		"url":        url,
		"expires_in": req.ExpiresIn,
		"method":     req.Method,
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

// MultipartUpload handles multipart upload
// POST /api/v1/storageage/:bucket/multipart
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
