package auth

import (
	"context"
	"time"

	"github.com/nimbleflux/fluxbase/internal/config"
	"github.com/nimbleflux/fluxbase/internal/settings"
)

// =============================================================================
// Mock Captcha Provider
// =============================================================================

// MockCaptchaProvider is a mock implementation of CaptchaProvider for testing
type MockCaptchaProvider struct {
	VerifyFunc func(ctx context.Context, token string, remoteIP string) (*CaptchaResult, error)
	NameValue  string
}

// Verify implements CaptchaProvider
func (m *MockCaptchaProvider) Verify(ctx context.Context, token string, remoteIP string) (*CaptchaResult, error) {
	if m.VerifyFunc != nil {
		return m.VerifyFunc(ctx, token, remoteIP)
	}
	return &CaptchaResult{Success: true}, nil
}

// Name implements CaptchaProvider
func (m *MockCaptchaProvider) Name() string {
	if m.NameValue != "" {
		return m.NameValue
	}
	return "mock"
}

// NewMockCaptchaService creates a CaptchaService with a mock provider for testing.
// This is useful for integration tests that need to test handler behavior with captcha.
func NewMockCaptchaService(cfg *config.CaptchaConfig, provider CaptchaProvider) *CaptchaService {
	if cfg == nil || !cfg.Enabled {
		return &CaptchaService{
			config: cfg,
		}
	}

	// Build enabled endpoints map for quick lookup
	enabledEndpoints := make(map[string]bool)
	for _, endpoint := range cfg.Endpoints {
		enabledEndpoints[endpoint] = true
	}

	return &CaptchaService{
		provider:         provider,
		config:           cfg,
		enabledEndpoints: enabledEndpoints,
	}
}

// NewTestCaptchaService creates a CaptchaService configured for testing.
// It creates a service that:
// - Requires captcha for the specified endpoints
// - Returns ErrCaptchaRequired for empty tokens
// - Returns ErrCaptchaInvalid for tokens other than validToken
// - Returns success for validToken
func NewTestCaptchaService(endpoints []string, validToken string) *CaptchaService {
	cfg := &config.CaptchaConfig{
		Enabled:   true,
		Provider:  "mock",
		SiteKey:   "test-site-key",
		SecretKey: "test-secret-key",
		Endpoints: endpoints,
	}

	mockProvider := &MockCaptchaProvider{
		NameValue: "mock",
		VerifyFunc: func(ctx context.Context, token string, remoteIP string) (*CaptchaResult, error) {
			if token == validToken {
				return &CaptchaResult{Success: true}, nil
			}
			return &CaptchaResult{
				Success:   false,
				ErrorCode: "invalid-token",
			}, nil
		},
	}

	return NewMockCaptchaService(cfg, mockProvider)
}

// NewDisabledCaptchaService creates a CaptchaService that is disabled for testing.
func NewDisabledCaptchaService() *CaptchaService {
	return &CaptchaService{
		config: &config.CaptchaConfig{
			Enabled: false,
		},
	}
}

// =============================================================================
// Test Service with Mock Dependencies
// =============================================================================

// NewTestAuthServiceWithSettings creates a Service with pre-configured settings for testing
func NewTestAuthServiceWithSettings(signupEnabled, passwordLoginEnabled bool) *Service {
	cfg := &config.AuthConfig{
		SignupEnabled:    signupEnabled,
		MagicLinkEnabled: true,
	}

	// Create a settings cache that returns our configured values
	cache := settings.NewSettingsCache(nil, time.Hour)

	cache.SetCachedValue("app.auth.signup_enabled", signupEnabled)
	cache.SetCachedValue("app.auth.disable_app_password_login", !passwordLoginEnabled)

	// Create a password hasher with minimal requirements for testing
	passwordHasher := NewPasswordHasherWithConfig(PasswordHasherConfig{
		Cost:          4, // Minimum bcrypt cost for fast tests
		MinLength:     1, // Allow any password for testing
		RequireUpper:  false,
		RequireLower:  false,
		RequireDigit:  false,
		RequireSymbol: false,
	})

	return &Service{
		settingsCache:  cache,
		config:         cfg,
		passwordHasher: passwordHasher,
	}
}
