package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

// AuthConfig contains authentication settings
type AuthConfig struct {
	JWTSecret           string        `mapstructure:"jwt_secret"`
	JWTExpiry           time.Duration `mapstructure:"jwt_expiry"`
	RefreshExpiry       time.Duration `mapstructure:"refresh_expiry"`
	ServiceRoleTTL      time.Duration `mapstructure:"service_role_ttl"` // TTL for service role tokens (default: 24h)
	AnonTTL             time.Duration `mapstructure:"anon_ttl"`         // TTL for anonymous tokens (default: 24h)
	MagicLinkExpiry     time.Duration `mapstructure:"magic_link_expiry"`
	PasswordResetExpiry time.Duration `mapstructure:"password_reset_expiry"`
	PasswordMinLen      int           `mapstructure:"password_min_length"`
	BcryptCost          int           `mapstructure:"bcrypt_cost"`
	SignupEnabled       bool          `mapstructure:"signup_enabled"`
	MagicLinkEnabled    bool          `mapstructure:"magic_link_enabled"`
	TOTPIssuer          string        `mapstructure:"totp_issuer"` // Issuer name displayed in authenticator apps for 2FA (e.g., "MyApp")

	// OAuth/OIDC provider configuration (unified for all providers)
	// Well-known providers (google, apple, microsoft) auto-detect issuer URLs
	// Custom providers require explicit issuer_url (supports base URLs like https://auth.domain.com or full .well-known URLs)
	OAuthProviders []OAuthProviderConfig `mapstructure:"oauth_providers"`

	// SAML SSO providers for enterprise authentication
	SAMLProviders []SAMLProviderConfig `mapstructure:"saml_providers"`

	// AllowUserClientKeys controls whether regular users can create their own client keys.
	// When false, only admins (service_role or instance_admin) can create/manage client keys,
	// and existing user-created keys are blocked from authenticating.
	// Default: true
	AllowUserClientKeys bool `mapstructure:"allow_user_client_keys"`

	// OAuthStateStorage configures how OAuth state tokens are stored.
	// "memory" - In-memory storage (default, single-instance only)
	// "database" - PostgreSQL storage (required for multi-instance deployments)
	// Default: "memory"
	OAuthStateStorage string `mapstructure:"oauth_state_storage"`
}

// SAMLProviderConfig represents a SAML 2.0 Identity Provider configuration
type SAMLProviderConfig struct {
	Name             string            `mapstructure:"name"`              // Provider name (e.g., "okta", "azure-ad")
	Enabled          bool              `mapstructure:"enabled"`           // Enable this provider
	IdPMetadataURL   string            `mapstructure:"idp_metadata_url"`  // IdP metadata URL (recommended)
	IdPMetadataXML   string            `mapstructure:"idp_metadata_xml"`  // IdP metadata XML (alternative to URL)
	EntityID         string            `mapstructure:"entity_id"`         // SP entity ID (unique identifier for this app)
	AcsURL           string            `mapstructure:"acs_url"`           // Assertion Consumer Service URL (callback)
	AttributeMapping map[string]string `mapstructure:"attribute_mapping"` // Map SAML attributes to user fields
	AutoCreateUsers  bool              `mapstructure:"auto_create_users"` // Create user if not exists
	DefaultRole      string            `mapstructure:"default_role"`      // Default role for new users (authenticated)

	// Security options
	AllowIDPInitiated        bool     `mapstructure:"allow_idp_initiated"`         // Allow IdP-initiated SSO (default: false for security)
	AllowedRedirectHosts     []string `mapstructure:"allowed_redirect_hosts"`      // Whitelist for RelayState redirect URLs
	AllowInsecureMetadataURL bool     `mapstructure:"allow_insecure_metadata_url"` // Allow HTTP metadata URLs (default: false)

	// Login targeting
	AllowDashboardLogin bool `mapstructure:"allow_dashboard_login"` // Allow for dashboard admin SSO (default: false)
	AllowAppLogin       bool `mapstructure:"allow_app_login"`       // Allow for app user authentication (default: true)

	// Role/Group-based access control
	RequiredGroups    []string `mapstructure:"required_groups"`     // User must be in at least ONE of these groups (OR logic)
	RequiredGroupsAll []string `mapstructure:"required_groups_all"` // User must be in ALL of these groups (AND logic)
	DeniedGroups      []string `mapstructure:"denied_groups"`       // Reject if user is in any of these groups
	GroupAttribute    string   `mapstructure:"group_attribute"`     // SAML attribute name for groups (default: "groups")

	// SP signing keys for SLO (Single Logout) - PEM-encoded
	SPCertificate string `mapstructure:"sp_certificate"` // PEM-encoded X.509 certificate for signing
	SPPrivateKey  string `mapstructure:"sp_private_key"` // PEM-encoded private key for signing

	// Logout signature verification
	RequireLogoutSignature *bool `mapstructure:"require_logout_signature"` // Require signed SAML logout messages (default: true)
}

// OAuthProviderConfig represents a unified OAuth/OIDC provider configuration
// Supports both well-known providers (Google, Apple, Microsoft) and custom providers
type OAuthProviderConfig struct {
	Name         string   `mapstructure:"name"`                    // Provider name (e.g., "google", "apple", "keycloak")
	Enabled      bool     `mapstructure:"enabled"`                 // Enable this provider (default: true)
	ClientID     string   `mapstructure:"client_id"`               // OAuth client ID (REQUIRED)
	ClientSecret string   `mapstructure:"client_secret,omitempty"` // Client secret (optional, can be stored in database)
	IssuerURL    string   `mapstructure:"issuer_url,omitempty"`    // OIDC issuer URL - supports base URLs (e.g., https://auth.domain.com) with auto-discovery or full .well-known URLs (auto-detected for well-known providers)
	RedirectURLs []string `mapstructure:"redirect_urls,omitempty"` // Additional allowed OAuth redirect URLs (the default callback URL is always included)
	Scopes       []string `mapstructure:"scopes,omitempty"`        // OAuth scopes
	DisplayName  string   `mapstructure:"display_name,omitempty"`  // Display name for UI

	// Login targeting
	AllowDashboardLogin bool `mapstructure:"allow_dashboard_login"` // Allow for dashboard admin SSO (default: false)
	AllowAppLogin       bool `mapstructure:"allow_app_login"`       // Allow for app user authentication (default: true)

	// Claims-based access control
	RequiredClaims map[string][]string `mapstructure:"required_claims"` // Claims that must be present in ID token, e.g., {"roles": ["admin"], "department": ["IT"]}
	DeniedClaims   map[string][]string `mapstructure:"denied_claims"`   // Deny access if these claim values are present
}

// Validate validates auth configuration
func (ac *AuthConfig) Validate() error {
	if ac.JWTSecret == "" {
		return fmt.Errorf("jwt_secret is required")
	}

	if ac.JWTSecret == "your-secret-key-change-in-production" {
		return fmt.Errorf("please set a secure JWT secret (current value is the default insecure value)")
	}

	// Validate JWT secret length (should be at least 32 characters for security)
	if len(ac.JWTSecret) < 32 {
		log.Warn().Msg("JWT secret is shorter than 32 characters - consider using a longer secret for better security")
	}

	// SECURITY: Validate JWT secret entropy to prevent weak secrets
	// Calculate Shannon entropy of the secret to ensure it has sufficient randomness
	entropy := calculateEntropy(ac.JWTSecret)
	// Minimum 4.5 bits per character Shannon entropy (catches repetitive patterns)
	// For reference: random alphanumeric = ~6 bits/char, all same = 0 bits, alternating = ~1 bit
	// 4.5 bits/char ensures good character variety without being overly strict
	minEntropyPerChar := 4.5
	if entropy < minEntropyPerChar {
		return fmt.Errorf("jwt_secret has insufficient entropy (%.2f bits < %.2f bits per character minimum). Generate a secure random secret: openssl rand -base64 32 | head -c 32", entropy, minEntropyPerChar)
	}

	// Validate expiry durations are positive
	if ac.JWTExpiry <= 0 {
		return fmt.Errorf("jwt_expiry must be positive, got: %v", ac.JWTExpiry)
	}
	if ac.RefreshExpiry <= 0 {
		return fmt.Errorf("refresh_expiry must be positive, got: %v", ac.RefreshExpiry)
	}
	if ac.MagicLinkExpiry <= 0 {
		return fmt.Errorf("magic_link_expiry must be positive, got: %v", ac.MagicLinkExpiry)
	}
	if ac.PasswordResetExpiry <= 0 {
		return fmt.Errorf("password_reset_expiry must be positive, got: %v", ac.PasswordResetExpiry)
	}

	// Validate password settings
	if ac.PasswordMinLen < 1 {
		return fmt.Errorf("password_min_length must be at least 1, got: %d", ac.PasswordMinLen)
	}
	if ac.PasswordMinLen < 8 {
		log.Warn().Int("min_length", ac.PasswordMinLen).Msg("Password minimum length is less than 8 - consider increasing for better security")
	}

	// Validate bcrypt cost (valid range is 4-31, recommended is 10-14)
	if ac.BcryptCost < 4 || ac.BcryptCost > 31 {
		return fmt.Errorf("bcrypt_cost must be between 4 and 31, got: %d", ac.BcryptCost)
	}

	// Validate OAuth providers
	providerNames := make(map[string]bool)
	for i, provider := range ac.OAuthProviders {
		if err := provider.Validate(); err != nil {
			return fmt.Errorf("oauth_providers[%d]: %w", i, err)
		}

		// Check for duplicate provider names
		if providerNames[provider.Name] {
			return fmt.Errorf("duplicate OAuth provider name: %s", provider.Name)
		}
		providerNames[provider.Name] = true
	}

	return nil
}

// Validate validates OAuth provider configuration
func (opc *OAuthProviderConfig) Validate() error {
	if opc.Name == "" {
		return fmt.Errorf("oauth provider name is required")
	}
	if opc.ClientID == "" {
		return fmt.Errorf("oauth provider '%s': client_id is required", opc.Name)
	}

	// Normalize name to lowercase
	opc.Name = strings.ToLower(opc.Name)

	// Check if well-known provider
	wellKnown := map[string]bool{
		"google":    true,
		"apple":     true,
		"microsoft": true,
	}

	// Custom providers require issuer_url
	if !wellKnown[opc.Name] && opc.IssuerURL == "" {
		return fmt.Errorf("oauth provider '%s': issuer_url is required for custom providers", opc.Name)
	}

	return nil
}
