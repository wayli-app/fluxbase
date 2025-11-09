package realtime

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Filter represents a Supabase-compatible filter for realtime subscriptions
// Format: column=operator.value
// Examples:
//   - created_by=eq.user123
//   - priority=gt.5
//   - status=in.(queued,running)
type Filter struct {
	Column   string
	Operator string
	Value    string
}

// filterRegex matches the Supabase filter format: column=operator.value
var filterRegex = regexp.MustCompile(`^(\w+)=(eq|neq|gt|gte|lt|lte|like|ilike|is|in)\.(.+)$`)

// ParseFilter parses a Supabase-compatible filter string
// Returns nil if filterStr is empty (no filter)
func ParseFilter(filterStr string) (*Filter, error) {
	if filterStr == "" {
		return nil, nil
	}

	matches := filterRegex.FindStringSubmatch(filterStr)
	if matches == nil || len(matches) != 4 {
		return nil, fmt.Errorf("invalid filter format: %s (expected: column=operator.value)", filterStr)
	}

	return &Filter{
		Column:   matches[1],
		Operator: matches[2],
		Value:    matches[3],
	}, nil
}

// Matches checks if a record matches this filter
// Returns true if the filter is nil (no filtering)
// Returns false if the column doesn't exist in the record
func (f *Filter) Matches(record map[string]interface{}) bool {
	if f == nil {
		return true // No filter means match all
	}

	recordValue, exists := record[f.Column]
	if !exists {
		return false
	}

	switch f.Operator {
	case "eq":
		return f.matchesEquality(recordValue, f.Value, true)

	case "neq":
		return f.matchesEquality(recordValue, f.Value, false)

	case "gt":
		return f.matchesNumericComparison(recordValue, f.Value, func(cmp int) bool { return cmp > 0 })

	case "gte":
		return f.matchesNumericComparison(recordValue, f.Value, func(cmp int) bool { return cmp >= 0 })

	case "lt":
		return f.matchesNumericComparison(recordValue, f.Value, func(cmp int) bool { return cmp < 0 })

	case "lte":
		return f.matchesNumericComparison(recordValue, f.Value, func(cmp int) bool { return cmp <= 0 })

	case "is":
		return f.matchesIs(recordValue, f.Value)

	case "in":
		return f.matchesIn(recordValue, f.Value)

	case "like":
		return f.matchesPattern(recordValue, f.Value, false)

	case "ilike":
		return f.matchesPattern(recordValue, f.Value, true)

	default:
		return false
	}
}

// matchesEquality checks equality/inequality
func (f *Filter) matchesEquality(recordValue interface{}, filterValue string, shouldEqual bool) bool {
	recordStr := fmt.Sprint(recordValue)
	matches := recordStr == filterValue

	if shouldEqual {
		return matches
	}
	return !matches
}

// matchesNumericComparison compares numeric values
func (f *Filter) matchesNumericComparison(recordValue interface{}, filterValue string, compareFn func(int) bool) bool {
	cmp := compareNumeric(recordValue, filterValue)
	return compareFn(cmp)
}

// matchesIs handles IS operator (for null and boolean values)
func (f *Filter) matchesIs(recordValue interface{}, filterValue string) bool {
	switch strings.ToLower(filterValue) {
	case "null":
		return recordValue == nil

	case "true":
		if b, ok := recordValue.(bool); ok {
			return b
		}
		return fmt.Sprint(recordValue) == "true"

	case "false":
		if b, ok := recordValue.(bool); ok {
			return !b
		}
		return fmt.Sprint(recordValue) == "false"

	default:
		return false
	}
}

// matchesIn checks if record value is in the list
// Format: (value1,value2,value3)
func (f *Filter) matchesIn(recordValue interface{}, filterValue string) bool {
	// Remove parentheses and split by comma
	listStr := strings.Trim(filterValue, "()")
	if listStr == "" {
		return false
	}

	values := strings.Split(listStr, ",")
	recordStr := fmt.Sprint(recordValue)

	for _, v := range values {
		if strings.TrimSpace(v) == recordStr {
			return true
		}
	}

	return false
}

// matchesPattern matches LIKE/ILIKE patterns
// Supabase uses * as wildcard (converted to SQL %)
func (f *Filter) matchesPattern(recordValue interface{}, pattern string, caseInsensitive bool) bool {
	recordStr := fmt.Sprint(recordValue)

	if caseInsensitive {
		recordStr = strings.ToLower(recordStr)
		pattern = strings.ToLower(pattern)
	}

	// Convert Supabase wildcard (*) to regex pattern
	// * -> .*
	// ? -> . (optional, not in Supabase spec but commonly used)
	regexPattern := regexp.QuoteMeta(pattern)
	regexPattern = strings.ReplaceAll(regexPattern, `\*`, ".*")
	regexPattern = "^" + regexPattern + "$"

	matched, err := regexp.MatchString(regexPattern, recordStr)
	if err != nil {
		return false
	}

	return matched
}

// compareNumeric compares two values numerically
// Returns: -1 if recordValue < filterValue, 0 if equal, 1 if greater
func compareNumeric(recordValue interface{}, filterValue string) int {
	// Parse filter value as float
	filterFloat, err := strconv.ParseFloat(filterValue, 64)
	if err != nil {
		// If filter value is not numeric, compare as strings
		recordStr := fmt.Sprint(recordValue)
		if recordStr < filterValue {
			return -1
		} else if recordStr > filterValue {
			return 1
		}
		return 0
	}

	// Convert record value to float
	var recordFloat float64

	switch v := recordValue.(type) {
	case float64:
		recordFloat = v
	case float32:
		recordFloat = float64(v)
	case int:
		recordFloat = float64(v)
	case int64:
		recordFloat = float64(v)
	case int32:
		recordFloat = float64(v)
	case string:
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			// Non-numeric string
			return 0
		}
		recordFloat = f
	default:
		// Unsupported type
		return 0
	}

	if recordFloat < filterFloat {
		return -1
	} else if recordFloat > filterFloat {
		return 1
	}
	return 0
}
