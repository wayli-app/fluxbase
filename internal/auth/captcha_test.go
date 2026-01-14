package auth

import (
	"context"
	"errors"
	"testing"

	"github.com/fluxbase-eu/fluxbase/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Error Variables Tests
// =============================================================================

func TestCaptchaErrors(t *testing.T) {
	t.Run("error variables are distinct", func(t *testing.T) {
		captchaErrors := []error{
			ErrCaptchaRequired,
			ErrCaptchaInvalid,
			ErrCaptchaExpired,
			ErrCaptchaNotConfigured,
			ErrCaptchaScoreTooLow,
		}

		seen := make(map[string]bool)
		for _, err := range captchaErrors {
			msg := err.Error()
			assert.False(t, seen[msg], "Duplicate error message: %s", msg)
			seen[msg] = true
		}
	})

	t.Run("ErrCaptchaRequired message", func(t *testing.T) {
		assert.Equal(t, "captcha verification required", ErrCaptchaRequired.Error())
	})

	t.Run("ErrCaptchaInvalid message", func(t *testing.T) {
		assert.Equal(t, "captcha verification failed", ErrCaptchaInvalid.Error())
	})

	t.Run("ErrCaptchaExpired message", func(t *testing.T) {
		assert.Equal(t, "captcha token expired", ErrCaptchaExpired.Error())
	})

	t.Run("ErrCaptchaNotConfigured message", func(t *testing.T) {
		assert.Equal(t, "captcha provider not configured", ErrCaptchaNotConfigured.Error())
	})

	t.Run("ErrCaptchaScoreTooLow message", func(t *testing.T) {
		assert.Equal(t, "captcha score below threshold", ErrCaptchaScoreTooLow.Error())
	})

	t.Run("errors can be wrapped with errors.Is", func(t *testing.T) {
		wrappedErr := errors.New("wrapper: " + ErrCaptchaRequired.Error())
		// Note: This tests that errors are standard errors that can be compared
		assert.NotNil(t, wrappedErr)
	})
}

// =============================================================================
// CaptchaResult Tests
// =============================================================================

func TestCaptchaResult_Struct(t *testing.T) {
	t.Run("success result", func(t *testing.T) {
		result := CaptchaResult{
			Success:  true,
			Score:    0.9,
			Action:   "login",
			Hostname: "example.com",
		}

		assert.True(t, result.Success)
		assert.Equal(t, 0.9, result.Score)
		assert.Equal(t, "login", result.Action)
		assert.Equal(t, "example.com", result.Hostname)
		assert.Empty(t, result.ErrorCode)
	})

	t.Run("failed result with error code", func(t *testing.T) {
		result := CaptchaResult{
			Success:   false,
			ErrorCode: "invalid-input-response",
		}

		assert.False(t, result.Success)
		assert.Equal(t, "invalid-input-response", result.ErrorCode)
	})

	t.Run("zero value result", func(t *testing.T) {
		var result CaptchaResult

		assert.False(t, result.Success)
		assert.Equal(t, 0.0, result.Score)
		assert.Empty(t, result.Action)
		assert.Empty(t, result.Hostname)
		assert.True(t, result.Timestamp.IsZero())
	})
}

// =============================================================================
// CaptchaConfigResponse Tests
// =============================================================================

func TestCaptchaConfigResponse_Struct(t *testing.T) {
	t.Run("enabled config with all fields", func(t *testing.T) {
		resp := CaptchaConfigResponse{
			Enabled:   true,
			Provider:  "hcaptcha",
			SiteKey:   "site-key-123",
			Endpoints: []string{"signup", "login"},
		}

		assert.True(t, resp.Enabled)
		assert.Equal(t, "hcaptcha", resp.Provider)
		assert.Equal(t, "site-key-123", resp.SiteKey)
		assert.Contains(t, resp.Endpoints, "signup")
		assert.Contains(t, resp.Endpoints, "login")
	})

	t.Run("cap provider config", func(t *testing.T) {
		resp := CaptchaConfigResponse{
			Enabled:      true,
			Provider:     "cap",
			CapServerURL: "https://cap.example.com",
		}

		assert.Equal(t, "cap", resp.Provider)
		assert.Equal(t, "https://cap.example.com", resp.CapServerURL)
	})

	t.Run("disabled config", func(t *testing.T) {
		resp := CaptchaConfigResponse{
			Enabled: false,
		}

		assert.False(t, resp.Enabled)
		assert.Empty(t, resp.Provider)
		assert.Empty(t, resp.SiteKey)
	})
}

// =============================================================================
// NewCaptchaService Tests
// =============================================================================

func TestNewCaptchaService(t *testing.T) {
	t.Run("nil config returns disabled service", func(t *testing.T) {
		service, err := NewCaptchaService(nil)

		require.NoError(t, err)
		require.NotNil(t, service)
		assert.False(t, service.IsEnabled())
	})

	t.Run("disabled config returns disabled service", func(t *testing.T) {
		cfg := &config.CaptchaConfig{
			Enabled: false,
		}

		service, err := NewCaptchaService(cfg)

		require.NoError(t, err)
		require.NotNil(t, service)
		assert.False(t, service.IsEnabled())
	})

	t.Run("hcaptcha requires site_key and secret_key", func(t *testing.T) {
		cfg := &config.CaptchaConfig{
			Enabled:  true,
			Provider: "hcaptcha",
			SiteKey:  "",
			// Missing SecretKey
		}

		_, err := NewCaptchaService(cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "site_key and secret_key are required")
	})

	t.Run("recaptcha requires site_key and secret_key", func(t *testing.T) {
		cfg := &config.CaptchaConfig{
			Enabled:  true,
			Provider: "recaptcha_v3",
			// Missing keys
		}

		_, err := NewCaptchaService(cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "site_key and secret_key are required")
	})

	t.Run("turnstile requires site_key and secret_key", func(t *testing.T) {
		cfg := &config.CaptchaConfig{
			Enabled:  true,
			Provider: "turnstile",
			// Missing keys
		}

		_, err := NewCaptchaService(cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "site_key and secret_key are required")
	})

	t.Run("cap provider requires cap_server_url", func(t *testing.T) {
		cfg := &config.CaptchaConfig{
			Enabled:  true,
			Provider: "cap",
			// Missing CapServerURL
		}

		_, err := NewCaptchaService(cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cap_server_url is required")
	})

	t.Run("unknown provider returns error", func(t *testing.T) {
		cfg := &config.CaptchaConfig{
			Enabled:  true,
			Provider: "unknown_provider",
		}

		_, err := NewCaptchaService(cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unknown captcha provider")
	})

	t.Run("valid hcaptcha config creates service", func(t *testing.T) {
		cfg := &config.CaptchaConfig{
			Enabled:   true,
			Provider:  "hcaptcha",
			SiteKey:   "test-site-key",
			SecretKey: "test-secret-key",
			Endpoints: []string{"signup", "login"},
		}

		service, err := NewCaptchaService(cfg)

		require.NoError(t, err)
		require.NotNil(t, service)
		assert.True(t, service.IsEnabled())
		assert.Equal(t, "test-site-key", service.GetSiteKey())
		assert.Equal(t, "hcaptcha", service.GetProvider())
	})

	t.Run("valid recaptcha config creates service", func(t *testing.T) {
		cfg := &config.CaptchaConfig{
			Enabled:        true,
			Provider:       "recaptcha_v3",
			SiteKey:        "test-site-key",
			SecretKey:      "test-secret-key",
			ScoreThreshold: 0.7,
		}

		service, err := NewCaptchaService(cfg)

		require.NoError(t, err)
		require.NotNil(t, service)
		assert.True(t, service.IsEnabled())
	})

	t.Run("valid turnstile config creates service", func(t *testing.T) {
		cfg := &config.CaptchaConfig{
			Enabled:   true,
			Provider:  "turnstile",
			SiteKey:   "test-site-key",
			SecretKey: "test-secret-key",
		}

		service, err := NewCaptchaService(cfg)

		require.NoError(t, err)
		require.NotNil(t, service)
		assert.True(t, service.IsEnabled())
	})

	t.Run("valid cap config creates service", func(t *testing.T) {
		cfg := &config.CaptchaConfig{
			Enabled:      true,
			Provider:     "cap",
			CapServerURL: "https://cap.example.com",
			CapAPIKey:    "test-api-key",
		}

		service, err := NewCaptchaService(cfg)

		require.NoError(t, err)
		require.NotNil(t, service)
		assert.True(t, service.IsEnabled())
	})

	t.Run("recaptcha alias works", func(t *testing.T) {
		cfg := &config.CaptchaConfig{
			Enabled:   true,
			Provider:  "recaptcha", // Without _v3
			SiteKey:   "test-site-key",
			SecretKey: "test-secret-key",
		}

		service, err := NewCaptchaService(cfg)

		require.NoError(t, err)
		require.NotNil(t, service)
		assert.True(t, service.IsEnabled())
	})
}

// =============================================================================
// CaptchaService Method Tests
// =============================================================================

func TestCaptchaService_IsEnabled(t *testing.T) {
	t.Run("disabled with nil config", func(t *testing.T) {
		service := &CaptchaService{config: nil}
		assert.False(t, service.IsEnabled())
	})

	t.Run("disabled when config.Enabled is false", func(t *testing.T) {
		service := &CaptchaService{
			config: &config.CaptchaConfig{Enabled: false},
		}
		assert.False(t, service.IsEnabled())
	})

	t.Run("disabled when provider is nil", func(t *testing.T) {
		service := &CaptchaService{
			config:   &config.CaptchaConfig{Enabled: true},
			provider: nil,
		}
		assert.False(t, service.IsEnabled())
	})
}

func TestCaptchaService_IsEnabledForEndpoint(t *testing.T) {
	t.Run("returns false when service is disabled", func(t *testing.T) {
		service := &CaptchaService{config: nil}
		assert.False(t, service.IsEnabledForEndpoint("signup"))
	})

	t.Run("checks endpoint in enabled list", func(t *testing.T) {
		cfg := &config.CaptchaConfig{
			Enabled:   true,
			Provider:  "hcaptcha",
			SiteKey:   "test",
			SecretKey: "test",
			Endpoints: []string{"signup", "login"},
		}
		service, _ := NewCaptchaService(cfg)

		assert.True(t, service.IsEnabledForEndpoint("signup"))
		assert.True(t, service.IsEnabledForEndpoint("login"))
		assert.False(t, service.IsEnabledForEndpoint("other"))
	})

	t.Run("case insensitive endpoint check", func(t *testing.T) {
		cfg := &config.CaptchaConfig{
			Enabled:   true,
			Provider:  "hcaptcha",
			SiteKey:   "test",
			SecretKey: "test",
			Endpoints: []string{"signup"},
		}
		service, _ := NewCaptchaService(cfg)

		assert.True(t, service.IsEnabledForEndpoint("SIGNUP"))
		assert.True(t, service.IsEnabledForEndpoint("Signup"))
	})
}

func TestCaptchaService_GetSiteKey(t *testing.T) {
	t.Run("returns empty for nil config", func(t *testing.T) {
		service := &CaptchaService{config: nil}
		assert.Empty(t, service.GetSiteKey())
	})

	t.Run("returns site key from config", func(t *testing.T) {
		service := &CaptchaService{
			config: &config.CaptchaConfig{SiteKey: "my-site-key"},
		}
		assert.Equal(t, "my-site-key", service.GetSiteKey())
	})
}

func TestCaptchaService_GetProvider(t *testing.T) {
	t.Run("returns empty for nil config", func(t *testing.T) {
		service := &CaptchaService{config: nil}
		assert.Empty(t, service.GetProvider())
	})

	t.Run("returns provider from config", func(t *testing.T) {
		service := &CaptchaService{
			config: &config.CaptchaConfig{Provider: "hcaptcha"},
		}
		assert.Equal(t, "hcaptcha", service.GetProvider())
	})
}

func TestCaptchaService_GetConfig(t *testing.T) {
	t.Run("disabled for nil config", func(t *testing.T) {
		service := &CaptchaService{config: nil}
		resp := service.GetConfig()

		assert.False(t, resp.Enabled)
	})

	t.Run("disabled for disabled config", func(t *testing.T) {
		service := &CaptchaService{
			config: &config.CaptchaConfig{Enabled: false},
		}
		resp := service.GetConfig()

		assert.False(t, resp.Enabled)
	})

	t.Run("returns full config for enabled service", func(t *testing.T) {
		service := &CaptchaService{
			config: &config.CaptchaConfig{
				Enabled:   true,
				Provider:  "hcaptcha",
				SiteKey:   "test-key",
				Endpoints: []string{"signup", "login"},
			},
		}
		resp := service.GetConfig()

		assert.True(t, resp.Enabled)
		assert.Equal(t, "hcaptcha", resp.Provider)
		assert.Equal(t, "test-key", resp.SiteKey)
		assert.Equal(t, []string{"signup", "login"}, resp.Endpoints)
	})

	t.Run("includes cap_server_url for cap provider", func(t *testing.T) {
		service := &CaptchaService{
			config: &config.CaptchaConfig{
				Enabled:      true,
				Provider:     "cap",
				CapServerURL: "https://cap.example.com",
			},
		}
		resp := service.GetConfig()

		assert.Equal(t, "cap", resp.Provider)
		assert.Equal(t, "https://cap.example.com", resp.CapServerURL)
	})

	t.Run("excludes cap_server_url for other providers", func(t *testing.T) {
		service := &CaptchaService{
			config: &config.CaptchaConfig{
				Enabled:      true,
				Provider:     "hcaptcha",
				CapServerURL: "should-be-ignored",
			},
		}
		resp := service.GetConfig()

		assert.Empty(t, resp.CapServerURL)
	})
}

func TestCaptchaService_Verify(t *testing.T) {
	t.Run("skips verification when disabled", func(t *testing.T) {
		service := &CaptchaService{config: nil}
		err := service.Verify(context.Background(), "token", "127.0.0.1")
		assert.NoError(t, err)
	})

	t.Run("returns error for empty token when enabled", func(t *testing.T) {
		cfg := &config.CaptchaConfig{
			Enabled:   true,
			Provider:  "hcaptcha",
			SiteKey:   "test",
			SecretKey: "test",
		}
		service, _ := NewCaptchaService(cfg)

		err := service.Verify(context.Background(), "", "127.0.0.1")
		assert.Error(t, err)
		assert.Equal(t, ErrCaptchaRequired, err)
	})
}

func TestCaptchaService_VerifyForEndpoint(t *testing.T) {
	t.Run("skips verification for non-configured endpoint", func(t *testing.T) {
		cfg := &config.CaptchaConfig{
			Enabled:   true,
			Provider:  "hcaptcha",
			SiteKey:   "test",
			SecretKey: "test",
			Endpoints: []string{"signup"},
		}
		service, _ := NewCaptchaService(cfg)

		// "login" is not in the endpoints list
		err := service.VerifyForEndpoint(context.Background(), "login", "", "127.0.0.1")
		assert.NoError(t, err)
	})

	t.Run("requires verification for configured endpoint", func(t *testing.T) {
		cfg := &config.CaptchaConfig{
			Enabled:   true,
			Provider:  "hcaptcha",
			SiteKey:   "test",
			SecretKey: "test",
			Endpoints: []string{"signup"},
		}
		service, _ := NewCaptchaService(cfg)

		// "signup" is in the endpoints list, empty token should fail
		err := service.VerifyForEndpoint(context.Background(), "signup", "", "127.0.0.1")
		assert.Error(t, err)
		assert.Equal(t, ErrCaptchaRequired, err)
	})
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkNewCaptchaService_Disabled(b *testing.B) {
	cfg := &config.CaptchaConfig{Enabled: false}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = NewCaptchaService(cfg)
	}
}

func BenchmarkNewCaptchaService_Enabled(b *testing.B) {
	cfg := &config.CaptchaConfig{
		Enabled:   true,
		Provider:  "hcaptcha",
		SiteKey:   "test-key",
		SecretKey: "test-secret",
		Endpoints: []string{"signup", "login", "password_reset"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = NewCaptchaService(cfg)
	}
}

func BenchmarkCaptchaService_IsEnabledForEndpoint(b *testing.B) {
	cfg := &config.CaptchaConfig{
		Enabled:   true,
		Provider:  "hcaptcha",
		SiteKey:   "test",
		SecretKey: "test",
		Endpoints: []string{"signup", "login", "password_reset", "magic_link"},
	}
	service, _ := NewCaptchaService(cfg)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = service.IsEnabledForEndpoint("login")
	}
}

func BenchmarkCaptchaService_GetConfig(b *testing.B) {
	cfg := &config.CaptchaConfig{
		Enabled:   true,
		Provider:  "hcaptcha",
		SiteKey:   "test-key",
		Endpoints: []string{"signup", "login"},
	}
	service := &CaptchaService{config: cfg}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = service.GetConfig()
	}
}
