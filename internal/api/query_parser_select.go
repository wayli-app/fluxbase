package api

import "strings"

// parseSelect parses the select parameter
func (qp *QueryParser) parseSelect(value string, params *QueryParams) error {
	// Parse format: select=id,name,posts(id,title,author(name))
	// Or with aggregations: select=category,count(*),sum(price),avg(rating)
	fields, embedded := qp.parseSelectFields(value)

	// Separate regular fields from aggregations
	regularFields := []string{}
	for _, field := range fields {
		if agg := qp.parseAggregation(field); agg != nil {
			params.Aggregations = append(params.Aggregations, *agg)
		} else {
			regularFields = append(regularFields, field)
		}
	}

	params.Select = regularFields

	for name, subSelect := range embedded {
		params.Embedded = append(params.Embedded, EmbeddedRelation{
			Name:   name,
			Select: subSelect,
		})
	}

	return nil
}

// parseSelectFields parses select fields and embedded relations
func (qp *QueryParser) parseSelectFields(value string) ([]string, map[string][]string) {
	fields := []string{}
	embedded := make(map[string][]string)

	// Known aggregation function names
	aggFuncs := map[string]bool{
		"count": true,
		"sum":   true,
		"avg":   true,
		"min":   true,
		"max":   true,
	}

	// Simple parser for nested parentheses
	var current strings.Builder
	var relationName string
	var depth int
	var inRelation bool
	var isAggregation bool

	for i := 0; i < len(value); i++ {
		ch := value[i]

		switch ch {
		case '(':
			if depth == 0 {
				relationName = strings.TrimSpace(current.String())
				// Check if this is an aggregation function
				isAggregation = aggFuncs[strings.ToLower(relationName)]
				if !isAggregation {
					// It's a relation, not an aggregation
					current.Reset()
					inRelation = true
				} else {
					// It's an aggregation function, keep building the field string
					current.WriteByte(ch)
				}
			} else {
				current.WriteByte(ch)
			}
			depth++

		case ')':
			depth--
			switch {
			case depth == 0 && inRelation && !isAggregation:
				// End of relation fields
				subFields := strings.Split(current.String(), ",")
				for j := range subFields {
					subFields[j] = strings.TrimSpace(subFields[j])
				}
				embedded[relationName] = subFields
				current.Reset()
				inRelation = false
			case depth == 0 && isAggregation:
				// End of aggregation function
				current.WriteByte(ch)
				isAggregation = false
			case depth > 0:
				current.WriteByte(ch)
			}

		case ',':
			if depth == 0 {
				if field := strings.TrimSpace(current.String()); field != "" {
					fields = append(fields, field)
				}
				current.Reset()
			} else {
				current.WriteByte(ch)
			}

		default:
			current.WriteByte(ch)
		}
	}

	// Add the last field
	if field := strings.TrimSpace(current.String()); field != "" {
		fields = append(fields, field)
	}

	return fields, embedded
}

// parseAggregation parses aggregation functions from a select field
// Examples: count(*), sum(price), avg(rating), count(id), min(created_at), max(updated_at)
func (qp *QueryParser) parseAggregation(field string) *Aggregation {
	field = strings.TrimSpace(field)

	// Check for aggregation function pattern: function(column) or function(*)
	funcEnd := strings.Index(field, "(")
	if funcEnd == -1 {
		return nil // Not an aggregation
	}

	funcName := strings.ToLower(strings.TrimSpace(field[:funcEnd]))
	remainder := field[funcEnd+1:]

	// Find closing parenthesis
	parenEnd := strings.Index(remainder, ")")
	if parenEnd == -1 {
		return nil // Malformed
	}

	column := strings.TrimSpace(remainder[:parenEnd])

	// Map function name to AggregateFunction
	var aggFunc AggregateFunction
	switch funcName {
	case "count":
		if column == "*" {
			aggFunc = AggCountAll
			column = "" // count(*) doesn't need a column
		} else {
			aggFunc = AggCount
		}
	case "sum":
		aggFunc = AggSum
	case "avg":
		aggFunc = AggAvg
	case "min":
		aggFunc = AggMin
	case "max":
		aggFunc = AggMax
	default:
		return nil // Unknown aggregation function
	}

	return &Aggregation{
		Function: aggFunc,
		Column:   column,
		Alias:    "", // Will be generated if needed
	}
}
