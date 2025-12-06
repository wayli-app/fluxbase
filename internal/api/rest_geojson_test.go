package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
			name: "has type and coordinates but not valid GeoJSON type",
			input: map[string]interface{}{
				"type":        "custom",
				"coordinates": "some value",
			},
			expected: false,
		},
		{
			name: "case sensitive type check - lowercase should fail",
			input: map[string]interface{}{
				"type":        "point",
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

// TestIsPartialGeoJSON tests detection of incomplete GeoJSON objects
func TestIsPartialGeoJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected bool
	}{
		{
			name: "partial - has type but no coordinates",
			input: map[string]interface{}{
				"type": "Point",
			},
			expected: true,
		},
		{
			name: "complete GeoJSON - has both",
			input: map[string]interface{}{
				"type":        "Point",
				"coordinates": []interface{}{-122.4783, 37.8199},
			},
			expected: false,
		},
		{
			name: "not partial - has coordinates but no type",
			input: map[string]interface{}{
				"coordinates": []interface{}{-122.4783, 37.8199},
			},
			expected: false,
		},
		{
			name:     "not a map",
			input:    "string",
			expected: false,
		},
		{
			name:     "nil input",
			input:    nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isPartialGeoJSON(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestIsGeometryColumn tests detection of PostGIS geometry column types
func TestIsGeometryColumn(t *testing.T) {
	tests := []struct {
		name     string
		dataType string
		expected bool
	}{
		{"geometry type", "geometry", true},
		{"geography type", "geography", true},
		{"geometry with SRID", "geometry(Point,4326)", true},
		{"geography with SRID", "geography(Polygon,4326)", true},
		{"USER-DEFINED (PostGIS)", "USER-DEFINED", true},
		{"text type", "text", false},
		{"integer type", "integer", false},
		{"jsonb type", "jsonb", false},
		{"empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isGeometryColumn(tt.dataType)
			assert.Equal(t, tt.expected, result)
		})
	}
}
