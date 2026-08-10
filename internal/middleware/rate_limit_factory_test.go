package middleware

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nimbleflux/fluxbase/internal/config"
	"github.com/nimbleflux/fluxbase/internal/settings"
)

// Unit tests for the pure surface of rate_limit_factory.go, which was at 0%:
// NewRateLimitFactory, the Create* family, getKeyFunc (all 8 key strategies),
// and the reflection helpers getConfigInt / getConfigDuration.
//
// All functions here are deterministic and I/O-free. HTTP context is driven via
// fiber.New() + c.Locals(), matching the pattern in clientkey_auth_test.go.
// The returned fiber.Handler is checked for non-nil but not invoked (invoking
// would require a real rate-limit Store, which is exercised elsewhere).

// =============================================================================
// NewRateLimitFactory + WithRateLimitSettingsCache
// =============================================================================

func TestNewRateLimitFactory(t *testing.T) {
	t.Parallel()

	t.Run("with security config builds reflect value", func(t *testing.T) {
		t.Parallel()
		sec := &config.SecurityConfig{AuthLoginRateLimit: 7}
		f := NewRateLimitFactory(sec, nil)
		require.NotNil(t, f)
		assert.True(t, f.configOpts.securityValue.IsValid(), "reflect.Value must be valid when security != nil")
		assert.Equal(t, 7, f.getConfigInt("AuthLoginRateLimit", 0))
	})

	t.Run("nil security config does not panic", func(t *testing.T) {
		t.Parallel()
		f := NewRateLimitFactory(nil, nil)
		require.NotNil(t, f)
		assert.False(t, f.configOpts.securityValue.IsValid(), "reflect.Value must be invalid when security == nil")
		// getConfigInt falls back to default when securityValue invalid.
		assert.Equal(t, 42, f.getConfigInt("AuthLoginRateLimit", 42))
	})

	t.Run("WithRateLimitSettingsCache sets the cache", func(t *testing.T) {
		t.Parallel()
		f := NewRateLimitFactory(nil, nil)
		assert.Nil(t, f.settings) // precondition
		// Construct a SettingsCache (zero-value is fine; we only check assignment).
		cache := &settings.SettingsCache{}
		WithRateLimitSettingsCache(cache)(f)
		assert.NotNil(t, f.settings)
	})
}

// =============================================================================
// Create (registry lookup + config override via reflection)
// =============================================================================

func TestRateLimitFactory_Create(t *testing.T) {
	t.Parallel()

	t.Run("unknown limiter errors", func(t *testing.T) {
		t.Parallel()
		f := NewRateLimitFactory(nil, nil)
		h, err := f.Create("does_not_exist")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown rate limiter")
		assert.Nil(t, h)
	})

	t.Run("known limiter returns handler with defaults when no security", func(t *testing.T) {
		t.Parallel()
		f := NewRateLimitFactory(nil, nil)
		h, err := f.Create("auth_login")
		require.NoError(t, err)
		assert.NotNil(t, h)
	})

	t.Run("security config overrides max and window", func(t *testing.T) {
		t.Parallel()
		sec := &config.SecurityConfig{
			AuthLoginRateLimit:  3,
			AuthLoginRateWindow: 2 * time.Minute,
		}
		f := NewRateLimitFactory(sec, nil)
		h, err := f.Create("auth_login")
		require.NoError(t, err)
		assert.NotNil(t, h)
		// The override is observable via getConfigInt/Duration (covered below);
		// here we confirm Create doesn't error when ConfigMaxField is set.
	})

	t.Run("security config present but field missing uses defaults", func(t *testing.T) {
		t.Parallel()
		// auth_2fa has ConfigMaxField set; pass a config with zero values.
		sec := &config.SecurityConfig{}
		f := NewRateLimitFactory(sec, nil)
		h, err := f.Create("auth_2fa")
		require.NoError(t, err)
		assert.NotNil(t, h)
	})
}

// =============================================================================
// CreateWithOverride
// =============================================================================

func TestRateLimitFactory_CreateWithOverride(t *testing.T) {
	t.Parallel()
	f := NewRateLimitFactory(nil, nil)

	t.Run("unknown limiter errors", func(t *testing.T) {
		t.Parallel()
		_, err := f.CreateWithOverride("nope", 1, time.Second)
		require.Error(t, err)
	})

	t.Run("known limiter returns handler with custom values", func(t *testing.T) {
		t.Parallel()
		h, err := f.CreateWithOverride("auth_login", 99, 30*time.Second)
		require.NoError(t, err)
		assert.NotNil(t, h)
	})
}

// =============================================================================
// CreateFromConfig
// =============================================================================

func TestRateLimitFactory_CreateFromConfig(t *testing.T) {
	t.Parallel()
	f := NewRateLimitFactory(nil, nil)

	t.Run("unknown limiter errors", func(t *testing.T) {
		t.Parallel()
		_, err := f.CreateFromConfig("nope", nil)
		require.Error(t, err)
	})

	t.Run("returns handler with defaults when cache nil", func(t *testing.T) {
		t.Parallel()
		h, err := f.CreateFromConfig("auth_login", nil)
		require.NoError(t, err)
		assert.NotNil(t, h)
	})

	t.Run("returns handler with cache present (currently defaults)", func(t *testing.T) {
		t.Parallel()
		cache := &settings.SettingsCache{}
		h, err := f.CreateFromConfig("auth_login", cache)
		require.NoError(t, err)
		assert.NotNil(t, h)
	})
}

// =============================================================================
// getKeyFunc — all 8 strategies + prefix fallback
// =============================================================================
//
// Each strategy returns a closure; we drive it with a fiber.Ctx whose Locals
// and body we control. The closure's output key encodes the strategy, so we
// assert on the prefix structure.

// keyFuncResult runs getKeyFunc(def) inside a fiber handler that captures the
// generated key. locals are set on the context; body (if non-nil) is the JSON
// request body. Returns the key string the KeyFunc produced.
func keyFuncResult(t *testing.T, def RateLimitDefinition, locals map[string]any, body []byte) string {
	t.Helper()
	f := NewRateLimitFactory(nil, nil)
	kf := f.getKeyFunc(def)

	var captured string
	app := fiber.New()
	app.Use(func(c fiber.Ctx) error {
		for k, v := range locals {
			c.Locals(k, v)
		}
		return c.Next()
	})
	app.Post("/test", func(c fiber.Ctx) error {
		captured = kf(c)
		return c.SendStatus(200)
	})

	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}
	req := httptest.NewRequest(http.MethodPost, "/test", bodyReader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := app.Test(req)
	require.NoError(t, err, "app.Test failed")
	require.Equal(t, 200, resp.StatusCode)
	return captured
}

func TestRateLimitFactory_GetKeyFunc_IP(t *testing.T) {
	t.Parallel()
	def := RateLimitDefinition{Name: "auth_login", KeyPrefix: "login", KeyStrategy: KeyStrategyIP}
	key := keyFuncResult(t, def, nil, nil)
	assert.Contains(t, key, "login:")
}

func TestRateLimitFactory_GetKeyFunc_User(t *testing.T) {
	t.Parallel()
	def := RateLimitDefinition{Name: "x", KeyPrefix: "x", KeyStrategy: KeyStrategyUser}

	t.Run("authenticated user key", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "x_user:user-123", keyFuncResult(t, def, map[string]any{"user_id": "user-123"}, nil))
	})

	t.Run("anonymous falls back to ip", func(t *testing.T) {
		t.Parallel()
		assert.Contains(t, keyFuncResult(t, def, map[string]any{"user_id": "anonymous"}, nil), "x_ip:")
	})

	t.Run("empty user id falls back to ip", func(t *testing.T) {
		t.Parallel()
		assert.Contains(t, keyFuncResult(t, def, map[string]any{"user_id": ""}, nil), "x_ip:")
	})

	t.Run("nil user id falls back to ip", func(t *testing.T) {
		t.Parallel()
		assert.Contains(t, keyFuncResult(t, def, nil, nil), "x_ip:")
	})

	t.Run("wrong-type user id falls back to ip", func(t *testing.T) {
		t.Parallel()
		assert.Contains(t, keyFuncResult(t, def, map[string]any{"user_id": 12345}, nil), "x_ip:")
	})
}

func TestRateLimitFactory_GetKeyFunc_ClientKey(t *testing.T) {
	t.Parallel()
	def := RateLimitDefinition{Name: "x", KeyPrefix: "x", KeyStrategy: KeyStrategyClientKey}

	t.Run("client key id present", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "x_key:ck-1", keyFuncResult(t, def, map[string]any{"client_key_id": "ck-1"}, nil))
	})

	t.Run("missing falls back to ip", func(t *testing.T) {
		t.Parallel()
		assert.Contains(t, keyFuncResult(t, def, nil, nil), "x_ip:")
	})

	t.Run("empty falls back to ip", func(t *testing.T) {
		t.Parallel()
		assert.Contains(t, keyFuncResult(t, def, map[string]any{"client_key_id": ""}, nil), "x_ip:")
	})
}

func TestRateLimitFactory_GetKeyFunc_ServiceKey(t *testing.T) {
	t.Parallel()
	def := RateLimitDefinition{Name: "x", KeyPrefix: "x", KeyStrategy: KeyStrategyServiceKey}

	t.Run("service key id present", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "x_key:sk-9", keyFuncResult(t, def, map[string]any{"service_key_id": "sk-9"}, nil))
	})

	t.Run("missing falls back to ip", func(t *testing.T) {
		t.Parallel()
		assert.Contains(t, keyFuncResult(t, def, nil, nil), "x_ip:")
	})
}

func TestRateLimitFactory_GetKeyFunc_Token(t *testing.T) {
	t.Parallel()
	def := RateLimitDefinition{Name: "x", KeyPrefix: "x", KeyStrategy: KeyStrategyToken}

	t.Run("long token truncated to 20 chars", func(t *testing.T) {
		t.Parallel()
		tok := "abcdefghijklmnopqrstuvwxyz0123456789" // 36 chars
		key := keyFuncResult(t, def, nil, []byte(`{"refresh_token":"`+tok+`"}`))
		assert.Equal(t, "x:abcdefghijklmnopqrst", key) // first 20
	})

	t.Run("short token used whole", func(t *testing.T) {
		t.Parallel()
		key := keyFuncResult(t, def, nil, []byte(`{"refresh_token":"short"}`))
		assert.Equal(t, "x:short", key)
	})

	t.Run("missing token falls back to ip", func(t *testing.T) {
		t.Parallel()
		// Empty body -> no token -> IP-based key (still has prefix).
		assert.Contains(t, keyFuncResult(t, def, nil, []byte(`{}`)), "x:")
	})

	t.Run("malformed body falls back to ip", func(t *testing.T) {
		t.Parallel()
		assert.Contains(t, keyFuncResult(t, def, nil, []byte(`{not json`)), "x:")
	})
}

func TestRateLimitFactory_GetKeyFunc_Email(t *testing.T) {
	t.Parallel()
	def := RateLimitDefinition{Name: "x", KeyPrefix: "x", KeyStrategy: KeyStrategyEmail}

	t.Run("email present", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "x:a@b.com", keyFuncResult(t, def, nil, []byte(`{"email":"a@b.com"}`)))
	})

	t.Run("missing email falls back to ip", func(t *testing.T) {
		t.Parallel()
		assert.Contains(t, keyFuncResult(t, def, nil, []byte(`{}`)), "x:")
	})
}

func TestRateLimitFactory_GetKeyFunc_Tiered(t *testing.T) {
	t.Parallel()
	def := RateLimitDefinition{Name: "x", KeyPrefix: "x", KeyStrategy: KeyStrategyTiered}

	t.Run("authenticated user", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "x_user:u-1", keyFuncResult(t, def, map[string]any{"user_id": "u-1"}, nil))
	})

	t.Run("anonymous falls back to ip", func(t *testing.T) {
		t.Parallel()
		assert.Contains(t, keyFuncResult(t, def, nil, nil), "x_ip:")
	})
}

func TestRateLimitFactory_GetKeyFunc_DefaultAndUnknownStrategy(t *testing.T) {
	t.Parallel()
	def := RateLimitDefinition{Name: "x", KeyPrefix: "x", KeyStrategy: "bogus"}
	assert.Contains(t, keyFuncResult(t, def, nil, nil), "x:") // default branch -> IP
}

func TestRateLimitFactory_GetKeyFunc_PrefixFallback(t *testing.T) {
	t.Parallel()
	// Empty KeyPrefix falls back to def.Name.
	def := RateLimitDefinition{Name: "my_limiter", KeyPrefix: "", KeyStrategy: KeyStrategyIP}
	assert.Contains(t, keyFuncResult(t, def, nil, nil), "my_limiter:")
}

// =============================================================================
// getConfigInt / getConfigDuration (reflection)
// =============================================================================

func TestRateLimitFactory_GetConfigInt(t *testing.T) {
	t.Parallel()

	t.Run("returns configured value", func(t *testing.T) {
		t.Parallel()
		sec := &config.SecurityConfig{AuthLoginRateLimit: 42}
		f := NewRateLimitFactory(sec, nil)
		assert.Equal(t, 42, f.getConfigInt("AuthLoginRateLimit", 0))
	})

	t.Run("missing field returns default", func(t *testing.T) {
		t.Parallel()
		sec := &config.SecurityConfig{}
		f := NewRateLimitFactory(sec, nil)
		assert.Equal(t, 99, f.getConfigInt("NonexistentField", 99))
	})

	t.Run("nil security returns default", func(t *testing.T) {
		t.Parallel()
		f := NewRateLimitFactory(nil, nil)
		assert.Equal(t, 5, f.getConfigInt("AuthLoginRateLimit", 5))
	})

	t.Run("non-int field returns default", func(t *testing.T) {
		t.Parallel()
		// EnableGlobalRateLimit is a bool, not int -> default returned.
		sec := &config.SecurityConfig{EnableGlobalRateLimit: true}
		f := NewRateLimitFactory(sec, nil)
		assert.Equal(t, 7, f.getConfigInt("EnableGlobalRateLimit", 7))
	})
}

func TestRateLimitFactory_GetConfigDuration(t *testing.T) {
	t.Parallel()

	t.Run("returns configured duration", func(t *testing.T) {
		t.Parallel()
		sec := &config.SecurityConfig{AuthLoginRateWindow: 3 * time.Minute}
		f := NewRateLimitFactory(sec, nil)
		assert.Equal(t, 3*time.Minute, f.getConfigDuration("AuthLoginRateWindow", 0))
	})

	t.Run("missing field returns default", func(t *testing.T) {
		t.Parallel()
		sec := &config.SecurityConfig{}
		f := NewRateLimitFactory(sec, nil)
		def := 10 * time.Second
		assert.Equal(t, def, f.getConfigDuration("Nope", def))
	})

	t.Run("nil security returns default", func(t *testing.T) {
		t.Parallel()
		f := NewRateLimitFactory(nil, nil)
		assert.Equal(t, time.Hour, f.getConfigDuration("AuthLoginRateWindow", time.Hour))
	})
}

// =============================================================================
// CreateTieredLimiter
// =============================================================================

func TestRateLimitFactory_CreateTieredLimiter(t *testing.T) {
	t.Parallel()
	f := NewRateLimitFactory(nil, nil)

	t.Run("unknown name errors", func(t *testing.T) {
		t.Parallel()
		_, err := f.CreateTieredLimiter("nope", 1, 2, 3, time.Minute)
		require.Error(t, err)
	})

	t.Run("returns handler for known name", func(t *testing.T) {
		t.Parallel()
		h, err := f.CreateTieredLimiter("auth_login", 10, 50, 100, time.Minute)
		require.NoError(t, err)
		assert.NotNil(t, h)
	})

	t.Run("prefix fallback when empty", func(t *testing.T) {
		t.Parallel()
		// Use a def with empty KeyPrefix to exercise the fallback; "auth_login"
		// has a prefix, so construct a synthetic registry entry instead.
		orig := Registry["auth_login"]
		Registry["__test_tiered"] = RateLimitDefinition{
			Name: "__test_tiered", KeyPrefix: "", KeyStrategy: KeyStrategyTiered,
		}
		t.Cleanup(func() { delete(Registry, "__test_tiered"); Registry["auth_login"] = orig })
		h, err := f.CreateTieredLimiter("__test_tiered", 1, 2, 3, time.Second)
		require.NoError(t, err)
		assert.NotNil(t, h)
	})
}

// =============================================================================
// CreateEmailBasedLimiter
// =============================================================================

func TestRateLimitFactory_CreateEmailBasedLimiter(t *testing.T) {
	t.Parallel()
	f := NewRateLimitFactory(nil, nil)

	t.Run("unknown name errors", func(t *testing.T) {
		t.Parallel()
		_, err := f.CreateEmailBasedLimiter("nope", 1, time.Second)
		require.Error(t, err)
	})

	t.Run("returns handler for known name", func(t *testing.T) {
		t.Parallel()
		h, err := f.CreateEmailBasedLimiter("auth_password_reset", 5, time.Minute)
		require.NoError(t, err)
		assert.NotNil(t, h)
	})
}

// =============================================================================
// createLimiter — default message generation
// =============================================================================

func TestRateLimitFactory_CreateLimiter_DefaultMessage(t *testing.T) {
	t.Parallel()
	f := NewRateLimitFactory(nil, nil)

	t.Run("uses def message when provided", func(t *testing.T) {
		t.Parallel()
		// auth_login has a Message in the registry; just confirm no panic + non-nil.
		h := f.createLimiter(Registry["auth_login"], 10, 15*time.Minute)
		assert.NotNil(t, h)
	})

	t.Run("generates message when def message empty", func(t *testing.T) {
		t.Parallel()
		def := RateLimitDefinition{Name: "x", Message: ""}
		h := f.createLimiter(def, 5, 10*time.Second)
		assert.NotNil(t, h)
	})
}
