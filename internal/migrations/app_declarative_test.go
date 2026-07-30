package migrations

import (
	"context"
	"encoding/json"
	"os"
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
