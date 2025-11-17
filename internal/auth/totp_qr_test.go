package auth

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestGenerateTOTPSecret(t *testing.T) {
	issuer := "Fluxbase"
	accountName := "test@example.com"

	secret, qrCodeDataURI, otpauthURI, err := GenerateTOTPSecret(issuer, accountName)
	if err != nil {
		t.Fatalf("Failed to generate TOTP secret: %v", err)
	}

	// Test secret is not empty and is base32
	if secret == "" {
		t.Error("Secret should not be empty")
	}
	if len(secret) < 16 {
		t.Error("Secret should be at least 16 characters")
	}

	// Test otpauth URI format
	if !strings.HasPrefix(otpauthURI, "otpauth://totp/") {
		t.Errorf("otpauth URI should start with 'otpauth://totp/', got: %s", otpauthURI)
	}
	if !strings.Contains(otpauthURI, issuer) {
		t.Errorf("otpauth URI should contain issuer '%s'", issuer)
	}
	if !strings.Contains(otpauthURI, accountName) {
		t.Errorf("otpauth URI should contain account name '%s'", accountName)
	}

	// Test QR code data URI format
	if !strings.HasPrefix(qrCodeDataURI, "data:image/png;base64,") {
		t.Errorf("QR code should be a data URI starting with 'data:image/png;base64,', got: %s", qrCodeDataURI[:50])
	}

	// Test that base64 data is valid
	base64Data := strings.TrimPrefix(qrCodeDataURI, "data:image/png;base64,")
	decoded, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		t.Errorf("Failed to decode base64 QR code data: %v", err)
	}

	// PNG files start with specific magic bytes
	if len(decoded) < 8 {
		t.Error("Decoded QR code data is too small to be a valid PNG")
	}
	pngMagic := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	for i := 0; i < 8; i++ {
		if decoded[i] != pngMagic[i] {
			t.Errorf("QR code data does not have PNG magic bytes at position %d", i)
			break
		}
	}

	t.Logf("✓ Generated TOTP secret successfully")
	t.Logf("  Secret: %s", secret)
	t.Logf("  URI: %s", otpauthURI)
	t.Logf("  QR Code size: %d bytes", len(decoded))
}

func TestGenerateTOTPSecretConsistency(t *testing.T) {
	// Generate two secrets for the same account
	secret1, qr1, uri1, err1 := GenerateTOTPSecret("Fluxbase", "user@test.com")
	secret2, qr2, uri2, err2 := GenerateTOTPSecret("Fluxbase", "user@test.com")

	if err1 != nil || err2 != nil {
		t.Fatalf("Failed to generate TOTP secrets: %v, %v", err1, err2)
	}

	// They should be different (new secret each time)
	if secret1 == secret2 {
		t.Error("Two generated secrets should be different")
	}
	if qr1 == qr2 {
		t.Error("Two generated QR codes should be different")
	}
	if uri1 == uri2 {
		t.Error("Two generated URIs should be different")
	}
}
