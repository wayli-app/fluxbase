package auth

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		name    string
		email   string
		wantErr error
	}{
		// Valid emails
		{
			name:    "valid simple email",
			email:   "user@example.com",
			wantErr: nil,
		},
		{
			name:    "valid email with subdomain",
			email:   "user@mail.example.com",
			wantErr: nil,
		},
		{
			name:    "valid email with plus sign",
			email:   "user+tag@example.com",
			wantErr: nil,
		},
		{
			name:    "valid email with dots in local part",
			email:   "first.last@example.com",
			wantErr: nil,
		},
		{
			name:    "valid email with numbers",
			email:   "user123@example123.com",
			wantErr: nil,
		},
		{
			name:    "valid email with hyphen in domain",
			email:   "user@my-example.com",
			wantErr: nil,
		},
		{
			name:    "valid email with whitespace trimmed",
			email:   "  user@example.com  ",
			wantErr: nil,
		},

		// Too short
		{
			name:    "email too short - single char",
			email:   "a",
			wantErr: ErrEmailTooShort,
		},
		{
			name:    "email too short - two chars",
			email:   "ab",
			wantErr: ErrEmailTooShort,
		},
		{
			name:    "empty email",
			email:   "",
			wantErr: ErrEmailTooShort,
		},
		{
			name:    "whitespace only",
			email:   "   ",
			wantErr: ErrEmailTooShort,
		},

		// Too long
		{
			name:    "email too long",
			email:   strings.Repeat("a", 250) + "@b.co",
			wantErr: ErrEmailTooLong,
		},

		// Invalid format
		{
			name:    "missing @",
			email:   "userexample.com",
			wantErr: ErrInvalidEmail,
		},
		{
			name:    "missing domain",
			email:   "user@",
			wantErr: ErrInvalidEmail,
		},
		{
			name:    "missing local part",
			email:   "@example.com",
			wantErr: ErrInvalidEmail,
		},
		{
			name:    "multiple @",
			email:   "user@@example.com",
			wantErr: ErrInvalidEmail,
		},
		{
			name:    "missing TLD",
			email:   "user@example",
			wantErr: ErrInvalidEmail,
		},
		{
			name:    "spaces in email",
			email:   "user name@example.com",
			wantErr: ErrInvalidEmail,
		},

		// Dangerous characters
		{
			name:    "contains angle brackets",
			email:   "user<script>@example.com",
			wantErr: ErrInvalidEmail,
		},
		{
			name:    "contains double quotes",
			email:   `user"test@example.com`,
			wantErr: ErrInvalidEmail,
		},
		{
			name:    "contains single quote",
			email:   "user'test@example.com",
			wantErr: ErrInvalidEmail,
		},
		{
			name:    "contains backslash",
			email:   `user\test@example.com`,
			wantErr: ErrInvalidEmail,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEmail(tt.email)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateEmail_BoundaryLength(t *testing.T) {
	t.Run("exactly minimum length", func(t *testing.T) {
		// Minimum is 3 characters: "a@b"
		err := ValidateEmail("a@b")
		// This might fail format validation but shouldn't fail length
		assert.NotErrorIs(t, err, ErrEmailTooShort)
	})

	t.Run("exactly maximum length", func(t *testing.T) {
		// 254 characters
		localPart := strings.Repeat("a", 243)
		email := localPart + "@example.com" // 243 + 12 = 255, too long
		err := ValidateEmail(email)
		assert.ErrorIs(t, err, ErrEmailTooLong)

		// Exactly 254
		localPart2 := strings.Repeat("a", 242)
		email2 := localPart2 + "@example.com" // 242 + 12 = 254, exactly max
		err = ValidateEmail(email2)
		// May fail for other reasons, but not length
		assert.NotErrorIs(t, err, ErrEmailTooLong)
	})
}

func TestValidateName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr error
	}{
		// Valid names
		{
			name:    "simple name",
			input:   "John",
			wantErr: nil,
		},
		{
			name:    "full name",
			input:   "John Doe",
			wantErr: nil,
		},
		{
			name:    "name with unicode",
			input:   "José García",
			wantErr: nil,
		},
		{
			name:    "single character",
			input:   "J",
			wantErr: nil,
		},
		{
			name:    "name with numbers",
			input:   "John Doe III",
			wantErr: nil,
		},
		{
			name:    "name with hyphen",
			input:   "Mary-Jane Watson",
			wantErr: nil,
		},
		{
			name:    "exactly 100 characters",
			input:   strings.Repeat("a", 100),
			wantErr: nil,
		},

		// Too short
		{
			name:    "empty name",
			input:   "",
			wantErr: ErrNameTooShort,
		},
		{
			name:    "whitespace only",
			input:   "   ",
			wantErr: ErrNameTooShort,
		},

		// Too long
		{
			name:    "name too long",
			input:   strings.Repeat("a", 101),
			wantErr: ErrNameTooLong,
		},
		{
			name:    "unicode name too long",
			input:   strings.Repeat("日", 101), // 101 runes, not bytes
			wantErr: ErrNameTooLong,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateName(tt.input)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateNameOptional(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr error
	}{
		// Valid (including empty)
		{
			name:    "empty is allowed",
			input:   "",
			wantErr: nil,
		},
		{
			name:    "whitespace only treated as empty",
			input:   "   ",
			wantErr: nil,
		},
		{
			name:    "valid name",
			input:   "John Doe",
			wantErr: nil,
		},
		{
			name:    "exactly 100 characters",
			input:   strings.Repeat("a", 100),
			wantErr: nil,
		},

		// Too long
		{
			name:    "name too long",
			input:   strings.Repeat("a", 101),
			wantErr: ErrNameTooLong,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateNameOptional(tt.input)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateAvatarURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr error
	}{
		// Valid URLs
		{
			name:    "empty is allowed",
			url:     "",
			wantErr: nil,
		},
		{
			name:    "valid http URL",
			url:     "http://example.com/avatar.png",
			wantErr: nil,
		},
		{
			name:    "valid https URL",
			url:     "https://example.com/avatar.png",
			wantErr: nil,
		},
		{
			name:    "valid URL with query params",
			url:     "https://example.com/avatar.png?size=100",
			wantErr: nil,
		},
		{
			name:    "valid URL with path",
			url:     "https://cdn.example.com/users/123/avatar.jpg",
			wantErr: nil,
		},

		// Invalid format
		{
			name:    "missing protocol",
			url:     "example.com/avatar.png",
			wantErr: ErrInvalidAvatarURL,
		},
		{
			name:    "invalid protocol ftp",
			url:     "ftp://example.com/avatar.png",
			wantErr: ErrInvalidAvatarURL,
		},
		{
			name:    "javascript protocol",
			url:     "javascript:alert('xss')",
			wantErr: ErrInvalidAvatarURL,
		},
		{
			name:    "javascript protocol uppercase",
			url:     "JAVASCRIPT:alert('xss')",
			wantErr: ErrInvalidAvatarURL,
		},
		{
			name:    "data URI",
			url:     "data:image/png;base64,abc123",
			wantErr: ErrInvalidAvatarURL,
		},
		{
			name:    "data URI uppercase",
			url:     "DATA:image/png;base64,abc123",
			wantErr: ErrInvalidAvatarURL,
		},

		// Too long
		{
			name:    "URL too long",
			url:     "https://example.com/" + strings.Repeat("a", 2050),
			wantErr: ErrAvatarURLTooLong,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAvatarURL(tt.url)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidationError(t *testing.T) {
	t.Run("Error method formats correctly", func(t *testing.T) {
		err := &ValidationError{
			Field:   "email",
			Message: "invalid email address",
		}

		assert.Equal(t, "email: invalid email address", err.Error())
	})

	t.Run("NewValidationError creates correct error", func(t *testing.T) {
		originalErr := ErrInvalidEmail
		vErr := NewValidationError("email", originalErr)

		assert.Equal(t, "email", vErr.Field)
		assert.Equal(t, originalErr.Error(), vErr.Message)
	})
}

func TestSignUpValidator(t *testing.T) {
	validator := &SignUpValidator{}

	t.Run("valid signup", func(t *testing.T) {
		err := validator.Validate("user@example.com", "password123")
		assert.NoError(t, err)
	})

	t.Run("invalid email", func(t *testing.T) {
		err := validator.Validate("invalid-email", "password123")
		require.Error(t, err)

		var vErr *ValidationError
		require.ErrorAs(t, err, &vErr)
		assert.Equal(t, "email", vErr.Field)
	})

	t.Run("empty password", func(t *testing.T) {
		err := validator.Validate("user@example.com", "")
		require.Error(t, err)

		var vErr *ValidationError
		require.ErrorAs(t, err, &vErr)
		assert.Equal(t, "password", vErr.Field)
	})

	t.Run("empty email", func(t *testing.T) {
		err := validator.Validate("", "password123")
		require.Error(t, err)

		var vErr *ValidationError
		require.ErrorAs(t, err, &vErr)
		assert.Equal(t, "email", vErr.Field)
	})
}

func TestDashboardUserValidator(t *testing.T) {
	validator := &DashboardUserValidator{}

	t.Run("ValidateCreate valid input", func(t *testing.T) {
		err := validator.ValidateCreate("user@example.com", "password123", "John Doe")
		assert.NoError(t, err)
	})

	t.Run("ValidateCreate invalid email", func(t *testing.T) {
		err := validator.ValidateCreate("invalid", "password123", "John Doe")
		require.Error(t, err)

		var vErr *ValidationError
		require.ErrorAs(t, err, &vErr)
		assert.Equal(t, "email", vErr.Field)
	})

	t.Run("ValidateCreate empty password", func(t *testing.T) {
		err := validator.ValidateCreate("user@example.com", "", "John Doe")
		require.Error(t, err)

		var vErr *ValidationError
		require.ErrorAs(t, err, &vErr)
		assert.Equal(t, "password", vErr.Field)
	})

	t.Run("ValidateCreate empty name", func(t *testing.T) {
		err := validator.ValidateCreate("user@example.com", "password123", "")
		require.Error(t, err)

		var vErr *ValidationError
		require.ErrorAs(t, err, &vErr)
		assert.Equal(t, "full_name", vErr.Field)
	})

	t.Run("ValidateCreate name too long", func(t *testing.T) {
		err := validator.ValidateCreate("user@example.com", "password123", strings.Repeat("a", 101))
		require.Error(t, err)

		var vErr *ValidationError
		require.ErrorAs(t, err, &vErr)
		assert.Equal(t, "full_name", vErr.Field)
	})

	t.Run("ValidateUpdateProfile valid input", func(t *testing.T) {
		avatarURL := "https://example.com/avatar.png"
		err := validator.ValidateUpdateProfile("John Doe", &avatarURL)
		assert.NoError(t, err)
	})

	t.Run("ValidateUpdateProfile nil avatar", func(t *testing.T) {
		err := validator.ValidateUpdateProfile("John Doe", nil)
		assert.NoError(t, err)
	})

	t.Run("ValidateUpdateProfile empty name", func(t *testing.T) {
		err := validator.ValidateUpdateProfile("", nil)
		require.Error(t, err)

		var vErr *ValidationError
		require.ErrorAs(t, err, &vErr)
		assert.Equal(t, "full_name", vErr.Field)
	})

	t.Run("ValidateUpdateProfile invalid avatar URL", func(t *testing.T) {
		avatarURL := "javascript:alert('xss')"
		err := validator.ValidateUpdateProfile("John Doe", &avatarURL)
		require.Error(t, err)

		var vErr *ValidationError
		require.ErrorAs(t, err, &vErr)
		assert.Equal(t, "avatar_url", vErr.Field)
	})
}

func TestUserUpdateValidator(t *testing.T) {
	validator := &UserUpdateValidator{}

	t.Run("ValidateUpdate with valid email", func(t *testing.T) {
		email := "new@example.com"
		req := UpdateUserRequest{
			Email: &email,
		}
		err := validator.ValidateUpdate(req)
		assert.NoError(t, err)
	})

	t.Run("ValidateUpdate with nil email", func(t *testing.T) {
		req := UpdateUserRequest{
			Email: nil,
		}
		err := validator.ValidateUpdate(req)
		assert.NoError(t, err)
	})

	t.Run("ValidateUpdate with invalid email", func(t *testing.T) {
		email := "invalid-email"
		req := UpdateUserRequest{
			Email: &email,
		}
		err := validator.ValidateUpdate(req)
		require.Error(t, err)

		var vErr *ValidationError
		require.ErrorAs(t, err, &vErr)
		assert.Equal(t, "email", vErr.Field)
	})

	t.Run("ValidateUpdate with empty email string", func(t *testing.T) {
		email := ""
		req := UpdateUserRequest{
			Email: &email,
		}
		err := validator.ValidateUpdate(req)
		require.Error(t, err)

		var vErr *ValidationError
		require.ErrorAs(t, err, &vErr)
		assert.Equal(t, "email", vErr.Field)
	})
}

func TestValidateOTPContact(t *testing.T) {
	email := "user@example.com"
	phone := "+1234567890"

	tests := []struct {
		name    string
		email   *string
		phone   *string
		wantErr error
	}{
		{
			name:    "valid - email provided",
			email:   &email,
			phone:   nil,
			wantErr: nil,
		},
		{
			name:    "valid - phone provided",
			email:   nil,
			phone:   &phone,
			wantErr: nil,
		},
		{
			name:    "valid - both provided",
			email:   &email,
			phone:   &phone,
			wantErr: nil,
		},
		{
			name:    "invalid - neither provided",
			email:   nil,
			phone:   nil,
			wantErr: ErrOTPContactRequired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateOTPContact(tt.email, tt.phone)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateDashboardRole(t *testing.T) {
	tests := []struct {
		name    string
		role    string
		wantErr error
	}{
		{
			name:    "valid - dashboard_admin",
			role:    "dashboard_admin",
			wantErr: nil,
		},
		{
			name:    "valid - dashboard_user",
			role:    "dashboard_user",
			wantErr: nil,
		},
		{
			name:    "invalid - admin",
			role:    "admin",
			wantErr: ErrInvalidDashboardRole,
		},
		{
			name:    "invalid - user",
			role:    "user",
			wantErr: ErrInvalidDashboardRole,
		},
		{
			name:    "invalid - empty",
			role:    "",
			wantErr: ErrInvalidDashboardRole,
		},
		{
			name:    "invalid - random string",
			role:    "superadmin",
			wantErr: ErrInvalidDashboardRole,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDashboardRole(tt.role)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateDashboardPassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  error
	}{
		{
			name:     "valid password - exactly 12 chars",
			password: "123456789012",
			wantErr:  nil,
		},
		{
			name:     "valid password - longer than 12 chars",
			password: "verysecurepassword123!",
			wantErr:  nil,
		},
		{
			name:     "invalid password - 11 chars",
			password: "12345678901",
			wantErr:  ErrDashboardPasswordTooShort,
		},
		{
			name:     "invalid password - empty",
			password: "",
			wantErr:  ErrDashboardPasswordTooShort,
		},
		{
			name:     "invalid password - 8 chars (regular min but not dashboard min)",
			password: "12345678",
			wantErr:  ErrDashboardPasswordTooShort,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDashboardPassword(tt.password)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidationConstants(t *testing.T) {
	t.Run("constants have expected values", func(t *testing.T) {
		assert.Equal(t, 3, MinEmailLength)
		assert.Equal(t, 254, MaxEmailLength)
		assert.Equal(t, 1, MinNameLength)
		assert.Equal(t, 100, MaxNameLength)
		assert.Equal(t, 2048, MaxAvatarURLLength)
		assert.Equal(t, 64, MaxMetadataKeyLength)
		assert.Equal(t, 1024, MaxMetadataValueLength)
		assert.Equal(t, 12, MinDashboardPasswordLength)
	})
}
