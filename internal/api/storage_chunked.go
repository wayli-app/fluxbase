package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/fluxbase-eu/fluxbase/internal/storage"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
)

// InitChunkedUploadRequest represents the request body for initializing a chunked upload
type InitChunkedUploadRequest struct {
	Path         string            `json:"path"`
	TotalSize    int64             `json:"total_size"`
	ChunkSize    int64             `json:"chunk_size,omitempty"`
	ContentType  string            `json:"content_type,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	CacheControl string            `json:"cache_control,omitempty"`
}

// ChunkedUploadSessionResponse represents the response for a chunked upload session
type ChunkedUploadSessionResponse struct {
	SessionID       string    `json:"session_id"`
	Bucket          string    `json:"bucket"`
	Path            string    `json:"path"`
	TotalSize       int64     `json:"total_size"`
	ChunkSize       int64     `json:"chunk_size"`
	TotalChunks     int       `json:"total_chunks"`
	CompletedChunks []int     `json:"completed_chunks"`
	Status          string    `json:"status"`
	ExpiresAt       time.Time `json:"expires_at"`
	CreatedAt       time.Time `json:"created_at"`
}

// UploadChunkResponse represents the response after uploading a chunk
type UploadChunkResponse struct {
	ChunkIndex int                          `json:"chunk_index"`
	ETag       string                       `json:"etag,omitempty"`
	Size       int64                        `json:"size"`
	Session    ChunkedUploadSessionResponse `json:"session"`
}

// CompleteChunkedUploadResponse represents the response after completing a chunked upload
type CompleteChunkedUploadResponse struct {
	ID          string `json:"id"`
	Path        string `json:"path"`
	FullPath    string `json:"full_path"`
	Size        int64  `json:"size"`
	ContentType string `json:"content_type,omitempty"`
}

// InitChunkedUpload initializes a new chunked upload session
// POST /api/v1/storage/:bucket/chunked/init
func (h *StorageHandler) InitChunkedUpload(c *fiber.Ctx) error {
	bucket := c.Params("bucket")
	if bucket == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "bucket is required",
		})
	}

	// Parse request body
	var req InitChunkedUploadRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body: " + err.Error(),
		})
	}

	if req.Path == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "path is required",
		})
	}

	if req.TotalSize <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "total_size must be greater than 0",
		})
	}

	// Default chunk size to 5MB if not specified
	chunkSize := req.ChunkSize
	if chunkSize <= 0 {
		chunkSize = 5 * 1024 * 1024 // 5MB
	}

	// Minimum chunk size is 5MB (S3 requirement for multipart upload, except last part)
	if chunkSize < 5*1024*1024 && req.TotalSize > chunkSize {
		chunkSize = 5 * 1024 * 1024
	}

	// Validate total size
	if err := h.storage.ValidateUploadSize(req.TotalSize); err != nil {
		return c.Status(fiber.StatusRequestEntityTooLarge).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Get owner ID from authenticated user
	ownerID := getUserID(c)

	// Prepare upload options
	opts := &storage.UploadOptions{
		ContentType:  req.ContentType,
		Metadata:     req.Metadata,
		CacheControl: req.CacheControl,
	}

	ctx := c.Context()

	// Initialize chunked upload with the storage provider
	var session *storage.ChunkedUploadSession
	var err error

	// Check provider type and call appropriate method
	switch provider := h.storage.Provider.(type) {
	case *storage.LocalStorage:
		session, err = provider.InitChunkedUpload(ctx, bucket, req.Path, req.TotalSize, chunkSize, opts)
	case *storage.S3Storage:
		session, err = provider.InitChunkedUpload(ctx, bucket, req.Path, req.TotalSize, chunkSize, opts)
	default:
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "storage provider does not support chunked uploads",
		})
	}

	if err != nil {
		log.Error().Err(err).Str("bucket", bucket).Str("path", req.Path).Msg("Failed to initialize chunked upload")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to initialize chunked upload: " + err.Error(),
		})
	}

	session.OwnerID = ownerID

	// Store session in database for persistence
	if err := h.storeChunkedUploadSession(ctx, session); err != nil {
		log.Warn().Err(err).Str("uploadID", session.UploadID).Msg("Failed to store session in database, session will be ephemeral")
	}

	log.Info().
		Str("uploadID", session.UploadID).
		Str("bucket", bucket).
		Str("path", req.Path).
		Int64("totalSize", req.TotalSize).
		Int("totalChunks", session.TotalChunks).
		Msg("Chunked upload session initialized")

	return c.Status(fiber.StatusCreated).JSON(ChunkedUploadSessionResponse{
		SessionID:       session.UploadID,
		Bucket:          session.Bucket,
		Path:            session.Key,
		TotalSize:       session.TotalSize,
		ChunkSize:       session.ChunkSize,
		TotalChunks:     session.TotalChunks,
		CompletedChunks: session.CompletedChunks,
		Status:          session.Status,
		ExpiresAt:       session.ExpiresAt,
		CreatedAt:       session.CreatedAt,
	})
}

// UploadChunk uploads a single chunk of a file
// PUT /api/v1/storage/:bucket/chunked/:uploadId/:chunkIndex
func (h *StorageHandler) UploadChunk(c *fiber.Ctx) error {
	bucket := c.Params("bucket")
	uploadID := c.Params("uploadId")
	chunkIndexStr := c.Params("chunkIndex")

	if bucket == "" || uploadID == "" || chunkIndexStr == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "bucket, uploadId, and chunkIndex are required",
		})
	}

	chunkIndex, err := strconv.Atoi(chunkIndexStr)
	if err != nil || chunkIndex < 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid chunkIndex: must be a non-negative integer",
		})
	}

	// Get chunk size from Content-Length header
	size := int64(c.Request().Header.ContentLength())
	if size <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Content-Length header is required",
		})
	}

	ctx := c.Context()

	// Retrieve session
	session, err := h.getChunkedUploadSession(ctx, uploadID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "upload session not found: " + err.Error(),
		})
	}

	// Verify bucket matches
	if session.Bucket != bucket {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "bucket mismatch",
		})
	}

	// Check if session is still active
	if session.Status != "active" {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": fmt.Sprintf("upload session is not active (status: %s)", session.Status),
		})
	}

	// Check expiration
	if time.Now().After(session.ExpiresAt) {
		return c.Status(fiber.StatusGone).JSON(fiber.Map{
			"error": "upload session has expired",
		})
	}

	// Get the request body as a stream reader
	// Try streaming first, fall back to buffered body
	var body io.Reader
	body = c.Request().BodyStream()
	if body == nil {
		// BodyStream can be nil if the body was buffered as bytes
		// Fall back to reading the buffered body
		bodyBytes := c.Body()
		if len(bodyBytes) == 0 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "request body is required",
			})
		}
		body = bytes.NewReader(bodyBytes)
	}

	// Upload the chunk
	var result *storage.ChunkResult

	switch provider := h.storage.Provider.(type) {
	case *storage.LocalStorage:
		result, err = provider.UploadChunk(ctx, session, chunkIndex, body, size)
	case *storage.S3Storage:
		result, err = provider.UploadChunk(ctx, session, chunkIndex, body, size)
	default:
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "storage provider does not support chunked uploads",
		})
	}

	if err != nil {
		log.Error().Err(err).Str("uploadID", uploadID).Int("chunkIndex", chunkIndex).Msg("Failed to upload chunk")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to upload chunk: " + err.Error(),
		})
	}

	// Update session with the completed chunk
	session.CompletedChunks = append(session.CompletedChunks, chunkIndex)
	if session.S3PartETags == nil {
		session.S3PartETags = make(map[int]string)
	}
	session.S3PartETags[chunkIndex] = result.ETag

	// Store updated session
	if err := h.updateChunkedUploadSession(ctx, session); err != nil {
		log.Warn().Err(err).Str("uploadID", uploadID).Msg("Failed to update session in database")
	}

	log.Debug().
		Str("uploadID", uploadID).
		Int("chunkIndex", chunkIndex).
		Int64("size", result.Size).
		Msg("Chunk uploaded")

	return c.Status(fiber.StatusOK).JSON(UploadChunkResponse{
		ChunkIndex: result.ChunkIndex,
		ETag:       result.ETag,
		Size:       result.Size,
		Session: ChunkedUploadSessionResponse{
			SessionID:       session.UploadID,
			Bucket:          session.Bucket,
			Path:            session.Key,
			TotalSize:       session.TotalSize,
			ChunkSize:       session.ChunkSize,
			TotalChunks:     session.TotalChunks,
			CompletedChunks: session.CompletedChunks,
			Status:          session.Status,
			ExpiresAt:       session.ExpiresAt,
			CreatedAt:       session.CreatedAt,
		},
	})
}

// CompleteChunkedUpload finalizes a chunked upload
// POST /api/v1/storage/:bucket/chunked/:uploadId/complete
func (h *StorageHandler) CompleteChunkedUpload(c *fiber.Ctx) error {
	bucket := c.Params("bucket")
	uploadID := c.Params("uploadId")

	if bucket == "" || uploadID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "bucket and uploadId are required",
		})
	}

	ctx := c.Context()

	// Retrieve session
	session, err := h.getChunkedUploadSession(ctx, uploadID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "upload session not found: " + err.Error(),
		})
	}

	// Verify bucket matches
	if session.Bucket != bucket {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "bucket mismatch",
		})
	}

	// Check if all chunks are uploaded
	if len(session.CompletedChunks) != session.TotalChunks {
		missingChunks := []int{}
		completedMap := make(map[int]bool)
		for _, idx := range session.CompletedChunks {
			completedMap[idx] = true
		}
		for i := 0; i < session.TotalChunks; i++ {
			if !completedMap[i] {
				missingChunks = append(missingChunks, i)
			}
		}
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":          "not all chunks have been uploaded",
			"missing_chunks": missingChunks,
			"uploaded":       len(session.CompletedChunks),
			"total":          session.TotalChunks,
		})
	}

	// Mark session as completing
	session.Status = "completing"
	_ = h.updateChunkedUploadSession(ctx, session)

	// Complete the upload
	var object *storage.Object

	switch provider := h.storage.Provider.(type) {
	case *storage.LocalStorage:
		object, err = provider.CompleteChunkedUpload(ctx, session)
	case *storage.S3Storage:
		object, err = provider.CompleteChunkedUpload(ctx, session)
	default:
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "storage provider does not support chunked uploads",
		})
	}

	if err != nil {
		session.Status = "active" // Revert status on failure
		_ = h.updateChunkedUploadSession(ctx, session)
		log.Error().Err(err).Str("uploadID", uploadID).Msg("Failed to complete chunked upload")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to complete chunked upload: " + err.Error(),
		})
	}

	// Store object record in database
	if err := h.storeUploadedObject(c, session, object); err != nil {
		log.Warn().Err(err).Str("uploadID", uploadID).Msg("Failed to store object in database")
	}

	// Mark session as completed and clean up
	session.Status = "completed"
	_ = h.deleteChunkedUploadSession(ctx, uploadID)

	log.Info().
		Str("uploadID", uploadID).
		Str("bucket", bucket).
		Str("path", session.Key).
		Int64("size", object.Size).
		Msg("Chunked upload completed")

	return c.Status(fiber.StatusOK).JSON(CompleteChunkedUploadResponse{
		ID:          object.ETag,
		Path:        object.Key,
		FullPath:    fmt.Sprintf("%s/%s", object.Bucket, object.Key),
		Size:        object.Size,
		ContentType: object.ContentType,
	})
}

// GetChunkedUploadStatus retrieves the status of a chunked upload session
// GET /api/v1/storage/:bucket/chunked/:uploadId/status
func (h *StorageHandler) GetChunkedUploadStatus(c *fiber.Ctx) error {
	bucket := c.Params("bucket")
	uploadID := c.Params("uploadId")

	if bucket == "" || uploadID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "bucket and uploadId are required",
		})
	}

	ctx := c.Context()

	// Retrieve session
	session, err := h.getChunkedUploadSession(ctx, uploadID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "upload session not found: " + err.Error(),
		})
	}

	// Verify bucket matches
	if session.Bucket != bucket {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "bucket mismatch",
		})
	}

	// Calculate missing chunks
	missingChunks := []int{}
	completedMap := make(map[int]bool)
	for _, idx := range session.CompletedChunks {
		completedMap[idx] = true
	}
	for i := 0; i < session.TotalChunks; i++ {
		if !completedMap[i] {
			missingChunks = append(missingChunks, i)
		}
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"session": ChunkedUploadSessionResponse{
			SessionID:       session.UploadID,
			Bucket:          session.Bucket,
			Path:            session.Key,
			TotalSize:       session.TotalSize,
			ChunkSize:       session.ChunkSize,
			TotalChunks:     session.TotalChunks,
			CompletedChunks: session.CompletedChunks,
			Status:          session.Status,
			ExpiresAt:       session.ExpiresAt,
			CreatedAt:       session.CreatedAt,
		},
		"missing_chunks": missingChunks,
	})
}

// AbortChunkedUpload aborts a chunked upload and cleans up
// DELETE /api/v1/storage/:bucket/chunked/:uploadId
func (h *StorageHandler) AbortChunkedUpload(c *fiber.Ctx) error {
	bucket := c.Params("bucket")
	uploadID := c.Params("uploadId")

	if bucket == "" || uploadID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "bucket and uploadId are required",
		})
	}

	ctx := c.Context()

	// Retrieve session
	session, err := h.getChunkedUploadSession(ctx, uploadID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "upload session not found: " + err.Error(),
		})
	}

	// Verify bucket matches
	if session.Bucket != bucket {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "bucket mismatch",
		})
	}

	// Abort the upload
	switch provider := h.storage.Provider.(type) {
	case *storage.LocalStorage:
		err = provider.AbortChunkedUpload(ctx, session)
	case *storage.S3Storage:
		err = provider.AbortChunkedUpload(ctx, session)
	default:
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "storage provider does not support chunked uploads",
		})
	}

	if err != nil {
		log.Error().Err(err).Str("uploadID", uploadID).Msg("Failed to abort chunked upload")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to abort chunked upload: " + err.Error(),
		})
	}

	// Delete session from database
	_ = h.deleteChunkedUploadSession(ctx, uploadID)

	log.Info().
		Str("uploadID", uploadID).
		Str("bucket", bucket).
		Msg("Chunked upload aborted")

	return c.SendStatus(fiber.StatusNoContent)
}

// Helper functions for session management

func (h *StorageHandler) storeChunkedUploadSession(ctx interface{}, session *storage.ChunkedUploadSession) error {
	// For now, sessions are stored by the storage provider (local storage stores in files)
	// Database storage can be added later for cross-server session sharing
	return nil
}

func (h *StorageHandler) getChunkedUploadSession(ctx interface{}, uploadID string) (*storage.ChunkedUploadSession, error) {
	// Try to get session from storage provider
	switch provider := h.storage.Provider.(type) {
	case *storage.LocalStorage:
		return provider.GetChunkedUploadSession(uploadID)
	case *storage.S3Storage:
		// S3 doesn't have local session storage, we need to query the database
		// For now, return an error - this would need database session storage
		return nil, fmt.Errorf("session not found (S3 sessions require database storage)")
	default:
		return nil, fmt.Errorf("storage provider does not support chunked upload sessions")
	}
}

func (h *StorageHandler) updateChunkedUploadSession(ctx interface{}, session *storage.ChunkedUploadSession) error {
	switch provider := h.storage.Provider.(type) {
	case *storage.LocalStorage:
		return provider.UpdateChunkedUploadSession(session)
	case *storage.S3Storage:
		// S3 sessions would be stored in database
		return nil
	default:
		return nil
	}
}

func (h *StorageHandler) deleteChunkedUploadSession(ctx interface{}, uploadID string) error {
	// Session cleanup is handled by the provider when completing/aborting
	return nil
}

func (h *StorageHandler) storeUploadedObject(fiberCtx interface{}, session *storage.ChunkedUploadSession, object *storage.Object) error {
	// Store the object record in the database
	// This mirrors the logic in storage_files.go for regular uploads

	// Get fiber context and database pool
	c, ok := fiberCtx.(*fiber.Ctx)
	if !ok {
		return fmt.Errorf("invalid context type")
	}

	db := h.db.Pool()

	// Insert object record into storage.objects table
	// Note: 'name' column is auto-generated from 'path', so we don't insert it directly
	query := `
		INSERT INTO storage.objects (bucket_id, path, size, mime_type, metadata, owner_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
		ON CONFLICT (bucket_id, path)
		DO UPDATE SET
			size = EXCLUDED.size,
			mime_type = EXCLUDED.mime_type,
			metadata = EXCLUDED.metadata,
			updated_at = NOW()
	`

	metadataJSON, _ := json.Marshal(object.Metadata)

	var ownerID interface{} = nil
	if session.OwnerID != "" && session.OwnerID != "anonymous" {
		ownerID = session.OwnerID
	}

	_, err := db.Exec(c.Context(), query,
		object.Bucket,
		object.Key,
		object.Size,
		object.ContentType,
		metadataJSON,
		ownerID,
	)

	return err
}
