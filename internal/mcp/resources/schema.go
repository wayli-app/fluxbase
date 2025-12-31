package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/fluxbase-eu/fluxbase/internal/database"
	"github.com/fluxbase-eu/fluxbase/internal/mcp"
)

// SchemaResource provides database schema information
type SchemaResource struct {
	schemaCache *database.SchemaCache
}

// NewSchemaResource creates a new schema resource
func NewSchemaResource(schemaCache *database.SchemaCache) *SchemaResource {
	return &SchemaResource{
		schemaCache: schemaCache,
	}
}

func (r *SchemaResource) URI() string {
	return "fluxbase://schema/tables"
}

func (r *SchemaResource) Name() string {
	return "Database Schema"
}

func (r *SchemaResource) Description() string {
	return "Complete database schema including tables, columns, types, and relationships"
}

func (r *SchemaResource) MimeType() string {
	return "application/json"
}

func (r *SchemaResource) RequiredScopes() []string {
	return []string{mcp.ScopeReadTables}
}

func (r *SchemaResource) Read(ctx context.Context, authCtx *mcp.AuthContext) ([]mcp.Content, error) {
	if r.schemaCache == nil {
		return nil, fmt.Errorf("schema cache not available")
	}

	// Get all tables from cache
	tables := r.schemaCache.GetAllTables()

	// Build schema representation
	schema := make(map[string]any)

	for _, table := range tables {
		// Skip internal schemas unless user has service role
		if isInternalSchema(table.Schema) && authCtx.UserRole != "service_role" && authCtx.UserRole != "dashboard_admin" {
			continue
		}

		key := fmt.Sprintf("%s.%s", table.Schema, table.Name)

		columns := make([]map[string]any, 0, len(table.Columns))
		for _, col := range table.Columns {
			colInfo := map[string]any{
				"name":     col.Name,
				"type":     col.DataType,
				"nullable": col.IsNullable,
			}
			if col.DefaultValue != nil {
				colInfo["default"] = *col.DefaultValue
			}
			if col.IsPrimaryKey {
				colInfo["primary_key"] = true
			}
			columns = append(columns, colInfo)
		}

		tableInfo := map[string]any{
			"schema":  table.Schema,
			"name":    table.Name,
			"columns": columns,
		}

		if len(table.PrimaryKey) > 0 {
			tableInfo["primary_key"] = table.PrimaryKey
		}

		if len(table.ForeignKeys) > 0 {
			fks := make([]map[string]any, 0, len(table.ForeignKeys))
			for _, fk := range table.ForeignKeys {
				fks = append(fks, map[string]any{
					"columns":            fk.Columns,
					"referenced_schema":  fk.ReferencedSchema,
					"referenced_table":   fk.ReferencedTable,
					"referenced_columns": fk.ReferencedColumns,
				})
			}
			tableInfo["foreign_keys"] = fks
		}

		schema[key] = tableInfo
	}

	// Serialize to JSON
	data, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to serialize schema: %w", err)
	}

	return []mcp.Content{mcp.TextContent(string(data))}, nil
}

// TableResource provides information about a specific table
type TableResource struct {
	schemaCache *database.SchemaCache
}

// NewTableResource creates a new table resource
func NewTableResource(schemaCache *database.SchemaCache) *TableResource {
	return &TableResource{
		schemaCache: schemaCache,
	}
}

func (r *TableResource) URI() string {
	return "fluxbase://schema/tables/{schema}/{table}"
}

func (r *TableResource) Name() string {
	return "Table Details"
}

func (r *TableResource) Description() string {
	return "Detailed information about a specific database table"
}

func (r *TableResource) MimeType() string {
	return "application/json"
}

func (r *TableResource) RequiredScopes() []string {
	return []string{mcp.ScopeReadTables}
}

// IsTemplate returns true since this resource uses URI templates
func (r *TableResource) IsTemplate() bool {
	return true
}

// MatchURI checks if a URI matches this resource template and extracts parameters
func (r *TableResource) MatchURI(uri string) (map[string]string, bool) {
	// Parse fluxbase://schema/tables/{schema}/{table}
	if !strings.HasPrefix(uri, "fluxbase://schema/tables/") {
		return nil, false
	}

	path := strings.TrimPrefix(uri, "fluxbase://schema/tables/")
	parts := strings.SplitN(path, "/", 2)
	if len(parts) != 2 {
		return nil, false
	}

	return map[string]string{
		"schema": parts[0],
		"table":  parts[1],
	}, true
}

func (r *TableResource) Read(ctx context.Context, authCtx *mcp.AuthContext) ([]mcp.Content, error) {
	return nil, fmt.Errorf("use ReadWithParams for template resources")
}

// ReadWithParams reads the resource with URI parameters
func (r *TableResource) ReadWithParams(ctx context.Context, authCtx *mcp.AuthContext, params map[string]string) ([]mcp.Content, error) {
	if r.schemaCache == nil {
		return nil, fmt.Errorf("schema cache not available")
	}

	schema := params["schema"]
	tableName := params["table"]

	if schema == "" || tableName == "" {
		return nil, fmt.Errorf("schema and table parameters are required")
	}

	// Check access to internal schemas
	if isInternalSchema(schema) && authCtx.UserRole != "service_role" && authCtx.UserRole != "dashboard_admin" {
		return nil, fmt.Errorf("access denied to internal schema: %s", schema)
	}

	// Get table from cache
	table := r.schemaCache.GetTable(schema, tableName)
	if table == nil {
		return nil, fmt.Errorf("table not found: %s.%s", schema, tableName)
	}

	// Build detailed table info
	columns := make([]map[string]any, 0, len(table.Columns))
	for _, col := range table.Columns {
		colInfo := map[string]any{
			"name":     col.Name,
			"type":     col.DataType,
			"nullable": col.IsNullable,
			"position": col.Position,
		}
		if col.DefaultValue != nil {
			colInfo["default"] = *col.DefaultValue
		}
		if col.IsPrimaryKey {
			colInfo["primary_key"] = true
		}
		if col.MaxLength != nil {
			colInfo["max_length"] = *col.MaxLength
		}
		columns = append(columns, colInfo)
	}

	tableInfo := map[string]any{
		"schema":  table.Schema,
		"name":    table.Name,
		"columns": columns,
	}

	if len(table.PrimaryKey) > 0 {
		tableInfo["primary_key"] = table.PrimaryKey
	}

	if len(table.ForeignKeys) > 0 {
		fks := make([]map[string]any, 0, len(table.ForeignKeys))
		for _, fk := range table.ForeignKeys {
			fks = append(fks, map[string]any{
				"name":               fk.Name,
				"columns":            fk.Columns,
				"referenced_schema":  fk.ReferencedSchema,
				"referenced_table":   fk.ReferencedTable,
				"referenced_columns": fk.ReferencedColumns,
			})
		}
		tableInfo["foreign_keys"] = fks
	}

	if len(table.Indexes) > 0 {
		indexes := make([]map[string]any, 0, len(table.Indexes))
		for _, idx := range table.Indexes {
			indexes = append(indexes, map[string]any{
				"name":    idx.Name,
				"columns": idx.Columns,
				"unique":  idx.IsUnique,
			})
		}
		tableInfo["indexes"] = indexes
	}

	// Serialize to JSON
	data, err := json.MarshalIndent(tableInfo, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to serialize table info: %w", err)
	}

	return []mcp.Content{mcp.TextContent(string(data))}, nil
}

// isInternalSchema checks if a schema is internal/system
func isInternalSchema(schema string) bool {
	internalSchemas := []string{
		"auth",
		"storage",
		"realtime",
		"functions",
		"jobs",
		"rpc",
		"logging",
		"ai",
		"branching",
		"pg_catalog",
		"information_schema",
	}

	for _, s := range internalSchemas {
		if schema == s {
			return true
		}
	}
	return false
}
