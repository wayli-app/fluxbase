package config

import (
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
)

// SecurityConfig contains security-related settings
type SecurityConfig struct {
	EnableGlobalRateLimit bool `mapstructure:"enable_global_rate_limit"` // Global API rate limiting (100 req/min per IP)

	// Service role token revocation behavior
	ServiceRoleFailOpen bool `mapstructure:"service_role_fail_open"` // If false (default), fail-closed when revocation check fails (503). If true, fail-open for backward compatibility.

	// Admin setup security token
	SetupToken string `mapstructure:"setup_token"` // Required token for admin setup. If empty, admin dashboard is disabled.

	// Rate limiting for specific endpoints
	AdminSetupRateLimit         int           `mapstructure:"admin_setup_rate_limit"`          // Max attempts for admin setup
	AdminSetupRateWindow        time.Duration `mapstructure:"admin_setup_rate_window"`         // Time window for admin setup rate limit
	AdminLoginRateLimit         int           `mapstructure:"admin_login_rate_limit"`          // Max attempts for admin login
	AdminLoginRateWindow        time.Duration `mapstructure:"admin_login_rate_window"`         // Time window for admin login rate limit
	DashboardLoginRateLimit     int           `mapstructure:"dashboard_login_rate_limit"`      // Max attempts for dashboard user login
	DashboardLoginRateWindow    time.Duration `mapstructure:"dashboard_login_rate_window"`     // Time window for dashboard user login rate limit
	AuthLoginRateLimit          int           `mapstructure:"auth_login_rate_limit"`           // Max attempts for auth login
	AuthLoginRateWindow         time.Duration `mapstructure:"auth_login_rate_window"`          // Time window for auth login rate limit
	AuthSignupRateLimit         int           `mapstructure:"auth_signup_rate_limit"`          // Max attempts for auth signup
	AuthSignupRateWindow        time.Duration `mapstructure:"auth_signup_rate_window"`         // Time window for auth signup rate limit
	AuthPasswordResetRateLimit  int           `mapstructure:"auth_password_reset_rate_limit"`  // Max attempts for password reset
	AuthPasswordResetRateWindow time.Duration `mapstructure:"auth_password_reset_rate_window"` // Time window for password reset rate limit
	Auth2FARateLimit            int           `mapstructure:"auth_2fa_rate_limit"`             // Max attempts for 2FA verification
	Auth2FARateWindow           time.Duration `mapstructure:"auth_2fa_rate_window"`            // Time window for 2FA rate limit
	AuthRefreshRateLimit        int           `mapstructure:"auth_refresh_rate_limit"`         // Max attempts for token refresh
	AuthRefreshRateWindow       time.Duration `mapstructure:"auth_refresh_rate_window"`        // Time window for token refresh rate limit
	AuthMagicLinkRateLimit      int           `mapstructure:"auth_magic_link_rate_limit"`      // Max attempts for magic link
	AuthMagicLinkRateWindow     time.Duration `mapstructure:"auth_magic_link_rate_window"`     // Time window for magic link rate limit

	// Rate limiting for service_role tokens (bypassed by default, but can be enabled)
	ServiceRoleRateLimit  int           `mapstructure:"service_role_rate_limit"`  // Max requests for service_role tokens (0 = unlimited)
	ServiceRoleRateWindow time.Duration `mapstructure:"service_role_rate_window"` // Time window for service_role rate limit

	// CAPTCHA configuration for bot protection
	Captcha CaptchaConfig `mapstructure:"captcha"`
}

// CaptchaConfig contains CAPTCHA verification settings for bot protection
type CaptchaConfig struct {
	Enabled        bool     `mapstructure:"enabled"`         // Enable CAPTCHA verification
	Provider       string   `mapstructure:"provider"`        // Provider: hcaptcha, recaptcha_v3, turnstile, cap
	SiteKey        string   `mapstructure:"site_key"`        // Public site key (sent to frontend)
	SecretKey      string   `mapstructure:"secret_key"`      // Secret key for server-side verification
	ScoreThreshold float64  `mapstructure:"score_threshold"` // Min score for reCAPTCHA v3 (0.0-1.0, default 0.5)
	Endpoints      []string `mapstructure:"endpoints"`       // Endpoints requiring CAPTCHA: signup, login, password_reset, magic_link
	// Cap provider settings (self-hosted proof-of-work CAPTCHA)
	CapServerURL string `mapstructure:"cap_server_url"` // URL of Cap server (e.g., http://localhost:3000)
	CapAPIKey    string `mapstructure:"cap_api_key"`    // API key for Cap server authentication
	// Adaptive trust settings for intelligent CAPTCHA decisions
	AdaptiveTrust AdaptiveTrustConfig `mapstructure:"adaptive_trust"`
}

// AdaptiveTrustConfig contains settings for the adaptive CAPTCHA trust system
type AdaptiveTrustConfig struct {
	Enabled bool `mapstructure:"enabled"` // Enable adaptive trust (skip CAPTCHA for trusted users)

	// Trust token settings
	TrustTokenTTL     time.Duration `mapstructure:"trust_token_ttl"`      // How long a CAPTCHA solution is trusted (default: 15m)
	TrustTokenBoundIP bool          `mapstructure:"trust_token_bound_ip"` // Token only valid from same IP (default: true)

	// Challenge settings
	ChallengeExpiry time.Duration `mapstructure:"challenge_expiry"` // How long a challenge_id is valid (default: 5m)

	// Trust score threshold - score below this requires CAPTCHA
	CaptchaThreshold int `mapstructure:"captcha_threshold"` // Default: 50

	// Trust signal weights (positive signals)
	WeightKnownIP          int `mapstructure:"weight_known_ip"`          // User logged in from this IP before (default: 30)
	WeightKnownDevice      int `mapstructure:"weight_known_device"`      // Device fingerprint seen before (default: 25)
	WeightRecentCaptcha    int `mapstructure:"weight_recent_captcha"`    // Solved CAPTCHA recently (default: 40)
	WeightVerifiedEmail    int `mapstructure:"weight_verified_email"`    // Email address is confirmed (default: 15)
	WeightAccountAge       int `mapstructure:"weight_account_age"`       // Account older than 7 days (default: 10)
	WeightSuccessfulLogins int `mapstructure:"weight_successful_logins"` // 3+ successful logins (default: 10)
	WeightMFAEnabled       int `mapstructure:"weight_mfa_enabled"`       // User has MFA configured (default: 20)

	// Trust signal weights (negative signals)
	WeightNewIP          int `mapstructure:"weight_new_ip"`          // Never seen this IP (default: -30)
	WeightNewDevice      int `mapstructure:"weight_new_device"`      // Unknown device fingerprint (default: -25)
	WeightFailedAttempts int `mapstructure:"weight_failed_attempts"` // Recent failed login attempts (default: -20)

	// Per-endpoint overrides (some actions always need CAPTCHA regardless of trust)
	AlwaysRequireEndpoints []string `mapstructure:"always_require_endpoints"` // Endpoints that always require CAPTCHA (default: ["password_reset"])
}

// Validate validates security configuration
func (sc *SecurityConfig) Validate() error {
	// Check for insecure default setup token if admin dashboard is enabled
	if sc.SetupToken != "" {
		insecureDefaults := []string{
			"your-secret-setup-token-change-in-production",
			"your-secret-setup-token",
			"changeme",
			"test",
		}
		for _, insecure := range insecureDefaults {
			if sc.SetupToken == insecure {
				return fmt.Errorf("please set a secure setup token (current value '%s' is insecure)", sc.SetupToken)
			}
		}

		// Warn if setup token is too short
		if len(sc.SetupToken) < 32 {
			log.Warn().Msg("Security setup token is shorter than 32 characters - consider using a longer token for better security")
		}
	}

	return nil
}
