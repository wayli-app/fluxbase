package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/fluxbase-eu/fluxbase/internal/database"
	"github.com/fluxbase-eu/fluxbase/internal/mcp"
)

// BucketsResource provides storage buckets information
type BucketsResource struct {
	db *database.Connection
}

// NewBucketsResource creates a new buckets resource
func NewBucketsResource(db *database.Connection) *BucketsResource {
	return &BucketsResource{
		db: db,
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
	if r.db == nil {
		return nil, fmt.Errorf("database connection not available")
	}

	// Query buckets from database
	rows, err := r.db.Query(ctx, `
		SELECT id, name, public, allowed_mime_types, max_file_size, created_at, updated_at
		FROM storage.buckets
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query buckets: %w", err)
	}
	defer rows.Close()

	// Build response
	type Bucket struct {
		ID               string    `json:"id"`
		Name             string    `json:"name"`
		Public           bool      `json:"public"`
		AllowedMimeTypes []string  `json:"allowed_mime_types"`
		MaxFileSize      *int64    `json:"max_file_size"`
		CreatedAt        time.Time `json:"created_at"`
		UpdatedAt        time.Time `json:"updated_at"`
	}

	bucketList := make([]map[string]any, 0)
	for rows.Next() {
		var bucket Bucket
		if err := rows.Scan(&bucket.ID, &bucket.Name, &bucket.Public, &bucket.AllowedMimeTypes, &bucket.MaxFileSize, &bucket.CreatedAt, &bucket.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan bucket: %w", err)
		}

		bucketInfo := map[string]any{
			"id":         bucket.ID,
			"name":       bucket.Name,
			"public":     bucket.Public,
			"created_at": bucket.CreatedAt.Format("2006-01-02T15:04:05Z"),
		}

		if bucket.MaxFileSize != nil && *bucket.MaxFileSize > 0 {
			bucketInfo["max_file_size"] = *bucket.MaxFileSize
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
