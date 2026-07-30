package schema

import (
	"strings"
	"testing"
)

func TestMakeSQLIdempotent_CreatePolicy(t *testing.T) {
	sql := `CREATE POLICY "my policy" ON public.users FOR SELECT USING (true);`
	out := MakeSQLIdempotent(sql)
	if !strings.Contains(out, `DROP POLICY IF EXISTS "my policy" ON "public"."users" CASCADE;`) {
		t.Errorf("expected DROP POLICY prepended, got:\n%s", out)
	}
	if !strings.Contains(out, `CREATE POLICY "my policy"`) {
		t.Errorf("expected original CREATE POLICY preserved, got:\n%s", out)
	}
	// DROP must come before CREATE.
	dropIdx := strings.Index(out, "DROP POLICY")
	createIdx := strings.Index(out, "CREATE POLICY")
	if dropIdx < 0 || createIdx < 0 || dropIdx > createIdx {
		t.Errorf("expected DROP before CREATE, got dropIdx=%d createIdx=%d", dropIdx, createIdx)
	}
}

func TestMakeSQLIdempotent_CreatePolicyUnquoted(t *testing.T) {
	sql := `CREATE POLICY my_policy ON users FOR SELECT USING (true);`
	out := MakeSQLIdempotent(sql)
	if !strings.Contains(out, `DROP POLICY IF EXISTS "my_policy"`) {
		t.Errorf("expected DROP POLICY for unquoted name, got:\n%s", out)
	}
}

func TestMakeSQLIdempotent_CreateTrigger(t *testing.T) {
	sql := `CREATE TRIGGER update_timestamp BEFORE UPDATE ON public.users FOR EACH ROW EXECUTE FUNCTION update_updated_at();`
	out := MakeSQLIdempotent(sql)
	if !strings.Contains(out, `DROP TRIGGER IF EXISTS "update_timestamp" ON "public"."users" CASCADE;`) {
		t.Errorf("expected DROP TRIGGER prepended, got:\n%s", out)
	}
	if !strings.Contains(out, `CREATE TRIGGER update_timestamp`) {
		t.Errorf("expected original CREATE TRIGGER preserved, got:\n%s", out)
	}
	// DROP must come before CREATE.
	dropIdx := strings.Index(out, "DROP TRIGGER")
	createIdx := strings.Index(out, "CREATE TRIGGER")
	if dropIdx < 0 || createIdx < 0 || dropIdx > createIdx {
		t.Errorf("expected DROP before CREATE, got dropIdx=%d createIdx=%d", dropIdx, createIdx)
	}
}

func TestMakeSQLIdempotent_CreateTriggerQuoted(t *testing.T) {
	sql := `CREATE TRIGGER "my trigger" AFTER INSERT ON users FOR EACH ROW EXECUTE FUNCTION foo();`
	out := MakeSQLIdempotent(sql)
	if !strings.Contains(out, `DROP TRIGGER IF EXISTS "my trigger"`) {
		t.Errorf("expected DROP TRIGGER for quoted name, got:\n%s", out)
	}
}

func TestMakeSQLIdempotent_CreateIndex(t *testing.T) {
	sql := `CREATE INDEX idx_users_email ON public.users (email);`
	out := MakeSQLIdempotent(sql)
	if !strings.Contains(out, `DROP INDEX IF EXISTS "idx_users_email";`) {
		t.Errorf("expected DROP INDEX prepended, got:\n%s", out)
	}
	if !strings.Contains(out, `CREATE INDEX idx_users_email`) {
		t.Errorf("expected original CREATE INDEX preserved, got:\n%s", out)
	}
	dropIdx := strings.Index(out, "DROP INDEX")
	createIdx := strings.Index(out, "CREATE INDEX")
	if dropIdx < 0 || createIdx < 0 || dropIdx > createIdx {
		t.Errorf("expected DROP before CREATE, got dropIdx=%d createIdx=%d", dropIdx, createIdx)
	}
}

func TestMakeSQLIdempotent_CreateIndexIfNotExistsPassthrough(t *testing.T) {
	// CREATE INDEX IF NOT EXISTS is already idempotent (no-op if exists); must NOT
	// get a DROP prepended (that would defeat IF NOT EXISTS and could drop a
	// concurrently-built index).
	sql := `CREATE INDEX IF NOT EXISTS idx_users_email ON public.users (email);`
	out := MakeSQLIdempotent(sql)
	if strings.Contains(out, "DROP INDEX") {
		t.Errorf("IF NOT EXISTS index must not get DROP prepended, got:\n%s", out)
	}
}

func TestMakeSQLIdempotent_CreateIndexConcurrentlyPassthrough(t *testing.T) {
	// CONCURRENTLY cannot run in a transaction and must not be dropped+recreated.
	sql := `CREATE INDEX CONCURRENTLY idx_users_email ON public.users (email);`
	out := MakeSQLIdempotent(sql)
	if strings.Contains(out, "DROP INDEX") {
		t.Errorf("CONCURRENTLY index must not get DROP prepended, got:\n%s", out)
	}
}

func TestMakeSQLIdempotent_AlterTableAddConstraint(t *testing.T) {
	// Note: the constraint-handler pattern matches the unqualified relname, so
	// schema-qualified ALTER TABLE statements are not transformed (pre-existing
	// behavior). Test with the unqualified form that the handler supports.
	sql := `ALTER TABLE users ADD CONSTRAINT users_email_key UNIQUE (email);`
	out := MakeSQLIdempotent(sql)
	if !strings.Contains(out, `DROP CONSTRAINT IF EXISTS "users_email_key";`) {
		t.Errorf("expected DROP CONSTRAINT prepended, got:\n%s", out)
	}
}

func TestMakeSQLIdempotent_NoTransformations(t *testing.T) {
	// Plain DDL with no policies/triggers/indexes/constraints is unchanged.
	sql := `CREATE TABLE IF NOT EXISTS users (id uuid PRIMARY KEY, name text);`
	out := MakeSQLIdempotent(sql)
	if out != sql {
		t.Errorf("expected no transformation, got:\n%s", out)
	}
}

func TestMakeSQLIdempotent_ParseFailureReturnsOriginal(t *testing.T) {
	// Unparseable SQL should return the original unchanged (not error/panic).
	sql := `THIS IS NOT SQL;;;`
	out := MakeSQLIdempotent(sql)
	if out != sql {
		t.Errorf("expected original on parse failure, got:\n%s", out)
	}
}
