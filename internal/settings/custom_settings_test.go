package settings

import (
	"testing"

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
