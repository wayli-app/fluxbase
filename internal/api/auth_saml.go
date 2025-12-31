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

// ListSAMLProviders returns all enabled SAML providers for app login
// GET /auth/saml/providers
func (h *SAMLHandler) ListSAMLProviders(c *fiber.Ctx) error {
	if h.samlService == nil {
		return c.JSON([]SAMLProviderResponse{})
	}

	// SECURITY: Only list providers that allow app login
	providers := h.samlService.GetProvidersForApp()
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

	// SECURITY: Validate that provider allows app login
	provider, err := h.samlService.GetProvider(providerName)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "SAML provider not found",
		})
	}
	if !provider.AllowAppLogin {
		log.Warn().
			Str("provider", providerName).
			Msg("Attempted to use dashboard-only SAML provider for app login")
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "SAML provider not enabled for application login",
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
	// SECURITY: Only try providers that allow app login
	var assertion *auth.SAMLAssertion
	var providerName string
	var provider *auth.SAMLProvider
	var err error

	for _, p := range h.samlService.GetProvidersForApp() {
		assertion, err = h.samlService.ParseAssertion(p.Name, samlResponse)
		if err == nil {
			providerName = p.Name
			provider = p
			break
		}
	}

	if assertion == nil || provider == nil {
		log.Warn().Msg("Failed to parse SAML assertion with any app-enabled provider")
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

	// RBAC: Validate group membership if configured (OPTIONAL for app users)
	if len(provider.RequiredGroups) > 0 || len(provider.RequiredGroupsAll) > 0 || len(provider.DeniedGroups) > 0 {
		groups := h.samlService.ExtractGroups(providerName, assertion)
		if err := h.samlService.ValidateGroupMembership(provider, groups); err != nil {
			log.Warn().
				Err(err).
				Str("provider", providerName).
				Str("email", email).
				Strs("groups", groups).
				Msg("App SSO access denied due to group membership")
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
	}

	// Provider was already validated above, use it directly

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
		ExpiresIn:    h.authService.GetAccessTokenExpirySeconds(),
		TokenType:    "bearer",
		User:         user,
	})
}

// HandleSAMLLogout handles SAML Single Logout (SLO)
// This endpoint handles both IdP-initiated logout (SAMLRequest) and SP-initiated logout callback (SAMLResponse)
// POST /auth/saml/slo
// GET /auth/saml/slo
func (h *SAMLHandler) HandleSAMLLogout(c *fiber.Ctx) error {
	if h.samlService == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "SAML is not configured",
		})
	}

	// Check for SAMLRequest (IdP-initiated logout) or SAMLResponse (SP-initiated callback)
	var samlRequest, samlResponse, relayState string

	if c.Method() == "POST" {
		samlRequest = c.FormValue("SAMLRequest")
		samlResponse = c.FormValue("SAMLResponse")
		relayState = c.FormValue("RelayState")
	} else {
		samlRequest = c.Query("SAMLRequest")
		samlResponse = c.Query("SAMLResponse")
		relayState = c.Query("RelayState")
	}

	// IdP-initiated logout - IdP sends LogoutRequest
	if samlRequest != "" {
		return h.handleIdPInitiatedLogout(c, samlRequest, relayState, c.Method() == "GET")
	}

	// SP-initiated logout callback - IdP sends LogoutResponse
	if samlResponse != "" {
		return h.handleSPLogoutCallback(c, samlResponse, relayState, c.Method() == "GET")
	}

	return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
		"error": "Missing SAMLRequest or SAMLResponse",
	})
}

// handleIdPInitiatedLogout processes a LogoutRequest from the IdP
func (h *SAMLHandler) handleIdPInitiatedLogout(c *fiber.Ctx, samlRequest, relayState string, isDeflated bool) error {
	ctx := c.Context()

	// Parse the LogoutRequest
	parsedRequest, providerName, err := h.samlService.ParseLogoutRequest(samlRequest, relayState, isDeflated)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to parse SAML LogoutRequest")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid LogoutRequest",
			"details": err.Error(),
		})
	}

	log.Info().
		Str("provider", providerName).
		Str("name_id", parsedRequest.NameID).
		Str("session_index", parsedRequest.SessionIndex).
		Msg("Processing IdP-initiated SAML logout")

	// Find the SAML session by NameID
	samlSession, err := h.samlService.GetSAMLSessionByNameID(ctx, providerName, parsedRequest.NameID)
	if err != nil {
		log.Warn().Err(err).Str("name_id", parsedRequest.NameID).Msg("SAML session not found for logout")
		// Still send success response - IdP expects confirmation even if we don't have the session
	} else {
		// Invalidate the user's JWT sessions
		if err := h.authService.RevokeAllUserTokens(ctx, samlSession.UserID, "SAML IdP-initiated logout"); err != nil {
			log.Warn().Err(err).Str("user_id", samlSession.UserID).Msg("Failed to revoke user tokens during SAML logout")
		}

		// Delete the SAML session
		if err := h.samlService.DeleteSAMLSessionByNameID(ctx, providerName, parsedRequest.NameID); err != nil {
			log.Warn().Err(err).Msg("Failed to delete SAML session")
		}
	}

	// Generate LogoutResponse to send back to IdP
	provider, err := h.samlService.GetProvider(providerName)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get provider for logout response",
		})
	}

	// Get IdP's SLO URL to send the response
	idpSloURL := provider.IdPSloURL
	if idpSloURL == "" {
		// No SLO URL, just return success
		return c.JSON(fiber.Map{
			"message": "logout successful",
		})
	}

	// Generate signed LogoutResponse
	responseURL, err := h.samlService.GenerateLogoutResponse(providerName, parsedRequest.ID, relayState)
	if err != nil {
		log.Error().Err(err).Msg("Failed to generate LogoutResponse")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to generate logout response",
		})
	}

	// Redirect to IdP with the LogoutResponse
	return c.Redirect(responseURL.String(), fiber.StatusFound)
}

// handleSPLogoutCallback processes the LogoutResponse from IdP after SP-initiated logout
func (h *SAMLHandler) handleSPLogoutCallback(c *fiber.Ctx, samlResponse, relayState string, isDeflated bool) error {
	// Parse the LogoutResponse
	parsedResponse, providerName, err := h.samlService.ParseLogoutResponse(samlResponse, isDeflated)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to parse SAML LogoutResponse")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid LogoutResponse",
			"details": err.Error(),
		})
	}

	log.Info().
		Str("provider", providerName).
		Str("status", parsedResponse.Status).
		Str("in_response_to", parsedResponse.InResponseTo).
		Msg("Received SAML LogoutResponse")

	// Check status
	if parsedResponse.Status != "urn:oasis:names:tc:SAML:2.0:status:Success" {
		log.Warn().
			Str("status", parsedResponse.Status).
			Str("message", parsedResponse.StatusMessage).
			Msg("SAML logout failed at IdP")
	}

	// Redirect to the original destination if RelayState is provided
	if relayState != "" {
		redirectURL, err := url.QueryUnescape(relayState)
		if err == nil && redirectURL != "" {
			provider, _ := h.samlService.GetProvider(providerName)
			var allowedHosts []string
			if provider != nil {
				allowedHosts = provider.AllowedRedirectHosts
			}

			validatedURL, err := auth.ValidateRelayState(redirectURL, allowedHosts)
			if err == nil && validatedURL != "" {
				return c.Redirect(validatedURL, fiber.StatusFound)
			}
		}
	}

	// Return JSON response
	return c.JSON(fiber.Map{
		"message": "logout successful",
		"status":  parsedResponse.Status,
	})
}

// InitiateSAMLLogout initiates SP-initiated SAML logout
// GET /auth/saml/logout/:provider
func (h *SAMLHandler) InitiateSAMLLogout(c *fiber.Ctx) error {
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

	// Get the current user from the JWT token
	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "authentication required",
		})
	}

	ctx := c.Context()

	// Get the user's SAML session
	samlSession, err := h.samlService.GetSAMLSessionByUserID(ctx, userID)
	if err != nil {
		log.Debug().Err(err).Str("user_id", userID).Msg("No SAML session found for user")
		// No SAML session - just do local logout
		return c.JSON(fiber.Map{
			"message":     "local logout successful",
			"saml_logout": false,
		})
	}

	// Check if the provider matches
	if samlSession.ProviderName != providerName {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "provider mismatch",
		})
	}

	// Check if IdP supports SLO
	idpSloURL, err := h.samlService.GetIdPSloURL(providerName)
	if err != nil || idpSloURL == "" {
		// No SLO support - do local logout only
		if err := h.samlService.DeleteSAMLSession(ctx, samlSession.ID); err != nil {
			log.Warn().Err(err).Msg("Failed to delete SAML session")
		}
		if err := h.authService.RevokeAllUserTokens(ctx, userID, "SAML logout (no SLO support)"); err != nil {
			log.Warn().Err(err).Msg("Failed to revoke user tokens")
		}
		return c.JSON(fiber.Map{
			"message":     "local logout successful",
			"saml_logout": false,
		})
	}

	// Check if SP signing keys are configured
	if !h.samlService.HasSigningKey(providerName) {
		// No signing key - do local logout only
		if err := h.samlService.DeleteSAMLSession(ctx, samlSession.ID); err != nil {
			log.Warn().Err(err).Msg("Failed to delete SAML session")
		}
		if err := h.authService.RevokeAllUserTokens(ctx, userID, "SAML logout (no signing key)"); err != nil {
			log.Warn().Err(err).Msg("Failed to revoke user tokens")
		}
		return c.JSON(fiber.Map{
			"message":     "local logout successful",
			"saml_logout": false,
			"warning":     "SP signing key not configured for SAML SLO",
		})
	}

	// Get redirect URL from query parameter (where to redirect after logout)
	redirectURL := c.Query("redirect_url", c.Query("redirect", ""))
	relayState := ""
	if redirectURL != "" {
		relayState = url.QueryEscape(redirectURL)
	}

	// Generate the LogoutRequest
	result, err := h.samlService.GenerateLogoutRequest(
		providerName,
		samlSession.NameID,
		samlSession.NameIDFormat,
		samlSession.SessionIndex,
		relayState,
	)
	if err != nil {
		log.Error().Err(err).Msg("Failed to generate SAML LogoutRequest")
		// Fall back to local logout
		if err := h.samlService.DeleteSAMLSession(ctx, samlSession.ID); err != nil {
			log.Warn().Err(err).Msg("Failed to delete SAML session")
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "failed to initiate SAML logout",
			"details": err.Error(),
		})
	}

	// Delete local session before redirecting (IdP will confirm logout)
	if err := h.samlService.DeleteSAMLSession(ctx, samlSession.ID); err != nil {
		log.Warn().Err(err).Msg("Failed to delete SAML session")
	}

	// Revoke JWT tokens
	if err := h.authService.RevokeAllUserTokens(ctx, userID, "SAML SP-initiated logout"); err != nil {
		log.Warn().Err(err).Msg("Failed to revoke user tokens")
	}

	// Check if client wants JSON response or redirect
	acceptHeader := c.Get("Accept")
	if strings.Contains(acceptHeader, "application/json") {
		return c.JSON(fiber.Map{
			"redirect_url": result.RedirectURL,
			"saml_logout":  true,
		})
	}

	// Redirect to IdP for logout
	return c.Redirect(result.RedirectURL, fiber.StatusFound)
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

	// SLO endpoints
	saml.Get("/logout/:provider", h.InitiateSAMLLogout) // SP-initiated logout (requires auth)
	saml.Post("/slo", h.HandleSAMLLogout)               // IdP-initiated logout & SP callback
	saml.Get("/slo", h.HandleSAMLLogout)                // Some IdPs use GET for SLO
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
