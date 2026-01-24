package ai

import (
	"fmt"
	"strings"

	pg_query "github.com/pganalyze/pg_query_go/v5"
)

// IntentValidator validates that LLM-generated SQL matches user intent
type IntentValidator struct {
	intentRules     []IntentRule
	requiredColumns RequiredColumnsMap
	defaultTable    string
}

// IntentValidationResult contains the result of intent validation
type IntentValidationResult struct {
	Valid           bool     `json:"valid"`
	Errors          []string `json:"errors,omitempty"`
	Suggestions     []string `json:"suggestions,omitempty"`
	MatchedKeywords []string `json:"matched_keywords,omitempty"`
}

// NewIntentValidator creates a new intent validator
func NewIntentValidator(rules []IntentRule, requiredCols RequiredColumnsMap, defaultTable string) *IntentValidator {
	return &IntentValidator{
		intentRules:     rules,
		requiredColumns: requiredCols,
		defaultTable:    defaultTable,
	}
}

// ValidateIntent validates that SQL matches the user's intent based on keywords
func (v *IntentValidator) ValidateIntent(userMessage, sql string, tablesAccessed []string) *IntentValidationResult {
	result := &IntentValidationResult{
		Valid:           true,
		MatchedKeywords: []string{},
	}

	if len(v.intentRules) == 0 {
		return result // No rules configured
	}

	lowerMessage := strings.ToLower(userMessage)
	lowerTables := make(map[string]bool)
	for _, t := range tablesAccessed {
		tableLower := strings.ToLower(t)
		lowerTables[tableLower] = true
		// Also add without schema prefix for matching (e.g., "public.my_trips" -> "my_trips")
		if idx := strings.LastIndex(tableLower, "."); idx != -1 {
			lowerTables[tableLower[idx+1:]] = true
		}
	}

	for _, rule := range v.intentRules {
		// Check if any keywords match
		keywordMatched := false
		for _, keyword := range rule.Keywords {
			if strings.Contains(lowerMessage, strings.ToLower(keyword)) {
				keywordMatched = true
				result.MatchedKeywords = append(result.MatchedKeywords, keyword)
			}
		}

		if !keywordMatched {
			continue
		}

		// Check forbidden table first (more specific error)
		if rule.ForbiddenTable != "" {
			if lowerTables[strings.ToLower(rule.ForbiddenTable)] {
				result.Valid = false
				result.Errors = append(result.Errors,
					fmt.Sprintf("Query about '%s' should NOT use table '%s'",
						strings.Join(rule.Keywords, "/"), rule.ForbiddenTable))
				if rule.RequiredTable != "" {
					result.Suggestions = append(result.Suggestions,
						fmt.Sprintf("Use the '%s' table instead of '%s'", rule.RequiredTable, rule.ForbiddenTable))
				} else {
					result.Suggestions = append(result.Suggestions,
						fmt.Sprintf("Do not use the '%s' table for this type of query", rule.ForbiddenTable))
				}
			}
		}

		// Check required table
		if rule.RequiredTable != "" {
			if !lowerTables[strings.ToLower(rule.RequiredTable)] {
				result.Valid = false
				result.Errors = append(result.Errors,
					fmt.Sprintf("Query about '%s' should use table '%s'",
						strings.Join(rule.Keywords, "/"), rule.RequiredTable))
				result.Suggestions = append(result.Suggestions,
					fmt.Sprintf("Please query the '%s' table instead", rule.RequiredTable))
			}
		}
	}

	return result
}

// ValidateToolCall validates that the tool being called matches the user's intent based on keywords
func (v *IntentValidator) ValidateToolCall(userMessage string, toolName string) *IntentValidationResult {
	result := &IntentValidationResult{
		Valid:           true,
		MatchedKeywords: []string{},
	}

	if len(v.intentRules) == 0 {
		return result // No rules configured
	}

	lowerMessage := strings.ToLower(userMessage)

	for _, rule := range v.intentRules {
		// Skip rules that don't have tool constraints
		if rule.RequiredTool == "" && rule.ForbiddenTool == "" {
			continue
		}

		// Check if any keywords match
		keywordMatched := false
		for _, keyword := range rule.Keywords {
			if strings.Contains(lowerMessage, strings.ToLower(keyword)) {
				keywordMatched = true
				result.MatchedKeywords = append(result.MatchedKeywords, keyword)
			}
		}

		if !keywordMatched {
			continue
		}

		// Check forbidden tool first (more specific error)
		if rule.ForbiddenTool != "" && toolName == rule.ForbiddenTool {
			result.Valid = false
			result.Errors = append(result.Errors,
				fmt.Sprintf("Query about '%s' should NOT use tool '%s'",
					strings.Join(rule.Keywords, "/"), rule.ForbiddenTool))
			if rule.RequiredTool != "" {
				result.Suggestions = append(result.Suggestions,
					fmt.Sprintf("Use the '%s' tool instead", rule.RequiredTool))
			} else {
				result.Suggestions = append(result.Suggestions,
					"Use a different tool for this query")
			}
		}

		// Check required tool
		if rule.RequiredTool != "" && toolName != rule.RequiredTool {
			result.Valid = false
			result.Errors = append(result.Errors,
				fmt.Sprintf("Query about '%s' should use tool '%s', not '%s'",
					strings.Join(rule.Keywords, "/"), rule.RequiredTool, toolName))
			result.Suggestions = append(result.Suggestions,
				fmt.Sprintf("Use the '%s' tool instead", rule.RequiredTool))
		}
	}

	return result
}

// ValidateRequiredColumns checks that required columns are included in SELECT queries
func (v *IntentValidator) ValidateRequiredColumns(sql string, tablesAccessed []string) *IntentValidationResult {
	result := &IntentValidationResult{
		Valid: true,
	}

	if len(v.requiredColumns) == 0 {
		return result // No required columns configured
	}

	// Parse SQL to extract selected columns
	selectedColumns, err := extractSelectedColumns(sql)
	if err != nil {
		// If we can't parse, skip this validation
		return result
	}

	// Check each accessed table
	for _, table := range tablesAccessed {
		tableLower := strings.ToLower(table)
		// Also check without schema prefix
		tableName := tableLower
		if idx := strings.LastIndex(tableLower, "."); idx != -1 {
			tableName = tableLower[idx+1:]
		}

		// Check both with and without schema prefix
		var requiredCols []string
		if cols, exists := v.requiredColumns[tableName]; exists {
			requiredCols = cols
		} else if cols, exists := v.requiredColumns[tableLower]; exists {
			requiredCols = cols
		}

		if len(requiredCols) == 0 {
			continue
		}

		// Check if SELECT * is used
		if selectedColumns["*"] {
			continue // SELECT * includes all columns
		}

		// Check for missing required columns
		for _, reqCol := range requiredCols {
			reqColLower := strings.ToLower(reqCol)
			if !selectedColumns[reqColLower] {
				result.Valid = false
				result.Errors = append(result.Errors,
					fmt.Sprintf("Query on '%s' must include column '%s'", table, reqCol))
				result.Suggestions = append(result.Suggestions,
					fmt.Sprintf("Add '%s' to your SELECT clause", reqCol))
			}
		}
	}

	return result
}

// extractSelectedColumns parses SQL and returns a set of selected column names
func extractSelectedColumns(sql string) (map[string]bool, error) {
	columns := make(map[string]bool)

	parseResult, err := pg_query.Parse(sql)
	if err != nil {
		return nil, err
	}

	if len(parseResult.Stmts) == 0 {
		return columns, nil
	}

	stmt := parseResult.Stmts[0].Stmt

	// Only process SELECT statements
	selectStmt, ok := stmt.Node.(*pg_query.Node_SelectStmt)
	if !ok || selectStmt.SelectStmt == nil {
		return columns, nil
	}

	// Walk the target list to find column references
	for _, target := range selectStmt.SelectStmt.TargetList {
		extractColumnsFromNode(target, columns)
	}

	return columns, nil
}

// extractColumnsFromNode extracts column names from AST nodes
func extractColumnsFromNode(node *pg_query.Node, columns map[string]bool) {
	if node == nil {
		return
	}

	switch n := node.Node.(type) {
	case *pg_query.Node_ResTarget:
		if n.ResTarget != nil {
			// If there's an alias, use it
			if n.ResTarget.Name != "" {
				columns[strings.ToLower(n.ResTarget.Name)] = true
			}
			extractColumnsFromNode(n.ResTarget.Val, columns)
		}
	case *pg_query.Node_ColumnRef:
		if n.ColumnRef != nil {
			for _, field := range n.ColumnRef.Fields {
				if star, ok := field.Node.(*pg_query.Node_AStar); ok && star != nil {
					columns["*"] = true
				} else if str, ok := field.Node.(*pg_query.Node_String_); ok {
					columns[strings.ToLower(str.String_.Sval)] = true
				}
			}
		}
	case *pg_query.Node_FuncCall:
		// For function calls, we need to check the arguments
		if n.FuncCall != nil {
			for _, arg := range n.FuncCall.Args {
				extractColumnsFromNode(arg, columns)
			}
		}
	}
}

// GetDefaultTable returns the configured default table
func (v *IntentValidator) GetDefaultTable() string {
	return v.defaultTable
}

// HasIntentRules returns true if intent rules are configured
func (v *IntentValidator) HasIntentRules() bool {
	return len(v.intentRules) > 0
}

// HasRequiredColumns returns true if required columns are configured
func (v *IntentValidator) HasRequiredColumns() bool {
	return len(v.requiredColumns) > 0
}

// GetForbiddenTools returns tools that should be excluded based on keyword matches in the user message
func (v *IntentValidator) GetForbiddenTools(userMessage string) []string {
	var forbidden []string
	lowerMessage := strings.ToLower(userMessage)

	for _, rule := range v.intentRules {
		if rule.ForbiddenTool == "" {
			continue
		}

		// Check if any keywords match
		for _, keyword := range rule.Keywords {
			if strings.Contains(lowerMessage, strings.ToLower(keyword)) {
				forbidden = append(forbidden, rule.ForbiddenTool)
				break // Only add once per rule
			}
		}
	}

	return forbidden
}
