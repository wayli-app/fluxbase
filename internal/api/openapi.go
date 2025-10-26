package api

import (
	"fmt"
	"strings"

	"github.com/wayli-app/fluxbase/internal/database"
	"github.com/gofiber/fiber/v2"
)

// OpenAPISpec represents the OpenAPI 3.0 specification
type OpenAPISpec struct {
	OpenAPI    string                       `json:"openapi"`
	Info       OpenAPIInfo                  `json:"info"`
	Servers    []OpenAPIServer              `json:"servers"`
	Paths      map[string]OpenAPIPath       `json:"paths"`
	Components OpenAPIComponents            `json:"components"`
}

type OpenAPIInfo struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Version     string `json:"version"`
}

type OpenAPIServer struct {
	URL         string `json:"url"`
	Description string `json:"description"`
}

type OpenAPIPath map[string]OpenAPIOperation

type OpenAPIOperation struct {
	Summary     string                        `json:"summary,omitempty"`
	Description string                        `json:"description,omitempty"`
	OperationID string                        `json:"operationId,omitempty"`
	Tags        []string                      `json:"tags,omitempty"`
	Parameters  []OpenAPIParameter            `json:"parameters,omitempty"`
	RequestBody *OpenAPIRequestBody           `json:"requestBody,omitempty"`
	Responses   map[string]OpenAPIResponse    `json:"responses"`
	Security    []map[string][]string         `json:"security,omitempty"`
}

type OpenAPIParameter struct {
	Name        string      `json:"name"`
	In          string      `json:"in"`
	Description string      `json:"description,omitempty"`
	Required    bool        `json:"required,omitempty"`
	Schema      interface{} `json:"schema"`
}

type OpenAPIRequestBody struct {
	Description string                `json:"description,omitempty"`
	Required    bool                  `json:"required,omitempty"`
	Content     map[string]OpenAPIMedia `json:"content"`
}

type OpenAPIMedia struct {
	Schema interface{} `json:"schema"`
}

type OpenAPIResponse struct {
	Description string                `json:"description"`
	Content     map[string]OpenAPIMedia `json:"content,omitempty"`
}

type OpenAPIComponents struct {
	Schemas         map[string]interface{} `json:"schemas,omitempty"`
	SecuritySchemes map[string]interface{} `json:"securitySchemes,omitempty"`
}

// OpenAPIHandler handles OpenAPI spec generation
type OpenAPIHandler struct {
	db *database.Connection
}

// NewOpenAPIHandler creates a new OpenAPI handler
func NewOpenAPIHandler(db *database.Connection) *OpenAPIHandler {
	return &OpenAPIHandler{db: db}
}

// RegisterRoutes registers OpenAPI routes
func (h *OpenAPIHandler) RegisterRoutes(app *fiber.App) {
	app.Get("/openapi.json", h.GetOpenAPISpec)
}

// GetOpenAPISpec generates and returns the OpenAPI specification
func (h *OpenAPIHandler) GetOpenAPISpec(c *fiber.Ctx) error {
	inspector := database.NewSchemaInspector(h.db)

	// Get all tables
	tables, err := inspector.GetAllTables(c.Context(), "public", "auth")
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to fetch database schema",
		})
	}

	// Get all functions for RPC endpoints
	functions, err := inspector.GetAllFunctions(c.Context(), "public", "auth")
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to fetch database functions",
		})
	}

	spec := h.generateSpec(tables, functions, c.BaseURL())
	return c.JSON(spec)
}

// generateSpec generates the complete OpenAPI spec
func (h *OpenAPIHandler) generateSpec(tables []database.TableInfo, functions []database.FunctionInfo, baseURL string) OpenAPISpec {
	spec := OpenAPISpec{
		OpenAPI: "3.0.0",
		Info: OpenAPIInfo{
			Title:       "Fluxbase REST API",
			Description: "Complete Fluxbase API including authentication, database tables, RPC functions, and admin endpoints",
			Version:     "1.0.0",
		},
		Servers: []OpenAPIServer{
			{
				URL:         baseURL,
				Description: "Current server",
			},
		},
		Paths: make(map[string]OpenAPIPath),
		Components: OpenAPIComponents{
			Schemas: make(map[string]interface{}),
			SecuritySchemes: map[string]interface{}{
				"bearerAuth": map[string]interface{}{
					"type":         "http",
					"scheme":       "bearer",
					"bearerFormat": "JWT",
					"description":  "JWT token obtained from /api/auth/signin or /api/auth/signup",
				},
			},
		},
	}

	// Add authentication endpoints
	h.addAuthEndpoints(&spec)

	// Generate paths and schemas for each table
	for _, table := range tables {
		h.addTableToSpec(&spec, table)
	}

	// Generate RPC endpoints for each function
	for _, function := range functions {
		h.addFunctionToSpec(&spec, function)
	}

	return spec
}

// addAuthEndpoints adds authentication endpoints to the spec
func (h *OpenAPIHandler) addAuthEndpoints(spec *OpenAPISpec) {
	// User schema
	spec.Components.Schemas["User"] = map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"id":         map[string]string{"type": "string", "format": "uuid"},
			"email":      map[string]string{"type": "string", "format": "email"},
			"created_at": map[string]string{"type": "string", "format": "date-time"},
			"updated_at": map[string]string{"type": "string", "format": "date-time"},
		},
	}

	// Token response schema
	spec.Components.Schemas["TokenResponse"] = map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"access_token":  map[string]string{"type": "string"},
			"refresh_token": map[string]string{"type": "string"},
			"token_type":    map[string]string{"type": "string"},
			"expires_in":    map[string]string{"type": "integer"},
			"user":          map[string]string{"$ref": "#/components/schemas/User"},
		},
	}

	// Signup request schema
	spec.Components.Schemas["SignupRequest"] = map[string]interface{}{
		"type": "object",
		"required": []string{"email", "password"},
		"properties": map[string]interface{}{
			"email":    map[string]string{"type": "string", "format": "email"},
			"password": map[string]string{"type": "string", "minLength": "8"},
		},
	}

	// Signin request schema
	spec.Components.Schemas["SigninRequest"] = map[string]interface{}{
		"type": "object",
		"required": []string{"email", "password"},
		"properties": map[string]interface{}{
			"email":    map[string]string{"type": "string", "format": "email"},
			"password": map[string]string{"type": "string"},
		},
	}

	// Refresh request schema
	spec.Components.Schemas["RefreshRequest"] = map[string]interface{}{
		"type": "object",
		"required": []string{"refresh_token"},
		"properties": map[string]interface{}{
			"refresh_token": map[string]string{"type": "string"},
		},
	}

	// Magic link request schema
	spec.Components.Schemas["MagicLinkRequest"] = map[string]interface{}{
		"type": "object",
		"required": []string{"email"},
		"properties": map[string]interface{}{
			"email": map[string]string{"type": "string", "format": "email"},
		},
	}

	// Error response schema
	spec.Components.Schemas["Error"] = map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"error": map[string]string{"type": "string"},
		},
	}

	// POST /api/auth/signup
	spec.Paths["/api/auth/signup"] = OpenAPIPath{
		"post": OpenAPIOperation{
			Summary:     "Sign up a new user",
			Description: "Create a new user account with email and password",
			OperationID: "auth_signup",
			Tags:        []string{"Authentication"},
			RequestBody: &OpenAPIRequestBody{
				Required: true,
				Content: map[string]OpenAPIMedia{
					"application/json": {
						Schema: map[string]string{"$ref": "#/components/schemas/SignupRequest"},
					},
				},
			},
			Responses: map[string]OpenAPIResponse{
				"200": {
					Description: "User created successfully",
					Content: map[string]OpenAPIMedia{
						"application/json": {
							Schema: map[string]string{"$ref": "#/components/schemas/TokenResponse"},
						},
					},
				},
				"400": {
					Description: "Invalid request",
					Content: map[string]OpenAPIMedia{
						"application/json": {
							Schema: map[string]string{"$ref": "#/components/schemas/Error"},
						},
					},
				},
			},
		},
	}

	// POST /api/auth/signin
	spec.Paths["/api/auth/signin"] = OpenAPIPath{
		"post": OpenAPIOperation{
			Summary:     "Sign in a user",
			Description: "Authenticate with email and password to get access tokens",
			OperationID: "auth_signin",
			Tags:        []string{"Authentication"},
			RequestBody: &OpenAPIRequestBody{
				Required: true,
				Content: map[string]OpenAPIMedia{
					"application/json": {
						Schema: map[string]string{"$ref": "#/components/schemas/SigninRequest"},
					},
				},
			},
			Responses: map[string]OpenAPIResponse{
				"200": {
					Description: "Successfully authenticated",
					Content: map[string]OpenAPIMedia{
						"application/json": {
							Schema: map[string]string{"$ref": "#/components/schemas/TokenResponse"},
						},
					},
				},
				"401": {
					Description: "Invalid credentials",
					Content: map[string]OpenAPIMedia{
						"application/json": {
							Schema: map[string]string{"$ref": "#/components/schemas/Error"},
						},
					},
				},
			},
		},
	}

	// POST /api/auth/signout
	spec.Paths["/api/auth/signout"] = OpenAPIPath{
		"post": OpenAPIOperation{
			Summary:     "Sign out a user",
			Description: "Invalidate the current session",
			OperationID: "auth_signout",
			Tags:        []string{"Authentication"},
			Security: []map[string][]string{
				{"bearerAuth": {}},
			},
			Responses: map[string]OpenAPIResponse{
				"200": {
					Description: "Successfully signed out",
				},
				"401": {
					Description: "Unauthorized",
					Content: map[string]OpenAPIMedia{
						"application/json": {
							Schema: map[string]string{"$ref": "#/components/schemas/Error"},
						},
					},
				},
			},
		},
	}

	// POST /api/auth/refresh
	spec.Paths["/api/auth/refresh"] = OpenAPIPath{
		"post": OpenAPIOperation{
			Summary:     "Refresh access token",
			Description: "Get a new access token using a refresh token",
			OperationID: "auth_refresh",
			Tags:        []string{"Authentication"},
			RequestBody: &OpenAPIRequestBody{
				Required: true,
				Content: map[string]OpenAPIMedia{
					"application/json": {
						Schema: map[string]string{"$ref": "#/components/schemas/RefreshRequest"},
					},
				},
			},
			Responses: map[string]OpenAPIResponse{
				"200": {
					Description: "New access token issued",
					Content: map[string]OpenAPIMedia{
						"application/json": {
							Schema: map[string]string{"$ref": "#/components/schemas/TokenResponse"},
						},
					},
				},
				"401": {
					Description: "Invalid refresh token",
					Content: map[string]OpenAPIMedia{
						"application/json": {
							Schema: map[string]string{"$ref": "#/components/schemas/Error"},
						},
					},
				},
			},
		},
	}

	// GET /api/auth/user
	spec.Paths["/api/auth/user"] = OpenAPIPath{
		"get": OpenAPIOperation{
			Summary:     "Get current user",
			Description: "Get the authenticated user's information",
			OperationID: "auth_get_user",
			Tags:        []string{"Authentication"},
			Security: []map[string][]string{
				{"bearerAuth": {}},
			},
			Responses: map[string]OpenAPIResponse{
				"200": {
					Description: "User information",
					Content: map[string]OpenAPIMedia{
						"application/json": {
							Schema: map[string]string{"$ref": "#/components/schemas/User"},
						},
					},
				},
				"401": {
					Description: "Unauthorized",
					Content: map[string]OpenAPIMedia{
						"application/json": {
							Schema: map[string]string{"$ref": "#/components/schemas/Error"},
						},
					},
				},
			},
		},
		"patch": OpenAPIOperation{
			Summary:     "Update current user",
			Description: "Update the authenticated user's information",
			OperationID: "auth_update_user",
			Tags:        []string{"Authentication"},
			Security: []map[string][]string{
				{"bearerAuth": {}},
			},
			RequestBody: &OpenAPIRequestBody{
				Required: true,
				Content: map[string]OpenAPIMedia{
					"application/json": {
						Schema: map[string]string{"$ref": "#/components/schemas/User"},
					},
				},
			},
			Responses: map[string]OpenAPIResponse{
				"200": {
					Description: "User updated successfully",
					Content: map[string]OpenAPIMedia{
						"application/json": {
							Schema: map[string]string{"$ref": "#/components/schemas/User"},
						},
					},
				},
				"401": {
					Description: "Unauthorized",
					Content: map[string]OpenAPIMedia{
						"application/json": {
							Schema: map[string]string{"$ref": "#/components/schemas/Error"},
						},
					},
				},
			},
		},
	}

	// POST /api/auth/magiclink
	spec.Paths["/api/auth/magiclink"] = OpenAPIPath{
		"post": OpenAPIOperation{
			Summary:     "Request magic link",
			Description: "Send a magic link to the user's email for passwordless authentication",
			OperationID: "auth_magiclink",
			Tags:        []string{"Authentication"},
			RequestBody: &OpenAPIRequestBody{
				Required: true,
				Content: map[string]OpenAPIMedia{
					"application/json": {
						Schema: map[string]string{"$ref": "#/components/schemas/MagicLinkRequest"},
					},
				},
			},
			Responses: map[string]OpenAPIResponse{
				"200": {
					Description: "Magic link sent successfully",
				},
				"400": {
					Description: "Invalid request",
					Content: map[string]OpenAPIMedia{
						"application/json": {
							Schema: map[string]string{"$ref": "#/components/schemas/Error"},
						},
					},
				},
			},
		},
	}

	// GET /api/auth/magiclink/verify
	spec.Paths["/api/auth/magiclink/verify"] = OpenAPIPath{
		"get": OpenAPIOperation{
			Summary:     "Verify magic link",
			Description: "Verify a magic link token and authenticate the user",
			OperationID: "auth_magiclink_verify",
			Tags:        []string{"Authentication"},
			Parameters: []OpenAPIParameter{
				{
					Name:        "token",
					In:          "query",
					Description: "Magic link token",
					Required:    true,
					Schema:      map[string]string{"type": "string"},
				},
			},
			Responses: map[string]OpenAPIResponse{
				"200": {
					Description: "Successfully authenticated",
					Content: map[string]OpenAPIMedia{
						"application/json": {
							Schema: map[string]string{"$ref": "#/components/schemas/TokenResponse"},
						},
					},
				},
				"401": {
					Description: "Invalid or expired token",
					Content: map[string]OpenAPIMedia{
						"application/json": {
							Schema: map[string]string{"$ref": "#/components/schemas/Error"},
						},
					},
				},
			},
		},
	}
}

// addTableToSpec adds paths and schema for a table
func (h *OpenAPIHandler) addTableToSpec(spec *OpenAPISpec, table database.TableInfo) {
	tableName := table.Name
	schemaName := table.Schema

	// Build the path (same logic as REST handler)
	path := h.buildTablePath(table)
	pathWithID := path + "/{id}"

	// Generate schema
	schemaRef := h.generateTableSchema(spec, table)

	// Add paths
	spec.Paths[path] = OpenAPIPath{
		"get": h.generateListOperation(tableName, schemaName, schemaRef),
		"post": h.generateCreateOperation(tableName, schemaName, schemaRef),
		"patch": h.generateBatchUpdateOperation(tableName, schemaName, schemaRef),
		"delete": h.generateBatchDeleteOperation(tableName, schemaName, schemaRef),
	}

	spec.Paths[pathWithID] = OpenAPIPath{
		"get": h.generateGetOperation(tableName, schemaName, schemaRef),
		"put": h.generateReplaceOperation(tableName, schemaName, schemaRef),
		"patch": h.generateUpdateOperation(tableName, schemaName, schemaRef),
		"delete": h.generateDeleteOperation(tableName, schemaName, schemaRef),
	}
}

// buildTablePath builds the REST API path for a table
func (h *OpenAPIHandler) buildTablePath(table database.TableInfo) string {
	tableName := table.Name
	if !strings.HasSuffix(tableName, "s") {
		if strings.HasSuffix(tableName, "y") {
			tableName = strings.TrimSuffix(tableName, "y") + "ies"
		} else if strings.HasSuffix(tableName, "x") ||
			strings.HasSuffix(tableName, "ch") ||
			strings.HasSuffix(tableName, "sh") {
			tableName += "es"
		} else {
			tableName += "s"
		}
	}

	// All database tables/views are under /api/tables/ prefix
	if table.Schema != "public" {
		return "/api/tables/" + table.Schema + "/" + tableName
	}
	return "/api/tables/" + tableName
}

// generateTableSchema generates JSON schema for a table
func (h *OpenAPIHandler) generateTableSchema(spec *OpenAPISpec, table database.TableInfo) string {
	schemaName := table.Schema + "." + table.Name

	properties := make(map[string]interface{})
	required := []string{}

	for _, col := range table.Columns {
		properties[col.Name] = h.columnToSchema(col)

		if !col.IsNullable && col.DefaultValue == nil {
			required = append(required, col.Name)
		}
	}

	schema := map[string]interface{}{
		"type":       "object",
		"properties": properties,
	}

	if len(required) > 0 {
		schema["required"] = required
	}

	spec.Components.Schemas[schemaName] = schema
	return "#/components/schemas/" + schemaName
}

// columnToSchema converts a column to JSON schema
func (h *OpenAPIHandler) columnToSchema(col database.ColumnInfo) map[string]interface{} {
	schema := make(map[string]interface{})

	// Map PostgreSQL types to JSON Schema types
	switch {
	case strings.Contains(col.DataType, "int"):
		schema["type"] = "integer"
	case strings.Contains(col.DataType, "numeric") || strings.Contains(col.DataType, "decimal") || strings.Contains(col.DataType, "float") || strings.Contains(col.DataType, "double"):
		schema["type"] = "number"
	case strings.Contains(col.DataType, "bool"):
		schema["type"] = "boolean"
	case strings.Contains(col.DataType, "json"):
		schema["type"] = "object"
	case strings.Contains(col.DataType, "array") || strings.HasPrefix(col.DataType, "_"):
		schema["type"] = "array"
		schema["items"] = map[string]string{"type": "string"}
	case strings.Contains(col.DataType, "timestamp") || strings.Contains(col.DataType, "date"):
		schema["type"] = "string"
		schema["format"] = "date-time"
	case strings.Contains(col.DataType, "uuid"):
		schema["type"] = "string"
		schema["format"] = "uuid"
	default:
		schema["type"] = "string"
	}

	if col.IsNullable {
		schema["nullable"] = true
	}

	return schema
}

// generateListOperation generates GET operation for listing records
func (h *OpenAPIHandler) generateListOperation(tableName, schemaName, schemaRef string) OpenAPIOperation {
	return OpenAPIOperation{
		Summary:     fmt.Sprintf("List %s records", tableName),
		Description: "Query and filter records with PostgREST-compatible syntax",
		OperationID: fmt.Sprintf("list_%s_%s", schemaName, tableName),
		Tags:        []string{schemaName + "." + tableName},
		Parameters:  h.getQueryParameters(),
		Responses: map[string]OpenAPIResponse{
			"200": {
				Description: "Successful response",
				Content: map[string]OpenAPIMedia{
					"application/json": {
						Schema: map[string]interface{}{
							"type": "array",
							"items": map[string]string{
								"$ref": schemaRef,
							},
						},
					},
				},
			},
		},
	}
}

// generateCreateOperation generates POST operation for creating records
func (h *OpenAPIHandler) generateCreateOperation(tableName, schemaName, schemaRef string) OpenAPIOperation {
	return OpenAPIOperation{
		Summary:     fmt.Sprintf("Create %s record(s)", tableName),
		Description: "Create a single record or batch insert multiple records",
		OperationID: fmt.Sprintf("create_%s_%s", schemaName, tableName),
		Tags:        []string{schemaName + "." + tableName},
		RequestBody: &OpenAPIRequestBody{
			Required: true,
			Content: map[string]OpenAPIMedia{
				"application/json": {
					Schema: map[string]interface{}{
						"oneOf": []interface{}{
							map[string]string{"$ref": schemaRef},
							map[string]interface{}{
								"type": "array",
								"items": map[string]string{
									"$ref": schemaRef,
								},
							},
						},
					},
				},
			},
		},
		Responses: map[string]OpenAPIResponse{
			"201": {
				Description: "Created successfully",
				Content: map[string]OpenAPIMedia{
					"application/json": {
						Schema: map[string]interface{}{
							"oneOf": []interface{}{
								map[string]string{"$ref": schemaRef},
								map[string]interface{}{
									"type": "array",
									"items": map[string]string{
										"$ref": schemaRef,
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

// generateGetOperation generates GET by ID operation
func (h *OpenAPIHandler) generateGetOperation(tableName, schemaName, schemaRef string) OpenAPIOperation {
	return OpenAPIOperation{
		Summary:     fmt.Sprintf("Get %s by ID", tableName),
		OperationID: fmt.Sprintf("get_%s_%s", schemaName, tableName),
		Tags:        []string{schemaName + "." + tableName},
		Parameters: []OpenAPIParameter{
			{
				Name:        "id",
				In:          "path",
				Description: "Record ID",
				Required:    true,
				Schema:      map[string]string{"type": "string"},
			},
		},
		Responses: map[string]OpenAPIResponse{
			"200": {
				Description: "Successful response",
				Content: map[string]OpenAPIMedia{
					"application/json": {
						Schema: map[string]string{"$ref": schemaRef},
					},
				},
			},
			"404": {
				Description: "Record not found",
			},
		},
	}
}

// generateUpdateOperation generates PATCH operation
func (h *OpenAPIHandler) generateUpdateOperation(tableName, schemaName, schemaRef string) OpenAPIOperation {
	return OpenAPIOperation{
		Summary:     fmt.Sprintf("Update %s", tableName),
		OperationID: fmt.Sprintf("update_%s_%s", schemaName, tableName),
		Tags:        []string{schemaName + "." + tableName},
		Parameters: []OpenAPIParameter{
			{
				Name:        "id",
				In:          "path",
				Description: "Record ID",
				Required:    true,
				Schema:      map[string]string{"type": "string"},
			},
		},
		RequestBody: &OpenAPIRequestBody{
			Required: true,
			Content: map[string]OpenAPIMedia{
				"application/json": {
					Schema: map[string]string{"$ref": schemaRef},
				},
			},
		},
		Responses: map[string]OpenAPIResponse{
			"200": {
				Description: "Updated successfully",
				Content: map[string]OpenAPIMedia{
					"application/json": {
						Schema: map[string]string{"$ref": schemaRef},
					},
				},
			},
		},
	}
}

// generateReplaceOperation generates PUT operation
func (h *OpenAPIHandler) generateReplaceOperation(tableName, schemaName, schemaRef string) OpenAPIOperation {
	return OpenAPIOperation{
		Summary:     fmt.Sprintf("Replace %s", tableName),
		OperationID: fmt.Sprintf("replace_%s_%s", schemaName, tableName),
		Tags:        []string{schemaName + "." + tableName},
		Parameters: []OpenAPIParameter{
			{
				Name:        "id",
				In:          "path",
				Description: "Record ID",
				Required:    true,
				Schema:      map[string]string{"type": "string"},
			},
		},
		RequestBody: &OpenAPIRequestBody{
			Required: true,
			Content: map[string]OpenAPIMedia{
				"application/json": {
					Schema: map[string]string{"$ref": schemaRef},
				},
			},
		},
		Responses: map[string]OpenAPIResponse{
			"200": {
				Description: "Replaced successfully",
				Content: map[string]OpenAPIMedia{
					"application/json": {
						Schema: map[string]string{"$ref": schemaRef},
					},
				},
			},
		},
	}
}

// generateDeleteOperation generates DELETE operation
func (h *OpenAPIHandler) generateDeleteOperation(tableName, schemaName, schemaRef string) OpenAPIOperation {
	return OpenAPIOperation{
		Summary:     fmt.Sprintf("Delete %s", tableName),
		OperationID: fmt.Sprintf("delete_%s_%s", schemaName, tableName),
		Tags:        []string{schemaName + "." + tableName},
		Parameters: []OpenAPIParameter{
			{
				Name:        "id",
				In:          "path",
				Description: "Record ID",
				Required:    true,
				Schema:      map[string]string{"type": "string"},
			},
		},
		Responses: map[string]OpenAPIResponse{
			"204": {
				Description: "Deleted successfully",
			},
			"404": {
				Description: "Record not found",
			},
		},
	}
}

// generateBatchUpdateOperation generates batch PATCH operation
func (h *OpenAPIHandler) generateBatchUpdateOperation(tableName, schemaName, schemaRef string) OpenAPIOperation {
	return OpenAPIOperation{
		Summary:     fmt.Sprintf("Batch update %s records", tableName),
		Description: "Update multiple records matching the filter criteria",
		OperationID: fmt.Sprintf("batch_update_%s_%s", schemaName, tableName),
		Tags:        []string{schemaName + "." + tableName},
		Parameters:  h.getQueryParameters(),
		RequestBody: &OpenAPIRequestBody{
			Required: true,
			Content: map[string]OpenAPIMedia{
				"application/json": {
					Schema: map[string]string{"$ref": schemaRef},
				},
			},
		},
		Responses: map[string]OpenAPIResponse{
			"200": {
				Description: "Updated successfully",
				Content: map[string]OpenAPIMedia{
					"application/json": {
						Schema: map[string]interface{}{
							"type": "array",
							"items": map[string]string{
								"$ref": schemaRef,
							},
						},
					},
				},
			},
		},
	}
}

// generateBatchDeleteOperation generates batch DELETE operation
func (h *OpenAPIHandler) generateBatchDeleteOperation(tableName, schemaName, schemaRef string) OpenAPIOperation {
	return OpenAPIOperation{
		Summary:     fmt.Sprintf("Batch delete %s records", tableName),
		Description: "Delete multiple records matching the filter criteria (requires at least one filter)",
		OperationID: fmt.Sprintf("batch_delete_%s_%s", schemaName, tableName),
		Tags:        []string{schemaName + "." + tableName},
		Parameters:  h.getQueryParameters(),
		Responses: map[string]OpenAPIResponse{
			"200": {
				Description: "Deleted successfully",
				Content: map[string]OpenAPIMedia{
					"application/json": {
						Schema: map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"deleted": map[string]string{"type": "integer"},
								"records": map[string]interface{}{
									"type": "array",
									"items": map[string]string{
										"$ref": schemaRef,
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

// getQueryParameters returns common query parameters
func (h *OpenAPIHandler) getQueryParameters() []OpenAPIParameter {
	return []OpenAPIParameter{
		{
			Name:        "select",
			In:          "query",
			Description: "Columns to select (comma-separated)",
			Schema:      map[string]string{"type": "string"},
		},
		{
			Name:        "order",
			In:          "query",
			Description: "Order by column (e.g., name.asc, created_at.desc)",
			Schema:      map[string]string{"type": "string"},
		},
		{
			Name:        "limit",
			In:          "query",
			Description: "Limit number of results",
			Schema:      map[string]string{"type": "integer"},
		},
		{
			Name:        "offset",
			In:          "query",
			Description: "Offset for pagination",
			Schema:      map[string]string{"type": "integer"},
		},
		{
			Name:        "filter",
			In:          "query",
			Description: "Filter using column.operator=value (e.g., name.eq=John, age.gt=18)",
			Schema:      map[string]string{"type": "string"},
		},
	}
}

// addFunctionToSpec adds a PostgreSQL function as an RPC endpoint to the spec
func (h *OpenAPIHandler) addFunctionToSpec(spec *OpenAPISpec, fn database.FunctionInfo) {
	// Build path for RPC endpoint
	path := h.buildFunctionPath(fn)

	// Build request body schema from parameters
	requestSchema := h.buildFunctionRequestSchema(fn)

	// Build response schema from return type
	responseSchema := h.buildFunctionResponseSchema(fn)

	// Create operation description
	description := fn.Description
	if description == "" {
		description = fmt.Sprintf("Call the %s.%s PostgreSQL function", fn.Schema, fn.Name)
	}

	// Add parameter details to description
	if len(fn.Parameters) > 0 {
		description += "\n\n**Parameters:**\n"
		for _, param := range fn.Parameters {
			paramDesc := fmt.Sprintf("- `%s` (%s)", param.Name, param.Type)
			if param.HasDefault {
				paramDesc += " - optional"
			} else {
				paramDesc += " - required"
			}
			description += paramDesc + "\n"
		}
	}

	// Add function metadata to description
	description += fmt.Sprintf("\n**Return Type:** %s", fn.ReturnType)
	description += fmt.Sprintf("\n**Volatility:** %s", fn.Volatility)

	// Create the RPC endpoint
	spec.Paths[path] = OpenAPIPath{
		"post": OpenAPIOperation{
			Summary:     fmt.Sprintf("RPC: %s", fn.Name),
			Description: description,
			OperationID: fmt.Sprintf("rpc_%s_%s", fn.Schema, fn.Name),
			Tags:        []string{"RPC Functions"},
			RequestBody: &OpenAPIRequestBody{
				Description: "Function parameters as JSON object",
				Required:    len(fn.Parameters) > 0,
				Content: map[string]OpenAPIMedia{
					"application/json": {
						Schema: requestSchema,
					},
				},
			},
			Responses: map[string]OpenAPIResponse{
				"200": {
					Description: "Successful function execution",
					Content: map[string]OpenAPIMedia{
						"application/json": {
							Schema: responseSchema,
						},
					},
				},
				"400": {
					Description: "Invalid parameters or function call failed",
					Content: map[string]OpenAPIMedia{
						"application/json": {
							Schema: map[string]interface{}{
								"type": "object",
								"properties": map[string]interface{}{
									"error": map[string]string{"type": "string"},
								},
							},
						},
					},
				},
				"500": {
					Description: "Internal server error",
					Content: map[string]OpenAPIMedia{
						"application/json": {
							Schema: map[string]interface{}{
								"type": "object",
								"properties": map[string]interface{}{
									"error": map[string]string{"type": "string"},
								},
							},
						},
					},
				},
			},
		},
	}
}

// buildFunctionPath builds the API path for a function
func (h *OpenAPIHandler) buildFunctionPath(fn database.FunctionInfo) string {
	if fn.Schema != "public" {
		return fmt.Sprintf("/api/rpc/%s/%s", fn.Schema, fn.Name)
	}
	return fmt.Sprintf("/api/rpc/%s", fn.Name)
}

// buildFunctionRequestSchema builds the request schema for a function
func (h *OpenAPIHandler) buildFunctionRequestSchema(fn database.FunctionInfo) interface{} {
	if len(fn.Parameters) == 0 {
		return map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{},
		}
	}

	properties := make(map[string]interface{})
	required := []string{}

	for _, param := range fn.Parameters {
		if param.Mode == "OUT" {
			continue // Skip OUT parameters
		}

		properties[param.Name] = map[string]interface{}{
			"type":        h.mapPostgreSQLTypeToJSON(param.Type),
			"description": fmt.Sprintf("Parameter of type %s", param.Type),
		}

		if !param.HasDefault {
			required = append(required, param.Name)
		}
	}

	schema := map[string]interface{}{
		"type":       "object",
		"properties": properties,
	}

	if len(required) > 0 {
		schema["required"] = required
	}

	return schema
}

// buildFunctionResponseSchema builds the response schema for a function
func (h *OpenAPIHandler) buildFunctionResponseSchema(fn database.FunctionInfo) interface{} {
	if fn.IsSetOf {
		// Function returns a set of rows (array)
		return map[string]interface{}{
			"type": "array",
			"items": map[string]interface{}{
				"type":        "object",
				"description": fmt.Sprintf("Row from %s", fn.ReturnType),
			},
		}
	} else {
		// Function returns a single value
		return map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"result": map[string]interface{}{
					"description": fmt.Sprintf("Result of type %s", fn.ReturnType),
				},
			},
		}
	}
}

// mapPostgreSQLTypeToJSON maps PostgreSQL types to JSON Schema types
func (h *OpenAPIHandler) mapPostgreSQLTypeToJSON(pgType string) string {
	pgTypeLower := strings.ToLower(pgType)

	switch {
	case strings.Contains(pgTypeLower, "int"):
		return "integer"
	case strings.Contains(pgTypeLower, "numeric") || strings.Contains(pgTypeLower, "decimal") ||
		 strings.Contains(pgTypeLower, "float") || strings.Contains(pgTypeLower, "double") ||
		 strings.Contains(pgTypeLower, "real"):
		return "number"
	case strings.Contains(pgTypeLower, "bool"):
		return "boolean"
	case strings.Contains(pgTypeLower, "json"):
		return "object"
	case strings.Contains(pgTypeLower, "array") || strings.HasPrefix(pgTypeLower, "_"):
		return "array"
	default:
		return "string"
	}
}
