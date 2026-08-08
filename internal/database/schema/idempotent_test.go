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

func TestMakeSQLIdempotent_CreateOrReplaceView(t *testing.T) {
	sql := `CREATE OR REPLACE VIEW my_tracker_data WITH (security_barrier=true) AS SELECT id, name FROM tracker_data WHERE user_id = 1;`
	out := MakeSQLIdempotent(sql)
	if !strings.Contains(out, `DROP VIEW IF EXISTS "my_tracker_data" CASCADE;`) {
		t.Errorf("expected DROP VIEW prepended, got:\n%s", out)
	}
	if !strings.Contains(out, `CREATE OR REPLACE VIEW my_tracker_data`) {
		t.Errorf("expected original CREATE VIEW preserved, got:\n%s", out)
	}
	dropIdx := strings.Index(out, "DROP VIEW")
	createIdx := strings.Index(out, "CREATE OR REPLACE VIEW")
	if dropIdx < 0 || createIdx < 0 || dropIdx > createIdx {
		t.Errorf("expected DROP before CREATE, got dropIdx=%d createIdx=%d", dropIdx, createIdx)
	}
}

func TestMakeSQLIdempotent_CreateOrReplaceViewQuoted(t *testing.T) {
	sql := `CREATE OR REPLACE VIEW "My View" AS SELECT 1;`
	out := MakeSQLIdempotent(sql)
	if !strings.Contains(out, `DROP VIEW IF EXISTS "My View" CASCADE;`) {
		t.Errorf("expected DROP VIEW for quoted name, got:\n%s", out)
	}
}

func TestMakeSQLIdempotent_CreateOrReplaceFunction(t *testing.T) {
	// CREATE OR REPLACE FUNCTION cannot change its return type (SQLSTATE 42P13).
	// Regression test: a DROP FUNCTION IF EXISTS with the arg-type signature must
	// be prepended so the function is recreated fresh.
	sql := `CREATE OR REPLACE FUNCTION public.get_public_trip_track(trip_uuid uuid) RETURNS TABLE(lat double precision, lng double precision, recorded_at timestamptz) LANGUAGE plpgsql AS $$ BEGIN END; $$;`
	out := MakeSQLIdempotent(sql)
	if !strings.Contains(out, `DROP FUNCTION IF EXISTS "public"."get_public_trip_track"(uuid) CASCADE;`) {
		t.Errorf("expected DROP FUNCTION with signature prepended, got:\n%s", out)
	}
	if !strings.Contains(out, `CREATE OR REPLACE FUNCTION public.get_public_trip_track`) {
		t.Errorf("expected original CREATE FUNCTION preserved, got:\n%s", out)
	}
	dropIdx := strings.Index(out, "DROP FUNCTION")
	createIdx := strings.Index(out, "CREATE OR REPLACE FUNCTION")
	if dropIdx < 0 || createIdx < 0 || dropIdx > createIdx {
		t.Errorf("expected DROP before CREATE, got dropIdx=%d createIdx=%d", dropIdx, createIdx)
	}
}

func TestMakeSQLIdempotent_CreateFunctionMultipleArgs(t *testing.T) {
	// Multiple arguments of different types: the signature must list them all.
	// Note: pgparser normalizes "double precision" to its canonical
	// "pg_catalog.float8" form; the DROP uses whatever the parser emits.
	sql := `CREATE FUNCTION public.calc_distance(lat double precision, lng double precision, user_uuid uuid) RETURNS integer LANGUAGE sql AS $$ SELECT 1; $$;`
	out := MakeSQLIdempotent(sql)
	if !strings.Contains(out, `DROP FUNCTION IF EXISTS "public"."calc_distance"(`) {
		t.Errorf("expected DROP FUNCTION with arg signature prepended, got:\n%s", out)
	}
	if !strings.Contains(out, `uuid) CASCADE;`) {
		t.Errorf("expected uuid as last arg in DROP signature, got:\n%s", out)
	}
}

func TestMakeSQLIdempotent_CreateFunctionNoArgs(t *testing.T) {
	sql := `CREATE FUNCTION public.now_utc() RETURNS timestamptz LANGUAGE sql AS $$ SELECT now(); $$;`
	out := MakeSQLIdempotent(sql)
	if !strings.Contains(out, `DROP FUNCTION IF EXISTS "public"."now_utc"() CASCADE;`) {
		t.Errorf("expected DROP FUNCTION with empty arg list, got:\n%s", out)
	}
}

func TestMakeSQLIdempotent_CreateFunctionUnqualified(t *testing.T) {
	// Unqualified function name (no schema): DROP uses the bare name.
	sql := `CREATE OR REPLACE FUNCTION my_func(id integer) RETURNS void LANGUAGE plpgsql AS $$ BEGIN END; $$;`
	out := MakeSQLIdempotent(sql)
	if !strings.Contains(out, `DROP FUNCTION IF EXISTS "my_func"(`) {
		t.Errorf("expected DROP FUNCTION for unqualified name, got:\n%s", out)
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
