package tools

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nimbleflux/fluxbase/internal/ai"
	"github.com/nimbleflux/fluxbase/internal/mcp"
)

// =============================================================================
// QueryKnowledgeGraphTool Tests
// =============================================================================

func TestQueryKnowledgeGraphTool_Name(t *testing.T) {
	tool := NewQueryKnowledgeGraphTool(nil)
	assert.Equal(t, "query_knowledge_graph", tool.Name())
}

func TestQueryKnowledgeGraphTool_Description(t *testing.T) {
	tool := NewQueryKnowledgeGraphTool(nil)
	desc := tool.Description()
	assert.NotEmpty(t, desc)
	assert.Contains(t, desc, "Query")
	assert.Contains(t, desc, "entities")
	assert.Contains(t, desc, "knowledge graph")
}

func TestQueryKnowledgeGraphTool_RequiredScopes(t *testing.T) {
	tool := NewQueryKnowledgeGraphTool(nil)
	scopes := tool.RequiredScopes()
	require.Len(t, scopes, 1)
	assert.Equal(t, mcp.ScopeReadVectors, scopes[0])
}

func TestQueryKnowledgeGraphTool_InputSchema(t *testing.T) {
	tool := NewQueryKnowledgeGraphTool(nil)
	schema := tool.InputSchema()

	t.Run("has object type", func(t *testing.T) {
		assert.Equal(t, "object", schema["type"])
	})

	t.Run("has knowledge_base_id property", func(t *testing.T) {
		props, ok := schema["properties"].(map[string]any)
		require.True(t, ok)

		prop, ok := props["knowledge_base_id"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "string", prop["type"])
		assert.NotEmpty(t, prop["description"])
	})

	t.Run("has entity_type property", func(t *testing.T) {
		props, ok := schema["properties"].(map[string]any)
		require.True(t, ok)

		prop, ok := props["entity_type"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "string", prop["type"])
		assert.Contains(t, prop["description"], "entity type")
	})

	t.Run("has search_query property", func(t *testing.T) {
		props, ok := schema["properties"].(map[string]any)
		require.True(t, ok)

		prop, ok := props["search_query"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "string", prop["type"])
	})

	t.Run("has limit property with defaults", func(t *testing.T) {
		props, ok := schema["properties"].(map[string]any)
		require.True(t, ok)

		prop, ok := props["limit"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "integer", prop["type"])
		assert.Equal(t, 50, prop["default"])
		assert.Equal(t, 200, prop["maximum"])
	})

	t.Run("has include_relationships property", func(t *testing.T) {
		props, ok := schema["properties"].(map[string]any)
		require.True(t, ok)

		prop, ok := props["include_relationships"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "boolean", prop["type"])
		assert.Equal(t, true, prop["default"])
	})

	t.Run("requires knowledge_base_id", func(t *testing.T) {
		required, ok := schema["required"].([]string)
		require.True(t, ok)
		require.Len(t, required, 1)
		assert.Contains(t, required, "knowledge_base_id")
	})
}

func TestNewQueryKnowledgeGraphTool(t *testing.T) {
	tool := NewQueryKnowledgeGraphTool(nil)
	require.NotNil(t, tool)
	assert.Nil(t, tool.knowledgeGraph)
}

func TestQueryKnowledgeGraphTool_Execute_NilKnowledgeGraph(t *testing.T) {
	tool := NewQueryKnowledgeGraphTool(nil)
	result, err := tool.Execute(context.Background(), map[string]any{
		"knowledge_base_id": "kb-123",
	}, nil)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.IsError)
	require.Len(t, result.Content, 1)
	assert.Contains(t, result.Content[0].Text, "Knowledge graph is not configured")
}

func TestQueryKnowledgeGraphTool_Execute_MissingKnowledgeBaseID(t *testing.T) {
	tool := NewQueryKnowledgeGraphTool(&ai.KnowledgeGraph{})

	tests := []struct {
		name string
		args map[string]any
	}{
		{"missing key", map[string]any{}},
		{"empty string", map[string]any{"knowledge_base_id": ""}},
		{"non-string value", map[string]any{"knowledge_base_id": 123}},
		{"nil value", map[string]any{"knowledge_base_id": nil}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tool.Execute(context.Background(), tt.args, nil)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "knowledge_base_id is required")
		})
	}
}

// =============================================================================
// FindRelatedEntitiesTool Tests
// =============================================================================

func TestFindRelatedEntitiesTool_Name(t *testing.T) {
	tool := NewFindRelatedEntitiesTool(nil)
	assert.Equal(t, "find_related_entities", tool.Name())
}

func TestFindRelatedEntitiesTool_Description(t *testing.T) {
	tool := NewFindRelatedEntitiesTool(nil)
	desc := tool.Description()
	assert.NotEmpty(t, desc)
	assert.Contains(t, desc, "Find")
	assert.Contains(t, desc, "related")
	assert.Contains(t, desc, "entities")
}

func TestFindRelatedEntitiesTool_RequiredScopes(t *testing.T) {
	tool := NewFindRelatedEntitiesTool(nil)
	scopes := tool.RequiredScopes()
	require.Len(t, scopes, 1)
	assert.Equal(t, mcp.ScopeReadVectors, scopes[0])
}

func TestFindRelatedEntitiesTool_InputSchema(t *testing.T) {
	tool := NewFindRelatedEntitiesTool(nil)
	schema := tool.InputSchema()

	t.Run("has object type", func(t *testing.T) {
		assert.Equal(t, "object", schema["type"])
	})

	t.Run("has knowledge_base_id property", func(t *testing.T) {
		props, ok := schema["properties"].(map[string]any)
		require.True(t, ok)

		prop, ok := props["knowledge_base_id"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "string", prop["type"])
	})

	t.Run("has entity_id property", func(t *testing.T) {
		props, ok := schema["properties"].(map[string]any)
		require.True(t, ok)

		prop, ok := props["entity_id"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "string", prop["type"])
	})

	t.Run("has max_depth property with constraints", func(t *testing.T) {
		props, ok := schema["properties"].(map[string]any)
		require.True(t, ok)

		prop, ok := props["max_depth"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "integer", prop["type"])
		assert.Equal(t, 2, prop["default"])
		assert.Equal(t, 1, prop["minimum"])
		assert.Equal(t, 5, prop["maximum"])
	})

	t.Run("has relationship_types property", func(t *testing.T) {
		props, ok := schema["properties"].(map[string]any)
		require.True(t, ok)

		prop, ok := props["relationship_types"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "array", prop["type"])

		items, ok := prop["items"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "string", items["type"])
	})

	t.Run("has limit property with defaults", func(t *testing.T) {
		props, ok := schema["properties"].(map[string]any)
		require.True(t, ok)

		prop, ok := props["limit"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "integer", prop["type"])
		assert.Equal(t, 100, prop["default"])
		assert.Equal(t, 500, prop["maximum"])
	})

	t.Run("requires knowledge_base_id and entity_id", func(t *testing.T) {
		required, ok := schema["required"].([]string)
		require.True(t, ok)
		require.Len(t, required, 2)
		assert.Contains(t, required, "knowledge_base_id")
		assert.Contains(t, required, "entity_id")
	})
}

func TestNewFindRelatedEntitiesTool(t *testing.T) {
	tool := NewFindRelatedEntitiesTool(nil)
	require.NotNil(t, tool)
	assert.Nil(t, tool.knowledgeGraph)
}

func TestFindRelatedEntitiesTool_Execute_NilKnowledgeGraph(t *testing.T) {
	tool := NewFindRelatedEntitiesTool(nil)
	result, err := tool.Execute(context.Background(), map[string]any{
		"knowledge_base_id": "kb-123",
		"entity_id":         "entity-456",
	}, nil)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.IsError)
	require.Len(t, result.Content, 1)
	assert.Contains(t, result.Content[0].Text, "Knowledge graph is not configured")
}

func TestFindRelatedEntitiesTool_Execute_MissingRequiredParams(t *testing.T) {
	tool := NewFindRelatedEntitiesTool(&ai.KnowledgeGraph{})

	t.Run("missing knowledge_base_id", func(t *testing.T) {
		_, err := tool.Execute(context.Background(), map[string]any{
			"entity_id": "entity-456",
		}, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "knowledge_base_id is required")
	})

	t.Run("empty knowledge_base_id", func(t *testing.T) {
		_, err := tool.Execute(context.Background(), map[string]any{
			"knowledge_base_id": "",
			"entity_id":         "entity-456",
		}, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "knowledge_base_id is required")
	})

	t.Run("non-string knowledge_base_id", func(t *testing.T) {
		_, err := tool.Execute(context.Background(), map[string]any{
			"knowledge_base_id": 42,
			"entity_id":         "entity-456",
		}, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "knowledge_base_id is required")
	})

	t.Run("missing entity_id", func(t *testing.T) {
		_, err := tool.Execute(context.Background(), map[string]any{
			"knowledge_base_id": "kb-123",
		}, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "entity_id is required")
	})

	t.Run("empty entity_id", func(t *testing.T) {
		_, err := tool.Execute(context.Background(), map[string]any{
			"knowledge_base_id": "kb-123",
			"entity_id":         "",
		}, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "entity_id is required")
	})

	t.Run("non-string entity_id", func(t *testing.T) {
		_, err := tool.Execute(context.Background(), map[string]any{
			"knowledge_base_id": "kb-123",
			"entity_id":         float64(99),
		}, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "entity_id is required")
	})
}

// =============================================================================
// BrowseKnowledgeGraphTool Tests
// =============================================================================

func TestBrowseKnowledgeGraphTool_Name(t *testing.T) {
	tool := NewBrowseKnowledgeGraphTool(nil)
	assert.Equal(t, "browse_knowledge_graph", tool.Name())
}

func TestBrowseKnowledgeGraphTool_Description(t *testing.T) {
	tool := NewBrowseKnowledgeGraphTool(nil)
	desc := tool.Description()
	assert.NotEmpty(t, desc)
	assert.Contains(t, desc, "Browse")
	assert.Contains(t, desc, "knowledge graph")
	assert.Contains(t, desc, "neighborhood")
}

func TestBrowseKnowledgeGraphTool_RequiredScopes(t *testing.T) {
	tool := NewBrowseKnowledgeGraphTool(nil)
	scopes := tool.RequiredScopes()
	require.Len(t, scopes, 1)
	assert.Equal(t, mcp.ScopeReadVectors, scopes[0])
}

func TestBrowseKnowledgeGraphTool_InputSchema(t *testing.T) {
	tool := NewBrowseKnowledgeGraphTool(nil)
	schema := tool.InputSchema()

	t.Run("has object type", func(t *testing.T) {
		assert.Equal(t, "object", schema["type"])
	})

	t.Run("has knowledge_base_id property", func(t *testing.T) {
		props, ok := schema["properties"].(map[string]any)
		require.True(t, ok)

		prop, ok := props["knowledge_base_id"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "string", prop["type"])
	})

	t.Run("has start_entity property", func(t *testing.T) {
		props, ok := schema["properties"].(map[string]any)
		require.True(t, ok)

		prop, ok := props["start_entity"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "string", prop["type"])
		assert.Contains(t, prop["description"], "entity ID or canonical name")
	})

	t.Run("has direction property", func(t *testing.T) {
		props, ok := schema["properties"].(map[string]any)
		require.True(t, ok)

		prop, ok := props["direction"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "string", prop["type"])
		assert.Equal(t, "both", prop["default"])

		enum, ok := prop["enum"].([]string)
		require.True(t, ok)
		assert.Contains(t, enum, "outgoing")
		assert.Contains(t, enum, "incoming")
		assert.Contains(t, enum, "both")
	})

	t.Run("has limit property with defaults", func(t *testing.T) {
		props, ok := schema["properties"].(map[string]any)
		require.True(t, ok)

		prop, ok := props["limit"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "integer", prop["type"])
		assert.Equal(t, 50, prop["default"])
		assert.Equal(t, 200, prop["maximum"])
	})

	t.Run("requires knowledge_base_id and start_entity", func(t *testing.T) {
		required, ok := schema["required"].([]string)
		require.True(t, ok)
		require.Len(t, required, 2)
		assert.Contains(t, required, "knowledge_base_id")
		assert.Contains(t, required, "start_entity")
	})
}

func TestNewBrowseKnowledgeGraphTool(t *testing.T) {
	tool := NewBrowseKnowledgeGraphTool(nil)
	require.NotNil(t, tool)
	assert.Nil(t, tool.knowledgeGraph)
}

func TestBrowseKnowledgeGraphTool_Execute_NilKnowledgeGraph(t *testing.T) {
	tool := NewBrowseKnowledgeGraphTool(nil)
	result, err := tool.Execute(context.Background(), map[string]any{
		"knowledge_base_id": "kb-123",
		"start_entity":      "entity-456",
	}, nil)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.IsError)
	require.Len(t, result.Content, 1)
	assert.Contains(t, result.Content[0].Text, "Knowledge graph is not configured")
}

func TestBrowseKnowledgeGraphTool_Execute_MissingRequiredParams(t *testing.T) {
	tool := NewBrowseKnowledgeGraphTool(&ai.KnowledgeGraph{})

	t.Run("missing knowledge_base_id", func(t *testing.T) {
		_, err := tool.Execute(context.Background(), map[string]any{
			"start_entity": "entity-456",
		}, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "knowledge_base_id is required")
	})

	t.Run("empty knowledge_base_id", func(t *testing.T) {
		_, err := tool.Execute(context.Background(), map[string]any{
			"knowledge_base_id": "",
			"start_entity":      "entity-456",
		}, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "knowledge_base_id is required")
	})

	t.Run("non-string knowledge_base_id", func(t *testing.T) {
		_, err := tool.Execute(context.Background(), map[string]any{
			"knowledge_base_id": true,
			"start_entity":      "entity-456",
		}, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "knowledge_base_id is required")
	})

	t.Run("missing start_entity", func(t *testing.T) {
		_, err := tool.Execute(context.Background(), map[string]any{
			"knowledge_base_id": "kb-123",
		}, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "start_entity is required")
	})

	t.Run("empty start_entity", func(t *testing.T) {
		_, err := tool.Execute(context.Background(), map[string]any{
			"knowledge_base_id": "kb-123",
			"start_entity":      "",
		}, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "start_entity is required")
	})

	t.Run("non-string start_entity", func(t *testing.T) {
		_, err := tool.Execute(context.Background(), map[string]any{
			"knowledge_base_id": "kb-123",
			"start_entity":      42,
		}, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "start_entity is required")
	})
}

// =============================================================================
// ToolHandler Interface Compliance Tests
// =============================================================================

func TestKnowledgeGraphTools_ImplementsToolHandler(t *testing.T) {
	t.Run("QueryKnowledgeGraphTool implements ToolHandler", func(t *testing.T) {
		var _ ToolHandler = &QueryKnowledgeGraphTool{}
	})

	t.Run("FindRelatedEntitiesTool implements ToolHandler", func(t *testing.T) {
		var _ ToolHandler = &FindRelatedEntitiesTool{}
	})

	t.Run("BrowseKnowledgeGraphTool implements ToolHandler", func(t *testing.T) {
		var _ ToolHandler = &BrowseKnowledgeGraphTool{}
	})
}

// =============================================================================
// Struct Field Tests
// =============================================================================

func TestKnowledgeGraphTools_Structs(t *testing.T) {
	t.Run("QueryKnowledgeGraphTool stores knowledgeGraph", func(t *testing.T) {
		tool := &QueryKnowledgeGraphTool{knowledgeGraph: nil}
		assert.Nil(t, tool.knowledgeGraph)
	})

	t.Run("FindRelatedEntitiesTool stores knowledgeGraph", func(t *testing.T) {
		tool := &FindRelatedEntitiesTool{knowledgeGraph: nil}
		assert.Nil(t, tool.knowledgeGraph)
	})

	t.Run("BrowseKnowledgeGraphTool stores knowledgeGraph", func(t *testing.T) {
		tool := &BrowseKnowledgeGraphTool{knowledgeGraph: nil}
		assert.Nil(t, tool.knowledgeGraph)
	})
}

// =============================================================================
// Execute with nil args
// =============================================================================

func TestQueryKnowledgeGraphTool_Execute_NilArgs(t *testing.T) {
	tool := NewQueryKnowledgeGraphTool(&ai.KnowledgeGraph{})
	_, err := tool.Execute(context.Background(), nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "knowledge_base_id is required")
}

func TestFindRelatedEntitiesTool_Execute_NilArgs(t *testing.T) {
	tool := NewFindRelatedEntitiesTool(&ai.KnowledgeGraph{})
	_, err := tool.Execute(context.Background(), nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "knowledge_base_id is required")
}

func TestBrowseKnowledgeGraphTool_Execute_NilArgs(t *testing.T) {
	tool := NewBrowseKnowledgeGraphTool(&ai.KnowledgeGraph{})
	_, err := tool.Execute(context.Background(), nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "knowledge_base_id is required")
}

// =============================================================================
// JSON Response Structure Tests (using mocked KnowledgeGraph)
// =============================================================================

// =============================================================================
// InputSchema Property Count Tests
// =============================================================================

func TestQueryKnowledgeGraphTool_InputSchema_PropertyCount(t *testing.T) {
	tool := NewQueryKnowledgeGraphTool(nil)
	schema := tool.InputSchema()
	props, ok := schema["properties"].(map[string]any)
	require.True(t, ok)
	assert.Len(t, props, 5)
}

func TestFindRelatedEntitiesTool_InputSchema_PropertyCount(t *testing.T) {
	tool := NewFindRelatedEntitiesTool(nil)
	schema := tool.InputSchema()
	props, ok := schema["properties"].(map[string]any)
	require.True(t, ok)
	assert.Len(t, props, 5)
}

func TestBrowseKnowledgeGraphTool_InputSchema_PropertyCount(t *testing.T) {
	tool := NewBrowseKnowledgeGraphTool(nil)
	schema := tool.InputSchema()
	props, ok := schema["properties"].(map[string]any)
	require.True(t, ok)
	assert.Len(t, props, 4)
}

// =============================================================================
// InputSchema Property Descriptions
// =============================================================================

func TestQueryKnowledgeGraphTool_InputSchema_PropertyDescriptions(t *testing.T) {
	tool := NewQueryKnowledgeGraphTool(nil)
	schema := tool.InputSchema()
	props := schema["properties"].(map[string]any)

	testCases := []struct {
		name     string
		propName string
		contains string
	}{
		{"knowledge_base_id description", "knowledge_base_id", "knowledge base"},
		{"entity_type description", "entity_type", "entity type"},
		{"search_query description", "search_query", "search"},
		{"limit description", "limit", "Maximum"},
		{"include_relationships description", "include_relationships", "relationships"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			prop, ok := props[tc.propName].(map[string]any)
			require.True(t, ok, "property %s should exist", tc.propName)
			desc, ok := prop["description"].(string)
			require.True(t, ok, "property %s should have description", tc.propName)
			assert.Contains(t, desc, tc.contains)
		})
	}
}

func TestFindRelatedEntitiesTool_InputSchema_PropertyDescriptions(t *testing.T) {
	tool := NewFindRelatedEntitiesTool(nil)
	schema := tool.InputSchema()
	props := schema["properties"].(map[string]any)

	testCases := []struct {
		name     string
		propName string
		contains string
	}{
		{"knowledge_base_id description", "knowledge_base_id", "knowledge base"},
		{"entity_id description", "entity_id", "entity"},
		{"max_depth description", "max_depth", "depth"},
		{"relationship_types description", "relationship_types", "relationship"},
		{"limit description", "limit", "Maximum"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			prop, ok := props[tc.propName].(map[string]any)
			require.True(t, ok, "property %s should exist", tc.propName)
			desc, ok := prop["description"].(string)
			require.True(t, ok, "property %s should have description", tc.propName)
			assert.Contains(t, desc, tc.contains)
		})
	}
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkQueryKnowledgeGraphTool_InputSchema(b *testing.B) {
	tool := NewQueryKnowledgeGraphTool(nil)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tool.InputSchema()
	}
}

func BenchmarkFindRelatedEntitiesTool_InputSchema(b *testing.B) {
	tool := NewFindRelatedEntitiesTool(nil)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tool.InputSchema()
	}
}

func BenchmarkBrowseKnowledgeGraphTool_InputSchema(b *testing.B) {
	tool := NewBrowseKnowledgeGraphTool(nil)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tool.InputSchema()
	}
}

func BenchmarkQueryKnowledgeGraphTool_Execute_NilGraph(b *testing.B) {
	tool := NewQueryKnowledgeGraphTool(nil)
	args := map[string]any{
		"knowledge_base_id": "kb-123",
	}
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = tool.Execute(ctx, args, nil)
	}
}

func BenchmarkFindRelatedEntitiesTool_Execute_NilGraph(b *testing.B) {
	tool := NewFindRelatedEntitiesTool(nil)
	args := map[string]any{
		"knowledge_base_id": "kb-123",
		"entity_id":         "entity-456",
	}
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = tool.Execute(ctx, args, nil)
	}
}

func BenchmarkBrowseKnowledgeGraphTool_Execute_NilGraph(b *testing.B) {
	tool := NewBrowseKnowledgeGraphTool(nil)
	args := map[string]any{
		"knowledge_base_id": "kb-123",
		"start_entity":      "entity-456",
	}
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = tool.Execute(ctx, args, nil)
	}
}

// =============================================================================
// Edge Cases
// =============================================================================
