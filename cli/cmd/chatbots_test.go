package cmd

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChatbotsList_Success(t *testing.T) {
	resetChatbotFlags()
	_, buf, cleanup := setupTestEnvWithHandler(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Contains(t, r.URL.Path, "/api/v1/admin/ai/chatbots")

		respondJSON(w, http.StatusOK, map[string]interface{}{
			"chatbots": []map[string]interface{}{
				{"id": "cb1", "name": "support-bot", "model": "gpt-4", "enabled": true},
				{"id": "cb2", "name": "sales-bot", "model": "gpt-3.5-turbo", "enabled": false},
			},
			"count": 2,
		})
	})
	defer cleanup()

	err := runChatbotsList(nil, []string{})
	require.NoError(t, err)

	var result []map[string]interface{}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &result))
	require.Len(t, result, 2)
	assert.Equal(t, "cb1", result[0]["id"])
	assert.Equal(t, "cb2", result[1]["id"])
}

func TestChatbotsList_Empty(t *testing.T) {
	resetChatbotFlags()
	_, _, cleanup := setupTestEnvWithHandler(t, func(w http.ResponseWriter, r *http.Request) {
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"chatbots": []map[string]interface{}{},
			"count":    0,
		})
	})
	defer cleanup()

	err := runChatbotsList(nil, []string{})
	require.NoError(t, err)
}

func TestChatbotsList_APIError(t *testing.T) {
	resetChatbotFlags()
	_, _, cleanup := setupTestEnvWithHandler(t, func(w http.ResponseWriter, r *http.Request) {
		respondError(w, http.StatusInternalServerError, "database error")
	})
	defer cleanup()

	err := runChatbotsList(nil, []string{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "database error")
}

func TestChatbotsList_NamespaceFilter_SendsQueryParam(t *testing.T) {
	resetChatbotFlags()
	cbListNamespace = "wayli"

	_, _, cleanup := setupTestEnvWithHandler(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "wayli", r.URL.Query().Get("namespace"),
			"CLI must forward --namespace as a query param to the API")

		respondJSON(w, http.StatusOK, map[string]interface{}{
			"chatbots": []map[string]interface{}{
				{"id": "cb1", "name": "wayli-bot", "model": "gpt-4", "enabled": true},
			},
			"count": 1,
		})
	})
	defer cleanup()

	err := runChatbotsList(nil, []string{})
	require.NoError(t, err)
}

func TestChatbotsList_NoNamespaceFilter_OmitsQueryParam(t *testing.T) {
	resetChatbotFlags()

	_, _, cleanup := setupTestEnvWithHandler(t, func(w http.ResponseWriter, r *http.Request) {
		// When --namespace not set, the API call must not include the param
		// so the server lists across all namespaces.
		_, hasNs := r.URL.Query()["namespace"]
		assert.False(t, hasNs, "expected no namespace query param when flag unset")

		respondJSON(w, http.StatusOK, map[string]interface{}{
			"chatbots": []map[string]interface{}{}, "count": 0,
		})
	})
	defer cleanup()

	err := runChatbotsList(nil, []string{})
	require.NoError(t, err)
}

func TestChatbotsGet_Success(t *testing.T) {
	resetChatbotFlags()
	_, buf, cleanup := setupTestEnvWithHandler(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Contains(t, r.URL.Path, "/api/v1/admin/ai/chatbots/cb1")

		respondJSON(w, http.StatusOK, map[string]interface{}{
			"id":            "cb1",
			"name":          "support-bot",
			"model":         "gpt-4",
			"system_prompt": "You are helpful.",
			"enabled":       true,
		})
	})
	defer cleanup()

	err := runChatbotsGet(nil, []string{"cb1"})
	require.NoError(t, err)

	var result map[string]interface{}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &result))
	assert.Equal(t, "cb1", result["id"])
	assert.Equal(t, "support-bot", result["name"])
}

func TestChatbotsGet_NotFound(t *testing.T) {
	resetChatbotFlags()
	_, _, cleanup := setupTestEnvWithHandler(t, func(w http.ResponseWriter, r *http.Request) {
		respondError(w, http.StatusNotFound, "chatbot not found")
	})
	defer cleanup()

	err := runChatbotsGet(nil, []string{"nonexistent"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "chatbot not found")
}

func TestChatbotsCreate_Success(t *testing.T) {
	resetChatbotFlags()
	cbSystemPrompt = "You are a helpful support assistant"
	cbModel = "gpt-4"
	cbTemperature = 0.7
	cbMaxTokens = 1024

	_, _, cleanup := setupTestEnvWithHandler(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Contains(t, r.URL.Path, "/api/v1/admin/ai/chatbots")

		var body map[string]interface{}
		readRequestBody(t, r, &body)
		assert.Equal(t, "support-bot", body["name"])
		assert.Equal(t, "gpt-4", body["model"])
		assert.Equal(t, "You are a helpful support assistant", body["system_prompt"])
		assert.Equal(t, true, body["enabled"])

		respondJSON(w, http.StatusCreated, map[string]interface{}{
			"id":   "cb-new",
			"name": "support-bot",
		})
	})
	defer cleanup()

	err := runChatbotsCreate(nil, []string{"support-bot"})
	require.NoError(t, err)
}

func TestChatbotsCreate_WithKnowledgeBase(t *testing.T) {
	resetChatbotFlags()
	cbKnowledgeBase = "kb123"

	_, _, cleanup := setupTestEnvWithHandler(t, func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		readRequestBody(t, r, &body)
		assert.Equal(t, []interface{}{"kb123"}, body["knowledge_base_ids"])

		respondJSON(w, http.StatusCreated, map[string]interface{}{
			"id":   "cb-new",
			"name": "test-bot",
		})
	})
	defer cleanup()

	err := runChatbotsCreate(nil, []string{"test-bot"})
	require.NoError(t, err)
}

func TestChatbotsCreate_APIError(t *testing.T) {
	resetChatbotFlags()
	cbModel = "gpt-3.5-turbo"
	cbTemperature = 0.7
	cbMaxTokens = 1024

	_, _, cleanup := setupTestEnvWithHandler(t, func(w http.ResponseWriter, r *http.Request) {
		respondError(w, http.StatusBadRequest, "invalid model")
	})
	defer cleanup()

	err := runChatbotsCreate(nil, []string{"test-bot"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid model")
}

func TestChatbotsUpdate_Success(t *testing.T) {
	resetChatbotFlags()
	cbSystemPrompt = "Updated prompt"
	cbModel = "gpt-4"

	_, _, cleanup := setupTestEnvWithHandler(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPut, r.Method)
		assert.Contains(t, r.URL.Path, "/api/v1/admin/ai/chatbots/cb1")

		var body map[string]interface{}
		readRequestBody(t, r, &body)
		assert.Equal(t, "Updated prompt", body["system_prompt"])
		assert.Equal(t, "gpt-4", body["model"])

		w.WriteHeader(http.StatusNoContent)
	})
	defer cleanup()

	err := runChatbotsUpdate(nil, []string{"cb1"})
	require.NoError(t, err)
}

func TestChatbotsUpdate_NoUpdates(t *testing.T) {
	resetChatbotFlags()

	err := runChatbotsUpdate(nil, []string{"cb1"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no updates specified")
}

func TestChatbotsUpdate_APIError(t *testing.T) {
	resetChatbotFlags()
	cbSystemPrompt = "Updated"

	_, _, cleanup := setupTestEnvWithHandler(t, func(w http.ResponseWriter, r *http.Request) {
		respondError(w, http.StatusNotFound, "chatbot not found")
	})
	defer cleanup()

	err := runChatbotsUpdate(nil, []string{"nonexistent"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "chatbot not found")
}

func TestChatbotsDelete_Success(t *testing.T) {
	resetChatbotFlags()
	_, _, cleanup := setupTestEnvWithHandler(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method)
		assert.Contains(t, r.URL.Path, "/api/v1/admin/ai/chatbots/cb1")

		w.WriteHeader(http.StatusNoContent)
	})
	defer cleanup()

	err := runChatbotsDelete(nil, []string{"cb1"})
	require.NoError(t, err)
}

func TestChatbotsDelete_APIError(t *testing.T) {
	resetChatbotFlags()
	_, _, cleanup := setupTestEnvWithHandler(t, func(w http.ResponseWriter, r *http.Request) {
		respondError(w, http.StatusNotFound, "chatbot not found")
	})
	defer cleanup()

	err := runChatbotsDelete(nil, []string{"nonexistent"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "chatbot not found")
}
