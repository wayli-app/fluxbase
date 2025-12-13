package rpc

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Annotation patterns for parsing SQL comments
var (
	namePattern           = regexp.MustCompile(`(?m)^--\s*@fluxbase:name\s+(.+)$`)
	descriptionPattern    = regexp.MustCompile(`(?m)^--\s*@fluxbase:description\s+(.+)$`)
	inputPattern          = regexp.MustCompile(`(?m)^--\s*@fluxbase:input\s+(.+)$`)
	outputPattern         = regexp.MustCompile(`(?m)^--\s*@fluxbase:output\s+(.+)$`)
	allowedTablesPattern  = regexp.MustCompile(`(?m)^--\s*@fluxbase:allowed-tables\s+(.+)$`)
	allowedSchemasPattern = regexp.MustCompile(`(?m)^--\s*@fluxbase:allowed-schemas\s+(.+)$`)
	maxExecTimePattern    = regexp.MustCompile(`(?m)^--\s*@fluxbase:max-execution-time\s+(.+)$`)
	requireRolePattern    = regexp.MustCompile(`(?m)^--\s*@fluxbase:require-role\s+(.+)$`)
	publicPattern         = regexp.MustCompile(`(?m)^--\s*@fluxbase:public\s+(.+)$`)
	versionPattern        = regexp.MustCompile(`(?m)^--\s*@fluxbase:version\s+(.+)$`)
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

	// Parse require role
	if matches := requireRolePattern.FindStringSubmatch(code); len(matches) > 1 {
		annotations.RequireRole = strings.TrimSpace(matches[1])
	}

	// Parse public flag
	if matches := publicPattern.FindStringSubmatch(code); len(matches) > 1 {
		value := strings.ToLower(strings.TrimSpace(matches[1]))
		annotations.IsPublic = value == "true" || value == "yes" || value == "1"
	}

	// Parse version
	if matches := versionPattern.FindStringSubmatch(code); len(matches) > 1 {
		if v, err := strconv.Atoi(strings.TrimSpace(matches[1])); err == nil {
			annotations.Version = v
		}
	}

	// Extract SQL query (everything that's not an annotation line)
	sqlQuery := extractSQLQuery(code)

	return annotations, sqlQuery, nil
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
	if annotations.RequireRole != "" {
		proc.RequireRole = &annotations.RequireRole
	}
	proc.IsPublic = annotations.IsPublic
	if annotations.Version > 0 {
		proc.Version = annotations.Version
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
