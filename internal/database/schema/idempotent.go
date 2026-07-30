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
