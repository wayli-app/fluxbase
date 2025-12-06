package api

import (
	"fmt"
	"strings"

	"github.com/fluxbase-eu/fluxbase/internal/database"
)

// isGeoJSON checks if a value looks like a GeoJSON object
func isGeoJSON(val interface{}) bool {
	m, ok := val.(map[string]interface{})
	if !ok {
		return false
	}
	// Check for GeoJSON structure: must have "type" and "coordinates"
	geoType, hasType := m["type"]
	_, hasCoords := m["coordinates"]
	if !hasType || !hasCoords {
		return false
	}
	// Verify it's a valid GeoJSON geometry type
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

// isPartialGeoJSON checks if a value looks like GeoJSON but is incomplete (has type but missing coordinates)
func isPartialGeoJSON(val interface{}) bool {
	m, ok := val.(map[string]interface{})
	if !ok {
		return false
	}
	// Has "type" field but no "coordinates" - likely invalid GeoJSON
	_, hasType := m["type"]
	_, hasCoords := m["coordinates"]
	return hasType && !hasCoords
}

// isGeometryColumn checks if a column type is a PostGIS geometry/geography type
func isGeometryColumn(dataType string) bool {
	dt := strings.ToLower(dataType)
	return strings.Contains(dt, "geometry") || strings.Contains(dt, "geography")
}

// buildSelectColumns builds a column list that converts geometry columns to GeoJSON
func buildSelectColumns(table database.TableInfo) string {
	columns := make([]string, 0, len(table.Columns))
	for _, col := range table.Columns {
		if isGeometryColumn(col.DataType) {
			// Convert geometry to GeoJSON
			columns = append(columns, fmt.Sprintf("ST_AsGeoJSON(%s)::jsonb AS %s", col.Name, col.Name))
		} else {
			columns = append(columns, col.Name)
		}
	}
	return strings.Join(columns, ", ")
}

// buildReturningClause builds a RETURNING clause that handles geometry columns
// by converting them to GeoJSON using ST_AsGeoJSON
func buildReturningClause(table database.TableInfo) string {
	return " RETURNING " + buildSelectColumns(table)
}
