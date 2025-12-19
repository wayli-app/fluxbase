// Package client provides the HTTP client for the Fluxbase API.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/fluxbase-eu/fluxbase/cli/config"
)

// Client is the Fluxbase API client
type Client struct {
	// BaseURL is the Fluxbase server URL
	BaseURL string

	// HTTPClient is the underlying HTTP client
	HTTPClient *http.Client

	// Profile is the active profile
	Profile *config.Profile

	// Config is the CLI configuration
	Config *config.Config

	// ConfigPath is the path to the config file
	ConfigPath string

	// CredentialManager handles credential storage
	CredentialManager *config.CredentialManager

	// Debug enables debug logging
	Debug bool

	// UserAgent to use for requests
	UserAgent string
}

// ClientOption configures the client
type ClientOption func(*Client)

// NewClient creates a new API client
func NewClient(cfg *config.Config, profile *config.Profile, opts ...ClientOption) *Client {
	c := &Client{
		BaseURL: profile.Server,
		Profile: profile,
		Config:  cfg,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		UserAgent: "fluxbase-cli/1.0",
	}

	c.CredentialManager = config.NewCredentialManager(cfg)

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// WithDebug enables debug mode
func WithDebug(debug bool) ClientOption {
	return func(c *Client) {
		c.Debug = debug
	}
}

// WithTimeout sets the HTTP timeout
func WithTimeout(timeout time.Duration) ClientOption {
	return func(c *Client) {
		c.HTTPClient.Timeout = timeout
	}
}

// WithConfigPath sets the config file path for saving updated tokens
func WithConfigPath(path string) ClientOption {
	return func(c *Client) {
		c.ConfigPath = path
	}
}

// Request makes an authenticated API request
func (c *Client) Request(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	return c.RequestWithQuery(ctx, method, path, body, nil)
}

// RequestWithQuery makes an authenticated API request with query parameters
func (c *Client) RequestWithQuery(ctx context.Context, method, path string, body interface{}, query url.Values) (*http.Response, error) {
	// Build URL
	u, err := url.Parse(c.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}
	u.Path = strings.TrimSuffix(u.Path, "/") + path
	if query != nil {
		u.RawQuery = query.Encode()
	}

	// Prepare body
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, method, u.String(), bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.UserAgent)

	// Add authentication
	if err := c.addAuth(req); err != nil {
		return nil, err
	}

	// Execute request
	if c.Debug {
		fmt.Printf("DEBUG: %s %s\n", method, u.String())
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	return resp, nil
}

// addAuth adds authentication to the request
func (c *Client) addAuth(req *http.Request) error {
	creds, err := c.CredentialManager.GetCredentials(c.Profile.Name)
	if err != nil {
		return fmt.Errorf("failed to get credentials: %w", err)
	}

	if creds == nil {
		return fmt.Errorf("not authenticated - run 'fluxbase auth login'")
	}

	// Check if token needs refresh
	if creds.AccessToken != "" && creds.ExpiresAt > 0 {
		// Refresh 60 seconds before expiry
		if time.Now().Unix() >= creds.ExpiresAt-60 {
			if creds.RefreshToken != "" {
				if err := c.refreshToken(creds); err != nil {
					// If refresh fails, try with current token anyway
					if c.Debug {
						fmt.Printf("DEBUG: Token refresh failed: %v\n", err)
					}
				}
			}
		}
		req.Header.Set("Authorization", "Bearer "+creds.AccessToken)
	} else if creds.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+creds.APIKey)
	} else {
		return fmt.Errorf("no valid credentials - run 'fluxbase auth login'")
	}

	return nil
}

// refreshToken refreshes the access token
func (c *Client) refreshToken(creds *config.Credentials) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Build refresh request
	u, err := url.Parse(c.BaseURL)
	if err != nil {
		return err
	}
	u.Path = "/api/v1/auth/refresh"

	body := map[string]string{
		"refresh_token": creds.RefreshToken,
	}
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewReader(data))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", c.UserAgent)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("refresh failed with status %d", resp.StatusCode)
	}

	var result struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int64  `json:"expires_in"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	// Update credentials
	creds.AccessToken = result.AccessToken
	if result.RefreshToken != "" {
		creds.RefreshToken = result.RefreshToken
	}
	if result.ExpiresIn > 0 {
		creds.ExpiresAt = time.Now().Unix() + result.ExpiresIn
	}

	// Save updated credentials
	useKeychain := c.Profile.CredentialStore == "keychain"
	if err := c.CredentialManager.SaveCredentials(c.Profile.Name, creds, useKeychain); err != nil {
		return err
	}

	// Save config to persist changes
	if c.ConfigPath != "" {
		if err := c.Config.Save(c.ConfigPath); err != nil {
			return err
		}
	}

	return nil
}

// Get performs a GET request
func (c *Client) Get(ctx context.Context, path string, query url.Values) (*http.Response, error) {
	return c.RequestWithQuery(ctx, http.MethodGet, path, nil, query)
}

// Post performs a POST request
func (c *Client) Post(ctx context.Context, path string, body interface{}) (*http.Response, error) {
	return c.Request(ctx, http.MethodPost, path, body)
}

// Put performs a PUT request
func (c *Client) Put(ctx context.Context, path string, body interface{}) (*http.Response, error) {
	return c.Request(ctx, http.MethodPut, path, body)
}

// Patch performs a PATCH request
func (c *Client) Patch(ctx context.Context, path string, body interface{}) (*http.Response, error) {
	return c.Request(ctx, http.MethodPatch, path, body)
}

// Delete performs a DELETE request
func (c *Client) Delete(ctx context.Context, path string) (*http.Response, error) {
	return c.Request(ctx, http.MethodDelete, path, nil)
}

// DoGet performs a GET request and decodes the response into target
func (c *Client) DoGet(ctx context.Context, path string, query url.Values, target interface{}) error {
	resp, err := c.Get(ctx, path, query)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	return decodeBody(resp, target)
}

// DoPost performs a POST request and decodes the response into target
func (c *Client) DoPost(ctx context.Context, path string, body interface{}, target interface{}) error {
	resp, err := c.Post(ctx, path, body)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	return decodeBody(resp, target)
}

// DoPut performs a PUT request and decodes the response into target
func (c *Client) DoPut(ctx context.Context, path string, body interface{}, target interface{}) error {
	resp, err := c.Put(ctx, path, body)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	return decodeBody(resp, target)
}

// DoDelete performs a DELETE request
func (c *Client) DoDelete(ctx context.Context, path string) error {
	resp, err := c.Delete(ctx, path)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 400 {
		return parseErrorBody(resp)
	}
	return nil
}

// DoPatch performs a PATCH request and decodes the response into target
func (c *Client) DoPatch(ctx context.Context, path string, body interface{}, target interface{}) error {
	resp, err := c.Patch(ctx, path, body)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	return decodeBody(resp, target)
}

// DoRequestWithQuery performs a request with query parameters and handles the response
func (c *Client) DoRequestWithQuery(ctx context.Context, method string, path string, body interface{}, query url.Values) error {
	resp, err := c.RequestWithQuery(ctx, method, path, body, query)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 400 {
		return parseErrorBody(resp)
	}
	return nil
}

// decodeBody decodes the response body into target
func decodeBody(resp *http.Response, target interface{}) error {
	if resp.StatusCode >= 400 {
		return parseErrorBody(resp)
	}
	if target == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(target)
}

// parseErrorBody parses an error response body
func parseErrorBody(resp *http.Response) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &APIError{
			StatusCode: resp.StatusCode,
			Message:    fmt.Sprintf("failed to read error response: %v", err),
		}
	}

	var apiErr APIError
	if err := json.Unmarshal(body, &apiErr); err != nil {
		return &APIError{
			StatusCode: resp.StatusCode,
			Message:    string(body),
		}
	}

	apiErr.StatusCode = resp.StatusCode
	return &apiErr
}

// APIError represents an API error response
type APIError struct {
	StatusCode int    `json:"-"`
	Message    string `json:"message"`
	Error_     string `json:"error"`
	Code       string `json:"code"`
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	if e.Error_ != "" {
		return e.Error_
	}
	return fmt.Sprintf("API error with status %d", e.StatusCode)
}

// ParseError parses an error response
func ParseError(resp *http.Response) error {
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &APIError{
			StatusCode: resp.StatusCode,
			Message:    fmt.Sprintf("failed to read error response: %v", err),
		}
	}

	var apiErr APIError
	if err := json.Unmarshal(body, &apiErr); err != nil {
		return &APIError{
			StatusCode: resp.StatusCode,
			Message:    string(body),
		}
	}

	apiErr.StatusCode = resp.StatusCode
	return &apiErr
}

// DecodeResponse decodes a successful response into the target
func DecodeResponse(resp *http.Response, target interface{}) error {
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		return ParseError(resp)
	}

	if target == nil {
		return nil
	}

	return json.NewDecoder(resp.Body).Decode(target)
}
