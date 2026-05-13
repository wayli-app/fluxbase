package storage

import (
	"context"
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/rs/zerolog/log"
)

// limitedReadCloser wraps a Reader with a Closer
type limitedReadCloser struct {
	reader io.Reader
	closer io.Closer
}

func (l *limitedReadCloser) Read(p []byte) (n int, err error) {
	return l.reader.Read(p)
}

func (l *limitedReadCloser) Close() error {
	return l.closer.Close()
}

// uploadIDRegex validates that an upload ID is a 32-character hex string
var uploadIDRegex = regexp.MustCompile(`^[a-f0-9]{32}$`)

// getChunkedUploadDir returns the path to the chunked upload directory for a session
func (ls *LocalStorage) getChunkedUploadDir(uploadID string) (string, error) {
	if !uploadIDRegex.MatchString(uploadID) {
		return "", fmt.Errorf("invalid upload ID format")
	}
	return filepath.Join(ls.basePath, ".chunked", uploadID), nil
}

// getChunkPath returns the path to a specific chunk file
func (ls *LocalStorage) getChunkPath(uploadID string, chunkIndex int) (string, error) {
	dir, err := ls.getChunkedUploadDir(uploadID)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, fmt.Sprintf("chunk_%06d", chunkIndex)), nil
}

// InitChunkedUpload starts a new chunked upload session for local storage
func (ls *LocalStorage) InitChunkedUpload(ctx context.Context, bucket, key string, totalSize int64, chunkSize int64, opts *UploadOptions) (*ChunkedUploadSession, error) {
	// Validate bucket and key
	if _, err := ls.getPath(bucket, key); err != nil {
		return nil, fmt.Errorf("invalid path: %w", err)
	}

	// Generate cryptographically secure upload ID to prevent session hijacking
	randomBytes := make([]byte, 16)
	if _, err := rand.Read(randomBytes); err != nil {
		return nil, fmt.Errorf("failed to generate secure upload ID: %w", err)
	}
	uploadID := hex.EncodeToString(randomBytes)

	// Create chunked upload directory
	chunkDir, err := ls.getChunkedUploadDir(uploadID)
	if err != nil {
		return nil, fmt.Errorf("invalid upload ID: %w", err)
	}
	if err := os.MkdirAll(chunkDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create chunk directory: %w", err)
	}

	totalChunks := int((totalSize + chunkSize - 1) / chunkSize)

	session := &ChunkedUploadSession{
		UploadID:        uploadID,
		Bucket:          bucket,
		Key:             key,
		TotalSize:       totalSize,
		ChunkSize:       chunkSize,
		TotalChunks:     totalChunks,
		CompletedChunks: []int{},
		Status:          "active",
		CreatedAt:       time.Now(),
		ExpiresAt:       time.Now().Add(24 * time.Hour),
	}

	if opts != nil {
		session.ContentType = opts.ContentType
		session.Metadata = opts.Metadata
		session.CacheControl = opts.CacheControl
	}

	// Save session metadata to a file
	sessionPath := filepath.Join(chunkDir, "session.json")
	sessionData, err := json.Marshal(session)
	if err != nil {
		_ = os.RemoveAll(chunkDir)
		return nil, fmt.Errorf("failed to marshal session: %w", err)
	}
	if err := os.WriteFile(sessionPath, sessionData, 0o644); err != nil {
		_ = os.RemoveAll(chunkDir)
		return nil, fmt.Errorf("failed to save session: %w", err)
	}

	log.Debug().
		Str("uploadID", uploadID).
		Str("bucket", bucket).
		Str("key", key).
		Int64("totalSize", totalSize).
		Int("totalChunks", totalChunks).
		Msg("Chunked upload session initialized")

	return session, nil
}

// UploadChunk uploads a single chunk of data for local storage
func (ls *LocalStorage) UploadChunk(ctx context.Context, session *ChunkedUploadSession, chunkIndex int, data io.Reader, size int64) (*ChunkResult, error) {
	if session == nil {
		return nil, fmt.Errorf("session is nil")
	}

	if chunkIndex < 0 || chunkIndex >= session.TotalChunks {
		return nil, fmt.Errorf("invalid chunk index: %d (total chunks: %d)", chunkIndex, session.TotalChunks)
	}

	// Verify session directory exists
	chunkDir, err := ls.getChunkedUploadDir(session.UploadID)
	if err != nil {
		return nil, fmt.Errorf("invalid upload ID: %w", err)
	}
	if _, err := os.Stat(chunkDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("upload session not found")
	}

	// Create chunk file
	chunkPath, err := ls.getChunkPath(session.UploadID, chunkIndex)
	if err != nil {
		return nil, fmt.Errorf("invalid upload ID: %w", err)
	}
	file, err := os.Create(chunkPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create chunk file: %w", err)
	}
	defer func() { _ = file.Close() }()

	// Calculate MD5 hash while writing
	hash := md5.New()
	writer := io.MultiWriter(file, hash)

	// Copy data to chunk file
	written, err := io.Copy(writer, data)
	if err != nil {
		_ = os.Remove(chunkPath)
		return nil, fmt.Errorf("failed to write chunk: %w", err)
	}

	etag := hex.EncodeToString(hash.Sum(nil))

	log.Debug().
		Str("uploadID", session.UploadID).
		Int("chunkIndex", chunkIndex).
		Int64("size", written).
		Msg("Chunk uploaded")

	return &ChunkResult{
		ChunkIndex: chunkIndex,
		ETag:       etag,
		Size:       written,
	}, nil
}

// CompleteChunkedUpload finalizes the upload and assembles the file for local storage
func (ls *LocalStorage) CompleteChunkedUpload(ctx context.Context, session *ChunkedUploadSession) (*Object, error) {
	if session == nil {
		return nil, fmt.Errorf("session is nil")
	}

	chunkDir, err := ls.getChunkedUploadDir(session.UploadID)
	if err != nil {
		return nil, fmt.Errorf("invalid upload ID: %w", err)
	}

	// Verify all chunks exist
	for i := 0; i < session.TotalChunks; i++ {
		chunkPath, cpErr := ls.getChunkPath(session.UploadID, i)
		if cpErr != nil {
			return nil, fmt.Errorf("invalid upload ID: %w", cpErr)
		}
		if _, err := os.Stat(chunkPath); os.IsNotExist(err) {
			return nil, fmt.Errorf("missing chunk %d", i)
		}
	}

	// Get destination path
	destPath, err := ls.getPath(session.Bucket, session.Key)
	if err != nil {
		return nil, fmt.Errorf("invalid destination path: %w", err)
	}

	// Create parent directories
	destDir := filepath.Dir(destPath)
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Create destination file
	destFile, err := os.Create(destPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create destination file: %w", err)
	}
	defer func() { _ = destFile.Close() }()

	// Calculate MD5 hash while assembling
	hash := md5.New()
	writer := io.MultiWriter(destFile, hash)

	// Concatenate all chunks
	var totalWritten int64
	for i := 0; i < session.TotalChunks; i++ {
		chunkPath, cpErr := ls.getChunkPath(session.UploadID, i)
		if cpErr != nil {
			return nil, fmt.Errorf("invalid upload ID: %w", cpErr)
		}
		chunkFile, err := os.Open(chunkPath)
		if err != nil {
			_ = destFile.Close()
			_ = os.Remove(destPath)
			return nil, fmt.Errorf("failed to open chunk %d: %w", i, err)
		}

		written, err := io.Copy(writer, chunkFile)
		_ = chunkFile.Close()
		if err != nil {
			_ = destFile.Close()
			_ = os.Remove(destPath)
			return nil, fmt.Errorf("failed to copy chunk %d: %w", i, err)
		}
		totalWritten += written
	}

	etag := hex.EncodeToString(hash.Sum(nil))

	// Save metadata if present
	if len(session.Metadata) > 0 || session.ContentType != "" {
		metaPath := destPath + ".meta"
		metaData := ""
		for k, v := range session.Metadata {
			metaData += fmt.Sprintf("%s=%s\n", k, v)
		}
		if session.ContentType != "" {
			metaData += fmt.Sprintf("content-type=%s\n", session.ContentType)
		}
		_ = os.WriteFile(metaPath, []byte(metaData), 0o644)
	}

	// Clean up chunk directory
	if err := os.RemoveAll(chunkDir); err != nil {
		log.Warn().Err(err).Str("uploadID", session.UploadID).Msg("Failed to clean up chunk directory")
	}

	// Get final file info
	info, err := os.Stat(destPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat final file: %w", err)
	}

	log.Info().
		Str("uploadID", session.UploadID).
		Str("bucket", session.Bucket).
		Str("key", session.Key).
		Int64("size", totalWritten).
		Msg("Chunked upload completed")

	return &Object{
		Key:          session.Key,
		Bucket:       session.Bucket,
		Size:         info.Size(),
		ContentType:  session.ContentType,
		LastModified: info.ModTime(),
		ETag:         etag,
		Metadata:     session.Metadata,
	}, nil
}

// AbortChunkedUpload cancels the upload and cleans up chunks for local storage
func (ls *LocalStorage) AbortChunkedUpload(ctx context.Context, session *ChunkedUploadSession) error {
	if session == nil {
		return fmt.Errorf("session is nil")
	}

	chunkDir, err := ls.getChunkedUploadDir(session.UploadID)
	if err != nil {
		return fmt.Errorf("invalid upload ID: %w", err)
	}

	// Remove the entire chunk directory
	if err := os.RemoveAll(chunkDir); err != nil {
		return fmt.Errorf("failed to remove chunk directory: %w", err)
	}

	log.Info().
		Str("uploadID", session.UploadID).
		Msg("Chunked upload aborted")

	return nil
}

// GetChunkedUploadSession retrieves a chunked upload session from local storage
func (ls *LocalStorage) GetChunkedUploadSession(uploadID string) (*ChunkedUploadSession, error) {
	chunkDir, err := ls.getChunkedUploadDir(uploadID)
	if err != nil {
		return nil, fmt.Errorf("invalid upload ID: %w", err)
	}
	sessionPath := filepath.Join(chunkDir, "session.json")

	sessionData, err := os.ReadFile(sessionPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("upload session not found")
		}
		return nil, fmt.Errorf("failed to read session: %w", err)
	}

	var session ChunkedUploadSession
	if err := json.Unmarshal(sessionData, &session); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %w", err)
	}

	// Update completed chunks by checking which chunk files exist
	session.CompletedChunks = []int{}
	for i := 0; i < session.TotalChunks; i++ {
		chunkPath, cpErr := ls.getChunkPath(uploadID, i)
		if cpErr != nil {
			return nil, fmt.Errorf("invalid upload ID: %w", cpErr)
		}
		if _, err := os.Stat(chunkPath); err == nil {
			session.CompletedChunks = append(session.CompletedChunks, i)
		}
	}

	return &session, nil
}

// UpdateChunkedUploadSession updates a session file after chunk upload
func (ls *LocalStorage) UpdateChunkedUploadSession(session *ChunkedUploadSession) error {
	chunkDir, err := ls.getChunkedUploadDir(session.UploadID)
	if err != nil {
		return fmt.Errorf("invalid upload ID: %w", err)
	}
	sessionPath := filepath.Join(chunkDir, "session.json")

	sessionData, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	if err := os.WriteFile(sessionPath, sessionData, 0o644); err != nil {
		return fmt.Errorf("failed to save session: %w", err)
	}

	return nil
}

// CleanupExpiredChunkedUploads removes expired chunked upload sessions and their files
// This should be called periodically to prevent storage leaks
func (ls *LocalStorage) CleanupExpiredChunkedUploads(ctx context.Context) (int, error) {
	chunkedDir := filepath.Join(ls.basePath, ".chunked")

	// Check if chunked directory exists
	if _, err := os.Stat(chunkedDir); os.IsNotExist(err) {
		return 0, nil // No chunked uploads to clean
	}

	entries, err := os.ReadDir(chunkedDir)
	if err != nil {
		return 0, fmt.Errorf("failed to read chunked upload directory: %w", err)
	}

	cleaned := 0
	now := time.Now()

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Check context cancellation
		select {
		case <-ctx.Done():
			return cleaned, ctx.Err()
		default:
		}

		uploadID := entry.Name()
		sessionPath := filepath.Join(chunkedDir, uploadID, "session.json")

		sessionData, err := os.ReadFile(sessionPath)
		if err != nil {
			// If we can't read the session, check directory age
			// Remove directories older than 48 hours with no valid session
			info, statErr := entry.Info()
			if statErr == nil && now.Sub(info.ModTime()) > 48*time.Hour {
				if rmErr := os.RemoveAll(filepath.Join(chunkedDir, uploadID)); rmErr == nil {
					cleaned++
					log.Debug().Str("upload_id", uploadID).Msg("Removed orphaned chunked upload directory")
				}
			}
			continue
		}

		var session ChunkedUploadSession
		if err := json.Unmarshal(sessionData, &session); err != nil {
			// Invalid session, remove if old
			info, statErr := entry.Info()
			if statErr == nil && now.Sub(info.ModTime()) > 48*time.Hour {
				if rmErr := os.RemoveAll(filepath.Join(chunkedDir, uploadID)); rmErr == nil {
					cleaned++
					log.Debug().Str("upload_id", uploadID).Msg("Removed chunked upload with invalid session")
				}
			}
			continue
		}

		// Check if session is expired
		if now.After(session.ExpiresAt) {
			if err := os.RemoveAll(filepath.Join(chunkedDir, uploadID)); err == nil {
				cleaned++
				log.Debug().
					Str("upload_id", uploadID).
					Str("bucket", session.Bucket).
					Str("key", session.Key).
					Time("expired_at", session.ExpiresAt).
					Msg("Removed expired chunked upload session")
			} else {
				log.Warn().Err(err).Str("upload_id", uploadID).Msg("Failed to remove expired chunked upload")
			}
		}
	}

	if cleaned > 0 {
		log.Info().Int("cleaned", cleaned).Msg("Cleaned up expired chunked upload sessions")
	}

	return cleaned, nil
}

// StartChunkedUploadCleanup starts a background goroutine to periodically clean up
// expired chunked upload sessions. Call this once when initializing the storage.
func (ls *LocalStorage) StartChunkedUploadCleanup(ctx context.Context) {
	go func() {
		defer func() {
			if rec := recover(); rec != nil {
				log.Error().
					Interface("panic", rec).
					Str("goroutine", "local_chunked_upload_cleanup").
					Msg("Panic in local storage chunked upload cleanup - recovered")
			}
		}()

		// Run cleanup every hour
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()

		// Also run once on startup after a short delay
		time.Sleep(30 * time.Second)
		if _, err := ls.CleanupExpiredChunkedUploads(ctx); err != nil {
			log.Error().Err(err).Msg("Failed to cleanup expired chunked uploads on startup")
		}

		for {
			select {
			case <-ticker.C:
				if _, err := ls.CleanupExpiredChunkedUploads(ctx); err != nil {
					log.Error().Err(err).Msg("Failed to cleanup expired chunked uploads")
				}
			case <-ctx.Done():
				return
			}
		}
	}()
}
