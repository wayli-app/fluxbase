package auth

import (
	"crypto/rand"
	"encoding/base32"
	"encoding/base64"
	"fmt"

	"github.com/pquerna/otp/totp"
	qrcode "github.com/skip2/go-qrcode"
	"golang.org/x/crypto/bcrypt"
)

// GenerateTOTPSecret generates a new TOTP secret, QR code image, and otpauth URI
// Returns: secret (base32), qrCodeDataURI (base64 PNG data URI), otpauthURI, error
func GenerateTOTPSecret(issuer, accountName string) (string, string, string, error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      issuer,
		AccountName: accountName,
	})
	if err != nil {
		return "", "", "", fmt.Errorf("failed to generate TOTP key: %w", err)
	}

	secret := key.Secret()
	otpauthURI := key.URL()

	// Generate QR code as PNG image
	qrCode, err := qrcode.Encode(otpauthURI, qrcode.Medium, 256)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to generate QR code: %w", err)
	}

	// Convert to base64 data URI
	qrCodeDataURI := "data:image/png;base64," + base64.StdEncoding.EncodeToString(qrCode)

	return secret, qrCodeDataURI, otpauthURI, nil
}

// VerifyTOTPCode verifies a TOTP code against a secret
func VerifyTOTPCode(code, secret string) (bool, error) {
	valid := totp.Validate(code, secret)
	return valid, nil
}

// GenerateBackupCodes generates a set of backup codes for 2FA recovery
// Returns both the plain codes (to show to user) and hashed codes (to store)
func GenerateBackupCodes(count int) ([]string, []string, error) {
	plainCodes := make([]string, count)
	hashedCodes := make([]string, count)

	for i := 0; i < count; i++ {
		code, err := generateAppBackupCode()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to generate backup code: %w", err)
		}

		plainCodes[i] = code

		// Hash the backup code using bcrypt
		hash, err := bcrypt.GenerateFromPassword([]byte(code), bcrypt.DefaultCost)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to hash backup code: %w", err)
		}

		hashedCodes[i] = string(hash)
	}

	return plainCodes, hashedCodes, nil
}

// VerifyBackupCode verifies a backup code against its hash
func VerifyBackupCode(code, hashedCode string) (bool, error) {
	err := bcrypt.CompareHashAndPassword([]byte(hashedCode), []byte(code))
	return err == nil, nil
}

// generateAppBackupCode generates a single random 8-character backup code for app users
func generateAppBackupCode() (string, error) {
	bytes := make([]byte, 5)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(bytes), nil
}
