package api

import (
	"fmt"
	"strings"

	"github.com/fluxbase-eu/fluxbase/internal/database"
	"github.com/gofiber/fiber/v2"
)

// OpenAPISpec represents the OpenAPI 3.0 specification
type OpenAPISpec struct {
	OpenAPI    string                 `json:"openapi"`
	Info       OpenAPIInfo            `json:"info"`
	Servers    []OpenAPIServer        `json:"servers"`
	Paths      map[string]OpenAPIPath `json:"paths"`
	Components OpenAPIComponents      `json:"components"`
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
	Summary     string                     `json:"summary,omitempty"`
	Description string                     `json:"description,omitempty"`
	OperationID string                     `json:"operationId,omitempty"`
	Tags        []string                   `json:"tags,omitempty"`
	Parameters  []OpenAPIParameter         `json:"parameters,omitempty"`
	RequestBody *OpenAPIRequestBody        `json:"requestBody,omitempty"`
	Responses   map[string]OpenAPIResponse `json:"responses"`
	Security    []map[string][]string      `json:"security,omitempty"`
}

type OpenAPIParameter struct {
	Name        string      `json:"name"`
	In          string      `json:"in"`
	Description string      `json:"description,omitempty"`
	Required    bool        `json:"required,omitempty"`
	Schema      interface{} `json:"schema"`
}

type OpenAPIRequestBody struct {
	Description string                  `json:"description,omitempty"`
	Required    bool                    `json:"required,omitempty"`
	Content     map[string]OpenAPIMedia `json:"content"`
}

type OpenAPIMedia struct {
	Schema interface{} `json:"schema"`
}

type OpenAPIResponse struct {
	Description string                  `json:"description"`
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

// GetOpenAPISpec generates and returns the OpenAPI specification
// Admin users get full spec with database schema; non-admin users get minimal spec
func (h *OpenAPIHandler) GetOpenAPISpec(c *fiber.Ctx) error {
	// Check if user has admin role
	role, _ := c.Locals("user_role").(string)
	isAdmin := role == "admin" || role == "dashboard_admin" || role == "service_role"

	// Non-admin users get minimal spec without database tables
	if !isAdmin {
		spec := h.generateMinimalSpec(c.BaseURL())
		return c.JSON(spec)
	}

	inspector := database.NewSchemaInspector(h.db)

	// Get all tables (admin only)
	tables, err := inspector.GetAllTables(c.Context(), "public", "auth")
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to fetch database schema",
		})
	}

	spec := h.generateSpec(tables, c.BaseURL())
	return c.JSON(spec)
}

// generateMinimalSpec generates a minimal OpenAPI spec without database schema details
func (h *OpenAPIHandler) generateMinimalSpec(baseURL string) OpenAPISpec {
	spec := OpenAPISpec{
		OpenAPI: "3.0.0",
		Info: OpenAPIInfo{
			Title:       "Fluxbase REST API",
			Description: "Fluxbase API - authenticate with admin credentials for full specification",
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
					"description":  "JWT token obtained from /api/v1/auth/signin or /api/v1/auth/signup",
				},
			},
		},
	}

	// Only add auth endpoints for unauthenticated users
	h.addAuthEndpoints(&spec)

	return spec
}

// generateSpec generates the complete OpenAPI spec
func (h *OpenAPIHandler) generateSpec(tables []database.TableInfo, baseURL string) OpenAPISpec {
	spec := OpenAPISpec{
		OpenAPI: "3.0.0",
		Info: OpenAPIInfo{
			Title:       "Fluxbase REST API",
			Description: "Complete Fluxbase API including authentication, database tables, and admin endpoints",
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
					"description":  "JWT token obtained from /api/v1/auth/signin or /api/v1/auth/signup",
				},
			},
		},
	}

	// Add authentication endpoints
	h.addAuthEndpoints(&spec)

	// Add storage endpoints
	h.addStorageEndpoints(&spec)

	// Add API key endpoints
	h.addAPIKeyEndpoints(&spec)

	// Add webhook endpoints
	h.addWebhookEndpoints(&spec)

	// Add monitoring endpoints
	h.addMonitoringEndpoints(&spec)

	// Add vector/embedding endpoints
	h.addVectorEndpoints(&spec)

	// Add realtime endpoints
	h.addRealtimeEndpoints(&spec)

	// Add edge functions endpoints
	h.addFunctionsEndpoints(&spec)

	// Add jobs endpoints
	h.addJobsEndpoints(&spec)

	// Add RPC endpoints
	h.addRPCEndpoints(&spec)

	// Add AI endpoints
	h.addAIEndpoints(&spec)

	// Add admin endpoints
	h.addAdminEndpoints(&spec)

	// Generate paths and schemas for each table
	for _, table := range tables {
		h.addTableToSpec(&spec, table)
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
		"type":     "object",
		"required": []string{"email", "password"},
		"properties": map[string]interface{}{
			"email":    map[string]string{"type": "string", "format": "email"},
			"password": map[string]string{"type": "string", "minLength": "8"},
		},
	}

	// Signin request schema
	spec.Components.Schemas["SigninRequest"] = map[string]interface{}{
		"type":     "object",
		"required": []string{"email", "password"},
		"properties": map[string]interface{}{
			"email":    map[string]string{"type": "string", "format": "email"},
			"password": map[string]string{"type": "string"},
		},
	}

	// Refresh request schema
	spec.Components.Schemas["RefreshRequest"] = map[string]interface{}{
		"type":     "object",
		"required": []string{"refresh_token"},
		"properties": map[string]interface{}{
			"refresh_token": map[string]string{"type": "string"},
		},
	}

	// Magic link request schema
	spec.Components.Schemas["MagicLinkRequest"] = map[string]interface{}{
		"type":     "object",
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

	// POST /api/v1/auth/signup
	spec.Paths["/api/v1/auth/signup"] = OpenAPIPath{
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

	// POST /api/v1/auth/signin
	spec.Paths["/api/v1/auth/signin"] = OpenAPIPath{
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

	// POST /api/v1/auth/signout
	spec.Paths["/api/v1/auth/signout"] = OpenAPIPath{
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

	// POST /api/v1/auth/refresh
	spec.Paths["/api/v1/auth/refresh"] = OpenAPIPath{
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

	// GET /api/v1/auth/user
	spec.Paths["/api/v1/auth/user"] = OpenAPIPath{
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

	// POST /api/v1/auth/magiclink
	spec.Paths["/api/v1/auth/magiclink"] = OpenAPIPath{
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

	// GET /api/v1/auth/magiclink/verify
	spec.Paths["/api/v1/auth/magiclink/verify"] = OpenAPIPath{
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

// addStorageEndpoints adds storage API endpoints to the spec
func (h *OpenAPIHandler) addStorageEndpoints(spec *OpenAPISpec) {
	// Storage object schema
	spec.Components.Schemas["StorageObject"] = map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"key":           map[string]string{"type": "string"},
			"size":          map[string]string{"type": "integer"},
			"content_type":  map[string]string{"type": "string"},
			"etag":          map[string]string{"type": "string"},
			"last_modified": map[string]string{"type": "string", "format": "date-time"},
			"metadata":      map[string]interface{}{"type": "object", "additionalProperties": map[string]string{"type": "string"}},
		},
	}

	// Bucket schema
	spec.Components.Schemas["Bucket"] = map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"name":         map[string]string{"type": "string"},
			"created_date": map[string]string{"type": "string", "format": "date-time"},
		},
	}

	// GET /api/v1/storage/buckets - List all buckets
	spec.Paths["/api/v1/storage/buckets"] = OpenAPIPath{
		"get": OpenAPIOperation{
			Summary:     "List all storage buckets",
			Description: "Retrieve a list of all available storage buckets",
			OperationID: "list_buckets",
			Tags:        []string{"Storage"},
			Responses: map[string]OpenAPIResponse{
				"200": {
					Description: "List of buckets",
					Content: map[string]OpenAPIMedia{
						"application/json": {
							Schema: map[string]interface{}{
								"type": "object",
								"properties": map[string]interface{}{
									"buckets": map[string]interface{}{
										"type":  "array",
										"items": map[string]string{"$ref": "#/components/schemas/Bucket"},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	// POST /api/v1/storage/buckets/:bucket - Create bucket
	spec.Paths["/api/v1/storage/buckets/{bucket}"] = OpenAPIPath{
		"post": OpenAPIOperation{
			Summary:     "Create a new storage bucket",
			Description: "Create a new bucket for storing files",
			OperationID: "create_bucket",
			Tags:        []string{"Storage"},
			Parameters: []OpenAPIParameter{
				{
					Name:        "bucket",
					In:          "path",
					Description: "Bucket name",
					Required:    true,
					Schema:      map[string]string{"type": "string"},
				},
			},
			Responses: map[string]OpenAPIResponse{
				"201": {
					Description: "Bucket created",
					Content: map[string]OpenAPIMedia{
						"application/json": {
							Schema: map[string]interface{}{
								"type": "object",
								"properties": map[string]interface{}{
									"bucket":  map[string]string{"type": "string"},
									"message": map[string]string{"type": "string"},
								},
							},
						},
					},
				},
				"409": {
					Description: "Bucket already exists",
					Content: map[string]OpenAPIMedia{
						"application/json": {
							Schema: map[string]string{"$ref": "#/components/schemas/Error"},
						},
					},
				},
			},
		},
		"delete": OpenAPIOperation{
			Summary:     "Delete a storage bucket",
			Description: "Delete an empty bucket",
			OperationID: "delete_bucket",
			Tags:        []string{"Storage"},
			Parameters: []OpenAPIParameter{
				{
					Name:        "bucket",
					In:          "path",
					Description: "Bucket name",
					Required:    true,
					Schema:      map[string]string{"type": "string"},
				},
			},
			Responses: map[string]OpenAPIResponse{
				"204": {
					Description: "Bucket deleted",
				},
				"404": {
					Description: "Bucket not found",
					Content: map[string]OpenAPIMedia{
						"application/json": {
							Schema: map[string]string{"$ref": "#/components/schemas/Error"},
						},
					},
				},
				"409": {
					Description: "Bucket is not empty",
					Content: map[string]OpenAPIMedia{
						"application/json": {
							Schema: map[string]string{"$ref": "#/components/schemas/Error"},
						},
					},
				},
			},
		},
	}

	// GET /api/v1/storage/:bucket - List files in bucket
	spec.Paths["/api/v1/storage/{bucket}"] = OpenAPIPath{
		"get": OpenAPIOperation{
			Summary:     "List files in bucket",
			Description: "List all files in a specific bucket with optional filtering",
			OperationID: "list_files",
			Tags:        []string{"Storage"},
			Parameters: []OpenAPIParameter{
				{
					Name:        "bucket",
					In:          "path",
					Description: "Bucket name",
					Required:    true,
					Schema:      map[string]string{"type": "string"},
				},
				{
					Name:        "prefix",
					In:          "query",
					Description: "Filter files by prefix",
					Required:    false,
					Schema:      map[string]string{"type": "string"},
				},
				{
					Name:        "delimiter",
					In:          "query",
					Description: "Delimiter for grouping",
					Required:    false,
					Schema:      map[string]string{"type": "string"},
				},
				{
					Name:        "limit",
					In:          "query",
					Description: "Maximum number of files to return",
					Required:    false,
					Schema:      map[string]string{"type": "integer", "default": "1000"},
				},
			},
			Responses: map[string]OpenAPIResponse{
				"200": {
					Description: "List of files",
					Content: map[string]OpenAPIMedia{
						"application/json": {
							Schema: map[string]interface{}{
								"type": "object",
								"properties": map[string]interface{}{
									"bucket": map[string]string{"type": "string"},
									"objects": map[string]interface{}{
										"type":  "array",
										"items": map[string]string{"$ref": "#/components/schemas/StorageObject"},
									},
									"prefixes": map[string]interface{}{
										"type":  "array",
										"items": map[string]string{"type": "string"},
									},
									"truncated": map[string]string{"type": "boolean"},
								},
							},
						},
					},
				},
			},
		},
	}

	// File operations: Upload, Download, Delete, Get Info
	spec.Paths["/api/v1/storage/{bucket}/{key}"] = OpenAPIPath{
		"post": OpenAPIOperation{
			Summary:     "Upload a file",
			Description: "Upload a file to the specified bucket and key",
			OperationID: "upload_file",
			Tags:        []string{"Storage"},
			Parameters: []OpenAPIParameter{
				{
					Name:        "bucket",
					In:          "path",
					Description: "Bucket name",
					Required:    true,
					Schema:      map[string]string{"type": "string"},
				},
				{
					Name:        "key",
					In:          "path",
					Description: "File key (path)",
					Required:    true,
					Schema:      map[string]string{"type": "string"},
				},
			},
			RequestBody: &OpenAPIRequestBody{
				Required:    true,
				Description: "File to upload",
				Content: map[string]OpenAPIMedia{
					"multipart/form-data": {
						Schema: map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"file": map[string]string{
									"type":   "string",
									"format": "binary",
								},
							},
							"required": []string{"file"},
						},
					},
				},
			},
			Responses: map[string]OpenAPIResponse{
				"201": {
					Description: "File uploaded",
					Content: map[string]OpenAPIMedia{
						"application/json": {
							Schema: map[string]string{"$ref": "#/components/schemas/StorageObject"},
						},
					},
				},
			},
		},
		"get": OpenAPIOperation{
			Summary:     "Download a file",
			Description: "Download a file from storage",
			OperationID: "download_file",
			Tags:        []string{"Storage"},
			Parameters: []OpenAPIParameter{
				{
					Name:        "bucket",
					In:          "path",
					Description: "Bucket name",
					Required:    true,
					Schema:      map[string]string{"type": "string"},
				},
				{
					Name:        "key",
					In:          "path",
					Description: "File key (path)",
					Required:    true,
					Schema:      map[string]string{"type": "string"},
				},
				{
					Name:        "download",
					In:          "query",
					Description: "Force download (set Content-Disposition header)",
					Required:    false,
					Schema:      map[string]string{"type": "boolean"},
				},
			},
			Responses: map[string]OpenAPIResponse{
				"200": {
					Description: "File content",
					Content: map[string]OpenAPIMedia{
						"application/octet-stream": {
							Schema: map[string]string{
								"type":   "string",
								"format": "binary",
							},
						},
					},
				},
				"404": {
					Description: "File not found",
					Content: map[string]OpenAPIMedia{
						"application/json": {
							Schema: map[string]string{"$ref": "#/components/schemas/Error"},
						},
					},
				},
			},
		},
		"head": OpenAPIOperation{
			Summary:     "Get file metadata",
			Description: "Get metadata about a file without downloading it",
			OperationID: "get_file_info",
			Tags:        []string{"Storage"},
			Parameters: []OpenAPIParameter{
				{
					Name:        "bucket",
					In:          "path",
					Description: "Bucket name",
					Required:    true,
					Schema:      map[string]string{"type": "string"},
				},
				{
					Name:        "key",
					In:          "path",
					Description: "File key (path)",
					Required:    true,
					Schema:      map[string]string{"type": "string"},
				},
			},
			Responses: map[string]OpenAPIResponse{
				"200": {
					Description: "File metadata",
					Content: map[string]OpenAPIMedia{
						"application/json": {
							Schema: map[string]string{"$ref": "#/components/schemas/StorageObject"},
						},
					},
				},
				"404": {
					Description: "File not found",
					Content: map[string]OpenAPIMedia{
						"application/json": {
							Schema: map[string]string{"$ref": "#/components/schemas/Error"},
						},
					},
				},
			},
		},
		"delete": OpenAPIOperation{
			Summary:     "Delete a file",
			Description: "Delete a file from storage",
			OperationID: "delete_file",
			Tags:        []string{"Storage"},
			Parameters: []OpenAPIParameter{
				{
					Name:        "bucket",
					In:          "path",
					Description: "Bucket name",
					Required:    true,
					Schema:      map[string]string{"type": "string"},
				},
				{
					Name:        "key",
					In:          "path",
					Description: "File key (path)",
					Required:    true,
					Schema:      map[string]string{"type": "string"},
				},
			},
			Responses: map[string]OpenAPIResponse{
				"204": {
					Description: "File deleted",
				},
				"404": {
					Description: "File not found",
					Content: map[string]OpenAPIMedia{
						"application/json": {
							Schema: map[string]string{"$ref": "#/components/schemas/Error"},
						},
					},
				},
			},
		},
	}

	// POST /api/v1/storage/:bucket/:key/signed-url - Generate signed URL
	spec.Paths["/api/v1/storage/{bucket}/{key}/signed-url"] = OpenAPIPath{
		"post": OpenAPIOperation{
			Summary:     "Generate signed URL",
			Description: "Generate a presigned URL for temporary file access (not supported for local storage)",
			OperationID: "generate_signed_url",
			Tags:        []string{"Storage"},
			Parameters: []OpenAPIParameter{
				{
					Name:        "bucket",
					In:          "path",
					Description: "Bucket name",
					Required:    true,
					Schema:      map[string]string{"type": "string"},
				},
				{
					Name:        "key",
					In:          "path",
					Description: "File key (path)",
					Required:    true,
					Schema:      map[string]string{"type": "string"},
				},
			},
			RequestBody: &OpenAPIRequestBody{
				Description: "Signed URL options",
				Content: map[string]OpenAPIMedia{
					"application/json": {
						Schema: map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"expires_in": map[string]string{
									"type":        "integer",
									"description": "URL expiration time in seconds (default: 900)",
								},
								"method": map[string]interface{}{
									"type":        "string",
									"enum":        []string{"GET", "PUT", "DELETE"},
									"description": "HTTP method for the signed URL (default: GET)",
								},
							},
						},
					},
				},
			},
			Responses: map[string]OpenAPIResponse{
				"200": {
					Description: "Signed URL generated",
					Content: map[string]OpenAPIMedia{
						"application/json": {
							Schema: map[string]interface{}{
								"type": "object",
								"properties": map[string]interface{}{
									"url":        map[string]string{"type": "string"},
									"expires_at": map[string]string{"type": "string", "format": "date-time"},
								},
							},
						},
					},
				},
				"501": {
					Description: "Not supported for local storage",
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

// addAPIKeyEndpoints adds API key management endpoints to the spec
func (h *OpenAPIHandler) addAPIKeyEndpoints(spec *OpenAPISpec) {
	// API Key schema
	spec.Components.Schemas["APIKey"] = map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"id":                    map[string]string{"type": "string", "format": "uuid"},
			"name":                  map[string]string{"type": "string"},
			"description":           map[string]string{"type": "string"},
			"key_prefix":            map[string]string{"type": "string"},
			"scopes":                map[string]interface{}{"type": "array", "items": map[string]string{"type": "string"}},
			"rate_limit_per_minute": map[string]string{"type": "integer"},
			"expires_at":            map[string]string{"type": "string", "format": "date-time"},
			"created_at":            map[string]string{"type": "string", "format": "date-time"},
			"last_used_at":          map[string]string{"type": "string", "format": "date-time"},
			"is_active":             map[string]string{"type": "boolean"},
		},
	}

	spec.Components.Schemas["CreateAPIKeyRequest"] = map[string]interface{}{
		"type":     "object",
		"required": []string{"name", "scopes"},
		"properties": map[string]interface{}{
			"name":                  map[string]string{"type": "string"},
			"description":           map[string]string{"type": "string"},
			"scopes":                map[string]interface{}{"type": "array", "items": map[string]string{"type": "string"}},
			"rate_limit_per_minute": map[string]string{"type": "integer"},
			"expires_at":            map[string]string{"type": "string", "format": "date-time"},
		},
	}

	spec.Components.Schemas["CreateAPIKeyResponse"] = map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"id":                    map[string]string{"type": "string", "format": "uuid"},
			"name":                  map[string]string{"type": "string"},
			"key":                   map[string]string{"type": "string", "description": "The full API key (only shown once)"},
			"key_prefix":            map[string]string{"type": "string"},
			"scopes":                map[string]interface{}{"type": "array", "items": map[string]string{"type": "string"}},
			"rate_limit_per_minute": map[string]string{"type": "integer"},
			"expires_at":            map[string]string{"type": "string", "format": "date-time"},
			"created_at":            map[string]string{"type": "string", "format": "date-time"},
		},
	}

	// GET /api/v1/client-keys - List client keys
	spec.Paths["/api/v1/client-keys"] = OpenAPIPath{
		"get": OpenAPIOperation{
			Summary:     "List client keys",
			Description: "Get all client keys for the authenticated user",
			OperationID: "apikeys_list",
			Tags:        []string{"client keys"},
			Security: []map[string][]string{
				{"bearerAuth": {}},
			},
			Responses: map[string]OpenAPIResponse{
				"200": {
					Description: "List of client keys",
					Content: map[string]OpenAPIMedia{
						"application/json": {
							Schema: map[string]interface{}{
								"type":  "array",
								"items": map[string]string{"$ref": "#/components/schemas/APIKey"},
							},
						},
					},
				},
			},
		},
		"post": OpenAPIOperation{
			Summary:     "Create API key",
			Description: "Create a new API key",
			OperationID: "apikeys_create",
			Tags:        []string{"client keys"},
			Security: []map[string][]string{
				{"bearerAuth": {}},
			},
			RequestBody: &OpenAPIRequestBody{
				Required: true,
				Content: map[string]OpenAPIMedia{
					"application/json": {
						Schema: map[string]string{"$ref": "#/components/schemas/CreateAPIKeyRequest"},
					},
				},
			},
			Responses: map[string]OpenAPIResponse{
				"201": {
					Description: "API key created",
					Content: map[string]OpenAPIMedia{
						"application/json": {
							Schema: map[string]string{"$ref": "#/components/schemas/CreateAPIKeyResponse"},
						},
					},
				},
			},
		},
	}

	// GET/PATCH/DELETE /api/v1/client-keys/:id
	spec.Paths["/api/v1/client-keys/{id}"] = OpenAPIPath{
		"get": OpenAPIOperation{
			Summary:     "Get API key",
			Description: "Get details of a specific API key",
			OperationID: "apikeys_get",
			Tags:        []string{"client keys"},
			Security: []map[string][]string{
				{"bearerAuth": {}},
			},
			Parameters: []OpenAPIParameter{
				{Name: "id", In: "path", Required: true, Schema: map[string]string{"type": "string", "format": "uuid"}},
			},
			Responses: map[string]OpenAPIResponse{
				"200": {
					Description: "API key details",
					Content: map[string]OpenAPIMedia{
						"application/json": {
							Schema: map[string]string{"$ref": "#/components/schemas/APIKey"},
						},
					},
				},
				"404": {
					Description: "API key not found",
					Content: map[string]OpenAPIMedia{
						"application/json": {
							Schema: map[string]string{"$ref": "#/components/schemas/Error"},
						},
					},
				},
			},
		},
		"patch": OpenAPIOperation{
			Summary:     "Update API key",
			Description: "Update an existing API key",
			OperationID: "apikeys_update",
			Tags:        []string{"client keys"},
			Security: []map[string][]string{
				{"bearerAuth": {}},
			},
			Parameters: []OpenAPIParameter{
				{Name: "id", In: "path", Required: true, Schema: map[string]string{"type": "string", "format": "uuid"}},
			},
			RequestBody: &OpenAPIRequestBody{
				Required: true,
				Content: map[string]OpenAPIMedia{
					"application/json": {
						Schema: map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"name":                  map[string]string{"type": "string"},
								"description":           map[string]string{"type": "string"},
								"scopes":                map[string]interface{}{"type": "array", "items": map[string]string{"type": "string"}},
								"rate_limit_per_minute": map[string]string{"type": "integer"},
							},
						},
					},
				},
			},
			Responses: map[string]OpenAPIResponse{
				"200": {
					Description: "API key updated",
					Content: map[string]OpenAPIMedia{
						"application/json": {
							Schema: map[string]string{"$ref": "#/components/schemas/APIKey"},
						},
					},
				},
			},
		},
		"delete": OpenAPIOperation{
			Summary:     "Delete API key",
			Description: "Delete an API key",
			OperationID: "apikeys_delete",
			Tags:        []string{"client keys"},
			Security: []map[string][]string{
				{"bearerAuth": {}},
			},
			Parameters: []OpenAPIParameter{
				{Name: "id", In: "path", Required: true, Schema: map[string]string{"type": "string", "format": "uuid"}},
			},
			Responses: map[string]OpenAPIResponse{
				"204": {
					Description: "API key deleted",
				},
			},
		},
	}

	// POST /api/v1/client-keys/:id/revoke
	spec.Paths["/api/v1/client-keys/{id}/revoke"] = OpenAPIPath{
		"post": OpenAPIOperation{
			Summary:     "Revoke API key",
			Description: "Revoke an API key (deactivate without deleting)",
			OperationID: "apikeys_revoke",
			Tags:        []string{"client keys"},
			Security: []map[string][]string{
				{"bearerAuth": {}},
			},
			Parameters: []OpenAPIParameter{
				{Name: "id", In: "path", Required: true, Schema: map[string]string{"type": "string", "format": "uuid"}},
			},
			Responses: map[string]OpenAPIResponse{
				"200": {
					Description: "API key revoked",
					Content: map[string]OpenAPIMedia{
						"application/json": {
							Schema: map[string]string{"$ref": "#/components/schemas/APIKey"},
						},
					},
				},
			},
		},
	}
}

// addWebhookEndpoints adds webhook management endpoints to the spec
func (h *OpenAPIHandler) addWebhookEndpoints(spec *OpenAPISpec) {
	// Webhook schema
	spec.Components.Schemas["Webhook"] = map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"id":          map[string]string{"type": "string", "format": "uuid"},
			"name":        map[string]string{"type": "string"},
			"url":         map[string]string{"type": "string", "format": "uri"},
			"events":      map[string]interface{}{"type": "array", "items": map[string]string{"type": "string"}},
			"secret":      map[string]string{"type": "string"},
			"is_active":   map[string]string{"type": "boolean"},
			"created_at":  map[string]string{"type": "string", "format": "date-time"},
			"updated_at":  map[string]string{"type": "string", "format": "date-time"},
			"last_status": map[string]string{"type": "string"},
		},
	}

	spec.Components.Schemas["WebhookDelivery"] = map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"id":            map[string]string{"type": "string", "format": "uuid"},
			"webhook_id":    map[string]string{"type": "string", "format": "uuid"},
			"event":         map[string]string{"type": "string"},
			"payload":       map[string]string{"type": "object"},
			"response_code": map[string]string{"type": "integer"},
			"response_body": map[string]string{"type": "string"},
			"created_at":    map[string]string{"type": "string", "format": "date-time"},
			"delivered_at":  map[string]string{"type": "string", "format": "date-time"},
		},
	}

	// GET/POST /api/v1/webhooks
	spec.Paths["/api/v1/webhooks"] = OpenAPIPath{
		"get": OpenAPIOperation{
			Summary:     "List webhooks",
			Description: "Get all webhooks",
			OperationID: "webhooks_list",
			Tags:        []string{"Webhooks"},
			Security: []map[string][]string{
				{"bearerAuth": {}},
			},
			Responses: map[string]OpenAPIResponse{
				"200": {
					Description: "List of webhooks",
					Content: map[string]OpenAPIMedia{
						"application/json": {
							Schema: map[string]interface{}{
								"type":  "array",
								"items": map[string]string{"$ref": "#/components/schemas/Webhook"},
							},
						},
					},
				},
			},
		},
		"post": OpenAPIOperation{
			Summary:     "Create webhook",
			Description: "Create a new webhook",
			OperationID: "webhooks_create",
			Tags:        []string{"Webhooks"},
			Security: []map[string][]string{
				{"bearerAuth": {}},
			},
			RequestBody: &OpenAPIRequestBody{
				Required: true,
				Content: map[string]OpenAPIMedia{
					"application/json": {
						Schema: map[string]interface{}{
							"type":     "object",
							"required": []string{"name", "url", "events"},
							"properties": map[string]interface{}{
								"name":   map[string]string{"type": "string"},
								"url":    map[string]string{"type": "string", "format": "uri"},
								"events": map[string]interface{}{"type": "array", "items": map[string]string{"type": "string"}},
								"secret": map[string]string{"type": "string"},
							},
						},
					},
				},
			},
			Responses: map[string]OpenAPIResponse{
				"201": {
					Description: "Webhook created",
					Content: map[string]OpenAPIMedia{
						"application/json": {
							Schema: map[string]string{"$ref": "#/components/schemas/Webhook"},
						},
					},
				},
			},
		},
	}

	// GET/PATCH/DELETE /api/v1/webhooks/:id
	spec.Paths["/api/v1/webhooks/{id}"] = OpenAPIPath{
		"get": OpenAPIOperation{
			Summary:     "Get webhook",
			Description: "Get webhook details",
			OperationID: "webhooks_get",
			Tags:        []string{"Webhooks"},
			Security: []map[string][]string{
				{"bearerAuth": {}},
			},
			Parameters: []OpenAPIParameter{
				{Name: "id", In: "path", Required: true, Schema: map[string]string{"type": "string", "format": "uuid"}},
			},
			Responses: map[string]OpenAPIResponse{
				"200": {
					Description: "Webhook details",
					Content: map[string]OpenAPIMedia{
						"application/json": {
							Schema: map[string]string{"$ref": "#/components/schemas/Webhook"},
						},
					},
				},
			},
		},
		"patch": OpenAPIOperation{
			Summary:     "Update webhook",
			Description: "Update a webhook",
			OperationID: "webhooks_update",
			Tags:        []string{"Webhooks"},
			Security: []map[string][]string{
				{"bearerAuth": {}},
			},
			Parameters: []OpenAPIParameter{
				{Name: "id", In: "path", Required: true, Schema: map[string]string{"type": "string", "format": "uuid"}},
			},
			RequestBody: &OpenAPIRequestBody{
				Required: true,
				Content: map[string]OpenAPIMedia{
					"application/json": {
						Schema: map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"name":      map[string]string{"type": "string"},
								"url":       map[string]string{"type": "string", "format": "uri"},
								"events":    map[string]interface{}{"type": "array", "items": map[string]string{"type": "string"}},
								"is_active": map[string]string{"type": "boolean"},
							},
						},
					},
				},
			},
			Responses: map[string]OpenAPIResponse{
				"200": {
					Description: "Webhook updated",
					Content: map[string]OpenAPIMedia{
						"application/json": {
							Schema: map[string]string{"$ref": "#/components/schemas/Webhook"},
						},
					},
				},
			},
		},
		"delete": OpenAPIOperation{
			Summary:     "Delete webhook",
			Description: "Delete a webhook",
			OperationID: "webhooks_delete",
			Tags:        []string{"Webhooks"},
			Security: []map[string][]string{
				{"bearerAuth": {}},
			},
			Parameters: []OpenAPIParameter{
				{Name: "id", In: "path", Required: true, Schema: map[string]string{"type": "string", "format": "uuid"}},
			},
			Responses: map[string]OpenAPIResponse{
				"204": {
					Description: "Webhook deleted",
				},
			},
		},
	}

	// POST /api/v1/webhooks/:id/test
	spec.Paths["/api/v1/webhooks/{id}/test"] = OpenAPIPath{
		"post": OpenAPIOperation{
			Summary:     "Test webhook",
			Description: "Send a test event to the webhook",
			OperationID: "webhooks_test",
			Tags:        []string{"Webhooks"},
			Security: []map[string][]string{
				{"bearerAuth": {}},
			},
			Parameters: []OpenAPIParameter{
				{Name: "id", In: "path", Required: true, Schema: map[string]string{"type": "string", "format": "uuid"}},
			},
			Responses: map[string]OpenAPIResponse{
				"200": {
					Description: "Test event sent",
					Content: map[string]OpenAPIMedia{
						"application/json": {
							Schema: map[string]interface{}{
								"type": "object",
								"properties": map[string]interface{}{
									"success":       map[string]string{"type": "boolean"},
									"response_code": map[string]string{"type": "integer"},
									"response_body": map[string]string{"type": "string"},
								},
							},
						},
					},
				},
			},
		},
	}

	// GET /api/v1/webhooks/:id/deliveries
	spec.Paths["/api/v1/webhooks/{id}/deliveries"] = OpenAPIPath{
		"get": OpenAPIOperation{
			Summary:     "List webhook deliveries",
			Description: "Get delivery history for a webhook",
			OperationID: "webhooks_deliveries",
			Tags:        []string{"Webhooks"},
			Security: []map[string][]string{
				{"bearerAuth": {}},
			},
			Parameters: []OpenAPIParameter{
				{Name: "id", In: "path", Required: true, Schema: map[string]string{"type": "string", "format": "uuid"}},
				{Name: "limit", In: "query", Schema: map[string]string{"type": "integer"}},
				{Name: "offset", In: "query", Schema: map[string]string{"type": "integer"}},
			},
			Responses: map[string]OpenAPIResponse{
				"200": {
					Description: "List of deliveries",
					Content: map[string]OpenAPIMedia{
						"application/json": {
							Schema: map[string]interface{}{
								"type":  "array",
								"items": map[string]string{"$ref": "#/components/schemas/WebhookDelivery"},
							},
						},
					},
				},
			},
		},
	}
}

// addMonitoringEndpoints adds monitoring endpoints to the spec
func (h *OpenAPIHandler) addMonitoringEndpoints(spec *OpenAPISpec) {
	// Metrics schema
	spec.Components.Schemas["SystemMetrics"] = map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"cpu_usage":    map[string]string{"type": "number"},
			"memory_usage": map[string]string{"type": "number"},
			"disk_usage":   map[string]string{"type": "number"},
			"connections":  map[string]string{"type": "integer"},
			"requests":     map[string]string{"type": "integer"},
			"timestamp":    map[string]string{"type": "string", "format": "date-time"},
		},
	}

	spec.Components.Schemas["HealthStatus"] = map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"status":   map[string]string{"type": "string"},
			"database": map[string]string{"type": "string"},
			"storage":  map[string]string{"type": "string"},
			"uptime":   map[string]string{"type": "integer"},
		},
	}

	// GET /api/v1/monitoring/metrics
	spec.Paths["/api/v1/monitoring/metrics"] = OpenAPIPath{
		"get": OpenAPIOperation{
			Summary:     "Get system metrics",
			Description: "Get current system metrics",
			OperationID: "monitoring_metrics",
			Tags:        []string{"Monitoring"},
			Security: []map[string][]string{
				{"bearerAuth": {}},
			},
			Responses: map[string]OpenAPIResponse{
				"200": {
					Description: "System metrics",
					Content: map[string]OpenAPIMedia{
						"application/json": {
							Schema: map[string]string{"$ref": "#/components/schemas/SystemMetrics"},
						},
					},
				},
			},
		},
	}

	// GET /api/v1/monitoring/health
	spec.Paths["/api/v1/monitoring/health"] = OpenAPIPath{
		"get": OpenAPIOperation{
			Summary:     "Get health status",
			Description: "Get system health status",
			OperationID: "monitoring_health",
			Tags:        []string{"Monitoring"},
			Security: []map[string][]string{
				{"bearerAuth": {}},
			},
			Responses: map[string]OpenAPIResponse{
				"200": {
					Description: "Health status",
					Content: map[string]OpenAPIMedia{
						"application/json": {
							Schema: map[string]string{"$ref": "#/components/schemas/HealthStatus"},
						},
					},
				},
			},
		},
	}

	// GET /api/v1/monitoring/logs
	spec.Paths["/api/v1/monitoring/logs"] = OpenAPIPath{
		"get": OpenAPIOperation{
			Summary:     "Get system logs",
			Description: "Get recent system logs",
			OperationID: "monitoring_logs",
			Tags:        []string{"Monitoring"},
			Security: []map[string][]string{
				{"bearerAuth": {}},
			},
			Parameters: []OpenAPIParameter{
				{Name: "level", In: "query", Description: "Log level filter", Schema: map[string]string{"type": "string"}},
				{Name: "limit", In: "query", Schema: map[string]string{"type": "integer"}},
				{Name: "since", In: "query", Description: "Start time", Schema: map[string]string{"type": "string", "format": "date-time"}},
			},
			Responses: map[string]OpenAPIResponse{
				"200": {
					Description: "System logs",
					Content: map[string]OpenAPIMedia{
						"application/json": {
							Schema: map[string]interface{}{
								"type": "array",
								"items": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"timestamp": map[string]string{"type": "string", "format": "date-time"},
										"level":     map[string]string{"type": "string"},
										"message":   map[string]string{"type": "string"},
										"metadata":  map[string]string{"type": "object"},
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

// addVectorEndpoints adds vector/embedding endpoints to the spec
func (h *OpenAPIHandler) addVectorEndpoints(spec *OpenAPISpec) {
	// Vector schemas
	spec.Components.Schemas["EmbedRequest"] = map[string]interface{}{
		"type":     "object",
		"required": []string{"input"},
		"properties": map[string]interface{}{
			"input": map[string]interface{}{
				"oneOf": []map[string]interface{}{
					{"type": "string"},
					{"type": "array", "items": map[string]string{"type": "string"}},
				},
			},
			"model": map[string]string{"type": "string"},
		},
	}

	spec.Components.Schemas["EmbedResponse"] = map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"embeddings": map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type":  "array",
					"items": map[string]string{"type": "number"},
				},
			},
			"model":      map[string]string{"type": "string"},
			"usage":      map[string]interface{}{"type": "object"},
			"dimensions": map[string]string{"type": "integer"},
		},
	}

	spec.Components.Schemas["VectorSearchRequest"] = map[string]interface{}{
		"type":     "object",
		"required": []string{"table", "column"},
		"properties": map[string]interface{}{
			"table":           map[string]string{"type": "string"},
			"column":          map[string]string{"type": "string"},
			"query":           map[string]string{"type": "string"},
			"query_embedding": map[string]interface{}{"type": "array", "items": map[string]string{"type": "number"}},
			"limit":           map[string]string{"type": "integer"},
			"threshold":       map[string]string{"type": "number"},
			"filter":          map[string]string{"type": "object"},
		},
	}

	spec.Components.Schemas["VectorSearchResult"] = map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"results": map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"id":         map[string]string{"type": "string"},
						"similarity": map[string]string{"type": "number"},
						"data":       map[string]string{"type": "object"},
					},
				},
			},
		},
	}

	// GET /api/v1/capabilities/vector
	spec.Paths["/api/v1/capabilities/vector"] = OpenAPIPath{
		"get": OpenAPIOperation{
			Summary:     "Get vector capabilities",
			Description: "Get available vector/embedding capabilities",
			OperationID: "vector_capabilities",
			Tags:        []string{"Vector"},
			Responses: map[string]OpenAPIResponse{
				"200": {
					Description: "Vector capabilities",
					Content: map[string]OpenAPIMedia{
						"application/json": {
							Schema: map[string]interface{}{
								"type": "object",
								"properties": map[string]interface{}{
									"enabled":    map[string]string{"type": "boolean"},
									"models":     map[string]interface{}{"type": "array", "items": map[string]string{"type": "string"}},
									"dimensions": map[string]string{"type": "integer"},
								},
							},
						},
					},
				},
			},
		},
	}

	// POST /api/v1/vector/embed
	spec.Paths["/api/v1/vector/embed"] = OpenAPIPath{
		"post": OpenAPIOperation{
			Summary:     "Generate embeddings",
			Description: "Generate vector embeddings for text input",
			OperationID: "vector_embed",
			Tags:        []string{"Vector"},
			Security: []map[string][]string{
				{"bearerAuth": {}},
			},
			RequestBody: &OpenAPIRequestBody{
				Required: true,
				Content: map[string]OpenAPIMedia{
					"application/json": {
						Schema: map[string]string{"$ref": "#/components/schemas/EmbedRequest"},
					},
				},
			},
			Responses: map[string]OpenAPIResponse{
				"200": {
					Description: "Embeddings generated",
					Content: map[string]OpenAPIMedia{
						"application/json": {
							Schema: map[string]string{"$ref": "#/components/schemas/EmbedResponse"},
						},
					},
				},
			},
		},
	}

	// POST /api/v1/vector/search
	spec.Paths["/api/v1/vector/search"] = OpenAPIPath{
		"post": OpenAPIOperation{
			Summary:     "Vector similarity search",
			Description: "Search for similar vectors in a table",
			OperationID: "vector_search",
			Tags:        []string{"Vector"},
			Security: []map[string][]string{
				{"bearerAuth": {}},
			},
			RequestBody: &OpenAPIRequestBody{
				Required: true,
				Content: map[string]OpenAPIMedia{
					"application/json": {
						Schema: map[string]string{"$ref": "#/components/schemas/VectorSearchRequest"},
					},
				},
			},
			Responses: map[string]OpenAPIResponse{
				"200": {
					Description: "Search results",
					Content: map[string]OpenAPIMedia{
						"application/json": {
							Schema: map[string]string{"$ref": "#/components/schemas/VectorSearchResult"},
						},
					},
				},
			},
		},
	}
}

// addRealtimeEndpoints adds realtime endpoints to the spec
func (h *OpenAPIHandler) addRealtimeEndpoints(spec *OpenAPISpec) {
	// Realtime schemas
	spec.Components.Schemas["RealtimeStats"] = map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"connections":      map[string]string{"type": "integer"},
			"channels":         map[string]string{"type": "integer"},
			"messages_per_sec": map[string]string{"type": "number"},
		},
	}

	spec.Components.Schemas["BroadcastRequest"] = map[string]interface{}{
		"type":     "object",
		"required": []string{"channel", "event", "payload"},
		"properties": map[string]interface{}{
			"channel": map[string]string{"type": "string"},
			"event":   map[string]string{"type": "string"},
			"payload": map[string]string{"type": "object"},
		},
	}

	// GET /realtime - WebSocket endpoint
	spec.Paths["/realtime"] = OpenAPIPath{
		"get": OpenAPIOperation{
			Summary:     "WebSocket connection",
			Description: "Establish a WebSocket connection for realtime updates. Upgrade to WebSocket protocol required.",
			OperationID: "realtime_connect",
			Tags:        []string{"Realtime"},
			Responses: map[string]OpenAPIResponse{
				"101": {
					Description: "Switching Protocols - WebSocket connection established",
				},
			},
		},
	}

	// GET /api/v1/realtime/stats
	spec.Paths["/api/v1/realtime/stats"] = OpenAPIPath{
		"get": OpenAPIOperation{
			Summary:     "Get realtime stats",
			Description: "Get current realtime connection statistics",
			OperationID: "realtime_stats",
			Tags:        []string{"Realtime"},
			Security: []map[string][]string{
				{"bearerAuth": {}},
			},
			Responses: map[string]OpenAPIResponse{
				"200": {
					Description: "Realtime statistics",
					Content: map[string]OpenAPIMedia{
						"application/json": {
							Schema: map[string]string{"$ref": "#/components/schemas/RealtimeStats"},
						},
					},
				},
			},
		},
	}

	// POST /api/v1/realtime/broadcast
	spec.Paths["/api/v1/realtime/broadcast"] = OpenAPIPath{
		"post": OpenAPIOperation{
			Summary:     "Broadcast message",
			Description: "Broadcast a message to a channel",
			OperationID: "realtime_broadcast",
			Tags:        []string{"Realtime"},
			Security: []map[string][]string{
				{"bearerAuth": {}},
			},
			RequestBody: &OpenAPIRequestBody{
				Required: true,
				Content: map[string]OpenAPIMedia{
					"application/json": {
						Schema: map[string]string{"$ref": "#/components/schemas/BroadcastRequest"},
					},
				},
			},
			Responses: map[string]OpenAPIResponse{
				"200": {
					Description: "Message broadcast",
					Content: map[string]OpenAPIMedia{
						"application/json": {
							Schema: map[string]interface{}{
								"type": "object",
								"properties": map[string]interface{}{
									"success":    map[string]string{"type": "boolean"},
									"recipients": map[string]string{"type": "integer"},
								},
							},
						},
					},
				},
			},
		},
	}
}

// addFunctionsEndpoints adds edge functions endpoints to the spec
func (h *OpenAPIHandler) addFunctionsEndpoints(spec *OpenAPISpec) {
	// Function schemas
	spec.Components.Schemas["EdgeFunction"] = map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"namespace":   map[string]string{"type": "string"},
			"name":        map[string]string{"type": "string"},
			"description": map[string]string{"type": "string"},
			"version":     map[string]string{"type": "string"},
			"created_at":  map[string]string{"type": "string", "format": "date-time"},
			"updated_at":  map[string]string{"type": "string", "format": "date-time"},
		},
	}

	// GET /api/v1/functions
	spec.Paths["/api/v1/functions"] = OpenAPIPath{
		"get": OpenAPIOperation{
			Summary:     "List functions",
			Description: "Get all edge functions",
			OperationID: "functions_list",
			Tags:        []string{"Functions"},
			Security: []map[string][]string{
				{"bearerAuth": {}},
			},
			Responses: map[string]OpenAPIResponse{
				"200": {
					Description: "List of functions",
					Content: map[string]OpenAPIMedia{
						"application/json": {
							Schema: map[string]interface{}{
								"type":  "array",
								"items": map[string]string{"$ref": "#/components/schemas/EdgeFunction"},
							},
						},
					},
				},
			},
		},
	}

	// POST /api/v1/functions/:namespace/:name - Invoke function
	spec.Paths["/api/v1/functions/{namespace}/{name}"] = OpenAPIPath{
		"post": OpenAPIOperation{
			Summary:     "Invoke function",
			Description: "Invoke an edge function",
			OperationID: "functions_invoke",
			Tags:        []string{"Functions"},
			Security: []map[string][]string{
				{"bearerAuth": {}},
			},
			Parameters: []OpenAPIParameter{
				{Name: "namespace", In: "path", Required: true, Schema: map[string]string{"type": "string"}},
				{Name: "name", In: "path", Required: true, Schema: map[string]string{"type": "string"}},
			},
			RequestBody: &OpenAPIRequestBody{
				Description: "Function input (optional)",
				Content: map[string]OpenAPIMedia{
					"application/json": {
						Schema: map[string]string{"type": "object"},
					},
				},
			},
			Responses: map[string]OpenAPIResponse{
				"200": {
					Description: "Function response",
					Content: map[string]OpenAPIMedia{
						"application/json": {
							Schema: map[string]string{"type": "object"},
						},
					},
				},
			},
		},
		"get": OpenAPIOperation{
			Summary:     "Invoke function (GET)",
			Description: "Invoke an edge function with GET request",
			OperationID: "functions_invoke_get",
			Tags:        []string{"Functions"},
			Security: []map[string][]string{
				{"bearerAuth": {}},
			},
			Parameters: []OpenAPIParameter{
				{Name: "namespace", In: "path", Required: true, Schema: map[string]string{"type": "string"}},
				{Name: "name", In: "path", Required: true, Schema: map[string]string{"type": "string"}},
			},
			Responses: map[string]OpenAPIResponse{
				"200": {
					Description: "Function response",
					Content: map[string]OpenAPIMedia{
						"application/json": {
							Schema: map[string]string{"type": "object"},
						},
					},
				},
			},
		},
	}
}

// addJobsEndpoints adds jobs endpoints to the spec
func (h *OpenAPIHandler) addJobsEndpoints(spec *OpenAPISpec) {
	// Job schemas
	spec.Components.Schemas["Job"] = map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"id":           map[string]string{"type": "string", "format": "uuid"},
			"function":     map[string]string{"type": "string"},
			"status":       map[string]string{"type": "string"},
			"input":        map[string]string{"type": "object"},
			"output":       map[string]string{"type": "object"},
			"error":        map[string]string{"type": "string"},
			"scheduled_at": map[string]string{"type": "string", "format": "date-time"},
			"started_at":   map[string]string{"type": "string", "format": "date-time"},
			"completed_at": map[string]string{"type": "string", "format": "date-time"},
			"created_at":   map[string]string{"type": "string", "format": "date-time"},
		},
	}

	spec.Components.Schemas["SubmitJobRequest"] = map[string]interface{}{
		"type":     "object",
		"required": []string{"function"},
		"properties": map[string]interface{}{
			"function":     map[string]string{"type": "string"},
			"input":        map[string]string{"type": "object"},
			"scheduled_at": map[string]string{"type": "string", "format": "date-time"},
			"priority":     map[string]string{"type": "integer"},
		},
	}

	// POST /api/v1/jobs/submit
	spec.Paths["/api/v1/jobs/submit"] = OpenAPIPath{
		"post": OpenAPIOperation{
			Summary:     "Submit job",
			Description: "Submit a new job to the queue",
			OperationID: "jobs_submit",
			Tags:        []string{"Jobs"},
			Security: []map[string][]string{
				{"bearerAuth": {}},
			},
			RequestBody: &OpenAPIRequestBody{
				Required: true,
				Content: map[string]OpenAPIMedia{
					"application/json": {
						Schema: map[string]string{"$ref": "#/components/schemas/SubmitJobRequest"},
					},
				},
			},
			Responses: map[string]OpenAPIResponse{
				"201": {
					Description: "Job submitted",
					Content: map[string]OpenAPIMedia{
						"application/json": {
							Schema: map[string]string{"$ref": "#/components/schemas/Job"},
						},
					},
				},
			},
		},
	}

	// GET /api/v1/jobs
	spec.Paths["/api/v1/jobs"] = OpenAPIPath{
		"get": OpenAPIOperation{
			Summary:     "List jobs",
			Description: "Get jobs for the authenticated user",
			OperationID: "jobs_list",
			Tags:        []string{"Jobs"},
			Security: []map[string][]string{
				{"bearerAuth": {}},
			},
			Parameters: []OpenAPIParameter{
				{Name: "status", In: "query", Description: "Filter by status", Schema: map[string]string{"type": "string"}},
				{Name: "limit", In: "query", Schema: map[string]string{"type": "integer"}},
				{Name: "offset", In: "query", Schema: map[string]string{"type": "integer"}},
			},
			Responses: map[string]OpenAPIResponse{
				"200": {
					Description: "List of jobs",
					Content: map[string]OpenAPIMedia{
						"application/json": {
							Schema: map[string]interface{}{
								"type":  "array",
								"items": map[string]string{"$ref": "#/components/schemas/Job"},
							},
						},
					},
				},
			},
		},
	}

	// GET /api/v1/jobs/:id
	spec.Paths["/api/v1/jobs/{id}"] = OpenAPIPath{
		"get": OpenAPIOperation{
			Summary:     "Get job",
			Description: "Get job details",
			OperationID: "jobs_get",
			Tags:        []string{"Jobs"},
			Security: []map[string][]string{
				{"bearerAuth": {}},
			},
			Parameters: []OpenAPIParameter{
				{Name: "id", In: "path", Required: true, Schema: map[string]string{"type": "string", "format": "uuid"}},
			},
			Responses: map[string]OpenAPIResponse{
				"200": {
					Description: "Job details",
					Content: map[string]OpenAPIMedia{
						"application/json": {
							Schema: map[string]string{"$ref": "#/components/schemas/Job"},
						},
					},
				},
			},
		},
	}

	// POST /api/v1/jobs/:id/cancel
	spec.Paths["/api/v1/jobs/{id}/cancel"] = OpenAPIPath{
		"post": OpenAPIOperation{
			Summary:     "Cancel job",
			Description: "Cancel a pending or running job",
			OperationID: "jobs_cancel",
			Tags:        []string{"Jobs"},
			Security: []map[string][]string{
				{"bearerAuth": {}},
			},
			Parameters: []OpenAPIParameter{
				{Name: "id", In: "path", Required: true, Schema: map[string]string{"type": "string", "format": "uuid"}},
			},
			Responses: map[string]OpenAPIResponse{
				"200": {
					Description: "Job cancelled",
					Content: map[string]OpenAPIMedia{
						"application/json": {
							Schema: map[string]string{"$ref": "#/components/schemas/Job"},
						},
					},
				},
			},
		},
	}

	// POST /api/v1/jobs/:id/retry
	spec.Paths["/api/v1/jobs/{id}/retry"] = OpenAPIPath{
		"post": OpenAPIOperation{
			Summary:     "Retry job",
			Description: "Retry a failed job",
			OperationID: "jobs_retry",
			Tags:        []string{"Jobs"},
			Security: []map[string][]string{
				{"bearerAuth": {}},
			},
			Parameters: []OpenAPIParameter{
				{Name: "id", In: "path", Required: true, Schema: map[string]string{"type": "string", "format": "uuid"}},
			},
			Responses: map[string]OpenAPIResponse{
				"200": {
					Description: "Job resubmitted",
					Content: map[string]OpenAPIMedia{
						"application/json": {
							Schema: map[string]string{"$ref": "#/components/schemas/Job"},
						},
					},
				},
			},
		},
	}
}

// addRPCEndpoints adds RPC endpoints to the spec
func (h *OpenAPIHandler) addRPCEndpoints(spec *OpenAPISpec) {
	// RPC schemas
	spec.Components.Schemas["RPCProcedure"] = map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"namespace":   map[string]string{"type": "string"},
			"name":        map[string]string{"type": "string"},
			"description": map[string]string{"type": "string"},
			"is_public":   map[string]string{"type": "boolean"},
			"parameters":  map[string]interface{}{"type": "array", "items": map[string]string{"type": "object"}},
		},
	}

	spec.Components.Schemas["RPCExecution"] = map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"id":           map[string]string{"type": "string", "format": "uuid"},
			"procedure":    map[string]string{"type": "string"},
			"status":       map[string]string{"type": "string"},
			"result":       map[string]string{"type": "object"},
			"error":        map[string]string{"type": "string"},
			"started_at":   map[string]string{"type": "string", "format": "date-time"},
			"completed_at": map[string]string{"type": "string", "format": "date-time"},
		},
	}

	// GET /api/v1/rpc/procedures
	spec.Paths["/api/v1/rpc/procedures"] = OpenAPIPath{
		"get": OpenAPIOperation{
			Summary:     "List RPC procedures",
			Description: "Get all available RPC procedures",
			OperationID: "rpc_procedures_list",
			Tags:        []string{"RPC"},
			Responses: map[string]OpenAPIResponse{
				"200": {
					Description: "List of procedures",
					Content: map[string]OpenAPIMedia{
						"application/json": {
							Schema: map[string]interface{}{
								"type":  "array",
								"items": map[string]string{"$ref": "#/components/schemas/RPCProcedure"},
							},
						},
					},
				},
			},
		},
	}

	// POST /api/v1/rpc/:namespace/:name
	spec.Paths["/api/v1/rpc/{namespace}/{name}"] = OpenAPIPath{
		"post": OpenAPIOperation{
			Summary:     "Invoke RPC procedure",
			Description: "Call a remote procedure",
			OperationID: "rpc_invoke",
			Tags:        []string{"RPC"},
			Security: []map[string][]string{
				{"bearerAuth": {}},
			},
			Parameters: []OpenAPIParameter{
				{Name: "namespace", In: "path", Required: true, Schema: map[string]string{"type": "string"}},
				{Name: "name", In: "path", Required: true, Schema: map[string]string{"type": "string"}},
			},
			RequestBody: &OpenAPIRequestBody{
				Description: "Procedure arguments",
				Content: map[string]OpenAPIMedia{
					"application/json": {
						Schema: map[string]string{"type": "object"},
					},
				},
			},
			Responses: map[string]OpenAPIResponse{
				"200": {
					Description: "Procedure result",
					Content: map[string]OpenAPIMedia{
						"application/json": {
							Schema: map[string]string{"$ref": "#/components/schemas/RPCExecution"},
						},
					},
				},
			},
		},
	}

	// GET /api/v1/rpc/executions/:id
	spec.Paths["/api/v1/rpc/executions/{id}"] = OpenAPIPath{
		"get": OpenAPIOperation{
			Summary:     "Get execution status",
			Description: "Get the status of an RPC execution",
			OperationID: "rpc_execution_get",
			Tags:        []string{"RPC"},
			Security: []map[string][]string{
				{"bearerAuth": {}},
			},
			Parameters: []OpenAPIParameter{
				{Name: "id", In: "path", Required: true, Schema: map[string]string{"type": "string", "format": "uuid"}},
			},
			Responses: map[string]OpenAPIResponse{
				"200": {
					Description: "Execution details",
					Content: map[string]OpenAPIMedia{
						"application/json": {
							Schema: map[string]string{"$ref": "#/components/schemas/RPCExecution"},
						},
					},
				},
			},
		},
	}

	// GET /api/v1/rpc/executions/:id/logs
	spec.Paths["/api/v1/rpc/executions/{id}/logs"] = OpenAPIPath{
		"get": OpenAPIOperation{
			Summary:     "Get execution logs",
			Description: "Get logs for an RPC execution",
			OperationID: "rpc_execution_logs",
			Tags:        []string{"RPC"},
			Security: []map[string][]string{
				{"bearerAuth": {}},
			},
			Parameters: []OpenAPIParameter{
				{Name: "id", In: "path", Required: true, Schema: map[string]string{"type": "string", "format": "uuid"}},
			},
			Responses: map[string]OpenAPIResponse{
				"200": {
					Description: "Execution logs",
					Content: map[string]OpenAPIMedia{
						"application/json": {
							Schema: map[string]interface{}{
								"type": "array",
								"items": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"timestamp": map[string]string{"type": "string", "format": "date-time"},
										"level":     map[string]string{"type": "string"},
										"message":   map[string]string{"type": "string"},
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

// addAIEndpoints adds AI/chatbot endpoints to the spec
func (h *OpenAPIHandler) addAIEndpoints(spec *OpenAPISpec) {
	// AI schemas
	spec.Components.Schemas["Chatbot"] = map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"id":            map[string]string{"type": "string", "format": "uuid"},
			"name":          map[string]string{"type": "string"},
			"description":   map[string]string{"type": "string"},
			"system_prompt": map[string]string{"type": "string"},
			"model":         map[string]string{"type": "string"},
			"is_public":     map[string]string{"type": "boolean"},
			"created_at":    map[string]string{"type": "string", "format": "date-time"},
		},
	}

	spec.Components.Schemas["Conversation"] = map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"id":         map[string]string{"type": "string", "format": "uuid"},
			"chatbot_id": map[string]string{"type": "string", "format": "uuid"},
			"user_id":    map[string]string{"type": "string", "format": "uuid"},
			"title":      map[string]string{"type": "string"},
			"created_at": map[string]string{"type": "string", "format": "date-time"},
			"updated_at": map[string]string{"type": "string", "format": "date-time"},
		},
	}

	// GET /ai/ws - WebSocket for AI chat
	spec.Paths["/ai/ws"] = OpenAPIPath{
		"get": OpenAPIOperation{
			Summary:     "AI Chat WebSocket",
			Description: "Establish a WebSocket connection for AI chat. Upgrade to WebSocket protocol required.",
			OperationID: "ai_chat_ws",
			Tags:        []string{"AI"},
			Responses: map[string]OpenAPIResponse{
				"101": {
					Description: "Switching Protocols - WebSocket connection established",
				},
			},
		},
	}

	// GET /api/v1/ai/chatbots
	spec.Paths["/api/v1/ai/chatbots"] = OpenAPIPath{
		"get": OpenAPIOperation{
			Summary:     "List chatbots",
			Description: "Get all public chatbots",
			OperationID: "ai_chatbots_list",
			Tags:        []string{"AI"},
			Responses: map[string]OpenAPIResponse{
				"200": {
					Description: "List of chatbots",
					Content: map[string]OpenAPIMedia{
						"application/json": {
							Schema: map[string]interface{}{
								"type":  "array",
								"items": map[string]string{"$ref": "#/components/schemas/Chatbot"},
							},
						},
					},
				},
			},
		},
	}

	// GET /api/v1/ai/chatbots/:id
	spec.Paths["/api/v1/ai/chatbots/{id}"] = OpenAPIPath{
		"get": OpenAPIOperation{
			Summary:     "Get chatbot",
			Description: "Get chatbot details",
			OperationID: "ai_chatbot_get",
			Tags:        []string{"AI"},
			Parameters: []OpenAPIParameter{
				{Name: "id", In: "path", Required: true, Schema: map[string]string{"type": "string", "format": "uuid"}},
			},
			Responses: map[string]OpenAPIResponse{
				"200": {
					Description: "Chatbot details",
					Content: map[string]OpenAPIMedia{
						"application/json": {
							Schema: map[string]string{"$ref": "#/components/schemas/Chatbot"},
						},
					},
				},
			},
		},
	}

	// GET /api/v1/ai/conversations
	spec.Paths["/api/v1/ai/conversations"] = OpenAPIPath{
		"get": OpenAPIOperation{
			Summary:     "List conversations",
			Description: "Get all conversations for the authenticated user",
			OperationID: "ai_conversations_list",
			Tags:        []string{"AI"},
			Security: []map[string][]string{
				{"bearerAuth": {}},
			},
			Parameters: []OpenAPIParameter{
				{Name: "chatbot_id", In: "query", Description: "Filter by chatbot", Schema: map[string]string{"type": "string", "format": "uuid"}},
				{Name: "limit", In: "query", Schema: map[string]string{"type": "integer"}},
				{Name: "offset", In: "query", Schema: map[string]string{"type": "integer"}},
			},
			Responses: map[string]OpenAPIResponse{
				"200": {
					Description: "List of conversations",
					Content: map[string]OpenAPIMedia{
						"application/json": {
							Schema: map[string]interface{}{
								"type":  "array",
								"items": map[string]string{"$ref": "#/components/schemas/Conversation"},
							},
						},
					},
				},
			},
		},
	}

	// GET/PATCH/DELETE /api/v1/ai/conversations/:id
	spec.Paths["/api/v1/ai/conversations/{id}"] = OpenAPIPath{
		"get": OpenAPIOperation{
			Summary:     "Get conversation",
			Description: "Get conversation details",
			OperationID: "ai_conversation_get",
			Tags:        []string{"AI"},
			Security: []map[string][]string{
				{"bearerAuth": {}},
			},
			Parameters: []OpenAPIParameter{
				{Name: "id", In: "path", Required: true, Schema: map[string]string{"type": "string", "format": "uuid"}},
			},
			Responses: map[string]OpenAPIResponse{
				"200": {
					Description: "Conversation details",
					Content: map[string]OpenAPIMedia{
						"application/json": {
							Schema: map[string]string{"$ref": "#/components/schemas/Conversation"},
						},
					},
				},
			},
		},
		"patch": OpenAPIOperation{
			Summary:     "Update conversation",
			Description: "Update conversation (e.g., title)",
			OperationID: "ai_conversation_update",
			Tags:        []string{"AI"},
			Security: []map[string][]string{
				{"bearerAuth": {}},
			},
			Parameters: []OpenAPIParameter{
				{Name: "id", In: "path", Required: true, Schema: map[string]string{"type": "string", "format": "uuid"}},
			},
			RequestBody: &OpenAPIRequestBody{
				Required: true,
				Content: map[string]OpenAPIMedia{
					"application/json": {
						Schema: map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"title": map[string]string{"type": "string"},
							},
						},
					},
				},
			},
			Responses: map[string]OpenAPIResponse{
				"200": {
					Description: "Conversation updated",
					Content: map[string]OpenAPIMedia{
						"application/json": {
							Schema: map[string]string{"$ref": "#/components/schemas/Conversation"},
						},
					},
				},
			},
		},
		"delete": OpenAPIOperation{
			Summary:     "Delete conversation",
			Description: "Delete a conversation",
			OperationID: "ai_conversation_delete",
			Tags:        []string{"AI"},
			Security: []map[string][]string{
				{"bearerAuth": {}},
			},
			Parameters: []OpenAPIParameter{
				{Name: "id", In: "path", Required: true, Schema: map[string]string{"type": "string", "format": "uuid"}},
			},
			Responses: map[string]OpenAPIResponse{
				"204": {
					Description: "Conversation deleted",
				},
			},
		},
	}
}

// addAdminEndpoints adds admin management endpoints to the spec
func (h *OpenAPIHandler) addAdminEndpoints(spec *OpenAPISpec) {
	// Admin user schema
	spec.Components.Schemas["AdminUser"] = map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"id":         map[string]string{"type": "string", "format": "uuid"},
			"email":      map[string]string{"type": "string", "format": "email"},
			"role":       map[string]string{"type": "string"},
			"created_at": map[string]string{"type": "string", "format": "date-time"},
			"updated_at": map[string]string{"type": "string", "format": "date-time"},
		},
	}

	// OAuth Provider schema
	spec.Components.Schemas["OAuthProvider"] = map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"id":            map[string]string{"type": "string"},
			"name":          map[string]string{"type": "string"},
			"client_id":     map[string]string{"type": "string"},
			"enabled":       map[string]string{"type": "boolean"},
			"allowed_roles": map[string]interface{}{"type": "array", "items": map[string]string{"type": "string"}},
		},
	}

	// Email Template schema
	spec.Components.Schemas["EmailTemplate"] = map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"type":       map[string]string{"type": "string"},
			"subject":    map[string]string{"type": "string"},
			"html_body":  map[string]string{"type": "string"},
			"text_body":  map[string]string{"type": "string"},
			"updated_at": map[string]string{"type": "string", "format": "date-time"},
		},
	}

	// Invitation schema
	spec.Components.Schemas["Invitation"] = map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"token":      map[string]string{"type": "string"},
			"email":      map[string]string{"type": "string", "format": "email"},
			"role":       map[string]string{"type": "string"},
			"expires_at": map[string]string{"type": "string", "format": "date-time"},
			"created_at": map[string]string{"type": "string", "format": "date-time"},
		},
	}

	// Session schema
	spec.Components.Schemas["Session"] = map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"id":         map[string]string{"type": "string", "format": "uuid"},
			"user_id":    map[string]string{"type": "string", "format": "uuid"},
			"user_agent": map[string]string{"type": "string"},
			"ip_address": map[string]string{"type": "string"},
			"created_at": map[string]string{"type": "string", "format": "date-time"},
			"expires_at": map[string]string{"type": "string", "format": "date-time"},
		},
	}

	// Extension schema
	spec.Components.Schemas["Extension"] = map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"name":        map[string]string{"type": "string"},
			"version":     map[string]string{"type": "string"},
			"description": map[string]string{"type": "string"},
			"enabled":     map[string]string{"type": "boolean"},
		},
	}

	// Admin User Management
	spec.Paths["/api/v1/admin/users"] = OpenAPIPath{
		"get": OpenAPIOperation{
			Summary:     "List users",
			Description: "Get all users (admin only)",
			OperationID: "admin_users_list",
			Tags:        []string{"Admin"},
			Security: []map[string][]string{
				{"bearerAuth": {}},
			},
			Parameters: []OpenAPIParameter{
				{Name: "search", In: "query", Description: "Search by email", Schema: map[string]string{"type": "string"}},
				{Name: "role", In: "query", Description: "Filter by role", Schema: map[string]string{"type": "string"}},
				{Name: "limit", In: "query", Schema: map[string]string{"type": "integer"}},
				{Name: "offset", In: "query", Schema: map[string]string{"type": "integer"}},
			},
			Responses: map[string]OpenAPIResponse{
				"200": {
					Description: "List of users",
					Content: map[string]OpenAPIMedia{
						"application/json": {
							Schema: map[string]interface{}{
								"type":  "array",
								"items": map[string]string{"$ref": "#/components/schemas/AdminUser"},
							},
						},
					},
				},
			},
		},
	}

	spec.Paths["/api/v1/admin/users/{id}"] = OpenAPIPath{
		"delete": OpenAPIOperation{
			Summary:     "Delete user",
			Description: "Delete a user (admin only)",
			OperationID: "admin_users_delete",
			Tags:        []string{"Admin"},
			Security: []map[string][]string{
				{"bearerAuth": {}},
			},
			Parameters: []OpenAPIParameter{
				{Name: "id", In: "path", Required: true, Schema: map[string]string{"type": "string", "format": "uuid"}},
			},
			Responses: map[string]OpenAPIResponse{
				"204": {
					Description: "User deleted",
				},
			},
		},
	}

	spec.Paths["/api/v1/admin/users/{id}/role"] = OpenAPIPath{
		"patch": OpenAPIOperation{
			Summary:     "Update user role",
			Description: "Update a user's role (admin only)",
			OperationID: "admin_users_update_role",
			Tags:        []string{"Admin"},
			Security: []map[string][]string{
				{"bearerAuth": {}},
			},
			Parameters: []OpenAPIParameter{
				{Name: "id", In: "path", Required: true, Schema: map[string]string{"type": "string", "format": "uuid"}},
			},
			RequestBody: &OpenAPIRequestBody{
				Required: true,
				Content: map[string]OpenAPIMedia{
					"application/json": {
						Schema: map[string]interface{}{
							"type":     "object",
							"required": []string{"role"},
							"properties": map[string]interface{}{
								"role": map[string]string{"type": "string"},
							},
						},
					},
				},
			},
			Responses: map[string]OpenAPIResponse{
				"200": {
					Description: "User role updated",
					Content: map[string]OpenAPIMedia{
						"application/json": {
							Schema: map[string]string{"$ref": "#/components/schemas/AdminUser"},
						},
					},
				},
			},
		},
	}

	// Invitations
	spec.Paths["/api/v1/admin/invitations"] = OpenAPIPath{
		"get": OpenAPIOperation{
			Summary:     "List invitations",
			Description: "Get all pending invitations",
			OperationID: "admin_invitations_list",
			Tags:        []string{"Admin"},
			Security: []map[string][]string{
				{"bearerAuth": {}},
			},
			Responses: map[string]OpenAPIResponse{
				"200": {
					Description: "List of invitations",
					Content: map[string]OpenAPIMedia{
						"application/json": {
							Schema: map[string]interface{}{
								"type":  "array",
								"items": map[string]string{"$ref": "#/components/schemas/Invitation"},
							},
						},
					},
				},
			},
		},
		"post": OpenAPIOperation{
			Summary:     "Create invitation",
			Description: "Create a new user invitation",
			OperationID: "admin_invitations_create",
			Tags:        []string{"Admin"},
			Security: []map[string][]string{
				{"bearerAuth": {}},
			},
			RequestBody: &OpenAPIRequestBody{
				Required: true,
				Content: map[string]OpenAPIMedia{
					"application/json": {
						Schema: map[string]interface{}{
							"type":     "object",
							"required": []string{"email"},
							"properties": map[string]interface{}{
								"email": map[string]string{"type": "string", "format": "email"},
								"role":  map[string]string{"type": "string"},
							},
						},
					},
				},
			},
			Responses: map[string]OpenAPIResponse{
				"201": {
					Description: "Invitation created",
					Content: map[string]OpenAPIMedia{
						"application/json": {
							Schema: map[string]string{"$ref": "#/components/schemas/Invitation"},
						},
					},
				},
			},
		},
	}

	spec.Paths["/api/v1/admin/invitations/{token}"] = OpenAPIPath{
		"delete": OpenAPIOperation{
			Summary:     "Revoke invitation",
			Description: "Revoke an invitation",
			OperationID: "admin_invitations_revoke",
			Tags:        []string{"Admin"},
			Security: []map[string][]string{
				{"bearerAuth": {}},
			},
			Parameters: []OpenAPIParameter{
				{Name: "token", In: "path", Required: true, Schema: map[string]string{"type": "string"}},
			},
			Responses: map[string]OpenAPIResponse{
				"204": {
					Description: "Invitation revoked",
				},
			},
		},
	}

	// OAuth Providers
	spec.Paths["/api/v1/admin/oauth/providers"] = OpenAPIPath{
		"get": OpenAPIOperation{
			Summary:     "List OAuth providers",
			Description: "Get all OAuth providers",
			OperationID: "admin_oauth_providers_list",
			Tags:        []string{"Admin"},
			Security: []map[string][]string{
				{"bearerAuth": {}},
			},
			Responses: map[string]OpenAPIResponse{
				"200": {
					Description: "List of OAuth providers",
					Content: map[string]OpenAPIMedia{
						"application/json": {
							Schema: map[string]interface{}{
								"type":  "array",
								"items": map[string]string{"$ref": "#/components/schemas/OAuthProvider"},
							},
						},
					},
				},
			},
		},
		"post": OpenAPIOperation{
			Summary:     "Create OAuth provider",
			Description: "Create a new OAuth provider",
			OperationID: "admin_oauth_providers_create",
			Tags:        []string{"Admin"},
			Security: []map[string][]string{
				{"bearerAuth": {}},
			},
			RequestBody: &OpenAPIRequestBody{
				Required: true,
				Content: map[string]OpenAPIMedia{
					"application/json": {
						Schema: map[string]interface{}{
							"type":     "object",
							"required": []string{"id", "name", "client_id", "client_secret"},
							"properties": map[string]interface{}{
								"id":            map[string]string{"type": "string"},
								"name":          map[string]string{"type": "string"},
								"client_id":     map[string]string{"type": "string"},
								"client_secret": map[string]string{"type": "string"},
								"enabled":       map[string]string{"type": "boolean"},
							},
						},
					},
				},
			},
			Responses: map[string]OpenAPIResponse{
				"201": {
					Description: "Provider created",
					Content: map[string]OpenAPIMedia{
						"application/json": {
							Schema: map[string]string{"$ref": "#/components/schemas/OAuthProvider"},
						},
					},
				},
			},
		},
	}

	spec.Paths["/api/v1/admin/oauth/providers/{id}"] = OpenAPIPath{
		"get": OpenAPIOperation{
			Summary:     "Get OAuth provider",
			Description: "Get OAuth provider details",
			OperationID: "admin_oauth_providers_get",
			Tags:        []string{"Admin"},
			Security: []map[string][]string{
				{"bearerAuth": {}},
			},
			Parameters: []OpenAPIParameter{
				{Name: "id", In: "path", Required: true, Schema: map[string]string{"type": "string"}},
			},
			Responses: map[string]OpenAPIResponse{
				"200": {
					Description: "Provider details",
					Content: map[string]OpenAPIMedia{
						"application/json": {
							Schema: map[string]string{"$ref": "#/components/schemas/OAuthProvider"},
						},
					},
				},
			},
		},
		"put": OpenAPIOperation{
			Summary:     "Update OAuth provider",
			Description: "Update an OAuth provider",
			OperationID: "admin_oauth_providers_update",
			Tags:        []string{"Admin"},
			Security: []map[string][]string{
				{"bearerAuth": {}},
			},
			Parameters: []OpenAPIParameter{
				{Name: "id", In: "path", Required: true, Schema: map[string]string{"type": "string"}},
			},
			RequestBody: &OpenAPIRequestBody{
				Required: true,
				Content: map[string]OpenAPIMedia{
					"application/json": {
						Schema: map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"name":          map[string]string{"type": "string"},
								"client_id":     map[string]string{"type": "string"},
								"client_secret": map[string]string{"type": "string"},
								"enabled":       map[string]string{"type": "boolean"},
							},
						},
					},
				},
			},
			Responses: map[string]OpenAPIResponse{
				"200": {
					Description: "Provider updated",
					Content: map[string]OpenAPIMedia{
						"application/json": {
							Schema: map[string]string{"$ref": "#/components/schemas/OAuthProvider"},
						},
					},
				},
			},
		},
		"delete": OpenAPIOperation{
			Summary:     "Delete OAuth provider",
			Description: "Delete an OAuth provider",
			OperationID: "admin_oauth_providers_delete",
			Tags:        []string{"Admin"},
			Security: []map[string][]string{
				{"bearerAuth": {}},
			},
			Parameters: []OpenAPIParameter{
				{Name: "id", In: "path", Required: true, Schema: map[string]string{"type": "string"}},
			},
			Responses: map[string]OpenAPIResponse{
				"204": {
					Description: "Provider deleted",
				},
			},
		},
	}

	// Sessions
	spec.Paths["/api/v1/admin/auth/sessions"] = OpenAPIPath{
		"get": OpenAPIOperation{
			Summary:     "List sessions",
			Description: "Get all active sessions",
			OperationID: "admin_sessions_list",
			Tags:        []string{"Admin"},
			Security: []map[string][]string{
				{"bearerAuth": {}},
			},
			Parameters: []OpenAPIParameter{
				{Name: "user_id", In: "query", Description: "Filter by user", Schema: map[string]string{"type": "string", "format": "uuid"}},
			},
			Responses: map[string]OpenAPIResponse{
				"200": {
					Description: "List of sessions",
					Content: map[string]OpenAPIMedia{
						"application/json": {
							Schema: map[string]interface{}{
								"type":  "array",
								"items": map[string]string{"$ref": "#/components/schemas/Session"},
							},
						},
					},
				},
			},
		},
	}

	spec.Paths["/api/v1/admin/auth/sessions/{id}"] = OpenAPIPath{
		"delete": OpenAPIOperation{
			Summary:     "Revoke session",
			Description: "Revoke a specific session",
			OperationID: "admin_sessions_revoke",
			Tags:        []string{"Admin"},
			Security: []map[string][]string{
				{"bearerAuth": {}},
			},
			Parameters: []OpenAPIParameter{
				{Name: "id", In: "path", Required: true, Schema: map[string]string{"type": "string", "format": "uuid"}},
			},
			Responses: map[string]OpenAPIResponse{
				"204": {
					Description: "Session revoked",
				},
			},
		},
	}

	// Email Templates
	spec.Paths["/api/v1/admin/email/templates"] = OpenAPIPath{
		"get": OpenAPIOperation{
			Summary:     "List email templates",
			Description: "Get all email templates",
			OperationID: "admin_email_templates_list",
			Tags:        []string{"Admin"},
			Security: []map[string][]string{
				{"bearerAuth": {}},
			},
			Responses: map[string]OpenAPIResponse{
				"200": {
					Description: "List of templates",
					Content: map[string]OpenAPIMedia{
						"application/json": {
							Schema: map[string]interface{}{
								"type":  "array",
								"items": map[string]string{"$ref": "#/components/schemas/EmailTemplate"},
							},
						},
					},
				},
			},
		},
	}

	spec.Paths["/api/v1/admin/email/templates/{type}"] = OpenAPIPath{
		"get": OpenAPIOperation{
			Summary:     "Get email template",
			Description: "Get a specific email template",
			OperationID: "admin_email_templates_get",
			Tags:        []string{"Admin"},
			Security: []map[string][]string{
				{"bearerAuth": {}},
			},
			Parameters: []OpenAPIParameter{
				{Name: "type", In: "path", Required: true, Description: "Template type (e.g., welcome, password_reset)", Schema: map[string]string{"type": "string"}},
			},
			Responses: map[string]OpenAPIResponse{
				"200": {
					Description: "Template details",
					Content: map[string]OpenAPIMedia{
						"application/json": {
							Schema: map[string]string{"$ref": "#/components/schemas/EmailTemplate"},
						},
					},
				},
			},
		},
		"put": OpenAPIOperation{
			Summary:     "Update email template",
			Description: "Update an email template",
			OperationID: "admin_email_templates_update",
			Tags:        []string{"Admin"},
			Security: []map[string][]string{
				{"bearerAuth": {}},
			},
			Parameters: []OpenAPIParameter{
				{Name: "type", In: "path", Required: true, Schema: map[string]string{"type": "string"}},
			},
			RequestBody: &OpenAPIRequestBody{
				Required: true,
				Content: map[string]OpenAPIMedia{
					"application/json": {
						Schema: map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"subject":   map[string]string{"type": "string"},
								"html_body": map[string]string{"type": "string"},
								"text_body": map[string]string{"type": "string"},
							},
						},
					},
				},
			},
			Responses: map[string]OpenAPIResponse{
				"200": {
					Description: "Template updated",
					Content: map[string]OpenAPIMedia{
						"application/json": {
							Schema: map[string]string{"$ref": "#/components/schemas/EmailTemplate"},
						},
					},
				},
			},
		},
	}

	spec.Paths["/api/v1/admin/email/templates/{type}/reset"] = OpenAPIPath{
		"post": OpenAPIOperation{
			Summary:     "Reset email template",
			Description: "Reset an email template to default",
			OperationID: "admin_email_templates_reset",
			Tags:        []string{"Admin"},
			Security: []map[string][]string{
				{"bearerAuth": {}},
			},
			Parameters: []OpenAPIParameter{
				{Name: "type", In: "path", Required: true, Schema: map[string]string{"type": "string"}},
			},
			Responses: map[string]OpenAPIResponse{
				"200": {
					Description: "Template reset to default",
					Content: map[string]OpenAPIMedia{
						"application/json": {
							Schema: map[string]string{"$ref": "#/components/schemas/EmailTemplate"},
						},
					},
				},
			},
		},
	}

	// Extensions
	spec.Paths["/api/v1/admin/extensions"] = OpenAPIPath{
		"get": OpenAPIOperation{
			Summary:     "List extensions",
			Description: "Get all database extensions",
			OperationID: "admin_extensions_list",
			Tags:        []string{"Admin"},
			Security: []map[string][]string{
				{"bearerAuth": {}},
			},
			Responses: map[string]OpenAPIResponse{
				"200": {
					Description: "List of extensions",
					Content: map[string]OpenAPIMedia{
						"application/json": {
							Schema: map[string]interface{}{
								"type":  "array",
								"items": map[string]string{"$ref": "#/components/schemas/Extension"},
							},
						},
					},
				},
			},
		},
	}

	spec.Paths["/api/v1/admin/extensions/{name}/enable"] = OpenAPIPath{
		"post": OpenAPIOperation{
			Summary:     "Enable extension",
			Description: "Enable a database extension",
			OperationID: "admin_extensions_enable",
			Tags:        []string{"Admin"},
			Security: []map[string][]string{
				{"bearerAuth": {}},
			},
			Parameters: []OpenAPIParameter{
				{Name: "name", In: "path", Required: true, Schema: map[string]string{"type": "string"}},
			},
			Responses: map[string]OpenAPIResponse{
				"200": {
					Description: "Extension enabled",
					Content: map[string]OpenAPIMedia{
						"application/json": {
							Schema: map[string]string{"$ref": "#/components/schemas/Extension"},
						},
					},
				},
			},
		},
	}

	spec.Paths["/api/v1/admin/extensions/{name}/disable"] = OpenAPIPath{
		"post": OpenAPIOperation{
			Summary:     "Disable extension",
			Description: "Disable a database extension",
			OperationID: "admin_extensions_disable",
			Tags:        []string{"Admin"},
			Security: []map[string][]string{
				{"bearerAuth": {}},
			},
			Parameters: []OpenAPIParameter{
				{Name: "name", In: "path", Required: true, Schema: map[string]string{"type": "string"}},
			},
			Responses: map[string]OpenAPIResponse{
				"200": {
					Description: "Extension disabled",
					Content: map[string]OpenAPIMedia{
						"application/json": {
							Schema: map[string]string{"$ref": "#/components/schemas/Extension"},
						},
					},
				},
			},
		},
	}

	// SQL Editor
	spec.Paths["/api/v1/admin/sql/execute"] = OpenAPIPath{
		"post": OpenAPIOperation{
			Summary:     "Execute SQL",
			Description: "Execute SQL query (dashboard admin only)",
			OperationID: "admin_sql_execute",
			Tags:        []string{"Admin"},
			Security: []map[string][]string{
				{"bearerAuth": {}},
			},
			RequestBody: &OpenAPIRequestBody{
				Required: true,
				Content: map[string]OpenAPIMedia{
					"application/json": {
						Schema: map[string]interface{}{
							"type":     "object",
							"required": []string{"query"},
							"properties": map[string]interface{}{
								"query": map[string]string{"type": "string"},
							},
						},
					},
				},
			},
			Responses: map[string]OpenAPIResponse{
				"200": {
					Description: "Query results",
					Content: map[string]OpenAPIMedia{
						"application/json": {
							Schema: map[string]interface{}{
								"type": "object",
								"properties": map[string]interface{}{
									"rows":          map[string]interface{}{"type": "array", "items": map[string]string{"type": "object"}},
									"columns":       map[string]interface{}{"type": "array", "items": map[string]string{"type": "string"}},
									"rows_affected": map[string]string{"type": "integer"},
								},
							},
						},
					},
				},
			},
		},
	}

	// Schema Management
	spec.Paths["/api/v1/admin/schemas"] = OpenAPIPath{
		"get": OpenAPIOperation{
			Summary:     "List schemas",
			Description: "Get all database schemas",
			OperationID: "admin_schemas_list",
			Tags:        []string{"Admin"},
			Security: []map[string][]string{
				{"bearerAuth": {}},
			},
			Responses: map[string]OpenAPIResponse{
				"200": {
					Description: "List of schemas",
					Content: map[string]OpenAPIMedia{
						"application/json": {
							Schema: map[string]interface{}{
								"type":  "array",
								"items": map[string]string{"type": "string"},
							},
						},
					},
				},
			},
		},
		"post": OpenAPIOperation{
			Summary:     "Create schema",
			Description: "Create a new database schema",
			OperationID: "admin_schemas_create",
			Tags:        []string{"Admin"},
			Security: []map[string][]string{
				{"bearerAuth": {}},
			},
			RequestBody: &OpenAPIRequestBody{
				Required: true,
				Content: map[string]OpenAPIMedia{
					"application/json": {
						Schema: map[string]interface{}{
							"type":     "object",
							"required": []string{"name"},
							"properties": map[string]interface{}{
								"name": map[string]string{"type": "string"},
							},
						},
					},
				},
			},
			Responses: map[string]OpenAPIResponse{
				"201": {
					Description: "Schema created",
				},
			},
		},
	}

	// Tables Management
	spec.Paths["/api/v1/admin/tables"] = OpenAPIPath{
		"get": OpenAPIOperation{
			Summary:     "List all tables",
			Description: "Get all database tables with metadata",
			OperationID: "admin_tables_list",
			Tags:        []string{"Admin"},
			Security: []map[string][]string{
				{"bearerAuth": {}},
			},
			Responses: map[string]OpenAPIResponse{
				"200": {
					Description: "List of tables",
					Content: map[string]OpenAPIMedia{
						"application/json": {
							Schema: map[string]interface{}{
								"type": "array",
								"items": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"schema":      map[string]string{"type": "string"},
										"name":        map[string]string{"type": "string"},
										"type":        map[string]string{"type": "string"},
										"row_count":   map[string]string{"type": "integer"},
										"size_bytes":  map[string]string{"type": "integer"},
										"description": map[string]string{"type": "string"},
									},
								},
							},
						},
					},
				},
			},
		},
		"post": OpenAPIOperation{
			Summary:     "Create table",
			Description: "Create a new database table",
			OperationID: "admin_tables_create",
			Tags:        []string{"Admin"},
			Security: []map[string][]string{
				{"bearerAuth": {}},
			},
			RequestBody: &OpenAPIRequestBody{
				Required: true,
				Content: map[string]OpenAPIMedia{
					"application/json": {
						Schema: map[string]interface{}{
							"type":     "object",
							"required": []string{"name", "columns"},
							"properties": map[string]interface{}{
								"schema": map[string]string{"type": "string"},
								"name":   map[string]string{"type": "string"},
								"columns": map[string]interface{}{
									"type": "array",
									"items": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"name":        map[string]string{"type": "string"},
											"type":        map[string]string{"type": "string"},
											"nullable":    map[string]string{"type": "boolean"},
											"default":     map[string]string{"type": "string"},
											"primary_key": map[string]string{"type": "boolean"},
											"unique":      map[string]string{"type": "boolean"},
											"references":  map[string]string{"type": "string"},
										},
									},
								},
							},
						},
					},
				},
			},
			Responses: map[string]OpenAPIResponse{
				"201": {
					Description: "Table created",
				},
			},
		},
	}

	spec.Paths["/api/v1/admin/tables/{schema}/{table}"] = OpenAPIPath{
		"get": OpenAPIOperation{
			Summary:     "Get table schema",
			Description: "Get detailed table schema information",
			OperationID: "admin_tables_get",
			Tags:        []string{"Admin"},
			Security: []map[string][]string{
				{"bearerAuth": {}},
			},
			Parameters: []OpenAPIParameter{
				{Name: "schema", In: "path", Required: true, Schema: map[string]string{"type": "string"}},
				{Name: "table", In: "path", Required: true, Schema: map[string]string{"type": "string"}},
			},
			Responses: map[string]OpenAPIResponse{
				"200": {
					Description: "Table schema details",
					Content: map[string]OpenAPIMedia{
						"application/json": {
							Schema: map[string]interface{}{
								"type": "object",
								"properties": map[string]interface{}{
									"schema":      map[string]string{"type": "string"},
									"name":        map[string]string{"type": "string"},
									"columns":     map[string]interface{}{"type": "array", "items": map[string]string{"type": "object"}},
									"indexes":     map[string]interface{}{"type": "array", "items": map[string]string{"type": "object"}},
									"constraints": map[string]interface{}{"type": "array", "items": map[string]string{"type": "object"}},
								},
							},
						},
					},
				},
			},
		},
		"delete": OpenAPIOperation{
			Summary:     "Delete table",
			Description: "Delete a database table",
			OperationID: "admin_tables_delete",
			Tags:        []string{"Admin"},
			Security: []map[string][]string{
				{"bearerAuth": {}},
			},
			Parameters: []OpenAPIParameter{
				{Name: "schema", In: "path", Required: true, Schema: map[string]string{"type": "string"}},
				{Name: "table", In: "path", Required: true, Schema: map[string]string{"type": "string"}},
			},
			Responses: map[string]OpenAPIResponse{
				"204": {
					Description: "Table deleted",
				},
			},
		},
		"patch": OpenAPIOperation{
			Summary:     "Rename table",
			Description: "Rename a database table",
			OperationID: "admin_tables_rename",
			Tags:        []string{"Admin"},
			Security: []map[string][]string{
				{"bearerAuth": {}},
			},
			Parameters: []OpenAPIParameter{
				{Name: "schema", In: "path", Required: true, Schema: map[string]string{"type": "string"}},
				{Name: "table", In: "path", Required: true, Schema: map[string]string{"type": "string"}},
			},
			RequestBody: &OpenAPIRequestBody{
				Required: true,
				Content: map[string]OpenAPIMedia{
					"application/json": {
						Schema: map[string]interface{}{
							"type":     "object",
							"required": []string{"new_name"},
							"properties": map[string]interface{}{
								"new_name": map[string]string{"type": "string"},
							},
						},
					},
				},
			},
			Responses: map[string]OpenAPIResponse{
				"200": {
					Description: "Table renamed",
				},
			},
		},
	}

	// Schema refresh
	spec.Paths["/api/v1/admin/schema/refresh"] = OpenAPIPath{
		"post": OpenAPIOperation{
			Summary:     "Refresh schema cache",
			Description: "Refresh the schema cache",
			OperationID: "admin_schema_refresh",
			Tags:        []string{"Admin"},
			Security: []map[string][]string{
				{"bearerAuth": {}},
			},
			Responses: map[string]OpenAPIResponse{
				"200": {
					Description: "Schema cache refreshed",
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
		"get":    h.generateListOperation(tableName, schemaName, schemaRef),
		"post":   h.generateCreateOperation(tableName, schemaName, schemaRef),
		"patch":  h.generateBatchUpdateOperation(tableName, schemaName, schemaRef),
		"delete": h.generateBatchDeleteOperation(tableName, schemaName, schemaRef),
	}

	spec.Paths[pathWithID] = OpenAPIPath{
		"get":    h.generateGetOperation(tableName, schemaName, schemaRef),
		"put":    h.generateReplaceOperation(tableName, schemaName, schemaRef),
		"patch":  h.generateUpdateOperation(tableName, schemaName, schemaRef),
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
		return "/api/v1/tables/" + table.Schema + "/" + tableName
	}
	return "/api/v1/tables/" + tableName
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
		Summary:     fmt.Sprintf("List %s.%s records", schemaName, tableName),
		Description: "Query and filter records with PostgREST-compatible syntax",
		OperationID: fmt.Sprintf("list_%s_%s", schemaName, tableName),
		Tags:        []string{"Tables"},
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
		Summary:     fmt.Sprintf("Create %s.%s record(s)", schemaName, tableName),
		Description: "Create a single record or batch insert multiple records",
		OperationID: fmt.Sprintf("create_%s_%s", schemaName, tableName),
		Tags:        []string{"Tables"},
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
		Summary:     fmt.Sprintf("Get %s.%s by ID", schemaName, tableName),
		OperationID: fmt.Sprintf("get_%s_%s", schemaName, tableName),
		Tags:        []string{"Tables"},
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
		Summary:     fmt.Sprintf("Update %s.%s", schemaName, tableName),
		OperationID: fmt.Sprintf("update_%s_%s", schemaName, tableName),
		Tags:        []string{"Tables"},
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
		Summary:     fmt.Sprintf("Replace %s.%s", schemaName, tableName),
		OperationID: fmt.Sprintf("replace_%s_%s", schemaName, tableName),
		Tags:        []string{"Tables"},
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
		Summary:     fmt.Sprintf("Delete %s.%s", schemaName, tableName),
		OperationID: fmt.Sprintf("delete_%s_%s", schemaName, tableName),
		Tags:        []string{"Tables"},
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
		Summary:     fmt.Sprintf("Batch update %s.%s records", schemaName, tableName),
		Description: "Update multiple records matching the filter criteria",
		OperationID: fmt.Sprintf("batch_update_%s_%s", schemaName, tableName),
		Tags:        []string{"Tables"},
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
		Summary:     fmt.Sprintf("Batch delete %s.%s records", schemaName, tableName),
		Description: "Delete multiple records matching the filter criteria (requires at least one filter)",
		OperationID: fmt.Sprintf("batch_delete_%s_%s", schemaName, tableName),
		Tags:        []string{"Tables"},
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
