package api

import (
	"context"

	"github.com/gofiber/fiber/v3"

	"github.com/nimbleflux/fluxbase/internal/database"
	"github.com/nimbleflux/fluxbase/internal/middleware"
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

func (h *OpenAPIHandler) requireDB(c fiber.Ctx) error {
	if h.db == nil {
		return fiber.NewError(fiber.StatusServiceUnavailable, "Database not initialized")
	}
	return nil
}

// GetOpenAPISpec generates and returns the OpenAPI specification
// Admin users get full spec with database schema; non-admin users get minimal spec
func (h *OpenAPIHandler) GetOpenAPISpec(c fiber.Ctx) error {
	// Check if user has admin role
	role, _ := c.Locals("user_role").(string)
	isAdmin := role == "admin" || role == "instance_admin" || role == "service_role" || role == "tenant_service"

	// Non-admin users get minimal spec without database tables
	if !isAdmin {
		spec := h.generateMinimalSpec(c.BaseURL())
		return c.JSON(spec)
	}

	ctx := context.Background()

	if err := h.requireDB(c); err != nil {
		return err
	}

	if userID := middleware.GetUserID(c); userID != "" {
		ctx = database.ContextWithAuth(ctx, userID, role, isAdmin)
	}

	inspector := database.NewSchemaInspector(h.db)

	// Get all tables (admin only)
	tables, err := inspector.GetAllTables(ctx, "public", "auth")
	if err != nil {
		return SendInternalError(c, "Failed to fetch database schema")
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

	// Add custom MCP tools and resources endpoints
	h.addMCPEndpoints(&spec)

	// Add admin endpoints
	h.addAdminEndpoints(&spec)

	// Generate paths and schemas for each table
	for _, table := range tables {
		h.addTableToSpec(&spec, table)
	}

	return spec
}
