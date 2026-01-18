package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
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

// NewCapProvider creates a new Cap provider with SSRF protection.
// Returns an error if the server URL is invalid or points to a private/internal address.
func NewCapProvider(serverURL, apiKey string, httpClient *http.Client) (*CapProvider, error) {
	cleanURL := strings.TrimSuffix(serverURL, "/")

	// Validate URL to prevent SSRF attacks
	if err := validateCapServerURL(cleanURL); err != nil {
		return nil, fmt.Errorf("invalid Cap server URL: %w", err)
	}

	return &CapProvider{
		serverURL:  cleanURL,
		apiKey:     apiKey,
		httpClient: httpClient,
	}, nil
}

// validateCapServerURL validates that a Cap server URL is safe to use.
// This prevents SSRF attacks by blocking internal/private IP addresses.
func validateCapServerURL(serverURL string) error {
	parsedURL, err := url.Parse(serverURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	// Only allow HTTPS in production (HTTP allowed for local development)
	if parsedURL.Scheme != "https" && parsedURL.Scheme != "http" {
		return fmt.Errorf("URL scheme must be http or https, got: %s", parsedURL.Scheme)
	}

	hostname := parsedURL.Hostname()
	if hostname == "" {
		return fmt.Errorf("URL must have a hostname")
	}

	// Check for localhost variants
	lowerHost := strings.ToLower(hostname)
	if lowerHost == "localhost" || lowerHost == "ip6-localhost" {
		return fmt.Errorf("localhost URLs are not allowed for Cap server")
	}

	// Check for common internal/cloud metadata hostnames
	blockedHostnames := []string{
		"metadata.google.internal",
		"metadata",
		"instance-data",
		"169.254.169.254", // AWS/GCP metadata endpoint
		"kubernetes.default",
		"kubernetes.default.svc",
	}
	for _, blocked := range blockedHostnames {
		if lowerHost == blocked || strings.HasSuffix(lowerHost, "."+blocked) {
			return fmt.Errorf("internal hostname '%s' is not allowed", hostname)
		}
	}

	// Resolve the hostname and check if it resolves to a private IP
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	resolver := net.Resolver{}
	ips, err := resolver.LookupIPAddr(ctx, hostname)
	if err != nil {
		// If DNS lookup fails, allow it (might be a valid external domain not resolvable at config time)
		// The actual request will fail later if the hostname is truly invalid
		return nil
	}

	for _, ip := range ips {
		if isPrivateIPForCap(ip.IP) {
			return fmt.Errorf("URL resolves to private IP address %s which is not allowed", ip.IP.String())
		}
	}

	return nil
}

// isPrivateIPForCap checks if an IP address is in a private range
func isPrivateIPForCap(ip net.IP) bool {
	if ip == nil {
		return false
	}

	// Check for loopback
	if ip.IsLoopback() {
		return true
	}

	// Check for link-local
	if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}

	// Check for private ranges
	privateBlocks := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"169.254.0.0/16", // AWS metadata endpoint range
		"127.0.0.0/8",    // Loopback
		"::1/128",        // IPv6 loopback
		"fc00::/7",       // IPv6 unique local
		"fe80::/10",      // IPv6 link local
	}

	for _, block := range privateBlocks {
		_, cidr, err := net.ParseCIDR(block)
		if err != nil {
			continue
		}
		if cidr.Contains(ip) {
			return true
		}
	}

	return false
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
