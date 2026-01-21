package custom

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Common errors
var (
	ErrToolNotFound       = errors.New("custom tool not found")
	ErrResourceNotFound   = errors.New("custom resource not found")
	ErrToolAlreadyExists  = errors.New("custom tool with this name already exists in namespace")
	ErrResourceExists     = errors.New("custom resource with this URI already exists in namespace")
	ErrInvalidInputSchema = errors.New("invalid input schema: must be a valid JSON Schema object")
)

// Storage handles database operations for custom MCP tools and resources.
type Storage struct {
	db *pgxpool.Pool
}

// NewStorage creates a new Storage instance.
func NewStorage(db *pgxpool.Pool) *Storage {
	return &Storage{db: db}
}

// Tool Operations

// CreateTool creates a new custom tool.
func (s *Storage) CreateTool(ctx context.Context, req *CreateToolRequest, createdBy *uuid.UUID) (*CustomTool, error) {
	// Set defaults
	namespace := req.Namespace
	if namespace == "" {
		namespace = "default"
	}

	inputSchema := req.InputSchema
	if inputSchema == nil {
		inputSchema = map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		}
	}

	requiredScopes := req.RequiredScopes
	if requiredScopes == nil {
		requiredScopes = []string{}
	}

	timeoutSeconds := req.TimeoutSeconds
	if timeoutSeconds <= 0 {
		timeoutSeconds = 30
	}

	memoryLimitMB := req.MemoryLimitMB
	if memoryLimitMB <= 0 {
		memoryLimitMB = 128
	}

	allowNet := true
	if req.AllowNet != nil {
		allowNet = *req.AllowNet
	}

	allowEnv := false
	if req.AllowEnv != nil {
		allowEnv = *req.AllowEnv
	}

	allowRead := false
	if req.AllowRead != nil {
		allowRead = *req.AllowRead
	}

	allowWrite := false
	if req.AllowWrite != nil {
		allowWrite = *req.AllowWrite
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	inputSchemaJSON, err := json.Marshal(inputSchema)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal input schema: %w", err)
	}

	tool := &CustomTool{}
	err = s.db.QueryRow(ctx, `
		INSERT INTO mcp.custom_tools (
			name, namespace, description, code, input_schema,
			required_scopes, timeout_seconds, memory_limit_mb,
			allow_net, allow_env, allow_read, allow_write,
			enabled, created_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		RETURNING id, name, namespace, description, code, input_schema,
			required_scopes, timeout_seconds, memory_limit_mb,
			allow_net, allow_env, allow_read, allow_write,
			enabled, version, created_by, created_at, updated_at
	`,
		req.Name, namespace, req.Description, req.Code, inputSchemaJSON,
		requiredScopes, timeoutSeconds, memoryLimitMB,
		allowNet, allowEnv, allowRead, allowWrite,
		enabled, createdBy,
	).Scan(
		&tool.ID, &tool.Name, &tool.Namespace, &tool.Description, &tool.Code, &tool.InputSchema,
		&tool.RequiredScopes, &tool.TimeoutSeconds, &tool.MemoryLimitMB,
		&tool.AllowNet, &tool.AllowEnv, &tool.AllowRead, &tool.AllowWrite,
		&tool.Enabled, &tool.Version, &tool.CreatedBy, &tool.CreatedAt, &tool.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrToolAlreadyExists
		}
		return nil, fmt.Errorf("failed to create custom tool: %w", err)
	}

	return tool, nil
}

// GetTool retrieves a custom tool by ID.
func (s *Storage) GetTool(ctx context.Context, id uuid.UUID) (*CustomTool, error) {
	tool := &CustomTool{}
	err := s.db.QueryRow(ctx, `
		SELECT id, name, namespace, description, code, input_schema,
			required_scopes, timeout_seconds, memory_limit_mb,
			allow_net, allow_env, allow_read, allow_write,
			enabled, version, created_by, created_at, updated_at
		FROM mcp.custom_tools
		WHERE id = $1
	`, id).Scan(
		&tool.ID, &tool.Name, &tool.Namespace, &tool.Description, &tool.Code, &tool.InputSchema,
		&tool.RequiredScopes, &tool.TimeoutSeconds, &tool.MemoryLimitMB,
		&tool.AllowNet, &tool.AllowEnv, &tool.AllowRead, &tool.AllowWrite,
		&tool.Enabled, &tool.Version, &tool.CreatedBy, &tool.CreatedAt, &tool.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrToolNotFound
		}
		return nil, fmt.Errorf("failed to get custom tool: %w", err)
	}

	return tool, nil
}

// GetToolByName retrieves a custom tool by name and namespace.
func (s *Storage) GetToolByName(ctx context.Context, name, namespace string) (*CustomTool, error) {
	if namespace == "" {
		namespace = "default"
	}

	tool := &CustomTool{}
	err := s.db.QueryRow(ctx, `
		SELECT id, name, namespace, description, code, input_schema,
			required_scopes, timeout_seconds, memory_limit_mb,
			allow_net, allow_env, allow_read, allow_write,
			enabled, version, created_by, created_at, updated_at
		FROM mcp.custom_tools
		WHERE name = $1 AND namespace = $2
	`, name, namespace).Scan(
		&tool.ID, &tool.Name, &tool.Namespace, &tool.Description, &tool.Code, &tool.InputSchema,
		&tool.RequiredScopes, &tool.TimeoutSeconds, &tool.MemoryLimitMB,
		&tool.AllowNet, &tool.AllowEnv, &tool.AllowRead, &tool.AllowWrite,
		&tool.Enabled, &tool.Version, &tool.CreatedBy, &tool.CreatedAt, &tool.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrToolNotFound
		}
		return nil, fmt.Errorf("failed to get custom tool by name: %w", err)
	}

	return tool, nil
}

// ListTools retrieves custom tools with optional filtering.
func (s *Storage) ListTools(ctx context.Context, filter ListToolsFilter) ([]*CustomTool, error) {
	query := `
		SELECT id, name, namespace, description, code, input_schema,
			required_scopes, timeout_seconds, memory_limit_mb,
			allow_net, allow_env, allow_read, allow_write,
			enabled, version, created_by, created_at, updated_at
		FROM mcp.custom_tools
		WHERE 1=1
	`
	args := []any{}
	argNum := 1

	if filter.Namespace != "" {
		query += fmt.Sprintf(" AND namespace = $%d", argNum)
		args = append(args, filter.Namespace)
		argNum++
	}

	if filter.EnabledOnly {
		query += " AND enabled = true"
	}

	query += " ORDER BY name ASC"

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argNum)
		args = append(args, filter.Limit)
		argNum++
	}

	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argNum)
		args = append(args, filter.Offset)
	}

	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list custom tools: %w", err)
	}
	defer rows.Close()

	var tools []*CustomTool
	for rows.Next() {
		tool := &CustomTool{}
		err := rows.Scan(
			&tool.ID, &tool.Name, &tool.Namespace, &tool.Description, &tool.Code, &tool.InputSchema,
			&tool.RequiredScopes, &tool.TimeoutSeconds, &tool.MemoryLimitMB,
			&tool.AllowNet, &tool.AllowEnv, &tool.AllowRead, &tool.AllowWrite,
			&tool.Enabled, &tool.Version, &tool.CreatedBy, &tool.CreatedAt, &tool.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan custom tool: %w", err)
		}
		tools = append(tools, tool)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating custom tools: %w", err)
	}

	return tools, nil
}

// UpdateTool updates an existing custom tool.
func (s *Storage) UpdateTool(ctx context.Context, id uuid.UUID, req *UpdateToolRequest) (*CustomTool, error) {
	// Get existing tool first
	existing, err := s.GetTool(ctx, id)
	if err != nil {
		return nil, err
	}

	// Apply updates
	if req.Name != nil {
		existing.Name = *req.Name
	}
	if req.Description != nil {
		existing.Description = *req.Description
	}
	if req.Code != nil {
		existing.Code = *req.Code
	}
	if req.InputSchema != nil {
		existing.InputSchema = req.InputSchema
	}
	if req.RequiredScopes != nil {
		existing.RequiredScopes = req.RequiredScopes
	}
	if req.TimeoutSeconds != nil {
		existing.TimeoutSeconds = *req.TimeoutSeconds
	}
	if req.MemoryLimitMB != nil {
		existing.MemoryLimitMB = *req.MemoryLimitMB
	}
	if req.AllowNet != nil {
		existing.AllowNet = *req.AllowNet
	}
	if req.AllowEnv != nil {
		existing.AllowEnv = *req.AllowEnv
	}
	if req.AllowRead != nil {
		existing.AllowRead = *req.AllowRead
	}
	if req.AllowWrite != nil {
		existing.AllowWrite = *req.AllowWrite
	}
	if req.Enabled != nil {
		existing.Enabled = *req.Enabled
	}

	inputSchemaJSON, err := json.Marshal(existing.InputSchema)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal input schema: %w", err)
	}

	tool := &CustomTool{}
	err = s.db.QueryRow(ctx, `
		UPDATE mcp.custom_tools SET
			name = $2,
			description = $3,
			code = $4,
			input_schema = $5,
			required_scopes = $6,
			timeout_seconds = $7,
			memory_limit_mb = $8,
			allow_net = $9,
			allow_env = $10,
			allow_read = $11,
			allow_write = $12,
			enabled = $13,
			version = version + 1
		WHERE id = $1
		RETURNING id, name, namespace, description, code, input_schema,
			required_scopes, timeout_seconds, memory_limit_mb,
			allow_net, allow_env, allow_read, allow_write,
			enabled, version, created_by, created_at, updated_at
	`,
		id, existing.Name, existing.Description, existing.Code, inputSchemaJSON,
		existing.RequiredScopes, existing.TimeoutSeconds, existing.MemoryLimitMB,
		existing.AllowNet, existing.AllowEnv, existing.AllowRead, existing.AllowWrite,
		existing.Enabled,
	).Scan(
		&tool.ID, &tool.Name, &tool.Namespace, &tool.Description, &tool.Code, &tool.InputSchema,
		&tool.RequiredScopes, &tool.TimeoutSeconds, &tool.MemoryLimitMB,
		&tool.AllowNet, &tool.AllowEnv, &tool.AllowRead, &tool.AllowWrite,
		&tool.Enabled, &tool.Version, &tool.CreatedBy, &tool.CreatedAt, &tool.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrToolAlreadyExists
		}
		return nil, fmt.Errorf("failed to update custom tool: %w", err)
	}

	return tool, nil
}

// DeleteTool deletes a custom tool by ID.
func (s *Storage) DeleteTool(ctx context.Context, id uuid.UUID) error {
	result, err := s.db.Exec(ctx, `DELETE FROM mcp.custom_tools WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to delete custom tool: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrToolNotFound
	}

	return nil
}

// SyncTool creates or updates a tool by name (upsert).
func (s *Storage) SyncTool(ctx context.Context, req *SyncToolRequest, createdBy *uuid.UUID) (*CustomTool, error) {
	namespace := req.Namespace
	if namespace == "" {
		namespace = "default"
	}

	// Check if tool exists
	existing, err := s.GetToolByName(ctx, req.Name, namespace)
	if err != nil && !errors.Is(err, ErrToolNotFound) {
		return nil, err
	}

	if existing != nil {
		if !req.Upsert {
			return nil, ErrToolAlreadyExists
		}
		// Update existing tool
		updateReq := &UpdateToolRequest{
			Description:    &req.Description,
			Code:           &req.Code,
			InputSchema:    req.InputSchema,
			RequiredScopes: req.RequiredScopes,
			TimeoutSeconds: &req.TimeoutSeconds,
			MemoryLimitMB:  &req.MemoryLimitMB,
			AllowNet:       req.AllowNet,
			AllowEnv:       req.AllowEnv,
			AllowRead:      req.AllowRead,
			AllowWrite:     req.AllowWrite,
			Enabled:        req.Enabled,
		}
		return s.UpdateTool(ctx, existing.ID, updateReq)
	}

	// Create new tool
	return s.CreateTool(ctx, &req.CreateToolRequest, createdBy)
}

// Resource Operations

// CreateResource creates a new custom resource.
func (s *Storage) CreateResource(ctx context.Context, req *CreateResourceRequest, createdBy *uuid.UUID) (*CustomResource, error) {
	// Set defaults
	namespace := req.Namespace
	if namespace == "" {
		namespace = "default"
	}

	mimeType := req.MimeType
	if mimeType == "" {
		mimeType = "application/json"
	}

	requiredScopes := req.RequiredScopes
	if requiredScopes == nil {
		requiredScopes = []string{}
	}

	timeoutSeconds := 10
	if req.TimeoutSeconds != nil && *req.TimeoutSeconds > 0 {
		timeoutSeconds = *req.TimeoutSeconds
	}

	cacheTTLSeconds := 60
	if req.CacheTTLSeconds != nil && *req.CacheTTLSeconds >= 0 {
		cacheTTLSeconds = *req.CacheTTLSeconds
	}

	isTemplate := false
	if req.IsTemplate != nil {
		isTemplate = *req.IsTemplate
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	resource := &CustomResource{}
	err := s.db.QueryRow(ctx, `
		INSERT INTO mcp.custom_resources (
			uri, name, namespace, description, mime_type,
			code, is_template, required_scopes,
			timeout_seconds, cache_ttl_seconds, enabled, created_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, uri, name, namespace, description, mime_type,
			code, is_template, required_scopes,
			timeout_seconds, cache_ttl_seconds, enabled, version,
			created_by, created_at, updated_at
	`,
		req.URI, req.Name, namespace, req.Description, mimeType,
		req.Code, isTemplate, requiredScopes,
		timeoutSeconds, cacheTTLSeconds, enabled, createdBy,
	).Scan(
		&resource.ID, &resource.URI, &resource.Name, &resource.Namespace, &resource.Description, &resource.MimeType,
		&resource.Code, &resource.IsTemplate, &resource.RequiredScopes,
		&resource.TimeoutSeconds, &resource.CacheTTLSeconds, &resource.Enabled, &resource.Version,
		&resource.CreatedBy, &resource.CreatedAt, &resource.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrResourceExists
		}
		return nil, fmt.Errorf("failed to create custom resource: %w", err)
	}

	return resource, nil
}

// GetResource retrieves a custom resource by ID.
func (s *Storage) GetResource(ctx context.Context, id uuid.UUID) (*CustomResource, error) {
	resource := &CustomResource{}
	err := s.db.QueryRow(ctx, `
		SELECT id, uri, name, namespace, description, mime_type,
			code, is_template, required_scopes,
			timeout_seconds, cache_ttl_seconds, enabled, version,
			created_by, created_at, updated_at
		FROM mcp.custom_resources
		WHERE id = $1
	`, id).Scan(
		&resource.ID, &resource.URI, &resource.Name, &resource.Namespace, &resource.Description, &resource.MimeType,
		&resource.Code, &resource.IsTemplate, &resource.RequiredScopes,
		&resource.TimeoutSeconds, &resource.CacheTTLSeconds, &resource.Enabled, &resource.Version,
		&resource.CreatedBy, &resource.CreatedAt, &resource.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrResourceNotFound
		}
		return nil, fmt.Errorf("failed to get custom resource: %w", err)
	}

	return resource, nil
}

// GetResourceByURI retrieves a custom resource by URI and namespace.
func (s *Storage) GetResourceByURI(ctx context.Context, uri, namespace string) (*CustomResource, error) {
	if namespace == "" {
		namespace = "default"
	}

	resource := &CustomResource{}
	err := s.db.QueryRow(ctx, `
		SELECT id, uri, name, namespace, description, mime_type,
			code, is_template, required_scopes,
			timeout_seconds, cache_ttl_seconds, enabled, version,
			created_by, created_at, updated_at
		FROM mcp.custom_resources
		WHERE uri = $1 AND namespace = $2
	`, uri, namespace).Scan(
		&resource.ID, &resource.URI, &resource.Name, &resource.Namespace, &resource.Description, &resource.MimeType,
		&resource.Code, &resource.IsTemplate, &resource.RequiredScopes,
		&resource.TimeoutSeconds, &resource.CacheTTLSeconds, &resource.Enabled, &resource.Version,
		&resource.CreatedBy, &resource.CreatedAt, &resource.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrResourceNotFound
		}
		return nil, fmt.Errorf("failed to get custom resource by URI: %w", err)
	}

	return resource, nil
}

// ListResources retrieves custom resources with optional filtering.
func (s *Storage) ListResources(ctx context.Context, filter ListResourcesFilter) ([]*CustomResource, error) {
	query := `
		SELECT id, uri, name, namespace, description, mime_type,
			code, is_template, required_scopes,
			timeout_seconds, cache_ttl_seconds, enabled, version,
			created_by, created_at, updated_at
		FROM mcp.custom_resources
		WHERE 1=1
	`
	args := []any{}
	argNum := 1

	if filter.Namespace != "" {
		query += fmt.Sprintf(" AND namespace = $%d", argNum)
		args = append(args, filter.Namespace)
		argNum++
	}

	if filter.EnabledOnly {
		query += " AND enabled = true"
	}

	query += " ORDER BY uri ASC"

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argNum)
		args = append(args, filter.Limit)
		argNum++
	}

	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argNum)
		args = append(args, filter.Offset)
	}

	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list custom resources: %w", err)
	}
	defer rows.Close()

	var resources []*CustomResource
	for rows.Next() {
		resource := &CustomResource{}
		err := rows.Scan(
			&resource.ID, &resource.URI, &resource.Name, &resource.Namespace, &resource.Description, &resource.MimeType,
			&resource.Code, &resource.IsTemplate, &resource.RequiredScopes,
			&resource.TimeoutSeconds, &resource.CacheTTLSeconds, &resource.Enabled, &resource.Version,
			&resource.CreatedBy, &resource.CreatedAt, &resource.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan custom resource: %w", err)
		}
		resources = append(resources, resource)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating custom resources: %w", err)
	}

	return resources, nil
}

// UpdateResource updates an existing custom resource.
func (s *Storage) UpdateResource(ctx context.Context, id uuid.UUID, req *UpdateResourceRequest) (*CustomResource, error) {
	// Get existing resource first
	existing, err := s.GetResource(ctx, id)
	if err != nil {
		return nil, err
	}

	// Apply updates
	if req.URI != nil {
		existing.URI = *req.URI
	}
	if req.Name != nil {
		existing.Name = *req.Name
	}
	if req.Description != nil {
		existing.Description = *req.Description
	}
	if req.MimeType != nil {
		existing.MimeType = *req.MimeType
	}
	if req.Code != nil {
		existing.Code = *req.Code
	}
	if req.IsTemplate != nil {
		existing.IsTemplate = *req.IsTemplate
	}
	if req.RequiredScopes != nil {
		existing.RequiredScopes = req.RequiredScopes
	}
	if req.TimeoutSeconds != nil {
		existing.TimeoutSeconds = *req.TimeoutSeconds
	}
	if req.CacheTTLSeconds != nil {
		existing.CacheTTLSeconds = *req.CacheTTLSeconds
	}
	if req.Enabled != nil {
		existing.Enabled = *req.Enabled
	}

	resource := &CustomResource{}
	err = s.db.QueryRow(ctx, `
		UPDATE mcp.custom_resources SET
			uri = $2,
			name = $3,
			description = $4,
			mime_type = $5,
			code = $6,
			is_template = $7,
			required_scopes = $8,
			timeout_seconds = $9,
			cache_ttl_seconds = $10,
			enabled = $11,
			version = version + 1
		WHERE id = $1
		RETURNING id, uri, name, namespace, description, mime_type,
			code, is_template, required_scopes,
			timeout_seconds, cache_ttl_seconds, enabled, version,
			created_by, created_at, updated_at
	`,
		id, existing.URI, existing.Name, existing.Description, existing.MimeType,
		existing.Code, existing.IsTemplate, existing.RequiredScopes,
		existing.TimeoutSeconds, existing.CacheTTLSeconds, existing.Enabled,
	).Scan(
		&resource.ID, &resource.URI, &resource.Name, &resource.Namespace, &resource.Description, &resource.MimeType,
		&resource.Code, &resource.IsTemplate, &resource.RequiredScopes,
		&resource.TimeoutSeconds, &resource.CacheTTLSeconds, &resource.Enabled, &resource.Version,
		&resource.CreatedBy, &resource.CreatedAt, &resource.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrResourceExists
		}
		return nil, fmt.Errorf("failed to update custom resource: %w", err)
	}

	return resource, nil
}

// DeleteResource deletes a custom resource by ID.
func (s *Storage) DeleteResource(ctx context.Context, id uuid.UUID) error {
	result, err := s.db.Exec(ctx, `DELETE FROM mcp.custom_resources WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to delete custom resource: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrResourceNotFound
	}

	return nil
}

// SyncResource creates or updates a resource by URI (upsert).
func (s *Storage) SyncResource(ctx context.Context, req *SyncResourceRequest, createdBy *uuid.UUID) (*CustomResource, error) {
	namespace := req.Namespace
	if namespace == "" {
		namespace = "default"
	}

	// Check if resource exists
	existing, err := s.GetResourceByURI(ctx, req.URI, namespace)
	if err != nil && !errors.Is(err, ErrResourceNotFound) {
		return nil, err
	}

	if existing != nil {
		if !req.Upsert {
			return nil, ErrResourceExists
		}
		// Update existing resource
		updateReq := &UpdateResourceRequest{
			Name:            &req.Name,
			Description:     &req.Description,
			MimeType:        &req.MimeType,
			Code:            &req.Code,
			IsTemplate:      req.IsTemplate,
			RequiredScopes:  req.RequiredScopes,
			TimeoutSeconds:  req.TimeoutSeconds,
			CacheTTLSeconds: req.CacheTTLSeconds,
			Enabled:         req.Enabled,
		}
		return s.UpdateResource(ctx, existing.ID, updateReq)
	}

	// Create new resource
	return s.CreateResource(ctx, &req.CreateResourceRequest, createdBy)
}

// Helper functions

func isUniqueViolation(err error) bool {
	return err != nil && (contains(err.Error(), "duplicate key") ||
		contains(err.Error(), "unique constraint") ||
		contains(err.Error(), "23505"))
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr, 0))
}

func containsAt(s, substr string, start int) bool {
	for i := start; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
