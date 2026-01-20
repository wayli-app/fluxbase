package custom

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestCustomTool_Defaults(t *testing.T) {
	tool := CustomTool{
		ID:        uuid.New(),
		Name:      "test_tool",
		Namespace: "default",
		Code:      "export function handler() {}",
	}

	assert.NotEmpty(t, tool.ID)
	assert.Equal(t, "test_tool", tool.Name)
	assert.Equal(t, "default", tool.Namespace)
}

func TestCustomResource_Defaults(t *testing.T) {
	resource := CustomResource{
		ID:        uuid.New(),
		URI:       "fluxbase://custom/test",
		Name:      "test_resource",
		Namespace: "default",
		MimeType:  "application/json",
		Code:      "export function handler() {}",
	}

	assert.NotEmpty(t, resource.ID)
	assert.Equal(t, "fluxbase://custom/test", resource.URI)
	assert.Equal(t, "application/json", resource.MimeType)
}

func TestCreateToolRequest_Validation(t *testing.T) {
	tests := []struct {
		name    string
		req     CreateToolRequest
		wantErr bool
	}{
		{
			name: "valid request",
			req: CreateToolRequest{
				Name: "my_tool",
				Code: "export function handler() {}",
			},
			wantErr: false,
		},
		{
			name: "with optional fields",
			req: CreateToolRequest{
				Name:           "my_tool",
				Code:           "export function handler() {}",
				Namespace:      "production",
				Description:    "A test tool",
				TimeoutSeconds: 60,
				MemoryLimitMB:  256,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Basic validation
			hasError := tt.req.Name == "" || tt.req.Code == ""
			assert.Equal(t, tt.wantErr, hasError)
		})
	}
}

func TestListToolsFilter(t *testing.T) {
	filter := ListToolsFilter{
		Namespace:   "production",
		EnabledOnly: true,
		Limit:       10,
		Offset:      5,
	}

	assert.Equal(t, "production", filter.Namespace)
	assert.True(t, filter.EnabledOnly)
	assert.Equal(t, 10, filter.Limit)
	assert.Equal(t, 5, filter.Offset)
}

func TestContent_Types(t *testing.T) {
	textContent := Content{
		Type: "text",
		Text: "Hello, world!",
	}
	assert.Equal(t, "text", textContent.Type)
	assert.Equal(t, "Hello, world!", textContent.Text)

	imageContent := Content{
		Type:     "image",
		MimeType: "image/png",
		Data:     "base64encodeddata",
	}
	assert.Equal(t, "image", imageContent.Type)
	assert.Equal(t, "image/png", imageContent.MimeType)
}

func TestCustomTool_Timestamps(t *testing.T) {
	now := time.Now()
	tool := CustomTool{
		ID:        uuid.New(),
		Name:      "test",
		Namespace: "default",
		Code:      "test",
		CreatedAt: now,
		UpdatedAt: now,
	}

	assert.False(t, tool.CreatedAt.IsZero())
	assert.False(t, tool.UpdatedAt.IsZero())
}

func TestSyncToolRequest(t *testing.T) {
	req := SyncToolRequest{
		CreateToolRequest: CreateToolRequest{
			Name:        "sync_tool",
			Code:        "export function handler() {}",
			Description: "Synced tool",
		},
		Upsert: true,
	}

	assert.Equal(t, "sync_tool", req.Name)
	assert.True(t, req.Upsert)
}
