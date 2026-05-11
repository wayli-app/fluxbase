package settings

import (
	"testing"
	"time"

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
			name:       "instance_admin can always edit",
			editableBy: []string{"admin"},
			userRole:   "instance_admin",
			expected:   true,
		},
		{
			name:       "admin can edit if in list",
			editableBy: []string{"admin", "instance_admin"},
			userRole:   "admin",
			expected:   true,
		},
		{
			name:       "admin can always edit",
			editableBy: []string{"instance_admin"},
			userRole:   "admin",
			expected:   true,
		},
		{
			name:       "service_role can always edit",
			editableBy: []string{"instance_admin"},
			userRole:   "service_role",
			expected:   true,
		},
		{
			name:       "unknown role cannot edit",
			editableBy: []string{"admin", "instance_admin"},
			userRole:   "user",
			expected:   false,
		},
		{
			name:       "empty editableBy list, instance_admin can still edit",
			editableBy: []string{},
			userRole:   "instance_admin",
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
				EditableBy:  []string{"instance_admin", "admin"},
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
			EditableBy:  []string{"instance_admin", "admin"},
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
		assert.Contains(t, setting.EditableBy, "instance_admin")
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
	svc := NewCustomSettingsService(nil, []byte("12345678901234567890123456789012"))
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
		result := CanEditSetting([]string{"instance_admin"}, "authenticated")
		assert.False(t, result)
	})

	t.Run("service_role bypasses editable_by check", func(t *testing.T) {
		result := CanEditSetting([]string{}, "service_role")
		assert.True(t, result)
	})
}

// =============================================================================
// Secret Setting Struct Tests
// =============================================================================

func TestSecretSettingMetadata_Struct(t *testing.T) {
	t.Run("creates metadata with all fields", func(t *testing.T) {
		id := uuid.New()
		userID := uuid.New()
		createdBy := uuid.New()

		metadata := SecretSettingMetadata{
			ID:          id,
			Key:         "secret.api.key",
			Description: "API key for external service",
			UserID:      &userID,
			CreatedBy:   &createdBy,
			UpdatedBy:   &createdBy,
		}

		assert.Equal(t, id, metadata.ID)
		assert.Equal(t, "secret.api.key", metadata.Key)
		assert.Equal(t, "API key for external service", metadata.Description)
		assert.NotNil(t, metadata.UserID)
		assert.Equal(t, userID, *metadata.UserID)
		assert.Equal(t, &createdBy, metadata.CreatedBy)
	})

	t.Run("system secret has nil user ID", func(t *testing.T) {
		metadata := SecretSettingMetadata{
			ID:  uuid.New(),
			Key: "system.encryption.key",
		}

		assert.Nil(t, metadata.UserID)
	})
}

func TestCreateSecretSettingRequest_Struct(t *testing.T) {
	t.Run("creates request with all fields", func(t *testing.T) {
		req := CreateSecretSettingRequest{
			Key:         "secret.db.password",
			Value:       "my-secure-password",
			Description: "Database password for production",
		}

		assert.Equal(t, "secret.db.password", req.Key)
		assert.Equal(t, "my-secure-password", req.Value)
		assert.Equal(t, "Database password for production", req.Description)
	})

	t.Run("minimal request", func(t *testing.T) {
		req := CreateSecretSettingRequest{
			Key:   "secret.token",
			Value: "token-value",
		}

		assert.Equal(t, "secret.token", req.Key)
		assert.Equal(t, "token-value", req.Value)
		assert.Empty(t, req.Description)
	})
}

func TestUpdateSecretSettingRequest_Struct(t *testing.T) {
	t.Run("update with new value", func(t *testing.T) {
		newValue := "updated-secret-value"
		req := UpdateSecretSettingRequest{
			Value: &newValue,
		}

		assert.NotNil(t, req.Value)
		assert.Equal(t, "updated-secret-value", *req.Value)
		assert.Nil(t, req.Description)
	})

	t.Run("update description only", func(t *testing.T) {
		newDesc := "Updated description"
		req := UpdateSecretSettingRequest{
			Description: &newDesc,
		}

		assert.Nil(t, req.Value)
		assert.NotNil(t, req.Description)
		assert.Equal(t, "Updated description", *req.Description)
	})

	t.Run("update both value and description", func(t *testing.T) {
		newValue := "new-value"
		newDesc := "New description"
		req := UpdateSecretSettingRequest{
			Value:       &newValue,
			Description: &newDesc,
		}

		assert.Equal(t, "new-value", *req.Value)
		assert.Equal(t, "New description", *req.Description)
	})
}

// =============================================================================
// User Setting Struct Tests
// =============================================================================

func TestUserSetting_Struct(t *testing.T) {
	t.Run("creates user setting with all fields", func(t *testing.T) {
		id := uuid.New()
		userID := uuid.New()

		setting := UserSetting{
			ID:          id,
			Key:         "user.theme",
			Value:       map[string]interface{}{"theme": "dark", "fontSize": 14},
			Description: "User's UI preferences",
			UserID:      userID,
		}

		assert.Equal(t, id, setting.ID)
		assert.Equal(t, "user.theme", setting.Key)
		assert.Equal(t, "dark", setting.Value["theme"])
		assert.Equal(t, 14, setting.Value["fontSize"])
		assert.Equal(t, "User's UI preferences", setting.Description)
		assert.Equal(t, userID, setting.UserID)
	})
}

func TestUserSettingWithSource_Struct(t *testing.T) {
	t.Run("user source", func(t *testing.T) {
		setting := UserSettingWithSource{
			Key:    "notifications.enabled",
			Value:  map[string]interface{}{"enabled": true},
			Source: "user",
		}

		assert.Equal(t, "notifications.enabled", setting.Key)
		assert.Equal(t, true, setting.Value["enabled"])
		assert.Equal(t, "user", setting.Source)
	})

	t.Run("system source (fallback)", func(t *testing.T) {
		setting := UserSettingWithSource{
			Key:    "notifications.enabled",
			Value:  map[string]interface{}{"enabled": false},
			Source: "system",
		}

		assert.Equal(t, "system", setting.Source)
	})
}

func TestCreateUserSettingRequest_Struct(t *testing.T) {
	t.Run("creates request with all fields", func(t *testing.T) {
		req := CreateUserSettingRequest{
			Key:         "user.preferences.display",
			Value:       map[string]interface{}{"compact": true, "showSidebar": false},
			Description: "Display preferences",
		}

		assert.Equal(t, "user.preferences.display", req.Key)
		assert.Equal(t, true, req.Value["compact"])
		assert.Equal(t, false, req.Value["showSidebar"])
		assert.Equal(t, "Display preferences", req.Description)
	})
}

func TestUpdateUserSettingRequest_Struct(t *testing.T) {
	t.Run("update value only", func(t *testing.T) {
		req := UpdateUserSettingRequest{
			Value: map[string]interface{}{"newKey": "newValue"},
		}

		assert.NotNil(t, req.Value)
		assert.Nil(t, req.Description)
	})

	t.Run("update with description", func(t *testing.T) {
		desc := "Updated description"
		req := UpdateUserSettingRequest{
			Value:       map[string]interface{}{"key": "value"},
			Description: &desc,
		}

		assert.Equal(t, "Updated description", *req.Description)
	})
}

// =============================================================================
// Benchmark Tests
// =============================================================================

func BenchmarkCanEditSetting(b *testing.B) {
	editableBy := []string{"instance_admin", "admin", "moderator", "editor"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CanEditSetting(editableBy, "editor")
	}
}

func BenchmarkCanEditSetting_AdminBypass(b *testing.B) {
	editableBy := []string{"moderator", "editor"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CanEditSetting(editableBy, "admin")
	}
}

func BenchmarkValidateKey(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ValidateKey("custom.settings.my.key.name")
	}
}

func BenchmarkValidateKey_Empty(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ValidateKey("")
	}
}

// =============================================================================
// Additional Value Type Validation Tests
// =============================================================================

func TestCreateCustomSettingRequest_ValueTypeValidation(t *testing.T) {
	validValueTypes := []string{"string", "number", "boolean", "json"}

	t.Run("all valid value types are accepted", func(t *testing.T) {
		for _, valueType := range validValueTypes {
			req := CreateCustomSettingRequest{
				Key:       "custom.test." + valueType,
				Value:     map[string]interface{}{"test": "data"},
				ValueType: valueType,
			}

			// Just validate the key - value type validation happens in CreateSetting
			err := ValidateKey(req.Key)
			assert.NoError(t, err, "value type %s should be valid", valueType)
		}
	})

	t.Run("value types are consistent", func(t *testing.T) {
		// Verify our test list matches what's documented
		expectedTypes := map[string]bool{
			"string":  true,
			"number":  true,
			"boolean": true,
			"json":    true,
		}

		for _, vt := range validValueTypes {
			assert.True(t, expectedTypes[vt], "value type %s should be in expected list", vt)
		}
	})
}

// =============================================================================
// Permission Edge Cases
// =============================================================================

func TestCanEditSetting_EdgeCases(t *testing.T) {
	t.Run("empty editable_by with non-admin role", func(t *testing.T) {
		result := CanEditSetting([]string{}, "authenticated")
		assert.False(t, result, "regular user should not edit when editable_by is empty")
	})

	t.Run("nil editable_by slice behaves like empty", func(t *testing.T) {
		result := CanEditSetting(nil, "authenticated")
		assert.False(t, result, "regular user should not edit with nil editable_by")
	})

	t.Run("case-sensitive role matching", func(t *testing.T) {
		editableBy := []string{"Admin", "Moderator"}
		// "admin" is a special role that bypasses the check, so it returns true
		// Use a non-special role to test case-sensitivity
		result := CanEditSetting(editableBy, "moderator") // lowercase, but list has "Moderator"
		assert.False(t, result, "role matching is case-sensitive")
	})

	t.Run("exact role match required", func(t *testing.T) {
		editableBy := []string{"moderator"}
		result := CanEditSetting(editableBy, "senior_moderator")
		assert.False(t, result, "exact role match is required")
	})

	t.Run("no wildcard support - only exact matches", func(t *testing.T) {
		specialRoles := []string{"*", "all", "any"}
		for _, role := range specialRoles {
			result := CanEditSetting([]string{role}, "authenticated")
			// The implementation doesn't support wildcards
			// Only instance_admin, admin, and service_role bypass the check
			assert.False(t, result, "special role '%s' should NOT match regular user", role)
		}
	})
}

// =============================================================================
// Key Validation Edge Cases
// =============================================================================

func TestValidateKey_EdgeCases(t *testing.T) {
	edgeCases := []struct {
		name    string
		key     string
		wantErr bool
	}{
		{"single character", "a", false},
		{"numeric key", "123", false},
		{"key with dots", "a.b.c.d.e", false},
		{"key with leading dot", ".starts-with-dot", false},
		{"key with trailing dot", "ends-with-dot.", false},
		{"key with consecutive dots", "a..b", false},
		{"unicode characters", "custom.设置.名称", false},
		{"very long key", string(make([]byte, 1000)), false}, // Just tests it doesn't panic
		{"key with special chars", "custom@test#key", false},
		{"key with spaces", "custom test key", false},
	}

	for _, tc := range edgeCases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateKey(tc.key)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// =============================================================================
// Setting Metadata Tests
// =============================================================================

func TestCustomSetting_MetadataHandling(t *testing.T) {
	t.Run("nil metadata is handled", func(t *testing.T) {
		setting := CustomSetting{
			ID:       uuid.New(),
			Key:      "custom.test",
			Value:    map[string]interface{}{},
			Metadata: nil,
		}

		assert.Nil(t, setting.Metadata)
	})

	t.Run("empty metadata map", func(t *testing.T) {
		setting := CustomSetting{
			ID:       uuid.New(),
			Key:      "custom.test",
			Value:    map[string]interface{}{},
			Metadata: map[string]interface{}{},
		}

		assert.NotNil(t, setting.Metadata)
		assert.Empty(t, setting.Metadata)
	})

	t.Run("metadata with various types", func(t *testing.T) {
		metadata := map[string]interface{}{
			"string":  "value",
			"number":  42,
			"boolean": true,
			"null":    nil,
			"array":   []string{"a", "b"},
			"object":  map[string]interface{}{"nested": "data"},
		}

		setting := CustomSetting{
			ID:       uuid.New(),
			Key:      "custom.test",
			Value:    map[string]interface{}{},
			Metadata: metadata,
		}

		assert.Equal(t, "value", setting.Metadata["string"])
		assert.Equal(t, 42, setting.Metadata["number"])
		assert.True(t, setting.Metadata["boolean"].(bool))
		assert.Nil(t, setting.Metadata["null"])
	})
}

// =============================================================================
// EditableBy Handling Tests
// =============================================================================

func TestCustomSetting_EditableByHandling(t *testing.T) {
	t.Run("nil editable_by is handled", func(t *testing.T) {
		setting := CustomSetting{
			ID:         uuid.New(),
			Key:        "custom.test",
			Value:      map[string]interface{}{},
			EditableBy: nil,
		}

		assert.Nil(t, setting.EditableBy)
		// nil editable_by means only admins can edit
		assert.True(t, CanEditSetting(setting.EditableBy, "instance_admin"))
		assert.False(t, CanEditSetting(setting.EditableBy, "authenticated"))
	})

	t.Run("empty editable_by array", func(t *testing.T) {
		setting := CustomSetting{
			ID:         uuid.New(),
			Key:        "custom.test",
			Value:      map[string]interface{}{},
			EditableBy: []string{},
		}

		assert.Empty(t, setting.EditableBy)
		assert.True(t, CanEditSetting(setting.EditableBy, "instance_admin"))
		assert.False(t, CanEditSetting(setting.EditableBy, "authenticated"))
	})

	t.Run("editable_by with duplicates", func(t *testing.T) {
		editableBy := []string{"admin", "moderator", "admin", "moderator"}
		setting := CustomSetting{
			ID:         uuid.New(),
			Key:        "custom.test",
			Value:      map[string]interface{}{},
			EditableBy: editableBy,
		}

		assert.True(t, CanEditSetting(setting.EditableBy, "admin"))
		// CanEditSetting should handle duplicates gracefully
	})
}

// =============================================================================
// Description Handling Tests
// =============================================================================

func TestCustomSetting_DescriptionHandling(t *testing.T) {
	t.Run("empty description", func(t *testing.T) {
		setting := CustomSetting{
			ID:          uuid.New(),
			Key:         "custom.test",
			Value:       map[string]interface{}{},
			Description: "",
		}

		assert.Equal(t, "", setting.Description)
	})

	t.Run("description with special characters", func(t *testing.T) {
		specialDesc := "Description with\nnewlines\ttabs\"quotes'apostrophes"
		setting := CustomSetting{
			ID:          uuid.New(),
			Key:         "custom.test",
			Value:       map[string]interface{}{},
			Description: specialDesc,
		}

		assert.Equal(t, specialDesc, setting.Description)
	})

	t.Run("description with unicode", func(t *testing.T) {
		unicodeDesc := "描述 Description 🌍 世界"
		setting := CustomSetting{
			ID:          uuid.New(),
			Key:         "custom.test",
			Value:       map[string]interface{}{},
			Description: unicodeDesc,
		}

		assert.Equal(t, unicodeDesc, setting.Description)
	})

	t.Run("very long description", func(t *testing.T) {
		longDesc := string(make([]byte, 10000))
		for i := range longDesc {
			longDesc = longDesc[:i] + "a" + longDesc[i+1:]
		}

		setting := CustomSetting{
			ID:          uuid.New(),
			Key:         "custom.test",
			Value:       map[string]interface{}{},
			Description: longDesc,
		}

		assert.Len(t, setting.Description, 10000)
	})
}

// =============================================================================
// Value Handling Tests
// =============================================================================

func TestCustomSetting_ValueHandling(t *testing.T) {
	t.Run("nil value map", func(t *testing.T) {
		setting := CustomSetting{
			ID:    uuid.New(),
			Key:   "custom.test",
			Value: nil,
		}

		assert.Nil(t, setting.Value)
	})

	t.Run("empty value map", func(t *testing.T) {
		setting := CustomSetting{
			ID:    uuid.New(),
			Key:   "custom.test",
			Value: map[string]interface{}{},
		}

		assert.NotNil(t, setting.Value)
		assert.Empty(t, setting.Value)
	})

	t.Run("value with nested structures", func(t *testing.T) {
		value := map[string]interface{}{
			"level1": map[string]interface{}{
				"level2": map[string]interface{}{
					"level3": "deep value",
				},
			},
			"array": []interface{}{1, 2, 3},
			"mixed": []interface{}{
				"string",
				42,
				true,
				map[string]interface{}{"nested": "object"},
			},
		}

		setting := CustomSetting{
			ID:    uuid.New(),
			Key:   "custom.test",
			Value: value,
		}

		assert.NotNil(t, setting.Value)
		assert.Equal(t, "deep value", setting.Value["level1"].(map[string]interface{})["level2"].(map[string]interface{})["level3"])
	})
}

// =============================================================================
// Timestamp Handling Tests
// =============================================================================

func TestCustomSetting_Timestamps(t *testing.T) {
	t.Run("created_at and updated_at are set", func(t *testing.T) {
		now := time.Now()
		setting := CustomSetting{
			ID:        uuid.New(),
			Key:       "custom.test",
			Value:     map[string]interface{}{},
			CreatedAt: now,
			UpdatedAt: now,
		}

		assert.False(t, setting.CreatedAt.IsZero())
		assert.False(t, setting.UpdatedAt.IsZero())
	})

	t.Run("updated_at can be after created_at", func(t *testing.T) {
		created := time.Now().Add(-24 * time.Hour)
		updated := time.Now()

		setting := CustomSetting{
			ID:        uuid.New(),
			Key:       "custom.test",
			Value:     map[string]interface{}{},
			CreatedAt: created,
			UpdatedAt: updated,
		}

		assert.True(t, setting.UpdatedAt.After(setting.CreatedAt))
	})
}

// =============================================================================
// User ID Tracking Tests
// =============================================================================

func TestCustomSetting_UserTracking(t *testing.T) {
	t.Run("created_by and updated_by tracking", func(t *testing.T) {
		createdBy := uuid.New()
		updatedBy := uuid.New()

		setting := CustomSetting{
			ID:        uuid.New(),
			Key:       "custom.test",
			Value:     map[string]interface{}{},
			CreatedBy: &createdBy,
			UpdatedBy: &updatedBy,
		}

		assert.Equal(t, createdBy, *setting.CreatedBy)
		assert.Equal(t, updatedBy, *setting.UpdatedBy)
	})

	t.Run("nil created_by and updated_by", func(t *testing.T) {
		setting := CustomSetting{
			ID:        uuid.New(),
			Key:       "custom.test",
			Value:     map[string]interface{}{},
			CreatedBy: nil,
			UpdatedBy: nil,
		}

		assert.Nil(t, setting.CreatedBy)
		assert.Nil(t, setting.UpdatedBy)
	})
}

// =============================================================================
// Secret Setting Tests
// =============================================================================

func TestSecretSetting_IsSecret(t *testing.T) {
	// Note: The current implementation doesn't have an IsSecret field on CustomSetting
	// This test documents the expected behavior when using separate tables/types
	t.Run("secret settings use separate type", func(t *testing.T) {
		metadata := SecretSettingMetadata{
			ID:          uuid.New(),
			Key:         "secret.test",
			Description: "Secret setting",
		}

		assert.Equal(t, "secret.test", metadata.Key)
		assert.Equal(t, "Secret setting", metadata.Description)
		// Value is never exposed in metadata
	})
}

func TestCreateSecretSettingRequest_Validation(t *testing.T) {
	t.Run("valid secret request", func(t *testing.T) {
		req := CreateSecretSettingRequest{
			Key:         "secret.api.key",
			Value:       "my-secret-value",
			Description: "API key secret",
		}

		assert.Equal(t, "secret.api.key", req.Key)
		assert.Equal(t, "my-secret-value", req.Value)
		assert.Equal(t, "API key secret", req.Description)
	})

	t.Run("secret with empty value", func(t *testing.T) {
		req := CreateSecretSettingRequest{
			Key:   "secret.empty",
			Value: "",
		}

		assert.Equal(t, "", req.Value)
	})

	t.Run("secret with special characters in value", func(t *testing.T) {
		specialValue := "key\nwith\nnewlines\tand\ttabs\"quotes'"
		req := CreateSecretSettingRequest{
			Key:   "secret.special",
			Value: specialValue,
		}

		assert.Equal(t, specialValue, req.Value)
	})
}

// =============================================================================
// Update Request Tests
// =============================================================================

func TestUpdateRequests_PartialUpdates(t *testing.T) {
	t.Run("UpdateCustomSettingRequest with only value", func(t *testing.T) {
		req := UpdateCustomSettingRequest{
			Value: map[string]interface{}{"updated": true},
		}

		assert.NotNil(t, req.Value)
		assert.Nil(t, req.Description)
		assert.Nil(t, req.EditableBy)
		assert.Nil(t, req.Metadata)
	})

	t.Run("UpdateCustomSettingRequest with only description", func(t *testing.T) {
		desc := "New description"
		req := UpdateCustomSettingRequest{
			Description: &desc,
		}

		assert.Nil(t, req.Value)
		assert.NotNil(t, req.Description)
		assert.Nil(t, req.EditableBy)
		assert.Nil(t, req.Metadata)
	})

	t.Run("UpdateSecretSettingRequest with only description", func(t *testing.T) {
		desc := "Updated description"
		req := UpdateSecretSettingRequest{
			Description: &desc,
		}

		assert.Nil(t, req.Value)
		assert.NotNil(t, req.Description)
	})

	t.Run("UpdateSecretSettingRequest with only value", func(t *testing.T) {
		val := "new-secret-value"
		req := UpdateSecretSettingRequest{
			Value: &val,
		}

		assert.NotNil(t, req.Value)
		assert.Nil(t, req.Description)
	})

	t.Run("UpdateUserSettingRequest partial", func(t *testing.T) {
		req := UpdateUserSettingRequest{
			Value: map[string]interface{}{"partial": "update"},
		}

		assert.NotNil(t, req.Value)
		assert.Nil(t, req.Description)
	})
}
