package api

import (
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

// OAuthProviderHandler handles OAuth provider configuration management
type OAuthProviderHandler struct {
	db *pgxpool.Pool
}

// NewOAuthProviderHandler creates a new OAuth provider handler
func NewOAuthProviderHandler(db *pgxpool.Pool) *OAuthProviderHandler {
	return &OAuthProviderHandler{db: db}
}

// OAuthProvider represents an OAuth provider configuration
type OAuthProvider struct {
	ID               uuid.UUID `json:"id"`
	ProviderName     string    `json:"provider_name"`
	DisplayName      string    `json:"display_name"`
	Enabled          bool      `json:"enabled"`
	ClientID         string    `json:"client_id"`
	ClientSecret     string    `json:"client_secret,omitempty"` // Omitted in GET responses
	RedirectURL      string    `json:"redirect_url"`
	Scopes           []string  `json:"scopes"`
	IsCustom         bool      `json:"is_custom"`
	AuthorizationURL *string   `json:"authorization_url,omitempty"`
	TokenURL         *string   `json:"token_url,omitempty"`
	UserInfoURL      *string   `json:"user_info_url,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// CreateOAuthProviderRequest represents a request to create an OAuth provider
type CreateOAuthProviderRequest struct {
	ProviderName     string   `json:"provider_name"`
	DisplayName      string   `json:"display_name"`
	Enabled          bool     `json:"enabled"`
	ClientID         string   `json:"client_id"`
	ClientSecret     string   `json:"client_secret"`
	RedirectURL      string   `json:"redirect_url"`
	Scopes           []string `json:"scopes"`
	IsCustom         bool     `json:"is_custom"`
	AuthorizationURL *string  `json:"authorization_url,omitempty"`
	TokenURL         *string  `json:"token_url,omitempty"`
	UserInfoURL      *string  `json:"user_info_url,omitempty"`
}

// UpdateOAuthProviderRequest represents a request to update an OAuth provider
type UpdateOAuthProviderRequest struct {
	DisplayName      *string  `json:"display_name,omitempty"`
	Enabled          *bool    `json:"enabled,omitempty"`
	ClientID         *string  `json:"client_id,omitempty"`
	ClientSecret     *string  `json:"client_secret,omitempty"`
	RedirectURL      *string  `json:"redirect_url,omitempty"`
	Scopes           []string `json:"scopes,omitempty"`
	AuthorizationURL *string  `json:"authorization_url,omitempty"`
	TokenURL         *string  `json:"token_url,omitempty"`
	UserInfoURL      *string  `json:"user_info_url,omitempty"`
}

// Auth settings types
type AuthSettings struct {
	EnableSignup             bool `json:"enable_signup"`
	RequireEmailVerification bool `json:"require_email_verification"`
	EnableMagicLink          bool `json:"enable_magic_link"`
	PasswordMinLength        int  `json:"password_min_length"`
	PasswordRequireUppercase bool `json:"password_require_uppercase"`
	PasswordRequireLowercase bool `json:"password_require_lowercase"`
	PasswordRequireNumber    bool `json:"password_require_number"`
	PasswordRequireSpecial   bool `json:"password_require_special"`
	SessionTimeoutMinutes    int  `json:"session_timeout_minutes"`
	MaxSessionsPerUser       int  `json:"max_sessions_per_user"`
}

var providerNamePattern = regexp.MustCompile(`^[a-z][a-z0-9_]{1,49}$`)

// ListOAuthProviders lists all OAuth providers
func (h *OAuthProviderHandler) ListOAuthProviders(c *fiber.Ctx) error {
	ctx := c.Context()

	query := `
		SELECT id, provider_name, display_name, enabled, client_id, redirect_url, scopes,
		       is_custom, authorization_url, token_url, user_info_url, created_at, updated_at
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
		err := rows.Scan(
			&p.ID, &p.ProviderName, &p.DisplayName, &p.Enabled, &p.ClientID,
			&p.RedirectURL, &p.Scopes, &p.IsCustom, &p.AuthorizationURL,
			&p.TokenURL, &p.UserInfoURL, &p.CreatedAt, &p.UpdatedAt,
		)
		if err != nil {
			log.Error().Err(err).Msg("Failed to scan OAuth provider")
			continue
		}
		// Don't return client_secret in list
		p.ClientSecret = ""
		providers = append(providers, p)
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
		       is_custom, authorization_url, token_url, user_info_url, created_at, updated_at
		FROM dashboard.oauth_providers
		WHERE id = $1
	`

	var p OAuthProvider
	err = h.db.QueryRow(ctx, query, providerID).Scan(
		&p.ID, &p.ProviderName, &p.DisplayName, &p.Enabled, &p.ClientID,
		&p.RedirectURL, &p.Scopes, &p.IsCustom, &p.AuthorizationURL,
		&p.TokenURL, &p.UserInfoURL, &p.CreatedAt, &p.UpdatedAt,
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

	query := `
		INSERT INTO dashboard.oauth_providers (
			provider_name, display_name, enabled, client_id, client_secret,
			redirect_url, scopes, is_custom, authorization_url, token_url,
			user_info_url, created_by, updated_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $12)
		RETURNING id, created_at, updated_at
	`

	var id uuid.UUID
	var createdAt, updatedAt time.Time
	err := h.db.QueryRow(
		ctx, query,
		req.ProviderName, req.DisplayName, req.Enabled, req.ClientID, req.ClientSecret,
		req.RedirectURL, req.Scopes, req.IsCustom, req.AuthorizationURL, req.TokenURL,
		req.UserInfoURL, userID,
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
		updates = append(updates, fmt.Sprintf("client_secret = $%d", argPos))
		args = append(args, *req.ClientSecret)
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

	query := "SELECT setting_key, setting_value FROM dashboard.auth_settings"
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

	// Get user ID from context
	userID := getUserIDFromContext(c)

	// Update each setting
	updateQuery := `
		UPDATE dashboard.auth_settings
		SET setting_value = $1, updated_by = $2
		WHERE setting_key = $3
	`

	updates := map[string]interface{}{
		"enable_signup":              req.EnableSignup,
		"require_email_verification": req.RequireEmailVerification,
		"enable_magic_link":          req.EnableMagicLink,
		"password_min_length":        req.PasswordMinLength,
		"password_require_uppercase": req.PasswordRequireUppercase,
		"password_require_lowercase": req.PasswordRequireLowercase,
		"password_require_number":    req.PasswordRequireNumber,
		"password_require_special":   req.PasswordRequireSpecial,
		"session_timeout_minutes":    req.SessionTimeoutMinutes,
		"max_sessions_per_user":      req.MaxSessionsPerUser,
	}

	for key, value := range updates {
		_, err := h.db.Exec(ctx, updateQuery, value, userID, key)
		if err != nil {
			log.Error().Err(err).Str("setting", key).Msg("Failed to update auth setting")
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
