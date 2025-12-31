package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

const capVerifyPath = "/api/token/validate"

// CapProvider implements CAPTCHA verification using Cap (self-hosted proof-of-work CAPTCHA)
// Cap is a privacy-first, self-hosted CAPTCHA solution that uses proof-of-work challenges
// instead of visual puzzles. See: https://capjs.js.org/
type CapProvider struct {
	serverURL  string
	apiKey     string
	httpClient *http.Client
}

// NewCapProvider creates a new Cap provider
func NewCapProvider(serverURL, apiKey string, httpClient *http.Client) *CapProvider {
	return &CapProvider{
		serverURL:  strings.TrimSuffix(serverURL, "/"),
		apiKey:     apiKey,
		httpClient: httpClient,
	}
}

// Name returns the provider name
func (p *CapProvider) Name() string {
	return "cap"
}

// capVerifyRequest represents the request body for Cap token validation
type capVerifyRequest struct {
	Token string `json:"token"`
}

// capVerifyResponse represents the response from Cap token validation
type capVerifyResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// Verify validates a Cap response token
func (p *CapProvider) Verify(ctx context.Context, token string, remoteIP string) (*CaptchaResult, error) {
	// Build request body
	reqBody := capVerifyRequest{Token: token}
	data, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create request
	verifyURL := p.serverURL + capVerifyPath
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, verifyURL, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bot "+p.apiKey)
	}

	// Send request
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Check HTTP status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("verification endpoint returned status %d", resp.StatusCode)
	}

	// Parse response
	var capResp capVerifyResponse
	if err := json.NewDecoder(resp.Body).Decode(&capResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	result := &CaptchaResult{
		Success: capResp.Success,
	}

	// Set error code if verification failed
	if !capResp.Success && capResp.Error != "" {
		result.ErrorCode = p.translateErrorCode(capResp.Error)
	}

	return result, nil
}

// translateErrorCode translates Cap error codes to user-friendly messages
func (p *CapProvider) translateErrorCode(code string) string {
	switch code {
	case "invalid_token":
		return "invalid captcha token"
	case "expired_token":
		return "captcha token expired"
	case "already_used":
		return "captcha token already used"
	case "invalid_solution":
		return "invalid proof-of-work solution"
	case "missing_token":
		return "missing captcha token"
	default:
		return fmt.Sprintf("verification failed: %s", code)
	}
}
