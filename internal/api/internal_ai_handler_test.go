package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInternalAIHandler_HandleChat_NoAIService(t *testing.T) {
	app := fiber.New()
	handler := NewInternalAIHandler(nil, nil, "")

	app.Post("/api/v1/internal/ai/chat", handler.HandleChat)

	reqBody := InternalChatRequest{
		Messages: []InternalChatMessage{
			{Role: "user", Content: "Hello"},
		},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/internal/ai/chat", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)

	respBody, _ := io.ReadAll(resp.Body)
	var result map[string]any
	json.Unmarshal(respBody, &result)
	assert.Contains(t, result["error"], "AI service not configured")
}

func TestInternalAIHandler_HandleChat_EmptyMessages(t *testing.T) {
	app := fiber.New()
	// We still need nil for aiStorage to test validation before provider lookup
	handler := NewInternalAIHandler(nil, nil, "test-provider")

	app.Post("/api/v1/internal/ai/chat", handler.HandleChat)

	reqBody := InternalChatRequest{
		Messages: []InternalChatMessage{},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/internal/ai/chat", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)

	// Should fail because aiStorage is nil
	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
}

func TestInternalAIHandler_HandleEmbed_NoService(t *testing.T) {
	app := fiber.New()
	handler := NewInternalAIHandler(nil, nil, "")

	app.Post("/api/v1/internal/ai/embed", handler.HandleEmbed)

	reqBody := InternalEmbedRequest{
		Text: "Hello world",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/internal/ai/embed", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)

	respBody, _ := io.ReadAll(resp.Body)
	var result map[string]any
	json.Unmarshal(respBody, &result)
	assert.Contains(t, result["error"], "Embedding service not configured")
}

func TestInternalAIHandler_HandleEmbed_EmptyText(t *testing.T) {
	app := fiber.New()
	// Handler with nil embedding service
	handler := NewInternalAIHandler(nil, nil, "")

	app.Post("/api/v1/internal/ai/embed", handler.HandleEmbed)

	reqBody := InternalEmbedRequest{
		Text: "",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/internal/ai/embed", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)

	// Should fail because embedding service is nil, not due to empty text validation
	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
}

func TestInternalAIHandler_HandleListProviders_NoService(t *testing.T) {
	app := fiber.New()
	handler := NewInternalAIHandler(nil, nil, "")

	app.Get("/api/v1/internal/ai/providers", handler.HandleListProviders)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/internal/ai/providers", nil)

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)

	respBody, _ := io.ReadAll(resp.Body)
	var result map[string]any
	json.Unmarshal(respBody, &result)
	assert.Contains(t, result["error"], "AI service not configured")
}

func TestInternalChatRequest_Parsing(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		wantErr  bool
		validate func(t *testing.T, req InternalChatRequest)
	}{
		{
			name: "basic message",
			json: `{"messages":[{"role":"user","content":"Hello"}]}`,
			validate: func(t *testing.T, req InternalChatRequest) {
				assert.Len(t, req.Messages, 1)
				assert.Equal(t, "user", req.Messages[0].Role)
				assert.Equal(t, "Hello", req.Messages[0].Content)
			},
		},
		{
			name: "with provider and model",
			json: `{"messages":[{"role":"system","content":"You are helpful"}],"provider":"openai","model":"gpt-4"}`,
			validate: func(t *testing.T, req InternalChatRequest) {
				assert.Equal(t, "openai", req.Provider)
				assert.Equal(t, "gpt-4", req.Model)
			},
		},
		{
			name: "with max_tokens and temperature",
			json: `{"messages":[{"role":"user","content":"Hi"}],"max_tokens":100,"temperature":0.5}`,
			validate: func(t *testing.T, req InternalChatRequest) {
				assert.Equal(t, 100, req.MaxTokens)
				assert.NotNil(t, req.Temperature)
				assert.Equal(t, 0.5, *req.Temperature)
			},
		},
		{
			name:    "invalid json",
			json:    `{"messages":`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req InternalChatRequest
			err := json.Unmarshal([]byte(tt.json), &req)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, req)
				}
			}
		})
	}
}

func TestInternalEmbedRequest_Parsing(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		wantErr  bool
		validate func(t *testing.T, req InternalEmbedRequest)
	}{
		{
			name: "basic text",
			json: `{"text":"Hello world"}`,
			validate: func(t *testing.T, req InternalEmbedRequest) {
				assert.Equal(t, "Hello world", req.Text)
			},
		},
		{
			name: "with provider",
			json: `{"text":"Hello","provider":"openai"}`,
			validate: func(t *testing.T, req InternalEmbedRequest) {
				assert.Equal(t, "Hello", req.Text)
				assert.Equal(t, "openai", req.Provider)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req InternalEmbedRequest
			err := json.Unmarshal([]byte(tt.json), &req)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, req)
				}
			}
		})
	}
}

func TestMarshalEmbedding(t *testing.T) {
	embedding := []float32{0.1, 0.2, 0.3, -0.5}
	result, err := marshalEmbedding(embedding)
	require.NoError(t, err)
	assert.Equal(t, "[0.1,0.2,0.3,-0.5]", result)
}
