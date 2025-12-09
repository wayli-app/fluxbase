package rpc

import (
	"encoding/json"
	"fmt"
	"strings"

	pg_query "github.com/pganalyze/pg_query_go/v5"
	"github.com/rs/zerolog/log"
)

// Validator handles validation of RPC inputs and SQL queries
type Validator struct {
}

// NewValidator creates a new validator
func NewValidator() *Validator {
	return &Validator{}
}

// ValidationResult represents the result of SQL validation
type ValidationResult struct {
	Valid          bool     `json:"valid"`
	Errors         []string `json:"errors,omitempty"`
	Warnings       []string `json:"warnings,omitempty"`
	TablesAccessed []string `json:"tables_accessed,omitempty"`
	OperationsUsed []string `json:"operations_used,omitempty"`
}

// ValidateInput validates input parameters against a JSON schema
func (v *Validator) ValidateInput(params map[string]interface{}, schemaBytes json.RawMessage) error {
	if schemaBytes == nil || len(schemaBytes) == 0 {
		// No schema defined, accept any input
		return nil
	}

	// Parse the schema
	var schema map[string]string
	if err := json.Unmarshal(schemaBytes, &schema); err != nil {
		return fmt.Errorf("invalid input schema: %w", err)
	}

	// Validate each required field
	for fieldName, fieldType := range schema {
		isOptional := IsOptionalField(fieldName)
		cleanName := CleanFieldName(fieldName)

		value, exists := params[cleanName]

		// Check required fields
		if !exists {
			if !isOptional {
				return fmt.Errorf("missing required parameter: %s", cleanName)
			}
			continue
		}

		// Validate type
		if err := v.validateType(cleanName, value, fieldType); err != nil {
			return err
		}
	}

	return nil
}

// validateType validates a value against an expected type
func (v *Validator) validateType(fieldName string, value interface{}, expectedType string) error {
	if value == nil {
		return nil // Null values are allowed
	}

	switch strings.ToLower(expectedType) {
	case "uuid":
		if _, ok := value.(string); !ok {
			return fmt.Errorf("parameter %s must be a UUID string", fieldName)
		}
	case "string", "text":
		if _, ok := value.(string); !ok {
			return fmt.Errorf("parameter %s must be a string", fieldName)
		}
	case "number", "int", "integer":
		switch value.(type) {
		case int, int32, int64, float32, float64, json.Number:
			// Valid numeric types
		default:
			return fmt.Errorf("parameter %s must be a number", fieldName)
		}
	case "float", "double", "decimal":
		switch value.(type) {
		case float32, float64, json.Number:
			// Valid float types
		default:
			return fmt.Errorf("parameter %s must be a decimal number", fieldName)
		}
	case "boolean", "bool":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("parameter %s must be a boolean", fieldName)
		}
	case "array":
		if _, ok := value.([]interface{}); !ok {
			return fmt.Errorf("parameter %s must be an array", fieldName)
		}
	case "object", "json", "jsonb":
		if _, ok := value.(map[string]interface{}); !ok {
			return fmt.Errorf("parameter %s must be an object", fieldName)
		}
	}

	return nil
}

// ValidateSQL validates a SQL query for safety and compliance
func (v *Validator) ValidateSQL(sql string, allowedTables, allowedSchemas []string) *ValidationResult {
	result := &ValidationResult{
		Valid:          true,
		TablesAccessed: []string{},
		OperationsUsed: []string{},
	}

	// Build lookup maps
	tablesMap := make(map[string]bool)
	for _, t := range allowedTables {
		tablesMap[strings.ToLower(t)] = true
	}

	schemasMap := make(map[string]bool)
	for _, s := range allowedSchemas {
		schemasMap[strings.ToLower(s)] = true
	}

	// Basic safety checks
	lowerSQL := strings.ToLower(sql)

	// Blocked patterns
	blockedPatterns := []string{
		"pg_catalog",
		"information_schema",
		"pg_temp",
		"pg_toast",
		"--",       // SQL comment injection
		"/*",       // Block comment injection
		"xp_",      // MSSQL stored procedures (defense in depth)
		"exec(",    // Execution attempt
		"execute(", // Execution attempt
	}

	for _, pattern := range blockedPatterns {
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

			if len(schemasMap) > 0 && !schemasMap[schema] {
				result.Valid = false
				result.Errors = append(result.Errors, fmt.Sprintf("Schema not allowed: %s", schema))
			}

			// If specific tables are allowed, check table name
			if len(tablesMap) > 0 {
				if !tablesMap[strings.ToLower(tableName)] && !tablesMap[tableLower] {
					result.Valid = false
					result.Errors = append(result.Errors, fmt.Sprintf("Table not allowed: %s", table))
				}
			}
		} else {
			// No schema prefix - check against allowed tables
			if len(tablesMap) > 0 && !tablesMap[tableLower] {
				result.Valid = false
				result.Errors = append(result.Errors, fmt.Sprintf("Table not allowed: %s", table))
			}
		}
	}

	return result
}

// ValidateAccess checks if a user with the given role can access the procedure
func (v *Validator) ValidateAccess(proc *Procedure, userRole string, isAuthenticated bool) error {
	// Check public access
	if !isAuthenticated && !proc.IsPublic {
		return fmt.Errorf("procedure requires authentication")
	}

	// Check role requirement
	if proc.RequireRole != nil && *proc.RequireRole != "" {
		requiredRole := *proc.RequireRole

		switch requiredRole {
		case "anon":
			// Anonymous access allowed
			return nil
		case "authenticated":
			if !isAuthenticated {
				return fmt.Errorf("procedure requires authentication")
			}
		default:
			// Specific role required
			if userRole != requiredRole && userRole != "service_role" && userRole != "dashboard_admin" {
				return fmt.Errorf("procedure requires role: %s", requiredRole)
			}
		}
	}

	return nil
}

// getOperationType determines the SQL operation type from a parsed statement
func (v *Validator) getOperationType(stmt *pg_query.Node) string {
	if stmt == nil {
		return ""
	}

	switch {
	case stmt.GetSelectStmt() != nil:
		return "SELECT"
	case stmt.GetInsertStmt() != nil:
		return "INSERT"
	case stmt.GetUpdateStmt() != nil:
		return "UPDATE"
	case stmt.GetDeleteStmt() != nil:
		return "DELETE"
	case stmt.GetCreateStmt() != nil:
		return "CREATE"
	case stmt.GetDropStmt() != nil:
		return "DROP"
	case stmt.GetAlterTableStmt() != nil:
		return "ALTER"
	case stmt.GetTruncateStmt() != nil:
		return "TRUNCATE"
	default:
		return "UNKNOWN"
	}
}

// extractTables extracts table names from a parsed statement
func (v *Validator) extractTables(stmt *pg_query.Node) []string {
	if stmt == nil {
		return nil
	}

	tables := make(map[string]bool)
	v.walkNode(stmt, tables)

	var result []string
	for table := range tables {
		result = append(result, table)
	}
	return result
}

// walkNode recursively walks the AST to find table references
func (v *Validator) walkNode(node *pg_query.Node, tables map[string]bool) {
	if node == nil {
		return
	}

	// Check for RangeVar (table reference)
	if rv := node.GetRangeVar(); rv != nil {
		tableName := rv.Relname
		if rv.Schemaname != "" {
			tableName = rv.Schemaname + "." + tableName
		}
		if tableName != "" {
			tables[tableName] = true
		}
	}

	// Check for SelectStmt
	if sel := node.GetSelectStmt(); sel != nil {
		for _, from := range sel.FromClause {
			v.walkNode(from, tables)
		}
		if sel.WhereClause != nil {
			v.walkNode(sel.WhereClause, tables)
		}
		for _, target := range sel.TargetList {
			v.walkNode(target, tables)
		}
	}

	// Check for InsertStmt
	if ins := node.GetInsertStmt(); ins != nil {
		if ins.Relation != nil {
			tableName := ins.Relation.Relname
			if ins.Relation.Schemaname != "" {
				tableName = ins.Relation.Schemaname + "." + tableName
			}
			tables[tableName] = true
		}
		if ins.SelectStmt != nil {
			v.walkNode(ins.SelectStmt, tables)
		}
	}

	// Check for UpdateStmt
	if upd := node.GetUpdateStmt(); upd != nil {
		if upd.Relation != nil {
			tableName := upd.Relation.Relname
			if upd.Relation.Schemaname != "" {
				tableName = upd.Relation.Schemaname + "." + tableName
			}
			tables[tableName] = true
		}
		for _, from := range upd.FromClause {
			v.walkNode(from, tables)
		}
	}

	// Check for DeleteStmt
	if del := node.GetDeleteStmt(); del != nil {
		if del.Relation != nil {
			tableName := del.Relation.Relname
			if del.Relation.Schemaname != "" {
				tableName = del.Relation.Schemaname + "." + tableName
			}
			tables[tableName] = true
		}
	}

	// Check for JoinExpr
	if join := node.GetJoinExpr(); join != nil {
		v.walkNode(join.Larg, tables)
		v.walkNode(join.Rarg, tables)
	}

	// Check for SubLink (subquery)
	if sublink := node.GetSubLink(); sublink != nil {
		v.walkNode(sublink.Subselect, tables)
	}

	// Check for CommonTableExpr (WITH clause)
	if cte := node.GetCommonTableExpr(); cte != nil {
		v.walkNode(cte.Ctequery, tables)
	}

	// Log unhandled node types for debugging
	log.Trace().Interface("node_type", fmt.Sprintf("%T", node.Node)).Msg("Walking AST node")
}
