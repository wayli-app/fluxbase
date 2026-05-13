package webhook

import (
	"context"
	"crypto/hmac"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// isPrivateIP checks if an IP address is in a private range
func isPrivateIP(ip net.IP) bool {
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

	// Check for private ranges (RFC 1918)
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

// validateWebhookHeaders validates that custom webhook headers are safe
// This prevents HTTP header injection attacks
func validateWebhookHeaders(headers map[string]string) error {
	// Blocklist of headers that shouldn't be overridden
	blockedHeaders := map[string]bool{
		"content-length":      true,
		"host":                true,
		"transfer-encoding":   true,
		"connection":          true,
		"keep-alive":          true,
		"proxy-authenticate":  true,
		"proxy-authorization": true,
		"te":                  true,
		"trailers":            true,
		"upgrade":             true,
	}

	for key, value := range headers {
		lowerKey := strings.ToLower(key)

		// Check for blocked headers
		if blockedHeaders[lowerKey] {
			return fmt.Errorf("header '%s' is not allowed to be overridden", key)
		}

		// Check for CRLF injection in header name
		if strings.ContainsAny(key, "\r\n") {
			return fmt.Errorf("header name '%s' contains invalid characters", key)
		}

		// Check for CRLF injection in header value
		if strings.ContainsAny(value, "\r\n") {
			return fmt.Errorf("header value for '%s' contains invalid characters", key)
		}

		// Limit header value length
		if len(value) > 8192 {
			return fmt.Errorf("header value for '%s' exceeds maximum length of 8192 bytes", key)
		}
	}

	return nil
}

// validateWebhookURL validates that a webhook URL is safe to call
// This prevents SSRF attacks by blocking internal/private IP addresses
func validateWebhookURL(webhookURL string) error {
	// Parse the URL
	parsedURL, err := url.Parse(webhookURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	// Only allow HTTP and HTTPS schemes
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("URL scheme must be http or https, got: %s", parsedURL.Scheme)
	}

	// Get hostname
	hostname := parsedURL.Hostname()
	if hostname == "" {
		return fmt.Errorf("URL must have a hostname")
	}

	// Check for localhost variants
	lowerHost := strings.ToLower(hostname)
	if lowerHost == "localhost" || lowerHost == "ip6-localhost" {
		return fmt.Errorf("localhost URLs are not allowed")
	}

	// Check for common internal hostnames
	blockedHostnames := []string{
		"metadata.google.internal",
		"metadata",
		"instance-data",
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
		// If DNS lookup fails, we can't verify - block it to be safe
		return fmt.Errorf("failed to resolve hostname: %w", err)
	}

	for _, ip := range ips {
		if isPrivateIP(ip.IP) {
			return fmt.Errorf("URL resolves to private IP address %s which is not allowed", ip.IP.String())
		}
	}

	return nil
}

// WebhookSignature represents a parsed webhook signature header
type WebhookSignature struct {
	Timestamp  int64
	Signatures []string
}

// ParseWebhookSignature parses an X-Fluxbase-Signature header
// Format: t=timestamp,v1=signature[,v1=signature2...]
// Example: t=1234567890,v1=abc123def456
func ParseWebhookSignature(header string) (*WebhookSignature, error) {
	if header == "" {
		return nil, fmt.Errorf("empty signature header")
	}

	sig := &WebhookSignature{}

	parts := strings.Split(header, ",")
	for _, part := range parts {
		kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(kv) != 2 {
			continue
		}

		key := strings.TrimSpace(kv[0])
		value := strings.TrimSpace(kv[1])

		switch key {
		case "t":
			ts, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid timestamp: %w", err)
			}
			sig.Timestamp = ts
		case "v1":
			sig.Signatures = append(sig.Signatures, value)
		}
	}

	if sig.Timestamp == 0 {
		return nil, fmt.Errorf("missing timestamp in signature")
	}
	if len(sig.Signatures) == 0 {
		return nil, fmt.Errorf("missing signature value")
	}

	return sig, nil
}

// VerifyWebhookSignature verifies a webhook signature
// Parameters:
//   - payload: the raw request body
//   - header: the X-Fluxbase-Signature header value
//   - secret: the webhook secret
//   - tolerance: maximum age of the signature (recommended: 5 minutes)
//
// Returns nil if signature is valid, error otherwise
func VerifyWebhookSignature(payload []byte, header, secret string, tolerance time.Duration) error {
	sig, err := ParseWebhookSignature(header)
	if err != nil {
		return fmt.Errorf("failed to parse signature: %w", err)
	}

	// Check timestamp is not too old (replay protection)
	signedAt := time.Unix(sig.Timestamp, 0)
	if time.Since(signedAt) > tolerance {
		return fmt.Errorf("signature timestamp too old (signed at %v, tolerance %v)", signedAt, tolerance)
	}

	// Check timestamp is not in the future (clock skew protection)
	if signedAt.After(time.Now().Add(tolerance)) {
		return fmt.Errorf("signature timestamp in the future")
	}

	// Compute expected signature
	expectedSig := generateTimestampedSignature(payload, secret, sig.Timestamp)

	// Compare signatures (constant time comparison to prevent timing attacks)
	for _, providedSig := range sig.Signatures {
		if hmac.Equal([]byte(expectedSig), []byte(providedSig)) {
			return nil
		}
	}

	return fmt.Errorf("signature mismatch")
}
