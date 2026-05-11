package branching

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nimbleflux/fluxbase/internal/config"
)

type mockTenantResolver struct {
	info *TenantDatabaseInfo
	err  error
}

func (m *mockTenantResolver) GetTenantDatabase(ctx context.Context, tenantID uuid.UUID) (*TenantDatabaseInfo, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.info, nil
}

type mockFDWRepairer struct {
	called  bool
	lastURL string
	lastTID uuid.UUID
	err     error
}

func (m *mockFDWRepairer) RepairFDWForBranch(ctx context.Context, branchDBURL string, tenantID uuid.UUID) error {
	m.called = true
	m.lastURL = branchDBURL
	m.lastTID = tenantID
	return m.err
}

func TestManager_ResolveTemplateDatabase_DefaultTenant(t *testing.T) {
	m := &Manager{
		mainDBName: "fluxbase",
	}

	branch := &Branch{
		TenantID: nil,
	}

	templateDB, err := m.resolveTemplateDatabase(context.Background(), branch, nil)
	assert.NoError(t, err)
	assert.Equal(t, "fluxbase", templateDB)
}

func TestManager_ResolveTemplateDatabase_TenantWithResolver(t *testing.T) {
	tenantID := uuid.New()
	m := &Manager{
		mainDBName: "fluxbase",
		tenantResolver: &mockTenantResolver{
			info: &TenantDatabaseInfo{
				DBName:    "tenant_acme",
				Slug:      "acme",
				IsDefault: false,
			},
		},
	}

	branch := &Branch{
		TenantID: &tenantID,
	}

	templateDB, err := m.resolveTemplateDatabase(context.Background(), branch, nil)
	assert.NoError(t, err)
	assert.Equal(t, "tenant_acme", templateDB)
}

func TestManager_ResolveTemplateDatabase_DefaultTenantWithResolver(t *testing.T) {
	tenantID := uuid.New()
	m := &Manager{
		mainDBName: "fluxbase",
		tenantResolver: &mockTenantResolver{
			info: &TenantDatabaseInfo{
				DBName:    "",
				Slug:      "default",
				IsDefault: true,
			},
		},
	}

	parentID := uuid.New()
	m.storage = NewStorage(nil, []byte(""))
	_ = parentID

	branch := &Branch{
		TenantID: &tenantID,
	}

	templateDB, err := m.resolveTemplateDatabase(context.Background(), branch, nil)
	assert.NoError(t, err)
	assert.Equal(t, "fluxbase", templateDB)
}

func TestManager_ResolveTemplateDatabase_ResolverError(t *testing.T) {
	tenantID := uuid.New()
	m := &Manager{
		mainDBName: "fluxbase",
		tenantResolver: &mockTenantResolver{
			err: assert.AnError,
		},
	}

	branch := &Branch{
		TenantID: &tenantID,
	}

	templateDB, err := m.resolveTemplateDatabase(context.Background(), branch, nil)
	assert.NoError(t, err)
	assert.Equal(t, "fluxbase", templateDB)
}

func TestManager_ResolveTemplateDatabase_NoResolver(t *testing.T) {
	tenantID := uuid.New()
	m := &Manager{
		mainDBName: "fluxbase",
	}

	branch := &Branch{
		TenantID: &tenantID,
	}

	templateDB, err := m.resolveTemplateDatabase(context.Background(), branch, nil)
	assert.NoError(t, err)
	assert.Equal(t, "fluxbase", templateDB)
}

func TestManager_SetTenantResolver(t *testing.T) {
	m := &Manager{}
	resolver := &mockTenantResolver{}
	m.SetTenantResolver(resolver)
	assert.Equal(t, resolver, m.tenantResolver)
}

func TestManager_SetFDWRepairer(t *testing.T) {
	m := &Manager{}
	repairer := &mockFDWRepairer{}
	m.SetFDWRepairer(repairer)
	assert.Equal(t, repairer, m.fdwRepairer)
}

func TestManager_RepairFDW_SkippedWhenNoTenant(t *testing.T) {
	repairer := &mockFDWRepairer{}
	m := &Manager{fdwRepairer: repairer}

	branch := &Branch{TenantID: nil}
	err := m.repairFDW(context.Background(), branch)
	assert.NoError(t, err)
	assert.False(t, repairer.called)
}

func TestManager_RepairFDW_SkippedWhenNoRepairer(t *testing.T) {
	tenantID := uuid.New()
	m := &Manager{}

	branch := &Branch{TenantID: &tenantID}
	err := m.repairFDW(context.Background(), branch)
	assert.NoError(t, err)
}

func TestTenantDatabaseInfo_Struct(t *testing.T) {
	info := &TenantDatabaseInfo{
		DBName:    "tenant_acme",
		Slug:      "acme",
		IsDefault: false,
	}
	assert.Equal(t, "tenant_acme", info.DBName)
	assert.Equal(t, "acme", info.Slug)
	assert.False(t, info.IsDefault)
}

func TestTenantDatabaseInfo_DefaultTenant(t *testing.T) {
	info := &TenantDatabaseInfo{
		DBName:    "",
		Slug:      "default",
		IsDefault: true,
	}
	assert.True(t, info.IsDefault)
	assert.Empty(t, info.DBName)
}

func TestManager_ResolveTemplateDatabase_SeparateTenantDB(t *testing.T) {
	tenantID := uuid.New()
	resolver := &mockTenantResolver{
		info: &TenantDatabaseInfo{
			DBName:    "tenant_corp",
			Slug:      "corp",
			IsDefault: false,
		},
	}
	m := &Manager{
		mainDBName:     "fluxbase",
		tenantResolver: resolver,
	}

	branch := &Branch{TenantID: &tenantID}

	templateDB, err := m.resolveTemplateDatabase(context.Background(), branch, nil)
	require.NoError(t, err)
	assert.Equal(t, "tenant_corp", templateDB,
		"Should use tenant's separate database when tenant has one")
}

func TestManager_ResolveTemplateDatabase_DefaultTenantFallsBack(t *testing.T) {
	tenantID := uuid.New()
	resolver := &mockTenantResolver{
		info: &TenantDatabaseInfo{
			DBName:    "",
			Slug:      "default",
			IsDefault: true,
		},
	}
	m := &Manager{
		mainDBName:     "fluxbase",
		tenantResolver: resolver,
	}

	branch := &Branch{TenantID: &tenantID}

	templateDB, err := m.resolveTemplateDatabase(context.Background(), branch, nil)
	require.NoError(t, err)
	assert.Equal(t, "fluxbase", templateDB,
		"Default tenant should fall back to main database")
}

func TestNewManager_WithTenantResolver(t *testing.T) {
	cfg := config.BranchingConfig{
		Enabled:          true,
		DatabasePrefix:   "branch_",
		MaxTotalBranches: 10,
	}
	storage := NewStorage(nil, []byte(""))

	m, err := NewManager(storage, cfg, nil, "postgres://user:pass@localhost:5432/fluxbase?sslmode=disable")
	require.NoError(t, err)
	require.NotNil(t, m)

	resolver := &mockTenantResolver{}
	m.SetTenantResolver(resolver)
	assert.Equal(t, resolver, m.tenantResolver)
}
