package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5"

	"github.com/nimbleflux/fluxbase/internal/storage"
)

// DB-backed chunked-upload session persistence for providers that don't keep
// sessions on disk (currently S3). LocalStorage persists session.json itself
// (see internal/storage/local_chunked.go); S3 has no local store, so its
// sessions live in the storage.chunked_upload_sessions table.
//
// All operations run under RLS using the request's authenticated user, so the
// storage_chunked_sessions_* policies (which require current_user_id() = owner_id
// for non-admins) enforce that a user can only read/write their own sessions.

// errChunkedSessionNotFound is returned when a session row is absent or not
// visible to the caller under RLS. Callers translate this to a 404.
var errChunkedSessionNotFound = errors.New("chunked upload session not found")

// createChunkedSessionDB inserts a new session row under RLS.
func (h *StorageHandler) createChunkedSessionDB(ctx context.Context, c fiber.Ctx, session *storage.ChunkedUploadSession) error {
	completedJSON, err := json.Marshal(session.CompletedChunks)
	if err != nil {
		return fmt.Errorf("failed to marshal completed_chunks: %w", err)
	}
	// map[int]string marshals to {"0":"etag",...} which round-trips correctly,
	// but on read keys come back as strings and must be converted (see below).
	etagsJSON, err := json.Marshal(session.S3PartETags)
	if err != nil {
		return fmt.Errorf("failed to marshal s3_part_etags: %w", err)
	}
	metadataJSON, err := json.Marshal(session.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	var ownerID any
	if session.OwnerID != "" && session.OwnerID != "anonymous" {
		ownerID = session.OwnerID
	}

	tx, err := h.getPool(c).Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := h.setRLSContext(ctx, tx, c); err != nil {
		return fmt.Errorf("failed to set RLS context: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO storage.chunked_upload_sessions
			(upload_id, bucket_id, path, total_size, chunk_size, total_chunks,
			 completed_chunks, content_type, metadata, cache_control, owner_id,
			 s3_upload_id, s3_part_etags, status, created_at, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
	`, session.UploadID, session.Bucket, session.Key, session.TotalSize, session.ChunkSize,
		session.TotalChunks, completedJSON, session.ContentType, metadataJSON, session.CacheControl,
		ownerID, session.S3UploadID, etagsJSON, session.Status, session.CreatedAt, session.ExpiresAt)
	if err != nil {
		return fmt.Errorf("failed to insert chunked upload session: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit chunked upload session: %w", err)
	}
	return nil
}

// getChunkedSessionDB loads a session row under RLS. Returns
// errChunkedSessionNotFound if the row is absent or not visible to the caller.
func (h *StorageHandler) getChunkedSessionDB(ctx context.Context, c fiber.Ctx, uploadID string) (*storage.ChunkedUploadSession, error) {
	tx, err := h.getPool(c).Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := h.setRLSContext(ctx, tx, c); err != nil {
		return nil, fmt.Errorf("failed to set RLS context: %w", err)
	}

	var (
		s             storage.ChunkedUploadSession
		completedJSON []byte
		etagsJSON     []byte
		metadataJSON  []byte
		cacheControl  sql.NullString
		contentType   sql.NullString
		s3UploadID    sql.NullString
		ownerID       sql.NullString
	)
	err = tx.QueryRow(ctx, `
		SELECT upload_id, bucket_id, path, total_size, chunk_size, total_chunks,
		       completed_chunks, content_type, metadata, cache_control, owner_id,
		       s3_upload_id, s3_part_etags, status, created_at, expires_at
		FROM storage.chunked_upload_sessions
		WHERE upload_id = $1
	`, uploadID).Scan(
		&s.UploadID, &s.Bucket, &s.Key, &s.TotalSize, &s.ChunkSize, &s.TotalChunks,
		&completedJSON, &contentType, &metadataJSON, &cacheControl, &ownerID,
		&s3UploadID, &etagsJSON, &s.Status, &s.CreatedAt, &s.ExpiresAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errChunkedSessionNotFound
		}
		return nil, fmt.Errorf("failed to query chunked upload session: %w", err)
	}

	s.ContentType = contentType.String
	s.CacheControl = cacheControl.String
	s.S3UploadID = s3UploadID.String
	s.OwnerID = ownerID.String

	if len(completedJSON) > 0 {
		if err := json.Unmarshal(completedJSON, &s.CompletedChunks); err != nil {
			return nil, fmt.Errorf("failed to unmarshal completed_chunks: %w", err)
		}
	}
	if len(etagsJSON) > 0 && string(etagsJSON) != "null" {
		// S3PartETags is map[int]string; JSON keys are strings, so decode into a
		// string-keyed map first and convert.
		var strKeys map[string]string
		if err := json.Unmarshal(etagsJSON, &strKeys); err != nil {
			return nil, fmt.Errorf("failed to unmarshal s3_part_etags: %w", err)
		}
		s.S3PartETags = make(map[int]string, len(strKeys))
		for k, v := range strKeys {
			var idx int
			if _, err := fmt.Sscanf(k, "%d", &idx); err != nil {
				return nil, fmt.Errorf("failed to parse part index %q: %w", k, err)
			}
			s.S3PartETags[idx] = v
		}
	}
	if len(metadataJSON) > 0 && string(metadataJSON) != "null" {
		if err := json.Unmarshal(metadataJSON, &s.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return &s, nil
}

// updateChunkedSessionDB upserts a session row under RLS.
func (h *StorageHandler) updateChunkedSessionDB(ctx context.Context, c fiber.Ctx, session *storage.ChunkedUploadSession) error {
	completedJSON, err := json.Marshal(session.CompletedChunks)
	if err != nil {
		return fmt.Errorf("failed to marshal completed_chunks: %w", err)
	}
	etagsJSON, err := json.Marshal(session.S3PartETags)
	if err != nil {
		return fmt.Errorf("failed to marshal s3_part_etags: %w", err)
	}

	tx, err := h.getPool(c).Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := h.setRLSContext(ctx, tx, c); err != nil {
		return fmt.Errorf("failed to set RLS context: %w", err)
	}

	_, err = tx.Exec(ctx, `
		UPDATE storage.chunked_upload_sessions
		SET completed_chunks = $1,
		    s3_part_etags = $2,
		    status = $3,
		    updated_at = NOW()
		WHERE upload_id = $4
	`, completedJSON, etagsJSON, session.Status, session.UploadID)
	if err != nil {
		return fmt.Errorf("failed to update chunked upload session: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit chunked upload session: %w", err)
	}
	return nil
}

// deleteChunkedSessionDB removes a session row under RLS.
func (h *StorageHandler) deleteChunkedSessionDB(ctx context.Context, c fiber.Ctx, uploadID string) error {
	tx, err := h.getPool(c).Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := h.setRLSContext(ctx, tx, c); err != nil {
		return fmt.Errorf("failed to set RLS context: %w", err)
	}

	_, err = tx.Exec(ctx, `DELETE FROM storage.chunked_upload_sessions WHERE upload_id = $1`, uploadID)
	if err != nil {
		return fmt.Errorf("failed to delete chunked upload session: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit chunked upload session delete: %w", err)
	}
	return nil
}

// isS3Provider reports whether the given provider is S3-backed (and thus needs
// DB-backed chunked sessions).
func isS3Provider(p storage.Provider) bool {
	_, ok := p.(*storage.S3Storage)
	return ok
}
