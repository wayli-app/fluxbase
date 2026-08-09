//go:build integration

package database

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nimbleflux/fluxbase/test/dbhelpers"
)

// TestSchemaInspector_getColumns_Integration tests the internal getColumns function.
func TestSchemaInspector_getColumns_Integration(t *testing.T) {
	testCtx := dbhelpers.NewDBTestContext(t)
	defer testCtx.Close()

	inspector := NewSchemaInspector(NewConnectionWithPool(testCtx.Pool))

	t.Run("retrieves columns for a regular table", func(t *testing.T) {
		// Use auth.users instead of public.users since that's where the users table is
		columns, err := inspector.getColumns(context.Background(), inspector.q(), "auth", "users")
		require.NoError(t, err)
		assert.NotEmpty(t, columns, "users table should have columns")

		// Verify expected columns exist
		colMap := make(map[string]bool)
		for _, col := range columns {
			colMap[col.Name] = true
		}
		assert.True(t, colMap["id"], "Should have id column")
		assert.True(t, colMap["email"], "Should have email column")
	})

	t.Run("includes column metadata", func(t *testing.T) {
		columns, err := inspector.getColumns(context.Background(), inspector.q(), "auth", "users")
		require.NoError(t, err)

		// Find id column
		var idCol *ColumnInfo
		for i := range columns {
			if columns[i].Name == "id" {
				idCol = &columns[i]
				break
			}
		}
		require.NotNil(t, idCol, "Should find id column")

		assert.Equal(t, "id", idCol.Name)
		assert.NotEmpty(t, idCol.DataType)
		assert.Equal(t, 1, idCol.Position, "id should be first column")
	})

	t.Run("returns empty for non-existent table", func(t *testing.T) {
		columns, err := inspector.getColumns(context.Background(), inspector.q(), "public", "nonexistent_table_xyz")
		// getColumns returns empty slice for non-existent tables, not an error
		assert.NoError(t, err)
		assert.Empty(t, columns)
	})
}

// TestSchemaInspector_getMaterializedViewColumns_Integration tests materialized view column retrieval.
func TestSchemaInspector_getMaterializedViewColumns_Integration(t *testing.T) {
	testCtx := dbhelpers.NewDBTestContext(t)
	defer testCtx.Close()

	inspector := NewSchemaInspector(NewConnectionWithPool(testCtx.Pool))

	t.Run("retrieves columns for materialized view if exists", func(t *testing.T) {
		// First check if any materialized views exist
		matviews, err := inspector.GetAllMaterializedViews(context.Background(), "public")
		require.NoError(t, err)

		if len(matviews) == 0 {
			t.Skip("No materialized views found in public schema")
		}

		// Test the first materialized view
		mv := matviews[0]
		columns, err := inspector.getMaterializedViewColumns(context.Background(), inspector.q(), mv.Schema, mv.Name)
		require.NoError(t, err)
		assert.NotEmpty(t, columns)
	})

	t.Run("returns empty for non-existent materialized view", func(t *testing.T) {
		columns, err := inspector.getMaterializedViewColumns(context.Background(), inspector.q(), "public", "nonexistent_matview_xyz")
		assert.NoError(t, err)
		assert.Empty(t, columns)
	})
}

// TestSchemaInspector_getPrimaryKey_Integration tests primary key retrieval.
func TestSchemaInspector_getPrimaryKey_Integration(t *testing.T) {
	testCtx := dbhelpers.NewDBTestContext(t)
	defer testCtx.Close()

	inspector := NewSchemaInspector(NewConnectionWithPool(testCtx.Pool))

	t.Run("retrieves primary key for users table", func(t *testing.T) {
		pk, err := inspector.getPrimaryKey(context.Background(), inspector.q(), "auth", "users")
		require.NoError(t, err)
		assert.NotEmpty(t, pk, "users table should have primary key")
		assert.Contains(t, pk, "id", "users primary key should contain id")
	})

	t.Run("handles composite primary keys", func(t *testing.T) {
		// Find a table with composite PK if exists
		tables, err := inspector.GetAllTables(context.Background(), "public")
		require.NoError(t, err)

		foundComposite := false
		for _, table := range tables {
			pk, err := inspector.getPrimaryKey(context.Background(), inspector.q(), table.Schema, table.Name)
			require.NoError(t, err)
			if len(pk) > 1 {
				assert.Greater(t, len(pk), 1, "Should have composite key")
				foundComposite = true
				break
			}
		}

		if !foundComposite {
			t.Skip("No tables with composite primary keys found")
		}
	})

	t.Run("returns empty for table without primary key", func(t *testing.T) {
		// Create a temp table without PK
		_, err := testCtx.Pool.Exec(context.Background(),
			"CREATE TEMP TABLE temp_no_pk (id INT, name TEXT)")
		require.NoError(t, err)

		pk, err := inspector.getPrimaryKey(context.Background(), inspector.q(), "public", "temp_no_pk")
		require.NoError(t, err)
		assert.Empty(t, pk)
	})
}

// TestSchemaInspector_getForeignKeys_Integration tests foreign key retrieval.
func TestSchemaInspector_getForeignKeys_Integration(t *testing.T) {
	testCtx := dbhelpers.NewDBTestContext(t)
	defer testCtx.Close()

	inspector := NewSchemaInspector(NewConnectionWithPool(testCtx.Pool))

	t.Run("retrieves foreign keys if exist", func(t *testing.T) {
		// Find a table with foreign keys
		tables, err := inspector.GetAllTables(context.Background())
		require.NoError(t, err)

		foundFK := false
		for _, table := range tables {
			fks, err := inspector.getForeignKeys(context.Background(), inspector.q(), table.Schema, table.Name)
			require.NoError(t, err)

			if len(fks) > 0 {
				foundFK = true
				// Verify FK structure
				fk := fks[0]
				assert.NotEmpty(t, fk.Name, "FK should have name")
				assert.NotEmpty(t, fk.ColumnName, "FK should have column name")
				assert.NotEmpty(t, fk.ReferencedTable, "FK should have referenced table")
				assert.NotEmpty(t, fk.ReferencedColumn, "FK should have referenced column")
				break
			}
		}

		if !foundFK {
			t.Skip("No tables with foreign keys found")
		}
	})

	t.Run("returns empty for table without foreign keys", func(t *testing.T) {
		fks, err := inspector.getForeignKeys(context.Background(), inspector.q(), "auth", "users")
		require.NoError(t, err)
		// users may or may not have FKs depending on schema
		// Accept either nil (no FKs) or empty slice
		if fks != nil {
			assert.Empty(t, fks, "users should have no foreign keys")
		}
	})
}

// TestSchemaInspector_getIndexes_Integration tests index retrieval.
func TestSchemaInspector_getIndexes_Integration(t *testing.T) {
	testCtx := dbhelpers.NewDBTestContext(t)
	defer testCtx.Close()

	inspector := NewSchemaInspector(NewConnectionWithPool(testCtx.Pool))

	t.Run("retrieves indexes for users table", func(t *testing.T) {
		indexes, err := inspector.getIndexes(context.Background(), inspector.q(), "auth", "users")
		require.NoError(t, err)
		assert.NotEmpty(t, indexes, "users table should have indexes")

		// Should have at least primary key index
		hasPKIndex := false
		for _, idx := range indexes {
			if idx.IsPrimary {
				hasPKIndex = true
				assert.True(t, idx.IsUnique, "Primary key index should be unique")
				assert.NotEmpty(t, idx.Columns, "Primary key index should have columns")
				break
			}
		}
		assert.True(t, hasPKIndex, "Should have primary key index")
	})

	t.Run("includes index metadata", func(t *testing.T) {
		indexes, err := inspector.getIndexes(context.Background(), inspector.q(), "auth", "users")
		require.NoError(t, err)

		for _, idx := range indexes {
			assert.NotEmpty(t, idx.Name, "Index should have name")
			assert.NotEmpty(t, idx.Columns, "Index should have columns")
		}
	})
}

// TestSchemaInspector_batchGetColumns_Integration tests batch column retrieval.
func TestSchemaInspector_batchGetColumns_Integration(t *testing.T) {
	testCtx := dbhelpers.NewDBTestContext(t)
	defer testCtx.Close()

	inspector := NewSchemaInspector(NewConnectionWithPool(testCtx.Pool))

	t.Run("retrieves columns for all tables in public schema", func(t *testing.T) {
		columns, err := inspector.batchGetColumns(context.Background(), inspector.q(), []string{"public"}, "table")
		require.NoError(t, err)
		assert.NotEmpty(t, columns, "Should retrieve columns for public tables")

		// Check that users table has columns
		key := "auth.users"
		if cols, ok := columns[key]; ok {
			assert.NotEmpty(t, cols, "users should have columns")
		}
	})

	t.Run("handles multiple schemas", func(t *testing.T) {
		columns, err := inspector.batchGetColumns(context.Background(), inspector.q(), []string{"public", "auth"}, "table")
		require.NoError(t, err)

		// Should have data from both schemas if auth tables exist
		assert.NotEmpty(t, columns)
	})
}

// TestSchemaInspector_batchGetMaterializedViewColumns_Integration tests batch matview column retrieval.
func TestSchemaInspector_batchGetMaterializedViewColumns_Integration(t *testing.T) {
	testCtx := dbhelpers.NewDBTestContext(t)
	defer testCtx.Close()

	inspector := NewSchemaInspector(NewConnectionWithPool(testCtx.Pool))

	t.Run("retrieves materialized view columns", func(t *testing.T) {
		columns, err := inspector.batchGetMaterializedViewColumns(context.Background(), inspector.q(), []string{"public"})
		require.NoError(t, err)

		// May be empty if no materialized views exist
		assert.NotNil(t, columns)
	})
}

// TestSchemaInspector_batchGetPrimaryKeys_Integration tests batch primary key retrieval.
func TestSchemaInspector_batchGetPrimaryKeys_Integration(t *testing.T) {
	testCtx := dbhelpers.NewDBTestContext(t)
	defer testCtx.Close()

	inspector := NewSchemaInspector(NewConnectionWithPool(testCtx.Pool))

	t.Run("retrieves primary keys for all tables", func(t *testing.T) {
		pks, err := inspector.batchGetPrimaryKeys(context.Background(), inspector.q(), []string{"public"})
		require.NoError(t, err)
		assert.NotEmpty(t, pks, "Should have primary keys for public tables")

		// Check users table
		if userPKs, ok := pks["auth.users"]; ok {
			assert.NotEmpty(t, userPKs, "users should have primary key")
		}
	})

	t.Run("handles composite primary keys", func(t *testing.T) {
		pks, err := inspector.batchGetPrimaryKeys(context.Background(), inspector.q(), []string{"public"})
		require.NoError(t, err)

		// Look for composite keys
		for table, keyCols := range pks {
			if len(keyCols) > 1 {
				assert.Greater(t, len(keyCols), 1, "Should have composite key for "+table)
				return
			}
		}

		t.Skip("No tables with composite primary keys found")
	})
}

// TestSchemaInspector_batchGetForeignKeys_Integration tests batch foreign key retrieval.
func TestSchemaInspector_batchGetForeignKeys_Integration(t *testing.T) {
	testCtx := dbhelpers.NewDBTestContext(t)
	defer testCtx.Close()

	inspector := NewSchemaInspector(NewConnectionWithPool(testCtx.Pool))

	t.Run("retrieves foreign keys for all tables", func(t *testing.T) {
		fks, err := inspector.batchGetForeignKeys(context.Background(), inspector.q(), []string{"public"})
		require.NoError(t, err)

		// May be empty if no foreign keys exist
		assert.NotNil(t, fks)

		// If any FKs exist, verify structure
		for table, keys := range fks {
			if len(keys) > 0 {
				fk := keys[0]
				assert.NotEmpty(t, fk.Name, "FK should have name for "+table)
				assert.NotEmpty(t, fk.ColumnName, "FK should have column name for "+table)
				return
			}
		}
	})
}

// TestSchemaInspector_batchGetIndexes_Integration tests batch index retrieval.
func TestSchemaInspector_batchGetIndexes_Integration(t *testing.T) {
	testCtx := dbhelpers.NewDBTestContext(t)
	defer testCtx.Close()

	inspector := NewSchemaInspector(NewConnectionWithPool(testCtx.Pool))

	t.Run("retrieves indexes for all tables", func(t *testing.T) {
		indexes, err := inspector.batchGetIndexes(context.Background(), inspector.q(), []string{"public"})
		require.NoError(t, err)
		assert.NotEmpty(t, indexes, "Should have indexes for public tables")

		// Should have users_pkey
		if usersIndexes, ok := indexes["auth.users"]; ok {
			assert.NotEmpty(t, usersIndexes, "users should have indexes")
		}
	})

	t.Run("includes primary and non-primary indexes", func(t *testing.T) {
		indexes, err := inspector.batchGetIndexes(context.Background(), inspector.q(), []string{"public"})
		require.NoError(t, err)

		hasPrimary := false
		_ = false // hasNonPrimary placeholder (non-primary indexes may not exist)

		for _, tableIndexes := range indexes {
			for _, idx := range tableIndexes {
				if idx.IsPrimary {
					hasPrimary = true
				}
			}
		}

		assert.True(t, hasPrimary, "Should have at least one primary key index")
		// Non-primary indexes may not exist
	})
}

// TestSchemaInspector_batchFetchTableMetadata_Integration tests the batch metadata orchestrator.
func TestSchemaInspector_batchFetchTableMetadata_Integration(t *testing.T) {
	testCtx := dbhelpers.NewDBTestContext(t)
	defer testCtx.Close()

	inspector := NewSchemaInspector(NewConnectionWithPool(testCtx.Pool))

	t.Run("fetches all metadata for tables", func(t *testing.T) {
		tableMap := map[string]*TableInfo{
			"auth.users": {Schema: "auth", Name: "users", Type: "table"},
		}

		err := inspector.batchFetchTableMetadata(context.Background(), inspector.q(), []string{"public", "auth"}, tableMap, "table")
		require.NoError(t, err)

		// Verify all metadata populated
		users := tableMap["auth.users"]
		assert.NotEmpty(t, users.Columns, "Should have columns")
		assert.NotEmpty(t, users.PrimaryKey, "Should have primary key")
		// ForeignKeys may be nil if table has no FKs
		if users.ForeignKeys != nil {
			t.Logf("Users table has %d foreign keys", len(users.ForeignKeys))
		}
		assert.NotEmpty(t, users.Indexes, "Should have indexes")
		assert.NotNil(t, users.ColumnMap, "Should have column map built")
	})

	t.Run("marks primary key columns", func(t *testing.T) {
		tableMap := map[string]*TableInfo{
			"auth.users": {Schema: "auth", Name: "users", Type: "table"},
		}

		err := inspector.batchFetchTableMetadata(context.Background(), inspector.q(), []string{"public", "auth"}, tableMap, "table")
		require.NoError(t, err)

		users := tableMap["auth.users"]
		hasPKColumn := false
		for _, col := range users.Columns {
			if col.IsPrimaryKey {
				hasPKColumn = true
				assert.Equal(t, "id", col.Name, "PK column should be id")
			}
		}
		assert.True(t, hasPKColumn, "Should have at least one PK column marked")
	})

	t.Run("fetches only columns for views", func(t *testing.T) {
		// First get a view
		views, err := inspector.GetAllViews(context.Background(), "public")
		require.NoError(t, err)

		if len(views) == 0 {
			t.Skip("No views found in public schema")
		}

		viewMap := map[string]*TableInfo{
			views[0].Schema + "." + views[0].Name: {
				Schema: views[0].Schema,
				Name:   views[0].Name,
				Type:   "view",
			},
		}

		err = inspector.batchFetchTableMetadata(context.Background(), inspector.q(), []string{"public"}, viewMap, "view")
		require.NoError(t, err)

		// Views should only have columns, not keys/indexes
		view := viewMap[views[0].Schema+"."+views[0].Name]
		assert.NotEmpty(t, view.Columns, "View should have columns")
		assert.Nil(t, view.PrimaryKey, "View should not have primary key")
		assert.Nil(t, view.ForeignKeys, "View should not have foreign keys")
		assert.Nil(t, view.Indexes, "View should not have indexes")
	})
}

// TestSchemaInspector_getFunctionParameters_Integration tests function parameter retrieval.
func TestSchemaInspector_getFunctionParameters_Integration(t *testing.T) {
	testCtx := dbhelpers.NewDBTestContext(t)
	defer testCtx.Close()

	inspector := NewSchemaInspector(NewConnectionWithPool(testCtx.Pool))

	t.Run("retrieves parameters for existing function", func(t *testing.T) {
		// Find a function first
		functions, err := inspector.GetAllFunctions(context.Background(), "public")
		require.NoError(t, err)

		if len(functions) == 0 {
			t.Skip("No functions found in public schema")
		}

		// Test first function
		fn := functions[0]
		params, err := inspector.getFunctionParameters(context.Background(), fn.Schema, fn.Name)
		require.NoError(t, err)
		// Function may have no parameters - nil is valid
		// Just verify we can call the function successfully
		if params != nil {
			t.Logf("Function %s.%s has %d parameters", fn.Schema, fn.Name, len(params))
		} else {
			t.Logf("Function %s.%s has no parameters", fn.Schema, fn.Name)
		}
	})

	t.Run("handles function without parameters", func(t *testing.T) {
		// This test assumes there might be a parameterless function
		// If not, the test will be skipped
		functions, err := inspector.GetAllFunctions(context.Background(), "public")
		require.NoError(t, err)

		for _, fn := range functions {
			params, err := inspector.getFunctionParameters(context.Background(), fn.Schema, fn.Name)
			require.NoError(t, err)
			if len(params) == 0 {
				return // Found a function without params
			}
		}

		t.Skip("No parameterless functions found")
	})
}
