package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCapProvider(t *testing.T) {
	tests := []struct {
		name           string
		serverURL      string
		apiKey         string
		expectedURL    string
	}{
		{
			name:        "URL without trailing slash",
			serverURL:   "https://cap.example.com",
			apiKey:      "test-key",
			expectedURL: "https://cap.example.com",
		},
		{
			name:        "URL with trailing slash",
			serverURL:   "https://cap.example.com/",
			apiKey:      "test-key",
			expectedURL: "https://cap.example.com",
		},
		{
			name:        "URL with multiple trailing slashes",
			serverURL:   "https://cap.example.com///",
			apiKey:      "test-key",
			expectedURL: "https://cap.example.com//",
		},
		{
			name:        "Empty API key",
			serverURL:   "https://cap.example.com",
			apiKey:      "",
			expectedURL: "https://cap.example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &http.Client{}
			provider := NewCapProvider(tt.serverURL, tt.apiKey, client)

			assert.NotNil(t, provider)
			assert.Equal(t, tt.expectedURL, provider.serverURL)
			assert.Equal(t, tt.apiKey, provider.apiKey)
			assert.Equal(t, client, provider.httpClient)
		})
	}
}

func TestCapProvider_Name(t *testing.T) {
	provider := NewCapProvider("https://cap.example.com", "test-key", &http.Client{})
	assert.Equal(t, "cap", provider.Name())
}

func TestCapProvider_Verify_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method
		assert.Equal(t, "POST", r.Method)

		// Verify path
		assert.Equal(t, "/api/token/validate", r.URL.Path)

		// Verify content type
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		// Verify authorization header
		assert.Equal(t, "Bot test-key", r.Header.Get("Authorization"))

		// Parse request body
		var reqBody capVerifyRequest
		err := json.NewDecoder(r.Body).Decode(&reqBody)
		require.NoError(t, err)
		assert.Equal(t, "test-token", reqBody.Token)

		// Return success response
		response := capVerifyResponse{
			Success: true,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := server.Client()
	provider := NewCapProvider(server.URL, "test-key", client)

	ctx := context.Background()
	result, err := provider.Verify(ctx, "test-token", "")

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)
	assert.Empty(t, result.ErrorCode)
}

func TestCapProvider_Verify_WithoutAPIKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify authorization header is not set
		assert.Empty(t, r.Header.Get("Authorization"))

		response := capVerifyResponse{
			Success: true,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := server.Client()
	provider := NewCapProvider(server.URL, "", client)

	ctx := context.Background()
	result, err := provider.Verify(ctx, "test-token", "")

	require.NoError(t, err)
	assert.True(t, result.Success)
}

func TestCapProvider_Verify_Failure(t *testing.T) {
	tests := []struct {
		name              string
		errorCode         string
		expectedTranslation string
	}{
		{
			name:              "invalid token",
			errorCode:         "invalid_token",
			expectedTranslation: "invalid captcha token",
		},
		{
			name:              "expired token",
			errorCode:         "expired_token",
			expectedTranslation: "captcha token expired",
		},
		{
			name:              "already used",
			errorCode:         "already_used",
			expectedTranslation: "captcha token already used",
		},
		{
			name:              "invalid solution",
			errorCode:         "invalid_solution",
			expectedTranslation: "invalid proof-of-work solution",
		},
		{
			name:              "missing token",
			errorCode:         "missing_token",
			expectedTranslation: "missing captcha token",
		},
		{
			name:              "unknown error",
			errorCode:         "unknown_error",
			expectedTranslation: "verification failed: unknown_error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				response := capVerifyResponse{
					Success: false,
					Error:   tt.errorCode,
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(response)
			}))
			defer server.Close()

			client := server.Client()
			provider := NewCapProvider(server.URL, "test-key", client)

			ctx := context.Background()
			result, err := provider.Verify(ctx, "test-token", "")

			require.NoError(t, err)
			assert.NotNil(t, result)
			assert.False(t, result.Success)
			assert.Equal(t, tt.expectedTranslation, result.ErrorCode)
		})
	}
}

func TestCapProvider_TranslateErrorCode(t *testing.T) {
	provider := NewCapProvider("https://cap.example.com", "test-key", &http.Client{})

	tests := []struct {
		code     string
		expected string
	}{
		{"invalid_token", "invalid captcha token"},
		{"expired_token", "captcha token expired"},
		{"already_used", "captcha token already used"},
		{"invalid_solution", "invalid proof-of-work solution"},
		{"missing_token", "missing captcha token"},
		{"custom_error", "verification failed: custom_error"},
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			result := provider.translateErrorCode(tt.code)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCapProvider_Verify_HTTPError(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
	}{
		{"bad request", http.StatusBadRequest},
		{"unauthorized", http.StatusUnauthorized},
		{"forbidden", http.StatusForbidden},
		{"not found", http.StatusNotFound},
		{"internal server error", http.StatusInternalServerError},
		{"service unavailable", http.StatusServiceUnavailable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			client := server.Client()
			provider := NewCapProvider(server.URL, "test-key", client)

			ctx := context.Background()
			result, err := provider.Verify(ctx, "test-token", "")

			require.Error(t, err)
			assert.Nil(t, result)
			assert.Contains(t, err.Error(), "verification endpoint returned status")
		})
	}
}

func TestCapProvider_Verify_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("invalid json"))
	}))
	defer server.Close()

	client := server.Client()
	provider := NewCapProvider(server.URL, "test-key", client)

	ctx := context.Background()
	result, err := provider.Verify(ctx, "test-token", "")

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to decode response")
}

func TestCapProvider_Verify_FailureWithoutErrorCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := capVerifyResponse{
			Success: false,
			Error:   "", // Empty error code
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := server.Client()
	provider := NewCapProvider(server.URL, "test-key", client)

	ctx := context.Background()
	result, err := provider.Verify(ctx, "test-token", "")

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.Success)
	assert.Empty(t, result.ErrorCode, "empty error code should not be translated")
}

func TestCapProvider_Verify_RemoteIPIgnored(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Cap provider doesn't use remoteIP parameter
		// Just verify the request body doesn't include IP
		var reqBody capVerifyRequest
		err := json.NewDecoder(r.Body).Decode(&reqBody)
		require.NoError(t, err)
		assert.Equal(t, "test-token", reqBody.Token)

		response := capVerifyResponse{
			Success: true,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := server.Client()
	provider := NewCapProvider(server.URL, "test-key", client)

	ctx := context.Background()
	result, err := provider.Verify(ctx, "test-token", "1.2.3.4")

	require.NoError(t, err)
	assert.True(t, result.Success)
}

func TestCapProvider_Verify_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// This should not be reached due to context cancellation
		t.Fatal("handler should not be called")
	}))
	defer server.Close()

	client := server.Client()
	provider := NewCapProvider(server.URL, "test-key", client)

	// Create cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	result, err := provider.Verify(ctx, "test-token", "")

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "context canceled")
}
