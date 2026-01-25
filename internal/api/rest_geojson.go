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

// isTextColumn checks if a column type is a text/string type that can be truncated
func isTextColumn(dataType string) bool {
	dt := strings.ToLower(dataType)
	return dt == "text" ||
		dt == "varchar" ||
		dt == "character varying" ||
		strings.HasPrefix(dt, "varchar(") ||
		strings.HasPrefix(dt, "character varying(") ||
		dt == "char" ||
		dt == "character" ||
		strings.HasPrefix(dt, "char(") ||
		strings.HasPrefix(dt, "character(")
}

// buildSelectColumns builds a column list that converts geometry columns to GeoJSON
func buildSelectColumns(table database.TableInfo) string {
	return buildSelectColumnsWithTruncation(table, nil)
}

// buildSelectColumnsWithTruncation builds a column list with optional text truncation
// If truncateLength is non-nil and > 0, text columns will be truncated to that length
func buildSelectColumnsWithTruncation(table database.TableInfo, truncateLength *int) string {
	columns := make([]string, 0, len(table.Columns))
	for _, col := range table.Columns {
		quotedName := quoteIdentifier(col.Name)
		if quotedName == "" {
			continue // Skip invalid column names
		}
		if isGeometryColumn(col.DataType) {
			// Convert geometry to GeoJSON
			columns = append(columns, fmt.Sprintf("ST_AsGeoJSON(%s)::jsonb AS %s", quotedName, quotedName))
		} else if truncateLength != nil && *truncateLength > 0 && isTextColumn(col.DataType) {
			// Truncate text columns: show first N chars + length indicator if truncated
			columns = append(columns, fmt.Sprintf(
				"CASE WHEN %s IS NULL THEN NULL WHEN LENGTH(%s) > %d THEN LEFT(%s, %d) || '... (' || LENGTH(%s) || ' chars)' ELSE %s END AS %s",
				quotedName, quotedName, *truncateLength, quotedName, *truncateLength, quotedName, quotedName, quotedName))
		} else {
			columns = append(columns, quotedName)
		}
	}
	return strings.Join(columns, ", ")
}

// buildReturningClause builds a RETURNING clause that handles geometry columns
// by converting them to GeoJSON using ST_AsGeoJSON
func buildReturningClause(table database.TableInfo) string {
	return " RETURNING " + buildSelectColumns(table)
}
