package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/fluxbase-eu/fluxbase/internal/mcp"
	"github.com/rs/zerolog/log"
)

const (
	httpRequestTimeout = 10 * time.Second
	maxResponseSize    = 1024 * 1024 // 1MB
	httpUserAgent      = "Fluxbase-MCP/1.0"
)

// HttpRequestTool implements the http_request MCP tool for making HTTP requests to external APIs
type HttpRequestTool struct {
	client *http.Client
}

// NewHttpRequestTool creates a new http_request tool
func NewHttpRequestTool() *HttpRequestTool {
	return &HttpRequestTool{
		client: &http.Client{
			Timeout: httpRequestTimeout,
			// Don't follow redirects - security measure against redirect attacks
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

func (t *HttpRequestTool) Name() string {
	return "http_request"
}

func (t *HttpRequestTool) Description() string {
	return "Make an HTTP GET request to an external API. Only whitelisted domains configured for this context are allowed. Returns JSON responses only."
}

func (t *HttpRequestTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"url": map[string]any{
				"type":        "string",
				"description": "The full URL to request (must be HTTPS and on an allowed domain)",
			},
			"method": map[string]any{
				"type":        "string",
				"description": "HTTP method - only GET is currently supported",
				"enum":        []string{"GET"},
				"default":     "GET",
			},
		},
		"required": []string{"url"},
	}
}

func (t *HttpRequestTool) RequiredScopes() []string {
	return []string{mcp.ScopeExecuteHTTP}
}

func (t *HttpRequestTool) Execute(ctx context.Context, args map[string]any, authCtx *mcp.AuthContext) (*mcp.ToolResult, error) {
	// Extract URL
	requestURL, ok := args["url"].(string)
	if !ok || requestURL == "" {
		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.ErrorContent("url is required")},
			IsError: true,
		}, nil
	}

	// Extract method (default to GET)
	method := "GET"
	if m, ok := args["method"].(string); ok && m != "" {
		method = strings.ToUpper(m)
	}

	// Get allowed domains from auth context metadata
	allowedDomains := authCtx.GetMetadataStringSlice(mcp.MetadataKeyHTTPAllowedDomains)

	log.Debug().
		Str("url", requestURL).
		Str("method", method).
		Strs("allowed_domains", allowedDomains).
		Msg("MCP: Executing HTTP request")

	// Execute the request
	result := t.executeRequest(ctx, requestURL, method, allowedDomains)

	// Convert result to JSON
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.ErrorContent(fmt.Sprintf("failed to serialize result: %v", err))},
			IsError: true,
		}, nil
	}

	if !result.Success {
		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.ErrorContent(string(resultJSON))},
			IsError: true,
		}, nil
	}

	return &mcp.ToolResult{
		Content: []mcp.Content{mcp.TextContent(string(resultJSON))},
	}, nil
}

// httpRequestResult represents the result of an HTTP request
type httpRequestResult struct {
	Success        bool        `json:"success"`
	Data           interface{} `json:"data,omitempty"`
	Status         int         `json:"status,omitempty"`
	Error          string      `json:"error,omitempty"`
	AllowedDomains []string    `json:"allowed_domains,omitempty"`
}

// executeRequest performs an HTTP request with security validations
func (t *HttpRequestTool) executeRequest(ctx context.Context, requestURL string, method string, allowedDomains []string) *httpRequestResult {
	// Validate method - only GET is supported
	if method != "GET" {
		return &httpRequestResult{
			Success: false,
			Error:   "Only GET method is supported",
		}
	}

	// Parse URL
	parsedURL, err := url.Parse(requestURL)
	if err != nil {
		return &httpRequestResult{
			Success: false,
			Error:   fmt.Sprintf("Invalid URL: %v", err),
		}
	}

	// Validate URL scheme - must be http or https
	if parsedURL.Scheme != "https" && parsedURL.Scheme != "http" {
		return &httpRequestResult{
			Success: false,
			Error:   "URL must use http or https scheme",
		}
	}

	hostname := parsedURL.Hostname()
	if hostname == "" {
		return &httpRequestResult{
			Success: false,
			Error:   "URL must have a hostname",
		}
	}

	// Require HTTPS for non-localhost domains (security)
	isLocalhost := hostname == "localhost" || hostname == "127.0.0.1" || hostname == "::1"
	if parsedURL.Scheme == "http" && !isLocalhost {
		return &httpRequestResult{
			Success: false,
			Error:   "HTTPS is required for non-localhost domains",
		}
	}

	// Check if URL contains credentials (block for security)
	if parsedURL.User != nil {
		return &httpRequestResult{
			Success: false,
			Error:   "URLs with embedded credentials are not allowed",
		}
	}

	// Validate domain whitelist
	if len(allowedDomains) == 0 {
		return &httpRequestResult{
			Success:        false,
			Error:          "No HTTP domains are allowed. Configure http_allowed_domains in your context.",
			AllowedDomains: []string{},
		}
	}

	if !isDomainAllowed(hostname, allowedDomains) {
		return &httpRequestResult{
			Success:        false,
			Error:          fmt.Sprintf("Domain '%s' is not in the allowed domains list", hostname),
			AllowedDomains: allowedDomains,
		}
	}

	// SSRF protection: Resolve hostname and block private IPs
	if err := validateNotPrivateIP(hostname); err != nil {
		return &httpRequestResult{
			Success: false,
			Error:   fmt.Sprintf("SSRF protection: %v", err),
		}
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, method, requestURL, nil)
	if err != nil {
		return &httpRequestResult{
			Success: false,
			Error:   fmt.Sprintf("Failed to create request: %v", err),
		}
	}

	// Set headers
	req.Header.Set("User-Agent", httpUserAgent)
	req.Header.Set("Accept", "application/json")

	// Execute request
	resp, err := t.client.Do(req)
	if err != nil {
		// Check for timeout
		if ctx.Err() == context.DeadlineExceeded {
			return &httpRequestResult{
				Success: false,
				Error:   "Request timeout (10s)",
			}
		}
		return &httpRequestResult{
			Success: false,
			Error:   fmt.Sprintf("Request failed: %v", err),
		}
	}
	defer func() { _ = resp.Body.Close() }()

	// Check content type is JSON before reading body
	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(strings.ToLower(contentType), "json") {
		return &httpRequestResult{
			Success: false,
			Status:  resp.StatusCode,
			Error:   fmt.Sprintf("Only JSON responses are supported, got: %s", contentType),
		}
	}

	// Check content-length header if present
	if contentLength := resp.ContentLength; contentLength > maxResponseSize {
		return &httpRequestResult{
			Success: false,
			Status:  resp.StatusCode,
			Error:   fmt.Sprintf("Response too large: %d bytes (max %d)", contentLength, maxResponseSize),
		}
	}

	// Read response body with size limit
	limitedReader := io.LimitReader(resp.Body, maxResponseSize+1)
	bodyBytes, err := io.ReadAll(limitedReader)
	if err != nil {
		return &httpRequestResult{
			Success: false,
			Status:  resp.StatusCode,
			Error:   fmt.Sprintf("Failed to read response: %v", err),
		}
	}

	// Check actual size
	if len(bodyBytes) > maxResponseSize {
		return &httpRequestResult{
			Success: false,
			Status:  resp.StatusCode,
			Error:   fmt.Sprintf("Response too large (max %d bytes)", maxResponseSize),
		}
	}

	// Check status code - accept 2xx responses
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Try to parse error from response
		var errorBody string
		if len(bodyBytes) > 0 && len(bodyBytes) <= 1024 {
			errorBody = string(bodyBytes)
		}
		return &httpRequestResult{
			Success: false,
			Status:  resp.StatusCode,
			Error:   fmt.Sprintf("HTTP %d: %s", resp.StatusCode, errorBody),
		}
	}

	// Parse JSON response
	var data interface{}
	if err := json.Unmarshal(bodyBytes, &data); err != nil {
		return &httpRequestResult{
			Success: false,
			Status:  resp.StatusCode,
			Error:   fmt.Sprintf("Failed to parse JSON response: %v", err),
		}
	}

	return &httpRequestResult{
		Success: true,
		Status:  resp.StatusCode,
		Data:    data,
	}
}

// isDomainAllowed checks if a hostname matches any allowed domain pattern
func isDomainAllowed(hostname string, allowedDomains []string) bool {
	hostname = strings.ToLower(hostname)
	for _, allowed := range allowedDomains {
		allowed = strings.ToLower(strings.TrimSpace(allowed))
		if allowed == "" {
			continue
		}

		// Exact match
		if hostname == allowed {
			return true
		}

		// Wildcard support: *.example.com matches sub.example.com and example.com
		if strings.HasPrefix(allowed, "*.") {
			baseDomain := strings.TrimPrefix(allowed, "*.")
			if hostname == baseDomain || strings.HasSuffix(hostname, "."+baseDomain) {
				return true
			}
		}
	}
	return false
}

// validateNotPrivateIP checks if hostname resolves to public IPs only
// This is SSRF protection
func validateNotPrivateIP(hostname string) error {
	// Check for blocked internal hostnames
	lowerHost := strings.ToLower(hostname)

	// Block localhost variants
	if lowerHost == "localhost" || lowerHost == "ip6-localhost" {
		return fmt.Errorf("localhost is not allowed")
	}

	// Block common internal/cloud metadata hostnames
	blockedHostnames := []string{
		"metadata.google.internal",
		"metadata",
		"instance-data",
		"kubernetes.default",
		"kubernetes.default.svc",
		"kubernetes.default.svc.cluster.local",
	}
	for _, blocked := range blockedHostnames {
		if lowerHost == blocked || strings.HasSuffix(lowerHost, "."+blocked) {
			return fmt.Errorf("internal hostname '%s' is not allowed", hostname)
		}
	}

	// Block .local domains
	if strings.HasSuffix(lowerHost, ".local") {
		return fmt.Errorf("local domain '%s' is not allowed", hostname)
	}

	// Resolve and check IPs
	resolveCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	resolver := net.Resolver{}
	ips, err := resolver.LookupIPAddr(resolveCtx, hostname)
	if err != nil {
		return fmt.Errorf("failed to resolve hostname: %w", err)
	}

	for _, ip := range ips {
		if isPrivateIPAddress(ip.IP) {
			return fmt.Errorf("hostname resolves to private IP %s", ip.IP.String())
		}
	}

	return nil
}

// isPrivateIPAddress checks if an IP address is in a private/internal range
func isPrivateIPAddress(ip net.IP) bool {
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

	// Check for private ranges (RFC 1918 and others)
	privateBlocks := []string{
		"10.0.0.0/8",      // Private-Use
		"172.16.0.0/12",   // Private-Use
		"192.168.0.0/16",  // Private-Use
		"169.254.0.0/16",  // Link-Local (AWS metadata endpoint range)
		"127.0.0.0/8",     // Loopback
		"::1/128",         // IPv6 loopback
		"fc00::/7",        // IPv6 unique local
		"fe80::/10",       // IPv6 link local
		"100.64.0.0/10",   // Carrier-Grade NAT
		"192.0.0.0/24",    // IETF Protocol Assignments
		"192.0.2.0/24",    // TEST-NET-1
		"198.51.100.0/24", // TEST-NET-2
		"203.0.113.0/24",  // TEST-NET-3
		"224.0.0.0/4",     // Multicast
		"240.0.0.0/4",     // Reserved
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
