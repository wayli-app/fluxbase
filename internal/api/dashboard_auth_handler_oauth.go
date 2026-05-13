package api

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"
	"golang.org/x/oauth2"

	"github.com/nimbleflux/fluxbase/internal/auth"
	"github.com/nimbleflux/fluxbase/internal/crypto"
	"github.com/nimbleflux/fluxbase/internal/database"
	"github.com/nimbleflux/fluxbase/internal/middleware"
)

// SSOProvider represents an SSO provider available for dashboard login
type SSOProvider struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Type     string `json:"type"`               // "oauth" or "saml"
	Provider string `json:"provider,omitempty"` // For OAuth: google, github, etc.
}

// GetSSOProviders returns the list of SSO providers available for dashboard login
func (h *DashboardAuthHandler) GetSSOProviders(c fiber.Ctx) error {
	ctx := c.RequestCtx()
	providers := []SSOProvider{}

	if err := h.requireDB(c); err != nil {
		return err
	}

	tenantID := middleware.GetTenantIDFromContext(c)
	oauthProviders, err := h.getOAuthProvidersForDashboard(ctx, tenantID)
	if err != nil {
		return SendInternalError(c, "Failed to fetch OAuth providers")
	}
	providers = append(providers, oauthProviders...)

	// Get SAML providers with allow_dashboard_login = true
	if h.samlService != nil {
		samlProviders := h.samlService.GetProvidersForDashboardWithTenant(c.RequestCtx(), middleware.GetTenantIDFromContext(c))
		for _, sp := range samlProviders {
			providers = append(providers, SSOProvider{
				ID:   sp.Name,
				Name: sp.Name,
				Type: "saml",
			})
		}
	}

	// Check if password login is disabled
	passwordLoginDisabled := h.isPasswordLoginDisabled(ctx)

	return c.JSON(fiber.Map{
		"providers":               providers,
		"password_login_disabled": passwordLoginDisabled,
	})
}

// getOAuthProvidersForDashboard fetches OAuth providers enabled for dashboard login
func (h *DashboardAuthHandler) getOAuthProvidersForDashboard(ctx context.Context, tenantID string) ([]SSOProvider, error) {
	providers := []SSOProvider{}

	err := database.WrapWithServiceRole(ctx, h.db, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT id, display_name, provider_name
		FROM platform.oauth_providers
		WHERE enabled = true AND allow_dashboard_login = true
		`)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var id uuid.UUID
			var displayName, providerName string
			if err := rows.Scan(&id, &displayName, &providerName); err != nil {
				return err
			}
			providers = append(providers, SSOProvider{
				ID:       providerName, // Use provider_name as ID for URL routing
				Name:     displayName,
				Type:     "oauth",
				Provider: providerName,
			})
		}
		return rows.Err()
	})
	if err != nil {
		return nil, err
	}

	return providers, nil
}

// InitiateOAuthLogin initiates an OAuth login flow for dashboard SSO
func (h *DashboardAuthHandler) InitiateOAuthLogin(c fiber.Ctx) error {
	providerID := c.Params("provider")
	redirectTo := c.Query("redirect_to", "/")
	ctx := c.RequestCtx()

	// Fetch the OAuth provider configuration
	var clientID, clientSecret, providerName string
	var scopes []string
	var isCustom bool
	var isEncrypted bool
	var authURL, tokenURL, userInfoURL *string
	err := database.WrapWithServiceRole(ctx, h.db, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `
			SELECT client_id, client_secret, provider_name, scopes,
			       is_custom, authorization_url, token_url, user_info_url,
			       COALESCE(is_encrypted, false) AS is_encrypted
		FROM platform.oauth_providers
		WHERE (id::text = $1 OR provider_name = $1) AND enabled = true AND allow_dashboard_login = true
		`, providerID).Scan(&clientID, &clientSecret, &providerName, &scopes, &isCustom, &authURL, &tokenURL, &userInfoURL, &isEncrypted)
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			log.Warn().
				Str("provider_id", providerID).
				Msg("OAuth provider not found or not enabled for dashboard login")
			return SendNotFound(c, "OAuth provider not found or not enabled for dashboard login")
		}
		log.Error().Err(err).Str("provider_id", providerID).Msg("Failed to fetch OAuth provider")
		return SendInternalError(c, "Failed to fetch OAuth provider")
	}

	log.Debug().
		Str("provider_id", providerID).
		Str("provider_name", providerName).
		Bool("is_custom", isCustom).
		Bool("has_auth_url", authURL != nil).
		Bool("has_token_url", tokenURL != nil).
		Msg("OAuth provider fetched for dashboard login")

	// Decrypt client secret if encrypted
	if isEncrypted && clientSecret != "" {
		decryptedSecret, decErr := crypto.DecryptWithBytesKey(clientSecret, h.encryptionKey)
		if decErr != nil {
			log.Error().Err(decErr).Str("provider", providerName).Msg("Failed to decrypt client secret")
			return SendInternalError(c, "Failed to decrypt client secret")
		}
		clientSecret = decryptedSecret
	}

	// Build OAuth config
	config := h.buildOAuthConfig(providerName, clientID, clientSecret, scopes, isCustom, authURL, tokenURL)
	if config == nil {
		log.Warn().
			Str("provider_name", providerName).
			Bool("is_custom", isCustom).
			Msg("Failed to build OAuth config - unsupported provider")
		return SendBadRequest(c, "Unsupported OAuth provider", ErrCodeInvalidInput)
	}

	// Generate state
	state, err := generateOAuthState()
	if err != nil {
		return SendInternalError(c, "Failed to generate state")
	}

	// Store state
	h.oauthStatesMu.Lock()
	h.oauthStates[state] = &dashboardOAuthState{
		Provider:    providerID,
		CreatedAt:   time.Now(),
		RedirectTo:  redirectTo,
		UserInfoURL: userInfoURL,
	}
	h.oauthStatesMu.Unlock()

	// Store config for callback
	h.oauthConfigsMu.Lock()
	h.oauthConfigs[state] = config
	h.oauthConfigsMu.Unlock()

	// Build auth URL options
	authURLOpts := []oauth2.AuthCodeOption{oauth2.AccessTypeOffline}

	// Add prompt=consent for Google to ensure refresh tokens on subsequent logins
	if strings.ToLower(providerName) == "google" {
		authURLOpts = append(authURLOpts, oauth2.SetAuthURLParam("prompt", "consent"))
	}

	// Redirect to OAuth provider
	authorizeURL := config.AuthCodeURL(state, authURLOpts...)

	log.Debug().
		Str("state", state).
		Str("provider", providerName).
		Str("authorize_url", authorizeURL).
		Msg("Dashboard OAuth login initiated")

	// Return JSON with authorization URL (client handles the redirect)
	return c.JSON(fiber.Map{
		"url":      authorizeURL,
		"provider": providerID,
	})
}

// buildOAuthConfig creates an OAuth2 config for the given provider
func (h *DashboardAuthHandler) buildOAuthConfig(provider, clientID, clientSecret string, scopes []string, isCustom bool, customAuthURL, customTokenURL *string) *oauth2.Config {
	callbackURL := h.baseURL + "/dashboard/auth/sso/oauth/" + provider + "/callback"

	var endpoint oauth2.Endpoint

	// If custom provider with URLs, use them
	if isCustom && customAuthURL != nil && customTokenURL != nil {
		endpoint = oauth2.Endpoint{
			AuthURL:  *customAuthURL,
			TokenURL: *customTokenURL,
		}
	} else {
		// Fall back to standard providers
		switch provider {
		case "google":
			endpoint = oauth2.Endpoint{
				AuthURL:  "https://accounts.google.com/o/oauth2/v2/auth",
				TokenURL: "https://oauth2.googleapis.com/token",
			}
			if len(scopes) == 0 {
				scopes = []string{"openid", "email", "profile"}
			}
		case "github":
			endpoint = oauth2.Endpoint{
				AuthURL:  "https://github.com/login/oauth/authorize",
				TokenURL: "https://github.com/login/oauth/access_token",
			}
			if len(scopes) == 0 {
				scopes = []string{"read:user", "user:email"}
			}
		case "microsoft":
			endpoint = oauth2.Endpoint{
				AuthURL:  "https://login.microsoftonline.com/common/oauth2/v2.0/authorize",
				TokenURL: "https://login.microsoftonline.com/common/oauth2/v2.0/token",
			}
			if len(scopes) == 0 {
				scopes = []string{"openid", "email", "profile", "offline_access"}
			}
		case "gitlab":
			endpoint = oauth2.Endpoint{
				AuthURL:  "https://gitlab.com/oauth/authorize",
				TokenURL: "https://gitlab.com/oauth/token",
			}
			if len(scopes) == 0 {
				scopes = []string{"read_user", "openid", "email", "offline_access"}
			}
		default:
			return nil
		}
	}

	return &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  callbackURL,
		Scopes:       scopes,
		Endpoint:     endpoint,
	}
}

// OAuthCallback handles the OAuth callback for dashboard SSO
func (h *DashboardAuthHandler) OAuthCallback(c fiber.Ctx) error {
	code := c.Query("code")
	state := c.Query("state")
	errorParam := c.Query("error")
	ctx := c.RequestCtx()

	codePreview := code
	if len(code) > 10 {
		codePreview = code[:10] + "..."
	}
	providerID := c.Params("provider")
	log.Debug().
		Str("state", state).
		Str("code", codePreview).
		Str("provider", providerID).
		Msg("Dashboard OAuth callback received")

	// Validate state from dashboard's own state store
	h.oauthStatesMu.Lock()
	dashState, stateExists := h.oauthStates[state]
	if stateExists {
		delete(h.oauthStates, state)
	}
	h.oauthStatesMu.Unlock()

	if !stateExists || dashState == nil {
		log.Warn().
			Str("state", state).
			Msg("Invalid or missing OAuth state in dashboard callback")
		return c.Redirect().To("/admin/login?error=" + url.QueryEscape("Invalid or expired state"))
	}

	// Retrieve stored OAuth config
	h.oauthConfigsMu.Lock()
	config, configExists := h.oauthConfigs[state]
	if configExists {
		delete(h.oauthConfigs, state)
	}
	h.oauthConfigsMu.Unlock()

	if !configExists || config == nil {
		log.Warn().
			Str("state", state).
			Msg("Missing OAuth config for dashboard callback")
		return c.Redirect().To("/admin/login?error=" + url.QueryEscape("OAuth configuration not found"))
	}

	// Verify provider matches the one from the initiation
	if providerID != "" && dashState.Provider != providerID {
		log.Warn().
			Str("url_provider", providerID).
			Str("state_provider", dashState.Provider).
			Msg("Provider mismatch in dashboard OAuth callback")
		return c.Redirect().To("/admin/login?error=" + url.QueryEscape("Provider mismatch"))
	}

	// This is a dashboard OAuth callback, process it
	if errorParam != "" {
		errorDesc := c.Query("error_description", errorParam)
		return c.Redirect().To("/admin/login?error=" + url.QueryEscape(errorDesc))
	}

	if code == "" {
		return c.Redirect().To("/admin/login?error=" + url.QueryEscape("Missing authorization code"))
	}

	userInfoURL := dashState.UserInfoURL

	// Log OAuth config details for debugging
	log.Debug().
		Str("provider", providerID).
		Str("redirect_uri", config.RedirectURL).
		Str("client_id", config.ClientID).
		Str("auth_url", config.Endpoint.AuthURL).
		Str("token_url", config.Endpoint.TokenURL).
		Msg("OAuth config for token exchange")

	// Exchange code for token
	token, err := config.Exchange(ctx, code)
	if err != nil {
		log.Error().
			Err(err).
			Str("provider", providerID).
			Str("redirect_uri", config.RedirectURL).
			Str("config_redirect_uri", config.RedirectURL).
			Msg("Failed to exchange OAuth authorization code")
		return c.Redirect().To("/admin/login?error=" + url.QueryEscape("Failed to exchange authorization code"))
	}

	// Fetch provider configuration for RBAC validation
	var requiredClaimsJSON, deniedClaimsJSON []byte
	var providerDisplayName string
	err = database.WrapWithServiceRole(ctx, h.db, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `
			SELECT display_name, required_claims, denied_claims
		FROM platform.oauth_providers
		WHERE (id::text = $1 OR provider_name = $1) AND enabled = true AND allow_dashboard_login = true
		`, providerID).Scan(&providerDisplayName, &requiredClaimsJSON, &deniedClaimsJSON)
	})
	if err != nil {
		log.Warn().
			Err(err).
			Str("provider", providerID).
			Msg("Failed to fetch OAuth provider config for RBAC validation")
		return c.Redirect().To("/admin/login?error=" + url.QueryEscape("OAuth provider configuration error"))
	}

	// Get user info from provider (includes ID token claims)
	userInfo, err := h.getUserInfoFromOAuth(ctx, config, token, userInfoURL)
	if err != nil {
		return c.Redirect().To("/admin/login?error=" + url.QueryEscape("Failed to get user info from provider"))
	}

	// Extract ID token claims (if available)
	var idTokenClaims map[string]interface{}
	if idTokenRaw, ok := token.Extra("id_token").(string); ok && idTokenRaw != "" {
		// Parse ID token (simple base64 decode of payload)
		idTokenClaims, err = parseIDTokenClaims(idTokenRaw)
		if err != nil {
			log.Warn().
				Err(err).
				Str("provider", providerID).
				Msg("Failed to parse ID token claims")
			// Use userInfo as fallback
			idTokenClaims = userInfo
		}
	} else {
		// Use userInfo as fallback if no ID token
		idTokenClaims = userInfo
	}

	// RBAC: Validate OAuth claims if configured
	if requiredClaimsJSON != nil || deniedClaimsJSON != nil {
		var requiredClaims, deniedClaims map[string][]string
		if requiredClaimsJSON != nil {
			if err := json.Unmarshal(requiredClaimsJSON, &requiredClaims); err != nil {
				log.Warn().Err(err).Msg("Failed to parse required_claims JSON")
			}
		}
		if deniedClaimsJSON != nil {
			if err := json.Unmarshal(deniedClaimsJSON, &deniedClaims); err != nil {
				log.Warn().Err(err).Msg("Failed to parse denied_claims JSON")
			}
		}

		provider := &auth.OAuthProviderRBAC{
			Name:           providerDisplayName,
			RequiredClaims: requiredClaims,
			DeniedClaims:   deniedClaims,
		}

		if err := auth.ValidateOAuthClaims(provider, idTokenClaims); err != nil {
			log.Warn().
				Err(err).
				Str("provider", providerID).
				Interface("claims", idTokenClaims).
				Msg("Dashboard OAuth access denied due to claims validation")
			return c.Redirect().To("/admin/login?error=" + url.QueryEscape(err.Error()))
		}
	}

	email, _ := userInfo["email"].(string)
	name, _ := userInfo["name"].(string)
	// Capitalize the first letter of each word in the name
	name = capitalizeWords(name)
	providerUserID, _ := userInfo["id"].(string)
	if providerUserID == "" {
		// Some providers use "sub" instead of "id"
		providerUserID, _ = userInfo["sub"].(string)
	}

	if email == "" {
		return c.Redirect().To("/admin/login?error=" + url.QueryEscape("Email not provided by OAuth provider"))
	}

	// Find or create dashboard user
	providerName := "oauth:" + providerID
	user, _, err := h.authService.FindOrCreateUserBySSO(ctx, email, name, providerName, providerUserID)
	if err != nil {
		log.Error().
			Err(err).
			Str("email", email).
			Str("provider", providerName).
			Str("provider_user_id", providerUserID).
			Msg("Failed to create or find dashboard user via SSO")
		return c.Redirect().To("/admin/login?error=" + url.QueryEscape("Failed to create or find user"))
	}

	// Login via SSO
	ipAddress := getIPAddress(c)
	userAgent := string(c.Request().Header.UserAgent())
	loginResp, err := h.authService.LoginViaSSO(ctx, user, ipAddress, userAgent)
	if err != nil {
		errMsg := "Login failed"
		if err.Error() == "account is locked" {
			errMsg = "Account is locked"
		} else if err.Error() == "account is inactive" {
			errMsg = "Account is inactive"
		}
		return c.Redirect().To("/admin/login?error=" + url.QueryEscape(errMsg))
	}

	// Redirect with tokens in URL fragment (for SPA to capture)
	redirectURL := dashState.RedirectTo
	if redirectURL == "" || redirectURL == "/" {
		redirectURL = "/admin"
	}
	return c.Redirect().To(fmt.Sprintf("/admin/login/callback#access_token=%s&refresh_token=%s&redirect_to=%s",
		url.QueryEscape(loginResp.AccessToken),
		url.QueryEscape(loginResp.RefreshToken),
		url.QueryEscape(redirectURL)))
}

// parseIDTokenClaims parses JWT ID token and extracts claims
// This is a simple implementation without signature verification (already verified by OAuth provider)
func parseIDTokenClaims(idToken string) (map[string]interface{}, error) {
	parts := strings.Split(idToken, ".")
	if len(parts) != 3 {
		return nil, errors.New("invalid ID token format")
	}

	// Decode payload (second part)
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("failed to decode ID token payload: %w", err)
	}

	var claims map[string]interface{}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, fmt.Errorf("failed to unmarshal ID token claims: %w", err)
	}

	return claims, nil
}

// getUserInfoFromOAuth fetches user info from OAuth provider
func (h *DashboardAuthHandler) getUserInfoFromOAuth(ctx context.Context, config *oauth2.Config, token *oauth2.Token, customUserInfoURL *string) (map[string]interface{}, error) {
	client := config.Client(ctx, token)

	// Determine user info URL - use custom URL if provided, otherwise use standard provider URLs
	var userInfoURL string
	if customUserInfoURL != nil && *customUserInfoURL != "" {
		userInfoURL = *customUserInfoURL
	} else {
		switch {
		case strings.Contains(config.Endpoint.AuthURL, "google"):
			userInfoURL = "https://www.googleapis.com/oauth2/v2/userinfo"
		case strings.Contains(config.Endpoint.AuthURL, "github"):
			userInfoURL = "https://api.github.com/user"
		case strings.Contains(config.Endpoint.AuthURL, "microsoft"):
			userInfoURL = "https://graph.microsoft.com/v1.0/me"
		case strings.Contains(config.Endpoint.AuthURL, "gitlab"):
			userInfoURL = "https://gitlab.com/api/v4/user"
		default:
			return nil, errors.New("unsupported provider")
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, userInfoURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var userInfo map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, err
	}

	// For GitHub, we need to fetch email separately if not in profile
	if strings.Contains(config.Endpoint.AuthURL, "github") {
		if _, ok := userInfo["email"]; !ok || userInfo["email"] == nil {
			emailReq, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/user/emails", nil)
			if err == nil {
				emailResp, err := client.Do(emailReq)
				if err == nil {
					defer func() { _ = emailResp.Body.Close() }()
					var emails []map[string]interface{}
					if err := json.NewDecoder(emailResp.Body).Decode(&emails); err == nil {
						for _, e := range emails {
							if primary, ok := e["primary"].(bool); ok && primary {
								userInfo["email"] = e["email"]
								break
							}
						}
					}
				}
			}
		}
	}

	return userInfo, nil
}

// generateOAuthState generates a random state string for OAuth
func generateOAuthState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
