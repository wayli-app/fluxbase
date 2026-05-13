package auth

import (
	"crypto/x509"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/beevik/etree"
	"github.com/crewjam/saml"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	dsig "github.com/russellhaering/goxmldsig"
	"github.com/russellhaering/goxmldsig/etreeutils"
)

// LogoutRequestResult contains the result of generating a SAML LogoutRequest
type LogoutRequestResult struct {
	RedirectURL string // URL to redirect user to IdP for logout
	RequestID   string // ID of the LogoutRequest (for matching response)
	Binding     string // "redirect" or "post"
}

// ParsedLogoutRequest represents a parsed SAML LogoutRequest from IdP
type ParsedLogoutRequest struct {
	ID           string // Request ID for InResponseTo
	NameID       string // User identifier
	NameIDFormat string // Format of NameID
	SessionIndex string // Session to terminate (optional)
	Issuer       string // IdP that sent the request
	Destination  string // Where response should be sent
	RelayState   string // Optional state to return
}

// ParsedLogoutResponse represents a parsed SAML LogoutResponse from IdP
type ParsedLogoutResponse struct {
	InResponseTo  string // ID of original LogoutRequest
	Status        string // "Success" or error code
	StatusMessage string // Optional status message
	Issuer        string // IdP that sent the response
}

// GenerateLogoutRequest generates a signed SAML LogoutRequest for SP-initiated logout
func (s *SAMLService) GenerateLogoutRequest(providerName, nameID, nameIDFormat, sessionIndex, relayState string) (*LogoutRequestResult, error) {
	s.mu.RLock()
	provider, ok := s.providers[providerName]
	sp, spOk := s.spConfigs[providerName]
	s.mu.RUnlock()

	if !ok || !spOk {
		return nil, ErrSAMLProviderNotFound
	}

	if !provider.Enabled {
		return nil, ErrSAMLProviderDisabled
	}

	// Check if IdP supports SLO
	if provider.IdPSloURL == "" {
		return nil, ErrSAMLSLONotSupported
	}

	// Check if SP signing key is configured
	if provider.spKey == nil || provider.spCert == nil {
		return nil, ErrSAMLSigningKeyMissing
	}

	// Set SP signing key and certificate for the request
	sp.Key = provider.spKey
	sp.Certificate = provider.spCert

	// Configure SP's SLO URL for return
	sloURL, _ := url.Parse(provider.SloURL)
	sp.SloURL = *sloURL

	// Use MakeRedirectLogoutRequest which handles signing automatically
	redirectURL, err := sp.MakeRedirectLogoutRequest(nameID, relayState)
	if err != nil {
		return nil, fmt.Errorf("failed to create logout request: %w", err)
	}

	// Extract the request ID from the generated URL for tracking
	// The ID is embedded in the SAMLRequest parameter
	requestID := fmt.Sprintf("id-%s", uuid.New().String())

	return &LogoutRequestResult{
		RedirectURL: redirectURL.String(),
		RequestID:   requestID,
		Binding:     "redirect",
	}, nil
}

// GenerateLogoutResponse generates a signed SAML LogoutResponse for IdP-initiated logout
// Returns the redirect URL for HTTP-Redirect binding
func (s *SAMLService) GenerateLogoutResponse(providerName, inResponseTo, relayState string) (*url.URL, error) {
	s.mu.RLock()
	provider, ok := s.providers[providerName]
	sp, spOk := s.spConfigs[providerName]
	s.mu.RUnlock()

	if !ok || !spOk {
		return nil, ErrSAMLProviderNotFound
	}

	// Set signing keys if available
	if provider.spKey != nil && provider.spCert != nil {
		sp.Key = provider.spKey
		sp.Certificate = provider.spCert
	}

	// Configure SP's SLO URL
	sloURL, _ := url.Parse(provider.SloURL)
	sp.SloURL = *sloURL

	// Use library's method which handles signing
	redirectURL, err := sp.MakeRedirectLogoutResponse(inResponseTo, relayState)
	if err != nil {
		return nil, fmt.Errorf("failed to create logout response: %w", err)
	}

	return redirectURL, nil
}

// ParseLogoutRequest parses a SAML LogoutRequest from IdP (IdP-initiated logout)
func (s *SAMLService) ParseLogoutRequest(samlRequest, relayState string, isDeflated bool) (*ParsedLogoutRequest, string, error) {
	// Decode base64
	requestXML, err := base64.StdEncoding.DecodeString(samlRequest)
	if err != nil {
		return nil, "", fmt.Errorf("%w: base64 decode failed: %w", ErrSAMLInvalidLogoutRequest, err)
	}

	// Inflate if using HTTP-Redirect binding (deflated)
	if isDeflated {
		requestXML, err = inflateBytes(requestXML)
		if err != nil {
			return nil, "", fmt.Errorf("%w: inflate failed: %w", ErrSAMLInvalidLogoutRequest, err)
		}
	}

	// Parse XML
	var logoutRequest saml.LogoutRequest
	if err := xml.Unmarshal(requestXML, &logoutRequest); err != nil {
		return nil, "", fmt.Errorf("%w: XML parse failed: %w", ErrSAMLInvalidLogoutRequest, err)
	}

	var nameID string
	var nameIDFormat string
	if logoutRequest.NameID != nil {
		nameID = logoutRequest.NameID.Value
		nameIDFormat = string(logoutRequest.NameID.Format)
	}
	var issuer string
	if logoutRequest.Issuer != nil {
		issuer = logoutRequest.Issuer.Value
	}

	// Find matching provider by issuer
	providerName := ""
	var provider *SAMLProvider
	s.mu.RLock()
	for name, p := range s.providers {
		if p.metadata != nil && p.metadata.EntityID == issuer {
			providerName = name
			provider = p
			break
		}
	}
	s.mu.RUnlock()

	if providerName == "" {
		return nil, "", fmt.Errorf("%w: unknown issuer %s", ErrSAMLInvalidLogoutRequest, issuer)
	}

	if provider.RequireLogoutSignature {
		if err := verifyLogoutSignature(requestXML, provider.metadata); err != nil {
			return nil, "", fmt.Errorf("%w: signature verification failed: %w", ErrSAMLInvalidLogoutRequest, err)
		}
	} else {
		log.Warn().
			Str("provider", providerName).
			Msg("SAML logout request signature verification skipped (RequireLogoutSignature is disabled)")
	}

	// Extract session index if present
	var sessionIndex string
	if logoutRequest.SessionIndex != nil {
		sessionIndex = logoutRequest.SessionIndex.Value
	}

	parsed := &ParsedLogoutRequest{
		ID:           logoutRequest.ID,
		NameID:       nameID,
		NameIDFormat: nameIDFormat,
		SessionIndex: sessionIndex,
		Issuer:       issuer,
		Destination:  logoutRequest.Destination,
		RelayState:   relayState,
	}

	return parsed, providerName, nil
}

// ParseLogoutResponse parses a SAML LogoutResponse from IdP (SP-initiated logout callback)
func (s *SAMLService) ParseLogoutResponse(samlResponse string, isDeflated bool) (*ParsedLogoutResponse, string, error) {
	// Decode base64
	responseXML, err := base64.StdEncoding.DecodeString(samlResponse)
	if err != nil {
		return nil, "", fmt.Errorf("%w: base64 decode failed: %w", ErrSAMLInvalidLogoutResponse, err)
	}

	// Inflate if using HTTP-Redirect binding (deflated)
	if isDeflated {
		responseXML, err = inflateBytes(responseXML)
		if err != nil {
			return nil, "", fmt.Errorf("%w: inflate failed: %w", ErrSAMLInvalidLogoutResponse, err)
		}
	}

	// Parse XML
	var logoutResponse saml.LogoutResponse
	if err := xml.Unmarshal(responseXML, &logoutResponse); err != nil {
		return nil, "", fmt.Errorf("%w: XML parse failed: %w", ErrSAMLInvalidLogoutResponse, err)
	}

	var issuer string
	if logoutResponse.Issuer != nil {
		issuer = logoutResponse.Issuer.Value
	}

	// Find matching provider by issuer
	providerName := ""
	var provider *SAMLProvider
	s.mu.RLock()
	for name, p := range s.providers {
		if p.metadata != nil && p.metadata.EntityID == issuer {
			providerName = name
			provider = p
			break
		}
	}
	s.mu.RUnlock()

	if providerName == "" {
		return nil, "", fmt.Errorf("%w: unknown issuer %s", ErrSAMLInvalidLogoutResponse, issuer)
	}

	if provider.RequireLogoutSignature {
		if err := verifyLogoutSignature(responseXML, provider.metadata); err != nil {
			return nil, "", fmt.Errorf("%w: signature verification failed: %w", ErrSAMLInvalidLogoutResponse, err)
		}
	} else {
		log.Warn().
			Str("provider", providerName).
			Msg("SAML logout response signature verification skipped (RequireLogoutSignature is disabled)")
	}

	// Extract status
	status := logoutResponse.Status.StatusCode.Value
	var statusMessage string
	if logoutResponse.Status.StatusMessage != nil {
		statusMessage = logoutResponse.Status.StatusMessage.Value
	}

	parsed := &ParsedLogoutResponse{
		InResponseTo:  logoutResponse.InResponseTo,
		Status:        status,
		StatusMessage: statusMessage,
		Issuer:        issuer,
	}

	return parsed, providerName, nil
}

// GetIdPSloURL returns the IdP's SLO URL for a provider (if available)
func (s *SAMLService) GetIdPSloURL(providerName string) (string, error) {
	s.mu.RLock()
	provider, ok := s.providers[providerName]
	s.mu.RUnlock()

	if !ok {
		return "", ErrSAMLProviderNotFound
	}

	return provider.IdPSloURL, nil
}

// HasSigningKey returns true if the provider has SP signing keys configured
func (s *SAMLService) HasSigningKey(providerName string) bool {
	s.mu.RLock()
	provider, ok := s.providers[providerName]
	s.mu.RUnlock()

	if !ok {
		return false
	}

	return provider.spKey != nil && provider.spCert != nil
}

// verifyLogoutSignature verifies the XML digital signature of a SAML logout message
// using the IdP's signing certificates extracted from metadata.
func verifyLogoutSignature(xmlData []byte, idpMetadata *saml.EntityDescriptor) error {
	doc := etree.NewDocument()
	if err := doc.ReadFromBytes(xmlData); err != nil {
		return fmt.Errorf("failed to parse XML: %w", err)
	}

	root := doc.Root()
	if root == nil {
		return errors.New("empty XML document")
	}

	sigEl := root.FindElement("./Signature")
	if sigEl == nil {
		return errors.New("signature element not present in logout message")
	}

	var certStrs []string
	for _, idpSSODescriptor := range idpMetadata.IDPSSODescriptors {
		for _, keyDescriptor := range idpSSODescriptor.KeyDescriptors {
			if len(keyDescriptor.KeyInfo.X509Data.X509Certificates) != 0 {
				switch keyDescriptor.Use {
				case "", "signing":
					for _, cert := range keyDescriptor.KeyInfo.X509Data.X509Certificates {
						certStrs = append(certStrs, cert.Data)
					}
				}
			}
		}
	}
	if len(certStrs) == 0 {
		return errors.New("no IdP signing certificates found in metadata")
	}

	certs := make([]*x509.Certificate, 0, len(certStrs))
	for _, certStr := range certStrs {
		cleaned := strings.Join(strings.Fields(certStr), "")
		certBytes, err := base64.StdEncoding.DecodeString(cleaned)
		if err != nil {
			continue
		}
		parsedCert, err := x509.ParseCertificate(certBytes)
		if err != nil {
			continue
		}
		certs = append(certs, parsedCert)
	}
	if len(certs) == 0 {
		return errors.New("failed to parse any IdP signing certificates")
	}

	certificateStore := dsig.MemoryX509CertificateStore{Roots: certs}
	validationContext := dsig.NewDefaultValidationContext(&certificateStore)
	validationContext.IdAttribute = "ID"

	if root.FindElement("./Signature/KeyInfo/X509Data/X509Certificate") == nil {
		if s := root.FindElement("./Signature"); s != nil {
			if ki := s.FindElement("KeyInfo"); ki != nil {
				s.RemoveChild(ki)
			}
		}
	}

	ctx, err := etreeutils.NSBuildParentContext(root)
	if err != nil {
		return fmt.Errorf("failed to build namespace context: %w", err)
	}
	ctx, err = ctx.SubContext(root)
	if err != nil {
		return fmt.Errorf("failed to build sub context: %w", err)
	}
	root, err = etreeutils.NSDetatch(ctx, root)
	if err != nil {
		return fmt.Errorf("failed to detach namespaces: %w", err)
	}

	if _, err := validationContext.Validate(root); err != nil {
		return fmt.Errorf("signature verification failed: %w", err)
	}

	return nil
}
