package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/fluxbase-eu/fluxbase/internal/auth"
	"github.com/fluxbase-eu/fluxbase/internal/config"
	"github.com/fluxbase-eu/fluxbase/internal/crypto"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
	"golang.org/x/oauth2"
)

// OAuthHandler handles OAuth authentication flow
type OAuthHandler struct {
	db              *pgxpool.Pool
	authSvc         *auth.Service
	jwtManager      *auth.JWTManager
	stateStore      *auth.StateStore
	logoutService   *auth.OAuthLogoutService
	baseURL         string
	encryptionKey   string                       // SECURITY: Used for AES-256-GCM encryption of OAuth tokens at rest
	configProviders []config.OAuthProviderConfig // OAuth providers from config file
}

// NewOAuthHandler creates a new OAuth handler
func NewOAuthHandler(db *pgxpool.Pool, authSvc *auth.Service, jwtManager *auth.JWTManager, baseURL, encryptionKey string, configProviders []config.OAuthProviderConfig) *OAuthHandler {
	stateStore := auth.NewStateStore()

	// Warn if encryption key is not set (OAuth tokens will be stored unencrypted)
	if encryptionKey == "" {
		log.Warn().Msg("FLUXBASE_ENCRYPTION_KEY not set: OAuth tokens will be stored unencrypted")
	} else if len(encryptionKey) != 32 {
		log.Warn().Msg("FLUXBASE_ENCRYPTION_KEY must be exactly 32 bytes for AES-256: OAuth tokens will be stored unencrypted")
		encryptionKey = "" // Clear invalid key
	}

	// Start cleanup goroutine for expired states
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			stateStore.Cleanup()
		}
	}()

	// Create logout service
	logoutService := auth.NewOAuthLogoutService(db, encryptionKey)

	// Start cleanup goroutine for expired logout states
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			if err := logoutService.CleanupExpiredLogoutStates(context.Background()); err != nil {
				log.Warn().Err(err).Msg("Failed to cleanup expired OAuth logout states")
			}
		}
	}()

	return &OAuthHandler{
		db:              db,
		authSvc:         authSvc,
		jwtManager:      jwtManager,
		stateStore:      stateStore,
		logoutService:   logoutService,
		baseURL:         baseURL,
		encryptionKey:   encryptionKey,
		configProviders: configProviders,
	}
}

// Authorize initiates the OAuth flow
// GET /api/v1/auth/oauth/:provider/authorize
func (h *OAuthHandler) Authorize(c *fiber.Ctx) error {
	ctx := c.Context()
	providerName := c.Params("provider")

	// Get optional redirect_uri parameter for custom callback URL
	redirectURI := c.Query("redirect_uri")

	// Get OAuth provider configuration from database
	oauthConfig, err := h.getProviderConfig(ctx, providerName)
	if err != nil {
		log.Error().Err(err).Str("provider", providerName).Msg("Failed to get OAuth provider config")
		return c.Status(400).JSON(fiber.Map{
			"error": fmt.Sprintf("OAuth provider '%s' not configured or disabled", providerName),
		})
	}

	// Override redirect URL if custom redirect_uri is provided
	if redirectURI != "" {
		// Build full URL if relative path is provided
		if redirectURI[0] == '/' {
			redirectURI = h.baseURL + redirectURI
		}
		oauthConfig.RedirectURL = redirectURI
	}

	// Generate state for CSRF protection
	state, err := auth.GenerateState()
	if err != nil {
		log.Error().Err(err).Msg("Failed to generate OAuth state")
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to initiate OAuth flow",
		})
	}

	// Store state with optional redirect URI for callback validation
	h.stateStore.Set(state, redirectURI)

	// Generate authorization URL
	authURL := oauthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline)

	log.Info().
		Str("provider", providerName).
		Str("state", state).
		Str("redirect_uri", redirectURI).
		Msg("OAuth authorization initiated")

	// Return JSON with authorization URL (SDK handles the redirect)
	return c.JSON(fiber.Map{
		"url":      authURL,
		"provider": providerName,
	})
}

// Callback handles the OAuth callback
// GET /api/v1/auth/oauth/:provider/callback
func (h *OAuthHandler) Callback(c *fiber.Ctx) error {
	ctx := c.Context()
	providerName := c.Params("provider")
	code := c.Query("code")
	state := c.Query("state")
	errorParam := c.Query("error")

	// Check for OAuth errors
	if errorParam != "" {
		errorDesc := c.Query("error_description", errorParam)
		log.Warn().
			Str("provider", providerName).
			Str("error", errorParam).
			Str("description", errorDesc).
			Msg("OAuth provider returned error")

		return c.Status(400).JSON(fiber.Map{
			"error":       "OAuth authentication failed",
			"description": errorDesc,
		})
	}

	// Validate state and retrieve metadata
	stateMetadata, valid := h.stateStore.GetAndValidate(state)
	if !valid {
		log.Warn().Str("provider", providerName).Str("state", state).Msg("Invalid OAuth state")
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid OAuth state parameter",
		})
	}

	// Get OAuth provider configuration
	oauthConfig, err := h.getProviderConfig(ctx, providerName)
	if err != nil {
		log.Error().Err(err).Str("provider", providerName).Msg("Failed to get OAuth provider config")
		return c.Status(400).JSON(fiber.Map{
			"error": "OAuth provider not configured",
		})
	}

	// Determine redirect_uri to use (query parameter takes precedence over state metadata for SDK compatibility)
	redirectURIParam := c.Query("redirect_uri")
	var finalRedirectURI string

	if redirectURIParam != "" {
		// SDK passed redirect_uri as query parameter
		finalRedirectURI = redirectURIParam
	} else if stateMetadata.RedirectURI != "" {
		// Use redirect_uri from state metadata (from authorize request)
		finalRedirectURI = stateMetadata.RedirectURI
	}

	// Override redirect URL if custom redirect_uri was provided
	if finalRedirectURI != "" {
		// Build full URL if relative path is provided
		if finalRedirectURI[0] == '/' {
			finalRedirectURI = h.baseURL + finalRedirectURI
		}
		oauthConfig.RedirectURL = finalRedirectURI
	}

	// Exchange code for token
	token, err := oauthConfig.Exchange(ctx, code)
	if err != nil {
		log.Error().Err(err).Str("provider", providerName).Msg("Failed to exchange OAuth code")
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to complete OAuth authentication",
		})
	}

	// Get user info from OAuth provider
	userInfo, err := h.getUserInfo(ctx, providerName, oauthConfig, token)
	if err != nil {
		log.Error().Err(err).Str("provider", providerName).Msg("Failed to get user info from OAuth provider")
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to retrieve user information",
		})
	}

	// Extract email and provider user ID
	email := h.extractEmail(providerName, userInfo)
	providerUserID := h.extractProviderUserID(providerName, userInfo)

	if email == "" || providerUserID == "" {
		log.Error().
			Str("provider", providerName).
			Interface("userInfo", userInfo).
			Msg("Missing required user information from OAuth provider")
		return c.Status(500).JSON(fiber.Map{
			"error": "OAuth provider did not return required user information",
		})
	}

	// RBAC: Fetch provider RBAC config and validate claims if configured (OPTIONAL for app users)
	var requiredClaimsJSON, deniedClaimsJSON []byte
	err = h.db.QueryRow(ctx, `
		SELECT required_claims, denied_claims
		FROM dashboard.oauth_providers
		WHERE provider_name = $1 AND enabled = TRUE AND allow_app_login = TRUE
	`, providerName).Scan(&requiredClaimsJSON, &deniedClaimsJSON)

	if err != nil && err.Error() != "no rows in result set" {
		log.Warn().Err(err).Msg("Failed to fetch OAuth provider RBAC config")
		// Continue without RBAC validation
	}

	// Extract and validate ID token claims if RBAC is configured
	if requiredClaimsJSON != nil || deniedClaimsJSON != nil {
		// Extract ID token claims
		var idTokenClaims map[string]interface{}
		if idTokenRaw, ok := token.Extra("id_token").(string); ok && idTokenRaw != "" {
			idTokenClaims, err = parseIDTokenClaims(idTokenRaw)
			if err != nil {
				log.Warn().Err(err).Msg("Failed to parse ID token claims")
				// Continue without claims validation
			}
		}

		// Validate claims if we have both config and claims
		if idTokenClaims != nil {
			var requiredClaims, deniedClaims map[string][]string
			if requiredClaimsJSON != nil {
				if err := json.Unmarshal(requiredClaimsJSON, &requiredClaims); err != nil {
					log.Warn().Err(err).Msg("Failed to unmarshal required_claims")
				}
			}
			if deniedClaimsJSON != nil {
				if err := json.Unmarshal(deniedClaimsJSON, &deniedClaims); err != nil {
					log.Warn().Err(err).Msg("Failed to unmarshal denied_claims")
				}
			}

			provider := &auth.OAuthProviderRBAC{
				Name:           providerName,
				RequiredClaims: requiredClaims,
				DeniedClaims:   deniedClaims,
			}

			if err := auth.ValidateOAuthClaims(provider, idTokenClaims); err != nil {
				log.Warn().
					Err(err).
					Str("provider", providerName).
					Interface("claims", idTokenClaims).
					Msg("App OAuth access denied due to claims validation")
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
					"error": err.Error(),
				})
			}
		}
	}

	// Create or link user
	user, isNewUser, err := h.createOrLinkOAuthUser(ctx, providerName, providerUserID, email, userInfo, token)
	if err != nil {
		log.Error().Err(err).Str("provider", providerName).Str("email", email).Msg("Failed to create/link OAuth user")
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to create user account",
		})
	}

	// Generate JWT tokens with metadata
	accessToken, refreshToken, _, err := h.jwtManager.GenerateTokenPair(user.ID, user.Email, user.Role, user.UserMetadata, user.AppMetadata)
	if err != nil {
		log.Error().Err(err).Str("user_id", user.ID).Msg("Failed to generate JWT tokens")
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to generate authentication token",
		})
	}

	log.Info().
		Str("provider", providerName).
		Str("user_id", user.ID).
		Str("email", email).
		Bool("is_new_user", isNewUser).
		Msg("OAuth authentication successful")

	return c.JSON(fiber.Map{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"user":          user,
		"is_new_user":   isNewUser,
	})
}

// ListEnabledProviders lists all enabled OAuth providers for app login
// GET /api/v1/auth/oauth/providers
func (h *OAuthHandler) ListEnabledProviders(c *fiber.Ctx) error {
	ctx := c.Context()

	// SECURITY: Only list providers that allow app login
	query := `
		SELECT provider_name, display_name, redirect_url
		FROM dashboard.oauth_providers
		WHERE enabled = TRUE AND allow_app_login = TRUE
		ORDER BY display_name
	`

	rows, err := h.db.Query(ctx, query)
	if err != nil {
		log.Error().Err(err).Msg("Failed to list enabled OAuth providers")
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to retrieve OAuth providers",
		})
	}
	defer rows.Close()

	providers := []fiber.Map{}
	for rows.Next() {
		var providerName, displayName, redirectURL string
		if err := rows.Scan(&providerName, &displayName, &redirectURL); err != nil {
			log.Error().Err(err).Msg("Failed to scan OAuth provider")
			continue
		}

		providers = append(providers, fiber.Map{
			"provider":      providerName,
			"display_name":  displayName,
			"authorize_url": fmt.Sprintf("%s/api/v1/auth/oauth/%s/authorize", h.baseURL, providerName),
		})
	}

	return c.JSON(fiber.Map{
		"providers": providers,
	})
}

// Helper functions

// getProviderConfig retrieves OAuth configuration from database
func (h *OAuthHandler) getProviderConfig(ctx context.Context, providerName string) (*oauth2.Config, error) {
	// SECURITY: Only allow providers that enable app login
	query := `
		SELECT client_id, client_secret, redirect_url, scopes,
		       authorization_url, token_url, is_custom, allow_app_login,
		       COALESCE(is_encrypted, false) AS is_encrypted
		FROM dashboard.oauth_providers
		WHERE provider_name = $1 AND enabled = TRUE
	`

	var clientID, clientSecret, redirectURL string
	var scopes []string
	var authURL, tokenURL *string
	var isCustom bool
	var allowAppLogin bool
	var isEncrypted bool

	err := h.db.QueryRow(ctx, query, providerName).Scan(
		&clientID, &clientSecret, &redirectURL, &scopes,
		&authURL, &tokenURL, &isCustom, &allowAppLogin, &isEncrypted,
	)

	if err == sql.ErrNoRows {
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
		decryptedSecret, decErr := crypto.Decrypt(clientSecret, h.encryptionKey)
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
	query := "SELECT user_info_url FROM dashboard.oauth_providers WHERE provider_name = $1"
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

// createOrLinkOAuthUser creates a new user or links OAuth to existing user
func (h *OAuthHandler) createOrLinkOAuthUser(
	ctx context.Context,
	providerName string,
	providerUserID string,
	email string,
	userInfo map[string]interface{},
	token *oauth2.Token,
) (*auth.User, bool, error) {
	tx, err := h.db.Begin(ctx)
	if err != nil {
		return nil, false, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	// Check if OAuth link already exists
	var userID uuid.UUID
	query := "SELECT user_id FROM auth.oauth_links WHERE provider = $1 AND provider_user_id = $2"
	err = tx.QueryRow(ctx, query, providerName, providerUserID).Scan(&userID)

	var user *auth.User
	isNewUser := false

	// pgx returns error for no rows, not sql.ErrNoRows
	if err != nil && err.Error() == "no rows in result set" || err == sql.ErrNoRows {
		// Check if user exists with this email
		var existingUserID uuid.UUID
		query = "SELECT id FROM auth.users WHERE email = $1"
		err = tx.QueryRow(ctx, query, email).Scan(&existingUserID)

		if err != nil && (err.Error() == "no rows in result set" || err == sql.ErrNoRows) {
			// Create new user
			userID = uuid.New()
			query = `
				INSERT INTO auth.users (id, email, email_verified, role, user_metadata)
				VALUES ($1, $2, TRUE, 'authenticated', $3)
			`
			_, err = tx.Exec(ctx, query, userID, email, userInfo)
			if err != nil {
				return nil, false, fmt.Errorf("failed to create user: %w", err)
			}
			isNewUser = true
		} else if err != nil {
			return nil, false, fmt.Errorf("failed to check existing user: %w", err)
		} else {
			// Link to existing user
			userID = existingUserID
		}

		// Create OAuth link
		query = `
			INSERT INTO auth.oauth_links (user_id, provider, provider_user_id, email, metadata)
			VALUES ($1, $2, $3, $4, $5)
		`
		_, err = tx.Exec(ctx, query, userID, providerName, providerUserID, email, userInfo)
		if err != nil {
			return nil, false, fmt.Errorf("failed to create OAuth link: %w", err)
		}
	} else if err != nil {
		return nil, false, fmt.Errorf("failed to check OAuth link: %w", err)
	}

	// SECURITY: Encrypt OAuth tokens before storing (if encryption key is configured)
	accessTokenToStore := token.AccessToken
	refreshTokenToStore := token.RefreshToken
	// Extract ID token for OIDC logout support
	var idTokenToStore string
	if idTokenRaw, ok := token.Extra("id_token").(string); ok {
		idTokenToStore = idTokenRaw
	}

	if h.encryptionKey != "" {
		var encErr error
		accessTokenToStore, encErr = crypto.EncryptIfNotEmpty(token.AccessToken, h.encryptionKey)
		if encErr != nil {
			return nil, false, fmt.Errorf("failed to encrypt access token: %w", encErr)
		}
		refreshTokenToStore, encErr = crypto.EncryptIfNotEmpty(token.RefreshToken, h.encryptionKey)
		if encErr != nil {
			return nil, false, fmt.Errorf("failed to encrypt refresh token: %w", encErr)
		}
		idTokenToStore, encErr = crypto.EncryptIfNotEmpty(idTokenToStore, h.encryptionKey)
		if encErr != nil {
			return nil, false, fmt.Errorf("failed to encrypt id token: %w", encErr)
		}
	}

	// Store OAuth token (including id_token for OIDC logout)
	query = `
		INSERT INTO auth.oauth_tokens (user_id, provider, access_token, refresh_token, id_token, token_expiry)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (user_id, provider)
		DO UPDATE SET
			access_token = EXCLUDED.access_token,
			refresh_token = EXCLUDED.refresh_token,
			id_token = EXCLUDED.id_token,
			token_expiry = EXCLUDED.token_expiry,
			updated_at = CURRENT_TIMESTAMP
	`
	_, err = tx.Exec(ctx, query, userID, providerName, accessTokenToStore, refreshTokenToStore, idTokenToStore, token.Expiry)
	if err != nil {
		return nil, false, fmt.Errorf("failed to store OAuth token: %w", err)
	}

	// Fetch user details
	query = `
		SELECT id, email, email_verified, role, created_at, updated_at
		FROM auth.users
		WHERE id = $1
	`
	user = &auth.User{}
	err = tx.QueryRow(ctx, query, userID).Scan(
		&user.ID, &user.Email, &user.EmailVerified, &user.Role,
		&user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, false, fmt.Errorf("failed to fetch user: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return nil, false, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return user, isNewUser, nil
}

// Logout initiates OAuth Single Logout
// POST /api/v1/auth/oauth/:provider/logout
func (h *OAuthHandler) Logout(c *fiber.Ctx) error {
	ctx := c.Context()
	providerName := c.Params("provider")

	// Get user ID from JWT
	userID := c.Locals("user_id")
	if userID == nil || userID.(string) == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authentication required",
		})
	}
	userIDStr := userID.(string)

	// Parse optional redirect URL from request body
	var reqBody struct {
		RedirectURL string `json:"redirect_url"`
	}
	_ = c.BodyParser(&reqBody)

	// Get provider configuration to check for SLO endpoints
	var revocationEndpoint, endSessionEndpoint, clientID, clientSecret *string
	var isEncrypted bool
	err := h.db.QueryRow(ctx, `
		SELECT client_id, client_secret, revocation_endpoint, end_session_endpoint,
		       COALESCE(is_encrypted, false) AS is_encrypted
		FROM dashboard.oauth_providers
		WHERE provider_name = $1 AND enabled = TRUE
	`, providerName).Scan(&clientID, &clientSecret, &revocationEndpoint, &endSessionEndpoint, &isEncrypted)

	if err != nil {
		log.Error().Err(err).Str("provider", providerName).Msg("Failed to get OAuth provider for logout")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("OAuth provider '%s' not found or disabled", providerName),
		})
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
		if isEncrypted && h.encryptionKey != "" {
			decrypted, err := crypto.Decrypt(*clientSecret, h.encryptionKey)
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
		if h.encryptionKey != "" && accessToken != "" {
			decrypted, err := crypto.Decrypt(accessToken, h.encryptionKey)
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
					if h.encryptionKey != "" {
						decrypted, err := crypto.Decrypt(idToken, h.encryptionKey)
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
func (h *OAuthHandler) LogoutCallback(c *fiber.Ctx) error {
	ctx := c.Context()
	providerName := c.Params("provider")
	state := c.Query("state")

	if state == "" {
		log.Warn().Str("provider", providerName).Msg("OAuth logout callback missing state parameter")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Missing state parameter",
		})
	}

	// Validate and consume the state
	logoutState, err := h.logoutService.ValidateLogoutState(ctx, state)
	if err != nil {
		log.Warn().Err(err).Str("provider", providerName).Str("state", state).Msg("Invalid or expired logout state")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid or expired logout state",
		})
	}

	log.Info().
		Str("provider", providerName).
		Str("user_id", logoutState.UserID).
		Msg("OAuth logout callback successful")

	// Redirect to the post-logout redirect URI if specified
	if logoutState.PostLogoutRedirectURI != "" && logoutState.PostLogoutRedirectURI != c.OriginalURL() {
		return c.Redirect(logoutState.PostLogoutRedirectURI)
	}

	return c.JSON(fiber.Map{
		"message":  "Logout successful",
		"provider": providerName,
	})
}

// GetAndValidateState validates and consumes a state token, returning its metadata
// Returns the state metadata and true if valid, nil and false if not found or expired
// This is used by the dashboard OAuth callback to validate states created by the app OAuth authorize endpoint
func (h *OAuthHandler) GetAndValidateState(state string) (*auth.StateMetadata, bool) {
	return h.stateStore.GetAndValidate(state)
}
