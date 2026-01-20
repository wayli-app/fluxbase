package custom

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDynamicToolHandler_Name(t *testing.T) {
	tool := &CustomTool{
		ID:        uuid.New(),
		Name:      "weather_forecast",
		Namespace: "default",
		Code:      "export function handler() {}",
	}

	handler := NewDynamicToolHandler(tool, nil)

	// Tool names are prefixed with "custom_"
	assert.Equal(t, "custom_weather_forecast", handler.Name())
}

func TestDynamicToolHandler_Description(t *testing.T) {
	tests := []struct {
		name        string
		description string
		want        string
	}{
		{
			name:        "with description",
			description: "Get weather forecast",
			want:        "Get weather forecast",
		},
		{
			name:        "without description",
			description: "",
			want:        "Custom tool: test_tool",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tool := &CustomTool{
				ID:          uuid.New(),
				Name:        "test_tool",
				Namespace:   "default",
				Description: tt.description,
				Code:        "export function handler() {}",
			}

			handler := NewDynamicToolHandler(tool, nil)
			assert.Equal(t, tt.want, handler.Description())
		})
	}
}

func TestDynamicToolHandler_InputSchema(t *testing.T) {
	tests := []struct {
		name        string
		inputSchema map[string]any
		wantType    string
	}{
		{
			name: "with custom schema",
			inputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"location": map[string]any{"type": "string"},
				},
			},
			wantType: "object",
		},
		{
			name:        "default schema",
			inputSchema: nil,
			wantType:    "object",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tool := &CustomTool{
				ID:          uuid.New(),
				Name:        "test_tool",
				Namespace:   "default",
				Code:        "export function handler() {}",
				InputSchema: tt.inputSchema,
			}

			handler := NewDynamicToolHandler(tool, nil)
			schema := handler.InputSchema()

			require.NotNil(t, schema)
			assert.Equal(t, tt.wantType, schema["type"])
		})
	}
}

func TestDynamicToolHandler_RequiredScopes(t *testing.T) {
	tests := []struct {
		name           string
		requiredScopes []string
		wantContains   []string
	}{
		{
			name:           "no additional scopes",
			requiredScopes: nil,
			wantContains:   []string{"execute:custom"},
		},
		{
			name:           "with additional scopes",
			requiredScopes: []string{"read:tables", "write:storage"},
			wantContains:   []string{"execute:custom", "read:tables", "write:storage"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tool := &CustomTool{
				ID:             uuid.New(),
				Name:           "test_tool",
				Namespace:      "default",
				Code:           "export function handler() {}",
				RequiredScopes: tt.requiredScopes,
			}

			handler := NewDynamicToolHandler(tool, nil)
			scopes := handler.RequiredScopes()

			for _, want := range tt.wantContains {
				assert.Contains(t, scopes, want)
			}
		})
	}
}

func TestDynamicResourceProvider_URI(t *testing.T) {
	resource := &CustomResource{
		ID:        uuid.New(),
		URI:       "fluxbase://custom/analytics",
		Name:      "Analytics",
		Namespace: "default",
		MimeType:  "application/json",
		Code:      "export function handler() {}",
	}

	provider := NewDynamicResourceProvider(resource, nil)
	assert.Equal(t, "fluxbase://custom/analytics", provider.URI())
}

func TestDynamicResourceProvider_MatchURI(t *testing.T) {
	tests := []struct {
		name          string
		resourceURI   string
		isTemplate    bool
		testURI       string
		wantMatch     bool
		wantParams    map[string]string
	}{
		{
			name:        "exact match non-template",
			resourceURI: "fluxbase://custom/analytics",
			isTemplate:  false,
			testURI:     "fluxbase://custom/analytics",
			wantMatch:   true,
			wantParams:  map[string]string{},
		},
		{
			name:        "no match non-template",
			resourceURI: "fluxbase://custom/analytics",
			isTemplate:  false,
			testURI:     "fluxbase://custom/other",
			wantMatch:   false,
			wantParams:  nil,
		},
		{
			name:        "template with one param",
			resourceURI: "fluxbase://custom/users/{id}",
			isTemplate:  true,
			testURI:     "fluxbase://custom/users/123",
			wantMatch:   true,
			wantParams:  map[string]string{"id": "123"},
		},
		{
			name:        "template with multiple params",
			resourceURI: "fluxbase://custom/users/{userId}/orders/{orderId}",
			isTemplate:  true,
			testURI:     "fluxbase://custom/users/abc/orders/xyz",
			wantMatch:   true,
			wantParams:  map[string]string{"userId": "abc", "orderId": "xyz"},
		},
		{
			name:        "template no match",
			resourceURI: "fluxbase://custom/users/{id}",
			isTemplate:  true,
			testURI:     "fluxbase://custom/products/123",
			wantMatch:   false,
			wantParams:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resource := &CustomResource{
				ID:         uuid.New(),
				URI:        tt.resourceURI,
				Name:       "Test Resource",
				Namespace:  "default",
				Code:       "export function handler() {}",
				IsTemplate: tt.isTemplate,
			}

			provider := NewDynamicResourceProvider(resource, nil)
			params, match := provider.MatchURI(tt.testURI)

			assert.Equal(t, tt.wantMatch, match)
			if tt.wantMatch {
				assert.Equal(t, tt.wantParams, params)
			}
		})
	}
}

func TestDynamicResourceProvider_IsTemplate(t *testing.T) {
	tests := []struct {
		name       string
		isTemplate bool
	}{
		{"is template", true},
		{"not template", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resource := &CustomResource{
				ID:         uuid.New(),
				URI:        "fluxbase://custom/test",
				Name:       "Test",
				Namespace:  "default",
				Code:       "export function handler() {}",
				IsTemplate: tt.isTemplate,
			}

			provider := NewDynamicResourceProvider(resource, nil)
			assert.Equal(t, tt.isTemplate, provider.IsTemplate())
		})
	}
}

func TestDynamicResourceProvider_RequiredScopes(t *testing.T) {
	resource := &CustomResource{
		ID:             uuid.New(),
		URI:            "fluxbase://custom/test",
		Name:           "Test",
		Namespace:      "default",
		Code:           "export function handler() {}",
		RequiredScopes: []string{"read:tables"},
	}

	provider := NewDynamicResourceProvider(resource, nil)
	scopes := provider.RequiredScopes()

	assert.Contains(t, scopes, "read:custom")
	assert.Contains(t, scopes, "read:tables")
}

func TestValidateToolCode(t *testing.T) {
	tests := []struct {
		name    string
		code    string
		wantErr bool
	}{
		{
			name:    "valid handler function",
			code:    "export function handler(args, ctx) { return { content: [] }; }",
			wantErr: false,
		},
		{
			name:    "valid default export",
			code:    "export default async function(args, ctx) { return 'ok'; }",
			wantErr: false,
		},
		{
			name:    "valid async function",
			code:    "export async function handler(args, ctx) { return 'ok'; }",
			wantErr: false,
		},
		{
			name:    "empty code",
			code:    "",
			wantErr: true,
		},
		{
			name:    "whitespace only",
			code:    "   \n\t  ",
			wantErr: true,
		},
		{
			name:    "no handler export",
			code:    "const x = 1; console.log(x);",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateToolCode(tt.code)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateResourceCode(t *testing.T) {
	// Resource code validation uses the same rules as tool code
	tests := []struct {
		name    string
		code    string
		wantErr bool
	}{
		{
			name:    "valid handler",
			code:    "export function handler(params, ctx) { return []; }",
			wantErr: false,
		},
		{
			name:    "empty code",
			code:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateResourceCode(tt.code)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
