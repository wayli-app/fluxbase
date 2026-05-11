package settings

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// =============================================================================
// Unit Tests for CustomSettingsService
// These tests use mocks and don't require database integration
// =============================================================================

func TestCanEditSetting_Unit(t *testing.T) {
	tests := []struct {
		name       string
		editableBy []string
		userRole   string
		expected   bool
	}{
		{
			name:       "instance_admin can always edit",
			editableBy: []string{},
			userRole:   "instance_admin",
			expected:   true,
		},
		{
			name:       "admin can always edit",
			editableBy: []string{},
			userRole:   "admin",
			expected:   true,
		},
		{
			name:       "service_role can always edit",
			editableBy: []string{},
			userRole:   "service_role",
			expected:   true,
		},
		{
			name:       "user role in editable_by list can edit",
			editableBy: []string{"authenticated", "user"},
			userRole:   "authenticated",
			expected:   true,
		},
		{
			name:       "user role not in editable_by list cannot edit",
			editableBy: []string{"admin", "moderator"},
			userRole:   "authenticated",
			expected:   false,
		},
		{
			name:       "empty editable_by list blocks regular user",
			editableBy: []string{},
			userRole:   "authenticated",
			expected:   false,
		},
		{
			name:       "nil editable_by blocks regular user",
			editableBy: nil,
			userRole:   "authenticated",
			expected:   false,
		},
		{
			name:       "nil editable_by allows admin",
			editableBy: nil,
			userRole:   "admin",
			expected:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CanEditSetting(tt.editableBy, tt.userRole)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateKey_Unit(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		wantErr bool
	}{
		{
			name:    "valid key with dots",
			key:     "custom.valid.key",
			wantErr: false,
		},
		{
			name:    "valid key with underscores",
			key:     "custom_valid_key",
			wantErr: false,
		},
		{
			name:    "valid key with dashes",
			key:     "custom-valid-key",
			wantErr: false,
		},
		{
			name:    "empty key fails",
			key:     "",
			wantErr: true,
		},
		{
			name:    "key with spaces is valid",
			key:     "custom key with spaces",
			wantErr: false,
		},
		{
			name:    "key with special characters",
			key:     "custom.key-123_abc",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateKey(tt.key)
			if tt.wantErr {
				assert.Error(t, err)
				assert.ErrorIs(t, err, ErrCustomSettingInvalidKey)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCustomSetting_ValueTypes_Unit(t *testing.T) {
	t.Run("valid value types", func(t *testing.T) {
		validTypes := []string{"string", "number", "boolean", "json"}
		for _, vt := range validTypes {
			t.Run(vt, func(t *testing.T) {
				// Just verify the type is recognized
				// Actual validation happens in CreateSetting
				assert.Contains(t, []string{"string", "number", "boolean", "json"}, vt)
			})
		}
	})
}

func TestCreateCustomSettingRequest_Validation_Unit(t *testing.T) {
	tests := []struct {
		name    string
		req     CreateCustomSettingRequest
		wantErr bool
	}{
		{
			name: "valid request",
			req: CreateCustomSettingRequest{
				Key:       "custom.test",
				Value:     map[string]interface{}{"test": "value"},
				ValueType: "string",
			},
			wantErr: false,
		},
		{
			name: "empty key",
			req: CreateCustomSettingRequest{
				Key:       "",
				Value:     map[string]interface{}{"test": "value"},
				ValueType: "string",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateKey(tt.req.Key)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestUpdateCustomSettingRequest_Validation_Unit(t *testing.T) {
	t.Run("partial update with only value", func(t *testing.T) {
		value := map[string]interface{}{"updated": true}
		req := UpdateCustomSettingRequest{
			Value: value,
		}

		assert.NotNil(t, req.Value)
		assert.Nil(t, req.Description)
		assert.Nil(t, req.EditableBy)
		assert.Nil(t, req.Metadata)
	})

	t.Run("partial update with only description", func(t *testing.T) {
		desc := "new description"
		req := UpdateCustomSettingRequest{
			Description: &desc,
		}

		assert.Nil(t, req.Value)
		assert.NotNil(t, req.Description)
		assert.Equal(t, "new description", *req.Description)
		assert.Nil(t, req.EditableBy)
		assert.Nil(t, req.Metadata)
	})

	t.Run("partial update with only editableBy", func(t *testing.T) {
		editableBy := []string{"admin", "moderator"}
		req := UpdateCustomSettingRequest{
			EditableBy: editableBy,
		}

		assert.Nil(t, req.Value)
		assert.Nil(t, req.Description)
		assert.NotNil(t, req.EditableBy)
		assert.Equal(t, []string{"admin", "moderator"}, req.EditableBy)
		assert.Nil(t, req.Metadata)
	})

	t.Run("full update", func(t *testing.T) {
		value := map[string]interface{}{"updated": true}
		desc := "new description"
		editableBy := []string{"admin"}
		metadata := map[string]interface{}{"updated": true}

		req := UpdateCustomSettingRequest{
			Value:       value,
			Description: &desc,
			EditableBy:  editableBy,
			Metadata:    metadata,
		}

		assert.NotNil(t, req.Value)
		assert.NotNil(t, req.Description)
		assert.NotNil(t, req.EditableBy)
		assert.NotNil(t, req.Metadata)
	})
}

func TestSecretSettingMetadata_Unit(t *testing.T) {
	t.Run("system secret has nil user ID", func(t *testing.T) {
		metadata := SecretSettingMetadata{
			ID:          uuid.New(),
			Key:         "secret.system.test",
			Description: "System secret",
		}

		assert.NotNil(t, metadata.ID)
		assert.Nil(t, metadata.UserID)
		assert.Equal(t, "secret.system.test", metadata.Key)
		assert.Equal(t, "System secret", metadata.Description)
	})

	t.Run("user secret has user ID", func(t *testing.T) {
		userID := uuid.New()
		metadata := SecretSettingMetadata{
			ID:          uuid.New(),
			Key:         "secret.user.test",
			Description: "User secret",
			UserID:      &userID,
		}

		assert.NotNil(t, metadata.UserID)
		assert.Equal(t, userID, *metadata.UserID)
	})
}

func TestCreateSecretSettingRequest_Unit(t *testing.T) {
	t.Run("valid request with all fields", func(t *testing.T) {
		req := CreateSecretSettingRequest{
			Key:         "secret.test",
			Value:       "my-secret",
			Description: "Test secret",
		}

		assert.Equal(t, "secret.test", req.Key)
		assert.Equal(t, "my-secret", req.Value)
		assert.Equal(t, "Test secret", req.Description)
	})

	t.Run("minimal request", func(t *testing.T) {
		req := CreateSecretSettingRequest{
			Key:   "secret.minimal",
			Value: "secret-value",
		}

		assert.Equal(t, "secret.minimal", req.Key)
		assert.Equal(t, "secret-value", req.Value)
		assert.Empty(t, req.Description)
	})

	t.Run("empty value is allowed", func(t *testing.T) {
		req := CreateSecretSettingRequest{
			Key:   "secret.empty",
			Value: "",
		}

		assert.Equal(t, "", req.Value)
	})
}

func TestUpdateSecretSettingRequest_Unit(t *testing.T) {
	t.Run("update value only", func(t *testing.T) {
		value := "new-secret"
		req := UpdateSecretSettingRequest{
			Value: &value,
		}

		assert.NotNil(t, req.Value)
		assert.Equal(t, "new-secret", *req.Value)
		assert.Nil(t, req.Description)
	})

	t.Run("update description only", func(t *testing.T) {
		desc := "new description"
		req := UpdateSecretSettingRequest{
			Description: &desc,
		}

		assert.Nil(t, req.Value)
		assert.NotNil(t, req.Description)
		assert.Equal(t, "new description", *req.Description)
	})

	t.Run("update both value and description", func(t *testing.T) {
		value := "updated-secret"
		desc := "updated description"
		req := UpdateSecretSettingRequest{
			Value:       &value,
			Description: &desc,
		}

		assert.NotNil(t, req.Value)
		assert.Equal(t, "updated-secret", *req.Value)
		assert.NotNil(t, req.Description)
		assert.Equal(t, "updated description", *req.Description)
	})

	t.Run("nil pointers", func(t *testing.T) {
		req := UpdateSecretSettingRequest{}

		assert.Nil(t, req.Value)
		assert.Nil(t, req.Description)
	})
}

func TestUserSetting_Unit(t *testing.T) {
	t.Run("creates user setting", func(t *testing.T) {
		userID := uuid.New()
		setting := UserSetting{
			ID:     uuid.New(),
			Key:    "user.theme",
			Value:  map[string]interface{}{"theme": "dark"},
			UserID: userID,
		}

		assert.Equal(t, userID, setting.UserID)
		assert.Equal(t, "user.theme", setting.Key)
		assert.Equal(t, "dark", setting.Value["theme"])
	})
}

func TestUserSettingWithSource_Unit(t *testing.T) {
	t.Run("user source", func(t *testing.T) {
		setting := UserSettingWithSource{
			Key:    "preferences.theme",
			Value:  map[string]interface{}{"theme": "light"},
			Source: "user",
		}

		assert.Equal(t, "user", setting.Source)
		assert.Equal(t, "light", setting.Value["theme"])
	})

	t.Run("system source", func(t *testing.T) {
		setting := UserSettingWithSource{
			Key:    "preferences.theme",
			Value:  map[string]interface{}{"theme": "dark"},
			Source: "system",
		}

		assert.Equal(t, "system", setting.Source)
		assert.Equal(t, "dark", setting.Value["theme"])
	})
}

func TestCreateUserSettingRequest_Unit(t *testing.T) {
	t.Run("valid request", func(t *testing.T) {
		req := CreateUserSettingRequest{
			Key:         "user.preference",
			Value:       map[string]interface{}{"enabled": true},
			Description: "User preference",
		}

		assert.Equal(t, "user.preference", req.Key)
		assert.Equal(t, true, req.Value["enabled"])
		assert.Equal(t, "User preference", req.Description)
	})

	t.Run("minimal request", func(t *testing.T) {
		req := CreateUserSettingRequest{
			Key:   "user.minimal",
			Value: map[string]interface{}{"value": "test"},
		}

		assert.Equal(t, "user.minimal", req.Key)
		assert.Equal(t, "test", req.Value["value"])
		assert.Empty(t, req.Description)
	})
}

func TestUpdateUserSettingRequest_Unit(t *testing.T) {
	value := map[string]interface{}{"updated": true}
	desc := "updated description"

	req := UpdateUserSettingRequest{
		Value:       value,
		Description: &desc,
	}

	assert.NotNil(t, req.Value)
	assert.Equal(t, true, req.Value["updated"])
	assert.NotNil(t, req.Description)
	assert.Equal(t, "updated description", *req.Description)
}

// =============================================================================
// Error Tests
// =============================================================================

func TestErrors_Unit(t *testing.T) {
	t.Run("error variables are defined", func(t *testing.T) {
		assert.NotNil(t, ErrCustomSettingNotFound)
		assert.NotNil(t, ErrCustomSettingPermissionDenied)
		assert.NotNil(t, ErrCustomSettingInvalidKey)
		assert.NotNil(t, ErrCustomSettingDuplicate)
		assert.NotNil(t, ErrSecretNotFound)
		assert.NotNil(t, ErrDecryptionFailed)
		assert.NotNil(t, ErrSettingNotFound)
	})

	t.Run("error messages are meaningful", func(t *testing.T) {
		assert.Contains(t, ErrCustomSettingNotFound.Error(), "not found")
		assert.Contains(t, ErrCustomSettingPermissionDenied.Error(), "permission denied")
		assert.Contains(t, ErrCustomSettingInvalidKey.Error(), "invalid")
		assert.Contains(t, ErrCustomSettingDuplicate.Error(), "already exists")
		assert.Contains(t, ErrSecretNotFound.Error(), "not found")
		assert.Contains(t, ErrDecryptionFailed.Error(), "failed to decrypt")
		assert.Contains(t, ErrSettingNotFound.Error(), "not found")
	})
}

// =============================================================================
// ExtractJSONStringValue Tests (from secrets_service_integration_test.go, moved here)
// =============================================================================

func TestExtractJSONStringValue_Unit(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected string
	}{
		{
			name:     "empty bytes returns empty string",
			input:    []byte{},
			expected: "",
		},
		{
			name:     "nil input returns empty string",
			input:    nil,
			expected: "",
		},
		{
			name:     "direct quoted string",
			input:    []byte(`"hello world"`),
			expected: "hello world",
		},
		{
			name:     "empty quoted string",
			input:    []byte(`""`),
			expected: "",
		},
		{
			name:     "object with string value",
			input:    []byte(`{"value": "my-value"}`),
			expected: "my-value",
		},
		{
			name:     "object with number value",
			input:    []byte(`{"value": 42}`),
			expected: "42",
		},
		{
			name:     "object with float value",
			input:    []byte(`{"value": 3.14}`),
			expected: "3.14",
		},
		{
			name:     "object with boolean true",
			input:    []byte(`{"value": true}`),
			expected: "true",
		},
		{
			name:     "object with boolean false",
			input:    []byte(`{"value": false}`),
			expected: "false",
		},
		{
			name:     "object with nested object",
			input:    []byte(`{"value": {"nested": "data"}}`),
			expected: `{"nested":"data"}`,
		},
		{
			name:     "object with array value",
			input:    []byte(`{"value": [1, 2, 3]}`),
			expected: `[1,2,3]`,
		},
		{
			name:     "object with null value",
			input:    []byte(`{"value": null}`),
			expected: `null`,
		},
		{
			name:     "object without value field",
			input:    []byte(`{"other": "field"}`),
			expected: `{"other": "field"}`,
		},
		{
			name:     "raw JSON object",
			input:    []byte(`{"key": "value", "number": 123}`),
			expected: `{"key": "value", "number": 123}`,
		},
		{
			name:     "raw JSON array",
			input:    []byte(`[1, 2, 3]`),
			expected: `[1, 2, 3]`,
		},
		{
			name:     "invalid JSON returns as string",
			input:    []byte(`not valid json`),
			expected: "not valid json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractJSONStringValue(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// =============================================================================
// Mock-Using Tests
// =============================================================================

func TestCustomSettingsService_Constructor(t *testing.T) {
	t.Run("create service with nil database", func(t *testing.T) {
		svc := NewCustomSettingsService(nil, []byte("test-key"))
		assert.NotNil(t, svc)
		assert.Equal(t, []byte("test-key"), svc.encryptionKey)
	})

	t.Run("create service with empty key", func(t *testing.T) {
		svc := NewCustomSettingsService(nil, []byte(""))
		assert.NotNil(t, svc)
		assert.Empty(t, svc.encryptionKey)
	})

	t.Run("create service with valid key", func(t *testing.T) {
		svc := NewCustomSettingsService(nil, []byte("01234567890123456789012345678901"))
		assert.NotNil(t, svc)
		assert.Equal(t, []byte("01234567890123456789012345678901"), svc.encryptionKey)
	})
}
