package migrations

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Unit tests for the pure planning/transform helpers in declarative.go.
//
// These functions turn pgschema's grouped plan output into the flat []Change
// slice the package reasons about, reorder it for safe execution, and extract
// table names from raw SQL for cross-reference. They are deterministic and
// DB-free — the rest of declarative.go shells out to pgschema or runs SQL.
//
// Convention matches app_declarative_test.go: testify, package migrations
// (white-box so unexported funcs are reachable).

// =============================================================================
// extractChangesFromGroups Tests
// =============================================================================
//
// Contract source: declarative.go:184 + the inline Path-format comment:
//   "schema.object_type.name" or "schema.object_type.constraint_name"
// step.Operation is one of "create"/"drop"/"alter". Steps with empty SQL are
// skipped (directive-only steps like "wait for index").

func TestExtractChangesFromGroups_NilPlan(t *testing.T) {
	t.Parallel()
	// A nil/empty plan yields no changes (and must not panic on nil Groups).
	got := extractChangesFromGroups(&Plan{})
	assert.Empty(t, got)
}

func testStep(sql, op, path string) PlanStep {
	return PlanStep{SQL: sql, Operation: op, Path: path}
}

func TestExtractChangesFromGroups_PathParsing(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		step        PlanStep
		wantSchema  string
		wantObjType string
		wantName    string
	}{
		{
			name:        "full 3-part path",
			step:        testStep("CREATE TABLE users (id int)", "create", "public.table.users"),
			wantSchema:  "public",
			wantObjType: "table",
			wantName:    "users",
		},
		{
			name:        "4-part path joins tail into name (constraint)",
			step:        testStep("ALTER TABLE orders ADD CONSTRAINT fk_x ...", "alter", "public.table.orders.fk_x"),
			wantSchema:  "public",
			wantObjType: "table",
			wantName:    "orders.fk_x",
		},
		{
			name:        "2-part path has empty name",
			step:        testStep("CREATE SCHEMA app", "create", "app.schema"),
			wantSchema:  "app",
			wantObjType: "schema",
			wantName:    "",
		},
		{
			name:        "1-part path has empty object type and name",
			step:        testStep("DO $$ BEGIN END $$", "create", "public"),
			wantSchema:  "public",
			wantObjType: "",
			wantName:    "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			plan := &Plan{Groups: []PlanGroup{{Steps: []PlanStep{tt.step}}}}
			got := extractChangesFromGroups(plan)
			require.Len(t, got, 1)
			c := got[0]
			assert.Equal(t, tt.wantSchema, c.Schema)
			assert.Equal(t, tt.wantObjType, c.ObjectType)
			assert.Equal(t, tt.wantName, c.Name)
			assert.Equal(t, tt.step.SQL, c.SQL)
		})
	}
}

func TestExtractChangesFromGroups_OperationToChangeType(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name            string
		op              string
		wantType        ChangeType
		wantDestructive bool
	}{
		{"create", "create", ChangeCreate, false},
		{"alter", "alter", ChangeAlter, false},
		{"drop marks destructive", "drop", ChangeDrop, true},
		{"unknown op leaves zero-value type", "rename", ChangeType(""), false},
		{"empty op leaves zero-value type", "", ChangeType(""), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			plan := &Plan{Groups: []PlanGroup{{Steps: []PlanStep{
				testStep("SOME SQL", tt.op, "public.table.t"),
			}}}}
			got := extractChangesFromGroups(plan)
			require.Len(t, got, 1)
			assert.Equal(t, tt.wantType, got[0].Type)
			assert.Equal(t, tt.wantDestructive, got[0].Destructive)
		})
	}
}

func TestExtractChangesFromGroups_SkipsDirectiveOnlySteps(t *testing.T) {
	t.Parallel()
	// Steps with empty SQL are skipped (directive-only, e.g. "wait for index").
	plan := &Plan{Groups: []PlanGroup{{Steps: []PlanStep{
		{SQL: "", Operation: "create", Path: "public.index.idx", Directive: &PlanDirective{Type: "wait"}},
		testStep("CREATE INDEX idx ON t (c)", "create", "public.index.idx"),
	}}}}
	got := extractChangesFromGroups(plan)
	require.Len(t, got, 1)
	assert.Equal(t, "CREATE INDEX idx ON t (c)", got[0].SQL)
}

func TestExtractChangesFromGroups_MultipleGroupsConcatenatedInOrder(t *testing.T) {
	t.Parallel()
	plan := &Plan{Groups: []PlanGroup{
		{Steps: []PlanStep{testStep("A", "create", "public.table.a")}},
		{Steps: []PlanStep{testStep("B", "create", "public.table.b"),
			testStep("C", "create", "public.table.c")}},
	}}
	got := extractChangesFromGroups(plan)
	require.Len(t, got, 3)
	assert.Equal(t, "A", got[0].SQL)
	assert.Equal(t, "B", got[1].SQL)
	assert.Equal(t, "C", got[2].SQL)
}

// =============================================================================
// changePriority Tests
// =============================================================================
//
// Contract source: declarative.go:1163. Priority buckets drive the stable sort
// in applyPlanDirectly: tables (0) before indexes/sequences (1) before
// misc/functions/triggers/grants (2) before policies (3). SQL prefix is matched
// case-insensitively after TrimSpace.

func TestChangePriority(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		sql  string
		want int
	}{
		{"create table", "CREATE TABLE users (id int)", 0},
		{"alter table", "ALTER TABLE users ADD COLUMN x int", 0},
		{"lowercase alter table", "  alter table users add column x int", 0},
		{"create index", "CREATE INDEX idx ON users (id)", 1},
		{"create unique index", "CREATE UNIQUE INDEX idx ON users (id)", 1},
		{"drop index", "DROP INDEX idx", 1},
		{"create sequence", "CREATE SEQUENCE seq", 1},
		{"create policy", "CREATE POLICY p ON users USING (true)", 3},
		{"alter policy", "ALTER POLICY p ON users USING (true)", 3},
		{"drop policy", "DROP POLICY p ON users", 3},
		{"create function", "CREATE FUNCTION f() RETURNS void AS $$ $$ LANGUAGE sql", 2},
		{"create trigger", "CREATE TRIGGER tr AFTER INSERT ON users", 2},
		{"grant", "GRANT SELECT ON users TO anon", 2},
		{"empty sql defaults to 2", "", 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, changePriority(Change{SQL: tt.sql}))
		})
	}
}

// TestChangePriority_SortOrder is a property-style test: sorting a mixed slice
// by changePriority puts all 0s before 1s before 2s before 3s (this is how
// applyPlanDirectly uses it via sort.SliceStable).
func TestChangePriority_SortOrder(t *testing.T) {
	t.Parallel()
	changes := []Change{
		{SQL: "CREATE POLICY p ON users"},
		{SQL: "CREATE TABLE users (id int)"},
		{SQL: "GRANT SELECT ON users"},
		{SQL: "CREATE INDEX idx ON users (id)"},
		{SQL: "ALTER TABLE users ADD COLUMN x int"},
	}
	sort.SliceStable(changes, func(i, j int) bool {
		return changePriority(changes[i]) < changePriority(changes[j])
	})
	// Expect order: tables, indexes, grants, policy.
	assert.Equal(t, "CREATE TABLE users (id int)", changes[0].SQL)
	assert.Equal(t, "ALTER TABLE users ADD COLUMN x int", changes[1].SQL)
	assert.Equal(t, "CREATE INDEX idx ON users (id)", changes[2].SQL)
	assert.Equal(t, "GRANT SELECT ON users", changes[3].SQL)
	assert.Equal(t, "CREATE POLICY p ON users", changes[4].SQL)
}

// =============================================================================
// hasDestructiveChanges Tests
// =============================================================================
//
// Contract source: declarative.go:842. Returns true iff any Change.Destructive.

func TestHasDestructiveChanges(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		changes []Change
		want    bool
	}{
		{"nil slice", nil, false},
		{"empty slice", []Change{}, false},
		{"none destructive", []Change{{Destructive: false}, {Destructive: false}}, false},
		{"one destructive", []Change{{Destructive: false}, {Destructive: true}}, true},
		{"all destructive", []Change{{Destructive: true}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, hasDestructiveChanges(tt.changes))
		})
	}
}

// =============================================================================
// SQL table-name extractors
// =============================================================================
//
// Contract: each extractor pulls the bare table name from a specific statement
// shape, stripping schema prefixes, quotes, and (for index/trigger) trailing
// semicolons. Non-matching statements return "".

func TestExtractAlterTableName(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		sql  string
		want string
	}{
		{"simple", "ALTER TABLE users ADD COLUMN x int", "users"},
		{"if exists", "ALTER TABLE IF EXISTS users ADD COLUMN x int", "users"},
		{"schema qualified", "ALTER TABLE platform.users ADD COLUMN x int", "users"},
		// KNOWN BUG (pinned, not endorsed): extractAlterTableName tokenizes with
		// strings.Fields BEFORE stripping quotes, so a quoted identifier that
		// contains whitespace ("User Table") is split into ["User", "Table"] and
		// only "User" is returned. The sensible contract is to return the full
		// quoted identifier. Impact: the partition-table skip logic in
		// applyPlanDirectly (declarative.go:669,699,710) silently fails to match
		// such tables, so ALTER TABLE on a quoted partition table won't be
		// skipped and can hit the SQLSTATE 42P16 the skip is meant to avoid.
		// Fix: tokenize respecting double-quoted identifiers. Tracked for a
		// follow-up; out of scope for this coverage PR.
		{"quoted multi-word name (BUG: mangled)", `ALTER TABLE "User Table" ADD COLUMN x int`, "User"},
		{"quoted single-word name works", `ALTER TABLE "users" ADD COLUMN x int`, "users"},
		{"lowercase", "alter table users add column x int", "users"},
		{"not an alter table", "CREATE TABLE users (id int)", ""},
		{"empty", "", ""},
		{"trailing semicolon kept by this extractor", "ALTER TABLE users;", "users;"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// NOTE the "trailing semicolon kept" case pins that extractAlterTableName
			// does NOT strip ';', unlike extractIndexTableName/extractDropTriggerTableName.
			// Per testing discipline: this is a pin of current behavior; if it proves
			// to be a bug (callers depend on clean names), flag and fix the code.
			assert.Equal(t, tt.want, extractAlterTableName(tt.sql))
		})
	}
}

func TestExtractIndexTableName(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		sql  string
		want string
	}{
		{"create index", "CREATE INDEX idx ON users (id)", "users"},
		{"create unique index", "CREATE UNIQUE INDEX idx ON users (id)", "users"},
		{"concurrently if not exists", "CREATE INDEX CONCURRENTLY IF NOT EXISTS idx ON users (id)", "users"},
		{"schema qualified", "CREATE INDEX idx ON platform.users (id)", "users"},
		// KNOWN BUG (pinned): same quoted-identifier tokenization issue as
		// extractAlterTableName — see that test's note. extractIndexTableName is
		// used for the partitionedTables lookup at declarative.go:727, so a
		// quoted multi-word partitioned table would not have CONCURRENTLY
		// stripped. Sensible contract: return "User Table".
		{"quoted multi-word (BUG: mangled)", `CREATE INDEX idx ON "User Table" (id)`, "User"},
		{"quoted single-word works", `CREATE INDEX idx ON "users" (id)`, "users"},
		{"trailing semicolon stripped", "CREATE INDEX idx ON users (id);", "users"},
		{"no ON clause", "CREATE INDEX idx", ""},
		{"not a create index", "ALTER TABLE users ADD COLUMN x", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, extractIndexTableName(tt.sql))
		})
	}
}

func TestExtractTriggerTableName(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		sql  string
		want string
	}{
		{"create trigger", "CREATE TRIGGER tr AFTER INSERT ON users FOR EACH ROW EXECUTE FUNCTION f()", "users"},
		{"create or replace", "CREATE OR REPLACE TRIGGER tr AFTER INSERT ON users EXECUTE FUNCTION f()", "users"},
		{"schema qualified", "CREATE TRIGGER tr AFTER INSERT ON platform.users EXECUTE FUNCTION f()", "users"},
		{"no ON clause", "CREATE TRIGGER tr AFTER INSERT", ""},
		{"not a trigger", "CREATE TABLE users (id int)", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, extractTriggerTableName(tt.sql))
		})
	}
}

func TestExtractDropTriggerTableName(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		sql  string
		want string
	}{
		{"drop trigger", "DROP TRIGGER IF EXISTS tr ON users", "users"},
		{"schema qualified", "DROP TRIGGER tr ON platform.users", "users"},
		{"trailing semicolon stripped", "DROP TRIGGER tr ON users;", "users"},
		{"no ON clause", "DROP TRIGGER tr", ""},
		{"not a drop trigger", "CREATE TABLE users (id int)", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, extractDropTriggerTableName(tt.sql))
		})
	}
}
