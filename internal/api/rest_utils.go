package api

import (
	"encoding/json"
	"strings"

	"github.com/fluxbase-eu/fluxbase/internal/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/twpayne/go-geom/encoding/geojson"
	"github.com/twpayne/go-geom/encoding/wkb"
)

// pgxRowsToJSON converts pgx rows to JSON-serializable format
func pgxRowsToJSON(rows pgx.Rows) ([]map[string]interface{}, error) {
	// Get column descriptions
	fields := rows.FieldDescriptions()

	results := []map[string]interface{}{}

	for rows.Next() {
		// Create a slice to hold the values
		values := make([]interface{}, len(fields))
		valuePtrs := make([]interface{}, len(fields))

		for i := range values {
			valuePtrs[i] = &values[i]
		}

		// Scan the row
		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}

		// Build the result map
		row := make(map[string]interface{})
		for i, field := range fields {
			columnName := string(field.Name)

			// Handle special types
			switch v := values[i].(type) {
			case []byte:
				// First, try to decode as PostGIS geometry (WKB format)
				geom, err := wkb.Unmarshal(v)
				if err == nil {
					// Successfully decoded as WKB, convert to GeoJSON
					geoJSON, err := geojson.Marshal(geom)
					if err == nil {
						var geoJSONData interface{}
						if err := json.Unmarshal(geoJSON, &geoJSONData); err == nil {
							row[columnName] = geoJSONData
							continue
						}
					}
				}

				// Not WKB geometry, try to parse as JSON
				var jsonData interface{}
				if err := json.Unmarshal(v, &jsonData); err == nil {
					row[columnName] = jsonData
				} else {
					// If not JSON, convert to string
					row[columnName] = string(v)
				}
			case [16]byte:
				// Convert UUID bytes to string
				uid, err := uuid.FromBytes(v[:])
				if err == nil {
					row[columnName] = uid.String()
				} else {
					row[columnName] = v
				}
			default:
				row[columnName] = v
			}
		}

		results = append(results, row)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
}

// getConflictTarget determines the conflict target for ON CONFLICT clause
// Returns the primary key columns as a comma-separated quoted string, or empty string if no PK exists
func (h *RESTHandler) getConflictTarget(table database.TableInfo) string {
	if len(table.PrimaryKey) == 0 {
		return ""
	}
	// Quote each column name to prevent SQL injection
	quotedColumns := make([]string, 0, len(table.PrimaryKey))
	for _, col := range table.PrimaryKey {
		quotedColumns = append(quotedColumns, quoteIdentifier(col))
	}
	return strings.Join(quotedColumns, ", ")
}

// getConflictTargetUnquoted returns unquoted column names for comparison purposes
func (h *RESTHandler) getConflictTargetUnquoted(table database.TableInfo) []string {
	return table.PrimaryKey
}

// isInConflictTarget checks if a column is part of the conflict target columns
func (h *RESTHandler) isInConflictTarget(column string, conflictTargetColumns []string) bool {
	for _, target := range conflictTargetColumns {
		if target == column {
			return true
		}
	}
	return false
}
