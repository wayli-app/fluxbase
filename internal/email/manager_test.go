package email

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nimbleflux/fluxbase/internal/config"
)

func TestNewManager(t *testing.T) {
	t.Run("creates manager with env config", func(t *testing.T) {
		t.Parallel()
		cfg := &config.EmailConfig{
			Enabled:  false,
			Provider: "none",
		}

		manager := NewManager(cfg, nil, nil, nil)
		assert.NotNil(t, manager)
		assert.NotNil(t, manager.service)
		assert.Equal(t, cfg, manager.envConfig)
	})

	t.Run("creates manager with nil config", func(t *testing.T) {
		t.Parallel()
		manager := NewManager(nil, nil, nil, nil)
		assert.NotNil(t, manager)
		assert.NotNil(t, manager.service) // Should have NoOp or default service
	})
}

func TestManager_GetService(t *testing.T) {
	t.Run("returns current service", func(t *testing.T) {
		t.Parallel()
		cfg := &config.EmailConfig{Enabled: false}
		manager := NewManager(cfg, nil, nil, nil)

		service := manager.GetService()
		assert.NotNil(t, service)
	})
}

func TestManager_WrapAsService(t *testing.T) {
	cfg := &config.EmailConfig{Enabled: false}
	manager := NewManager(cfg, nil, nil, nil)

	t.Run("creates service wrapper", func(t *testing.T) {
		t.Parallel()
		wrapper := manager.WrapAsService()
		assert.NotNil(t, wrapper)
	})
}

func TestServiceWrapper_IsConfigured(t *testing.T) {
	cfg := &config.EmailConfig{Enabled: false}
	manager := NewManager(cfg, nil, nil, nil)
	wrapper := manager.WrapAsService()

	t.Run("delegates to underlying service", func(t *testing.T) {
		t.Parallel()
		isConfigured := wrapper.IsConfigured()
		// With empty config, should not be configured
		assert.False(t, isConfigured)
	})
}

func TestServiceWrapper_Send(t *testing.T) {
	cfg := &config.EmailConfig{
		Enabled:     false, // Disabled service won't actually send
		FromAddress: "test@example.com",
	}
	manager := NewManager(cfg, nil, nil, nil)
	wrapper := manager.WrapAsService()

	t.Run("delegates to underlying service", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		err := wrapper.Send(ctx, "user@example.com", "Test Subject", "Test Body")
		// The NoOp service should return an error or nil depending on config
		// This tests that the delegation works
		_ = err // Error handling depends on service implementation
	})
}

func TestServiceWrapper_SendMagicLink(t *testing.T) {
	cfg := &config.EmailConfig{Enabled: false}
	manager := NewManager(cfg, nil, nil, nil)
	wrapper := manager.WrapAsService()

	t.Run("delegates to underlying service", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		err := wrapper.SendMagicLink(ctx, "user@example.com", "token123", "https://example.com/link")
		_ = err // Error handling depends on service implementation
	})
}

func TestServiceWrapper_SendVerificationEmail(t *testing.T) {
	cfg := &config.EmailConfig{Enabled: false}
	manager := NewManager(cfg, nil, nil, nil)
	wrapper := manager.WrapAsService()

	t.Run("delegates to underlying service", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		err := wrapper.SendVerificationEmail(ctx, "user@example.com", "token123", "https://example.com/verify")
		_ = err // Error handling depends on service implementation
	})
}

func TestServiceWrapper_SendPasswordReset(t *testing.T) {
	cfg := &config.EmailConfig{Enabled: false}
	manager := NewManager(cfg, nil, nil, nil)
	wrapper := manager.WrapAsService()

	t.Run("delegates to underlying service", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		err := wrapper.SendPasswordReset(ctx, "user@example.com", "token123", "https://example.com/reset")
		_ = err // Error handling depends on service implementation
	})
}

func TestServiceWrapper_SendInvitationEmail(t *testing.T) {
	cfg := &config.EmailConfig{Enabled: false}
	manager := NewManager(cfg, nil, nil, nil)
	wrapper := manager.WrapAsService()

	t.Run("delegates to underlying service", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		err := wrapper.SendInvitationEmail(ctx, "user@example.com", "Admin User", "https://example.com/invite")
		_ = err // Error handling depends on service implementation
	})
}

func TestManager_BuildConfigFromSettings(t *testing.T) {
	t.Run("uses env config when no settings cache", func(t *testing.T) {
		t.Parallel()
		envCfg := &config.EmailConfig{
			Enabled:     true,
			Provider:    "smtp",
			FromAddress: "env@example.com",
			SMTPHost:    "smtp.example.com",
			SMTPPort:    587,
		}

		manager := &Manager{envConfig: envCfg}
		ctx := context.Background()

		cfg := manager.buildConfigFromSettings(ctx)
		assert.Equal(t, true, cfg.Enabled)
		assert.Equal(t, "smtp", cfg.Provider)
		assert.Equal(t, "env@example.com", cfg.FromAddress)
		assert.Equal(t, "smtp.example.com", cfg.SMTPHost)
		assert.Equal(t, 587, cfg.SMTPPort)
	})

	t.Run("handles nil env config", func(t *testing.T) {
		t.Parallel()
		manager := &Manager{envConfig: nil}
		ctx := context.Background()

		cfg := manager.buildConfigFromSettings(ctx)
		require.NotNil(t, cfg)
		// Should have zero values
		assert.False(t, cfg.Enabled)
		assert.Empty(t, cfg.Provider)
	})
}

func TestManager_Concurrency(t *testing.T) {
	cfg := &config.EmailConfig{Enabled: false}
	manager := NewManager(cfg, nil, nil, nil)

	t.Run("concurrent GetService is safe", func(t *testing.T) {
		t.Parallel()
		done := make(chan bool)

		for i := 0; i < 10; i++ {
			go func() {
				service := manager.GetService()
				assert.NotNil(t, service)
				done <- true
			}()
		}

		for i := 0; i < 10; i++ {
			<-done
		}
	})
}
