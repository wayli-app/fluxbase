package api

import (
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fluxbase-eu/fluxbase/internal/database"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// isAdminUser Tests
// =============================================================================

func TestIsAdminUser(t *testing.T) {
	tests := []struct {
		name     string
		role     interface{}
		expected bool
	}{
		{
			name:     "admin role",
			role:     "admin",
			expected: true,
		},
		{
			name:     "dashboard_admin role",
			role:     "dashboard_admin",
			expected: true,
		},
		{
			name:     "authenticated role",
			role:     "authenticated",
			expected: false,
		},
		{
			name:     "anon role",
			role:     "anon",
			expected: false,
		},
		{
			name:     "service_role",
			role:     "service_role",
			expected: false,
		},
		{
			name:     "empty string",
			role:     "",
			expected: false,
		},
		{
			name:     "nil role",
			role:     nil,
			expected: false,
		},
		{
			name:     "non-string type",
			role:     123,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()

			var result bool
			app.Get("/test", func(c *fiber.Ctx) error {
				if tt.role != nil {
					c.Locals("user_role", tt.role)
				}
				result = isAdminUser(c)
				return c.SendStatus(200)
			})

			req := httptest.NewRequest("GET", "/test", nil)
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()

			assert.Equal(t, tt.expected, result)
		})
	}
}

// =============================================================================
// isGeoJSON Tests
// =============================================================================

func TestIsGeoJSON(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		expected bool
	}{
		{
			name: "valid Point",
			value: map[string]interface{}{
				"type":        "Point",
				"coordinates": []float64{0.0, 0.0},
			},
			expected: true,
		},
		{
			name: "valid LineString",
			value: map[string]interface{}{
				"type":        "LineString",
				"coordinates": [][]float64{{0.0, 0.0}, {1.0, 1.0}},
			},
			expected: true,
		},
		{
			name: "valid Polygon",
			value: map[string]interface{}{
				"type":        "Polygon",
				"coordinates": [][][]float64{{{0, 0}, {1, 0}, {1, 1}, {0, 1}, {0, 0}}},
			},
			expected: true,
		},
		{
			name: "valid MultiPoint",
			value: map[string]interface{}{
				"type":        "MultiPoint",
				"coordinates": [][]float64{{0, 0}, {1, 1}},
			},
			expected: true,
		},
		{
			name: "valid MultiLineString",
			value: map[string]interface{}{
				"type":        "MultiLineString",
				"coordinates": [][][]float64{{{0, 0}, {1, 1}}, {{2, 2}, {3, 3}}},
			},
			expected: true,
		},
		{
			name: "valid MultiPolygon",
			value: map[string]interface{}{
				"type":        "MultiPolygon",
				"coordinates": [][][][]float64{{{{0, 0}, {1, 0}, {1, 1}, {0, 0}}}},
			},
			expected: true,
		},
		{
			name: "valid GeometryCollection",
			value: map[string]interface{}{
				"type":        "GeometryCollection",
				"coordinates": []interface{}{}, // simplified
			},
			expected: true,
		},
		{
			name: "missing type",
			value: map[string]interface{}{
				"coordinates": []float64{0.0, 0.0},
			},
			expected: false,
		},
		{
			name: "missing coordinates",
			value: map[string]interface{}{
				"type": "Point",
			},
			expected: false,
		},
		{
			name: "invalid type string",
			value: map[string]interface{}{
				"type":        "InvalidType",
				"coordinates": []float64{0.0, 0.0},
			},
			expected: false,
		},
		{
			name: "type is not string",
			value: map[string]interface{}{
				"type":        123,
				"coordinates": []float64{0.0, 0.0},
			},
			expected: false,
		},
		{
			name:     "not a map",
			value:    "not a map",
			expected: false,
		},
		{
			name:     "nil value",
			value:    nil,
			expected: false,
		},
		{
			name:     "empty map",
			value:    map[string]interface{}{},
			expected: false,
		},
		{
			name: "Feature (not a geometry)",
			value: map[string]interface{}{
				"type":       "Feature",
				"geometry":   map[string]interface{}{},
				"properties": map[string]interface{}{},
			},
			expected: false, // Feature is not in validTypes
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isGeoJSON(tt.value)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// =============================================================================
// isPartialGeoJSON Tests
// =============================================================================

func TestIsPartialGeoJSON(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		expected bool
	}{
		{
			name: "has type but no coordinates",
			value: map[string]interface{}{
				"type": "Point",
			},
			expected: true,
		},
		{
			name: "has both type and coordinates",
			value: map[string]interface{}{
				"type":        "Point",
				"coordinates": []float64{0.0, 0.0},
			},
			expected: false,
		},
		{
			name: "has coordinates but no type",
			value: map[string]interface{}{
				"coordinates": []float64{0.0, 0.0},
			},
			expected: false,
		},
		{
			name:     "empty map",
			value:    map[string]interface{}{},
			expected: false,
		},
		{
			name:     "not a map",
			value:    "string",
			expected: false,
		},
		{
			name:     "nil",
			value:    nil,
			expected: false,
		},
		{
			name: "type with extra fields but no coordinates",
			value: map[string]interface{}{
				"type":       "Point",
				"crs":        map[string]interface{}{},
				"properties": map[string]interface{}{},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isPartialGeoJSON(tt.value)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// =============================================================================
// isGeometryColumn Tests
// =============================================================================

func TestIsGeometryColumn(t *testing.T) {
	tests := []struct {
		name     string
		dataType string
		expected bool
	}{
		{
			name:     "geometry type",
			dataType: "geometry",
			expected: true,
		},
		{
			name:     "geometry with SRID",
			dataType: "geometry(Point,4326)",
			expected: true,
		},
		{
			name:     "geography type",
			dataType: "geography",
			expected: true,
		},
		{
			name:     "geography with SRID",
			dataType: "geography(Point,4326)",
			expected: true,
		},
		{
			name:     "GEOMETRY uppercase",
			dataType: "GEOMETRY",
			expected: true,
		},
		{
			name:     "GEOGRAPHY uppercase",
			dataType: "GEOGRAPHY",
			expected: true,
		},
		{
			name:     "text type",
			dataType: "text",
			expected: false,
		},
		{
			name:     "integer type",
			dataType: "integer",
			expected: false,
		},
		{
			name:     "jsonb type",
			dataType: "jsonb",
			expected: false,
		},
		{
			name:     "uuid type",
			dataType: "uuid",
			expected: false,
		},
		{
			name:     "empty string",
			dataType: "",
			expected: false,
		},
		{
			name:     "geom prefix (not geometry)",
			dataType: "geom_data",
			expected: false, // doesn't contain "geometry" or "geography"
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isGeometryColumn(tt.dataType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// =============================================================================
// buildSelectColumns Tests
// =============================================================================

func TestBuildSelectColumns(t *testing.T) {
	t.Run("table with no geometry columns", func(t *testing.T) {
		table := database.TableInfo{
			Name: "users",
			Columns: []database.ColumnInfo{
				{Name: "id", DataType: "uuid"},
				{Name: "name", DataType: "text"},
				{Name: "email", DataType: "text"},
			},
		}

		result := buildSelectColumns(table)
		assert.Contains(t, result, `"id"`)
		assert.Contains(t, result, `"name"`)
		assert.Contains(t, result, `"email"`)
		assert.NotContains(t, result, "ST_AsGeoJSON")
	})

	t.Run("table with geometry column", func(t *testing.T) {
		table := database.TableInfo{
			Name: "locations",
			Columns: []database.ColumnInfo{
				{Name: "id", DataType: "uuid"},
				{Name: "name", DataType: "text"},
				{Name: "location", DataType: "geometry(Point,4326)"},
			},
		}

		result := buildSelectColumns(table)
		assert.Contains(t, result, `"id"`)
		assert.Contains(t, result, `"name"`)
		assert.Contains(t, result, "ST_AsGeoJSON")
		assert.Contains(t, result, `"location"`)
	})

	t.Run("table with geography column", func(t *testing.T) {
		table := database.TableInfo{
			Name: "routes",
			Columns: []database.ColumnInfo{
				{Name: "id", DataType: "integer"},
				{Name: "path", DataType: "geography"},
			},
		}

		result := buildSelectColumns(table)
		assert.Contains(t, result, "ST_AsGeoJSON")
	})

	t.Run("empty columns", func(t *testing.T) {
		table := database.TableInfo{
			Name:    "empty",
			Columns: []database.ColumnInfo{},
		}

		result := buildSelectColumns(table)
		assert.Empty(t, result)
	})
}

// =============================================================================
// buildReturningClause Tests
// =============================================================================

func TestBuildReturningClause(t *testing.T) {
	t.Run("returns RETURNING prefix", func(t *testing.T) {
		table := database.TableInfo{
			Name: "users",
			Columns: []database.ColumnInfo{
				{Name: "id", DataType: "uuid"},
				{Name: "name", DataType: "text"},
			},
		}

		result := buildReturningClause(table)
		assert.True(t, strings.HasPrefix(result, " RETURNING "))
	})

	t.Run("includes all columns", func(t *testing.T) {
		table := database.TableInfo{
			Name: "items",
			Columns: []database.ColumnInfo{
				{Name: "id", DataType: "integer"},
				{Name: "title", DataType: "text"},
				{Name: "created_at", DataType: "timestamp"},
			},
		}

		result := buildReturningClause(table)
		assert.Contains(t, result, `"id"`)
		assert.Contains(t, result, `"title"`)
		assert.Contains(t, result, `"created_at"`)
	})

	t.Run("converts geometry columns", func(t *testing.T) {
		table := database.TableInfo{
			Name: "places",
			Columns: []database.ColumnInfo{
				{Name: "id", DataType: "uuid"},
				{Name: "geom", DataType: "geometry"},
			},
		}

		result := buildReturningClause(table)
		assert.Contains(t, result, "ST_AsGeoJSON")
	})
}

// =============================================================================
// quoteIdentifier Tests
// =============================================================================

func TestQuoteIdentifier_CRUD(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple identifier",
			input:    "column_name",
			expected: `"column_name"`,
		},
		{
			name:     "identifier with uppercase",
			input:    "ColumnName",
			expected: `"ColumnName"`,
		},
		{
			name:     "identifier with numbers",
			input:    "column1",
			expected: `"column1"`,
		},
		{
			name:     "identifier with underscore",
			input:    "my_column",
			expected: `"my_column"`,
		},
		{
			name:     "embedded double quote",
			input:    `col"name`,
			expected: "", // invalid identifier
		},
		{
			name:     "SQL injection attempt",
			input:    "col; DROP TABLE users;--",
			expected: "", // invalid identifier
		},
		{
			name:     "empty string",
			input:    "",
			expected: "", // invalid
		},
		{
			name:     "single character",
			input:    "a",
			expected: `"a"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := quoteIdentifier(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// =============================================================================
// isValidIdentifier Tests
// =============================================================================

func TestIsValidIdentifier_CRUD(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "simple lowercase",
			input:    "column",
			expected: true,
		},
		{
			name:     "with underscore",
			input:    "column_name",
			expected: true,
		},
		{
			name:     "with numbers",
			input:    "column123",
			expected: true,
		},
		{
			name:     "starts with underscore",
			input:    "_private",
			expected: true,
		},
		{
			name:     "mixed case",
			input:    "ColumnName",
			expected: true,
		},
		{
			name:     "SQL injection",
			input:    "col; DROP TABLE--",
			expected: false,
		},
		{
			name:     "with quotes",
			input:    `col"name`,
			expected: false,
		},
		{
			name:     "with semicolon",
			input:    "col;name",
			expected: false,
		},
		{
			name:     "empty string",
			input:    "",
			expected: false,
		},
		{
			name:     "with spaces",
			input:    "column name",
			expected: false,
		},
		{
			name:     "with dash",
			input:    "column-name",
			expected: false,
		},
		{
			name:     "with special chars",
			input:    "col@name",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidIdentifier(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// =============================================================================
// RESTHandler Method Tests
// =============================================================================

func TestRESTHandler_getConflictTarget(t *testing.T) {
	handler := &RESTHandler{}

	t.Run("single primary key", func(t *testing.T) {
		table := database.TableInfo{
			Name:       "users",
			PrimaryKey: []string{"id"},
		}

		result := handler.getConflictTarget(table)
		assert.Equal(t, `"id"`, result)
	})

	t.Run("composite primary key", func(t *testing.T) {
		table := database.TableInfo{
			Name:       "user_roles",
			PrimaryKey: []string{"user_id", "role_id"},
		}

		result := handler.getConflictTarget(table)
		assert.Contains(t, result, `"user_id"`)
		assert.Contains(t, result, `"role_id"`)
		assert.Contains(t, result, ", ")
	})

	t.Run("no primary key", func(t *testing.T) {
		table := database.TableInfo{
			Name:       "logs",
			PrimaryKey: []string{},
		}

		result := handler.getConflictTarget(table)
		assert.Empty(t, result)
	})
}

func TestRESTHandler_getConflictTargetUnquoted(t *testing.T) {
	handler := &RESTHandler{}

	t.Run("returns primary key columns", func(t *testing.T) {
		table := database.TableInfo{
			PrimaryKey: []string{"id", "tenant_id"},
		}

		result := handler.getConflictTargetUnquoted(table)
		assert.Equal(t, []string{"id", "tenant_id"}, result)
	})

	t.Run("empty primary key", func(t *testing.T) {
		table := database.TableInfo{
			PrimaryKey: []string{},
		}

		result := handler.getConflictTargetUnquoted(table)
		assert.Empty(t, result)
	})
}

func TestRESTHandler_isInConflictTarget(t *testing.T) {
	handler := &RESTHandler{}

	tests := []struct {
		name           string
		column         string
		conflictTarget []string
		expected       bool
	}{
		{
			name:           "column in target",
			column:         "id",
			conflictTarget: []string{"id", "tenant_id"},
			expected:       true,
		},
		{
			name:           "column not in target",
			column:         "name",
			conflictTarget: []string{"id", "tenant_id"},
			expected:       false,
		},
		{
			name:           "empty target",
			column:         "id",
			conflictTarget: []string{},
			expected:       false,
		},
		{
			name:           "single column target",
			column:         "id",
			conflictTarget: []string{"id"},
			expected:       true,
		},
		{
			name:           "case sensitive",
			column:         "ID",
			conflictTarget: []string{"id"},
			expected:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.isInConflictTarget(tt.column, tt.conflictTarget)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// =============================================================================
// Handler Integration Tests (with mock Fiber app)
// =============================================================================

func TestMakePostHandler_ValidationErrors(t *testing.T) {
	app := fiber.New()
	handler := &RESTHandler{}

	table := database.TableInfo{
		Schema:     "public",
		Name:       "items",
		PrimaryKey: []string{"id"},
		Columns: []database.ColumnInfo{
			{Name: "id", DataType: "uuid"},
			{Name: "name", DataType: "text"},
		},
	}

	app.Post("/items", handler.makePostHandler(table))

	t.Run("invalid JSON body", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/items", strings.NewReader(`{invalid`))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, 400, resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		assert.Contains(t, string(body), "Invalid request body")
	})

	t.Run("unknown column", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/items", strings.NewReader(`{"unknown_column":"value"}`))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, 400, resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		assert.Contains(t, string(body), "Unknown column")
	})

	t.Run("partial GeoJSON returns error", func(t *testing.T) {
		// First, add a geometry column
		tableWithGeom := database.TableInfo{
			Schema:     "public",
			Name:       "places",
			PrimaryKey: []string{"id"},
			Columns: []database.ColumnInfo{
				{Name: "id", DataType: "uuid"},
				{Name: "location", DataType: "geometry"},
			},
		}

		app2 := fiber.New()
		app2.Post("/places", handler.makePostHandler(tableWithGeom))

		// Send partial GeoJSON (missing coordinates)
		req := httptest.NewRequest("POST", "/places", strings.NewReader(`{"id":"123","location":{"type":"Point"}}`))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app2.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, 400, resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		assert.Contains(t, string(body), "Invalid GeoJSON")
	})
}

func TestMakePutHandler_ValidationErrors(t *testing.T) {
	app := fiber.New()
	handler := &RESTHandler{}

	table := database.TableInfo{
		Schema:     "public",
		Name:       "items",
		PrimaryKey: []string{"id"},
		Columns: []database.ColumnInfo{
			{Name: "id", DataType: "uuid"},
			{Name: "name", DataType: "text"},
		},
	}

	app.Put("/items/:id", handler.makePutHandler(table))

	t.Run("invalid JSON body", func(t *testing.T) {
		req := httptest.NewRequest("PUT", "/items/123", strings.NewReader(`{invalid`))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, 400, resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		assert.Contains(t, string(body), "Invalid request body")
	})

	t.Run("unknown column", func(t *testing.T) {
		req := httptest.NewRequest("PUT", "/items/123", strings.NewReader(`{"unknown_column":"value"}`))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, 400, resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		assert.Contains(t, string(body), "Unknown column")
	})
}

// =============================================================================
// Prefer Header Parsing Tests
// =============================================================================

func TestPreferHeaderParsing(t *testing.T) {
	tests := []struct {
		name             string
		preferHeader     string
		isUpsert         bool
		ignoreDuplicates bool
		defaultToNull    bool
	}{
		{
			name:             "merge-duplicates",
			preferHeader:     "resolution=merge-duplicates",
			isUpsert:         true,
			ignoreDuplicates: false,
		},
		{
			name:             "ignore-duplicates",
			preferHeader:     "resolution=ignore-duplicates",
			isUpsert:         true,
			ignoreDuplicates: true,
		},
		{
			name:          "missing=default",
			preferHeader:  "missing=default",
			isUpsert:      false,
			defaultToNull: true,
		},
		{
			name:          "combined preferences",
			preferHeader:  "resolution=merge-duplicates, missing=default",
			isUpsert:      true,
			defaultToNull: true,
		},
		{
			name:             "empty header",
			preferHeader:     "",
			isUpsert:         false,
			ignoreDuplicates: false,
			defaultToNull:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isUpsert := strings.Contains(tt.preferHeader, "resolution=merge-duplicates") || strings.Contains(tt.preferHeader, "resolution=ignore-duplicates")
			ignoreDuplicates := strings.Contains(tt.preferHeader, "resolution=ignore-duplicates")
			defaultToNull := strings.Contains(tt.preferHeader, "missing=default")

			assert.Equal(t, tt.isUpsert, isUpsert)
			assert.Equal(t, tt.ignoreDuplicates, ignoreDuplicates)
			assert.Equal(t, tt.defaultToNull, defaultToNull)
		})
	}
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkIsGeoJSON(b *testing.B) {
	value := map[string]interface{}{
		"type":        "Point",
		"coordinates": []float64{0.0, 0.0},
	}

	for i := 0; i < b.N; i++ {
		_ = isGeoJSON(value)
	}
}

func BenchmarkIsPartialGeoJSON(b *testing.B) {
	value := map[string]interface{}{
		"type": "Point",
	}

	for i := 0; i < b.N; i++ {
		_ = isPartialGeoJSON(value)
	}
}

func BenchmarkQuoteIdentifier(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = quoteIdentifier("column_name")
	}
}

func BenchmarkIsValidIdentifier(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = isValidIdentifier("column_name")
	}
}

func BenchmarkBuildSelectColumns(b *testing.B) {
	table := database.TableInfo{
		Name: "users",
		Columns: []database.ColumnInfo{
			{Name: "id", DataType: "uuid"},
			{Name: "name", DataType: "text"},
			{Name: "email", DataType: "text"},
			{Name: "created_at", DataType: "timestamp"},
			{Name: "location", DataType: "geometry(Point,4326)"},
		},
	}

	for i := 0; i < b.N; i++ {
		_ = buildSelectColumns(table)
	}
}

func BenchmarkBuildReturningClause(b *testing.B) {
	table := database.TableInfo{
		Name: "users",
		Columns: []database.ColumnInfo{
			{Name: "id", DataType: "uuid"},
			{Name: "name", DataType: "text"},
			{Name: "email", DataType: "text"},
		},
	}

	for i := 0; i < b.N; i++ {
		_ = buildReturningClause(table)
	}
}

func BenchmarkIsInConflictTarget(b *testing.B) {
	handler := &RESTHandler{}
	conflictTarget := []string{"id", "tenant_id", "org_id"}

	for i := 0; i < b.N; i++ {
		_ = handler.isInConflictTarget("name", conflictTarget)
	}
}
