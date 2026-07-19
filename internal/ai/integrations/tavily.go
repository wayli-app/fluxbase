package integrations

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// DefaultTavilyBaseURL is the production Tavily API endpoint. Override
// via EnvConfig.TavilyBaseURL (env: FLUXBASE_AI_TAVILY_BASE_URL) for
// testing or self-hosted mirrors.
const DefaultTavilyBaseURL = "https://api.tavily.com"

// TavilyClient calls the Tavily search and extract endpoints.
//
// API reference: https://docs.tavily.com/documentation/api-reference/endpoint
// (docs.tavily.com is the canonical source; the SDK here is a thin
// net/http wrapper to avoid adding a third-party Go dependency.)
type TavilyClient struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// NewTavilyClient constructs a TavilyClient. If baseURL is empty,
// DefaultTavilyBaseURL is used. The HTTP client defaults to a 30s timeout
// if nil — override for custom retry / TLS configs.
func NewTavilyClient(apiKey, baseURL string, httpClient *http.Client) *TavilyClient {
	if baseURL == "" {
		baseURL = DefaultTavilyBaseURL
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	return &TavilyClient{
		apiKey:     apiKey,
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: httpClient,
	}
}

// SearchOptions controls a single Tavily search call.
type SearchOptions struct {
	Query          string   // required
	MaxResults     int      // default 5; Tavily caps at ~20
	SearchDepth    string   // "basic" (default, faster) | "advanced" (slower, more thorough)
	IncludeDomains []string // optional allowlist
	ExcludeDomains []string // optional blocklist
	IncludeAnswer  bool     // ask Tavily to synthesize a short answer
	IncludeRaw     bool     // include raw HTML in results (large; off by default)
}

// SearchResult is the parsed Tavily search response.
type SearchResult struct {
	Query        string       `json:"query"`
	Answer       string       `json:"answer,omitempty"` // populated when IncludeAnswer=true
	Results      []SearchItem `json:"results"`
	ResponseTime float64      `json:"response_time"` // seconds
}

// SearchItem is one result in SearchResult.Results.
type SearchItem struct {
	Title   string  `json:"title"`
	URL     string  `json:"url"`
	Content string  `json:"content"` // cleaned markdown
	Raw     string  `json:"raw,omitempty"`
	Score   float64 `json:"score"` // 0..1 relevance
}

// tavilySearchResponse is the raw JSON shape from Tavily's /search endpoint.
// We don't surface every field — only what the agent needs.
type tavilySearchResponse struct {
	QueryResponse string `json:"response_time"`
	Answer        string `json:"answer,omitempty"`
	Results       []struct {
		Title   string  `json:"title"`
		URL     string  `json:"url"`
		Content string  `json:"content"`
		Raw     string  `json:"raw_content,omitempty"`
		Score   float64 `json:"score"`
	} `json:"results"`
}

// Search calls Tavily /search.
func (c *TavilyClient) Search(ctx context.Context, opts SearchOptions) (*SearchResult, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("tavily: api_key is required")
	}
	if opts.Query == "" {
		return nil, fmt.Errorf("tavily: query is required")
	}
	if opts.MaxResults <= 0 {
		opts.MaxResults = 5
	}
	if opts.SearchDepth == "" {
		opts.SearchDepth = "basic"
	}

	body := map[string]any{
		"api_key":             c.apiKey,
		"query":               opts.Query,
		"max_results":         opts.MaxResults,
		"search_depth":        opts.SearchDepth,
		"include_answer":      opts.IncludeAnswer,
		"include_raw_content": opts.IncludeRaw,
	}
	if len(opts.IncludeDomains) > 0 {
		body["include_domains"] = opts.IncludeDomains
	}
	if len(opts.ExcludeDomains) > 0 {
		body["exclude_domains"] = opts.ExcludeDomains
	}

	respBody, err := c.postJSON(ctx, "/search", body)
	if err != nil {
		return nil, err
	}

	var raw tavilySearchResponse
	if err := json.Unmarshal(respBody, &raw); err != nil {
		return nil, fmt.Errorf("tavily: failed to parse search response: %w", err)
	}

	out := &SearchResult{
		Answer: raw.Answer,
	}
	out.Results = make([]SearchItem, len(raw.Results))
	for i, r := range raw.Results {
		out.Results[i] = SearchItem{
			Title:   r.Title,
			URL:     r.URL,
			Content: r.Content,
			Raw:     r.Raw,
			Score:   r.Score,
		}
	}
	return out, nil
}

// ExtractOptions controls the /extract call.
type ExtractOptions struct {
	URLs    []string // required, 1..10 URLs per Tavily docs
	Include []string // optional section selector
	Exclude []string // optional section selector
}

// ExtractResult is the parsed Tavily extract response.
type ExtractResult struct {
	Results []struct {
		URL        string `json:"url"`
		RawContent string `json:"raw_content"`
		Failed     bool   `json:"failed,omitempty"`
		Error      string `json:"error,omitempty"`
	} `json:"results"`
	FailedCount int `json:"failed_count"`
}

// Extract calls Tavily /extract. Useful when the agent has a specific URL
// (e.g., the top search hit) and wants full page content as markdown.
func (c *TavilyClient) Extract(ctx context.Context, opts ExtractOptions) (*ExtractResult, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("tavily: api_key is required")
	}
	if len(opts.URLs) == 0 {
		return nil, fmt.Errorf("tavily: at least one URL is required")
	}

	body := map[string]any{
		"api_key": c.apiKey,
		"urls":    opts.URLs,
	}
	if len(opts.Include) > 0 {
		body["include"] = opts.Include
	}
	if len(opts.Exclude) > 0 {
		body["exclude"] = opts.Exclude
	}

	respBody, err := c.postJSON(ctx, "/extract", body)
	if err != nil {
		return nil, err
	}

	var out ExtractResult
	if err := json.Unmarshal(respBody, &out); err != nil {
		return nil, fmt.Errorf("tavily: failed to parse extract response: %w", err)
	}
	return &out, nil
}

// Ping verifies the API key by running a minimal /search call. Used by
// the test-connection endpoint in the admin UI. Returns nil on success.
func (c *TavilyClient) Ping(ctx context.Context) error {
	_, err := c.Search(ctx, SearchOptions{
		Query:      "test",
		MaxResults: 1,
	})
	return err
}

// postJSON is the shared HTTP helper. Centralizes header/timeout/error
// handling so Search and Extract stay focused on their own shape parsing.
func (c *TavilyClient) postJSON(ctx context.Context, path string, body any) ([]byte, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("tavily: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("tavily: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("tavily: HTTP call: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("tavily: read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Try to extract a Tavily error message; fall back to raw body
		var errResp struct {
			Detail  string `json:"detail"`
			Message string `json:"message"`
		}
		_ = json.Unmarshal(respBody, &errResp)
		if errResp.Detail != "" {
			return nil, fmt.Errorf("tavily: %s (HTTP %d)", errResp.Detail, resp.StatusCode)
		}
		if errResp.Message != "" {
			return nil, fmt.Errorf("tavily: %s (HTTP %d)", errResp.Message, resp.StatusCode)
		}
		return nil, fmt.Errorf("tavily: HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}
