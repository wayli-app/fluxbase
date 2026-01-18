package api

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/fluxbase-eu/fluxbase/internal/database"
	"github.com/gofiber/fiber/v2"
)

// SchemaExportHandler handles schema export operations for type generation
type SchemaExportHandler struct {
	schemaCache *database.SchemaCache
	inspector   *database.SchemaInspector
}

// NewSchemaExportHandler creates a new schema export handler
func NewSchemaExportHandler(schemaCache *database.SchemaCache, inspector *database.SchemaInspector) *SchemaExportHandler {
	return &SchemaExportHandler{
		schemaCache: schemaCache,
		inspector:   inspector,
	}
}

// TypeScriptExportRequest represents a request for TypeScript type generation
type TypeScriptExportRequest struct {
	Schemas          []string `json:"schemas"`           // Schemas to include (default: ["public"])
	IncludeFunctions bool     `json:"include_functions"` // Include RPC function types
	IncludeViews     bool     `json:"include_views"`     // Include view types
	Format           string   `json:"format"`            // "types" (interfaces only) or "full" (with helpers)
}

// HandleExportTypeScript generates TypeScript type definitions from the database schema
func (h *SchemaExportHandler) HandleExportTypeScript(c *fiber.Ctx) error {
	ctx := c.Context()

	// Parse request
	var req TypeScriptExportRequest
	if err := c.BodyParser(&req); err != nil {
		// Use defaults for GET requests or invalid body
		req = TypeScriptExportRequest{
			Schemas:          []string{"public"},
			IncludeFunctions: true,
			IncludeViews:     true,
			Format:           "types",
		}
	}

	// Apply defaults
	if len(req.Schemas) == 0 {
		req.Schemas = []string{"public"}
	}
	if req.Format == "" {
		req.Format = "types"
	}

	// Generate TypeScript
	output, err := h.generateTypeScript(ctx, req)
	if err != nil {
		return SendInternalError(c, "Failed to generate TypeScript types: "+err.Error())
	}

	// Set content type based on request method and accept header
	// POST requests always return JSON (for programmatic use like CLI)
	// GET requests return plain text unless Accept header specifies JSON
	accept := c.Get("Accept")
	if c.Method() == "POST" || strings.Contains(accept, "application/json") {
		return c.JSON(fiber.Map{
			"typescript": output,
			"schemas":    req.Schemas,
		})
	}

	// Return plain TypeScript for GET requests (for browser/curl use)
	c.Set("Content-Type", "text/plain; charset=utf-8")
	c.Set("Content-Disposition", "inline; filename=\"types.ts\"")
	return c.SendString(output)
}

// generateTypeScript generates TypeScript definitions from database schema
func (h *SchemaExportHandler) generateTypeScript(ctx context.Context, req TypeScriptExportRequest) (string, error) {
	var sb strings.Builder

	// Header
	sb.WriteString("// Auto-generated TypeScript types from Fluxbase database schema\n")
	sb.WriteString("// Generated at: " + getCurrentTimestamp() + "\n")
	sb.WriteString("// Schemas: " + strings.Join(req.Schemas, ", ") + "\n\n")

	// Generate table types
	tables, err := h.schemaCache.GetAllTables(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get tables: %w", err)
	}

	// Filter by requested schemas
	filteredTables := filterBySchema(tables, req.Schemas)

	// Group tables by schema
	tablesBySchema := groupBySchema(filteredTables)

	// Generate types for each schema
	schemaNames := make([]string, 0, len(tablesBySchema))
	for schema := range tablesBySchema {
		schemaNames = append(schemaNames, schema)
	}
	sort.Strings(schemaNames)

	for _, schema := range schemaNames {
		schemaTables := tablesBySchema[schema]
		sb.WriteString(h.generateSchemaTypes(schema, schemaTables, "table"))
	}

	// Generate view types if requested
	if req.IncludeViews {
		views, err := h.schemaCache.GetAllViews(ctx)
		if err != nil {
			return "", fmt.Errorf("failed to get views: %w", err)
		}

		filteredViews := filterBySchema(views, req.Schemas)
		if len(filteredViews) > 0 {
			sb.WriteString("// ==================== Views ====================\n\n")
			viewsBySchema := groupBySchema(filteredViews)

			viewSchemaNames := make([]string, 0, len(viewsBySchema))
			for schema := range viewsBySchema {
				viewSchemaNames = append(viewSchemaNames, schema)
			}
			sort.Strings(viewSchemaNames)

			for _, schema := range viewSchemaNames {
				schemaViews := viewsBySchema[schema]
				sb.WriteString(h.generateSchemaTypes(schema, schemaViews, "view"))
			}
		}

		// Materialized views
		matViews, err := h.schemaCache.GetAllMaterializedViews(ctx)
		if err != nil {
			return "", fmt.Errorf("failed to get materialized views: %w", err)
		}

		filteredMatViews := filterBySchema(matViews, req.Schemas)
		if len(filteredMatViews) > 0 {
			sb.WriteString("// ==================== Materialized Views ====================\n\n")
			matViewsBySchema := groupBySchema(filteredMatViews)

			matViewSchemaNames := make([]string, 0, len(matViewsBySchema))
			for schema := range matViewsBySchema {
				matViewSchemaNames = append(matViewSchemaNames, schema)
			}
			sort.Strings(matViewSchemaNames)

			for _, schema := range matViewSchemaNames {
				schemaMatViews := matViewsBySchema[schema]
				sb.WriteString(h.generateSchemaTypes(schema, schemaMatViews, "materialized_view"))
			}
		}
	}

	// Generate function types if requested
	if req.IncludeFunctions {
		functions, err := h.inspector.GetAllFunctions(ctx, req.Schemas...)
		if err != nil {
			return "", fmt.Errorf("failed to get functions: %w", err)
		}

		if len(functions) > 0 {
			sb.WriteString("// ==================== RPC Functions ====================\n\n")
			sb.WriteString(h.generateFunctionTypes(functions))
		}
	}

	// Generate Database namespace export
	sb.WriteString("// ==================== Database Namespace ====================\n\n")
	sb.WriteString(h.generateDatabaseNamespace(tablesBySchema, req))

	return sb.String(), nil
}

// generateSchemaTypes generates TypeScript types for a schema's tables/views
func (h *SchemaExportHandler) generateSchemaTypes(schema string, tables []database.TableInfo, objectType string) string {
	var sb strings.Builder

	schemaNamespace := toPascalCase(schema)
	sb.WriteString(fmt.Sprintf("// ==================== %s Schema (%ss) ====================\n\n", schemaNamespace, objectType))

	for _, table := range tables {
		sb.WriteString(h.generateTableTypes(table))
	}

	return sb.String()
}

// generateTableTypes generates TypeScript types for a single table
func (h *SchemaExportHandler) generateTableTypes(table database.TableInfo) string {
	var sb strings.Builder

	typeName := toPascalCase(table.Name)
	schemaPrefix := ""
	if table.Schema != "public" {
		schemaPrefix = toPascalCase(table.Schema)
	}

	fullTypeName := typeName
	if schemaPrefix != "" {
		fullTypeName = schemaPrefix + typeName
	}

	// Row type (what you get from SELECT)
	sb.WriteString(fmt.Sprintf("/** %s.%s row type */\n", table.Schema, table.Name))
	sb.WriteString(fmt.Sprintf("export interface %sRow {\n", fullTypeName))
	for _, col := range table.Columns {
		tsType := pgTypeToTS(col.DataType)
		nullable := ""
		if col.IsNullable && !col.IsPrimaryKey {
			nullable = " | null"
		}
		comment := ""
		if col.IsPrimaryKey {
			comment = " // Primary key"
		} else if col.IsForeignKey {
			comment = " // Foreign key"
		}
		sb.WriteString(fmt.Sprintf("  %s: %s%s;%s\n", col.Name, tsType, nullable, comment))
	}
	sb.WriteString("}\n\n")

	// Insert type (what you send for INSERT)
	sb.WriteString(fmt.Sprintf("/** %s.%s insert type */\n", table.Schema, table.Name))
	sb.WriteString(fmt.Sprintf("export interface %sInsert {\n", fullTypeName))
	for _, col := range table.Columns {
		tsType := pgTypeToTS(col.DataType)
		optional := ""
		nullable := ""

		// Column is optional if it has a default value or is nullable
		hasDefault := col.DefaultValue != nil && *col.DefaultValue != ""
		if hasDefault || col.IsNullable {
			optional = "?"
		}
		if col.IsNullable {
			nullable = " | null"
		}

		sb.WriteString(fmt.Sprintf("  %s%s: %s%s;\n", col.Name, optional, tsType, nullable))
	}
	sb.WriteString("}\n\n")

	// Update type (what you send for UPDATE - all optional)
	sb.WriteString(fmt.Sprintf("/** %s.%s update type */\n", table.Schema, table.Name))
	sb.WriteString(fmt.Sprintf("export interface %sUpdate {\n", fullTypeName))
	for _, col := range table.Columns {
		tsType := pgTypeToTS(col.DataType)
		nullable := ""
		if col.IsNullable {
			nullable = " | null"
		}
		sb.WriteString(fmt.Sprintf("  %s?: %s%s;\n", col.Name, tsType, nullable))
	}
	sb.WriteString("}\n\n")

	return sb.String()
}

// generateFunctionTypes generates TypeScript types for RPC functions
func (h *SchemaExportHandler) generateFunctionTypes(functions []database.FunctionInfo) string {
	var sb strings.Builder

	// Group by schema
	funcsBySchema := make(map[string][]database.FunctionInfo)
	for _, fn := range functions {
		funcsBySchema[fn.Schema] = append(funcsBySchema[fn.Schema], fn)
	}

	schemaNames := make([]string, 0, len(funcsBySchema))
	for schema := range funcsBySchema {
		schemaNames = append(schemaNames, schema)
	}
	sort.Strings(schemaNames)

	for _, schema := range schemaNames {
		schemaFuncs := funcsBySchema[schema]
		schemaNamespace := toPascalCase(schema)

		sb.WriteString(fmt.Sprintf("// %s schema functions\n", schema))

		for _, fn := range schemaFuncs {
			funcName := toPascalCase(fn.Name)
			fullFuncName := funcName
			if schema != "public" {
				fullFuncName = schemaNamespace + funcName
			}

			// Generate args type if function has parameters
			if len(fn.Parameters) > 0 {
				sb.WriteString(fmt.Sprintf("/** Arguments for %s.%s */\n", fn.Schema, fn.Name))
				sb.WriteString(fmt.Sprintf("export interface %sArgs {\n", fullFuncName))
				for _, param := range fn.Parameters {
					if param.Mode == "OUT" {
						continue // Skip output parameters
					}
					tsType := pgTypeToTS(param.Type)
					optional := ""
					if param.HasDefault {
						optional = "?"
					}
					paramName := param.Name
					if paramName == "" {
						paramName = fmt.Sprintf("arg%d", param.Position)
					}
					sb.WriteString(fmt.Sprintf("  %s%s: %s;\n", paramName, optional, tsType))
				}
				sb.WriteString("}\n\n")
			}

			// Generate return type
			returnType := pgTypeToTS(fn.ReturnType)
			if fn.IsSetOf {
				returnType += "[]"
			}
			sb.WriteString(fmt.Sprintf("/** Return type for %s.%s */\n", fn.Schema, fn.Name))
			sb.WriteString(fmt.Sprintf("export type %sReturn = %s;\n\n", fullFuncName, returnType))
		}
	}

	return sb.String()
}

// generateDatabaseNamespace generates a Database namespace containing all types
func (h *SchemaExportHandler) generateDatabaseNamespace(tablesBySchema map[string][]database.TableInfo, req TypeScriptExportRequest) string {
	var sb strings.Builder

	sb.WriteString("export namespace Database {\n")

	schemaNames := make([]string, 0, len(tablesBySchema))
	for schema := range tablesBySchema {
		schemaNames = append(schemaNames, schema)
	}
	sort.Strings(schemaNames)

	for _, schema := range schemaNames {
		tables := tablesBySchema[schema]
		schemaNamespace := toPascalCase(schema)

		sb.WriteString(fmt.Sprintf("  export namespace %s {\n", schemaNamespace))
		sb.WriteString("    export interface Tables {\n")

		for _, table := range tables {
			typeName := toPascalCase(table.Name)
			fullTypeName := typeName
			if schema != "public" {
				fullTypeName = schemaNamespace + typeName
			}
			sb.WriteString(fmt.Sprintf("      %s: {\n", table.Name))
			sb.WriteString(fmt.Sprintf("        Row: %sRow;\n", fullTypeName))
			sb.WriteString(fmt.Sprintf("        Insert: %sInsert;\n", fullTypeName))
			sb.WriteString(fmt.Sprintf("        Update: %sUpdate;\n", fullTypeName))
			sb.WriteString("      };\n")
		}

		sb.WriteString("    }\n")
		sb.WriteString("  }\n")
	}

	sb.WriteString("}\n")

	return sb.String()
}

// pgTypeToTS converts PostgreSQL data types to TypeScript types
func pgTypeToTS(pgType string) string {
	// Normalize the type name (lowercase, strip array suffix temporarily)
	normalizedType := strings.ToLower(strings.TrimSpace(pgType))
	isArray := strings.HasSuffix(normalizedType, "[]") || strings.HasPrefix(normalizedType, "array")

	// Handle ARRAY type syntax: ARRAY[type] or type[]
	baseType := normalizedType
	if isArray {
		baseType = strings.TrimSuffix(baseType, "[]")
		baseType = strings.TrimPrefix(baseType, "array")
		baseType = strings.Trim(baseType, "[]")
	}

	// Handle type with precision/scale: numeric(10,2), varchar(255), etc.
	if idx := strings.Index(baseType, "("); idx > 0 {
		baseType = baseType[:idx]
	}

	// Handle character varying -> varchar
	baseType = strings.ReplaceAll(baseType, "character varying", "varchar")
	baseType = strings.ReplaceAll(baseType, "character", "char")
	baseType = strings.ReplaceAll(baseType, "double precision", "float8")
	baseType = strings.ReplaceAll(baseType, "timestamp without time zone", "timestamp")
	baseType = strings.ReplaceAll(baseType, "timestamp with time zone", "timestamptz")
	baseType = strings.ReplaceAll(baseType, "time without time zone", "time")
	baseType = strings.ReplaceAll(baseType, "time with time zone", "timetz")

	var tsType string
	switch baseType {
	// String types
	case "text", "varchar", "char", "bpchar", "name", "citext", "uuid":
		tsType = "string"

	// Numeric types
	case "int2", "int4", "int8", "smallint", "integer", "bigint",
		"float4", "float8", "real", "numeric", "decimal", "money",
		"smallserial", "serial", "bigserial", "oid":
		tsType = "number"

	// Boolean
	case "bool", "boolean":
		tsType = "boolean"

	// JSON types
	case "json", "jsonb":
		tsType = "Record<string, unknown>"

	// Date/Time types
	case "date", "timestamp", "timestamptz", "time", "timetz", "interval":
		tsType = "string" // ISO 8601 string representation

	// Binary types
	case "bytea":
		tsType = "string" // Base64 encoded

	// Network types
	case "inet", "cidr", "macaddr", "macaddr8":
		tsType = "string"

	// Geometric types
	case "point", "line", "lseg", "box", "path", "polygon", "circle":
		tsType = "string" // PostgreSQL geometric string representation

	// Range types
	case "int4range", "int8range", "numrange", "tsrange", "tstzrange", "daterange":
		tsType = "string" // Range string representation

	// Full-text search
	case "tsvector", "tsquery":
		tsType = "string"

	// Vector type (pgvector)
	case "vector":
		tsType = "number[]"

	// XML
	case "xml":
		tsType = "string"

	// Enum types (custom) - return string as default
	// User can override with custom type definitions
	case "void":
		tsType = "void"

	// Record/composite types
	case "record":
		tsType = "Record<string, unknown>"

	// SETOF types
	default:
		if strings.HasPrefix(baseType, "setof ") {
			innerType := strings.TrimPrefix(baseType, "setof ")
			return pgTypeToTS(innerType) + "[]"
		}
		// Unknown types default to unknown
		tsType = "unknown"
	}

	if isArray {
		return tsType + "[]"
	}
	return tsType
}

// toPascalCase converts a snake_case or kebab-case string to PascalCase
func toPascalCase(s string) string {
	// Replace underscores and hyphens with spaces for splitting
	s = strings.ReplaceAll(s, "_", " ")
	s = strings.ReplaceAll(s, "-", " ")

	words := strings.Fields(s)
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(string(word[0])) + strings.ToLower(word[1:])
		}
	}
	return strings.Join(words, "")
}

// filterBySchema filters tables to only include those in the specified schemas
func filterBySchema(tables []database.TableInfo, schemas []string) []database.TableInfo {
	schemaSet := make(map[string]bool)
	for _, s := range schemas {
		schemaSet[s] = true
	}

	result := make([]database.TableInfo, 0)
	for _, t := range tables {
		if schemaSet[t.Schema] {
			result = append(result, t)
		}
	}
	return result
}

// groupBySchema groups tables by their schema
func groupBySchema(tables []database.TableInfo) map[string][]database.TableInfo {
	result := make(map[string][]database.TableInfo)
	for _, t := range tables {
		result[t.Schema] = append(result[t.Schema], t)
	}
	return result
}

// getCurrentTimestamp returns the current time as an ISO 8601 string
func getCurrentTimestamp() string {
	return "runtime"
}

// sanitizeIdentifier ensures an identifier is safe for use in TypeScript
var tsIdentifierRegex = regexp.MustCompile(`^[a-zA-Z_$][a-zA-Z0-9_$]*$`)

func sanitizeIdentifier(name string) string {
	if tsIdentifierRegex.MatchString(name) {
		return name
	}
	// Quote the identifier if it contains special characters
	return fmt.Sprintf("'%s'", strings.ReplaceAll(name, "'", "\\'"))
}
