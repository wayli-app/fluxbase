package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/fluxbase-eu/fluxbase/internal/config"
)

// Common CAPTCHA errors
var (
	ErrCaptchaRequired      = errors.New("captcha verification required")
	ErrCaptchaInvalid       = errors.New("captcha verification failed")
	ErrCaptchaExpired       = errors.New("captcha token expired")
	ErrCaptchaNotConfigured = errors.New("captcha provider not configured")
	ErrCaptchaScoreTooLow   = errors.New("captcha score below threshold")
)

// CaptchaProvider defines the interface for CAPTCHA verification providers
type CaptchaProvider interface {
	// Verify validates a CAPTCHA token and returns the verification result
	// remoteIP is the client's IP address (used for additional validation)
	Verify(ctx context.Context, token string, remoteIP string) (*CaptchaResult, error)

	// Name returns the provider name (hcaptcha, recaptcha_v3, turnstile)
	Name() string
}

// CaptchaResult contains the result of a CAPTCHA verification
type CaptchaResult struct {
	Success   bool      `json:"success"`
	Score     float64   `json:"score,omitempty"`     // Risk score (0.0-1.0, only for reCAPTCHA v3)
	Action    string    `json:"action,omitempty"`    // Action name (only for reCAPTCHA v3)
	Hostname  string    `json:"hostname,omitempty"`  // Hostname where the challenge was solved
	Timestamp time.Time `json:"timestamp,omitempty"` // When the challenge was solved
	ErrorCode string    `json:"error_code,omitempty"`
}

// CaptchaService manages CAPTCHA verification across different providers
type CaptchaService struct {
	provider         CaptchaProvider
	config           *config.CaptchaConfig
	httpClient       *http.Client
	enabledEndpoints map[string]bool
}

// NewCaptchaService creates a new CAPTCHA service based on configuration
func NewCaptchaService(cfg *config.CaptchaConfig) (*CaptchaService, error) {
	if cfg == nil || !cfg.Enabled {
		return &CaptchaService{
			config: cfg,
		}, nil
	}

	// Create HTTP client with timeout
	httpClient := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Create provider based on configuration
	var provider CaptchaProvider
	switch strings.ToLower(cfg.Provider) {
	case "hcaptcha":
		// Validate required fields for hCaptcha
		if cfg.SiteKey == "" || cfg.SecretKey == "" {
			return nil, fmt.Errorf("captcha site_key and secret_key are required for hcaptcha provider")
		}
		provider = NewHCaptchaProvider(cfg.SecretKey, httpClient)
	case "recaptcha_v3", "recaptcha":
		// Validate required fields for reCAPTCHA
		if cfg.SiteKey == "" || cfg.SecretKey == "" {
			return nil, fmt.Errorf("captcha site_key and secret_key are required for recaptcha provider")
		}
		provider = NewReCaptchaProvider(cfg.SecretKey, cfg.ScoreThreshold, httpClient)
	case "turnstile":
		// Validate required fields for Turnstile
		if cfg.SiteKey == "" || cfg.SecretKey == "" {
			return nil, fmt.Errorf("captcha site_key and secret_key are required for turnstile provider")
		}
		provider = NewTurnstileProvider(cfg.SecretKey, httpClient)
	case "cap":
		// Validate required fields for Cap
		if cfg.CapServerURL == "" {
			return nil, fmt.Errorf("cap_server_url is required for cap provider")
		}
		capProvider, err := NewCapProvider(cfg.CapServerURL, cfg.CapAPIKey, httpClient)
		if err != nil {
			return nil, fmt.Errorf("failed to create cap provider: %w", err)
		}
		provider = capProvider
	default:
		return nil, fmt.Errorf("unknown captcha provider: %s", cfg.Provider)
	}

	// Build enabled endpoints map for quick lookup
	enabledEndpoints := make(map[string]bool)
	for _, endpoint := range cfg.Endpoints {
		enabledEndpoints[strings.ToLower(endpoint)] = true
	}

	return &CaptchaService{
		provider:         provider,
		config:           cfg,
		httpClient:       httpClient,
		enabledEndpoints: enabledEndpoints,
	}, nil
}

// IsEnabled returns whether CAPTCHA verification is enabled
func (s *CaptchaService) IsEnabled() bool {
	return s.config != nil && s.config.Enabled && s.provider != nil
}

// IsEnabledForEndpoint checks if CAPTCHA is enabled for a specific endpoint
func (s *CaptchaService) IsEnabledForEndpoint(endpoint string) bool {
	if !s.IsEnabled() {
		return false
	}
	return s.enabledEndpoints[strings.ToLower(endpoint)]
}

// GetSiteKey returns the public site key (safe to expose to frontend)
func (s *CaptchaService) GetSiteKey() string {
	if s.config == nil {
		return ""
	}
	return s.config.SiteKey
}

// GetProvider returns the configured provider name
func (s *CaptchaService) GetProvider() string {
	if s.config == nil {
		return ""
	}
	return s.config.Provider
}

// Verify validates a CAPTCHA token
// Returns nil if verification succeeds, or an error if it fails
func (s *CaptchaService) Verify(ctx context.Context, token string, remoteIP string) error {
	if !s.IsEnabled() {
		return nil // CAPTCHA is disabled, skip verification
	}

	if token == "" {
		return ErrCaptchaRequired
	}

	// Check for test bypass token (for development/testing only)
	// WARNING: Never set TestBypassToken in production environments
	if s.config.TestBypassToken != "" && token == s.config.TestBypassToken {
		return nil // Bypass verification with test token
	}

	result, err := s.provider.Verify(ctx, token, remoteIP)
	if err != nil {
		return fmt.Errorf("captcha verification error: %w", err)
	}

	if !result.Success {
		if result.ErrorCode != "" {
			return fmt.Errorf("%w: %s", ErrCaptchaInvalid, result.ErrorCode)
		}
		return ErrCaptchaInvalid
	}

	return nil
}

// VerifyForEndpoint validates CAPTCHA for a specific endpoint
// Only verifies if the endpoint is configured to require CAPTCHA
func (s *CaptchaService) VerifyForEndpoint(ctx context.Context, endpoint, token, remoteIP string) error {
	if !s.IsEnabledForEndpoint(endpoint) {
		return nil // CAPTCHA not required for this endpoint
	}

	return s.Verify(ctx, token, remoteIP)
}

// CaptchaConfigResponse is the public configuration returned to clients
type CaptchaConfigResponse struct {
	Enabled      bool     `json:"enabled"`
	Provider     string   `json:"provider,omitempty"`
	SiteKey      string   `json:"site_key,omitempty"`
	Endpoints    []string `json:"endpoints,omitempty"`
	CapServerURL string   `json:"cap_server_url,omitempty"` // Cap provider: URL for widget to load from
}

// GetConfig returns the public CAPTCHA configuration for clients
func (s *CaptchaService) GetConfig() CaptchaConfigResponse {
	if s.config == nil || !s.config.Enabled {
		return CaptchaConfigResponse{Enabled: false}
	}

	response := CaptchaConfigResponse{
		Enabled:   true,
		Provider:  s.config.Provider,
		SiteKey:   s.config.SiteKey,
		Endpoints: s.config.Endpoints,
	}

	// Include Cap-specific fields when using Cap provider
	if strings.ToLower(s.config.Provider) == "cap" {
		response.CapServerURL = s.config.CapServerURL
	}

	return response
}

// ReloadFromSettings reloads the captcha configuration from database settings
// Priority order: Config/Env → Database
func (s *CaptchaService) ReloadFromSettings(ctx context.Context, settingsCache *SettingsCache, envConfig *config.SecurityConfig) error {
	// Create a new config to load settings into
	newConfig := &config.CaptchaConfig{}

	// Priority 1: Check if config/env has captcha settings
	// If envConfig has a provider set, it takes precedence
	if envConfig != nil && envConfig.Captcha.Provider != "" {
		// Config takes precedence, use it directly
		newConfig = &envConfig.Captcha
	} else {
		// No config settings, load from database
		if settingsCache != nil {
			newConfig.Enabled = settingsCache.GetBool(ctx, "app.security.captcha.enabled", false)
			newConfig.Provider = settingsCache.GetString(ctx, "app.security.captcha.provider", "hcaptcha")
			newConfig.SiteKey = settingsCache.GetString(ctx, "app.security.captcha.site_key", "")
			newConfig.SecretKey = settingsCache.GetString(ctx, "app.security.captcha.secret_key", "")
			newConfig.CapServerURL = settingsCache.GetString(ctx, "app.security.captcha.cap_server_url", "")
			newConfig.CapAPIKey = settingsCache.GetString(ctx, "app.security.captcha.cap_api_key", "")

			// Load complex types using GetJSON
			var scoreThreshold float64
			if err := settingsCache.GetJSON(ctx, "app.security.captcha.score_threshold", &scoreThreshold); err == nil {
				newConfig.ScoreThreshold = scoreThreshold
			} else {
				newConfig.ScoreThreshold = 0.5 // default
			}

			var endpoints []string
			if err := settingsCache.GetJSON(ctx, "app.security.captcha.endpoints", &endpoints); err == nil {
				newConfig.Endpoints = endpoints
			} else {
				newConfig.Endpoints = []string{"signup", "login", "password_reset", "magic_link"} // defaults
			}
		}
	}

	// Create a new service with the new config
	newService, err := NewCaptchaService(newConfig)
	if err != nil {
		return fmt.Errorf("failed to create captcha service with new settings: %w", err)
	}

	// Update current service fields
	s.config = newService.config
	s.provider = newService.provider
	s.enabledEndpoints = newService.enabledEndpoints
	s.httpClient = newService.httpClient

	return nil
}

// postVerify is a helper function to make HTTP POST requests to CAPTCHA verification endpoints
func postVerify(ctx context.Context, client *http.Client, verifyURL string, data url.Values) (map[string]interface{}, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, verifyURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("verification endpoint returned status %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result, nil
}
