package api

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// validPolicyNameRegex ensures policy names are safe PostgreSQL identifiers
var validPolicyNameRegex = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// Policy represents a PostgreSQL RLS policy
type Policy struct {
	Schema     string   `json:"schema"`
	Table      string   `json:"table"`
	PolicyName string   `json:"policy_name"`
	Permissive string   `json:"permissive"` // "PERMISSIVE" or "RESTRICTIVE"
	Roles      []string `json:"roles"`
	Command    string   `json:"command"` // ALL, SELECT, INSERT, UPDATE, DELETE
	Using      *string  `json:"using"`   // USING expression
	WithCheck  *string  `json:"with_check"` // WITH CHECK expression
}

// TableRLSStatus represents RLS status for a table
type TableRLSStatus struct {
	Schema      string   `json:"schema"`
	Table       string   `json:"table"`
	RLSEnabled  bool     `json:"rls_enabled"`
	RLSForced   bool     `json:"rls_forced"`
	PolicyCount int      `json:"policy_count"`
	Policies    []Policy `json:"policies"`
	HasWarnings bool     `json:"has_warnings"`
}

// SecurityWarning represents a security issue detected
type SecurityWarning struct {
	ID          string `json:"id"`
	Severity    string `json:"severity"` // critical, high, medium, low
	Category    string `json:"category"`
	Schema      string `json:"schema"`
	Table       string `json:"table"`
	PolicyName  string `json:"policy_name,omitempty"`
	Message     string `json:"message"`
	Suggestion  string `json:"suggestion"`
	FixSQL      string `json:"fix_sql,omitempty"`
}

// CreatePolicyRequest is the request body for creating a policy
type CreatePolicyRequest struct {
	Schema     string   `json:"schema"`
	Table      string   `json:"table"`
	Name       string   `json:"name"`
	Command    string   `json:"command"`    // ALL, SELECT, INSERT, UPDATE, DELETE
	Permissive bool     `json:"permissive"` // true = PERMISSIVE, false = RESTRICTIVE
	Roles      []string `json:"roles"`
	Using      string   `json:"using"`
	WithCheck  string   `json:"with_check"`
}

// ListPolicies returns all RLS policies
// GET /api/v1/admin/policies
func (s *Server) ListPolicies(c *fiber.Ctx) error {
	ctx := c.Context()
	schema := c.Query("schema", "")

	query := `
		SELECT
			schemaname,
			tablename,
			policyname,
			permissive,
			roles,
			cmd,
			qual,
			with_check
		FROM pg_policies
		WHERE schemaname NOT IN ('pg_catalog', 'information_schema')
	`
	args := []interface{}{}

	if schema != "" {
		query += " AND schemaname = $1"
		args = append(args, schema)
	}
	query += " ORDER BY schemaname, tablename, policyname"

	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return SendError(c, fiber.StatusInternalServerError, "QUERY_FAILED", err.Error())
	}
	defer rows.Close()

	policies := []Policy{}
	for rows.Next() {
		var p Policy
		var roles []string
		err := rows.Scan(
			&p.Schema, &p.Table, &p.PolicyName, &p.Permissive,
			&roles, &p.Command, &p.Using, &p.WithCheck,
		)
		if err != nil {
			return SendError(c, fiber.StatusInternalServerError, "SCAN_FAILED", err.Error())
		}
		p.Roles = roles
		policies = append(policies, p)
	}

	return c.JSON(policies)
}

// GetTablesWithRLS returns all tables with their RLS status and policies
// GET /api/v1/admin/tables/rls
func (s *Server) GetTablesWithRLS(c *fiber.Ctx) error {
	ctx := c.Context()
	schema := c.Query("schema", "public")

	// Get tables with RLS status
	tablesQuery := `
		SELECT
			n.nspname as schema,
			c.relname as table_name,
			c.relrowsecurity as rls_enabled,
			c.relforcerowsecurity as rls_forced
		FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = $1
		AND c.relkind = 'r'
		AND c.relname NOT LIKE 'pg_%'
		ORDER BY c.relname
	`

	tablesRows, err := s.db.Query(ctx, tablesQuery, schema)
	if err != nil {
		return SendError(c, fiber.StatusInternalServerError, "TABLES_QUERY_FAILED", err.Error())
	}
	defer tablesRows.Close()

	tablesMap := make(map[string]*TableRLSStatus)
	for tablesRows.Next() {
		var t TableRLSStatus
		err := tablesRows.Scan(&t.Schema, &t.Table, &t.RLSEnabled, &t.RLSForced)
		if err != nil {
			return SendError(c, fiber.StatusInternalServerError, "TABLES_SCAN_FAILED", err.Error())
		}
		t.Policies = []Policy{}
		tablesMap[t.Table] = &t
	}

	// Get policies for tables in this schema
	policiesQuery := `
		SELECT
			tablename,
			policyname,
			permissive,
			roles,
			cmd,
			qual,
			with_check
		FROM pg_policies
		WHERE schemaname = $1
		ORDER BY tablename, policyname
	`

	policyRows, err := s.db.Query(ctx, policiesQuery, schema)
	if err != nil {
		return SendError(c, fiber.StatusInternalServerError, "POLICIES_QUERY_FAILED", err.Error())
	}
	defer policyRows.Close()

	for policyRows.Next() {
		var tableName string
		var p Policy
		var roles []string
		err := policyRows.Scan(
			&tableName, &p.PolicyName, &p.Permissive,
			&roles, &p.Command, &p.Using, &p.WithCheck,
		)
		if err != nil {
			return SendError(c, fiber.StatusInternalServerError, "POLICY_SCAN_FAILED", err.Error())
		}
		p.Schema = schema
		p.Table = tableName
		p.Roles = roles

		if table, exists := tablesMap[tableName]; exists {
			table.Policies = append(table.Policies, p)
			table.PolicyCount = len(table.Policies)
		}
	}

	// Check for warnings
	for _, table := range tablesMap {
		// Warning: RLS disabled on public table
		if !table.RLSEnabled && table.Schema == "public" {
			table.HasWarnings = true
		}
		// Warning: RLS enabled but no policies
		if table.RLSEnabled && table.PolicyCount == 0 {
			table.HasWarnings = true
		}
	}

	// Convert to slice
	tables := make([]TableRLSStatus, 0, len(tablesMap))
	for _, t := range tablesMap {
		tables = append(tables, *t)
	}

	return c.JSON(tables)
}

// GetTableRLSStatus returns RLS status and policies for a specific table
// GET /api/v1/admin/tables/:schema/:table/rls
func (s *Server) GetTableRLSStatus(c *fiber.Ctx) error {
	ctx := c.Context()
	schema := c.Params("schema")
	table := c.Params("table")

	// Get RLS status
	var status TableRLSStatus
	status.Schema = schema
	status.Table = table

	err := s.db.QueryRow(ctx, `
		SELECT relrowsecurity, relforcerowsecurity
		FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = $1 AND c.relname = $2
	`, schema, table).Scan(&status.RLSEnabled, &status.RLSForced)
	if err != nil {
		return SendNotFound(c, "Table not found")
	}

	// Get policies
	rows, err := s.db.Query(ctx, `
		SELECT policyname, permissive, roles, cmd, qual, with_check
		FROM pg_policies
		WHERE schemaname = $1 AND tablename = $2
		ORDER BY policyname
	`, schema, table)
	if err != nil {
		return SendError(c, fiber.StatusInternalServerError, "QUERY_FAILED", err.Error())
	}
	defer rows.Close()

	status.Policies = []Policy{}
	for rows.Next() {
		var p Policy
		var roles []string
		err := rows.Scan(&p.PolicyName, &p.Permissive, &roles, &p.Command, &p.Using, &p.WithCheck)
		if err != nil {
			continue
		}
		p.Schema = schema
		p.Table = table
		p.Roles = roles
		status.Policies = append(status.Policies, p)
	}
	status.PolicyCount = len(status.Policies)

	return c.JSON(status)
}

// ToggleTableRLS enables or disables RLS on a table
// POST /api/v1/admin/tables/:schema/:table/rls/toggle
func (s *Server) ToggleTableRLS(c *fiber.Ctx) error {
	ctx := c.Context()
	schema := c.Params("schema")
	table := c.Params("table")

	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.BodyParser(&req); err != nil {
		return SendBadRequest(c, "Invalid request body")
	}

	// Validate table exists
	var exists bool
	err := s.db.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM pg_class c
			JOIN pg_namespace n ON n.oid = c.relnamespace
			WHERE n.nspname = $1 AND c.relname = $2
		)
	`, schema, table).Scan(&exists)
	if err != nil || !exists {
		return SendNotFound(c, "Table not found")
	}

	// Toggle RLS
	action := "DISABLE"
	if req.Enabled {
		action = "ENABLE"
	}

	sql := fmt.Sprintf(
		"ALTER TABLE %s.%s %s ROW LEVEL SECURITY",
		quoteIdentifier(schema),
		quoteIdentifier(table),
		action,
	)

	_, err = s.db.Exec(ctx, sql)
	if err != nil {
		return SendError(c, fiber.StatusInternalServerError, "RLS_TOGGLE_FAILED", err.Error())
	}

	return c.JSON(fiber.Map{
		"success":     true,
		"rls_enabled": req.Enabled,
	})
}

// CreatePolicy creates a new RLS policy
// POST /api/v1/admin/policies
func (s *Server) CreatePolicy(c *fiber.Ctx) error {
	ctx := c.Context()

	var req CreatePolicyRequest
	if err := c.BodyParser(&req); err != nil {
		return SendBadRequest(c, "Invalid request body")
	}

	// Validate inputs
	if req.Schema == "" || req.Table == "" || req.Name == "" {
		return SendBadRequest(c, "schema, table, and name are required")
	}

	// Validate policy name format
	if !validPolicyNameRegex.MatchString(req.Name) {
		return SendBadRequest(c, "Invalid policy name: must start with a letter or underscore, followed by letters, digits, or underscores")
	}

	validCommands := map[string]bool{"ALL": true, "SELECT": true, "INSERT": true, "UPDATE": true, "DELETE": true}
	if !validCommands[strings.ToUpper(req.Command)] {
		return SendBadRequest(c, "command must be ALL, SELECT, INSERT, UPDATE, or DELETE")
	}

	// Build CREATE POLICY statement
	permissive := "PERMISSIVE"
	if !req.Permissive {
		permissive = "RESTRICTIVE"
	}

	roles := "PUBLIC"
	if len(req.Roles) > 0 {
		quotedRoles := make([]string, len(req.Roles))
		for i, r := range req.Roles {
			quotedRoles[i] = quoteIdentifier(r)
		}
		roles = strings.Join(quotedRoles, ", ")
	}

	sql := fmt.Sprintf(
		"CREATE POLICY %s ON %s.%s AS %s FOR %s TO %s",
		quoteIdentifier(req.Name),
		quoteIdentifier(req.Schema),
		quoteIdentifier(req.Table),
		permissive,
		strings.ToUpper(req.Command),
		roles,
	)

	if req.Using != "" {
		sql += fmt.Sprintf(" USING (%s)", req.Using)
	}
	if req.WithCheck != "" {
		sql += fmt.Sprintf(" WITH CHECK (%s)", req.WithCheck)
	}

	_, err := s.db.Exec(ctx, sql)
	if err != nil {
		return SendError(c, fiber.StatusBadRequest, "POLICY_CREATE_FAILED", err.Error())
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"sql":     sql,
	})
}

// DeletePolicy drops an RLS policy
// DELETE /api/v1/admin/policies/:schema/:table/:policy
func (s *Server) DeletePolicy(c *fiber.Ctx) error {
	ctx := c.Context()
	schema := c.Params("schema")
	table := c.Params("table")
	policy := c.Params("policy")

	sql := fmt.Sprintf(
		"DROP POLICY %s ON %s.%s",
		quoteIdentifier(policy),
		quoteIdentifier(schema),
		quoteIdentifier(table),
	)

	_, err := s.db.Exec(ctx, sql)
	if err != nil {
		return SendError(c, fiber.StatusBadRequest, "POLICY_DELETE_FAILED", err.Error())
	}

	return c.JSON(fiber.Map{"success": true})
}

// GetSecurityWarnings scans for security issues
// GET /api/v1/admin/security/warnings
func (s *Server) GetSecurityWarnings(c *fiber.Ctx) error {
	ctx := c.Context()
	warnings := []SecurityWarning{}

	// Check 1: Tables in public schema without RLS
	rows1, err := s.db.Query(ctx, `
		SELECT c.relname
		FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = 'public'
		AND c.relkind = 'r'
		AND NOT c.relrowsecurity
		AND c.relname NOT LIKE 'pg_%'
		AND c.relname NOT LIKE '_pg_%'
	`)
	if err == nil {
		defer rows1.Close()
		for rows1.Next() {
			var tableName string
			rows1.Scan(&tableName)
			warnings = append(warnings, SecurityWarning{
				ID:         fmt.Sprintf("no-rls-%s", tableName),
				Severity:   "critical",
				Category:   "rls",
				Schema:     "public",
				Table:      tableName,
				Message:    fmt.Sprintf("Table '%s' does not have Row Level Security enabled", tableName),
				Suggestion: "Enable RLS and create appropriate policies to restrict data access",
				FixSQL:     fmt.Sprintf("ALTER TABLE public.%s ENABLE ROW LEVEL SECURITY;", quoteIdentifier(tableName)),
			})
		}
	}

	// Check 2: RLS enabled but no policies
	rows2, err := s.db.Query(ctx, `
		SELECT n.nspname, c.relname
		FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE c.relrowsecurity = true
		AND c.relkind = 'r'
		AND NOT EXISTS (
			SELECT 1 FROM pg_policies p
			WHERE p.schemaname = n.nspname AND p.tablename = c.relname
		)
		AND n.nspname NOT IN ('pg_catalog', 'information_schema')
	`)
	if err == nil {
		defer rows2.Close()
		for rows2.Next() {
			var schemaName, tableName string
			rows2.Scan(&schemaName, &tableName)
			warnings = append(warnings, SecurityWarning{
				ID:         fmt.Sprintf("no-policies-%s-%s", schemaName, tableName),
				Severity:   "high",
				Category:   "rls",
				Schema:     schemaName,
				Table:      tableName,
				Message:    fmt.Sprintf("Table '%s.%s' has RLS enabled but no policies defined - all access is denied", schemaName, tableName),
				Suggestion: "Create at least one policy to allow intended access patterns",
			})
		}
	}

	// Check 3: Overly permissive policies (USING true for non-SELECT)
	rows3, err := s.db.Query(ctx, `
		SELECT schemaname, tablename, policyname, cmd
		FROM pg_policies
		WHERE qual = 'true'
		AND cmd != 'SELECT'
	`)
	if err == nil {
		defer rows3.Close()
		for rows3.Next() {
			var schemaName, tableName, policyName, cmd string
			rows3.Scan(&schemaName, &tableName, &policyName, &cmd)
			warnings = append(warnings, SecurityWarning{
				ID:         fmt.Sprintf("permissive-%s-%s-%s", schemaName, tableName, policyName),
				Severity:   "high",
				Category:   "policy",
				Schema:     schemaName,
				Table:      tableName,
				PolicyName: policyName,
				Message:    fmt.Sprintf("Policy '%s' on %s.%s uses 'USING (true)' for %s - allows unrestricted access", policyName, schemaName, tableName, cmd),
				Suggestion: "Restrict the USING clause to appropriate conditions",
			})
		}
	}

	// Check 4: Anon role has write access
	rows4, err := s.db.Query(ctx, `
		SELECT schemaname, tablename, policyname, cmd
		FROM pg_policies
		WHERE 'anon' = ANY(roles)
		AND cmd IN ('INSERT', 'UPDATE', 'DELETE', 'ALL')
	`)
	if err == nil {
		defer rows4.Close()
		for rows4.Next() {
			var schemaName, tableName, policyName, cmd string
			rows4.Scan(&schemaName, &tableName, &policyName, &cmd)
			warnings = append(warnings, SecurityWarning{
				ID:         fmt.Sprintf("anon-write-%s-%s-%s", schemaName, tableName, policyName),
				Severity:   "high",
				Category:   "policy",
				Schema:     schemaName,
				Table:      tableName,
				PolicyName: policyName,
				Message:    fmt.Sprintf("Policy '%s' grants %s access to anonymous users", policyName, cmd),
				Suggestion: "Review if anonymous write access is intentional",
			})
		}
	}

	// Check 5: Missing WITH CHECK on INSERT/UPDATE policies
	rows5, err := s.db.Query(ctx, `
		SELECT schemaname, tablename, policyname, cmd
		FROM pg_policies
		WHERE cmd IN ('INSERT', 'UPDATE', 'ALL')
		AND with_check IS NULL
		AND permissive = 'PERMISSIVE'
	`)
	if err == nil {
		defer rows5.Close()
		for rows5.Next() {
			var schemaName, tableName, policyName, cmd string
			rows5.Scan(&schemaName, &tableName, &policyName, &cmd)
			warnings = append(warnings, SecurityWarning{
				ID:         fmt.Sprintf("no-check-%s-%s-%s", schemaName, tableName, policyName),
				Severity:   "medium",
				Category:   "policy",
				Schema:     schemaName,
				Table:      tableName,
				PolicyName: policyName,
				Message:    fmt.Sprintf("Policy '%s' has no WITH CHECK clause for %s operations", policyName, cmd),
				Suggestion: "Add WITH CHECK to validate data on insert/update",
			})
		}
	}

	// Check 6: Tables with sensitive columns but no RLS
	rows6, err := s.db.Query(ctx, `
		SELECT DISTINCT t.table_schema, t.table_name, c.column_name
		FROM information_schema.columns c
		JOIN information_schema.tables t ON t.table_schema = c.table_schema AND t.table_name = c.table_name
		JOIN pg_class pc ON pc.relname = t.table_name
		JOIN pg_namespace pn ON pn.oid = pc.relnamespace AND pn.nspname = t.table_schema
		WHERE t.table_schema = 'public'
		AND t.table_type = 'BASE TABLE'
		AND NOT pc.relrowsecurity
		AND c.column_name ~* '(password|secret|token|api_key|apikey|private_key|credit_card|ssn|social_security)'
	`)
	if err == nil {
		defer rows6.Close()
		for rows6.Next() {
			var schemaName, tableName, columnName string
			rows6.Scan(&schemaName, &tableName, &columnName)
			warnings = append(warnings, SecurityWarning{
				ID:         fmt.Sprintf("sensitive-no-rls-%s-%s-%s", schemaName, tableName, columnName),
				Severity:   "critical",
				Category:   "sensitive-data",
				Schema:     schemaName,
				Table:      tableName,
				Message:    fmt.Sprintf("Table '%s.%s' contains sensitive column '%s' but RLS is not enabled", schemaName, tableName, columnName),
				Suggestion: "Enable RLS immediately to protect sensitive data",
				FixSQL:     fmt.Sprintf("ALTER TABLE %s.%s ENABLE ROW LEVEL SECURITY;", quoteIdentifier(schemaName), quoteIdentifier(tableName)),
			})
		}
	}

	// Check 7: RLS enabled but not forced (table owner can bypass)
	rows7, err := s.db.Query(ctx, `
		SELECT n.nspname, c.relname
		FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE c.relrowsecurity = true
		AND c.relforcerowsecurity = false
		AND c.relkind = 'r'
		AND n.nspname NOT IN ('pg_catalog', 'information_schema', 'auth', 'storage')
	`)
	if err == nil {
		defer rows7.Close()
		for rows7.Next() {
			var schemaName, tableName string
			rows7.Scan(&schemaName, &tableName)
			warnings = append(warnings, SecurityWarning{
				ID:         fmt.Sprintf("no-force-rls-%s-%s", schemaName, tableName),
				Severity:   "medium",
				Category:   "rls",
				Schema:     schemaName,
				Table:      tableName,
				Message:    fmt.Sprintf("Table '%s.%s' has RLS enabled but not forced - table owner can bypass policies", schemaName, tableName),
				Suggestion: "Consider enabling FORCE ROW LEVEL SECURITY for complete protection",
				FixSQL:     fmt.Sprintf("ALTER TABLE %s.%s FORCE ROW LEVEL SECURITY;", quoteIdentifier(schemaName), quoteIdentifier(tableName)),
			})
		}
	}

	// Check 8: Policies that grant access to PUBLIC role
	rows8, err := s.db.Query(ctx, `
		SELECT schemaname, tablename, policyname, cmd
		FROM pg_policies
		WHERE 'public' = ANY(roles)
		AND schemaname NOT IN ('pg_catalog', 'information_schema')
	`)
	if err == nil {
		defer rows8.Close()
		for rows8.Next() {
			var schemaName, tableName, policyName, cmd string
			rows8.Scan(&schemaName, &tableName, &policyName, &cmd)
			warnings = append(warnings, SecurityWarning{
				ID:         fmt.Sprintf("public-access-%s-%s-%s", schemaName, tableName, policyName),
				Severity:   "high",
				Category:   "policy",
				Schema:     schemaName,
				Table:      tableName,
				PolicyName: policyName,
				Message:    fmt.Sprintf("Policy '%s' grants %s access to PUBLIC role (all database users)", policyName, cmd),
				Suggestion: "Restrict access to specific roles like 'authenticated' or 'anon'",
			})
		}
	}

	// Check 9: Tables with CASCADE delete rules that could expose data
	rows9, err := s.db.Query(ctx, `
		SELECT DISTINCT
			tc.table_schema,
			tc.table_name,
			ccu.table_schema as ref_schema,
			ccu.table_name as ref_table,
			rc.delete_rule
		FROM information_schema.table_constraints tc
		JOIN information_schema.referential_constraints rc
			ON tc.constraint_name = rc.constraint_name
		JOIN information_schema.constraint_column_usage ccu
			ON ccu.constraint_name = tc.constraint_name
		WHERE tc.constraint_type = 'FOREIGN KEY'
		AND rc.delete_rule = 'CASCADE'
		AND tc.table_schema = 'public'
	`)
	if err == nil {
		defer rows9.Close()
		for rows9.Next() {
			var schemaName, tableName, refSchema, refTable, deleteRule string
			rows9.Scan(&schemaName, &tableName, &refSchema, &refTable, &deleteRule)
			warnings = append(warnings, SecurityWarning{
				ID:         fmt.Sprintf("cascade-delete-%s-%s-%s-%s", schemaName, tableName, refSchema, refTable),
				Severity:   "low",
				Category:   "data-integrity",
				Schema:     schemaName,
				Table:      tableName,
				Message:    fmt.Sprintf("Table '%s.%s' has CASCADE DELETE to '%s.%s' - deleting parent rows will delete child data", schemaName, tableName, refSchema, refTable),
				Suggestion: "Review if CASCADE DELETE is appropriate; consider RESTRICT or SET NULL for better data protection",
			})
		}
	}

	// Check 10: Tables without user ownership pattern (no user_id/owner_id column)
	rows10, err := s.db.Query(ctx, `
		SELECT t.table_schema, t.table_name
		FROM information_schema.tables t
		JOIN pg_class c ON c.relname = t.table_name
		JOIN pg_namespace n ON n.oid = c.relnamespace AND n.nspname = t.table_schema
		WHERE t.table_schema = 'public'
		AND t.table_type = 'BASE TABLE'
		AND c.relrowsecurity = true
		AND NOT EXISTS (
			SELECT 1 FROM information_schema.columns col
			WHERE col.table_schema = t.table_schema
			AND col.table_name = t.table_name
			AND col.column_name ~* '(user_id|owner_id|created_by|author_id)'
		)
		AND t.table_name NOT IN ('_migrations', 'schema_migrations')
	`)
	if err == nil {
		defer rows10.Close()
		for rows10.Next() {
			var schemaName, tableName string
			rows10.Scan(&schemaName, &tableName)
			warnings = append(warnings, SecurityWarning{
				ID:         fmt.Sprintf("no-owner-column-%s-%s", schemaName, tableName),
				Severity:   "medium",
				Category:   "design",
				Schema:     schemaName,
				Table:      tableName,
				Message:    fmt.Sprintf("Table '%s.%s' has RLS but no user_id/owner_id column - row-level ownership may be difficult", schemaName, tableName),
				Suggestion: "Consider adding a user_id or owner_id column to enable user-based access control",
			})
		}
	}

	// Calculate summary
	summary := struct {
		Total    int `json:"total"`
		Critical int `json:"critical"`
		High     int `json:"high"`
		Medium   int `json:"medium"`
		Low      int `json:"low"`
	}{}

	for _, w := range warnings {
		summary.Total++
		switch w.Severity {
		case "critical":
			summary.Critical++
		case "high":
			summary.High++
		case "medium":
			summary.Medium++
		case "low":
			summary.Low++
		}
	}

	return c.JSON(fiber.Map{
		"warnings": warnings,
		"summary":  summary,
	})
}

// GetPolicyTemplates returns pre-built policy templates
// GET /api/v1/admin/policies/templates
func (s *Server) GetPolicyTemplates(c *fiber.Ctx) error {
	templates := []fiber.Map{
		{
			"id":          "user-owns-row",
			"name":        "User can only access own rows",
			"description": "Restricts access to rows where the user_id column matches the authenticated user",
			"command":     "ALL",
			"using":       "auth.uid() = user_id",
			"with_check":  "auth.uid() = user_id",
		},
		{
			"id":          "authenticated-read",
			"name":        "Authenticated users can read all",
			"description": "Allows any authenticated user to read all rows",
			"command":     "SELECT",
			"using":       "auth.role() = 'authenticated'",
			"with_check":  "",
		},
		{
			"id":          "public-read",
			"name":        "Public read-only access",
			"description": "Allows anyone (including anonymous) to read all rows",
			"command":     "SELECT",
			"using":       "true",
			"with_check":  "",
		},
		{
			"id":          "admin-full-access",
			"name":        "Admin full access",
			"description": "Allows admin users full access to all rows",
			"command":     "ALL",
			"using":       "auth.jwt() ->> 'role' = 'admin'",
			"with_check":  "auth.jwt() ->> 'role' = 'admin'",
		},
		{
			"id":          "owner-modify",
			"name":        "Owner can modify, others can read",
			"description": "Owner can read/write, others can only read",
			"command":     "ALL",
			"using":       "true",
			"with_check":  "auth.uid() = owner_id",
		},
		{
			"id":          "team-access",
			"name":        "Team-based access",
			"description": "Users can access rows belonging to their team",
			"command":     "ALL",
			"using":       "team_id IN (SELECT team_id FROM team_members WHERE user_id = auth.uid())",
			"with_check":  "team_id IN (SELECT team_id FROM team_members WHERE user_id = auth.uid())",
		},
	}

	return c.JSON(templates)
}
