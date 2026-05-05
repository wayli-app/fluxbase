package client

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nimbleflux/fluxbase/cli/config"
)

func newTestClient(serverURL string) *Client {
	cfg := config.New()
	profile := &config.Profile{
		Name:   "test",
		Server: serverURL,
		Credentials: &config.Credentials{
			APIKey: "test-token",
		},
	}
	return NewClient(cfg, profile)
}

func TestNewClient_Options(t *testing.T) {
	cfg := config.New()
	profile := &config.Profile{
		Name:   "test",
		Server: "http://localhost:8080",
	}

	c := NewClient(
		cfg, profile,
		WithDebug(true),
		WithTimeout(10*time.Second),
		WithConfigPath("/tmp/test-config"),
	)

	assert.True(t, c.Debug)
	assert.Equal(t, 10*time.Second, c.HTTPClient.Timeout)
	assert.Equal(t, "/tmp/test-config", c.ConfigPath)
}

func TestNewClient_Defaults(t *testing.T) {
	cfg := config.New()
	profile := &config.Profile{
		Name:   "test",
		Server: "http://localhost:8080",
	}

	c := NewClient(cfg, profile)
	assert.Equal(t, "http://localhost:8080", c.BaseURL)
	assert.Equal(t, 30*time.Second, c.HTTPClient.Timeout)
	assert.Equal(t, "fluxbase-cli/1.0", c.UserAgent)
	assert.False(t, c.Debug)
}

func TestDoGet_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/test", r.URL.Path)
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{
				{"id": 1, "name": "item1"},
				{"id": 2, "name": "item2"},
			},
		})
	}))
	defer server.Close()

	c := newTestClient(server.URL)
	var result map[string]interface{}
	err := c.DoGet(context.Background(), "/api/v1/test", nil, &result)
	require.NoError(t, err)
	assert.NotNil(t, result["data"])
}

func TestDoGet_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "resource not found",
			"code":    "NOT_FOUND",
		})
	}))
	defer server.Close()

	c := newTestClient(server.URL)
	var result map[string]interface{}
	err := c.DoGet(context.Background(), "/api/v1/missing", nil, &result)

	require.Error(t, err)
	var apiErr *APIError
	require.True(t, errors.As(err, &apiErr))
	assert.Equal(t, http.StatusNotFound, apiErr.StatusCode)
	assert.Equal(t, "resource not found", apiErr.Message)
	assert.Equal(t, "NOT_FOUND", apiErr.Code)
}

func TestDoGet_InternalError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal server error"))
	}))
	defer server.Close()

	c := newTestClient(server.URL)
	var result map[string]interface{}
	err := c.DoGet(context.Background(), "/api/v1/test", nil, &result)

	require.Error(t, err)
	var apiErr *APIError
	require.True(t, errors.As(err, &apiErr))
	assert.Equal(t, http.StatusInternalServerError, apiErr.StatusCode)
}

func TestDoPost_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)

		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]string
		require.NoError(t, json.Unmarshal(body, &reqBody))
		assert.Equal(t, "test-func", reqBody["name"])

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   "func-123",
			"name": "test-func",
		})
	}))
	defer server.Close()

	c := newTestClient(server.URL)
	var result map[string]interface{}
	err := c.DoPost(context.Background(), "/api/v1/functions", map[string]string{
		"name": "test-func",
	}, &result)

	require.NoError(t, err)
	assert.Equal(t, "func-123", result["id"])
	assert.Equal(t, "test-func", result["name"])
}

func TestDoPost_Conflict(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "resource already exists",
		})
	}))
	defer server.Close()

	c := newTestClient(server.URL)
	var result map[string]interface{}
	err := c.DoPost(context.Background(), "/api/v1/test", nil, &result)

	require.Error(t, err)
	var apiErr *APIError
	require.True(t, errors.As(err, &apiErr))
	assert.Equal(t, http.StatusConflict, apiErr.StatusCode)
	assert.Equal(t, "resource already exists", apiErr.Message)
}

func TestDoPut_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPut, r.Method)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":     "func-123",
			"status": "updated",
		})
	}))
	defer server.Close()

	c := newTestClient(server.URL)
	var result map[string]interface{}
	err := c.DoPut(context.Background(), "/api/v1/functions/func-123", map[string]string{
		"status": "active",
	}, &result)

	require.NoError(t, err)
	assert.Equal(t, "updated", result["status"])
}

func TestDoDelete_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	c := newTestClient(server.URL)
	err := c.DoDelete(context.Background(), "/api/v1/functions/func-123")
	assert.NoError(t, err)
}

func TestDoDelete_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "not found",
		})
	}))
	defer server.Close()

	c := newTestClient(server.URL)
	err := c.DoDelete(context.Background(), "/api/v1/functions/missing")

	require.Error(t, err)
	var apiErr *APIError
	require.True(t, errors.As(err, &apiErr))
	assert.Equal(t, http.StatusNotFound, apiErr.StatusCode)
}

func TestDoPatch_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPatch, r.Method)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"updated": true,
		})
	}))
	defer server.Close()

	c := newTestClient(server.URL)
	var result map[string]interface{}
	err := c.DoPatch(context.Background(), "/api/v1/test", map[string]string{"key": "val"}, &result)

	require.NoError(t, err)
	assert.True(t, result["updated"].(bool))
}

func TestContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second) // Block to let context cancel
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	c := newTestClient(server.URL)
	var result map[string]interface{}
	err := c.DoGet(ctx, "/api/v1/test", nil, &result)

	assert.Error(t, err)
}

func TestAPIError_Error(t *testing.T) {
	t.Run("with message", func(t *testing.T) {
		e := &APIError{StatusCode: 400, Message: "bad request"}
		assert.Equal(t, "bad request", e.Error())
	})

	t.Run("with error field", func(t *testing.T) {
		e := &APIError{StatusCode: 500, Error_: "internal error"}
		assert.Equal(t, "internal error", e.Error())
	})

	t.Run("fallback to status code", func(t *testing.T) {
		e := &APIError{StatusCode: 503}
		assert.Equal(t, "API error with status 503", e.Error())
	})
}

func TestAddAuth_NoCredentials(t *testing.T) {
	cfg := config.New()
	profile := &config.Profile{
		Name:   "test",
		Server: "http://localhost:8080",
		// No credentials
	}
	c := NewClient(cfg, profile)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	err := c.addAuth(req)
	assert.Error(t, err)
	// When no credentials are set, either "not authenticated" or "profile not found" is expected
	assert.Error(t, err)
}

func TestAddAuth_BranchHeader(t *testing.T) {
	var capturedBranch string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedBranch = r.Header.Get("X-Fluxbase-Branch")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{})
	}))
	defer server.Close()

	cfg := config.New()
	profile := &config.Profile{
		Name:          "test",
		Server:        server.URL,
		DefaultBranch: "feature-x",
		Credentials:   &config.Credentials{APIKey: "test-key"},
	}
	c := NewClient(cfg, profile)

	var result map[string]interface{}
	_ = c.DoGet(context.Background(), "/api/v1/test", nil, &result)

	assert.Equal(t, "feature-x", capturedBranch)
}

func TestAddAuth_MainBranch_NoHeader(t *testing.T) {
	var capturedBranch string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedBranch = r.Header.Get("X-Fluxbase-Branch")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{})
	}))
	defer server.Close()

	cfg := config.New()
	profile := &config.Profile{
		Name:          "test",
		Server:        server.URL,
		DefaultBranch: "main",
		Credentials:   &config.Credentials{APIKey: "test-key"},
	}
	c := NewClient(cfg, profile)

	var result map[string]interface{}
	_ = c.DoGet(context.Background(), "/api/v1/test", nil, &result)

	assert.Empty(t, capturedBranch)
}

func TestParseError(t *testing.T) {
	t.Run("valid JSON error", func(t *testing.T) {
		body := `{"message":"validation failed","code":"INVALID_INPUT","error":"bad data"}`
		resp := &http.Response{
			StatusCode: 400,
			Body:       io.NopCloser(strings.NewReader(body)),
		}

		err := ParseError(resp)
		require.Error(t, err)
		var apiErr *APIError
		require.True(t, errors.As(err, &apiErr))
		assert.Equal(t, 400, apiErr.StatusCode)
		assert.Equal(t, "validation failed", apiErr.Message)
		assert.Equal(t, "INVALID_INPUT", apiErr.Code)
		assert.Equal(t, "bad data", apiErr.Error_)
	})

	t.Run("non-JSON body", func(t *testing.T) {
		resp := &http.Response{
			StatusCode: 500,
			Body:       io.NopCloser(strings.NewReader("plain text error")),
		}

		err := ParseError(resp)
		require.Error(t, err)
		var apiErr *APIError
		require.True(t, errors.As(err, &apiErr))
		assert.Equal(t, 500, apiErr.StatusCode)
		assert.Equal(t, "plain text error", apiErr.Message)
	})

	t.Run("empty body", func(t *testing.T) {
		resp := &http.Response{
			StatusCode: 500,
			Body:       io.NopCloser(strings.NewReader("")),
		}

		err := ParseError(resp)
		require.Error(t, err)
		var apiErr *APIError
		require.True(t, errors.As(err, &apiErr))
		assert.Equal(t, 500, apiErr.StatusCode)
	})
}

func TestDecodeResponse(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		body := `{"name":"test","count":42}`
		resp := &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(body)),
		}

		var result map[string]interface{}
		err := DecodeResponse(resp, &result)
		require.NoError(t, err)
		assert.Equal(t, "test", result["name"])
	})

	t.Run("nil target", func(t *testing.T) {
		resp := &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader("{}")),
		}

		err := DecodeResponse(resp, nil)
		assert.NoError(t, err)
	})

	t.Run("error status", func(t *testing.T) {
		body := `{"message":"unauthorized"}`
		resp := &http.Response{
			StatusCode: 401,
			Body:       io.NopCloser(strings.NewReader(body)),
		}

		err := DecodeResponse(resp, nil)
		require.Error(t, err)
		var apiErr *APIError
		require.True(t, errors.As(err, &apiErr))
		assert.Equal(t, 401, apiErr.StatusCode)
	})
}

func TestQueryParameters(t *testing.T) {
	var capturedQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{})
	}))
	defer server.Close()

	c := newTestClient(server.URL)

	query := make(url.Values)
	query.Set("limit", "10")
	query.Set("offset", "20")

	var result map[string]interface{}
	err := c.DoGet(context.Background(), "/api/v1/test", query, &result)
	require.NoError(t, err)
	assert.Contains(t, capturedQuery, "limit=10")
	assert.Contains(t, capturedQuery, "offset=20")
}
