package ai

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/nimbleflux/fluxbase/internal/database"
)

func TestMetadataToJSON_EmptyMap(t *testing.T) {
	result, err := metadataToJSON(map[string]interface{}{})
	require.NoError(t, err)
	require.Equal(t, "{}", string(result))
}

func TestMetadataToJSON_NilMap(t *testing.T) {
	result, err := metadataToJSON(nil)
	require.NoError(t, err)
	require.Equal(t, "{}", string(result))
}

func TestMetadataToJSON_BasicTypes(t *testing.T) {
	m := map[string]interface{}{
		"string":  "value",
		"int":     42,
		"bool":    true,
		"strings": []string{"a", "b", "c"},
	}

	result, err := metadataToJSON(m)
	require.NoError(t, err)

	// Verify it's valid JSON by unmarshaling
	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal(result, &parsed))

	require.Equal(t, "value", parsed["string"])
	require.Equal(t, float64(42), parsed["int"]) // JSON numbers are float64
	require.Equal(t, true, parsed["bool"])
}

func TestMetadataToJSON_SpecialCharacters(t *testing.T) {
	tests := []struct {
		name  string
		input map[string]interface{}
	}{
		{
			name: "quotes",
			input: map[string]interface{}{
				"value": `string with "quotes" inside`,
			},
		},
		{
			name: "backslashes",
			input: map[string]interface{}{
				"path": `c:\path\to\file`,
			},
		},
		{
			name: "newlines",
			input: map[string]interface{}{
				"multiline": "line1\nline2\rline3",
			},
		},
		{
			name: "unicode",
			input: map[string]interface{}{
				"emoji": "emoji 🎉 test",
			},
		},
		{
			name: "mixed special chars",
			input: map[string]interface{}{
				"quote":     `he said "hello"`,
				"backslash": `c:\test\path`,
				"newline":   "line1\nline2",
				"tab":       "col1\tcol2",
				"unicode":   "日本語 🎉",
			},
		},
		{
			name: "table type values",
			input: map[string]interface{}{
				"table_type": "BASE TABLE",
				"schema":     "public",
				"table":      "place_visits",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := metadataToJSON(tt.input)
			require.NoError(t, err, "metadataToJSON should not error on %s", tt.name)

			// Most important check: PostgreSQL would accept this as valid JSON
			var parsed map[string]interface{}
			err = json.Unmarshal(result, &parsed)
			require.NoError(t, err, "Result should be valid JSON: %s", string(result))

			// Verify all keys are present
			for k := range tt.input {
				_, exists := parsed[k]
				require.True(t, exists, "Key %s should exist in parsed result", k)
			}
		})
	}
}

func TestMetadataToJSON_TableExporterMetadata(t *testing.T) {
	// Simulate the exact metadata structure used in ExportTable
	m := map[string]interface{}{
		"schema":           "public",
		"table":            "place_visits",
		"entity_type":      "table",
		"source":           "database_export",
		"table_type":       "BASE TABLE",
		"rls_enabled":      false,
		"exported_columns": 5,
		"total_columns":    10,
		"columns_filtered": true,
		"columns":          []string{"id", "name", "created_at"},
	}

	result, err := metadataToJSON(m)
	require.NoError(t, err)

	// Verify it's valid JSON
	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal(result, &parsed))

	// Verify key fields
	require.Equal(t, "public", parsed["schema"])
	require.Equal(t, "place_visits", parsed["table"])
	require.Equal(t, "BASE TABLE", parsed["table_type"])
}

// =============================================================================
// generateTableDocument — pure markdown builder for an exported table
// =============================================================================
//
// Builds a markdown doc from a *database.TableInfo struct (title, description,
// primary key, columns table with markers, JSONB schemas). The receiver `e` is
// not used, so a zero TableExporter is a valid fixture. Currently 0.0%.

func TestGenerateTableDocument_TitleAndDescription(t *testing.T) {
	t.Parallel()
	table := &database.TableInfo{
		Schema: "public",
		Name:   "users",
		Type:   "BASE TABLE",
	}
	doc := (&TableExporter{}).generateTableDocument(table, ExportTableRequest{})

	if !strings.Contains(doc, "# Table: public.users") {
		t.Errorf("missing title heading; got:\n%s", doc)
	}
	if !strings.Contains(doc, "Database **BASE TABLE** in schema `public`") {
		t.Errorf("missing description; got:\n%s", doc)
	}
}

func TestGenerateTableDocument_RLSNote(t *testing.T) {
	t.Parallel()
	table := &database.TableInfo{Schema: "s", Name: "t", Type: "BASE TABLE", RLSEnabled: true}
	doc := (&TableExporter{}).generateTableDocument(table, ExportTableRequest{})
	if !strings.Contains(doc, "Row Level Security (RLS) is enabled") {
		t.Errorf("missing RLS note; got:\n%s", doc)
	}
}

func TestGenerateTableDocument_PrimaryKey(t *testing.T) {
	t.Parallel()
	table := &database.TableInfo{
		Schema:     "public",
		Name:       "users",
		Type:       "BASE TABLE",
		PrimaryKey: []string{"id", "tenant_id"},
	}
	doc := (&TableExporter{}).generateTableDocument(table, ExportTableRequest{})
	if !strings.Contains(doc, "**Primary Key:**") {
		t.Errorf("missing primary key line; got:\n%s", doc)
	}
	if !strings.Contains(doc, "`id`") || !strings.Contains(doc, "`tenant_id`") {
		t.Errorf("primary key values not rendered; got:\n%s", doc)
	}
}

func TestGenerateTableDocument_ColumnMarkers(t *testing.T) {
	t.Parallel()
	defVal := "nextval('seq')"
	table := &database.TableInfo{
		Schema: "public", Name: "users", Type: "BASE TABLE",
		Columns: []database.ColumnInfo{
			{Name: "id", DataType: "bigint", IsNullable: false, IsPrimaryKey: true, DefaultValue: &defVal},
			{Name: "org_id", DataType: "bigint", IsForeignKey: true},
			{Name: "email", DataType: "text", IsUnique: true},
			{Name: "bio", DataType: "text", IsNullable: true},
		},
	}
	doc := (&TableExporter{}).generateTableDocument(table, ExportTableRequest{})

	// PK marker on the id column.
	if !strings.Contains(doc, "🔑 id") {
		t.Errorf("missing 🔑 PK marker on id; got:\n%s", doc)
	}
	// FK marker on org_id.
	if !strings.Contains(doc, "🔗 org_id") {
		t.Errorf("missing 🔗 FK marker on org_id; got:\n%s", doc)
	}
	// Unique marker on email.
	if !strings.Contains(doc, "email") || !strings.Contains(doc, "🦄") {
		t.Errorf("missing 🦄 unique marker on email; got:\n%s", doc)
	}
	// Nullable rendering: bio is nullable → "NULL"; id is NOT NULL.
	if !strings.Contains(doc, "| bio | text | NULL |") {
		t.Errorf("expected NULL nullable for bio; got:\n%s", doc)
	}
	if !strings.Contains(doc, "| 🔑 id | bigint | NOT NULL | nextval('seq')") {
		t.Errorf("expected NOT NULL + default for id; got:\n%s", doc)
	}
}

func TestGenerateTableDocument_ColumnFiltering(t *testing.T) {
	t.Parallel()
	table := &database.TableInfo{
		Schema: "public", Name: "users", Type: "BASE TABLE",
		Columns: []database.ColumnInfo{
			{Name: "id", DataType: "bigint"},
			{Name: "email", DataType: "text"},
			{Name: "bio", DataType: "text"},
		},
	}
	// Request only the email column.
	doc := (&TableExporter{}).generateTableDocument(table, ExportTableRequest{Columns: []string{"email"}})

	if !strings.Contains(doc, "Exporting 1 of 3 columns") {
		t.Errorf("missing export-count note; got:\n%s", doc)
	}
	// email should appear; id and bio should not (as column-row entries).
	if !strings.Contains(doc, "| email |") {
		t.Errorf("filtered-in column email missing; got:\n%s", doc)
	}
	if strings.Contains(doc, "| id |") || strings.Contains(doc, "| bio |") {
		t.Errorf("filtered-out columns should not appear; got:\n%s", doc)
	}
}

func TestGenerateTableDocument_JSONBSchema(t *testing.T) {
	t.Parallel()
	table := &database.TableInfo{
		Schema: "public", Name: "events", Type: "BASE TABLE",
		Columns: []database.ColumnInfo{{
			Name:     "payload",
			DataType: "jsonb",
			JSONBSchema: &database.JSONBSchemaInfo{
				Properties: map[string]database.JSONBProperty{
					"event": {Type: "string", Description: "event name"},
				},
				Required: []string{"event"},
			},
		}},
	}
	doc := (&TableExporter{}).generateTableDocument(table, ExportTableRequest{})
	if !strings.Contains(doc, "## JSONB Column: `payload`") {
		t.Errorf("missing JSONB section; got:\n%s", doc)
	}
	if !strings.Contains(doc, "| event | string | yes | event name |") {
		t.Errorf("missing JSONB property row; got:\n%s", doc)
	}
}
