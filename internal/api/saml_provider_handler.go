package api

import (
	"context"
	"database/sql"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/crewjam/saml/samlsp"
	"github.com/fluxbase-eu/fluxbase/internal/auth"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

// SAMLProviderHandler handles SAML provider configuration management
type SAMLProviderHandler struct {
	db          *pgxpool.Pool
	samlService *auth.SAMLService
	httpClient  *http.Client
}

// NewSAMLProviderHandler creates a new SAML provider handler
func NewSAMLProviderHandler(db *pgxpool.Pool, samlService *auth.SAMLService) *SAMLProviderHandler {
	return &SAMLProviderHandler{
		db:          db,
		samlService: samlService,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SAMLProviderConfig represents a SAML provider configuration for API responses
type SAMLProviderConfig struct {
	ID                   uuid.UUID         `json:"id"`
	Name                 string            `json:"name"`
	DisplayName          string            `json:"display_name"`
	Enabled              bool              `json:"enabled"`
	EntityID             string            `json:"entity_id"`
	AcsURL               string            `json:"acs_url"`
	IdPMetadataURL       *string           `json:"idp_metadata_url,omitempty"`
	IdPMetadataXML       *string           `json:"idp_metadata_xml,omitempty"`
	IdPEntityID          *string           `json:"idp_entity_id,omitempty"`
	IdPSsoURL            *string           `json:"idp_sso_url,omitempty"`
	AttributeMapping     map[string]string `json:"attribute_mapping"`
	AutoCreateUsers      bool              `json:"auto_create_users"`
	DefaultRole          string            `json:"default_role"`
	AllowDashboardLogin  bool              `json:"allow_dashboard_login"`
	AllowAppLogin        bool              `json:"allow_app_login"`
	AllowIDPInitiated    bool              `json:"allow_idp_initiated"`
	AllowedRedirectHosts []string          `json:"allowed_redirect_hosts"`
	RequiredGroups       []string          `json:"required_groups,omitempty"`
	RequiredGroupsAll    []string          `json:"required_groups_all,omitempty"`
	DeniedGroups         []string          `json:"denied_groups,omitempty"`
	GroupAttribute       string            `json:"group_attribute,omitempty"`
	Source               string            `json:"source"` // "database" or "config"
	CreatedAt            time.Time         `json:"created_at"`
	UpdatedAt            time.Time         `json:"updated_at"`
}

// CreateSAMLProviderRequest represents a request to create a SAML provider
type CreateSAMLProviderRequest struct {
	Name                 string            `json:"name"`
	DisplayName          string            `json:"display_name"`
	Enabled              bool              `json:"enabled"`
	IdPMetadataURL       *string           `json:"idp_metadata_url,omitempty"`
	IdPMetadataXML       *string           `json:"idp_metadata_xml,omitempty"`
	AttributeMapping     map[string]string `json:"attribute_mapping,omitempty"`
	AutoCreateUsers      *bool             `json:"auto_create_users,omitempty"`
	DefaultRole          *string           `json:"default_role,omitempty"`
	AllowDashboardLogin  *bool             `json:"allow_dashboard_login,omitempty"`
	AllowAppLogin        *bool             `json:"allow_app_login,omitempty"`
	AllowIDPInitiated    *bool             `json:"allow_idp_initiated,omitempty"`
	AllowedRedirectHosts []string          `json:"allowed_redirect_hosts,omitempty"`
	RequiredGroups       []string          `json:"required_groups,omitempty"`
	RequiredGroupsAll    []string          `json:"required_groups_all,omitempty"`
	DeniedGroups         []string          `json:"denied_groups,omitempty"`
	GroupAttribute       *string           `json:"group_attribute,omitempty"`
}

// UpdateSAMLProviderRequest represents a request to update a SAML provider
type UpdateSAMLProviderRequest struct {
	DisplayName          *string           `json:"display_name,omitempty"`
	Enabled              *bool             `json:"enabled,omitempty"`
	IdPMetadataURL       *string           `json:"idp_metadata_url,omitempty"`
	IdPMetadataXML       *string           `json:"idp_metadata_xml,omitempty"`
	AttributeMapping     map[string]string `json:"attribute_mapping,omitempty"`
	AutoCreateUsers      *bool             `json:"auto_create_users,omitempty"`
	DefaultRole          *string           `json:"default_role,omitempty"`
	AllowDashboardLogin  *bool             `json:"allow_dashboard_login,omitempty"`
	AllowAppLogin        *bool             `json:"allow_app_login,omitempty"`
	AllowIDPInitiated    *bool             `json:"allow_idp_initiated,omitempty"`
	AllowedRedirectHosts []string          `json:"allowed_redirect_hosts,omitempty"`
	RequiredGroups       []string          `json:"required_groups,omitempty"`
	RequiredGroupsAll    []string          `json:"required_groups_all,omitempty"`
	DeniedGroups         []string          `json:"denied_groups,omitempty"`
	GroupAttribute       *string           `json:"group_attribute,omitempty"`
}

// ValidateMetadataRequest represents a request to validate SAML metadata
type ValidateMetadataRequest struct {
	MetadataURL *string `json:"metadata_url,omitempty"`
	MetadataXML *string `json:"metadata_xml,omitempty"`
}

// ValidateMetadataResponse represents the response from metadata validation
type ValidateMetadataResponse struct {
	Valid       bool    `json:"valid"`
	EntityID    string  `json:"entity_id,omitempty"`
	SsoURL      string  `json:"sso_url,omitempty"`
	SloURL      string  `json:"slo_url,omitempty"`
	Certificate string  `json:"certificate,omitempty"`
	Error       *string `json:"error,omitempty"`
}

var samlProviderNamePattern = regexp.MustCompile(`^[a-z][a-z0-9_-]{1,49}$`)

// ListSAMLProviders lists all SAML providers (database + config)
func (h *SAMLProviderHandler) ListSAMLProviders(c *fiber.Ctx) error {
	ctx := c.Context()

	// Get database-managed providers
	query := `
		SELECT id, name, COALESCE(display_name, name), enabled, entity_id, acs_url,
		       idp_metadata_url, idp_metadata_xml, attribute_mapping, auto_create_users,
		       default_role, COALESCE(allow_dashboard_login, false), COALESCE(allow_app_login, true),
		       COALESCE(allow_idp_initiated, false), COALESCE(allowed_redirect_hosts, ARRAY[]::TEXT[]),
		       COALESCE(required_groups, ARRAY[]::TEXT[]), COALESCE(required_groups_all, ARRAY[]::TEXT[]),
		       COALESCE(denied_groups, ARRAY[]::TEXT[]), COALESCE(group_attribute, 'groups'),
		       COALESCE(source, 'database'), created_at, updated_at
		FROM auth.saml_providers
		ORDER BY name
	`

	rows, err := h.db.Query(ctx, query)
	if err != nil {
		log.Error().Err(err).Msg("Failed to list SAML providers")
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to retrieve SAML providers",
		})
	}
	defer rows.Close()

	providers := []SAMLProviderConfig{}
	dbProviderNames := make(map[string]bool)

	for rows.Next() {
		var p SAMLProviderConfig
		var attrMapping map[string]string

		err := rows.Scan(
			&p.ID, &p.Name, &p.DisplayName, &p.Enabled, &p.EntityID, &p.AcsURL,
			&p.IdPMetadataURL, &p.IdPMetadataXML, &attrMapping, &p.AutoCreateUsers,
			&p.DefaultRole, &p.AllowDashboardLogin, &p.AllowAppLogin,
			&p.AllowIDPInitiated, &p.AllowedRedirectHosts,
			&p.RequiredGroups, &p.RequiredGroupsAll, &p.DeniedGroups, &p.GroupAttribute,
			&p.Source, &p.CreatedAt, &p.UpdatedAt,
		)
		if err != nil {
			log.Error().Err(err).Msg("Failed to scan SAML provider")
			continue
		}

		p.AttributeMapping = attrMapping
		if p.AttributeMapping == nil {
			p.AttributeMapping = make(map[string]string)
		}

		// Don't return raw metadata XML in list (too large)
		p.IdPMetadataXML = nil

		providers = append(providers, p)
		dbProviderNames[p.Name] = true
	}

	// Also include config-file providers that aren't in the database
	if h.samlService != nil {
		for _, configProvider := range h.samlService.ListProviders() {
			if !dbProviderNames[configProvider.Name] {
				providers = append(providers, SAMLProviderConfig{
					ID:                   uuid.MustParse(configProvider.ID),
					Name:                 configProvider.Name,
					DisplayName:          configProvider.Name,
					Enabled:              configProvider.Enabled,
					EntityID:             configProvider.EntityID,
					AcsURL:               configProvider.AcsURL,
					AttributeMapping:     configProvider.AttributeMapping,
					AutoCreateUsers:      configProvider.AutoCreateUsers,
					DefaultRole:          configProvider.DefaultRole,
					AllowDashboardLogin:  false, // Config providers default to app-only
					AllowAppLogin:        true,
					AllowIDPInitiated:    configProvider.AllowIDPInitiated,
					AllowedRedirectHosts: configProvider.AllowedRedirectHosts,
					Source:               "config",
					CreatedAt:            configProvider.CreatedAt,
					UpdatedAt:            configProvider.UpdatedAt,
				})
			}
		}
	}

	return c.JSON(providers)
}

// GetSAMLProvider gets a single SAML provider by ID
func (h *SAMLProviderHandler) GetSAMLProvider(c *fiber.Ctx) error {
	ctx := c.Context()
	id := c.Params("id")

	providerID, err := uuid.Parse(id)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid provider ID",
		})
	}

	query := `
		SELECT id, name, COALESCE(display_name, name), enabled, entity_id, acs_url,
		       idp_metadata_url, idp_metadata_xml, attribute_mapping, auto_create_users,
		       default_role, COALESCE(allow_dashboard_login, false), COALESCE(allow_app_login, true),
		       COALESCE(allow_idp_initiated, false), COALESCE(allowed_redirect_hosts, ARRAY[]::TEXT[]),
		       COALESCE(source, 'database'), created_at, updated_at
		FROM auth.saml_providers
		WHERE id = $1
	`

	var p SAMLProviderConfig
	var attrMapping map[string]string

	err = h.db.QueryRow(ctx, query, providerID).Scan(
		&p.ID, &p.Name, &p.DisplayName, &p.Enabled, &p.EntityID, &p.AcsURL,
		&p.IdPMetadataURL, &p.IdPMetadataXML, &attrMapping, &p.AutoCreateUsers,
		&p.DefaultRole, &p.AllowDashboardLogin, &p.AllowAppLogin,
		&p.AllowIDPInitiated, &p.AllowedRedirectHosts, &p.Source,
		&p.CreatedAt, &p.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return c.Status(404).JSON(fiber.Map{
			"error": "SAML provider not found",
		})
	}
	if err != nil {
		log.Error().Err(err).Str("id", id).Msg("Failed to get SAML provider")
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to retrieve SAML provider",
		})
	}

	p.AttributeMapping = attrMapping
	if p.AttributeMapping == nil {
		p.AttributeMapping = make(map[string]string)
	}

	return c.JSON(p)
}

// CreateSAMLProvider creates a new SAML provider
func (h *SAMLProviderHandler) CreateSAMLProvider(c *fiber.Ctx) error {
	ctx := c.Context()
	var req CreateSAMLProviderRequest

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate provider name
	if !samlProviderNamePattern.MatchString(req.Name) {
		return c.Status(400).JSON(fiber.Map{
			"error": "Provider name must start with a letter and contain only lowercase letters, numbers, underscores, and hyphens (2-50 chars)",
		})
	}

	// Require metadata URL or XML
	if (req.IdPMetadataURL == nil || *req.IdPMetadataURL == "") &&
		(req.IdPMetadataXML == nil || *req.IdPMetadataXML == "") {
		return c.Status(400).JSON(fiber.Map{
			"error": "Either idp_metadata_url or idp_metadata_xml must be provided",
		})
	}

	// Validate and parse metadata
	metadataInfo, err := h.validateMetadata(ctx, req.IdPMetadataURL, req.IdPMetadataXML)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": fmt.Sprintf("Invalid IdP metadata: %v", err),
		})
	}

	// Set defaults
	displayName := req.DisplayName
	if displayName == "" {
		displayName = req.Name
	}

	autoCreateUsers := true
	if req.AutoCreateUsers != nil {
		autoCreateUsers = *req.AutoCreateUsers
	}

	defaultRole := "authenticated"
	if req.DefaultRole != nil && *req.DefaultRole != "" {
		defaultRole = *req.DefaultRole
	}

	allowDashboardLogin := false
	if req.AllowDashboardLogin != nil {
		allowDashboardLogin = *req.AllowDashboardLogin
	}

	allowAppLogin := true
	if req.AllowAppLogin != nil {
		allowAppLogin = *req.AllowAppLogin
	}

	allowIDPInitiated := false
	if req.AllowIDPInitiated != nil {
		allowIDPInitiated = *req.AllowIDPInitiated
	}

	groupAttribute := "groups"
	if req.GroupAttribute != nil && *req.GroupAttribute != "" {
		groupAttribute = *req.GroupAttribute
	}

	attrMapping := req.AttributeMapping
	if attrMapping == nil {
		attrMapping = map[string]string{
			"email": "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress",
			"name":  "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/name",
		}
	}

	// Generate entity ID and ACS URL
	baseURL := c.BaseURL()
	entityID := fmt.Sprintf("%s/api/v1/auth/saml/metadata/%s", baseURL, req.Name)
	acsURL := fmt.Sprintf("%s/api/v1/auth/saml/acs", baseURL)

	query := `
		INSERT INTO auth.saml_providers (
			name, display_name, enabled, entity_id, acs_url, idp_metadata_url,
			idp_metadata_xml, idp_metadata_cached, idp_metadata_cached_at,
			attribute_mapping, auto_create_users, default_role,
			allow_dashboard_login, allow_app_login, allow_idp_initiated,
			allowed_redirect_hosts, required_groups, required_groups_all, denied_groups, group_attribute, source
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, 'database')
		RETURNING id, created_at, updated_at
	`

	var id uuid.UUID
	var createdAt, updatedAt time.Time

	err = h.db.QueryRow(
		ctx, query,
		req.Name, displayName, req.Enabled, entityID, acsURL,
		req.IdPMetadataURL, req.IdPMetadataXML, metadataInfo.CachedXML,
		attrMapping, autoCreateUsers, defaultRole,
		allowDashboardLogin, allowAppLogin, allowIDPInitiated,
		req.AllowedRedirectHosts, req.RequiredGroups, req.RequiredGroupsAll, req.DeniedGroups, groupAttribute,
	).Scan(&id, &createdAt, &updatedAt)

	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "unique constraint") {
			return c.Status(409).JSON(fiber.Map{
				"error": fmt.Sprintf("SAML provider '%s' already exists", req.Name),
			})
		}
		log.Error().Err(err).Str("provider", req.Name).Msg("Failed to create SAML provider")
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to create SAML provider",
		})
	}

	// Reload SAML service to pick up new provider
	if h.samlService != nil {
		if err := h.reloadSAMLProvider(ctx, req.Name); err != nil {
			log.Warn().Err(err).Str("provider", req.Name).Msg("Failed to reload SAML provider into service")
		}
	}

	log.Info().Str("id", id.String()).Str("provider", req.Name).Msg("SAML provider created")

	return c.Status(201).JSON(fiber.Map{
		"success":    true,
		"id":         id,
		"provider":   req.Name,
		"entity_id":  entityID,
		"acs_url":    acsURL,
		"message":    fmt.Sprintf("SAML provider '%s' created successfully", displayName),
		"created_at": createdAt.Format(time.RFC3339),
		"updated_at": updatedAt.Format(time.RFC3339),
	})
}

// UpdateSAMLProvider updates an existing SAML provider
func (h *SAMLProviderHandler) UpdateSAMLProvider(c *fiber.Ctx) error {
	ctx := c.Context()
	id := c.Params("id")

	providerID, err := uuid.Parse(id)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid provider ID",
		})
	}

	// Check if provider exists and is database-managed
	var source string
	var providerName string
	err = h.db.QueryRow(ctx, "SELECT name, COALESCE(source, 'database') FROM auth.saml_providers WHERE id = $1", providerID).Scan(&providerName, &source)
	if err == sql.ErrNoRows {
		return c.Status(404).JSON(fiber.Map{
			"error": "SAML provider not found",
		})
	}
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to check provider",
		})
	}

	if source == "config" {
		return c.Status(403).JSON(fiber.Map{
			"error": "Cannot modify config-file managed providers. Edit your fluxbase.yaml instead.",
		})
	}

	var req UpdateSAMLProviderRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// If metadata is being updated, validate it
	var metadataInfo *metadataValidationResult
	if (req.IdPMetadataURL != nil && *req.IdPMetadataURL != "") ||
		(req.IdPMetadataXML != nil && *req.IdPMetadataXML != "") {
		metadataInfo, err = h.validateMetadata(ctx, req.IdPMetadataURL, req.IdPMetadataXML)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{
				"error": fmt.Sprintf("Invalid IdP metadata: %v", err),
			})
		}
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
	if req.IdPMetadataURL != nil {
		updates = append(updates, fmt.Sprintf("idp_metadata_url = $%d", argPos))
		args = append(args, *req.IdPMetadataURL)
		argPos++
	}
	if req.IdPMetadataXML != nil {
		updates = append(updates, fmt.Sprintf("idp_metadata_xml = $%d", argPos))
		args = append(args, *req.IdPMetadataXML)
		argPos++
	}
	if metadataInfo != nil {
		updates = append(updates, fmt.Sprintf("idp_metadata_cached = $%d", argPos))
		args = append(args, metadataInfo.CachedXML)
		argPos++
		updates = append(updates, "idp_metadata_cached_at = NOW()")
	}
	if req.AttributeMapping != nil {
		updates = append(updates, fmt.Sprintf("attribute_mapping = $%d", argPos))
		args = append(args, req.AttributeMapping)
		argPos++
	}
	if req.AutoCreateUsers != nil {
		updates = append(updates, fmt.Sprintf("auto_create_users = $%d", argPos))
		args = append(args, *req.AutoCreateUsers)
		argPos++
	}
	if req.DefaultRole != nil {
		updates = append(updates, fmt.Sprintf("default_role = $%d", argPos))
		args = append(args, *req.DefaultRole)
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
	if req.AllowIDPInitiated != nil {
		updates = append(updates, fmt.Sprintf("allow_idp_initiated = $%d", argPos))
		args = append(args, *req.AllowIDPInitiated)
		argPos++
	}
	if req.AllowedRedirectHosts != nil {
		updates = append(updates, fmt.Sprintf("allowed_redirect_hosts = $%d", argPos))
		args = append(args, req.AllowedRedirectHosts)
		argPos++
	}
	if req.RequiredGroups != nil {
		updates = append(updates, fmt.Sprintf("required_groups = $%d", argPos))
		args = append(args, req.RequiredGroups)
		argPos++
	}
	if req.RequiredGroupsAll != nil {
		updates = append(updates, fmt.Sprintf("required_groups_all = $%d", argPos))
		args = append(args, req.RequiredGroupsAll)
		argPos++
	}
	if req.DeniedGroups != nil {
		updates = append(updates, fmt.Sprintf("denied_groups = $%d", argPos))
		args = append(args, req.DeniedGroups)
		argPos++
	}
	if req.GroupAttribute != nil {
		updates = append(updates, fmt.Sprintf("group_attribute = $%d", argPos))
		args = append(args, *req.GroupAttribute)
		argPos++
	}

	if len(updates) == 0 {
		return c.Status(400).JSON(fiber.Map{
			"error": "No fields to update",
		})
	}

	query := fmt.Sprintf(
		"UPDATE auth.saml_providers SET %s WHERE id = $1 RETURNING display_name",
		strings.Join(updates, ", "),
	)

	var displayName string
	err = h.db.QueryRow(ctx, query, args...).Scan(&displayName)

	if err == sql.ErrNoRows {
		return c.Status(404).JSON(fiber.Map{
			"error": "SAML provider not found",
		})
	}
	if err != nil {
		log.Error().Err(err).Str("id", id).Msg("Failed to update SAML provider")
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to update SAML provider",
		})
	}

	// Reload SAML service
	if h.samlService != nil {
		if err := h.reloadSAMLProvider(ctx, providerName); err != nil {
			log.Warn().Err(err).Str("provider", providerName).Msg("Failed to reload SAML provider into service")
		}
	}

	log.Info().Str("id", id).Msg("SAML provider updated")

	return c.JSON(fiber.Map{
		"success": true,
		"message": fmt.Sprintf("SAML provider '%s' updated successfully", displayName),
	})
}

// DeleteSAMLProvider deletes a SAML provider
func (h *SAMLProviderHandler) DeleteSAMLProvider(c *fiber.Ctx) error {
	ctx := c.Context()
	id := c.Params("id")

	providerID, err := uuid.Parse(id)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid provider ID",
		})
	}

	// Check source before deleting
	var source string
	var providerName string
	err = h.db.QueryRow(ctx, "SELECT name, COALESCE(source, 'database') FROM auth.saml_providers WHERE id = $1", providerID).Scan(&providerName, &source)
	if err == sql.ErrNoRows {
		return c.Status(404).JSON(fiber.Map{
			"error": "SAML provider not found",
		})
	}
	if source == "config" {
		return c.Status(403).JSON(fiber.Map{
			"error": "Cannot delete config-file managed providers. Remove from your fluxbase.yaml instead.",
		})
	}

	query := "DELETE FROM auth.saml_providers WHERE id = $1 RETURNING display_name"

	var displayName string
	err = h.db.QueryRow(ctx, query, providerID).Scan(&displayName)

	if err == sql.ErrNoRows {
		return c.Status(404).JSON(fiber.Map{
			"error": "SAML provider not found",
		})
	}
	if err != nil {
		log.Error().Err(err).Str("id", id).Msg("Failed to delete SAML provider")
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to delete SAML provider",
		})
	}

	// Remove from SAML service
	if h.samlService != nil {
		h.samlService.RemoveProvider(providerName)
	}

	log.Info().Str("id", id).Str("provider", displayName).Msg("SAML provider deleted")

	return c.JSON(fiber.Map{
		"success": true,
		"message": fmt.Sprintf("SAML provider '%s' deleted successfully", displayName),
	})
}

// ValidateMetadata validates SAML IdP metadata from URL or XML
func (h *SAMLProviderHandler) ValidateMetadata(c *fiber.Ctx) error {
	var req ValidateMetadataRequest

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if (req.MetadataURL == nil || *req.MetadataURL == "") &&
		(req.MetadataXML == nil || *req.MetadataXML == "") {
		return c.Status(400).JSON(fiber.Map{
			"error": "Either metadata_url or metadata_xml must be provided",
		})
	}

	result, err := h.validateMetadata(c.Context(), req.MetadataURL, req.MetadataXML)
	if err != nil {
		errStr := err.Error()
		return c.JSON(ValidateMetadataResponse{
			Valid: false,
			Error: &errStr,
		})
	}

	return c.JSON(ValidateMetadataResponse{
		Valid:       true,
		EntityID:    result.EntityID,
		SsoURL:      result.SsoURL,
		SloURL:      result.SloURL,
		Certificate: result.Certificate,
	})
}

// UploadMetadata handles file upload for IdP metadata XML
func (h *SAMLProviderHandler) UploadMetadata(c *fiber.Ctx) error {
	file, err := c.FormFile("metadata")
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "No metadata file provided",
		})
	}

	// Check file size (max 1MB)
	if file.Size > 1024*1024 {
		return c.Status(400).JSON(fiber.Map{
			"error": "Metadata file too large (max 1MB)",
		})
	}

	// Read file content
	f, err := file.Open()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to read file",
		})
	}
	defer f.Close()

	content, err := io.ReadAll(f)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to read file content",
		})
	}

	xmlStr := string(content)
	result, err := h.validateMetadata(c.Context(), nil, &xmlStr)
	if err != nil {
		errStr := err.Error()
		return c.JSON(ValidateMetadataResponse{
			Valid: false,
			Error: &errStr,
		})
	}

	return c.JSON(fiber.Map{
		"valid":       true,
		"entity_id":   result.EntityID,
		"sso_url":     result.SsoURL,
		"slo_url":     result.SloURL,
		"certificate": result.Certificate,
		"metadata":    xmlStr,
	})
}

// GetSPMetadata returns the Service Provider metadata XML for a provider
func (h *SAMLProviderHandler) GetSPMetadata(c *fiber.Ctx) error {
	providerName := c.Params("provider")

	if h.samlService == nil {
		return c.Status(503).JSON(fiber.Map{
			"error": "SAML service not available",
		})
	}

	metadata, err := h.samlService.GetSPMetadata(providerName)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": "Provider not found or not initialized",
		})
	}

	c.Set("Content-Type", "application/xml")
	return c.Send(metadata)
}

// metadataValidationResult holds parsed metadata info
type metadataValidationResult struct {
	EntityID    string
	SsoURL      string
	SloURL      string
	Certificate string
	CachedXML   string
}

// validateMetadata fetches and validates SAML metadata
func (h *SAMLProviderHandler) validateMetadata(ctx context.Context, metadataURL, metadataXML *string) (*metadataValidationResult, error) {
	var xmlData []byte
	var err error

	if metadataURL != nil && *metadataURL != "" {
		// Validate URL
		if !strings.HasPrefix(*metadataURL, "https://") {
			return nil, fmt.Errorf("metadata URL must use HTTPS")
		}

		// Fetch metadata
		req, err := http.NewRequestWithContext(ctx, "GET", *metadataURL, nil)
		if err != nil {
			return nil, fmt.Errorf("invalid URL: %w", err)
		}

		resp, err := h.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch metadata: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("metadata fetch returned status %d", resp.StatusCode)
		}

		xmlData, err = io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read metadata: %w", err)
		}
	} else if metadataXML != nil && *metadataXML != "" {
		xmlData = []byte(*metadataXML)
	} else {
		return nil, fmt.Errorf("no metadata provided")
	}

	// Check if valid XML
	var test interface{}
	if err := xml.Unmarshal(xmlData, &test); err != nil {
		return nil, fmt.Errorf("invalid XML: %w", err)
	}

	// Parse SAML metadata
	metadata, err := samlsp.ParseMetadata(xmlData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse SAML metadata: %w", err)
	}

	result := &metadataValidationResult{
		EntityID:  metadata.EntityID,
		CachedXML: string(xmlData),
	}

	// Find IdP descriptor
	for _, desc := range metadata.IDPSSODescriptors {
		// Get SSO URL
		for _, sso := range desc.SingleSignOnServices {
			if result.SsoURL == "" {
				result.SsoURL = sso.Location
			}
		}

		// Get SLO URL
		for _, slo := range desc.SingleLogoutServices {
			if result.SloURL == "" {
				result.SloURL = slo.Location
			}
		}

		// Get certificate
		for _, kd := range desc.KeyDescriptors {
			if kd.Use == "signing" || kd.Use == "" {
				for _, cert := range kd.KeyInfo.X509Data.X509Certificates {
					if result.Certificate == "" {
						// Return truncated cert for display
						if len(cert.Data) > 100 {
							result.Certificate = cert.Data[:50] + "..." + cert.Data[len(cert.Data)-50:]
						} else {
							result.Certificate = cert.Data
						}
					}
				}
			}
		}
	}

	if result.SsoURL == "" {
		return nil, fmt.Errorf("no SSO URL found in metadata")
	}

	return result, nil
}

// reloadSAMLProvider reloads a provider into the SAML service from the database
func (h *SAMLProviderHandler) reloadSAMLProvider(ctx context.Context, name string) error {
	// For now, just log that we need to implement this
	// The full implementation would reload the provider from DB into the SAML service
	log.Info().Str("provider", name).Msg("SAML provider needs reload (not yet implemented)")
	return nil
}
