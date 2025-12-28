package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"

	"github.com/google/uuid"
	"golang.org/x/crypto/hkdf"
)

var (
	// ErrInvalidKey is returned when the encryption key is invalid
	ErrInvalidKey = errors.New("encryption key must be exactly 32 bytes for AES-256")
	// ErrInvalidCiphertext is returned when the ciphertext is malformed
	ErrInvalidCiphertext = errors.New("invalid ciphertext")
	// ErrDecryptionFailed is returned when decryption fails (wrong key or corrupted data)
	ErrDecryptionFailed = errors.New("decryption failed")
)

// Encrypt encrypts plaintext using AES-256-GCM and returns a base64-encoded ciphertext.
// The key must be exactly 32 bytes.
func Encrypt(plaintext, key string) (string, error) {
	if len(key) != 32 {
		return "", ErrInvalidKey
	}

	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// Create a nonce with the standard GCM nonce size (12 bytes)
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Seal encrypts and authenticates the plaintext
	// The nonce is prepended to the ciphertext
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)

	// Return base64-encoded result
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts a base64-encoded ciphertext using AES-256-GCM.
// The key must be exactly 32 bytes.
func Decrypt(ciphertext, key string) (string, error) {
	if len(key) != 32 {
		return "", ErrInvalidKey
	}

	// Decode base64
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64: %w", err)
	}

	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// The nonce is prepended to the ciphertext
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", ErrInvalidCiphertext
	}

	nonce, ciphertextBytes := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", ErrDecryptionFailed
	}

	return string(plaintext), nil
}

// EncryptIfNotEmpty encrypts the value only if it's non-empty.
// Returns empty string for empty input.
func EncryptIfNotEmpty(value, key string) (string, error) {
	if value == "" {
		return "", nil
	}
	return Encrypt(value, key)
}

// DecryptIfNotEmpty decrypts the value only if it's non-empty.
// Returns empty string for empty input.
func DecryptIfNotEmpty(ciphertext, key string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}
	return Decrypt(ciphertext, key)
}

// ValidateKey checks if the key is valid for AES-256 encryption.
func ValidateKey(key string) error {
	if key == "" {
		return errors.New("encryption key is not set")
	}
	if len(key) != 32 {
		return ErrInvalidKey
	}
	return nil
}

// DeriveUserKey derives a user-specific encryption key from the master key using HKDF.
// This ensures that user secrets cannot be decrypted without knowing both the master key AND user ID.
// The derived key is deterministic for the same master key and user ID combination.
func DeriveUserKey(masterKey string, userID uuid.UUID) (string, error) {
	if len(masterKey) != 32 {
		return "", ErrInvalidKey
	}

	// Use HKDF (HMAC-based Key Derivation Function) to derive a user-specific key
	// Salt: user ID as bytes
	// Info: context string to bind the key to this specific purpose
	hkdfReader := hkdf.New(sha256.New, []byte(masterKey), []byte(userID.String()), []byte("fluxbase-user-settings-v1"))

	derivedKey := make([]byte, 32)
	if _, err := io.ReadFull(hkdfReader, derivedKey); err != nil {
		return "", fmt.Errorf("failed to derive user key: %w", err)
	}

	return string(derivedKey), nil
}
