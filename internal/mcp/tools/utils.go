package tools

import (
	"encoding/json"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/twpayne/go-geom/encoding/geojson"
	"github.com/twpayne/go-geom/encoding/wkb"
)

// pgxRowsToJSON converts pgx rows to JSON-serializable format
// This is duplicated from internal/api/rest_utils.go to avoid circular imports
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
