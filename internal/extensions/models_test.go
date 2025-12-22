package extensions

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestExtension_Struct(t *testing.T) {
	t.Run("creates extension with all fields", func(t *testing.T) {
		now := time.Now()
		enabledBy := "admin-user"

		ext := Extension{
			ID:               "ext-123",
			Name:             "postgis",
			DisplayName:      "PostGIS",
			Description:      "Spatial database extender",
			Category:         "geospatial",
			IsCore:           false,
			RequiresRestart:  true,
			DocumentationURL: "https://postgis.net",
			IsEnabled:        true,
			IsInstalled:      true,
			InstalledVersion: "3.4.0",
			EnabledAt:        &now,
			EnabledBy:        &enabledBy,
			CreatedAt:        now,
			UpdatedAt:        now,
		}

		assert.Equal(t, "ext-123", ext.ID)
		assert.Equal(t, "postgis", ext.Name)
		assert.Equal(t, "PostGIS", ext.DisplayName)
		assert.Equal(t, "geospatial", ext.Category)
		assert.False(t, ext.IsCore)
		assert.True(t, ext.RequiresRestart)
		assert.True(t, ext.IsEnabled)
		assert.True(t, ext.IsInstalled)
		assert.Equal(t, "3.4.0", ext.InstalledVersion)
		assert.Equal(t, &enabledBy, ext.EnabledBy)
	})

	t.Run("handles optional fields as nil", func(t *testing.T) {
		ext := Extension{
			ID:   "ext-456",
			Name: "uuid-ossp",
		}

		assert.Nil(t, ext.EnabledAt)
		assert.Nil(t, ext.EnabledBy)
		assert.Empty(t, ext.Description)
		assert.Empty(t, ext.InstalledVersion)
	})
}

func TestAvailableExtension_Struct(t *testing.T) {
	now := time.Now()

	ext := AvailableExtension{
		ID:               "avail-123",
		Name:             "pg_stat_statements",
		DisplayName:      "PG Stat Statements",
		Description:      "Track execution statistics of SQL statements",
		Category:         "monitoring",
		IsCore:           true,
		RequiresRestart:  false,
		DocumentationURL: "https://www.postgresql.org/docs/current/pgstatstatements.html",
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	assert.Equal(t, "avail-123", ext.ID)
	assert.Equal(t, "pg_stat_statements", ext.Name)
	assert.Equal(t, "monitoring", ext.Category)
	assert.True(t, ext.IsCore)
	assert.False(t, ext.RequiresRestart)
}

func TestEnabledExtension_Struct(t *testing.T) {
	t.Run("creates enabled extension record", func(t *testing.T) {
		now := time.Now()
		enabledBy := "admin"

		enabled := EnabledExtension{
			ID:            "enabled-123",
			ExtensionName: "pgcrypto",
			EnabledAt:     now,
			EnabledBy:     &enabledBy,
			IsActive:      true,
			CreatedAt:     now,
			UpdatedAt:     now,
		}

		assert.Equal(t, "enabled-123", enabled.ID)
		assert.Equal(t, "pgcrypto", enabled.ExtensionName)
		assert.Equal(t, "admin", *enabled.EnabledBy)
		assert.True(t, enabled.IsActive)
		assert.Nil(t, enabled.DisabledAt)
		assert.Nil(t, enabled.DisabledBy)
		assert.Nil(t, enabled.ErrorMessage)
	})

	t.Run("handles disabled extension with error", func(t *testing.T) {
		now := time.Now()
		disabledAt := now.Add(-time.Hour)
		disabledBy := "system"
		errorMsg := "Extension requires superuser privileges"

		enabled := EnabledExtension{
			ID:            "enabled-456",
			ExtensionName: "pg_cron",
			EnabledAt:     now.Add(-24 * time.Hour),
			DisabledAt:    &disabledAt,
			DisabledBy:    &disabledBy,
			IsActive:      false,
			ErrorMessage:  &errorMsg,
		}

		assert.False(t, enabled.IsActive)
		assert.Equal(t, "system", *enabled.DisabledBy)
		assert.Contains(t, *enabled.ErrorMessage, "superuser")
	})
}

func TestCategory_Struct(t *testing.T) {
	cat := Category{
		ID:    "geospatial",
		Name:  "Geospatial",
		Count: 5,
	}

	assert.Equal(t, "geospatial", cat.ID)
	assert.Equal(t, "Geospatial", cat.Name)
	assert.Equal(t, 5, cat.Count)
}

func TestListExtensionsResponse_Struct(t *testing.T) {
	resp := ListExtensionsResponse{
		Extensions: []Extension{
			{ID: "1", Name: "postgis"},
			{ID: "2", Name: "pgcrypto"},
		},
		Categories: []Category{
			{ID: "geo", Name: "Geospatial", Count: 1},
			{ID: "crypto", Name: "Cryptography", Count: 1},
		},
	}

	assert.Len(t, resp.Extensions, 2)
	assert.Len(t, resp.Categories, 2)
}

func TestExtensionStatusResponse_Struct(t *testing.T) {
	t.Run("enabled extension status", func(t *testing.T) {
		status := ExtensionStatusResponse{
			Name:             "postgis",
			IsEnabled:        true,
			IsInstalled:      true,
			InstalledVersion: "3.4.0",
		}

		assert.Equal(t, "postgis", status.Name)
		assert.True(t, status.IsEnabled)
		assert.True(t, status.IsInstalled)
		assert.Equal(t, "3.4.0", status.InstalledVersion)
		assert.Empty(t, status.Error)
	})

	t.Run("failed extension status", func(t *testing.T) {
		status := ExtensionStatusResponse{
			Name:        "pg_cron",
			IsEnabled:   false,
			IsInstalled: false,
			Error:       "Extension not available",
		}

		assert.False(t, status.IsEnabled)
		assert.False(t, status.IsInstalled)
		assert.Equal(t, "Extension not available", status.Error)
	})
}

func TestEnableExtensionRequest_Struct(t *testing.T) {
	t.Run("with custom schema", func(t *testing.T) {
		req := EnableExtensionRequest{
			Schema: "extensions",
		}
		assert.Equal(t, "extensions", req.Schema)
	})

	t.Run("without schema (defaults to public)", func(t *testing.T) {
		req := EnableExtensionRequest{}
		assert.Empty(t, req.Schema)
	})
}

func TestEnableExtensionResponse_Struct(t *testing.T) {
	t.Run("successful enable", func(t *testing.T) {
		resp := EnableExtensionResponse{
			Name:    "postgis",
			Success: true,
			Message: "Extension enabled successfully",
			Version: "3.4.0",
		}

		assert.Equal(t, "postgis", resp.Name)
		assert.True(t, resp.Success)
		assert.Contains(t, resp.Message, "successfully")
		assert.Equal(t, "3.4.0", resp.Version)
	})

	t.Run("failed enable", func(t *testing.T) {
		resp := EnableExtensionResponse{
			Name:    "pg_cron",
			Success: false,
			Message: "Insufficient privileges",
		}

		assert.False(t, resp.Success)
		assert.Empty(t, resp.Version)
	})
}

func TestDisableExtensionResponse_Struct(t *testing.T) {
	resp := DisableExtensionResponse{
		Name:    "pgcrypto",
		Success: true,
		Message: "Extension disabled successfully",
	}

	assert.Equal(t, "pgcrypto", resp.Name)
	assert.True(t, resp.Success)
}

func TestPostgresExtension_Struct(t *testing.T) {
	t.Run("installed extension", func(t *testing.T) {
		version := "1.6"
		ext := PostgresExtension{
			Name:             "uuid-ossp",
			DefaultVersion:   "1.1",
			InstalledVersion: &version,
			Comment:          "generate universally unique identifiers (UUIDs)",
		}

		assert.Equal(t, "uuid-ossp", ext.Name)
		assert.Equal(t, "1.1", ext.DefaultVersion)
		assert.Equal(t, "1.6", *ext.InstalledVersion)
		assert.Contains(t, ext.Comment, "UUID")
	})

	t.Run("not installed extension", func(t *testing.T) {
		ext := PostgresExtension{
			Name:           "postgis",
			DefaultVersion: "3.4.0",
			Comment:        "PostGIS geometry and geography spatial types",
		}

		assert.Nil(t, ext.InstalledVersion)
	})
}

func TestCategoryDisplayNames(t *testing.T) {
	t.Run("contains expected categories", func(t *testing.T) {
		expectedCategories := []string{
			"core", "geospatial", "ai_ml", "monitoring", "scheduling",
			"data_types", "text_search", "indexing", "networking",
			"testing", "maintenance", "performance", "foreign_data",
			"triggers", "sampling", "utilities",
		}

		for _, cat := range expectedCategories {
			_, exists := CategoryDisplayNames[cat]
			assert.True(t, exists, "Category %s should exist", cat)
		}
	})

	t.Run("display names are human readable", func(t *testing.T) {
		assert.Equal(t, "Core", CategoryDisplayNames["core"])
		assert.Equal(t, "Geospatial", CategoryDisplayNames["geospatial"])
		assert.Equal(t, "AI & Machine Learning", CategoryDisplayNames["ai_ml"])
		assert.Equal(t, "Text Search", CategoryDisplayNames["text_search"])
		assert.Equal(t, "Foreign Data", CategoryDisplayNames["foreign_data"])
	})
}
