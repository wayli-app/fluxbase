package auth

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

const hcaptchaVerifyURL = "https://api.hcaptcha.com/siteverify"

// HCaptchaProvider implements CAPTCHA verification using hCaptcha
type HCaptchaProvider struct {
	secretKey  string
	httpClient *http.Client
}

// NewHCaptchaProvider creates a new hCaptcha provider
func NewHCaptchaProvider(secretKey string, httpClient *http.Client) *HCaptchaProvider {
	return &HCaptchaProvider{
		secretKey:  secretKey,
		httpClient: httpClient,
	}
}

// Name returns the provider name
func (p *HCaptchaProvider) Name() string {
	return "hcaptcha"
}

// Verify validates an hCaptcha response token
func (p *HCaptchaProvider) Verify(ctx context.Context, token string, remoteIP string) (*CaptchaResult, error) {
	data := url.Values{}
	data.Set("secret", p.secretKey)
	data.Set("response", token)
	if remoteIP != "" {
		data.Set("remoteip", remoteIP)
	}

	resp, err := postVerify(ctx, p.httpClient, hcaptchaVerifyURL, data)
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

	// Parse error codes
	if errorCodes, ok := resp["error-codes"].([]interface{}); ok && len(errorCodes) > 0 {
		if code, ok := errorCodes[0].(string); ok {
			result.ErrorCode = p.translateErrorCode(code)
		}
	}

	return result, nil
}

// translateErrorCode translates hCaptcha error codes to user-friendly messages
func (p *HCaptchaProvider) translateErrorCode(code string) string {
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
	case "invalid-or-already-seen-response":
		return "captcha token already used"
	case "not-using-dummy-passcode":
		return "test mode misconfigured"
	case "sitekey-secret-mismatch":
		return "site key and secret key mismatch"
	default:
		return fmt.Sprintf("verification failed: %s", code)
	}
}
