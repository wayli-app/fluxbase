package api

import (
	"testing"

	"github.com/fluxbase-eu/fluxbase/internal/database"
	"github.com/stretchr/testify/assert"
)

func TestGetConflictTarget(t *testing.T) {
	h := &RESTHandler{}

	tests := []struct {
		name     string
		table    database.TableInfo
		expected string
	}{
		{
			name: "single column primary key",
			table: database.TableInfo{
				Name:       "users",
				PrimaryKey: []string{"id"},
			},
			expected: `"id"`,
		},
		{
			name: "composite primary key",
			table: database.TableInfo{
				Name:       "user_roles",
				PrimaryKey: []string{"user_id", "role_id"},
			},
			expected: `"user_id", "role_id"`,
		},
		{
			name: "three column composite key",
			table: database.TableInfo{
				Name:       "assignments",
				PrimaryKey: []string{"project_id", "user_id", "role"},
			},
			expected: `"project_id", "user_id", "role"`,
		},
		{
			name: "no primary key",
			table: database.TableInfo{
				Name:       "logs",
				PrimaryKey: []string{},
			},
			expected: "",
		},
		{
			name: "nil primary key",
			table: database.TableInfo{
				Name:       "temp_table",
				PrimaryKey: nil,
			},
			expected: "",
		},
		{
			name: "primary key with underscores",
			table: database.TableInfo{
				Name:       "special",
				PrimaryKey: []string{"user_id"},
			},
			expected: `"user_id"`,
		},
		{
			name: "primary key with mixed case",
			table: database.TableInfo{
				Name:       "mixed",
				PrimaryKey: []string{"UserId", "RoleId"},
			},
			expected: `"UserId", "RoleId"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := h.getConflictTarget(tt.table)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetConflictTargetUnquoted(t *testing.T) {
	h := &RESTHandler{}

	tests := []struct {
		name     string
		table    database.TableInfo
		expected []string
	}{
		{
			name: "single column",
			table: database.TableInfo{
				PrimaryKey: []string{"id"},
			},
			expected: []string{"id"},
		},
		{
			name: "multiple columns",
			table: database.TableInfo{
				PrimaryKey: []string{"user_id", "role_id"},
			},
			expected: []string{"user_id", "role_id"},
		},
		{
			name: "empty primary key",
			table: database.TableInfo{
				PrimaryKey: []string{},
			},
			expected: []string{},
		},
		{
			name: "nil primary key",
			table: database.TableInfo{
				PrimaryKey: nil,
			},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := h.getConflictTargetUnquoted(tt.table)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsInConflictTarget(t *testing.T) {
	h := &RESTHandler{}

	tests := []struct {
		name                  string
		column                string
		conflictTargetColumns []string
		expected              bool
	}{
		{
			name:                  "column found at start",
			column:                "id",
			conflictTargetColumns: []string{"id", "name", "email"},
			expected:              true,
		},
		{
			name:                  "column found in middle",
			column:                "name",
			conflictTargetColumns: []string{"id", "name", "email"},
			expected:              true,
		},
		{
			name:                  "column found at end",
			column:                "email",
			conflictTargetColumns: []string{"id", "name", "email"},
			expected:              true,
		},
		{
			name:                  "column not found",
			column:                "age",
			conflictTargetColumns: []string{"id", "name", "email"},
			expected:              false,
		},
		{
			name:                  "empty conflict target",
			column:                "id",
			conflictTargetColumns: []string{},
			expected:              false,
		},
		{
			name:                  "nil conflict target",
			column:                "id",
			conflictTargetColumns: nil,
			expected:              false,
		},
		{
			name:                  "single column match",
			column:                "id",
			conflictTargetColumns: []string{"id"},
			expected:              true,
		},
		{
			name:                  "single column no match",
			column:                "name",
			conflictTargetColumns: []string{"id"},
			expected:              false,
		},
		{
			name:                  "case sensitive match",
			column:                "UserId",
			conflictTargetColumns: []string{"UserId", "RoleId"},
			expected:              true,
		},
		{
			name:                  "case sensitive no match",
			column:                "userid",
			conflictTargetColumns: []string{"UserId", "RoleId"},
			expected:              false,
		},
		{
			name:                  "partial match should fail",
			column:                "user",
			conflictTargetColumns: []string{"user_id", "role_id"},
			expected:              false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := h.isInConflictTarget(tt.column, tt.conflictTargetColumns)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConflictTarget_Integration(t *testing.T) {
	h := &RESTHandler{}

	// Create a realistic table with composite PK
	table := database.TableInfo{
		Schema:     "public",
		Name:       "user_permissions",
		PrimaryKey: []string{"user_id", "resource_id", "permission"},
	}

	// Get quoted conflict target
	conflictTarget := h.getConflictTarget(table)
	assert.Equal(t, `"user_id", "resource_id", "permission"`, conflictTarget)

	// Get unquoted columns
	unquotedColumns := h.getConflictTargetUnquoted(table)
	assert.Equal(t, []string{"user_id", "resource_id", "permission"}, unquotedColumns)

	// Test isInConflictTarget for all PK columns
	assert.True(t, h.isInConflictTarget("user_id", unquotedColumns))
	assert.True(t, h.isInConflictTarget("resource_id", unquotedColumns))
	assert.True(t, h.isInConflictTarget("permission", unquotedColumns))

	// Test isInConflictTarget for non-PK columns
	assert.False(t, h.isInConflictTarget("created_at", unquotedColumns))
	assert.False(t, h.isInConflictTarget("updated_at", unquotedColumns))
}

func TestConflictTarget_EmptyPrimaryKey(t *testing.T) {
	h := &RESTHandler{}

	table := database.TableInfo{
		Schema:     "public",
		Name:       "logs",
		PrimaryKey: []string{},
	}

	// Should return empty string for no PK
	conflictTarget := h.getConflictTarget(table)
	assert.Equal(t, "", conflictTarget)

	// Should return empty slice
	unquotedColumns := h.getConflictTargetUnquoted(table)
	assert.Empty(t, unquotedColumns)

	// Any column check should return false
	assert.False(t, h.isInConflictTarget("id", unquotedColumns))
}

func TestConflictTarget_InvalidIdentifiers(t *testing.T) {
	h := &RESTHandler{}

	tests := []struct {
		name       string
		primaryKey []string
		expected   string
		desc       string
	}{
		{
			name:       "column with embedded quotes rejected",
			primaryKey: []string{"col\"umn"},
			expected:   "",
			desc:       "invalid identifier should return empty",
		},
		{
			name:       "column with special chars rejected",
			primaryKey: []string{"col;DROP TABLE"},
			expected:   "",
			desc:       "SQL injection attempt should be blocked",
		},
		{
			name:       "valid identifier passes",
			primaryKey: []string{"user_id"},
			expected:   `"user_id"`,
			desc:       "valid identifiers work normally",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			table := database.TableInfo{
				PrimaryKey: tt.primaryKey,
			}
			result := h.getConflictTarget(table)
			assert.Equal(t, tt.expected, result, tt.desc)
		})
	}
}
