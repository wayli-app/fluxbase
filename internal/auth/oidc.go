package auth

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/fluxbase-eu/fluxbase/internal/config"
	"github.com/rs/zerolog/log"
)

// Well-known OIDC issuer URLs
var wellKnownIssuers = map[string]string{
	"google":    "https://accounts.google.com",
	"apple":     "https://appleid.apple.com",
	"microsoft": "https://login.microsoftonline.com/common/v2.0",
}

// IDTokenClaims contains the claims extracted from an OIDC ID token
type IDTokenClaims struct {
	Subject       string // Provider's user ID (sub claim)
	Email         string
	EmailVerified bool
	Name          string
	Picture       string
	Nonce         string
}

// OIDCVerifier handles OIDC ID token verification for multiple providers
type OIDCVerifier struct {
	mu        sync.RWMutex
	verifiers map[string]*oidc.IDTokenVerifier
	providers map[string]*oidc.Provider
	clientIDs map[string]string
}

// NewOIDCVerifier creates a new OIDC verifier with the configured providers
func NewOIDCVerifier(ctx context.Context, cfg *config.AuthConfig) (*OIDCVerifier, error) {
	v := &OIDCVerifier{
		verifiers: make(map[string]*oidc.IDTokenVerifier),
		providers: make(map[string]*oidc.Provider),
		clientIDs: make(map[string]string),
	}

	// Initialize from unified oauth_providers array
	for _, providerCfg := range cfg.OAuthProviders {
		if !providerCfg.Enabled {
			log.Debug().Str("provider", providerCfg.Name).Msg("Skipping disabled OAuth provider")
			continue
		}

		// Determine issuer URL (auto-detect for well-known providers)
		issuerURL := providerCfg.IssuerURL
		if issuerURL == "" {
			if knownIssuer, ok := wellKnownIssuers[providerCfg.Name]; ok {
				issuerURL = knownIssuer
			} else {
				log.Warn().
					Str("provider", providerCfg.Name).
					Msg("Skipping provider: issuer_url not specified and not a well-known provider")
				continue
			}
		}

		if err := v.addProvider(ctx, providerCfg.Name, issuerURL, providerCfg.ClientID); err != nil {
			log.Warn().
				Err(err).
				Str("name", providerCfg.Name).
				Str("issuer", issuerURL).
				Msg("Failed to initialize OIDC provider")
		}
	}

	log.Info().
		Int("provider_count", len(v.verifiers)).
		Msg("OIDC verifier initialized")

	return v, nil
}

// addProvider adds an OIDC provider to the verifier
func (v *OIDCVerifier) addProvider(ctx context.Context, name, issuerURL, clientID string) error {
	provider, err := oidc.NewProvider(ctx, issuerURL)
	if err != nil {
		return fmt.Errorf("failed to create OIDC provider for %s: %w", name, err)
	}

	verifier := provider.Verifier(&oidc.Config{
		ClientID: clientID,
	})

	v.mu.Lock()
	v.providers[name] = provider
	v.verifiers[name] = verifier
	v.clientIDs[name] = clientID
	v.mu.Unlock()

	log.Info().
		Str("provider", name).
		Str("issuer", issuerURL).
		Msg("OIDC provider initialized")

	return nil
}

// Verify verifies an ID token from the specified provider and returns the claims
func (v *OIDCVerifier) Verify(ctx context.Context, providerName, idToken, expectedNonce string) (*IDTokenClaims, error) {
	name := strings.ToLower(providerName)

	v.mu.RLock()
	verifier, ok := v.verifiers[name]
	v.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("OIDC provider not configured: %s", providerName)
	}

	// Verify the ID token signature and standard claims
	token, err := verifier.Verify(ctx, idToken)
	if err != nil {
		return nil, fmt.Errorf("failed to verify ID token: %w", err)
	}

	// Extract claims
	var claims struct {
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
		Name          string `json:"name"`
		Picture       string `json:"picture"`
		Nonce         string `json:"nonce"`
	}

	if err := token.Claims(&claims); err != nil {
		return nil, fmt.Errorf("failed to extract claims: %w", err)
	}

	// Verify nonce if provided
	if expectedNonce != "" && claims.Nonce != expectedNonce {
		return nil, fmt.Errorf("nonce mismatch")
	}

	return &IDTokenClaims{
		Subject:       token.Subject,
		Email:         claims.Email,
		EmailVerified: claims.EmailVerified,
		Name:          claims.Name,
		Picture:       claims.Picture,
		Nonce:         claims.Nonce,
	}, nil
}

// IsProviderConfigured checks if a provider is configured
func (v *OIDCVerifier) IsProviderConfigured(providerName string) bool {
	name := strings.ToLower(providerName)
	v.mu.RLock()
	defer v.mu.RUnlock()
	_, ok := v.verifiers[name]
	return ok
}

// ListProviders returns a list of configured provider names
func (v *OIDCVerifier) ListProviders() []string {
	v.mu.RLock()
	defer v.mu.RUnlock()

	providers := make([]string, 0, len(v.verifiers))
	for name := range v.verifiers {
		providers = append(providers, name)
	}
	return providers
}
