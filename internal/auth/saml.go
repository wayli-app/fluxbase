package auth

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/crewjam/saml"
	"github.com/crewjam/saml/samlsp"
	"github.com/fluxbase-eu/fluxbase/internal/config"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

var (
	ErrSAMLProviderNotFound     = errors.New("SAML provider not found")
	ErrSAMLProviderDisabled     = errors.New("SAML provider is disabled")
	ErrSAMLMetadataFetchFailed  = errors.New("failed to fetch IdP metadata")
	ErrSAMLMetadataParseFailed  = errors.New("failed to parse IdP metadata")
	ErrSAMLMetadataInsecureURL  = errors.New("SAML metadata URL must use HTTPS")
	ErrSAMLAssertionInvalid     = errors.New("invalid SAML assertion")
	ErrSAMLAssertionExpired     = errors.New("SAML assertion has expired")
	ErrSAMLAssertionReplayed    = errors.New("SAML assertion has already been used")
	ErrSAMLAudienceMismatch     = errors.New("SAML assertion audience does not match service provider")
	ErrSAMLMissingEmail         = errors.New("email attribute not found in SAML assertion")
	ErrSAMLUserCreationDisabled = errors.New("automatic user creation is disabled for this provider")
	ErrSAMLInvalidRelayState    = errors.New("invalid RelayState redirect URL")
)

// SAMLProvider represents a configured SAML Identity Provider
type SAMLProvider struct {
	ID               string            `json:"id"`
	Name             string            `json:"name"`
	Enabled          bool              `json:"enabled"`
	EntityID         string            `json:"entity_id"`
	AcsURL           string            `json:"acs_url"`
	SsoURL           string            `json:"sso_url"`           // IdP's SSO endpoint
	SloURL           string            `json:"slo_url,omitempty"` // IdP's SLO endpoint (optional)
	Certificate      string            `json:"certificate"`       // IdP's signing certificate
	AttributeMapping map[string]string `json:"attribute_mapping"`
	AutoCreateUsers  bool              `json:"auto_create_users"`
	DefaultRole      string            `json:"default_role"`
	CreatedAt        time.Time         `json:"created_at"`
	UpdatedAt        time.Time         `json:"updated_at"`

	// Security options
	AllowIDPInitiated        bool     `json:"allow_idp_initiated"`         // Allow IdP-initiated SSO (default: false)
	AllowedRedirectHosts     []string `json:"allowed_redirect_hosts"`      // Whitelist for RelayState redirects
	AllowInsecureMetadataURL bool     `json:"allow_insecure_metadata_url"` // Allow HTTP metadata URLs

	// Login targeting
	AllowDashboardLogin bool `json:"allow_dashboard_login"` // Allow for dashboard admin SSO
	AllowAppLogin       bool `json:"allow_app_login"`       // Allow for app user authentication

	// Cached parsed metadata
	idpDescriptor *saml.IDPSSODescriptor
	metadata      *saml.EntityDescriptor
}

// SAMLSession represents an active SAML authentication session
type SAMLSession struct {
	ID           string                 `json:"id"`
	UserID       string                 `json:"user_id"`
	ProviderID   string                 `json:"provider_id,omitempty"`
	ProviderName string                 `json:"provider_name"`
	NameID       string                 `json:"name_id"`
	NameIDFormat string                 `json:"name_id_format,omitempty"`
	SessionIndex string                 `json:"session_index,omitempty"`
	Attributes   map[string]interface{} `json:"attributes,omitempty"`
	ExpiresAt    *time.Time             `json:"expires_at,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
}

// SAMLAssertion represents parsed SAML assertion data
type SAMLAssertion struct {
	ID           string
	NameID       string
	NameIDFormat string
	SessionIndex string
	Attributes   map[string][]string
	IssueInstant time.Time
	NotBefore    time.Time
	NotOnOrAfter time.Time
}

// SAMLService manages SAML SSO functionality
type SAMLService struct {
	db         *pgxpool.Pool
	baseURL    string
	providers  map[string]*SAMLProvider
	spConfigs  map[string]*saml.ServiceProvider
	httpClient *http.Client
	mu         sync.RWMutex
}

// NewSAMLService creates a new SAML service
func NewSAMLService(db *pgxpool.Pool, baseURL string, configs []config.SAMLProviderConfig) (*SAMLService, error) {
	s := &SAMLService{
		db:        db,
		baseURL:   strings.TrimSuffix(baseURL, "/"),
		providers: make(map[string]*SAMLProvider),
		spConfigs: make(map[string]*saml.ServiceProvider),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	// Initialize providers from config
	for _, cfg := range configs {
		if !cfg.Enabled {
			log.Debug().Str("provider", cfg.Name).Msg("Skipping disabled SAML provider")
			continue
		}

		if err := s.AddProviderFromConfig(cfg); err != nil {
			log.Warn().Err(err).Str("provider", cfg.Name).Msg("Failed to initialize SAML provider")
			continue
		}

		log.Info().Str("provider", cfg.Name).Msg("Initialized SAML provider")
	}

	return s, nil
}

// AddProviderFromConfig adds a SAML provider from configuration
func (s *SAMLService) AddProviderFromConfig(cfg config.SAMLProviderConfig) error {
	provider := &SAMLProvider{
		ID:                       uuid.New().String(),
		Name:                     cfg.Name,
		Enabled:                  cfg.Enabled,
		EntityID:                 cfg.EntityID,
		AcsURL:                   cfg.AcsURL,
		AttributeMapping:         cfg.AttributeMapping,
		AutoCreateUsers:          cfg.AutoCreateUsers,
		DefaultRole:              cfg.DefaultRole,
		AllowIDPInitiated:        cfg.AllowIDPInitiated,
		AllowedRedirectHosts:     cfg.AllowedRedirectHosts,
		AllowInsecureMetadataURL: cfg.AllowInsecureMetadataURL,
		CreatedAt:                time.Now(),
		UpdatedAt:                time.Now(),
	}

	// Set default ACS URL if not specified
	if provider.AcsURL == "" {
		provider.AcsURL = fmt.Sprintf("%s/auth/saml/acs", s.baseURL)
	}

	// Set default entity ID if not specified
	if provider.EntityID == "" {
		provider.EntityID = fmt.Sprintf("%s/auth/saml", s.baseURL)
	}

	// Set default attribute mapping
	if provider.AttributeMapping == nil {
		provider.AttributeMapping = map[string]string{
			"email": "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress",
			"name":  "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/name",
		}
	}

	// Set default role
	if provider.DefaultRole == "" {
		provider.DefaultRole = "authenticated"
	}

	// Fetch and parse IdP metadata
	var metadataXML []byte
	var err error

	if cfg.IdPMetadataURL != "" {
		// Validate HTTPS requirement for metadata URL
		if err := validateMetadataURL(cfg.IdPMetadataURL, cfg.AllowInsecureMetadataURL); err != nil {
			return err
		}
		metadataXML, err = s.fetchMetadata(cfg.IdPMetadataURL)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrSAMLMetadataFetchFailed, err)
		}
	} else if cfg.IdPMetadataXML != "" {
		metadataXML = []byte(cfg.IdPMetadataXML)
	} else {
		return errors.New("either idp_metadata_url or idp_metadata_xml must be provided")
	}

	// Parse metadata
	metadata, err := samlsp.ParseMetadata(metadataXML)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrSAMLMetadataParseFailed, err)
	}

	provider.metadata = metadata

	// Get IdP descriptor - find one with HTTP-POST or HTTP-Redirect binding
	var idpDescriptor *saml.IDPSSODescriptor
	for i := range metadata.IDPSSODescriptors {
		desc := &metadata.IDPSSODescriptors[i]
		// Check if this descriptor supports POST or Redirect binding
		for _, sso := range desc.SingleSignOnServices {
			if sso.Binding == saml.HTTPPostBinding || sso.Binding == saml.HTTPRedirectBinding {
				idpDescriptor = desc
				break
			}
		}
		if idpDescriptor != nil {
			break
		}
	}
	if idpDescriptor == nil {
		return errors.New("no suitable IdP SSO descriptor found in metadata")
	}

	provider.idpDescriptor = idpDescriptor

	// Extract SSO URL
	for _, sso := range idpDescriptor.SingleSignOnServices {
		if sso.Binding == saml.HTTPPostBinding || sso.Binding == saml.HTTPRedirectBinding {
			provider.SsoURL = sso.Location
			break
		}
	}

	// Extract SLO URL if available
	for _, slo := range idpDescriptor.SingleLogoutServices {
		if slo.Binding == saml.HTTPPostBinding || slo.Binding == saml.HTTPRedirectBinding {
			provider.SloURL = slo.Location
			break
		}
	}

	// Extract certificate
	for _, keyDescriptor := range idpDescriptor.KeyDescriptors {
		if keyDescriptor.Use == "signing" || keyDescriptor.Use == "" {
			for _, cert := range keyDescriptor.KeyInfo.X509Data.X509Certificates {
				provider.Certificate = cert.Data
				break
			}
			break
		}
	}

	// Create SAML Service Provider config
	acsURL, _ := url.Parse(provider.AcsURL)
	entityID, _ := url.Parse(provider.EntityID)
	metadataURL, _ := url.Parse(fmt.Sprintf("%s/auth/saml/metadata/%s", s.baseURL, cfg.Name))

	sp := &saml.ServiceProvider{
		EntityID:          entityID.String(),
		AcsURL:            *acsURL,
		MetadataURL:       *metadataURL,
		IDPMetadata:       metadata,
		AllowIDPInitiated: cfg.AllowIDPInitiated, // Use config setting instead of hardcoded true
	}

	s.mu.Lock()
	s.providers[cfg.Name] = provider
	s.spConfigs[cfg.Name] = sp
	s.mu.Unlock()

	return nil
}

// fetchMetadata fetches IdP metadata from a URL
func (s *SAMLService) fetchMetadata(metadataURL string) ([]byte, error) {
	resp, err := s.httpClient.Get(metadataURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("metadata fetch returned status %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

// GetProvider returns a SAML provider by name
func (s *SAMLService) GetProvider(name string) (*SAMLProvider, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	provider, ok := s.providers[name]
	if !ok {
		return nil, ErrSAMLProviderNotFound
	}

	if !provider.Enabled {
		return nil, ErrSAMLProviderDisabled
	}

	return provider, nil
}

// ListProviders returns all enabled SAML providers
func (s *SAMLService) ListProviders() []*SAMLProvider {
	s.mu.RLock()
	defer s.mu.RUnlock()

	providers := make([]*SAMLProvider, 0, len(s.providers))
	for _, p := range s.providers {
		if p.Enabled {
			providers = append(providers, p)
		}
	}

	return providers
}

// GenerateAuthRequest generates a SAML AuthnRequest for the given provider
func (s *SAMLService) GenerateAuthRequest(providerName string, relayState string) (string, string, error) {
	s.mu.RLock()
	sp, ok := s.spConfigs[providerName]
	provider, provOk := s.providers[providerName]
	s.mu.RUnlock()

	if !ok || !provOk {
		return "", "", ErrSAMLProviderNotFound
	}

	if !provider.Enabled {
		return "", "", ErrSAMLProviderDisabled
	}

	// Create AuthnRequest
	req, err := sp.MakeAuthenticationRequest(provider.SsoURL, saml.HTTPRedirectBinding, saml.HTTPPostBinding)
	if err != nil {
		return "", "", fmt.Errorf("failed to create AuthnRequest: %w", err)
	}

	// Build redirect URL
	redirectURL, err := req.Redirect(relayState, sp)
	if err != nil {
		return "", "", fmt.Errorf("failed to build redirect URL: %w", err)
	}

	return redirectURL.String(), req.ID, nil
}

// GetSPMetadata returns the SP metadata XML for a provider
func (s *SAMLService) GetSPMetadata(providerName string) ([]byte, error) {
	s.mu.RLock()
	sp, ok := s.spConfigs[providerName]
	s.mu.RUnlock()

	if !ok {
		return nil, ErrSAMLProviderNotFound
	}

	metadata := sp.Metadata()
	return xml.MarshalIndent(metadata, "", "  ")
}

// ParseAssertion parses and validates a SAML assertion
func (s *SAMLService) ParseAssertion(providerName string, samlResponse string) (*SAMLAssertion, error) {
	s.mu.RLock()
	sp, ok := s.spConfigs[providerName]
	provider, provOk := s.providers[providerName]
	s.mu.RUnlock()

	if !ok || !provOk {
		return nil, ErrSAMLProviderNotFound
	}

	if !provider.Enabled {
		return nil, ErrSAMLProviderDisabled
	}

	// Decode base64 response
	responseXML, err := base64.StdEncoding.DecodeString(samlResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to decode SAML response: %w", err)
	}

	// Parse response - ParseXMLResponse needs: XML bytes, possible request IDs, current URL
	// We use the ACS URL as the current URL and nil for request IDs (for IdP-initiated flows)
	assertion, err := sp.ParseXMLResponse(responseXML, nil, sp.AcsURL)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrSAMLAssertionInvalid, err)
	}

	// Validate time conditions
	now := time.Now()
	if now.Before(assertion.Conditions.NotBefore) {
		return nil, fmt.Errorf("%w: assertion not yet valid", ErrSAMLAssertionInvalid)
	}
	if now.After(assertion.Conditions.NotOnOrAfter) {
		return nil, ErrSAMLAssertionExpired
	}

	// Validate audience restriction
	// The assertion must be intended for our service provider
	if len(assertion.Conditions.AudienceRestrictions) > 0 {
		audienceValid := false
		for _, audienceRestriction := range assertion.Conditions.AudienceRestrictions {
			// Audience must match our Entity ID or Metadata URL
			if audienceRestriction.Audience.Value == sp.EntityID ||
				audienceRestriction.Audience.Value == sp.MetadataURL.String() {
				audienceValid = true
				break
			}
		}
		if !audienceValid {
			log.Warn().
				Str("provider", providerName).
				Str("expected_entity_id", sp.EntityID).
				Msg("SAML assertion audience mismatch")
			return nil, ErrSAMLAudienceMismatch
		}
	}

	// Extract attributes
	attrs := make(map[string][]string)
	for _, attrStatement := range assertion.AttributeStatements {
		for _, attr := range attrStatement.Attributes {
			values := make([]string, len(attr.Values))
			for i, v := range attr.Values {
				values[i] = v.Value
			}
			attrs[attr.Name] = values
			// Also store by FriendlyName if available
			if attr.FriendlyName != "" {
				attrs[attr.FriendlyName] = values
			}
		}
	}

	// Get session index from AuthnStatement
	var sessionIndex string
	for _, authnStatement := range assertion.AuthnStatements {
		if authnStatement.SessionIndex != "" {
			sessionIndex = authnStatement.SessionIndex
			break
		}
	}

	return &SAMLAssertion{
		ID:           assertion.ID,
		NameID:       string(assertion.Subject.NameID.Value),
		NameIDFormat: string(assertion.Subject.NameID.Format),
		SessionIndex: sessionIndex,
		Attributes:   attrs,
		IssueInstant: assertion.IssueInstant,
		NotBefore:    assertion.Conditions.NotBefore,
		NotOnOrAfter: assertion.Conditions.NotOnOrAfter,
	}, nil
}

// ExtractUserInfo extracts user information from SAML assertion using attribute mapping
func (s *SAMLService) ExtractUserInfo(providerName string, assertion *SAMLAssertion) (email, name string, err error) {
	s.mu.RLock()
	provider, ok := s.providers[providerName]
	s.mu.RUnlock()

	if !ok {
		return "", "", ErrSAMLProviderNotFound
	}

	// Try to find email
	emailAttr := provider.AttributeMapping["email"]
	if emailAttr == "" {
		emailAttr = "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress"
	}

	if values, ok := assertion.Attributes[emailAttr]; ok && len(values) > 0 {
		email = values[0]
	} else {
		// Try common email attribute names
		for _, attrName := range []string{"email", "Email", "emailAddress", "mail", "urn:oid:0.9.2342.19200300.100.1.3"} {
			if values, ok := assertion.Attributes[attrName]; ok && len(values) > 0 {
				email = values[0]
				break
			}
		}
	}

	if email == "" {
		// Use NameID as email if it looks like an email
		if strings.Contains(assertion.NameID, "@") {
			email = assertion.NameID
		} else {
			return "", "", ErrSAMLMissingEmail
		}
	}

	// Try to find name
	nameAttr := provider.AttributeMapping["name"]
	if nameAttr == "" {
		nameAttr = "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/name"
	}

	if values, ok := assertion.Attributes[nameAttr]; ok && len(values) > 0 {
		name = values[0]
	} else {
		// Try common name attribute names
		for _, attrName := range []string{"name", "displayName", "cn", "urn:oid:2.5.4.3"} {
			if values, ok := assertion.Attributes[attrName]; ok && len(values) > 0 {
				name = values[0]
				break
			}
		}
	}

	// Try first name + last name if name is empty
	if name == "" {
		var firstName, lastName string
		for _, attrName := range []string{"firstName", "givenName", "urn:oid:2.5.4.42"} {
			if values, ok := assertion.Attributes[attrName]; ok && len(values) > 0 {
				firstName = values[0]
				break
			}
		}
		for _, attrName := range []string{"lastName", "surname", "sn", "urn:oid:2.5.4.4"} {
			if values, ok := assertion.Attributes[attrName]; ok && len(values) > 0 {
				lastName = values[0]
				break
			}
		}
		if firstName != "" || lastName != "" {
			name = strings.TrimSpace(firstName + " " + lastName)
		}
	}

	// Sanitize name to prevent XSS attacks from malicious IdP attributes
	name = SanitizeSAMLAttribute(name)

	return email, name, nil
}

// CheckAssertionReplay checks if an assertion ID has been used before (replay attack prevention)
func (s *SAMLService) CheckAssertionReplay(ctx context.Context, assertionID string, expiresAt time.Time) (bool, error) {
	// Try to insert the assertion ID
	_, err := s.db.Exec(ctx, `
		INSERT INTO auth.saml_assertion_ids (assertion_id, expires_at)
		VALUES ($1, $2)
		ON CONFLICT (assertion_id) DO NOTHING
	`, assertionID, expiresAt)
	if err != nil {
		return false, err
	}

	// Check if it was inserted (new) or already existed (replay)
	var exists bool
	err = s.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM auth.saml_assertion_ids
			WHERE assertion_id = $1 AND created_at < NOW() - INTERVAL '1 second'
		)
	`, assertionID).Scan(&exists)
	if err != nil {
		return false, err
	}

	return exists, nil // true = replay, false = new
}

// CreateSAMLSession creates a new SAML session for tracking
func (s *SAMLService) CreateSAMLSession(ctx context.Context, session *SAMLSession) error {
	_, err := s.db.Exec(ctx, `
		INSERT INTO auth.saml_sessions (id, user_id, provider_id, provider_name, name_id, name_id_format, session_index, attributes, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`,
		session.ID,
		session.UserID,
		session.ProviderID,
		session.ProviderName,
		session.NameID,
		session.NameIDFormat,
		session.SessionIndex,
		session.Attributes,
		session.ExpiresAt,
	)
	return err
}

// DeleteSAMLSession deletes a SAML session (for logout)
func (s *SAMLService) DeleteSAMLSession(ctx context.Context, sessionID string) error {
	_, err := s.db.Exec(ctx, `DELETE FROM auth.saml_sessions WHERE id = $1`, sessionID)
	return err
}

// CleanupExpiredAssertions removes expired assertion IDs from the replay prevention table
func (s *SAMLService) CleanupExpiredAssertions(ctx context.Context) error {
	_, err := s.db.Exec(ctx, `DELETE FROM auth.saml_assertion_ids WHERE expires_at < NOW()`)
	return err
}

// Helper function to parse PEM certificate
func parseCertificate(certPEM string) (*x509.Certificate, error) {
	certData, err := base64.StdEncoding.DecodeString(certPEM)
	if err != nil {
		return nil, err
	}
	return x509.ParseCertificate(certData)
}

// Helper function to parse PEM private key
func parsePrivateKey(keyPEM string) (*rsa.PrivateKey, error) {
	keyData, err := base64.StdEncoding.DecodeString(keyPEM)
	if err != nil {
		return nil, err
	}
	key, err := x509.ParsePKCS8PrivateKey(keyData)
	if err != nil {
		key, err = x509.ParsePKCS1PrivateKey(keyData)
		if err != nil {
			return nil, err
		}
	}
	rsaKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("not an RSA private key")
	}
	return rsaKey, nil
}

// validateMetadataURL validates that a metadata URL uses HTTPS (unless explicitly allowed)
func validateMetadataURL(metadataURL string, allowInsecure bool) error {
	u, err := url.Parse(metadataURL)
	if err != nil {
		return fmt.Errorf("invalid metadata URL: %w", err)
	}

	if u.Scheme != "https" {
		if allowInsecure {
			log.Warn().Str("url", metadataURL).Msg("Using insecure HTTP for SAML metadata URL")
		} else {
			return fmt.Errorf("%w: got %s", ErrSAMLMetadataInsecureURL, u.Scheme)
		}
	}

	return nil
}

// ValidateRelayState validates that a RelayState URL is safe for redirect
// Returns the validated URL or an error if the URL is not allowed
func ValidateRelayState(relayState string, allowedHosts []string) (string, error) {
	if relayState == "" {
		return "", nil
	}

	// Parse the URL
	u, err := url.Parse(relayState)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrSAMLInvalidRelayState, err)
	}

	// Block protocol-relative URLs (//evil.com/path)
	if strings.HasPrefix(relayState, "//") {
		return "", fmt.Errorf("%w: protocol-relative URLs not allowed", ErrSAMLInvalidRelayState)
	}

	// Allow relative URLs (same-origin) - they have no host
	if u.Host == "" {
		return relayState, nil
	}

	// If no allowed hosts configured, reject all absolute URLs
	if len(allowedHosts) == 0 {
		return "", fmt.Errorf("%w: absolute URLs require allowed_redirect_hosts configuration", ErrSAMLInvalidRelayState)
	}

	// Check against allowed hosts
	for _, allowed := range allowedHosts {
		// Exact match or subdomain match
		if u.Host == allowed || strings.HasSuffix(u.Host, "."+allowed) {
			return relayState, nil
		}
	}

	log.Warn().
		Str("relay_state", relayState).
		Str("host", u.Host).
		Strs("allowed_hosts", allowedHosts).
		Msg("RelayState redirect blocked - host not in allowed list")

	return "", fmt.Errorf("%w: host %q not in allowed list", ErrSAMLInvalidRelayState, u.Host)
}

// SanitizeSAMLAttribute cleans a SAML attribute value for safe storage and display
// This prevents XSS attacks from malicious IdP attribute values
func SanitizeSAMLAttribute(value string) string {
	// Remove null bytes
	value = strings.ReplaceAll(value, "\x00", "")

	// Remove other control characters except standard whitespace
	var sanitized strings.Builder
	for _, r := range value {
		// Allow printable characters (32-126) and standard whitespace (tab, newline, carriage return)
		// Exclude DEL character (127/0x7F) and other control characters
		if (r >= 32 && r < 127) || r == '\t' || r == '\n' || r == '\r' || r > 127 {
			sanitized.WriteRune(r)
		}
	}
	value = sanitized.String()

	// Trim and limit length to prevent excessively long values
	value = strings.TrimSpace(value)
	if len(value) > 1024 {
		value = value[:1024]
	}

	return value
}

// RemoveProvider removes a SAML provider by name
func (s *SAMLService) RemoveProvider(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.providers, name)
	delete(s.spConfigs, name)

	log.Info().Str("provider", name).Msg("SAML provider removed")
}

// LoadProvidersFromDB loads SAML providers from the database
func (s *SAMLService) LoadProvidersFromDB(ctx context.Context) error {
	query := `
		SELECT id, name, enabled, entity_id, acs_url,
		       idp_metadata_url, idp_metadata_xml, idp_metadata_cached,
		       attribute_mapping, auto_create_users, default_role,
		       COALESCE(allow_dashboard_login, false), COALESCE(allow_app_login, true),
		       COALESCE(allow_idp_initiated, false), COALESCE(allowed_redirect_hosts, ARRAY[]::TEXT[]),
		       created_at, updated_at
		FROM auth.saml_providers
		WHERE enabled = true AND COALESCE(source, 'database') = 'database'
	`

	rows, err := s.db.Query(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to query SAML providers: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			id                   string
			name                 string
			enabled              bool
			entityID             string
			acsURL               string
			metadataURL          *string
			metadataXML          *string
			metadataCached       *string
			attrMapping          map[string]string
			autoCreateUsers      bool
			defaultRole          string
			allowDashboardLogin  bool
			allowAppLogin        bool
			allowIDPInitiated    bool
			allowedRedirectHosts []string
			createdAt            time.Time
			updatedAt            time.Time
		)

		err := rows.Scan(
			&id, &name, &enabled, &entityID, &acsURL,
			&metadataURL, &metadataXML, &metadataCached,
			&attrMapping, &autoCreateUsers, &defaultRole,
			&allowDashboardLogin, &allowAppLogin,
			&allowIDPInitiated, &allowedRedirectHosts,
			&createdAt, &updatedAt,
		)
		if err != nil {
			log.Error().Err(err).Msg("Failed to scan SAML provider from database")
			continue
		}

		// Skip if already loaded from config
		s.mu.RLock()
		_, exists := s.providers[name]
		s.mu.RUnlock()
		if exists {
			log.Debug().Str("provider", name).Msg("Skipping DB provider - already loaded from config")
			continue
		}

		// Determine which metadata to use
		var metadataToUse string
		if metadataCached != nil && *metadataCached != "" {
			metadataToUse = *metadataCached
		} else if metadataXML != nil && *metadataXML != "" {
			metadataToUse = *metadataXML
		} else if metadataURL != nil && *metadataURL != "" {
			// Need to fetch metadata
			xmlData, err := s.fetchMetadata(*metadataURL)
			if err != nil {
				log.Warn().Err(err).Str("provider", name).Msg("Failed to fetch SAML metadata from URL")
				continue
			}
			metadataToUse = string(xmlData)
		} else {
			log.Warn().Str("provider", name).Msg("No SAML metadata available")
			continue
		}

		// Parse metadata
		metadata, err := samlsp.ParseMetadata([]byte(metadataToUse))
		if err != nil {
			log.Warn().Err(err).Str("provider", name).Msg("Failed to parse SAML metadata")
			continue
		}

		// Find IdP descriptor
		var idpDescriptor *saml.IDPSSODescriptor
		for i := range metadata.IDPSSODescriptors {
			desc := &metadata.IDPSSODescriptors[i]
			for _, sso := range desc.SingleSignOnServices {
				if sso.Binding == saml.HTTPPostBinding || sso.Binding == saml.HTTPRedirectBinding {
					idpDescriptor = desc
					break
				}
			}
			if idpDescriptor != nil {
				break
			}
		}
		if idpDescriptor == nil {
			log.Warn().Str("provider", name).Msg("No suitable IdP SSO descriptor found")
			continue
		}

		// Extract SSO URL
		var ssoURL string
		for _, sso := range idpDescriptor.SingleSignOnServices {
			if sso.Binding == saml.HTTPPostBinding || sso.Binding == saml.HTTPRedirectBinding {
				ssoURL = sso.Location
				break
			}
		}

		// Extract SLO URL
		var sloURL string
		for _, slo := range idpDescriptor.SingleLogoutServices {
			if slo.Binding == saml.HTTPPostBinding || slo.Binding == saml.HTTPRedirectBinding {
				sloURL = slo.Location
				break
			}
		}

		// Extract certificate
		var certificate string
		for _, kd := range idpDescriptor.KeyDescriptors {
			if kd.Use == "signing" || kd.Use == "" {
				for _, cert := range kd.KeyInfo.X509Data.X509Certificates {
					certificate = cert.Data
					break
				}
				break
			}
		}

		provider := &SAMLProvider{
			ID:                   id,
			Name:                 name,
			Enabled:              enabled,
			EntityID:             entityID,
			AcsURL:               acsURL,
			SsoURL:               ssoURL,
			SloURL:               sloURL,
			Certificate:          certificate,
			AttributeMapping:     attrMapping,
			AutoCreateUsers:      autoCreateUsers,
			DefaultRole:          defaultRole,
			AllowIDPInitiated:    allowIDPInitiated,
			AllowedRedirectHosts: allowedRedirectHosts,
			CreatedAt:            createdAt,
			UpdatedAt:            updatedAt,
			idpDescriptor:        idpDescriptor,
			metadata:             metadata,
			AllowDashboardLogin:  allowDashboardLogin,
			AllowAppLogin:        allowAppLogin,
		}

		// Create SAML Service Provider config
		acsURLParsed, _ := url.Parse(acsURL)
		entityIDParsed, _ := url.Parse(entityID)
		metadataURLParsed, _ := url.Parse(fmt.Sprintf("%s/auth/saml/metadata/%s", s.baseURL, name))

		sp := &saml.ServiceProvider{
			EntityID:          entityIDParsed.String(),
			AcsURL:            *acsURLParsed,
			MetadataURL:       *metadataURLParsed,
			IDPMetadata:       metadata,
			AllowIDPInitiated: allowIDPInitiated,
		}

		s.mu.Lock()
		s.providers[name] = provider
		s.spConfigs[name] = sp
		s.mu.Unlock()

		log.Info().Str("provider", name).Msg("Loaded SAML provider from database")
	}

	return nil
}

// GetProvidersForDashboard returns providers that allow dashboard login
func (s *SAMLService) GetProvidersForDashboard() []*SAMLProvider {
	s.mu.RLock()
	defer s.mu.RUnlock()

	providers := make([]*SAMLProvider, 0)
	for _, p := range s.providers {
		if p.Enabled && p.AllowDashboardLogin {
			providers = append(providers, p)
		}
	}

	return providers
}

// GetProvidersForApp returns providers that allow app login
func (s *SAMLService) GetProvidersForApp() []*SAMLProvider {
	s.mu.RLock()
	defer s.mu.RUnlock()

	providers := make([]*SAMLProvider, 0)
	for _, p := range s.providers {
		if p.Enabled && p.AllowAppLogin {
			providers = append(providers, p)
		}
	}

	return providers
}
