package ai

import (
	"fmt"
	"strings"

	pg_query "github.com/pganalyze/pg_query_go/v5"
	"github.com/rs/zerolog/log"
)

// SQLValidator validates SQL queries for safety and compliance
type SQLValidator struct {
	allowedSchemas    map[string]bool
	allowedTables     map[string]bool
	allowedOperations map[string]bool
	blockedPatterns   []string
}

// ValidationResult represents the result of SQL validation
type ValidationResult struct {
	Valid           bool     `json:"valid"`
	Errors          []string `json:"errors,omitempty"`
	Warnings        []string `json:"warnings,omitempty"`
	TablesAccessed  []string `json:"tables_accessed,omitempty"`
	OperationsUsed  []string `json:"operations_used,omitempty"`
	NormalizedQuery string   `json:"normalized_query,omitempty"`
}

// NewSQLValidator creates a new SQL validator with the given constraints
func NewSQLValidator(allowedSchemas, allowedTables, allowedOperations []string) *SQLValidator {
	// Build lookup maps
	schemas := make(map[string]bool)
	for _, s := range allowedSchemas {
		schemas[strings.ToLower(s)] = true
	}

	tables := make(map[string]bool)
	for _, t := range allowedTables {
		tables[strings.ToLower(t)] = true
	}

	operations := make(map[string]bool)
	for _, o := range allowedOperations {
		operations[strings.ToUpper(o)] = true
	}

	return &SQLValidator{
		allowedSchemas:    schemas,
		allowedTables:     tables,
		allowedOperations: operations,
		blockedPatterns: []string{
			"pg_catalog",
			"information_schema",
			"pg_temp",
			"pg_toast",
			"--",       // SQL comment injection
			"/*",       // Block comment injection
			"xp_",      // MSSQL stored procedures (defense in depth)
			"exec(",    // Execution attempt
			"execute(", // Execution attempt
		},
	}
}

// Validate validates a SQL query
func (v *SQLValidator) Validate(sql string) *ValidationResult {
	result := &ValidationResult{
		Valid:          true,
		TablesAccessed: []string{},
		OperationsUsed: []string{},
	}

	// Basic safety checks
	lowerSQL := strings.ToLower(sql)

	// Check for blocked patterns
	for _, pattern := range v.blockedPatterns {
		if strings.Contains(lowerSQL, strings.ToLower(pattern)) {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("Query contains blocked pattern: %s", pattern))
		}
	}

	// Parse the SQL
	parseResult, err := pg_query.Parse(sql)
	if err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to parse SQL: %s", err.Error()))
		return result
	}

	// Check for multiple statements
	if len(parseResult.Stmts) > 1 {
		result.Valid = false
		result.Errors = append(result.Errors, "Multiple SQL statements not allowed")
		return result
	}

	if len(parseResult.Stmts) == 0 {
		result.Valid = false
		result.Errors = append(result.Errors, "Empty SQL statement")
		return result
	}

	// Get the statement
	stmt := parseResult.Stmts[0].Stmt

	// Determine operation type
	operationType := v.getOperationType(stmt)
	if operationType != "" {
		result.OperationsUsed = append(result.OperationsUsed, operationType)

		// Check if operation is allowed
		if !v.allowedOperations[operationType] {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("Operation not allowed: %s", operationType))
		}
	}

	// Extract and validate tables
	tables := v.extractTables(stmt)
	result.TablesAccessed = tables

	for _, table := range tables {
		tableLower := strings.ToLower(table)

		// Check schema restrictions
		if strings.Contains(table, ".") {
			parts := strings.SplitN(table, ".", 2)
			schema := strings.ToLower(parts[0])
			tableName := parts[1]

			if !v.allowedSchemas[schema] {
				result.Valid = false
				result.Errors = append(result.Errors, fmt.Sprintf("Schema not allowed: %s", schema))
			}

			// If specific tables are allowed, check table name
			if len(v.allowedTables) > 0 {
				if !v.allowedTables[strings.ToLower(tableName)] && !v.allowedTables[tableLower] {
					result.Valid = false
					result.Errors = append(result.Errors, fmt.Sprintf("Table not allowed: %s", table))
				}
			}
		} else {
			// No schema prefix - check against allowed tables
			if len(v.allowedTables) > 0 && !v.allowedTables[tableLower] {
				result.Valid = false
				result.Errors = append(result.Errors, fmt.Sprintf("Table not allowed: %s", table))
			}
		}
	}

	// Check for dangerous functions in SELECT queries
	if operationType == "SELECT" {
		v.checkDangerousFunctions(stmt, result)
	}

	// Normalize the query for logging
	result.NormalizedQuery = v.normalizeQuery(sql)

	return result
}

// getOperationType determines the type of SQL operation
func (v *SQLValidator) getOperationType(stmt *pg_query.Node) string {
	if stmt == nil {
		return ""
	}

	switch stmt.Node.(type) {
	case *pg_query.Node_SelectStmt:
		return "SELECT"
	case *pg_query.Node_InsertStmt:
		return "INSERT"
	case *pg_query.Node_UpdateStmt:
		return "UPDATE"
	case *pg_query.Node_DeleteStmt:
		return "DELETE"
	case *pg_query.Node_CreateStmt:
		return "CREATE"
	case *pg_query.Node_DropStmt:
		return "DROP"
	case *pg_query.Node_AlterTableStmt:
		return "ALTER"
	case *pg_query.Node_TruncateStmt:
		return "TRUNCATE"
	case *pg_query.Node_GrantStmt:
		return "GRANT"
	default:
		return "UNKNOWN"
	}
}

// extractTables extracts table names from a parsed SQL statement
func (v *SQLValidator) extractTables(stmt *pg_query.Node) []string {
	tables := make(map[string]bool)

	v.walkNode(stmt, func(node *pg_query.Node) {
		if node == nil {
			return
		}

		switch n := node.Node.(type) {
		case *pg_query.Node_RangeVar:
			if n.RangeVar != nil {
				tableName := n.RangeVar.Relname
				if n.RangeVar.Schemaname != "" {
					tableName = n.RangeVar.Schemaname + "." + tableName
				}
				if tableName != "" {
					tables[tableName] = true
				}
			}
		}
	})

	result := make([]string, 0, len(tables))
	for t := range tables {
		result = append(result, t)
	}
	return result
}

// walkNode recursively walks a parse tree and calls the visitor function
func (v *SQLValidator) walkNode(node *pg_query.Node, visitor func(*pg_query.Node)) {
	if node == nil {
		return
	}

	visitor(node)

	// Walk children based on node type
	switch n := node.Node.(type) {
	case *pg_query.Node_SelectStmt:
		if n.SelectStmt != nil {
			for _, target := range n.SelectStmt.TargetList {
				v.walkNode(target, visitor)
			}
			for _, from := range n.SelectStmt.FromClause {
				v.walkNode(from, visitor)
			}
			v.walkNode(n.SelectStmt.WhereClause, visitor)
			for _, group := range n.SelectStmt.GroupClause {
				v.walkNode(group, visitor)
			}
			v.walkNode(n.SelectStmt.HavingClause, visitor)
			for _, order := range n.SelectStmt.SortClause {
				v.walkNode(order, visitor)
			}
			v.walkNode(n.SelectStmt.LimitOffset, visitor)
			v.walkNode(n.SelectStmt.LimitCount, visitor)
			// Handle CTEs
			if n.SelectStmt.WithClause != nil {
				for _, cte := range n.SelectStmt.WithClause.Ctes {
					v.walkNode(cte, visitor)
				}
			}
			// Handle subqueries (UNION, INTERSECT, EXCEPT)
			if n.SelectStmt.Larg != nil {
				v.walkNode(&pg_query.Node{Node: &pg_query.Node_SelectStmt{SelectStmt: n.SelectStmt.Larg}}, visitor)
			}
			if n.SelectStmt.Rarg != nil {
				v.walkNode(&pg_query.Node{Node: &pg_query.Node_SelectStmt{SelectStmt: n.SelectStmt.Rarg}}, visitor)
			}
		}

	case *pg_query.Node_InsertStmt:
		if n.InsertStmt != nil {
			if n.InsertStmt.Relation != nil {
				v.walkNode(&pg_query.Node{Node: &pg_query.Node_RangeVar{RangeVar: n.InsertStmt.Relation}}, visitor)
			}
			v.walkNode(n.InsertStmt.SelectStmt, visitor)
		}

	case *pg_query.Node_UpdateStmt:
		if n.UpdateStmt != nil {
			if n.UpdateStmt.Relation != nil {
				v.walkNode(&pg_query.Node{Node: &pg_query.Node_RangeVar{RangeVar: n.UpdateStmt.Relation}}, visitor)
			}
			for _, from := range n.UpdateStmt.FromClause {
				v.walkNode(from, visitor)
			}
			v.walkNode(n.UpdateStmt.WhereClause, visitor)
		}

	case *pg_query.Node_DeleteStmt:
		if n.DeleteStmt != nil {
			if n.DeleteStmt.Relation != nil {
				v.walkNode(&pg_query.Node{Node: &pg_query.Node_RangeVar{RangeVar: n.DeleteStmt.Relation}}, visitor)
			}
			v.walkNode(n.DeleteStmt.WhereClause, visitor)
		}

	case *pg_query.Node_RangeVar:
		// Already handled by visitor

	case *pg_query.Node_JoinExpr:
		if n.JoinExpr != nil {
			v.walkNode(n.JoinExpr.Larg, visitor)
			v.walkNode(n.JoinExpr.Rarg, visitor)
			v.walkNode(n.JoinExpr.Quals, visitor)
		}

	case *pg_query.Node_RangeSubselect:
		if n.RangeSubselect != nil {
			v.walkNode(n.RangeSubselect.Subquery, visitor)
		}

	case *pg_query.Node_SubLink:
		if n.SubLink != nil {
			v.walkNode(n.SubLink.Subselect, visitor)
		}

	case *pg_query.Node_FuncCall:
		if n.FuncCall != nil {
			for _, arg := range n.FuncCall.Args {
				v.walkNode(arg, visitor)
			}
		}

	case *pg_query.Node_AExpr:
		if n.AExpr != nil {
			v.walkNode(n.AExpr.Lexpr, visitor)
			v.walkNode(n.AExpr.Rexpr, visitor)
		}

	case *pg_query.Node_BoolExpr:
		if n.BoolExpr != nil {
			for _, arg := range n.BoolExpr.Args {
				v.walkNode(arg, visitor)
			}
		}

	case *pg_query.Node_CommonTableExpr:
		if n.CommonTableExpr != nil {
			v.walkNode(n.CommonTableExpr.Ctequery, visitor)
		}

	case *pg_query.Node_ResTarget:
		if n.ResTarget != nil {
			v.walkNode(n.ResTarget.Val, visitor)
		}
	}
}

// checkDangerousFunctions checks for potentially dangerous function calls
func (v *SQLValidator) checkDangerousFunctions(stmt *pg_query.Node, result *ValidationResult) {
	dangerousFunctions := map[string]bool{
		"pg_read_file":        true,
		"pg_read_binary_file": true,
		"pg_ls_dir":           true,
		"lo_import":           true,
		"lo_export":           true,
		"dblink":              true,
		"dblink_exec":         true,
		"set_config":          true,
		"current_setting":     true,
	}

	v.walkNode(stmt, func(node *pg_query.Node) {
		if node == nil {
			return
		}

		if fc, ok := node.Node.(*pg_query.Node_FuncCall); ok && fc.FuncCall != nil {
			funcName := ""
			for _, part := range fc.FuncCall.Funcname {
				if str, ok := part.Node.(*pg_query.Node_String_); ok {
					funcName = str.String_.Sval
				}
			}

			if dangerousFunctions[strings.ToLower(funcName)] {
				result.Valid = false
				result.Errors = append(result.Errors, fmt.Sprintf("Dangerous function not allowed: %s", funcName))
			}
		}
	})
}

// normalizeQuery normalizes a SQL query for logging (removes extra whitespace)
func (v *SQLValidator) normalizeQuery(sql string) string {
	// Simple normalization - collapse whitespace
	sql = strings.TrimSpace(sql)
	sql = strings.ReplaceAll(sql, "\n", " ")
	sql = strings.ReplaceAll(sql, "\t", " ")

	// Collapse multiple spaces
	for strings.Contains(sql, "  ") {
		sql = strings.ReplaceAll(sql, "  ", " ")
	}

	return sql
}

// ValidateAndNormalize validates and returns a normalized version of the query
func (v *SQLValidator) ValidateAndNormalize(sql string) (*ValidationResult, string, error) {
	result := v.Validate(sql)

	if !result.Valid {
		return result, "", fmt.Errorf("validation failed: %s", strings.Join(result.Errors, "; "))
	}

	// Normalize the query
	normalized := result.NormalizedQuery
	if normalized == "" {
		normalized = v.normalizeQuery(sql)
	}

	log.Debug().
		Str("original", sql).
		Str("normalized", normalized).
		Strs("tables", result.TablesAccessed).
		Strs("operations", result.OperationsUsed).
		Msg("SQL validation passed")

	return result, normalized, nil
}

// WrapRangeVar is a helper to create a RangeVar node for testing
func WrapRangeVar(rv *pg_query.RangeVar) *pg_query.Node {
	return &pg_query.Node{
		Node: &pg_query.Node_RangeVar{
			RangeVar: rv,
		},
	}
}
