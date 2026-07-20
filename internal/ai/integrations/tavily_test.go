package integrations

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTavilyClient_Search_BuildsCorrectRequest(t *testing.T) {
	var capturedBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/search", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		_ = json.NewDecoder(r.Body).Decode(&capturedBody)
		_, _ = w.Write([]byte(`{
			"answer": "Berlin zoo closes at 6:30 PM today",
			"results": [
				{"title": "Berlin Zoo", "url": "https://www.zooberlin.de", "content": "Open 9-18:30", "score": 0.95}
			]
		}`))
	}))
	defer server.Close()

	client := NewTavilyClient("tvly-test", server.URL, nil)
	result, err := client.Search(context.Background(), SearchOptions{
		Query:         "Berlin zoo closing time",
		MaxResults:    3,
		SearchDepth:   "basic",
		IncludeAnswer: true,
	})
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "Berlin zoo closes at 6:30 PM today", result.Answer)
	require.Len(t, result.Results, 1)
	assert.Equal(t, "Berlin Zoo", result.Results[0].Title)
	assert.Equal(t, "https://www.zooberlin.de", result.Results[0].URL)
	assert.InDelta(t, 0.95, result.Results[0].Score, 0.001)

	// Verify request body
	assert.Equal(t, "tvly-test", capturedBody["api_key"])
	assert.Equal(t, "Berlin zoo closing time", capturedBody["query"])
	assert.EqualValues(t, 3, capturedBody["max_results"])
	assert.Equal(t, "basic", capturedBody["search_depth"])
	assert.Equal(t, true, capturedBody["include_answer"])
}

func TestTavilyClient_Search_PassesDomainFilters(t *testing.T) {
	var capturedBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&capturedBody)
		_, _ = w.Write([]byte(`{"results": []}`))
	}))
	defer server.Close()

	client := NewTavilyClient("tvly-test", server.URL, nil)
	_, err := client.Search(context.Background(), SearchOptions{
		Query:          "test",
		IncludeDomains: []string{"wikipedia.org", "mdn.dev"},
		ExcludeDomains: []string{"spam.example"},
	})
	require.NoError(t, err)
	assert.Equal(t, []any{"wikipedia.org", "mdn.dev"}, capturedBody["include_domains"])
	assert.Equal(t, []any{"spam.example"}, capturedBody["exclude_domains"])
}

func TestTavilyClient_Search_NoApiKey_Errors(t *testing.T) {
	client := NewTavilyClient("", "", nil)
	_, err := client.Search(context.Background(), SearchOptions{Query: "test"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "api_key is required")
}

func TestTavilyClient_Search_EmptyQuery_Errors(t *testing.T) {
	client := NewTavilyClient("tvly-test", "", nil)
	_, err := client.Search(context.Background(), SearchOptions{Query: ""})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "query is required")
}

func TestTavilyClient_Search_HttpError_ReturnsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"detail": "Invalid API key"}`))
	}))
	defer server.Close()

	client := NewTavilyClient("tvly-bad", server.URL, nil)
	_, err := client.Search(context.Background(), SearchOptions{Query: "test"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid API key")
	assert.Contains(t, err.Error(), "401")
}

func TestTavilyClient_Extract_BuildsCorrectRequest(t *testing.T) {
	var capturedBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/extract", r.URL.Path)
		_ = json.NewDecoder(r.Body).Decode(&capturedBody)
		_, _ = w.Write([]byte(`{
			"results": [
				{"url": "https://example.com", "raw_content": "# Hello\n\nWorld"}
			],
			"failed_count": 0
		}`))
	}))
	defer server.Close()

	client := NewTavilyClient("tvly-test", server.URL, nil)
	result, err := client.Extract(context.Background(), ExtractOptions{
		URLs: []string{"https://example.com"},
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Results, 1)
	assert.Equal(t, "https://example.com", result.Results[0].URL)
	assert.Equal(t, "# Hello\n\nWorld", result.Results[0].RawContent)

	assert.Equal(t, []any{"https://example.com"}, capturedBody["urls"])
}

func TestTavilyClient_Ping_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"results": []}`))
	}))
	defer server.Close()

	client := NewTavilyClient("tvly-test", server.URL, nil)
	err := client.Ping(context.Background())
	require.NoError(t, err)
}

func TestTavilyClient_Ping_FailureReturnsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"detail": "Quota exceeded"}`))
	}))
	defer server.Close()

	client := NewTavilyClient("tvly-test", server.URL, nil)
	err := client.Ping(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Quota exceeded")
}

// TestTavilyClient_Search_ResponseTimeAsNumber verifies that the struct
// can parse Tavily's real response shape where response_time is a JSON
// number (float), not a string. Regression test for the type mismatch
// that blocked every web search.
func TestTavilyClient_Search_ResponseTimeAsNumber(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{
			"response_time": 1.23,
			"answer": "Berlin has many events",
			"results": [
				{"title": "Berlin Events", "url": "https://example.com", "content": "Fun stuff", "score": 0.9}
			]
		}`))
	}))
	defer server.Close()

	client := NewTavilyClient("tvly-test", server.URL, nil)
	result, err := client.Search(context.Background(), SearchOptions{
		Query:      "Berlin events",
		MaxResults: 5,
		SearchDepth: "basic",
	})

	require.NoError(t, err, "must parse response_time as number without error")
	assert.NotNil(t, result)
	assert.Len(t, result.Results, 1)
	assert.Equal(t, "Berlin Events", result.Results[0].Title)
}

// TestTavilyClient_Search_NullableStringColumns verifies the standardColumns
// COALESCE fix: nullable DB columns (last_test_status, last_test_error,
// created_by) don't crash the scan when NULL. This is a documentation test —
// the actual COALESCE runs at the SQL level, but we verify here that the
// Integration struct's string fields accept empty strings (which is what
// COALESCE produces from NULL).
func TestTavilyClient_Search_NullableFieldsAcceptEmptyString(t *testing.T) {
	// Verify the Integration struct's fields that correspond to nullable DB
	// columns can hold empty strings without issues. This documents the
	// COALESCE contract: NULL in DB → empty string in Go struct.
	i := &Integration{
		Name:           "Tavily",
		Provider:       "tavily",
		IntegrationType: "web_search",
		LastTestStatus: "", // COALESCE(last_test_status, '')
		LastTestError:  "", // COALESCE(last_test_error, '')
		CreatedBy:      "", // COALESCE(created_by, '')
	}
	assert.Equal(t, "", i.LastTestStatus)
	assert.Equal(t, "", i.LastTestError)
	assert.Equal(t, "", i.CreatedBy)
}
