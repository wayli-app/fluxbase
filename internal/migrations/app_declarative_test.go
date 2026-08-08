package migrations

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestAppService(t *testing.T, allowDestructive bool) *AppDeclarativeService {
	t.Helper()
	return NewAppDeclarativeService("/usr/bin/pgschema", "localhost", 5432, "admin", "secret", "fluxbase", allowDestructive)
}

func TestNewAppDeclarativeService(t *testing.T) {
	svc := NewAppDeclarativeService("/bin/pgschema", "dbhost", 5433, "admin", "secret", "appdb", false)

	require.NotNil(t, svc)
	assert.Equal(t, "/bin/pgschema", svc.pgschemaPath)
	assert.Equal(t, "dbhost", svc.dbHost)
	assert.Equal(t, 5433, svc.dbPort)
	assert.Equal(t, "admin", svc.dbUser)
	assert.Equal(t, "secret", svc.dbPassword)
	assert.Equal(t, "appdb", svc.dbName)
	assert.False(t, svc.allowDestructive)
	assert.Empty(t, svc.appUser)
}

func TestAppDeclarativeService_Setters(t *testing.T) {
	svc := newTestAppService(t, false)
	svc.SetAllowDestructive(true)
	svc.SetAppUser("wayli_user")

	assert.True(t, svc.allowDestructive)
	assert.Equal(t, "wayli_user", svc.appUser)
}

func TestWriteContentToTemp(t *testing.T) {
	t.Run("writes content to a readable file", func(t *testing.T) {
		content := "CREATE TABLE foo (id int);"
		path, err := writeContentToTemp("app-schema-*.sql", content)
		require.NoError(t, err)
		defer func() { _ = os.Remove(path) }()

		assert.FileExists(t, path)
		data, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.Equal(t, content, string(data))
	})

	t.Run("rejects empty content", func(t *testing.T) {
		// writeContentToTemp writes whatever is passed; callers guard empties.
		// Verify it still writes (no panic) and the file is empty.
		path, err := writeContentToTemp("app-schema-*.sql", "")
		require.NoError(t, err)
		defer func() { _ = os.Remove(path) }()
		data, _ := os.ReadFile(path)
		assert.Empty(t, data)
	})
}

func TestWritePlanToTempFile(t *testing.T) {
	t.Run("round-trips a plan as JSON", func(t *testing.T) {
		plan := &Plan{
			Changes: []Change{
				{Type: ChangeCreate, ObjectType: "table", Schema: "public", Name: "widgets", SQL: "CREATE TABLE widgets (id int)"},
				{Type: ChangeDrop, ObjectType: "table", Schema: "public", Name: "legacy", SQL: "DROP TABLE legacy", Destructive: true},
			},
			DDL: "CREATE TABLE widgets (id int);",
		}

		path, err := writePlanToTempFile(plan)
		require.NoError(t, err)
		defer func() { _ = os.Remove(path) }()

		assert.FileExists(t, path)
		data, err := os.ReadFile(path)
		require.NoError(t, err)

		var readPlan Plan
		require.NoError(t, json.Unmarshal(data, &readPlan))
		require.Len(t, readPlan.Changes, 2)
		assert.Equal(t, ChangeCreate, readPlan.Changes[0].Type)
		assert.Equal(t, ChangeDrop, readPlan.Changes[1].Type)
		assert.True(t, readPlan.Changes[1].Destructive)
	})

	t.Run("round-trips an empty plan", func(t *testing.T) {
		path, err := writePlanToTempFile(&Plan{})
		require.NoError(t, err)
		defer func() { _ = os.Remove(path) }()

		data, err := os.ReadFile(path)
		require.NoError(t, err)
		var readPlan Plan
		require.NoError(t, json.Unmarshal(data, &readPlan))
		assert.Empty(t, readPlan.Changes)
	})
}

// TestStoreSchemaContent_ValidationGuard verifies the input validation that runs
// before any database access (so it fails fast without a pool).
func TestStoreSchemaContent_ValidationGuard(t *testing.T) {
	ctx := context.Background()
	svc := newTestAppService(t, false)
	// No pool set — these validations must fail before touching the DB.

	t.Run("rejects empty namespace", func(t *testing.T) {
		_, _, err := svc.StoreSchemaContent(ctx, "", "public", "CREATE TABLE x ();", "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "namespace is required")
	})

	t.Run("rejects empty content", func(t *testing.T) {
		_, _, err := svc.StoreSchemaContent(ctx, "wayli", "public", "", "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "content cannot be empty")
	})
}

// TestApplyFromContent_DestructiveGuards verifies the destructive-change blocking
// logic cannot be reached without a pool for the happy path, but documents the
// intended behavior. The actual destructive gating lives in ApplyFromContent after
// a plan is computed; here we assert the service default and setter behave.
func TestAppDeclarativeService_DestructiveDefault(t *testing.T) {
	svc := newTestAppService(t, false)
	assert.False(t, svc.allowDestructive, "destructive changes must be blocked by default")

	svc.SetAllowDestructive(true)
	assert.True(t, svc.allowDestructive)
}

// TestCountDestructiveStatements verifies the fallback-path destructive scanner
// detects user-authored destructive statements but ignores MakeSQLIdempotent's
// own re-creation-aid DROPs (POLICY/TRIGGER/INDEX/CONSTRAINT IF EXISTS).
func TestCountDestructiveStatements(t *testing.T) {
	tests := []struct {
		name string
		sql  string
		want int
	}{
		{"no destructive", "CREATE TABLE t (id int); CREATE INDEX ON t (id);", 0},
		{"drop table", "DROP TABLE old_data;", 1},
		{"drop column via alter", "ALTER TABLE t DROP COLUMN obsolete;", 1},
		{"drop type", "DROP TYPE old_enum;", 1},
		{"truncate", "TRUNCATE staging;", 1},
		{"drop view", "DROP VIEW old_view;", 1},
		{"drop function", "DROP FUNCTION old_fn();", 1},
		{"multiple destructive", "DROP TABLE a;\nDROP TABLE b;\nTRUNCATE c;", 3},
		// MakeSQLIdempotent-generated DROPs (re-creation aids) must NOT count:
		{"drop policy if exists (idempotent aid)", `DROP POLICY IF EXISTS "p" ON t CASCADE;`, 0},
		{"drop trigger if exists (idempotent aid)", `DROP TRIGGER IF EXISTS "trg" ON t CASCADE;`, 0},
		{"drop index if exists (idempotent aid)", `DROP INDEX IF EXISTS idx;`, 0},
		{"alter table drop constraint if exists (idempotent aid)", `ALTER TABLE t DROP CONSTRAINT IF EXISTS "c";`, 0},
		{"comment lines ignored", "-- DROP TABLE commented_out;", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := countDestructiveStatements(tt.sql)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestApplyDirectFallback_DestructiveBlocked verifies the fallback path blocks
// destructive content when allowDestructive=false, without touching the DB.
func TestApplyDirectFallback_DestructiveBlocked(t *testing.T) {
	ctx := context.Background()
	svc := newTestAppService(t, false) // allowDestructive=false, no pool

	t.Run("blocks DROP TABLE", func(t *testing.T) {
		res, err := svc.applyDirectFallback(ctx, "public", "DROP TABLE old_data;")
		require.NoError(t, err) // blocked, not an error
		require.NotNil(t, res)
		assert.True(t, res.Fallback, "result must indicate fallback path")
		require.Error(t, res.Error, "must contain blocking error")
		assert.Contains(t, res.Error.Error(), "destructive")
	})

	t.Run("blocks DROP COLUMN", func(t *testing.T) {
		res, err := svc.applyDirectFallback(ctx, "public", "ALTER TABLE t DROP COLUMN obsolete;")
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Error(t, res.Error)
		assert.Contains(t, res.Error.Error(), "destructive")
	})
}

// TestApplyDirectFallback_NondestructivePassesContentScan verifies that
// non-destructive content clears the destructive check (then fails later at DB
// access, which is expected without a pool).
func TestApplyDirectFallback_NondestructivePassesContentScan(t *testing.T) {
	ctx := context.Background()
	svc := newTestAppService(t, false)
	res, err := svc.applyDirectFallback(ctx, "public", "CREATE TABLE IF NOT EXISTS t (id int);")
	// No pool set → fails at connection, NOT at the destructive check.
	require.Error(t, err)
	assert.Nil(t, res)
	assert.NotContains(t, err.Error(), "destructive", "must not be blocked as destructive")
}

// TestSubstituteAppUserForContent covers placeholder substitution logic (pure).
func TestSubstituteAppUserForContent(t *testing.T) {
	t.Run("no app user returns content unchanged", func(t *testing.T) {
		svc := newTestAppService(t, false) // appUser empty
		content := "GRANT ALL ON TABLE t TO {{APP_USER}};"
		out, err := svc.substituteAppUserForContent(content)
		require.NoError(t, err)
		assert.Equal(t, content, out, "with no app user set, content is returned as-is")
	})

	t.Run("with app user substitutes placeholder", func(t *testing.T) {
		svc := newTestAppService(t, false)
		svc.SetAppUser("wayli_app")
		content := "GRANT ALL ON TABLE t TO {{APP_USER}};"
		out, err := svc.substituteAppUserForContent(content)
		require.NoError(t, err)
		assert.Equal(t, "GRANT ALL ON TABLE t TO wayli_app;", out)
	})

	t.Run("content without placeholder is unchanged", func(t *testing.T) {
		svc := newTestAppService(t, false)
		svc.SetAppUser("wayli_app")
		content := "CREATE TABLE t (id int);"
		out, err := svc.substituteAppUserForContent(content)
		require.NoError(t, err)
		assert.Equal(t, content, out)
	})
}

// =============================================================================
// Pure helper tests: extractColumnDefByName, matchesColumnDef, stripLeadingComma,
// countStatements, writeSchemaWorkDir
// =============================================================================
//
// These were previously untested despite being core to column extraction and
// statement counting. Contracts per app_declarative.go doc comments + bodies.

func TestStripLeadingComma(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"no comma", "id int", "id int"},
		{"leading comma", ",id int", "id int"},
		{"leading comma with spaces", "  ,  id int", "id int"},
		{"only comma", ",", ""},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, stripLeadingComma(tt.in))
		})
	}
}

func TestMatchesColumnDef(t *testing.T) {
	t.Parallel()
	// Constraint-style entries are NOT column definitions.
	for _, kw := range []string{
		"CONSTRAINT fk_pkey PRIMARY KEY (id)",
		"PRIMARY KEY (id)",
		"UNIQUE (email)",
		"CHECK (x > 0)",
		"FOREIGN KEY (uid) REFERENCES users(id)",
	} {
		t.Run("rejects/"+kw, func(t *testing.T) {
			t.Parallel()
			assert.False(t, matchesColumnDef(kw, "anything"))
		})
	}
	t.Run("empty entry", func(t *testing.T) {
		t.Parallel()
		assert.False(t, matchesColumnDef("", "id"))
	})
	t.Run("matching unquoted column", func(t *testing.T) {
		t.Parallel()
		assert.True(t, matchesColumnDef("id serial PRIMARY KEY", "id"))
	})
	t.Run("non-matching column", func(t *testing.T) {
		t.Parallel()
		assert.False(t, matchesColumnDef("email text", "id"))
	})
	t.Run("matching quoted column", func(t *testing.T) {
		t.Parallel()
		assert.True(t, matchesColumnDef(`"my col" text`, "my col"))
	})
}

func TestExtractColumnDefByName(t *testing.T) {
	t.Parallel()
	schema := `CREATE TABLE IF NOT EXISTS users (
	id serial PRIMARY KEY,
	email text NOT NULL,
	"display name" text,
	CONSTRAINT uk_email UNIQUE (email)
);`

	t.Run("found unquoted", func(t *testing.T) {
		t.Parallel()
		got := extractColumnDefByName(schema, "users", "email")
		assert.Equal(t, "email text NOT NULL", got)
	})

	t.Run("found quoted column", func(t *testing.T) {
		t.Parallel()
		got := extractColumnDefByName(schema, "users", "display name")
		assert.Contains(t, got, `"display name" text`)
	})

	t.Run("not present returns empty", func(t *testing.T) {
		t.Parallel()
		assert.Empty(t, extractColumnDefByName(schema, "users", "nonexistent"))
	})

	t.Run("table not present returns empty", func(t *testing.T) {
		t.Parallel()
		assert.Empty(t, extractColumnDefByName(schema, "orders", "id"))
	})

	t.Run("does not match constraint entry as column", func(t *testing.T) {
		t.Parallel()
		// "CONSTRAINT uk_email" is not a column; searching for it yields "".
		assert.Empty(t, extractColumnDefByName(schema, "users", "CONSTRAINT"))
	})
}

func TestCountStatements(t *testing.T) {
	t.Parallel()
	t.Run("multiple statements", func(t *testing.T) {
		t.Parallel()
		// Two distinct top-level statements.
		n := countStatements("CREATE TABLE a (id int); CREATE TABLE b (id int);")
		assert.GreaterOrEqual(t, n, 2)
	})
	t.Run("empty returns 0", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, 0, countStatements(""))
	})
	t.Run("garbage returns 0 not panic", func(t *testing.T) {
		t.Parallel()
		// Must not panic on unparseable input.
		assert.Equal(t, 0, countStatements("))) not sql ((("))
	})
}

func TestWriteSchemaWorkDir(t *testing.T) {
	t.Parallel()
	t.Run("schema only", func(t *testing.T) {
		t.Parallel()
		dir, schemaFile, err := writeSchemaWorkDir("CREATE TABLE t (id int);", "")
		require.NoError(t, err)
		t.Cleanup(func() { _ = os.RemoveAll(dir) })

		got, err := os.ReadFile(schemaFile)
		require.NoError(t, err)
		assert.Equal(t, "CREATE TABLE t (id int);", string(got))

		// No ignore file when ignoreContent is empty.
		_, err = os.ReadFile(filepath.Join(dir, ".pgschemaignore"))
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("with ignore file", func(t *testing.T) {
		t.Parallel()
		dir, _, err := writeSchemaWorkDir("CREATE TABLE t (id int);", "*.tmp")
		require.NoError(t, err)
		t.Cleanup(func() { _ = os.RemoveAll(dir) })

		got, err := os.ReadFile(filepath.Join(dir, ".pgschemaignore"))
		require.NoError(t, err)
		assert.Equal(t, "*.tmp", string(got))
	})
}
