package api

import (
	"github.com/fluxbase-eu/fluxbase/internal/database"
	"github.com/fluxbase-eu/fluxbase/internal/storage"
)

// StorageHandler handles file storage operations
// Methods are split across multiple files:
// - storage_files.go: UploadFile, DownloadFile, DeleteFile, GetFileInfo, ListFiles
// - storage_buckets.go: CreateBucket, UpdateBucketSettings, DeleteBucket, ListBuckets
// - storage_signed.go: GenerateSignedURL, DownloadSignedObject
// - storage_multipart.go: MultipartUpload
// - storage_sharing.go: ShareObject, RevokeShare, ListShares
// - storage_utils.go: helper functions (detectContentType, parseMetadata, getUserID, setRLSContext)
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
