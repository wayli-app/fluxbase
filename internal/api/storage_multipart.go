package api

import (
	"fmt"
	"mime/multipart"

	"github.com/fluxbase-eu/fluxbase/internal/storage"
	"github.com/gofiber/fiber/v2"
)

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

// uploadMultipartFile uploads a single file from multipart form
func uploadMultipartFile(c *fiber.Ctx, svc *storage.Service, bucket, key string, file *multipart.FileHeader) error {
	src, err := file.Open()
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer func() { _ = src.Close() }()

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
