package auth

import (
	"errors"
	"fmt"
	"net/mail"
	"regexp"
	"strings"
	"unicode/utf8"
)

// Validation constants for field limits
const (
	// Email validation
	MinEmailLength = 3   // Minimum is "a@b"
	MaxEmailLength = 254 // RFC 5321 limit

	// Name/display name validation
	MinNameLength = 1
	MaxNameLength = 100

	// Avatar URL validation
	MaxAvatarURLLength = 2048

	// General metadata limits
	MaxMetadataKeyLength   = 64
	MaxMetadataValueLength = 1024
)

var (
	// ErrInvalidEmail is returned when email format is invalid
	ErrInvalidEmail = errors.New("invalid email address")
	// ErrEmailTooLong is returned when email exceeds max length
	ErrEmailTooLong = errors.New("email address is too long (max 254 characters)")
	// ErrEmailTooShort is returned when email is too short
	ErrEmailTooShort = errors.New("email address is too short")
	// ErrNameTooLong is returned when name exceeds max length
	ErrNameTooLong = errors.New("name is too long (max 100 characters)")
	// ErrNameTooShort is returned when name is empty
	ErrNameTooShort = errors.New("name is required")
	// ErrAvatarURLTooLong is returned when avatar URL exceeds max length
	ErrAvatarURLTooLong = errors.New("avatar URL is too long (max 2048 characters)")
	// ErrInvalidAvatarURL is returned when avatar URL format is invalid
	ErrInvalidAvatarURL = errors.New("invalid avatar URL format")
)

// emailRegex provides a basic email format validation
// This is intentionally permissive - actual validation is done by mail.ParseAddress
var emailRegex = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)

// urlRegex validates URL format for avatar URLs
var urlRegex = regexp.MustCompile(`^https?://[^\s]+$`)

// ValidateEmail validates an email address for format and length
func ValidateEmail(email string) error {
	// Trim whitespace
	email = strings.TrimSpace(email)

	// Check length first
	if utf8.RuneCountInString(email) < MinEmailLength {
		return ErrEmailTooShort
	}
	if utf8.RuneCountInString(email) > MaxEmailLength {
		return ErrEmailTooLong
	}

	// Basic format check
	if !emailRegex.MatchString(email) {
		return ErrInvalidEmail
	}

	// Use Go's mail package for proper RFC 5322 validation
	_, err := mail.ParseAddress(email)
	if err != nil {
		return ErrInvalidEmail
	}

	// Additional checks for dangerous characters
	if strings.ContainsAny(email, "<>\"'\\") {
		return ErrInvalidEmail
	}

	return nil
}

// ValidateName validates a name field (full name, display name, etc.)
func ValidateName(name string) error {
	// Trim whitespace
	name = strings.TrimSpace(name)

	// Check minimum length
	if utf8.RuneCountInString(name) < MinNameLength {
		return ErrNameTooShort
	}

	// Check maximum length
	if utf8.RuneCountInString(name) > MaxNameLength {
		return ErrNameTooLong
	}

	return nil
}

// ValidateNameOptional validates a name field that can be empty
func ValidateNameOptional(name string) error {
	// Trim whitespace
	name = strings.TrimSpace(name)

	// Empty is allowed for optional fields
	if name == "" {
		return nil
	}

	// Check maximum length
	if utf8.RuneCountInString(name) > MaxNameLength {
		return ErrNameTooLong
	}

	return nil
}

// ValidateAvatarURL validates an avatar URL
func ValidateAvatarURL(url string) error {
	// Empty is allowed
	if url == "" {
		return nil
	}

	// Trim whitespace
	url = strings.TrimSpace(url)

	// Check maximum length
	if utf8.RuneCountInString(url) > MaxAvatarURLLength {
		return ErrAvatarURLTooLong
	}

	// Validate URL format (must be http or https)
	if !urlRegex.MatchString(url) {
		return ErrInvalidAvatarURL
	}

	// Check for javascript: or data: URLs which could be XSS vectors
	lowerURL := strings.ToLower(url)
	if strings.HasPrefix(lowerURL, "javascript:") || strings.HasPrefix(lowerURL, "data:") {
		return ErrInvalidAvatarURL
	}

	return nil
}

// ValidationError wraps multiple validation errors
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// NewValidationError creates a new validation error
func NewValidationError(field string, err error) *ValidationError {
	return &ValidationError{
		Field:   field,
		Message: err.Error(),
	}
}

// SignUpValidator validates signup requests
type SignUpValidator struct{}

// Validate validates a signup request
func (v *SignUpValidator) Validate(email, password string) error {
	// Validate email
	if err := ValidateEmail(email); err != nil {
		return NewValidationError("email", err)
	}

	// Password validation is handled by PasswordHasher.ValidatePassword
	// This is just a sanity check
	if password == "" {
		return NewValidationError("password", errors.New("password is required"))
	}

	return nil
}

// DashboardUserValidator validates dashboard user operations
type DashboardUserValidator struct{}

// ValidateCreate validates dashboard user creation
func (v *DashboardUserValidator) ValidateCreate(email, password, fullName string) error {
	// Validate email
	if err := ValidateEmail(email); err != nil {
		return NewValidationError("email", err)
	}

	// Validate password presence (length validation done by bcrypt/hasher)
	if password == "" {
		return NewValidationError("password", errors.New("password is required"))
	}

	// Validate full name
	if err := ValidateName(fullName); err != nil {
		return NewValidationError("full_name", err)
	}

	return nil
}

// ValidateUpdateProfile validates profile update
func (v *DashboardUserValidator) ValidateUpdateProfile(fullName string, avatarURL *string) error {
	// Validate full name
	if err := ValidateName(fullName); err != nil {
		return NewValidationError("full_name", err)
	}

	// Validate avatar URL if provided
	if avatarURL != nil {
		if err := ValidateAvatarURL(*avatarURL); err != nil {
			return NewValidationError("avatar_url", err)
		}
	}

	return nil
}

// UserUpdateValidator validates user update operations
type UserUpdateValidator struct{}

// ValidateUpdate validates user update request
func (v *UserUpdateValidator) ValidateUpdate(req UpdateUserRequest) error {
	// Validate email if provided
	if req.Email != nil {
		if err := ValidateEmail(*req.Email); err != nil {
			return NewValidationError("email", err)
		}
	}

	return nil
}
