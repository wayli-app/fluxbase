package schema

import (
	"fmt"
	"strings"

	"github.com/pgplex/pgparser/nodes"
	"github.com/pgplex/pgparser/parser"
	"github.com/rs/zerolog/log"
)

// MakeSQLIdempotent transforms SQL to be idempotent by prepending DROP IF EXISTS
// statements before CREATE POLICY and ALTER TABLE ADD CONSTRAINT statements.
// This allows schema SQL files to be safely re-applied to existing databases.
func MakeSQLIdempotent(sql string) string {
	stmts, err := parser.Parse(sql)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to parse SQL for idempotency transformation, using original")
		return sql
	}

	if stmts == nil || len(stmts.Items) == 0 {
		return sql
	}

	type dropInfo struct {
		pattern  string
		dropSQL  string
		foundPos int
	}
	var drops []dropInfo

	for _, item := range stmts.Items {
		switch stmt := item.(type) {
		case *nodes.CreatePolicyStmt:
			if stmt.Table != nil {
				tableName := formatRangeVar(stmt.Table)
				dropSQL := fmt.Sprintf("DROP POLICY IF EXISTS %s ON %s CASCADE;\n", quoteIdent(stmt.PolicyName), tableName)
				patternQuoted := "CREATE POLICY \"" + stmt.PolicyName + "\""
				patternUnquoted := "CREATE POLICY " + stmt.PolicyName
				drops = append(drops, dropInfo{pattern: patternQuoted, dropSQL: dropSQL, foundPos: -1})
				drops = append(drops, dropInfo{pattern: patternUnquoted, dropSQL: dropSQL, foundPos: -1})
			}

		case *nodes.CreateTrigStmt:
			// CREATE TRIGGER is not idempotent: re-running fails if the trigger
			// exists. Prepend DROP TRIGGER IF EXISTS so a declarative schema file
			// can be re-applied (and trigger definitions updated) safely.
			if stmt.Trigname != "" && stmt.Relation != nil {
				tableName := formatRangeVar(stmt.Relation)
				dropSQL := fmt.Sprintf("DROP TRIGGER IF EXISTS %s ON %s CASCADE;\n", quoteIdent(stmt.Trigname), tableName)
				patternQuoted := "CREATE TRIGGER \"" + stmt.Trigname + "\""
				patternUnquoted := "CREATE TRIGGER " + stmt.Trigname
				drops = append(drops, dropInfo{pattern: patternQuoted, dropSQL: dropSQL, foundPos: -1})
				drops = append(drops, dropInfo{pattern: patternUnquoted, dropSQL: dropSQL, foundPos: -1})
			}

		case *nodes.IndexStmt:
			// CREATE INDEX (without IF NOT EXISTS, without CONCURRENTLY) is not
			// idempotent and can't update an existing index's definition. Prepend
			// DROP INDEX IF EXISTS so index definition changes propagate on
			// re-apply. Skip IF NOT EXISTS (already a no-op-safe no-op) and
			// CONCURRENTLY (cannot run in a transaction / can't be dropped+recreated
			// atomically). Constraint-backed indexes (Primary/Isconstraint) are
			// managed via ALTER TABLE, not here.
			if stmt.Idxname != "" && !stmt.IfNotExists && !stmt.Concurrent && !stmt.Primary && !stmt.Isconstraint {
				dropSQL := fmt.Sprintf("DROP INDEX IF EXISTS %s;\n", quoteIdent(stmt.Idxname))
				patternQuoted := "CREATE INDEX \"" + stmt.Idxname + "\""
				patternUnquoted := "CREATE INDEX " + stmt.Idxname
				drops = append(drops, dropInfo{pattern: patternQuoted, dropSQL: dropSQL, foundPos: -1})
				drops = append(drops, dropInfo{pattern: patternUnquoted, dropSQL: dropSQL, foundPos: -1})
			}

		case *nodes.ViewStmt:
			// CREATE OR REPLACE VIEW cannot change column names, types, or order
			// of an existing view. When the view definition changes (e.g. a new
			// column is added to the underlying table and selected), OR REPLACE
			// fails with "cannot change name of view column". Prepend
			// DROP VIEW IF EXISTS so the view is recreated fresh.
			if stmt.View != nil {
				viewName := formatRangeVar(stmt.View)
				dropSQL := fmt.Sprintf("DROP VIEW IF EXISTS %s CASCADE;\n", viewName)
				patternQuoted := "CREATE OR REPLACE VIEW \"" + stmt.View.Relname + "\""
				patternUnquoted := "CREATE OR REPLACE VIEW " + stmt.View.Relname
				drops = append(drops, dropInfo{pattern: patternQuoted, dropSQL: dropSQL, foundPos: -1})
				drops = append(drops, dropInfo{pattern: patternUnquoted, dropSQL: dropSQL, foundPos: -1})
			}

		case *nodes.CreateFunctionStmt:
			// CREATE OR REPLACE FUNCTION cannot change a function's argument
			// types or RETURN TYPE (Postgres rejects with SQLSTATE 42P13:
			// "cannot change return type of existing function"). Prepend
			// DROP FUNCTION IF EXISTS <name>(<argtypes>) CASCADE so the
			// function is recreated fresh on re-apply. The signature (argument
			// type list) is required by DROP FUNCTION to identify the overload.
			fnName, fnQualBare := formatFunctionName(stmt.Funcname)
			if fnName != "" {
				argTypes := formatFunctionArgTypes(stmt.Parameters)
				dropSQL := fmt.Sprintf("DROP FUNCTION IF EXISTS %s(%s) CASCADE;\n", fnName, argTypes)
				// Match both "CREATE FUNCTION" and "CREATE OR REPLACE FUNCTION"
				// forms (and quoted/unqualified name variants).
				for _, p := range functionCreatePatterns(fnQualBare) {
					drops = append(drops, dropInfo{pattern: p, dropSQL: dropSQL, foundPos: -1})
				}
			}

		case *nodes.AlterTableStmt:
			if stmt.Cmds != nil && stmt.Relation != nil {
				for _, cmd := range stmt.Cmds.Items {
					alterCmd, ok := cmd.(*nodes.AlterTableCmd)
					if !ok {
						continue
					}
					if alterCmd.Subtype == 17 && alterCmd.Def != nil {
						if constraint, ok := alterCmd.Def.(*nodes.Constraint); ok && constraint.Conname != "" {
							tableName := formatRangeVar(stmt.Relation)
							dropSQL := fmt.Sprintf("ALTER TABLE %s DROP CONSTRAINT IF EXISTS %s;\n", tableName, quoteIdent(constraint.Conname))
							pattern := "ALTER TABLE " + stmt.Relation.Relname
							drops = append(drops, dropInfo{pattern: pattern, dropSQL: dropSQL, foundPos: -1})
							break
						}
					}
				}
			}
		}
	}

	if len(drops) == 0 {
		return sql
	}

	upperSQL := strings.ToUpper(sql)
	lastFoundPos := make(map[string]int)

	for i := range drops {
		upperPattern := strings.ToUpper(drops[i].pattern)
		searchStart := lastFoundPos[upperPattern]
		idx := strings.Index(upperSQL[searchStart:], upperPattern)
		if idx != -1 {
			drops[i].foundPos = searchStart + idx
			lastFoundPos[upperPattern] = drops[i].foundPos + len(upperPattern)
		}
	}

	for i := 0; i < len(drops)-1; i++ {
		for j := i + 1; j < len(drops); j++ {
			if drops[i].foundPos < drops[j].foundPos {
				drops[i], drops[j] = drops[j], drops[i]
			}
		}
	}

	result := sql
	for _, drop := range drops {
		if drop.foundPos >= 0 {
			result = result[:drop.foundPos] + drop.dropSQL + result[drop.foundPos:]
		}
	}

	return result
}

func formatRangeVar(rv *nodes.RangeVar) string {
	if rv.Schemaname != "" {
		return fmt.Sprintf("%s.%s", quoteIdent(rv.Schemaname), quoteIdent(rv.Relname))
	}
	return quoteIdent(rv.Relname)
}

func quoteIdent(name string) string {
	if strings.HasPrefix(name, `"`) && strings.HasSuffix(name, `"`) {
		return name
	}
	return fmt.Sprintf(`"%s"`, strings.ReplaceAll(name, `"`, `""`))
}

// formatFunctionName builds the function name from a CreateFunctionStmt.Funcname
// list (a *List of *nodes.String, e.g. ["public", "get_public_trip_track"]).
// Returns ("", "") if the list is empty.
//   - First return: schema-qualified, quoted form for the DROP statement
//     (e.g. `"public"."get_public_trip_track"`).
//   - Second return: schema-qualified, unquoted form as it typically appears in
//     the CREATE statement (e.g. `public.get_public_trip_track`), used to build
//     the textual match pattern.
func formatFunctionName(funcname *nodes.List) (string, string) {
	if funcname == nil || len(funcname.Items) == 0 {
		return "", ""
	}
	parts := make([]string, 0, len(funcname.Items))
	for _, item := range funcname.Items {
		s, ok := item.(*nodes.String)
		if !ok || s == nil {
			return "", ""
		}
		parts = append(parts, s.Str)
	}
	quoted := make([]string, len(parts))
	for i, p := range parts {
		quoted[i] = quoteIdent(p)
	}
	return strings.Join(quoted, "."), strings.Join(parts, ".")
}

// formatFunctionArgTypes builds the comma-separated IN-argument-type list for
// a DROP FUNCTION signature (e.g. "uuid, integer") from a CreateFunctionStmt
// Parameters list. Returns an empty string for no args.
//
// PostgreSQL identifies a function by its name + IN-argument types only. The
// RETURNS TABLE(...) columns and OUT args are parsed as Parameters too (with
// mode FUNC_PARAM_TABLE / FUNC_PARAM_OUT) and MUST be excluded, or the DROP's
// signature won't match the function. Only IN (i), INOUT (b), and VARIADIC (v)
// args form the signature.
func formatFunctionArgTypes(parameters *nodes.List) string {
	if parameters == nil {
		return ""
	}
	var types []string
	for _, item := range parameters.Items {
		param, ok := item.(*nodes.FunctionParameter)
		if !ok || param == nil || param.ArgType == nil {
			continue
		}
		switch param.Mode {
		case nodes.FUNC_PARAM_IN, nodes.FUNC_PARAM_INOUT, nodes.FUNC_PARAM_VARIADIC:
			// included in the signature
		default:
			// FUNC_PARAM_OUT / FUNC_PARAM_TABLE — not part of the identity.
			continue
		}
		types = append(types, formatTypeName(param.ArgType))
	}
	return strings.Join(types, ", ")
}

// formatTypeName renders a *nodes.TypeName as a SQL type reference (e.g.
// "uuid", "double precision", "public.geometry"). Uses the qualified Names
// list when present; falls back to "<unknown>" if unparseable (rare).
func formatTypeName(tn *nodes.TypeName) string {
	if tn == nil || tn.Names == nil || len(tn.Names.Items) == 0 {
		return ""
	}
	parts := make([]string, 0, len(tn.Names.Items))
	for _, item := range tn.Names.Items {
		s, ok := item.(*nodes.String)
		if !ok || s == nil {
			continue
		}
		parts = append(parts, s.Str)
	}
	name := strings.Join(parts, ".")
	// "double precision" and similar are parsed as a single list element
	// ("double precision"); keep them as-is. Array types carry ArrayBounds.
	if tn.ArrayBounds != nil && len(tn.ArrayBounds.Items) > 0 {
		name += "[]"
	}
	return name
}

// functionCreatePatterns returns the textual CREATE-statement prefixes to
// locate in the original SQL so the DROP is injected immediately before them.
// Covers "CREATE FUNCTION" and "CREATE OR REPLACE FUNCTION", matching the
// qualified name as it typically appears (unquoted, e.g.
// `public.get_public_trip_track`) and as a fully-quoted form. Matching is
// case-insensitive (the caller upper-cases both the pattern and the SQL).
func functionCreatePatterns(qualBareName string) []string {
	if qualBareName == "" {
		return nil
	}
	var out []string
	for _, kw := range []string{"CREATE OR REPLACE FUNCTION ", "CREATE FUNCTION "} {
		out = append(out, kw+qualBareName)
	}
	return out
}
