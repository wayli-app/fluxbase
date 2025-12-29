package auth

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

const turnstileVerifyURL = "https://challenges.cloudflare.com/turnstile/v0/siteverify"

// TurnstileProvider implements CAPTCHA verification using Cloudflare Turnstile
type TurnstileProvider struct {
	secretKey  string
	httpClient *http.Client
}

// NewTurnstileProvider creates a new Cloudflare Turnstile provider
func NewTurnstileProvider(secretKey string, httpClient *http.Client) *TurnstileProvider {
	return &TurnstileProvider{
		secretKey:  secretKey,
		httpClient: httpClient,
	}
}

// Name returns the provider name
func (p *TurnstileProvider) Name() string {
	return "turnstile"
}

// Verify validates a Cloudflare Turnstile response token
func (p *TurnstileProvider) Verify(ctx context.Context, token string, remoteIP string) (*CaptchaResult, error) {
	data := url.Values{}
	data.Set("secret", p.secretKey)
	data.Set("response", token)
	if remoteIP != "" {
		data.Set("remoteip", remoteIP)
	}

	resp, err := postVerify(ctx, p.httpClient, turnstileVerifyURL, data)
	if err != nil {
		return nil, err
	}

	result := &CaptchaResult{}

	// Parse success field
	if success, ok := resp["success"].(bool); ok {
		result.Success = success
	}

	// Parse hostname
	if hostname, ok := resp["hostname"].(string); ok {
		result.Hostname = hostname
	}

	// Parse challenge timestamp
	if ts, ok := resp["challenge_ts"].(string); ok {
		if parsed, err := time.Parse(time.RFC3339, ts); err == nil {
			result.Timestamp = parsed
		}
	}

	// Parse action (Turnstile can include custom action name)
	if action, ok := resp["action"].(string); ok {
		result.Action = action
	}

	// Parse error codes
	if errorCodes, ok := resp["error-codes"].([]interface{}); ok && len(errorCodes) > 0 {
		if code, ok := errorCodes[0].(string); ok {
			result.ErrorCode = p.translateErrorCode(code)
		}
	}

	return result, nil
}

// translateErrorCode translates Turnstile error codes to user-friendly messages
func (p *TurnstileProvider) translateErrorCode(code string) string {
	switch code {
	case "missing-input-secret":
		return "missing secret key"
	case "invalid-input-secret":
		return "invalid secret key"
	case "missing-input-response":
		return "missing captcha response"
	case "invalid-input-response":
		return "invalid captcha response"
	case "bad-request":
		return "bad request"
	case "timeout-or-duplicate":
		return "captcha token expired or already used"
	case "internal-error":
		return "internal server error"
	default:
		return fmt.Sprintf("verification failed: %s", code)
	}
}
