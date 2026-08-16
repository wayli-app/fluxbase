package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

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

// Callback handles the OAuth callback
// GET /api/v1/auth/oauth/:provider/callback
func (h *OAuthHandler) Callback(c fiber.Ctx) error {
	ctx := c.RequestCtx()
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

		return SendBadRequest(c, "OAuth authentication failed: "+errorDesc, "OAUTH_AUTH_FAILED")
	}

	// Validate state and retrieve metadata
	stateMetadata, valid := h.stateStore.GetAndValidate(ctx, state)
	if !valid {
		log.Warn().Str("provider", providerName).Str("state", state).Msg("Invalid OAuth state")
		return SendBadRequest(c, "Invalid OAuth state parameter", "INVALID_STATE")
	}

	if err := h.requireDB(c); err != nil {
		return err
	}

	tenantID := middleware.GetTenantIDFromContext(c)

	// Get OAuth provider configuration
	providerCfg, err := h.getProviderConfig(ctx, providerName, tenantID)
	if err != nil {
		log.Error().Err(err).Str("provider", providerName).Msg("Failed to get OAuth provider config")
		return SendBadRequest(c, "OAuth provider not configured", "PROVIDER_NOT_CONFIGURED")
	}
	oauthConfig := providerCfg.Config

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
		// SECURITY: the redirect URI must match one of the configured redirect URLs
		if !matchRedirectURL(providerCfg.RedirectURLs, finalRedirectURI, h.baseURL) {
			log.Warn().
				Str("provider", providerName).
				Str("redirect_uri", finalRedirectURI).
				Msg("Rejected redirect_uri not present in provider's configured redirect URLs")
			return SendBadRequest(c, "Redirect URI is not in the provider's configured redirect URLs", "INVALID_REDIRECT_URI")
		}
		oauthConfig.RedirectURL = finalRedirectURI
	}

	// Exchange code for token
	token, err := oauthConfig.Exchange(ctx, code)
	if err != nil {
		log.Error().Err(err).Str("provider", providerName).Msg("Failed to exchange OAuth code")
		return SendInternalError(c, "Failed to complete OAuth authentication")
	}

	// Get user info from OAuth provider
	userInfo, err := h.getUserInfo(ctx, providerName, oauthConfig, token)
	if err != nil {
		log.Error().Err(err).Str("provider", providerName).Msg("Failed to get user info from OAuth provider")
		return SendInternalError(c, "Failed to retrieve user information")
	}

	// Extract email and provider user ID
	email := h.extractEmail(providerName, userInfo)
	providerUserID := h.extractProviderUserID(providerName, userInfo)

	if email == "" || providerUserID == "" {
		log.Error().
			Str("provider", providerName).
			Interface("userInfo", userInfo).
			Msg("Missing required user information from OAuth provider")
		return SendInternalError(c, "OAuth provider did not return required user information")
	}

	// RBAC: Fetch provider RBAC config and validate claims if configured (OPTIONAL for app users)
	var requiredClaimsJSON, deniedClaimsJSON []byte
	err = h.db.QueryRow(ctx, `
		SELECT required_claims, denied_claims
		FROM platform.oauth_providers
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
				return SendForbidden(c, err.Error(), "OAUTH_ACCESS_DENIED")
			}
		}
	}

	// Create or link user
	user, isNewUser, err := h.createOrLinkOAuthUser(ctx, providerName, providerUserID, email, userInfo, token)
	if err != nil {
		log.Error().Err(err).Str("provider", providerName).Str("email", email).Msg("Failed to create/link OAuth user")
		return SendInternalError(c, "Failed to create user account")
	}

	tokenResp, err := h.authSvc.GenerateTokensForUser(ctx, user.ID)
	if err != nil {
		log.Error().Err(err).Str("user_id", user.ID).Msg("Failed to generate tokens and create session")
		return SendInternalError(c, "Failed to generate authentication token")
	}

	log.Info().
		Str("provider", providerName).
		Str("user_id", user.ID).
		Str("email", email).
		Bool("is_new_user", isNewUser).
		Msg("OAuth authentication successful")

	return c.JSON(fiber.Map{
		"access_token":  tokenResp.AccessToken,
		"refresh_token": tokenResp.RefreshToken,
		"expires_in":    tokenResp.ExpiresIn,
		"user":          user,
		"is_new_user":   isNewUser,
	})
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
	var user *auth.User
	var isNewUser bool

	err := database.WrapWithServiceRole(ctx, h.db, func(tx pgx.Tx) error {
		// Check if OAuth link already exists
		var userID uuid.UUID
		query := "SELECT user_id FROM auth.oauth_links WHERE provider = $1 AND provider_user_id = $2"
		err := tx.QueryRow(ctx, query, providerName, providerUserID).Scan(&userID)

		// pgx returns error for no rows, not sql.ErrNoRows
		if err != nil && err.Error() == "no rows in result set" || errors.Is(err, sql.ErrNoRows) {
			// Check if user exists with this email
			var existingUserID uuid.UUID
			query = "SELECT id FROM auth.users WHERE email = $1"
			err = tx.QueryRow(ctx, query, email).Scan(&existingUserID)

			if err != nil && (err.Error() == "no rows in result set" || errors.Is(err, sql.ErrNoRows)) {
				// Create new user
				userID = uuid.New()
				query = `
					INSERT INTO auth.users (id, email, email_verified, role, user_metadata)
					VALUES ($1, $2, TRUE, 'authenticated', $3)
				`
				_, err = tx.Exec(ctx, query, userID, email, userInfo)
				if err != nil {
					return fmt.Errorf("failed to create user: %w", err)
				}
				isNewUser = true
			} else {
				switch {
				case err != nil:
					return fmt.Errorf("failed to check existing user: %w", err)
				default:
					// Link to existing user
					userID = existingUserID
				}
			}

			// Create OAuth link
			query = `
				INSERT INTO auth.oauth_links (user_id, provider, provider_user_id, email, metadata)
				VALUES ($1, $2, $3, $4, $5)
			`
			_, err = tx.Exec(ctx, query, userID, providerName, providerUserID, email, userInfo)
			if err != nil {
				return fmt.Errorf("failed to create OAuth link: %w", err)
			}
		} else if err != nil {
			return fmt.Errorf("failed to check OAuth link: %w", err)
		}

		// SECURITY: Encrypt OAuth tokens before storing (if encryption key is configured)
		accessTokenToStore := token.AccessToken
		refreshTokenToStore := token.RefreshToken
		// Extract ID token for OIDC logout support
		var idTokenToStore string
		if idTokenRaw, ok := token.Extra("id_token").(string); ok {
			idTokenToStore = idTokenRaw
		}

		if len(h.encryptionKey) > 0 {
			var encErr error
			accessTokenToStore, encErr = crypto.EncryptIfNotEmptyWithBytesKey(token.AccessToken, h.encryptionKey)
			if encErr != nil {
				return fmt.Errorf("failed to encrypt access token: %w", encErr)
			}
			refreshTokenToStore, encErr = crypto.EncryptIfNotEmptyWithBytesKey(token.RefreshToken, h.encryptionKey)
			if encErr != nil {
				return fmt.Errorf("failed to encrypt refresh token: %w", encErr)
			}
			idTokenToStore, encErr = crypto.EncryptIfNotEmptyWithBytesKey(idTokenToStore, h.encryptionKey)
			if encErr != nil {
				return fmt.Errorf("failed to encrypt id token: %w", encErr)
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
			return fmt.Errorf("failed to store OAuth token: %w", err)
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
			return fmt.Errorf("failed to fetch user: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, false, err
	}

	return user, isNewUser, nil
}
