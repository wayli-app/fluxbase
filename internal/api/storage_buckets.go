package api

import (
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
)

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
