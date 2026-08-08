package auth

import (
	"bytes"
	"compress/flate"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"io"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/beevik/etree"
	"github.com/crewjam/saml"
	dsig "github.com/russellhaering/goxmldsig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// This file adds real coverage for the SAML logout code path, which previously
// only had smoke tests for the empty/invalid-input error branches (see
// saml_test.go). The functions under test are pure logic over XML/byte input
// (base64, inflate, XML unmarshal, XML signature verification) and need no
// database.
//
// Conventions here match the rest of internal/auth/*_test.go: testify-based,
// helpers local to this file. (This diverges from internal/crypto's
// stdlib-only style, but matches the surrounding auth package convention.)

const (
	testEntityID     = "https://idp.example.com/saml"
	testProviderName = "test-idp"
)

// =============================================================================
// Test helpers
// =============================================================================

// testSigningMaterial generates a fresh RSA keypair and a self-signed X.509
// certificate, returning everything the logout tests need. Generating per-run
// keeps the tests hermetic (no committed fixtures) and exercises the real
// crypto path each time.
type testSigningMaterial struct {
	key     *rsa.PrivateKey
	cert    *x509.Certificate
	certPEM string
	certDER []byte
	certB64 string // base64 of the DER bytes, no PEM headers
}

func newTestSigningMaterial(t *testing.T) *testSigningMaterial {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "test-idp"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	require.NoError(t, err)
	cert, err := x509.ParseCertificate(der)
	require.NoError(t, err)

	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})

	return &testSigningMaterial{
		key:     key,
		cert:    cert,
		certPEM: string(pemBytes),
		certDER: der,
		certB64: base64.StdEncoding.EncodeToString(der),
	}
}

// testIDPMetadata builds a minimal crewjam EntityDescriptor carrying one
// signing certificate. This mirrors what ParseLogoutRequest / verifyLogoutSignature
// read off provider.metadata.
func testIDPMetadata(tm *testSigningMaterial) *saml.EntityDescriptor {
	return &saml.EntityDescriptor{
		EntityID: testEntityID,
		IDPSSODescriptors: []saml.IDPSSODescriptor{
			{
				SSODescriptor: saml.SSODescriptor{
					RoleDescriptor: saml.RoleDescriptor{
						KeyDescriptors: []saml.KeyDescriptor{
							{
								Use: "signing",
								KeyInfo: saml.KeyInfo{
									X509Data: saml.X509Data{
										X509Certificates: []saml.X509Certificate{{Data: tm.certB64}},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

// newTestSAMLService builds a SAMLService preloaded with a single provider
// whose metadata matches testEntityID, so the issuer-lookup path resolves.
func newTestSAMLService(t *testing.T, tm *testSigningMaterial, requireSig bool) *SAMLService {
	t.Helper()
	svc, err := NewSAMLService(nil, "https://example.com", nil)
	require.NoError(t, err)
	svc.providers[testProviderName] = &SAMLProvider{
		Name:                   testProviderName,
		Enabled:                true,
		EntityID:               testEntityID,
		IdPSloURL:              "https://idp.example.com/slo",
		metadata:               testIDPMetadata(tm),
		RequireLogoutSignature: requireSig,
	}
	return svc
}

// buildLogoutRequestXML constructs a minimal but well-formed SAML LogoutRequest
// as raw XML bytes via the crewjam type's own Element() serializer.
func buildLogoutRequestXML(t *testing.T, issuer, nameID string) []byte {
	t.Helper()
	req := &saml.LogoutRequest{
		ID:           "req-123",
		Version:      "2.0",
		IssueInstant: time.Now().UTC(),
		Destination:  "https://example.com/slo",
		Issuer:       &saml.Issuer{Value: issuer},
		NameID:       &saml.NameID{Value: nameID, Format: "urn:oasis:names:tc:SAML:2.0:nameid-format:transient"},
	}
	doc := etree.NewDocument()
	doc.SetRoot(req.Element())
	xmlBytes, err := doc.WriteToString()
	require.NoError(t, err)
	return []byte(xmlBytes)
}

// buildLogoutResponseXML constructs a minimal well-formed SAML LogoutResponse.
func buildLogoutResponseXML(t *testing.T, issuer string) []byte {
	t.Helper()
	resp := &saml.LogoutResponse{
		ID:           "resp-123",
		InResponseTo: "req-123",
		Version:      "2.0",
		IssueInstant: time.Now().UTC(),
		Destination:  "https://example.com/slo",
		Issuer:       &saml.Issuer{Value: issuer},
		Status: saml.Status{
			StatusCode: saml.StatusCode{Value: saml.StatusSuccess},
		},
	}
	doc := etree.NewDocument()
	doc.SetRoot(resp.Element())
	xmlBytes, err := doc.WriteToString()
	require.NoError(t, err)
	return []byte(xmlBytes)
}

// signLogoutXMLEnveloped wraps an etree document in an enveloped XML signature
// using the provided RSA key, matching how verifyLogoutSignature validates.
func signLogoutXMLEnveloped(t *testing.T, xmlBytes []byte, tm *testSigningMaterial) []byte {
	t.Helper()
	doc := etree.NewDocument()
	require.NoError(t, doc.ReadFromBytes(xmlBytes))
	root := doc.Root()
	require.NotNil(t, root)

	ctx, err := dsig.NewSigningContext(tm.key, [][]byte{tm.certDER})
	require.NoError(t, err)
	signed, err := ctx.SignEnveloped(root)
	require.NoError(t, err)

	doc2 := etree.NewDocument()
	doc2.SetRoot(signed)
	out, err := doc2.WriteToBytes()
	require.NoError(t, err)
	return out
}

// deflateAndBase64 mimics the HTTP-Redirect binding transport: raw XML →
// DEFLATE → base64.
func deflateAndBase64(t *testing.T, xmlBytes []byte) string {
	t.Helper()
	var buf bytes.Buffer
	w, err := flate.NewWriter(&buf, flate.DefaultCompression)
	require.NoError(t, err)
	_, err = w.Write(xmlBytes)
	require.NoError(t, err)
	require.NoError(t, w.Close())
	return base64.StdEncoding.EncodeToString(buf.Bytes())
}

// =============================================================================
// ParseLogoutRequest Tests
// =============================================================================

func TestParseLogoutRequest_Base64Only_Success(t *testing.T) {
	tm := newTestSigningMaterial(t)
	svc := newTestSAMLService(t, tm, false) // sig not required

	xmlBytes := buildLogoutRequestXML(t, testEntityID, "user@example.com")
	encoded := base64.StdEncoding.EncodeToString(xmlBytes)

	parsed, providerName, err := svc.ParseLogoutRequest(encoded, "relay-state", false)
	require.NoError(t, err)
	require.NotNil(t, parsed)

	assert.Equal(t, testProviderName, providerName)
	assert.Equal(t, "req-123", parsed.ID)
	assert.Equal(t, "user@example.com", parsed.NameID)
	assert.Contains(t, parsed.NameIDFormat, "transient")
	assert.Equal(t, testEntityID, parsed.Issuer)
	assert.Equal(t, "relay-state", parsed.RelayState)
}

func TestParseLogoutRequest_Deflated_Success(t *testing.T) {
	tm := newTestSigningMaterial(t)
	svc := newTestSAMLService(t, tm, false)

	xmlBytes := buildLogoutRequestXML(t, testEntityID, "user@example.com")
	encoded := deflateAndBase64(t, xmlBytes)

	parsed, _, err := svc.ParseLogoutRequest(encoded, "", true)
	require.NoError(t, err)
	require.NotNil(t, parsed)
	assert.Equal(t, "user@example.com", parsed.NameID)
}

func TestParseLogoutRequest_InvalidBase64(t *testing.T) {
	tm := newTestSigningMaterial(t)
	svc := newTestSAMLService(t, tm, false)

	_, _, err := svc.ParseLogoutRequest("!!!not-base64!!!", "", false)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrSAMLInvalidLogoutRequest)
}

func TestParseLogoutRequest_MalformedXML(t *testing.T) {
	tm := newTestSigningMaterial(t)
	svc := newTestSAMLService(t, tm, false)

	encoded := base64.StdEncoding.EncodeToString([]byte("<not><valid saml"))
	_, _, err := svc.ParseLogoutRequest(encoded, "", false)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrSAMLInvalidLogoutRequest)
	assert.Contains(t, err.Error(), "XML parse failed")
}

func TestParseLogoutRequest_UnknownIssuer(t *testing.T) {
	tm := newTestSigningMaterial(t)
	svc := newTestSAMLService(t, tm, false)

	// Issuer in the request does not match any provider's metadata EntityID.
	xmlBytes := buildLogoutRequestXML(t, "https://unknown.example.com/saml", "user@example.com")
	encoded := base64.StdEncoding.EncodeToString(xmlBytes)

	_, _, err := svc.ParseLogoutRequest(encoded, "", false)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrSAMLInvalidLogoutRequest)
	assert.Contains(t, err.Error(), "unknown issuer")
}

func TestParseLogoutRequest_RequiresSignature_SucceedsWhenSigned(t *testing.T) {
	tm := newTestSigningMaterial(t)
	svc := newTestSAMLService(t, tm, true) // RequireLogoutSignature = true

	xmlBytes := buildLogoutRequestXML(t, testEntityID, "user@example.com")
	signed := signLogoutXMLEnveloped(t, xmlBytes, tm)
	encoded := base64.StdEncoding.EncodeToString(signed)

	parsed, _, err := svc.ParseLogoutRequest(encoded, "", false)
	require.NoError(t, err)
	require.NotNil(t, parsed)
	assert.Equal(t, "user@example.com", parsed.NameID)
}

func TestParseLogoutRequest_RequiresSignature_FailsWhenUnsigned(t *testing.T) {
	tm := newTestSigningMaterial(t)
	svc := newTestSAMLService(t, tm, true) // RequireLogoutSignature = true

	xmlBytes := buildLogoutRequestXML(t, testEntityID, "user@example.com")
	encoded := base64.StdEncoding.EncodeToString(xmlBytes)

	_, _, err := svc.ParseLogoutRequest(encoded, "", false)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrSAMLInvalidLogoutRequest)
	assert.Contains(t, err.Error(), "signature")
}

// =============================================================================
// ParseLogoutResponse Tests
// =============================================================================

func TestParseLogoutResponse_Base64Only_Success(t *testing.T) {
	tm := newTestSigningMaterial(t)
	svc := newTestSAMLService(t, tm, false)

	xmlBytes := buildLogoutResponseXML(t, testEntityID)
	encoded := base64.StdEncoding.EncodeToString(xmlBytes)

	parsed, providerName, err := svc.ParseLogoutResponse(encoded, false)
	require.NoError(t, err)
	require.NotNil(t, parsed)

	assert.Equal(t, testProviderName, providerName)
	assert.Equal(t, "req-123", parsed.InResponseTo)
	assert.Equal(t, saml.StatusSuccess, parsed.Status)
	assert.Equal(t, testEntityID, parsed.Issuer)
}

func TestParseLogoutResponse_Deflated_Success(t *testing.T) {
	tm := newTestSigningMaterial(t)
	svc := newTestSAMLService(t, tm, false)

	xmlBytes := buildLogoutResponseXML(t, testEntityID)
	encoded := deflateAndBase64(t, xmlBytes)

	parsed, _, err := svc.ParseLogoutResponse(encoded, true)
	require.NoError(t, err)
	require.NotNil(t, parsed)
	assert.Equal(t, saml.StatusSuccess, parsed.Status)
}

func TestParseLogoutResponse_InvalidBase64(t *testing.T) {
	tm := newTestSigningMaterial(t)
	svc := newTestSAMLService(t, tm, false)

	_, _, err := svc.ParseLogoutResponse("!!!not-base64!!!", false)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrSAMLInvalidLogoutResponse)
}

func TestParseLogoutResponse_UnknownIssuer(t *testing.T) {
	tm := newTestSigningMaterial(t)
	svc := newTestSAMLService(t, tm, false)

	xmlBytes := buildLogoutResponseXML(t, "https://unknown.example.com/saml")
	encoded := base64.StdEncoding.EncodeToString(xmlBytes)

	_, _, err := svc.ParseLogoutResponse(encoded, false)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrSAMLInvalidLogoutResponse)
	assert.Contains(t, err.Error(), "unknown issuer")
}

// =============================================================================
// GetIdPSloURL Tests
// =============================================================================

func TestGetIdPSloURL(t *testing.T) {
	tm := newTestSigningMaterial(t)
	svc := newTestSAMLService(t, tm, false)

	t.Run("provider present returns slo url", func(t *testing.T) {
		got, err := svc.GetIdPSloURL(testProviderName)
		require.NoError(t, err)
		assert.Equal(t, "https://idp.example.com/slo", got)
	})

	t.Run("provider absent returns ErrSAMLProviderNotFound", func(t *testing.T) {
		_, err := svc.GetIdPSloURL("nope")
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrSAMLProviderNotFound)
	})
}

// =============================================================================
// HasSigningKey Tests
// =============================================================================

func TestHasSigningKey(t *testing.T) {
	tm := newTestSigningMaterial(t)
	svc := newTestSAMLService(t, tm, false)

	// newTestSAMLService does not set spKey/spCert, so HasSigningKey is false.
	assert.False(t, svc.HasSigningKey(testProviderName))
	assert.False(t, svc.HasSigningKey("nonexistent"))
}

// =============================================================================
// inflateBytes Tests
// =============================================================================

func TestInflateBytes_ValidDeflate(t *testing.T) {
	original := []byte("<hello>world</hello>")

	var compressed bytes.Buffer
	w, err := flate.NewWriter(&compressed, flate.DefaultCompression)
	require.NoError(t, err)
	_, err = w.Write(original)
	require.NoError(t, err)
	require.NoError(t, w.Close())

	got, err := inflateBytes(compressed.Bytes())
	require.NoError(t, err)
	assert.Equal(t, original, got)
}

func TestInflateBytes_NonDeflateData(t *testing.T) {
	// Plain text is not a valid DEFLATE stream; inflate should error.
	got, err := inflateBytes([]byte("not deflated at all"))
	// Per discipline: only assert the error here. If a future change makes
	// inflateBytes tolerate non-deflate input, revisit this expectation.
	require.Error(t, err)
	assert.Empty(t, got)
}

func TestInflateBytes_Garbage(t *testing.T) {
	_, err := inflateBytes([]byte{0xFF, 0xFF, 0xFF, 0xFF})
	require.Error(t, err)
}

// =============================================================================
// verifyLogoutSignature Tests
// =============================================================================

func TestVerifyLogoutSignature_NoSignatureElement(t *testing.T) {
	tm := newTestSigningMaterial(t)
	md := testIDPMetadata(tm)

	xmlBytes := buildLogoutRequestXML(t, testEntityID, "user@example.com")
	err := verifyLogoutSignature(xmlBytes, md)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "signature element not present")
}

func TestVerifyLogoutSignature_EmptyDocument(t *testing.T) {
	tm := newTestSigningMaterial(t)
	md := testIDPMetadata(tm)

	err := verifyLogoutSignature([]byte(""), md)
	require.Error(t, err)
	// Empty input → either parse failure or empty-document error; both are fine.
	assert.True(t,
		strings.Contains(err.Error(), "empty") || strings.Contains(err.Error(), "parse"),
		"unexpected error: %v", err)
}

func TestVerifyLogoutSignature_ValidSignature(t *testing.T) {
	tm := newTestSigningMaterial(t)
	md := testIDPMetadata(tm)

	xmlBytes := buildLogoutRequestXML(t, testEntityID, "user@example.com")
	signed := signLogoutXMLEnveloped(t, xmlBytes, tm)

	err := verifyLogoutSignature(signed, md)
	assert.NoError(t, err)
}

func TestVerifyLogoutSignature_TamperedPayload(t *testing.T) {
	tm := newTestSigningMaterial(t)
	md := testIDPMetadata(tm)

	xmlBytes := buildLogoutRequestXML(t, testEntityID, "user@example.com")
	signed := signLogoutXMLEnveloped(t, xmlBytes, tm)

	// Tamper with the signed payload AFTER signing but BEFORE verification.
	// Replacing the NameID value invalidates the signature's digest.
	tampered := bytes.Replace(signed, []byte("user@example.com"), []byte("attacker@example.com"), 1)
	if bytes.Equal(tampered, signed) {
		t.Fatal("tamper did not modify the bytes; test is ineffective")
	}

	err := verifyLogoutSignature(tampered, md)
	require.Error(t, err)
}

func TestVerifyLogoutSignature_NoCertificatesInMetadata(t *testing.T) {
	tm := newTestSigningMaterial(t)
	xmlBytes := buildLogoutRequestXML(t, testEntityID, "user@example.com")
	signed := signLogoutXMLEnveloped(t, xmlBytes, tm)

	// Metadata with no signing certificates → distinct error path.
	md := &saml.EntityDescriptor{EntityID: testEntityID}
	err := verifyLogoutSignature(signed, md)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no IdP signing certificates")
}

// Ensure we reference io so the import stays valid even if helpers evolve.
var _ = io.EOF

// Guard: keep the errors import meaningful for future error-typing cases.
var _ = errors.Is
