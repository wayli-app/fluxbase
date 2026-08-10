package migrations

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pgplex/pgparser/nodes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nimbleflux/fluxbase/internal/database/bootstrap"
)

// Unit tests for the additional pure helpers in declarative.go that were at 0%
// -short coverage: firstSQLIdentifier (its unterminated-quote branch),
// extractColumnSQL, loadCrossSchemaFKNames, filterManagedFKDrops,
// CalculateFingerprint, preprocessSchemaFile, writePlanToTemp, and the
// trivial constructor/setters.
//
// All are deterministic and DB-free. Convention matches declarative_test.go:
// testify, package migrations (white-box so unexported funcs are reachable).

// =============================================================================
// firstSQLIdentifier (direct coverage of the core token reader)
// =============================================================================
//
// Contract source: declarative.go:1097. The table-name extractors delegate to
// this, so it is only covered indirectly there. The unterminated-quote branch
// (closed==false -> "") is not exercised by any existing test.

func TestFirstSQLIdentifier(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"empty", "", ""},
		{"only whitespace", "   ", ""},
		{"simple unquoted", "users", "users"},
		{"leading whitespace skipped", "  users", "users"},
		{"trailing semicolon stripped", "users;", "users"},
		{"stops at whitespace", "users rest of statement", "users"},
		{"schema-qualified prefix stripped", "platform.users", "users"},
		{"schema-qualified with trailing semicolon", "platform.users;", "users"},
		{"quoted single word", `"users"`, "users"},
		{"quoted multi-word preserved", `"User Table"`, "User Table"},
		{"quoted reserved word", `"order"`, "order"},
		{"quoted with escaped double-quote", `"with "" inside"`, `with " inside`},
		{"unterminated quote returns empty", `"unterminated`, ""},
		{"empty quotes return empty", `""`, ""},
		// Quoted identifiers stop at the closing quote, so a following
		// schema-qualified segment ("schema"."table") is never consumed and the
		// first quoted name is returned whole. A dot *inside* quotes, however, is
		// subject to the trailing-segment split (documents current behavior).
		{"quoted then dot never reached", `"schema"."table"`, "schema"},
		{"dot inside quotes splits to last segment", `"a.b.c"`, "c"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, firstSQLIdentifier(tt.in))
		})
	}
}

// =============================================================================
// extractColumnSQL
// =============================================================================
//
// Contract source: declarative.go:1150. Given the original SQL text and a
// pgparser byte offset (nodes.ParseLoc) pointing at a column definition, return
// the raw column SQL: scan back past whitespace to the column start, then
// forward to the next top-level comma or closing paren (honoring paren depth so
// types like varchar(255) don't split early). loc<0 or loc>=len -> "".

func TestExtractColumnSQL(t *testing.T) {
	t.Parallel()

	t.Run("negative location returns empty", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "", extractColumnSQL("id int", -1))
	})

	t.Run("location past end returns empty", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "", extractColumnSQL("id int", nodes.ParseLoc(100)))
	})

	t.Run("single column at start", func(t *testing.T) {
		t.Parallel()
		// loc points at 'i' of "id"; no comma/paren -> returns whole tail trimmed.
		assert.Equal(t, "id int", extractColumnSQL("id int", nodes.ParseLoc(0)))
	})

	t.Run("column terminated by comma", func(t *testing.T) {
		t.Parallel()
		sql := "CREATE TABLE t (id int, name text)"
		loc := nodes.ParseLoc(strings.Index(sql, "id"))
		assert.Equal(t, "id int", extractColumnSQL(sql, loc))
	})

	t.Run("column terminated by closing paren", func(t *testing.T) {
		t.Parallel()
		sql := "CREATE TABLE t (id int)"
		loc := nodes.ParseLoc(strings.Index(sql, "id"))
		assert.Equal(t, "id int", extractColumnSQL(sql, loc))
	})

	t.Run("paren depth tracked so type parens do not split", func(t *testing.T) {
		t.Parallel()
		sql := "id varchar(255), name text"
		loc := nodes.ParseLoc(0) // 'i' of id
		assert.Equal(t, "id varchar(255)", extractColumnSQL(sql, loc))
	})

	t.Run("second column selected by its own offset", func(t *testing.T) {
		t.Parallel()
		sql := "id int, email varchar(255), created_at timestamptz"
		loc := nodes.ParseLoc(strings.Index(sql, "email"))
		assert.Equal(t, "email varchar(255)", extractColumnSQL(sql, loc))
	})
}

// =============================================================================
// loadCrossSchemaFKNames
// =============================================================================
//
// Contract source: declarative.go:857. Parses post-schema-fks.sql and returns a
// set of FK constraint names matched by `conname\s*=\s*'([^']+)'`. A missing
// file yields an empty (non-nil) map. The result is cached in the package var
// crossSchemaFKNames, so each test resets it before and after to stay isolated.

// NOTE: TestLoadCrossSchemaFKNames does NOT run in parallel. loadCrossSchemaFKNames
// caches its result in the package-level crossSchemaFKNames var; parallel sibling
// tests mutating the same var would race and make the cache assertions flaky.
func TestLoadCrossSchemaFKNames(t *testing.T) {
	// Ensure the package-level cache does not leak into or out of this test.
	t.Cleanup(func() { crossSchemaFKNames = nil })

	t.Run("missing file returns empty map", func(t *testing.T) {
		crossSchemaFKNames = nil
		s := newServiceWithDir(t, "/nonexistent/dir/for/test")
		got := s.loadCrossSchemaFKNames()
		assert.NotNil(t, got)
		assert.Empty(t, got)
	})

	t.Run("extracts each conname once", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "post-schema-fks.sql"),
			[]byte(strings.Join([]string{
				`DO $$ BEGIN`,
				`  IF NOT EXISTS (SELECT 1 WHERE conname = 'fk_orders_user') THEN`,
				`  END IF;`,
				`  IF NOT EXISTS (SELECT 1 WHERE conname = 'fk_items_order') THEN`,
				`  END IF;`,
			}, "\n")), 0o600))

		crossSchemaFKNames = nil
		s := newServiceWithDir(t, dir)
		got := s.loadCrossSchemaFKNames()
		assert.True(t, got["fk_orders_user"])
		assert.True(t, got["fk_items_order"])
		assert.Len(t, got, 2)
	})

	t.Run("file without matches returns empty map", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "post-schema-fks.sql"),
			[]byte("-- no constraints here\nSELECT 1;\n"), 0o600))

		crossSchemaFKNames = nil
		s := newServiceWithDir(t, dir)
		got := s.loadCrossSchemaFKNames()
		assert.Empty(t, got)
	})

	t.Run("subsequent call returns cached result", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "post-schema-fks.sql"),
			[]byte("WHERE conname = 'cached_fk'\n"), 0o600))

		crossSchemaFKNames = nil
		s := newServiceWithDir(t, dir)
		first := s.loadCrossSchemaFKNames()
		require.True(t, first["cached_fk"])

		// Remove the file; a cached second call must still return the prior name.
		require.NoError(t, os.Remove(filepath.Join(dir, "post-schema-fks.sql")))
		second := s.loadCrossSchemaFKNames()
		assert.True(t, second["cached_fk"], "should return cached names, not re-read disk")
	})
}

// =============================================================================
// filterManagedFKDrops
// =============================================================================
//
// Contract source: declarative.go:884. Removes DROP changes whose Name appears
// in the cross-schema FK set (those FKs are applied separately in Phase 2).

// NOTE: not parallel — filterManagedFKDrops reads the shared crossSchemaFKNames
// cache, which the serial TestLoadCrossSchemaFKNames also mutates.
func TestFilterManagedFKDrops(t *testing.T) {
	t.Cleanup(func() { crossSchemaFKNames = nil })

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "post-schema-fks.sql"),
		[]byte("WHERE conname = 'managed_fk'\n"), 0o600))
	crossSchemaFKNames = nil
	s := newServiceWithDir(t, dir)

	changes := []Change{
		{Type: ChangeCreate, Name: "t", SQL: "CREATE TABLE t (id int)"},
		{Type: ChangeDrop, Name: "managed_fk", SQL: "ALTER TABLE t DROP CONSTRAINT managed_fk"},
		{Type: ChangeDrop, Name: "unmanaged_fk", SQL: "ALTER TABLE t DROP CONSTRAINT unmanaged_fk"},
		{Type: ChangeAlter, Name: "managed_fk", SQL: "ALTER TABLE t ALTER COLUMN x TYPE int"},
	}
	got := s.filterManagedFKDrops(changes)
	// The managed DROP is filtered; the unmanaged DROP and the ALTER (even with a
	// managed name) are retained.
	require.Len(t, got, 3)
	assert.Equal(t, "CREATE TABLE t (id int)", got[0].SQL)
	assert.Equal(t, "unmanaged_fk", got[1].Name)
	assert.Equal(t, ChangeAlter, got[2].Type)
}

func TestFilterManagedFKDrops_Empty(t *testing.T) {
	t.Cleanup(func() { crossSchemaFKNames = nil })
	crossSchemaFKNames = nil
	s := newServiceWithDir(t, t.TempDir())
	// No managed names -> nothing filtered.
	in := []Change{{Type: ChangeDrop, Name: "anything", SQL: "x"}}
	got := s.filterManagedFKDrops(in)
	require.Len(t, got, 1)
}

// =============================================================================
// CalculateFingerprint
// =============================================================================
//
// Contract source: declarative.go:409. SHA-256 over the concatenation of each
// schema file's bytes plus a "|" separator, in config.Schemas order. Missing
// files are skipped silently; other read errors return an error.

func TestCalculateFingerprint(t *testing.T) {
	t.Parallel()

	writeSchemas := func(t *testing.T, dir string, files map[string]string) {
		t.Helper()
		for name, content := range files {
			require.NoError(t, os.WriteFile(filepath.Join(dir, name+".sql"), []byte(content), 0o600))
		}
	}

	t.Run("empty schema list yields sha256 of empty input", func(t *testing.T) {
		t.Parallel()
		s := newServiceWithSchemas(t, t.TempDir(), nil)
		got, err := s.CalculateFingerprint()
		require.NoError(t, err)
		// sha256 of "" (no schemas iterated, nothing written to the hasher)
		want := hex.EncodeToString(sha256.New().Sum(nil))
		assert.Equal(t, want, got)
	})

	t.Run("missing schema files are skipped, not errored", func(t *testing.T) {
		t.Parallel()
		// Two configured schemas but only one file present.
		dir := t.TempDir()
		writeSchemas(t, dir, map[string]string{"auth": "CREATE TABLE auth.users (id int);\n"})
		s := newServiceWithSchemas(t, dir, []string{"auth", "platform"})

		got, err := s.CalculateFingerprint()
		require.NoError(t, err)
		// Matches the fingerprint of just the present file (platform is skipped).
		h := sha256.New()
		h.Write([]byte("CREATE TABLE auth.users (id int);\n"))
		h.Write([]byte("|"))
		assert.Equal(t, hex.EncodeToString(h.Sum(nil)), got)
	})

	t.Run("deterministic for identical inputs", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		writeSchemas(t, dir, map[string]string{
			"platform": "CREATE SCHEMA platform;\n",
			"auth":     "CREATE TABLE auth.users (id int);\n",
		})
		s := newServiceWithSchemas(t, dir, []string{"platform", "auth"})

		first, err := s.CalculateFingerprint()
		require.NoError(t, err)
		second, err := s.CalculateFingerprint()
		require.NoError(t, err)
		assert.Equal(t, first, second)
	})

	t.Run("different content yields different fingerprint", func(t *testing.T) {
		t.Parallel()
		dirA := t.TempDir()
		dirB := t.TempDir()
		writeSchemas(t, dirA, map[string]string{"auth": "CREATE TABLE auth.users (id int);\n"})
		writeSchemas(t, dirB, map[string]string{"auth": "CREATE TABLE auth.users (id uuid);\n"})

		a := newServiceWithSchemas(t, dirA, []string{"auth"})
		b := newServiceWithSchemas(t, dirB, []string{"auth"})
		fa, err := a.CalculateFingerprint()
		require.NoError(t, err)
		fb, err := b.CalculateFingerprint()
		require.NoError(t, err)
		assert.NotEqual(t, fa, fb)
	})

	t.Run("schema order affects fingerprint", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		writeSchemas(t, dir, map[string]string{
			"platform": "A;\n",
			"auth":     "B;\n",
		})
		ab := newServiceWithSchemas(t, dir, []string{"platform", "auth"})
		ba := newServiceWithSchemas(t, dir, []string{"auth", "platform"})

		fab, err := ab.CalculateFingerprint()
		require.NoError(t, err)
		fba, err := ba.CalculateFingerprint()
		require.NoError(t, err)
		assert.NotEqual(t, fab, fba)
	})
}

// =============================================================================
// preprocessSchemaFile
// =============================================================================
//
// Contract source: declarative.go:84. When appUser is unset, returns the
// original path with a nil cleanup. When the file has no {{APP_USER}}
// placeholder, likewise. When it does, writes a temp file with the placeholder
// substituted and returns its path plus a cleanup func that removes it.

func TestPreprocessSchemaFile(t *testing.T) {
	t.Parallel()

	t.Run("unset app user returns original path and nil cleanup", func(t *testing.T) {
		t.Parallel()
		s := newServiceWithDir(t, t.TempDir())
		s.SetAppUser("")
		path, cleanup, err := s.preprocessSchemaFile("/some/path/auth.sql")
		require.NoError(t, err)
		assert.Equal(t, "/some/path/auth.sql", path)
		assert.Nil(t, cleanup)
	})

	t.Run("set app user but no placeholder returns original path", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		src := filepath.Join(dir, "auth.sql")
		require.NoError(t, os.WriteFile(src, []byte("CREATE TABLE auth.users (id int);\n"), 0o600))

		s := newServiceWithDir(t, dir)
		s.SetAppUser("fluxbase_app")
		path, cleanup, err := s.preprocessSchemaFile(src)
		require.NoError(t, err)
		assert.Equal(t, src, path)
		assert.Nil(t, cleanup)
	})

	t.Run("placeholder substituted into temp file and cleanup removes it", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		src := filepath.Join(dir, "auth.sql")
		require.NoError(t, os.WriteFile(src,
			[]byte("GRANT SELECT ON auth.users TO "+bootstrap.AppUserPlaceholder+";\n"), 0o600))

		s := newServiceWithDir(t, dir)
		s.SetAppUser("fluxbase_app")
		path, cleanup, err := s.preprocessSchemaFile(src)
		require.NoError(t, err)
		require.NotNil(t, cleanup)
		defer cleanup()

		assert.NotEqual(t, src, path)
		// Temp file must contain the substituted role, not the placeholder.
		got, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.Contains(t, string(got), "fluxbase_app")
		assert.NotContains(t, string(got), bootstrap.AppUserPlaceholder)
	})

	t.Run("invalid app user returns error", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		src := filepath.Join(dir, "auth.sql")
		require.NoError(t, os.WriteFile(src,
			[]byte("GRANT SELECT TO "+bootstrap.AppUserPlaceholder+";\n"), 0o600))

		s := newServiceWithDir(t, dir)
		s.SetAppUser("not a valid identifier!") // fails the pg-identifier check
		_, _, err := s.preprocessSchemaFile(src)
		require.Error(t, err)
	})

	t.Run("missing file returns read error", func(t *testing.T) {
		t.Parallel()
		s := newServiceWithDir(t, t.TempDir())
		s.SetAppUser("fluxbase_app")
		_, _, err := s.preprocessSchemaFile(filepath.Join(t.TempDir(), "absent.sql"))
		require.Error(t, err)
	})
}

// =============================================================================
// writePlanToTemp
// =============================================================================
//
// Contract source: declarative.go:429. Marshals the plan to a temp file and
// returns its path (caller is responsible for removal). Round-trips as valid
// JSON containing the plan's groups/changes.

func TestWritePlanToTemp(t *testing.T) {
	t.Parallel()
	s := newServiceWithDir(t, t.TempDir())

	plan := &Plan{
		Version: "1",
		Groups: []PlanGroup{{Steps: []PlanStep{
			{SQL: "CREATE TABLE t (id int)", Operation: "create", Path: "public.table.t"},
		}}},
		Changes: []Change{{Type: ChangeCreate, Schema: "public", Name: "t", SQL: "CREATE TABLE t (id int)"}},
	}

	path, err := s.writePlanToTemp(plan)
	require.NoError(t, err)
	defer func() { _ = os.Remove(path) }()

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	// The file must be valid JSON that round-trips to the same plan fields.
	var back Plan
	require.NoError(t, json.Unmarshal(data, &back))
	require.Len(t, back.Groups, 1)
	assert.Equal(t, "CREATE TABLE t (id int)", back.Groups[0].Steps[0].SQL)
	require.Len(t, back.Changes, 1)
	assert.Equal(t, "t", back.Changes[0].Name)
}

// =============================================================================
// Constructor + setters
// =============================================================================
//
// These were at 0% but are load-bearing wiring. A thin test locks in the field
// assignment contract so a future refactor can't silently drop a field.

func TestNewDeclarativeService_AndSetters(t *testing.T) {
	t.Parallel()
	s := NewDeclarativeService("/bin/pgschema", "localhost", 5432, "postgres", "pw", "fluxbase",
		DeclarativeConfig{SchemaDir: "/schemas", Schemas: []string{"auth"}, AllowDestructive: true, LockTimeout: 3})

	assert.Equal(t, "/bin/pgschema", s.pgschemaPath)
	assert.Equal(t, "localhost", s.dbHost)
	assert.Equal(t, 5432, s.dbPort)
	assert.Equal(t, "postgres", s.dbUser)
	assert.Equal(t, "fluxbase", s.dbName)
	assert.Equal(t, "/schemas", s.config.SchemaDir)
	assert.Equal(t, []string{"auth"}, s.config.Schemas)
	assert.True(t, s.config.AllowDestructive)
	assert.Equal(t, 3, s.config.LockTimeout)

	// Setters mutate the service in place.
	s.SetPool(nil) // exercises SetPool (nil is a valid no-op value)
	s.SetAppUser("fluxbase_app")
	assert.Equal(t, "fluxbase_app", s.appUser)
	assert.Nil(t, s.pool)
}

// =============================================================================
// test helpers
// =============================================================================

// newServiceWithDir builds a DeclarativeService whose SchemaDir is dir and
// Schemas is empty. Callers needing specific schemas use newServiceWithSchemas.
func newServiceWithDir(t *testing.T, dir string) *DeclarativeService {
	t.Helper()
	return NewDeclarativeService("", "", 0, "", "", "", DeclarativeConfig{SchemaDir: dir, Schemas: nil})
}

// newServiceWithSchemas builds a DeclarativeService with the given SchemaDir
// and configured schema list (used by fingerprint tests that care about order
// and file presence).
func newServiceWithSchemas(t *testing.T, dir string, schemas []string) *DeclarativeService {
	t.Helper()
	return NewDeclarativeService("", "", 0, "", "", "", DeclarativeConfig{SchemaDir: dir, Schemas: schemas})
}
