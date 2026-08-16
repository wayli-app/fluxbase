package api

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog/log"
	"golang.org/x/oauth2"

	"github.com/nimbleflux/fluxbase/internal/auth"
	"github.com/nimbleflux/fluxbase/internal/config"
	"github.com/nimbleflux/fluxbase/internal/database"
	"github.com/nimbleflux/fluxbase/internal/middleware"
)

// OAuthHandler handles OAuth authentication flow
type OAuthHandler struct {
	db              *database.Connection
	authSvc         *auth.Service
	jwtManager      *auth.JWTManager
	stateStore      *auth.StateStore
	logoutService   *auth.OAuthLogoutService
	baseURL         string
	encryptionKey   []byte                       // SECURITY: Used for AES-256-GCM encryption of OAuth tokens at rest
	configProviders []config.OAuthProviderConfig // OAuth providers from config file
	stopCleanup     chan struct{}                // Signal to stop cleanup goroutines
	stopped         int32                        // Atomic flag to prevent double-close (0=running, 1=stopped)
}

// NewOAuthHandler creates a new OAuth handler
func NewOAuthHandler(db *database.Connection, authSvc *auth.Service, jwtManager *auth.JWTManager, baseURL string, encryptionKey []byte, configProviders []config.OAuthProviderConfig) *OAuthHandler {
	stateStore := auth.NewStateStore()

	// SECURITY: Validate encryption key for OAuth token storage
	// OAuth tokens (refresh tokens especially) are sensitive credentials that should be encrypted at rest
	if len(encryptionKey) == 0 {
		log.Error().Msg("SECURITY WARNING: FLUXBASE_ENCRYPTION_KEY not set - OAuth tokens will be stored UNENCRYPTED in database. " +
			"Set a 32-byte encryption key in production to protect OAuth refresh tokens.")
	} else if len(encryptionKey) != 32 {
		log.Error().
			Int("key_length", len(encryptionKey)).
			Int("required_length", 32).
			Msg("SECURITY WARNING: FLUXBASE_ENCRYPTION_KEY must be exactly 32 bytes for AES-256 - OAuth tokens will be stored UNENCRYPTED. " +
				"Fix the key length to enable encryption.")
		encryptionKey = nil // Clear invalid key
	}

	// Create logout service
	logoutService := auth.NewOAuthLogoutService(db, encryptionKey)

	// Create stop channel for cleanup goroutines
	stopCleanup := make(chan struct{})

	// Start cleanup goroutine for expired states
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := stateStore.Cleanup(context.Background()); err != nil {
					log.Warn().Err(err).Msg("Failed to cleanup expired OAuth states")
				}
			case <-stopCleanup:
				return
			}
		}
	}()

	// Start cleanup goroutine for expired logout states
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := logoutService.CleanupExpiredLogoutStates(context.Background()); err != nil {
					log.Warn().Err(err).Msg("Failed to cleanup expired OAuth logout states")
				}
			case <-stopCleanup:
				return
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
		stopCleanup:     stopCleanup,
	}
}

func (h *OAuthHandler) requireDB(c fiber.Ctx) error {
	if h.db == nil {
		return fiber.NewError(fiber.StatusServiceUnavailable, "Database not initialized")
	}
	return nil
}

func (h *OAuthHandler) requireAuthService(c fiber.Ctx) error {
	if h.authSvc == nil {
		return fiber.NewError(fiber.StatusServiceUnavailable, "Auth service not initialized")
	}
	return nil
}

func (h *OAuthHandler) requireLogoutService(c fiber.Ctx) error {
	if h.logoutService == nil {
		return fiber.NewError(fiber.StatusServiceUnavailable, "Logout service not initialized")
	}
	return nil
}

// Stop stops the cleanup goroutines
func (h *OAuthHandler) Stop() {
	// Check if already stopped (prevent double-close)
	if !atomic.CompareAndSwapInt32(&h.stopped, 0, 1) {
		return
	}
	if h.stopCleanup != nil {
		close(h.stopCleanup)
	}
}

// Authorize initiates the OAuth flow
// GET /api/v1/auth/oauth/:provider/authorize
func (h *OAuthHandler) Authorize(c fiber.Ctx) error {
	ctx := c.RequestCtx()
	providerName := c.Params("provider")

	// Get optional redirect_uri parameter for custom callback URL
	redirectURI := c.Query("redirect_uri")

	// Get tenant context for tenant-specific provider lookup
	tenantID := middleware.GetTenantIDFromContext(c)

	if err := h.requireDB(c); err != nil {
		return err
	}

	providerCfg, err := h.getProviderConfig(ctx, providerName, tenantID)
	if err != nil {
		log.Error().Err(err).Str("provider", providerName).Str("tenant_id", tenantID).Msg("Failed to get OAuth provider config")
		return SendBadRequest(c, fmt.Sprintf("OAuth provider '%s' not configured or disabled", providerName), "PROVIDER_NOT_CONFIGURED")
	}
	oauthConfig := providerCfg.Config

	// Override redirect URL if custom redirect_uri is provided
	if redirectURI != "" {
		// Build full URL if relative path is provided
		if redirectURI[0] == '/' {
			redirectURI = h.baseURL + redirectURI
		}
		// SECURITY: the override must match one of the configured redirect URLs
		if !matchRedirectURL(providerCfg.RedirectURLs, redirectURI, h.baseURL) {
			log.Warn().
				Str("provider", providerName).
				Str("redirect_uri", redirectURI).
				Msg("Rejected redirect_uri not present in provider's configured redirect URLs")
			return SendBadRequest(c, "Redirect URI is not in the provider's configured redirect URLs", "INVALID_REDIRECT_URI")
		}
		oauthConfig.RedirectURL = redirectURI
	}

	// Generate state for CSRF protection
	state, err := auth.GenerateState()
	if err != nil {
		log.Error().Err(err).Msg("Failed to generate OAuth state")
		return SendInternalError(c, "Failed to initiate OAuth flow")
	}

	// Store state with optional redirect URI for callback validation
	metadata := auth.StateMetadata{RedirectURI: redirectURI}
	if err := h.stateStore.Set(ctx, state, metadata); err != nil {
		log.Error().Err(err).Msg("Failed to store OAuth state")
		return SendInternalError(c, "Failed to initiate OAuth flow")
	}

	// Build auth URL options
	authURLOpts := []oauth2.AuthCodeOption{oauth2.AccessTypeOffline}

	// Add prompt=consent for Google to ensure refresh tokens on subsequent logins
	if strings.ToLower(providerName) == "google" {
		authURLOpts = append(authURLOpts, oauth2.SetAuthURLParam("prompt", "consent"))
	}

	// Generate authorization URL
	authURL := oauthConfig.AuthCodeURL(state, authURLOpts...)

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

// ListEnabledProviders lists all enabled OAuth providers for app login
// GET /api/v1/auth/oauth/providers
func (h *OAuthHandler) ListEnabledProviders(c fiber.Ctx) error {
	ctx := c.RequestCtx()

	if err := h.requireDB(c); err != nil {
		return err
	}

	tenantID := middleware.GetTenantIDFromContext(c)
	query := `
		SELECT provider_name, display_name, redirect_url
		FROM platform.oauth_providers
		WHERE enabled = TRUE AND allow_app_login = TRUE
		  AND (tenant_id = $1::uuid OR tenant_id IS NULL)
		ORDER BY display_name
	`

	var tenantUUID interface{}
	if tenantID != "" {
		tenantUUID = tenantID
	}
	rows, err := h.db.Query(ctx, query, tenantUUID)
	if err != nil {
		log.Error().Err(err).Msg("Failed to list enabled OAuth providers")
		return SendInternalError(c, "Failed to retrieve OAuth providers")
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

// GetAndValidateState validates and consumes a state token, returning its metadata
// Returns the state metadata and true if valid, nil and false if not found or expired
// This is used by the dashboard OAuth callback to validate states created by the app OAuth authorize endpoint
func (h *OAuthHandler) GetAndValidateState(state string) (*auth.StateMetadata, bool) {
	return h.stateStore.GetAndValidate(context.Background(), state)
}
