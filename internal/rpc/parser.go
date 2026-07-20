package rpc

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/nimbleflux/fluxbase/internal/loader"
)

// Annotation patterns for parsing SQL comments
var (
	namePattern                 = regexp.MustCompile(`(?m)^--\s*@fluxbase:name\s+(.+)$`)
	descriptionPattern          = regexp.MustCompile(`(?m)^--\s*@fluxbase:description\s+(.+)$`)
	inputPattern                = regexp.MustCompile(`(?m)^--\s*@fluxbase:input\s+(.+)$`)
	outputPattern               = regexp.MustCompile(`(?m)^--\s*@fluxbase:output\s+(.+)$`)
	allowedTablesPattern        = regexp.MustCompile(`(?m)^--\s*@fluxbase:allowed-tables\s+(.+)$`)
	allowedSchemasPattern       = regexp.MustCompile(`(?m)^--\s*@fluxbase:allowed-schemas\s+(.+)$`)
	maxExecTimePattern          = regexp.MustCompile(`(?m)^--\s*@fluxbase:max-execution-time\s+(.+)$`)
	requireRolePattern          = regexp.MustCompile(`(?m)^--\s*@fluxbase:require-role\s+(.+)$`)
	publicPattern               = regexp.MustCompile(`(?m)^--\s*@fluxbase:public\s+(.+)$`)
	disableExecutionLogsPattern = regexp.MustCompile(`(?m)^--\s*@fluxbase:disable-execution-logs(?:\s+(true|false))?$`)
	versionPattern              = regexp.MustCompile(`(?m)^--\s*@fluxbase:version\s+(.+)$`)
	schedulePattern             = regexp.MustCompile(`(?m)^--\s*@fluxbase:schedule\s+(.+)$`)
)

// ParseAnnotations parses annotations from SQL code and returns the annotations and cleaned SQL query
func ParseAnnotations(code string) (*Annotations, string, error) {
	annotations := DefaultAnnotations()

	// Parse name
	if matches := namePattern.FindStringSubmatch(code); len(matches) > 1 {
		annotations.Name = strings.TrimSpace(matches[1])
	}

	// Parse description
	if matches := descriptionPattern.FindStringSubmatch(code); len(matches) > 1 {
		annotations.Description = strings.TrimSpace(matches[1])
	}

	// Parse input schema
	if matches := inputPattern.FindStringSubmatch(code); len(matches) > 1 {
		input := strings.TrimSpace(matches[1])
		// ponytail: multi-line @fluxbase:input blocks. The single-line regex
		// only captures the first line — for multi-line JSON the first line
		// is just "{" or "{ "field?":"text",". extractJSONObject scans forward
		// in the source collecting subsequent "-- ..." lines until braces
		// balance, then we parse the assembled string as JSON.
		if input == "{" || (strings.HasPrefix(input, "{") && !strings.HasSuffix(strings.TrimSpace(input), "}")) {
			input = extractMultilineAnnotationValue(code, inputPattern, input)
		}
		if input != "any" && input != "" {
			schema, err := parseSchemaString(input)
			if err == nil {
				annotations.InputSchema = schema
			}
		}
	}

	// Parse output schema
	if matches := outputPattern.FindStringSubmatch(code); len(matches) > 1 {
		output := strings.TrimSpace(matches[1])
		if output == "{" || (strings.HasPrefix(output, "{") && !strings.HasSuffix(strings.TrimSpace(output), "}")) {
			output = extractMultilineAnnotationValue(code, outputPattern, output)
		}
		if output != "any" && output != "" {
			schema, err := parseSchemaString(output)
			if err == nil {
				annotations.OutputSchema = schema
			}
		}
	}

	// Parse allowed tables
	if matches := allowedTablesPattern.FindStringSubmatch(code); len(matches) > 1 {
		tables := parseCommaSeparatedList(matches[1])
		if len(tables) > 0 {
			annotations.AllowedTables = tables
		}
	}

	// Parse allowed schemas
	if matches := allowedSchemasPattern.FindStringSubmatch(code); len(matches) > 1 {
		schemas := parseCommaSeparatedList(matches[1])
		if len(schemas) > 0 {
			annotations.AllowedSchemas = schemas
		}
	}

	// Parse max execution time
	if matches := maxExecTimePattern.FindStringSubmatch(code); len(matches) > 1 {
		duration, err := parseDuration(strings.TrimSpace(matches[1]))
		if err == nil {
			annotations.MaxExecutionTime = duration
		}
	}

	// Parse require role (supports comma-separated list of roles)
	if matches := requireRolePattern.FindStringSubmatch(code); len(matches) > 1 {
		rolesStr := strings.TrimSpace(matches[1])
		var roles []string
		for _, role := range strings.Split(rolesStr, ",") {
			role = strings.TrimSpace(role)
			if role != "" {
				roles = append(roles, role)
			}
		}
		if len(roles) > 0 {
			annotations.RequireRoles = roles
		}
	}

	// Parse public flag
	if matches := publicPattern.FindStringSubmatch(code); len(matches) > 1 {
		value := strings.ToLower(strings.TrimSpace(matches[1]))
		annotations.IsPublic = value == "true" || value == "yes" || value == "1"
	}

	// Parse disable-execution-logs flag
	if matches := disableExecutionLogsPattern.FindStringSubmatch(code); matches != nil {
		// If no value specified or value is "true", disable logs
		if len(matches) <= 1 || matches[1] == "" || matches[1] == "true" {
			annotations.DisableExecutionLogs = true
		}
	}

	// Parse version
	if matches := versionPattern.FindStringSubmatch(code); len(matches) > 1 {
		if v, err := strconv.Atoi(strings.TrimSpace(matches[1])); err == nil {
			annotations.Version = v
		}
	}

	// Parse schedule (cron expression)
	if matches := schedulePattern.FindStringSubmatch(code); len(matches) > 1 {
		schedule := strings.TrimSpace(matches[1])
		if schedule != "" {
			annotations.Schedule = &schedule
		}
	}

	// Extract SQL query (everything that's not an annotation line)
	sqlQuery := extractSQLQuery(code)

	return annotations, sqlQuery, nil
}

// extractMultilineAnnotationValue assembles a multi-line JSON annotation value
// when the single-line regex only captured the opening brace(s).
//
// Background: inputPattern/outputPattern use `^(?m)--\s*@fluxbase:input (.+)$`
// which matches only the first line. For a multi-line spec like:
//
//	-- @fluxbase:input {
//	--   "trip_id?": "uuid",
//	--   "trip_title?": "text"
//	-- }
//
// the regex captures only "{", which fails JSON parsing. This helper finds
// the annotation's start position in `code`, then scans forward collecting
// subsequent `-- ...` lines (stripping the comment prefix) until the JSON
// braces balance, and returns the assembled single-string JSON.
//
// `initial` is the value already captured by the regex; returned as-is when
// braces already balance (the common single-line case).
func extractMultilineAnnotationValue(code string, startRe *regexp.Regexp, initial string) string {
	if balancedBraces(initial) {
		return initial
	}
	loc := startRe.FindStringSubmatchIndex(code)
	if loc == nil || len(loc) < 4 {
		return initial
	}
	// Submatch index layout: [fullStart, fullEnd, groupStart, groupEnd].
	// Group 1 is the captured value.
	groupStart, groupEnd := loc[2], loc[3]
	if groupStart < 0 || groupEnd < 0 || groupEnd > len(code) {
		return initial
	}

	// Scan forward from the end of the captured first line.
	var sb strings.Builder
	sb.WriteString(code[groupStart:groupEnd])

	// Walk subsequent lines. Each must start with `--` (after optional
	// leading whitespace) to count as a continuation of the annotation.
	// Stop at the first non-comment line or when braces balance.
	pos := groupEnd
	for pos < len(code) {
		// Find the next newline.
		nl := strings.IndexByte(code[pos:], '\n')
		var line string
		if nl < 0 {
			line = code[pos:]
			pos = len(code)
		} else {
			line = code[pos : pos+nl]
			pos += nl + 1
		}
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		// Must be a SQL comment line (`-- ...`).
		if !strings.HasPrefix(trimmed, "--") {
			break
		}
		content := strings.TrimSpace(strings.TrimPrefix(trimmed, "--"))
		// Stop if this is a new annotation (`@fluxbase:`).
		if strings.HasPrefix(content, "@fluxbase:") {
			break
		}
		sb.WriteString(content)
		if balancedBraces(sb.String()) {
			break
		}
	}
	assembled := strings.TrimSpace(sb.String())
	if balancedBraces(assembled) {
		return assembled
	}
	return initial
}

// balancedBraces returns true if `s` has balanced `{` and `}` characters,
// ignoring braces inside double-quoted strings. Good enough for the JSON
// annotation use case — we don't need a full JSON parser here.
func balancedBraces(s string) bool {
	depth := 0
	inString := false
	escaped := false
	for _, r := range s {
		if escaped {
			escaped = false
			continue
		}
		switch r {
		case '\\':
			if inString {
				escaped = true
			}
		case '"':
			inString = !inString
		case '{':
			if !inString {
				depth++
			}
		case '}':
			if !inString {
				depth--
				if depth < 0 {
					return false
				}
			}
		}
	}
	return depth == 0 && !inString
}

// parseSchemaString parses a JSON-like schema string into a map
// Supports: {"field": "type", "optional_field?": "type"}
func parseSchemaString(input string) (map[string]string, error) {
	input = strings.TrimSpace(input)

	// Try parsing as JSON first
	var schema map[string]string
	if err := json.Unmarshal([]byte(input), &schema); err == nil {
		return schema, nil
	}

	// Fallback: try parsing simple format like "field1:type1, field2:type2"
	schema = make(map[string]string)
	parts := strings.Split(input, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		kv := strings.SplitN(part, ":", 2)
		if len(kv) == 2 {
			key := strings.TrimSpace(kv[0])
			value := strings.TrimSpace(kv[1])
			schema[key] = value
		}
	}

	if len(schema) == 0 {
		return nil, nil
	}
	return schema, nil
}

// parseCommaSeparatedList parses a comma-separated list of values
func parseCommaSeparatedList(input string) []string {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil
	}

	parts := strings.Split(input, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

// parseDuration parses a duration string like "30s", "5m", "1h"
func parseDuration(input string) (time.Duration, error) {
	input = strings.TrimSpace(input)

	// Try standard Go duration parsing first
	if d, err := time.ParseDuration(input); err == nil {
		return d, nil
	}

	// Handle simple formats without unit suffix (assume seconds)
	if v, err := strconv.Atoi(input); err == nil {
		return time.Duration(v) * time.Second, nil
	}

	return 0, nil
}

// extractSQLQuery removes annotation lines and returns the SQL query
func extractSQLQuery(code string) string {
	lines := strings.Split(code, "\n")
	var sqlLines []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Skip annotation lines
		if strings.HasPrefix(trimmed, "-- @fluxbase:") {
			continue
		}
		sqlLines = append(sqlLines, line)
	}

	// Trim leading/trailing empty lines
	result := strings.TrimSpace(strings.Join(sqlLines, "\n"))
	return result
}

// ApplyAnnotations applies parsed annotations to a Procedure
func ApplyAnnotations(proc *Procedure, annotations *Annotations) {
	if annotations.Name != "" {
		proc.Name = annotations.Name
	}
	if annotations.Description != "" {
		proc.Description = annotations.Description
	}
	if len(annotations.AllowedTables) > 0 {
		proc.AllowedTables = annotations.AllowedTables
	}
	if len(annotations.AllowedSchemas) > 0 {
		proc.AllowedSchemas = annotations.AllowedSchemas
	}
	if annotations.MaxExecutionTime > 0 {
		proc.MaxExecutionTimeSeconds = int(annotations.MaxExecutionTime.Seconds())
	}
	if len(annotations.RequireRoles) > 0 {
		proc.RequireRoles = annotations.RequireRoles
	}
	proc.IsPublic = annotations.IsPublic
	proc.DisableExecutionLogs = annotations.DisableExecutionLogs
	if annotations.Version > 0 {
		proc.Version = annotations.Version
	}
	if annotations.Schedule != nil {
		proc.Schedule = annotations.Schedule
	}

	// Convert input/output schemas to JSON
	if annotations.InputSchema != nil {
		if data, err := json.Marshal(annotations.InputSchema); err == nil {
			proc.InputSchema = data
		}
	}
	if annotations.OutputSchema != nil {
		if data, err := json.Marshal(annotations.OutputSchema); err == nil {
			proc.OutputSchema = data
		}
	}
}

// SchemaTypeToGoType maps schema type names to Go/PostgreSQL types
func SchemaTypeToGoType(schemaType string) string {
	switch strings.ToLower(schemaType) {
	case "uuid":
		return "uuid"
	case "string", "text":
		return "text"
	case "number", "int", "integer":
		return "integer"
	case "float", "double", "decimal":
		return "numeric"
	case "boolean", "bool":
		return "boolean"
	case "timestamp", "datetime":
		return "timestamptz"
	case "date":
		return "date"
	case "time":
		return "time"
	case "json", "jsonb", "object":
		return "jsonb"
	case "array":
		return "jsonb"
	default:
		return "text"
	}
}

// IsOptionalField checks if a field name indicates it's optional (ends with ?)
func IsOptionalField(fieldName string) bool {
	return strings.HasSuffix(fieldName, "?")
}

// CleanFieldName removes the optional marker from a field name
func CleanFieldName(fieldName string) string {
	return strings.TrimSuffix(fieldName, "?")
}

type RPCAnnotations struct {
	Description    string
	IsPublic       bool
	Timeout        int
	RequireRoles   []string
	AllowedTables  []string
	AllowedSchemas []string
	Schedule       *string
	DisableLogs    bool
}

func ParseRPCAnnotations(sqlCode string) RPCAnnotations {
	annotations := loader.ParseAnnotations(sqlCode, []string{"--"})
	config := RPCAnnotations{
		Timeout:        30,
		AllowedSchemas: []string{"public"},
		AllowedTables:  []string{},
	}

	if v, ok := annotations["description"]; ok {
		config.Description = v
	}
	if _, ok := annotations["public"]; ok {
		config.IsPublic = true
	}
	if v, ok := annotations["timeout"]; ok {
		if t, err := strconv.Atoi(v); err == nil && t > 0 {
			config.Timeout = t
		}
	}
	if v, ok := annotations["require-role"]; ok {
		roles := loader.ParseRoleList(v)
		if len(roles) > 0 {
			config.RequireRoles = roles
		}
	}
	if v, ok := annotations["allowed-tables"]; ok {
		tables := loader.ParseCommaList(v)
		if len(tables) > 0 {
			config.AllowedTables = tables
		}
	}
	if v, ok := annotations["allowed-schemas"]; ok {
		schemas := loader.ParseCommaList(v)
		if len(schemas) > 0 {
			config.AllowedSchemas = schemas
		}
	}
	if v, ok := annotations["schedule"]; ok {
		schedule := strings.Trim(v, "\"'")
		if schedule != "" {
			config.Schedule = &schedule
		}
	}
	if _, ok := annotations["disable-logs"]; ok {
		config.DisableLogs = true
	}

	return config
}
