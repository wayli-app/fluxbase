package api

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/fluxbase-eu/fluxbase/internal/auth"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// SAMLHandler handles SAML SSO endpoints
type SAMLHandler struct {
	samlService *auth.SAMLService
	authService *auth.Service
}

// NewSAMLHandler creates a new SAML handler
func NewSAMLHandler(samlService *auth.SAMLService, authService *auth.Service) *SAMLHandler {
	return &SAMLHandler{
		samlService: samlService,
		authService: authService,
	}
}

// SAMLProviderResponse represents a SAML provider for API responses
type SAMLProviderResponse struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	EntityID string `json:"entity_id"`
	SsoURL   string `json:"sso_url"`
	LoginURL string `json:"login_url"`
	Enabled  bool   `json:"enabled"`
}

// SAMLLoginResponse represents the response for initiating SAML login
type SAMLLoginResponse struct {
	RedirectURL string `json:"redirect_url"`
}

// SAMLCallbackResponse represents the response after successful SAML authentication
type SAMLCallbackResponse struct {
	AccessToken  string     `json:"access_token"`
	RefreshToken string     `json:"refresh_token"`
	ExpiresIn    int64      `json:"expires_in"`
	TokenType    string     `json:"token_type"`
	User         *auth.User `json:"user"`
}

// ListSAMLProviders returns all enabled SAML providers
// GET /auth/saml/providers
func (h *SAMLHandler) ListSAMLProviders(c *fiber.Ctx) error {
	if h.samlService == nil {
		return c.JSON([]SAMLProviderResponse{})
	}

	providers := h.samlService.ListProviders()
	response := make([]SAMLProviderResponse, 0, len(providers))

	baseURL := c.BaseURL()

	for _, p := range providers {
		response = append(response, SAMLProviderResponse{
			ID:       p.ID,
			Name:     p.Name,
			EntityID: p.EntityID,
			SsoURL:   p.SsoURL,
			LoginURL: fmt.Sprintf("%s/auth/saml/login/%s", baseURL, p.Name),
			Enabled:  p.Enabled,
		})
	}

	return c.JSON(response)
}

// GetSPMetadata returns the SP metadata XML for a provider
// GET /auth/saml/metadata/:provider
func (h *SAMLHandler) GetSPMetadata(c *fiber.Ctx) error {
	if h.samlService == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "SAML is not configured",
		})
	}

	providerName := c.Params("provider")
	if providerName == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "provider name is required",
		})
	}

	metadata, err := h.samlService.GetSPMetadata(providerName)
	if err != nil {
		if err == auth.ErrSAMLProviderNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "SAML provider not found",
			})
		}
		log.Error().Err(err).Str("provider", providerName).Msg("Failed to get SP metadata")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to generate metadata",
		})
	}

	c.Set("Content-Type", "application/xml")
	return c.Send(metadata)
}

// InitiateSAMLLogin initiates SAML login by redirecting to the IdP
// GET /auth/saml/login/:provider
func (h *SAMLHandler) InitiateSAMLLogin(c *fiber.Ctx) error {
	if h.samlService == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "SAML is not configured",
		})
	}

	providerName := c.Params("provider")
	if providerName == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "provider name is required",
		})
	}

	// Get redirect URL from query parameter (where to redirect after login)
	redirectURL := c.Query("redirect_url", c.Query("redirect", ""))

	// Create relay state with redirect URL
	relayState := ""
	if redirectURL != "" {
		relayState = url.QueryEscape(redirectURL)
	}

	// Generate AuthnRequest and get redirect URL
	authURL, _, err := h.samlService.GenerateAuthRequest(providerName, relayState)
	if err != nil {
		if err == auth.ErrSAMLProviderNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "SAML provider not found",
			})
		}
		if err == auth.ErrSAMLProviderDisabled {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "SAML provider is disabled",
			})
		}
		log.Error().Err(err).Str("provider", providerName).Msg("Failed to generate SAML AuthnRequest")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to initiate SAML login",
		})
	}

	// Check if client wants JSON response (API call) or redirect (browser)
	acceptHeader := c.Get("Accept")
	if strings.Contains(acceptHeader, "application/json") {
		return c.JSON(SAMLLoginResponse{
			RedirectURL: authURL,
		})
	}

	// Redirect to IdP
	return c.Redirect(authURL, fiber.StatusFound)
}

// HandleSAMLAssertion handles the SAML assertion callback from the IdP
// POST /auth/saml/acs
func (h *SAMLHandler) HandleSAMLAssertion(c *fiber.Ctx) error {
	if h.samlService == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "SAML is not configured",
		})
	}

	// Get SAML response from form data
	samlResponse := c.FormValue("SAMLResponse")
	if samlResponse == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "SAMLResponse is required",
		})
	}

	relayState := c.FormValue("RelayState")

	// Determine provider from the request
	// The provider name could be in the RelayState or we need to decode the response to find out
	// For now, try all providers until one validates
	var assertion *auth.SAMLAssertion
	var providerName string
	var err error

	for _, provider := range h.samlService.ListProviders() {
		assertion, err = h.samlService.ParseAssertion(provider.Name, samlResponse)
		if err == nil {
			providerName = provider.Name
			break
		}
	}

	if assertion == nil {
		log.Warn().Msg("Failed to parse SAML assertion with any provider")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":   "invalid SAML assertion",
			"details": "The SAML response could not be validated by any configured provider",
		})
	}

	// Check for replay attack
	isReplay, err := h.samlService.CheckAssertionReplay(c.Context(), assertion.ID, assertion.NotOnOrAfter)
	if err != nil {
		log.Error().Err(err).Msg("Failed to check assertion replay")
	}
	if isReplay {
		log.Warn().Str("assertion_id", assertion.ID).Msg("SAML assertion replay detected")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "SAML assertion has already been used",
		})
	}

	// Extract user info from assertion
	email, name, err := h.samlService.ExtractUserInfo(providerName, assertion)
	if err != nil {
		log.Warn().Err(err).Str("provider", providerName).Msg("Failed to extract user info from SAML assertion")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "failed to extract user info",
			"details": err.Error(),
		})
	}

	// Get provider config
	provider, _ := h.samlService.GetProvider(providerName)

	// Find or create user
	ctx := c.Context()
	user, err := h.authService.GetUserByEmail(ctx, email)
	if err != nil {
		// User doesn't exist - check if auto-create is enabled
		if !provider.AutoCreateUsers {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "user does not exist and automatic creation is disabled",
			})
		}

		// Create new user
		user, err = h.authService.CreateSAMLUser(ctx, email, name, providerName, assertion.NameID, assertion.Attributes)
		if err != nil {
			log.Error().Err(err).Str("email", email).Msg("Failed to create SAML user")
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to create user",
			})
		}
	} else {
		// Update existing user's SAML identity link
		if err := h.authService.LinkSAMLIdentity(ctx, user.ID, providerName, assertion.NameID, assertion.Attributes); err != nil {
			log.Warn().Err(err).Str("user_id", user.ID).Msg("Failed to update SAML identity link")
		}
	}

	// Create SAML session for SLO support
	sessionID := uuid.New().String()
	expiresAt := assertion.NotOnOrAfter
	samlSession := &auth.SAMLSession{
		ID:           sessionID,
		UserID:       user.ID,
		ProviderName: providerName,
		NameID:       assertion.NameID,
		NameIDFormat: assertion.NameIDFormat,
		SessionIndex: assertion.SessionIndex,
		Attributes:   convertAttributes(assertion.Attributes),
		ExpiresAt:    &expiresAt,
		CreatedAt:    time.Now(),
	}

	if err := h.samlService.CreateSAMLSession(ctx, samlSession); err != nil {
		log.Warn().Err(err).Msg("Failed to create SAML session")
	}

	// Generate JWT tokens
	signInResp, err := h.authService.GenerateTokensForUser(ctx, user.ID)
	if err != nil {
		log.Error().Err(err).Str("user_id", user.ID).Msg("Failed to generate tokens")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to generate tokens",
		})
	}
	accessToken := signInResp.AccessToken
	refreshToken := signInResp.RefreshToken

	// Determine response type
	// If RelayState contains a redirect URL, redirect with token in fragment
	if relayState != "" {
		redirectURL, err := url.QueryUnescape(relayState)
		if err == nil && redirectURL != "" {
			// Validate RelayState URL to prevent open redirect attacks
			validatedURL, err := auth.ValidateRelayState(redirectURL, provider.AllowedRedirectHosts)
			if err != nil {
				log.Warn().Err(err).Str("relay_state", redirectURL).Msg("Invalid RelayState rejected")
				// Fall through to JSON response instead of redirecting to untrusted URL
			} else if validatedURL != "" {
				// Append tokens to redirect URL as fragment
				separator := "#"
				if strings.Contains(validatedURL, "#") {
					separator = "&"
				}
				validatedURL = fmt.Sprintf("%s%saccess_token=%s&refresh_token=%s&token_type=bearer",
					validatedURL, separator,
					url.QueryEscape(accessToken),
					url.QueryEscape(refreshToken),
				)
				return c.Redirect(validatedURL, fiber.StatusFound)
			}
		}
	}

	// Return JSON response
	return c.JSON(SAMLCallbackResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    3600, // TODO: Get from config
		TokenType:    "bearer",
		User:         user,
	})
}

// HandleSAMLLogout handles SAML Single Logout (SLO)
// POST /auth/saml/slo
func (h *SAMLHandler) HandleSAMLLogout(c *fiber.Ctx) error {
	if h.samlService == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "SAML is not configured",
		})
	}

	// SLO can be initiated by IdP or SP
	// For now, just return success - full SLO implementation requires more work
	return c.JSON(fiber.Map{
		"message": "logout successful",
	})
}

// Helper function to convert map[string][]string to map[string]interface{}
func convertAttributes(attrs map[string][]string) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range attrs {
		if len(v) == 1 {
			result[k] = v[0]
		} else {
			result[k] = v
		}
	}
	return result
}

// RegisterSAMLRoutes registers SAML-related routes
func (h *SAMLHandler) RegisterRoutes(router fiber.Router) {
	saml := router.Group("/saml")

	// Public endpoints
	saml.Get("/providers", h.ListSAMLProviders)
	saml.Get("/metadata/:provider", h.GetSPMetadata)
	saml.Get("/login/:provider", h.InitiateSAMLLogin)
	saml.Post("/acs", h.HandleSAMLAssertion)
	saml.Post("/slo", h.HandleSAMLLogout)
}

// Helper for JSON encoding errors
func jsonError(c *fiber.Ctx, status int, message string) error {
	return c.Status(status).JSON(fiber.Map{
		"error": message,
	})
}

// CreateSAMLUser method to add to auth.Service
type CreateSAMLUserRequest struct {
	Email      string
	Name       string
	Provider   string
	NameID     string
	Attributes map[string][]string
}

// Ensure auth.Service has the required methods - they will need to be added
// The following are stubs showing what needs to be added to internal/auth/service.go:
//
// func (s *Service) CreateSAMLUser(ctx context.Context, email, name, provider, nameID string, attrs map[string][]string) (*User, error)
// func (s *Service) LinkSAMLIdentity(ctx context.Context, userID, provider, nameID string, attrs map[string][]string) error
// func (s *Service) GenerateTokensForUser(ctx context.Context, user *User) (accessToken, refreshToken string, err error)
