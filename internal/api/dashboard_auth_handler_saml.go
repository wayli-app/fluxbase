package api

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/auth"
	"github.com/nimbleflux/fluxbase/internal/database"
	"github.com/nimbleflux/fluxbase/internal/middleware"
)

// InitiateSAMLLogin initiates a SAML login flow for dashboard SSO
func (h *DashboardAuthHandler) InitiateSAMLLogin(c fiber.Ctx) error {
	providerIDOrName := c.Params("provider")
	redirectTo := c.Query("redirect_to", "/")
	ctx := c.RequestCtx()

	if h.samlService == nil {
		return SendNotInitialized(c, "SAML service")
	}

	if err := h.requireDB(c); err != nil {
		return err
	}

	var providerName string
	var allowDashboardLogin bool
	err := database.WrapWithServiceRole(ctx, h.db, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `
			SELECT name, COALESCE(allow_dashboard_login, false)
			FROM auth.saml_providers
			WHERE (id::text = $1 OR name = $1) AND enabled = true
		`, providerIDOrName).Scan(&providerName, &allowDashboardLogin)
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			log.Warn().
				Str("provider_id", providerIDOrName).
				Msg("SAML provider not found for dashboard login")
			return SendNotFound(c, "SAML provider not found or not enabled for dashboard login")
		}
		return SendInternalError(c, "Failed to fetch SAML provider")
	}

	// Check if provider allows dashboard login
	if !allowDashboardLogin {
		log.Warn().
			Str("provider", providerName).
			Msg("SAML provider not enabled for dashboard login")
		return SendForbidden(c, "SAML provider not enabled for dashboard login", ErrCodeAccessDenied)
	}

	// Get provider from service (by name)
	provider, err := h.samlService.GetProvider(providerName)
	if err != nil || provider == nil {
		return SendNotFound(c, "SAML provider not found")
	}

	// Generate SAML AuthnRequest
	authURL, _, err := h.samlService.GenerateAuthRequest(providerName, redirectTo)
	if err != nil {
		return SendInternalError(c, "Failed to create SAML request")
	}

	return c.Redirect().To(authURL)
}

// SAMLACSCallback handles the SAML Assertion Consumer Service callback for dashboard SSO
func (h *DashboardAuthHandler) SAMLACSCallback(c fiber.Ctx) error {
	ctx := c.RequestCtx()

	if h.samlService == nil {
		return SendNotInitialized(c, "SAML service")
	}

	samlResponse := c.FormValue("SAMLResponse")
	relayState := c.FormValue("RelayState")

	if samlResponse == "" {
		return c.Redirect().To("/admin/login?error=" + url.QueryEscape("Missing SAML response"))
	}

	// Find the provider from relay state or try all dashboard-enabled providers
	var assertion *auth.SAMLAssertion
	var providerName string
	var parseErr error

	// Get all dashboard-enabled SAML providers
	dashboardProviders := h.samlService.GetProvidersForDashboardWithTenant(c.RequestCtx(), middleware.GetTenantIDFromContext(c))

	// If no dashboard providers configured
	if len(dashboardProviders) == 0 {
		log.Warn().Msg("No SAML providers enabled for dashboard login")
		return c.Redirect().To("/admin/login?error=" + url.QueryEscape("No SAML providers configured for dashboard"))
	}

	for _, provider := range dashboardProviders {
		assertion, parseErr = h.samlService.ParseAssertion(provider.Name, samlResponse)
		if parseErr == nil {
			providerName = provider.Name
			break
		}
	}

	if assertion == nil {
		log.Warn().Err(parseErr).Msg("Could not parse SAML assertion with any dashboard provider")
		return c.Redirect().To("/admin/login?error=" + url.QueryEscape("Invalid SAML assertion"))
	}

	// Check if provider allows dashboard login
	provider, _ := h.samlService.GetProvider(providerName)
	if provider == nil || !provider.AllowDashboardLogin {
		log.Warn().Str("provider", providerName).Msg("SAML provider not enabled for dashboard login")
		return c.Redirect().To("/admin/login?error=" + url.QueryEscape("SAML provider not enabled for dashboard login"))
	}

	// Extract user info using the service method
	email, name, err := h.samlService.ExtractUserInfo(providerName, assertion)
	if err != nil {
		// Fallback to manual extraction from attributes map
		email = getFirstAttribute(assertion.Attributes, "email")
		if email == "" {
			email = getFirstAttribute(assertion.Attributes, "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress")
		}
		if email == "" {
			email = assertion.NameID
		}

		name = getFirstAttribute(assertion.Attributes, "displayName")
		if name == "" {
			name = getFirstAttribute(assertion.Attributes, "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/name")
		}
		if name == "" {
			firstName := getFirstAttribute(assertion.Attributes, "firstName")
			lastName := getFirstAttribute(assertion.Attributes, "lastName")
			if firstName != "" || lastName != "" {
				name = strings.TrimSpace(firstName + " " + lastName)
			}
		}
	}

	// Capitalize the first letter of each word in the name
	name = capitalizeWords(name)

	providerUserID := assertion.NameID
	if providerUserID == "" {
		providerUserID = email
	}

	if email == "" {
		return c.Redirect().To("/admin/login?error=" + url.QueryEscape("Email not provided in SAML assertion"))
	}

	// RBAC: Validate group membership if configured
	if len(provider.RequiredGroups) > 0 || len(provider.RequiredGroupsAll) > 0 || len(provider.DeniedGroups) > 0 {
		groups := h.samlService.ExtractGroups(providerName, assertion)
		if err := h.samlService.ValidateGroupMembership(provider, groups); err != nil {
			log.Warn().
				Err(err).
				Str("provider", providerName).
				Str("email", email).
				Strs("groups", groups).
				Msg("Dashboard SSO access denied due to group membership")
			return c.Redirect().To("/admin/login?error=" + url.QueryEscape(err.Error()))
		}
	}

	// Find or create dashboard user
	samlProviderName := "saml:" + providerName
	user, _, err := h.authService.FindOrCreateUserBySSO(ctx, email, name, samlProviderName, providerUserID)
	if err != nil {
		log.Error().
			Err(err).
			Str("email", email).
			Str("provider", samlProviderName).
			Str("provider_user_id", providerUserID).
			Msg("Failed to create or find dashboard user via SAML SSO")
		return c.Redirect().To("/admin/login?error=" + url.QueryEscape("Failed to create or find user"))
	}

	// Login via SSO
	ipAddress := getIPAddress(c)
	userAgent := string(c.Request().Header.UserAgent())
	loginResp, err := h.authService.LoginViaSSO(ctx, user, ipAddress, userAgent)
	if err != nil {
		errMsg := "Login failed"
		if err.Error() == "account is locked" {
			errMsg = "Account is locked"
		} else if err.Error() == "account is inactive" {
			errMsg = "Account is inactive"
		}
		return c.Redirect().To("/admin/login?error=" + url.QueryEscape(errMsg))
	}

	// Create SAML session for SLO support
	samlSession := &auth.SAMLSession{
		ID:           uuid.New().String(),
		UserID:       user.ID.String(),
		ProviderName: providerName,
		NameID:       assertion.NameID,
		NameIDFormat: assertion.NameIDFormat,
		SessionIndex: assertion.SessionIndex,
		Attributes:   convertSAMLAttributesToMap(assertion.Attributes),
		ExpiresAt:    &assertion.NotOnOrAfter,
		CreatedAt:    time.Now(),
	}

	if err := h.samlService.CreateSAMLSession(ctx, samlSession); err != nil {
		log.Warn().Err(err).Str("user_id", user.ID.String()).Msg("Failed to create SAML session for dashboard user")
	}

	// Redirect with tokens
	redirectURL := relayState
	if redirectURL == "" || redirectURL == "/" {
		redirectURL = "/admin"
	}
	return c.Redirect().To(fmt.Sprintf("/admin/login/callback#access_token=%s&refresh_token=%s&redirect_to=%s",
		url.QueryEscape(loginResp.AccessToken),
		url.QueryEscape(loginResp.RefreshToken),
		url.QueryEscape(redirectURL)))
}

// getFirstAttribute returns the first value for a SAML attribute or empty string
func getFirstAttribute(attributes map[string][]string, key string) string {
	if values, ok := attributes[key]; ok && len(values) > 0 {
		return values[0]
	}
	return ""
}

// convertSAMLAttributesToMap converts SAML attributes to a map[string]interface{} for storage
func convertSAMLAttributesToMap(attrs map[string][]string) map[string]interface{} {
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
