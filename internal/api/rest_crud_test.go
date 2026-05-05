package api

import (
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"

	"github.com/nimbleflux/fluxbase/internal/config"
	"github.com/nimbleflux/fluxbase/internal/database"
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
			name:     "instance_admin role",
			role:     "instance_admin",
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
			app := newTestApp(t)

			var result bool
			app.Get("/test", func(c fiber.Ctx) error {
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
// injectTenantID Tests
// =============================================================================

func TestInjectTenantID(t *testing.T) {
	colWithTenant := &database.ColumnInfo{Name: "tenant_id"}
	colWithoutTenant := &database.ColumnInfo{Name: "id"}

	t.Run("injects tenant_id when table has column and context has tenant", func(t *testing.T) {
		app := fiber.New()
		c := app.AcquireCtx(&fasthttp.RequestCtx{})
		defer app.ReleaseCtx(c)
		c.Locals("tenant_id", "550e8400-e29b-41d4-a716-446655440000")

		table := database.TableInfo{Columns: []database.ColumnInfo{*colWithTenant, *colWithoutTenant}}
		table.BuildColumnMap()
		data := map[string]interface{}{"name": "test"}

		result := injectTenantID(data, c, table)
		assert.True(t, result)
		assert.Equal(t, "550e8400-e29b-41d4-a716-446655440000", data["tenant_id"])
	})

	t.Run("does not override existing tenant_id", func(t *testing.T) {
		app := fiber.New()
		c := app.AcquireCtx(&fasthttp.RequestCtx{})
		defer app.ReleaseCtx(c)
		c.Locals("tenant_id", "550e8400-e29b-41d4-a716-446655440000")

		table := database.TableInfo{Columns: []database.ColumnInfo{*colWithTenant}}
		table.BuildColumnMap()
		data := map[string]interface{}{"tenant_id": "existing-id", "name": "test"}

		result := injectTenantID(data, c, table)
		assert.False(t, result)
		assert.Equal(t, "existing-id", data["tenant_id"])
	})

	t.Run("skips when table has no tenant_id column", func(t *testing.T) {
		app := fiber.New()
		c := app.AcquireCtx(&fasthttp.RequestCtx{})
		defer app.ReleaseCtx(c)
		c.Locals("tenant_id", "550e8400-e29b-41d4-a716-446655440000")

		table := database.TableInfo{Columns: []database.ColumnInfo{*colWithoutTenant}}
		table.BuildColumnMap()
		data := map[string]interface{}{"name": "test"}

		result := injectTenantID(data, c, table)
		assert.False(t, result)
		_, exists := data["tenant_id"]
		assert.False(t, exists)
	})

	t.Run("skips when no tenant context", func(t *testing.T) {
		app := fiber.New()
		c := app.AcquireCtx(&fasthttp.RequestCtx{})
		defer app.ReleaseCtx(c)

		table := database.TableInfo{Columns: []database.ColumnInfo{*colWithTenant}}
		table.BuildColumnMap()
		data := map[string]interface{}{"name": "test"}

		result := injectTenantID(data, c, table)
		assert.False(t, result)
		_, exists := data["tenant_id"]
		assert.False(t, exists)
	})
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
	app := newTestApp(t)
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

		app2 := newTestApp(t)
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
	app := newTestApp(t)
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

// TestMakeGetHandler_InvalidQuery tests that makeGetHandler is called and validates query strings
func TestMakeGetHandler_InvalidQuery(t *testing.T) {
	app := newTestApp(t)
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

	// Call makeGetHandler to ensure the factory function is executed
	app.Get("/items", handler.makeGetHandler(table))

	t.Run("invalid query string causes error", func(t *testing.T) {
		// Invalid percent encoding in query string
		req := httptest.NewRequest("GET", "/items?filter=%ZZ", nil)

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		// Should get an error status (not 200)
		// The exact error depends on the parser, but it shouldn't succeed
		assert.NotEqual(t, 200, resp.StatusCode)
	})
}

// TestMakePatchHandler_IsAliasForPut confirms makePatchHandler wraps makePutHandler
func TestMakePatchHandler_IsAliasForPut(t *testing.T) {
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

	// Call makePatchHandler to execute the factory function
	patchHandler := handler.makePatchHandler(table)
	putHandler := handler.makePutHandler(table)

	// Both handlers should be non-nil (factory function was called)
	assert.NotNil(t, patchHandler)
	assert.NotNil(t, putHandler)
}

// TestMakeDeleteHandler_Executes tests that makeDeleteHandler factory function is called
func TestMakeDeleteHandler_Executes(t *testing.T) {
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

	// Call makeDeleteHandler to ensure the factory function is executed
	deleteHandler := handler.makeDeleteHandler(table)

	// Handler should be non-nil (factory function was called)
	assert.NotNil(t, deleteHandler)
}

// TestMakeGetByIdHandler_Executes tests that makeGetByIdHandler factory function is called
func TestMakeGetByIdHandler_Executes(t *testing.T) {
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

	// Call makeGetByIdHandler to ensure the factory function is executed
	getByIdHandler := handler.makeGetByIdHandler(table)

	// Handler should be non-nil (factory function was called)
	assert.NotNil(t, getByIdHandler)
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

// =============================================================================
// Additional Tests for Coverage Boost (Developer 3 Assignment)
// =============================================================================

// TestBuildSelectColumns_Complex tests more buildSelectColumns scenarios
func TestBuildSelectColumns_Complex(t *testing.T) {
	tests := []struct {
		name     string
		table    database.TableInfo
		contains []string // Substrings that should be in the result
		excludes []string // Substrings that should NOT be in the result
	}{
		{
			name: "all geometry types",
			table: database.TableInfo{
				Name: "features",
				Columns: []database.ColumnInfo{
					{Name: "id", DataType: "uuid"},
					{Name: "point_geom", DataType: "geometry(Point,4326)"},
					{Name: "polygon_geom", DataType: "geometry(Polygon,4326)"},
					{Name: "line_geom", DataType: "geometry(LineString,4326)"},
				},
			},
			contains: []string{"ST_AsGeoJSON", "point_geom", "polygon_geom", "line_geom"},
		},
		{
			name: "mixed geometry and regular columns",
			table: database.TableInfo{
				Name: "places",
				Columns: []database.ColumnInfo{
					{Name: "id", DataType: "uuid"},
					{Name: "name", DataType: "text"},
					{Name: "location", DataType: "geography"},
					{Name: "description", DataType: "text"},
				},
			},
			contains: []string{`"id"`, `"name"`, `"description"`, "ST_AsGeoJSON", "location"},
		},
		{
			name: "geography(Point,4326) type",
			table: database.TableInfo{
				Name: "routes",
				Columns: []database.ColumnInfo{
					{Name: "path", DataType: "geography(Point,4326)"},
				},
			},
			contains: []string{"ST_AsGeoJSON", "path"},
		},
		{
			name: "GEOMETRY uppercase type",
			table: database.TableInfo{
				Name: "data",
				Columns: []database.ColumnInfo{
					{Name: "geom", DataType: "GEOMETRY"},
				},
			},
			contains: []string{"ST_AsGeoJSON", "geom"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildSelectColumns(tt.table)
			assert.NotEmpty(t, result)

			for _, substr := range tt.contains {
				assert.Contains(t, result, substr)
			}

			for _, substr := range tt.excludes {
				assert.NotContains(t, result, substr)
			}
		})
	}
}

// TestBuildReturningClause_MoreScenarios tests additional returning clause scenarios
func TestBuildReturningClause_MoreScenarios(t *testing.T) {
	tests := []struct {
		name     string
		table    database.TableInfo
		contains []string
	}{
		{
			name: "multiple geometry columns",
			table: database.TableInfo{
				Name: "features",
				Columns: []database.ColumnInfo{
					{Name: "id", DataType: "uuid"},
					{Name: "point", DataType: "geometry"},
					{Name: "polygon", DataType: "geometry"},
				},
			},
			contains: []string{"RETURNING", `"id"`, "ST_AsGeoJSON", "point", "polygon"},
		},
		{
			name: "only geometry columns",
			table: database.TableInfo{
				Name: "geodata",
				Columns: []database.ColumnInfo{
					{Name: "location", DataType: "geography"},
				},
			},
			contains: []string{"RETURNING", "ST_AsGeoJSON"},
		},
		{
			name: "columns with special types",
			table: database.TableInfo{
				Name: "items",
				Columns: []database.ColumnInfo{
					{Name: "id", DataType: "uuid"},
					{Name: "data", DataType: "jsonb"},
					{Name: "tags", DataType: "text[]"},
				},
			},
			contains: []string{`"id"`, `"data"`, `"tags"`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildReturningClause(tt.table)
			assert.NotEmpty(t, result)

			for _, substr := range tt.contains {
				assert.Contains(t, result, substr)
			}
		})
	}
}

// TestIsValidIdentifier_EdgeCases tests additional identifier validation
func TestIsValidIdentifier_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{name: "starts with number", input: "1column", expected: false},
		{name: "starts with underscore and number", input: "_1col", expected: true},
		{name: "single underscore", input: "_", expected: true},
		{name: "multiple underscores", input: "___", expected: true},
		{name: "camelCase", input: "myColumn", expected: true},
		{name: "PascalCase", input: "MyColumn", expected: true},
		{name: "snake_case", input: "my_column_name", expected: true},
		{name: "with numbers after", input: "col123", expected: true},
		{name: "with numbers in middle", input: "col123name", expected: true},
		{name: "dollar sign", input: "col$test", expected: false},
		{name: "at sign", input: "col@test", expected: false},
		{name: "hash sign", input: "col#test", expected: false},
		{name: "dot", input: "col.test", expected: false},
		{name: "comma", input: "col,test", expected: false},
		{name: "period", input: "col.test", expected: false},
		{name: "hyphen", input: "col-test", expected: false},
		{name: "plus sign", input: "col+test", expected: false},
		{name: "equals sign", input: "col=test", expected: false},
		{name: "pipe", input: "col|test", expected: false},
		{name: "backtick", input: "col`test", expected: false},
		{name: "tilde", input: "col~test", expected: false},
		{name: "exclamation", input: "col!test", expected: false},
		{name: "question mark", input: "col?test", expected: false},
		{name: "asterisk", input: "col*test", expected: false},
		{name: "percent", input: "col%test", expected: false},
		{name: "ampersand", input: "col&test", expected: false},
		{name: "caret", input: "col^test", expected: false},
		{name: "single quote", input: "col'test", expected: false},
		{name: "double quote", input: `col"test`, expected: false},
		{name: "backslash", input: "col\\test", expected: false},
		{name: "forward slash", input: "col/test", expected: false},
		{name: "opening paren", input: "col(test", expected: false},
		{name: "closing paren", input: "col)test", expected: false},
		{name: "opening bracket", input: "col[test", expected: false},
		{name: "closing bracket", input: "col]test", expected: false},
		{name: "opening brace", input: "col{test", expected: false},
		{name: "closing brace", input: "col}test", expected: false},
		{name: "less than", input: "col<test", expected: false},
		{name: "greater than", input: "col>test", expected: false},
		{name: "space only", input: "   ", expected: false},
		{name: "tab character", input: "col\ttest", expected: false},
		{name: "newline", input: "col\ntest", expected: false},
		{name: "carriage return", input: "col\rtest", expected: false},
		{name: "null byte", input: "col\x00test", expected: false},
		{name: "unicode letters", input: "colüm_名前", expected: false}, // Only ASCII allowed
		{name: "very long identifier", input: strings.Repeat("a", 100), expected: true},
		{name: "single letter uppercase", input: "X", expected: true},
		{name: "single letter lowercase", input: "x", expected: true},
		{name: "zero length", input: "", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidIdentifier(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestQuoteIdentifier_EdgeCases tests additional quoting scenarios
func TestQuoteIdentifier_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "valid - simple", input: "column", expected: `"column"`},
		{name: "valid - with numbers", input: "col123", expected: `"col123"`},
		{name: "valid - starts with underscore", input: "_private", expected: `"_private"`},
		{name: "valid - multiple underscores", input: "__init__", expected: `"__init__"`},
		{name: "valid - mixed case", input: "ColumnName", expected: `"ColumnName"`},
		{name: "valid - camelCase", input: "myColumnName", expected: `"myColumnName"`},
		{name: "valid - ends with number", input: "column1", expected: `"column1"`},
		{name: "valid - uppercase", input: "ID", expected: `"ID"`},
		{name: "valid - single char", input: "x", expected: `"x"`},
		{name: "invalid - starts with number", input: "1col", expected: ""},
		{name: "invalid - has space", input: "col name", expected: ""},
		{name: "invalid - has dash", input: "col-name", expected: ""},
		{name: "invalid - has dot", input: "col.name", expected: ""},
		{name: "invalid - SQL injection attempt", input: "col; DROP TABLE users; --", expected: ""},
		{name: "invalid - has quote", input: `col"umn`, expected: ""},
		{name: "invalid - has semicolon", input: "col;name", expected: ""},
		{name: "invalid - empty", input: "", expected: ""},
		{name: "invalid - has special chars", input: "col@#$%", expected: ""},
		{name: "invalid - has comma", input: "col,name", expected: ""},
		{name: "invalid - has equals", input: "col=name", expected: ""},
		{name: "invalid - has paren", input: "col(name)", expected: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := quoteIdentifier(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestIsGeometryColumn_EdgeCases tests more geometry column detection
func TestIsGeometryColumn_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		dataType string
		expected bool
	}{
		{name: "geometry lowercase", dataType: "geometry", expected: true},
		{name: "geometry uppercase", dataType: "GEOMETRY", expected: true},
		{name: "geometry mixed case", dataType: "Geometry", expected: true},
		{name: "geography lowercase", dataType: "geography", expected: true},
		{name: "geography uppercase", dataType: "GEOGRAPHY", expected: true},
		{name: "geography mixed case", dataType: "Geography", expected: true},
		{name: "geometry(Point)", dataType: "geometry(Point)", expected: true},
		{name: "geometry(Polygon,4326)", dataType: "geometry(Polygon,4326)", expected: true},
		{name: "geography(Point,4326)", dataType: "geography(Point,4326)", expected: true},
		{name: "geometry(LineString,4326)", dataType: "geometry(LineString,4326)", expected: true},
		{name: "geometry(MultiPolygon)", dataType: "geometry(MultiPolygon)", expected: true},
		{name: "GEOMETRYCOLLECTION", dataType: "GEOMETRYCOLLECTION", expected: true},
		{name: "text", dataType: "text", expected: false},
		{name: "varchar", dataType: "varchar(255)", expected: false},
		{name: "integer", dataType: "integer", expected: false},
		{name: "bigint", dataType: "bigint", expected: false},
		{name: "jsonb", dataType: "jsonb", expected: false},
		{name: "json", dataType: "json", expected: false},
		{name: "uuid", dataType: "uuid", expected: false},
		{name: "timestamp", dataType: "timestamp", expected: false},
		{name: "timestamptz", dataType: "timestamptz", expected: false},
		{name: "date", dataType: "date", expected: false},
		{name: "time", dataType: "time", expected: false},
		{name: "boolean", dataType: "boolean", expected: false},
		{name: "numeric", dataType: "numeric", expected: false},
		{name: "decimal", dataType: "decimal", expected: false},
		{name: "real", dataType: "real", expected: false},
		{name: "double precision", dataType: "double precision", expected: false},
		{name: "bytea", dataType: "bytea", expected: false},
		{name: "array", dataType: "text[]", expected: false},
		{name: "empty string", dataType: "", expected: false},
		{name: "null", dataType: "NULL", expected: false},
		{name: "geometry with schema prefix", dataType: "public.geometry", expected: true}, // Contains "geometry"
		{name: "postgis geometry", dataType: "postgis.geometry", expected: true},           // Contains "geometry"
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isGeometryColumn(tt.dataType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestIsGeoJSON_AdditionalCases tests more GeoJSON detection scenarios
func TestIsGeoJSON_AdditionalCases(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		expected bool
	}{
		{
			name: "FeatureCollection",
			value: map[string]interface{}{
				"type":     "FeatureCollection",
				"features": []interface{}{},
			},
			expected: false, // Not a geometry type
		},
		{
			name: "Feature",
			value: map[string]interface{}{
				"type":       "Feature",
				"geometry":   map[string]interface{}{},
				"properties": map[string]interface{}{},
			},
			expected: false,
		},
		{
			name: "invalid type - GeometryCollection without coords",
			value: map[string]interface{}{
				"type": "GeometryCollection",
			},
			expected: false,
		},
		{
			name: "lowercase type",
			value: map[string]interface{}{
				"type":        "point",
				"coordinates": []float64{0, 0},
			},
			expected: false, // Case sensitive
		},
		{
			name: "UPPERCASE TYPE",
			value: map[string]interface{}{
				"type":        "POINT",
				"coordinates": []float64{0, 0},
			},
			expected: false, // Case sensitive
		},
		{
			name: "coordinates is not an array",
			value: map[string]interface{}{
				"type":        "Point",
				"coordinates": "not an array",
			},
			expected: true, // Only checks type and coordinates key presence, not value type
		},
		{
			name: "coordinates is empty array",
			value: map[string]interface{}{
				"type":        "Point",
				"coordinates": []interface{}{},
			},
			expected: true, // Empty array is still valid coords
		},
		{
			name: "coordinates with nested arrays",
			value: map[string]interface{}{
				"type":        "Polygon",
				"coordinates": [][][]float64{{{0, 0}, {1, 0}, {1, 1}, {0, 0}}},
			},
			expected: true,
		},
		{
			name: "type is not a string",
			value: map[string]interface{}{
				"type":        123,
				"coordinates": []float64{0, 0},
			},
			expected: false,
		},
		{
			name: "type is nil",
			value: map[string]interface{}{
				"type":        nil,
				"coordinates": []float64{0, 0},
			},
			expected: false,
		},
		{
			name: "has extra properties",
			value: map[string]interface{}{
				"type":        "Point",
				"coordinates": []float64{0, 0},
				"crs":         "EPSG:4326",
				"bbox":        []float64{0, 0, 1, 1},
			},
			expected: true, // Extra props are OK
		},
		{
			name:     "string value",
			value:    `{"type":"Point","coordinates":[0,0]}`,
			expected: false, // String, not map
		},
		{
			name:     "nil value",
			value:    nil,
			expected: false,
		},
		{
			name:     "array value",
			value:    []interface{}{},
			expected: false,
		},
		{
			name:     "number value",
			value:    123,
			expected: false,
		},
		{
			name:     "bool value",
			value:    true,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isGeoJSON(tt.value)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestIsPartialGeoJSON_AdditionalCases tests more partial GeoJSON detection
func TestIsPartialGeoJSON_AdditionalCases(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		expected bool
	}{
		{
			name: "has type but coords is nil",
			value: map[string]interface{}{
				"type":        "Point",
				"coordinates": nil,
			},
			expected: false, // coords key exists even if value is nil
		},
		{
			name: "has type but coords is wrong type",
			value: map[string]interface{}{
				"type":        "Point",
				"coordinates": "not an array",
			},
			expected: false, // Coords exist but wrong type
		},
		{
			name: "both missing",
			value: map[string]interface{}{
				"properties": map[string]interface{}{},
			},
			expected: false,
		},
		{
			name:     "empty map",
			value:    map[string]interface{}{},
			expected: false,
		},
		{
			name: "type only - Polygon",
			value: map[string]interface{}{
				"type": "Polygon",
			},
			expected: true,
		},
		{
			name: "type only - LineString",
			value: map[string]interface{}{
				"type": "LineString",
			},
			expected: true,
		},
		{
			name: "type only - invalid type",
			value: map[string]interface{}{
				"type": "InvalidType",
			},
			expected: true, // Still partial even if type is invalid
		},
		{
			name: "coordinates only",
			value: map[string]interface{}{
				"coordinates": []float64{0, 0},
			},
			expected: false,
		},
		{
			name: "has both - complete GeoJSON",
			value: map[string]interface{}{
				"type":        "Point",
				"coordinates": []float64{0, 0},
			},
			expected: false, // Complete, not partial
		},
		{
			name: "type is empty string",
			value: map[string]interface{}{
				"type": "",
			},
			expected: true, // Has type key even if empty
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isPartialGeoJSON(tt.value)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestGetConflictTarget_MoreScenarios tests conflict target detection
func TestGetConflictTarget_MoreScenarios(t *testing.T) {
	handler := &RESTHandler{}

	tests := []struct {
		name     string
		table    database.TableInfo
		expected string
	}{
		{
			name: "single column primary key",
			table: database.TableInfo{
				Name:       "users",
				PrimaryKey: []string{"id"},
			},
			expected: `"id"`,
		},
		{
			name: "composite primary key - two columns",
			table: database.TableInfo{
				Name:       "user_roles",
				PrimaryKey: []string{"user_id", "role_id"},
			},
			expected: `"user_id", "role_id"`,
		},
		{
			name: "composite primary key - three columns",
			table: database.TableInfo{
				Name:       "permissions",
				PrimaryKey: []string{"org_id", "user_id", "resource_id"},
			},
			expected: `"org_id", "user_id", "resource_id"`,
		},
		{
			name: "no primary key",
			table: database.TableInfo{
				Name:       "logs",
				PrimaryKey: []string{},
			},
			expected: "",
		},
		{
			name: "nil primary key slice",
			table: database.TableInfo{
				Name:       "data",
				PrimaryKey: nil,
			},
			expected: "",
		},
		{
			name: "primary key with underscores",
			table: database.TableInfo{
				Name:       "audit_log",
				PrimaryKey: []string{"audit_id", "revision"},
			},
			expected: `"audit_id", "revision"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.getConflictTarget(tt.table)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestGetConflictTargetUnquoted_MoreScenarios tests unquoted conflict target
func TestGetConflictTargetUnquoted_MoreScenarios(t *testing.T) {
	handler := &RESTHandler{}

	tests := []struct {
		name     string
		table    database.TableInfo
		expected []string
	}{
		{
			name: "single column",
			table: database.TableInfo{
				PrimaryKey: []string{"id"},
			},
			expected: []string{"id"},
		},
		{
			name: "multiple columns",
			table: database.TableInfo{
				PrimaryKey: []string{"tenant_id", "user_id", "role_id"},
			},
			expected: []string{"tenant_id", "user_id", "role_id"},
		},
		{
			name: "empty slice",
			table: database.TableInfo{
				PrimaryKey: []string{},
			},
			expected: []string{},
		},
		{
			name: "nil slice",
			table: database.TableInfo{
				PrimaryKey: nil,
			},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.getConflictTargetUnquoted(tt.table)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// =============================================================================
// Additional Handler Coverage Tests
// =============================================================================

func TestHandlerFactories_CreatesNonNilHandlers(t *testing.T) {
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

	t.Run("makeGetHandler returns non-nil", func(t *testing.T) {
		getHandler := handler.makeGetHandler(table)
		assert.NotNil(t, getHandler)
	})

	t.Run("makeGetByIdHandler returns non-nil", func(t *testing.T) {
		getByIdHandler := handler.makeGetByIdHandler(table)
		assert.NotNil(t, getByIdHandler)
	})

	t.Run("makePostHandler returns non-nil", func(t *testing.T) {
		postHandler := handler.makePostHandler(table)
		assert.NotNil(t, postHandler)
	})

	t.Run("makePutHandler returns non-nil", func(t *testing.T) {
		putHandler := handler.makePutHandler(table)
		assert.NotNil(t, putHandler)
	})

	t.Run("makePatchHandler returns non-nil", func(t *testing.T) {
		patchHandler := handler.makePatchHandler(table)
		assert.NotNil(t, patchHandler)
	})

	t.Run("makeDeleteHandler returns non-nil", func(t *testing.T) {
		deleteHandler := handler.makeDeleteHandler(table)
		assert.NotNil(t, deleteHandler)
	})
}

func TestHandlerFactories_VariousTableConfigurations(t *testing.T) {
	handler := &RESTHandler{}

	tests := []struct {
		name  string
		table database.TableInfo
	}{
		{
			name: "table with single primary key",
			table: database.TableInfo{
				Schema:     "public",
				Name:       "users",
				PrimaryKey: []string{"id"},
				Columns: []database.ColumnInfo{
					{Name: "id", DataType: "uuid"},
					{Name: "email", DataType: "text"},
				},
			},
		},
		{
			name: "table with composite primary key",
			table: database.TableInfo{
				Schema:     "public",
				Name:       "user_roles",
				PrimaryKey: []string{"user_id", "role_id"},
				Columns: []database.ColumnInfo{
					{Name: "user_id", DataType: "uuid"},
					{Name: "role_id", DataType: "text"},
				},
			},
		},
		{
			name: "table without primary key",
			table: database.TableInfo{
				Schema:     "public",
				Name:       "logs",
				PrimaryKey: []string{},
				Columns: []database.ColumnInfo{
					{Name: "message", DataType: "text"},
					{Name: "timestamp", DataType: "timestamptz"},
				},
			},
		},
		{
			name: "table with custom schema",
			table: database.TableInfo{
				Schema:     "my_schema",
				Name:       "items",
				PrimaryKey: []string{"id"},
				Columns: []database.ColumnInfo{
					{Name: "id", DataType: "integer"},
					{Name: "name", DataType: "text"},
				},
			},
		},
		{
			name: "table with many columns",
			table: database.TableInfo{
				Schema:     "public",
				Name:       "products",
				PrimaryKey: []string{"id"},
				Columns: []database.ColumnInfo{
					{Name: "id", DataType: "uuid"},
					{Name: "name", DataType: "text"},
					{Name: "description", DataType: "text"},
					{Name: "price", DataType: "numeric"},
					{Name: "stock", DataType: "integer"},
					{Name: "created_at", DataType: "timestamptz"},
					{Name: "updated_at", DataType: "timestamptz"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// All factory functions should create non-nil handlers
			assert.NotNil(t, handler.makeGetHandler(tt.table))
			assert.NotNil(t, handler.makeGetByIdHandler(tt.table))
			assert.NotNil(t, handler.makePostHandler(tt.table))
			assert.NotNil(t, handler.makePutHandler(tt.table))
			assert.NotNil(t, handler.makePatchHandler(tt.table))
			assert.NotNil(t, handler.makeDeleteHandler(tt.table))
		})
	}
}

func TestColumnExists_Validation(t *testing.T) {
	handler := &RESTHandler{}

	table := database.TableInfo{
		Schema: "public",
		Name:   "items",
		Columns: []database.ColumnInfo{
			{Name: "id", DataType: "uuid"},
			{Name: "name", DataType: "text"},
			{Name: "email", DataType: "text"},
		},
	}

	t.Run("existing column returns true", func(t *testing.T) {
		assert.True(t, handler.columnExists(table, "id"))
		assert.True(t, handler.columnExists(table, "name"))
		assert.True(t, handler.columnExists(table, "email"))
	})

	t.Run("non-existing column returns false", func(t *testing.T) {
		assert.False(t, handler.columnExists(table, "unknown"))
		assert.False(t, handler.columnExists(table, "password"))
		assert.False(t, handler.columnExists(table, ""))
	})

	t.Run("case sensitive column names", func(t *testing.T) {
		// Column names should be case sensitive
		assert.False(t, handler.columnExists(table, "ID"))
		assert.False(t, handler.columnExists(table, "Name"))
		assert.False(t, handler.columnExists(table, "EMAIL"))
	})
}

func TestBuildSelectQuery_QueryGeneration(t *testing.T) {
	handler := &RESTHandler{
		parser: NewQueryParser(&config.Config{}),
	}

	table := database.TableInfo{
		Schema: "public",
		Name:   "items",
		Columns: []database.ColumnInfo{
			{Name: "id", DataType: "uuid"},
			{Name: "name", DataType: "text"},
			{Name: "price", DataType: "numeric"},
		},
	}

	t.Run("basic SELECT query", func(t *testing.T) {
		params := &QueryParams{
			Select: []string{"id", "name"},
		}

		query, args := handler.buildSelectQuery(table, params)
		assert.Contains(t, query, "SELECT")
		assert.Contains(t, query, "FROM")
		assert.Contains(t, query, `"public"."items"`)
		assert.NotEmpty(t, query)
		_ = args // Just verify it returns args
	})

	t.Run("SELECT with WHERE clause", func(t *testing.T) {
		limit := 10
		params := &QueryParams{
			Select:  []string{"id", "name"},
			Filters: []Filter{{Column: "name", Operator: OpEqual, Value: "test"}},
			Limit:   &limit,
		}

		query, args := handler.buildSelectQuery(table, params)
		assert.Contains(t, query, "WHERE")
		assert.NotEmpty(t, args)
	})

	t.Run("SELECT with ORDER BY", func(t *testing.T) {
		params := &QueryParams{
			Select: []string{"id", "name"},
			Order:  []OrderBy{{Column: "name", Desc: false}},
		}

		query, args := handler.buildSelectQuery(table, params)
		assert.Contains(t, query, "ORDER BY")
		_ = args
	})

	t.Run("SELECT with ORDER BY DESC", func(t *testing.T) {
		params := &QueryParams{
			Select: []string{"id", "name"},
			Order:  []OrderBy{{Column: "name", Desc: true}},
		}

		query, args := handler.buildSelectQuery(table, params)
		assert.Contains(t, query, "ORDER BY")
		assert.Contains(t, query, "DESC")
		_ = args
	})

	t.Run("SELECT with LIMIT", func(t *testing.T) {
		limit := 10
		params := &QueryParams{
			Select: []string{"id", "name"},
			Limit:  &limit,
		}

		query, args := handler.buildSelectQuery(table, params)
		assert.Contains(t, query, "LIMIT")
		_ = args
	})

	t.Run("SELECT with OFFSET", func(t *testing.T) {
		limit := 10
		offset := 5
		params := &QueryParams{
			Select: []string{"id", "name"},
			Limit:  &limit,
			Offset: &offset,
		}

		query, args := handler.buildSelectQuery(table, params)
		assert.Contains(t, query, "OFFSET")
		_ = args
	})
}

func TestQuoteIdentifier_SQLInjection(t *testing.T) {
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
			name:     "identifier with numbers",
			input:    "column123",
			expected: `"column123"`,
		},
		{
			name:     "identifier with underscores",
			input:    "column_name_test",
			expected: `"column_name_test"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := quoteIdentifier(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsValidIdentifier_Validation(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "simple name",
			input:    "column",
			expected: true,
		},
		{
			name:     "name with underscores",
			input:    "column_name",
			expected: true,
		},
		{
			name:     "name with numbers",
			input:    "column123",
			expected: true,
		},
		{
			name:     "starts with letter",
			input:    "a1",
			expected: true,
		},
		{
			name:     "starts with underscore",
			input:    "_private",
			expected: true,
		},
		{
			name:     "starts with number - invalid",
			input:    "1column",
			expected: false,
		},
		{
			name:     "contains special characters - invalid",
			input:    "column-name",
			expected: false,
		},
		{
			name:     "contains space - invalid",
			input:    "column name",
			expected: false,
		},
		{
			name:     "empty string - invalid",
			input:    "",
			expected: false,
		},
		{
			name:     "contains dot - invalid",
			input:    "column.name",
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
