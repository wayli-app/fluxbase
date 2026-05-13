package api

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/auth"
	"github.com/nimbleflux/fluxbase/internal/config"
	"github.com/nimbleflux/fluxbase/internal/database"
	"github.com/nimbleflux/fluxbase/internal/middleware"
)

type MCPOAuthHandler struct {
	db          *database.Connection
	config      *config.MCPConfig
	authService *auth.Service
	baseURL     string
	publicURL   string
}

func NewMCPOAuthHandler(db *database.Connection, cfg *config.MCPConfig, authService *auth.Service, baseURL, publicURL string) *MCPOAuthHandler {
	return &MCPOAuthHandler{
		db:          db,
		config:      cfg,
		authService: authService,
		baseURL:     baseURL,
		publicURL:   publicURL,
	}
}

func (h *MCPOAuthHandler) HandleAuthorizationServerMetadata(c fiber.Ctx) error {
	if !h.config.OAuth.Enabled {
		return SendNotFound(c, "OAuth is not enabled for MCP")
	}

	issuer := h.getIssuer()

	metadata := fiber.Map{
		"issuer":                                issuer,
		"authorization_endpoint":                issuer + h.config.BasePath + "/oauth/authorize",
		"token_endpoint":                        issuer + h.config.BasePath + "/oauth/token",
		"revocation_endpoint":                   issuer + h.config.BasePath + "/oauth/revoke",
		"response_types_supported":              []string{"code"},
		"response_modes_supported":              []string{"query"},
		"grant_types_supported":                 []string{"authorization_code", "refresh_token"},
		"token_endpoint_auth_methods_supported": []string{"none"},
		"code_challenge_methods_supported":      []string{"S256"},
		"scopes_supported": []string{
			"tables:read", "tables:write",
			"functions:execute", "rpc:execute",
			"storage:read", "storage:write",
			"execute:jobs", "read:vectors",
			"read:schema", "admin:ddl",
		},
	}

	if h.config.OAuth.DCREnabled {
		metadata["registration_endpoint"] = issuer + h.config.BasePath + "/oauth/register"
	}

	return c.JSON(metadata)
}

func (h *MCPOAuthHandler) HandleProtectedResourceMetadata(c fiber.Ctx) error {
	if !h.config.OAuth.Enabled {
		return SendNotFound(c, "OAuth is not enabled for MCP")
	}

	issuer := h.getIssuer()

	return c.JSON(fiber.Map{
		"resource":                 issuer + h.config.BasePath,
		"authorization_servers":    []string{issuer},
		"scopes_supported":         []string{"tables:read", "tables:write", "read:schema"},
		"bearer_methods_supported": []string{"header"},
	})
}

func (h *MCPOAuthHandler) HandleClientRegistration(c fiber.Ctx) error {
	if !h.config.OAuth.Enabled || !h.config.OAuth.DCREnabled {
		return SendNotFound(c, "Dynamic Client Registration is not enabled")
	}

	var req struct {
		ClientName   string   `json:"client_name"`
		RedirectURIs []string `json:"redirect_uris"`
		Scopes       string   `json:"scope"`
	}

	if err := ParseBody(c, &req); err != nil {
		return err
	}

	if req.ClientName == "" {
		req.ClientName = "MCP Client"
	}

	if len(req.RedirectURIs) == 0 {
		return SendErrorWithCode(c, fiber.StatusBadRequest, "At least one redirect_uri is required", "invalid_redirect_uri")
	}

	for _, uri := range req.RedirectURIs {
		if !h.isRedirectURIAllowed(uri) {
			return SendErrorWithCode(c, fiber.StatusBadRequest, fmt.Sprintf("Redirect URI not allowed: %s", uri), "invalid_redirect_uri")
		}
	}

	scopes := []string{"tables:read", "read:schema"}
	if req.Scopes != "" {
		scopes = strings.Split(req.Scopes, " ")
	}

	clientID, err := generateSecureToken("mcp_", 32)
	if err != nil {
		log.Error().Err(err).Msg("Failed to generate client ID")
		return SendInternalError(c, "Failed to generate client credentials")
	}

	tenantCtx := middleware.CtxWithTenant(c)
	tenantID := database.TenantFromContext(tenantCtx)
	err = database.WrapWithServiceRoleAndTenant(tenantCtx, h.db, tenantID, func(tx pgx.Tx) error {
		_, err := tx.Exec(tenantCtx, `
			INSERT INTO auth.mcp_oauth_clients (client_id, client_name, client_type, redirect_uris, scopes, metadata)
			VALUES ($1, $2, 'public', $3, $4, $5)
		`, clientID, req.ClientName, req.RedirectURIs, scopes, map[string]any{
			"user_agent":    c.Get("User-Agent"),
			"registered_at": time.Now().UTC(),
		})
		return err
	})
	if err != nil {
		log.Error().Err(err).Msg("Failed to register OAuth client")
		return SendInternalError(c, "Failed to register client")
	}

	log.Info().
		Str("client_id", clientID).
		Str("client_name", req.ClientName).
		Strs("redirect_uris", req.RedirectURIs).
		Msg("MCP OAuth client registered")

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"client_id":                  clientID,
		"client_name":                req.ClientName,
		"redirect_uris":              req.RedirectURIs,
		"scope":                      strings.Join(scopes, " "),
		"token_endpoint_auth_method": "none",
		"grant_types":                []string{"authorization_code", "refresh_token"},
		"response_types":             []string{"code"},
	})
}

func (h *MCPOAuthHandler) HandleAuthorize(c fiber.Ctx) error {
	if !h.config.OAuth.Enabled {
		return SendNotFound(c, "OAuth is not enabled")
	}

	clientID := c.Query("client_id")
	redirectURI := c.Query("redirect_uri")
	responseType := c.Query("response_type")
	scope := c.Query("scope")
	state := c.Query("state")
	codeChallenge := c.Query("code_challenge")
	codeChallengeMethod := c.Query("code_challenge_method")

	if clientID == "" {
		return SendErrorWithCode(c, fiber.StatusBadRequest, "client_id is required", "invalid_request")
	}

	if responseType != "code" {
		return SendErrorWithCode(c, fiber.StatusBadRequest, "Only response_type=code is supported", "unsupported_response_type")
	}

	if codeChallenge == "" {
		return SendErrorWithCode(c, fiber.StatusBadRequest, "code_challenge is required (PKCE)", "invalid_request")
	}

	if codeChallengeMethod != "S256" && codeChallengeMethod != "" {
		return SendErrorWithCode(c, fiber.StatusBadRequest, "code_challenge_method must be S256", "invalid_request")
	}
	if codeChallengeMethod == "" {
		codeChallengeMethod = "S256"
	}

	var client struct {
		ClientID     string   `db:"client_id"`
		ClientName   string   `db:"client_name"`
		RedirectURIs []string `db:"redirect_uris"`
		Scopes       []string `db:"scopes"`
		IsActive     bool     `db:"is_active"`
	}

	err := database.WrapWithServiceRole(c.RequestCtx(), h.db, func(tx pgx.Tx) error {
		return tx.QueryRow(c.RequestCtx(), `
			SELECT client_id, client_name, redirect_uris, scopes, is_active
			FROM auth.mcp_oauth_clients
			WHERE client_id = $1
		`, clientID).Scan(&client.ClientID, &client.ClientName, &client.RedirectURIs, &client.Scopes, &client.IsActive)
	})
	if err != nil {
		return SendErrorWithCode(c, fiber.StatusBadRequest, "Client not found", "invalid_client")
	}

	if !client.IsActive {
		return SendErrorWithCode(c, fiber.StatusBadRequest, "Client is inactive", "invalid_client")
	}

	if redirectURI == "" && len(client.RedirectURIs) == 1 {
		redirectURI = client.RedirectURIs[0]
	}

	if !h.isURIInList(redirectURI, client.RedirectURIs) {
		return SendErrorWithCode(c, fiber.StatusBadRequest, "Redirect URI not registered for this client", "invalid_redirect_uri")
	}

	requestedScopes := client.Scopes
	if scope != "" {
		requestedScopes = strings.Split(scope, " ")
		for _, s := range requestedScopes {
			if !h.contains(client.Scopes, s) {
				return SendErrorWithCode(c, fiber.StatusBadRequest, fmt.Sprintf("Scope '%s' not allowed for this client", s), "invalid_scope")
			}
		}
	}

	userID := h.extractUserFromRequest(c)

	if userID == nil {
		authURL := h.getIssuer() + h.config.BasePath + "/oauth/authorize?" + string(c.Request().URI().QueryString())
		loginURL := h.getIssuer() + "/admin/login?return_to=" + url.QueryEscape(authURL)
		return c.Redirect().Status(fiber.StatusFound).To(loginURL)
	}

	code, err := generateSecureToken("", 32)
	if err != nil {
		log.Error().Err(err).Msg("Failed to generate authorization code")
		return h.redirectWithError(c, redirectURI, "server_error", "Failed to generate authorization code", state)
	}

	tenantCtx := middleware.CtxWithTenant(c)
	tenantID := database.TenantFromContext(tenantCtx)
	err = database.WrapWithServiceRoleAndTenant(tenantCtx, h.db, tenantID, func(tx pgx.Tx) error {
		_, err := tx.Exec(tenantCtx, `
			INSERT INTO auth.mcp_oauth_codes (code, client_id, user_id, redirect_uri, scopes, code_challenge, code_challenge_method, state)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`, code, clientID, userID, redirectURI, requestedScopes, codeChallenge, codeChallengeMethod, state)
		return err
	})
	if err != nil {
		log.Error().Err(err).Msg("Failed to store authorization code")
		return h.redirectWithError(c, redirectURI, "server_error", "Failed to process authorization", state)
	}

	log.Debug().
		Str("client_id", clientID).
		Str("code", code[:8]+"...").
		Msg("MCP OAuth authorization code issued")

	redirectURL, _ := url.Parse(redirectURI)
	q := redirectURL.Query()
	q.Set("code", code)
	if state != "" {
		q.Set("state", state)
	}
	redirectURL.RawQuery = q.Encode()

	return c.Redirect().Status(fiber.StatusFound).To(redirectURL.String())
}

func (h *MCPOAuthHandler) HandleAuthorizeConsent(c fiber.Ctx) error {
	return h.HandleAuthorize(c)
}

func (h *MCPOAuthHandler) HandleToken(c fiber.Ctx) error {
	if !h.config.OAuth.Enabled {
		return SendNotFound(c, "OAuth is not enabled")
	}

	grantType := c.FormValue("grant_type")

	switch grantType {
	case "authorization_code":
		return h.handleAuthorizationCodeGrant(c)
	case "refresh_token":
		return h.handleRefreshTokenGrant(c)
	default:
		return SendErrorWithCode(c, fiber.StatusBadRequest, "Only authorization_code and refresh_token grants are supported", "unsupported_grant_type")
	}
}

func (h *MCPOAuthHandler) handleAuthorizationCodeGrant(c fiber.Ctx) error {
	code := c.FormValue("code")
	clientID := c.FormValue("client_id")
	redirectURI := c.FormValue("redirect_uri")
	codeVerifier := c.FormValue("code_verifier")

	if code == "" || clientID == "" || codeVerifier == "" {
		return SendErrorWithCode(c, fiber.StatusBadRequest, "code, client_id, and code_verifier are required", "invalid_request")
	}

	var authCode struct {
		ClientID            string
		UserID              *string
		RedirectURI         string
		Scopes              []string
		CodeChallenge       string
		CodeChallengeMethod string
		ExpiresAt           time.Time
	}

	err := database.WrapWithServiceRole(c.RequestCtx(), h.db, func(tx pgx.Tx) error {
		return tx.QueryRow(c.RequestCtx(), `
			SELECT client_id, user_id, redirect_uri, scopes, code_challenge, code_challenge_method, expires_at
			FROM auth.mcp_oauth_codes
			WHERE code = $1
		`, code).Scan(
			&authCode.ClientID, &authCode.UserID, &authCode.RedirectURI,
			&authCode.Scopes, &authCode.CodeChallenge, &authCode.CodeChallengeMethod, &authCode.ExpiresAt,
		)
	})
	if err != nil {
		return SendErrorWithCode(c, fiber.StatusBadRequest, "Invalid authorization code", "invalid_grant")
	}

	_ = database.WrapWithServiceRole(c.RequestCtx(), h.db, func(tx pgx.Tx) error {
		_, err := tx.Exec(c.RequestCtx(), `DELETE FROM auth.mcp_oauth_codes WHERE code = $1`, code)
		return err
	})

	if time.Now().After(authCode.ExpiresAt) {
		return SendErrorWithCode(c, fiber.StatusBadRequest, "Authorization code has expired", "invalid_grant")
	}

	if authCode.ClientID != clientID {
		return SendErrorWithCode(c, fiber.StatusBadRequest, "Client ID mismatch", "invalid_grant")
	}

	if redirectURI != "" && authCode.RedirectURI != redirectURI {
		return SendErrorWithCode(c, fiber.StatusBadRequest, "Redirect URI mismatch", "invalid_grant")
	}

	if !h.verifyPKCE(codeVerifier, authCode.CodeChallenge, authCode.CodeChallengeMethod) {
		return SendErrorWithCode(c, fiber.StatusBadRequest, "Invalid code_verifier", "invalid_grant")
	}

	accessToken, err := generateSecureToken("mcp_at_", 32)
	if err != nil {
		return SendInternalError(c, "Failed to generate access token")
	}

	refreshToken, err := generateSecureToken("mcp_rt_", 32)
	if err != nil {
		return SendInternalError(c, "Failed to generate refresh token")
	}

	accessTokenExpiry := time.Now().Add(h.config.OAuth.TokenExpiry)
	refreshTokenExpiry := time.Now().Add(h.config.OAuth.RefreshTokenExpiry)

	accessTokenHash := hashToken(accessToken)
	refreshTokenHash := hashToken(refreshToken)

	tenantCtx := middleware.CtxWithTenant(c)
	tenantID := database.TenantFromContext(tenantCtx)

	err = database.WrapWithServiceRoleAndTenant(tenantCtx, h.db, tenantID, func(tx pgx.Tx) error {
		_, err := tx.Exec(tenantCtx, `
			INSERT INTO auth.mcp_oauth_tokens (token_type, token_hash, client_id, user_id, scopes, expires_at)
			VALUES ('access', $1, $2, $3, $4, $5)
		`, accessTokenHash, clientID, authCode.UserID, authCode.Scopes, accessTokenExpiry)
		return err
	})
	if err != nil {
		log.Error().Err(err).Msg("Failed to store access token")
		return SendInternalError(c, "Failed to store access token")
	}

	err = database.WrapWithServiceRoleAndTenant(tenantCtx, h.db, tenantID, func(tx pgx.Tx) error {
		_, err := tx.Exec(tenantCtx, `
			INSERT INTO auth.mcp_oauth_tokens (token_type, token_hash, client_id, user_id, scopes, expires_at)
			VALUES ('refresh', $1, $2, $3, $4, $5)
		`, refreshTokenHash, clientID, authCode.UserID, authCode.Scopes, refreshTokenExpiry)
		return err
	})
	if err != nil {
		log.Error().Err(err).Msg("Failed to store refresh token")
		return SendInternalError(c, "Failed to store refresh token")
	}

	log.Info().
		Str("client_id", clientID).
		Strs("scopes", authCode.Scopes).
		Msg("MCP OAuth tokens issued")

	return c.JSON(fiber.Map{
		"access_token":  accessToken,
		"token_type":    "Bearer",
		"expires_in":    int(h.config.OAuth.TokenExpiry.Seconds()),
		"refresh_token": refreshToken,
		"scope":         strings.Join(authCode.Scopes, " "),
	})
}

func (h *MCPOAuthHandler) handleRefreshTokenGrant(c fiber.Ctx) error {
	refreshToken := c.FormValue("refresh_token")
	clientID := c.FormValue("client_id")

	if refreshToken == "" || clientID == "" {
		return SendErrorWithCode(c, fiber.StatusBadRequest, "refresh_token and client_id are required", "invalid_request")
	}

	refreshTokenHash := hashToken(refreshToken)

	var token struct {
		ID       string
		ClientID string
		UserID   *string
		Scopes   []string
	}

	err := database.WrapWithServiceRole(c.RequestCtx(), h.db, func(tx pgx.Tx) error {
		return tx.QueryRow(c.RequestCtx(), `
			SELECT id, client_id, user_id, scopes
			FROM auth.mcp_oauth_tokens
			WHERE token_hash = $1 AND token_type = 'refresh' AND NOT is_revoked AND expires_at > NOW()
		`, refreshTokenHash).Scan(&token.ID, &token.ClientID, &token.UserID, &token.Scopes)
	})
	if err != nil {
		return SendErrorWithCode(c, fiber.StatusBadRequest, "Invalid refresh token", "invalid_grant")
	}

	if token.ClientID != clientID {
		return SendErrorWithCode(c, fiber.StatusBadRequest, "Client ID mismatch", "invalid_grant")
	}

	_ = database.WrapWithServiceRole(c.RequestCtx(), h.db, func(tx pgx.Tx) error {
		_, err := tx.Exec(c.RequestCtx(), `
			UPDATE auth.mcp_oauth_tokens
			SET is_revoked = true, revoked_at = NOW(), revoked_reason = 'rotated'
			WHERE id = $1
		`, token.ID)
		return err
	})

	newAccessToken, _ := generateSecureToken("mcp_at_", 32)
	newRefreshToken, _ := generateSecureToken("mcp_rt_", 32)

	accessTokenExpiry := time.Now().Add(h.config.OAuth.TokenExpiry)
	refreshTokenExpiry := time.Now().Add(h.config.OAuth.RefreshTokenExpiry)

	accessTokenHash := hashToken(newAccessToken)
	newRefreshTokenHash := hashToken(newRefreshToken)

	tenantCtx := middleware.CtxWithTenant(c)
	tenantID := database.TenantFromContext(tenantCtx)

	_ = database.WrapWithServiceRoleAndTenant(tenantCtx, h.db, tenantID, func(tx pgx.Tx) error {
		_, err := tx.Exec(tenantCtx, `
			INSERT INTO auth.mcp_oauth_tokens (token_type, token_hash, client_id, user_id, scopes, expires_at)
			VALUES ('access', $1, $2, $3, $4, $5)
		`, accessTokenHash, clientID, token.UserID, token.Scopes, accessTokenExpiry)
		return err
	})

	_ = database.WrapWithServiceRoleAndTenant(tenantCtx, h.db, tenantID, func(tx pgx.Tx) error {
		_, err := tx.Exec(tenantCtx, `
			INSERT INTO auth.mcp_oauth_tokens (token_type, token_hash, client_id, user_id, scopes, expires_at)
			VALUES ('refresh', $1, $2, $3, $4, $5)
		`, newRefreshTokenHash, clientID, token.UserID, token.Scopes, refreshTokenExpiry)
		return err
	})

	return c.JSON(fiber.Map{
		"access_token":  newAccessToken,
		"token_type":    "Bearer",
		"expires_in":    int(h.config.OAuth.TokenExpiry.Seconds()),
		"refresh_token": newRefreshToken,
		"scope":         strings.Join(token.Scopes, " "),
	})
}

func (h *MCPOAuthHandler) HandleRevoke(c fiber.Ctx) error {
	if !h.config.OAuth.Enabled {
		return SendNotFound(c, "OAuth is not enabled")
	}

	token := c.FormValue("token")
	if token == "" {
		return SendErrorWithCode(c, fiber.StatusBadRequest, "token is required", "invalid_request")
	}

	tokenHash := hashToken(token)

	var rowsAffected int64
	err := database.WrapWithServiceRole(c.RequestCtx(), h.db, func(tx pgx.Tx) error {
		tag, err := tx.Exec(c.RequestCtx(), `
			UPDATE auth.mcp_oauth_tokens
			SET is_revoked = true, revoked_at = NOW(), revoked_reason = 'user_revoked'
			WHERE token_hash = $1
		`, tokenHash)
		if err != nil {
			return err
		}
		rowsAffected = tag.RowsAffected()
		return nil
	})
	if err != nil {
		log.Error().Err(err).Msg("Failed to revoke token")
	}

	if rowsAffected > 0 {
		log.Debug().Msg("MCP OAuth token revoked")
	}

	return c.SendStatus(fiber.StatusOK)
}

func (h *MCPOAuthHandler) ValidateAccessToken(c fiber.Ctx, token string) (clientID string, userID *string, scopes []string, err error) {
	tokenHash := hashToken(token)

	err = database.WrapWithServiceRole(c.RequestCtx(), h.db, func(tx pgx.Tx) error {
		return tx.QueryRow(c.RequestCtx(), `
			SELECT client_id, user_id, scopes
			FROM auth.mcp_oauth_tokens
			WHERE token_hash = $1 AND token_type = 'access' AND NOT is_revoked AND expires_at > NOW()
		`, tokenHash).Scan(&clientID, &userID, &scopes)
	})

	return
}

func (h *MCPOAuthHandler) getIssuer() string {
	if h.publicURL != "" {
		return h.publicURL
	}
	return h.baseURL
}

func (h *MCPOAuthHandler) isRedirectURIAllowed(uri string) bool {
	allowedPatterns := h.config.OAuth.AllowedRedirectURIs
	if len(allowedPatterns) == 0 {
		allowedPatterns = config.DefaultMCPOAuthRedirectURIs()
	}

	for _, pattern := range allowedPatterns {
		if h.matchURIPattern(uri, pattern) {
			return true
		}
	}
	return false
}

func (h *MCPOAuthHandler) matchURIPattern(uri, pattern string) bool {
	if uri == pattern {
		return true
	}

	if strings.HasSuffix(pattern, "://") {
		return strings.HasPrefix(uri, pattern)
	}

	if strings.Contains(pattern, ":*") {
		prefix := strings.Split(pattern, ":*")[0]
		if strings.HasPrefix(uri, prefix+":") {
			parsedURI, err := url.Parse(uri)
			if err != nil {
				return false
			}
			parsedPattern, err := url.Parse(strings.Replace(pattern, ":*", ":1234", 1))
			if err != nil {
				return false
			}
			return parsedURI.Hostname() == parsedPattern.Hostname()
		}
	}

	if strings.Contains(pattern, "*") {
		parts := strings.Split(pattern, "*")
		if len(parts) == 2 {
			return strings.HasPrefix(uri, parts[0]) && strings.HasSuffix(uri, parts[1])
		}
	}

	return false
}

func (h *MCPOAuthHandler) isURIInList(uri string, list []string) bool {
	for _, u := range list {
		if u == uri || h.matchURIPattern(uri, u) {
			return true
		}
	}
	return false
}

func (h *MCPOAuthHandler) contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func (h *MCPOAuthHandler) extractUserFromRequest(c fiber.Ctx) *string {
	authHeader := c.Get("Authorization")
	if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
		token := strings.TrimPrefix(authHeader, "Bearer ")

		if !strings.HasPrefix(token, "mcp_at_") && h.authService != nil {
			claims, err := h.authService.JWTManager().ValidateToken(token)
			if err == nil {
				isRevoked, err := h.authService.TokenBlacklistService().IsTokenRevoked(c.RequestCtx(), claims.ID, "", time.Time{})
				if err == nil && !isRevoked {
					return &claims.UserID
				}
			}
		}
	}

	accessToken := c.Cookies(AccessTokenCookieName)
	if accessToken != "" && h.authService != nil {
		claims, err := h.authService.JWTManager().ValidateToken(accessToken)
		if err == nil {
			isRevoked, err := h.authService.TokenBlacklistService().IsTokenRevoked(c.RequestCtx(), claims.ID, "", time.Time{})
			if err == nil && !isRevoked {
				return &claims.UserID
			}
		}
	}

	adminToken := c.Cookies("fluxbase_admin_token")
	if adminToken != "" && h.authService != nil {
		token := adminToken
		if len(token) >= 2 && token[0] == '"' && token[len(token)-1] == '"' {
			token = token[1 : len(token)-1]
		}
		claims, err := h.authService.JWTManager().ValidateToken(token)
		if err == nil {
			isRevoked, err := h.authService.TokenBlacklistService().IsTokenRevoked(c.RequestCtx(), claims.ID, "", time.Time{})
			if err == nil && !isRevoked {
				return &claims.UserID
			}
		}
	}

	return nil
}

func (h *MCPOAuthHandler) verifyPKCE(verifier, challenge, method string) bool {
	if method != "S256" {
		return false
	}

	hash := sha256.Sum256([]byte(verifier))
	computed := base64.RawURLEncoding.EncodeToString(hash[:])

	return computed == challenge
}

func (h *MCPOAuthHandler) redirectWithError(c fiber.Ctx, redirectURI, errorCode, errorDesc, state string) error {
	u, _ := url.Parse(redirectURI)
	q := u.Query()
	q.Set("error", errorCode)
	q.Set("error_description", errorDesc)
	if state != "" {
		q.Set("state", state)
	}
	u.RawQuery = q.Encode()
	return c.Redirect().Status(fiber.StatusFound).To(u.String())
}

func generateSecureToken(prefix string, length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return prefix + hex.EncodeToString(bytes), nil
}

func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

func matchRedirectURI(pattern, uri string) bool {
	if uri == pattern {
		return true
	}

	if strings.HasSuffix(pattern, "://") {
		return strings.HasPrefix(uri, pattern)
	}

	if strings.Contains(pattern, ":*") {
		prefix := strings.Split(pattern, ":*")[0]
		if strings.HasPrefix(uri, prefix+":") {
			parsedURI, err := url.Parse(uri)
			if err != nil {
				return false
			}
			parsedPattern, err := url.Parse(strings.Replace(pattern, ":*", ":1234", 1))
			if err != nil {
				return false
			}
			return parsedURI.Hostname() == parsedPattern.Hostname()
		}
	}

	if strings.Contains(pattern, "*") {
		parts := strings.Split(pattern, "*")
		if len(parts) == 2 {
			return strings.HasPrefix(uri, parts[0]) && strings.HasSuffix(uri, parts[1])
		}
	}

	return false
}

func verifyPKCE(verifier, challenge, method string) bool {
	if method != "S256" {
		return false
	}

	hash := sha256.Sum256([]byte(verifier))
	computed := base64.RawURLEncoding.EncodeToString(hash[:])

	return computed == challenge
}

func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return ""
	}
	for i := range bytes {
		bytes[i] = charset[bytes[i]%byte(len(charset))]
	}
	return string(bytes)
}

func nullIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// fiber:context-methods migrated
