package api

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
			"redirect_url":  map[string]string{"type": "string", "description": "First configured redirect URL (backward compat)"},
			"redirect_urls": map[string]interface{}{"type": "array", "items": map[string]string{"type": "string"}, "description": "Allowed OAuth redirect URLs (allowlist); first entry is the default"},
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
								"redirect_url":  map[string]string{"type": "string"},
								"redirect_urls": map[string]interface{}{"type": "array", "items": map[string]string{"type": "string"}},
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
								"redirect_url":  map[string]string{"type": "string"},
								"redirect_urls": map[string]interface{}{"type": "array", "items": map[string]string{"type": "string"}},
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
