package api

import (
	"fmt"
	"strings"
)

// parseFilter parses filter parameters
func (qp *QueryParser) parseFilter(key, value string, params *QueryParams) error {
	// Handle logical operators
	if key == "or" {
		return qp.parseLogicalFilter(value, params, true)
	}
	if key == "and" {
		return qp.parseLogicalFilter(value, params, false)
	}

	// Check for classic format first: column.operator=value
	// This takes precedence over PostgREST format
	if strings.Contains(key, ".") {
		parts := strings.SplitN(key, ".", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid filter format: %s", key)
		}

		column := parts[0]
		operator := FilterOperator(parts[1])

		// Parse value based on operator
		var filterValue interface{}
		switch operator {
		case OpIn:
			// Parse array values: (1,2,3) or ["a","b","c"]
			filterValue = qp.parseArrayValue(value)
		case OpIs:
			// Parse null/true/false - H-14: Validate boolean values
			switch value {
			case "null":
				filterValue = nil
			case "true":
				filterValue = true
			case "false":
				filterValue = false
			default:
				return fmt.Errorf("invalid value for OpIs operator: %s (must be null, true, or false)", value)
			}
		default:
			filterValue = value
		}

		params.Filters = append(params.Filters, Filter{
			Column:   column,
			Operator: operator,
			Value:    filterValue,
			IsOr:     false,
		})

		return nil
	}

	// Try PostgREST format: column=operator.value
	// Split value by first dot to extract operator
	dotIndex := strings.Index(value, ".")
	if dotIndex > 0 {
		// PostgREST format: column=operator.value
		column := key
		operatorStr := value[:dotIndex]
		filterValue := value[dotIndex+1:]

		operator := FilterOperator(operatorStr)

		// Parse value based on operator
		var parsedValue interface{}
		switch operator {
		case OpIn:
			// Parse array values: (1,2,3) or ["a","b","c"]
			parsedValue = qp.parseArrayValue(filterValue)
		case OpIs:
			// Parse null/true/false - H-14: Validate boolean values
			switch filterValue {
			case "null":
				parsedValue = nil
			case "true":
				parsedValue = true
			case "false":
				parsedValue = false
			default:
				return fmt.Errorf("invalid value for OpIs operator: %s (must be null, true, or false)", filterValue)
			}
		default:
			parsedValue = filterValue
		}

		params.Filters = append(params.Filters, Filter{
			Column:   column,
			Operator: operator,
			Value:    parsedValue,
			IsOr:     false,
		})

		return nil
	}

	// If neither format matched, return an error
	return fmt.Errorf("invalid filter format: %s", key)
}

// parseLogicalFilter parses or/and grouped filters with support for nested expressions
// Supports formats like:
//   - or=(name.eq.John,age.gt.30)
//   - and=(or(col.lt.min1,col.gt.max1),or(col.lt.min2,col.gt.max2))
func (qp *QueryParser) parseLogicalFilter(value string, params *QueryParams, isOr bool) error {
	// Parse format: or=(name.eq.John,age.gt.30)
	// Only remove one pair of outer parentheses (not all leading/trailing parens)
	if strings.HasPrefix(value, "(") && strings.HasSuffix(value, ")") {
		value = value[1 : len(value)-1]
	}

	// Use parentheses-aware splitting to handle nested expressions
	filters, err := qp.parseNestedFilters(value)
	if err != nil {
		return err
	}

	for _, filter := range filters {
		filter = strings.TrimSpace(filter)
		if filter == "" {
			continue
		}

		// Check for nested or() expression
		if strings.HasPrefix(filter, "or(") && strings.HasSuffix(filter, ")") {
			// Nested OR expression - parse recursively with new group ID
			innerValue := strings.TrimPrefix(filter, "or(")
			innerValue = strings.TrimSuffix(innerValue, ")")
			if err := qp.parseNestedOrGroup(innerValue, params); err != nil {
				return err
			}
			continue
		}

		// Check for nested and() expression
		if strings.HasPrefix(filter, "and(") && strings.HasSuffix(filter, ")") {
			// Nested AND expression - parse recursively
			innerValue := strings.TrimPrefix(filter, "and(")
			innerValue = strings.TrimSuffix(innerValue, ")")
			if err := qp.parseLogicalFilter(innerValue, params, false); err != nil {
				return err
			}
			continue
		}

		// Regular filter: column.operator.value
		parts := strings.SplitN(filter, ".", 3)
		if len(parts) != 3 {
			return fmt.Errorf("invalid filter format in logical group: %s", filter)
		}

		column := parts[0]
		operator := FilterOperator(parts[1])
		rawValue := parts[2]

		// Parse value based on operator (same logic as regular filter parsing)
		var parsedValue interface{}
		switch operator {
		case OpIn:
			// Parse array values: (1,2,3) or ["a","b","c"]
			parsedValue = qp.parseArrayValue(rawValue)
		case OpIs:
			// Parse null/true/false - H-14: Validate boolean values
			switch rawValue {
			case "null":
				parsedValue = nil
			case "true":
				parsedValue = true
			case "false":
				parsedValue = false
			default:
				return fmt.Errorf("invalid value for OpIs operator: %s (must be null, true, or false)", rawValue)
			}
		default:
			parsedValue = rawValue
		}

		params.Filters = append(params.Filters, Filter{
			Column:   column,
			Operator: operator,
			Value:    parsedValue,
			IsOr:     isOr,
		})
	}

	return nil
}

// parseNestedOrGroup parses an OR group and assigns a unique group ID to all filters
func (qp *QueryParser) parseNestedOrGroup(value string, params *QueryParams) error {
	// Increment group counter for this OR group
	params.orGroupCounter++
	groupID := params.orGroupCounter

	// Split by comma (respecting parentheses)
	filters, err := qp.parseNestedFilters(value)
	if err != nil {
		return err
	}

	for _, filter := range filters {
		filter = strings.TrimSpace(filter)
		if filter == "" {
			continue
		}

		// Parse each filter: column.operator.value
		parts := strings.SplitN(filter, ".", 3)
		if len(parts) != 3 {
			return fmt.Errorf("invalid filter format in OR group: %s", filter)
		}

		column := parts[0]
		operator := FilterOperator(parts[1])
		rawValue := parts[2]

		// Parse value based on operator (same logic as regular filter parsing)
		var parsedValue interface{}
		switch operator {
		case OpIn:
			// Parse array values: (1,2,3) or ["a","b","c"]
			parsedValue = qp.parseArrayValue(rawValue)
		case OpIs:
			// Parse null/true/false - H-14: Validate boolean values
			switch rawValue {
			case "null":
				parsedValue = nil
			case "true":
				parsedValue = true
			case "false":
				parsedValue = false
			default:
				return fmt.Errorf("invalid value for OpIs operator: %s (must be null, true, or false)", rawValue)
			}
		default:
			parsedValue = rawValue
		}

		params.Filters = append(params.Filters, Filter{
			Column:    column,
			Operator:  operator,
			Value:     parsedValue,
			IsOr:      true,
			OrGroupID: groupID,
		})
	}

	return nil
}

// parseNestedFilters splits a filter string by commas while respecting parentheses nesting
func (qp *QueryParser) parseNestedFilters(value string) ([]string, error) {
	var filters []string
	var current strings.Builder
	depth := 0

	for _, ch := range value {
		switch ch {
		case '(':
			depth++
			current.WriteRune(ch)
		case ')':
			depth--
			current.WriteRune(ch)
			if depth < 0 {
				return nil, fmt.Errorf("unbalanced parentheses in filter expression")
			}
		case ',':
			if depth == 0 {
				if s := strings.TrimSpace(current.String()); s != "" {
					filters = append(filters, s)
				}
				current.Reset()
			} else {
				current.WriteRune(ch)
			}
		default:
			current.WriteRune(ch)
		}
	}

	if depth != 0 {
		return nil, fmt.Errorf("unbalanced parentheses in filter expression")
	}

	if s := strings.TrimSpace(current.String()); s != "" {
		filters = append(filters, s)
	}

	return filters, nil
}

// parseArrayValue parses array values from string
func (qp *QueryParser) parseArrayValue(value string) []string {
	// Remove parentheses or brackets
	value = strings.Trim(value, "()[]")

	// Split by comma
	items := strings.Split(value, ",")
	result := make([]string, len(items))

	for i, item := range items {
		// Remove quotes if present
		result[i] = strings.Trim(strings.TrimSpace(item), "\"'")
	}

	return result
}
