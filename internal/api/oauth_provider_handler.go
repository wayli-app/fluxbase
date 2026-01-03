package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/fluxbase-eu/fluxbase/internal/auth"
	"github.com/fluxbase-eu/fluxbase/internal/config"
	"github.com/fluxbase-eu/fluxbase/internal/crypto"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

// OAuthProviderHandler handles OAuth provider configuration management
type OAuthProviderHandler struct {
	db              *pgxpool.Pool
	settingsCache   *auth.SettingsCache
	encryptionKey   string
	configProviders []config.OAuthProviderConfig
	baseURL         string
}

// NewOAuthProviderHandler creates a new OAuth provider handler
func NewOAuthProviderHandler(db *pgxpool.Pool, settingsCache *auth.SettingsCache, encryptionKey, baseURL string, configProviders []config.OAuthProviderConfig) *OAuthProviderHandler {
	return &OAuthProviderHandler{
		db:              db,
		settingsCache:   settingsCache,
		encryptionKey:   encryptionKey,
		configProviders: configProviders,
		baseURL:         baseURL,
	}
}

// EncryptExistingSecrets encrypts any plaintext client secrets in the database.
// This should be called on startup to migrate existing secrets to encrypted format.
func (h *OAuthProviderHandler) EncryptExistingSecrets(ctx context.Context) error {
	// Find all providers with unencrypted secrets
	query := `
		SELECT id, provider_name, client_secret
		FROM dashboard.oauth_providers
		WHERE (is_encrypted IS NULL OR is_encrypted = false)
		  AND client_secret IS NOT NULL
		  AND client_secret != ''
	`

	rows, err := h.db.Query(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to query unencrypted providers: %w", err)
	}
	defer rows.Close()

	var toEncrypt []struct {
		ID           uuid.UUID
		ProviderName string
		ClientSecret string
	}

	for rows.Next() {
		var provider struct {
			ID           uuid.UUID
			ProviderName string
			ClientSecret string
		}
		if err := rows.Scan(&provider.ID, &provider.ProviderName, &provider.ClientSecret); err != nil {
			log.Error().Err(err).Msg("Failed to scan OAuth provider for encryption")
			continue
		}
		toEncrypt = append(toEncrypt, provider)
	}

	if len(toEncrypt) == 0 {
		return nil
	}

	log.Info().Int("count", len(toEncrypt)).Msg("Encrypting existing OAuth provider secrets")

	// Encrypt each secret
	for _, provider := range toEncrypt {
		encryptedSecret, encErr := crypto.Encrypt(provider.ClientSecret, h.encryptionKey)
		if encErr != nil {
			log.Error().Err(encErr).Str("provider", provider.ProviderName).Msg("Failed to encrypt client secret")
			continue
		}

		_, updateErr := h.db.Exec(ctx, `
			UPDATE dashboard.oauth_providers
			SET client_secret = $1, is_encrypted = true
			WHERE id = $2
		`, encryptedSecret, provider.ID)

		if updateErr != nil {
			log.Error().Err(updateErr).Str("provider", provider.ProviderName).Msg("Failed to update encrypted client secret")
			continue
		}

		log.Info().Str("provider", provider.ProviderName).Msg("Encrypted OAuth provider client secret")
	}

	return nil
}

// OAuthProvider represents an OAuth provider configuration
type OAuthProvider struct {
	ID                  uuid.UUID           `json:"id"`
	ProviderName        string              `json:"provider_name"`
	DisplayName         string              `json:"display_name"`
	Enabled             bool                `json:"enabled"`
	ClientID            string              `json:"client_id"`
	ClientSecret        string              `json:"client_secret,omitempty"` // Omitted in GET responses
	HasSecret           bool                `json:"has_secret"`              // Indicates if a client secret is set
	RedirectURL         string              `json:"redirect_url"`
	Scopes              []string            `json:"scopes"`
	IsCustom            bool                `json:"is_custom"`
	AuthorizationURL    *string             `json:"authorization_url,omitempty"`
	TokenURL            *string             `json:"token_url,omitempty"`
	UserInfoURL         *string             `json:"user_info_url,omitempty"`
	RevocationEndpoint  *string             `json:"revocation_endpoint,omitempty"`  // OAuth 2.0 Token Revocation (RFC 7009)
	EndSessionEndpoint  *string             `json:"end_session_endpoint,omitempty"` // OIDC RP-Initiated Logout
	AllowDashboardLogin bool                `json:"allow_dashboard_login"`
	AllowAppLogin       bool                `json:"allow_app_login"`
	RequiredClaims      map[string][]string `json:"required_claims,omitempty"`
	DeniedClaims        map[string][]string `json:"denied_claims,omitempty"`
	Source              string              `json:"source,omitempty"` // "database" or "config"
	CreatedAt           time.Time           `json:"created_at"`
	UpdatedAt           time.Time           `json:"updated_at"`
}

// CreateOAuthProviderRequest represents a request to create an OAuth provider
type CreateOAuthProviderRequest struct {
	ProviderName        string              `json:"provider_name"`
	DisplayName         string              `json:"display_name"`
	Enabled             bool                `json:"enabled"`
	ClientID            string              `json:"client_id"`
	ClientSecret        string              `json:"client_secret"`
	RedirectURL         string              `json:"redirect_url"`
	Scopes              []string            `json:"scopes"`
	IsCustom            bool                `json:"is_custom"`
	AuthorizationURL    *string             `json:"authorization_url,omitempty"`
	TokenURL            *string             `json:"token_url,omitempty"`
	UserInfoURL         *string             `json:"user_info_url,omitempty"`
	RevocationEndpoint  *string             `json:"revocation_endpoint,omitempty"`  // OAuth 2.0 Token Revocation (RFC 7009)
	EndSessionEndpoint  *string             `json:"end_session_endpoint,omitempty"` // OIDC RP-Initiated Logout
	AllowDashboardLogin *bool               `json:"allow_dashboard_login,omitempty"`
	AllowAppLogin       *bool               `json:"allow_app_login,omitempty"`
	RequiredClaims      map[string][]string `json:"required_claims,omitempty"`
	DeniedClaims        map[string][]string `json:"denied_claims,omitempty"`
}

// UpdateOAuthProviderRequest represents a request to update an OAuth provider
type UpdateOAuthProviderRequest struct {
	DisplayName         *string             `json:"display_name,omitempty"`
	Enabled             *bool               `json:"enabled,omitempty"`
	ClientID            *string             `json:"client_id,omitempty"`
	ClientSecret        *string             `json:"client_secret,omitempty"`
	RedirectURL         *string             `json:"redirect_url,omitempty"`
	Scopes              []string            `json:"scopes,omitempty"`
	AuthorizationURL    *string             `json:"authorization_url,omitempty"`
	TokenURL            *string             `json:"token_url,omitempty"`
	UserInfoURL         *string             `json:"user_info_url,omitempty"`
	RevocationEndpoint  *string             `json:"revocation_endpoint,omitempty"`  // OAuth 2.0 Token Revocation (RFC 7009)
	EndSessionEndpoint  *string             `json:"end_session_endpoint,omitempty"` // OIDC RP-Initiated Logout
	AllowDashboardLogin *bool               `json:"allow_dashboard_login,omitempty"`
	AllowAppLogin       *bool               `json:"allow_app_login,omitempty"`
	RequiredClaims      map[string][]string `json:"required_claims,omitempty"`
	DeniedClaims        map[string][]string `json:"denied_claims,omitempty"`
}

// Auth settings types
type AuthSettings struct {
	EnableSignup                  bool                       `json:"enable_signup"`
	RequireEmailVerification      bool                       `json:"require_email_verification"`
	EnableMagicLink               bool                       `json:"enable_magic_link"`
	PasswordMinLength             int                        `json:"password_min_length"`
	PasswordRequireUppercase      bool                       `json:"password_require_uppercase"`
	PasswordRequireLowercase      bool                       `json:"password_require_lowercase"`
	PasswordRequireNumber         bool                       `json:"password_require_number"`
	PasswordRequireSpecial        bool                       `json:"password_require_special"`
	SessionTimeoutMinutes         int                        `json:"session_timeout_minutes"`
	MaxSessionsPerUser            int                        `json:"max_sessions_per_user"`
	DisableDashboardPasswordLogin bool                       `json:"disable_dashboard_password_login"`
	Overrides                     map[string]SettingOverride `json:"_overrides,omitempty"`
}

// SettingOverride contains override information for a specific setting
type SettingOverride struct {
	IsOverridden bool   `json:"is_overridden"`
	EnvVar       string `json:"env_var"`
}

var providerNamePattern = regexp.MustCompile(`^[a-z][a-z0-9_]{1,49}$`)

// ListOAuthProviders lists all OAuth providers
func (h *OAuthProviderHandler) ListOAuthProviders(c *fiber.Ctx) error {
	ctx := c.Context()

	query := `
		SELECT id, provider_name, display_name, enabled, client_id, redirect_url, scopes,
		       is_custom, authorization_url, token_url, user_info_url,
		       revocation_endpoint, end_session_endpoint,
		       COALESCE(allow_dashboard_login, false), COALESCE(allow_app_login, true),
		       required_claims, denied_claims,
		       created_at, updated_at,
		       (client_secret IS NOT NULL AND client_secret != '') AS has_secret
		FROM dashboard.oauth_providers
		ORDER BY display_name
	`

	rows, err := h.db.Query(ctx, query)
	if err != nil {
		log.Error().Err(err).Msg("Failed to list OAuth providers")
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to retrieve OAuth providers",
		})
	}
	defer rows.Close()

	providers := []OAuthProvider{}
	for rows.Next() {
		var p OAuthProvider
		var requiredClaimsJSON, deniedClaimsJSON []byte
		err := rows.Scan(
			&p.ID, &p.ProviderName, &p.DisplayName, &p.Enabled, &p.ClientID,
			&p.RedirectURL, &p.Scopes, &p.IsCustom, &p.AuthorizationURL,
			&p.TokenURL, &p.UserInfoURL, &p.RevocationEndpoint, &p.EndSessionEndpoint,
			&p.AllowDashboardLogin, &p.AllowAppLogin,
			&requiredClaimsJSON, &deniedClaimsJSON,
			&p.CreatedAt, &p.UpdatedAt, &p.HasSecret,
		)
		if err != nil {
			log.Error().Err(err).Msg("Failed to scan OAuth provider")
			continue
		}
		// Unmarshal RBAC fields
		if requiredClaimsJSON != nil {
			_ = json.Unmarshal(requiredClaimsJSON, &p.RequiredClaims)
		}
		if deniedClaimsJSON != nil {
			_ = json.Unmarshal(deniedClaimsJSON, &p.DeniedClaims)
		}
		// Don't return client_secret in list
		p.ClientSecret = ""
		p.Source = "database"
		providers = append(providers, p)
	}

	// Track which providers already exist in database (by name)
	dbProviderNames := make(map[string]bool)
	for _, p := range providers {
		dbProviderNames[p.ProviderName] = true
	}

	// Add config-based providers that aren't already in the database
	for _, cp := range h.configProviders {
		if dbProviderNames[cp.Name] {
			// Skip - already exists in database
			continue
		}

		// Determine display name
		displayName := cp.DisplayName
		if displayName == "" {
			// Capitalize first letter
			if len(cp.Name) > 0 {
				displayName = strings.ToUpper(cp.Name[:1]) + cp.Name[1:]
			} else {
				displayName = cp.Name
			}
		}

		// Build redirect URL
		redirectURL := fmt.Sprintf("%s/api/v1/auth/oauth/%s/callback", h.baseURL, cp.Name)

		configProvider := OAuthProvider{
			ID:                  uuid.MustParse("00000000-0000-0000-0000-000000000000"), // Placeholder ID for config providers
			ProviderName:        cp.Name,
			DisplayName:         displayName,
			Enabled:             cp.Enabled,
			ClientID:            cp.ClientID,
			HasSecret:           cp.ClientSecret != "",
			RedirectURL:         redirectURL,
			Scopes:              cp.Scopes,
			IsCustom:            cp.IssuerURL != "",
			AllowDashboardLogin: cp.AllowDashboardLogin,
			AllowAppLogin:       cp.AllowAppLogin,
			RequiredClaims:      cp.RequiredClaims,
			DeniedClaims:        cp.DeniedClaims,
			Source:              "config",
			CreatedAt:           time.Now(),
			UpdatedAt:           time.Now(),
		}
		providers = append(providers, configProvider)
	}

	return c.JSON(providers)
}

// GetOAuthProvider gets a single OAuth provider by ID
func (h *OAuthProviderHandler) GetOAuthProvider(c *fiber.Ctx) error {
	ctx := c.Context()
	id := c.Params("id")

	providerID, err := uuid.Parse(id)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid provider ID",
		})
	}

	query := `
		SELECT id, provider_name, display_name, enabled, client_id, redirect_url, scopes,
		       is_custom, authorization_url, token_url, user_info_url,
		       revocation_endpoint, end_session_endpoint,
		       COALESCE(allow_dashboard_login, false), COALESCE(allow_app_login, true),
		       required_claims, denied_claims,
		       created_at, updated_at,
		       (client_secret IS NOT NULL AND client_secret != '') AS has_secret
		FROM dashboard.oauth_providers
		WHERE id = $1
	`

	var p OAuthProvider
	var requiredClaimsJSON, deniedClaimsJSON []byte
	err = h.db.QueryRow(ctx, query, providerID).Scan(
		&p.ID, &p.ProviderName, &p.DisplayName, &p.Enabled, &p.ClientID,
		&p.RedirectURL, &p.Scopes, &p.IsCustom, &p.AuthorizationURL,
		&p.TokenURL, &p.UserInfoURL, &p.RevocationEndpoint, &p.EndSessionEndpoint,
		&p.AllowDashboardLogin, &p.AllowAppLogin,
		&requiredClaimsJSON, &deniedClaimsJSON,
		&p.CreatedAt, &p.UpdatedAt, &p.HasSecret,
	)

	if err == sql.ErrNoRows {
		return c.Status(404).JSON(fiber.Map{
			"error": "OAuth provider not found",
		})
	}
	if err != nil {
		log.Error().Err(err).Str("id", id).Msg("Failed to get OAuth provider")
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to retrieve OAuth provider",
		})
	}

	// Unmarshal RBAC fields
	if requiredClaimsJSON != nil {
		_ = json.Unmarshal(requiredClaimsJSON, &p.RequiredClaims)
	}
	if deniedClaimsJSON != nil {
		_ = json.Unmarshal(deniedClaimsJSON, &p.DeniedClaims)
	}

	// Don't return client_secret
	p.ClientSecret = ""
	return c.JSON(p)
}

// CreateOAuthProvider creates a new OAuth provider
func (h *OAuthProviderHandler) CreateOAuthProvider(c *fiber.Ctx) error {
	ctx := c.Context()
	var req CreateOAuthProviderRequest

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate provider name
	if !providerNamePattern.MatchString(req.ProviderName) {
		return c.Status(400).JSON(fiber.Map{
			"error": "Provider name must start with a letter and contain only lowercase letters, numbers, and underscores (2-50 chars)",
		})
	}

	// Validate required fields
	if req.DisplayName == "" || req.ClientID == "" || req.ClientSecret == "" || req.RedirectURL == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "Missing required fields: display_name, client_id, client_secret, redirect_url",
		})
	}

	// For custom providers, require custom URLs
	if req.IsCustom {
		if req.AuthorizationURL == nil || req.TokenURL == nil || req.UserInfoURL == nil {
			return c.Status(400).JSON(fiber.Map{
				"error": "Custom providers require authorization_url, token_url, and user_info_url",
			})
		}
	}

	// Get user ID from context (set by auth middleware)
	userID := getUserIDFromContext(c)

	// Set defaults for new fields
	allowDashboardLogin := false
	if req.AllowDashboardLogin != nil {
		allowDashboardLogin = *req.AllowDashboardLogin
	}
	allowAppLogin := true
	if req.AllowAppLogin != nil {
		allowAppLogin = *req.AllowAppLogin
	}

	// Marshal RBAC fields to JSON
	var requiredClaimsJSON, deniedClaimsJSON []byte
	if len(req.RequiredClaims) > 0 {
		requiredClaimsJSON, _ = json.Marshal(req.RequiredClaims)
	}
	if len(req.DeniedClaims) > 0 {
		deniedClaimsJSON, _ = json.Marshal(req.DeniedClaims)
	}

	// Encrypt client secret before storing
	encryptedSecret, err := crypto.Encrypt(req.ClientSecret, h.encryptionKey)
	if err != nil {
		log.Error().Err(err).Str("provider", req.ProviderName).Msg("Failed to encrypt client secret")
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to encrypt client secret",
		})
	}

	query := `
		INSERT INTO dashboard.oauth_providers (
			provider_name, display_name, enabled, client_id, client_secret,
			redirect_url, scopes, is_custom, authorization_url, token_url,
			user_info_url, revocation_endpoint, end_session_endpoint,
			allow_dashboard_login, allow_app_login, required_claims, denied_claims,
			created_by, updated_by, is_encrypted
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $18, true)
		RETURNING id, created_at, updated_at
	`

	var id uuid.UUID
	var createdAt, updatedAt time.Time
	err = h.db.QueryRow(
		ctx, query,
		req.ProviderName, req.DisplayName, req.Enabled, req.ClientID, encryptedSecret,
		req.RedirectURL, req.Scopes, req.IsCustom, req.AuthorizationURL, req.TokenURL,
		req.UserInfoURL, req.RevocationEndpoint, req.EndSessionEndpoint,
		allowDashboardLogin, allowAppLogin, requiredClaimsJSON, deniedClaimsJSON, userID,
	).Scan(&id, &createdAt, &updatedAt)

	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			return c.Status(409).JSON(fiber.Map{
				"error": fmt.Sprintf("OAuth provider '%s' already exists", req.ProviderName),
			})
		}
		log.Error().Err(err).Str("provider", req.ProviderName).Msg("Failed to create OAuth provider")
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to create OAuth provider",
		})
	}

	log.Info().Str("id", id.String()).Str("provider", req.ProviderName).Msg("OAuth provider created")

	return c.Status(201).JSON(fiber.Map{
		"success":    true,
		"id":         id,
		"provider":   req.ProviderName,
		"message":    fmt.Sprintf("OAuth provider '%s' created successfully", req.DisplayName),
		"created_at": createdAt.Format(time.RFC3339),
		"updated_at": updatedAt.Format(time.RFC3339),
	})
}

// UpdateOAuthProvider updates an existing OAuth provider
func (h *OAuthProviderHandler) UpdateOAuthProvider(c *fiber.Ctx) error {
	ctx := c.Context()
	id := c.Params("id")

	providerID, err := uuid.Parse(id)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid provider ID",
		})
	}

	var req UpdateOAuthProviderRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Build dynamic update query
	updates := []string{}
	args := []interface{}{providerID}
	argPos := 2

	if req.DisplayName != nil {
		updates = append(updates, fmt.Sprintf("display_name = $%d", argPos))
		args = append(args, *req.DisplayName)
		argPos++
	}
	if req.Enabled != nil {
		updates = append(updates, fmt.Sprintf("enabled = $%d", argPos))
		args = append(args, *req.Enabled)
		argPos++
	}
	if req.ClientID != nil {
		updates = append(updates, fmt.Sprintf("client_id = $%d", argPos))
		args = append(args, *req.ClientID)
		argPos++
	}
	if req.ClientSecret != nil && *req.ClientSecret != "" {
		// Encrypt client secret before storing
		encryptedSecret, encErr := crypto.Encrypt(*req.ClientSecret, h.encryptionKey)
		if encErr != nil {
			log.Error().Err(encErr).Msg("Failed to encrypt client secret")
			return c.Status(500).JSON(fiber.Map{
				"error": "Failed to encrypt client secret",
			})
		}
		updates = append(updates, fmt.Sprintf("client_secret = $%d", argPos))
		args = append(args, encryptedSecret)
		argPos++
		updates = append(updates, fmt.Sprintf("is_encrypted = $%d", argPos))
		args = append(args, true)
		argPos++
	}
	if req.RedirectURL != nil {
		updates = append(updates, fmt.Sprintf("redirect_url = $%d", argPos))
		args = append(args, *req.RedirectURL)
		argPos++
	}
	if req.Scopes != nil {
		updates = append(updates, fmt.Sprintf("scopes = $%d", argPos))
		args = append(args, req.Scopes)
		argPos++
	}
	if req.AuthorizationURL != nil {
		updates = append(updates, fmt.Sprintf("authorization_url = $%d", argPos))
		args = append(args, req.AuthorizationURL)
		argPos++
	}
	if req.TokenURL != nil {
		updates = append(updates, fmt.Sprintf("token_url = $%d", argPos))
		args = append(args, req.TokenURL)
		argPos++
	}
	if req.UserInfoURL != nil {
		updates = append(updates, fmt.Sprintf("user_info_url = $%d", argPos))
		args = append(args, req.UserInfoURL)
		argPos++
	}
	if req.RevocationEndpoint != nil {
		updates = append(updates, fmt.Sprintf("revocation_endpoint = $%d", argPos))
		args = append(args, req.RevocationEndpoint)
		argPos++
	}
	if req.EndSessionEndpoint != nil {
		updates = append(updates, fmt.Sprintf("end_session_endpoint = $%d", argPos))
		args = append(args, req.EndSessionEndpoint)
		argPos++
	}
	if req.AllowDashboardLogin != nil {
		updates = append(updates, fmt.Sprintf("allow_dashboard_login = $%d", argPos))
		args = append(args, *req.AllowDashboardLogin)
		argPos++
	}
	if req.AllowAppLogin != nil {
		updates = append(updates, fmt.Sprintf("allow_app_login = $%d", argPos))
		args = append(args, *req.AllowAppLogin)
		argPos++
	}
	if req.RequiredClaims != nil {
		requiredClaimsJSON, _ := json.Marshal(req.RequiredClaims)
		updates = append(updates, fmt.Sprintf("required_claims = $%d", argPos))
		args = append(args, requiredClaimsJSON)
		argPos++
	}
	if req.DeniedClaims != nil {
		deniedClaimsJSON, _ := json.Marshal(req.DeniedClaims)
		updates = append(updates, fmt.Sprintf("denied_claims = $%d", argPos))
		args = append(args, deniedClaimsJSON)
		argPos++
	}

	if len(updates) == 0 {
		return c.Status(400).JSON(fiber.Map{
			"error": "No fields to update",
		})
	}

	// Add updated_by
	userID := getUserIDFromContext(c)
	updates = append(updates, fmt.Sprintf("updated_by = $%d", argPos))
	args = append(args, userID)

	query := fmt.Sprintf(
		"UPDATE dashboard.oauth_providers SET %s WHERE id = $1 RETURNING display_name",
		strings.Join(updates, ", "),
	)

	var displayName string
	err = h.db.QueryRow(ctx, query, args...).Scan(&displayName)

	if err == sql.ErrNoRows {
		return c.Status(404).JSON(fiber.Map{
			"error": "OAuth provider not found",
		})
	}
	if err != nil {
		log.Error().Err(err).Str("id", id).Msg("Failed to update OAuth provider")
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to update OAuth provider",
		})
	}

	log.Info().Str("id", id).Msg("OAuth provider updated")

	return c.JSON(fiber.Map{
		"success": true,
		"message": fmt.Sprintf("OAuth provider '%s' updated successfully", displayName),
	})
}

// DeleteOAuthProvider deletes an OAuth provider
func (h *OAuthProviderHandler) DeleteOAuthProvider(c *fiber.Ctx) error {
	ctx := c.Context()
	id := c.Params("id")

	providerID, err := uuid.Parse(id)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid provider ID",
		})
	}

	query := "DELETE FROM dashboard.oauth_providers WHERE id = $1 RETURNING display_name"

	var displayName string
	err = h.db.QueryRow(ctx, query, providerID).Scan(&displayName)

	if err == sql.ErrNoRows {
		return c.Status(404).JSON(fiber.Map{
			"error": "OAuth provider not found",
		})
	}
	if err != nil {
		log.Error().Err(err).Str("id", id).Msg("Failed to delete OAuth provider")
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to delete OAuth provider",
		})
	}

	log.Info().Str("id", id).Str("provider", displayName).Msg("OAuth provider deleted")

	return c.JSON(fiber.Map{
		"success": true,
		"message": fmt.Sprintf("OAuth provider '%s' deleted successfully", displayName),
	})
}

// GetAuthSettings retrieves authentication settings
func (h *OAuthProviderHandler) GetAuthSettings(c *fiber.Ctx) error {
	ctx := c.Context()

	query := "SELECT key, value FROM app.settings WHERE category = 'auth'"
	rows, err := h.db.Query(ctx, query)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get auth settings")
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to retrieve auth settings",
		})
	}
	defer rows.Close()

	settings := AuthSettings{
		PasswordMinLength:     8,
		SessionTimeoutMinutes: 60,
		MaxSessionsPerUser:    5,
		Overrides:             make(map[string]SettingOverride),
	}

	for rows.Next() {
		var key string
		var value interface{}
		if err := rows.Scan(&key, &value); err != nil {
			continue
		}

		// Parse JSONB values
		switch key {
		case "enable_signup":
			if v, ok := value.(bool); ok {
				settings.EnableSignup = v
			}
		case "require_email_verification":
			if v, ok := value.(bool); ok {
				settings.RequireEmailVerification = v
			}
		case "enable_magic_link":
			if v, ok := value.(bool); ok {
				settings.EnableMagicLink = v
			}
		case "password_min_length":
			if v, ok := value.(float64); ok {
				settings.PasswordMinLength = int(v)
			}
		case "password_require_uppercase":
			if v, ok := value.(bool); ok {
				settings.PasswordRequireUppercase = v
			}
		case "password_require_lowercase":
			if v, ok := value.(bool); ok {
				settings.PasswordRequireLowercase = v
			}
		case "password_require_number":
			if v, ok := value.(bool); ok {
				settings.PasswordRequireNumber = v
			}
		case "password_require_special":
			if v, ok := value.(bool); ok {
				settings.PasswordRequireSpecial = v
			}
		case "session_timeout_minutes":
			if v, ok := value.(float64); ok {
				settings.SessionTimeoutMinutes = int(v)
			}
		case "max_sessions_per_user":
			if v, ok := value.(float64); ok {
				settings.MaxSessionsPerUser = int(v)
			}
		case "disable_dashboard_password_login":
			if v, ok := value.(bool); ok {
				settings.DisableDashboardPasswordLogin = v
			}
		}
	}

	// Populate override information for settings that can be overridden
	if h.settingsCache != nil {
		settingsMap := map[string]string{
			"enable_signup":              "app.auth.enable_signup",
			"enable_magic_link":          "app.auth.enable_magic_link",
			"password_min_length":        "app.auth.password_min_length",
			"require_email_verification": "app.auth.require_email_verification",
		}

		for fieldName, settingKey := range settingsMap {
			if h.settingsCache.IsOverriddenByEnv(settingKey) {
				settings.Overrides[fieldName] = SettingOverride{
					IsOverridden: true,
					EnvVar:       h.settingsCache.GetEnvVarName(settingKey),
				}
			}
		}
	}

	return c.JSON(settings)
}

// UpdateAuthSettings updates authentication settings
func (h *OAuthProviderHandler) UpdateAuthSettings(c *fiber.Ctx) error {
	ctx := c.Context()
	var req AuthSettings

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Check for environment variable overrides before updating
	if h.settingsCache != nil {
		settingsKeyMap := map[string]string{
			"enable_signup":              "app.auth.enable_signup",
			"enable_magic_link":          "app.auth.enable_magic_link",
			"password_min_length":        "app.auth.password_min_length",
			"require_email_verification": "app.auth.require_email_verification",
		}

		for fieldName, settingKey := range settingsKeyMap {
			if h.settingsCache.IsOverriddenByEnv(settingKey) {
				return c.Status(409).JSON(fiber.Map{
					"error": fmt.Sprintf("Setting '%s' cannot be updated because it is overridden by an environment variable", fieldName),
					"code":  "ENV_OVERRIDE",
					"field": fieldName,
				})
			}
		}
	}

	// Upsert each setting (insert or update if exists)
	upsertQuery := `
		INSERT INTO app.settings (key, value, category, updated_at)
		VALUES ($1, $2, 'auth', NOW())
		ON CONFLICT (key) DO UPDATE
		SET value = EXCLUDED.value, updated_at = NOW()
	`

	// If trying to disable password login, verify SSO providers exist
	if req.DisableDashboardPasswordLogin {
		hasSSO, err := h.hasDashboardSSOProviders(ctx)
		if err != nil {
			log.Error().Err(err).Msg("Failed to check SSO providers")
			return c.Status(500).JSON(fiber.Map{
				"error": "Failed to verify SSO providers",
			})
		}
		if !hasSSO {
			return c.Status(400).JSON(fiber.Map{
				"error": "Cannot disable password login: No SSO providers are configured for dashboard login. Configure at least one OAuth or SAML provider with 'Allow dashboard login' enabled first.",
				"code":  "NO_SSO_PROVIDERS",
			})
		}
	}

	updates := map[string]interface{}{
		"enable_signup":                    req.EnableSignup,
		"require_email_verification":       req.RequireEmailVerification,
		"enable_magic_link":                req.EnableMagicLink,
		"password_min_length":              req.PasswordMinLength,
		"password_require_uppercase":       req.PasswordRequireUppercase,
		"password_require_lowercase":       req.PasswordRequireLowercase,
		"password_require_number":          req.PasswordRequireNumber,
		"password_require_special":         req.PasswordRequireSpecial,
		"session_timeout_minutes":          req.SessionTimeoutMinutes,
		"max_sessions_per_user":            req.MaxSessionsPerUser,
		"disable_dashboard_password_login": req.DisableDashboardPasswordLogin,
	}

	for key, value := range updates {
		_, err := h.db.Exec(ctx, upsertQuery, key, value)
		if err != nil {
			log.Error().Err(err).Str("setting", key).Msg("Failed to upsert auth setting")
			return c.Status(500).JSON(fiber.Map{
				"error": fmt.Sprintf("Failed to update setting: %s", key),
			})
		}
	}

	log.Info().Msg("Auth settings updated successfully")

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Authentication settings updated successfully",
	})
}

// hasDashboardSSOProviders checks if any SSO providers are configured for dashboard login
func (h *OAuthProviderHandler) hasDashboardSSOProviders(ctx context.Context) (bool, error) {
	// Check OAuth providers
	var oauthCount int
	err := h.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM dashboard.oauth_providers
		WHERE enabled = true AND allow_dashboard_login = true
	`).Scan(&oauthCount)
	if err != nil {
		return false, err
	}
	if oauthCount > 0 {
		return true, nil
	}

	// Check SAML providers
	var samlCount int
	err = h.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM auth.saml_providers
		WHERE enabled = true AND allow_dashboard_login = true
	`).Scan(&samlCount)
	if err != nil {
		return false, err
	}

	return samlCount > 0, nil
}

// Helper function to get user ID from context (set by auth middleware)
func getUserIDFromContext(c *fiber.Ctx) *uuid.UUID {
	// Try to get from dashboard auth (set by middleware)
	if userIDStr := c.Locals("user_id"); userIDStr != nil {
		if uid, err := uuid.Parse(userIDStr.(string)); err == nil {
			return &uid
		}
	}
	return nil
}
