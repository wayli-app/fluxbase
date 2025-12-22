package settings

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestCanEditSetting(t *testing.T) {
	tests := []struct {
		name       string
		editableBy []string
		userRole   string
		expected   bool
	}{
		{
			name:       "dashboard_admin can always edit",
			editableBy: []string{"admin"},
			userRole:   "dashboard_admin",
			expected:   true,
		},
		{
			name:       "admin can edit if in list",
			editableBy: []string{"admin", "dashboard_admin"},
			userRole:   "admin",
			expected:   true,
		},
		{
			name:       "admin can always edit",
			editableBy: []string{"dashboard_admin"},
			userRole:   "admin",
			expected:   true,
		},
		{
			name:       "service_role can always edit",
			editableBy: []string{"dashboard_admin"},
			userRole:   "service_role",
			expected:   true,
		},
		{
			name:       "unknown role cannot edit",
			editableBy: []string{"admin", "dashboard_admin"},
			userRole:   "user",
			expected:   false,
		},
		{
			name:       "empty editableBy list, dashboard_admin can still edit",
			editableBy: []string{},
			userRole:   "dashboard_admin",
			expected:   true,
		},
		{
			name:       "empty editableBy list, admin can still edit",
			editableBy: []string{},
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

func TestValidateKey(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		wantErr bool
	}{
		{
			name:    "valid key",
			key:     "custom.test.valid",
			wantErr: false,
		},
		{
			name:    "simple key",
			key:     "mykey",
			wantErr: false,
		},
		{
			name:    "key with underscores",
			key:     "custom_key_name",
			wantErr: false,
		},
		{
			name:    "key with dashes",
			key:     "custom-key-name",
			wantErr: false,
		},
		{
			name:    "empty key fails",
			key:     "",
			wantErr: true,
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

func TestCreateCustomSettingRequest_Validation(t *testing.T) {
	tests := []struct {
		name       string
		req        CreateCustomSettingRequest
		shouldFail bool
		reason     string
	}{
		{
			name: "valid request with all fields",
			req: CreateCustomSettingRequest{
				Key:         "custom.test.key",
				Value:       map[string]interface{}{"enabled": true},
				ValueType:   "json",
				Description: "Test description",
				EditableBy:  []string{"dashboard_admin", "admin"},
				Metadata:    map[string]interface{}{"category": "test"},
			},
			shouldFail: false,
		},
		{
			name: "valid request with minimal fields",
			req: CreateCustomSettingRequest{
				Key:   "custom.minimal",
				Value: map[string]interface{}{"value": "test"},
			},
			shouldFail: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Just validate the key field since that's what we can test without a database
			err := ValidateKey(tt.req.Key)
			if tt.shouldFail {
				assert.Error(t, err, tt.reason)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCustomSetting_Struct(t *testing.T) {
	t.Run("creates setting with all fields", func(t *testing.T) {
		id := uuid.New()
		createdBy := uuid.New()

		setting := CustomSetting{
			ID:          id,
			Key:         "custom.test.key",
			Value:       map[string]interface{}{"enabled": true, "count": 42},
			ValueType:   "json",
			Description: "A test setting",
			EditableBy:  []string{"dashboard_admin", "admin"},
			Metadata:    map[string]interface{}{"version": "1.0"},
			CreatedBy:   &createdBy,
			UpdatedBy:   &createdBy,
		}

		assert.Equal(t, id, setting.ID)
		assert.Equal(t, "custom.test.key", setting.Key)
		assert.Equal(t, true, setting.Value["enabled"])
		assert.Equal(t, 42, setting.Value["count"])
		assert.Equal(t, "json", setting.ValueType)
		assert.Equal(t, "A test setting", setting.Description)
		assert.Len(t, setting.EditableBy, 2)
		assert.Contains(t, setting.EditableBy, "dashboard_admin")
		assert.Equal(t, "1.0", setting.Metadata["version"])
		assert.Equal(t, &createdBy, setting.CreatedBy)
	})

	t.Run("handles nil optional fields", func(t *testing.T) {
		setting := CustomSetting{
			ID:        uuid.New(),
			Key:       "custom.minimal",
			Value:     map[string]interface{}{},
			ValueType: "string",
		}

		assert.Nil(t, setting.CreatedBy)
		assert.Nil(t, setting.UpdatedBy)
		assert.Empty(t, setting.Description)
		assert.Nil(t, setting.Metadata)
		assert.Nil(t, setting.EditableBy)
	})
}

func TestUpdateCustomSettingRequest_Struct(t *testing.T) {
	t.Run("creates update request with all fields", func(t *testing.T) {
		desc := "Updated description"
		req := UpdateCustomSettingRequest{
			Value:       map[string]interface{}{"updated": true},
			Description: &desc,
			EditableBy:  []string{"admin"},
			Metadata:    map[string]interface{}{"updated_reason": "test"},
		}

		assert.Equal(t, true, req.Value["updated"])
		assert.Equal(t, "Updated description", *req.Description)
		assert.Contains(t, req.EditableBy, "admin")
		assert.Equal(t, "test", req.Metadata["updated_reason"])
	})

	t.Run("handles partial update", func(t *testing.T) {
		req := UpdateCustomSettingRequest{
			Value: map[string]interface{}{"only": "value"},
		}

		assert.Nil(t, req.Description)
		assert.Nil(t, req.EditableBy)
		assert.Nil(t, req.Metadata)
	})
}

func TestCustomSettingErrors(t *testing.T) {
	t.Run("error types are defined", func(t *testing.T) {
		assert.NotNil(t, ErrCustomSettingNotFound)
		assert.NotNil(t, ErrCustomSettingPermissionDenied)
		assert.NotNil(t, ErrCustomSettingInvalidKey)
		assert.NotNil(t, ErrCustomSettingDuplicate)
	})

	t.Run("error messages are meaningful", func(t *testing.T) {
		assert.Contains(t, ErrCustomSettingNotFound.Error(), "not found")
		assert.Contains(t, ErrCustomSettingPermissionDenied.Error(), "permission denied")
		assert.Contains(t, ErrCustomSettingInvalidKey.Error(), "invalid")
		assert.Contains(t, ErrCustomSettingDuplicate.Error(), "already exists")
	})
}

func TestNewCustomSettingsService(t *testing.T) {
	// Just test that it doesn't panic with nil db
	// Real database integration tests would use an actual connection
	svc := NewCustomSettingsService(nil)
	assert.NotNil(t, svc)
}

func TestCanEditSetting_AdditionalCases(t *testing.T) {
	t.Run("user role in editable_by list can edit", func(t *testing.T) {
		result := CanEditSetting([]string{"moderator", "editor"}, "editor")
		assert.True(t, result)
	})

	t.Run("user role not in editable_by list cannot edit", func(t *testing.T) {
		result := CanEditSetting([]string{"moderator", "editor"}, "viewer")
		assert.False(t, result)
	})

	t.Run("authenticated user cannot edit admin-only settings", func(t *testing.T) {
		result := CanEditSetting([]string{"dashboard_admin"}, "authenticated")
		assert.False(t, result)
	})

	t.Run("service_role bypasses editable_by check", func(t *testing.T) {
		result := CanEditSetting([]string{}, "service_role")
		assert.True(t, result)
	})
}
