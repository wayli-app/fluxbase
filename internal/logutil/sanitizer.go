// Package logutil provides logging utilities for sanitization
package logutil

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	singleQuoteRe = regexp.MustCompile(`'(?:[^']|'')*'`)
	paramRe       = regexp.MustCompile(`\$\d+`)
	dollarQuoteRe = regexp.MustCompile(`\$\$[^$]*\$\$`)
	dollarTagRe   = regexp.MustCompile(`\$[a-zA-Z0-9_]+\$.*?\$[a-zA-Z0-9_]+\$`)
	numericRe     = regexp.MustCompile(`\b\d+(?:\.\d+)?(?:[eE][+-]?\d+)?\b`)
	ipRe          = regexp.MustCompile(`\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\b`)
	uuidRe        = regexp.MustCompile(`\b[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}\b`)
)

// SanitizeSQL removes sensitive data from SQL queries by replacing literal values
// with placeholders. This prevents passwords, PII, and other sensitive data from
// appearing in logs.
//
// Replacements:
// - String literals (single quotes): '<redacted>'
// - Numeric literals: <num>
// - Boolean values (TRUE/FALSE): <bool>
// - NULL: <null>
// - $1, $2, etc. parameter placeholders: kept as-is
//
// Example:
//
//	SELECT * FROM users WHERE email = 'user@example.com' AND id = 123
//	=> SELECT * FROM users WHERE email = '<redacted>' AND id = <num>
func SanitizeSQL(query string) string {
	query = singleQuoteRe.ReplaceAllString(query, "'<redacted>'")

	params := paramRe.FindAllString(query, -1)
	for i, param := range params {
		query = strings.Replace(query, param, "\x00PARAM"+fmt.Sprint(i)+"\x00", 1)
	}

	query = dollarQuoteRe.ReplaceAllString(query, "$$$$<redacted>$$$$")

	query = dollarTagRe.ReplaceAllString(query, "$$<redacted>$$")

	query = numericRe.ReplaceAllString(query, "<num>")

	for i, param := range params {
		query = strings.Replace(query, "\x00PARAM"+fmt.Sprint(i)+"\x00", param, 1)
	}

	query = strings.ReplaceAll(query, " TRUE", " <bool>")
	query = strings.ReplaceAll(query, " FALSE", " <bool>")
	query = strings.ReplaceAll(query, " NULL", " <null>")

	query = ipRe.ReplaceAllString(query, "<ip>")

	query = uuidRe.ReplaceAllString(query, "<uuid>")

	return query
}

// ExtractDDLMetadata extracts the operation type and object name from a DDL statement.
// This returns only the operation metadata for logging, not the full DDL statement
// which may contain sensitive schema details.
//
// Example inputs and outputs:
//
//	CREATE TABLE users (id SERIAL, name TEXT)
//	=> "CREATE TABLE users"
//
//	ALTER TABLE users ADD COLUMN email TEXT
//	=> "ALTER TABLE users ADD COLUMN"
//
//	DROP INDEX CONCURRENTLY idx_users_email
//	=> "DROP INDEX idx_users_email"
//
//	CREATE OR REPLACE FUNCTION get_user(id INTEGER)
//	=> "CREATE FUNCTION get_user"
func ExtractDDLMetadata(ddl string) string {
	ddl = strings.TrimSpace(ddl)
	words := strings.Fields(ddl)
	if len(words) == 0 {
		return ""
	}

	// Get the operation type (first word)
	operation := strings.ToUpper(words[0])

	// Handle CREATE OR REPLACE / CREATE [UNIQUE] INDEX
	var metadata string
	switch operation {
	case "CREATE":
		metadata = extractCreateMetadata(words)
	case "ALTER":
		metadata = extractAlterMetadata(words)
	case "DROP":
		metadata = extractDropMetadata(words)
	case "TRUNCATE":
		metadata = extractTruncateMetadata(words)
	case "RENAME":
		metadata = extractRenameMetadata(words)
	case "GRANT", "REVOKE":
		metadata = extractGrantRevokeMetadata(words)
	case "COMMENT":
		metadata = extractCommentMetadata(words)
	default:
		// For unknown DDL, return first 3 words max
		maxWords := 3
		if len(words) < maxWords {
			maxWords = len(words)
		}
		metadata = strings.Join(words[:maxWords], " ")
	}

	return metadata
}

// extractCreateMetadata extracts metadata from CREATE statements
func extractCreateMetadata(words []string) string {
	if len(words) < 2 {
		return "CREATE"
	}

	// Handle CREATE [OR REPLACE] [MATERIALIZED] VIEW / TABLE / INDEX / FUNCTION / TRIGGER / SCHEMA
	// Skip "OR REPLACE" if present
	idx := 1
	if idx < len(words) && strings.ToUpper(words[idx]) == "OR" {
		idx += 2 // Skip "REPLACE"
	}

	// Skip "MATERIALIZED" if present
	if idx < len(words) && strings.ToUpper(words[idx]) == "MATERIALIZED" {
		idx++
	}

	if idx >= len(words) {
		return "CREATE"
	}

	objectType := strings.ToUpper(words[idx])

	// For CREATE [UNIQUE] INDEX, skip UNIQUE
	if objectType == "UNIQUE" {
		idx++
		if idx < len(words) {
			objectType = strings.ToUpper(words[idx])
		}
	}

	// Get object name if available
	idx++
	if idx < len(words) {
		objectName := words[idx]
		// Handle IF NOT EXISTS - skip to get actual name
		if strings.ToUpper(objectName) == "IF" {
			idx += 3 // Skip "IF NOT EXISTS"
			if idx < len(words) {
				objectName = words[idx]
			} else {
				objectName = ""
			}
		}
		// Strip quotes from object name (e.g., "uuid-ossp" -> uuid-ossp)
		objectName = strings.Trim(objectName, `"`)
		// Strip function parameters (e.g., get_user(id -> get_user)
		if idx := strings.Index(objectName, "("); idx != -1 {
			objectName = objectName[:idx]
		}
		if objectName != "" {
			return "CREATE " + objectType + " " + objectName
		}
	}

	return "CREATE " + objectType
}

// extractAlterMetadata extracts metadata from ALTER statements
func extractAlterMetadata(words []string) string {
	if len(words) < 3 {
		return "ALTER"
	}

	objectType := strings.ToUpper(words[1])
	objectName := words[2]
	action := ""

	// Find the action (ADD, DROP, ALTER, RENAME COLUMN, etc.)
	for i := 3; i < len(words); i++ {
		word := strings.ToUpper(words[i])
		if word == "ADD" || word == "DROP" || word == "ALTER" || word == "RENAME" {
			// Handle "RENAME COLUMN"
			if word == "RENAME" && i+1 < len(words) && strings.ToUpper(words[i+1]) == "COLUMN" {
				action = "RENAME COLUMN"
				break
			}
			// Handle "ADD CONSTRAINT", "DROP CONSTRAINT"
			if (word == "ADD" || word == "DROP") && i+1 < len(words) && strings.ToUpper(words[i+1]) == "CONSTRAINT" {
				action = word + " CONSTRAINT"
				break
			}
			action = word
			break
		}
	}

	if action != "" {
		return "ALTER " + objectType + " " + objectName + " " + action
	}
	return "ALTER " + objectType + " " + objectName
}

// extractDropMetadata extracts metadata from DROP statements
func extractDropMetadata(words []string) string {
	if len(words) < 2 {
		return "DROP"
	}

	objectType := strings.ToUpper(words[1])

	// Handle DROP [IF EXISTS]
	idx := 2
	if idx < len(words) && strings.ToUpper(words[idx]) == "IF" {
		idx += 2 // Skip "IF EXISTS"
	}

	// Get object name if available
	var objectName string
	if idx < len(words) {
		objectName = words[idx]

		// Handle CONCURRENTLY for indexes
		if objectType == "INDEX" && strings.ToUpper(objectName) == "CONCURRENTLY" {
			idx++
			if idx < len(words) {
				objectName = words[idx]
			} else {
				objectName = ""
			}
		}
	}

	if objectName != "" {
		return "DROP " + objectType + " " + objectName
	}
	return "DROP " + objectType
}

// extractTruncateMetadata extracts metadata from TRUNCATE statements
func extractTruncateMetadata(words []string) string {
	if len(words) < 2 {
		return "TRUNCATE"
	}

	// TRUNCATE [TABLE] [ONLY] table_name
	idx := 1
	if idx < len(words) && strings.ToUpper(words[idx]) == "TABLE" {
		idx++
	}
	if idx < len(words) && strings.ToUpper(words[idx]) == "ONLY" {
		idx++
	}

	if idx < len(words) {
		tableName := words[idx]
		return "TRUNCATE TABLE " + tableName
	}

	return "TRUNCATE TABLE"
}

// extractRenameMetadata extracts metadata from RENAME statements
func extractRenameMetadata(words []string) string {
	if len(words) < 4 {
		return "RENAME"
	}

	// RENAME [TABLE | INDEX | COLUMN] old_name TO new_name
	objectType := strings.ToUpper(words[1])
	oldName := words[2]

	return "RENAME " + objectType + " " + oldName
}

// extractGrantRevokeMetadata extracts metadata from GRANT/REVOKE statements
func extractGrantRevokeMetadata(words []string) string {
	if len(words) < 3 {
		return words[0]
	}

	// GRANT privileges ON object_type object_name TO role
	// REVOKE [GRANT OPTION FOR] privileges ON object_type object_name FROM role

	operation := strings.ToUpper(words[0])

	// Skip privileges to find ON clause
	idx := 1
	for idx < len(words) && strings.ToUpper(words[idx]) != "ON" {
		idx++
	}

	if idx+1 >= len(words) {
		return operation
	}

	objectType := strings.ToUpper(words[idx+1])
	objectName := ""
	if idx+2 < len(words) {
		objectName = words[idx+2]
	}

	if objectName != "" {
		return operation + " ON " + objectType + " " + objectName
	}
	return operation + " ON " + objectType
}

// extractCommentMetadata extracts metadata from COMMENT statements
func extractCommentMetadata(words []string) string {
	if len(words) < 4 {
		return "COMMENT"
	}

	// COMMENT ON object_type object_name IS 'text'
	objectType := strings.ToUpper(words[2])
	objectName := words[3]

	return "COMMENT ON " + objectType + " " + objectName
}
