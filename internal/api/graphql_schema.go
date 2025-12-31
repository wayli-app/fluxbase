package api

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/fluxbase-eu/fluxbase/internal/database"
	"github.com/graphql-go/graphql"
	"github.com/rs/zerolog/log"
)

// GraphQLSchemaGenerator generates GraphQL schema from database tables
type GraphQLSchemaGenerator struct {
	schemaCache *database.SchemaCache
	db          *database.Connection

	mu              sync.RWMutex
	schema          *graphql.Schema
	objectTypes     map[string]*graphql.Object      // "schema_table" -> GraphQL object type
	inputTypes      map[string]*graphql.InputObject // "schema_table_input" -> GraphQL input type
	filterTypes     map[string]*graphql.InputObject // "schema_table_filter" -> GraphQL filter input type
	orderByTypes    map[string]*graphql.InputObject // "schema_table_order_by" -> GraphQL order by input type
	introspectionOn bool
	resolverFactory *GraphQLResolverFactory
}

// NewGraphQLSchemaGenerator creates a new schema generator
func NewGraphQLSchemaGenerator(schemaCache *database.SchemaCache, db *database.Connection, introspectionOn bool) *GraphQLSchemaGenerator {
	return &GraphQLSchemaGenerator{
		schemaCache:     schemaCache,
		db:              db,
		objectTypes:     make(map[string]*graphql.Object),
		inputTypes:      make(map[string]*graphql.InputObject),
		filterTypes:     make(map[string]*graphql.InputObject),
		orderByTypes:    make(map[string]*graphql.InputObject),
		introspectionOn: introspectionOn,
	}
}

// SetResolverFactory sets the resolver factory for query execution
func (g *GraphQLSchemaGenerator) SetResolverFactory(factory *GraphQLResolverFactory) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.resolverFactory = factory
}

// GetSchema returns the current GraphQL schema, regenerating if needed
func (g *GraphQLSchemaGenerator) GetSchema(ctx context.Context) (*graphql.Schema, error) {
	g.mu.RLock()
	if g.schema != nil {
		g.mu.RUnlock()
		return g.schema, nil
	}
	g.mu.RUnlock()

	return g.regenerateSchema(ctx)
}

// InvalidateSchema forces schema regeneration on next access
func (g *GraphQLSchemaGenerator) InvalidateSchema() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.schema = nil
}

// regenerateSchema rebuilds the GraphQL schema from database tables
func (g *GraphQLSchemaGenerator) regenerateSchema(ctx context.Context) (*graphql.Schema, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	// Clear previous types
	g.objectTypes = make(map[string]*graphql.Object)
	g.inputTypes = make(map[string]*graphql.InputObject)
	g.filterTypes = make(map[string]*graphql.InputObject)
	g.orderByTypes = make(map[string]*graphql.InputObject)

	// Get all tables from schema cache
	tables, err := g.schemaCache.GetAllTables(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get tables: %w", err)
	}

	// Also get views
	views, err := g.schemaCache.GetAllViews(ctx)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to get views for GraphQL schema")
	} else {
		tables = append(tables, views...)
	}

	// Get materialized views
	matViews, err := g.schemaCache.GetAllMaterializedViews(ctx)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to get materialized views for GraphQL schema")
	} else {
		tables = append(tables, matViews...)
	}

	// Filter to only public schema tables (for now)
	// In the future, we can support multiple schemas
	var publicTables []database.TableInfo
	for _, t := range tables {
		if t.Schema == "public" {
			publicTables = append(publicTables, t)
		}
	}

	// Generate object types for each table (first pass - create stubs)
	for _, table := range publicTables {
		typeName := g.tableToTypeName(table.Schema, table.Name)
		g.objectTypes[typeName] = graphql.NewObject(graphql.ObjectConfig{
			Name:        typeName,
			Description: fmt.Sprintf("Auto-generated type for table %s.%s", table.Schema, table.Name),
			Fields:      graphql.Fields{}, // Will be populated later
		})
	}

	// Second pass - populate fields (allows for circular references via foreign keys)
	for _, table := range publicTables {
		typeName := g.tableToTypeName(table.Schema, table.Name)
		objType := g.objectTypes[typeName]

		// Generate fields for this table
		fields := g.generateTableFields(table)
		for name, field := range fields {
			objType.AddFieldConfig(name, field)
		}

		// Generate input type
		g.inputTypes[typeName+"Input"] = g.generateInputType(table)

		// Generate filter type
		g.filterTypes[typeName+"Filter"] = g.generateFilterType(table)

		// Generate order by type
		g.orderByTypes[typeName+"OrderBy"] = g.generateOrderByType(table)
	}

	// Build query fields
	queryFields := graphql.Fields{}

	// Always add a _health field for introspection even when no tables are exposed
	queryFields["_health"] = &graphql.Field{
		Type:        graphql.String,
		Description: "Health check endpoint - returns 'ok' if the GraphQL API is functioning",
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			return "ok", nil
		},
	}

	for _, table := range publicTables {
		typeName := g.tableToTypeName(table.Schema, table.Name)
		objType := g.objectTypes[typeName]
		filterType := g.filterTypes[typeName+"Filter"]
		orderByType := g.orderByTypes[typeName+"OrderBy"]

		// Collection query (e.g., users, posts)
		collectionName := g.tableToCollectionName(table.Name)
		queryFields[collectionName] = &graphql.Field{
			Type:        graphql.NewList(objType),
			Description: fmt.Sprintf("Query %s records", table.Name),
			Args: graphql.FieldConfigArgument{
				"filter": &graphql.ArgumentConfig{
					Type:        filterType,
					Description: "Filter conditions",
				},
				"orderBy": &graphql.ArgumentConfig{
					Type:        graphql.NewList(orderByType),
					Description: "Sort order",
				},
				"limit": &graphql.ArgumentConfig{
					Type:        graphql.Int,
					Description: "Maximum number of records to return",
				},
				"offset": &graphql.ArgumentConfig{
					Type:        graphql.Int,
					Description: "Number of records to skip",
				},
			},
			Resolve: g.makeCollectionResolver(table),
		}

		// Single record query by primary key (e.g., user, post)
		if len(table.PrimaryKey) > 0 {
			singleName := g.tableToSingleName(table.Name)
			pkArgs := g.generatePrimaryKeyArgs(table)
			if len(pkArgs) > 0 {
				queryFields[singleName] = &graphql.Field{
					Type:        objType,
					Description: fmt.Sprintf("Get a single %s by primary key", table.Name),
					Args:        pkArgs,
					Resolve:     g.makeSingleResolver(table),
				}
			}
		}
	}

	// Build mutation fields
	mutationFields := graphql.Fields{}
	for _, table := range publicTables {
		if table.Type != "table" {
			continue // Only allow mutations on tables, not views
		}

		typeName := g.tableToTypeName(table.Schema, table.Name)
		objType := g.objectTypes[typeName]
		inputType := g.inputTypes[typeName+"Input"]
		filterType := g.filterTypes[typeName+"Filter"]

		// Insert mutation
		insertName := "insert" + typeName
		mutationFields[insertName] = &graphql.Field{
			Type:        objType,
			Description: fmt.Sprintf("Insert a new %s record", table.Name),
			Args: graphql.FieldConfigArgument{
				"data": &graphql.ArgumentConfig{
					Type:        graphql.NewNonNull(inputType),
					Description: "Data to insert",
				},
			},
			Resolve: g.makeInsertResolver(table),
		}

		// Insert many mutation
		insertManyName := "insertMany" + typeName
		mutationFields[insertManyName] = &graphql.Field{
			Type:        graphql.NewList(objType),
			Description: fmt.Sprintf("Insert multiple %s records", table.Name),
			Args: graphql.FieldConfigArgument{
				"data": &graphql.ArgumentConfig{
					Type:        graphql.NewNonNull(graphql.NewList(graphql.NewNonNull(inputType))),
					Description: "Array of data to insert",
				},
			},
			Resolve: g.makeInsertManyResolver(table),
		}

		// Update mutation
		if len(table.PrimaryKey) > 0 {
			updateName := "update" + typeName
			pkArgs := g.generatePrimaryKeyArgs(table)
			pkArgs["data"] = &graphql.ArgumentConfig{
				Type:        graphql.NewNonNull(inputType),
				Description: "Data to update",
			}
			mutationFields[updateName] = &graphql.Field{
				Type:        objType,
				Description: fmt.Sprintf("Update a %s record by primary key", table.Name),
				Args:        pkArgs,
				Resolve:     g.makeUpdateResolver(table),
			}

			// Update many mutation
			updateManyName := "updateMany" + typeName
			mutationFields[updateManyName] = &graphql.Field{
				Type:        graphql.NewList(objType),
				Description: fmt.Sprintf("Update multiple %s records matching filter", table.Name),
				Args: graphql.FieldConfigArgument{
					"filter": &graphql.ArgumentConfig{
						Type:        graphql.NewNonNull(filterType),
						Description: "Filter conditions",
					},
					"data": &graphql.ArgumentConfig{
						Type:        graphql.NewNonNull(inputType),
						Description: "Data to update",
					},
				},
				Resolve: g.makeUpdateManyResolver(table),
			}

			// Delete mutation
			deleteName := "delete" + typeName
			mutationFields[deleteName] = &graphql.Field{
				Type:        objType,
				Description: fmt.Sprintf("Delete a %s record by primary key", table.Name),
				Args:        g.generatePrimaryKeyArgs(table),
				Resolve:     g.makeDeleteResolver(table),
			}

			// Delete many mutation
			deleteManyName := "deleteMany" + typeName
			mutationFields[deleteManyName] = &graphql.Field{
				Type:        graphql.Int,
				Description: fmt.Sprintf("Delete multiple %s records matching filter", table.Name),
				Args: graphql.FieldConfigArgument{
					"filter": &graphql.ArgumentConfig{
						Type:        graphql.NewNonNull(filterType),
						Description: "Filter conditions",
					},
				},
				Resolve: g.makeDeleteManyResolver(table),
			}
		}
	}

	// Create schema config
	schemaConfig := graphql.SchemaConfig{
		Query: graphql.NewObject(graphql.ObjectConfig{
			Name:   "Query",
			Fields: queryFields,
		}),
	}

	// Only add mutations if there are any
	if len(mutationFields) > 0 {
		schemaConfig.Mutation = graphql.NewObject(graphql.ObjectConfig{
			Name:   "Mutation",
			Fields: mutationFields,
		})
	}

	// Create schema
	schema, err := graphql.NewSchema(schemaConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create GraphQL schema: %w", err)
	}

	g.schema = &schema
	return &schema, nil
}

// generateTableFields generates GraphQL fields for a table's columns
func (g *GraphQLSchemaGenerator) generateTableFields(table database.TableInfo) graphql.Fields {
	fields := graphql.Fields{}

	for _, col := range table.Columns {
		fieldName := g.columnToFieldName(col.Name)
		fields[fieldName] = &graphql.Field{
			Type:        PostgresTypeToGraphQL(col.DataType, col.IsNullable),
			Description: fmt.Sprintf("Column %s (%s)", col.Name, col.DataType),
			Resolve: func(colName string) graphql.FieldResolveFn {
				return func(p graphql.ResolveParams) (interface{}, error) {
					if source, ok := p.Source.(map[string]interface{}); ok {
						return source[colName], nil
					}
					return nil, nil
				}
			}(col.Name),
		}
	}

	// Add foreign key relationship fields
	for _, fk := range table.ForeignKeys {
		// Find the referenced table's type
		refTypeName := g.tableToTypeName(table.Schema, fk.ReferencedTable)
		if refType, ok := g.objectTypes[refTypeName]; ok {
			// Create a field for the relationship (singular, e.g., "author" for author_id)
			relFieldName := g.fkToRelationName(fk.ColumnName)
			fields[relFieldName] = &graphql.Field{
				Type:        refType,
				Description: fmt.Sprintf("Related %s via %s", fk.ReferencedTable, fk.ColumnName),
				Resolve:     g.makeForeignKeyResolver(table, fk),
			}
		}
	}

	return fields
}

// generateInputType generates a GraphQL input type for insert/update operations
func (g *GraphQLSchemaGenerator) generateInputType(table database.TableInfo) *graphql.InputObject {
	typeName := g.tableToTypeName(table.Schema, table.Name)
	fields := graphql.InputObjectConfigFieldMap{}

	for _, col := range table.Columns {
		// Skip auto-generated columns for input
		if isAutoGenerated(col) {
			continue
		}

		fieldName := g.columnToFieldName(col.Name)
		fields[fieldName] = &graphql.InputObjectFieldConfig{
			Type:        postgresTypeToGraphQLInput(col.DataType),
			Description: fmt.Sprintf("Input for column %s", col.Name),
		}
	}

	return graphql.NewInputObject(graphql.InputObjectConfig{
		Name:        typeName + "Input",
		Description: fmt.Sprintf("Input type for %s.%s", table.Schema, table.Name),
		Fields:      fields,
	})
}

// generateFilterType generates a GraphQL input type for filtering
func (g *GraphQLSchemaGenerator) generateFilterType(table database.TableInfo) *graphql.InputObject {
	typeName := g.tableToTypeName(table.Schema, table.Name)
	fields := graphql.InputObjectConfigFieldMap{}

	for _, col := range table.Columns {
		fieldName := g.columnToFieldName(col.Name)
		ops := GetFilterOperatorsForType(col.DataType)
		baseType := postgresTypeToGraphQLInput(col.DataType)

		// eq - equals
		if ops.Eq {
			fields[fieldName+"_eq"] = &graphql.InputObjectFieldConfig{
				Type: baseType,
			}
		}

		// neq - not equals
		if ops.Neq {
			fields[fieldName+"_neq"] = &graphql.InputObjectFieldConfig{
				Type: baseType,
			}
		}

		// gt - greater than
		if ops.Gt {
			fields[fieldName+"_gt"] = &graphql.InputObjectFieldConfig{
				Type: baseType,
			}
		}

		// gte - greater than or equal
		if ops.Gte {
			fields[fieldName+"_gte"] = &graphql.InputObjectFieldConfig{
				Type: baseType,
			}
		}

		// lt - less than
		if ops.Lt {
			fields[fieldName+"_lt"] = &graphql.InputObjectFieldConfig{
				Type: baseType,
			}
		}

		// lte - less than or equal
		if ops.Lte {
			fields[fieldName+"_lte"] = &graphql.InputObjectFieldConfig{
				Type: baseType,
			}
		}

		// like - pattern match
		if ops.Like {
			fields[fieldName+"_like"] = &graphql.InputObjectFieldConfig{
				Type: graphql.String,
			}
		}

		// ilike - case-insensitive pattern match
		if ops.ILike {
			fields[fieldName+"_ilike"] = &graphql.InputObjectFieldConfig{
				Type: graphql.String,
			}
		}

		// in - in array
		if ops.In {
			fields[fieldName+"_in"] = &graphql.InputObjectFieldConfig{
				Type: graphql.NewList(baseType),
			}
		}

		// is_null - null check
		if ops.IsNull {
			fields[fieldName+"_is_null"] = &graphql.InputObjectFieldConfig{
				Type: graphql.Boolean,
			}
		}

		// contains - JSON contains (@>)
		if ops.Contains {
			fields[fieldName+"_contains"] = &graphql.InputObjectFieldConfig{
				Type: JSONScalar,
			}
		}

		// contained_by - JSON contained by (<@)
		if ops.ContainedBy {
			fields[fieldName+"_contained_by"] = &graphql.InputObjectFieldConfig{
				Type: JSONScalar,
			}
		}
	}

	// Add logical operators
	// Note: We use lazy initialization to avoid circular reference issues
	filterTypeName := typeName + "Filter"
	fields["_and"] = &graphql.InputObjectFieldConfig{
		Type:        graphql.NewList(graphql.NewInputObject(graphql.InputObjectConfig{Name: filterTypeName + "And", Fields: fields})),
		Description: "Logical AND",
	}
	fields["_or"] = &graphql.InputObjectFieldConfig{
		Type:        graphql.NewList(graphql.NewInputObject(graphql.InputObjectConfig{Name: filterTypeName + "Or", Fields: fields})),
		Description: "Logical OR",
	}
	fields["_not"] = &graphql.InputObjectFieldConfig{
		Type:        graphql.NewInputObject(graphql.InputObjectConfig{Name: filterTypeName + "Not", Fields: fields}),
		Description: "Logical NOT",
	}

	return graphql.NewInputObject(graphql.InputObjectConfig{
		Name:        filterTypeName,
		Description: fmt.Sprintf("Filter type for %s.%s", table.Schema, table.Name),
		Fields:      fields,
	})
}

// generateOrderByType generates a GraphQL input type for ordering
func (g *GraphQLSchemaGenerator) generateOrderByType(table database.TableInfo) *graphql.InputObject {
	typeName := g.tableToTypeName(table.Schema, table.Name)
	fields := graphql.InputObjectConfigFieldMap{}

	// Order direction enum
	orderDirEnum := graphql.NewEnum(graphql.EnumConfig{
		Name: typeName + "OrderDirection",
		Values: graphql.EnumValueConfigMap{
			"ASC": &graphql.EnumValueConfig{
				Value:       "ASC",
				Description: "Ascending order",
			},
			"DESC": &graphql.EnumValueConfig{
				Value:       "DESC",
				Description: "Descending order",
			},
			"ASC_NULLS_FIRST": &graphql.EnumValueConfig{
				Value:       "ASC_NULLS_FIRST",
				Description: "Ascending with nulls first",
			},
			"ASC_NULLS_LAST": &graphql.EnumValueConfig{
				Value:       "ASC_NULLS_LAST",
				Description: "Ascending with nulls last",
			},
			"DESC_NULLS_FIRST": &graphql.EnumValueConfig{
				Value:       "DESC_NULLS_FIRST",
				Description: "Descending with nulls first",
			},
			"DESC_NULLS_LAST": &graphql.EnumValueConfig{
				Value:       "DESC_NULLS_LAST",
				Description: "Descending with nulls last",
			},
		},
	})

	for _, col := range table.Columns {
		fieldName := g.columnToFieldName(col.Name)
		fields[fieldName] = &graphql.InputObjectFieldConfig{
			Type: orderDirEnum,
		}
	}

	return graphql.NewInputObject(graphql.InputObjectConfig{
		Name:        typeName + "OrderBy",
		Description: fmt.Sprintf("Ordering for %s.%s", table.Schema, table.Name),
		Fields:      fields,
	})
}

// generatePrimaryKeyArgs generates GraphQL arguments for primary key lookup
func (g *GraphQLSchemaGenerator) generatePrimaryKeyArgs(table database.TableInfo) graphql.FieldConfigArgument {
	args := graphql.FieldConfigArgument{}

	for _, pkCol := range table.PrimaryKey {
		// Find the column info
		var col *database.ColumnInfo
		for i := range table.Columns {
			if table.Columns[i].Name == pkCol {
				col = &table.Columns[i]
				break
			}
		}
		if col == nil {
			continue
		}

		fieldName := g.columnToFieldName(pkCol)
		args[fieldName] = &graphql.ArgumentConfig{
			Type:        graphql.NewNonNull(postgresTypeToGraphQLInput(col.DataType)),
			Description: fmt.Sprintf("Primary key column %s", pkCol),
		}
	}

	return args
}

// Helper functions for naming conventions

// tableToTypeName converts a table name to a GraphQL type name (PascalCase)
func (g *GraphQLSchemaGenerator) tableToTypeName(schema, table string) string {
	return toPascalCase(table)
}

// tableToCollectionName converts a table name to a collection query name (camelCase, plural)
func (g *GraphQLSchemaGenerator) tableToCollectionName(table string) string {
	return toCamelCase(table)
}

// tableToSingleName converts a table name to a single record query name (camelCase, singular)
func (g *GraphQLSchemaGenerator) tableToSingleName(table string) string {
	name := toCamelCase(table)
	// Simple singularization - remove trailing 's' if present
	if len(name) > 1 && name[len(name)-1] == 's' {
		// Handle special cases
		if strings.HasSuffix(name, "ies") {
			return name[:len(name)-3] + "y"
		}
		if strings.HasSuffix(name, "es") && !strings.HasSuffix(name, "ves") {
			return name[:len(name)-2]
		}
		return name[:len(name)-1]
	}
	return name
}

// columnToFieldName converts a column name to a GraphQL field name (camelCase)
func (g *GraphQLSchemaGenerator) columnToFieldName(column string) string {
	return toCamelCase(column)
}

// fkToRelationName converts a foreign key column name to a relation field name
func (g *GraphQLSchemaGenerator) fkToRelationName(fkColumn string) string {
	// Remove common suffixes like _id, Id
	name := fkColumn
	if strings.HasSuffix(name, "_id") {
		name = name[:len(name)-3]
	} else if strings.HasSuffix(name, "Id") {
		name = name[:len(name)-2]
	}
	return toCamelCase(name)
}

// toPascalCase converts a snake_case string to PascalCase
func toPascalCase(s string) string {
	words := splitWords(s)
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + strings.ToLower(word[1:])
		}
	}
	return strings.Join(words, "")
}

// toCamelCase converts a snake_case string to camelCase
func toCamelCase(s string) string {
	words := splitWords(s)
	for i, word := range words {
		if i == 0 {
			words[i] = strings.ToLower(word)
		} else if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + strings.ToLower(word[1:])
		}
	}
	return strings.Join(words, "")
}

// splitWords splits a string by underscores, hyphens, and camelCase boundaries
func splitWords(s string) []string {
	// First split by underscores and hyphens
	re := regexp.MustCompile(`[_\-]+`)
	parts := re.Split(s, -1)

	// Then handle any remaining camelCase
	var words []string
	for _, part := range parts {
		if part == "" {
			continue
		}
		// Split camelCase
		camelRe := regexp.MustCompile(`([a-z])([A-Z])`)
		part = camelRe.ReplaceAllString(part, "${1} ${2}")
		for _, word := range strings.Fields(part) {
			if word != "" {
				words = append(words, word)
			}
		}
	}
	return words
}

// isAutoGenerated checks if a column is auto-generated (serial, uuid default, etc.)
func isAutoGenerated(col database.ColumnInfo) bool {
	if col.DefaultValue == nil {
		return false
	}
	def := strings.ToLower(*col.DefaultValue)

	// Serial/identity columns
	if strings.Contains(def, "nextval(") {
		return true
	}

	// UUID generation
	if strings.Contains(def, "gen_random_uuid()") || strings.Contains(def, "uuid_generate") {
		return true
	}

	// Timestamp defaults
	if strings.Contains(def, "now()") || strings.Contains(def, "current_timestamp") {
		return true
	}

	return false
}

// postgresTypeToGraphQLInput maps PostgreSQL types to GraphQL input types
func postgresTypeToGraphQLInput(pgType string) graphql.Input {
	switch pgType {
	case "text", "varchar", "character varying", "char", "character", "name", "citext":
		return graphql.String
	case "integer", "int", "int4", "smallint", "int2", "serial", "serial4":
		return graphql.Int
	case "bigint", "int8", "bigserial", "serial8":
		return BigIntScalar
	case "real", "float4", "double precision", "float8", "numeric", "decimal", "money":
		return graphql.Float
	case "boolean", "bool":
		return graphql.Boolean
	case "uuid":
		return UUIDScalar
	case "json", "jsonb":
		return JSONScalar
	case "timestamp", "timestamp without time zone", "timestamp with time zone", "timestamptz",
		"date", "time", "time without time zone", "time with time zone", "timetz", "interval":
		return DateTimeScalar
	default:
		// Default to string for unknown types
		return graphql.String
	}
}
