package api

import (
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/wayli-app/fluxbase/internal/database"
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
					"description":  "JWT token obtained from /api/v1/auth/signin or /api/v1/auth/signup",
				},
			},
		},
	}

	// Add authentication endpoints
	h.addAuthEndpoints(&spec)

	// Add storage endpoints
	h.addStorageEndpoints(&spec)

	// Generate paths and schemas for each table
	for _, table := range tables {
		h.addTableToSpec(&spec, table)
	}

	// RPC endpoints are currently not included in the API Explorer
	// In the future, we may add user-defined functions when we have a way to distinguish them
	_ = functions // Suppress unused variable warning

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
