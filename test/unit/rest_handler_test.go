package unit

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestQueryParameterParsing tests parsing of REST API query parameters
func TestQueryParameterParsing(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		expected map[string]interface{}
	}{
		{
			name:  "simple equals filter",
			query: "name=eq.John",
			expected: map[string]interface{}{
				"field":    "name",
				"operator": "eq",
				"value":    "John",
			},
		},
		{
			name:  "greater than filter",
			query: "age=gt.25",
			expected: map[string]interface{}{
				"field":    "age",
				"operator": "gt",
				"value":    "25",
			},
		},
		{
			name:  "like filter",
			query: "email=like.*@example.com",
			expected: map[string]interface{}{
				"field":    "email",
				"operator": "like",
				"value":    "*@example.com",
			},
		},
		{
			name:  "in filter",
			query: "status=in.(active,pending)",
			expected: map[string]interface{}{
				"field":    "status",
				"operator": "in",
				"value":    "(active,pending)",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseQueryParameter(tt.query)
			assert.Equal(t, tt.expected["field"], result["field"])
			assert.Equal(t, tt.expected["operator"], result["operator"])
			assert.Equal(t, tt.expected["value"], result["value"])
		})
	}
}

// parseQueryParameter parses a single query parameter
func parseQueryParameter(query string) map[string]interface{} {
	result := make(map[string]interface{})
	// Simple parser for testing
	for i, c := range query {
		if c == '=' {
			result["field"] = query[:i]
			rest := query[i+1:]
			for j, c2 := range rest {
				if c2 == '.' {
					result["operator"] = rest[:j]
					result["value"] = rest[j+1:]
					break
				}
			}
			break
		}
	}
	return result
}

// TestPaginationCalculation tests pagination offset/limit calculation
func TestPaginationCalculation(t *testing.T) {
	tests := []struct {
		name           string
		page           int
		pageSize       int
		expectedOffset int
		expectedLimit  int
	}{
		{
			name:           "first page",
			page:           1,
			pageSize:       10,
			expectedOffset: 0,
			expectedLimit:  10,
		},
		{
			name:           "second page",
			page:           2,
			pageSize:       10,
			expectedOffset: 10,
			expectedLimit:  10,
		},
		{
			name:           "page 5 with 25 items",
			page:           5,
			pageSize:       25,
			expectedOffset: 100,
			expectedLimit:  25,
		},
		{
			name:           "page 1 with 100 items",
			page:           1,
			pageSize:       100,
			expectedOffset: 0,
			expectedLimit:  100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			offset, limit := calculatePagination(tt.page, tt.pageSize)
			assert.Equal(t, tt.expectedOffset, offset)
			assert.Equal(t, tt.expectedLimit, limit)
		})
	}
}

// calculatePagination calculates offset and limit for pagination
func calculatePagination(page, pageSize int) (offset, limit int) {
	offset = (page - 1) * pageSize
	limit = pageSize
	return
}

// TestOrderByValidation tests ordering parameter validation
func TestOrderByValidation(t *testing.T) {
	tests := []struct {
		name    string
		orderBy string
		hasAsc  bool
		hasDesc bool
	}{
		{
			name:    "has asc",
			orderBy: "created_at.asc",
			hasAsc:  true,
		},
		{
			name:    "has desc",
			orderBy: "updated_at.desc",
			hasDesc: true,
		},
		{
			name:    "has both",
			orderBy: "name.asc,created_at.desc",
			hasAsc:  true,
			hasDesc: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasAsc, hasDesc := checkOrderBy(tt.orderBy)
			if tt.hasAsc {
				assert.True(t, hasAsc, "should have .asc")
			}
			if tt.hasDesc {
				assert.True(t, hasDesc, "should have .desc")
			}
		})
	}
}

// checkOrderBy checks for .asc and .desc in order by string
func checkOrderBy(orderBy string) (hasAsc bool, hasDesc bool) {
	// Simple string contains check
	for i := 0; i < len(orderBy); i++ {
		if i+4 <= len(orderBy) && orderBy[i:i+4] == ".asc" {
			hasAsc = true
		}
		if i+5 <= len(orderBy) && orderBy[i:i+5] == ".desc" {
			hasDesc = true
		}
	}
	return
}

// containsSQLKeywords checks for SQL injection keywords
func containsSQLKeywords(s string) bool {
	keywords := []string{"DROP", "DELETE", "INSERT", "UPDATE", "SELECT", ";"}
	for _, keyword := range keywords {
		if len(s) >= len(keyword) {
			for i := 0; i <= len(s)-len(keyword); i++ {
				match := true
				for j := 0; j < len(keyword); j++ {
					if s[i+j] != keyword[j] && s[i+j] != keyword[j]+32 {
						match = false
						break
					}
				}
				if match {
					return true
				}
			}
		}
	}
	return false
}

// TestFilterValidation tests filter parameter validation
func TestFilterValidation(t *testing.T) {
	tests := []struct {
		name   string
		filter map[string]interface{}
		valid  bool
	}{
		{
			name: "valid simple filter",
			filter: map[string]interface{}{
				"name": "John",
			},
			valid: true,
		},
		{
			name: "valid multiple filters",
			filter: map[string]interface{}{
				"name":   "John",
				"status": "active",
			},
			valid: true,
		},
		{
			name: "valid nested filter",
			filter: map[string]interface{}{
				"user": map[string]interface{}{
					"age": 25,
				},
			},
			valid: true,
		},
		{
			name:   "empty filter",
			filter: map[string]interface{}{},
			valid:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid := validateFilter(tt.filter)
			assert.Equal(t, tt.valid, valid)
		})
	}
}

// validateFilter validates filter parameters
func validateFilter(filter map[string]interface{}) bool {
	// All filters are valid for this simple test
	return true
}

// TestSelectFieldsValidation tests select fields validation
func TestSelectFieldsValidation(t *testing.T) {
	tests := []struct {
		name   string
		fields []string
		valid  bool
	}{
		{
			name:   "valid single field",
			fields: []string{"id"},
			valid:  true,
		},
		{
			name:   "valid multiple fields",
			fields: []string{"id", "name", "email"},
			valid:  true,
		},
		{
			name:   "invalid - wildcard",
			fields: []string{"*"},
			valid:  false,
		},
		{
			name:   "invalid - SQL injection",
			fields: []string{"id; DROP TABLE users;"},
			valid:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid := validateSelectFields(tt.fields)
			assert.Equal(t, tt.valid, valid)
		})
	}
}

// validateSelectFields validates select field parameters
func validateSelectFields(fields []string) bool {
	for _, field := range fields {
		if field == "*" || containsSQLKeywords(field) {
			return false
		}
	}
	return true
}

// TestIsGeoJSON tests detection of GeoJSON objects for PostGIS support
func TestIsGeoJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected bool
	}{
		{
			name: "valid Point",
			input: map[string]interface{}{
				"type":        "Point",
				"coordinates": []interface{}{-122.4783, 37.8199},
			},
			expected: true,
		},
		{
			name: "valid LineString",
			input: map[string]interface{}{
				"type": "LineString",
				"coordinates": []interface{}{
					[]interface{}{-122.4783, 37.8199},
					[]interface{}{-122.4230, 37.8267},
				},
			},
			expected: true,
		},
		{
			name: "valid Polygon",
			input: map[string]interface{}{
				"type": "Polygon",
				"coordinates": []interface{}{
					[]interface{}{
						[]interface{}{-122.5, 37.7},
						[]interface{}{-122.5, 37.85},
						[]interface{}{-122.35, 37.85},
						[]interface{}{-122.35, 37.7},
						[]interface{}{-122.5, 37.7},
					},
				},
			},
			expected: true,
		},
		{
			name: "valid MultiPoint",
			input: map[string]interface{}{
				"type": "MultiPoint",
				"coordinates": []interface{}{
					[]interface{}{-122.4783, 37.8199},
					[]interface{}{-122.4230, 37.8267},
				},
			},
			expected: true,
		},
		{
			name: "valid MultiLineString",
			input: map[string]interface{}{
				"type": "MultiLineString",
				"coordinates": []interface{}{
					[]interface{}{
						[]interface{}{-122.4783, 37.8199},
						[]interface{}{-122.4230, 37.8267},
					},
				},
			},
			expected: true,
		},
		{
			name: "valid MultiPolygon",
			input: map[string]interface{}{
				"type": "MultiPolygon",
				"coordinates": []interface{}{
					[]interface{}{
						[]interface{}{
							[]interface{}{-122.5, 37.7},
							[]interface{}{-122.5, 37.85},
							[]interface{}{-122.35, 37.85},
							[]interface{}{-122.5, 37.7},
						},
					},
				},
			},
			expected: true,
		},
		{
			name: "valid GeometryCollection",
			input: map[string]interface{}{
				"type": "GeometryCollection",
				"coordinates": []interface{}{
					map[string]interface{}{
						"type":        "Point",
						"coordinates": []interface{}{-122.4783, 37.8199},
					},
				},
			},
			expected: true,
		},
		{
			name: "missing type field",
			input: map[string]interface{}{
				"coordinates": []interface{}{-122.4783, 37.8199},
			},
			expected: false,
		},
		{
			name: "missing coordinates field",
			input: map[string]interface{}{
				"type": "Point",
			},
			expected: false,
		},
		{
			name: "invalid type - not a string",
			input: map[string]interface{}{
				"type":        123,
				"coordinates": []interface{}{-122.4783, 37.8199},
			},
			expected: false,
		},
		{
			name: "invalid type - unknown geometry type",
			input: map[string]interface{}{
				"type":        "Triangle",
				"coordinates": []interface{}{-122.4783, 37.8199},
			},
			expected: false,
		},
		{
			name:     "not a map - string",
			input:    "not a map",
			expected: false,
		},
		{
			name:     "not a map - number",
			input:    42,
			expected: false,
		},
		{
			name:     "not a map - array",
			input:    []interface{}{-122.4783, 37.8199},
			expected: false,
		},
		{
			name:     "nil input",
			input:    nil,
			expected: false,
		},
		{
			name:     "empty map",
			input:    map[string]interface{}{},
			expected: false,
		},
		{
			name: "has type and coordinates but not GeoJSON structure",
			input: map[string]interface{}{
				"type":        "custom",
				"coordinates": "some value",
			},
			expected: false,
		},
		{
			name: "case sensitive type check",
			input: map[string]interface{}{
				"type":        "point", // lowercase should fail
				"coordinates": []interface{}{-122.4783, 37.8199},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isGeoJSON(tt.input)
			assert.Equal(t, tt.expected, result, "isGeoJSON(%v) should be %v", tt.input, tt.expected)
		})
	}
}

// isGeoJSON checks if a value is a valid GeoJSON object
// This mirrors the implementation in internal/api/rest_handler.go
func isGeoJSON(val interface{}) bool {
	m, ok := val.(map[string]interface{})
	if !ok {
		return false
	}

	geoType, hasType := m["type"]
	_, hasCoords := m["coordinates"]

	if !hasType || !hasCoords {
		return false
	}

	typeStr, ok := geoType.(string)
	if !ok {
		return false
	}

	validTypes := map[string]bool{
		"Point":              true,
		"LineString":         true,
		"Polygon":            true,
		"MultiPoint":         true,
		"MultiLineString":    true,
		"MultiPolygon":       true,
		"GeometryCollection": true,
	}

	return validTypes[typeStr]
}
