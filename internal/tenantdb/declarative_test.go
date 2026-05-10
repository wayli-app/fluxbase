package tenantdb

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func strPtr(s string) *string {
	return &s
}

func newTestService(t *testing.T, config DeclarativeConfig) *DeclarativeService {
	t.Helper()
	return NewDeclarativeService(config, "/usr/bin/pgschema", "localhost", 5432, "user", "pass", nil)
}

func TestNewDeclarativeService(t *testing.T) {
	config := DeclarativeConfig{
		Enabled:          true,
		SchemaDir:        "/schemas",
		OnCreate:         true,
		OnStartup:        false,
		AllowDestructive: false,
	}

	svc := NewDeclarativeService(config, "/bin/pgschema", "dbhost", 5433, "admin", "secret", nil)

	require.NotNil(t, svc)
	assert.Equal(t, config, svc.config)
	assert.Equal(t, "/bin/pgschema", svc.pgschemaPath)
	assert.Equal(t, "dbhost", svc.dbHost)
	assert.Equal(t, 5433, svc.dbPort)
	assert.Equal(t, "admin", svc.dbUser)
	assert.Equal(t, "secret", svc.dbPassword)
	assert.Nil(t, svc.adminPool)
}

func TestDeclarativeConfig_Fields(t *testing.T) {
	config := DeclarativeConfig{
		Enabled:          true,
		SchemaDir:        "/data/schemas",
		OnCreate:         true,
		OnStartup:        true,
		AllowDestructive: true,
	}

	svc := newTestService(t, config)

	assert.True(t, svc.config.Enabled)
	assert.Equal(t, "/data/schemas", svc.config.SchemaDir)
	assert.True(t, svc.config.OnCreate)
	assert.True(t, svc.config.OnStartup)
	assert.True(t, svc.config.AllowDestructive)
}

func TestGetSchemaFilePath(t *testing.T) {
	tests := []struct {
		name      string
		schemaDir string
		slug      string
		expected  string
	}{
		{
			name:      "basic path",
			schemaDir: "/schemas",
			slug:      "acme",
			expected:  filepath.Join("/schemas", "acme", "public.sql"),
		},
		{
			name:      "relative path",
			schemaDir: "schemas",
			slug:      "my-tenant",
			expected:  filepath.Join("schemas", "my-tenant", "public.sql"),
		},
		{
			name:      "nested dir",
			schemaDir: "/opt/fluxbase/schemas",
			slug:      "tenant-123",
			expected:  filepath.Join("/opt/fluxbase/schemas", "tenant-123", "public.sql"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newTestService(t, DeclarativeConfig{SchemaDir: tt.schemaDir})
			result := svc.GetSchemaFilePath(tt.slug)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHasSchemaFile(t *testing.T) {
	t.Run("returns true when file exists", func(t *testing.T) {
		tmpDir := t.TempDir()
		slug := "test-tenant"
		schemaDir := filepath.Join(tmpDir, "schemas")
		require.NoError(t, os.MkdirAll(filepath.Join(schemaDir, slug), 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(schemaDir, slug, "public.sql"), []byte("CREATE TABLE foo ();"), 0o644))

		svc := newTestService(t, DeclarativeConfig{SchemaDir: schemaDir})
		assert.True(t, svc.HasSchemaFile(slug))
	})

	t.Run("returns false when file does not exist", func(t *testing.T) {
		tmpDir := t.TempDir()
		svc := newTestService(t, DeclarativeConfig{SchemaDir: tmpDir})
		assert.False(t, svc.HasSchemaFile("nonexistent"))
	})

	t.Run("returns false when directory does not exist", func(t *testing.T) {
		svc := newTestService(t, DeclarativeConfig{SchemaDir: "/nonexistent/path"})
		assert.False(t, svc.HasSchemaFile("any-tenant"))
	})
}

func TestCalculateFingerprint(t *testing.T) {
	t.Run("returns correct sha256", func(t *testing.T) {
		tmpDir := t.TempDir()
		slug := "fp-tenant"
		schemaDir := filepath.Join(tmpDir, "schemas")
		slugDir := filepath.Join(schemaDir, slug)
		require.NoError(t, os.MkdirAll(slugDir, 0o755))

		content := []byte("CREATE TABLE users (id serial primary key);")
		require.NoError(t, os.WriteFile(filepath.Join(slugDir, "public.sql"), content, 0o644))

		expected := sha256.Sum256(content)
		expectedHex := hex.EncodeToString(expected[:])

		svc := newTestService(t, DeclarativeConfig{SchemaDir: schemaDir})
		fp, err := svc.CalculateFingerprint(slug)
		require.NoError(t, err)
		assert.Equal(t, expectedHex, fp)
	})

	t.Run("returns error when file does not exist", func(t *testing.T) {
		tmpDir := t.TempDir()
		svc := newTestService(t, DeclarativeConfig{SchemaDir: tmpDir})
		_, err := svc.CalculateFingerprint("missing")
		assert.Error(t, err)
	})

	t.Run("different content gives different fingerprint", func(t *testing.T) {
		tmpDir := t.TempDir()
		slug1 := "tenant-a"
		slug2 := "tenant-b"
		schemaDir := filepath.Join(tmpDir, "schemas")

		for _, slug := range []string{slug1, slug2} {
			dir := filepath.Join(schemaDir, slug)
			require.NoError(t, os.MkdirAll(dir, 0o755))
		}

		require.NoError(t, os.WriteFile(filepath.Join(schemaDir, slug1, "public.sql"), []byte("CREATE TABLE a ();"), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(schemaDir, slug2, "public.sql"), []byte("CREATE TABLE b ();"), 0o644))

		svc := newTestService(t, DeclarativeConfig{SchemaDir: schemaDir})
		fp1, err := svc.CalculateFingerprint(slug1)
		require.NoError(t, err)
		fp2, err := svc.CalculateFingerprint(slug2)
		require.NoError(t, err)
		assert.NotEqual(t, fp1, fp2)
	})

	t.Run("same content gives same fingerprint", func(t *testing.T) {
		tmpDir := t.TempDir()
		slug := "stable-tenant"
		schemaDir := filepath.Join(tmpDir, "schemas")
		dir := filepath.Join(schemaDir, slug)
		require.NoError(t, os.MkdirAll(dir, 0o755))

		content := []byte("CREATE TABLE stable (id int);")
		require.NoError(t, os.WriteFile(filepath.Join(dir, "public.sql"), content, 0o644))

		svc := newTestService(t, DeclarativeConfig{SchemaDir: schemaDir})
		fp1, err := svc.CalculateFingerprint(slug)
		require.NoError(t, err)
		fp2, err := svc.CalculateFingerprint(slug)
		require.NoError(t, err)
		assert.Equal(t, fp1, fp2)
	})
}

func TestPlanForTenant(t *testing.T) {
	ctx := context.Background()

	t.Run("returns error when tenant uses main database with nil DBName", func(t *testing.T) {
		svc := newTestService(t, DeclarativeConfig{Enabled: true})
		tenant := &Tenant{ID: "1", Slug: "default", DBName: nil}
		_, err := svc.PlanForTenant(ctx, tenant)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "main database")
	})

	t.Run("returns error when tenant uses main database with empty DBName", func(t *testing.T) {
		svc := newTestService(t, DeclarativeConfig{Enabled: true})
		tenant := &Tenant{ID: "1", Slug: "default", DBName: strPtr("")}
		_, err := svc.PlanForTenant(ctx, tenant)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "main database")
	})

	t.Run("returns empty plan when no schema file exists", func(t *testing.T) {
		tmpDir := t.TempDir()
		svc := newTestService(t, DeclarativeConfig{Enabled: true, SchemaDir: tmpDir})
		tenant := &Tenant{ID: "1", Slug: "no-file-tenant", DBName: strPtr("tenant_db")}
		plan, err := svc.PlanForTenant(ctx, tenant)
		require.NoError(t, err)
		require.NotNil(t, plan)
		assert.Empty(t, plan.Changes)
		assert.False(t, plan.HasChanges)
		assert.Empty(t, plan.DDL)
	})
}

func TestApplyTenantSchema(t *testing.T) {
	ctx := context.Background()

	t.Run("returns nil when disabled", func(t *testing.T) {
		svc := newTestService(t, DeclarativeConfig{Enabled: false})
		tenant := &Tenant{ID: "1", Slug: "acme", DBName: strPtr("tenant_acme")}
		err := svc.ApplyTenantSchema(ctx, tenant)
		assert.NoError(t, err)
	})

	t.Run("returns nil when tenant uses main database", func(t *testing.T) {
		svc := newTestService(t, DeclarativeConfig{Enabled: true})
		tenant := &Tenant{ID: "1", Slug: "default", DBName: nil}
		err := svc.ApplyTenantSchema(ctx, tenant)
		assert.NoError(t, err)
	})

	t.Run("returns nil when no schema file exists", func(t *testing.T) {
		tmpDir := t.TempDir()
		svc := newTestService(t, DeclarativeConfig{Enabled: true, SchemaDir: tmpDir})
		tenant := &Tenant{ID: "1", Slug: "no-file", DBName: strPtr("tenant_db")}
		err := svc.ApplyTenantSchema(ctx, tenant)
		assert.NoError(t, err)
	})

	t.Run("returns nil for tenant with empty db_name", func(t *testing.T) {
		svc := newTestService(t, DeclarativeConfig{Enabled: true})
		tenant := &Tenant{ID: "1", Slug: "empty-db", DBName: strPtr("")}
		err := svc.ApplyTenantSchema(ctx, tenant)
		assert.NoError(t, err)
	})
}

func TestApplyAllTenantSchemas(t *testing.T) {
	ctx := context.Background()

	t.Run("returns nil when disabled", func(t *testing.T) {
		svc := newTestService(t, DeclarativeConfig{Enabled: false})
		err := svc.ApplyAllTenantSchemas(ctx, nil)
		assert.NoError(t, err)
	})
}

func TestGetTenantSchemaStatus(t *testing.T) {
	ctx := context.Background()

	t.Run("returns basic status when no schema file", func(t *testing.T) {
		tmpDir := t.TempDir()
		svc := newTestService(t, DeclarativeConfig{SchemaDir: tmpDir})
		tenant := &Tenant{ID: "tenant-1", Slug: "acme", DBName: strPtr("tenant_acme")}

		status, err := svc.GetTenantSchemaStatus(ctx, tenant)
		require.NoError(t, err)
		require.NotNil(t, status)
		assert.Equal(t, "tenant-1", status.TenantID)
		assert.Equal(t, "acme", status.TenantSlug)
		assert.Contains(t, status.SchemaFile, "acme")
		assert.Empty(t, status.SchemaFingerprint)
	})

	t.Run("returns status with fingerprint when file exists", func(t *testing.T) {
		tmpDir := t.TempDir()
		slug := "fp-tenant"
		schemaDir := filepath.Join(tmpDir, "schemas")
		slugDir := filepath.Join(schemaDir, slug)
		require.NoError(t, os.MkdirAll(slugDir, 0o755))

		content := []byte("CREATE TABLE test (id int);")
		require.NoError(t, os.WriteFile(filepath.Join(slugDir, "public.sql"), content, 0o644))

		expected := sha256.Sum256(content)
		expectedHex := hex.EncodeToString(expected[:])

		svc := newTestService(t, DeclarativeConfig{SchemaDir: schemaDir})
		tenant := &Tenant{ID: "tenant-2", Slug: slug, DBName: nil}

		status, err := svc.GetTenantSchemaStatus(ctx, tenant)
		require.NoError(t, err)
		require.NotNil(t, status)
		assert.Equal(t, "tenant-2", status.TenantID)
		assert.Equal(t, slug, status.TenantSlug)
		assert.Equal(t, expectedHex, status.SchemaFingerprint)
		assert.NotEmpty(t, status.SchemaFile)
	})

	t.Run("returns error on unreadable file", func(t *testing.T) {
		tmpDir := t.TempDir()
		slug := "bad-tenant"
		schemaDir := filepath.Join(tmpDir, "schemas")
		slugDir := filepath.Join(schemaDir, slug)
		require.NoError(t, os.MkdirAll(slugDir, 0o755))

		filePath := filepath.Join(slugDir, "public.sql")
		require.NoError(t, os.WriteFile(filePath, []byte("data"), 0o644))
		require.NoError(t, os.Chmod(filePath, 0o000))

		defer func() { _ = os.Chmod(filePath, 0o644) }()

		svc := newTestService(t, DeclarativeConfig{SchemaDir: schemaDir})
		tenant := &Tenant{ID: "tenant-3", Slug: slug, DBName: strPtr("tenant_db")}

		_, err := svc.GetTenantSchemaStatus(ctx, tenant)
		assert.Error(t, err)
	})
}

func TestWritePlanToTemp(t *testing.T) {
	t.Run("writes valid JSON file that exists", func(t *testing.T) {
		svc := newTestService(t, DeclarativeConfig{})
		plan := &Plan{
			Changes: []Change{
				{Type: "create", ObjectType: "table", Schema: "public", Name: "users", SQL: "CREATE TABLE users (id int)", Destructive: false},
			},
			DDL:        "CREATE TABLE users (id int);",
			HasChanges: true,
		}

		path, err := svc.writePlanToTemp(plan)
		require.NoError(t, err)
		defer func() { _ = os.Remove(path) }()

		assert.FileExists(t, path)

		data, err := os.ReadFile(path)
		require.NoError(t, err)

		var readPlan Plan
		require.NoError(t, json.Unmarshal(data, &readPlan))
		assert.True(t, readPlan.HasChanges)
		require.Len(t, readPlan.Changes, 1)
		assert.Equal(t, "create", readPlan.Changes[0].Type)
		assert.Equal(t, "table", readPlan.Changes[0].ObjectType)
		assert.Equal(t, "public", readPlan.Changes[0].Schema)
		assert.Equal(t, "users", readPlan.Changes[0].Name)
		assert.Equal(t, "CREATE TABLE users (id int)", readPlan.Changes[0].SQL)
		assert.False(t, readPlan.Changes[0].Destructive)
		assert.Equal(t, "CREATE TABLE users (id int);", readPlan.DDL)
	})

	t.Run("writes empty plan", func(t *testing.T) {
		svc := newTestService(t, DeclarativeConfig{})
		plan := &Plan{}

		path, err := svc.writePlanToTemp(plan)
		require.NoError(t, err)
		defer func() { _ = os.Remove(path) }()

		data, err := os.ReadFile(path)
		require.NoError(t, err)

		var readPlan Plan
		require.NoError(t, json.Unmarshal(data, &readPlan))
		assert.Empty(t, readPlan.Changes)
		assert.False(t, readPlan.HasChanges)
	})

	t.Run("preserves multiple changes", func(t *testing.T) {
		svc := newTestService(t, DeclarativeConfig{})
		plan := &Plan{
			Changes: []Change{
				{Type: "create", ObjectType: "table", Schema: "public", Name: "t1", SQL: "CREATE TABLE t1 (id int)"},
				{Type: "alter", ObjectType: "table", Schema: "public", Name: "t2", SQL: "ALTER TABLE t2 ADD COLUMN x text"},
				{Type: "drop", ObjectType: "table", Schema: "public", Name: "t3", SQL: "DROP TABLE t3", Destructive: true},
			},
			HasChanges: true,
		}

		path, err := svc.writePlanToTemp(plan)
		require.NoError(t, err)
		defer func() { _ = os.Remove(path) }()

		data, err := os.ReadFile(path)
		require.NoError(t, err)

		var readPlan Plan
		require.NoError(t, json.Unmarshal(data, &readPlan))
		require.Len(t, readPlan.Changes, 3)
		assert.Equal(t, "drop", readPlan.Changes[2].Type)
		assert.True(t, readPlan.Changes[2].Destructive)
	})
}

func TestApplyTenantSchemaFromContent(t *testing.T) {
	ctx := context.Background()

	t.Run("returns error when disabled", func(t *testing.T) {
		svc := newTestService(t, DeclarativeConfig{Enabled: false})
		tenant := &Tenant{ID: "1", Slug: "acme", DBName: strPtr("tenant_acme")}
		err := svc.ApplyTenantSchemaFromContent(ctx, tenant, "CREATE TABLE foo ();")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not enabled")
	})

	t.Run("returns error when tenant uses main database", func(t *testing.T) {
		svc := newTestService(t, DeclarativeConfig{Enabled: true})
		tenant := &Tenant{ID: "1", Slug: "default", DBName: nil}
		err := svc.ApplyTenantSchemaFromContent(ctx, tenant, "CREATE TABLE foo ();")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "main database")
	})

	t.Run("returns error when content is empty", func(t *testing.T) {
		svc := newTestService(t, DeclarativeConfig{Enabled: true})
		tenant := &Tenant{ID: "1", Slug: "acme", DBName: strPtr("tenant_acme")}
		err := svc.ApplyTenantSchemaFromContent(ctx, tenant, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be empty")
	})
}

func TestDumpTenantSchema(t *testing.T) {
	ctx := context.Background()

	t.Run("returns error when tenant uses main database with nil DBName", func(t *testing.T) {
		svc := newTestService(t, DeclarativeConfig{Enabled: true})
		tenant := &Tenant{ID: "1", Slug: "default", DBName: nil}
		err := svc.DumpTenantSchema(ctx, tenant, "/tmp/dump.sql")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "main database")
	})

	t.Run("returns error when tenant uses main database with empty DBName", func(t *testing.T) {
		svc := newTestService(t, DeclarativeConfig{Enabled: true})
		tenant := &Tenant{ID: "1", Slug: "default", DBName: strPtr("")}
		err := svc.DumpTenantSchema(ctx, tenant, "/tmp/dump.sql")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "main database")
	})
}
