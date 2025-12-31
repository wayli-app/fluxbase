package resources

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/fluxbase-eu/fluxbase/internal/mcp"
	"github.com/fluxbase-eu/fluxbase/internal/storage"
)

// BucketsResource provides storage buckets information
type BucketsResource struct {
	service *storage.Service
}

// NewBucketsResource creates a new buckets resource
func NewBucketsResource(service *storage.Service) *BucketsResource {
	return &BucketsResource{
		service: service,
	}
}

func (r *BucketsResource) URI() string {
	return "fluxbase://storage/buckets"
}

func (r *BucketsResource) Name() string {
	return "Storage Buckets"
}

func (r *BucketsResource) Description() string {
	return "List of available storage buckets with their configurations"
}

func (r *BucketsResource) MimeType() string {
	return "application/json"
}

func (r *BucketsResource) RequiredScopes() []string {
	return []string{mcp.ScopeReadStorage}
}

func (r *BucketsResource) Read(ctx context.Context, authCtx *mcp.AuthContext) ([]mcp.Content, error) {
	if r.service == nil {
		return nil, fmt.Errorf("storage service not available")
	}

	// List all buckets
	buckets, err := r.service.ListBuckets(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list buckets: %w", err)
	}

	// Build response
	bucketList := make([]map[string]any, 0, len(buckets))
	for _, bucket := range buckets {
		bucketInfo := map[string]any{
			"name":       bucket.Name,
			"public":     bucket.Public,
			"created_at": bucket.CreatedAt.Format("2006-01-02T15:04:05Z"),
		}

		if bucket.FileSizeLimit != nil && *bucket.FileSizeLimit > 0 {
			bucketInfo["file_size_limit"] = *bucket.FileSizeLimit
		}

		if len(bucket.AllowedMimeTypes) > 0 {
			bucketInfo["allowed_mime_types"] = bucket.AllowedMimeTypes
		}

		bucketList = append(bucketList, bucketInfo)
	}

	result := map[string]any{
		"buckets": bucketList,
		"count":   len(bucketList),
	}

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to serialize buckets: %w", err)
	}

	return []mcp.Content{mcp.TextContent(string(data))}, nil
}
