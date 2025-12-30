package auth

import (
	"errors"
	"unicode"

	"golang.org/x/crypto/bcrypt"
)

var (
	// ErrWeakPassword is returned when a password doesn't meet minimum requirements
	ErrWeakPassword = errors.New("password does not meet minimum requirements")
	// ErrPasswordTooLong is returned when password exceeds maximum length
	ErrPasswordTooLong = errors.New("password exceeds maximum length")
)

const (
	// MinPasswordLength is the minimum required password length
	MinPasswordLength = 12 // Increased from 8 for better security
	// MaxPasswordLength is the maximum allowed password length (bcrypt limit is 72)
	MaxPasswordLength = 72
	// DefaultBcryptCost is the default cost for bcrypt hashing
	DefaultBcryptCost = 12
)

// PasswordHasher handles password hashing and validation
type PasswordHasher struct {
	cost          int
	minLength     int
	requireUpper  bool
	requireLower  bool
	requireDigit  bool
	requireSymbol bool
}

// PasswordHasherConfig configures password requirements
type PasswordHasherConfig struct {
	Cost          int
	MinLength     int
	RequireUpper  bool
	RequireLower  bool
	RequireDigit  bool
	RequireSymbol bool
}

// NewPasswordHasher creates a new password hasher with default settings
// Default requires uppercase, lowercase, and digit (no symbol for mobile usability)
func NewPasswordHasher() *PasswordHasher {
	return &PasswordHasher{
		cost:          DefaultBcryptCost,
		minLength:     MinPasswordLength,
		requireUpper:  true,  // Require at least one uppercase letter
		requireLower:  true,  // Require at least one lowercase letter
		requireDigit:  true,  // Require at least one digit
		requireSymbol: false, // No symbol required for mobile usability
	}
}

// NewPasswordHasherWithConfig creates a password hasher with custom configuration
func NewPasswordHasherWithConfig(config PasswordHasherConfig) *PasswordHasher {
	cost := config.Cost
	if cost == 0 {
		cost = DefaultBcryptCost
	}

	minLength := config.MinLength
	if minLength == 0 {
		minLength = MinPasswordLength
	}

	return &PasswordHasher{
		cost:          cost,
		minLength:     minLength,
		requireUpper:  config.RequireUpper,
		requireLower:  config.RequireLower,
		requireDigit:  config.RequireDigit,
		requireSymbol: config.RequireSymbol,
	}
}

// HashPassword hashes a password using bcrypt
func (h *PasswordHasher) HashPassword(password string) (string, error) {
	// Validate password first
	if err := h.ValidatePassword(password); err != nil {
		return "", err
	}

	// Hash the password
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), h.cost)
	if err != nil {
		return "", err
	}

	return string(hashedBytes), nil
}

// ComparePassword compares a plain password with a hashed password
func (h *PasswordHasher) ComparePassword(hashedPassword, plainPassword string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(plainPassword))
}

// ValidatePassword validates a password against configured requirements
func (h *PasswordHasher) ValidatePassword(password string) error {
	// Check length
	if len(password) < h.minLength {
		return ErrWeakPassword
	}

	if len(password) > MaxPasswordLength {
		return ErrPasswordTooLong
	}

	// Check character requirements
	var (
		hasUpper  bool
		hasLower  bool
		hasDigit  bool
		hasSymbol bool
	)

	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsDigit(char):
			hasDigit = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSymbol = true
		}
	}

	if h.requireUpper && !hasUpper {
		return ErrWeakPassword
	}

	if h.requireLower && !hasLower {
		return ErrWeakPassword
	}

	if h.requireDigit && !hasDigit {
		return ErrWeakPassword
	}

	if h.requireSymbol && !hasSymbol {
		return ErrWeakPassword
	}

	return nil
}

// NeedsRehash checks if a password hash needs to be regenerated with a new cost
func (h *PasswordHasher) NeedsRehash(hashedPassword string) bool {
	cost, err := bcrypt.Cost([]byte(hashedPassword))
	if err != nil {
		return true
	}

	return cost != h.cost
}
