package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog/log"
	"golang.org/x/oauth2"

	"github.com/nimbleflux/fluxbase/internal/auth"
	"github.com/nimbleflux/fluxbase/internal/crypto"
	"github.com/nimbleflux/fluxbase/internal/database"
	"github.com/nimbleflux/fluxbase/internal/middleware"
)

// getProviderConfig retrieves OAuth configuration from database
// Supports tenant-specific providers with fallback to platform-level providers
func (h *OAuthHandler) getProviderConfig(ctx context.Context, providerName string, tenantID string) (*oauth2.Config, error) {
	// SECURITY: Only allow providers that enable app login
	// Priority: tenant-specific provider > platform-level provider

	var clientID, clientSecret, redirectURL string
	var scopes []string
	var authURL, tokenURL *string
	var isCustom bool
	var allowAppLogin bool
	var isEncrypted bool

	// Priority: tenant-specific provider > platform-level provider
	query := `
		SELECT client_id, client_secret, redirect_url, scopes,
		       authorization_url, token_url, is_custom, allow_app_login,
		       COALESCE(is_encrypted, false) AS is_encrypted
		FROM platform.oauth_providers
		WHERE provider_name = $1 AND enabled = TRUE
		  AND (tenant_id = $2::uuid OR tenant_id IS NULL)
		ORDER BY tenant_id IS NULL
		LIMIT 1
	`
	var tenantUUID interface{}
	if tenantID != "" {
		tenantUUID = tenantID
	}
	err := h.db.QueryRow(ctx, query, providerName, tenantUUID).Scan(
		&clientID, &clientSecret, &redirectURL, &scopes,
		&authURL, &tokenURL, &isCustom, &allowAppLogin, &isEncrypted,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("OAuth provider '%s' not found or disabled", providerName)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query OAuth provider: %w", err)
	}
	// SECURITY: Validate that provider allows app login
	if !allowAppLogin {
		return nil, fmt.Errorf("OAuth provider '%s' not enabled for application login", providerName)
	}

	// Decrypt client secret if encrypted
	if isEncrypted && clientSecret != "" {
		decryptedSecret, decErr := crypto.DecryptWithBytesKey(clientSecret, h.encryptionKey)
		if decErr != nil {
			log.Error().Err(decErr).Str("provider", providerName).Msg("Failed to decrypt client secret")
			return nil, fmt.Errorf("failed to decrypt client secret for provider '%s'", providerName)
		}
		clientSecret = decryptedSecret
	}

	// Build OAuth2 config
	config := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes:       scopes,
	}

	// Set endpoint based on provider type
	if isCustom && authURL != nil && tokenURL != nil {
		config.Endpoint = oauth2.Endpoint{
			AuthURL:  *authURL,
			TokenURL: *tokenURL,
		}
	} else {
		config.Endpoint = h.getStandardEndpoint(providerName)
	}

	return config, nil
}

// getStandardEndpoint returns OAuth endpoints for standard providers
func (h *OAuthHandler) getStandardEndpoint(providerName string) oauth2.Endpoint {
	manager := auth.NewOAuthManager()
	return manager.GetEndpoint(auth.OAuthProvider(providerName))
}

// getUserInfo retrieves user information from OAuth provider
func (h *OAuthHandler) getUserInfo(ctx context.Context, providerName string, config *oauth2.Config, token *oauth2.Token) (map[string]interface{}, error) {
	client := config.Client(ctx, token)

	// Get user info URL from database
	var userInfoURL *string
	query := "SELECT user_info_url FROM platform.oauth_providers WHERE provider_name = $1"
	err := h.db.QueryRow(ctx, query, providerName).Scan(&userInfoURL)

	if err != nil || userInfoURL == nil {
		// Use default URL for standard providers
		manager := auth.NewOAuthManager()
		url := manager.GetUserInfoURL(auth.OAuthProvider(providerName))
		userInfoURL = &url
	}

	// Fetch user info
	req, err := http.NewRequestWithContext(ctx, "GET", *userInfoURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user info: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("user info endpoint returned status %d", resp.StatusCode)
	}

	var userInfo map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	return userInfo, nil
}

// extractEmail extracts email from OAuth user info
func (h *OAuthHandler) extractEmail(providerName string, userInfo map[string]interface{}) string {
	// Most providers use "email" field
	if email, ok := userInfo["email"].(string); ok && email != "" {
		return email
	}

	// GitHub may not provide email
	if providerName == "github" {
		if login, ok := userInfo["login"].(string); ok {
			return fmt.Sprintf("%s@users.noreply.github.com", login)
		}
	}

	return ""
}

// extractProviderUserID extracts provider user ID from OAuth user info
func (h *OAuthHandler) extractProviderUserID(providerName string, userInfo map[string]interface{}) string {
	// Try "id" field (most common)
	if id, ok := userInfo["id"].(string); ok {
		return id
	}

	// Try numeric ID (GitHub, Facebook)
	if id, ok := userInfo["id"].(float64); ok {
		return fmt.Sprintf("%.0f", id)
	}

	// Try "sub" field (OIDC standard)
	if sub, ok := userInfo["sub"].(string); ok {
		return sub
	}

	return ""
}

// ProviderTokenResponse represents the response for getting provider tokens
type ProviderTokenResponse struct {
	Provider     string   `json:"provider"`
	AccessToken  string   `json:"access_token"`
	RefreshToken string   `json:"refresh_token,omitempty"`
	TokenExpiry  string   `json:"token_expiry"`
	ExpiresIn    int      `json:"expires_in"`
	IDToken      string   `json:"id_token,omitempty"`
	Scopes       []string `json:"scopes,omitempty"`
	TokenType    string   `json:"token_type"`
}

// GetProviderToken retrieves the OAuth provider tokens for the authenticated user
// This endpoint allows users to retrieve their stored OAuth tokens to make API calls
// to the provider (e.g., Google Drive API).
// GET /api/v1/auth/oauth/:provider/token
func (h *OAuthHandler) GetProviderToken(c fiber.Ctx) error {
	ctx := c.RequestCtx()
	providerName := c.Params("provider")

	userIDStr := middleware.GetUserID(c)
	if userIDStr == "" {
		return SendUnauthorized(c, "Authentication required", "AUTH_REQUIRED")
	}

	if err := h.requireDB(c); err != nil {
		return err
	}

	if err := h.requireLogoutService(c); err != nil {
		return err
	}

	oauthConfig, err := h.getProviderConfigForToken(ctx, providerName)
	if err != nil {
		log.Error().Err(err).Str("provider", providerName).Msg("Failed to get OAuth provider config for token retrieval")
		return SendBadRequest(c, fmt.Sprintf("OAuth provider '%s' not configured or disabled", providerName), "PROVIDER_NOT_CONFIGURED")
	}

	storedToken, err := h.logoutService.GetUserOAuthToken(ctx, userIDStr, providerName)
	if err != nil {
		if errors.Is(err, auth.ErrOAuthTokenNotFound) {
			return SendErrorWithDetails(c, fiber.StatusNotFound,
				"No OAuth token found for this provider", "OAUTH_TOKEN_NOT_FOUND",
				"You need to sign in with this provider first", "",
				fiber.Map{
					"provider":      providerName,
					"authorize_url": fmt.Sprintf("%s/api/v1/auth/oauth/%s/authorize", h.baseURL, providerName),
				})
		}
		log.Error().Err(err).Str("provider", providerName).Str("user_id", userIDStr).Msg("Failed to get stored OAuth token")
		return SendInternalError(c, "Failed to retrieve OAuth token")
	}

	accessToken := storedToken.AccessToken
	refreshToken := storedToken.RefreshToken
	idToken := storedToken.IDToken

	if len(h.encryptionKey) > 0 {
		if accessToken != "" {
			decrypted, decErr := crypto.DecryptWithBytesKey(accessToken, h.encryptionKey)
			if decErr == nil {
				accessToken = decrypted
			} else {
				log.Warn().Err(decErr).Str("provider", providerName).Msg("Failed to decrypt access token")
			}
		}
		if refreshToken != "" {
			decrypted, decErr := crypto.DecryptWithBytesKey(refreshToken, h.encryptionKey)
			if decErr == nil {
				refreshToken = decrypted
			}
		}
		if idToken != "" {
			decrypted, decErr := crypto.DecryptWithBytesKey(idToken, h.encryptionKey)
			if decErr == nil {
				idToken = decrypted
			}
		}
	}

	tokenExpiry := storedToken.TokenExpiry
	needsRefresh := !tokenExpiry.IsZero() && time.Now().After(tokenExpiry.Add(-5*time.Minute))

	if needsRefresh && refreshToken != "" {
		log.Info().Str("provider", providerName).Str("user_id", userIDStr).Msg("OAuth token expired or expiring soon, attempting refresh")

		token := &oauth2.Token{
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
			Expiry:       tokenExpiry,
		}
		if idToken != "" {
			token = token.WithExtra(map[string]interface{}{"id_token": idToken})
		}

		newToken, refreshErr := oauthConfig.TokenSource(ctx, token).Token()
		if refreshErr != nil {
			log.Warn().Err(refreshErr).Str("provider", providerName).Str("user_id", userIDStr).Msg("Failed to refresh OAuth token, returning existing token")
		} else {
			accessToken = newToken.AccessToken
			refreshToken = newToken.RefreshToken
			tokenExpiry = newToken.Expiry
			if rawIDToken, ok := newToken.Extra("id_token").(string); ok {
				idToken = rawIDToken
			}

			go func() {
				refreshCtx := context.Background()
				if tid := middleware.GetTenantIDFromContext(c); tid != "" {
					refreshCtx = database.ContextWithTenant(refreshCtx, tid)
				}
				accessTokenToStore := newToken.AccessToken
				refreshTokenToStore := newToken.RefreshToken
				idTokenToStore := idToken

				if len(h.encryptionKey) > 0 {
					var encErr error
					accessTokenToStore, encErr = crypto.EncryptIfNotEmptyWithBytesKey(newToken.AccessToken, h.encryptionKey)
					if encErr != nil {
						log.Warn().Err(encErr).Str("provider", providerName).Msg("Failed to encrypt refreshed access token")
						return
					}
					refreshTokenToStore, encErr = crypto.EncryptIfNotEmptyWithBytesKey(newToken.RefreshToken, h.encryptionKey)
					if encErr != nil {
						log.Warn().Err(encErr).Str("provider", providerName).Msg("Failed to encrypt refreshed refresh token")
						return
					}
					idTokenToStore, encErr = crypto.EncryptIfNotEmptyWithBytesKey(idTokenToStore, h.encryptionKey)
					if encErr != nil {
						log.Warn().Err(encErr).Str("provider", providerName).Msg("Failed to encrypt refreshed id token")
						return
					}
				}

				_, err := h.db.Exec(refreshCtx, `
					UPDATE auth.oauth_tokens
					SET access_token = $1, refresh_token = $2, id_token = $3, token_expiry = $4, updated_at = CURRENT_TIMESTAMP
					WHERE user_id = $5 AND provider = $6
				`, accessTokenToStore, refreshTokenToStore, idTokenToStore, newToken.Expiry, userIDStr, providerName)
				if err != nil {
					log.Warn().Err(err).Str("provider", providerName).Str("user_id", userIDStr).Msg("Failed to update refreshed OAuth token in database")
				} else {
					log.Info().Str("provider", providerName).Str("user_id", userIDStr).Msg("OAuth token refreshed and updated in database")
				}
			}()
		}
	}

	expiresIn := 0
	if !tokenExpiry.IsZero() {
		expiresIn = int(time.Until(tokenExpiry).Seconds())
		if expiresIn < 0 {
			expiresIn = 0
		}
	}

	var scopes []string
	if oauthConfig.Scopes != nil {
		scopes = oauthConfig.Scopes
	}

	response := ProviderTokenResponse{
		Provider:     providerName,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenExpiry:  tokenExpiry.UTC().Format(time.RFC3339),
		ExpiresIn:    expiresIn,
		IDToken:      idToken,
		Scopes:       scopes,
		TokenType:    "Bearer",
	}

	log.Info().
		Str("provider", providerName).
		Str("user_id", userIDStr).
		Bool("was_refreshed", needsRefresh).
		Msg("OAuth provider token retrieved")

	return c.JSON(response)
}

// getProviderConfigForToken retrieves OAuth configuration for token operations
// Unlike getProviderConfig, this doesn't require allow_app_login to be true
// since the user already has a stored token from a previous OAuth flow
func (h *OAuthHandler) getProviderConfigForToken(ctx context.Context, providerName string) (*oauth2.Config, error) {
	query := `
		SELECT client_id, client_secret, redirect_url, scopes,
		       authorization_url, token_url, is_custom,
		       COALESCE(is_encrypted, false) AS is_encrypted
		FROM platform.oauth_providers
		WHERE provider_name = $1 AND enabled = TRUE
	`

	var clientID, clientSecret, redirectURL string
	var scopes []string
	var authURL, tokenURL *string
	var isCustom bool
	var isEncrypted bool

	err := h.db.QueryRow(ctx, query, providerName).Scan(
		&clientID, &clientSecret, &redirectURL, &scopes,
		&authURL, &tokenURL, &isCustom, &isEncrypted,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("OAuth provider '%s' not found or disabled", providerName)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query OAuth provider: %w", err)
	}

	if isEncrypted && clientSecret != "" {
		decryptedSecret, decErr := crypto.DecryptWithBytesKey(clientSecret, h.encryptionKey)
		if decErr != nil {
			log.Error().Err(decErr).Str("provider", providerName).Msg("Failed to decrypt client secret")
			return nil, fmt.Errorf("failed to decrypt client secret for provider '%s'", providerName)
		}
		clientSecret = decryptedSecret
	}

	config := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes:       scopes,
	}

	if isCustom && authURL != nil && tokenURL != nil {
		config.Endpoint = oauth2.Endpoint{
			AuthURL:  *authURL,
			TokenURL: *tokenURL,
		}
	} else {
		config.Endpoint = h.getStandardEndpoint(providerName)
	}

	return config, nil
}

// Logout initiates OAuth Single Logout
// POST /api/v1/auth/oauth/:provider/logout
func (h *OAuthHandler) Logout(c fiber.Ctx) error {
	ctx := c.RequestCtx()
	providerName := c.Params("provider")

	// Get user ID from JWT
	userIDStr := middleware.GetUserID(c)
	if userIDStr == "" {
		return SendUnauthorized(c, "Authentication required", "AUTH_REQUIRED")
	}

	var reqBody struct {
		RedirectURL string `json:"redirect_url"`
	}
	_ = c.Bind().Body(&reqBody)

	if err := h.requireDB(c); err != nil {
		return err
	}

	if err := h.requireLogoutService(c); err != nil {
		return err
	}

	if err := h.requireAuthService(c); err != nil {
		return err
	}

	var revocationEndpoint, endSessionEndpoint, clientID, clientSecret *string
	var isEncrypted bool
	err := h.db.QueryRow(ctx, `
		SELECT client_id, client_secret, revocation_endpoint, end_session_endpoint,
		       COALESCE(is_encrypted, false) AS is_encrypted
		FROM platform.oauth_providers
		WHERE provider_name = $1 AND enabled = TRUE
	`, providerName).Scan(&clientID, &clientSecret, &revocationEndpoint, &endSessionEndpoint, &isEncrypted)
	if err != nil {
		log.Error().Err(err).Str("provider", providerName).Msg("Failed to get OAuth provider for logout")
		return SendBadRequest(c, fmt.Sprintf("OAuth provider '%s' not found or disabled", providerName), "PROVIDER_NOT_FOUND")
	}

	// Use default endpoints if not configured
	if revocationEndpoint == nil || *revocationEndpoint == "" {
		defaultEndpoint := auth.GetDefaultRevocationEndpoint(auth.OAuthProvider(providerName))
		revocationEndpoint = &defaultEndpoint
	}
	if endSessionEndpoint == nil || *endSessionEndpoint == "" {
		defaultEndpoint := auth.GetDefaultEndSessionEndpoint(auth.OAuthProvider(providerName))
		endSessionEndpoint = &defaultEndpoint
	}

	// Decrypt client secret if encrypted
	clientSecretDecrypted := ""
	if clientSecret != nil && *clientSecret != "" {
		if isEncrypted && len(h.encryptionKey) > 0 {
			decrypted, err := crypto.DecryptWithBytesKey(*clientSecret, h.encryptionKey)
			if err != nil {
				log.Warn().Err(err).Msg("Failed to decrypt client secret for logout")
			} else {
				clientSecretDecrypted = decrypted
			}
		} else {
			clientSecretDecrypted = *clientSecret
		}
	}

	result := &auth.OAuthLogoutResult{
		Provider:             providerName,
		LocalLogoutComplete:  false,
		ProviderTokenRevoked: false,
		RequiresRedirect:     false,
	}

	// Get user's stored OAuth token
	storedToken, err := h.logoutService.GetUserOAuthToken(ctx, userIDStr, providerName)
	if err != nil {
		log.Warn().Err(err).Str("provider", providerName).Str("user_id", userIDStr).Msg("No OAuth token found for logout")
		// Continue with local logout even if no token found
	}

	// Try to revoke token at provider (RFC 7009)
	if storedToken != nil && revocationEndpoint != nil && *revocationEndpoint != "" {
		// Decrypt access token if encrypted
		accessToken := storedToken.AccessToken
		if len(h.encryptionKey) > 0 && accessToken != "" {
			decrypted, err := crypto.DecryptWithBytesKey(accessToken, h.encryptionKey)
			if err == nil {
				accessToken = decrypted
			}
		}

		if accessToken != "" && clientID != nil {
			err = h.logoutService.RevokeTokenAtProvider(ctx, *revocationEndpoint, accessToken, "access_token", *clientID, clientSecretDecrypted)
			if err != nil {
				log.Warn().Err(err).Str("provider", providerName).Msg("Failed to revoke token at provider")
				result.Warning = "Token revocation at provider failed"
			} else {
				result.ProviderTokenRevoked = true
				log.Info().Str("provider", providerName).Str("user_id", userIDStr).Msg("OAuth token revoked at provider")
			}
		}
	}

	// Generate OIDC logout URL if provider supports it
	if endSessionEndpoint != nil && *endSessionEndpoint != "" {
		// Generate state for CSRF protection
		state, err := auth.GenerateLogoutState()
		if err != nil {
			log.Error().Err(err).Msg("Failed to generate logout state")
		} else {
			// Determine post-logout redirect URI
			postLogoutRedirectURI := reqBody.RedirectURL
			if postLogoutRedirectURI == "" {
				postLogoutRedirectURI = fmt.Sprintf("%s/api/v1/auth/oauth/%s/logout/callback", h.baseURL, providerName)
			}

			// Store logout state for callback validation
			err = h.logoutService.StoreLogoutState(ctx, userIDStr, providerName, state, postLogoutRedirectURI)
			if err != nil {
				log.Error().Err(err).Msg("Failed to store logout state")
			} else {
				// Get ID token for id_token_hint
				idToken := ""
				if storedToken != nil && storedToken.IDToken != "" {
					idToken = storedToken.IDToken
					// Decrypt if encrypted
					if len(h.encryptionKey) > 0 {
						decrypted, err := crypto.DecryptWithBytesKey(idToken, h.encryptionKey)
						if err == nil {
							idToken = decrypted
						}
					}
				}

				// Generate logout URL
				logoutURL, err := h.logoutService.GenerateOIDCLogoutURL(*endSessionEndpoint, idToken, postLogoutRedirectURI, state)
				if err != nil {
					log.Warn().Err(err).Msg("Failed to generate OIDC logout URL")
				} else {
					result.RequiresRedirect = true
					result.RedirectURL = logoutURL
				}
			}
		}
	}

	// Revoke local JWT tokens
	if err := h.authSvc.RevokeAllUserTokens(ctx, userIDStr, "OAuth logout"); err != nil {
		log.Error().Err(err).Str("user_id", userIDStr).Msg("Failed to revoke local tokens")
	} else {
		result.LocalLogoutComplete = true
	}

	// Delete stored OAuth token
	if err := h.logoutService.DeleteUserOAuthToken(ctx, userIDStr, providerName); err != nil {
		log.Warn().Err(err).Str("provider", providerName).Msg("Failed to delete stored OAuth token")
	}

	log.Info().
		Str("provider", providerName).
		Str("user_id", userIDStr).
		Bool("local_logout", result.LocalLogoutComplete).
		Bool("provider_revoked", result.ProviderTokenRevoked).
		Bool("requires_redirect", result.RequiresRedirect).
		Msg("OAuth logout completed")

	return c.JSON(result)
}

// LogoutCallback handles the callback after OIDC logout
// GET /api/v1/auth/oauth/:provider/logout/callback
func (h *OAuthHandler) LogoutCallback(c fiber.Ctx) error {
	ctx := c.RequestCtx()
	providerName := c.Params("provider")
	state := c.Query("state")

	if state == "" {
		log.Warn().Str("provider", providerName).Msg("OAuth logout callback missing state parameter")
		return SendBadRequest(c, "Missing state parameter", "MISSING_STATE")
	}

	if err := h.requireLogoutService(c); err != nil {
		return err
	}

	logoutState, err := h.logoutService.ValidateLogoutState(ctx, state)
	if err != nil {
		log.Warn().Err(err).Str("provider", providerName).Str("state", state).Msg("Invalid or expired logout state")
		return SendBadRequest(c, "Invalid or expired logout state", "INVALID_LOGOUT_STATE")
	}

	log.Info().
		Str("provider", providerName).
		Str("user_id", logoutState.UserID).
		Msg("OAuth logout callback successful")

	// Redirect to the post-logout redirect URI if specified
	if logoutState.PostLogoutRedirectURI != "" && logoutState.PostLogoutRedirectURI != c.OriginalURL() {
		return c.Redirect().To(logoutState.PostLogoutRedirectURI)
	}

	return c.JSON(fiber.Map{
		"message":  "Logout successful",
		"provider": providerName,
	})
}
