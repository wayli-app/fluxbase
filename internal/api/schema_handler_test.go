package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// SchemaRelationship Struct Tests
// =============================================================================

func TestSchemaRelationship_Struct(t *testing.T) {
	t.Run("all fields accessible", func(t *testing.T) {
		rel := SchemaRelationship{
			ID:             "fk_user_id_1",
			SourceSchema:   "public",
			SourceTable:    "posts",
			SourceColumn:   "user_id",
			TargetSchema:   "public",
			TargetTable:    "users",
			TargetColumn:   "id",
			ConstraintName: "posts_user_id_fkey",
			OnDelete:       "CASCADE",
			OnUpdate:       "NO ACTION",
			Cardinality:    "many-to-one",
		}

		assert.Equal(t, "fk_user_id_1", rel.ID)
		assert.Equal(t, "public", rel.SourceSchema)
		assert.Equal(t, "posts", rel.SourceTable)
		assert.Equal(t, "user_id", rel.SourceColumn)
		assert.Equal(t, "public", rel.TargetSchema)
		assert.Equal(t, "users", rel.TargetTable)
		assert.Equal(t, "id", rel.TargetColumn)
		assert.Equal(t, "posts_user_id_fkey", rel.ConstraintName)
		assert.Equal(t, "CASCADE", rel.OnDelete)
		assert.Equal(t, "NO ACTION", rel.OnUpdate)
		assert.Equal(t, "many-to-one", rel.Cardinality)
	})

	t.Run("JSON serialization", func(t *testing.T) {
		rel := SchemaRelationship{
			ID:             "fk_1",
			SourceSchema:   "public",
			SourceTable:    "orders",
			SourceColumn:   "customer_id",
			TargetSchema:   "public",
			TargetTable:    "customers",
			TargetColumn:   "id",
			ConstraintName: "orders_customer_fk",
			OnDelete:       "RESTRICT",
			OnUpdate:       "CASCADE",
			Cardinality:    "many-to-one",
		}

		data, err := json.Marshal(rel)
		require.NoError(t, err)

		assert.Contains(t, string(data), `"id":"fk_1"`)
		assert.Contains(t, string(data), `"source_schema":"public"`)
		assert.Contains(t, string(data), `"source_table":"orders"`)
		assert.Contains(t, string(data), `"source_column":"customer_id"`)
		assert.Contains(t, string(data), `"target_schema":"public"`)
		assert.Contains(t, string(data), `"target_table":"customers"`)
		assert.Contains(t, string(data), `"target_column":"id"`)
		assert.Contains(t, string(data), `"constraint_name":"orders_customer_fk"`)
		assert.Contains(t, string(data), `"on_delete":"RESTRICT"`)
		assert.Contains(t, string(data), `"on_update":"CASCADE"`)
		assert.Contains(t, string(data), `"cardinality":"many-to-one"`)
	})

	t.Run("JSON deserialization", func(t *testing.T) {
		jsonData := `{
			"id": "fk_test",
			"source_schema": "app",
			"source_table": "items",
			"source_column": "category_id",
			"target_schema": "app",
			"target_table": "categories",
			"target_column": "id",
			"constraint_name": "items_category_fk",
			"on_delete": "SET NULL",
			"on_update": "NO ACTION",
			"cardinality": "many-to-one"
		}`

		var rel SchemaRelationship
		err := json.Unmarshal([]byte(jsonData), &rel)
		require.NoError(t, err)

		assert.Equal(t, "fk_test", rel.ID)
		assert.Equal(t, "app", rel.SourceSchema)
		assert.Equal(t, "items", rel.SourceTable)
		assert.Equal(t, "category_id", rel.SourceColumn)
		assert.Equal(t, "app", rel.TargetSchema)
		assert.Equal(t, "categories", rel.TargetTable)
		assert.Equal(t, "id", rel.TargetColumn)
		assert.Equal(t, "items_category_fk", rel.ConstraintName)
		assert.Equal(t, "SET NULL", rel.OnDelete)
		assert.Equal(t, "NO ACTION", rel.OnUpdate)
		assert.Equal(t, "many-to-one", rel.Cardinality)
	})

	t.Run("cardinality values", func(t *testing.T) {
		validCardinalities := []string{"one-to-one", "one-to-many", "many-to-one"}

		for _, cardinality := range validCardinalities {
			rel := SchemaRelationship{Cardinality: cardinality}
			assert.Contains(t, validCardinalities, rel.Cardinality)
		}
	})
}

// =============================================================================
// SchemaNode Struct Tests
// =============================================================================

func TestSchemaNode_Struct(t *testing.T) {
	t.Run("all fields accessible", func(t *testing.T) {
		rowEstimate := int64(1000)
		comment := "Users table"

		node := SchemaNode{
			Schema: "public",
			Name:   "users",
			Columns: []SchemaNodeColumn{
				{Name: "id", DataType: "uuid", IsPrimaryKey: true},
				{Name: "email", DataType: "text", Nullable: false},
			},
			PrimaryKey:       []string{"id"},
			RLSEnabled:       true,
			ForceRLS:         false,
			RowEstimate:      &rowEstimate,
			Comment:          &comment,
			IncomingRelCount: 5,
			OutgoingRelCount: 2,
		}

		assert.Equal(t, "public", node.Schema)
		assert.Equal(t, "users", node.Name)
		assert.Len(t, node.Columns, 2)
		assert.Equal(t, []string{"id"}, node.PrimaryKey)
		assert.True(t, node.RLSEnabled)
		assert.False(t, node.ForceRLS)
		assert.Equal(t, int64(1000), *node.RowEstimate)
		assert.Equal(t, "Users table", *node.Comment)
		assert.Equal(t, 5, node.IncomingRelCount)
		assert.Equal(t, 2, node.OutgoingRelCount)
	})

	t.Run("JSON serialization with optional fields", func(t *testing.T) {
		rowEstimate := int64(500)

		node := SchemaNode{
			Schema:           "public",
			Name:             "posts",
			Columns:          []SchemaNodeColumn{},
			PrimaryKey:       []string{"id"},
			RLSEnabled:       true,
			ForceRLS:         true,
			RowEstimate:      &rowEstimate,
			Comment:          nil, // nil should be omitted
			IncomingRelCount: 0,
			OutgoingRelCount: 1,
		}

		data, err := json.Marshal(node)
		require.NoError(t, err)

		assert.Contains(t, string(data), `"schema":"public"`)
		assert.Contains(t, string(data), `"name":"posts"`)
		assert.Contains(t, string(data), `"rls_enabled":true`)
		assert.Contains(t, string(data), `"force_rls":true`)
		assert.Contains(t, string(data), `"row_estimate":500`)
		assert.NotContains(t, string(data), `"comment"`) // Should be omitted when nil
	})

	t.Run("JSON serialization without optional fields", func(t *testing.T) {
		node := SchemaNode{
			Schema:     "public",
			Name:       "minimal_table",
			Columns:    []SchemaNodeColumn{},
			PrimaryKey: []string{},
			RLSEnabled: false,
			ForceRLS:   false,
			// RowEstimate and Comment are nil
		}

		data, err := json.Marshal(node)
		require.NoError(t, err)

		assert.Contains(t, string(data), `"schema":"public"`)
		assert.Contains(t, string(data), `"name":"minimal_table"`)
		assert.NotContains(t, string(data), `"row_estimate"`)
		assert.NotContains(t, string(data), `"comment"`)
	})

	t.Run("composite primary key", func(t *testing.T) {
		node := SchemaNode{
			Schema: "public",
			Name:   "order_items",
			Columns: []SchemaNodeColumn{
				{Name: "order_id", DataType: "uuid", IsPrimaryKey: true},
				{Name: "product_id", DataType: "uuid", IsPrimaryKey: true},
				{Name: "quantity", DataType: "integer", IsPrimaryKey: false},
			},
			PrimaryKey: []string{"order_id", "product_id"},
		}

		assert.Len(t, node.PrimaryKey, 2)
		assert.Contains(t, node.PrimaryKey, "order_id")
		assert.Contains(t, node.PrimaryKey, "product_id")
	})
}

// =============================================================================
// SchemaNodeColumn Struct Tests
// =============================================================================

func TestSchemaNodeColumn_Struct(t *testing.T) {
	t.Run("basic column", func(t *testing.T) {
		col := SchemaNodeColumn{
			Name:         "id",
			DataType:     "uuid",
			Nullable:     false,
			IsPrimaryKey: true,
			IsForeignKey: false,
			FKTarget:     nil,
			DefaultValue: nil,
			IsUnique:     true,
			IsIndexed:    true,
			Comment:      nil,
		}

		assert.Equal(t, "id", col.Name)
		assert.Equal(t, "uuid", col.DataType)
		assert.False(t, col.Nullable)
		assert.True(t, col.IsPrimaryKey)
		assert.False(t, col.IsForeignKey)
		assert.Nil(t, col.FKTarget)
		assert.True(t, col.IsUnique)
		assert.True(t, col.IsIndexed)
	})

	t.Run("foreign key column", func(t *testing.T) {
		fkTarget := "public.users.id"
		col := SchemaNodeColumn{
			Name:         "user_id",
			DataType:     "uuid",
			Nullable:     false,
			IsPrimaryKey: false,
			IsForeignKey: true,
			FKTarget:     &fkTarget,
			IsUnique:     false,
			IsIndexed:    true,
		}

		assert.True(t, col.IsForeignKey)
		assert.Equal(t, "public.users.id", *col.FKTarget)
	})

	t.Run("column with default value", func(t *testing.T) {
		defaultValue := "gen_random_uuid()"
		col := SchemaNodeColumn{
			Name:         "id",
			DataType:     "uuid",
			DefaultValue: &defaultValue,
		}

		assert.Equal(t, "gen_random_uuid()", *col.DefaultValue)
	})

	t.Run("column with comment", func(t *testing.T) {
		comment := "User's email address"
		col := SchemaNodeColumn{
			Name:     "email",
			DataType: "text",
			Comment:  &comment,
		}

		assert.Equal(t, "User's email address", *col.Comment)
	})

	t.Run("JSON serialization with all optional fields", func(t *testing.T) {
		fkTarget := "public.categories.id"
		defaultValue := "1"
		comment := "Category reference"

		col := SchemaNodeColumn{
			Name:         "category_id",
			DataType:     "integer",
			Nullable:     true,
			IsPrimaryKey: false,
			IsForeignKey: true,
			FKTarget:     &fkTarget,
			DefaultValue: &defaultValue,
			IsUnique:     false,
			IsIndexed:    true,
			Comment:      &comment,
		}

		data, err := json.Marshal(col)
		require.NoError(t, err)

		assert.Contains(t, string(data), `"name":"category_id"`)
		assert.Contains(t, string(data), `"data_type":"integer"`)
		assert.Contains(t, string(data), `"nullable":true`)
		assert.Contains(t, string(data), `"is_primary_key":false`)
		assert.Contains(t, string(data), `"is_foreign_key":true`)
		assert.Contains(t, string(data), `"fk_target":"public.categories.id"`)
		assert.Contains(t, string(data), `"default_value":"1"`)
		assert.Contains(t, string(data), `"is_unique":false`)
		assert.Contains(t, string(data), `"is_indexed":true`)
		assert.Contains(t, string(data), `"comment":"Category reference"`)
	})

	t.Run("JSON serialization omits nil optional fields", func(t *testing.T) {
		col := SchemaNodeColumn{
			Name:         "name",
			DataType:     "text",
			Nullable:     false,
			IsPrimaryKey: false,
			IsForeignKey: false,
			// FKTarget, DefaultValue, Comment are nil
		}

		data, err := json.Marshal(col)
		require.NoError(t, err)

		assert.NotContains(t, string(data), `"fk_target"`)
		assert.NotContains(t, string(data), `"default_value"`)
		assert.NotContains(t, string(data), `"comment"`)
	})
}

// =============================================================================
// SchemaGraphResponse Struct Tests
// =============================================================================

func TestSchemaGraphResponse_Struct(t *testing.T) {
	t.Run("all fields accessible", func(t *testing.T) {
		resp := SchemaGraphResponse{
			Nodes: []SchemaNode{
				{Schema: "public", Name: "users"},
				{Schema: "public", Name: "posts"},
			},
			Edges: []SchemaRelationship{
				{ID: "fk1", SourceTable: "posts", TargetTable: "users"},
			},
			Schemas: []string{"public"},
		}

		assert.Len(t, resp.Nodes, 2)
		assert.Len(t, resp.Edges, 1)
		assert.Equal(t, []string{"public"}, resp.Schemas)
	})

	t.Run("JSON serialization", func(t *testing.T) {
		resp := SchemaGraphResponse{
			Nodes: []SchemaNode{
				{Schema: "public", Name: "users", Columns: []SchemaNodeColumn{}},
			},
			Edges:   []SchemaRelationship{},
			Schemas: []string{"public", "auth"},
		}

		data, err := json.Marshal(resp)
		require.NoError(t, err)

		assert.Contains(t, string(data), `"nodes"`)
		assert.Contains(t, string(data), `"edges"`)
		assert.Contains(t, string(data), `"schemas"`)
	})

	t.Run("empty response", func(t *testing.T) {
		resp := SchemaGraphResponse{
			Nodes:   []SchemaNode{},
			Edges:   []SchemaRelationship{},
			Schemas: []string{},
		}

		assert.Empty(t, resp.Nodes)
		assert.Empty(t, resp.Edges)
		assert.Empty(t, resp.Schemas)
	})

	t.Run("multiple schemas", func(t *testing.T) {
		resp := SchemaGraphResponse{
			Nodes: []SchemaNode{
				{Schema: "public", Name: "users"},
				{Schema: "auth", Name: "identities"},
				{Schema: "storage", Name: "buckets"},
			},
			Edges:   []SchemaRelationship{},
			Schemas: []string{"public", "auth", "storage"},
		}

		assert.Len(t, resp.Schemas, 3)
		assert.Contains(t, resp.Schemas, "public")
		assert.Contains(t, resp.Schemas, "auth")
		assert.Contains(t, resp.Schemas, "storage")
	})
}

// =============================================================================
// GetSchemaGraph Handler Tests
// =============================================================================

func TestGetSchemaGraph_ParameterParsing(t *testing.T) {
	t.Run("default schemas parameter", func(t *testing.T) {
		app := newTestApp(t)
		h := &SchemaGraphHandlers{db: nil}

		app.Get("/schema/graph", h.GetSchemaGraph)

		req := httptest.NewRequest(http.MethodGet, "/schema/graph", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.NotEqual(t, fiber.StatusNotFound, resp.StatusCode)
	})

	t.Run("custom schemas parameter", func(t *testing.T) {
		app := newTestApp(t)
		h := &SchemaGraphHandlers{db: nil}

		app.Get("/schema/graph", h.GetSchemaGraph)

		req := httptest.NewRequest(http.MethodGet, "/schema/graph?schemas=public,auth,storage", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.NotEqual(t, fiber.StatusNotFound, resp.StatusCode)
	})

	t.Run("single schema parameter", func(t *testing.T) {
		app := newTestApp(t)
		h := &SchemaGraphHandlers{db: nil}

		app.Get("/schema/graph", h.GetSchemaGraph)

		req := httptest.NewRequest(http.MethodGet, "/schema/graph?schemas=auth", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.NotEqual(t, fiber.StatusNotFound, resp.StatusCode)
	})

	t.Run("schemas with whitespace", func(t *testing.T) {
		app := newTestApp(t)
		h := &SchemaGraphHandlers{db: nil}

		app.Get("/schema/graph", h.GetSchemaGraph)

		req := httptest.NewRequest(http.MethodGet, "/schema/graph?schemas=public,%20auth,%20storage", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.NotEqual(t, fiber.StatusNotFound, resp.StatusCode)
	})
}

// =============================================================================
// GetTableRelationships Handler Tests
// =============================================================================

func TestGetTableRelationships_ParameterValidation(t *testing.T) {
	t.Run("missing schema parameter", func(t *testing.T) {
		app := newTestApp(t)
		h := &SchemaGraphHandlers{db: nil}

		app.Get("/tables/:schema/:table/relationships", h.GetTableRelationships)

		req := httptest.NewRequest(http.MethodGet, "/tables//users/relationships", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.True(t, resp.StatusCode == fiber.StatusNotFound || resp.StatusCode == fiber.StatusBadRequest)
	})

	t.Run("missing table parameter", func(t *testing.T) {
		app := newTestApp(t)
		h := &SchemaGraphHandlers{db: nil}

		app.Get("/tables/:schema/:table/relationships", h.GetTableRelationships)

		req := httptest.NewRequest(http.MethodGet, "/tables/public//relationships", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.True(t, resp.StatusCode == fiber.StatusNotFound || resp.StatusCode == fiber.StatusBadRequest)
	})

	t.Run("valid schema and table parameters", func(t *testing.T) {
		app := newTestApp(t)
		h := &SchemaGraphHandlers{db: nil}

		app.Get("/tables/:schema/:table/relationships", h.GetTableRelationships)

		req := httptest.NewRequest(http.MethodGet, "/tables/public/users/relationships", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.NotEqual(t, fiber.StatusNotFound, resp.StatusCode)
		assert.NotEqual(t, fiber.StatusBadRequest, resp.StatusCode)
	})

	t.Run("schema and table with underscores", func(t *testing.T) {
		app := newTestApp(t)
		h := &SchemaGraphHandlers{db: nil}

		app.Get("/tables/:schema/:table/relationships", h.GetTableRelationships)

		req := httptest.NewRequest(http.MethodGet, "/tables/my_schema/my_table/relationships", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.NotEqual(t, fiber.StatusNotFound, resp.StatusCode)
		assert.NotEqual(t, fiber.StatusBadRequest, resp.StatusCode)
	})
}

// =============================================================================
// Data Type Tests
// =============================================================================

func TestSchemaDataTypes(t *testing.T) {
	t.Run("common PostgreSQL data types", func(t *testing.T) {
		dataTypes := []string{
			"uuid",
			"text",
			"varchar",
			"integer",
			"bigint",
			"smallint",
			"boolean",
			"timestamp",
			"timestamptz",
			"date",
			"time",
			"timetz",
			"json",
			"jsonb",
			"numeric",
			"decimal",
			"real",
			"double precision",
			"bytea",
			"inet",
			"cidr",
			"macaddr",
		}

		for _, dt := range dataTypes {
			col := SchemaNodeColumn{
				Name:     "test_col",
				DataType: dt,
			}
			assert.Equal(t, dt, col.DataType)
		}
	})

	t.Run("array data types", func(t *testing.T) {
		col := SchemaNodeColumn{
			Name:     "tags",
			DataType: "text[]",
		}
		assert.Equal(t, "text[]", col.DataType)

		col2 := SchemaNodeColumn{
			Name:     "scores",
			DataType: "integer[]",
		}
		assert.Equal(t, "integer[]", col2.DataType)
	})
}

// =============================================================================
// Relationship Rule Tests
// =============================================================================

func TestRelationshipRules(t *testing.T) {
	t.Run("on_delete rules", func(t *testing.T) {
		rules := []string{"NO ACTION", "RESTRICT", "CASCADE", "SET NULL", "SET DEFAULT"}

		for _, rule := range rules {
			rel := SchemaRelationship{
				ID:       "test",
				OnDelete: rule,
			}
			assert.Equal(t, rule, rel.OnDelete)
		}
	})

	t.Run("on_update rules", func(t *testing.T) {
		rules := []string{"NO ACTION", "RESTRICT", "CASCADE", "SET NULL", "SET DEFAULT"}

		for _, rule := range rules {
			rel := SchemaRelationship{
				ID:       "test",
				OnUpdate: rule,
			}
			assert.Equal(t, rule, rel.OnUpdate)
		}
	})
}

// =============================================================================
// Complete Schema Graph Example Tests
// =============================================================================

func TestCompleteSchemaGraphExample(t *testing.T) {
	t.Run("typical blog schema", func(t *testing.T) {
		// Build a typical blog schema for testing
		rowEstimate := int64(100)

		usersNode := SchemaNode{
			Schema: "public",
			Name:   "users",
			Columns: []SchemaNodeColumn{
				{Name: "id", DataType: "uuid", IsPrimaryKey: true, IsUnique: true, IsIndexed: true},
				{Name: "email", DataType: "text", IsUnique: true, IsIndexed: true},
				{Name: "name", DataType: "text", Nullable: true},
				{Name: "created_at", DataType: "timestamptz"},
			},
			PrimaryKey:       []string{"id"},
			RLSEnabled:       true,
			RowEstimate:      &rowEstimate,
			IncomingRelCount: 2, // posts.user_id, comments.user_id
			OutgoingRelCount: 0,
		}

		fkTarget := "public.users.id"
		postsNode := SchemaNode{
			Schema: "public",
			Name:   "posts",
			Columns: []SchemaNodeColumn{
				{Name: "id", DataType: "uuid", IsPrimaryKey: true},
				{Name: "user_id", DataType: "uuid", IsForeignKey: true, FKTarget: &fkTarget, IsIndexed: true},
				{Name: "title", DataType: "text"},
				{Name: "content", DataType: "text"},
			},
			PrimaryKey:       []string{"id"},
			RLSEnabled:       true,
			IncomingRelCount: 1, // comments.post_id
			OutgoingRelCount: 1, // user_id -> users
		}

		userPostsEdge := SchemaRelationship{
			ID:             "posts_user_id_fkey",
			SourceSchema:   "public",
			SourceTable:    "posts",
			SourceColumn:   "user_id",
			TargetSchema:   "public",
			TargetTable:    "users",
			TargetColumn:   "id",
			ConstraintName: "posts_user_id_fkey",
			OnDelete:       "CASCADE",
			OnUpdate:       "NO ACTION",
			Cardinality:    "many-to-one",
		}

		response := SchemaGraphResponse{
			Nodes:   []SchemaNode{usersNode, postsNode},
			Edges:   []SchemaRelationship{userPostsEdge},
			Schemas: []string{"public"},
		}

		// Verify the schema graph structure
		assert.Len(t, response.Nodes, 2)
		assert.Len(t, response.Edges, 1)

		// Verify serialization
		data, err := json.Marshal(response)
		require.NoError(t, err)
		assert.NotEmpty(t, data)

		// Verify deserialization
		var parsed SchemaGraphResponse
		err = json.Unmarshal(data, &parsed)
		require.NoError(t, err)

		assert.Len(t, parsed.Nodes, 2)
		assert.Len(t, parsed.Edges, 1)
		assert.Equal(t, "posts_user_id_fkey", parsed.Edges[0].ConstraintName)
	})
}
