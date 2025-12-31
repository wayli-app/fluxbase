package tools

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/fluxbase-eu/fluxbase/internal/mcp"
	"github.com/fluxbase-eu/fluxbase/internal/storage"
	"github.com/rs/zerolog/log"
)

// ListObjectsTool implements the list_objects MCP tool
type ListObjectsTool struct {
	service *storage.Service
}

// NewListObjectsTool creates a new list_objects tool
func NewListObjectsTool(service *storage.Service) *ListObjectsTool {
	return &ListObjectsTool{
		service: service,
	}
}

func (t *ListObjectsTool) Name() string {
	return "list_objects"
}

func (t *ListObjectsTool) Description() string {
	return "List objects in a storage bucket. Returns file metadata without content."
}

func (t *ListObjectsTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"bucket": map[string]any{
				"type":        "string",
				"description": "The bucket name to list objects from",
			},
			"prefix": map[string]any{
				"type":        "string",
				"description": "Optional prefix to filter objects (e.g., 'images/' for objects in that folder)",
			},
			"limit": map[string]any{
				"type":        "integer",
				"description": "Maximum number of objects to return (default: 100, max: 1000)",
				"default":     100,
				"maximum":     1000,
			},
			"start_after": map[string]any{
				"type":        "string",
				"description": "Start listing after this key (for pagination)",
			},
		},
		"required": []string{"bucket"},
	}
}

func (t *ListObjectsTool) RequiredScopes() []string {
	return []string{mcp.ScopeReadStorage}
}

func (t *ListObjectsTool) Execute(ctx context.Context, args map[string]any, authCtx *mcp.AuthContext) (*mcp.ToolResult, error) {
	bucket, ok := args["bucket"].(string)
	if !ok || bucket == "" {
		return nil, fmt.Errorf("bucket is required")
	}

	prefix := ""
	if p, ok := args["prefix"].(string); ok {
		prefix = p
	}

	maxKeys := 100
	if l, ok := args["limit"].(float64); ok {
		maxKeys = int(l)
		if maxKeys > 1000 {
			maxKeys = 1000
		}
	}

	startAfter := ""
	if s, ok := args["start_after"].(string); ok {
		startAfter = s
	}

	log.Debug().
		Str("bucket", bucket).
		Str("prefix", prefix).
		Int("max_keys", maxKeys).
		Msg("MCP: Listing objects")

	// List objects
	listOpts := &storage.ListOptions{
		Prefix:     prefix,
		MaxKeys:    maxKeys,
		StartAfter: startAfter,
	}
	listResult, err := t.service.Provider.List(ctx, bucket, listOpts)
	if err != nil {
		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.ErrorContent(fmt.Sprintf("Failed to list objects: %v", err))},
			IsError: true,
		}, nil
	}

	// Convert to response format
	objectList := make([]map[string]any, 0, len(listResult.Objects))
	for _, obj := range listResult.Objects {
		objectList = append(objectList, map[string]any{
			"key":           obj.Key,
			"size":          obj.Size,
			"content_type":  obj.ContentType,
			"etag":          obj.ETag,
			"last_modified": obj.LastModified.Format("2006-01-02T15:04:05Z"),
		})
	}

	result := map[string]any{
		"objects":      objectList,
		"count":        len(objectList),
		"is_truncated": listResult.IsTruncated,
	}
	if listResult.NextMarker != "" {
		result["next_marker"] = listResult.NextMarker
	}

	resultJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.ErrorContent(fmt.Sprintf("Failed to serialize result: %v", err))},
			IsError: true,
		}, nil
	}

	return &mcp.ToolResult{
		Content: []mcp.Content{mcp.TextContent(string(resultJSON))},
	}, nil
}

// DownloadObjectTool implements the download_object MCP tool
type DownloadObjectTool struct {
	service *storage.Service
}

// NewDownloadObjectTool creates a new download_object tool
func NewDownloadObjectTool(service *storage.Service) *DownloadObjectTool {
	return &DownloadObjectTool{
		service: service,
	}
}

func (t *DownloadObjectTool) Name() string {
	return "download_object"
}

func (t *DownloadObjectTool) Description() string {
	return "Download a file from storage. Returns base64-encoded content for binary files or text content for text files."
}

func (t *DownloadObjectTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"bucket": map[string]any{
				"type":        "string",
				"description": "The bucket name",
			},
			"key": map[string]any{
				"type":        "string",
				"description": "The object key (path within the bucket)",
			},
		},
		"required": []string{"bucket", "key"},
	}
}

func (t *DownloadObjectTool) RequiredScopes() []string {
	return []string{mcp.ScopeReadStorage}
}

func (t *DownloadObjectTool) Execute(ctx context.Context, args map[string]any, authCtx *mcp.AuthContext) (*mcp.ToolResult, error) {
	bucket, ok := args["bucket"].(string)
	if !ok || bucket == "" {
		return nil, fmt.Errorf("bucket is required")
	}

	key, ok := args["key"].(string)
	if !ok || key == "" {
		return nil, fmt.Errorf("key is required")
	}

	log.Debug().
		Str("bucket", bucket).
		Str("key", key).
		Msg("MCP: Downloading object")

	// Download file
	reader, obj, err := t.service.Provider.Download(ctx, bucket, key, nil)
	if err != nil {
		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.ErrorContent(fmt.Sprintf("Failed to download file: %v", err))},
			IsError: true,
		}, nil
	}
	defer reader.Close()

	// Read content (limit to 10MB to prevent memory issues)
	const maxSize = 10 * 1024 * 1024
	data, err := io.ReadAll(io.LimitReader(reader, maxSize))
	if err != nil {
		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.ErrorContent(fmt.Sprintf("Failed to read file content: %v", err))},
			IsError: true,
		}, nil
	}

	// Determine if content is text or binary
	isText := isTextContentType(obj.ContentType)

	result := map[string]any{
		"key":          obj.Key,
		"size":         obj.Size,
		"content_type": obj.ContentType,
		"etag":         obj.ETag,
	}

	if isText {
		result["content"] = string(data)
		result["encoding"] = "text"
	} else {
		result["content"] = base64.StdEncoding.EncodeToString(data)
		result["encoding"] = "base64"
	}

	if len(data) >= maxSize {
		result["truncated"] = true
	}

	resultJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.ErrorContent(fmt.Sprintf("Failed to serialize result: %v", err))},
			IsError: true,
		}, nil
	}

	return &mcp.ToolResult{
		Content: []mcp.Content{mcp.TextContent(string(resultJSON))},
	}, nil
}

// UploadObjectTool implements the upload_object MCP tool
type UploadObjectTool struct {
	service *storage.Service
}

// NewUploadObjectTool creates a new upload_object tool
func NewUploadObjectTool(service *storage.Service) *UploadObjectTool {
	return &UploadObjectTool{
		service: service,
	}
}

func (t *UploadObjectTool) Name() string {
	return "upload_object"
}

func (t *UploadObjectTool) Description() string {
	return "Upload a file to storage. Content can be text or base64-encoded binary data."
}

func (t *UploadObjectTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"bucket": map[string]any{
				"type":        "string",
				"description": "The bucket name",
			},
			"key": map[string]any{
				"type":        "string",
				"description": "The object key (path within the bucket)",
			},
			"content": map[string]any{
				"type":        "string",
				"description": "File content (text or base64-encoded)",
			},
			"content_type": map[string]any{
				"type":        "string",
				"description": "MIME type of the file (default: auto-detect from key)",
			},
			"encoding": map[string]any{
				"type":        "string",
				"description": "Content encoding: 'text' or 'base64' (default: 'text')",
				"enum":        []string{"text", "base64"},
				"default":     "text",
			},
		},
		"required": []string{"bucket", "key", "content"},
	}
}

func (t *UploadObjectTool) RequiredScopes() []string {
	return []string{mcp.ScopeWriteStorage}
}

func (t *UploadObjectTool) Execute(ctx context.Context, args map[string]any, authCtx *mcp.AuthContext) (*mcp.ToolResult, error) {
	bucket, ok := args["bucket"].(string)
	if !ok || bucket == "" {
		return nil, fmt.Errorf("bucket is required")
	}

	key, ok := args["key"].(string)
	if !ok || key == "" {
		return nil, fmt.Errorf("key is required")
	}

	content, ok := args["content"].(string)
	if !ok {
		return nil, fmt.Errorf("content is required")
	}

	encoding := "text"
	if e, ok := args["encoding"].(string); ok {
		encoding = e
	}

	contentType := ""
	if ct, ok := args["content_type"].(string); ok {
		contentType = ct
	}

	log.Debug().
		Str("bucket", bucket).
		Str("key", key).
		Str("encoding", encoding).
		Msg("MCP: Uploading object")

	// Decode content
	var data []byte
	var err error
	if encoding == "base64" {
		data, err = base64.StdEncoding.DecodeString(content)
		if err != nil {
			return nil, fmt.Errorf("failed to decode base64 content: %w", err)
		}
	} else {
		data = []byte(content)
	}

	// Validate size
	if err := t.service.ValidateUploadSize(int64(len(data))); err != nil {
		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.ErrorContent(fmt.Sprintf("File too large: %v", err))},
			IsError: true,
		}, nil
	}

	// Upload
	opts := &storage.UploadOptions{
		ContentType: contentType,
	}

	obj, err := t.service.Provider.Upload(ctx, bucket, key, bytes.NewReader(data), int64(len(data)), opts)
	if err != nil {
		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.ErrorContent(fmt.Sprintf("Failed to upload file: %v", err))},
			IsError: true,
		}, nil
	}

	result := map[string]any{
		"key":          obj.Key,
		"size":         obj.Size,
		"content_type": obj.ContentType,
		"etag":         obj.ETag,
	}

	resultJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.ErrorContent(fmt.Sprintf("Failed to serialize result: %v", err))},
			IsError: true,
		}, nil
	}

	return &mcp.ToolResult{
		Content: []mcp.Content{mcp.TextContent(string(resultJSON))},
	}, nil
}

// DeleteObjectTool implements the delete_object MCP tool
type DeleteObjectTool struct {
	service *storage.Service
}

// NewDeleteObjectTool creates a new delete_object tool
func NewDeleteObjectTool(service *storage.Service) *DeleteObjectTool {
	return &DeleteObjectTool{
		service: service,
	}
}

func (t *DeleteObjectTool) Name() string {
	return "delete_object"
}

func (t *DeleteObjectTool) Description() string {
	return "Delete a file from storage."
}

func (t *DeleteObjectTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"bucket": map[string]any{
				"type":        "string",
				"description": "The bucket name",
			},
			"key": map[string]any{
				"type":        "string",
				"description": "The object key (path within the bucket)",
			},
		},
		"required": []string{"bucket", "key"},
	}
}

func (t *DeleteObjectTool) RequiredScopes() []string {
	return []string{mcp.ScopeWriteStorage}
}

func (t *DeleteObjectTool) Execute(ctx context.Context, args map[string]any, authCtx *mcp.AuthContext) (*mcp.ToolResult, error) {
	bucket, ok := args["bucket"].(string)
	if !ok || bucket == "" {
		return nil, fmt.Errorf("bucket is required")
	}

	key, ok := args["key"].(string)
	if !ok || key == "" {
		return nil, fmt.Errorf("key is required")
	}

	log.Debug().
		Str("bucket", bucket).
		Str("key", key).
		Msg("MCP: Deleting object")

	// Delete file
	err := t.service.Provider.Delete(ctx, bucket, key)
	if err != nil {
		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.ErrorContent(fmt.Sprintf("Failed to delete file: %v", err))},
			IsError: true,
		}, nil
	}

	result := map[string]any{
		"deleted": true,
		"bucket":  bucket,
		"key":     key,
	}

	resultJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.ErrorContent(fmt.Sprintf("Failed to serialize result: %v", err))},
			IsError: true,
		}, nil
	}

	return &mcp.ToolResult{
		Content: []mcp.Content{mcp.TextContent(string(resultJSON))},
	}, nil
}

// isTextContentType checks if a content type is text-based
func isTextContentType(contentType string) bool {
	textTypes := []string{
		"text/",
		"application/json",
		"application/xml",
		"application/javascript",
		"application/typescript",
		"application/x-yaml",
		"application/yaml",
	}

	ct := strings.ToLower(contentType)
	for _, prefix := range textTypes {
		if strings.HasPrefix(ct, prefix) {
			return true
		}
	}
	return false
}
