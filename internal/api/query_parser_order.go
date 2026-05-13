package api

import (
	"fmt"
	"strings"
)

// parseOrder parses the order parameter
func (qp *QueryParser) parseOrder(value string, params *QueryParams) error {
	// Parse format: order=name.asc,created_at.desc.nullslast
	// Vector ordering format: order=embedding.vec_cos.[0.1,0.2,...].asc
	orders := splitOrderParams(value)

	for _, order := range orders {
		order = strings.TrimSpace(order)
		if order == "" {
			continue
		}

		// Check for vector ordering format: column.vec_op.[vector].direction
		// The vector is enclosed in brackets, so we need special parsing
		if vectorOrder, ok := qp.parseVectorOrder(order); ok {
			params.Order = append(params.Order, vectorOrder)
			continue
		}

		// Standard ordering: column.direction.nulls
		parts := strings.Split(order, ".")
		if len(parts) < 2 {
			return fmt.Errorf("invalid order format: %s", order)
		}

		// Validate column name to prevent SQL injection
		colName := parts[0]
		if !isValidIdentifier(colName) {
			return fmt.Errorf("invalid order column name: %s", colName)
		}

		orderBy := OrderBy{
			Column: colName,
			Desc:   parts[1] == "desc",
		}

		// Check for nulls first/last
		if len(parts) > 2 {
			switch parts[2] {
			case "nullsfirst":
				orderBy.Nulls = "first"
			case "nullslast":
				orderBy.Nulls = "last"
			}
		}

		params.Order = append(params.Order, orderBy)
	}

	return nil
}

// splitOrderParams splits order parameters by comma, respecting brackets
func splitOrderParams(value string) []string {
	var orders []string
	var current strings.Builder
	bracketDepth := 0

	for _, ch := range value {
		switch ch {
		case '[':
			bracketDepth++
			current.WriteRune(ch)
		case ']':
			bracketDepth--
			current.WriteRune(ch)
		case ',':
			if bracketDepth == 0 {
				if s := strings.TrimSpace(current.String()); s != "" {
					orders = append(orders, s)
				}
				current.Reset()
			} else {
				current.WriteRune(ch)
			}
		default:
			current.WriteRune(ch)
		}
	}

	if s := strings.TrimSpace(current.String()); s != "" {
		orders = append(orders, s)
	}

	return orders
}

// parseVectorOrder parses vector ordering format: column.vec_op.[vector].direction
// Example: embedding.vec_cos.[0.1,0.2,0.3].asc
func (qp *QueryParser) parseVectorOrder(order string) (OrderBy, bool) {
	// Look for vector operator pattern
	vectorOps := []string{".vec_l2.", ".vec_cos.", ".vec_ip."}
	opIdx := -1
	var opStr string

	for _, op := range vectorOps {
		if idx := strings.Index(order, op); idx > 0 {
			opIdx = idx
			opStr = strings.Trim(op, ".")
			break
		}
	}

	if opIdx < 0 {
		return OrderBy{}, false
	}

	// Extract column name
	colName := order[:opIdx]
	if !isValidIdentifier(colName) {
		return OrderBy{}, false
	}

	// Extract the rest after the operator
	remainder := order[opIdx+len(opStr)+2:] // +2 for the dots

	// Find the vector value in brackets
	bracketStart := strings.Index(remainder, "[")
	bracketEnd := strings.LastIndex(remainder, "]")

	if bracketStart < 0 || bracketEnd < bracketStart {
		return OrderBy{}, false
	}

	vectorStr := remainder[bracketStart : bracketEnd+1]

	// Get direction if present (after the closing bracket)
	var desc bool
	afterVector := remainder[bracketEnd+1:]
	if strings.Contains(afterVector, ".desc") {
		desc = true
	}
	// Default is ASC (ascending) for distance-based ordering (lower = more similar)

	return OrderBy{
		Column:      colName,
		Desc:        desc,
		VectorOp:    FilterOperator(opStr),
		VectorValue: vectorStr,
	}, true
}
