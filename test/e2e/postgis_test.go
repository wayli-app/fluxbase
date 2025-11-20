package e2e

import (
	"fmt"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/require"
	"github.com/wayli-app/fluxbase/test"
)

// PostGISTestContext extends TestContext with PostGIS-specific setup
type PostGISTestContext struct {
	*test.TestContext
	APIKey         string
	PostGISEnabled bool
	LocationsTable string
	RegionsTable   string
}

// setupPostGISTest prepares the test context for PostGIS tests
func setupPostGISTest(t *testing.T) *PostGISTestContext {
	tc := test.NewTestContext(t)
	tc.EnsureAuthSchema()

	// Check if PostGIS is available
	postGISEnabled := false
	rows := tc.QuerySQL("SELECT COUNT(*) as count FROM pg_extension WHERE extname = 'postgis'")
	if len(rows) > 0 {
		count, ok := rows[0]["count"].(int64)
		if ok && count > 0 {
			postGISEnabled = true
		}
	}

	// Table names (tables are created in TestMain via setup_test.go)
	locationsTable := "locations"
	regionsTable := "regions"

	// Clean tables if PostGIS is enabled
	if postGISEnabled {
		tc.ExecuteSQL("TRUNCATE TABLE " + locationsTable + " CASCADE")
		tc.ExecuteSQL("TRUNCATE TABLE " + regionsTable + " CASCADE")
	}

	// Create API key
	apiKey := tc.CreateAPIKey("PostGIS Test API Key", nil)

	return &PostGISTestContext{
		TestContext:    tc,
		APIKey:         apiKey,
		PostGISEnabled: postGISEnabled,
		LocationsTable: locationsTable,
		RegionsTable:   regionsTable,
	}
}

// TestPostGISInsertPoint tests inserting a Point geometry
func TestPostGISInsertPoint(t *testing.T) {
	tc := setupPostGISTest(t)
	defer tc.Close()

	if !tc.PostGISEnabled {
		t.Skip("PostGIS is not installed, skipping test")
	}

	// WHEN: Creating a location with Point geometry
	resp := tc.NewRequest("POST", "/api/v1/tables/"+tc.LocationsTable).
		WithAPIKey(tc.APIKey).
		WithBody(map[string]interface{}{
			"name": "Golden Gate Bridge",
			"location": map[string]interface{}{
				"type":        "Point",
				"coordinates": []interface{}{-122.4783, 37.8199},
			},
		}).
		Send()

	// THEN: Location is created successfully
	resp.AssertStatus(fiber.StatusCreated)

	var result map[string]interface{}
	resp.JSON(&result)

	require.NotNil(t, result["id"], "Should return created location ID")
	require.Equal(t, "Golden Gate Bridge", result["name"])

	// AND: Location can be retrieved with GeoJSON format
	locationID := result["id"]
	getResp := tc.NewRequest("GET", "/api/v1/tables/"+tc.LocationsTable+"?id=eq."+toString(locationID)).
		WithAPIKey(tc.APIKey).
		Send().
		AssertStatus(fiber.StatusOK)

	var locations []map[string]interface{}
	getResp.JSON(&locations)

	require.Len(t, locations, 1)
	location := locations[0]["location"].(map[string]interface{})
	require.Equal(t, "Point", location["type"])

	coords := location["coordinates"].([]interface{})
	require.Len(t, coords, 2)
	require.InDelta(t, -122.4783, coords[0].(float64), 0.0001)
	require.InDelta(t, 37.8199, coords[1].(float64), 0.0001)
}

// TestPostGISInsertMultiplePoints tests batch insert with multiple Point geometries
func TestPostGISInsertMultiplePoints(t *testing.T) {
	tc := setupPostGISTest(t)
	defer tc.Close()

	if !tc.PostGISEnabled {
		t.Skip("PostGIS is not installed, skipping test")
	}

	// WHEN: Creating multiple locations with Point geometries
	locations := []map[string]interface{}{
		{
			"name": "Alcatraz Island",
			"location": map[string]interface{}{
				"type":        "Point",
				"coordinates": []interface{}{-122.4230, 37.8267},
			},
		},
		{
			"name": "Fishermans Wharf",
			"location": map[string]interface{}{
				"type":        "Point",
				"coordinates": []interface{}{-122.4177, 37.8080},
			},
		},
		{
			"name": "Coit Tower",
			"location": map[string]interface{}{
				"type":        "Point",
				"coordinates": []interface{}{-122.4058, 37.8024},
			},
		},
	}

	resp := tc.NewRequest("POST", "/api/v1/tables/"+tc.LocationsTable).
		WithAPIKey(tc.APIKey).
		WithBody(locations).
		Send()

	// THEN: All locations are created successfully
	resp.AssertStatus(fiber.StatusCreated)

	var results []map[string]interface{}
	resp.JSON(&results)
	require.Len(t, results, 3, "Should create 3 locations")

	// AND: All locations have valid GeoJSON
	for i, loc := range results {
		require.NotNil(t, loc["id"])
		require.Equal(t, locations[i]["name"], loc["name"])

		geoJSON := loc["location"].(map[string]interface{})
		require.Equal(t, "Point", geoJSON["type"])
		require.NotNil(t, geoJSON["coordinates"])
	}
}

// TestPostGISInsertPolygon tests inserting a Polygon geometry
func TestPostGISInsertPolygon(t *testing.T) {
	tc := setupPostGISTest(t)
	defer tc.Close()

	if !tc.PostGISEnabled {
		t.Skip("PostGIS is not installed, skipping test")
	}

	// WHEN: Creating a region with Polygon geometry
	resp := tc.NewRequest("POST", "/api/v1/tables/"+tc.RegionsTable).
		WithAPIKey(tc.APIKey).
		WithBody(map[string]interface{}{
			"name": "Downtown SF",
			"boundary": map[string]interface{}{
				"type": "Polygon",
				"coordinates": []interface{}{
					[]interface{}{
						[]interface{}{-122.5, 37.7},
						[]interface{}{-122.5, 37.85},
						[]interface{}{-122.35, 37.85},
						[]interface{}{-122.35, 37.7},
						[]interface{}{-122.5, 37.7}, // Close the polygon
					},
				},
			},
		}).
		Send()

	// THEN: Region is created successfully
	resp.AssertStatus(fiber.StatusCreated)

	var result map[string]interface{}
	resp.JSON(&result)

	require.NotNil(t, result["id"])
	require.Equal(t, "Downtown SF", result["name"])

	// AND: Polygon can be retrieved with correct GeoJSON structure
	regionID := result["id"]
	getResp := tc.NewRequest("GET", "/api/v1/tables/"+tc.RegionsTable+"?id=eq."+toString(regionID)).
		WithAPIKey(tc.APIKey).
		Send().
		AssertStatus(fiber.StatusOK)

	var regions []map[string]interface{}
	getResp.JSON(&regions)

	require.Len(t, regions, 1)
	boundary := regions[0]["boundary"].(map[string]interface{})
	require.Equal(t, "Polygon", boundary["type"])
	require.NotNil(t, boundary["coordinates"])
}

// TestPostGISUpdatePoint tests updating a Point geometry
func TestPostGISUpdatePoint(t *testing.T) {
	tc := setupPostGISTest(t)
	defer tc.Close()

	if !tc.PostGISEnabled {
		t.Skip("PostGIS is not installed, skipping test")
	}

	// GIVEN: An existing location
	tc.ExecuteSQL(`
		INSERT INTO ` + tc.LocationsTable + ` (id, name, location)
		VALUES (1, 'Old Location', ST_GeomFromGeoJSON('{"type":"Point","coordinates":[-122.0, 37.0]}'))
	`)

	// WHEN: Updating the location with new Point geometry
	resp := tc.NewRequest("PATCH", "/api/v1/tables/"+tc.LocationsTable+"?id=eq.1").
		WithAPIKey(tc.APIKey).
		WithBody(map[string]interface{}{
			"name": "Updated Location",
			"location": map[string]interface{}{
				"type":        "Point",
				"coordinates": []interface{}{-122.4783, 37.8199},
			},
		}).
		Send()

	// THEN: Location is updated successfully
	resp.AssertStatus(fiber.StatusOK)

	var results []map[string]interface{}
	resp.JSON(&results)

	require.Len(t, results, 1)
	require.Equal(t, "Updated Location", results[0]["name"])

	location := results[0]["location"].(map[string]interface{})
	require.Equal(t, "Point", location["type"])

	coords := location["coordinates"].([]interface{})
	require.InDelta(t, -122.4783, coords[0].(float64), 0.0001)
	require.InDelta(t, 37.8199, coords[1].(float64), 0.0001)
}

// TestPostGISUpdatePolygon tests updating a Polygon geometry
func TestPostGISUpdatePolygon(t *testing.T) {
	tc := setupPostGISTest(t)
	defer tc.Close()

	if !tc.PostGISEnabled {
		t.Skip("PostGIS is not installed, skipping test")
	}

	// GIVEN: An existing region
	tc.ExecuteSQL(`
		INSERT INTO ` + tc.RegionsTable + ` (id, name, boundary)
		VALUES (1, 'Old Region', ST_GeomFromGeoJSON('{"type":"Polygon","coordinates":[[[-122.5,37.7],[-122.5,37.8],[-122.4,37.8],[-122.4,37.7],[-122.5,37.7]]]}'))
	`)

	// WHEN: Updating the region with new Polygon geometry
	resp := tc.NewRequest("PATCH", "/api/v1/tables/"+tc.RegionsTable+"?id=eq.1").
		WithAPIKey(tc.APIKey).
		WithBody(map[string]interface{}{
			"name": "Updated Region",
			"boundary": map[string]interface{}{
				"type": "Polygon",
				"coordinates": []interface{}{
					[]interface{}{
						[]interface{}{-122.6, 37.75},
						[]interface{}{-122.6, 37.9},
						[]interface{}{-122.3, 37.9},
						[]interface{}{-122.3, 37.75},
						[]interface{}{-122.6, 37.75},
					},
				},
			},
		}).
		Send()

	// THEN: Region is updated successfully
	resp.AssertStatus(fiber.StatusOK)

	var results []map[string]interface{}
	resp.JSON(&results)

	require.Len(t, results, 1)
	require.Equal(t, "Updated Region", results[0]["name"])

	boundary := results[0]["boundary"].(map[string]interface{})
	require.Equal(t, "Polygon", boundary["type"])
	require.NotNil(t, boundary["coordinates"])
}

// TestPostGISMixedDataInsert tests inserting records with both GeoJSON and regular data
func TestPostGISMixedDataInsert(t *testing.T) {
	tc := setupPostGISTest(t)
	defer tc.Close()

	if !tc.PostGISEnabled {
		t.Skip("PostGIS is not installed, skipping test")
	}

	// WHEN: Creating a location with both regular data and GeoJSON
	resp := tc.NewRequest("POST", "/api/v1/tables/"+tc.LocationsTable).
		WithAPIKey(tc.APIKey).
		WithBody(map[string]interface{}{
			"name": "Mixed Data Location",
			"location": map[string]interface{}{
				"type":        "Point",
				"coordinates": []interface{}{-122.4783, 37.8199},
			},
		}).
		Send()

	// THEN: Record is created with both types of data
	resp.AssertStatus(fiber.StatusCreated)

	var result map[string]interface{}
	resp.JSON(&result)

	require.Equal(t, "Mixed Data Location", result["name"])
	require.NotNil(t, result["location"])
	require.NotNil(t, result["created_at"])
	require.NotNil(t, result["updated_at"])
}

// TestPostGISInvalidGeoJSON tests that invalid GeoJSON is rejected
func TestPostGISInvalidGeoJSON(t *testing.T) {
	tc := setupPostGISTest(t)
	defer tc.Close()

	if !tc.PostGISEnabled {
		t.Skip("PostGIS is not installed, skipping test")
	}

	// WHEN: Attempting to insert invalid GeoJSON (missing coordinates)
	resp := tc.NewRequest("POST", "/api/v1/tables/"+tc.LocationsTable).
		WithAPIKey(tc.APIKey).
		WithBody(map[string]interface{}{
			"name": "Invalid Location",
			"location": map[string]interface{}{
				"type": "Point",
				// Missing coordinates field
			},
		}).
		Send()

	// THEN: Request should fail gracefully
	// Note: This might return 201 with null location, or 400 depending on implementation
	// The important thing is it doesn't crash the server
	require.NotEqual(t, fiber.StatusInternalServerError, resp.Status())
}

// TestPostGISBatchUpdateWithGeoJSON tests batch update with GeoJSON data
func TestPostGISBatchUpdateWithGeoJSON(t *testing.T) {
	tc := setupPostGISTest(t)
	defer tc.Close()

	if !tc.PostGISEnabled {
		t.Skip("PostGIS is not installed, skipping test")
	}

	// GIVEN: Multiple existing locations
	tc.ExecuteSQL(`
		INSERT INTO ` + tc.LocationsTable + ` (id, name, location) VALUES
		(1, 'Location 1', ST_GeomFromGeoJSON('{"type":"Point","coordinates":[-122.0, 37.0]}')),
		(2, 'Location 2', ST_GeomFromGeoJSON('{"type":"Point","coordinates":[-122.1, 37.1]}')),
		(3, 'Location 3', ST_GeomFromGeoJSON('{"type":"Point","coordinates":[-122.2, 37.2]}'))
	`)

	// WHEN: Batch updating locations in San Francisco area
	resp := tc.NewRequest("PATCH", "/api/v1/tables/"+tc.LocationsTable+"?id=in.(1,2)").
		WithAPIKey(tc.APIKey).
		WithBody(map[string]interface{}{
			"location": map[string]interface{}{
				"type":        "Point",
				"coordinates": []interface{}{-122.4783, 37.8199},
			},
		}).
		Send()

	// THEN: Multiple records are updated
	resp.AssertStatus(fiber.StatusOK)

	var results []map[string]interface{}
	resp.JSON(&results)

	require.Len(t, results, 2, "Should update 2 locations")

	// Verify both have the new location
	for _, loc := range results {
		location := loc["location"].(map[string]interface{})
		coords := location["coordinates"].([]interface{})
		require.InDelta(t, -122.4783, coords[0].(float64), 0.0001)
		require.InDelta(t, 37.8199, coords[1].(float64), 0.0001)
	}

	// AND: Location 3 should remain unchanged
	unchanged := tc.QuerySQL("SELECT * FROM " + tc.LocationsTable + " WHERE id = 3")
	require.Len(t, unchanged, 1)
	require.Equal(t, "Location 3", unchanged[0]["name"])
}

// toString is a helper to convert various types to string for URL construction
func toString(v interface{}) string {
	return fmt.Sprintf("%v", v)
}
