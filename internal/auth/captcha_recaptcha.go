package auth

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

const recaptchaVerifyURL = "https://www.google.com/recaptcha/api/siteverify"

// ReCaptchaProvider implements CAPTCHA verification using Google reCAPTCHA v3
type ReCaptchaProvider struct {
	secretKey      string
	scoreThreshold float64
	httpClient     *http.Client
}

// NewReCaptchaProvider creates a new reCAPTCHA v3 provider
func NewReCaptchaProvider(secretKey string, scoreThreshold float64, httpClient *http.Client) *ReCaptchaProvider {
	// Default score threshold if not set
	if scoreThreshold <= 0 || scoreThreshold > 1.0 {
		scoreThreshold = 0.5
	}
	return &ReCaptchaProvider{
		secretKey:      secretKey,
		scoreThreshold: scoreThreshold,
		httpClient:     httpClient,
	}
}

// Name returns the provider name
func (p *ReCaptchaProvider) Name() string {
	return "recaptcha_v3"
}

// Verify validates a reCAPTCHA v3 response token
func (p *ReCaptchaProvider) Verify(ctx context.Context, token string, remoteIP string) (*CaptchaResult, error) {
	data := url.Values{}
	data.Set("secret", p.secretKey)
	data.Set("response", token)
	if remoteIP != "" {
		data.Set("remoteip", remoteIP)
	}

	resp, err := postVerify(ctx, p.httpClient, recaptchaVerifyURL, data)
	if err != nil {
		return nil, err
	}

	result := &CaptchaResult{}

	// Parse success field
	if success, ok := resp["success"].(bool); ok {
		result.Success = success
	}

	// Parse score (reCAPTCHA v3 specific)
	if score, ok := resp["score"].(float64); ok {
		result.Score = score
		// Check if score meets threshold
		if result.Success && score < p.scoreThreshold {
			result.Success = false
			result.ErrorCode = fmt.Sprintf("score %.2f below threshold %.2f", score, p.scoreThreshold)
		}
	}

	// Parse action (reCAPTCHA v3 specific)
	if action, ok := resp["action"].(string); ok {
		result.Action = action
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

// translateErrorCode translates reCAPTCHA error codes to user-friendly messages
func (p *ReCaptchaProvider) translateErrorCode(code string) string {
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
	default:
		return fmt.Sprintf("verification failed: %s", code)
	}
}
