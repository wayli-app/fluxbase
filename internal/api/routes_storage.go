package api

import (
	"github.com/nimbleflux/fluxbase/internal/api/routes"
	"github.com/nimbleflux/fluxbase/internal/middleware"
)

func (s *Server) buildStorageRouteDeps() *routes.StorageDeps {
	return &routes.StorageDeps{
		RequireAuth:            s.requireAuth,
		OptionalAuth:           s.optionalAuth,
		RequireScope:           middleware.RequireScope,
		DownloadSignedObject:   s.Storage.Handler.DownloadSignedObject,
		GetTransformConfig:     s.Storage.Handler.GetTransformConfig,
		ListBuckets:            s.Storage.Handler.ListBuckets,
		CreateBucket:           s.Storage.Handler.CreateBucket,
		UpdateBucketSettings:   s.Storage.Handler.UpdateBucketSettings,
		DeleteBucket:           s.Storage.Handler.DeleteBucket,
		ListFiles:              s.Storage.Handler.ListFiles,
		MultipartUpload:        s.Storage.Handler.MultipartUpload,
		ShareObject:            s.Storage.Handler.ShareObject,
		RevokeShare:            s.Storage.Handler.RevokeShare,
		ListShares:             s.Storage.Handler.ListShares,
		GenerateSignedURL:      s.Storage.Handler.GenerateSignedURL,
		StreamUpload:           s.Storage.Handler.StreamUpload,
		StorageUploadLimiter:   middleware.StorageUploadLimiter(s.sharedMiddlewareStorage),
		InitChunkedUpload:      s.Storage.Handler.InitChunkedUpload,
		UploadChunk:            s.Storage.Handler.UploadChunk,
		CompleteChunkedUpload:  s.Storage.Handler.CompleteChunkedUpload,
		GetChunkedUploadStatus: s.Storage.Handler.GetChunkedUploadStatus,
		AbortChunkedUpload:     s.Storage.Handler.AbortChunkedUpload,
		UploadFile:             s.Storage.Handler.UploadFile,
		DownloadFile:           s.Storage.Handler.DownloadFile,
		DeleteFile:             s.Storage.Handler.DeleteFile,

		TenantMiddleware:   s.Middleware.Tenant,
		TenantDBMiddleware: s.Middleware.TenantDB,
	}
}
