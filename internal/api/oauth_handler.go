package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
	"github.com/wayli-app/fluxbase/internal/auth"
	"golang.org/x/oauth2"
)

// OAuthHandler handles OAuth authentication flow
type OAuthHandler struct {
	db         *pgxpool.Pool
	authSvc    *auth.Service
	jwtManager *auth.JWTManager
	stateStore *auth.StateStore
	baseURL    string
}

// NewOAuthHandler creates a new OAuth handler
func NewOAuthHandler(db *pgxpool.Pool, authSvc *auth.Service, jwtManager *auth.JWTManager, baseURL string) *OAuthHandler {
	stateStore := auth.NewStateStore()

	// Start cleanup goroutine for expired states
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			stateStore.Cleanup()
		}
	}()

	return &OAuthHandler{
		db:         db,
		authSvc:    authSvc,
		jwtManager: jwtManager,
		stateStore: stateStore,
		baseURL:    baseURL,
	}
}

// Authorize initiates the OAuth flow
// GET /api/v1/auth/oauth/:provider/authorize
func (h *OAuthHandler) Authorize(c *fiber.Ctx) error {
	ctx := c.Context()
	providerName := c.Params("provider")

	// Get OAuth provider configuration from database
	oauthConfig, err := h.getProviderConfig(ctx, providerName)
	if err != nil {
		log.Error().Err(err).Str("provider", providerName).Msg("Failed to get OAuth provider config")
		return c.Status(400).JSON(fiber.Map{
			"error": fmt.Sprintf("OAuth provider '%s' not configured or disabled", providerName),
		})
	}

	// Generate state for CSRF protection
	state, err := auth.GenerateState()
	if err != nil {
		log.Error().Err(err).Msg("Failed to generate OAuth state")
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to initiate OAuth flow",
		})
	}

	// Store state
	h.stateStore.Set(state)

	// Generate authorization URL
	authURL := oauthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline)

	log.Info().
		Str("provider", providerName).
		Str("state", state).
		Msg("OAuth authorization initiated")

	// Redirect to OAuth provider
	return c.Redirect(authURL)
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

	// Validate state
	if !h.stateStore.Validate(state) {
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

// ListEnabledProviders lists all enabled OAuth providers
// GET /api/v1/auth/oauth/providers
func (h *OAuthHandler) ListEnabledProviders(c *fiber.Ctx) error {
	ctx := c.Context()

	query := `
		SELECT provider_name, display_name, redirect_url
		FROM dashboard.oauth_providers
		WHERE enabled = TRUE
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
	query := `
		SELECT client_id, client_secret, redirect_url, scopes,
		       authorization_url, token_url, is_custom
		FROM dashboard.oauth_providers
		WHERE provider_name = $1 AND enabled = TRUE
	`

	var clientID, clientSecret, redirectURL string
	var scopes []string
	var authURL, tokenURL *string
	var isCustom bool

	err := h.db.QueryRow(ctx, query, providerName).Scan(
		&clientID, &clientSecret, &redirectURL, &scopes,
		&authURL, &tokenURL, &isCustom,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("OAuth provider '%s' not found or disabled", providerName)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query OAuth provider: %w", err)
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
	defer resp.Body.Close()

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
				INSERT INTO auth.users (id, email, email_verified, role, metadata)
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

	// Store OAuth token
	query = `
		INSERT INTO auth.oauth_tokens (user_id, provider, access_token, refresh_token, token_expiry)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id, provider)
		DO UPDATE SET
			access_token = EXCLUDED.access_token,
			refresh_token = EXCLUDED.refresh_token,
			token_expiry = EXCLUDED.token_expiry,
			updated_at = CURRENT_TIMESTAMP
	`
	_, err = tx.Exec(ctx, query, userID, providerName, token.AccessToken, token.RefreshToken, token.Expiry)
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
